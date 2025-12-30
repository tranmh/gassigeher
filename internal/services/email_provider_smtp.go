package services

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

// SMTPProvider implements EmailProvider interface using standard SMTP
type SMTPProvider struct {
	host      string
	port      int
	username  string
	password  string
	fromEmail string
	bccAdmin  string
	useTLS    bool // STARTTLS (port 587)
	useSSL    bool // Direct SSL/TLS (port 465)
}

// NewSMTPProvider creates a new SMTP email provider
func NewSMTPProvider(config *EmailConfig) (EmailProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	provider := &SMTPProvider{
		host:      config.SMTPHost,
		port:      config.SMTPPort,
		username:  config.SMTPUsername,
		password:  config.SMTPPassword,
		fromEmail: config.SMTPFromEmail,
		bccAdmin:  config.BCCAdmin,
		useTLS:    config.SMTPUseTLS,
		useSSL:    config.SMTPUseSSL,
	}

	// Validate configuration
	if err := provider.ValidateConfig(); err != nil {
		return nil, err
	}

	return provider, nil
}

// SendEmail sends an email via SMTP
func (p *SMTPProvider) SendEmail(to, subject, body string) error {
	// Validate recipient email
	if _, err := mail.ParseAddress(to); err != nil {
		return fmt.Errorf("invalid recipient email address: %v", err)
	}

	// Build recipient list (includes BCC if configured)
	recipients := []string{to}
	if p.bccAdmin != "" {
		recipients = append(recipients, p.bccAdmin)
	}

	// Create MIME message with proper headers
	message := p.buildMIMEMessage(to, subject, body)

	// Send email based on SSL/TLS configuration
	if p.useSSL {
		// Direct SSL/TLS connection (port 465)
		return p.sendWithSSL(recipients, message)
	}

	// STARTTLS connection (port 587) or plain (port 25)
	return p.sendWithTLS(recipients, message)
}

// sendWithSSL sends email using direct SSL/TLS connection (port 465)
func (p *SMTPProvider) sendWithSSL(recipients []string, message []byte) error {
	// Create TLS configuration
	tlsConfig := &tls.Config{
		ServerName: p.host,
		MinVersion: tls.VersionTLS12,
	}

	// Establish TLS connection
	addr := fmt.Sprintf("%s:%d", p.host, p.port)
	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 10 * time.Second},
		"tcp",
		addr,
		tlsConfig,
	)
	if err != nil {
		return fmt.Errorf("failed to establish SSL/TLS connection: %v", err)
	}
	defer conn.Close()

	// Create SMTP client over TLS connection
	client, err := smtp.NewClient(conn, p.host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %v", err)
	}
	defer client.Close()

	// Authenticate
	if p.username != "" && p.password != "" {
		auth := smtp.PlainAuth("", p.username, p.password, p.host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %v", err)
		}
	}

	// Set sender
	if err := client.Mail(p.fromEmail); err != nil {
		return fmt.Errorf("failed to set sender: %v", err)
	}

	// Set recipients (including BCC)
	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %v", recipient, err)
		}
	}

	// Send message data
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to send DATA command: %v", err)
	}
	// BUG FIX: Use defer to ensure writer is always closed, even on errors
	defer writer.Close()

	_, err = writer.Write(message)
	if err != nil {
		return fmt.Errorf("failed to write message: %v", err)
	}

	// BUG FIX: Log but don't fail on client.Quit() errors - email was already sent
	if err := client.Quit(); err != nil {
		fmt.Printf("Warning: SMTP QUIT command failed (SSL): %v\n", err)
	}
	return nil
}

// sendWithTLS sends email using STARTTLS (port 587) or plain connection
func (p *SMTPProvider) sendWithTLS(recipients []string, message []byte) error {
	// Server address
	addr := fmt.Sprintf("%s:%d", p.host, p.port)

	// Create authentication
	var auth smtp.Auth
	if p.username != "" && p.password != "" {
		auth = smtp.PlainAuth("", p.username, p.password, p.host)
	}

	// If TLS is enabled, use custom dialer with STARTTLS
	if p.useTLS {
		return p.sendWithSTARTTLS(addr, auth, recipients, message)
	}

	// Plain SMTP (not recommended, but supported)
	return smtp.SendMail(addr, auth, p.fromEmail, recipients, message)
}

// sendWithSTARTTLS sends email using STARTTLS upgrade
func (p *SMTPProvider) sendWithSTARTTLS(addr string, auth smtp.Auth, recipients []string, message []byte) error {
	// Connect to SMTP server
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %v", err)
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, p.host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %v", err)
	}
	defer client.Close()

	// Send STARTTLS command
	tlsConfig := &tls.Config{
		ServerName: p.host,
		MinVersion: tls.VersionTLS12,
	}

	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("STARTTLS failed: %v", err)
	}

	// Authenticate after STARTTLS
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %v", err)
		}
	}

	// Set sender
	if err := client.Mail(p.fromEmail); err != nil {
		return fmt.Errorf("failed to set sender: %v", err)
	}

	// Set recipients (including BCC)
	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %v", recipient, err)
		}
	}

	// Send message data
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to send DATA command: %v", err)
	}
	// BUG FIX: Use defer to ensure writer is always closed, even on errors
	defer writer.Close()

	_, err = writer.Write(message)
	if err != nil {
		return fmt.Errorf("failed to write message: %v", err)
	}

	// BUG FIX: Log but don't fail on client.Quit() errors - email was already sent
	if err := client.Quit(); err != nil {
		fmt.Printf("Warning: SMTP QUIT command failed (STARTTLS): %v\n", err)
	}
	return nil
}

// buildMIMEMessage creates a properly formatted MIME email message
func (p *SMTPProvider) buildMIMEMessage(to, subject, htmlBody string) []byte {
	// Parse from address to get proper format
	fromAddr, err := mail.ParseAddress(p.fromEmail)
	if err != nil {
		// Fallback to simple format
		fromAddr = &mail.Address{Address: p.fromEmail}
	}

	// Parse to address
	toAddr, err := mail.ParseAddress(to)
	if err != nil {
		toAddr = &mail.Address{Address: to}
	}

	// Build headers
	headers := make(map[string]string)
	headers["From"] = fromAddr.String()
	headers["To"] = toAddr.String()
	headers["Subject"] = encodeRFC2047(subject)
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"
	headers["Content-Transfer-Encoding"] = "quoted-printable"
	headers["Date"] = time.Now().Format(time.RFC1123Z)

	// Add BCC header if configured (for audit trail, recipient won't see it)
	if p.bccAdmin != "" {
		bccAddr, err := mail.ParseAddress(p.bccAdmin)
		if err != nil {
			bccAddr = &mail.Address{Address: p.bccAdmin}
		}
		headers["Bcc"] = bccAddr.String()
	}

	// Build message
	var msg strings.Builder

	// Write headers
	for key, value := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}

	// Blank line between headers and body
	msg.WriteString("\r\n")

	// Write body (quoted-printable encoded for UTF-8 support)
	msg.WriteString(encodeQuotedPrintable(htmlBody))

	return []byte(msg.String())
}

// encodeRFC2047 encodes a string using RFC 2047 for email headers (supports UTF-8)
func encodeRFC2047(s string) string {
	// Check if encoding is needed (contains non-ASCII characters)
	needsEncoding := false
	for _, c := range s {
		if c > 127 {
			needsEncoding = true
			break
		}
	}

	if !needsEncoding {
		return s
	}

	// RFC 2047 encoding: =?UTF-8?Q?encoded_text?=
	return fmt.Sprintf("=?UTF-8?B?%s?=", encodeBase64(s))
}

// encodeBase64 encodes a string to base64 for RFC 2047
func encodeBase64(s string) string {
	// Simple base64 encoding
	const base64Table = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	data := []byte(s)

	var result strings.Builder
	for i := 0; i < len(data); i += 3 {
		// Get up to 3 bytes
		b1, b2, b3 := data[i], byte(0), byte(0)
		n := 1
		if i+1 < len(data) {
			b2 = data[i+1]
			n = 2
		}
		if i+2 < len(data) {
			b3 = data[i+2]
			n = 3
		}

		// Encode to 4 base64 characters
		result.WriteByte(base64Table[b1>>2])
		result.WriteByte(base64Table[((b1&0x03)<<4)|(b2>>4)])
		if n > 1 {
			result.WriteByte(base64Table[((b2&0x0F)<<2)|(b3>>6)])
		} else {
			result.WriteByte('=')
		}
		if n > 2 {
			result.WriteByte(base64Table[b3&0x3F])
		} else {
			result.WriteByte('=')
		}
	}

	return result.String()
}

// encodeQuotedPrintable encodes HTML body using quoted-printable encoding
func encodeQuotedPrintable(s string) string {
	var result strings.Builder
	lineLen := 0
	maxLineLen := 76

	for i := 0; i < len(s); i++ {
		c := s[i]

		// Check if character needs encoding
		needsEncoding := c < 33 || c > 126 || c == '='

		if needsEncoding {
			// Encode as =XX where XX is hex
			encoded := fmt.Sprintf("=%02X", c)

			// Check line length
			if lineLen+len(encoded) > maxLineLen {
				result.WriteString("=\r\n") // Soft line break
				lineLen = 0
			}

			result.WriteString(encoded)
			lineLen += len(encoded)
		} else {
			// Check line length for regular character
			if lineLen >= maxLineLen {
				result.WriteString("=\r\n") // Soft line break
				lineLen = 0
			}

			result.WriteByte(c)
			lineLen++

			// Handle CRLF
			if c == '\n' {
				lineLen = 0
			}
		}
	}

	return result.String()
}

// ValidateConfig validates the SMTP provider configuration
func (p *SMTPProvider) ValidateConfig() error {
	if p.host == "" {
		return fmt.Errorf("invalid email configuration: SMTP_HOST is required")
	}

	if p.port == 0 {
		return fmt.Errorf("invalid email configuration: SMTP_PORT is required")
	}

	// Validate port range
	if p.port < 1 || p.port > 65535 {
		return fmt.Errorf("invalid email configuration: SMTP_PORT must be between 1 and 65535")
	}

	if p.fromEmail == "" {
		return fmt.Errorf("invalid email configuration: SMTP_FROM_EMAIL is required")
	}

	// Validate from email format
	if _, err := mail.ParseAddress(p.fromEmail); err != nil {
		return fmt.Errorf("invalid email configuration: SMTP_FROM_EMAIL is not a valid email address")
	}

	// Validate BCC email format if provided
	if p.bccAdmin != "" {
		if _, err := mail.ParseAddress(p.bccAdmin); err != nil {
			return fmt.Errorf("invalid email configuration: EMAIL_BCC_ADMIN is not a valid email address")
		}
	}

	// Username and password are optional (some SMTP servers don't require auth)
	// But if one is provided, both should be provided
	if (p.username != "" && p.password == "") || (p.username == "" && p.password != "") {
		return fmt.Errorf("invalid email configuration: both SMTP_USERNAME and SMTP_PASSWORD must be provided together")
	}

	// Validate TLS/SSL configuration
	if p.useTLS && p.useSSL {
		return fmt.Errorf("invalid email configuration: cannot use both SMTP_USE_TLS and SMTP_USE_SSL")
	}

	// Port validation for common configurations
	if p.useSSL && p.port != 465 {
		fmt.Printf("Warning: SMTP_USE_SSL is true but port is %d (expected 465)\n", p.port)
	}
	if p.useTLS && p.port != 587 {
		fmt.Printf("Warning: SMTP_USE_TLS is true but port is %d (expected 587)\n", p.port)
	}

	return nil
}

// Close closes any open connections (SMTP is stateless, so this is a no-op)
func (p *SMTPProvider) Close() error {
	// SMTP connections are created and closed per-email
	// No persistent connection to close
	return nil
}

// GetFromEmail returns the configured from email address
func (p *SMTPProvider) GetFromEmail() string {
	return p.fromEmail
}
