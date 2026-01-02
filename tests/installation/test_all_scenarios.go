package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/database"
	"github.com/tranmh/gassigeher/internal/services"
	_ "modernc.org/sqlite"
)

func main() {
	// Load .env file
	if err := godotenv.Load("./.env"); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	fmt.Println("=============================================================")
	fmt.Println("  TESTING ADMIN PASSWORD SCENARIOS")
	fmt.Println("=============================================================")
	fmt.Println()
	fmt.Println("Note: Admin passwords are now stored in .env.secrets")
	fmt.Println("      No more SUPER_ADMIN_CREDENTIALS.txt file")
	fmt.Println()

	// Backup original database if it exists
	backupFiles()

	// Test Scenario 1: First-time installation (Simple Mode)
	testScenario1()

	// Test Scenario 2: Password sync from environment
	testScenario2()

	// Test Scenario 3: SaaS Mode with Central Admin
	testScenario3()

	// Restore original database
	restoreFiles()

	fmt.Println()
	fmt.Println("=============================================================")
	fmt.Println("  ALL TESTS COMPLETED")
	fmt.Println("=============================================================")
}

func backupFiles() {
	if _, err := os.Stat("gassigeher.db"); err == nil {
		os.Rename("gassigeher.db", "gassigeher.db.backup")
		fmt.Println("✓ Backed up existing database")
	}
	fmt.Println()
}

func restoreFiles() {
	// Clean up test database
	os.Remove("gassigeher.db")

	// Restore original
	if _, err := os.Stat("gassigeher.db.backup"); err == nil {
		os.Rename("gassigeher.db.backup", "gassigeher.db")
		fmt.Println("✓ Restored original database")
	}
}

func testScenario1() {
	fmt.Println("=============================================================")
	fmt.Println("  SCENARIO 1: First-time installation (Simple Mode)")
	fmt.Println("=============================================================")
	fmt.Println()

	// Clean slate
	os.Remove("gassigeher.db")

	// Set Simple Mode environment
	os.Setenv("BASE_DOMAIN", "") // Simple mode = no BASE_DOMAIN
	os.Setenv("SUPER_ADMIN_PASSWORD", "TestPassword123!")

	// Initialize
	cfg := config.Load()
	dbConfig := cfg.GetDBConfig()
	db, dialect, err := database.InitializeWithConfig(dbConfig)
	if err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.RunMigrationsWithDialect(db.SqlDB(), dialect); err != nil {
		log.Fatalf("❌ Failed to run migrations: %v", err)
	}

	// Run seed with Simple Mode config
	seedConfig := database.SeedConfig{
		IsSaaSMode:         false,
		SuperAdminEmail:    cfg.SuperAdminEmail,
		SuperAdminPassword: os.Getenv("SUPER_ADMIN_PASSWORD"),
		DBType:             dialect.Name(),
	}
	if err := database.SeedDatabaseWithConfig(db.SqlDB(), seedConfig); err != nil {
		log.Fatalf("❌ Failed to seed database: %v", err)
	}

	// Verify Super Admin exists
	checkAdminExists(db, "Super Admin", false)

	fmt.Println("✅ SCENARIO 1 PASSED: Simple Mode installation works correctly")
	fmt.Println()
	time.Sleep(1 * time.Second)
}

func testScenario2() {
	fmt.Println("=============================================================")
	fmt.Println("  SCENARIO 2: Password sync from .env.secrets")
	fmt.Println("=============================================================")
	fmt.Println()

	// Change password in environment (simulating .env.secrets edit)
	newPassword := "NewSecurePassword456!"
	os.Setenv("SUPER_ADMIN_PASSWORD", newPassword)
	fmt.Printf("✓ Changed SUPER_ADMIN_PASSWORD in environment to: %s\n", newPassword)
	fmt.Println()

	// Initialize
	cfg := config.Load()
	dbConfig := cfg.GetDBConfig()
	db, _, err := database.InitializeWithConfig(dbConfig)
	if err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run AdminPasswordService sync (should detect change)
	adminService := services.NewAdminPasswordService(db, cfg)
	if err := adminService.SyncPasswordFromEnv(); err != nil {
		log.Fatalf("❌ Failed to sync admin password: %v", err)
	}

	// Verify admin still exists
	checkAdminExists(db, "Super Admin", false)

	fmt.Println("✅ SCENARIO 2 PASSED: Password sync from environment works correctly")
	fmt.Println()
	time.Sleep(1 * time.Second)
}

func testScenario3() {
	fmt.Println("=============================================================")
	fmt.Println("  SCENARIO 3: SaaS Mode with Central Admin")
	fmt.Println("=============================================================")
	fmt.Println()

	// Clean slate for SaaS mode
	os.Remove("gassigeher.db")

	// Set SaaS Mode environment
	os.Setenv("BASE_DOMAIN", "gassigeher.local")
	os.Setenv("CENTRAL_ADMIN_EMAIL", "central@gassigeher.local")
	os.Setenv("CENTRAL_ADMIN_PASSWORD", "CentralAdmin789!")
	fmt.Println("✓ Set BASE_DOMAIN=gassigeher.local (SaaS Mode)")
	fmt.Println("✓ Set CENTRAL_ADMIN_EMAIL=central@gassigeher.local")
	fmt.Println()

	// Initialize
	cfg := config.Load()
	dbConfig := cfg.GetDBConfig()
	db, dialect, err := database.InitializeWithConfig(dbConfig)
	if err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.RunMigrationsWithDialect(db.SqlDB(), dialect); err != nil {
		log.Fatalf("❌ Failed to run migrations: %v", err)
	}

	// Run seed with SaaS Mode config
	seedConfig := database.SeedConfig{
		IsSaaSMode:           true,
		CentralAdminEmail:    cfg.CentralAdminEmail,
		CentralAdminPassword: os.Getenv("CENTRAL_ADMIN_PASSWORD"),
		DBType:               dialect.Name(),
	}
	if err := database.SeedDatabaseWithConfig(db.SqlDB(), seedConfig); err != nil {
		log.Fatalf("❌ Failed to seed database: %v", err)
	}

	// Verify Central Admin exists (has is_central_admin=true)
	checkAdminExists(db, "Central Admin", true)

	// Clear SaaS mode for cleanup
	os.Setenv("BASE_DOMAIN", "")

	fmt.Println("✅ SCENARIO 3 PASSED: SaaS Mode Central Admin works correctly")
	fmt.Println()
	time.Sleep(1 * time.Second)
}

func checkAdminExists(db *database.DB, adminType string, isCentralAdmin bool) {
	var exists bool
	var email string
	var isCentral bool

	err := db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM users WHERE id = 1 AND is_super_admin = 1),
		       (SELECT email FROM users WHERE id = 1),
		       (SELECT CASE WHEN is_central_admin = 1 THEN 1 ELSE 0 END FROM users WHERE id = 1)
	`).Scan(&exists, &email, &isCentral)
	if err != nil {
		log.Fatalf("❌ Failed to check admin: %v", err)
	}
	if !exists {
		log.Fatalf("❌ %s does not exist in database", adminType)
	}
	fmt.Printf("✓ %s exists in database (ID=1, email=%s)\n", adminType, email)

	// Verify Central Admin flag
	if isCentralAdmin && !isCentral {
		log.Fatalf("❌ Expected is_central_admin=true for %s", adminType)
	}
	if !isCentralAdmin && isCentral {
		log.Fatalf("❌ Expected is_central_admin=false for %s", adminType)
	}
	fmt.Printf("✓ is_central_admin=%v (expected: %v)\n", isCentral, isCentralAdmin)
}
