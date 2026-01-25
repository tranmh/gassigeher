package services

import (
	"bytes"
	"fmt"
	"html/template"
)

// SendAccountDeactivated sends an email when account is deactivated
func (s *EmailService) SendAccountDeactivated(to, name, reason string) error {
	subject := "Ihr Konto wurde deaktiviert - Gassigeher"

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
        .warning-box { background-color: #fff3cd; padding: 20px; margin: 20px 0; border-radius: 6px; border-left: 4px solid #ffc107; }
        .info-box { background-color: white; padding: 15px; margin: 20px 0; border-radius: 6px; border-left: 4px solid #17a2b8; }
        .footer { text-align: center; margin-top: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Ihr Konto wurde deaktiviert</h1>
        </div>
        <div class="content">
            <p>Hallo {{.Name}},</p>
            <p>Ihr Konto wurde deaktiviert und Sie können sich derzeit nicht anmelden.</p>

            <div class="warning-box">
                <strong>Grund der Deaktivierung:</strong><br>
                {{.Reason}}
            </div>

            <div class="info-box">
                <h4 style="margin-top: 0;">Wie kann ich mein Konto reaktivieren?</h4>
                <p>Wenn Sie Ihr Konto reaktivieren möchten, können Sie eine Reaktivierungsanfrage über die <a href="{{.BaseURL}}/login.html" style="color: #82b965; text-decoration: underline;">Anmeldeseite</a> stellen. Ein Administrator wird Ihre Anfrage prüfen.</p>
            </div>

            <p style="text-align: center; margin-top: 20px;">
                <a href="{{.BaseURL}}/login.html" style="display: inline-block; padding: 12px 30px; background-color: #82b965; color: white; text-decoration: none; border-radius: 6px;">Zur Anmeldung</a>
            </p>

            <p>Bei Fragen wenden Sie sich bitte an unseren Support.</p>
        </div>
        <div class="footer">
            <p>© 2025 Gassigeher. Alle Rechte vorbehalten.</p>
        </div>
    </div>
</body>
</html>
`

	t := template.Must(template.New("deactivated").Parse(tmpl))
	var body bytes.Buffer
	data := map[string]string{
		"Name":    name,
		"Reason":  reason,
		"BaseURL": s.baseURL,
	}
	if err := t.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return s.SendEmail(to, subject, body.String())
}

// SendAccountReactivated sends an email when account is reactivated
func (s *EmailService) SendAccountReactivated(to, name string, message *string) error {
	subject := "Ihr Konto wurde wieder aktiviert - Gassigeher"

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
            <h1>Willkommen zurück!</h1>
        </div>
        <div class="content">
            <p>Hallo {{.Name}},</p>
            <p>Ihr Konto wurde reaktiviert und Sie können sich wieder anmelden!</p>

            <div class="success-box">
                <p style="margin: 0;">Sie können jetzt wieder:</p>
                <ul style="margin: 10px 0;">
                    <li>Hunde buchen</li>
                    <li>Ihre Buchungen verwalten</li>
                    <li>Ihr Profil bearbeiten</li>
                </ul>
            </div>

            {{if .Message}}
            <div class="message-box">
                <strong>Nachricht vom Administrator:</strong><br>
                {{.Message}}
            </div>
            {{end}}

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

	t := template.Must(template.New("reactivated").Parse(tmpl))
	var body bytes.Buffer
	data := map[string]interface{}{
		"Name":    name,
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

// SendReactivationDenied sends an email when reactivation request is denied
func (s *EmailService) SendReactivationDenied(to, name string, message *string) error {
	subject := "Ihre Reaktivierungsanfrage - Gassigeher"

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
        .info-box { background-color: #fff3cd; padding: 20px; margin: 20px 0; border-radius: 6px; border-left: 4px solid #ffc107; }
        .message-box { background-color: white; padding: 15px; margin: 20px 0; border-radius: 6px; border-left: 4px solid #17a2b8; }
        .footer { text-align: center; margin-top: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Reaktivierungsanfrage</h1>
        </div>
        <div class="content">
            <p>Hallo {{.Name}},</p>
            <p>Vielen Dank für Ihre Reaktivierungsanfrage.</p>

            <div class="info-box">
                <p style="margin: 0;">
                    Leider können wir Ihre Anfrage derzeit nicht genehmigen. Ihr Konto bleibt deaktiviert.
                </p>
            </div>

            {{if .Message}}
            <div class="message-box">
                <strong>Nachricht vom Administrator:</strong><br>
                {{.Message}}
            </div>
            {{end}}

            <p>Bei Fragen wenden Sie sich bitte an unseren Support.</p>
        </div>
        <div class="footer">
            <p>© 2025 Gassigeher. Alle Rechte vorbehalten.</p>
        </div>
    </div>
</body>
</html>
`

	t := template.Must(template.New("denied_reactivation").Parse(tmpl))
	var body bytes.Buffer
	data := map[string]interface{}{
		"Name": name,
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

// SendAccountDeletionConfirmation sends a confirmation email after account deletion
func (s *EmailService) SendAccountDeletionConfirmation(to, name string) error {
	subject := "Ihr Konto wurde gelöscht - Gassigeher"

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
        .info-box { background-color: #fff3cd; padding: 20px; margin: 20px 0; border-radius: 6px; border-left: 4px solid #ffc107; }
        .detail-box { background-color: white; padding: 20px; margin: 20px 0; border-radius: 6px; }
        .footer { text-align: center; margin-top: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Ihr Konto wurde gelöscht</h1>
        </div>
        <div class="content">
            <p>Hallo {{.Name}},</p>
            <p>Ihre Löschungsanfrage wurde durchgeführt. Ihr Konto wurde gemäß DSGVO-Richtlinien gelöscht.</p>

            <div class="detail-box">
                <h4 style="margin-top: 0;">Was wurde gelöscht:</h4>
                <ul style="margin: 10px 0;">
                    <li>Ihre persönlichen Daten (Name, E-Mail, Telefon)</li>
                    <li>Ihr Passwort</li>
                    <li>Ihr Profilfoto</li>
                </ul>

                <h4>Was wurde anonymisiert:</h4>
                <ul style="margin: 10px 0;">
                    <li>Ihre Spaziergangshistorie (für Hundepflegeaufzeichnungen)</li>
                    <li>Ihre Notizen zu Spaziergängen</li>
                </ul>
            </div>

            <div class="info-box">
                <strong>Rechtliche Hinweise:</strong>
                <p style="margin: 10px 0;">
                    Diese E-Mail dient als rechtlicher Nachweis Ihrer Kontolöschung. Die Spaziergangshistorie wird aus legitimen Interessen der Tierpflege aufbewahrt, aber vollständig anonymisiert.
                </p>
            </div>

            <p>Vielen Dank, dass Sie Gassigeher genutzt haben.</p>
        </div>
        <div class="footer">
            <p>© 2025 Gassigeher. Alle Rechte vorbehalten.</p>
        </div>
    </div>
</body>
</html>
`

	t := template.Must(template.New("deletion").Parse(tmpl))
	var body bytes.Buffer
	data := map[string]string{
		"Name": name,
	}
	if err := t.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return s.SendEmail(to, subject, body.String())
}
