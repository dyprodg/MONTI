package ticker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/dennisdiepolder/monti/backend/internal/websocket"
	"github.com/rs/zerolog"
)

// TimeMessage represents the time update message sent to clients
type TimeMessage struct {
	Timestamp  string `json:"timestamp"`
	ServerTime int64  `json:"serverTime"`
}

// Ticker periodically broadcasts time updates to the hub
type Ticker struct {
	hub      *websocket.Hub
	interval time.Duration
	logger   zerolog.Logger
}

// NewTicker creates a new Ticker
func NewTicker(hub *websocket.Hub, interval time.Duration, logger zerolog.Logger) *Ticker {
	return &Ticker{
		hub:      hub,
		interval: interval,
		logger:   logger,
	}
}

// Start begins broadcasting time updates
func (t *Ticker) Start(ctx context.Context) {
	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()

	t.logger.Info().Dur("interval", t.interval).Msg("ticker started")

	for {
		select {
		case <-ctx.Done():
			t.logger.Info().Msg("ticker stopped")
			return

		case now := <-ticker.C:
			message := TimeMessage{
				Timestamp:  now.Format(time.RFC3339),
				ServerTime: now.Unix(),
			}

			data, err := json.Marshal(message)
			if err != nil {
				t.logger.Error().Err(err).Msg("failed to marshal time message")
				continue
			}

			t.hub.Broadcast(data)
			t.logger.Debug().
				Str("timestamp", message.Timestamp).
				Int("clients", t.hub.ClientCount()).
				Msg("broadcasted time update")
		}
	}
}
