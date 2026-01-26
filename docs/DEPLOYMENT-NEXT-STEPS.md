# Deployment - Next Steps

## Current State

### Done
- [x] S3 bucket for Terraform state
- [x] ECR repositories (backend, agentsim)
- [x] S3 + CloudFront for frontend
- [x] EC2 instance with IAM role
- [x] Security groups
- [x] SSM Parameter Store for secrets (Keycloak, Grafana passwords)
- [x] IAM user for CI/CD with minimal permissions
- [x] Production docker-compose
- [x] GitHub Actions workflows (basic)

### Infrastructure Outputs
| Resource | Value |
|----------|-------|
| EC2 IP | 3.69.80.81 |
| EC2 Instance ID | i-01b85e089462de008 |
| CloudFront URL | https://d2r12k4idg6xiy.cloudfront.net |
| CloudFront ID | E2IWUI1W1OUXHT |
| ECR Backend | 283919506801.dkr.ecr.eu-central-1.amazonaws.com/monti-backend |
| ECR AgentSim | 283919506801.dkr.ecr.eu-central-1.amazonaws.com/monti-agentsim |
| CI/CD Access Key | AKIAUEGXL3VYZ3PPXAXB |

---

## Phase 1: CI/CD Improvements (Priority: High)

### 1.1 Use GitHub OIDC instead of IAM Access Keys
**Why:** No long-lived credentials stored in GitHub. More secure.

```hcl
# Infra/iam-github-oidc.tf
resource "aws_iam_openid_connect_provider" "github" {
  url             = "https://token.actions.githubusercontent.com"
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = ["..."]
}

resource "aws_iam_role" "github_actions" {
  name = "monti-github-actions-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = { Federated = aws_iam_openid_connect_provider.github.arn }
      Action = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringEquals = {
          "token.actions.githubusercontent.com:aud" = "sts.amazonaws.com"
        }
        StringLike = {
          "token.actions.githubusercontent.com:sub" = "repo:YOUR_ORG/monti:*"
        }
      }
    }]
  })
}
```

Then in GitHub Actions:
```yaml
- uses: aws-actions/configure-aws-credentials@v4
  with:
    role-to-assume: arn:aws:iam::283919506801:role/monti-github-actions-role
    aws-region: eu-central-1
```

### 1.2 Use GitHub Environments
**Why:** Separate secrets per environment, require approvals for prod.

- Create `production` environment in GitHub repo settings
- Move secrets to environment-specific secrets
- Add required reviewers for production deployments

### 1.3 Add Deployment Caching
**Why:** Faster builds.

```yaml
- uses: actions/cache@v4
  with:
    path: |
      ~/.docker
      /tmp/.buildx-cache
    key: ${{ runner.os }}-docker-${{ hashFiles('**/Dockerfile') }}
```

### 1.4 Add Build Matrix for Multi-arch (Optional)
**Why:** Support ARM64 if needed later.

---

## Phase 2: Security Hardening (Priority: High)

### 2.1 HTTPS for EC2 Services
Options:
- **Option A:** Put ALB in front of EC2 with ACM certificate
- **Option B:** Use Caddy/Traefik as reverse proxy with Let's Encrypt
- **Option C:** Use nginx with certbot

### 2.2 Restrict Security Group SSH
```hcl
# Change from 0.0.0.0/0 to your IP
variable "allowed_ssh_cidrs" {
  default = ["YOUR_IP/32"]
}
```

### 2.3 Enable VPC Flow Logs
For audit trail of network traffic.

### 2.4 Keycloak Production Mode
Currently using `start-dev`. Change to:
```yaml
command: start --optimized
```
Requires building optimized image.

---

## Phase 3: Domain & SSL (Priority: Medium)

### 3.1 Register/Configure Domain
- Point `monti.app` (or your domain) to CloudFront
- Point `api.monti.app` to EC2 (via ALB or directly)
- Point `auth.monti.app` to Keycloak

### 3.2 ACM Certificates
```hcl
resource "aws_acm_certificate" "main" {
  domain_name       = "monti.app"
  validation_method = "DNS"

  subject_alternative_names = [
    "*.monti.app"
  ]
}
```

### 3.3 Update CloudFront with Custom Domain
Uncomment aliases in `s3-frontend.tf`.

---

## Phase 4: Monitoring & Alerting (Priority: Medium)

### 4.1 CloudWatch Alarms
- EC2 CPU > 80%
- EC2 disk space < 20%
- Backend health check failures

### 4.2 SNS Notifications
Email/Slack alerts for alarms.

### 4.3 Log Aggregation
- CloudWatch Logs agent on EC2
- Or ship to external service (Datadog, etc.)

---

## Phase 5: Backup & Recovery (Priority: Medium)

### 5.1 EBS Snapshots
Automated daily snapshots of EC2 root volume.

### 5.2 Keycloak Realm Export
Regular export of Keycloak realm config.

### 5.3 Grafana Dashboard Backup
Export dashboards to git.

---

## Phase 6: Cost Optimization (Priority: Low)

### 6.1 Reserved Instances or Savings Plans
If running 24/7, consider reserved capacity.

### 6.2 Auto-scaling (Future)
If traffic grows, move to ECS/EKS with auto-scaling.

### 6.3 S3 Lifecycle Rules
Clean up old CloudFront logs, etc.

---

## Quick Wins (Do First)

1. [ ] Add GitHub OIDC (remove access keys)
2. [ ] Restrict SSH to your IP
3. [ ] Set up HTTPS (even self-signed for testing)
4. [ ] Add basic CloudWatch alarm for EC2

---

## GitHub Secrets Needed (Current)

```
AWS_ACCESS_KEY_ID=AKIAUEGXL3VYZ3PPXAXB
AWS_SECRET_ACCESS_KEY=<from terraform output>
CF_DISTRIBUTION_ID=E2IWUI1W1OUXHT
EC2_HOST=3.69.80.81
EC2_SSH_KEY=<contents of ~/.ssh/monti-key.pem>
VITE_API_URL=http://3.69.80.81:8080
VITE_WS_URL=ws://3.69.80.81:8080/ws
VITE_OIDC_ISSUER=http://3.69.80.81:8180/realms/monti
VITE_OIDC_CLIENT_ID=monti-app
```

After GitHub OIDC setup, remove `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`.
