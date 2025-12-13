package models

import (
	"testing"
)

// TestValidatePhone tests phone number validation
func TestValidatePhone(t *testing.T) {
	tests := []struct {
		name    string
		phone   string
		wantErr bool
		reason  string
	}{
		// Valid phone numbers
		{
			name:    "Valid German number with spaces",
			phone:   "0123 456789",
			wantErr: false,
			reason:  "Standard German format should be valid",
		},
		{
			name:    "Valid international number with +49",
			phone:   "+49 123 456789",
			wantErr: false,
			reason:  "International format with country code should be valid",
		},
		{
			name:    "Valid number with hyphens",
			phone:   "0123-456789",
			wantErr: false,
			reason:  "Format with hyphens should be valid",
		},
		{
			name:    "Valid number with dots",
			phone:   "0123.456789",
			wantErr: false,
			reason:  "Format with dots should be valid",
		},
		{
			name:    "Valid number with parentheses",
			phone:   "(0123) 456789",
			wantErr: false,
			reason:  "Format with parentheses should be valid",
		},
		{
			name:    "Valid 10-digit number",
			phone:   "0123456789",
			wantErr: false,
			reason:  "10-digit format should be valid",
		},
		{
			name:    "Valid 7-digit minimum",
			phone:   "0123456",
			wantErr: false,
			reason:  "Minimum 7 digits should be valid",
		},
		{
			name:    "Valid international 7-digit",
			phone:   "+49 123456",
			wantErr: false,
			reason:  "International format with 7 digits should be valid",
		},

		// CURRENT BUGS - These should fail but currently pass
		{
			name:    "CURRENT BUG: Single digit",
			phone:   "1",
			wantErr: true,
			reason:  "Single digit should be invalid (too short)",
		},
		{
			name:    "CURRENT BUG: Two digits",
			phone:   "12",
			wantErr: true,
			reason:  "Two digits should be invalid (too short)",
		},
		{
			name:    "CURRENT BUG: Multiple plus signs",
			phone:   "++123456789",
			wantErr: true,
			reason:  "Multiple plus signs should be invalid",
		},
		{
			name:    "CURRENT BUG: Ends with hyphen",
			phone:   "0123456789-",
			wantErr: true,
			reason:  "Should not end with separator",
		},
		{
			name:    "CURRENT BUG: Unmatched opening parenthesis",
			phone:   "(0123 456789",
			wantErr: true,
			reason:  "Unmatched parenthesis should be invalid",
		},
		{
			name:    "CURRENT BUG: Unmatched closing parenthesis",
			phone:   "0123) 456789",
			wantErr: true,
			reason:  "Unmatched parenthesis should be invalid",
		},
		{
			name:    "CURRENT BUG: Only 5 digits",
			phone:   "01234",
			wantErr: true,
			reason:  "5 digits is too short (minimum 7)",
		},
		{
			name:    "CURRENT BUG: Only 6 digits",
			phone:   "012345",
			wantErr: true,
			reason:  "6 digits is too short (minimum 7)",
		},

		// Invalid cases
		{
			name:    "Empty phone",
			phone:   "",
			wantErr: true,
			reason:  "Empty phone should be invalid",
		},
		{
			name:    "Only spaces",
			phone:   "   ",
			wantErr: true,
			reason:  "Only spaces should be invalid",
		},
		{
			name:    "Contains letters",
			phone:   "0123 ABCD 5678",
			wantErr: true,
			reason:  "Phone with letters should be invalid",
		},
		{
			name:    "Contains special characters",
			phone:   "0123#456789",
			wantErr: true,
			reason:  "Phone with invalid special characters should be invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePhone(tt.phone)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePhone(%q) error = %v, wantErr %v (reason: %s)", tt.phone, err, tt.wantErr, tt.reason)
			}
		})
	}
}

// TestRegisterRequest_Validate tests RegisterRequest validation
func TestRegisterRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     RegisterRequest
		wantErr bool
	}{
		{
			name: "Valid registration request",
			req: RegisterRequest{
				Name:                 "John Doe",
				Email:                "john@example.com",
				Phone:                "0123 456789",
				Password:             "securePass123",
				ConfirmPassword:      "securePass123",
				AcceptTerms:          true,
				RegistrationPassword: "ABC12345",
			},
			wantErr: false,
		},
		{
			name: "Invalid phone format",
			req: RegisterRequest{
				Name:                 "John Doe",
				Email:                "john@example.com",
				Phone:                "123",
				Password:             "securePass123",
				ConfirmPassword:      "securePass123",
				AcceptTerms:          true,
				RegistrationPassword: "ABC12345",
			},
			wantErr: true,
		},
		{
			name: "Empty name",
			req: RegisterRequest{
				Name:                 "",
				Email:                "john@example.com",
				Phone:                "0123 456789",
				Password:             "securePass123",
				ConfirmPassword:      "securePass123",
				AcceptTerms:          true,
				RegistrationPassword: "ABC12345",
			},
			wantErr: true,
		},
		{
			name: "Empty email",
			req: RegisterRequest{
				Name:                 "John Doe",
				Email:                "",
				Phone:                "0123 456789",
				Password:             "securePass123",
				ConfirmPassword:      "securePass123",
				AcceptTerms:          true,
				RegistrationPassword: "ABC12345",
			},
			wantErr: true,
		},
		{
			name: "Password too short",
			req: RegisterRequest{
				Name:                 "John Doe",
				Email:                "john@example.com",
				Phone:                "0123 456789",
				Password:             "short",
				ConfirmPassword:      "short",
				AcceptTerms:          true,
				RegistrationPassword: "ABC12345",
			},
			wantErr: true,
		},
		{
			name: "Passwords don't match",
			req: RegisterRequest{
				Name:                 "John Doe",
				Email:                "john@example.com",
				Phone:                "0123 456789",
				Password:             "securePass123",
				ConfirmPassword:      "securePass456",
				AcceptTerms:          true,
				RegistrationPassword: "ABC12345",
			},
			wantErr: true,
		},
		{
			name: "Terms not accepted",
			req: RegisterRequest{
				Name:                 "John Doe",
				Email:                "john@example.com",
				Phone:                "0123 456789",
				Password:             "securePass123",
				ConfirmPassword:      "securePass123",
				AcceptTerms:          false,
				RegistrationPassword: "ABC12345",
			},
			wantErr: true,
		},
		{
			name: "Empty registration password",
			req: RegisterRequest{
				Name:                 "John Doe",
				Email:                "john@example.com",
				Phone:                "0123 456789",
				Password:             "securePass123",
				ConfirmPassword:      "securePass123",
				AcceptTerms:          true,
				RegistrationPassword: "",
			},
			wantErr: true,
		},
		{
			name: "Invalid registration password format",
			req: RegisterRequest{
				Name:                 "John Doe",
				Email:                "john@example.com",
				Phone:                "0123 456789",
				Password:             "securePass123",
				ConfirmPassword:      "securePass123",
				AcceptTerms:          true,
				RegistrationPassword: "ABC123", // Too short
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestUpdateProfileRequest_Validate tests UpdateProfileRequest validation
func TestUpdateProfileRequest_Validate(t *testing.T) {
	phone1 := "0123 456789"
	phone2 := "123"  // Invalid
	email1 := "new@example.com"
	name1 := "New Name"
	emptyString := ""

	tests := []struct {
		name    string
		req     UpdateProfileRequest
		wantErr bool
	}{
		{
			name: "Valid update with phone",
			req: UpdateProfileRequest{
				Phone: &phone1,
			},
			wantErr: false,
		},
		{
			name: "Invalid phone update",
			req: UpdateProfileRequest{
				Phone: &phone2,
			},
			wantErr: true,
		},
		{
			name: "Valid email update",
			req: UpdateProfileRequest{
				Email: &email1,
			},
			wantErr: false,
		},
		{
			name: "Empty email update",
			req: UpdateProfileRequest{
				Email: &emptyString,
			},
			wantErr: true,
		},
		{
			name: "Valid name update",
			req: UpdateProfileRequest{
				Name: &name1,
			},
			wantErr: false,
		},
		{
			name: "Empty name update",
			req: UpdateProfileRequest{
				Name: &emptyString,
			},
			wantErr: true,
		},
		{
			name: "Multiple valid updates",
			req: UpdateProfileRequest{
				Name:  &name1,
				Email: &email1,
				Phone: &phone1,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
