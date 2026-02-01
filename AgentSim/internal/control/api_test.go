package control

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
)

func setupTestAPI(running bool) (*API, *mux.Router) {
	logger := zerolog.Nop()
	api := NewAPI(logger)

	api.SetHandlers(
		func(count int) error { return nil },
		func() error { return nil },
		func(count int) error { return nil },
		func() map[string]interface{} { return map[string]interface{}{"calls": 0} },
		func() map[string]interface{} { return map[string]interface{}{"sim_running": false} },
	)

	if running {
		api.mu.Lock()
		api.status.Running = true
		api.mu.Unlock()
	}

	router := mux.NewRouter()
	api.SetupRoutes(router)
	return api, router
}

func TestHealthHandler(t *testing.T) {
	_, router := setupTestAPI(false)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["status"] != "healthy" {
		t.Fatalf("expected status healthy, got %s", body["status"])
	}
}

func TestStatusHandler(t *testing.T) {
	_, router := setupTestAPI(false)

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)
	if body["running"] != false {
		t.Fatalf("expected running=false, got %v", body["running"])
	}
}

func TestStartHandler(t *testing.T) {
	_, router := setupTestAPI(false)

	payload := `{"activeAgents": 50}`
	req := httptest.NewRequest(http.MethodPost, "/start", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)
	if body["active_agents"] != float64(50) {
		t.Fatalf("expected active_agents=50, got %v", body["active_agents"])
	}
}

func TestStartHandler_AlreadyRunning(t *testing.T) {
	_, router := setupTestAPI(true)

	payload := `{"activeAgents": 50}`
	req := httptest.NewRequest(http.MethodPost, "/start", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestStopHandler(t *testing.T) {
	_, router := setupTestAPI(true)

	req := httptest.NewRequest(http.MethodPost, "/stop", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestStopHandler_NotRunning(t *testing.T) {
	_, router := setupTestAPI(false)

	req := httptest.NewRequest(http.MethodPost, "/stop", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestScaleHandler(t *testing.T) {
	_, router := setupTestAPI(false)

	payload := `{"activeAgents": 500}`
	req := httptest.NewRequest(http.MethodPost, "/scale", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)
	if body["active_agents"] != float64(500) {
		t.Fatalf("expected active_agents=500, got %v", body["active_agents"])
	}
}

func TestScaleHandler_InvalidCount(t *testing.T) {
	_, router := setupTestAPI(false)

	payload := `{"activeAgents": 99999}`
	req := httptest.NewRequest(http.MethodPost, "/scale", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
