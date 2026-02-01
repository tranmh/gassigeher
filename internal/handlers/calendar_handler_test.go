package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/database"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/repository"

	_ "modernc.org/sqlite"
)

// setupCalendarTestDB creates a test database for calendar tests
func setupCalendarTestDB(t *testing.T) *database.DB {
	rawDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create users table
	_, err = rawDB.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tenant_id INTEGER DEFAULT 0,
			first_name TEXT,
			last_name TEXT,
			email TEXT,
			phone TEXT,
			password_hash TEXT,
			is_admin INTEGER DEFAULT 0,
			is_super_admin INTEGER DEFAULT 0,
			is_central_admin INTEGER DEFAULT 0,
			is_verified INTEGER DEFAULT 1,
			is_active INTEGER DEFAULT 1,
			is_deleted INTEGER DEFAULT 0,
			must_change_password INTEGER DEFAULT 0,
			verification_token TEXT,
			verification_token_expires TIMESTAMP,
			password_reset_token TEXT,
			password_reset_expires TIMESTAMP,
			calendar_token TEXT,
			profile_photo TEXT,
			anonymous_id TEXT,
			terms_accepted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_activity_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			deactivated_at TIMESTAMP,
			deactivation_reason TEXT,
			reactivated_at TIMESTAMP,
			deleted_at TIMESTAMP,
			failed_login_attempts INTEGER DEFAULT 0,
			locked_until TEXT,
			last_failed_login TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create users table: %v", err)
	}

	// Create bookings table
	_, err = rawDB.Exec(`
		CREATE TABLE bookings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tenant_id INTEGER DEFAULT 0,
			user_id INTEGER NOT NULL,
			dog_id INTEGER NOT NULL,
			date TEXT NOT NULL,
			scheduled_time TEXT NOT NULL,
			status TEXT DEFAULT 'scheduled',
			completed_at TIMESTAMP,
			reminder_sent_at TIMESTAMP,
			user_notes TEXT,
			admin_cancellation_reason TEXT,
			requires_approval INTEGER DEFAULT 0,
			approval_status TEXT DEFAULT 'approved',
			approved_by INTEGER,
			approved_at TIMESTAMP,
			rejection_reason TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create bookings table: %v", err)
	}

	// Create dogs table
	_, err = rawDB.Exec(`
		CREATE TABLE dogs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tenant_id INTEGER DEFAULT 0,
			name TEXT NOT NULL,
			breed TEXT NOT NULL,
			pickup_location TEXT,
			walk_duration INTEGER,
			is_available INTEGER DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create dogs table: %v", err)
	}

	sqlxDB := sqlx.NewDb(rawDB, "sqlite")
	return database.WrapSqlxDB(sqlxDB, database.NewSQLiteDialect())
}

// createCalendarTestUser creates a test user and returns the user ID
func createCalendarTestUser(t *testing.T, db *database.DB) int {
	result, err := db.Exec(`
		INSERT INTO users (tenant_id, first_name, last_name, email, is_verified, is_active, terms_accepted_at)
		VALUES (0, 'Test', 'User', 'test@example.com', 1, 1, CURRENT_TIMESTAMP)
	`)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	id, _ := result.LastInsertId()
	return int(id)
}

// createCalendarTestDog creates a test dog and returns the dog ID
func createCalendarTestDog(t *testing.T, db *database.DB) int {
	result, err := db.Exec(`
		INSERT INTO dogs (tenant_id, name, breed, pickup_location, walk_duration) 
		VALUES (0, 'Buddy', 'Labrador', 'Tierheim Eingang', 45)
	`)
	if err != nil {
		t.Fatalf("Failed to create test dog: %v", err)
	}
	id, _ := result.LastInsertId()
	return int(id)
}

// createCalendarTestBooking creates a test booking
func createCalendarTestBooking(t *testing.T, db *database.DB, userID, dogID int, date string) int {
	result, err := db.Exec(`
		INSERT INTO bookings (tenant_id, user_id, dog_id, date, scheduled_time, status)
		VALUES (0, ?, ?, ?, '10:00', 'scheduled')
	`, userID, dogID, date)
	if err != nil {
		t.Fatalf("Failed to create test booking: %v", err)
	}
	id, _ := result.LastInsertId()
	return int(id)
}

// addCalendarUserContext adds user context to request
func addCalendarUserContext(req *http.Request, userID int) *http.Request {
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, "tenantID", 0)
	return req.WithContext(ctx)
}

func TestCalendarHandler_GetToken(t *testing.T) {
	db := setupCalendarTestDB(t)
	defer db.Close()

	cfg := &config.Config{
		BaseURL: "https://gassigeher.example.com",
	}
	handler := NewCalendarHandler(db, cfg)
	userID := createCalendarTestUser(t, db)

	tests := []struct {
		name           string
		userID         int
		setupContext   bool
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:           "get token for authenticated user",
			userID:         userID,
			setupContext:   true,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var resp CalendarTokenResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if resp.Token == "" {
					t.Error("Expected token to be non-empty")
				}
				if !resp.HasToken {
					t.Error("Expected HasToken to be true")
				}
				if !strings.Contains(resp.FeedURL, "/api/calendar/feed/") {
					t.Errorf("FeedURL should contain /api/calendar/feed/, got: %s", resp.FeedURL)
				}
				if !strings.HasPrefix(resp.WebcalURL, "webcal://") {
					t.Errorf("WebcalURL should start with webcal://, got: %s", resp.WebcalURL)
				}
			},
		},
		{
			name:           "unauthorized without context",
			userID:         0,
			setupContext:   false,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/calendar/token", nil)
			if tt.setupContext {
				req = addCalendarUserContext(req, tt.userID)
			}
			
			rr := httptest.NewRecorder()
			handler.GetToken(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, rr.Code, rr.Body.String())
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, rr.Body.Bytes())
			}
		})
	}
}

func TestCalendarHandler_GetToken_Consistency(t *testing.T) {
	db := setupCalendarTestDB(t)
	defer db.Close()

	cfg := &config.Config{
		BaseURL: "https://gassigeher.example.com",
	}
	handler := NewCalendarHandler(db, cfg)
	userID := createCalendarTestUser(t, db)

	// First request - should create token
	req1 := httptest.NewRequest("GET", "/api/calendar/token", nil)
	req1 = addCalendarUserContext(req1, userID)
	rr1 := httptest.NewRecorder()
	handler.GetToken(rr1, req1)

	var resp1 CalendarTokenResponse
	json.Unmarshal(rr1.Body.Bytes(), &resp1)

	// Second request - should return same token
	req2 := httptest.NewRequest("GET", "/api/calendar/token", nil)
	req2 = addCalendarUserContext(req2, userID)
	rr2 := httptest.NewRecorder()
	handler.GetToken(rr2, req2)

	var resp2 CalendarTokenResponse
	json.Unmarshal(rr2.Body.Bytes(), &resp2)

	if resp1.Token != resp2.Token {
		t.Errorf("Token should be consistent across requests. Got %s and %s", resp1.Token, resp2.Token)
	}
}

func TestCalendarHandler_RegenerateToken(t *testing.T) {
	db := setupCalendarTestDB(t)
	defer db.Close()

	cfg := &config.Config{
		BaseURL: "https://gassigeher.example.com",
	}
	handler := NewCalendarHandler(db, cfg)
	userID := createCalendarTestUser(t, db)

	// First, get initial token
	req1 := httptest.NewRequest("GET", "/api/calendar/token", nil)
	req1 = addCalendarUserContext(req1, userID)
	rr1 := httptest.NewRecorder()
	handler.GetToken(rr1, req1)

	var resp1 CalendarTokenResponse
	json.Unmarshal(rr1.Body.Bytes(), &resp1)
	oldToken := resp1.Token

	// Regenerate token
	req2 := httptest.NewRequest("POST", "/api/calendar/token/regenerate", nil)
	req2 = addCalendarUserContext(req2, userID)
	rr2 := httptest.NewRecorder()
	handler.RegenerateToken(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr2.Code)
	}

	var resp2 CalendarTokenResponse
	json.Unmarshal(rr2.Body.Bytes(), &resp2)

	if resp2.Token == oldToken {
		t.Error("Regenerated token should be different from old token")
	}

	if resp2.Token == "" {
		t.Error("New token should not be empty")
	}
}

func TestCalendarHandler_GetFeed(t *testing.T) {
	db := setupCalendarTestDB(t)
	defer db.Close()

	cfg := &config.Config{
		BaseURL: "https://gassigeher.example.com",
	}
	handler := NewCalendarHandler(db, cfg)
	userID := createCalendarTestUser(t, db)

	// First, get a token
	req1 := httptest.NewRequest("GET", "/api/calendar/token", nil)
	req1 = addCalendarUserContext(req1, userID)
	rr1 := httptest.NewRecorder()
	handler.GetToken(rr1, req1)

	var tokenResp CalendarTokenResponse
	json.Unmarshal(rr1.Body.Bytes(), &tokenResp)

	tests := []struct {
		name           string
		token          string
		expectedStatus int
		checkResponse  func(t *testing.T, body string, headers http.Header)
	}{
		{
			name:           "valid token returns iCal feed",
			token:          tokenResp.Token,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string, headers http.Header) {
				// Check content type
				contentType := headers.Get("Content-Type")
				if !strings.Contains(contentType, "text/calendar") {
					t.Errorf("Expected Content-Type text/calendar, got %s", contentType)
				}

				// Check iCal structure
				if !strings.Contains(body, "BEGIN:VCALENDAR") {
					t.Error("Response should contain BEGIN:VCALENDAR")
				}
				if !strings.Contains(body, "END:VCALENDAR") {
					t.Error("Response should contain END:VCALENDAR")
				}
				if !strings.Contains(body, "VERSION:2.0") {
					t.Error("Response should contain VERSION:2.0")
				}
			},
		},
		{
			name:           "invalid token returns 404",
			token:          "invalid-token-12345",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "empty token returns 400",
			token:          "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/calendar/feed/"+tt.token, nil)
			req = mux.SetURLVars(req, map[string]string{"token": tt.token})
			
			rr := httptest.NewRecorder()
			handler.GetFeed(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, rr.Code, rr.Body.String())
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, rr.Body.String(), rr.Header())
			}
		})
	}
}

func TestCalendarHandler_GetFeed_WithBookings(t *testing.T) {
	db := setupCalendarTestDB(t)
	defer db.Close()

	cfg := &config.Config{
		BaseURL: "https://gassigeher.example.com",
	}
	handler := NewCalendarHandler(db, cfg)
	userID := createCalendarTestUser(t, db)
	dogID := createCalendarTestDog(t, db)

	// Create a booking for tomorrow
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	createCalendarTestBooking(t, db, userID, dogID, tomorrow)

	// Get a token
	req1 := httptest.NewRequest("GET", "/api/calendar/token", nil)
	req1 = addCalendarUserContext(req1, userID)
	rr1 := httptest.NewRecorder()
	handler.GetToken(rr1, req1)

	var tokenResp CalendarTokenResponse
	json.Unmarshal(rr1.Body.Bytes(), &tokenResp)

	// Get feed
	req := httptest.NewRequest("GET", "/api/calendar/feed/"+tokenResp.Token, nil)
	req = mux.SetURLVars(req, map[string]string{"token": tokenResp.Token})
	
	rr := httptest.NewRecorder()
	handler.GetFeed(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	body := rr.Body.String()

	// Should contain a VEVENT for the booking
	if !strings.Contains(body, "BEGIN:VEVENT") {
		t.Error("Response should contain BEGIN:VEVENT for booking")
	}
	if !strings.Contains(body, "END:VEVENT") {
		t.Error("Response should contain END:VEVENT")
	}
	if !strings.Contains(body, "Buddy") {
		t.Error("Response should contain dog name 'Buddy'")
	}
}

func TestCalendarHandler_GetFeed_WithIcsExtension(t *testing.T) {
	db := setupCalendarTestDB(t)
	defer db.Close()

	cfg := &config.Config{
		BaseURL: "https://gassigeher.example.com",
	}
	handler := NewCalendarHandler(db, cfg)
	userID := createCalendarTestUser(t, db)

	// Get a token
	req1 := httptest.NewRequest("GET", "/api/calendar/token", nil)
	req1 = addCalendarUserContext(req1, userID)
	rr1 := httptest.NewRecorder()
	handler.GetToken(rr1, req1)

	var tokenResp CalendarTokenResponse
	json.Unmarshal(rr1.Body.Bytes(), &tokenResp)

	// Request with .ics extension
	tokenWithExt := tokenResp.Token + ".ics"
	req := httptest.NewRequest("GET", "/api/calendar/feed/"+tokenWithExt, nil)
	req = mux.SetURLVars(req, map[string]string{"token": tokenWithExt})
	
	rr := httptest.NewRecorder()
	handler.GetFeed(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 with .ics extension, got %d", rr.Code)
	}
}

func TestCalendarHandler_GetFeed_InactiveUser(t *testing.T) {
	db := setupCalendarTestDB(t)
	defer db.Close()

	cfg := &config.Config{
		BaseURL: "https://gassigeher.example.com",
	}
	handler := NewCalendarHandler(db, cfg)
	userID := createCalendarTestUser(t, db)

	// Get a token while user is active
	req1 := httptest.NewRequest("GET", "/api/calendar/token", nil)
	req1 = addCalendarUserContext(req1, userID)
	rr1 := httptest.NewRecorder()
	handler.GetToken(rr1, req1)

	var tokenResp CalendarTokenResponse
	json.Unmarshal(rr1.Body.Bytes(), &tokenResp)

	// Deactivate the user
	userRepo := repository.NewUserRepository(db)
	userRepo.Deactivate(userID, "test deactivation")

	// Try to get feed - should fail
	req := httptest.NewRequest("GET", "/api/calendar/feed/"+tokenResp.Token, nil)
	req = mux.SetURLVars(req, map[string]string{"token": tokenResp.Token})
	
	rr := httptest.NewRecorder()
	handler.GetFeed(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 for inactive user, got %d", rr.Code)
	}
}

func TestGenerateToken(t *testing.T) {
	token1, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken failed: %v", err)
	}

	if len(token1) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("Expected token length 64, got %d", len(token1))
	}

	// Tokens should be unique
	token2, _ := generateToken()
	if token1 == token2 {
		t.Error("Generated tokens should be unique")
	}
}

func TestEscapeICalText(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple text", "simple text"},
		{"text with, comma", "text with\\, comma"},
		{"text with; semicolon", "text with\\; semicolon"},
		{"text with\nnewline", "text with\\nnewline"},
		{"text with \\ backslash", "text with \\\\ backslash"},
		{"mixed, text; with\nspecial", "mixed\\, text\\; with\\nspecial"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := escapeICalText(tt.input)
			if result != tt.expected {
				t.Errorf("escapeICalText(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetICalStatus(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{"scheduled", "CONFIRMED"},
		{"pending_approval", "TENTATIVE"},
		{"cancelled", "CANCELLED"},
		{"completed", "CONFIRMED"},
		{"unknown", "CONFIRMED"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := getICalStatus(tt.status)
			if result != tt.expected {
				t.Errorf("getICalStatus(%q) = %q, want %q", tt.status, result, tt.expected)
			}
		})
	}
}

func TestFoldICalLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int // expected max line length
	}{
		{
			name:   "short line unchanged",
			input:  "SUMMARY:Short text",
			maxLen: 75,
		},
		{
			name:   "long line is folded",
			input:  "SUMMARY:This is a very long summary that should definitely be folded because it exceeds 75 characters",
			maxLen: 75,
		},
		{
			name:   "very long line with special chars",
			input:  "DESCRIPTION:Ein sehr langer deutscher Text mit Umlauten äöü und Sonderzeichen die alle korrekt behandelt werden müssen",
			maxLen: 75,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := foldICalLine(tt.input)
			
			// Check that result ends with CRLF
			if !strings.HasSuffix(result, "\r\n") {
				t.Error("folded line should end with CRLF")
			}

			// Check that no line exceeds maxLen (excluding CRLF)
			lines := strings.Split(strings.TrimSuffix(result, "\r\n"), "\r\n")
			for i, line := range lines {
				if len(line) > tt.maxLen {
					t.Errorf("line %d exceeds max length: %d > %d", i, len(line), tt.maxLen)
				}
			}

			// Continuation lines should start with space
			for i := 1; i < len(lines); i++ {
				if !strings.HasPrefix(lines[i], " ") {
					t.Errorf("continuation line %d should start with space", i)
				}
			}
		})
	}
}

func TestCalendarHandler_GetFeed_WithLocationAndDuration(t *testing.T) {
	db := setupCalendarTestDB(t)
	defer db.Close()

	cfg := &config.Config{
		BaseURL: "https://gassigeher.example.com",
	}
	handler := NewCalendarHandler(db, cfg)
	userID := createCalendarTestUser(t, db)
	dogID := createCalendarTestDog(t, db) // Has pickup_location and walk_duration

	// Create a booking for tomorrow
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	createCalendarTestBooking(t, db, userID, dogID, tomorrow)

	// Get a token
	req1 := httptest.NewRequest("GET", "/api/calendar/token", nil)
	req1 = addCalendarUserContext(req1, userID)
	rr1 := httptest.NewRecorder()
	handler.GetToken(rr1, req1)

	var tokenResp CalendarTokenResponse
	json.Unmarshal(rr1.Body.Bytes(), &tokenResp)

	// Get feed
	req := httptest.NewRequest("GET", "/api/calendar/feed/"+tokenResp.Token, nil)
	req = mux.SetURLVars(req, map[string]string{"token": tokenResp.Token})
	
	rr := httptest.NewRecorder()
	handler.GetFeed(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	body := rr.Body.String()

	// Should contain LOCATION from dog's pickup_location
	if !strings.Contains(body, "LOCATION:") {
		t.Error("Response should contain LOCATION field")
	}
	if !strings.Contains(body, "Tierheim Eingang") {
		t.Error("Response should contain dog's pickup location")
	}

	// Should contain dog breed in description
	if !strings.Contains(body, "Labrador") {
		t.Error("Response should contain dog breed in description")
	}
}

func TestCalendarHandler_RegenerateToken_NewToken(t *testing.T) {
	db := setupCalendarTestDB(t)
	defer db.Close()

	cfg := &config.Config{
		BaseURL: "https://gassigeher.example.com",
	}
	handler := NewCalendarHandler(db, cfg)
	userID := createCalendarTestUser(t, db)

	// Get initial token
	req1 := httptest.NewRequest("GET", "/api/calendar/token", nil)
	req1 = addCalendarUserContext(req1, userID)
	rr1 := httptest.NewRecorder()
	handler.GetToken(rr1, req1)

	var resp1 CalendarTokenResponse
	json.Unmarshal(rr1.Body.Bytes(), &resp1)

	// Old token should work
	req2 := httptest.NewRequest("GET", "/api/calendar/feed/"+resp1.Token, nil)
	req2 = mux.SetURLVars(req2, map[string]string{"token": resp1.Token})
	rr2 := httptest.NewRecorder()
	handler.GetFeed(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Error("Old token should work before regeneration")
	}

	// Regenerate token
	req3 := httptest.NewRequest("POST", "/api/calendar/token/regenerate", nil)
	req3 = addCalendarUserContext(req3, userID)
	rr3 := httptest.NewRecorder()
	handler.RegenerateToken(rr3, req3)

	var resp3 CalendarTokenResponse
	json.Unmarshal(rr3.Body.Bytes(), &resp3)

	// Old token should NOT work anymore
	req4 := httptest.NewRequest("GET", "/api/calendar/feed/"+resp1.Token, nil)
	req4 = mux.SetURLVars(req4, map[string]string{"token": resp1.Token})
	rr4 := httptest.NewRecorder()
	handler.GetFeed(rr4, req4)
	if rr4.Code == http.StatusOK {
		t.Error("Old token should NOT work after regeneration")
	}

	// New token should work
	req5 := httptest.NewRequest("GET", "/api/calendar/feed/"+resp3.Token, nil)
	req5 = mux.SetURLVars(req5, map[string]string{"token": resp3.Token})
	rr5 := httptest.NewRecorder()
	handler.GetFeed(rr5, req5)
	if rr5.Code != http.StatusOK {
		t.Error("New token should work after regeneration")
	}
}
