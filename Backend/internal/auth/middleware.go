package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/dennisdiepolder/monti/backend/internal/types"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Email            string           `json:"email"`
	Name             string           `json:"name"`
	Role             string           `json:"role"`
	Groups           []string         `json:"groups"`
	BusinessUnits    []string         `json:"businessUnits"`    // Extracted from groups (e.g., SGB, NGB, RGB)
	AllowedLocations []types.Location `json:"allowedLocations"` // Computed from BUs or admin override
	jwt.RegisteredClaims
}

type contextKey string

const UserContextKey contextKey = "user"

// JWKSManager handles JWKS fetching and caching
type JWKSManager struct {
	jwks       keyfunc.Keyfunc
	issuerURL  string
	mu         sync.RWMutex
	lastUpdate time.Time
}

var (
	jwksManager *JWKSManager
	jwksOnce    sync.Once
)

// InitJWKS initializes the JWKS manager for token verification
// Call this on server startup in production mode
func InitJWKS(issuerURL string) error {
	var initErr error
	jwksOnce.Do(func() {
		jwksManager = &JWKSManager{issuerURL: issuerURL}
		initErr = jwksManager.refresh()
	})
	return initErr
}

// refresh fetches the JWKS from the OIDC provider
func (m *JWKSManager) refresh() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Construct JWKS URL (Keycloak format)
	jwksURL := strings.TrimSuffix(m.issuerURL, "/") + "/protocol/openid-connect/certs"
	log.Printf("[Auth] Fetching JWKS from: %s", jwksURL)

	// Create keyfunc with options
	k, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		return fmt.Errorf("failed to create keyfunc: %w", err)
	}

	m.jwks = k
	m.lastUpdate = time.Now()
	log.Printf("[Auth] JWKS loaded successfully")
	return nil
}

// getKeyfunc returns the JWT keyfunc for token verification
func (m *JWKSManager) getKeyfunc() jwt.Keyfunc {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.jwks == nil {
		return nil
	}
	return m.jwks.Keyfunc
}

// Middleware validates JWT tokens from OIDC provider
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health check
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		// In development mode, you can bypass auth
		skipAuth := os.Getenv("SKIP_AUTH")
		if skipAuth == "true" {
			log.Println("[Auth] SKIP_AUTH enabled - bypassing authentication")
			// Create a default dev user with admin role (sees all locations)
			ctx := context.WithValue(r.Context(), UserContextKey, &Claims{
				Email:            "dev@monti.local",
				Name:             "Dev User",
				Role:             "admin",
				Groups:           []string{"developers", "monti-admins"},
				BusinessUnits:    []string{}, // Admin doesn't need specific BUs
				AllowedLocations: types.AllLocations,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Extract token from Authorization header or query parameter
		tokenString := extractToken(r)
		if tokenString == "" {
			log.Println("[Auth] Missing authorization token")
			http.Error(w, "Unauthorized: Missing token", http.StatusUnauthorized)
			return
		}

		// Validate token
		claims, err := validateToken(tokenString)
		if err != nil {
			log.Printf("[Auth] Token validation failed: %v", err)
			http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
			return
		}

		log.Printf("[Auth] User authenticated: %s (%s)", claims.Email, claims.Role)

		// Add user to context
		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractToken gets the token from Authorization header or query parameter
func extractToken(r *http.Request) string {
	// Try Authorization header first
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString != authHeader {
			return tokenString
		}
	}

	// Try query parameter (for WebSocket connections)
	token := r.URL.Query().Get("token")
	if token != "" {
		return token
	}

	return ""
}

// validateToken validates the JWT token with optional signature verification
func validateToken(tokenString string) (*Claims, error) {
	env := os.Getenv("ENV")
	verifySignature := os.Getenv("VERIFY_JWT_SIGNATURE") == "true"

	// In production, verify signature by default
	if env != "development" && env != "" {
		verifySignature = true
	}

	var token *jwt.Token
	var err error

	if verifySignature {
		// Production: Verify signature using JWKS
		token, err = parseAndVerifyToken(tokenString)
		if err != nil {
			return nil, err
		}
	} else {
		// Development: Parse without verification (for local testing)
		log.Println("[Auth] WARNING: JWT signature verification disabled (development mode)")
		token, _, err = new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
		if err != nil {
			return nil, fmt.Errorf("failed to parse token: %w", err)
		}
	}

	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Create Claims struct
	claims := &Claims{}

	// Extract email
	if email, ok := mapClaims["email"].(string); ok {
		claims.Email = email
	}

	// Extract name
	if name, ok := mapClaims["name"].(string); ok {
		claims.Name = name
	} else if preferredUsername, ok := mapClaims["preferred_username"].(string); ok {
		claims.Name = preferredUsername
	}

	// Extract role from various possible locations
	claims.Role = extractRoleFromMapClaims(mapClaims)

	// Extract groups
	claims.Groups = extractGroupsFromMapClaims(mapClaims)

	// Extract business units from groups and compute allowed locations
	claims.BusinessUnits = extractBusinessUnits(claims.Groups)
	claims.AllowedLocations = computeAllowedLocations(claims.Role, claims.BusinessUnits)

	// Extract standard claims
	if sub, ok := mapClaims["sub"].(string); ok {
		claims.Subject = sub
	}

	// Check expiration (for unverified tokens - verified tokens check this automatically)
	if !verifySignature {
		if exp, ok := mapClaims["exp"].(float64); ok {
			expTime := time.Unix(int64(exp), 0)
			claims.ExpiresAt = jwt.NewNumericDate(expTime)
			if expTime.Before(time.Now()) {
				return nil, fmt.Errorf("token expired")
			}
		}
	}

	log.Printf("[Auth] Token parsed: email=%s, role=%s, groups=%v, businessUnits=%v, allowedLocations=%v",
		claims.Email, claims.Role, claims.Groups, claims.BusinessUnits, claims.AllowedLocations)

	return claims, nil
}

// parseAndVerifyToken verifies the JWT signature using JWKS
func parseAndVerifyToken(tokenString string) (*jwt.Token, error) {
	// Ensure JWKS is initialized
	if jwksManager == nil {
		issuer := os.Getenv("OIDC_ISSUER")
		if issuer == "" {
			return nil, fmt.Errorf("OIDC_ISSUER not configured for production JWT verification")
		}
		if err := InitJWKS(issuer); err != nil {
			return nil, fmt.Errorf("failed to initialize JWKS: %w", err)
		}
	}

	keyfunc := jwksManager.getKeyfunc()
	if keyfunc == nil {
		return nil, fmt.Errorf("JWKS not available")
	}

	// Parse and verify the token
	token, err := jwt.Parse(tokenString, keyfunc, jwt.WithValidMethods([]string{"RS256", "RS384", "RS512", "ES256", "ES384", "ES512"}))
	if err != nil {
		return nil, fmt.Errorf("token verification failed: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return token, nil
}

// extractRoleFromMapClaims extracts role from various possible token claim locations
func extractRoleFromMapClaims(mapClaims jwt.MapClaims) string {
	// Check realm_access.roles (Keycloak)
	if realmAccess, ok := mapClaims["realm_access"].(map[string]interface{}); ok {
		if roles, ok := realmAccess["roles"].([]interface{}); ok {
			// Priority order: admin > supervisor > agent > viewer
			for _, priority := range []string{"admin", "supervisor", "agent", "viewer"} {
				for _, role := range roles {
					if roleStr, ok := role.(string); ok && roleStr == priority {
						return roleStr
					}
				}
			}
		}
	}

	// Check cognito:groups (AWS Cognito)
	if cognitoGroups, ok := mapClaims["cognito:groups"].([]interface{}); ok {
		for _, group := range cognitoGroups {
			if groupStr, ok := group.(string); ok {
				if strings.Contains(groupStr, "admin") {
					return "admin"
				}
				if strings.Contains(groupStr, "supervisor") {
					return "supervisor"
				}
				if strings.Contains(groupStr, "agent") {
					return "agent"
				}
			}
		}
	}

	// Check custom:groups
	if customGroups, ok := mapClaims["custom:groups"].([]interface{}); ok {
		for _, group := range customGroups {
			if groupStr, ok := group.(string); ok {
				if strings.Contains(groupStr, "admin") {
					return "admin"
				}
				if strings.Contains(groupStr, "supervisor") {
					return "supervisor"
				}
				if strings.Contains(groupStr, "agent") {
					return "agent"
				}
			}
		}
	}

	return "viewer" // default role
}

// extractGroupsFromMapClaims extracts groups from token claims
func extractGroupsFromMapClaims(mapClaims jwt.MapClaims) []string {
	var groups []string

	// Check groups claim
	if groupsClaim, ok := mapClaims["groups"].([]interface{}); ok {
		for _, group := range groupsClaim {
			if groupStr, ok := group.(string); ok {
				groups = append(groups, groupStr)
			}
		}
	}

	// Check cognito:groups
	if cognitoGroups, ok := mapClaims["cognito:groups"].([]interface{}); ok {
		for _, group := range cognitoGroups {
			if groupStr, ok := group.(string); ok {
				groups = append(groups, groupStr)
			}
		}
	}

	return groups
}

// GetUserFromContext retrieves user claims from request context
func GetUserFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(UserContextKey).(*Claims)
	return claims, ok
}

// HasRole checks if user has specific role
func HasRole(claims *Claims, role string) bool {
	return claims.Role == role
}

// InGroup checks if user is in specific group
func InGroup(claims *Claims, group string) bool {
	for _, g := range claims.Groups {
		if g == group {
			return true
		}
	}
	return false
}

// extractBusinessUnits parses business unit names from group paths
// Groups are expected in format: /business-units/SGB, /business-units/NGB, etc.
func extractBusinessUnits(groups []string) []string {
	var businessUnits []string
	buPrefix := "/business-units/"

	for _, group := range groups {
		if strings.HasPrefix(group, buPrefix) {
			bu := strings.TrimPrefix(group, buPrefix)
			// Remove any trailing path components
			if idx := strings.Index(bu, "/"); idx > 0 {
				bu = bu[:idx]
			}
			if bu != "" {
				businessUnits = append(businessUnits, bu)
			}
		}
	}

	return businessUnits
}

// computeAllowedLocations maps business units to their allowed locations
// Admin role overrides and gets access to all locations
// If no BUs are assigned, returns empty slice (fail secure)
func computeAllowedLocations(role string, businessUnits []string) []types.Location {
	// Admin role sees everything
	if role == "admin" {
		return types.AllLocations
	}

	// Build unique set of allowed locations from all assigned BUs
	locationSet := make(map[types.Location]bool)
	for _, buName := range businessUnits {
		bu := types.BusinessUnit(buName)
		if locations, ok := types.BULocationMapping[bu]; ok {
			for _, loc := range locations {
				locationSet[loc] = true
			}
		}
	}

	// Convert set to slice
	var allowedLocations []types.Location
	for loc := range locationSet {
		allowedLocations = append(allowedLocations, loc)
	}

	return allowedLocations
}

// IsLocationAllowed checks if a location is in the allowed locations list
func (c *Claims) IsLocationAllowed(location types.Location) bool {
	for _, loc := range c.AllowedLocations {
		if loc == location {
			return true
		}
	}
	return false
}
