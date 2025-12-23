package services

import (
	"crypto/rand"
	"database/sql"
	"fmt"
)

// ProvisioningService handles default data setup for new tenants
type ProvisioningService struct {
	db *sql.DB
}

// NewProvisioningService creates a new provisioning service
func NewProvisioningService(db *sql.DB) *ProvisioningService {
	return &ProvisioningService{db: db}
}

// CreateDefaultColors creates default color categories for a tenant
func (s *ProvisioningService) CreateDefaultColors(tx *sql.Tx, tenantID int) error {
	colors := []struct {
		Name      string
		HexCode   string
		SortOrder int
	}{
		{"Grün", "#22c55e", 1},
		{"Gelb", "#eab308", 2},
		{"Orange", "#f97316", 3},
		{"Rot", "#ef4444", 4},
		{"Blau", "#3b82f6", 5},
	}

	for _, c := range colors {
		_, err := tx.Exec(
			`INSERT INTO color_categories (tenant_id, name, hex_code, sort_order) VALUES (?, ?, ?, ?)`,
			tenantID, c.Name, c.HexCode, c.SortOrder,
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

	for _, r := range rules {
		_, err := tx.Exec(
			`INSERT INTO booking_time_rules (tenant_id, day_type, rule_name, start_time, end_time, is_blocked) VALUES (?, ?, ?, ?, ?, ?)`,
			tenantID, r.DayType, r.RuleName, r.StartTime, r.EndTime, r.IsBlocked,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// generateRegistrationPassword generates a random 8-character alphanumeric password
// Format: uppercase letters and digits (e.g., "AB12CD34")
func generateRegistrationPassword() (string, error) {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, 8)
	randomBytes := make([]byte, 8)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	for i := 0; i < 8; i++ {
		result[i] = chars[randomBytes[i]%byte(len(chars))]
	}

	return string(result), nil
}

// CreateDefaultSettings creates default system settings for a tenant
func (s *ProvisioningService) CreateDefaultSettings(tx *sql.Tx, tenantID int) error {
	// Generate a random registration password for this tenant
	registrationPassword, err := generateRegistrationPassword()
	if err != nil {
		return fmt.Errorf("failed to generate registration password: %w", err)
	}

	settings := map[string]string{
		"booking_advance_days":      "14",
		"cancellation_notice_hours": "12",
		"auto_deactivation_days":    "365",
		"registration_password":     registrationPassword,
	}

	for key, value := range settings {
		// Use INSERT OR REPLACE to handle schema where key alone is unique
		// This handles cases where the original schema hasn't been fully migrated
		// to the (tenant_id, key) composite key.
		// Note: 'key' is a reserved word in SQL, so we use backticks/quotes.
		_, err := tx.Exec(
			"INSERT OR REPLACE INTO system_settings (tenant_id, `key`, value) VALUES (?, ?, ?)",
			tenantID, key, value,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateDefaultSubscription creates a Free plan subscription for a new tenant
func (s *ProvisioningService) CreateDefaultSubscription(tx *sql.Tx, tenantID int) error {
	_, err := tx.Exec(
		`INSERT INTO tenant_subscriptions (tenant_id, plan_id, status, created_at, updated_at) VALUES (?, 1, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		tenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}
	return nil
}

// ProvisionTenant creates all default data for a new tenant
func (s *ProvisioningService) ProvisionTenant(tx *sql.Tx, tenantID int) error {
	// Create default color categories
	if err := s.CreateDefaultColors(tx, tenantID); err != nil {
		return fmt.Errorf("CreateDefaultColors failed: %w", err)
	}

	// Create default booking time rules
	if err := s.CreateDefaultBookingRules(tx, tenantID); err != nil {
		return fmt.Errorf("CreateDefaultBookingRules failed: %w", err)
	}

	// Create default system settings
	if err := s.CreateDefaultSettings(tx, tenantID); err != nil {
		return fmt.Errorf("CreateDefaultSettings failed: %w", err)
	}

	// Create Free plan subscription
	if err := s.CreateDefaultSubscription(tx, tenantID); err != nil {
		return fmt.Errorf("CreateDefaultSubscription failed: %w", err)
	}

	return nil
}
