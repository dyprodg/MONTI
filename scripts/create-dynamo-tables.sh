#!/bin/bash
# Create DynamoDB tables for local development
# Usage: ./scripts/create-dynamo-tables.sh [endpoint]

ENDPOINT="${1:-http://localhost:8000}"

echo "Creating DynamoDB tables at $ENDPOINT..."

aws dynamodb create-table \
  --table-name monti-call-records \
  --attribute-definitions \
    AttributeName=DateKey,AttributeType=S \
    AttributeName=CallID,AttributeType=S \
  --key-schema \
    AttributeName=DateKey,KeyType=HASH \
    AttributeName=CallID,KeyType=RANGE \
  --billing-mode PAY_PER_REQUEST \
  --endpoint-url "$ENDPOINT" \
  --region eu-central-1 \
  2>/dev/null && echo "Created monti-call-records" || echo "monti-call-records already exists"

aws dynamodb create-table \
  --table-name monti-agent-daily-stats \
  --attribute-definitions \
    AttributeName=AgentID,AttributeType=S \
    AttributeName=Date,AttributeType=S \
  --key-schema \
    AttributeName=AgentID,KeyType=HASH \
    AttributeName=Date,KeyType=RANGE \
  --billing-mode PAY_PER_REQUEST \
  --endpoint-url "$ENDPOINT" \
  --region eu-central-1 \
  2>/dev/null && echo "Created monti-agent-daily-stats" || echo "monti-agent-daily-stats already exists"

echo "Done."
