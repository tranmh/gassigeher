package handlers

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
	"github.com/tranmh/gassigeher/internal/services"
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
	emailService, err := services.NewEmailService(services.ConfigToEmailConfig(cfg))
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

	// Get admin status from context
	isAdmin, _ := r.Context().Value(middleware.IsAdminKey).(bool)

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

	// Create response with user data + is_admin flag
	// Keep user fields at top level for backward compatibility
	type UserResponse struct {
		*models.User
		IsAdmin bool `json:"is_admin"`
	}

	response := &UserResponse{
		User:    user,
		IsAdmin: isAdmin,
	}

	respondJSON(w, http.StatusOK, response)
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

	// Validate input (includes phone number validation)
	if err := req.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
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

	// Update fields (Note: FirstName and LastName can only be edited by admins)
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
	if emailChanged && user.Email != nil && h.emailService != nil {
		go h.emailService.SendVerificationEmail(*user.Email, user.FirstName, *user.VerificationToken)
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

// DeleteAccount deletes the current user's account (GDPR anonymization)
func (h *UserHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse request to get password confirmation
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Password == "" {
		respondError(w, http.StatusBadRequest, "Password is required to confirm deletion")
		return
	}

	// Get user
	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if user == nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Verify password
	if user.PasswordHash == nil || !h.authService.CheckPassword(req.Password, *user.PasswordHash) {
		respondError(w, http.StatusUnauthorized, "Invalid password")
		return
	}

	// Store email for confirmation before deletion
	var emailForConfirmation string
	if user.Email != nil {
		emailForConfirmation = *user.Email
	}

	// Delete account (GDPR anonymization)
	if err := h.userRepo.DeleteAccount(userID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete account")
		return
	}

	// Send confirmation email to original email
	if emailForConfirmation != "" && h.emailService != nil {
		go h.emailService.SendAccountDeletionConfirmation(emailForConfirmation, user.FirstName)
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Account deleted successfully"})
}

// ListUsers lists all users (admin only)
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	// Parse filters
	var activeOnly *bool
	if activeParam := r.URL.Query().Get("active"); activeParam != "" {
		active := activeParam == "true" || activeParam == "1"
		activeOnly = &active
	}

	users, err := h.userRepo.FindAll(activeOnly)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get users")
		return
	}

	// Don't return sensitive data
	for _, user := range users {
		user.PasswordHash = nil
		user.VerificationToken = nil
		user.PasswordResetToken = nil
	}

	respondJSON(w, http.StatusOK, users)
}

// GetUser gets a user by ID (admin only)
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	// Get user ID from URL
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid user ID")
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

// DeactivateUser deactivates a user account (admin only)
func (h *UserHandler) DeactivateUser(w http.ResponseWriter, r *http.Request) {
	// Get user ID from URL
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Parse request
	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Reason == "" {
		respondError(w, http.StatusBadRequest, "Reason is required")
		return
	}

	// Get user
	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if user == nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Deactivate
	if err := h.userRepo.Deactivate(userID, req.Reason); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to deactivate user")
		return
	}

	// Send email notification
	if user.Email != nil && h.emailService != nil {
		go h.emailService.SendAccountDeactivated(*user.Email, user.FirstName, req.Reason)
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "User deactivated successfully"})
}

// ActivateUser activates a user account (admin only)
func (h *UserHandler) ActivateUser(w http.ResponseWriter, r *http.Request) {
	// Get user ID from URL
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Parse optional message
	var req struct {
		Message *string `json:"message,omitempty"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	// Get user
	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if user == nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Activate
	if err := h.userRepo.Activate(userID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to activate user")
		return
	}

	// Send email notification
	if user.Email != nil && h.emailService != nil {
		go h.emailService.SendAccountReactivated(*user.Email, user.FirstName, req.Message)
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "User activated successfully"})
}

// PromoteToAdmin promotes a user to admin role (Super Admin only)
// DONE: Phase 4
func (h *UserHandler) PromoteToAdmin(w http.ResponseWriter, r *http.Request) {
	// Extract super admin from context (middleware already verified)
	isSuperAdmin, _ := r.Context().Value(middleware.IsSuperAdminKey).(bool)
	if !isSuperAdmin {
		respondError(w, http.StatusForbidden, "Only Super Admin can promote users")
		return
	}

	// Get user ID from URL
	vars := mux.Vars(r)
	userIDStr := vars["id"]
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Get target user
	targetUser, err := h.userRepo.FindByID(userID)
	if err != nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}
	if targetUser == nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Validation checks
	if targetUser.IsSuperAdmin {
		respondError(w, http.StatusBadRequest, "Cannot modify Super Admin")
		return
	}

	if targetUser.IsAdmin {
		respondError(w, http.StatusBadRequest, "User is already an admin")
		return
	}

	// Promote user
	err = h.userRepo.PromoteToAdmin(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to promote user")
		return
	}

	// Get updated user
	updatedUser, err := h.userRepo.FindByID(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to retrieve updated user")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "User promoted to admin successfully",
		"user":    updatedUser,
	})
}

// DemoteAdmin revokes admin privileges (Super Admin only)
// DONE: Phase 4
func (h *UserHandler) DemoteAdmin(w http.ResponseWriter, r *http.Request) {
	// Extract super admin from context
	isSuperAdmin, _ := r.Context().Value(middleware.IsSuperAdminKey).(bool)
	if !isSuperAdmin {
		respondError(w, http.StatusForbidden, "Only Super Admin can demote admins")
		return
	}

	// Get user ID from URL
	vars := mux.Vars(r)
	userIDStr := vars["id"]
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Get target user
	targetUser, err := h.userRepo.FindByID(userID)
	if err != nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}
	if targetUser == nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Validation checks
	if targetUser.IsSuperAdmin {
		respondError(w, http.StatusBadRequest, "Cannot demote Super Admin")
		return
	}

	if !targetUser.IsAdmin {
		respondError(w, http.StatusBadRequest, "User is not an admin")
		return
	}

	// Demote user
	err = h.userRepo.DemoteAdmin(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to demote admin")
		return
	}

	// Get updated user
	updatedUser, err := h.userRepo.FindByID(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to retrieve updated user")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Admin privileges revoked successfully",
		"user":    updatedUser,
	})
}

// AdminUpdateUser allows admins to update user profiles (including names)
func (h *UserHandler) AdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	// Get user ID from URL
	vars := mux.Vars(r)
	userIDStr := vars["id"]
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Parse request body
	var req models.AdminUpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Get target user
	targetUser, err := h.userRepo.FindByID(userID)
	if err != nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}
	if targetUser == nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Cannot edit deleted users
	if targetUser.IsDeleted {
		respondError(w, http.StatusBadRequest, "Cannot edit deleted user")
		return
	}

	// Apply updates
	if req.FirstName != nil {
		targetUser.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		targetUser.LastName = *req.LastName
	}
	if req.Email != nil {
		// Check if email is already taken by another user
		existingUser, _ := h.userRepo.FindByEmail(*req.Email)
		if existingUser != nil && existingUser.ID != userID {
			respondError(w, http.StatusConflict, "E-Mail wird bereits verwendet")
			return
		}
		targetUser.Email = req.Email
	}
	if req.Phone != nil {
		targetUser.Phone = req.Phone
	}

	// Save updates
	err = h.userRepo.Update(targetUser)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	// Get updated user
	updatedUser, err := h.userRepo.FindByID(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to retrieve updated user")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "User updated successfully",
		"user":    updatedUser,
	})
}

// DONE: Phase 4 - Handler methods complete
