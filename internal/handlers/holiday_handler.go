package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
	"github.com/tranmh/gassigeher/internal/services"
)

type HolidayHandler struct {
	holidayRepo    *repository.HolidayRepository
	holidayService *services.HolidayService
}

func NewHolidayHandler(
	holidayRepo *repository.HolidayRepository,
	holidayService *services.HolidayService,
) *HolidayHandler {
	return &HolidayHandler{
		holidayRepo:    holidayRepo,
		holidayService: holidayService,
	}
}

// GetHolidays returns all holidays for a year
// GET /api/holidays?year=2025
func (h *HolidayHandler) GetHolidays(w http.ResponseWriter, r *http.Request) {
	yearStr := r.URL.Query().Get("year")
	year := time.Now().Year() // Default to current year

	if yearStr != "" {
		y, err := strconv.Atoi(yearStr)
		if err == nil {
			year = y
		}
	}

	holidays, err := h.holidayService.GetHolidaysForYear(year)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to load holidays")
		return
	}

	respondJSON(w, http.StatusOK, holidays)
}

// CreateHoliday adds a custom holiday (admin only)
// POST /api/holidays
func (h *HolidayHandler) CreateHoliday(w http.ResponseWriter, r *http.Request) {
	// Check admin permission
	isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
	if !ok || !isAdmin {
		respondError(w, http.StatusForbidden, "Admin access required")
		return
	}

	adminID, _ := r.Context().Value(middleware.UserIDKey).(int)

	var holiday models.CustomHoliday
	if err := json.NewDecoder(r.Body).Decode(&holiday); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Set source and createdBy before validation
	holiday.Source = "admin"
	holiday.CreatedBy = &adminID

	if err := holiday.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.holidayRepo.CreateHoliday(&holiday); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create holiday")
		return
	}

	respondJSON(w, http.StatusCreated, holiday)
}

// UpdateHoliday updates a holiday (admin only)
// PUT /api/holidays/:id
func (h *HolidayHandler) UpdateHoliday(w http.ResponseWriter, r *http.Request) {
	// Check admin permission
	isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
	if !ok || !isAdmin {
		respondError(w, http.StatusForbidden, "Admin access required")
		return
	}

	// Extract ID from path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		respondError(w, http.StatusBadRequest, "Invalid holiday ID")
		return
	}
	idStr := pathParts[len(pathParts)-1]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid holiday ID")
		return
	}

	var holiday models.CustomHoliday
	if err := json.NewDecoder(r.Body).Decode(&holiday); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.holidayRepo.UpdateHoliday(id, &holiday); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update holiday")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Holiday updated successfully",
	})
}

// DeleteHoliday deletes a holiday (admin only)
// DELETE /api/holidays/:id
func (h *HolidayHandler) DeleteHoliday(w http.ResponseWriter, r *http.Request) {
	// Check admin permission
	isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
	if !ok || !isAdmin {
		respondError(w, http.StatusForbidden, "Admin access required")
		return
	}

	// Extract ID from path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		respondError(w, http.StatusBadRequest, "Invalid holiday ID")
		return
	}
	idStr := pathParts[len(pathParts)-1]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid holiday ID")
		return
	}

	if err := h.holidayRepo.DeleteHoliday(id); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete holiday")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Holiday deleted successfully",
	})
}
