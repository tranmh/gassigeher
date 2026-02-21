package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
)

// PreviewRecurringBooking generates a preview of dates for a recurring booking
// POST /api/v1/bookings/recurring/preview
func (h *BookingHandler) PreviewRecurringBooking(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok {
		respondError(w, http.StatusInternalServerError, "Request validation failed")
		return
	}

	var req models.RecurringBookingPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Compute end date from weeks if needed
	endDate := ""
	if req.EndDate != nil {
		endDate = *req.EndDate
	} else if req.Weeks != nil {
		startDate, _ := time.Parse("2006-01-02", req.StartDate)
		endDate = startDate.AddDate(0, 0, *req.Weeks*7-1).Format("2006-01-02")
	}

	// Check max weeks setting
	maxWeeks := 8 // default
	maxWeeksSetting, err := h.settingsRepo.Get(tenantID, "recurring_booking_max_weeks")
	if err == nil && maxWeeksSetting != nil {
		if parsed, parseErr := strconv.Atoi(maxWeeksSetting.Value); parseErr == nil && parsed > 0 {
			maxWeeks = parsed
		}
	}

	startDate, _ := time.Parse("2006-01-02", req.StartDate)
	endDateParsed, _ := time.Parse("2006-01-02", endDate)
	maxEndDate := startDate.AddDate(0, 0, maxWeeks*7)
	if endDateParsed.After(maxEndDate) {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("Recurring bookings can span at most %d weeks", maxWeeks))
		return
	}

	// Validate user can access this dog
	user, err := h.userRepo.FindByIDAndTenant(userID, tenantID)
	if err != nil || user == nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}
	if !user.IsActive {
		respondError(w, http.StatusForbidden, "Your account is deactivated")
		return
	}

	dog, err := h.dogRepo.FindByIDAndTenant(req.DogID, tenantID)
	if err != nil || dog == nil {
		respondError(w, http.StatusNotFound, "Dog not found")
		return
	}

	// Check color access
	userColorIDs, err := h.userColorRepo.GetUserColorIDs(tenantID, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check user permissions")
		return
	}
	dogColorID := 0
	if dog.ColorID != nil {
		dogColorID = *dog.ColorID
	}
	if !repository.CanUserAccessDogByColor(userColorIDs, dogColorID) {
		respondError(w, http.StatusForbidden, "Du hast nicht die erforderliche Farbkategorie für diesen Hund")
		return
	}

	// Generate dates
	dates, err := models.GenerateRecurringDates(req.RecurrenceType, req.DayOfWeek, req.IntervalDays, req.StartDate, endDate)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Validate each date
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	var plannedDates []*models.PlannedDate
	availableCount := 0
	conflictCount := 0

	for _, date := range dates {
		pd := &models.PlannedDate{Date: date}

		bookingDate, _ := time.Parse("2006-01-02", date)

		// Check if in the past
		if bookingDate.Before(today) {
			pd.Status = "conflict"
			pd.Reason = "Datum liegt in der Vergangenheit"
			conflictCount++
			plannedDates = append(plannedDates, pd)
			continue
		}

		// Check if dog is available
		if !dog.IsAvailable {
			pd.Status = "unavailable"
			pd.Reason = "Hund ist derzeit nicht verfügbar"
			conflictCount++
			plannedDates = append(plannedDates, pd)
			continue
		}

		// Check blocked dates
		isBlocked, _ := h.blockedDateRepo.IsBlockedForDog(date, req.DogID, tenantID)
		if isBlocked {
			pd.Status = "blocked"
			pd.Reason = "Datum ist gesperrt"
			conflictCount++
			plannedDates = append(plannedDates, pd)
			continue
		}

		// Validate booking time rules
		if err := h.bookingTimeService.ValidateBookingTime(r.Context(), tenantID, date, req.ScheduledTime); err != nil {
			pd.Status = "conflict"
			pd.Reason = err.Error()
			conflictCount++
			plannedDates = append(plannedDates, pd)
			continue
		}

		// Check period availability
		isAvailable, _, _, periodErr := h.bookingTimeService.CheckPeriodAvailability(
			r.Context(), tenantID, req.DogID, date, req.ScheduledTime,
		)
		if periodErr != nil {
			pd.Status = "conflict"
			pd.Reason = periodErr.Error()
			conflictCount++
			plannedDates = append(plannedDates, pd)
			continue
		}
		if !isAvailable {
			pd.Status = "conflict"
			pd.Reason = "Hund ist bereits für diese Zeit gebucht"
			conflictCount++
			plannedDates = append(plannedDates, pd)
			continue
		}

		// Check daily booking limit
		canBook, _, _, _ := h.bookingTimeService.CheckDailyBookingLimit(tenantID, req.DogID, date)
		if !canBook {
			pd.Status = "limit_reached"
			pd.Reason = "Tägliches Buchungslimit erreicht"
			conflictCount++
			plannedDates = append(plannedDates, pd)
			continue
		}

		pd.Status = "available"
		availableCount++
		plannedDates = append(plannedDates, pd)
	}

	response := &models.RecurringBookingPreviewResponse{
		DogID:          req.DogID,
		ScheduledTime:  req.ScheduledTime,
		PlannedDates:   plannedDates,
		AvailableCount: availableCount,
		ConflictCount:  conflictCount,
	}

	respondJSON(w, http.StatusOK, response)
}

// CreateRecurringBooking creates a recurring booking series with individual bookings
// POST /api/v1/bookings/recurring
func (h *BookingHandler) CreateRecurringBooking(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok {
		respondError(w, http.StatusInternalServerError, "Request validation failed")
		return
	}

	var req models.CreateRecurringBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	endDate := req.ComputeEndDate()

	// Check max weeks setting
	maxWeeks := 8
	maxWeeksSetting, err := h.settingsRepo.Get(tenantID, "recurring_booking_max_weeks")
	if err == nil && maxWeeksSetting != nil {
		if parsed, parseErr := strconv.Atoi(maxWeeksSetting.Value); parseErr == nil && parsed > 0 {
			maxWeeks = parsed
		}
	}

	startDate, _ := time.Parse("2006-01-02", req.StartDate)
	endDateParsed, _ := time.Parse("2006-01-02", endDate)
	maxEndDate := startDate.AddDate(0, 0, maxWeeks*7)
	if endDateParsed.After(maxEndDate) {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("Wiederkehrende Buchungen können maximal %d Wochen umfassen", maxWeeks))
		return
	}

	// Validate user
	user, err := h.userRepo.FindByIDAndTenant(userID, tenantID)
	if err != nil {
		if isNotFoundOrTenantError(err) {
			respondError(w, http.StatusNotFound, "User not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}
	if !user.IsActive {
		respondError(w, http.StatusForbidden, "Your account is deactivated")
		return
	}

	// Validate dog
	dog, err := h.dogRepo.FindByIDAndTenant(req.DogID, tenantID)
	if err != nil {
		if isNotFoundOrTenantError(err) {
			respondError(w, http.StatusNotFound, "Dog not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to get dog")
		return
	}
	if !dog.IsAvailable {
		respondError(w, http.StatusBadRequest, "Hund ist derzeit nicht verfügbar")
		return
	}

	// Check color access
	userColorIDs, err := h.userColorRepo.GetUserColorIDs(tenantID, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check user permissions")
		return
	}
	dogColorID := 0
	if dog.ColorID != nil {
		dogColorID = *dog.ColorID
	}
	if !repository.CanUserAccessDogByColor(userColorIDs, dogColorID) {
		respondError(w, http.StatusForbidden, "Du hast nicht die erforderliche Farbkategorie für diesen Hund")
		return
	}

	// Check per-user active series limit
	maxSeries := 3
	maxSeriesSetting, err := h.settingsRepo.Get(tenantID, "max_active_recurring_series")
	if err == nil && maxSeriesSetting != nil {
		if parsed, parseErr := strconv.Atoi(maxSeriesSetting.Value); parseErr == nil && parsed > 0 {
			maxSeries = parsed
		}
	}

	activeCount, err := h.recurringBookingRepo.CountActiveByUser(userID, tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check recurring series limit")
		return
	}
	if activeCount >= maxSeries {
		respondError(w, http.StatusConflict, fmt.Sprintf("Du hast bereits %d aktive wiederkehrende Buchungsserien. Maximum ist %d.", activeCount, maxSeries))
		return
	}

	// Generate dates
	dates, err := models.GenerateRecurringDates(req.RecurrenceType, req.DayOfWeek, req.IntervalDays, req.StartDate, endDate)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Build excluded dates set for quick lookup
	excludedSet := make(map[string]bool)
	for _, d := range req.ExcludedDates {
		excludedSet[d] = true
	}

	// Filter dates: remove excluded and validate each one
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	var validDates []string
	var skippedDates []string
	seriesRequiresApproval := false

	for _, date := range dates {
		// Skip excluded dates
		if excludedSet[date] {
			skippedDates = append(skippedDates, date)
			continue
		}

		bookingDate, _ := time.Parse("2006-01-02", date)

		// Skip past dates
		if bookingDate.Before(today) {
			skippedDates = append(skippedDates, date)
			continue
		}

		// Skip blocked dates
		isBlocked, _ := h.blockedDateRepo.IsBlockedForDog(date, req.DogID, tenantID)
		if isBlocked {
			skippedDates = append(skippedDates, date)
			continue
		}

		// Validate booking time rules
		if err := h.bookingTimeService.ValidateBookingTime(r.Context(), tenantID, date, req.ScheduledTime); err != nil {
			skippedDates = append(skippedDates, date)
			continue
		}

		// Check period availability
		isAvailable, _, _, periodErr := h.bookingTimeService.CheckPeriodAvailability(
			r.Context(), tenantID, req.DogID, date, req.ScheduledTime,
		)
		if periodErr != nil || !isAvailable {
			skippedDates = append(skippedDates, date)
			continue
		}

		// Check daily booking limit
		canBook, _, _, _ := h.bookingTimeService.CheckDailyBookingLimit(tenantID, req.DogID, date)
		if !canBook {
			skippedDates = append(skippedDates, date)
			continue
		}

		// Check if this time requires approval
		requiresApproval, _ := h.bookingTimeService.RequiresApproval(tenantID, req.ScheduledTime)
		if requiresApproval {
			seriesRequiresApproval = true
		}

		validDates = append(validDates, date)
	}

	if len(validDates) == 0 {
		respondError(w, http.StatusBadRequest, "Keine verfügbaren Termine für die gewählte Serie. Alle Termine sind entweder gesperrt, belegt oder ausgeschlossen.")
		return
	}

	// Create the recurring series record
	series := &models.RecurringBookingSeries{
		TenantID:       tenantID,
		UserID:         userID,
		DogID:          req.DogID,
		RecurrenceType: req.RecurrenceType,
		DayOfWeek:      req.DayOfWeek,
		IntervalDays:   req.IntervalDays,
		ScheduledTime:  req.ScheduledTime,
		StartDate:      req.StartDate,
		EndDate:        endDate,
	}

	if err := h.recurringBookingRepo.Create(series); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create recurring booking series")
		return
	}

	// Create individual bookings for each valid date
	var createdBookings []*models.Booking
	for _, date := range validDates {
		// Determine approval for this specific time
		requiresApproval, _ := h.bookingTimeService.RequiresApproval(tenantID, req.ScheduledTime)

		booking := &models.Booking{
			TenantID:         tenantID,
			UserID:           userID,
			DogID:            req.DogID,
			Date:             date,
			ScheduledTime:    req.ScheduledTime,
			RecurrenceID:     &series.ID,
			RequiresApproval: requiresApproval,
		}

		if requiresApproval {
			booking.ApprovalStatus = "pending"
		} else {
			booking.ApprovalStatus = "approved"
		}

		if err := h.bookingRepo.Create(booking); err != nil {
			// Skip bookings that fail due to unique constraint (race condition)
			if isUniqueConstraintError(err.Error()) {
				log.Printf("Skipping duplicate booking for recurring series %d on %s: %v", series.ID, date, err)
				continue
			}
			log.Printf("Failed to create booking for recurring series %d on %s: %v", series.ID, date, err)
			continue
		}

		createdBookings = append(createdBookings, booking)
	}

	// Update user activity
	h.userRepo.UpdateLastActivity(userID)

	// Send recurring booking confirmation email
	if user.Email != nil && h.emailService != nil {
		bookedDates := make([]string, len(createdBookings))
		for i, b := range createdBookings {
			bookedDates[i] = b.Date
		}
		go h.emailService.SendRecurringBookingConfirmation(
			*user.Email,
			user.FirstName,
			dog.Name,
			bookedDates,
			req.ScheduledTime,
			seriesRequiresApproval,
		)
	}

	// Prepare response
	series.Bookings = createdBookings
	series.TotalBookings = len(createdBookings)
	series.RemainingBookings = len(createdBookings)
	series.Dog = dog

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"series":        series,
		"created_count": len(createdBookings),
		"skipped_count": len(skippedDates),
		"skipped_dates": skippedDates,
	})
}

// GetMyRecurringSeries returns all recurring series for the current user
// GET /api/v1/bookings/recurring
func (h *BookingHandler) GetMyRecurringSeries(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok {
		respondError(w, http.StatusInternalServerError, "Request validation failed")
		return
	}

	seriesList, err := h.recurringBookingRepo.FindByUserID(userID, tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch recurring series")
		return
	}

	// Enrich with booking counts
	currentDate := time.Now().UTC().Format("2006-01-02")
	for _, s := range seriesList {
		bookings, err := h.bookingRepo.FindByRecurrenceID(s.ID, tenantID)
		if err == nil {
			s.TotalBookings = len(bookings)
			remaining := 0
			for _, b := range bookings {
				if b.Status == "scheduled" && b.Date >= currentDate {
					remaining++
				}
			}
			s.RemainingBookings = remaining
		}
	}

	respondJSON(w, http.StatusOK, seriesList)
}

// GetRecurringSeries returns details of a specific recurring series
// GET /api/v1/bookings/recurring/{id}
func (h *BookingHandler) GetRecurringSeries(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok {
		respondError(w, http.StatusInternalServerError, "Request validation failed")
		return
	}

	vars := mux.Vars(r)
	seriesID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid series ID")
		return
	}

	series, err := h.recurringBookingRepo.FindByIDAndTenant(seriesID, tenantID)
	if err != nil {
		if isNotFoundOrTenantError(err) {
			respondError(w, http.StatusNotFound, "Recurring series not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to fetch recurring series")
		return
	}

	// Check ownership (non-admin can only see their own)
	isAdmin, _ := r.Context().Value(middleware.IsAdminKey).(bool)
	if !isAdmin && series.UserID != userID {
		respondError(w, http.StatusForbidden, "Access denied")
		return
	}

	// Attach bookings
	bookings, err := h.bookingRepo.FindByRecurrenceID(seriesID, tenantID)
	if err == nil {
		series.Bookings = bookings
		series.TotalBookings = len(bookings)
		currentDate := time.Now().UTC().Format("2006-01-02")
		remaining := 0
		for _, b := range bookings {
			if b.Status == "scheduled" && b.Date >= currentDate {
				remaining++
			}
		}
		series.RemainingBookings = remaining
	}

	respondJSON(w, http.StatusOK, series)
}

// CancelRecurringSeries cancels a recurring series and its future bookings
// PUT /api/v1/bookings/recurring/{id}/cancel
func (h *BookingHandler) CancelRecurringSeries(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok {
		respondError(w, http.StatusInternalServerError, "Request validation failed")
		return
	}

	vars := mux.Vars(r)
	seriesID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid series ID")
		return
	}

	series, err := h.recurringBookingRepo.FindByIDAndTenant(seriesID, tenantID)
	if err != nil {
		if isNotFoundOrTenantError(err) {
			respondError(w, http.StatusNotFound, "Recurring series not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to fetch recurring series")
		return
	}

	// Check ownership
	if series.UserID != userID {
		respondError(w, http.StatusForbidden, "Access denied")
		return
	}

	if series.Status != "active" {
		respondError(w, http.StatusBadRequest, "Series is already cancelled or completed")
		return
	}

	// Read optional cancellation reason from request body
	reason := "Wiederkehrende Buchungsserie storniert"
	var cancelReq struct {
		Reason *string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&cancelReq); err == nil && cancelReq.Reason != nil && *cancelReq.Reason != "" {
		reason = *cancelReq.Reason
	}
	cancelledCount, err := h.bookingRepo.CancelFutureByRecurrenceID(seriesID, tenantID, reason)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to cancel future bookings")
		return
	}

	// Cancel the series itself
	if err := h.recurringBookingRepo.Cancel(seriesID, tenantID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to cancel recurring series")
		return
	}

	// Send cancellation email
	if h.emailService != nil {
		user, _ := h.userRepo.FindByIDAndTenant(userID, tenantID)
		dog, _ := h.dogRepo.FindByIDAndTenant(series.DogID, tenantID)
		if user != nil && user.Email != nil && dog != nil {
			go h.emailService.SendRecurringSeriesCancelled(
				*user.Email,
				user.FirstName,
				dog.Name,
				cancelledCount,
				reason,
			)
		}
	}

	// Update user activity
	h.userRepo.UpdateLastActivity(userID)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":         "Wiederkehrende Buchungsserie erfolgreich storniert",
		"cancelled_count": cancelledCount,
	})
}

// AdminListRecurringSeries lists all recurring series for admin
// GET /api/v1/admin/bookings/recurring
func (h *BookingHandler) AdminListRecurringSeries(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok {
		respondError(w, http.StatusInternalServerError, "Request validation failed")
		return
	}

	// Parse filters
	filter := &models.RecurringBookingFilterRequest{}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = &status
	}
	if userIDStr := r.URL.Query().Get("user_id"); userIDStr != "" {
		if uid, err := strconv.Atoi(userIDStr); err == nil {
			filter.UserID = &uid
		}
	}
	if dogIDStr := r.URL.Query().Get("dog_id"); dogIDStr != "" {
		if did, err := strconv.Atoi(dogIDStr); err == nil {
			filter.DogID = &did
		}
	}

	seriesList, err := h.recurringBookingRepo.FindAll(tenantID, filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch recurring series")
		return
	}

	// Enrich with booking counts and user info
	currentDate := time.Now().UTC().Format("2006-01-02")
	for _, s := range seriesList {
		bookings, err := h.bookingRepo.FindByRecurrenceID(s.ID, tenantID)
		if err == nil {
			s.TotalBookings = len(bookings)
			remaining := 0
			for _, b := range bookings {
				if b.Status == "scheduled" && b.Date >= currentDate {
					remaining++
				}
			}
			s.RemainingBookings = remaining

			// Extract cancellation reason from bookings for cancelled/rejected series
			if s.Status == "cancelled" {
				for _, b := range bookings {
					if b.AdminCancellationReason != nil && *b.AdminCancellationReason != "" {
						s.CancellationReason = b.AdminCancellationReason
						break
					}
					if b.RejectionReason != nil && *b.RejectionReason != "" {
						s.CancellationReason = b.RejectionReason
						break
					}
				}
			}
		}

		// Attach user info
		user, err := h.userRepo.FindByIDAndTenant(s.UserID, tenantID)
		if err == nil && user != nil {
			s.User = user
		}
	}

	respondJSON(w, http.StatusOK, seriesList)
}

// AdminCancelRecurringSeries allows admin to cancel any recurring series
// PUT /api/v1/admin/bookings/recurring/{id}/cancel
func (h *BookingHandler) AdminCancelRecurringSeries(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok {
		respondError(w, http.StatusInternalServerError, "Request validation failed")
		return
	}

	vars := mux.Vars(r)
	seriesID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid series ID")
		return
	}

	// Parse reason
	var reqBody struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if reqBody.Reason == "" {
		respondError(w, http.StatusBadRequest, "Reason is required for admin cancellation")
		return
	}

	series, err := h.recurringBookingRepo.FindByIDAndTenant(seriesID, tenantID)
	if err != nil {
		if isNotFoundOrTenantError(err) {
			respondError(w, http.StatusNotFound, "Recurring series not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to fetch recurring series")
		return
	}

	if series.Status != "active" {
		respondError(w, http.StatusBadRequest, "Series is already cancelled or completed")
		return
	}

	// Cancel future bookings
	cancelledCount, err := h.bookingRepo.CancelFutureByRecurrenceID(seriesID, tenantID, reqBody.Reason)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to cancel future bookings")
		return
	}

	// Cancel the series
	if err := h.recurringBookingRepo.Cancel(seriesID, tenantID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to cancel recurring series")
		return
	}

	// Send notification to user
	if h.emailService != nil {
		user, _ := h.userRepo.FindByIDAndTenant(series.UserID, tenantID)
		dog, _ := h.dogRepo.FindByIDAndTenant(series.DogID, tenantID)
		if user != nil && user.Email != nil && dog != nil {
			go h.emailService.SendRecurringSeriesCancelled(
				*user.Email,
				user.FirstName,
				dog.Name,
				cancelledCount,
				reqBody.Reason,
			)
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":         "Wiederkehrende Buchungsserie erfolgreich storniert",
		"cancelled_count": cancelledCount,
	})
}

// AdminApproveRecurringSeries approves all pending bookings in a series
// PUT /api/v1/admin/bookings/recurring/{id}/approve
func (h *BookingHandler) AdminApproveRecurringSeries(w http.ResponseWriter, r *http.Request) {
	adminID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok {
		respondError(w, http.StatusInternalServerError, "Request validation failed")
		return
	}

	vars := mux.Vars(r)
	seriesID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid series ID")
		return
	}

	series, err := h.recurringBookingRepo.FindByIDAndTenant(seriesID, tenantID)
	if err != nil {
		if isNotFoundOrTenantError(err) {
			respondError(w, http.StatusNotFound, "Recurring series not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to fetch recurring series")
		return
	}

	approvedCount, err := h.bookingRepo.ApproveByRecurrenceID(seriesID, tenantID, adminID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to approve bookings")
		return
	}

	// Send notification
	if h.emailService != nil && approvedCount > 0 {
		user, _ := h.userRepo.FindByIDAndTenant(series.UserID, tenantID)
		dog, _ := h.dogRepo.FindByIDAndTenant(series.DogID, tenantID)
		if user != nil && user.Email != nil && dog != nil {
			bookedDates := []string{}
			bookings, _ := h.bookingRepo.FindByRecurrenceID(seriesID, tenantID)
			for _, b := range bookings {
				if b.ApprovalStatus == "approved" && b.Status == "scheduled" {
					bookedDates = append(bookedDates, b.Date)
				}
			}
			go h.emailService.SendRecurringBookingConfirmation(
				*user.Email,
				user.FirstName,
				dog.Name,
				bookedDates,
				series.ScheduledTime,
				false, // no longer pending approval
			)
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":        "Alle Buchungen der Serie wurden genehmigt",
		"approved_count": approvedCount,
	})
}

// AdminRejectRecurringSeries rejects all pending bookings in a series
// PUT /api/v1/admin/bookings/recurring/{id}/reject
func (h *BookingHandler) AdminRejectRecurringSeries(w http.ResponseWriter, r *http.Request) {
	adminID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok {
		respondError(w, http.StatusInternalServerError, "Request validation failed")
		return
	}

	vars := mux.Vars(r)
	seriesID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid series ID")
		return
	}

	var reqBody struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if reqBody.Reason == "" {
		respondError(w, http.StatusBadRequest, "Reason is required")
		return
	}

	series, err := h.recurringBookingRepo.FindByIDAndTenant(seriesID, tenantID)
	if err != nil {
		if isNotFoundOrTenantError(err) {
			respondError(w, http.StatusNotFound, "Recurring series not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to fetch recurring series")
		return
	}

	rejectedCount, err := h.bookingRepo.RejectByRecurrenceID(seriesID, tenantID, adminID, reqBody.Reason)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to reject bookings")
		return
	}

	// Cancel the series since all bookings are rejected
	if err := h.recurringBookingRepo.Cancel(seriesID, tenantID); err != nil {
		log.Printf("Warning: Failed to cancel series %d after rejection: %v", seriesID, err)
	}

	// Send notification
	if h.emailService != nil {
		user, _ := h.userRepo.FindByIDAndTenant(series.UserID, tenantID)
		dog, _ := h.dogRepo.FindByIDAndTenant(series.DogID, tenantID)
		if user != nil && user.Email != nil && dog != nil {
			go h.emailService.SendRecurringSeriesCancelled(
				*user.Email,
				user.FirstName,
				dog.Name,
				rejectedCount,
				reqBody.Reason,
			)
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":        "Alle Buchungen der Serie wurden abgelehnt",
		"rejected_count": rejectedCount,
	})
}
