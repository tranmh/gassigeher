package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
	"github.com/tranmh/gassigeher/internal/services"
)

// BlockedDateHandler handles blocked date-related HTTP requests
type BlockedDateHandler struct {
	db              *sql.DB
	cfg             *config.Config
	blockedDateRepo *repository.BlockedDateRepository
	bookingRepo     *repository.BookingRepository
	userRepo        *repository.UserRepository
	dogRepo         *repository.DogRepository
	emailService    *services.EmailService
}

// NewBlockedDateHandler creates a new blocked date handler
func NewBlockedDateHandler(db *sql.DB, cfg *config.Config) *BlockedDateHandler {
	// Initialize email service (fail gracefully if email not configured)
	emailService, err := services.NewEmailService(services.ConfigToEmailConfig(cfg))
	if err != nil {
		fmt.Printf("Warning: Failed to initialize email service in BlockedDateHandler: %v\n", err)
	}

	return &BlockedDateHandler{
		db:              db,
		cfg:             cfg,
		blockedDateRepo: repository.NewBlockedDateRepository(db),
		bookingRepo:     repository.NewBookingRepository(db),
		userRepo:        repository.NewUserRepository(db),
		dogRepo:         repository.NewDogRepository(db),
		emailService:    emailService,
	}
}

// ListBlockedDates lists all blocked dates
func (h *BlockedDateHandler) ListBlockedDates(w http.ResponseWriter, r *http.Request) {
	blockedDates, err := h.blockedDateRepo.FindAll()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get blocked dates")
		return
	}

	respondJSON(w, http.StatusOK, blockedDates)
}

// CreateBlockedDate creates a new blocked date (admin only)
func (h *BlockedDateHandler) CreateBlockedDate(w http.ResponseWriter, r *http.Request) {
	// Get admin user ID from context
	userID, _ := r.Context().Value(middleware.UserIDKey).(int)

	// Parse request
	var req models.CreateBlockedDateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Create blocked date
	blockedDate := &models.BlockedDate{
		Date:      req.Date,
		Reason:    req.Reason,
		CreatedBy: userID,
	}

	if err := h.blockedDateRepo.Create(blockedDate); err != nil {
		if err.Error() == "date is already blocked" {
			respondError(w, http.StatusConflict, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to create blocked date")
		return
	}

	// Find all scheduled bookings on this date
	status := "scheduled"
	filter := &models.BookingFilterRequest{
		DateFrom: &req.Date,
		DateTo:   &req.Date,
		Status:   &status,
	}
	bookings, err := h.bookingRepo.FindAll(filter)
	if err != nil {
		fmt.Printf("Warning: Failed to find bookings for date %s: %v\n", req.Date, err)
		// Continue even if we can't find bookings - at least the date is blocked
	}

	// Cancel each booking and notify users
	cancelledCount := 0
	cancellationReason := fmt.Sprintf("Datum wurde durch Administration gesperrt: %s", req.Reason)

	for _, booking := range bookings {
		// Cancel the booking
		if err := h.bookingRepo.Cancel(booking.ID, &cancellationReason); err != nil {
			fmt.Printf("Warning: Failed to cancel booking %d: %v\n", booking.ID, err)
			continue
		}
		cancelledCount++

		// Get user details for email
		user, err := h.userRepo.FindByID(booking.UserID)
		if err != nil {
			fmt.Printf("Warning: Failed to get user %d for cancellation email: %v\n", booking.UserID, err)
			continue
		}

		// Get dog details for email
		dog, err := h.dogRepo.FindByID(booking.DogID)
		if err != nil {
			fmt.Printf("Warning: Failed to get dog %d for cancellation email: %v\n", booking.DogID, err)
			continue
		}

		// Send cancellation email (in goroutine, don't block)
		if h.emailService != nil && user.Email != nil {
			go func(userEmail, userName, dogName, date, scheduledTime, reason string) {
				if err := h.emailService.SendAdminCancellation(userEmail, userName, dogName, date, scheduledTime, reason); err != nil {
					fmt.Printf("Warning: Failed to send cancellation email to %s: %v\n", userEmail, err)
				}
			}(*user.Email, user.Name, dog.Name, booking.Date, booking.ScheduledTime, cancellationReason)
		}
	}

	// Return response with cancellation count
	response := map[string]interface{}{
		"blocked_date":      blockedDate,
		"cancelled_bookings": cancelledCount,
	}

	respondJSON(w, http.StatusCreated, response)
}

// DeleteBlockedDate deletes a blocked date (admin only)
func (h *BlockedDateHandler) DeleteBlockedDate(w http.ResponseWriter, r *http.Request) {
	// Get ID from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid blocked date ID")
		return
	}

	// Delete blocked date
	if err := h.blockedDateRepo.Delete(id); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete blocked date")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Blocked date deleted successfully"})
}
