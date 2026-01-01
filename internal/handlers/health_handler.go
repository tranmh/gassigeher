package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/tranmh/gassigeher/internal/database"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	db *database.DB
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *database.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// HealthCheck represents a single health check result
type HealthCheck struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Latency string `json:"latency,omitempty"`
}

// HealthResponse represents the full health response
type HealthResponse struct {
	Status    string                 `json:"status"`
	Checks    map[string]HealthCheck `json:"checks"`
	Timestamp string                 `json:"timestamp"`
}

// Health returns basic liveness status (is the process running?)
// GET /api/health
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Ready returns readiness status (can we serve traffic?)
// GET /api/ready
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "ok",
		Checks:    make(map[string]HealthCheck),
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Check database connectivity
	if h.db != nil {
		start := time.Now()
		err := h.db.Ping()
		latency := time.Since(start)

		if err != nil {
			response.Status = "degraded"
			response.Checks["database"] = HealthCheck{
				Status:  "fail",
				Message: "Database connection failed", // Sanitized - don't leak internal error details
			}
		} else {
			response.Checks["database"] = HealthCheck{
				Status:  "ok",
				Latency: latency.String(),
			}
		}

		// Check if we can actually query
		start = time.Now()
		var count int
		err = h.db.QueryRow("SELECT 1").Scan(&count)
		latency = time.Since(start)

		if err != nil {
			response.Status = "degraded"
			response.Checks["database_query"] = HealthCheck{
				Status:  "fail",
				Message: "Database query failed", // Sanitized - don't leak internal error details
			}
		} else {
			response.Checks["database_query"] = HealthCheck{
				Status:  "ok",
				Latency: latency.String(),
			}
		}
	} else {
		response.Status = "degraded"
		response.Checks["database"] = HealthCheck{
			Status:  "fail",
			Message: "Database not configured",
		}
	}

	// Set appropriate status code
	statusCode := http.StatusOK
	if response.Status != "ok" {
		statusCode = http.StatusServiceUnavailable
	}

	respondJSON(w, statusCode, response)
}

// DetailedHealth returns detailed health information including system stats
// GET /api/health/detailed
func (h *HealthHandler) DetailedHealth(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "ok",
		Checks:    make(map[string]HealthCheck),
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Database checks
	if h.db != nil {
		// Ping check
		start := time.Now()
		err := h.db.Ping()
		latency := time.Since(start)

		if err != nil {
			response.Status = "degraded"
			response.Checks["database_ping"] = HealthCheck{
				Status:  "fail",
				Message: "Database ping failed", // Sanitized - don't leak internal error details
			}
		} else {
			response.Checks["database_ping"] = HealthCheck{
				Status:  "ok",
				Latency: latency.String(),
			}
		}

		// Connection pool stats
		stats := h.db.Stats()
		response.Checks["database_pool"] = HealthCheck{
			Status:  "ok",
			Message: formatPoolStats(stats),
		}

		// Table count check (verify schema is accessible)
		start = time.Now()
		var tableCount int
		err = h.db.QueryRow(`
			SELECT COUNT(*) FROM sqlite_master WHERE type='table'
		`).Scan(&tableCount)
		latency = time.Since(start)

		if err != nil {
			// Try MySQL/PostgreSQL syntax
			err = h.db.QueryRow(`
				SELECT COUNT(*) FROM information_schema.tables
				WHERE table_schema = DATABASE()
			`).Scan(&tableCount)
		}

		if err != nil {
			response.Checks["database_schema"] = HealthCheck{
				Status:  "warn",
				Message: "Could not verify schema",
			}
		} else {
			response.Checks["database_schema"] = HealthCheck{
				Status:  "ok",
				Message: formatTableCount(tableCount),
				Latency: latency.String(),
			}
		}
	}

	statusCode := http.StatusOK
	if response.Status != "ok" {
		statusCode = http.StatusServiceUnavailable
	}

	respondJSON(w, statusCode, response)
}

func formatPoolStats(stats sql.DBStats) string {
	return "open=" + itoa(stats.OpenConnections) +
		" in_use=" + itoa(stats.InUse) +
		" idle=" + itoa(stats.Idle) +
		" max_open=" + itoa(stats.MaxOpenConnections)
}

func formatTableCount(count int) string {
	return itoa(count) + " tables accessible"
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	s := ""
	for i > 0 {
		s = string(rune('0'+i%10)) + s
		i /= 10
	}
	return s
}
