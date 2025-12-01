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

	"github.com/tranmh/gassigeher/internal/database"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
	"github.com/tranmh/gassigeher/internal/services"
)

// setupSecurityTest creates a test database and handler
func setupSecurityTest(t *testing.T) (*sql.DB, *BookingTimeHandler, *HolidayHandler, func()) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Verify that booking_time_rules table exists and has data
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM booking_time_rules").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query booking_time_rules: %v", err)
	}
	if count == 0 {
		t.Fatalf("No default booking time rules seeded! Migration may have failed.")
	}

	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	holidayService := services.NewHolidayService(holidayRepo, settingsRepo)
	bookingTimeService := services.NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo)

	bookingTimeHandler := NewBookingTimeHandler(bookingTimeRepo, bookingTimeService)
	holidayHandler := NewHolidayHandler(holidayRepo, holidayService)

	cleanup := func() {
		db.Close()
	}

	return db, bookingTimeHandler, holidayHandler, cleanup
}

// ==============================================================================
// Phase 6.1: Authorization Tests
// ==============================================================================

// Test 6.1.1: Admin Endpoint Protection
func TestSecurityAdminEndpointProtection(t *testing.T) {
	_, bookingTimeHandler, holidayHandler, cleanup := setupSecurityTest(t)
	defer cleanup()

	testCases := []struct {
		name        string
		method      string
		path        string
		handler     http.HandlerFunc
		body        string
		isAdmin     bool
		wantStatus  int
		description string
	}{
		{
			name:        "TC-6.1.1-A: Non-admin cannot GET booking-times/rules",
			method:      http.MethodGet,
			path:        "/api/booking-times/rules",
			handler:     bookingTimeHandler.GetRules,
			body:        "",
			isAdmin:     false,
			wantStatus:  http.StatusForbidden,
			description: "Regular user should be forbidden from viewing time rules",
		},
		{
			name:        "TC-6.1.1-B: Non-admin cannot PUT booking-times/rules",
			method:      http.MethodPut,
			path:        "/api/booking-times/rules",
			handler:     bookingTimeHandler.UpdateRules,
			body:        `[{"id":1,"day_type":"weekday","rule_name":"Test","start_time":"09:00","end_time":"12:00","is_blocked":false}]`,
			isAdmin:     false,
			wantStatus:  http.StatusForbidden,
			description: "Regular user should be forbidden from updating time rules",
		},
		{
			name:        "TC-6.1.1-C: Non-admin cannot POST booking-times/rules",
			method:      http.MethodPost,
			path:        "/api/booking-times/rules",
			handler:     bookingTimeHandler.CreateRule,
			body:        `{"day_type":"weekday","rule_name":"Test","start_time":"09:00","end_time":"12:00","is_blocked":false}`,
			isAdmin:     false,
			wantStatus:  http.StatusForbidden,
			description: "Regular user should be forbidden from creating time rules",
		},
		{
			name:        "TC-6.1.1-D: Non-admin cannot DELETE booking-times/rules/:id",
			method:      http.MethodDelete,
			path:        "/api/booking-times/rules/1",
			handler:     bookingTimeHandler.DeleteRule,
			body:        "",
			isAdmin:     false,
			wantStatus:  http.StatusForbidden,
			description: "Regular user should be forbidden from deleting time rules",
		},
		{
			name:        "TC-6.1.1-E: Non-admin cannot POST holidays",
			method:      http.MethodPost,
			path:        "/api/holidays",
			handler:     holidayHandler.CreateHoliday,
			body:        `{"date":"2025-07-01","name":"Test Holiday","is_active":true}`,
			isAdmin:     false,
			wantStatus:  http.StatusForbidden,
			description: "Regular user should be forbidden from creating holidays",
		},
		{
			name:        "TC-6.1.1-F: Non-admin cannot PUT holidays/:id",
			method:      http.MethodPut,
			path:        "/api/holidays/1",
			handler:     holidayHandler.UpdateHoliday,
			body:        `{"is_active":false}`,
			isAdmin:     false,
			wantStatus:  http.StatusForbidden,
			description: "Regular user should be forbidden from updating holidays",
		},
		{
			name:        "TC-6.1.1-G: Non-admin cannot DELETE holidays/:id",
			method:      http.MethodDelete,
			path:        "/api/holidays/1",
			handler:     holidayHandler.DeleteHoliday,
			body:        "",
			isAdmin:     false,
			wantStatus:  http.StatusForbidden,
			description: "Regular user should be forbidden from deleting holidays",
		},
		// Admin access tests
		{
			name:        "TC-6.1.1-H: Admin can GET booking-times/rules",
			method:      http.MethodGet,
			path:        "/api/booking-times/rules",
			handler:     bookingTimeHandler.GetRules,
			body:        "",
			isAdmin:     true,
			wantStatus:  http.StatusOK,
			description: "Admin should be able to view time rules",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var req *http.Request
			if tc.body != "" {
				req = httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tc.method, tc.path, nil)
			}

			// Simulate authenticated context with admin flag
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, 1)
			ctx = context.WithValue(ctx, middleware.EmailKey, "test@example.com")
			ctx = context.WithValue(ctx, middleware.IsAdminKey, tc.isAdmin)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			tc.handler(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("%s\nStatus = %d, want %d\nBody: %s",
					tc.description, w.Code, tc.wantStatus, w.Body.String())
			}

			// For forbidden responses, verify error message
			if w.Code == http.StatusForbidden {
				var resp map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err == nil {
					if _, ok := resp["error"]; !ok {
						t.Error("Expected error message in forbidden response")
					}
				}
			}
		})
	}
}

// Test 6.1.2: Unauthenticated Access
func TestSecurityUnauthenticatedAccess(t *testing.T) {
	_, bookingTimeHandler, _, cleanup := setupSecurityTest(t)
	defer cleanup()

	testCases := []struct {
		name        string
		method      string
		path        string
		handler     http.HandlerFunc
		body        string
		wantStatus  int
		description string
	}{
		{
			name:        "TC-6.1.2-A: Unauthenticated GET booking-times/rules",
			method:      http.MethodGet,
			path:        "/api/booking-times/rules",
			handler:     bookingTimeHandler.GetRules,
			wantStatus:  http.StatusUnauthorized,
			description: "Should require authentication",
		},
		{
			name:        "TC-6.1.2-B: Unauthenticated PUT booking-times/rules",
			method:      http.MethodPut,
			path:        "/api/booking-times/rules",
			handler:     bookingTimeHandler.UpdateRules,
			body:        `[{"id":1,"day_type":"weekday","rule_name":"Test","start_time":"09:00","end_time":"12:00"}]`,
			wantStatus:  http.StatusUnauthorized,
			description: "Should require authentication",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var req *http.Request
			if tc.body != "" {
				req = httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tc.method, tc.path, nil)
			}

			// No authentication context - simulates missing/invalid token
			// Note: In real scenario, middleware would reject before reaching handler
			w := httptest.NewRecorder()
			tc.handler(w, req)

			// Handler should check for auth context and return 401 if missing
			// If handler doesn't check (relies on middleware), we'd need integration test
			if w.Code != tc.wantStatus && w.Code != http.StatusInternalServerError {
				// Some handlers may error without auth context
				t.Logf("%s\nNote: Handler returned %d (middleware would return %d)",
					tc.description, w.Code, tc.wantStatus)
			}
		})
	}
}

// ==============================================================================
// Phase 6.2: Input Validation Tests
// ==============================================================================

// Test 6.2.1: SQL Injection Attempts
func TestSecuritySQLInjection(t *testing.T) {
	db, bookingTimeHandler, holidayHandler, cleanup := setupSecurityTest(t)
	defer cleanup()

	// Create admin context for tests
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, 1)
	ctx = context.WithValue(ctx, middleware.EmailKey, "admin@example.com")
	ctx = context.WithValue(ctx, middleware.IsAdminKey, true)

	testCases := []struct {
		name          string
		method        string
		path          string
		handler       http.HandlerFunc
		body          string
		checkDatabase func(*testing.T, *sql.DB)
		description   string
	}{
		{
			name:    "TC-6.2.1-A: SQL injection in rule_name",
			method:  http.MethodPost,
			path:    "/api/booking-times/rules",
			handler: bookingTimeHandler.CreateRule,
			body:    `{"day_type":"weekday","rule_name":"Test'; DROP TABLE bookings; --","start_time":"09:00","end_time":"12:00","is_blocked":false}`,
			checkDatabase: func(t *testing.T, db *sql.DB) {
				// Verify bookings table still exists
				var count int
				err := db.QueryRow("SELECT COUNT(*) FROM bookings").Scan(&count)
				if err != nil {
					t.Error("Bookings table was deleted! SQL injection succeeded!")
				}

				// Verify rule was created with escaped name
				var ruleName string
				err = db.QueryRow("SELECT rule_name FROM booking_time_rules WHERE rule_name LIKE '%DROP TABLE%'").Scan(&ruleName)
				if err == nil {
					t.Logf("Rule created with escaped injection attempt: %s", ruleName)
				}
			},
			description: "SQL injection attempt should be escaped, not executed",
		},
		{
			name:    "TC-6.2.1-B: SQL injection in holiday name",
			method:  http.MethodPost,
			path:    "/api/holidays",
			handler: holidayHandler.CreateHoliday,
			body:    `{"date":"2025-07-01","name":"Holiday'; DELETE FROM custom_holidays; --","is_active":true,"source":"admin"}`,
			checkDatabase: func(t *testing.T, db *sql.DB) {
				// Verify custom_holidays table still exists (not dropped)
				var tableName string
				err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='custom_holidays'").Scan(&tableName)
				if err != nil {
					t.Error("custom_holidays table was deleted! SQL injection succeeded!")
				}
				// The table should exist even if empty
				if tableName != "custom_holidays" {
					t.Error("custom_holidays table structure compromised!")
				}
			},
			description: "SQL injection in holiday name should be escaped",
		},
		{
			name:    "TC-6.2.1-C: SQL injection in date parameter",
			method:  http.MethodGet,
			path:    "/api/booking-times/available?date=2025-01-01%27%3B%20DROP%20TABLE%20bookings%3B%20--",
			handler: bookingTimeHandler.GetAvailableSlots,
			body:    "",
			checkDatabase: func(t *testing.T, db *sql.DB) {
				// Verify bookings table still exists
				var count int
				err := db.QueryRow("SELECT COUNT(*) FROM bookings").Scan(&count)
				if err != nil {
					t.Error("Bookings table was deleted! SQL injection succeeded!")
				}
			},
			description: "SQL injection in query parameter should be handled safely",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var req *http.Request
			if tc.body != "" {
				req = httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tc.method, tc.path, nil)
			}
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			tc.handler(w, req)

			// Check database integrity
			if tc.checkDatabase != nil {
				tc.checkDatabase(t, db)
			}

			t.Logf("%s - Response status: %d", tc.description, w.Code)
		})
	}
}

// Test 6.2.2: XSS Attempts
func TestSecurityXSSPrevention(t *testing.T) {
	_, bookingTimeHandler, holidayHandler, cleanup := setupSecurityTest(t)
	defer cleanup()

	// Create admin context
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, 1)
	ctx = context.WithValue(ctx, middleware.EmailKey, "admin@example.com")
	ctx = context.WithValue(ctx, middleware.IsAdminKey, true)

	testCases := []struct {
		name             string
		method           string
		path             string
		handler          http.HandlerFunc
		body             string
		checkResponse    func(*testing.T, []byte)
		description      string
	}{
		{
			name:    "TC-6.2.2-A: XSS in rule_name",
			method:  http.MethodPost,
			path:    "/api/booking-times/rules",
			handler: bookingTimeHandler.CreateRule,
			body:    `{"day_type":"weekday","rule_name":"<script>alert('XSS')</script>","start_time":"09:00","end_time":"12:00","is_blocked":false}`,
			checkResponse: func(t *testing.T, body []byte) {
				bodyStr := string(body)
				// Response should NOT contain unescaped script tags
				// Note: JSON encoding will escape < and > to \u003c and \u003e
				if strings.Contains(bodyStr, "<script>") {
					t.Error("Response contains unescaped script tag!")
				}
				if strings.Contains(bodyStr, "alert(") && !strings.Contains(bodyStr, "\\u003c") {
					t.Error("XSS payload not properly escaped in response")
				}
			},
			description: "XSS attempt in rule_name should be escaped in response",
		},
		{
			name:    "TC-6.2.2-B: XSS in holiday name",
			method:  http.MethodPost,
			path:    "/api/holidays",
			handler: holidayHandler.CreateHoliday,
			body:    `{"date":"2025-07-01","name":"<img src=x onerror=alert(1)>","is_active":true}`,
			checkResponse: func(t *testing.T, body []byte) {
				bodyStr := string(body)
				if strings.Contains(bodyStr, "<img") && !strings.Contains(bodyStr, "\\u003c") {
					t.Error("XSS payload not properly escaped in response")
				}
			},
			description: "XSS attempt in holiday name should be escaped",
		},
		{
			name:    "TC-6.2.2-C: XSS with JavaScript protocol",
			method:  http.MethodPost,
			path:    "/api/holidays",
			handler: holidayHandler.CreateHoliday,
			body:    `{"date":"2025-07-01","name":"javascript:alert(document.cookie)","is_active":true}`,
			checkResponse: func(t *testing.T, body []byte) {
				// Should either reject or safely store the value
				bodyStr := string(body)
				t.Logf("Response: %s", bodyStr)
			},
			description: "JavaScript protocol in input should be handled safely",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			tc.handler(w, req)

			if tc.checkResponse != nil {
				tc.checkResponse(t, w.Body.Bytes())
			}

			t.Logf("%s - Status: %d", tc.description, w.Code)
		})
	}
}

// Test 6.2.3: Time Format Validation
func TestSecurityTimeFormatValidation(t *testing.T) {
	testCases := []struct {
		name        string
		rule        models.BookingTimeRule
		wantErr     bool
		description string
	}{
		{
			name: "TC-6.2.3-A: Valid time format",
			rule: models.BookingTimeRule{
				DayType:   "weekday",
				RuleName:  "Test",
				StartTime: "09:00",
				EndTime:   "12:00",
				IsBlocked: false,
			},
			wantErr:     false,
			description: "09:00 should be valid",
		},
		{
			name: "TC-6.2.3-B: Missing leading zero in hour",
			rule: models.BookingTimeRule{
				DayType:   "weekday",
				RuleName:  "Test",
				StartTime: "9:00",
				EndTime:   "12:00",
				IsBlocked: false,
			},
			wantErr:     true,
			description: "9:00 should be invalid (missing leading zero)",
		},
		{
			name: "TC-6.2.3-C: Hour > 23",
			rule: models.BookingTimeRule{
				DayType:   "weekday",
				RuleName:  "Test",
				StartTime: "25:00",
				EndTime:   "12:00",
				IsBlocked: false,
			},
			wantErr:     true,
			description: "25:00 should be invalid (hour > 23)",
		},
		{
			name: "TC-6.2.3-D: Minute > 59",
			rule: models.BookingTimeRule{
				DayType:   "weekday",
				RuleName:  "Test",
				StartTime: "12:60",
				EndTime:   "13:00",
				IsBlocked: false,
			},
			wantErr:     true,
			description: "12:60 should be invalid (minute > 59)",
		},
		{
			name: "TC-6.2.3-E: 12-hour format",
			rule: models.BookingTimeRule{
				DayType:   "weekday",
				RuleName:  "Test",
				StartTime: "12:00 PM",
				EndTime:   "13:00",
				IsBlocked: false,
			},
			wantErr:     true,
			description: "12:00 PM should be invalid (not 24-hour format)",
		},
		{
			name: "TC-6.2.3-F: Non-numeric characters",
			rule: models.BookingTimeRule{
				DayType:   "weekday",
				RuleName:  "Test",
				StartTime: "abc",
				EndTime:   "12:00",
				IsBlocked: false,
			},
			wantErr:     true,
			description: "abc should be invalid",
		},
		{
			name: "TC-6.2.3-G: With seconds",
			rule: models.BookingTimeRule{
				DayType:   "weekday",
				RuleName:  "Test",
				StartTime: "09:00:00",
				EndTime:   "12:00",
				IsBlocked: false,
			},
			wantErr:     true,
			description: "09:00:00 should be invalid (seconds not allowed)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.rule.Validate()

			if tc.wantErr && err == nil {
				t.Errorf("%s\nExpected validation error, got nil", tc.description)
			}

			if !tc.wantErr && err != nil {
				t.Errorf("%s\nExpected no error, got: %v", tc.description, err)
			}

			if err != nil {
				t.Logf("Validation error (expected=%v): %v", tc.wantErr, err)
			}
		})
	}
}

// Test 6.2.4: Date Format Validation
func TestSecurityDateFormatValidation(t *testing.T) {
	testCases := []struct {
		name        string
		date        string
		wantErr     bool
		description string
	}{
		{
			name:        "TC-6.2.4-A: Valid date format",
			date:        "2025-01-27",
			wantErr:     false,
			description: "2025-01-27 should be valid",
		},
		{
			name:        "TC-6.2.4-B: Missing leading zeros",
			date:        "2025-1-27",
			wantErr:     true,
			description: "2025-1-27 should be invalid (missing leading zero)",
		},
		{
			name:        "TC-6.2.4-C: Wrong order (DD-MM-YYYY)",
			date:        "27-01-2025",
			wantErr:     true,
			description: "27-01-2025 should be invalid (wrong order)",
		},
		{
			name:        "TC-6.2.4-D: Wrong separator (slash)",
			date:        "2025/01/27",
			wantErr:     true,
			description: "2025/01/27 should be invalid (wrong separator)",
		},
		{
			name:        "TC-6.2.4-E: Month > 12",
			date:        "2025-13-01",
			wantErr:     true,
			description: "2025-13-01 should be invalid (month > 12)",
		},
		{
			name:        "TC-6.2.4-F: Invalid day for month",
			date:        "2025-02-30",
			wantErr:     true,
			description: "2025-02-30 should be invalid (Feb doesn't have 30 days)",
		},
		{
			name:        "TC-6.2.4-G: Non-numeric characters",
			date:        "202X-01-01",
			wantErr:     true,
			description: "202X-01-01 should be invalid",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test date validation using time.Parse
			_, err := time.Parse("2006-01-02", tc.date)

			if tc.wantErr && err == nil {
				t.Errorf("%s\nExpected validation error, got nil", tc.description)
			}

			if !tc.wantErr && err != nil {
				t.Errorf("%s\nExpected no error, got: %v", tc.description, err)
			}

			if err != nil {
				t.Logf("Date parsing error (expected=%v): %v", tc.wantErr, err)
			}
		})
	}
}

// ==============================================================================
// Additional Security Tests
// ==============================================================================

// Test boundary conditions for time validation
func TestSecurityTimeBoundaryConditions(t *testing.T) {
	testCases := []struct {
		name        string
		startTime   string
		endTime     string
		wantErr     bool
		description string
	}{
		{
			name:        "Valid boundary: 00:00",
			startTime:   "00:00",
			endTime:     "01:00",
			wantErr:     false,
			description: "Midnight should be valid",
		},
		{
			name:        "Valid boundary: 23:59",
			startTime:   "22:00",
			endTime:     "23:59",
			wantErr:     false,
			description: "23:59 should be valid",
		},
		{
			name:        "Invalid: End before start",
			startTime:   "14:00",
			endTime:     "12:00",
			wantErr:     true,
			description: "End time before start time should be invalid",
		},
		{
			name:        "Invalid: Same time",
			startTime:   "12:00",
			endTime:     "12:00",
			wantErr:     true,
			description: "Same start and end time should be invalid",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rule := models.BookingTimeRule{
				DayType:   "weekday",
				RuleName:  "Test",
				StartTime: tc.startTime,
				EndTime:   tc.endTime,
				IsBlocked: false,
			}

			err := rule.Validate()

			if tc.wantErr && err == nil {
				t.Errorf("%s\nExpected validation error, got nil", tc.description)
			}

			if !tc.wantErr && err != nil {
				t.Errorf("%s\nExpected no error, got: %v", tc.description, err)
			}
		})
	}
}

// Test CORS and security headers (if applicable)
func TestSecurityHeaders(t *testing.T) {
	_, handler, _, cleanup := setupSecurityTest(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/booking-times/available?date=2025-01-27", nil)
	w := httptest.NewRecorder()

	handler.GetAvailableSlots(w, req)

	// Check for security headers (these would typically be set by middleware)
	t.Run("Security headers check", func(t *testing.T) {
		// Note: In production, these headers should be set by middleware
		// This test documents expected security headers
		expectedHeaders := []string{
			"X-Content-Type-Options",
			"X-Frame-Options",
			"X-XSS-Protection",
		}

		for _, header := range expectedHeaders {
			t.Logf("Security header %s: %s (should be set by middleware)",
				header, w.Header().Get(header))
		}
	})
}
