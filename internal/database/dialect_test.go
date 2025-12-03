package database

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAllDialects_InterfaceCompliance tests that all dialects implement the interface correctly
func TestAllDialects_InterfaceCompliance(t *testing.T) {
	dialects := []Dialect{
		NewSQLiteDialect(),
		NewMySQLDialect(),
		NewPostgreSQLDialect(),
	}

	for _, dialect := range dialects {
		t.Run(dialect.Name(), func(t *testing.T) {
			// Verify all methods return non-empty strings
			assert.NotEmpty(t, dialect.Name())
			assert.NotEmpty(t, dialect.GetDriverName())
			assert.NotEmpty(t, dialect.GetAutoIncrement())
			assert.NotEmpty(t, dialect.GetBooleanType())
			assert.NotEmpty(t, dialect.GetBooleanDefault(true))
			assert.NotEmpty(t, dialect.GetBooleanDefault(false))
			assert.NotEmpty(t, dialect.GetTextType(255))
			assert.NotEmpty(t, dialect.GetTextType(0))
			assert.NotEmpty(t, dialect.GetTimestampType())
			assert.NotEmpty(t, dialect.GetCurrentDate())
			assert.NotEmpty(t, dialect.GetCurrentTimestamp())
			assert.NotEmpty(t, dialect.GetPlaceholder(1))
			assert.NotEmpty(t, dialect.GetInsertOrIgnore("test_table", []string{"col1"}, "?"))
			assert.NotEmpty(t, dialect.GetAddColumnSyntax("test_table", "new_col", "TEXT"))
		})
	}
}

// TestSQLiteDialect tests SQLite-specific behavior
func TestSQLiteDialect(t *testing.T) {
	dialect := NewSQLiteDialect()

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "sqlite", dialect.Name())
	})

	t.Run("DriverName", func(t *testing.T) {
		assert.Equal(t, "sqlite", dialect.GetDriverName())
	})

	t.Run("AutoIncrement", func(t *testing.T) {
		result := dialect.GetAutoIncrement()
		assert.Equal(t, "INTEGER PRIMARY KEY AUTOINCREMENT", result)
		assert.Contains(t, result, "AUTOINCREMENT")
	})

	t.Run("BooleanType", func(t *testing.T) {
		assert.Equal(t, "INTEGER", dialect.GetBooleanType())
	})

	t.Run("BooleanDefault", func(t *testing.T) {
		assert.Equal(t, "1", dialect.GetBooleanDefault(true))
		assert.Equal(t, "0", dialect.GetBooleanDefault(false))
	})

	t.Run("TextType", func(t *testing.T) {
		assert.Equal(t, "TEXT", dialect.GetTextType(0))
		assert.Equal(t, "TEXT", dialect.GetTextType(255))
		// SQLite ignores maxLength
	})

	t.Run("TimestampType", func(t *testing.T) {
		assert.Equal(t, "TIMESTAMP", dialect.GetTimestampType())
	})

	t.Run("CurrentDate", func(t *testing.T) {
		assert.Equal(t, "date('now')", dialect.GetCurrentDate())
	})

	t.Run("CurrentTimestamp", func(t *testing.T) {
		assert.Equal(t, "CURRENT_TIMESTAMP", dialect.GetCurrentTimestamp())
	})

	t.Run("Placeholder", func(t *testing.T) {
		assert.Equal(t, "?", dialect.GetPlaceholder(1))
		assert.Equal(t, "?", dialect.GetPlaceholder(2))
		assert.Equal(t, "?", dialect.GetPlaceholder(10))
		// SQLite always uses ?
	})

	t.Run("SupportsIfNotExistsColumn", func(t *testing.T) {
		assert.False(t, dialect.SupportsIfNotExistsColumn())
		// Conservative - assume older SQLite
	})

	t.Run("InsertOrIgnore", func(t *testing.T) {
		result := dialect.GetInsertOrIgnore("settings", []string{"key", "value"}, "?, ?")
		assert.Equal(t, "INSERT OR IGNORE INTO settings (key, value) VALUES (?, ?)", result)
		assert.Contains(t, result, "INSERT OR IGNORE")
	})

	t.Run("AddColumnSyntax", func(t *testing.T) {
		result := dialect.GetAddColumnSyntax("dogs", "photo_thumbnail", "TEXT")
		assert.Equal(t, "ALTER TABLE dogs ADD COLUMN photo_thumbnail TEXT", result)
		assert.NotContains(t, result, "IF NOT EXISTS")
		// SQLite (before 3.35) doesn't support IF NOT EXISTS
	})

	t.Run("TableCreationSuffix", func(t *testing.T) {
		assert.Empty(t, dialect.GetTableCreationSuffix())
		// SQLite doesn't need suffix
	})
}

// TestMySQLDialect tests MySQL-specific behavior
func TestMySQLDialect(t *testing.T) {
	dialect := NewMySQLDialect()

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "mysql", dialect.Name())
	})

	t.Run("DriverName", func(t *testing.T) {
		assert.Equal(t, "mysql", dialect.GetDriverName())
	})

	t.Run("AutoIncrement", func(t *testing.T) {
		result := dialect.GetAutoIncrement()
		assert.Equal(t, "INT AUTO_INCREMENT PRIMARY KEY", result)
		assert.Contains(t, result, "AUTO_INCREMENT")
		assert.NotContains(t, result, "AUTOINCREMENT") // SQLite syntax
	})

	t.Run("BooleanType", func(t *testing.T) {
		assert.Equal(t, "TINYINT(1)", dialect.GetBooleanType())
	})

	t.Run("BooleanDefault", func(t *testing.T) {
		assert.Equal(t, "1", dialect.GetBooleanDefault(true))
		assert.Equal(t, "0", dialect.GetBooleanDefault(false))
	})

	t.Run("TextType", func(t *testing.T) {
		assert.Equal(t, "TEXT", dialect.GetTextType(0))
		assert.Equal(t, "VARCHAR(255)", dialect.GetTextType(255))
		assert.Equal(t, "VARCHAR(100)", dialect.GetTextType(100))
	})

	t.Run("TimestampType", func(t *testing.T) {
		assert.Equal(t, "DATETIME", dialect.GetTimestampType())
	})

	t.Run("CurrentDate", func(t *testing.T) {
		assert.Equal(t, "CURDATE()", dialect.GetCurrentDate())
	})

	t.Run("CurrentTimestamp", func(t *testing.T) {
		assert.Equal(t, "CURRENT_TIMESTAMP", dialect.GetCurrentTimestamp())
	})

	t.Run("Placeholder", func(t *testing.T) {
		assert.Equal(t, "?", dialect.GetPlaceholder(1))
		assert.Equal(t, "?", dialect.GetPlaceholder(2))
		// MySQL uses ? like SQLite
	})

	t.Run("SupportsIfNotExistsColumn", func(t *testing.T) {
		assert.False(t, dialect.SupportsIfNotExistsColumn())
		// MySQL doesn't reliably support IF NOT EXISTS for ADD COLUMN
	})

	t.Run("InsertOrIgnore", func(t *testing.T) {
		result := dialect.GetInsertOrIgnore("settings", []string{"key", "value"}, "?, ?")
		assert.Equal(t, "INSERT IGNORE INTO settings (key, value) VALUES (?, ?)", result)
		assert.Contains(t, result, "INSERT IGNORE")
		assert.NotContains(t, result, "OR IGNORE") // SQLite syntax
	})

	t.Run("AddColumnSyntax", func(t *testing.T) {
		result := dialect.GetAddColumnSyntax("dogs", "photo_thumbnail", "TEXT")
		assert.Equal(t, "ALTER TABLE dogs ADD COLUMN photo_thumbnail TEXT", result)
		assert.NotContains(t, result, "IF NOT EXISTS")
	})

	t.Run("TableCreationSuffix", func(t *testing.T) {
		suffix := dialect.GetTableCreationSuffix()
		assert.NotEmpty(t, suffix)
		assert.Contains(t, suffix, "ENGINE=InnoDB")
		assert.Contains(t, suffix, "CHARSET=utf8mb4")
	})
}

// TestPostgreSQLDialect tests PostgreSQL-specific behavior
func TestPostgreSQLDialect(t *testing.T) {
	dialect := NewPostgreSQLDialect()

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "postgres", dialect.Name())
	})

	t.Run("DriverName", func(t *testing.T) {
		assert.Equal(t, "postgres", dialect.GetDriverName())
	})

	t.Run("AutoIncrement", func(t *testing.T) {
		result := dialect.GetAutoIncrement()
		assert.Equal(t, "SERIAL PRIMARY KEY", result)
		assert.Contains(t, result, "SERIAL")
	})

	t.Run("BooleanType", func(t *testing.T) {
		assert.Equal(t, "BOOLEAN", dialect.GetBooleanType())
	})

	t.Run("BooleanDefault", func(t *testing.T) {
		assert.Equal(t, "TRUE", dialect.GetBooleanDefault(true))
		assert.Equal(t, "FALSE", dialect.GetBooleanDefault(false))
	})

	t.Run("TextType", func(t *testing.T) {
		assert.Equal(t, "TEXT", dialect.GetTextType(0))
		assert.Equal(t, "VARCHAR(255)", dialect.GetTextType(255))
		assert.Equal(t, "VARCHAR(100)", dialect.GetTextType(100))
	})

	t.Run("TimestampType", func(t *testing.T) {
		assert.Equal(t, "TIMESTAMP WITH TIME ZONE", dialect.GetTimestampType())
	})

	t.Run("CurrentDate", func(t *testing.T) {
		assert.Equal(t, "CURRENT_DATE", dialect.GetCurrentDate())
	})

	t.Run("CurrentTimestamp", func(t *testing.T) {
		assert.Equal(t, "CURRENT_TIMESTAMP", dialect.GetCurrentTimestamp())
	})

	t.Run("Placeholder", func(t *testing.T) {
		assert.Equal(t, "?", dialect.GetPlaceholder(1))
		assert.Equal(t, "?", dialect.GetPlaceholder(2))
		// We use ? everywhere, pq driver converts to $1, $2
	})

	t.Run("SupportsIfNotExistsColumn", func(t *testing.T) {
		assert.True(t, dialect.SupportsIfNotExistsColumn())
		// PostgreSQL 9.6+ supports this
	})

	t.Run("InsertOrIgnore", func(t *testing.T) {
		result := dialect.GetInsertOrIgnore("settings", []string{"key", "value"}, "?, ?")
		expected := "INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT DO NOTHING"
		assert.Equal(t, expected, result)
		assert.Contains(t, result, "ON CONFLICT DO NOTHING")
	})

	t.Run("AddColumnSyntax", func(t *testing.T) {
		result := dialect.GetAddColumnSyntax("dogs", "photo_thumbnail", "TEXT")
		assert.Equal(t, "ALTER TABLE dogs ADD COLUMN IF NOT EXISTS photo_thumbnail TEXT", result)
		assert.Contains(t, result, "IF NOT EXISTS")
		// PostgreSQL supports IF NOT EXISTS
	})

	t.Run("TableCreationSuffix", func(t *testing.T) {
		assert.Empty(t, dialect.GetTableCreationSuffix())
		// PostgreSQL doesn't need suffix
	})
}

// TestDialectFactory tests the dialect factory
func TestDialectFactory(t *testing.T) {
	factory := NewDialectFactory()

	t.Run("GetDialect_SQLite", func(t *testing.T) {
		dialect, err := factory.GetDialect("sqlite")
		assert.NoError(t, err)
		assert.NotNil(t, dialect)
		assert.Equal(t, "sqlite", dialect.Name())
	})

	t.Run("GetDialect_MySQL", func(t *testing.T) {
		dialect, err := factory.GetDialect("mysql")
		assert.NoError(t, err)
		assert.NotNil(t, dialect)
		assert.Equal(t, "mysql", dialect.Name())
	})

	t.Run("GetDialect_PostgreSQL", func(t *testing.T) {
		dialect, err := factory.GetDialect("postgres")
		assert.NoError(t, err)
		assert.NotNil(t, dialect)
		assert.Equal(t, "postgres", dialect.Name())
	})

	t.Run("GetDialect_PostgreSQL_Alias", func(t *testing.T) {
		dialect, err := factory.GetDialect("postgresql")
		assert.NoError(t, err)
		assert.NotNil(t, dialect)
		assert.Equal(t, "postgres", dialect.Name())
	})

	t.Run("GetDialect_Empty_DefaultsToSQLite", func(t *testing.T) {
		dialect, err := factory.GetDialect("")
		assert.NoError(t, err)
		assert.NotNil(t, dialect)
		assert.Equal(t, "sqlite", dialect.Name())
	})

	t.Run("GetDialect_CaseInsensitive", func(t *testing.T) {
		testCases := []string{"SQLITE", "SQLite", "MySQL", "MYSQL", "Postgres", "POSTGRES"}
		for _, dbType := range testCases {
			dialect, err := factory.GetDialect(dbType)
			assert.NoError(t, err, "Should handle case: %s", dbType)
			assert.NotNil(t, dialect)
		}
	})

	t.Run("GetDialect_Unsupported", func(t *testing.T) {
		dialect, err := factory.GetDialect("oracle")
		assert.Error(t, err)
		assert.Nil(t, dialect)
		assert.Contains(t, err.Error(), "unsupported database type")
	})

	t.Run("GetSupportedDatabases", func(t *testing.T) {
		databases := factory.GetSupportedDatabases()
		assert.Len(t, databases, 3) // sqlite, mysql, postgres (not postgresql alias)
		assert.Contains(t, databases, "sqlite")
		assert.Contains(t, databases, "mysql")
		assert.Contains(t, databases, "postgres")
	})

	t.Run("IsSupported", func(t *testing.T) {
		assert.True(t, factory.IsSupported("sqlite"))
		assert.True(t, factory.IsSupported("mysql"))
		assert.True(t, factory.IsSupported("postgres"))
		assert.True(t, factory.IsSupported("postgresql")) // Alias
		assert.False(t, factory.IsSupported("oracle"))
		assert.False(t, factory.IsSupported("mssql"))
	})
}

// TestDialect_AutoIncrementSyntax tests auto-increment syntax differences
func TestDialect_AutoIncrementSyntax(t *testing.T) {
	testCases := []struct {
		name     string
		dialect  Dialect
		expected string
	}{
		{"SQLite", NewSQLiteDialect(), "INTEGER PRIMARY KEY AUTOINCREMENT"},
		{"MySQL", NewMySQLDialect(), "INT AUTO_INCREMENT PRIMARY KEY"},
		{"PostgreSQL", NewPostgreSQLDialect(), "SERIAL PRIMARY KEY"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.dialect.GetAutoIncrement()
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestDialect_BooleanHandling tests boolean type differences
func TestDialect_BooleanHandling(t *testing.T) {
	testCases := []struct {
		name         string
		dialect      Dialect
		expectedType string
		expectedTrue string
		expectedFalse string
	}{
		{"SQLite", NewSQLiteDialect(), "INTEGER", "1", "0"},
		{"MySQL", NewMySQLDialect(), "TINYINT(1)", "1", "0"},
		{"PostgreSQL", NewPostgreSQLDialect(), "BOOLEAN", "TRUE", "FALSE"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedType, tc.dialect.GetBooleanType())
			assert.Equal(t, tc.expectedTrue, tc.dialect.GetBooleanDefault(true))
			assert.Equal(t, tc.expectedFalse, tc.dialect.GetBooleanDefault(false))
		})
	}
}

// TestDialect_TextTypeHandling tests text type with size limits
func TestDialect_TextTypeHandling(t *testing.T) {
	testCases := []struct {
		name             string
		dialect          Dialect
		maxLength        int
		expected         string
		description      string
	}{
		{"SQLite_Unlimited", NewSQLiteDialect(), 0, "TEXT", "Unlimited text"},
		{"SQLite_Sized", NewSQLiteDialect(), 255, "TEXT", "SQLite ignores size"},
		{"MySQL_Unlimited", NewMySQLDialect(), 0, "TEXT", "Unlimited text"},
		{"MySQL_Sized", NewMySQLDialect(), 255, "VARCHAR(255)", "Sized text"},
		{"PostgreSQL_Unlimited", NewPostgreSQLDialect(), 0, "TEXT", "Unlimited text"},
		{"PostgreSQL_Sized", NewPostgreSQLDialect(), 255, "VARCHAR(255)", "Sized text"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.dialect.GetTextType(tc.maxLength)
			assert.Equal(t, tc.expected, result, tc.description)
		})
	}
}

// TestDialect_DateTimeFunctions tests current date/timestamp expressions
func TestDialect_DateTimeFunctions(t *testing.T) {
	testCases := []struct {
		name              string
		dialect           Dialect
		expectedDate      string
		expectedTimestamp string
	}{
		{"SQLite", NewSQLiteDialect(), "date('now')", "CURRENT_TIMESTAMP"},
		{"MySQL", NewMySQLDialect(), "CURDATE()", "CURRENT_TIMESTAMP"},
		{"PostgreSQL", NewPostgreSQLDialect(), "CURRENT_DATE", "CURRENT_TIMESTAMP"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedDate, tc.dialect.GetCurrentDate())
			assert.Equal(t, tc.expectedTimestamp, tc.dialect.GetCurrentTimestamp())
		})
	}
}

// TestDialect_PlaceholderSyntax tests placeholder generation
func TestDialect_PlaceholderSyntax(t *testing.T) {
	testCases := []struct {
		name     string
		dialect  Dialect
		position int
		expected string
	}{
		{"SQLite_Pos1", NewSQLiteDialect(), 1, "?"},
		{"SQLite_Pos5", NewSQLiteDialect(), 5, "?"},
		{"MySQL_Pos1", NewMySQLDialect(), 1, "?"},
		{"MySQL_Pos5", NewMySQLDialect(), 5, "?"},
		{"PostgreSQL_Pos1", NewPostgreSQLDialect(), 1, "?"},
		{"PostgreSQL_Pos5", NewPostgreSQLDialect(), 5, "?"},
		// Note: We use ? everywhere, drivers handle conversion
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.dialect.GetPlaceholder(tc.position)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestDialect_InsertOrIgnoreSyntax tests insert-or-ignore differences
func TestDialect_InsertOrIgnoreSyntax(t *testing.T) {
	columns := []string{"key", "value"}
	placeholders := "?, ?"

	testCases := []struct {
		name     string
		dialect  Dialect
		contains []string
	}{
		{"SQLite", NewSQLiteDialect(), []string{"INSERT OR IGNORE"}},
		{"MySQL", NewMySQLDialect(), []string{"INSERT IGNORE"}},
		{"PostgreSQL", NewPostgreSQLDialect(), []string{"INSERT INTO", "ON CONFLICT DO NOTHING"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.dialect.GetInsertOrIgnore("settings", columns, placeholders)
			for _, expectedFragment := range tc.contains {
				assert.Contains(t, result, expectedFragment)
			}
			// Verify it's valid SQL structure
			assert.Contains(t, result, "INSERT")
			assert.Contains(t, result, "settings")
			assert.Contains(t, result, "key, value")
			assert.Contains(t, result, "VALUES")
		})
	}
}

// TestDialect_AddColumnSyntax tests ADD COLUMN differences
func TestDialect_AddColumnSyntax(t *testing.T) {
	testCases := []struct {
		name            string
		dialect         Dialect
		supportsIfNotExists bool
	}{
		{"SQLite", NewSQLiteDialect(), false},
		{"MySQL", NewMySQLDialect(), false},
		{"PostgreSQL", NewPostgreSQLDialect(), true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.dialect.GetAddColumnSyntax("test_table", "new_column", "TEXT")

			assert.Contains(t, result, "ALTER TABLE")
			assert.Contains(t, result, "ADD COLUMN")
			assert.Contains(t, result, "test_table")
			assert.Contains(t, result, "new_column")
			assert.Contains(t, result, "TEXT")

			if tc.supportsIfNotExists {
				assert.Contains(t, result, "IF NOT EXISTS")
			} else {
				assert.NotContains(t, result, "IF NOT EXISTS")
			}
		})
	}
}

// TestDialect_TableCreationSuffix tests table creation suffixes
func TestDialect_TableCreationSuffix(t *testing.T) {
	testCases := []struct {
		name     string
		dialect  Dialect
		isEmpty  bool
		contains []string
	}{
		{"SQLite", NewSQLiteDialect(), true, nil},
		{"MySQL", NewMySQLDialect(), false, []string{"ENGINE=InnoDB", "CHARSET=utf8mb4"}},
		{"PostgreSQL", NewPostgreSQLDialect(), true, nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.dialect.GetTableCreationSuffix()

			if tc.isEmpty {
				assert.Empty(t, result)
			} else {
				assert.NotEmpty(t, result)
				for _, fragment := range tc.contains {
					assert.Contains(t, result, fragment)
				}
			}
		})
	}
}

// TestDialect_Consistency tests that all dialects are consistent
func TestDialect_Consistency(t *testing.T) {
	dialects := []Dialect{
		NewSQLiteDialect(),
		NewMySQLDialect(),
		NewPostgreSQLDialect(),
	}

	t.Run("AllDialectsHaveUniqueName", func(t *testing.T) {
		names := make(map[string]bool)
		for _, dialect := range dialects {
			name := dialect.Name()
			assert.False(t, names[name], "Duplicate dialect name: %s", name)
			names[name] = true
		}
	})

	t.Run("AllPlaceholdersNonEmpty", func(t *testing.T) {
		for _, dialect := range dialects {
			for i := 1; i <= 5; i++ {
				placeholder := dialect.GetPlaceholder(i)
				assert.NotEmpty(t, placeholder,
					"Dialect %s placeholder %d is empty", dialect.Name(), i)
			}
		}
	})

	t.Run("AllTypesNonEmpty", func(t *testing.T) {
		for _, dialect := range dialects {
			assert.NotEmpty(t, dialect.GetAutoIncrement())
			assert.NotEmpty(t, dialect.GetBooleanType())
			assert.NotEmpty(t, dialect.GetTextType(0))
			assert.NotEmpty(t, dialect.GetTextType(255))
			assert.NotEmpty(t, dialect.GetTimestampType())
		}
	})
}

// TestDialect_RealWorldQueries tests with realistic query patterns
func TestDialect_RealWorldQueries(t *testing.T) {
	dialects := []Dialect{
		NewSQLiteDialect(),
		NewMySQLDialect(),
		NewPostgreSQLDialect(),
	}

	for _, dialect := range dialects {
		t.Run(dialect.Name()+"_CreateTableQuery", func(t *testing.T) {
			// Simulate creating a users table
			query := fmt.Sprintf(`
				CREATE TABLE IF NOT EXISTS users (
					id %s,
					name %s NOT NULL,
					email %s UNIQUE,
					is_active %s DEFAULT %s,
					created_at %s DEFAULT %s
				)%s`,
				dialect.GetAutoIncrement(),
				dialect.GetTextType(255),
				dialect.GetTextType(255),
				dialect.GetBooleanType(),
				dialect.GetBooleanDefault(true),
				dialect.GetTimestampType(),
				dialect.GetCurrentTimestamp(),
				dialect.GetTableCreationSuffix(),
			)

			// Verify query contains expected keywords
			assert.Contains(t, query, "CREATE TABLE")
			assert.Contains(t, query, "users")
			assert.Contains(t, query, "id")
			assert.Contains(t, query, "PRIMARY KEY")
			assert.NotEmpty(t, query)
		})

		t.Run(dialect.Name()+"_InsertOrIgnoreQuery", func(t *testing.T) {
			query := dialect.GetInsertOrIgnore("settings",
				[]string{"key", "value"},
				"?, ?")

			assert.Contains(t, query, "INSERT")
			assert.Contains(t, query, "settings")
			assert.Contains(t, query, "VALUES")
			assert.NotEmpty(t, query)
		})
	}
}

// TestDialect_BackwardCompatibility tests that changes don't break existing code
func TestDialect_BackwardCompatibility(t *testing.T) {
	t.Run("GetDialect_Function_StillWorks", func(t *testing.T) {
		// Old helper function should still work
		dialect := GetDialect("sqlite")
		assert.NotNil(t, dialect)
		assert.Equal(t, "sqlite", dialect.Name())

		// Unknown type should fallback to SQLite
		dialect = GetDialect("unknown")
		assert.NotNil(t, dialect)
		assert.Equal(t, "sqlite", dialect.Name())
	})
}
