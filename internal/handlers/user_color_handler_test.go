package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gorilla/mux"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// getColorIDByName looks up a color ID by name for tenant 1
func getColorIDByName(t *testing.T, db *sql.DB, colorName string) int {
	var colorID int
	err := db.QueryRow(`SELECT id FROM color_categories WHERE tenant_id = 1 AND LOWER(name) = LOWER(?)`, colorName).Scan(&colorID)
	if err != nil {
		t.Fatalf("Failed to get color ID for %s: %v", colorName, err)
	}
	return colorID
}

// TestUserColorHandler_GetUserColors tests getting user's assigned colors
func TestUserColorHandler_GetUserColors(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{JWTSecret: "test-secret", JWTExpirationHours: 24}
	handler := NewUserColorHandler(db, cfg)

	// Create admin and regular user
	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin User", "blue")
	db.Exec("UPDATE users SET is_admin = 1 WHERE id = ?", adminID)

	regularUserID := testutil.SeedTestUser(t, db, "user@example.com", "Test User", "green")

	t.Run("returns user colors as admin", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/users/"+strconv.Itoa(regularUserID)+"/colors", nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(regularUserID)})
		ctx := contextWithTenant(req.Context(), 1, adminID, true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetUserColors(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var colors []map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &colors)

		// User was seeded with green level, so should have at least 1 color
		if len(colors) < 1 {
			t.Errorf("Expected at least 1 color, got %d", len(colors))
		}
	})

	t.Run("returns 403 for non-admin", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/users/"+strconv.Itoa(regularUserID)+"/colors", nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(regularUserID)})
		ctx := contextWithTenant(req.Context(), 1, regularUserID, false) // Not admin
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetUserColors(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", rec.Code)
		}
	})

	t.Run("returns 404 for non-existent user", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/users/99999/colors", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "99999"})
		ctx := contextWithTenant(req.Context(), 1, adminID, true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetUserColors(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})

	t.Run("returns 400 for invalid user ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/users/invalid/colors", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
		ctx := contextWithTenant(req.Context(), 1, adminID, true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetUserColors(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})
}

// TestUserColorHandler_AddColorToUser tests adding a color to a user
func TestUserColorHandler_AddColorToUser(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{JWTSecret: "test-secret", JWTExpirationHours: 24}
	handler := NewUserColorHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin User", "blue")
	db.Exec("UPDATE users SET is_admin = 1 WHERE id = ?", adminID)

	// Create user without colors
	userID := testutil.SeedTestUserWithoutColors(t, db, "nocolor@example.com", "No Color User", "green")

	t.Run("adds color to user", func(t *testing.T) {
		colorID := getColorIDByName(t, db, "gruen")
		reqBody := map[string]int{"color_id": colorID}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/admin/users/"+strconv.Itoa(userID)+"/colors", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(userID)})
		ctx := contextWithTenant(req.Context(), 1, adminID, true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.AddColorToUser(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify color was added
		var count int
		db.QueryRow("SELECT COUNT(*) FROM user_colors WHERE user_id = ? AND color_id = ?", userID, colorID).Scan(&count)
		if count != 1 {
			t.Errorf("Expected color to be added, got count %d", count)
		}
	})

	t.Run("returns 409 if user already has color", func(t *testing.T) {
		colorID := getColorIDByName(t, db, "gruen")
		reqBody := map[string]int{"color_id": colorID} // Same color again
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/admin/users/"+strconv.Itoa(userID)+"/colors", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(userID)})
		ctx := contextWithTenant(req.Context(), 1, adminID, true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.AddColorToUser(rec, req)

		if rec.Code != http.StatusConflict {
			t.Errorf("Expected status 409, got %d", rec.Code)
		}
	})

	t.Run("returns 403 for non-admin", func(t *testing.T) {
		colorID := getColorIDByName(t, db, "gelb")
		reqBody := map[string]int{"color_id": colorID}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/admin/users/"+strconv.Itoa(userID)+"/colors", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(userID)})
		ctx := contextWithTenant(req.Context(), 1, userID, false) // Not admin
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.AddColorToUser(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", rec.Code)
		}
	})

	t.Run("returns 400 for invalid color ID", func(t *testing.T) {
		reqBody := map[string]int{"color_id": 0}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/admin/users/"+strconv.Itoa(userID)+"/colors", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(userID)})
		ctx := contextWithTenant(req.Context(), 1, adminID, true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.AddColorToUser(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	t.Run("returns 400 for non-existent color", func(t *testing.T) {
		reqBody := map[string]int{"color_id": 99999}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/admin/users/"+strconv.Itoa(userID)+"/colors", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(userID)})
		ctx := contextWithTenant(req.Context(), 1, adminID, true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.AddColorToUser(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})
}

// TestUserColorHandler_RemoveColorFromUser tests removing a color from a user
func TestUserColorHandler_RemoveColorFromUser(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{JWTSecret: "test-secret", JWTExpirationHours: 24}
	handler := NewUserColorHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin User", "blue")
	db.Exec("UPDATE users SET is_admin = 1 WHERE id = ?", adminID)

	// Create user with colors
	userID := testutil.SeedTestUser(t, db, "user@example.com", "Test User", "green")

	t.Run("removes color from user", func(t *testing.T) {
		colorID := getColorIDByName(t, db, "gruen")
		// First verify user has the color
		var count int
		db.QueryRow("SELECT COUNT(*) FROM user_colors WHERE user_id = ? AND color_id = ?", userID, colorID).Scan(&count)
		if count == 0 {
			t.Skip("User doesn't have gruen color, skipping test")
		}

		req := httptest.NewRequest("DELETE", "/api/admin/users/"+strconv.Itoa(userID)+"/colors/"+strconv.Itoa(colorID), nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(userID), "colorId": strconv.Itoa(colorID)})
		ctx := contextWithTenant(req.Context(), 1, adminID, true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.RemoveColorFromUser(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify color was removed
		db.QueryRow("SELECT COUNT(*) FROM user_colors WHERE user_id = ? AND color_id = ?", userID, colorID).Scan(&count)
		if count != 0 {
			t.Errorf("Expected color to be removed, got count %d", count)
		}
	})

	t.Run("returns 404 if user doesn't have color", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/admin/users/"+strconv.Itoa(userID)+"/colors/99", nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(userID), "colorId": "99"})
		ctx := contextWithTenant(req.Context(), 1, adminID, true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.RemoveColorFromUser(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})

	t.Run("returns 403 for non-admin", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/admin/users/"+strconv.Itoa(userID)+"/colors/1", nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(userID), "colorId": "1"})
		ctx := contextWithTenant(req.Context(), 1, userID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.RemoveColorFromUser(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", rec.Code)
		}
	})
}

// TestUserColorHandler_SetUserColors tests setting all colors for a user
func TestUserColorHandler_SetUserColors(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{JWTSecret: "test-secret", JWTExpirationHours: 24}
	handler := NewUserColorHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin User", "blue")
	db.Exec("UPDATE users SET is_admin = 1 WHERE id = ?", adminID)

	userID := testutil.SeedTestUser(t, db, "user@example.com", "Test User", "green")

	t.Run("replaces all user colors", func(t *testing.T) {
		gruen := getColorIDByName(t, db, "gruen")
		gelb := getColorIDByName(t, db, "gelb")
		orange := getColorIDByName(t, db, "orange")
		reqBody := map[string][]int{"color_ids": {gruen, gelb, orange}}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/admin/users/"+strconv.Itoa(userID)+"/colors", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(userID)})
		ctx := contextWithTenant(req.Context(), 1, adminID, true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.SetUserColors(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify colors were set
		var count int
		db.QueryRow("SELECT COUNT(*) FROM user_colors WHERE user_id = ?", userID).Scan(&count)
		if count != 3 {
			t.Errorf("Expected 3 colors, got %d", count)
		}
	})

	t.Run("can set empty colors", func(t *testing.T) {
		reqBody := map[string][]int{"color_ids": {}}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/admin/users/"+strconv.Itoa(userID)+"/colors", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(userID)})
		ctx := contextWithTenant(req.Context(), 1, adminID, true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.SetUserColors(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify all colors were removed
		var count int
		db.QueryRow("SELECT COUNT(*) FROM user_colors WHERE user_id = ?", userID).Scan(&count)
		if count != 0 {
			t.Errorf("Expected 0 colors after clearing, got %d", count)
		}
	})

	t.Run("returns 403 for non-admin", func(t *testing.T) {
		colorID := getColorIDByName(t, db, "gruen")
		reqBody := map[string][]int{"color_ids": {colorID}}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/admin/users/"+strconv.Itoa(userID)+"/colors", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(userID)})
		ctx := contextWithTenant(req.Context(), 1, userID, false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.SetUserColors(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", rec.Code)
		}
	})

	t.Run("returns 400 for non-existent color ID", func(t *testing.T) {
		colorID := getColorIDByName(t, db, "gruen")
		reqBody := map[string][]int{"color_ids": {colorID, 99999}}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/admin/users/"+strconv.Itoa(userID)+"/colors", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(userID)})
		ctx := contextWithTenant(req.Context(), 1, adminID, true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.SetUserColors(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	t.Run("returns 404 for non-existent user", func(t *testing.T) {
		reqBody := map[string][]int{"color_ids": {1}}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/admin/users/99999/colors", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": "99999"})
		ctx := contextWithTenant(req.Context(), 1, adminID, true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.SetUserColors(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})
}
