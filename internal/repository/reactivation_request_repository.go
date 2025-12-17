package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
)

// ReactivationRequestRepository handles reactivation request database operations
type ReactivationRequestRepository struct {
	db *sql.DB
}

// NewReactivationRequestRepository creates a new reactivation request repository
func NewReactivationRequestRepository(db *sql.DB) *ReactivationRequestRepository {
	return &ReactivationRequestRepository{db: db}
}

// Create creates a new reactivation request
func (r *ReactivationRequestRepository) Create(tenantID int, request *models.ReactivationRequest) error {
	query := `
		INSERT INTO reactivation_requests (tenant_id, user_id, status, created_at)
		VALUES (?, ?, 'pending', ?)
	`

	now := time.Now()
	result, err := r.db.Exec(query, tenantID, request.UserID, now)
	if err != nil {
		return fmt.Errorf("failed to create reactivation request: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get request ID: %w", err)
	}

	request.ID = int(id)
	request.TenantID = tenantID
	request.Status = "pending"
	request.CreatedAt = now

	return nil
}

// FindByID finds a reactivation request by ID within a tenant
func (r *ReactivationRequestRepository) FindByID(tenantID int, id int) (*models.ReactivationRequest, error) {
	query := `
		SELECT id, tenant_id, user_id, status, admin_message, reviewed_by, reviewed_at, created_at
		FROM reactivation_requests
		WHERE id = ? AND tenant_id = ?
	`

	request := &models.ReactivationRequest{}
	err := r.db.QueryRow(query, id, tenantID).Scan(
		&request.ID,
		&request.TenantID,
		&request.UserID,
		&request.Status,
		&request.AdminMessage,
		&request.ReviewedBy,
		&request.ReviewedAt,
		&request.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find reactivation request: %w", err)
	}

	return request, nil
}

// FindAllPending finds all pending reactivation requests within a tenant
func (r *ReactivationRequestRepository) FindAllPending(tenantID int) ([]*models.ReactivationRequest, error) {
	query := `
		SELECT id, tenant_id, user_id, status, admin_message, reviewed_by, reviewed_at, created_at
		FROM reactivation_requests
		WHERE status = 'pending' AND tenant_id = ?
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending requests: %w", err)
	}
	defer rows.Close()

	requests := []*models.ReactivationRequest{}
	for rows.Next() {
		request := &models.ReactivationRequest{}
		err := rows.Scan(
			&request.ID,
			&request.TenantID,
			&request.UserID,
			&request.Status,
			&request.AdminMessage,
			&request.ReviewedBy,
			&request.ReviewedAt,
			&request.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reactivation request: %w", err)
		}
		requests = append(requests, request)
	}

	return requests, nil
}

// Approve approves a reactivation request within a tenant
func (r *ReactivationRequestRepository) Approve(tenantID int, id int, reviewerID int, message *string) error {
	query := `
		UPDATE reactivation_requests
		SET status = 'approved', reviewed_by = ?, reviewed_at = ?, admin_message = ?
		WHERE id = ? AND tenant_id = ?
	`

	now := time.Now()
	_, err := r.db.Exec(query, reviewerID, now, message, id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to approve request: %w", err)
	}

	return nil
}

// Deny denies a reactivation request within a tenant
func (r *ReactivationRequestRepository) Deny(tenantID int, id int, reviewerID int, message *string) error {
	query := `
		UPDATE reactivation_requests
		SET status = 'denied', reviewed_by = ?, reviewed_at = ?, admin_message = ?
		WHERE id = ? AND tenant_id = ?
	`

	now := time.Now()
	_, err := r.db.Exec(query, reviewerID, now, message, id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to deny request: %w", err)
	}

	return nil
}

// HasPendingRequest checks if user has a pending reactivation request within a tenant
func (r *ReactivationRequestRepository) HasPendingRequest(tenantID int, userID int) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM reactivation_requests
		WHERE user_id = ? AND status = 'pending' AND tenant_id = ?
	`

	var count int
	err := r.db.QueryRow(query, userID, tenantID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check pending request: %w", err)
	}

	return count > 0, nil
}

// FindByUserID finds reactivation requests by user ID within a tenant
func (r *ReactivationRequestRepository) FindByUserID(tenantID int, userID int) ([]*models.ReactivationRequest, error) {
	query := `
		SELECT id, tenant_id, user_id, status, admin_message, reviewed_by, reviewed_at, created_at
		FROM reactivation_requests
		WHERE user_id = ? AND tenant_id = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, userID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query reactivation requests: %w", err)
	}
	defer rows.Close()

	requests := []*models.ReactivationRequest{}
	for rows.Next() {
		request := &models.ReactivationRequest{}
		err := rows.Scan(
			&request.ID,
			&request.TenantID,
			&request.UserID,
			&request.Status,
			&request.AdminMessage,
			&request.ReviewedBy,
			&request.ReviewedAt,
			&request.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reactivation request: %w", err)
		}
		requests = append(requests, request)
	}

	return requests, nil
}
