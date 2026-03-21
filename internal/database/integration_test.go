package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInitializeWithConfig_SQLite tests SQLite initialization
func TestInitializeWithConfig_SQLite(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	config := &DBConfig{
		Type: "sqlite",
		Path: dbPath,
	}

	db, dialect, err := InitializeWithConfig(config)
	require.NoError(t, err)
	defer db.Close()

	// Verify dialect
	assert.Equal(t, "sqlite", dialect.Name())

	// Verify connection works
	err = db.Ping()
	assert.NoError(t, err)

	// Verify foreign keys enabled
	var fkEnabled int
	err = db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
	assert.NoError(t, err)
	assert.Equal(t, 1, fkEnabled, "Foreign keys should be enabled")

	// Run migrations
	err = RunMigrationsWithDialect(db.SqlDB(), dialect)
	assert.NoError(t, err)

	// Verify tables created
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	assert.NoError(t, err)
}

// TestInitializeWithConfig_PostgreSQL tests PostgreSQL initialization (if available)
func TestInitializeWithConfig_PostgreSQL(t *testing.T) {
	// Get PostgreSQL test connection string
	dsn := os.Getenv("DB_TEST_POSTGRES")
	if dsn == "" {
		t.Skip("PostgreSQL test database not configured (set DB_TEST_POSTGRES)")
	}

	config := &DBConfig{
		Type:             "postgres",
		ConnectionString: dsn,
	}

	db, dialect, err := InitializeWithConfig(config)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer db.Close()

	// Clean database first
	tables := []string{"bookings", "blocked_dates",
		"reactivation_requests", "dogs", "users", "system_settings", "schema_migrations"}
	for _, table := range tables {
		db.Exec("DROP TABLE IF EXISTS " + table + " CASCADE")
	}

	// Verify dialect
	assert.Equal(t, "postgres", dialect.Name())

	// Verify connection works
	err = db.Ping()
	assert.NoError(t, err)

	// Run migrations
	err = RunMigrationsWithDialect(db.SqlDB(), dialect)
	assert.NoError(t, err)

	// Verify tables created
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	assert.NoError(t, err)

	// Verify timezone set to UTC
	var timezone string
	err = db.QueryRow("SHOW TIME ZONE").Scan(&timezone)
	if err == nil {
		assert.Equal(t, "UTC", timezone, "Timezone should be UTC")
	}

	// Verify connection pool configured
	stats := db.Stats()
	assert.Equal(t, 25, stats.MaxOpenConnections, "MaxOpenConns should be 25")
}

// TestBuildPostgreSQLDSN tests PostgreSQL connection string builder
func TestBuildPostgreSQLDSN(t *testing.T) {
	testCases := []struct {
		name     string
		config   *DBConfig
		expected string
	}{
		{
			name: "All fields specified with SSL",
			config: &DBConfig{
				Host:     "db.example.com",
				Port:     5432,
				Database: "mydb",
				Username: "myuser",
				Password: "mypass",
				SSLMode:  "require",
			},
			expected: "postgres://myuser:mypass@db.example.com:5432/mydb?sslmode=require",
		},
		{
			name: "Default port and SSL disabled",
			config: &DBConfig{
				Host:     "localhost",
				Port:     0, // Should use default 5432
				Database: "gassigeher",
				Username: "user",
				Password: "pass",
				SSLMode:  "",
			},
			expected: "postgres://user:pass@localhost:5432/gassigeher?sslmode=disable",
		},
		{
			name: "Empty host defaults to localhost",
			config: &DBConfig{
				Host:     "",
				Port:     5432,
				Database: "gassigeher",
				Username: "user",
				Password: "pass",
				SSLMode:  "disable",
			},
			expected: "postgres://user:pass@localhost:5432/gassigeher?sslmode=disable",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := buildPostgreSQLDSN(tc.config)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestBackwardCompatibility_Initialize tests that old Initialize() still works
func TestBackwardCompatibility_Initialize(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_compat.db")

	// Old API should still work
	db, err := Initialize(dbPath)
	require.NoError(t, err)
	defer db.Close()

	// Verify it works
	err = db.Ping()
	assert.NoError(t, err)

	// Run old migrations (should still work)
	err = RunMigrations(db.SqlDB())
	assert.NoError(t, err)

	// Verify tables created
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	assert.NoError(t, err)
}

// TestDialectFactory_Integration tests dialect creation in real scenario
func TestDialectFactory_Integration(t *testing.T) {
	testCases := []struct {
		dbType          string
		expectedDialect string
	}{
		{"sqlite", "sqlite"},
		{"", "sqlite"}, // Empty defaults to SQLite
		{"postgres", "postgres"},
		{"postgresql", "postgres"}, // Alias
		{"SQLITE", "sqlite"},       // Case insensitive
		{"POSTGRES", "postgres"},
	}

	for _, tc := range testCases {
		t.Run("Type_"+tc.dbType, func(t *testing.T) {
			factory := NewDialectFactory()
			dialect, err := factory.GetDialect(tc.dbType)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedDialect, dialect.Name())
		})
	}
}

// TestConfigureConnectionPool tests connection pool configuration
func TestConfigureConnectionPool(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_pool.db")
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer db.Close()

	config := &DBConfig{
		MaxOpenConns:    50,
		MaxIdleConns:    10,
		ConnMaxLifetime: 10, // minutes
	}

	configureConnectionPool(db, config)

	stats := db.Stats()
	assert.Equal(t, 50, stats.MaxOpenConnections)
	// Note: MaxIdleConns and ConnMaxLifetime can't be verified directly via stats
	// but SetMaxIdleConns and SetConnMaxLifetime were called
}
