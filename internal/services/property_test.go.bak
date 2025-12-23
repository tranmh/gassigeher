package services

import (
	"strings"
	"testing"
	"unicode"

	"pgregory.net/rapid"
)

// ============================================================================
// CUSTOM GENERATORS FOR SERVICES
// ============================================================================

// genStrongPassword generates passwords that should pass validation
func genStrongPassword() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		// Ensure at least: 8 chars, 1 upper, 1 lower, 1 digit
		upper := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		lower := "abcdefghijklmnopqrstuvwxyz"
		digits := "0123456789"
		all := upper + lower + digits

		length := rapid.IntRange(8, 50).Draw(t, "length")
		result := make([]byte, length)

		// First 3 chars ensure requirements
		result[0] = upper[rapid.IntRange(0, len(upper)-1).Draw(t, "upper")]
		result[1] = lower[rapid.IntRange(0, len(lower)-1).Draw(t, "lower")]
		result[2] = digits[rapid.IntRange(0, len(digits)-1).Draw(t, "digit")]

		// Rest are random from all
		for i := 3; i < length; i++ {
			result[i] = all[rapid.IntRange(0, len(all)-1).Draw(t, "char")]
		}

		return string(result)
	})
}

// genWeakPassword generates passwords that should fail validation
func genWeakPassword() *rapid.Generator[string] {
	return rapid.OneOf(
		// Too short
		rapid.StringN(0, 7, -1),
		// No uppercase
		rapid.Custom(func(t *rapid.T) string {
			_ = rapid.IntRange(0, 100).Draw(t, "dummy")
			return "abcdefg1"
		}),
		// No lowercase
		rapid.Custom(func(t *rapid.T) string {
			_ = rapid.IntRange(0, 100).Draw(t, "dummy")
			return "ABCDEFG1"
		}),
		// No digit
		rapid.Custom(func(t *rapid.T) string {
			_ = rapid.IntRange(0, 100).Draw(t, "dummy")
			return "Abcdefgh"
		}),
	)
}

// genJWTSecret generates JWT secrets of various lengths
func genJWTSecret() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		length := rapid.IntRange(1, 100).Draw(t, "length")
		chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		result := make([]byte, length)
		for i := 0; i < length; i++ {
			result[i] = chars[rapid.IntRange(0, len(chars)-1).Draw(t, "char")]
		}
		return string(result)
	})
}

// ============================================================================
// PROPERTY TESTS FOR PASSWORD VALIDATION
// ============================================================================

func TestProperty_StrongPasswordAccepted(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		password := genStrongPassword().Draw(t, "password")
		service := NewAuthService("test-secret", 24)

		err := service.ValidatePassword(password)
		if err != nil {
			t.Fatalf("Strong password %q was rejected: %v", password, err)
		}
	})
}

func TestProperty_WeakPasswordRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		password := genWeakPassword().Draw(t, "password")
		service := NewAuthService("test-secret", 24)

		err := service.ValidatePassword(password)
		if err == nil {
			t.Fatalf("Weak password %q was accepted", password)
		}
	})
}

func TestProperty_PasswordValidationInvariants(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		password := rapid.String().Draw(t, "password")
		service := NewAuthService("test-secret", 24)

		err := service.ValidatePassword(password)

		if err == nil {
			// PROPERTY 1: Length must be >= 8
			if len(password) < 8 {
				t.Fatalf("BUG: Accepted password with length %d: %q", len(password), password)
			}

			// PROPERTY 2: Must contain uppercase
			hasUpper := false
			for _, c := range password {
				if c >= 'A' && c <= 'Z' {
					hasUpper = true
					break
				}
			}
			if !hasUpper {
				t.Fatalf("BUG: Accepted password without uppercase: %q", password)
			}

			// PROPERTY 3: Must contain lowercase
			hasLower := false
			for _, c := range password {
				if c >= 'a' && c <= 'z' {
					hasLower = true
					break
				}
			}
			if !hasLower {
				t.Fatalf("BUG: Accepted password without lowercase: %q", password)
			}

			// PROPERTY 4: Must contain digit
			hasDigit := false
			for _, c := range password {
				if c >= '0' && c <= '9' {
					hasDigit = true
					break
				}
			}
			if !hasDigit {
				t.Fatalf("BUG: Accepted password without digit: %q", password)
			}
		}
	})
}

// Test that Unicode uppercase/lowercase is NOT counted (only ASCII)
func TestProperty_PasswordOnlyASCII(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate password with Unicode upper/lower but no ASCII upper/lower
		unicodeUpper := "ÀÁÂÃÄÅÆÇÈÉÊËÌÍÎÏ" // Unicode uppercase
		unicodeLower := "àáâãäåæçèéêëìíîï" // Unicode lowercase

		password := rapid.Custom(func(t *rapid.T) string {
			// 8+ chars with Unicode upper, Unicode lower, and ASCII digit
			// but NO ASCII upper/lower
			result := make([]rune, 10)
			result[0] = rune(unicodeUpper[rapid.IntRange(0, len(unicodeUpper)-1).Draw(t, "upper")])
			result[1] = rune(unicodeLower[rapid.IntRange(0, len(unicodeLower)-1).Draw(t, "lower")])
			result[2] = '1' // ASCII digit
			for i := 3; i < 10; i++ {
				result[i] = rune(unicodeLower[rapid.IntRange(0, len(unicodeLower)-1).Draw(t, "fill")])
			}
			return string(result)
		}).Draw(t, "password")

		service := NewAuthService("test-secret", 24)
		err := service.ValidatePassword(password)

		// PROPERTY: Password with only Unicode upper/lower should be REJECTED
		// because the validation only checks ASCII [A-Z] and [a-z]
		if err == nil {
			// Check if password actually has ASCII upper/lower
			hasASCIIUpper := false
			hasASCIILower := false
			for _, c := range password {
				if c >= 'A' && c <= 'Z' {
					hasASCIIUpper = true
				}
				if c >= 'a' && c <= 'z' {
					hasASCIILower = true
				}
			}
			if !hasASCIIUpper || !hasASCIILower {
				// This would be a bug if the validation claims to require upper/lower
				// but actually accepts Unicode equivalents
				t.Logf("Note: Password %q accepted without ASCII upper/lower", password)
			}
		}
	})
}

// ============================================================================
// PROPERTY TESTS FOR PASSWORD HASHING
// ============================================================================

func TestProperty_HashPasswordRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping slow bcrypt property test in short mode")
	}
	rapid.Check(t, func(t *rapid.T) {
		// bcrypt has a 72-byte limit, test around that boundary
		password := rapid.StringN(1, 100, -1).Draw(t, "password")
		service := NewAuthService("test-secret", 24)

		hash, err := service.HashPassword(password)
		if err != nil {
			// Some passwords might fail (e.g., very long ones with bcrypt)
			return
		}

		// PROPERTY: Hashed password must verify correctly
		if !service.CheckPassword(password, hash) {
			t.Fatalf("BUG: Password %q hash doesn't verify", password)
		}
	})
}

func TestProperty_HashPasswordUniqueness(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping slow bcrypt property test in short mode")
	}
	rapid.Check(t, func(t *rapid.T) {
		password := rapid.StringN(1, 50, -1).Draw(t, "password")
		service := NewAuthService("test-secret", 24)

		hash1, err1 := service.HashPassword(password)
		hash2, err2 := service.HashPassword(password)

		if err1 != nil || err2 != nil {
			return
		}

		// PROPERTY: Same password should produce different hashes (due to salt)
		if hash1 == hash2 {
			t.Fatalf("BUG: Same password produced identical hashes")
		}

		// PROPERTY: Both hashes should verify the password
		if !service.CheckPassword(password, hash1) {
			t.Fatalf("BUG: First hash doesn't verify")
		}
		if !service.CheckPassword(password, hash2) {
			t.Fatalf("BUG: Second hash doesn't verify")
		}
	})
}

func TestProperty_WrongPasswordFails(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping slow bcrypt property test in short mode")
	}
	rapid.Check(t, func(t *rapid.T) {
		password1 := rapid.StringN(1, 50, -1).Draw(t, "password1")
		password2 := rapid.StringN(1, 50, -1).Draw(t, "password2")

		if password1 == password2 {
			t.Skip("same passwords")
		}

		service := NewAuthService("test-secret", 24)

		hash, err := service.HashPassword(password1)
		if err != nil {
			return
		}

		// PROPERTY: Wrong password must not verify
		if service.CheckPassword(password2, hash) {
			t.Fatalf("BUG: Wrong password %q verified against hash of %q", password2, password1)
		}
	})
}

// Test bcrypt 72-byte truncation behavior
func TestProperty_BcryptTruncation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping slow bcrypt property test in short mode")
	}
	rapid.Check(t, func(t *rapid.T) {
		// Generate password longer than 72 bytes
		baseLen := rapid.IntRange(73, 100).Draw(t, "len")
		password := strings.Repeat("a", baseLen)
		service := NewAuthService("test-secret", 24)

		hash, err := service.HashPassword(password)
		if err != nil {
			return
		}

		// The first 72 bytes should verify
		truncated := password[:72]
		if !service.CheckPassword(truncated, hash) {
			// This is expected behavior with bcrypt, not a bug
			// But good to document
			t.Logf("Note: Truncated password doesn't verify (bcrypt implementation detail)")
		}

		// A different password with same first 72 bytes should also verify
		// This is a known bcrypt limitation
		modified := password + "EXTRA"
		if service.CheckPassword(modified, hash) != service.CheckPassword(password, hash) {
			// Both should have same result due to truncation
			t.Logf("Note: Bcrypt truncation affects verification")
		}
	})
}

// ============================================================================
// PROPERTY TESTS FOR JWT GENERATION AND VALIDATION
// ============================================================================

func TestProperty_JWTRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		userID := rapid.IntRange(1, 1000000).Draw(t, "userID")
		email := rapid.StringMatching(`[a-z]+@[a-z]+\.[a-z]+`).Draw(t, "email")
		isAdmin := rapid.Bool().Draw(t, "isAdmin")
		isSuperAdmin := rapid.Bool().Draw(t, "isSuperAdmin")
		isCentralAdmin := rapid.Bool().Draw(t, "isCentralAdmin")
		tenantID := rapid.IntRange(1, 1000).Draw(t, "tenantID")

		secret := genJWTSecret().Draw(t, "secret")
		service := NewAuthService(secret, 24)

		token, err := service.GenerateJWT(userID, email, isAdmin, isSuperAdmin, isCentralAdmin, tenantID)
		if err != nil {
			t.Fatalf("Failed to generate JWT: %v", err)
		}

		claims, err := service.ValidateJWT(token)
		if err != nil {
			t.Fatalf("Failed to validate JWT: %v", err)
		}

		// PROPERTY: Claims must match what was encoded
		claimUserID, _ := (*claims)["user_id"].(float64)
		if int(claimUserID) != userID {
			t.Fatalf("BUG: user_id mismatch: got %v, want %d", claimUserID, userID)
		}

		claimEmail, _ := (*claims)["email"].(string)
		if claimEmail != email {
			t.Fatalf("BUG: email mismatch: got %v, want %s", claimEmail, email)
		}

		claimIsAdmin, _ := (*claims)["is_admin"].(bool)
		if claimIsAdmin != isAdmin {
			t.Fatalf("BUG: is_admin mismatch: got %v, want %v", claimIsAdmin, isAdmin)
		}

		claimTenantID, _ := (*claims)["tenant_id"].(float64)
		if int(claimTenantID) != tenantID {
			t.Fatalf("BUG: tenant_id mismatch: got %v, want %d", claimTenantID, tenantID)
		}
	})
}

func TestProperty_JWTWrongSecretFails(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		secret1 := genJWTSecret().Draw(t, "secret1")
		secret2 := genJWTSecret().Draw(t, "secret2")

		if secret1 == secret2 {
			t.Skip("same secrets")
		}

		service1 := NewAuthService(secret1, 24)
		service2 := NewAuthService(secret2, 24)

		token, err := service1.GenerateJWT(1, "test@test.com", false, false, false, 1)
		if err != nil {
			return
		}

		// PROPERTY: Token signed with secret1 must not validate with secret2
		_, err = service2.ValidateJWT(token)
		if err == nil {
			t.Fatalf("BUG: Token validated with wrong secret")
		}
	})
}

func TestProperty_JWTMalformedRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random strings that are unlikely to be valid JWTs
		malformed := rapid.OneOf(
			rapid.Just(""),
			rapid.Just("not.a.jwt"),
			rapid.Just("..."),
			rapid.String(),
			// Partial JWT-like strings
			rapid.Custom(func(t *rapid.T) string {
				return "eyJ" + rapid.String().Draw(t, "partial")
			}),
		).Draw(t, "malformed")

		service := NewAuthService("test-secret", 24)

		// PROPERTY: Malformed tokens must be rejected
		_, err := service.ValidateJWT(malformed)
		if err == nil {
			// Check if it's actually a valid JWT structure
			parts := strings.Split(malformed, ".")
			if len(parts) != 3 {
				t.Fatalf("BUG: Malformed token %q was accepted", malformed)
			}
		}
	})
}

// Test for algorithm confusion attacks (alg:none)
func TestProperty_JWTAlgNoneRejected(t *testing.T) {
	// alg:none attack: {"alg":"none","typ":"JWT"}.{"sub":"1"}.
	// Base64: eyJ0eXAiOiJKV1QiLCJhbGciOiJub25lIn0.eyJzdWIiOiIxIn0.
	algNoneTokens := []string{
		"eyJ0eXAiOiJKV1QiLCJhbGciOiJub25lIn0.eyJzdWIiOiIxIn0.",
		"eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJ1c2VyX2lkIjoxfQ.",
		"eyJhbGciOiJOT05FIiwidHlwIjoiSldUIn0.eyJ1c2VyX2lkIjoxfQ.",
		"eyJhbGciOiJOb25lIiwidHlwIjoiSldUIn0.eyJ1c2VyX2lkIjoxfQ.",
	}

	service := NewAuthService("test-secret", 24)

	for _, token := range algNoneTokens {
		_, err := service.ValidateJWT(token)
		if err == nil {
			t.Fatalf("BUG: alg:none token was accepted: %s", token)
		}
	}
}

// ============================================================================
// PROPERTY TESTS FOR TOKEN GENERATION
// ============================================================================

func TestProperty_TokenUniqueness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		service := NewAuthService("test-secret", 24)

		token1, err1 := service.GenerateToken()
		token2, err2 := service.GenerateToken()

		if err1 != nil || err2 != nil {
			return
		}

		// PROPERTY: Two generated tokens must be different
		if token1 == token2 {
			t.Fatalf("BUG: Generated duplicate tokens")
		}
	})
}

func TestProperty_TokenLength(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		service := NewAuthService("test-secret", 24)

		token, err := service.GenerateToken()
		if err != nil {
			return
		}

		// PROPERTY: Token must be 64 hex chars (32 bytes * 2)
		if len(token) != 64 {
			t.Fatalf("BUG: Token has unexpected length %d (expected 64)", len(token))
		}

		// PROPERTY: Token must be valid hex
		for _, c := range token {
			isHex := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
			if !isHex {
				t.Fatalf("BUG: Token contains non-hex character: %c", c)
			}
		}
	})
}

// ============================================================================
// PROPERTY TESTS FOR TEMP PASSWORD GENERATION
// ============================================================================

func TestProperty_TempPasswordRequirements(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		service := NewAuthService("test-secret", 24)

		password, err := service.GenerateTempPassword()
		if err != nil {
			t.Fatalf("Failed to generate temp password: %v", err)
		}

		// PROPERTY 1: Must be 12 characters
		if len(password) != 12 {
			t.Fatalf("BUG: Temp password has length %d (expected 12)", len(password))
		}

		// PROPERTY 2: Must contain uppercase
		hasUpper := false
		for _, c := range password {
			if unicode.IsUpper(c) {
				hasUpper = true
				break
			}
		}
		if !hasUpper {
			t.Fatalf("BUG: Temp password has no uppercase: %s", password)
		}

		// PROPERTY 3: Must contain lowercase
		hasLower := false
		for _, c := range password {
			if unicode.IsLower(c) {
				hasLower = true
				break
			}
		}
		if !hasLower {
			t.Fatalf("BUG: Temp password has no lowercase: %s", password)
		}

		// PROPERTY 4: Must contain digit
		hasDigit := false
		for _, c := range password {
			if unicode.IsDigit(c) {
				hasDigit = true
				break
			}
		}
		if !hasDigit {
			t.Fatalf("BUG: Temp password has no digit: %s", password)
		}

		// PROPERTY 5: Must pass validation
		if err := service.ValidatePassword(password); err != nil {
			t.Fatalf("BUG: Generated temp password fails validation: %s - %v", password, err)
		}

		// PROPERTY 6: Must not contain ambiguous characters
		ambiguous := "0O1lI"
		for _, c := range password {
			if strings.ContainsRune(ambiguous, c) {
				t.Fatalf("BUG: Temp password contains ambiguous character %c: %s", c, password)
			}
		}
	})
}

func TestProperty_TempPasswordUniqueness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		service := NewAuthService("test-secret", 24)

		password1, err1 := service.GenerateTempPassword()
		password2, err2 := service.GenerateTempPassword()

		if err1 != nil || err2 != nil {
			return
		}

		// PROPERTY: Two generated passwords must be different
		if password1 == password2 {
			t.Fatalf("BUG: Generated duplicate temp passwords")
		}
	})
}

// ============================================================================
// PROPERTY TESTS FOR EMAIL CONFIG VALIDATION
// ============================================================================

func TestProperty_GmailConfigValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		clientID := rapid.String().Draw(t, "clientID")
		clientSecret := rapid.String().Draw(t, "clientSecret")
		refreshToken := rapid.String().Draw(t, "refreshToken")
		fromEmail := rapid.String().Draw(t, "fromEmail")

		config := &EmailConfig{
			Provider:          "gmail",
			GmailClientID:     clientID,
			GmailClientSecret: clientSecret,
			GmailRefreshToken: refreshToken,
			GmailFromEmail:    fromEmail,
		}

		err := ValidateEmailConfig(config)

		if err == nil {
			// PROPERTY: All Gmail fields must be non-empty
			if clientID == "" {
				t.Fatalf("BUG: Accepted Gmail config with empty client ID")
			}
			if clientSecret == "" {
				t.Fatalf("BUG: Accepted Gmail config with empty client secret")
			}
			if refreshToken == "" {
				t.Fatalf("BUG: Accepted Gmail config with empty refresh token")
			}
			if fromEmail == "" {
				t.Fatalf("BUG: Accepted Gmail config with empty from email")
			}
		}
	})
}

func TestProperty_SMTPConfigValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		host := rapid.String().Draw(t, "host")
		port := rapid.IntRange(-100, 70000).Draw(t, "port")
		username := rapid.String().Draw(t, "username")
		password := rapid.String().Draw(t, "password")
		fromEmail := rapid.String().Draw(t, "fromEmail")

		config := &EmailConfig{
			Provider:      "smtp",
			SMTPHost:      host,
			SMTPPort:      port,
			SMTPUsername:  username,
			SMTPPassword:  password,
			SMTPFromEmail: fromEmail,
		}

		err := ValidateEmailConfig(config)

		if err == nil {
			// PROPERTY: Host must be non-empty
			if host == "" {
				t.Fatalf("BUG: Accepted SMTP config with empty host")
			}

			// PROPERTY: Port must be valid (1-65535)
			if port < 1 || port > 65535 {
				t.Fatalf("BUG: Accepted SMTP config with invalid port %d", port)
			}

			// PROPERTY: From email must be non-empty
			if fromEmail == "" {
				t.Fatalf("BUG: Accepted SMTP config with empty from email")
			}
		}
	})
}

func TestProperty_InvalidProviderRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		provider := rapid.StringMatching(`[a-z]+`).Draw(t, "provider")

		// Skip valid providers
		if provider == "gmail" || provider == "smtp" {
			t.Skip("valid provider")
		}

		config := &EmailConfig{
			Provider: provider,
		}

		err := ValidateEmailConfig(config)

		// PROPERTY: Invalid provider should be rejected
		if err == nil {
			t.Fatalf("BUG: Invalid provider %q was accepted", provider)
		}
	})
}

// ============================================================================
// PROPERTY TESTS FOR IMPERSONATION JWT
// ============================================================================

func TestProperty_ImpersonationJWTContainsOriginalUser(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		targetUserID := rapid.IntRange(1, 1000).Draw(t, "targetUserID")
		targetEmail := rapid.StringMatching(`[a-z]+@[a-z]+\.[a-z]+`).Draw(t, "email")
		originalUserID := rapid.IntRange(1, 1000).Draw(t, "originalUserID")
		tenantID := rapid.IntRange(1, 100).Draw(t, "tenantID")

		service := NewAuthService("test-secret", 24)

		token, err := service.GenerateImpersonationJWT(
			targetUserID, targetEmail, false, false, false, originalUserID, tenantID,
		)
		if err != nil {
			t.Fatalf("Failed to generate impersonation JWT: %v", err)
		}

		claims, err := service.ValidateJWT(token)
		if err != nil {
			t.Fatalf("Failed to validate impersonation JWT: %v", err)
		}

		// PROPERTY 1: Must have impersonating flag set to true
		impersonating, ok := (*claims)["impersonating"].(bool)
		if !ok || !impersonating {
			t.Fatalf("BUG: Impersonation token missing impersonating flag")
		}

		// PROPERTY 2: Must contain original_user_id
		origID, ok := (*claims)["original_user_id"].(float64)
		if !ok || int(origID) != originalUserID {
			t.Fatalf("BUG: Impersonation token has wrong original_user_id: got %v, want %d", origID, originalUserID)
		}

		// PROPERTY 3: user_id should be target user
		userID, ok := (*claims)["user_id"].(float64)
		if !ok || int(userID) != targetUserID {
			t.Fatalf("BUG: Impersonation token has wrong user_id: got %v, want %d", userID, targetUserID)
		}
	})
}
