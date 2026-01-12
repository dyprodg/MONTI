# Infra

Infrastructure as Code (IaC) for provisioning and managing AWS resources for MONTI.

## Purpose

Terraform-based infrastructure management for:
- AWS IAM Identity Center configuration
- Backend infrastructure (Lambda/ECS/EC2)
- Database and cache (RDS/DynamoDB/Redis)
- Networking (VPC, subnets, security groups)
- Frontend hosting (S3, CloudFront)
- Monitoring and logging (CloudWatch)

## Rules

### Terraform Best Practices
- Use Terraform >= 1.0
- Follow HCL formatting (`terraform fmt`)
- Use meaningful resource names
- Tag all resources consistently (Environment, Project, ManagedBy)
- Use data sources for existing resources

### Module Organization
- Create reusable modules for common patterns
- Module structure: `modules/<module-name>/`
- Each module should have: `main.tf`, `variables.tf`, `outputs.tf`, `README.md`
- Version modules if sharing across projects

### File Structure
```
Infra/
├── environments/
│   ├── dev/
│   ├── staging/
│   └── prod/
├── modules/
│   ├── vpc/
│   ├── database/
│   ├── backend/
│   ├── iam/
│   └── ...
├── main.tf
├── variables.tf
├── outputs.tf
├── backend.tf
└── README.md
```

### State Management
- Use remote state (S3 backend)
- Enable state locking (DynamoDB)
- Never commit `.tfstate` files
- Use separate state files per environment
- Implement state backup strategy

### Variables & Secrets
- All configurable values must be variables
- Use `.tfvars` files for environment-specific values
- Never commit secrets to git
- Use AWS Secrets Manager or Parameter Store for sensitive data
- Document all variables in `variables.tf`

### Security
- Follow principle of least privilege for IAM
- Enable encryption at rest and in transit
- Use private subnets for databases and backend services
- Implement proper security groups (no 0.0.0.0/0 unless necessary)
- Enable AWS CloudTrail and Config
- Regular security audits with `tfsec` or similar tools

### Networking
- Multi-AZ deployment for high availability
- Public and private subnets
- NAT gateway for private subnet internet access
- VPC peering if needed for services
- Network ACLs and security groups properly configured

### Resource Naming
- Use consistent naming convention: `<project>-<environment>-<resource-type>-<name>`
- Example: `monti-prod-rds-main`, `monti-dev-ecs-backend`

### Outputs
- Export important values as outputs (IDs, ARNs, endpoints)
- Document what each output is used for
- Use outputs for cross-stack references

### Cost Optimization
- Use appropriate instance sizes
- Enable auto-scaling where applicable
- Use spot instances for non-critical workloads
- Set up cost alerts and budgets
- Review and clean up unused resources regularly

### Multi-Environment Support
- Support dev, staging, and production environments
- Use workspaces or separate state files
- Environment-specific variables
- Separate AWS accounts per environment (recommended)

### Documentation
- Document architecture decisions
- Include network diagrams
- List all manual steps (if any)
- Document disaster recovery procedures
- Keep README updated

### Testing
- Validate Terraform files (`terraform validate`)
- Plan before apply (`terraform plan`)
- Use `terraform-compliance` for policy testing
- Test in dev environment before production
- Implement drift detection

### Change Management
- Always run `terraform plan` and review before applying
- Use `-target` flag only when absolutely necessary
- Document significant infrastructure changes
- Implement approval process for production changes
- Keep change log

### Modules to Create
1. **VPC Module**: Networking setup (VPC, subnets, routing)
2. **IAM Module**: Identity Center, roles, policies
3. **Database Module**: RDS/DynamoDB setup
4. **Cache Module**: ElastiCache Redis setup
5. **Backend Module**: ECS/Lambda/EC2 for Go backend
6. **Frontend Module**: S3 + CloudFront for web app
7. **Monitoring Module**: CloudWatch, alarms, dashboards

## Dependencies

- Must not depend on application code (Backend, Frontend, AgentSim)
- Can reference CICD for deployment automation

## CI/CD

This infrastructure will only be planned/applied when:
- Files in `Infra/` directory are changed
- Root configuration files affecting infrastructure are changed
- Explicitly triggered via workflow dispatch

### CI Steps
1. Terraform format check (`terraform fmt -check`)
2. Terraform validation (`terraform validate`)
3. Security scanning (`tfsec`, `checkov`)
4. Generate plan (`terraform plan`)
5. Plan should be reviewed before apply

### CD Steps (requires approval)
1. Apply Terraform changes (`terraform apply`)
2. Verify resources are created correctly
3. Run smoke tests
4. Update documentation if needed

### Terraform Backend Configuration
```hcl
terraform {
  backend "s3" {
    bucket         = "monti-terraform-state"
    key            = "env/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true
    dynamodb_table = "monti-terraform-locks"
  }
}
```

## Getting Started

1. Install Terraform
2. Configure AWS credentials
3. Initialize Terraform: `terraform init`
4. Select workspace: `terraform workspace select dev`
5. Plan changes: `terraform plan -var-file=environments/dev/terraform.tfvars`
6. Apply changes: `terraform apply -var-file=environments/dev/terraform.tfvars`
