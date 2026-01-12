# CICD

CI/CD configuration and workflows for the MONTI monorepo.

## Purpose

Centralized CI/CD configuration using GitHub Actions for:
- Building and testing each service independently
- Deploying services only when their code changes
- Managing infrastructure changes via Terraform
- Running security scans and quality checks
- Coordinating deployments across the monorepo

## Rules

### Monorepo Strategy
- Each service (Frontend, Backend, AgentSim, Infra) has its own workflow
- Workflows only trigger when relevant files change
- Use path filters to detect changes
- Support manual workflow dispatch for all workflows

### Path-Based Triggers
Each workflow should only run when files in its respective directory change:
- `Frontend/**` → Frontend workflow
- `Backend/**` → Backend workflow
- `AgentSim/**` → AgentSim workflow
- `Infra/**` → Infrastructure workflow
- Root config files may trigger all workflows

### Workflow Organization
```
CICD/
├── workflows/
│   ├── frontend.yml
│   ├── backend.yml
│   ├── agent-sim.yml
│   ├── infra.yml
│   └── security.yml
├── scripts/
│   ├── check-changes.sh
│   └── deploy.sh
└── README.md
```

### Common Workflow Structure
Each workflow should follow this pattern:
1. **Trigger**: On push/PR to main + path filters + workflow_dispatch
2. **Checkout**: Checkout code
3. **Setup**: Install dependencies and tools
4. **Lint**: Run linters and formatters
5. **Test**: Run unit and integration tests
6. **Build**: Build artifacts
7. **Security**: Run security scans
8. **Deploy**: Deploy if on main branch (optional)

### Frontend Workflow
```yaml
on:
  push:
    branches: [main]
    paths:
      - 'Frontend/**'
      - '.github/workflows/frontend.yml'
  pull_request:
    branches: [main]
    paths:
      - 'Frontend/**'
  workflow_dispatch:
```

Steps:
1. Checkout code
2. Setup Node.js
3. Install dependencies
4. Run ESLint
5. Run TypeScript check
6. Run tests with coverage
7. Build production bundle
8. Analyze bundle size
9. Run security audit
10. Deploy to S3/CloudFront (if main branch)

### Backend Workflow
```yaml
on:
  push:
    branches: [main]
    paths:
      - 'Backend/**'
      - '.github/workflows/backend.yml'
  pull_request:
    branches: [main]
    paths:
      - 'Backend/**'
  workflow_dispatch:
```

Steps:
1. Checkout code
2. Setup Go
3. Run `go fmt` check
4. Run `go vet`
5. Run golangci-lint
6. Run tests with coverage
7. Run gosec security scan
8. Build binary
9. Build Docker image (optional)
10. Deploy to AWS (if main branch)

### AgentSim Workflow
```yaml
on:
  push:
    branches: [main]
    paths:
      - 'AgentSim/**'
      - '.github/workflows/agent-sim.yml'
  pull_request:
    branches: [main]
    paths:
      - 'AgentSim/**'
  workflow_dispatch:
```

Steps: Similar to Backend workflow

### Infrastructure Workflow
```yaml
on:
  push:
    branches: [main]
    paths:
      - 'Infra/**'
      - '.github/workflows/infra.yml'
  pull_request:
    branches: [main]
    paths:
      - 'Infra/**'
  workflow_dispatch:
```

Steps:
1. Checkout code
2. Setup Terraform
3. Run `terraform fmt -check`
4. Run `terraform validate`
5. Run tfsec security scan
6. Run `terraform plan`
7. Post plan as PR comment
8. Apply changes (if main branch + approved)

### Secrets Management
- Store secrets in GitHub Secrets
- Never log secrets
- Use environment-specific secrets
- Rotate secrets regularly
- Document required secrets in README

Required secrets:
- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- `AWS_REGION`
- Other service-specific secrets

### Caching Strategy
- Cache dependencies to speed up builds
- Node modules cache for Frontend
- Go modules cache for Backend
- Terraform providers cache
- Docker layer cache

### Environment Configuration
Support multiple environments:
- **Development**: Automatic deployment on merge to `develop` branch
- **Staging**: Automatic deployment on merge to `main` branch
- **Production**: Manual approval required

### Notifications
- Notify on workflow failures
- Post deployment status
- Security scan alerts
- Consider Slack/Discord/Email integration

### Quality Gates
Workflows should fail if:
- Tests fail
- Coverage below threshold (70% for Frontend, 80% for Backend)
- Linting errors
- Security vulnerabilities found (high/critical)
- Build fails

### Deployment Strategy
- Use rolling deployments for zero downtime
- Implement health checks before marking deployment successful
- Rollback capability on failure
- Deploy to staging first, then production

### Monitoring & Logging
- Track workflow execution times
- Monitor success/failure rates
- Log important deployment events
- Set up alerts for repeated failures

### Documentation
- Document all workflows in this README
- Comment complex workflow steps
- Keep architecture diagrams updated
- Document manual intervention procedures

### Best Practices
- Keep workflows DRY (use reusable workflows/composite actions)
- Pin action versions for stability
- Use conditional steps to optimize runtime
- Implement proper error handling
- Clean up artifacts after deployment

### Security Scanning
Include these scans:
- **Frontend**: npm audit, Snyk, OWASP dependency check
- **Backend**: gosec, govulncheck
- **Infra**: tfsec, checkov
- **Containers**: Trivy, Snyk

### Performance Optimization
- Run independent jobs in parallel
- Use matrix strategy for multi-version testing
- Minimize workflow runtime (target < 10 minutes)
- Only run necessary steps based on changes

## CI/CD

Changes to CICD configurations will trigger validation:
- Validate YAML syntax
- Test workflow changes in a safe environment
- Review changes carefully as they affect all services

## Getting Started

1. Copy workflow files to `.github/workflows/` directory:
   ```bash
   mkdir -p .github/workflows
   cp CICD/workflows/* .github/workflows/
   ```

2. Configure GitHub Secrets in repository settings

3. Test workflows with workflow_dispatch

4. Monitor first runs and adjust as needed

## Workflow Examples

See `workflows/` directory for complete workflow definitions.
