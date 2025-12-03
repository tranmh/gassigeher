package main

import (
	"encoding/json"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/cron"
	"github.com/tranmh/gassigeher/internal/database"
	"github.com/tranmh/gassigeher/internal/handlers"
	"github.com/tranmh/gassigeher/internal/logging"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/repository"
	"github.com/tranmh/gassigeher/internal/services"
	"github.com/tranmh/gassigeher/internal/static"
	"github.com/tranmh/gassigeher/internal/version"
)

func main() {
	// Parse command-line flags
	envPath := flag.String("env", "./.env", "Path to the .env file")
	flag.Parse()

	// Check if the .env file exists
	if _, err := os.Stat(*envPath); os.IsNotExist(err) {
		log.Printf("No .env found, using env vars")
	} else {
		if err := godotenv.Load(*envPath); err != nil {
			log.Fatalf("Error loading .env: %v", err)
		}
		log.Printf("Loaded from: %s", *envPath)
	}

	// Load environment variables from specified path
	if err := godotenv.Load(*envPath); err != nil {
		log.Fatalf("Error loading .env file from %s: %v", *envPath, err)
	}

	// Initialize logger with rotation support
	// Configuration from environment variables with defaults
	logConfig := &logging.Config{
		LogDir:         getEnvOrDefault("LOG_DIR", "./logs"),
		MaxAgeDays:     getEnvIntOrDefault("LOG_MAX_AGE_DAYS", 30),
		CompressSizeMB: getEnvIntOrDefault("LOG_COMPRESS_SIZE_MB", 10),
		ConsoleOutput:  getEnvBoolOrDefault("LOG_CONSOLE_OUTPUT", true),
	}

	logger, err := logging.NewLogger(logConfig)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Close()

	log.Printf("Loaded environment variables from: %s", *envPath)
	log.Printf("Log files will be written to: %s (retention: %d days, compress > %dMB)",
		logConfig.LogDir, logConfig.MaxAgeDays, logConfig.CompressSizeMB)

	// Load configuration
	cfg := config.Load()

	// Initialize database with multi-database support
	dbConfig := cfg.GetDBConfig()
	db, dialect, err := database.InitializeWithConfig(dbConfig)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Log database type for transparency
	log.Printf("Using database: %s", dialect.Name())

	// Run migrations with dialect support
	if err := database.RunMigrationsWithDialect(db, dialect); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// DONE: Phase 2 - Run seed data (first-time installations)
	if err := database.SeedDatabase(db, cfg.SuperAdminEmail); err != nil {
		log.Fatalf("Failed to seed database: %v", err)
	}

	// DONE: Phase 2 - Check and update Super Admin password
	superAdminService := services.NewSuperAdminService(db, cfg)
	if err := superAdminService.CheckAndUpdatePassword(); err != nil {
		log.Printf("Warning: Failed to check Super Admin password: %v", err)
		// Don't exit - allow server to start
	}

	// Initialize router
	router := mux.NewRouter()

	// Apply global middleware
	router.Use(middleware.LoggingMiddleware)
	router.Use(middleware.SecurityHeadersMiddleware)
	router.Use(middleware.CORSMiddleware(cfg.BaseURL))

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db, cfg)
	userHandler := handlers.NewUserHandler(db, cfg)
	dogHandler := handlers.NewDogHandler(db, cfg)
	bookingHandler := handlers.NewBookingHandler(db, cfg)
	blockedDateHandler := handlers.NewBlockedDateHandler(db, cfg)
	settingsHandler := handlers.NewSettingsHandler(db, cfg)
	experienceHandler := handlers.NewExperienceRequestHandler(db, cfg)
	reactivationHandler := handlers.NewReactivationRequestHandler(db, cfg)
	dashboardHandler := handlers.NewDashboardHandler(db, cfg)
	healthHandler := handlers.NewHealthHandler()
	router.HandleFunc("/api/health", healthHandler.Health).Methods("GET")

	// Initialize booking time repositories and services
	bookingTimeRepo := repository.NewBookingTimeRepository(db)
	holidayRepo := repository.NewHolidayRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	holidayService := services.NewHolidayService(holidayRepo, settingsRepo)
	bookingTimeService := services.NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo)

	// Initialize booking time handlers
	bookingTimeHandler := handlers.NewBookingTimeHandler(bookingTimeRepo, bookingTimeService)
	holidayHandler := handlers.NewHolidayHandler(holidayRepo, holidayService)

	// Start cron service for auto-completion and reminders
	cronService := cron.NewCronService(db, cfg)
	cronService.Start()
	defer cronService.Stop()

	// Version endpoint (public)
	router.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(version.Get())
	}).Methods("GET")

	// Public routes
	router.HandleFunc("/api/auth/register", authHandler.Register).Methods("POST")
	router.HandleFunc("/api/auth/verify-email", authHandler.VerifyEmail).Methods("POST")
	// BUG FIX #6: Add rate limiting to login endpoint
	loginRoute := router.PathPrefix("/api/auth/login").Subrouter()
	loginRoute.Use(middleware.RateLimitLogin)
	loginRoute.HandleFunc("", authHandler.Login).Methods("POST")
	// DONE: BUG #6 - Rate limiting applied to login
	router.HandleFunc("/api/auth/forgot-password", authHandler.ForgotPassword).Methods("POST")
	router.HandleFunc("/api/auth/reset-password", authHandler.ResetPassword).Methods("POST")

	// Reactivation request (public - for deactivated users)
	router.HandleFunc("/api/reactivation-requests", reactivationHandler.CreateRequest).Methods("POST")

	// Booking time routes (public - for time slot availability)
	router.HandleFunc("/api/booking-times/available", bookingTimeHandler.GetAvailableSlots).Methods("GET")
	router.HandleFunc("/api/booking-times/rules-for-date", bookingTimeHandler.GetRulesForDate).Methods("GET")
	router.HandleFunc("/api/holidays", holidayHandler.GetHolidays).Methods("GET")

	// Featured dogs (public - for homepage)
	router.HandleFunc("/api/dogs/featured", dogHandler.GetFeaturedDogs).Methods("GET")

	// Protected routes (authenticated users)
	protected := router.PathPrefix("/api").Subrouter()
	protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))

	// Auth
	protected.HandleFunc("/auth/change-password", authHandler.ChangePassword).Methods("PUT")

	// Users
	protected.HandleFunc("/users/me", userHandler.GetMe).Methods("GET")
	protected.HandleFunc("/users/me", userHandler.UpdateMe).Methods("PUT")
	protected.HandleFunc("/users/me/photo", userHandler.UploadPhoto).Methods("POST")
	protected.HandleFunc("/users/me", userHandler.DeleteAccount).Methods("DELETE")

	// Dogs (read-only for authenticated users)
	protected.HandleFunc("/dogs", dogHandler.ListDogs).Methods("GET")
	protected.HandleFunc("/dogs/breeds", dogHandler.GetBreeds).Methods("GET")
	protected.HandleFunc("/dogs/{id}", dogHandler.GetDog).Methods("GET")

	// Bookings (authenticated users)
	protected.HandleFunc("/bookings", bookingHandler.ListBookings).Methods("GET")
	protected.HandleFunc("/bookings", bookingHandler.CreateBooking).Methods("POST")
	protected.HandleFunc("/bookings/{id}", bookingHandler.GetBooking).Methods("GET")
	protected.HandleFunc("/bookings/{id}/cancel", bookingHandler.CancelBooking).Methods("PUT")
	protected.HandleFunc("/bookings/{id}/notes", bookingHandler.AddNotes).Methods("PUT")
	protected.HandleFunc("/bookings/calendar/{year}/{month}", bookingHandler.GetCalendarData).Methods("GET")

	// Blocked dates (read-only for authenticated users)
	protected.HandleFunc("/blocked-dates", blockedDateHandler.ListBlockedDates).Methods("GET")

	// Experience requests (authenticated users)
	protected.HandleFunc("/experience-requests", experienceHandler.CreateRequest).Methods("POST")
	protected.HandleFunc("/experience-requests", experienceHandler.ListRequests).Methods("GET")

	// Admin-only routes
	admin := protected.PathPrefix("").Subrouter()
	admin.Use(middleware.RequireAdmin)

	// Dog management (admin only)
	admin.HandleFunc("/dogs", dogHandler.CreateDog).Methods("POST")
	admin.HandleFunc("/dogs/{id}", dogHandler.UpdateDog).Methods("PUT")
	admin.HandleFunc("/dogs/{id}", dogHandler.DeleteDog).Methods("DELETE")
	admin.HandleFunc("/dogs/{id}/photo", dogHandler.UploadDogPhoto).Methods("POST")
	admin.HandleFunc("/dogs/{id}/availability", dogHandler.ToggleAvailability).Methods("PUT")
	admin.HandleFunc("/dogs/{id}/featured", dogHandler.SetFeatured).Methods("PUT")

	// Blocked dates management (admin only)
	admin.HandleFunc("/blocked-dates", blockedDateHandler.CreateBlockedDate).Methods("POST")
	admin.HandleFunc("/blocked-dates/{id}", blockedDateHandler.DeleteBlockedDate).Methods("DELETE")

	// Booking management (admin only)
	admin.HandleFunc("/bookings/{id}/move", bookingHandler.MoveBooking).Methods("PUT")

	// System settings (admin only)
	admin.HandleFunc("/settings", settingsHandler.GetAllSettings).Methods("GET")
	admin.HandleFunc("/settings/{key}", settingsHandler.UpdateSetting).Methods("PUT")

	// Experience requests management (admin only)
	admin.HandleFunc("/experience-requests/{id}/approve", experienceHandler.ApproveRequest).Methods("PUT")
	admin.HandleFunc("/experience-requests/{id}/deny", experienceHandler.DenyRequest).Methods("PUT")

	// User management (admin only)
	admin.HandleFunc("/users", userHandler.ListUsers).Methods("GET")
	admin.HandleFunc("/users/{id}", userHandler.GetUser).Methods("GET")
	admin.HandleFunc("/users/{id}/activate", userHandler.ActivateUser).Methods("PUT")
	admin.HandleFunc("/users/{id}/deactivate", userHandler.DeactivateUser).Methods("PUT")

	// Reactivation requests management (admin only)
	admin.HandleFunc("/reactivation-requests", reactivationHandler.ListRequests).Methods("GET")
	admin.HandleFunc("/reactivation-requests/{id}/approve", reactivationHandler.ApproveRequest).Methods("PUT")
	admin.HandleFunc("/reactivation-requests/{id}/deny", reactivationHandler.DenyRequest).Methods("PUT")

	// Admin dashboard (admin only)
	admin.HandleFunc("/admin/stats", dashboardHandler.GetStats).Methods("GET")
	admin.HandleFunc("/admin/activity", dashboardHandler.GetRecentActivity).Methods("GET")

	// Booking time management (admin only)
	admin.HandleFunc("/admin/booking-times/rules", bookingTimeHandler.GetRules).Methods("GET")
	admin.HandleFunc("/admin/booking-times/rules", bookingTimeHandler.UpdateRules).Methods("PUT")
	admin.HandleFunc("/admin/booking-times/rules", bookingTimeHandler.CreateRule).Methods("POST")
	admin.HandleFunc("/admin/booking-times/rules/{id}", bookingTimeHandler.DeleteRule).Methods("DELETE")

	// Holiday management (admin only)
	admin.HandleFunc("/admin/holidays", holidayHandler.CreateHoliday).Methods("POST")
	admin.HandleFunc("/admin/holidays/{id}", holidayHandler.UpdateHoliday).Methods("PUT")
	admin.HandleFunc("/admin/holidays/{id}", holidayHandler.DeleteHoliday).Methods("DELETE")

	// Booking approval management (admin only)
	admin.HandleFunc("/bookings/pending-approvals", bookingHandler.GetPendingApprovals).Methods("GET")
	admin.HandleFunc("/bookings/{id}/approve", bookingHandler.ApprovePendingBooking).Methods("PUT")
	admin.HandleFunc("/bookings/{id}/reject", bookingHandler.RejectPendingBooking).Methods("PUT")

	// DONE: Phase 4 - Super Admin routes (authenticated + admin + super admin)
	superAdmin := admin.PathPrefix("").Subrouter()
	superAdmin.Use(middleware.RequireSuperAdmin)
	superAdmin.HandleFunc("/admin/users/{id}/promote", userHandler.PromoteToAdmin).Methods("POST")
	superAdmin.HandleFunc("/admin/users/{id}/demote", userHandler.DemoteAdmin).Methods("POST")

	// Uploads directory (user photos, dog photos) - must remain on filesystem
	router.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))

	// Get embedded frontend filesystem
	frontendFS, err := static.FrontendFS()
	if err != nil {
		log.Fatalf("Failed to get embedded frontend: %v", err)
	}

	// Serve specific HTML pages without .html extension
	router.HandleFunc("/verify", func(w http.ResponseWriter, r *http.Request) {
		serveEmbeddedFile(w, r, frontendFS, "verify.html")
	}).Methods("GET")
	router.HandleFunc("/reset-password", func(w http.ResponseWriter, r *http.Request) {
		serveEmbeddedFile(w, r, frontendFS, "reset-password.html")
	}).Methods("GET")
	router.HandleFunc("/forgot-password", func(w http.ResponseWriter, r *http.Request) {
		serveEmbeddedFile(w, r, frontendFS, "forgot-password.html")
	}).Methods("GET")

	// Static files from embedded frontend
	router.PathPrefix("/").Handler(http.FileServer(http.FS(frontendFS)))

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s...", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// Helper functions for environment variable parsing

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// serveEmbeddedFile serves a file from the embedded filesystem
func serveEmbeddedFile(w http.ResponseWriter, r *http.Request, fsys fs.FS, filename string) {
	content, err := fs.ReadFile(fsys, filename)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Set content type based on extension
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(content)
}
