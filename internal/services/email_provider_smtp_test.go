package services

import (
	"strings"
	"testing"
)

// TestNewSMTPProvider tests SMTP provider creation
func TestNewSMTPProvider(t *testing.T) {
	tests := []struct {
		name        string
		config      *EmailConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid_strato_config",
			config: &EmailConfig{
				SMTPHost:      "smtp.strato.de",
				SMTPPort:      465,
				SMTPUsername:  "test@example.com",
				SMTPPassword:  "password",
				SMTPFromEmail: "test@example.com",
				SMTPUseSSL:    true,
				SMTPUseTLS:    false,
			},
			expectError: false,
		},
		{
			name: "valid_office365_config",
			config: &EmailConfig{
				SMTPHost:      "smtp.office365.com",
				SMTPPort:      587,
				SMTPUsername:  "test@example.com",
				SMTPPassword:  "password",
				SMTPFromEmail: "test@example.com",
				SMTPUseTLS:    true,
				SMTPUseSSL:    false,
			},
			expectError: false,
		},
		{
			name: "valid_config_no_auth",
			config: &EmailConfig{
				SMTPHost:      "smtp.example.com",
				SMTPPort:      25,
				SMTPFromEmail: "test@example.com",
				SMTPUseTLS:    false,
				SMTPUseSSL:    false,
			},
			expectError: false,
		},
		{
			name: "valid_config_with_bcc",
			config: &EmailConfig{
				SMTPHost:      "smtp.example.com",
				SMTPPort:      587,
				SMTPUsername:  "test@example.com",
				SMTPPassword:  "password",
				SMTPFromEmail: "test@example.com",
				BCCAdmin:      "admin@example.com",
				SMTPUseTLS:    true,
			},
			expectError: false,
		},
		{
			name:        "nil_config",
			config:      nil,
			expectError: true,
			errorMsg:    "config cannot be nil",
		},
		{
			name: "missing_host",
			config: &EmailConfig{
				SMTPPort:      587,
				SMTPFromEmail: "test@example.com",
			},
			expectError: true,
			errorMsg:    "SMTP_HOST is required",
		},
		{
			name: "missing_port",
			config: &EmailConfig{
				SMTPHost:      "smtp.example.com",
				SMTPFromEmail: "test@example.com",
			},
			expectError: true,
			errorMsg:    "SMTP_PORT is required",
		},
		{
			name: "invalid_port_range",
			config: &EmailConfig{
				SMTPHost:      "smtp.example.com",
				SMTPPort:      99999,
				SMTPFromEmail: "test@example.com",
			},
			expectError: true,
			errorMsg:    "must be between 1 and 65535",
		},
		{
			name: "missing_from_email",
			config: &EmailConfig{
				SMTPHost: "smtp.example.com",
				SMTPPort: 587,
			},
			expectError: true,
			errorMsg:    "SMTP_FROM_EMAIL is required",
		},
		{
			name: "invalid_from_email",
			config: &EmailConfig{
				SMTPHost:      "smtp.example.com",
				SMTPPort:      587,
				SMTPFromEmail: "not-an-email",
			},
			expectError: true,
			errorMsg:    "not a valid email address",
		},
		{
			name: "invalid_bcc_email",
			config: &EmailConfig{
				SMTPHost:      "smtp.example.com",
				SMTPPort:      587,
				SMTPFromEmail: "test@example.com",
				BCCAdmin:      "not-an-email",
			},
			expectError: true,
			errorMsg:    "EMAIL_BCC_ADMIN is not a valid email address",
		},
		{
			name: "username_without_password",
			config: &EmailConfig{
				SMTPHost:      "smtp.example.com",
				SMTPPort:      587,
				SMTPUsername:  "test@example.com",
				SMTPFromEmail: "test@example.com",
			},
			expectError: true,
			errorMsg:    "both SMTP_USERNAME and SMTP_PASSWORD must be provided together",
		},
		{
			name: "password_without_username",
			config: &EmailConfig{
				SMTPHost:      "smtp.example.com",
				SMTPPort:      587,
				SMTPPassword:  "password",
				SMTPFromEmail: "test@example.com",
			},
			expectError: true,
			errorMsg:    "both SMTP_USERNAME and SMTP_PASSWORD must be provided together",
		},
		{
			name: "both_tls_and_ssl",
			config: &EmailConfig{
				SMTPHost:      "smtp.example.com",
				SMTPPort:      587,
				SMTPFromEmail: "test@example.com",
				SMTPUseTLS:    true,
				SMTPUseSSL:    true,
			},
			expectError: true,
			errorMsg:    "cannot use both SMTP_USE_TLS and SMTP_USE_SSL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewSMTPProvider(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
					return
				}
				if provider == nil {
					t.Error("Expected provider to be created")
				}
			}
		})
	}
}

// TestSMTPProvider_ValidateConfig tests configuration validation
func TestSMTPProvider_ValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		provider    *SMTPProvider
		expectError bool
	}{
		{
			name: "valid_configuration",
			provider: &SMTPProvider{
				host:      "smtp.example.com",
				port:      587,
				fromEmail: "test@example.com",
				useTLS:    true,
			},
			expectError: false,
		},
		{
			name: "missing_host",
			provider: &SMTPProvider{
				port:      587,
				fromEmail: "test@example.com",
			},
			expectError: true,
		},
		{
			name: "missing_port",
			provider: &SMTPProvider{
				host:      "smtp.example.com",
				fromEmail: "test@example.com",
			},
			expectError: true,
		},
		{
			name: "invalid_port",
			provider: &SMTPProvider{
				host:      "smtp.example.com",
				port:      -1,
				fromEmail: "test@example.com",
			},
			expectError: true,
		},
		{
			name: "missing_from_email",
			provider: &SMTPProvider{
				host: "smtp.example.com",
				port: 587,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.provider.ValidateConfig()
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

// TestSMTPProvider_GetFromEmail tests getting from email
func TestSMTPProvider_GetFromEmail(t *testing.T) {
	provider := &SMTPProvider{
		fromEmail: "test@example.com",
	}

	fromEmail := provider.GetFromEmail()
	if fromEmail != "test@example.com" {
		t.Errorf("Expected from email 'test@example.com', got '%s'", fromEmail)
	}
}

// TestSMTPProvider_Close tests closing provider
func TestSMTPProvider_Close(t *testing.T) {
	provider := &SMTPProvider{
		host:      "smtp.example.com",
		port:      587,
		fromEmail: "test@example.com",
	}

	err := provider.Close()
	if err != nil {
		t.Errorf("Expected no error on close, got: %v", err)
	}
}

// TestBuildMIMEMessage tests MIME message building
func TestBuildMIMEMessage(t *testing.T) {
	tests := []struct {
		name        string
		provider    *SMTPProvider
		to          string
		subject     string
		body        string
		checkHeader string
		checkValue  string
	}{
		{
			name: "basic_message",
			provider: &SMTPProvider{
				fromEmail: "sender@example.com",
			},
			to:          "recipient@example.com",
			subject:     "Test Subject",
			body:        "<html><body>Test</body></html>",
			checkHeader: "Subject:",
			checkValue:  "Test Subject",
		},
		{
			name: "message_with_bcc",
			provider: &SMTPProvider{
				fromEmail: "sender@example.com",
				bccAdmin:  "admin@example.com",
			},
			to:          "recipient@example.com",
			subject:     "Test",
			body:        "<html><body>Test</body></html>",
			checkHeader: "Bcc:",
			checkValue:  "admin@example.com",
		},
		{
			name: "message_with_german_characters",
			provider: &SMTPProvider{
				fromEmail: "sender@example.com",
			},
			to:          "recipient@example.com",
			subject:     "Schöne Grüße",
			body:        "<html><body>Äpfel und Öl</body></html>",
			checkHeader: "Subject:",
			checkValue:  "=?UTF-8?B?", // Base64 encoded UTF-8
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := tt.provider.buildMIMEMessage(tt.to, tt.subject, tt.body)
			messageStr := string(message)

			// Check for required headers
			if !strings.Contains(messageStr, "MIME-Version: 1.0") {
				t.Error("Message missing MIME-Version header")
			}
			if !strings.Contains(messageStr, "Content-Type: text/html; charset=UTF-8") {
				t.Error("Message missing Content-Type header")
			}
			if !strings.Contains(messageStr, "Content-Transfer-Encoding: quoted-printable") {
				t.Error("Message missing Content-Transfer-Encoding header")
			}

			// Check specific header/value
			if tt.checkHeader != "" {
				if !strings.Contains(messageStr, tt.checkHeader) {
					t.Errorf("Message missing header: %s", tt.checkHeader)
				}
				if tt.checkValue != "" && !strings.Contains(messageStr, tt.checkValue) {
					t.Errorf("Message missing value '%s' in header '%s'", tt.checkValue, tt.checkHeader)
				}
			}

			// Check structure (headers, blank line, body)
			if !strings.Contains(messageStr, "\r\n\r\n") {
				t.Error("Message missing blank line between headers and body")
			}
		})
	}
}

// TestEncodeRFC2047 tests RFC 2047 header encoding
func TestEncodeRFC2047(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "ascii_no_encoding",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "german_umlauts",
			input:    "Schöne Grüße",
			expected: "=?UTF-8?B?", // Should start with this for Base64 UTF-8
		},
		{
			name:     "special_characters",
			input:    "Über uns",
			expected: "=?UTF-8?B?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := encodeRFC2047(tt.input)

			if tt.expected == "=?UTF-8?B?" {
				// For encoded strings, just check they start correctly
				if !strings.HasPrefix(result, tt.expected) {
					t.Errorf("Expected encoded string to start with '%s', got '%s'", tt.expected, result)
				}
				if !strings.HasSuffix(result, "?=") {
					t.Errorf("Expected encoded string to end with '?=', got '%s'", result)
				}
			} else {
				// For ASCII strings, expect exact match
				if result != tt.expected {
					t.Errorf("Expected '%s', got '%s'", tt.expected, result)
				}
			}
		})
	}
}

// TestEncodeBase64 tests base64 encoding
func TestEncodeBase64(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple_text",
			input:    "Hello",
			expected: "SGVsbG8=",
		},
		{
			name:     "text_with_padding",
			input:    "Hi",
			expected: "SGk=",
		},
		{
			name:     "single_character",
			input:    "A",
			expected: "QQ==",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := encodeBase64(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestEncodeQuotedPrintable tests quoted-printable encoding
func TestEncodeQuotedPrintable(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:  "ascii_text",
			input: "Hello World",
			contains: []string{
				"Hello", // Should contain the word
				"World", // Should contain the word
				"=",     // Spaces may be encoded as =20
			},
		},
		{
			name:  "german_umlauts",
			input: "Schöne Grüße",
			contains: []string{
				"Sch", // Plain text part
				"=C3", // UTF-8 encoded characters (ö, ü, ß start with C3)
			},
		},
		{
			name:  "special_characters",
			input: "äöüß",
			contains: []string{
				"=C3", // UTF-8 encoded characters start with this
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := encodeQuotedPrintable(tt.input)

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected result to contain '%s', got '%s'", expected, result)
				}
			}
		})
	}
}

// TestSMTPProvider_SendEmail_Validation tests email validation in SendEmail
func TestSMTPProvider_SendEmail_Validation(t *testing.T) {
	// Note: These tests only validate input, they don't actually send emails
	provider := &SMTPProvider{
		host:      "smtp.example.com",
		port:      587,
		fromEmail: "sender@example.com",
		useTLS:    true,
	}

	tests := []struct {
		name        string
		to          string
		subject     string
		body        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "invalid_recipient_email",
			to:          "not-an-email",
			subject:     "Test",
			body:        "Test",
			expectError: true,
			errorMsg:    "invalid recipient email address",
		},
		{
			name:        "empty_recipient",
			to:          "",
			subject:     "Test",
			body:        "Test",
			expectError: true,
			errorMsg:    "invalid recipient email address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This will fail at validation, before trying to connect
			err := provider.SendEmail(tt.to, tt.subject, tt.body)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				// Note: Valid emails will fail with connection error since we're not mocking
				// the SMTP server. This is expected for these unit tests.
				if err == nil {
					t.Error("Expected connection error for unmocked SMTP")
				}
			}
		})
	}
}

// TestSMTPProvider_PortWarnings tests port configuration warnings
func TestSMTPProvider_PortWarnings(t *testing.T) {
	tests := []struct {
		name   string
		config *EmailConfig
	}{
		{
			name: "port_465_without_ssl",
			config: &EmailConfig{
				SMTPHost:      "smtp.example.com",
				SMTPPort:      465,
				SMTPFromEmail: "test@example.com",
				SMTPUseSSL:    false,
			},
		},
		{
			name: "port_587_without_tls",
			config: &EmailConfig{
				SMTPHost:      "smtp.example.com",
				SMTPPort:      587,
				SMTPFromEmail: "test@example.com",
				SMTPUseTLS:    false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// These should create providers but may log warnings
			provider, err := NewSMTPProvider(tt.config)
			if err != nil {
				t.Errorf("Expected provider creation despite warning, got error: %v", err)
			}
			if provider == nil {
				t.Error("Expected provider to be created")
			}
		})
	}
}

// ============================================================================
// BUG FIX TESTS: SMTP Connection Handling
// ============================================================================

// TestSMTPProvider_WriterCloseWithDefer documents the bug fix for writer.Close() handling
// BUG: In sendWithSSL() and sendWithSTARTTLS(), writer.Close() was called manually
// which could be skipped on early returns or panics. Using defer ensures cleanup.
func TestSMTPProvider_WriterCloseWithDefer(t *testing.T) {
	// This test documents the bug fix pattern
	// The fix ensures writer.Close() is called via defer for proper cleanup:
	//
	// BEFORE (buggy):
	//   writer, err := client.Data()
	//   if err != nil { return err }
	//   _, err = writer.Write(message)
	//   if err != nil {
	//       writer.Close()  // Manual close - easy to miss or skip
	//       return err
	//   }
	//   err = writer.Close()
	//   if err != nil { return err }
	//
	// AFTER (fixed):
	//   writer, err := client.Data()
	//   if err != nil { return err }
	//   defer writer.Close()  // Guaranteed cleanup via defer
	//   _, err = writer.Write(message)
	//   if err != nil { return err }

	t.Log("BUG FIX: sendWithSSL() and sendWithSTARTTLS() now use defer writer.Close()")
	t.Log("This ensures the data writer is always closed, even on errors or panics")
}

// TestSMTPProvider_QuitErrorHandling documents the bug fix for client.Quit() handling
// BUG: client.Quit() errors caused the function to fail even though the email was sent
// FIX: Log but don't fail on client.Quit() errors
func TestSMTPProvider_QuitErrorHandling(t *testing.T) {
	// This test documents the bug fix pattern
	// The fix ensures client.Quit() errors are logged but don't fail the operation:
	//
	// BEFORE (buggy):
	//   return client.Quit()  // Error here fails the whole operation
	//
	// AFTER (fixed):
	//   if err := client.Quit(); err != nil {
	//       // Log but don't fail - email was already sent
	//       fmt.Printf("Warning: SMTP QUIT command failed: %v\n", err)
	//   }
	//   return nil

	t.Log("BUG FIX: client.Quit() errors are now logged but don't fail the operation")
	t.Log("The email is successfully sent even if QUIT fails (connection cleanup issue)")
}
