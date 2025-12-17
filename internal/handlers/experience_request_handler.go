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

// ExperienceRequestHandler handles experience request-related HTTP requests
type ExperienceRequestHandler struct {
	db            *sql.DB
	cfg           *config.Config
	requestRepo   *repository.ExperienceRequestRepository
	userRepo      *repository.UserRepository
	userColorRepo *repository.UserColorRepository
	emailService  *services.EmailService
}

// NewExperienceRequestHandler creates a new experience request handler
func NewExperienceRequestHandler(db *sql.DB, cfg *config.Config) *ExperienceRequestHandler {
	emailService, err := services.NewEmailService(services.ConfigToEmailConfig(cfg))
	if err != nil {
		// Log error but don't fail
		println("Warning: Failed to initialize email service:", err.Error())
	}

	return &ExperienceRequestHandler{
		db:            db,
		cfg:           cfg,
		requestRepo:   repository.NewExperienceRequestRepository(db),
		userRepo:      repository.NewUserRepository(db),
		userColorRepo: repository.NewUserColorRepository(db),
		emailService:  emailService,
	}
}

// CreateRequest creates a new experience level request
func (h *ExperienceRequestHandler) CreateRequest(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse request
	var req models.CreateExperienceRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Get user
	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}
	if user == nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Check if user already has this level or higher
	// Determine current level from user's assigned colors
	// Color IDs: 1=gruen, 2=gelb, 3=orange, 4=hellblau, 5=dunkelblau
	colors, err := h.userColorRepo.GetUserColors(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get user colors")
		return
	}
	currentLevel := "green"
	for _, color := range colors {
		if color.ID == 4 || color.ID == 5 {
			currentLevel = "blue"
			break
		}
		if color.ID == 2 || color.ID == 3 {
			currentLevel = "orange"
			// Don't break, continue checking for blue
		}
	}
	requestedLevel := req.RequestedLevel

	if currentLevel == "blue" {
		respondError(w, http.StatusBadRequest, "You already have the highest level")
		return
	}

	if currentLevel == "orange" && requestedLevel == "orange" {
		respondError(w, http.StatusBadRequest, "You already have this level")
		return
	}

	if currentLevel == "green" && requestedLevel == "blue" {
		respondError(w, http.StatusBadRequest, "You must first get orange level")
		return
	}

	// Check if user already has a pending request for this level
	hasPending, err := h.requestRepo.HasPendingRequest(userID, requestedLevel)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check pending requests")
		return
	}
	if hasPending {
		respondError(w, http.StatusConflict, "You already have a pending request for this level")
		return
	}

	// Create request
	experienceRequest := &models.ExperienceRequest{
		UserID:         userID,
		RequestedLevel: requestedLevel,
	}

	if err := h.requestRepo.Create(experienceRequest); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create request")
		return
	}

	respondJSON(w, http.StatusCreated, experienceRequest)
}

// ListRequests lists experience requests (user sees own, admin sees all pending)
func (h *ExperienceRequestHandler) ListRequests(w http.ResponseWriter, r *http.Request) {
	// Get user ID and admin status from context
	userID, _ := r.Context().Value(middleware.UserIDKey).(int)
	isAdmin, _ := r.Context().Value(middleware.IsAdminKey).(bool)

	var requests []*models.ExperienceRequest
	var err error

	if isAdmin {
		// Admin sees all pending requests
		requests, err = h.requestRepo.FindAllPending()
	} else {
		// User sees their own requests
		requests, err = h.requestRepo.FindByUserID(userID)
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get requests")
		return
	}

	// If admin, populate user details
	if isAdmin {
		for _, req := range requests {
			user, err := h.userRepo.FindByID(req.UserID)
			if err == nil && user != nil {
				req.User = user
			}
		}
	}

	respondJSON(w, http.StatusOK, requests)
}

// ApproveRequest approves an experience request (admin only)
func (h *ExperienceRequestHandler) ApproveRequest(w http.ResponseWriter, r *http.Request) {
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
	var req models.ReviewExperienceRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body
		req = models.ReviewExperienceRequestRequest{}
	}

	// Get experience request
	experienceRequest, err := h.requestRepo.FindByID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get request")
		return
	}
	if experienceRequest == nil {
		respondError(w, http.StatusNotFound, "Request not found")
		return
	}

	// Check if already reviewed
	if experienceRequest.Status != "pending" {
		respondError(w, http.StatusBadRequest, "Request has already been reviewed")
		return
	}

	// Get user
	user, err := h.userRepo.FindByID(experienceRequest.UserID)
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

	// Assign colors based on the requested experience level
	// Color IDs: 1=gruen, 2=gelb, 3=orange, 4=hellblau, 5=dunkelblau
	colorsByLevel := map[string][]int{
		"green":  {1},             // only gruen
		"orange": {1, 2, 3},       // gruen, gelb, orange
		"blue":   {1, 2, 3, 4, 5}, // all main colors
	}
	if colors, ok := colorsByLevel[experienceRequest.RequestedLevel]; ok {
		if err := h.userColorRepo.SetUserColors(user.ID, colors, reviewerID); err != nil {
			// Log but don't fail the approval
			println("Warning: Failed to assign colors to user:", err.Error())
		}
	}

	// Send email notification
	if user.Email != nil && h.emailService != nil {
		go h.emailService.SendExperienceLevelApproved(*user.Email, user.FirstName, experienceRequest.RequestedLevel, req.Message)
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Request approved"})
}

// DenyRequest denies an experience request (admin only)
func (h *ExperienceRequestHandler) DenyRequest(w http.ResponseWriter, r *http.Request) {
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
	var req models.ReviewExperienceRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body
		req = models.ReviewExperienceRequestRequest{}
	}

	// Get experience request
	experienceRequest, err := h.requestRepo.FindByID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get request")
		return
	}
	if experienceRequest == nil {
		respondError(w, http.StatusNotFound, "Request not found")
		return
	}

	// Check if already reviewed
	if experienceRequest.Status != "pending" {
		respondError(w, http.StatusBadRequest, "Request has already been reviewed")
		return
	}

	// Get user
	user, err := h.userRepo.FindByID(experienceRequest.UserID)
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
		go h.emailService.SendExperienceLevelDenied(*user.Email, user.FirstName, experienceRequest.RequestedLevel, req.Message)
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Request denied"})
}
