package config

import (
	"os"
	"testing"
)

// TestGetEnv tests the getEnv helper function
func TestGetEnv(t *testing.T) {
	t.Run("returns env value when set", func(t *testing.T) {
		os.Setenv("TEST_CONFIG_VAR", "test-value")
		defer os.Unsetenv("TEST_CONFIG_VAR")

		result := getEnv("TEST_CONFIG_VAR", "default")
		if result != "test-value" {
			t.Errorf("Expected 'test-value', got '%s'", result)
		}
	})

	t.Run("returns default when env not set", func(t *testing.T) {
		os.Unsetenv("TEST_UNSET_VAR")

		result := getEnv("TEST_UNSET_VAR", "default-value")
		if result != "default-value" {
			t.Errorf("Expected 'default-value', got '%s'", result)
		}
	})

	t.Run("returns default when env is empty string", func(t *testing.T) {
		os.Setenv("TEST_EMPTY_VAR", "")
		defer os.Unsetenv("TEST_EMPTY_VAR")

		result := getEnv("TEST_EMPTY_VAR", "default")
		if result != "default" {
			t.Errorf("Expected 'default', got '%s'", result)
		}
	})
}

// TestGetEnvAsInt tests the getEnvAsInt helper function
func TestGetEnvAsInt(t *testing.T) {
	t.Run("returns int value when valid", func(t *testing.T) {
		os.Setenv("TEST_INT_VAR", "42")
		defer os.Unsetenv("TEST_INT_VAR")

		result := getEnvAsInt("TEST_INT_VAR", 10)
		if result != 42 {
			t.Errorf("Expected 42, got %d", result)
		}
	})

	t.Run("returns default for invalid int", func(t *testing.T) {
		os.Setenv("TEST_INVALID_INT", "not-a-number")
		defer os.Unsetenv("TEST_INVALID_INT")

		result := getEnvAsInt("TEST_INVALID_INT", 99)
		if result != 99 {
			t.Errorf("Expected 99 (default), got %d", result)
		}
	})

	t.Run("returns default when not set", func(t *testing.T) {
		os.Unsetenv("TEST_UNSET_INT")

		result := getEnvAsInt("TEST_UNSET_INT", 25)
		if result != 25 {
			t.Errorf("Expected 25 (default), got %d", result)
		}
	})

	t.Run("handles negative numbers", func(t *testing.T) {
		os.Setenv("TEST_NEGATIVE_INT", "-100")
		defer os.Unsetenv("TEST_NEGATIVE_INT")

		result := getEnvAsInt("TEST_NEGATIVE_INT", 0)
		if result != -100 {
			t.Errorf("Expected -100, got %d", result)
		}
	})

	t.Run("handles zero", func(t *testing.T) {
		os.Setenv("TEST_ZERO_INT", "0")
		defer os.Unsetenv("TEST_ZERO_INT")

		result := getEnvAsInt("TEST_ZERO_INT", 5)
		if result != 0 {
			t.Errorf("Expected 0, got %d", result)
		}
	})
}

// TestGetEnvAsBool tests the getEnvAsBool helper function
func TestGetEnvAsBool(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		setEnv   bool
		want     bool
	}{
		{"true lowercase", "true", true, true},
		{"TRUE uppercase", "TRUE", true, true},
		{"True mixed", "True", true, true},
		{"1", "1", true, true},
		{"yes", "yes", true, true},
		{"YES uppercase", "YES", true, true},
		{"false", "false", true, false},
		{"0", "0", true, false},
		{"no", "no", true, false},
		{"random string", "random", true, false},
		{"empty string returns default", "", true, false}, // default is false
		{"unset returns default", "", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				os.Setenv("TEST_BOOL_VAR", tt.envValue)
				defer os.Unsetenv("TEST_BOOL_VAR")
			} else {
				os.Unsetenv("TEST_BOOL_VAR")
			}

			result := getEnvAsBool("TEST_BOOL_VAR", false)
			if result != tt.want {
				t.Errorf("getEnvAsBool(%q) = %v, want %v", tt.envValue, result, tt.want)
			}
		})
	}

	t.Run("returns true default when unset", func(t *testing.T) {
		os.Unsetenv("TEST_BOOL_DEFAULT_TRUE")

		result := getEnvAsBool("TEST_BOOL_DEFAULT_TRUE", true)
		if result != true {
			t.Errorf("Expected true (default), got false")
		}
	})
}

// TestLoad tests the Load function
func TestLoad(t *testing.T) {
	// Clear any existing env vars that might affect tests
	os.Unsetenv("DB_TYPE")
	os.Unsetenv("PORT")
	os.Unsetenv("JWT_EXPIRATION_HOURS")

	t.Run("returns config with default values", func(t *testing.T) {
		cfg := Load()

		if cfg.DBType != "sqlite" {
			t.Errorf("Expected DBType 'sqlite', got '%s'", cfg.DBType)
		}
		if cfg.Port != "8080" {
			t.Errorf("Expected Port '8080', got '%s'", cfg.Port)
		}
		if cfg.JWTExpirationHours != 24 {
			t.Errorf("Expected JWTExpirationHours 24, got %d", cfg.JWTExpirationHours)
		}
		if cfg.BookingAdvanceDays != 14 {
			t.Errorf("Expected BookingAdvanceDays 14, got %d", cfg.BookingAdvanceDays)
		}
		if cfg.MaxUploadSizeMB != 5 {
			t.Errorf("Expected MaxUploadSizeMB 5, got %d", cfg.MaxUploadSizeMB)
		}
		if cfg.EmailProvider != "gmail" {
			t.Errorf("Expected EmailProvider 'gmail', got '%s'", cfg.EmailProvider)
		}
	})

	t.Run("loads values from environment", func(t *testing.T) {
		os.Setenv("DB_TYPE", "mysql")
		os.Setenv("PORT", "3000")
		os.Setenv("JWT_EXPIRATION_HOURS", "48")
		os.Setenv("SAAS_MODE", "true")
		os.Setenv("USE_S3", "1")
		defer func() {
			os.Unsetenv("DB_TYPE")
			os.Unsetenv("PORT")
			os.Unsetenv("JWT_EXPIRATION_HOURS")
			os.Unsetenv("SAAS_MODE")
			os.Unsetenv("USE_S3")
		}()

		cfg := Load()

		if cfg.DBType != "mysql" {
			t.Errorf("Expected DBType 'mysql', got '%s'", cfg.DBType)
		}
		if cfg.Port != "3000" {
			t.Errorf("Expected Port '3000', got '%s'", cfg.Port)
		}
		if cfg.JWTExpirationHours != 48 {
			t.Errorf("Expected JWTExpirationHours 48, got %d", cfg.JWTExpirationHours)
		}
		if !cfg.SaaSMode {
			t.Error("Expected SaaSMode true, got false")
		}
		if !cfg.UseS3 {
			t.Error("Expected UseS3 true, got false")
		}
	})
}

// TestGetDBConfig tests the GetDBConfig method
func TestGetDBConfig(t *testing.T) {
	t.Run("builds SQLite config correctly", func(t *testing.T) {
		os.Setenv("DB_TYPE", "sqlite")
		os.Setenv("DATABASE_PATH", "/path/to/db.sqlite")
		defer func() {
			os.Unsetenv("DB_TYPE")
			os.Unsetenv("DATABASE_PATH")
		}()

		cfg := Load()
		dbConfig := cfg.GetDBConfig()

		if dbConfig.Type != "sqlite" {
			t.Errorf("Expected Type 'sqlite', got '%s'", dbConfig.Type)
		}
		if dbConfig.Path != "/path/to/db.sqlite" {
			t.Errorf("Expected Path '/path/to/db.sqlite', got '%s'", dbConfig.Path)
		}
	})

	t.Run("builds MySQL config correctly", func(t *testing.T) {
		os.Setenv("DB_TYPE", "mysql")
		os.Setenv("DB_HOST", "localhost")
		os.Setenv("DB_PORT", "3306")
		os.Setenv("DB_NAME", "testdb")
		os.Setenv("DB_USER", "testuser")
		os.Setenv("DB_PASSWORD", "testpass")
		os.Setenv("DB_MAX_OPEN_CONNS", "50")
		os.Setenv("DB_MAX_IDLE_CONNS", "10")
		defer func() {
			os.Unsetenv("DB_TYPE")
			os.Unsetenv("DB_HOST")
			os.Unsetenv("DB_PORT")
			os.Unsetenv("DB_NAME")
			os.Unsetenv("DB_USER")
			os.Unsetenv("DB_PASSWORD")
			os.Unsetenv("DB_MAX_OPEN_CONNS")
			os.Unsetenv("DB_MAX_IDLE_CONNS")
		}()

		cfg := Load()
		dbConfig := cfg.GetDBConfig()

		if dbConfig.Type != "mysql" {
			t.Errorf("Expected Type 'mysql', got '%s'", dbConfig.Type)
		}
		if dbConfig.Host != "localhost" {
			t.Errorf("Expected Host 'localhost', got '%s'", dbConfig.Host)
		}
		if dbConfig.Port != 3306 {
			t.Errorf("Expected Port 3306, got %d", dbConfig.Port)
		}
		if dbConfig.Database != "testdb" {
			t.Errorf("Expected Database 'testdb', got '%s'", dbConfig.Database)
		}
		if dbConfig.Username != "testuser" {
			t.Errorf("Expected Username 'testuser', got '%s'", dbConfig.Username)
		}
		if dbConfig.Password != "testpass" {
			t.Errorf("Expected Password 'testpass', got '%s'", dbConfig.Password)
		}
		if dbConfig.MaxOpenConns != 50 {
			t.Errorf("Expected MaxOpenConns 50, got %d", dbConfig.MaxOpenConns)
		}
		if dbConfig.MaxIdleConns != 10 {
			t.Errorf("Expected MaxIdleConns 10, got %d", dbConfig.MaxIdleConns)
		}
	})

	t.Run("builds PostgreSQL config with SSL mode", func(t *testing.T) {
		os.Setenv("DB_TYPE", "postgres")
		os.Setenv("DB_SSLMODE", "require")
		defer func() {
			os.Unsetenv("DB_TYPE")
			os.Unsetenv("DB_SSLMODE")
		}()

		cfg := Load()
		dbConfig := cfg.GetDBConfig()

		if dbConfig.Type != "postgres" {
			t.Errorf("Expected Type 'postgres', got '%s'", dbConfig.Type)
		}
		if dbConfig.SSLMode != "require" {
			t.Errorf("Expected SSLMode 'require', got '%s'", dbConfig.SSLMode)
		}
	})

	t.Run("uses connection string when provided", func(t *testing.T) {
		connStr := "mysql://user:pass@localhost:3306/testdb"
		os.Setenv("DB_CONNECTION_STRING", connStr)
		defer os.Unsetenv("DB_CONNECTION_STRING")

		cfg := Load()
		dbConfig := cfg.GetDBConfig()

		if dbConfig.ConnectionString != connStr {
			t.Errorf("Expected ConnectionString '%s', got '%s'", connStr, dbConfig.ConnectionString)
		}
	})
}
