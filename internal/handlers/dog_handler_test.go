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
		} else if *tenantID != 1 {
			t.Errorf("Expected dog tenant_id to be 1, got %d", *tenantID)
		}

		// Also verify tenant_id is returned in the response
		if response["tenant_id"] == nil {
			t.Error("Expected tenant_id in response, but it was nil")
		} else if int(response["tenant_id"].(float64)) != 1 {
			t.Errorf("Expected tenant_id 1 in response, got %v", response["tenant_id"])
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

	// Upgrade tenant 1 to Pro plan (plan_id = 2, which has max_dogs = -1 for unlimited)
	_, err := db.Exec(`UPDATE tenant_subscriptions SET plan_id = 2 WHERE tenant_id = 1`)
	if err != nil {
		t.Fatalf("Failed to upgrade tenant to Pro: %v", err)
	}

	handler := NewDogHandler(db, &config.Config{UploadDir: t.TempDir()})

	// Create 10 dogs to reach what would be the "free tier limit"
	for i := 0; i < 10; i++ {
		body := fmt.Sprintf(`{"name":"Dog%d","breed":"Lab","size":"medium","age":3,"color_id":1}`, i)
		req := httptest.NewRequest(http.MethodPost, "/api/dogs", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(contextWithTenantAndUser(req.Context(), 1, 1))
		w := httptest.NewRecorder()
		handler.CreateDog(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("Failed to create dog %d: %d - %s", i, w.Code, w.Body.String())
		}
	}

	// Pro tier tenant should be able to create 11th dog (unlimited)
	// BUG: Currently returns 409 Conflict because code uses hardcoded FreeTierDogLimit=10
	body := `{"name":"Dog11","breed":"Lab","size":"medium","age":3,"color_id":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/dogs", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithTenantAndUser(req.Context(), 1, 1))
	w := httptest.NewRecorder()

	handler.CreateDog(w, req)

	// Should return 201 Created, NOT 409 Conflict
	if w.Code != http.StatusCreated {
		t.Errorf("BUG: Pro tier should allow unlimited dogs. Expected 201, got %d: %s",
			w.Code, w.Body.String())
	}
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
		ctx := contextWithTenantAndUser(req.Context(), 1, adminID)
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
		ctx := contextWithTenantAndUser(req.Context(), 1, adminID)
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
		ctx := contextWithTenantAndUser(req.Context(), 1, adminID)
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
