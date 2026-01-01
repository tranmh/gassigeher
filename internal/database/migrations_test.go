package database

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigrationRegistry tests that all migrations are registered
func TestMigrationRegistry(t *testing.T) {
	migrations := GetAllMigrations()

	t.Run("Schema_migration_registered", func(t *testing.T) {
		assert.GreaterOrEqual(t, len(migrations), 1, "Should have at least 1 migration")
	})

	t.Run("Migrations_have_unique_IDs", func(t *testing.T) {
		ids := make(map[string]bool)
		for _, m := range migrations {
			assert.False(t, ids[m.ID], "Duplicate migration ID: %s", m.ID)
			ids[m.ID] = true
		}
	})

	t.Run("Migrations_sorted_by_ID", func(t *testing.T) {
		for i := 0; i < len(migrations)-1; i++ {
			assert.Less(t, migrations[i].ID, migrations[i+1].ID,
				"Migrations should be sorted by ID")
		}
	})

	t.Run("All_migrations_have_descriptions", func(t *testing.T) {
		for _, m := range migrations {
			assert.NotEmpty(t, m.Description, "Migration %s missing description", m.ID)
		}
	})

	t.Run("All_migrations_support_sqlite", func(t *testing.T) {
		for _, m := range migrations {
			sql, ok := m.Up["sqlite"]
			assert.True(t, ok, "Migration %s missing SQL for sqlite", m.ID)
			assert.NotEmpty(t, sql, "Migration %s has empty SQL for sqlite", m.ID)
		}
	})
}

// TestRunMigrations_SQLite tests running migrations on SQLite
func TestRunMigrations_SQLite(t *testing.T) {
	// Create temporary SQLite database
	dbPath := filepath.Join(t.TempDir(), "test_migrations.db")
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer db.Close()

	dialect := NewSQLiteDialect()

	// Apply dialect settings
	err = dialect.ApplySettings(db)
	require.NoError(t, err)

	// Run migrations
	err = RunMigrationsWithDialect(db, dialect)
	assert.NoError(t, err, "Migrations should succeed")

	// Verify schema_migrations table created
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, count, 1, "Should have at least 1 applied migration")

	// Verify core tables created
	tables := []string{
		"tenants", "users", "dogs", "bookings", "blocked_dates",
		"color_categories", "system_settings",
	}

	for _, table := range tables {
		err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		assert.NoError(t, err, "Table %s should exist", table)
	}

	// Verify photo_thumbnail column exists in dogs table
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('dogs') WHERE name='photo_thumbnail'
	`).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count, "photo_thumbnail column should exist")
}

// TestRunMigrations_Idempotent tests that migrations can be run multiple times
func TestRunMigrations_Idempotent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_idempotent.db")
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer db.Close()

	dialect := NewSQLiteDialect()
	err = dialect.ApplySettings(db)
	require.NoError(t, err)

	// Run migrations first time
	err = RunMigrationsWithDialect(db, dialect)
	assert.NoError(t, err, "First migration run should succeed")

	// Get migration count
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	assert.NoError(t, err)
	firstCount := count

	// Run migrations second time (should be idempotent)
	err = RunMigrationsWithDialect(db, dialect)
	assert.NoError(t, err, "Second migration run should succeed (idempotent)")

	// Count should still be the same (no duplicates)
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, firstCount, count, "Should have same number of migrations (no duplicates)")
}

// TestGetMigrationStatus tests migration status reporting
func TestGetMigrationStatus(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_status.db")
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer db.Close()

	dialect := NewSQLiteDialect()
	err = dialect.ApplySettings(db)
	require.NoError(t, err)

	migrations := GetAllMigrations()
	totalMigrations := len(migrations)

	// Before migrations
	applied, pending, err := GetMigrationStatus(db, dialect)
	assert.NoError(t, err)
	assert.Equal(t, 0, applied)
	assert.Equal(t, totalMigrations, pending)

	// After migrations
	err = RunMigrationsWithDialect(db, dialect)
	require.NoError(t, err)

	applied, pending, err = GetMigrationStatus(db, dialect)
	assert.NoError(t, err)
	assert.Equal(t, totalMigrations, applied)
	assert.Equal(t, 0, pending)
}

// TestMigrationRunner_HandlesDuplicateColumn tests graceful handling of duplicate column errors
func TestMigrationRunner_HandlesDuplicateColumn(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_duplicate.db")
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer db.Close()

	dialect := NewSQLiteDialect()
	err = dialect.ApplySettings(db)
	require.NoError(t, err)

	// Run all migrations
	err = RunMigrationsWithDialect(db, dialect)
	require.NoError(t, err)

	// Manually try to add photo_thumbnail again (should be handled gracefully)
	_, err = db.Exec("ALTER TABLE dogs ADD COLUMN photo_thumbnail TEXT")

	// Error expected (column exists), but migration system should handle it
	assert.Error(t, err, "Direct execution should fail")
	assert.Contains(t, err.Error(), "duplicate column")

	// But if we run through migration system again, it should handle it
	err = RunMigrationsWithDialect(db, dialect)
	assert.NoError(t, err, "Migration system should handle duplicate gracefully")
}

// TestCreateSchemaMigrationsTable tests schema_migrations table creation
func TestCreateSchemaMigrationsTable(t *testing.T) {
	dialects := []struct {
		name    string
		dialect Dialect
		setup   func(t *testing.T) *sql.DB
	}{
		{
			name:    "SQLite",
			dialect: NewSQLiteDialect(),
			setup: func(t *testing.T) *sql.DB {
				db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "test.db"))
				require.NoError(t, err)
				return db
			},
		},
	}

	for _, tc := range dialects {
		t.Run(tc.name, func(t *testing.T) {
			db := tc.setup(t)
			defer db.Close()

			// Apply settings
			err := tc.dialect.ApplySettings(db)
			require.NoError(t, err)

			// Create schema_migrations table
			err = createSchemaMigrationsTable(db, tc.dialect)
			assert.NoError(t, err)

			// Verify table exists
			var count int
			err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
			assert.NoError(t, err)
			assert.Equal(t, 0, count, "Table should be empty initially")

			// Verify we can insert a migration record
			err = markMigrationAsApplied(db, tc.dialect, "test_migration_001")
			assert.NoError(t, err)

			err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
			assert.NoError(t, err)
			assert.Equal(t, 1, count)

			// Verify we can query applied migrations
			applied, err := getAppliedMigrations(db)
			assert.NoError(t, err)
			assert.True(t, applied["test_migration_001"])
		})
	}
}

// TestIsAlreadyExistsError tests error detection for different databases
func TestIsAlreadyExistsError(t *testing.T) {
	testCases := []struct {
		name     string
		dialect  Dialect
		errMsg   string
		expected bool
	}{
		{"SQLite_AlreadyExists", NewSQLiteDialect(), "table users already exists", true},
		{"SQLite_DuplicateColumn", NewSQLiteDialect(), "duplicate column name: photo", true},
		{"SQLite_OtherError", NewSQLiteDialect(), "syntax error", false},
		{"PostgreSQL_AlreadyExists", NewPostgreSQLDialect(), "relation \"users\" already exists", true},
		{"PostgreSQL_DuplicateColumn", NewPostgreSQLDialect(), "column \"photo\" of relation \"dogs\" already exists", true},
		{"PostgreSQL_OtherError", NewPostgreSQLDialect(), "syntax error", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := fmt.Errorf("%s", tc.errMsg)
			result := isAlreadyExistsError(err, tc.dialect)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestMigrationRunner_CreatesForeignKeys tests that foreign keys are created properly
func TestMigrationRunner_CreatesForeignKeys(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_fk.db")
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer db.Close()

	dialect := NewSQLiteDialect()
	err = dialect.ApplySettings(db)
	require.NoError(t, err, "Foreign keys should be enabled")

	// Run migrations
	err = RunMigrationsWithDialect(db, dialect)
	require.NoError(t, err)

	// Create a tenant first (needed for foreign key)
	_, err = db.Exec(`INSERT INTO tenants (slug, name, contact_email) VALUES ('test', 'Test', 'test@test.com')`)
	require.NoError(t, err)

	// Test foreign key constraint (insert booking with invalid user_id)
	_, err = db.Exec(`
		INSERT INTO bookings (tenant_id, user_id, dog_id, date, scheduled_time)
		VALUES (1, 99999, 1, '2025-12-01', '09:00')
	`)

	// Should fail due to foreign key constraint
	assert.Error(t, err, "Foreign key constraint should prevent invalid user_id")
}

// TestMigrationRunner_CreatesIndexes tests that indexes are created
func TestMigrationRunner_CreatesIndexes(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_indexes.db")
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer db.Close()

	dialect := NewSQLiteDialect()
	err = dialect.ApplySettings(db)
	require.NoError(t, err)

	// Run migrations
	err = RunMigrationsWithDialect(db, dialect)
	require.NoError(t, err)

	// Check that indexes exist (SQLite-specific query)
	indexes := []string{
		"idx_users_email",
		"idx_users_last_activity",
		"idx_dogs_available",
		"idx_bookings_tenant",
	}

	for _, indexName := range indexes {
		var count int
		err = db.QueryRow(`
			SELECT COUNT(*) FROM sqlite_master
			WHERE type='index' AND name=?
		`, indexName).Scan(&count)

		assert.NoError(t, err)
		assert.Equal(t, 1, count, "Index %s should exist", indexName)
	}
}
