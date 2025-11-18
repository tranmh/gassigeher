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
	"github.com/tranm/gassigeher/internal/config"
	"github.com/tranm/gassigeher/internal/models"
	"github.com/tranm/gassigeher/internal/repository"
	"github.com/tranm/gassigeher/internal/services"
)

// BookingHandler handles booking-related HTTP requests
type BookingHandler struct {
	db                   *sql.DB
	cfg                  *config.Config
	bookingRepo          *repository.BookingRepository
	dogRepo              *repository.DogRepository
	userRepo             *repository.UserRepository
	blockedDateRepo      *repository.BlockedDateRepository
	settingsRepo         *repository.SettingsRepository
	emailService         *services.EmailService
}

// NewBookingHandler creates a new booking handler
func NewBookingHandler(db *sql.DB, cfg *config.Config) *BookingHandler {
	emailService, err := services.NewEmailService(
		cfg.GmailClientID,
		cfg.GmailClientSecret,
		cfg.GmailRefreshToken,
		cfg.GmailFromEmail,
	)
	if err != nil {
		// Log error but don't fail - emails will fail gracefully
		fmt.Printf("Warning: Failed to initialize email service: %v\n", err)
	}

	return &BookingHandler{
		db:                   db,
		cfg:                  cfg,
		bookingRepo:          repository.NewBookingRepository(db),
		dogRepo:              repository.NewDogRepository(db),
		userRepo:             repository.NewUserRepository(db),
		blockedDateRepo:      repository.NewBlockedDateRepository(db),
		settingsRepo:         repository.NewSettingsRepository(db),
		emailService:         emailService,
	}
}

// CreateBooking creates a new booking
func (h *BookingHandler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(int)
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

	// Check experience level access
	if !repository.CanUserAccessDog(user.ExperienceLevel, dog.Category) {
		respondError(w, http.StatusForbidden, "You don't have the required experience level for this dog")
		return
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

	// Check if date is blocked
	isBlocked, err := h.blockedDateRepo.IsBlocked(req.Date)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check blocked dates")
		return
	}
	if isBlocked {
		respondError(w, http.StatusBadRequest, "This date is blocked")
		return
	}

	// Check for double-booking
	isDoubleBooked, err := h.bookingRepo.CheckDoubleBooking(req.DogID, req.Date, req.WalkType)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check availability")
		return
	}
	if isDoubleBooked {
		respondError(w, http.StatusConflict, "This dog is already booked for this time")
		return
	}

	// Create booking
	booking := &models.Booking{
		UserID:        userID,
		DogID:         req.DogID,
		Date:          req.Date,
		WalkType:      req.WalkType,
		ScheduledTime: req.ScheduledTime,
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
	if user.Email != nil {
		go h.emailService.SendBookingConfirmation(*user.Email, user.Name, dog.Name, booking.Date, booking.WalkType, booking.ScheduledTime)
	}

	respondJSON(w, http.StatusCreated, booking)
}

// ListBookings lists bookings
func (h *BookingHandler) ListBookings(w http.ResponseWriter, r *http.Request) {
	// Get user ID and admin status from context
	userID, _ := r.Context().Value("user_id").(int)
	isAdmin, _ := r.Context().Value("is_admin").(bool)

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

	if walkType := r.URL.Query().Get("walk_type"); walkType != "" {
		filter.WalkType = &walkType
	}

	// Non-admins can only see their own bookings
	if !isAdmin {
		filter.UserID = &userID
	} else if userIDStr := r.URL.Query().Get("user_id"); userIDStr != "" {
		uid, _ := strconv.Atoi(userIDStr)
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
	userID, _ := r.Context().Value("user_id").(int)
	isAdmin, _ := r.Context().Value("is_admin").(bool)

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
	userID, _ := r.Context().Value("user_id").(int)
	isAdmin, _ := r.Context().Value("is_admin").(bool)

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
		bookingDateTime := booking.Date + " " + booking.ScheduledTime
		bookingTime, _ := time.Parse("2006-01-02 15:04", bookingDateTime)
		now := time.Now()
		hoursUntilBooking := bookingTime.Sub(now).Hours()

		if hoursUntilBooking < float64(noticeHours) {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("Bookings must be cancelled at least %d hours in advance", noticeHours))
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
	if booking.User.Email != nil {
		if isAdmin && req.Reason != nil {
			// Admin cancelled
			go h.emailService.SendAdminCancellation(*booking.User.Email, booking.User.Name, booking.Dog.Name, booking.Date, booking.WalkType, *req.Reason)
		} else {
			// User cancelled
			go h.emailService.SendBookingCancellation(*booking.User.Email, booking.User.Name, booking.Dog.Name, booking.Date, booking.WalkType)
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
	userID, _ := r.Context().Value("user_id").(int)

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
	userID, _ := r.Context().Value("user_id").(int)

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
	oldWalkType := booking.WalkType
	oldTime := booking.ScheduledTime

	// Check if new date is blocked
	isBlocked, err := h.blockedDateRepo.IsBlocked(req.Date)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check blocked dates")
		return
	}
	if isBlocked {
		respondError(w, http.StatusBadRequest, "The new date is blocked")
		return
	}

	// Check for double-booking at new time
	isDoubleBooked, err := h.bookingRepo.CheckDoubleBooking(booking.DogID, req.Date, req.WalkType)
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
	booking.WalkType = req.WalkType
	booking.ScheduledTime = req.ScheduledTime

	if err := h.bookingRepo.Update(booking); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to move booking")
		return
	}

	// Update user last activity
	h.userRepo.UpdateLastActivity(userID)

	// Send email notification to user
	if booking.User.Email != nil {
		go h.emailService.SendBookingMoved(
			*booking.User.Email,
			booking.User.Name,
			booking.Dog.Name,
			oldDate,
			oldWalkType,
			oldTime,
			req.Date,
			req.WalkType,
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
	userID, _ := r.Context().Value("user_id").(int)

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

	// Create a map of blocked dates
	blockedByDate := make(map[string]*models.BlockedDate)
	for _, blocked := range blockedDates {
		blockedByDate[blocked.Date] = blocked
	}

	// Build days array
	days := []*models.CalendarDay{}
	for d := firstDay; !d.After(lastDay); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		day := &models.CalendarDay{
			Date:     dateStr,
			Bookings: bookingsByDate[dateStr],
		}

		if blocked, ok := blockedByDate[dateStr]; ok {
			day.IsBlocked = true
			day.BlockedReason = &blocked.Reason
		}

		if day.Bookings == nil {
			day.Bookings = []*models.Booking{}
		}

		days = append(days, day)
	}

	response := &models.CalendarResponse{
		Year:  year,
		Month: month,
		Days:  days,
	}

	respondJSON(w, http.StatusOK, response)
}
