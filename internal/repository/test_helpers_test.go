package repository

import (
	"database/sql"
	"sync"
	"testing"
	"time"
)

var (
	testCounter   = 0
	testCounterMu sync.Mutex
)

// createTestTenant creates a tenant for testing and returns its ID
func createTestTenant(t *testing.T, db *sql.DB) int {
	t.Helper()
	testCounterMu.Lock()
	testCounter++
	counter := testCounter
	testCounterMu.Unlock()

	slug := "test-tenant-" + time.Now().Format("150405") + "-" + string(rune('A'+counter%26)) + string(rune('0'+counter/26%10))
	now := time.Now()
	result, err := db.Exec(`INSERT INTO tenants (slug, name, status, contact_email, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		slug, "Test Tenant", "active", "test@test.com", now, now)
	if err != nil {
		t.Fatalf("Failed to create test tenant: %v", err)
	}
	id, _ := result.LastInsertId()
	return int(id)
}

// createTestUser creates a user for testing and returns its ID
func createTestUser(t *testing.T, db *sql.DB, tenantID int) int {
	t.Helper()
	testCounterMu.Lock()
	testCounter++
	counter := testCounter
	testCounterMu.Unlock()

	now := time.Now()
	email := "testuser" + time.Now().Format("150405") + "-" + string(rune('a'+counter%26)) + "@test.com"
	result, err := db.Exec(`INSERT INTO users (tenant_id, first_name, last_name, email, password_hash, is_active, is_verified, terms_accepted_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		tenantID, "Test", "User", email, "hash", 1, 1, now, now, now)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	id, _ := result.LastInsertId()
	return int(id)
}
