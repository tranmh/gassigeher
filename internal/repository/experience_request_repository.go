package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
)

// ExperienceRequestRepository handles experience request database operations
type ExperienceRequestRepository struct {
	db *sql.DB
}

// NewExperienceRequestRepository creates a new experience request repository
func NewExperienceRequestRepository(db *sql.DB) *ExperienceRequestRepository {
	return &ExperienceRequestRepository{db: db}
}

// Create creates a new experience request
func (r *ExperienceRequestRepository) Create(tenantID int, request *models.ExperienceRequest) error {
	query := `
		INSERT INTO experience_requests (tenant_id, user_id, requested_level, status, created_at)
		VALUES (?, ?, ?, 'pending', ?)
	`

	now := time.Now()
	result, err := r.db.Exec(query, tenantID, request.UserID, request.RequestedLevel, now)
	if err != nil {
		return fmt.Errorf("failed to create experience request: %w", err)
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

// FindByID finds an experience request by ID within a tenant
func (r *ExperienceRequestRepository) FindByID(tenantID int, id int) (*models.ExperienceRequest, error) {
	query := `
		SELECT id, tenant_id, user_id, requested_level, status, admin_message, reviewed_by, reviewed_at, created_at
		FROM experience_requests
		WHERE id = ? AND tenant_id = ?
	`

	request := &models.ExperienceRequest{}
	err := r.db.QueryRow(query, id, tenantID).Scan(
		&request.ID,
		&request.TenantID,
		&request.UserID,
		&request.RequestedLevel,
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
		return nil, fmt.Errorf("failed to find experience request: %w", err)
	}

	return request, nil
}

// FindByUserID finds experience requests by user ID within a tenant
func (r *ExperienceRequestRepository) FindByUserID(tenantID int, userID int) ([]*models.ExperienceRequest, error) {
	query := `
		SELECT id, tenant_id, user_id, requested_level, status, admin_message, reviewed_by, reviewed_at, created_at
		FROM experience_requests
		WHERE user_id = ? AND tenant_id = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, userID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query experience requests: %w", err)
	}
	defer rows.Close()

	requests := []*models.ExperienceRequest{}
	for rows.Next() {
		request := &models.ExperienceRequest{}
		err := rows.Scan(
			&request.ID,
			&request.TenantID,
			&request.UserID,
			&request.RequestedLevel,
			&request.Status,
			&request.AdminMessage,
			&request.ReviewedBy,
			&request.ReviewedAt,
			&request.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan experience request: %w", err)
		}
		requests = append(requests, request)
	}

	return requests, nil
}

// FindAllPending finds all pending experience requests within a tenant
func (r *ExperienceRequestRepository) FindAllPending(tenantID int) ([]*models.ExperienceRequest, error) {
	query := `
		SELECT id, tenant_id, user_id, requested_level, status, admin_message, reviewed_by, reviewed_at, created_at
		FROM experience_requests
		WHERE status = 'pending' AND tenant_id = ?
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending requests: %w", err)
	}
	defer rows.Close()

	requests := []*models.ExperienceRequest{}
	for rows.Next() {
		request := &models.ExperienceRequest{}
		err := rows.Scan(
			&request.ID,
			&request.TenantID,
			&request.UserID,
			&request.RequestedLevel,
			&request.Status,
			&request.AdminMessage,
			&request.ReviewedBy,
			&request.ReviewedAt,
			&request.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan experience request: %w", err)
		}
		requests = append(requests, request)
	}

	return requests, nil
}

// Approve approves an experience request within a tenant
func (r *ExperienceRequestRepository) Approve(tenantID int, id int, reviewerID int, message *string) error {
	query := `
		UPDATE experience_requests
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

// Deny denies an experience request within a tenant
func (r *ExperienceRequestRepository) Deny(tenantID int, id int, reviewerID int, message *string) error {
	query := `
		UPDATE experience_requests
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

// HasPendingRequest checks if user has a pending request for a level within a tenant
func (r *ExperienceRequestRepository) HasPendingRequest(tenantID int, userID int, level string) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM experience_requests
		WHERE user_id = ? AND requested_level = ? AND status = 'pending' AND tenant_id = ?
	`

	var count int
	err := r.db.QueryRow(query, userID, level, tenantID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check pending request: %w", err)
	}

	return count > 0, nil
}
