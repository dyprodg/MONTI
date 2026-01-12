# Auth Quick Start - TL;DR

Get authentication working in 5 minutes.

---

## 1. Start Everything

```bash
docker compose up -d
```

Wait ~30 seconds for Keycloak to start.

---

## 2. Setup Keycloak

### Open Admin Console
http://localhost:8180

Login: `admin` / `admin`

### Create Realm
1. Click dropdown (top-left) → "Create Realm"
2. Name: `monti`
3. Click "Create"

### Create Client
1. Clients → Create client
2. Client ID: `monti-app`
3. Next → Enable "Standard flow" → Next
4. Valid redirect URIs: `http://localhost:5173/*`
5. Web origins: `http://localhost:5173`
6. Save

### Create Role
1. Realm roles → Create role
2. Name: `admin`
3. Save

### Create User
1. Users → Add user
2. Username: `admin@monti.local`
3. Email: `admin@monti.local`
4. Email verified: ON
5. Create
6. Credentials tab → Set password → `admin123` → Temporary: OFF → Save
7. Role mapping tab → Assign role → Select `admin` → Assign

---

## 3. Test It

### Open App
http://localhost:5173

### Login
- Username: `admin@monti.local`
- Password: `admin123`

### Success!
You should see the dashboard with your name in the top-right.

---

## Troubleshooting

**Problem**: Invalid redirect URI error
- **Fix**: Check client "Valid redirect URIs" includes `http://localhost:5173/*`

**Problem**: Can't connect to Keycloak
- **Fix**: `docker compose logs keycloak` - wait for "Running the server" message

**Problem**: Unauthorized error
- **Fix**: Check backend logs: `docker compose logs backend`

---

## Full Documentation

- **Keycloak Setup**: [KEYCLOAK_SETUP.md](./KEYCLOAK_SETUP.md)
- **Auth Architecture**: [AUTH_PRACTICAL_GUIDE.md](./AUTH_PRACTICAL_GUIDE.md)
- **Production Setup**: [AUTH_SETUP_GUIDE.md](./AUTH_SETUP_GUIDE.md)

---

## Commands

```bash
# Start
docker compose up -d

# Logs
docker compose logs -f

# Stop
docker compose down

# Fresh start
docker compose down -v && docker compose up -d
```

---

That's it! You now have:
- ✅ Login page
- ✅ SSO authentication (Keycloak)
- ✅ Protected dashboard
- ✅ User info & logout
- ✅ Auth tokens in WebSocket
- ✅ Same code works for AWS in production
