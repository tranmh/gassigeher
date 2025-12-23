package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// contextWithCentralAdmin creates a context with central admin authentication
func contextWithCentralAdmin(ctx context.Context, userID int, email string) context.Context {
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, middleware.EmailKey, email)
	ctx = context.WithValue(ctx, middleware.IsAdminKey, true)
	ctx = context.WithValue(ctx, middleware.IsSuperAdminKey, false)
	ctx = context.WithValue(ctx, middleware.IsCentralAdminKey, true)
	ctx = context.WithValue(ctx, middleware.TenantIDKey, 0) // Central admin has no tenant
	return ctx
}

// TestCentralAdminHandler_GetPlatformStats tests getting platform-wide statistics
func TestCentralAdminHandler_GetPlatformStats(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewCentralAdminHandler(db, cfg)

	// Create a central admin user
	adminID := testutil.SeedTestUser(t, db, "central@example.com", "Central Admin", "green")
	db.Exec("UPDATE users SET is_central_admin = 1 WHERE id = ?", adminID)

	// Note: SetupTestDB already creates tenant with id=1 and slug='test-tenant'
	// We can use this existing tenant for stats testing

	t.Run("returns platform stats", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/central-admin/stats", nil)
		ctx := contextWithCentralAdmin(req.Context(), adminID, "central@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetPlatformStats(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var stats PlatformStats
		if err := json.Unmarshal(rec.Body.Bytes(), &stats); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify stats structure
		if stats.TotalTenants < 1 {
			t.Errorf("Expected at least 1 tenant, got %d", stats.TotalTenants)
		}

		if stats.GeneratedAt.IsZero() {
			t.Error("GeneratedAt should be set")
		}
	})
}

// TestCentralAdminHandler_ListTenants tests listing all tenants
func TestCentralAdminHandler_ListTenants(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewCentralAdminHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "central@example.com", "Central Admin", "green")
	db.Exec("UPDATE users SET is_central_admin = 1 WHERE id = ?", adminID)

	// Create additional test tenants (tenant with slug 'test-tenant' already exists from SetupTestDB)
	_, err := db.Exec(`INSERT INTO tenants (slug, name, contact_email, status, created_at) VALUES ('tenant-one', 'Tenant One', 'one@tenant.com', 'active', ?)`, time.Now())
	if err != nil {
		t.Fatalf("Failed to create tenant 1: %v", err)
	}
	_, err = db.Exec(`INSERT INTO tenants (slug, name, contact_email, status, created_at) VALUES ('tenant-two', 'Tenant Two', 'two@tenant.com', 'suspended', ?)`, time.Now())
	if err != nil {
		t.Fatalf("Failed to create tenant 2: %v", err)
	}

	t.Run("returns all tenants", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/central-admin/tenants", nil)
		ctx := contextWithCentralAdmin(req.Context(), adminID, "central@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListTenants(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var tenants []TenantListItem
		if err := json.Unmarshal(rec.Body.Bytes(), &tenants); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(tenants) < 3 {
			t.Errorf("Expected at least 3 tenants (test-tenant + 2 created), got %d", len(tenants))
		}
	})

	t.Run("filters by active only", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/central-admin/tenants?active_only=true", nil)
		ctx := contextWithCentralAdmin(req.Context(), adminID, "central@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListTenants(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var tenants []TenantListItem
		json.Unmarshal(rec.Body.Bytes(), &tenants)

		for _, tenant := range tenants {
			if tenant.Status != "active" {
				t.Errorf("Expected only active tenants, got status: %s", tenant.Status)
			}
		}
	})

	t.Run("searches by name", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/central-admin/tenants?search=One", nil)
		ctx := contextWithCentralAdmin(req.Context(), adminID, "central@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListTenants(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var tenants []TenantListItem
		json.Unmarshal(rec.Body.Bytes(), &tenants)

		found := false
		for _, tenant := range tenants {
			if tenant.Name == "Tenant One" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find 'Tenant One' in search results")
		}
	})
}

// TestCentralAdminHandler_GetTenant tests getting a specific tenant
func TestCentralAdminHandler_GetTenant(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewCentralAdminHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "central@example.com", "Central Admin", "green")
	db.Exec("UPDATE users SET is_central_admin = 1 WHERE id = ?", adminID)

	// Create test tenant
	result, _ := db.Exec(`INSERT INTO tenants (slug, name, contact_email, status, created_at) VALUES ('test-get', 'Test Get Tenant', 'get@tenant.com', 'active', ?)`, time.Now())
	tenantID, _ := result.LastInsertId()

	t.Run("returns tenant by ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/central-admin/tenants/"+strconv.FormatInt(tenantID, 10), nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.FormatInt(tenantID, 10)})
		ctx := contextWithCentralAdmin(req.Context(), adminID, "central@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetTenant(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["tenant"] == nil {
			t.Error("Expected tenant in response")
		}
	})

	t.Run("returns 404 for non-existent tenant", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/central-admin/tenants/99999", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "99999"})
		ctx := contextWithCentralAdmin(req.Context(), adminID, "central@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetTenant(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})

	t.Run("returns 400 for invalid tenant ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/central-admin/tenants/invalid", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
		ctx := contextWithCentralAdmin(req.Context(), adminID, "central@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetTenant(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})
}

// TestCentralAdminHandler_UpdateTenant tests updating tenant information
func TestCentralAdminHandler_UpdateTenant(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewCentralAdminHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "central@example.com", "Central Admin", "green")
	db.Exec("UPDATE users SET is_central_admin = 1 WHERE id = ?", adminID)

	// Create test tenant
	result, _ := db.Exec(`INSERT INTO tenants (slug, name, contact_email, status, created_at) VALUES ('test-update', 'Test Update', 'update@tenant.com', 'active', ?)`, time.Now())
	tenantID, _ := result.LastInsertId()

	t.Run("updates tenant name", func(t *testing.T) {
		newName := "Updated Tenant Name"
		body, _ := json.Marshal(map[string]string{"name": newName})

		req := httptest.NewRequest("PUT", "/api/central-admin/tenants/"+strconv.FormatInt(tenantID, 10), bytes.NewReader(body))
		req = mux.SetURLVars(req, map[string]string{"id": strconv.FormatInt(tenantID, 10)})
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithCentralAdmin(req.Context(), adminID, "central@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateTenant(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify in database
		var dbName string
		db.QueryRow("SELECT name FROM tenants WHERE id = ?", tenantID).Scan(&dbName)
		if dbName != newName {
			t.Errorf("Expected name to be %s, got %s", newName, dbName)
		}
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/api/central-admin/tenants/"+strconv.FormatInt(tenantID, 10), bytes.NewReader([]byte("invalid json")))
		req = mux.SetURLVars(req, map[string]string{"id": strconv.FormatInt(tenantID, 10)})
		req.Header.Set("Content-Type", "application/json")
		ctx := contextWithCentralAdmin(req.Context(), adminID, "central@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.UpdateTenant(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})
}

// TestCentralAdminHandler_ActivateTenant tests activating a suspended tenant
func TestCentralAdminHandler_ActivateTenant(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewCentralAdminHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "central@example.com", "Central Admin", "green")
	db.Exec("UPDATE users SET is_central_admin = 1 WHERE id = ?", adminID)

	// Create suspended tenant
	result, _ := db.Exec(`INSERT INTO tenants (slug, name, contact_email, status, created_at) VALUES ('suspended-tenant', 'Suspended', 'suspended@tenant.com', 'suspended', ?)`, time.Now())
	tenantID, _ := result.LastInsertId()

	t.Run("activates suspended tenant", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/central-admin/tenants/"+strconv.FormatInt(tenantID, 10)+"/activate", nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.FormatInt(tenantID, 10)})
		ctx := contextWithCentralAdmin(req.Context(), adminID, "central@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ActivateTenant(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify in database
		var status string
		db.QueryRow("SELECT status FROM tenants WHERE id = ?", tenantID).Scan(&status)
		if status != "active" {
			t.Errorf("Expected status to be 'active', got %s", status)
		}
	})

	t.Run("returns 404 for non-existent tenant", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/central-admin/tenants/99999/activate", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "99999"})
		ctx := contextWithCentralAdmin(req.Context(), adminID, "central@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ActivateTenant(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})
}

// TestCentralAdminHandler_DeactivateTenant tests deactivating (suspending) a tenant
func TestCentralAdminHandler_DeactivateTenant(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewCentralAdminHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "central@example.com", "Central Admin", "green")
	db.Exec("UPDATE users SET is_central_admin = 1 WHERE id = ?", adminID)

	// Create active tenant
	result, _ := db.Exec(`INSERT INTO tenants (slug, name, contact_email, status, created_at) VALUES ('active-tenant', 'Active', 'active@tenant.com', 'active', ?)`, time.Now())
	tenantID, _ := result.LastInsertId()

	t.Run("deactivates active tenant", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/central-admin/tenants/"+strconv.FormatInt(tenantID, 10)+"/deactivate", nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.FormatInt(tenantID, 10)})
		ctx := contextWithCentralAdmin(req.Context(), adminID, "central@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DeactivateTenant(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify in database
		var status string
		db.QueryRow("SELECT status FROM tenants WHERE id = ?", tenantID).Scan(&status)
		if status != "suspended" {
			t.Errorf("Expected status to be 'suspended', got %s", status)
		}
	})
}

// TestCentralAdminHandler_ListCentralAdmins tests listing all central admins
func TestCentralAdminHandler_ListCentralAdmins(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewCentralAdminHandler(db, cfg)

	// Create central admin users
	admin1ID := testutil.SeedTestUser(t, db, "central1@example.com", "Central One", "green")
	db.Exec("UPDATE users SET is_central_admin = 1 WHERE id = ?", admin1ID)

	admin2ID := testutil.SeedTestUser(t, db, "central2@example.com", "Central Two", "green")
	db.Exec("UPDATE users SET is_central_admin = 1 WHERE id = ?", admin2ID)

	// Create regular user (should not appear)
	testutil.SeedTestUser(t, db, "regular@example.com", "Regular User", "green")

	t.Run("returns only central admins", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/central-admin/admins", nil)
		ctx := contextWithCentralAdmin(req.Context(), admin1ID, "central1@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListCentralAdmins(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var admins []map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &admins)

		if len(admins) < 2 {
			t.Errorf("Expected at least 2 central admins, got %d", len(admins))
		}

		// Verify no regular users
		for _, admin := range admins {
			email := admin["email"].(string)
			if email == "regular@example.com" {
				t.Error("Regular user should not be in central admins list")
			}
		}
	})
}

// TestCentralAdminHandler_PromoteToCentralAdmin tests promoting a user to central admin
func TestCentralAdminHandler_PromoteToCentralAdmin(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewCentralAdminHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "central@example.com", "Central Admin", "green")
	db.Exec("UPDATE users SET is_central_admin = 1 WHERE id = ?", adminID)

	regularUserID := testutil.SeedTestUser(t, db, "regular@example.com", "Regular User", "green")

	t.Run("promotes regular user to central admin", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/central-admin/admins/"+strconv.Itoa(regularUserID)+"/promote", nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(regularUserID)})
		ctx := contextWithCentralAdmin(req.Context(), adminID, "central@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.PromoteToCentralAdmin(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify in database
		var isCentralAdmin bool
		db.QueryRow("SELECT is_central_admin FROM users WHERE id = ?", regularUserID).Scan(&isCentralAdmin)
		if !isCentralAdmin {
			t.Error("User should be promoted to central admin")
		}
	})

	t.Run("returns error if already central admin", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/central-admin/admins/"+strconv.Itoa(adminID)+"/promote", nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(adminID)})
		ctx := contextWithCentralAdmin(req.Context(), adminID, "central@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.PromoteToCentralAdmin(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	t.Run("returns 404 for non-existent user", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/central-admin/admins/99999/promote", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "99999"})
		ctx := contextWithCentralAdmin(req.Context(), adminID, "central@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.PromoteToCentralAdmin(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})
}

// TestCentralAdminHandler_DemoteFromCentralAdmin tests removing central admin privileges
func TestCentralAdminHandler_DemoteFromCentralAdmin(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewCentralAdminHandler(db, cfg)

	admin1ID := testutil.SeedTestUser(t, db, "central1@example.com", "Central One", "green")
	db.Exec("UPDATE users SET is_central_admin = 1 WHERE id = ?", admin1ID)

	admin2ID := testutil.SeedTestUser(t, db, "central2@example.com", "Central Two", "green")
	db.Exec("UPDATE users SET is_central_admin = 1 WHERE id = ?", admin2ID)

	t.Run("demotes central admin", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/central-admin/admins/"+strconv.Itoa(admin2ID)+"/demote", nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(admin2ID)})
		ctx := contextWithCentralAdmin(req.Context(), admin1ID, "central1@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DemoteFromCentralAdmin(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify in database
		var isCentralAdmin bool
		db.QueryRow("SELECT is_central_admin FROM users WHERE id = ?", admin2ID).Scan(&isCentralAdmin)
		if isCentralAdmin {
			t.Error("User should be demoted from central admin")
		}
	})

	t.Run("prevents self-demotion", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/central-admin/admins/"+strconv.Itoa(admin1ID)+"/demote", nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(admin1ID)})
		ctx := contextWithCentralAdmin(req.Context(), admin1ID, "central1@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DemoteFromCentralAdmin(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for self-demotion, got %d", rec.Code)
		}
	})

	t.Run("returns error if not central admin", func(t *testing.T) {
		regularUserID := testutil.SeedTestUser(t, db, "regular@example.com", "Regular", "green")

		req := httptest.NewRequest("POST", "/api/central-admin/admins/"+strconv.Itoa(regularUserID)+"/demote", nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.Itoa(regularUserID)})
		ctx := contextWithCentralAdmin(req.Context(), admin1ID, "central1@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.DemoteFromCentralAdmin(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})
}

// TestCentralAdminHandler_GetTenantUsers tests getting users for a specific tenant
func TestCentralAdminHandler_GetTenantUsers(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewCentralAdminHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "central@example.com", "Central Admin", "green")
	db.Exec("UPDATE users SET is_central_admin = 1 WHERE id = ?", adminID)

	// Create tenant and users
	result, _ := db.Exec(`INSERT INTO tenants (slug, name, contact_email, status, created_at) VALUES ('users-tenant', 'Users Tenant', 'users@tenant.com', 'active', ?)`, time.Now())
	tenantID, _ := result.LastInsertId()

	// Create users for this tenant
	user1ID := testutil.SeedTestUser(t, db, "user1@tenant.com", "User One", "green")
	db.Exec("UPDATE users SET tenant_id = ? WHERE id = ?", tenantID, user1ID)

	user2ID := testutil.SeedTestUser(t, db, "user2@tenant.com", "User Two", "green")
	db.Exec("UPDATE users SET tenant_id = ? WHERE id = ?", tenantID, user2ID)

	t.Run("returns users for tenant", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/central-admin/tenants/"+strconv.FormatInt(tenantID, 10)+"/users", nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.FormatInt(tenantID, 10)})
		ctx := contextWithCentralAdmin(req.Context(), adminID, "central@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetTenantUsers(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var users []map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &users)

		if len(users) < 2 {
			t.Errorf("Expected at least 2 users, got %d", len(users))
		}

		// Verify sensitive data is removed
		for _, user := range users {
			if user["password_hash"] != nil {
				t.Error("password_hash should be removed from response")
			}
		}
	})

	t.Run("returns 404 for non-existent tenant", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/central-admin/tenants/99999/users", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "99999"})
		ctx := contextWithCentralAdmin(req.Context(), adminID, "central@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.GetTenantUsers(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})
}

// TestCentralAdminHandler_SearchUsers tests searching users across all tenants
func TestCentralAdminHandler_SearchUsers(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewCentralAdminHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "central@example.com", "Central Admin", "green")
	db.Exec("UPDATE users SET is_central_admin = 1 WHERE id = ?", adminID)

	// Create users with specific names
	testutil.SeedTestUser(t, db, "john.doe@example.com", "John Doe", "green")
	testutil.SeedTestUser(t, db, "jane.doe@example.com", "Jane Doe", "green")
	testutil.SeedTestUser(t, db, "bob.smith@example.com", "Bob Smith", "green")

	t.Run("searches users by name", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/central-admin/users/search?q=Doe", nil)
		ctx := contextWithCentralAdmin(req.Context(), adminID, "central@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.SearchUsers(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var users []map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &users)

		// Should find both John and Jane Doe
		if len(users) < 2 {
			t.Errorf("Expected at least 2 users matching 'Doe', got %d", len(users))
		}
	})

	t.Run("returns 400 without search term", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/central-admin/users/search", nil)
		ctx := contextWithCentralAdmin(req.Context(), adminID, "central@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.SearchUsers(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})
}

// TestCentralAdminHandler_ExportTenantData tests GDPR-compliant data export
func TestCentralAdminHandler_ExportTenantData(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewCentralAdminHandler(db, cfg)

	adminID := testutil.SeedTestUser(t, db, "central@example.com", "Central Admin", "green")
	db.Exec("UPDATE users SET is_central_admin = 1 WHERE id = ?", adminID)

	// Create tenant with data
	result, _ := db.Exec(`INSERT INTO tenants (slug, name, contact_email, status, created_at) VALUES ('export-tenant', 'Export Tenant', 'export@tenant.com', 'active', ?)`, time.Now())
	tenantID, _ := result.LastInsertId()

	// Create users for this tenant
	userID := testutil.SeedTestUser(t, db, "export-user@example.com", "Export User", "green")
	db.Exec("UPDATE users SET tenant_id = ? WHERE id = ?", tenantID, userID)

	t.Run("exports tenant data", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/central-admin/tenants/"+strconv.FormatInt(tenantID, 10)+"/export", nil)
		req = mux.SetURLVars(req, map[string]string{"id": strconv.FormatInt(tenantID, 10)})
		ctx := contextWithCentralAdmin(req.Context(), adminID, "central@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ExportTenantData(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var export map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &export)

		// Verify export structure
		if export["tenant"] == nil {
			t.Error("Expected tenant in export")
		}
		if export["users"] == nil {
			t.Error("Expected users in export")
		}
		if export["exported_at"] == nil {
			t.Error("Expected exported_at timestamp")
		}
	})

	t.Run("returns 404 for non-existent tenant", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/central-admin/tenants/99999/export", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "99999"})
		ctx := contextWithCentralAdmin(req.Context(), adminID, "central@example.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ExportTenantData(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})
}
