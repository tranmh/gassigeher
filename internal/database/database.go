package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	_ "github.com/lib/pq"
	// _ "modernc.org/sqlite"      // CGO-based SQLite (faster, but requires CGO) - DISABLED for Windows
	_ "modernc.org/sqlite" // Pure Go SQLite (slower, but cross-compiles easily)
)

// Note: Migration files (001_*.go, 002_*.go, etc.) are in this package
// and register themselves via init() functions

// DBConfig holds database configuration
type DBConfig struct {
	Type             string // sqlite, postgres
	ConnectionString string // Full connection string (optional, overrides other fields)

	// SQLite-specific
	Path string

	// PostgreSQL-specific
	Host     string
	Port     int
	Database string
	Username string
	Password string
	SSLMode  string // PostgreSQL: disable, require, verify-full

	// Connection pool (PostgreSQL only)
	MaxOpenConns    int // Max simultaneous connections
	MaxIdleConns    int // Idle connections to keep
	ConnMaxLifetime int // Max connection age (minutes)
}

// DB wraps sqlx.DB and provides helper methods for cross-database compatibility
// All queries using ? placeholders are automatically converted to the correct format
type DB struct {
	*sqlx.DB
	dialect Dialect
}

// Rebind converts ? placeholders to the database-specific format ($1, $2 for postgres)
// This should be called on all queries with ? placeholders
func (db *DB) Rebind(query string) string {
	return db.DB.Rebind(query)
}

// GetDialect returns the database dialect
func (db *DB) GetDialect() Dialect {
	return db.dialect
}

// GetDialectName returns the database type name (sqlite, mysql, postgres)
func (db *DB) GetDialectName() string {
	return db.dialect.Name()
}

// SqlDB returns the underlying *sql.DB for compatibility with code that needs it
// (e.g., migrations, some third-party libraries)
func (db *DB) SqlDB() *sql.DB {
	return db.DB.DB
}

// SqlxDB returns the underlying *sqlx.DB for direct sqlx access
func (db *DB) SqlxDB() *sqlx.DB {
	return db.DB
}

// Exec executes a query with automatic placeholder rebinding
func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.DB.Exec(db.DB.Rebind(query), args...)
}

// ExecContext executes a query with context and automatic placeholder rebinding
func (db *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return db.DB.ExecContext(ctx, db.DB.Rebind(query), args...)
}

// Query executes a query that returns rows with automatic placeholder rebinding
func (db *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.DB.Query(db.DB.Rebind(query), args...)
}

// QueryContext executes a query with context that returns rows with automatic placeholder rebinding
func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return db.DB.QueryContext(ctx, db.DB.Rebind(query), args...)
}

// QueryRow executes a query that returns at most one row with automatic placeholder rebinding
func (db *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	return db.DB.QueryRow(db.DB.Rebind(query), args...)
}

// QueryRowContext executes a query with context that returns at most one row with automatic placeholder rebinding
func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return db.DB.QueryRowContext(ctx, db.DB.Rebind(query), args...)
}

// Prepare creates a prepared statement with automatic placeholder rebinding
func (db *DB) Prepare(query string) (*sql.Stmt, error) {
	return db.DB.Prepare(db.DB.Rebind(query))
}

// PrepareContext creates a prepared statement with context and automatic placeholder rebinding
func (db *DB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return db.DB.PrepareContext(ctx, db.DB.Rebind(query))
}

// Begin starts a transaction
func (db *DB) Begin() (*sql.Tx, error) {
	return db.DB.Begin()
}

// BeginTx starts a transaction with options
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return db.DB.BeginTx(ctx, opts)
}

// RebindQuery returns the rebound query for use in transactions
func (db *DB) RebindQuery(query string) string {
	return db.DB.Rebind(query)
}

// InsertReturningID executes an INSERT and returns the generated ID
// For PostgreSQL, the query should NOT include RETURNING - this method adds it
// For SQLite, uses LastInsertId()
func (db *DB) InsertReturningID(query string, args ...interface{}) (int64, error) {
	if db.dialect.Name() == "postgres" {
		// PostgreSQL requires RETURNING clause
		query = strings.TrimRight(query, " \t\n;") + " RETURNING id"
		var id int64
		err := db.QueryRow(query, args...).Scan(&id)
		if err != nil {
			return 0, err
		}
		return id, nil
	}

	// SQLite uses LastInsertId
	result, err := db.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// BoolValue returns the appropriate boolean representation for the database
// PostgreSQL: true/false, SQLite: 1/0
func (db *DB) BoolValue(b bool) interface{} {
	if db.dialect.Name() == "postgres" {
		return b
	}
	// SQLite uses integers for booleans
	if b {
		return 1
	}
	return 0
}

// BoolPlaceholder returns the correct comparison value for boolean columns in WHERE clauses
// For PostgreSQL, use this when comparing boolean columns
// Example: WHERE is_active = db.BoolPlaceholder(true)
// Returns: "true" for postgres, "1" for sqlite/mysql
func (db *DB) BoolPlaceholder(b bool) string {
	return db.dialect.GetBooleanDefault(b)
}

// Initialize creates and opens the database connection (OLD - backward compatible)
// Kept for backward compatibility with existing code
// New code should use InitializeWithConfig() for multi-database support
func Initialize(dbPath string) (*DB, error) {
	config := &DBConfig{
		Type: "sqlite",
		Path: dbPath,
	}
	db, _, err := InitializeWithConfig(config)
	return db, err
}

// WrapDB creates a new DB wrapper from an existing sqlx.DB and dialect
// This is useful for testing when you need to create a DB wrapper manually
func WrapDB(sqlxDB *sqlx.DB, dialect Dialect) *DB {
	return &DB{
		DB:      sqlxDB,
		dialect: dialect,
	}
}

// InitializeWithConfig creates and opens the database connection with full configuration
// Returns both the database connection and the dialect
func InitializeWithConfig(config *DBConfig) (*DB, Dialect, error) {
	var db *sqlx.DB
	var err error

	// Create dialect factory
	factory := NewDialectFactory()

	// Get dialect for database type
	dialect, err := factory.GetDialect(config.Type)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create dialect: %w", err)
	}

	// Build connection string and open database based on type
	switch dialect.Name() {
	case "sqlite":
		dsn := config.Path
		if dsn == "" {
			dsn = "./gassigeher.db"
		}
		db, err = sqlx.Open(dialect.GetDriverName(), dsn)

	case "postgres":
		dsn := config.ConnectionString
		if dsn == "" {
			dsn = buildPostgreSQLDSN(config)
		}
		db, err = sqlx.Open(dialect.GetDriverName(), dsn)

	default:
		return nil, nil, fmt.Errorf("unsupported database type: %s", dialect.Name())
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool (MySQL and PostgreSQL only)
	if dialect.Name() != "sqlite" {
		configureConnectionPoolSqlx(db, config)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Apply database-specific settings (needs underlying sql.DB)
	if err := dialect.ApplySettings(db.DB); err != nil {
		return nil, nil, fmt.Errorf("failed to apply database settings: %w", err)
	}

	// Wrap in our DB type
	wrappedDB := &DB{
		DB:      db,
		dialect: dialect,
	}

	return wrappedDB, dialect, nil
}

// WrapSqlxDB wraps an existing sqlx.DB in our DB type with a dialect
// This is useful for creating DB instances in tests or when you have an existing sqlx.DB
func WrapSqlxDB(db *sqlx.DB, dialect Dialect) *DB {
	return &DB{
		DB:      db,
		dialect: dialect,
	}
}

// buildPostgreSQLDSN builds a PostgreSQL connection string
// Format: postgres://username:password@host:port/database?sslmode=disable
func buildPostgreSQLDSN(config *DBConfig) string {
	host := config.Host
	if host == "" {
		host = "localhost"
	}

	port := config.Port
	if port == 0 {
		port = 5432 // Default PostgreSQL port
	}

	database := config.Database
	if database == "" {
		database = "gassigeher"
	}

	sslMode := config.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}

	// Build PostgreSQL connection string
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		config.Username,
		config.Password,
		host,
		port,
		database,
		sslMode,
	)

	return dsn
}

// configureConnectionPool sets connection pool parameters for PostgreSQL
// SQLite doesn't need connection pooling (single file database)
func configureConnectionPool(db *sql.DB, config *DBConfig) {
	maxOpen := config.MaxOpenConns
	if maxOpen == 0 {
		maxOpen = 25 // Default
	}

	maxIdle := config.MaxIdleConns
	if maxIdle == 0 {
		maxIdle = 5 // Default
	}

	maxLifetime := config.ConnMaxLifetime
	if maxLifetime == 0 {
		maxLifetime = 5 // Default: 5 minutes
	}

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(time.Duration(maxLifetime) * time.Minute)
}

// configureConnectionPoolSqlx sets connection pool parameters for sqlx.DB
func configureConnectionPoolSqlx(db *sqlx.DB, config *DBConfig) {
	maxOpen := config.MaxOpenConns
	if maxOpen == 0 {
		maxOpen = 25 // Default
	}

	maxIdle := config.MaxIdleConns
	if maxIdle == 0 {
		maxIdle = 5 // Default
	}

	maxLifetime := config.ConnMaxLifetime
	if maxLifetime == 0 {
		maxLifetime = 5 // Default: 5 minutes
	}

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(time.Duration(maxLifetime) * time.Minute)
}

// RunMigrations runs all database migrations (OLD - backward compatible)
// This function now delegates to the new migration system
func RunMigrations(db *sql.DB) error {
	// Use SQLite dialect by default (for backward compatibility)
	// If you need other databases, use RunMigrationsWithDialect directly
	dialect := &SQLiteDialect{}
	return RunMigrationsWithDialect(db, dialect)
}

const createUsersTable = `
CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  email TEXT UNIQUE,
  phone TEXT,
  password_hash TEXT,
  experience_level TEXT DEFAULT 'green' CHECK(experience_level IN ('green', 'blue', 'orange')),
  is_verified INTEGER DEFAULT 0,
  is_active INTEGER DEFAULT 1,
  is_deleted INTEGER DEFAULT 0,
  verification_token TEXT,
  verification_token_expires TIMESTAMP,
  password_reset_token TEXT,
  password_reset_expires TIMESTAMP,
  profile_photo TEXT,
  anonymous_id TEXT UNIQUE,
  terms_accepted_at TIMESTAMP NOT NULL,
  last_activity_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  deactivated_at TIMESTAMP,
  deactivation_reason TEXT,
  reactivated_at TIMESTAMP,
  deleted_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_last_activity ON users(last_activity_at, is_active);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
`

const createDogsTable = `
CREATE TABLE IF NOT EXISTS dogs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  breed TEXT NOT NULL,
  size TEXT CHECK(size IN ('small', 'medium', 'large')),
  age INTEGER,
  category TEXT CHECK(category IN ('green', 'blue', 'orange')),
  photo TEXT,
  special_needs TEXT,
  pickup_location TEXT,
  walk_route TEXT,
  walk_duration INTEGER,
  special_instructions TEXT,
  default_morning_time TEXT,
  default_evening_time TEXT,
  is_available INTEGER DEFAULT 1,
  unavailable_reason TEXT,
  unavailable_since TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_dogs_available ON dogs(is_available, category);
`

const createBookingsTable = `
CREATE TABLE IF NOT EXISTS bookings (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  dog_id INTEGER NOT NULL,
  date DATE NOT NULL,
  walk_type TEXT CHECK(walk_type IN ('morning', 'evening')),
  scheduled_time TEXT NOT NULL,
  status TEXT DEFAULT 'scheduled' CHECK(status IN ('scheduled', 'completed', 'cancelled')),
  completed_at TIMESTAMP,
  user_notes TEXT,
  admin_cancellation_reason TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (dog_id) REFERENCES dogs(id) ON DELETE CASCADE,
  UNIQUE(dog_id, date, walk_type)
);
`

const createBlockedDatesTable = `
CREATE TABLE IF NOT EXISTS blocked_dates (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  date DATE NOT NULL UNIQUE,
  reason TEXT NOT NULL,
  created_by INTEGER NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (created_by) REFERENCES users(id)
);
`

const createExperienceRequestsTable = `
CREATE TABLE IF NOT EXISTS experience_requests (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  requested_level TEXT CHECK(requested_level IN ('blue', 'orange')),
  status TEXT DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'denied')),
  admin_message TEXT,
  reviewed_by INTEGER,
  reviewed_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (reviewed_by) REFERENCES users(id)
);
`

const createSystemSettingsTable = `
CREATE TABLE IF NOT EXISTS system_settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`

const createReactivationRequestsTable = `
CREATE TABLE IF NOT EXISTS reactivation_requests (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  status TEXT DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'denied')),
  admin_message TEXT,
  reviewed_by INTEGER,
  reviewed_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (reviewed_by) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_reactivation_pending ON reactivation_requests(status, created_at);
`

const insertDefaultSettings = `
INSERT OR IGNORE INTO system_settings (key, value) VALUES
  ('booking_advance_days', '14'),
  ('cancellation_notice_hours', '12'),
  ('auto_deactivation_days', '365');
`

const addPhotoThumbnailColumn = `
-- Add photo_thumbnail column to dogs table
-- Uses ALTER TABLE which will fail if column exists, so we catch the error
-- SQLite doesn't support IF NOT EXISTS for ALTER TABLE ADD COLUMN before version 3.35.0
ALTER TABLE dogs ADD COLUMN photo_thumbnail TEXT;
`
