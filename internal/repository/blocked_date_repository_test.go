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

// TestBlockedDateRepository_CreateWithDogID tests creating dog-specific blocked dates
func TestBlockedDateRepository_CreateWithDogID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewBlockedDateRepository(db)

	adminID := testutil.SeedTestUser(t, db, "admin@test.com", "Admin", "orange")
	dogID := testutil.SeedTestDog(t, db, "Buddy", "Labrador", "green")

	t.Run("create dog-specific block", func(t *testing.T) {
		dogIDPtr := dogID
		blockedDate := &models.BlockedDate{
			Date:      "2025-12-25",
			Reason:    "Vet appointment",
			CreatedBy: adminID,
			DogID:     &dogIDPtr,
		}

		err := repo.Create(blockedDate)
		if err != nil {
			t.Fatalf("Create() with dog_id failed: %v", err)
		}

		if blockedDate.ID == 0 {
			t.Error("BlockedDate ID should be set after creation")
		}
	})

	t.Run("allow same date for different dogs", func(t *testing.T) {
		date := "2025-12-26"
		dog1ID := testutil.SeedTestDog(t, db, "Dog1", "Labrador", "green")
		dog2ID := testutil.SeedTestDog(t, db, "Dog2", "Beagle", "blue")

		// Block date for dog1
		dog1IDPtr := dog1ID
		bd1 := &models.BlockedDate{
			Date:      date,
			Reason:    "Block for Dog1",
			CreatedBy: adminID,
			DogID:     &dog1IDPtr,
		}
		err := repo.Create(bd1)
		if err != nil {
			t.Fatalf("Create for dog1 failed: %v", err)
		}

		// Block same date for dog2 - should succeed
		dog2IDPtr := dog2ID
		bd2 := &models.BlockedDate{
			Date:      date,
			Reason:    "Block for Dog2",
			CreatedBy: adminID,
			DogID:     &dog2IDPtr,
		}
		err = repo.Create(bd2)
		if err != nil {
			t.Fatalf("Create for dog2 should succeed: %v", err)
		}
	})

	t.Run("duplicate dog-date combination fails", func(t *testing.T) {
		date := "2025-12-27"
		dogID := testutil.SeedTestDog(t, db, "DogDupe", "Poodle", "green")

		// First block
		dogIDPtr := dogID
		bd1 := &models.BlockedDate{
			Date:      date,
			Reason:    "First block",
			CreatedBy: adminID,
			DogID:     &dogIDPtr,
		}
		repo.Create(bd1)

		// Duplicate - should fail
		bd2 := &models.BlockedDate{
			Date:      date,
			Reason:    "Duplicate block",
			CreatedBy: adminID,
			DogID:     &dogIDPtr,
		}
		err := repo.Create(bd2)
		if err == nil {
			t.Error("Expected error for duplicate dog-date combination")
		}
	})
}

// TestBlockedDateRepository_IsBlockedForDog tests checking if a date is blocked for a specific dog
func TestBlockedDateRepository_IsBlockedForDog(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewBlockedDateRepository(db)

	adminID := testutil.SeedTestUser(t, db, "admin@test.com", "Admin", "orange")
	dog1ID := testutil.SeedTestDog(t, db, "Buddy", "Labrador", "green")
	dog2ID := testutil.SeedTestDog(t, db, "Max", "Beagle", "blue")

	// Create a global block
	globalDate := "2025-12-25"
	testutil.SeedTestBlockedDate(t, db, globalDate, "Christmas - Global Block", adminID)

	// Create a dog-specific block
	dogSpecificDate := "2025-12-26"
	testutil.SeedTestBlockedDateForDog(t, db, dogSpecificDate, "Vet for Buddy", adminID, dog1ID)

	t.Run("global block affects all dogs", func(t *testing.T) {
		// Dog1 should be blocked on global date
		isBlocked, err := repo.IsBlockedForDog(globalDate, dog1ID)
		if err != nil {
			t.Fatalf("IsBlockedForDog() failed: %v", err)
		}
		if !isBlocked {
			t.Error("Dog1 should be blocked on globally blocked date")
		}

		// Dog2 should also be blocked on global date
		isBlocked, err = repo.IsBlockedForDog(globalDate, dog2ID)
		if err != nil {
			t.Fatalf("IsBlockedForDog() failed: %v", err)
		}
		if !isBlocked {
			t.Error("Dog2 should be blocked on globally blocked date")
		}
	})

	t.Run("dog-specific block only affects that dog", func(t *testing.T) {
		// Dog1 should be blocked on dog-specific date
		isBlocked, err := repo.IsBlockedForDog(dogSpecificDate, dog1ID)
		if err != nil {
			t.Fatalf("IsBlockedForDog() failed: %v", err)
		}
		if !isBlocked {
			t.Error("Dog1 should be blocked on its specific blocked date")
		}

		// Dog2 should NOT be blocked on dog1's specific date
		isBlocked, err = repo.IsBlockedForDog(dogSpecificDate, dog2ID)
		if err != nil {
			t.Fatalf("IsBlockedForDog() failed: %v", err)
		}
		if isBlocked {
			t.Error("Dog2 should NOT be blocked on Dog1's specific blocked date")
		}
	})

	t.Run("unblocked date is not blocked", func(t *testing.T) {
		isBlocked, err := repo.IsBlockedForDog("2025-12-30", dog1ID)
		if err != nil {
			t.Fatalf("IsBlockedForDog() failed: %v", err)
		}
		if isBlocked {
			t.Error("Dog should not be blocked on unblocked date")
		}
	})
}

// TestBlockedDateRepository_GetBlockedDogsForDate tests getting blocked dog IDs for a date
func TestBlockedDateRepository_GetBlockedDogsForDate(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewBlockedDateRepository(db)

	adminID := testutil.SeedTestUser(t, db, "admin@test.com", "Admin", "orange")
	dog1ID := testutil.SeedTestDog(t, db, "Buddy", "Labrador", "green")
	dog2ID := testutil.SeedTestDog(t, db, "Max", "Beagle", "blue")
	dog3ID := testutil.SeedTestDog(t, db, "Rex", "Shepherd", "orange")

	testDate := "2025-12-28"

	// Block dog1 and dog2 on test date (dog3 remains available)
	testutil.SeedTestBlockedDateForDog(t, db, testDate, "Block dog1", adminID, dog1ID)
	testutil.SeedTestBlockedDateForDog(t, db, testDate, "Block dog2", adminID, dog2ID)

	t.Run("returns blocked dog IDs", func(t *testing.T) {
		globalBlock, blockedIDs, err := repo.GetBlockedDogsForDate(testDate)
		if err != nil {
			t.Fatalf("GetBlockedDogsForDate() failed: %v", err)
		}

		if globalBlock {
			t.Error("Expected no global block for this date")
		}

		if len(blockedIDs) != 2 {
			t.Errorf("Expected 2 blocked dogs, got %d", len(blockedIDs))
		}

		// Check both dogs are in the list
		hasD1, hasD2 := false, false
		for _, id := range blockedIDs {
			if id == dog1ID {
				hasD1 = true
			}
			if id == dog2ID {
				hasD2 = true
			}
		}
		if !hasD1 || !hasD2 {
			t.Error("Expected both dog1 and dog2 in blocked list")
		}

		// Dog3 should not be in the list
		for _, id := range blockedIDs {
			if id == dog3ID {
				t.Error("Dog3 should not be in blocked list")
			}
		}
	})

	t.Run("returns empty list for unblocked date", func(t *testing.T) {
		globalBlock, blockedIDs, err := repo.GetBlockedDogsForDate("2025-12-30")
		if err != nil {
			t.Fatalf("GetBlockedDogsForDate() failed: %v", err)
		}

		if globalBlock {
			t.Error("Expected no global block for unblocked date")
		}

		if len(blockedIDs) != 0 {
			t.Errorf("Expected 0 blocked dogs for unblocked date, got %d", len(blockedIDs))
		}
	})

	t.Run("returns global block flag when date is globally blocked", func(t *testing.T) {
		globalDate := "2025-12-31"
		testutil.SeedTestBlockedDate(t, db, globalDate, "New Year - Global", adminID)

		globalBlock, _, err := repo.GetBlockedDogsForDate(globalDate)
		if err != nil {
			t.Fatalf("GetBlockedDogsForDate() failed: %v", err)
		}

		if !globalBlock {
			t.Error("Expected global block flag to be true for globally blocked date")
		}
	})
}

// TestBlockedDateRepository_FindByDateAndDog tests finding a blocked date by date and dog
func TestBlockedDateRepository_FindByDateAndDog(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewBlockedDateRepository(db)

	adminID := testutil.SeedTestUser(t, db, "admin@test.com", "Admin", "orange")
	dogID := testutil.SeedTestDog(t, db, "Buddy", "Labrador", "green")

	testDate := "2025-12-29"
	testutil.SeedTestBlockedDateForDog(t, db, testDate, "Vet appointment", adminID, dogID)

	t.Run("finds existing dog-specific block", func(t *testing.T) {
		dogIDPtr := dogID
		blocked, err := repo.FindByDateAndDog(testDate, &dogIDPtr)
		if err != nil {
			t.Fatalf("FindByDateAndDog() failed: %v", err)
		}

		if blocked == nil {
			t.Fatal("Expected to find blocked date")
		}

		if blocked.Reason != "Vet appointment" {
			t.Errorf("Expected reason 'Vet appointment', got '%s'", blocked.Reason)
		}

		if blocked.DogID == nil || *blocked.DogID != dogID {
			t.Error("Expected dog_id to match")
		}
	})

	t.Run("returns nil for non-blocked dog", func(t *testing.T) {
		otherDogID := testutil.SeedTestDog(t, db, "Other", "Poodle", "green")
		otherDogIDPtr := otherDogID
		blocked, err := repo.FindByDateAndDog(testDate, &otherDogIDPtr)
		if err != nil {
			t.Fatalf("FindByDateAndDog() failed: %v", err)
		}

		if blocked != nil {
			t.Error("Expected nil for non-blocked dog")
		}
	})

	t.Run("returns nil for non-blocked date", func(t *testing.T) {
		dogIDPtr := dogID
		blocked, err := repo.FindByDateAndDog("2025-12-30", &dogIDPtr)
		if err != nil {
			t.Fatalf("FindByDateAndDog() failed: %v", err)
		}

		if blocked != nil {
			t.Error("Expected nil for non-blocked date")
		}
	})
}

// TestBlockedDateRepository_FindAllWithDogInfo tests listing all blocked dates with dog info
func TestBlockedDateRepository_FindAllWithDogInfo(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewBlockedDateRepository(db)

	adminID := testutil.SeedTestUser(t, db, "admin@test.com", "Admin", "orange")
	dogID := testutil.SeedTestDog(t, db, "Buddy", "Labrador", "green")

	// Create global block
	testutil.SeedTestBlockedDate(t, db, "2025-12-25", "Christmas - Global", adminID)

	// Create dog-specific block
	testutil.SeedTestBlockedDateForDog(t, db, "2025-12-26", "Vet for Buddy", adminID, dogID)

	t.Run("returns both global and dog-specific blocks with dog info", func(t *testing.T) {
		dates, err := repo.FindAll()
		if err != nil {
			t.Fatalf("FindAll() failed: %v", err)
		}

		if len(dates) != 2 {
			t.Errorf("Expected 2 blocked dates, got %d", len(dates))
		}

		// Check for global block (no dog)
		var globalFound, dogSpecificFound bool
		for _, bd := range dates {
			if bd.DogID == nil {
				globalFound = true
				if bd.DogName != nil {
					t.Error("Global block should have nil DogName")
				}
			} else if *bd.DogID == dogID {
				dogSpecificFound = true
				if bd.DogName == nil || *bd.DogName != "Buddy" {
					t.Errorf("Expected DogName 'Buddy', got %v", bd.DogName)
				}
			}
		}

		if !globalFound {
			t.Error("Expected to find global block")
		}
		if !dogSpecificFound {
			t.Error("Expected to find dog-specific block")
		}
	})
}
