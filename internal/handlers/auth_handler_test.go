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

	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
	"github.com/tranmh/gassigeher/internal/services"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// DONE: TestAuthHandler_Register tests user registration endpoint
func TestAuthHandler_Register(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewAuthHandler(db, cfg)

	// Set up a known registration password for testing
	const testRegPassword = "TEST1234"
	db.Exec(`UPDATE system_settings SET value = ? WHERE key = 'registration_password'`, testRegPassword)

	t.Run("successful registration", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"first_name":            "Test",
			"last_name":             "User",
			"email":                 "newuser@example.com",
			"phone":                 "+49 123 456789",
			"password":              "Test1234",
			"confirm_password":      "Test1234",
			"accept_terms":          true,
			"registration_password": testRegPassword,
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
				name: "missing first name",
				reqBody: map[string]interface{}{
					"last_name":        "User",
					"email":            "test@example.com",
					"phone":            "+49 123 456789",
					"password":         "Test1234",
					"confirm_password": "Test1234",
					"accept_terms":     true,
				},
				expected: "Vorname ist erforderlich",
			},
			{
				name: "missing last name",
				reqBody: map[string]interface{}{
					"first_name":       "Test",
					"email":            "test@example.com",
					"phone":            "+49 123 456789",
					"password":         "Test1234",
					"confirm_password": "Test1234",
					"accept_terms":     true,
				},
				expected: "Nachname ist erforderlich",
			},
			{
				name: "missing email",
				reqBody: map[string]interface{}{
					"first_name":       "Test",
					"last_name":        "User",
					"phone":            "+49 123 456789",
					"password":         "Test1234",
					"confirm_password": "Test1234",
					"accept_terms":     true,
				},
				expected: "E-Mail ist erforderlich",
			},
			{
				name: "missing phone",
				reqBody: map[string]interface{}{
					"first_name":       "Test",
					"last_name":        "User",
					"email":            "test@example.com",
					"password":         "Test1234",
					"confirm_password": "Test1234",
					"accept_terms":     true,
				},
				expected: "Telefonnummer ist erforderlich",
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
			"first_name":       "Test",
			"last_name":        "User",
			"email":            "test@example.com",
			"phone":            "+49 123 4567890",
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

		if !strings.Contains(rec.Body.String(), "stimmen nicht überein") {
			t.Errorf("Expected password mismatch error, got: %s", rec.Body.String())
		}
	})

	t.Run("terms not accepted", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"first_name":       "Test",
			"last_name":        "User",
			"email":            "test@example.com",
			"phone":            "+49 123 456789",
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
			"first_name":       "Test",
			"last_name":        "User",
			"email":            "test@example.com",
			"phone":            "+49 123 456789",
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
			"first_name":            "New",
			"last_name":             "User",
			"email":                 "existing@example.com",
			"phone":                 "+49 123 4567890",
			"password":              "Test1234",
			"confirm_password":      "Test1234",
			"accept_terms":          true,
			"registration_password": testRegPassword,
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
		FirstName:       "Test",
		LastName:        "User",
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
			FirstName:       "Unverified",
			LastName:        "User",
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

		// SECURITY FIX: Now returns 401 with generic message to prevent enumeration
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401 for unverified user (security fix), got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)
		if response["error"] != "Ungültige Anmeldedaten" {
			t.Errorf("Expected generic error message, got %q", response["error"])
		}
	})

	t.Run("inactive user", func(t *testing.T) {
		// Create inactive user
		inactiveEmail := "inactive@example.com"
		inactiveUser := &models.User{
			FirstName:       "Inactive",
			LastName:        "User",
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

		// SECURITY FIX: Now returns 401 with generic message to prevent enumeration
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401 for inactive user (security fix), got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)
		if response["error"] != "Ungültige Anmeldedaten" {
			t.Errorf("Expected generic error message, got %q", response["error"])
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

	// DONE: Security test - prevent account enumeration via uniform error messages
	t.Run("SECURITY: uniform errors prevent account enumeration", func(t *testing.T) {
		// This test ensures attackers cannot determine account state by error messages
		// All authentication failures should return identical errors

		// Create users in different states
		unverifiedEmail := "security_unverified@example.com"
		unverifiedUser := &models.User{
			FirstName:       "Unverified",
			LastName:        "Security Test",
			Email:           &unverifiedEmail,
			PasswordHash:    &hash,
			ExperienceLevel: "green",
			IsVerified:      false, // UNVERIFIED
			IsActive:        true,
			TermsAcceptedAt: time.Now(),
			LastActivityAt:  time.Now(),
		}
		userRepo.Create(unverifiedUser)

		deactivatedEmail := "security_deactivated@example.com"
		deactivatedUser := &models.User{
			FirstName:       "Deactivated",
			LastName:        "Security Test",
			Email:           &deactivatedEmail,
			PasswordHash:    &hash,
			ExperienceLevel: "green",
			IsVerified:      true,
			IsActive:        false, // DEACTIVATED
			TermsAcceptedAt: time.Now(),
			LastActivityAt:  time.Now(),
		}
		userRepo.Create(deactivatedUser)

		// Test scenarios that should all return IDENTICAL error messages
		testCases := []struct {
			name     string
			email    string
			password string
			desc     string
		}{
			{"non-existent user", "nobody@example.com", "Test1234", "user not in database"},
			{"wrong password", "test@example.com", "WrongPass123", "correct email, wrong password"},
			{"unverified user", "security_unverified@example.com", "Test1234", "unverified account"},
			{"deactivated user", "security_deactivated@example.com", "Test1234", "deactivated account"},
		}

		var errorMessages []string
		var statusCodes []int

		for _, tc := range testCases {
			reqBody := map[string]string{
				"email":    tc.email,
				"password": tc.password,
			}

			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			handler.Login(rec, req)

			statusCodes = append(statusCodes, rec.Code)

			var response map[string]interface{}
			json.Unmarshal(rec.Body.Bytes(), &response)
			if msg, ok := response["error"].(string); ok {
				errorMessages = append(errorMessages, msg)
			}

			t.Logf("%s (%s): status=%d, error=%q", tc.name, tc.desc, rec.Code, errorMessages[len(errorMessages)-1])
		}

		// SECURITY CHECK: All error messages should be IDENTICAL
		if len(errorMessages) < 2 {
			t.Fatal("Need at least 2 error messages to compare")
		}

		firstMessage := errorMessages[0]
		for i, msg := range errorMessages {
			if msg != firstMessage {
				t.Errorf("SECURITY VULNERABILITY: Error message %d differs from first: %q vs %q (allows account enumeration!)",
					i, msg, firstMessage)
			}
		}

		// SECURITY CHECK: All status codes should be IDENTICAL (401 Unauthorized)
		firstStatus := statusCodes[0]
		expectedStatus := http.StatusUnauthorized // Should be 401 for all auth failures

		if firstStatus != expectedStatus {
			t.Errorf("Expected status %d for auth failures, got %d", expectedStatus, firstStatus)
		}

		for i, code := range statusCodes {
			if code != expectedStatus {
				t.Errorf("SECURITY VULNERABILITY: Status code %d differs: %d vs %d (allows account enumeration!)",
					i, code, expectedStatus)
			}
		}

		// Log the uniform error message (should be generic like "Ungültige Anmeldedaten")
		t.Logf("✅ SECURITY: All failures return uniform error: %q with status %d", firstMessage, expectedStatus)
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
		FirstName:       "Test",
		LastName:        "User",
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
		FirstName:                "Verify",
		LastName:                 "Me",
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
			FirstName:                "Expired",
			LastName:                 "Token",
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

// DONE: TestAuthHandler_ForgotPassword tests forgot password endpoint
func TestAuthHandler_ForgotPassword(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{JWTSecret: "test-secret"}
	handler := NewAuthHandler(db, cfg)
	userRepo := repository.NewUserRepository(db)

	t.Run("valid email - user exists", func(t *testing.T) {
		email := "reset@example.com"
		testutil.SeedTestUser(t, db, email, "Reset User", "green")

		reqBody := map[string]string{
			"email": email,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/forgot-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ForgotPassword(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		// Verify user has reset token
		user, _ := userRepo.FindByEmail(email)
		if user.PasswordResetToken == nil {
			t.Error("Expected password reset token to be set")
		}
		if user.PasswordResetExpires == nil {
			t.Error("Expected password reset expiration to be set")
		}
	})

	t.Run("empty email", func(t *testing.T) {
		reqBody := map[string]string{
			"email": "",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/forgot-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ForgotPassword(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	t.Run("user does not exist - security response", func(t *testing.T) {
		reqBody := map[string]string{
			"email": "nonexistent@example.com",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/forgot-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ForgotPassword(rec, req)

		// Should still return 200 for security (don't reveal if email exists)
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200 for security, got %d", rec.Code)
		}
	})

	t.Run("invalid request body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/auth/forgot-password", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ForgotPassword(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})
}

// DONE: TestAuthHandler_ResetPassword tests password reset with token
func TestAuthHandler_ResetPassword(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{JWTSecret: "test-secret"}
	handler := NewAuthHandler(db, cfg)
	userRepo := repository.NewUserRepository(db)
	authService := services.NewAuthService(cfg.JWTSecret, 24)

	t.Run("valid token and matching passwords", func(t *testing.T) {
		email := "resetvalid@example.com"
		userID := testutil.SeedTestUser(t, db, email, "Reset User", "green")

		// Generate reset token
		resetToken, _ := authService.GenerateToken()
		expires := time.Now().Add(1 * time.Hour)

		// Get user and set reset token
		user, _ := userRepo.FindByID(userID)
		user.PasswordResetToken = &resetToken
		user.PasswordResetExpires = &expires
		userRepo.Update(user)

		reqBody := map[string]string{
			"token":            resetToken,
			"password":         "NewPassword123!",
			"confirm_password": "NewPassword123!",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/reset-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ResetPassword(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		// Verify token cleared
		updatedUser, _ := userRepo.FindByID(userID)
		if updatedUser.PasswordResetToken != nil {
			t.Error("Expected password reset token to be cleared")
		}
	})

	t.Run("empty token", func(t *testing.T) {
		reqBody := map[string]string{
			"token":            "",
			"password":         "NewPassword123!",
			"confirm_password": "NewPassword123!",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/reset-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ResetPassword(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	t.Run("passwords do not match", func(t *testing.T) {
		reqBody := map[string]string{
			"token":            "some-token",
			"password":         "NewPassword123!",
			"confirm_password": "DifferentPassword456!",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/reset-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ResetPassword(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	t.Run("invalid password - too short", func(t *testing.T) {
		reqBody := map[string]string{
			"token":            "some-token",
			"password":         "short",
			"confirm_password": "short",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/reset-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ResetPassword(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	t.Run("invalid token - user not found", func(t *testing.T) {
		reqBody := map[string]string{
			"token":            "invalid-token-xyz",
			"password":         "NewPassword123!",
			"confirm_password": "NewPassword123!",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/reset-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ResetPassword(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		email := "resetexpired@example.com"
		userID := testutil.SeedTestUser(t, db, email, "Expired User", "green")

		// Generate reset token with expired time
		resetToken, _ := authService.GenerateToken()
		expires := time.Now().Add(-1 * time.Hour) // Expired 1 hour ago

		// Get user and set expired reset token
		user, _ := userRepo.FindByID(userID)
		user.PasswordResetToken = &resetToken
		user.PasswordResetExpires = &expires
		userRepo.Update(user)

		reqBody := map[string]string{
			"token":            resetToken,
			"password":         "NewPassword123!",
			"confirm_password": "NewPassword123!",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/auth/reset-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ResetPassword(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for expired token, got %d", rec.Code)
		}
	})

	t.Run("invalid request body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/auth/reset-password", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ResetPassword(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
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
