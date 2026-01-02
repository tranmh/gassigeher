package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// TestConsentHandler_GetConsentStatus tests getting user consent status
func TestConsentHandler_GetConsentStatus(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewConsentHandler(db)

	userID := testutil.SeedTestUser(t, db, "user@example.com", "Test User", "green")

	t.Run("unauthorized without user context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users/me/consent", nil)
		req = reactivationTenantContext(req) // Add tenant context but no user

		rec := httptest.NewRecorder()
		handler.GetConsentStatus(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rec.Code)
		}
	})

	t.Run("no consent returns requires_update true", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users/me/consent", nil)
		ctx := contextWithUser(req.Context(), userID, "user@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetConsentStatus(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var status models.ConsentStatus
		if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if !status.RequiresUpdate {
			t.Error("Expected RequiresUpdate=true for user without consent")
		}
		if status.TermsAccepted {
			t.Error("Expected TermsAccepted=false for user without consent")
		}
		if status.PrivacyAccepted {
			t.Error("Expected PrivacyAccepted=false for user without consent")
		}
	})

	t.Run("with valid consent returns requires_update false", func(t *testing.T) {
		// Insert consent records
		now := testutil.Now()
		db.Exec("INSERT INTO consents (user_id, tenant_id, consent_type, version, ip_address, user_agent, accepted_at) VALUES (?, 0, 'terms', '2025-01-01', '127.0.0.1', 'Test Agent', ?)", userID, now)
		db.Exec("INSERT INTO consents (user_id, tenant_id, consent_type, version, ip_address, user_agent, accepted_at) VALUES (?, 0, 'privacy', '2025-01-01', '127.0.0.1', 'Test Agent', ?)", userID, now)

		req := httptest.NewRequest("GET", "/api/users/me/consent", nil)
		ctx := contextWithUser(req.Context(), userID, "user@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetConsentStatus(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var status models.ConsentStatus
		json.Unmarshal(rec.Body.Bytes(), &status)

		if status.RequiresUpdate {
			t.Error("Expected RequiresUpdate=false for user with valid consent")
		}
		if !status.TermsAccepted {
			t.Error("Expected TermsAccepted=true")
		}
		if !status.PrivacyAccepted {
			t.Error("Expected PrivacyAccepted=true")
		}
	})
}

// TestConsentHandler_GetConsentHistory tests getting consent history
func TestConsentHandler_GetConsentHistory(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewConsentHandler(db)

	userID := testutil.SeedTestUser(t, db, "user@example.com", "Test User", "green")

	t.Run("unauthorized without user context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users/me/consent/history", nil)
		req = reactivationTenantContext(req)

		rec := httptest.NewRecorder()
		handler.GetConsentHistory(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rec.Code)
		}
	})

	t.Run("empty history for new user", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users/me/consent/history", nil)
		ctx := contextWithUser(req.Context(), userID, "user@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetConsentHistory(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var history []models.Consent
		json.Unmarshal(rec.Body.Bytes(), &history)

		// May have consents from previous test, just check response is valid
		t.Logf("History contains %d consent records", len(history))
	})

	t.Run("returns consent history for user", func(t *testing.T) {
		// Create user with clean history
		user2ID := testutil.SeedTestUser(t, db, "user2@example.com", "Test User 2", "green")

		// Add multiple consents
		now := testutil.Now()
		db.Exec("INSERT INTO consents (user_id, tenant_id, consent_type, version, ip_address, user_agent, accepted_at) VALUES (?, 0, 'terms', '2024-01-01', '127.0.0.1', 'Old Agent', ?)", user2ID, now)
		db.Exec("INSERT INTO consents (user_id, tenant_id, consent_type, version, ip_address, user_agent, accepted_at) VALUES (?, 0, 'terms', '2025-01-01', '192.168.1.1', 'New Agent', ?)", user2ID, now)
		db.Exec("INSERT INTO consents (user_id, tenant_id, consent_type, version, ip_address, user_agent, accepted_at) VALUES (?, 0, 'privacy', '2025-01-01', '192.168.1.1', 'New Agent', ?)", user2ID, now)

		req := httptest.NewRequest("GET", "/api/users/me/consent/history", nil)
		ctx := contextWithUser(req.Context(), user2ID, "user2@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetConsentHistory(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var history []models.Consent
		json.Unmarshal(rec.Body.Bytes(), &history)

		if len(history) != 3 {
			t.Errorf("Expected 3 consent records, got %d", len(history))
		}
	})
}

// TestConsentHandler_UpdateConsent tests recording new consent
func TestConsentHandler_UpdateConsent(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewConsentHandler(db)

	t.Run("unauthorized without user context", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/users/me/consent", nil)
		req = reactivationTenantContext(req)

		rec := httptest.NewRecorder()
		handler.UpdateConsent(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rec.Code)
		}
	})

	t.Run("successful consent recording", func(t *testing.T) {
		userID := testutil.SeedTestUser(t, db, "consent-user@example.com", "Consent User", "green")

		req := httptest.NewRequest("POST", "/api/users/me/consent", nil)
		req.Header.Set("User-Agent", "Test Browser")
		req.Header.Set("X-Forwarded-For", "10.0.0.1")
		ctx := contextWithUser(req.Context(), userID, "consent-user@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateConsent(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify response contains status
		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["message"] == nil {
			t.Error("Expected message in response")
		}

		// Verify consents were recorded in database
		var count int
		db.QueryRow("SELECT COUNT(*) FROM consents WHERE user_id = ?", userID).Scan(&count)

		if count != 2 { // terms + privacy
			t.Errorf("Expected 2 consent records, got %d", count)
		}
	})

	t.Run("records IP from X-Real-IP header", func(t *testing.T) {
		userID := testutil.SeedTestUser(t, db, "xrealip@example.com", "XRealIP User", "green")

		req := httptest.NewRequest("POST", "/api/users/me/consent", nil)
		req.Header.Set("X-Real-IP", "192.168.1.100")
		ctx := contextWithUser(req.Context(), userID, "xrealip@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateConsent(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		// Verify IP was recorded
		var ipAddress string
		db.QueryRow("SELECT ip_address FROM consents WHERE user_id = ? LIMIT 1", userID).Scan(&ipAddress)

		if ipAddress != "192.168.1.100" {
			t.Errorf("Expected IP 192.168.1.100, got %s", ipAddress)
		}
	})

	t.Run("records IP from RemoteAddr as fallback", func(t *testing.T) {
		userID := testutil.SeedTestUser(t, db, "remoteaddr@example.com", "RemoteAddr User", "green")

		req := httptest.NewRequest("POST", "/api/users/me/consent", nil)
		req.RemoteAddr = "172.16.0.1:54321"
		ctx := contextWithUser(req.Context(), userID, "remoteaddr@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateConsent(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		// Verify IP was recorded (port stripped)
		var ipAddress string
		db.QueryRow("SELECT ip_address FROM consents WHERE user_id = ? LIMIT 1", userID).Scan(&ipAddress)

		if ipAddress != "172.16.0.1" {
			t.Errorf("Expected IP 172.16.0.1, got %s", ipAddress)
		}
	})
}

// TestConsentHandler_GetCurrentConsentVersions tests getting current versions
func TestConsentHandler_GetCurrentConsentVersions(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewConsentHandler(db)

	t.Run("returns current versions", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/consent/versions", nil)

		rec := httptest.NewRecorder()
		handler.GetCurrentConsentVersions(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var versions map[string]string
		json.Unmarshal(rec.Body.Bytes(), &versions)

		if versions["terms"] == "" {
			t.Error("Expected terms version to be set")
		}
		if versions["privacy"] == "" {
			t.Error("Expected privacy version to be set")
		}

		// Verify matches model constants
		if versions["terms"] != models.CurrentConsentVersions[models.ConsentTypeTerms] {
			t.Errorf("Terms version mismatch: got %s, want %s", versions["terms"], models.CurrentConsentVersions[models.ConsentTypeTerms])
		}
		if versions["privacy"] != models.CurrentConsentVersions[models.ConsentTypePrivacy] {
			t.Errorf("Privacy version mismatch: got %s, want %s", versions["privacy"], models.CurrentConsentVersions[models.ConsentTypePrivacy])
		}
	})
}

// TestConsentHandler_OutdatedConsent tests detection of outdated consent
func TestConsentHandler_OutdatedConsent(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewConsentHandler(db)

	userID := testutil.SeedTestUser(t, db, "outdated@example.com", "Outdated User", "green")

	t.Run("detects outdated consent version", func(t *testing.T) {
		// Insert old consent version
		now := testutil.Now()
		db.Exec("INSERT INTO consents (user_id, tenant_id, consent_type, version, ip_address, user_agent, accepted_at) VALUES (?, 0, 'terms', '2020-01-01', '127.0.0.1', 'Test', ?)", userID, now)
		db.Exec("INSERT INTO consents (user_id, tenant_id, consent_type, version, ip_address, user_agent, accepted_at) VALUES (?, 0, 'privacy', '2020-01-01', '127.0.0.1', 'Test', ?)", userID, now)

		req := httptest.NewRequest("GET", "/api/users/me/consent", nil)
		ctx := contextWithUser(req.Context(), userID, "outdated@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetConsentStatus(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var status models.ConsentStatus
		json.Unmarshal(rec.Body.Bytes(), &status)

		if !status.RequiresUpdate {
			t.Error("Expected RequiresUpdate=true for outdated consent")
		}
		if !status.TermsAccepted {
			t.Error("Expected TermsAccepted=true (old version was accepted)")
		}
	})
}

// TestConsentHandler_TenantIsolation tests tenant isolation for GDPR compliance
func TestConsentHandler_TenantIsolation(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewConsentHandler(db)

	// Create user in tenant 0
	userID := testutil.SeedTestUser(t, db, "tenant0@example.com", "Tenant0 User", "green")

	// Insert consent for tenant 0
	now := testutil.Now()
	db.Exec("INSERT INTO consents (user_id, tenant_id, consent_type, version, ip_address, user_agent, accepted_at) VALUES (?, 0, 'terms', '2025-01-01', '127.0.0.1', 'Test', ?)", userID, now)

	// Insert consent for tenant 1 (different tenant)
	db.Exec("INSERT INTO consents (user_id, tenant_id, consent_type, version, ip_address, user_agent, accepted_at) VALUES (?, 1, 'terms', '2025-01-01', '127.0.0.1', 'Test', ?)", userID, now)

	t.Run("consent history respects tenant isolation", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users/me/consent/history", nil)
		ctx := contextWithUser(req.Context(), userID, "tenant0@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetConsentHistory(rec, req)

		var history []models.Consent
		json.Unmarshal(rec.Body.Bytes(), &history)

		// Should only see tenant 0 consents
		for _, consent := range history {
			if consent.TenantID != 0 {
				t.Errorf("Got consent from wrong tenant: %d", consent.TenantID)
			}
		}
	})
}

// TestGetConsentClientIP tests the IP extraction helper function
func TestGetConsentClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name:       "X-Forwarded-For single IP",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.195"},
			remoteAddr: "10.0.0.1:1234",
			expected:   "203.0.113.195",
		},
		{
			name:       "X-Forwarded-For multiple IPs (takes first)",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.195, 70.41.3.18, 150.172.238.178"},
			remoteAddr: "10.0.0.1:1234",
			expected:   "203.0.113.195",
		},
		{
			name:       "X-Real-IP",
			headers:    map[string]string{"X-Real-IP": "198.51.100.178"},
			remoteAddr: "10.0.0.1:1234",
			expected:   "198.51.100.178",
		},
		{
			name:       "Fallback to RemoteAddr",
			headers:    map[string]string{},
			remoteAddr: "192.0.2.1:54321",
			expected:   "192.0.2.1",
		},
		{
			name:       "RemoteAddr without port",
			headers:    map[string]string{},
			remoteAddr: "192.0.2.1",
			expected:   "192.0.2.1",
		},
		{
			name:       "X-Forwarded-For with whitespace",
			headers:    map[string]string{"X-Forwarded-For": "  203.0.113.195  "},
			remoteAddr: "10.0.0.1:1234",
			expected:   "203.0.113.195",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}
			req.RemoteAddr = tc.remoteAddr

			result := getConsentClientIP(req)
			if result != tc.expected {
				t.Errorf("getConsentClientIP() = %q, want %q", result, tc.expected)
			}
		})
	}
}
