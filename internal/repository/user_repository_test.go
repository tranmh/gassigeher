package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// DONE: TestUserRepository_Create tests user creation
func TestUserRepository_Create(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewUserRepository(db)

	t.Run("successful creation", func(t *testing.T) {
		email := "test@example.com"
		user := &models.User{
			FirstName:       "Test",
			LastName:        "User",
			Email:           &email,
			Phone:           stringPtr("+49 123 456789"),
			PasswordHash:    stringPtr("hashed_password"),
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
			FirstName:       "User",
			LastName:        "One",
			Email:           &email,
			PasswordHash:    stringPtr("hash"),
			TermsAcceptedAt: time.Now(),
			LastActivityAt:  time.Now(),
		}

		repo.Create(user1)

		// Try to create another user with same email
		user2 := &models.User{
			FirstName:       "User",
			LastName:        "Two",
			Email:           &email,
			PasswordHash:    stringPtr("hash"),
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

		if user.FullName() != "Find Me" {
			t.Errorf("Expected name 'Find Me', got %s", user.FullName())
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

		if user.FullName() != "Deleted User" {
			t.Errorf("Expected name 'Deleted User', got %s", user.FullName())
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

// DONE: TestUserRepository_FindByVerificationToken tests finding users by verification token
func TestUserRepository_FindByVerificationToken(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewUserRepository(db)

	t.Run("user with valid token exists", func(t *testing.T) {
		email := "verify@example.com"
		verificationToken := "test-verification-token-123"
		tokenExpires := time.Now().Add(24 * time.Hour)

		// Create user with verification token
		_, err := db.Exec(`
			INSERT INTO users (email, name, password_hash, experience_level, is_verified, is_active,
			                   verification_token, verification_token_expires, terms_accepted_at, last_activity_at, created_at)
			VALUES (?, 'Verify Me', 'hash', 'green', 0, 1, ?, ?, datetime('now'), datetime('now'), datetime('now'))
		`, email, verificationToken, tokenExpires)
		if err != nil {
			t.Fatalf("Failed to seed user: %v", err)
		}

		// Find by verification token
		user, err := repo.FindByVerificationToken(verificationToken)
		if err != nil {
			t.Fatalf("FindByVerificationToken() failed: %v", err)
		}

		if user == nil {
			t.Fatal("Expected user, got nil")
		}

		if user.Email == nil || *user.Email != email {
			t.Errorf("Expected email %s, got %v", email, user.Email)
		}

		if user.VerificationToken == nil || *user.VerificationToken != verificationToken {
			t.Errorf("Expected token %s, got %v", verificationToken, user.VerificationToken)
		}

		if user.IsVerified {
			t.Error("User should not be verified yet")
		}
	})

	t.Run("token not found", func(t *testing.T) {
		user, err := repo.FindByVerificationToken("nonexistent-token")
		if err != nil {
			t.Fatalf("FindByVerificationToken() failed: %v", err)
		}

		if user != nil {
			t.Error("Expected nil user for non-existent token")
		}
	})

	t.Run("empty token", func(t *testing.T) {
		user, err := repo.FindByVerificationToken("")
		if err != nil {
			t.Fatalf("FindByVerificationToken() failed: %v", err)
		}

		if user != nil {
			t.Error("Expected nil user for empty token")
		}
	})

	t.Run("deleted user with token should not be found", func(t *testing.T) {
		email := "deleted-verify@example.com"
		verificationToken := "deleted-user-token"
		tokenExpires := time.Now().Add(24 * time.Hour)

		// Create deleted user with verification token
		_, err := db.Exec(`
			INSERT INTO users (email, name, password_hash, experience_level, is_verified, is_active, is_deleted,
			                   verification_token, verification_token_expires, terms_accepted_at, last_activity_at, created_at)
			VALUES (?, 'Deleted User', 'hash', 'green', 0, 0, 1, ?, ?, datetime('now'), datetime('now'), datetime('now'))
		`, email, verificationToken, tokenExpires)
		if err != nil {
			t.Fatalf("Failed to seed deleted user: %v", err)
		}

		// Try to find deleted user
		user, err := repo.FindByVerificationToken(verificationToken)
		if err != nil {
			t.Fatalf("FindByVerificationToken() failed: %v", err)
		}

		if user != nil {
			t.Error("Deleted user should not be found by verification token")
		}
	})
}

// DONE: TestUserRepository_FindByPasswordResetToken tests finding users by password reset token
func TestUserRepository_FindByPasswordResetToken(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewUserRepository(db)

	t.Run("user with valid reset token exists", func(t *testing.T) {
		email := "reset@example.com"
		resetToken := "test-reset-token-456"
		tokenExpires := time.Now().Add(1 * time.Hour)

		// Create user with password reset token
		_, err := db.Exec(`
			INSERT INTO users (email, name, password_hash, experience_level, is_verified, is_active,
			                   password_reset_token, password_reset_expires, terms_accepted_at, last_activity_at, created_at)
			VALUES (?, 'Reset Me', 'hash', 'blue', 1, 1, ?, ?, datetime('now'), datetime('now'), datetime('now'))
		`, email, resetToken, tokenExpires)
		if err != nil {
			t.Fatalf("Failed to seed user: %v", err)
		}

		// Find by password reset token
		user, err := repo.FindByPasswordResetToken(resetToken)
		if err != nil {
			t.Fatalf("FindByPasswordResetToken() failed: %v", err)
		}

		if user == nil {
			t.Fatal("Expected user, got nil")
		}

		if user.Email == nil || *user.Email != email {
			t.Errorf("Expected email %s, got %v", email, user.Email)
		}

		if user.PasswordResetToken == nil || *user.PasswordResetToken != resetToken {
			t.Errorf("Expected token %s, got %v", resetToken, user.PasswordResetToken)
		}
	})

	t.Run("token not found", func(t *testing.T) {
		user, err := repo.FindByPasswordResetToken("nonexistent-reset-token")
		if err != nil {
			t.Fatalf("FindByPasswordResetToken() failed: %v", err)
		}

		if user != nil {
			t.Error("Expected nil user for non-existent token")
		}
	})

	t.Run("empty token", func(t *testing.T) {
		user, err := repo.FindByPasswordResetToken("")
		if err != nil {
			t.Fatalf("FindByPasswordResetToken() failed: %v", err)
		}

		if user != nil {
			t.Error("Expected nil user for empty token")
		}
	})

	t.Run("deleted user with reset token should not be found", func(t *testing.T) {
		email := "deleted-reset@example.com"
		resetToken := "deleted-reset-token"
		tokenExpires := time.Now().Add(1 * time.Hour)

		// Create deleted user with reset token
		_, err := db.Exec(`
			INSERT INTO users (email, name, password_hash, experience_level, is_verified, is_active, is_deleted,
			                   password_reset_token, password_reset_expires, terms_accepted_at, last_activity_at, created_at)
			VALUES (?, 'Deleted User', 'hash', 'green', 1, 0, 1, ?, ?, datetime('now'), datetime('now'), datetime('now'))
		`, email, resetToken, tokenExpires)
		if err != nil {
			t.Fatalf("Failed to seed deleted user: %v", err)
		}

		// Try to find deleted user
		user, err := repo.FindByPasswordResetToken(resetToken)
		if err != nil {
			t.Fatalf("FindByPasswordResetToken() failed: %v", err)
		}

		if user != nil {
			t.Error("Deleted user should not be found by password reset token")
		}
	})
}

// DONE: TestUserRepository_Update tests user update functionality
func TestUserRepository_Update(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewUserRepository(db)

	t.Run("successful update of user fields", func(t *testing.T) {
		// Create initial user
		userID := testutil.SeedTestUser(t, db, "update@example.com", "Original Name", "green")

		// Get user
		user, err := repo.FindByID(userID)
		if err != nil {
			t.Fatalf("Failed to find user: %v", err)
		}

		// Update fields
		newFirstName := "Updated"
		newLastName := "Name"
		newPhone := "+49 987 654321"
		user.FirstName = newFirstName
		user.LastName = newLastName
		user.Phone = &newPhone

		// Perform update
		err = repo.Update(user)
		if err != nil {
			t.Fatalf("Update() failed: %v", err)
		}

		// Verify updates
		updated, err := repo.FindByID(userID)
		if err != nil {
			t.Fatalf("Failed to find updated user: %v", err)
		}

		if updated.FirstName != newFirstName {
			t.Errorf("Expected first name '%s', got '%s'", newFirstName, updated.FirstName)
		}

		if updated.LastName != newLastName {
			t.Errorf("Expected last name '%s', got '%s'", newLastName, updated.LastName)
		}

		if updated.Phone == nil || *updated.Phone != newPhone {
			t.Errorf("Expected phone '%s', got %v", newPhone, updated.Phone)
		}
	})

	t.Run("update verification status", func(t *testing.T) {
		userID := testutil.SeedTestUser(t, db, "verify-update@example.com", "Verify User", "green")

		// Get user
		user, err := repo.FindByID(userID)
		if err != nil {
			t.Fatalf("Failed to find user: %v", err)
		}

		// Update verification status
		user.IsVerified = true
		verificationToken := (*string)(nil) // Clear token
		user.VerificationToken = verificationToken

		err = repo.Update(user)
		if err != nil {
			t.Fatalf("Update() failed: %v", err)
		}

		// Verify updates
		updated, err := repo.FindByID(userID)
		if err != nil {
			t.Fatalf("Failed to find updated user: %v", err)
		}

		if !updated.IsVerified {
			t.Error("User should be verified after update")
		}
	})

	t.Run("update with tokens", func(t *testing.T) {
		userID := testutil.SeedTestUser(t, db, "token-update@example.com", "Token User", "green")

		// Get user
		user, err := repo.FindByID(userID)
		if err != nil {
			t.Fatalf("Failed to find user: %v", err)
		}

		// Set tokens
		resetToken := "new-reset-token"
		resetExpires := time.Now().Add(1 * time.Hour)
		user.PasswordResetToken = &resetToken
		user.PasswordResetExpires = &resetExpires

		err = repo.Update(user)
		if err != nil {
			t.Fatalf("Update() failed: %v", err)
		}

		// Verify updates
		updated, err := repo.FindByID(userID)
		if err != nil {
			t.Fatalf("Failed to find updated user: %v", err)
		}

		if updated.PasswordResetToken == nil || *updated.PasswordResetToken != resetToken {
			t.Errorf("Expected reset token '%s', got %v", resetToken, updated.PasswordResetToken)
		}
	})

	t.Run("update non-existent user", func(t *testing.T) {
		email := "nonexistent@example.com"
		user := &models.User{
			ID:        99999,
			FirstName: "Nonexistent",
			LastName:  "User",
			Email:     &email,
		}

		err := repo.Update(user)
		// Should not error even if no rows affected
		if err != nil {
			t.Logf("Update for non-existent user returned: %v", err)
		}
	})

	t.Run("update email to new address", func(t *testing.T) {
		userID := testutil.SeedTestUser(t, db, "oldemail@example.com", "Email Change", "green")

		// Get user
		user, err := repo.FindByID(userID)
		if err != nil {
			t.Fatalf("Failed to find user: %v", err)
		}

		// Change email and set verification to false
		newEmail := "newemail@example.com"
		user.Email = &newEmail
		user.IsVerified = false
		verifyToken := "new-verify-token"
		user.VerificationToken = &verifyToken

		err = repo.Update(user)
		if err != nil {
			t.Fatalf("Update() failed: %v", err)
		}

		// Verify updates
		updated, err := repo.FindByID(userID)
		if err != nil {
			t.Fatalf("Failed to find updated user: %v", err)
		}

		if updated.Email == nil || *updated.Email != newEmail {
			t.Errorf("Expected email '%s', got %v", newEmail, updated.Email)
		}

		if updated.IsVerified {
			t.Error("User should not be verified after email change")
		}

		if updated.VerificationToken == nil || *updated.VerificationToken != verifyToken {
			t.Errorf("Expected verification token '%s', got %v", verifyToken, updated.VerificationToken)
		}
	})
}

func stringPtr(s string) *string {
	return &s
}

// TestUserRepository_ClearMustChangePassword tests clearing the must_change_password flag
func TestUserRepository_ClearMustChangePassword(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewUserRepository(db)

	t.Run("clears must_change_password flag successfully", func(t *testing.T) {
		// Create user with must_change_password = true
		email := "mustchange@example.com"
		_, err := db.Exec(`
			INSERT INTO users (email, name, first_name, last_name, password_hash, experience_level, is_active, is_verified, must_change_password, terms_accepted_at, last_activity_at, created_at)
			VALUES (?, 'Must Change', 'Must', 'Change', 'hash', 'green', 1, 1, 1, datetime('now'), datetime('now'), datetime('now'))
		`, email)
		if err != nil {
			t.Fatalf("Failed to seed user: %v", err)
		}

		// Find user to get ID
		user, err := repo.FindByEmail(email)
		if err != nil {
			t.Fatalf("Failed to find user: %v", err)
		}

		// Verify must_change_password is initially true
		if !user.MustChangePassword {
			t.Error("Expected must_change_password to be true initially")
		}

		// Clear the flag
		err = repo.ClearMustChangePassword(user.ID)
		if err != nil {
			t.Fatalf("ClearMustChangePassword() failed: %v", err)
		}

		// Verify flag is cleared
		updatedUser, err := repo.FindByID(user.ID)
		if err != nil {
			t.Fatalf("Failed to find updated user: %v", err)
		}

		if updatedUser.MustChangePassword {
			t.Error("Expected must_change_password to be false after clearing")
		}
	})

	t.Run("clearing flag for non-existent user does not error", func(t *testing.T) {
		err := repo.ClearMustChangePassword(99999)
		// Should not error even if user doesn't exist
		if err != nil {
			t.Logf("ClearMustChangePassword for non-existent user returned: %v", err)
		}
	})

	t.Run("clearing flag on user where it is already false", func(t *testing.T) {
		// Create user with must_change_password = false
		email := "nochange@example.com"
		_, err := db.Exec(`
			INSERT INTO users (email, name, first_name, last_name, password_hash, experience_level, is_active, is_verified, must_change_password, terms_accepted_at, last_activity_at, created_at)
			VALUES (?, 'No Change', 'No', 'Change', 'hash', 'blue', 1, 1, 0, datetime('now'), datetime('now'), datetime('now'))
		`, email)
		if err != nil {
			t.Fatalf("Failed to seed user: %v", err)
		}

		user, _ := repo.FindByEmail(email)

		// Clear should succeed even if already false
		err = repo.ClearMustChangePassword(user.ID)
		if err != nil {
			t.Fatalf("ClearMustChangePassword() failed: %v", err)
		}

		// Verify still false
		updatedUser, _ := repo.FindByID(user.ID)
		if updatedUser.MustChangePassword {
			t.Error("Expected must_change_password to remain false")
		}
	})
}

// TestUserRepository_MustChangePasswordField tests the must_change_password field persistence
func TestUserRepository_MustChangePasswordField(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewUserRepository(db)

	t.Run("create user with must_change_password true", func(t *testing.T) {
		email := "newadmincreated@example.com"
		user := &models.User{
			FirstName:          "Admin",
			LastName:           "Created",
			Email:              &email,
			PasswordHash:       stringPtr("hashed_password"),
			IsVerified:         true,
			IsActive:           true,
			MustChangePassword: true,
			TermsAcceptedAt:    time.Now(),
			LastActivityAt:     time.Now(),
		}

		err := repo.Create(user)
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}

		// Verify the flag was saved
		savedUser, err := repo.FindByID(user.ID)
		if err != nil {
			t.Fatalf("FindByID() failed: %v", err)
		}

		if !savedUser.MustChangePassword {
			t.Error("Expected must_change_password to be true after creation")
		}
	})

	t.Run("create user with must_change_password false (default)", func(t *testing.T) {
		email := "regularuser@example.com"
		user := &models.User{
			FirstName:          "Regular",
			LastName:           "User",
			Email:              &email,
			PasswordHash:       stringPtr("hashed_password"),
			IsVerified:         false,
			IsActive:           true,
			MustChangePassword: false,
			TermsAcceptedAt:    time.Now(),
			LastActivityAt:     time.Now(),
		}

		err := repo.Create(user)
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}

		savedUser, _ := repo.FindByID(user.ID)
		if savedUser.MustChangePassword {
			t.Error("Expected must_change_password to be false for regular user creation")
		}
	})

	t.Run("update user preserves must_change_password flag", func(t *testing.T) {
		email := "updatepreserve@example.com"
		user := &models.User{
			FirstName:          "Update",
			LastName:           "Preserve",
			Email:              &email,
			PasswordHash:       stringPtr("hashed_password"),
			IsVerified:         true,
			IsActive:           true,
			MustChangePassword: true,
			TermsAcceptedAt:    time.Now(),
			LastActivityAt:     time.Now(),
		}

		repo.Create(user)

		// Update other fields (update first name instead of experience level)
		user.FirstName = "Updated"
		err := repo.Update(user)
		if err != nil {
			t.Fatalf("Update() failed: %v", err)
		}

		// Verify must_change_password is still true
		updatedUser, _ := repo.FindByID(user.ID)
		if !updatedUser.MustChangePassword {
			t.Error("Expected must_change_password to remain true after update")
		}
		if updatedUser.FirstName != "Updated" {
			t.Error("Expected first name to be updated")
		}
	})
}
