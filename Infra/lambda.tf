# Lambda function to clear DynamoDB tables nightly at 02:00 UTC

data "archive_file" "dynamo_cleanup" {
  type        = "zip"
  source_file = "${path.module}/lambda/dynamo_cleanup.py"
  output_path = "${path.module}/lambda/dynamo_cleanup.zip"
}

# IAM role for Lambda execution
resource "aws_iam_role" "lambda_dynamo_cleanup" {
  name = "${var.project_name}-lambda-dynamo-cleanup"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })
}

resource "aws_iam_role_policy" "lambda_dynamo_cleanup" {
  name = "${var.project_name}-lambda-dynamo-cleanup"
  role = aws_iam_role.lambda_dynamo_cleanup.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "dynamodb:Scan",
          "dynamodb:BatchWriteItem",
          "dynamodb:DeleteItem"
        ]
        Resource = [
          aws_dynamodb_table.call_records.arn,
          aws_dynamodb_table.agent_daily_stats.arn
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Resource = "arn:aws:logs:${var.aws_region}:${data.aws_caller_identity.current.account_id}:*"
      }
    ]
  })
}

# Lambda function
resource "aws_lambda_function" "dynamo_cleanup" {
  filename         = data.archive_file.dynamo_cleanup.output_path
  source_code_hash = data.archive_file.dynamo_cleanup.output_base64sha256
  function_name    = "${var.project_name}-dynamo-cleanup"
  role             = aws_iam_role.lambda_dynamo_cleanup.arn
  handler          = "dynamo_cleanup.handler"
  runtime          = "python3.12"
  timeout          = 900 # 15 minutes
  memory_size      = 256

  environment {
    variables = {
      CALL_RECORDS_TABLE = aws_dynamodb_table.call_records.name
      AGENT_DAILY_TABLE  = aws_dynamodb_table.agent_daily_stats.name
    }
  }

  tags = {
    Name = "${var.project_name}-dynamo-cleanup"
  }
}

# EventBridge rule - 02:00 UTC every night
resource "aws_cloudwatch_event_rule" "nightly_dynamo_cleanup" {
  name                = "${var.project_name}-nightly-dynamo-cleanup"
  description         = "Clear DynamoDB tables at 02:00 UTC daily"
  schedule_expression = "cron(0 2 * * ? *)"

  tags = {
    Name = "${var.project_name}-nightly-dynamo-cleanup"
  }
}

resource "aws_cloudwatch_event_target" "dynamo_cleanup" {
  rule = aws_cloudwatch_event_rule.nightly_dynamo_cleanup.name
  arn  = aws_lambda_function.dynamo_cleanup.arn
}

resource "aws_lambda_permission" "allow_eventbridge" {
  statement_id  = "AllowEventBridgeInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.dynamo_cleanup.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.nightly_dynamo_cleanup.arn
}
