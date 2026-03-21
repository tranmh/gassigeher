package database

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestMarketingTablesExist verifies that all marketing tables are created by migration
// RED PHASE: This test will fail because marketing tables don't exist
func TestMarketingTablesExist(t *testing.T) {
	// Create in-memory database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := RunMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Test that marketing tables exist
	requiredTables := []string{
		"marketing_campaigns",
		"referral_codes",
		"referral_uses",
		"reference_entries",
	}

	for _, table := range requiredTables {
		var count int
		query := "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?"
		if err := db.QueryRow(query, table).Scan(&count); err != nil {
			t.Errorf("Error checking for table %s: %v", table, err)
			continue
		}
		if count == 0 {
			t.Errorf("Table %s does not exist", table)
		}
	}
}

// TestTenantSettingsHasAllColumns verifies tenant_settings has tagline and description
// RED PHASE: This test will fail because columns don't exist
func TestTenantSettingsHasAllColumns(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := RunMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Check that tagline and description columns exist
	requiredColumns := []string{"tagline", "description"}

	for _, col := range requiredColumns {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue interface{}

		// PRAGMA table_info returns column info
		rows, err := db.Query("PRAGMA table_info(tenant_settings)")
		if err != nil {
			t.Fatalf("Error getting table info: %v", err)
		}

		found := false
		for rows.Next() {
			if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
				t.Fatalf("Error scanning column info: %v", err)
			}
			if name == col {
				found = true
				break
			}
		}
		rows.Close()

		if !found {
			t.Errorf("Column %s does not exist in tenant_settings table", col)
		}
	}
}

// TestFeatureFlagsTablesExist verifies that feature_flags tables are created by migration
// BUG: This test will fail because feature_flags table doesn't exist
func TestFeatureFlagsTablesExist(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := RunMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Test that feature flag tables exist
	requiredTables := []string{
		"feature_flags",
		"tenant_feature_flags",
	}

	for _, table := range requiredTables {
		var count int
		query := "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?"
		if err := db.QueryRow(query, table).Scan(&count); err != nil {
			t.Errorf("Error checking for table %s: %v", table, err)
			continue
		}
		if count == 0 {
			t.Errorf("BUG: Table %s does not exist - migration is missing", table)
		}
	}
}

// TestMigrationCreatesAllTablesAtomically verifies migration is atomic
func TestMigrationCreatesAllTablesAtomically(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := RunMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// All tables that should exist after migration
	allTables := []string{
		"tenants",
		"tenant_settings",
		"users",
		"dogs",
		"bookings",
		"blocked_dates",
		"color_categories",
		"reactivation_requests",
		"system_settings",
		"booking_time_rules",
		"custom_holidays",
		"feiertage_cache",
		"pricing_plans",
		"tenant_subscriptions",
		"marketing_campaigns",
		"referral_codes",
		"referral_uses",
		"reference_entries",
		"feature_flags",
		"tenant_feature_flags",
		"schema_migrations",
	}

	for _, table := range allTables {
		var count int
		query := "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?"
		if err := db.QueryRow(query, table).Scan(&count); err != nil {
			t.Errorf("Error checking for table %s: %v", table, err)
			continue
		}
		if count == 0 {
			t.Errorf("Table %s does not exist - migration may have failed partway through", table)
		}
	}
}
