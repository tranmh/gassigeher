package models

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// FuzzValidateSlug tests slug validation with random inputs
// Looking for: panics, regex catastrophic backtracking, unexpected behavior
func FuzzValidateSlug(f *testing.F) {
	// Seed corpus with interesting inputs
	f.Add("valid-slug")
	f.Add("a")
	f.Add("ab")
	f.Add("abc")
	f.Add("")
	f.Add("UPPERCASE")
	f.Add("with spaces")
	f.Add("with_underscore")
	f.Add("with.dot")
	f.Add("123numeric")
	f.Add("-starts-with-dash")
	f.Add("ends-with-dash-")
	f.Add("double--dash")
	f.Add("a" + strings.Repeat("b", 1000)) // Long input
	f.Add("ü-unicode")
	f.Add("emoji-🐕")
	f.Add("\x00null")
	f.Add("\n\r\t")
	f.Add("admin")
	f.Add("api")
	f.Add("www")
	f.Add("demo")

	f.Fuzz(func(t *testing.T, slug string) {
		// Should not panic
		err := ValidateSlug(slug)

		// If valid, verify invariants
		if err == nil {
			// Valid slugs must be 3-100 chars
			if len(slug) < 3 || len(slug) > 100 {
				t.Errorf("ValidateSlug accepted slug with invalid length %d: %q", len(slug), slug)
			}
			// Valid slugs must be lowercase
			if slug != strings.ToLower(slug) {
				t.Errorf("ValidateSlug accepted non-lowercase slug: %q", slug)
			}
			// Valid slugs must not start or end with dash
			if strings.HasPrefix(slug, "-") || strings.HasSuffix(slug, "-") {
				t.Errorf("ValidateSlug accepted slug with dash at boundary: %q", slug)
			}
			// Valid slugs must start with letter
			if len(slug) > 0 && (slug[0] < 'a' || slug[0] > 'z') {
				t.Errorf("ValidateSlug accepted slug not starting with letter: %q", slug)
			}
		}
	})
}

// FuzzValidateHexColor tests hex color validation
func FuzzValidateHexColor(f *testing.F) {
	// Seed corpus
	f.Add("#ffffff")
	f.Add("#000000")
	f.Add("#FF0000")
	f.Add("#abc")
	f.Add("")
	f.Add("ffffff")
	f.Add("#fff")
	f.Add("#gggggg")
	f.Add("#1234567")
	f.Add("#12345")
	f.Add("##ffffff")
	f.Add("#ffffff ")
	f.Add(" #ffffff")
	f.Add("\x00#ffffff")

	f.Fuzz(func(t *testing.T, color string) {
		// Should not panic
		valid := ValidateHexColor(color)

		// If valid AND non-empty, verify invariants
		// Note: Empty string is valid by design (means "use preset")
		if valid && color != "" {
			// Must be exactly 7 chars (#RRGGBB)
			if len(color) != 7 {
				t.Errorf("ValidateHexColor accepted non-empty color with length %d: %q", len(color), color)
			}
			// Must start with #
			if !strings.HasPrefix(color, "#") {
				t.Errorf("ValidateHexColor accepted non-empty color without #: %q", color)
			}
		}
	})
}

// FuzzValidatePhone tests phone number validation
func FuzzValidatePhone(f *testing.F) {
	// Seed corpus
	f.Add("+49 123 456789")
	f.Add("0123456789")
	f.Add("+1-555-123-4567")
	f.Add("(555) 123-4567")
	f.Add("")
	f.Add("abc")
	f.Add("123")
	f.Add("+")
	f.Add("++49123456789")
	f.Add(strings.Repeat("9", 100))
	f.Add("☎️")
	f.Add("\x00123456789")

	f.Fuzz(func(t *testing.T, phone string) {
		// Should not panic
		err := ValidatePhone(phone)

		// If valid and not empty, verify basic invariants
		if err == nil && phone != "" {
			// Should contain at least some digits
			hasDigit := false
			for _, c := range phone {
				if c >= '0' && c <= '9' {
					hasDigit = true
					break
				}
			}
			if !hasDigit {
				t.Errorf("ValidatePhone accepted phone without digits: %q", phone)
			}
		}
	})
}

// FuzzValidateThemePreset tests theme preset validation
func FuzzValidateThemePreset(f *testing.F) {
	// Seed corpus
	f.Add("default")
	f.Add("nature")
	f.Add("ocean")
	f.Add("sunset")
	f.Add("custom")
	f.Add("")
	f.Add("NATURE")
	f.Add("nature ")
	f.Add(" nature")
	f.Add("unknown")

	f.Fuzz(func(t *testing.T, preset string) {
		// Should not panic
		_ = ValidateThemePreset(preset)
	})
}

// FuzzDateParsing tests date parsing robustness
func FuzzDateParsing(f *testing.F) {
	// Seed corpus with various date formats
	f.Add("2025-12-25")
	f.Add("2025-01-01")
	f.Add("2025-12-31")
	f.Add("2025-02-29") // Invalid leap year
	f.Add("2024-02-29") // Valid leap year
	f.Add("2025-13-01") // Invalid month
	f.Add("2025-00-01") // Invalid month
	f.Add("2025-12-32") // Invalid day
	f.Add("2025-12-00") // Invalid day
	f.Add("")
	f.Add("not-a-date")
	f.Add("25-12-2025")  // Wrong format
	f.Add("2025/12/25")  // Wrong separator
	f.Add("2025-12-25 ") // Trailing space
	f.Add(" 2025-12-25") // Leading space
	f.Add("2025-12-25T00:00:00Z")
	f.Add("9999-12-31")
	f.Add("0001-01-01")
	f.Add("-001-01-01")

	f.Fuzz(func(t *testing.T, date string) {
		// Should not panic
		_, err := time.Parse("2006-01-02", date)

		// If parsing succeeded, verify the date round-trips
		if err == nil {
			parsed, _ := time.Parse("2006-01-02", date)
			formatted := parsed.Format("2006-01-02")
			// Note: This may differ due to normalization (e.g., 2025-02-30 -> 2025-03-02)
			// We just verify no panic occurs
			_ = formatted
		}
	})
}

// FuzzTimeParsing tests time parsing robustness
func FuzzTimeParsing(f *testing.F) {
	// Seed corpus
	f.Add("09:00")
	f.Add("00:00")
	f.Add("23:59")
	f.Add("24:00") // Invalid
	f.Add("9:00")  // Missing leading zero
	f.Add("09:0")  // Missing trailing zero
	f.Add("")
	f.Add("09:00:00") // With seconds
	f.Add("09:00 AM") // 12-hour format
	f.Add("-1:00")
	f.Add("09:-1")
	f.Add("ab:cd")

	f.Fuzz(func(t *testing.T, timeStr string) {
		// Should not panic
		_, _ = time.Parse("15:04", timeStr)
	})
}

// FuzzCreateBookingRequestValidate tests booking request validation
func FuzzCreateBookingRequestValidate(f *testing.F) {
	// Seed corpus
	f.Add(1, "2025-12-25", "14:00")
	f.Add(0, "2025-12-25", "14:00")
	f.Add(-1, "2025-12-25", "14:00")
	f.Add(1, "", "14:00")
	f.Add(1, "2025-12-25", "")
	f.Add(1, "invalid", "14:00")
	f.Add(1, "2025-12-25", "invalid")
	f.Add(1, "2025-12-25", "25:00")
	f.Add(999999, "9999-12-31", "23:59")

	f.Fuzz(func(t *testing.T, dogID int, date, scheduledTime string) {
		req := CreateBookingRequest{
			DogID:         dogID,
			Date:          date,
			ScheduledTime: scheduledTime,
		}

		// Should not panic
		err := req.Validate()

		// If valid, verify invariants
		if err == nil {
			if dogID <= 0 {
				t.Errorf("Validate accepted invalid dogID: %d", dogID)
			}
			if date == "" {
				t.Errorf("Validate accepted empty date")
			}
			if scheduledTime == "" {
				t.Errorf("Validate accepted empty scheduledTime")
			}
		}
	})
}

// FuzzMoveBookingRequestValidate tests move booking request validation
func FuzzMoveBookingRequestValidate(f *testing.F) {
	// Seed corpus
	f.Add("2025-12-25", "14:00", "Room change")
	f.Add("", "14:00", "Reason")
	f.Add("2025-12-25", "", "Reason")
	f.Add("2025-12-25", "14:00", "")
	f.Add("invalid-date", "14:00", "Reason")

	f.Fuzz(func(t *testing.T, date, scheduledTime, reason string) {
		req := MoveBookingRequest{
			Date:          date,
			ScheduledTime: scheduledTime,
			Reason:        reason,
		}

		// Should not panic
		_ = req.Validate()
	})
}

// FuzzRegisterRequestValidate tests user registration validation
func FuzzRegisterRequestValidate(f *testing.F) {
	// Seed corpus
	f.Add("test@example.com", "password123", "John", "Doe", "")
	f.Add("", "password123", "John", "Doe", "")
	f.Add("test@example.com", "", "John", "Doe", "")
	f.Add("test@example.com", "password123", "", "Doe", "")
	f.Add("test@example.com", "password123", "John", "", "")
	f.Add("invalid-email", "password123", "John", "Doe", "")
	f.Add("test@example.com", "short", "John", "Doe", "")
	f.Add("test@example.com", strings.Repeat("a", 1000), "John", "Doe", "")

	f.Fuzz(func(t *testing.T, email, password, firstName, lastName, phone string) {
		req := RegisterRequest{
			Email:     email,
			Password:  password,
			FirstName: firstName,
			LastName:  lastName,
			Phone:     phone,
		}

		// Should not panic
		_ = req.Validate()
	})
}

// FuzzUnicodeHandling tests Unicode handling in various validation functions
func FuzzUnicodeHandling(f *testing.F) {
	// Seed with various Unicode categories
	f.Add("normal")
	f.Add("日本語")
	f.Add("العربية")
	f.Add("🐕🦮🐩")
	f.Add("Ñoño")
	f.Add("café")
	f.Add("\u0000") // Null
	f.Add("\uFEFF") // BOM
	f.Add("\u200B") // Zero-width space
	f.Add("\u202E") // RTL override
	f.Add("a\u0300") // Combining character

	f.Fuzz(func(t *testing.T, input string) {
		// Verify all validation functions handle Unicode without panic
		if utf8.ValidString(input) {
			_ = ValidateSlug(input)
			_ = ValidateHexColor(input)
			_ = ValidatePhone(input)
			_ = ValidateThemePreset(input)
		}
	})
}
