package repository

import (
	"testing"

	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// DONE: TestBlockedDateRepository_Create tests blocked date creation
func TestBlockedDateRepository_Create(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewBlockedDateRepository(db)

	// Create admin user for createdBy foreign key
	adminID := testutil.SeedTestUser(t, db, "admin@test.com", "Admin", "orange")

	t.Run("successful creation", func(t *testing.T) {
		blockedDate := &models.BlockedDate{
			Date:      "2025-12-25",
			Reason:    "Christmas",
			CreatedBy: adminID,
		}

		err := repo.Create(blockedDate)
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}

		if blockedDate.ID == 0 {
			t.Error("BlockedDate ID should be set after creation")
		}
	})

	t.Run("duplicate date", func(t *testing.T) {
		date := "2025-01-01"

		bd1 := &models.BlockedDate{
			Date:      date,
			Reason:    "New Year",
			CreatedBy: adminID,
		}
		repo.Create(bd1)

		bd2 := &models.BlockedDate{
			Date:      date,
			Reason:    "Duplicate",
			CreatedBy: adminID,
		}

		err := repo.Create(bd2)
		if err == nil {
			t.Error("Expected error for duplicate date")
		}
	})
}

// DONE: TestBlockedDateRepository_FindAll tests listing all blocked dates
func TestBlockedDateRepository_FindAll(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewBlockedDateRepository(db)

	adminID := testutil.SeedTestUser(t, db, "admin@test.com", "Admin", "orange")

	t.Run("empty list", func(t *testing.T) {
		dates, err := repo.FindAll()
		if err != nil {
			t.Fatalf("FindAll() failed: %v", err)
		}

		if len(dates) != 0 {
			t.Errorf("Expected 0 blocked dates, got %d", len(dates))
		}
	})

	t.Run("multiple blocked dates", func(t *testing.T) {
		testutil.SeedTestBlockedDate(t, db, "2025-12-25", "Christmas", adminID)
		testutil.SeedTestBlockedDate(t, db, "2025-12-26", "Boxing Day", adminID)

		dates, err := repo.FindAll()
		if err != nil {
			t.Fatalf("FindAll() failed: %v", err)
		}

		if len(dates) != 2 {
			t.Errorf("Expected 2 blocked dates, got %d", len(dates))
		}
	})
}

// DONE: TestBlockedDateRepository_FindByDate tests finding blocked date by specific date
func TestBlockedDateRepository_FindByDate(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewBlockedDateRepository(db)

	adminID := testutil.SeedTestUser(t, db, "admin@test.com", "Admin", "orange")

	testDate := "2025-12-25"
	testutil.SeedTestBlockedDate(t, db, testDate, "Christmas", adminID)

	t.Run("date exists", func(t *testing.T) {
		blockedDate, err := repo.FindByDate(testDate)
		if err != nil {
			t.Fatalf("FindByDate() failed: %v", err)
		}

		// Date might be returned with timestamp, check if it contains the date
		if blockedDate.Date[:10] != testDate {
			t.Errorf("Expected date to start with %s, got %s", testDate, blockedDate.Date)
		}

		if blockedDate.Reason != "Christmas" {
			t.Errorf("Expected reason 'Christmas', got %s", blockedDate.Reason)
		}
	})

	t.Run("date not found", func(t *testing.T) {
		blockedDate, _ := repo.FindByDate("2025-01-01")
		if blockedDate != nil {
			t.Error("Expected nil for non-existent date")
		}
	})

	t.Run("empty date string", func(t *testing.T) {
		blockedDate, _ := repo.FindByDate("")
		if blockedDate != nil {
			t.Error("Expected nil for empty date")
		}
	})
}

// DONE: TestBlockedDateRepository_IsBlocked tests checking if a date is blocked
func TestBlockedDateRepository_IsBlocked(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewBlockedDateRepository(db)

	adminID := testutil.SeedTestUser(t, db, "admin@test.com", "Admin", "orange")

	blockedDate := "2025-12-25"
	testutil.SeedTestBlockedDate(t, db, blockedDate, "Christmas", adminID)

	t.Run("date is blocked", func(t *testing.T) {
		isBlocked, err := repo.IsBlocked(blockedDate)
		if err != nil {
			t.Fatalf("IsBlocked() failed: %v", err)
		}

		if !isBlocked {
			t.Error("Date should be blocked")
		}
	})

	t.Run("date is not blocked", func(t *testing.T) {
		isBlocked, err := repo.IsBlocked("2025-01-01")
		if err != nil {
			t.Fatalf("IsBlocked() failed: %v", err)
		}

		if isBlocked {
			t.Error("Date should not be blocked")
		}
	})

	t.Run("empty date", func(t *testing.T) {
		isBlocked, err := repo.IsBlocked("")
		if err != nil {
			t.Logf("IsBlocked('') returned error: %v", err)
		}

		if isBlocked {
			t.Error("Empty date should not be blocked")
		}
	})
}

// DONE: TestBlockedDateRepository_Delete tests deleting blocked dates
func TestBlockedDateRepository_Delete(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewBlockedDateRepository(db)

	adminID := testutil.SeedTestUser(t, db, "admin@test.com", "Admin", "orange")

	t.Run("successful deletion", func(t *testing.T) {
		blockedID := testutil.SeedTestBlockedDate(t, db, "2025-12-25", "Christmas", adminID)

		err := repo.Delete(blockedID)
		if err != nil {
			t.Fatalf("Delete() failed: %v", err)
		}

		// Verify deletion
		isBlocked, _ := repo.IsBlocked("2025-12-25")
		if isBlocked {
			t.Error("Date should no longer be blocked after deletion")
		}
	})

	t.Run("delete non-existent blocked date", func(t *testing.T) {
		err := repo.Delete(99999)
		// Should handle gracefully
		if err != nil {
			t.Logf("Delete non-existent blocked date returned: %v", err)
		}
	})
}
