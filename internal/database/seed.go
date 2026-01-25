package database

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// SeedConfig holds configuration for database seeding
type SeedConfig struct {
	// Mode detection
	IsSaaSMode bool // true = SaaS multi-tenant, false = Simple single-tenant

	// Simple Mode settings
	SuperAdminEmail    string // Email for Super Admin (Simple Mode)
	SuperAdminPassword string // Password from SUPER_ADMIN_PASSWORD env var

	// SaaS Mode settings
	CentralAdminEmail    string // Email for Central Admin (SaaS Mode)
	CentralAdminPassword string // Password from CENTRAL_ADMIN_PASSWORD env var

	// Database dialect
	DBType string // "sqlite" or "postgres"
}

// TestUser holds test user credentials for display
type TestUser struct {
	FirstName string
	LastName  string
	Email     string
	Password  string
	Level     string
}

// EnsureDefaultTenant creates the default tenant (id=0) for Simple-Mode
// This tenant is used when not running in SaaS mode (no BASE_DOMAIN set)
// All tenant_id columns default to 0, referencing this tenant
// MUST be called before any other seed operations to satisfy foreign key constraints
func EnsureDefaultTenant(db *sql.DB, dbType string) error {
	// Check if default tenant already exists
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM tenants WHERE id = 0").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check default tenant: %w", err)
	}

	if count > 0 {
		// Default tenant exists, but check if tenant_settings also exists
		var settingsCount int
		err = db.QueryRow("SELECT COUNT(*) FROM tenant_settings WHERE tenant_id = 0").Scan(&settingsCount)
		if err != nil {
			return fmt.Errorf("failed to check default tenant settings: %w", err)
		}

		if settingsCount == 0 {
			// Create missing tenant_settings for existing default tenant
			now := time.Now()
			var settingsQuery string
			if dbType == "postgres" {
				settingsQuery = `INSERT INTO tenant_settings (tenant_id, theme_preset, created_at, updated_at)
					VALUES (0, 'classic', $1, $2)`
			} else {
				settingsQuery = `INSERT INTO tenant_settings (tenant_id, theme_preset, created_at, updated_at)
					VALUES (0, 'classic', ?, ?)`
			}
			_, err = db.Exec(settingsQuery, now, now)
			if err != nil {
				return fmt.Errorf("failed to create default tenant settings: %w", err)
			}
			log.Println("✓ Created missing tenant settings for default tenant (id=0)")
		}

		return nil // Default tenant already exists
	}

	// Create default tenant with id=0
	now := time.Now()

	var query string
	if dbType == "postgres" {
		query = `INSERT INTO tenants (id, slug, name, contact_email, status, created_at, updated_at)
			VALUES (0, 'default', 'Default Tenant', 'default@localhost', 'active', $1, $2)`
	} else {
		query = `INSERT INTO tenants (id, slug, name, contact_email, status, created_at, updated_at)
			VALUES (0, 'default', 'Default Tenant', 'default@localhost', 'active', ?, ?)`
	}

	_, err = db.Exec(query, now, now)
	if err != nil {
		return fmt.Errorf("failed to create default tenant: %w", err)
	}

	log.Println("✓ Created default tenant (id=0) for Simple-Mode")

	// Create default tenant_settings for the default tenant
	var settingsQuery string
	if dbType == "postgres" {
		settingsQuery = `INSERT INTO tenant_settings (tenant_id, theme_preset, created_at, updated_at)
			VALUES (0, 'classic', $1, $2)`
	} else {
		settingsQuery = `INSERT INTO tenant_settings (tenant_id, theme_preset, created_at, updated_at)
			VALUES (0, 'classic', ?, ?)`
	}

	_, err = db.Exec(settingsQuery, now, now)
	if err != nil {
		return fmt.Errorf("failed to create default tenant settings: %w", err)
	}

	log.Println("✓ Created default tenant settings for Simple-Mode")
	return nil
}

// SeedDatabase generates initial seed data for first-time installations
// Only runs if users table is empty
// Set SKIP_SEED=true to skip (useful for E2E tests that manage their own data)
//
// Deprecated: Use SeedDatabaseWithConfig for new code
func SeedDatabase(db *sql.DB, superAdminEmail string, dbType ...string) error {
	// Determine database type (default to sqlite for backward compatibility)
	dialect := "sqlite"
	if len(dbType) > 0 && dbType[0] != "" {
		dialect = dbType[0]
	}

	// Get password from environment (new approach)
	// Try SUPER_ADMIN_PASSWORD first, fall back to generating one
	password := os.Getenv("SUPER_ADMIN_PASSWORD")
	if password == "" {
		password = generateSecurePassword(20)
		log.Println("Warning: SUPER_ADMIN_PASSWORD not set in .env.secrets, generated random password")
	}

	// Detect SaaS mode from environment
	isSaaSMode := os.Getenv("BASE_DOMAIN") != ""

	config := SeedConfig{
		IsSaaSMode:           isSaaSMode,
		SuperAdminEmail:      superAdminEmail,
		SuperAdminPassword:   password,
		CentralAdminEmail:    os.Getenv("CENTRAL_ADMIN_EMAIL"),
		CentralAdminPassword: os.Getenv("CENTRAL_ADMIN_PASSWORD"),
		DBType:               dialect,
	}

	return SeedDatabaseWithConfig(db, config)
}

// SeedDatabaseWithConfig generates initial seed data with explicit configuration
// This is the preferred method for new code
func SeedDatabaseWithConfig(db *sql.DB, cfg SeedConfig) error {
	// Check if seeding is disabled (for E2E tests)
	if os.Getenv("SKIP_SEED") == "true" {
		log.Println("SKIP_SEED=true, skipping seed data generation")
		return nil
	}

	// Ensure default tenant exists (for foreign key constraints)
	if err := EnsureDefaultTenant(db, cfg.DBType); err != nil {
		return fmt.Errorf("failed to ensure default tenant: %w", err)
	}

	// Check if users table is empty
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check users count: %w", err)
	}

	if count > 0 {
		log.Println("Database already seeded, skipping seed data generation")
		return nil
	}

	log.Println("Empty database detected, generating seed data...")

	if cfg.IsSaaSMode {
		return seedSaaSMode(db, cfg)
	}
	return seedSimpleMode(db, cfg)
}

// seedSimpleMode creates the Super Admin for a single-tenant deployment
func seedSimpleMode(db *sql.DB, cfg SeedConfig) error {
	// Validate Super Admin email
	if cfg.SuperAdminEmail == "" {
		return fmt.Errorf("SUPER_ADMIN_EMAIL not set - cannot create Super Admin for Simple Mode")
	}

	// Get or generate password
	password := cfg.SuperAdminPassword
	if password == "" {
		password = generateSecurePassword(20)
		log.Println("Warning: SUPER_ADMIN_PASSWORD not set, generated random password")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash Super Admin password: %w", err)
	}

	now := time.Now()

	// Insert Super Admin
	var insertQuery string
	if cfg.DBType == "postgres" {
		insertQuery = `
			INSERT INTO users (
				id, tenant_id, first_name, last_name, email, password_hash,
				is_admin, is_super_admin, is_central_admin, is_verified, is_active,
				terms_accepted_at, last_activity_at, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`
	} else {
		insertQuery = `
			INSERT INTO users (
				id, tenant_id, first_name, last_name, email, password_hash,
				is_admin, is_super_admin, is_central_admin, is_verified, is_active,
				terms_accepted_at, last_activity_at, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	}

	// Super Admin for Simple Mode: is_super_admin=true, is_central_admin=false
	_, err = db.Exec(insertQuery,
		1, 0, "Super", "Admin", cfg.SuperAdminEmail, string(hashedPassword),
		true, true, false, true, true, now, now, now, now)
	if err != nil {
		return fmt.Errorf("failed to create Super Admin: %w", err)
	}

	// Reset PostgreSQL sequence after explicit ID insert
	if cfg.DBType == "postgres" {
		_, err = db.Exec("SELECT setval('users_id_seq', (SELECT COALESCE(MAX(id), 1) FROM users))")
		if err != nil {
			log.Printf("Warning: failed to reset users_id_seq: %v", err)
		}
	}

	log.Println("✓ Super Admin created (ID: 1) - Local shelter administrator")

	// Print setup complete message
	printSimpleModeSetup(cfg.SuperAdminEmail, password)

	log.Println("✓ Seed data generation completed successfully (Simple Mode)")
	return nil
}

// seedSaaSMode creates the Central Admin for a multi-tenant deployment
func seedSaaSMode(db *sql.DB, cfg SeedConfig) error {
	// Validate Central Admin email
	if cfg.CentralAdminEmail == "" {
		return fmt.Errorf("CENTRAL_ADMIN_EMAIL not set - cannot create Central Admin for SaaS Mode")
	}

	// Get or generate password
	password := cfg.CentralAdminPassword
	if password == "" {
		password = generateSecurePassword(20)
		log.Println("Warning: CENTRAL_ADMIN_PASSWORD not set, generated random password")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash Central Admin password: %w", err)
	}

	now := time.Now()

	// Insert Central Admin
	var insertQuery string
	if cfg.DBType == "postgres" {
		insertQuery = `
			INSERT INTO users (
				id, tenant_id, first_name, last_name, email, password_hash,
				is_admin, is_super_admin, is_central_admin, is_verified, is_active,
				terms_accepted_at, last_activity_at, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`
	} else {
		insertQuery = `
			INSERT INTO users (
				id, tenant_id, first_name, last_name, email, password_hash,
				is_admin, is_super_admin, is_central_admin, is_verified, is_active,
				terms_accepted_at, last_activity_at, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	}

	// Central Admin for SaaS Mode: is_super_admin=true, is_central_admin=true
	_, err = db.Exec(insertQuery,
		1, 0, "Central", "Admin", cfg.CentralAdminEmail, string(hashedPassword),
		true, true, true, true, true, now, now, now, now)
	if err != nil {
		return fmt.Errorf("failed to create Central Admin: %w", err)
	}

	// Reset PostgreSQL sequence after explicit ID insert
	if cfg.DBType == "postgres" {
		_, err = db.Exec("SELECT setval('users_id_seq', (SELECT COALESCE(MAX(id), 1) FROM users))")
		if err != nil {
			log.Printf("Warning: failed to reset users_id_seq: %v", err)
		}
	}

	log.Println("✓ Central Admin created (ID: 1) - Platform administrator for SaaS")

	// Note: Demo tenant is created by DemoSeedService.EnsureDemoTenant() in main.go
	log.Println("✓ Demo tenant will be created by DemoSeedService on startup")

	// Print setup complete message
	printSaaSModeSetup(cfg.CentralAdminEmail, password)

	log.Println("✓ Seed data generation completed successfully (SaaS Mode)")
	return nil
}

// generateSecurePassword generates a cryptographically secure random password
// Uses crypto/rand for unpredictable random bytes
func generateSecurePassword(length int) string {
	lowercase := "abcdefghijklmnopqrstuvwxyz"
	uppercase := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numbers := "0123456789"
	special := "!@#$%^&*"
	allChars := lowercase + uppercase + numbers + special

	password := make([]byte, length)

	secureRandomIndex := func(max int) int {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
		if err != nil {
			log.Printf("CRITICAL: crypto/rand failed during password generation: %v", err)
			panic("crypto/rand failed: " + err.Error())
		}
		return int(n.Int64())
	}

	// Ensure at least one of each type
	password[0] = lowercase[secureRandomIndex(len(lowercase))]
	password[1] = uppercase[secureRandomIndex(len(uppercase))]
	password[2] = numbers[secureRandomIndex(len(numbers))]
	password[3] = special[secureRandomIndex(len(special))]

	// Fill rest randomly
	for i := 4; i < length; i++ {
		password[i] = allChars[secureRandomIndex(len(allChars))]
	}

	// Shuffle using Fisher-Yates with crypto/rand
	for i := len(password) - 1; i > 0; i-- {
		j := secureRandomIndex(i + 1)
		password[i], password[j] = password[j], password[i]
	}

	return string(password)
}

// printSimpleModeSetup prints setup info for Simple Mode
func printSimpleModeSetup(email, password string) {
	fmt.Println()
	fmt.Println("=============================================================")
	fmt.Println("  GASSIGEHER - INSTALLATION COMPLETE (Simple Mode)")
	fmt.Println("=============================================================")
	fmt.Println()
	fmt.Println("SUPER ADMIN CREDENTIALS:")
	fmt.Printf("  Email:    %s\n", email)
	fmt.Printf("  Password: %s\n", password)
	fmt.Println()
	fmt.Println("ACCESS:")
	fmt.Println("  Application: http://localhost:8080")
	fmt.Println("  Admin Panel: http://localhost:8080/admin-dashboard.html")
	fmt.Println()
	fmt.Println("IMPORTANT:")
	fmt.Println("- Password is stored in .env.secrets as SUPER_ADMIN_PASSWORD")
	fmt.Println("- To change password: edit .env.secrets and restart server")
	fmt.Println("- Keep .env.secrets secure and never commit to git")
	fmt.Println()
	fmt.Println("=============================================================")
	fmt.Println()
}

// printSaaSModeSetup prints setup info for SaaS Mode
func printSaaSModeSetup(email, password string) {
	fmt.Println()
	fmt.Println("=============================================================")
	fmt.Println("  GASSIGEHER SaaS - INSTALLATION COMPLETE (SaaS Mode)")
	fmt.Println("=============================================================")
	fmt.Println()
	fmt.Println("CENTRAL ADMIN CREDENTIALS (Platform Administrator):")
	fmt.Printf("  Email:    %s\n", email)
	fmt.Printf("  Password: %s\n", password)
	fmt.Println()
	fmt.Println("ACCESS POINTS:")
	fmt.Println("  Landing Page:    http://gassigeher.local:8080/landing/")
	fmt.Println("  Central Admin:   http://gassigeher.local:8080/central/")
	fmt.Println()
	fmt.Println("NEXT STEPS:")
	fmt.Println("  1. Login at /central/ with Central Admin credentials")
	fmt.Println("  2. Create tenants (animal shelters)")
	fmt.Println("  3. Each tenant gets their own Super Admin on registration")
	fmt.Println()
	fmt.Println("IMPORTANT:")
	fmt.Println("- Password is stored in .env.secrets as CENTRAL_ADMIN_PASSWORD")
	fmt.Println("- To change password: edit .env.secrets and restart server")
	fmt.Println("- Keep .env.secrets secure and never commit to git")
	fmt.Println()
	fmt.Println("=============================================================")
	fmt.Println()
}

// Helper functions for creating pointers to values
func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
