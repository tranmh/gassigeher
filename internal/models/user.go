package models

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

// User represents a user in the system
type User struct {
	ID                       int              `json:"id"`
	FirstName                string           `json:"first_name"`
	LastName                 string           `json:"last_name"`
	Email                    *string          `json:"email,omitempty"`
	Phone                    *string          `json:"phone,omitempty"`
	PasswordHash             *string          `json:"-"`
	ExperienceLevel          string           `json:"experience_level"`
	Colors                   []ColorCategory  `json:"colors,omitempty"`
	// DONE: Admin flags
	IsAdmin                  bool             `json:"is_admin"`
	IsSuperAdmin             bool             `json:"is_super_admin"`
	IsVerified               bool       `json:"is_verified"`
	IsActive                 bool       `json:"is_active"`
	IsDeleted                bool       `json:"is_deleted"`
	MustChangePassword       bool       `json:"must_change_password"`
	VerificationToken        *string    `json:"-"`
	VerificationTokenExpires *time.Time `json:"-"`
	PasswordResetToken       *string    `json:"-"`
	PasswordResetExpires     *time.Time `json:"-"`
	ProfilePhoto             *string    `json:"profile_photo,omitempty"`
	AnonymousID              *string    `json:"anonymous_id,omitempty"`
	TermsAcceptedAt          time.Time  `json:"terms_accepted_at"`
	LastActivityAt           time.Time  `json:"last_activity_at"`
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
	MustChangePassword bool   `json:"must_change_password"`
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

// AdminUpdateUserRequest represents admin profile update payload (can edit names and experience level)
type AdminUpdateUserRequest struct {
	FirstName       *string `json:"first_name,omitempty"`
	LastName        *string `json:"last_name,omitempty"`
	Email           *string `json:"email,omitempty"`
	Phone           *string `json:"phone,omitempty"`
	ExperienceLevel *string `json:"experience_level,omitempty"`
}

// Phone regex: allows digits, country code, separators, and balanced parentheses
// Supports formats like: 0123456789, +49 123456789, (0123) 456789, 0123-456789
var phoneRegex = regexp.MustCompile(`^\+?[\s\-\.]?(?:\()?[0-9]{1,4}(?:\))?[\s\-\.]?[0-9]{1,4}[\s\-\.]?[0-9]{3,}[\s\-\.]?[0-9]{0,4}$`)

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
	if strings.TrimSpace(r.Email) == "" {
		return errors.New("E-Mail ist erforderlich")
	}
	if err := ValidatePhone(r.Phone); err != nil {
		return err
	}
	if r.Password == "" {
		return errors.New("Passwort ist erforderlich")
	}
	if len(r.Password) < 8 {
		return errors.New("Passwort muss mindestens 8 Zeichen lang sein")
	}
	if r.Password != r.ConfirmPassword {
		return errors.New("Passwörter stimmen nicht überein")
	}
	if !r.AcceptTerms {
		return errors.New("Sie müssen die AGB akzeptieren")
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
	if u.Email != nil && strings.TrimSpace(*u.Email) == "" {
		return errors.New("E-Mail darf nicht leer sein")
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
	if a.Email != nil && strings.TrimSpace(*a.Email) == "" {
		return errors.New("E-Mail darf nicht leer sein")
	}
	if a.Phone != nil {
		if err := ValidatePhone(*a.Phone); err != nil {
			return err
		}
	}
	if a.ExperienceLevel != nil {
		validLevels := map[string]bool{"green": true, "orange": true, "blue": true}
		if !validLevels[*a.ExperienceLevel] {
			return errors.New("Ungültiges Erfahrungslevel (green, orange, blue)")
		}
	}
	return nil
}

// AdminCreateUserRequest represents admin user creation payload
type AdminCreateUserRequest struct {
	FirstName       string  `json:"first_name"`
	LastName        string  `json:"last_name"`
	Email           string  `json:"email"`
	Phone           *string `json:"phone,omitempty"`
	ExperienceLevel string  `json:"experience_level"`
	IsAdmin         bool    `json:"is_admin"`
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
	if r.Email == "" {
		return errors.New("E-Mail ist erforderlich")
	}
	// Validate experience level
	validLevels := map[string]bool{"green": true, "orange": true, "blue": true}
	if !validLevels[r.ExperienceLevel] {
		return errors.New("Ungültiges Erfahrungslevel (green, orange, blue)")
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
