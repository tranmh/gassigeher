package database

import "database/sql"

// Dialect defines database-specific SQL syntax and behaviors
// This interface allows the application to work with SQLite and PostgreSQL
// using a single codebase with database-specific adaptations where needed
type Dialect interface {
	// Name returns the database name (sqlite, postgres)
	Name() string

	// GetDriverName returns the Go driver name for sql.Open()
	GetDriverName() string

	// GetAutoIncrement returns the auto-increment syntax for primary keys
	// SQLite: "INTEGER PRIMARY KEY AUTOINCREMENT"
	// PostgreSQL: "SERIAL PRIMARY KEY"
	GetAutoIncrement() string

	// GetBooleanType returns the boolean column type
	// SQLite: "INTEGER" (stores 0/1)
	// PostgreSQL: "BOOLEAN" (stores true/false)
	GetBooleanType() string

	// GetBooleanDefault returns the default value syntax for boolean false
	// SQLite: "0"
	// PostgreSQL: "FALSE"
	GetBooleanDefault(value bool) string

	// GetTextType returns the text column type with optional max length
	// maxLength = 0 means unlimited TEXT
	// maxLength > 0 means VARCHAR with size limit
	// SQLite: "TEXT" (ignores maxLength)
	// PostgreSQL: "VARCHAR(n)" or "TEXT"
	GetTextType(maxLength int) string

	// GetTimestampType returns the timestamp column type
	// SQLite: "TIMESTAMP"
	// PostgreSQL: "TIMESTAMP WITH TIME ZONE"
	GetTimestampType() string

	// GetCurrentDate returns SQL expression for current date
	// Use this in DEFAULT clauses or queries
	// SQLite: "date('now')"
	// PostgreSQL: "CURRENT_DATE"
	GetCurrentDate() string

	// GetCurrentTimestamp returns SQL expression for current timestamp
	// All databases support CURRENT_TIMESTAMP, but this allows customization
	GetCurrentTimestamp() string

	// GetPlaceholder returns the placeholder syntax for parameterized queries
	// position is 1-indexed (first parameter is position 1)
	// SQLite: "?" (ignores position)
	// PostgreSQL: "$1", "$2", "$3", etc. (uses position)
	// Note: Go's database/sql with lib/pq driver actually handles ? → $n conversion
	// So we can keep using ? everywhere!
	GetPlaceholder(position int) string

	// SupportsIfNotExistsColumn returns true if database supports
	// ALTER TABLE ADD COLUMN IF NOT EXISTS syntax
	// SQLite: false (before 3.35.0)
	// PostgreSQL: true (9.6+)
	SupportsIfNotExistsColumn() bool

	// GetInsertOrIgnore returns the SQL for insert-or-ignore semantics
	// Used for idempotent inserts (e.g., default settings)
	// tableName: name of table
	// columns: column names
	// placeholders: already-generated placeholder string
	// SQLite: "INSERT OR IGNORE INTO table (cols) VALUES (placeholders)"
	// PostgreSQL: "INSERT INTO table (cols) VALUES (placeholders) ON CONFLICT DO NOTHING"
	GetInsertOrIgnore(tableName string, columns []string, placeholders string) string

	// GetAddColumnSyntax returns SQL for adding a column
	// Handles IF NOT EXISTS for databases that support it
	// Returns SQL that's idempotent (safe to run multiple times)
	GetAddColumnSyntax(tableName, columnName, columnType string) string

	// ApplySettings applies database-specific settings after connection
	// SQLite: PRAGMA foreign_keys = ON
	// PostgreSQL: May set timezone, etc.
	ApplySettings(db *sql.DB) error

	// GetTableCreationSuffix returns any suffix needed after table definition
	// SQLite: "" (no suffix)
	// PostgreSQL: "" (no suffix needed)
	GetTableCreationSuffix() string

	// QuoteIdentifier returns the quoted identifier for table/column names
	// SQLite: "name" or `name` or name (flexible)
	// PostgreSQL: "name" (double quotes)
	// Generally not needed since we don't use reserved words
	QuoteIdentifier(identifier string) string

	// ConvertGoTime returns SQL expression to convert Go time.Time to database format
	// Used when inserting timestamps from Go code
	// All databases handle this via driver, but allows customization
	ConvertGoTime(goTime string) string
}

// GetDialect returns the appropriate dialect for a database type
// Deprecated: Use NewDialectFactory().GetDialect() instead
// Kept for backward compatibility
func GetDialect(dbType string) Dialect {
	factory := NewDialectFactory()
	dialect, err := factory.GetDialect(dbType)
	if err != nil {
		// Fallback to SQLite for backward compatibility
		return NewSQLiteDialect()
	}
	return dialect
}
