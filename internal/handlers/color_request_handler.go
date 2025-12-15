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
)

// ColorRequestHandler handles color request-related HTTP requests
type ColorRequestHandler struct {
	db               *sql.DB
	cfg              *config.Config
	requestRepo      *repository.ColorRequestRepository
	colorRepo        *repository.ColorCategoryRepository
	userColorRepo    *repository.UserColorRepository
}

// NewColorRequestHandler creates a new color request handler
func NewColorRequestHandler(db *sql.DB, cfg *config.Config) *ColorRequestHandler {
	return &ColorRequestHandler{
		db:            db,
		cfg:           cfg,
		requestRepo:   repository.NewColorRequestRepository(db),
		colorRepo:     repository.NewColorCategoryRepository(db),
		userColorRepo: repository.NewUserColorRepository(db),
	}
}

// CreateRequest creates a new color request (user)
func (h *ColorRequestHandler) CreateRequest(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Parse request
	var req models.CreateColorRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Check if color exists
	color, err := h.colorRepo.FindByID(req.ColorID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check color")
		return
	}
	if color == nil {
		respondError(w, http.StatusBadRequest, "Color not found")
		return
	}

	// Check if user already has this color
	hasColor, err := h.userColorRepo.HasColor(userID, req.ColorID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check user colors")
		return
	}
	if hasColor {
		respondError(w, http.StatusBadRequest, "You already have this color")
		return
	}

	// Check if user has any pending request
	hasPending, err := h.requestRepo.HasPendingRequest(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check pending requests")
		return
	}
	if hasPending {
		respondError(w, http.StatusConflict, "You already have a pending color request")
		return
	}

	// Create the request
	colorRequest := &models.ColorRequest{
		UserID:  userID,
		ColorID: req.ColorID,
	}

	if err := h.requestRepo.Create(colorRequest); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create color request")
		return
	}

	respondJSON(w, http.StatusCreated, colorRequest)
}

// ListRequests lists color requests (user: own, admin: all pending)
func (h *ColorRequestHandler) ListRequests(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Check if admin
	isAdmin, _ := r.Context().Value(middleware.IsAdminKey).(bool)

	var requests []*models.ColorRequest
	var err error

	if isAdmin {
		// Admin sees all pending requests
		requests, err = h.requestRepo.FindAllPending()
	} else {
		// User sees only their own requests
		requests, err = h.requestRepo.FindByUserID(userID)
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get color requests")
		return
	}

	respondJSON(w, http.StatusOK, requests)
}

// ApproveRequest approves a color request (admin only)
func (h *ColorRequestHandler) ApproveRequest(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
	if !ok || !isAdmin {
		respondError(w, http.StatusForbidden, "Only admin can approve requests")
		return
	}

	adminID, _ := r.Context().Value(middleware.UserIDKey).(int)

	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request ID")
		return
	}

	// Get the request
	colorRequest, err := h.requestRepo.FindByID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get request")
		return
	}
	if colorRequest == nil {
		respondError(w, http.StatusNotFound, "Request not found")
		return
	}

	if colorRequest.Status != "pending" {
		respondError(w, http.StatusBadRequest, "Request is not pending")
		return
	}

	// Parse optional message
	var req struct {
		Message string `json:"message"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	var message *string
	if req.Message != "" {
		message = &req.Message
	}

	// Approve the request
	if err := h.requestRepo.Approve(id, adminID, message); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to approve request")
		return
	}

	// Add color to user
	if err := h.userColorRepo.AddColorToUser(colorRequest.UserID, colorRequest.ColorID, adminID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to add color to user")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Request approved successfully"})
}

// DenyRequest denies a color request (admin only)
func (h *ColorRequestHandler) DenyRequest(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
	if !ok || !isAdmin {
		respondError(w, http.StatusForbidden, "Only admin can deny requests")
		return
	}

	adminID, _ := r.Context().Value(middleware.UserIDKey).(int)

	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request ID")
		return
	}

	// Get the request
	colorRequest, err := h.requestRepo.FindByID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get request")
		return
	}
	if colorRequest == nil {
		respondError(w, http.StatusNotFound, "Request not found")
		return
	}

	if colorRequest.Status != "pending" {
		respondError(w, http.StatusBadRequest, "Request is not pending")
		return
	}

	// Parse optional message
	var req struct {
		Message string `json:"message"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	var message *string
	if req.Message != "" {
		message = &req.Message
	}

	// Deny the request
	if err := h.requestRepo.Deny(id, adminID, message); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to deny request")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Request denied successfully"})
}

// GetRequest gets a single color request by ID
func (h *ColorRequestHandler) GetRequest(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	isAdmin, _ := r.Context().Value(middleware.IsAdminKey).(bool)

	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request ID")
		return
	}

	colorRequest, err := h.requestRepo.FindByIDWithDetails(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get request")
		return
	}
	if colorRequest == nil {
		respondError(w, http.StatusNotFound, "Request not found")
		return
	}

	// Check authorization: admin can see all, user can only see their own
	if !isAdmin && colorRequest.UserID != userID {
		respondError(w, http.StatusForbidden, "Not authorized to view this request")
		return
	}

	respondJSON(w, http.StatusOK, colorRequest)
}
