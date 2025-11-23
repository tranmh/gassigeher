package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/tranm/gassigeher/internal/config"
	"github.com/tranm/gassigeher/internal/database"
	"github.com/tranm/gassigeher/internal/services"
)

func main() {
	// Load .env file
	if err := godotenv.Load("./.env"); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	fmt.Println("=============================================================")
	fmt.Println("  TESTING ALL 3 SUPER ADMIN SCENARIOS")
	fmt.Println("=============================================================")
	fmt.Println()

	// Backup original files if they exist
	backupFiles()

	// Test Scenario 1: First-time installation
	testScenario1()

	// Test Scenario 2: Existing database + missing credentials
	testScenario2()

	// Test Scenario 3: Normal startup with existing credentials
	testScenario3()

	// Restore original files
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
	if _, err := os.Stat("SUPER_ADMIN_CREDENTIALS.txt"); err == nil {
		os.Rename("SUPER_ADMIN_CREDENTIALS.txt", "SUPER_ADMIN_CREDENTIALS.txt.backup")
		fmt.Println("✓ Backed up existing credentials file")
	}
	fmt.Println()
}

func restoreFiles() {
	// Clean up test files
	os.Remove("gassigeher.db")
	os.Remove("SUPER_ADMIN_CREDENTIALS.txt")

	// Restore originals
	if _, err := os.Stat("gassigeher.db.backup"); err == nil {
		os.Rename("gassigeher.db.backup", "gassigeher.db")
		fmt.Println("✓ Restored original database")
	}
	if _, err := os.Stat("SUPER_ADMIN_CREDENTIALS.txt.backup"); err == nil {
		os.Rename("SUPER_ADMIN_CREDENTIALS.txt.backup", "SUPER_ADMIN_CREDENTIALS.txt")
		fmt.Println("✓ Restored original credentials file")
	}
}

func testScenario1() {
	fmt.Println("=============================================================")
	fmt.Println("  SCENARIO 1: First-time installation (empty database)")
	fmt.Println("=============================================================")
	fmt.Println()

	// Clean slate
	os.Remove("gassigeher.db")
	os.Remove("SUPER_ADMIN_CREDENTIALS.txt")

	// Initialize
	cfg := config.Load()
	dbConfig := cfg.GetDBConfig()
	db, dialect, err := database.InitializeWithConfig(dbConfig)
	if err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.RunMigrationsWithDialect(db, dialect); err != nil {
		log.Fatalf("❌ Failed to run migrations: %v", err)
	}

	// Run seed (should create Super Admin and credentials file)
	if err := database.SeedDatabase(db, cfg.SuperAdminEmail); err != nil {
		log.Fatalf("❌ Failed to seed database: %v", err)
	}

	// Verify
	checkSuperAdminExists(db)
	checkCredentialsFileExists()

	fmt.Println("✅ SCENARIO 1 PASSED: First-time installation works correctly")
	fmt.Println()
	time.Sleep(2 * time.Second)
}

func testScenario2() {
	fmt.Println("=============================================================")
	fmt.Println("  SCENARIO 2: Existing database + missing credentials file")
	fmt.Println("=============================================================")
	fmt.Println()

	// Delete only the credentials file
	os.Remove("SUPER_ADMIN_CREDENTIALS.txt")
	fmt.Println("✓ Deleted credentials file (simulating loss)")
	fmt.Println()

	// Initialize
	cfg := config.Load()
	dbConfig := cfg.GetDBConfig()
	db, _, err := database.InitializeWithConfig(dbConfig)
	if err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run SuperAdminService check (should regenerate credentials)
	superAdminService := services.NewSuperAdminService(db, cfg)
	if err := superAdminService.CheckAndUpdatePassword(); err != nil {
		log.Fatalf("❌ Failed to check/update Super Admin: %v", err)
	}

	// Verify
	checkSuperAdminExists(db)
	checkCredentialsFileExists()

	fmt.Println("✅ SCENARIO 2 PASSED: Credentials file regenerated successfully")
	fmt.Println()
	time.Sleep(2 * time.Second)
}

func testScenario3() {
	fmt.Println("=============================================================")
	fmt.Println("  SCENARIO 3: Normal startup with existing credentials file")
	fmt.Println("=============================================================")
	fmt.Println()

	// Both database and credentials file exist from scenario 2
	fmt.Println("✓ Database and credentials file both exist")
	fmt.Println()

	// Initialize
	cfg := config.Load()
	dbConfig := cfg.GetDBConfig()
	db, _, err := database.InitializeWithConfig(dbConfig)
	if err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run SuperAdminService check (should validate and continue)
	superAdminService := services.NewSuperAdminService(db, cfg)
	if err := superAdminService.CheckAndUpdatePassword(); err != nil {
		log.Fatalf("❌ Failed to check Super Admin: %v", err)
	}

	// Verify
	checkSuperAdminExists(db)
	checkCredentialsFileExists()

	fmt.Println("✅ SCENARIO 3 PASSED: Normal startup validation works correctly")
	fmt.Println()
	time.Sleep(2 * time.Second)
}

func checkSuperAdminExists(db *sql.DB) {
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = 1 AND is_super_admin = 1)").Scan(&exists)
	if err != nil {
		log.Fatalf("❌ Failed to check Super Admin: %v", err)
	}
	if !exists {
		log.Fatalf("❌ Super Admin does not exist in database")
	}
	fmt.Println("✓ Super Admin exists in database (ID=1)")
}

func checkCredentialsFileExists() {
	if _, err := os.Stat("SUPER_ADMIN_CREDENTIALS.txt"); os.IsNotExist(err) {
		log.Fatalf("❌ SUPER_ADMIN_CREDENTIALS.txt file does not exist")
	}

	// Read and verify file has content
	content, err := os.ReadFile("SUPER_ADMIN_CREDENTIALS.txt")
	if err != nil {
		log.Fatalf("❌ Failed to read credentials file: %v", err)
	}

	if len(content) < 100 {
		log.Fatalf("❌ Credentials file is too short (possibly corrupt)")
	}

	fmt.Println("✓ SUPER_ADMIN_CREDENTIALS.txt exists and contains data")

	// Display first few lines for verification
	lines := ""
	for i, b := range content {
		lines += string(b)
		if b == '\n' && len(lines) > 200 {
			break
		}
		if i > 500 {
			break
		}
	}
	fmt.Println()
	fmt.Println("File preview:")
	fmt.Println(lines)
}
