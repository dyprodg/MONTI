#!/bin/bash
# Export Keycloak realm configuration for backup
# Creates keycloak/realm-export.json

set -e

KEYCLOAK_URL="${KEYCLOAK_URL:-http://localhost:8180}"
ADMIN_USER="${KEYCLOAK_ADMIN:-admin}"
ADMIN_PASS="${KEYCLOAK_ADMIN_PASSWORD:-admin}"
REALM="${REALM:-monti}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
OUTPUT_DIR="$PROJECT_DIR/keycloak"
OUTPUT_FILE="$OUTPUT_DIR/realm-export.json"

echo "=== MONTI Keycloak Export ==="
echo "Keycloak URL: $KEYCLOAK_URL"
echo "Realm: $REALM"

# Ensure output directory exists
mkdir -p "$OUTPUT_DIR"

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

# Export realm via partial export endpoint
# Note: This exports realm settings, clients, roles, and groups
# Users are NOT exported by default for security (credentials)
echo "Exporting realm '$REALM'..."

# Partial export includes most configuration
curl -sf -X POST "$KEYCLOAK_URL/admin/realms/$REALM/partial-export?exportClients=true&exportGroupsAndRoles=true" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -o "$OUTPUT_FILE"

if [ -f "$OUTPUT_FILE" ] && [ -s "$OUTPUT_FILE" ]; then
  # Pretty print the JSON
  python3 -m json.tool "$OUTPUT_FILE" > "$OUTPUT_FILE.tmp" && mv "$OUTPUT_FILE.tmp" "$OUTPUT_FILE"

  echo ""
  echo "=== Export Complete ==="
  echo "Output: $OUTPUT_FILE"
  echo "Size: $(wc -c < "$OUTPUT_FILE" | tr -d ' ') bytes"
  echo ""
  echo "Note: User credentials are NOT exported for security."
  echo "After import, run setup-keycloak.sh to recreate users."
  echo ""
  echo "To import this config:"
  echo "  ./scripts/import-keycloak.sh"
else
  echo "ERROR: Export failed or file is empty"
  exit 1
fi
