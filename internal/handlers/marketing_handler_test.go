package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/tranmh/gassigeher/internal/database"
	"github.com/tranmh/gassigeher/internal/middleware"
)

func setupMarketingTestDB(t *testing.T) *database.DB {
	rawDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Create required tables
	_, err = rawDB.Exec(`
		CREATE TABLE referral_codes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL UNIQUE,
			referrer_tenant_id INTEGER,
			referrer_email TEXT,
			discount_months_referrer INTEGER DEFAULT 3,
			discount_months_referee INTEGER DEFAULT 1,
			uses_count INTEGER DEFAULT 0,
			max_uses INTEGER,
			is_active INTEGER DEFAULT 1,
			expires_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE marketing_campaigns (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			config TEXT,
			is_active INTEGER DEFAULT 0,
			start_date TIMESTAMP,
			end_date TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE reference_entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tenant_id INTEGER NOT NULL UNIQUE,
			display_name TEXT NOT NULL,
			city TEXT,
			federal_state TEXT,
			website_url TEXT,
			testimonial TEXT,
			logo_url TEXT,
			is_approved INTEGER DEFAULT 0,
			display_order INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE referral_uses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code_id INTEGER NOT NULL,
			referee_tenant_id INTEGER NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE tenants (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			slug TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			status TEXT DEFAULT 'active',
			contact_email TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	// Wrap in database.DB for auto-rebinding
	sqlxDB := sqlx.NewDb(rawDB, "sqlite3")
	return database.WrapSqlxDB(sqlxDB, database.NewSQLiteDialect())
}

// ========== BUG: Test that expires_at accepts RFC3339 format ==========

func TestCreateReferralCode_ExpiresAt_RFC3339Format(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	// Test with RFC3339 format (ISO 8601 with time)
	// Use a future date to avoid "date in past" validation error
	reqBody := `{
		"code": "RFC3339TEST",
		"discount_months_referrer": 3,
		"discount_months_referee": 1,
		"expires_at": "2027-12-31T23:59:59Z"
	}`

	req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/referral-codes", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CreateReferralCode(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// Parse response
	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Check expires_at was parsed correctly
	expiresAt, ok := response["expires_at"].(string)
	if !ok {
		t.Fatalf("expires_at not found in response")
	}

	// Verify RFC3339 format is parsed correctly
	parsedTime, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		t.Fatalf("Failed to parse expires_at: %v", err)
	}

	if parsedTime.Year() != 2027 {
		t.Errorf("Expected year 2027, got %d. expires_at was: %s", parsedTime.Year(), expiresAt)
	}
	if parsedTime.Month() != 12 {
		t.Errorf("Expected month 12, got %d", parsedTime.Month())
	}
	if parsedTime.Day() != 31 {
		t.Errorf("Expected day 31, got %d", parsedTime.Day())
	}
}

func TestCreateReferralCode_ExpiresAt_DateOnlyFormat(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	// Test with date-only format (YYYY-MM-DD) - use future date
	reqBody := `{
		"code": "DATEONLYTEST",
		"discount_months_referrer": 3,
		"discount_months_referee": 1,
		"expires_at": "2026-06-15"
	}`

	req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/referral-codes", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CreateReferralCode(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// Parse response
	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Check expires_at was parsed correctly
	expiresAt, ok := response["expires_at"].(string)
	if !ok {
		t.Fatalf("expires_at not found in response")
	}

	parsedTime, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		t.Fatalf("Failed to parse expires_at: %v", err)
	}

	if parsedTime.Year() != 2026 {
		t.Errorf("Expected year 2026, got %d", parsedTime.Year())
	}
	if parsedTime.Month() != 6 {
		t.Errorf("Expected month 6, got %d", parsedTime.Month())
	}
	if parsedTime.Day() != 15 {
		t.Errorf("Expected day 15, got %d", parsedTime.Day())
	}
}

func TestCreateReferralCode_ExpiresAt_InvalidFormat_ReturnsError(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	// Test with invalid date format
	reqBody := `{
		"code": "INVALIDDATE",
		"discount_months_referrer": 3,
		"discount_months_referee": 1,
		"expires_at": "31-12-2025"
	}`

	req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/referral-codes", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CreateReferralCode(rec, req)

	// BUG: Currently this silently accepts invalid dates and stores zero time
	// It should return an error for invalid date formats
	if rec.Code == http.StatusCreated {
		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)
		expiresAt := response["expires_at"].(string)
		// If we got here with zero time, that's a bug
		if expiresAt == "0001-01-01T00:00:00Z" {
			t.Errorf("Invalid date format was silently accepted and stored as zero time")
		}
	}
	// Expected: should return 400 Bad Request for invalid date format
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid date format, got %d", rec.Code)
	}
}

// ========== BUG #1: Past expiry date should be rejected ==========

func TestCreateReferralCode_PastExpiryDate_ReturnsError(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	// Test with past expiry date
	reqBody := `{
		"code": "PASTEXPIRY",
		"discount_months_referrer": 3,
		"discount_months_referee": 1,
		"expires_at": "2020-01-01T00:00:00Z"
	}`

	req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/referral-codes", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CreateReferralCode(rec, req)

	// BUG: Currently accepts past expiry dates
	// Should return 400 Bad Request
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for past expiry date, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// ========== BUG #2: XSS should be sanitized in referral codes ==========

func TestCreateReferralCode_XSSInCode_Sanitized(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	// Test with XSS attempt in code
	reqBody := `{
		"code": "TEST<script>alert(1)</script>",
		"discount_months_referrer": 3,
		"discount_months_referee": 1
	}`

	req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/referral-codes", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CreateReferralCode(rec, req)

	// Should either reject or sanitize the code
	if rec.Code == http.StatusCreated {
		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)
		code := response["code"].(string)
		// Code should not contain HTML tags
		if code != "TESTALERT1" && code != "TEST" {
			// If it contains any < or > characters, it's a bug
			for _, c := range code {
				if c == '<' || c == '>' {
					t.Errorf("Code contains HTML characters which is a XSS risk: %s", code)
					break
				}
			}
		}
	}
}

// ========== BUG #3: Negative discount months should be rejected ==========

func TestCreateReferralCode_NegativeDiscount_ReturnsError(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	// Test with negative discount months
	reqBody := `{
		"code": "NEGATIVEDISC",
		"discount_months_referrer": -5,
		"discount_months_referee": -10
	}`

	req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/referral-codes", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CreateReferralCode(rec, req)

	// BUG: Currently accepts negative discount months
	// Should return 400 Bad Request
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for negative discount months, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// ========== BUG #4: Discount months should have a reasonable max limit ==========

func TestCreateReferralCode_ExcessiveDiscount_ReturnsError(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	// Test with excessively large discount months (more than 24 months / 2 years is unreasonable)
	reqBody := `{
		"code": "HUGEDISC",
		"discount_months_referrer": 999999999,
		"discount_months_referee": 999999999
	}`

	req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/referral-codes", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CreateReferralCode(rec, req)

	// BUG: Currently accepts unreasonably large discount values
	// Should return 400 Bad Request (max 24 months reasonable)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for excessive discount months, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// ========== BUG #5: Referral code should only contain alphanumeric and limited special chars ==========

func TestCreateReferralCode_InvalidCharacters_ReturnsError(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	testCases := []struct {
		name string
		code string
	}{
		{"SQL injection", "'; DROP TABLE referral_codes;--"},
		{"HTML tags", "<script>alert(1)</script>"},
		{"Spaces", "CODE WITH SPACES"},
		{"Special chars", "CODE@#$%^&*()"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := `{"code": "` + tc.code + `", "discount_months_referrer": 1, "discount_months_referee": 1}`

			req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/referral-codes", bytes.NewBufferString(reqBody))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
			ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			handler.CreateReferralCode(rec, req)

			// Should reject codes with invalid characters
			if rec.Code == http.StatusCreated {
				var response map[string]interface{}
				json.Unmarshal(rec.Body.Bytes(), &response)
				storedCode := response["code"].(string)
				// Check if dangerous characters are present (hyphens are allowed)
				for _, c := range storedCode {
					if c == '<' || c == '>' || c == '\'' || c == ';' {
						t.Errorf("Code '%s' contains potentially dangerous character '%c'", storedCode, c)
					}
				}
			}
		})
	}
}
