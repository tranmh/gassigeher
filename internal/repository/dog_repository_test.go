package repository

import (
	"testing"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/testutil"
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

	t.Run("creation with all fields", func(t *testing.T) {
		specialNeeds := "Needs gentle handling"
		pickupLocation := "Building A"
		walkRoute := "Park trail"
		walkDuration := 30
		specialInstructions := "Please use harness"
		morningTime := "09:00"
		eveningTime := "17:00"
		photo := "dog.jpg"

		dog := &models.Dog{
			Name:                "Complete Dog",
			Breed:               "Golden Retriever",
			Size:                "large",
			Age:                 7,
			Category:            "orange",
			IsAvailable:         true,
			SpecialNeeds:        &specialNeeds,
			PickupLocation:      &pickupLocation,
			WalkRoute:           &walkRoute,
			WalkDuration:        &walkDuration,
			SpecialInstructions: &specialInstructions,
			DefaultMorningTime:  &morningTime,
			DefaultEveningTime:  &eveningTime,
			Photo:               &photo,
		}

		err := repo.Create(dog)
		if err != nil {
			t.Fatalf("Create() with all fields failed: %v", err)
		}

		if dog.ID == 0 {
			t.Error("Dog ID should be set after creation")
		}

		// Verify all fields
		created, _ := repo.FindByID(dog.ID)
		if created.SpecialNeeds == nil || *created.SpecialNeeds != specialNeeds {
			t.Errorf("Expected special needs '%s', got %v", specialNeeds, created.SpecialNeeds)
		}
		if created.WalkRoute == nil || *created.WalkRoute != walkRoute {
			t.Errorf("Expected walk route '%s', got %v", walkRoute, created.WalkRoute)
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

		// ColorID 1 = gruen (green)
		if dog.ColorID == nil || *dog.ColorID != 1 {
			t.Errorf("Expected color_id 1, got %v", dog.ColorID)
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

	t.Run("filter by breed", func(t *testing.T) {
		breed := "Labrador"
		filter := &models.DogFilterRequest{
			Breed: &breed,
		}

		dogs, err := repo.FindAll(filter)
		if err != nil {
			t.Fatalf("FindAll with breed filter failed: %v", err)
		}

		if len(dogs) != 1 {
			t.Errorf("Expected 1 Labrador, got %d", len(dogs))
		}

		if len(dogs) > 0 && dogs[0].Breed != "Labrador" {
			t.Errorf("Expected breed 'Labrador', got %s", dogs[0].Breed)
		}
	})

	t.Run("filter by breed - case insensitive", func(t *testing.T) {
		breed := "beagle"
		filter := &models.DogFilterRequest{
			Breed: &breed,
		}

		dogs, err := repo.FindAll(filter)
		if err != nil {
			t.Fatalf("FindAll with breed filter failed: %v", err)
		}

		if len(dogs) != 1 {
			t.Errorf("Expected 1 Beagle (case insensitive), got %d", len(dogs))
		}
	})

	t.Run("filter by size", func(t *testing.T) {
		size := "medium"
		filter := &models.DogFilterRequest{
			Size: &size,
		}

		dogs, err := repo.FindAll(filter)
		if err != nil {
			t.Fatalf("FindAll with size filter failed: %v", err)
		}

		// All test dogs are seeded with medium size by SeedTestDog
		if len(dogs) != 3 {
			t.Errorf("Expected 3 medium dogs, got %d", len(dogs))
		}
	})

	t.Run("filter by age range", func(t *testing.T) {
		minAge := 4
		maxAge := 6
		filter := &models.DogFilterRequest{
			MinAge: &minAge,
			MaxAge: &maxAge,
		}

		dogs, err := repo.FindAll(filter)
		if err != nil {
			t.Fatalf("FindAll with age filter failed: %v", err)
		}

		// All test dogs are seeded with age 5 by SeedTestDog
		if len(dogs) != 3 {
			t.Errorf("Expected 3 dogs in age range 4-6, got %d", len(dogs))
		}

		for _, dog := range dogs {
			if dog.Age < minAge || dog.Age > maxAge {
				t.Errorf("Dog %s age %d not in range %d-%d", dog.Name, dog.Age, minAge, maxAge)
			}
		}
	})

	t.Run("filter by min age only", func(t *testing.T) {
		minAge := 3
		filter := &models.DogFilterRequest{
			MinAge: &minAge,
		}

		dogs, err := repo.FindAll(filter)
		if err != nil {
			t.Fatalf("FindAll with min age filter failed: %v", err)
		}

		for _, dog := range dogs {
			if dog.Age < minAge {
				t.Errorf("Dog %s age %d is below minimum %d", dog.Name, dog.Age, minAge)
			}
		}
	})

	t.Run("filter by max age only", func(t *testing.T) {
		maxAge := 10
		filter := &models.DogFilterRequest{
			MaxAge: &maxAge,
		}

		dogs, err := repo.FindAll(filter)
		if err != nil {
			t.Fatalf("FindAll with max age filter failed: %v", err)
		}

		for _, dog := range dogs {
			if dog.Age > maxAge {
				t.Errorf("Dog %s age %d is above maximum %d", dog.Name, dog.Age, maxAge)
			}
		}
	})

	t.Run("filter by search - name", func(t *testing.T) {
		search := "Bella"
		filter := &models.DogFilterRequest{
			Search: &search,
		}

		dogs, err := repo.FindAll(filter)
		if err != nil {
			t.Fatalf("FindAll with search filter failed: %v", err)
		}

		if len(dogs) != 1 {
			t.Errorf("Expected 1 dog matching 'Bella', got %d", len(dogs))
		}

		if len(dogs) > 0 && dogs[0].Name != "Bella" {
			t.Errorf("Expected dog 'Bella', got %s", dogs[0].Name)
		}
	})

	t.Run("filter by search - breed", func(t *testing.T) {
		search := "Shepherd"
		filter := &models.DogFilterRequest{
			Search: &search,
		}

		dogs, err := repo.FindAll(filter)
		if err != nil {
			t.Fatalf("FindAll with search filter failed: %v", err)
		}

		if len(dogs) != 1 {
			t.Errorf("Expected 1 dog matching 'Shepherd', got %d", len(dogs))
		}

		if len(dogs) > 0 && dogs[0].Breed != "German Shepherd" {
			t.Errorf("Expected breed 'German Shepherd', got %s", dogs[0].Breed)
		}
	})

	t.Run("filter by search - case insensitive partial match", func(t *testing.T) {
		search := "bel"
		filter := &models.DogFilterRequest{
			Search: &search,
		}

		dogs, err := repo.FindAll(filter)
		if err != nil {
			t.Fatalf("FindAll with search filter failed: %v", err)
		}

		if len(dogs) != 1 {
			t.Errorf("Expected 1 dog matching 'bel', got %d", len(dogs))
		}
	})

	t.Run("filter with multiple criteria", func(t *testing.T) {
		category := "blue"
		available := true
		filter := &models.DogFilterRequest{
			Category:  &category,
			Available: &available,
		}

		dogs, err := repo.FindAll(filter)
		if err != nil {
			t.Fatalf("FindAll with multiple filters failed: %v", err)
		}

		if len(dogs) != 1 {
			t.Errorf("Expected 1 blue available dog, got %d", len(dogs))
		}

		if len(dogs) > 0 {
			// ColorID 5 = dunkelblau (blue)
			if dogs[0].ColorID == nil || *dogs[0].ColorID != 5 {
				t.Errorf("Expected color_id 5, got %v", dogs[0].ColorID)
			}
			if !dogs[0].IsAvailable {
				t.Error("Expected dog to be available")
			}
		}
	})

	t.Run("no results - breed not found", func(t *testing.T) {
		breed := "Poodle"
		filter := &models.DogFilterRequest{
			Breed: &breed,
		}

		dogs, err := repo.FindAll(filter)
		if err != nil {
			t.Fatalf("FindAll with breed filter failed: %v", err)
		}

		if len(dogs) != 0 {
			t.Errorf("Expected 0 Poodles, got %d", len(dogs))
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
		newColorID := 5 // dunkelblau (blue)
		dog.ColorID = &newColorID

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
		if updated.ColorID == nil || *updated.ColorID != 5 {
			t.Errorf("Expected color_id 5, got %v", updated.ColorID)
		}
	})

	t.Run("update with optional fields", func(t *testing.T) {
		dogID := testutil.SeedTestDog(t, db, "Max", "Beagle", "blue")
		dog, _ := repo.FindByID(dogID)

		// Update optional fields
		specialNeeds := "Needs medication"
		walkRoute := "Park trail"
		dog.SpecialNeeds = &specialNeeds
		dog.WalkRoute = &walkRoute

		err := repo.Update(dog)
		if err != nil {
			t.Fatalf("Update() failed: %v", err)
		}

		// Verify updates
		updated, _ := repo.FindByID(dogID)
		if updated.SpecialNeeds == nil || *updated.SpecialNeeds != specialNeeds {
			t.Errorf("Expected special needs '%s', got %v", specialNeeds, updated.SpecialNeeds)
		}
		if updated.WalkRoute == nil || *updated.WalkRoute != walkRoute {
			t.Errorf("Expected walk route '%s', got %v", walkRoute, updated.WalkRoute)
		}
	})

	t.Run("update non-existent dog", func(t *testing.T) {
		dog := &models.Dog{
			ID:       99999,
			Name:     "Nonexistent",
			Breed:    "Test",
			Size:     "small",
			Age:      3,
			Category: "green",
		}

		err := repo.Update(dog)
		// Should not error even if no rows updated
		if err != nil {
			t.Logf("Update non-existent dog returned: %v", err)
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

	t.Run("cannot delete dog with future bookings", func(t *testing.T) {
		dogID := testutil.SeedTestDog(t, db, "Max", "Beagle", "blue")
		userID := testutil.SeedTestUser(t, db, "user@example.com", "Test User", "blue")

		// Create future booking for this dog
		futureDate := time.Now().AddDate(0, 0, 7).Format("2006-01-02")
		testutil.SeedTestBooking(t, db, userID, dogID, futureDate, "09:00", "scheduled")

		// Try to delete dog
		err := repo.Delete(dogID)

		if err == nil {
			t.Error("Expected error when deleting dog with future bookings, got nil")
		}

		// Verify dog still exists
		dog, _ := repo.FindByID(dogID)
		if dog == nil {
			t.Error("Dog should still exist after failed deletion")
		}
	})

	t.Run("can delete dog with only past bookings", func(t *testing.T) {
		dogID := testutil.SeedTestDog(t, db, "Rocky", "Shepherd", "orange")
		userID := testutil.SeedTestUser(t, db, "pastuser@example.com", "Past User", "orange")

		// Create past booking for this dog
		pastDate := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
		testutil.SeedTestBooking(t, db, userID, dogID, pastDate, "16:00", "completed")

		// Should be able to delete dog with past bookings only
		err := repo.Delete(dogID)
		if err != nil {
			t.Fatalf("Delete() should succeed with only past bookings, got error: %v", err)
		}

		// Verify dog is deleted
		dog, _ := repo.FindByID(dogID)
		if dog != nil {
			t.Error("Dog should be deleted")
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

		// Blue user tests (new order: green < orange < blue, so blue is highest)
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
			name:         "blue user can access orange dog",
			userLevel:    "blue",
			dogCategory:  "orange",
			expectedAccess: true,
		},

		// Orange user tests (new order: green < orange < blue)
		{
			name:         "orange user can access green dog",
			userLevel:    "orange",
			dogCategory:  "green",
			expectedAccess: true,
		},
		{
			name:         "orange user cannot access blue dog",
			userLevel:    "orange",
			dogCategory:  "blue",
			expectedAccess: false,
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

// TestCanUserAccessDogByColor tests color-based access control (new system)
func TestCanUserAccessDogByColor(t *testing.T) {
	tests := []struct {
		name           string
		userColorIDs   []int
		dogColorID     int
		expectedAccess bool
	}{
		// User has matching color
		{
			name:           "user with matching color can access",
			userColorIDs:   []int{1, 2, 3},
			dogColorID:     2,
			expectedAccess: true,
		},
		{
			name:           "user with single matching color can access",
			userColorIDs:   []int{5},
			dogColorID:     5,
			expectedAccess: true,
		},

		// User does not have matching color
		{
			name:           "user without matching color cannot access",
			userColorIDs:   []int{1, 2, 3},
			dogColorID:     4,
			expectedAccess: false,
		},
		{
			name:           "user with different single color cannot access",
			userColorIDs:   []int{5},
			dogColorID:     3,
			expectedAccess: false,
		},

		// Edge cases
		{
			name:           "user with no colors cannot access any dog",
			userColorIDs:   []int{},
			dogColorID:     1,
			expectedAccess: false,
		},
		{
			name:           "nil color list cannot access any dog",
			userColorIDs:   nil,
			dogColorID:     1,
			expectedAccess: false,
		},
		{
			name:           "dog with color ID 0 cannot be accessed",
			userColorIDs:   []int{1, 2, 3},
			dogColorID:     0,
			expectedAccess: false,
		},

		// Multiple colors - user has many
		{
			name:           "user with many colors - first matches",
			userColorIDs:   []int{1, 2, 3, 4, 5, 6, 7},
			dogColorID:     1,
			expectedAccess: true,
		},
		{
			name:           "user with many colors - last matches",
			userColorIDs:   []int{1, 2, 3, 4, 5, 6, 7},
			dogColorID:     7,
			expectedAccess: true,
		},
		{
			name:           "user with many colors - middle matches",
			userColorIDs:   []int{1, 2, 3, 4, 5, 6, 7},
			dogColorID:     4,
			expectedAccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CanUserAccessDogByColor(tt.userColorIDs, tt.dogColorID)
			if result != tt.expectedAccess {
				t.Errorf("CanUserAccessDogByColor(%v, %d) = %v, expected %v",
					tt.userColorIDs, tt.dogColorID, result, tt.expectedAccess)
			}
		})
	}
}
