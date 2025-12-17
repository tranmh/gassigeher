package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
	"github.com/tranmh/gassigeher/internal/services"
)

// BookingHandler handles booking-related HTTP requests
type BookingHandler struct {
	db                   *sql.DB
	cfg                  *config.Config
	bookingRepo          *repository.BookingRepository
	dogRepo              *repository.DogRepository
	userRepo             *repository.UserRepository
	userColorRepo        *repository.UserColorRepository
	blockedDateRepo      *repository.BlockedDateRepository
	settingsRepo         *repository.SettingsRepository
	bookingTimeService   *services.BookingTimeService
	emailService         *services.EmailService
}

// NewBookingHandler creates a new booking handler
func NewBookingHandler(db *sql.DB, cfg *config.Config) *BookingHandler {
	emailService, err := services.NewEmailService(services.ConfigToEmailConfig(cfg))
	if err != nil {
		// Log error but don't fail - emails will fail gracefully
		fmt.Printf("Warning: Failed to initialize email service: %v\n", err)
	}

	// Initialize booking time service
	settingsRepo := repository.NewSettingsRepository(db)
	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	holidayService := services.NewHolidayService(holidayRepo, settingsRepo)
	bookingTimeService := services.NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo)

	return &BookingHandler{
		db:                   db,
		cfg:                  cfg,
		bookingRepo:          repository.NewBookingRepository(db),
		dogRepo:              repository.NewDogRepository(db),
		userRepo:             repository.NewUserRepository(db),
		userColorRepo:        repository.NewUserColorRepository(db),
		blockedDateRepo:      repository.NewBlockedDateRepository(db),
		settingsRepo:         settingsRepo,
		bookingTimeService:   bookingTimeService,
		emailService:         emailService,
	}
}

// CreateBooking creates a new booking
func (h *BookingHandler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse request
	var req models.CreateBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Get user to check experience level
	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}
	if user == nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Check if user is active
	if !user.IsActive {
		respondError(w, http.StatusForbidden, "Your account is deactivated")
		return
	}

	// Get dog
	dog, err := h.dogRepo.FindByID(req.DogID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get dog")
		return
	}
	if dog == nil {
		respondError(w, http.StatusNotFound, "Dog not found")
		return
	}

	// Check if dog is available
	if !dog.IsAvailable {
		respondError(w, http.StatusBadRequest, "Dog is currently unavailable")
		return
	}

	// Check color-based access (admins and super admins bypass this check)
	// Uses the NEW color system: user must have the dog's color assigned
	if !user.IsAdmin && !user.IsSuperAdmin {
		// Get user's assigned colors
		userColorIDs, err := h.userColorRepo.GetUserColorIDs(userID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to check user permissions")
			return
		}

		// Check if user has the dog's color
		dogColorID := 0
		if dog.ColorID != nil {
			dogColorID = *dog.ColorID
		}

		if !repository.CanUserAccessDogByColor(userColorIDs, dogColorID) {
			respondError(w, http.StatusForbidden, "Du hast nicht die erforderliche Farbkategorie für diesen Hund")
			return
		}
	}

	// BUGFIX #4: Check if date is in the past with consistent timezone handling
	// Parse date in UTC to match how dates are stored
	bookingDate, parseErr := time.Parse("2006-01-02", req.Date)
	if parseErr != nil {
		respondError(w, http.StatusBadRequest, "Invalid date format")
		return
	}

	// Compare dates in UTC, not local timezone
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	if bookingDate.Before(today) {
		respondError(w, http.StatusBadRequest, "Cannot book dates in the past")
		return
	}

	// Check booking advance limit
	advanceSetting, err := h.settingsRepo.Get("booking_advance_days")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get settings")
		return
	}
	advanceDays := 14 // default
	if advanceSetting != nil {
		advanceDays, _ = strconv.Atoi(advanceSetting.Value)
	}
	maxDate := today.AddDate(0, 0, advanceDays)
	if bookingDate.After(maxDate) {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("Cannot book more than %d days in advance", advanceDays))
		return
	}

	// Check if date is blocked for this specific dog
	isBlocked, err := h.blockedDateRepo.IsBlockedForDog(req.Date, req.DogID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check blocked dates")
		return
	}
	if isBlocked {
		respondError(w, http.StatusBadRequest, "This date is blocked for this dog")
		return
	}

	// Check for double-booking
	isDoubleBooked, err := h.bookingRepo.CheckDoubleBooking(req.DogID, req.Date, req.ScheduledTime)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check availability")
		return
	}
	if isDoubleBooked {
		respondError(w, http.StatusConflict, "This dog is already booked for this time")
		return
	}

	// Validate booking time (check if time is allowed/blocked)
	if err := h.bookingTimeService.ValidateBookingTime(req.Date, req.ScheduledTime); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Check if booking requires approval
	requiresApproval, err := h.bookingTimeService.RequiresApproval(req.ScheduledTime)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check approval requirements")
		return
	}

	// Create booking
	booking := &models.Booking{
		UserID:           userID,
		DogID:            req.DogID,
		Date:             req.Date,
		ScheduledTime:    req.ScheduledTime,
		RequiresApproval: requiresApproval,
	}

	// Set approval status based on whether approval is required
	if requiresApproval {
		booking.ApprovalStatus = "pending"
	} else {
		booking.ApprovalStatus = "approved"
	}

	if err := h.bookingRepo.Create(booking); err != nil {
		// BUGFIX #2: Detect UNIQUE constraint violation (race condition scenario)
		// SQLite returns error containing "UNIQUE constraint failed" when duplicate booking occurs
		if strings.Contains(err.Error(), "UNIQUE constraint") || strings.Contains(err.Error(), "unique constraint") {
			respondError(w, http.StatusConflict, "This dog is already booked for this time")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to create booking")
		return
	}

	// Update user last activity
	h.userRepo.UpdateLastActivity(userID)

	// Send confirmation email
	if user.Email != nil && h.emailService != nil {
		go h.emailService.SendBookingConfirmation(*user.Email, user.FirstName, dog.Name, booking.Date, booking.ScheduledTime)
	}

	respondJSON(w, http.StatusCreated, booking)
}

// ListBookings lists bookings
func (h *BookingHandler) ListBookings(w http.ResponseWriter, r *http.Request) {
	// Get user ID and admin status from context
	userID, _ := r.Context().Value(middleware.UserIDKey).(int)
	isAdmin, _ := r.Context().Value(middleware.IsAdminKey).(bool)

	// Parse query parameters
	filter := &models.BookingFilterRequest{}

	if dogIDStr := r.URL.Query().Get("dog_id"); dogIDStr != "" {
		dogID, _ := strconv.Atoi(dogIDStr)
		filter.DogID = &dogID
	}

	if dateFrom := r.URL.Query().Get("date_from"); dateFrom != "" {
		filter.DateFrom = &dateFrom
	}

	if dateTo := r.URL.Query().Get("date_to"); dateTo != "" {
		filter.DateTo = &dateTo
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = &status
	}

	// Check if this is a calendar view request (needs to see ALL bookings for availability)
	isCalendarView := r.URL.Query().Get("calendar_view") == "true"

	// Non-admins can only see their own bookings UNLESS it's for calendar availability view
	if !isAdmin && !isCalendarView {
		filter.UserID = &userID
	} else if isAdmin && r.URL.Query().Get("user_id") != "" {
		// Admin can filter by specific user
		uid, _ := strconv.Atoi(r.URL.Query().Get("user_id"))
		filter.UserID = &uid
	}

	// Get bookings
	bookings, err := h.bookingRepo.FindAll(filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get bookings")
		return
	}

	respondJSON(w, http.StatusOK, bookings)
}

// GetBooking gets a booking by ID
func (h *BookingHandler) GetBooking(w http.ResponseWriter, r *http.Request) {
	// Get booking ID from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid booking ID")
		return
	}

	// Get user ID and admin status
	userID, _ := r.Context().Value(middleware.UserIDKey).(int)
	isAdmin, _ := r.Context().Value(middleware.IsAdminKey).(bool)

	// Get booking
	booking, err := h.bookingRepo.FindByID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get booking")
		return
	}
	if booking == nil {
		respondError(w, http.StatusNotFound, "Booking not found")
		return
	}

	// Check authorization (user can only see their own bookings)
	if !isAdmin && booking.UserID != userID {
		respondError(w, http.StatusForbidden, "Access denied")
		return
	}

	respondJSON(w, http.StatusOK, booking)
}

// CancelBooking cancels a booking
func (h *BookingHandler) CancelBooking(w http.ResponseWriter, r *http.Request) {
	// Get booking ID from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid booking ID")
		return
	}

	// Get user ID and admin status
	userID, _ := r.Context().Value(middleware.UserIDKey).(int)
	isAdmin, _ := r.Context().Value(middleware.IsAdminKey).(bool)

	// Parse request
	var req models.CancelBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body
		req = models.CancelBookingRequest{}
	}

	// Get booking
	booking, err := h.bookingRepo.FindByIDWithDetails(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get booking")
		return
	}
	if booking == nil {
		respondError(w, http.StatusNotFound, "Booking not found")
		return
	}

	// Check authorization
	if !isAdmin && booking.UserID != userID {
		respondError(w, http.StatusForbidden, "Access denied")
		return
	}

	// Check if already cancelled or completed
	if booking.Status != "scheduled" {
		respondError(w, http.StatusBadRequest, "Booking is already "+booking.Status)
		return
	}

	// For non-admin users, check cancellation notice period
	if !isAdmin {
		noticeSetting, err := h.settingsRepo.Get("cancellation_notice_hours")
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to get settings")
			return
		}
		noticeHours := 12 // default
		if noticeSetting != nil {
			noticeHours, _ = strconv.Atoi(noticeSetting.Value)
		}

		// Parse booking date and time
		// booking.Date comes from DB as ISO 8601 format (e.g., "2025-11-27T00:00:00Z")
		// We need to extract just the date part (YYYY-MM-DD) and combine with scheduled time
		fmt.Printf("[CANCEL DEBUG] Raw booking.Date from DB: '%s'\n", booking.Date)
		fmt.Printf("[CANCEL DEBUG] Raw booking.ScheduledTime from DB: '%s'\n", booking.ScheduledTime)

		// Parse the ISO date first to get a time.Time object
		dateOnly, err := time.Parse(time.RFC3339, booking.Date)
		if err != nil {
			// Try simple date format as fallback
			dateOnly, err = time.Parse("2006-01-02", booking.Date)
			if err != nil {
				fmt.Printf("[CANCEL ERROR] Failed to parse booking date: %v\n", err)
				respondError(w, http.StatusInternalServerError, "Failed to parse booking date: "+err.Error())
				return
			}
		}

		// Format date as YYYY-MM-DD and combine with time
		dateStr := dateOnly.Format("2006-01-02")
		bookingDateTime := dateStr + " " + booking.ScheduledTime
		fmt.Printf("[CANCEL DEBUG] Combined datetime string: '%s'\n", bookingDateTime)

		bookingTime, err := time.Parse("2006-01-02 15:04", bookingDateTime)
		if err != nil {
			fmt.Printf("[CANCEL ERROR] Failed to parse booking datetime: %v\n", err)
			respondError(w, http.StatusInternalServerError, "Failed to parse booking date/time: "+err.Error())
			return
		}

		now := time.Now()
		hoursUntilBooking := bookingTime.Sub(now).Hours()
		fmt.Printf("[CANCEL DEBUG] Booking time: %v, Now: %v, Hours until: %.2f, Required: %d\n",
			bookingTime, now, hoursUntilBooking, noticeHours)

		if hoursUntilBooking < float64(noticeHours) {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("Buchungen müssen mindestens %d Stunden im Voraus storniert werden. Verbleibende Zeit: %.1f Stunden", noticeHours, hoursUntilBooking))
			return
		}
	}

	// Cancel booking
	if err := h.bookingRepo.Cancel(id, req.Reason); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to cancel booking")
		return
	}

	// Update user last activity
	h.userRepo.UpdateLastActivity(userID)

	// Send cancellation email
	if booking.User.Email != nil && h.emailService != nil {
		if isAdmin && req.Reason != nil {
			// Admin cancelled
			go h.emailService.SendAdminCancellation(*booking.User.Email, booking.User.FirstName, booking.Dog.Name, booking.Date, booking.ScheduledTime, *req.Reason)
		} else {
			// User cancelled
			go h.emailService.SendBookingCancellation(*booking.User.Email, booking.User.FirstName, booking.Dog.Name, booking.Date, booking.ScheduledTime)
		}
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Booking cancelled successfully"})
}

// AddNotes adds notes to a completed booking
func (h *BookingHandler) AddNotes(w http.ResponseWriter, r *http.Request) {
	// Get booking ID from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid booking ID")
		return
	}

	// Get user ID
	userID, _ := r.Context().Value(middleware.UserIDKey).(int)

	// Parse request
	var req models.AddNotesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Notes == "" {
		respondError(w, http.StatusBadRequest, "Notes cannot be empty")
		return
	}

	// Get booking
	booking, err := h.bookingRepo.FindByID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get booking")
		return
	}
	if booking == nil {
		respondError(w, http.StatusNotFound, "Booking not found")
		return
	}

	// Check authorization
	if booking.UserID != userID {
		respondError(w, http.StatusForbidden, "Access denied")
		return
	}

	// Check if booking is completed
	if booking.Status != "completed" {
		respondError(w, http.StatusBadRequest, "Can only add notes to completed bookings")
		return
	}

	// Add notes
	if err := h.bookingRepo.AddNotes(id, req.Notes); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Notes added successfully"})
}

// MoveBooking moves a booking to a new date/time (admin only)
func (h *BookingHandler) MoveBooking(w http.ResponseWriter, r *http.Request) {
	// Get booking ID from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid booking ID")
		return
	}

	// Get admin user ID
	userID, _ := r.Context().Value(middleware.UserIDKey).(int)

	// Parse request
	var req models.MoveBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Get booking with details
	booking, err := h.bookingRepo.FindByIDWithDetails(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get booking")
		return
	}
	if booking == nil {
		respondError(w, http.StatusNotFound, "Booking not found")
		return
	}

	// Check if booking can be moved (only scheduled bookings)
	if booking.Status != "scheduled" {
		respondError(w, http.StatusBadRequest, "Can only move scheduled bookings")
		return
	}

	// Store old values for email
	oldDate := booking.Date
	oldTime := booking.ScheduledTime

	// Check if new date is blocked for this dog
	isBlocked, err := h.blockedDateRepo.IsBlockedForDog(req.Date, booking.DogID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check blocked dates")
		return
	}
	if isBlocked {
		respondError(w, http.StatusBadRequest, "The new date is blocked for this dog")
		return
	}

	// Check for double-booking at new time
	isDoubleBooked, err := h.bookingRepo.CheckDoubleBooking(booking.DogID, req.Date, req.ScheduledTime)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check availability")
		return
	}
	if isDoubleBooked {
		respondError(w, http.StatusConflict, "Dog is already booked for this time")
		return
	}

	// Update booking
	booking.Date = req.Date
	booking.ScheduledTime = req.ScheduledTime

	if err := h.bookingRepo.Update(booking); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to move booking")
		return
	}

	// Update user last activity
	h.userRepo.UpdateLastActivity(userID)

	// Send email notification to user
	if booking.User.Email != nil && h.emailService != nil {
		go h.emailService.SendBookingMoved(
			*booking.User.Email,
			booking.User.FirstName,
			booking.Dog.Name,
			oldDate,
			oldTime,
			req.Date,
			req.ScheduledTime,
			req.Reason,
		)
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Booking moved successfully"})
}

// GetCalendarData gets calendar data for a specific month
func (h *BookingHandler) GetCalendarData(w http.ResponseWriter, r *http.Request) {
	// Get year and month from URL
	vars := mux.Vars(r)
	year, err := strconv.Atoi(vars["year"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid year")
		return
	}
	month, err := strconv.Atoi(vars["month"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid month")
		return
	}

	// Get user ID from context
	userID, _ := r.Context().Value(middleware.UserIDKey).(int)

	// Get bookings for the month
	filter := &models.BookingFilterRequest{
		UserID: &userID,
		Year:   &year,
		Month:  &month,
	}
	bookings, err := h.bookingRepo.FindAll(filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get bookings")
		return
	}

	// Get blocked dates
	blockedDates, err := h.blockedDateRepo.FindAll()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get blocked dates")
		return
	}

	// Build calendar response
	// Get first and last day of month
	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, -1)

	// Create a map of bookings by date
	bookingsByDate := make(map[string][]*models.Booking)
	for _, booking := range bookings {
		bookingsByDate[booking.Date] = append(bookingsByDate[booking.Date], booking)
	}

	// Create a map of global blocked dates (dog_id IS NULL)
	globalBlockedByDate := make(map[string]*models.BlockedDate)
	for _, blocked := range blockedDates {
		// Only add to global map if it's a global block (dog_id = nil)
		if blocked.DogID == nil {
			globalBlockedByDate[blocked.Date] = blocked
		}
	}

	// Build days array
	days := []*models.CalendarDay{}
	for d := firstDay; !d.After(lastDay); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		day := &models.CalendarDay{
			Date:     dateStr,
			Bookings: bookingsByDate[dateStr],
		}

		// Only mark as blocked if it's a global block (applies to all dogs)
		if blocked, ok := globalBlockedByDate[dateStr]; ok {
			day.IsBlocked = true
			day.BlockedReason = &blocked.Reason
		}

		if day.Bookings == nil {
			day.Bookings = []*models.Booking{}
		}

		days = append(days, day)
	}

	response := &models.CalendarResponse{
		Year:         year,
		Month:        month,
		Days:         days,
		BlockedDates: blockedDates, // Include all blocked dates for frontend to handle dog-specific blocks
	}

	respondJSON(w, http.StatusOK, response)
}

// GetPendingApprovals returns all bookings awaiting approval (admin only)
// GET /api/bookings/pending-approvals
func (h *BookingHandler) GetPendingApprovals(w http.ResponseWriter, r *http.Request) {
	// Check admin permission
	isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
	if !ok || !isAdmin {
		respondError(w, http.StatusForbidden, "Admin access required")
		return
	}

	bookings, err := h.bookingRepo.GetPendingApprovalBookings()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to load pending bookings")
		return
	}

	respondJSON(w, http.StatusOK, bookings)
}

// ApprovePendingBooking approves a morning walk booking
// PUT /api/bookings/:id/approve
func (h *BookingHandler) ApprovePendingBooking(w http.ResponseWriter, r *http.Request) {
	// Check admin permission
	isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
	if !ok || !isAdmin {
		respondError(w, http.StatusForbidden, "Admin access required")
		return
	}

	adminID, _ := r.Context().Value(middleware.UserIDKey).(int)

	// Extract booking ID from path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		respondError(w, http.StatusBadRequest, "Invalid booking ID")
		return
	}
	// Path is /api/bookings/{id}/approve, so ID is at index len-2
	idStr := pathParts[len(pathParts)-2]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid booking ID")
		return
	}

	if err := h.bookingRepo.ApproveBooking(id, adminID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Send email notification to user
	if h.emailService != nil {
		booking, err := h.bookingRepo.FindByIDWithDetails(id)
		if err == nil && booking != nil && booking.User != nil && booking.User.Email != nil && *booking.User.Email != "" {
			go h.emailService.SendBookingApproved(
				*booking.User.Email,
				booking.User.FirstName,
				booking.Dog.Name,
				booking.Date,
				booking.ScheduledTime,
			)
		}
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Booking approved successfully",
	})
}

// RejectPendingBooking rejects a morning walk booking
// PUT /api/bookings/:id/reject
func (h *BookingHandler) RejectPendingBooking(w http.ResponseWriter, r *http.Request) {
	// Check admin permission
	isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
	if !ok || !isAdmin {
		respondError(w, http.StatusForbidden, "Admin access required")
		return
	}

	adminID, _ := r.Context().Value(middleware.UserIDKey).(int)

	// Extract booking ID from path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		respondError(w, http.StatusBadRequest, "Invalid booking ID")
		return
	}
	// Path is /api/bookings/{id}/reject, so ID is at index len-2
	idStr := pathParts[len(pathParts)-2]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid booking ID")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Reason == "" {
		respondError(w, http.StatusBadRequest, "Rejection reason required")
		return
	}

	// Get booking details before rejecting (for email)
	booking, _ := h.bookingRepo.FindByIDWithDetails(id)

	if err := h.bookingRepo.RejectBooking(id, adminID, req.Reason); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Send email notification to user with reason
	if h.emailService != nil && booking != nil && booking.User != nil && booking.User.Email != nil && *booking.User.Email != "" {
		go h.emailService.SendBookingRejected(
			*booking.User.Email,
			booking.User.FirstName,
			booking.Dog.Name,
			booking.Date,
			booking.ScheduledTime,
			req.Reason,
		)
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Booking rejected successfully",
	})
}
