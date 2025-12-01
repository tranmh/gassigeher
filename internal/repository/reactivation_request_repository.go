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
func (r *ReactivationRequestRepository) Create(request *models.ReactivationRequest) error {
	query := `
		INSERT INTO reactivation_requests (user_id, status, created_at)
		VALUES (?, 'pending', ?)
	`

	now := time.Now()
	result, err := r.db.Exec(query, request.UserID, now)
	if err != nil {
		return fmt.Errorf("failed to create reactivation request: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get request ID: %w", err)
	}

	request.ID = int(id)
	request.Status = "pending"
	request.CreatedAt = now

	return nil
}

// FindByID finds a reactivation request by ID
func (r *ReactivationRequestRepository) FindByID(id int) (*models.ReactivationRequest, error) {
	query := `
		SELECT id, user_id, status, admin_message, reviewed_by, reviewed_at, created_at
		FROM reactivation_requests
		WHERE id = ?
	`

	request := &models.ReactivationRequest{}
	err := r.db.QueryRow(query, id).Scan(
		&request.ID,
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

// FindAllPending finds all pending reactivation requests
func (r *ReactivationRequestRepository) FindAllPending() ([]*models.ReactivationRequest, error) {
	query := `
		SELECT id, user_id, status, admin_message, reviewed_by, reviewed_at, created_at
		FROM reactivation_requests
		WHERE status = 'pending'
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending requests: %w", err)
	}
	defer rows.Close()

	requests := []*models.ReactivationRequest{}
	for rows.Next() {
		request := &models.ReactivationRequest{}
		err := rows.Scan(
			&request.ID,
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

// Approve approves a reactivation request
func (r *ReactivationRequestRepository) Approve(id int, reviewerID int, message *string) error {
	query := `
		UPDATE reactivation_requests
		SET status = 'approved', reviewed_by = ?, reviewed_at = ?, admin_message = ?
		WHERE id = ?
	`

	now := time.Now()
	_, err := r.db.Exec(query, reviewerID, now, message, id)
	if err != nil {
		return fmt.Errorf("failed to approve request: %w", err)
	}

	return nil
}

// Deny denies a reactivation request
func (r *ReactivationRequestRepository) Deny(id int, reviewerID int, message *string) error {
	query := `
		UPDATE reactivation_requests
		SET status = 'denied', reviewed_by = ?, reviewed_at = ?, admin_message = ?
		WHERE id = ?
	`

	now := time.Now()
	_, err := r.db.Exec(query, reviewerID, now, message, id)
	if err != nil {
		return fmt.Errorf("failed to deny request: %w", err)
	}

	return nil
}

// HasPendingRequest checks if user has a pending reactivation request
func (r *ReactivationRequestRepository) HasPendingRequest(userID int) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM reactivation_requests
		WHERE user_id = ? AND status = 'pending'
	`

	var count int
	err := r.db.QueryRow(query, userID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check pending request: %w", err)
	}

	return count > 0, nil
}

// FindByUserID finds reactivation requests by user ID
func (r *ReactivationRequestRepository) FindByUserID(userID int) ([]*models.ReactivationRequest, error) {
	query := `
		SELECT id, user_id, status, admin_message, reviewed_by, reviewed_at, created_at
		FROM reactivation_requests
		WHERE user_id = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query reactivation requests: %w", err)
	}
	defer rows.Close()

	requests := []*models.ReactivationRequest{}
	for rows.Next() {
		request := &models.ReactivationRequest{}
		err := rows.Scan(
			&request.ID,
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
