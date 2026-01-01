package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/database"
	"github.com/tranmh/gassigeher/internal/logging"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
	"github.com/tranmh/gassigeher/internal/services"
)

// UserHandler handles user-related endpoints
type UserHandler struct {
	db            *database.DB
	userRepo      *repository.UserRepository
	userColorRepo *repository.UserColorRepository
	authService   *services.AuthService
	emailService  *services.EmailService
	imageService  *services.ImageService // For profile photo processing
	s3Service     *services.S3Service    // SaaS: For S3 storage
	config        *config.Config
}

// NewUserHandler creates a new user handler
func NewUserHandler(db *database.DB, cfg *config.Config) *UserHandler {
	emailService, err := services.NewEmailService(services.ConfigToEmailConfig(cfg))
	if err != nil {
		println("Warning: Failed to initialize email service:", err.Error())
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
			fmt.Printf("Warning: Failed to initialize S3 service in UserHandler: %v\n", err)
		} else {
			fmt.Printf("S3 storage enabled for UserHandler\n")
		}
	}

	return &UserHandler{
		db:            db,
		userRepo:      repository.NewUserRepository(db),
		userColorRepo: repository.NewUserColorRepository(db),
		authService:   services.NewAuthService(cfg.JWTSecret, cfg.JWTExpirationHours),
		emailService:  emailService,
		imageService:  services.NewImageService(cfg.UploadDir), // Basic service, tenant set per-request
		s3Service:     s3Service,
		config:        cfg,
	}
}

// getImageServiceForRequest returns an ImageService with the correct tenant slug for the request
// In SaaS-Mode, this includes tenant isolation. In Simple-Mode, returns the default service.
func (h *UserHandler) getImageServiceForRequest(r *http.Request) *services.ImageService {
	tenantSlug, _ := r.Context().Value(middleware.TenantSlugKey).(string)
	if tenantSlug != "" {
		// SaaS-Mode: Create tenant-aware service
		return services.NewImageServiceWithTenant(h.config.UploadDir, tenantSlug)
	}
	// Simple-Mode: Use default service (no tenant isolation)
	return h.imageService
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

	// Get impersonation status from context
	isImpersonating, _ := r.Context().Value(middleware.IsImpersonatingKey).(bool)
	originalUserID, _ := r.Context().Value(middleware.OriginalUserIDKey).(int)

	// SaaS: Get tenant_id from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

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

	// Fetch user's colors
	colorPtrs, err := h.userColorRepo.GetUserColors(tenantID, userID)
	if err != nil {
		log.Printf("Warning: Failed to get user colors: %v", err)
		colorPtrs = []*models.ColorCategory{}
	}
	// Convert []*ColorCategory to []ColorCategory
	user.Colors = make([]models.ColorCategory, len(colorPtrs))
	for i, c := range colorPtrs {
		if c != nil {
			user.Colors[i] = *c
		}
	}

	// Create response with user data + is_admin flag + impersonation info
	// Keep user fields at top level for backward compatibility
	type UserResponse struct {
		*models.User
		IsAdmin         bool `json:"is_admin"`
		IsImpersonating bool `json:"is_impersonating"`
		OriginalUserID  int  `json:"original_user_id,omitempty"`
	}

	response := &UserResponse{
		User:            user,
		IsAdmin:         isAdmin,
		IsImpersonating: isImpersonating,
		OriginalUserID:  originalUserID,
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
			// Check if new email already exists (within same tenant)
			existingUser, err := h.userRepo.FindByEmail(newEmail, user.TenantID)
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

	// Validate file extension
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

	// Get user first (needed for cleanup)
	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if user == nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	var photoPath string

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

		// Upload to S3
		objectPath := fmt.Sprintf("users/user_%d_profile%s", userID, ext)
		photoURL, err := h.s3Service.Upload(context.Background(), tenantSlug, objectPath, fileData, contentType)
		if err != nil {
			log.Printf("Failed to upload user photo to S3: %v", err)
			respondError(w, http.StatusInternalServerError, "Failed to upload photo")
			return
		}

		photoPath = photoURL
		log.Printf("Uploaded user %d photo to S3: %s", userID, photoURL)
	} else {
		// Local filesystem storage with tenant isolation
		// Get tenant-aware ImageService for this request
		imageService := h.getImageServiceForRequest(r)

		// Delete old photo if exists (before uploading new one)
		if user.ProfilePhoto != nil && *user.ProfilePhoto != "" {
			imageService.DeleteUserPhoto(userID)
		}

		// Process and save the photo using ImageService
		// This handles tenant isolation, unique filenames, and resizing
		var err error
		photoPath, err = imageService.ProcessUserPhoto(file, userID, ext)
		if err != nil {
			log.Printf("Failed to process user photo: %v", err)
			respondError(w, http.StatusInternalServerError, "Failed to save file")
			return
		}
	}

	user.ProfilePhoto = &photoPath
	if err := h.userRepo.Update(user); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update profile")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Photo uploaded successfully",
		"photo":   photoPath,
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

	// SaaS: Get tenant_id from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	users, err := h.userRepo.FindAll(activeOnly, tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get users")
		return
	}

	// Don't return sensitive data and fetch colors for each user
	for _, user := range users {
		user.PasswordHash = nil
		user.VerificationToken = nil
		user.PasswordResetToken = nil

		// Fetch user's colors
		if h.userColorRepo != nil {
			colorPtrs, err := h.userColorRepo.GetUserColors(tenantID, user.ID)
			if err == nil && colorPtrs != nil {
				user.Colors = make([]models.ColorCategory, len(colorPtrs))
				for i, c := range colorPtrs {
					if c != nil {
						user.Colors[i] = *c
					}
				}
			}
		}
	}

	respondJSON(w, http.StatusOK, users)
}

// GetUser gets a user by ID (admin only)
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	// SaaS: Get tenant_id from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

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

	// SaaS SECURITY: Verify user belongs to the requesting tenant
	// (tenant_id=0 is valid for Simple-Mode, so always check tenant match)
	if user.TenantID != tenantID {
		// Return 404 to prevent tenant enumeration
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Don't return sensitive data
	user.PasswordHash = nil
	user.VerificationToken = nil
	user.PasswordResetToken = nil

	// Fetch user's colors
	if h.userColorRepo != nil {
		colorPtrs, err := h.userColorRepo.GetUserColors(tenantID, userID)
		if err == nil && colorPtrs != nil {
			user.Colors = make([]models.ColorCategory, len(colorPtrs))
			for i, c := range colorPtrs {
				if c != nil {
					user.Colors[i] = *c
				}
			}
		}
	}

	respondJSON(w, http.StatusOK, user)
}

// DeactivateUser deactivates a user account (admin only)
func (h *UserHandler) DeactivateUser(w http.ResponseWriter, r *http.Request) {
	// SaaS SECURITY: Get tenant_id from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

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

	// SaaS SECURITY: Verify user belongs to the requesting tenant
	// (tenant_id=0 is valid for Simple-Mode, so always check tenant match)
	if user.TenantID != tenantID {
		// Return 404 to prevent tenant enumeration
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
	// SaaS SECURITY: Get tenant_id from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	// Get user ID from URL
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Parse optional message
	// BUG FIX #9: Check error from json.NewDecoder().Decode() and log warning if not io.EOF
	var req struct {
		Message *string `json:"message,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		log.Printf("Warning: Failed to decode optional message in ActivateUser: %v", err)
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

	// SaaS SECURITY: Verify user belongs to the requesting tenant
	// (tenant_id=0 is valid for Simple-Mode, so always check tenant match)
	if user.TenantID != tenantID {
		// Return 404 to prevent tenant enumeration
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
	// SaaS SECURITY: Get tenant_id from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

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

	// SaaS SECURITY: Verify user belongs to the requesting tenant
	// (tenant_id=0 is valid for Simple-Mode, so always check tenant match)
	if targetUser.TenantID != tenantID {
		// Return 404 to prevent tenant enumeration
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
	// SaaS SECURITY: Get tenant_id from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

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

	// SaaS SECURITY: Verify user belongs to the requesting tenant
	// (tenant_id=0 is valid for Simple-Mode, so always check tenant match)
	if targetUser.TenantID != tenantID {
		// Return 404 to prevent tenant enumeration
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

// ImpersonateUser allows super-admin to act as another user (not super-admin)
// SaaS SECURITY: Super Admin can ONLY impersonate users within their own tenant
func (h *UserHandler) ImpersonateUser(w http.ResponseWriter, r *http.Request) {
	// Extract super admin from context (middleware already verified)
	isSuperAdmin, _ := r.Context().Value(middleware.IsSuperAdminKey).(bool)
	if !isSuperAdmin {
		respondError(w, http.StatusForbidden, "Only Super Admin can impersonate users")
		return
	}

	// SaaS SECURITY: Get tenant_id from context - Super Admin is tenant-scoped
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	// Get current super-admin user ID
	currentUserID, _ := r.Context().Value(middleware.UserIDKey).(int)

	// Get target user ID from URL
	vars := mux.Vars(r)
	targetUserIDStr := vars["id"]
	targetUserID, err := strconv.Atoi(targetUserIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Cannot impersonate yourself
	if targetUserID == currentUserID {
		respondError(w, http.StatusBadRequest, "Cannot impersonate yourself")
		return
	}

	// Get target user
	targetUser, err := h.userRepo.FindByID(targetUserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if targetUser == nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// SaaS SECURITY: Verify target user belongs to the Super Admin's tenant
	// Super Admin can only impersonate users within their own tenant
	// (Central Admin uses a different endpoint for cross-tenant impersonation)
	if targetUser.TenantID != tenantID {
		// Return 404 to prevent tenant enumeration
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Cannot impersonate deleted users
	if targetUser.IsDeleted {
		respondError(w, http.StatusBadRequest, "Cannot impersonate deleted user")
		return
	}

	// Cannot impersonate inactive users
	if !targetUser.IsActive {
		respondError(w, http.StatusBadRequest, "Cannot impersonate inactive user")
		return
	}

	// Cannot impersonate super-admin
	if targetUser.IsSuperAdmin {
		respondError(w, http.StatusForbidden, "Cannot impersonate Super Admin")
		return
	}

	// Get target user's email
	targetEmail := ""
	if targetUser.Email != nil {
		targetEmail = *targetUser.Email
	}

	// Generate impersonation JWT
	// SaaS: Include target user's tenant_id and is_central_admin
	token, err := h.authService.GenerateImpersonationJWT(
		targetUserID,
		targetEmail,
		targetUser.IsAdmin,
		targetUser.IsSuperAdmin,
		targetUser.IsCentralAdmin,
		currentUserID,
		targetUser.TenantID,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	// Audit log
	clientIP := logging.GetClientIP(r)
	log.Printf("AUDIT: Super-admin %d started impersonating user %d (%s %s) from IP %s",
		currentUserID, targetUserID, targetUser.FirstName, targetUser.LastName, clientIP)

	// Don't return sensitive data
	targetUser.PasswordHash = nil
	targetUser.VerificationToken = nil
	targetUser.PasswordResetToken = nil

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user":  targetUser,
	})
}

// EndImpersonation ends the impersonation session and returns to super-admin
func (h *UserHandler) EndImpersonation(w http.ResponseWriter, r *http.Request) {
	// Check if currently impersonating
	isImpersonating, _ := r.Context().Value(middleware.IsImpersonatingKey).(bool)
	if !isImpersonating {
		respondError(w, http.StatusBadRequest, "Not currently impersonating")
		return
	}

	// Get original super-admin user ID
	originalUserID, ok := r.Context().Value(middleware.OriginalUserIDKey).(int)
	if !ok || originalUserID == 0 {
		respondError(w, http.StatusBadRequest, "Invalid impersonation session")
		return
	}

	// Get impersonated user ID for audit log
	impersonatedUserID, _ := r.Context().Value(middleware.UserIDKey).(int)

	// Get original super-admin user
	originalUser, err := h.userRepo.FindByID(originalUserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if originalUser == nil {
		respondError(w, http.StatusNotFound, "Original user not found")
		return
	}

	// Get original user's email
	originalEmail := ""
	if originalUser.Email != nil {
		originalEmail = *originalUser.Email
	}

	// Generate normal JWT for super-admin (no impersonation claims)
	// SaaS: Include original user's tenant_id and is_central_admin
	token, err := h.authService.GenerateJWT(
		originalUserID,
		originalEmail,
		originalUser.IsAdmin,
		originalUser.IsSuperAdmin,
		originalUser.IsCentralAdmin,
		originalUser.TenantID,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	// Audit log
	clientIP := logging.GetClientIP(r)
	log.Printf("AUDIT: Super-admin %d ended impersonation of user %d from IP %s",
		originalUserID, impersonatedUserID, clientIP)

	// Don't return sensitive data
	originalUser.PasswordHash = nil
	originalUser.VerificationToken = nil
	originalUser.PasswordResetToken = nil

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user":  originalUser,
	})
}

// AdminUpdateUser allows admins to update user profiles (including names)
func (h *UserHandler) AdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	// SaaS SECURITY: Get tenant_id from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

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

	// SaaS SECURITY: Verify user belongs to the requesting tenant
	// (tenant_id=0 is valid for Simple-Mode, so always check tenant match)
	if targetUser.TenantID != tenantID {
		// Return 404 to prevent tenant enumeration
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
		// Check if email is already taken by another user (within same tenant)
		existingUser, _ := h.userRepo.FindByEmail(*req.Email, targetUser.TenantID)
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

// AdminCreateUser creates a new user (admin only, Super Admin can create admins)
func (h *UserHandler) AdminCreateUser(w http.ResponseWriter, r *http.Request) {
	// Check if current user is admin
	isAdmin, _ := r.Context().Value(middleware.IsAdminKey).(bool)
	isSuperAdmin, _ := r.Context().Value(middleware.IsSuperAdminKey).(bool)

	if !isAdmin {
		respondError(w, http.StatusForbidden, "Admin access required")
		return
	}

	// Parse request body
	var req models.AdminCreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Only Super Admin can create admin users
	if req.IsAdmin && !isSuperAdmin {
		respondError(w, http.StatusForbidden, "Nur Super Admin kann Admin-Benutzer erstellen")
		return
	}

	// SaaS: Get tenant_id from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	// Check email uniqueness (within tenant)
	existing, err := h.userRepo.FindByEmail(req.Email, tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if existing != nil {
		respondError(w, http.StatusConflict, "E-Mail wird bereits verwendet")
		return
	}

	// Generate temporary password
	tempPassword, err := h.authService.GenerateTempPassword()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate password")
		return
	}

	// Hash password
	passwordHash, err := h.authService.HashPassword(tempPassword)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	// Create user
	// SaaS: Include tenant_id for multi-tenancy
	user := &models.User{
		TenantID:           tenantID,
		FirstName:          req.FirstName,
		LastName:           req.LastName,
		Email:              &req.Email,
		Phone:              req.Phone,
		PasswordHash:       &passwordHash,
		IsAdmin:            req.IsAdmin,
		IsSuperAdmin:       false, // Cannot create super admin via API
		IsVerified:         true,  // Skip email verification for admin-created users
		IsActive:           true,
		IsDeleted:          false,
		MustChangePassword: true, // Force password change on first login
		TermsAcceptedAt:    time.Now(),
		LastActivityAt:     time.Now(),
	}

	if err := h.userRepo.Create(user); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	// Assign colors to user if specified
	if len(req.ColorIDs) > 0 && h.userColorRepo != nil {
		// Get current user ID (admin who's creating) for granted_by field
		currentUserID, _ := r.Context().Value(middleware.UserIDKey).(int)
		if err := h.userColorRepo.SetUserColors(tenantID, user.ID, req.ColorIDs, currentUserID); err != nil {
			// Log error but don't fail the request - user was already created
			log.Printf("Warning: Failed to assign colors to user %d: %v\n", user.ID, err)
		}
	}

	// Send temp password email synchronously to ensure delivery
	// If email fails, return temp password to admin so they can communicate it manually
	emailSent := false
	if h.emailService != nil {
		if err := h.emailService.SendTempPasswordEmail(req.Email, req.FirstName, tempPassword); err != nil {
			log.Printf("ERROR: Failed to send temp password email to %s: %v", req.Email, err)
		} else {
			emailSent = true
		}
	}

	// Don't return sensitive data
	user.PasswordHash = nil

	// Build response based on email delivery status
	response := map[string]interface{}{
		"user": user,
	}

	if emailSent {
		response["message"] = "Benutzer erfolgreich erstellt. Temporäres Passwort wurde per E-Mail gesendet."
	} else {
		// Email failed - return temp password so admin can share it manually
		response["message"] = "Benutzer erstellt, aber E-Mail konnte nicht gesendet werden. Bitte teilen Sie das temporäre Passwort manuell mit."
		response["temp_password"] = tempPassword
		response["email_failed"] = true
	}

	respondJSON(w, http.StatusCreated, response)
}

// AdminDeleteUser deletes a user account (super-admin only, GDPR anonymization)
func (h *UserHandler) AdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	// SaaS SECURITY: Get tenant_id from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	// Check if current user is super admin
	isSuperAdmin, _ := r.Context().Value(middleware.IsSuperAdminKey).(bool)
	if !isSuperAdmin {
		respondError(w, http.StatusForbidden, "Nur Super-Admins können Benutzer löschen")
		return
	}

	// Get current user ID
	currentUserID, _ := r.Context().Value(middleware.UserIDKey).(int)

	// Get target user ID from URL
	vars := mux.Vars(r)
	userIDStr := vars["id"]
	targetUserID, err := strconv.Atoi(userIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Cannot delete yourself
	if targetUserID == currentUserID {
		respondError(w, http.StatusBadRequest, "Sie können Ihr eigenes Konto nicht löschen")
		return
	}

	// Get target user
	targetUser, err := h.userRepo.FindByID(targetUserID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Benutzer nicht gefunden")
		return
	}
	if targetUser == nil {
		respondError(w, http.StatusNotFound, "Benutzer nicht gefunden")
		return
	}

	// SaaS SECURITY: Verify user belongs to the requesting tenant
	// (tenant_id=0 is valid for Simple-Mode, so always check tenant match)
	if targetUser.TenantID != tenantID {
		// Return 404 to prevent tenant enumeration
		respondError(w, http.StatusNotFound, "Benutzer nicht gefunden")
		return
	}

	// Check if already deleted
	if targetUser.IsDeleted {
		respondError(w, http.StatusBadRequest, "Benutzer wurde bereits gelöscht")
		return
	}

	// Cannot delete super admin users
	if targetUser.IsSuperAdmin {
		respondError(w, http.StatusForbidden, "Super-Admin kann nicht gelöscht werden")
		return
	}

	// Store email for confirmation before deletion
	var emailForConfirmation string
	var userName string
	if targetUser.Email != nil {
		emailForConfirmation = *targetUser.Email
	}
	userName = targetUser.FirstName

	// Delete account (GDPR anonymization)
	if err := h.userRepo.DeleteAccount(targetUserID); err != nil {
		// BUG FIX: Log detailed error internally, return generic message to user
		// This prevents leaking database/system error details to potential attackers
		log.Printf("ERROR: Failed to delete user %d: %v", targetUserID, err)
		respondError(w, http.StatusInternalServerError, "Fehler beim Löschen des Benutzers")
		return
	}

	// Send confirmation email to the deleted user
	if emailForConfirmation != "" && h.emailService != nil {
		go h.emailService.SendAccountDeletionConfirmation(emailForConfirmation, userName)
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Benutzer erfolgreich gelöscht"})
}

// ExportMyData exports all personal data for the authenticated user (GDPR compliance)
// GET /api/users/me/export
func (h *UserHandler) ExportMyData(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	tenantID := middleware.GetTenantID(r)

	// Get user data
	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Benutzerdaten")
		return
	}
	if user == nil {
		respondError(w, http.StatusNotFound, "Benutzer nicht gefunden")
		return
	}

	// Sanitize sensitive fields
	user.PasswordHash = nil
	user.VerificationToken = nil
	user.PasswordResetToken = nil
	user.VerificationTokenExpires = nil

	export := map[string]interface{}{
		"user":        user,
		"exported_at": time.Now().Format(time.RFC3339),
		"tenant_id":   tenantID,
	}

	// Get user's bookings
	var bookings []map[string]interface{}
	rows, err := h.db.Query(`
		SELECT b.id, b.date, b.walk_type, b.status, b.notes, b.created_at,
		       d.name as dog_name, d.breed as dog_breed
		FROM bookings b
		LEFT JOIN dogs d ON b.dog_id = d.id
		WHERE b.user_id = ? AND b.tenant_id = ?
		ORDER BY b.date DESC`, userID, tenantID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var booking struct {
				ID        int
				Date      string
				WalkType  string
				Status    string
				Notes     *string
				CreatedAt time.Time
				DogName   *string
				DogBreed  *string
			}
			if err := rows.Scan(&booking.ID, &booking.Date, &booking.WalkType, &booking.Status,
				&booking.Notes, &booking.CreatedAt, &booking.DogName, &booking.DogBreed); err == nil {
				bookings = append(bookings, map[string]interface{}{
					"id":         booking.ID,
					"date":       booking.Date,
					"walk_type":  booking.WalkType,
					"status":     booking.Status,
					"notes":      booking.Notes,
					"created_at": booking.CreatedAt,
					"dog_name":   booking.DogName,
					"dog_breed":  booking.DogBreed,
				})
			}
		}
		// BUG FIX: Check for errors during iteration - MUST fail export for GDPR compliance
		if err := rows.Err(); err != nil {
			log.Printf("ERROR: Failed to iterate bookings for user %d: %v", userID, err)
			respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Buchungsdaten")
			return
		}
	}
	export["bookings"] = bookings
	export["booking_count"] = len(bookings)

	// Get user's walk reports
	var walkReports []map[string]interface{}
	rows, err = h.db.Query(`
		SELECT wr.id, wr.booking_id, wr.weather, wr.mood_before, wr.mood_after,
		       wr.walked_distance_meters, wr.duration_minutes, wr.notes, wr.created_at
		FROM walk_reports wr
		JOIN bookings b ON wr.booking_id = b.id
		WHERE b.user_id = ? AND b.tenant_id = ?
		ORDER BY wr.created_at DESC`, userID, tenantID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var report struct {
				ID              int
				BookingID       int
				Weather         *string
				MoodBefore      *string
				MoodAfter       *string
				DistanceMeters  *int
				DurationMinutes *int
				Notes           *string
				CreatedAt       time.Time
			}
			if err := rows.Scan(&report.ID, &report.BookingID, &report.Weather, &report.MoodBefore,
				&report.MoodAfter, &report.DistanceMeters, &report.DurationMinutes,
				&report.Notes, &report.CreatedAt); err == nil {
				walkReports = append(walkReports, map[string]interface{}{
					"id":                     report.ID,
					"booking_id":             report.BookingID,
					"weather":                report.Weather,
					"mood_before":            report.MoodBefore,
					"mood_after":             report.MoodAfter,
					"walked_distance_meters": report.DistanceMeters,
					"duration_minutes":       report.DurationMinutes,
					"notes":                  report.Notes,
					"created_at":             report.CreatedAt,
				})
			}
		}
		// BUG FIX: Check for errors during iteration - MUST fail export for GDPR compliance
		if err := rows.Err(); err != nil {
			log.Printf("ERROR: Failed to iterate walk reports for user %d: %v", userID, err)
			respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Laufberichte")
			return
		}
	}
	export["walk_reports"] = walkReports

	// Get user's experience requests
	var experienceRequests []map[string]interface{}
	rows, err = h.db.Query(`
		SELECT id, requested_level, reason, status, admin_notes, created_at, updated_at
		FROM experience_requests
		WHERE user_id = ? AND tenant_id = ?
		ORDER BY created_at DESC`, userID, tenantID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var req struct {
				ID             int
				RequestedLevel string
				Reason         string
				Status         string
				AdminNotes     *string
				CreatedAt      time.Time
				UpdatedAt      *time.Time
			}
			if err := rows.Scan(&req.ID, &req.RequestedLevel, &req.Reason, &req.Status,
				&req.AdminNotes, &req.CreatedAt, &req.UpdatedAt); err == nil {
				experienceRequests = append(experienceRequests, map[string]interface{}{
					"id":              req.ID,
					"requested_level": req.RequestedLevel,
					"reason":          req.Reason,
					"status":          req.Status,
					"admin_notes":     req.AdminNotes,
					"created_at":      req.CreatedAt,
					"updated_at":      req.UpdatedAt,
				})
			}
		}
		// BUG FIX: Check for errors during iteration - MUST fail export for GDPR compliance
		if err := rows.Err(); err != nil {
			log.Printf("ERROR: Failed to iterate experience requests for user %d: %v", userID, err)
			respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Erfahrungsanfragen")
			return
		}
	}
	export["experience_requests"] = experienceRequests

	// Get user's color requests
	var colorRequests []map[string]interface{}
	rows, err = h.db.Query(`
		SELECT cr.id, cr.status, cr.reason, cr.admin_notes, cr.created_at, cr.updated_at,
		       cc.name as color_name, cc.hex_code
		FROM color_requests cr
		LEFT JOIN color_categories cc ON cr.color_id = cc.id
		WHERE cr.user_id = ? AND cr.tenant_id = ?
		ORDER BY cr.created_at DESC`, userID, tenantID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var req struct {
				ID         int
				Status     string
				Reason     *string
				AdminNotes *string
				CreatedAt  time.Time
				UpdatedAt  *time.Time
				ColorName  *string
				HexCode    *string
			}
			if err := rows.Scan(&req.ID, &req.Status, &req.Reason, &req.AdminNotes,
				&req.CreatedAt, &req.UpdatedAt, &req.ColorName, &req.HexCode); err == nil {
				colorRequests = append(colorRequests, map[string]interface{}{
					"id":          req.ID,
					"status":      req.Status,
					"reason":      req.Reason,
					"admin_notes": req.AdminNotes,
					"created_at":  req.CreatedAt,
					"updated_at":  req.UpdatedAt,
					"color_name":  req.ColorName,
					"hex_code":    req.HexCode,
				})
			}
		}
		// BUG FIX: Check for errors during iteration - MUST fail export for GDPR compliance
		if err := rows.Err(); err != nil {
			log.Printf("ERROR: Failed to iterate color requests for user %d: %v", userID, err)
			respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Farbanfragen")
			return
		}
	}
	export["color_requests"] = colorRequests

	// Fetch user's assigned colors
	colorPtrs, _ := h.userColorRepo.GetUserColors(tenantID, userID)
	var colors []map[string]interface{}
	for _, c := range colorPtrs {
		if c != nil {
			colors = append(colors, map[string]interface{}{
				"id":       c.ID,
				"name":     c.Name,
				"hex_code": c.HexCode,
			})
		}
	}
	export["assigned_colors"] = colors

	// Set headers for file download
	w.Header().Set("Content-Disposition", "attachment; filename=meine-daten.json")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	// Log the export for audit
	log.Printf("AUDIT: User %d exported their personal data from tenant %d", userID, tenantID)

	json.NewEncoder(w).Encode(export)
}
