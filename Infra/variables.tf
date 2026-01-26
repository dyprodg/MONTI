variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "eu-central-1"
}

variable "project_name" {
  description = "Project name for resource naming"
  type        = string
  default     = "monti"
}

variable "environment" {
  description = "Environment (dev, staging, prod)"
  type        = string
  default     = "prod"
}

variable "domain_name" {
  description = "Main domain name (e.g., monti.app)"
  type        = string
  default     = "monti.app"
}

variable "ec2_instance_type" {
  description = "EC2 instance type for backend"
  type        = string
  default     = "t3.small"
}

variable "ec2_key_name" {
  description = "Name of the SSH key pair for EC2"
  type        = string
  default     = "monti-key"
}

variable "allowed_ssh_cidrs" {
  description = "CIDR blocks allowed to SSH to EC2"
  type        = list(string)
  default     = ["0.0.0.0/0"] # Restrict this in production!
}
