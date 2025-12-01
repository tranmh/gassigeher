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
func (r *ExperienceRequestRepository) Create(request *models.ExperienceRequest) error {
	query := `
		INSERT INTO experience_requests (user_id, requested_level, status, created_at)
		VALUES (?, ?, 'pending', ?)
	`

	now := time.Now()
	result, err := r.db.Exec(query, request.UserID, request.RequestedLevel, now)
	if err != nil {
		return fmt.Errorf("failed to create experience request: %w", err)
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

// FindByID finds an experience request by ID
func (r *ExperienceRequestRepository) FindByID(id int) (*models.ExperienceRequest, error) {
	query := `
		SELECT id, user_id, requested_level, status, admin_message, reviewed_by, reviewed_at, created_at
		FROM experience_requests
		WHERE id = ?
	`

	request := &models.ExperienceRequest{}
	err := r.db.QueryRow(query, id).Scan(
		&request.ID,
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

// FindByUserID finds experience requests by user ID
func (r *ExperienceRequestRepository) FindByUserID(userID int) ([]*models.ExperienceRequest, error) {
	query := `
		SELECT id, user_id, requested_level, status, admin_message, reviewed_by, reviewed_at, created_at
		FROM experience_requests
		WHERE user_id = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query experience requests: %w", err)
	}
	defer rows.Close()

	requests := []*models.ExperienceRequest{}
	for rows.Next() {
		request := &models.ExperienceRequest{}
		err := rows.Scan(
			&request.ID,
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

// FindAllPending finds all pending experience requests
func (r *ExperienceRequestRepository) FindAllPending() ([]*models.ExperienceRequest, error) {
	query := `
		SELECT id, user_id, requested_level, status, admin_message, reviewed_by, reviewed_at, created_at
		FROM experience_requests
		WHERE status = 'pending'
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending requests: %w", err)
	}
	defer rows.Close()

	requests := []*models.ExperienceRequest{}
	for rows.Next() {
		request := &models.ExperienceRequest{}
		err := rows.Scan(
			&request.ID,
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

// Approve approves an experience request
func (r *ExperienceRequestRepository) Approve(id int, reviewerID int, message *string) error {
	query := `
		UPDATE experience_requests
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

// Deny denies an experience request
func (r *ExperienceRequestRepository) Deny(id int, reviewerID int, message *string) error {
	query := `
		UPDATE experience_requests
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

// HasPendingRequest checks if user has a pending request for a level
func (r *ExperienceRequestRepository) HasPendingRequest(userID int, level string) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM experience_requests
		WHERE user_id = ? AND requested_level = ? AND status = 'pending'
	`

	var count int
	err := r.db.QueryRow(query, userID, level).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check pending request: %w", err)
	}

	return count > 0, nil
}
