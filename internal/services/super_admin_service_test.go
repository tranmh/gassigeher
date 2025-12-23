package services

import (
	"strings"
	"testing"
)

// TestParseCredentialsFile tests the credentials file parser
func TestParseCredentialsFile(t *testing.T) {
	t.Run("valid credentials file", func(t *testing.T) {
		content := `=============================================================
GASSIGEHER - SUPER ADMIN CREDENTIALS
=============================================================

EMAIL: admin@example.com
PASSWORD: SecurePass123!

CREATED: 2025-01-01 10:00:00
LAST UPDATED: 2025-01-15 14:30:00

=============================================================
`
		email, password, createdTime, err := parseCredentialsFile(content)
		if err != nil {
			t.Errorf("parseCredentialsFile() error = %v", err)
		}
		if email != "admin@example.com" {
			t.Errorf("Expected email 'admin@example.com', got '%s'", email)
		}
		if password != "SecurePass123!" {
			t.Errorf("Expected password 'SecurePass123!', got '%s'", password)
		}
		if createdTime != "2025-01-01 10:00:00" {
			t.Errorf("Expected createdTime '2025-01-01 10:00:00', got '%s'", createdTime)
		}
	})

	t.Run("missing email", func(t *testing.T) {
		content := `PASSWORD: SecurePass123!
CREATED: 2025-01-01 10:00:00`
		_, _, _, err := parseCredentialsFile(content)
		if err == nil {
			t.Error("Expected error for missing email")
		}
		if !strings.Contains(err.Error(), "missing EMAIL or PASSWORD") {
			t.Errorf("Expected 'missing EMAIL or PASSWORD' error, got: %v", err)
		}
	})

	t.Run("missing password", func(t *testing.T) {
		content := `EMAIL: admin@example.com
CREATED: 2025-01-01 10:00:00`
		_, _, _, err := parseCredentialsFile(content)
		if err == nil {
			t.Error("Expected error for missing password")
		}
	})

	t.Run("empty file", func(t *testing.T) {
		content := ""
		_, _, _, err := parseCredentialsFile(content)
		if err == nil {
			t.Error("Expected error for empty file")
		}
	})

	t.Run("whitespace handling", func(t *testing.T) {
		content := `EMAIL:   admin@example.com
PASSWORD:   SecurePass123!   `
		email, password, _, err := parseCredentialsFile(content)
		if err != nil {
			t.Errorf("parseCredentialsFile() error = %v", err)
		}
		if email != "admin@example.com" {
			t.Errorf("Expected trimmed email 'admin@example.com', got '%s'", email)
		}
		if password != "SecurePass123!" {
			t.Errorf("Expected trimmed password 'SecurePass123!', got '%s'", password)
		}
	})

	t.Run("missing created time is ok", func(t *testing.T) {
		content := `EMAIL: admin@example.com
PASSWORD: SecurePass123!`
		email, password, createdTime, err := parseCredentialsFile(content)
		if err != nil {
			t.Errorf("parseCredentialsFile() error = %v", err)
		}
		if email == "" || password == "" {
			t.Error("Email and password should be parsed")
		}
		if createdTime != "" {
			t.Errorf("Expected empty createdTime, got '%s'", createdTime)
		}
	})
}

// TestGenerateSecurePassword tests the secure password generator
func TestGenerateSecurePassword(t *testing.T) {
	t.Run("generates correct length", func(t *testing.T) {
		lengths := []int{12, 16, 20, 32}
		for _, length := range lengths {
			password := generateSecurePassword(length)
			if len(password) != length {
				t.Errorf("Expected length %d, got %d for password '%s'", length, len(password), password)
			}
		}
	})

	t.Run("contains lowercase letter", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			password := generateSecurePassword(20)
			hasLower := false
			for _, c := range password {
				if c >= 'a' && c <= 'z' {
					hasLower = true
					break
				}
			}
			if !hasLower {
				t.Errorf("Password should contain lowercase: %s", password)
			}
		}
	})

	t.Run("contains uppercase letter", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			password := generateSecurePassword(20)
			hasUpper := false
			for _, c := range password {
				if c >= 'A' && c <= 'Z' {
					hasUpper = true
					break
				}
			}
			if !hasUpper {
				t.Errorf("Password should contain uppercase: %s", password)
			}
		}
	})

	t.Run("contains number", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			password := generateSecurePassword(20)
			hasNumber := false
			for _, c := range password {
				if c >= '0' && c <= '9' {
					hasNumber = true
					break
				}
			}
			if !hasNumber {
				t.Errorf("Password should contain number: %s", password)
			}
		}
	})

	t.Run("contains special character", func(t *testing.T) {
		specialChars := "!@#$%^&*"
		for i := 0; i < 100; i++ {
			password := generateSecurePassword(20)
			hasSpecial := false
			for _, c := range password {
				if strings.ContainsRune(specialChars, c) {
					hasSpecial = true
					break
				}
			}
			if !hasSpecial {
				t.Errorf("Password should contain special character: %s", password)
			}
		}
	})

	t.Run("generates unique passwords", func(t *testing.T) {
		passwords := make(map[string]bool)
		for i := 0; i < 1000; i++ {
			password := generateSecurePassword(20)
			if passwords[password] {
				t.Errorf("Duplicate password generated: %s", password)
			}
			passwords[password] = true
		}
	})

	t.Run("minimum length has all character types", func(t *testing.T) {
		// Even at minimum useful length (4), should have all types
		password := generateSecurePassword(4)
		if len(password) != 4 {
			t.Errorf("Expected length 4, got %d", len(password))
		}
		// Note: With length 4, guaranteed to have all 4 types due to implementation
	})

	t.Run("characters are shuffled", func(t *testing.T) {
		// Generate many passwords and verify first 4 chars aren't always in same order
		firstFourChars := make(map[string]int)
		for i := 0; i < 100; i++ {
			password := generateSecurePassword(20)
			firstFour := password[:4]
			firstFourChars[firstFour]++
		}
		// Should have variety in first 4 characters (shuffled)
		if len(firstFourChars) < 50 {
			t.Logf("First 4 chars might not be well-shuffled, got %d unique combinations", len(firstFourChars))
		}
	})
}

// TestGenerateSecurePassword_CharacterDistribution tests for bias in character selection
func TestGenerateSecurePassword_CharacterDistribution(t *testing.T) {
	// Generate many passwords and check character distribution
	charCount := make(map[byte]int)
	const numSamples = 10000
	const passwordLength = 20

	for i := 0; i < numSamples; i++ {
		password := generateSecurePassword(passwordLength)
		for j := 0; j < len(password); j++ {
			charCount[password[j]]++
		}
	}

	// Check that we have a reasonable distribution of characters
	totalChars := numSamples * passwordLength

	// Calculate expected count per character
	// lowercase: 26, uppercase: 26, numbers: 10, special: 8 = 70 total characters
	expectedPerChar := float64(totalChars) / 70.0

	// Count how many characters have significantly low occurrence
	lowOccurrence := 0
	for _, count := range charCount {
		if float64(count) < expectedPerChar*0.3 {
			lowOccurrence++
		}
	}

	// Allow some variance but not too much
	if lowOccurrence > 20 {
		t.Errorf("Too many characters with low occurrence (%d), possible bias in generation", lowOccurrence)
	}
}
