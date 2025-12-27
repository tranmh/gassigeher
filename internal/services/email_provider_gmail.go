package services

import (
	"encoding/base64"
	"fmt"
	"log"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// GmailProvider implements EmailProvider using Gmail API
type GmailProvider struct {
	service   *gmail.Service
	fromEmail string
	bccAdmin  string // Optional: BCC all emails to this address
}

// NewGmailProvider creates a new Gmail email provider
func NewGmailProvider(config *EmailConfig) (EmailProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("email config cannot be nil")
	}

	// Validate required fields
	if config.GmailClientID == "" {
		return nil, fmt.Errorf("Gmail client ID is required")
	}
	if config.GmailClientSecret == "" {
		return nil, fmt.Errorf("Gmail client secret is required")
	}
	if config.GmailRefreshToken == "" {
		return nil, fmt.Errorf("Gmail refresh token is required")
	}
	if config.GmailFromEmail == "" {
		return nil, fmt.Errorf("Gmail from email is required")
	}

	// Setup OAuth2 configuration
	oauthConfig := &oauth2.Config{
		ClientID:     config.GmailClientID,
		ClientSecret: config.GmailClientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{gmail.GmailSendScope},
	}

	token := &oauth2.Token{
		RefreshToken: config.GmailRefreshToken,
		TokenType:    "Bearer",
	}

	client := oauthConfig.Client(oauth2.NoContext, token)

	// Create Gmail service
	service, err := gmail.NewService(oauth2.NoContext, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail service: %w", err)
	}

	return &GmailProvider{
		service:   service,
		fromEmail: config.GmailFromEmail,
		bccAdmin:  config.BCCAdmin,
	}, nil
}

// sanitizeHeader removes CRLF characters to prevent email header injection
func sanitizeHeader(s string) string {
	// Remove carriage return, newline, and null bytes
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\r' && s[i] != '\n' && s[i] != 0 {
			result = append(result, s[i])
		}
	}
	return string(result)
}

// SendEmail sends an email via Gmail API
func (p *GmailProvider) SendEmail(to, subject, body string) error {
	var message gmail.Message

	// Sanitize header values to prevent header injection attacks
	to = sanitizeHeader(to)
	subject = sanitizeHeader(subject)
	fromEmail := sanitizeHeader(p.fromEmail)
	bccAdmin := sanitizeHeader(p.bccAdmin)

	// Build email content with optional BCC
	emailContent := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n", fromEmail, to)

	// Add BCC header if configured
	if bccAdmin != "" {
		emailContent += fmt.Sprintf("Bcc: %s\r\n", bccAdmin)
	}

	emailContent += fmt.Sprintf("Subject: %s\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n\r\n"+
		"%s", subject, body)

	// Encode message
	message.Raw = base64.URLEncoding.EncodeToString([]byte(emailContent))

	// Send via Gmail API
	_, err := p.service.Users.Messages.Send("me", &message).Do()
	if err != nil {
		return fmt.Errorf("failed to send email via Gmail: %w", err)
	}

	// Log success (include BCC info if set)
	if p.bccAdmin != "" {
		log.Printf("Email sent to %s (BCC: %s): %s", to, p.bccAdmin, subject)
	} else {
		log.Printf("Email sent to %s: %s", to, subject)
	}

	return nil
}

// ValidateConfig validates the Gmail provider configuration
func (p *GmailProvider) ValidateConfig() error {
	if p.service == nil {
		return fmt.Errorf("Gmail service not initialized")
	}
	if p.fromEmail == "" {
		return fmt.Errorf("from email is required")
	}
	return nil
}

// Close closes the Gmail provider (no persistent connection to close)
func (p *GmailProvider) Close() error {
	// Gmail API doesn't maintain persistent connections
	// Nothing to close
	return nil
}

// GetFromEmail returns the from email address
func (p *GmailProvider) GetFromEmail() string {
	return p.fromEmail
}
