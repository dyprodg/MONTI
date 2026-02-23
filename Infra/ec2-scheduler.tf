# EventBridge Scheduler for EC2 start/stop
# Runs the backend EC2 only during 14:00-16:00 Berlin time, weekdays

# IAM role for EventBridge Scheduler
resource "aws_iam_role" "ec2_scheduler" {
  name = "${var.project_name}-ec2-scheduler"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "scheduler.amazonaws.com"
        }
      }
    ]
  })
}

resource "aws_iam_role_policy" "ec2_scheduler" {
  name = "${var.project_name}-ec2-scheduler"
  role = aws_iam_role.ec2_scheduler.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ec2:StartInstances",
          "ec2:StopInstances"
        ]
        Resource = aws_instance.backend.arn
      }
    ]
  })
}

# Schedule group
resource "aws_scheduler_schedule_group" "ec2" {
  name = "${var.project_name}-ec2-scheduler"
}

# Start EC2 at 13:45 Berlin time (15 min boot buffer before 14:00 window)
resource "aws_scheduler_schedule" "ec2_start" {
  name       = "${var.project_name}-ec2-start"
  group_name = aws_scheduler_schedule_group.ec2.name

  schedule_expression          = "cron(45 13 ? * MON-FRI *)"
  schedule_expression_timezone = "Europe/Berlin"

  flexible_time_window {
    mode = "OFF"
  }

  target {
    arn      = "arn:aws:scheduler:::aws-sdk:ec2:startInstances"
    role_arn = aws_iam_role.ec2_scheduler.arn

    input = jsonencode({
      InstanceIds = [aws_instance.backend.id]
    })
  }
}

# Stop EC2 at 16:05 Berlin time (5 min grace after 16:00 window closes)
resource "aws_scheduler_schedule" "ec2_stop" {
  name       = "${var.project_name}-ec2-stop"
  group_name = aws_scheduler_schedule_group.ec2.name

  schedule_expression          = "cron(5 16 ? * MON-FRI *)"
  schedule_expression_timezone = "Europe/Berlin"

  flexible_time_window {
    mode = "OFF"
  }

  target {
    arn      = "arn:aws:scheduler:::aws-sdk:ec2:stopInstances"
    role_arn = aws_iam_role.ec2_scheduler.arn

    input = jsonencode({
      InstanceIds = [aws_instance.backend.id]
    })
  }
}
