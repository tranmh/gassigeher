package models

import "time"

// DemoTenantState represents the state of a demo tenant
type DemoTenantState struct {
	ID            int        `json:"id"`
	TenantID      int        `json:"tenant_id"`
	AdminPassword string     `json:"admin_password"` // Plain text - intentionally for demo display
	LastResetAt   *time.Time `json:"last_reset_at,omitempty"`
	NextResetAt   *time.Time `json:"next_reset_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// DemoCredentials represents the credentials displayed on the demo landing page
type DemoCredentials struct {
	AdminEmail    string `json:"admin_email"`
	AdminPassword string `json:"admin_password"`
	NextResetAt   string `json:"next_reset_at"`  // Formatted for display (e.g., "01.01.2025 00:00")
	LastResetAt   string `json:"last_reset_at"`  // Formatted for display
}

// DemoStatus represents the demo status returned by the API
type DemoStatus struct {
	IsDemo      bool   `json:"is_demo"`
	NextResetAt string `json:"next_reset_at,omitempty"` // Formatted for display
}

// DemoUser represents a demo user for display on the landing page
type DemoUser struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Level    string `json:"level"`     // green, orange, blue
	LevelDE  string `json:"level_de"`  // German: Anfänger, Fortgeschritten, Experte
}

// DemoDog represents a demo dog for display on the landing page
type DemoDog struct {
	Name     string `json:"name"`
	Breed    string `json:"breed"`
	Category string `json:"category"` // green, orange, blue
}
