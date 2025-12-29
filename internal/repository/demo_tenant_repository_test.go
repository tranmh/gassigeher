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

	// Create test tenant with ID=10 for demo state (tenant 0 and 1 are already created by SetupTestDB)
	now := time.Now()
	_, err := db.Exec(`INSERT INTO tenants (id, slug, name, status, contact_email, created_at, updated_at) VALUES (10, 'demo', 'Demo', 'active', 'demo@test.com', ?, ?)`, now, now)
	if err != nil {
		t.Fatalf("Failed to create test tenant: %v", err)
	}

	t.Run("create valid state", func(t *testing.T) {
		nextReset := now.Add(24 * time.Hour)
		state := &models.DemoTenantState{
			TenantID:      10,
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
		// Tenant 10 already has state from previous test
		state := &models.DemoTenantState{
			TenantID:      10,
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

	// Create test tenant with ID=10 for demo state (tenant 0 and 1 are already created by SetupTestDB)
	now := time.Now()
	_, err := db.Exec(`INSERT INTO tenants (id, slug, name, status, contact_email, created_at, updated_at) VALUES (10, 'demo', 'Demo', 'active', 'demo@test.com', ?, ?)`, now, now)
	if err != nil {
		t.Fatalf("Failed to create test tenant: %v", err)
	}

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
		nextReset := now.Add(24 * time.Hour)
		createState := &models.DemoTenantState{
			TenantID:      10,
			AdminPassword: "retrievepassword",
			LastResetAt:   &now,
			NextResetAt:   &nextReset,
		}
		err := repo.CreateState(createState)
		if err != nil {
			t.Fatalf("CreateState() failed: %v", err)
		}

		// Get state
		state, err := repo.GetState(10)
		if err != nil {
			t.Fatalf("GetState() failed: %v", err)
		}

		if state == nil {
			t.Fatal("Expected state to be found")
		}

		if state.AdminPassword != "retrievepassword" {
			t.Errorf("Expected password 'retrievepassword', got %s", state.AdminPassword)
		}

		if state.TenantID != 10 {
			t.Errorf("Expected tenant_id 10, got %d", state.TenantID)
		}
	})
}

// TestDemoTenantRepository_UpdateState tests updating demo state
func TestDemoTenantRepository_UpdateState(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewDemoTenantRepository(db)

	// Create test tenant with ID=10 for demo state (tenant 0 and 1 are already created by SetupTestDB)
	now := time.Now()
	_, err := db.Exec(`INSERT INTO tenants (id, slug, name, status, contact_email, created_at, updated_at) VALUES (10, 'demo', 'Demo', 'active', 'demo@test.com', ?, ?)`, now, now)
	if err != nil {
		t.Fatalf("Failed to create test tenant: %v", err)
	}

	t.Run("update existing state", func(t *testing.T) {
		// Create state first
		createState := &models.DemoTenantState{
			TenantID:      10,
			AdminPassword: "oldpassword",
			LastResetAt:   &now,
		}
		err := repo.CreateState(createState)
		if err != nil {
			t.Fatalf("CreateState() failed: %v", err)
		}

		// Update state
		newReset := now.Add(48 * time.Hour)
		err = repo.UpdateState(10, "newpassword", &now, &newReset)
		if err != nil {
			t.Fatalf("UpdateState() failed: %v", err)
		}

		// Verify update
		state, _ := repo.GetState(10)
		if state.AdminPassword != "newpassword" {
			t.Errorf("Expected password 'newpassword', got %s", state.AdminPassword)
		}
	})
}

// TestDemoTenantRepository_GetCredentials tests retrieving formatted credentials
func TestDemoTenantRepository_GetCredentials(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewDemoTenantRepository(db)

	// Create test tenant with ID=10 for demo state (tenant 0 and 1 are already created by SetupTestDB)
	now := time.Now()
	_, err := db.Exec(`INSERT INTO tenants (id, slug, name, status, contact_email, created_at, updated_at) VALUES (10, 'demo', 'Demo', 'active', 'demo@test.com', ?, ?)`, now, now)
	if err != nil {
		t.Fatalf("Failed to create test tenant: %v", err)
	}

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
		nextReset := now.Add(24 * time.Hour)
		state := &models.DemoTenantState{
			TenantID:      10,
			AdminPassword: "demopassword",
			LastResetAt:   &now,
			NextResetAt:   &nextReset,
		}
		err := repo.CreateState(state)
		if err != nil {
			t.Fatalf("CreateState() failed: %v", err)
		}

		// Create super admin user for tenant 10
		nowStr := testutil.Now()
		_, err = db.Exec(`
			INSERT INTO users (tenant_id, email, first_name, last_name, password_hash, is_super_admin, is_verified, is_active, terms_accepted_at, last_activity_at, created_at)
			VALUES (10, 'admin@demo.test', 'Demo', 'Admin', 'hash', 1, 1, 1, ?, ?, ?)
		`, nowStr, nowStr, nowStr)
		if err != nil {
			t.Fatalf("Failed to create super admin: %v", err)
		}

		// Get credentials
		creds, err := repo.GetCredentials(10)
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

	// Create test tenant with ID=10 for demo state (tenant 0 and 1 are already created by SetupTestDB)
	now := time.Now()
	_, err := db.Exec(`INSERT INTO tenants (id, slug, name, status, contact_email, created_at, updated_at) VALUES (10, 'demo', 'Demo', 'active', 'demo@test.com', ?, ?)`, now, now)
	if err != nil {
		t.Fatalf("Failed to create test tenant: %v", err)
	}

	t.Run("delete existing state", func(t *testing.T) {
		// Create state
		state := &models.DemoTenantState{
			TenantID:      10,
			AdminPassword: "deletepassword",
			LastResetAt:   &now,
		}
		err := repo.CreateState(state)
		if err != nil {
			t.Fatalf("CreateState() failed: %v", err)
		}

		// Delete state
		err = repo.DeleteState(10)
		if err != nil {
			t.Fatalf("DeleteState() failed: %v", err)
		}

		// Verify deletion
		deleted, _ := repo.GetState(10)
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
