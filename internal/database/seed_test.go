package database

import (
	"testing"
)

// TestGenerateSecurePassword_IsCryptographicallySecure tests that password generation
// uses cryptographically secure random bytes, not predictable math/rand
func TestGenerateSecurePassword_IsCryptographicallySecure(t *testing.T) {
	// Generate multiple passwords rapidly - if using math/rand with time seed,
	// passwords generated in quick succession may be identical or highly predictable
	passwords := make(map[string]bool)
	collisions := 0
	iterations := 100

	for i := 0; i < iterations; i++ {
		pw := generateSecurePassword(20)
		if passwords[pw] {
			collisions++
		}
		passwords[pw] = true
	}

	// With crypto/rand, collision probability for 20-char passwords is effectively zero
	// With math/rand seeded by time, we may see collisions when run quickly
	if collisions > 0 {
		t.Errorf("Found %d password collisions in %d iterations - suggests predictable RNG", collisions, iterations)
	}

	// Additional check: verify we got unique passwords
	if len(passwords) != iterations {
		t.Errorf("Expected %d unique passwords, got %d - suggests predictable RNG", iterations, len(passwords))
	}
}

// TestGenerateSecurePassword_RequiredCharacterTypes verifies password contains
// required character types (lowercase, uppercase, numbers, special)
func TestGenerateSecurePassword_RequiredCharacterTypes(t *testing.T) {
	// Generate multiple passwords and verify each meets requirements
	for i := 0; i < 10; i++ {
		pw := generateSecurePassword(20)

		hasLower := false
		hasUpper := false
		hasNumber := false
		hasSpecial := false

		for _, c := range pw {
			switch {
			case c >= 'a' && c <= 'z':
				hasLower = true
			case c >= 'A' && c <= 'Z':
				hasUpper = true
			case c >= '0' && c <= '9':
				hasNumber = true
			case c == '!' || c == '@' || c == '#' || c == '$' || c == '%' || c == '^' || c == '&' || c == '*':
				hasSpecial = true
			}
		}

		if !hasLower {
			t.Errorf("Password %q missing lowercase letter", pw)
		}
		if !hasUpper {
			t.Errorf("Password %q missing uppercase letter", pw)
		}
		if !hasNumber {
			t.Errorf("Password %q missing number", pw)
		}
		if !hasSpecial {
			t.Errorf("Password %q missing special character", pw)
		}
	}
}

// TestGenerateSecurePassword_Length verifies password has correct length
func TestGenerateSecurePassword_Length(t *testing.T) {
	lengths := []int{8, 12, 16, 20, 32}

	for _, length := range lengths {
		pw := generateSecurePassword(length)
		if len(pw) != length {
			t.Errorf("Expected password length %d, got %d", length, len(pw))
		}
	}
}
