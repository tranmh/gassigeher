package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tranm/gassigeher/internal/config"
	"github.com/tranm/gassigeher/internal/models"
	"github.com/tranm/gassigeher/internal/repository"
	"github.com/tranm/gassigeher/internal/services"
	"github.com/tranm/gassigeher/internal/testutil"
)

// DONE: TestUserHandler_GetMe tests getting current user profile
func TestUserHandler_GetMe(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewUserHandler(db, cfg)

	// Create test user
	authService := services.NewAuthService(cfg.JWTSecret, cfg.JWTExpirationHours)
	hash, _ := authService.HashPassword("Test1234")
	userRepo := repository.NewUserRepository(db)

	email := "getme@example.com"
	user := &models.User{
		Name:            "Get Me User",
		Email:           &email,
		PasswordHash:    &hash,
		ExperienceLevel: "blue",
		IsVerified:      true,
		IsActive:        true,
		TermsAcceptedAt: time.Now(),
		LastActivityAt:  time.Now(),
	}
	userRepo.Create(user)

	t.Run("successful get current user", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users/me", nil)
		ctx := contextWithUser(req.Context(), user.ID, *user.Email, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetMe(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response models.User
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response.ID != user.ID {
			t.Errorf("Expected user ID %d, got %d", user.ID, response.ID)
		}

		if response.Name != "Get Me User" {
			t.Errorf("Expected name 'Get Me User', got %s", response.Name)
		}

		if response.ExperienceLevel != "blue" {
			t.Errorf("Expected level 'blue', got %s", response.ExperienceLevel)
		}
	})

	t.Run("missing user context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users/me", nil)
		// No context set

		rec := httptest.NewRecorder()
		handler.GetMe(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401 when context missing, got %d", rec.Code)
		}
	})

	t.Run("non-existent user in context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users/me", nil)
		ctx := contextWithUser(req.Context(), 99999, "nonexistent@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetMe(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})
}

// DONE: TestUserHandler_UpdateMe tests updating current user profile
func TestUserHandler_UpdateMe(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewUserHandler(db, cfg)

	// Create test user
	authService := services.NewAuthService(cfg.JWTSecret, cfg.JWTExpirationHours)
	hash, _ := authService.HashPassword("Test1234")
	userRepo := repository.NewUserRepository(db)

	email := "update@example.com"
	phone := "+49 123 456789"
	user := &models.User{
		Name:            "Original Name",
		Email:           &email,
		Phone:           &phone,
		PasswordHash:    &hash,
		ExperienceLevel: "green",
		IsVerified:      true,
		IsActive:        true,
		TermsAcceptedAt: time.Now(),
		LastActivityAt:  time.Now(),
	}
	userRepo.Create(user)

	t.Run("update name only", func(t *testing.T) {
		newName := "Updated Name"
		reqBody := map[string]interface{}{
			"name": newName,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/users/me", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), user.ID, *user.Email, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateMe(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify update
		updatedUser, _ := userRepo.FindByID(user.ID)
		if updatedUser.Name != newName {
			t.Errorf("Expected name '%s', got '%s'", newName, updatedUser.Name)
		}
	})

	t.Run("update phone only", func(t *testing.T) {
		newPhone := "+49 987 654321"
		reqBody := map[string]interface{}{
			"phone": newPhone,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/users/me", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), user.ID, *user.Email, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateMe(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		// Verify update
		updatedUser, _ := userRepo.FindByID(user.ID)
		if updatedUser.Phone == nil || *updatedUser.Phone != newPhone {
			t.Errorf("Expected phone '%s', got %v", newPhone, updatedUser.Phone)
		}
	})

	t.Run("update email triggers verification", func(t *testing.T) {
		newEmail := "newemail@example.com"
		reqBody := map[string]interface{}{
			"email": newEmail,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/users/me", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), user.ID, *user.Email, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateMe(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		// Verify email updated and user unverified
		updatedUser, _ := userRepo.FindByID(user.ID)
		if updatedUser.Email == nil || *updatedUser.Email != newEmail {
			t.Errorf("Expected email '%s', got %v", newEmail, updatedUser.Email)
		}

		if updatedUser.IsVerified {
			t.Error("User should be unverified after email change")
		}

		if updatedUser.VerificationToken == nil || *updatedUser.VerificationToken == "" {
			t.Error("Verification token should be set after email change")
		}
	})

	t.Run("update with duplicate email", func(t *testing.T) {
		// Create another user
		existingEmail := "existing@example.com"
		testutil.SeedTestUser(t, db, existingEmail, "Existing User", "green")

		reqBody := map[string]interface{}{
			"email": existingEmail,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/users/me", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), user.ID, *user.Email, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateMe(rec, req)

		if rec.Code != http.StatusConflict {
			t.Errorf("Expected status 409 for duplicate email, got %d", rec.Code)
		}
	})
}

// DONE: TestUserHandler_DeleteAccount tests GDPR-compliant account deletion
func TestUserHandler_DeleteAccount(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewUserHandler(db, cfg)

	// Create test user
	authService := services.NewAuthService(cfg.JWTSecret, cfg.JWTExpirationHours)
	hash, _ := authService.HashPassword("Test1234")
	userRepo := repository.NewUserRepository(db)

	email := "delete@example.com"
	user := &models.User{
		Name:            "Delete Me",
		Email:           &email,
		PasswordHash:    &hash,
		ExperienceLevel: "green",
		IsVerified:      true,
		IsActive:        true,
		TermsAcceptedAt: time.Now(),
		LastActivityAt:  time.Now(),
	}
	userRepo.Create(user)

	t.Run("successful account deletion", func(t *testing.T) {
		reqBody := map[string]string{
			"password": "Test1234",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("DELETE", "/api/users/me", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), user.ID, *user.Email, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DeleteAccount(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify user is anonymized
		deletedUser, err := userRepo.FindByID(user.ID)
		if err != nil {
			t.Fatalf("User should still exist but be anonymized: %v", err)
		}

		if deletedUser.Name != "Deleted User" {
			t.Errorf("Expected name 'Deleted User', got %s", deletedUser.Name)
		}

		if deletedUser.Email != nil {
			t.Error("Email should be NULL after deletion")
		}

		if !deletedUser.IsDeleted {
			t.Error("IsDeleted flag should be true")
		}

		if deletedUser.AnonymousID == nil {
			t.Error("AnonymousID should be set")
		}
	})

	t.Run("missing user context", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/users/me", nil)
		// No context

		rec := httptest.NewRecorder()
		handler.DeleteAccount(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rec.Code)
		}
	})
}
