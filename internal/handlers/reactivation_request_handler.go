package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
	"github.com/tranmh/gassigeher/internal/services"
)

// ReactivationRequestHandler handles reactivation request-related HTTP requests
type ReactivationRequestHandler struct {
	db          *sql.DB
	cfg         *config.Config
	requestRepo *repository.ReactivationRequestRepository
	userRepo    *repository.UserRepository
	emailService *services.EmailService
}

// NewReactivationRequestHandler creates a new reactivation request handler
func NewReactivationRequestHandler(db *sql.DB, cfg *config.Config) *ReactivationRequestHandler {
	emailService, err := services.NewEmailService(services.ConfigToEmailConfig(cfg))
	if err != nil {
		println("Warning: Failed to initialize email service:", err.Error())
	}

	return &ReactivationRequestHandler{
		db:           db,
		cfg:          cfg,
		requestRepo:  repository.NewReactivationRequestRepository(db),
		userRepo:     repository.NewUserRepository(db),
		emailService: emailService,
	}
}

// CreateRequest creates a new reactivation request (for deactivated users)
func (h *ReactivationRequestHandler) CreateRequest(w http.ResponseWriter, r *http.Request) {
	// Parse request to get user email (since they can't be authenticated)
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Email == "" {
		respondError(w, http.StatusBadRequest, "Email is required")
		return
	}

	// Find user by email
	user, err := h.userRepo.FindByEmail(req.Email)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if user == nil {
		// Don't reveal if user exists or not for security
		respondJSON(w, http.StatusOK, map[string]string{"message": "If your account exists and is deactivated, a request has been sent"})
		return
	}

	// Check if user is actually deactivated
	if user.IsActive {
		respondJSON(w, http.StatusOK, map[string]string{"message": "Your account is already active"})
		return
	}

	// Check if user already has a pending request
	hasPending, err := h.requestRepo.HasPendingRequest(user.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check pending requests")
		return
	}
	if hasPending {
		respondJSON(w, http.StatusOK, map[string]string{"message": "You already have a pending request"})
		return
	}

	// Create request
	reactivationRequest := &models.ReactivationRequest{
		UserID: user.ID,
	}

	if err := h.requestRepo.Create(reactivationRequest); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create request")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]string{"message": "Reactivation request submitted"})
}

// ListRequests lists reactivation requests (admin sees all pending)
func (h *ReactivationRequestHandler) ListRequests(w http.ResponseWriter, r *http.Request) {
	requests, err := h.requestRepo.FindAllPending()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get requests")
		return
	}

	// Populate user details
	for _, req := range requests {
		user, err := h.userRepo.FindByID(req.UserID)
		if err == nil && user != nil {
			req.User = user
		}
	}

	respondJSON(w, http.StatusOK, requests)
}

// ApproveRequest approves a reactivation request (admin only)
func (h *ReactivationRequestHandler) ApproveRequest(w http.ResponseWriter, r *http.Request) {
	// Get request ID from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request ID")
		return
	}

	// Get admin user ID
	reviewerID, _ := r.Context().Value(middleware.UserIDKey).(int)

	// Parse request body
	var req models.ReviewReactivationRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body
		req = models.ReviewReactivationRequestRequest{}
	}

	// Get reactivation request
	reactivationRequest, err := h.requestRepo.FindByID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get request")
		return
	}
	if reactivationRequest == nil {
		respondError(w, http.StatusNotFound, "Request not found")
		return
	}

	// Check if already reviewed
	if reactivationRequest.Status != "pending" {
		respondError(w, http.StatusBadRequest, "Request has already been reviewed")
		return
	}

	// Get user
	user, err := h.userRepo.FindByID(reactivationRequest.UserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}
	if user == nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Approve request
	if err := h.requestRepo.Approve(id, reviewerID, req.Message); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to approve request")
		return
	}

	// Activate user
	if err := h.userRepo.Activate(reactivationRequest.UserID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to activate user")
		return
	}

	// Send email notification
	if user.Email != nil && h.emailService != nil {
		go h.emailService.SendAccountReactivated(*user.Email, user.FirstName, req.Message)
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Request approved and user reactivated"})
}

// DenyRequest denies a reactivation request (admin only)
func (h *ReactivationRequestHandler) DenyRequest(w http.ResponseWriter, r *http.Request) {
	// Get request ID from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request ID")
		return
	}

	// Get admin user ID
	reviewerID, _ := r.Context().Value(middleware.UserIDKey).(int)

	// Parse request body
	var req models.ReviewReactivationRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body
		req = models.ReviewReactivationRequestRequest{}
	}

	// Get reactivation request
	reactivationRequest, err := h.requestRepo.FindByID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get request")
		return
	}
	if reactivationRequest == nil {
		respondError(w, http.StatusNotFound, "Request not found")
		return
	}

	// Check if already reviewed
	if reactivationRequest.Status != "pending" {
		respondError(w, http.StatusBadRequest, "Request has already been reviewed")
		return
	}

	// Get user
	user, err := h.userRepo.FindByID(reactivationRequest.UserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}
	if user == nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Deny request
	if err := h.requestRepo.Deny(id, reviewerID, req.Message); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to deny request")
		return
	}

	// Send email notification
	if user.Email != nil && h.emailService != nil {
		go h.emailService.SendReactivationDenied(*user.Email, user.FirstName, req.Message)
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Request denied"})
}
