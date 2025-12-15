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

		err := repo.Create(color)
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
		err := repo.Create(color1)
		if err != nil {
			t.Fatalf("First Create() failed: %v", err)
		}

		// Second creation with same name should fail
		color2 := &models.ColorCategory{
			Name:      "unique-color",
			HexCode:   "#ddeeff",
			SortOrder: 102,
		}
		err = repo.Create(color2)
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

		color, err := repo.FindByID(colorID)
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
		color, _ := repo.FindByID(99999)
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

		color, err := repo.FindByName("named-color")
		if err != nil {
			t.Fatalf("FindByName() failed: %v", err)
		}

		if color.Name != "named-color" {
			t.Errorf("Expected name 'named-color', got %s", color.Name)
		}
	})

	t.Run("color not found", func(t *testing.T) {
		color, _ := repo.FindByName("non-existent")
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
		colors, err := repo.FindAll()
		if err != nil {
			t.Fatalf("FindAll() failed: %v", err)
		}

		// Should have at least the 7 default colors
		if len(colors) < 7 {
			t.Errorf("Expected at least 7 colors, got %d", len(colors))
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

		color, _ := repo.FindByID(colorID)
		color.Name = "updated-name"
		color.HexCode = "#999999"

		err := repo.Update(color)
		if err != nil {
			t.Fatalf("Update() failed: %v", err)
		}

		// Verify update
		updated, _ := repo.FindByID(colorID)
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

		err := repo.Delete(colorID)
		if err != nil {
			t.Fatalf("Delete() failed: %v", err)
		}

		// Verify deletion
		deleted, _ := repo.FindByID(colorID)
		if deleted != nil {
			t.Error("Color should be deleted")
		}
	})

	t.Run("fails to delete color with dogs assigned", func(t *testing.T) {
		colorID := testutil.SeedTestColorCategory(t, db, "has-dogs", "#333333", 50)

		// Create a dog with this color
		_, err := db.Exec(`INSERT INTO dogs (name, breed, size, age, category, color_id, is_available, created_at)
			VALUES (?, ?, ?, ?, ?, ?, 1, datetime('now'))`, "TestDog", "Mix", "medium", 3, "green", colorID)
		if err != nil {
			t.Fatalf("Failed to create test dog: %v", err)
		}

		// Try to delete - should fail
		err = repo.Delete(colorID)
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
		count, err := repo.Count()
		if err != nil {
			t.Fatalf("Count() failed: %v", err)
		}

		// Should have at least 7 default colors from migration
		if count < 7 {
			t.Errorf("Expected at least 7 colors, got %d", count)
		}
	})
}

// TestColorCategoryRepository_CountDogsWithColor tests counting dogs per color
func TestColorCategoryRepository_CountDogsWithColor(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewColorCategoryRepository(db)

	t.Run("color with no dogs", func(t *testing.T) {
		colorID := testutil.SeedTestColorCategory(t, db, "no-dogs", "#444444", 60)

		count, err := repo.CountDogsWithColor(colorID)
		if err != nil {
			t.Fatalf("CountDogsWithColor() failed: %v", err)
		}

		if count != 0 {
			t.Errorf("Expected 0 dogs, got %d", count)
		}
	})

	t.Run("color with dogs", func(t *testing.T) {
		colorID := testutil.SeedTestColorCategory(t, db, "with-dogs", "#555555", 70)

		// Create dogs with this color
		_, _ = db.Exec(`INSERT INTO dogs (name, breed, size, age, category, color_id, is_available, created_at)
			VALUES (?, ?, ?, ?, ?, ?, 1, datetime('now'))`, "Dog1", "Mix", "medium", 3, "green", colorID)
		_, _ = db.Exec(`INSERT INTO dogs (name, breed, size, age, category, color_id, is_available, created_at)
			VALUES (?, ?, ?, ?, ?, ?, 1, datetime('now'))`, "Dog2", "Lab", "large", 5, "green", colorID)

		count, err := repo.CountDogsWithColor(colorID)
		if err != nil {
			t.Fatalf("CountDogsWithColor() failed: %v", err)
		}

		if count != 2 {
			t.Errorf("Expected 2 dogs, got %d", count)
		}
	})
}
