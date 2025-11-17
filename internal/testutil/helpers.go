package testutil

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/tranm/gassigeher/internal/database"
)

// DONE: SetupTestDB creates an in-memory SQLite database for testing
func SetupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Run migrations
	err = database.RunMigrations(db)
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
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
func SeedTestBooking(t *testing.T, db *sql.DB, userID, dogID int, date, walkType, scheduledTime, status string) int {
	now := time.Now()
	result, err := db.Exec(`
		INSERT INTO bookings (user_id, dog_id, date, walk_type, scheduled_time, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, userID, dogID, date, walkType, scheduledTime, status, now)

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
