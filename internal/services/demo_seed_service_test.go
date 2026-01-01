package services

import (
	"strings"
	"testing"
	"time"

	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// testConfig returns a config suitable for testing
func testConfig() *config.Config {
	return testutil.TestConfig()
}

// TestDemoSeedService_GenerateRandomPassword tests password generation
func TestDemoSeedService_GenerateRandomPassword(t *testing.T) {
	db := testutil.SetupTestDB(t)
	service := NewDemoSeedService(db, testConfig())

	t.Run("generates 12 character password", func(t *testing.T) {
		password := service.generateRandomPassword()
		if len(password) != 12 {
			t.Errorf("Expected password length 12, got %d", len(password))
		}
	})

	t.Run("generates unique passwords", func(t *testing.T) {
		passwords := make(map[string]bool)
		for i := 0; i < 100; i++ {
			password := service.generateRandomPassword()
			if passwords[password] {
				t.Errorf("Duplicate password generated: %s", password)
			}
			passwords[password] = true
		}
	})

	t.Run("generates hex string", func(t *testing.T) {
		password := service.generateRandomPassword()
		// Hex characters only
		for _, c := range password {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("Password contains non-hex character: %c", c)
			}
		}
	})
}

// TestDemoSeedService_CalculateNextResetTime tests next reset calculation
func TestDemoSeedService_CalculateNextResetTime(t *testing.T) {
	db := testutil.SetupTestDB(t)
	service := NewDemoSeedService(db, testConfig())

	t.Run("calculates next midnight", func(t *testing.T) {
		nextReset := service.calculateNextResetTime()

		// Should be in the future
		if !nextReset.After(time.Now()) {
			t.Error("Next reset time should be in the future")
		}

		// Should be at midnight (00:00:00)
		if nextReset.Hour() != 0 || nextReset.Minute() != 0 || nextReset.Second() != 0 {
			t.Errorf("Expected midnight, got %s", nextReset.Format("15:04:05"))
		}
	})

	t.Run("next reset is within 24 hours", func(t *testing.T) {
		nextReset := service.calculateNextResetTime()
		maxTime := time.Now().Add(25 * time.Hour) // Add some buffer

		if nextReset.After(maxTime) {
			t.Error("Next reset should be within 24 hours")
		}
	})
}

// TestDemoSeedService_EnsureDemoTenant tests demo tenant creation
func TestDemoSeedService_EnsureDemoTenant(t *testing.T) {
	db := testutil.SetupTestDB(t)
	service := NewDemoSeedService(db, testConfig())

	t.Run("creates demo tenant on first run", func(t *testing.T) {
		err := service.EnsureDemoTenant()
		if err != nil {
			t.Fatalf("EnsureDemoTenant() failed: %v", err)
		}

		// Check tenant was created
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM tenants WHERE slug = ?", "demo").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count tenants: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected 1 demo tenant, got %d", count)
		}

		// Check is_demo flag
		var isDemo bool
		err = db.QueryRow("SELECT is_demo FROM tenants WHERE slug = ?", "demo").Scan(&isDemo)
		if err != nil {
			t.Fatalf("Failed to check is_demo: %v", err)
		}
		if !isDemo {
			t.Error("Expected is_demo to be true")
		}
	})

	t.Run("skips creation if tenant exists", func(t *testing.T) {
		// Run again - should not fail
		err := service.EnsureDemoTenant()
		if err != nil {
			t.Fatalf("EnsureDemoTenant() failed on second run: %v", err)
		}

		// Still should have only 1 tenant
		var count int
		db.QueryRow("SELECT COUNT(*) FROM tenants WHERE slug = ?", "demo").Scan(&count)
		if count != 1 {
			t.Errorf("Expected 1 demo tenant after second run, got %d", count)
		}
	})
}

// TestDemoSeedService_SeedDemoData tests demo data seeding
func TestDemoSeedService_SeedDemoData(t *testing.T) {
	db := testutil.SetupTestDB(t)
	service := NewDemoSeedService(db, testConfig())

	// First create demo tenant
	err := service.EnsureDemoTenant()
	if err != nil {
		t.Fatalf("EnsureDemoTenant() failed: %v", err)
	}

	// Get demo tenant ID
	var tenantID int
	err = db.QueryRow("SELECT id FROM tenants WHERE slug = ?", "demo").Scan(&tenantID)
	if err != nil {
		t.Fatalf("Failed to get demo tenant ID: %v", err)
	}

	t.Run("creates color categories", func(t *testing.T) {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM color_categories WHERE tenant_id = ?", tenantID).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count colors: %v", err)
		}
		if count < 5 {
			t.Errorf("Expected at least 5 color categories, got %d", count)
		}
	})

	t.Run("creates demo users", func(t *testing.T) {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM users WHERE tenant_id = ?", tenantID).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count users: %v", err)
		}
		if count < 4 {
			t.Errorf("Expected at least 4 demo users, got %d", count)
		}
	})

	t.Run("creates super admin", func(t *testing.T) {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM users WHERE tenant_id = ? AND is_super_admin = 1", tenantID).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count super admins: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected 1 super admin, got %d", count)
		}
	})

	t.Run("creates demo dogs", func(t *testing.T) {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM dogs WHERE tenant_id = ?", tenantID).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count dogs: %v", err)
		}
		if count < 5 {
			t.Errorf("Expected at least 5 demo dogs, got %d", count)
		}
	})

	t.Run("creates demo bookings", func(t *testing.T) {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM bookings WHERE tenant_id = ?", tenantID).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count bookings: %v", err)
		}
		if count < 3 {
			t.Errorf("Expected at least 3 demo bookings, got %d", count)
		}
	})

	t.Run("creates demo state", func(t *testing.T) {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM demo_tenant_state WHERE tenant_id = ?", tenantID).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count demo state: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected 1 demo state record, got %d", count)
		}
	})
}

// TestDemoSeedService_ResetDemoTenant tests demo reset functionality
func TestDemoSeedService_ResetDemoTenant(t *testing.T) {
	db := testutil.SetupTestDB(t)
	service := NewDemoSeedService(db, testConfig())

	// First create demo tenant
	err := service.EnsureDemoTenant()
	if err != nil {
		t.Fatalf("EnsureDemoTenant() failed: %v", err)
	}

	// Get demo tenant ID
	var tenantID int
	db.QueryRow("SELECT id FROM tenants WHERE slug = ?", "demo").Scan(&tenantID)

	// Get original password
	var originalPassword string
	db.QueryRow("SELECT admin_password FROM demo_tenant_state WHERE tenant_id = ?", tenantID).Scan(&originalPassword)

	t.Run("reset uses fixed password for testing", func(t *testing.T) {
		err := service.ResetDemoTenant()
		if err != nil {
			t.Fatalf("ResetDemoTenant() failed: %v", err)
		}

		// Check password is the fixed demo password (for easy testing)
		var newPassword string
		db.QueryRow("SELECT admin_password FROM demo_tenant_state WHERE tenant_id = ?", tenantID).Scan(&newPassword)
		if newPassword != DemoAdminPassword {
			t.Errorf("Expected password to be %s, got %s", DemoAdminPassword, newPassword)
		}
	})

	t.Run("reset recreates data", func(t *testing.T) {
		// Delete some data first (Max is a demo dog name)
		db.Exec("DELETE FROM dogs WHERE tenant_id = ? AND name = 'Max'", tenantID)

		var countBefore int
		db.QueryRow("SELECT COUNT(*) FROM dogs WHERE tenant_id = ? AND name = 'Max'", tenantID).Scan(&countBefore)
		if countBefore != 0 {
			t.Errorf("Expected 0 Max dogs before reset, got %d", countBefore)
		}

		// Reset
		err := service.ResetDemoTenant()
		if err != nil {
			t.Fatalf("ResetDemoTenant() failed: %v", err)
		}

		// Check data is recreated
		var countAfter int
		db.QueryRow("SELECT COUNT(*) FROM dogs WHERE tenant_id = ? AND name = 'Max'", tenantID).Scan(&countAfter)
		if countAfter != 1 {
			t.Errorf("Expected 1 Max dog after reset, got %d", countAfter)
		}
	})
}

// TestDemoSeedService_GetDemoTenantID tests getting demo tenant ID
func TestDemoSeedService_GetDemoTenantID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	service := NewDemoSeedService(db, testConfig())

	t.Run("returns 0 when no demo tenant", func(t *testing.T) {
		id := service.GetDemoTenantID()
		if id != 0 {
			t.Errorf("Expected 0 when no demo tenant, got %d", id)
		}
	})

	t.Run("returns correct ID after creation", func(t *testing.T) {
		err := service.EnsureDemoTenant()
		if err != nil {
			t.Fatalf("EnsureDemoTenant() failed: %v", err)
		}

		id := service.GetDemoTenantID()
		if id == 0 {
			t.Error("Expected non-zero demo tenant ID")
		}
	})
}

// TestDemoSeedService_DeleteAllTenantData tests data deletion
func TestDemoSeedService_DeleteAllTenantData(t *testing.T) {
	db := testutil.SetupTestDB(t)
	service := NewDemoSeedService(db, testConfig())

	// Create demo tenant with data
	err := service.EnsureDemoTenant()
	if err != nil {
		t.Fatalf("EnsureDemoTenant() failed: %v", err)
	}

	var tenantID int
	db.QueryRow("SELECT id FROM tenants WHERE slug = ?", "demo").Scan(&tenantID)

	t.Run("deletes all tenant data", func(t *testing.T) {
		// Verify data exists
		var userCount, dogCount int
		db.QueryRow("SELECT COUNT(*) FROM users WHERE tenant_id = ?", tenantID).Scan(&userCount)
		db.QueryRow("SELECT COUNT(*) FROM dogs WHERE tenant_id = ?", tenantID).Scan(&dogCount)

		if userCount == 0 || dogCount == 0 {
			t.Skip("No data to delete")
		}

		// Delete data
		err := service.deleteAllTenantData(tenantID)
		if err != nil {
			t.Fatalf("deleteAllTenantData() failed: %v", err)
		}

		// Verify deletion
		db.QueryRow("SELECT COUNT(*) FROM users WHERE tenant_id = ?", tenantID).Scan(&userCount)
		db.QueryRow("SELECT COUNT(*) FROM dogs WHERE tenant_id = ?", tenantID).Scan(&dogCount)

		if userCount != 0 {
			t.Errorf("Expected 0 users after delete, got %d", userCount)
		}
		if dogCount != 0 {
			t.Errorf("Expected 0 dogs after delete, got %d", dogCount)
		}
	})
}

// TestDemoUserPassword tests that demo user password constant is set
func TestDemoUserPassword(t *testing.T) {
	if DemoUserPassword == "" {
		t.Error("DemoUserPassword should not be empty")
	}

	if DemoUserPassword != "demo1234" {
		t.Errorf("Expected DemoUserPassword to be 'demo1234', got %s", DemoUserPassword)
	}
}

// TestDemoTenantSlug tests demo tenant slug from config
func TestDemoTenantSlug(t *testing.T) {
	cfg := testConfig()

	slug := cfg.DemoTenantSlug()
	if slug == "" {
		t.Error("DemoTenantSlug() should not return empty string")
	}

	// The demo tenant slug is always "demo"
	if slug != "demo" {
		t.Errorf("Expected DemoTenantSlug() to be 'demo', got %s", slug)
	}
}

// TestDemoAdminEmail tests demo admin email generated from config
func TestDemoAdminEmail(t *testing.T) {
	cfg := testConfig()

	email := cfg.DemoAdminEmail()
	if email == "" {
		t.Error("DemoAdminEmail() should not return empty string")
	}

	// With test.local as BaseDomain, the email should be admin@demo.test.local
	expected := "admin@demo.test.local"
	if email != expected {
		t.Errorf("Expected DemoAdminEmail() to be '%s', got %s", expected, email)
	}

	// Test with production-like domain
	prodCfg := &config.Config{BaseDomain: "gassigeher.org"}
	prodEmail := prodCfg.DemoAdminEmail()
	expectedProd := "admin@demo.gassigeher.org"
	if prodEmail != expectedProd {
		t.Errorf("Expected DemoAdminEmail() with production domain to be '%s', got %s", expectedProd, prodEmail)
	}
}

// ====================================================================================
// BUG #2: MISSING BOOKING TIME RULES IN DEMO TENANT
// ====================================================================================
// Demo tenants should have default booking time rules so users can actually book.
// Without these rules, the booking time slots API returns empty/null.
// ====================================================================================

// TestDemoSeedService_SeedDemoData_CreatesBookingTimeRules tests that booking time rules are created
// TDD RED PHASE: This test should FAIL until we add booking time rules seeding
func TestDemoSeedService_SeedDemoData_CreatesBookingTimeRules(t *testing.T) {
	db := testutil.SetupTestDB(t)
	service := NewDemoSeedService(db, testConfig())

	// Create demo tenant
	err := service.EnsureDemoTenant()
	if err != nil {
		t.Fatalf("EnsureDemoTenant() failed: %v", err)
	}

	// Get demo tenant ID
	var tenantID int
	err = db.QueryRow("SELECT id FROM tenants WHERE slug = ?", "demo").Scan(&tenantID)
	if err != nil {
		t.Fatalf("Failed to get demo tenant ID: %v", err)
	}

	// BUG: Check if booking time rules were created
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM booking_time_rules WHERE tenant_id = ?", tenantID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count booking time rules: %v", err)
	}

	// Should have at least 5 rules (weekday morning, weekday lunch, weekday afternoon, weekend morning, weekend afternoon)
	if count < 5 {
		t.Errorf("BUG: Expected at least 5 booking time rules for demo tenant, got %d. Users cannot book without these rules!", count)
	}
}

// TestDemoSeedService_ResetDemoTenant_PreservesBookingTimeRules tests that reset recreates rules
// TDD RED PHASE: This test should FAIL
func TestDemoSeedService_ResetDemoTenant_PreservesBookingTimeRules(t *testing.T) {
	db := testutil.SetupTestDB(t)
	service := NewDemoSeedService(db, testConfig())

	// Create demo tenant
	err := service.EnsureDemoTenant()
	if err != nil {
		t.Fatalf("EnsureDemoTenant() failed: %v", err)
	}

	// Get demo tenant ID
	var tenantID int
	db.QueryRow("SELECT id FROM tenants WHERE slug = ?", "demo").Scan(&tenantID)

	// Reset tenant
	err = service.ResetDemoTenant()
	if err != nil {
		t.Fatalf("ResetDemoTenant() failed: %v", err)
	}

	// Check booking time rules after reset
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM booking_time_rules WHERE tenant_id = ?", tenantID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count booking time rules: %v", err)
	}

	if count < 5 {
		t.Errorf("BUG: Expected at least 5 booking time rules after reset, got %d", count)
	}
}

// TestDemoSeedService_SeedDemoColors_Idempotent tests that color seeding is idempotent
func TestDemoSeedService_SeedDemoColors_Idempotent(t *testing.T) {
	db := testutil.SetupTestDB(t)
	service := NewDemoSeedService(db, testConfig())

	// First create demo tenant (without going through EnsureDemoTenant to control the flow)
	now := testutil.Now()
	_, err := db.Exec(`INSERT INTO tenants (slug, name, status, contact_email, federal_state, is_demo, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"test-demo", "Test Demo", "active", "test@demo.org", "BW", 1, now, now)
	if err != nil {
		t.Fatalf("Failed to create test tenant: %v", err)
	}

	var tenantID int
	err = db.QueryRow("SELECT id FROM tenants WHERE slug = ?", "test-demo").Scan(&tenantID)
	if err != nil {
		t.Fatalf("Failed to get tenant ID: %v", err)
	}

	t.Run("seedDemoColors can be called twice without error", func(t *testing.T) {
		// First call
		err := service.seedDemoColors(tenantID)
		if err != nil {
			t.Fatalf("First seedDemoColors() call failed: %v", err)
		}

		// Count colors after first call
		var count1 int
		db.QueryRow("SELECT COUNT(*) FROM color_categories WHERE tenant_id = ?", tenantID).Scan(&count1)

		// Second call - should be idempotent (no error, same count)
		err = service.seedDemoColors(tenantID)
		if err != nil {
			t.Fatalf("Second seedDemoColors() call failed: %v", err)
		}

		// Count colors after second call - should be same
		var count2 int
		db.QueryRow("SELECT COUNT(*) FROM color_categories WHERE tenant_id = ?", tenantID).Scan(&count2)

		if count2 != count1 {
			t.Errorf("Expected same color count after second call (%d), got %d", count1, count2)
		}
	})
}

// TestDemoSeedService_Security_NoPasswordInLogs tests that passwords are not logged
// SECURITY: GASSI-2025-001 - Demo password should not be logged in plaintext
func TestDemoSeedService_Security_NoPasswordInLogs(t *testing.T) {
	t.Run("formatDemoResetLogMessage does not contain password", func(t *testing.T) {
		password := "supersecretpassword123"
		nextReset := time.Now().Add(24 * time.Hour)

		message := formatDemoResetLogMessage(password, nextReset)

		// SECURITY: The log message must NOT contain the password
		if strings.Contains(message, password) {
			t.Errorf("SECURITY VIOLATION: Log message contains password! Message: %s", message)
		}

		// Should still contain useful information
		if !strings.Contains(message, "Demo tenant reset complete") {
			t.Error("Log message should indicate demo reset completion")
		}

		if !strings.Contains(message, nextReset.Format("2006-01-02 15:04")) {
			t.Error("Log message should contain next reset time")
		}
	})

	t.Run("formatDemoResetLogMessage masks password indication", func(t *testing.T) {
		password := "anothersecret456"
		nextReset := time.Now().Add(24 * time.Hour)

		message := formatDemoResetLogMessage(password, nextReset)

		// Should indicate password was reset without revealing it
		if !strings.Contains(message, "password reset") && !strings.Contains(message, "new password generated") {
			// At minimum, should not contain the actual password
			if strings.Contains(message, password) {
				t.Errorf("SECURITY VIOLATION: Password leaked in log message")
			}
		}
	})
}
