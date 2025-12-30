package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsWeakMetricsPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		want     bool
	}{
		// Weak passwords
		{"empty", "", true},
		{"short", "abc123", true},
		{"common weak - password", "password", true},
		{"common weak - secret", "secret", true},
		{"pattern - change-this", "change-this-in-production-1234567890", true},
		{"pattern - changeme", "changeme12345678901234567890123456", true},
		{"pattern - default", "default-metrics-password-1234567890", true},
		{"pattern - prometheus", "prometheus-secret-key-1234567890123", true},
		{"pattern - admin", "admin-metrics-password-123456789012", true},
		{"pattern - test", "test-metrics-password-1234567890123", true},

		// Strong passwords
		{"strong base64", "K7gNU3sdo+OL0wNhqoVWhr3g6s1xYv72ol/pe/Unols=", false},
		{"strong random", "xvz1evFS4wEEPTGEFPHBog==AAAAAElFTkSuQmCC", false},
		{"strong 32+ chars", "abcdefghijklmnopqrstuvwxyz123456", false},
		{"strong uuid-like", "550e8400-e29b-41d4-a716-446655440000aa", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsWeakMetricsPassword(tt.password)
			if got != tt.want {
				t.Errorf("IsWeakMetricsPassword(%q) = %v, want %v", tt.password, got, tt.want)
			}
		})
	}
}

func TestGenerateSecureMetricsPassword(t *testing.T) {
	// Generate multiple passwords and verify they're unique and strong
	passwords := make(map[string]bool)

	for i := 0; i < 10; i++ {
		password, err := GenerateSecureMetricsPassword()
		if err != nil {
			t.Fatalf("GenerateSecureMetricsPassword() error = %v", err)
		}

		// Check length (base64 of 32 bytes = 43-44 chars)
		if len(password) < 40 {
			t.Errorf("Generated password too short: %d chars", len(password))
		}

		// Check it's not weak
		if IsWeakMetricsPassword(password) {
			t.Errorf("Generated password is detected as weak: %s", password)
		}

		// Check uniqueness
		if passwords[password] {
			t.Errorf("Duplicate password generated: %s", password)
		}
		passwords[password] = true
	}
}

func TestUpdateEnvFileMetricsPassword(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")

	// Test case 1: Update existing METRICS_PASSWORD
	t.Run("update existing password", func(t *testing.T) {
		content := `PORT=8080
METRICS_PASSWORD=old-weak-password
DATABASE_PATH=./test.db
`
		if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
			t.Fatalf("Failed to write test .env: %v", err)
		}

		newPassword := "new-strong-password-1234567890abcdef"
		if err := UpdateEnvFileMetricsPassword(envPath, newPassword); err != nil {
			t.Fatalf("UpdateEnvFileMetricsPassword() error = %v", err)
		}

		// Read updated content
		updated, err := os.ReadFile(envPath)
		if err != nil {
			t.Fatalf("Failed to read updated .env: %v", err)
		}

		// Verify new password is present
		if !strings.Contains(string(updated), "METRICS_PASSWORD="+newPassword) {
			t.Errorf("Updated .env doesn't contain new password. Got:\n%s", updated)
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
				if !strings.Contains(string(backup), "old-weak-password") {
					t.Error("Backup doesn't contain old password")
				}
			}
		}
		if !backupFound {
			t.Error("No backup file was created")
		}
	})

	// Test case 2: Append METRICS_PASSWORD if not present
	t.Run("append if not present", func(t *testing.T) {
		content := `PORT=8080
DATABASE_PATH=./test.db
`
		if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
			t.Fatalf("Failed to write test .env: %v", err)
		}

		newPassword := "appended-password-1234567890abcdefgh"
		if err := UpdateEnvFileMetricsPassword(envPath, newPassword); err != nil {
			t.Fatalf("UpdateEnvFileMetricsPassword() error = %v", err)
		}

		updated, _ := os.ReadFile(envPath)
		if !strings.Contains(string(updated), "METRICS_PASSWORD="+newPassword) {
			t.Errorf("METRICS_PASSWORD was not appended. Got:\n%s", updated)
		}
	})
}

func TestEnforceStrongMetricsPassword_FreshInstall(t *testing.T) {
	// Create temp directory for .env
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")

	// Create .env with weak password
	content := `PORT=8080
METRICS_PASSWORD=weak-password
`
	if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write test .env: %v", err)
	}

	// Test enforcement on fresh install with weak password
	// Pass isFreshInstall=true to simulate fresh install detection
	newPassword, changed, err := EnforceStrongMetricsPassword(true, "weak-password", envPath)
	if err != nil {
		t.Fatalf("EnforceStrongMetricsPassword() error = %v", err)
	}

	if !changed {
		t.Error("Expected password to be changed on fresh install with weak password")
	}

	if newPassword == "weak-password" {
		t.Error("New password should be different from weak password")
	}

	if IsWeakMetricsPassword(newPassword) {
		t.Errorf("New password is still weak: %s", newPassword)
	}

	// Verify .env was updated
	updated, _ := os.ReadFile(envPath)
	if !strings.Contains(string(updated), "METRICS_PASSWORD="+newPassword) {
		t.Error(".env was not updated with new password")
	}
}

func TestEnforceStrongMetricsPassword_NotFreshInstall(t *testing.T) {
	// Test with weak password on existing installation (should only warn, not change)
	// Pass isFreshInstall=false to simulate existing installation
	newPassword, changed, err := EnforceStrongMetricsPassword(false, "weak-password", "")
	if err != nil {
		t.Fatalf("EnforceStrongMetricsPassword() error = %v", err)
	}

	if changed {
		t.Error("Should not change password on existing installation")
	}

	if newPassword != "weak-password" {
		t.Error("Should return original password on existing installation")
	}
}

func TestEnforceStrongMetricsPassword_AlreadyStrong(t *testing.T) {
	strongPassword := "K7gNU3sdo+OL0wNhqoVWhr3g6s1xYv72ol/pe/Unols="

	// Test with already strong password on fresh install
	// Pass isFreshInstall=true - even on fresh install, strong password should not change
	newPassword, changed, err := EnforceStrongMetricsPassword(true, strongPassword, "")
	if err != nil {
		t.Fatalf("EnforceStrongMetricsPassword() error = %v", err)
	}

	if changed {
		t.Error("Should not change already strong password")
	}

	if newPassword != strongPassword {
		t.Error("Should return original strong password unchanged")
	}
}
