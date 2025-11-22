package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"      // CGO-based SQLite (faster, but requires CGO)
	_ "modernc.org/sqlite"               // Pure Go SQLite (slower, but cross-compiles easily)
)

// Note: Migration files (001_*.go, 002_*.go, etc.) are in this package
// and register themselves via init() functions

// DBConfig holds database configuration
type DBConfig struct {
	Type             string // sqlite, mysql, postgres
	ConnectionString string // Full connection string (optional, overrides other fields)

	// SQLite-specific
	Path string

	// MySQL/PostgreSQL-specific
	Host     string
	Port     int
	Database string
	Username string
	Password string
	SSLMode  string // PostgreSQL: disable, require, verify-full

	// Connection pool (MySQL/PostgreSQL only)
	MaxOpenConns    int // Max simultaneous connections
	MaxIdleConns    int // Idle connections to keep
	ConnMaxLifetime int // Max connection age (minutes)
}

// Initialize creates and opens the database connection (OLD - backward compatible)
// Kept for backward compatibility with existing code
// New code should use InitializeWithConfig() for multi-database support
func Initialize(dbPath string) (*sql.DB, error) {
	config := &DBConfig{
		Type: "sqlite",
		Path: dbPath,
	}
	db, _, err := InitializeWithConfig(config)
	return db, err
}

// InitializeWithConfig creates and opens the database connection with full configuration
// Returns both the database connection and the dialect
func InitializeWithConfig(config *DBConfig) (*sql.DB, Dialect, error) {
	var db *sql.DB
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
		db, err = sql.Open(dialect.GetDriverName(), dsn)

	case "mysql":
		dsn := config.ConnectionString
		if dsn == "" {
			dsn = buildMySQLDSN(config)
		}
		db, err = sql.Open(dialect.GetDriverName(), dsn)

	case "postgres":
		dsn := config.ConnectionString
		if dsn == "" {
			dsn = buildPostgreSQLDSN(config)
		}
		db, err = sql.Open(dialect.GetDriverName(), dsn)

	default:
		return nil, nil, fmt.Errorf("unsupported database type: %s", dialect.Name())
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool (MySQL and PostgreSQL only)
	if dialect.Name() != "sqlite" {
		configureConnectionPool(db, config)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Apply database-specific settings
	if err := dialect.ApplySettings(db); err != nil {
		return nil, nil, fmt.Errorf("failed to apply database settings: %w", err)
	}

	return db, dialect, nil
}

// buildMySQLDSN builds a MySQL connection string
// Format: username:password@tcp(host:port)/database?parseTime=true&charset=utf8mb4
func buildMySQLDSN(config *DBConfig) string {
	host := config.Host
	if host == "" {
		host = "localhost"
	}

	port := config.Port
	if port == 0 {
		port = 3306 // Default MySQL port
	}

	database := config.Database
	if database == "" {
		database = "gassigeher"
	}

	// Build DSN
	// parseTime=true is required for scanning time.Time fields
	// charset=utf8mb4 for full Unicode support (including emoji)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci",
		config.Username,
		config.Password,
		host,
		port,
		database,
	)

	return dsn
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

// configureConnectionPool sets connection pool parameters for MySQL and PostgreSQL
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

// RunMigrations runs all database migrations
func RunMigrations(db *sql.DB) error {
	migrations := []string{
		createUsersTable,
		createDogsTable,
		createBookingsTable,
		createBlockedDatesTable,
		createExperienceRequestsTable,
		createSystemSettingsTable,
		createReactivationRequestsTable,
		insertDefaultSettings,
		addPhotoThumbnailColumn,
	}

	for i, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			// Ignore error if column already exists (ALTER TABLE ADD COLUMN error)
			if i == len(migrations)-1 && (err.Error() == "duplicate column name: photo_thumbnail" ||
				err.Error() == "SQLSTATE 42S21: duplicate column name: photo_thumbnail") {
				continue
			}
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}

	return nil
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
