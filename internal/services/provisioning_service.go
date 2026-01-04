package services

import (
	"crypto/rand"
	"database/sql"
	"fmt"

	"github.com/tranmh/gassigeher/internal/database"
)

// ProvisioningService handles default data setup for new tenants
type ProvisioningService struct {
	db *database.DB
}

// NewProvisioningService creates a new provisioning service
func NewProvisioningService(db *database.DB) *ProvisioningService {
	return &ProvisioningService{db: db}
}

// CreateDefaultColors creates default color categories for a tenant
// Each color includes a unique pattern_icon for color-blind accessibility
func (s *ProvisioningService) CreateDefaultColors(tx *sql.Tx, tenantID int) error {
	colors := []struct {
		Name        string
		HexCode     string
		PatternIcon string
		SortOrder   int
	}{
		{"Grün", "#22c55e", "circle", 1},
		{"Gelb", "#eab308", "star", 2},
		{"Orange", "#f97316", "triangle", 3},
		{"Rot", "#ef4444", "square", 4},
		{"Blau", "#3b82f6", "diamond", 5},
	}

	// Rebind query for PostgreSQL (? -> $1, $2, ...)
	query := s.db.RebindQuery(`INSERT INTO color_categories (tenant_id, name, hex_code, pattern_icon, sort_order) VALUES (?, ?, ?, ?, ?)`)

	for _, c := range colors {
		_, err := tx.Exec(
			query,
			tenantID, c.Name, c.HexCode, c.PatternIcon, c.SortOrder,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateDefaultBookingRules creates default booking time rules for a tenant
func (s *ProvisioningService) CreateDefaultBookingRules(tx *sql.Tx, tenantID int) error {
	rules := []struct {
		DayType   string
		RuleName  string
		StartTime string
		EndTime   string
		IsBlocked bool
	}{
		{"weekday", "morning", "08:00", "12:00", false},
		{"weekday", "lunch", "12:00", "14:00", true},
		{"weekday", "afternoon", "14:00", "18:00", false},
		{"weekend", "morning", "09:00", "12:00", false},
		{"weekend", "afternoon", "14:00", "17:00", false},
		{"holiday", "morning", "10:00", "12:00", false},
		{"holiday", "afternoon", "14:00", "16:00", false},
	}

	// Rebind query for PostgreSQL (? -> $1, $2, ...)
	query := s.db.RebindQuery(`INSERT INTO booking_time_rules (tenant_id, day_type, rule_name, start_time, end_time, is_blocked) VALUES (?, ?, ?, ?, ?, ?)`)

	for _, r := range rules {
		_, err := tx.Exec(
			query,
			tenantID, r.DayType, r.RuleName, r.StartTime, r.EndTime, s.db.BoolValue(r.IsBlocked),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// generateRegistrationPassword generates a random 8-character alphanumeric password
// Format: uppercase letters and digits (e.g., "AB12CD34")
// SECURITY FIX: Uses rejection sampling to avoid modulo bias
func generateRegistrationPassword() (string, error) {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const charsLen = byte(len(chars)) // 36 characters
	// Calculate the largest multiple of charsLen that fits in a byte
	// 256 / 36 = 7 remainder 4, so 7 * 36 = 252
	const maxUnbiased = byte(252) // Largest value divisible by 36 below 256

	result := make([]byte, 8)
	randomByte := make([]byte, 1)

	for i := 0; i < 8; i++ {
		// Use rejection sampling to avoid modulo bias
		for {
			if _, err := rand.Read(randomByte); err != nil {
				return "", fmt.Errorf("failed to generate random bytes: %w", err)
			}
			// Reject values >= maxUnbiased to ensure uniform distribution
			if randomByte[0] < maxUnbiased {
				result[i] = chars[randomByte[0]%charsLen]
				break
			}
			// Rare case (4/256 ≈ 1.5% chance): try again
		}
	}

	return string(result), nil
}

// CreateDefaultSettings creates default system settings for a tenant
// federalState should be a valid German state code (e.g., "BW", "BY", "NW")
// which will be used for automatic holiday detection via feiertage-api.de
func (s *ProvisioningService) CreateDefaultSettings(tx *sql.Tx, tenantID int, federalState string) error {
	// Generate a random registration password for this tenant
	registrationPassword, err := generateRegistrationPassword()
	if err != nil {
		return fmt.Errorf("failed to generate registration password: %w", err)
	}

	// Default to BW if no federal state provided
	if federalState == "" {
		federalState = "BW"
	}

	settings := map[string]string{
		"booking_advance_days":      "14",
		"cancellation_notice_hours": "12",
		"auto_deactivation_days":    "365",
		"registration_password":     registrationPassword,
		"feiertage_state":           federalState, // Use tenant's federal state for holiday detection
		"use_feiertage_api":         "true",       // Enable holiday API by default
	}

	// Rebind queries for PostgreSQL (? -> $1, $2, ...)
	// Note: "key" is a reserved word, use double quotes for SQLite/PostgreSQL compatibility
	updateQuery := s.db.RebindQuery(`UPDATE system_settings SET value = ? WHERE tenant_id = ? AND "key" = ?`)
	insertQuery := s.db.RebindQuery(`INSERT INTO system_settings (tenant_id, "key", value) VALUES (?, ?, ?)`)

	for key, value := range settings {
		// Use UPDATE-then-INSERT pattern for cross-database compatibility
		// (INSERT OR REPLACE is SQLite-specific, ON CONFLICT is PostgreSQL-specific)
		result, err := tx.Exec(updateQuery, value, tenantID, key)
		if err != nil {
			return err
		}
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			// Row doesn't exist, insert it
			_, err = tx.Exec(insertQuery, tenantID, key, value)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// CreateDefaultSubscription creates a Free plan subscription for a new tenant
func (s *ProvisioningService) CreateDefaultSubscription(tx *sql.Tx, tenantID int) error {
	// Rebind query for PostgreSQL (? -> $1, $2, ...)
	query := s.db.RebindQuery(`INSERT INTO tenant_subscriptions (tenant_id, plan_id, status, created_at, updated_at) VALUES (?, 1, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	_, err := tx.Exec(query, tenantID)
	if err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}
	return nil
}

// AssignAllColorsToUser assigns all tenant colors to a user (for super-admin)
func (s *ProvisioningService) AssignAllColorsToUser(tx *sql.Tx, tenantID, userID int) error {
	// Get all color IDs for this tenant
	query := s.db.RebindQuery(`SELECT id FROM color_categories WHERE tenant_id = ?`)
	rows, err := tx.Query(query, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get colors: %w", err)
	}
	defer rows.Close()

	var colorIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("failed to scan color id: %w", err)
		}
		colorIDs = append(colorIDs, id)
	}

	// Assign each color to the user
	insertQuery := s.db.RebindQuery(`INSERT INTO user_colors (tenant_id, user_id, color_id, granted_by) VALUES (?, ?, ?, ?)`)
	for _, colorID := range colorIDs {
		_, err := tx.Exec(insertQuery, tenantID, userID, colorID, userID)
		if err != nil {
			return fmt.Errorf("failed to assign color %d to user: %w", colorID, err)
		}
	}

	return nil
}

// CreateDemoDog creates a single demo dog for new tenants to test immediately
func (s *ProvisioningService) CreateDemoDog(tx *sql.Tx, tenantID int) error {
	// Get the first (green) color for the demo dog
	query := s.db.RebindQuery(`SELECT id FROM color_categories WHERE tenant_id = ? ORDER BY sort_order ASC LIMIT 1`)
	var colorID int
	err := tx.QueryRow(query, tenantID).Scan(&colorID)
	if err != nil {
		return fmt.Errorf("failed to get first color: %w", err)
	}

	// Create demo dog
	insertQuery := s.db.RebindQuery(`
		INSERT INTO dogs (tenant_id, name, breed, size, age, color_id, is_available, is_featured,
		                  special_needs, pickup_location, walk_route, walk_duration, special_instructions)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	_, err = tx.Exec(insertQuery,
		tenantID,
		"Demo-Hund",                                   // name
		"Mischling",                                   // breed
		"medium",                                      // size
		3,                                             // age
		colorID,                                       // color_id (green - accessible to all)
		s.db.BoolValue(true),                          // is_available
		s.db.BoolValue(true),                          // is_featured
		"Keine besonderen Bedürfnisse",                // special_needs
		"Zwinger 1",                                   // pickup_location
		"Standardrunde um das Tierheim",               // walk_route
		30,                                            // walk_duration
		"Dies ist ein Demo-Hund zum Testen. Sie können diesen Hund bearbeiten oder löschen.", // special_instructions
	)
	if err != nil {
		return fmt.Errorf("failed to create demo dog: %w", err)
	}

	return nil
}

// ProvisionTenant creates all default data for a new tenant
// federalState is used to configure holiday detection for the tenant's region
// adminUserID is the ID of the super-admin user to assign all colors to
func (s *ProvisioningService) ProvisionTenant(tx *sql.Tx, tenantID int, federalState string, adminUserID int) error {
	// Create default color categories
	if err := s.CreateDefaultColors(tx, tenantID); err != nil {
		return fmt.Errorf("CreateDefaultColors failed: %w", err)
	}

	// Assign all colors to the super-admin
	if adminUserID > 0 {
		if err := s.AssignAllColorsToUser(tx, tenantID, adminUserID); err != nil {
			return fmt.Errorf("AssignAllColorsToUser failed: %w", err)
		}
	}

	// Create default booking time rules
	if err := s.CreateDefaultBookingRules(tx, tenantID); err != nil {
		return fmt.Errorf("CreateDefaultBookingRules failed: %w", err)
	}

	// Create default system settings (including feiertage_state from federalState)
	if err := s.CreateDefaultSettings(tx, tenantID, federalState); err != nil {
		return fmt.Errorf("CreateDefaultSettings failed: %w", err)
	}

	// Create Free plan subscription
	if err := s.CreateDefaultSubscription(tx, tenantID); err != nil {
		return fmt.Errorf("CreateDefaultSubscription failed: %w", err)
	}

	// Create a demo dog so the admin can test immediately
	if err := s.CreateDemoDog(tx, tenantID); err != nil {
		return fmt.Errorf("CreateDemoDog failed: %w", err)
	}

	return nil
}
