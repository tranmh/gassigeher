package services

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/database"
	"golang.org/x/crypto/bcrypt"
)

// AdminPasswordService handles admin password synchronization from .env.secrets
// In Simple Mode: syncs SUPER_ADMIN_PASSWORD
// In SaaS Mode: syncs CENTRAL_ADMIN_PASSWORD
type AdminPasswordService struct {
	db  *database.DB
	cfg *config.Config
}

// NewAdminPasswordService creates a new AdminPasswordService
func NewAdminPasswordService(db *database.DB, cfg *config.Config) *AdminPasswordService {
	return &AdminPasswordService{
		db:  db,
		cfg: cfg,
	}
}

// SyncPasswordFromEnv reads password from environment and updates database if changed
// This runs on every server startup to detect password changes in .env.secrets
// Returns nil if no admin exists yet (first startup - seed.go will create them)
func (s *AdminPasswordService) SyncPasswordFromEnv() error {
	// Determine which password to check based on mode
	var envVarName string
	var adminType string

	if s.cfg.IsSaaSMode() {
		envVarName = "CENTRAL_ADMIN_PASSWORD"
		adminType = "Central Admin"
	} else {
		envVarName = "SUPER_ADMIN_PASSWORD"
		adminType = "Super Admin"
	}

	// Get password from environment (.env.secrets)
	password := os.Getenv(envVarName)
	if password == "" {
		// Password not set - this is okay for first-time installation
		// seed.go will generate one if needed
		log.Printf("%s password not set in .env.secrets (%s)", adminType, envVarName)
		return nil
	}

	// Check if admin exists in database (ID=1)
	var currentHash string
	err := s.db.QueryRow("SELECT password_hash FROM users WHERE id = 1").Scan(&currentHash)
	if err != nil {
		// Admin doesn't exist yet - this is okay for first-time installation
		log.Printf("%s not found in database (first startup)", adminType)
		return nil
	}

	// Check if password has changed
	if bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(password)) == nil {
		// Password unchanged, no action needed
		log.Printf("%s password unchanged", adminType)
		return nil
	}

	// Password changed! Hash new password and update database
	log.Printf("%s password change detected in .env.secrets, updating...", adminType)
	newHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	updateQuery := s.db.Rebind("UPDATE users SET password_hash = ?, updated_at = ? WHERE id = 1")
	_, err = s.db.Exec(updateQuery, string(newHash), time.Now())
	if err != nil {
		return fmt.Errorf("failed to update password in database: %w", err)
	}

	log.Printf("✓ %s password updated successfully from .env.secrets", adminType)
	return nil
}

// GetAdminEmail returns the appropriate admin email based on mode
func (s *AdminPasswordService) GetAdminEmail() string {
	return s.cfg.GetAdminEmail()
}

// IsSaaSMode returns true if running in SaaS mode
func (s *AdminPasswordService) IsSaaSMode() bool {
	return s.cfg.IsSaaSMode()
}
