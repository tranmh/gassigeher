package services

import (
	"fmt"
	"log"
	"strings"

	"github.com/tranmh/gassigeher/internal/config"
)

// NewEmailProvider creates an email provider based on configuration
func NewEmailProvider(config *EmailConfig) (EmailProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("email config cannot be nil")
	}

	// Normalize provider name
	provider := strings.ToLower(strings.TrimSpace(config.Provider))

	// Default to Gmail if not specified (backward compatibility)
	if provider == "" {
		provider = "gmail"
	}

	switch provider {
	case "gmail":
		return NewGmailProvider(config)

	case "smtp":
		return NewSMTPProvider(config)

	default:
		return nil, fmt.Errorf("unsupported email provider: %s (supported: gmail, smtp)", config.Provider)
	}
}

// ValidateEmailConfig validates email configuration before creating provider
func ValidateEmailConfig(config *EmailConfig) error {
	if config == nil {
		return fmt.Errorf("email config cannot be nil")
	}

	provider := strings.ToLower(strings.TrimSpace(config.Provider))
	if provider == "" {
		provider = "gmail" // Default
	}

	switch provider {
	case "gmail":
		return validateGmailConfig(config)
	case "smtp":
		return validateSMTPConfig(config)
	default:
		return fmt.Errorf("unsupported email provider: %s", config.Provider)
	}
}

// validateGmailConfig validates Gmail-specific configuration
func validateGmailConfig(config *EmailConfig) error {
	if config.GmailClientID == "" {
		return fmt.Errorf("GMAIL_CLIENT_ID is required for Gmail provider")
	}
	if config.GmailClientSecret == "" {
		return fmt.Errorf("GMAIL_CLIENT_SECRET is required for Gmail provider")
	}
	if config.GmailRefreshToken == "" {
		return fmt.Errorf("GMAIL_REFRESH_TOKEN is required for Gmail provider")
	}
	if config.GmailFromEmail == "" {
		return fmt.Errorf("GMAIL_FROM_EMAIL is required for Gmail provider")
	}
	return nil
}

// validateSMTPConfig validates SMTP-specific configuration
func validateSMTPConfig(config *EmailConfig) error {
	if config.SMTPHost == "" {
		return fmt.Errorf("SMTP_HOST is required for SMTP provider")
	}
	if config.SMTPPort == 0 {
		return fmt.Errorf("SMTP_PORT is required for SMTP provider")
	}
	if config.SMTPPort < 1 || config.SMTPPort > 65535 {
		return fmt.Errorf("SMTP_PORT must be between 1 and 65535")
	}
	// Username and password are optional (some SMTP servers don't require auth)
	// But if one is provided, both should be provided
	if (config.SMTPUsername != "" && config.SMTPPassword == "") || (config.SMTPUsername == "" && config.SMTPPassword != "") {
		return fmt.Errorf("both SMTP_USERNAME and SMTP_PASSWORD must be provided together")
	}
	if config.SMTPFromEmail == "" {
		return fmt.Errorf("SMTP_FROM_EMAIL is required for SMTP provider")
	}

	// Validate TLS/SSL configuration
	if config.SMTPUseTLS && config.SMTPUseSSL {
		return fmt.Errorf("cannot use both SMTP_USE_TLS and SMTP_USE_SSL (choose one)")
	}

	// Recommend TLS/SSL based on port
	if config.SMTPPort == 465 && !config.SMTPUseSSL {
		log.Printf("Warning: Port 465 typically requires SMTP_USE_SSL=true")
	}
	if config.SMTPPort == 587 && !config.SMTPUseTLS {
		log.Printf("Warning: Port 587 typically requires SMTP_USE_TLS=true")
	}

	return nil
}

// ConfigToEmailConfig converts application config to email config
// This helper avoids circular dependency between config and services packages
func ConfigToEmailConfig(cfg *config.Config) *EmailConfig {
	return &EmailConfig{
		Provider:          cfg.EmailProvider,
		GmailClientID:     cfg.GmailClientID,
		GmailClientSecret: cfg.GmailClientSecret,
		GmailRefreshToken: cfg.GmailRefreshToken,
		GmailFromEmail:    cfg.GmailFromEmail,
		SMTPHost:          cfg.SMTPHost,
		SMTPPort:          cfg.SMTPPort,
		SMTPUsername:      cfg.SMTPUsername,
		SMTPPassword:      cfg.SMTPPassword,
		SMTPFromEmail:     cfg.SMTPFromEmail,
		SMTPUseTLS:        cfg.SMTPUseTLS,
		SMTPUseSSL:        cfg.SMTPUseSSL,
		BCCAdmin:          cfg.EmailBCCAdmin,
		BaseURL:           cfg.BaseURL,
	}
}
