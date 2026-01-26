# IAM User for GitHub Actions CI/CD
# Minimal permissions - only what's needed for deployments

resource "aws_iam_user" "github_actions" {
  name = "${var.project_name}-github-actions"
  path = "/cicd/"
}

resource "aws_iam_access_key" "github_actions" {
  user = aws_iam_user.github_actions.name
}

# Policy for ECR push/pull
resource "aws_iam_user_policy" "ecr_policy" {
  name = "${var.project_name}-cicd-ecr"
  user = aws_iam_user.github_actions.name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "ECRAuth"
        Effect = "Allow"
        Action = [
          "ecr:GetAuthorizationToken"
        ]
        Resource = "*"
      },
      {
        Sid    = "ECRPushPull"
        Effect = "Allow"
        Action = [
          "ecr:BatchCheckLayerAvailability",
          "ecr:GetDownloadUrlForLayer",
          "ecr:BatchGetImage",
          "ecr:PutImage",
          "ecr:InitiateLayerUpload",
          "ecr:UploadLayerPart",
          "ecr:CompleteLayerUpload"
        ]
        Resource = [
          aws_ecr_repository.backend.arn,
          aws_ecr_repository.agentsim.arn
        ]
      }
    ]
  })
}

# Policy for S3 frontend deployment
resource "aws_iam_user_policy" "s3_policy" {
  name = "${var.project_name}-cicd-s3"
  user = aws_iam_user.github_actions.name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "S3FrontendSync"
        Effect = "Allow"
        Action = [
          "s3:PutObject",
          "s3:GetObject",
          "s3:DeleteObject",
          "s3:ListBucket"
        ]
        Resource = [
          aws_s3_bucket.frontend.arn,
          "${aws_s3_bucket.frontend.arn}/*"
        ]
      }
    ]
  })
}

# Policy for CloudFront invalidation
resource "aws_iam_user_policy" "cloudfront_policy" {
  name = "${var.project_name}-cicd-cloudfront"
  user = aws_iam_user.github_actions.name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "CloudFrontInvalidation"
        Effect = "Allow"
        Action = [
          "cloudfront:CreateInvalidation",
          "cloudfront:GetInvalidation",
          "cloudfront:ListInvalidations"
        ]
        Resource = aws_cloudfront_distribution.frontend.arn
      }
    ]
  })
}

# Outputs for the CI/CD credentials
output "cicd_access_key_id" {
  description = "Access key ID for GitHub Actions"
  value       = aws_iam_access_key.github_actions.id
  sensitive   = false
}

output "cicd_secret_access_key" {
  description = "Secret access key for GitHub Actions (sensitive)"
  value       = aws_iam_access_key.github_actions.secret
  sensitive   = true
}
