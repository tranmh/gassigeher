package database

import (
	"fmt"
	"strings"
)

// DialectFactory creates database dialects based on type
type DialectFactory struct {
	supportedDialects map[string]func() Dialect
}

// NewDialectFactory creates a new dialect factory
func NewDialectFactory() *DialectFactory {
	factory := &DialectFactory{
		supportedDialects: make(map[string]func() Dialect),
	}

	// Register supported dialects (SQLite and PostgreSQL only)
	factory.Register("sqlite", func() Dialect { return NewSQLiteDialect() })
	factory.Register("postgres", func() Dialect { return NewPostgreSQLDialect() })
	factory.Register("postgresql", func() Dialect { return NewPostgreSQLDialect() }) // Alias

	return factory
}

// Register adds a dialect constructor to the factory
// This allows custom dialects to be registered
func (f *DialectFactory) Register(name string, constructor func() Dialect) {
	f.supportedDialects[strings.ToLower(name)] = constructor
}

// GetDialect returns a dialect for the given database type
// dbType: "sqlite", "postgres", or "postgresql"
// Returns error if database type is not supported
func (f *DialectFactory) GetDialect(dbType string) (Dialect, error) {
	// Normalize to lowercase
	dbType = strings.ToLower(strings.TrimSpace(dbType))

	// Default to SQLite if empty
	if dbType == "" {
		dbType = "sqlite"
	}

	// Get constructor
	constructor, ok := f.supportedDialects[dbType]
	if !ok {
		return nil, fmt.Errorf("unsupported database type: %s (supported: sqlite, postgres)", dbType)
	}

	// Create and return dialect
	return constructor(), nil
}

// GetSupportedDatabases returns list of supported database types
func (f *DialectFactory) GetSupportedDatabases() []string {
	databases := make([]string, 0, len(f.supportedDialects))
	for name := range f.supportedDialects {
		// Skip aliases
		if name != "postgresql" {
			databases = append(databases, name)
		}
	}
	return databases
}

// IsSupported checks if a database type is supported
func (f *DialectFactory) IsSupported(dbType string) bool {
	dbType = strings.ToLower(strings.TrimSpace(dbType))
	_, ok := f.supportedDialects[dbType]
	return ok
}
