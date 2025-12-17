package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// DONE: TestExperienceRequestHandler_CreateRequest tests creating experience level requests
func TestExperienceRequestHandler_CreateRequest(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewExperienceRequestHandler(db, cfg)

	greenUserID := testutil.SeedTestUser(t, db, "green@example.com", "Green User", "green")
	blueUserID := testutil.SeedTestUser(t, db, "blue@example.com", "Blue User", "blue")

	// New level order: green < orange < blue (blue is highest)
	t.Run("green user requests orange level", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"requested_level": "orange",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/experience-requests", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), greenUserID, "green@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateRequest(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["id"] == nil {
			t.Error("Expected request ID in response")
		}
	})

	t.Run("blue user already has highest level", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"requested_level": "orange",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/experience-requests", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), blueUserID, "blue@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateRequest(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for highest level, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("invalid requested level", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"requested_level": "invalid",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/experience-requests", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), greenUserID, "green@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateRequest(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid level, got %d", rec.Code)
		}
	})

	t.Run("duplicate pending request", func(t *testing.T) {
		// Create a new green user for this test
		dupGreenID := testutil.SeedTestUser(t, db, "dupgreen@example.com", "Dup Green", "green")

		// Create first request for orange (green can request orange in new order)
		testutil.SeedTestExperienceRequest(t, db, dupGreenID, "orange", "pending")

		// Try to create duplicate
		reqBody := map[string]interface{}{
			"requested_level": "orange",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/experience-requests", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), dupGreenID, "dupgreen@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateRequest(rec, req)

		if rec.Code != http.StatusConflict {
			t.Errorf("Expected status 409 for duplicate pending request, got %d", rec.Code)
		}
	})

	t.Run("green user cannot request blue directly", func(t *testing.T) {
		// Create new green user (new order: green < orange < blue, so green cannot skip to blue)
		newGreenID := testutil.SeedTestUser(t, db, "newgreen@example.com", "New Green", "green")

		reqBody := map[string]interface{}{
			"requested_level": "blue",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/experience-requests", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), newGreenID, "newgreen@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateRequest(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for skipping level, got %d", rec.Code)
		}
	})
}

// DONE: TestExperienceRequestHandler_ListRequests tests listing experience requests
func TestExperienceRequestHandler_ListRequests(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewExperienceRequestHandler(db, cfg)

	user1ID := testutil.SeedTestUser(t, db, "user1@example.com", "User 1", "green")
	user2ID := testutil.SeedTestUser(t, db, "user2@example.com", "User 2", "green")
	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")

	// Create requests
	testutil.SeedTestExperienceRequest(t, db, user1ID, "blue", "pending")
	testutil.SeedTestExperienceRequest(t, db, user2ID, "blue", "pending")

	t.Run("admin sees all requests", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/experience-requests", nil)
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListRequests(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var requests []map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &requests)

		if len(requests) < 2 {
			t.Errorf("Expected at least 2 requests, got %d", len(requests))
		}
	})

	t.Run("user sees only own requests", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/experience-requests", nil)
		ctx := contextWithUser(req.Context(), user1ID, "user1@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListRequests(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var requests []map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &requests)

		if len(requests) != 1 {
			t.Errorf("Expected 1 request for user1, got %d", len(requests))
		}
	})
}

// DONE: TestExperienceRequestHandler_ApproveRequest tests approving requests (admin only)
func TestExperienceRequestHandler_ApproveRequest(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewExperienceRequestHandler(db, cfg)

	userID := testutil.SeedTestUser(t, db, "user@example.com", "User", "green")
	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")

	requestID := testutil.SeedTestExperienceRequest(t, db, userID, "blue", "pending")

	t.Run("successful approval by admin", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"approved": true,
			"message":  "Great progress!",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/experience-requests/"+fmt.Sprintf("%d", requestID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", requestID)})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ApproveRequest(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify request is approved (color assignments tested separately in TestExperienceRequestHandler_ApproveRequest_AssignsColors)
		var requestStatus string
		db.QueryRow("SELECT status FROM experience_requests WHERE id = ?", requestID).Scan(&requestStatus)

		if requestStatus != "approved" {
			t.Errorf("Expected status 'approved', got %s", requestStatus)
		}
	})

	t.Run("approve non-existent request", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"approved": true,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/experience-requests/99999", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": "99999"})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ApproveRequest(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})
}

// TestExperienceRequestHandler_ApproveRequest_AssignsColors tests that approving
// an experience request also assigns the appropriate colors to the user
func TestExperienceRequestHandler_ApproveRequest_AssignsColors(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewExperienceRequestHandler(db, cfg)

	// Create user without colors for precise control
	userID := testutil.SeedTestUserWithoutColors(t, db, "colortest@example.com", "Color Test", "green")
	adminID := testutil.SeedTestUser(t, db, "admin-color@example.com", "Admin", "blue")

	// Color categories are already seeded by migration 024 with IDs 1-7:
	// 1=gruen, 2=gelb, 3=orange, 4=hellblau, 5=dunkelblau, 6=helllila, 7=dunkellila

	t.Run("green_to_orange_assigns_colors_1_2_3", func(t *testing.T) {
		// Create request from green to orange
		requestID := testutil.SeedTestExperienceRequest(t, db, userID, "orange", "pending")

		reqBody := map[string]interface{}{
			"approved": true,
			"message":  "Welcome to orange!",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/experience-requests/"+fmt.Sprintf("%d", requestID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", requestID)})
		ctx := contextWithUser(req.Context(), adminID, "admin-color@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ApproveRequest(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify colors were assigned (orange level gets colors 1, 2, 3)
		var colorCount int
		err := db.QueryRow("SELECT COUNT(*) FROM user_colors WHERE user_id = ?", userID).Scan(&colorCount)
		if err != nil {
			t.Fatalf("Failed to query user colors: %v", err)
		}

		if colorCount != 3 {
			t.Errorf("Expected 3 colors (gruen, gelb, orange) for orange level, got %d", colorCount)
		}

		// Verify specific colors are present
		var hasGruen, hasGelb, hasOrange int
		db.QueryRow("SELECT COUNT(*) FROM user_colors WHERE user_id = ? AND color_id = 1", userID).Scan(&hasGruen)
		db.QueryRow("SELECT COUNT(*) FROM user_colors WHERE user_id = ? AND color_id = 2", userID).Scan(&hasGelb)
		db.QueryRow("SELECT COUNT(*) FROM user_colors WHERE user_id = ? AND color_id = 3", userID).Scan(&hasOrange)

		if hasGruen != 1 || hasGelb != 1 || hasOrange != 1 {
			t.Errorf("Expected colors 1,2,3 for orange level. gruen=%d, gelb=%d, orange=%d", hasGruen, hasGelb, hasOrange)
		}
	})

	t.Run("orange_to_blue_assigns_colors_1_2_3_4_5", func(t *testing.T) {
		// Create a new user for this test
		blueUserID := testutil.SeedTestUserWithoutColors(t, db, "bluetest@example.com", "Blue Test", "orange")

		// Create request from orange to blue
		requestID := testutil.SeedTestExperienceRequest(t, db, blueUserID, "blue", "pending")

		reqBody := map[string]interface{}{
			"approved": true,
			"message":  "Welcome to blue!",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/experience-requests/"+fmt.Sprintf("%d", requestID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", requestID)})
		ctx := contextWithUser(req.Context(), adminID, "admin-color@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ApproveRequest(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify colors were assigned (blue level gets colors 1, 2, 3, 4, 5)
		var colorCount int
		err := db.QueryRow("SELECT COUNT(*) FROM user_colors WHERE user_id = ?", blueUserID).Scan(&colorCount)
		if err != nil {
			t.Fatalf("Failed to query user colors: %v", err)
		}

		if colorCount != 5 {
			t.Errorf("Expected 5 colors for blue level, got %d", colorCount)
		}
	})
}

// DONE: TestExperienceRequestHandler_DenyRequest tests denying requests (admin only)
func TestExperienceRequestHandler_DenyRequest(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewExperienceRequestHandler(db, cfg)

	userID := testutil.SeedTestUser(t, db, "user@example.com", "User", "green")
	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")

	requestID := testutil.SeedTestExperienceRequest(t, db, userID, "blue", "pending")

	t.Run("successful denial by admin", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"approved": false,
			"message":  "Need more experience",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/experience-requests/"+fmt.Sprintf("%d", requestID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", requestID)})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DenyRequest(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify request is denied
		var requestStatus string
		db.QueryRow("SELECT status FROM experience_requests WHERE id = ?", requestID).Scan(&requestStatus)

		if requestStatus != "denied" {
			t.Errorf("Expected status 'denied', got %s", requestStatus)
		}
	})
}
