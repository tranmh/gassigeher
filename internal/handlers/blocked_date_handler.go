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
	// SaaS: Extract tenant ID from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	blockedDates, err := h.blockedDateRepo.FindAll(tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get blocked dates")
		return
	}

	respondJSON(w, http.StatusOK, blockedDates)
}

// CreateBlockedDate creates a new blocked date (admin only)
// Now supports optional dog_id for dog-specific blocking
func (h *BlockedDateHandler) CreateBlockedDate(w http.ResponseWriter, r *http.Request) {
	// Get admin user ID from context
	userID, _ := r.Context().Value(middleware.UserIDKey).(int)
	// SaaS: Extract tenant ID from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

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

	// If dog_id is provided, verify the dog exists and belongs to tenant
	var dogName string
	if req.DogID != nil {
		dog, err := h.dogRepo.FindByIDAndTenant(*req.DogID, tenantID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to verify dog")
			return
		}
		if dog == nil {
			respondError(w, http.StatusNotFound, "Dog not found")
			return
		}
		dogName = dog.Name
	}

	// Create blocked date
	blockedDate := &models.BlockedDate{
		TenantID:  tenantID, // SaaS: Set tenant ID
		Date:      req.Date,
		DogID:     req.DogID,
		Reason:    req.Reason,
		CreatedBy: userID,
	}

	if err := h.blockedDateRepo.Create(blockedDate); err != nil {
		errStr := err.Error()
		if errStr == "this dog is already blocked for this date" || errStr == "this date is already globally blocked" {
			respondError(w, http.StatusConflict, errStr)
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to create blocked date")
		return
	}

	// Find bookings to cancel based on block type
	status := "scheduled"
	filter := &models.BookingFilterRequest{
		TenantID: &tenantID, // SaaS: Filter by tenant
		DateFrom: &req.Date,
		DateTo:   &req.Date,
		Status:   &status,
	}

	if req.DogID != nil {
		// Dog-specific block: only cancel bookings for this dog
		filter.DogID = req.DogID
	}
	// For global block (req.DogID == nil): cancel ALL bookings on this date

	bookings, err := h.bookingRepo.FindAll(filter)
	if err != nil {
		fmt.Printf("Warning: Failed to find bookings for date %s: %v\n", req.Date, err)
		// Continue even if we can't find bookings - at least the date is blocked
	}

	// Cancel each booking and notify users
	cancelledCount := 0
	var cancellationReason string
	if req.DogID != nil {
		cancellationReason = fmt.Sprintf("Hund '%s' wurde für dieses Datum gesperrt: %s", dogName, req.Reason)
	} else {
		cancellationReason = fmt.Sprintf("Datum wurde durch Administration gesperrt: %s", req.Reason)
	}

	for _, booking := range bookings {
		// Cancel the booking
		if err := h.bookingRepo.Cancel(booking.ID, tenantID, &cancellationReason); err != nil {
			fmt.Printf("Warning: Failed to cancel booking %d: %v\n", booking.ID, err)
			continue
		}
		cancelledCount++

		// Get user details for email (with tenant verification for defense in depth)
		user, err := h.userRepo.FindByIDAndTenant(booking.UserID, tenantID)
		if err != nil || user == nil {
			fmt.Printf("Warning: Failed to get user %d for cancellation email: %v\n", booking.UserID, err)
			continue
		}

		// Get dog details for email (with tenant verification for defense in depth)
		dog, err := h.dogRepo.FindByIDAndTenant(booking.DogID, tenantID)
		if err != nil || dog == nil {
			fmt.Printf("Warning: Failed to get dog %d for cancellation email: %v\n", booking.DogID, err)
			continue
		}

		// Send cancellation email (in goroutine, don't block)
		if h.emailService != nil && user.Email != nil {
			go func(userEmail, userName, dogName, date, scheduledTime, reason string) {
				if err := h.emailService.SendAdminCancellation(userEmail, userName, dogName, date, scheduledTime, reason); err != nil {
					fmt.Printf("Warning: Failed to send cancellation email to %s: %v\n", userEmail, err)
				}
			}(*user.Email, user.FirstName, dog.Name, booking.Date, booking.ScheduledTime, cancellationReason)
		}
	}

	// Set dog name in response if dog-specific
	if req.DogID != nil {
		blockedDate.DogName = &dogName
	}

	// Return response with cancellation count
	response := map[string]interface{}{
		"blocked_date":       blockedDate,
		"cancelled_bookings": cancelledCount,
	}

	respondJSON(w, http.StatusCreated, response)
}

// DeleteBlockedDate deletes a blocked date (admin only)
// SaaS SECURITY FIX: Now verifies tenant ownership before deletion
func (h *BlockedDateHandler) DeleteBlockedDate(w http.ResponseWriter, r *http.Request) {
	// Get ID from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid blocked date ID")
		return
	}

	// SaaS SECURITY: Extract tenant ID and verify ownership
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	// Get blocked date to verify tenant ownership
	blockedDate, err := h.blockedDateRepo.FindByID(id, tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get blocked date")
		return
	}
	if blockedDate == nil {
		// Not found or belongs to different tenant
		respondError(w, http.StatusNotFound, "Blocked date not found")
		return
	}

	// Double-check tenant ownership for defense-in-depth
	// (tenant_id=0 is valid for Simple-Mode, so always check tenant match)
	if blockedDate.TenantID != tenantID {
		respondError(w, http.StatusNotFound, "Blocked date not found")
		return
	}

	// Delete blocked date (with tenant isolation enforcement at DB level)
	if err := h.blockedDateRepo.Delete(id, tenantID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete blocked date")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Blocked date deleted successfully"})
}
