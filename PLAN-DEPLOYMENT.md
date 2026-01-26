# MONTI Deployment Plan

## Architecture Overview

Split architecture with two CI/CD pipelines:

```
┌─────────────────────────────────────────────────────────────┐
│                         Users                                │
└─────────────────────────────────────────────────────────────┘
                    │                       │
                    ▼                       ▼
         ┌──────────────────┐    ┌──────────────────┐
         │   CloudFront     │    │   CloudFront     │
         │   (CDN/HTTPS)    │    │   or ALB         │
         └──────────────────┘    └──────────────────┘
                    │                       │
                    ▼                       ▼
         ┌──────────────────┐    ┌──────────────────┐
         │       S3         │    │       EC2        │
         │   (Frontend)     │    │    (Backend)     │
         │   React build    │    │  Docker services │
         └──────────────────┘    └──────────────────┘
                                           │
                                    ┌──────┴──────┐
                                    │  Keycloak   │
                                    │  Grafana    │
                                    │  Prometheus │
                                    └─────────────┘
```

## Why This Architecture

| Aspect | All on EC2 | Frontend S3 + Backend EC2 |
|--------|-----------|---------------------------|
| **Security** | One breach = everything | Frontend is just static files, no secrets |
| **Scalability** | Server handles everything | S3/CloudFront handles millions of requests |
| **Cost** | Bigger EC2 needed | S3 is cents/month, smaller EC2 |
| **Deployment** | Restart affects both | Deploy frontend without touching backend |
| **Availability** | EC2 down = app down | Frontend still loads, shows "API unavailable" |
| **Caching** | You manage nginx | CloudFront CDN built-in |

## Component Mapping

| Component | Where | Pipeline Trigger |
|-----------|-------|------------------|
| React Frontend | S3 + CloudFront | Changes in `Frontend/**` |
| Go Backend | EC2 Docker | Changes in `Backend/**` |
| Keycloak | EC2 Docker | Manual (rarely changes) |
| Monitoring | EC2 Docker | Manual |

---

## Implementation Steps

### Step 1: AWS Infrastructure Setup

- [ ] Create S3 bucket for frontend (`monti-frontend`)
- [ ] Set up CloudFront distribution pointing to S3
- [ ] Launch EC2 instance (t3.small recommended)
- [ ] Allocate Elastic IP for EC2
- [ ] Configure Security Groups (ports: 80, 443, 8180, 3000)
- [ ] Set up domain DNS (e.g., monti.app, api.monti.app, auth.monti.app)
- [ ] Configure SSL certificates (ACM for CloudFront, Let's Encrypt for EC2)

### Step 2: EC2 Initial Setup

```bash
# SSH into EC2
ssh -i your-key.pem ec2-user@your-ec2-ip

# Install Docker
sudo yum install docker -y
sudo systemctl start docker
sudo systemctl enable docker
sudo usermod -aG docker ec2-user

# Install docker-compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Clone repo
git clone https://github.com/you/monti.git
cd monti

# Create .env with production values
nano .env

# Start services
docker-compose up -d
```

### Step 3: GitHub Secrets Configuration

Add these secrets to GitHub repository:

| Secret | Description |
|--------|-------------|
| `AWS_ACCESS_KEY_ID` | AWS IAM access key |
| `AWS_SECRET_ACCESS_KEY` | AWS IAM secret |
| `CF_DISTRIBUTION_ID` | CloudFront distribution ID |
| `DOCKERHUB_USERNAME` | DockerHub username |
| `DOCKERHUB_TOKEN` | DockerHub access token |
| `EC2_HOST` | EC2 public IP or domain |
| `EC2_SSH_KEY` | Private SSH key for EC2 |

### Step 4: Create CI/CD Pipelines

---

## Pipeline 1: Frontend (S3)

Create `.github/workflows/frontend.yml`:

```yaml
name: Deploy Frontend

on:
  push:
    branches: [main]
    paths: ['Frontend/**']

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Node
        uses: actions/setup-node@v4
        with:
          node-version: '20'

      - name: Build
        run: |
          cd Frontend
          npm ci
          npm run build
        env:
          VITE_API_URL: https://api.monti.app
          VITE_OIDC_ISSUER: https://auth.monti.app/realms/monti

      - name: Configure AWS
        uses: aws-actions/configure-aws-credentials@v4
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: eu-central-1

      - name: Deploy to S3
        run: aws s3 sync Frontend/dist s3://monti-frontend --delete

      - name: Invalidate CloudFront
        run: aws cloudfront create-invalidation --distribution-id ${{ secrets.CF_DISTRIBUTION_ID }} --paths "/*"
```

---

## Pipeline 2: Backend (EC2)

Create `.github/workflows/backend.yml`:

```yaml
name: Deploy Backend

on:
  push:
    branches: [main]
    paths: ['Backend/**', 'docker-compose.yml']

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Login to DockerHub
        uses: docker/login-action@v3
        with:
          username: ${{ secrets.DOCKERHUB_USERNAME }}
          password: ${{ secrets.DOCKERHUB_TOKEN }}

      - name: Build and push
        run: |
          docker build -t yourusername/monti-backend:${{ github.sha }} -t yourusername/monti-backend:latest ./Backend
          docker push yourusername/monti-backend:latest
          docker push yourusername/monti-backend:${{ github.sha }}

      - name: Deploy to EC2
        uses: appleboy/ssh-action@v1
        with:
          host: ${{ secrets.EC2_HOST }}
          username: ec2-user
          key: ${{ secrets.EC2_SSH_KEY }}
          script: |
            cd /home/ec2-user/monti
            docker-compose pull backend
            docker-compose up -d backend
```

---

## Cost Estimate (EU Region)

| Service | Monthly Cost |
|---------|--------------|
| S3 + CloudFront (frontend) | ~$1-5 |
| EC2 t3.small (backend) | ~$15 |
| Elastic IP | Free (when attached) |
| **Total** | **~$20/month** |

---

## Production Environment Variables

### Frontend (build-time)
```
VITE_API_URL=https://api.monti.app
VITE_OIDC_ISSUER=https://auth.monti.app/realms/monti
VITE_OIDC_CLIENT_ID=monti-app
```

### Backend (runtime)
```
ENV=production
OIDC_ISSUER=https://auth.monti.app/realms/monti
VERIFY_JWT_SIGNATURE=true
```

### Keycloak
```
KEYCLOAK_ADMIN=admin
KEYCLOAK_ADMIN_PASSWORD=<strong-password>
KC_HOSTNAME=auth.monti.app
KC_PROXY=edge
```

---

## Production Checklist

- [ ] AWS infrastructure created
- [ ] Domain DNS configured
- [ ] SSL/TLS certificates set up
- [ ] GitHub secrets configured
- [ ] CI/CD pipelines created
- [ ] Production environment variables set
- [ ] Keycloak admin password changed
- [ ] Keycloak realm exported as backup
- [ ] Test deployment with both pipelines
- [ ] Verify frontend can reach backend API
- [ ] Verify authentication flow works
- [ ] Set up monitoring alerts (optional)
- [ ] Configure backup strategy (optional)

---

## Manual Deployment (Fallback)

If CI/CD fails, deploy manually:

### Frontend
```bash
cd Frontend
npm ci && npm run build
aws s3 sync dist s3://monti-frontend --delete
aws cloudfront create-invalidation --distribution-id XXXXX --paths "/*"
```

### Backend
```bash
docker build -t yourusername/monti-backend:latest ./Backend
docker push yourusername/monti-backend:latest
ssh ec2-user@your-ec2-ip "cd monti && docker-compose pull backend && docker-compose up -d backend"
```
