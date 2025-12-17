package models

import "time"

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
