# EC2 Instance for Backend services

# IAM role for EC2 to pull from ECR
resource "aws_iam_role" "ec2_role" {
  name = "${var.project_name}-ec2-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
      }
    ]
  })
}

# Policy for ECR access
resource "aws_iam_role_policy" "ecr_policy" {
  name = "${var.project_name}-ecr-policy"
  role = aws_iam_role.ec2_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ecr:GetAuthorizationToken",
          "ecr:BatchCheckLayerAvailability",
          "ecr:GetDownloadUrlForLayer",
          "ecr:BatchGetImage"
        ]
        Resource = "*"
      }
    ]
  })
}

# Policy for SSM Parameter Store access
resource "aws_iam_role_policy" "ssm_policy" {
  name = "${var.project_name}-ssm-policy"
  role = aws_iam_role.ec2_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ssm:GetParameter",
          "ssm:GetParameters",
          "ssm:GetParametersByPath"
        ]
        Resource = "arn:aws:ssm:${var.aws_region}:${data.aws_caller_identity.current.account_id}:parameter/${var.project_name}/*"
      }
    ]
  })
}

# Instance profile
resource "aws_iam_instance_profile" "ec2_profile" {
  name = "${var.project_name}-ec2-profile"
  role = aws_iam_role.ec2_role.name
}

# Elastic IP for consistent public IP
resource "aws_eip" "backend" {
  domain = "vpc"
}

# EC2 Instance
resource "aws_instance" "backend" {
  ami                    = data.aws_ami.amazon_linux_2023.id
  instance_type          = var.ec2_instance_type
  key_name               = var.ec2_key_name
  vpc_security_group_ids = [aws_security_group.backend.id]
  iam_instance_profile   = aws_iam_instance_profile.ec2_profile.name

  root_block_device {
    volume_size = 30
    volume_type = "gp3"
    encrypted   = true
  }

  user_data = base64encode(<<-EOF
    #!/bin/bash
    set -e
    exec > >(tee /var/log/user-data.log) 2>&1

    echo "=== Starting EC2 setup ==="

    # Update system
    dnf update -y

    # Install Docker
    dnf install -y docker git jq
    systemctl start docker
    systemctl enable docker
    usermod -aG docker ec2-user

    # Install Docker Compose
    curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    chmod +x /usr/local/bin/docker-compose

    # Create app directory
    mkdir -p /home/ec2-user/monti
    chown -R ec2-user:ec2-user /home/ec2-user/monti

    # ECR login
    aws ecr get-login-password --region ${var.aws_region} | docker login --username AWS --password-stdin ${data.aws_caller_identity.current.account_id}.dkr.ecr.${var.aws_region}.amazonaws.com

    # Create script to fetch secrets and generate .env
    cat > /home/ec2-user/monti/fetch-secrets.sh << 'SCRIPT'
#!/bin/bash
set -e
REGION="${var.aws_region}"
PROJECT="${var.project_name}"
ECR_REGISTRY="${data.aws_caller_identity.current.account_id}.dkr.ecr.${var.aws_region}.amazonaws.com"

# Fetch secrets from SSM
KEYCLOAK_PW=$(aws ssm get-parameter --name "/$PROJECT/keycloak/admin-password" --with-decryption --query 'Parameter.Value' --output text --region $REGION)
GRAFANA_PW=$(aws ssm get-parameter --name "/$PROJECT/grafana/admin-password" --with-decryption --query 'Parameter.Value' --output text --region $REGION)

# Get public IP for hostnames
PUBLIC_IP=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4)

# Generate .env file
cat > /home/ec2-user/monti/.env << ENVFILE
# Auto-generated from SSM Parameter Store
ECR_REGISTRY=$ECR_REGISTRY

# Keycloak
KEYCLOAK_ADMIN=admin
KEYCLOAK_ADMIN_PASSWORD=$KEYCLOAK_PW
KC_HOSTNAME=$PUBLIC_IP

# Grafana
GF_ADMIN_USER=admin
GF_ADMIN_PASSWORD=$GRAFANA_PW
GF_ROOT_URL=http://$PUBLIC_IP:3000

# Backend
ALLOWED_ORIGINS=http://$PUBLIC_IP:5173,https://*.cloudfront.net
ENVFILE

echo "Secrets fetched and .env created"
SCRIPT
    chmod +x /home/ec2-user/monti/fetch-secrets.sh
    chown ec2-user:ec2-user /home/ec2-user/monti/fetch-secrets.sh

    # Add ECR login and secrets refresh to cron
    cat > /etc/cron.d/monti << 'CRON'
# ECR login refresh (tokens expire after 12h)
0 */6 * * * root aws ecr get-login-password --region ${var.aws_region} | docker login --username AWS --password-stdin ${data.aws_caller_identity.current.account_id}.dkr.ecr.${var.aws_region}.amazonaws.com
CRON

    echo "=== EC2 setup complete ==="
    echo "Run: cd /home/ec2-user/monti && ./fetch-secrets.sh"
  EOF
  )

  tags = {
    Name = "${var.project_name}-backend"
  }

  lifecycle {
    ignore_changes = [ami] # Don't recreate on AMI updates
  }
}

# Associate Elastic IP with instance
resource "aws_eip_association" "backend" {
  instance_id   = aws_instance.backend.id
  allocation_id = aws_eip.backend.id
}
