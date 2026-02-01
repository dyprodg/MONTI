# DNS (Route53) and SSL (ACM) configuration

locals {
  frontend_fqdn = "${var.frontend_subdomain}.${var.domain_name}"
  backend_fqdn  = "${var.backend_subdomain}.${var.domain_name}"
  grafana_fqdn  = "monti-grafana.${var.domain_name}"
}

# Existing hosted zone
data "aws_route53_zone" "main" {
  name = var.domain_name
}

# --- Frontend: ACM certificate in us-east-1 (required by CloudFront) ---

resource "aws_acm_certificate" "frontend" {
  provider          = aws.us_east_1
  domain_name       = local.frontend_fqdn
  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_route53_record" "frontend_cert_validation" {
  for_each = {
    for dvo in aws_acm_certificate.frontend.domain_validation_options : dvo.domain_name => {
      name   = dvo.resource_record_name
      record = dvo.resource_record_value
      type   = dvo.resource_record_type
    }
  }

  zone_id = data.aws_route53_zone.main.zone_id
  name    = each.value.name
  type    = each.value.type
  records = [each.value.record]
  ttl     = 60
}

resource "aws_acm_certificate_validation" "frontend" {
  provider                = aws.us_east_1
  certificate_arn         = aws_acm_certificate.frontend.arn
  validation_record_fqdns = [for record in aws_route53_record.frontend_cert_validation : record.fqdn]
}

# --- DNS Records ---

# Frontend: CNAME to CloudFront
resource "aws_route53_record" "frontend" {
  zone_id = data.aws_route53_zone.main.zone_id
  name    = local.frontend_fqdn
  type    = "A"

  alias {
    name                   = aws_cloudfront_distribution.frontend.domain_name
    zone_id                = aws_cloudfront_distribution.frontend.hosted_zone_id
    evaluate_target_health = false
  }
}

# Backend: A record to EC2 Elastic IP
resource "aws_route53_record" "backend" {
  zone_id = data.aws_route53_zone.main.zone_id
  name    = local.backend_fqdn
  type    = "A"
  ttl     = 300
  records = [aws_eip.backend.public_ip]
}

# Grafana: A record to EC2 Elastic IP
resource "aws_route53_record" "grafana" {
  zone_id = data.aws_route53_zone.main.zone_id
  name    = local.grafana_fqdn
  type    = "A"
  ttl     = 300
  records = [aws_eip.backend.public_ip]
}
