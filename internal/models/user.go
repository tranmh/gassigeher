package models

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// User represents a user in the system
type User struct {
	ID                       int              `json:"id"`
	TenantID                 int              `json:"tenant_id,omitempty"` // SaaS: Tenant this user belongs to
	FirstName                string           `json:"first_name"`
	LastName                 string           `json:"last_name"`
	Email                    *string          `json:"email,omitempty"`
	Phone                    *string          `json:"phone,omitempty"`
	PasswordHash             *string          `json:"-"`
	Colors                   []ColorCategory  `json:"colors,omitempty"`
	// DONE: Admin flags
	IsAdmin                  bool             `json:"is_admin"`
	IsSuperAdmin             bool             `json:"is_super_admin"`
	IsCentralAdmin           bool             `json:"is_central_admin"` // SaaS: Platform-wide admin (not tied to tenant)
	IsVerified               bool       `json:"is_verified"`
	IsActive                 bool       `json:"is_active"`
	IsDeleted                bool       `json:"is_deleted"`
	MustChangePassword       bool       `json:"must_change_password"`
	VerificationToken        *string    `json:"-"`
	VerificationTokenExpires *time.Time `json:"-"`
	PasswordResetToken       *string    `json:"-"`
	PasswordResetExpires     *time.Time `json:"-"`
	CalendarToken            *string    `json:"-"` // Token for iCal feed authentication
	ProfilePhoto             *string    `json:"profile_photo,omitempty"`
	AnonymousID              *string    `json:"anonymous_id,omitempty"`
	TermsAcceptedAt          time.Time  `json:"terms_accepted_at"`
	LastActivityAt           time.Time  `json:"last_activity_at"`
	// Brute force protection fields
	FailedLoginAttempts      int        `json:"failed_login_attempts,omitempty"`
	LockedUntil              *time.Time `json:"locked_until,omitempty"`
	LastFailedLogin          *time.Time `json:"last_failed_login,omitempty"`
	DeactivatedAt            *time.Time `json:"deactivated_at,omitempty"`
	DeactivationReason       *string    `json:"deactivation_reason,omitempty"`
	ReactivatedAt            *time.Time `json:"reactivated_at,omitempty"`
	DeletedAt                *time.Time `json:"deleted_at,omitempty"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

// FullName returns the user's full name (FirstName LastName)
func (u *User) FullName() string {
	if u.LastName == "" {
		return u.FirstName
	}
	return u.FirstName + " " + u.LastName
}

// RegisterRequest represents the registration payload
type RegisterRequest struct {
	FirstName            string `json:"first_name"`
	LastName             string `json:"last_name"`
	Email                string `json:"email"`
	Phone                string `json:"phone"`
	Password             string `json:"password"`
	ConfirmPassword      string `json:"confirm_password"`
	AcceptTerms          bool   `json:"accept_terms"`
	AcceptPrivacy        bool   `json:"accept_privacy"`
	RegistrationPassword string `json:"registration_password"`
}

// LoginRequest represents the login payload
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Token              string `json:"token"`
	User               *User  `json:"user"`
	IsAdmin            bool   `json:"is_admin"`
	IsCentralAdmin     bool   `json:"is_central_admin"`     // SaaS: true if user is platform-wide admin
	MustChangePassword bool   `json:"must_change_password"`
	RedirectTo         string `json:"redirect_to"`          // SaaS: where to redirect after login
}

// VerifyEmailRequest represents email verification payload
type VerifyEmailRequest struct {
	Token string `json:"token"`
}

// ForgotPasswordRequest represents forgot password payload
type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

// ResetPasswordRequest represents password reset payload
type ResetPasswordRequest struct {
	Token           string `json:"token"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

// ChangePasswordRequest represents change password payload
type ChangePasswordRequest struct {
	OldPassword     string `json:"old_password"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

// UpdateProfileRequest represents profile update payload
// Note: FirstName and LastName can only be edited by admins
type UpdateProfileRequest struct {
	Email *string `json:"email,omitempty"`
	Phone *string `json:"phone,omitempty"`
}

// AdminUpdateUserRequest represents admin profile update payload (can edit names)
type AdminUpdateUserRequest struct {
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Email     *string `json:"email,omitempty"`
	Phone     *string `json:"phone,omitempty"`
}

// phoneRegex: Simple, ReDoS-safe regex for phone validation
// SECURITY FIX: Previous regex was vulnerable to catastrophic backtracking (ReDoS)
// This simple regex allows: optional +, then digits with optional separators (space, dash, dot, parentheses)
// Actual validation logic is done in ValidatePhone function
var phoneRegex = regexp.MustCompile(`^[+]?[0-9()\s.\-]{7,30}$`)

// emailRegex validates basic email format
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// Password complexity constants
const (
	PasswordMinLength = 8
	PasswordMaxLength = 128
)

// hasUppercase checks if string contains at least one uppercase letter
func hasUppercase(s string) bool {
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			return true
		}
	}
	return false
}

// hasLowercase checks if string contains at least one lowercase letter
func hasLowercase(s string) bool {
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			return true
		}
	}
	return false
}

// hasDigit checks if string contains at least one digit
func hasDigit(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

// hasSpecialChar checks if string contains at least one special character
func hasSpecialChar(s string) bool {
	specialChars := "!@#$%^&*()_+-=[]{}|;':\",./<>?"
	for _, r := range s {
		if strings.ContainsRune(specialChars, r) {
			return true
		}
	}
	return false
}

// ValidatePasswordComplexity validates that a password meets complexity requirements
// SECURITY FIX: Enforce strong password policy to prevent brute force attacks
// Requirements: 8+ chars, at least one uppercase, one lowercase, one digit
func ValidatePasswordComplexity(password string) error {
	if len(password) < PasswordMinLength {
		return errors.New("Passwort muss mindestens 8 Zeichen lang sein")
	}
	if len(password) > PasswordMaxLength {
		return errors.New("Passwort darf maximal 128 Zeichen lang sein")
	}
	if !hasUppercase(password) {
		return errors.New("Passwort muss mindestens einen Großbuchstaben enthalten")
	}
	if !hasLowercase(password) {
		return errors.New("Passwort muss mindestens einen Kleinbuchstaben enthalten")
	}
	if !hasDigit(password) {
		return errors.New("Passwort muss mindestens eine Ziffer enthalten")
	}
	return nil
}

// ValidateURL validates a URL for security and format
// SECURITY FIX: Prevents XSS via malicious URLs (javascript:, data:, etc.)
// Only allows http and https schemes
func ValidateURL(urlStr string) error {
	urlStr = strings.TrimSpace(urlStr)
	if urlStr == "" {
		return nil // Empty URLs are allowed (optional field)
	}

	// Parse the URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return errors.New("Ungültiges URL-Format")
	}

	// SECURITY: Only allow http and https schemes
	// Blocks: javascript:, data:, file:, vbscript:, etc.
	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme != "http" && scheme != "https" {
		return errors.New("URL muss mit http:// oder https:// beginnen")
	}

	// Verify the URL has a host
	if parsedURL.Host == "" {
		return errors.New("URL muss einen gültigen Host enthalten")
	}

	// Check for reasonable URL length (prevent DoS via extremely long URLs)
	if len(urlStr) > 2048 {
		return errors.New("URL ist zu lang (maximal 2048 Zeichen)")
	}

	return nil
}

// ValidateEmail validates an email address for security and format
// SECURITY: Prevents CRLF/header injection attacks and validates basic format
func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return errors.New("E-Mail ist erforderlich")
	}

	// SECURITY: Check for control characters that could enable header injection
	// This prevents CRLF injection (\r\n), null bytes (\x00), and other control chars
	for _, r := range email {
		if r < 32 || r == 127 { // ASCII control characters (0-31) and DEL (127)
			return errors.New("E-Mail enthält ungültige Steuerzeichen")
		}
	}

	// Validate basic email format
	if !emailRegex.MatchString(email) {
		return errors.New("Ungültiges E-Mail-Format")
	}

	return nil
}

// ValidatePhone validates a phone number format
func ValidatePhone(phone string) error {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return errors.New("Telefonnummer ist erforderlich")
	}

	// Remove all spaces, hyphens, dots for length check
	digitsOnly := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phone)

	// Minimum 7 digits required
	if len(digitsOnly) < 7 {
		return errors.New("Telefonnummer muss mindestens 7 Ziffern enthalten")
	}

	// Check for balanced parentheses
	openParen := strings.Count(phone, "(")
	closeParen := strings.Count(phone, ")")
	if openParen != closeParen {
		return errors.New("Ungültige Telefonnummer. Bitte verwenden Sie ein gültiges Format (z.B. 0123 456789 oder +49 123 456789)")
	}

	// Check that phone doesn't end with separator
	if len(phone) > 0 && (phone[len(phone)-1] == '-' || phone[len(phone)-1] == '.' || phone[len(phone)-1] == ' ') {
		return errors.New("Ungültige Telefonnummer. Bitte verwenden Sie ein gültiges Format (z.B. 0123 456789 oder +49 123 456789)")
	}

	if !phoneRegex.MatchString(phone) {
		return errors.New("Ungültige Telefonnummer. Bitte verwenden Sie ein gültiges Format (z.B. 0123 456789 oder +49 123 456789)")
	}
	return nil
}

// registrationPasswordRegex validates 8 alphanumeric characters
var registrationPasswordRegex = regexp.MustCompile(`^[a-zA-Z0-9]{8}$`)

// Validate validates the RegisterRequest
func (r *RegisterRequest) Validate() error {
	if strings.TrimSpace(r.FirstName) == "" {
		return errors.New("Vorname ist erforderlich")
	}
	if strings.TrimSpace(r.LastName) == "" {
		return errors.New("Nachname ist erforderlich")
	}
	if err := ValidateEmail(r.Email); err != nil {
		return err
	}
	if err := ValidatePhone(r.Phone); err != nil {
		return err
	}
	if r.Password == "" {
		return errors.New("Passwort ist erforderlich")
	}
	// SECURITY FIX: Use strong password complexity validation
	if err := ValidatePasswordComplexity(r.Password); err != nil {
		return err
	}
	if r.Password != r.ConfirmPassword {
		return errors.New("Passwörter stimmen nicht überein")
	}
	if !r.AcceptTerms {
		return errors.New("Sie müssen die AGB akzeptieren")
	}
	if !r.AcceptPrivacy {
		return errors.New("Sie müssen die Datenschutzerklärung akzeptieren")
	}
	// Validate registration password format
	if strings.TrimSpace(r.RegistrationPassword) == "" {
		return errors.New("Registrierungspasswort ist erforderlich")
	}
	if !registrationPasswordRegex.MatchString(r.RegistrationPassword) {
		return errors.New("Registrierungspasswort muss genau 8 alphanumerische Zeichen enthalten")
	}
	return nil
}

// Validate validates the UpdateProfileRequest
func (u *UpdateProfileRequest) Validate() error {
	if u.Email != nil {
		if err := ValidateEmail(*u.Email); err != nil {
			return err
		}
	}
	if u.Phone != nil {
		if err := ValidatePhone(*u.Phone); err != nil {
			return err
		}
	}
	return nil
}

// Validate validates the AdminUpdateUserRequest
func (a *AdminUpdateUserRequest) Validate() error {
	if a.FirstName != nil && strings.TrimSpace(*a.FirstName) == "" {
		return errors.New("Vorname darf nicht leer sein")
	}
	if a.LastName != nil && strings.TrimSpace(*a.LastName) == "" {
		return errors.New("Nachname darf nicht leer sein")
	}
	if a.Email != nil {
		if err := ValidateEmail(*a.Email); err != nil {
			return err
		}
	}
	if a.Phone != nil {
		if err := ValidatePhone(*a.Phone); err != nil {
			return err
		}
	}
	return nil
}

// AdminCreateUserRequest represents admin user creation payload
type AdminCreateUserRequest struct {
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Email     string  `json:"email"`
	Phone     *string `json:"phone,omitempty"`
	IsAdmin   bool    `json:"is_admin"`
	ColorIDs  []int   `json:"color_ids,omitempty"` // Color categories to assign to user
}

// Validate validates the AdminCreateUserRequest and trims whitespace from fields
func (r *AdminCreateUserRequest) Validate() error {
	// Trim whitespace from all string fields
	r.FirstName = strings.TrimSpace(r.FirstName)
	r.LastName = strings.TrimSpace(r.LastName)
	r.Email = strings.TrimSpace(r.Email)

	if r.FirstName == "" {
		return errors.New("Vorname ist erforderlich")
	}
	if r.LastName == "" {
		return errors.New("Nachname ist erforderlich")
	}
	if err := ValidateEmail(r.Email); err != nil {
		return err
	}
	if r.Phone != nil && *r.Phone != "" {
		trimmedPhone := strings.TrimSpace(*r.Phone)
		r.Phone = &trimmedPhone
		if err := ValidatePhone(*r.Phone); err != nil {
			return err
		}
	}
	return nil
}
