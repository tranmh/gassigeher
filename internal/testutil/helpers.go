package testutil

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/database"
)

// Shared database connection for PostgreSQL tests
// This avoids recreating the schema for each test
var (
	sharedDB       *database.DB
	sharedRawDB    *sql.DB  // raw sql.DB for operations that need it
	sharedDialect  database.Dialect
	sharedDBType   string
	sharedDBMu     sync.Mutex
	sharedDBInited bool
)

// Define context keys locally to avoid import cycle with middleware
// These must match the values in internal/middleware/middleware.go
type contextKey string

const (
	testTenantIDKey contextKey = "tenantID"
	testUserIDKey   contextKey = "userID"
)

// SetupTestDB creates a test database with auto-detection
// It checks for DB_TEST_POSTGRES environment variable
// and uses the corresponding database if available. Falls back to SQLite.
// This enables running the same tests against all databases by setting env vars.
//
// For PostgreSQL, this uses a shared connection with table truncation
// instead of dropping and recreating tables for each test (10x+ faster).
func SetupTestDB(t *testing.T) *database.DB {
	// Use the fast version that reuses connections for PostgreSQL
	return SetupTestDBFast(t)
}

// SetupTestDBWithType creates a test database of the specified type
// Supports: sqlite (in-memory), postgres
// For PostgreSQL, requires test database to be available (via Docker or local install)
func SetupTestDBWithType(t *testing.T, dbType string) *database.DB {
	var rawDB *sql.DB
	var dialect database.Dialect
	var err error
	var driverName string

	switch dbType {
	case "sqlite", "":
		// Use in-memory SQLite for fast testing
		rawDB, err = sql.Open("sqlite", ":memory:")
		if err != nil {
			t.Fatalf("Failed to open SQLite test database: %v", err)
		}
		dialect = database.NewSQLiteDialect()
		driverName = "sqlite"

		// Set max connections to 1 to avoid issues with in-memory databases
		// (each connection would get its own database otherwise)
		rawDB.SetMaxOpenConns(1)

		// Apply SQLite settings (PRAGMA foreign_keys, etc.)
		if err := dialect.ApplySettings(rawDB); err != nil {
			t.Fatalf("Failed to apply SQLite settings: %v", err)
		}

	case "postgres":
		// Use test PostgreSQL database (requires DB_TEST_POSTGRES env var)
		dsn := os.Getenv("DB_TEST_POSTGRES")
		if dsn == "" {
			t.Skip("PostgreSQL test database not configured (set DB_TEST_POSTGRES env var)")
			return nil
		}

		rawDB, err = sql.Open("postgres", dsn)
		if err != nil {
			t.Fatalf("Failed to open PostgreSQL test database: %v", err)
		}
		dialect = database.NewPostgreSQLDialect()
		driverName = "postgres"

		// Test connection
		if err := rawDB.Ping(); err != nil {
			t.Skipf("PostgreSQL test database not available: %v", err)
			return nil
		}

		// Apply PostgreSQL settings
		if err := dialect.ApplySettings(rawDB); err != nil {
			t.Fatalf("Failed to apply PostgreSQL settings: %v", err)
		}

		// Clean test database before use
		cleanPostgreSQLTestDBRaw(t, rawDB)

	default:
		t.Fatalf("Unsupported database type for testing: %s", dbType)
		return nil
	}

	// Run migrations with dialect
	err = database.RunMigrationsWithDialect(rawDB, dialect)
	if err != nil {
		t.Fatalf("Failed to run migrations on %s: %v", dbType, err)
	}

	// Use Go time for cross-database compatibility (not datetime('now') which is SQLite-specific)
	now := time.Now().Format("2006-01-02 15:04:05")

	// Default tenant (id=0) is already created by schema 001_schema.go
	// Just create subscription if it doesn't exist
	_, _ = rawDB.Exec(`
		INSERT OR IGNORE INTO tenant_subscriptions (tenant_id, plan_id, status, created_at, updated_at)
		VALUES (0, 1, 'active', ?, ?)
	`, now, now)

	// Create tenant 1 for cross-tenant security tests (many tests need this)
	_, _ = rawDB.Exec(`
		INSERT OR IGNORE INTO tenants (id, slug, name, status, contact_email, created_at, updated_at)
		VALUES (1, 'tenant-1', 'Tenant 1', 'active', 'tenant1@example.com', ?, ?)
	`, now, now)

	// Color categories for tenant_id=0 are already seeded in schema 001_schema.go
	// No need to insert them here

	// Insert system settings for tenant 0 (default)
	_, _ = rawDB.Exec(`INSERT INTO system_settings (tenant_id, key, value, updated_at) VALUES
		(0, 'booking_advance_days', '14', ?),
		(0, 'cancellation_notice_hours', '12', ?),
		(0, 'auto_deactivation_days', '365', ?),
		(0, 'morning_walk_requires_approval', 'true', ?),
		(0, 'use_feiertage_api', 'false', ?),
		(0, 'feiertage_state', 'BW', ?),
		(0, 'booking_time_granularity', '15', ?),
		(0, 'feiertage_cache_days', '7', ?),
		(0, 'site_logo', '', ?),
		(0, 'registration_password', 'TEST1234', ?),
		(0, 'whatsapp_group_enabled', 'false', ?),
		(0, 'whatsapp_group_link', '', ?),
		(0, 'default_color_for_new_users', '1', ?)
	`, now, now, now, now, now, now, now, now, now, now, now, now, now)

	// Insert booking time rules for tenant 0 (default)
	// Weekday rules with blocked periods (German names for error messages)
	_, _ = rawDB.Exec(`INSERT INTO booking_time_rules (tenant_id, day_type, rule_name, start_time, end_time, is_blocked, created_at, updated_at) VALUES
		(0, 'weekday', 'Vormittag', '08:30', '12:00', 0, ?, ?),
		(0, 'weekday', 'Mittagspause', '12:00', '14:00', 1, ?, ?),
		(0, 'weekday', 'Nachmittag', '14:00', '17:00', 0, ?, ?),
		(0, 'weekday', 'Fütterungszeit', '17:00', '18:00', 1, ?, ?),
		(0, 'weekday', 'Abend', '18:00', '19:00', 0, ?, ?),
		(0, 'weekend', 'Vormittag', '09:00', '12:00', 0, ?, ?),
		(0, 'weekend', 'Nachmittag', '14:00', '17:00', 0, ?, ?),
		(0, 'holiday', 'Vormittag', '10:00', '12:00', 0, ?, ?),
		(0, 'holiday', 'Nachmittag', '14:00', '16:00', 0, ?, ?)
	`, now, now, now, now, now, now, now, now, now, now, now, now, now, now, now, now, now, now)

	// Wrap in database.DB for cross-database support
	sqlxDB := sqlx.NewDb(rawDB, driverName)
	wrappedDB := database.WrapSqlxDB(sqlxDB, dialect)

	// Cleanup after test
	t.Cleanup(func() {
		wrappedDB.Close()
	})

	return wrappedDB
}

// allTables lists all tables in the correct order for deletion (children first, then parents)
// This ensures foreign key constraints are respected
var allTables = []string{
	// Child tables first (have foreign key references)
	"walk_report_photos",
	"walk_reports",
	"color_requests",
	"user_colors",
	"bookings",
	"blocked_dates",
	"experience_requests",
	"reactivation_requests",
	"dogs",
	// SaaS tables
	"demo_tenant_state",
	"tenant_subscriptions",
	"tenant_settings",
	// Parent tables
	"pricing_plans",
	"users",
	"color_categories",
	"booking_time_rules",
	"custom_holidays",
	"feiertage_cache",
	"system_settings",
	"tenants",
	// Migration tracking table
	"schema_migrations",
}

// cleanPostgreSQLTestDB drops all tables in the test database
func cleanPostgreSQLTestDB(t *testing.T, db *database.DB) {
	// Drop all tables with CASCADE to handle foreign keys
	for _, table := range allTables {
		_, _ = db.Exec("DROP TABLE IF EXISTS " + table + " CASCADE")
	}
}

// cleanPostgreSQLTestDBRaw drops all tables in the test database using raw *sql.DB
func cleanPostgreSQLTestDBRaw(t *testing.T, db *sql.DB) {
	// Drop all tables with CASCADE to handle foreign keys
	for _, table := range allTables {
		_, _ = db.Exec("DROP TABLE IF EXISTS " + table + " CASCADE")
	}
}

// dataTables lists tables that contain test data (excludes schema_migrations and pricing_plans)
// These are truncated between tests for fast reset
var dataTables = []string{
	"walk_report_photos",
	"walk_reports",
	"color_requests",
	"user_colors",
	"bookings",
	"blocked_dates",
	"experience_requests",
	"reactivation_requests",
	"dogs",
	"demo_tenant_state",
	"tenant_subscriptions",
	"tenant_settings",
	"users",
	"color_categories",
	"booking_time_rules",
	"custom_holidays",
	"feiertage_cache",
	"system_settings",
	"tenants",
}

// SetupTestDBFast returns a test database using a shared connection for PostgreSQL
// This is MUCH faster than SetupTestDB because it:
// 1. Reuses the database connection across tests
// 2. Truncates tables instead of dropping and recreating them
// 3. Only runs migrations once per test run
// For SQLite, it still creates a fresh in-memory database (already fast)
func SetupTestDBFast(t *testing.T) *database.DB {
	// Determine database type
	dbType := "sqlite"
	if os.Getenv("DB_TEST_POSTGRES") != "" {
		dbType = "postgres"
	}

	// SQLite: always use fresh in-memory database (it's fast enough)
	if dbType == "sqlite" {
		return SetupTestDBWithType(t, "sqlite")
	}

	// PostgreSQL: use shared connection with truncation
	sharedDBMu.Lock()
	defer sharedDBMu.Unlock()

	// Initialize shared connection if needed
	if !sharedDBInited || sharedDBType != dbType {
		if sharedRawDB != nil {
			sharedRawDB.Close()
		}
		initSharedDB(t, dbType)
	}

	// Truncate all data tables and reset to clean state
	truncateAndResetData(t, sharedRawDB, sharedDialect)

	// Don't close the shared connection in cleanup - it's reused
	return sharedDB
}

// initSharedDB initializes the shared database connection
func initSharedDB(t *testing.T, dbType string) {
	var err error
	var driverName string

	switch dbType {
	case "postgres":
		dsn := os.Getenv("DB_TEST_POSTGRES")
		sharedRawDB, err = sql.Open("postgres", dsn)
		if err != nil {
			t.Fatalf("Failed to open PostgreSQL: %v", err)
		}
		sharedDialect = database.NewPostgreSQLDialect()
		driverName = "postgres"
	}

	if err := sharedRawDB.Ping(); err != nil {
		t.Skipf("Database not available: %v", err)
	}

	if err := sharedDialect.ApplySettings(sharedRawDB); err != nil {
		t.Fatalf("Failed to apply settings: %v", err)
	}

	// Drop all tables first to ensure clean state
	cleanPostgreSQLTestDBRaw(t, sharedRawDB)

	// Run migrations once
	if err := database.RunMigrationsWithDialect(sharedRawDB, sharedDialect); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Wrap in database.DB for cross-database support
	sqlxDB := sqlx.NewDb(sharedRawDB, driverName)
	sharedDB = database.WrapSqlxDB(sqlxDB, sharedDialect)

	sharedDBType = dbType
	sharedDBInited = true
}

// truncateAndResetData truncates all data tables and inserts base test data
func truncateAndResetData(t *testing.T, db *sql.DB, dialect database.Dialect) {
	now := time.Now().Format("2006-01-02 15:04:05")

	// Disable FK checks for truncation (PostgreSQL only)
	db.Exec("SET session_replication_role = 'replica'")

	// Truncate data tables (but not schema_migrations or pricing_plans)
	for _, table := range dataTables {
		db.Exec("TRUNCATE TABLE " + table + " CASCADE")
	}

	// Re-enable FK checks
	db.Exec("SET session_replication_role = 'origin'")

	// Insert base test data
	// 1. Test tenant
	_, err := db.Exec(`
		INSERT INTO tenants (id, slug, name, status, contact_email, federal_state, created_at, updated_at)
		VALUES (0, 'default', 'Default', 'active', 'admin@localhost', 'BW', ?, ?)
	`, now, now)
	if err != nil {
		t.Fatalf("Failed to create test tenant: %v", err)
	}

	// 2. Test subscription (plan_id=1 is the 'Free' plan from pricing_plans)
	_, err = db.Exec(`
		INSERT INTO tenant_subscriptions (tenant_id, plan_id, status, created_at, updated_at)
		VALUES (0, 1, 'active', ?, ?)
	`, now, now)
	if err != nil {
		t.Fatalf("Failed to create test subscription: %v", err)
	}

	// 3. Color categories for tenant_id=0 are already seeded in schema 001_schema.go
	// No need to insert them here

	// 4. Default system settings (all 13 settings expected by tests) - use tenant_id=0
	_, _ = db.Exec(`INSERT INTO system_settings (tenant_id, `+"`key`"+`, value, updated_at) VALUES
		(0, 'booking_advance_days', '14', ?),
		(0, 'cancellation_notice_hours', '12', ?),
		(0, 'auto_deactivation_days', '365', ?),
		(0, 'morning_walk_requires_approval', 'true', ?),
		(0, 'use_feiertage_api', 'false', ?),
		(0, 'feiertage_state', 'BW', ?),
		(0, 'booking_time_granularity', '15', ?),
		(0, 'feiertage_cache_days', '7', ?),
		(0, 'site_logo', '', ?),
		(0, 'registration_password', 'TEST1234', ?),
		(0, 'whatsapp_group_enabled', 'false', ?),
		(0, 'whatsapp_group_link', '', ?),
		(0, 'default_color_for_new_users', '1', ?)
	`, now, now, now, now, now, now, now, now, now, now, now, now, now)

	// 5. Default booking time rules (matches expected test data, German names) - use tenant_id=0
	_, _ = db.Exec(`INSERT INTO booking_time_rules (tenant_id, day_type, rule_name, start_time, end_time, is_blocked, created_at, updated_at) VALUES
		(0, 'weekday', 'Vormittag', '08:30', '12:00', 0, ?, ?),
		(0, 'weekday', 'Mittagspause', '12:00', '14:00', 1, ?, ?),
		(0, 'weekday', 'Nachmittag', '14:00', '17:00', 0, ?, ?),
		(0, 'weekday', 'Fütterungszeit', '17:00', '18:00', 1, ?, ?),
		(0, 'weekday', 'Abend', '18:00', '19:00', 0, ?, ?),
		(0, 'weekend', 'Vormittag', '09:00', '12:00', 0, ?, ?),
		(0, 'weekend', 'Nachmittag', '14:00', '17:00', 0, ?, ?),
		(0, 'holiday', 'Vormittag', '10:00', '12:00', 0, ?, ?),
		(0, 'holiday', 'Nachmittag', '14:00', '16:00', 0, ?, ?)
	`, now, now, now, now, now, now, now, now, now, now, now, now, now, now, now, now, now, now)
}

// DONE: SeedTestUser creates a test user and returns the ID
// Name is split: first word = first_name, rest = last_name
// Also assigns colors based on level parameter for the color system
func SeedTestUser(t *testing.T, db *database.DB, email, name, level string) int {
	now := time.Now()

	// Split name into first_name and last_name
	firstName := name
	lastName := ""
	parts := splitName(name)
	if len(parts) > 0 {
		firstName = parts[0]
		if len(parts) > 1 {
			lastName = parts[1]
		}
	}

	result, err := db.Exec(`
		INSERT INTO users (tenant_id, email, first_name, last_name, phone, password_hash, is_verified, is_active, terms_accepted_at, last_activity_at, created_at)
		VALUES (0, ?, ?, ?, ?, ?, 1, 1, ?, ?, ?)
	`, email, firstName, lastName, "+49 123 456789", "test_hash", now, now, now)

	if err != nil {
		t.Fatalf("Failed to seed test user: %v", err)
	}

	userID, _ := result.LastInsertId()

	// Assign colors based on level parameter for the color system
	// Query color IDs from database by name (case-insensitive) for tenant 1
	colorNamesByLevel := map[string][]string{
		"green":  {"gruen"},                                        // only gruen
		"orange": {"gruen", "gelb", "orange"},                      // gruen, gelb, orange
		"blue":   {"gruen", "gelb", "orange", "hellblau", "dunkelblau"}, // all main colors
	}
	colorNames, ok := colorNamesByLevel[level]
	if !ok {
		colorNames = colorNamesByLevel["green"] // default to green
	}

	for _, colorName := range colorNames {
		var colorID int
		err := db.QueryRow(`SELECT id FROM color_categories WHERE tenant_id = 0 AND LOWER(name) = LOWER(?)`, colorName).Scan(&colorID)
		if err != nil {
			// Color might not exist in test DB - that's ok for some tests
			continue
		}
		_, _ = db.Exec(`INSERT INTO user_colors (tenant_id, user_id, color_id) VALUES (0, ?, ?)`, userID, colorID)
	}

	return int(userID)
}

// SeedTestUserWithoutColors creates a test user without assigning any colors
// Use this for tests that specifically need to test users with no color assignments
func SeedTestUserWithoutColors(t *testing.T, db *database.DB, email, name, level string) int {
	now := time.Now()

	// Split name into first_name and last_name
	firstName := name
	lastName := ""
	parts := splitName(name)
	if len(parts) > 0 {
		firstName = parts[0]
		if len(parts) > 1 {
			lastName = parts[1]
		}
	}

	result, err := db.Exec(`
		INSERT INTO users (tenant_id, email, first_name, last_name, phone, password_hash, is_verified, is_active, terms_accepted_at, last_activity_at, created_at)
		VALUES (0, ?, ?, ?, ?, ?, 1, 1, ?, ?, ?)
	`, email, firstName, lastName, "+49 123 456789", "test_hash", now, now, now)

	if err != nil {
		t.Fatalf("Failed to seed test user without colors: %v", err)
	}

	id, _ := result.LastInsertId()
	return int(id)
}

// splitName splits a name into first and last name parts
func splitName(name string) []string {
	parts := []string{}
	current := ""
	for _, r := range name {
		if r == ' ' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	// First word is first_name, rest joined is last_name
	if len(parts) == 0 {
		return []string{"", ""}
	}
	if len(parts) == 1 {
		return []string{parts[0], ""}
	}
	lastName := ""
	for i := 1; i < len(parts); i++ {
		if i > 1 {
			lastName += " "
		}
		lastName += parts[i]
	}
	return []string{parts[0], lastName}
}

// DONE: SeedTestDog creates a test dog and returns the ID
// Sets color_id based on category parameter for the color system
func SeedTestDog(t *testing.T, db *database.DB, name, breed, category string) int {
	now := time.Now()

	// Map category to color name for the color system
	colorNameByCategory := map[string]string{
		"green":  "gruen",     // green dogs
		"orange": "orange",    // orange dogs
		"blue":   "dunkelblau", // blue dogs
	}
	colorName, ok := colorNameByCategory[category]
	if !ok {
		colorName = "gruen" // default to green
	}

	// Query color ID from database
	var colorID int
	err := db.QueryRow(`SELECT id FROM color_categories WHERE tenant_id = 0 AND LOWER(name) = LOWER(?)`, colorName).Scan(&colorID)
	if err != nil {
		t.Fatalf("Failed to find color %s for dog: %v", colorName, err)
	}

	result, err := db.Exec(`
		INSERT INTO dogs (tenant_id, name, breed, size, age, color_id, is_available, created_at)
		VALUES (0, ?, ?, ?, ?, ?, 1, ?)
	`, name, breed, "medium", 5, colorID, now)

	if err != nil {
		t.Fatalf("Failed to seed test dog: %v", err)
	}

	id, _ := result.LastInsertId()
	return int(id)
}

// DONE: SeedTestBooking creates a test booking and returns the ID
func SeedTestBooking(t *testing.T, db *database.DB, userID, dogID int, date, scheduledTime, status string) int {
	now := time.Now()
	result, err := db.Exec(`
		INSERT INTO bookings (tenant_id, user_id, dog_id, date, scheduled_time, status, created_at)
		VALUES (0, ?, ?, ?, ?, ?, ?)
	`, userID, dogID, date, scheduledTime, status, now)

	if err != nil {
		t.Fatalf("Failed to seed test booking: %v", err)
	}

	id, _ := result.LastInsertId()
	return int(id)
}

// DONE: SeedTestBlockedDate creates a test blocked date and returns the ID
func SeedTestBlockedDate(t *testing.T, db *database.DB, date, reason string, createdBy int) int {
	now := time.Now()
	result, err := db.Exec(`
		INSERT INTO blocked_dates (tenant_id, date, reason, created_by, created_at)
		VALUES (0, ?, ?, ?, ?)
	`, date, reason, createdBy, now)

	if err != nil {
		t.Fatalf("Failed to seed test blocked date: %v", err)
	}

	id, _ := result.LastInsertId()
	return int(id)
}

// SeedTestBlockedDateForDog creates a test blocked date for a specific dog and returns the ID
func SeedTestBlockedDateForDog(t *testing.T, db *database.DB, date, reason string, createdBy int, dogID int) int {
	now := time.Now()
	result, err := db.Exec(`
		INSERT INTO blocked_dates (tenant_id, date, reason, created_by, dog_id, created_at)
		VALUES (0, ?, ?, ?, ?, ?)
	`, date, reason, createdBy, dogID, now)

	if err != nil {
		t.Fatalf("Failed to seed test blocked date for dog: %v", err)
	}

	id, _ := result.LastInsertId()
	return int(id)
}

// DONE: SeedTestExperienceRequest creates a test experience request and returns the ID
func SeedTestExperienceRequest(t *testing.T, db *database.DB, userID int, requestedLevel, status string) int {
	now := time.Now()
	result, err := db.Exec(`
		INSERT INTO experience_requests (tenant_id, user_id, requested_level, status, created_at)
		VALUES (0, ?, ?, ?, ?)
	`, userID, requestedLevel, status, now)

	if err != nil {
		t.Fatalf("Failed to seed test experience request: %v", err)
	}

	id, _ := result.LastInsertId()
	return int(id)
}

// SeedTestWalkReport creates a test walk report and returns the ID
func SeedTestWalkReport(t *testing.T, db *database.DB, bookingID int, behaviorRating int, energyLevel, notes string) int {
	now := time.Now()
	result, err := db.Exec(`
		INSERT INTO walk_reports (tenant_id, booking_id, behavior_rating, energy_level, notes, created_at, updated_at)
		VALUES (0, ?, ?, ?, ?, ?, ?)
	`, bookingID, behaviorRating, energyLevel, notes, now, now)

	if err != nil {
		t.Fatalf("Failed to seed test walk report: %v", err)
	}

	id, _ := result.LastInsertId()
	return int(id)
}

// SeedTestColorCategory creates a test color category and returns the ID
func SeedTestColorCategory(t *testing.T, db *database.DB, name, hexCode string, sortOrder int) int {
	now := time.Now()
	result, err := db.Exec(`
		INSERT INTO color_categories (tenant_id, name, hex_code, pattern_icon, sort_order, created_at, updated_at)
		VALUES (0, ?, ?, ?, ?, ?, ?)
	`, name, hexCode, "circle", sortOrder, now, now)

	if err != nil {
		t.Fatalf("Failed to seed test color category: %v", err)
	}

	id, _ := result.LastInsertId()
	return int(id)
}

// SeedTestUserColor adds a color to a user
func SeedTestUserColor(t *testing.T, db *database.DB, userID, colorID int) {
	now := time.Now()
	_, err := db.Exec(`
		INSERT INTO user_colors (tenant_id, user_id, color_id, granted_at)
		VALUES (0, ?, ?, ?)
	`, userID, colorID, now)

	if err != nil {
		t.Fatalf("Failed to seed test user color: %v", err)
	}
}

// SeedTestColorRequest creates a test color request and returns the ID
func SeedTestColorRequest(t *testing.T, db *database.DB, userID, colorID int, status string) int {
	now := time.Now()
	result, err := db.Exec(`
		INSERT INTO color_requests (tenant_id, user_id, color_id, status, created_at)
		VALUES (0, ?, ?, ?, ?)
	`, userID, colorID, status, now)

	if err != nil {
		t.Fatalf("Failed to seed test color request: %v", err)
	}

	id, _ := result.LastInsertId()
	return int(id)
}

// SeedTestDogCustom creates a test dog with custom parameters and returns the ID
// colorID should be a valid color ID from color_categories table
func SeedTestDogCustom(t *testing.T, db *database.DB, name, breed, size string, age int, colorID int) int {
	now := time.Now()
	result, err := db.Exec(`
		INSERT INTO dogs (tenant_id, name, breed, size, age, color_id, is_available, created_at, updated_at)
		VALUES (0, ?, ?, ?, ?, ?, 1, ?, ?)
	`, name, breed, size, age, colorID, now, now)

	if err != nil {
		t.Fatalf("Failed to seed test dog: %v", err)
	}

	id, _ := result.LastInsertId()
	return int(id)
}

// SeedUserColor is an alias for SeedTestUserColor for backward compatibility
func SeedUserColor(t *testing.T, db *database.DB, userID, colorID int) {
	SeedTestUserColor(t, db, userID, colorID)
}

// GetFutureDate returns a date string N days in the future
func GetFutureDate(daysFromNow int) string {
	return time.Now().AddDate(0, 0, daysFromNow).Format("2006-01-02")
}

// Now returns the current timestamp formatted for SQL queries
// Use this instead of SQLite-specific datetime('now')
func Now() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// NowTime returns the current time as a time.Time for SQL queries
func NowTime() time.Time {
	return time.Now()
}

// InsertAndGetID executes an INSERT statement and returns the last inserted ID
// This is a cross-database compatible way to handle INSERT ... RETURNING id
// which is not supported in MySQL
func InsertAndGetID(t *testing.T, db *database.DB, query string, args ...interface{}) int {
	result, err := db.Exec(query, args...)
	if err != nil {
		t.Fatalf("Failed to execute INSERT: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get last insert ID: %v", err)
	}
	return int(id)
}

// DONE: CountRows returns the count of rows in a table
func CountRows(t *testing.T, db *database.DB, table string) int {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count rows in %s: %v", table, err)
	}
	return count
}

// DONE: ClearTable deletes all rows from a table
func ClearTable(t *testing.T, db *database.DB, table string) {
	_, err := db.Exec("DELETE FROM " + table)
	if err != nil {
		t.Fatalf("Failed to clear table %s: %v", table, err)
	}
}

// ContextWithTenantID returns a context with the tenant ID set
// Used in tests to simulate authenticated tenant context
// NOTE: Uses local key that matches middleware.TenantIDKey value
func ContextWithTenantID(ctx context.Context, tenantID int) context.Context {
	return context.WithValue(ctx, testTenantIDKey, tenantID)
}

// ContextWithUserID returns a context with the user ID set
// Used in tests to simulate authenticated user context
// NOTE: Uses local key that matches middleware.UserIDKey value
func ContextWithUserID(ctx context.Context, userID int) context.Context {
	return context.WithValue(ctx, testUserIDKey, userID)
}

// ContextWithAuth returns a context with both tenant ID and user ID set
func ContextWithAuth(ctx context.Context, tenantID, userID int) context.Context {
	ctx = context.WithValue(ctx, testTenantIDKey, tenantID)
	ctx = context.WithValue(ctx, testUserIDKey, userID)
	return ctx
}

// GetTenantIDKey returns the tenant ID context key for use in handler tests
// This allows handler tests to use context.WithValue directly with the correct key type
func GetTenantIDKey() interface{} {
	return testTenantIDKey
}

// GetUserIDKey returns the user ID context key for use in handler tests
func GetUserIDKey() interface{} {
	return testUserIDKey
}

// SetupBenchmarkDB creates a test database for benchmark tests
// This function is similar to SetupTestDB but uses *testing.B for error handling
func SetupBenchmarkDB(b *testing.B) *database.DB {
	b.Helper()

	// Use in-memory SQLite for benchmarks
	rawDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		b.Fatalf("Failed to open SQLite test database: %v", err)
	}

	dialect := database.NewSQLiteDialect()

	// Set max connections to 1 for in-memory database
	rawDB.SetMaxOpenConns(1)

	// Apply SQLite settings
	if err := dialect.ApplySettings(rawDB); err != nil {
		b.Fatalf("Failed to apply SQLite settings: %v", err)
	}

	// Wrap in sqlx.DB
	sqlxDB := sqlx.NewDb(rawDB, "sqlite")

	// Create wrapped database.DB
	db := database.WrapDB(sqlxDB, dialect)

	// Run migrations
	if err := database.RunMigrationsWithDialect(rawDB, dialect); err != nil {
		b.Fatalf("Failed to run migrations: %v", err)
	}

	// Seed basic test data
	seedBenchmarkDB(b, db)

	return db
}

// seedBenchmarkDB seeds the benchmark database with minimal required data
func seedBenchmarkDB(b *testing.B, db *database.DB) {
	b.Helper()

	now := time.Now()

	// Create default tenant for tenant_id=0
	_, _ = db.Exec(`INSERT OR IGNORE INTO tenants (id, slug, name, status, contact_email, created_at, updated_at) VALUES (0, 'default', 'Default', 'active', 'admin@test.com', ?, ?)`, now, now)

	// Create subscription for tenant
	_, _ = db.Exec(`INSERT INTO tenant_subscriptions (tenant_id, plan_id, status, created_at, updated_at) VALUES (0, 1, 'active', ?, ?)`, now, now)

	// Seed color categories for tenant_id=0
	_, _ = db.Exec(`INSERT OR IGNORE INTO color_categories (id, tenant_id, name, hex_code, description, experience_level, display_order, created_at, updated_at) VALUES
		(1, 0, 'gruen', '#22c55e', 'Anfänger-Hunde', 1, 1, ?, ?),
		(2, 0, 'dunkelblau', '#3b82f6', 'Fortgeschrittene Hunde', 2, 2, ?, ?),
		(3, 0, 'orange', '#f97316', 'Erfahrene Gassigeher', 3, 3, ?, ?)
	`, now, now, now, now, now, now)
}

// TestConfig returns a config suitable for testing
// Uses test.local as BaseDomain for domain-related tests
func TestConfig() *config.Config {
	return &config.Config{
		Port:       "8080",
		BaseURL:    "http://localhost:8080",
		BaseDomain: "test.local",
		JWTSecret:  "test-secret-key-for-testing-only",
	}
}
