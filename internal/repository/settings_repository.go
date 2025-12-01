package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
)

// SettingsRepository handles system settings database operations
type SettingsRepository struct {
	db *sql.DB
}

// NewSettingsRepository creates a new settings repository
func NewSettingsRepository(db *sql.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// Get retrieves a setting by key
func (r *SettingsRepository) Get(key string) (*models.SystemSetting, error) {
	query := `
		SELECT key, value, updated_at
		FROM system_settings
		WHERE key = ?
	`

	setting := &models.SystemSetting{}
	err := r.db.QueryRow(query, key).Scan(
		&setting.Key,
		&setting.Value,
		&setting.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get setting: %w", err)
	}

	return setting, nil
}

// GetAll retrieves all settings
func (r *SettingsRepository) GetAll() ([]*models.SystemSetting, error) {
	query := `
		SELECT key, value, updated_at
		FROM system_settings
		ORDER BY key ASC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query settings: %w", err)
	}
	defer rows.Close()

	settings := []*models.SystemSetting{}
	for rows.Next() {
		setting := &models.SystemSetting{}
		err := rows.Scan(
			&setting.Key,
			&setting.Value,
			&setting.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan setting: %w", err)
		}
		settings = append(settings, setting)
	}

	return settings, nil
}

// Update updates a setting value
func (r *SettingsRepository) Update(key, value string) error {
	query := `
		UPDATE system_settings
		SET value = ?, updated_at = ?
		WHERE key = ?
	`

	result, err := r.db.Exec(query, value, time.Now(), key)
	if err != nil {
		return fmt.Errorf("failed to update setting: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("setting not found")
	}

	return nil
}
