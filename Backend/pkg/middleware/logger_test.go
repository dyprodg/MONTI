package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
)

func TestLogger(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with logger middleware
	loggedHandler := Logger(logger)(handler)

	// Create a test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Execute the request
	loggedHandler.ServeHTTP(rec, req)

	// Check response
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Parse log output
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log entry: %v", err)
	}

	// Verify log fields
	if logEntry["method"] != "GET" {
		t.Errorf("expected method GET, got %v", logEntry["method"])
	}
	if logEntry["path"] != "/test" {
		t.Errorf("expected path /test, got %v", logEntry["path"])
	}
	if logEntry["status"] != float64(200) {
		t.Errorf("expected status 200, got %v", logEntry["status"])
	}
	if logEntry["message"] != "request completed" {
		t.Errorf("expected message 'request completed', got %v", logEntry["message"])
	}
}

func TestLoggerWithErrorStatus(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	})

	loggedHandler := Logger(logger)(handler)

	req := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	rec := httptest.NewRecorder()

	loggedHandler.ServeHTTP(rec, req)

	var logEntry map[string]interface{}
	json.Unmarshal(buf.Bytes(), &logEntry)

	if logEntry["status"] != float64(404) {
		t.Errorf("expected status 404, got %v", logEntry["status"])
	}
}
