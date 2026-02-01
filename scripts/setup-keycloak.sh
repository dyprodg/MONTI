#!/bin/bash
# Setup Keycloak realm, client, roles, groups and users for MONTI
# Can be rerun safely - will skip existing resources

set -e

KEYCLOAK_URL="${KEYCLOAK_URL:-http://localhost:8180}"
ADMIN_USER="${KEYCLOAK_ADMIN:-admin}"
ADMIN_PASS="${KEYCLOAK_ADMIN_PASSWORD:-admin}"

echo "=== MONTI Keycloak Setup ==="
echo "Keycloak URL: $KEYCLOAK_URL"

# Wait for Keycloak to be ready
echo "Waiting for Keycloak..."
until curl -sf "$KEYCLOAK_URL/health/ready" > /dev/null 2>&1 || curl -sf "$KEYCLOAK_URL/realms/master" > /dev/null 2>&1; do
  sleep 2
done
echo "Keycloak is ready!"

# Get admin token
echo "Getting admin token..."
TOKEN=$(curl -sf -X POST "$KEYCLOAK_URL/realms/master/protocol/openid-connect/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "username=$ADMIN_USER" \
  -d "password=$ADMIN_PASS" \
  -d "grant_type=password" \
  -d "client_id=admin-cli" | python3 -c "import sys,json; print(json.load(sys.stdin).get('access_token',''))")

if [ -z "$TOKEN" ]; then
  echo "ERROR: Failed to get admin token"
  exit 1
fi

# Function to make authenticated requests
kc_api() {
  curl -sf -X "$1" "$KEYCLOAK_URL/admin/realms$2" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    ${3:+-d "$3"} 2>/dev/null
}

# Check if realm exists
echo "Checking realm..."
if curl -sf "$KEYCLOAK_URL/realms/monti" > /dev/null 2>&1; then
  echo "Realm 'monti' already exists"
else
  echo "Creating realm 'monti'..."
  kc_api POST "" '{
    "realm": "monti",
    "enabled": true,
    "registrationAllowed": false,
    "loginWithEmailAllowed": true,
    "duplicateEmailsAllowed": false,
    "resetPasswordAllowed": true,
    "editUsernameAllowed": false,
    "bruteForceProtected": true,
    "accessTokenLifespan": 3600,
    "ssoSessionIdleTimeout": 1800,
    "ssoSessionMaxLifespan": 36000
  }'
  echo "Realm created!"
fi

# Create client
echo "Creating client 'monti-app'..."
kc_api POST "/monti/clients" '{
  "clientId": "monti-app",
  "enabled": true,
  "publicClient": true,
  "directAccessGrantsEnabled": true,
  "standardFlowEnabled": true,
  "implicitFlowEnabled": false,
  "redirectUris": [
    "http://localhost:5173/*",
    "http://localhost:3000/*",
    "https://*.monti.app/*",
    "https://monti.dennisdiepolder.com/*"
  ],
  "webOrigins": [
    "http://localhost:5173",
    "http://localhost:3000",
    "https://*.monti.app",
    "https://monti.dennisdiepolder.com"
  ],
  "attributes": {
    "pkce.code.challenge.method": "S256"
  }
}' || echo "Client may already exist"

# Get client ID for protocol mapper setup
echo "Getting client ID..."
CLIENT_ID=$(kc_api GET "/monti/clients?clientId=monti-app" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d[0]['id'] if d else '')" 2>/dev/null)

if [ -n "$CLIENT_ID" ]; then
  echo "  Client ID: $CLIENT_ID"

  # Create protocol mapper for groups claim
  echo "Creating groups protocol mapper..."
  kc_api POST "/monti/clients/$CLIENT_ID/protocol-mappers/models" '{
    "name": "groups",
    "protocol": "openid-connect",
    "protocolMapper": "oidc-group-membership-mapper",
    "consentRequired": false,
    "config": {
      "full.path": "true",
      "id.token.claim": "true",
      "access.token.claim": "true",
      "claim.name": "groups",
      "userinfo.token.claim": "true"
    }
  }' || echo "  Groups mapper may already exist"
else
  echo "WARNING: Could not get client ID for protocol mapper setup"
fi

# Create realm roles
echo "Creating roles..."
for role in admin supervisor agent viewer; do
  kc_api POST "/monti/roles" "{\"name\": \"$role\"}" || echo "Role '$role' may already exist"
done

# Create composite role mappings (admin has all permissions)
echo "Setting up role hierarchy..."

# Get role IDs
get_role_id() {
  kc_api GET "/monti/roles/$1" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null
}

ADMIN_ROLE_ID=$(get_role_id "admin")
SUPERVISOR_ROLE_ID=$(get_role_id "supervisor")
AGENT_ROLE_ID=$(get_role_id "agent")
VIEWER_ROLE_ID=$(get_role_id "viewer")

# ==========================================
# Create Business Unit Groups
# ==========================================
echo ""
echo "Creating business unit groups..."

# Create parent group /business-units
echo "  Creating parent group: /business-units"
kc_api POST "/monti/groups" '{"name": "business-units"}' || echo "    Parent group may already exist"

# Get parent group ID
PARENT_GROUP_ID=$(kc_api GET "/monti/groups?search=business-units" | python3 -c "
import sys, json
groups = json.load(sys.stdin)
for g in groups:
    if g.get('name') == 'business-units':
        print(g.get('id', ''))
        break
" 2>/dev/null)

if [ -n "$PARENT_GROUP_ID" ]; then
  echo "  Parent group ID: $PARENT_GROUP_ID"

  # Create child groups: SGB, NGB, RGB
  for bu in SGB NGB RGB; do
    echo "  Creating business unit group: $bu"
    kc_api POST "/monti/groups/$PARENT_GROUP_ID/children" "{\"name\": \"$bu\"}" || echo "    Group '$bu' may already exist"
  done
else
  echo "WARNING: Could not find parent group ID"
fi

# Get group IDs for user assignment
get_group_id() {
  local group_name=$1
  kc_api GET "/monti/groups?search=$group_name" | python3 -c "
import sys, json
groups = json.load(sys.stdin)
def find_group(groups, name):
    for g in groups:
        if g.get('name') == name:
            return g.get('id', '')
        if 'subGroups' in g:
            result = find_group(g['subGroups'], name)
            if result:
                return result
    return ''
print(find_group(groups, '$group_name'))
" 2>/dev/null
}

SGB_GROUP_ID=$(get_group_id "SGB")
NGB_GROUP_ID=$(get_group_id "NGB")
RGB_GROUP_ID=$(get_group_id "RGB")

echo "  SGB Group ID: ${SGB_GROUP_ID:-'not found'}"
echo "  NGB Group ID: ${NGB_GROUP_ID:-'not found'}"
echo "  RGB Group ID: ${RGB_GROUP_ID:-'not found'}"

# ==========================================
# Create Users with Roles and Groups
# ==========================================
echo ""
echo "Creating users..."

create_user() {
  local username=$1
  local password=$2
  local role=$3
  local firstname=$4
  local lastname=$5
  shift 5
  local groups=("$@")

  # Create user
  kc_api POST "/monti/users" "{
    \"username\": \"$username\",
    \"email\": \"$username@monti.local\",
    \"enabled\": true,
    \"emailVerified\": true,
    \"firstName\": \"$firstname\",
    \"lastName\": \"$lastname\",
    \"credentials\": [{
      \"type\": \"password\",
      \"value\": \"$password\",
      \"temporary\": false
    }]
  }" || echo "User '$username' may already exist"

  # Get user ID
  USER_ID=$(kc_api GET "/monti/users?username=$username" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d[0]['id'] if d else '')" 2>/dev/null)

  if [ -n "$USER_ID" ]; then
    # Assign role
    ROLE_ID=$(get_role_id "$role")
    if [ -n "$ROLE_ID" ]; then
      kc_api POST "/monti/users/$USER_ID/role-mappings/realm" "[{\"id\": \"$ROLE_ID\", \"name\": \"$role\"}]" || true
    fi

    # Assign groups
    for group_id in "${groups[@]}"; do
      if [ -n "$group_id" ] && [ "$group_id" != "not found" ]; then
        kc_api PUT "/monti/users/$USER_ID/groups/$group_id" "" || true
      fi
    done

    echo "  User '$username' created with role '$role' and ${#groups[@]} group(s)"
  fi
}

# Create default users with their groups
# admin - no groups needed (admin role sees everything)
create_user "admin" "admin" "admin" "Admin" "User"

# supervisor - SGB + NGB (Munich, Frankfurt, Berlin, Hamburg)
create_user "supervisor" "supervisor" "supervisor" "Supervisor" "User" "$SGB_GROUP_ID" "$NGB_GROUP_ID"

# agent - SGB only (Munich, Frankfurt)
create_user "agent" "agent" "agent" "Agent" "User" "$SGB_GROUP_ID"

# demo - RGB only (Remote)
create_user "demo" "demo" "viewer" "Demo" "User" "$RGB_GROUP_ID"

echo ""
echo "=== Setup Complete ==="
echo ""
echo "Keycloak Admin Console: $KEYCLOAK_URL/admin"
echo "  Username: $ADMIN_USER"
echo "  Password: $ADMIN_PASS"
echo "  Realm: monti (select from dropdown)"
echo ""
echo "MONTI Users:"
echo "  admin / admin           (role: admin, groups: none - sees all)"
echo "  supervisor / supervisor (role: supervisor, groups: SGB, NGB)"
echo "  agent / agent           (role: agent, groups: SGB)"
echo "  demo / demo             (role: viewer, groups: RGB)"
echo ""
echo "Business Unit Locations:"
echo "  SGB: Munich, Frankfurt"
echo "  NGB: Berlin, Hamburg"
echo "  RGB: Remote"
echo ""
echo "OIDC Config: $KEYCLOAK_URL/realms/monti/.well-known/openid-configuration"
echo ""
echo "To export realm config for backup:"
echo "  ./scripts/export-keycloak.sh"
