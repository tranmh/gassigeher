package handlers

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tranm/gassigeher/internal/config"
	"github.com/tranm/gassigeher/internal/middleware"
	"github.com/tranm/gassigeher/internal/models"
	"github.com/tranm/gassigeher/internal/repository"
	"github.com/tranm/gassigeher/internal/services"
)

// UserHandler handles user-related endpoints
type UserHandler struct {
	userRepo     *repository.UserRepository
	authService  *services.AuthService
	emailService *services.EmailService
	config       *config.Config
}

// NewUserHandler creates a new user handler
func NewUserHandler(db *sql.DB, cfg *config.Config) *UserHandler {
	emailService, err := services.NewEmailService(
		cfg.GmailClientID,
		cfg.GmailClientSecret,
		cfg.GmailRefreshToken,
		cfg.GmailFromEmail,
	)
	if err != nil {
		println("Warning: Failed to initialize email service:", err.Error())
	}

	return &UserHandler{
		userRepo:     repository.NewUserRepository(db),
		authService:  services.NewAuthService(cfg.JWTSecret, cfg.JWTExpirationHours),
		emailService: emailService,
		config:       cfg,
	}
}

// GetMe returns the current user's profile
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if user == nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Don't return sensitive data
	user.PasswordHash = nil
	user.VerificationToken = nil
	user.PasswordResetToken = nil

	respondJSON(w, http.StatusOK, user)
}

// UpdateMe updates the current user's profile
func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if user == nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Track if email changed
	emailChanged := false

	// Update fields
	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		user.Name = *req.Name
	}

	if req.Phone != nil && strings.TrimSpace(*req.Phone) != "" {
		user.Phone = req.Phone
	}

	// Handle email change - requires re-verification
	if req.Email != nil && strings.TrimSpace(*req.Email) != "" {
		newEmail := strings.TrimSpace(*req.Email)

		// Check if email actually changed
		if user.Email != nil && *user.Email != newEmail {
			// Check if new email already exists
			existingUser, err := h.userRepo.FindByEmail(newEmail)
			if err != nil {
				respondError(w, http.StatusInternalServerError, "Database error")
				return
			}
			if existingUser != nil {
				respondError(w, http.StatusConflict, "Email already in use")
				return
			}

			// Generate new verification token
			token, err := h.authService.GenerateToken()
			if err != nil {
				respondError(w, http.StatusInternalServerError, "Failed to generate token")
				return
			}

			user.Email = &newEmail
			user.VerificationToken = &token
			user.IsVerified = false
			emailChanged = true

			// Set token expiration
			expires := time.Now().Add(24 * time.Hour)
			user.VerificationTokenExpires = &expires
		}
	}

	if err := h.userRepo.Update(user); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update profile")
		return
	}

	// Send verification email if email changed
	if emailChanged && user.Email != nil {
		go h.emailService.SendVerificationEmail(*user.Email, user.Name, *user.VerificationToken)
	}

	// Don't return sensitive data
	user.PasswordHash = nil
	user.VerificationToken = nil
	user.PasswordResetToken = nil

	message := "Profile updated successfully"
	if emailChanged {
		message = "Profile updated. Please check your new email to verify it."
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": message,
		"user":    user,
	})
}

// UploadPhoto handles profile photo upload
func (h *UserHandler) UploadPhoto(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
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

	// Validate file type
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		respondError(w, http.StatusBadRequest, "Only JPEG and PNG files are allowed")
		return
	}

	// Create upload directory if it doesn't exist
	userDir := filepath.Join(h.config.UploadDir, "users")
	if err := os.MkdirAll(userDir, 0755); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create upload directory")
		return
	}

	// Generate filename
	filename := filepath.Join("users", filepath.Base(header.Filename))
	destPath := filepath.Join(h.config.UploadDir, filename)

	// Save file
	dest, err := os.Create(destPath)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to save file")
		return
	}
	defer dest.Close()

	if _, err := io.Copy(dest, file); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to save file")
		return
	}

	// Update user profile
	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if user == nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Delete old photo if exists
	if user.ProfilePhoto != nil && *user.ProfilePhoto != "" {
		oldPath := filepath.Join(h.config.UploadDir, *user.ProfilePhoto)
		os.Remove(oldPath) // Ignore errors
	}

	user.ProfilePhoto = &filename
	if err := h.userRepo.Update(user); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update profile")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Photo uploaded successfully",
		"photo":   filename,
	})
}
