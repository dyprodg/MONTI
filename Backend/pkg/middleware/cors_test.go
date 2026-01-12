package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS(t *testing.T) {
	allowedOrigins := []string{"http://localhost:5173", "http://example.com"}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	corsHandler := CORS(allowedOrigins)(handler)

	tests := []struct {
		name           string
		origin         string
		method         string
		expectCORS     bool
		expectedOrigin string
	}{
		{
			name:           "allowed origin",
			origin:         "http://localhost:5173",
			method:         http.MethodGet,
			expectCORS:     true,
			expectedOrigin: "http://localhost:5173",
		},
		{
			name:           "another allowed origin",
			origin:         "http://example.com",
			method:         http.MethodGet,
			expectCORS:     true,
			expectedOrigin: "http://example.com",
		},
		{
			name:       "disallowed origin",
			origin:     "http://evil.com",
			method:     http.MethodGet,
			expectCORS: false,
		},
		{
			name:           "preflight request",
			origin:         "http://localhost:5173",
			method:         http.MethodOptions,
			expectCORS:     true,
			expectedOrigin: "http://localhost:5173",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			rec := httptest.NewRecorder()
			corsHandler.ServeHTTP(rec, req)

			if tt.expectCORS {
				acao := rec.Header().Get("Access-Control-Allow-Origin")
				if acao != tt.expectedOrigin {
					t.Errorf("expected Access-Control-Allow-Origin %s, got %s", tt.expectedOrigin, acao)
				}
			} else {
				acao := rec.Header().Get("Access-Control-Allow-Origin")
				if acao != "" {
					t.Errorf("expected no Access-Control-Allow-Origin header, got %s", acao)
				}
			}
		})
	}
}
