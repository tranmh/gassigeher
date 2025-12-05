package models

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

// User represents a user in the system
type User struct {
	ID                       int        `json:"id"`
	Name                     string     `json:"name"`
	Email                    *string    `json:"email,omitempty"`
	Phone                    *string    `json:"phone,omitempty"`
	PasswordHash             *string    `json:"-"`
	ExperienceLevel          string     `json:"experience_level"`
	// DONE: Admin flags
	IsAdmin                  bool       `json:"is_admin"`
	IsSuperAdmin             bool       `json:"is_super_admin"`
	IsVerified               bool       `json:"is_verified"`
	IsActive                 bool       `json:"is_active"`
	IsDeleted                bool       `json:"is_deleted"`
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

// RegisterRequest represents the registration payload
type RegisterRequest struct {
	Name            string `json:"name"`
	Email           string `json:"email"`
	Phone           string `json:"phone"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
	AcceptTerms     bool   `json:"accept_terms"`
}

// LoginRequest represents the login payload
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Token   string `json:"token"`
	User    *User  `json:"user"`
	IsAdmin bool   `json:"is_admin"`
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
type UpdateProfileRequest struct {
	Name  *string `json:"name,omitempty"`
	Email *string `json:"email,omitempty"`
	Phone *string `json:"phone,omitempty"`
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

// Validate validates the RegisterRequest
func (r *RegisterRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("Name ist erforderlich")
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
	return nil
}

// Validate validates the UpdateProfileRequest
func (u *UpdateProfileRequest) Validate() error {
	if u.Name != nil && strings.TrimSpace(*u.Name) == "" {
		return errors.New("Name darf nicht leer sein")
	}
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
