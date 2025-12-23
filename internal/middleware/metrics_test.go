package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestMetricsCollectorInitialization tests that the global metrics collector is initialized
func TestMetricsCollectorInitialization(t *testing.T) {
	if Metrics == nil {
		t.Fatal("Global Metrics collector is nil")
	}

	if Metrics.requestCounts == nil {
		t.Error("requestCounts map is nil")
	}
	if Metrics.requestDurations == nil {
		t.Error("requestDurations map is nil")
	}
	if Metrics.errorCounts == nil {
		t.Error("errorCounts map is nil")
	}
	if Metrics.tenantRequestCounts == nil {
		t.Error("tenantRequestCounts map is nil")
	}
	if Metrics.tenantDurations == nil {
		t.Error("tenantDurations map is nil")
	}
	if Metrics.tenantActiveRequests == nil {
		t.Error("tenantActiveRequests map is nil")
	}
}

// TestNormalizePath tests path normalization
func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/api/v1/users/123", "/api/v1/users/:id"},
		{"/api/v1/dogs/456/photo", "/api/v1/dogs/:id/photo"},
		{"/api/v1/bookings/789/cancel", "/api/v1/bookings/:id/cancel"},
		{"/api/v1/users", "/api/v1/users"},
		{"/api/health", "/api/health"},
		{"/", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizePath(tt.input)
			if result != tt.expected {
				t.Errorf("normalizePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestMetricsMiddleware tests basic request tracking
func TestMetricsMiddleware(t *testing.T) {
	// Reset metrics for clean test
	Metrics.mu.Lock()
	Metrics.requestCounts = make(map[string]int64)
	Metrics.requestDurations = make(map[string]*durationStats)
	Metrics.errorCounts = make(map[string]int64)
	Metrics.tenantRequestCounts = make(map[string]int64)
	Metrics.tenantDurations = make(map[string]*durationStats)
	Metrics.mu.Unlock()

	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with metrics middleware
	wrapped := MetricsMiddleware(handler)

	// Make a test request
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Verify request was counted
	Metrics.mu.RLock()
	defer Metrics.mu.RUnlock()

	key := "GET|/api/v1/health|200"
	if count, ok := Metrics.requestCounts[key]; !ok || count != 1 {
		t.Errorf("Expected requestCounts[%q] = 1, got %d", key, count)
	}

	// Verify duration was recorded
	if stats, ok := Metrics.requestDurations[key]; !ok || stats == nil {
		t.Errorf("Expected duration stats for %q", key)
	}
}

// TestMetricsMiddlewareErrorTracking tests error counting
func TestMetricsMiddlewareErrorTracking(t *testing.T) {
	// Reset metrics
	Metrics.mu.Lock()
	Metrics.requestCounts = make(map[string]int64)
	Metrics.errorCounts = make(map[string]int64)
	Metrics.requestDurations = make(map[string]*durationStats)
	Metrics.tenantRequestCounts = make(map[string]int64)
	Metrics.tenantDurations = make(map[string]*durationStats)
	Metrics.mu.Unlock()

	// Create handler that returns 404
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	wrapped := MetricsMiddleware(handler)

	req := httptest.NewRequest("GET", "/api/v1/notfound", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	Metrics.mu.RLock()
	defer Metrics.mu.RUnlock()

	// Verify error was counted
	if count, ok := Metrics.errorCounts["404"]; !ok || count != 1 {
		t.Errorf("Expected errorCounts[404] = 1, got %d", count)
	}
}

// TestGetMetrics tests the JSON metrics response
func TestGetMetrics(t *testing.T) {
	// Reset metrics
	Metrics.mu.Lock()
	Metrics.requestCounts = make(map[string]int64)
	Metrics.requestDurations = make(map[string]*durationStats)
	Metrics.errorCounts = make(map[string]int64)
	Metrics.tenantRequestCounts = make(map[string]int64)
	Metrics.tenantDurations = make(map[string]*durationStats)

	// Add some test data
	Metrics.requestCounts["GET|/api/health|200"] = 10
	Metrics.requestCounts["POST|/api/login|200"] = 5
	Metrics.requestCounts["GET|/api/users|401"] = 2
	Metrics.errorCounts["401"] = 2
	Metrics.requestDurations["GET|/api/health|200"] = &durationStats{
		sum: 0.5, count: 10, min: 0.01, max: 0.1,
	}
	Metrics.mu.Unlock()

	response := Metrics.GetMetrics()

	if response.TotalRequests != 17 {
		t.Errorf("Expected TotalRequests = 17, got %d", response.TotalRequests)
	}

	if response.RequestsByMethod["GET"] != 12 {
		t.Errorf("Expected GET requests = 12, got %d", response.RequestsByMethod["GET"])
	}

	if response.RequestsByStatus["200"] != 15 {
		t.Errorf("Expected 200 status = 15, got %d", response.RequestsByStatus["200"])
	}

	if response.ErrorCounts["401"] != 2 {
		t.Errorf("Expected 401 errors = 2, got %d", response.ErrorCounts["401"])
	}
}

// TestGetPrometheusMetrics tests the Prometheus format output
func TestGetPrometheusMetrics(t *testing.T) {
	// Reset metrics
	Metrics.mu.Lock()
	Metrics.requestCounts = make(map[string]int64)
	Metrics.requestDurations = make(map[string]*durationStats)
	Metrics.errorCounts = make(map[string]int64)
	Metrics.tenantRequestCounts = make(map[string]int64)
	Metrics.tenantDurations = make(map[string]*durationStats)

	// Add some test data
	Metrics.requestCounts["GET|/api/health|200"] = 10
	Metrics.requestDurations["GET|/api/health|200"] = &durationStats{
		sum: 0.5, count: 10, min: 0.01, max: 0.1,
	}
	Metrics.mu.Unlock()

	output := Metrics.GetPrometheusMetrics()

	// Verify contains expected metrics
	if !strings.Contains(output, "# HELP http_requests_total") {
		t.Error("Missing http_requests_total HELP")
	}
	if !strings.Contains(output, "# TYPE http_requests_total counter") {
		t.Error("Missing http_requests_total TYPE")
	}
	if !strings.Contains(output, `http_requests_total{method="GET",path="/api/health",status="200"} 10`) {
		t.Error("Missing http_requests_total metric line")
	}
	if !strings.Contains(output, "http_active_connections") {
		t.Error("Missing http_active_connections metric")
	}
	if !strings.Contains(output, "process_uptime_seconds") {
		t.Error("Missing process_uptime_seconds metric")
	}
}

// TestTenantMetrics tests tenant-specific metrics
func TestTenantMetrics(t *testing.T) {
	// Reset metrics
	Metrics.mu.Lock()
	Metrics.requestCounts = make(map[string]int64)
	Metrics.requestDurations = make(map[string]*durationStats)
	Metrics.errorCounts = make(map[string]int64)
	Metrics.tenantRequestCounts = make(map[string]int64)
	Metrics.tenantDurations = make(map[string]*durationStats)

	// Add tenant-specific test data
	Metrics.tenantRequestCounts["1|GET|/api/dogs|200"] = 50
	Metrics.tenantRequestCounts["2|GET|/api/dogs|200"] = 30
	Metrics.tenantDurations["1|GET|/api/dogs|200"] = &durationStats{
		sum: 2.5, count: 50, min: 0.01, max: 0.2,
	}
	Metrics.mu.Unlock()

	output := Metrics.GetPrometheusMetrics()

	// Verify contains tenant metrics
	if !strings.Contains(output, "# HELP http_requests_by_tenant_total") {
		t.Error("Missing http_requests_by_tenant_total HELP")
	}
	if !strings.Contains(output, `tenant_id="1"`) {
		t.Error("Missing tenant_id=1 in metrics")
	}
	if !strings.Contains(output, `tenant_id="2"`) {
		t.Error("Missing tenant_id=2 in metrics")
	}
	if !strings.Contains(output, "http_request_duration_by_tenant_seconds") {
		t.Error("Missing http_request_duration_by_tenant_seconds metric")
	}
}

// TestMetricsResponseWriter tests the response writer wrapper
func TestMetricsResponseWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	wrapped := &metricsResponseWriter{ResponseWriter: rec, statusCode: http.StatusOK}

	// Test default status code
	if wrapped.statusCode != http.StatusOK {
		t.Errorf("Default status code should be 200, got %d", wrapped.statusCode)
	}

	// Test WriteHeader
	wrapped.WriteHeader(http.StatusNotFound)
	if wrapped.statusCode != http.StatusNotFound {
		t.Errorf("Status code should be 404 after WriteHeader, got %d", wrapped.statusCode)
	}

	// Verify underlying recorder also got the status
	if rec.Code != http.StatusNotFound {
		t.Errorf("Underlying recorder should have 404, got %d", rec.Code)
	}
}

// TestDurationStats tests duration statistics tracking
func TestDurationStats(t *testing.T) {
	// Reset metrics
	Metrics.mu.Lock()
	Metrics.requestCounts = make(map[string]int64)
	Metrics.requestDurations = make(map[string]*durationStats)
	Metrics.tenantRequestCounts = make(map[string]int64)
	Metrics.tenantDurations = make(map[string]*durationStats)
	Metrics.errorCounts = make(map[string]int64)
	Metrics.mu.Unlock()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := MetricsMiddleware(handler)

	// Make multiple requests
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)
	}

	Metrics.mu.RLock()
	defer Metrics.mu.RUnlock()

	key := "GET|/api/v1/test|200"
	stats, ok := Metrics.requestDurations[key]
	if !ok {
		t.Fatal("Duration stats not recorded")
	}

	if stats.count != 5 {
		t.Errorf("Expected count = 5, got %d", stats.count)
	}

	if stats.min <= 0 || stats.max <= 0 {
		t.Error("Min/max should be positive")
	}

	if stats.min > stats.max {
		t.Error("Min should be <= max")
	}

	if stats.sum <= 0 {
		t.Error("Sum should be positive")
	}
}
