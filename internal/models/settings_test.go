package models

import (
	"strings"
	"testing"
)

// DONE: TestUpdateSettingRequest_Validate tests UpdateSettingRequest validation
func TestUpdateSettingRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateSettingRequest
		wantErr bool
	}{
		{
			name: "Valid request with numeric value",
			req: UpdateSettingRequest{
				Value: "14",
			},
			wantErr: false,
		},
		{
			name: "Valid request with text value",
			req: UpdateSettingRequest{
				Value: "Some configuration value",
			},
			wantErr: false,
		},
		{
			name: "Valid request with boolean-like value",
			req: UpdateSettingRequest{
				Value: "true",
			},
			wantErr: false,
		},
		{
			name: "Empty value",
			req: UpdateSettingRequest{
				Value: "",
			},
			wantErr: true,
		},
		{
			name: "Missing value",
			req:  UpdateSettingRequest{},
			wantErr: true,
		},
		{
			name: "Whitespace only value",
			req: UpdateSettingRequest{
				Value: "   ",
			},
			wantErr: false, // Current implementation only checks for empty string, not whitespace
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Additional check: if error expected, verify error type
			if tt.wantErr && err != nil {
				if _, ok := err.(*ValidationError); !ok {
					t.Errorf("Expected ValidationError, got %T", err)
				}
			}
		})
	}
}

// HIGH-7: Test ValidateSetting for type-specific and range validation
func TestValidateSetting(t *testing.T) {
	// Test booking_advance_days range validation
	t.Run("booking_advance_days valid range", func(t *testing.T) {
		// Valid: 1-365 days
		validValues := []string{"1", "7", "14", "30", "365"}
		for _, v := range validValues {
			if err := ValidateSetting("booking_advance_days", v); err != nil {
				t.Errorf("ValidateSetting(booking_advance_days, %q) should pass, got error: %v", v, err)
			}
		}
	})

	t.Run("booking_advance_days exceeds max", func(t *testing.T) {
		// Invalid: > 365 days
		err := ValidateSetting("booking_advance_days", "366")
		if err == nil {
			t.Error("ValidateSetting(booking_advance_days, 366) should fail - exceeds 365 day max")
		}
		if err != nil && !strings.Contains(err.Error(), "365") {
			t.Errorf("Error message should mention max value 365, got: %v", err)
		}
	})

	t.Run("booking_advance_days too large value", func(t *testing.T) {
		// Invalid: unreasonably large value
		err := ValidateSetting("booking_advance_days", "999999")
		if err == nil {
			t.Error("ValidateSetting(booking_advance_days, 999999) should fail - exceeds max")
		}
	})

	// Test cancellation_notice_hours range validation
	t.Run("cancellation_notice_hours valid range", func(t *testing.T) {
		// Valid: 1-168 hours (1 week max)
		validValues := []string{"1", "12", "24", "48", "168"}
		for _, v := range validValues {
			if err := ValidateSetting("cancellation_notice_hours", v); err != nil {
				t.Errorf("ValidateSetting(cancellation_notice_hours, %q) should pass, got error: %v", v, err)
			}
		}
	})

	t.Run("cancellation_notice_hours exceeds max", func(t *testing.T) {
		// Invalid: > 168 hours
		err := ValidateSetting("cancellation_notice_hours", "169")
		if err == nil {
			t.Error("ValidateSetting(cancellation_notice_hours, 169) should fail - exceeds 168 hour max")
		}
		if err != nil && !strings.Contains(err.Error(), "168") {
			t.Errorf("Error message should mention max value 168, got: %v", err)
		}
	})

	// Test auto_deactivation_days range validation
	t.Run("auto_deactivation_days valid range", func(t *testing.T) {
		// Valid: 30-730 days (1 month to 2 years)
		validValues := []string{"30", "90", "180", "365", "730"}
		for _, v := range validValues {
			if err := ValidateSetting("auto_deactivation_days", v); err != nil {
				t.Errorf("ValidateSetting(auto_deactivation_days, %q) should pass, got error: %v", v, err)
			}
		}
	})

	t.Run("auto_deactivation_days below min", func(t *testing.T) {
		// Invalid: < 30 days (too aggressive)
		err := ValidateSetting("auto_deactivation_days", "29")
		if err == nil {
			t.Error("ValidateSetting(auto_deactivation_days, 29) should fail - below 30 day min")
		}
		if err != nil && !strings.Contains(err.Error(), "30") {
			t.Errorf("Error message should mention min value 30, got: %v", err)
		}
	})

	t.Run("auto_deactivation_days exceeds max", func(t *testing.T) {
		// Invalid: > 730 days
		err := ValidateSetting("auto_deactivation_days", "731")
		if err == nil {
			t.Error("ValidateSetting(auto_deactivation_days, 731) should fail - exceeds 730 day max")
		}
		if err != nil && !strings.Contains(err.Error(), "730") {
			t.Errorf("Error message should mention max value 730, got: %v", err)
		}
	})

	// Test common error cases
	t.Run("non-numeric value for numeric setting", func(t *testing.T) {
		err := ValidateSetting("booking_advance_days", "abc")
		if err == nil {
			t.Error("ValidateSetting with non-numeric value should fail")
		}
	})

	t.Run("negative value for numeric setting", func(t *testing.T) {
		err := ValidateSetting("booking_advance_days", "-5")
		if err == nil {
			t.Error("ValidateSetting with negative value should fail")
		}
	})

	t.Run("zero value for numeric setting", func(t *testing.T) {
		err := ValidateSetting("booking_advance_days", "0")
		if err == nil {
			t.Error("ValidateSetting with zero value should fail")
		}
	})

	// Test unknown settings (should pass - no validation needed)
	t.Run("unknown setting passes through", func(t *testing.T) {
		err := ValidateSetting("some_other_setting", "any_value")
		if err != nil {
			t.Errorf("Unknown settings should pass validation, got: %v", err)
		}
	})

	// Test registration_password (8 alphanumeric chars)
	t.Run("registration_password valid", func(t *testing.T) {
		err := ValidateSetting("registration_password", "Abcd1234")
		if err != nil {
			t.Errorf("Valid registration_password should pass, got: %v", err)
		}
	})

	t.Run("registration_password too short", func(t *testing.T) {
		err := ValidateSetting("registration_password", "abc123")
		if err == nil {
			t.Error("registration_password with < 8 chars should fail")
		}
	})

	t.Run("registration_password non-alphanumeric", func(t *testing.T) {
		err := ValidateSetting("registration_password", "abc!@#$%")
		if err == nil {
			t.Error("registration_password with special chars should fail")
		}
	})

	// Test whatsapp_group_enabled (boolean)
	t.Run("whatsapp_group_enabled valid true", func(t *testing.T) {
		err := ValidateSetting("whatsapp_group_enabled", "true")
		if err != nil {
			t.Errorf("whatsapp_group_enabled 'true' should pass, got: %v", err)
		}
	})

	t.Run("whatsapp_group_enabled valid false", func(t *testing.T) {
		err := ValidateSetting("whatsapp_group_enabled", "false")
		if err != nil {
			t.Errorf("whatsapp_group_enabled 'false' should pass, got: %v", err)
		}
	})

	t.Run("whatsapp_group_enabled invalid", func(t *testing.T) {
		err := ValidateSetting("whatsapp_group_enabled", "yes")
		if err == nil {
			t.Error("whatsapp_group_enabled with invalid value should fail")
		}
	})

	// Test whatsapp_group_link
	t.Run("whatsapp_group_link valid", func(t *testing.T) {
		err := ValidateSetting("whatsapp_group_link", "https://chat.whatsapp.com/ABC123")
		if err != nil {
			t.Errorf("Valid whatsapp_group_link should pass, got: %v", err)
		}
	})

	t.Run("whatsapp_group_link empty allowed", func(t *testing.T) {
		err := ValidateSetting("whatsapp_group_link", "")
		if err != nil {
			t.Errorf("Empty whatsapp_group_link should be allowed, got: %v", err)
		}
	})

	t.Run("whatsapp_group_link invalid URL", func(t *testing.T) {
		err := ValidateSetting("whatsapp_group_link", "https://evil.com/phishing")
		if err == nil {
			t.Error("whatsapp_group_link with wrong domain should fail")
		}
	})
}
