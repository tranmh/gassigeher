package models

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// SystemSetting represents a system configuration setting
type SystemSetting struct {
	TenantID  int       `json:"tenant_id,omitempty"` // SaaS: Tenant this setting belongs to (0 = global)
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UpdateSettingRequest represents a request to update a setting
type UpdateSettingRequest struct {
	Value string `json:"value"`
}

// Validate validates the update setting request
func (r *UpdateSettingRequest) Validate() error {
	if r.Value == "" {
		return &ValidationError{Field: "value", Message: "Value is required"}
	}

	return nil
}

// SettingConstraint defines the validation constraints for a setting
type SettingConstraint struct {
	Type     string // "integer", "boolean", "string", "whatsapp_link", "registration_password"
	Min      int    // For integer type: minimum value
	Max      int    // For integer type: maximum value
	Required bool   // Whether empty value is invalid
}

// settingConstraints defines all known settings and their validation rules
var settingConstraints = map[string]SettingConstraint{
	"booking_advance_days": {
		Type:     "integer",
		Min:      1,
		Max:      365, // Can't book more than 1 year in advance
		Required: true,
	},
	"cancellation_notice_hours": {
		Type:     "integer",
		Min:      1,
		Max:      168, // Max 1 week (168 hours) notice required
		Required: true,
	},
	"auto_deactivation_days": {
		Type:     "integer",
		Min:      30,  // Minimum 30 days (1 month)
		Max:      730, // Maximum 730 days (2 years)
		Required: true,
	},
	"registration_password": {
		Type:     "registration_password",
		Required: true,
	},
	"whatsapp_group_enabled": {
		Type:     "boolean",
		Required: true,
	},
	"whatsapp_group_link": {
		Type:     "whatsapp_link",
		Required: false, // Empty is allowed
	},
	"max_bookings_per_dog_per_day": {
		Type:     "integer",
		Min:      1,
		Max:      10,
		Required: true,
	},
	"recurring_booking_max_weeks": {
		Type:     "integer",
		Min:      1,
		Max:      52,
		Required: true,
	},
	"max_active_recurring_series": {
		Type:     "integer",
		Min:      1,
		Max:      20,
		Required: true,
	},
}

// ValidateSetting validates a setting value based on its key
// HIGH-7: Type-specific and range validation for settings
func ValidateSetting(key, value string) error {
	constraint, exists := settingConstraints[key]
	if !exists {
		// Unknown setting - no validation rules defined, allow anything
		return nil
	}

	// Check required
	if constraint.Required && value == "" {
		return &ValidationError{
			Field:   key,
			Message: "Value is required",
		}
	}

	// Type-specific validation
	switch constraint.Type {
	case "integer":
		return validateIntegerSetting(key, value, constraint)
	case "boolean":
		return validateBooleanSetting(key, value)
	case "registration_password":
		return validateRegistrationPassword(key, value)
	case "whatsapp_link":
		return validateWhatsAppLink(key, value)
	}

	return nil
}

// validateIntegerSetting validates integer settings with range constraints
func validateIntegerSetting(key, value string, constraint SettingConstraint) error {
	val, err := strconv.Atoi(value)
	if err != nil {
		return &ValidationError{
			Field:   key,
			Message: "Value must be a valid integer",
		}
	}

	if val < constraint.Min {
		return &ValidationError{
			Field:   key,
			Message: fmt.Sprintf("Value must be at least %d", constraint.Min),
		}
	}

	if val > constraint.Max {
		return &ValidationError{
			Field:   key,
			Message: fmt.Sprintf("Value must not exceed %d", constraint.Max),
		}
	}

	return nil
}

// validateBooleanSetting validates boolean settings (must be "true" or "false")
func validateBooleanSetting(key, value string) error {
	if value != "true" && value != "false" {
		return &ValidationError{
			Field:   key,
			Message: "Value must be 'true' or 'false'",
		}
	}
	return nil
}

// validateRegistrationPassword validates the registration password format (8 alphanumeric chars)
func validateRegistrationPassword(key, value string) error {
	if !regexp.MustCompile(`^[a-zA-Z0-9]{8}$`).MatchString(value) {
		return &ValidationError{
			Field:   key,
			Message: "Registration password must be exactly 8 alphanumeric characters",
		}
	}
	return nil
}

// validateWhatsAppLink validates WhatsApp group links
func validateWhatsAppLink(key, value string) error {
	// Empty is allowed for whatsapp_group_link
	if value == "" {
		return nil
	}

	if !strings.HasPrefix(value, "https://chat.whatsapp.com/") {
		return &ValidationError{
			Field:   key,
			Message: "WhatsApp group link must start with https://chat.whatsapp.com/",
		}
	}
	return nil
}
