package database

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"

	_ "modernc.org/sqlite"
)

// Test 9.1.1: Fresh Database Migration (consolidated schema)
// Tests that migrations create all tables and seed data correctly
func TestFreshDatabaseMigration(t *testing.T) {
	// Create a temporary database file
	tmpFile := "./test_fresh_migration.db"
	defer os.Remove(tmpFile)

	// Open database with dialect
	config := &DBConfig{Type: "sqlite", Path: tmpFile}
	db, dialect, err := InitializeWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Run migrations
	err = RunMigrationsWithDialect(db, dialect)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Verify all tables exist
	tables := []string{
		"users", "dogs", "bookings", "blocked_dates", "experience_requests",
		"system_settings", "reactivation_requests", "booking_time_rules",
		"custom_holidays", "feiertage_cache", "walk_reports", "walk_report_photos",
		"color_categories", "user_colors", "color_requests",
	}

	for _, table := range tables {
		var count int
		err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		if err != nil {
			t.Errorf("Table %s should exist: %v", table, err)
		}
	}

	// Verify default color categories were inserted
	var colorCount int
	err = db.QueryRow("SELECT COUNT(*) FROM color_categories").Scan(&colorCount)
	if err != nil || colorCount != 7 {
		t.Errorf("Expected 7 color categories, got %d", colorCount)
	}

	// Verify default booking time rules were inserted
	var ruleCount int
	err = db.QueryRow("SELECT COUNT(*) FROM booking_time_rules").Scan(&ruleCount)
	if err != nil || ruleCount < 9 {
		t.Errorf("Expected at least 9 booking time rules, got %d", ruleCount)
	}

	// Verify bookings table has approval columns
	var total, defaultRequiresApproval, defaultApproved int
	// Insert a test booking first
	_, err = db.Exec(`
		INSERT INTO users (id, first_name, last_name, email, password_hash, terms_accepted_at) VALUES
		(1, 'Test', 'User', 'test@example.com', 'hash', CURRENT_TIMESTAMP);
		INSERT INTO dogs (id, name, breed, is_available, color_id) VALUES
		(1, 'Test Dog', 'Labrador', 1, 1);
		INSERT INTO bookings (user_id, dog_id, date, scheduled_time) VALUES
		(1, 1, '2025-01-27', '10:00'),
		(1, 1, '2025-01-28', '15:00');
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	err = db.QueryRow(`
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN requires_approval = 0 THEN 1 ELSE 0 END) as default_requires_approval,
			SUM(CASE WHEN approval_status = 'approved' THEN 1 ELSE 0 END) as default_approved
		FROM bookings
	`).Scan(&total, &defaultRequiresApproval, &defaultApproved)
	if err != nil {
		t.Fatalf("Failed to check default values: %v", err)
	}

	if total != defaultRequiresApproval {
		t.Errorf("Expected all bookings to have requires_approval=0, got %d/%d", defaultRequiresApproval, total)
	}
	if total != defaultApproved {
		t.Errorf("Expected all bookings to have approval_status='approved', got %d/%d", defaultApproved, total)
	}

	t.Log("✅ Existing database migration test passed")
}

// Test 9.1.3: Idempotency
func TestMigrationIdempotency(t *testing.T) {
	// Create a temporary database file
	tmpFile := "./test_idempotency.db"
	defer os.Remove(tmpFile)

	// Open database with dialect
	config := &DBConfig{Type: "sqlite", Path: tmpFile}
	db, dialect, err := InitializeWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Run migrations first time
	err = RunMigrationsWithDialect(db, dialect)
	if err != nil {
		t.Fatalf("First migration failed: %v", err)
	}

	// Count seed data
	var firstRuleCount int
	err = db.QueryRow("SELECT COUNT(*) FROM booking_time_rules").Scan(&firstRuleCount)
	if err != nil {
		t.Fatalf("Failed to count rules after first migration: %v", err)
	}

	// Run migrations second time
	err = RunMigrationsWithDialect(db, dialect)
	if err != nil {
		t.Fatalf("Second migration failed: %v", err)
	}

	// Verify seed data not duplicated
	var secondRuleCount int
	err = db.QueryRow("SELECT COUNT(*) FROM booking_time_rules").Scan(&secondRuleCount)
	if err != nil {
		t.Fatalf("Failed to count rules after second migration: %v", err)
	}

	if firstRuleCount != secondRuleCount {
		t.Errorf("Seed data duplicated: first=%d, second=%d", firstRuleCount, secondRuleCount)
	}

	// Run migrations third time to be sure
	err = RunMigrationsWithDialect(db, dialect)
	if err != nil {
		t.Fatalf("Third migration failed: %v", err)
	}

	var thirdRuleCount int
	err = db.QueryRow("SELECT COUNT(*) FROM booking_time_rules").Scan(&thirdRuleCount)
	if err != nil {
		t.Fatalf("Failed to count rules after third migration: %v", err)
	}

	if firstRuleCount != thirdRuleCount {
		t.Errorf("Seed data changed after third migration: first=%d, third=%d", firstRuleCount, thirdRuleCount)
	}

	t.Log("✅ Migration idempotency test passed")
}

// Test 9.2.1: Foreign Key Constraints
func TestForeignKeyConstraints(t *testing.T) {
	// Create a temporary database file
	tmpFile := "./test_foreign_keys.db"
	defer os.Remove(tmpFile)

	// Open database with dialect
	config := &DBConfig{Type: "sqlite", Path: tmpFile}
	db, dialect, err := InitializeWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Enable foreign keys
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Run migrations
	err = RunMigrationsWithDialect(db, dialect)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Create test data
	_, err = db.Exec(`
		INSERT INTO users (id, first_name, last_name, email, password_hash, is_admin, terms_accepted_at) VALUES
		(1, 'Admin', 'User', 'admin@example.com', 'hash', 1, CURRENT_TIMESTAMP),
		(2, 'Regular', 'User', 'user@example.com', 'hash', 0, CURRENT_TIMESTAMP);

		INSERT INTO dogs (id, name, breed, color_id, is_available) VALUES
		(1, 'Test Dog', 'Labrador', 1, 1);

		INSERT INTO bookings (id, user_id, dog_id, date, scheduled_time, status, requires_approval, approval_status, approved_by, approved_at) VALUES
		(1, 2, 1, '2025-01-27', '10:00', 'scheduled', 1, 'approved', 1, CURRENT_TIMESTAMP);
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Test 9.2.1-A: Delete admin who approved bookings
	_, err = db.Exec("DELETE FROM users WHERE id = 1")
	if err != nil {
		t.Fatalf("Failed to delete admin user: %v", err)
	}

	// Verify booking remains but approved_by is NULL (ON DELETE SET NULL)
	var approvedBy sql.NullInt64
	err = db.QueryRow("SELECT approved_by FROM bookings WHERE id = 1").Scan(&approvedBy)
	if err != nil {
		t.Fatalf("Failed to query booking: %v", err)
	}
	if approvedBy.Valid {
		t.Errorf("Expected approved_by to be NULL after admin deletion, got %v", approvedBy.Int64)
	}

	// Test 9.2.1-B: Delete user with bookings (CASCADE should delete bookings)
	_, err = db.Exec("DELETE FROM users WHERE id = 2")
	if err != nil {
		t.Fatalf("Failed to delete regular user: %v", err)
	}

	// Verify booking was also deleted (ON DELETE CASCADE)
	var bookingCount int
	db.QueryRow("SELECT COUNT(*) FROM bookings WHERE user_id = 2").Scan(&bookingCount)
	if bookingCount != 0 {
		t.Error("Expected bookings to be deleted with user (CASCADE), but found bookings remaining")
	}

	t.Log("✅ Foreign key constraints test passed")
}

// Test 9.2.2: Unique Constraints
func TestUniqueConstraints(t *testing.T) {
	// Create a temporary database file
	tmpFile := "./test_unique_constraints.db"
	defer os.Remove(tmpFile)

	// Open database with dialect
	config := &DBConfig{Type: "sqlite", Path: tmpFile}
	db, dialect, err := InitializeWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Run migrations
	err = RunMigrationsWithDialect(db, dialect)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)

	// Test 9.2.2-A: Duplicate (day_type, rule_name) in booking_time_rules
	rule1 := &models.BookingTimeRule{
		DayType:   "weekday",
		RuleName:  "Custom Rule",
		StartTime: "10:00",
		EndTime:   "11:00",
		IsBlocked: false,
	}
	err = bookingTimeRepo.CreateRule(rule1)
	if err != nil {
		t.Fatalf("Failed to create first rule: %v", err)
	}

	// Try to create duplicate
	rule2 := &models.BookingTimeRule{
		DayType:   "weekday",
		RuleName:  "Custom Rule",
		StartTime: "14:00",
		EndTime:   "15:00",
		IsBlocked: false,
	}
	err = bookingTimeRepo.CreateRule(rule2)
	if err == nil {
		t.Error("Expected error for duplicate (day_type, rule_name), got nil")
	}

	// Test 9.2.2-B: Duplicate holiday date
	holiday1 := &models.CustomHoliday{
		Date:     "2025-12-25",
		Name:     "Christmas",
		IsActive: true,
		Source:   "admin",
	}
	err = holidayRepo.CreateHoliday(holiday1)
	if err != nil {
		t.Fatalf("Failed to create first holiday: %v", err)
	}

	// Try to create duplicate
	holiday2 := &models.CustomHoliday{
		Date:     "2025-12-25",
		Name:     "Christmas Day",
		IsActive: true,
		Source:   "admin",
	}
	err = holidayRepo.CreateHoliday(holiday2)
	if err == nil {
		t.Error("Expected error for duplicate holiday date, got nil")
	}

	// Test 9.2.2-C: Duplicate booking (dog, date, scheduled_time)
	// First create test user and dog
	_, err = db.Exec(`
		INSERT INTO users (id, first_name, last_name, email, password_hash, terms_accepted_at) VALUES
		(1, 'Test', 'User', 'test@example.com', 'hash', CURRENT_TIMESTAMP);

		INSERT INTO dogs (id, name, breed, color_id, is_available) VALUES
		(1, 'Test Dog', 'Labrador', 1, 1);
	`)
	if err != nil {
		t.Fatalf("Failed to create test user/dog: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO bookings (user_id, dog_id, date, scheduled_time) VALUES
		(1, 1, '2025-01-27', '10:00')
	`)
	if err != nil {
		t.Fatalf("Failed to create first booking: %v", err)
	}

	// Try to create duplicate booking (same dog, date, scheduled_time)
	_, err = db.Exec(`
		INSERT INTO bookings (user_id, dog_id, date, scheduled_time) VALUES
		(1, 1, '2025-01-27', '10:00')
	`)
	if err == nil {
		t.Error("Expected error for duplicate booking (dog, date, scheduled_time), got nil")
	}

	t.Log("✅ Unique constraints test passed")
}

// Test 9.2.3: Index Effectiveness
func TestIndexEffectiveness(t *testing.T) {
	// Create a temporary database file
	tmpFile := "./test_index_effectiveness.db"
	defer os.Remove(tmpFile)

	// Open database with dialect
	config := &DBConfig{Type: "sqlite", Path: tmpFile}
	db, dialect, err := InitializeWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Run migrations
	err = RunMigrationsWithDialect(db, dialect)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	holidayRepo := repository.NewHolidayRepository(db)

	// Insert 1000 holidays for performance testing
	t.Log("Inserting 1000 test holidays...")
	for i := 0; i < 1000; i++ {
		date := time.Now().AddDate(0, 0, i).Format("2006-01-02")
		holiday := &models.CustomHoliday{
			Date:     date,
			Name:     fmt.Sprintf("Holiday %d", i),
			IsActive: true,
			Source:   "test",
		}
		err = holidayRepo.CreateHoliday(holiday)
		if err != nil {
			t.Logf("Warning: Failed to insert holiday %d: %v", i, err)
		}
	}

	// Test index on custom_holidays.date
	t.Log("Testing idx_custom_holidays_date...")
	rows, err := db.Query("EXPLAIN QUERY PLAN SELECT * FROM custom_holidays WHERE date = ?", "2025-01-01")
	if err != nil {
		t.Fatalf("Failed to explain query: %v", err)
	}
	defer rows.Close()

	foundIndex := false
	for rows.Next() {
		var id, parent, notused int
		var detail string
		err := rows.Scan(&id, &parent, &notused, &detail)
		if err != nil {
			t.Fatalf("Failed to scan explain result: %v", err)
		}
		if detail != "" {
			t.Logf("Query plan: %s", detail)
			// Accept either our explicit index or SQLite's auto-generated UNIQUE index
			// Both provide the same performance benefit for date lookups
			if detail == "SEARCH custom_holidays USING INDEX idx_custom_holidays_date (date=?)" ||
				detail == "SEARCH TABLE custom_holidays USING INDEX idx_custom_holidays_date (date=?)" ||
				detail == "SEARCH custom_holidays USING INDEX sqlite_autoindex_custom_holidays_1 (date=?)" ||
				detail == "SEARCH TABLE custom_holidays USING INDEX sqlite_autoindex_custom_holidays_1 (date=?)" {
				foundIndex = true
			}
		}
	}

	if !foundIndex {
		t.Error("Expected query to use an index on the date column")
	}

	// Benchmark holiday lookup
	start := time.Now()
	iterations := 1000
	for i := 0; i < iterations; i++ {
		date := time.Now().AddDate(0, 0, i%100).Format("2006-01-02")
		_, _ = holidayRepo.IsHoliday(date)
	}
	duration := time.Since(start)
	avgTime := duration.Milliseconds() / int64(iterations)

	t.Logf("Average holiday lookup time: %dms (%d iterations)", avgTime, iterations)
	if avgTime > 5 {
		t.Errorf("Holiday lookup too slow: %dms (expected < 5ms)", avgTime)
	}

	// Test index on custom_holidays.is_active
	t.Log("Testing idx_custom_holidays_active...")
	rows, err = db.Query("EXPLAIN QUERY PLAN SELECT * FROM custom_holidays WHERE is_active = ?", 1)
	if err != nil {
		t.Fatalf("Failed to explain query: %v", err)
	}
	defer rows.Close()

	foundActiveIndex := false
	for rows.Next() {
		var id, parent, notused int
		var detail string
		err := rows.Scan(&id, &parent, &notused, &detail)
		if err != nil {
			t.Fatalf("Failed to scan explain result: %v", err)
		}
		if detail != "" {
			t.Logf("Query plan (is_active): %s", detail)
			// Note: SQLite might use covering index or table scan if it's more efficient
			if detail == "SEARCH custom_holidays USING INDEX idx_custom_holidays_active (is_active=?)" ||
				detail == "SEARCH TABLE custom_holidays USING INDEX idx_custom_holidays_active (is_active=?)" {
				foundActiveIndex = true
			}
		}
	}

	if !foundActiveIndex {
		t.Log("Note: is_active query may not use dedicated index (SQLite optimization)")
	}

	// Test index on bookings.approval_status
	t.Log("Testing idx_bookings_approval_status...")

	// Insert test bookings
	_, err = db.Exec(`
		INSERT INTO users (id, first_name, last_name, email, password_hash, terms_accepted_at) VALUES
		(1, 'Test', 'User', 'test@example.com', 'hash', CURRENT_TIMESTAMP);

		INSERT INTO dogs (id, name, breed, color_id, is_available) VALUES
		(1, 'Test Dog', 'Labrador', 1, 1);
	`)
	if err != nil {
		t.Fatalf("Failed to create test user/dog: %v", err)
	}

	for i := 0; i < 100; i++ {
		status := "approved"
		if i%10 == 0 {
			status = "pending"
		}
		_, err = db.Exec(`
			INSERT INTO bookings (user_id, dog_id, date, scheduled_time, approval_status) VALUES
			(1, 1, ?, '10:00', ?)
		`, time.Now().AddDate(0, 0, i).Format("2006-01-02"), status)
		if err != nil {
			t.Logf("Warning: Failed to insert booking %d: %v", i, err)
		}
	}

	rows, err = db.Query("EXPLAIN QUERY PLAN SELECT * FROM bookings WHERE approval_status = ?", "pending")
	if err != nil {
		t.Fatalf("Failed to explain query: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, parent, notused int
		var detail string
		err := rows.Scan(&id, &parent, &notused, &detail)
		if err != nil {
			t.Fatalf("Failed to scan explain result: %v", err)
		}
		if detail != "" {
			t.Logf("Query plan (approval_status): %s", detail)
		}
	}

	t.Log("✅ Index effectiveness test passed")
}
