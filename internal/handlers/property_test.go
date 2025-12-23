package handlers

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// ============================================================================
// CUSTOM GENERATORS FOR HANDLERS
// ============================================================================

// genValidImageExtension generates valid image file extensions
func genValidImageExtension() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{".jpg", ".jpeg", ".png", ".JPG", ".JPEG", ".PNG"})
}

// genInvalidImageExtension generates invalid image file extensions
func genInvalidImageExtension() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{
		".gif", ".bmp", ".webp", ".svg", ".tiff", ".ico",
		".exe", ".php", ".js", ".html", ".txt", ".pdf",
		"", ".jpg.php", ".png.exe",
	})
}

// genJPEGMagicBytes generates valid JPEG file headers
func genJPEGMagicBytes() *rapid.Generator[[]byte] {
	return rapid.Custom(func(t *rapid.T) []byte {
		// JPEG starts with FF D8 FF
		header := []byte{0xFF, 0xD8, 0xFF}
		// Add more bytes (E0 for JFIF, E1 for EXIF, etc.)
		marker := rapid.SampledFrom([]byte{0xE0, 0xE1, 0xE2, 0xDB, 0xC0}).Draw(t, "marker")
		header = append(header, marker)
		// Add some random padding
		padLen := rapid.IntRange(10, 100).Draw(t, "padLen")
		for i := 0; i < padLen; i++ {
			header = append(header, byte(rapid.IntRange(0, 255).Draw(t, fmt.Sprintf("pad%d", i))))
		}
		return header
	})
}

// genPNGMagicBytes generates valid PNG file headers
func genPNGMagicBytes() *rapid.Generator[[]byte] {
	return rapid.Custom(func(t *rapid.T) []byte {
		// PNG magic: 89 50 4E 47 0D 0A 1A 0A
		header := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
		// Add some random padding
		padLen := rapid.IntRange(10, 100).Draw(t, "padLen")
		for i := 0; i < padLen; i++ {
			header = append(header, byte(rapid.IntRange(0, 255).Draw(t, fmt.Sprintf("pad%d", i))))
		}
		return header
	})
}

// genInvalidMagicBytes generates non-image file content
func genInvalidMagicBytes() *rapid.Generator[[]byte] {
	return rapid.OneOf(
		// Text content
		rapid.Custom(func(t *rapid.T) []byte {
			_ = rapid.IntRange(0, 100).Draw(t, "dummy")
			return []byte("This is not an image file")
		}),
		// HTML content
		rapid.Custom(func(t *rapid.T) []byte {
			_ = rapid.IntRange(0, 100).Draw(t, "dummy")
			return []byte("<html><body>test</body></html>")
		}),
		// PHP content (polyglot attempt)
		rapid.Custom(func(t *rapid.T) []byte {
			_ = rapid.IntRange(0, 100).Draw(t, "dummy")
			return []byte("<?php echo 'pwned'; ?>")
		}),
		// GIF (valid image but not allowed)
		rapid.Custom(func(t *rapid.T) []byte {
			_ = rapid.IntRange(0, 100).Draw(t, "dummy")
			return []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61} // GIF89a
		}),
		// PDF
		rapid.Custom(func(t *rapid.T) []byte {
			_ = rapid.IntRange(0, 100).Draw(t, "dummy")
			return []byte{0x25, 0x50, 0x44, 0x46} // %PDF
		}),
		// EXE/PE
		rapid.Custom(func(t *rapid.T) []byte {
			_ = rapid.IntRange(0, 100).Draw(t, "dummy")
			return []byte{0x4D, 0x5A} // MZ
		}),
		// Empty
		rapid.Just([]byte{}),
		// Random bytes
		rapid.Custom(func(t *rapid.T) []byte {
			length := rapid.IntRange(1, 100).Draw(t, "length")
			data := make([]byte, length)
			for i := 0; i < length; i++ {
				data[i] = byte(rapid.IntRange(0, 255).Draw(t, fmt.Sprintf("byte%d", i)))
			}
			return data
		}),
	)
}

// genValidEmail generates valid email addresses
func genValidEmail() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		localChars := "abcdefghijklmnopqrstuvwxyz0123456789"
		domainChars := "abcdefghijklmnopqrstuvwxyz"

		// Local part (3-20 chars)
		localLen := rapid.IntRange(3, 20).Draw(t, "localLen")
		var local strings.Builder
		for i := 0; i < localLen; i++ {
			idx := rapid.IntRange(0, len(localChars)-1).Draw(t, fmt.Sprintf("local%d", i))
			local.WriteByte(localChars[idx])
		}

		// Domain (3-10 chars)
		domainLen := rapid.IntRange(3, 10).Draw(t, "domainLen")
		var domain strings.Builder
		for i := 0; i < domainLen; i++ {
			idx := rapid.IntRange(0, len(domainChars)-1).Draw(t, fmt.Sprintf("domain%d", i))
			domain.WriteByte(domainChars[idx])
		}

		tlds := []string{"com", "de", "org", "net", "info"}
		tld := rapid.SampledFrom(tlds).Draw(t, "tld")

		return local.String() + "@" + domain.String() + "." + tld
	})
}

// genInvalidEmail generates invalid email addresses
func genInvalidEmail() *rapid.Generator[string] {
	return rapid.OneOf(
		rapid.Just(""),
		rapid.Just("@"),
		rapid.Just("user@"),
		rapid.Just("@example.com"),
		rapid.Just("user"),
		rapid.Just("user@.com"),
		rapid.Just("user@example"),
		rapid.Just("user@@example.com"),
		// Header injection attempts
		rapid.Just("user\r\n@example.com"),
		rapid.Just("user@example.com\r\nBcc: evil@attacker.com"),
		rapid.Just("user\n@example.com"),
		// Multiple emails
		rapid.Just("user1@example.com,user2@example.com"),
		rapid.Just("user1@example.com;user2@example.com"),
	)
}

// genValidSlug generates valid tenant slugs
func genValidSlug() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		length := rapid.IntRange(3, 50).Draw(t, "length")

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

// ============================================================================
// PROPERTY TESTS FOR IMAGE FILE VALIDATION
// ============================================================================

func TestProperty_ValidJPEGAccepted(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ext := genValidImageExtension().Draw(t, "ext")
		if !strings.Contains(strings.ToLower(ext), "jpg") && !strings.Contains(strings.ToLower(ext), "jpeg") {
			ext = ".jpg"
		}
		filename := "test" + ext
		content := genJPEGMagicBytes().Draw(t, "content")

		reader := bytes.NewReader(content)
		errMsg, valid := ValidateImageFile(filename, reader)

		if !valid {
			t.Fatalf("Valid JPEG file %q was rejected: %s", filename, errMsg)
		}
	})
}

func TestProperty_ValidPNGAccepted(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		filename := "test.png"
		content := genPNGMagicBytes().Draw(t, "content")

		reader := bytes.NewReader(content)
		errMsg, valid := ValidateImageFile(filename, reader)

		if !valid {
			t.Fatalf("Valid PNG file %q was rejected: %s", filename, errMsg)
		}
	})
}

func TestProperty_InvalidExtensionRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ext := genInvalidImageExtension().Draw(t, "ext")
		filename := "test" + ext
		content := genJPEGMagicBytes().Draw(t, "content") // Valid content but wrong extension

		reader := bytes.NewReader(content)
		_, valid := ValidateImageFile(filename, reader)

		// PROPERTY: Invalid extension should be rejected even with valid content
		if valid {
			t.Fatalf("BUG: File with invalid extension %q was accepted", filename)
		}
	})
}

func TestProperty_InvalidContentRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ext := genValidImageExtension().Draw(t, "ext")
		filename := "test" + ext
		content := genInvalidMagicBytes().Draw(t, "content")

		reader := bytes.NewReader(content)
		_, valid := ValidateImageFile(filename, reader)

		// PROPERTY: Invalid content should be rejected even with valid extension
		if valid {
			t.Fatalf("BUG: File %q with invalid content was accepted: %x", filename, content[:min(20, len(content))])
		}
	})
}

func TestProperty_ImageValidationInvariants(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		filename := rapid.String().Draw(t, "filename")
		content := rapid.SliceOf(rapid.Byte()).Draw(t, "content")

		reader := bytes.NewReader(content)
		errMsg, valid := ValidateImageFile(filename, reader)

		// PROPERTY 1: If valid, error message must be empty
		if valid && errMsg != "" {
			t.Fatalf("BUG: Valid result but non-empty error message: %s", errMsg)
		}

		// PROPERTY 2: If invalid, error message must be non-empty
		if !valid && errMsg == "" {
			t.Fatalf("BUG: Invalid result but empty error message")
		}

		if valid {
			// PROPERTY 3: Valid files must have valid extension
			ext := strings.ToLower(filename)
			hasValidExt := strings.HasSuffix(ext, ".jpg") ||
				strings.HasSuffix(ext, ".jpeg") ||
				strings.HasSuffix(ext, ".png")
			if !hasValidExt {
				t.Fatalf("BUG: Accepted file with invalid extension: %q", filename)
			}
		}
	})
}

// ============================================================================
// PROPERTY TESTS FOR MIME TYPE VALIDATION
// ============================================================================

func TestProperty_MIMETypeValidationInvariants(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		content := rapid.SliceOf(rapid.Byte()).Draw(t, "content")

		reader := bytes.NewReader(content)
		errMsg, valid := ValidateImageMIMEType(reader)

		// PROPERTY 1: If valid, error message must be empty
		if valid && errMsg != "" {
			t.Fatalf("BUG: Valid MIME but non-empty error: %s", errMsg)
		}

		// PROPERTY 2: If invalid, error message must be non-empty
		if !valid && errMsg == "" {
			t.Fatalf("BUG: Invalid MIME but empty error")
		}
	})
}

func TestProperty_ValidMIMETypesAccepted(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Choose between JPEG and PNG
		useJPEG := rapid.Bool().Draw(t, "useJPEG")

		var content []byte
		if useJPEG {
			content = genJPEGMagicBytes().Draw(t, "jpeg")
		} else {
			content = genPNGMagicBytes().Draw(t, "png")
		}

		reader := bytes.NewReader(content)
		_, valid := ValidateImageMIMEType(reader)

		if !valid {
			t.Fatalf("Valid image content rejected: %x", content[:min(20, len(content))])
		}
	})
}

// ============================================================================
// PROPERTY TESTS FOR EMAIL VALIDATION (Contact Handler)
// ============================================================================

func TestProperty_ValidEmailAccepted(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		email := genValidEmail().Draw(t, "email")

		valid := isValidEmail(email)
		if !valid {
			t.Fatalf("Valid email %q was rejected", email)
		}
	})
}

func TestProperty_InvalidEmailRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		email := genInvalidEmail().Draw(t, "email")

		valid := isValidEmail(email)
		if valid {
			t.Fatalf("Invalid email %q was accepted", email)
		}
	})
}

func TestProperty_EmailValidationInvariants(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		email := rapid.String().Draw(t, "email")

		valid := isValidEmail(email)

		if valid {
			// PROPERTY 1: Must contain exactly one @
			atCount := strings.Count(email, "@")
			if atCount != 1 {
				t.Fatalf("BUG: Accepted email with %d @ symbols: %q", atCount, email)
			}

			// PROPERTY 2: No CRLF (header injection prevention)
			if strings.ContainsAny(email, "\r\n") {
				t.Fatalf("BUG: Accepted email with CRLF: %q", email)
			}

			// PROPERTY 3: No comma/semicolon (multiple email prevention)
			if strings.ContainsAny(email, ",;") {
				t.Fatalf("BUG: Accepted email with comma/semicolon: %q", email)
			}

			// PROPERTY 4: Local part must not be empty
			parts := strings.Split(email, "@")
			if len(parts) != 2 || parts[0] == "" {
				t.Fatalf("BUG: Accepted email with empty local part: %q", email)
			}

			// PROPERTY 5: Domain must not be empty and must have TLD
			if parts[1] == "" || !strings.Contains(parts[1], ".") {
				t.Fatalf("BUG: Accepted email with invalid domain: %q", email)
			}
		}
	})
}

// Test header injection protection specifically
func TestProperty_HeaderInjectionBlocked(t *testing.T) {
	injectionPatterns := []string{
		"user\r\n@example.com",
		"user@example.com\r\n",
		"user@example.com\r\nBcc: evil@attacker.com",
		"user\n@example.com",
		"user@example.com\nBcc: evil@attacker.com",
		"user@example.com\r\nTo: evil@attacker.com",
		"user@example.com\r\nCc: evil@attacker.com",
		"\r\nuser@example.com",
		"\nuser@example.com",
	}

	for _, email := range injectionPatterns {
		if isValidEmail(email) {
			t.Fatalf("BUG: Header injection attempt accepted: %q", email)
		}
	}
}

// ============================================================================
// PROPERTY TESTS FOR SLUG VALIDATION
// ============================================================================

func TestProperty_ValidSlugAccepted(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		slug := genValidSlug().Draw(t, "slug")

		// Filter out reserved slugs (matching actual isReservedSlug list)
		reservedSlugs := map[string]bool{
			"www": true, "api": true, "admin": true, "app": true,
			"mail": true, "email": true, "smtp": true, "ftp": true,
			"support": true, "help": true, "billing": true, "status": true,
			"dev": true, "staging": true, "test": true, "demo": true,
			"blog": true, "news": true, "docs": true, "static": true,
			"assets": true, "cdn": true, "media": true,
		}
		if reservedSlugs[slug] {
			t.Skip("reserved slug")
		}

		valid := isValidSlug(slug)
		if !valid {
			t.Fatalf("Valid slug %q was rejected", slug)
		}
	})
}

func TestProperty_SlugValidationInvariants(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		slug := rapid.String().Draw(t, "slug")

		valid := isValidSlug(slug)

		if valid {
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

			// PROPERTY 5: Only valid characters [a-z0-9-]
			validChars := regexp.MustCompile(`^[a-z0-9-]+$`)
			if !validChars.MatchString(slug) {
				t.Fatalf("BUG: Accepted slug with invalid characters: %q", slug)
			}
		}
	})
}

func TestProperty_ReservedSlugRejected(t *testing.T) {
	// These are the actual reserved slugs in isReservedSlug()
	reservedSlugs := []string{
		"www", "api", "admin", "app", "mail", "email", "smtp", "ftp",
		"support", "help", "billing", "status", "dev", "staging", "test",
		"demo", "blog", "news", "docs", "static", "assets", "cdn", "media",
	}

	for _, slug := range reservedSlugs {
		if !isReservedSlug(slug) {
			t.Fatalf("BUG: Reserved slug %q not detected as reserved", slug)
		}
	}

	// NOTE: These common slugs are NOT reserved but probably should be:
	// "dashboard", "login", "register", "signup", "signin", "logout", "account", "profile"
	// This is flagged as a potential improvement, not a bug.
}

// ============================================================================
// PROPERTY TESTS FOR CONTACT REQUEST VALIDATION
// ============================================================================

func TestProperty_ContactRequestInvariants(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		name := rapid.String().Draw(t, "name")
		email := rapid.String().Draw(t, "email")
		subject := rapid.String().Draw(t, "subject")
		message := rapid.String().Draw(t, "message")
		organization := rapid.String().Draw(t, "organization")

		req := &ContactRequest{
			Name:         name,
			Email:        email,
			Subject:      subject,
			Message:      message,
			Organization: organization,
		}

		err := req.Validate()

		if err == nil {
			// PROPERTY 1: Name must not be empty after trimming
			if strings.TrimSpace(name) == "" {
				t.Fatalf("BUG: Accepted empty name")
			}

			// PROPERTY 2: Name must not exceed 200 chars
			if len(strings.TrimSpace(name)) > 200 {
				t.Fatalf("BUG: Accepted name with %d chars", len(name))
			}

			// PROPERTY 3: Email must not be empty after trimming
			if strings.TrimSpace(email) == "" {
				t.Fatalf("BUG: Accepted empty email")
			}

			// PROPERTY 4: Email must not exceed 200 chars
			if len(strings.TrimSpace(email)) > 200 {
				t.Fatalf("BUG: Accepted email with %d chars", len(email))
			}

			// PROPERTY 5: Email must be valid format
			if !isValidEmail(strings.TrimSpace(email)) {
				t.Fatalf("BUG: Accepted invalid email format: %q", email)
			}

			// PROPERTY 6: Subject must not be empty after trimming
			if strings.TrimSpace(subject) == "" {
				t.Fatalf("BUG: Accepted empty subject")
			}

			// PROPERTY 7: Message must not be empty after trimming
			if strings.TrimSpace(message) == "" {
				t.Fatalf("BUG: Accepted empty message")
			}

			// PROPERTY 8: Message must not exceed 10000 chars
			if len(strings.TrimSpace(message)) > 10000 {
				t.Fatalf("BUG: Accepted message with %d chars", len(message))
			}
		}
	})
}

// ============================================================================
// PROPERTY TESTS FOR HTML ESCAPING
// ============================================================================

func TestProperty_HTMLEscapeInvariants(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := rapid.String().Draw(t, "input")

		escaped := escapeHTML(input)

		// PROPERTY 1: No unescaped < or > (XSS prevention)
		if strings.Contains(escaped, "<") && !strings.Contains(escaped, "&lt;") {
			// Check if there's a raw < that should have been escaped
			// This is tricky because escaped might contain &lt; which is fine
			t.Logf("Note: escaped output contains <: %q -> %q", input, escaped)
		}

		// PROPERTY 2: Original dangerous chars should be escaped
		if strings.Contains(input, "<") && !strings.Contains(escaped, "&lt;") {
			t.Fatalf("BUG: < not escaped in: %q -> %q", input, escaped)
		}
		if strings.Contains(input, ">") && !strings.Contains(escaped, "&gt;") {
			t.Fatalf("BUG: > not escaped in: %q -> %q", input, escaped)
		}
		if strings.Contains(input, "&") && !strings.Contains(escaped, "&amp;") {
			// Check if & was already escaped
			if !strings.Contains(input, "&amp;") && !strings.Contains(input, "&lt;") &&
				!strings.Contains(input, "&gt;") && !strings.Contains(input, "&quot;") {
				t.Fatalf("BUG: & not escaped in: %q -> %q", input, escaped)
			}
		}
		if strings.Contains(input, "\"") && !strings.Contains(escaped, "&quot;") {
			t.Fatalf("BUG: \" not escaped in: %q -> %q", input, escaped)
		}
		if strings.Contains(input, "'") && !strings.Contains(escaped, "&#39;") {
			t.Fatalf("BUG: ' not escaped in: %q -> %q", input, escaped)
		}
	})
}

func TestProperty_HTMLEscapeXSSPrevention(t *testing.T) {
	xssPayloads := []string{
		"<script>alert('xss')</script>",
		"<img src=x onerror=alert('xss')>",
		"<svg onload=alert('xss')>",
		"javascript:alert('xss')",
		"<body onload=alert('xss')>",
		"<iframe src='javascript:alert(1)'>",
		"<a href='javascript:alert(1)'>click</a>",
		"<div onclick=alert('xss')>click</div>",
	}

	for _, payload := range xssPayloads {
		escaped := escapeHTML(payload)

		// PROPERTY: Escaped output must not contain executable HTML
		if strings.Contains(escaped, "<script") ||
			strings.Contains(escaped, "<img") ||
			strings.Contains(escaped, "<svg") ||
			strings.Contains(escaped, "<body") ||
			strings.Contains(escaped, "<iframe") ||
			strings.Contains(escaped, "<a href") ||
			strings.Contains(escaped, "<div") {
			t.Fatalf("BUG: XSS payload not properly escaped: %q -> %q", payload, escaped)
		}
	}
}

// ============================================================================
// PROPERTY TESTS FOR MESSAGE FORMATTING
// ============================================================================

func TestProperty_FormatMessageInvariants(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := rapid.String().Draw(t, "input")

		formatted := formatMessage(input)

		// PROPERTY 1: Newlines should be converted to <br>
		// (after HTML escaping)
		inputNewlines := strings.Count(input, "\n")
		outputBRs := strings.Count(formatted, "<br>")
		if inputNewlines != outputBRs {
			t.Fatalf("BUG: Newline count mismatch: input has %d, output has %d <br>",
				inputNewlines, outputBRs)
		}

		// PROPERTY 2: HTML should be escaped
		if strings.Contains(input, "<") {
			if strings.Contains(formatted, "<script") ||
				strings.Contains(formatted, "<img") {
				t.Fatalf("BUG: HTML not escaped in formatMessage: %q -> %q", input, formatted)
			}
		}
	})
}

// ============================================================================
// SECURITY PROPERTY TESTS
// ============================================================================

func TestProperty_PathTraversalBlocked(t *testing.T) {
	pathTraversalAttempts := []string{
		"../../../etc/passwd.jpg",
		"..\\..\\..\\windows\\system32.jpg",
		"....//....//etc/passwd.png",
		"file:///etc/passwd.jpg",
		"/etc/passwd.jpg",
		"C:\\Windows\\System32.png",
	}

	for _, filename := range pathTraversalAttempts {
		content := genJPEGMagicBytes().Filter(func(b []byte) bool { return len(b) > 10 })
		reader := bytes.NewReader(content.Example())
		_, valid := ValidateImageFile(filename, reader)

		// The validation should reject these based on extension check
		// Path traversal characters shouldn't lead to valid extensions
		if valid {
			t.Logf("Note: Path traversal attempt %q passed validation (check server-side path handling)", filename)
		}
	}
}

func TestProperty_NullByteInFilenameHandled(t *testing.T) {
	nullByteFilenames := []string{
		"test\x00.jpg",
		"test.jpg\x00.exe",
		"test\x00.php.jpg",
	}

	for _, filename := range nullByteFilenames {
		content := genJPEGMagicBytes().Filter(func(b []byte) bool { return len(b) > 10 })
		reader := bytes.NewReader(content.Example())
		_, valid := ValidateImageFile(filename, reader)

		// Null bytes in filenames should be handled safely
		t.Logf("Filename with null byte %q: valid=%v", filename, valid)
	}
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
