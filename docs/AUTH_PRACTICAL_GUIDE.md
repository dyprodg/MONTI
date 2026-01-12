# Practical Auth Guide for MONTI

This guide explains the **realistic authentication flow** for MONTI that works identically in development and production.

---

## The Problem with My Previous Approach

❌ **Bad**: Build a token-based system, then rebuild for OIDC later
✅ **Good**: Build with OIDC from day 1, use mock provider for local dev

---

## How It Actually Works

### User Experience (Same in Dev and Production)

1. **User opens the app** → `http://localhost:5173` (dev) or `https://monti.company.com` (prod)
2. **App checks for authentication** → No token found
3. **Redirect to login** → User sees "Sign in with AWS" button
4. **Click sign in** → Redirects to:
   - **Dev**: Mock login page (you run locally)
   - **Prod**: AWS IAM Identity Center login screen
5. **User enters credentials** → Submits login
6. **Provider validates** → Creates token
7. **Redirect back to app** → `http://localhost:5173/callback?code=abc123`
8. **App exchanges code for token** → Standard OAuth2 flow
9. **Token stored in browser** → LocalStorage or SessionStorage
10. **Dashboard loads** → Token sent with every API/WebSocket request

### Where Does the Token Come From?

The token comes from the **OIDC provider** (identity provider):
- **Local Dev**: A simple mock server you run that pretends to be AWS SSO
- **Production**: Real AWS IAM Identity Center

---

## Recommended Solution: Use Keycloak for Local Development

**Keycloak** is an open-source identity provider that mimics AWS IAM Identity Center perfectly. It runs in Docker locally.

### Why Keycloak?

- ✅ Full OIDC/OAuth2 support (same as AWS)
- ✅ Web UI for managing users/roles
- ✅ Runs in Docker (no complex setup)
- ✅ Your code stays identical between dev and prod
- ✅ Just change the OIDC URL in config

---

## Step-by-Step Setup

### Step 1: Add Keycloak to Docker Compose

Update your `docker-compose.yml`:

```yaml
version: '3.8'

services:
  backend:
    build: ./Backend
    ports:
      - "8080:8080"
    environment:
      - ENV=development
      - OIDC_ISSUER=http://keycloak:8180/realms/monti
      - OIDC_CLIENT_ID=monti-app
      - OIDC_CLIENT_SECRET=dev-secret-change-in-prod
    depends_on:
      - keycloak
    networks:
      - monti-network

  frontend:
    build: ./Frontend
    ports:
      - "5173:5173"
    environment:
      - VITE_WS_URL=ws://localhost:8080/ws
      - VITE_API_URL=http://localhost:8080/api
      - VITE_OIDC_ISSUER=http://localhost:8180/realms/monti
      - VITE_OIDC_CLIENT_ID=monti-app
      - VITE_OIDC_REDIRECT_URI=http://localhost:5173/callback
    networks:
      - monti-network

  # Local identity provider
  keycloak:
    image: quay.io/keycloak/keycloak:23.0
    command: start-dev
    ports:
      - "8180:8180"
    environment:
      - KEYCLOAK_ADMIN=admin
      - KEYCLOAK_ADMIN_PASSWORD=admin
      - KC_HTTP_PORT=8180
      - KC_HOSTNAME=localhost
      - KC_HOSTNAME_STRICT=false
      - KC_HOSTNAME_STRICT_HTTPS=false
    networks:
      - monti-network

networks:
  monti-network:
    driver: bridge
```

### Step 2: Configure Keycloak

After starting Docker Compose:

```bash
docker compose up -d
```

1. **Open Keycloak Admin Console**: http://localhost:8180
2. **Login**: admin / admin
3. **Create Realm**:
   - Click dropdown (top left) → "Create Realm"
   - Name: `monti`
   - Click "Create"

4. **Create Client** (this is your app):
   - Clients → Create Client
   - Client ID: `monti-app`
   - Client Protocol: `openid-connect`
   - Next
   - Enable: "Standard flow", "Direct access grants"
   - Valid redirect URIs:
     - `http://localhost:5173/*`
     - `http://localhost:5173/callback`
   - Web Origins: `http://localhost:5173`
   - Save
   - Go to "Credentials" tab
   - Copy the Client Secret

5. **Create Users**:
   - Users → Create User
   - Username: `admin@monti.local`
   - Email: `admin@monti.local`
   - Email verified: ON
   - Save
   - Go to "Credentials" tab
   - Set password: `admin123` (uncheck "Temporary")

6. **Create Roles**:
   - Realm Roles → Create Role
   - Role name: `admin`
   - Save
   - Repeat for: `manager`, `viewer`

7. **Assign Role to User**:
   - Users → admin@monti.local → Role Mapping
   - Assign role: `admin`

8. **Create Groups** (optional but recommended):
   - Groups → Create Group
   - Name: `monti-admins`
   - Add members → Select admin@monti.local

### Step 3: Update Frontend

Install OIDC library:

```bash
cd Frontend
npm install oidc-client-ts
```

Create auth service:

**File: `Frontend/src/services/auth.ts`**

```typescript
import { UserManager, User as OidcUser, WebStorageStateStore } from 'oidc-client-ts'

export interface User {
  email: string
  name: string
  role: string
  groups: string[]
}

class AuthService {
  private userManager: UserManager

  constructor() {
    const config = {
      authority: import.meta.env.VITE_OIDC_ISSUER || 'http://localhost:8180/realms/monti',
      client_id: import.meta.env.VITE_OIDC_CLIENT_ID || 'monti-app',
      redirect_uri: import.meta.env.VITE_OIDC_REDIRECT_URI || 'http://localhost:5173/callback',
      post_logout_redirect_uri: window.location.origin,
      response_type: 'code',
      scope: 'openid profile email roles',
      userStore: new WebStorageStateStore({ store: window.localStorage }),
      automaticSilentRenew: true,
      silent_redirect_uri: `${window.location.origin}/silent-renew.html`,
    }

    this.userManager = new UserManager(config)
  }

  // Start login flow
  async login(): Promise<void> {
    await this.userManager.signinRedirect()
  }

  // Handle callback after login
  async handleCallback(): Promise<User | null> {
    try {
      const user = await this.userManager.signinRedirectCallback()
      return this.mapUser(user)
    } catch (error) {
      console.error('Login callback error:', error)
      return null
    }
  }

  // Get current user
  async getUser(): Promise<User | null> {
    const oidcUser = await this.userManager.getUser()
    if (!oidcUser || oidcUser.expired) {
      return null
    }
    return this.mapUser(oidcUser)
  }

  // Get access token
  async getToken(): Promise<string | null> {
    const user = await this.userManager.getUser()
    return user?.access_token || null
  }

  // Logout
  async logout(): Promise<void> {
    await this.userManager.signoutRedirect()
  }

  // Check if user is authenticated
  async isAuthenticated(): Promise<boolean> {
    const user = await this.userManager.getUser()
    return user !== null && !user.expired
  }

  // Map OIDC user to our User type
  private mapUser(oidcUser: OidcUser): User {
    const profile = oidcUser.profile

    return {
      email: profile.email as string,
      name: profile.name as string || profile.preferred_username as string,
      role: this.extractRole(profile),
      groups: this.extractGroups(profile),
    }
  }

  // Extract role from token claims
  private extractRole(profile: any): string {
    // Check different possible locations for roles
    // Keycloak puts roles in realm_access.roles
    if (profile.realm_access?.roles) {
      const roles = profile.realm_access.roles
      if (roles.includes('admin')) return 'admin'
      if (roles.includes('manager')) return 'manager'
      if (roles.includes('viewer')) return 'viewer'
    }

    // AWS Cognito puts roles in cognito:groups
    if (profile['cognito:groups']) {
      const groups = profile['cognito:groups']
      if (groups.includes('monti-admins')) return 'admin'
      if (groups.includes('monti-managers')) return 'manager'
      if (groups.includes('monti-viewers')) return 'viewer'
    }

    // Default to viewer
    return 'viewer'
  }

  // Extract groups from token claims
  private extractGroups(profile: any): string[] {
    return profile.groups || profile['cognito:groups'] || []
  }
}

export const authService = new AuthService()
```

### Step 4: Create Auth Context

**File: `Frontend/src/contexts/AuthContext.tsx`**

```typescript
import { createContext, useContext, useEffect, useState, ReactNode } from 'react'
import { authService, User } from '../services/auth'

interface AuthContextType {
  user: User | null
  loading: boolean
  isAuthenticated: boolean
  login: () => Promise<void>
  logout: () => Promise<void>
  getToken: () => Promise<string | null>
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const initAuth = async () => {
      try {
        const currentUser = await authService.getUser()
        setUser(currentUser)
      } catch (error) {
        console.error('Auth init error:', error)
      } finally {
        setLoading(false)
      }
    }

    initAuth()
  }, [])

  const login = async () => {
    await authService.login()
  }

  const logout = async () => {
    await authService.logout()
    setUser(null)
  }

  const getToken = async () => {
    return await authService.getToken()
  }

  return (
    <AuthContext.Provider
      value={{
        user,
        loading,
        isAuthenticated: user !== null,
        login,
        logout,
        getToken,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export const useAuth = () => {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider')
  }
  return context
}
```

### Step 5: Create Login and Callback Pages

**File: `Frontend/src/pages/Login.tsx`**

```tsx
import { useEffect } from 'react'
import { useAuth } from '../contexts/AuthContext'
import { useNavigate } from 'react-router-dom'

export const Login = () => {
  const { login, isAuthenticated } = useAuth()
  const navigate = useNavigate()

  useEffect(() => {
    if (isAuthenticated) {
      navigate('/')
    }
  }, [isAuthenticated, navigate])

  const handleLogin = async () => {
    await login()
  }

  return (
    <div style={{
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      minHeight: '100vh',
      backgroundColor: '#f9fafb'
    }}>
      <div style={{
        backgroundColor: 'white',
        padding: '48px',
        borderRadius: '12px',
        boxShadow: '0 1px 3px 0 rgb(0 0 0 / 0.1)',
        textAlign: 'center'
      }}>
        <h1 style={{ fontSize: '36px', marginBottom: '16px' }}>MONTI</h1>
        <p style={{ color: '#6b7280', marginBottom: '32px' }}>
          Live Call Center Monitoring
        </p>

        <button
          onClick={handleLogin}
          style={{
            backgroundColor: '#3b82f6',
            color: 'white',
            padding: '12px 32px',
            borderRadius: '8px',
            border: 'none',
            fontSize: '16px',
            fontWeight: '600',
            cursor: 'pointer'
          }}
        >
          Sign in with SSO
        </button>
      </div>
    </div>
  )
}
```

**File: `Frontend/src/pages/Callback.tsx`**

```tsx
import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { authService } from '../services/auth'

export const Callback = () => {
  const navigate = useNavigate()

  useEffect(() => {
    const handleCallback = async () => {
      try {
        await authService.handleCallback()
        navigate('/')
      } catch (error) {
        console.error('Callback error:', error)
        navigate('/login')
      }
    }

    handleCallback()
  }, [navigate])

  return (
    <div style={{
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      minHeight: '100vh'
    }}>
      <p>Processing login...</p>
    </div>
  )
}
```

### Step 6: Update App.tsx with Routes

```tsx
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider, useAuth } from './contexts/AuthContext'
import { Login } from './pages/Login'
import { Callback } from './pages/Callback'
import { Dashboard } from './pages/Dashboard' // Your existing dashboard

const ProtectedRoute = ({ children }: { children: React.ReactNode }) => {
  const { isAuthenticated, loading } = useAuth()

  if (loading) {
    return <div>Loading...</div>
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  return <>{children}</>
}

function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/callback" element={<Callback />} />
          <Route
            path="/"
            element={
              <ProtectedRoute>
                <Dashboard />
              </ProtectedRoute>
            }
          />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  )
}

export default App
```

---

## Testing Locally

1. **Start everything**:
```bash
docker compose up -d
```

2. **Open browser**: http://localhost:5173

3. **You'll be redirected to**: http://localhost:5173/login

4. **Click "Sign in with SSO"**

5. **Redirected to Keycloak**: http://localhost:8180/realms/monti/protocol/openid-connect/auth...

6. **Login**: admin@monti.local / admin123

7. **Redirected back**: http://localhost:5173/callback

8. **Then to dashboard**: http://localhost:5173/

9. **Token is stored** in localStorage automatically

---

## Switching to Production (AWS IAM Identity Center)

When you're ready for production, **only configuration changes** are needed:

### 1. Set up AWS IAM Identity Center

```bash
# AWS Console → IAM Identity Center → Applications
1. Create custom OIDC application
2. Name: MONTI
3. Redirect URIs: https://monti.company.com/callback
4. Note the Client ID and Issuer URL
```

### 2. Update Environment Variables

**Production `.env` file:**

```env
# Backend
OIDC_ISSUER=https://your-sso-portal.awsapps.com/
OIDC_CLIENT_ID=your-client-id
OIDC_CLIENT_SECRET=your-client-secret

# Frontend
VITE_OIDC_ISSUER=https://your-sso-portal.awsapps.com/
VITE_OIDC_CLIENT_ID=your-client-id
VITE_OIDC_REDIRECT_URI=https://monti.company.com/callback
```

### 3. Update Role Mapping

AWS uses groups differently. Update `extractRole()` in auth.ts:

```typescript
private extractRole(profile: any): string {
  // AWS IAM Identity Center puts groups here
  const groups = profile['custom:groups'] || profile.groups || []

  if (groups.includes('monti-admins')) return 'admin'
  if (groups.includes('monti-managers')) return 'manager'
  if (groups.includes('monti-viewers')) return 'viewer'

  return 'viewer'
}
```

**That's it!** No code changes, just configuration.

---

## Comparison: Dev vs Production

| Aspect | Local Dev (Keycloak) | Production (AWS IAM) |
|--------|---------------------|---------------------|
| OIDC Provider | Keycloak (Docker) | AWS IAM Identity Center |
| Users | Created manually in Keycloak | Synced from company directory |
| Login URL | localhost:8180 | your-portal.awsapps.com |
| Code Changes | **NONE** | **NONE** |
| Config Changes | 3 environment variables | 3 environment variables |

---

## User Management

### Local Development
- Open Keycloak admin: http://localhost:8180
- Add users manually
- Assign roles
- Test different permissions

### Production
- Users managed in AWS IAM Identity Center
- Synced from Active Directory / Okta / Google Workspace
- IT admin assigns groups
- Groups mapped to roles in your app

---

## Benefits of This Approach

✅ **Identical flow** in dev and production
✅ **No code changes** when moving to production
✅ **Real OIDC testing** in development
✅ **Team can test auth** locally without AWS access
✅ **Standard OAuth2 flow** (industry best practice)
✅ **Works on any machine** - just needs Docker

---

## Quick Commands

```bash
# Start with auth
docker compose up -d

# Access Keycloak admin
open http://localhost:8180

# Access app
open http://localhost:5173

# Stop everything
docker compose down

# View logs
docker compose logs -f keycloak
docker compose logs -f backend
```

---

## What Happens on a Random Machine?

1. **Developer pulls code**
2. **Runs**: `docker compose up -d`
3. **Opens**: http://localhost:5173
4. **Sees**: Login page
5. **Clicks**: "Sign in with SSO"
6. **Redirected to**: Keycloak login (localhost:8180)
7. **Enters**: admin@monti.local / admin123
8. **Gets**: Access token automatically
9. **Redirected to**: Dashboard
10. **Token stored**: In browser localStorage
11. **Works**: Until token expires (typically 1 hour)

**Next time they open the app**: Already logged in (token still valid)!

---

## Summary

This approach gives you:
- ✅ Production-like auth in development
- ✅ Zero code changes for production
- ✅ Real login experience for testing
- ✅ Works on any machine with Docker
- ✅ Easy to explain to team members
- ✅ Industry standard (OIDC/OAuth2)
