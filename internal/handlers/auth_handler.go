package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
	"github.com/tranmh/gassigeher/internal/services"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	userRepo          *repository.UserRepository
	userColorRepo     *repository.UserColorRepository
	settingsRepo      *repository.SettingsRepository
	authService       *services.AuthService
	emailService      *services.EmailService
	bruteForceService *services.BruteForceService
	config            *config.Config
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(db *sql.DB, cfg *config.Config) *AuthHandler {
	emailService, err := services.NewEmailService(services.ConfigToEmailConfig(cfg))
	if err != nil {
		// Log error but don't fail - emails will fail gracefully
		fmt.Printf("Warning: Failed to initialize email service: %v\n", err)
	}

	return &AuthHandler{
		userRepo:          repository.NewUserRepository(db),
		userColorRepo:     repository.NewUserColorRepository(db),
		settingsRepo:      repository.NewSettingsRepository(db),
		authService:       services.NewAuthService(cfg.JWTSecret, cfg.JWTExpirationHours),
		emailService:      emailService,
		bruteForceService: services.NewBruteForceService(),
		config:            cfg,
	}
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate input (includes phone number validation)
	if err := req.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// SaaS: Get tenant_id from context (set by TenantMiddleware)
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	// Validate registration password against stored value
	storedPassword, err := h.settingsRepo.Get(tenantID, "registration_password")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if storedPassword == nil || !strings.EqualFold(storedPassword.Value, req.RegistrationPassword) {
		respondError(w, http.StatusBadRequest, "Ungültiges Registrierungspasswort")
		return
	}

	// Validate password strength
	if err := h.authService.ValidatePassword(req.Password); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Check if user already exists (within tenant)
	existing, err := h.userRepo.FindByEmail(req.Email, tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if existing != nil {
		respondError(w, http.StatusConflict, "Email already registered")
		return
	}

	// Hash password
	passwordHash, err := h.authService.HashPassword(req.Password)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	// Generate verification token
	verificationToken, err := h.authService.GenerateToken()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate verification token")
		return
	}

	expires := time.Now().Add(24 * time.Hour)

	// Create user
	// SaaS: Include tenant_id for multi-tenancy
	user := &models.User{
		TenantID:                 tenantID,
		FirstName:                req.FirstName,
		LastName:                 req.LastName,
		Email:                    &req.Email,
		Phone:                    &req.Phone,
		PasswordHash:             &passwordHash,
		IsVerified:               false,
		IsActive:                 true,
		IsDeleted:                false,
		VerificationToken:        &verificationToken,
		VerificationTokenExpires: &expires,
		TermsAcceptedAt:          time.Now(),
		LastActivityAt:           time.Now(),
	}

	if err := h.userRepo.Create(user); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	// Assign default color (green = ID 1) to new user
	// Green users start with only the green color
	if h.userColorRepo != nil {
		if err := h.userColorRepo.SetUserColors(tenantID, user.ID, []int{1}, user.ID); err != nil {
			// Log but don't fail registration
			fmt.Printf("Warning: Failed to assign default color to user %d: %v\n", user.ID, err)
		}
	}

	// Send verification email
	if h.emailService != nil {
		if err := h.emailService.SendVerificationEmail(req.Email, req.FirstName, verificationToken); err != nil {
			fmt.Printf("Failed to send verification email: %v\n", err)
			// Don't fail the registration if email fails
		}

		// Send admin notification email (to Super Admin)
		if h.config.SuperAdminEmail != "" {
			fullName := req.FirstName + " " + req.LastName
			go func() {
				if err := h.emailService.SendNewUserRegistrationNotification(
					h.config.SuperAdminEmail,
					fullName,
					req.Email,
					req.Phone,
				); err != nil {
					fmt.Printf("Failed to send admin notification email: %v\n", err)
				}
			}()
		}
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "Registration successful. Please check your email to verify your account.",
		"user_id": user.ID,
	})
}

// VerifyEmail handles email verification
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req models.VerifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if strings.TrimSpace(req.Token) == "" {
		respondError(w, http.StatusBadRequest, "Token is required")
		return
	}

	// Find user by token
	user, err := h.userRepo.FindByVerificationToken(req.Token)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if user == nil {
		respondError(w, http.StatusNotFound, "Invalid or expired verification token")
		return
	}

	// Check if already verified
	if user.IsVerified {
		respondError(w, http.StatusBadRequest, "Email already verified")
		return
	}

	// Check if token expired
	if user.VerificationTokenExpires != nil && time.Now().After(*user.VerificationTokenExpires) {
		respondError(w, http.StatusBadRequest, "Verification token expired")
		return
	}

	// Mark as verified
	user.IsVerified = true
	user.VerificationToken = nil
	user.VerificationTokenExpires = nil

	if err := h.userRepo.Update(user); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to verify user")
		return
	}

	// Send welcome email
	if h.emailService != nil && user.Email != nil {
		if err := h.emailService.SendWelcomeEmail(*user.Email, user.FirstName); err != nil {
			fmt.Printf("Failed to send welcome email: %v\n", err)
		}
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Email verified successfully. You can now login.",
	})
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Password) == "" {
		respondError(w, http.StatusBadRequest, "Email and password are required")
		return
	}

	// SaaS: Get tenant_id from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	// Brute force protection: Create key from email and IP
	clientIP := getClientIPForAuth(r)
	bruteForceKey := req.Email + ":" + clientIP

	// Check if account is locked out
	if locked, remaining := h.bruteForceService.IsLocked(bruteForceKey); locked {
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(remaining.Seconds())))
		respondError(w, http.StatusTooManyRequests,
			fmt.Sprintf("Zu viele fehlgeschlagene Anmeldeversuche. Bitte warten Sie %d Sekunden.", int(remaining.Seconds())))
		return
	}

	// Find user (within tenant)
	user, err := h.userRepo.FindByEmail(req.Email, tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if user == nil || user.PasswordHash == nil {
		// Record failed attempt (user not found)
		h.bruteForceService.RecordFailure(bruteForceKey)
		respondError(w, http.StatusUnauthorized, "Ungültige Anmeldedaten")
		return
	}

	// Check password
	if !h.authService.CheckPassword(req.Password, *user.PasswordHash) {
		// Record failed attempt (wrong password)
		lockoutDuration := h.bruteForceService.RecordFailure(bruteForceKey)
		if lockoutDuration > 0 {
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(lockoutDuration.Seconds())))
			respondError(w, http.StatusTooManyRequests,
				fmt.Sprintf("Zu viele fehlgeschlagene Anmeldeversuche. Konto für %d Sekunden gesperrt.", int(lockoutDuration.Seconds())))
			return
		}
		respondError(w, http.StatusUnauthorized, "Ungültige Anmeldedaten")
		return
	}

	// SECURITY FIX: Return uniform error messages to prevent account enumeration
	// Don't reveal if account is unverified or deactivated

	// Check if verified
	if !user.IsVerified {
		// Send verification reminder email in background (don't block response)
		if user.Email != nil && user.VerificationToken != nil && h.emailService != nil {
			go h.emailService.SendVerificationEmail(*user.Email, user.FirstName, *user.VerificationToken)
		}
		respondError(w, http.StatusUnauthorized, "Ungültige Anmeldedaten")
		return
	}

	// Check if active
	if !user.IsActive {
		// Could send reactivation instructions via email (don't reveal in response)
		respondError(w, http.StatusUnauthorized, "Ungültige Anmeldedaten")
		return
	}

	// Clear brute force failures on successful login
	h.bruteForceService.ClearFailures(bruteForceKey)

	// Update last activity
	if err := h.userRepo.UpdateLastActivity(user.ID); err != nil {
		fmt.Printf("Failed to update last activity: %v\n", err)
	}

	// DONE: Get admin status from database (not config)
	isAdmin := user.IsAdmin
	isSuperAdmin := user.IsSuperAdmin // DONE: Phase 3

	// Generate JWT
	// DONE: Phase 3 - Include isSuperAdmin in JWT
	// SaaS: Include tenant_id in JWT
	token, err := h.authService.GenerateJWT(user.ID, req.Email, isAdmin, isSuperAdmin, user.TenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	respondJSON(w, http.StatusOK, models.LoginResponse{
		Token:              token,
		User:               user,
		IsAdmin:            isAdmin,
		MustChangePassword: user.MustChangePassword,
	})
}

// ForgotPassword handles password reset request
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req models.ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if strings.TrimSpace(req.Email) == "" {
		respondError(w, http.StatusBadRequest, "Email is required")
		return
	}

	// SaaS: Get tenant_id from context
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	// Find user (within tenant)
	user, err := h.userRepo.FindByEmail(req.Email, tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Always return success even if user doesn't exist (security)
	if user == nil {
		respondJSON(w, http.StatusOK, map[string]string{
			"message": "If an account exists with this email, you will receive a password reset link.",
		})
		return
	}

	// Generate reset token
	resetToken, err := h.authService.GenerateToken()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate reset token")
		return
	}

	expires := time.Now().Add(1 * time.Hour)
	user.PasswordResetToken = &resetToken
	user.PasswordResetExpires = &expires

	if err := h.userRepo.Update(user); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to save reset token")
		return
	}

	// Send reset email
	if h.emailService != nil && user.Email != nil {
		if err := h.emailService.SendPasswordResetEmail(*user.Email, user.FirstName, resetToken); err != nil {
			fmt.Printf("Failed to send password reset email: %v\n", err)
		}
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "If an account exists with this email, you will receive a password reset link.",
	})
}

// ResetPassword handles password reset with token
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req models.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if strings.TrimSpace(req.Token) == "" {
		respondError(w, http.StatusBadRequest, "Token is required")
		return
	}

	if req.Password != req.ConfirmPassword {
		respondError(w, http.StatusBadRequest, "Passwords do not match")
		return
	}

	// Validate password
	if err := h.authService.ValidatePassword(req.Password); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Find user by token
	user, err := h.userRepo.FindByPasswordResetToken(req.Token)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if user == nil {
		respondError(w, http.StatusNotFound, "Invalid or expired reset token")
		return
	}

	// Check if token expired
	if user.PasswordResetExpires != nil && time.Now().After(*user.PasswordResetExpires) {
		respondError(w, http.StatusBadRequest, "Reset token expired")
		return
	}

	// Hash new password
	passwordHash, err := h.authService.HashPassword(req.Password)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	// Update password and clear token
	user.PasswordHash = &passwordHash
	user.PasswordResetToken = nil
	user.PasswordResetExpires = nil

	if err := h.userRepo.Update(user); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update password")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Password reset successful. You can now login with your new password.",
	})
}

// ChangePassword handles password change for logged-in users
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.NewPassword != req.ConfirmPassword {
		respondError(w, http.StatusBadRequest, "Passwords do not match")
		return
	}

	// Validate new password
	if err := h.authService.ValidatePassword(req.NewPassword); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Get user
	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if user == nil || user.PasswordHash == nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Verify old password
	if !h.authService.CheckPassword(req.OldPassword, *user.PasswordHash) {
		respondError(w, http.StatusUnauthorized, "Incorrect old password")
		return
	}

	// Hash new password
	newHash, err := h.authService.HashPassword(req.NewPassword)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	user.PasswordHash = &newHash
	if err := h.userRepo.Update(user); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update password")
		return
	}

	// Clear must_change_password flag if set
	if user.MustChangePassword {
		if err := h.userRepo.ClearMustChangePassword(user.ID); err != nil {
			fmt.Printf("Warning: Failed to clear must_change_password flag: %v\n", err)
		}
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Password changed successfully",
	})
}

// Helper functions

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// getClientIPForAuth extracts the real client IP address for brute force tracking
func getClientIPForAuth(r *http.Request) string {
	// Check X-Forwarded-For header (from reverse proxy)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take first IP if comma-separated
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return strings.TrimSpace(xff[:i])
			}
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr (strip port)
	addr := r.RemoteAddr
	if colonIdx := strings.LastIndex(addr, ":"); colonIdx != -1 {
		// Check if this is IPv6 address (has brackets)
		if bracketIdx := strings.LastIndex(addr, "]"); bracketIdx != -1 && bracketIdx < colonIdx {
			return addr[:colonIdx]
		} else if !strings.Contains(addr, "[") {
			// IPv4 address
			return addr[:colonIdx]
		}
	}
	return addr
}
