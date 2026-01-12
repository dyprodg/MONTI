# Keycloak Setup Guide for MONTI

This guide walks you through setting up Keycloak for local authentication in MONTI.

---

## Quick Start

### 1. Start All Services

```bash
docker compose up -d
```

This starts:
- **Keycloak** (http://localhost:8180) - Identity provider
- **Backend** (http://localhost:8080) - API server
- **Frontend** (http://localhost:5173) - Web app

### 2. Wait for Services to Start

```bash
# Watch the logs
docker compose logs -f keycloak

# Wait for this message:
# "Running the server in development mode. DO NOT use this configuration in production."
```

### 3. Access Keycloak Admin Console

Open: http://localhost:8180

- Username: `admin`
- Password: `admin`

---

## Keycloak Configuration

### Step 1: Create Realm

A realm is an isolated space for managing users and applications.

1. Click the dropdown in the top-left (currently shows "master")
2. Click **"Create Realm"**
3. Enter:
   - **Realm name**: `monti`
4. Click **"Create"**

### Step 2: Create Client (MONTI Application)

A client represents your application that will use Keycloak for authentication.

1. In the left sidebar, click **"Clients"**
2. Click **"Create client"**
3. **General Settings**:
   - Client type: `OpenID Connect`
   - Client ID: `monti-app`
   - Click **"Next"**

4. **Capability config**:
   - Client authentication: `OFF` (public client)
   - Authorization: `OFF`
   - Authentication flow:
     - ✅ Standard flow
     - ✅ Direct access grants
   - Click **"Next"**

5. **Login settings**:
   - Root URL: `http://localhost:5173`
   - Valid redirect URIs:
     - `http://localhost:5173/*`
     - `http://localhost:5173/callback`
   - Valid post logout redirect URIs:
     - `http://localhost:5173/*`
   - Web origins:
     - `http://localhost:5173`
   - Click **"Save"**

### Step 3: Create Roles

Roles define what users can do in your application.

1. In the left sidebar, click **"Realm roles"**
2. Click **"Create role"**
3. Create these three roles:

**Admin Role:**
- Role name: `admin`
- Description: `Full administrative access`
- Click **"Save"**

**Manager Role:**
- Role name: `manager`
- Description: `Manage agents and teams`
- Click **"Save"**

**Viewer Role:**
- Role name: `viewer`
- Description: `Read-only access`
- Click **"Save"**

### Step 4: Create Users

1. In the left sidebar, click **"Users"**
2. Click **"Add user"**

**Admin User:**
- Username: `admin@monti.local`
- Email: `admin@monti.local`
- Email verified: `ON`
- First name: `Admin`
- Last name: `User`
- Click **"Create"**

3. Go to **"Credentials"** tab
   - Click **"Set password"**
   - Password: `admin123`
   - Temporary: `OFF`
   - Click **"Save"**
   - Confirm: **"Save password"**

4. Go to **"Role mapping"** tab
   - Click **"Assign role"**
   - Select `admin`
   - Click **"Assign"**

**Manager User:**
- Repeat the above steps with:
  - Username: `manager@monti.local`
  - Email: `manager@monti.local`
  - Password: `manager123`
  - Assign role: `manager`

**Viewer User:**
- Repeat the above steps with:
  - Username: `viewer@monti.local`
  - Email: `viewer@monti.local`
  - Password: `viewer123`
  - Assign role: `viewer`

### Step 5: Configure Client Scopes (Optional but Recommended)

This ensures roles are included in the token.

1. In the left sidebar, click **"Client scopes"**
2. Click **"roles"**
3. Go to **"Mappers"** tab
4. Click **"realm roles"**
5. Ensure these settings:
   - Token Claim Name: `realm_access.roles`
   - Add to ID token: `ON`
   - Add to access token: `ON`
   - Add to userinfo: `ON`

---

## Testing the Setup

### Test Authentication Flow

1. **Open the app**: http://localhost:5173
2. **You should see**: Login page with "Sign in with SSO" button
3. **Click**: "Sign in with SSO"
4. **Redirected to**: Keycloak login (http://localhost:8180/realms/monti/protocol/openid-connect/auth...)
5. **Login with**:
   - Username: `admin@monti.local`
   - Password: `admin123`
6. **Redirected back**: http://localhost:5173/callback
7. **Then to dashboard**: http://localhost:5173/
8. **You should see**: Dashboard with your name and role

### Test Different Roles

Logout and login with different users to see different permissions:

```bash
# Admin - Full access
admin@monti.local / admin123

# Manager - Manage agents/teams
manager@monti.local / manager123

# Viewer - Read-only
viewer@monti.local / viewer123
```

### Verify Token in Browser

1. Open browser DevTools (F12)
2. Go to **"Application"** → **"Local Storage"** → `http://localhost:5173`
3. Look for keys starting with `oidc.`
4. You should see the access token and user info

### Test WebSocket with Auth

1. Open browser DevTools (F12)
2. Go to **"Network"** tab
3. Filter by **"WS"** (WebSocket)
4. You should see connection to `ws://localhost:8080/ws?token=...`
5. Token is automatically appended to WebSocket URL

### Check Backend Logs

```bash
docker compose logs -f backend

# You should see:
# [Auth] User authenticated: admin@monti.local (admin)
```

---

## Troubleshooting

### Issue: "Invalid redirect URI"

**Symptom:** After login, Keycloak shows "Invalid redirect URI"

**Solution:**
1. Go to Keycloak Admin → Clients → monti-app
2. Check "Valid redirect URIs" includes:
   - `http://localhost:5173/*`
   - `http://localhost:5173/callback`
3. Save and try again

### Issue: "Role not found in token"

**Symptom:** User logged in but role shows as "viewer" even though you assigned "admin"

**Solution:**
1. Keycloak Admin → Client Scopes → roles → Mappers
2. Ensure "realm roles" mapper exists
3. Check "Add to access token" is enabled
4. Logout and login again (token needs refresh)

### Issue: "CORS error"

**Symptom:** Browser console shows CORS error

**Solution:**
1. Keycloak Admin → Clients → monti-app
2. Check "Web origins" includes: `http://localhost:5173`
3. Save and refresh browser

### Issue: "Token expired"

**Symptom:** Dashboard loads then immediately redirects to login

**Solution:**
- This is normal - tokens expire after ~5 minutes by default
- The app should auto-refresh tokens silently
- If not working, check browser console for errors

### Issue: "Cannot connect to Keycloak"

**Symptom:** Login button does nothing or shows network error

**Solution:**
```bash
# Check if Keycloak is running
docker compose ps

# If not running, start it
docker compose up -d keycloak

# Check logs
docker compose logs keycloak
```

### Issue: Backend shows "Unauthorized"

**Symptom:** WebSocket connection fails with 401

**Solution:**
1. Check backend environment variables:
   ```bash
   docker compose exec backend env | grep OIDC
   ```
2. Should show:
   - `OIDC_ISSUER=http://keycloak:8180/realms/monti`
   - `OIDC_CLIENT_ID=monti-app`
3. If wrong, update docker-compose.yml and restart

---

## Advanced Configuration

### Change Token Expiration

1. Keycloak Admin → Realm settings → Tokens
2. Adjust:
   - Access Token Lifespan: `5 minutes` (default)
   - SSO Session Idle: `30 minutes`
   - SSO Session Max: `10 hours`
3. Click **"Save"**

### Add More Users

1. Keycloak Admin → Users → Add user
2. Fill in details
3. Go to Credentials → Set password
4. Go to Role mapping → Assign role
5. Test login with new user

### Create Groups (Optional)

Groups allow you to manage multiple users with the same roles.

1. Keycloak Admin → Groups
2. Create group: `MONTI Admins`
3. Add members
4. Assign roles to the group
5. All members inherit the roles

---

## Environment Variables

### Backend (.env or docker-compose.yml)

```env
ENV=development
OIDC_ISSUER=http://keycloak:8180/realms/monti
OIDC_CLIENT_ID=monti-app
OIDC_AUDIENCE=monti-app
SKIP_AUTH=false  # Set to true to bypass auth for testing
```

### Frontend (.env)

```env
VITE_WS_URL=ws://localhost:8080/ws
VITE_API_URL=http://localhost:8080/api
VITE_OIDC_ISSUER=http://localhost:8180/realms/monti
VITE_OIDC_CLIENT_ID=monti-app
VITE_OIDC_REDIRECT_URI=http://localhost:5173/callback
```

---

## Quick Commands

```bash
# Start everything
docker compose up -d

# View logs
docker compose logs -f

# View Keycloak logs only
docker compose logs -f keycloak

# View backend logs only
docker compose logs -f backend

# Restart backend after code changes
docker compose restart backend

# Stop everything
docker compose down

# Stop and remove volumes (fresh start)
docker compose down -v

# Access Keycloak admin
open http://localhost:8180

# Access MONTI app
open http://localhost:5173
```

---

## Next Steps

1. ✅ Keycloak is running
2. ✅ Realm, client, and users are created
3. ✅ App can authenticate users
4. ✅ Tokens are validated
5. ✅ WebSocket uses auth tokens

### What's Next?

- **Add more users** for your team
- **Test different roles** to ensure permissions work
- **Implement role-based features** in the frontend
- **Add API endpoints** that check user permissions
- **Prepare for production** by setting up AWS IAM Identity Center

---

## Production Migration

When ready for production, you'll:

1. **Set up AWS IAM Identity Center**
2. **Update environment variables**:
   ```env
   OIDC_ISSUER=https://your-portal.awsapps.com/
   OIDC_CLIENT_ID=your-aws-client-id
   ```
3. **No code changes needed** - just configuration!

---

## Support

If you encounter issues:

1. Check the troubleshooting section above
2. View logs: `docker compose logs -f`
3. Check browser console (F12) for errors
4. Verify Keycloak admin settings

For production AWS setup, refer to:
- `docs/AUTH_PRACTICAL_GUIDE.md` - Full authentication guide
- `docs/AUTH_SETUP_GUIDE.md` - Production AWS setup
