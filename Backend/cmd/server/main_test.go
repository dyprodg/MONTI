package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	healthHandler(rec, req)

	// Check status code
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Check content type
	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	// Parse response body
	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Check response fields
	if response["status"] != "ok" {
		t.Errorf("expected status ok, got %s", response["status"])
	}
	if response["service"] != "monti-backend" {
		t.Errorf("expected service monti-backend, got %s", response["service"])
	}
}

func TestHealthHandlerMethods(t *testing.T) {
	tests := []struct {
		method         string
		expectedStatus int
	}{
		{http.MethodGet, http.StatusOK},
		{http.MethodPost, http.StatusOK},    // Handler doesn't check method
		{http.MethodPut, http.StatusOK},     // Handler doesn't check method
		{http.MethodDelete, http.StatusOK},  // Handler doesn't check method
		{http.MethodOptions, http.StatusOK}, // Handler doesn't check method
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/health", nil)
			rec := httptest.NewRecorder()

			healthHandler(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}
