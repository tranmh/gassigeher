package services

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
)

// AuditService handles business event audit logging
type AuditService struct {
	db *sql.DB
}

// NewAuditService creates a new audit service
func NewAuditService(db *sql.DB) *AuditService {
	return &AuditService{db: db}
}

// Log records an audit event
func (s *AuditService) Log(r *http.Request, tenantID int, userID *int, action, entityType string, entityID *int, oldValue, newValue interface{}) {
	go s.logAsync(r, tenantID, userID, action, entityType, entityID, oldValue, newValue)
}

// logAsync performs the actual logging asynchronously
func (s *AuditService) logAsync(r *http.Request, tenantID int, userID *int, action, entityType string, entityID *int, oldValue, newValue interface{}) {
	var oldJSON, newJSON *string

	if oldValue != nil {
		b, err := json.Marshal(oldValue)
		if err == nil {
			str := string(b)
			oldJSON = &str
		}
	}
	if newValue != nil {
		b, err := json.Marshal(newValue)
		if err == nil {
			str := string(b)
			newJSON = &str
		}
	}

	ipAddress := getClientIP(r)
	userAgent := ""
	if r != nil {
		userAgent = r.UserAgent()
	}

	_, err := s.db.Exec(`
		INSERT INTO audit_logs (tenant_id, user_id, action, entity_type, entity_id,
		                        old_value, new_value, ip_address, user_agent, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		tenantID, userID, action, entityType, entityID,
		oldJSON, newJSON, ipAddress, userAgent, time.Now(),
	)

	if err != nil {
		log.Printf("AUDIT ERROR: Failed to log audit event: %v (action=%s, tenant=%d)", err, action, tenantID)
	}
}

// LogSimple records a simple audit event without old/new values
func (s *AuditService) LogSimple(r *http.Request, tenantID int, userID *int, action, entityType string, entityID *int) {
	s.Log(r, tenantID, userID, action, entityType, entityID, nil, nil)
}

// LogWithMessage records an audit event with a simple message as new_value
func (s *AuditService) LogWithMessage(r *http.Request, tenantID int, userID *int, action, entityType string, entityID *int, message string) {
	s.Log(r, tenantID, userID, action, entityType, entityID, nil, map[string]string{"message": message})
}

// Query retrieves audit logs based on filter criteria
func (s *AuditService) Query(filter *models.AuditLogFilter) ([]*models.AuditLog, error) {
	query := `
		SELECT id, tenant_id, user_id, action, entity_type, entity_id,
		       old_value, new_value, ip_address, user_agent, created_at
		FROM audit_logs
		WHERE 1=1
	`
	var args []interface{}

	if filter.TenantID != nil {
		query += " AND tenant_id = ?"
		args = append(args, *filter.TenantID)
	}
	if filter.UserID != nil {
		query += " AND user_id = ?"
		args = append(args, *filter.UserID)
	}
	if filter.Action != nil {
		query += " AND action = ?"
		args = append(args, *filter.Action)
	}
	if filter.EntityType != nil {
		query += " AND entity_type = ?"
		args = append(args, *filter.EntityType)
	}
	if filter.EntityID != nil {
		query += " AND entity_id = ?"
		args = append(args, *filter.EntityID)
	}
	if filter.StartDate != nil {
		query += " AND created_at >= ?"
		args = append(args, *filter.StartDate)
	}
	if filter.EndDate != nil {
		query += " AND created_at <= ?"
		args = append(args, *filter.EndDate)
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	} else {
		query += " LIMIT 100" // Default limit
	}

	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.AuditLog
	for rows.Next() {
		var l models.AuditLog
		err := rows.Scan(
			&l.ID, &l.TenantID, &l.UserID, &l.Action, &l.EntityType, &l.EntityID,
			&l.OldValue, &l.NewValue, &l.IPAddress, &l.UserAgent, &l.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, &l)
	}

	return logs, nil
}

// Count returns the total count of audit logs matching the filter
func (s *AuditService) Count(filter *models.AuditLogFilter) (int, error) {
	query := "SELECT COUNT(*) FROM audit_logs WHERE 1=1"
	var args []interface{}

	if filter.TenantID != nil {
		query += " AND tenant_id = ?"
		args = append(args, *filter.TenantID)
	}
	if filter.UserID != nil {
		query += " AND user_id = ?"
		args = append(args, *filter.UserID)
	}
	if filter.Action != nil {
		query += " AND action = ?"
		args = append(args, *filter.Action)
	}
	if filter.EntityType != nil {
		query += " AND entity_type = ?"
		args = append(args, *filter.EntityType)
	}
	if filter.StartDate != nil {
		query += " AND created_at >= ?"
		args = append(args, *filter.StartDate)
	}
	if filter.EndDate != nil {
		query += " AND created_at <= ?"
		args = append(args, *filter.EndDate)
	}

	var count int
	err := s.db.QueryRow(query, args...).Scan(&count)
	return count, err
}

// getClientIP extracts the client IP address from a request
func getClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}

	// Check X-Forwarded-For header (from reverse proxy)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP (client IP)
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	if r.RemoteAddr != "" {
		// Strip port if present
		if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
			return r.RemoteAddr[:idx]
		}
		return r.RemoteAddr
	}

	return ""
}
