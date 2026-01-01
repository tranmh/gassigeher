package services

// EmailProvider defines the interface for email sending across different providers
// Supports Gmail API, SMTP (Strato, Office365, etc.)
type EmailProvider interface {
	// SendEmail sends an email with HTML body
	// Automatically includes BCC if configured in the provider
	SendEmail(to, subject, body string) error

	// ValidateConfig validates the provider configuration
	ValidateConfig() error

	// Close closes any open connections
	Close() error

	// GetFromEmail returns the from email address
	GetFromEmail() string
}

// EmailConfig holds configuration for all email providers
type EmailConfig struct {
	// Provider selection
	Provider string // "gmail" or "smtp"

	// Gmail API settings
	GmailClientID     string
	GmailClientSecret string
	GmailRefreshToken string
	GmailFromEmail    string

	// SMTP settings
	SMTPHost      string
	SMTPPort      int
	SMTPUsername  string
	SMTPPassword  string
	SMTPFromEmail string
	SMTPUseTLS    bool // Use STARTTLS (port 587)
	SMTPUseSSL    bool // Use SSL/TLS (port 465)

	// BCC settings (applies to all providers)
	// Optional: BCC all emails to this address for audit trail
	// Leave empty to disable
	BCCAdmin string

	// Base URL for email links (e.g., "https://gassigeher.com")
	BaseURL string

	// Base domain for multi-tenant (e.g., "gassigeher.org" or "gassigeher.local")
	BaseDomain string
}
