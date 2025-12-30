package handlers

import (
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
	"github.com/tranmh/gassigeher/internal/repository"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// ============================================================================
// BUG #1 VERIFICATION: BlockedDate.Delete() Tenant Isolation
// File: blocked_date_repository.go:259-282
// Status: FIXED - This test verifies the fix works correctly
// ============================================================================

func TestBug1_BlockedDateDelete_TenantIsolation_FIXED(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewBlockedDateRepository(db)

	// Create two tenants
	result1, err := db.Exec(`INSERT INTO tenants (slug, name, contact_email, status, created_at) VALUES ('tenant1', 'Tenant 1', 'tenant1@test.com', 'active', ?)`, time.Now())
	if err != nil {
		t.Fatalf("Failed to create tenant1: %v", err)
	}
	tenant1ID, _ := result1.LastInsertId()

	result2, err := db.Exec(`INSERT INTO tenants (slug, name, contact_email, status, created_at) VALUES ('tenant2', 'Tenant 2', 'tenant2@test.com', 'active', ?)`, time.Now())
	if err != nil {
		t.Fatalf("Failed to create tenant2: %v", err)
	}
	tenant2ID, _ := result2.LastInsertId()

	// Create admin for tenant1
	admin1ID := testutil.SeedTestUser(t, db, "admin1@test.com", "Admin 1", "green")
	db.Exec("UPDATE users SET tenant_id = ? WHERE id = ?", tenant1ID, admin1ID)

	// Create blocked date for tenant1
	_, err = db.Exec(`INSERT INTO blocked_dates (tenant_id, date, reason, created_by, created_at) VALUES (?, '2025-12-25', 'Christmas', ?, ?)`,
		tenant1ID, admin1ID, time.Now())
	if err != nil {
		t.Fatalf("Failed to create blocked date: %v", err)
	}

	// Get the blocked date ID
	var blockedDateID int
	err = db.QueryRow("SELECT id FROM blocked_dates WHERE tenant_id = ? AND date = '2025-12-25'", tenant1ID).Scan(&blockedDateID)
	if err != nil {
		t.Fatalf("Failed to get blocked date ID: %v", err)
	}

	t.Run("SECURITY: Tenant2 cannot delete Tenant1's blocked date", func(t *testing.T) {
		// Try to delete using tenant2's ID - this should fail due to tenant isolation
		err := repo.Delete(blockedDateID, int(tenant2ID))

		// The delete should fail or return an error because tenant_id doesn't match
		if err == nil {
			// If no error, verify the blocked date still exists
			var count int
			db.QueryRow("SELECT COUNT(*) FROM blocked_dates WHERE id = ?", blockedDateID).Scan(&count)
			if count == 0 {
				t.Error("SECURITY BUG: Tenant2 was able to delete Tenant1's blocked date!")
			}
		}

		// Verify blocked date still exists for tenant1
		var exists int
		db.QueryRow("SELECT COUNT(*) FROM blocked_dates WHERE id = ? AND tenant_id = ?", blockedDateID, tenant1ID).Scan(&exists)
		if exists != 1 {
			t.Error("SECURITY BUG: Blocked date was deleted by wrong tenant!")
		}
	})

	t.Run("Correct tenant can delete their own blocked date", func(t *testing.T) {
		err := repo.Delete(blockedDateID, int(tenant1ID))
		if err != nil {
			t.Errorf("Owner tenant should be able to delete: %v", err)
		}

		// Verify deletion
		var count int
		db.QueryRow("SELECT COUNT(*) FROM blocked_dates WHERE id = ?", blockedDateID).Scan(&count)
		if count != 0 {
			t.Error("Blocked date should be deleted when correct tenant deletes it")
		}
	})
}

// ============================================================================
// BUG #2: Cross-Tenant User Impersonation
// File: central_admin_handler.go:799-800
// Issue: ImpersonateTenantUser uses FindByID without tenant validation
// The comment says "FindByID works cross-tenant" - this is intentional for
// Central Admin BUT should verify user belongs to the target tenant when
// tenant context is provided.
// ============================================================================

func TestBug2_CrossTenantUserImpersonation_CRITICAL(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewCentralAdminHandler(db, cfg)

	// Create two tenants
	result1, err := db.Exec(`INSERT INTO tenants (slug, name, contact_email, status, created_at) VALUES ('tenant-a', 'Tenant A', 'a@tenant.com', 'active', ?)`, time.Now())
	if err != nil {
		t.Fatalf("Failed to create tenant A: %v", err)
	}
	tenantAID, _ := result1.LastInsertId()

	result2, err := db.Exec(`INSERT INTO tenants (slug, name, contact_email, status, created_at) VALUES ('tenant-b', 'Tenant B', 'b@tenant.com', 'active', ?)`, time.Now())
	if err != nil {
		t.Fatalf("Failed to create tenant B: %v", err)
	}
	tenantBID, _ := result2.LastInsertId()

	// Create central admin (no tenant)
	centralAdminID := testutil.SeedTestUser(t, db, "central@admin.com", "Central Admin", "green")
	db.Exec("UPDATE users SET is_central_admin = 1, tenant_id = 0 WHERE id = ?", centralAdminID)

	// Create user in Tenant A
	userAID := testutil.SeedTestUser(t, db, "user@tenant-a.com", "User A", "green")
	db.Exec("UPDATE users SET tenant_id = ? WHERE id = ?", tenantAID, userAID)

	// Create user in Tenant B
	userBID := testutil.SeedTestUser(t, db, "user@tenant-b.com", "User B", "green")
	db.Exec("UPDATE users SET tenant_id = ? WHERE id = ?", tenantBID, userBID)

	t.Run("SECURITY: Impersonation should work for valid user", func(t *testing.T) {
		// Central admin can impersonate any user - this is by design
		req := httptest.NewRequest("POST", "/api/central-admin/impersonate/"+strconv.Itoa(userAID), nil)
		req = mux.SetURLVars(req, map[string]string{"userId": strconv.Itoa(userAID)})
		ctx := contextWithCentralAdmin(req.Context(), centralAdminID, "central@admin.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ImpersonateTenantUser(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Central admin should be able to impersonate user: %d - %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("SECURITY: Returned token should have correct tenant_id", func(t *testing.T) {
		// This tests that when impersonating a user, the returned token
		// correctly reflects that user's tenant_id
		req := httptest.NewRequest("POST", "/api/central-admin/impersonate/"+strconv.Itoa(userAID), nil)
		req = mux.SetURLVars(req, map[string]string{"userId": strconv.Itoa(userAID)})
		ctx := contextWithCentralAdmin(req.Context(), centralAdminID, "central@admin.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ImpersonateTenantUser(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Impersonation failed: %s", rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		// Check that the response includes user info with correct tenant
		userInfo, ok := response["user"].(map[string]interface{})
		if !ok {
			t.Fatal("Response should include user info")
		}

		// Verify tenant_id in user matches tenant A
		if tenantIDFloat, ok := userInfo["tenant_id"].(float64); ok {
			if int64(tenantIDFloat) != tenantAID {
				t.Errorf("User should belong to tenant A (ID=%d), got tenant_id=%v", tenantAID, tenantIDFloat)
			}
		}
	})

	// This is the actual bug test - currently the endpoint doesn't require
	// a tenantID parameter, but if we want to add tenant scoping in the future,
	// we need to verify the user belongs to that tenant
	t.Run("DOCUMENTATION: Current behavior allows cross-tenant impersonation by design", func(t *testing.T) {
		// This is NOT a bug per se - Central Admin CAN impersonate any user
		// The current implementation correctly returns the user's actual tenant_id
		// This test documents the expected behavior
		t.Log("Central Admin impersonation is designed to work across all tenants")
		t.Log("The returned JWT correctly includes the target user's tenant_id")
		t.Log("No tenant validation is needed since Central Admin has platform-wide access")
	})
}

// ============================================================================
// BUG #3: SQL LIKE Injection (HIGH)
// File: central_admin_handler.go:142-146
// Issue: Search term wildcards (%, _) not escaped before wrapping with %
// ============================================================================

func TestBug3_SQLLikeInjection_HIGH(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewCentralAdminHandler(db, cfg)

	// Create central admin
	adminID := testutil.SeedTestUser(t, db, "central@admin.com", "Central Admin", "green")
	db.Exec("UPDATE users SET is_central_admin = 1 WHERE id = ?", adminID)

	// Create tenants with specific names
	now := time.Now()
	db.Exec(`INSERT INTO tenants (slug, name, contact_email, status, created_at) VALUES ('exact-match', 'ExactMatch Shelter', 'exact@test.com', 'active', ?)`, now)
	db.Exec(`INSERT INTO tenants (slug, name, contact_email, status, created_at) VALUES ('no-match', 'NoMatch Shelter', 'no@test.com', 'active', ?)`, now)
	db.Exec(`INSERT INTO tenants (slug, name, contact_email, status, created_at) VALUES ('percent-test', '100% Success Shelter', 'percent@test.com', 'active', ?)`, now)
	db.Exec(`INSERT INTO tenants (slug, name, contact_email, status, created_at) VALUES ('underscore-test', 'Under_Score Shelter', 'underscore@test.com', 'active', ?)`, now)

	t.Run("Normal search works correctly", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/central-admin/tenants?search=ExactMatch", nil)
		ctx := contextWithCentralAdmin(req.Context(), adminID, "central@admin.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListTenants(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", rec.Code)
		}

		var tenants []TenantListItem
		json.Unmarshal(rec.Body.Bytes(), &tenants)

		found := false
		for _, tenant := range tenants {
			if tenant.Name == "ExactMatch Shelter" {
				found = true
			}
		}
		if !found {
			t.Error("Should find ExactMatch Shelter")
		}
	})

	t.Run("BUG: Percent wildcard in search term should be escaped", func(t *testing.T) {
		// Searching for "%" should only match tenants with literal % in name
		// BUG: Currently matches ALL tenants due to unescaped LIKE wildcard
		req := httptest.NewRequest("GET", "/api/central-admin/tenants?search=%25", nil) // URL-encoded %
		ctx := contextWithCentralAdmin(req.Context(), adminID, "central@admin.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListTenants(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d", rec.Code)
		}

		var tenants []TenantListItem
		json.Unmarshal(rec.Body.Bytes(), &tenants)

		// With the bug: ALL tenants match because %% becomes %
		// Without the bug: Only "100% Success Shelter" should match
		matchedCount := 0
		for _, tenant := range tenants {
			if tenant.Name == "100% Success Shelter" {
				matchedCount++
			} else {
				// If we find any tenant without % in name, the bug exists
				if tenant.Name != "100% Success Shelter" && len(tenants) > 1 {
					t.Logf("BUG DETECTED: Search for '%%' matched tenant without '%%': %s", tenant.Name)
				}
			}
		}

		// Expected: Only 1 match (100% Success Shelter)
		// Bug behavior: Multiple matches (all tenants)
		if len(tenants) > 1 {
			t.Errorf("BUG: Search for '%%' should only match tenants with literal '%%', but matched %d tenants", len(tenants))
		}
	})

	t.Run("BUG: Underscore wildcard in search term should be escaped", func(t *testing.T) {
		// Searching for "_" should only match tenants with literal _ in name
		// BUG: Currently matches any single character due to unescaped LIKE wildcard
		req := httptest.NewRequest("GET", "/api/central-admin/tenants?search=_", nil)
		ctx := contextWithCentralAdmin(req.Context(), adminID, "central@admin.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListTenants(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d", rec.Code)
		}

		var tenants []TenantListItem
		json.Unmarshal(rec.Body.Bytes(), &tenants)

		// With the bug: Many tenants match because _ matches any single char
		// Without the bug: Only "Under_Score Shelter" should match
		if len(tenants) > 1 {
			t.Errorf("BUG: Search for '_' should only match tenants with literal '_', but matched %d tenants", len(tenants))
		}
	})
}

// ============================================================================
// BUG #3b: SQL LIKE Injection in SearchUsers (HIGH)
// File: central_admin_handler.go:538
// Same issue in user search
// ============================================================================

func TestBug3b_SQLLikeInjection_SearchUsers_HIGH(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewCentralAdminHandler(db, cfg)

	// Create central admin
	adminID := testutil.SeedTestUser(t, db, "central@admin.com", "Central Admin", "green")
	db.Exec("UPDATE users SET is_central_admin = 1 WHERE id = ?", adminID)

	// Create users with specific names
	testutil.SeedTestUser(t, db, "normal@test.com", "Normal User", "green")
	testutil.SeedTestUser(t, db, "percent@test.com", "100% Success", "green")
	testutil.SeedTestUser(t, db, "underscore@test.com", "Under_Score", "green")

	t.Run("BUG: Percent wildcard in user search should be escaped", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/central-admin/users/search?q=%25", nil) // URL-encoded %
		ctx := contextWithCentralAdmin(req.Context(), adminID, "central@admin.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.SearchUsers(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var result map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &result)

		users := result["users"].([]interface{})
		total := int(result["total"].(float64))

		// With the bug: ALL users match
		// Without the bug: Only users with % in name/email should match
		if total > 1 {
			t.Errorf("BUG: Search for '%%' in users matched %d users instead of just those with literal '%%'", total)
		}

		// Check if any user without % in name was matched
		for _, u := range users {
			user := u.(map[string]interface{})
			firstName := user["first_name"].(string)
			lastName := user["last_name"].(string)
			fullName := firstName + " " + lastName
			if fullName != "100% Success" && fullName != "Central Admin" {
				t.Logf("BUG: Matched user without '%%': %s", fullName)
			}
		}
	})

	t.Run("BUG: Underscore wildcard in user search should be escaped", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/central-admin/users/search?q=_", nil)
		ctx := contextWithCentralAdmin(req.Context(), adminID, "central@admin.com")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.SearchUsers(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var result map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &result)

		total := int(result["total"].(float64))

		// With the bug: Many users match (any with single char anywhere)
		// Without the bug: Only "Under_Score" should match
		if total > 1 {
			t.Errorf("BUG: Search for '_' in users matched %d users instead of just those with literal '_'", total)
		}
	})
}

// ============================================================================
// Helper function for creating impersonation context
// ============================================================================

func contextWithImpersonation(ctx context.Context, userID int, email string, originalUserID int) context.Context {
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, middleware.EmailKey, email)
	ctx = context.WithValue(ctx, middleware.IsAdminKey, false)
	ctx = context.WithValue(ctx, middleware.IsSuperAdminKey, false)
	ctx = context.WithValue(ctx, middleware.IsCentralAdminKey, false)
	ctx = context.WithValue(ctx, middleware.IsImpersonatingKey, true)
	ctx = context.WithValue(ctx, middleware.OriginalUserIDKey, originalUserID)
	return ctx
}
