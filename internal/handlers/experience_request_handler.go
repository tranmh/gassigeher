package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

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
	colorRepo     *repository.ColorCategoryRepository
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
		colorRepo:     repository.NewColorCategoryRepository(db),
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

	// SaaS: Extract tenant ID from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

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

	// Get user (with tenant verification)
	user, err := h.userRepo.FindByIDAndTenant(userID, tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}
	if user == nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Check if user already has this level or higher
	// Determine current level from user's assigned colors by color name
	// Level hierarchy: green < orange < blue (blue is highest)
	colors, err := h.userColorRepo.GetUserColors(tenantID, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get user colors")
		return
	}
	currentLevel := "green"
	for _, color := range colors {
		colorNameLower := strings.ToLower(color.Name)
		if colorNameLower == "hellblau" || colorNameLower == "dunkelblau" {
			currentLevel = "blue"
			break
		}
		if colorNameLower == "gelb" || colorNameLower == "orange" {
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
	hasPending, err := h.requestRepo.HasPendingRequest(tenantID, userID, requestedLevel)
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

	if err := h.requestRepo.Create(tenantID, experienceRequest); err != nil {
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
	// SaaS: Extract tenant ID from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	var requests []*models.ExperienceRequest
	var err error

	if isAdmin {
		// Admin sees all pending requests
		requests, err = h.requestRepo.FindAllPending(tenantID)
	} else {
		// User sees their own requests
		requests, err = h.requestRepo.FindByUserID(tenantID, userID)
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get requests")
		return
	}

	// If admin, populate user details (with tenant verification for defense in depth)
	if isAdmin {
		for _, req := range requests {
			user, err := h.userRepo.FindByIDAndTenant(req.UserID, tenantID)
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
	// SaaS: Extract tenant ID from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	// Parse request body
	var req models.ReviewExperienceRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body
		req = models.ReviewExperienceRequestRequest{}
	}

	// Get experience request
	experienceRequest, err := h.requestRepo.FindByID(tenantID, id)
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

	// Get user (with tenant verification for defense in depth)
	user, err := h.userRepo.FindByIDAndTenant(experienceRequest.UserID, tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}
	if user == nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Approve request
	if err := h.requestRepo.Approve(tenantID, id, reviewerID, req.Message); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to approve request")
		return
	}

	// Assign colors based on the requested experience level
	// Look up color IDs by name (dynamically, not hardcoded)
	colorNamesByLevel := map[string][]string{
		"green":  {"gruen"},
		"orange": {"gruen", "gelb", "orange"},
		"blue":   {"gruen", "gelb", "orange", "hellblau", "dunkelblau"},
	}
	if colorNames, ok := colorNamesByLevel[experienceRequest.RequestedLevel]; ok {
		var colorIDs []int
		for _, colorName := range colorNames {
			color, err := h.colorRepo.FindByName(tenantID, colorName)
			if err == nil && color != nil {
				colorIDs = append(colorIDs, color.ID)
			}
		}
		if len(colorIDs) > 0 {
			if err := h.userColorRepo.SetUserColors(tenantID, user.ID, colorIDs, reviewerID); err != nil {
				// Log but don't fail the approval
				println("Warning: Failed to assign colors to user:", err.Error())
			}
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
	// SaaS: Extract tenant ID from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	// Parse request body
	var req models.ReviewExperienceRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body
		req = models.ReviewExperienceRequestRequest{}
	}

	// Get experience request
	experienceRequest, err := h.requestRepo.FindByID(tenantID, id)
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

	// Get user (with tenant verification for defense in depth)
	user, err := h.userRepo.FindByIDAndTenant(experienceRequest.UserID, tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}
	if user == nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Deny request
	if err := h.requestRepo.Deny(tenantID, id, reviewerID, req.Message); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to deny request")
		return
	}

	// Send email notification
	if user.Email != nil && h.emailService != nil {
		go h.emailService.SendExperienceLevelDenied(*user.Email, user.FirstName, experienceRequest.RequestedLevel, req.Message)
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Request denied"})
}
