package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
)

// DemoTenantRepository handles demo tenant state database operations
type DemoTenantRepository struct {
	db DBExecutor
}

// NewDemoTenantRepository creates a new demo tenant repository
func NewDemoTenantRepository(db DBExecutor) *DemoTenantRepository {
	return &DemoTenantRepository{db: db}
}

// GetState retrieves the demo state for a tenant
func (r *DemoTenantRepository) GetState(tenantID int) (*models.DemoTenantState, error) {
	query := `
		SELECT id, tenant_id, admin_password, last_reset_at, next_reset_at, created_at, updated_at
		FROM demo_tenant_state
		WHERE tenant_id = ?
	`

	state := &models.DemoTenantState{}
	err := r.db.QueryRow(query, tenantID).Scan(
		&state.ID,
		&state.TenantID,
		&state.AdminPassword,
		&state.LastResetAt,
		&state.NextResetAt,
		&state.CreatedAt,
		&state.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get demo state: %w", err)
	}

	return state, nil
}

// CreateState creates the initial demo state
func (r *DemoTenantRepository) CreateState(state *models.DemoTenantState) error {
	query := `
		INSERT INTO demo_tenant_state (tenant_id, admin_password, last_reset_at, next_reset_at)
		VALUES (?, ?, ?, ?)
	`

	id, err := r.db.InsertReturningID(
		query,
		state.TenantID,
		state.AdminPassword,
		state.LastResetAt,
		state.NextResetAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create demo state: %w", err)
	}

	state.ID = int(id)
	state.CreatedAt = time.Now()
	state.UpdatedAt = time.Now()
	return nil
}

// UpdateState updates the demo state after a reset
func (r *DemoTenantRepository) UpdateState(tenantID int, password string, lastReset, nextReset *time.Time) error {
	query := `
		UPDATE demo_tenant_state SET
			admin_password = ?,
			last_reset_at = ?,
			next_reset_at = ?,
			updated_at = ?
		WHERE tenant_id = ?
	`

	now := time.Now()
	_, err := r.db.Exec(query, password, lastReset, nextReset, now, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update demo state: %w", err)
	}

	return nil
}

// GetCredentials retrieves formatted credentials for display
func (r *DemoTenantRepository) GetCredentials(tenantID int) (*models.DemoCredentials, error) {
	query := `
		SELECT dts.admin_password, dts.last_reset_at, dts.next_reset_at, u.email
		FROM demo_tenant_state dts
		JOIN users u ON u.tenant_id = dts.tenant_id AND u.is_super_admin = ?
		WHERE dts.tenant_id = ?
		LIMIT 1
	`

	var password string
	var lastReset, nextReset *time.Time
	var email string

	err := r.db.QueryRow(query, r.db.BoolValue(true), tenantID).Scan(&password, &lastReset, &nextReset, &email)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	credentials := &models.DemoCredentials{
		AdminEmail:    email,
		AdminPassword: password,
	}

	// Format times for display (German format)
	if lastReset != nil {
		credentials.LastResetAt = lastReset.Format("02.01.2006 15:04")
	}
	if nextReset != nil {
		credentials.NextResetAt = nextReset.Format("02.01.2006 15:04")
	}

	return credentials, nil
}

// DeleteState removes the demo state for a tenant
func (r *DemoTenantRepository) DeleteState(tenantID int) error {
	query := `DELETE FROM demo_tenant_state WHERE tenant_id = ?`
	_, err := r.db.Exec(query, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete demo state: %w", err)
	}
	return nil
}
