package models

import (
	"encoding/json"
	"time"
)

// MarketingCampaign represents a marketing campaign
type MarketingCampaign struct {
	ID          int        `json:"id"`
	Type        string     `json:"type"` // fomo_countdown, referral, reference_page, custom
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	Config      *string    `json:"config,omitempty"` // JSON config
	IsActive    bool       `json:"is_active"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// FOMOConfig holds configuration for FOMO countdown campaigns
type FOMOConfig struct {
	TotalSlots     int    `json:"total_slots"`     // e.g., 30 free Pro slots
	RemainingSlots int    `json:"remaining_slots"` // Updated as slots are claimed
	Message        string `json:"message"`         // e.g., "Nur noch X Plätze kostenlos!"
	CTAText        string `json:"cta_text"`        // Call to action text
	CTALink        string `json:"cta_link"`        // Link for CTA button
	// Benefit granted to early tenants who claim a slot
	BenefitType   string `json:"benefit_type,omitempty"`   // "free_pro_months" for free Pro subscription
	BenefitMonths int    `json:"benefit_months,omitempty"` // Number of free months (e.g., 12 for 1 year)
}

// GetFOMOConfig parses the JSON config into FOMOConfig
func (c *MarketingCampaign) GetFOMOConfig() (*FOMOConfig, error) {
	if c.Config == nil {
		return nil, nil
	}
	var config FOMOConfig
	if err := json.Unmarshal([]byte(*c.Config), &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// SetFOMOConfig serializes FOMOConfig to JSON and sets it
func (c *MarketingCampaign) SetFOMOConfig(config *FOMOConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	str := string(data)
	c.Config = &str
	return nil
}

// ReferralCode represents a referral code
type ReferralCode struct {
	ID                     int        `json:"id"`
	Code                   string     `json:"code"`
	ReferrerTenantID       *int       `json:"referrer_tenant_id,omitempty"`
	ReferrerEmail          *string    `json:"referrer_email,omitempty"`
	DiscountMonthsReferrer int        `json:"discount_months_referrer"` // Months free for referrer
	DiscountMonthsReferee  int        `json:"discount_months_referee"`  // Months free for referee
	UsesCount              int        `json:"uses_count"`
	MaxUses                *int       `json:"max_uses,omitempty"`
	IsActive               bool       `json:"is_active"`
	ExpiresAt              *time.Time `json:"expires_at,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`

	// Computed fields (not stored in DB)
	ReferrerTenantName *string `json:"referrer_tenant_name,omitempty"`
}

// IsValid checks if the referral code can be used
func (r *ReferralCode) IsValid() bool {
	if !r.IsActive {
		return false
	}
	if r.MaxUses != nil && r.UsesCount >= *r.MaxUses {
		return false
	}
	if r.ExpiresAt != nil && time.Now().After(*r.ExpiresAt) {
		return false
	}
	return true
}

// ReferralUse tracks when a referral code is used
type ReferralUse struct {
	ID              int       `json:"id"`
	CodeID          int       `json:"code_id"`
	RefereeTenantID int       `json:"referee_tenant_id"`
	AppliedAt       time.Time `json:"applied_at"`

	// Joined fields
	Code             *string `json:"code,omitempty"`
	RefereeTenantName *string `json:"referee_tenant_name,omitempty"`
}

// ReferenceEntry represents a shelter entry on the public reference page
type ReferenceEntry struct {
	ID           int       `json:"id"`
	TenantID     int       `json:"tenant_id"`
	DisplayName  string    `json:"display_name"`
	City         *string   `json:"city,omitempty"`
	FederalState *string   `json:"federal_state,omitempty"`
	WebsiteURL   *string   `json:"website_url,omitempty"`
	Testimonial  *string   `json:"testimonial,omitempty"`
	LogoURL      *string   `json:"logo_url,omitempty"`
	IsApproved   bool      `json:"is_approved"`
	DisplayOrder int       `json:"display_order"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Request/Response types

// CreateReferralCodeRequest is the request to create a referral code
type CreateReferralCodeRequest struct {
	Code                   string  `json:"code"`
	ReferrerEmail          *string `json:"referrer_email,omitempty"`
	DiscountMonthsReferrer int     `json:"discount_months_referrer"`
	DiscountMonthsReferee  int     `json:"discount_months_referee"`
	MaxUses                *int    `json:"max_uses,omitempty"`
	ExpiresAt              *string `json:"expires_at,omitempty"` // ISO date string
}

// ApplyReferralCodeRequest is the request to apply a referral code
type ApplyReferralCodeRequest struct {
	Code string `json:"code"`
}

// UpdateCampaignRequest is the request to update a campaign
type UpdateCampaignRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Config      *string `json:"config,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
	StartDate   *string `json:"start_date,omitempty"`
	EndDate     *string `json:"end_date,omitempty"`
}

// CreateReferenceEntryRequest is the request to create/update a reference entry
type CreateReferenceEntryRequest struct {
	DisplayName  string  `json:"display_name"`
	City         *string `json:"city,omitempty"`
	FederalState *string `json:"federal_state,omitempty"`
	WebsiteURL   *string `json:"website_url,omitempty"`
	Testimonial  *string `json:"testimonial,omitempty"`
}

// MarketingStatsResponse contains marketing statistics
type MarketingStatsResponse struct {
	ActiveCampaigns     int `json:"active_campaigns"`
	TotalReferralCodes  int `json:"total_referral_codes"`
	TotalReferralUses   int `json:"total_referral_uses"`
	ApprovedReferences  int `json:"approved_references"`
	PendingReferences   int `json:"pending_references"`
}
