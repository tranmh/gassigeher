package repository

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
	_ "modernc.org/sqlite"
)

// setupTestDBForHolidays creates a test database with custom_holidays and feiertage_cache tables
func setupTestDBForHolidays(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create custom_holidays table
	createHolidaysSQL := `
	CREATE TABLE IF NOT EXISTS custom_holidays (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		date TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		is_active INTEGER DEFAULT 1,
		source TEXT DEFAULT 'admin',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		created_by INTEGER,
		FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
	);
	`

	// Create feiertage_cache table
	createCacheSQL := `
	CREATE TABLE IF NOT EXISTS feiertage_cache (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		year INTEGER NOT NULL,
		state TEXT NOT NULL,
		data TEXT NOT NULL,
		fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL,
		UNIQUE(year, state)
	);
	`

	_, err = db.Exec(createHolidaysSQL)
	if err != nil {
		t.Fatalf("Failed to create custom_holidays table: %v", err)
	}

	_, err = db.Exec(createCacheSQL)
	if err != nil {
		t.Fatalf("Failed to create feiertage_cache table: %v", err)
	}

	return db
}

// seedHolidays seeds the database with test holiday data
func seedHolidays(t *testing.T, db *sql.DB) {
	holidays := []struct {
		date     string
		name     string
		isActive int
		source   string
	}{
		{"2025-01-01", "Neujahrstag", 1, "api"},
		{"2025-01-06", "Heilige Drei Könige", 1, "api"},
		{"2025-04-18", "Karfreitag", 1, "api"},
		{"2025-04-21", "Ostermontag", 1, "api"},
		{"2025-05-01", "Tag der Arbeit", 1, "api"},
		{"2025-12-25", "Weihnachten", 1, "api"},
		{"2025-12-26", "Zweiter Weihnachtstag", 1, "api"},
		{"2025-02-14", "Valentine's Day", 0, "admin"}, // Inactive
	}

	for _, h := range holidays {
		_, err := db.Exec(`
			INSERT INTO custom_holidays (date, name, is_active, source)
			VALUES (?, ?, ?, ?)
		`, h.date, h.name, h.isActive, h.source)
		if err != nil {
			t.Fatalf("Failed to seed holiday %s: %v", h.name, err)
		}
	}
}

// Test 2.2.1: IsHoliday - Lookup Performance
func TestIsHoliday_KnownHoliday(t *testing.T) {
	db := setupTestDBForHolidays(t)
	defer db.Close()

	seedHolidays(t, db)
	repo := NewHolidayRepository(db)

	testCases := []struct {
		date     string
		expected bool
	}{
		{"2025-01-01", true},  // Neujahrstag (active)
		{"2025-01-06", true},  // Heilige Drei Könige (active)
		{"2025-12-25", true},  // Weihnachten (active)
		{"2025-01-15", false}, // Not a holiday
		{"2025-02-14", false}, // Valentine's Day (inactive)
	}

	for _, tc := range testCases {
		t.Run(tc.date, func(t *testing.T) {
			isHoliday, err := repo.IsHoliday(tc.date)
			if err != nil {
				t.Fatalf("IsHoliday failed: %v", err)
			}

			if isHoliday != tc.expected {
				t.Errorf("IsHoliday(%s) = %v, want %v", tc.date, isHoliday, tc.expected)
			}
		})
	}
}

func TestIsHoliday_Performance(t *testing.T) {
	db := setupTestDBForHolidays(t)
	defer db.Close()

	// Insert 100 holidays
	for i := 0; i < 100; i++ {
		date := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i).Format("2006-01-02")
		_, err := db.Exec(`
			INSERT INTO custom_holidays (date, name, is_active, source)
			VALUES (?, ?, 1, 'test')
		`, date, fmt.Sprintf("Holiday %d", i))
		if err != nil {
			t.Fatalf("Failed to insert holiday: %v", err)
		}
	}

	repo := NewHolidayRepository(db)

	// Benchmark lookup
	start := time.Now()
	for i := 0; i < 1000; i++ {
		_, err := repo.IsHoliday("2025-06-15")
		if err != nil {
			t.Fatalf("IsHoliday failed: %v", err)
		}
	}
	elapsed := time.Since(start)

	avgTime := elapsed / 1000
	if avgTime > 10*time.Millisecond {
		t.Errorf("Average lookup time %v exceeds 10ms threshold", avgTime)
	}
}

// Test 2.2.2: CreateHoliday - Duplicate Handling
func TestCreateHoliday_UniqueDate(t *testing.T) {
	db := setupTestDBForHolidays(t)
	defer db.Close()

	repo := NewHolidayRepository(db)

	holiday := &models.CustomHoliday{
		Date:     "2025-07-01",
		Name:     "Custom Holiday",
		IsActive: true,
		Source:   "admin",
	}

	err := repo.CreateHoliday(holiday)
	if err != nil {
		t.Fatalf("CreateHoliday failed: %v", err)
	}

	// Verify ID assigned
	if holiday.ID == 0 {
		t.Error("Expected ID to be assigned, got 0")
	}

	// Verify holiday created
	isHoliday, err := repo.IsHoliday("2025-07-01")
	if err != nil {
		t.Fatalf("IsHoliday failed: %v", err)
	}

	if !isHoliday {
		t.Error("Expected holiday to exist")
	}
}

func TestCreateHoliday_DuplicateDate(t *testing.T) {
	db := setupTestDBForHolidays(t)
	defer db.Close()

	seedHolidays(t, db)
	repo := NewHolidayRepository(db)

	// Try to create duplicate date
	holiday := &models.CustomHoliday{
		Date:     "2025-01-01", // Already exists
		Name:     "Duplicate Holiday",
		IsActive: true,
		Source:   "admin",
	}

	err := repo.CreateHoliday(holiday)
	if err == nil {
		t.Error("Expected error for duplicate date, got nil")
	}
}

// Test 2.2.3: GetCachedHolidays - Expiration Check
func TestGetCachedHolidays_ValidCache(t *testing.T) {
	db := setupTestDBForHolidays(t)
	defer db.Close()

	repo := NewHolidayRepository(db)

	// Insert valid cache entry
	expiresAt := time.Now().AddDate(0, 0, 7) // 7 days from now
	_, err := db.Exec(`
		INSERT INTO feiertage_cache (year, state, data, fetched_at, expires_at)
		VALUES (?, ?, ?, ?, ?)
	`, 2025, "BW", `{"holidays":[]}`, time.Now(), expiresAt)
	if err != nil {
		t.Fatalf("Failed to insert cache: %v", err)
	}

	// Retrieve cache
	cached, err := repo.GetCachedHolidays(2025, "BW")
	if err != nil {
		t.Fatalf("GetCachedHolidays failed: %v", err)
	}

	if cached == "" {
		t.Error("Expected cached data, got empty string")
	}

	if cached != `{"holidays":[]}` {
		t.Errorf("Expected cached data to match, got: %s", cached)
	}
}

func TestGetCachedHolidays_ExpiredCache(t *testing.T) {
	db := setupTestDBForHolidays(t)
	defer db.Close()

	repo := NewHolidayRepository(db)

	// Insert expired cache entry
	expiresAt := time.Now().AddDate(0, 0, -1) // 1 day ago
	_, err := db.Exec(`
		INSERT INTO feiertage_cache (year, state, data, fetched_at, expires_at)
		VALUES (?, ?, ?, ?, ?)
	`, 2025, "BW", `{"holidays":[]}`, time.Now(), expiresAt)
	if err != nil {
		t.Fatalf("Failed to insert cache: %v", err)
	}

	// Retrieve cache (should return empty for expired)
	cached, err := repo.GetCachedHolidays(2025, "BW")
	if err != nil {
		t.Fatalf("GetCachedHolidays failed: %v", err)
	}

	if cached != "" {
		t.Errorf("Expected empty string for expired cache, got: %s", cached)
	}
}

func TestGetCachedHolidays_NoCache(t *testing.T) {
	db := setupTestDBForHolidays(t)
	defer db.Close()

	repo := NewHolidayRepository(db)

	// Retrieve non-existent cache
	cached, err := repo.GetCachedHolidays(2025, "BW")
	if err != nil {
		t.Fatalf("GetCachedHolidays failed: %v", err)
	}

	if cached != "" {
		t.Errorf("Expected empty string for no cache, got: %s", cached)
	}
}

// Test SetCachedHolidays
func TestSetCachedHolidays_NewCache(t *testing.T) {
	db := setupTestDBForHolidays(t)
	defer db.Close()

	repo := NewHolidayRepository(db)

	err := repo.SetCachedHolidays(2025, "BW", `{"holidays":[]}`, 7)
	if err != nil {
		t.Fatalf("SetCachedHolidays failed: %v", err)
	}

	// Verify cache created
	cached, err := repo.GetCachedHolidays(2025, "BW")
	if err != nil {
		t.Fatalf("GetCachedHolidays failed: %v", err)
	}

	if cached == "" {
		t.Error("Expected cached data, got empty string")
	}
}

func TestSetCachedHolidays_ReplaceExisting(t *testing.T) {
	db := setupTestDBForHolidays(t)
	defer db.Close()

	repo := NewHolidayRepository(db)

	// Set initial cache
	err := repo.SetCachedHolidays(2025, "BW", `{"old":true}`, 7)
	if err != nil {
		t.Fatalf("SetCachedHolidays failed: %v", err)
	}

	// Replace cache
	err = repo.SetCachedHolidays(2025, "BW", `{"new":true}`, 7)
	if err != nil {
		t.Fatalf("SetCachedHolidays failed: %v", err)
	}

	// Verify new cache
	cached, err := repo.GetCachedHolidays(2025, "BW")
	if err != nil {
		t.Fatalf("GetCachedHolidays failed: %v", err)
	}

	if cached != `{"new":true}` {
		t.Errorf("Expected new cache data, got: %s", cached)
	}
}

// Test GetHolidaysByYear
func TestGetHolidaysByYear_2025(t *testing.T) {
	db := setupTestDBForHolidays(t)
	defer db.Close()

	seedHolidays(t, db)
	repo := NewHolidayRepository(db)

	holidays, err := repo.GetHolidaysByYear(2025)
	if err != nil {
		t.Fatalf("GetHolidaysByYear failed: %v", err)
	}

	// Should return 7 active holidays (Valentine's is inactive)
	expectedCount := 7
	if len(holidays) != expectedCount {
		t.Errorf("Expected %d holidays, got %d", expectedCount, len(holidays))
	}

	// Verify all are from 2025
	for _, h := range holidays {
		if h.Date[:4] != "2025" {
			t.Errorf("Expected year 2025, got date: %s", h.Date)
		}
	}

	// Verify all are active
	for _, h := range holidays {
		if !h.IsActive {
			t.Errorf("Expected all holidays to be active, got inactive: %s", h.Name)
		}
	}

	// Verify ordered by date
	for i := 1; i < len(holidays); i++ {
		if holidays[i-1].Date > holidays[i].Date {
			t.Errorf("Holidays not ordered by date: %s > %s", holidays[i-1].Date, holidays[i].Date)
		}
	}
}

func TestGetHolidaysByYear_2026_Empty(t *testing.T) {
	db := setupTestDBForHolidays(t)
	defer db.Close()

	seedHolidays(t, db)
	repo := NewHolidayRepository(db)

	holidays, err := repo.GetHolidaysByYear(2026)
	if err != nil {
		t.Fatalf("GetHolidaysByYear failed: %v", err)
	}

	if len(holidays) != 0 {
		t.Errorf("Expected 0 holidays for 2026, got %d", len(holidays))
	}
}

// Test UpdateHoliday
func TestUpdateHoliday_ToggleActive(t *testing.T) {
	db := setupTestDBForHolidays(t)
	defer db.Close()

	seedHolidays(t, db)
	repo := NewHolidayRepository(db)

	// Get first holiday
	holidays, _ := repo.GetHolidaysByYear(2025)
	if len(holidays) == 0 {
		t.Fatal("No holidays found")
	}

	originalHoliday := holidays[0]

	// Toggle is_active
	updatedHoliday := &models.CustomHoliday{
		Name:     originalHoliday.Name,
		IsActive: false, // Toggle to false
	}

	err := repo.UpdateHoliday(originalHoliday.ID, updatedHoliday)
	if err != nil {
		t.Fatalf("UpdateHoliday failed: %v", err)
	}

	// Verify update
	isHoliday, _ := repo.IsHoliday(originalHoliday.Date)
	if isHoliday {
		t.Error("Expected holiday to be inactive (not found by IsHoliday)")
	}
}

func TestUpdateHoliday_ChangeName(t *testing.T) {
	db := setupTestDBForHolidays(t)
	defer db.Close()

	seedHolidays(t, db)
	repo := NewHolidayRepository(db)

	// Get first holiday
	holidays, _ := repo.GetHolidaysByYear(2025)
	if len(holidays) == 0 {
		t.Fatal("No holidays found")
	}

	originalHoliday := holidays[0]

	// Change name
	updatedHoliday := &models.CustomHoliday{
		Name:     "Updated Name",
		IsActive: originalHoliday.IsActive,
	}

	err := repo.UpdateHoliday(originalHoliday.ID, updatedHoliday)
	if err != nil {
		t.Fatalf("UpdateHoliday failed: %v", err)
	}

	// Verify update
	holidays, _ = repo.GetHolidaysByYear(2025)
	found := false
	for _, h := range holidays {
		if h.ID == originalHoliday.ID {
			found = true
			if h.Name != "Updated Name" {
				t.Errorf("Expected name 'Updated Name', got '%s'", h.Name)
			}
		}
	}

	if !found {
		t.Error("Updated holiday not found")
	}
}

// Test DeleteHoliday
func TestDeleteHoliday_ExistingHoliday(t *testing.T) {
	db := setupTestDBForHolidays(t)
	defer db.Close()

	seedHolidays(t, db)
	repo := NewHolidayRepository(db)

	// Get count before delete
	holidaysBefore, _ := repo.GetHolidaysByYear(2025)
	countBefore := len(holidaysBefore)

	if countBefore == 0 {
		t.Fatal("No holidays to delete")
	}

	// Delete first holiday
	err := repo.DeleteHoliday(holidaysBefore[0].ID)
	if err != nil {
		t.Fatalf("DeleteHoliday failed: %v", err)
	}

	// Verify deletion
	holidaysAfter, _ := repo.GetHolidaysByYear(2025)
	countAfter := len(holidaysAfter)

	if countAfter != countBefore-1 {
		t.Errorf("Expected %d holidays after delete, got %d", countBefore-1, countAfter)
	}

	// Verify holiday no longer exists
	isHoliday, _ := repo.IsHoliday(holidaysBefore[0].Date)
	if isHoliday {
		t.Error("Deleted holiday still exists")
	}
}

func TestDeleteHoliday_NonExistentID(t *testing.T) {
	db := setupTestDBForHolidays(t)
	defer db.Close()

	seedHolidays(t, db)
	repo := NewHolidayRepository(db)

	// Delete non-existent holiday
	err := repo.DeleteHoliday(9999)
	if err != nil {
		t.Fatalf("DeleteHoliday with non-existent ID should not error, got: %v", err)
	}

	// Verify count unchanged
	holidays, _ := repo.GetHolidaysByYear(2025)
	if len(holidays) != 7 {
		t.Errorf("Expected 7 holidays (unchanged), got %d", len(holidays))
	}
}

// Test cache expiration timing
func TestCacheExpiration_7Days(t *testing.T) {
	db := setupTestDBForHolidays(t)
	defer db.Close()

	repo := NewHolidayRepository(db)

	// Set cache with 7 days expiration
	err := repo.SetCachedHolidays(2025, "BW", `{"test":true}`, 7)
	if err != nil {
		t.Fatalf("SetCachedHolidays failed: %v", err)
	}

	// Verify expires_at is set correctly
	var expiresAt time.Time
	err = db.QueryRow(`SELECT expires_at FROM feiertage_cache WHERE year = ? AND state = ?`, 2025, "BW").Scan(&expiresAt)
	if err != nil {
		t.Fatalf("Failed to query expires_at: %v", err)
	}

	expectedExpiry := time.Now().AddDate(0, 0, 7)
	diff := expiresAt.Sub(expectedExpiry)

	// Allow 1 second tolerance
	if diff < 0 {
		diff = -diff
	}
	if diff > time.Second {
		t.Errorf("Expiry time not set correctly. Expected ~%v, got %v (diff: %v)", expectedExpiry, expiresAt, diff)
	}
}

// ========================================
// Phase 7: Performance Testing
// ========================================

// Test 7.1.1: Holiday Lookup Performance Benchmark
// Purpose: Verify holiday check is fast with index
func BenchmarkIsHoliday(b *testing.B) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		b.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create tables with indexes
	_, _ = db.Exec(`
		CREATE TABLE IF NOT EXISTS custom_holidays (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			is_active INTEGER DEFAULT 1,
			source TEXT DEFAULT 'admin',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			created_by INTEGER
		);
		CREATE INDEX IF NOT EXISTS idx_custom_holidays_date ON custom_holidays(date);
		CREATE INDEX IF NOT EXISTS idx_custom_holidays_active ON custom_holidays(is_active);
	`)

	// Insert 1000 holidays
	for i := 0; i < 1000; i++ {
		date := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i).Format("2006-01-02")
		_, _ = db.Exec(`INSERT INTO custom_holidays (date, name, is_active, source) VALUES (?, ?, 1, 'test')`,
			date, fmt.Sprintf("Holiday %d", i))
	}

	repo := NewHolidayRepository(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = repo.IsHoliday("2025-06-15")
	}
}

// Test 7.1.1: Holiday Lookup Performance with Large Dataset
func BenchmarkIsHoliday_LargeDataset(b *testing.B) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		b.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create tables with indexes
	_, _ = db.Exec(`
		CREATE TABLE IF NOT EXISTS custom_holidays (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			is_active INTEGER DEFAULT 1,
			source TEXT DEFAULT 'admin',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			created_by INTEGER
		);
		CREATE INDEX IF NOT EXISTS idx_custom_holidays_date ON custom_holidays(date);
		CREATE INDEX IF NOT EXISTS idx_custom_holidays_active ON custom_holidays(is_active);
	`)

	// Insert 10,000 holidays for stress test
	for i := 0; i < 10000; i++ {
		date := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i).Format("2006-01-02")
		_, _ = db.Exec(`INSERT INTO custom_holidays (date, name, is_active, source) VALUES (?, ?, 1, 'test')`,
			date, fmt.Sprintf("Holiday %d", i))
	}

	repo := NewHolidayRepository(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = repo.IsHoliday("2030-06-15")
	}
}

// Test 7.1.1: Year Holiday List Performance
func BenchmarkGetHolidaysByYear(b *testing.B) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		b.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create tables
	_, _ = db.Exec(`
		CREATE TABLE IF NOT EXISTS custom_holidays (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			is_active INTEGER DEFAULT 1,
			source TEXT DEFAULT 'admin',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			created_by INTEGER
		);
		CREATE INDEX IF NOT EXISTS idx_custom_holidays_date ON custom_holidays(date);
		CREATE INDEX IF NOT EXISTS idx_custom_holidays_active ON custom_holidays(is_active);
	`)

	// Insert multiple years of holidays
	for year := 2025; year <= 2030; year++ {
		for i := 0; i < 20; i++ {
			date := time.Date(year, time.Month(i%12+1), (i%28)+1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
			_, _ = db.Exec(`INSERT INTO custom_holidays (date, name, is_active, source) VALUES (?, ?, 1, 'test')`,
				date, fmt.Sprintf("Holiday %d", i))
		}
	}

	repo := NewHolidayRepository(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = repo.GetHolidaysByYear(2027)
	}
}

// Test 7.4.1: Cache Hit Rate Test
func TestCacheEffectiveness(t *testing.T) {
	db := setupTestDBForHolidays(t)
	defer db.Close()

	repo := NewHolidayRepository(db)

	// Set cache
	err := repo.SetCachedHolidays(2025, "BW", `{"holidays":[]}`, 7)
	if err != nil {
		t.Fatalf("SetCachedHolidays failed: %v", err)
	}

	// Make 100 requests
	cacheHits := 0
	cacheMisses := 0

	for i := 0; i < 100; i++ {
		cached, err := repo.GetCachedHolidays(2025, "BW")
		if err != nil {
			t.Fatalf("GetCachedHolidays failed: %v", err)
		}

		if cached != "" {
			cacheHits++
		} else {
			cacheMisses++
		}
	}

	// Verify cache effectiveness (should be 100% hits)
	expectedHits := 100
	if cacheHits != expectedHits {
		t.Errorf("Expected %d cache hits, got %d (misses: %d)", expectedHits, cacheHits, cacheMisses)
	}

	// Calculate hit rate
	hitRate := float64(cacheHits) / float64(cacheHits+cacheMisses) * 100
	if hitRate < 99.0 {
		t.Errorf("Cache hit rate %.2f%% is below 99%% threshold", hitRate)
	}

	t.Logf("Cache hit rate: %.2f%% (%d hits, %d misses)", hitRate, cacheHits, cacheMisses)
}
