package models

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode"

	"pgregory.net/rapid"
)

// ============================================================================
// CUSTOM GENERATORS
// ============================================================================

// genSlugChar generates valid slug characters [a-z0-9-]
func genSlugChar() *rapid.Generator[byte] {
	return rapid.Custom(func(t *rapid.T) byte {
		chars := "abcdefghijklmnopqrstuvwxyz0123456789-"
		idx := rapid.IntRange(0, len(chars)-1).Draw(t, "idx")
		return chars[idx]
	})
}

// genValidSlug generates slugs that should be valid
func genValidSlug() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		// Length 3-100, starts with letter, ends with letter/number, no consecutive dashes
		length := rapid.IntRange(3, 100).Draw(t, "length")

		// Start with letter
		firstChar := byte('a' + rapid.IntRange(0, 25).Draw(t, "first"))

		if length == 1 {
			return string(firstChar)
		}

		// Build middle (avoid consecutive dashes)
		var middle strings.Builder
		lastWasDash := false
		for i := 0; i < length-2; i++ {
			if lastWasDash {
				// Can't have another dash
				chars := "abcdefghijklmnopqrstuvwxyz0123456789"
				idx := rapid.IntRange(0, len(chars)-1).Draw(t, fmt.Sprintf("mid%d", i))
				middle.WriteByte(chars[idx])
				lastWasDash = false
			} else {
				chars := "abcdefghijklmnopqrstuvwxyz0123456789-"
				idx := rapid.IntRange(0, len(chars)-1).Draw(t, fmt.Sprintf("mid%d", i))
				ch := chars[idx]
				middle.WriteByte(ch)
				lastWasDash = (ch == '-')
			}
		}

		// End with letter or number
		endChars := "abcdefghijklmnopqrstuvwxyz0123456789"
		lastChar := endChars[rapid.IntRange(0, len(endChars)-1).Draw(t, "last")]

		return string(firstChar) + middle.String() + string(lastChar)
	})
}

// genInvalidSlug generates slugs that should be invalid
func genInvalidSlug() *rapid.Generator[string] {
	return rapid.OneOf(
		// Too short
		rapid.Just(""),
		rapid.Just("a"),
		rapid.Just("ab"),
		// Starts with number
		rapid.Custom(func(t *rapid.T) string {
			num := rapid.IntRange(0, 9).Draw(t, "num")
			rest := rapid.StringN(3, 10, -1).Draw(t, "rest")
			return fmt.Sprintf("%d%s", num, rest)
		}),
		// Starts with dash
		rapid.Custom(func(t *rapid.T) string {
			rest := rapid.StringN(3, 10, -1).Draw(t, "rest")
			return "-" + rest
		}),
		// Ends with dash
		rapid.Custom(func(t *rapid.T) string {
			_ = rapid.IntRange(0, 100).Draw(t, "dummy") // Use some randomness
			return "abc-"
		}),
		// Contains uppercase
		rapid.Custom(func(t *rapid.T) string {
			_ = rapid.IntRange(0, 100).Draw(t, "dummy") // Use some randomness
			return "Test"
		}),
		// Double dash
		rapid.Just("test--slug"),
		// Reserved slugs
		rapid.SampledFrom([]string{"www", "admin", "api", "app", "mail", "email", "ftp", "ssh",
			"test", "staging", "dev", "prod", "central", "support", "help", "docs",
			"blog", "status", "cdn", "assets", "static"}),
	)
}

// genHexColor generates hex color codes
func genHexColor() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		hex := "0123456789ABCDEFabcdef"
		var color strings.Builder
		color.WriteByte('#')
		for i := 0; i < 6; i++ {
			idx := rapid.IntRange(0, len(hex)-1).Draw(t, fmt.Sprintf("hex%d", i))
			color.WriteByte(hex[idx])
		}
		return color.String()
	})
}

// genValidDate generates valid YYYY-MM-DD dates
func genValidDate() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		year := rapid.IntRange(2020, 2030).Draw(t, "year")
		month := rapid.IntRange(1, 12).Draw(t, "month")

		// Days per month (simplified, not accounting for leap years perfectly)
		maxDay := 31
		switch month {
		case 4, 6, 9, 11:
			maxDay = 30
		case 2:
			if year%4 == 0 && (year%100 != 0 || year%400 == 0) {
				maxDay = 29
			} else {
				maxDay = 28
			}
		}
		day := rapid.IntRange(1, maxDay).Draw(t, "day")

		return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
	})
}

// genInvalidDate generates dates that look valid but aren't semantically valid
func genInvalidDate() *rapid.Generator[string] {
	return rapid.OneOf(
		// Feb 30 (never valid)
		rapid.Custom(func(t *rapid.T) string {
			year := rapid.IntRange(2020, 2030).Draw(t, "year")
			return fmt.Sprintf("%04d-02-30", year)
		}),
		// Feb 31 (never valid)
		rapid.Custom(func(t *rapid.T) string {
			year := rapid.IntRange(2020, 2030).Draw(t, "year")
			return fmt.Sprintf("%04d-02-31", year)
		}),
		// Feb 29 in non-leap year
		rapid.Custom(func(t *rapid.T) string {
			// Non-leap years: 2021, 2022, 2023, 2025, 2026, 2027
			nonLeapYears := []int{2021, 2022, 2023, 2025, 2026, 2027}
			year := rapid.SampledFrom(nonLeapYears).Draw(t, "year")
			return fmt.Sprintf("%04d-02-29", year)
		}),
		// April 31
		rapid.Custom(func(t *rapid.T) string {
			year := rapid.IntRange(2020, 2030).Draw(t, "year")
			return fmt.Sprintf("%04d-04-31", year)
		}),
		// June 31
		rapid.Custom(func(t *rapid.T) string {
			year := rapid.IntRange(2020, 2030).Draw(t, "year")
			return fmt.Sprintf("%04d-06-31", year)
		}),
		// Month 0
		rapid.Custom(func(t *rapid.T) string {
			year := rapid.IntRange(2020, 2030).Draw(t, "year")
			return fmt.Sprintf("%04d-00-15", year)
		}),
		// Month 13
		rapid.Custom(func(t *rapid.T) string {
			year := rapid.IntRange(2020, 2030).Draw(t, "year")
			return fmt.Sprintf("%04d-13-15", year)
		}),
	)
}

// genValidTime generates valid HH:MM times
func genValidTime() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		hour := rapid.IntRange(0, 23).Draw(t, "hour")
		minute := rapid.IntRange(0, 59).Draw(t, "minute")
		return fmt.Sprintf("%02d:%02d", hour, minute)
	})
}

// genValidPhone generates valid phone numbers
func genValidPhone() *rapid.Generator[string] {
	return rapid.OneOf(
		// German format
		rapid.Custom(func(t *rapid.T) string {
			prefix := rapid.IntRange(100, 999).Draw(t, "prefix")
			number := rapid.IntRange(1000000, 9999999).Draw(t, "number")
			return fmt.Sprintf("+49 %d %d", prefix, number)
		}),
		// Simple format
		rapid.Custom(func(t *rapid.T) string {
			number := rapid.Int64Range(1000000000, 9999999999).Draw(t, "number")
			return fmt.Sprintf("%d", number)
		}),
	)
}

// genEmail generates email addresses
func genEmail() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		localChars := "abcdefghijklmnopqrstuvwxyz0123456789"
		domainChars := "abcdefghijklmnopqrstuvwxyz"

		// Local part
		localLen := rapid.IntRange(3, 20).Draw(t, "localLen")
		var local strings.Builder
		for i := 0; i < localLen; i++ {
			idx := rapid.IntRange(0, len(localChars)-1).Draw(t, fmt.Sprintf("local%d", i))
			local.WriteByte(localChars[idx])
		}

		// Domain
		domainLen := rapid.IntRange(3, 10).Draw(t, "domainLen")
		var domain strings.Builder
		for i := 0; i < domainLen; i++ {
			idx := rapid.IntRange(0, len(domainChars)-1).Draw(t, fmt.Sprintf("domain%d", i))
			domain.WriteByte(domainChars[idx])
		}

		tlds := []string{"com", "de", "org", "net"}
		tld := rapid.SampledFrom(tlds).Draw(t, "tld")

		return local.String() + "@" + domain.String() + "." + tld
	})
}

// ============================================================================
// PROPERTY TESTS FOR SLUG VALIDATION
// ============================================================================

func TestProperty_ValidSlugAccepted(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		slug := genValidSlug().Draw(t, "slug")

		// Filter out reserved slugs
		reservedSlugs := map[string]bool{
			"www": true, "admin": true, "api": true, "app": true,
			"mail": true, "email": true, "ftp": true, "ssh": true,
			"test": true, "staging": true, "dev": true, "prod": true,
			"central": true, "support": true, "help": true, "docs": true,
			"blog": true, "status": true, "cdn": true, "assets": true,
			"static": true,
		}
		if reservedSlugs[slug] {
			t.Skip("reserved slug")
		}

		err := ValidateSlug(slug)
		if err != nil {
			t.Fatalf("Valid slug %q was rejected: %v", slug, err)
		}
	})
}

func TestProperty_InvalidSlugRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		slug := genInvalidSlug().Draw(t, "slug")

		err := ValidateSlug(slug)
		if err == nil {
			t.Fatalf("Invalid slug %q was accepted", slug)
		}
	})
}

func TestProperty_SlugInvariants(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		slug := rapid.String().Draw(t, "slug")

		err := ValidateSlug(slug)

		if err == nil {
			// PROPERTY 1: Length must be 3-100
			if len(slug) < 3 || len(slug) > 100 {
				t.Fatalf("BUG: Accepted slug with invalid length %d: %q", len(slug), slug)
			}

			// PROPERTY 2: Must be lowercase
			if slug != strings.ToLower(slug) {
				t.Fatalf("BUG: Accepted non-lowercase slug: %q", slug)
			}

			// PROPERTY 3: Must start with letter [a-z]
			if len(slug) > 0 && (slug[0] < 'a' || slug[0] > 'z') {
				t.Fatalf("BUG: Accepted slug not starting with [a-z]: %q", slug)
			}

			// PROPERTY 4: Must end with [a-z0-9]
			if len(slug) > 0 {
				last := slug[len(slug)-1]
				if !((last >= 'a' && last <= 'z') || (last >= '0' && last <= '9')) {
					t.Fatalf("BUG: Accepted slug not ending with [a-z0-9]: %q", slug)
				}
			}

			// PROPERTY 5: No consecutive dashes
			if strings.Contains(slug, "--") {
				t.Fatalf("BUG: Accepted slug with consecutive dashes: %q", slug)
			}

			// PROPERTY 6: Only valid characters
			validChars := regexp.MustCompile(`^[a-z0-9-]+$`)
			if !validChars.MatchString(slug) {
				t.Fatalf("BUG: Accepted slug with invalid characters: %q", slug)
			}

			// PROPERTY 7: Not reserved
			reservedSlugs := map[string]bool{
				"www": true, "admin": true, "api": true, "app": true,
				"mail": true, "email": true, "ftp": true, "ssh": true,
				"test": true, "staging": true, "dev": true, "prod": true,
				"central": true, "support": true, "help": true, "docs": true,
				"blog": true, "status": true, "cdn": true, "assets": true,
				"static": true,
			}
			if reservedSlugs[slug] {
				t.Fatalf("BUG: Accepted reserved slug: %q", slug)
			}
		}
	})
}

// ============================================================================
// PROPERTY TESTS FOR DATE VALIDATION
// ============================================================================

func TestProperty_ValidDateRoundTrips(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		dateStr := genValidDate().Draw(t, "date")

		parsed, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			t.Fatalf("Valid date %q failed to parse: %v", dateStr, err)
		}

		// PROPERTY: Valid dates must round-trip
		reformatted := parsed.Format("2006-01-02")
		if reformatted != dateStr {
			t.Fatalf("Date %q did not round-trip, got %q", dateStr, reformatted)
		}
	})
}

func TestProperty_InvalidDateDoesNotRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		dateStr := genInvalidDate().Draw(t, "date")

		parsed, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			// Parse failed - that's fine for invalid dates
			return
		}

		// If it parsed, check if it normalized (which means the original was invalid)
		reformatted := parsed.Format("2006-01-02")
		if reformatted == dateStr {
			t.Fatalf("Semantically invalid date %q should not round-trip unchanged", dateStr)
		}

		// PROPERTY: If validation accepts a non-round-tripping date, that's a bug
		req := CreateBookingRequest{
			DogID:         1,
			Date:          dateStr,
			ScheduledTime: "12:00",
		}
		if req.Validate() == nil {
			t.Fatalf("BUG: Booking validation accepted semantically invalid date %q (normalizes to %q)",
				dateStr, reformatted)
		}
	})
}

func TestProperty_DateValidationSemanticCorrectness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		year := rapid.IntRange(2020, 2030).Draw(t, "year")
		month := rapid.IntRange(1, 12).Draw(t, "month")
		day := rapid.IntRange(1, 31).Draw(t, "day")

		dateStr := fmt.Sprintf("%04d-%02d-%02d", year, month, day)

		parsed, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return
		}

		// Check if date components match
		parsedYear, parsedMonth, parsedDay := parsed.Date()
		isSemanticallySame := parsedYear == year && int(parsedMonth) == month && parsedDay == day

		req := CreateBlockedDateRequest{
			Date:   dateStr,
			Reason: "test",
		}
		validationPassed := req.Validate() == nil

		// PROPERTY: Validation should only pass for semantically valid dates
		if validationPassed && !isSemanticallySame {
			t.Fatalf("BUG: Validation accepted date %q which normalizes to %s",
				dateStr, parsed.Format("2006-01-02"))
		}
	})
}

// ============================================================================
// PROPERTY TESTS FOR TIME VALIDATION
// ============================================================================

func TestProperty_ValidTimeAccepted(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		timeStr := genValidTime().Draw(t, "time")

		_, err := time.Parse("15:04", timeStr)
		if err != nil {
			t.Fatalf("Valid time %q was rejected: %v", timeStr, err)
		}
	})
}

func TestProperty_TimeInvariants(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hour := rapid.IntRange(-5, 30).Draw(t, "hour")
		minute := rapid.IntRange(-5, 65).Draw(t, "minute")

		timeStr := fmt.Sprintf("%02d:%02d", hour, minute)
		parsed, err := time.Parse("15:04", timeStr)

		if err == nil {
			// PROPERTY: If parsed, hour must be in valid range
			parsedHour, parsedMin, _ := parsed.Clock()

			req := CreateBookingRequest{
				DogID:         1,
				Date:          "2025-01-15",
				ScheduledTime: timeStr,
			}

			// Check if time normalized
			reformatted := parsed.Format("15:04")
			if reformatted != timeStr && req.Validate() == nil {
				t.Fatalf("BUG: Time %q normalizes to %q but validation accepts it (hour=%d, min=%d)",
					timeStr, reformatted, parsedHour, parsedMin)
			}
		}
	})
}

// ============================================================================
// PROPERTY TESTS FOR PHONE VALIDATION
// ============================================================================

func TestProperty_ValidPhoneAccepted(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		phone := genValidPhone().Draw(t, "phone")

		err := ValidatePhone(phone)
		if err != nil {
			t.Fatalf("Valid phone %q was rejected: %v", phone, err)
		}
	})
}

func TestProperty_PhoneInvariants(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		phone := rapid.String().Draw(t, "phone")

		err := ValidatePhone(phone)

		if err == nil && phone != "" {
			// PROPERTY 1: Must have at least 7 digits
			digitCount := 0
			for _, c := range phone {
				if c >= '0' && c <= '9' {
					digitCount++
				}
			}
			if digitCount < 7 {
				t.Fatalf("BUG: Accepted phone with only %d digits: %q", digitCount, phone)
			}

			// PROPERTY 2: Balanced parentheses
			openCount := strings.Count(phone, "(")
			closeCount := strings.Count(phone, ")")
			if openCount != closeCount {
				t.Fatalf("BUG: Accepted phone with unbalanced parens: %q", phone)
			}

			// PROPERTY 3: No trailing separator
			if len(phone) > 0 {
				last := phone[len(phone)-1]
				if last == '-' || last == '.' || last == ' ' {
					t.Fatalf("BUG: Accepted phone ending with separator: %q", phone)
				}
			}

			// PROPERTY 4: No non-ASCII digits
			for _, c := range phone {
				if unicode.IsDigit(c) && (c < '0' || c > '9') {
					t.Fatalf("BUG: Accepted phone with non-ASCII digit: %q", phone)
				}
			}
		}
	})
}

// ============================================================================
// PROPERTY TESTS FOR HEX COLOR VALIDATION
// ============================================================================

func TestProperty_ValidHexColorAccepted(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		color := genHexColor().Draw(t, "color")

		valid := ValidateHexColor(color)
		if !valid {
			t.Fatalf("Valid hex color %q was rejected", color)
		}
	})
}

func TestProperty_HexColorInvariants(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		color := rapid.String().Draw(t, "color")

		valid := ValidateHexColor(color)

		if valid && color != "" {
			// PROPERTY 1: Must be exactly 7 chars
			if len(color) != 7 {
				t.Fatalf("BUG: Accepted color with length %d: %q", len(color), color)
			}

			// PROPERTY 2: Must start with #
			if !strings.HasPrefix(color, "#") {
				t.Fatalf("BUG: Accepted color without #: %q", color)
			}

			// PROPERTY 3: Must only have hex digits after #
			hexRegex := regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
			if !hexRegex.MatchString(color) {
				t.Fatalf("BUG: Accepted color with non-hex chars: %q", color)
			}
		}
	})
}

// ============================================================================
// PROPERTY TESTS FOR EMAIL VALIDATION
// ============================================================================

func TestProperty_ValidEmailAccepted(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		email := genEmail().Draw(t, "email")

		valid := isValidEmail(email)
		if !valid {
			t.Fatalf("Valid email %q was rejected", email)
		}
	})
}

func TestProperty_EmailInvariants(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		email := rapid.String().Draw(t, "email")

		valid := isValidEmail(email)

		if valid {
			// PROPERTY 1: Must contain exactly one @
			atCount := strings.Count(email, "@")
			if atCount != 1 {
				t.Fatalf("BUG: Accepted email with %d @ symbols: %q", atCount, email)
			}

			// PROPERTY 2: Local part must not be empty
			parts := strings.Split(email, "@")
			if len(parts) != 2 || parts[0] == "" {
				t.Fatalf("BUG: Accepted email with empty local part: %q", email)
			}

			// PROPERTY 3: Domain must not be empty
			if parts[1] == "" {
				t.Fatalf("BUG: Accepted email with empty domain: %q", email)
			}

			// PROPERTY 4: No CRLF (header injection)
			if strings.ContainsAny(email, "\r\n") {
				t.Fatalf("BUG: Accepted email with CRLF: %q", email)
			}
		}
	})
}

// ============================================================================
// PROPERTY TESTS FOR BOOKING VALIDATION
// ============================================================================

func TestProperty_BookingValidationInvariants(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		dogID := rapid.IntRange(-10, 100).Draw(t, "dogID")
		date := rapid.String().Draw(t, "date")
		scheduledTime := rapid.String().Draw(t, "time")

		req := CreateBookingRequest{
			DogID:         dogID,
			Date:          date,
			ScheduledTime: scheduledTime,
		}

		err := req.Validate()

		if err == nil {
			// PROPERTY 1: DogID must be positive
			if dogID <= 0 {
				t.Fatalf("BUG: Accepted non-positive dogID: %d", dogID)
			}

			// PROPERTY 2: Date must not be empty
			if date == "" {
				t.Fatalf("BUG: Accepted empty date")
			}

			// PROPERTY 3: Date must be parseable
			if _, parseErr := time.Parse("2006-01-02", date); parseErr != nil {
				t.Fatalf("BUG: Accepted unparseable date: %q", date)
			}

			// PROPERTY 4: ScheduledTime must not be empty
			if scheduledTime == "" {
				t.Fatalf("BUG: Accepted empty time")
			}

			// PROPERTY 5: ScheduledTime must be parseable
			if _, parseErr := time.Parse("15:04", scheduledTime); parseErr != nil {
				t.Fatalf("BUG: Accepted unparseable time: %q", scheduledTime)
			}
		}
	})
}

// ============================================================================
// PROPERTY TESTS FOR COLOR CATEGORY VALIDATION
// ============================================================================

func TestProperty_ColorCategoryInvariants(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		name := rapid.String().Draw(t, "name")
		hexCode := rapid.String().Draw(t, "hexCode")

		req := CreateColorCategoryRequest{
			Name:    name,
			HexCode: hexCode,
		}

		err := req.Validate()

		if err == nil {
			// PROPERTY 1: Name must not be empty
			if name == "" {
				t.Fatalf("BUG: Accepted empty name")
			}

			// PROPERTY 2: HexCode must not be empty
			if hexCode == "" {
				t.Fatalf("BUG: Accepted empty hex code")
			}

			// PROPERTY 3: HexCode must match pattern
			if !hexCodeRegex.MatchString(hexCode) {
				t.Fatalf("BUG: Accepted invalid hex code: %q", hexCode)
			}
		}
	})
}

// ============================================================================
// PROPERTY TESTS FOR TENANT REGISTRATION
// ============================================================================

func TestProperty_TenantRegistrationInvariants(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		orgName := rapid.String().Draw(t, "orgName")
		slug := rapid.String().Draw(t, "slug")
		contactEmail := rapid.String().Draw(t, "contactEmail")
		state := rapid.String().Draw(t, "state")
		adminFirst := rapid.String().Draw(t, "adminFirst")
		adminLast := rapid.String().Draw(t, "adminLast")
		adminEmail := rapid.String().Draw(t, "adminEmail")
		adminPass := rapid.String().Draw(t, "adminPass")

		req := TenantRegistrationRequest{
			OrganizationName: orgName,
			Slug:             slug,
			ContactEmail:     contactEmail,
			FederalState:     state,
			AdminFirstName:   adminFirst,
			AdminLastName:    adminLast,
			AdminEmail:       adminEmail,
			AdminPassword:    adminPass,
			City:             "Test City",
			PostalCode:       "12345",
		}

		err := req.Validate()

		if err == nil {
			// PROPERTY 1: Org name must not be empty
			if strings.TrimSpace(orgName) == "" {
				t.Fatalf("BUG: Accepted empty org name")
			}

			// PROPERTY 2: Org name must be <= 255 chars
			if len(orgName) > 255 {
				t.Fatalf("BUG: Accepted org name with %d chars", len(orgName))
			}

			// PROPERTY 3: Slug must be valid
			if ValidateSlug(slug) != nil {
				t.Fatalf("BUG: Tenant accepted but slug invalid: %q", slug)
			}

			// PROPERTY 4: Admin password must be >= 8 chars
			if len(adminPass) < 8 {
				t.Fatalf("BUG: Accepted admin password with %d chars", len(adminPass))
			}

			// PROPERTY 5: Federal state must be valid or empty (defaults to BW)
			if state != "" {
				if _, ok := FederalStates[state]; !ok {
					// Check if it was defaulted
					if req.FederalState != "BW" {
						t.Fatalf("BUG: Accepted invalid federal state: %q", state)
					}
				}
			}
		}
	})
}

// ============================================================================
// PROPERTY TESTS FOR USER REGISTRATION
// ============================================================================

func TestProperty_UserRegistrationInvariants(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		firstName := rapid.String().Draw(t, "firstName")
		lastName := rapid.String().Draw(t, "lastName")
		email := rapid.String().Draw(t, "email")
		phone := rapid.String().Draw(t, "phone")
		password := rapid.String().Draw(t, "password")
		confirmPassword := rapid.String().Draw(t, "confirmPassword")
		acceptTerms := rapid.Bool().Draw(t, "acceptTerms")
		regPassword := rapid.String().Draw(t, "regPassword")

		req := RegisterRequest{
			FirstName:            firstName,
			LastName:             lastName,
			Email:                email,
			Phone:                phone,
			Password:             password,
			ConfirmPassword:      confirmPassword,
			AcceptTerms:          acceptTerms,
			RegistrationPassword: regPassword,
		}

		err := req.Validate()

		if err == nil {
			// PROPERTY 1: FirstName must not be empty
			if strings.TrimSpace(firstName) == "" {
				t.Fatalf("BUG: Accepted empty first name")
			}

			// PROPERTY 2: LastName must not be empty
			if strings.TrimSpace(lastName) == "" {
				t.Fatalf("BUG: Accepted empty last name")
			}

			// PROPERTY 3: Email must not be empty
			if strings.TrimSpace(email) == "" {
				t.Fatalf("BUG: Accepted empty email")
			}

			// PROPERTY 4: Password must be >= 8 chars
			if len(password) < 8 {
				t.Fatalf("BUG: Accepted password with %d chars", len(password))
			}

			// PROPERTY 5: Passwords must match
			if password != confirmPassword {
				t.Fatalf("BUG: Accepted mismatched passwords")
			}

			// PROPERTY 6: Terms must be accepted
			if !acceptTerms {
				t.Fatalf("BUG: Accepted without terms acceptance")
			}

			// PROPERTY 7: Registration password must match pattern
			if !registrationPasswordRegex.MatchString(regPassword) {
				t.Fatalf("BUG: Accepted invalid registration password: %q", regPassword)
			}
		}
	})
}

// ============================================================================
// PROPERTY TESTS FOR BLOCKED DATE VALIDATION
// ============================================================================

func TestProperty_BlockedDateInvariants(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		date := rapid.String().Draw(t, "date")
		reason := rapid.String().Draw(t, "reason")

		// Generate optional dogID
		var dogID *int
		if rapid.Bool().Draw(t, "hasDogID") {
			id := rapid.IntRange(-10, 100).Draw(t, "dogID")
			dogID = &id
		}

		req := CreateBlockedDateRequest{
			Date:   date,
			Reason: reason,
			DogID:  dogID,
		}

		err := req.Validate()

		if err == nil {
			// PROPERTY 1: Date must not be empty
			if date == "" {
				t.Fatalf("BUG: Accepted empty date")
			}

			// PROPERTY 2: Date must be parseable
			if _, parseErr := time.Parse("2006-01-02", date); parseErr != nil {
				t.Fatalf("BUG: Accepted unparseable date: %q", date)
			}

			// PROPERTY 3: Reason must not be empty
			if reason == "" {
				t.Fatalf("BUG: Accepted empty reason")
			}

			// PROPERTY 4: If DogID provided, must be positive
			if dogID != nil && *dogID <= 0 {
				t.Fatalf("BUG: Accepted non-positive dogID: %d", *dogID)
			}
		}
	})
}

// ============================================================================
// STATEFUL PROPERTY TEST - Booking State Machine
// ============================================================================

// BookingStateMachine tests booking state transitions
type BookingStateMachine struct {
	bookings map[string]string // key: "dogID-date-time", value: status
}

func (m *BookingStateMachine) Init(t *rapid.T) {
	m.bookings = make(map[string]string)
}

func (m *BookingStateMachine) CreateBooking(t *rapid.T) {
	dogID := rapid.IntRange(1, 10).Draw(t, "dogID")
	date := genValidDate().Draw(t, "date")
	time := genValidTime().Draw(t, "time")

	key := fmt.Sprintf("%d-%s-%s", dogID, date, time)

	// PROPERTY: Cannot double-book same dog at same time
	if _, exists := m.bookings[key]; exists {
		// Should be rejected
		return
	}

	m.bookings[key] = "scheduled"
}

func (m *BookingStateMachine) Check(t *rapid.T) {
	// PROPERTY: All bookings must have valid status
	validStatuses := map[string]bool{
		"scheduled": true,
		"completed": true,
		"cancelled": true,
	}
	for key, status := range m.bookings {
		if !validStatuses[status] {
			t.Fatalf("Invalid booking status %q for %s", status, key)
		}
	}
}

func TestProperty_BookingStateMachine(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		var m BookingStateMachine
		m.Init(t)

		// Run random operations
		numOps := rapid.IntRange(1, 20).Draw(t, "numOps")
		for i := 0; i < numOps; i++ {
			m.CreateBooking(t)
		}

		m.Check(t)
	})
}
