# Auth & Access Setup Guide for MONTI

This guide explains how to implement authentication and authorization for MONTI, both for **local development** and **production deployment**.

---

## Table of Contents
1. [Local Development Options](#local-development-options)
2. [Recommended Approach for Local Dev](#recommended-approach-for-local-dev)
3. [Production Setup with AWS IAM Identity Center](#production-setup-with-aws-iam-identity-center)
4. [Backend Implementation](#backend-implementation)
5. [Frontend Implementation](#frontend-implementation)
6. [Testing Auth Locally](#testing-auth-locally)

---

## Local Development Options

For local development, you have several options:

### Option 1: Mock Auth (Easiest for Development)
- Hardcode a fake JWT token or user session
- Use environment variables to enable/disable auth
- Skip authentication entirely in development mode
- **Pros**: Fast to implement, no external dependencies
- **Cons**: Doesn't test real auth flow

### Option 2: Self-Signed JWT (Recommended for Local)
- Generate your own JWT tokens locally
- Backend validates tokens using a local secret key
- Simulates production auth without external services
- **Pros**: Tests full auth flow, no external dependencies
- **Cons**: Need to manually generate tokens

### Option 3: Local OIDC Provider (Most Realistic)
- Run a local identity provider (like Keycloak or Auth0 dev)
- Full OAuth2/OIDC flow locally
- **Pros**: Identical to production flow
- **Cons**: Complex setup, requires Docker

---

## Recommended Approach for Local Dev

I recommend **Option 2: Self-Signed JWT** for local development. Here's why:
- Tests your actual auth middleware
- No external dependencies
- Easy to switch between different users/roles
- Simple transition to production

### How It Works

```
┌─────────────┐         ┌─────────────┐         ┌──────────────┐
│   Frontend  │         │   Backend   │         │   (Future)   │
│             │         │             │         │ IAM Identity │
│  - Dev Mode │────────▶│  - Validates│         │    Center    │
│    (Local)  │  JWT    │    JWT      │         │              │
│  - Prod Mode│────────▶│  - Checks   │────────▶│  (Production)│
│             │  Token  │    Claims   │  OIDC   │              │
└─────────────┘         └─────────────┘         └──────────────┘
```

---

## Step-by-Step: Local Development Setup

### Step 1: Create a JWT Generator Script

Create a simple script to generate development tokens:

**File: `scripts/generate-dev-token.sh`**

```bash
#!/bin/bash

# This generates a mock JWT for local development
# In production, IAM Identity Center will issue real tokens

JWT_SECRET="your-local-dev-secret-change-this-in-production"
USER_EMAIL=${1:-"dev@monti.local"}
USER_ROLE=${2:-"admin"}

# JWT Header
header='{"alg":"HS256","typ":"JWT"}'

# JWT Payload
payload=$(cat <<EOF
{
  "sub": "${USER_EMAIL}",
  "email": "${USER_EMAIL}",
  "role": "${USER_ROLE}",
  "groups": ["developers", "admins"],
  "iat": $(date +%s),
  "exp": $(($(date +%s) + 86400))
}
EOF
)

# Base64 encode
header_b64=$(echo -n "$header" | base64 | tr -d '=' | tr '+/' '-_')
payload_b64=$(echo -n "$payload" | base64 | tr -d '=' | tr '+/' '-_')

# Create signature
signature=$(echo -n "${header_b64}.${payload_b64}" | openssl dgst -sha256 -hmac "$JWT_SECRET" -binary | base64 | tr -d '=' | tr '+/' '-_')

# Complete JWT
jwt="${header_b64}.${payload_b64}.${signature}"

echo "Development JWT Token:"
echo "$jwt"
echo ""
echo "Use this in your API requests:"
echo "Authorization: Bearer $jwt"
```

**Make it executable:**
```bash
chmod +x scripts/generate-dev-token.sh
```

**Usage:**
```bash
# Generate token for admin user
./scripts/generate-dev-token.sh "admin@monti.local" "admin"

# Generate token for viewer user
./scripts/generate-dev-token.sh "viewer@monti.local" "viewer"
```

### Step 2: Environment Configuration

**Backend `.env` file:**
```env
# Local Development
ENV=development
JWT_SECRET=your-local-dev-secret-change-this-in-production
JWT_ISSUER=monti-local-dev

# Production (uncomment when deploying)
# ENV=production
# OIDC_ISSUER=https://your-aws-sso-url.awsapps.com/
# OIDC_CLIENT_ID=your-client-id
# OIDC_CLIENT_SECRET=your-client-secret
```

**Frontend `.env` file:**
```env
# Local Development
VITE_ENV=development
VITE_WS_URL=ws://localhost:8080/ws
VITE_API_URL=http://localhost:8080/api

# Auth - Local Development
VITE_AUTH_MODE=dev
VITE_DEV_TOKEN=your-jwt-token-here

# Auth - Production (uncomment when deploying)
# VITE_AUTH_MODE=oidc
# VITE_OIDC_ISSUER=https://your-aws-sso-url.awsapps.com/
# VITE_OIDC_CLIENT_ID=your-client-id
# VITE_OIDC_REDIRECT_URI=http://localhost:5173/callback
```

### Step 3: Backend Auth Middleware

**File: `Backend/internal/auth/middleware.go`**

```go
package auth

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Email  string   `json:"email"`
	Role   string   `json:"role"`
	Groups []string `json:"groups"`
	jwt.RegisteredClaims
}

type contextKey string

const UserContextKey contextKey = "user"

// Middleware validates JWT tokens
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health check
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		// In development mode, you can bypass auth
		if os.Getenv("ENV") == "development" && os.Getenv("SKIP_AUTH") == "true" {
			// Create a default dev user
			ctx := context.WithValue(r.Context(), UserContextKey, &Claims{
				Email:  "dev@monti.local",
				Role:   "admin",
				Groups: []string{"developers"},
			})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing authorization header", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		// Validate token
		claims, err := validateToken(tokenString)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid token: %v", err), http.StatusUnauthorized)
			return
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func validateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Return secret key
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			return nil, fmt.Errorf("JWT_SECRET not set")
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		// Check expiration
		if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
			return nil, fmt.Errorf("token expired")
		}
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
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
```

**File: `Backend/internal/auth/permissions.go`**

```go
package auth

import "fmt"

// Action represents an action a user can perform
type Action string

const (
	ViewDashboard   Action = "view_dashboard"
	ViewAgents      Action = "view_agents"
	ManageAgents    Action = "manage_agents"
	ViewTeams       Action = "view_teams"
	ManageTeams     Action = "manage_teams"
	ViewAnalytics   Action = "view_analytics"
	ManageSystem    Action = "manage_system"
)

// Can checks if user can perform an action
func Can(claims *Claims, action Action) bool {
	// Admin can do everything
	if claims.Role == "admin" {
		return true
	}

	// Manager permissions
	if claims.Role == "manager" {
		switch action {
		case ViewDashboard, ViewAgents, ViewTeams, ViewAnalytics:
			return true
		case ManageAgents, ManageTeams:
			return true
		}
	}

	// Viewer permissions
	if claims.Role == "viewer" {
		switch action {
		case ViewDashboard, ViewAgents, ViewTeams:
			return true
		}
	}

	return false
}

// RequirePermission is middleware that checks permissions
func RequirePermission(action Action) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := GetUserFromContext(r.Context())
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if !Can(claims, action) {
				http.Error(w, fmt.Sprintf("Forbidden: requires %s permission", action), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
```

### Step 4: Update Backend Server

**File: `Backend/cmd/server/main.go`**

```go
package main

import (
	"log"
	"net/http"
	"os"

	"monti/internal/auth"
	"monti/internal/handlers"
	"monti/internal/websocket"
)

func main() {
	// Initialize WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	// Create router
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Protected WebSocket endpoint
	mux.Handle("/ws", auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleWebSocket(hub, w, r)
	})))

	// Protected API endpoints
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/agents", handlers.GetAgents)
	apiMux.HandleFunc("/teams", handlers.GetTeams)

	// Apply auth middleware to all /api routes
	mux.Handle("/api/", http.StripPrefix("/api", auth.Middleware(apiMux)))

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Printf("Environment: %s", os.Getenv("ENV"))
	if os.Getenv("SKIP_AUTH") == "true" {
		log.Println("⚠️  WARNING: Authentication is DISABLED")
	}

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
```

### Step 5: Frontend Auth Service

**File: `Frontend/src/services/auth.ts`**

```typescript
export interface User {
  email: string
  role: string
  groups: string[]
}

export interface AuthState {
  isAuthenticated: boolean
  user: User | null
  token: string | null
}

class AuthService {
  private token: string | null = null
  private user: User | null = null

  // Initialize auth
  async init(): Promise<AuthState> {
    const env = import.meta.env.VITE_ENV || 'development'
    const authMode = import.meta.env.VITE_AUTH_MODE || 'dev'

    if (env === 'development' && authMode === 'dev') {
      // Use dev token from .env or generate one
      this.token = import.meta.env.VITE_DEV_TOKEN || this.generateDevToken()
      this.user = this.decodeToken(this.token)

      // Store in localStorage for persistence
      localStorage.setItem('auth_token', this.token)

      return {
        isAuthenticated: true,
        user: this.user,
        token: this.token,
      }
    }

    // Production: Check for token in localStorage
    const storedToken = localStorage.getItem('auth_token')
    if (storedToken) {
      this.token = storedToken
      this.user = this.decodeToken(storedToken)

      // Validate token is not expired
      if (this.isTokenValid()) {
        return {
          isAuthenticated: true,
          user: this.user,
          token: this.token,
        }
      }
    }

    // Not authenticated
    return {
      isAuthenticated: false,
      user: null,
      token: null,
    }
  }

  // Get current token
  getToken(): string | null {
    return this.token
  }

  // Get current user
  getUser(): User | null {
    return this.user
  }

  // Check if user has permission
  can(action: string): boolean {
    if (!this.user) return false

    // Admin can do everything
    if (this.user.role === 'admin') return true

    // Define permissions per role
    const permissions: Record<string, string[]> = {
      manager: ['view_dashboard', 'view_agents', 'view_teams', 'manage_agents'],
      viewer: ['view_dashboard', 'view_agents', 'view_teams'],
    }

    const rolePermissions = permissions[this.user.role] || []
    return rolePermissions.includes(action)
  }

  // Logout
  logout(): void {
    this.token = null
    this.user = null
    localStorage.removeItem('auth_token')
  }

  // Private: Generate dev token (basic, not secure)
  private generateDevToken(): string {
    // This is a mock token for development
    // In production, you'll get real tokens from IAM Identity Center
    return import.meta.env.VITE_DEV_TOKEN || 'dev-token-placeholder'
  }

  // Private: Decode JWT token
  private decodeToken(token: string): User | null {
    try {
      const payload = token.split('.')[1]
      const decoded = JSON.parse(atob(payload))

      return {
        email: decoded.email || 'unknown',
        role: decoded.role || 'viewer',
        groups: decoded.groups || [],
      }
    } catch (error) {
      console.error('Failed to decode token:', error)
      return null
    }
  }

  // Private: Check if token is valid
  private isTokenValid(): boolean {
    if (!this.token) return false

    try {
      const payload = this.token.split('.')[1]
      const decoded = JSON.parse(atob(payload))
      const exp = decoded.exp * 1000 // Convert to milliseconds

      return Date.now() < exp
    } catch {
      return false
    }
  }
}

export const authService = new AuthService()
```

### Step 6: Update Frontend to Use Auth

**File: `Frontend/src/hooks/useAuth.ts`**

```typescript
import { useState, useEffect } from 'react'
import { authService, type AuthState } from '../services/auth'

export const useAuth = () => {
  const [authState, setAuthState] = useState<AuthState>({
    isAuthenticated: false,
    user: null,
    token: null,
  })
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const initAuth = async () => {
      const state = await authService.init()
      setAuthState(state)
      setLoading(false)
    }

    initAuth()
  }, [])

  const can = (action: string) => authService.can(action)
  const logout = () => {
    authService.logout()
    setAuthState({
      isAuthenticated: false,
      user: null,
      token: null,
    })
  }

  return {
    ...authState,
    loading,
    can,
    logout,
  }
}
```

**Update `Frontend/src/hooks/useWebSocket.ts`** to include auth token:

```typescript
// Add this to the connect function
const connect = useCallback(() => {
  if (wsServiceRef.current) {
    // Get auth token
    const token = authService.getToken()

    // Add token to WebSocket URL (you can also send it after connection)
    const wsUrl = token
      ? `${url}?token=${token}`
      : url

    wsServiceRef.current.connect()
  }
}, [url])
```

---

## Testing Auth Locally

### Quick Start for Local Development

1. **Set SKIP_AUTH mode (easiest):**

```bash
# Backend/.env
ENV=development
SKIP_AUTH=true
```

This bypasses auth completely with a default dev user.

2. **Or use JWT tokens:**

Generate a token:
```bash
./scripts/generate-dev-token.sh "your-email@company.com" "admin"
```

Copy the token and add to `Frontend/.env`:
```env
VITE_DEV_TOKEN=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

3. **Start the application:**

```bash
# Terminal 1 - Backend
cd Backend
go run cmd/server/main.go

# Terminal 2 - Frontend
cd Frontend
npm run dev
```

4. **Test different roles:**

Generate tokens with different roles:
```bash
# Admin user
./scripts/generate-dev-token.sh "admin@monti.local" "admin"

# Manager user
./scripts/generate-dev-token.sh "manager@monti.local" "manager"

# Viewer user
./scripts/generate-dev-token.sh "viewer@monti.local" "viewer"
```

Update your `.env` file and reload the frontend to test different permissions.

---

## Production Setup with AWS IAM Identity Center

When you're ready to deploy to production, you'll need to set up AWS IAM Identity Center (formerly AWS SSO).

### Prerequisites
1. AWS Account with admin access
2. AWS IAM Identity Center enabled
3. User directory configured (AWS Managed or external like Active Directory)

### Steps for Production

#### 1. Enable IAM Identity Center

```bash
# In AWS Console
1. Go to IAM Identity Center
2. Click "Enable"
3. Choose your identity source (AWS Managed or External)
```

#### 2. Create Application in IAM Identity Center

```bash
# In IAM Identity Center Console
1. Go to Applications → Add Application
2. Choose "Custom SAML 2.0 application" or "OAuth 2.0"
3. Name: "MONTI Dashboard"
4. Redirect URLs:
   - https://your-domain.com/callback
   - http://localhost:5173/callback (for local testing)
5. Note the Client ID and Client Secret
```

#### 3. Configure Groups and Permissions

```bash
# Create groups in IAM Identity Center
1. Groups → Create Group
   - monti-admins
   - monti-managers
   - monti-viewers

2. Add users to groups
3. Assign the MONTI application to these groups
```

#### 4. Update Backend for OIDC

You'll need to modify the auth middleware to support OIDC token validation using AWS Cognito's public keys.

#### 5. Update Frontend for OIDC Flow

Implement the OAuth2 authorization code flow in your frontend.

---

## Role & Permission Model

Here's the recommended role structure:

| Role      | Permissions                                              |
|-----------|----------------------------------------------------------|
| **Admin** | Full access - view, manage agents, teams, system config  |
| **Manager**| View dashboard, agents, teams + manage agents/teams     |
| **Viewer**| View only - dashboard, agents, teams (read-only)        |

### Group to Role Mapping

```
IAM Identity Center Group → Backend Role
monti-admins            → admin
monti-managers          → manager
monti-viewers           → viewer
```

---

## Next Steps

1. **Start with local development** using SKIP_AUTH or self-signed JWTs
2. **Implement auth middleware** in the backend
3. **Add auth service** to the frontend
4. **Test locally** with different roles
5. **Set up IAM Identity Center** when ready for production
6. **Update to OIDC** flow for production deployment

---

## Useful Commands

```bash
# Generate dev token
./scripts/generate-dev-token.sh "user@example.com" "admin"

# Run backend with auth disabled
SKIP_AUTH=true go run cmd/server/main.go

# Run backend with JWT validation
ENV=development JWT_SECRET=your-secret go run cmd/server/main.go

# Test authenticated endpoint
curl -H "Authorization: Bearer YOUR_TOKEN" http://localhost:8080/api/agents
```

---

## Troubleshooting

**Issue: "Invalid token" error**
- Check JWT_SECRET matches between token generator and backend
- Verify token hasn't expired (default: 24 hours)
- Ensure token format is correct (Bearer token)

**Issue: "Unauthorized" even with SKIP_AUTH=true**
- Check .env file is loaded correctly
- Verify ENV=development is set
- Restart backend server after changing .env

**Issue: Frontend can't connect to WebSocket**
- Check if auth token is being sent with WebSocket connection
- Verify CORS settings in backend
- Check browser console for error messages

---

## Security Notes

⚠️ **Important for Production:**
- Never commit JWT_SECRET to git
- Use strong secrets (minimum 32 characters)
- Rotate secrets regularly
- Use HTTPS in production
- Set appropriate token expiration times
- Implement token refresh mechanism
- Use environment-specific secrets (dev/staging/prod)
