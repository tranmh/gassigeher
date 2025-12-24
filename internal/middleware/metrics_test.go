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

// TestTrimMap tests the trimMap helper function
func TestTrimMap(t *testing.T) {
	tests := []struct {
		name       string
		input      map[string]int64
		maxEntries int
		wantLen    int
	}{
		{
			name:       "map smaller than max",
			input:      map[string]int64{"a": 1, "b": 2},
			maxEntries: 10,
			wantLen:    2,
		},
		{
			name:       "map equal to max",
			input:      map[string]int64{"a": 1, "b": 2, "c": 3},
			maxEntries: 3,
			wantLen:    3,
		},
		{
			name: "map larger than max - keeps highest counts",
			input: map[string]int64{
				"low1":  1,
				"low2":  2,
				"med":   5,
				"high1": 10,
				"high2": 20,
			},
			maxEntries: 3,
			wantLen:    3,
		},
		{
			name:       "empty map",
			input:      map[string]int64{},
			maxEntries: 10,
			wantLen:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimMap(tt.input, tt.maxEntries)
			if len(got) != tt.wantLen {
				t.Errorf("trimMap() len = %d, want %d", len(got), tt.wantLen)
			}

			// Verify that highest counts are preserved
			if tt.name == "map larger than max - keeps highest counts" {
				if _, ok := got["high2"]; !ok {
					t.Error("trimMap() should keep 'high2' (highest count)")
				}
				if _, ok := got["high1"]; !ok {
					t.Error("trimMap() should keep 'high1' (second highest)")
				}
				if _, ok := got["med"]; !ok {
					t.Error("trimMap() should keep 'med' (third highest)")
				}
			}
		})
	}
}

// TestMetricsCleanup tests the cleanup method
func TestMetricsCleanup(t *testing.T) {
	// Create a new collector for testing
	collector := &MetricsCollector{
		requestCounts:        make(map[string]int64),
		requestDurations:     make(map[string]*durationStats),
		errorCounts:          make(map[string]int64),
		requestSizes:         make(map[string]int64),
		tenantRequestCounts:  make(map[string]int64),
		tenantDurations:      make(map[string]*durationStats),
		tenantActiveRequests: make(map[int]int64),
	}

	// Add more than 1000 entries
	for i := 0; i < 1500; i++ {
		key := "GET|/api/test" + string(rune(i%26+'a')) + "|200"
		collector.requestCounts[key] = int64(i)
		collector.requestDurations[key] = &durationStats{sum: float64(i), count: 1, min: float64(i), max: float64(i)}
	}

	// Run cleanup
	collector.cleanup()

	// Should be trimmed to 1000
	if len(collector.requestCounts) > 1000 {
		t.Errorf("requestCounts len = %d, want <= 1000", len(collector.requestCounts))
	}
}

// TestMetricsCleanupTenantMetrics tests cleanup of tenant-specific metrics
func TestMetricsCleanupTenantMetrics(t *testing.T) {
	collector := &MetricsCollector{
		requestCounts:        make(map[string]int64),
		requestDurations:     make(map[string]*durationStats),
		errorCounts:          make(map[string]int64),
		requestSizes:         make(map[string]int64),
		tenantRequestCounts:  make(map[string]int64),
		tenantDurations:      make(map[string]*durationStats),
		tenantActiveRequests: make(map[int]int64),
	}

	// Add more than 1000 tenant entries
	for i := 0; i < 1500; i++ {
		key := "1|GET|/api/test" + string(rune(i%26+'a')) + "|200"
		collector.tenantRequestCounts[key] = int64(i)
		collector.tenantDurations[key] = &durationStats{sum: float64(i), count: 1, min: float64(i), max: float64(i)}
	}

	// Run cleanup
	collector.cleanup()

	// Should be trimmed to 1000
	if len(collector.tenantRequestCounts) > 1000 {
		t.Errorf("tenantRequestCounts len = %d, want <= 1000", len(collector.tenantRequestCounts))
	}
}

// TestMetricsCleanupSmallMaps tests that small maps are not affected by cleanup
func TestMetricsCleanupSmallMaps(t *testing.T) {
	collector := &MetricsCollector{
		requestCounts:        make(map[string]int64),
		requestDurations:     make(map[string]*durationStats),
		errorCounts:          make(map[string]int64),
		requestSizes:         make(map[string]int64),
		tenantRequestCounts:  make(map[string]int64),
		tenantDurations:      make(map[string]*durationStats),
		tenantActiveRequests: make(map[int]int64),
	}

	// Add fewer than 1000 entries
	for i := 0; i < 100; i++ {
		key := "GET|/api/test" + string(rune(i%26+'a')) + "|200"
		collector.requestCounts[key] = int64(i)
	}

	initialLen := len(collector.requestCounts)

	// Run cleanup
	collector.cleanup()

	// Should not change
	if len(collector.requestCounts) != initialLen {
		t.Errorf("requestCounts len = %d, want %d (unchanged)", len(collector.requestCounts), initialLen)
	}
}

// TestRecordLogin tests the login recording function
func TestRecordLogin(t *testing.T) {
	// Save original values
	Metrics.mu.Lock()
	originalSuccess := Metrics.successLogins
	originalFailed := Metrics.failedLogins
	Metrics.mu.Unlock()

	// Test success login
	RecordLogin(true)
	Metrics.mu.RLock()
	if Metrics.successLogins != originalSuccess+1 {
		t.Errorf("successLogins = %d, want %d", Metrics.successLogins, originalSuccess+1)
	}
	Metrics.mu.RUnlock()

	// Test failed login
	RecordLogin(false)
	Metrics.mu.RLock()
	if Metrics.failedLogins != originalFailed+1 {
		t.Errorf("failedLogins = %d, want %d", Metrics.failedLogins, originalFailed+1)
	}
	Metrics.mu.RUnlock()

	// Restore original values
	Metrics.mu.Lock()
	Metrics.successLogins = originalSuccess
	Metrics.failedLogins = originalFailed
	Metrics.mu.Unlock()
}

// TestLatencyStatsStruct tests the LatencyStats struct
func TestLatencyStatsStruct(t *testing.T) {
	stats := LatencyStats{
		Avg: 10.5,
		Min: 1.0,
		Max: 50.0,
	}

	if stats.Avg != 10.5 {
		t.Errorf("Avg = %f, want 10.5", stats.Avg)
	}
	if stats.Min != 1.0 {
		t.Errorf("Min = %f, want 1.0", stats.Min)
	}
	if stats.Max != 50.0 {
		t.Errorf("Max = %f, want 50.0", stats.Max)
	}
}

// BUG 1: trimMap returns non-deterministic results with duplicate counts
// RED PHASE: This test should FAIL until we fix the bug
func TestTrimMap_DeterministicWithDuplicateCounts(t *testing.T) {
	// All entries have the same count - result should be deterministic
	m := map[string]int64{
		"a": 10,
		"b": 10,
		"c": 10,
		"d": 10,
		"e": 10,
	}

	// Run multiple times and verify we always get the same result
	firstResult := trimMap(m, 3)
	firstKeys := make([]string, 0, len(firstResult))
	for k := range firstResult {
		firstKeys = append(firstKeys, k)
	}

	for i := 0; i < 10; i++ {
		result := trimMap(m, 3)
		if len(result) != 3 {
			t.Errorf("Run %d: trimMap() returned %d entries, want 3", i, len(result))
		}

		// Check that we get the same keys every time
		for k := range firstResult {
			if _, ok := result[k]; !ok {
				t.Errorf("Run %d: trimMap() returned different keys - non-deterministic behavior", i)
				break
			}
		}
	}
}

// TestDurationStatsMinMax tests updating min/max values
func TestDurationStatsMinMax(t *testing.T) {
	stats := &durationStats{
		sum:   0.5,
		count: 5,
		min:   0.05,
		max:   0.2,
	}

	// Verify initial state
	if stats.min != 0.05 {
		t.Errorf("min = %f, want 0.05", stats.min)
	}
	if stats.max != 0.2 {
		t.Errorf("max = %f, want 0.2", stats.max)
	}

	// Simulate update with new minimum
	newDuration := 0.01
	if newDuration < stats.min {
		stats.min = newDuration
	}
	if stats.min != 0.01 {
		t.Errorf("min after update = %f, want 0.01", stats.min)
	}

	// Simulate update with new maximum
	newDuration = 0.5
	if newDuration > stats.max {
		stats.max = newDuration
	}
	if stats.max != 0.5 {
		t.Errorf("max after update = %f, want 0.5", stats.max)
	}
}
