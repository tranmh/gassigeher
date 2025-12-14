package repository

import (
	"testing"

	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// TestWalkReportRepository_Basic tests basic CRUD operations
func TestWalkReportRepository_Basic(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewWalkReportRepository(db)

	// Create test user, dog, and completed booking
	userID := testutil.SeedTestUser(t, db, "test@example.com", "Test User", "green")
	dogID := testutil.SeedTestDog(t, db, "Max", "Labrador", "green")
	bookingID := testutil.SeedTestBooking(t, db, userID, dogID, "2025-01-01", "09:00", "completed")

	// Test Create
	notes := "Great walk!"
	report := &models.WalkReport{
		BookingID:      bookingID,
		BehaviorRating: 4,
		EnergyLevel:    "medium",
		Notes:          &notes,
	}

	err := repo.Create(report)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}
	if report.ID == 0 {
		t.Error("WalkReport ID should be set after creation")
	}

	// Test FindByID
	found, err := repo.FindByID(report.ID)
	if err != nil {
		t.Fatalf("FindByID() failed: %v", err)
	}
	if found == nil {
		t.Fatal("Expected report, got nil")
	}
	if found.BehaviorRating != 4 {
		t.Errorf("Expected rating 4, got %d", found.BehaviorRating)
	}

	// Test FindByBookingID
	byBooking, err := repo.FindByBookingID(bookingID)
	if err != nil {
		t.Fatalf("FindByBookingID() failed: %v", err)
	}
	if byBooking == nil {
		t.Fatal("Expected report by booking, got nil")
	}

	// Test Update
	report.BehaviorRating = 5
	err = repo.Update(report)
	if err != nil {
		t.Fatalf("Update() failed: %v", err)
	}
	updated, _ := repo.FindByID(report.ID)
	if updated.BehaviorRating != 5 {
		t.Errorf("Expected rating 5 after update, got %d", updated.BehaviorRating)
	}

	// Test helper methods
	userIDResult, err := repo.GetBookingUserID(bookingID)
	if err != nil {
		t.Fatalf("GetBookingUserID() failed: %v", err)
	}
	if userIDResult != userID {
		t.Errorf("Expected user ID %d, got %d", userID, userIDResult)
	}

	isCompleted, err := repo.IsBookingCompleted(bookingID)
	if err != nil {
		t.Fatalf("IsBookingCompleted() failed: %v", err)
	}
	if !isCompleted {
		t.Error("Expected booking to be completed")
	}

	// Test Delete
	err = repo.Delete(report.ID)
	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}
	deleted, _ := repo.FindByID(report.ID)
	if deleted != nil {
		t.Error("Report should be deleted")
	}
}

// TestWalkReportRepository_FindByDogID tests the FindByDogID method separately
func TestWalkReportRepository_FindByDogID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewWalkReportRepository(db)

	userID := testutil.SeedTestUser(t, db, "test@example.com", "Test User", "green")
	dogID := testutil.SeedTestDog(t, db, "Max", "Labrador", "green")

	// Create bookings and reports
	booking1ID := testutil.SeedTestBooking(t, db, userID, dogID, "2025-01-01", "09:00", "completed")
	booking2ID := testutil.SeedTestBooking(t, db, userID, dogID, "2025-01-02", "10:00", "completed")

	testutil.SeedTestWalkReport(t, db, booking1ID, 4, "medium", "Walk 1")
	testutil.SeedTestWalkReport(t, db, booking2ID, 5, "high", "Walk 2")

	reports, err := repo.FindByDogID(dogID, 10)
	if err != nil {
		t.Fatalf("FindByDogID() failed: %v", err)
	}
	if len(reports) != 2 {
		t.Errorf("Expected 2 reports, got %d", len(reports))
	}
}

// TestWalkReportRepository_Stats tests GetReportStats
func TestWalkReportRepository_Stats(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewWalkReportRepository(db)

	userID := testutil.SeedTestUser(t, db, "test@example.com", "Test User", "green")
	dogID := testutil.SeedTestDog(t, db, "Max", "Labrador", "green")

	booking1ID := testutil.SeedTestBooking(t, db, userID, dogID, "2025-01-01", "09:00", "completed")
	booking2ID := testutil.SeedTestBooking(t, db, userID, dogID, "2025-01-02", "10:00", "completed")

	testutil.SeedTestWalkReport(t, db, booking1ID, 4, "medium", "Walk 1")
	testutil.SeedTestWalkReport(t, db, booking2ID, 5, "high", "Walk 2")

	stats, err := repo.GetReportStats(dogID)
	if err != nil {
		t.Fatalf("GetReportStats() failed: %v", err)
	}
	if stats.TotalWalks != 2 {
		t.Errorf("Expected 2 total walks, got %d", stats.TotalWalks)
	}
}
