# Remote state configuration
# Note: Run bootstrap first to create the S3 bucket

terraform {
  backend "s3" {
    bucket = "monti-terraform-state"
    key    = "infrastructure/terraform.tfstate"
    region = "eu-central-1"

    # Terraform 1.6+ supports native S3 state locking
    use_lockfile = true
  }
}
