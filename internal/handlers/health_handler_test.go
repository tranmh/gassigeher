package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHealthHandler_Health tests the health check endpoint
func TestHealthHandler_Health(t *testing.T) {
	handler := NewHealthHandler()

	t.Run("returns ok status", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/health", nil)
		rec := httptest.NewRecorder()

		handler.Health(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["status"] != "ok" {
			t.Errorf("Expected status 'ok', got '%s'", response["status"])
		}
	})

	t.Run("works with any HTTP method", func(t *testing.T) {
		methods := []string{"GET", "POST", "HEAD"}
		for _, method := range methods {
			req := httptest.NewRequest(method, "/api/health", nil)
			rec := httptest.NewRecorder()

			handler.Health(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("Expected status 200 for %s, got %d", method, rec.Code)
			}
		}
	})
}
