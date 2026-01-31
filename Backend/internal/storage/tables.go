package storage

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/rs/zerolog"
)

// CreateTablesIfNotExist creates DynamoDB tables for local development
func CreateTablesIfNotExist(ctx context.Context, client *dynamodb.Client, config DynamoConfig, logger zerolog.Logger) error {
	tables := []struct {
		name string
		pk   string
		sk   string
	}{
		{config.CallRecordsTable, "DateKey", "CallID"},
		{config.AgentDailyTable, "AgentID", "Date"},
	}

	for _, table := range tables {
		_, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
			TableName: aws.String(table.name),
		})
		if err == nil {
			logger.Info().Str("table", table.name).Msg("table already exists")
			continue
		}

		_, err = client.CreateTable(ctx, &dynamodb.CreateTableInput{
			TableName: aws.String(table.name),
			KeySchema: []dbtypes.KeySchemaElement{
				{AttributeName: aws.String(table.pk), KeyType: dbtypes.KeyTypeHash},
				{AttributeName: aws.String(table.sk), KeyType: dbtypes.KeyTypeRange},
			},
			AttributeDefinitions: []dbtypes.AttributeDefinition{
				{AttributeName: aws.String(table.pk), AttributeType: dbtypes.ScalarAttributeTypeS},
				{AttributeName: aws.String(table.sk), AttributeType: dbtypes.ScalarAttributeTypeS},
			},
			BillingMode: dbtypes.BillingModePayPerRequest,
		})
		if err != nil {
			return fmt.Errorf("failed to create table %s: %w", table.name, err)
		}
		logger.Info().Str("table", table.name).Msg("table created")
	}

	return nil
}
