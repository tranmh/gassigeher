package services

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"strings"
)

// EmailService handles sending emails via any email provider
type EmailService struct {
	provider EmailProvider
	baseURL  string // Base URL for email links
}

// NewEmailService creates a new email service with the specified provider
func NewEmailService(config *EmailConfig) (*EmailService, error) {
	if config == nil {
		return nil, fmt.Errorf("email config cannot be nil")
	}

	// Validate configuration
	if err := ValidateEmailConfig(config); err != nil {
		return nil, fmt.Errorf("invalid email configuration: %w", err)
	}

	// Create provider using factory
	provider, err := NewEmailProvider(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create email provider: %w", err)
	}

	// Validate provider
	if err := provider.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("provider validation failed: %w", err)
	}

	log.Printf("Email service initialized with provider: %s (from: %s)", config.Provider, provider.GetFromEmail())
	if config.BCCAdmin != "" {
		log.Printf("BCC admin copy enabled: %s", config.BCCAdmin)
	}

	// Use default base URL if not provided
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	return &EmailService{
		provider: provider,
		baseURL:  baseURL,
	}, nil
}

// NewEmailServiceLegacy creates email service using legacy Gmail API parameters (backward compatibility)
// DEPRECATED: Use NewEmailService(config) instead
func NewEmailServiceLegacy(clientID, clientSecret, refreshToken, fromEmail string) (*EmailService, error) {
	config := &EmailConfig{
		Provider:          "gmail",
		GmailClientID:     clientID,
		GmailClientSecret: clientSecret,
		GmailRefreshToken: refreshToken,
		GmailFromEmail:    fromEmail,
	}
	return NewEmailService(config)
}

// SendEmail sends an email using the configured provider
// Skips sending for demo tenant emails (ending with @demo.gassigeher.org)
func (s *EmailService) SendEmail(to, subject, body string) error {
	// Skip emails for demo tenant users
	if isDemoEmail(to) {
		log.Printf("Skipping email to demo tenant user: %s (subject: %s)", to, subject)
		return nil
	}
	return s.provider.SendEmail(to, subject, body)
}

// isDemoEmail checks if an email address belongs to a demo tenant user
func isDemoEmail(email string) bool {
	return strings.HasSuffix(email, "@demo.gassigeher.org")
}

// SendVerificationEmail sends an email verification link
func (s *EmailService) SendVerificationEmail(to, name, token string) error {
	subject := "Willkommen bei Gassigeher - E-Mail-Adresse bestätigen"

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Titillium, Arial, sans-serif; line-height: 1.6; color: #26272b; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #82b965; color: white; padding: 20px; text-align: center; border-radius: 6px 6px 0 0; }
        .content { background-color: #f9f9f9; padding: 30px; border-radius: 0 0 6px 6px; }
        .button { display: inline-block; padding: 12px 30px; background-color: #82b965; color: white; text-decoration: none; border-radius: 6px; margin: 20px 0; }
        .footer { text-align: center; margin-top: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🐕 Willkommen bei Gassigeher</h1>
        </div>
        <div class="content">
            <p>Hallo {{.Name}},</p>
            <p>vielen Dank für Ihre Registrierung bei Gassigeher! Bitte bestätigen Sie Ihre E-Mail-Adresse, um Ihr Konto zu aktivieren.</p>
            <p style="text-align: center;">
                <a href="{{.BaseURL}}/verify?token={{.Token}}" class="button">E-Mail-Adresse bestätigen</a>
            </p>
            <p>Oder kopieren Sie diesen Link in Ihren Browser:</p>
            <p style="word-break: break-all; font-size: 12px; color: #666;">
                {{.BaseURL}}/verify?token={{.Token}}
            </p>
            <p>Dieser Link ist 24 Stunden gültig.</p>
            <p>Wenn Sie sich nicht bei Gassigeher registriert haben, können Sie diese E-Mail ignorieren.</p>
        </div>
        <div class="footer">
            <p>© 2025 Gassigeher. Alle Rechte vorbehalten.</p>
        </div>
    </div>
</body>
</html>
`

	t := template.Must(template.New("verification").Parse(tmpl))
	var body bytes.Buffer
	if err := t.Execute(&body, map[string]string{
		"Name":    name,
		"Token":   token,
		"BaseURL": s.baseURL,
	}); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return s.SendEmail(to, subject, body.String())
}

// SendWelcomeEmail sends a welcome email after verification
func (s *EmailService) SendWelcomeEmail(to, name string) error {
	subject := "Los geht's! Ihr Konto ist aktiviert"

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Titillium, Arial, sans-serif; line-height: 1.6; color: #26272b; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #82b965; color: white; padding: 20px; text-align: center; border-radius: 6px 6px 0 0; }
        .content { background-color: #f9f9f9; padding: 30px; border-radius: 0 0 6px 6px; }
        .feature { margin: 15px 0; padding: 15px; background-color: white; border-left: 4px solid #82b965; }
        .footer { text-align: center; margin-top: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🎉 Willkommen bei Gassigeher!</h1>
        </div>
        <div class="content">
            <p>Hallo {{.Name}},</p>
            <p>Ihr Konto ist jetzt aktiviert! Sie können sofort mit dem Buchen von Hunden beginnen.</p>

            <h3>So funktioniert's:</h3>

            <div class="feature">
                <strong>🐶 Hunde durchsuchen</strong><br>
                Sehen Sie sich alle verfügbaren Hunde an und filtern Sie nach Größe, Rasse und Erfahrungslevel.
            </div>

            <div class="feature">
                <strong>📅 Termine buchen</strong><br>
                Wählen Sie einen Hund und einen Zeitpunkt für Ihren Spaziergang. Sie können die vorgeschlagenen Zeiten anpassen.
            </div>

            <div class="feature">
                <strong>⭐ Erfahrungslevel</strong><br>
                Sie starten als "Grün" (Anfänger). Sie können höhere Levels beantragen, um Zugang zu anspruchsvolleren Hunden zu erhalten:
                <ul>
                    <li><strong>Grün:</strong> Alle Anfänger (Standard)</li>
                    <li><strong>Blau:</strong> Erfahrene Gassigeher</li>
                    <li><strong>Orange:</strong> Nur erfahrene Gassigeher</li>
                </ul>
            </div>

            <p>Bei Fragen oder Problemen wenden Sie sich bitte an unseren Support.</p>

            <p style="text-align: center; margin-top: 30px;">
                <a href="{{.BaseURL}}" style="display: inline-block; padding: 12px 30px; background-color: #82b965; color: white; text-decoration: none; border-radius: 6px;">Zur Anwendung</a>
            </p>
        </div>
        <div class="footer">
            <p>© 2025 Gassigeher. Alle Rechte vorbehalten.</p>
        </div>
    </div>
</body>
</html>
`

	t := template.Must(template.New("welcome").Parse(tmpl))
	var body bytes.Buffer
	if err := t.Execute(&body, map[string]string{
		"Name":    name,
		"BaseURL": s.baseURL,
	}); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return s.SendEmail(to, subject, body.String())
}

// SendTempPasswordEmail sends an email with temporary password for admin-created users
func (s *EmailService) SendTempPasswordEmail(to, name, tempPassword string) error {
	subject := "Ihr Konto wurde erstellt - Gassigeher"

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Titillium, Arial, sans-serif; line-height: 1.6; color: #26272b; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #82b965; color: white; padding: 20px; text-align: center; border-radius: 6px 6px 0 0; }
        .content { background-color: #f9f9f9; padding: 30px; border-radius: 0 0 6px 6px; }
        .password-box { background-color: #fff3cd; padding: 20px; margin: 20px 0; border-radius: 6px; border-left: 4px solid #ffc107; text-align: center; }
        .password { font-size: 1.5rem; font-family: monospace; font-weight: bold; letter-spacing: 2px; color: #26272b; }
        .warning { background-color: #f8d7da; padding: 15px; margin: 20px 0; border-radius: 6px; border-left: 4px solid #dc3545; }
        .footer { text-align: center; margin-top: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🐕 Willkommen bei Gassigeher!</h1>
        </div>
        <div class="content">
            <p>Hallo {{.Name}},</p>
            <p>Ein Administrator hat ein Konto für Sie erstellt. Sie können sich jetzt mit den folgenden Zugangsdaten anmelden:</p>

            <p><strong>E-Mail:</strong> {{.Email}}</p>

            <div class="password-box">
                <p style="margin: 0 0 10px 0;">Ihr temporäres Passwort:</p>
                <span class="password">{{.TempPassword}}</span>
            </div>

            <div class="warning">
                <strong>⚠️ Wichtig:</strong> Sie werden bei der ersten Anmeldung aufgefordert, ein neues Passwort zu wählen. Das temporäre Passwort ist nur für die erste Anmeldung gültig.
            </div>

            <p style="text-align: center; margin-top: 30px;">
                <a href="{{.BaseURL}}/login.html" style="display: inline-block; padding: 12px 30px; background-color: #82b965; color: white; text-decoration: none; border-radius: 6px;">Jetzt anmelden</a>
            </p>
        </div>
        <div class="footer">
            <p>© 2025 Gassigeher. Alle Rechte vorbehalten.</p>
        </div>
    </div>
</body>
</html>
`

	t := template.Must(template.New("temp_password").Parse(tmpl))
	var body bytes.Buffer
	if err := t.Execute(&body, map[string]string{
		"Name":         name,
		"Email":        to,
		"TempPassword": tempPassword,
		"BaseURL":      s.baseURL,
	}); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return s.SendEmail(to, subject, body.String())
}

// SendPasswordResetEmail sends a password reset link
func (s *EmailService) SendPasswordResetEmail(to, name, token string) error {
	subject := "Passwort zurücksetzen - Gassigeher"

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Titillium, Arial, sans-serif; line-height: 1.6; color: #26272b; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #82b965; color: white; padding: 20px; text-align: center; border-radius: 6px 6px 0 0; }
        .content { background-color: #f9f9f9; padding: 30px; border-radius: 0 0 6px 6px; }
        .button { display: inline-block; padding: 12px 30px; background-color: #82b965; color: white; text-decoration: none; border-radius: 6px; margin: 20px 0; }
        .warning { background-color: #fff3cd; border-left: 4px solid #ffc107; padding: 15px; margin: 20px 0; }
        .footer { text-align: center; margin-top: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔑 Passwort zurücksetzen</h1>
        </div>
        <div class="content">
            <p>Hallo {{.Name}},</p>
            <p>Sie haben eine Anfrage zum Zurücksetzen Ihres Passworts gestellt. Klicken Sie auf den Button unten, um ein neues Passwort festzulegen.</p>
            <p style="text-align: center;">
                <a href="{{.BaseURL}}/reset-password?token={{.Token}}" class="button">Neues Passwort festlegen</a>
            </p>
            <p>Oder kopieren Sie diesen Link in Ihren Browser:</p>
            <p style="word-break: break-all; font-size: 12px; color: #666;">
                {{.BaseURL}}/reset-password?token={{.Token}}
            </p>
            <div class="warning">
                <strong>⚠️ Wichtig:</strong> Dieser Link ist nur 1 Stunde gültig.
            </div>
            <p>Wenn Sie diese Anfrage nicht gestellt haben, können Sie diese E-Mail ignorieren. Ihr Passwort bleibt unverändert.</p>
        </div>
        <div class="footer">
            <p>© 2025 Gassigeher. Alle Rechte vorbehalten.</p>
        </div>
    </div>
</body>
</html>
`

	t := template.Must(template.New("reset").Parse(tmpl))
	var body bytes.Buffer
	if err := t.Execute(&body, map[string]string{
		"Name":    name,
		"Token":   token,
		"BaseURL": s.baseURL,
	}); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return s.SendEmail(to, subject, body.String())
}

// SendBookingConfirmation sends a booking confirmation email
func (s *EmailService) SendBookingConfirmation(to, name, dogName, date, scheduledTime string) error {
	subject := fmt.Sprintf("Buchungsbestätigung - %s", dogName)

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #26272b; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #82b965; color: white; padding: 20px; text-align: center; border-radius: 6px 6px 0 0; }
        .content { background-color: #f9f9f9; padding: 30px; border-radius: 0 0 6px 6px; }
        .booking-details { background-color: white; padding: 20px; margin: 20px 0; border-radius: 6px; border-left: 4px solid #82b965; }
        .detail-row { margin: 10px 0; }
        .label { font-weight: 600; color: #666; }
        .footer { text-align: center; margin-top: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>✅ Buchung bestätigt!</h1>
        </div>
        <div class="content">
            <p>Hallo {{.Name}},</p>
            <p>Ihre Buchung wurde erfolgreich bestätigt.</p>

            <div class="booking-details">
                <h3 style="margin-top: 0;">Buchungsdetails</h3>
                <div class="detail-row">
                    <span class="label">Hund:</span> {{.DogName}}
                </div>
                <div class="detail-row">
                    <span class="label">Datum:</span> {{.Date}}
                </div>
                <div class="detail-row">
                    <span class="label">Uhrzeit:</span> {{.ScheduledTime}} Uhr
                </div>
            </div>

            <p>Sie erhalten eine Erinnerung 1 Stunde vor Ihrem Spaziergang.</p>
            <p>Falls Sie den Termin stornieren möchten, tun Sie dies bitte mindestens 12 Stunden im Voraus über Ihr Dashboard.</p>
        </div>
        <div class="footer">
            <p>© 2025 Gassigeher. Alle Rechte vorbehalten.</p>
        </div>
    </div>
</body>
</html>
`

	t := template.Must(template.New("booking").Parse(tmpl))
	var body bytes.Buffer
	data := map[string]string{
		"Name":          name,
		"DogName":       dogName,
		"Date":          date,
		"ScheduledTime": scheduledTime,
	}
	if err := t.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return s.SendEmail(to, subject, body.String())
}

// SendBookingCancellation sends a booking cancellation confirmation (user-initiated)
func (s *EmailService) SendBookingCancellation(to, name, dogName, date, scheduledTime string) error {
	subject := fmt.Sprintf("Buchung storniert - %s", dogName)

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #26272b; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #dc3545; color: white; padding: 20px; text-align: center; border-radius: 6px 6px 0 0; }
        .content { background-color: #f9f9f9; padding: 30px; border-radius: 0 0 6px 6px; }
        .booking-details { background-color: white; padding: 20px; margin: 20px 0; border-radius: 6px; border-left: 4px solid #dc3545; }
        .detail-row { margin: 10px 0; }
        .label { font-weight: 600; color: #666; }
        .footer { text-align: center; margin-top: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Buchung storniert</h1>
        </div>
        <div class="content">
            <p>Hallo {{.Name}},</p>
            <p>Ihre Buchung wurde erfolgreich storniert.</p>

            <div class="booking-details">
                <h3 style="margin-top: 0;">Stornierte Buchung</h3>
                <div class="detail-row">
                    <span class="label">Hund:</span> {{.DogName}}
                </div>
                <div class="detail-row">
                    <span class="label">Datum:</span> {{.Date}}
                </div>
                <div class="detail-row">
                    <span class="label">Uhrzeit:</span> {{.ScheduledTime}} Uhr
                </div>
            </div>

            <p>Sie können jederzeit eine neue Buchung vornehmen.</p>
        </div>
        <div class="footer">
            <p>© 2025 Gassigeher. Alle Rechte vorbehalten.</p>
        </div>
    </div>
</body>
</html>
`

	t := template.Must(template.New("cancellation").Parse(tmpl))
	var body bytes.Buffer
	data := map[string]string{
		"Name":          name,
		"DogName":       dogName,
		"Date":          date,
		"ScheduledTime": scheduledTime,
	}
	if err := t.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return s.SendEmail(to, subject, body.String())
}

// SendAdminCancellation sends an admin cancellation notification
func (s *EmailService) SendAdminCancellation(to, name, dogName, date, scheduledTime, reason string) error {
	subject := fmt.Sprintf("Deine Buchung wurde storniert - %s", dogName)

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #26272b; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #dc3545; color: white; padding: 20px; text-align: center; border-radius: 6px 6px 0 0; }
        .content { background-color: #f9f9f9; padding: 30px; border-radius: 0 0 6px 6px; }
        .booking-details { background-color: white; padding: 20px; margin: 20px 0; border-radius: 6px; border-left: 4px solid #dc3545; }
        .reason-box { background-color: #fff3cd; padding: 15px; margin: 20px 0; border-radius: 6px; border-left: 4px solid #ffc107; }
        .detail-row { margin: 10px 0; }
        .label { font-weight: 600; color: #666; }
        .footer { text-align: center; margin-top: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Buchung storniert</h1>
        </div>
        <div class="content">
            <p>Hallo {{.Name}},</p>
            <p>Leider mussten wir Ihre folgende Buchung stornieren:</p>

            <div class="booking-details">
                <h3 style="margin-top: 0;">Stornierte Buchung</h3>
                <div class="detail-row">
                    <span class="label">Hund:</span> {{.DogName}}
                </div>
                <div class="detail-row">
                    <span class="label">Datum:</span> {{.Date}}
                </div>
                <div class="detail-row">
                    <span class="label">Uhrzeit:</span> {{.ScheduledTime}} Uhr
                </div>
            </div>

            <div class="reason-box">
                <strong>Grund der Stornierung:</strong><br>
                {{.Reason}}
            </div>

            <p>Wir entschuldigen uns für die Unannehmlichkeiten. Sie können gerne einen anderen Termin buchen.</p>
        </div>
        <div class="footer">
            <p>© 2025 Gassigeher. Alle Rechte vorbehalten.</p>
        </div>
    </div>
</body>
</html>
`

	t := template.Must(template.New("admin_cancel").Parse(tmpl))
	var body bytes.Buffer
	data := map[string]string{
		"Name":          name,
		"DogName":       dogName,
		"Date":          date,
		"ScheduledTime": scheduledTime,
		"Reason":        reason,
	}
	if err := t.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return s.SendEmail(to, subject, body.String())
}

// SendBookingReminder sends a reminder 1 hour before the booking
func (s *EmailService) SendBookingReminder(to, name, dogName, date, scheduledTime string) error {
	subject := fmt.Sprintf("Erinnerung: Gassirunde mit %s in 1 Stunde", dogName)

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #26272b; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #17a2b8; color: white; padding: 20px; text-align: center; border-radius: 6px 6px 0 0; }
        .content { background-color: #f9f9f9; padding: 30px; border-radius: 0 0 6px 6px; }
        .booking-details { background-color: white; padding: 20px; margin: 20px 0; border-radius: 6px; border-left: 4px solid #17a2b8; }
        .detail-row { margin: 10px 0; }
        .label { font-weight: 600; color: #666; }
        .footer { text-align: center; margin-top: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔔 Erinnerung</h1>
        </div>
        <div class="content">
            <p>Hallo {{.Name}},</p>
            <p>Dies ist eine Erinnerung an Ihren bevorstehenden Spaziergang:</p>

            <div class="booking-details">
                <h3 style="margin-top: 0;">Ihr Spaziergang</h3>
                <div class="detail-row">
                    <span class="label">Hund:</span> {{.DogName}}
                </div>
                <div class="detail-row">
                    <span class="label">Datum:</span> {{.Date}}
                </div>
                <div class="detail-row">
                    <span class="label">Uhrzeit:</span> {{.ScheduledTime}} Uhr
                </div>
            </div>

            <p>Viel Spaß beim Spaziergang!</p>
        </div>
        <div class="footer">
            <p>© 2025 Gassigeher. Alle Rechte vorbehalten.</p>
        </div>
    </div>
</body>
</html>
`

	t := template.Must(template.New("reminder").Parse(tmpl))
	var body bytes.Buffer
	data := map[string]string{
		"Name":          name,
		"DogName":       dogName,
		"Date":          date,
		"ScheduledTime": scheduledTime,
	}
	if err := t.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return s.SendEmail(to, subject, body.String())
}

// SendBookingMoved sends an email when admin moves a booking
func (s *EmailService) SendBookingMoved(to, name, dogName, oldDate, oldTime, newDate, newTime, reason string) error {
	subject := fmt.Sprintf("Deine Buchung wurde verschoben - %s", dogName)

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #26272b; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #17a2b8; color: white; padding: 20px; text-align: center; border-radius: 6px 6px 0 0; }
        .content { background-color: #f9f9f9; padding: 30px; border-radius: 0 0 6px 6px; }
        .booking-details { background-color: white; padding: 20px; margin: 20px 0; border-radius: 6px; }
        .old-details { border-left: 4px solid #dc3545; }
        .new-details { border-left: 4px solid #28a745; margin-top: 20px; }
        .reason-box { background-color: #fff3cd; padding: 15px; margin: 20px 0; border-radius: 6px; border-left: 4px solid #ffc107; }
        .detail-row { margin: 10px 0; }
        .label { font-weight: 600; color: #666; }
        .footer { text-align: center; margin-top: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Buchung verschoben</h1>
        </div>
        <div class="content">
            <p>Hallo {{.Name}},</p>
            <p>Ihre Buchung wurde auf einen neuen Termin verschoben:</p>

            <div class="booking-details old-details">
                <h3 style="margin-top: 0; color: #dc3545;">Alter Termin</h3>
                <div class="detail-row">
                    <span class="label">Hund:</span> {{.DogName}}
                </div>
                <div class="detail-row">
                    <span class="label">Datum:</span> {{.OldDate}}
                </div>
                <div class="detail-row">
                    <span class="label">Uhrzeit:</span> {{.OldTime}} Uhr
                </div>
            </div>

            <div class="booking-details new-details">
                <h3 style="margin-top: 0; color: #28a745;">Neuer Termin</h3>
                <div class="detail-row">
                    <span class="label">Hund:</span> {{.DogName}}
                </div>
                <div class="detail-row">
                    <span class="label">Datum:</span> {{.NewDate}}
                </div>
                <div class="detail-row">
                    <span class="label">Uhrzeit:</span> {{.NewTime}} Uhr
                </div>
            </div>

            <div class="reason-box">
                <strong>Grund der Verschiebung:</strong><br>
                {{.Reason}}
            </div>

            <p>Wir entschuldigen uns für die Unannehmlichkeiten. Bei Fragen oder Problemen wenden Sie sich bitte an uns.</p>
        </div>
        <div class="footer">
            <p>© 2025 Gassigeher. Alle Rechte vorbehalten.</p>
        </div>
    </div>
</body>
</html>
`

	t := template.Must(template.New("moved").Parse(tmpl))
	var body bytes.Buffer
	data := map[string]string{
		"Name":    name,
		"DogName": dogName,
		"OldDate": oldDate,
		"OldTime": oldTime,
		"NewDate": newDate,
		"NewTime": newTime,
		"Reason":  reason,
	}
	if err := t.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return s.SendEmail(to, subject, body.String())
}

// SendBookingApproved sends a notification when a pending booking is approved by admin
func (s *EmailService) SendBookingApproved(to, name, dogName, date, scheduledTime string) error {
	subject := fmt.Sprintf("Buchung genehmigt - %s am %s", dogName, date)

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #26272b; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #28a745; color: white; padding: 20px; text-align: center; border-radius: 6px 6px 0 0; }
        .content { background-color: #f9f9f9; padding: 30px; border-radius: 0 0 6px 6px; }
        .booking-details { background-color: white; padding: 20px; margin: 20px 0; border-radius: 6px; border-left: 4px solid #28a745; }
        .detail-row { margin: 10px 0; }
        .label { font-weight: 600; color: #666; }
        .footer { text-align: center; margin-top: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>✅ Buchung genehmigt!</h1>
        </div>
        <div class="content">
            <p>Hallo {{.Name}},</p>
            <p>Gute Nachrichten! Ihre Buchungsanfrage wurde genehmigt.</p>

            <div class="booking-details">
                <h3 style="margin-top: 0;">Buchungsdetails</h3>
                <div class="detail-row">
                    <span class="label">Hund:</span> {{.DogName}}
                </div>
                <div class="detail-row">
                    <span class="label">Datum:</span> {{.Date}}
                </div>
                <div class="detail-row">
                    <span class="label">Uhrzeit:</span> {{.ScheduledTime}} Uhr
                </div>
            </div>

            <p>Sie können nun wie geplant mit {{.DogName}} spazieren gehen.</p>
            <p>Falls Sie den Termin stornieren möchten, tun Sie dies bitte mindestens 12 Stunden im Voraus über Ihr Dashboard.</p>
        </div>
        <div class="footer">
            <p>© 2025 Gassigeher. Alle Rechte vorbehalten.</p>
        </div>
    </div>
</body>
</html>
`

	t := template.Must(template.New("approval").Parse(tmpl))
	var body bytes.Buffer
	data := map[string]string{
		"Name":          name,
		"DogName":       dogName,
		"Date":          date,
		"ScheduledTime": scheduledTime,
	}
	if err := t.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return s.SendEmail(to, subject, body.String())
}

// SendBookingRejected sends a notification when a pending booking is rejected by admin
func (s *EmailService) SendBookingRejected(to, name, dogName, date, scheduledTime, reason string) error {
	subject := fmt.Sprintf("Buchung abgelehnt - %s am %s", dogName, date)

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #26272b; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #dc3545; color: white; padding: 20px; text-align: center; border-radius: 6px 6px 0 0; }
        .content { background-color: #f9f9f9; padding: 30px; border-radius: 0 0 6px 6px; }
        .booking-details { background-color: white; padding: 20px; margin: 20px 0; border-radius: 6px; border-left: 4px solid #dc3545; }
        .reason-box { background-color: #fff3cd; padding: 15px; margin: 20px 0; border-radius: 6px; border-left: 4px solid #ffc107; }
        .detail-row { margin: 10px 0; }
        .label { font-weight: 600; color: #666; }
        .footer { text-align: center; margin-top: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>❌ Buchung abgelehnt</h1>
        </div>
        <div class="content">
            <p>Hallo {{.Name}},</p>
            <p>Leider mussten wir Ihre Buchungsanfrage ablehnen.</p>

            <div class="booking-details">
                <h3 style="margin-top: 0;">Buchungsdetails</h3>
                <div class="detail-row">
                    <span class="label">Hund:</span> {{.DogName}}
                </div>
                <div class="detail-row">
                    <span class="label">Datum:</span> {{.Date}}
                </div>
                <div class="detail-row">
                    <span class="label">Uhrzeit:</span> {{.ScheduledTime}} Uhr
                </div>
            </div>

            <div class="reason-box">
                <strong>Begründung:</strong>
                <p style="margin-bottom: 0;">{{.Reason}}</p>
            </div>

            <p>Bitte versuchen Sie eine Buchung zu einem anderen Zeitpunkt oder kontaktieren Sie uns bei Fragen.</p>
        </div>
        <div class="footer">
            <p>© 2025 Gassigeher. Alle Rechte vorbehalten.</p>
        </div>
    </div>
</body>
</html>
`

	t := template.Must(template.New("rejection").Parse(tmpl))
	var body bytes.Buffer
	data := map[string]string{
		"Name":          name,
		"DogName":       dogName,
		"Date":          date,
		"ScheduledTime": scheduledTime,
		"Reason":        reason,
	}
	if err := t.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return s.SendEmail(to, subject, body.String())
}

// SendExperienceLevelApproved sends an email when experience level request is approved
func (s *EmailService) SendExperienceLevelApproved(to, name, level string, message *string) error {
	levelLabel := "Blau"
	if level == "orange" {
		levelLabel = "Orange"
	}

	subject := fmt.Sprintf("Ihr Antrag auf %s Level wurde genehmigt", levelLabel)

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #26272b; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #28a745; color: white; padding: 20px; text-align: center; border-radius: 6px 6px 0 0; }
        .content { background-color: #f9f9f9; padding: 30px; border-radius: 0 0 6px 6px; }
        .success-box { background-color: #d4edda; padding: 20px; margin: 20px 0; border-radius: 6px; border-left: 4px solid #28a745; }
        .message-box { background-color: white; padding: 15px; margin: 20px 0; border-radius: 6px; border-left: 4px solid #17a2b8; }
        .footer { text-align: center; margin-top: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>✅ Glückwunsch!</h1>
        </div>
        <div class="content">
            <p>Hallo {{.Name}},</p>
            <p>Ihr Antrag auf <strong>{{.Level}} Level</strong> wurde genehmigt!</p>

            <div class="success-box">
                <h3 style="margin-top: 0;">Sie haben jetzt Zugang zu:</h3>
                <p style="margin: 5px 0;">
                    {{if eq .Level "Blau"}}
                    ✓ Grüne Hunde (Anfänger)<br>
                    ✓ Blaue Hunde (Erfahrene)
                    {{else}}
                    ✓ Grüne Hunde (Anfänger)<br>
                    ✓ Blaue Hunde (Erfahrene)<br>
                    ✓ Orange Hunde (Nur Erfahrene)
                    {{end}}
                </p>
            </div>

            {{if .Message}}
            <div class="message-box">
                <strong>Nachricht vom Administrator:</strong><br>
                {{.Message}}
            </div>
            {{end}}

            <p>Sie können jetzt sofort Hunde Ihres neuen Levels buchen!</p>

            <p style="text-align: center; margin-top: 30px;">
                <a href="{{.BaseURL}}/dogs.html" style="display: inline-block; padding: 12px 30px; background-color: #82b965; color: white; text-decoration: none; border-radius: 6px;">Hunde anzeigen</a>
            </p>
        </div>
        <div class="footer">
            <p>© 2025 Gassigeher. Alle Rechte vorbehalten.</p>
        </div>
    </div>
</body>
</html>
`

	t := template.Must(template.New("approved").Parse(tmpl))
	var body bytes.Buffer
	data := map[string]interface{}{
		"Name":    name,
		"Level":   levelLabel,
		"BaseURL": s.baseURL,
		"Message": func() string {
			if message != nil {
				return *message
			}
			return ""
		}(),
	}
	if err := t.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return s.SendEmail(to, subject, body.String())
}

// SendNewUserRegistrationNotification sends an email to admin when a new user registers
func (s *EmailService) SendNewUserRegistrationNotification(adminEmail, userName, userEmail, userPhone string) error {
	subject := "Neue Registrierung - " + userName

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #26272b; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #17a2b8; color: white; padding: 20px; text-align: center; border-radius: 6px 6px 0 0; }
        .content { background-color: #f9f9f9; padding: 30px; border-radius: 0 0 6px 6px; }
        .user-details { background-color: white; padding: 20px; margin: 20px 0; border-radius: 6px; border-left: 4px solid #17a2b8; }
        .detail-row { margin: 10px 0; }
        .label { font-weight: 600; color: #666; }
        .whatsapp-hint { background-color: #dcf8c6; padding: 15px; margin: 20px 0; border-radius: 6px; border-left: 4px solid #25d366; }
        .footer { text-align: center; margin-top: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>👤 Neue Registrierung</h1>
        </div>
        <div class="content">
            <p>Ein neuer Benutzer hat sich bei Gassigeher registriert:</p>

            <div class="user-details">
                <h3 style="margin-top: 0;">Benutzerdetails</h3>
                <div class="detail-row">
                    <span class="label">Name:</span> {{.UserName}}
                </div>
                <div class="detail-row">
                    <span class="label">E-Mail:</span> {{.UserEmail}}
                </div>
                <div class="detail-row">
                    <span class="label">Telefon:</span> {{.UserPhone}}
                </div>
            </div>

            <div class="whatsapp-hint">
                <strong>💬 WhatsApp-Gruppe</strong><br>
                Falls der Benutzer der WhatsApp-Gruppe nicht selbst beigetreten ist, können Sie ihn mit der obigen Telefonnummer manuell hinzufügen.
            </div>

            <p style="font-size: 0.9rem; color: #666;">
                Der Benutzer muss seine E-Mail-Adresse noch bestätigen, bevor er sich anmelden kann.
            </p>
        </div>
        <div class="footer">
            <p>© 2025 Gassigeher. Alle Rechte vorbehalten.</p>
        </div>
    </div>
</body>
</html>
`

	t := template.Must(template.New("new_user").Parse(tmpl))
	var body bytes.Buffer
	data := map[string]string{
		"UserName":  userName,
		"UserEmail": userEmail,
		"UserPhone": userPhone,
	}
	if err := t.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return s.SendEmail(adminEmail, subject, body.String())
}

// SendExperienceLevelDenied sends an email when experience level request is denied
func (s *EmailService) SendExperienceLevelDenied(to, name, level string, message *string) error {
	levelLabel := "Blau"
	if level == "orange" {
		levelLabel = "Orange"
	}

	subject := fmt.Sprintf("Ihr Antrag auf %s Level", levelLabel)

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #26272b; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #ffc107; color: #26272b; padding: 20px; text-align: center; border-radius: 6px 6px 0 0; }
        .content { background-color: #f9f9f9; padding: 30px; border-radius: 0 0 6px 6px; }
        .info-box { background-color: #fff3cd; padding: 20px; margin: 20px 0; border-radius: 6px; border-left: 4px solid #ffc107; }
        .message-box { background-color: white; padding: 15px; margin: 20px 0; border-radius: 6px; border-left: 4px solid #17a2b8; }
        .footer { text-align: center; margin-top: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Ihr Antrag auf {{.Level}} Level</h1>
        </div>
        <div class="content">
            <p>Hallo {{.Name}},</p>
            <p>Vielen Dank für Ihren Antrag auf <strong>{{.Level}} Level</strong>.</p>

            <div class="info-box">
                <p style="margin: 0;">
                    Leider können wir Ihren Antrag derzeit nicht genehmigen. Sammeln Sie weiterhin Erfahrung und versuchen Sie es später erneut!
                </p>
            </div>

            {{if .Message}}
            <div class="message-box">
                <strong>Nachricht vom Administrator:</strong><br>
                {{.Message}}
            </div>
            {{end}}

            <p>Sie können weiterhin Hunde Ihres aktuellen Levels buchen und jederzeit einen neuen Antrag stellen.</p>
        </div>
        <div class="footer">
            <p>© 2025 Gassigeher. Alle Rechte vorbehalten.</p>
        </div>
    </div>
</body>
</html>
`

	t := template.Must(template.New("denied").Parse(tmpl))
	var body bytes.Buffer
	data := map[string]interface{}{
		"Name":  name,
		"Level": levelLabel,
		"Message": func() string {
			if message != nil {
				return *message
			}
			return ""
		}(),
	}
	if err := t.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return s.SendEmail(to, subject, body.String())
}

// SendTenantWelcomeEmail sends a welcome email when a new tenant is created (SaaS)
func (s *EmailService) SendTenantWelcomeEmail(to, tenantName, adminName, tenantSlug, loginURL string) error {
	subject := fmt.Sprintf("Willkommen bei Gassigeher - %s ist bereit!", tenantName)

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #26272b; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #82b965; color: white; padding: 20px; text-align: center; border-radius: 6px 6px 0 0; }
        .content { background-color: #f9f9f9; padding: 30px; border-radius: 0 0 6px 6px; }
        .welcome-box { background-color: white; padding: 20px; margin: 20px 0; border-radius: 6px; border-left: 4px solid #82b965; }
        .cta-button { display: inline-block; background-color: #82b965; color: white; padding: 15px 30px; text-decoration: none; border-radius: 6px; font-weight: bold; margin: 20px 0; }
        .cta-button:hover { background-color: #6fa050; }
        .steps-list { background-color: white; padding: 20px; margin: 20px 0; border-radius: 6px; }
        .steps-list h3 { margin-top: 0; color: #82b965; }
        .steps-list ol { margin: 0; padding-left: 20px; }
        .steps-list li { margin: 10px 0; }
        .subdomain { font-family: monospace; background-color: #e9ecef; padding: 2px 8px; border-radius: 4px; }
        .footer { text-align: center; margin-top: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Willkommen bei Gassigeher!</h1>
        </div>
        <div class="content">
            <p>Hallo {{.AdminName}},</p>

            <div class="welcome-box">
                <h3 style="margin-top: 0; color: #82b965;">{{.TenantName}} ist bereit!</h3>
                <p>Ihr Tierheim wurde erfolgreich eingerichtet und ist ab sofort unter folgender Adresse erreichbar:</p>
                <p style="text-align: center; font-size: 1.2rem;">
                    <strong><a href="{{.LoginURL}}" class="subdomain">{{.TenantSlug}}.gassigeher.org</a></strong>
                </p>
            </div>

            <div style="text-align: center;">
                <a href="{{.LoginURL}}" class="cta-button">Jetzt anmelden</a>
            </div>

            <div class="steps-list">
                <h3>Nachste Schritte</h3>
                <ol>
                    <li><strong>Anmelden:</strong> Loggen Sie sich mit Ihren Zugangsdaten ein</li>
                    <li><strong>Hunde hinzufugen:</strong> Erfassen Sie Ihre Hunde im System</li>
                    <li><strong>Team einladen:</strong> Fugen Sie weitere Administratoren hinzu</li>
                    <li><strong>Freiwillige einladen:</strong> Teilen Sie den Registrierungslink</li>
                    <li><strong>Design anpassen:</strong> Passen Sie Farben und Logo an</li>
                </ol>
            </div>

            <p style="font-size: 0.9rem; color: #666;">
                Bei Fragen oder Problemen konnen Sie uns jederzeit unter <a href="mailto:support@gassigeher.org">support@gassigeher.org</a> erreichen.
            </p>
        </div>
        <div class="footer">
            <p>Gassigeher - Die Gassi-Verwaltung fur Tierheime</p>
            <p>Diese E-Mail wurde automatisch generiert.</p>
        </div>
    </div>
</body>
</html>
`

	data := struct {
		TenantName string
		AdminName  string
		TenantSlug string
		LoginURL   string
	}{
		TenantName: tenantName,
		AdminName:  adminName,
		TenantSlug: tenantSlug,
		LoginURL:   loginURL,
	}

	t, err := template.New("tenant_welcome").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var body bytes.Buffer
	if err := t.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return s.SendEmail(to, subject, body.String())
}
