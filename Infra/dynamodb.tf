# DynamoDB tables for call records and agent daily stats

resource "aws_dynamodb_table" "call_records" {
  name         = "${var.project_name}-call-records"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "DateKey"
  range_key    = "CallID"

  attribute {
    name = "DateKey"
    type = "S"
  }

  attribute {
    name = "CallID"
    type = "S"
  }

  tags = {
    Name = "${var.project_name}-call-records"
  }
}

resource "aws_dynamodb_table" "agent_daily_stats" {
  name         = "${var.project_name}-agent-daily-stats"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "AgentID"
  range_key    = "Date"

  attribute {
    name = "AgentID"
    type = "S"
  }

  attribute {
    name = "Date"
    type = "S"
  }

  tags = {
    Name = "${var.project_name}-agent-daily-stats"
  }
}
