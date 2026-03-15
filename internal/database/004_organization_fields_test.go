package database

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// TestMigration004_OrganizationFields verifies the migration adds organization columns
func TestMigration004_OrganizationFields(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_org_fields.db")
	rawDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer rawDB.Close()

	dialect := NewSQLiteDialect()
	if err := dialect.ApplySettings(rawDB); err != nil {
		t.Fatalf("Failed to apply settings: %v", err)
	}

	// Run all migrations
	if err := RunMigrationsWithDialect(rawDB, dialect); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	// Verify organization columns exist in tenant_settings
	columns := []string{
		"organization_name", "organization_address", "organization_email",
		"organization_phone", "privacy_officer_email",
	}
	for _, col := range columns {
		var count int
		err := rawDB.QueryRow(`
			SELECT COUNT(*) FROM pragma_table_info('tenant_settings') WHERE name = ?
		`, col).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to check column %s: %v", col, err)
		}
		if count != 1 {
			t.Errorf("Expected column %s to exist in tenant_settings, but it doesn't", col)
		}
	}

	// Verify we can insert and read organization fields
	_, err = rawDB.Exec(`
		INSERT INTO tenants (id, slug, name, contact_email, status, created_at, updated_at)
		VALUES (99, 'migration-test', 'Migration Test', 'test@test.com', 'active', datetime('now'), datetime('now'))
	`)
	if err != nil {
		t.Fatalf("Failed to insert test tenant: %v", err)
	}

	_, err = rawDB.Exec(`
		INSERT INTO tenant_settings (tenant_id, theme_preset, organization_name, organization_address,
			organization_email, organization_phone, privacy_officer_email, created_at, updated_at)
		VALUES (99, 'classic', 'Test Org', 'Test Address', 'test@org.de', '0123', 'dsgvo@org.de',
			datetime('now'), datetime('now'))
	`)
	if err != nil {
		t.Fatalf("Failed to insert tenant_settings with org fields: %v", err)
	}

	var orgName, orgEmail, privacyEmail sql.NullString
	err = rawDB.QueryRow(`
		SELECT organization_name, organization_email, privacy_officer_email
		FROM tenant_settings WHERE tenant_id = 99
	`).Scan(&orgName, &orgEmail, &privacyEmail)
	if err != nil {
		t.Fatalf("Failed to read org fields: %v", err)
	}

	if !orgName.Valid || orgName.String != "Test Org" {
		t.Errorf("organization_name: expected 'Test Org', got %v", orgName)
	}
	if !orgEmail.Valid || orgEmail.String != "test@org.de" {
		t.Errorf("organization_email: expected 'test@org.de', got %v", orgEmail)
	}
	if !privacyEmail.Valid || privacyEmail.String != "dsgvo@org.de" {
		t.Errorf("privacy_officer_email: expected 'dsgvo@org.de', got %v", privacyEmail)
	}

	// Verify NULL values work (fields should be nullable)
	_, err = rawDB.Exec(`
		INSERT INTO tenants (id, slug, name, contact_email, status, created_at, updated_at)
		VALUES (100, 'null-test', 'Null Test', 'null@test.com', 'active', datetime('now'), datetime('now'))
	`)
	if err != nil {
		t.Fatalf("Failed to insert null test tenant: %v", err)
	}

	_, err = rawDB.Exec(`
		INSERT INTO tenant_settings (tenant_id, theme_preset, created_at, updated_at)
		VALUES (100, 'classic', datetime('now'), datetime('now'))
	`)
	if err != nil {
		t.Fatalf("Failed to insert tenant_settings without org fields: %v", err)
	}

	err = rawDB.QueryRow(`
		SELECT organization_name, organization_email, privacy_officer_email
		FROM tenant_settings WHERE tenant_id = 100
	`).Scan(&orgName, &orgEmail, &privacyEmail)
	if err != nil {
		t.Fatalf("Failed to read null org fields: %v", err)
	}

	if orgName.Valid {
		t.Error("Expected organization_name to be NULL for new settings")
	}
	if orgEmail.Valid {
		t.Error("Expected organization_email to be NULL for new settings")
	}
}
