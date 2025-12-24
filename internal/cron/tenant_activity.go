package cron

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// TenantActivityChecker handles checking tenant activity and flagging inactive tenants
type TenantActivityChecker struct {
	db              *sql.DB
	inactivityDays  int // Number of days without activity to be considered inactive
}

// NewTenantActivityChecker creates a new tenant activity checker
func NewTenantActivityChecker(db *sql.DB, inactivityDays int) *TenantActivityChecker {
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
	cutoffDate := time.Now().AddDate(0, 0, -c.inactivityDays)

	for rows.Next() {
		var tenantID int
		var slug, name string

		if err := rows.Scan(&tenantID, &slug, &name); err != nil {
			log.Printf("Error scanning tenant: %v", err)
			continue
		}

		// Check last booking date for this tenant
		var lastBooking *time.Time
		bookingQuery := `
			SELECT MAX(created_at)
			FROM bookings
			WHERE tenant_id = ?
		`
		c.db.QueryRow(bookingQuery, tenantID).Scan(&lastBooking)

		// Check last user activity for this tenant
		var lastActivity *time.Time
		activityQuery := `
			SELECT MAX(last_activity_at)
			FROM users
			WHERE tenant_id = ? AND is_active = 1
		`
		c.db.QueryRow(activityQuery, tenantID).Scan(&lastActivity)

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
			log.Printf("Tenant '%s' (ID: %d) flagged as inactive - last activity: %v",
				slug, tenantID, mostRecentActivity)
		}
	}

	// Check for errors during row iteration
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating tenant rows: %w", err)
	}

	log.Printf("Tenant activity check complete. Found %d inactive tenants", inactiveCount)
	return nil
}

// GetInactiveTenants returns a list of tenants with no recent activity
func (c *TenantActivityChecker) GetInactiveTenants() ([]TenantActivity, error) {
	cutoffDate := time.Now().AddDate(0, 0, -c.inactivityDays)

	query := `
		SELECT
			t.id,
			t.slug,
			t.name,
			(SELECT MAX(created_at) FROM bookings WHERE tenant_id = t.id) as last_booking,
			(SELECT MAX(last_activity_at) FROM users WHERE tenant_id = t.id AND is_active = 1) as last_login,
			(SELECT COUNT(*) FROM bookings WHERE tenant_id = t.id) as total_bookings,
			(SELECT COUNT(*) FROM users WHERE tenant_id = t.id AND is_active = 1) as active_users
		FROM tenants t
		WHERE t.status = 'active'
		ORDER BY t.name
	`

	rows, err := c.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []TenantActivity

	for rows.Next() {
		var activity TenantActivity
		var lastBooking, lastLogin *time.Time

		if err := rows.Scan(
			&activity.TenantID,
			&activity.TenantSlug,
			&activity.TenantName,
			&lastBooking,
			&lastLogin,
			&activity.TotalBookings,
			&activity.ActiveUsers,
		); err != nil {
			return nil, err
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
	query := `
		SELECT
			t.id,
			t.slug,
			t.name,
			(SELECT MAX(created_at) FROM bookings WHERE tenant_id = t.id) as last_booking,
			(SELECT MAX(last_activity_at) FROM users WHERE tenant_id = t.id AND is_active = 1) as last_login,
			(SELECT COUNT(*) FROM bookings WHERE tenant_id = t.id) as total_bookings,
			(SELECT COUNT(*) FROM users WHERE tenant_id = t.id AND is_active = 1) as active_users
		FROM tenants t
		WHERE t.status = 'active'
		ORDER BY t.name
	`

	rows, err := c.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cutoffDate := time.Now().AddDate(0, 0, -c.inactivityDays)
	var results []TenantActivity

	for rows.Next() {
		var activity TenantActivity
		var lastBooking, lastLogin *time.Time

		if err := rows.Scan(
			&activity.TenantID,
			&activity.TenantSlug,
			&activity.TenantName,
			&lastBooking,
			&lastLogin,
			&activity.TotalBookings,
			&activity.ActiveUsers,
		); err != nil {
			return nil, err
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
