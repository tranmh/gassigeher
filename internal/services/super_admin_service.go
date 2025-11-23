package services

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/tranm/gassigeher/internal/config"
	"golang.org/x/crypto/bcrypt"
)

// SuperAdminService handles Super Admin password file management
type SuperAdminService struct {
	db  *sql.DB
	cfg *config.Config
}

// NewSuperAdminService creates a new SuperAdminService
// DONE
func NewSuperAdminService(db *sql.DB, cfg *config.Config) *SuperAdminService {
	return &SuperAdminService{
		db:  db,
		cfg: cfg,
	}
}

// CheckAndUpdatePassword reads credentials file and updates password if changed
// This runs on every server startup to detect password changes
// DONE
func (s *SuperAdminService) CheckAndUpdatePassword() error {
	filePath := "SUPER_ADMIN_CREDENTIALS.txt"

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Credentials file doesn't exist - check if Super Admin exists in database
		var superAdminExists bool
		err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = 1 AND is_super_admin = 1)").Scan(&superAdminExists)
		if err != nil {
			return fmt.Errorf("failed to check if Super Admin exists: %w", err)
		}

		if superAdminExists {
			// Super Admin exists but credentials file is missing - regenerate it with new password
			log.Println("Super Admin exists but credentials file is missing - generating new password...")
			return s.regenerateCredentialsFile()
		}

		log.Println("Super Admin credentials file not found (okay for first-time installation)")
		return nil
	}

	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read credentials file: %w", err)
	}

	// Parse EMAIL and PASSWORD lines
	email, password, createdTime, err := parseCredentialsFile(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse credentials file: %w", err)
	}

	// Verify email matches config
	if email != s.cfg.SuperAdminEmail {
		return fmt.Errorf("email in credentials file (%s) doesn't match SUPER_ADMIN_EMAIL in .env (%s)",
			email, s.cfg.SuperAdminEmail)
	}

	// Get current Super Admin password hash from database
	var currentHash string
	err = s.db.QueryRow("SELECT password_hash FROM users WHERE id = 1").Scan(&currentHash)
	if err != nil {
		return fmt.Errorf("failed to get Super Admin from database: %w", err)
	}

	// Check if password has changed
	if bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(password)) == nil {
		// Password unchanged, no action needed
		log.Println("Super Admin password unchanged")
		return nil
	}

	// Password changed! Hash new password and update database
	log.Println("Super Admin password change detected, updating...")
	newHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	_, err = s.db.Exec("UPDATE users SET password_hash = ?, updated_at = ? WHERE id = 1", string(newHash), time.Now())
	if err != nil {
		return fmt.Errorf("failed to update password in database: %w", err)
	}

	// Update file with confirmation and new timestamp
	err = s.writeUpdatedCredentialsFile(email, password, createdTime, true)
	if err != nil {
		return fmt.Errorf("failed to update credentials file: %w", err)
	}

	log.Println("✓ Super Admin password updated successfully")

	return nil
}

// parseCredentialsFile extracts email, password, and created time from credentials file
// DONE
func parseCredentialsFile(content string) (email, password, createdTime string, err error) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "EMAIL:") {
			email = strings.TrimSpace(strings.TrimPrefix(line, "EMAIL:"))
		}
		if strings.HasPrefix(line, "PASSWORD:") {
			password = strings.TrimSpace(strings.TrimPrefix(line, "PASSWORD:"))
		}
		if strings.HasPrefix(line, "CREATED:") {
			createdTime = strings.TrimSpace(strings.TrimPrefix(line, "CREATED:"))
		}
	}

	if email == "" || password == "" {
		return "", "", "", errors.New("invalid credentials file format: missing EMAIL or PASSWORD")
	}

	return email, password, createdTime, nil
}

// writeUpdatedCredentialsFile rewrites the credentials file with updated timestamp
// DONE
func (s *SuperAdminService) writeUpdatedCredentialsFile(email, password, createdTime string, changed bool) error {
	if createdTime == "" {
		createdTime = time.Now().Format("2006-01-02 15:04:05")
	}

	changeConfirmation := ""
	if changed {
		changeConfirmation = "\nPASSWORD CHANGE CONFIRMED: ✓\n"
	}

	content := fmt.Sprintf(`=============================================================
GASSIGEHER - SUPER ADMIN CREDENTIALS
=============================================================

EMAIL: %s
PASSWORD: %s

CREATED: %s
LAST UPDATED: %s%s

=============================================================
HOW TO CHANGE PASSWORD:
=============================================================

1. Edit the PASSWORD line above with your new password
2. Save this file
3. Restart the Gassigeher server
4. Server will hash and save the new password
5. This file will be updated with confirmation

IMPORTANT:
- Keep this file secure (never commit to git)
- This is the ONLY way to change Super Admin password
- Super Admin email cannot be changed (defined in .env)

=============================================================
`, email, password, createdTime, time.Now().Format("2006-01-02 15:04:05"), changeConfirmation)

	return os.WriteFile("SUPER_ADMIN_CREDENTIALS.txt", []byte(content), 0600)
}

// regenerateCredentialsFile generates a new password for Super Admin and creates credentials file
// Used when credentials file is missing but Super Admin exists in database
// DONE
func (s *SuperAdminService) regenerateCredentialsFile() error {
	// Generate new secure password
	newPassword := generateSecurePassword(20)

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update database with new password
	_, err = s.db.Exec("UPDATE users SET password_hash = ?, updated_at = ? WHERE id = 1", string(hashedPassword), time.Now())
	if err != nil {
		return fmt.Errorf("failed to update Super Admin password: %w", err)
	}

	// Create credentials file
	now := time.Now().Format("2006-01-02 15:04:05")
	content := fmt.Sprintf(`=============================================================
GASSIGEHER - SUPER ADMIN CREDENTIALS
=============================================================

EMAIL: %s
PASSWORD: %s

CREATED: %s
LAST UPDATED: %s

IMPORTANT: This password was REGENERATED because the credentials file was missing.

=============================================================
HOW TO CHANGE PASSWORD:
=============================================================

1. Edit the PASSWORD line above with your new password
2. Save this file
3. Restart the Gassigeher server
4. Server will hash and save the new password
5. This file will be updated with confirmation

IMPORTANT:
- Keep this file secure (never commit to git)
- This is the ONLY way to change Super Admin password
- Super Admin email cannot be changed (defined in .env)

=============================================================
`, s.cfg.SuperAdminEmail, newPassword, now, now)

	err = os.WriteFile("SUPER_ADMIN_CREDENTIALS.txt", []byte(content), 0600)
	if err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	// Print to console
	fmt.Println()
	fmt.Println("=============================================================")
	fmt.Println("  SUPER ADMIN CREDENTIALS REGENERATED")
	fmt.Println("=============================================================")
	fmt.Println()
	fmt.Println("The credentials file was missing, so a NEW password was generated:")
	fmt.Println()
	fmt.Printf("  Email:    %s\n", s.cfg.SuperAdminEmail)
	fmt.Printf("  Password: %s\n", newPassword)
	fmt.Println()
	fmt.Println("IMPORTANT:")
	fmt.Println("- New password saved to: SUPER_ADMIN_CREDENTIALS.txt")
	fmt.Println("- Old password is no longer valid")
	fmt.Println("- Change password by editing the file and restarting server")
	fmt.Println()
	fmt.Println("=============================================================")
	fmt.Println()

	log.Println("✓ Super Admin credentials file regenerated successfully")
	return nil
}

// generateSecurePassword generates a secure random password using crypto/rand
// DONE
func generateSecurePassword(length int) string {
	// Character sets for password generation
	lowercase := "abcdefghijklmnopqrstuvwxyz"
	uppercase := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numbers := "0123456789"
	special := "!@#$%^&*"
	allChars := lowercase + uppercase + numbers + special

	password := make([]byte, length)

	// Helper function to get random character from charset
	randomChar := func(charset string) byte {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		return charset[n.Int64()]
	}

	// Ensure at least one of each type
	password[0] = randomChar(lowercase)
	password[1] = randomChar(uppercase)
	password[2] = randomChar(numbers)
	password[3] = randomChar(special)

	// Fill rest randomly
	for i := 4; i < length; i++ {
		password[i] = randomChar(allChars)
	}

	// Shuffle password characters
	for i := len(password) - 1; i > 0; i-- {
		j, _ := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		password[i], password[j.Int64()] = password[j.Int64()], password[i]
	}

	return string(password)
}

// DONE
