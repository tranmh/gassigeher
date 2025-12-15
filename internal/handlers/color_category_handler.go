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

// ColorCategoryHandler handles color category-related HTTP requests
type ColorCategoryHandler struct {
	db        *sql.DB
	cfg       *config.Config
	colorRepo *repository.ColorCategoryRepository
}

// NewColorCategoryHandler creates a new color category handler
func NewColorCategoryHandler(db *sql.DB, cfg *config.Config) *ColorCategoryHandler {
	return &ColorCategoryHandler{
		db:        db,
		cfg:       cfg,
		colorRepo: repository.NewColorCategoryRepository(db),
	}
}

// ListColors returns all color categories (public endpoint)
func (h *ColorCategoryHandler) ListColors(w http.ResponseWriter, r *http.Request) {
	colors, err := h.colorRepo.FindAll()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get colors")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"colors": colors})
}

// GetColor returns a single color category by ID
func (h *ColorCategoryHandler) GetColor(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid color ID")
		return
	}

	color, err := h.colorRepo.FindByID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get color")
		return
	}

	if color == nil {
		respondError(w, http.StatusNotFound, "Color not found")
		return
	}

	respondJSON(w, http.StatusOK, color)
}

// CreateColor creates a new color category (super-admin only)
func (h *ColorCategoryHandler) CreateColor(w http.ResponseWriter, r *http.Request) {
	// Check if user is super admin
	isSuperAdmin, ok := r.Context().Value(middleware.IsSuperAdminKey).(bool)
	if !ok || !isSuperAdmin {
		respondError(w, http.StatusForbidden, "Only super admin can create colors")
		return
	}

	// Check max colors limit (15)
	count, err := h.colorRepo.Count()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check color count")
		return
	}
	if count >= 15 {
		respondError(w, http.StatusBadRequest, "Maximum 15 colors allowed")
		return
	}

	// Parse request
	var req models.CreateColorCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Check if name already exists
	existing, _ := h.colorRepo.FindByName(req.Name)
	if existing != nil {
		respondError(w, http.StatusConflict, "Color with this name already exists")
		return
	}

	// Get next sort order
	sortOrder, err := h.colorRepo.GetNextSortOrder()
	if err != nil {
		sortOrder = count + 1
	}

	// Create color
	color := &models.ColorCategory{
		Name:        req.Name,
		HexCode:     req.HexCode,
		PatternIcon: req.PatternIcon,
		SortOrder:   sortOrder,
	}

	if err := h.colorRepo.Create(color); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create color")
		return
	}

	respondJSON(w, http.StatusCreated, color)
}

// UpdateColor updates a color category (super-admin only)
func (h *ColorCategoryHandler) UpdateColor(w http.ResponseWriter, r *http.Request) {
	// Check if user is super admin
	isSuperAdmin, ok := r.Context().Value(middleware.IsSuperAdminKey).(bool)
	if !ok || !isSuperAdmin {
		respondError(w, http.StatusForbidden, "Only super admin can update colors")
		return
	}

	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid color ID")
		return
	}

	// Get existing color
	color, err := h.colorRepo.FindByID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get color")
		return
	}
	if color == nil {
		respondError(w, http.StatusNotFound, "Color not found")
		return
	}

	// Parse request
	var req models.UpdateColorCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Apply updates
	if req.Name != nil && *req.Name != "" {
		// Check if new name conflicts
		existing, _ := h.colorRepo.FindByName(*req.Name)
		if existing != nil && existing.ID != id {
			respondError(w, http.StatusConflict, "Color with this name already exists")
			return
		}
		color.Name = *req.Name
	}
	if req.HexCode != nil && *req.HexCode != "" {
		color.HexCode = *req.HexCode
	}
	if req.PatternIcon != nil {
		color.PatternIcon = req.PatternIcon
	}
	if req.SortOrder != nil {
		color.SortOrder = *req.SortOrder
	}

	if err := h.colorRepo.Update(color); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update color")
		return
	}

	respondJSON(w, http.StatusOK, color)
}

// DeleteColor deletes a color category (super-admin only)
func (h *ColorCategoryHandler) DeleteColor(w http.ResponseWriter, r *http.Request) {
	// Check if user is super admin
	isSuperAdmin, ok := r.Context().Value(middleware.IsSuperAdminKey).(bool)
	if !ok || !isSuperAdmin {
		respondError(w, http.StatusForbidden, "Only super admin can delete colors")
		return
	}

	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid color ID")
		return
	}

	// Check if color exists
	color, err := h.colorRepo.FindByID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get color")
		return
	}
	if color == nil {
		respondError(w, http.StatusNotFound, "Color not found")
		return
	}

	// Check min colors limit (3)
	count, err := h.colorRepo.Count()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check color count")
		return
	}
	if count <= 3 {
		respondError(w, http.StatusBadRequest, "Minimum 3 colors required")
		return
	}

	// Check if any dogs are assigned to this color
	dogCount, err := h.colorRepo.CountDogsWithColor(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check dogs")
		return
	}
	if dogCount > 0 {
		respondError(w, http.StatusBadRequest, "Cannot delete color with dogs assigned. Reassign dogs first.")
		return
	}

	if err := h.colorRepo.Delete(id); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete color")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Color deleted successfully"})
}

// GetColorStats returns stats for a color (dogs count, users count)
func (h *ColorCategoryHandler) GetColorStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid color ID")
		return
	}

	dogCount, err := h.colorRepo.CountDogsWithColor(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to count dogs")
		return
	}

	userCount, err := h.colorRepo.CountUsersWithColor(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to count users")
		return
	}

	respondJSON(w, http.StatusOK, map[string]int{
		"dog_count":  dogCount,
		"user_count": userCount,
	})
}
