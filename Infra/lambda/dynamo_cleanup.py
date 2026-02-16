"""Lambda function to clear all items from MONTI DynamoDB tables.

Runs nightly at 02:00 UTC via EventBridge cron schedule.
Can also be invoked manually via CLI:
  aws lambda invoke --function-name monti-dynamo-cleanup /dev/stdout
"""

import os

import boto3

REGION = os.environ.get("AWS_REGION", "eu-central-1")

TABLES = [
    {
        "name": os.environ.get("CALL_RECORDS_TABLE", "monti-call-records"),
        "pk": "DateKey",
        "sk": "CallID",
    },
    {
        "name": os.environ.get("AGENT_DAILY_TABLE", "monti-agent-daily-stats"),
        "pk": "AgentID",
        "sk": "Date",
    },
]


def truncate_table(dynamodb, table_name, pk, sk):
    """Delete all items from a DynamoDB table using scan + batch delete."""
    table = dynamodb.Table(table_name)
    total = 0

    scan_kwargs = {
        "ProjectionExpression": f"#pk, #sk",
        "ExpressionAttributeNames": {"#pk": pk, "#sk": sk},
    }

    while True:
        response = table.scan(**scan_kwargs)
        items = response.get("Items", [])

        if not items:
            break

        with table.batch_writer() as batch:
            for item in items:
                batch.delete_item(Key={pk: item[pk], sk: item[sk]})
                total += 1

        if "LastEvaluatedKey" not in response:
            break
        scan_kwargs["ExclusiveStartKey"] = response["LastEvaluatedKey"]

    return total


def handler(event, context):
    dynamodb = boto3.resource("dynamodb", region_name=REGION)

    results = {}
    for table_config in TABLES:
        count = truncate_table(
            dynamodb,
            table_config["name"],
            table_config["pk"],
            table_config["sk"],
        )
        results[table_config["name"]] = count
        print(f"Deleted {count} items from {table_config['name']}")

    return {"statusCode": 200, "body": results}


if __name__ == "__main__":
    result = handler({}, None)
    print(result)
