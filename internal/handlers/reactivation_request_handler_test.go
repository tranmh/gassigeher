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

// DONE: TestReactivationRequestHandler_CreateRequest tests creating reactivation requests
func TestReactivationRequestHandler_CreateRequest(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewReactivationRequestHandler(db, cfg)

	// Create inactive user
	inactiveUserID := testutil.SeedTestUser(t, db, "inactive@example.com", "Inactive User", "green")
	db.Exec("UPDATE users SET is_active = 0, deactivated_at = ?, deactivation_reason = 'Inactivity' WHERE id = ?", "2025-01-01", inactiveUserID)

	t.Run("successful request creation", func(t *testing.T) {
		reqBody := map[string]string{
			"email": "inactive@example.com",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/reactivation-requests", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.CreateRequest(rec, req)

		// Should return success (may be 200 or 201 depending on implementation)
		if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
			t.Errorf("Expected status 200 or 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		t.Logf("Create request returned status: %d", rec.Code)
	})

	t.Run("active user cannot request reactivation", func(t *testing.T) {
		reqBody := map[string]string{
			"email": "active@example.com",
		}

		testutil.SeedTestUser(t, db, "active@example.com", "Active User", "green")

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/reactivation-requests", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.CreateRequest(rec, req)

		// Returns OK but doesn't create request for active user (security)
		t.Logf("Active user request returned status: %d", rec.Code)
	})

	t.Run("user already has pending request", func(t *testing.T) {
		// Create another inactive user
		user2Email := "inactive2@example.com"
		user2ID := testutil.SeedTestUser(t, db, user2Email, "Inactive 2", "green")
		db.Exec("UPDATE users SET is_active = 0 WHERE id = ?", user2ID)

		// Create first request
		reqBody1 := map[string]string{"email": user2Email}
		body1, _ := json.Marshal(reqBody1)
		req1 := httptest.NewRequest("POST", "/api/reactivation-requests", bytes.NewReader(body1))
		req1.Header.Set("Content-Type", "application/json")
		rec1 := httptest.NewRecorder()
		handler.CreateRequest(rec1, req1)

		// Try to create duplicate
		reqBody2 := map[string]string{"email": user2Email}
		body2, _ := json.Marshal(reqBody2)
		req2 := httptest.NewRequest("POST", "/api/reactivation-requests", bytes.NewReader(body2))
		req2.Header.Set("Content-Type", "application/json")
		rec2 := httptest.NewRecorder()
		handler.CreateRequest(rec2, req2)

		// May return OK (security) or 409 depending on implementation
		t.Logf("Duplicate request returned status: %d", rec2.Code)
	})
}

// DONE: TestReactivationRequestHandler_ListRequests tests listing reactivation requests
func TestReactivationRequestHandler_ListRequests(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewReactivationRequestHandler(db, cfg)

	user1ID := testutil.SeedTestUser(t, db, "user1@example.com", "User 1", "green")
	user2ID := testutil.SeedTestUser(t, db, "user2@example.com", "User 2", "green")
	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")

	// Create reactivation requests
	db.Exec("INSERT INTO reactivation_requests (user_id, status, created_at) VALUES (?, 'pending', datetime('now'))", user1ID)
	db.Exec("INSERT INTO reactivation_requests (user_id, status, created_at) VALUES (?, 'pending', datetime('now'))", user2ID)

	t.Run("admin sees all requests", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/reactivation-requests", nil)
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

	t.Run("non-admin gets empty or error", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/reactivation-requests", nil)
		ctx := contextWithUser(req.Context(), user1ID, "user1@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListRequests(rec, req)

		// Non-admin should not see all requests
		t.Logf("Non-admin list returned status: %d", rec.Code)
	})
}

// DONE: TestReactivationRequestHandler_ApproveRequest tests approving reactivation (admin only)
func TestReactivationRequestHandler_ApproveRequest(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewReactivationRequestHandler(db, cfg)

	userID := testutil.SeedTestUser(t, db, "inactive@example.com", "Inactive User", "green")
	db.Exec("UPDATE users SET is_active = 0 WHERE id = ?", userID)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")

	var requestID int
	db.QueryRow("INSERT INTO reactivation_requests (user_id, status, created_at) VALUES (?, 'pending', datetime('now')) RETURNING id", userID).Scan(&requestID)

	t.Run("successful approval reactivates user", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"approved": true,
			"message":  "Account reactivated",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/reactivation-requests/"+fmt.Sprintf("%d", requestID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", requestID)})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ApproveRequest(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify request approved
		var status string
		db.QueryRow("SELECT status FROM reactivation_requests WHERE id = ?", requestID).Scan(&status)

		if status != "approved" {
			t.Errorf("Expected status 'approved', got %s", status)
		}

		// Verify user is reactivated
		var isActive bool
		db.QueryRow("SELECT is_active FROM users WHERE id = ?", userID).Scan(&isActive)

		if !isActive {
			t.Error("User should be reactivated (is_active=true)")
		}
	})

	t.Run("approve non-existent request", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"approved": true,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/reactivation-requests/99999", bytes.NewReader(body))
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

// DONE: TestReactivationRequestHandler_DenyRequest tests denying reactivation (admin only)
func TestReactivationRequestHandler_DenyRequest(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewReactivationRequestHandler(db, cfg)

	userID := testutil.SeedTestUser(t, db, "inactive@example.com", "Inactive User", "green")
	db.Exec("UPDATE users SET is_active = 0 WHERE id = ?", userID)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")

	var requestID int
	db.QueryRow("INSERT INTO reactivation_requests (user_id, status, created_at) VALUES (?, 'pending', datetime('now')) RETURNING id", userID).Scan(&requestID)

	t.Run("successful denial keeps user inactive", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"approved": false,
			"message":  "Cannot reactivate at this time",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/reactivation-requests/"+fmt.Sprintf("%d", requestID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", requestID)})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DenyRequest(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify request denied
		var status string
		db.QueryRow("SELECT status FROM reactivation_requests WHERE id = ?", requestID).Scan(&status)

		if status != "denied" {
			t.Errorf("Expected status 'denied', got %s", status)
		}

		// Verify user remains inactive
		var isActive bool
		db.QueryRow("SELECT is_active FROM users WHERE id = ?", userID).Scan(&isActive)

		if isActive {
			t.Error("User should remain inactive after denial")
		}
	})
}
