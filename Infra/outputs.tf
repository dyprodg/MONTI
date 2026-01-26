# Outputs

# ECR
output "ecr_backend_url" {
  description = "ECR repository URL for backend"
  value       = aws_ecr_repository.backend.repository_url
}

output "ecr_agentsim_url" {
  description = "ECR repository URL for agentsim"
  value       = aws_ecr_repository.agentsim.repository_url
}

# Frontend
output "frontend_bucket_name" {
  description = "S3 bucket name for frontend"
  value       = aws_s3_bucket.frontend.id
}

output "cloudfront_distribution_id" {
  description = "CloudFront distribution ID"
  value       = aws_cloudfront_distribution.frontend.id
}

output "cloudfront_domain_name" {
  description = "CloudFront domain name"
  value       = aws_cloudfront_distribution.frontend.domain_name
}

# EC2
output "ec2_public_ip" {
  description = "Public IP of backend EC2 instance"
  value       = aws_eip.backend.public_ip
}

output "ec2_instance_id" {
  description = "EC2 instance ID"
  value       = aws_instance.backend.id
}

# Useful commands
output "ecr_login_command" {
  description = "Command to login to ECR"
  value       = "aws ecr get-login-password --region ${var.aws_region} | docker login --username AWS --password-stdin ${data.aws_caller_identity.current.account_id}.dkr.ecr.${var.aws_region}.amazonaws.com"
}

output "ssh_command" {
  description = "SSH command to connect to EC2"
  value       = "ssh -i ~/.ssh/${var.ec2_key_name}.pem ec2-user@${aws_eip.backend.public_ip}"
}
