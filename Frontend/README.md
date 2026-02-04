# Frontend

React-based web dashboard for real-time call center agent monitoring. Connects to the backend via WebSocket for live status updates.

## Key Dependencies

- React 18, TypeScript, Vite
- oidc-client-ts (Keycloak authentication)

## Local Development

```bash
cd Frontend
npm install
npm run dev
```

Open http://localhost:5173 in your browser.

## Build

```bash
npm run build    # Production build to dist/
npm run preview  # Preview production build locally
```
