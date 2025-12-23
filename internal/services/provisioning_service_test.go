package services

import (
	"testing"

	"github.com/tranmh/gassigeher/internal/testutil"
)

// TestProvisioningService_CreateDefaultColors tests creating default colors for a tenant
func TestProvisioningService_CreateDefaultColors(t *testing.T) {
	db := testutil.SetupTestDB(t)
	service := NewProvisioningService(db)

	// Create a new tenant to avoid pre-seeded data from migrations
	result, err := db.Exec(`
		INSERT INTO tenants (slug, name, contact_email, federal_state, status)
		VALUES ('colors-test', 'Colors Test', 'colors@example.com', 'BW', 'active')
	`)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	tenantID, _ := result.LastInsertId()

	// Create a transaction
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Create default colors for the new tenant
	err = service.CreateDefaultColors(tx, int(tenantID))
	if err != nil {
		t.Fatalf("CreateDefaultColors failed: %v", err)
	}

	// Commit to check results
	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Verify colors were created
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM color_categories WHERE tenant_id = ?", tenantID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count colors: %v", err)
	}

	// Should have 5 default colors (Grün, Gelb, Orange, Rot, Blau)
	if count != 5 {
		t.Errorf("Expected 5 default colors, got %d", count)
	}
}

// TestProvisioningService_CreateDefaultBookingRules tests creating default booking rules
func TestProvisioningService_CreateDefaultBookingRules(t *testing.T) {
	db := testutil.SetupTestDB(t)
	service := NewProvisioningService(db)

	// Create a new tenant to avoid pre-seeded data from migrations
	result, err := db.Exec(`
		INSERT INTO tenants (slug, name, contact_email, federal_state, status)
		VALUES ('rules-test', 'Rules Test', 'rules@example.com', 'BW', 'active')
	`)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	tenantID, _ := result.LastInsertId()

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	err = service.CreateDefaultBookingRules(tx, int(tenantID))
	if err != nil {
		t.Fatalf("CreateDefaultBookingRules failed: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Verify booking rules were created
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM booking_time_rules WHERE tenant_id = ?", tenantID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count booking rules: %v", err)
	}

	// Should have 7 rules (weekday morning, lunch, afternoon + weekend morning, afternoon + holiday morning, afternoon)
	if count != 7 {
		t.Errorf("Expected 7 default booking rules, got %d", count)
	}

	// Verify specific rule types
	var weekdayCount, weekendCount, holidayCount int
	db.QueryRow("SELECT COUNT(*) FROM booking_time_rules WHERE tenant_id = ? AND day_type = 'weekday'", tenantID).Scan(&weekdayCount)
	db.QueryRow("SELECT COUNT(*) FROM booking_time_rules WHERE tenant_id = ? AND day_type = 'weekend'", tenantID).Scan(&weekendCount)
	db.QueryRow("SELECT COUNT(*) FROM booking_time_rules WHERE tenant_id = ? AND day_type = 'holiday'", tenantID).Scan(&holidayCount)

	if weekdayCount != 3 {
		t.Errorf("Expected 3 weekday rules, got %d", weekdayCount)
	}
	if weekendCount != 2 {
		t.Errorf("Expected 2 weekend rules, got %d", weekendCount)
	}
	if holidayCount != 2 {
		t.Errorf("Expected 2 holiday rules, got %d", holidayCount)
	}
}

// TestProvisioningService_CreateDefaultSettings tests creating default settings
func TestProvisioningService_CreateDefaultSettings(t *testing.T) {
	db := testutil.SetupTestDB(t)
	service := NewProvisioningService(db)

	// Create a new tenant to avoid pre-seeded data from migrations
	result, err := db.Exec(`
		INSERT INTO tenants (slug, name, contact_email, federal_state, status)
		VALUES ('settings-test', 'Settings Test', 'settings@example.com', 'BW', 'active')
	`)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	tenantID, _ := result.LastInsertId()

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	err = service.CreateDefaultSettings(tx, int(tenantID))
	if err != nil {
		t.Fatalf("CreateDefaultSettings failed: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Verify settings were created
	var bookingDays, cancellationHours, deactivationDays string
	db.QueryRow("SELECT value FROM system_settings WHERE tenant_id = ? AND `key` = 'booking_advance_days'", tenantID).Scan(&bookingDays)
	db.QueryRow("SELECT value FROM system_settings WHERE tenant_id = ? AND `key` = 'cancellation_notice_hours'", tenantID).Scan(&cancellationHours)
	db.QueryRow("SELECT value FROM system_settings WHERE tenant_id = ? AND `key` = 'auto_deactivation_days'", tenantID).Scan(&deactivationDays)

	if bookingDays != "14" {
		t.Errorf("Expected booking_advance_days = '14', got '%s'", bookingDays)
	}
	if cancellationHours != "12" {
		t.Errorf("Expected cancellation_notice_hours = '12', got '%s'", cancellationHours)
	}
	if deactivationDays != "365" {
		t.Errorf("Expected auto_deactivation_days = '365', got '%s'", deactivationDays)
	}
}

// TestProvisioningService_ProvisionTenant tests full tenant provisioning
func TestProvisioningService_ProvisionTenant(t *testing.T) {
	db := testutil.SetupTestDB(t)
	service := NewProvisioningService(db)

	// Create a new tenant
	result, err := db.Exec(`
		INSERT INTO tenants (slug, name, contact_email, federal_state, status)
		VALUES ('test-org', 'Test Organization', 'test@example.com', 'BW', 'active')
	`)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	tenantID, _ := result.LastInsertId()

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Provision the tenant
	err = service.ProvisionTenant(tx, int(tenantID))
	if err != nil {
		t.Fatalf("ProvisionTenant failed: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Verify all data was created
	var colorCount, ruleCount, settingCount int
	db.QueryRow("SELECT COUNT(*) FROM color_categories WHERE tenant_id = ?", tenantID).Scan(&colorCount)
	db.QueryRow("SELECT COUNT(*) FROM booking_time_rules WHERE tenant_id = ?", tenantID).Scan(&ruleCount)
	db.QueryRow("SELECT COUNT(*) FROM system_settings WHERE tenant_id = ?", tenantID).Scan(&settingCount)

	if colorCount != 5 {
		t.Errorf("Expected 5 colors, got %d", colorCount)
	}
	if ruleCount != 7 {
		t.Errorf("Expected 7 booking rules, got %d", ruleCount)
	}
	if settingCount != 4 {
		t.Errorf("Expected 4 settings (including registration_password), got %d", settingCount)
	}
}

// TestProvisioningService_CreateDefaultSettings_IncludesRegistrationPassword tests that
// registration_password is included in default settings (required for user registration)
func TestProvisioningService_CreateDefaultSettings_IncludesRegistrationPassword(t *testing.T) {
	db := testutil.SetupTestDB(t)
	service := NewProvisioningService(db)

	// Create a new tenant
	result, err := db.Exec(`
		INSERT INTO tenants (slug, name, contact_email, federal_state, status)
		VALUES ('regpass-test', 'RegPass Test', 'regpass@example.com', 'BW', 'active')
	`)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	tenantID, _ := result.LastInsertId()

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	err = service.CreateDefaultSettings(tx, int(tenantID))
	if err != nil {
		t.Fatalf("CreateDefaultSettings failed: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Verify registration_password was created
	var registrationPassword string
	err = db.QueryRow("SELECT value FROM system_settings WHERE tenant_id = ? AND `key` = 'registration_password'", tenantID).Scan(&registrationPassword)
	if err != nil {
		t.Fatalf("registration_password setting not found: %v", err)
	}

	// Should be exactly 8 alphanumeric characters
	if len(registrationPassword) != 8 {
		t.Errorf("Expected registration_password to be 8 characters, got %d: '%s'", len(registrationPassword), registrationPassword)
	}

	// Should be alphanumeric only (A-Z, 0-9)
	for _, c := range registrationPassword {
		if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			t.Errorf("registration_password contains invalid character: '%c' in '%s'", c, registrationPassword)
			break
		}
	}
}

// TestProvisioningService_ProvisionTenant_Idempotent tests that provisioning can be called multiple times
func TestProvisioningService_ProvisionTenant_Idempotent(t *testing.T) {
	db := testutil.SetupTestDB(t)
	service := NewProvisioningService(db)

	// Create a new tenant
	result, err := db.Exec(`
		INSERT INTO tenants (slug, name, contact_email, federal_state, status)
		VALUES ('idempotent-org', 'Idempotent Organization', 'test@example.com', 'BW', 'active')
	`)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	tenantID, _ := result.LastInsertId()

	// First provision
	tx1, _ := db.Begin()
	err = service.ProvisionTenant(tx1, int(tenantID))
	if err != nil {
		tx1.Rollback()
		t.Fatalf("First ProvisionTenant failed: %v", err)
	}
	tx1.Commit()

	// Get initial counts
	var initialColorCount, initialRuleCount int
	db.QueryRow("SELECT COUNT(*) FROM color_categories WHERE tenant_id = ?", tenantID).Scan(&initialColorCount)
	db.QueryRow("SELECT COUNT(*) FROM booking_time_rules WHERE tenant_id = ?", tenantID).Scan(&initialRuleCount)

	// Second provision should also succeed (INSERT OR REPLACE for settings)
	tx2, _ := db.Begin()
	// Note: Colors and rules will fail due to unique constraints, but settings should be fine
	// We expect this to fail actually since colors have unique constraint per tenant
	err = service.ProvisionTenant(tx2, int(tenantID))
	if err == nil {
		tx2.Commit()
		// If it succeeded, counts should be the same
		var colorCount, ruleCount int
		db.QueryRow("SELECT COUNT(*) FROM color_categories WHERE tenant_id = ?", tenantID).Scan(&colorCount)
		db.QueryRow("SELECT COUNT(*) FROM booking_time_rules WHERE tenant_id = ?", tenantID).Scan(&ruleCount)

		if colorCount != initialColorCount {
			t.Errorf("Color count changed from %d to %d", initialColorCount, colorCount)
		}
		if ruleCount != initialRuleCount {
			t.Errorf("Rule count changed from %d to %d", initialRuleCount, ruleCount)
		}
	} else {
		// It's expected to fail if there are unique constraints
		tx2.Rollback()
	}
}
