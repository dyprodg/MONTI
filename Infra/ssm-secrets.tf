# SSM Parameter Store for secrets (free tier)
# These are SecureString parameters - encrypted at rest

resource "random_password" "keycloak_admin" {
  length  = 24
  special = true
}

resource "random_password" "grafana_admin" {
  length  = 24
  special = false # Grafana has issues with some special chars
}

resource "aws_ssm_parameter" "keycloak_admin_password" {
  name        = "/${var.project_name}/keycloak/admin-password"
  description = "Keycloak admin password"
  type        = "SecureString"
  value       = random_password.keycloak_admin.result
}

resource "aws_ssm_parameter" "grafana_admin_password" {
  name        = "/${var.project_name}/grafana/admin-password"
  description = "Grafana admin password"
  type        = "SecureString"
  value       = random_password.grafana_admin.result
}

# Output the passwords (only shown once, store them safely!)
output "keycloak_admin_password" {
  description = "Keycloak admin password"
  value       = random_password.keycloak_admin.result
  sensitive   = true
}

output "grafana_admin_password" {
  description = "Grafana admin password"
  value       = random_password.grafana_admin.result
  sensitive   = true
}
