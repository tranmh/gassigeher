package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthService provides authentication utilities
type AuthService struct {
	jwtSecret          string
	jwtExpirationHours int
}

// NewAuthService creates a new auth service
func NewAuthService(jwtSecret string, jwtExpirationHours int) *AuthService {
	return &AuthService{
		jwtSecret:          jwtSecret,
		jwtExpirationHours: jwtExpirationHours,
	}
}

// HashPassword hashes a password using bcrypt
func (s *AuthService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

// CheckPassword verifies a password against a hash
func (s *AuthService) CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateToken generates a random token for verification or reset
func (s *AuthService) GenerateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// GenerateTempPassword generates a secure temporary password
// Returns a 12-character password with uppercase, lowercase, and numbers
// Uses crypto/rand.Int to eliminate modulo bias for uniform distribution
func (s *AuthService) GenerateTempPassword() (string, error) {
	// Character set excluding ambiguous characters (0, O, l, 1, I)
	const charset = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	const uppercase = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	const lowercase = "abcdefghjkmnpqrstuvwxyz"
	const digits = "23456789"
	const length = 12

	result := make([]byte, length)

	// Generate random characters using crypto/rand.Int (no modulo bias)
	for i := 0; i < length; i++ {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", fmt.Errorf("failed to generate temp password: %w", err)
		}
		result[i] = charset[idx.Int64()]
	}

	// Ensure password meets requirements by placing required characters
	// Position 0: uppercase, Position 1: lowercase, Position 2: number
	upperIdx, err := rand.Int(rand.Reader, big.NewInt(int64(len(uppercase))))
	if err != nil {
		return "", fmt.Errorf("failed to generate temp password: %w", err)
	}
	result[0] = uppercase[upperIdx.Int64()]

	lowerIdx, err := rand.Int(rand.Reader, big.NewInt(int64(len(lowercase))))
	if err != nil {
		return "", fmt.Errorf("failed to generate temp password: %w", err)
	}
	result[1] = lowercase[lowerIdx.Int64()]

	digitIdx, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
	if err != nil {
		return "", fmt.Errorf("failed to generate temp password: %w", err)
	}
	result[2] = digits[digitIdx.Int64()]

	return string(result), nil
}

// GenerateJWT generates a JWT token for a user
// DONE: Phase 3 - Updated to include is_super_admin claim
// SaaS: Updated to include tenant_id and is_central_admin claims
func (s *AuthService) GenerateJWT(userID int, email string, isAdmin bool, isSuperAdmin bool, isCentralAdmin bool, tenantID int) (string, error) {
	claims := jwt.MapClaims{
		"user_id":          userID,
		"email":            email,
		"is_admin":         isAdmin,
		"is_super_admin":   isSuperAdmin,
		"is_central_admin": isCentralAdmin, // SaaS: Platform-wide admin
		"tenant_id":        tenantID,       // SaaS: Tenant ID for multi-tenancy
		"exp":              time.Now().Add(time.Hour * time.Duration(s.jwtExpirationHours)).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// DONE: Phase 3 - JWT now includes is_admin and is_super_admin claims
// SaaS: JWT now includes tenant_id claim

// GenerateImpersonationJWT generates a JWT token for impersonation
// Includes original_user_id, impersonating flag, tenant_id, and is_central_admin for audit trail
func (s *AuthService) GenerateImpersonationJWT(targetUserID int, targetEmail string, targetIsAdmin bool, targetIsSuperAdmin bool, targetIsCentralAdmin bool, originalUserID int, tenantID int) (string, error) {
	claims := jwt.MapClaims{
		"user_id":          targetUserID,
		"email":            targetEmail,
		"is_admin":         targetIsAdmin,
		"is_super_admin":   targetIsSuperAdmin,
		"is_central_admin": targetIsCentralAdmin, // SaaS: Platform-wide admin
		"tenant_id":        tenantID,             // SaaS: Tenant ID for multi-tenancy
		"original_user_id": originalUserID,
		"impersonating":    true,
		"exp":              time.Now().Add(time.Hour * time.Duration(s.jwtExpirationHours)).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign impersonation token: %w", err)
	}

	return tokenString, nil
}

// ValidateJWT validates and parses a JWT token
func (s *AuthService) ValidateJWT(tokenString string) (*jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return &claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// ValidatePassword checks if a password meets requirements
func (s *AuthService) ValidatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	hasUpper := false
	hasLower := false
	hasNumber := false

	for _, char := range password {
		switch {
		case 'A' <= char && char <= 'Z':
			hasUpper = true
		case 'a' <= char && char <= 'z':
			hasLower = true
		case '0' <= char && char <= '9':
			hasNumber = true
		}
	}

	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if !hasNumber {
		return fmt.Errorf("password must contain at least one number")
	}

	return nil
}
