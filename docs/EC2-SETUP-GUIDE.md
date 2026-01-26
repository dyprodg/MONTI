# EC2 Manual Setup Guide

## Quick Reference

| Resource | Value |
|----------|-------|
| EC2 IP | `3.69.80.81` |
| SSH Key | `~/.ssh/monti-key.pem` |
| SSH Command | `ssh -i ~/.ssh/monti-key.pem ec2-user@3.69.80.81` |

## Stop/Start EC2 (Save Money)

```bash
# Stop EC2 (pauses billing for compute, keeps EBS)
aws ec2 stop-instances --instance-ids i-01b85e089462de008

# Start EC2 (IP stays same due to Elastic IP)
aws ec2 start-instances --instance-ids i-01b85e089462de008

# Check status
aws ec2 describe-instances --instance-ids i-01b85e089462de008 --query 'Reservations[0].Instances[0].State.Name'
```

## Manual Setup Steps

### 1. SSH into EC2

```bash
ssh -i ~/.ssh/monti-key.pem ec2-user@3.69.80.81
```

### 2. Verify Docker is installed

```bash
docker --version
docker-compose --version
```

If not installed (user-data may still be running):
```bash
# Check user-data log
sudo tail -f /var/log/user-data.log

# Or install manually
sudo dnf install -y docker git
sudo systemctl start docker && sudo systemctl enable docker
sudo usermod -aG docker ec2-user
# Log out and back in for group to take effect

# Install docker-compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
```

### 3. Clone the repository

```bash
cd ~
git clone https://github.com/YOUR_USERNAME/monti.git
cd monti
```

### 4. Login to ECR

```bash
aws ecr get-login-password --region eu-central-1 | docker login --username AWS --password-stdin 283919506801.dkr.ecr.eu-central-1.amazonaws.com
```

### 5. Fetch secrets and create .env

```bash
# Run the fetch-secrets script (created by user-data)
./fetch-secrets.sh

# Or manually create .env
cat > .env << 'EOF'
ECR_REGISTRY=283919506801.dkr.ecr.eu-central-1.amazonaws.com

# Keycloak
KEYCLOAK_ADMIN=admin
KEYCLOAK_ADMIN_PASSWORD=<from SSM or terraform output>
KC_HOSTNAME=3.69.80.81

# Grafana
GF_ADMIN_USER=admin
GF_ADMIN_PASSWORD=<from SSM or terraform output>
GF_ROOT_URL=http://3.69.80.81:3000

# Backend
ALLOWED_ORIGINS=http://3.69.80.81:5173,https://d2r12k4idg6xiy.cloudfront.net
EOF
```

### 6. Build and push images (first time only)

On your local machine:
```bash
# Login to ECR
aws ecr get-login-password --region eu-central-1 | docker login --username AWS --password-stdin 283919506801.dkr.ecr.eu-central-1.amazonaws.com

# Build and push backend
docker build -t 283919506801.dkr.ecr.eu-central-1.amazonaws.com/monti-backend:latest ./Backend
docker push 283919506801.dkr.ecr.eu-central-1.amazonaws.com/monti-backend:latest

# Build and push agentsim
docker build -t 283919506801.dkr.ecr.eu-central-1.amazonaws.com/monti-agentsim:latest ./AgentSim
docker push 283919506801.dkr.ecr.eu-central-1.amazonaws.com/monti-agentsim:latest
```

### 7. Start services on EC2

```bash
# Pull images and start
docker-compose -f docker-compose.prod.yml pull
docker-compose -f docker-compose.prod.yml up -d

# Check status
docker-compose -f docker-compose.prod.yml ps

# View logs
docker-compose -f docker-compose.prod.yml logs -f
```

### 8. Verify services

| Service | URL |
|---------|-----|
| Backend API | http://3.69.80.81:8080/health |
| Keycloak | http://3.69.80.81:8180 |
| Grafana | http://3.69.80.81:3000 |
| Prometheus | http://3.69.80.81:9090 |
| Frontend (CloudFront) | https://d2r12k4idg6xiy.cloudfront.net |

## Troubleshooting

### Check logs
```bash
docker-compose -f docker-compose.prod.yml logs backend
docker-compose -f docker-compose.prod.yml logs keycloak
```

### Restart a service
```bash
docker-compose -f docker-compose.prod.yml restart backend
```

### ECR token expired
```bash
aws ecr get-login-password --region eu-central-1 | docker login --username AWS --password-stdin 283919506801.dkr.ecr.eu-central-1.amazonaws.com
```

### Fetch secrets again
```bash
# From SSM directly
aws ssm get-parameter --name "/monti/keycloak/admin-password" --with-decryption --query 'Parameter.Value' --output text --region eu-central-1
aws ssm get-parameter --name "/monti/grafana/admin-password" --with-decryption --query 'Parameter.Value' --output text --region eu-central-1
```

