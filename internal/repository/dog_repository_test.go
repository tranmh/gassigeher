package repository

import (
	"testing"

	"github.com/tranm/gassigeher/internal/models"
	"github.com/tranm/gassigeher/internal/testutil"
)

// DONE: TestDogRepository_Create tests dog creation
func TestDogRepository_Create(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewDogRepository(db)

	t.Run("successful creation", func(t *testing.T) {
		dog := &models.Dog{
			Name:        "Bella",
			Breed:       "Labrador",
			Size:        "large",
			Age:         5,
			Category:    "green",
			IsAvailable: true,
		}

		err := repo.Create(dog)
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}

		if dog.ID == 0 {
			t.Error("Dog ID should be set after creation")
		}
	})

	t.Run("creation with optional fields", func(t *testing.T) {
		needs := "Needs gentle handling"
		pickup := "Main entrance"
		route := "Park route"
		duration := 45
		instructions := "Please use short leash"
		morningTime := "09:00"
		eveningTime := "16:00"

		dog := &models.Dog{
			Name:                "Max",
			Breed:               "Beagle",
			Size:                "medium",
			Age:                 3,
			Category:            "blue",
			SpecialNeeds:        &needs,
			PickupLocation:      &pickup,
			WalkRoute:           &route,
			WalkDuration:        &duration,
			SpecialInstructions: &instructions,
			DefaultMorningTime:  &morningTime,
			DefaultEveningTime:  &eveningTime,
			IsAvailable:         true,
		}

		err := repo.Create(dog)
		if err != nil {
			t.Fatalf("Create() with optional fields failed: %v", err)
		}

		if dog.ID == 0 {
			t.Error("Dog ID should be set")
		}
	})
}

// DONE: TestDogRepository_FindByID tests finding dogs by ID
func TestDogRepository_FindByID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewDogRepository(db)

	t.Run("dog exists", func(t *testing.T) {
		dogID := testutil.SeedTestDog(t, db, "Bella", "Labrador", "green")

		dog, err := repo.FindByID(dogID)
		if err != nil {
			t.Fatalf("FindByID() failed: %v", err)
		}

		if dog.ID != dogID {
			t.Errorf("Expected ID %d, got %d", dogID, dog.ID)
		}

		if dog.Name != "Bella" {
			t.Errorf("Expected name 'Bella', got %s", dog.Name)
		}

		if dog.Category != "green" {
			t.Errorf("Expected category 'green', got %s", dog.Category)
		}
	})

	t.Run("dog not found", func(t *testing.T) {
		dog, err := repo.FindByID(99999)
		if err == nil && dog != nil {
			t.Error("Expected error or nil dog for non-existent ID")
		}
	})
}

// DONE: TestDogRepository_FindAll tests listing and filtering dogs
func TestDogRepository_FindAll(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewDogRepository(db)

	// Seed test dogs
	testutil.SeedTestDog(t, db, "Bella", "Labrador", "green")
	testutil.SeedTestDog(t, db, "Max", "Beagle", "blue")
	testutil.SeedTestDog(t, db, "Rocky", "German Shepherd", "orange")

	t.Run("all dogs - no filter", func(t *testing.T) {
		dogs, err := repo.FindAll(nil)
		if err != nil {
			t.Fatalf("FindAll() failed: %v", err)
		}

		if len(dogs) != 3 {
			t.Errorf("Expected 3 dogs, got %d", len(dogs))
		}
	})

	t.Run("filter by category", func(t *testing.T) {
		category := "green"
		filter := &models.DogFilterRequest{
			Category: &category,
		}

		dogs, err := repo.FindAll(filter)
		if err != nil {
			t.Fatalf("FindAll with category filter failed: %v", err)
		}

		if len(dogs) != 1 {
			t.Errorf("Expected 1 green dog, got %d", len(dogs))
		}

		if len(dogs) > 0 && dogs[0].Name != "Bella" {
			t.Errorf("Expected dog 'Bella', got %s", dogs[0].Name)
		}
	})

	t.Run("filter by available", func(t *testing.T) {
		available := true
		filter := &models.DogFilterRequest{
			Available: &available,
		}

		dogs, err := repo.FindAll(filter)
		if err != nil {
			t.Fatalf("FindAll with available filter failed: %v", err)
		}

		// All seeded dogs are available
		if len(dogs) != 3 {
			t.Errorf("Expected 3 available dogs, got %d", len(dogs))
		}

		for _, dog := range dogs {
			if !dog.IsAvailable {
				t.Errorf("Dog %s should be available", dog.Name)
			}
		}
	})
}

// DONE: TestDogRepository_Update tests updating dog information
func TestDogRepository_Update(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewDogRepository(db)

	t.Run("successful update", func(t *testing.T) {
		dogID := testutil.SeedTestDog(t, db, "Bella", "Labrador", "green")

		// Get dog
		dog, _ := repo.FindByID(dogID)

		// Update fields
		dog.Name = "Bella Updated"
		dog.Age = 6
		dog.Category = "blue"

		err := repo.Update(dog)
		if err != nil {
			t.Fatalf("Update() failed: %v", err)
		}

		// Verify updates
		updated, _ := repo.FindByID(dogID)
		if updated.Name != "Bella Updated" {
			t.Errorf("Expected name 'Bella Updated', got %s", updated.Name)
		}
		if updated.Age != 6 {
			t.Errorf("Expected age 6, got %d", updated.Age)
		}
		if updated.Category != "blue" {
			t.Errorf("Expected category 'blue', got %s", updated.Category)
		}
	})
}

// DONE: TestDogRepository_Delete tests dog deletion
func TestDogRepository_Delete(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewDogRepository(db)

	t.Run("successful deletion", func(t *testing.T) {
		dogID := testutil.SeedTestDog(t, db, "Bella", "Labrador", "green")

		err := repo.Delete(dogID)
		if err != nil {
			t.Fatalf("Delete() failed: %v", err)
		}

		// Verify dog is deleted
		dog, err := repo.FindByID(dogID)
		if dog != nil {
			t.Error("Dog should be deleted")
		}
	})

	t.Run("delete non-existent dog", func(t *testing.T) {
		err := repo.Delete(99999)
		// Should not error or should handle gracefully
		if err != nil {
			t.Logf("Delete non-existent dog returned: %v", err)
		}
	})
}

// DONE: TestDogRepository_ToggleAvailability tests toggling dog availability
func TestDogRepository_ToggleAvailability(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewDogRepository(db)

	t.Run("make dog unavailable with reason", func(t *testing.T) {
		dogID := testutil.SeedTestDog(t, db, "Bella", "Labrador", "green")

		reason := "Sick"
		err := repo.ToggleAvailability(dogID, false, &reason)
		if err != nil {
			t.Fatalf("ToggleAvailability() failed: %v", err)
		}

		// Verify dog is unavailable
		dog, _ := repo.FindByID(dogID)
		if dog.IsAvailable {
			t.Error("Dog should be unavailable")
		}
		if dog.UnavailableReason == nil || *dog.UnavailableReason != reason {
			t.Errorf("Expected unavailable reason '%s', got %v", reason, dog.UnavailableReason)
		}
		if dog.UnavailableSince == nil {
			t.Error("UnavailableSince should be set")
		}
	})

	t.Run("make dog available again", func(t *testing.T) {
		dogID := testutil.SeedTestDog(t, db, "Max", "Beagle", "blue")

		// First make unavailable
		reason := "Test"
		repo.ToggleAvailability(dogID, false, &reason)

		// Then make available
		err := repo.ToggleAvailability(dogID, true, nil)
		if err != nil {
			t.Fatalf("ToggleAvailability(true) failed: %v", err)
		}

		// Verify dog is available
		dog, _ := repo.FindByID(dogID)
		if !dog.IsAvailable {
			t.Error("Dog should be available")
		}
		if dog.UnavailableReason != nil && *dog.UnavailableReason != "" {
			t.Error("UnavailableReason should be cleared")
		}
	})
}

// DONE: TestDogRepository_GetBreeds tests getting unique breed list
func TestDogRepository_GetBreeds(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewDogRepository(db)

	t.Run("get breeds from multiple dogs", func(t *testing.T) {
		// Seed dogs with different breeds
		testutil.SeedTestDog(t, db, "Bella", "Labrador", "green")
		testutil.SeedTestDog(t, db, "Max", "Beagle", "blue")
		testutil.SeedTestDog(t, db, "Rocky", "Labrador", "green") // Duplicate breed

		breeds, err := repo.GetBreeds()
		if err != nil {
			t.Fatalf("GetBreeds() failed: %v", err)
		}

		// Should return unique breeds
		if len(breeds) != 2 {
			t.Errorf("Expected 2 unique breeds, got %d", len(breeds))
		}

		// Check breeds are present (order may vary)
		hasLabrador := false
		hasBeagle := false
		for _, breed := range breeds {
			if breed == "Labrador" {
				hasLabrador = true
			}
			if breed == "Beagle" {
				hasBeagle = true
			}
		}

		if !hasLabrador {
			t.Error("Expected 'Labrador' in breeds")
		}
		if !hasBeagle {
			t.Error("Expected 'Beagle' in breeds")
		}
	})

	t.Run("no dogs in database", func(t *testing.T) {
		// Use fresh DB
		db2 := testutil.SetupTestDB(t)
		repo2 := NewDogRepository(db2)

		breeds, err := repo2.GetBreeds()
		if err != nil {
			t.Fatalf("GetBreeds() on empty database failed: %v", err)
		}

		if len(breeds) != 0 {
			t.Errorf("Expected 0 breeds, got %d", len(breeds))
		}
	})
}

// DONE: TestCanUserAccessDog tests experience level access control
func TestCanUserAccessDog(t *testing.T) {
	tests := []struct {
		name         string
		userLevel    string
		dogCategory  string
		expectedAccess bool
	}{
		// Green user tests
		{
			name:         "green user can access green dog",
			userLevel:    "green",
			dogCategory:  "green",
			expectedAccess: true,
		},
		{
			name:         "green user cannot access blue dog",
			userLevel:    "green",
			dogCategory:  "blue",
			expectedAccess: false,
		},
		{
			name:         "green user cannot access orange dog",
			userLevel:    "green",
			dogCategory:  "orange",
			expectedAccess: false,
		},

		// Blue user tests
		{
			name:         "blue user can access green dog",
			userLevel:    "blue",
			dogCategory:  "green",
			expectedAccess: true,
		},
		{
			name:         "blue user can access blue dog",
			userLevel:    "blue",
			dogCategory:  "blue",
			expectedAccess: true,
		},
		{
			name:         "blue user cannot access orange dog",
			userLevel:    "blue",
			dogCategory:  "orange",
			expectedAccess: false,
		},

		// Orange user tests
		{
			name:         "orange user can access green dog",
			userLevel:    "orange",
			dogCategory:  "green",
			expectedAccess: true,
		},
		{
			name:         "orange user can access blue dog",
			userLevel:    "orange",
			dogCategory:  "blue",
			expectedAccess: true,
		},
		{
			name:         "orange user can access orange dog",
			userLevel:    "orange",
			dogCategory:  "orange",
			expectedAccess: true,
		},

		// Case insensitivity tests
		{
			name:         "case insensitive - GREEN user, green dog",
			userLevel:    "GREEN",
			dogCategory:  "green",
			expectedAccess: true,
		},
		{
			name:         "case insensitive - Blue user, BLUE dog",
			userLevel:    "Blue",
			dogCategory:  "BLUE",
			expectedAccess: true,
		},

		// Invalid level tests
		{
			name:         "invalid user level",
			userLevel:    "red",
			dogCategory:  "green",
			expectedAccess: false,
		},
		{
			name:         "invalid dog category",
			userLevel:    "green",
			dogCategory:  "purple",
			expectedAccess: false,
		},
		{
			name:         "empty user level",
			userLevel:    "",
			dogCategory:  "green",
			expectedAccess: false,
		},
		{
			name:         "empty dog category",
			userLevel:    "green",
			dogCategory:  "",
			expectedAccess: false,
		},
		{
			name:         "both invalid",
			userLevel:    "invalid",
			dogCategory:  "invalid",
			expectedAccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CanUserAccessDog(tt.userLevel, tt.dogCategory)
			if result != tt.expectedAccess {
				t.Errorf("CanUserAccessDog(%q, %q) = %v, expected %v",
					tt.userLevel, tt.dogCategory, result, tt.expectedAccess)
			}
		})
	}
}
