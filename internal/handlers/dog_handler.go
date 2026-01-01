package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/database"
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
	subscriptionRepo *repository.SubscriptionRepository  // SaaS: For checking dog limits
	colorRepo        *repository.ColorCategoryRepository // For resolving legacy category to color_id
	imageService     *services.ImageService
	emailService     *services.EmailService
	s3Service        *services.S3Service // SaaS: For S3 storage
	config           *config.Config
}

// NewDogHandler creates a new dog handler
func NewDogHandler(db *database.DB, cfg *config.Config) *DogHandler {
	// Handle nil config gracefully
	if cfg == nil {
		log.Printf("WARNING: DogHandler created with nil config, using defaults")
		cfg = &config.Config{}
	}

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
		subscriptionRepo: repository.NewSubscriptionRepository(db),  // SaaS: For dog limit checks
		colorRepo:        repository.NewColorCategoryRepository(db), // For resolving legacy category to color_id
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

	// SaaS: Get tenant_id from context for isolation
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)

	// SaaS SECURITY: Require valid tenant context (block tenantID=0 bypass)
	if !ok {
		respondError(w, http.StatusNotFound, "Dog not found")
		return
	}

	dog, err := h.dogRepo.FindByIDAndTenant(id, tenantID)
	if err != nil {
		if isNotFoundOrTenantError(err) {
			respondError(w, http.StatusNotFound, "Dog not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	respondJSON(w, http.StatusOK, dog)
}

// CreateDog handles POST /api/dogs - create a new dog (admin only)
func (h *DogHandler) CreateDog(w http.ResponseWriter, r *http.Request) {
	var req models.CreateDogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// SaaS: Get tenant_id from context (tenant_id=0 is valid for Simple-Mode)
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)

	// Get dog limit from subscription (will be used for atomic create)
	// tenant_id=0 (Simple-Mode) can also have subscription limits
	var dogLimit int = -1 // Default unlimited if no subscription
	if ok {
		var err error
		dogLimit, err = h.subscriptionRepo.GetTenantDogLimit(tenantID)
		if err != nil {
			log.Printf("ERROR: Failed to get dog limit for tenant %d: %v", tenantID, err)
			respondError(w, http.StatusInternalServerError, "Failed to check subscription")
			return
		}
	}

	// Validate and sanitize required fields (prevents XSS and enforces length limits)
	sanitizedName, valErr := ValidateDogName(req.Name)
	if valErr != nil {
		respondError(w, http.StatusBadRequest, valErr.Message)
		return
	}

	sanitizedBreed, valErr := ValidateDogBreed(req.Breed)
	if valErr != nil {
		respondError(w, http.StatusBadRequest, valErr.Message)
		return
	}

	if req.Size != "small" && req.Size != "medium" && req.Size != "large" {
		respondError(w, http.StatusBadRequest, "Size must be small, medium, or large")
		return
	}

	// Validate age is not negative
	if req.Age < 0 {
		respondError(w, http.StatusBadRequest, "Age cannot be negative")
		return
	}

	// Validate and sanitize optional fields
	sanitizedSpecialNeeds, valErr := ValidateDogSpecialNeeds(req.SpecialNeeds)
	if valErr != nil {
		respondError(w, http.StatusBadRequest, valErr.Message)
		return
	}

	sanitizedPickupLocation, valErr := ValidateDogPickupLocation(req.PickupLocation)
	if valErr != nil {
		respondError(w, http.StatusBadRequest, valErr.Message)
		return
	}

	sanitizedWalkRoute, valErr := ValidateDogWalkRoute(req.WalkRoute)
	if valErr != nil {
		respondError(w, http.StatusBadRequest, valErr.Message)
		return
	}

	sanitizedSpecialInstructions, valErr := ValidateDogSpecialInstructions(req.SpecialInstructions)
	if valErr != nil {
		respondError(w, http.StatusBadRequest, valErr.Message)
		return
	}

	sanitizedExternalLink, valErr := ValidateDogExternalLink(req.ExternalLink)
	if valErr != nil {
		respondError(w, http.StatusBadRequest, valErr.Message)
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

	// Create dog with sanitized values
	dog := &models.Dog{
		TenantID:            tenantID, // SaaS: Set tenant_id from context
		Name:                sanitizedName,
		Breed:               sanitizedBreed,
		Size:                req.Size,
		Age:                 req.Age,
		Category:            category,
		ColorID:             req.ColorID,
		SpecialNeeds:        sanitizedSpecialNeeds,
		PickupLocation:      sanitizedPickupLocation,
		WalkRoute:           sanitizedWalkRoute,
		WalkDuration:        req.WalkDuration,
		SpecialInstructions: sanitizedSpecialInstructions,
		DefaultMorningTime:  req.DefaultMorningTime,
		DefaultEveningTime:  req.DefaultEveningTime,
		ExternalLink:        sanitizedExternalLink,
		IsAvailable:         true, // Default to available
	}

	// SaaS: Use atomic create with limit check to prevent race conditions
	if err := h.dogRepo.CreateWithLimitCheck(dog, dogLimit); err != nil {
		if err == repository.ErrDogLimitExceeded {
			// Get current count for error response
			currentCount, _ := h.dogRepo.CountByTenant(tenantID)
			respondJSON(w, http.StatusConflict, map[string]interface{}{
				"error":         "Hundelimit erreicht",
				"message":       fmt.Sprintf("Sie haben das Maximum von %d Hunden für Ihren Plan erreicht. Bitte upgraden Sie auf Pro für unbegrenzte Hunde.", dogLimit),
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

	// SaaS: Get tenant_id from context for isolation
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)

	// SaaS SECURITY: Require valid tenant context (block tenantID=0 bypass)
	if !ok {
		respondError(w, http.StatusNotFound, "Dog not found")
		return
	}

	// Get existing dog with tenant verification
	dog, err := h.dogRepo.FindByIDAndTenant(id, tenantID)
	if err != nil {
		if isNotFoundOrTenantError(err) {
			respondError(w, http.StatusNotFound, "Dog not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Parse update request
	var req models.UpdateDogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Update fields if provided (with validation and XSS sanitization)
	if req.Name != nil {
		sanitizedName, valErr := ValidateDogName(*req.Name)
		if valErr != nil {
			respondError(w, http.StatusBadRequest, valErr.Message)
			return
		}
		dog.Name = sanitizedName
	}
	if req.Breed != nil {
		sanitizedBreed, valErr := ValidateDogBreed(*req.Breed)
		if valErr != nil {
			respondError(w, http.StatusBadRequest, valErr.Message)
			return
		}
		dog.Breed = sanitizedBreed
	}
	if req.Size != nil {
		dog.Size = *req.Size
	}
	if req.Age != nil {
		if *req.Age < 0 {
			respondError(w, http.StatusBadRequest, "Age cannot be negative")
			return
		}
		dog.Age = *req.Age
	}
	if req.Category != nil {
		dog.Category = *req.Category
	}
	if req.ColorID != nil {
		dog.ColorID = req.ColorID
	}
	if req.SpecialNeeds != nil {
		sanitized, valErr := ValidateDogSpecialNeeds(req.SpecialNeeds)
		if valErr != nil {
			respondError(w, http.StatusBadRequest, valErr.Message)
			return
		}
		dog.SpecialNeeds = sanitized
	}
	if req.PickupLocation != nil {
		sanitized, valErr := ValidateDogPickupLocation(req.PickupLocation)
		if valErr != nil {
			respondError(w, http.StatusBadRequest, valErr.Message)
			return
		}
		dog.PickupLocation = sanitized
	}
	if req.WalkRoute != nil {
		sanitized, valErr := ValidateDogWalkRoute(req.WalkRoute)
		if valErr != nil {
			respondError(w, http.StatusBadRequest, valErr.Message)
			return
		}
		dog.WalkRoute = sanitized
	}
	if req.WalkDuration != nil {
		dog.WalkDuration = req.WalkDuration
	}
	if req.SpecialInstructions != nil {
		sanitized, valErr := ValidateDogSpecialInstructions(req.SpecialInstructions)
		if valErr != nil {
			respondError(w, http.StatusBadRequest, valErr.Message)
			return
		}
		dog.SpecialInstructions = sanitized
	}
	if req.DefaultMorningTime != nil {
		dog.DefaultMorningTime = req.DefaultMorningTime
	}
	if req.DefaultEveningTime != nil {
		dog.DefaultEveningTime = req.DefaultEveningTime
	}
	if req.ExternalLink != nil {
		sanitized, valErr := ValidateDogExternalLink(req.ExternalLink)
		if valErr != nil {
			respondError(w, http.StatusBadRequest, valErr.Message)
			return
		}
		dog.ExternalLink = sanitized
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

	// SaaS SECURITY: Get tenant_id from context for cross-tenant check
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)

	// SaaS SECURITY: Require valid tenant context (block tenantID=0 bypass)
	if !ok {
		respondError(w, http.StatusNotFound, "Dog not found")
		return
	}

	// Check if force delete is requested
	force := r.URL.Query().Get("force") == "true"

	if force {
		// Force delete: cancel all future bookings and delete dog
		dog, err := h.dogRepo.FindByIDAndTenant(id, tenantID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to fetch dog")
			return
		}
		if dog == nil {
			respondError(w, http.StatusNotFound, "Dog not found")
			return
		}

		// Get all future bookings (filtered by tenantID for SaaS isolation)
		bookings, err := h.dogRepo.GetFutureBookings(id, tenantID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to fetch bookings")
			return
		}

		// Cancel all future bookings
		cancellationReason := fmt.Sprintf("Hund %s wurde aus dem System entfernt", dog.Name)
		for _, booking := range bookings {
			// Cancel the booking
			err := h.bookingRepo.Cancel(booking.ID, tenantID, &cancellationReason)
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
		if err := h.dogRepo.ForceDelete(id, tenantID); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to delete dog")
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"message":         "Hund erfolgreich gelöscht",
			"cancelled_count": len(bookings),
		})
		return
	}

	// Normal delete (will fail if future bookings exist)
	dog, err := h.dogRepo.FindByIDAndTenant(id, tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch dog")
		return
	}
	if dog == nil {
		respondError(w, http.StatusNotFound, "Dog not found")
		return
	}

	err = h.dogRepo.Delete(id, tenantID)
	if err != nil {
		if strings.Contains(err.Error(), "future bookings") {
			// Get the future bookings to return to frontend (filtered by tenantID for SaaS isolation)
			bookings, fetchErr := h.dogRepo.GetFutureBookings(id, tenantID)
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

	// SaaS SECURITY: Get tenant_id from context and validate presence
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok {
		// BUG FIX: Tenant context is missing (middleware bypass or misconfiguration)
		respondError(w, http.StatusInternalServerError, "Request validation failed")
		return
	}

	// Get existing dog with tenant verification
	dog, err := h.dogRepo.FindByIDAndTenant(id, tenantID)
	if err != nil {
		// BUG FIX: Handle ErrNotFound and ErrTenantMismatch properly
		if isNotFoundOrTenantError(err) {
			respondError(w, http.StatusNotFound, "Dog not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if dog == nil {
		respondError(w, http.StatusNotFound, "Dog not found")
		return
	}

	// SECURITY: Limit request body size to prevent DoS attacks
	// MaxBytesReader prevents reading more than maxSize bytes, returning error immediately
	maxSizeMB := h.config.MaxUploadSizeMB
	if maxSizeMB <= 0 {
		maxSizeMB = 10 // Default 10MB if not configured
	}
	maxSize := int64(maxSizeMB) << 20 // Convert MB to bytes
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	// Parse multipart form
	if err := r.ParseMultipartForm(maxSize); err != nil {
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
	if _, err := file.Seek(0, 0); err != nil {
		log.Printf("Error seeking file: %v", err)
		respondError(w, http.StatusInternalServerError, "Fehler beim Verarbeiten der Datei")
		return
	}

	// BUG FIX: Capture old photo path BEFORE processing new photo
	// We'll delete old photos AFTER successfully processing the new one
	var oldPhotoPath string
	if dog.Photo != nil && *dog.Photo != "" {
		oldPhotoPath = *dog.Photo
	}

	var fullPath, thumbPath string

	// Use ImageService for BOTH local and S3 storage
	// ImageService handles resizing, compression, and thumbnail generation correctly
	var imageService *services.ImageService
	if h.s3Service != nil && h.config.UseS3 {
		// S3 mode: Create tenant-aware ImageService with S3 support
		tenantSlug, _ := r.Context().Value(middleware.TenantSlugKey).(string)
		if tenantSlug == "" {
			tenantSlug = "default"
		}
		imageService = services.NewImageServiceWithS3(h.config.UploadDir, h.s3Service, tenantSlug)
		log.Printf("Using S3 storage for dog %d photo upload (tenant: %s)", id, tenantSlug)
	} else {
		// Local storage: Use existing imageService
		imageService = h.imageService
	}

	// Process the uploaded photo (resize, compress, create thumbnail)
	// This works for BOTH local filesystem and S3 storage
	fullPath, thumbPath, err = imageService.ProcessDogPhoto(file, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to process image: %v", err))
		return
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

	// BUG FIX: Delete old photos ONLY after successfully processing and saving new photo
	// This prevents data loss if new photo processing fails
	// Note: If the old photo used the new naming scheme (dog_{id}_full.jpg), it has already
	// been overwritten by the new photo. We only need to clean up photos with old naming scheme.
	if oldPhotoPath != "" && oldPhotoPath != fullPath {
		// Only delete if old path differs from new path (old naming scheme)
		oldFullPath := filepath.Join(h.config.UploadDir, oldPhotoPath)
		os.Remove(oldFullPath) // Ignore errors if file doesn't exist
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":   "Photo uploaded successfully",
		"photo":     fullPath,
		"thumbnail": thumbPath,
	})
}

// DeleteDogPhoto handles DELETE /api/dogs/:id/photo - delete dog photo (admin only)
func (h *DogHandler) DeleteDogPhoto(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid dog ID")
		return
	}

	// SaaS: Get tenant_id from context for isolation
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)

	// SaaS SECURITY: Require valid tenant context (block tenantID=0 bypass)
	if !ok {
		respondError(w, http.StatusNotFound, "Dog not found")
		return
	}

	// Get existing dog with tenant verification
	dog, err := h.dogRepo.FindByIDAndTenant(id, tenantID)
	if err != nil {
		// BUG FIX: Handle ErrNotFound and ErrTenantMismatch properly
		if isNotFoundOrTenantError(err) {
			respondError(w, http.StatusNotFound, "Dog not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if dog == nil {
		respondError(w, http.StatusNotFound, "Dog not found")
		return
	}

	// Check if dog has a photo
	if dog.Photo == nil || *dog.Photo == "" {
		respondError(w, http.StatusNotFound, "Dog has no photo to delete")
		return
	}

	// Delete photo files (handles both S3 and local storage)
	if h.s3Service != nil && h.config.UseS3 {
		// S3 deletion: Use s3Service directly (imageService doesn't have S3 configured)
		tenantSlug, ok := r.Context().Value(middleware.TenantSlugKey).(string)
		if !ok || tenantSlug == "" {
			// This should not happen - middleware should always set tenant slug
			// Log warning and use safe fallback to prevent nil pointer
			log.Printf("WARNING: Missing tenant slug in S3 delete for dog %d, using 'default'", id)
			tenantSlug = "default"
		}

		ctx := r.Context()

		// Try all possible extensions since upload preserves original extension
		extensions := []string{".jpg", ".jpeg", ".png"}
		var deleteErrors []string
		for _, ext := range extensions {
			fullPath := fmt.Sprintf("dogs/dog_%d_full%s", id, ext)
			thumbPath := fmt.Sprintf("dogs/dog_%d_thumb%s", id, ext)

			// Delete from S3 - log errors but continue (file may not exist with this extension)
			if err := h.s3Service.DeleteByPath(ctx, tenantSlug, fullPath); err != nil {
				deleteErrors = append(deleteErrors, fmt.Sprintf("%s: %v", fullPath, err))
			}
			if err := h.s3Service.DeleteByPath(ctx, tenantSlug, thumbPath); err != nil {
				deleteErrors = append(deleteErrors, fmt.Sprintf("%s: %v", thumbPath, err))
			}
		}

		// Audit log: record photo deletion
		if len(deleteErrors) > 0 {
			log.Printf("AUDIT: Dog %d photo deletion in tenant %s had errors: %v", id, tenantSlug, deleteErrors)
		} else {
			log.Printf("AUDIT: Dog %d photos deleted successfully in tenant %s", id, tenantSlug)
		}
	} else {
		// Local filesystem deletion
		h.imageService.DeleteDogPhotos(id)
	}

	// Update database to clear photo fields
	dog.Photo = nil
	dog.PhotoThumbnail = nil

	if err := h.dogRepo.Update(dog); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update dog")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Foto erfolgreich gelöscht",
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

	// SaaS: Get tenant_id from context for isolation
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)

	// SaaS SECURITY: Require valid tenant context (block tenantID=0 bypass)
	if !ok {
		respondError(w, http.StatusNotFound, "Dog not found")
		return
	}

	dog, err := h.dogRepo.FindByIDAndTenant(id, tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if dog == nil {
		respondError(w, http.StatusNotFound, "Dog not found")
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
	if err := h.dogRepo.ToggleAvailability(id, tenantID, req.IsAvailable, req.UnavailableReason); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to toggle availability")
		return
	}

	// Get updated dog (tenant already verified above)
	dog, err = h.dogRepo.FindByIDAndTenant(id, tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch updated dog")
		return
	}

	respondJSON(w, http.StatusOK, dog)
}

// GetBreeds handles GET /api/dogs/breeds - get list of all breeds
func (h *DogHandler) GetBreeds(w http.ResponseWriter, r *http.Request) {
	// SaaS: Get tenant_id from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	breeds, err := h.dogRepo.GetBreeds(tenantID)
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

	// SaaS: Get tenant_id from context for isolation
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)

	// SaaS SECURITY: Require valid tenant context (block tenantID=0 bypass)
	if !ok {
		respondError(w, http.StatusNotFound, "Dog not found")
		return
	}

	// Check if dog exists and belongs to tenant
	dog, err := h.dogRepo.FindByIDAndTenant(id, tenantID)
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
	if err := h.dogRepo.SetFeatured(id, tenantID, req.IsFeatured); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update featured status")
		return
	}

	// Get updated dog (tenant already verified above)
	dog, err = h.dogRepo.FindByIDAndTenant(id, tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch updated dog")
		return
	}

	respondJSON(w, http.StatusOK, dog)
}
