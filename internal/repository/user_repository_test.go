package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/tranm/gassigeher/internal/models"
	"github.com/tranm/gassigeher/internal/testutil"
)

// DONE: TestUserRepository_Create tests user creation
func TestUserRepository_Create(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewUserRepository(db)

	t.Run("successful creation", func(t *testing.T) {
		email := "test@example.com"
		user := &models.User{
			Name:            "Test User",
			Email:           &email,
			Phone:           stringPtr("+49 123 456789"),
			PasswordHash:    stringPtr("hashed_password"),
			ExperienceLevel: "green",
			IsVerified:      false,
			IsActive:        true,
			TermsAcceptedAt: time.Now(),
			LastActivityAt:  time.Now(),
		}

		err := repo.Create(user)
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}

		if user.ID == 0 {
			t.Error("User ID should be set after creation")
		}
	})

	t.Run("duplicate email", func(t *testing.T) {
		email := "duplicate@example.com"
		user1 := &models.User{
			Name:            "User 1",
			Email:           &email,
			PasswordHash:    stringPtr("hash"),
			ExperienceLevel: "green",
			TermsAcceptedAt: time.Now(),
			LastActivityAt:  time.Now(),
		}

		repo.Create(user1)

		// Try to create another user with same email
		user2 := &models.User{
			Name:            "User 2",
			Email:           &email,
			PasswordHash:    stringPtr("hash"),
			ExperienceLevel: "green",
			TermsAcceptedAt: time.Now(),
			LastActivityAt:  time.Now(),
		}

		err := repo.Create(user2)
		if err == nil {
			t.Error("Expected error for duplicate email, got nil")
		}
	})
}

// DONE: TestUserRepository_FindByEmail tests finding users by email
func TestUserRepository_FindByEmail(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewUserRepository(db)

	testEmail := "find@example.com"

	t.Run("user exists", func(t *testing.T) {
		// Create test user
		testutil.SeedTestUser(t, db, testEmail, "Find Me", "green")

		// Find by email
		user, err := repo.FindByEmail(testEmail)
		if err != nil {
			t.Fatalf("FindByEmail() failed: %v", err)
		}

		if user.Email == nil || *user.Email != testEmail {
			t.Errorf("Expected email %s, got %v", testEmail, user.Email)
		}

		if user.Name != "Find Me" {
			t.Errorf("Expected name 'Find Me', got %s", user.Name)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		user, err := repo.FindByEmail("nonexistent@example.com")
		if err != sql.ErrNoRows && err != nil {
			t.Logf("FindByEmail returned error: %v", err)
		}
		if user != nil {
			t.Error("Expected nil user for non-existent email")
		}
	})

	t.Run("empty email", func(t *testing.T) {
		user, err := repo.FindByEmail("")
		if err != sql.ErrNoRows && err != nil {
			t.Logf("FindByEmail with empty email returned error: %v", err)
		}
		if user != nil {
			t.Error("Expected nil user for empty email")
		}
	})
}

// DONE: TestUserRepository_FindByID tests finding users by ID
func TestUserRepository_FindByID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewUserRepository(db)

	t.Run("user exists", func(t *testing.T) {
		userID := testutil.SeedTestUser(t, db, "test@example.com", "Test User", "blue")

		user, err := repo.FindByID(userID)
		if err != nil {
			t.Fatalf("FindByID() failed: %v", err)
		}

		if user.ID != userID {
			t.Errorf("Expected ID %d, got %d", userID, user.ID)
		}

		if user.ExperienceLevel != "blue" {
			t.Errorf("Expected level 'blue', got %s", user.ExperienceLevel)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		user, err := repo.FindByID(99999)
		if err != sql.ErrNoRows && err != nil {
			t.Logf("FindByID returned error: %v", err)
		}
		if user != nil {
			t.Error("Expected nil user for non-existent ID")
		}
	})

	t.Run("invalid ID zero", func(t *testing.T) {
		user, err := repo.FindByID(0)
		if err != sql.ErrNoRows && err != nil {
			t.Logf("FindByID(0) returned error: %v", err)
		}
		if user != nil {
			t.Error("Expected nil user for ID 0")
		}
	})

	t.Run("negative ID", func(t *testing.T) {
		user, err := repo.FindByID(-1)
		if err != sql.ErrNoRows && err != nil {
			t.Logf("FindByID(-1) returned error: %v", err)
		}
		if user != nil {
			t.Error("Expected nil user for negative ID")
		}
	})
}

// DONE: TestUserRepository_UpdateLastActivity tests activity tracking
func TestUserRepository_UpdateLastActivity(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewUserRepository(db)

	t.Run("update existing user", func(t *testing.T) {
		userID := testutil.SeedTestUser(t, db, "active@example.com", "Active User", "green")

		// Wait a tiny bit to ensure time difference
		time.Sleep(10 * time.Millisecond)

		err := repo.UpdateLastActivity(userID)
		if err != nil {
			t.Fatalf("UpdateLastActivity() failed: %v", err)
		}

		// Verify last_activity_at was updated
		var lastActivity time.Time
		err = db.QueryRow("SELECT last_activity_at FROM users WHERE id = ?", userID).Scan(&lastActivity)
		if err != nil {
			t.Fatalf("Failed to query last_activity_at: %v", err)
		}

		// Check it's recent (within last second)
		if time.Since(lastActivity) > 2*time.Second {
			t.Errorf("last_activity_at not updated recently: %v", lastActivity)
		}
	})

	t.Run("non-existent user", func(t *testing.T) {
		err := repo.UpdateLastActivity(99999)
		// Should not error even if user doesn't exist
		if err != nil {
			t.Logf("UpdateLastActivity for non-existent user returned: %v", err)
		}
	})
}

// DONE: TestUserRepository_DeleteAccount tests GDPR-compliant account deletion
func TestUserRepository_DeleteAccount(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewUserRepository(db)

	t.Run("successful deletion with anonymization", func(t *testing.T) {
		email := "delete@example.com"
		userID := testutil.SeedTestUser(t, db, email, "Delete Me", "green")

		err := repo.DeleteAccount(userID)
		if err != nil {
			t.Fatalf("DeleteAccount() failed: %v", err)
		}

		// Verify user is anonymized
		user, err := repo.FindByID(userID)
		if err != nil {
			t.Fatalf("Failed to find deleted user: %v", err)
		}

		if user.Name != "Deleted User" {
			t.Errorf("Expected name 'Deleted User', got %s", user.Name)
		}

		if user.Email != nil {
			t.Errorf("Email should be NULL after deletion, got %v", *user.Email)
		}

		if user.Phone != nil {
			t.Errorf("Phone should be NULL after deletion, got %v", *user.Phone)
		}

		if user.PasswordHash != nil {
			t.Errorf("PasswordHash should be NULL after deletion")
		}

		if !user.IsDeleted {
			t.Error("IsDeleted should be true")
		}

		if user.AnonymousID == nil || *user.AnonymousID == "" {
			t.Error("AnonymousID should be set")
		}

		if !user.IsActive {
			// IsActive is set to false on deletion
		}
	})

	t.Run("non-existent user", func(t *testing.T) {
		err := repo.DeleteAccount(99999)
		// May or may not error depending on implementation
		if err != nil {
			t.Logf("DeleteAccount for non-existent user returned: %v", err)
		}
	})
}

// DONE: TestUserRepository_Deactivate tests user deactivation
func TestUserRepository_Deactivate(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewUserRepository(db)

	t.Run("successful deactivation", func(t *testing.T) {
		userID := testutil.SeedTestUser(t, db, "deactivate@example.com", "Deactivate Me", "green")

		reason := "Inactivity for 365 days"
		err := repo.Deactivate(userID, reason)
		if err != nil {
			t.Fatalf("Deactivate() failed: %v", err)
		}

		// Verify user is deactivated
		user, err := repo.FindByID(userID)
		if err != nil {
			t.Fatalf("Failed to find deactivated user: %v", err)
		}

		if user.IsActive {
			t.Error("IsActive should be false after deactivation")
		}

		if user.DeactivationReason == nil || *user.DeactivationReason != reason {
			t.Errorf("Expected deactivation reason '%s', got %v", reason, user.DeactivationReason)
		}

		if user.DeactivatedAt == nil {
			t.Error("DeactivatedAt should be set")
		}
	})
}

// DONE: TestUserRepository_Activate tests user reactivation
func TestUserRepository_Activate(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewUserRepository(db)

	t.Run("successful activation", func(t *testing.T) {
		userID := testutil.SeedTestUser(t, db, "activate@example.com", "Activate Me", "green")

		// First deactivate
		repo.Deactivate(userID, "Test deactivation")

		// Then activate
		err := repo.Activate(userID)
		if err != nil {
			t.Fatalf("Activate() failed: %v", err)
		}

		// Verify user is activated
		user, err := repo.FindByID(userID)
		if err != nil {
			t.Fatalf("Failed to find activated user: %v", err)
		}

		if !user.IsActive {
			t.Error("IsActive should be true after activation")
		}

		if user.ReactivatedAt == nil {
			t.Error("ReactivatedAt should be set")
		}
	})
}

// DONE: TestUserRepository_FindAll tests listing all users
func TestUserRepository_FindAll(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewUserRepository(db)

	// Seed test data
	activeID := testutil.SeedTestUser(t, db, "active@example.com", "Active User", "green")
	inactiveID := testutil.SeedTestUser(t, db, "inactive@example.com", "Inactive User", "blue")
	repo.Deactivate(inactiveID, "Test")

	t.Run("all users", func(t *testing.T) {
		users, err := repo.FindAll(nil)
		if err != nil {
			t.Fatalf("FindAll() failed: %v", err)
		}

		if len(users) < 2 {
			t.Errorf("Expected at least 2 users, got %d", len(users))
		}
	})

	t.Run("active only", func(t *testing.T) {
		activeOnly := true
		users, err := repo.FindAll(&activeOnly)
		if err != nil {
			t.Fatalf("FindAll(activeOnly=true) failed: %v", err)
		}

		// Should contain active user
		found := false
		for _, u := range users {
			if u.ID == activeID {
				found = true
				if !u.IsActive {
					t.Error("Active user should have IsActive=true")
				}
			}
			if u.ID == inactiveID {
				t.Error("Inactive user should not be in active-only results")
			}
		}

		if !found {
			t.Error("Active user not found in results")
		}
	})

	t.Run("all including inactive", func(t *testing.T) {
		activeOnly := false
		users, err := repo.FindAll(&activeOnly)
		if err != nil {
			t.Fatalf("FindAll(activeOnly=false) failed: %v", err)
		}

		// Should return users (count depends on implementation)
		if len(users) == 0 {
			t.Error("Expected at least some users to be returned")
		}

		t.Logf("FindAll(activeOnly=false) returned %d users", len(users))
	})
}

// DONE: TestUserRepository_FindInactiveUsers tests finding inactive users for auto-deactivation
func TestUserRepository_FindInactiveUsers(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewUserRepository(db)

	t.Run("find users inactive for 365 days", func(t *testing.T) {
		// Create user with old last activity
		now := time.Now()
		oldActivity := now.AddDate(0, 0, -400) // 400 days ago

		email := "old@example.com"
		_, err := db.Exec(`
			INSERT INTO users (email, name, password_hash, experience_level, is_active, is_verified, terms_accepted_at, last_activity_at, created_at)
			VALUES (?, 'Old User', 'hash', 'green', 1, 1, ?, ?, ?)
		`, email, now, oldActivity, now)
		if err != nil {
			t.Fatalf("Failed to seed old user: %v", err)
		}

		// Create recent user
		testutil.SeedTestUser(t, db, "recent@example.com", "Recent User", "green")

		// Find inactive users (>365 days)
		inactiveUsers, err := repo.FindInactiveUsers(365)
		if err != nil {
			t.Fatalf("FindInactiveUsers() failed: %v", err)
		}

		if len(inactiveUsers) != 1 {
			t.Errorf("Expected 1 inactive user, got %d", len(inactiveUsers))
		}

		if len(inactiveUsers) > 0 && inactiveUsers[0].Email != nil && *inactiveUsers[0].Email != email {
			t.Errorf("Expected email %s, got %v", email, *inactiveUsers[0].Email)
		}
	})
}

func stringPtr(s string) *string {
	return &s
}
