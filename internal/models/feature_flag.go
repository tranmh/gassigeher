package models

import (
	"regexp"
	"time"
)

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

// featureFlagKeyPattern validates feature flag keys (alphanumeric + underscores)
var featureFlagKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

// Validate validates CreateFeatureFlagRequest (HIGH-6 fix)
func (r *CreateFeatureFlagRequest) Validate() error {
	// Key is required
	if r.Key == "" {
		return &ValidationError{Field: "key", Message: "Key ist erforderlich"}
	}
	// Key format: lowercase alphanumeric + underscores, 1-64 chars
	if !featureFlagKeyPattern.MatchString(r.Key) {
		return &ValidationError{Field: "key", Message: "Key muss mit Kleinbuchstaben beginnen und darf nur Kleinbuchstaben, Zahlen und Unterstriche enthalten (max 64 Zeichen)"}
	}
	// Name is required
	if r.Name == "" {
		return &ValidationError{Field: "name", Message: "Name ist erforderlich"}
	}
	// Name length limit
	if len(r.Name) > 255 {
		return &ValidationError{Field: "name", Message: "Name darf maximal 255 Zeichen lang sein"}
	}
	// Description length limit (optional but bounded)
	if len(r.Description) > 1000 {
		return &ValidationError{Field: "description", Message: "Beschreibung darf maximal 1000 Zeichen lang sein"}
	}
	return nil
}

// Validate validates UpdateFeatureFlagRequest (HIGH-6 fix)
func (r *UpdateFeatureFlagRequest) Validate() error {
	// Name length limit if provided
	if r.Name != nil && len(*r.Name) > 255 {
		return &ValidationError{Field: "name", Message: "Name darf maximal 255 Zeichen lang sein"}
	}
	// Description length limit if provided
	if r.Description != nil && len(*r.Description) > 1000 {
		return &ValidationError{Field: "description", Message: "Beschreibung darf maximal 1000 Zeichen lang sein"}
	}
	return nil
}
