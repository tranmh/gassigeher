package services

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/tranmh/gassigeher/internal/database"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
	"github.com/tranmh/gassigeher/internal/testutil"
	_ "github.com/mattn/go-sqlite3"
)

// Test 1.1.1: ValidateBookingTime - Weekday Allowed Times
func TestValidateBookingTime_WeekdayAllowed(t *testing.T) {
	db := testutil.SetupTestDB(t)

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo)

	testCases := []struct {
		name    string
		date    string
		time    string
		wantErr bool
	}{
		{"Morning window", "2025-01-27", "09:30", false},
		{"Morning window end", "2025-01-27", "11:45", false},
		{"Afternoon window", "2025-01-27", "14:45", false},
		{"Evening window", "2025-01-27", "18:30", false},
		{"Tuesday morning", "2025-01-28", "10:00", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := service.ValidateBookingTime(tc.date, tc.time)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateBookingTime() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// Test 1.1.2: ValidateBookingTime - Weekday Blocked Times
func TestValidateBookingTime_WeekdayBlocked(t *testing.T) {
	db := testutil.SetupTestDB(t)

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo)

	testCases := []struct {
		name            string
		date            string
		time            string
		wantErrContains string
	}{
		{"Lunch block start", "2025-01-27", "13:00", "Mittagspause"},
		{"Lunch block middle", "2025-01-27", "13:45", "Mittagspause"},
		{"Feeding block start", "2025-01-27", "17:00", "Fütterungszeit"},
		{"Feeding block middle", "2025-01-27", "17:30", "Fütterungszeit"},
		{"Before opening", "2025-01-27", "08:00", "außerhalb"},
		{"After closing", "2025-01-27", "20:00", "außerhalb"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := service.ValidateBookingTime(tc.date, tc.time)
			if err == nil {
				t.Error("Expected error, got nil")
				return
			}
			if !strings.Contains(err.Error(), tc.wantErrContains) {
				t.Errorf("Error %v should contain %q", err, tc.wantErrContains)
			}
		})
	}
}

// Test 1.1.3: ValidateBookingTime - Weekend Times
func TestValidateBookingTime_WeekendTimes(t *testing.T) {
	db := testutil.SetupTestDB(t)

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo)

	testCases := []struct {
		name    string
		date    string
		time    string
		wantErr bool
	}{
		{"Saturday morning", "2025-01-25", "10:00", false},
		{"Saturday afternoon", "2025-01-25", "15:00", false},
		{"Sunday morning", "2025-01-26", "11:30", false},
		{"Sunday afternoon", "2025-01-26", "16:30", false},
		{"Saturday feeding block", "2025-01-25", "12:30", true},
		{"Saturday lunch block", "2025-01-25", "13:30", true},
		{"Saturday outside window", "2025-01-25", "17:30", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := service.ValidateBookingTime(tc.date, tc.time)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateBookingTime() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// Test 1.1.4: ValidateBookingTime - Holiday Times
func TestValidateBookingTime_HolidayTimes(t *testing.T) {
	db := testutil.SetupTestDB(t)

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo)

	// Seed holiday: 2025-01-01 (Neujahrstag)
	holiday := &models.CustomHoliday{
		Date:     "2025-01-01",
		Name:     "Neujahrstag",
		IsActive: true,
		Source:   "test",
	}
	err := holidayRepo.CreateHoliday(holiday)
	if err != nil {
		t.Fatalf("Failed to seed holiday: %v", err)
	}

	testCases := []struct {
		name    string
		date    string
		time    string
		wantErr bool
	}{
		{"Holiday morning (weekend rules)", "2025-01-01", "10:00", false},
		{"Holiday afternoon (weekend rules)", "2025-01-01", "15:00", false},
		{"Holiday feeding block (weekend)", "2025-01-01", "12:30", true},
		{"Holiday lunch block (weekend)", "2025-01-01", "13:30", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := service.ValidateBookingTime(tc.date, tc.time)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateBookingTime() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// Test 1.1.5: GetAvailableTimeSlots - Granularity
func TestGetAvailableTimeSlots_Granularity(t *testing.T) {
	db := testutil.SetupTestDB(t)

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo)

	// Test weekday
	slots, err := service.GetAvailableTimeSlots("2025-01-27") // Monday
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify 15-minute intervals present in morning window
	expectedSlots := []string{
		"09:00", "09:15", "09:30", "09:45",
		"10:00", "10:15", "10:30", "10:45",
		"11:00", "11:15", "11:30", "11:45",
	}

	for _, expected := range expectedSlots {
		if !containsTimeSlot(slots, expected) {
			t.Errorf("Expected slot %s not found in results", expected)
		}
	}

	// Verify blocked times NOT present
	blockedSlots := []string{"13:00", "13:15", "13:30", "13:45", "17:00", "17:15"}
	for _, blocked := range blockedSlots {
		if containsTimeSlot(slots, blocked) {
			t.Errorf("Blocked slot %s should not be in results", blocked)
		}
	}

	// Verify slots are in correct format (HH:MM)
	for _, slot := range slots {
		if len(slot) != 5 || slot[2] != ':' {
			t.Errorf("Slot %s has invalid format, expected HH:MM", slot)
		}
	}
}

// Test 1.1.6: RequiresApproval - Morning Walk Detection
func TestRequiresApproval(t *testing.T) {
	db := testutil.SetupTestDB(t)

	settingsRepo := repository.NewSettingsRepository(db)

	// Enable morning approval setting (should already exist from migration)
	err := settingsRepo.Update("morning_walk_requires_approval", "true")
	if err != nil {
		t.Fatalf("Failed to update test setting: %v", err)
	}

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo)

	testCases := []struct {
		time string
		want bool
	}{
		{"09:00", true},
		{"10:30", true},
		{"11:45", true},
		{"12:00", false}, // Boundary
		{"14:00", false},
		{"18:00", false},
	}

	for _, tc := range testCases {
		t.Run(tc.time, func(t *testing.T) {
			requires, err := service.RequiresApproval(tc.time)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if requires != tc.want {
				t.Errorf("RequiresApproval(%s) = %v, want %v", tc.time, requires, tc.want)
			}
		})
	}
}

// Test 1.1.7: GetDayType - Day Type Classification
func TestGetDayType(t *testing.T) {
	db := testutil.SetupTestDB(t)

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo)

	// Seed holidays
	holidays := []models.CustomHoliday{
		{Date: "2025-01-01", Name: "Neujahrstag", IsActive: true, Source: "test"},
		{Date: "2025-01-06", Name: "Heilige Drei Könige", IsActive: true, Source: "test"},
	}
	for _, h := range holidays {
		holiday := h
		_ = holidayRepo.CreateHoliday(&holiday)
	}

	testCases := []struct {
		name string
		date string
		want string
	}{
		{"Monday weekday", "2025-01-27", "weekday"},
		{"Tuesday weekday", "2025-01-28", "weekday"},
		{"Saturday weekend", "2025-01-25", "weekend"},
		{"Sunday weekend", "2025-01-26", "weekend"},
		{"Wednesday holiday (Neujahr)", "2025-01-01", "weekend"},
		{"Monday holiday (Heilige 3 Könige)", "2025-01-06", "weekend"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Use GetRulesForDate to test day type indirectly
			rules, err := service.GetRulesForDate(tc.date)
			if err != nil {
				t.Fatalf("GetRulesForDate() error = %v", err)
			}

			// Verify rules are for correct day type
			for _, rule := range rules {
				if rule.DayType != tc.want {
					t.Errorf("Expected rules for %s, got rules for %s", tc.want, rule.DayType)
					break
				}
			}
		})
	}
}

// Helper function
func containsTimeSlot(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ========================================
// Phase 7: Performance Testing
// ========================================

// Test 7.1.2: Available Slots Generation Performance
// Purpose: Verify time slot generation is fast
func BenchmarkGetAvailableTimeSlots(b *testing.B) {
	// Use the benchmark itself as a testing.T-compatible interface
	db, _ := sql.Open("sqlite3", ":memory:")
	defer db.Close()

	// Setup database (manual migration to avoid T dependency)
	dialect := database.NewSQLiteDialect()
	_ = dialect.ApplySettings(db)
	_ = database.RunMigrationsWithDialect(db, dialect)

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GetAvailableTimeSlots("2025-01-27")
	}
}

// Test 7.1.2: Available Slots Generation for Weekend
func BenchmarkGetAvailableTimeSlots_Weekend(b *testing.B) {
	db, _ := sql.Open("sqlite3", ":memory:")
	defer db.Close()

	dialect := database.NewSQLiteDialect()
	_ = dialect.ApplySettings(db)
	_ = database.RunMigrationsWithDialect(db, dialect)

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GetAvailableTimeSlots("2025-01-25") // Saturday
	}
}

// Test 7.1.3: Booking Validation Performance
// Purpose: Verify booking validation completes quickly
func BenchmarkValidateBookingTime(b *testing.B) {
	db, _ := sql.Open("sqlite3", ":memory:")
	defer db.Close()

	dialect := database.NewSQLiteDialect()
	_ = dialect.ApplySettings(db)
	_ = database.RunMigrationsWithDialect(db, dialect)

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.ValidateBookingTime("2025-01-27", "15:00")
	}
}

// Test 7.1.3: Booking Validation with Holiday Check
func BenchmarkValidateBookingTime_WithHolidayCheck(b *testing.B) {
	db, _ := sql.Open("sqlite3", ":memory:")
	defer db.Close()

	dialect := database.NewSQLiteDialect()
	_ = dialect.ApplySettings(db)
	_ = database.RunMigrationsWithDialect(db, dialect)

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo)

	// Add some holidays to test holiday check performance
	for i := 1; i <= 50; i++ {
		holiday := &models.CustomHoliday{
			Date:     fmt.Sprintf("2025-%02d-%02d", (i%12)+1, (i%28)+1),
			Name:     fmt.Sprintf("Holiday %d", i),
			IsActive: true,
			Source:   "test",
		}
		_ = holidayRepo.CreateHoliday(holiday)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.ValidateBookingTime("2025-01-27", "15:00")
	}
}

// Test 7.1.3: Multiple Booking Validations
func TestValidateBookingTime_Performance(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo)

	// Validate 100 bookings and measure time
	start := time.Now()
	for i := 0; i < 100; i++ {
		_ = service.ValidateBookingTime("2025-01-27", "15:00")
	}
	elapsed := time.Since(start)

	// Target: < 500ms for 100 validations (5ms per validation)
	if elapsed > 500*time.Millisecond {
		t.Errorf("100 validations took %v, expected < 500ms", elapsed)
	}

	t.Logf("100 booking validations completed in %v (avg: %v per validation)",
		elapsed, elapsed/100)
}

// Test 7.1.2: Available Slots Generation Performance Test
func TestGetAvailableTimeSlots_Performance(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	holidayService := NewHolidayService(holidayRepo, settingsRepo)
	service := NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo)

	// Generate slots 100 times and measure time
	start := time.Now()
	for i := 0; i < 100; i++ {
		_, _ = service.GetAvailableTimeSlots("2025-01-27")
	}
	elapsed := time.Since(start)

	// Target: < 1000ms for 100 generations (10ms per generation)
	if elapsed > 1000*time.Millisecond {
		t.Errorf("100 slot generations took %v, expected < 1000ms", elapsed)
	}

	t.Logf("100 time slot generations completed in %v (avg: %v per generation)",
		elapsed, elapsed/100)
}
