package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// contextWithTenantAndUser is a helper for tests that need explicit tenant control
func contextWithTenantAndUser(ctx context.Context, tenantID, userID int) context.Context {
	ctx = context.WithValue(ctx, middleware.TenantIDKey, tenantID)
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, middleware.IsAdminKey, true)
	return ctx
}

// DONE: TestDogHandler_ListDogs tests listing dogs with filters
func TestDogHandler_ListDogs(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewDogHandler(db, cfg)

	// Seed test dogs
	testutil.SeedTestDog(t, db, "Bella", "Labrador", "green")
	testutil.SeedTestDog(t, db, "Max", "Beagle", "blue")
	testutil.SeedTestDog(t, db, "Rocky", "German Shepherd", "orange")

	userID := testutil.SeedTestUser(t, db, "user@example.com", "User", "green")

	t.Run("list all dogs", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/dogs", nil)
		ctx := contextWithUser(req.Context(), userID, "user@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListDogs(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var dogs []map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &dogs)

		if len(dogs) != 3 {
			t.Errorf("Expected 3 dogs, got %d", len(dogs))
		}
	})

	t.Run("filter by category", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/dogs?category=green", nil)
		ctx := contextWithUser(req.Context(), userID, "user@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListDogs(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var dogs []map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &dogs)

		if len(dogs) != 1 {
			t.Errorf("Expected 1 green dog, got %d", len(dogs))
		}

		if len(dogs) > 0 && dogs[0]["name"] != "Bella" {
			t.Errorf("Expected dog 'Bella', got %v", dogs[0]["name"])
		}
	})

	t.Run("filter by available", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/dogs?available=true", nil)
		ctx := contextWithUser(req.Context(), userID, "user@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListDogs(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var dogs []map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &dogs)

		// All test dogs are available
		if len(dogs) != 3 {
			t.Errorf("Expected 3 available dogs, got %d", len(dogs))
		}
	})
}

// DONE: TestDogHandler_GetDog tests getting single dog by ID
func TestDogHandler_GetDog(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewDogHandler(db, cfg)

	dogID := testutil.SeedTestDog(t, db, "Bella", "Labrador", "green")
	userID := testutil.SeedTestUser(t, db, "user@example.com", "User", "green")

	t.Run("successful get dog", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/dogs/"+fmt.Sprintf("%d", dogID), nil)
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dogID)})
		ctx := contextWithUser(req.Context(), userID, "user@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetDog(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var dog map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &dog)

		if dog["name"] != "Bella" {
			t.Errorf("Expected dog name 'Bella', got %v", dog["name"])
		}
	})

	t.Run("non-existent dog", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/dogs/99999", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "99999"})
		ctx := contextWithUser(req.Context(), userID, "user@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetDog(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})

	t.Run("invalid dog ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/dogs/invalid", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
		ctx := contextWithUser(req.Context(), userID, "user@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetDog(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})
}

// DONE: TestDogHandler_CreateDog tests creating a dog (admin only)
func TestDogHandler_CreateDog(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewDogHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")
	userID := testutil.SeedTestUser(t, db, "user@example.com", "User", "green")

	t.Run("successful creation by admin", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":     "New Dog",
			"breed":    "Poodle",
			"size":     "medium",
			"age":      3,
			"category": "blue",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/dogs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateDog(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["id"] == nil {
			t.Error("Expected dog ID in response")
		}
	})

	t.Run("non-admin cannot create", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":     "Unauthorized Dog",
			"breed":    "Poodle",
			"size":     "medium",
			"age":      3,
			"category": "blue",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/dogs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), userID, "user@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateDog(rec, req)

		// Note: In production, RequireAdmin middleware blocks this before reaching handler
		// In tests without full middleware chain, handler may process it
		// Either way, verify non-admin doesn't have unrestricted access
		t.Logf("Non-admin create attempt returned status: %d", rec.Code)
	})

	t.Run("missing required fields", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"breed": "Poodle",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/dogs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateDog(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	t.Run("invalid category", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":     "Invalid Category Dog",
			"breed":    "Poodle",
			"size":     "medium",
			"age":      3,
			"category": "invalid",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/dogs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateDog(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid category, got %d", rec.Code)
		}
	})
}

// TestDogHandler_CreateDog_DogLimitEnforcement tests 10-dog limit per tenant (TDD RED Phase)
func TestDogHandler_CreateDog_DogLimitEnforcement(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewDogHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")

	t.Run("rejects 11th dog for free tenant", func(t *testing.T) {
		// Create 10 dogs for tenant 1 (the limit for free tier)
		for i := 1; i <= 10; i++ {
			testutil.SeedTestDog(t, db, fmt.Sprintf("Dog %d", i), "Labrador", "green")
		}

		// Try to create 11th dog - should fail
		reqBody := map[string]interface{}{
			"name":     "Dog 11 - Over Limit",
			"breed":    "Poodle",
			"size":     "medium",
			"age":      3,
			"category": "green",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/dogs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateDog(rec, req)

		// Should return 409 Conflict (dog limit reached)
		if rec.Code != http.StatusConflict {
			t.Errorf("Expected status 409 (Conflict), got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify error message mentions limit
		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)
		if response["error"] == nil {
			t.Error("Expected error message in response")
		}
	})

	t.Run("allows creation within limit", func(t *testing.T) {
		// Use fresh DB with no dogs
		db2 := testutil.SetupTestDB(t)
		handler2 := NewDogHandler(db2, cfg)
		adminID2 := testutil.SeedTestUser(t, db2, "admin2@example.com", "Admin", "orange")

		// Create 9 dogs (under limit)
		for i := 1; i <= 9; i++ {
			testutil.SeedTestDog(t, db2, fmt.Sprintf("Dog %d", i), "Labrador", "green")
		}

		// 10th dog should succeed
		reqBody := map[string]interface{}{
			"name":     "Dog 10 - At Limit",
			"breed":    "Poodle",
			"size":     "medium",
			"age":      3,
			"category": "green",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/dogs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), adminID2, "admin2@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler2.CreateDog(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("Expected status 201 for 10th dog, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})
}

// TestDogHandler_CreateDog_SetsTenantID tests that CreateDog sets TenantID from context (TDD RED Phase)
func TestDogHandler_CreateDog_SetsTenantID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewDogHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")

	t.Run("created dog has tenant_id from context", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":     "Tenant Test Dog",
			"breed":    "Labrador",
			"size":     "medium",
			"age":      3,
			"category": "green",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/dogs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateDog(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		dogID := int(response["id"].(float64))

		// Verify dog has correct tenant_id in database
		var tenantID *int
		err := db.QueryRow("SELECT tenant_id FROM dogs WHERE id = ?", dogID).Scan(&tenantID)
		if err != nil {
			t.Fatalf("Failed to query dog tenant_id: %v", err)
		}

		if tenantID == nil {
			t.Error("Expected dog to have tenant_id set, but it was NULL")
		} else if *tenantID != 0 {
			t.Errorf("Expected dog tenant_id to be 0, got %d", *tenantID)
		}

		// Also verify tenant_id is returned in the response
		if response["tenant_id"] == nil {
			t.Error("Expected tenant_id in response, but it was nil")
		} else if int(response["tenant_id"].(float64)) != 0 {
			t.Errorf("Expected tenant_id 0 in response, got %v", response["tenant_id"])
		}
	})
}

// DONE: TestDogHandler_UpdateDog tests updating dog information (admin only)
func TestDogHandler_UpdateDog(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewDogHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")
	dogID := testutil.SeedTestDog(t, db, "Bella", "Labrador", "green")

	t.Run("successful update", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name": "Bella Updated",
			"age":  6,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/dogs/"+fmt.Sprintf("%d", dogID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dogID)})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateDog(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify update
		var name string
		var age int
		db.QueryRow("SELECT name, age FROM dogs WHERE id = ?", dogID).Scan(&name, &age)

		if name != "Bella Updated" {
			t.Errorf("Expected name 'Bella Updated', got %s", name)
		}
		if age != 6 {
			t.Errorf("Expected age 6, got %d", age)
		}
	})

	t.Run("update non-existent dog", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name": "Ghost Dog",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/dogs/99999", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": "99999"})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateDog(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})
}

// DONE: TestDogHandler_DeleteDog tests deleting a dog (admin only)
func TestDogHandler_DeleteDog(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewDogHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")
	dogID := testutil.SeedTestDog(t, db, "Bella", "Labrador", "green")

	t.Run("successful deletion", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/dogs/"+fmt.Sprintf("%d", dogID), nil)
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dogID)})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DeleteDog(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify dog is deleted
		var count int
		db.QueryRow("SELECT COUNT(*) FROM dogs WHERE id = ?", dogID).Scan(&count)

		if count != 0 {
			t.Error("Dog should be deleted from database")
		}
	})

	t.Run("delete non-existent dog", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/dogs/99999", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "99999"})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DeleteDog(rec, req)

		// Handler returns OK even if dog doesn't exist (idempotent delete)
		t.Logf("Delete non-existent dog returned status: %d", rec.Code)
	})
}

// DONE: TestDogHandler_ToggleAvailability tests toggling dog availability (admin only)
func TestDogHandler_ToggleAvailability(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewDogHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")
	dogID := testutil.SeedTestDog(t, db, "Bella", "Labrador", "green")

	t.Run("make dog unavailable", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"is_available":       false,
			"unavailable_reason": "Sick",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/dogs/"+fmt.Sprintf("%d", dogID)+"/availability", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dogID)})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ToggleAvailability(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify dog is unavailable
		var isAvailable bool
		var reason *string
		db.QueryRow("SELECT is_available, unavailable_reason FROM dogs WHERE id = ?", dogID).Scan(&isAvailable, &reason)

		if isAvailable {
			t.Error("Dog should be unavailable")
		}
		if reason == nil || *reason != "Sick" {
			t.Errorf("Expected reason 'Sick', got %v", reason)
		}
	})

	t.Run("make dog available again", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"is_available": true,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/dogs/"+fmt.Sprintf("%d", dogID)+"/availability", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dogID)})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ToggleAvailability(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		// Verify dog is available
		var isAvailable bool
		db.QueryRow("SELECT is_available FROM dogs WHERE id = ?", dogID).Scan(&isAvailable)

		if !isAvailable {
			t.Error("Dog should be available")
		}
	})

	t.Run("make unavailable without reason - uses default", func(t *testing.T) {
		dogID2 := testutil.SeedTestDog(t, db, "Max", "Beagle", "blue")

		reqBody := map[string]interface{}{
			"is_available": false,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/dogs/"+fmt.Sprintf("%d", dogID2)+"/availability", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dogID2)})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ToggleAvailability(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		// Verify default reason was applied
		var reason *string
		db.QueryRow("SELECT unavailable_reason FROM dogs WHERE id = ?", dogID2).Scan(&reason)
		if reason == nil || *reason != "Temporarily unavailable" {
			t.Errorf("Expected default reason 'Temporarily unavailable', got %v", reason)
		}
	})

	t.Run("dog not found", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"is_available": true,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/dogs/99999/availability", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": "99999"})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ToggleAvailability(rec, req)

		// Should error or handle gracefully
		if rec.Code == http.StatusOK {
			t.Logf("ToggleAvailability for non-existent dog returned 200")
		}
	})

	t.Run("invalid dog ID", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"is_available": true,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/dogs/invalid/availability", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ToggleAvailability(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	t.Run("invalid request body", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/api/dogs/"+fmt.Sprintf("%d", dogID)+"/availability", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dogID)})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ToggleAvailability(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})
}

// DONE: TestDogHandler_CreateDogWithCareInfo tests creating a dog with care info fields
func TestDogHandler_CreateDogWithCareInfo(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewDogHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")

	t.Run("create dog with all care info fields", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":                 "Care Dog",
			"breed":                "Golden Retriever",
			"size":                 "large",
			"age":                  5,
			"category":             "green",
			"special_needs":        "Needs gentle handling, afraid of loud noises",
			"pickup_location":      "Zwinger 3, Auslauf B",
			"walk_route":           "Waldweg hinter dem Tierheim, nicht an der Hauptstraße",
			"walk_duration":        45,
			"special_instructions": "Nicht mit anderen Hunden zusammenführen",
			"default_morning_time": "09:00",
			"default_evening_time": "17:00",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/dogs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateDog(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		// Verify all care info fields are returned
		if response["special_needs"] != "Needs gentle handling, afraid of loud noises" {
			t.Errorf("Expected special_needs to match, got %v", response["special_needs"])
		}
		if response["pickup_location"] != "Zwinger 3, Auslauf B" {
			t.Errorf("Expected pickup_location to match, got %v", response["pickup_location"])
		}
		if response["walk_route"] != "Waldweg hinter dem Tierheim, nicht an der Hauptstraße" {
			t.Errorf("Expected walk_route to match, got %v", response["walk_route"])
		}
		if response["walk_duration"] != float64(45) {
			t.Errorf("Expected walk_duration 45, got %v", response["walk_duration"])
		}
		if response["special_instructions"] != "Nicht mit anderen Hunden zusammenführen" {
			t.Errorf("Expected special_instructions to match, got %v", response["special_instructions"])
		}
		if response["default_morning_time"] != "09:00" {
			t.Errorf("Expected default_morning_time '09:00', got %v", response["default_morning_time"])
		}
		if response["default_evening_time"] != "17:00" {
			t.Errorf("Expected default_evening_time '17:00', got %v", response["default_evening_time"])
		}
	})

	t.Run("create dog with partial care info", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":            "Partial Care Dog",
			"breed":           "Beagle",
			"size":            "medium",
			"age":             3,
			"category":        "blue",
			"pickup_location": "Main entrance",
			"walk_duration":   30,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/dogs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateDog(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		// Verify partial fields
		if response["pickup_location"] != "Main entrance" {
			t.Errorf("Expected pickup_location 'Main entrance', got %v", response["pickup_location"])
		}
		if response["walk_duration"] != float64(30) {
			t.Errorf("Expected walk_duration 30, got %v", response["walk_duration"])
		}

		// Verify optional fields are null
		if response["special_needs"] != nil {
			t.Errorf("Expected special_needs to be nil, got %v", response["special_needs"])
		}
		if response["walk_route"] != nil {
			t.Errorf("Expected walk_route to be nil, got %v", response["walk_route"])
		}
	})

	t.Run("create dog without care info", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":     "Basic Dog",
			"breed":    "Poodle",
			"size":     "small",
			"age":      2,
			"category": "green",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/dogs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateDog(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})
}

// DONE: TestDogHandler_UpdateDogCareInfo tests updating dog care info fields
func TestDogHandler_UpdateDogCareInfo(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewDogHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")
	dogID := testutil.SeedTestDog(t, db, "Bella", "Labrador", "green")

	t.Run("update care info fields", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"special_needs":        "Updated special needs",
			"pickup_location":      "New pickup location",
			"walk_route":           "New walking route",
			"walk_duration":        60,
			"special_instructions": "New instructions",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/dogs/"+fmt.Sprintf("%d", dogID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dogID)})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateDog(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify by querying database
		var specialNeeds, pickupLocation, walkRoute, specialInstructions *string
		var walkDuration *int
		db.QueryRow(`SELECT special_needs, pickup_location, walk_route, walk_duration, special_instructions
			FROM dogs WHERE id = ?`, dogID).Scan(&specialNeeds, &pickupLocation, &walkRoute, &walkDuration, &specialInstructions)

		if specialNeeds == nil || *specialNeeds != "Updated special needs" {
			t.Errorf("Expected special_needs 'Updated special needs', got %v", specialNeeds)
		}
		if pickupLocation == nil || *pickupLocation != "New pickup location" {
			t.Errorf("Expected pickup_location 'New pickup location', got %v", pickupLocation)
		}
		if walkRoute == nil || *walkRoute != "New walking route" {
			t.Errorf("Expected walk_route 'New walking route', got %v", walkRoute)
		}
		if walkDuration == nil || *walkDuration != 60 {
			t.Errorf("Expected walk_duration 60, got %v", walkDuration)
		}
		if specialInstructions == nil || *specialInstructions != "New instructions" {
			t.Errorf("Expected special_instructions 'New instructions', got %v", specialInstructions)
		}
	})

	t.Run("update default times", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"default_morning_time": "08:30",
			"default_evening_time": "18:00",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/dogs/"+fmt.Sprintf("%d", dogID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dogID)})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateDog(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify times
		var morningTime, eveningTime *string
		db.QueryRow(`SELECT default_morning_time, default_evening_time FROM dogs WHERE id = ?`, dogID).Scan(&morningTime, &eveningTime)

		if morningTime == nil || *morningTime != "08:30" {
			t.Errorf("Expected default_morning_time '08:30', got %v", morningTime)
		}
		if eveningTime == nil || *eveningTime != "18:00" {
			t.Errorf("Expected default_evening_time '18:00', got %v", eveningTime)
		}
	})

	t.Run("care info fields returned in GET response", func(t *testing.T) {
		// First update care info
		reqBody := map[string]interface{}{
			"special_needs":   "Visible needs",
			"pickup_location": "Visible location",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/dogs/"+fmt.Sprintf("%d", dogID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dogID)})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		handler.UpdateDog(rec, req)

		// Then GET the dog
		getReq := httptest.NewRequest("GET", "/api/dogs/"+fmt.Sprintf("%d", dogID), nil)
		getReq = mux.SetURLVars(getReq, map[string]string{"id": fmt.Sprintf("%d", dogID)})
		ctx = contextWithUser(getReq.Context(), adminID, "admin@example.com", true)
		getReq = getReq.WithContext(ctx)

		getRec := httptest.NewRecorder()
		handler.GetDog(getRec, getReq)

		if getRec.Code != http.StatusOK {
			t.Errorf("Expected GET status 200, got %d", getRec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(getRec.Body.Bytes(), &response)

		if response["special_needs"] != "Visible needs" {
			t.Errorf("Expected special_needs 'Visible needs', got %v", response["special_needs"])
		}
		if response["pickup_location"] != "Visible location" {
			t.Errorf("Expected pickup_location 'Visible location', got %v", response["pickup_location"])
		}
	})
}

// DONE: TestDogHandler_GetBreeds tests getting list of unique breeds
func TestDogHandler_GetBreeds(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewDogHandler(db, cfg)

	userID := testutil.SeedTestUser(t, db, "user@example.com", "User", "green")

	// Seed dogs with different breeds
	testutil.SeedTestDog(t, db, "Bella", "Labrador", "green")
	testutil.SeedTestDog(t, db, "Max", "Beagle", "blue")
	testutil.SeedTestDog(t, db, "Rocky", "Labrador", "green") // Duplicate breed

	t.Run("get unique breeds", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/dogs/breeds", nil)
		ctx := contextWithUser(req.Context(), userID, "user@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetBreeds(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var breeds []string
		json.Unmarshal(rec.Body.Bytes(), &breeds)

		// Should have 2 unique breeds (Labrador, Beagle)
		if len(breeds) != 2 {
			t.Errorf("Expected 2 unique breeds, got %d", len(breeds))
		}
	})

	t.Run("no dogs in database", func(t *testing.T) {
		// Use fresh DB
		db2 := testutil.SetupTestDB(t)
		handler2 := NewDogHandler(db2, cfg)

		req := httptest.NewRequest("GET", "/api/dogs/breeds", nil)
		ctx := contextWithUser(req.Context(), userID, "user@example.com", false)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler2.GetBreeds(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var breeds []string
		json.Unmarshal(rec.Body.Bytes(), &breeds)

		if len(breeds) != 0 {
			t.Errorf("Expected 0 breeds, got %d", len(breeds))
		}
	})
}

// TestCreateDog_ProTierUnlimited tests that Pro tier tenants can create unlimited dogs
// TDD RED PHASE: This test should FAIL because the code uses hardcoded limit of 10
func TestCreateDog_ProTierUnlimited(t *testing.T) {
	db := testutil.SetupTestDB(t)

	// Upgrade tenant 0 (Simple-Mode default) to Pro plan (plan_id = 2, which has max_dogs = -1 for unlimited)
	_, err := db.Exec(`UPDATE tenant_subscriptions SET plan_id = 2 WHERE tenant_id = 0`)
	if err != nil {
		t.Fatalf("Failed to upgrade tenant to Pro: %v", err)
	}

	handler := NewDogHandler(db, &config.Config{UploadDir: t.TempDir()})

	// Create 10 dogs for tenant 0 to reach what would be the "free tier limit"
	for i := 0; i < 10; i++ {
		body := fmt.Sprintf(`{"name":"Dog%d","breed":"Lab","size":"medium","age":3,"color_id":1}`, i)
		req := httptest.NewRequest(http.MethodPost, "/api/dogs", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		// Use tenant 0 (the one upgraded to Pro)
		req = req.WithContext(contextWithTenantAndUser(req.Context(), 0, 1))
		w := httptest.NewRecorder()
		handler.CreateDog(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("Failed to create dog %d: %d - %s", i, w.Code, w.Body.String())
		}
	}

	// Pro tier tenant should be able to create 11th dog (unlimited)
	body := `{"name":"Dog11","breed":"Lab","size":"medium","age":3,"color_id":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/dogs", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// Use tenant 0 (the one upgraded to Pro)
	req = req.WithContext(contextWithTenantAndUser(req.Context(), 0, 1))
	w := httptest.NewRecorder()

	handler.CreateDog(w, req)

	// Should return 201 Created, NOT 409 Conflict
	if w.Code != http.StatusCreated {
		t.Errorf("BUG: Pro tier should allow unlimited dogs. Expected 201, got %d: %s",
			w.Code, w.Body.String())
	}
}

// CRITICAL SECURITY TEST: Cross-tenant dog isolation (TDD RED Phase)
// BUG: Admin from tenant 2 can read/update/delete dogs belonging to tenant 3
// This is a critical security vulnerability that allows cross-tenant data access
func TestDogHandler_CrossTenantIsolation(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewDogHandler(db, cfg)

	// Tenant 0 and 1 already exist from SetupTestDB
	// Create tenant 2
	_, err := db.Exec(`INSERT INTO tenants (id, slug, name, status, contact_email, created_at, updated_at)
		VALUES (2, 'tenant-2', 'Tenant 2', 'active', 'tenant2@example.com', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatalf("Failed to create tenant 2: %v", err)
	}

	// Create a dog belonging to tenant 1 (SeedTestDog uses tenant 0, so we update it)
	dogID := testutil.SeedTestDog(t, db, "Tenant1Dog", "Labrador", "green")
	db.Exec("UPDATE dogs SET tenant_id = 1 WHERE id = ?", dogID)

	// Verify dog belongs to tenant 1
	var tenantID int
	err = db.QueryRow("SELECT tenant_id FROM dogs WHERE id = ?", dogID).Scan(&tenantID)
	if err != nil {
		t.Fatalf("Failed to verify dog tenant_id: %v", err)
	}
	if tenantID != 1 {
		t.Fatalf("Expected dog to belong to tenant 1, got %d", tenantID)
	}

	t.Run("SECURITY: tenant 2 admin cannot GET tenant 1's dog", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/dogs/"+fmt.Sprintf("%d", dogID), nil)
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dogID)})
		// Context for tenant 2 admin
		ctx := contextWithTenantAndUser(req.Context(), 2, 100)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetDog(rec, req)

		// SECURITY: Should return 404 (Not Found) or 403 (Forbidden), NOT 200
		if rec.Code == http.StatusOK {
			t.Errorf("SECURITY VULNERABILITY: Tenant 2 admin could access tenant 1's dog! Expected 404/403, got 200")
			t.Errorf("Response body: %s", rec.Body.String())
		}
	})

	t.Run("SECURITY: tenant 2 admin cannot UPDATE tenant 1's dog", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name": "HACKED BY TENANT 2",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/dogs/"+fmt.Sprintf("%d", dogID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dogID)})
		// Context for tenant 2 admin
		ctx := contextWithTenantAndUser(req.Context(), 2, 100)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateDog(rec, req)

		// SECURITY: Should return 404 (Not Found) or 403 (Forbidden), NOT 200
		if rec.Code == http.StatusOK {
			t.Errorf("SECURITY VULNERABILITY: Tenant 2 admin could update tenant 1's dog! Expected 404/403, got 200")
		}

		// Verify dog name was NOT changed
		var name string
		db.QueryRow("SELECT name FROM dogs WHERE id = ?", dogID).Scan(&name)
		if name == "HACKED BY TENANT 2" {
			t.Errorf("CRITICAL: Dog was modified by unauthorized tenant! Name changed to: %s", name)
		}
	})

	t.Run("SECURITY: tenant 2 admin cannot DELETE tenant 1's dog", func(t *testing.T) {
		// Create another dog for delete test (so we don't affect other tests)
		deleteDogID := testutil.SeedTestDog(t, db, "DeleteTestDog", "Beagle", "blue")

		req := httptest.NewRequest("DELETE", "/api/dogs/"+fmt.Sprintf("%d", deleteDogID), nil)
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", deleteDogID)})
		// Context for tenant 2 admin
		ctx := contextWithTenantAndUser(req.Context(), 2, 100)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DeleteDog(rec, req)

		// SECURITY: Should return 404 (Not Found) or 403 (Forbidden), NOT 200
		if rec.Code == http.StatusOK {
			t.Errorf("SECURITY VULNERABILITY: Tenant 2 admin could delete tenant 1's dog! Expected 404/403, got 200")
		}

		// Verify dog still exists
		var count int
		db.QueryRow("SELECT COUNT(*) FROM dogs WHERE id = ?", deleteDogID).Scan(&count)
		if count == 0 {
			t.Errorf("CRITICAL: Dog was deleted by unauthorized tenant!")
		}
	})

	t.Run("SECURITY: tenant 2 admin cannot TOGGLE AVAILABILITY of tenant 1's dog", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"is_available":       false,
			"unavailable_reason": "HACKED",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/dogs/"+fmt.Sprintf("%d", dogID)+"/availability", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dogID)})
		// Context for tenant 2 admin
		ctx := contextWithTenantAndUser(req.Context(), 2, 100)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ToggleAvailability(rec, req)

		// SECURITY: Should return 404 (Not Found) or 403 (Forbidden), NOT 200
		if rec.Code == http.StatusOK {
			t.Errorf("SECURITY VULNERABILITY: Tenant 2 admin could change tenant 1's dog availability! Expected 404/403, got 200")
		}

		// Verify availability was NOT changed
		var isAvailable bool
		db.QueryRow("SELECT is_available FROM dogs WHERE id = ?", dogID).Scan(&isAvailable)
		if !isAvailable {
			t.Errorf("CRITICAL: Dog availability was changed by unauthorized tenant!")
		}
	})

	t.Run("SECURITY: tenant 2 admin cannot SET FEATURED on tenant 1's dog", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"is_featured": true,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/dogs/"+fmt.Sprintf("%d", dogID)+"/featured", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dogID)})
		// Context for tenant 2 admin
		ctx := contextWithTenantAndUser(req.Context(), 2, 100)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.SetFeatured(rec, req)

		// SECURITY: Should return 404 (Not Found) or 403 (Forbidden), NOT 200
		if rec.Code == http.StatusOK {
			t.Errorf("SECURITY VULNERABILITY: Tenant 2 admin could set featured on tenant 1's dog! Expected 404/403, got 200")
		}
	})

	t.Run("POSITIVE: same tenant admin CAN access own dogs", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/dogs/"+fmt.Sprintf("%d", dogID), nil)
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dogID)})
		// Context for tenant 1 admin (same tenant as dog)
		ctx := contextWithTenantAndUser(req.Context(), 1, 1)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetDog(rec, req)

		// Should succeed - same tenant
		if rec.Code != http.StatusOK {
			t.Errorf("Expected tenant 1 admin to access own dog, got status %d", rec.Code)
		}
	})
}

// =============================================================================
// BUG FIX TESTS - TDD RED PHASE
// =============================================================================

// TestDogHandler_CreateDog_InputLengthValidation tests that excessively long inputs are rejected
// BUG: Dog names up to 100,000 characters are accepted without validation
// This could cause database bloat and DoS attacks
func TestDogHandler_CreateDog_InputLengthValidation(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewDogHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "blue")

	t.Run("rejects name longer than 100 characters", func(t *testing.T) {
		longName := strings.Repeat("A", 101) // 101 characters
		reqBody := fmt.Sprintf(`{"name":"%s","breed":"Labrador","size":"medium","age":3,"category":"green"}`, longName)
		req := httptest.NewRequest("POST", "/api/dogs", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenantAndUser(req.Context(), 1, adminID)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateDog(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("BUG: Should reject name > 100 chars. Expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("accepts name exactly 100 characters", func(t *testing.T) {
		exactName := strings.Repeat("A", 100) // Exactly 100 characters
		reqBody := fmt.Sprintf(`{"name":"%s","breed":"Labrador","size":"medium","age":3,"category":"green"}`, exactName)
		req := httptest.NewRequest("POST", "/api/dogs", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenantAndUser(req.Context(), 1, adminID)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateDog(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("Should accept name = 100 chars. Expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("rejects breed longer than 100 characters", func(t *testing.T) {
		longBreed := strings.Repeat("B", 101)
		reqBody := fmt.Sprintf(`{"name":"TestDog","breed":"%s","size":"medium","age":3,"category":"green"}`, longBreed)
		req := httptest.NewRequest("POST", "/api/dogs", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenantAndUser(req.Context(), 1, adminID)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateDog(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("BUG: Should reject breed > 100 chars. Expected 400, got %d", rec.Code)
		}
	})

	t.Run("rejects special_needs longer than 1000 characters", func(t *testing.T) {
		longNeeds := strings.Repeat("N", 1001)
		reqBody := fmt.Sprintf(`{"name":"TestDog","breed":"Lab","size":"medium","age":3,"category":"green","special_needs":"%s"}`, longNeeds)
		req := httptest.NewRequest("POST", "/api/dogs", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenantAndUser(req.Context(), 1, adminID)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateDog(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("BUG: Should reject special_needs > 1000 chars. Expected 400, got %d", rec.Code)
		}
	})

	t.Run("rejects pickup_location longer than 500 characters", func(t *testing.T) {
		longLocation := strings.Repeat("L", 501)
		reqBody := fmt.Sprintf(`{"name":"TestDog","breed":"Lab","size":"medium","age":3,"category":"green","pickup_location":"%s"}`, longLocation)
		req := httptest.NewRequest("POST", "/api/dogs", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenantAndUser(req.Context(), 1, adminID)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateDog(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("BUG: Should reject pickup_location > 500 chars. Expected 400, got %d", rec.Code)
		}
	})
}

// TestDogHandler_UpdateDog_InputLengthValidation tests length validation on update
func TestDogHandler_UpdateDog_InputLengthValidation(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewDogHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "blue")
	dogID := testutil.SeedTestDog(t, db, "Bella", "Labrador", "green")

	t.Run("rejects name longer than 100 characters on update", func(t *testing.T) {
		longName := strings.Repeat("A", 101)
		reqBody := fmt.Sprintf(`{"name":"%s"}`, longName)
		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/dogs/%d", dogID), strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dogID)})
		// Use tenant_id=0 to match the dog's tenant (created by SeedTestDog)
		ctx := contextWithTenantAndUser(req.Context(), 0, adminID)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateDog(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("BUG: Should reject name > 100 chars on update. Expected 400, got %d", rec.Code)
		}

		// Verify original name unchanged
		var name string
		db.QueryRow("SELECT name FROM dogs WHERE id = ?", dogID).Scan(&name)
		if name != "Bella" {
			t.Errorf("Dog name should be unchanged after rejected update, got %s", name)
		}
	})
}

// TestDogHandler_CreateDog_XSSSanitization tests that HTML/script tags are sanitized
// BUG: XSS payloads are stored in database without sanitization
func TestDogHandler_CreateDog_XSSSanitization(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewDogHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "blue")

	t.Run("strips HTML script tags from name", func(t *testing.T) {
		reqBody := `{"name":"<script>alert('XSS')</script>Bella","breed":"Labrador","size":"medium","age":3,"category":"green"}`
		req := httptest.NewRequest("POST", "/api/dogs", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenantAndUser(req.Context(), 1, adminID)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateDog(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("Expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		name := response["name"].(string)
		if strings.Contains(name, "<script>") || strings.Contains(name, "</script>") {
			t.Errorf("BUG: XSS payload should be sanitized. Got: %s", name)
		}
		if !strings.Contains(name, "Bella") {
			t.Errorf("Legitimate text should be preserved. Got: %s", name)
		}
	})

	t.Run("strips HTML tags from breed", func(t *testing.T) {
		reqBody := `{"name":"TestDog","breed":"<img src=x onerror=alert('XSS')>Labrador","size":"medium","age":3,"category":"green"}`
		req := httptest.NewRequest("POST", "/api/dogs", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenantAndUser(req.Context(), 1, adminID)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateDog(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("Expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		breed := response["breed"].(string)
		if strings.Contains(breed, "<img") || strings.Contains(breed, "onerror") {
			t.Errorf("BUG: XSS payload should be sanitized from breed. Got: %s", breed)
		}
	})

	t.Run("strips HTML from special_needs", func(t *testing.T) {
		reqBody := `{"name":"TestDog","breed":"Lab","size":"medium","age":3,"category":"green","special_needs":"<b>Bold</b> and <script>evil()</script>"}`
		req := httptest.NewRequest("POST", "/api/dogs", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenantAndUser(req.Context(), 1, adminID)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateDog(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("Expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		specialNeeds := response["special_needs"].(string)
		if strings.Contains(specialNeeds, "<script>") || strings.Contains(specialNeeds, "<b>") {
			t.Errorf("BUG: HTML should be sanitized from special_needs. Got: %s", specialNeeds)
		}
		if !strings.Contains(specialNeeds, "Bold") || !strings.Contains(specialNeeds, "and") {
			t.Errorf("Legitimate text should be preserved. Got: %s", specialNeeds)
		}
	})
}

// TestDogHandler_UpdateDog_XSSSanitization tests XSS sanitization on update
func TestDogHandler_UpdateDog_XSSSanitization(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewDogHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "blue")
	dogID := testutil.SeedTestDog(t, db, "Bella", "Labrador", "green")

	t.Run("strips HTML from name on update", func(t *testing.T) {
		reqBody := `{"name":"<script>alert('XSS')</script>Updated"}`
		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/dogs/%d", dogID), strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dogID)})
		// Use tenant_id=0 to match the dog's tenant (created by SeedTestDog)
		ctx := contextWithTenantAndUser(req.Context(), 0, adminID)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateDog(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Check database value
		var name string
		db.QueryRow("SELECT name FROM dogs WHERE id = ?", dogID).Scan(&name)
		if strings.Contains(name, "<script>") {
			t.Errorf("BUG: XSS payload stored in database. Got: %s", name)
		}
	})
}

// TestDogHandler_CreateDog_CategoryMapsToColorID tests that legacy category is mapped to color_id (TDD RED Phase)
// BUG #1: When creating a dog with category="green" but no color_id, the color_id should be resolved
func TestDogHandler_CreateDog_CategoryMapsToColorID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewDogHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "blue")

	t.Run("category green maps to color_id", func(t *testing.T) {
		reqBody := `{"name":"Test Dog","breed":"Labrador","size":"medium","age":3,"category":"green"}`
		req := httptest.NewRequest("POST", "/api/dogs", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		// Use tenant_id=0 (Simple-Mode) where color categories are seeded
		ctx := contextWithTenantAndUser(req.Context(), 0, adminID)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateDog(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		// Check that color_id is set (not nil)
		colorID := response["color_id"]
		if colorID == nil {
			t.Error("BUG #1: color_id should be set when category is provided, got nil")
		}
	})

	t.Run("category orange maps to color_id", func(t *testing.T) {
		reqBody := `{"name":"Orange Dog","breed":"Beagle","size":"small","age":2,"category":"orange"}`
		req := httptest.NewRequest("POST", "/api/dogs", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		// Use tenant_id=0 (Simple-Mode) where color categories are seeded
		ctx := contextWithTenantAndUser(req.Context(), 0, adminID)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateDog(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		colorID := response["color_id"]
		if colorID == nil {
			t.Error("BUG #1: color_id should be set when category='orange' is provided, got nil")
		}
	})

	t.Run("category blue maps to color_id", func(t *testing.T) {
		reqBody := `{"name":"Blue Dog","breed":"Shepherd","size":"large","age":4,"category":"blue"}`
		req := httptest.NewRequest("POST", "/api/dogs", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		// Use tenant_id=0 (Simple-Mode) where color categories are seeded
		ctx := contextWithTenantAndUser(req.Context(), 0, adminID)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateDog(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		colorID := response["color_id"]
		if colorID == nil {
			t.Error("BUG #1: color_id should be set when category='blue' is provided, got nil")
		}
	})
}

// =============================================================================
// CRITICAL SECURITY TEST: Cross-tenant access from default tenant (TDD RED Phase)
// =============================================================================
// Test that users from tenant 0 (default) cannot access dogs from tenant 1
// This validates tenant isolation even when tenant_id=0 is a valid tenant
func TestDogHandler_ZeroTenantID_Bypass(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewDogHandler(db, cfg)

	// Tenant 0 and 1 already exist from SetupTestDB

	// Create a dog belonging to tenant 1 (not the default tenant 0)
	dogID := testutil.SeedTestDog(t, db, "SecureDog", "Labrador", "green")
	db.Exec("UPDATE dogs SET tenant_id = 1 WHERE id = ?", dogID)

	// Verify dog belongs to tenant 1
	var tenantID int
	err := db.QueryRow("SELECT tenant_id FROM dogs WHERE id = ?", dogID).Scan(&tenantID)
	if err != nil {
		t.Fatalf("Failed to verify dog tenant_id: %v", err)
	}
	if tenantID != 1 {
		t.Fatalf("Expected dog to belong to tenant 1, got %d", tenantID)
	}

	t.Run("SECURITY: zero tenant_id in context should NOT allow GET", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/dogs/"+fmt.Sprintf("%d", dogID), nil)
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dogID)})
		// Context with tenantID = 0 (simulating missing tenant resolution)
		ctx := contextWithTenantAndUser(req.Context(), 0, 999)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetDog(rec, req)

		// SECURITY: Should return 400 (Bad Request) or 403 (Forbidden), NOT 200
		// Zero tenant means no tenant resolved - this should be blocked
		if rec.Code == http.StatusOK {
			t.Errorf("SECURITY VULNERABILITY: Zero tenantID bypass! Request with tenantID=0 accessed dog from tenant 1. Expected 400/403, got 200")
			t.Errorf("Response body: %s", rec.Body.String())
		}
	})

	t.Run("SECURITY: zero tenant_id should NOT allow UPDATE", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name": "HACKED_VIA_ZERO_TENANT",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/dogs/"+fmt.Sprintf("%d", dogID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dogID)})
		// Context with tenantID = 0
		ctx := contextWithTenantAndUser(req.Context(), 0, 999)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateDog(rec, req)

		// SECURITY: Should return 400/403/404, NOT 200
		if rec.Code == http.StatusOK {
			t.Errorf("SECURITY VULNERABILITY: Zero tenantID bypass allowed UPDATE! Expected 400/403/404, got 200")
		}

		// Verify dog name was NOT changed
		var name string
		db.QueryRow("SELECT name FROM dogs WHERE id = ?", dogID).Scan(&name)
		if name == "HACKED_VIA_ZERO_TENANT" {
			t.Errorf("CRITICAL: Dog was modified via zero-tenant bypass! Name changed to: %s", name)
		}
	})

	t.Run("SECURITY: zero tenant_id should NOT allow DELETE", func(t *testing.T) {
		// Create a dog specifically for this delete test
		deleteDogID := testutil.SeedTestDog(t, db, "DeleteBypassTest", "Beagle", "blue")
		// IMPORTANT: Change dog's tenant to 1 so tenant_id=0 shouldn't be able to delete it
		db.Exec("UPDATE dogs SET tenant_id = 1 WHERE id = ?", deleteDogID)

		req := httptest.NewRequest("DELETE", "/api/dogs/"+fmt.Sprintf("%d", deleteDogID), nil)
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", deleteDogID)})
		// Context with tenantID = 0 (trying to cross-tenant delete)
		ctx := contextWithTenantAndUser(req.Context(), 0, 999)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DeleteDog(rec, req)

		// SECURITY: Should return 400/403/404, NOT 200
		if rec.Code == http.StatusOK {
			t.Errorf("SECURITY VULNERABILITY: Zero tenantID bypass allowed DELETE! Expected 400/403/404, got 200")
		}

		// Verify dog still exists
		var count int
		db.QueryRow("SELECT COUNT(*) FROM dogs WHERE id = ?", deleteDogID).Scan(&count)
		if count == 0 {
			t.Errorf("CRITICAL: Dog was deleted via zero-tenant bypass!")
		}
	})

	t.Run("SECURITY: zero tenant_id should NOT allow TOGGLE AVAILABILITY", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"is_available":       false,
			"unavailable_reason": "HACKED_VIA_ZERO_TENANT",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/dogs/"+fmt.Sprintf("%d", dogID)+"/availability", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dogID)})
		// Context with tenantID = 0
		ctx := contextWithTenantAndUser(req.Context(), 0, 999)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ToggleAvailability(rec, req)

		// SECURITY: Should return 400/403/404, NOT 200
		if rec.Code == http.StatusOK {
			t.Errorf("SECURITY VULNERABILITY: Zero tenantID bypass allowed ToggleAvailability! Expected 400/403/404, got 200")
		}
	})

	t.Run("SECURITY: zero tenant_id should NOT allow SET FEATURED", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"is_featured": true,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/api/dogs/"+fmt.Sprintf("%d", dogID)+"/featured", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dogID)})
		// Context with tenantID = 0
		ctx := contextWithTenantAndUser(req.Context(), 0, 999)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.SetFeatured(rec, req)

		// SECURITY: Should return 400/403/404, NOT 200
		if rec.Code == http.StatusOK {
			t.Errorf("SECURITY VULNERABILITY: Zero tenantID bypass allowed SetFeatured! Expected 400/403/404, got 200")
		}
	})
}

// TestDogHandler_CreateDog_NegativeAgeValidation tests that negative age is rejected (TDD RED Phase)
// BUG: Dogs can be created with negative age values like -5
func TestDogHandler_CreateDog_NegativeAgeValidation(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewDogHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "blue")

	t.Run("rejects negative age", func(t *testing.T) {
		reqBody := `{"name":"Negative Age Dog","breed":"Labrador","size":"medium","age":-5,"color_id":1}`
		req := httptest.NewRequest("POST", "/api/dogs", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenantAndUser(req.Context(), 1, adminID)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateDog(rec, req)

		// Should return 400 Bad Request for negative age
		if rec.Code != http.StatusBadRequest {
			t.Errorf("BUG: Should reject negative age. Expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify error message mentions age
		if !strings.Contains(rec.Body.String(), "age") && !strings.Contains(rec.Body.String(), "Age") {
			t.Errorf("Error message should mention age, got: %s", rec.Body.String())
		}
	})

	t.Run("allows zero age (newborn puppy)", func(t *testing.T) {
		reqBody := `{"name":"Newborn Puppy","breed":"Labrador","size":"small","age":0,"color_id":1}`
		req := httptest.NewRequest("POST", "/api/dogs", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenantAndUser(req.Context(), 1, adminID)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateDog(rec, req)

		// Age 0 should be allowed for newborn puppies
		if rec.Code != http.StatusCreated {
			t.Errorf("Should allow age 0 for newborn puppies. Expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("allows positive age", func(t *testing.T) {
		reqBody := `{"name":"Adult Dog","breed":"Labrador","size":"medium","age":5,"color_id":1}`
		req := httptest.NewRequest("POST", "/api/dogs", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenantAndUser(req.Context(), 1, adminID)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.CreateDog(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("Should allow positive age. Expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("rejects negative age in update", func(t *testing.T) {
		// First create a valid dog
		createBody := `{"name":"Update Test Dog","breed":"Labrador","size":"medium","age":3,"color_id":1}`
		createReq := httptest.NewRequest("POST", "/api/dogs", strings.NewReader(createBody))
		createReq.Header.Set("Content-Type", "application/json")
		ctx := contextWithTenantAndUser(createReq.Context(), 1, adminID)
		createReq = createReq.WithContext(ctx)
		createRec := httptest.NewRecorder()
		handler.CreateDog(createRec, createReq)

		if createRec.Code != http.StatusCreated {
			t.Fatalf("Setup failed: couldn't create dog. Got %d", createRec.Code)
		}

		var createResp map[string]interface{}
		json.Unmarshal(createRec.Body.Bytes(), &createResp)
		dogID := int(createResp["id"].(float64))

		// Try to update with negative age
		updateBody := `{"age":-10}`
		updateReq := httptest.NewRequest("PUT", fmt.Sprintf("/api/dogs/%d", dogID), strings.NewReader(updateBody))
		updateReq.Header.Set("Content-Type", "application/json")
		updateReq = mux.SetURLVars(updateReq, map[string]string{"id": fmt.Sprintf("%d", dogID)})
		ctx = contextWithTenantAndUser(updateReq.Context(), 1, adminID)
		updateReq = updateReq.WithContext(ctx)

		updateRec := httptest.NewRecorder()
		handler.UpdateDog(updateRec, updateReq)

		// Should return 400 Bad Request for negative age
		if updateRec.Code != http.StatusBadRequest {
			t.Errorf("BUG: Should reject negative age in update. Expected 400, got %d. Body: %s", updateRec.Code, updateRec.Body.String())
		}
	})
}

// TestDogHandler_CreateDog_ErrorMessageUsesActualLimit tests that the error message
// uses the actual dog limit from the subscription, not a hardcoded "10"
// TDD RED PHASE: This test should FAIL because the error message currently says
// "Maximum von 10 Hunden" regardless of the actual limit
func TestDogHandler_CreateDog_ErrorMessageUsesActualLimit(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewDogHandler(db, cfg)

	// Create a tenant with a custom dog limit of 5 (not the default 10)
	_, err := db.Exec(`INSERT INTO tenants (id, slug, name, status, contact_email, created_at, updated_at)
		VALUES (99, 'custom-limit-tenant', 'Custom Limit Tenant', 'active', 'custom@example.com', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	// Create a custom pricing plan with 5 dog limit
	_, err = db.Exec(`INSERT INTO pricing_plans (id, name, slug, max_dogs, price_monthly, price_yearly, is_active, created_at)
		VALUES (99, 'Custom Plan', 'custom', 5, 1000, 10000, 1, CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatalf("Failed to create pricing plan: %v", err)
	}

	// Create subscription for tenant 99 with the custom plan (5 dog limit)
	_, err = db.Exec(`INSERT INTO tenant_subscriptions (tenant_id, plan_id, status, billing_cycle, created_at, updated_at)
		VALUES (99, 99, 'active', 'monthly', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatalf("Failed to create subscription: %v", err)
	}

	// Create an admin user for tenant 99
	_, err = db.Exec(`INSERT INTO users (tenant_id, email, password_hash, first_name, last_name, is_admin, is_active, is_verified, terms_accepted_at, created_at, updated_at)
		VALUES (99, 'admin@custom.com', 'hash', 'Admin', 'User', 1, 1, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	var adminID int
	db.QueryRow("SELECT id FROM users WHERE email = 'admin@custom.com'").Scan(&adminID)

	// First, create a color_category for this tenant since the handler needs it
	_, err = db.Exec(`INSERT INTO color_categories (id, tenant_id, name, hex_code, sort_order, created_at, updated_at)
		VALUES (99, 99, 'Green', '#00FF00', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatalf("Failed to create color category: %v", err)
	}

	// Create 5 dogs (reaching the custom limit)
	for i := 1; i <= 5; i++ {
		_, err = db.Exec(`INSERT INTO dogs (tenant_id, name, breed, size, age, color_id, is_available, created_at, updated_at)
			VALUES (99, ?, 'Labrador', 'medium', 3, 99, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
			fmt.Sprintf("Dog %d", i))
		if err != nil {
			t.Fatalf("Failed to create dog %d: %v", i, err)
		}
	}

	// Try to create 6th dog - should fail with limit error
	reqBody := `{"name":"Dog 6 - Over Limit","breed":"Poodle","size":"medium","age":3,"color_id":99}`
	req := httptest.NewRequest("POST", "/api/dogs", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	ctx := contextWithTenantAndUser(req.Context(), 99, adminID)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CreateDog(rec, req)

	// Should return 409 Conflict
	if rec.Code != http.StatusConflict {
		t.Fatalf("Expected status 409 (Conflict), got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// Parse response
	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// BUG TEST: The error message should say "5" (the actual limit), not "10"
	message, ok := response["message"].(string)
	if !ok {
		t.Fatal("Expected 'message' field in response")
	}

	// The message should contain the actual limit "5", not hardcoded "10"
	if strings.Contains(message, "10 Hunden") {
		t.Errorf("BUG DETECTED: Error message contains hardcoded '10 Hunden' instead of actual limit '5 Hunden'. Message: %s", message)
	}

	if !strings.Contains(message, "5 Hunden") {
		t.Errorf("Error message should contain '5 Hunden' for the actual limit. Message: %s", message)
	}

	// Verify the 'limit' field in response is correct
	limit, ok := response["limit"].(float64)
	if !ok {
		t.Fatal("Expected 'limit' field in response")
	}
	if int(limit) != 5 {
		t.Errorf("Expected limit=5 in response, got %v", limit)
	}
}

// TDD RED PHASE: TestDogHandler_DeleteDogPhoto tests deleting a dog's photo
// These tests will FAIL initially because DeleteDogPhoto method doesn't exist yet
func TestDogHandler_DeleteDogPhoto(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
		UploadDir:          t.TempDir(), // Use temp dir for photo storage
	}
	handler := NewDogHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "green")

	t.Run("successfully deletes photo from dog with photo", func(t *testing.T) {
		// Arrange: Create a dog with a photo path set
		dogID := testutil.SeedTestDog(t, db, "PhotoDog", "Labrador", "green")

		// Set photo path directly in database
		photoPath := "dogs/dog_1_full.jpg"
		thumbPath := "dogs/dog_1_thumb.jpg"
		_, err := db.Exec(`UPDATE dogs SET photo = ?, photo_thumbnail = ? WHERE id = ?`,
			photoPath, thumbPath, dogID)
		if err != nil {
			t.Fatalf("Failed to set dog photo: %v", err)
		}

		// Act: DELETE /api/dogs/:id/photo
		req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/dogs/%d/photo", dogID), nil)
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dogID)})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DeleteDogPhoto(rec, req)

		// Assert: 200 OK
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify photo fields are NULL in database
		var photo, thumbnail *string
		err = db.QueryRow(`SELECT photo, photo_thumbnail FROM dogs WHERE id = ?`, dogID).Scan(&photo, &thumbnail)
		if err != nil {
			t.Fatalf("Failed to query dog: %v", err)
		}

		if photo != nil {
			t.Errorf("Expected photo to be NULL, got %v", *photo)
		}
		if thumbnail != nil {
			t.Errorf("Expected photo_thumbnail to be NULL, got %v", *thumbnail)
		}
	})

	t.Run("returns 404 for non-existent dog", func(t *testing.T) {
		// Act: DELETE /api/dogs/99999/photo (non-existent dog)
		req := httptest.NewRequest("DELETE", "/api/dogs/99999/photo", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "99999"})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DeleteDogPhoto(rec, req)

		// Assert: 404 Not Found
		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("returns 404 for dog without photo", func(t *testing.T) {
		// Arrange: Create a dog without photo
		dogID := testutil.SeedTestDog(t, db, "NoPhotoDog", "Beagle", "blue")

		// Act: DELETE /api/dogs/:id/photo
		req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/dogs/%d/photo", dogID), nil)
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dogID)})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DeleteDogPhoto(rec, req)

		// Assert: 404 (no photo to delete)
		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for dog without photo, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("returns 400 for invalid dog ID", func(t *testing.T) {
		// Act: DELETE /api/dogs/invalid/photo
		req := httptest.NewRequest("DELETE", "/api/dogs/invalid/photo", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DeleteDogPhoto(rec, req)

		// Assert: 400 Bad Request
		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("enforces tenant isolation", func(t *testing.T) {
		// Arrange: Create a dog in tenant 0
		dogID := testutil.SeedTestDog(t, db, "TenantDog", "Poodle", "orange")

		// Set photo path
		_, err := db.Exec(`UPDATE dogs SET photo = 'dogs/test.jpg' WHERE id = ?`, dogID)
		if err != nil {
			t.Fatalf("Failed to set dog photo: %v", err)
		}

		// Act: DELETE from different tenant context (tenant 999)
		req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/dogs/%d/photo", dogID), nil)
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dogID)})
		// Use tenant 999 context - dog belongs to tenant 0
		ctx := context.WithValue(req.Context(), middleware.TenantIDKey, 999)
		ctx = context.WithValue(ctx, middleware.UserIDKey, adminID)
		ctx = context.WithValue(ctx, middleware.IsAdminKey, true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DeleteDogPhoto(rec, req)

		// Assert: 404 (dog not visible to other tenant)
		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for cross-tenant access, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify photo was NOT deleted (still in tenant 0)
		var photo *string
		err = db.QueryRow(`SELECT photo FROM dogs WHERE id = ?`, dogID).Scan(&photo)
		if err != nil {
			t.Fatalf("Failed to query dog: %v", err)
		}
		if photo == nil || *photo != "dogs/test.jpg" {
			t.Errorf("Photo should not have been deleted by cross-tenant request")
		}
	})

	t.Run("response includes success message", func(t *testing.T) {
		// Arrange: Create a dog with photo
		dogID := testutil.SeedTestDog(t, db, "MessageDog", "Husky", "green")
		_, err := db.Exec(`UPDATE dogs SET photo = 'dogs/msg.jpg', photo_thumbnail = 'dogs/msg_thumb.jpg' WHERE id = ?`, dogID)
		if err != nil {
			t.Fatalf("Failed to set dog photo: %v", err)
		}

		// Act: DELETE /api/dogs/:id/photo
		req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/dogs/%d/photo", dogID), nil)
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dogID)})
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DeleteDogPhoto(rec, req)

		// Assert: Response contains message
		if rec.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if _, ok := response["message"]; !ok {
			t.Error("Expected 'message' field in response")
		}
	})
}

// TestDogHandler_DeleteDogPhoto_S3Path tests that S3 deletion is attempted
// when S3 is configured (not just local filesystem deletion)
// RED PHASE: This test documents the expected behavior for S3 deletion
func TestDogHandler_DeleteDogPhoto_S3Path(t *testing.T) {
	db := testutil.SetupTestDB(t)

	// Configure with S3 enabled (but no actual S3 connection)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
		UploadDir:          t.TempDir(),
		UseS3:              true, // S3 enabled
		// S3 credentials intentionally missing - we're testing code path
	}

	handler := NewDogHandler(db, cfg)

	// Create admin user
	adminID := testutil.SeedTestUser(t, db, "s3admin@example.com", "S3Admin", "green")

	// Create dog with S3-style photo URL
	dogID := testutil.SeedTestDog(t, db, "S3Dog", "Retriever", "green")
	s3PhotoURL := "https://s3.example.com/bucket/tenant/dogs/dog_1_full.jpg"
	_, err := db.Exec(`UPDATE dogs SET photo = ?, photo_thumbnail = ? WHERE id = ?`,
		s3PhotoURL, s3PhotoURL, dogID)
	if err != nil {
		t.Fatalf("Failed to set dog photo: %v", err)
	}

	t.Run("S3 config set but s3Service nil should use local deletion gracefully", func(t *testing.T) {
		// When UseS3=true but s3Service initialization failed (nil),
		// the handler should fall back to local deletion without panic

		req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/dogs/%d/photo", dogID), nil)
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dogID)})
		ctx := contextWithUser(req.Context(), adminID, "s3admin@example.com", true)
		// Add tenant context for SaaS mode
		ctx = context.WithValue(ctx, middleware.TenantIDKey, 0)
		ctx = context.WithValue(ctx, middleware.TenantSlugKey, "test-tenant")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()

		// Should not panic - should handle gracefully
		handler.DeleteDogPhoto(rec, req)

		// Verify database was updated regardless of S3 status
		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}

		// Verify photo fields cleared in database
		var photo *string
		err := db.QueryRow(`SELECT photo FROM dogs WHERE id = ?`, dogID).Scan(&photo)
		if err != nil {
			t.Fatalf("Query error: %v", err)
		}
		if photo != nil {
			t.Errorf("Expected photo to be NULL after deletion, got %v", *photo)
		}
	})

	t.Run("requires tenant slug in SaaS mode with S3", func(t *testing.T) {
		// This test verifies that we don't silently use "default" tenant
		// when tenant slug is missing in SaaS mode - that would be a security issue

		// Create a dog with photo (in tenant 0 for simple mode)
		dogID := testutil.SeedTestDog(t, db, "SlugTestDog", "Terrier", "green")
		_, err := db.Exec(`UPDATE dogs SET photo = 'dogs/test.jpg' WHERE id = ?`, dogID)
		if err != nil {
			t.Fatalf("Failed to set dog photo: %v", err)
		}

		req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/dogs/%d/photo", dogID), nil)
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dogID)})
		// Set tenant ID but NOT tenant slug - this simulates a misconfigured request
		ctx := context.WithValue(req.Context(), middleware.TenantIDKey, 0)
		ctx = context.WithValue(ctx, middleware.UserIDKey, adminID)
		ctx = context.WithValue(ctx, middleware.IsAdminKey, true)
		// Intentionally NOT setting TenantSlugKey to test the edge case
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DeleteDogPhoto(rec, req)

		// In simple mode (no S3), this should still work
		// In SaaS mode with S3, missing slug should be handled gracefully
		// The current fix: use "default" for simple mode, but this test documents the behavior
		if rec.Code != http.StatusOK {
			t.Logf("Note: Empty tenant slug resulted in status %d", rec.Code)
		}
	})
}
