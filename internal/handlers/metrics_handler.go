package handlers

import (
	"net/http"

	"github.com/tranmh/gassigeher/internal/middleware"
)

// MetricsHandler handles metrics endpoints
type MetricsHandler struct{}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{}
}

// GetMetrics returns metrics in JSON format
// GET /api/metrics
func (h *MetricsHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := middleware.Metrics.GetMetrics()
	respondJSON(w, http.StatusOK, metrics)
}

// GetPrometheusMetrics returns metrics in Prometheus text format
// GET /metrics
func (h *MetricsHandler) GetPrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(middleware.Metrics.GetPrometheusMetrics()))
}
