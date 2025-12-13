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

// Create creates a new blocked date (global or dog-specific)
func (r *BlockedDateRepository) Create(blockedDate *models.BlockedDate) error {
	query := `
		INSERT INTO blocked_dates (date, dog_id, reason, created_by, created_at)
		VALUES (?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.Exec(query,
		blockedDate.Date,
		blockedDate.DogID, // Can be nil for global block
		blockedDate.Reason,
		blockedDate.CreatedBy,
		now,
	)

	if err != nil {
		// Check for unique constraint violation (different messages by DB)
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "unique") || strings.Contains(errStr, "duplicate") {
			if blockedDate.DogID != nil {
				return fmt.Errorf("this dog is already blocked for this date")
			}
			return fmt.Errorf("this date is already globally blocked")
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

// FindAll finds all blocked dates with optional dog name via JOIN
func (r *BlockedDateRepository) FindAll() ([]*models.BlockedDate, error) {
	query := `
		SELECT bd.id, bd.date, bd.dog_id, d.name, bd.reason, bd.created_by, bd.created_at
		FROM blocked_dates bd
		LEFT JOIN dogs d ON bd.dog_id = d.id
		ORDER BY bd.date ASC, bd.dog_id ASC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query blocked dates: %w", err)
	}
	defer rows.Close()

	blockedDates := []*models.BlockedDate{}
	for rows.Next() {
		blockedDate := &models.BlockedDate{}
		var dogName sql.NullString
		err := rows.Scan(
			&blockedDate.ID,
			&blockedDate.Date,
			&blockedDate.DogID,
			&dogName,
			&blockedDate.Reason,
			&blockedDate.CreatedBy,
			&blockedDate.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan blocked date: %w", err)
		}
		if dogName.Valid {
			blockedDate.DogName = &dogName.String
		}
		blockedDates = append(blockedDates, blockedDate)
	}

	return blockedDates, nil
}

// FindByDate finds a global blocked date by date (dog_id IS NULL)
func (r *BlockedDateRepository) FindByDate(date string) (*models.BlockedDate, error) {
	query := `
		SELECT id, date, dog_id, reason, created_by, created_at
		FROM blocked_dates
		WHERE date = ? AND dog_id IS NULL
	`

	blockedDate := &models.BlockedDate{}
	err := r.db.QueryRow(query, date).Scan(
		&blockedDate.ID,
		&blockedDate.Date,
		&blockedDate.DogID,
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

// FindByDateAndDog finds a blocked date by date and optional dog_id
func (r *BlockedDateRepository) FindByDateAndDog(date string, dogID *int) (*models.BlockedDate, error) {
	var query string
	var args []interface{}

	if dogID == nil {
		query = `
			SELECT id, date, dog_id, reason, created_by, created_at
			FROM blocked_dates
			WHERE date = ? AND dog_id IS NULL
		`
		args = []interface{}{date}
	} else {
		query = `
			SELECT id, date, dog_id, reason, created_by, created_at
			FROM blocked_dates
			WHERE date = ? AND dog_id = ?
		`
		args = []interface{}{date, *dogID}
	}

	blockedDate := &models.BlockedDate{}
	err := r.db.QueryRow(query, args...).Scan(
		&blockedDate.ID,
		&blockedDate.Date,
		&blockedDate.DogID,
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

// IsBlocked checks if a date is globally blocked (dog_id IS NULL)
// For backward compatibility - checks only global blocks
func (r *BlockedDateRepository) IsBlocked(date string) (bool, error) {
	query := `SELECT COUNT(*) FROM blocked_dates WHERE date = ? AND dog_id IS NULL`

	var count int
	err := r.db.QueryRow(query, date).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if date is blocked: %w", err)
	}

	return count > 0, nil
}

// IsBlockedForDog checks if a date is blocked for a specific dog
// Returns true if there's a global block (dog_id IS NULL) OR a dog-specific block
func (r *BlockedDateRepository) IsBlockedForDog(date string, dogID int) (bool, error) {
	query := `
		SELECT COUNT(*) FROM blocked_dates
		WHERE date = ? AND (dog_id IS NULL OR dog_id = ?)
	`

	var count int
	err := r.db.QueryRow(query, date, dogID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if date is blocked for dog: %w", err)
	}

	return count > 0, nil
}

// GetBlockedDogsForDate returns list of dog IDs blocked for a specific date
// Returns globalBlock=true if date is globally blocked (all dogs)
// Returns specific dogIDs if only certain dogs are blocked
func (r *BlockedDateRepository) GetBlockedDogsForDate(date string) (globalBlock bool, dogIDs []int, err error) {
	query := `SELECT dog_id FROM blocked_dates WHERE date = ?`

	rows, err := r.db.Query(query, date)
	if err != nil {
		return false, nil, fmt.Errorf("failed to query blocked dogs: %w", err)
	}
	defer rows.Close()

	dogIDs = []int{}
	for rows.Next() {
		var dogID sql.NullInt64
		if err := rows.Scan(&dogID); err != nil {
			return false, nil, fmt.Errorf("failed to scan dog_id: %w", err)
		}
		if !dogID.Valid {
			// NULL dog_id means global block
			return true, nil, nil
		}
		dogIDs = append(dogIDs, int(dogID.Int64))
	}

	return false, dogIDs, nil
}
