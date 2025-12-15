package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gorilla/mux"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// TestColorRequestHandler_CreateRequest tests creating color requests
func TestColorRequestHandler_CreateRequest(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewColorRequestHandler(db, cfg)

	userID := testutil.SeedTestUser(t, db, "user@example.com", "Test User", "green")
	colorID := testutil.SeedTestColorCategory(t, db, "request-color", "#123456", 100)

	t.Run("user can request color they don't have", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"color_id": colorID,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/color-requests", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUserForColor(req.Context(), userID, "user@example.com", false)
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

	t.Run("user cannot request color they already have", func(t *testing.T) {
		user2ID := testutil.SeedTestUser(t, db, "user2@example.com", "User 2", "green")
		color2ID := testutil.SeedTestColorCategory(t, db, "has-color", "#aabbcc", 110)

		// Give user the color
		testutil.SeedTestUserColor(t, db, user2ID, color2ID)

		reqBody := map[string]interface{}{
			"color_id": color2ID,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/color-requests", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUserForColor(req.Context(), user2ID, "user2@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateRequest(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for already owned color, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("user with pending request cannot create another", func(t *testing.T) {
		user3ID := testutil.SeedTestUser(t, db, "user3@example.com", "User 3", "green")
		color3ID := testutil.SeedTestColorCategory(t, db, "pending-color", "#ddeeff", 120)
		color4ID := testutil.SeedTestColorCategory(t, db, "another-color", "#112233", 130)

		// Create pending request
		testutil.SeedTestColorRequest(t, db, user3ID, color3ID, "pending")

		reqBody := map[string]interface{}{
			"color_id": color4ID,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/color-requests", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUserForColor(req.Context(), user3ID, "user3@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateRequest(rec, req)

		if rec.Code != http.StatusConflict {
			t.Errorf("Expected status 409 for pending request, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("invalid color ID fails", func(t *testing.T) {
		user4ID := testutil.SeedTestUser(t, db, "user4@example.com", "User 4", "green")

		reqBody := map[string]interface{}{
			"color_id": 0,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/color-requests", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUserForColor(req.Context(), user4ID, "user4@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateRequest(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid color ID, got %d", rec.Code)
		}
	})
}

// TestColorRequestHandler_ListRequests tests listing color requests
func TestColorRequestHandler_ListRequests(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewColorRequestHandler(db, cfg)

	userID := testutil.SeedTestUser(t, db, "user@example.com", "Test User", "green")
	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "blue")
	colorID := testutil.SeedTestColorCategory(t, db, "list-color", "#123456", 200)

	// Create some requests
	testutil.SeedTestColorRequest(t, db, userID, colorID, "pending")

	t.Run("user sees only their own requests", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/color-requests", nil)
		ctx := contextWithUserForColor(req.Context(), userID, "user@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListRequests(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response []map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if len(response) == 0 {
			t.Error("Expected at least one request")
		}

		// All requests should belong to this user
		for _, req := range response {
			if int(req["user_id"].(float64)) != userID {
				t.Error("User should only see their own requests")
			}
		}
	})

	t.Run("admin sees all pending requests", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/color-requests", nil)
		ctx := contextWithUserForColor(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListRequests(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})
}

// TestColorRequestHandler_ApproveRequest tests approving color requests
func TestColorRequestHandler_ApproveRequest(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewColorRequestHandler(db, cfg)

	userID := testutil.SeedTestUser(t, db, "user@example.com", "Test User", "green")
	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "blue")
	colorID := testutil.SeedTestColorCategory(t, db, "approve-color", "#123456", 300)
	requestID := testutil.SeedTestColorRequest(t, db, userID, colorID, "pending")

	t.Run("admin can approve request", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"message": "Willkommen!",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/color-requests/"+strconv.Itoa(requestID)+"/approve", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(requestID)})
		ctx := contextWithUserForColor(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ApproveRequest(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})
}

// TestColorRequestHandler_DenyRequest tests denying color requests
func TestColorRequestHandler_DenyRequest(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewColorRequestHandler(db, cfg)

	userID := testutil.SeedTestUser(t, db, "user@example.com", "Test User", "green")
	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "blue")
	colorID := testutil.SeedTestColorCategory(t, db, "deny-color", "#123456", 400)
	requestID := testutil.SeedTestColorRequest(t, db, userID, colorID, "pending")

	t.Run("admin can deny request", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"message": "Bitte erst Einweisung absolvieren",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/color-requests/"+strconv.Itoa(requestID)+"/deny", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(requestID)})
		ctx := contextWithUserForColor(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DenyRequest(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})
}

// contextWithUserForColor creates context with user data
func contextWithUserForColor(ctx context.Context, userID int, email string, isAdmin bool) context.Context {
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, middleware.EmailKey, email)
	ctx = context.WithValue(ctx, middleware.IsAdminKey, isAdmin)
	ctx = context.WithValue(ctx, middleware.IsSuperAdminKey, false)
	return ctx
}
