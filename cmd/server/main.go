package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/tranm/gassigeher/internal/config"
	"github.com/tranm/gassigeher/internal/cron"
	"github.com/tranm/gassigeher/internal/database"
	"github.com/tranm/gassigeher/internal/handlers"
	"github.com/tranm/gassigeher/internal/middleware"
	"github.com/tranm/gassigeher/internal/repository"
	"github.com/tranm/gassigeher/internal/services"
)

func main() {
	// Parse command-line flags
	envPath := flag.String("env", "./.env", "Path to the .env file")
	flag.Parse()

	// Check if the .env file exists
	if _, err := os.Stat(*envPath); os.IsNotExist(err) {
		log.Fatalf("Error: .env file not found at path: %s", *envPath)
	}

	// Load environment variables from specified path
	if err := godotenv.Load(*envPath); err != nil {
		log.Fatalf("Error loading .env file from %s: %v", *envPath, err)
	}
	log.Printf("Loaded environment variables from: %s", *envPath)

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
	cronService := cron.NewCronService(db)
	cronService.Start()
	defer cronService.Stop()

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
	admin.HandleFunc("/booking-times/rules", bookingTimeHandler.GetRules).Methods("GET")
	admin.HandleFunc("/booking-times/rules", bookingTimeHandler.UpdateRules).Methods("PUT")
	admin.HandleFunc("/booking-times/rules", bookingTimeHandler.CreateRule).Methods("POST")
	admin.HandleFunc("/booking-times/rules/{id}", bookingTimeHandler.DeleteRule).Methods("DELETE")

	// Holiday management (admin only)
	admin.HandleFunc("/holidays", holidayHandler.CreateHoliday).Methods("POST")
	admin.HandleFunc("/holidays/{id}", holidayHandler.UpdateHoliday).Methods("PUT")
	admin.HandleFunc("/holidays/{id}", holidayHandler.DeleteHoliday).Methods("DELETE")

	// Booking approval management (admin only)
	admin.HandleFunc("/bookings/pending-approvals", bookingHandler.GetPendingApprovals).Methods("GET")
	admin.HandleFunc("/bookings/{id}/approve", bookingHandler.ApprovePendingBooking).Methods("PUT")
	admin.HandleFunc("/bookings/{id}/reject", bookingHandler.RejectPendingBooking).Methods("PUT")

	// DONE: Phase 4 - Super Admin routes (authenticated + admin + super admin)
	superAdmin := admin.PathPrefix("").Subrouter()
	superAdmin.Use(middleware.RequireSuperAdmin)
	superAdmin.HandleFunc("/admin/users/{id}/promote", userHandler.PromoteToAdmin).Methods("POST")
	superAdmin.HandleFunc("/admin/users/{id}/demote", userHandler.DemoteAdmin).Methods("POST")

	// Uploads directory (user photos, dog photos)
	router.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))

	// Serve specific HTML pages without .html extension
	router.HandleFunc("/verify", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./frontend/verify.html")
	}).Methods("GET")

	// Static files
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./frontend")))

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
