package repository

import (
	"testing"

	"github.com/tranmh/gassigeher/internal/testutil"
)

// TestUserColorRepository_AddColorToUser tests adding color to user
func TestUserColorRepository_AddColorToUser(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewUserColorRepository(db)

	t.Run("successful add", func(t *testing.T) {
		userID := testutil.SeedTestUser(t, db, "user1@test.com", "Test User", "green")
		colorID := testutil.SeedTestColorCategory(t, db, "add-color", "#111111", 10)
		adminID := testutil.SeedTestUser(t, db, "admin@test.com", "Admin", "blue")

		err := repo.AddColorToUser(userID, colorID, adminID)
		if err != nil {
			t.Fatalf("AddColorToUser() failed: %v", err)
		}

		// Verify color was added
		hasColor, _ := repo.HasColor(userID, colorID)
		if !hasColor {
			t.Error("User should have the color after adding")
		}
	})

	t.Run("duplicate fails", func(t *testing.T) {
		userID := testutil.SeedTestUser(t, db, "user2@test.com", "Test User 2", "green")
		colorID := testutil.SeedTestColorCategory(t, db, "dup-color", "#222222", 20)
		adminID := testutil.SeedTestUser(t, db, "admin2@test.com", "Admin 2", "blue")

		// First add
		err := repo.AddColorToUser(userID, colorID, adminID)
		if err != nil {
			t.Fatalf("First AddColorToUser() failed: %v", err)
		}

		// Second add should fail
		err = repo.AddColorToUser(userID, colorID, adminID)
		if err == nil {
			t.Error("Expected error for duplicate user-color assignment")
		}
	})
}

// TestUserColorRepository_RemoveColorFromUser tests removing color from user
func TestUserColorRepository_RemoveColorFromUser(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewUserColorRepository(db)

	t.Run("successful remove", func(t *testing.T) {
		userID := testutil.SeedTestUser(t, db, "user3@test.com", "Test User 3", "green")
		colorID := testutil.SeedTestColorCategory(t, db, "remove-color", "#333333", 30)

		testutil.SeedTestUserColor(t, db, userID, colorID)

		err := repo.RemoveColorFromUser(userID, colorID)
		if err != nil {
			t.Fatalf("RemoveColorFromUser() failed: %v", err)
		}

		// Verify color was removed
		hasColor, _ := repo.HasColor(userID, colorID)
		if hasColor {
			t.Error("User should not have the color after removing")
		}
	})

	t.Run("no error if color not assigned", func(t *testing.T) {
		userID := testutil.SeedTestUser(t, db, "user4@test.com", "Test User 4", "green")
		colorID := testutil.SeedTestColorCategory(t, db, "not-assigned", "#444444", 40)

		// User doesn't have this color, but remove should not error
		err := repo.RemoveColorFromUser(userID, colorID)
		if err != nil {
			t.Fatalf("RemoveColorFromUser() should not error for non-assigned color: %v", err)
		}
	})
}

// TestUserColorRepository_GetUserColors tests getting user's colors
func TestUserColorRepository_GetUserColors(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewUserColorRepository(db)

	t.Run("user with multiple colors", func(t *testing.T) {
		userID := testutil.SeedTestUser(t, db, "user5@test.com", "Test User 5", "green")
		color1ID := testutil.SeedTestColorCategory(t, db, "color-a", "#aaaaaa", 50)
		color2ID := testutil.SeedTestColorCategory(t, db, "color-b", "#bbbbbb", 60)
		color3ID := testutil.SeedTestColorCategory(t, db, "color-c", "#cccccc", 70)

		testutil.SeedTestUserColor(t, db, userID, color1ID)
		testutil.SeedTestUserColor(t, db, userID, color2ID)
		testutil.SeedTestUserColor(t, db, userID, color3ID)

		colors, err := repo.GetUserColors(userID)
		if err != nil {
			t.Fatalf("GetUserColors() failed: %v", err)
		}

		if len(colors) != 3 {
			t.Errorf("Expected 3 colors, got %d", len(colors))
		}
	})

	t.Run("user with no colors", func(t *testing.T) {
		userID := testutil.SeedTestUser(t, db, "user6@test.com", "Test User 6", "green")

		colors, err := repo.GetUserColors(userID)
		if err != nil {
			t.Fatalf("GetUserColors() failed: %v", err)
		}

		if len(colors) != 0 {
			t.Errorf("Expected 0 colors, got %d", len(colors))
		}
	})
}

// TestUserColorRepository_HasColor tests checking if user has color
func TestUserColorRepository_HasColor(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewUserColorRepository(db)

	userID := testutil.SeedTestUser(t, db, "user7@test.com", "Test User 7", "green")
	color1ID := testutil.SeedTestColorCategory(t, db, "has-this", "#111111", 80)
	color2ID := testutil.SeedTestColorCategory(t, db, "not-this", "#222222", 90)

	testutil.SeedTestUserColor(t, db, userID, color1ID)

	t.Run("user has color", func(t *testing.T) {
		hasColor, err := repo.HasColor(userID, color1ID)
		if err != nil {
			t.Fatalf("HasColor() failed: %v", err)
		}

		if !hasColor {
			t.Error("Expected true, user has this color")
		}
	})

	t.Run("user does not have color", func(t *testing.T) {
		hasColor, err := repo.HasColor(userID, color2ID)
		if err != nil {
			t.Fatalf("HasColor() failed: %v", err)
		}

		if hasColor {
			t.Error("Expected false, user does not have this color")
		}
	})
}

// TestUserColorRepository_GetUserColorIDs tests getting user's color IDs
func TestUserColorRepository_GetUserColorIDs(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewUserColorRepository(db)

	t.Run("user with colors", func(t *testing.T) {
		userID := testutil.SeedTestUser(t, db, "user8@test.com", "Test User 8", "green")
		color1ID := testutil.SeedTestColorCategory(t, db, "id-color-1", "#111111", 100)
		color2ID := testutil.SeedTestColorCategory(t, db, "id-color-2", "#222222", 110)

		testutil.SeedTestUserColor(t, db, userID, color1ID)
		testutil.SeedTestUserColor(t, db, userID, color2ID)

		colorIDs, err := repo.GetUserColorIDs(userID)
		if err != nil {
			t.Fatalf("GetUserColorIDs() failed: %v", err)
		}

		if len(colorIDs) != 2 {
			t.Errorf("Expected 2 color IDs, got %d", len(colorIDs))
		}

		// Verify both IDs are present
		found1, found2 := false, false
		for _, id := range colorIDs {
			if id == color1ID {
				found1 = true
			}
			if id == color2ID {
				found2 = true
			}
		}
		if !found1 || !found2 {
			t.Error("Expected both color IDs to be present")
		}
	})

	t.Run("user with no colors returns empty slice", func(t *testing.T) {
		userID := testutil.SeedTestUser(t, db, "user9@test.com", "Test User 9", "green")

		colorIDs, err := repo.GetUserColorIDs(userID)
		if err != nil {
			t.Fatalf("GetUserColorIDs() failed: %v", err)
		}

		if colorIDs == nil {
			t.Error("Expected empty slice, not nil")
		}

		if len(colorIDs) != 0 {
			t.Errorf("Expected 0 color IDs, got %d", len(colorIDs))
		}
	})
}
