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

func setupHolidayHandlerTest(t *testing.T) (*sql.DB, *HolidayHandler, func()) {
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
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	holidayService := services.NewHolidayService(holidayRepo, settingsRepo)
	handler := NewHolidayHandler(holidayRepo, holidayService)

	cleanup := func() {
		db.Close()
	}

	return db, handler, cleanup
}

// Test 3.2.1: GET /api/holidays
func TestGetHolidays(t *testing.T) {
	db, handler, cleanup := setupHolidayHandlerTest(t)
	defer cleanup()

	// Seed some test holidays
	holidayRepo := repository.NewHolidayRepository(db)
	testHolidays := []models.CustomHoliday{
		{
			Date:     "2025-01-01",
			Name:     "Neujahrstag",
			IsActive: true,
			Source:   "api",
		},
		{
			Date:     "2025-12-25",
			Name:     "Weihnachten",
			IsActive: true,
			Source:   "api",
		},
		{
			Date:     "2026-01-01",
			Name:     "Neujahrstag 2026",
			IsActive: true,
			Source:   "api",
		},
	}

	for _, holiday := range testHolidays {
		h := holiday
		if err := holidayRepo.CreateHoliday(&h); err != nil {
			t.Fatalf("Failed to create test holiday: %v", err)
		}
	}

	testCases := []struct {
		name       string
		query      string
		wantStatus int
		checkResp  func(*testing.T, []byte)
	}{
		{
			name:       "TC-3.2.1-A: Get holidays for 2025",
			query:      "?year=2025",
			wantStatus: http.StatusOK,
			checkResp: func(t *testing.T, body []byte) {
				var holidays []models.CustomHoliday
				if err := json.Unmarshal(body, &holidays); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if len(holidays) < 2 {
					t.Errorf("Expected at least 2 holidays for 2025, got %d", len(holidays))
				}
				// Verify all are from 2025
				for _, h := range holidays {
					if !contains2025(h.Date) {
						t.Errorf("Expected 2025 holiday, got %s", h.Date)
					}
				}
			},
		},
		{
			name:       "TC-3.2.1-B: Get holidays for 2026",
			query:      "?year=2026",
			wantStatus: http.StatusOK,
			checkResp: func(t *testing.T, body []byte) {
				var holidays []models.CustomHoliday
				if err := json.Unmarshal(body, &holidays); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				// Should have at least 1 holiday for 2026
				if len(holidays) < 1 {
					t.Errorf("Expected at least 1 holiday for 2026, got %d", len(holidays))
				}
			},
		},
		{
			name:       "TC-3.2.1-C: No year parameter (defaults to current year)",
			query:      "",
			wantStatus: http.StatusOK,
			checkResp: func(t *testing.T, body []byte) {
				var holidays []models.CustomHoliday
				if err := json.Unmarshal(body, &holidays); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				// Should return array (may be empty for current year)
			},
		},
		{
			name:       "TC-3.2.1-D: Invalid year parameter",
			query:      "?year=invalid",
			wantStatus: http.StatusOK, // Falls back to current year
			checkResp:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/holidays"+tc.query, nil)
			w := httptest.NewRecorder()

			handler.GetHolidays(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("Status = %d, want %d. Body: %s", w.Code, tc.wantStatus, w.Body.String())
			}

			if tc.checkResp != nil {
				tc.checkResp(t, w.Body.Bytes())
			}
		})
	}
}

// Test 3.2.2: POST /api/holidays
func TestCreateHoliday(t *testing.T) {
	_, handler, cleanup := setupHolidayHandlerTest(t)
	defer cleanup()

	testCases := []struct {
		name       string
		holiday    models.CustomHoliday
		adminID    int
		wantStatus int
		checkResp  func(*testing.T, []byte)
	}{
		{
			name: "TC-3.2.2-A: Valid holiday",
			holiday: models.CustomHoliday{
				Date:     "2025-07-01",
				Name:     "Custom Holiday",
				IsActive: true,
			},
			adminID:    1,
			wantStatus: http.StatusCreated,
			checkResp: func(t *testing.T, body []byte) {
				var holiday models.CustomHoliday
				if err := json.Unmarshal(body, &holiday); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if holiday.ID == 0 {
					t.Error("Expected holiday ID to be assigned")
				}
				if holiday.Source != "admin" {
					t.Errorf("Expected source 'admin', got %s", holiday.Source)
				}
			},
		},
		{
			name: "TC-3.2.2-B: Invalid date format",
			holiday: models.CustomHoliday{
				Date:     "01-07-2025", // Wrong format
				Name:     "Test Holiday",
				IsActive: true,
			},
			adminID:    1,
			wantStatus: http.StatusBadRequest,
			checkResp:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.holiday)
			req := httptest.NewRequest(http.MethodPost, "/api/holidays", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			// Set admin context
			ctx := contextWithUser(req.Context(), tc.adminID, "admin@example.com", true)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			handler.CreateHoliday(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("Status = %d, want %d. Body: %s", w.Code, tc.wantStatus, w.Body.String())
			}

			if tc.checkResp != nil {
				tc.checkResp(t, w.Body.Bytes())
			}
		})
	}
}

// Test 3.2.3: PUT /api/holidays/:id
func TestUpdateHoliday(t *testing.T) {
	db, handler, cleanup := setupHolidayHandlerTest(t)
	defer cleanup()

	// Create a test holiday to update
	holidayRepo := repository.NewHolidayRepository(db)
	testHoliday := &models.CustomHoliday{
		Date:     "2025-08-01",
		Name:     "Test Holiday",
		IsActive: true,
		Source:   "admin",
	}
	if err := holidayRepo.CreateHoliday(testHoliday); err != nil {
		t.Fatalf("Failed to create test holiday: %v", err)
	}

	testCases := []struct {
		name       string
		holidayID  int
		update     models.CustomHoliday
		wantStatus int
		checkResp  func(*testing.T, []byte)
	}{
		{
			name:      "TC-3.2.3-A: Toggle is_active",
			holidayID: testHoliday.ID,
			update: models.CustomHoliday{
				Date:     testHoliday.Date,
				Name:     testHoliday.Name,
				IsActive: false, // Changed
				Source:   testHoliday.Source,
			},
			wantStatus: http.StatusOK,
			checkResp: func(t *testing.T, body []byte) {
				var resp map[string]string
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if resp["message"] == "" {
					t.Error("Expected success message")
				}
			},
		},
		{
			name:      "TC-3.2.3-B: Change name",
			holidayID: testHoliday.ID,
			update: models.CustomHoliday{
				Date:     testHoliday.Date,
				Name:     "Updated Holiday Name",
				IsActive: testHoliday.IsActive,
				Source:   testHoliday.Source,
			},
			wantStatus: http.StatusOK,
			checkResp:  nil,
		},
		{
			name:      "TC-3.2.3-C: Non-existent ID",
			holidayID: 99999,
			update: models.CustomHoliday{
				Date:     "2025-09-01",
				Name:     "Test",
				IsActive: true,
				Source:   "admin",
			},
			wantStatus: http.StatusOK, // No rows affected, but not an error
			checkResp:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.update)
			path := "/api/holidays/" + strconv.Itoa(tc.holidayID)
			req := httptest.NewRequest(http.MethodPut, path, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			// Set admin context
			ctx := contextWithUser(req.Context(), 1, "admin@example.com", true)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			handler.UpdateHoliday(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("Status = %d, want %d. Body: %s", w.Code, tc.wantStatus, w.Body.String())
			}

			if tc.checkResp != nil {
				tc.checkResp(t, w.Body.Bytes())
			}
		})
	}
}

// Test 3.2.4: DELETE /api/holidays/:id
func TestDeleteHoliday(t *testing.T) {
	db, handler, cleanup := setupHolidayHandlerTest(t)
	defer cleanup()

	// Create test holidays
	holidayRepo := repository.NewHolidayRepository(db)
	adminHoliday := &models.CustomHoliday{
		Date:     "2025-10-01",
		Name:     "Admin Created",
		IsActive: true,
		Source:   "admin",
	}
	apiHoliday := &models.CustomHoliday{
		Date:     "2025-11-01",
		Name:     "API Sourced",
		IsActive: true,
		Source:   "api",
	}

	if err := holidayRepo.CreateHoliday(adminHoliday); err != nil {
		t.Fatalf("Failed to create admin holiday: %v", err)
	}
	if err := holidayRepo.CreateHoliday(apiHoliday); err != nil {
		t.Fatalf("Failed to create API holiday: %v", err)
	}

	testCases := []struct {
		name       string
		holidayID  int
		wantStatus int
	}{
		{
			name:       "TC-3.2.4-A: Delete admin-created holiday",
			holidayID:  adminHoliday.ID,
			wantStatus: http.StatusOK,
		},
		{
			name:       "TC-3.2.4-B: Delete API-sourced holiday",
			holidayID:  apiHoliday.ID,
			wantStatus: http.StatusOK,
		},
		{
			name:       "TC-3.2.4-C: Delete non-existent holiday",
			holidayID:  99999,
			wantStatus: http.StatusOK, // Idempotent
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := "/api/holidays/" + strconv.Itoa(tc.holidayID)
			req := httptest.NewRequest(http.MethodDelete, path, nil)

			// Set admin context
			ctx := contextWithUser(req.Context(), 1, "admin@example.com", true)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			handler.DeleteHoliday(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("Status = %d, want %d. Body: %s", w.Code, tc.wantStatus, w.Body.String())
			}

			// Note: Holiday deletion verification would require GetHolidayByID method
			// For now, we trust the status code indicates successful deletion
		})
	}
}

// Test invalid ID format for update and delete
func TestInvalidHolidayID(t *testing.T) {
	_, handler, cleanup := setupHolidayHandlerTest(t)
	defer cleanup()

	testCases := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{
			name:       "TC-3.2.3-D: Invalid ID format for update",
			method:     http.MethodPut,
			path:       "/api/holidays/invalid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Invalid ID format for delete",
			method:     http.MethodDelete,
			path:       "/api/holidays/abc",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body := bytes.NewReader([]byte(`{"date":"2025-01-01","name":"Test","is_active":true,"source":"admin"}`))
			req := httptest.NewRequest(tc.method, tc.path, body)
			req.Header.Set("Content-Type", "application/json")

			// Set admin context
			ctx := contextWithUser(req.Context(), 1, "admin@example.com", true)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			if tc.method == http.MethodPut {
				handler.UpdateHoliday(w, req)
			} else {
				handler.DeleteHoliday(w, req)
			}

			if w.Code != tc.wantStatus {
				t.Errorf("Status = %d, want %d. Body: %s", w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}

// Helper function
func contains2025(date string) bool {
	return len(date) >= 4 && date[:4] == "2025"
}
