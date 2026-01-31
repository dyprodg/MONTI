package storage

import "os"

// DynamoMode represents the DynamoDB connection mode
type DynamoMode string

const (
	DynamoModeLocal DynamoMode = "local"
	DynamoModeAWS   DynamoMode = "aws"
	DynamoModeNone  DynamoMode = "none"
)

// DynamoConfig holds DynamoDB configuration
type DynamoConfig struct {
	Mode              DynamoMode
	Endpoint          string // for local mode
	Region            string
	CallRecordsTable  string
	AgentDailyTable   string
}

// LoadDynamoConfig loads DynamoDB config from environment
func LoadDynamoConfig() DynamoConfig {
	mode := DynamoMode(getEnv("DYNAMO_MODE", "none"))
	if mode != DynamoModeLocal && mode != DynamoModeAWS {
		mode = DynamoModeNone
	}

	return DynamoConfig{
		Mode:             mode,
		Endpoint:         getEnv("DYNAMO_ENDPOINT", "http://localhost:8000"),
		Region:           getEnv("DYNAMO_REGION", "eu-central-1"),
		CallRecordsTable: getEnv("DYNAMO_CALL_RECORDS_TABLE", "monti-call-records"),
		AgentDailyTable:  getEnv("DYNAMO_AGENT_DAILY_TABLE", "monti-agent-daily-stats"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
