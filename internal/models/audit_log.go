package models

import "time"

// AuditLog represents a business audit log entry
type AuditLog struct {
	ID         int       `json:"id"`
	TenantID   int       `json:"tenant_id"`
	UserID     *int      `json:"user_id,omitempty"`
	Action     string    `json:"action"`      // e.g., "booking.created", "user.promoted"
	EntityType string    `json:"entity_type"` // e.g., "booking", "user", "dog"
	EntityID   *int      `json:"entity_id,omitempty"`
	OldValue   *string   `json:"old_value,omitempty"`  // JSON representation
	NewValue   *string   `json:"new_value,omitempty"`  // JSON representation
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	CreatedAt  time.Time `json:"created_at"`
}

// AuditAction constants for standardized action names
const (
	// Booking actions
	AuditActionBookingCreated   = "booking.created"
	AuditActionBookingCancelled = "booking.cancelled"
	AuditActionBookingApproved  = "booking.approved"
	AuditActionBookingRejected  = "booking.rejected"
	AuditActionBookingMoved     = "booking.moved"
	AuditActionBookingCompleted = "booking.completed"

	// User actions
	AuditActionUserCreated     = "user.created"
	AuditActionUserUpdated     = "user.updated"
	AuditActionUserDeleted     = "user.deleted"
	AuditActionUserPromoted    = "user.promoted"
	AuditActionUserDemoted     = "user.demoted"
	AuditActionUserActivated   = "user.activated"
	AuditActionUserDeactivated = "user.deactivated"
	AuditActionUserLogin       = "user.login"
	AuditActionUserLogout      = "user.logout"
	AuditActionUserImpersonated = "user.impersonated"

	// Dog actions
	AuditActionDogCreated = "dog.created"
	AuditActionDogUpdated = "dog.updated"
	AuditActionDogDeleted = "dog.deleted"

	// Settings actions
	AuditActionSettingsChanged = "settings.changed"
	AuditActionThemeChanged    = "theme.changed"

	// Data actions
	AuditActionDataExported = "data.exported"
	AuditActionDataImported = "data.imported"

	// Color request actions
	AuditActionColorRequested = "color.requested"
	AuditActionColorApproved  = "color.approved"
	AuditActionColorDenied    = "color.denied"

	// Tenant actions
	AuditActionTenantCreated    = "tenant.created"
	AuditActionTenantUpdated    = "tenant.updated"
	AuditActionTenantActivated  = "tenant.activated"
	AuditActionTenantDeactivated = "tenant.deactivated"
)

// EntityType constants
const (
	EntityTypeBooking           = "booking"
	EntityTypeUser              = "user"
	EntityTypeDog               = "dog"
	EntityTypeSettings          = "settings"
	EntityTypeTenant            = "tenant"
	EntityTypeColorRequest = "color_request"
	EntityTypeTheme             = "theme"
)

// AuditLogFilter for querying audit logs
type AuditLogFilter struct {
	TenantID   *int
	UserID     *int
	Action     *string
	EntityType *string
	EntityID   *int
	StartDate  *time.Time
	EndDate    *time.Time
	Limit      int
	Offset     int
}
