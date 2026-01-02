package repository

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

// DBExecutor interface for database operations with cross-database support
// This interface supports both *sql.DB (for backward compatibility) and *sqlx.DB
// When using sqlx.DB, queries are automatically rebound for the correct database
type DBExecutor interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	Prepare(query string) (*sql.Stmt, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	Begin() (*sql.Tx, error)
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)

	// Cross-database helpers
	// InsertReturningID executes INSERT and returns generated ID
	// For PostgreSQL uses RETURNING, for SQLite/MySQL uses LastInsertId
	InsertReturningID(query string, args ...interface{}) (int64, error)

	// GetDialectName returns the database type (sqlite, mysql, postgres)
	GetDialectName() string

	// BoolValue returns the appropriate boolean representation for the database
	// PostgreSQL: true/false, SQLite/MySQL: 1/0
	BoolValue(b bool) interface{}

	// RebindQuery converts ? placeholders to the database-specific format ($1, $2 for postgres)
	// Use this when executing queries within raw *sql.Tx transactions
	RebindQuery(query string) string
}

// DBRebinder interface for components that need to rebind queries
// (e.g., for queries executed within transactions)
type DBRebinder interface {
	RebindQuery(query string) string
}

// DBWithRebind combines DBExecutor with rebinding capability
type DBWithRebind interface {
	DBExecutor
	DBRebinder
}

// RebindingDB wraps sqlx.DB and automatically rebinds queries for cross-database support
// Use this wrapper when passing database to repositories
type RebindingDB struct {
	*sqlx.DB
}

// NewRebindingDB creates a new RebindingDB wrapper
func NewRebindingDB(db *sqlx.DB) *RebindingDB {
	return &RebindingDB{DB: db}
}

// Exec executes a query with automatic placeholder rebinding
func (db *RebindingDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.DB.Exec(db.DB.Rebind(query), args...)
}

// ExecContext executes a query with context and automatic placeholder rebinding
func (db *RebindingDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return db.DB.ExecContext(ctx, db.DB.Rebind(query), args...)
}

// Query executes a query that returns rows with automatic placeholder rebinding
func (db *RebindingDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.DB.Query(db.DB.Rebind(query), args...)
}

// QueryContext executes a query with context that returns rows with automatic placeholder rebinding
func (db *RebindingDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return db.DB.QueryContext(ctx, db.DB.Rebind(query), args...)
}

// QueryRow executes a query that returns at most one row with automatic placeholder rebinding
func (db *RebindingDB) QueryRow(query string, args ...interface{}) *sql.Row {
	return db.DB.QueryRow(db.DB.Rebind(query), args...)
}

// QueryRowContext executes a query with context that returns at most one row with automatic placeholder rebinding
func (db *RebindingDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return db.DB.QueryRowContext(ctx, db.DB.Rebind(query), args...)
}

// Prepare creates a prepared statement with automatic placeholder rebinding
func (db *RebindingDB) Prepare(query string) (*sql.Stmt, error) {
	return db.DB.Prepare(db.DB.Rebind(query))
}

// PrepareContext creates a prepared statement with context and automatic placeholder rebinding
func (db *RebindingDB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return db.DB.PrepareContext(ctx, db.DB.Rebind(query))
}

// SqlDB returns the underlying *sql.DB for compatibility
func (db *RebindingDB) SqlDB() *sql.DB {
	return db.DB.DB
}

// Begin starts a transaction and returns a RebindingTx that auto-rebinds queries
func (db *RebindingDB) Begin() (*sql.Tx, error) {
	return db.DB.Begin()
}

// BeginTx starts a transaction with options and returns a RebindingTx that auto-rebinds queries
func (db *RebindingDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return db.DB.BeginTx(ctx, opts)
}

// RebindQuery returns the rebound query for use in transactions
// Use this when executing queries within a transaction
func (db *RebindingDB) RebindQuery(query string) string {
	return db.DB.Rebind(query)
}

// InsertReturningID executes INSERT and returns generated ID
// RebindingDB assumes SQLite/MySQL behavior (uses LastInsertId)
// For PostgreSQL support, use database.DB instead
func (db *RebindingDB) InsertReturningID(query string, args ...interface{}) (int64, error) {
	result, err := db.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetDialectName returns the database type
// RebindingDB tries to detect from the driver name
func (db *RebindingDB) GetDialectName() string {
	return db.DB.DriverName()
}

// BoolValue returns the appropriate boolean representation for the database
// RebindingDB assumes SQLite/MySQL behavior (uses 1/0)
func (db *RebindingDB) BoolValue(b bool) interface{} {
	if b {
		return 1
	}
	return 0
}
