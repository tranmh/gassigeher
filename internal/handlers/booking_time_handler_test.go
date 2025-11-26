package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/tranm/gassigeher/internal/database"
	"github.com/tranm/gassigeher/internal/models"
	"github.com/tranm/gassigeher/internal/repository"
	"github.com/tranm/gassigeher/internal/services"
)

func setupBookingTimeHandlerTest(t *testing.T) (*sql.DB, *BookingTimeHandler, func()) {
	// Create in-memory test database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Run migrations
	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Setup repositories and services
	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	holidayService := services.NewHolidayService(holidayRepo, settingsRepo)
	bookingTimeService := services.NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo)
	handler := NewBookingTimeHandler(bookingTimeRepo, bookingTimeService)

	cleanup := func() {
		db.Close()
	}

	return db, handler, cleanup
}

// Test 3.1.1: GET /api/booking-times/available
func TestGetAvailableSlots(t *testing.T) {
	_, handler, cleanup := setupBookingTimeHandlerTest(t)
	defer cleanup()

	testCases := []struct {
		name       string
		query      string
		wantStatus int
		checkBody  func(*testing.T, []byte)
	}{
		{
			name:       "TC-3.1.1-A: Valid weekday",
			query:      "?date=2025-01-27",
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				slots, ok := resp["slots"].([]interface{})
				if !ok {
					t.Error("Expected slots array in response")
					return
				}
				if len(slots) == 0 {
					t.Error("Expected slots, got empty array")
				}
				// Verify date in response
				date, ok := resp["date"].(string)
				if !ok || date != "2025-01-27" {
					t.Errorf("Expected date 2025-01-27, got %v", date)
				}
			},
		},
		{
			name:       "TC-3.1.1-B: Valid weekend",
			query:      "?date=2025-01-25",
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				slots, ok := resp["slots"].([]interface{})
				if !ok || len(slots) == 0 {
					t.Error("Expected weekend slots")
				}
			},
		},
		{
			name:       "TC-3.1.1-C: Missing date parameter",
			query:      "",
			wantStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if _, ok := resp["error"]; !ok {
					t.Error("Expected error message in response")
				}
			},
		},
		{
			name:       "TC-3.1.1-D: Invalid date format",
			query:      "?date=invalid-date",
			wantStatus: http.StatusBadRequest,
			checkBody:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/booking-times/available"+tc.query, nil)
			w := httptest.NewRecorder()

			handler.GetAvailableSlots(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("Status = %d, want %d. Body: %s", w.Code, tc.wantStatus, w.Body.String())
			}

			if tc.checkBody != nil {
				tc.checkBody(t, w.Body.Bytes())
			}
		})
	}
}

// Test 3.1.2: GET /api/booking-times/rules
func TestGetRules(t *testing.T) {
	_, handler, cleanup := setupBookingTimeHandlerTest(t)
	defer cleanup()

	testCases := []struct {
		name          string
		authHeader    string
		isAdmin       bool
		wantStatus    int
		checkResponse func(*testing.T, []byte)
	}{
		{
			name:       "TC-3.1.2-A: Admin can get rules",
			authHeader: "Bearer valid-token",
			isAdmin:    true,
			wantStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var rules map[string][]models.BookingTimeRule
				if err := json.Unmarshal(body, &rules); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if len(rules) == 0 {
					t.Error("Expected rules, got empty map")
				}
				// Check that we have both weekday and weekend rules
				if _, ok := rules["weekday"]; !ok {
					t.Error("Expected weekday rules")
				}
				if _, ok := rules["weekend"]; !ok {
					t.Error("Expected weekend rules")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/booking-times/rules", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}

			// Set up admin context for the test
			ctx := contextWithUser(req.Context(), 1, "admin@example.com", tc.isAdmin)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			handler.GetRules(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("Status = %d, want %d. Body: %s", w.Code, tc.wantStatus, w.Body.String())
			}

			if tc.checkResponse != nil {
				tc.checkResponse(t, w.Body.Bytes())
			}
		})
	}
}

// Test 3.1.3: PUT /api/booking-times/rules
func TestUpdateRules(t *testing.T) {
	db, handler, cleanup := setupBookingTimeHandlerTest(t)
	defer cleanup()

	// Get existing rule to update
	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	rules, err := bookingTimeRepo.GetRulesByDayType("weekday")
	if err != nil || len(rules) == 0 {
		t.Fatalf("Failed to get existing rules: %v", err)
	}
	existingRule := rules[0]

	testCases := []struct {
		name       string
		rules      []models.BookingTimeRule
		wantStatus int
		checkError func(*testing.T, []byte)
	}{
		{
			name: "TC-3.1.3-A: Valid rule update",
			rules: []models.BookingTimeRule{
				{
					ID:        existingRule.ID,
					DayType:   existingRule.DayType,
					RuleName:  existingRule.RuleName,
					StartTime: "09:00",
					EndTime:   "11:00", // Changed
					IsBlocked: existingRule.IsBlocked,
				},
			},
			wantStatus: http.StatusOK,
			checkError: nil,
		},
		{
			name: "TC-3.1.3-B: Invalid time format",
			rules: []models.BookingTimeRule{
				{
					ID:        existingRule.ID,
					DayType:   existingRule.DayType,
					RuleName:  existingRule.RuleName,
					StartTime: "9:00", // Invalid format (missing leading zero)
					EndTime:   "11:00",
					IsBlocked: existingRule.IsBlocked,
				},
			},
			wantStatus: http.StatusBadRequest,
			checkError: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				if err := json.Unmarshal(body, &resp); err == nil {
					if _, ok := resp["error"]; !ok {
						t.Error("Expected error message")
					}
				}
			},
		},
		{
			name: "TC-3.1.3-C: End time before start time",
			rules: []models.BookingTimeRule{
				{
					ID:        existingRule.ID,
					DayType:   existingRule.DayType,
					RuleName:  existingRule.RuleName,
					StartTime: "14:00",
					EndTime:   "12:00", // Before start time
					IsBlocked: existingRule.IsBlocked,
				},
			},
			wantStatus: http.StatusBadRequest,
			checkError: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				if err := json.Unmarshal(body, &resp); err == nil {
					if _, ok := resp["error"]; !ok {
						t.Error("Expected error message")
					}
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.rules)
			req := httptest.NewRequest(http.MethodPut, "/api/booking-times/rules", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			// Set up admin context
			ctx := contextWithUser(req.Context(), 1, "admin@example.com", true)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			handler.UpdateRules(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("Status = %d, want %d. Body: %s", w.Code, tc.wantStatus, w.Body.String())
			}

			if tc.checkError != nil {
				tc.checkError(t, w.Body.Bytes())
			}
		})
	}
}

// Test 3.1.4: POST /api/booking-times/rules
func TestCreateRule(t *testing.T) {
	_, handler, cleanup := setupBookingTimeHandlerTest(t)
	defer cleanup()

	testCases := []struct {
		name       string
		rule       models.BookingTimeRule
		wantStatus int
		checkResp  func(*testing.T, []byte)
	}{
		{
			name: "TC-3.1.4-A: Valid new rule",
			rule: models.BookingTimeRule{
				DayType:   "weekday",
				RuleName:  "Test New Rule",
				StartTime: "20:00",
				EndTime:   "21:00",
				IsBlocked: false,
			},
			wantStatus: http.StatusCreated,
			checkResp: func(t *testing.T, body []byte) {
				var rule models.BookingTimeRule
				if err := json.Unmarshal(body, &rule); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if rule.ID == 0 {
					t.Error("Expected rule ID to be assigned")
				}
				if rule.RuleName != "Test New Rule" {
					t.Errorf("Expected RuleName 'Test New Rule', got %s", rule.RuleName)
				}
			},
		},
		{
			name: "TC-3.1.4-C: Missing required field",
			rule: models.BookingTimeRule{
				DayType:   "", // Missing
				RuleName:  "Test",
				StartTime: "09:00",
				EndTime:   "10:00",
				IsBlocked: false,
			},
			wantStatus: http.StatusBadRequest,
			checkResp:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.rule)
			req := httptest.NewRequest(http.MethodPost, "/api/booking-times/rules", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			// Set up admin context
			ctx := contextWithUser(req.Context(), 1, "admin@example.com", true)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			handler.CreateRule(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("Status = %d, want %d. Body: %s", w.Code, tc.wantStatus, w.Body.String())
			}

			if tc.checkResp != nil {
				tc.checkResp(t, w.Body.Bytes())
			}
		})
	}
}

// Test 3.1.5: DELETE /api/booking-times/rules/:id
func TestDeleteRule(t *testing.T) {
	db, handler, cleanup := setupBookingTimeHandlerTest(t)
	defer cleanup()

	// Create a test rule to delete
	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	testRule := &models.BookingTimeRule{
		DayType:   "weekday",
		RuleName:  "Test Delete Rule",
		StartTime: "20:00",
		EndTime:   "21:00",
		IsBlocked: false,
	}
	if err := bookingTimeRepo.CreateRule(testRule); err != nil {
		t.Fatalf("Failed to create test rule: %v", err)
	}

	testCases := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{
			name:       "TC-3.1.5-A: Delete existing rule",
			path:       "/api/booking-times/rules/" + strconv.Itoa(testRule.ID),
			wantStatus: http.StatusOK,
		},
		{
			name:       "TC-3.1.5-B: Delete non-existent rule",
			path:       "/api/booking-times/rules/99999",
			wantStatus: http.StatusOK, // Idempotent
		},
		{
			name:       "TC-3.1.5-C: Invalid ID format",
			path:       "/api/booking-times/rules/invalid",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, tc.path, nil)

			// Set up admin context
			ctx := contextWithUser(req.Context(), 1, "admin@example.com", true)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			handler.DeleteRule(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("Status = %d, want %d. Body: %s", w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}

// Test GetRulesForDate endpoint
func TestGetRulesForDate(t *testing.T) {
	_, handler, cleanup := setupBookingTimeHandlerTest(t)
	defer cleanup()

	testCases := []struct {
		name       string
		query      string
		wantStatus int
		checkBody  func(*testing.T, []byte)
	}{
		{
			name:       "Valid weekday date",
			query:      "?date=2025-01-27",
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var rules []models.BookingTimeRule
				if err := json.Unmarshal(body, &rules); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if len(rules) == 0 {
					t.Error("Expected rules for weekday")
				}
				// Verify we got weekday rules
				foundWeekdayRule := false
				for _, rule := range rules {
					if rule.DayType == "weekday" {
						foundWeekdayRule = true
						break
					}
				}
				if !foundWeekdayRule {
					t.Error("Expected weekday rules")
				}
			},
		},
		{
			name:       "Valid weekend date",
			query:      "?date=2025-01-25",
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var rules []models.BookingTimeRule
				if err := json.Unmarshal(body, &rules); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				// Verify we got weekend rules
				foundWeekendRule := false
				for _, rule := range rules {
					if rule.DayType == "weekend" {
						foundWeekendRule = true
						break
					}
				}
				if !foundWeekendRule {
					t.Error("Expected weekend rules")
				}
			},
		},
		{
			name:       "Missing date parameter",
			query:      "",
			wantStatus: http.StatusBadRequest,
			checkBody:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/booking-times/rules-for-date"+tc.query, nil)
			w := httptest.NewRecorder()

			handler.GetRulesForDate(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("Status = %d, want %d. Body: %s", w.Code, tc.wantStatus, w.Body.String())
			}

			if tc.checkBody != nil {
				tc.checkBody(t, w.Body.Bytes())
			}
		})
	}
}

// Benchmark for GetAvailableSlots performance
func BenchmarkGetAvailableSlots(b *testing.B) {
	_, handler, cleanup := setupBookingTimeHandlerTest(&testing.T{})
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/booking-times/available?date=2025-01-27", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.GetAvailableSlots(w, req)
	}
}
