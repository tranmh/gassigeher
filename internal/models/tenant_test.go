package models

import (
	"testing"
)

// TestValidateSlug tests the slug validation function
func TestValidateSlug(t *testing.T) {
	tests := []struct {
		name    string
		slug    string
		wantErr bool
	}{
		{"valid slug", "tierheim-goeppingen", false},
		{"valid slug with numbers", "tierheim123", false},
		{"valid minimum length", "abc", false},
		{"empty slug", "", true},
		{"too short", "ab", true},
		{"too long", string(make([]byte, 101)), true},
		{"starts with hyphen", "-tierheim", true},
		{"ends with hyphen", "tierheim-", true},
		{"consecutive hyphens", "tierheim--goeppingen", true},
		{"uppercase letters", "Tierheim", true},
		{"contains underscore", "tierheim_goeppingen", true},
		{"contains space", "tierheim goeppingen", true},
		{"reserved slug www", "www", true},
		{"reserved slug admin", "admin", true},
		{"reserved slug api", "api", true},
		{"reserved slug test", "test", true},
		{"reserved slug mail", "mail", true},
		{"demo is not reserved", "demo", false},
		{"single character", "a", true},
		// BUGFIX: Changed from false to true - slugs must start with letter (to match isValidSlug in handler)
		{"only numbers", "123456", true},
		{"starts with number", "1tierheim", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSlug(tt.slug)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSlug(%q) error = %v, wantErr %v", tt.slug, err, tt.wantErr)
			}
		})
	}
}

// TestTenantRegistrationRequest_Validate tests registration validation
func TestTenantRegistrationRequest_Validate(t *testing.T) {
	validRequest := TenantRegistrationRequest{
		OrganizationName: "Tierheim Göppingen",
		Slug:             "tierheim-goeppingen",
		ContactEmail:     "info@tierheim.de",
		City:             "Göppingen",
		PostalCode:       "73033",
		FederalState:     "BW",
		AdminFirstName:   "Max",
		AdminLastName:    "Mustermann",
		AdminEmail:       "admin@tierheim.de",
		AdminPassword:    "Password123", // Must meet complexity: uppercase, lowercase, digit
	}

	t.Run("valid request", func(t *testing.T) {
		req := validRequest
		if err := req.Validate(); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("empty organization name", func(t *testing.T) {
		req := validRequest
		req.OrganizationName = ""
		if err := req.Validate(); err == nil {
			t.Error("Expected error for empty organization name")
		}
	})

	t.Run("organization name too long", func(t *testing.T) {
		req := validRequest
		req.OrganizationName = string(make([]byte, 256))
		if err := req.Validate(); err == nil {
			t.Error("Expected error for too long organization name")
		}
	})

	t.Run("invalid slug", func(t *testing.T) {
		req := validRequest
		req.Slug = "admin"
		if err := req.Validate(); err == nil {
			t.Error("Expected error for reserved slug")
		}
	})

	t.Run("empty contact email", func(t *testing.T) {
		req := validRequest
		req.ContactEmail = ""
		if err := req.Validate(); err == nil {
			t.Error("Expected error for empty contact email")
		}
	})

	t.Run("invalid contact email format", func(t *testing.T) {
		req := validRequest
		req.ContactEmail = "invalid-email"
		if err := req.Validate(); err == nil {
			t.Error("Expected error for invalid contact email")
		}
	})

	t.Run("empty city", func(t *testing.T) {
		req := validRequest
		req.City = ""
		if err := req.Validate(); err == nil {
			t.Error("Expected error for empty city")
		}
	})

	t.Run("empty postal code", func(t *testing.T) {
		req := validRequest
		req.PostalCode = ""
		if err := req.Validate(); err == nil {
			t.Error("Expected error for empty postal code")
		}
	})

	t.Run("invalid federal state", func(t *testing.T) {
		req := validRequest
		req.FederalState = "XX"
		if err := req.Validate(); err == nil {
			t.Error("Expected error for invalid federal state")
		}
	})

	t.Run("empty federal state uses default", func(t *testing.T) {
		req := validRequest
		req.FederalState = ""
		if err := req.Validate(); err != nil {
			t.Errorf("Expected no error with default federal state, got %v", err)
		}
		if req.FederalState != "BW" {
			t.Errorf("Expected FederalState to be 'BW', got '%s'", req.FederalState)
		}
	})

	t.Run("empty admin first name", func(t *testing.T) {
		req := validRequest
		req.AdminFirstName = ""
		if err := req.Validate(); err == nil {
			t.Error("Expected error for empty admin first name")
		}
	})

	t.Run("empty admin last name", func(t *testing.T) {
		req := validRequest
		req.AdminLastName = ""
		if err := req.Validate(); err == nil {
			t.Error("Expected error for empty admin last name")
		}
	})

	t.Run("empty admin email", func(t *testing.T) {
		req := validRequest
		req.AdminEmail = ""
		if err := req.Validate(); err == nil {
			t.Error("Expected error for empty admin email")
		}
	})

	t.Run("invalid admin email format", func(t *testing.T) {
		req := validRequest
		req.AdminEmail = "not-an-email"
		if err := req.Validate(); err == nil {
			t.Error("Expected error for invalid admin email")
		}
	})

	t.Run("admin password too short", func(t *testing.T) {
		req := validRequest
		req.AdminPassword = "short"
		if err := req.Validate(); err == nil {
			t.Error("Expected error for short admin password")
		}
	})
}

// TestValidateThemePreset tests theme preset validation
func TestValidateThemePreset(t *testing.T) {
	tests := []struct {
		name   string
		preset string
		want   bool
	}{
		{"classic preset", "classic", true},
		{"ocean preset", "ocean", true},
		{"forest preset", "forest", true},
		{"sunset preset", "sunset", true},
		{"lavender preset", "lavender", true},
		{"coral preset", "coral", true},
		{"midnight preset", "midnight", true},
		{"emerald preset", "emerald", true},
		{"rose preset", "rose", true},
		{"slate preset", "slate", true},
		{"invalid preset", "invalid", false},
		{"empty preset", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateThemePreset(tt.preset); got != tt.want {
				t.Errorf("ValidateThemePreset(%q) = %v, want %v", tt.preset, got, tt.want)
			}
		})
	}
}

// TestValidateHexColor tests hex color validation
func TestValidateHexColor(t *testing.T) {
	tests := []struct {
		name  string
		color string
		want  bool
	}{
		{"valid hex color lowercase", "#82b965", true},
		{"valid hex color uppercase", "#82B965", true},
		{"valid hex color mixed", "#82B96f", true},
		{"empty string is valid", "", true},
		{"missing hash", "82b965", false},
		{"too short", "#82b96", false},
		{"too long", "#82b9650", false},
		{"invalid characters", "#82b96g", false},
		{"only hash", "#", false},
		{"three digit hex", "#82b", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateHexColor(tt.color); got != tt.want {
				t.Errorf("ValidateHexColor(%q) = %v, want %v", tt.color, got, tt.want)
			}
		})
	}
}

// TestTenantSettings_GetThemeColors tests theme color retrieval
func TestTenantSettings_GetThemeColors(t *testing.T) {
	t.Run("uses preset when no custom colors", func(t *testing.T) {
		settings := &TenantSettings{
			ThemePreset: "ocean",
		}
		colors := settings.GetThemeColors()
		expected := ThemePresets["ocean"]
		if colors.Primary != expected.Primary {
			t.Errorf("Expected Primary %s, got %s", expected.Primary, colors.Primary)
		}
	})

	t.Run("uses custom colors when provided", func(t *testing.T) {
		primary := "#ff0000"
		secondary := "#00ff00"
		settings := &TenantSettings{
			ThemePreset:    "classic",
			ColorPrimary:   &primary,
			ColorSecondary: &secondary,
		}
		colors := settings.GetThemeColors()
		if colors.Primary != primary {
			t.Errorf("Expected Primary %s, got %s", primary, colors.Primary)
		}
		if colors.Secondary != secondary {
			t.Errorf("Expected Secondary %s, got %s", secondary, colors.Secondary)
		}
	})

	t.Run("uses defaults for missing custom colors", func(t *testing.T) {
		primary := "#ff0000"
		settings := &TenantSettings{
			ThemePreset:  "classic",
			ColorPrimary: &primary,
		}
		colors := settings.GetThemeColors()
		if colors.Primary != primary {
			t.Errorf("Expected Primary %s, got %s", primary, colors.Primary)
		}
		// Secondary should use default
		if colors.Secondary != "#26272b" {
			t.Errorf("Expected default Secondary #26272b, got %s", colors.Secondary)
		}
	})

	t.Run("fallback to classic for invalid preset", func(t *testing.T) {
		settings := &TenantSettings{
			ThemePreset: "invalid-preset",
		}
		colors := settings.GetThemeColors()
		classic := ThemePresets["classic"]
		if colors.Primary != classic.Primary {
			t.Errorf("Expected classic Primary %s, got %s", classic.Primary, colors.Primary)
		}
	})

	t.Run("empty string custom color uses default", func(t *testing.T) {
		empty := ""
		settings := &TenantSettings{
			ThemePreset:  "classic",
			ColorPrimary: &empty,
		}
		colors := settings.GetThemeColors()
		// Should use preset, not custom
		classic := ThemePresets["classic"]
		if colors.Primary != classic.Primary {
			t.Errorf("Expected classic Primary %s, got %s", classic.Primary, colors.Primary)
		}
	})
}

// TestTenant_StatusMethods tests tenant status helper methods
func TestTenant_StatusMethods(t *testing.T) {
	t.Run("IsActive returns true for active tenant", func(t *testing.T) {
		tenant := &Tenant{Status: TenantStatusActive}
		if !tenant.IsActive() {
			t.Error("Expected IsActive() to return true")
		}
		if tenant.IsSuspended() {
			t.Error("Expected IsSuspended() to return false")
		}
		if tenant.IsDeleted() {
			t.Error("Expected IsDeleted() to return false")
		}
	})

	t.Run("IsSuspended returns true for suspended tenant", func(t *testing.T) {
		tenant := &Tenant{Status: TenantStatusSuspended}
		if tenant.IsActive() {
			t.Error("Expected IsActive() to return false")
		}
		if !tenant.IsSuspended() {
			t.Error("Expected IsSuspended() to return true")
		}
		if tenant.IsDeleted() {
			t.Error("Expected IsDeleted() to return false")
		}
	})

	t.Run("IsDeleted returns true for deleted tenant", func(t *testing.T) {
		tenant := &Tenant{Status: TenantStatusDeleted}
		if tenant.IsActive() {
			t.Error("Expected IsActive() to return false")
		}
		if tenant.IsSuspended() {
			t.Error("Expected IsSuspended() to return false")
		}
		if !tenant.IsDeleted() {
			t.Error("Expected IsDeleted() to return true")
		}
	})
}

// TestFederalStates tests the federal states map
func TestFederalStates(t *testing.T) {
	// Verify all 16 German federal states are present
	if len(FederalStates) != 16 {
		t.Errorf("Expected 16 federal states, got %d", len(FederalStates))
	}

	// Check some specific states
	expectedStates := map[string]string{
		"BW": "Baden-Württemberg",
		"BY": "Bayern",
		"BE": "Berlin",
		"NW": "Nordrhein-Westfalen",
	}

	for code, name := range expectedStates {
		if FederalStates[code] != name {
			t.Errorf("Expected FederalStates[%q] = %q, got %q", code, name, FederalStates[code])
		}
	}
}
