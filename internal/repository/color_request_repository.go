package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
)

// ColorRequestRepository handles color request database operations
type ColorRequestRepository struct {
	db *sql.DB
}

// NewColorRequestRepository creates a new color request repository
func NewColorRequestRepository(db *sql.DB) *ColorRequestRepository {
	return &ColorRequestRepository{db: db}
}

// Create creates a new color request
func (r *ColorRequestRepository) Create(request *models.ColorRequest) error {
	query := `
		INSERT INTO color_requests (user_id, color_id, status, created_at)
		VALUES (?, ?, 'pending', ?)
	`

	now := time.Now()
	result, err := r.db.Exec(query, request.UserID, request.ColorID, now)
	if err != nil {
		return fmt.Errorf("failed to create color request: %w", err)
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

// FindByID finds a color request by ID
func (r *ColorRequestRepository) FindByID(id int) (*models.ColorRequest, error) {
	query := `
		SELECT id, user_id, color_id, status, admin_message, reviewed_by, reviewed_at, created_at
		FROM color_requests
		WHERE id = ?
	`

	request := &models.ColorRequest{}
	err := r.db.QueryRow(query, id).Scan(
		&request.ID,
		&request.UserID,
		&request.ColorID,
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
		return nil, fmt.Errorf("failed to find color request: %w", err)
	}

	return request, nil
}

// FindByIDWithDetails finds a color request by ID with user and color data
func (r *ColorRequestRepository) FindByIDWithDetails(id int) (*models.ColorRequest, error) {
	query := `
		SELECT cr.id, cr.user_id, cr.color_id, cr.status, cr.admin_message, cr.reviewed_by, cr.reviewed_at, cr.created_at,
		       u.id, u.first_name, u.last_name, u.email,
		       c.id, c.name, c.hex_code, c.pattern_icon
		FROM color_requests cr
		LEFT JOIN users u ON u.id = cr.user_id
		LEFT JOIN color_categories c ON c.id = cr.color_id
		WHERE cr.id = ?
	`

	request := &models.ColorRequest{
		User:  &models.User{},
		Color: &models.ColorCategory{},
	}

	var userID, colorID sql.NullInt64
	var firstName, lastName, email sql.NullString
	var colorName, colorHex sql.NullString
	var patternIcon sql.NullString

	err := r.db.QueryRow(query, id).Scan(
		&request.ID,
		&request.UserID,
		&request.ColorID,
		&request.Status,
		&request.AdminMessage,
		&request.ReviewedBy,
		&request.ReviewedAt,
		&request.CreatedAt,
		&userID,
		&firstName,
		&lastName,
		&email,
		&colorID,
		&colorName,
		&colorHex,
		&patternIcon,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find color request with details: %w", err)
	}

	if userID.Valid {
		request.User.ID = int(userID.Int64)
		request.User.FirstName = firstName.String
		request.User.LastName = lastName.String
		request.User.Email = &email.String
	} else {
		request.User = nil
	}

	if colorID.Valid {
		request.Color.ID = int(colorID.Int64)
		request.Color.Name = colorName.String
		request.Color.HexCode = colorHex.String
		if patternIcon.Valid {
			request.Color.PatternIcon = &patternIcon.String
		}
	} else {
		request.Color = nil
	}

	return request, nil
}

// FindByUserID finds color requests by user ID
func (r *ColorRequestRepository) FindByUserID(userID int) ([]*models.ColorRequest, error) {
	query := `
		SELECT cr.id, cr.user_id, cr.color_id, cr.status, cr.admin_message, cr.reviewed_by, cr.reviewed_at, cr.created_at,
		       c.id, c.name, c.hex_code, c.pattern_icon
		FROM color_requests cr
		LEFT JOIN color_categories c ON c.id = cr.color_id
		WHERE cr.user_id = ?
		ORDER BY cr.created_at DESC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query color requests: %w", err)
	}
	defer rows.Close()

	requests := []*models.ColorRequest{}
	for rows.Next() {
		request := &models.ColorRequest{
			Color: &models.ColorCategory{},
		}

		var colorID sql.NullInt64
		var colorName, colorHex sql.NullString
		var patternIcon sql.NullString

		err := rows.Scan(
			&request.ID,
			&request.UserID,
			&request.ColorID,
			&request.Status,
			&request.AdminMessage,
			&request.ReviewedBy,
			&request.ReviewedAt,
			&request.CreatedAt,
			&colorID,
			&colorName,
			&colorHex,
			&patternIcon,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan color request: %w", err)
		}

		if colorID.Valid {
			request.Color.ID = int(colorID.Int64)
			request.Color.Name = colorName.String
			request.Color.HexCode = colorHex.String
			if patternIcon.Valid {
				request.Color.PatternIcon = &patternIcon.String
			}
		} else {
			request.Color = nil
		}

		requests = append(requests, request)
	}

	return requests, nil
}

// FindAllPending finds all pending color requests with user and color details
func (r *ColorRequestRepository) FindAllPending() ([]*models.ColorRequest, error) {
	query := `
		SELECT cr.id, cr.user_id, cr.color_id, cr.status, cr.admin_message, cr.reviewed_by, cr.reviewed_at, cr.created_at,
		       u.id, u.first_name, u.last_name, u.email,
		       c.id, c.name, c.hex_code, c.pattern_icon
		FROM color_requests cr
		LEFT JOIN users u ON u.id = cr.user_id
		LEFT JOIN color_categories c ON c.id = cr.color_id
		WHERE cr.status = 'pending'
		ORDER BY cr.created_at ASC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending requests: %w", err)
	}
	defer rows.Close()

	requests := []*models.ColorRequest{}
	for rows.Next() {
		request := &models.ColorRequest{
			User:  &models.User{},
			Color: &models.ColorCategory{},
		}

		var userID, colorID sql.NullInt64
		var firstName, lastName, email sql.NullString
		var colorName, colorHex sql.NullString
		var patternIcon sql.NullString

		err := rows.Scan(
			&request.ID,
			&request.UserID,
			&request.ColorID,
			&request.Status,
			&request.AdminMessage,
			&request.ReviewedBy,
			&request.ReviewedAt,
			&request.CreatedAt,
			&userID,
			&firstName,
			&lastName,
			&email,
			&colorID,
			&colorName,
			&colorHex,
			&patternIcon,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan color request: %w", err)
		}

		if userID.Valid {
			request.User.ID = int(userID.Int64)
			request.User.FirstName = firstName.String
			request.User.LastName = lastName.String
			request.User.Email = &email.String
		} else {
			request.User = nil
		}

		if colorID.Valid {
			request.Color.ID = int(colorID.Int64)
			request.Color.Name = colorName.String
			request.Color.HexCode = colorHex.String
			if patternIcon.Valid {
				request.Color.PatternIcon = &patternIcon.String
			}
		} else {
			request.Color = nil
		}

		requests = append(requests, request)
	}

	return requests, nil
}

// Approve approves a color request
func (r *ColorRequestRepository) Approve(id int, reviewerID int, message *string) error {
	query := `
		UPDATE color_requests
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

// Deny denies a color request
func (r *ColorRequestRepository) Deny(id int, reviewerID int, message *string) error {
	query := `
		UPDATE color_requests
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

// HasPendingRequest checks if user has any pending color request
func (r *ColorRequestRepository) HasPendingRequest(userID int) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM color_requests
		WHERE user_id = ? AND status = 'pending'
	`

	var count int
	err := r.db.QueryRow(query, userID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check pending request: %w", err)
	}

	return count > 0, nil
}

// HasPendingRequestForColor checks if user has a pending request for a specific color
func (r *ColorRequestRepository) HasPendingRequestForColor(userID int, colorID int) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM color_requests
		WHERE user_id = ? AND color_id = ? AND status = 'pending'
	`

	var count int
	err := r.db.QueryRow(query, userID, colorID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check pending request for color: %w", err)
	}

	return count > 0, nil
}
