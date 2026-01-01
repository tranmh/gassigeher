package services

import (
	"database/sql"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/tranmh/gassigeher/internal/database"
)

// setupTestDB creates an in-memory SQLite database with migrations
func setupTestDB(t *testing.T) *database.DB {
	rawDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Run migrations
	dialect := &database.SQLiteDialect{}
	if err := database.RunMigrationsWithDialect(rawDB, dialect); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Clear global data from migrations to start fresh for tenant-specific tests
	// This removes colors, settings, etc. that were inserted without tenant_id
	rawDB.Exec("DELETE FROM color_categories WHERE tenant_id IS NULL OR tenant_id = 0")
	rawDB.Exec("DELETE FROM system_settings WHERE tenant_id IS NULL OR tenant_id = 0")
	rawDB.Exec("DELETE FROM booking_time_rules WHERE tenant_id IS NULL OR tenant_id = 0")

	// Wrap in database.DB for cross-database support
	sqlxDB := sqlx.NewDb(rawDB, "sqlite3")
	return database.WrapSqlxDB(sqlxDB, dialect)
}

// TestLocalDevTenantConfigs verifies the tenant configurations
func TestLocalDevTenantConfigs(t *testing.T) {
	t.Run("has exactly 4 tenants defined", func(t *testing.T) {
		if len(LocalDevTenants) != 4 {
			t.Errorf("Expected 4 local dev tenants, got %d", len(LocalDevTenants))
		}
	})

	t.Run("tenant slugs are demo1-demo4", func(t *testing.T) {
		expectedSlugs := []string{"demo1", "demo2", "demo3", "demo4"}
		for i, tenant := range LocalDevTenants {
			if tenant.Slug != expectedSlugs[i] {
				t.Errorf("Expected tenant %d slug '%s', got '%s'", i, expectedSlugs[i], tenant.Slug)
			}
		}
	})

	t.Run("profiles are assigned correctly", func(t *testing.T) {
		expectedProfiles := []LocalDevProfile{ProfileEmpty, ProfileSmall, ProfileMedium, ProfileStress}
		for i, tenant := range LocalDevTenants {
			if tenant.Profile != expectedProfiles[i] {
				t.Errorf("Expected tenant %d profile '%s', got '%s'", i, expectedProfiles[i], tenant.Profile)
			}
		}
	})
}

// TestGetProfileFromSlug tests the profile lookup function
func TestGetProfileFromSlug(t *testing.T) {
	tests := []struct {
		slug string
		want LocalDevProfile
	}{
		{"demo1", ProfileEmpty},
		{"demo2", ProfileSmall},
		{"demo3", ProfileMedium},
		{"demo4", ProfileStress},
		{"unknown", ProfileEmpty}, // Default
		{"demo", ProfileEmpty},    // Not a local dev tenant
		{"", ProfileEmpty},        // Empty slug
	}

	for _, tt := range tests {
		t.Run(tt.slug, func(t *testing.T) {
			got := GetProfileFromSlug(tt.slug)
			if got != tt.want {
				t.Errorf("GetProfileFromSlug(%q) = %v, want %v", tt.slug, got, tt.want)
			}
		})
	}
}

// TestLocalDevPassword verifies the password constant
func TestLocalDevPassword(t *testing.T) {
	if LocalDevPassword != "localdev1234" {
		t.Errorf("Expected LocalDevPassword 'localdev1234', got '%s'", LocalDevPassword)
	}
}

// TestNewLocalDevSeedService tests service creation
func TestNewLocalDevSeedService(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewLocalDevSeedService(db)

	if service == nil {
		t.Fatal("NewLocalDevSeedService returned nil")
	}
	if service.db == nil {
		t.Error("Service db is nil")
	}
	if service.tenantRepo == nil {
		t.Error("Service tenantRepo is nil")
	}
	if service.userRepo == nil {
		t.Error("Service userRepo is nil")
	}
	if service.dogRepo == nil {
		t.Error("Service dogRepo is nil")
	}
	if service.bookingRepo == nil {
		t.Error("Service bookingRepo is nil")
	}
	if service.colorRepo == nil {
		t.Error("Service colorRepo is nil")
	}
}

// TestEnsureLocalDevTenants tests single tenant creation
// Note: Creating multiple tenants in same DB may hit constraint issues in tests
// due to global constraints that aren't tenant-scoped in test migrations
func TestEnsureLocalDevTenants(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewLocalDevSeedService(db)

	// Test single tenant creation (demo1)
	err := service.ensureTenant(LocalDevTenants[0])
	if err != nil {
		t.Fatalf("ensureTenant failed: %v", err)
	}

	// Verify tenant was created
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM tenants WHERE slug = 'demo1'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count tenants: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 tenant, got %d", count)
	}

	// Second call should be idempotent
	err = service.ensureTenant(LocalDevTenants[0])
	if err != nil {
		t.Fatalf("Second ensureTenant call failed: %v", err)
	}

	// Count should still be 1
	err = db.QueryRow("SELECT COUNT(*) FROM tenants WHERE slug = 'demo1'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count tenants: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 tenant after second call, got %d", count)
	}
}

// TestSeedProfileEmpty tests the empty profile
func TestSeedProfileEmpty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewLocalDevSeedService(db)

	// Create demo1 tenant only
	err := service.ensureTenant(LocalDevTenants[0]) // demo1 = ProfileEmpty
	if err != nil {
		t.Fatalf("ensureTenant failed: %v", err)
	}

	// Get tenant ID
	var tenantID int
	err = db.QueryRow("SELECT id FROM tenants WHERE slug = 'demo1'").Scan(&tenantID)
	if err != nil {
		t.Fatalf("Failed to get tenant ID: %v", err)
	}

	// Verify only admin user exists
	var userCount int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE tenant_id = ?", tenantID).Scan(&userCount)
	if err != nil {
		t.Fatalf("Failed to count users: %v", err)
	}

	if userCount != 1 {
		t.Errorf("Expected 1 user (admin only) for empty profile, got %d", userCount)
	}

	// Verify no dogs
	var dogCount int
	err = db.QueryRow("SELECT COUNT(*) FROM dogs WHERE tenant_id = ?", tenantID).Scan(&dogCount)
	if err != nil {
		t.Fatalf("Failed to count dogs: %v", err)
	}

	if dogCount != 0 {
		t.Errorf("Expected 0 dogs for empty profile, got %d", dogCount)
	}

	// Verify no bookings
	var bookingCount int
	err = db.QueryRow("SELECT COUNT(*) FROM bookings WHERE tenant_id = ?", tenantID).Scan(&bookingCount)
	if err != nil {
		t.Fatalf("Failed to count bookings: %v", err)
	}

	if bookingCount != 0 {
		t.Errorf("Expected 0 bookings for empty profile, got %d", bookingCount)
	}
}

// TestSeedProfileSmall tests the small profile
func TestSeedProfileSmall(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewLocalDevSeedService(db)

	// Create demo2 tenant
	err := service.ensureTenant(LocalDevTenants[1]) // demo2 = ProfileSmall
	if err != nil {
		t.Fatalf("ensureTenant failed: %v", err)
	}

	// Get tenant ID
	var tenantID int
	err = db.QueryRow("SELECT id FROM tenants WHERE slug = 'demo2'").Scan(&tenantID)
	if err != nil {
		t.Fatalf("Failed to get tenant ID: %v", err)
	}

	// Verify users: 1 admin + 3 regular = 4
	var userCount int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE tenant_id = ?", tenantID).Scan(&userCount)
	if err != nil {
		t.Fatalf("Failed to count users: %v", err)
	}

	if userCount != 4 {
		t.Errorf("Expected 4 users for small profile, got %d", userCount)
	}

	// Verify dogs: 5
	var dogCount int
	err = db.QueryRow("SELECT COUNT(*) FROM dogs WHERE tenant_id = ?", tenantID).Scan(&dogCount)
	if err != nil {
		t.Fatalf("Failed to count dogs: %v", err)
	}

	if dogCount != 5 {
		t.Errorf("Expected 5 dogs for small profile, got %d", dogCount)
	}

	// Verify bookings: 10 (5 past + 5 future)
	var bookingCount int
	err = db.QueryRow("SELECT COUNT(*) FROM bookings WHERE tenant_id = ?", tenantID).Scan(&bookingCount)
	if err != nil {
		t.Fatalf("Failed to count bookings: %v", err)
	}

	if bookingCount != 10 {
		t.Errorf("Expected 10 bookings for small profile, got %d", bookingCount)
	}
}

// TestSeedProfileMedium tests the medium profile
func TestSeedProfileMedium(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewLocalDevSeedService(db)

	// Create demo3 tenant
	err := service.ensureTenant(LocalDevTenants[2]) // demo3 = ProfileMedium
	if err != nil {
		t.Fatalf("ensureTenant failed: %v", err)
	}

	// Get tenant ID
	var tenantID int
	err = db.QueryRow("SELECT id FROM tenants WHERE slug = 'demo3'").Scan(&tenantID)
	if err != nil {
		t.Fatalf("Failed to get tenant ID: %v", err)
	}

	// Verify users: 1 admin + 10 regular = 11
	var userCount int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE tenant_id = ?", tenantID).Scan(&userCount)
	if err != nil {
		t.Fatalf("Failed to count users: %v", err)
	}

	if userCount != 11 {
		t.Errorf("Expected 11 users for medium profile, got %d", userCount)
	}

	// Verify dogs: 15
	var dogCount int
	err = db.QueryRow("SELECT COUNT(*) FROM dogs WHERE tenant_id = ?", tenantID).Scan(&dogCount)
	if err != nil {
		t.Fatalf("Failed to count dogs: %v", err)
	}

	if dogCount != 15 {
		t.Errorf("Expected 15 dogs for medium profile, got %d", dogCount)
	}

	// Verify bookings: 50
	var bookingCount int
	err = db.QueryRow("SELECT COUNT(*) FROM bookings WHERE tenant_id = ?", tenantID).Scan(&bookingCount)
	if err != nil {
		t.Fatalf("Failed to count bookings: %v", err)
	}

	if bookingCount != 50 {
		t.Errorf("Expected 50 bookings for medium profile, got %d", bookingCount)
	}

	// Verify blocked dates
	var blockedCount int
	err = db.QueryRow("SELECT COUNT(*) FROM blocked_dates WHERE tenant_id = ?", tenantID).Scan(&blockedCount)
	if err != nil {
		t.Fatalf("Failed to count blocked dates: %v", err)
	}

	if blockedCount < 1 {
		t.Errorf("Expected at least 1 blocked date for medium profile, got %d", blockedCount)
	}
}

// TestSeedProfileStress tests the stress profile
func TestSeedProfileStress(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewLocalDevSeedService(db)

	// Create demo4 tenant
	err := service.ensureTenant(LocalDevTenants[3]) // demo4 = ProfileStress
	if err != nil {
		t.Fatalf("ensureTenant failed: %v", err)
	}

	// Get tenant ID
	var tenantID int
	err = db.QueryRow("SELECT id FROM tenants WHERE slug = 'demo4'").Scan(&tenantID)
	if err != nil {
		t.Fatalf("Failed to get tenant ID: %v", err)
	}

	// Verify users: 1 admin + 100 regular + edge cases
	var userCount int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE tenant_id = ?", tenantID).Scan(&userCount)
	if err != nil {
		t.Fatalf("Failed to count users: %v", err)
	}

	if userCount < 100 {
		t.Errorf("Expected at least 100 users for stress profile, got %d", userCount)
	}

	// Verify dogs: 50
	var dogCount int
	err = db.QueryRow("SELECT COUNT(*) FROM dogs WHERE tenant_id = ?", tenantID).Scan(&dogCount)
	if err != nil {
		t.Fatalf("Failed to count dogs: %v", err)
	}

	if dogCount != 50 {
		t.Errorf("Expected 50 dogs for stress profile, got %d", dogCount)
	}

	// Verify bookings: 500
	var bookingCount int
	err = db.QueryRow("SELECT COUNT(*) FROM bookings WHERE tenant_id = ?", tenantID).Scan(&bookingCount)
	if err != nil {
		t.Fatalf("Failed to count bookings: %v", err)
	}

	if bookingCount != 500 {
		t.Errorf("Expected 500 bookings for stress profile, got %d", bookingCount)
	}
}

// TestResetTenant tests tenant reset functionality
func TestResetTenant(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewLocalDevSeedService(db)

	// Create demo2 tenant first
	err := service.ensureTenant(LocalDevTenants[1]) // demo2 = ProfileSmall
	if err != nil {
		t.Fatalf("ensureTenant failed: %v", err)
	}

	// Get initial counts
	var initialUserCount, initialDogCount int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE tenant_id = (SELECT id FROM tenants WHERE slug = 'demo2')").Scan(&initialUserCount)
	db.QueryRow("SELECT COUNT(*) FROM dogs WHERE tenant_id = (SELECT id FROM tenants WHERE slug = 'demo2')").Scan(&initialDogCount)

	// Reset the tenant
	err = service.ResetTenant("demo2")
	if err != nil {
		t.Fatalf("ResetTenant failed: %v", err)
	}

	// Verify counts are the same (data was reset to same profile)
	var finalUserCount, finalDogCount int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE tenant_id = (SELECT id FROM tenants WHERE slug = 'demo2')").Scan(&finalUserCount)
	db.QueryRow("SELECT COUNT(*) FROM dogs WHERE tenant_id = (SELECT id FROM tenants WHERE slug = 'demo2')").Scan(&finalDogCount)

	if finalUserCount != initialUserCount {
		t.Errorf("User count changed after reset: %d -> %d", initialUserCount, finalUserCount)
	}
	if finalDogCount != initialDogCount {
		t.Errorf("Dog count changed after reset: %d -> %d", initialDogCount, finalDogCount)
	}
}

// TestResetTenantUnknown tests resetting unknown tenant
func TestResetTenantUnknown(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewLocalDevSeedService(db)

	err := service.ResetTenant("unknown-tenant")
	if err == nil {
		t.Error("Expected error when resetting unknown tenant, got nil")
	}
}

// TestAdminUserCreation tests that admin user is created with correct properties
func TestAdminUserCreation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewLocalDevSeedService(db)

	err := service.ensureTenant(LocalDevTenants[0]) // demo1
	if err != nil {
		t.Fatalf("ensureTenant failed: %v", err)
	}

	// Query admin user
	var firstName, lastName string
	var isAdmin, isSuperAdmin, isVerified, isActive int
	err = db.QueryRow(`
		SELECT first_name, last_name, is_admin, is_super_admin, is_verified, is_active
		FROM users
		WHERE tenant_id = (SELECT id FROM tenants WHERE slug = 'demo1')
		AND is_admin = 1
	`).Scan(&firstName, &lastName, &isAdmin, &isSuperAdmin, &isVerified, &isActive)
	if err != nil {
		t.Fatalf("Failed to query admin user: %v", err)
	}

	if firstName != "Admin" {
		t.Errorf("Expected admin first_name 'Admin', got '%s'", firstName)
	}
	if lastName != "Local" {
		t.Errorf("Expected admin last_name 'Local', got '%s'", lastName)
	}
	if isAdmin != 1 {
		t.Error("Expected admin is_admin = true")
	}
	if isSuperAdmin != 1 {
		t.Error("Expected admin is_super_admin = true")
	}
	if isVerified != 1 {
		t.Error("Expected admin is_verified = true")
	}
	if isActive != 1 {
		t.Error("Expected admin is_active = true")
	}
}

// TestColorCategoriesCreation tests that color categories are created
func TestColorCategoriesCreation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewLocalDevSeedService(db)

	err := service.ensureTenant(LocalDevTenants[0]) // demo1
	if err != nil {
		t.Fatalf("ensureTenant failed: %v", err)
	}

	// Query color categories
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM color_categories
		WHERE tenant_id = (SELECT id FROM tenants WHERE slug = 'demo1')
	`).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count color categories: %v", err)
	}

	if count != 5 {
		t.Errorf("Expected 5 color categories, got %d", count)
	}
}

// TestLocalDevSeedService_SeedsBookingTimeRules tests that booking_time_rules are created (TDD RED Phase)
// BUG #5: Local dev tenants don't have booking_time_rules which causes booking failures
func TestLocalDevSeedService_SeedsBookingTimeRules(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewLocalDevSeedService(db)

	// Ensure tenants exist
	err := service.EnsureLocalDevTenants()
	if err != nil {
		t.Fatalf("EnsureLocalDevTenants() error: %v", err)
	}

	// Check that demo1 has booking_time_rules
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM booking_time_rules
		WHERE tenant_id = (SELECT id FROM tenants WHERE slug = 'demo1')
	`).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count booking_time_rules: %v", err)
	}

	if count < 7 {
		t.Errorf("BUG #5: Expected at least 7 booking_time_rules for demo1, got %d", count)
	}
}
