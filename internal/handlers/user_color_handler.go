package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/repository"
)

// UserColorHandler handles user color-related HTTP requests (admin)
type UserColorHandler struct {
	db            *sql.DB
	cfg           *config.Config
	userColorRepo *repository.UserColorRepository
	colorRepo     *repository.ColorCategoryRepository
	userRepo      *repository.UserRepository
}

// NewUserColorHandler creates a new user color handler
func NewUserColorHandler(db *sql.DB, cfg *config.Config) *UserColorHandler {
	return &UserColorHandler{
		db:            db,
		cfg:           cfg,
		userColorRepo: repository.NewUserColorRepository(db),
		colorRepo:     repository.NewColorCategoryRepository(db),
		userRepo:      repository.NewUserRepository(db),
	}
}

// GetUserColors returns all colors assigned to a user (admin)
func (h *UserColorHandler) GetUserColors(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
	if !ok || !isAdmin {
		respondError(w, http.StatusForbidden, "Only admin can view user colors")
		return
	}

	vars := mux.Vars(r)
	userIDStr := vars["id"]
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Check if user exists
	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}
	if user == nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	colors, err := h.userColorRepo.GetUserColors(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get user colors")
		return
	}

	respondJSON(w, http.StatusOK, colors)
}

// AddColorToUser adds a color to a user (admin)
func (h *UserColorHandler) AddColorToUser(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
	if !ok || !isAdmin {
		respondError(w, http.StatusForbidden, "Only admin can add colors to users")
		return
	}

	adminID, _ := r.Context().Value(middleware.UserIDKey).(int)

	vars := mux.Vars(r)
	userIDStr := vars["id"]
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Check if user exists
	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}
	if user == nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Parse request
	var req struct {
		ColorID int `json:"color_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.ColorID <= 0 {
		respondError(w, http.StatusBadRequest, "Color ID is required")
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
		respondError(w, http.StatusConflict, "User already has this color")
		return
	}

	// Add color to user
	if err := h.userColorRepo.AddColorToUser(userID, req.ColorID, adminID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to add color to user")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Color added successfully"})
}

// RemoveColorFromUser removes a color from a user (admin)
func (h *UserColorHandler) RemoveColorFromUser(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
	if !ok || !isAdmin {
		respondError(w, http.StatusForbidden, "Only admin can remove colors from users")
		return
	}

	vars := mux.Vars(r)
	userIDStr := vars["id"]
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	colorIDStr := vars["colorId"]
	colorID, err := strconv.Atoi(colorIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid color ID")
		return
	}

	// Check if user exists
	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}
	if user == nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Check if user has this color
	hasColor, err := h.userColorRepo.HasColor(userID, colorID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check user colors")
		return
	}
	if !hasColor {
		respondError(w, http.StatusNotFound, "User does not have this color")
		return
	}

	// Remove color from user
	if err := h.userColorRepo.RemoveColorFromUser(userID, colorID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to remove color from user")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Color removed successfully"})
}

// SetUserColors sets all colors for a user (admin) - replaces existing colors
func (h *UserColorHandler) SetUserColors(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
	if !ok || !isAdmin {
		respondError(w, http.StatusForbidden, "Only admin can set user colors")
		return
	}

	adminID, _ := r.Context().Value(middleware.UserIDKey).(int)

	vars := mux.Vars(r)
	userIDStr := vars["id"]
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Check if user exists
	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get user")
		return
	}
	if user == nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Parse request
	var req struct {
		ColorIDs []int `json:"color_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate all color IDs exist
	for _, colorID := range req.ColorIDs {
		color, err := h.colorRepo.FindByID(colorID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to check color")
			return
		}
		if color == nil {
			respondError(w, http.StatusBadRequest, "Color not found: "+strconv.Itoa(colorID))
			return
		}
	}

	// Set user colors (replace all)
	if err := h.userColorRepo.SetUserColors(userID, req.ColorIDs, adminID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to set user colors")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Colors updated successfully"})
}
