// cmd/migrate-to-saas/main.go
// Migration tool to convert existing single-tenant Gassigeher data to multi-tenant SaaS
//
// Usage:
//   go run cmd/migrate-to-saas/main.go -name "Tierheim Göppingen" -slug "tierheim-goeppingen" -email "admin@tierheim-goeppingen.de"
//
// This script:
// 1. Creates a new tenant with the specified name and slug
// 2. Updates all existing records to belong to the new tenant
// 3. Creates default tenant settings
// 4. Promotes the first super admin to central admin (optional)

package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Parse command line flags
	tenantName := flag.String("name", "", "Tenant name (e.g., 'Tierheim Göppingen')")
	tenantSlug := flag.String("slug", "", "Tenant slug for subdomain (e.g., 'tierheim-goeppingen')")
	contactEmail := flag.String("email", "", "Contact email for tenant")
	federalState := flag.String("state", "BW", "Federal state code for holidays (default: BW)")
	promoteCentralAdmin := flag.Bool("promote-central-admin", true, "Promote super admin to central admin")
	dryRun := flag.Bool("dry-run", false, "Show what would be done without making changes")
	baseDomain := flag.String("domain", "example.com", "Base domain for tenant URLs (e.g., 'gassigeher.org')")
	dbType := flag.String("db-type", "sqlite", "Database type: sqlite, postgres, mysql")
	dbPath := flag.String("db-path", "./gassigeher.db", "SQLite database path")
	dbURL := flag.String("db-url", "", "Database connection URL (for postgres/mysql)")

	flag.Parse()

	// Validate required flags
	if *tenantName == "" || *tenantSlug == "" || *contactEmail == "" {
		fmt.Println("Gassigeher Single-to-Multi Tenant Migration Tool")
		fmt.Println("================================================")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  go run cmd/migrate-to-saas/main.go -name <name> -slug <slug> -email <email>")
		fmt.Println()
		fmt.Println("Required flags:")
		fmt.Println("  -name    Tenant name (e.g., 'Tierheim Göppingen')")
		fmt.Println("  -slug    Subdomain slug (e.g., 'tierheim-goeppingen')")
		fmt.Println("  -email   Contact email")
		fmt.Println()
		fmt.Println("Optional flags:")
		fmt.Println("  -state                  Federal state (default: BW)")
		fmt.Println("  -promote-central-admin  Promote super admin (default: true)")
		fmt.Println("  -dry-run                Show changes without applying")
		fmt.Println("  -db-type                Database type (default: sqlite)")
		fmt.Println("  -db-path                SQLite path (default: ./gassigeher.db)")
		fmt.Println("  -db-url                 Connection URL for postgres/mysql")
		os.Exit(1)
	}

	// Validate slug format
	slug := strings.ToLower(*tenantSlug)
	if !isValidSlug(slug) {
		log.Fatal("Invalid slug format. Use lowercase letters, numbers, and hyphens only (3-100 chars)")
	}

	// Connect to database
	var db *sql.DB
	var err error

	switch *dbType {
	case "sqlite":
		db, err = sql.Open("sqlite3", *dbPath)
	case "postgres":
		if *dbURL == "" {
			*dbURL = os.Getenv("DATABASE_URL")
		}
		if *dbURL == "" {
			log.Fatal("Database URL required for PostgreSQL (use -db-url or DATABASE_URL env)")
		}
		db, err = sql.Open("postgres", *dbURL)
	case "mysql":
		if *dbURL == "" {
			*dbURL = os.Getenv("DATABASE_URL")
		}
		if *dbURL == "" {
			log.Fatal("Database URL required for MySQL (use -db-url or DATABASE_URL env)")
		}
		db, err = sql.Open("mysql", *dbURL)
	default:
		log.Fatalf("Unsupported database type: %s", *dbType)
	}

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Verify connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Connected to database successfully")

	if *dryRun {
		log.Println("[DRY RUN] No changes will be made")
	}

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Failed to start transaction: %v", err)
	}
	defer tx.Rollback()

	// Check if tenant slug already exists
	var existingCount int
	err = tx.QueryRow("SELECT COUNT(*) FROM tenants WHERE slug = ?", slug).Scan(&existingCount)
	if err == nil && existingCount > 0 {
		log.Fatalf("Tenant with slug '%s' already exists", slug)
	}

	// Check for existing data without tenant_id
	tables := []string{
		"users", "dogs", "bookings", "blocked_dates",
		"color_categories", "user_colors", "color_requests",
		"experience_requests", "reactivation_requests",
		"system_settings", "booking_time_rules", "custom_holidays",
	}

	log.Println("\n=== Pre-migration Analysis ===")
	totalRecords := 0
	for _, table := range tables {
		var count int
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE tenant_id IS NULL OR tenant_id = 0", table)
		err := tx.QueryRow(query).Scan(&count)
		if err != nil {
			log.Printf("  %s: error checking - %v", table, err)
			continue
		}
		log.Printf("  %s: %d records to migrate", table, count)
		totalRecords += count
	}
	log.Printf("  TOTAL: %d records\n", totalRecords)

	if *dryRun {
		log.Println("\n[DRY RUN] Would create tenant and migrate records")
		log.Printf("  Tenant Name: %s", *tenantName)
		log.Printf("  Tenant Slug: %s", slug)
		log.Printf("  Contact Email: %s", *contactEmail)
		log.Printf("  Federal State: %s", *federalState)
		log.Printf("  URL: https://%s.%s", slug, *baseDomain)
		return
	}

	// 1. Create tenant
	log.Println("\n=== Creating Tenant ===")
	var tenantID int64
	result, err := tx.Exec(`
		INSERT INTO tenants (slug, name, status, contact_email, federal_state, created_at, updated_at)
		VALUES (?, ?, 'active', ?, ?, datetime('now'), datetime('now'))
	`, slug, *tenantName, *contactEmail, *federalState)
	if err != nil {
		log.Fatalf("Failed to create tenant: %v", err)
	}
	tenantID, _ = result.LastInsertId()
	log.Printf("Created tenant ID: %d", tenantID)

	// 2. Create tenant settings
	log.Println("\n=== Creating Tenant Settings ===")
	_, err = tx.Exec(`
		INSERT INTO tenant_settings (tenant_id, theme_preset, created_at, updated_at)
		VALUES (?, 'classic', datetime('now'), datetime('now'))
	`, tenantID)
	if err != nil {
		log.Printf("Warning: Failed to create tenant settings: %v", err)
	} else {
		log.Println("Created default tenant settings (classic theme)")
	}

	// 3. Migrate all existing records
	log.Println("\n=== Migrating Records ===")
	for _, table := range tables {
		query := fmt.Sprintf("UPDATE %s SET tenant_id = ? WHERE tenant_id IS NULL OR tenant_id = 0", table)
		result, err := tx.Exec(query, tenantID)
		if err != nil {
			log.Printf("  %s: ERROR - %v", table, err)
			continue
		}
		rows, _ := result.RowsAffected()
		if rows > 0 {
			log.Printf("  %s: migrated %d records", table, rows)
		}
	}

	// 4. Optionally promote super admin to central admin
	if *promoteCentralAdmin {
		log.Println("\n=== Promoting Super Admin to Central Admin ===")
		result, err := tx.Exec(`
			UPDATE users SET is_central_admin = 1 WHERE is_super_admin = 1
		`)
		if err != nil {
			log.Printf("Warning: Failed to promote super admin: %v", err)
		} else {
			rows, _ := result.RowsAffected()
			log.Printf("Promoted %d super admin(s) to central admin", rows)
		}
	}

	// 5. Commit transaction
	log.Println("\n=== Committing Changes ===")
	if err := tx.Commit(); err != nil {
		log.Fatalf("Failed to commit transaction: %v", err)
	}

	log.Println("\n=== Migration Complete ===")
	log.Printf("Tenant Name:  %s", *tenantName)
	log.Printf("Tenant Slug:  %s", slug)
	log.Printf("Tenant ID:    %d", tenantID)
	log.Printf("Tenant URL:   https://%s.%s", slug, *baseDomain)
	log.Println("\nNext steps:")
	log.Printf("1. Update BASE_DOMAIN in .env to '%s'", *baseDomain)
	log.Printf("2. Configure DNS wildcard record: *.%s -> your server", *baseDomain)
	log.Println("3. Start the server with the new configuration")
	log.Printf("4. Access your tenant at: https://%s.%s", slug, *baseDomain)
}

// isValidSlug validates tenant slug format
func isValidSlug(slug string) bool {
	if len(slug) < 3 || len(slug) > 100 {
		return false
	}
	for _, c := range slug {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return false
		}
	}
	// Cannot start or end with hyphen
	if slug[0] == '-' || slug[len(slug)-1] == '-' {
		return false
	}
	return true
}
