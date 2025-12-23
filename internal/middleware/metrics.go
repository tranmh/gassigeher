package middleware

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// MetricsCollector collects HTTP metrics
type MetricsCollector struct {
	mu sync.RWMutex

	// Request counts by method, path, status
	requestCounts map[string]int64

	// Request durations (sum and count for average calculation)
	requestDurations map[string]*durationStats

	// Active connections
	activeConnections int64

	// Error counts by type
	errorCounts map[string]int64

	// Request sizes
	requestSizes map[string]int64

	// Start time for uptime calculation
	startTime time.Time
}

type durationStats struct {
	sum   float64
	count int64
	min   float64
	max   float64
}

// Global metrics collector instance
var Metrics = &MetricsCollector{
	requestCounts:    make(map[string]int64),
	requestDurations: make(map[string]*durationStats),
	errorCounts:      make(map[string]int64),
	requestSizes:     make(map[string]int64),
	startTime:        time.Now(),
}

// MetricsMiddleware collects request metrics
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		Metrics.mu.Lock()
		Metrics.activeConnections++
		Metrics.mu.Unlock()

		defer func() {
			Metrics.mu.Lock()
			Metrics.activeConnections--
			Metrics.mu.Unlock()
		}()

		// Wrap response writer to capture status code
		wrapped := &metricsResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()
		path := normalizePath(r.URL.Path)
		key := r.Method + "|" + path + "|" + strconv.Itoa(wrapped.statusCode)

		Metrics.mu.Lock()
		defer Metrics.mu.Unlock()

		// Increment request count
		Metrics.requestCounts[key]++

		// Update duration stats
		if Metrics.requestDurations[key] == nil {
			Metrics.requestDurations[key] = &durationStats{min: duration, max: duration}
		}
		stats := Metrics.requestDurations[key]
		stats.sum += duration
		stats.count++
		if duration < stats.min {
			stats.min = duration
		}
		if duration > stats.max {
			stats.max = duration
		}

		// Track errors
		if wrapped.statusCode >= 400 {
			errorKey := strconv.Itoa(wrapped.statusCode)
			Metrics.errorCounts[errorKey]++
		}
	})
}

type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *metricsResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Regex patterns for path normalization
var (
	idPattern   = regexp.MustCompile(`/\d+`)
	uuidPattern = regexp.MustCompile(`/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
)

// normalizePath replaces dynamic path segments with placeholders to reduce cardinality
func normalizePath(path string) string {
	// Replace numeric IDs with :id
	path = idPattern.ReplaceAllString(path, "/:id")

	// Replace UUIDs with :uuid
	path = uuidPattern.ReplaceAllString(path, "/:uuid")

	return path
}

// MetricsResponse represents the /metrics endpoint response
type MetricsResponse struct {
	Uptime            string                    `json:"uptime"`
	ActiveConnections int64                     `json:"active_connections"`
	TotalRequests     int64                     `json:"total_requests"`
	RequestsByStatus  map[string]int64          `json:"requests_by_status"`
	RequestsByMethod  map[string]int64          `json:"requests_by_method"`
	RequestsByPath    map[string]int64          `json:"requests_by_path"`
	ErrorCounts       map[string]int64          `json:"error_counts"`
	Latency           map[string]LatencyStats   `json:"latency"`
}

type LatencyStats struct {
	Avg float64 `json:"avg_ms"`
	Min float64 `json:"min_ms"`
	Max float64 `json:"max_ms"`
}

// GetMetrics returns current metrics
func (m *MetricsCollector) GetMetrics() *MetricsResponse {
	m.mu.RLock()
	defer m.mu.RUnlock()

	response := &MetricsResponse{
		Uptime:            time.Since(m.startTime).Round(time.Second).String(),
		ActiveConnections: m.activeConnections,
		RequestsByStatus:  make(map[string]int64),
		RequestsByMethod:  make(map[string]int64),
		RequestsByPath:    make(map[string]int64),
		ErrorCounts:       make(map[string]int64),
		Latency:           make(map[string]LatencyStats),
	}

	// Aggregate metrics
	for key, count := range m.requestCounts {
		parts := strings.Split(key, "|")
		if len(parts) != 3 {
			continue
		}
		method, path, status := parts[0], parts[1], parts[2]

		response.TotalRequests += count
		response.RequestsByStatus[status] += count
		response.RequestsByMethod[method] += count
		response.RequestsByPath[path] += count
	}

	// Copy error counts
	for k, v := range m.errorCounts {
		response.ErrorCounts[k] = v
	}

	// Calculate latency stats
	for key, stats := range m.requestDurations {
		parts := strings.Split(key, "|")
		if len(parts) != 3 {
			continue
		}
		path := parts[1]

		if stats.count > 0 {
			response.Latency[path] = LatencyStats{
				Avg: (stats.sum / float64(stats.count)) * 1000, // Convert to ms
				Min: stats.min * 1000,
				Max: stats.max * 1000,
			}
		}
	}

	return response
}

// GetPrometheusMetrics returns metrics in Prometheus text format
func (m *MetricsCollector) GetPrometheusMetrics() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sb strings.Builder

	// Help and type declarations
	sb.WriteString("# HELP http_requests_total Total number of HTTP requests\n")
	sb.WriteString("# TYPE http_requests_total counter\n")

	for key, count := range m.requestCounts {
		parts := strings.Split(key, "|")
		if len(parts) != 3 {
			continue
		}
		method, path, status := parts[0], parts[1], parts[2]
		sb.WriteString("http_requests_total{method=\"" + method + "\",path=\"" + path + "\",status=\"" + status + "\"} " + strconv.FormatInt(count, 10) + "\n")
	}

	sb.WriteString("\n# HELP http_request_duration_seconds HTTP request duration in seconds\n")
	sb.WriteString("# TYPE http_request_duration_seconds summary\n")

	for key, stats := range m.requestDurations {
		parts := strings.Split(key, "|")
		if len(parts) != 3 {
			continue
		}
		method, path := parts[0], parts[1]

		if stats.count > 0 {
			avg := stats.sum / float64(stats.count)
			sb.WriteString("http_request_duration_seconds{method=\"" + method + "\",path=\"" + path + "\",quantile=\"0.5\"} " + strconv.FormatFloat(avg, 'f', 6, 64) + "\n")
		}
	}

	sb.WriteString("\n# HELP http_active_connections Number of active HTTP connections\n")
	sb.WriteString("# TYPE http_active_connections gauge\n")
	sb.WriteString("http_active_connections " + strconv.FormatInt(m.activeConnections, 10) + "\n")

	sb.WriteString("\n# HELP process_uptime_seconds Process uptime in seconds\n")
	sb.WriteString("# TYPE process_uptime_seconds gauge\n")
	sb.WriteString("process_uptime_seconds " + strconv.FormatFloat(time.Since(m.startTime).Seconds(), 'f', 0, 64) + "\n")

	return sb.String()
}
