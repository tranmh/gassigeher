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
	ctx = context.WithValue(ctx, middleware.TenantIDKey, 0) // Use test tenant
	return ctx
}

// colorCtxAdmin creates context with regular admin user for color tests
func colorCtxAdmin(ctx context.Context, userID int, email string) context.Context {
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, middleware.EmailKey, email)
	ctx = context.WithValue(ctx, middleware.IsAdminKey, true)
	ctx = context.WithValue(ctx, middleware.IsSuperAdminKey, false)
	ctx = context.WithValue(ctx, middleware.TenantIDKey, 0) // Use test tenant
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
		ctx := context.WithValue(req.Context(), middleware.TenantIDKey, 0) // Use test tenant
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.ListColors(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response struct {
			Colors []map[string]interface{} `json:"colors"`
		}
		json.Unmarshal(rec.Body.Bytes(), &response)

		// Should have at least 5 default colors from migration
		if len(response.Colors) < 5 {
			t.Errorf("Expected at least 5 colors, got %d", len(response.Colors))
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

		// Create a dog with this color (tenant_id=0 to match color's tenant)
		now := testutil.Now()
		_, err := db.Exec(`INSERT INTO dogs (tenant_id, name, breed, size, age, color_id, is_available, created_at)
			VALUES (0, ?, ?, ?, ?, ?, 1, ?)`, "TestDog", "Mix", "medium", 3, colorID, now)
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

// TestColorCategoryHandler_GetColorDogs tests the dogs-by-color endpoint
func TestColorCategoryHandler_GetColorDogs(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewColorCategoryHandler(db, cfg)

	superAdminID := testutil.SeedTestUser(t, db, "super@example.com", "Super Admin", "blue")

	t.Run("returns dogs for color", func(t *testing.T) {
		colorID := testutil.SeedTestColorCategory(t, db, "dogs-endpoint", "#aa0000", 400)

		// Create dogs with this color
		now := testutil.Now()
		_, _ = db.Exec(`INSERT INTO dogs (tenant_id, name, breed, size, age, color_id, is_available, created_at)
			VALUES (0, ?, ?, ?, ?, ?, 1, ?)`, "Rex", "Labrador", "large", 4, colorID, now)
		_, _ = db.Exec(`INSERT INTO dogs (tenant_id, name, breed, size, age, color_id, is_available, created_at)
			VALUES (0, ?, ?, ?, ?, ?, 0, ?)`, "Fifi", "Pudel", "small", 2, colorID, now)

		req := httptest.NewRequest("GET", "/api/colors/"+intToStr(colorID)+"/dogs", nil)
		req = mux.SetURLVars(req, map[string]string{"id": intToStr(colorID)})
		ctx := colorCtxSuperAdmin(req.Context(), superAdminID, "super@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetColorDogs(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response struct {
			Dogs []map[string]interface{} `json:"dogs"`
		}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if len(response.Dogs) != 2 {
			t.Fatalf("Expected 2 dogs, got %d", len(response.Dogs))
		}

		// Verify dog data is present
		firstDog := response.Dogs[0]
		if firstDog["name"] == nil || firstDog["breed"] == nil {
			t.Error("Expected dog name and breed in response")
		}
	})

	t.Run("returns empty array for color with no dogs", func(t *testing.T) {
		colorID := testutil.SeedTestColorCategory(t, db, "no-dogs-ep", "#bb0000", 401)

		req := httptest.NewRequest("GET", "/api/colors/"+intToStr(colorID)+"/dogs", nil)
		req = mux.SetURLVars(req, map[string]string{"id": intToStr(colorID)})
		ctx := colorCtxSuperAdmin(req.Context(), superAdminID, "super@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetColorDogs(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response struct {
			Dogs []map[string]interface{} `json:"dogs"`
		}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if len(response.Dogs) != 0 {
			t.Errorf("Expected 0 dogs, got %d", len(response.Dogs))
		}
	})

	t.Run("invalid color ID returns 400", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/colors/abc/dogs", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "abc"})
		ctx := colorCtxSuperAdmin(req.Context(), superAdminID, "super@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetColorDogs(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})
}

// TestColorCategoryHandler_GetColorUsers tests the users-by-color endpoint
func TestColorCategoryHandler_GetColorUsers(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewColorCategoryHandler(db, cfg)

	superAdminID := testutil.SeedTestUser(t, db, "super@example.com", "Super Admin", "blue")

	t.Run("returns users for color", func(t *testing.T) {
		colorID := testutil.SeedTestColorCategory(t, db, "users-endpoint", "#cc0000", 500)

		userID1 := testutil.SeedTestUserWithoutColors(t, db, "user1@example.com", "Max Mustermann", "green")
		userID2 := testutil.SeedTestUserWithoutColors(t, db, "user2@example.com", "Anna Schmidt", "green")
		testutil.SeedTestUserColor(t, db, userID1, colorID)
		testutil.SeedTestUserColor(t, db, userID2, colorID)

		req := httptest.NewRequest("GET", "/api/colors/"+intToStr(colorID)+"/users", nil)
		req = mux.SetURLVars(req, map[string]string{"id": intToStr(colorID)})
		ctx := colorCtxSuperAdmin(req.Context(), superAdminID, "super@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetColorUsers(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response struct {
			Users []map[string]interface{} `json:"users"`
		}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if len(response.Users) != 2 {
			t.Fatalf("Expected 2 users, got %d", len(response.Users))
		}

		// Verify user data is present
		firstUser := response.Users[0]
		if firstUser["first_name"] == nil || firstUser["last_name"] == nil {
			t.Error("Expected user first_name and last_name in response")
		}
	})

	t.Run("returns empty array for color with no users", func(t *testing.T) {
		colorID := testutil.SeedTestColorCategory(t, db, "no-users-ep", "#dd0000", 501)

		req := httptest.NewRequest("GET", "/api/colors/"+intToStr(colorID)+"/users", nil)
		req = mux.SetURLVars(req, map[string]string{"id": intToStr(colorID)})
		ctx := colorCtxSuperAdmin(req.Context(), superAdminID, "super@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetColorUsers(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response struct {
			Users []map[string]interface{} `json:"users"`
		}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if len(response.Users) != 0 {
			t.Errorf("Expected 0 users, got %d", len(response.Users))
		}
	})

	t.Run("excludes deleted and inactive users", func(t *testing.T) {
		colorID := testutil.SeedTestColorCategory(t, db, "excl-users-ep", "#ee0000", 502)

		deletedID := testutil.SeedTestUserWithoutColors(t, db, "del-ep@example.com", "Del User", "green")
		inactiveID := testutil.SeedTestUserWithoutColors(t, db, "inact-ep@example.com", "Inact User", "green")
		activeID := testutil.SeedTestUserWithoutColors(t, db, "active-ep@example.com", "Active User", "green")
		testutil.SeedTestUserColor(t, db, deletedID, colorID)
		testutil.SeedTestUserColor(t, db, inactiveID, colorID)
		testutil.SeedTestUserColor(t, db, activeID, colorID)

		_, _ = db.Exec(`UPDATE users SET is_deleted = 1 WHERE id = ?`, deletedID)
		_, _ = db.Exec(`UPDATE users SET is_active = 0 WHERE id = ?`, inactiveID)

		req := httptest.NewRequest("GET", "/api/colors/"+intToStr(colorID)+"/users", nil)
		req = mux.SetURLVars(req, map[string]string{"id": intToStr(colorID)})
		ctx := colorCtxSuperAdmin(req.Context(), superAdminID, "super@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetColorUsers(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response struct {
			Users []map[string]interface{} `json:"users"`
		}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if len(response.Users) != 1 {
			t.Errorf("Expected 1 user (only active), got %d", len(response.Users))
		}
	})
}

// intToStr converts int to string
func intToStr(i int) string {
	return strconv.Itoa(i)
}
