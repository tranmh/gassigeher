package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
)

// MarketingRepository handles marketing data operations
type MarketingRepository struct {
	db DBExecutor
}

// NewMarketingRepository creates a new marketing repository
func NewMarketingRepository(db DBExecutor) *MarketingRepository {
	return &MarketingRepository{db: db}
}

// ========== Campaigns ==========

// ListCampaigns returns all marketing campaigns
func (r *MarketingRepository) ListCampaigns() ([]*models.MarketingCampaign, error) {
	query := `SELECT id, type, name, description, config, is_active, start_date, end_date, created_at, updated_at
			  FROM marketing_campaigns ORDER BY created_at DESC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var campaigns []*models.MarketingCampaign
	for rows.Next() {
		c := &models.MarketingCampaign{}
		if err := rows.Scan(&c.ID, &c.Type, &c.Name, &c.Description, &c.Config, &c.IsActive, &c.StartDate, &c.EndDate, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		campaigns = append(campaigns, c)
	}

	// HIGH BUG FIX: Check for errors that occurred during iteration
	// rows.Err() returns any error encountered during iteration that wasn't
	// returned by Scan(). This is important for catching connection issues
	// or other database errors that may occur mid-iteration.
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating marketing campaigns: %w", err)
	}

	return campaigns, nil
}

// GetCampaign returns a campaign by ID
func (r *MarketingRepository) GetCampaign(id int) (*models.MarketingCampaign, error) {
	query := `SELECT id, type, name, description, config, is_active, start_date, end_date, created_at, updated_at
			  FROM marketing_campaigns WHERE id = ?`
	c := &models.MarketingCampaign{}
	err := r.db.QueryRow(query, id).Scan(&c.ID, &c.Type, &c.Name, &c.Description, &c.Config, &c.IsActive, &c.StartDate, &c.EndDate, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

// GetActiveCampaignByType returns the active campaign of a specific type
func (r *MarketingRepository) GetActiveCampaignByType(campaignType string) (*models.MarketingCampaign, error) {
	now := FormatTimestamp(time.Now())
	query := `SELECT id, type, name, description, config, is_active, start_date, end_date, created_at, updated_at
			  FROM marketing_campaigns
			  WHERE type = ? AND is_active = ?
			  AND (start_date IS NULL OR start_date <= ?)
			  AND (end_date IS NULL OR end_date >= ?)
			  LIMIT 1`
	c := &models.MarketingCampaign{}
	err := r.db.QueryRow(query, campaignType, r.db.BoolValue(true), now, now).Scan(&c.ID, &c.Type, &c.Name, &c.Description, &c.Config, &c.IsActive, &c.StartDate, &c.EndDate, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

// CreateCampaign creates a new campaign
func (r *MarketingRepository) CreateCampaign(c *models.MarketingCampaign) error {
	now := time.Now()
	nowFormatted := FormatTimestamp(now)

	var startDateFormatted, endDateFormatted *string
	if c.StartDate != nil {
		s := FormatTimestamp(*c.StartDate)
		startDateFormatted = &s
	}
	if c.EndDate != nil {
		e := FormatTimestamp(*c.EndDate)
		endDateFormatted = &e
	}

	query := `INSERT INTO marketing_campaigns (type, name, description, config, is_active, start_date, end_date, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	id, err := r.db.InsertReturningID(query, c.Type, c.Name, c.Description, c.Config, r.db.BoolValue(c.IsActive), startDateFormatted, endDateFormatted, nowFormatted, nowFormatted)
	if err != nil {
		return err
	}
	c.ID = int(id)
	c.CreatedAt = now
	c.UpdatedAt = now
	return nil
}

// UpdateCampaign updates a campaign
func (r *MarketingRepository) UpdateCampaign(c *models.MarketingCampaign) error {
	var startDateFormatted, endDateFormatted *string
	if c.StartDate != nil {
		s := FormatTimestamp(*c.StartDate)
		startDateFormatted = &s
	}
	if c.EndDate != nil {
		e := FormatTimestamp(*c.EndDate)
		endDateFormatted = &e
	}

	query := `UPDATE marketing_campaigns SET name = ?, description = ?, config = ?, is_active = ?, start_date = ?, end_date = ?, updated_at = ?
			  WHERE id = ?`
	_, err := r.db.Exec(query, c.Name, c.Description, c.Config, r.db.BoolValue(c.IsActive), startDateFormatted, endDateFormatted, FormatTimestamp(time.Now()), c.ID)
	return err
}

// DeleteCampaign deletes a campaign
func (r *MarketingRepository) DeleteCampaign(id int) error {
	_, err := r.db.Exec("DELETE FROM marketing_campaigns WHERE id = ?", id)
	return err
}

// ClaimFOMOSlot attempts to atomically claim a slot from an active FOMO campaign.
// Returns the campaign and config if successful, nil if no slots available.
// Uses optimistic locking with retry to handle concurrent slot claims safely.
func (r *MarketingRepository) ClaimFOMOSlot() (*models.MarketingCampaign, *models.FOMOConfig, error) {
	// Retry loop for optimistic locking (handles concurrent claims)
	for attempts := 0; attempts < 3; attempts++ {
		// Get active FOMO campaign
		campaign, err := r.GetActiveCampaignByType("fomo_countdown")
		if err != nil {
			return nil, nil, err
		}
		if campaign == nil {
			return nil, nil, nil // No active FOMO campaign
		}

		// Parse config
		config, err := campaign.GetFOMOConfig()
		if err != nil {
			return nil, nil, err
		}
		if config == nil || config.RemainingSlots <= 0 {
			return nil, nil, nil // No slots available
		}

		// Store original value for optimistic lock
		originalSlots := config.RemainingSlots

		// Decrement remaining slots
		config.RemainingSlots--

		// Auto-deactivate campaign if no more slots
		if config.RemainingSlots <= 0 {
			campaign.IsActive = false
		}

		// Update config on campaign
		if err := campaign.SetFOMOConfig(config); err != nil {
			return nil, nil, err
		}

		// Try to update with optimistic locking
		updated, err := r.UpdateCampaignOptimistic(campaign, originalSlots)
		if err != nil {
			return nil, nil, err
		}
		if updated {
			return campaign, config, nil
		}
		// Concurrent update detected, retry
	}

	// Failed after max retries (rare edge case)
	return nil, nil, nil
}

// UpdateCampaignOptimistic updates a campaign only if remaining_slots matches expected value.
// Returns true if update succeeded, false if concurrent modification detected.
// This implements optimistic locking to prevent race conditions in slot claiming.
func (r *MarketingRepository) UpdateCampaignOptimistic(c *models.MarketingCampaign, expectedSlots int) (bool, error) {
	var startDateFormatted, endDateFormatted *string
	if c.StartDate != nil {
		s := FormatTimestamp(*c.StartDate)
		startDateFormatted = &s
	}
	if c.EndDate != nil {
		e := FormatTimestamp(*c.EndDate)
		endDateFormatted = &e
	}

	// Use LIKE patterns to check remaining_slots value in JSON config
	// Must match exact number to avoid false positives (e.g., 30 matching 300)
	// JSON can have: "remaining_slots":30, or "remaining_slots":30}
	patternWithComma := fmt.Sprintf("%%\"remaining_slots\":%d,%%", expectedSlots)
	patternWithBrace := fmt.Sprintf("%%\"remaining_slots\":%d}%%", expectedSlots)

	query := `UPDATE marketing_campaigns
			  SET name = ?, description = ?, config = ?, is_active = ?, start_date = ?, end_date = ?, updated_at = ?
			  WHERE id = ? AND (config LIKE ? OR config LIKE ?)`

	result, err := r.db.Exec(query, c.Name, c.Description, c.Config, c.IsActive,
		startDateFormatted, endDateFormatted, FormatTimestamp(time.Now()), c.ID, patternWithComma, patternWithBrace)
	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return rowsAffected > 0, nil
}

// ========== Referral Codes ==========

// ListReferralCodes returns all referral codes
func (r *MarketingRepository) ListReferralCodes() ([]*models.ReferralCode, error) {
	query := `SELECT rc.id, rc.code, rc.referrer_tenant_id, rc.referrer_email,
			  rc.discount_months_referrer, rc.discount_months_referee, rc.uses_count, rc.max_uses,
			  rc.is_active, rc.expires_at, rc.created_at, rc.updated_at, t.name
			  FROM referral_codes rc
			  LEFT JOIN tenants t ON rc.referrer_tenant_id = t.id
			  ORDER BY rc.created_at DESC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var codes []*models.ReferralCode
	for rows.Next() {
		c := &models.ReferralCode{}
		if err := rows.Scan(&c.ID, &c.Code, &c.ReferrerTenantID, &c.ReferrerEmail,
			&c.DiscountMonthsReferrer, &c.DiscountMonthsReferee, &c.UsesCount, &c.MaxUses,
			&c.IsActive, &c.ExpiresAt, &c.CreatedAt, &c.UpdatedAt, &c.ReferrerTenantName); err != nil {
			return nil, err
		}
		codes = append(codes, c)
	}
	return codes, nil
}

// GetReferralCode returns a referral code by ID
func (r *MarketingRepository) GetReferralCode(id int) (*models.ReferralCode, error) {
	query := `SELECT id, code, referrer_tenant_id, referrer_email, discount_months_referrer, discount_months_referee,
			  uses_count, max_uses, is_active, expires_at, created_at, updated_at
			  FROM referral_codes WHERE id = ?`
	c := &models.ReferralCode{}
	err := r.db.QueryRow(query, id).Scan(&c.ID, &c.Code, &c.ReferrerTenantID, &c.ReferrerEmail,
		&c.DiscountMonthsReferrer, &c.DiscountMonthsReferee, &c.UsesCount, &c.MaxUses,
		&c.IsActive, &c.ExpiresAt, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

// GetReferralCodeByCode returns a referral code by its code string
func (r *MarketingRepository) GetReferralCodeByCode(code string) (*models.ReferralCode, error) {
	query := `SELECT id, code, referrer_tenant_id, referrer_email, discount_months_referrer, discount_months_referee,
			  uses_count, max_uses, is_active, expires_at, created_at, updated_at
			  FROM referral_codes WHERE code = ?`
	c := &models.ReferralCode{}
	err := r.db.QueryRow(query, code).Scan(&c.ID, &c.Code, &c.ReferrerTenantID, &c.ReferrerEmail,
		&c.DiscountMonthsReferrer, &c.DiscountMonthsReferee, &c.UsesCount, &c.MaxUses,
		&c.IsActive, &c.ExpiresAt, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

// CreateReferralCode creates a new referral code
func (r *MarketingRepository) CreateReferralCode(c *models.ReferralCode) error {
	now := time.Now()
	nowFormatted := FormatTimestamp(now)

	var expiresAtFormatted *string
	if c.ExpiresAt != nil {
		e := FormatTimestamp(*c.ExpiresAt)
		expiresAtFormatted = &e
	}

	query := `INSERT INTO referral_codes (code, referrer_tenant_id, referrer_email, discount_months_referrer,
			  discount_months_referee, max_uses, is_active, expires_at, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	id, err := r.db.InsertReturningID(query, c.Code, c.ReferrerTenantID, c.ReferrerEmail, c.DiscountMonthsReferrer,
		c.DiscountMonthsReferee, c.MaxUses, r.db.BoolValue(c.IsActive), expiresAtFormatted, nowFormatted, nowFormatted)
	if err != nil {
		return fmt.Errorf("failed to create referral code: %w", err)
	}
	c.ID = int(id)
	c.CreatedAt = now
	c.UpdatedAt = now
	return nil
}

// UpdateReferralCode updates a referral code
func (r *MarketingRepository) UpdateReferralCode(c *models.ReferralCode) error {
	var expiresAtFormatted *string
	if c.ExpiresAt != nil {
		e := FormatTimestamp(*c.ExpiresAt)
		expiresAtFormatted = &e
	}

	query := `UPDATE referral_codes SET code = ?, referrer_email = ?, discount_months_referrer = ?,
			  discount_months_referee = ?, max_uses = ?, is_active = ?, expires_at = ?, updated_at = ?
			  WHERE id = ?`
	_, err := r.db.Exec(query, c.Code, c.ReferrerEmail, c.DiscountMonthsReferrer, c.DiscountMonthsReferee,
		c.MaxUses, r.db.BoolValue(c.IsActive), expiresAtFormatted, FormatTimestamp(time.Now()), c.ID)
	return err
}

// ErrReferralCodeMaxUsesReached is returned when a referral code has reached its max_uses limit
var ErrReferralCodeMaxUsesReached = fmt.Errorf("referral code has reached max uses")

// IncrementReferralCodeUses atomically increments the use count for a referral code
// BUG #6 FIX: Only increments if under max_uses limit (or unlimited when max_uses is NULL)
// This prevents race conditions where multiple concurrent requests could exceed the limit
func (r *MarketingRepository) IncrementReferralCodeUses(id int) error {
	// Atomic conditional update: only increment if under limit
	query := `UPDATE referral_codes
		SET uses_count = uses_count + 1, updated_at = ?
		WHERE id = ?
		AND (max_uses IS NULL OR uses_count < max_uses)`

	result, err := r.db.Exec(query, FormatTimestamp(time.Now()), id)
	if err != nil {
		return fmt.Errorf("failed to increment referral code uses: %w", err)
	}

	// Check if update actually happened
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		// Either referral code doesn't exist, or max_uses reached
		return ErrReferralCodeMaxUsesReached
	}

	return nil
}

// DeleteReferralCode deletes a referral code
func (r *MarketingRepository) DeleteReferralCode(id int) error {
	_, err := r.db.Exec("DELETE FROM referral_codes WHERE id = ?", id)
	return err
}

// ========== Referral Uses ==========

// RecordReferralUse records the use of a referral code
func (r *MarketingRepository) RecordReferralUse(codeID, refereeTenantID int) error {
	query := `INSERT INTO referral_uses (code_id, referee_tenant_id, applied_at) VALUES (?, ?, ?)`
	_, err := r.db.Exec(query, codeID, refereeTenantID, FormatTimestamp(time.Now()))
	return err
}

// GetReferralUses returns all uses for a referral code
func (r *MarketingRepository) GetReferralUses(codeID int) ([]*models.ReferralUse, error) {
	query := `SELECT ru.id, ru.code_id, ru.referee_tenant_id, ru.applied_at, rc.code, t.name
			  FROM referral_uses ru
			  JOIN referral_codes rc ON ru.code_id = rc.id
			  JOIN tenants t ON ru.referee_tenant_id = t.id
			  WHERE ru.code_id = ?
			  ORDER BY ru.applied_at DESC`
	rows, err := r.db.Query(query, codeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var uses []*models.ReferralUse
	for rows.Next() {
		u := &models.ReferralUse{}
		if err := rows.Scan(&u.ID, &u.CodeID, &u.RefereeTenantID, &u.AppliedAt, &u.Code, &u.RefereeTenantName); err != nil {
			return nil, err
		}
		uses = append(uses, u)
	}
	return uses, nil
}

// HasTenantUsedReferral checks if a tenant has already used a referral code
func (r *MarketingRepository) HasTenantUsedReferral(tenantID int) (bool, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM referral_uses WHERE referee_tenant_id = ?", tenantID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ========== Reference Entries ==========

// ListReferenceEntries returns reference entries (optionally filtered by approval status)
func (r *MarketingRepository) ListReferenceEntries(approvedOnly bool) ([]*models.ReferenceEntry, error) {
	var rows *sql.Rows
	var err error
	if approvedOnly {
		query := `SELECT id, tenant_id, display_name, city, federal_state, website_url, testimonial, logo_url, is_approved, display_order, created_at, updated_at
			  FROM reference_entries WHERE is_approved = ? ORDER BY display_order ASC, display_name ASC`
		rows, err = r.db.Query(query, r.db.BoolValue(true))
	} else {
		query := `SELECT id, tenant_id, display_name, city, federal_state, website_url, testimonial, logo_url, is_approved, display_order, created_at, updated_at
			  FROM reference_entries ORDER BY display_order ASC, display_name ASC`
		rows, err = r.db.Query(query)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*models.ReferenceEntry
	for rows.Next() {
		e := &models.ReferenceEntry{}
		if err := rows.Scan(&e.ID, &e.TenantID, &e.DisplayName, &e.City, &e.FederalState, &e.WebsiteURL, &e.Testimonial, &e.LogoURL, &e.IsApproved, &e.DisplayOrder, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// GetReferenceEntry returns a reference entry by ID
func (r *MarketingRepository) GetReferenceEntry(id int) (*models.ReferenceEntry, error) {
	query := `SELECT id, tenant_id, display_name, city, federal_state, website_url, testimonial, logo_url, is_approved, display_order, created_at, updated_at
			  FROM reference_entries WHERE id = ?`
	e := &models.ReferenceEntry{}
	err := r.db.QueryRow(query, id).Scan(&e.ID, &e.TenantID, &e.DisplayName, &e.City, &e.FederalState, &e.WebsiteURL, &e.Testimonial, &e.LogoURL, &e.IsApproved, &e.DisplayOrder, &e.CreatedAt, &e.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return e, nil
}

// GetReferenceEntryByTenant returns a reference entry by tenant ID
func (r *MarketingRepository) GetReferenceEntryByTenant(tenantID int) (*models.ReferenceEntry, error) {
	query := `SELECT id, tenant_id, display_name, city, federal_state, website_url, testimonial, logo_url, is_approved, display_order, created_at, updated_at
			  FROM reference_entries WHERE tenant_id = ?`
	e := &models.ReferenceEntry{}
	err := r.db.QueryRow(query, tenantID).Scan(&e.ID, &e.TenantID, &e.DisplayName, &e.City, &e.FederalState, &e.WebsiteURL, &e.Testimonial, &e.LogoURL, &e.IsApproved, &e.DisplayOrder, &e.CreatedAt, &e.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return e, nil
}

// CreateReferenceEntry creates a new reference entry
func (r *MarketingRepository) CreateReferenceEntry(e *models.ReferenceEntry) error {
	now := FormatTimestamp(time.Now())
	query := `INSERT INTO reference_entries (tenant_id, display_name, city, federal_state, website_url, testimonial, logo_url, is_approved, display_order, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	id, err := r.db.InsertReturningID(query, e.TenantID, e.DisplayName, e.City, e.FederalState, e.WebsiteURL, e.Testimonial, e.LogoURL, r.db.BoolValue(e.IsApproved), e.DisplayOrder, now, now)
	if err != nil {
		return fmt.Errorf("failed to create reference entry: %w", err)
	}
	e.ID = int(id)
	return nil
}

// UpdateReferenceEntry updates a reference entry
func (r *MarketingRepository) UpdateReferenceEntry(e *models.ReferenceEntry) error {
	query := `UPDATE reference_entries SET display_name = ?, city = ?, federal_state = ?, website_url = ?, testimonial = ?, logo_url = ?, is_approved = ?, display_order = ?, updated_at = ?
			  WHERE id = ?`
	_, err := r.db.Exec(query, e.DisplayName, e.City, e.FederalState, e.WebsiteURL, e.Testimonial, e.LogoURL, r.db.BoolValue(e.IsApproved), e.DisplayOrder, FormatTimestamp(time.Now()), e.ID)
	return err
}

// ApproveReferenceEntry approves a reference entry
func (r *MarketingRepository) ApproveReferenceEntry(id int) error {
	_, err := r.db.Exec("UPDATE reference_entries SET is_approved = ?, updated_at = ? WHERE id = ?", r.db.BoolValue(true), FormatTimestamp(time.Now()), id)
	return err
}

// DeleteReferenceEntry deletes a reference entry
func (r *MarketingRepository) DeleteReferenceEntry(id int) error {
	_, err := r.db.Exec("DELETE FROM reference_entries WHERE id = ?", id)
	return err
}

// ========== Stats ==========

// GetMarketingStats returns marketing statistics
func (r *MarketingRepository) GetMarketingStats() (*models.MarketingStatsResponse, error) {
	stats := &models.MarketingStatsResponse{}

	// Active campaigns
	r.db.QueryRow("SELECT COUNT(*) FROM marketing_campaigns WHERE is_active = ?", r.db.BoolValue(true)).Scan(&stats.ActiveCampaigns)

	// Total referral codes
	r.db.QueryRow("SELECT COUNT(*) FROM referral_codes").Scan(&stats.TotalReferralCodes)

	// Total referral uses
	r.db.QueryRow("SELECT COUNT(*) FROM referral_uses").Scan(&stats.TotalReferralUses)

	// Approved references
	r.db.QueryRow("SELECT COUNT(*) FROM reference_entries WHERE is_approved = ?", r.db.BoolValue(true)).Scan(&stats.ApprovedReferences)

	// Pending references
	r.db.QueryRow("SELECT COUNT(*) FROM reference_entries WHERE is_approved = ?", r.db.BoolValue(false)).Scan(&stats.PendingReferences)

	return stats, nil
}
