package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/tranmh/gassigeher/internal/database"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/services"
)

// AuditHandler handles audit log endpoints
type AuditHandler struct {
	auditService *services.AuditService
}

// NewAuditHandler creates a new audit handler
func NewAuditHandler(db *database.DB) *AuditHandler {
	return &AuditHandler{
		auditService: services.NewAuditService(db),
	}
}

// ListAuditLogs returns paginated audit logs for the tenant
// GET /api/admin/audit-logs
func (h *AuditHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.GetTenantID(r)

	filter := &models.AuditLogFilter{
		TenantID: &tenantID,
	}

	// Parse query parameters
	if action := r.URL.Query().Get("action"); action != "" {
		filter.Action = &action
	}
	if entityType := r.URL.Query().Get("entity_type"); entityType != "" {
		filter.EntityType = &entityType
	}
	if entityIDStr := r.URL.Query().Get("entity_id"); entityIDStr != "" {
		if entityID, err := strconv.Atoi(entityIDStr); err == nil {
			filter.EntityID = &entityID
		}
	}
	if userIDStr := r.URL.Query().Get("user_id"); userIDStr != "" {
		if userID, err := strconv.Atoi(userIDStr); err == nil {
			filter.UserID = &userID
		}
	}
	if startDateStr := r.URL.Query().Get("start_date"); startDateStr != "" {
		if startDate, err := time.Parse("2006-01-02", startDateStr); err == nil {
			filter.StartDate = &startDate
		}
	}
	if endDateStr := r.URL.Query().Get("end_date"); endDateStr != "" {
		if endDate, err := time.Parse("2006-01-02", endDateStr); err == nil {
			// Set to end of day
			endDate = endDate.Add(24*time.Hour - time.Second)
			filter.EndDate = &endDate
		}
	}

	// Pagination
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 500 {
			filter.Limit = limit
		}
	}
	if filter.Limit == 0 {
		filter.Limit = 50 // Default
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	// Get logs
	logs, err := h.auditService.Query(filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Audit-Logs")
		return
	}

	// Get total count for pagination
	total, _ := h.auditService.Count(filter)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"logs":   logs,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// GetAuditLogActions returns available audit log action types
// GET /api/admin/audit-logs/actions
func (h *AuditHandler) GetAuditLogActions(w http.ResponseWriter, r *http.Request) {
	actions := []map[string]string{
		{"value": models.AuditActionBookingCreated, "label": "Buchung erstellt"},
		{"value": models.AuditActionBookingCancelled, "label": "Buchung storniert"},
		{"value": models.AuditActionBookingApproved, "label": "Buchung genehmigt"},
		{"value": models.AuditActionBookingRejected, "label": "Buchung abgelehnt"},
		{"value": models.AuditActionBookingMoved, "label": "Buchung verschoben"},
		{"value": models.AuditActionUserCreated, "label": "Benutzer erstellt"},
		{"value": models.AuditActionUserUpdated, "label": "Benutzer aktualisiert"},
		{"value": models.AuditActionUserDeleted, "label": "Benutzer gelöscht"},
		{"value": models.AuditActionUserPromoted, "label": "Benutzer befördert"},
		{"value": models.AuditActionUserDemoted, "label": "Benutzer degradiert"},
		{"value": models.AuditActionUserActivated, "label": "Benutzer aktiviert"},
		{"value": models.AuditActionUserDeactivated, "label": "Benutzer deaktiviert"},
		{"value": models.AuditActionUserLogin, "label": "Benutzer angemeldet"},
		{"value": models.AuditActionUserImpersonated, "label": "Benutzer imitiert"},
		{"value": models.AuditActionDogCreated, "label": "Hund erstellt"},
		{"value": models.AuditActionDogUpdated, "label": "Hund aktualisiert"},
		{"value": models.AuditActionDogDeleted, "label": "Hund gelöscht"},
		{"value": models.AuditActionSettingsChanged, "label": "Einstellungen geändert"},
		{"value": models.AuditActionThemeChanged, "label": "Theme geändert"},
		{"value": models.AuditActionDataExported, "label": "Daten exportiert"},
		{"value": models.AuditActionExperienceRequested, "label": "Erfahrung angefragt"},
		{"value": models.AuditActionExperienceApproved, "label": "Erfahrung genehmigt"},
		{"value": models.AuditActionExperienceDenied, "label": "Erfahrung abgelehnt"},
		{"value": models.AuditActionColorRequested, "label": "Farbe angefragt"},
		{"value": models.AuditActionColorApproved, "label": "Farbe genehmigt"},
		{"value": models.AuditActionColorDenied, "label": "Farbe abgelehnt"},
	}

	respondJSON(w, http.StatusOK, actions)
}

// GetAuditLogEntityTypes returns available entity types
// GET /api/admin/audit-logs/entity-types
func (h *AuditHandler) GetAuditLogEntityTypes(w http.ResponseWriter, r *http.Request) {
	entityTypes := []map[string]string{
		{"value": models.EntityTypeBooking, "label": "Buchung"},
		{"value": models.EntityTypeUser, "label": "Benutzer"},
		{"value": models.EntityTypeDog, "label": "Hund"},
		{"value": models.EntityTypeSettings, "label": "Einstellungen"},
		{"value": models.EntityTypeTenant, "label": "Tierheim"},
		{"value": models.EntityTypeExperienceRequest, "label": "Erfahrungsanfrage"},
		{"value": models.EntityTypeColorRequest, "label": "Farbanfrage"},
		{"value": models.EntityTypeTheme, "label": "Theme"},
	}

	respondJSON(w, http.StatusOK, entityTypes)
}
