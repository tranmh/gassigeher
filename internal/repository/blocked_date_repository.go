package repository

import (
	"database/sql"
	"fmt"
	"time"
	"strings"

	"github.com/tranmh/gassigeher/internal/models"
)

// BlockedDateRepository handles blocked date database operations
type BlockedDateRepository struct {
	db *sql.DB
}

// NewBlockedDateRepository creates a new blocked date repository
func NewBlockedDateRepository(db *sql.DB) *BlockedDateRepository {
	return &BlockedDateRepository{db: db}
}

// Create creates a new blocked date
func (r *BlockedDateRepository) Create(blockedDate *models.BlockedDate) error {
	query := `
		INSERT INTO blocked_dates (date, reason, created_by, created_at)
		VALUES (?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.Exec(query,
		blockedDate.Date,
		blockedDate.Reason,
		blockedDate.CreatedBy,
		now,
	)

	if err != nil {
		// Check for unique constraint violation
		if strings.Contains(err.Error(), "UNIQUE constraint failed: blocked_dates.date") {
			return fmt.Errorf("date is already blocked")
		}
		return fmt.Errorf("failed to create blocked date: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get blocked date ID: %w", err)
	}

	blockedDate.ID = int(id)
	blockedDate.CreatedAt = now

	return nil
}

// FindAll finds all blocked dates
func (r *BlockedDateRepository) FindAll() ([]*models.BlockedDate, error) {
	query := `
		SELECT id, date, reason, created_by, created_at
		FROM blocked_dates
		ORDER BY date ASC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query blocked dates: %w", err)
	}
	defer rows.Close()

	blockedDates := []*models.BlockedDate{}
	for rows.Next() {
		blockedDate := &models.BlockedDate{}
		err := rows.Scan(
			&blockedDate.ID,
			&blockedDate.Date,
			&blockedDate.Reason,
			&blockedDate.CreatedBy,
			&blockedDate.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan blocked date: %w", err)
		}
		blockedDates = append(blockedDates, blockedDate)
	}

	return blockedDates, nil
}

// FindByDate finds a blocked date by date
func (r *BlockedDateRepository) FindByDate(date string) (*models.BlockedDate, error) {
	query := `
		SELECT id, date, reason, created_by, created_at
		FROM blocked_dates
		WHERE date = ?
	`

	blockedDate := &models.BlockedDate{}
	err := r.db.QueryRow(query, date).Scan(
		&blockedDate.ID,
		&blockedDate.Date,
		&blockedDate.Reason,
		&blockedDate.CreatedBy,
		&blockedDate.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find blocked date: %w", err)
	}

	return blockedDate, nil
}

// Delete deletes a blocked date
func (r *BlockedDateRepository) Delete(id int) error {
	query := `DELETE FROM blocked_dates WHERE id = ?`

	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete blocked date: %w", err)
	}

	return nil
}

// IsBlocked checks if a date is blocked
func (r *BlockedDateRepository) IsBlocked(date string) (bool, error) {
	query := `SELECT COUNT(*) FROM blocked_dates WHERE date = ?`

	var count int
	err := r.db.QueryRow(query, date).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if date is blocked: %w", err)
	}

	return count > 0, nil
}
