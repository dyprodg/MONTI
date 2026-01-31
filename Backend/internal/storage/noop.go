package storage

import "github.com/dennisdiepolder/monti/backend/internal/types"

// Store defines the storage interface
type Store interface {
	SaveCallRecord(record types.CallRecord) error
	SaveAgentDailyStats(stats types.AgentDailyStats) error
	GetCallRecords(dateKey string) ([]types.CallRecord, error)
	GetAgentDailyStats(agentID string) ([]types.AgentDailyStats, error)
	GetAgentCallsByDate(agentID, date string) ([]types.CallRecord, error)
	TruncateAll() error
}

// NoopStore is a no-op implementation when DynamoDB is disabled
type NoopStore struct{}

func NewNoopStore() *NoopStore { return &NoopStore{} }

func (s *NoopStore) SaveCallRecord(_ types.CallRecord) error              { return nil }
func (s *NoopStore) SaveAgentDailyStats(_ types.AgentDailyStats) error    { return nil }
func (s *NoopStore) GetCallRecords(_ string) ([]types.CallRecord, error)  { return nil, nil }
func (s *NoopStore) GetAgentDailyStats(_ string) ([]types.AgentDailyStats, error) { return nil, nil }
func (s *NoopStore) GetAgentCallsByDate(_, _ string) ([]types.CallRecord, error)  { return nil, nil }
func (s *NoopStore) TruncateAll() error                                           { return nil }
