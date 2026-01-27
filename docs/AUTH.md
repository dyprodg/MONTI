# MONTI Authentication Guide

This document describes the authentication and authorization system for MONTI.

## Overview

MONTI uses **Keycloak** as the identity provider (IdP) with **OpenID Connect (OIDC)** protocol. Authentication flow uses **Authorization Code with PKCE** for maximum security in the browser-based SPA.

## Authentication Flow

```
┌─────────────┐     1. Login Click      ┌─────────────┐
│   Browser   │ ───────────────────────→│  Keycloak   │
│  (Frontend) │                         │    IdP      │
└─────────────┘                         └─────────────┘
      │                                       │
      │         2. Redirect to Login          │
      │←──────────────────────────────────────│
      │                                       │
      │         3. User enters creds          │
      │───────────────────────────────────────→
      │                                       │
      │    4. Auth Code (via redirect)        │
      │←──────────────────────────────────────│
      │                                       │
      │    5. Exchange code for tokens        │
      │───────────────────────────────────────→
      │       (with PKCE code_verifier)       │
      │                                       │
      │    6. Access Token + Refresh Token    │
      │←──────────────────────────────────────│
      │                                       │
┌─────────────┐                         ┌─────────────┐
│   Browser   │   7. API Request        │   Backend   │
│  (Frontend) │───────────────────────→│   Server    │
└─────────────┘   Authorization: Bearer │             │
                                        └─────────────┘
                                              │
                              8. Verify signature via JWKS
                                              │
                              9. Extract claims & authorize
```

### Flow Details

1. User clicks "Login" in the frontend
2. Frontend generates PKCE `code_verifier` and `code_challenge`
3. Browser redirects to Keycloak with `code_challenge`
4. User authenticates in Keycloak
5. Keycloak redirects back with authorization code
6. Frontend exchanges code + `code_verifier` for tokens
7. Frontend stores tokens and makes API requests with `Authorization: Bearer <token>`
8. Backend verifies JWT signature using Keycloak's JWKS
9. Backend extracts claims and authorizes the request

## Role Definitions

| Role | Description | Access Level |
|------|-------------|--------------|
| `admin` | System administrator | All agents, all locations, all features |
| `supervisor` | Team supervisor | Multiple business units, can view/manage assigned agents |
| `agent` | Call center agent | Own business unit, limited features |
| `viewer` | Read-only access | View-only for assigned locations |

### Role Hierarchy

```
admin > supervisor > agent > viewer
```

An admin can do everything a supervisor can do, and so on.

## Business Units & Locations

Business units control which physical locations (call centers) a user can access.

### Business Unit Mapping

| Business Unit | Code | Locations |
|---------------|------|-----------|
| Süd-Geschäftsbereich | SGB | Munich, Frankfurt |
| Nord-Geschäftsbereich | NGB | Berlin, Hamburg |
| Remote-Geschäftsbereich | RGB | Remote |

### User Access Examples

| User | Role | Groups | Can See |
|------|------|--------|---------|
| admin | admin | (none) | All locations |
| supervisor | supervisor | SGB, NGB | Munich, Frankfurt, Berlin, Hamburg |
| agent | agent | SGB | Munich, Frankfurt |
| demo | viewer | RGB | Remote only |

## JWT Token Structure

Access tokens from Keycloak contain these claims:

```json
{
  "exp": 1699999999,
  "iat": 1699996399,
  "sub": "user-uuid",
  "email": "user@example.com",
  "preferred_username": "username",
  "name": "Full Name",
  "realm_access": {
    "roles": ["agent", "default-roles-monti"]
  },
  "groups": [
    "/business-units/SGB",
    "/business-units/NGB"
  ]
}
```

### Important Claims

| Claim | Description |
|-------|-------------|
| `email` | User's email address |
| `preferred_username` | Username |
| `name` | Display name |
| `realm_access.roles` | Array of assigned realm roles |
| `groups` | Array of group paths (includes business units) |

The backend extracts business units from the `groups` claim by looking for paths matching `/business-units/<BU>`.

## Development Setup

### Prerequisites

- Docker and Docker Compose
- Keycloak running on port 8180

### Quick Start

```bash
# 1. Start all services
docker compose up -d

# 2. Wait for Keycloak to be ready
docker compose logs -f keycloak
# Wait until you see "Keycloak started"

# 3. Run setup script
./scripts/setup-keycloak.sh

# 4. (Optional) Export config for backup
./scripts/export-keycloak.sh

# 5. Open the app
open http://localhost:5173
```

### Test Users

| Username | Password | Role | Groups |
|----------|----------|------|--------|
| admin | admin | admin | (none - sees all) |
| supervisor | supervisor | supervisor | SGB, NGB |
| agent | agent | agent | SGB |
| demo | demo | viewer | RGB |

### Keycloak Admin UI

- URL: http://localhost:8180/admin
- Username: admin
- Password: admin
- Realm: monti (select from dropdown after login)

### Environment Variables

Backend environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `ENV` | Environment (development/production) | - |
| `SKIP_AUTH` | Bypass authentication entirely | false |
| `VERIFY_JWT_SIGNATURE` | Force JWT signature verification | auto (true in prod) |
| `OIDC_ISSUER` | Keycloak realm URL | http://localhost:8180/realms/monti |

Frontend environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `VITE_OIDC_ISSUER` | Keycloak realm URL | http://localhost:8180/realms/monti |
| `VITE_OIDC_CLIENT_ID` | OIDC client ID | monti-app |

## Production Configuration

### Production URLs

| Service | URL |
|---------|-----|
| Keycloak Issuer | `https://montibackend.dennisdiepolder.com/realms/monti` |
| Keycloak Admin | `https://montibackend.dennisdiepolder.com/admin` |
| JWKS Endpoint | `https://montibackend.dennisdiepolder.com/realms/monti/protocol/openid-connect/certs` |
| Frontend Redirect | `https://monti.dennisdiepolder.com` |

Caddy on the EC2 instance reverse-proxies `/realms/*`, `/admin/*`, `/resources/*`, `/js/*` to Keycloak, so Keycloak is accessible via the backend domain.

### Checklist

- [x] HTTPS/TLS termination configured (Caddy with automatic certs)
- [ ] `ENV=production` set on backend
- [ ] `VERIFY_JWT_SIGNATURE=true` (automatic in production)
- [ ] `OIDC_ISSUER` points to `https://montibackend.dennisdiepolder.com/realms/monti`
- [ ] Keycloak has production admin credentials (not admin/admin)
- [ ] Token lifespans reviewed for production
- [ ] Realm exported and backed up
- [ ] CORS origins include `https://monti.dennisdiepolder.com`

### Security Considerations

1. **JWT Verification**: In production, all JWT tokens are verified using JWKS from Keycloak. The backend fetches and caches the public keys.

2. **Token Lifespans**: Default access token lifespan is 1 hour. Adjust in Keycloak realm settings for production.

3. **PKCE**: The frontend uses PKCE (S256) to prevent authorization code interception.

4. **Groups Claim**: Ensure the `groups` protocol mapper is configured on the client to include group membership in tokens.

## Disaster Recovery

### If Keycloak Gets Messed Up

Option 1: Keep data and re-run setup
```bash
docker compose down
docker compose up -d
./scripts/setup-keycloak.sh
```

Option 2: Nuclear option - full reset
```bash
docker compose down
docker volume rm monti_keycloak-data
docker compose up -d
./scripts/setup-keycloak.sh
```

### Backup & Restore

Export realm config (run after setup or before changes):
```bash
./scripts/export-keycloak.sh
# Creates keycloak/realm-export.json
```

Import realm config (after fresh Keycloak install):
```bash
./scripts/import-keycloak.sh
./scripts/setup-keycloak.sh  # Recreate users
```

## Troubleshooting

### Token not being verified

Check environment:
```bash
# Backend should have
ENV=production
OIDC_ISSUER=http://keycloak:8180/realms/monti
```

Check JWKS endpoint is accessible from backend:
```bash
curl http://localhost:8180/realms/monti/protocol/openid-connect/certs
```

### Groups not in token

1. Go to Keycloak Admin → Clients → monti-app → Client scopes
2. Verify "groups" mapper exists with:
   - Mapper type: Group Membership
   - Full group path: ON
   - Add to access token: ON

Or re-run `./scripts/setup-keycloak.sh` to recreate the mapper.

### User can't see expected locations

1. Check user's group memberships in Keycloak Admin → Users → [user] → Groups
2. Verify group path is `/business-units/SGB` (not just `SGB`)
3. Check the token contains correct groups claim

### CORS errors

Ensure Keycloak client has correct web origins:
- `http://localhost:5173` (development)
- `https://monti.dennisdiepolder.com` (production)

## API Endpoints

### Protected Endpoints

All API endpoints except `/health` require authentication.

```
Authorization: Bearer <access_token>
```

### WebSocket Authentication

WebSocket connections pass the token as a query parameter:

```
ws://localhost:8080/ws?token=<access_token>
```

## Code References

- Backend middleware: `Backend/internal/auth/middleware.go`
- Types & mappings: `Backend/internal/types/types.go`
- Keycloak setup: `scripts/setup-keycloak.sh`
