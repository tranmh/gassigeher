package repository

import (
	"testing"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// TestDemoTenantRepository_CreateState tests creating demo state
func TestDemoTenantRepository_CreateState(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewDemoTenantRepository(db)

	t.Run("create valid state", func(t *testing.T) {
		now := time.Now()
		nextReset := now.Add(24 * time.Hour)
		state := &models.DemoTenantState{
			TenantID:      1,
			AdminPassword: "testpassword123",
			LastResetAt:   &now,
			NextResetAt:   &nextReset,
		}

		err := repo.CreateState(state)
		if err != nil {
			t.Fatalf("CreateState() failed: %v", err)
		}

		if state.ID == 0 {
			t.Error("Expected state ID to be set after creation")
		}
	})

	t.Run("create duplicate tenant fails", func(t *testing.T) {
		// Tenant 1 already has state from previous test
		now := time.Now()
		state := &models.DemoTenantState{
			TenantID:      1,
			AdminPassword: "anotherpassword",
			LastResetAt:   &now,
		}

		err := repo.CreateState(state)
		if err == nil {
			t.Error("Expected error when creating duplicate state for same tenant")
		}
	})
}

// TestDemoTenantRepository_GetState tests retrieving demo state
func TestDemoTenantRepository_GetState(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewDemoTenantRepository(db)

	t.Run("state not found returns nil", func(t *testing.T) {
		state, err := repo.GetState(999)
		if err != nil {
			t.Fatalf("GetState() failed: %v", err)
		}
		if state != nil {
			t.Error("Expected nil for non-existent state")
		}
	})

	t.Run("get existing state", func(t *testing.T) {
		// Create state first
		now := time.Now()
		nextReset := now.Add(24 * time.Hour)
		createState := &models.DemoTenantState{
			TenantID:      1,
			AdminPassword: "retrievepassword",
			LastResetAt:   &now,
			NextResetAt:   &nextReset,
		}
		err := repo.CreateState(createState)
		if err != nil {
			t.Fatalf("CreateState() failed: %v", err)
		}

		// Get state
		state, err := repo.GetState(1)
		if err != nil {
			t.Fatalf("GetState() failed: %v", err)
		}

		if state == nil {
			t.Fatal("Expected state to be found")
		}

		if state.AdminPassword != "retrievepassword" {
			t.Errorf("Expected password 'retrievepassword', got %s", state.AdminPassword)
		}

		if state.TenantID != 1 {
			t.Errorf("Expected tenant_id 1, got %d", state.TenantID)
		}
	})
}

// TestDemoTenantRepository_UpdateState tests updating demo state
func TestDemoTenantRepository_UpdateState(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewDemoTenantRepository(db)

	t.Run("update existing state", func(t *testing.T) {
		// Create state first
		now := time.Now()
		createState := &models.DemoTenantState{
			TenantID:      1,
			AdminPassword: "oldpassword",
			LastResetAt:   &now,
		}
		err := repo.CreateState(createState)
		if err != nil {
			t.Fatalf("CreateState() failed: %v", err)
		}

		// Update state
		newReset := now.Add(48 * time.Hour)
		err = repo.UpdateState(1, "newpassword", &now, &newReset)
		if err != nil {
			t.Fatalf("UpdateState() failed: %v", err)
		}

		// Verify update
		state, _ := repo.GetState(1)
		if state.AdminPassword != "newpassword" {
			t.Errorf("Expected password 'newpassword', got %s", state.AdminPassword)
		}
	})
}

// TestDemoTenantRepository_GetCredentials tests retrieving formatted credentials
func TestDemoTenantRepository_GetCredentials(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewDemoTenantRepository(db)

	t.Run("no credentials returns nil", func(t *testing.T) {
		creds, err := repo.GetCredentials(999)
		if err != nil {
			t.Fatalf("GetCredentials() failed: %v", err)
		}
		if creds != nil {
			t.Error("Expected nil for non-existent credentials")
		}
	})

	t.Run("get credentials with super admin", func(t *testing.T) {
		// Create demo state
		now := time.Now()
		nextReset := now.Add(24 * time.Hour)
		state := &models.DemoTenantState{
			TenantID:      1,
			AdminPassword: "demopassword",
			LastResetAt:   &now,
			NextResetAt:   &nextReset,
		}
		err := repo.CreateState(state)
		if err != nil {
			t.Fatalf("CreateState() failed: %v", err)
		}

		// Create super admin user for tenant 1
		nowStr := testutil.Now()
		_, err = db.Exec(`
			INSERT INTO users (tenant_id, email, first_name, last_name, password_hash, is_super_admin, is_verified, is_active, terms_accepted_at, last_activity_at, created_at)
			VALUES (1, 'admin@demo.test', 'Demo', 'Admin', 'hash', 1, 1, 1, ?, ?, ?)
		`, nowStr, nowStr, nowStr)
		if err != nil {
			t.Fatalf("Failed to create super admin: %v", err)
		}

		// Get credentials
		creds, err := repo.GetCredentials(1)
		if err != nil {
			t.Fatalf("GetCredentials() failed: %v", err)
		}

		if creds == nil {
			t.Fatal("Expected credentials to be found")
		}

		if creds.AdminEmail != "admin@demo.test" {
			t.Errorf("Expected admin email 'admin@demo.test', got %s", creds.AdminEmail)
		}

		if creds.AdminPassword != "demopassword" {
			t.Errorf("Expected admin password 'demopassword', got %s", creds.AdminPassword)
		}
	})
}

// TestDemoTenantRepository_DeleteState tests deleting demo state
func TestDemoTenantRepository_DeleteState(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewDemoTenantRepository(db)

	t.Run("delete existing state", func(t *testing.T) {
		// Create state
		now := time.Now()
		state := &models.DemoTenantState{
			TenantID:      1,
			AdminPassword: "deletepassword",
			LastResetAt:   &now,
		}
		err := repo.CreateState(state)
		if err != nil {
			t.Fatalf("CreateState() failed: %v", err)
		}

		// Delete state
		err = repo.DeleteState(1)
		if err != nil {
			t.Fatalf("DeleteState() failed: %v", err)
		}

		// Verify deletion
		deleted, _ := repo.GetState(1)
		if deleted != nil {
			t.Error("Expected state to be deleted")
		}
	})

	t.Run("delete non-existent state no error", func(t *testing.T) {
		err := repo.DeleteState(9999)
		if err != nil {
			t.Errorf("Expected no error when deleting non-existent state, got: %v", err)
		}
	})
}
