package storage

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/dennisdiepolder/monti/backend/internal/types"
	"github.com/rs/zerolog"
)

// DynamoDBStore implements Store using AWS DynamoDB
type DynamoDBStore struct {
	client *dynamodb.Client
	config DynamoConfig
	logger zerolog.Logger
}

// NewDynamoDBStore creates a new DynamoDB store
func NewDynamoDBStore(ctx context.Context, cfg DynamoConfig, logger zerolog.Logger) (*DynamoDBStore, error) {
	var client *dynamodb.Client

	if cfg.Mode == DynamoModeLocal {
		// For local mode, build the client directly without LoadDefaultConfig.
		// LoadDefaultConfig probes the EC2 IMDS endpoint which hangs on EC2
		// instances when static credentials are intended.
		client = dynamodb.New(dynamodb.Options{
			Region:       cfg.Region,
			BaseEndpoint: aws.String(cfg.Endpoint),
			Credentials:  credentials.NewStaticCredentialsProvider("local", "local", ""),
		})
	} else {
		awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.Region))
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config: %w", err)
		}
		client = dynamodb.NewFromConfig(awsCfg)
	}

	store := &DynamoDBStore{
		client: client,
		config: cfg,
		logger: logger,
	}

	// Create tables in local mode
	if cfg.Mode == DynamoModeLocal {
		if err := CreateTablesIfNotExist(ctx, client, cfg, logger); err != nil {
			return nil, err
		}
	}

	logger.Info().
		Str("mode", string(cfg.Mode)).
		Str("region", cfg.Region).
		Msg("DynamoDB store initialized")

	return store, nil
}

func (s *DynamoDBStore) SaveCallRecord(record types.CallRecord) error {
	item, err := attributevalue.MarshalMap(record)
	if err != nil {
		return fmt.Errorf("failed to marshal call record: %w", err)
	}

	_, err = s.client.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName: aws.String(s.config.CallRecordsTable),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to save call record: %w", err)
	}
	return nil
}

func (s *DynamoDBStore) SaveAgentDailyStats(stats types.AgentDailyStats) error {
	item, err := attributevalue.MarshalMap(stats)
	if err != nil {
		return fmt.Errorf("failed to marshal agent daily stats: %w", err)
	}

	_, err = s.client.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName: aws.String(s.config.AgentDailyTable),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to save agent daily stats: %w", err)
	}
	return nil
}

func (s *DynamoDBStore) GetCallRecords(dateKey string) ([]types.CallRecord, error) {
	keyCond := expression.Key("DateKey").Equal(expression.Value(dateKey))
	expr, err := expression.NewBuilder().WithKeyCondition(keyCond).Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build expression: %w", err)
	}

	result, err := s.client.Query(context.Background(), &dynamodb.QueryInput{
		TableName:                 aws.String(s.config.CallRecordsTable),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query call records: %w", err)
	}

	var records []types.CallRecord
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &records); err != nil {
		return nil, fmt.Errorf("failed to unmarshal call records: %w", err)
	}
	return records, nil
}

func (s *DynamoDBStore) GetAgentDailyStats(agentID string) ([]types.AgentDailyStats, error) {
	keyCond := expression.Key("AgentID").Equal(expression.Value(agentID))
	expr, err := expression.NewBuilder().WithKeyCondition(keyCond).Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build expression: %w", err)
	}

	result, err := s.client.Query(context.Background(), &dynamodb.QueryInput{
		TableName:                 aws.String(s.config.AgentDailyTable),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query agent daily stats: %w", err)
	}

	var stats []types.AgentDailyStats
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &stats); err != nil {
		return nil, fmt.Errorf("failed to unmarshal agent daily stats: %w", err)
	}
	return stats, nil
}

func (s *DynamoDBStore) GetAgentCallsByDate(agentID, date string) ([]types.CallRecord, error) {
	// Scan call records for this date filtered by agentID
	keyCond := expression.Key("DateKey").Equal(expression.Value(date))
	filter := expression.Name("AgentID").Equal(expression.Value(agentID))
	expr, err := expression.NewBuilder().WithKeyCondition(keyCond).WithFilter(filter).Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build expression: %w", err)
	}

	result, err := s.client.Query(context.Background(), &dynamodb.QueryInput{
		TableName:                 aws.String(s.config.CallRecordsTable),
		KeyConditionExpression:    expr.KeyCondition(),
		FilterExpression:          expr.Filter(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query agent calls: %w", err)
	}

	var records []types.CallRecord
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &records); err != nil {
		return nil, fmt.Errorf("failed to unmarshal call records: %w", err)
	}
	return records, nil
}

// NewStore creates the appropriate store based on configuration
func NewStore(ctx context.Context, logger zerolog.Logger) (Store, error) {
	cfg := LoadDynamoConfig()

	switch cfg.Mode {
	case DynamoModeLocal, DynamoModeAWS:
		return NewDynamoDBStore(ctx, cfg, logger)
	default:
		logger.Info().Msg("DynamoDB disabled (DYNAMO_MODE=none)")
		return NewNoopStore(), nil
	}
}

// TruncateAll deletes all items from both DynamoDB tables (scan + batch delete)
func (s *DynamoDBStore) TruncateAll() error {
	tables := []struct {
		name string
		pk   string
		sk   string
	}{
		{s.config.CallRecordsTable, "DateKey", "CallID"},
		{s.config.AgentDailyTable, "AgentID", "Date"},
	}

	for _, table := range tables {
		if err := s.truncateTable(table.name, table.pk, table.sk); err != nil {
			return fmt.Errorf("failed to truncate %s: %w", table.name, err)
		}
	}
	return nil
}

func (s *DynamoDBStore) truncateTable(tableName, pk, sk string) error {
	var lastKey map[string]dbtypes.AttributeValue

	for {
		input := &dynamodb.ScanInput{
			TableName:            aws.String(tableName),
			ProjectionExpression: aws.String("#pk, #sk"),
			ExpressionAttributeNames: map[string]string{
				"#pk": pk,
				"#sk": sk,
			},
			Limit: aws.Int32(500),
		}
		if lastKey != nil {
			input.ExclusiveStartKey = lastKey
		}

		result, err := s.client.Scan(context.Background(), input)
		if err != nil {
			return err
		}

		// Batch delete in groups of 25
		for i := 0; i < len(result.Items); i += 25 {
			end := i + 25
			if end > len(result.Items) {
				end = len(result.Items)
			}

			requests := make([]dbtypes.WriteRequest, 0, end-i)
			for _, item := range result.Items[i:end] {
				requests = append(requests, dbtypes.WriteRequest{
					DeleteRequest: &dbtypes.DeleteRequest{
						Key: map[string]dbtypes.AttributeValue{
							pk: item[pk],
							sk: item[sk],
						},
					},
				})
			}

			_, err := s.client.BatchWriteItem(context.Background(), &dynamodb.BatchWriteItemInput{
				RequestItems: map[string][]dbtypes.WriteRequest{
					tableName: requests,
				},
			})
			if err != nil {
				return err
			}
		}

		lastKey = result.LastEvaluatedKey
		if lastKey == nil {
			break
		}
	}

	s.logger.Info().Str("table", tableName).Msg("table truncated")
	return nil
}

// DynamoDBStore also implements a method needed by callqueue for global secondary index queries
// using a simple scan with filter. For production, a GSI on AgentID would be more efficient.
func (s *DynamoDBStore) queryByFilter(tableName string, filterExpr string, values map[string]dbtypes.AttributeValue) (*dynamodb.ScanOutput, error) {
	return s.client.Scan(context.Background(), &dynamodb.ScanInput{
		TableName:                 aws.String(tableName),
		FilterExpression:          aws.String(filterExpr),
		ExpressionAttributeValues: values,
	})
}
