package models

import "time"

// FeatureFlag represents a feature toggle for gradual rollout
type FeatureFlag struct {
	ID          int        `json:"id"`
	Key         string     `json:"key"`          // Unique flag identifier (e.g., "new_booking_ui", "experimental_search")
	Name        string     `json:"name"`         // Human-readable name
	Description string     `json:"description"`  // What this flag controls
	IsGlobal    bool       `json:"is_global"`    // True = applies to all tenants, False = per-tenant
	IsEnabled   bool       `json:"is_enabled"`   // Global enable/disable state
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// TenantFeatureFlag represents a per-tenant feature flag override
type TenantFeatureFlag struct {
	ID            int       `json:"id"`
	TenantID      int       `json:"tenant_id"`
	FeatureFlagID int       `json:"feature_flag_id"`
	IsEnabled     bool      `json:"is_enabled"`
	EnabledAt     time.Time `json:"enabled_at"`
	EnabledBy     *int      `json:"enabled_by,omitempty"` // User ID who enabled/disabled
}

// FeatureFlagWithStatus includes the flag with its effective status for a tenant
type FeatureFlagWithStatus struct {
	FeatureFlag
	TenantEnabled     *bool      `json:"tenant_enabled,omitempty"`     // Tenant-specific override
	EffectiveEnabled  bool       `json:"effective_enabled"`            // What's actually active
	TenantEnabledAt   *time.Time `json:"tenant_enabled_at,omitempty"`
}

// Common feature flag keys (defined as constants for type safety)
const (
	FeatureFlagNewBookingUI     = "new_booking_ui"
	FeatureFlagAdvancedSearch   = "advanced_search"
	FeatureFlagBulkOperations   = "bulk_operations"
	FeatureFlagExperimentalAPI  = "experimental_api"
	FeatureFlagDarkMode         = "dark_mode"
	FeatureFlagMobileApp        = "mobile_app_integration"
	FeatureFlagCalendarSync     = "calendar_sync"
	FeatureFlagSMSNotifications = "sms_notifications"
)

// CreateFeatureFlagRequest represents the request to create a feature flag
type CreateFeatureFlagRequest struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsGlobal    bool   `json:"is_global"`
	IsEnabled   bool   `json:"is_enabled"`
}

// UpdateFeatureFlagRequest represents the request to update a feature flag
type UpdateFeatureFlagRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	IsGlobal    *bool   `json:"is_global,omitempty"`
	IsEnabled   *bool   `json:"is_enabled,omitempty"`
}

// SetTenantFeatureFlagRequest represents the request to set a tenant's feature flag
type SetTenantFeatureFlagRequest struct {
	IsEnabled bool `json:"is_enabled"`
}
