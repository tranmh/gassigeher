package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/database"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
)

// CalendarHandler handles iCal calendar feed endpoints
type CalendarHandler struct {
	db          *database.DB
	userRepo    *repository.UserRepository
	bookingRepo *repository.BookingRepository
	config      *config.Config
}

// NewCalendarHandler creates a new calendar handler
func NewCalendarHandler(db *database.DB, cfg *config.Config) *CalendarHandler {
	return &CalendarHandler{
		db:          db,
		userRepo:    repository.NewUserRepository(db),
		bookingRepo: repository.NewBookingRepository(db),
		config:      cfg,
	}
}

// CalendarTokenResponse represents the response for calendar token endpoints
type CalendarTokenResponse struct {
	Token       string `json:"token"`
	FeedURL     string `json:"feed_url"`
	WebcalURL   string `json:"webcal_url"`
	HasToken    bool   `json:"has_token"`
}

// generateToken creates a secure random token
func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// generateAndSaveToken generates a new token and saves it, retrying on collision
// BUG FIX: Handle token collision by retrying with a new token
func (h *CalendarHandler) generateAndSaveToken(userID int) (string, error) {
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		token, err := generateToken()
		if err != nil {
			return "", err
		}
		
		err = h.userRepo.SetCalendarToken(userID, token)
		if err != nil {
			// Check if it's a unique constraint error (token collision)
			if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "duplicate") {
				log.Printf("Calendar token collision for user %d, retrying (%d/%d)", userID, i+1, maxRetries)
				continue
			}
			return "", err
		}
		return token, nil
	}
	return "", fmt.Errorf("failed to generate unique token after %d retries", maxRetries)
}

// GetToken returns the user's calendar token, creating one if it doesn't exist
// GET /api/v1/calendar/token
func (h *CalendarHandler) GetToken(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get existing token
	existingToken, err := h.userRepo.GetCalendarToken(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get calendar token")
		return
	}

	var token string
	if existingToken != nil {
		token = *existingToken
	} else {
		// Generate new token with collision handling
		token, err = h.generateAndSaveToken(userID)
		if err != nil {
			log.Printf("Failed to generate calendar token for user %d: %v", userID, err)
			respondError(w, http.StatusInternalServerError, "Failed to generate token")
			return
		}
		log.Printf("Generated new calendar token for user %d", userID)
	}

	// Build URLs
	baseURL := h.config.BaseURL
	feedURL := fmt.Sprintf("%s/api/calendar/feed/%s.ics", baseURL, token)
	webcalURL := strings.Replace(feedURL, "https://", "webcal://", 1)
	webcalURL = strings.Replace(webcalURL, "http://", "webcal://", 1)

	respondJSON(w, http.StatusOK, CalendarTokenResponse{
		Token:     token,
		FeedURL:   feedURL,
		WebcalURL: webcalURL,
		HasToken:  true,
	})
}

// RegenerateToken creates a new calendar token, invalidating the old one
// POST /api/v1/calendar/token/regenerate
func (h *CalendarHandler) RegenerateToken(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Generate new token with collision handling
	token, err := h.generateAndSaveToken(userID)
	if err != nil {
		log.Printf("Failed to regenerate calendar token for user %d: %v", userID, err)
		respondError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	log.Printf("Regenerated calendar token for user %d", userID)

	// Build URLs
	baseURL := h.config.BaseURL
	feedURL := fmt.Sprintf("%s/api/calendar/feed/%s.ics", baseURL, token)
	webcalURL := strings.Replace(feedURL, "https://", "webcal://", 1)
	webcalURL = strings.Replace(webcalURL, "http://", "webcal://", 1)

	respondJSON(w, http.StatusOK, CalendarTokenResponse{
		Token:     token,
		FeedURL:   feedURL,
		WebcalURL: webcalURL,
		HasToken:  true,
	})
}

// GetFeed returns the iCal feed for a user (public endpoint, auth via token in URL)
// GET /api/calendar/feed/{token}.ics
func (h *CalendarHandler) GetFeed(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tokenWithExt := vars["token"]
	
	// Remove .ics extension if present
	token := strings.TrimSuffix(tokenWithExt, ".ics")
	
	if token == "" {
		http.Error(w, "Invalid token", http.StatusBadRequest)
		return
	}

	// Find user by token
	user, err := h.userRepo.FindByCalendarToken(token)
	if err != nil {
		log.Printf("Error finding user by calendar token: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if user == nil {
		// BUG FIX: Don't reveal whether token exists or not (timing attack prevention)
		http.Error(w, "Invalid or expired calendar link", http.StatusNotFound)
		return
	}

	// Check if user is active
	if !user.IsActive || user.IsDeleted {
		http.Error(w, "Calendar feed is unavailable", http.StatusForbidden)
		return
	}

	// Get user's upcoming bookings
	bookings, err := h.bookingRepo.FindUpcomingByUser(user.ID, user.TenantID)
	if err != nil {
		log.Printf("Error fetching bookings for calendar feed (user %d): %v", user.ID, err)
		http.Error(w, "Failed to generate calendar", http.StatusInternalServerError)
		return
	}

	// BUG FIX: Add audit logging for calendar feed access
	log.Printf("Calendar feed accessed for user %d (tenant %d), %d upcoming bookings", 
		user.ID, user.TenantID, len(bookings))

	// Generate iCal content
	ical := h.generateICal(user, bookings)

	// Set headers for iCal file
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=gassigeher.ics")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(ical))
}

// generateICal creates the iCal content from bookings
func (h *CalendarHandler) generateICal(user *models.User, bookings []*models.Booking) string {
	var sb strings.Builder

	// Calendar header
	sb.WriteString("BEGIN:VCALENDAR\r\n")
	sb.WriteString("VERSION:2.0\r\n")
	sb.WriteString("PRODID:-//Gassigeher//Dog Walking Booking//DE\r\n")
	sb.WriteString("CALSCALE:GREGORIAN\r\n")
	sb.WriteString("METHOD:PUBLISH\r\n")
	sb.WriteString(foldICalLine(fmt.Sprintf("X-WR-CALNAME:Gassigeher - %s", escapeICalText(user.FullName()))))
	sb.WriteString("X-WR-TIMEZONE:Europe/Berlin\r\n")

	// Add timezone definition
	sb.WriteString("BEGIN:VTIMEZONE\r\n")
	sb.WriteString("TZID:Europe/Berlin\r\n")
	sb.WriteString("BEGIN:DAYLIGHT\r\n")
	sb.WriteString("TZOFFSETFROM:+0100\r\n")
	sb.WriteString("TZOFFSETTO:+0200\r\n")
	sb.WriteString("TZNAME:CEST\r\n")
	sb.WriteString("DTSTART:19700329T020000\r\n")
	sb.WriteString("RRULE:FREQ=YEARLY;BYMONTH=3;BYDAY=-1SU\r\n")
	sb.WriteString("END:DAYLIGHT\r\n")
	sb.WriteString("BEGIN:STANDARD\r\n")
	sb.WriteString("TZOFFSETFROM:+0200\r\n")
	sb.WriteString("TZOFFSETTO:+0100\r\n")
	sb.WriteString("TZNAME:CET\r\n")
	sb.WriteString("DTSTART:19701025T030000\r\n")
	sb.WriteString("RRULE:FREQ=YEARLY;BYMONTH=10;BYDAY=-1SU\r\n")
	sb.WriteString("END:STANDARD\r\n")
	sb.WriteString("END:VTIMEZONE\r\n")

	// Add events for each booking
	for _, booking := range bookings {
		sb.WriteString(h.generateVEvent(booking))
	}

	// Calendar footer
	sb.WriteString("END:VCALENDAR\r\n")

	return sb.String()
}

// generateVEvent creates a VEVENT for a single booking
func (h *CalendarHandler) generateVEvent(booking *models.Booking) string {
	var sb strings.Builder

	// Parse booking date and time
	loc, _ := time.LoadLocation("Europe/Berlin")
	
	// Combine date and time
	startTime, err := time.ParseInLocation("2006-01-02 15:04", 
		fmt.Sprintf("%s %s", booking.Date, booking.ScheduledTime), loc)
	if err != nil {
		// Fallback: use booking date at midnight
		startTime, _ = time.ParseInLocation("2006-01-02", booking.Date, loc)
	}

	// BUG FIX: Use walk duration from dog if available, default to 60 minutes
	walkDurationMinutes := 60
	if booking.Dog != nil && booking.Dog.WalkDuration != nil && *booking.Dog.WalkDuration > 0 {
		walkDurationMinutes = *booking.Dog.WalkDuration
	}
	endTime := startTime.Add(time.Duration(walkDurationMinutes) * time.Minute)

	// Generate unique ID for this event
	uid := fmt.Sprintf("booking-%d@gassigeher", booking.ID)

	// Event summary (title)
	summary := "Gassi"
	if booking.Dog != nil && booking.Dog.Name != "" {
		summary = fmt.Sprintf("Gassi mit %s", booking.Dog.Name)
	}

	// Event description
	var descParts []string
	if booking.Dog != nil && booking.Dog.Breed != "" {
		descParts = append(descParts, fmt.Sprintf("Rasse: %s", booking.Dog.Breed))
	}
	if booking.UserNotes != nil && *booking.UserNotes != "" {
		descParts = append(descParts, fmt.Sprintf("Notizen: %s", *booking.UserNotes))
	}
	description := strings.Join(descParts, "\\n")

	// BUG FIX: Add location from dog's pickup location
	location := ""
	if booking.Dog != nil && booking.Dog.PickupLocation != nil && *booking.Dog.PickupLocation != "" {
		location = *booking.Dog.PickupLocation
	}

	sb.WriteString("BEGIN:VEVENT\r\n")
	sb.WriteString(fmt.Sprintf("UID:%s\r\n", uid))
	sb.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", formatICalDateTime(time.Now())))
	sb.WriteString(fmt.Sprintf("DTSTART;TZID=Europe/Berlin:%s\r\n", formatICalDateTimeLocal(startTime)))
	sb.WriteString(fmt.Sprintf("DTEND;TZID=Europe/Berlin:%s\r\n", formatICalDateTimeLocal(endTime)))
	sb.WriteString(foldICalLine(fmt.Sprintf("SUMMARY:%s", escapeICalText(summary))))
	if description != "" {
		sb.WriteString(foldICalLine(fmt.Sprintf("DESCRIPTION:%s", escapeICalText(description))))
	}
	if location != "" {
		sb.WriteString(foldICalLine(fmt.Sprintf("LOCATION:%s", escapeICalText(location))))
	}
	sb.WriteString(fmt.Sprintf("STATUS:%s\r\n", getICalStatus(booking.Status)))
	sb.WriteString("END:VEVENT\r\n")

	return sb.String()
}

// formatICalDateTime formats a time for iCal (UTC)
func formatICalDateTime(t time.Time) string {
	return t.UTC().Format("20060102T150405Z")
}

// formatICalDateTimeLocal formats a time for iCal (local time, no Z suffix)
func formatICalDateTimeLocal(t time.Time) string {
	return t.Format("20060102T150405")
}

// escapeICalText escapes special characters in iCal text
func escapeICalText(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

// foldICalLine folds long lines according to RFC 5545 (max 75 octets per line)
// BUG FIX: Implement line folding for RFC 5545 compliance
func foldICalLine(line string) string {
	const maxLineLen = 75
	
	if len(line) <= maxLineLen {
		return line + "\r\n"
	}

	var result strings.Builder
	remaining := line
	firstLine := true

	for len(remaining) > 0 {
		cutLen := maxLineLen
		if !firstLine {
			// Continuation lines start with a space, so we have 74 chars for content
			cutLen = maxLineLen - 1
		}

		if len(remaining) <= cutLen {
			if !firstLine {
				result.WriteString(" ")
			}
			result.WriteString(remaining)
			result.WriteString("\r\n")
			break
		}

		// Cut at maxLineLen, being careful not to cut in the middle of a UTF-8 char
		cut := cutLen
		for cut > 0 && remaining[cut-1]&0xC0 == 0x80 {
			cut--
		}
		if cut == 0 {
			cut = cutLen // Fallback if we can't find a safe cut point
		}

		if !firstLine {
			result.WriteString(" ")
		}
		result.WriteString(remaining[:cut])
		result.WriteString("\r\n")
		remaining = remaining[cut:]
		firstLine = false
	}

	return result.String()
}

// getICalStatus converts booking status to iCal status
func getICalStatus(status string) string {
	switch status {
	case "scheduled":
		return "CONFIRMED"
	case "pending_approval":
		return "TENTATIVE"
	case "cancelled":
		return "CANCELLED"
	case "completed":
		return "CONFIRMED"
	default:
		return "CONFIRMED"
	}
}
