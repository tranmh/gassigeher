package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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

// DogHandler handles dog-related endpoints
type DogHandler struct {
	dogRepo          *repository.DogRepository
	userRepo         *repository.UserRepository
	bookingRepo      *repository.BookingRepository
	subscriptionRepo *repository.SubscriptionRepository    // SaaS: For checking dog limits
	colorRepo        *repository.ColorCategoryRepository   // For resolving legacy category to color_id
	imageService     *services.ImageService
	emailService     *services.EmailService
	s3Service        *services.S3Service // SaaS: For S3 storage
	config           *config.Config
}

// NewDogHandler creates a new dog handler
func NewDogHandler(db *sql.DB, cfg *config.Config) *DogHandler {
	// Initialize email service (may fail gracefully)
	emailService, err := services.NewEmailService(services.ConfigToEmailConfig(cfg))
	if err != nil {
		fmt.Printf("Warning: Failed to initialize email service in DogHandler: %v\n", err)
	}

	// Initialize S3 service if enabled
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
		s3Service, err = services.NewS3Service(s3Config)
		if err != nil {
			fmt.Printf("Warning: Failed to initialize S3 service in DogHandler: %v\n", err)
		} else {
			fmt.Printf("S3 storage enabled for DogHandler\n")
		}
	}

	return &DogHandler{
		dogRepo:          repository.NewDogRepository(db),
		userRepo:         repository.NewUserRepository(db),
		bookingRepo:      repository.NewBookingRepository(db),
		subscriptionRepo: repository.NewSubscriptionRepository(db),    // SaaS: For dog limit checks
		colorRepo:        repository.NewColorCategoryRepository(db),   // For resolving legacy category to color_id
		imageService:     services.NewImageService(cfg.UploadDir),
		emailService:     emailService,
		s3Service:        s3Service,
		config:           cfg,
	}
}

// ListDogs handles GET /api/dogs - list all dogs with optional filters
func (h *DogHandler) ListDogs(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for filtering
	filter := &models.DogFilterRequest{}

	if breed := r.URL.Query().Get("breed"); breed != "" {
		filter.Breed = &breed
	}

	if size := r.URL.Query().Get("size"); size != "" {
		filter.Size = &size
	}

	if minAge := r.URL.Query().Get("min_age"); minAge != "" {
		if age, err := strconv.Atoi(minAge); err == nil {
			filter.MinAge = &age
		}
	}

	if maxAge := r.URL.Query().Get("max_age"); maxAge != "" {
		if age, err := strconv.Atoi(maxAge); err == nil {
			filter.MaxAge = &age
		}
	}

	if category := r.URL.Query().Get("category"); category != "" {
		filter.Category = &category
	}

	// Accept both "available" and "is_available" for backwards compatibility
	availableParam := r.URL.Query().Get("available")
	if availableParam == "" {
		availableParam = r.URL.Query().Get("is_available")
	}
	if availableParam != "" {
		avail := availableParam == "true" || availableParam == "1"
		filter.Available = &avail
	}

	if search := r.URL.Query().Get("search"); search != "" {
		filter.Search = &search
	}

	// SaaS: Get tenant_id from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	// Get dogs
	dogs, err := h.dogRepo.FindAll(filter, tenantID)
	if err != nil {
		log.Printf("ERROR: Failed to fetch dogs: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to fetch dogs")
		return
	}

	// Return all dogs - frontend handles permission display using color_id
	// Users can see all dogs but locked dogs (those without matching color) show a banner
	respondJSON(w, http.StatusOK, dogs)
}

// GetDog handles GET /api/dogs/:id - get a single dog
func (h *DogHandler) GetDog(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid dog ID")
		return
	}

	dog, err := h.dogRepo.FindByID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if dog == nil {
		respondError(w, http.StatusNotFound, "Dog not found")
		return
	}

	respondJSON(w, http.StatusOK, dog)
}

// FreeTierDogLimit is the maximum number of dogs allowed for free tier tenants
const FreeTierDogLimit = 10

// CreateDog handles POST /api/dogs - create a new dog (admin only)
func (h *DogHandler) CreateDog(w http.ResponseWriter, r *http.Request) {
	var req models.CreateDogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// SaaS: Get tenant_id from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	// SaaS: Get dog limit from subscription (will be used for atomic create)
	var dogLimit int = -1 // Default unlimited for non-tenant mode
	if tenantID > 0 {
		var err error
		dogLimit, err = h.subscriptionRepo.GetTenantDogLimit(tenantID)
		if err != nil {
			log.Printf("ERROR: Failed to get dog limit for tenant %d: %v", tenantID, err)
			respondError(w, http.StatusInternalServerError, "Failed to check subscription")
			return
		}
	}

	// Validate required fields
	if strings.TrimSpace(req.Name) == "" {
		respondError(w, http.StatusBadRequest, "Name is required")
		return
	}

	if strings.TrimSpace(req.Breed) == "" {
		respondError(w, http.StatusBadRequest, "Breed is required")
		return
	}

	if req.Size != "small" && req.Size != "medium" && req.Size != "large" {
		respondError(w, http.StatusBadRequest, "Size must be small, medium, or large")
		return
	}

	// Validate color_id is provided (new color system)
	// Category is now legacy - if color_id is provided, category validation is skipped
	if req.ColorID == nil && req.Category == "" {
		respondError(w, http.StatusBadRequest, "Color is required")
		return
	}

	// If only legacy category is provided, validate it and resolve to color_id
	if req.ColorID == nil && req.Category != "" {
		if req.Category != "green" && req.Category != "blue" && req.Category != "orange" {
			respondError(w, http.StatusBadRequest, "Category must be green, blue, or orange")
			return
		}

		// Resolve legacy category to color_id
		color, err := h.colorRepo.FindByLegacyCategory(tenantID, req.Category)
		if err != nil {
			log.Printf("ERROR: Failed to resolve legacy category %s: %v", req.Category, err)
			respondError(w, http.StatusInternalServerError, "Failed to resolve color category")
			return
		}
		if color != nil {
			req.ColorID = &color.ID
		}
	}

	// Set default category for database CHECK constraint (legacy field)
	// When using new color system, category is not sent but DB requires valid value
	category := req.Category
	if category == "" {
		category = "green" // Default to satisfy CHECK constraint
	}

	// Create dog
	dog := &models.Dog{
		TenantID:            tenantID, // SaaS: Set tenant_id from context
		Name:                req.Name,
		Breed:               req.Breed,
		Size:                req.Size,
		Age:                 req.Age,
		Category:            category,
		ColorID:             req.ColorID,
		SpecialNeeds:        req.SpecialNeeds,
		PickupLocation:      req.PickupLocation,
		WalkRoute:           req.WalkRoute,
		WalkDuration:        req.WalkDuration,
		SpecialInstructions: req.SpecialInstructions,
		DefaultMorningTime:  req.DefaultMorningTime,
		DefaultEveningTime:  req.DefaultEveningTime,
		ExternalLink:        req.ExternalLink,
		IsAvailable:         true, // Default to available
	}

	// SaaS: Use atomic create with limit check to prevent race conditions
	if err := h.dogRepo.CreateWithLimitCheck(dog, dogLimit); err != nil {
		if err == repository.ErrDogLimitExceeded {
			// Get current count for error response
			currentCount, _ := h.dogRepo.CountByTenant(tenantID)
			respondJSON(w, http.StatusConflict, map[string]interface{}{
				"error":         "Hundelimit erreicht",
				"message":       "Sie haben das Maximum von 10 Hunden für das kostenlose Konto erreicht. Bitte upgraden Sie auf Pro für unbegrenzte Hunde.",
				"current_count": currentCount,
				"limit":         dogLimit,
			})
			return
		}
		log.Printf("ERROR: Failed to create dog: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to create dog")
		return
	}

	respondJSON(w, http.StatusCreated, dog)
}

// UpdateDog handles PUT /api/dogs/:id - update a dog (admin only)
func (h *DogHandler) UpdateDog(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid dog ID")
		return
	}

	// Get existing dog
	dog, err := h.dogRepo.FindByID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if dog == nil {
		respondError(w, http.StatusNotFound, "Dog not found")
		return
	}

	// Parse update request
	var req models.UpdateDogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Update fields if provided
	if req.Name != nil {
		// Validate name is not empty
		if strings.TrimSpace(*req.Name) == "" {
			respondError(w, http.StatusBadRequest, "Name is required")
			return
		}
		dog.Name = *req.Name
	}
	if req.Breed != nil {
		dog.Breed = *req.Breed
	}
	if req.Size != nil {
		dog.Size = *req.Size
	}
	if req.Age != nil {
		dog.Age = *req.Age
	}
	if req.Category != nil {
		dog.Category = *req.Category
	}
	if req.ColorID != nil {
		dog.ColorID = req.ColorID
	}
	if req.SpecialNeeds != nil {
		dog.SpecialNeeds = req.SpecialNeeds
	}
	if req.PickupLocation != nil {
		dog.PickupLocation = req.PickupLocation
	}
	if req.WalkRoute != nil {
		dog.WalkRoute = req.WalkRoute
	}
	if req.WalkDuration != nil {
		dog.WalkDuration = req.WalkDuration
	}
	if req.SpecialInstructions != nil {
		dog.SpecialInstructions = req.SpecialInstructions
	}
	if req.DefaultMorningTime != nil {
		dog.DefaultMorningTime = req.DefaultMorningTime
	}
	if req.DefaultEveningTime != nil {
		dog.DefaultEveningTime = req.DefaultEveningTime
	}
	if req.ExternalLink != nil {
		dog.ExternalLink = req.ExternalLink
	}

	// Update in database
	if err := h.dogRepo.Update(dog); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update dog")
		return
	}

	respondJSON(w, http.StatusOK, dog)
}

// DeleteDog handles DELETE /api/dogs/:id - delete a dog (admin only)
func (h *DogHandler) DeleteDog(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid dog ID")
		return
	}

	// Check if force delete is requested
	force := r.URL.Query().Get("force") == "true"

	if force {
		// Force delete: cancel all future bookings and delete dog
		dog, err := h.dogRepo.FindByID(id)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to fetch dog")
			return
		}
		if dog == nil {
			respondError(w, http.StatusNotFound, "Dog not found")
			return
		}

		// Get all future bookings
		bookings, err := h.dogRepo.GetFutureBookings(id)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to fetch bookings")
			return
		}

		// Cancel all future bookings
		cancellationReason := fmt.Sprintf("Hund %s wurde aus dem System entfernt", dog.Name)
		for _, booking := range bookings {
			// Cancel the booking
			err := h.bookingRepo.Cancel(booking.ID, &cancellationReason)
			if err != nil {
				log.Printf("ERROR: Failed to cancel booking %d: %v", booking.ID, err)
				continue
			}

			// Send cancellation email to user if email service is available and user has email
			if h.emailService != nil && booking.User != nil && booking.User.Email != nil && *booking.User.Email != "" {
				go h.emailService.SendBookingCancellation(
					*booking.User.Email,
					booking.User.FirstName,
					dog.Name,
					booking.Date,
					booking.ScheduledTime,
				)
			}
		}

		// Now delete the dog
		if err := h.dogRepo.ForceDelete(id); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to delete dog")
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"message":          "Hund erfolgreich gelöscht",
			"cancelled_count":  len(bookings),
		})
		return
	}

	// Normal delete (will fail if future bookings exist)
	err = h.dogRepo.Delete(id)
	if err != nil {
		if strings.Contains(err.Error(), "future bookings") {
			// Get the future bookings to return to frontend
			bookings, fetchErr := h.dogRepo.GetFutureBookings(id)
			if fetchErr != nil {
				respondError(w, http.StatusInternalServerError, "Failed to fetch bookings")
				return
			}

			// Return conflict with booking details
			respondJSON(w, http.StatusConflict, map[string]interface{}{
				"error":    "Hund hat zukünftige Buchungen",
				"bookings": bookings,
			})
		} else {
			respondError(w, http.StatusInternalServerError, "Failed to delete dog")
		}
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Hund erfolgreich gelöscht",
	})
}

// UploadDogPhoto handles POST /api/dogs/:id/photo - upload dog photo (admin only)
func (h *DogHandler) UploadDogPhoto(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid dog ID")
		return
	}

	// Get existing dog
	dog, err := h.dogRepo.FindByID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if dog == nil {
		respondError(w, http.StatusNotFound, "Dog not found")
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(int64(h.config.MaxUploadSizeMB) << 20); err != nil {
		respondError(w, http.StatusBadRequest, "File too large or invalid form")
		return
	}

	file, header, err := r.FormFile("photo")
	if err != nil {
		respondError(w, http.StatusBadRequest, "No file uploaded")
		return
	}
	defer file.Close()

	// Validate file extension (quick validation)
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		respondError(w, http.StatusBadRequest, "Only JPEG and PNG files are allowed")
		return
	}

	// Validate MIME type (magic bytes) to prevent file type spoofing
	if errMsg, valid := ValidateImageMIMEType(file); !valid {
		respondError(w, http.StatusBadRequest, errMsg)
		return
	}
	// Reset file reader position after MIME check
	file.Seek(0, 0)

	// Delete old photos if they exist (before processing new ones)
	if dog.Photo != nil && *dog.Photo != "" {
		// Use ImageService to delete both full and thumbnail
		// This handles the new naming scheme (dog_{id}_full.jpg, dog_{id}_thumb.jpg)
		h.imageService.DeleteDogPhotos(id)

		// Also try to delete old photo with original naming scheme (backward compatibility)
		oldPath := filepath.Join(h.config.UploadDir, *dog.Photo)
		os.Remove(oldPath) // Ignore errors if file doesn't exist
	}

	var fullPath, thumbPath string

	// SaaS: Use S3 storage if enabled
	if h.s3Service != nil && h.config.UseS3 {
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
		fullObjectPath := fmt.Sprintf("dogs/dog_%d_full%s", id, ext)
		fullURL, err := h.s3Service.Upload(context.Background(), tenantSlug, fullObjectPath, fileData, contentType)
		if err != nil {
			log.Printf("Failed to upload dog photo to S3: %v", err)
			respondError(w, http.StatusInternalServerError, "Failed to upload photo")
			return
		}

		// For S3, we store the full URL in the database
		fullPath = fullURL
		thumbPath = fullURL // For now, use same URL for thumbnail (S3 doesn't auto-resize)

		log.Printf("Uploaded dog %d photo to S3: %s", id, fullURL)
	} else {
		// Local filesystem storage
		// Process the uploaded photo (resize, compress, create thumbnail)
		fullPath, thumbPath, err = h.imageService.ProcessDogPhoto(file, id)
		if err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to process image: %v", err))
			return
		}
	}

	// Update dog with new photo paths
	dog.Photo = &fullPath
	dog.PhotoThumbnail = &thumbPath

	if err := h.dogRepo.Update(dog); err != nil {
		// If database update fails, clean up the newly created files (only for local storage)
		if h.s3Service == nil || !h.config.UseS3 {
			h.imageService.DeleteDogPhotos(id)
		}
		respondError(w, http.StatusInternalServerError, "Failed to update dog")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":   "Photo uploaded successfully",
		"photo":     fullPath,
		"thumbnail": thumbPath,
	})
}

// ToggleAvailability handles PUT /api/dogs/:id/availability - toggle availability (admin only)
func (h *DogHandler) ToggleAvailability(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid dog ID")
		return
	}

	var req models.ToggleAvailabilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// If marking as unavailable, reason is optional but recommended
	if !req.IsAvailable && (req.UnavailableReason == nil || *req.UnavailableReason == "") {
		defaultReason := "Temporarily unavailable"
		req.UnavailableReason = &defaultReason
	}

	// Toggle availability
	if err := h.dogRepo.ToggleAvailability(id, req.IsAvailable, req.UnavailableReason); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to toggle availability")
		return
	}

	// Get updated dog
	dog, err := h.dogRepo.FindByID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch updated dog")
		return
	}

	respondJSON(w, http.StatusOK, dog)
}

// GetBreeds handles GET /api/dogs/breeds - get list of all breeds
func (h *DogHandler) GetBreeds(w http.ResponseWriter, r *http.Request) {
	breeds, err := h.dogRepo.GetBreeds()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch breeds")
		return
	}

	respondJSON(w, http.StatusOK, breeds)
}

// GetFeaturedDogs handles GET /api/dogs/featured - get featured dogs for homepage (public)
func (h *DogHandler) GetFeaturedDogs(w http.ResponseWriter, r *http.Request) {
	// SaaS: Get tenant_id from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	dogs, err := h.dogRepo.GetFeatured(tenantID)
	if err != nil {
		log.Printf("Error fetching featured dogs: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to fetch featured dogs")
		return
	}

	respondJSON(w, http.StatusOK, dogs)
}

// SetFeatured handles PUT /api/dogs/:id/featured - set featured status (admin only)
func (h *DogHandler) SetFeatured(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid dog ID")
		return
	}

	// Parse request body
	var req struct {
		IsFeatured bool `json:"is_featured"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Check if dog exists
	dog, err := h.dogRepo.FindByID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if dog == nil {
		respondError(w, http.StatusNotFound, "Dog not found")
		return
	}

	// Note: No limit on featured dogs - frontend randomly displays 3 from all featured
	// This gives all featured dogs a chance to be shown to visitors

	// Update featured status
	if err := h.dogRepo.SetFeatured(id, req.IsFeatured); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update featured status")
		return
	}

	// Get updated dog
	dog, err = h.dogRepo.FindByID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch updated dog")
		return
	}

	respondJSON(w, http.StatusOK, dog)
}
