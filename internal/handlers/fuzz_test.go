package handlers

import (
	"bytes"
	"strings"
	"testing"
)

// FuzzIsValidSlug tests the slug validation function
func FuzzIsValidSlug(f *testing.F) {
	// Seed corpus
	f.Add("valid-slug")
	f.Add("a")
	f.Add("ab")
	f.Add("abc")
	f.Add("")
	f.Add("UPPERCASE")
	f.Add("MixedCase")
	f.Add("with spaces")
	f.Add("with_underscore")
	f.Add("with.dot")
	f.Add("123numeric")
	f.Add("-starts-with-dash")
	f.Add("ends-with-dash-")
	f.Add("double--dash")
	f.Add(strings.Repeat("a", 101)) // Over max length
	f.Add(strings.Repeat("a", 100)) // At max length
	f.Add("ü-unicode")
	f.Add("emoji-🐕")
	f.Add("\x00null")
	f.Add("\n\r\t")
	f.Add("a-" + strings.Repeat("b", 1000)) // Very long

	f.Fuzz(func(t *testing.T, slug string) {
		// Should not panic
		result := isValidSlug(slug)

		// Verify invariants
		if result {
			// Valid slugs must be 3-100 chars
			if len(slug) < 3 || len(slug) > 100 {
				t.Errorf("isValidSlug returned true for slug with length %d: %q", len(slug), slug)
			}
			// Must be lowercase
			if slug != strings.ToLower(slug) {
				t.Errorf("isValidSlug returned true for non-lowercase slug: %q", slug)
			}
			// Must start with letter
			if len(slug) > 0 && (slug[0] < 'a' || slug[0] > 'z') {
				t.Errorf("isValidSlug returned true for slug not starting with letter: %q", slug)
			}
		}
	})
}

// FuzzIsReservedSlug tests reserved slug detection
func FuzzIsReservedSlug(f *testing.F) {
	// Seed corpus with known reserved slugs
	f.Add("admin")
	f.Add("api")
	f.Add("www")
	f.Add("mail")
	f.Add("demo")
	f.Add("test")
	f.Add("app")
	f.Add("dashboard")
	f.Add("login")
	f.Add("register")
	// Non-reserved
	f.Add("mycompany")
	f.Add("shelter123")
	f.Add("")
	f.Add("ADMIN") // Uppercase version
	f.Add(" admin")

	f.Fuzz(func(t *testing.T, slug string) {
		// Should not panic
		_ = isReservedSlug(slug)
	})
}

// FuzzValidateImageFile tests image file validation
func FuzzValidateImageFile(f *testing.F) {
	// Seed corpus with various file types
	// JPEG magic bytes
	f.Add("test.jpg", []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46})
	// PNG magic bytes
	f.Add("test.png", []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
	// GIF (not allowed)
	f.Add("test.gif", []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61})
	// Text file with .jpg extension
	f.Add("fake.jpg", []byte("This is not an image"))
	// HTML file
	f.Add("test.html", []byte("<html><body>test</body></html>"))
	// PHP file with image extension
	f.Add("shell.jpg.php", []byte("<?php echo 'pwned'; ?>"))
	// Empty file
	f.Add("empty.jpg", []byte{})
	// Very long filename
	f.Add(strings.Repeat("a", 1000)+".jpg", []byte{0xFF, 0xD8, 0xFF})
	// Null bytes in filename
	f.Add("test\x00.jpg", []byte{0xFF, 0xD8, 0xFF})
	// Special characters
	f.Add("test<script>.jpg", []byte{0xFF, 0xD8, 0xFF})
	f.Add("../../etc/passwd.jpg", []byte{0xFF, 0xD8, 0xFF})

	f.Fuzz(func(t *testing.T, filename string, content []byte) {
		reader := bytes.NewReader(content)

		// Should not panic
		_, valid := ValidateImageFile(filename, reader)

		// If valid, verify some basic invariants
		if valid {
			// Should have valid extension
			lowerName := strings.ToLower(filename)
			if !strings.HasSuffix(lowerName, ".jpg") &&
				!strings.HasSuffix(lowerName, ".jpeg") &&
				!strings.HasSuffix(lowerName, ".png") {
				t.Errorf("ValidateImageFile accepted file with invalid extension: %q", filename)
			}
		}
	})
}

// FuzzValidateImageMIMEType tests MIME type detection
func FuzzValidateImageMIMEType(f *testing.F) {
	// JPEG
	f.Add([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00})
	// PNG
	f.Add([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
	// GIF
	f.Add([]byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61})
	// BMP
	f.Add([]byte{0x42, 0x4D})
	// PDF
	f.Add([]byte{0x25, 0x50, 0x44, 0x46})
	// ZIP
	f.Add([]byte{0x50, 0x4B, 0x03, 0x04})
	// EXE
	f.Add([]byte{0x4D, 0x5A})
	// Text
	f.Add([]byte("Hello, World!"))
	// HTML
	f.Add([]byte("<html>"))
	// Empty
	f.Add([]byte{})
	// Random bytes
	f.Add([]byte{0x00, 0x01, 0x02, 0x03, 0x04})
	// Truncated JPEG
	f.Add([]byte{0xFF, 0xD8})

	f.Fuzz(func(t *testing.T, content []byte) {
		reader := bytes.NewReader(content)

		// Should not panic
		// Note: ValidateImageMIMEType returns (errorMessage, valid), not (mimeType, valid)
		errMsg, valid := ValidateImageMIMEType(reader)

		// If valid, error message should be empty
		if valid && errMsg != "" {
			t.Errorf("ValidateImageMIMEType returned valid=true but non-empty error: %s", errMsg)
		}

		// If invalid, error message should be non-empty
		if !valid && errMsg == "" {
			t.Errorf("ValidateImageMIMEType returned valid=false but empty error message")
		}
	})
}

// FuzzContactRequestValidation tests contact form validation
func FuzzContactRequestValidation(f *testing.F) {
	// Seed corpus
	f.Add("John Doe", "john@example.com", "Hello, this is a message")
	f.Add("", "john@example.com", "Message")
	f.Add("John", "", "Message")
	f.Add("John", "john@example.com", "")
	f.Add("John", "invalid-email", "Message")
	f.Add("John", "john@", "Message")
	f.Add("John", "@example.com", "Message")
	f.Add(strings.Repeat("a", 1000), "john@example.com", "Message")
	f.Add("John", strings.Repeat("a", 100)+"@example.com", "Message")
	f.Add("John", "john@example.com", strings.Repeat("a", 10000))
	// Header injection attempts
	f.Add("John\r\nBcc: attacker@evil.com", "john@example.com", "Message")
	f.Add("John", "john@example.com\r\nBcc: attacker@evil.com", "Message")
	f.Add("John", "john@example.com", "Message\r\nBcc: attacker@evil.com")

	f.Fuzz(func(t *testing.T, name, email, message string) {
		req := &ContactRequest{
			Name:    name,
			Email:   email,
			Message: message,
		}

		// Should not panic
		err := req.Validate()

		// If valid, verify header injection is blocked
		if err == nil {
			if strings.Contains(name, "\r") || strings.Contains(name, "\n") {
				t.Errorf("Validate accepted name with CRLF: %q", name)
			}
			if strings.Contains(email, "\r") || strings.Contains(email, "\n") {
				t.Errorf("Validate accepted email with CRLF: %q", email)
			}
		}
	})
}

// FuzzIsValidEmail tests email validation
func FuzzIsValidEmail(f *testing.F) {
	// Seed corpus
	f.Add("user@example.com")
	f.Add("user.name@example.com")
	f.Add("user+tag@example.com")
	f.Add("user@subdomain.example.com")
	f.Add("")
	f.Add("@")
	f.Add("user@")
	f.Add("@example.com")
	f.Add("user")
	f.Add("user@.com")
	f.Add("user@example")
	f.Add("user@example.")
	f.Add("user@@example.com")
	f.Add("user @example.com")
	f.Add("user@ example.com")
	f.Add("<script>@example.com")
	f.Add("user@<script>.com")
	f.Add(strings.Repeat("a", 100) + "@example.com")
	f.Add("user@" + strings.Repeat("a", 100) + ".com")
	// Header injection
	f.Add("user\r\n@example.com")
	f.Add("user@example.com\r\nBcc: evil@attacker.com")

	f.Fuzz(func(t *testing.T, email string) {
		// Should not panic
		result := isValidEmail(email)

		// If valid, verify no CRLF
		if result {
			if strings.Contains(email, "\r") || strings.Contains(email, "\n") {
				t.Errorf("isValidEmail accepted email with CRLF: %q", email)
			}
		}
	})
}
