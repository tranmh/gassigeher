package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/services"
)

// Email validation regex - RFC 5322 simplified
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// isValidEmail validates email format and prevents header injection
func isValidEmail(email string) bool {
	// Check for header injection attempts (newlines)
	if strings.ContainsAny(email, "\r\n") {
		return false
	}
	// Check for multiple emails (comma/semicolon separation)
	if strings.ContainsAny(email, ",;") {
		return false
	}
	// Validate format with regex
	return emailRegex.MatchString(email)
}

// ContactHandler handles contact form submissions
type ContactHandler struct {
	cfg          *config.Config
	emailService *services.EmailService
}

// NewContactHandler creates a new contact handler
func NewContactHandler(cfg *config.Config) *ContactHandler {
	emailService, _ := services.NewEmailService(services.ConfigToEmailConfig(cfg))
	return &ContactHandler{
		cfg:          cfg,
		emailService: emailService,
	}
}

// ContactRequest represents a contact form submission
type ContactRequest struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	Subject      string `json:"subject"`
	Organization string `json:"organization"`
	Message      string `json:"message"`
}

// Validate validates the contact request
func (r *ContactRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	r.Email = strings.TrimSpace(r.Email)
	r.Subject = strings.TrimSpace(r.Subject)
	r.Organization = strings.TrimSpace(r.Organization)
	r.Message = strings.TrimSpace(r.Message)

	if r.Name == "" {
		return fmt.Errorf("Name ist erforderlich")
	}
	if len(r.Name) > 200 {
		return fmt.Errorf("Name ist zu lang")
	}
	if r.Email == "" {
		return fmt.Errorf("E-Mail ist erforderlich")
	}
	if len(r.Email) > 200 {
		return fmt.Errorf("E-Mail ist zu lang")
	}
	// Validate email format more thoroughly
	if !isValidEmail(r.Email) {
		return fmt.Errorf("Ungültige E-Mail-Adresse")
	}
	if r.Subject == "" {
		return fmt.Errorf("Betreff ist erforderlich")
	}
	if r.Message == "" {
		return fmt.Errorf("Nachricht ist erforderlich")
	}
	if len(r.Message) > 10000 {
		return fmt.Errorf("Nachricht ist zu lang (max. 10000 Zeichen)")
	}

	return nil
}

// Submit handles contact form submissions
// POST /api/contact
func (h *ContactHandler) Submit(w http.ResponseWriter, r *http.Request) {
	var req ContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Anfrage")
		return
	}

	if err := req.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Determine recipient email based on subject
	recipientEmail := h.cfg.ContactEmail
	if recipientEmail == "" {
		recipientEmail = "kontakt@gassigeher.org" // Fallback
	}

	// Map subject to German text
	subjectMap := map[string]string{
		"general":     "Allgemeine Anfrage",
		"support":     "Technischer Support",
		"sales":       "Vertrieb / Pro-Plan",
		"partnership": "Partnerschaft",
		"press":       "Presse",
		"other":       "Sonstiges",
	}
	subjectText := subjectMap[req.Subject]
	if subjectText == "" {
		subjectText = req.Subject
	}

	// Send email notification
	if h.emailService != nil {
		go h.sendContactNotification(recipientEmail, req, subjectText)
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Nachricht erfolgreich gesendet",
	})
}

// sendContactNotification sends an email notification for a contact form submission
func (h *ContactHandler) sendContactNotification(to string, req ContactRequest, subjectText string) {
	subject := fmt.Sprintf("[Gassigeher Kontakt] %s von %s", subjectText, req.Name)

	organizationInfo := ""
	if req.Organization != "" {
		organizationInfo = fmt.Sprintf(`
			<tr>
				<td style="padding: 8px 12px; background: #f5f5f5; border-bottom: 1px solid #e0e0e0;"><strong>Organisation:</strong></td>
				<td style="padding: 8px 12px; border-bottom: 1px solid #e0e0e0;">%s</td>
			</tr>`, escapeHTML(req.Organization))
	}

	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #82b965; color: white; padding: 20px; text-align: center; border-radius: 6px 6px 0 0; }
        .content { background-color: #ffffff; padding: 30px; border: 1px solid #e0e0e0; border-radius: 0 0 6px 6px; }
        .info-table { width: 100%%; border-collapse: collapse; margin: 20px 0; }
        .message-box { background: #f9f9f9; border-left: 4px solid #82b965; padding: 15px; margin: 20px 0; }
        .footer { text-align: center; margin-top: 20px; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2 style="margin: 0;">Neue Kontaktanfrage</h2>
        </div>
        <div class="content">
            <p>Eine neue Nachricht wurde über das Kontaktformular eingereicht:</p>

            <table class="info-table">
                <tr>
                    <td style="padding: 8px 12px; background: #f5f5f5; border-bottom: 1px solid #e0e0e0; width: 120px;"><strong>Name:</strong></td>
                    <td style="padding: 8px 12px; border-bottom: 1px solid #e0e0e0;">%s</td>
                </tr>
                <tr>
                    <td style="padding: 8px 12px; background: #f5f5f5; border-bottom: 1px solid #e0e0e0;"><strong>E-Mail:</strong></td>
                    <td style="padding: 8px 12px; border-bottom: 1px solid #e0e0e0;"><a href="mailto:%s">%s</a></td>
                </tr>
                %s
                <tr>
                    <td style="padding: 8px 12px; background: #f5f5f5; border-bottom: 1px solid #e0e0e0;"><strong>Betreff:</strong></td>
                    <td style="padding: 8px 12px; border-bottom: 1px solid #e0e0e0;">%s</td>
                </tr>
            </table>

            <p><strong>Nachricht:</strong></p>
            <div class="message-box">
                %s
            </div>

            <p style="color: #666; font-size: 0.9rem;">
                Sie können direkt auf diese E-Mail antworten, um mit %s in Kontakt zu treten.
            </p>
        </div>
        <div class="footer">
            <p>Diese E-Mail wurde automatisch vom Gassigeher Kontaktformular generiert.</p>
        </div>
    </div>
</body>
</html>
`,
		escapeHTML(req.Name),
		escapeHTML(req.Email),
		escapeHTML(req.Email),
		organizationInfo,
		escapeHTML(subjectText),
		formatMessage(req.Message),
		escapeHTML(req.Name),
	)

	if err := h.emailService.SendEmail(to, subject, body); err != nil {
		log.Printf("ERROR: Failed to send contact notification: %v", err)
	} else {
		log.Printf("Contact form submission from %s <%s> - Subject: %s", req.Name, req.Email, subjectText)
	}
}

// escapeHTML escapes HTML special characters
func escapeHTML(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(s)
}

// formatMessage formats the message for HTML display
func formatMessage(s string) string {
	escaped := escapeHTML(s)
	// Convert newlines to <br>
	return strings.ReplaceAll(escaped, "\n", "<br>")
}
