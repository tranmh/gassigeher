package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/tranm/gassigeher/internal/middleware"
	"github.com/tranm/gassigeher/internal/models"
	"github.com/tranm/gassigeher/internal/repository"
	"github.com/tranm/gassigeher/internal/services"
)

type BookingTimeHandler struct {
	bookingTimeRepo    *repository.BookingTimeRepository
	bookingTimeService *services.BookingTimeService
}

func NewBookingTimeHandler(
	bookingTimeRepo *repository.BookingTimeRepository,
	bookingTimeService *services.BookingTimeService,
) *BookingTimeHandler {
	return &BookingTimeHandler{
		bookingTimeRepo:    bookingTimeRepo,
		bookingTimeService: bookingTimeService,
	}
}

// GetAvailableSlots returns available time slots for a date
// GET /api/booking-times/available?date=YYYY-MM-DD
func (h *BookingTimeHandler) GetAvailableSlots(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	if date == "" {
		respondError(w, http.StatusBadRequest, "date parameter required")
		return
	}

	slots, err := h.bookingTimeService.GetAvailableTimeSlots(date)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"date":  date,
		"slots": slots,
	})
}

// GetRules returns all time rules
// GET /api/booking-times/rules
func (h *BookingTimeHandler) GetRules(w http.ResponseWriter, r *http.Request) {
	// Check admin permission
	isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
	if !ok || !isAdmin {
		respondError(w, http.StatusForbidden, "Admin access required")
		return
	}

	rules, err := h.bookingTimeRepo.GetAllRules()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to load rules")
		return
	}

	respondJSON(w, http.StatusOK, rules)
}

// GetRulesForDate returns applicable rules for a specific date
// GET /api/booking-times/rules-for-date?date=YYYY-MM-DD
func (h *BookingTimeHandler) GetRulesForDate(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	if date == "" {
		respondError(w, http.StatusBadRequest, "date parameter required")
		return
	}

	rules, err := h.bookingTimeService.GetRulesForDate(date)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, rules)
}

// UpdateRules updates time rules (admin only)
// PUT /api/booking-times/rules
func (h *BookingTimeHandler) UpdateRules(w http.ResponseWriter, r *http.Request) {
	// Check admin permission
	isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
	if !ok || !isAdmin {
		respondError(w, http.StatusForbidden, "Admin access required")
		return
	}

	var rules []models.BookingTimeRule
	if err := json.NewDecoder(r.Body).Decode(&rules); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate each rule
	for _, rule := range rules {
		if err := rule.Validate(); err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	// Update each rule
	for _, rule := range rules {
		if err := h.bookingTimeRepo.UpdateRule(rule.ID, &rule); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to update rule")
			return
		}
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Rules updated successfully",
	})
}

// CreateRule creates a new time rule (admin only)
// POST /api/booking-times/rules
func (h *BookingTimeHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	// Check admin permission
	isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
	if !ok || !isAdmin {
		respondError(w, http.StatusForbidden, "Admin access required")
		return
	}

	var rule models.BookingTimeRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := rule.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.bookingTimeRepo.CreateRule(&rule); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create rule")
		return
	}

	respondJSON(w, http.StatusCreated, rule)
}

// DeleteRule deletes a time rule (admin only)
// DELETE /api/booking-times/rules/:id
func (h *BookingTimeHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	// Check admin permission
	isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
	if !ok || !isAdmin {
		respondError(w, http.StatusForbidden, "Admin access required")
		return
	}

	// Extract ID from path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		respondError(w, http.StatusBadRequest, "Invalid rule ID")
		return
	}
	idStr := pathParts[len(pathParts)-1]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid rule ID")
		return
	}

	if err := h.bookingTimeRepo.DeleteRule(id); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete rule")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Rule deleted successfully",
	})
}
