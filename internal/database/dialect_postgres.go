package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

// PostgreSQLDialect implements the Dialect interface for PostgreSQL
type PostgreSQLDialect struct{}

// NewPostgreSQLDialect creates a new PostgreSQL dialect
func NewPostgreSQLDialect() *PostgreSQLDialect {
	return &PostgreSQLDialect{}
}

// Name returns the database name
func (d *PostgreSQLDialect) Name() string {
	return "postgres"
}

// GetDriverName returns the Go driver name for sql.Open()
func (d *PostgreSQLDialect) GetDriverName() string {
	return "postgres"
}

// GetAutoIncrement returns the auto-increment syntax for primary keys
// PostgreSQL uses SERIAL (which is shorthand for INTEGER with auto-sequence)
func (d *PostgreSQLDialect) GetAutoIncrement() string {
	return "SERIAL PRIMARY KEY"
}

// GetBooleanType returns the boolean column type
// PostgreSQL has a native BOOLEAN type
func (d *PostgreSQLDialect) GetBooleanType() string {
	return "BOOLEAN"
}

// GetBooleanDefault returns the default value for a boolean
func (d *PostgreSQLDialect) GetBooleanDefault(value bool) string {
	if value {
		return "TRUE"
	}
	return "FALSE"
}

// GetTextType returns the text column type
// PostgreSQL supports both VARCHAR(n) and TEXT
// TEXT has no length limit and is efficient in PostgreSQL
func (d *PostgreSQLDialect) GetTextType(maxLength int) string {
	if maxLength == 0 {
		// Unlimited text
		return "TEXT"
	}
	// Sized text (for indexed fields)
	return fmt.Sprintf("VARCHAR(%d)", maxLength)
}

// GetTimestampType returns the timestamp column type
// PostgreSQL TIMESTAMP WITH TIME ZONE is recommended for UTC storage
func (d *PostgreSQLDialect) GetTimestampType() string {
	return "TIMESTAMP WITH TIME ZONE"
}

// GetCurrentDate returns SQL expression for current date
func (d *PostgreSQLDialect) GetCurrentDate() string {
	return "CURRENT_DATE"
}

// GetCurrentTimestamp returns SQL expression for current timestamp
func (d *PostgreSQLDialect) GetCurrentTimestamp() string {
	return "CURRENT_TIMESTAMP"
}

// GetPlaceholder returns the placeholder syntax
// PostgreSQL uses $1, $2, $3, ... for parameters
// HOWEVER: The lib/pq driver when used with database/sql package
// can handle ? placeholders and convert them automatically!
// For maximum compatibility, we keep using ? everywhere
func (d *PostgreSQLDialect) GetPlaceholder(position int) string {
	// Note: We use ? everywhere in our queries, and the pq driver
	// handles the conversion when using database/sql.
	// If we were using pq directly, we'd need $1, $2, etc.
	return "?"
}

// SupportsIfNotExistsColumn returns whether database supports
// ALTER TABLE ADD COLUMN IF NOT EXISTS
// PostgreSQL 9.6+ supports this syntax
func (d *PostgreSQLDialect) SupportsIfNotExistsColumn() bool {
	return true // PostgreSQL 9.6+ (we require 12+)
}

// GetInsertOrIgnore returns the SQL for insert-or-ignore semantics
// PostgreSQL uses ON CONFLICT DO NOTHING (requires unique constraint or primary key)
func (d *PostgreSQLDialect) GetInsertOrIgnore(tableName string, columns []string, placeholders string) string {
	columnList := strings.Join(columns, ", ")
	// For system_settings, the key column is PRIMARY KEY, so ON CONFLICT works
	// If no unique constraint exists, this would need the conflict target specified
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON CONFLICT DO NOTHING",
		tableName, columnList, placeholders)
}

// GetAddColumnSyntax returns SQL for adding a column
// PostgreSQL supports IF NOT EXISTS for ADD COLUMN (9.6+)
func (d *PostgreSQLDialect) GetAddColumnSyntax(tableName, columnName, columnType string) string {
	return fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS %s %s",
		tableName, columnName, columnType)
}

// ApplySettings applies PostgreSQL-specific settings
func (d *PostgreSQLDialect) ApplySettings(db *sql.DB) error {
	// Set timezone to UTC for consistency
	if _, err := db.Exec("SET TIME ZONE 'UTC'"); err != nil {
		return fmt.Errorf("failed to set timezone: %w", err)
	}

	// Set client encoding to UTF8
	if _, err := db.Exec("SET client_encoding = 'UTF8'"); err != nil {
		return fmt.Errorf("failed to set encoding: %w", err)
	}

	// Set datestyle for consistent date handling
	if _, err := db.Exec("SET datestyle = 'ISO, YMD'"); err != nil {
		// Non-fatal
		log.Printf("Warning: Failed to set datestyle: %v", err)
	}

	return nil
}

// GetTableCreationSuffix returns PostgreSQL table creation suffix
// PostgreSQL doesn't need special suffixes (storage is configurable elsewhere)
func (d *PostgreSQLDialect) GetTableCreationSuffix() string {
	return ""
}

// QuoteIdentifier returns the quoted identifier using double quotes
// PostgreSQL uses double quotes for identifiers
func (d *PostgreSQLDialect) QuoteIdentifier(identifier string) string {
	// Generally not needed in our queries
	// Return as-is unless identifier is a reserved word
	return identifier
}

// ConvertGoTime returns SQL expression to convert Go time.Time
// PostgreSQL driver handles Go time.Time automatically
func (d *PostgreSQLDialect) ConvertGoTime(goTime string) string {
	return goTime // Driver handles conversion
}
