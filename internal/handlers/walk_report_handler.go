package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
	"github.com/tranmh/gassigeher/internal/services"
)

// WalkReportHandler handles walk report-related HTTP requests
type WalkReportHandler struct {
	db              *sql.DB
	cfg             *config.Config
	walkReportRepo  *repository.WalkReportRepository
	bookingRepo     *repository.BookingRepository
	dogRepo         *repository.DogRepository
	imageService    *services.ImageService
}

// NewWalkReportHandler creates a new walk report handler
func NewWalkReportHandler(db *sql.DB, cfg *config.Config) *WalkReportHandler {
	return &WalkReportHandler{
		db:              db,
		cfg:             cfg,
		walkReportRepo:  repository.NewWalkReportRepository(db),
		bookingRepo:     repository.NewBookingRepository(db),
		dogRepo:         repository.NewDogRepository(db),
		imageService:    services.NewImageService(cfg.UploadDir),
	}
}

// CreateReport creates a new walk report for a completed booking
func (h *WalkReportHandler) CreateReport(w http.ResponseWriter, r *http.Request) {
	// Get user ID and admin status from context
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	isAdmin, _ := r.Context().Value(middleware.IsAdminKey).(bool)

	// Parse request
	var req models.CreateWalkReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Check if user owns this booking (admins can create for any booking)
	bookingUserID, err := h.walkReportRepo.GetBookingUserID(req.BookingID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Buchung nicht gefunden")
		return
	}

	if bookingUserID != userID && !isAdmin {
		respondError(w, http.StatusForbidden, "Sie können nur Berichte für Ihre eigenen Buchungen erstellen")
		return
	}

	// Check if booking is completed
	isCompleted, err := h.walkReportRepo.IsBookingCompleted(req.BookingID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check booking status")
		return
	}

	if !isCompleted {
		respondError(w, http.StatusBadRequest, "Berichte können nur für abgeschlossene Buchungen erstellt werden")
		return
	}

	// Check if report already exists
	existingReport, err := h.walkReportRepo.FindByBookingID(req.BookingID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check existing report")
		return
	}

	if existingReport != nil {
		respondError(w, http.StatusConflict, "Für diese Buchung existiert bereits ein Bericht")
		return
	}

	// Create report
	report := &models.WalkReport{
		BookingID:      req.BookingID,
		BehaviorRating: req.BehaviorRating,
		EnergyLevel:    req.EnergyLevel,
		Notes:          req.Notes,
	}

	if err := h.walkReportRepo.Create(report); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create report")
		return
	}

	respondJSON(w, http.StatusCreated, report)
}

// GetReport gets a walk report by ID
func (h *WalkReportHandler) GetReport(w http.ResponseWriter, r *http.Request) {
	// Get report ID from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid report ID")
		return
	}

	report, err := h.walkReportRepo.FindByID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get report")
		return
	}

	if report == nil {
		respondError(w, http.StatusNotFound, "Bericht nicht gefunden")
		return
	}

	respondJSON(w, http.StatusOK, report)
}

// GetReportByBooking gets a walk report by booking ID
func (h *WalkReportHandler) GetReportByBooking(w http.ResponseWriter, r *http.Request) {
	// Get booking ID from URL
	vars := mux.Vars(r)
	bookingID, err := strconv.Atoi(vars["bookingId"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid booking ID")
		return
	}

	report, err := h.walkReportRepo.FindByBookingID(bookingID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get report")
		return
	}

	if report == nil {
		respondError(w, http.StatusNotFound, "Bericht nicht gefunden")
		return
	}

	respondJSON(w, http.StatusOK, report)
}

// GetDogWalkReports gets all walk reports for a dog
func (h *WalkReportHandler) GetDogWalkReports(w http.ResponseWriter, r *http.Request) {
	// Get dog ID from URL
	vars := mux.Vars(r)
	dogID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid dog ID")
		return
	}

	// Get dog info
	dog, err := h.dogRepo.FindByID(dogID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get dog")
		return
	}

	if dog == nil {
		respondError(w, http.StatusNotFound, "Hund nicht gefunden")
		return
	}

	// Get limit from query params (default 10)
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	// Get reports
	reports, err := h.walkReportRepo.FindByDogID(dogID, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get reports")
		return
	}

	// Get stats
	stats, err := h.walkReportRepo.GetReportStats(dogID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get stats")
		return
	}

	response := &models.DogWalkReportsResponse{
		Dog:     dog,
		Stats:   stats,
		Reports: reports,
	}

	respondJSON(w, http.StatusOK, response)
}

// UpdateReport updates a walk report
func (h *WalkReportHandler) UpdateReport(w http.ResponseWriter, r *http.Request) {
	// Get user ID and admin status from context
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	isAdmin, _ := r.Context().Value(middleware.IsAdminKey).(bool)

	// Get report ID from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid report ID")
		return
	}

	// Get existing report
	report, err := h.walkReportRepo.FindByID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get report")
		return
	}

	if report == nil {
		respondError(w, http.StatusNotFound, "Bericht nicht gefunden")
		return
	}

	// Check if user owns this report's booking (admins can update any report)
	bookingUserID, err := h.walkReportRepo.GetBookingUserID(report.BookingID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to verify ownership")
		return
	}

	if bookingUserID != userID && !isAdmin {
		respondError(w, http.StatusForbidden, "Sie können nur Ihre eigenen Berichte bearbeiten")
		return
	}

	// Parse request
	var req models.UpdateWalkReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Update report
	report.BehaviorRating = req.BehaviorRating
	report.EnergyLevel = req.EnergyLevel
	report.Notes = req.Notes

	if err := h.walkReportRepo.Update(report); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update report")
		return
	}

	respondJSON(w, http.StatusOK, report)
}

// DeleteReport deletes a walk report
func (h *WalkReportHandler) DeleteReport(w http.ResponseWriter, r *http.Request) {
	// Get user ID and admin status from context
	userID, _ := r.Context().Value(middleware.UserIDKey).(int)
	isAdmin, _ := r.Context().Value(middleware.IsAdminKey).(bool)

	// Get report ID from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid report ID")
		return
	}

	// Get existing report
	report, err := h.walkReportRepo.FindByID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get report")
		return
	}

	if report == nil {
		respondError(w, http.StatusNotFound, "Bericht nicht gefunden")
		return
	}

	// Check authorization (user owns booking OR is admin)
	if !isAdmin {
		bookingUserID, err := h.walkReportRepo.GetBookingUserID(report.BookingID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to verify ownership")
			return
		}

		if bookingUserID != userID {
			respondError(w, http.StatusForbidden, "Sie können nur Ihre eigenen Berichte löschen")
			return
		}
	}

	// Delete photos from disk first
	for _, photo := range report.Photos {
		h.imageService.DeleteWalkReportPhoto(photo.PhotoPath, photo.PhotoThumbnail)
	}

	// Delete report (photos cascade deleted in DB)
	if err := h.walkReportRepo.Delete(id); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete report")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Report deleted"})
}

// UploadPhoto uploads a photo to a walk report
func (h *WalkReportHandler) UploadPhoto(w http.ResponseWriter, r *http.Request) {
	// Get user ID and admin status from context
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	isAdmin, _ := r.Context().Value(middleware.IsAdminKey).(bool)

	// Get report ID from URL
	vars := mux.Vars(r)
	reportID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid report ID")
		return
	}

	// Get existing report
	report, err := h.walkReportRepo.FindByID(reportID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get report")
		return
	}

	if report == nil {
		respondError(w, http.StatusNotFound, "Bericht nicht gefunden")
		return
	}

	// Check if user owns this report's booking (admins can upload to any report)
	bookingUserID, err := h.walkReportRepo.GetBookingUserID(report.BookingID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to verify ownership")
		return
	}

	if bookingUserID != userID && !isAdmin {
		respondError(w, http.StatusForbidden, "Sie können nur Fotos zu Ihren eigenen Berichten hinzufügen")
		return
	}

	// Check photo limit (max 3)
	photoCount, err := h.walkReportRepo.CountPhotos(reportID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to count photos")
		return
	}

	if photoCount >= 3 {
		respondError(w, http.StatusBadRequest, "Maximal 3 Fotos pro Bericht erlaubt")
		return
	}

	// Parse multipart form
	maxSize := int64(h.cfg.MaxUploadSizeMB) * 1024 * 1024
	if err := r.ParseMultipartForm(maxSize); err != nil {
		respondError(w, http.StatusBadRequest, "Datei zu groß")
		return
	}

	// Get file
	file, header, err := r.FormFile("photo")
	if err != nil {
		respondError(w, http.StatusBadRequest, "Keine Datei hochgeladen")
		return
	}
	defer file.Close()

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		respondError(w, http.StatusBadRequest, "Nur JPEG und PNG Dateien erlaubt")
		return
	}

	// Process and save the photo
	fullPath, thumbPath, err := h.imageService.ProcessWalkReportPhoto(file, reportID, photoCount+1)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to process photo")
		return
	}

	// Add photo to database
	photo, err := h.walkReportRepo.AddPhoto(reportID, fullPath, thumbPath, photoCount)
	if err != nil {
		// Clean up files if DB insert fails
		h.imageService.DeleteWalkReportPhoto(fullPath, thumbPath)
		respondError(w, http.StatusInternalServerError, "Failed to save photo")
		return
	}

	respondJSON(w, http.StatusCreated, photo)
}

// DeletePhoto deletes a photo from a walk report
func (h *WalkReportHandler) DeletePhoto(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get report and photo IDs from URL
	vars := mux.Vars(r)
	reportID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid report ID")
		return
	}

	photoID, err := strconv.Atoi(vars["photoId"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid photo ID")
		return
	}

	// Get existing report
	report, err := h.walkReportRepo.FindByID(reportID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get report")
		return
	}

	if report == nil {
		respondError(w, http.StatusNotFound, "Bericht nicht gefunden")
		return
	}

	// Check if user owns this report's booking (admins can delete any photo)
	isAdmin, _ := r.Context().Value(middleware.IsAdminKey).(bool)
	bookingUserID, err := h.walkReportRepo.GetBookingUserID(report.BookingID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to verify ownership")
		return
	}

	if bookingUserID != userID && !isAdmin {
		respondError(w, http.StatusForbidden, "Sie können nur Fotos aus Ihren eigenen Berichten löschen")
		return
	}

	// Get photo to delete
	photo, err := h.walkReportRepo.GetPhotoByID(photoID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get photo")
		return
	}

	if photo == nil {
		respondError(w, http.StatusNotFound, "Foto nicht gefunden")
		return
	}

	// Verify photo belongs to this report
	if photo.WalkReportID != reportID {
		respondError(w, http.StatusBadRequest, "Foto gehört nicht zu diesem Bericht")
		return
	}

	// Delete from database
	if err := h.walkReportRepo.DeletePhoto(photoID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete photo")
		return
	}

	// Delete files from disk
	h.imageService.DeleteWalkReportPhoto(photo.PhotoPath, photo.PhotoThumbnail)

	respondJSON(w, http.StatusOK, map[string]string{"message": "Photo deleted"})
}
