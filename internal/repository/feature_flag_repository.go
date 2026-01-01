package repository

import (
	"database/sql"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
)

// FeatureFlagRepository handles feature flag database operations
type FeatureFlagRepository struct {
	db DBExecutor
}

// NewFeatureFlagRepository creates a new feature flag repository
func NewFeatureFlagRepository(db DBExecutor) *FeatureFlagRepository {
	return &FeatureFlagRepository{db: db}
}

// GetAll returns all feature flags
func (r *FeatureFlagRepository) GetAll() ([]*models.FeatureFlag, error) {
	rows, err := r.db.Query(`
		SELECT id, key, name, description, is_global, is_enabled, created_at, updated_at
		FROM feature_flags
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flags []*models.FeatureFlag
	for rows.Next() {
		f := &models.FeatureFlag{}
		var description sql.NullString
		err := rows.Scan(&f.ID, &f.Key, &f.Name, &description, &f.IsGlobal, &f.IsEnabled, &f.CreatedAt, &f.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if description.Valid {
			f.Description = description.String
		}
		flags = append(flags, f)
	}
	return flags, nil
}

// GetByKey returns a feature flag by its key
func (r *FeatureFlagRepository) GetByKey(key string) (*models.FeatureFlag, error) {
	f := &models.FeatureFlag{}
	var description sql.NullString
	err := r.db.QueryRow(`
		SELECT id, key, name, description, is_global, is_enabled, created_at, updated_at
		FROM feature_flags
		WHERE key = ?
	`, key).Scan(&f.ID, &f.Key, &f.Name, &description, &f.IsGlobal, &f.IsEnabled, &f.CreatedAt, &f.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if description.Valid {
		f.Description = description.String
	}
	return f, nil
}

// GetByID returns a feature flag by its ID
func (r *FeatureFlagRepository) GetByID(id int) (*models.FeatureFlag, error) {
	f := &models.FeatureFlag{}
	var description sql.NullString
	err := r.db.QueryRow(`
		SELECT id, key, name, description, is_global, is_enabled, created_at, updated_at
		FROM feature_flags
		WHERE id = ?
	`, id).Scan(&f.ID, &f.Key, &f.Name, &description, &f.IsGlobal, &f.IsEnabled, &f.CreatedAt, &f.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if description.Valid {
		f.Description = description.String
	}
	return f, nil
}

// Create creates a new feature flag
func (r *FeatureFlagRepository) Create(f *models.FeatureFlag) error {
	now := time.Now()
	query := `
		INSERT INTO feature_flags (key, name, description, is_global, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	id, err := r.db.InsertReturningID(query, f.Key, f.Name, f.Description, r.db.BoolValue(f.IsGlobal), r.db.BoolValue(f.IsEnabled), now, now)
	if err != nil {
		return err
	}

	f.ID = int(id)
	f.CreatedAt = now
	f.UpdatedAt = now
	return nil
}

// Update updates an existing feature flag
func (r *FeatureFlagRepository) Update(f *models.FeatureFlag) error {
	now := time.Now()
	_, err := r.db.Exec(`
		UPDATE feature_flags
		SET name = ?, description = ?, is_global = ?, is_enabled = ?, updated_at = ?
		WHERE id = ?
	`, f.Name, f.Description, r.db.BoolValue(f.IsGlobal), r.db.BoolValue(f.IsEnabled), now, f.ID)
	if err != nil {
		return err
	}
	f.UpdatedAt = now
	return nil
}

// Delete deletes a feature flag
func (r *FeatureFlagRepository) Delete(id int) error {
	// First delete tenant overrides
	_, err := r.db.Exec(`DELETE FROM tenant_feature_flags WHERE feature_flag_id = ?`, id)
	if err != nil {
		return err
	}
	// Then delete the flag itself
	_, err = r.db.Exec(`DELETE FROM feature_flags WHERE id = ?`, id)
	return err
}

// GetAllWithTenantStatus returns all flags with their status for a specific tenant
func (r *FeatureFlagRepository) GetAllWithTenantStatus(tenantID int) ([]*models.FeatureFlagWithStatus, error) {
	rows, err := r.db.Query(`
		SELECT
			f.id, f.key, f.name, f.description, f.is_global, f.is_enabled, f.created_at, f.updated_at,
			tf.is_enabled, tf.enabled_at
		FROM feature_flags f
		LEFT JOIN tenant_feature_flags tf ON f.id = tf.feature_flag_id AND tf.tenant_id = ?
		ORDER BY f.name ASC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flags []*models.FeatureFlagWithStatus
	for rows.Next() {
		f := &models.FeatureFlagWithStatus{}
		var description sql.NullString
		var tenantEnabled sql.NullBool
		var tenantEnabledAt sql.NullTime
		err := rows.Scan(
			&f.ID, &f.Key, &f.Name, &description, &f.IsGlobal, &f.IsEnabled, &f.CreatedAt, &f.UpdatedAt,
			&tenantEnabled, &tenantEnabledAt,
		)
		if err != nil {
			return nil, err
		}
		if description.Valid {
			f.Description = description.String
		}
		if tenantEnabled.Valid {
			f.TenantEnabled = &tenantEnabled.Bool
			f.EffectiveEnabled = tenantEnabled.Bool
		} else {
			// No tenant override - use global setting if it's a global flag
			if f.IsGlobal {
				f.EffectiveEnabled = f.IsEnabled
			} else {
				f.EffectiveEnabled = false
			}
		}
		if tenantEnabledAt.Valid {
			f.TenantEnabledAt = &tenantEnabledAt.Time
		}
		flags = append(flags, f)
	}
	return flags, nil
}

// IsEnabled checks if a feature is enabled for a tenant
func (r *FeatureFlagRepository) IsEnabled(key string, tenantID int) (bool, error) {
	var isGlobal, globalEnabled bool
	var tenantEnabled sql.NullBool

	err := r.db.QueryRow(`
		SELECT f.is_global, f.is_enabled, tf.is_enabled
		FROM feature_flags f
		LEFT JOIN tenant_feature_flags tf ON f.id = tf.feature_flag_id AND tf.tenant_id = ?
		WHERE f.key = ?
	`, tenantID, key).Scan(&isGlobal, &globalEnabled, &tenantEnabled)

	if err == sql.ErrNoRows {
		return false, nil // Flag doesn't exist
	}
	if err != nil {
		return false, err
	}

	// Tenant override takes precedence
	if tenantEnabled.Valid {
		return tenantEnabled.Bool, nil
	}

	// Fall back to global setting if it's a global flag
	if isGlobal {
		return globalEnabled, nil
	}

	// Non-global flag with no tenant override = disabled
	return false, nil
}

// SetTenantFlag sets a feature flag for a specific tenant
func (r *FeatureFlagRepository) SetTenantFlag(tenantID, flagID int, isEnabled bool, enabledBy *int) error {
	now := time.Now()
	boolVal := r.db.BoolValue(isEnabled)

	// Try to insert, if exists update
	result, err := r.db.Exec(`
		INSERT INTO tenant_feature_flags (tenant_id, feature_flag_id, is_enabled, enabled_at, enabled_by)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(tenant_id, feature_flag_id) DO UPDATE SET
			is_enabled = ?, enabled_at = ?, enabled_by = ?
	`, tenantID, flagID, boolVal, now, enabledBy, boolVal, now, enabledBy)

	if err != nil {
		// Fallback for MySQL which uses different syntax
		_, err = r.db.Exec(`
			INSERT INTO tenant_feature_flags (tenant_id, feature_flag_id, is_enabled, enabled_at, enabled_by)
			VALUES (?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE is_enabled = ?, enabled_at = ?, enabled_by = ?
		`, tenantID, flagID, boolVal, now, enabledBy, boolVal, now, enabledBy)
		if err != nil {
			return err
		}
	} else {
		// Check if insert was successful (for databases that don't support ON CONFLICT)
		affected, _ := result.RowsAffected()
		if affected == 0 {
			// Row exists, do an update
			_, err = r.db.Exec(`
				UPDATE tenant_feature_flags
				SET is_enabled = ?, enabled_at = ?, enabled_by = ?
				WHERE tenant_id = ? AND feature_flag_id = ?
			`, boolVal, now, enabledBy, tenantID, flagID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// RemoveTenantFlag removes a tenant-specific feature flag override
func (r *FeatureFlagRepository) RemoveTenantFlag(tenantID, flagID int) error {
	_, err := r.db.Exec(`
		DELETE FROM tenant_feature_flags
		WHERE tenant_id = ? AND feature_flag_id = ?
	`, tenantID, flagID)
	return err
}

// GetTenantFlag gets a tenant's feature flag override
func (r *FeatureFlagRepository) GetTenantFlag(tenantID, flagID int) (*models.TenantFeatureFlag, error) {
	tf := &models.TenantFeatureFlag{}
	var enabledBy sql.NullInt64
	err := r.db.QueryRow(`
		SELECT id, tenant_id, feature_flag_id, is_enabled, enabled_at, enabled_by
		FROM tenant_feature_flags
		WHERE tenant_id = ? AND feature_flag_id = ?
	`, tenantID, flagID).Scan(&tf.ID, &tf.TenantID, &tf.FeatureFlagID, &tf.IsEnabled, &tf.EnabledAt, &enabledBy)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if enabledBy.Valid {
		userID := int(enabledBy.Int64)
		tf.EnabledBy = &userID
	}
	return tf, nil
}
