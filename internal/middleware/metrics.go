package middleware

import (
	"database/sql"
	"log"
	"net/http"
	"regexp"
	"runtime"
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

	// Tenant-aware metrics (tenant_id|method|path|status -> count)
	tenantRequestCounts map[string]int64

	// Tenant request durations
	tenantDurations map[string]*durationStats

	// Active requests per tenant
	tenantActiveRequests map[int]int64

	// Business metrics (refreshed periodically from database)
	db              *sql.DB
	totalBookings   int64
	totalDogs       int64
	totalTenants    int64
	activeTenants   int64
	activeUsers     int64 // users active in last 30 days
	successLogins   int64
	failedLogins    int64
}

type durationStats struct {
	sum   float64
	count int64
	min   float64
	max   float64
}

// Global metrics collector instance
var Metrics = &MetricsCollector{
	requestCounts:        make(map[string]int64),
	requestDurations:     make(map[string]*durationStats),
	errorCounts:          make(map[string]int64),
	requestSizes:         make(map[string]int64),
	startTime:            time.Now(),
	tenantRequestCounts:  make(map[string]int64),
	tenantDurations:      make(map[string]*durationStats),
	tenantActiveRequests: make(map[int]int64),
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

		// Get tenant ID from context (may be 0 if no tenant)
		tenantID := GetTenantID(r)

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

		// Track tenant-specific metrics (only if tenant is present)
		if tenantID > 0 {
			tenantKey := strconv.Itoa(tenantID) + "|" + r.Method + "|" + path + "|" + strconv.Itoa(wrapped.statusCode)
			Metrics.tenantRequestCounts[tenantKey]++

			// Update tenant duration stats
			if Metrics.tenantDurations[tenantKey] == nil {
				Metrics.tenantDurations[tenantKey] = &durationStats{min: duration, max: duration}
			}
			tenantStats := Metrics.tenantDurations[tenantKey]
			tenantStats.sum += duration
			tenantStats.count++
			if duration < tenantStats.min {
				tenantStats.min = duration
			}
			if duration > tenantStats.max {
				tenantStats.max = duration
			}
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

	// Tenant-specific metrics
	if len(m.tenantRequestCounts) > 0 {
		sb.WriteString("\n# HELP http_requests_by_tenant_total Total HTTP requests per tenant\n")
		sb.WriteString("# TYPE http_requests_by_tenant_total counter\n")

		for key, count := range m.tenantRequestCounts {
			parts := strings.Split(key, "|")
			if len(parts) != 4 {
				continue
			}
			tenantID, method, path, status := parts[0], parts[1], parts[2], parts[3]
			sb.WriteString("http_requests_by_tenant_total{tenant_id=\"" + tenantID + "\",method=\"" + method + "\",path=\"" + path + "\",status=\"" + status + "\"} " + strconv.FormatInt(count, 10) + "\n")
		}

		sb.WriteString("\n# HELP http_request_duration_by_tenant_seconds HTTP request duration per tenant in seconds\n")
		sb.WriteString("# TYPE http_request_duration_by_tenant_seconds summary\n")

		for key, stats := range m.tenantDurations {
			parts := strings.Split(key, "|")
			if len(parts) != 4 {
				continue
			}
			tenantID, method, path := parts[0], parts[1], parts[2]

			if stats.count > 0 {
				avg := stats.sum / float64(stats.count)
				sb.WriteString("http_request_duration_by_tenant_seconds{tenant_id=\"" + tenantID + "\",method=\"" + method + "\",path=\"" + path + "\",quantile=\"0.5\"} " + strconv.FormatFloat(avg, 'f', 6, 64) + "\n")
				sb.WriteString("http_request_duration_by_tenant_seconds{tenant_id=\"" + tenantID + "\",method=\"" + method + "\",path=\"" + path + "\",quantile=\"min\"} " + strconv.FormatFloat(stats.min, 'f', 6, 64) + "\n")
				sb.WriteString("http_request_duration_by_tenant_seconds{tenant_id=\"" + tenantID + "\",method=\"" + method + "\",path=\"" + path + "\",quantile=\"max\"} " + strconv.FormatFloat(stats.max, 'f', 6, 64) + "\n")
			}
		}
	}

	// Business metrics (from database)
	sb.WriteString("\n# HELP gassigeher_bookings_total Total number of bookings\n")
	sb.WriteString("# TYPE gassigeher_bookings_total gauge\n")
	sb.WriteString("gassigeher_bookings_total " + strconv.FormatInt(m.totalBookings, 10) + "\n")

	sb.WriteString("\n# HELP gassigeher_dogs_total Total number of dogs\n")
	sb.WriteString("# TYPE gassigeher_dogs_total gauge\n")
	sb.WriteString("gassigeher_dogs_total " + strconv.FormatInt(m.totalDogs, 10) + "\n")

	sb.WriteString("\n# HELP gassigeher_tenants_total Total number of tenants\n")
	sb.WriteString("# TYPE gassigeher_tenants_total gauge\n")
	sb.WriteString("gassigeher_tenants_total " + strconv.FormatInt(m.totalTenants, 10) + "\n")

	sb.WriteString("\n# HELP gassigeher_tenants_active Active tenants\n")
	sb.WriteString("# TYPE gassigeher_tenants_active gauge\n")
	sb.WriteString("gassigeher_tenants_active " + strconv.FormatInt(m.activeTenants, 10) + "\n")

	sb.WriteString("\n# HELP gassigeher_users_active Users active in last 30 days\n")
	sb.WriteString("# TYPE gassigeher_users_active gauge\n")
	sb.WriteString("gassigeher_users_active " + strconv.FormatInt(m.activeUsers, 10) + "\n")

	sb.WriteString("\n# HELP gassigeher_logins_success_total Successful login attempts\n")
	sb.WriteString("# TYPE gassigeher_logins_success_total counter\n")
	sb.WriteString("gassigeher_logins_success_total " + strconv.FormatInt(m.successLogins, 10) + "\n")

	sb.WriteString("\n# HELP gassigeher_logins_failed_total Failed login attempts\n")
	sb.WriteString("# TYPE gassigeher_logins_failed_total counter\n")
	sb.WriteString("gassigeher_logins_failed_total " + strconv.FormatInt(m.failedLogins, 10) + "\n")

	// Go runtime metrics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	sb.WriteString("\n# HELP go_goroutines Number of goroutines\n")
	sb.WriteString("# TYPE go_goroutines gauge\n")
	sb.WriteString("go_goroutines " + strconv.Itoa(runtime.NumGoroutine()) + "\n")

	sb.WriteString("\n# HELP go_memory_alloc_bytes Allocated memory in bytes\n")
	sb.WriteString("# TYPE go_memory_alloc_bytes gauge\n")
	sb.WriteString("go_memory_alloc_bytes " + strconv.FormatUint(memStats.Alloc, 10) + "\n")

	sb.WriteString("\n# HELP go_memory_sys_bytes Total memory from OS in bytes\n")
	sb.WriteString("# TYPE go_memory_sys_bytes gauge\n")
	sb.WriteString("go_memory_sys_bytes " + strconv.FormatUint(memStats.Sys, 10) + "\n")

	sb.WriteString("\n# HELP go_gc_runs_total Total garbage collection runs\n")
	sb.WriteString("# TYPE go_gc_runs_total counter\n")
	sb.WriteString("go_gc_runs_total " + strconv.FormatUint(uint64(memStats.NumGC), 10) + "\n")

	return sb.String()
}

// InitBusinessMetrics initializes database-backed business metrics
// and starts a background goroutine to refresh them periodically
func InitBusinessMetrics(db *sql.DB) {
	Metrics.mu.Lock()
	Metrics.db = db
	Metrics.mu.Unlock()

	// Initial refresh
	Metrics.refreshBusinessMetrics()

	// Start periodic refresh (every 5 minutes)
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			Metrics.refreshBusinessMetrics()
		}
	}()

	// Start periodic cleanup (every hour) to prevent unbounded memory growth
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			Metrics.cleanup()
		}
	}()

	log.Println("Business metrics initialized (refresh every 5 minutes, cleanup every hour)")
}

// cleanup removes old entries from metrics maps to prevent unbounded memory growth
// It keeps only the top 1000 entries by count for each map
func (m *MetricsCollector) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	const maxEntries = 1000

	// Cleanup requestCounts - keep top entries by count
	if len(m.requestCounts) > maxEntries {
		m.requestCounts = trimMap(m.requestCounts, maxEntries)
		log.Printf("Metrics cleanup: trimmed requestCounts to %d entries", len(m.requestCounts))
	}

	// Cleanup requestDurations
	if len(m.requestDurations) > maxEntries {
		// Keep entries that exist in requestCounts
		newDurations := make(map[string]*durationStats)
		for key, stats := range m.requestDurations {
			if _, exists := m.requestCounts[key]; exists {
				newDurations[key] = stats
			}
		}
		m.requestDurations = newDurations
		log.Printf("Metrics cleanup: trimmed requestDurations to %d entries", len(m.requestDurations))
	}

	// Cleanup tenantRequestCounts
	if len(m.tenantRequestCounts) > maxEntries {
		m.tenantRequestCounts = trimMap(m.tenantRequestCounts, maxEntries)
		log.Printf("Metrics cleanup: trimmed tenantRequestCounts to %d entries", len(m.tenantRequestCounts))
	}

	// Cleanup tenantDurations
	if len(m.tenantDurations) > maxEntries {
		newDurations := make(map[string]*durationStats)
		for key, stats := range m.tenantDurations {
			if _, exists := m.tenantRequestCounts[key]; exists {
				newDurations[key] = stats
			}
		}
		m.tenantDurations = newDurations
		log.Printf("Metrics cleanup: trimmed tenantDurations to %d entries", len(m.tenantDurations))
	}
}

// trimMap keeps only the top N entries by value (count)
// When counts are equal, keys are sorted alphabetically for deterministic results
func trimMap(m map[string]int64, maxEntries int) map[string]int64 {
	if len(m) <= maxEntries {
		return m
	}

	// Create slice of key-count pairs for sorting
	type kv struct {
		key   string
		count int64
	}
	pairs := make([]kv, 0, len(m))
	for k, v := range m {
		pairs = append(pairs, kv{k, v})
	}

	// Sort by count descending, then by key ascending for determinism
	for i := 0; i < len(pairs); i++ {
		for j := i + 1; j < len(pairs); j++ {
			// Sort by count descending
			if pairs[i].count < pairs[j].count {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			} else if pairs[i].count == pairs[j].count && pairs[i].key > pairs[j].key {
				// When counts are equal, sort by key ascending for determinism
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}

	// Keep top N entries
	newMap := make(map[string]int64)
	for i := 0; i < maxEntries && i < len(pairs); i++ {
		newMap[pairs[i].key] = pairs[i].count
	}

	return newMap
}

// refreshBusinessMetrics fetches counts from the database
func (m *MetricsCollector) refreshBusinessMetrics() {
	if m.db == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Total bookings
	var bookings int64
	if err := m.db.QueryRow("SELECT COUNT(*) FROM bookings").Scan(&bookings); err == nil {
		m.totalBookings = bookings
	}

	// Total dogs
	var dogs int64
	if err := m.db.QueryRow("SELECT COUNT(*) FROM dogs").Scan(&dogs); err == nil {
		m.totalDogs = dogs
	}

	// Total tenants (if SaaS mode - table may not exist in simple mode)
	var tenants int64
	if err := m.db.QueryRow("SELECT COUNT(*) FROM tenants").Scan(&tenants); err == nil {
		m.totalTenants = tenants
	}

	// Active tenants (status = 'active')
	var activeTenants int64
	if err := m.db.QueryRow("SELECT COUNT(*) FROM tenants WHERE status = 'active'").Scan(&activeTenants); err == nil {
		m.activeTenants = activeTenants
	}

	// Active users (logged in last 30 days)
	var activeUsers int64
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30).Format("2006-01-02 15:04:05")
	if err := m.db.QueryRow("SELECT COUNT(*) FROM users WHERE last_activity_at > ?", thirtyDaysAgo).Scan(&activeUsers); err == nil {
		m.activeUsers = activeUsers
	}
}

// RecordLogin records a login attempt (success or failure)
func RecordLogin(success bool) {
	Metrics.mu.Lock()
	defer Metrics.mu.Unlock()

	if success {
		Metrics.successLogins++
	} else {
		Metrics.failedLogins++
	}
}
