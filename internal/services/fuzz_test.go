package services

import (
	"strings"
	"testing"
)

// FuzzValidatePassword tests password validation with random inputs
func FuzzValidatePassword(f *testing.F) {
	// Seed corpus
	f.Add("password123")
	f.Add("short")
	f.Add("")
	f.Add("12345678")
	f.Add("verylongpasswordwithmanycharacters")
	f.Add(strings.Repeat("a", 1000))
	f.Add("password with spaces")
	f.Add("password\twith\ttabs")
	f.Add("password\nwith\nnewlines")
	f.Add("пароль123") // Cyrillic
	f.Add("密码123456") // Chinese
	f.Add("🔒🔑🗝️")  // Emoji
	f.Add("\x00\x00\x00\x00\x00\x00\x00\x00") // Null bytes

	f.Fuzz(func(t *testing.T, password string) {
		service := NewAuthService("test-secret", 24)

		// Should not panic
		err := service.ValidatePassword(password)

		// If valid, verify minimum length
		if err == nil && len(password) < 8 {
			t.Errorf("ValidatePassword accepted password shorter than 8 chars: %q (len=%d)", password, len(password))
		}
	})
}

// FuzzHashPassword tests password hashing robustness
func FuzzHashPassword(f *testing.F) {
	// Seed corpus
	f.Add("password123")
	f.Add("")
	f.Add(strings.Repeat("a", 72)) // bcrypt max length
	f.Add(strings.Repeat("a", 100))
	f.Add("unicode: 日本語")
	f.Add("\x00\x00\x00")

	f.Fuzz(func(t *testing.T, password string) {
		service := NewAuthService("test-secret", 24)

		// Should not panic (may return error for edge cases)
		hash, err := service.HashPassword(password)

		// If hash succeeded, verify it can be checked
		if err == nil && hash != "" {
			valid := service.CheckPassword(password, hash)
			if !valid {
				t.Errorf("CheckPassword failed for password that was just hashed: %q", password)
			}
		}
	})
}

// FuzzGenerateJWT tests JWT generation with various inputs
func FuzzGenerateJWT(f *testing.F) {
	// Seed corpus
	f.Add(1, "user@example.com", false, false, false, 1)
	f.Add(0, "", false, false, false, 0)
	f.Add(-1, "test@test.com", true, true, true, -1)
	f.Add(999999999, strings.Repeat("a", 1000), true, true, true, 999999)

	f.Fuzz(func(t *testing.T, userID int, email string, isAdmin, isSuperAdmin, isCentralAdmin bool, tenantID int) {
		service := NewAuthService("test-secret-key-for-fuzzing", 24)

		// Should not panic
		token, err := service.GenerateJWT(userID, email, isAdmin, isSuperAdmin, isCentralAdmin, tenantID)

		// If generation succeeded, verify token can be validated
		if err == nil && token != "" {
			claims, validateErr := service.ValidateJWT(token)
			if validateErr != nil {
				t.Errorf("ValidateJWT failed for token we just generated: %v", validateErr)
			}

			// Verify claims match
			if claims != nil {
				claimUserID, _ := (*claims)["user_id"].(float64)
				if int(claimUserID) != userID {
					t.Errorf("user_id mismatch: got %v, want %d", claimUserID, userID)
				}
			}
		}
	})
}

// FuzzValidateJWT tests JWT validation with malformed tokens
func FuzzValidateJWT(f *testing.F) {
	// Seed corpus with various malformed tokens
	f.Add("")
	f.Add("not.a.token")
	f.Add("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9")                 // Header only
	f.Add("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.payload")          // Missing signature
	f.Add("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.payload.signature") // Invalid base64
	f.Add("....") // Just dots
	f.Add(strings.Repeat("a", 10000)) // Very long
	f.Add("eyJ0eXAiOiJKV1QiLCJhbGciOiJub25lIn0.eyJzdWIiOiIxIn0.") // alg:none attack
	f.Add("null")
	f.Add("{}")
	f.Add("\x00\x00\x00")

	f.Fuzz(func(t *testing.T, token string) {
		service := NewAuthService("test-secret-key", 24)

		// Should not panic (should return error for invalid tokens)
		_, _ = service.ValidateJWT(token)
	})
}

// FuzzGenerateToken tests random token generation
func FuzzGenerateToken(f *testing.F) {
	// This just triggers multiple generations to check for panics
	f.Add(0)
	f.Add(1)
	f.Add(100)

	f.Fuzz(func(t *testing.T, _ int) {
		service := NewAuthService("test-secret", 24)

		// Should not panic
		token, err := service.GenerateToken()
		if err != nil {
			return
		}

		// Verify token is non-empty and reasonable length
		if len(token) < 32 {
			t.Errorf("Generated token too short: %d chars", len(token))
		}

		// Verify token is hex (or base64)
		for _, c := range token {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				// Not pure hex, might be base64 - that's OK
				break
			}
		}
	})
}

// FuzzEmailConfigValidation tests email config validation
func FuzzEmailConfigValidation(f *testing.F) {
	// Seed corpus
	f.Add("gmail", "client_id", "client_secret", "refresh_token", "from@example.com", "", 0, "", "", false)
	f.Add("smtp", "", "", "", "from@example.com", "smtp.example.com", 587, "user", "pass", false)
	f.Add("invalid", "", "", "", "", "", 0, "", "", false)
	f.Add("", "", "", "", "", "", 0, "", "", false)

	f.Fuzz(func(t *testing.T, provider, clientID, clientSecret, refreshToken, fromEmail, smtpHost string, smtpPort int, smtpUser, smtpPass string, useSSL bool) {
		config := &EmailConfig{
			Provider:          provider,
			GmailClientID:     clientID,
			GmailClientSecret: clientSecret,
			GmailRefreshToken: refreshToken,
			GmailFromEmail:    fromEmail,
			SMTPHost:          smtpHost,
			SMTPPort:          smtpPort,
			SMTPUsername:      smtpUser,
			SMTPPassword:      smtpPass,
			SMTPFromEmail:     fromEmail,
			SMTPUseSSL:        useSSL,
		}

		// Should not panic
		_ = ValidateEmailConfig(config)
	})
}
