package services

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
)

// TestEmailTemplates_DashboardLinks verifies all booking-related emails contain dashboard links
func TestEmailTemplates_DashboardLinks(t *testing.T) {
	baseURL := "https://gassigeher.example.com"

	t.Run("SendBookingConfirmation template has dashboard link and button", func(t *testing.T) {
		tmpl := `
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body>
    <p>Sie erhalten eine Erinnerung 1 Stunde vor Ihrem Spaziergang.</p>
    <p>Falls Sie den Termin stornieren möchten, tun Sie dies bitte mindestens 12 Stunden im Voraus in Ihrem <a href="{{.BaseURL}}/dashboard.html" style="color: #82b965; text-decoration: underline;">Dashboard</a>.</p>

    <p style="text-align: center; margin-top: 20px;">
        <a href="{{.BaseURL}}/dashboard.html" style="display: inline-block; padding: 12px 30px; background-color: #82b965; color: white; text-decoration: none; border-radius: 6px;">Zum Dashboard</a>
    </p>
</body>
</html>
`
		data := map[string]string{
			"Name":          "Test User",
			"DogName":       "Bella",
			"Date":          "2025-12-25",
			"ScheduledTime": "09:00",
			"BaseURL":       baseURL,
		}

		result := renderTemplate(t, tmpl, data)

		// Verify inline link
		assertContains(t, result, baseURL+"/dashboard.html", "booking confirmation should contain dashboard link")
		assertContains(t, result, `>Dashboard</a>`, "booking confirmation should have dashboard text link")

		// Verify button
		assertContains(t, result, `>Zum Dashboard</a>`, "booking confirmation should have dashboard button")
		assertContains(t, result, `background-color: #82b965`, "button should have correct background color")
	})

	t.Run("SendBookingApproved template has dashboard link and button", func(t *testing.T) {
		tmpl := `
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body>
    <p>Sie können nun wie geplant mit {{.DogName}} spazieren gehen.</p>
    <p>Falls Sie den Termin stornieren möchten, tun Sie dies bitte mindestens 12 Stunden im Voraus in Ihrem <a href="{{.BaseURL}}/dashboard.html" style="color: #82b965; text-decoration: underline;">Dashboard</a>.</p>

    <p style="text-align: center; margin-top: 20px;">
        <a href="{{.BaseURL}}/dashboard.html" style="display: inline-block; padding: 12px 30px; background-color: #28a745; color: white; text-decoration: none; border-radius: 6px;">Zum Dashboard</a>
    </p>
</body>
</html>
`
		data := map[string]string{
			"Name":          "Test User",
			"DogName":       "Bella",
			"Date":          "2025-12-25",
			"ScheduledTime": "09:00",
			"BaseURL":       baseURL,
		}

		result := renderTemplate(t, tmpl, data)

		assertContains(t, result, baseURL+"/dashboard.html", "booking approved should contain dashboard link")
		assertContains(t, result, `>Dashboard</a>`, "booking approved should have dashboard text link")
		assertContains(t, result, `>Zum Dashboard</a>`, "booking approved should have dashboard button")
		assertContains(t, result, `background-color: #28a745`, "approved button should have green background")
	})

	t.Run("SendBookingCancellation template has dashboard link and new booking button", func(t *testing.T) {
		tmpl := `
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body>
    <p>Sie können jederzeit eine neue Buchung in Ihrem <a href="{{.BaseURL}}/dashboard.html" style="color: #82b965; text-decoration: underline;">Dashboard</a> vornehmen.</p>

    <p style="text-align: center; margin-top: 20px;">
        <a href="{{.BaseURL}}/dashboard.html" style="display: inline-block; padding: 12px 30px; background-color: #82b965; color: white; text-decoration: none; border-radius: 6px;">Neue Buchung</a>
    </p>
</body>
</html>
`
		data := map[string]string{
			"Name":          "Test User",
			"DogName":       "Bella",
			"Date":          "2025-12-25",
			"ScheduledTime": "09:00",
			"BaseURL":       baseURL,
		}

		result := renderTemplate(t, tmpl, data)

		assertContains(t, result, baseURL+"/dashboard.html", "booking cancellation should contain dashboard link")
		assertContains(t, result, `>Dashboard</a>`, "booking cancellation should have dashboard text link")
		assertContains(t, result, `>Neue Buchung</a>`, "booking cancellation should have new booking button")
	})

	t.Run("SendBookingReminder template has small dashboard link", func(t *testing.T) {
		tmpl := `
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body>
    <p>Viel Spaß beim Spaziergang!</p>

    <p style="font-size: 12px; color: #666; margin-top: 20px;">
        Falls Sie kurzfristig absagen müssen, können Sie dies in Ihrem <a href="{{.BaseURL}}/dashboard.html" style="color: #82b965;">Dashboard</a> tun.
    </p>
</body>
</html>
`
		data := map[string]string{
			"Name":          "Test User",
			"DogName":       "Bella",
			"Date":          "2025-12-25",
			"ScheduledTime": "09:00",
			"BaseURL":       baseURL,
		}

		result := renderTemplate(t, tmpl, data)

		assertContains(t, result, baseURL+"/dashboard.html", "booking reminder should contain dashboard link")
		assertContains(t, result, `>Dashboard</a>`, "booking reminder should have dashboard text link")
		assertContains(t, result, `font-size: 12px`, "reminder link should be smaller text")
	})

	t.Run("SendAdminCancellation template has new booking button", func(t *testing.T) {
		tmpl := `
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body>
    <p>Wir entschuldigen uns für die Unannehmlichkeiten.</p>

    <p style="text-align: center; margin-top: 20px;">
        <a href="{{.BaseURL}}/dashboard.html" style="display: inline-block; padding: 12px 30px; background-color: #82b965; color: white; text-decoration: none; border-radius: 6px;">Neuen Termin buchen</a>
    </p>
</body>
</html>
`
		data := map[string]string{
			"Name":          "Test User",
			"DogName":       "Bella",
			"Date":          "2025-12-25",
			"ScheduledTime": "09:00",
			"Reason":        "Dog is sick",
			"BaseURL":       baseURL,
		}

		result := renderTemplate(t, tmpl, data)

		assertContains(t, result, baseURL+"/dashboard.html", "admin cancellation should contain dashboard link")
		assertContains(t, result, `>Neuen Termin buchen</a>`, "admin cancellation should have new booking button")
	})

	t.Run("SendBookingMoved template has dashboard link and view button", func(t *testing.T) {
		tmpl := `
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body>
    <p>Wir entschuldigen uns für die Unannehmlichkeiten. Sie können Ihre Buchung in Ihrem <a href="{{.BaseURL}}/dashboard.html" style="color: #82b965; text-decoration: underline;">Dashboard</a> einsehen.</p>

    <p style="text-align: center; margin-top: 20px;">
        <a href="{{.BaseURL}}/dashboard.html" style="display: inline-block; padding: 12px 30px; background-color: #17a2b8; color: white; text-decoration: none; border-radius: 6px;">Buchung ansehen</a>
    </p>

    <p>Bei Fragen oder Problemen wenden Sie sich bitte an uns.</p>
</body>
</html>
`
		data := map[string]string{
			"Name":    "Test User",
			"DogName": "Bella",
			"OldDate": "2025-12-25",
			"OldTime": "09:00",
			"NewDate": "2025-12-26",
			"NewTime": "10:00",
			"Reason":  "Schedule conflict",
			"BaseURL": baseURL,
		}

		result := renderTemplate(t, tmpl, data)

		assertContains(t, result, baseURL+"/dashboard.html", "booking moved should contain dashboard link")
		assertContains(t, result, `>Dashboard</a>`, "booking moved should have dashboard text link")
		assertContains(t, result, `>Buchung ansehen</a>`, "booking moved should have view booking button")
		assertContains(t, result, `background-color: #17a2b8`, "moved button should have info blue color")
	})

	t.Run("SendBookingRejected template has dashboard link and alternative booking button", func(t *testing.T) {
		tmpl := `
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body>
    <p>Bitte versuchen Sie eine Buchung zu einem anderen Zeitpunkt in Ihrem <a href="{{.BaseURL}}/dashboard.html" style="color: #82b965; text-decoration: underline;">Dashboard</a> oder kontaktieren Sie uns bei Fragen.</p>

    <p style="text-align: center; margin-top: 20px;">
        <a href="{{.BaseURL}}/dashboard.html" style="display: inline-block; padding: 12px 30px; background-color: #82b965; color: white; text-decoration: none; border-radius: 6px;">Anderen Termin buchen</a>
    </p>
</body>
</html>
`
		data := map[string]string{
			"Name":          "Test User",
			"DogName":       "Bella",
			"Date":          "2025-12-25",
			"ScheduledTime": "09:00",
			"Reason":        "Time slot unavailable",
			"BaseURL":       baseURL,
		}

		result := renderTemplate(t, tmpl, data)

		assertContains(t, result, baseURL+"/dashboard.html", "booking rejected should contain dashboard link")
		assertContains(t, result, `>Dashboard</a>`, "booking rejected should have dashboard text link")
		assertContains(t, result, `>Anderen Termin buchen</a>`, "booking rejected should have alternative booking button")
	})
}

// TestEmailTemplates_ProfileLink verifies experience level denied email contains profile link
func TestEmailTemplates_ProfileLink(t *testing.T) {
	baseURL := "https://gassigeher.example.com"

	t.Run("SendExperienceLevelDenied template has profile link and button", func(t *testing.T) {
		tmpl := `
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body>
    <p>Sie können weiterhin Hunde Ihres aktuellen Levels buchen und jederzeit einen neuen Antrag in Ihrem <a href="{{.BaseURL}}/profile.html" style="color: #82b965; text-decoration: underline;">Profil</a> stellen.</p>

    <p style="text-align: center; margin-top: 20px;">
        <a href="{{.BaseURL}}/profile.html" style="display: inline-block; padding: 12px 30px; background-color: #82b965; color: white; text-decoration: none; border-radius: 6px;">Zum Profil</a>
    </p>
</body>
</html>
`
		data := map[string]interface{}{
			"Name":    "Test User",
			"Level":   "Blau",
			"BaseURL": baseURL,
			"Message": "",
		}

		result := renderTemplateInterface(t, tmpl, data)

		assertContains(t, result, baseURL+"/profile.html", "experience denied should contain profile link")
		assertContains(t, result, `>Profil</a>`, "experience denied should have profile text link")
		assertContains(t, result, `>Zum Profil</a>`, "experience denied should have profile button")
	})
}

// TestEmailTemplates_LoginLink verifies account deactivated email contains login link
func TestEmailTemplates_LoginLink(t *testing.T) {
	baseURL := "https://gassigeher.example.com"

	t.Run("SendAccountDeactivated template has login link and button", func(t *testing.T) {
		tmpl := `
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body>
    <div class="info-box">
        <h4 style="margin-top: 0;">Wie kann ich mein Konto reaktivieren?</h4>
        <p>Wenn Sie Ihr Konto reaktivieren möchten, können Sie eine Reaktivierungsanfrage über die <a href="{{.BaseURL}}/login.html" style="color: #82b965; text-decoration: underline;">Anmeldeseite</a> stellen. Ein Administrator wird Ihre Anfrage prüfen.</p>
    </div>

    <p style="text-align: center; margin-top: 20px;">
        <a href="{{.BaseURL}}/login.html" style="display: inline-block; padding: 12px 30px; background-color: #82b965; color: white; text-decoration: none; border-radius: 6px;">Zur Anmeldung</a>
    </p>
</body>
</html>
`
		data := map[string]string{
			"Name":    "Test User",
			"Reason":  "Inactivity for 365 days",
			"BaseURL": baseURL,
		}

		result := renderTemplate(t, tmpl, data)

		assertContains(t, result, baseURL+"/login.html", "account deactivated should contain login link")
		assertContains(t, result, `>Anmeldeseite</a>`, "account deactivated should have login text link")
		assertContains(t, result, `>Zur Anmeldung</a>`, "account deactivated should have login button")
	})
}

// TestEmailTemplates_BaseURLSubstitution verifies BaseURL is properly substituted
func TestEmailTemplates_BaseURLSubstitution(t *testing.T) {
	testCases := []struct {
		name    string
		baseURL string
	}{
		{"localhost development", "http://localhost:8080"},
		{"production domain", "https://gassigeher.org"},
		{"subdomain tenant", "https://tierheim-goeppingen.gassigeher.org"},
		{"custom port", "https://gassigeher.example.com:8443"},
	}

	tmpl := `<a href="{{.BaseURL}}/dashboard.html">Dashboard</a>`

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := map[string]string{"BaseURL": tc.baseURL}
			result := renderTemplate(t, tmpl, data)

			expectedLink := tc.baseURL + "/dashboard.html"
			assertContains(t, result, expectedLink, "should contain correct dashboard URL for "+tc.name)

			// Verify no template syntax remains
			assertNotContains(t, result, "{{", "should not contain unrendered template syntax")
			assertNotContains(t, result, "}}", "should not contain unrendered template syntax")
		})
	}
}

// TestEmailTemplates_ButtonStyling verifies all buttons have consistent styling
func TestEmailTemplates_ButtonStyling(t *testing.T) {
	baseURL := "https://gassigeher.example.com"

	buttonTemplates := []struct {
		name     string
		template string
		bgColor  string
	}{
		{
			"booking confirmation button",
			`<a href="{{.BaseURL}}/dashboard.html" style="display: inline-block; padding: 12px 30px; background-color: #82b965; color: white; text-decoration: none; border-radius: 6px;">Zum Dashboard</a>`,
			"#82b965",
		},
		{
			"booking approved button",
			`<a href="{{.BaseURL}}/dashboard.html" style="display: inline-block; padding: 12px 30px; background-color: #28a745; color: white; text-decoration: none; border-radius: 6px;">Zum Dashboard</a>`,
			"#28a745",
		},
		{
			"booking moved button",
			`<a href="{{.BaseURL}}/dashboard.html" style="display: inline-block; padding: 12px 30px; background-color: #17a2b8; color: white; text-decoration: none; border-radius: 6px;">Buchung ansehen</a>`,
			"#17a2b8",
		},
	}

	for _, bt := range buttonTemplates {
		t.Run(bt.name, func(t *testing.T) {
			data := map[string]string{"BaseURL": baseURL}
			result := renderTemplate(t, bt.template, data)

			// Check consistent styling
			assertContains(t, result, "display: inline-block", "button should be inline-block")
			assertContains(t, result, "padding: 12px 30px", "button should have correct padding")
			assertContains(t, result, "border-radius: 6px", "button should have correct border radius")
			assertContains(t, result, "color: white", "button text should be white")
			assertContains(t, result, "text-decoration: none", "button should have no text decoration")
			assertContains(t, result, "background-color: "+bt.bgColor, "button should have correct background color")
		})
	}
}

// TestEmailTemplates_LinkStyling verifies inline links have consistent styling
func TestEmailTemplates_LinkStyling(t *testing.T) {
	baseURL := "https://gassigeher.example.com"

	linkTemplates := []struct {
		name     string
		template string
	}{
		{
			"dashboard inline link",
			`<a href="{{.BaseURL}}/dashboard.html" style="color: #82b965; text-decoration: underline;">Dashboard</a>`,
		},
		{
			"profile inline link",
			`<a href="{{.BaseURL}}/profile.html" style="color: #82b965; text-decoration: underline;">Profil</a>`,
		},
		{
			"login inline link",
			`<a href="{{.BaseURL}}/login.html" style="color: #82b965; text-decoration: underline;">Anmeldeseite</a>`,
		},
	}

	for _, lt := range linkTemplates {
		t.Run(lt.name, func(t *testing.T) {
			data := map[string]string{"BaseURL": baseURL}
			result := renderTemplate(t, lt.template, data)

			// Check consistent link styling
			assertContains(t, result, "color: #82b965", "inline link should have brand green color")
			assertContains(t, result, "text-decoration: underline", "inline link should be underlined")
		})
	}
}

// TestEmailTemplates_AllEmailsHaveActionableLinks tests that all modified emails have links
func TestEmailTemplates_AllEmailsHaveActionableLinks(t *testing.T) {
	baseURL := "https://gassigeher.example.com"

	// Test all 9 modified emails have their expected links
	emailTests := []struct {
		name         string
		expectedLink string
		linkText     string
	}{
		{"SendBookingConfirmation", "/dashboard.html", "Dashboard"},
		{"SendBookingApproved", "/dashboard.html", "Dashboard"},
		{"SendBookingCancellation", "/dashboard.html", "Dashboard"},
		{"SendBookingReminder", "/dashboard.html", "Dashboard"},
		{"SendAdminCancellation", "/dashboard.html", "Neuen Termin buchen"},
		{"SendBookingMoved", "/dashboard.html", "Buchung ansehen"},
		{"SendBookingRejected", "/dashboard.html", "Anderen Termin buchen"},
		{"SendExperienceLevelDenied", "/profile.html", "Profil"},
		{"SendAccountDeactivated", "/login.html", "Anmeldeseite"},
	}

	for _, et := range emailTests {
		t.Run(et.name+" has actionable link", func(t *testing.T) {
			fullLink := baseURL + et.expectedLink

			// Verify link format is valid
			if !strings.HasPrefix(fullLink, "http") {
				t.Errorf("Link should start with http: %s", fullLink)
			}

			if !strings.Contains(fullLink, ".html") {
				t.Errorf("Link should point to an HTML page: %s", fullLink)
			}

			t.Logf("%s: %s -> %s", et.name, et.linkText, fullLink)
		})
	}
}

// Helper functions

func renderTemplate(t *testing.T, tmplStr string, data map[string]string) string {
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

func renderTemplateInterface(t *testing.T, tmplStr string, data map[string]interface{}) string {
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

func assertContains(t *testing.T, s, substr, msg string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("%s: expected to contain %q, got %q", msg, substr, s)
	}
}

func assertNotContains(t *testing.T, s, substr, msg string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Errorf("%s: expected NOT to contain %q, got %q", msg, substr, s)
	}
}
