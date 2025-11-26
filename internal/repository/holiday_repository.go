package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/tranm/gassigeher/internal/models"
)

type HolidayRepository struct {
	db *sql.DB
}

func NewHolidayRepository(db *sql.DB) *HolidayRepository {
	return &HolidayRepository{db: db}
}

// GetHolidaysByYear returns all active holidays for a specific year
func (r *HolidayRepository) GetHolidaysByYear(year int) ([]models.CustomHoliday, error) {
	query := `
		SELECT id, date, name, is_active, source, created_at, created_by
		FROM custom_holidays
		WHERE is_active = 1
		  AND date LIKE ?
		ORDER BY date ASC
	`

	yearPrefix := fmt.Sprintf("%d-%%", year)
	rows, err := r.db.Query(query, yearPrefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanHolidays(rows)
}

// IsHoliday checks if a specific date is a holiday
func (r *HolidayRepository) IsHoliday(date string) (bool, error) {
	query := `
		SELECT COUNT(*) FROM custom_holidays
		WHERE date = ? AND is_active = 1
	`

	var count int
	err := r.db.QueryRow(query, date).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// CreateHoliday adds a custom holiday
func (r *HolidayRepository) CreateHoliday(holiday *models.CustomHoliday) error {
	query := `
		INSERT INTO custom_holidays (date, name, is_active, source, created_by)
		VALUES (?, ?, ?, ?, ?)
	`

	isActive := 1
	if !holiday.IsActive {
		isActive = 0
	}

	result, err := r.db.Exec(query, holiday.Date, holiday.Name, isActive, holiday.Source, holiday.CreatedBy)
	if err != nil {
		return err
	}

	id, _ := result.LastInsertId()
	holiday.ID = int(id)
	return nil
}

// UpdateHoliday updates a holiday
func (r *HolidayRepository) UpdateHoliday(id int, holiday *models.CustomHoliday) error {
	query := `
		UPDATE custom_holidays
		SET name = ?, is_active = ?
		WHERE id = ?
	`

	isActive := 1
	if !holiday.IsActive {
		isActive = 0
	}

	_, err := r.db.Exec(query, holiday.Name, isActive, id)
	return err
}

// DeleteHoliday deletes a holiday
func (r *HolidayRepository) DeleteHoliday(id int) error {
	query := `DELETE FROM custom_holidays WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

// GetCachedHolidays retrieves cached holidays from API
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

// SetCachedHolidays stores API response in cache
func (r *HolidayRepository) SetCachedHolidays(year int, state string, data string, cacheDays int) error {
	expiresAt := time.Now().AddDate(0, 0, cacheDays)

	query := `
		INSERT OR REPLACE INTO feiertage_cache (year, state, data, fetched_at, expires_at)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(query, year, state, data, time.Now(), expiresAt)
	return err
}

// scanHolidays helper to scan holiday rows
func (r *HolidayRepository) scanHolidays(rows *sql.Rows) ([]models.CustomHoliday, error) {
	var holidays []models.CustomHoliday

	for rows.Next() {
		var h models.CustomHoliday
		var isActive int
		var createdBy sql.NullInt64

		err := rows.Scan(&h.ID, &h.Date, &h.Name, &isActive, &h.Source, &h.CreatedAt, &createdBy)
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
