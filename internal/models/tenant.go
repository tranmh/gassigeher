package models

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

// Tenant represents an organization (Tierheim) in the multi-tenant system
type Tenant struct {
	ID              int        `json:"id"`
	Slug            string     `json:"slug"`              // Subdomain: "tierheim-goeppingen"
	Name            string     `json:"name"`              // Display name: "Tierheim Göppingen"
	Status          string     `json:"status"`            // active, suspended, deleted
	ContactEmail    string     `json:"contact_email"`
	ContactPhone    *string    `json:"contact_phone,omitempty"`
	Address         *string    `json:"address,omitempty"`
	City            *string    `json:"city,omitempty"`
	PostalCode      *string    `json:"postal_code,omitempty"`
	FederalState    string     `json:"federal_state"`     // For holiday API (default: "BW")
	IsDemo          bool       `json:"is_demo"`           // Demo tenant flag
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	SuspendedAt     *time.Time `json:"suspended_at,omitempty"`
	SuspendedReason *string    `json:"suspended_reason,omitempty"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`

	// Related data (loaded separately)
	Settings *TenantSettings `json:"settings,omitempty"`
}

// TenantSettings represents the customizable settings for a tenant
type TenantSettings struct {
	ID              int       `json:"id"`
	TenantID        int       `json:"tenant_id"`
	ThemePreset     string    `json:"theme_preset"`      // classic, ocean, forest, etc.
	ColorPrimary    *string   `json:"color_primary,omitempty"`
	ColorSecondary  *string   `json:"color_secondary,omitempty"`
	ColorAccent     *string   `json:"color_accent,omitempty"`
	ColorBackground *string   `json:"color_background,omitempty"`
	ColorText       *string   `json:"color_text,omitempty"`
	LogoURL         *string   `json:"logo_url,omitempty"`
	FaviconURL      *string   `json:"favicon_url,omitempty"`
	WelcomeMessage  *string   `json:"welcome_message,omitempty"`
	FooterText      *string   `json:"footer_text,omitempty"`
	WebsiteURL      *string   `json:"website_url,omitempty"`
	DonationURL     *string   `json:"donation_url,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// TenantRegistrationRequest represents the tenant registration payload
type TenantRegistrationRequest struct {
	// Organization info
	OrganizationName string `json:"organization_name"`
	Slug             string `json:"slug"`
	ContactEmail     string `json:"contact_email"`
	ContactPhone     string `json:"contact_phone,omitempty"`
	Address          string `json:"address,omitempty"`
	City             string `json:"city"`
	PostalCode       string `json:"postal_code"`
	FederalState     string `json:"federal_state"`

	// Admin user info
	AdminFirstName string `json:"admin_first_name"`
	AdminLastName  string `json:"admin_last_name"`
	AdminEmail     string `json:"admin_email"`
	AdminPassword  string `json:"admin_password"`
}

// TenantUpdateRequest represents the tenant update payload
type TenantUpdateRequest struct {
	Name         string  `json:"name,omitempty"`
	ContactEmail string  `json:"contact_email,omitempty"`
	ContactPhone *string `json:"contact_phone,omitempty"`
	Address      *string `json:"address,omitempty"`
	City         *string `json:"city,omitempty"`
	PostalCode   *string `json:"postal_code,omitempty"`
	FederalState string  `json:"federal_state,omitempty"`
}

// TenantSettingsUpdateRequest represents the tenant settings update payload
type TenantSettingsUpdateRequest struct {
	ThemePreset     string  `json:"theme_preset,omitempty"`
	ColorPrimary    *string `json:"color_primary,omitempty"`
	ColorSecondary  *string `json:"color_secondary,omitempty"`
	ColorAccent     *string `json:"color_accent,omitempty"`
	ColorBackground *string `json:"color_background,omitempty"`
	ColorText       *string `json:"color_text,omitempty"`
	WelcomeMessage  *string `json:"welcome_message,omitempty"`
	FooterText      *string `json:"footer_text,omitempty"`
	WebsiteURL      *string `json:"website_url,omitempty"`
	DonationURL     *string `json:"donation_url,omitempty"`
}

// TenantStats represents statistics for a tenant
type TenantStats struct {
	TenantID          int `json:"tenant_id"`
	TotalUsers        int `json:"total_users"`
	ActiveUsers       int `json:"active_users"`
	TotalDogs         int `json:"total_dogs"`
	AvailableDogs     int `json:"available_dogs"`
	TotalBookings     int `json:"total_bookings"`
	BookingsThisMonth int `json:"bookings_this_month"`
}

// Theme presets with their color values
type ThemeColors struct {
	Primary    string `json:"primary"`
	Secondary  string `json:"secondary"`
	Accent     string `json:"accent"`
	Background string `json:"background"`
	Text       string `json:"text"`
}

// ThemePresets contains all available theme presets
var ThemePresets = map[string]ThemeColors{
	"classic": {
		Primary:    "#82b965",
		Secondary:  "#26272b",
		Accent:     "#4a90e2",
		Background: "#fef9f3",
		Text:       "#2c3e34",
	},
	"ocean": {
		Primary:    "#0077b6",
		Secondary:  "#023e8a",
		Accent:     "#48cae4",
		Background: "#f0f9ff",
		Text:       "#1a365d",
	},
	"forest": {
		Primary:    "#2d6a4f",
		Secondary:  "#1b4332",
		Accent:     "#52b788",
		Background: "#f0fdf4",
		Text:       "#14532d",
	},
	"sunset": {
		Primary:    "#f97316",
		Secondary:  "#7c2d12",
		Accent:     "#fb923c",
		Background: "#fff7ed",
		Text:       "#431407",
	},
	"lavender": {
		Primary:    "#7c3aed",
		Secondary:  "#4c1d95",
		Accent:     "#a78bfa",
		Background: "#faf5ff",
		Text:       "#3b0764",
	},
	"coral": {
		Primary:    "#f43f5e",
		Secondary:  "#881337",
		Accent:     "#fb7185",
		Background: "#fff1f2",
		Text:       "#4c0519",
	},
	"midnight": {
		Primary:    "#3b82f6",
		Secondary:  "#1e3a5f",
		Accent:     "#60a5fa",
		Background: "#f8fafc",
		Text:       "#0f172a",
	},
	"emerald": {
		Primary:    "#10b981",
		Secondary:  "#064e3b",
		Accent:     "#34d399",
		Background: "#ecfdf5",
		Text:       "#022c22",
	},
	"rose": {
		Primary:    "#ec4899",
		Secondary:  "#831843",
		Accent:     "#f472b6",
		Background: "#fdf2f8",
		Text:       "#500724",
	},
	"slate": {
		Primary:    "#64748b",
		Secondary:  "#334155",
		Accent:     "#94a3b8",
		Background: "#f8fafc",
		Text:       "#1e293b",
	},
}

// Tenant status constants
const (
	TenantStatusActive    = "active"
	TenantStatusSuspended = "suspended"
	TenantStatusDeleted   = "deleted"
)

// German federal states
var FederalStates = map[string]string{
	"BW": "Baden-Württemberg",
	"BY": "Bayern",
	"BE": "Berlin",
	"BB": "Brandenburg",
	"HB": "Bremen",
	"HH": "Hamburg",
	"HE": "Hessen",
	"MV": "Mecklenburg-Vorpommern",
	"NI": "Niedersachsen",
	"NW": "Nordrhein-Westfalen",
	"RP": "Rheinland-Pfalz",
	"SL": "Saarland",
	"SN": "Sachsen",
	"ST": "Sachsen-Anhalt",
	"SH": "Schleswig-Holstein",
	"TH": "Thüringen",
}

// Validate validates the tenant registration request
func (r *TenantRegistrationRequest) Validate() error {
	// Organization validation
	if strings.TrimSpace(r.OrganizationName) == "" {
		return errors.New("organization name is required")
	}
	if len(r.OrganizationName) > 255 {
		return errors.New("organization name must be less than 255 characters")
	}

	// Slug validation
	if err := ValidateSlug(r.Slug); err != nil {
		return err
	}

	// Contact email validation
	if strings.TrimSpace(r.ContactEmail) == "" {
		return errors.New("contact email is required")
	}
	if !isValidEmail(r.ContactEmail) {
		return errors.New("invalid contact email format")
	}

	// City validation
	if strings.TrimSpace(r.City) == "" {
		return errors.New("city is required")
	}

	// Postal code validation
	if strings.TrimSpace(r.PostalCode) == "" {
		return errors.New("postal code is required")
	}

	// Federal state validation
	if r.FederalState == "" {
		r.FederalState = "BW" // Default
	}
	if _, ok := FederalStates[r.FederalState]; !ok {
		return errors.New("invalid federal state code")
	}

	// Admin user validation
	if strings.TrimSpace(r.AdminFirstName) == "" {
		return errors.New("admin first name is required")
	}
	if strings.TrimSpace(r.AdminLastName) == "" {
		return errors.New("admin last name is required")
	}
	if strings.TrimSpace(r.AdminEmail) == "" {
		return errors.New("admin email is required")
	}
	if !isValidEmail(r.AdminEmail) {
		return errors.New("invalid admin email format")
	}
	if len(r.AdminPassword) < 8 {
		return errors.New("admin password must be at least 8 characters")
	}

	return nil
}

// ValidateSlug validates a tenant slug
func ValidateSlug(slug string) error {
	if strings.TrimSpace(slug) == "" {
		return errors.New("slug is required")
	}
	if len(slug) < 3 {
		return errors.New("slug must be at least 3 characters")
	}
	if len(slug) > 100 {
		return errors.New("slug must be less than 100 characters")
	}

	// Slug must be lowercase alphanumeric with hyphens
	slugRegex := regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)
	if !slugRegex.MatchString(slug) {
		return errors.New("slug must contain only lowercase letters, numbers, and hyphens")
	}

	// No consecutive hyphens
	if strings.Contains(slug, "--") {
		return errors.New("slug cannot contain consecutive hyphens")
	}

	// Reserved slugs (note: "demo" is intentionally NOT reserved - it's used for demo tenant)
	reservedSlugs := []string{
		"www", "admin", "api", "app", "mail", "email", "ftp", "ssh",
		"test", "staging", "dev", "prod", "central", "support",
		"help", "docs", "blog", "status", "cdn", "assets", "static",
	}
	for _, reserved := range reservedSlugs {
		if slug == reserved {
			return errors.New("this subdomain is reserved")
		}
	}

	return nil
}

// ValidateThemePreset validates a theme preset name
func ValidateThemePreset(preset string) bool {
	_, ok := ThemePresets[preset]
	return ok
}

// ValidateHexColor validates a hex color string
func ValidateHexColor(color string) bool {
	if color == "" {
		return true // Empty is valid (use preset)
	}
	hexRegex := regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
	return hexRegex.MatchString(color)
}

// GetThemeColors returns the theme colors for a tenant settings
func (s *TenantSettings) GetThemeColors() ThemeColors {
	// If custom colors are set, use them
	if s.ColorPrimary != nil && *s.ColorPrimary != "" {
		return ThemeColors{
			Primary:    deref(s.ColorPrimary, "#82b965"),
			Secondary:  deref(s.ColorSecondary, "#26272b"),
			Accent:     deref(s.ColorAccent, "#4a90e2"),
			Background: deref(s.ColorBackground, "#fef9f3"),
			Text:       deref(s.ColorText, "#2c3e34"),
		}
	}

	// Otherwise use preset
	if preset, ok := ThemePresets[s.ThemePreset]; ok {
		return preset
	}

	// Fallback to classic
	return ThemePresets["classic"]
}

// Helper function to dereference string pointer with default
func deref(s *string, def string) string {
	if s == nil || *s == "" {
		return def
	}
	return *s
}

// isValidEmail checks if email format is valid
func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// IsActive returns true if the tenant is active
func (t *Tenant) IsActive() bool {
	return t.Status == TenantStatusActive
}

// IsSuspended returns true if the tenant is suspended
func (t *Tenant) IsSuspended() bool {
	return t.Status == TenantStatusSuspended
}

// IsDeleted returns true if the tenant is deleted
func (t *Tenant) IsDeleted() bool {
	return t.Status == TenantStatusDeleted
}
