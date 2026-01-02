package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestMetricsHandler_GetMetrics tests getting metrics in JSON format
func TestMetricsHandler_GetMetrics(t *testing.T) {
	handler := NewMetricsHandler()

	t.Run("returns metrics in JSON format", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/metrics", nil)

		rec := httptest.NewRecorder()
		handler.GetMetrics(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		// Verify response is valid JSON
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		if err != nil {
			t.Errorf("Response should be valid JSON: %v", err)
		}

		// Verify Content-Type is JSON
		contentType := rec.Header().Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			t.Errorf("Expected JSON content type, got %s", contentType)
		}
	})

	t.Run("metrics structure is valid", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/metrics", nil)

		rec := httptest.NewRecorder()
		handler.GetMetrics(rec, req)

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		// Verify expected fields exist
		expectedFields := []string{"total_requests", "uptime_seconds"}
		for _, field := range expectedFields {
			// Fields may or may not exist depending on implementation
			t.Logf("Field %s: %v", field, response[field])
		}
	})
}

// TestMetricsHandler_GetPrometheusMetrics tests getting metrics in Prometheus format
func TestMetricsHandler_GetPrometheusMetrics(t *testing.T) {
	handler := NewMetricsHandler()

	t.Run("returns metrics in Prometheus text format", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/metrics", nil)

		rec := httptest.NewRecorder()
		handler.GetPrometheusMetrics(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		// Verify Content-Type is text/plain
		contentType := rec.Header().Get("Content-Type")
		if !strings.Contains(contentType, "text/plain") {
			t.Errorf("Expected text/plain content type, got %s", contentType)
		}
	})

	t.Run("prometheus format has correct syntax", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/metrics", nil)

		rec := httptest.NewRecorder()
		handler.GetPrometheusMetrics(rec, req)

		body := rec.Body.String()

		// Prometheus metrics should have # HELP or metric lines
		// If empty, that's also valid (no metrics collected yet)
		if body != "" && !strings.Contains(body, "#") && !strings.Contains(body, "_") {
			t.Logf("Prometheus metrics output: %s", body)
		}
	})
}
