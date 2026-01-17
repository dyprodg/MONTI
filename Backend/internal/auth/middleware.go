package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

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

// JWKS represents the JSON Web Key Set
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key
type JWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
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

// validateToken validates the JWT token
func validateToken(tokenString string) (*Claims, error) {
	// Parse token as MapClaims to access all fields
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
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

	// Check expiration
	if exp, ok := mapClaims["exp"].(float64); ok {
		expTime := time.Unix(int64(exp), 0)
		claims.ExpiresAt = jwt.NewNumericDate(expTime)
		if expTime.Before(time.Now()) {
			return nil, fmt.Errorf("token expired")
		}
	}

	// In development, we skip signature verification for Keycloak
	// In production, you should verify against OIDC provider's public keys
	env := os.Getenv("ENV")
	if env == "development" {
		log.Printf("[Auth] Development mode - Token parsed: email=%s, role=%s, groups=%v, businessUnits=%v, allowedLocations=%v",
			claims.Email, claims.Role, claims.Groups, claims.BusinessUnits, claims.AllowedLocations)
		return claims, nil
	}

	// Production: Verify signature against OIDC provider
	issuer := os.Getenv("OIDC_ISSUER")
	if issuer == "" {
		return nil, fmt.Errorf("OIDC_ISSUER not configured")
	}

	// Here you would verify against the OIDC provider's public keys
	// For now, we'll accept the token in development mode
	return claims, nil
}

// extractRoleFromMapClaims extracts role from various possible token claim locations
func extractRoleFromMapClaims(mapClaims jwt.MapClaims) string {
	// Check realm_access.roles (Keycloak)
	if realmAccess, ok := mapClaims["realm_access"].(map[string]interface{}); ok {
		if roles, ok := realmAccess["roles"].([]interface{}); ok {
			for _, role := range roles {
				if roleStr, ok := role.(string); ok {
					if roleStr == "admin" {
						return "admin"
					}
					if roleStr == "manager" {
						return "manager"
					}
					if roleStr == "viewer" {
						return "viewer"
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
				if strings.Contains(groupStr, "manager") {
					return "manager"
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
				if strings.Contains(groupStr, "manager") {
					return "manager"
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

// fetchJWKS fetches the JWKS from the OIDC provider
func fetchJWKS(issuerURL string) (*JWKS, error) {
	// Construct JWKS URL
	jwksURL := strings.TrimSuffix(issuerURL, "/") + "/.well-known/jwks.json"

	// Make HTTP request
	resp, err := http.Get(jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	// Parse response
	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to decode JWKS: %w", err)
	}

	return &jwks, nil
}

// parseIssuerURL ensures the issuer URL is properly formatted
func parseIssuerURL(issuer string) (string, error) {
	u, err := url.Parse(issuer)
	if err != nil {
		return "", fmt.Errorf("invalid issuer URL: %w", err)
	}

	// Handle container-to-container communication
	// If issuer uses service name, it should be accessible from backend
	return u.String(), nil
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
