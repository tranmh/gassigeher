package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tranm/gassigeher/internal/config"
	"github.com/tranm/gassigeher/internal/middleware"
	"github.com/tranm/gassigeher/internal/models"
	"github.com/tranm/gassigeher/internal/repository"
	"github.com/tranm/gassigeher/internal/services"
	"github.com/tranm/gassigeher/internal/testutil"
)

// DONE: TestAuthHandler_Register tests user registration endpoint
func TestAuthHandler_Register(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewAuthHandler(db, cfg)

	t.Run("successful registration", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":             "Test User",
			"email":            "newuser@example.com",
			"phone":            "+49 123 456789",
			"password":         "Test1234",
			"confirm_password": "Test1234",
			"accept_terms":     true,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.Register(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["message"] == nil {
			t.Error("Expected message in response")
		}
	})

	t.Run("missing required fields", func(t *testing.T) {
		tests := []struct {
			name     string
			reqBody  map[string]interface{}
			expected string
		}{
			{
				name: "missing name",
				reqBody: map[string]interface{}{
					"email":            "test@example.com",
					"phone":            "+49 123",
					"password":         "Test1234",
					"confirm_password": "Test1234",
					"accept_terms":     true,
				},
				expected: "Name is required",
			},
			{
				name: "missing email",
				reqBody: map[string]interface{}{
					"name":             "Test",
					"phone":            "+49 123",
					"password":         "Test1234",
					"confirm_password": "Test1234",
					"accept_terms":     true,
				},
				expected: "Email is required",
			},
			{
				name: "missing phone",
				reqBody: map[string]interface{}{
					"name":             "Test",
					"email":            "test@example.com",
					"password":         "Test1234",
					"confirm_password": "Test1234",
					"accept_terms":     true,
				},
				expected: "Phone is required",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				body, _ := json.Marshal(tt.reqBody)
				req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")

				rec := httptest.NewRecorder()
				handler.Register(rec, req)

				if rec.Code != http.StatusBadRequest {
					t.Errorf("Expected status 400, got %d", rec.Code)
				}

				if !strings.Contains(rec.Body.String(), tt.expected) {
					t.Errorf("Expected error message to contain '%s', got: %s", tt.expected, rec.Body.String())
				}
			})
		}
	})

	t.Run("password mismatch", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":             "Test",
			"email":            "test@example.com",
			"phone":            "+49 123",
			"password":         "Test1234",
			"confirm_password": "Different1234",
			"accept_terms":     true,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.Register(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}

		if !strings.Contains(rec.Body.String(), "do not match") {
			t.Errorf("Expected password mismatch error, got: %s", rec.Body.String())
		}
	})

	t.Run("terms not accepted", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":             "Test",
			"email":            "test@example.com",
			"phone":            "+49 123",
			"password":         "Test1234",
			"confirm_password": "Test1234",
			"accept_terms":     false,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.Register(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	t.Run("weak password", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":             "Test",
			"email":            "test@example.com",
			"phone":            "+49 123",
			"password":         "weak",
			"confirm_password": "weak",
			"accept_terms":     true,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.Register(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for weak password, got %d", rec.Code)
		}
	})

	t.Run("duplicate email", func(t *testing.T) {
		// Create existing user
		testutil.SeedTestUser(t, db, "existing@example.com", "Existing User", "green")

		reqBody := map[string]interface{}{
			"name":             "New User",
			"email":            "existing@example.com",
			"phone":            "+49 123",
			"password":         "Test1234",
			"confirm_password": "Test1234",
			"accept_terms":     true,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.Register(rec, req)

		if rec.Code != http.StatusConflict {
			t.Errorf("Expected status 409, got %d", rec.Code)
		}
	})
}

// DONE: TestAuthHandler_Login tests user login endpoint
func TestAuthHandler_Login(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewAuthHandler(db, cfg)

	// Create test user
	authService := services.NewAuthService(cfg.JWTSecret, cfg.JWTExpirationHours)
	hash, _ := authService.HashPassword("Test1234")
	userRepo := repository.NewUserRepository(db)

	email := "test@example.com"
	user := &models.User{
		Name:            "Test User",
		Email:           &email,
		PasswordHash:    &hash,
		ExperienceLevel: "green",
		IsVerified:      true,
		IsActive:        true,
		TermsAcceptedAt: time.Now(),
		LastActivityAt:  time.Now(),
	}
	userRepo.Create(user)

	t.Run("successful login", func(t *testing.T) {
		reqBody := map[string]string{
			"email":    "test@example.com",
			"password": "Test1234",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.Login(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response models.LoginResponse
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response.Token == "" {
			t.Error("Expected token in response")
		}

		if response.User == nil {
			t.Error("Expected user in response")
		}
	})

	t.Run("invalid password", func(t *testing.T) {
		reqBody := map[string]string{
			"email":    "test@example.com",
			"password": "WrongPassword",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.Login(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rec.Code)
		}
	})

	t.Run("non-existent user", func(t *testing.T) {
		reqBody := map[string]string{
			"email":    "nonexistent@example.com",
			"password": "Test1234",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.Login(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rec.Code)
		}
	})

	t.Run("unverified user", func(t *testing.T) {
		// Create unverified user
		unverifiedEmail := "unverified@example.com"
		unverifiedUser := &models.User{
			Name:            "Unverified",
			Email:           &unverifiedEmail,
			PasswordHash:    &hash,
			ExperienceLevel: "green",
			IsVerified:      false,
			IsActive:        true,
			TermsAcceptedAt: time.Now(),
			LastActivityAt:  time.Now(),
		}
		userRepo.Create(unverifiedUser)

		reqBody := map[string]string{
			"email":    "unverified@example.com",
			"password": "Test1234",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.Login(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status 403 for unverified user, got %d", rec.Code)
		}
	})

	t.Run("inactive user", func(t *testing.T) {
		// Create inactive user
		inactiveEmail := "inactive@example.com"
		inactiveUser := &models.User{
			Name:            "Inactive",
			Email:           &inactiveEmail,
			PasswordHash:    &hash,
			ExperienceLevel: "green",
			IsVerified:      true,
			IsActive:        false,
			TermsAcceptedAt: time.Now(),
			LastActivityAt:  time.Now(),
		}
		userRepo.Create(inactiveUser)

		reqBody := map[string]string{
			"email":    "inactive@example.com",
			"password": "Test1234",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.Login(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status 403 for inactive user, got %d", rec.Code)
		}
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.Login(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid JSON, got %d", rec.Code)
		}
	})
}

// DONE: TestAuthHandler_ChangePassword tests password change endpoint
func TestAuthHandler_ChangePassword(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewAuthHandler(db, cfg)

	// Create test user
	authService := services.NewAuthService(cfg.JWTSecret, cfg.JWTExpirationHours)
	hash, _ := authService.HashPassword("OldPass123")
	userRepo := repository.NewUserRepository(db)

	email := "changepass@example.com"
	user := &models.User{
		Name:            "Test User",
		Email:           &email,
		PasswordHash:    &hash,
		ExperienceLevel: "green",
		IsVerified:      true,
		IsActive:        true,
		TermsAcceptedAt: time.Now(),
		LastActivityAt:  time.Now(),
	}
	userRepo.Create(user)

	t.Run("successful password change", func(t *testing.T) {
		reqBody := map[string]string{
			"old_password":     "OldPass123",
			"new_password":     "NewPass456",
			"confirm_password": "NewPass456",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/auth/change-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		// Add user context
		ctx := contextWithUser(req.Context(), user.ID, *user.Email, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ChangePassword(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify new password works
		updatedUser, _ := userRepo.FindByID(user.ID)
		if !authService.CheckPassword("NewPass456", *updatedUser.PasswordHash) {
			t.Error("New password should be set correctly")
		}
	})

	t.Run("wrong old password", func(t *testing.T) {
		reqBody := map[string]string{
			"old_password":     "WrongOld123",
			"new_password":     "NewPass789",
			"confirm_password": "NewPass789",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/auth/change-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		ctx := contextWithUser(req.Context(), user.ID, *user.Email, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ChangePassword(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401 for wrong old password, got %d", rec.Code)
		}
	})

	t.Run("new passwords don't match", func(t *testing.T) {
		reqBody := map[string]string{
			"old_password":     "OldPass123",
			"new_password":     "NewPass123",
			"confirm_password": "Different123",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/auth/change-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		ctx := contextWithUser(req.Context(), user.ID, *user.Email, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ChangePassword(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	t.Run("weak new password", func(t *testing.T) {
		reqBody := map[string]string{
			"old_password":     "OldPass123",
			"new_password":     "weak",
			"confirm_password": "weak",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/auth/change-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		ctx := contextWithUser(req.Context(), user.ID, *user.Email, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ChangePassword(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for weak password, got %d", rec.Code)
		}
	})
}

// DONE: TestAuthHandler_VerifyEmail tests email verification endpoint
func TestAuthHandler_VerifyEmail(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewAuthHandler(db, cfg)

	// Create unverified user with verification token
	authService := services.NewAuthService(cfg.JWTSecret, cfg.JWTExpirationHours)
	hash, _ := authService.HashPassword("Test1234")
	token, _ := authService.GenerateToken()
	tokenExpires := time.Now().Add(24 * time.Hour)

	userRepo := repository.NewUserRepository(db)
	email := "verify@example.com"
	user := &models.User{
		Name:                     "Verify Me",
		Email:                    &email,
		PasswordHash:             &hash,
		ExperienceLevel:          "green",
		IsVerified:               false,
		IsActive:                 true,
		VerificationToken:        &token,
		VerificationTokenExpires: &tokenExpires,
		TermsAcceptedAt:          time.Now(),
		LastActivityAt:           time.Now(),
	}
	userRepo.Create(user)

	t.Run("successful verification", func(t *testing.T) {
		reqBody := map[string]string{
			"token": token,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/verify-email", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.VerifyEmail(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify user is now verified
		verifiedUser, _ := userRepo.FindByID(user.ID)
		if !verifiedUser.IsVerified {
			t.Error("User should be verified")
		}

		if verifiedUser.VerificationToken != nil && *verifiedUser.VerificationToken != "" {
			t.Error("Verification token should be cleared")
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		reqBody := map[string]string{
			"token": "invalid-token-xyz",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/verify-email", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.VerifyEmail(rec, req)

		// Should return error (400 or 404 depending on implementation)
		if rec.Code != http.StatusBadRequest && rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 400 or 404, got %d", rec.Code)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		// Create user with expired token
		expiredToken, _ := authService.GenerateToken()
		expiredTime := time.Now().Add(-1 * time.Hour) // Already expired

		email2 := "expired@example.com"
		expiredUser := &models.User{
			Name:                     "Expired Token",
			Email:                    &email2,
			PasswordHash:             &hash,
			ExperienceLevel:          "green",
			IsVerified:               false,
			IsActive:                 true,
			VerificationToken:        &expiredToken,
			VerificationTokenExpires: &expiredTime,
			TermsAcceptedAt:          time.Now(),
			LastActivityAt:           time.Now(),
		}
		userRepo.Create(expiredUser)

		reqBody := map[string]string{
			"token": expiredToken,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/verify-email", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.VerifyEmail(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for expired token, got %d", rec.Code)
		}
	})
}

// Helper function to add user context to request
// Note: Some handlers use middleware constants, others use string keys
// This helper adds both for compatibility
func contextWithUser(ctx context.Context, userID int, email string, isAdmin bool) context.Context {
	// Middleware constants (used by UserHandler, etc.)
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, middleware.EmailKey, email)
	ctx = context.WithValue(ctx, middleware.IsAdminKey, isAdmin)

	// String keys (used by BookingHandler, etc.)
	ctx = context.WithValue(ctx, "user_id", userID)
	ctx = context.WithValue(ctx, "email", email)
	ctx = context.WithValue(ctx, "is_admin", isAdmin)

	return ctx
}
