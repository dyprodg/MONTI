# Frontend

React 18 SPA with TypeScript and Vite. Connects to the backend via WebSocket for real-time agent status updates.

## Project Structure

```
Frontend/
├── src/
│   ├── components/     # UI components
│   ├── hooks/          # Custom React hooks
│   ├── services/       # Auth, WebSocket client
│   ├── types/          # TypeScript types
│   └── App.tsx
├── .env.development    # Dev environment
├── .env.example        # Template
├── vite.config.ts
├── package.json
├── Dockerfile          # Production build
└── Dockerfile.dev      # Dev with hot reload
```

## Environment Variables

| Variable | Description | Dev Default |
|----------|-------------|-------------|
| `VITE_WS_URL` | WebSocket URL | `ws://localhost:8080/ws` |
| `VITE_API_URL` | Backend API URL | `http://localhost:8080` |
| `VITE_OIDC_ISSUER` | Keycloak realm URL | `http://localhost:8180/realms/monti` |
| `VITE_OIDC_CLIENT_ID` | OIDC client ID | `monti-app` |
| `VITE_OIDC_REDIRECT_URI` | OAuth callback URI | `http://localhost:5173` |

## Local Development

```bash
# With Docker (hot reload via volume mounts)
docker compose up -d frontend

# Without Docker
cd Frontend
npm install
npm run dev     # http://localhost:5173
```

Vite is configured with polling for file watching inside Docker containers.

## Build

```bash
cd Frontend
npm run build   # Output: dist/
```

The production build is a static SPA. All routes fall back to `index.html` (handled by CloudFront/S3 or the dev server).

## Deploy to S3

The frontend is hosted as a static site on S3 behind CloudFront.

```bash
# Build
cd Frontend
npm run build

# Sync to S3
aws s3 sync dist/ s3://monti-frontend-prod/ --delete

# Invalidate CloudFront cache
aws cloudfront create-invalidation \
  --distribution-id <DISTRIBUTION_ID> \
  --paths "/*"
```

The CloudFront distribution is configured with a custom error response that returns `index.html` for 404s (SPA routing).

## Authentication

Uses `oidc-client-ts` for Authorization Code flow with PKCE.

- Login redirects to Keycloak
- Tokens stored in browser session storage
- WebSocket connection passes token as query parameter: `ws://host/ws?token=<access_token>`
- User's business unit groups determine which agents are visible

See [AUTH.md](AUTH.md) for full details.

## Key Dependencies

| Package | Purpose |
|---------|---------|
| `react` / `react-dom` | UI framework |
| `react-router-dom` | Client-side routing |
| `oidc-client-ts` | OIDC authentication |
| `vite` | Build tool & dev server |
| `vitest` | Unit testing |
| `@testing-library/react` | Component testing |

## Testing

```bash
cd Frontend
npm run test        # Run tests
npm run test:watch  # Watch mode
```

## Production Environment

For production, set environment variables at build time:

```bash
VITE_WS_URL=wss://montibackend.dennisdiepolder.com/ws
VITE_OIDC_ISSUER=https://montibackend.dennisdiepolder.com/realms/monti
VITE_OIDC_CLIENT_ID=monti-app
VITE_OIDC_REDIRECT_URI=https://monti.dennisdiepolder.com
npm run build
```

These are baked into the bundle at build time (Vite `VITE_` prefix convention).
