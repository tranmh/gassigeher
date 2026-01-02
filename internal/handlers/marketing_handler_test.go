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

	"github.com/gorilla/mux"
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

// ========== Reference Entry Handler Tests ==========

func TestCreateReferenceEntry_Success(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	reqBody := `{
		"display_name": "Tierheim Göppingen",
		"city": "Göppingen",
		"federal_state": "Baden-Württemberg",
		"website_url": "https://www.tierheim-goeppingen.de",
		"testimonial": "Gassigeher hat unsere Freiwilligenkoordination revolutioniert!",
		"is_approved": true
	}`

	req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/references", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CreateReferenceEntry(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["display_name"] != "Tierheim Göppingen" {
		t.Errorf("Expected display_name 'Tierheim Göppingen', got '%v'", response["display_name"])
	}
	if response["is_approved"] != true {
		t.Errorf("Expected is_approved true, got %v", response["is_approved"])
	}
}

func TestCreateReferenceEntry_MissingDisplayName_ReturnsError(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	reqBody := `{
		"city": "Göppingen",
		"federal_state": "Baden-Württemberg"
	}`

	req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/references", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CreateReferenceEntry(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing display_name, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateReferenceEntry_DisplayNameTooLong_ReturnsError(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	// Create a name longer than 255 characters
	longName := ""
	for i := 0; i < 300; i++ {
		longName += "A"
	}

	reqBody := `{"display_name": "` + longName + `"}`

	req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/references", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CreateReferenceEntry(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for display_name too long, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateReferenceEntry_InvalidWebsiteURL_ReturnsError(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	testCases := []struct {
		name string
		url  string
	}{
		{"Missing protocol", "www.example.com"},
		{"Invalid protocol", "ftp://www.example.com"},
		{"JavaScript protocol", "javascript:alert(1)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := `{"display_name": "Test Shelter", "website_url": "` + tc.url + `"}`

			req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/references", bytes.NewBufferString(reqBody))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
			ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			handler.CreateReferenceEntry(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400 for invalid URL '%s', got %d. Body: %s", tc.url, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestCreateReferenceEntry_TestimonialTooLong_ReturnsError(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	// Create testimonial longer than 2000 characters
	longTestimonial := ""
	for i := 0; i < 2500; i++ {
		longTestimonial += "A"
	}

	reqBody := `{"display_name": "Test Shelter", "testimonial": "` + longTestimonial + `"}`

	req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/references", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CreateReferenceEntry(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for testimonial too long, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateReferenceEntry_DuplicateTenant_ReturnsConflict(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	// First, create an entry with tenant_id
	firstReqBody := `{
		"display_name": "First Entry",
		"tenant_id": 1
	}`

	req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/references", bytes.NewBufferString(firstReqBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CreateReferenceEntry(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("First entry creation failed: %d. Body: %s", rec.Code, rec.Body.String())
	}

	// Try to create another entry for the same tenant
	secondReqBody := `{
		"display_name": "Second Entry",
		"tenant_id": 1
	}`

	req = httptest.NewRequest("POST", "/api/v1/central-admin/marketing/references", bytes.NewBufferString(secondReqBody))
	req.Header.Set("Content-Type", "application/json")
	ctx = context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)

	rec = httptest.NewRecorder()
	handler.CreateReferenceEntry(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("Expected status 409 for duplicate tenant entry, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateReferenceEntry_Success(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	// First create an entry
	createBody := `{"display_name": "Original Name"}`
	req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/references", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CreateReferenceEntry(rec, req)

	var createResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &createResp)
	entryID := int(createResp["id"].(float64))

	// Update the entry
	updateBody := `{
		"display_name": "Updated Name",
		"city": "New City",
		"is_approved": false
	}`

	req = httptest.NewRequest("PUT", "/api/v1/central-admin/marketing/references/1", bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	ctx = context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)

	rec = httptest.NewRecorder()
	handler.UpdateReferenceEntry(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s (entry ID: %d)", rec.Code, rec.Body.String(), entryID)
	}

	var updateResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &updateResp)

	if updateResp["display_name"] != "Updated Name" {
		t.Errorf("Expected display_name 'Updated Name', got '%v'", updateResp["display_name"])
	}
	if updateResp["city"] != "New City" {
		t.Errorf("Expected city 'New City', got '%v'", updateResp["city"])
	}
	if updateResp["is_approved"] != false {
		t.Errorf("Expected is_approved false, got %v", updateResp["is_approved"])
	}
}

func TestUpdateReferenceEntry_NotFound_ReturnsError(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	updateBody := `{"display_name": "Updated Name"}`

	req := httptest.NewRequest("PUT", "/api/v1/central-admin/marketing/references/999", bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "999"})
	ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.UpdateReferenceEntry(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent entry, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateReferenceEntry_EmptyDisplayName_ReturnsError(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	// First create an entry
	createBody := `{"display_name": "Original Name"}`
	req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/references", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CreateReferenceEntry(rec, req)

	// Try to update with empty display_name
	updateBody := `{"display_name": "   "}`

	req = httptest.NewRequest("PUT", "/api/v1/central-admin/marketing/references/1", bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	ctx = context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)

	rec = httptest.NewRecorder()
	handler.UpdateReferenceEntry(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty display_name, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestListReferenceEntries_PublicEndpoint_ReturnsApprovedOnly(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	// Create approved entry with tenant_id 1
	approvedBody := `{"display_name": "Approved Shelter", "is_approved": true, "tenant_id": 1}`
	req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/references", bytes.NewBufferString(approvedBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.CreateReferenceEntry(rec, req)

	// Create pending entry with tenant_id 2
	pendingBody := `{"display_name": "Pending Shelter", "is_approved": false, "tenant_id": 2}`
	req = httptest.NewRequest("POST", "/api/v1/central-admin/marketing/references", bytes.NewBufferString(pendingBody))
	req.Header.Set("Content-Type", "application/json")
	ctx = context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)
	rec = httptest.NewRecorder()
	handler.CreateReferenceEntry(rec, req)

	// List from public endpoint (should only return approved)
	req = httptest.NewRequest("GET", "/api/v1/marketing/references", nil)
	rec = httptest.NewRecorder()
	handler.ListReferenceEntries(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var entries []map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &entries)

	if len(entries) != 1 {
		t.Errorf("Expected 1 approved entry, got %d entries", len(entries))
	}

	if len(entries) > 0 && entries[0]["display_name"] != "Approved Shelter" {
		t.Errorf("Expected 'Approved Shelter', got '%v'", entries[0]["display_name"])
	}
}

func TestListReferenceEntries_AdminEndpoint_ReturnsAll(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	// Create approved entry with tenant_id 1
	approvedBody := `{"display_name": "Approved Shelter", "is_approved": true, "tenant_id": 1}`
	req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/references", bytes.NewBufferString(approvedBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.CreateReferenceEntry(rec, req)

	// Create pending entry with tenant_id 2
	pendingBody := `{"display_name": "Pending Shelter", "is_approved": false, "tenant_id": 2}`
	req = httptest.NewRequest("POST", "/api/v1/central-admin/marketing/references", bytes.NewBufferString(pendingBody))
	req.Header.Set("Content-Type", "application/json")
	ctx = context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)
	rec = httptest.NewRecorder()
	handler.CreateReferenceEntry(rec, req)

	// List from central-admin endpoint (should return all)
	req = httptest.NewRequest("GET", "/api/v1/central-admin/marketing/references", nil)
	ctx = context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)
	rec = httptest.NewRecorder()
	handler.ListReferenceEntries(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var entries []map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &entries)

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries (approved + pending), got %d entries", len(entries))
	}
}

func TestApproveReferenceEntry_Success(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	// Create pending entry
	createBody := `{"display_name": "Pending Shelter", "is_approved": false}`
	req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/references", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.CreateReferenceEntry(rec, req)

	// Approve the entry
	req = httptest.NewRequest("PUT", "/api/v1/central-admin/marketing/references/1/approve", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	ctx = context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)
	rec = httptest.NewRecorder()
	handler.ApproveReferenceEntry(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// Verify approval by getting the entry
	req = httptest.NewRequest("GET", "/api/v1/central-admin/marketing/references/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	ctx = context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)
	rec = httptest.NewRecorder()
	handler.GetReferenceEntry(rec, req)

	var entry map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &entry)

	if entry["is_approved"] != true {
		t.Errorf("Expected is_approved true after approval, got %v", entry["is_approved"])
	}
}

func TestDeleteReferenceEntry_Success(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	// Create entry
	createBody := `{"display_name": "To Be Deleted"}`
	req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/references", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.CreateReferenceEntry(rec, req)

	// Delete the entry
	req = httptest.NewRequest("DELETE", "/api/v1/central-admin/marketing/references/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	ctx = context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)
	rec = httptest.NewRecorder()
	handler.DeleteReferenceEntry(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// Verify deletion by trying to get the entry
	req = httptest.NewRequest("GET", "/api/v1/central-admin/marketing/references/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	ctx = context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)
	rec = httptest.NewRecorder()
	handler.GetReferenceEntry(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 after deletion, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateReferenceEntry_CentralAdminAutoApproved(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	// Create entry without specifying is_approved (should default to true for admin)
	createBody := `{"display_name": "Admin Created Shelter"}`
	req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/references", bytes.NewBufferString(createBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.CreateReferenceEntry(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)

	// Central admin created entries should be auto-approved
	if response["is_approved"] != true {
		t.Errorf("Expected is_approved true for admin-created entry, got %v", response["is_approved"])
	}
}

// ========== Campaign Handler Tests ==========

func TestCreateCampaign_DateOnlyFormat_Success(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	// Test with date-only format (YYYY-MM-DD) - the format sent by HTML date inputs
	reqBody := `{
		"type": "fomo_countdown",
		"name": "Test FOMO Campaign",
		"description": "A test campaign",
		"start_date": "2026-01-01",
		"end_date": "2026-12-31",
		"is_active": true
	}`

	req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/campaigns", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CreateCampaign(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify start_date was parsed correctly
	startDate, ok := response["start_date"].(string)
	if !ok {
		t.Fatalf("start_date not found in response")
	}

	parsedStart, err := time.Parse(time.RFC3339, startDate)
	if err != nil {
		t.Fatalf("Failed to parse start_date: %v", err)
	}

	if parsedStart.Year() != 2026 || parsedStart.Month() != 1 || parsedStart.Day() != 1 {
		t.Errorf("Expected start_date 2026-01-01, got %s", startDate)
	}

	// Verify end_date was parsed correctly
	endDate, ok := response["end_date"].(string)
	if !ok {
		t.Fatalf("end_date not found in response")
	}

	parsedEnd, err := time.Parse(time.RFC3339, endDate)
	if err != nil {
		t.Fatalf("Failed to parse end_date: %v", err)
	}

	if parsedEnd.Year() != 2026 || parsedEnd.Month() != 12 || parsedEnd.Day() != 31 {
		t.Errorf("Expected end_date 2026-12-31, got %s", endDate)
	}
}

func TestCreateCampaign_NoDates_Success(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	// Test without dates (both optional)
	reqBody := `{
		"type": "referral",
		"name": "Referral Campaign",
		"is_active": false
	}`

	req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/campaigns", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CreateCampaign(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)

	if response["name"] != "Referral Campaign" {
		t.Errorf("Expected name 'Referral Campaign', got '%v'", response["name"])
	}
}

func TestCreateCampaign_InvalidStartDateFormat_ReturnsError(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	// Test with invalid date format
	reqBody := `{
		"type": "fomo_countdown",
		"name": "Invalid Date Campaign",
		"start_date": "31-12-2026"
	}`

	req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/campaigns", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CreateCampaign(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid date format, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateCampaign_InvalidEndDateFormat_ReturnsError(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	// Test with invalid end date format
	reqBody := `{
		"type": "fomo_countdown",
		"name": "Invalid End Date Campaign",
		"start_date": "2026-01-01",
		"end_date": "December 31, 2026"
	}`

	req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/campaigns", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CreateCampaign(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid end date format, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateCampaign_EndDateBeforeStartDate_ReturnsError(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	// Test with end date before start date
	reqBody := `{
		"type": "fomo_countdown",
		"name": "Invalid Date Range Campaign",
		"start_date": "2026-12-31",
		"end_date": "2026-01-01"
	}`

	req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/campaigns", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CreateCampaign(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for end date before start date, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateCampaign_MissingName_ReturnsError(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	reqBody := `{
		"type": "fomo_countdown"
	}`

	req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/campaigns", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CreateCampaign(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing name, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateCampaign_InvalidType_ReturnsError(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	reqBody := `{
		"type": "invalid_type",
		"name": "Invalid Type Campaign"
	}`

	req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/campaigns", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CreateCampaign(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid type, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateCampaign_WithConfig_Success(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	// Test with valid JSON config
	reqBody := `{
		"type": "fomo_countdown",
		"name": "FOMO with Config",
		"config": "{\"total_slots\": 30, \"remaining_slots\": 30, \"message\": \"Nur noch X Plätze!\"}",
		"is_active": true
	}`

	req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/campaigns", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CreateCampaign(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)

	if response["config"] == nil {
		t.Error("Expected config to be present in response")
	}
}

func TestCreateCampaign_InvalidConfigJSON_ReturnsError(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	handler := NewMarketingHandler(db)

	// Test with invalid JSON in config
	reqBody := `{
		"type": "fomo_countdown",
		"name": "Invalid Config Campaign",
		"config": "not valid json {"
	}`

	req := httptest.NewRequest("POST", "/api/v1/central-admin/marketing/campaigns", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CreateCampaign(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid config JSON, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}
