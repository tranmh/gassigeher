package testutil

import (
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
	"github.com/tranmh/gassigeher/internal/database"
)

// SetupTestDB creates a test database (default: in-memory SQLite)
// For backward compatibility, this defaults to SQLite
// Use SetupTestDBWithType() to test with MySQL or PostgreSQL
func SetupTestDB(t *testing.T) *sql.DB {
	return SetupTestDBWithType(t, "sqlite")
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

	// Cleanup after test
	t.Cleanup(func() {
		db.Close()
	})

	return db
}

// cleanMySQLTestDB drops all tables in the test database
func cleanMySQLTestDB(t *testing.T, db *sql.DB) {
	// Disable foreign key checks temporarily
	_, _ = db.Exec("SET FOREIGN_KEY_CHECKS = 0")

	// Drop tables if they exist
	tables := []string{"bookings", "blocked_dates", "experience_requests",
		"reactivation_requests", "dogs", "users", "system_settings", "schema_migrations"}
	for _, table := range tables {
		_, _ = db.Exec("DROP TABLE IF EXISTS " + table)
	}

	// Re-enable foreign key checks
	_, _ = db.Exec("SET FOREIGN_KEY_CHECKS = 1")
}

// cleanPostgreSQLTestDB drops all tables in the test database
func cleanPostgreSQLTestDB(t *testing.T, db *sql.DB) {
	// Drop tables if they exist (CASCADE to handle foreign keys)
	tables := []string{"bookings", "blocked_dates", "experience_requests",
		"reactivation_requests", "dogs", "users", "system_settings", "schema_migrations"}
	for _, table := range tables {
		_, _ = db.Exec("DROP TABLE IF EXISTS " + table + " CASCADE")
	}
}

// DONE: SeedTestUser creates a test user and returns the ID
func SeedTestUser(t *testing.T, db *sql.DB, email, name, level string) int {
	now := time.Now()
	result, err := db.Exec(`
		INSERT INTO users (email, name, phone, password_hash, experience_level, is_verified, is_active, terms_accepted_at, last_activity_at, created_at)
		VALUES (?, ?, ?, ?, ?, 1, 1, ?, ?, ?)
	`, email, name, "+49 123 456789", "test_hash", level, now, now, now)

	if err != nil {
		t.Fatalf("Failed to seed test user: %v", err)
	}

	id, _ := result.LastInsertId()
	return int(id)
}

// DONE: SeedTestDog creates a test dog and returns the ID
func SeedTestDog(t *testing.T, db *sql.DB, name, breed, category string) int {
	now := time.Now()
	result, err := db.Exec(`
		INSERT INTO dogs (name, breed, size, age, category, is_available, created_at)
		VALUES (?, ?, ?, ?, ?, 1, ?)
	`, name, breed, "medium", 5, category, now)

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
		INSERT INTO bookings (user_id, dog_id, date, scheduled_time, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
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
		INSERT INTO blocked_dates (date, reason, created_by, created_at)
		VALUES (?, ?, ?, ?)
	`, date, reason, createdBy, now)

	if err != nil {
		t.Fatalf("Failed to seed test blocked date: %v", err)
	}

	id, _ := result.LastInsertId()
	return int(id)
}

// DONE: SeedTestExperienceRequest creates a test experience request and returns the ID
func SeedTestExperienceRequest(t *testing.T, db *sql.DB, userID int, requestedLevel, status string) int {
	now := time.Now()
	result, err := db.Exec(`
		INSERT INTO experience_requests (user_id, requested_level, status, created_at)
		VALUES (?, ?, ?, ?)
	`, userID, requestedLevel, status, now)

	if err != nil {
		t.Fatalf("Failed to seed test experience request: %v", err)
	}

	id, _ := result.LastInsertId()
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
