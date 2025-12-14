package services

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestAuthService_HashPassword(t *testing.T) {
	service := NewAuthService("test-secret", 24)

	password := "TestPassword123"
	hash, err := service.HashPassword(password)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if hash == "" {
		t.Error("Expected hash to be generated")
	}

	if hash == password {
		t.Error("Expected hash to be different from password")
	}
}

func TestAuthService_CheckPassword(t *testing.T) {
	service := NewAuthService("test-secret", 24)

	password := "TestPassword123"
	hash, _ := service.HashPassword(password)

	// Test correct password
	if !service.CheckPassword(password, hash) {
		t.Error("Expected password to match hash")
	}

	// Test incorrect password
	if service.CheckPassword("WrongPassword", hash) {
		t.Error("Expected incorrect password to not match")
	}
}

func TestAuthService_GenerateToken(t *testing.T) {
	service := NewAuthService("test-secret", 24)

	token1, err := service.GenerateToken()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(token1) != 64 { // 32 bytes = 64 hex characters
		t.Errorf("Expected token length 64, got %d", len(token1))
	}

	// Test uniqueness
	token2, _ := service.GenerateToken()
	if token1 == token2 {
		t.Error("Expected tokens to be unique")
	}
}

func TestAuthService_GenerateJWT(t *testing.T) {
	service := NewAuthService("test-secret", 24)

	tokenString, err := service.GenerateJWT(1, "test@example.com", false, false)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if tokenString == "" {
		t.Error("Expected JWT token to be generated")
	}

	// Parse and verify token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})

	if err != nil {
		t.Errorf("Expected token to be valid, got %v", err)
	}

	if !token.Valid {
		t.Error("Expected token to be valid")
	}

	// Check claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("Expected claims to be MapClaims")
	}

	if claims["user_id"].(float64) != 1 {
		t.Error("Expected user_id to be 1")
	}

	if claims["email"].(string) != "test@example.com" {
		t.Error("Expected email to be test@example.com")
	}

	if claims["is_admin"].(bool) != false {
		t.Error("Expected is_admin to be false")
	}
}

func TestAuthService_ValidateJWT(t *testing.T) {
	service := NewAuthService("test-secret", 24)

	// Generate valid token
	tokenString, _ := service.GenerateJWT(1, "test@example.com", true, false)

	// Validate token
	claims, err := service.ValidateJWT(tokenString)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if (*claims)["user_id"].(float64) != 1 {
		t.Error("Expected user_id to be 1")
	}

	if (*claims)["is_admin"].(bool) != true {
		t.Error("Expected is_admin to be true")
	}

	// Test invalid token
	_, err = service.ValidateJWT("invalid-token")
	if err == nil {
		t.Error("Expected error for invalid token")
	}
}

func TestAuthService_ValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"Valid password", "Test123Pass", false},
		{"Too short", "Test1", true},
		{"No uppercase", "test123pass", true},
		{"No lowercase", "TEST123PASS", true},
		{"No number", "TestPassword", true},
		{"Valid complex", "MyP@ssw0rd", false},
	}

	service := NewAuthService("test-secret", 24)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidatePassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// DONE: TestAuthService_JWTExpiration tests token expiration
func TestAuthService_JWTExpiration(t *testing.T) {
	// Create service with 0 hour expiration for testing
	service := &AuthService{
		jwtSecret:          "test-secret",
		jwtExpirationHours: 0,
	}

	// Generate token that expires immediately
	tokenString, _ := service.GenerateJWT(1, "test@example.com", false, false)

	// Wait a moment
	time.Sleep(1 * time.Second)

	// Try to validate - should fail due to expiration
	_, err := service.ValidateJWT(tokenString)
	if err == nil {
		t.Error("Expected error for expired token")
	}
}

// DONE: TestAuthService_HashPassword_EdgeCases tests password hashing edge cases
func TestAuthService_HashPassword_EdgeCases(t *testing.T) {
	service := NewAuthService("test-secret", 24)

	tests := []struct {
		name     string
		password string
	}{
		{"empty password", ""},
		{"long password 72 chars", "TestPassword123456789012345678901234567890123456789012345678901234"},
		{"special characters", "P@ssw0rd!#$%^&*()"},
		{"unicode characters", "Päßwörd123"},
		{"spaces", "Pass word 123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := service.HashPassword(tt.password)
			if err != nil {
				t.Errorf("HashPassword() error = %v", err)
			}
			if hash == "" {
				t.Error("Hash should not be empty")
			}
			// Verify we can check it
			if !service.CheckPassword(tt.password, hash) {
				t.Error("Password should match generated hash")
			}
		})
	}
}

// DONE: TestAuthService_GenerateJWT_AdminClaims tests admin claims in JWT
func TestAuthService_GenerateJWT_AdminClaims(t *testing.T) {
	service := NewAuthService("test-secret", 24)

	t.Run("admin user", func(t *testing.T) {
		tokenString, err := service.GenerateJWT(1, "admin@example.com", true, false)
		if err != nil {
			t.Fatalf("GenerateJWT() failed: %v", err)
		}

		claims, err := service.ValidateJWT(tokenString)
		if err != nil {
			t.Fatalf("ValidateJWT() failed: %v", err)
		}

		if (*claims)["is_admin"].(bool) != true {
			t.Error("Admin flag should be true")
		}
	})

	t.Run("non-admin user", func(t *testing.T) {
		tokenString, err := service.GenerateJWT(2, "user@example.com", false, false)
		if err != nil {
			t.Fatalf("GenerateJWT() failed: %v", err)
		}

		claims, err := service.ValidateJWT(tokenString)
		if err != nil {
			t.Fatalf("ValidateJWT() failed: %v", err)
		}

		if (*claims)["is_admin"].(bool) != false {
			t.Error("Admin flag should be false")
		}
	})
}

// DONE: TestAuthService_ValidateJWT_InvalidTokens tests various invalid token scenarios
func TestAuthService_ValidateJWT_InvalidTokens(t *testing.T) {
	service := NewAuthService("test-secret", 24)

	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"random string", "not-a-jwt-token"},
		{"malformed jwt", "eyJhbGciOiJIUzI1.malformed.token"},
		{"wrong signature", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxfQ.wrong_signature"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := service.ValidateJWT(tt.token)
			if err == nil {
				t.Error("Expected error for invalid token")
			}
			if claims != nil {
				t.Error("Expected nil claims for invalid token")
			}
		})
	}
}

// DONE: TestAuthService_ValidateJWT_WrongSecret tests token validation with wrong secret
func TestAuthService_ValidateJWT_WrongSecret(t *testing.T) {
	service1 := NewAuthService("secret-1", 24)
	service2 := NewAuthService("secret-2", 24)

	// Generate token with service1
	tokenString, _ := service1.GenerateJWT(1, "test@example.com", false, false)

	// Try to validate with service2 (different secret)
	claims, err := service2.ValidateJWT(tokenString)
	if err == nil {
		t.Error("Expected error when validating token with wrong secret")
	}
	if claims != nil {
		t.Error("Expected nil claims when secret doesn't match")
	}
}

// DONE: TestAuthService_ValidatePassword_EdgeCases tests password validation edge cases
func TestAuthService_ValidatePassword_EdgeCases(t *testing.T) {
	service := NewAuthService("test-secret", 24)

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"exactly 8 chars valid", "Test1234", false},
		{"exactly 7 chars", "Test123", true},
		{"only numbers", "12345678", true},
		{"only letters lowercase", "testtest", true},
		{"only letters uppercase", "TESTTEST", true},
		{"letters and numbers no case mix", "test1234", true},
		{"very long valid", "TestPassword123456789012345678901234567890", false},
		{"empty", "", true},
		{"only spaces", "        ", true},
		{"leading/trailing spaces", "  Test123  ", false}, // Spaces allowed if other criteria met
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidatePassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword(%q) error = %v, wantErr %v", tt.password, err, tt.wantErr)
			}
		})
	}
}

// TestAuthService_GenerateTempPassword tests temporary password generation
func TestAuthService_GenerateTempPassword(t *testing.T) {
	service := NewAuthService("test-secret", 24)

	t.Run("generates 12 character password", func(t *testing.T) {
		password, err := service.GenerateTempPassword()
		if err != nil {
			t.Errorf("GenerateTempPassword() error = %v", err)
		}
		if len(password) != 12 {
			t.Errorf("Expected password length 12, got %d", len(password))
		}
	})

	t.Run("password meets validation requirements", func(t *testing.T) {
		password, err := service.GenerateTempPassword()
		if err != nil {
			t.Fatalf("GenerateTempPassword() error = %v", err)
		}

		// Generated password should pass validation
		err = service.ValidatePassword(password)
		if err != nil {
			t.Errorf("Generated password failed validation: %v (password: %s)", err, password)
		}
	})

	t.Run("generates unique passwords", func(t *testing.T) {
		passwords := make(map[string]bool)
		for i := 0; i < 100; i++ {
			password, err := service.GenerateTempPassword()
			if err != nil {
				t.Fatalf("GenerateTempPassword() error = %v", err)
			}
			if passwords[password] {
				t.Errorf("Duplicate password generated: %s", password)
			}
			passwords[password] = true
		}
	})

	t.Run("contains uppercase letter", func(t *testing.T) {
		password, _ := service.GenerateTempPassword()
		hasUpper := false
		for _, c := range password {
			if c >= 'A' && c <= 'Z' {
				hasUpper = true
				break
			}
		}
		if !hasUpper {
			t.Errorf("Password should contain uppercase letter: %s", password)
		}
	})

	t.Run("contains lowercase letter", func(t *testing.T) {
		password, _ := service.GenerateTempPassword()
		hasLower := false
		for _, c := range password {
			if c >= 'a' && c <= 'z' {
				hasLower = true
				break
			}
		}
		if !hasLower {
			t.Errorf("Password should contain lowercase letter: %s", password)
		}
	})

	t.Run("contains number", func(t *testing.T) {
		password, _ := service.GenerateTempPassword()
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
	})

	t.Run("password can be hashed and verified", func(t *testing.T) {
		password, err := service.GenerateTempPassword()
		if err != nil {
			t.Fatalf("GenerateTempPassword() error = %v", err)
		}

		hash, err := service.HashPassword(password)
		if err != nil {
			t.Fatalf("HashPassword() error = %v", err)
		}

		if !service.CheckPassword(password, hash) {
			t.Error("Password should match generated hash")
		}
	})
}

// TestAuthService_GenerateImpersonationJWT tests impersonation JWT generation
func TestAuthService_GenerateImpersonationJWT(t *testing.T) {
	service := NewAuthService("test-secret", 24)

	t.Run("generates valid impersonation token", func(t *testing.T) {
		tokenString, err := service.GenerateImpersonationJWT(2, "user@example.com", false, false, 1)
		if err != nil {
			t.Fatalf("GenerateImpersonationJWT() error = %v", err)
		}

		if tokenString == "" {
			t.Error("Expected JWT token to be generated")
		}

		// Parse and verify token
		claims, err := service.ValidateJWT(tokenString)
		if err != nil {
			t.Fatalf("ValidateJWT() failed: %v", err)
		}

		// Check user_id is target user
		if (*claims)["user_id"].(float64) != 2 {
			t.Errorf("Expected user_id to be 2, got %v", (*claims)["user_id"])
		}

		// Check email is target user
		if (*claims)["email"].(string) != "user@example.com" {
			t.Errorf("Expected email to be user@example.com, got %v", (*claims)["email"])
		}
	})
}

// TestAuthService_GenerateImpersonationJWT_ContainsOriginalUserID tests original user ID claim
func TestAuthService_GenerateImpersonationJWT_ContainsOriginalUserID(t *testing.T) {
	service := NewAuthService("test-secret", 24)

	tokenString, err := service.GenerateImpersonationJWT(2, "user@example.com", false, false, 1)
	if err != nil {
		t.Fatalf("GenerateImpersonationJWT() error = %v", err)
	}

	claims, err := service.ValidateJWT(tokenString)
	if err != nil {
		t.Fatalf("ValidateJWT() failed: %v", err)
	}

	originalUserID, ok := (*claims)["original_user_id"].(float64)
	if !ok {
		t.Fatal("Expected original_user_id claim to exist")
	}

	if originalUserID != 1 {
		t.Errorf("Expected original_user_id to be 1, got %v", originalUserID)
	}
}

// TestAuthService_GenerateImpersonationJWT_ContainsImpersonatingFlag tests impersonating flag
func TestAuthService_GenerateImpersonationJWT_ContainsImpersonatingFlag(t *testing.T) {
	service := NewAuthService("test-secret", 24)

	tokenString, err := service.GenerateImpersonationJWT(2, "user@example.com", false, false, 1)
	if err != nil {
		t.Fatalf("GenerateImpersonationJWT() error = %v", err)
	}

	claims, err := service.ValidateJWT(tokenString)
	if err != nil {
		t.Fatalf("ValidateJWT() failed: %v", err)
	}

	impersonating, ok := (*claims)["impersonating"].(bool)
	if !ok {
		t.Fatal("Expected impersonating claim to exist")
	}

	if !impersonating {
		t.Error("Expected impersonating to be true")
	}
}

// TestAuthService_ValidateJWT_ExtractsImpersonationClaims tests extraction of impersonation claims
func TestAuthService_ValidateJWT_ExtractsImpersonationClaims(t *testing.T) {
	service := NewAuthService("test-secret", 24)

	// Generate impersonation token with admin impersonating regular user
	tokenString, err := service.GenerateImpersonationJWT(5, "regular@example.com", false, false, 1)
	if err != nil {
		t.Fatalf("GenerateImpersonationJWT() error = %v", err)
	}

	claims, err := service.ValidateJWT(tokenString)
	if err != nil {
		t.Fatalf("ValidateJWT() failed: %v", err)
	}

	// Check all expected claims
	tests := []struct {
		name     string
		key      string
		expected interface{}
	}{
		{"user_id", "user_id", float64(5)},
		{"email", "email", "regular@example.com"},
		{"is_admin", "is_admin", false},
		{"is_super_admin", "is_super_admin", false},
		{"original_user_id", "original_user_id", float64(1)},
		{"impersonating", "impersonating", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, ok := (*claims)[tt.key]
			if !ok {
				t.Errorf("Expected claim %s to exist", tt.key)
				return
			}
			if value != tt.expected {
				t.Errorf("Expected %s to be %v, got %v", tt.key, tt.expected, value)
			}
		})
	}
}

// TestAuthService_GenerateImpersonationJWT_PreservesAdminFlags tests admin flags are correct for target
func TestAuthService_GenerateImpersonationJWT_PreservesAdminFlags(t *testing.T) {
	service := NewAuthService("test-secret", 24)

	t.Run("impersonating regular admin", func(t *testing.T) {
		tokenString, err := service.GenerateImpersonationJWT(3, "admin@example.com", true, false, 1)
		if err != nil {
			t.Fatalf("GenerateImpersonationJWT() error = %v", err)
		}

		claims, err := service.ValidateJWT(tokenString)
		if err != nil {
			t.Fatalf("ValidateJWT() failed: %v", err)
		}

		if (*claims)["is_admin"].(bool) != true {
			t.Error("Expected is_admin to be true for admin target")
		}
		if (*claims)["is_super_admin"].(bool) != false {
			t.Error("Expected is_super_admin to be false for regular admin")
		}
	})

	t.Run("impersonating regular user", func(t *testing.T) {
		tokenString, err := service.GenerateImpersonationJWT(5, "user@example.com", false, false, 1)
		if err != nil {
			t.Fatalf("GenerateImpersonationJWT() error = %v", err)
		}

		claims, err := service.ValidateJWT(tokenString)
		if err != nil {
			t.Fatalf("ValidateJWT() failed: %v", err)
		}

		if (*claims)["is_admin"].(bool) != false {
			t.Error("Expected is_admin to be false for regular user")
		}
		if (*claims)["is_super_admin"].(bool) != false {
			t.Error("Expected is_super_admin to be false for regular user")
		}
	})
}
