package testutil

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
	"github.com/tranmh/gassigeher/internal/database"
)

// Shared database connection for MySQL/PostgreSQL tests
// This avoids recreating the schema for each test
var (
	sharedDB       *sql.DB
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
// It checks for DB_TEST_MYSQL and DB_TEST_POSTGRES environment variables
// and uses the corresponding database if available. Falls back to SQLite.
// This enables running the same tests against all databases by setting env vars.
//
// For MySQL/PostgreSQL, this uses a shared connection with table truncation
// instead of dropping and recreating tables for each test (10x+ faster).
func SetupTestDB(t *testing.T) *sql.DB {
	// Use the fast version that reuses connections for MySQL/PostgreSQL
	return SetupTestDBFast(t)
}

// SetupTestDBWithType creates a test database of the specified type
// Supports: sqlite (in-memory), mysql, postgres
// For MySQL/PostgreSQL, requires test database to be available (via Docker or local install)
func SetupTestDBWithType(t *testing.T, dbType string) *sql.DB {
	var db *sql.DB
	var dialect database.Dialect
	var err error

	switch dbType {
	case "sqlite", "":
		// Use in-memory SQLite for fast testing
		db, err = sql.Open("sqlite", ":memory:")
		if err != nil {
			t.Fatalf("Failed to open SQLite test database: %v", err)
		}
		dialect = database.NewSQLiteDialect()

		// Set max connections to 1 to avoid issues with in-memory databases
		// (each connection would get its own database otherwise)
		db.SetMaxOpenConns(1)

		// Apply SQLite settings (PRAGMA foreign_keys, etc.)
		if err := dialect.ApplySettings(db); err != nil {
			t.Fatalf("Failed to apply SQLite settings: %v", err)
		}

	case "mysql":
		// Use test MySQL database (requires DB_TEST_MYSQL env var)
		dsn := os.Getenv("DB_TEST_MYSQL")
		if dsn == "" {
			t.Skip("MySQL test database not configured (set DB_TEST_MYSQL env var)")
			return nil
		}

		// Ensure multiStatements=true is enabled for running migrations with multiple statements
		if !strings.Contains(dsn, "multiStatements=true") {
			if strings.Contains(dsn, "?") {
				dsn = dsn + "&multiStatements=true"
			} else {
				dsn = dsn + "?multiStatements=true"
			}
		}

		db, err = sql.Open("mysql", dsn)
		if err != nil {
			t.Fatalf("Failed to open MySQL test database: %v", err)
		}
		dialect = database.NewMySQLDialect()

		// Test connection
		if err := db.Ping(); err != nil {
			t.Skipf("MySQL test database not available: %v", err)
			return nil
		}

		// Apply MySQL settings
		if err := dialect.ApplySettings(db); err != nil {
			t.Fatalf("Failed to apply MySQL settings: %v", err)
		}

		// Clean test database before use
		cleanMySQLTestDB(t, db)

	case "postgres":
		// Use test PostgreSQL database (requires DB_TEST_POSTGRES env var)
		dsn := os.Getenv("DB_TEST_POSTGRES")
		if dsn == "" {
			t.Skip("PostgreSQL test database not configured (set DB_TEST_POSTGRES env var)")
			return nil
		}

		db, err = sql.Open("postgres", dsn)
		if err != nil {
			t.Fatalf("Failed to open PostgreSQL test database: %v", err)
		}
		dialect = database.NewPostgreSQLDialect()

		// Test connection
		if err := db.Ping(); err != nil {
			t.Skipf("PostgreSQL test database not available: %v", err)
			return nil
		}

		// Apply PostgreSQL settings
		if err := dialect.ApplySettings(db); err != nil {
			t.Fatalf("Failed to apply PostgreSQL settings: %v", err)
		}

		// Clean test database before use
		cleanPostgreSQLTestDB(t, db)

	default:
		t.Fatalf("Unsupported database type for testing: %s", dbType)
	}

	// Run migrations with dialect
	err = database.RunMigrationsWithDialect(db, dialect)
	if err != nil {
		t.Fatalf("Failed to run migrations on %s: %v", dbType, err)
	}

	// Use Go time for cross-database compatibility (not datetime('now') which is SQLite-specific)
	now := time.Now().Format("2006-01-02 15:04:05")

	// Create a test tenant with id=1 for all tests
	_, err = db.Exec(`
		INSERT INTO tenants (id, slug, name, status, contact_email, federal_state, created_at, updated_at)
		VALUES (1, 'test-tenant', 'Test Tenant', 'active', 'test@example.com', 'BW', ?, ?)
	`, now, now)
	if err != nil {
		t.Fatalf("Failed to create test tenant: %v", err)
	}

	// SaaS: Create default free subscription for test tenant
	// Migration 009 seeds subscriptions for existing tenants, but tenant 1 is created after migrations
	_, err = db.Exec(`
		INSERT INTO tenant_subscriptions (tenant_id, plan_id, status, created_at, updated_at)
		VALUES (1, 1, 'active', ?, ?)
	`, now, now)
	if err != nil {
		t.Fatalf("Failed to create test subscription: %v", err)
	}

	// Update all default seed data to belong to test tenant
	// This updates the default data inserted by migration 002 which has tenant_id = NULL
	_, _ = db.Exec(`UPDATE color_categories SET tenant_id = 1 WHERE tenant_id IS NULL`)
	_, _ = db.Exec(`UPDATE booking_time_rules SET tenant_id = 1 WHERE tenant_id IS NULL`)
	_, _ = db.Exec(`UPDATE system_settings SET tenant_id = 1 WHERE tenant_id IS NULL`)

	// Cleanup after test
	t.Cleanup(func() {
		db.Close()
	})

	return db
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

// cleanMySQLTestDB drops all tables in the test database
func cleanMySQLTestDB(t *testing.T, db *sql.DB) {
	// Disable foreign key checks temporarily
	_, _ = db.Exec("SET FOREIGN_KEY_CHECKS = 0")

	// Drop all tables
	for _, table := range allTables {
		_, _ = db.Exec("DROP TABLE IF EXISTS " + table)
	}

	// Re-enable foreign key checks
	_, _ = db.Exec("SET FOREIGN_KEY_CHECKS = 1")
}

// cleanPostgreSQLTestDB drops all tables in the test database
func cleanPostgreSQLTestDB(t *testing.T, db *sql.DB) {
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

// SetupTestDBFast returns a test database using a shared connection for MySQL/PostgreSQL
// This is MUCH faster than SetupTestDB because it:
// 1. Reuses the database connection across tests
// 2. Truncates tables instead of dropping and recreating them
// 3. Only runs migrations once per test run
// For SQLite, it still creates a fresh in-memory database (already fast)
func SetupTestDBFast(t *testing.T) *sql.DB {
	// Determine database type
	dbType := "sqlite"
	if os.Getenv("DB_TEST_MYSQL") != "" {
		dbType = "mysql"
	} else if os.Getenv("DB_TEST_POSTGRES") != "" {
		dbType = "postgres"
	}

	// SQLite: always use fresh in-memory database (it's fast enough)
	if dbType == "sqlite" {
		return SetupTestDBWithType(t, "sqlite")
	}

	// MySQL/PostgreSQL: use shared connection with truncation
	sharedDBMu.Lock()
	defer sharedDBMu.Unlock()

	// Initialize shared connection if needed
	if !sharedDBInited || sharedDBType != dbType {
		if sharedDB != nil {
			sharedDB.Close()
		}
		initSharedDB(t, dbType)
	}

	// Truncate all data tables and reset to clean state
	truncateAndResetData(t, sharedDB, sharedDialect)

	// Don't close the shared connection in cleanup - it's reused
	return sharedDB
}

// initSharedDB initializes the shared database connection
func initSharedDB(t *testing.T, dbType string) {
	var err error

	switch dbType {
	case "mysql":
		dsn := os.Getenv("DB_TEST_MYSQL")
		if !strings.Contains(dsn, "multiStatements=true") {
			if strings.Contains(dsn, "?") {
				dsn = dsn + "&multiStatements=true"
			} else {
				dsn = dsn + "?multiStatements=true"
			}
		}
		sharedDB, err = sql.Open("mysql", dsn)
		if err != nil {
			t.Fatalf("Failed to open MySQL: %v", err)
		}
		sharedDialect = database.NewMySQLDialect()

	case "postgres":
		dsn := os.Getenv("DB_TEST_POSTGRES")
		sharedDB, err = sql.Open("postgres", dsn)
		if err != nil {
			t.Fatalf("Failed to open PostgreSQL: %v", err)
		}
		sharedDialect = database.NewPostgreSQLDialect()
	}

	if err := sharedDB.Ping(); err != nil {
		t.Skipf("Database not available: %v", err)
	}

	if err := sharedDialect.ApplySettings(sharedDB); err != nil {
		t.Fatalf("Failed to apply settings: %v", err)
	}

	// Drop all tables first to ensure clean state
	if dbType == "mysql" {
		cleanMySQLTestDB(t, sharedDB)
	} else {
		cleanPostgreSQLTestDB(t, sharedDB)
	}

	// Run migrations once
	if err := database.RunMigrationsWithDialect(sharedDB, sharedDialect); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	sharedDBType = dbType
	sharedDBInited = true
}

// truncateAndResetData truncates all data tables and inserts base test data
func truncateAndResetData(t *testing.T, db *sql.DB, dialect database.Dialect) {
	now := time.Now().Format("2006-01-02 15:04:05")

	// Disable FK checks for truncation
	switch dialect.Name() {
	case "mysql":
		db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	case "postgres":
		db.Exec("SET session_replication_role = 'replica'")
	}

	// Truncate data tables (but not schema_migrations or pricing_plans)
	for _, table := range dataTables {
		switch dialect.Name() {
		case "mysql":
			db.Exec("TRUNCATE TABLE " + table)
		case "postgres":
			db.Exec("TRUNCATE TABLE " + table + " CASCADE")
		}
	}

	// Re-enable FK checks
	switch dialect.Name() {
	case "mysql":
		db.Exec("SET FOREIGN_KEY_CHECKS = 1")
	case "postgres":
		db.Exec("SET session_replication_role = 'origin'")
	}

	// Insert base test data
	// 1. Test tenant
	_, err := db.Exec(`
		INSERT INTO tenants (id, slug, name, status, contact_email, federal_state, created_at, updated_at)
		VALUES (1, 'test-tenant', 'Test Tenant', 'active', 'test@example.com', 'BW', ?, ?)
	`, now, now)
	if err != nil {
		t.Fatalf("Failed to create test tenant: %v", err)
	}

	// 2. Test subscription
	_, err = db.Exec(`
		INSERT INTO tenant_subscriptions (tenant_id, plan_id, status, created_at, updated_at)
		VALUES (1, 1, 'active', ?, ?)
	`, now, now)
	if err != nil {
		t.Fatalf("Failed to create test subscription: %v", err)
	}

	// 3. Default color categories (from migration 002)
	_, _ = db.Exec(`INSERT INTO color_categories (tenant_id, name, hex_code, pattern_icon, sort_order, created_at, updated_at) VALUES
		(1, 'gruen', '#82b965', 'circle', 1, ?, ?),
		(1, 'gelb', '#f9c74f', 'triangle', 2, ?, ?),
		(1, 'orange', '#f3722c', 'square', 3, ?, ?),
		(1, 'hellblau', '#90e0ef', 'diamond', 4, ?, ?),
		(1, 'dunkelblau', '#4361ee', 'star', 5, ?, ?)
	`, now, now, now, now, now, now, now, now, now, now)

	// 4. Default system settings
	_, _ = db.Exec(`INSERT INTO system_settings (tenant_id, ` + "`key`" + `, value, updated_at) VALUES
		(1, 'booking_advance_days', '14', ?),
		(1, 'cancellation_notice_hours', '12', ?),
		(1, 'auto_deactivation_days', '365', ?)
	`, now, now, now)

	// 5. Default booking time rules (simplified set)
	_, _ = db.Exec(`INSERT INTO booking_time_rules (tenant_id, day_type, rule_name, start_time, end_time, is_blocked, created_at, updated_at) VALUES
		(1, 'weekday', 'morning', '08:00', '12:00', 0, ?, ?),
		(1, 'weekday', 'afternoon', '14:00', '18:00', 0, ?, ?),
		(1, 'weekend', 'morning', '09:00', '12:00', 0, ?, ?),
		(1, 'weekend', 'afternoon', '14:00', '17:00', 0, ?, ?),
		(1, 'holiday', 'morning', '10:00', '12:00', 0, ?, ?),
		(1, 'holiday', 'afternoon', '14:00', '16:00', 0, ?, ?)
	`, now, now, now, now, now, now, now, now, now, now, now, now)
}

// DONE: SeedTestUser creates a test user and returns the ID
// Name is split: first word = first_name, rest = last_name
// Also assigns colors based on level parameter for the color system
func SeedTestUser(t *testing.T, db *sql.DB, email, name, level string) int {
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
		VALUES (1, ?, ?, ?, ?, ?, 1, 1, ?, ?, ?)
	`, email, firstName, lastName, "+49 123 456789", "test_hash", now, now, now)

	if err != nil {
		t.Fatalf("Failed to seed test user: %v", err)
	}

	userID, _ := result.LastInsertId()

	// Assign colors based on level parameter for the color system
	// Color IDs: 1=gruen, 2=gelb, 3=orange, 4=hellblau, 5=dunkelblau
	colorsByLevel := map[string][]int{
		"green":  {1},             // only gruen
		"orange": {1, 2, 3},       // gruen, gelb, orange
		"blue":   {1, 2, 3, 4, 5}, // all main colors
	}
	colors, ok := colorsByLevel[level]
	if !ok {
		colors = colorsByLevel["green"] // default to green
	}

	for _, colorID := range colors {
		_, err := db.Exec(`INSERT INTO user_colors (tenant_id, user_id, color_id) VALUES (1, ?, ?)`, userID, colorID)
		if err != nil {
			// Color might not exist in test DB - that's ok for some tests
			// Just log but don't fail
		}
	}

	return int(userID)
}

// SeedTestUserWithoutColors creates a test user without assigning any colors
// Use this for tests that specifically need to test users with no color assignments
func SeedTestUserWithoutColors(t *testing.T, db *sql.DB, email, name, level string) int {
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
		VALUES (1, ?, ?, ?, ?, ?, 1, 1, ?, ?, ?)
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
func SeedTestDog(t *testing.T, db *sql.DB, name, breed, category string) int {
	now := time.Now()

	// Map category to color_id for the color system
	// Color IDs: 1=gruen, 2=gelb, 3=orange, 4=hellblau, 5=dunkelblau
	colorByCategory := map[string]int{
		"green":  1, // gruen
		"orange": 3, // orange
		"blue":   5, // dunkelblau
	}
	colorID, ok := colorByCategory[category]
	if !ok {
		colorID = 1 // default to gruen
	}

	result, err := db.Exec(`
		INSERT INTO dogs (tenant_id, name, breed, size, age, color_id, is_available, created_at)
		VALUES (1, ?, ?, ?, ?, ?, 1, ?)
	`, name, breed, "medium", 5, colorID, now)

	if err != nil {
		t.Fatalf("Failed to seed test dog: %v", err)
	}

	id, _ := result.LastInsertId()
	return int(id)
}

// DONE: SeedTestBooking creates a test booking and returns the ID
func SeedTestBooking(t *testing.T, db *sql.DB, userID, dogID int, date, scheduledTime, status string) int {
	now := time.Now()
	result, err := db.Exec(`
		INSERT INTO bookings (tenant_id, user_id, dog_id, date, scheduled_time, status, created_at)
		VALUES (1, ?, ?, ?, ?, ?, ?)
	`, userID, dogID, date, scheduledTime, status, now)

	if err != nil {
		t.Fatalf("Failed to seed test booking: %v", err)
	}

	id, _ := result.LastInsertId()
	return int(id)
}

// DONE: SeedTestBlockedDate creates a test blocked date and returns the ID
func SeedTestBlockedDate(t *testing.T, db *sql.DB, date, reason string, createdBy int) int {
	now := time.Now()
	result, err := db.Exec(`
		INSERT INTO blocked_dates (tenant_id, date, reason, created_by, created_at)
		VALUES (1, ?, ?, ?, ?)
	`, date, reason, createdBy, now)

	if err != nil {
		t.Fatalf("Failed to seed test blocked date: %v", err)
	}

	id, _ := result.LastInsertId()
	return int(id)
}

// SeedTestBlockedDateForDog creates a test blocked date for a specific dog and returns the ID
func SeedTestBlockedDateForDog(t *testing.T, db *sql.DB, date, reason string, createdBy int, dogID int) int {
	now := time.Now()
	result, err := db.Exec(`
		INSERT INTO blocked_dates (tenant_id, date, reason, created_by, dog_id, created_at)
		VALUES (1, ?, ?, ?, ?, ?)
	`, date, reason, createdBy, dogID, now)

	if err != nil {
		t.Fatalf("Failed to seed test blocked date for dog: %v", err)
	}

	id, _ := result.LastInsertId()
	return int(id)
}

// DONE: SeedTestExperienceRequest creates a test experience request and returns the ID
func SeedTestExperienceRequest(t *testing.T, db *sql.DB, userID int, requestedLevel, status string) int {
	now := time.Now()
	result, err := db.Exec(`
		INSERT INTO experience_requests (tenant_id, user_id, requested_level, status, created_at)
		VALUES (1, ?, ?, ?, ?)
	`, userID, requestedLevel, status, now)

	if err != nil {
		t.Fatalf("Failed to seed test experience request: %v", err)
	}

	id, _ := result.LastInsertId()
	return int(id)
}

// SeedTestWalkReport creates a test walk report and returns the ID
func SeedTestWalkReport(t *testing.T, db *sql.DB, bookingID int, behaviorRating int, energyLevel, notes string) int {
	now := time.Now()
	result, err := db.Exec(`
		INSERT INTO walk_reports (tenant_id, booking_id, behavior_rating, energy_level, notes, created_at, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, ?)
	`, bookingID, behaviorRating, energyLevel, notes, now, now)

	if err != nil {
		t.Fatalf("Failed to seed test walk report: %v", err)
	}

	id, _ := result.LastInsertId()
	return int(id)
}

// SeedTestColorCategory creates a test color category and returns the ID
func SeedTestColorCategory(t *testing.T, db *sql.DB, name, hexCode string, sortOrder int) int {
	now := time.Now()
	result, err := db.Exec(`
		INSERT INTO color_categories (tenant_id, name, hex_code, pattern_icon, sort_order, created_at, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, ?)
	`, name, hexCode, "circle", sortOrder, now, now)

	if err != nil {
		t.Fatalf("Failed to seed test color category: %v", err)
	}

	id, _ := result.LastInsertId()
	return int(id)
}

// SeedTestUserColor adds a color to a user
func SeedTestUserColor(t *testing.T, db *sql.DB, userID, colorID int) {
	now := time.Now()
	_, err := db.Exec(`
		INSERT INTO user_colors (tenant_id, user_id, color_id, granted_at)
		VALUES (1, ?, ?, ?)
	`, userID, colorID, now)

	if err != nil {
		t.Fatalf("Failed to seed test user color: %v", err)
	}
}

// SeedTestColorRequest creates a test color request and returns the ID
func SeedTestColorRequest(t *testing.T, db *sql.DB, userID, colorID int, status string) int {
	now := time.Now()
	result, err := db.Exec(`
		INSERT INTO color_requests (tenant_id, user_id, color_id, status, created_at)
		VALUES (1, ?, ?, ?, ?)
	`, userID, colorID, status, now)

	if err != nil {
		t.Fatalf("Failed to seed test color request: %v", err)
	}

	id, _ := result.LastInsertId()
	return int(id)
}

// SeedTestDogCustom creates a test dog with custom parameters and returns the ID
// colorID should be a valid color ID from color_categories table
func SeedTestDogCustom(t *testing.T, db *sql.DB, name, breed, size string, age int, colorID int) int {
	now := time.Now()
	result, err := db.Exec(`
		INSERT INTO dogs (tenant_id, name, breed, size, age, color_id, is_available, created_at, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, 1, ?, ?)
	`, name, breed, size, age, colorID, now, now)

	if err != nil {
		t.Fatalf("Failed to seed test dog: %v", err)
	}

	id, _ := result.LastInsertId()
	return int(id)
}

// SeedUserColor is an alias for SeedTestUserColor for backward compatibility
func SeedUserColor(t *testing.T, db *sql.DB, userID, colorID int) {
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
func InsertAndGetID(t *testing.T, db *sql.DB, query string, args ...interface{}) int {
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
func CountRows(t *testing.T, db *sql.DB, table string) int {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count rows in %s: %v", table, err)
	}
	return count
}

// DONE: ClearTable deletes all rows from a table
func ClearTable(t *testing.T, db *sql.DB, table string) {
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
