# AUTH RECOVERY PLAN

## Problem
Keycloak was destroyed. All auth configuration lost. Need to reverse-engineer the permission system from code.

## Priority: CRITICAL

---

## Phase 1: Scan Backend Auth Implementation

### Files to Analyze
```
Backend/
├── cmd/server/main.go          # Auth middleware setup
├── internal/
│   ├── auth/                   # Auth package (if exists)
│   ├── middleware/             # JWT validation, role checks
│   ├── handlers/               # Permission checks in handlers
│   ├── websocket/              # WS handshake auth, agent filtering
│   └── types/                  # User, Role, Group types
```

### Questions to Answer
1. How does the backend validate JWT tokens?
2. What claims does it expect? (roles, groups, permissions)
3. How does WebSocket handshake use auth?
4. How are agents filtered by user permissions?
5. What role hierarchy exists? (admin > supervisor > ?)
6. How are location/department groups used for filtering?

---

## Phase 2: Scan Frontend Auth Implementation

### Files to Analyze
```
Frontend/
├── src/
│   ├── auth/                   # OIDC client setup
│   ├── context/                # Auth context, user state
│   ├── hooks/                  # useAuth, usePermissions
│   ├── components/             # Permission-gated components
│   └── types/                  # User, Permission types
```

### Questions to Answer
1. What OIDC claims does frontend read?
2. How does it pass auth to WebSocket?
3. What UI elements are permission-gated?
4. How does it filter dashboard data by group?

---

## Phase 3: Document Permission Structure

### Expected Structure (to verify)
```
Roles:
  - admin: Full access, all locations
  - supervisor: Full access, all locations
  - viewer/agent: Limited to assigned groups

Groups (Locations):
  - berlin
  - munich
  - hamburg
  - frankfurt
  - remote
  - all (special: sees everything)

Groups (Departments?):
  - sales
  - support
  - technical
  - retention
```

### Token Claims Structure (to verify)
```json
{
  "sub": "user-id",
  "preferred_username": "username",
  "realm_access": {
    "roles": ["admin", "supervisor", "viewer"]
  },
  "groups": ["/berlin", "/sales"],
  "resource_access": {
    "monti-app": {
      "roles": []
    }
  }
}
```

---

## Phase 4: Create Keycloak Setup Script

After understanding the auth system, create:

1. `scripts/setup-keycloak.sh` - Automated realm/client setup
2. `scripts/setup-users.sh` - Create all users with correct roles/groups
3. `keycloak/realm-export.json` - Exportable realm config for backup

---

## Phase 5: Document & Prevent Future Loss

1. Update CLAUDE.md with strict Keycloak rules
2. Create `docs/AUTH.md` with full permission documentation
3. Add realm export to git (sanitized, no secrets)
4. Add docker-compose health checks

---

## Grep Patterns to Search

```bash
# Backend auth patterns
grep -r "jwt" Backend/
grep -r "token" Backend/
grep -r "auth" Backend/
grep -r "role" Backend/
grep -r "group" Backend/
grep -r "permission" Backend/
grep -r "claims" Backend/
grep -r "middleware" Backend/
grep -r "handshake" Backend/

# Frontend auth patterns
grep -r "oidc" Frontend/src/
grep -r "auth" Frontend/src/
grep -r "token" Frontend/src/
grep -r "role" Frontend/src/
grep -r "group" Frontend/src/
grep -r "permission" Frontend/src/
grep -r "claims" Frontend/src/
```

---

## Files to Read First

1. `Backend/cmd/server/main.go` - Entry point, middleware chain
2. `Backend/internal/websocket/` - All files, WS auth
3. `Frontend/src/` - Look for auth/, context/, hooks/
4. Any file with "auth", "jwt", "permission" in name

---

## Expected Deliverables

1. [ ] Complete auth flow diagram
2. [ ] Keycloak realm configuration (groups, roles, client scopes)
3. [ ] User list with exact role/group assignments
4. [ ] Automated setup script that can recreate everything
5. [ ] Realm export JSON for backup
6. [ ] AUTH.md documentation

---

## REMINDER FOR CLAUDE

**NEVER AGAIN:**
- Delete Keycloak container
- Remove keycloak-data volume
- Run docker compose down -v
- Modify auth without explicit permission

**ALWAYS:**
- Ask before touching auth
- Export realm config before any changes
- Test auth changes in isolation first
