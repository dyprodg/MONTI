# Temporary test EC2 instance to verify Terraform pipeline
# DELETE THIS FILE after confirming the pipeline works

resource "aws_instance" "pipeline_test" {
  ami           = data.aws_ami.amazon_linux_2023.id
  instance_type = "t3.nano"

  tags = {
    Name    = "${var.project_name}-pipeline-test"
    Purpose = "CI/CD pipeline verification - safe to delete"
  }
}

output "pipeline_test_instance_id" {
  description = "Test instance ID (delete after verification)"
  value       = aws_instance.pipeline_test.id
}
