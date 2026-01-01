package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
)

// BlockedDateRepository handles blocked date database operations
type BlockedDateRepository struct {
	db DBExecutor
}

// NewBlockedDateRepository creates a new blocked date repository
func NewBlockedDateRepository(db DBExecutor) *BlockedDateRepository {
	return &BlockedDateRepository{db: db}
}

// Create creates a new blocked date (global or dog-specific)
// SaaS: Now includes tenant_id for multi-tenancy
func (r *BlockedDateRepository) Create(blockedDate *models.BlockedDate) error {
	query := `
		INSERT INTO blocked_dates (tenant_id, date, dog_id, reason, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	now := time.Now()

	// tenant_id=0 is valid for Simple-Mode (non-SaaS)
	id, err := r.db.InsertReturningID(query,
		blockedDate.TenantID,
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

	blockedDate.ID = int(id)
	blockedDate.CreatedAt = now

	return nil
}

// FindAll finds all blocked dates with optional dog name via JOIN
// SaaS: Now supports tenant filtering (tenantID=0 for all tenants)
func (r *BlockedDateRepository) FindAll(tenantID int) ([]*models.BlockedDate, error) {
	query := `
		SELECT bd.id, bd.tenant_id, bd.date, bd.dog_id, d.name, bd.reason, bd.created_by, bd.created_at
		FROM blocked_dates bd
		LEFT JOIN dogs d ON bd.dog_id = d.id
		WHERE 1=1
	`
	args := []interface{}{}

	// SaaS: Always filter by tenant (tenant_id=0 is valid for Simple-Mode)
	query += " AND bd.tenant_id = ?"
	args = append(args, tenantID)

	query += " ORDER BY bd.date ASC, bd.dog_id ASC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query blocked dates: %w", err)
	}
	defer rows.Close()

	blockedDates := []*models.BlockedDate{}
	for rows.Next() {
		blockedDate := &models.BlockedDate{}
		var tenantIDNull sql.NullInt64
		var dogName sql.NullString
		err := rows.Scan(
			&blockedDate.ID,
			&tenantIDNull,
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
		if tenantIDNull.Valid {
			blockedDate.TenantID = int(tenantIDNull.Int64)
		}
		if dogName.Valid {
			blockedDate.DogName = &dogName.String
		}
		blockedDates = append(blockedDates, blockedDate)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating blocked dates: %w", err)
	}

	return blockedDates, nil
}

// FindByDate finds a global blocked date by date (dog_id IS NULL)
// SaaS: Now supports tenant filtering (tenantID=0 for all tenants)
func (r *BlockedDateRepository) FindByDate(date string, tenantID int) (*models.BlockedDate, error) {
	query := `
		SELECT id, tenant_id, date, dog_id, reason, created_by, created_at
		FROM blocked_dates
		WHERE date = ? AND dog_id IS NULL
	`
	args := []interface{}{date}

	// SaaS: Always filter by tenant (tenant_id=0 is valid for Simple-Mode)
	query += " AND tenant_id = ?"
	args = append(args, tenantID)

	blockedDate := &models.BlockedDate{}
	var tenantIDNull sql.NullInt64
	err := r.db.QueryRow(query, args...).Scan(
		&blockedDate.ID,
		&tenantIDNull,
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

	if tenantIDNull.Valid {
		blockedDate.TenantID = int(tenantIDNull.Int64)
	}

	return blockedDate, nil
}

// FindByDateAndDog finds a blocked date by date and optional dog_id
// SaaS: Now supports tenant filtering (tenantID=0 for all tenants)
func (r *BlockedDateRepository) FindByDateAndDog(date string, dogID *int, tenantID int) (*models.BlockedDate, error) {
	var query string
	var args []interface{}

	if dogID == nil {
		query = `
			SELECT id, tenant_id, date, dog_id, reason, created_by, created_at
			FROM blocked_dates
			WHERE date = ? AND dog_id IS NULL
		`
		args = []interface{}{date}
	} else {
		query = `
			SELECT id, tenant_id, date, dog_id, reason, created_by, created_at
			FROM blocked_dates
			WHERE date = ? AND dog_id = ?
		`
		args = []interface{}{date, *dogID}
	}

	// SaaS: Always filter by tenant (tenant_id=0 is valid for Simple-Mode)
	query += " AND tenant_id = ?"
	args = append(args, tenantID)

	blockedDate := &models.BlockedDate{}
	var tenantIDNull sql.NullInt64
	err := r.db.QueryRow(query, args...).Scan(
		&blockedDate.ID,
		&tenantIDNull,
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

	if tenantIDNull.Valid {
		blockedDate.TenantID = int(tenantIDNull.Int64)
	}

	return blockedDate, nil
}

// FindByID finds a blocked date by ID
// SaaS SECURITY: Requires tenant_id to verify ownership before operations
func (r *BlockedDateRepository) FindByID(id int, tenantID int) (*models.BlockedDate, error) {
	query := `
		SELECT id, tenant_id, date, dog_id, reason, created_by, created_at
		FROM blocked_dates
		WHERE id = ?
	`
	args := []interface{}{id}

	// SaaS: Always filter by tenant (tenant_id=0 is valid for Simple-Mode)
	query += " AND tenant_id = ?"
	args = append(args, tenantID)

	blockedDate := &models.BlockedDate{}
	var tenantIDNull sql.NullInt64
	err := r.db.QueryRow(query, args...).Scan(
		&blockedDate.ID,
		&tenantIDNull,
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

	if tenantIDNull.Valid {
		blockedDate.TenantID = int(tenantIDNull.Int64)
	}

	return blockedDate, nil
}

// Delete deletes a blocked date with tenant isolation enforcement
// SaaS SECURITY: This function now enforces tenant isolation at the database level
func (r *BlockedDateRepository) Delete(id int, tenantID int) error {
	var query string
	var args []interface{}

	// Always filter by tenant_id (tenant_id=0 is valid for Simple-Mode)
	query = `DELETE FROM blocked_dates WHERE id = ? AND tenant_id = ?`
	args = []interface{}{id, tenantID}

	result, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete blocked date: %w", err)
	}

	// Verify a row was actually deleted
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to verify deletion: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("blocked date not found or access denied")
	}

	return nil
}

// IsBlocked checks if a date is globally blocked (dog_id IS NULL)
// For backward compatibility - checks only global blocks
// SaaS: Always filters by tenant (tenant_id=0 is valid for Simple-Mode)
func (r *BlockedDateRepository) IsBlocked(date string, tenantID int) (bool, error) {
	query := `SELECT COUNT(*) FROM blocked_dates WHERE date = ? AND dog_id IS NULL AND tenant_id = ?`
	args := []interface{}{date, tenantID}

	var count int
	err := r.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if date is blocked: %w", err)
	}

	return count > 0, nil
}

// IsBlockedForDog checks if a date is blocked for a specific dog
// Returns true if there's a global block (dog_id IS NULL) OR a dog-specific block
// SaaS: Always filters by tenant (tenant_id=0 is valid for Simple-Mode)
func (r *BlockedDateRepository) IsBlockedForDog(date string, dogID int, tenantID int) (bool, error) {
	query := `
		SELECT COUNT(*) FROM blocked_dates
		WHERE date = ? AND (dog_id IS NULL OR dog_id = ?) AND tenant_id = ?
	`
	args := []interface{}{date, dogID, tenantID}

	var count int
	err := r.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if date is blocked for dog: %w", err)
	}

	return count > 0, nil
}

// GetBlockedDogsForDate returns list of dog IDs blocked for a specific date
// Returns globalBlock=true if date is globally blocked (all dogs)
// Returns specific dogIDs if only certain dogs are blocked
// SaaS: Always filters by tenant (tenant_id=0 is valid for Simple-Mode)
func (r *BlockedDateRepository) GetBlockedDogsForDate(date string, tenantID int) (globalBlock bool, dogIDs []int, err error) {
	query := `SELECT dog_id FROM blocked_dates WHERE date = ? AND tenant_id = ?`
	args := []interface{}{date, tenantID}

	rows, err := r.db.Query(query, args...)
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

	if err := rows.Err(); err != nil {
		return false, nil, fmt.Errorf("error iterating blocked dogs: %w", err)
	}

	return false, dogIDs, nil
}
