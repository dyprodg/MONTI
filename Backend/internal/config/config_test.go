package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr bool
		check   func(*testing.T, *Config)
	}{
		{
			name: "default values",
			env:  map[string]string{},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Port != "8080" {
					t.Errorf("expected port 8080, got %s", cfg.Port)
				}
				if cfg.LogLevel != "info" {
					t.Errorf("expected log level info, got %s", cfg.LogLevel)
				}
				if cfg.WSReadTimeout != 60*time.Second {
					t.Errorf("expected WSReadTimeout 60s, got %v", cfg.WSReadTimeout)
				}
			},
		},
		{
			name: "custom values",
			env: map[string]string{
				"PORT":             "9000",
				"LOG_LEVEL":        "debug",
				"WS_READ_TIMEOUT":  "30",
				"WS_WRITE_TIMEOUT": "5",
				"ALLOWED_ORIGINS":  "http://example.com,http://test.com",
			},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Port != "9000" {
					t.Errorf("expected port 9000, got %s", cfg.Port)
				}
				if cfg.LogLevel != "debug" {
					t.Errorf("expected log level debug, got %s", cfg.LogLevel)
				}
				if cfg.WSReadTimeout != 30*time.Second {
					t.Errorf("expected WSReadTimeout 30s, got %v", cfg.WSReadTimeout)
				}
				if cfg.WSWriteTimeout != 5*time.Second {
					t.Errorf("expected WSWriteTimeout 5s, got %v", cfg.WSWriteTimeout)
				}
				if len(cfg.AllowedOrigins) != 2 {
					t.Errorf("expected 2 allowed origins, got %d", len(cfg.AllowedOrigins))
				}
			},
		},
		{
			name: "invalid WS_READ_TIMEOUT",
			env: map[string]string{
				"WS_READ_TIMEOUT": "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid WS_WRITE_TIMEOUT",
			env: map[string]string{
				"WS_WRITE_TIMEOUT": "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Clearenv()

			// Set test environment variables
			for k, v := range tt.env {
				os.Setenv(k, v)
			}

			// Load config
			cfg, err := Load()

			// Check error
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Run custom checks
			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestWebSocketConstants(t *testing.T) {
	// Clear environment and set clean defaults
	os.Clearenv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// PongWait should equal WSReadTimeout
	if cfg.PongWait != cfg.WSReadTimeout {
		t.Errorf("PongWait (%v) should equal WSReadTimeout (%v)", cfg.PongWait, cfg.WSReadTimeout)
	}

	// PingPeriod should be less than PongWait
	if cfg.PingPeriod >= cfg.PongWait {
		t.Errorf("PingPeriod (%v) should be less than PongWait (%v)", cfg.PingPeriod, cfg.PongWait)
	}

	// WriteWait should equal WSWriteTimeout
	if cfg.WriteWait != cfg.WSWriteTimeout {
		t.Errorf("WriteWait (%v) should equal WSWriteTimeout (%v)", cfg.WriteWait, cfg.WSWriteTimeout)
	}

	// MaxMessageSize should be set
	if cfg.MaxMessageSize <= 0 {
		t.Errorf("MaxMessageSize should be positive, got %d", cfg.MaxMessageSize)
	}
}
