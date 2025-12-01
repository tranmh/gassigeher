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

	t.Run("All_15_migrations_registered", func(t *testing.T) {
		assert.Len(t, migrations, 15, "Should have 15 migrations")
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

	t.Run("All_migrations_support_all_databases", func(t *testing.T) {
		requiredDialects := []string{"sqlite", "mysql", "postgres"}

		for _, m := range migrations {
			for _, dialect := range requiredDialects {
				sql, ok := m.Up[dialect]
				assert.True(t, ok, "Migration %s missing SQL for %s", m.ID, dialect)
				assert.NotEmpty(t, sql, "Migration %s has empty SQL for %s", m.ID, dialect)
			}
		}
	})
}

// TestRunMigrations_SQLite tests running migrations on SQLite
func TestRunMigrations_SQLite(t *testing.T) {
	// Create temporary SQLite database
	dbPath := filepath.Join(t.TempDir(), "test_migrations.db")
	db, err := sql.Open("sqlite3", dbPath)
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
	assert.Equal(t, 15, count, "Should have 15 applied migrations")

	// Verify all tables created
	tables := []string{
		"users", "dogs", "bookings", "blocked_dates",
		"experience_requests", "system_settings", "reactivation_requests",
	}

	for _, table := range tables {
		err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		assert.NoError(t, err, "Table %s should exist", table)
	}

	// Verify default settings inserted (3 from migration 008 + 5 from migration 012)
	err = db.QueryRow("SELECT COUNT(*) FROM system_settings").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 8, count, "Should have 8 default settings")

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
	db, err := sql.Open("sqlite3", dbPath)
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
	assert.Equal(t, 15, count)

	// Run migrations second time (should be idempotent)
	err = RunMigrationsWithDialect(db, dialect)
	assert.NoError(t, err, "Second migration run should succeed (idempotent)")

	// Count should still be 15 (no duplicates)
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 15, count, "Should still have 15 migrations (no duplicates)")
}

// TestGetMigrationStatus tests migration status reporting
func TestGetMigrationStatus(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_status.db")
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	defer db.Close()

	dialect := NewSQLiteDialect()
	err = dialect.ApplySettings(db)
	require.NoError(t, err)

	// Before migrations
	applied, pending, err := GetMigrationStatus(db, dialect)
	assert.NoError(t, err)
	assert.Equal(t, 0, applied)
	assert.Equal(t, 15, pending)

	// After migrations
	err = RunMigrationsWithDialect(db, dialect)
	require.NoError(t, err)

	applied, pending, err = GetMigrationStatus(db, dialect)
	assert.NoError(t, err)
	assert.Equal(t, 15, applied)
	assert.Equal(t, 0, pending)
}

// TestMigrationRunner_HandlesDuplicateColumn tests graceful handling of duplicate column errors
func TestMigrationRunner_HandlesDuplicateColumn(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_duplicate.db")
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	defer db.Close()

	dialect := NewSQLiteDialect()
	err = dialect.ApplySettings(db)
	require.NoError(t, err)

	// Run all migrations
	err = RunMigrationsWithDialect(db, dialect)
	require.NoError(t, err)

	// Manually try to add photo_thumbnail again (should be handled gracefully)
	// This simulates running migration 009 again
	_, err = db.Exec("ALTER TABLE dogs ADD COLUMN photo_thumbnail TEXT")

	// Error expected (column exists), but migration system should handle it
	assert.Error(t, err, "Direct execution should fail")
	assert.Contains(t, err.Error(), "duplicate column")

	// But if we run through migration system again, it should handle it
	err = RunMigrationsWithDialect(db, dialect)
	assert.NoError(t, err, "Migration system should handle duplicate gracefully")
}

// TestMigration_SQLConsistency tests that SQL is valid for each database
func TestMigration_SQLConsistency(t *testing.T) {
	migrations := GetAllMigrations()

	for _, m := range migrations {
		t.Run(m.ID, func(t *testing.T) {
			// Test SQLite SQL
			sqliteSQL := m.Up["sqlite"]
			assert.NotEmpty(t, sqliteSQL)

			// Test MySQL SQL
			mysqlSQL := m.Up["mysql"]
			assert.NotEmpty(t, mysqlSQL)

			// If creating a table, MySQL should have ENGINE clause
			if contains(mysqlSQL, "CREATE TABLE") {
				assert.Contains(t, mysqlSQL, "ENGINE=InnoDB",
					"MySQL CREATE TABLE should specify InnoDB engine")
				assert.Contains(t, mysqlSQL, "CHARSET=utf8mb4",
					"MySQL CREATE TABLE should specify utf8mb4 charset")
			}

			// Test PostgreSQL SQL
			postgresSQL := m.Up["postgres"]
			assert.NotEmpty(t, postgresSQL)

			// All SQL should contain either TABLE, INSERT, or ALTER (valid migration types)
			isValid := contains(sqliteSQL, "TABLE") || contains(sqliteSQL, "INSERT") || contains(sqliteSQL, "ALTER")
			assert.True(t, isValid, "Migration SQL should contain TABLE, INSERT, or ALTER")
		})
	}
}

// TestMigration_TypeConsistency tests that type mappings are correct
func TestMigration_TypeConsistency(t *testing.T) {
	migrations := GetAllMigrations()

	// Check first migration (users table) for proper type mappings
	usersMigration := migrations[0]
	assert.Equal(t, "001_create_users_table", usersMigration.ID)

	t.Run("SQLite_uses_INTEGER_PRIMARY_KEY_AUTOINCREMENT", func(t *testing.T) {
		assert.Contains(t, usersMigration.Up["sqlite"], "INTEGER PRIMARY KEY AUTOINCREMENT")
	})

	t.Run("MySQL_uses_INT_AUTO_INCREMENT", func(t *testing.T) {
		assert.Contains(t, usersMigration.Up["mysql"], "INT AUTO_INCREMENT PRIMARY KEY")
	})

	t.Run("PostgreSQL_uses_SERIAL", func(t *testing.T) {
		assert.Contains(t, usersMigration.Up["postgres"], "SERIAL PRIMARY KEY")
	})

	t.Run("SQLite_uses_INTEGER_for_booleans", func(t *testing.T) {
		// is_verified INTEGER DEFAULT 0
		assert.Contains(t, usersMigration.Up["sqlite"], "is_verified INTEGER")
	})

	t.Run("MySQL_uses_TINYINT_for_booleans", func(t *testing.T) {
		assert.Contains(t, usersMigration.Up["mysql"], "is_verified TINYINT(1)")
	})

	t.Run("PostgreSQL_uses_BOOLEAN", func(t *testing.T) {
		assert.Contains(t, usersMigration.Up["postgres"], "is_verified BOOLEAN")
	})
}

// TestMigration_InsertOrIgnore tests that migration 008 uses correct syntax
func TestMigration_InsertOrIgnore(t *testing.T) {
	migrations := GetAllMigrations()

	// Find migration 008 (insert default settings)
	var settingsMigration *Migration
	for _, m := range migrations {
		if m.ID == "008_insert_default_settings" {
			settingsMigration = m
			break
		}
	}

	require.NotNil(t, settingsMigration, "Migration 008 should exist")

	t.Run("SQLite_uses_INSERT_OR_IGNORE", func(t *testing.T) {
		assert.Contains(t, settingsMigration.Up["sqlite"], "INSERT OR IGNORE")
	})

	t.Run("MySQL_uses_INSERT_IGNORE", func(t *testing.T) {
		assert.Contains(t, settingsMigration.Up["mysql"], "INSERT IGNORE")
	})

	t.Run("PostgreSQL_uses_ON_CONFLICT", func(t *testing.T) {
		assert.Contains(t, settingsMigration.Up["postgres"], "ON CONFLICT")
		assert.Contains(t, settingsMigration.Up["postgres"], "DO NOTHING")
	})

	t.Run("All_insert_same_values", func(t *testing.T) {
		// All should insert the same 3 settings
		for dialect, sql := range settingsMigration.Up {
			assert.Contains(t, sql, "booking_advance_days", "Missing setting in %s", dialect)
			assert.Contains(t, sql, "cancellation_notice_hours", "Missing setting in %s", dialect)
			assert.Contains(t, sql, "auto_deactivation_days", "Missing setting in %s", dialect)
		}
	})
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
				db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "test.db"))
				require.NoError(t, err)
				return db
			},
		},
		// MySQL and PostgreSQL tests would go here when test databases are available
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
			err = markMigrationAsApplied(db, "test_migration_001")
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

// TestMigrationOrder tests that migrations are applied in correct order
func TestMigrationOrder(t *testing.T) {
	migrations := GetAllMigrations()

	expectedOrder := []string{
		"001_create_users_table",
		"002_create_dogs_table",
		"003_create_bookings_table",
		"004_create_blocked_dates_table",
		"005_create_experience_requests_table",
		"006_create_system_settings_table",
		"007_create_reactivation_requests_table",
		"008_insert_default_settings",
		"009_add_photo_thumbnail_column",
		"010_add_admin_flags",
		"012_booking_times",
		"013_remove_walk_type",
		"014_add_featured_dogs",
		"015_add_external_link",
		"016_add_reminder_sent",
	}

	assert.Len(t, migrations, len(expectedOrder))

	for i, expected := range expectedOrder {
		assert.Equal(t, expected, migrations[i].ID,
			"Migration %d should be %s", i+1, expected)
	}
}

// TestMigrationRunner_PartialApplication tests applying migrations incrementally
func TestMigrationRunner_PartialApplication(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_partial.db")
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	defer db.Close()

	dialect := NewSQLiteDialect()
	err = dialect.ApplySettings(db)
	require.NoError(t, err)

	// Create schema_migrations table and mark some migrations as applied
	err = createSchemaMigrationsTable(db, dialect)
	require.NoError(t, err)

	// Simulate that first 3 migrations were already applied
	for i := 1; i <= 3; i++ {
		migrationID := fmt.Sprintf("00%d_", i) // This won't match actual IDs perfectly
		err = markMigrationAsApplied(db, migrationID)
		require.NoError(t, err)
	}

	// Now run all migrations
	// Should only apply migrations 4-9 (plus 1-3 that match actual IDs)
	err = RunMigrationsWithDialect(db, dialect)
	assert.NoError(t, err)

	// Verify all migrations applied
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, count, 9, "Should have at least 9 migrations applied")
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
		{"MySQL_AlreadyExists", NewMySQLDialect(), "Table 'users' already exists", true},
		{"MySQL_DuplicateColumn", NewMySQLDialect(), "Duplicate column name 'photo'", true},
		{"MySQL_OtherError", NewMySQLDialect(), "syntax error", false},
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
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	defer db.Close()

	dialect := NewSQLiteDialect()
	err = dialect.ApplySettings(db)
	require.NoError(t, err, "Foreign keys should be enabled")

	// Run migrations
	err = RunMigrationsWithDialect(db, dialect)
	require.NoError(t, err)

	// Test foreign key constraint (insert booking with invalid user_id)
	_, err = db.Exec(`
		INSERT INTO bookings (user_id, dog_id, date, walk_type, scheduled_time)
		VALUES (99999, 1, '2025-12-01', 'morning', '09:00')
	`)

	// Should fail due to foreign key constraint
	assert.Error(t, err, "Foreign key constraint should prevent invalid user_id")
}

// TestMigrationRunner_CreatesIndexes tests that indexes are created
func TestMigrationRunner_CreatesIndexes(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_indexes.db")
	db, err := sql.Open("sqlite3", dbPath)
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
		"idx_reactivation_pending",
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
