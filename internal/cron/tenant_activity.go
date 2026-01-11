package cron

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/tranmh/gassigeher/internal/database"
)

// parseTimestampString parses a timestamp string from the database.
// It handles multiple formats including the legacy Go format with monotonic clock suffix.
func parseTimestampString(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}

	// Try RFC3339 first (most common after fix)
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	// Try RFC3339Nano
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}

	// Handle legacy format with monotonic clock suffix
	// Format: "2025-12-24 14:30:00.123456789 +0100 CET m=+123.456"
	if strings.Contains(s, " m=") {
		// Remove the monotonic clock suffix
		parts := strings.Split(s, " m=")
		cleaned := parts[0]

		// Try parsing the cleaned string
		layouts := []string{
			"2006-01-02 15:04:05.999999999 -0700 MST",
			"2006-01-02 15:04:05.999999999 -0700",
			"2006-01-02 15:04:05 -0700 MST",
			"2006-01-02 15:04:05 -0700",
		}

		for _, layout := range layouts {
			if t, err := time.Parse(layout, cleaned); err == nil {
				return t, nil
			}
		}
	}

	// Try some other common formats
	additionalLayouts := []string{
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, layout := range additionalLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse timestamp: %s", s)
}

// TenantActivityChecker handles checking tenant activity and flagging inactive tenants
type TenantActivityChecker struct {
	db              *database.DB
	inactivityDays  int // Number of days without activity to be considered inactive
}

// NewTenantActivityChecker creates a new tenant activity checker
func NewTenantActivityChecker(db *database.DB, inactivityDays int) *TenantActivityChecker {
	if inactivityDays <= 0 {
		inactivityDays = 30 // Default: 30 days
	}
	return &TenantActivityChecker{
		db:             db,
		inactivityDays: inactivityDays,
	}
}

// TenantActivity represents activity information for a tenant
type TenantActivity struct {
	TenantID         int        `json:"tenant_id"`
	TenantSlug       string     `json:"tenant_slug"`
	TenantName       string     `json:"tenant_name"`
	LastBookingDate  *time.Time `json:"last_booking_date,omitempty"`
	LastUserLogin    *time.Time `json:"last_user_login,omitempty"`
	DaysInactive     int        `json:"days_inactive"`
	TotalBookings    int        `json:"total_bookings"`
	ActiveUsers      int        `json:"active_users"`
	IsInactive       bool       `json:"is_inactive"`
}

// CheckAndFlagInactiveTenants checks all tenants for inactivity
// This is run as a daily cron job
// It updates the tenant's inactivity_flagged_at field to track when they were flagged
func (c *TenantActivityChecker) CheckAndFlagInactiveTenants() error {
	log.Println("Starting tenant activity check...")

	// Get all active tenants
	query := `
		SELECT id, slug, name
		FROM tenants
		WHERE status = 'active'
	`

	rows, err := c.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	var inactiveCount int
	var flaggedCount int
	cutoffDate := time.Now().AddDate(0, 0, -c.inactivityDays)

	// Collect tenants to flag (avoid holding row lock while updating)
	type tenantInfo struct {
		id   int
		slug string
		name string
	}
	var tenantsToFlag []tenantInfo

	for rows.Next() {
		var tenantID int
		var slug, name string

		if err := rows.Scan(&tenantID, &slug, &name); err != nil {
			log.Printf("Error scanning tenant: %v", err)
			continue
		}

		// Check last booking date for this tenant
		// BUG FIX: Check and log database errors instead of ignoring them
		var lastBooking *time.Time
		bookingQuery := c.db.Rebind(`
			SELECT MAX(created_at)
			FROM bookings
			WHERE tenant_id = ?
		`)
		if err := c.db.QueryRow(bookingQuery, tenantID).Scan(&lastBooking); err != nil && err != sql.ErrNoRows {
			log.Printf("Error querying bookings for tenant %d: %v", tenantID, err)
			continue // Skip this tenant due to database error
		}

		// Check last user activity for this tenant
		// BUG FIX: Check and log database errors instead of ignoring them
		var lastActivity *time.Time
		activityQuery := c.db.Rebind(`
			SELECT MAX(last_activity_at)
			FROM users
			WHERE tenant_id = ? AND is_active = ?
		`)
		if err := c.db.QueryRow(activityQuery, tenantID, c.db.BoolValue(true)).Scan(&lastActivity); err != nil && err != sql.ErrNoRows {
			log.Printf("Error querying user activity for tenant %d: %v", tenantID, err)
			continue // Skip this tenant due to database error
		}

		// Determine the most recent activity
		var mostRecentActivity *time.Time
		if lastBooking != nil && lastActivity != nil {
			if lastBooking.After(*lastActivity) {
				mostRecentActivity = lastBooking
			} else {
				mostRecentActivity = lastActivity
			}
		} else if lastBooking != nil {
			mostRecentActivity = lastBooking
		} else if lastActivity != nil {
			mostRecentActivity = lastActivity
		}

		// Check if tenant is inactive
		isInactive := mostRecentActivity == nil || mostRecentActivity.Before(cutoffDate)

		if isInactive {
			inactiveCount++
			tenantsToFlag = append(tenantsToFlag, tenantInfo{id: tenantID, slug: slug, name: name})
			log.Printf("Tenant '%s' (ID: %d) identified as inactive - last activity: %v",
				slug, tenantID, mostRecentActivity)
		}
	}

	// Check for errors during row iteration
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating tenant rows: %w", err)
	}

	// Flag inactive tenants in the database
	// BUG FIX: Update inactivity_flagged_at (not just updated_at)
	flagQuery := c.db.Rebind(`
		UPDATE tenants
		SET inactivity_flagged_at = ?
		WHERE id = ? AND status = 'active'
	`)
	now := time.Now()

	for _, tenant := range tenantsToFlag {
		_, err := c.db.Exec(flagQuery, now, tenant.id)
		if err != nil {
			log.Printf("Error flagging tenant %s (ID: %d): %v", tenant.slug, tenant.id, err)
			continue
		}
		flaggedCount++
		log.Printf("Tenant '%s' (ID: %d) flagged as inactive in database", tenant.slug, tenant.id)
	}

	log.Printf("Tenant activity check complete. Found %d inactive tenants, flagged %d", inactiveCount, flaggedCount)
	return nil
}

// GetInactiveTenants returns a list of tenants with no recent activity
func (c *TenantActivityChecker) GetInactiveTenants() ([]TenantActivity, error) {
	cutoffDate := time.Now().AddDate(0, 0, -c.inactivityDays)

	// Use parameterized boolean for PostgreSQL compatibility
	isActiveTrue := c.db.BoolValue(true)
	query := c.db.Rebind(`
		SELECT
			t.id,
			t.slug,
			t.name,
			(SELECT MAX(created_at) FROM bookings WHERE tenant_id = t.id) as last_booking,
			(SELECT MAX(last_activity_at) FROM users WHERE tenant_id = t.id AND is_active = ?) as last_login,
			(SELECT COUNT(*) FROM bookings WHERE tenant_id = t.id) as total_bookings,
			(SELECT COUNT(*) FROM users WHERE tenant_id = t.id AND is_active = ?) as active_users
		FROM tenants t
		WHERE t.status = 'active'
		ORDER BY t.name
	`)

	rows, err := c.db.Query(query, isActiveTrue, isActiveTrue)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []TenantActivity

	for rows.Next() {
		var activity TenantActivity
		// Scan timestamps as strings to handle legacy format with monotonic clock suffix
		var lastBookingStr, lastLoginStr *string

		if err := rows.Scan(
			&activity.TenantID,
			&activity.TenantSlug,
			&activity.TenantName,
			&lastBookingStr,
			&lastLoginStr,
			&activity.TotalBookings,
			&activity.ActiveUsers,
		); err != nil {
			return nil, err
		}

		// Parse timestamps using helper that handles legacy format
		var lastBooking, lastLogin *time.Time
		if lastBookingStr != nil {
			if t, err := parseTimestampString(*lastBookingStr); err == nil {
				lastBooking = &t
			}
		}
		if lastLoginStr != nil {
			if t, err := parseTimestampString(*lastLoginStr); err == nil {
				lastLogin = &t
			}
		}

		activity.LastBookingDate = lastBooking
		activity.LastUserLogin = lastLogin

		// Calculate days inactive from most recent activity
		var mostRecentActivity *time.Time
		if lastBooking != nil && lastLogin != nil {
			if lastBooking.After(*lastLogin) {
				mostRecentActivity = lastBooking
			} else {
				mostRecentActivity = lastLogin
			}
		} else if lastBooking != nil {
			mostRecentActivity = lastBooking
		} else if lastLogin != nil {
			mostRecentActivity = lastLogin
		}

		if mostRecentActivity != nil {
			activity.DaysInactive = int(time.Since(*mostRecentActivity).Hours() / 24)
		} else {
			activity.DaysInactive = 999 // No activity ever
		}

		activity.IsInactive = mostRecentActivity == nil || mostRecentActivity.Before(cutoffDate)

		// Only include inactive tenants
		if activity.IsInactive {
			results = append(results, activity)
		}
	}

	return results, nil
}

// GetAllTenantActivity returns activity info for all tenants
func (c *TenantActivityChecker) GetAllTenantActivity() ([]TenantActivity, error) {
	// Use parameterized boolean for PostgreSQL compatibility
	isActiveTrue := c.db.BoolValue(true)
	query := c.db.Rebind(`
		SELECT
			t.id,
			t.slug,
			t.name,
			(SELECT MAX(created_at) FROM bookings WHERE tenant_id = t.id) as last_booking,
			(SELECT MAX(last_activity_at) FROM users WHERE tenant_id = t.id AND is_active = ?) as last_login,
			(SELECT COUNT(*) FROM bookings WHERE tenant_id = t.id) as total_bookings,
			(SELECT COUNT(*) FROM users WHERE tenant_id = t.id AND is_active = ?) as active_users
		FROM tenants t
		WHERE t.status = 'active'
		ORDER BY t.name
	`)

	rows, err := c.db.Query(query, isActiveTrue, isActiveTrue)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cutoffDate := time.Now().AddDate(0, 0, -c.inactivityDays)
	var results []TenantActivity

	for rows.Next() {
		var activity TenantActivity
		// Scan timestamps as strings to handle legacy format with monotonic clock suffix
		var lastBookingStr, lastLoginStr *string

		if err := rows.Scan(
			&activity.TenantID,
			&activity.TenantSlug,
			&activity.TenantName,
			&lastBookingStr,
			&lastLoginStr,
			&activity.TotalBookings,
			&activity.ActiveUsers,
		); err != nil {
			return nil, err
		}

		// Parse timestamps using helper that handles legacy format
		var lastBooking, lastLogin *time.Time
		if lastBookingStr != nil {
			if t, err := parseTimestampString(*lastBookingStr); err == nil {
				lastBooking = &t
			}
		}
		if lastLoginStr != nil {
			if t, err := parseTimestampString(*lastLoginStr); err == nil {
				lastLogin = &t
			}
		}

		activity.LastBookingDate = lastBooking
		activity.LastUserLogin = lastLogin

		// Calculate days inactive from most recent activity
		var mostRecentActivity *time.Time
		if lastBooking != nil && lastLogin != nil {
			if lastBooking.After(*lastLogin) {
				mostRecentActivity = lastBooking
			} else {
				mostRecentActivity = lastLogin
			}
		} else if lastBooking != nil {
			mostRecentActivity = lastBooking
		} else if lastLogin != nil {
			mostRecentActivity = lastLogin
		}

		if mostRecentActivity != nil {
			activity.DaysInactive = int(time.Since(*mostRecentActivity).Hours() / 24)
		} else {
			activity.DaysInactive = 999 // No activity ever
		}

		activity.IsInactive = mostRecentActivity == nil || mostRecentActivity.Before(cutoffDate)

		results = append(results, activity)
	}

	return results, nil
}

// SetInactivityThreshold updates the inactivity threshold
func (c *TenantActivityChecker) SetInactivityThreshold(days int) {
	if days > 0 {
		c.inactivityDays = days
	}
}
