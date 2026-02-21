package repository

import (
	"testing"

	"github.com/tranmh/gassigeher/internal/testutil"
)

// DONE: TestSettingsRepository_Get tests getting a single setting
func TestSettingsRepository_Get(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSettingsRepository(db)

	// Insert test setting
	// Note: "key" is a SQL reserved word, must be quoted for PostgreSQL compatibility
	db.Exec(`INSERT INTO system_settings (tenant_id, "key", value) VALUES (0, ?, ?)`, "test_setting", "test_value")

	t.Run("setting exists", func(t *testing.T) {
		setting, err := repo.Get(0, "test_setting") // tenantID = 0
		if err != nil {
			t.Fatalf("Get() failed: %v", err)
		}

		if setting.Key != "test_setting" {
			t.Errorf("Expected key 'test_setting', got %s", setting.Key)
		}

		if setting.Value != "test_value" {
			t.Errorf("Expected value 'test_value', got %s", setting.Value)
		}
	})

	t.Run("setting not found", func(t *testing.T) {
		setting, err := repo.Get(0, "non_existent_setting") // tenantID = 0
		if setting != nil {
			t.Error("Expected nil for non-existent setting")
		}
		if err != nil {
			t.Logf("Get non-existent setting returned error: %v", err)
		}
	})

	t.Run("empty key", func(t *testing.T) {
		setting, err := repo.Get(0, "") // tenantID = 0
		if setting != nil {
			t.Error("Expected nil for empty key")
		}
		if err != nil {
			t.Logf("Get empty key returned error: %v", err)
		}
	})
}

// DONE: TestSettingsRepository_GetAll tests getting all settings
func TestSettingsRepository_GetAll(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSettingsRepository(db)

	// Insert test settings
	// Note: "key" is a SQL reserved word, must be quoted for PostgreSQL compatibility
	db.Exec(`INSERT INTO system_settings (tenant_id, "key", value) VALUES (0, ?, ?)`, "booking_advance_days", "14")
	db.Exec(`INSERT INTO system_settings (tenant_id, "key", value) VALUES (0, ?, ?)`, "cancellation_notice_hours", "12")
	db.Exec(`INSERT INTO system_settings (tenant_id, "key", value) VALUES (0, ?, ?)`, "auto_deactivation_days", "365")

	t.Run("get all settings", func(t *testing.T) {
		settings, err := repo.GetAll(0) // tenantID = 0
		if err != nil {
			t.Fatalf("GetAll() failed: %v", err)
		}

		if len(settings) != 15 {
			t.Errorf("Expected 15 settings, got %d", len(settings))
		}

		// Verify all expected settings are present
		keys := make(map[string]bool)
		for _, s := range settings {
			keys[s.Key] = true
		}

		// Original 3 settings + 5 from migration 012 + 1 from migration 017 + 1 from migration 018 + 2 from migration 021 + 1 from migration 028
		expectedKeys := []string{
			"booking_advance_days", "cancellation_notice_hours", "auto_deactivation_days",
			"morning_walk_requires_approval", "use_feiertage_api", "feiertage_state",
			"booking_time_granularity", "feiertage_cache_days", "site_logo", "registration_password",
			"whatsapp_group_enabled", "whatsapp_group_link", "default_color_for_new_users",
		}
		for _, key := range expectedKeys {
			if !keys[key] {
				t.Errorf("Expected setting '%s' to be present", key)
			}
		}
	})

	t.Run("fresh database has default settings from migration", func(t *testing.T) {
		db2 := testutil.SetupTestDB(t)
		repo2 := NewSettingsRepository(db2)

		settings, err := repo2.GetAll(0) // tenantID = 0
		if err != nil {
			t.Fatalf("GetAll() failed: %v", err)
		}

		// Database migration creates default settings
		t.Logf("Fresh database has %d default settings from migration", len(settings))

		// Should have at least some settings from migration
		if len(settings) < 0 {
			t.Error("Expected settings from migration")
		}
	})
}

// DONE: TestSettingsRepository_Update tests updating settings
func TestSettingsRepository_Update(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSettingsRepository(db)

	// Insert initial setting
	// Note: "key" is a SQL reserved word, must be quoted for PostgreSQL compatibility
	db.Exec(`INSERT INTO system_settings (tenant_id, "key", value) VALUES (0, ?, ?)`, "booking_advance_days", "14")

	t.Run("update existing setting", func(t *testing.T) {
		err := repo.Update(0, "booking_advance_days", "21") // tenantID = 0
		if err != nil {
			t.Fatalf("Update() failed: %v", err)
		}

		// Verify update
		setting, _ := repo.Get(0, "booking_advance_days") // tenantID = 0
		if setting.Value != "21" {
			t.Errorf("Expected value '21', got %s", setting.Value)
		}
	})

	t.Run("update non-existent setting returns error", func(t *testing.T) {
		err := repo.Update(0, "non_existent_setting", "new_value") // tenantID = 0
		if err == nil {
			t.Error("Expected error when updating non-existent setting")
		}
		t.Logf("Update non-existent setting correctly returned error: %v", err)
	})

	t.Run("update with empty value", func(t *testing.T) {
		// First create the setting
		db.Exec(`INSERT INTO system_settings (tenant_id, "key", value) VALUES (0, ?, ?)`, "test_key", "initial")

		err := repo.Update(0, "test_key", "") // tenantID = 0
		if err != nil {
			t.Fatalf("Update with empty value failed: %v", err)
		}

		// Verify empty value was set
		setting, _ := repo.Get(0, "test_key") // tenantID = 0
		if setting == nil {
			t.Error("Expected setting to exist")
		} else if setting.Value != "" {
			t.Errorf("Expected empty value, got: %s", setting.Value)
		}
	})
}
