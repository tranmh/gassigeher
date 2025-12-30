package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// =============================================================================
// BUG #1: GLOBAL EMAIL UNIQUENESS - Regular User Registration
// =============================================================================
// CRITICAL: Same email can be registered as a regular user in multiple tenants.
// This creates login ambiguity - at login, which tenant should be used?
//
// Current behavior:
// - Tenant registration (admin): EmailExistsGlobally() check - CORRECT
// - Regular user registration: FindByEmail(email, tenantID) - ONLY checks within tenant - BUG!
//
// Expected behavior:
// - ALL email addresses should be globally unique across all tenants
// - Registration should fail if email exists in ANY tenant
// =============================================================================

// TestAuthHandler_Register_RejectsDuplicateEmailAcrossTenants tests that the same
// email cannot be used to register as a user in multiple tenants
// TDD RED PHASE: This test should FAIL until we implement global email uniqueness
func TestAuthHandler_Register_RejectsDuplicateEmailAcrossTenants(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewAuthHandler(db, cfg)

	// Set up registration password for both tenants
	const testRegPassword = "TEST1234"
	db.Exec(`INSERT OR REPLACE INTO system_settings (tenant_id, key, value) VALUES (0, 'registration_password', ?)`, testRegPassword)
	db.Exec(`INSERT OR REPLACE INTO system_settings (tenant_id, key, value) VALUES (1, 'registration_password', ?)`, testRegPassword)

	// First registration in tenant 0 - should succeed
	reqBody1 := map[string]interface{}{
		"first_name":            "Max",
		"last_name":             "Mustermann",
		"email":                 "shared@crosstenanttest.com",
		"phone":                 "+49 123 456789",
		"password":              "SecurePass123!",
		"confirm_password":      "SecurePass123!",
		"accept_terms":          true,
		"accept_privacy":        true,
		"registration_password": testRegPassword,
	}
	body1, _ := json.Marshal(reqBody1)

	req1 := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	// Tenant 0 context
	ctx1 := context.WithValue(req1.Context(), middleware.TenantIDKey, 0)
	req1 = req1.WithContext(ctx1)

	rec1 := httptest.NewRecorder()
	handler.Register(rec1, req1)

	if rec1.Code != http.StatusCreated {
		t.Fatalf("First registration should succeed, got %d: %s", rec1.Code, rec1.Body.String())
	}

	// Verify user was created
	var userCount int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", "shared@crosstenanttest.com").Scan(&userCount)
	if userCount != 1 {
		t.Fatalf("Expected 1 user with shared email, got %d", userCount)
	}

	// Second registration with SAME email in DIFFERENT tenant (tenant 1) - should FAIL
	reqBody2 := map[string]interface{}{
		"first_name":            "Anna",
		"last_name":             "Schmidt",
		"email":                 "shared@crosstenanttest.com", // SAME EMAIL!
		"phone":                 "+49 987 654321",
		"password":              "AnotherPass456!",
		"confirm_password":      "AnotherPass456!",
		"accept_terms":          true,
		"accept_privacy":        true,
		"registration_password": testRegPassword,
	}
	body2, _ := json.Marshal(reqBody2)

	req2 := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	// Tenant 1 context - DIFFERENT TENANT
	ctx2 := context.WithValue(req2.Context(), middleware.TenantIDKey, 1)
	req2 = req2.WithContext(ctx2)

	rec2 := httptest.NewRecorder()
	handler.Register(rec2, req2)

	// BUG CHECK: If this returns 201, the bug exists!
	if rec2.Code == http.StatusCreated {
		// Count users with this email - should be only 1
		var finalCount int
		db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", "shared@crosstenanttest.com").Scan(&finalCount)

		if finalCount > 1 {
			t.Errorf("BUG #1 CONFIRMED: Same email 'shared@crosstenanttest.com' registered in %d different tenants!", finalCount)
			t.Errorf("This creates login ambiguity - which tenant should the user log into?")
		}
	}

	// Expected behavior: Should NOT create a second user
	// (either reject with error or return 201 but not create user for security)
	var finalUserCount int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", "shared@crosstenanttest.com").Scan(&finalUserCount)

	if finalUserCount > 1 {
		t.Errorf("BUG: Found %d users with email 'shared@crosstenanttest.com' - should be only 1!", finalUserCount)
		t.Errorf("Global email uniqueness is NOT enforced for regular user registration")
	}
}

// TestAuthHandler_Register_DifferentEmailsDifferentTenants verifies that different
// emails in different tenants still work correctly
func TestAuthHandler_Register_DifferentEmailsDifferentTenants(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewAuthHandler(db, cfg)

	// Set up registration password for both tenants
	const testRegPassword = "TEST1234"
	db.Exec(`INSERT OR REPLACE INTO system_settings (tenant_id, key, value) VALUES (0, 'registration_password', ?)`, testRegPassword)
	db.Exec(`INSERT OR REPLACE INTO system_settings (tenant_id, key, value) VALUES (1, 'registration_password', ?)`, testRegPassword)

	// First registration in tenant 0
	reqBody1 := map[string]interface{}{
		"first_name":            "User",
		"last_name":             "One",
		"email":                 "user1@uniquetest.com", // Unique email
		"phone":                 "+49 111 111111",
		"password":              "SecurePass123!",
		"confirm_password":      "SecurePass123!",
		"accept_terms":          true,
		"accept_privacy":        true,
		"registration_password": testRegPassword,
	}
	body1, _ := json.Marshal(reqBody1)

	req1 := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	ctx1 := context.WithValue(req1.Context(), middleware.TenantIDKey, 0)
	req1 = req1.WithContext(ctx1)

	rec1 := httptest.NewRecorder()
	handler.Register(rec1, req1)

	if rec1.Code != http.StatusCreated {
		t.Fatalf("First registration should succeed, got %d", rec1.Code)
	}

	// Second registration in tenant 1 with DIFFERENT email - should succeed
	reqBody2 := map[string]interface{}{
		"first_name":            "User",
		"last_name":             "Two",
		"email":                 "user2@uniquetest.com", // Different email
		"phone":                 "+49 222 222222",
		"password":              "SecurePass456!",
		"confirm_password":      "SecurePass456!",
		"accept_terms":          true,
		"accept_privacy":        true,
		"registration_password": testRegPassword,
	}
	body2, _ := json.Marshal(reqBody2)

	req2 := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	ctx2 := context.WithValue(req2.Context(), middleware.TenantIDKey, 1)
	req2 = req2.WithContext(ctx2)

	rec2 := httptest.NewRecorder()
	handler.Register(rec2, req2)

	if rec2.Code != http.StatusCreated {
		t.Errorf("Second registration with different email should succeed, got %d: %s",
			rec2.Code, rec2.Body.String())
	}
}

// TestAuthHandler_Register_EmailExistsInDifferentTenant_SecurityResponse tests
// that the error response doesn't reveal which tenant has the email (security)
func TestAuthHandler_Register_EmailExistsInDifferentTenant_SecurityResponse(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewAuthHandler(db, cfg)

	// Set up registration password
	const testRegPassword = "TEST1234"
	db.Exec(`INSERT OR REPLACE INTO system_settings (tenant_id, key, value) VALUES (0, 'registration_password', ?)`, testRegPassword)
	db.Exec(`INSERT OR REPLACE INTO system_settings (tenant_id, key, value) VALUES (1, 'registration_password', ?)`, testRegPassword)

	// First registration in tenant 0
	email := "security-test@crosstenanttest.com"
	reqBody1 := map[string]interface{}{
		"first_name":            "First",
		"last_name":             "User",
		"email":                 email,
		"phone":                 "+49 123 456789",
		"password":              "SecurePass123!",
		"confirm_password":      "SecurePass123!",
		"accept_terms":          true,
		"accept_privacy":        true,
		"registration_password": testRegPassword,
	}
	body1, _ := json.Marshal(reqBody1)

	req1 := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	ctx1 := context.WithValue(req1.Context(), middleware.TenantIDKey, 0)
	req1 = req1.WithContext(ctx1)

	rec1 := httptest.NewRecorder()
	handler.Register(rec1, req1)

	if rec1.Code != http.StatusCreated {
		t.Fatalf("First registration should succeed, got %d", rec1.Code)
	}

	// Second registration attempt in tenant 1 with same email
	reqBody2 := map[string]interface{}{
		"first_name":            "Second",
		"last_name":             "User",
		"email":                 email, // Same email
		"phone":                 "+49 987 654321",
		"password":              "AnotherPass456!",
		"confirm_password":      "AnotherPass456!",
		"accept_terms":          true,
		"accept_privacy":        true,
		"registration_password": testRegPassword,
	}
	body2, _ := json.Marshal(reqBody2)

	req2 := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	ctx2 := context.WithValue(req2.Context(), middleware.TenantIDKey, 1)
	req2 = req2.WithContext(ctx2)

	rec2 := httptest.NewRecorder()
	handler.Register(rec2, req2)

	// For security reasons, the response should be the same as successful registration
	// to prevent email enumeration across tenants
	// Current behavior returns 201 but creates a new user (bug)
	// Fixed behavior should return 201 but NOT create a new user

	var userCount int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", email).Scan(&userCount)

	if userCount > 1 {
		t.Errorf("BUG: Email exists in %d tenants. Global uniqueness not enforced!", userCount)
	}

	// Response should be same as successful registration (security: prevent enumeration)
	if rec2.Code != http.StatusCreated {
		t.Logf("Note: Registration with existing global email returned %d (should return 201 for security)", rec2.Code)
	}
}
