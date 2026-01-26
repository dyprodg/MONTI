#!/bin/bash
# Import Keycloak realm configuration from backup
# Reads from keycloak/realm-export.json

set -e

KEYCLOAK_URL="${KEYCLOAK_URL:-http://localhost:8180}"
ADMIN_USER="${KEYCLOAK_ADMIN:-admin}"
ADMIN_PASS="${KEYCLOAK_ADMIN_PASSWORD:-admin}"
REALM="${REALM:-monti}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
INPUT_FILE="${1:-$PROJECT_DIR/keycloak/realm-export.json}"

echo "=== MONTI Keycloak Import ==="
echo "Keycloak URL: $KEYCLOAK_URL"
echo "Realm: $REALM"
echo "Input file: $INPUT_FILE"

# Check if input file exists
if [ ! -f "$INPUT_FILE" ]; then
  echo "ERROR: Input file not found: $INPUT_FILE"
  echo ""
  echo "Usage: $0 [realm-export.json]"
  echo "  Default file: keycloak/realm-export.json"
  exit 1
fi

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

# Check if realm already exists
echo "Checking if realm '$REALM' exists..."
if curl -sf "$KEYCLOAK_URL/realms/$REALM" > /dev/null 2>&1; then
  echo "WARNING: Realm '$REALM' already exists!"
  echo ""
  read -p "Do you want to DELETE and recreate it? (yes/no): " confirm
  if [ "$confirm" = "yes" ]; then
    echo "Deleting existing realm..."
    curl -sf -X DELETE "$KEYCLOAK_URL/admin/realms/$REALM" \
      -H "Authorization: Bearer $TOKEN"
    echo "Realm deleted."
  else
    echo "Aborting. Use partial-import to update existing realm."
    echo ""
    echo "Alternative: Partial import (updates existing, doesn't delete)"
    echo "  curl -X POST \"$KEYCLOAK_URL/admin/realms/$REALM/partialImport\" \\"
    echo "    -H \"Authorization: Bearer \$TOKEN\" \\"
    echo "    -H \"Content-Type: application/json\" \\"
    echo "    -d @\"$INPUT_FILE\""
    exit 0
  fi
fi

# Import realm
echo "Importing realm from $INPUT_FILE..."
RESPONSE=$(curl -sf -X POST "$KEYCLOAK_URL/admin/realms" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d @"$INPUT_FILE" 2>&1) || {
    echo "ERROR: Import failed"
    echo "$RESPONSE"
    exit 1
  }

echo ""
echo "=== Import Complete ==="
echo ""
echo "Realm '$REALM' imported successfully!"
echo ""
echo "IMPORTANT: Users were NOT imported (security)."
echo "Run setup script to recreate users:"
echo "  ./scripts/setup-keycloak.sh"
echo ""
echo "Keycloak Admin Console: $KEYCLOAK_URL/admin"
echo "  Username: $ADMIN_USER"
echo "  Password: $ADMIN_PASS"
