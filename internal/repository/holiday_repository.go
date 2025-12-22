package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
)

type HolidayRepository struct {
	db *sql.DB
}

func NewHolidayRepository(db *sql.DB) *HolidayRepository {
	return &HolidayRepository{db: db}
}

// GetHolidaysByYear returns all active holidays for a specific year within a tenant
func (r *HolidayRepository) GetHolidaysByYear(tenantID int, year int) ([]models.CustomHoliday, error) {
	query := `
		SELECT id, tenant_id, date, name, is_active, source, created_at, created_by
		FROM custom_holidays
		WHERE is_active = 1
		  AND date LIKE ?
		  AND tenant_id = ?
		ORDER BY date ASC
	`

	yearPrefix := fmt.Sprintf("%d-%%", year)
	rows, err := r.db.Query(query, yearPrefix, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanHolidays(rows)
}

// IsHoliday checks if a specific date is a holiday within a tenant
func (r *HolidayRepository) IsHoliday(tenantID int, date string) (bool, error) {
	query := `
		SELECT COUNT(*) FROM custom_holidays
		WHERE date = ? AND is_active = 1 AND tenant_id = ?
	`

	var count int
	err := r.db.QueryRow(query, date, tenantID).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// CreateHoliday adds a custom holiday for a tenant
func (r *HolidayRepository) CreateHoliday(tenantID int, holiday *models.CustomHoliday) error {
	query := `
		INSERT INTO custom_holidays (tenant_id, date, name, is_active, source, created_by)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	isActive := 1
	if !holiday.IsActive {
		isActive = 0
	}

	result, err := r.db.Exec(query, tenantID, holiday.Date, holiday.Name, isActive, holiday.Source, holiday.CreatedBy)
	if err != nil {
		return err
	}

	id, _ := result.LastInsertId()
	holiday.ID = int(id)
	holiday.TenantID = tenantID
	return nil
}

// UpdateHoliday updates a holiday within a tenant
func (r *HolidayRepository) UpdateHoliday(tenantID int, id int, holiday *models.CustomHoliday) error {
	query := `
		UPDATE custom_holidays
		SET name = ?, is_active = ?
		WHERE id = ? AND tenant_id = ?
	`

	isActive := 1
	if !holiday.IsActive {
		isActive = 0
	}

	_, err := r.db.Exec(query, holiday.Name, isActive, id, tenantID)
	return err
}

// DeleteHoliday deletes a holiday within a tenant
func (r *HolidayRepository) DeleteHoliday(tenantID int, id int) error {
	query := `DELETE FROM custom_holidays WHERE id = ? AND tenant_id = ?`
	_, err := r.db.Exec(query, id, tenantID)
	return err
}

// GetCachedHolidays retrieves cached holidays from API (global cache, not tenant-specific)
func (r *HolidayRepository) GetCachedHolidays(year int, state string) (string, error) {
	query := `
		SELECT data FROM feiertage_cache
		WHERE year = ? AND state = ? AND expires_at > ?
	`

	var data string
	err := r.db.QueryRow(query, year, state, time.Now()).Scan(&data)
	if err == sql.ErrNoRows {
		return "", nil // Cache miss
	}
	if err != nil {
		return "", err
	}

	return data, nil
}

// SetCachedHolidays stores API response in cache (global cache, not tenant-specific)
// Uses standard SQL pattern (UPDATE then INSERT if no rows affected) for multi-database compatibility
func (r *HolidayRepository) SetCachedHolidays(year int, state string, data string, cacheDays int) error {
	expiresAt := time.Now().AddDate(0, 0, cacheDays)
	now := time.Now()

	// Try UPDATE first (standard SQL pattern for upsert)
	updateQuery := `UPDATE feiertage_cache SET data = ?, fetched_at = ?, expires_at = ? WHERE year = ? AND state = ?`
	result, err := r.db.Exec(updateQuery, data, now, expiresAt, year, state)
	if err != nil {
		return fmt.Errorf("failed to update cache: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		// Row doesn't exist, INSERT it
		insertQuery := `INSERT INTO feiertage_cache (year, state, data, fetched_at, expires_at) VALUES (?, ?, ?, ?, ?)`
		_, err = r.db.Exec(insertQuery, year, state, data, now, expiresAt)
		if err != nil {
			return fmt.Errorf("failed to insert cache: %w", err)
		}
	}
	return nil
}

// scanHolidays helper to scan holiday rows
func (r *HolidayRepository) scanHolidays(rows *sql.Rows) ([]models.CustomHoliday, error) {
	var holidays []models.CustomHoliday

	for rows.Next() {
		var h models.CustomHoliday
		var isActive int
		var createdBy sql.NullInt64

		err := rows.Scan(&h.ID, &h.TenantID, &h.Date, &h.Name, &isActive, &h.Source, &h.CreatedAt, &createdBy)
		if err != nil {
			return nil, err
		}

		h.IsActive = isActive == 1
		if createdBy.Valid {
			id := int(createdBy.Int64)
			h.CreatedBy = &id
		}

		holidays = append(holidays, h)
	}

	return holidays, nil
}
