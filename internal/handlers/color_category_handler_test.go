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

// colorCtxSuperAdmin creates context with super admin user for color tests
func colorCtxSuperAdmin(ctx context.Context, userID int, email string) context.Context {
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, middleware.EmailKey, email)
	ctx = context.WithValue(ctx, middleware.IsAdminKey, true)
	ctx = context.WithValue(ctx, middleware.IsSuperAdminKey, true)
	return ctx
}

// colorCtxAdmin creates context with regular admin user for color tests
func colorCtxAdmin(ctx context.Context, userID int, email string) context.Context {
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, middleware.EmailKey, email)
	ctx = context.WithValue(ctx, middleware.IsAdminKey, true)
	ctx = context.WithValue(ctx, middleware.IsSuperAdminKey, false)
	return ctx
}

// TestColorCategoryHandler_ListColors tests listing all color categories
func TestColorCategoryHandler_ListColors(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewColorCategoryHandler(db, cfg)

	t.Run("returns all colors - public endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/colors", nil)
		rec := httptest.NewRecorder()

		handler.ListColors(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response struct {
			Colors []map[string]interface{} `json:"colors"`
		}
		json.Unmarshal(rec.Body.Bytes(), &response)

		// Should have at least 7 default colors from migration
		if len(response.Colors) < 7 {
			t.Errorf("Expected at least 7 colors, got %d", len(response.Colors))
		}
	})
}

// TestColorCategoryHandler_CreateColor tests creating color categories
func TestColorCategoryHandler_CreateColor(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewColorCategoryHandler(db, cfg)

	superAdminID := testutil.SeedTestUser(t, db, "super@example.com", "Super Admin", "blue")
	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "blue")

	t.Run("super admin can create color", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":         "test-new-color",
			"hex_code":     "#aabbcc",
			"pattern_icon": "star",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/colors", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := colorCtxSuperAdmin(req.Context(), superAdminID, "super@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateColor(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["id"] == nil {
			t.Error("Expected color ID in response")
		}
	})

	t.Run("regular admin cannot create color", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":     "admin-color",
			"hex_code": "#ddeeff",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/colors", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := colorCtxAdmin(req.Context(), adminID, "admin@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateColor(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected status 403 for non-super-admin, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("invalid hex code fails validation", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":     "bad-color",
			"hex_code": "invalid",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/colors", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := colorCtxSuperAdmin(req.Context(), superAdminID, "super@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateColor(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid hex code, got %d", rec.Code)
		}
	})
}

// TestColorCategoryHandler_UpdateColor tests updating color categories
func TestColorCategoryHandler_UpdateColor(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewColorCategoryHandler(db, cfg)

	superAdminID := testutil.SeedTestUser(t, db, "super@example.com", "Super Admin", "blue")
	colorID := testutil.SeedTestColorCategory(t, db, "update-me", "#111111", 100)

	t.Run("super admin can update color", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":     "updated-name",
			"hex_code": "#222222",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/colors/"+intToStr(colorID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": intToStr(colorID)})
		ctx := colorCtxSuperAdmin(req.Context(), superAdminID, "super@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateColor(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})
}

// TestColorCategoryHandler_DeleteColor tests deleting color categories
func TestColorCategoryHandler_DeleteColor(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewColorCategoryHandler(db, cfg)

	superAdminID := testutil.SeedTestUser(t, db, "super@example.com", "Super Admin", "blue")

	t.Run("super admin can delete color without dogs", func(t *testing.T) {
		colorID := testutil.SeedTestColorCategory(t, db, "delete-me", "#333333", 200)

		req := httptest.NewRequest("DELETE", "/api/colors/"+intToStr(colorID), nil)
		req = mux.SetURLVars(req, map[string]string{"id": intToStr(colorID)})
		ctx := colorCtxSuperAdmin(req.Context(), superAdminID, "super@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DeleteColor(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("cannot delete color with dogs assigned", func(t *testing.T) {
		colorID := testutil.SeedTestColorCategory(t, db, "has-dogs", "#444444", 300)

		// Create a dog with this color
		_, err := db.Exec(`INSERT INTO dogs (name, breed, size, age, color_id, is_available, created_at)
			VALUES (?, ?, ?, ?, ?, 1, datetime('now'))`, "TestDog", "Mix", "medium", 3, colorID)
		if err != nil {
			t.Fatalf("Failed to create test dog: %v", err)
		}

		req := httptest.NewRequest("DELETE", "/api/colors/"+intToStr(colorID), nil)
		req = mux.SetURLVars(req, map[string]string{"id": intToStr(colorID)})
		ctx := colorCtxSuperAdmin(req.Context(), superAdminID, "super@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DeleteColor(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for color with dogs, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})
}

// intToStr converts int to string
func intToStr(i int) string {
	return strconv.Itoa(i)
}
