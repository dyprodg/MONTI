# Frontend

Web application for the MONTI call center live monitoring dashboard.

## Purpose

User-facing web application that:
- Displays real-time agent status across teams and locations
- Connects to backend via WebSocket for live updates
- Provides filtering and sorting capabilities
- Implements responsive, performant UI for 2000+ agents
- Handles authentication via AWS IAM Identity Center

## Rules

### Framework Choice
- Use modern framework: React, Next.js, or Astro
- TypeScript required for type safety
- Use established state management (Context API, Zustand, or Redux)

### Code Organization
- Component-based architecture
- Separate components, hooks, services, and utilities
- Keep components small and focused (single responsibility)
- Use folder structure: `components/`, `pages/`, `hooks/`, `services/`, `utils/`, `types/`

### UI/UX
- Mobile-first responsive design
- Accessible (WCAG 2.1 AA compliance)
- Loading states for all async operations
- Error boundaries for graceful error handling
- Consistent design system/component library

### WebSocket Integration
- Implement reconnection logic with exponential backoff
- Handle connection state (connecting, connected, disconnected)
- Buffer updates during disconnection
- Display connection status to user
- Efficient render updates (avoid unnecessary re-renders)

### Performance
- Virtual scrolling/windowing for large agent lists
- Debounce/throttle user inputs
- Code splitting and lazy loading
- Optimize bundle size (< 500KB initial load)
- Lighthouse score > 90
- First Contentful Paint < 1.5s

### Data Display
- Show grouped/aggregated data, not 2000 individual rows
- Implement efficient filtering (client-side for cached data)
- Sortable columns
- Search functionality
- Export capabilities (optional)

### Authentication
- OIDC/OAuth integration with AWS IAM Identity Center
- Handle token refresh
- Protected routes (redirect to login if unauthenticated)
- Store tokens securely (httpOnly cookies or secure storage)
- Logout functionality

### Security
- No sensitive data in client-side code
- XSS prevention (sanitize user inputs)
- CSRF protection
- Content Security Policy headers
- Dependency vulnerability scanning

### State Management
- Centralized state for agent data
- Separate UI state from server state
- Immutable state updates
- Consider using query libraries (React Query, SWR) for server state

### Testing
- Unit tests for utilities and hooks
- Component tests for UI components
- Integration tests for key user flows
- E2E tests for critical paths (Playwright/Cypress)
- Minimum 70% coverage

### Styling
- Use CSS-in-JS, CSS modules, or Tailwind CSS
- No inline styles except for dynamic values
- Follow BEM or similar naming convention
- Dark mode support (optional)
- Consistent spacing and typography

### Configuration
- Environment variables for API endpoints
- Build-time vs runtime configuration clearly separated
- Different configs for dev, staging, production

### Build & Optimization
- Production builds must be optimized
- Tree shaking enabled
- Minification and compression
- Source maps for debugging (not in production)
- Static asset caching strategy

## Dependencies

- Must not depend on Backend or AgentSim code directly
- Communicate with Backend only via API/WebSocket
- Must not depend on Infra code

## CI/CD

This service will only be built and tested when:
- Files in `Frontend/` directory are changed
- Root configuration files affecting all services are changed
- Explicitly triggered via workflow dispatch

### Build Steps
1. Install dependencies (`npm ci` or `yarn install --frozen-lockfile`)
2. Run linter (ESLint)
3. Run type checking (TypeScript)
4. Run unit tests with coverage
5. Build production bundle
6. Run bundle size analysis
7. Run E2E tests (optional in CI)
8. Security audit (npm audit or yarn audit)

### Deployment
- Deploy to S3 + CloudFront, Vercel, or similar
- Invalidate CDN cache after deployment
- Use environment-specific configurations
- Smoke tests post-deployment
