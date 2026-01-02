package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
)

// SettingsRepository handles system settings database operations
type SettingsRepository struct {
	db DBExecutor
}

// NewSettingsRepository creates a new settings repository
func NewSettingsRepository(db DBExecutor) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// Get retrieves a setting by key within a tenant
func (r *SettingsRepository) Get(tenantID int, key string) (*models.SystemSetting, error) {
	// Note: "key" is a SQL reserved word, must be quoted for PostgreSQL compatibility
	query := `
		SELECT tenant_id, "key", value, updated_at
		FROM system_settings
		WHERE "key" = ? AND tenant_id = ?
	`

	setting := &models.SystemSetting{}
	err := r.db.QueryRow(query, key, tenantID).Scan(
		&setting.TenantID,
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

// GetAll retrieves all settings for a tenant
func (r *SettingsRepository) GetAll(tenantID int) ([]*models.SystemSetting, error) {
	// Note: "key" is a SQL reserved word, must be quoted for PostgreSQL compatibility
	query := `
		SELECT tenant_id, "key", value, updated_at
		FROM system_settings
		WHERE tenant_id = ?
		ORDER BY "key" ASC
	`

	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query settings: %w", err)
	}
	defer rows.Close()

	settings := []*models.SystemSetting{}
	for rows.Next() {
		setting := &models.SystemSetting{}
		err := rows.Scan(
			&setting.TenantID,
			&setting.Key,
			&setting.Value,
			&setting.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan setting: %w", err)
		}
		settings = append(settings, setting)
	}

	// BUG FIX: Check for errors during iteration
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating settings: %w", err)
	}

	return settings, nil
}

// Update updates a setting value within a tenant
func (r *SettingsRepository) Update(tenantID int, key, value string) error {
	// Note: "key" is a SQL reserved word, must be quoted for PostgreSQL compatibility
	query := `
		UPDATE system_settings
		SET value = ?, updated_at = ?
		WHERE "key" = ? AND tenant_id = ?
	`

	result, err := r.db.Exec(query, value, time.Now(), key, tenantID)
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

// Create creates a new setting for a tenant
func (r *SettingsRepository) Create(tenantID int, key, value string) error {
	// Note: "key" is a SQL reserved word, must be quoted for PostgreSQL compatibility
	query := `
		INSERT INTO system_settings (tenant_id, "key", value, updated_at)
		VALUES (?, ?, ?, ?)
	`

	_, err := r.db.Exec(query, tenantID, key, value, time.Now())
	if err != nil {
		return fmt.Errorf("failed to create setting: %w", err)
	}

	return nil
}

// Upsert creates or updates a setting for a tenant
func (r *SettingsRepository) Upsert(tenantID int, key, value string) error {
	// Try to update first
	err := r.Update(tenantID, key, value)
	if err != nil && err.Error() == "setting not found" {
		// Setting doesn't exist, create it
		return r.Create(tenantID, key, value)
	}
	return err
}
