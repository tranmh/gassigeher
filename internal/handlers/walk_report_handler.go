package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
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
	s3Service       *services.S3Service // SaaS: For S3 storage
}

// NewWalkReportHandler creates a new walk report handler
func NewWalkReportHandler(db *sql.DB, cfg *config.Config) *WalkReportHandler {
	// Initialize S3 service if configured
	var s3Service *services.S3Service
	if cfg.UseS3 {
		s3Config := &services.S3Config{
			Endpoint:   cfg.S3Endpoint,
			AccessKey:  cfg.S3AccessKey,
			SecretKey:  cfg.S3SecretKey,
			BucketName: cfg.S3BucketName,
			Region:     cfg.S3Region,
			PublicURL:  cfg.S3PublicURL,
			UseSSL:     cfg.S3UseSSL,
		}
		var err error
		s3Service, err = services.NewS3Service(s3Config)
		if err != nil {
			println("Warning: Failed to initialize S3 service for walk reports:", err.Error())
		}
	}

	return &WalkReportHandler{
		db:              db,
		cfg:             cfg,
		walkReportRepo:  repository.NewWalkReportRepository(db),
		bookingRepo:     repository.NewBookingRepository(db),
		dogRepo:         repository.NewDogRepository(db),
		imageService:    services.NewImageService(cfg.UploadDir),
		s3Service:       s3Service,
	}
}

// getImageServiceForRequest returns an ImageService with the correct tenant slug for the request
// In SaaS-Mode, this ensures photos are stored in tenant-specific directories
func (h *WalkReportHandler) getImageServiceForRequest(r *http.Request) *services.ImageService {
	tenantSlug, _ := r.Context().Value(middleware.TenantSlugKey).(string)
	if tenantSlug != "" {
		// SaaS-Mode: Return tenant-aware ImageService
		return services.NewImageServiceWithTenant(h.cfg.UploadDir, tenantSlug)
	}
	// Simple-Mode: Use default ImageService (no tenant prefix)
	return h.imageService
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
	// SaaS: Extract tenant ID from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

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
	bookingUserID, err := h.walkReportRepo.GetBookingUserID(tenantID, req.BookingID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Buchung nicht gefunden")
		return
	}

	if bookingUserID != userID && !isAdmin {
		respondError(w, http.StatusForbidden, "Sie können nur Berichte für Ihre eigenen Buchungen erstellen")
		return
	}

	// Check if booking is completed
	isCompleted, err := h.walkReportRepo.IsBookingCompleted(tenantID, req.BookingID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to check booking status")
		return
	}

	if !isCompleted {
		respondError(w, http.StatusBadRequest, "Berichte können nur für abgeschlossene Buchungen erstellt werden")
		return
	}

	// Check if report already exists
	existingReport, err := h.walkReportRepo.FindByBookingID(tenantID, req.BookingID)
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

	if err := h.walkReportRepo.Create(tenantID, report); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create report")
		return
	}

	respondJSON(w, http.StatusCreated, report)
}

// GetReport gets a walk report by ID
func (h *WalkReportHandler) GetReport(w http.ResponseWriter, r *http.Request) {
	// SaaS: Extract tenant ID from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	// Get report ID from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid report ID")
		return
	}

	report, err := h.walkReportRepo.FindByID(tenantID, id)
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
	// SaaS: Extract tenant ID from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	// Get booking ID from URL
	vars := mux.Vars(r)
	bookingID, err := strconv.Atoi(vars["bookingId"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid booking ID")
		return
	}

	report, err := h.walkReportRepo.FindByBookingID(tenantID, bookingID)
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
	// SaaS: Extract tenant ID from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

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
	reports, err := h.walkReportRepo.FindByDogID(tenantID, dogID, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get reports")
		return
	}

	// Get stats
	stats, err := h.walkReportRepo.GetReportStats(tenantID, dogID)
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
	// SaaS: Extract tenant ID from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	// Get report ID from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid report ID")
		return
	}

	// Get existing report
	report, err := h.walkReportRepo.FindByID(tenantID, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get report")
		return
	}

	if report == nil {
		respondError(w, http.StatusNotFound, "Bericht nicht gefunden")
		return
	}

	// Check if user owns this report's booking (admins can update any report)
	bookingUserID, err := h.walkReportRepo.GetBookingUserID(tenantID, report.BookingID)
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

	if err := h.walkReportRepo.Update(tenantID, report); err != nil {
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
	// SaaS: Extract tenant ID from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	// Get report ID from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid report ID")
		return
	}

	// Get existing report
	report, err := h.walkReportRepo.FindByID(tenantID, id)
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
		bookingUserID, err := h.walkReportRepo.GetBookingUserID(tenantID, report.BookingID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to verify ownership")
			return
		}

		if bookingUserID != userID {
			respondError(w, http.StatusForbidden, "Sie können nur Ihre eigenen Berichte löschen")
			return
		}
	}

	// Delete photos from storage (S3 or local filesystem)
	for _, photo := range report.Photos {
		// Check if photo is stored in S3 (URL starts with http)
		if strings.HasPrefix(photo.PhotoPath, "http://") || strings.HasPrefix(photo.PhotoPath, "https://") {
			// S3 storage - deletion handled separately if needed
			// For now, we skip S3 deletion as it requires parsing the URL
			continue
		}
		// Local storage - use tenant-aware ImageService
		imageService := h.getImageServiceForRequest(r)
		imageService.DeleteWalkReportPhoto(photo.PhotoPath, photo.PhotoThumbnail)
	}

	// Delete report (photos cascade deleted in DB)
	if err := h.walkReportRepo.Delete(tenantID, id); err != nil {
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
	// SaaS: Extract tenant ID from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	// Get report ID from URL
	vars := mux.Vars(r)
	reportID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid report ID")
		return
	}

	// Get existing report
	report, err := h.walkReportRepo.FindByID(tenantID, reportID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get report")
		return
	}

	if report == nil {
		respondError(w, http.StatusNotFound, "Bericht nicht gefunden")
		return
	}

	// Check if user owns this report's booking (admins can upload to any report)
	bookingUserID, err := h.walkReportRepo.GetBookingUserID(tenantID, report.BookingID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to verify ownership")
		return
	}

	if bookingUserID != userID && !isAdmin {
		respondError(w, http.StatusForbidden, "Sie können nur Fotos zu Ihren eigenen Berichten hinzufügen")
		return
	}

	// Check photo limit (max 3)
	photoCount, err := h.walkReportRepo.CountPhotos(tenantID, reportID)
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

	// Validate MIME type (magic bytes) to prevent file type spoofing
	if errMsg, valid := ValidateImageMIMEType(file); !valid {
		respondError(w, http.StatusBadRequest, errMsg)
		return
	}
	// Reset file reader position after MIME check
	file.Seek(0, 0)

	var fullPath, thumbPath string

	// SaaS: Use S3 storage if enabled
	if h.s3Service != nil && h.cfg.UseS3 {
		// Get tenant slug from context
		tenantSlug, _ := r.Context().Value(middleware.TenantSlugKey).(string)
		if tenantSlug == "" {
			tenantSlug = "default"
		}

		// Read file data
		fileData, err := io.ReadAll(file)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to read file")
			return
		}

		// Determine content type
		contentType := "image/jpeg"
		if ext == ".png" {
			contentType = "image/png"
		}

		// Upload full-size photo to S3
		fullObjectPath := fmt.Sprintf("walk_reports/report_%d_photo_%d%s", reportID, photoCount+1, ext)
		fullURL, err := h.s3Service.Upload(r.Context(), tenantSlug, fullObjectPath, fileData, contentType)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to upload photo to S3")
			return
		}

		// For S3, we store the full URL in the database
		fullPath = fullURL
		thumbPath = fullURL // S3 uses same URL for now (no auto-resize)
	} else {
		// Local filesystem storage with tenant isolation
		imageService := h.getImageServiceForRequest(r)
		var err error
		fullPath, thumbPath, err = imageService.ProcessWalkReportPhoto(file, reportID, photoCount+1)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to process photo")
			return
		}
	}

	// Add photo to database
	photo, err := h.walkReportRepo.AddPhoto(tenantID, reportID, fullPath, thumbPath, photoCount)
	if err != nil {
		// Clean up files if DB insert fails (only for local storage)
		if h.s3Service == nil || !h.cfg.UseS3 {
			imageService := h.getImageServiceForRequest(r)
			imageService.DeleteWalkReportPhoto(fullPath, thumbPath)
		}
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
	// SaaS: Extract tenant ID from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

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
	report, err := h.walkReportRepo.FindByID(tenantID, reportID)
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
	bookingUserID, err := h.walkReportRepo.GetBookingUserID(tenantID, report.BookingID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to verify ownership")
		return
	}

	if bookingUserID != userID && !isAdmin {
		respondError(w, http.StatusForbidden, "Sie können nur Fotos aus Ihren eigenen Berichten löschen")
		return
	}

	// Get photo to delete
	photo, err := h.walkReportRepo.GetPhotoByID(tenantID, photoID)
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
	if err := h.walkReportRepo.DeletePhoto(tenantID, photoID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete photo")
		return
	}

	// Delete files from storage (S3 or local filesystem)
	// Check if photo is stored in S3 (URL starts with http)
	if !strings.HasPrefix(photo.PhotoPath, "http://") && !strings.HasPrefix(photo.PhotoPath, "https://") {
		// Local storage - use tenant-aware ImageService
		imageService := h.getImageServiceForRequest(r)
		imageService.DeleteWalkReportPhoto(photo.PhotoPath, photo.PhotoThumbnail)
	}
	// S3 storage deletion is skipped for now (requires parsing URL)

	respondJSON(w, http.StatusOK, map[string]string{"message": "Photo deleted"})
}
