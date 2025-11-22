package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/tranm/gassigeher/internal/config"
	"github.com/tranm/gassigeher/internal/middleware"
	"github.com/tranm/gassigeher/internal/models"
	"github.com/tranm/gassigeher/internal/repository"
	"github.com/tranm/gassigeher/internal/services"
)

// DogHandler handles dog-related endpoints
type DogHandler struct {
	dogRepo      *repository.DogRepository
	userRepo     *repository.UserRepository
	imageService *services.ImageService
	config       *config.Config
}

// NewDogHandler creates a new dog handler
func NewDogHandler(db *sql.DB, cfg *config.Config) *DogHandler {
	return &DogHandler{
		dogRepo:      repository.NewDogRepository(db),
		userRepo:     repository.NewUserRepository(db),
		imageService: services.NewImageService(cfg.UploadDir),
		config:       cfg,
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

	if available := r.URL.Query().Get("available"); available != "" {
		avail := available == "true" || available == "1"
		filter.Available = &avail
	}

	if search := r.URL.Query().Get("search"); search != "" {
		filter.Search = &search
	}

	// Get dogs
	dogs, err := h.dogRepo.FindAll(filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch dogs")
		return
	}

	// If user is authenticated, filter based on their experience level
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if ok {
		user, err := h.userRepo.FindByID(userID)
		if err == nil && user != nil {
			filteredDogs := []*models.Dog{}
			for _, dog := range dogs {
				// Check if user can access this dog
				if repository.CanUserAccessDog(user.ExperienceLevel, dog.Category) {
					filteredDogs = append(filteredDogs, dog)
				} else {
					// Include but mark as inaccessible (frontend will handle display)
					filteredDogs = append(filteredDogs, dog)
				}
			}
			dogs = filteredDogs
		}
	}

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

// CreateDog handles POST /api/dogs - create a new dog (admin only)
func (h *DogHandler) CreateDog(w http.ResponseWriter, r *http.Request) {
	var req models.CreateDogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
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

	if req.Category != "green" && req.Category != "blue" && req.Category != "orange" {
		respondError(w, http.StatusBadRequest, "Category must be green, blue, or orange")
		return
	}

	// Create dog
	dog := &models.Dog{
		Name:                req.Name,
		Breed:               req.Breed,
		Size:                req.Size,
		Age:                 req.Age,
		Category:            req.Category,
		SpecialNeeds:        req.SpecialNeeds,
		PickupLocation:      req.PickupLocation,
		WalkRoute:           req.WalkRoute,
		WalkDuration:        req.WalkDuration,
		SpecialInstructions: req.SpecialInstructions,
		DefaultMorningTime:  req.DefaultMorningTime,
		DefaultEveningTime:  req.DefaultEveningTime,
		IsAvailable:         true, // Default to available
	}

	if err := h.dogRepo.Create(dog); err != nil {
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

	// Delete (will fail if future bookings exist)
	if err := h.dogRepo.Delete(id); err != nil {
		if strings.Contains(err.Error(), "future bookings") {
			respondError(w, http.StatusConflict, "Cannot delete dog with future bookings")
		} else {
			respondError(w, http.StatusInternalServerError, "Failed to delete dog")
		}
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Dog deleted successfully",
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

	// Validate file type (checking extension first for quick validation)
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		respondError(w, http.StatusBadRequest, "Only JPEG and PNG files are allowed")
		return
	}

	// Delete old photos if they exist (before processing new ones)
	if dog.Photo != nil && *dog.Photo != "" {
		// Use ImageService to delete both full and thumbnail
		// This handles the new naming scheme (dog_{id}_full.jpg, dog_{id}_thumb.jpg)
		h.imageService.DeleteDogPhotos(id)

		// Also try to delete old photo with original naming scheme (backward compatibility)
		oldPath := filepath.Join(h.config.UploadDir, *dog.Photo)
		os.Remove(oldPath) // Ignore errors if file doesn't exist
	}

	// Process the uploaded photo (resize, compress, create thumbnail)
	fullPath, thumbPath, err := h.imageService.ProcessDogPhoto(file, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to process image: %v", err))
		return
	}

	// Update dog with new photo paths
	dog.Photo = &fullPath
	dog.PhotoThumbnail = &thumbPath

	if err := h.dogRepo.Update(dog); err != nil {
		// If database update fails, clean up the newly created files
		h.imageService.DeleteDogPhotos(id)
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
