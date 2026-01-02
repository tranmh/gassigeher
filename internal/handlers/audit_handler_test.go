package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/services"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// TestAuditHandler_ListAuditLogs tests listing audit logs
func TestAuditHandler_ListAuditLogs(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewAuditHandler(db)
	auditService := services.NewAuditService(db)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")

	t.Run("returns empty list when no logs", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/audit-logs", nil)
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListAuditLogs(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["total"] == nil {
			t.Error("Expected total field in response")
		}
		if response["limit"] == nil {
			t.Error("Expected limit field in response")
		}
	})

	t.Run("returns audit logs with pagination", func(t *testing.T) {
		// Create some audit logs
		userID := adminID
		entityID := 1
		auditService.LogSimple(nil, 0, &userID, models.AuditActionUserLogin, models.EntityTypeUser, &entityID)
		auditService.LogSimple(nil, 0, &userID, models.AuditActionDogCreated, models.EntityTypeDog, &entityID)
		auditService.LogSimple(nil, 0, &userID, models.AuditActionBookingCreated, models.EntityTypeBooking, &entityID)

		// Wait for async logs
		testutil.WaitForAsync()

		req := httptest.NewRequest("GET", "/api/admin/audit-logs?limit=2", nil)
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListAuditLogs(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		logs := response["logs"].([]interface{})
		if len(logs) > 2 {
			t.Errorf("Expected max 2 logs, got %d", len(logs))
		}
	})

	t.Run("filters by action type", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/audit-logs?action=user_login", nil)
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListAuditLogs(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["logs"] != nil {
			logs := response["logs"].([]interface{})
			for _, log := range logs {
				logMap := log.(map[string]interface{})
				if logMap["action"] != "user_login" {
					t.Errorf("Expected action user_login, got %s", logMap["action"])
				}
			}
		}
	})

	t.Run("filters by entity type", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/audit-logs?entity_type=dog", nil)
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListAuditLogs(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["logs"] != nil {
			logs := response["logs"].([]interface{})
			for _, log := range logs {
				logMap := log.(map[string]interface{})
				if logMap["entity_type"] != "dog" {
					t.Errorf("Expected entity_type dog, got %s", logMap["entity_type"])
				}
			}
		}
	})

	t.Run("filters by date range", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/audit-logs?start_date=2025-01-01&end_date=2030-12-31", nil)
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListAuditLogs(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}
	})

	t.Run("handles invalid date format gracefully", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/audit-logs?start_date=invalid", nil)
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListAuditLogs(rec, req)

		// Should not error, just ignore invalid date
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200 (ignore invalid date), got %d", rec.Code)
		}
	})

	t.Run("limit cannot exceed 500", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/audit-logs?limit=1000", nil)
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListAuditLogs(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		// Should use default limit since 1000 > 500
		limit := int(response["limit"].(float64))
		if limit > 500 {
			t.Errorf("Expected limit <= 500, got %d", limit)
		}
	})

	t.Run("negative offset is ignored", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/audit-logs?offset=-10", nil)
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListAuditLogs(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		offset := int(response["offset"].(float64))
		if offset < 0 {
			t.Errorf("Expected offset >= 0, got %d", offset)
		}
	})
}

// TestAuditHandler_TenantIsolation tests that audit logs are tenant-isolated
func TestAuditHandler_TenantIsolation(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewAuditHandler(db)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")

	// Insert audit logs for different tenants directly
	now := testutil.Now()
	db.Exec("INSERT INTO audit_logs (tenant_id, user_id, action, entity_type, entity_id, created_at) VALUES (0, ?, 'user_login', 'user', 1, ?)", adminID, now)
	db.Exec("INSERT INTO audit_logs (tenant_id, user_id, action, entity_type, entity_id, created_at) VALUES (1, ?, 'user_login', 'user', 1, ?)", adminID, now)
	db.Exec("INSERT INTO audit_logs (tenant_id, user_id, action, entity_type, entity_id, created_at) VALUES (2, ?, 'user_login', 'user', 1, ?)", adminID, now)

	t.Run("only returns logs for current tenant", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/audit-logs", nil)
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListAuditLogs(rec, req)

		if rec.Code != http.StatusOK {
			t.Logf("Got status %d with body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["logs"] != nil {
			logs := response["logs"].([]interface{})
			for _, log := range logs {
				logMap := log.(map[string]interface{})
				tenantID := int(logMap["tenant_id"].(float64))
				if tenantID != 0 {
					t.Errorf("Expected tenant_id 0, got %d - tenant isolation bug!", tenantID)
				}
			}
		}
	})
}

// TestAuditHandler_GetAuditLogActions tests getting available actions
func TestAuditHandler_GetAuditLogActions(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewAuditHandler(db)

	t.Run("returns list of actions", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/audit-logs/actions", nil)

		rec := httptest.NewRecorder()
		handler.GetAuditLogActions(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var actions []map[string]string
		json.Unmarshal(rec.Body.Bytes(), &actions)

		if len(actions) == 0 {
			t.Error("Expected at least one action")
		}

		// Verify structure
		for _, action := range actions {
			if action["value"] == "" {
				t.Error("Expected action value to be non-empty")
			}
			if action["label"] == "" {
				t.Error("Expected action label to be non-empty")
			}
		}
	})

	t.Run("includes all documented action types", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/audit-logs/actions", nil)

		rec := httptest.NewRecorder()
		handler.GetAuditLogActions(rec, req)

		var actions []map[string]string
		json.Unmarshal(rec.Body.Bytes(), &actions)

		expectedActions := []string{
			models.AuditActionBookingCreated,
			models.AuditActionBookingCancelled,
			models.AuditActionUserCreated,
			models.AuditActionUserLogin,
			models.AuditActionDogCreated,
		}

		actionValues := make(map[string]bool)
		for _, action := range actions {
			actionValues[action["value"]] = true
		}

		for _, expected := range expectedActions {
			if !actionValues[expected] {
				t.Errorf("Missing action type: %s", expected)
			}
		}
	})
}

// TestAuditHandler_GetAuditLogEntityTypes tests getting available entity types
func TestAuditHandler_GetAuditLogEntityTypes(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewAuditHandler(db)

	t.Run("returns list of entity types", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/audit-logs/entity-types", nil)

		rec := httptest.NewRecorder()
		handler.GetAuditLogEntityTypes(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var entityTypes []map[string]string
		json.Unmarshal(rec.Body.Bytes(), &entityTypes)

		if len(entityTypes) == 0 {
			t.Error("Expected at least one entity type")
		}

		// Verify structure
		for _, et := range entityTypes {
			if et["value"] == "" {
				t.Error("Expected entity type value to be non-empty")
			}
			if et["label"] == "" {
				t.Error("Expected entity type label to be non-empty")
			}
		}
	})

	t.Run("includes all documented entity types", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/audit-logs/entity-types", nil)

		rec := httptest.NewRecorder()
		handler.GetAuditLogEntityTypes(rec, req)

		var entityTypes []map[string]string
		json.Unmarshal(rec.Body.Bytes(), &entityTypes)

		expectedTypes := []string{
			models.EntityTypeBooking,
			models.EntityTypeUser,
			models.EntityTypeDog,
			models.EntityTypeSettings,
		}

		typeValues := make(map[string]bool)
		for _, et := range entityTypes {
			typeValues[et["value"]] = true
		}

		for _, expected := range expectedTypes {
			if !typeValues[expected] {
				t.Errorf("Missing entity type: %s", expected)
			}
		}
	})
}

// TestAuditHandler_SQLInjection tests for SQL injection vulnerabilities
func TestAuditHandler_SQLInjection(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewAuditHandler(db)

	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")

	t.Run("action filter is safe", func(t *testing.T) {
		// Use URL-encoded payload to avoid breaking HTTP parsing
		req := httptest.NewRequest("GET", "/api/admin/audit-logs?action=%27%3B+DROP+TABLE+audit_logs%3B+--", nil)
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListAuditLogs(rec, req)

		// Should not crash and table should still exist
		if rec.Code != http.StatusOK {
			t.Logf("Got status %d (expected for malformed input)", rec.Code)
		}

		// Verify table still exists
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM audit_logs").Scan(&count)
		if err != nil {
			t.Errorf("Table should still exist: %v", err)
		}
	})

	t.Run("entity_type filter is safe", func(t *testing.T) {
		// URL-encoded: user' OR '1'='1
		req := httptest.NewRequest("GET", "/api/admin/audit-logs?entity_type=user%27+OR+%271%27%3D%271", nil)
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListAuditLogs(rec, req)

		if rec.Code != http.StatusOK {
			t.Logf("Got status %d (expected for malformed input)", rec.Code)
		}
	})

	t.Run("entity_id filter only accepts integers", func(t *testing.T) {
		// URL-encoded: 1;DELETE FROM users;
		req := httptest.NewRequest("GET", "/api/admin/audit-logs?entity_id=1%3BDELETE+FROM+users%3B", nil)
		ctx := contextWithUser(req.Context(), adminID, "admin@example.com", true)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ListAuditLogs(rec, req)

		// Should not error - invalid integer is ignored
		if rec.Code != http.StatusOK {
			t.Logf("Got status %d", rec.Code)
		}

		// Verify users table still exists
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
		if err != nil {
			t.Errorf("Users table should still exist: %v", err)
		}
	})
}

// TestAuditService_LogsOldAndNewValues tests that audit service correctly logs changes
func TestAuditService_LogsOldAndNewValues(t *testing.T) {
	db := testutil.SetupTestDB(t)
	auditService := services.NewAuditService(db)

	// Create a test user for the foreign key constraint
	now := testutil.Now()
	result, err := db.Exec("INSERT INTO users (tenant_id, email, password_hash, first_name, last_name, terms_accepted_at, created_at, updated_at) VALUES (0, 'audit-test@example.com', 'hash', 'Audit', 'Tester', ?, ?, ?)", now, now, now)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	userIDInt64, _ := result.LastInsertId()
	userID := int(userIDInt64)
	entityID := 42

	t.Run("logs with old and new values", func(t *testing.T) {
		oldValue := map[string]string{"name": "Old Name"}
		newValue := map[string]string{"name": "New Name"}

		auditService.Log(nil, 0, &userID, models.AuditActionUserUpdated, models.EntityTypeUser, &entityID, oldValue, newValue)

		// Wait for async log
		testutil.WaitForAsync()

		// Verify the log was created
		var oldJSON, newJSON *string
		err := db.QueryRow("SELECT old_value, new_value FROM audit_logs WHERE action = ? AND entity_id = ?", models.AuditActionUserUpdated, entityID).Scan(&oldJSON, &newJSON)
		if err != nil {
			t.Fatalf("Failed to query audit log: %v", err)
		}

		if oldJSON == nil {
			t.Error("Expected old_value to be set")
		}
		if newJSON == nil {
			t.Error("Expected new_value to be set")
		}
	})

	t.Run("logs with message helper", func(t *testing.T) {
		messageEntityID := 99
		auditService.LogWithMessage(nil, 0, &userID, models.AuditActionSettingsChanged, models.EntityTypeSettings, &messageEntityID, "Changed auto-deactivation days")

		testutil.WaitForAsync()

		var newJSON *string
		err := db.QueryRow("SELECT new_value FROM audit_logs WHERE action = ? AND entity_id = ?", models.AuditActionSettingsChanged, messageEntityID).Scan(&newJSON)
		if err != nil {
			t.Fatalf("Failed to query audit log: %v", err)
		}

		if newJSON == nil || *newJSON == "" {
			t.Error("Expected new_value to contain message")
		}
	})
}
