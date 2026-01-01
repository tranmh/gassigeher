package repository

import (
	"database/sql"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/tranmh/gassigeher/internal/database"
	"github.com/tranmh/gassigeher/internal/models"
)

// setupFeatureFlagTestDB creates a test database with the feature_flags schema
// BUG: This test will fail if the feature_flags migration is missing
func setupFeatureFlagTestDB(t *testing.T) *database.DB {
	rawDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Create required tables - this is what the migration should create
	_, err = rawDB.Exec(`
		CREATE TABLE IF NOT EXISTS feature_flags (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			description TEXT,
			is_global INTEGER DEFAULT 0,
			is_enabled INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS tenant_feature_flags (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tenant_id INTEGER NOT NULL,
			feature_flag_id INTEGER NOT NULL,
			is_enabled INTEGER DEFAULT 0,
			enabled_at TIMESTAMP,
			enabled_by INTEGER,
			UNIQUE(tenant_id, feature_flag_id)
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	// Wrap in database.DB
	sqlxDB := sqlx.NewDb(rawDB, "sqlite3")
	return database.WrapSqlxDB(sqlxDB, database.NewSQLiteDialect())
}

// TestFeatureFlagRepository_Create tests creating a feature flag
func TestFeatureFlagRepository_Create(t *testing.T) {
	db := setupFeatureFlagTestDB(t)
	defer db.Close()

	repo := NewFeatureFlagRepository(db)

	flag := &models.FeatureFlag{
		Key:         "test_feature",
		Name:        "Test Feature",
		Description: "A test feature flag",
		IsGlobal:    true,
		IsEnabled:   false,
	}

	err := repo.Create(flag)
	if err != nil {
		t.Fatalf("Failed to create feature flag: %v", err)
	}

	if flag.ID == 0 {
		t.Error("Expected flag ID to be set after create")
	}
}

// TestFeatureFlagRepository_GetAll tests getting all feature flags
func TestFeatureFlagRepository_GetAll(t *testing.T) {
	db := setupFeatureFlagTestDB(t)
	defer db.Close()

	repo := NewFeatureFlagRepository(db)

	// Create a few flags
	flags := []*models.FeatureFlag{
		{Key: "feature_a", Name: "Feature A", IsGlobal: true, IsEnabled: true},
		{Key: "feature_b", Name: "Feature B", IsGlobal: false, IsEnabled: false},
	}

	for _, f := range flags {
		err := repo.Create(f)
		if err != nil {
			t.Fatalf("Failed to create flag: %v", err)
		}
	}

	// Get all
	result, err := repo.GetAll()
	if err != nil {
		t.Fatalf("Failed to get all flags: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 flags, got %d", len(result))
	}
}

// TestFeatureFlagRepository_GetByKey tests getting a flag by key
func TestFeatureFlagRepository_GetByKey(t *testing.T) {
	db := setupFeatureFlagTestDB(t)
	defer db.Close()

	repo := NewFeatureFlagRepository(db)

	// Create a flag
	flag := &models.FeatureFlag{
		Key:       "unique_key",
		Name:      "Unique Feature",
		IsGlobal:  true,
		IsEnabled: true,
	}
	err := repo.Create(flag)
	if err != nil {
		t.Fatalf("Failed to create flag: %v", err)
	}

	// Get by key
	result, err := repo.GetByKey("unique_key")
	if err != nil {
		t.Fatalf("Failed to get flag by key: %v", err)
	}

	if result == nil {
		t.Fatal("Expected flag to be found")
	}
	if result.Key != "unique_key" {
		t.Errorf("Expected key 'unique_key', got '%s'", result.Key)
	}
}

// TestFeatureFlagRepository_IsEnabled tests checking if a flag is enabled
func TestFeatureFlagRepository_IsEnabled(t *testing.T) {
	db := setupFeatureFlagTestDB(t)
	defer db.Close()

	repo := NewFeatureFlagRepository(db)

	// Create a global enabled flag
	flag := &models.FeatureFlag{
		Key:       "global_enabled",
		Name:      "Global Enabled",
		IsGlobal:  true,
		IsEnabled: true,
	}
	err := repo.Create(flag)
	if err != nil {
		t.Fatalf("Failed to create flag: %v", err)
	}

	// Check if enabled for tenant 1
	enabled, err := repo.IsEnabled("global_enabled", 1)
	if err != nil {
		t.Fatalf("Failed to check if enabled: %v", err)
	}

	if !enabled {
		t.Error("Expected global enabled flag to be enabled for tenant")
	}
}

// TestFeatureFlagRepository_SetTenantFlag tests setting a tenant-specific flag
func TestFeatureFlagRepository_SetTenantFlag(t *testing.T) {
	db := setupFeatureFlagTestDB(t)
	defer db.Close()

	repo := NewFeatureFlagRepository(db)

	// Create a flag
	flag := &models.FeatureFlag{
		Key:       "tenant_flag",
		Name:      "Tenant Flag",
		IsGlobal:  false,
		IsEnabled: false,
	}
	err := repo.Create(flag)
	if err != nil {
		t.Fatalf("Failed to create flag: %v", err)
	}

	// Set tenant override
	err = repo.SetTenantFlag(1, flag.ID, true, nil)
	if err != nil {
		t.Fatalf("Failed to set tenant flag: %v", err)
	}

	// Check if enabled for tenant 1
	enabled, err := repo.IsEnabled("tenant_flag", 1)
	if err != nil {
		t.Fatalf("Failed to check if enabled: %v", err)
	}

	if !enabled {
		t.Error("Expected tenant flag to be enabled after override")
	}
}
