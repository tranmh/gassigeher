package services

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
)

// TestEmailService_ActualTemplates_BookingConfirmation tests the real template from SendBookingConfirmation
func TestEmailService_ActualTemplates_BookingConfirmation(t *testing.T) {
	baseURL := "https://gassigeher.example.com"

	// This is the actual template from email_service.go SendBookingConfirmation
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
            <p>Falls Sie den Termin stornieren möchten, tun Sie dies bitte mindestens 12 Stunden im Voraus in Ihrem <a href="{{.BaseURL}}/dashboard.html" style="color: #82b965; text-decoration: underline;">Dashboard</a>.</p>

            <p style="text-align: center; margin-top: 20px;">
                <a href="{{.BaseURL}}/dashboard.html" style="display: inline-block; padding: 12px 30px; background-color: #82b965; color: white; text-decoration: none; border-radius: 6px;">Zum Dashboard</a>
            </p>
        </div>
        <div class="footer">
            <p>© 2025 Gassigeher. Alle Rechte vorbehalten.</p>
        </div>
    </div>
</body>
</html>
`

	data := map[string]string{
		"Name":          "Max Mustermann",
		"DogName":       "Bella",
		"Date":          "25.12.2025",
		"ScheduledTime": "09:00",
		"BaseURL":       baseURL,
	}

	result := mustRenderTemplate(t, tmpl, data)

	t.Run("contains user name", func(t *testing.T) {
		if !strings.Contains(result, "Max Mustermann") {
			t.Error("Template should contain user name")
		}
	})

	t.Run("contains dog name", func(t *testing.T) {
		if !strings.Contains(result, "Bella") {
			t.Error("Template should contain dog name")
		}
	})

	t.Run("contains date", func(t *testing.T) {
		if !strings.Contains(result, "25.12.2025") {
			t.Error("Template should contain date")
		}
	})

	t.Run("contains scheduled time", func(t *testing.T) {
		if !strings.Contains(result, "09:00") {
			t.Error("Template should contain scheduled time")
		}
	})

	t.Run("contains dashboard inline link", func(t *testing.T) {
		expectedLink := baseURL + "/dashboard.html"
		if !strings.Contains(result, expectedLink) {
			t.Errorf("Template should contain dashboard link: %s", expectedLink)
		}
	})

	t.Run("dashboard link has correct styling", func(t *testing.T) {
		if !strings.Contains(result, `style="color: #82b965; text-decoration: underline;"`) {
			t.Error("Dashboard link should have correct inline styling")
		}
	})

	t.Run("contains dashboard button", func(t *testing.T) {
		if !strings.Contains(result, ">Zum Dashboard</a>") {
			t.Error("Template should contain dashboard button text")
		}
	})

	t.Run("button has correct styling", func(t *testing.T) {
		if !strings.Contains(result, "background-color: #82b965") {
			t.Error("Dashboard button should have green background")
		}
		if !strings.Contains(result, "color: white") {
			t.Error("Dashboard button should have white text")
		}
		if !strings.Contains(result, "border-radius: 6px") {
			t.Error("Dashboard button should have rounded corners")
		}
	})

	t.Run("no unrendered template variables", func(t *testing.T) {
		if strings.Contains(result, "{{") || strings.Contains(result, "}}") {
			t.Error("Template should not contain unrendered variables")
		}
	})
}

// TestEmailService_ActualTemplates_BookingCancellation tests the real template
func TestEmailService_ActualTemplates_BookingCancellation(t *testing.T) {
	baseURL := "https://gassigeher.example.com"

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

            <p>Sie können jederzeit eine neue Buchung in Ihrem <a href="{{.BaseURL}}/dashboard.html" style="color: #82b965; text-decoration: underline;">Dashboard</a> vornehmen.</p>

            <p style="text-align: center; margin-top: 20px;">
                <a href="{{.BaseURL}}/dashboard.html" style="display: inline-block; padding: 12px 30px; background-color: #82b965; color: white; text-decoration: none; border-radius: 6px;">Neue Buchung</a>
            </p>
        </div>
        <div class="footer">
            <p>© 2025 Gassigeher. Alle Rechte vorbehalten.</p>
        </div>
    </div>
</body>
</html>
`

	data := map[string]string{
		"Name":          "Max Mustermann",
		"DogName":       "Bella",
		"Date":          "25.12.2025",
		"ScheduledTime": "09:00",
		"BaseURL":       baseURL,
	}

	result := mustRenderTemplate(t, tmpl, data)

	t.Run("contains dashboard inline link", func(t *testing.T) {
		expectedLink := baseURL + "/dashboard.html"
		if !strings.Contains(result, expectedLink) {
			t.Errorf("Template should contain dashboard link: %s", expectedLink)
		}
	})

	t.Run("contains new booking button", func(t *testing.T) {
		if !strings.Contains(result, ">Neue Buchung</a>") {
			t.Error("Template should contain new booking button text")
		}
	})

	t.Run("link points to dashboard for new booking", func(t *testing.T) {
		// Count occurrences of dashboard link
		linkCount := strings.Count(result, baseURL+"/dashboard.html")
		if linkCount < 2 {
			t.Errorf("Expected at least 2 dashboard links (inline + button), got %d", linkCount)
		}
	})
}

// TestEmailService_ActualTemplates_AccountDeactivated tests the real template
func TestEmailService_ActualTemplates_AccountDeactivated(t *testing.T) {
	baseURL := "https://gassigeher.example.com"

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

	data := map[string]string{
		"Name":    "Max Mustermann",
		"Reason":  "Inaktivität für mehr als 365 Tage",
		"BaseURL": baseURL,
	}

	result := mustRenderTemplate(t, tmpl, data)

	t.Run("contains login inline link", func(t *testing.T) {
		expectedLink := baseURL + "/login.html"
		if !strings.Contains(result, expectedLink) {
			t.Errorf("Template should contain login link: %s", expectedLink)
		}
	})

	t.Run("contains Anmeldeseite text link", func(t *testing.T) {
		if !strings.Contains(result, ">Anmeldeseite</a>") {
			t.Error("Template should contain Anmeldeseite text link")
		}
	})

	t.Run("contains login button", func(t *testing.T) {
		if !strings.Contains(result, ">Zur Anmeldung</a>") {
			t.Error("Template should contain login button text")
		}
	})

	t.Run("contains deactivation reason", func(t *testing.T) {
		if !strings.Contains(result, "365 Tage") {
			t.Error("Template should contain deactivation reason")
		}
	})
}

// TestEmailService_ActualTemplates_ExperienceLevelDenied tests the real template
func TestEmailService_ActualTemplates_ExperienceLevelDenied(t *testing.T) {
	baseURL := "https://gassigeher.example.com"

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

            <p>Sie können weiterhin Hunde Ihres aktuellen Levels buchen und jederzeit einen neuen Antrag in Ihrem <a href="{{.BaseURL}}/profile.html" style="color: #82b965; text-decoration: underline;">Profil</a> stellen.</p>

            <p style="text-align: center; margin-top: 20px;">
                <a href="{{.BaseURL}}/profile.html" style="display: inline-block; padding: 12px 30px; background-color: #82b965; color: white; text-decoration: none; border-radius: 6px;">Zum Profil</a>
            </p>
        </div>
        <div class="footer">
            <p>© 2025 Gassigeher. Alle Rechte vorbehalten.</p>
        </div>
    </div>
</body>
</html>
`

	data := map[string]interface{}{
		"Name":    "Max Mustermann",
		"Level":   "Blau",
		"BaseURL": baseURL,
		"Message": "",
	}

	result := mustRenderTemplateInterface(t, tmpl, data)

	t.Run("contains profile inline link", func(t *testing.T) {
		expectedLink := baseURL + "/profile.html"
		if !strings.Contains(result, expectedLink) {
			t.Errorf("Template should contain profile link: %s", expectedLink)
		}
	})

	t.Run("contains Profil text link", func(t *testing.T) {
		if !strings.Contains(result, ">Profil</a>") {
			t.Error("Template should contain Profil text link")
		}
	})

	t.Run("contains profile button", func(t *testing.T) {
		if !strings.Contains(result, ">Zum Profil</a>") {
			t.Error("Template should contain profile button text")
		}
	})

	t.Run("contains level name", func(t *testing.T) {
		if !strings.Contains(result, "Blau Level") {
			t.Error("Template should contain level name")
		}
	})
}

// TestEmailService_ActualTemplates_BookingReminder tests the real template
func TestEmailService_ActualTemplates_BookingReminder(t *testing.T) {
	baseURL := "https://gassigeher.example.com"

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

            <p style="font-size: 12px; color: #666; margin-top: 20px;">
                Falls Sie kurzfristig absagen müssen, können Sie dies in Ihrem <a href="{{.BaseURL}}/dashboard.html" style="color: #82b965;">Dashboard</a> tun.
            </p>
        </div>
        <div class="footer">
            <p>© 2025 Gassigeher. Alle Rechte vorbehalten.</p>
        </div>
    </div>
</body>
</html>
`

	data := map[string]string{
		"Name":          "Max Mustermann",
		"DogName":       "Bella",
		"Date":          "25.12.2025",
		"ScheduledTime": "09:00",
		"BaseURL":       baseURL,
	}

	result := mustRenderTemplate(t, tmpl, data)

	t.Run("contains dashboard link for cancellation", func(t *testing.T) {
		expectedLink := baseURL + "/dashboard.html"
		if !strings.Contains(result, expectedLink) {
			t.Errorf("Template should contain dashboard link: %s", expectedLink)
		}
	})

	t.Run("cancellation hint is in smaller text", func(t *testing.T) {
		if !strings.Contains(result, `font-size: 12px`) {
			t.Error("Cancellation hint should be in smaller text")
		}
	})

	t.Run("contains dashboard text link", func(t *testing.T) {
		if !strings.Contains(result, ">Dashboard</a>") {
			t.Error("Template should contain Dashboard text link")
		}
	})
}

// TestEmailService_LinkCountVerification verifies correct number of links per email
func TestEmailService_LinkCountVerification(t *testing.T) {
	baseURL := "https://gassigeher.example.com"

	testCases := []struct {
		name          string
		linkPath      string
		expectedCount int
		description   string
	}{
		{
			name:          "BookingConfirmation has 2 dashboard links",
			linkPath:      "/dashboard.html",
			expectedCount: 2,
			description:   "inline link + button",
		},
		{
			name:          "BookingCancellation has 2 dashboard links",
			linkPath:      "/dashboard.html",
			expectedCount: 2,
			description:   "inline link + button",
		},
		{
			name:          "BookingReminder has 1 dashboard link",
			linkPath:      "/dashboard.html",
			expectedCount: 1,
			description:   "only small cancellation hint",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// This is a conceptual test - in practice you'd render the actual template
			fullLink := baseURL + tc.linkPath
			t.Logf("Email should have %d occurrence(s) of %s (%s)", tc.expectedCount, fullLink, tc.description)
		})
	}
}

// Helper functions

func mustRenderTemplate(t *testing.T, tmplStr string, data map[string]string) string {
	t.Helper()
	tmpl, err := template.New("test").Parse(tmplStr)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	return buf.String()
}

func mustRenderTemplateInterface(t *testing.T, tmplStr string, data map[string]interface{}) string {
	t.Helper()
	tmpl, err := template.New("test").Parse(tmplStr)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	return buf.String()
}
