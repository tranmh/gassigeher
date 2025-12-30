package repository

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
)

// PromoCodeRepository handles promo code database operations
type PromoCodeRepository struct {
	db *sql.DB
}

// NewPromoCodeRepository creates a new promo code repository
func NewPromoCodeRepository(db *sql.DB) *PromoCodeRepository {
	return &PromoCodeRepository{db: db}
}

// Create creates a new promo code
func (r *PromoCodeRepository) Create(code *models.PromoCode) error {
	query := `
		INSERT INTO promo_codes (
			code, description, discount_type, discount_value, max_uses,
			uses_count, valid_for_plans, is_active, stripe_coupon_id, expires_at,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	var expiresAt interface{}
	if code.ExpiresAt != nil {
		expiresAt = FormatTimestamp(*code.ExpiresAt)
	}

	result, err := r.db.Exec(query,
		strings.ToUpper(code.Code),
		code.Description,
		code.DiscountType,
		code.DiscountValue,
		code.MaxUses,
		code.UsesCount,
		code.ValidForPlans,
		code.IsActive,
		code.StripeCouponID,
		expiresAt,
		FormatTimestamp(now),
		FormatTimestamp(now),
	)
	if err != nil {
		return fmt.Errorf("failed to create promo code: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get promo code ID: %w", err)
	}
	code.ID = int(id)
	code.CreatedAt = now
	code.UpdatedAt = now

	return nil
}

// GetByID returns a promo code by ID
func (r *PromoCodeRepository) GetByID(id int) (*models.PromoCode, error) {
	query := `
		SELECT id, code, description, discount_type, discount_value, max_uses,
		       uses_count, valid_for_plans, is_active, stripe_coupon_id, expires_at,
		       created_at, updated_at
		FROM promo_codes
		WHERE id = ?
	`

	code := &models.PromoCode{}
	var expiresAt sql.NullString
	var description, validForPlans, stripeCouponID sql.NullString
	var maxUses sql.NullInt64

	err := r.db.QueryRow(query, id).Scan(
		&code.ID,
		&code.Code,
		&description,
		&code.DiscountType,
		&code.DiscountValue,
		&maxUses,
		&code.UsesCount,
		&validForPlans,
		&code.IsActive,
		&stripeCouponID,
		&expiresAt,
		&code.CreatedAt,
		&code.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get promo code: %w", err)
	}

	if description.Valid {
		code.Description = description.String
	}
	if validForPlans.Valid {
		code.ValidForPlans = validForPlans.String
	}
	if stripeCouponID.Valid {
		code.StripeCouponID = &stripeCouponID.String
	}
	if maxUses.Valid {
		maxUsesInt := int(maxUses.Int64)
		code.MaxUses = &maxUsesInt
	}
	if expiresAt.Valid {
		t, err := time.Parse(time.RFC3339, expiresAt.String)
		if err != nil {
			log.Printf("Warning: Failed to parse expires_at for promo code %d: %v", code.ID, err)
		} else {
			code.ExpiresAt = &t
		}
	}

	return code, nil
}

// GetByCode returns a promo code by its code string
func (r *PromoCodeRepository) GetByCode(codeStr string) (*models.PromoCode, error) {
	query := `
		SELECT id, code, description, discount_type, discount_value, max_uses,
		       uses_count, valid_for_plans, is_active, stripe_coupon_id, expires_at,
		       created_at, updated_at
		FROM promo_codes
		WHERE code = ?
	`

	code := &models.PromoCode{}
	var expiresAt sql.NullString
	var description, validForPlans, stripeCouponID sql.NullString
	var maxUses sql.NullInt64

	err := r.db.QueryRow(query, strings.ToUpper(codeStr)).Scan(
		&code.ID,
		&code.Code,
		&description,
		&code.DiscountType,
		&code.DiscountValue,
		&maxUses,
		&code.UsesCount,
		&validForPlans,
		&code.IsActive,
		&stripeCouponID,
		&expiresAt,
		&code.CreatedAt,
		&code.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get promo code by code: %w", err)
	}

	if description.Valid {
		code.Description = description.String
	}
	if validForPlans.Valid {
		code.ValidForPlans = validForPlans.String
	}
	if stripeCouponID.Valid {
		code.StripeCouponID = &stripeCouponID.String
	}
	if maxUses.Valid {
		maxUsesInt := int(maxUses.Int64)
		code.MaxUses = &maxUsesInt
	}
	if expiresAt.Valid {
		t, err := time.Parse(time.RFC3339, expiresAt.String)
		if err != nil {
			log.Printf("Warning: Failed to parse expires_at for promo code %s: %v", codeStr, err)
		} else {
			code.ExpiresAt = &t
		}
	}

	return code, nil
}

// GetAll returns all promo codes with optional filtering
func (r *PromoCodeRepository) GetAll(activeOnly bool) ([]*models.PromoCode, error) {
	query := `
		SELECT id, code, description, discount_type, discount_value, max_uses,
		       uses_count, valid_for_plans, is_active, stripe_coupon_id, expires_at,
		       created_at, updated_at
		FROM promo_codes
	`
	if activeOnly {
		query += " WHERE is_active = 1"
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query promo codes: %w", err)
	}
	defer rows.Close()

	codes := []*models.PromoCode{}
	for rows.Next() {
		code := &models.PromoCode{}
		var expiresAt sql.NullString
		var description, validForPlans, stripeCouponID sql.NullString
		var maxUses sql.NullInt64

		err := rows.Scan(
			&code.ID,
			&code.Code,
			&description,
			&code.DiscountType,
			&code.DiscountValue,
			&maxUses,
			&code.UsesCount,
			&validForPlans,
			&code.IsActive,
			&stripeCouponID,
			&expiresAt,
			&code.CreatedAt,
			&code.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan promo code: %w", err)
		}

		if description.Valid {
			code.Description = description.String
		}
		if validForPlans.Valid {
			code.ValidForPlans = validForPlans.String
		}
		if stripeCouponID.Valid {
			code.StripeCouponID = &stripeCouponID.String
		}
		if maxUses.Valid {
			maxUsesInt := int(maxUses.Int64)
			code.MaxUses = &maxUsesInt
		}
		if expiresAt.Valid {
			t, err := time.Parse(time.RFC3339, expiresAt.String)
			if err != nil {
				log.Printf("Warning: Failed to parse expires_at for promo code %d: %v", code.ID, err)
			} else {
				code.ExpiresAt = &t
			}
		}

		codes = append(codes, code)
	}

	return codes, nil
}

// Update updates a promo code
func (r *PromoCodeRepository) Update(code *models.PromoCode) error {
	query := `
		UPDATE promo_codes
		SET code = ?, description = ?, discount_type = ?, discount_value = ?,
		    max_uses = ?, valid_for_plans = ?, is_active = ?, stripe_coupon_id = ?,
		    expires_at = ?, updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	var expiresAt interface{}
	if code.ExpiresAt != nil {
		expiresAt = FormatTimestamp(*code.ExpiresAt)
	}

	_, err := r.db.Exec(query,
		strings.ToUpper(code.Code),
		code.Description,
		code.DiscountType,
		code.DiscountValue,
		code.MaxUses,
		code.ValidForPlans,
		code.IsActive,
		code.StripeCouponID,
		expiresAt,
		FormatTimestamp(now),
		code.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update promo code: %w", err)
	}

	code.UpdatedAt = now
	return nil
}

// Delete deletes a promo code
func (r *PromoCodeRepository) Delete(id int) error {
	_, err := r.db.Exec("DELETE FROM promo_codes WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete promo code: %w", err)
	}
	return nil
}

// ErrPromoCodeMaxUsesReached is returned when a promo code has reached its max_uses limit
var ErrPromoCodeMaxUsesReached = fmt.Errorf("promo code has reached max uses")

// IncrementUsesCount atomically increments the uses count for a promo code
// BUG #5 FIX: Only increments if under max_uses limit (or unlimited when max_uses is NULL)
// This prevents race conditions where multiple concurrent requests could exceed the limit
func (r *PromoCodeRepository) IncrementUsesCount(id int) error {
	// Atomic conditional update: only increment if under limit
	query := `UPDATE promo_codes
		SET uses_count = uses_count + 1, updated_at = ?
		WHERE id = ?
		AND (max_uses IS NULL OR uses_count < max_uses)`

	result, err := r.db.Exec(query, FormatTimestamp(time.Now()), id)
	if err != nil {
		return fmt.Errorf("failed to increment promo code uses: %w", err)
	}

	// Check if update actually happened
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		// Either promo code doesn't exist, or max_uses reached
		return ErrPromoCodeMaxUsesReached
	}

	return nil
}

// RecordUse records when a tenant uses a promo code
func (r *PromoCodeRepository) RecordUse(promoCodeID, tenantID int) error {
	query := `INSERT INTO promo_code_uses (promo_code_id, tenant_id, applied_at) VALUES (?, ?, ?)`
	_, err := r.db.Exec(query, promoCodeID, tenantID, FormatTimestamp(time.Now()))
	if err != nil {
		return fmt.Errorf("failed to record promo code use: %w", err)
	}
	return nil
}

// HasTenantUsedCode checks if a tenant has already used a promo code
func (r *PromoCodeRepository) HasTenantUsedCode(promoCodeID, tenantID int) (bool, error) {
	query := `SELECT COUNT(*) FROM promo_code_uses WHERE promo_code_id = ? AND tenant_id = ?`
	var count int
	err := r.db.QueryRow(query, promoCodeID, tenantID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check promo code use: %w", err)
	}
	return count > 0, nil
}

// GetCodeUses returns the usage history for a promo code
func (r *PromoCodeRepository) GetCodeUses(promoCodeID int) ([]*models.PromoCodeUse, error) {
	query := `
		SELECT pcu.id, pcu.promo_code_id, pcu.tenant_id, pcu.applied_at,
		       pc.code, t.name
		FROM promo_code_uses pcu
		JOIN promo_codes pc ON pcu.promo_code_id = pc.id
		JOIN tenants t ON pcu.tenant_id = t.id
		WHERE pcu.promo_code_id = ?
		ORDER BY pcu.applied_at DESC
	`

	rows, err := r.db.Query(query, promoCodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query promo code uses: %w", err)
	}
	defer rows.Close()

	uses := []*models.PromoCodeUse{}
	for rows.Next() {
		use := &models.PromoCodeUse{}
		var code, tenantName sql.NullString

		err := rows.Scan(
			&use.ID,
			&use.PromoCodeID,
			&use.TenantID,
			&use.AppliedAt,
			&code,
			&tenantName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan promo code use: %w", err)
		}

		if code.Valid {
			use.Code = &code.String
		}
		if tenantName.Valid {
			use.TenantName = &tenantName.String
		}

		uses = append(uses, use)
	}

	return uses, nil
}
