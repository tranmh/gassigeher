package repository

import (
	"testing"

	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// TestColorCategoryRepository_Create tests color category creation
func TestColorCategoryRepository_Create(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewColorCategoryRepository(db)

	t.Run("successful creation", func(t *testing.T) {
		patternIcon := "star"
		color := &models.ColorCategory{
			Name:        "test-color",
			HexCode:     "#ff5500",
			PatternIcon: &patternIcon,
			SortOrder:   100,
		}

		err := repo.Create(0, color) // tenantID = 0
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}

		if color.ID == 0 {
			t.Error("ColorCategory ID should be set after creation")
		}
	})

	t.Run("duplicate name fails", func(t *testing.T) {
		// First creation
		color1 := &models.ColorCategory{
			Name:      "unique-color",
			HexCode:   "#aabbcc",
			SortOrder: 101,
		}
		err := repo.Create(0, color1) // tenantID = 0
		if err != nil {
			t.Fatalf("First Create() failed: %v", err)
		}

		// Second creation with same name should fail
		color2 := &models.ColorCategory{
			Name:      "unique-color",
			HexCode:   "#ddeeff",
			SortOrder: 102,
		}
		err = repo.Create(0, color2) // tenantID = 0
		if err == nil {
			t.Error("Expected error for duplicate name, got nil")
		}
	})
}

// TestColorCategoryRepository_FindByID tests finding color by ID
func TestColorCategoryRepository_FindByID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewColorCategoryRepository(db)

	t.Run("color exists", func(t *testing.T) {
		colorID := testutil.SeedTestColorCategory(t, db, "find-me", "#123456", 10)

		color, err := repo.FindByID(0, colorID) // tenantID = 0
		if err != nil {
			t.Fatalf("FindByID() failed: %v", err)
		}

		if color.ID != colorID {
			t.Errorf("Expected ID %d, got %d", colorID, color.ID)
		}

		if color.Name != "find-me" {
			t.Errorf("Expected name 'find-me', got %s", color.Name)
		}

		if color.HexCode != "#123456" {
			t.Errorf("Expected hex_code '#123456', got %s", color.HexCode)
		}
	})

	t.Run("color not found", func(t *testing.T) {
		color, _ := repo.FindByID(0, 99999) // tenantID = 0
		if color != nil {
			t.Error("Expected nil for non-existent ID")
		}
	})
}

// TestColorCategoryRepository_FindByName tests finding color by name
func TestColorCategoryRepository_FindByName(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewColorCategoryRepository(db)

	t.Run("color exists", func(t *testing.T) {
		testutil.SeedTestColorCategory(t, db, "named-color", "#abcdef", 20)

		color, err := repo.FindByName(0, "named-color") // tenantID = 0
		if err != nil {
			t.Fatalf("FindByName() failed: %v", err)
		}

		if color.Name != "named-color" {
			t.Errorf("Expected name 'named-color', got %s", color.Name)
		}
	})

	t.Run("color not found", func(t *testing.T) {
		color, _ := repo.FindByName(0, "non-existent") // tenantID = 0
		if color != nil {
			t.Error("Expected nil for non-existent name")
		}
	})
}

// TestColorCategoryRepository_FindAll tests finding all colors
func TestColorCategoryRepository_FindAll(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewColorCategoryRepository(db)

	// Note: Migration 024 creates 7 default colors
	t.Run("returns all colors ordered by sort_order", func(t *testing.T) {
		colors, err := repo.FindAll(0) // tenantID = 0
		if err != nil {
			t.Fatalf("FindAll() failed: %v", err)
		}

		// Should have at least the 5 default colors
		if len(colors) < 5 {
			t.Errorf("Expected at least 5 colors, got %d", len(colors))
		}

		// Verify ordering
		for i := 1; i < len(colors); i++ {
			if colors[i].SortOrder < colors[i-1].SortOrder {
				t.Error("Colors should be ordered by sort_order")
			}
		}
	})
}

// TestColorCategoryRepository_Update tests updating color
func TestColorCategoryRepository_Update(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewColorCategoryRepository(db)

	t.Run("successful update", func(t *testing.T) {
		colorID := testutil.SeedTestColorCategory(t, db, "to-update", "#111111", 30)

		color, _ := repo.FindByID(0, colorID) // tenantID = 0
		color.Name = "updated-name"
		color.HexCode = "#999999"

		err := repo.Update(0, color) // tenantID = 0
		if err != nil {
			t.Fatalf("Update() failed: %v", err)
		}

		// Verify update
		updated, _ := repo.FindByID(0, colorID) // tenantID = 0
		if updated.Name != "updated-name" {
			t.Errorf("Expected name 'updated-name', got %s", updated.Name)
		}
		if updated.HexCode != "#999999" {
			t.Errorf("Expected hex_code '#999999', got %s", updated.HexCode)
		}
	})
}

// TestColorCategoryRepository_Delete tests deleting color
func TestColorCategoryRepository_Delete(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewColorCategoryRepository(db)

	t.Run("successful delete - no dogs assigned", func(t *testing.T) {
		colorID := testutil.SeedTestColorCategory(t, db, "to-delete", "#222222", 40)

		err := repo.Delete(0, colorID) // tenantID = 0
		if err != nil {
			t.Fatalf("Delete() failed: %v", err)
		}

		// Verify deletion
		deleted, _ := repo.FindByID(0, colorID) // tenantID = 0
		if deleted != nil {
			t.Error("Color should be deleted")
		}
	})

	t.Run("fails to delete color with dogs assigned", func(t *testing.T) {
		colorID := testutil.SeedTestColorCategory(t, db, "has-dogs", "#333333", 50)

		// Create a dog with this color
		now := testutil.Now()
		_, err := db.Exec(`INSERT INTO dogs (tenant_id, name, breed, size, age, color_id, is_available, created_at)
			VALUES (0, ?, ?, ?, ?, ?, 1, ?)`, "TestDog", "Mix", "medium", 3, colorID, now)
		if err != nil {
			t.Fatalf("Failed to create test dog: %v", err)
		}

		// Try to delete - should fail
		err = repo.Delete(0, colorID) // tenantID = 0
		if err == nil {
			t.Error("Expected error when deleting color with dogs assigned")
		}
	})
}

// TestColorCategoryRepository_Count tests counting colors
func TestColorCategoryRepository_Count(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewColorCategoryRepository(db)

	t.Run("returns correct count", func(t *testing.T) {
		count, err := repo.Count(0) // tenantID = 0
		if err != nil {
			t.Fatalf("Count() failed: %v", err)
		}

		// Should have at least 5 default colors from migration
		if count < 5 {
			t.Errorf("Expected at least 5 colors, got %d", count)
		}
	})
}

// TestColorCategoryRepository_CountDogsWithColor tests counting dogs per color
func TestColorCategoryRepository_CountDogsWithColor(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewColorCategoryRepository(db)

	t.Run("color with no dogs", func(t *testing.T) {
		colorID := testutil.SeedTestColorCategory(t, db, "no-dogs", "#444444", 60)

		count, err := repo.CountDogsWithColor(0, colorID) // tenantID = 0
		if err != nil {
			t.Fatalf("CountDogsWithColor() failed: %v", err)
		}

		if count != 0 {
			t.Errorf("Expected 0 dogs, got %d", count)
		}
	})

	t.Run("color with dogs", func(t *testing.T) {
		colorID := testutil.SeedTestColorCategory(t, db, "with-dogs", "#555555", 70)

		// Create dogs with this color (include tenant_id = 0)
		now := testutil.Now()
		_, _ = db.Exec(`INSERT INTO dogs (tenant_id, name, breed, size, age, color_id, is_available, created_at)
			VALUES (0, ?, ?, ?, ?, ?, 1, ?)`, "Dog1", "Mix", "medium", 3, colorID, now)
		_, _ = db.Exec(`INSERT INTO dogs (tenant_id, name, breed, size, age, color_id, is_available, created_at)
			VALUES (0, ?, ?, ?, ?, ?, 1, ?)`, "Dog2", "Lab", "large", 5, colorID, now)

		count, err := repo.CountDogsWithColor(0, colorID) // tenantID = 0
		if err != nil {
			t.Fatalf("CountDogsWithColor() failed: %v", err)
		}

		if count != 2 {
			t.Errorf("Expected 2 dogs, got %d", count)
		}
	})
}

// TestColorCategoryRepository_FindDogsWithColor tests finding dogs by color
func TestColorCategoryRepository_FindDogsWithColor(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewColorCategoryRepository(db)

	t.Run("color with no dogs", func(t *testing.T) {
		colorID := testutil.SeedTestColorCategory(t, db, "empty-dogs", "#aaaaaa", 80)

		dogs, err := repo.FindDogsWithColor(0, colorID) // tenantID = 0
		if err != nil {
			t.Fatalf("FindDogsWithColor() failed: %v", err)
		}

		if len(dogs) != 0 {
			t.Errorf("Expected 0 dogs, got %d", len(dogs))
		}
	})

	t.Run("color with dogs returns correct data", func(t *testing.T) {
		colorID := testutil.SeedTestColorCategory(t, db, "dogs-detail", "#bbbbbb", 81)

		now := testutil.Now()
		_, _ = db.Exec(`INSERT INTO dogs (tenant_id, name, breed, size, age, color_id, is_available, created_at)
			VALUES (0, ?, ?, ?, ?, ?, 1, ?)`, "Bello", "Schäferhund", "large", 5, colorID, now)
		_, _ = db.Exec(`INSERT INTO dogs (tenant_id, name, breed, size, age, color_id, is_available, created_at)
			VALUES (0, ?, ?, ?, ?, ?, 0, ?)`, "Luna", "Dackel", "small", 3, colorID, now)

		dogs, err := repo.FindDogsWithColor(0, colorID) // tenantID = 0
		if err != nil {
			t.Fatalf("FindDogsWithColor() failed: %v", err)
		}

		if len(dogs) != 2 {
			t.Fatalf("Expected 2 dogs, got %d", len(dogs))
		}

		// Ordered by name ASC: Bello, Luna
		if dogs[0]["name"] != "Bello" {
			t.Errorf("Expected first dog 'Bello', got %v", dogs[0]["name"])
		}
		if dogs[0]["breed"] != "Schäferhund" {
			t.Errorf("Expected breed 'Schäferhund', got %v", dogs[0]["breed"])
		}
		if dogs[0]["is_available"] != true {
			t.Errorf("Expected first dog available, got %v", dogs[0]["is_available"])
		}

		if dogs[1]["name"] != "Luna" {
			t.Errorf("Expected second dog 'Luna', got %v", dogs[1]["name"])
		}
		if dogs[1]["is_available"] != false {
			t.Errorf("Expected second dog unavailable, got %v", dogs[1]["is_available"])
		}
	})

	t.Run("does not return dogs from other tenant", func(t *testing.T) {
		colorID := testutil.SeedTestColorCategory(t, db, "tenant-iso-dogs", "#cccccc", 82)

		now := testutil.Now()
		// Dog in tenant 0
		_, _ = db.Exec(`INSERT INTO dogs (tenant_id, name, breed, size, age, color_id, is_available, created_at)
			VALUES (0, ?, ?, ?, ?, ?, 1, ?)`, "MyDog", "Mix", "medium", 2, colorID, now)
		// Dog in tenant 999 (different tenant)
		_, _ = db.Exec(`INSERT INTO dogs (tenant_id, name, breed, size, age, color_id, is_available, created_at)
			VALUES (999, ?, ?, ?, ?, ?, 1, ?)`, "OtherDog", "Lab", "large", 4, colorID, now)

		dogs, err := repo.FindDogsWithColor(0, colorID) // tenantID = 0
		if err != nil {
			t.Fatalf("FindDogsWithColor() failed: %v", err)
		}

		if len(dogs) != 1 {
			t.Errorf("Expected 1 dog (tenant isolation), got %d", len(dogs))
		}
	})
}

// TestColorCategoryRepository_FindUsersWithColor tests finding users by color
func TestColorCategoryRepository_FindUsersWithColor(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewColorCategoryRepository(db)

	t.Run("color with no users", func(t *testing.T) {
		colorID := testutil.SeedTestColorCategory(t, db, "empty-users", "#dddddd", 90)

		users, err := repo.FindUsersWithColor(0, colorID) // tenantID = 0
		if err != nil {
			t.Fatalf("FindUsersWithColor() failed: %v", err)
		}

		if len(users) != 0 {
			t.Errorf("Expected 0 users, got %d", len(users))
		}
	})

	t.Run("color with users returns correct data", func(t *testing.T) {
		colorID := testutil.SeedTestColorCategory(t, db, "users-detail", "#eeeeee", 91)

		userID1 := testutil.SeedTestUserWithoutColors(t, db, "alice@example.com", "Alice Müller", "green")
		userID2 := testutil.SeedTestUserWithoutColors(t, db, "bob@example.com", "Bob Schmidt", "green")
		testutil.SeedTestUserColor(t, db, userID1, colorID)
		testutil.SeedTestUserColor(t, db, userID2, colorID)

		users, err := repo.FindUsersWithColor(0, colorID) // tenantID = 0
		if err != nil {
			t.Fatalf("FindUsersWithColor() failed: %v", err)
		}

		if len(users) != 2 {
			t.Fatalf("Expected 2 users, got %d", len(users))
		}

		// Ordered by last_name ASC: Müller, Schmidt
		if users[0]["first_name"] != "Alice" {
			t.Errorf("Expected first user 'Alice', got %v", users[0]["first_name"])
		}
		if users[0]["last_name"] != "Müller" {
			t.Errorf("Expected last_name 'Müller', got %v", users[0]["last_name"])
		}
		if users[0]["email"] != "alice@example.com" {
			t.Errorf("Expected email 'alice@example.com', got %v", users[0]["email"])
		}

		if users[1]["first_name"] != "Bob" {
			t.Errorf("Expected second user 'Bob', got %v", users[1]["first_name"])
		}
	})

	t.Run("excludes deleted users", func(t *testing.T) {
		colorID := testutil.SeedTestColorCategory(t, db, "excl-deleted", "#f0f0f0", 92)

		userID := testutil.SeedTestUserWithoutColors(t, db, "deleted@example.com", "Deleted User", "green")
		testutil.SeedTestUserColor(t, db, userID, colorID)

		// Mark user as deleted
		_, err := db.Exec(`UPDATE users SET is_deleted = 1 WHERE id = ?`, userID)
		if err != nil {
			t.Fatalf("Failed to mark user as deleted: %v", err)
		}

		users, err := repo.FindUsersWithColor(0, colorID) // tenantID = 0
		if err != nil {
			t.Fatalf("FindUsersWithColor() failed: %v", err)
		}

		if len(users) != 0 {
			t.Errorf("Expected 0 users (deleted excluded), got %d", len(users))
		}
	})

	t.Run("excludes inactive users", func(t *testing.T) {
		colorID := testutil.SeedTestColorCategory(t, db, "excl-inactive", "#f1f1f1", 93)

		userID := testutil.SeedTestUserWithoutColors(t, db, "inactive@example.com", "Inactive User", "green")
		testutil.SeedTestUserColor(t, db, userID, colorID)

		// Mark user as inactive
		_, err := db.Exec(`UPDATE users SET is_active = 0 WHERE id = ?`, userID)
		if err != nil {
			t.Fatalf("Failed to mark user as inactive: %v", err)
		}

		users, err := repo.FindUsersWithColor(0, colorID) // tenantID = 0
		if err != nil {
			t.Fatalf("FindUsersWithColor() failed: %v", err)
		}

		if len(users) != 0 {
			t.Errorf("Expected 0 users (inactive excluded), got %d", len(users))
		}
	})
}
