package services

import (
	"strings"
	"testing"

	"github.com/tranmh/gassigeher/internal/config"
)

// TestValidateEmailConfig_Gmail tests Gmail configuration validation
func TestValidateEmailConfig_Gmail(t *testing.T) {
	t.Run("valid gmail config", func(t *testing.T) {
		cfg := &EmailConfig{
			Provider:          "gmail",
			GmailClientID:     "client-id",
			GmailClientSecret: "client-secret",
			GmailRefreshToken: "refresh-token",
			GmailFromEmail:    "sender@gmail.com",
		}
		err := ValidateEmailConfig(cfg)
		if err != nil {
			t.Errorf("Expected no error for valid gmail config, got: %v", err)
		}
	})

	t.Run("gmail missing client id", func(t *testing.T) {
		cfg := &EmailConfig{
			Provider:          "gmail",
			GmailClientSecret: "client-secret",
			GmailRefreshToken: "refresh-token",
			GmailFromEmail:    "sender@gmail.com",
		}
		err := ValidateEmailConfig(cfg)
		if err == nil {
			t.Error("Expected error for missing client id")
		}
		if !strings.Contains(err.Error(), "GMAIL_CLIENT_ID") {
			t.Errorf("Error should mention GMAIL_CLIENT_ID: %v", err)
		}
	})

	t.Run("gmail missing client secret", func(t *testing.T) {
		cfg := &EmailConfig{
			Provider:          "gmail",
			GmailClientID:     "client-id",
			GmailRefreshToken: "refresh-token",
			GmailFromEmail:    "sender@gmail.com",
		}
		err := ValidateEmailConfig(cfg)
		if err == nil {
			t.Error("Expected error for missing client secret")
		}
		if !strings.Contains(err.Error(), "GMAIL_CLIENT_SECRET") {
			t.Errorf("Error should mention GMAIL_CLIENT_SECRET: %v", err)
		}
	})

	t.Run("gmail missing refresh token", func(t *testing.T) {
		cfg := &EmailConfig{
			Provider:          "gmail",
			GmailClientID:     "client-id",
			GmailClientSecret: "client-secret",
			GmailFromEmail:    "sender@gmail.com",
		}
		err := ValidateEmailConfig(cfg)
		if err == nil {
			t.Error("Expected error for missing refresh token")
		}
		if !strings.Contains(err.Error(), "GMAIL_REFRESH_TOKEN") {
			t.Errorf("Error should mention GMAIL_REFRESH_TOKEN: %v", err)
		}
	})

	t.Run("gmail missing from email", func(t *testing.T) {
		cfg := &EmailConfig{
			Provider:          "gmail",
			GmailClientID:     "client-id",
			GmailClientSecret: "client-secret",
			GmailRefreshToken: "refresh-token",
		}
		err := ValidateEmailConfig(cfg)
		if err == nil {
			t.Error("Expected error for missing from email")
		}
		if !strings.Contains(err.Error(), "GMAIL_FROM_EMAIL") {
			t.Errorf("Error should mention GMAIL_FROM_EMAIL: %v", err)
		}
	})

	t.Run("empty provider defaults to gmail", func(t *testing.T) {
		cfg := &EmailConfig{
			Provider:          "",
			GmailClientID:     "client-id",
			GmailClientSecret: "client-secret",
			GmailRefreshToken: "refresh-token",
			GmailFromEmail:    "sender@gmail.com",
		}
		err := ValidateEmailConfig(cfg)
		if err != nil {
			t.Errorf("Expected no error when provider empty (defaults to gmail), got: %v", err)
		}
	})
}

// TestValidateEmailConfig_SMTP tests SMTP configuration validation
func TestValidateEmailConfig_SMTP(t *testing.T) {
	t.Run("valid smtp config", func(t *testing.T) {
		cfg := &EmailConfig{
			Provider:      "smtp",
			SMTPHost:      "smtp.example.com",
			SMTPPort:      587,
			SMTPUsername:  "user",
			SMTPPassword:  "pass",
			SMTPFromEmail: "sender@example.com",
			SMTPUseTLS:    true,
		}
		err := ValidateEmailConfig(cfg)
		if err != nil {
			t.Errorf("Expected no error for valid smtp config, got: %v", err)
		}
	})

	t.Run("smtp without auth is valid", func(t *testing.T) {
		cfg := &EmailConfig{
			Provider:      "smtp",
			SMTPHost:      "smtp.example.com",
			SMTPPort:      25,
			SMTPFromEmail: "sender@example.com",
		}
		err := ValidateEmailConfig(cfg)
		if err != nil {
			t.Errorf("Expected no error for smtp without auth, got: %v", err)
		}
	})

	t.Run("smtp missing host", func(t *testing.T) {
		cfg := &EmailConfig{
			Provider:      "smtp",
			SMTPPort:      587,
			SMTPFromEmail: "sender@example.com",
		}
		err := ValidateEmailConfig(cfg)
		if err == nil {
			t.Error("Expected error for missing host")
		}
		if !strings.Contains(err.Error(), "SMTP_HOST") {
			t.Errorf("Error should mention SMTP_HOST: %v", err)
		}
	})

	t.Run("smtp missing port", func(t *testing.T) {
		cfg := &EmailConfig{
			Provider:      "smtp",
			SMTPHost:      "smtp.example.com",
			SMTPFromEmail: "sender@example.com",
		}
		err := ValidateEmailConfig(cfg)
		if err == nil {
			t.Error("Expected error for missing port")
		}
		if !strings.Contains(err.Error(), "SMTP_PORT") {
			t.Errorf("Error should mention SMTP_PORT: %v", err)
		}
	})

	t.Run("smtp invalid port range", func(t *testing.T) {
		testCases := []struct {
			name string
			port int
		}{
			{"negative port", -1},
			{"zero port", 0},
			{"port too high", 70000},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cfg := &EmailConfig{
					Provider:      "smtp",
					SMTPHost:      "smtp.example.com",
					SMTPPort:      tc.port,
					SMTPFromEmail: "sender@example.com",
				}
				err := ValidateEmailConfig(cfg)
				if err == nil {
					t.Errorf("Expected error for port %d", tc.port)
				}
			})
		}
	})

	t.Run("smtp username without password", func(t *testing.T) {
		cfg := &EmailConfig{
			Provider:      "smtp",
			SMTPHost:      "smtp.example.com",
			SMTPPort:      587,
			SMTPUsername:  "user",
			SMTPFromEmail: "sender@example.com",
		}
		err := ValidateEmailConfig(cfg)
		if err == nil {
			t.Error("Expected error when username provided without password")
		}
		if !strings.Contains(err.Error(), "together") {
			t.Errorf("Error should mention both must be provided together: %v", err)
		}
	})

	t.Run("smtp password without username", func(t *testing.T) {
		cfg := &EmailConfig{
			Provider:      "smtp",
			SMTPHost:      "smtp.example.com",
			SMTPPort:      587,
			SMTPPassword:  "pass",
			SMTPFromEmail: "sender@example.com",
		}
		err := ValidateEmailConfig(cfg)
		if err == nil {
			t.Error("Expected error when password provided without username")
		}
	})

	t.Run("smtp missing from email", func(t *testing.T) {
		cfg := &EmailConfig{
			Provider: "smtp",
			SMTPHost: "smtp.example.com",
			SMTPPort: 587,
		}
		err := ValidateEmailConfig(cfg)
		if err == nil {
			t.Error("Expected error for missing from email")
		}
		if !strings.Contains(err.Error(), "SMTP_FROM_EMAIL") {
			t.Errorf("Error should mention SMTP_FROM_EMAIL: %v", err)
		}
	})

	t.Run("smtp both tls and ssl is invalid", func(t *testing.T) {
		cfg := &EmailConfig{
			Provider:      "smtp",
			SMTPHost:      "smtp.example.com",
			SMTPPort:      587,
			SMTPFromEmail: "sender@example.com",
			SMTPUseTLS:    true,
			SMTPUseSSL:    true,
		}
		err := ValidateEmailConfig(cfg)
		if err == nil {
			t.Error("Expected error when both TLS and SSL enabled")
		}
		if !strings.Contains(err.Error(), "cannot use both") {
			t.Errorf("Error should mention cannot use both: %v", err)
		}
	})
}

// TestValidateEmailConfig_UnsupportedProvider tests unsupported provider handling
func TestValidateEmailConfig_UnsupportedProvider(t *testing.T) {
	t.Run("unsupported provider", func(t *testing.T) {
		cfg := &EmailConfig{
			Provider: "mailgun",
		}
		err := ValidateEmailConfig(cfg)
		if err == nil {
			t.Error("Expected error for unsupported provider")
		}
		if !strings.Contains(err.Error(), "unsupported") {
			t.Errorf("Error should mention unsupported: %v", err)
		}
	})

	t.Run("nil config", func(t *testing.T) {
		err := ValidateEmailConfig(nil)
		if err == nil {
			t.Error("Expected error for nil config")
		}
		if !strings.Contains(err.Error(), "cannot be nil") {
			t.Errorf("Error should mention cannot be nil: %v", err)
		}
	})
}

// TestNewEmailProvider tests the email provider factory
func TestNewEmailProvider(t *testing.T) {
	t.Run("nil config returns error", func(t *testing.T) {
		_, err := NewEmailProvider(nil)
		if err == nil {
			t.Error("Expected error for nil config")
		}
		if !strings.Contains(err.Error(), "cannot be nil") {
			t.Errorf("Error should mention cannot be nil: %v", err)
		}
	})

	t.Run("unsupported provider returns error", func(t *testing.T) {
		cfg := &EmailConfig{
			Provider: "sendgrid",
		}
		_, err := NewEmailProvider(cfg)
		if err == nil {
			t.Error("Expected error for unsupported provider")
		}
		if !strings.Contains(err.Error(), "unsupported") {
			t.Errorf("Error should mention unsupported: %v", err)
		}
	})

	t.Run("smtp provider returns smtp provider", func(t *testing.T) {
		cfg := &EmailConfig{
			Provider:      "smtp",
			SMTPHost:      "smtp.example.com",
			SMTPPort:      587,
			SMTPFromEmail: "sender@example.com",
			SMTPUseTLS:    true,
		}
		provider, err := NewEmailProvider(cfg)
		if err != nil {
			t.Errorf("Expected no error for valid smtp config, got: %v", err)
		}
		if provider == nil {
			t.Error("Expected provider to be non-nil")
		}
		// Verify it's an SMTP provider by checking GetFromEmail
		if provider.GetFromEmail() != "sender@example.com" {
			t.Errorf("Expected from email 'sender@example.com', got '%s'", provider.GetFromEmail())
		}
	})

	t.Run("provider name is case insensitive", func(t *testing.T) {
		testCases := []string{"SMTP", "Smtp", "sMtP", " smtp ", "GMAIL", "Gmail"}
		for _, name := range testCases {
			cfg := &EmailConfig{
				Provider: name,
			}
			// For SMTP variants, add required config
			if strings.ToLower(strings.TrimSpace(name)) == "smtp" {
				cfg.SMTPHost = "smtp.example.com"
				cfg.SMTPPort = 587
				cfg.SMTPFromEmail = "sender@example.com"
			}
			// We expect error for gmail (missing creds) but not "unsupported" error
			_, err := NewEmailProvider(cfg)
			if err != nil && strings.Contains(err.Error(), "unsupported") {
				t.Errorf("Provider %q should be recognized (case insensitive)", name)
			}
		}
	})
}

// TestConfigToEmailConfig tests the config conversion helper
func TestConfigToEmailConfig(t *testing.T) {
	t.Run("converts all fields", func(t *testing.T) {
		cfg := &config.Config{
			EmailProvider:     "smtp",
			GmailClientID:     "client-id",
			GmailClientSecret: "client-secret",
			GmailRefreshToken: "refresh-token",
			GmailFromEmail:    "gmail@example.com",
			SMTPHost:          "smtp.example.com",
			SMTPPort:          587,
			SMTPUsername:      "smtp-user",
			SMTPPassword:      "smtp-pass",
			SMTPFromEmail:     "smtp@example.com",
			SMTPUseTLS:        true,
			SMTPUseSSL:        false,
			EmailBCCAdmin:     "admin@example.com",
			BaseURL:           "https://example.com",
		}

		emailCfg := ConfigToEmailConfig(cfg)

		if emailCfg.Provider != "smtp" {
			t.Errorf("Expected provider 'smtp', got '%s'", emailCfg.Provider)
		}
		if emailCfg.GmailClientID != "client-id" {
			t.Errorf("Expected GmailClientID 'client-id', got '%s'", emailCfg.GmailClientID)
		}
		if emailCfg.GmailClientSecret != "client-secret" {
			t.Errorf("Expected GmailClientSecret 'client-secret', got '%s'", emailCfg.GmailClientSecret)
		}
		if emailCfg.GmailRefreshToken != "refresh-token" {
			t.Errorf("Expected GmailRefreshToken 'refresh-token', got '%s'", emailCfg.GmailRefreshToken)
		}
		if emailCfg.GmailFromEmail != "gmail@example.com" {
			t.Errorf("Expected GmailFromEmail 'gmail@example.com', got '%s'", emailCfg.GmailFromEmail)
		}
		if emailCfg.SMTPHost != "smtp.example.com" {
			t.Errorf("Expected SMTPHost 'smtp.example.com', got '%s'", emailCfg.SMTPHost)
		}
		if emailCfg.SMTPPort != 587 {
			t.Errorf("Expected SMTPPort 587, got %d", emailCfg.SMTPPort)
		}
		if emailCfg.SMTPUsername != "smtp-user" {
			t.Errorf("Expected SMTPUsername 'smtp-user', got '%s'", emailCfg.SMTPUsername)
		}
		if emailCfg.SMTPPassword != "smtp-pass" {
			t.Errorf("Expected SMTPPassword 'smtp-pass', got '%s'", emailCfg.SMTPPassword)
		}
		if emailCfg.SMTPFromEmail != "smtp@example.com" {
			t.Errorf("Expected SMTPFromEmail 'smtp@example.com', got '%s'", emailCfg.SMTPFromEmail)
		}
		if !emailCfg.SMTPUseTLS {
			t.Error("Expected SMTPUseTLS to be true")
		}
		if emailCfg.SMTPUseSSL {
			t.Error("Expected SMTPUseSSL to be false")
		}
		if emailCfg.BCCAdmin != "admin@example.com" {
			t.Errorf("Expected BCCAdmin 'admin@example.com', got '%s'", emailCfg.BCCAdmin)
		}
		if emailCfg.BaseURL != "https://example.com" {
			t.Errorf("Expected BaseURL 'https://example.com', got '%s'", emailCfg.BaseURL)
		}
	})

	t.Run("handles empty config", func(t *testing.T) {
		cfg := &config.Config{}
		emailCfg := ConfigToEmailConfig(cfg)
		if emailCfg == nil {
			t.Error("Expected non-nil email config")
		}
		// Default values should be empty strings and zero
		if emailCfg.Provider != "" {
			t.Errorf("Expected empty provider, got '%s'", emailCfg.Provider)
		}
		if emailCfg.SMTPPort != 0 {
			t.Errorf("Expected zero port, got %d", emailCfg.SMTPPort)
		}
	})
}
