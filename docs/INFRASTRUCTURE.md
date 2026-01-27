# Infrastructure

MONTI runs on AWS with Terraform-managed infrastructure. The frontend is a static site on S3 + CloudFront. The backend services run on a single EC2 instance with Docker Compose and Caddy as the reverse proxy.

## Architecture

```
                  ┌─────────────────┐
                  │    Route53      │
                  │  DNS Records    │
                  └────────┬────────┘
                           │
              ┌────────────┴────────────┐
              │                         │
   monti.dennisdiepolder.com   montibackend.dennisdiepolder.com
              │                         │
    ┌─────────▼─────────┐    ┌─────────▼─────────┐
    │    CloudFront      │    │    EC2 t3.small    │
    │  + S3 (static)     │    │  Elastic IP        │
    │                    │    │  3.69.80.81         │
    │  ACM cert          │    │                    │
    │  (us-east-1)       │    │  Caddy :80/:443    │
    └────────────────────┘    │    ├─► Backend     │
                              │    ├─► Keycloak    │
                              │    AgentSim        │
                              │    Prometheus      │
                              │    Grafana         │
                              └────────────────────┘
```

## Domains

| Subdomain | Target | Purpose |
|-----------|--------|---------|
| `monti.dennisdiepolder.com` | CloudFront | Frontend SPA |
| `montibackend.dennisdiepolder.com` | EC2 Elastic IP | Backend API, Keycloak |

## Terraform

All infrastructure is defined in `Infra/`.

```
Infra/
├── main.tf           # Providers, EC2, security group, ECR
├── variables.tf      # Configuration variables
├── outputs.tf        # Outputs (IPs, URLs, ECR repos)
├── s3-frontend.tf    # S3 bucket + CloudFront distribution
└── dns-ssl.tf        # Route53 records + ACM certificate
```

### Key Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `aws_region` | `eu-central-1` | AWS region |
| `domain_name` | `dennisdiepolder.com` | Root domain |
| `frontend_subdomain` | `monti` | Frontend subdomain |
| `backend_subdomain` | `montibackend` | Backend subdomain |
| `ec2_instance_type` | `t3.small` | EC2 instance type |
| `ec2_key_name` | `monti-key` | SSH key pair name |

### Apply Changes

```bash
cd Infra
terraform init
terraform plan
terraform apply
```

### Outputs

After `terraform apply`, these outputs are available:

- `ec2_public_ip` -- EC2 elastic IP
- `frontend_url` -- CloudFront URL
- `backend_url` -- Backend domain URL
- `ecr_backend_url` -- ECR repository for backend image
- `ecr_agentsim_url` -- ECR repository for agentsim image
- `cloudfront_distribution_id` -- For cache invalidation
- `ssh_command` -- SSH command to connect to EC2

## EC2 Setup

The EC2 instance runs Amazon Linux 2023 with Docker and Docker Compose.

### SSH Access

```bash
ssh -i ~/.ssh/monti-key.pem ec2-user@3.69.80.81
```

### Security Group

| Port | Protocol | Source | Purpose |
|------|----------|--------|---------|
| 22 | TCP | Configured CIDRs | SSH |
| 80 | TCP | 0.0.0.0/0 | HTTP (Caddy redirects to HTTPS) |
| 443 | TCP | 0.0.0.0/0 | HTTPS (Caddy) |
| 3001 | TCP | 0.0.0.0/0 | Grafana |
| 9090 | TCP | 0.0.0.0/0 | Prometheus |

### Docker Compose (Production)

Production uses `docker-compose.prod.yml` which includes:
- **Caddy** as reverse proxy (ports 80/443, automatic TLS)
- Backend and AgentSim images from ECR
- Keycloak with production hostname
- Prometheus and Grafana for monitoring

```bash
# On EC2
docker compose -f docker-compose.prod.yml up -d
docker compose -f docker-compose.prod.yml logs -f
```

## Caddy Reverse Proxy

Caddy handles TLS termination and routes requests on `montibackend.dennisdiepolder.com`:

| Path Pattern | Target |
|-------------|--------|
| `/realms/*` | Keycloak |
| `/admin/*` | Keycloak |
| `/resources/*` | Keycloak |
| `/js/*` | Keycloak |
| `/*` (everything else) | Backend |

Configuration is in `Caddyfile` at the project root.

## S3 + CloudFront (Frontend)

- S3 bucket: `monti-frontend-prod`
- CloudFront distribution with custom domain and ACM certificate
- Custom error response: 404 -> `/index.html` (SPA routing)
- Origin Access Control (OAC) for S3

### Deploy Frontend

```bash
# Build
cd Frontend && npm run build

# Upload to S3
aws s3 sync dist/ s3://monti-frontend-prod/ --delete

# Invalidate CDN cache
aws cloudfront create-invalidation \
  --distribution-id $(terraform -chdir=Infra output -raw cloudfront_distribution_id) \
  --paths "/*"
```

## ECR (Container Registry)

Two ECR repositories for Docker images:

| Repository | Purpose |
|-----------|---------|
| `monti-backend` | Backend Go server |
| `monti-agentsim` | Agent simulator |

### Push Images

```bash
# Login to ECR
aws ecr get-login-password --region eu-central-1 | \
  docker login --username AWS --password-stdin <ACCOUNT_ID>.dkr.ecr.eu-central-1.amazonaws.com

# Build, tag, push (backend)
docker build -t monti-backend Backend/
docker tag monti-backend:latest <ECR_URL>/monti-backend:latest
docker push <ECR_URL>/monti-backend:latest

# On EC2: pull and restart
docker compose -f docker-compose.prod.yml pull
docker compose -f docker-compose.prod.yml up -d
```

## Cost Estimate

| Resource | Monthly Cost |
|----------|-------------|
| EC2 t3.small (minimum) | ~$15 |
| EC2 t3.medium (recommended for 2000 agents) | ~$30 |
| S3 + CloudFront | ~$1-5 |
| Route53 hosted zone | ~$0.50 |
| ECR | ~$1 |
| **Total** | **~$17-37** |

## Resource Usage (2000 agents)

| Container | CPU | Memory |
|-----------|-----|--------|
| Backend | ~110% (1.1 cores) | ~260 MB |
| AgentSim | ~11% | ~192 MB |
| Keycloak | ~1% | ~382 MB |
| Grafana | ~1% | ~337 MB |
| Prometheus | ~0% | ~127 MB |
| **Total** | **~1.2 cores** | **~1.3 GB** |

t3.small (2 vCPU, 2 GB RAM) is the minimum. t3.medium (2 vCPU, 4 GB RAM) is recommended for headroom.
