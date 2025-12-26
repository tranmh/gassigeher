package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/repository"
)

func setupExportTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Create all required tables
	_, err = db.Exec(`
		CREATE TABLE tenants (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			slug TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			status TEXT DEFAULT 'active',
			contact_email TEXT,
			contact_phone TEXT,
			address TEXT,
			city TEXT,
			postal_code TEXT,
			federal_state TEXT,
			is_demo INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			suspended_at TIMESTAMP,
			suspended_reason TEXT,
			deleted_at TIMESTAMP
		);

		CREATE TABLE tenant_settings (
			tenant_id INTEGER PRIMARY KEY,
			theme_preset TEXT DEFAULT 'classic',
			primary_color TEXT,
			secondary_color TEXT,
			accent_color TEXT,
			logo_url TEXT,
			favicon_url TEXT,
			tagline TEXT,
			description TEXT,
			welcome_message TEXT,
			footer_text TEXT,
			website_url TEXT,
			donation_url TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tenant_id INTEGER,
			email TEXT NOT NULL,
			first_name TEXT,
			last_name TEXT,
			phone TEXT,
			password_hash TEXT,
			is_admin INTEGER DEFAULT 0,
			is_super_admin INTEGER DEFAULT 0,
			is_central_admin INTEGER DEFAULT 0,
			is_verified INTEGER DEFAULT 1,
			is_active INTEGER DEFAULT 1,
			is_deleted INTEGER DEFAULT 0,
			profile_photo TEXT,
			terms_accepted_at TIMESTAMP,
			last_activity_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE dogs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tenant_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			breed TEXT NOT NULL,
			size TEXT,
			age INTEGER,
			color_id INTEGER,
			photo TEXT,
			photo_thumbnail TEXT,
			special_needs TEXT,
			pickup_location TEXT,
			walk_route TEXT,
			walk_duration INTEGER,
			special_instructions TEXT,
			default_morning_time TEXT,
			default_evening_time TEXT,
			is_available INTEGER DEFAULT 1,
			is_featured INTEGER DEFAULT 0,
			unavailable_reason TEXT,
			unavailable_since TIMESTAMP,
			external_link TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE bookings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tenant_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			dog_id INTEGER NOT NULL,
			date TEXT NOT NULL,
			walk_type TEXT NOT NULL,
			status TEXT DEFAULT 'scheduled',
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE blocked_dates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tenant_id INTEGER NOT NULL,
			date TEXT NOT NULL,
			reason TEXT,
			dog_id INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	return db
}

// BUG #4: Test that tenant export includes dogs correctly
func TestTenantHandler_ExportTenantData_IncludesDogs(t *testing.T) {
	db := setupExportTestDB(t)
	defer db.Close()

	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewTenantHandler(db, cfg)

	// Create a tenant
	result, err := db.Exec(`INSERT INTO tenants (slug, name, contact_email, status, federal_state, created_at, updated_at) VALUES ('test-tenant', 'Test Tenant', 'test@tenant.com', 'active', 'BW', ?, ?)`, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	tenantID, _ := result.LastInsertId()

	// Create tenant settings
	_, err = db.Exec(`INSERT INTO tenant_settings (tenant_id) VALUES (?)`, tenantID)
	if err != nil {
		t.Fatalf("Failed to create tenant settings: %v", err)
	}

	// Create an admin user for this tenant
	result, err = db.Exec(`INSERT INTO users (tenant_id, email, first_name, last_name, is_admin, is_verified, terms_accepted_at) VALUES (?, 'admin@test.com', 'Admin', 'User', 1, 1, ?)`, tenantID, time.Now())
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	userID, _ := result.LastInsertId()

	// Create dogs for this tenant
	_, err = db.Exec(`INSERT INTO dogs (tenant_id, name, breed, size, age, is_available) VALUES (?, 'Bella', 'Labrador', 'large', 3, 1)`, tenantID)
	if err != nil {
		t.Fatalf("Failed to create dog 1: %v", err)
	}
	_, err = db.Exec(`INSERT INTO dogs (tenant_id, name, breed, size, age, is_available) VALUES (?, 'Max', 'Golden Retriever', 'large', 5, 1)`, tenantID)
	if err != nil {
		t.Fatalf("Failed to create dog 2: %v", err)
	}
	_, err = db.Exec(`INSERT INTO dogs (tenant_id, name, breed, size, age, is_available) VALUES (?, 'Luna', 'Border Collie', 'medium', 2, 1)`, tenantID)
	if err != nil {
		t.Fatalf("Failed to create dog 3: %v", err)
	}

	// Verify tenant exists in DB
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM tenants WHERE id = ?", tenantID).Scan(&count); err != nil {
		t.Fatalf("Failed to verify tenant: %v", err)
	}
	if count == 0 {
		t.Fatalf("Tenant not found in database, tenantID=%d", tenantID)
	}

	// Test direct query with all columns
	var testSlug string
	err = db.QueryRow(`SELECT slug FROM tenants WHERE id = ?`, tenantID).Scan(&testSlug)
	if err != nil {
		t.Fatalf("Direct query failed: %v", err)
	}
	t.Logf("Direct query succeeded: slug=%s, tenantID=%d", testSlug, tenantID)

	// Test using repository directly
	tenantRepo := repository.NewTenantRepository(db)
	foundTenant, err := tenantRepo.FindByID(int(tenantID))
	if err != nil {
		t.Logf("TenantRepo.FindByID error: %v", err)
	}
	if foundTenant == nil {
		t.Logf("TenantRepo.FindByID returned nil tenant")
	} else {
		t.Logf("TenantRepo.FindByID succeeded: %+v", foundTenant)
	}

	// Create request with tenant context
	req := httptest.NewRequest("GET", "/api/v1/admin/tenant/export", nil)
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, int(tenantID))
	ctx = context.WithValue(ctx, middleware.UserIDKey, int(userID))
	ctx = context.WithValue(ctx, middleware.IsAdminKey, true)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ExportTenantData(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s (tenantID=%d)", rec.Code, rec.Body.String(), tenantID)
	}

	// Parse response
	var export map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &export); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// BUG #4: dogs should be present and not nil
	dogs, ok := export["dogs"]
	if !ok {
		t.Errorf("Export should contain 'dogs' key")
	}
	if dogs == nil {
		t.Errorf("dogs should not be nil")
	}

	// Should have 3 dogs
	dogCount, _ := export["dog_count"].(float64)
	if dogCount != 3 {
		t.Errorf("Expected dog_count=3, got %v", dogCount)
	}

	// Verify dogs array
	dogsArr, ok := dogs.([]interface{})
	if !ok {
		t.Errorf("dogs should be an array, got %T", dogs)
	}
	if len(dogsArr) != 3 {
		t.Errorf("Expected 3 dogs in array, got %d", len(dogsArr))
	}
}

// BUG #4: Test central admin export includes dogs
func TestCentralAdminHandler_ExportTenantData_IncludesDogs(t *testing.T) {
	db := setupExportTestDB(t)
	defer db.Close()

	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewCentralAdminHandler(db, cfg)

	// Create a tenant
	result, err := db.Exec(`INSERT INTO tenants (slug, name, contact_email, status, federal_state, created_at, updated_at) VALUES ('test-tenant', 'Test Tenant', 'test@tenant.com', 'active', 'BW', ?, ?)`, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	tenantID, _ := result.LastInsertId()

	// Create a central admin user
	result, err = db.Exec(`INSERT INTO users (email, first_name, last_name, is_central_admin, is_verified, terms_accepted_at) VALUES ('central@admin.com', 'Central', 'Admin', 1, 1, ?)`, time.Now())
	if err != nil {
		t.Fatalf("Failed to create central admin: %v", err)
	}
	adminID, _ := result.LastInsertId()

	// Create dogs for the tenant
	_, err = db.Exec(`INSERT INTO dogs (tenant_id, name, breed, size, age, is_available, is_featured) VALUES (?, 'Bella', 'Labrador', 'large', 3, 1, 0)`, tenantID)
	if err != nil {
		t.Fatalf("Failed to create dog: %v", err)
	}
	_, err = db.Exec(`INSERT INTO dogs (tenant_id, name, breed, size, age, is_available, is_featured) VALUES (?, 'Max', 'Golden Retriever', 'large', 5, 1, 1)`, tenantID)
	if err != nil {
		t.Fatalf("Failed to create dog: %v", err)
	}

	// Create request
	req := httptest.NewRequest("GET", "/api/v1/central-admin/tenants/"+strconv.FormatInt(tenantID, 10)+"/export", nil)
	req = mux.SetURLVars(req, map[string]string{"id": strconv.FormatInt(tenantID, 10)})
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, int(adminID))
	ctx = context.WithValue(ctx, middleware.IsCentralAdminKey, true)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ExportTenantData(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// Parse response
	var export map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &export); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// BUG #4: dogs should be present and not nil
	dogs := export["dogs"]
	if dogs == nil {
		t.Errorf("dogs should not be nil, got: %v", dogs)
	}

	// Verify dogs array
	dogsArr, ok := dogs.([]interface{})
	if !ok {
		t.Errorf("dogs should be an array, got %T", dogs)
	} else if len(dogsArr) != 2 {
		t.Errorf("Expected 2 dogs in array, got %d", len(dogsArr))
	}
}
