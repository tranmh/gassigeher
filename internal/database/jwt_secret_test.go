package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsWeakJWTSecret(t *testing.T) {
	tests := []struct {
		name   string
		secret string
		want   bool
	}{
		// Weak secrets
		{"empty", "", true},
		{"short", "abc123", true},
		{"common weak - secret", "secret", true},
		{"common weak - password", "password", true},
		{"pattern - change-this", "change-this-in-production", true},
		{"pattern - dev-secret", "dev-secret-do-not-use-in-production-1234567890", true},
		{"pattern - your-secret", "your-secret-key-here", true},
		{"pattern - insecure", "this-is-insecure-key-12345678901234567890", true},
		{"pattern - default", "default-jwt-secret-key-12345678901234567890", true},
		{"pattern - changeme", "changeme12345678901234567890123456", true},
		{"pattern - placeholder", "placeholder-secret-key-1234567890123456", true},

		// Strong secrets
		{"strong base64", "K7gNU3sdo+OL0wNhqoVWhr3g6s1xYv72ol/pe/Unols=", false},
		{"strong random", "xvz1evFS4wEEPTGEFPHBog==AAAAAElFTkSuQmCC", false},
		{"strong 32+ chars", "abcdefghijklmnopqrstuvwxyz123456", false},
		{"strong uuid-like", "550e8400-e29b-41d4-a716-446655440000aa", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsWeakJWTSecret(tt.secret)
			if got != tt.want {
				t.Errorf("IsWeakJWTSecret(%q) = %v, want %v", tt.secret, got, tt.want)
			}
		})
	}
}

func TestGenerateSecureJWTSecret(t *testing.T) {
	// Generate multiple secrets and verify they're unique and strong
	secrets := make(map[string]bool)

	for i := 0; i < 10; i++ {
		secret, err := GenerateSecureJWTSecret()
		if err != nil {
			t.Fatalf("GenerateSecureJWTSecret() error = %v", err)
		}

		// Check length (base64 of 32 bytes = 43-44 chars)
		if len(secret) < 40 {
			t.Errorf("Generated secret too short: %d chars", len(secret))
		}

		// Check it's not weak
		if IsWeakJWTSecret(secret) {
			t.Errorf("Generated secret is detected as weak: %s", secret)
		}

		// Check uniqueness
		if secrets[secret] {
			t.Errorf("Duplicate secret generated: %s", secret)
		}
		secrets[secret] = true
	}
}

func TestUpdateEnvFile(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")

	// Test case 1: Update existing JWT_SECRET
	t.Run("update existing secret", func(t *testing.T) {
		content := `PORT=8080
JWT_SECRET=old-weak-secret
DATABASE_PATH=./test.db
`
		if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
			t.Fatalf("Failed to write test .env: %v", err)
		}

		newSecret := "new-strong-secret-1234567890abcdef"
		if err := UpdateEnvFile(envPath, newSecret); err != nil {
			t.Fatalf("UpdateEnvFile() error = %v", err)
		}

		// Read updated content
		updated, err := os.ReadFile(envPath)
		if err != nil {
			t.Fatalf("Failed to read updated .env: %v", err)
		}

		// Verify new secret is present
		if !strings.Contains(string(updated), "JWT_SECRET="+newSecret) {
			t.Errorf("Updated .env doesn't contain new secret. Got:\n%s", updated)
		}

		// Verify other values preserved
		if !strings.Contains(string(updated), "PORT=8080") {
			t.Error("PORT was not preserved")
		}
		if !strings.Contains(string(updated), "DATABASE_PATH=./test.db") {
			t.Error("DATABASE_PATH was not preserved")
		}

		// Verify backup was created
		files, _ := os.ReadDir(tmpDir)
		var backupFound bool
		for _, f := range files {
			if strings.HasPrefix(f.Name(), ".env.bak.") {
				backupFound = true
				// Verify backup contains old content
				backup, _ := os.ReadFile(filepath.Join(tmpDir, f.Name()))
				if !strings.Contains(string(backup), "old-weak-secret") {
					t.Error("Backup doesn't contain old secret")
				}
			}
		}
		if !backupFound {
			t.Error("No backup file was created")
		}
	})

	// Test case 2: Append JWT_SECRET if not present
	t.Run("append if not present", func(t *testing.T) {
		content := `PORT=8080
DATABASE_PATH=./test.db
`
		if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
			t.Fatalf("Failed to write test .env: %v", err)
		}

		newSecret := "appended-secret-1234567890abcdefgh"
		if err := UpdateEnvFile(envPath, newSecret); err != nil {
			t.Fatalf("UpdateEnvFile() error = %v", err)
		}

		updated, _ := os.ReadFile(envPath)
		if !strings.Contains(string(updated), "JWT_SECRET="+newSecret) {
			t.Errorf("JWT_SECRET was not appended. Got:\n%s", updated)
		}
	})
}

func TestIsFreshInstall(t *testing.T) {
	// Create in-memory SQLite database
	db, dialect, err := InitializeWithConfig(&DBConfig{Type: "sqlite", Path: ":memory:"})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Before migrations - should be fresh install
	t.Run("before migrations", func(t *testing.T) {
		if !IsFreshInstall(db) {
			t.Error("IsFreshInstall() = false, want true (no schema_migrations table)")
		}
	})

	// After migrations - should not be fresh install
	t.Run("after migrations", func(t *testing.T) {
		if err := RunMigrationsWithDialect(db, dialect); err != nil {
			t.Fatalf("Failed to run migrations: %v", err)
		}

		if IsFreshInstall(db) {
			t.Error("IsFreshInstall() = true, want false (schema_migrations exists with entries)")
		}
	})
}

func TestEnforceStrongJWTSecret_FreshInstall(t *testing.T) {
	// Create temp directory for .env
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")

	// Create .env with weak secret
	content := `PORT=8080
JWT_SECRET=dev-secret-weak
`
	if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test .env: %v", err)
	}

	// Test enforcement on fresh install with weak secret
	// Pass isFreshInstall=true to simulate fresh install detection
	newSecret, changed, err := EnforceStrongJWTSecret(true, "dev-secret-weak", envPath)
	if err != nil {
		t.Fatalf("EnforceStrongJWTSecret() error = %v", err)
	}

	if !changed {
		t.Error("Expected secret to be changed on fresh install with weak secret")
	}

	if newSecret == "dev-secret-weak" {
		t.Error("New secret should be different from weak secret")
	}

	if IsWeakJWTSecret(newSecret) {
		t.Errorf("New secret is still weak: %s", newSecret)
	}

	// Verify .env was updated
	updated, _ := os.ReadFile(envPath)
	if !strings.Contains(string(updated), "JWT_SECRET="+newSecret) {
		t.Error(".env was not updated with new secret")
	}
}

func TestEnforceStrongJWTSecret_NotFreshInstall(t *testing.T) {
	// Test with weak secret on existing installation (should only warn, not change)
	// Pass isFreshInstall=false to simulate existing installation
	newSecret, changed, err := EnforceStrongJWTSecret(false, "weak-secret", "")
	if err != nil {
		t.Fatalf("EnforceStrongJWTSecret() error = %v", err)
	}

	if changed {
		t.Error("Should not change secret on existing installation")
	}

	if newSecret != "weak-secret" {
		t.Error("Should return original secret on existing installation")
	}
}

func TestEnforceStrongJWTSecret_AlreadyStrong(t *testing.T) {
	strongSecret := "K7gNU3sdo+OL0wNhqoVWhr3g6s1xYv72ol/pe/Unols="

	// Test with already strong secret on fresh install
	// Pass isFreshInstall=true - even on fresh install, strong secret should not change
	newSecret, changed, err := EnforceStrongJWTSecret(true, strongSecret, "")
	if err != nil {
		t.Fatalf("EnforceStrongJWTSecret() error = %v", err)
	}

	if changed {
		t.Error("Should not change already strong secret")
	}

	if newSecret != strongSecret {
		t.Error("Should return original strong secret unchanged")
	}
}
