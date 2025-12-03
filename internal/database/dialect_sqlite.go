package database

import (
	"database/sql"
	"fmt"
	"strings"
)

// SQLiteDialect implements the Dialect interface for SQLite
type SQLiteDialect struct{}

// NewSQLiteDialect creates a new SQLite dialect
func NewSQLiteDialect() *SQLiteDialect {
	return &SQLiteDialect{}
}

// Name returns the database name
func (d *SQLiteDialect) Name() string {
	return "sqlite"
}

// GetDriverName returns the Go driver name for sql.Open()
func (d *SQLiteDialect) GetDriverName() string {
	return "sqlite"
}

// GetAutoIncrement returns the auto-increment syntax for primary keys
func (d *SQLiteDialect) GetAutoIncrement() string {
	return "INTEGER PRIMARY KEY AUTOINCREMENT"
}

// GetBooleanType returns the boolean column type
// SQLite uses INTEGER to store booleans (0 = false, 1 = true)
func (d *SQLiteDialect) GetBooleanType() string {
	return "INTEGER"
}

// GetBooleanDefault returns the default value for a boolean
func (d *SQLiteDialect) GetBooleanDefault(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

// GetTextType returns the text column type
// SQLite's TEXT type is flexible and doesn't need size limits
// maxLength is ignored for SQLite
func (d *SQLiteDialect) GetTextType(maxLength int) string {
	return "TEXT"
}

// GetTimestampType returns the timestamp column type
func (d *SQLiteDialect) GetTimestampType() string {
	return "TIMESTAMP"
}

// GetCurrentDate returns SQL expression for current date
func (d *SQLiteDialect) GetCurrentDate() string {
	return "date('now')"
}

// GetCurrentTimestamp returns SQL expression for current timestamp
func (d *SQLiteDialect) GetCurrentTimestamp() string {
	return "CURRENT_TIMESTAMP"
}

// GetPlaceholder returns the placeholder syntax
// SQLite uses ? for all parameters
func (d *SQLiteDialect) GetPlaceholder(position int) string {
	return "?"
}

// SupportsIfNotExistsColumn returns whether database supports
// ALTER TABLE ADD COLUMN IF NOT EXISTS
// SQLite before 3.35.0 does not support this
func (d *SQLiteDialect) SupportsIfNotExistsColumn() bool {
	return false // Conservative - assume older SQLite version
}

// GetInsertOrIgnore returns the SQL for insert-or-ignore semantics
func (d *SQLiteDialect) GetInsertOrIgnore(tableName string, columns []string, placeholders string) string {
	columnList := strings.Join(columns, ", ")
	return fmt.Sprintf("INSERT OR IGNORE INTO %s (%s) VALUES (%s)",
		tableName, columnList, placeholders)
}

// GetAddColumnSyntax returns SQL for adding a column
// SQLite doesn't support IF NOT EXISTS before 3.35.0
// Caller must handle duplicate column error
func (d *SQLiteDialect) GetAddColumnSyntax(tableName, columnName, columnType string) string {
	return fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s",
		tableName, columnName, columnType)
}

// ApplySettings applies SQLite-specific settings
// Enables foreign key constraints (disabled by default in SQLite)
func (d *SQLiteDialect) ApplySettings(db *sql.DB) error {
	// Enable foreign keys (critical for referential integrity)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Optional: Set journal mode to WAL for better concurrency
	// Uncomment if needed:
	// if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
	//     return fmt.Errorf("failed to set journal mode: %w", err)
	// }

	return nil
}

// GetTableCreationSuffix returns any suffix needed after table definition
// SQLite doesn't need any suffix
func (d *SQLiteDialect) GetTableCreationSuffix() string {
	return ""
}

// QuoteIdentifier returns the quoted identifier
// SQLite is flexible with quotes, but backticks are not standard
// Using double quotes for consistency with PostgreSQL
func (d *SQLiteDialect) QuoteIdentifier(identifier string) string {
	// Generally not needed in our queries
	// Return as-is unless identifier is a reserved word
	return identifier
}

// ConvertGoTime returns SQL expression to convert Go time.Time
// SQLite handles Go time.Time automatically via driver
func (d *SQLiteDialect) ConvertGoTime(goTime string) string {
	return goTime // Driver handles conversion
}
