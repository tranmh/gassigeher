package services

import (
	"testing"
	"time"

	"github.com/tranm/gassigeher/internal/models"
	"github.com/tranm/gassigeher/internal/repository"
	"github.com/tranm/gassigeher/internal/testutil"
)

// Test 1.2.1: IsHoliday - Known Holidays
func TestIsHoliday_KnownHolidays(t *testing.T) {
	db := testutil.SetupTestDB(t)

	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	service := NewHolidayService(holidayRepo, settingsRepo)

	// Seed holidays table with test data
	holidays := []models.CustomHoliday{
		{Date: "2025-01-01", Name: "Neujahrstag", IsActive: true, Source: "test"},
		{Date: "2025-01-06", Name: "Heilige Drei Könige", IsActive: true, Source: "test"},
		{Date: "2025-12-25", Name: "Weihnachten", IsActive: true, Source: "test"},
		{Date: "2025-02-14", Name: "Valentine's Day", IsActive: false, Source: "test"}, // Inactive
	}

	for _, h := range holidays {
		holiday := h
		err := holidayRepo.CreateHoliday(&holiday)
		if err != nil {
			t.Fatalf("Failed to seed holiday: %v", err)
		}
	}

	testCases := []struct {
		name     string
		date     string
		expected bool
	}{
		{"Neujahrstag", "2025-01-01", true},
		{"Heilige Drei Könige", "2025-01-06", true},
		{"Weihnachten", "2025-12-25", true},
		{"Non-holiday", "2025-01-15", false},
		{"Valentine's (inactive)", "2025-02-14", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := service.IsHoliday(tc.date)
			if err != nil {
				t.Fatalf("IsHoliday() error = %v", err)
			}
			if result != tc.expected {
				t.Errorf("IsHoliday(%s) = %v, expected %v", tc.date, result, tc.expected)
			}
		})
	}
}

// Test 1.2.2: FetchAndCacheHolidays - API Integration
func TestFetchAndCacheHolidays_APIIntegration(t *testing.T) {
	db := testutil.SetupTestDB(t)

	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	service := NewHolidayService(holidayRepo, settingsRepo)

	// Set state to BW (should already exist from migration)
	_ = settingsRepo.Update("feiertage_state", "BW")

	// First fetch - should call API
	err := service.FetchAndCacheHolidays(2025)
	if err != nil {
		// If API is not reachable, skip this test
		t.Skipf("API not reachable (this is OK for offline testing): %v", err)
		return
	}

	// Verify cache created
	cached, err := holidayRepo.GetCachedHolidays(2025, "BW")
	if err != nil {
		t.Fatalf("Cache not created: %v", err)
	}
	if cached == "" {
		t.Error("Expected cached data, got empty string")
	}

	// Verify holidays inserted
	holidays, err := holidayRepo.GetHolidaysByYear(2025)
	if err != nil {
		t.Fatalf("Failed to get holidays: %v", err)
	}
	if len(holidays) < 10 { // BW has ~12 holidays
		t.Errorf("Expected at least 10 holidays, got %d", len(holidays))
	}

	// Second fetch - should use cache (no API call)
	// We can't easily verify no API call without mocking, but we can verify it succeeds
	err = service.FetchAndCacheHolidays(2025)
	if err != nil {
		t.Errorf("Cache fetch failed: %v", err)
	}

	// Verify holidays still present
	holidays2, err := holidayRepo.GetHolidaysByYear(2025)
	if err != nil {
		t.Fatalf("Failed to get holidays after cache fetch: %v", err)
	}
	if len(holidays2) < len(holidays) {
		t.Errorf("Expected at least %d holidays after cache fetch, got %d", len(holidays), len(holidays2))
	}
}

// Test 1.2.3: GetHolidaysForYear - Filtering
func TestGetHolidaysForYear_Filtering(t *testing.T) {
	db := testutil.SetupTestDB(t)

	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	service := NewHolidayService(holidayRepo, settingsRepo)

	// Disable API usage for this test
	_ = settingsRepo.Update("use_feiertage_api", "false")

	// Seed holidays for different years
	holidays2025 := []models.CustomHoliday{
		{Date: "2025-01-01", Name: "Neujahrstag 2025", IsActive: true, Source: "test"},
		{Date: "2025-12-25", Name: "Weihnachten 2025", IsActive: true, Source: "test"},
	}

	holidays2026 := []models.CustomHoliday{
		{Date: "2026-01-01", Name: "Neujahrstag 2026", IsActive: true, Source: "test"},
		{Date: "2026-12-25", Name: "Weihnachten 2026", IsActive: true, Source: "test"},
	}

	for _, h := range holidays2025 {
		holiday := h
		_ = holidayRepo.CreateHoliday(&holiday)
	}

	for _, h := range holidays2026 {
		holiday := h
		_ = holidayRepo.CreateHoliday(&holiday)
	}

	testCases := []struct {
		year          int
		expectedCount int
		note          string
	}{
		{2025, 2, "2025 holidays"},
		{2026, 2, "2026 holidays"},
		{2024, 0, "No data for 2024"},
	}

	for _, tc := range testCases {
		t.Run(tc.note, func(t *testing.T) {
			holidays, err := service.GetHolidaysForYear(tc.year)
			if err != nil {
				t.Fatalf("GetHolidaysForYear(%d) error = %v", tc.year, err)
			}
			if len(holidays) != tc.expectedCount {
				t.Errorf("GetHolidaysForYear(%d) returned %d holidays, expected %d",
					tc.year, len(holidays), tc.expectedCount)
			}
		})
	}
}

// Test 1.2.4: Cache Expiration
func TestCacheExpiration(t *testing.T) {
	db := testutil.SetupTestDB(t)

	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)

	// Set cache days to a short duration for testing
	_ = settingsRepo.Update("feiertage_cache_days", "7")

	// Create expired cache entry manually
	expiresAt := time.Now().AddDate(0, 0, -1) // Yesterday (expired)
	_, err := db.Exec(`INSERT INTO feiertage_cache (year, state, data, fetched_at, expires_at)
		VALUES (?, ?, ?, ?, ?)`, 2025, "BW", `{"Test": {"datum": "2025-01-01"}}`, time.Now().AddDate(0, 0, -8), expiresAt)
	if err != nil {
		t.Fatalf("Failed to create expired cache: %v", err)
	}

	// Test that expired cache returns empty string
	cached, err := holidayRepo.GetCachedHolidays(2025, "BW")
	if err != nil {
		t.Fatalf("GetCachedHolidays error: %v", err)
	}
	if cached != "" {
		t.Error("Expected empty string for expired cache, got data")
	}

	// Test valid cache (not expired)
	validExpiresAt := time.Now().AddDate(0, 0, 3) // 3 days from now
	_, err = db.Exec(`INSERT INTO feiertage_cache (year, state, data, fetched_at, expires_at)
		VALUES (?, ?, ?, ?, ?)`, 2026, "BW", `{"Test2": {"datum": "2026-01-01"}}`, time.Now(), validExpiresAt)
	if err != nil {
		t.Fatalf("Failed to create valid cache: %v", err)
	}

	// Test that valid cache returns data
	cached2, err := holidayRepo.GetCachedHolidays(2026, "BW")
	if err != nil {
		t.Fatalf("GetCachedHolidays error: %v", err)
	}
	if cached2 == "" {
		t.Error("Expected cached data for valid cache, got empty string")
	}
}

// Test IsHoliday with API enabled (integration test)
func TestIsHoliday_WithAPIEnabled(t *testing.T) {
	db := testutil.SetupTestDB(t)

	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	service := NewHolidayService(holidayRepo, settingsRepo)

	// Enable API usage
	_ = settingsRepo.Update("use_feiertage_api", "true")

	// Check a known holiday
	result, err := service.IsHoliday("2025-01-01") // Neujahrstag
	if err != nil {
		t.Skipf("API error (OK for offline testing): %v", err)
		return
	}

	if !result {
		t.Error("Expected 2025-01-01 to be detected as holiday")
	}

	// Check a non-holiday
	result2, err := service.IsHoliday("2025-01-15")
	if err != nil {
		t.Fatalf("IsHoliday error: %v", err)
	}

	if result2 {
		t.Error("Expected 2025-01-15 to NOT be a holiday")
	}
}
