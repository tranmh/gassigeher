package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

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

	// Demo tenant: Ensure demo tenant exists with sample data
	demoSeedService := services.NewDemoSeedService(db)
	if err := demoSeedService.EnsureDemoTenant(); err != nil {
		log.Printf("Warning: Failed to ensure demo tenant: %v", err)
		// Don't exit - demo is optional
	}

	// Initialize router
	router := mux.NewRouter()

	// Apply global middleware
	// SaaS Phase 5: Global rate limiter (100 requests/second burst 200 per IP)
	router.Use(middleware.GlobalRateLimit(100, 200))
	router.Use(middleware.MetricsMiddleware) // Collect request metrics
	router.Use(middleware.LoggingMiddleware)
	router.Use(middleware.SecurityHeadersMiddleware)
	router.Use(middleware.CORSMiddleware(cfg.BaseURL))

	// SaaS: Tenant resolution middleware (resolves subdomain to tenant_id)
	// Must be after CORS but before auth middleware
	tenantRepo := repository.NewTenantRepository(db)
	if cfg.BaseDomain != "" {
		router.Use(middleware.TenantMiddleware(tenantRepo, cfg.BaseDomain))
		log.Printf("SaaS mode: Tenant middleware enabled for domain *.%s", cfg.BaseDomain)

		// SaaS Phase: Per-tenant rate limiting (after tenant is resolved)
		// Free tier: 30 req/s tenant-wide, 20 req/s per-IP
		// Pro tier: 100 req/s tenant-wide, 50 req/s per-IP
		router.Use(middleware.TenantRateLimit(db))
		log.Println("SaaS mode: Per-tenant rate limiting enabled")
	}

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
	healthHandler := handlers.NewHealthHandler(db)
	walkReportHandler := handlers.NewWalkReportHandler(db, cfg)
	colorCategoryHandler := handlers.NewColorCategoryHandler(db, cfg)
	colorRequestHandler := handlers.NewColorRequestHandler(db, cfg)
	userColorHandler := handlers.NewUserColorHandler(db, cfg)
	themeHandler := handlers.NewThemeHandler(db)
	tenantHandler := handlers.NewTenantHandler(db, cfg)
	centralAdminHandler := handlers.NewCentralAdminHandler(db, cfg)
	contactHandler := handlers.NewContactHandler(cfg)
	demoHandler := handlers.NewDemoHandler(db)
	auditHandler := handlers.NewAuditHandler(db)
	metricsHandler := handlers.NewMetricsHandler()
	consentHandler := handlers.NewConsentHandler(db)
	featureFlagHandler := handlers.NewFeatureFlagHandler(db)

	// Initialize global cache service
	cacheService := services.NewDefaultCacheService()
	cacheHandler := handlers.NewCacheHandler(cacheService)
	log.Println("Cache service initialized (5min TTL, 10000 max entries)")

	// SaaS: Initialize Stripe service and billing handler
	var stripeService *services.StripeService
	if cfg.StripeSecretKey != "" {
		stripeService = services.NewStripeService(
			cfg.StripeSecretKey,
			cfg.StripePublishableKey,
			cfg.StripePriceMonthly,
			cfg.StripePriceYearly,
			cfg.BaseURL,
		)
		if cfg.StripeWebhookSecret != "" {
			stripeService.SetWebhookSecret(cfg.StripeWebhookSecret)
		}
		log.Println("Stripe payment service initialized")
	}
	billingHandler := handlers.NewBillingHandler(db, stripeService)

	// Infrastructure endpoints (unversioned - for monitoring tools)
	router.HandleFunc("/api/health", healthHandler.Health).Methods("GET")
	router.HandleFunc("/api/ready", healthHandler.Ready).Methods("GET")
	router.HandleFunc("/api/health/detailed", healthHandler.DetailedHealth).Methods("GET")
	router.HandleFunc("/metrics", metricsHandler.GetPrometheusMetrics).Methods("GET")
	router.HandleFunc("/api/metrics", metricsHandler.GetMetrics).Methods("GET")

	// API Version redirect middleware (redirects legacy /api/* to /api/v1/*)
	// Excluded: /api/health, /api/ready, /api/version, /api/metrics
	router.Use(middleware.APIVersionRedirect)

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

	// Version endpoint (public, unversioned for infrastructure)
	router.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(version.Get())
	}).Methods("GET")

	// ========================================
	// API v1 Routes (versioned endpoints)
	// ========================================

	// Public routes
	// Auth endpoint rate limiting: 3 requests per minute per IP (conservative)
	// Shared bucket across register, forgot-password, reset-password
	registerRoute := router.PathPrefix("/api/v1/auth/register").Subrouter()
	registerRoute.Use(middleware.RateLimitAuthEndpoint)
	registerRoute.HandleFunc("", authHandler.Register).Methods("POST")

	router.HandleFunc("/api/v1/auth/verify-email", authHandler.VerifyEmail).Methods("POST")

	// BUG FIX #6: Add rate limiting to login endpoint (5/min - separate from auth endpoints)
	loginRoute := router.PathPrefix("/api/v1/auth/login").Subrouter()
	loginRoute.Use(middleware.RateLimitLogin)
	loginRoute.HandleFunc("", authHandler.Login).Methods("POST")
	// DONE: BUG #6 - Rate limiting applied to login

	// Password reset endpoints with auth rate limiting (3/min shared bucket)
	forgotPasswordRoute := router.PathPrefix("/api/v1/auth/forgot-password").Subrouter()
	forgotPasswordRoute.Use(middleware.RateLimitAuthEndpoint)
	forgotPasswordRoute.HandleFunc("", authHandler.ForgotPassword).Methods("POST")

	resetPasswordRoute := router.PathPrefix("/api/v1/auth/reset-password").Subrouter()
	resetPasswordRoute.Use(middleware.RateLimitAuthEndpoint)
	resetPasswordRoute.HandleFunc("", authHandler.ResetPassword).Methods("POST")

	// Reactivation request (public - for deactivated users)
	router.HandleFunc("/api/v1/reactivation-requests", reactivationHandler.CreateRequest).Methods("POST")

	// Booking time routes (public - for time slot availability)
	router.HandleFunc("/api/v1/booking-times/available", bookingTimeHandler.GetAvailableSlots).Methods("GET")
	router.HandleFunc("/api/v1/booking-times/rules-for-date", bookingTimeHandler.GetRulesForDate).Methods("GET")
	router.HandleFunc("/api/v1/holidays", holidayHandler.GetHolidays).Methods("GET")

	// Featured dogs (public - for homepage)
	router.HandleFunc("/api/v1/dogs/featured", dogHandler.GetFeaturedDogs).Methods("GET")

	// Color categories (public - for filters)
	router.HandleFunc("/api/v1/colors", colorCategoryHandler.ListColors).Methods("GET")

	// Site logo (public - for displaying logo on all pages)
	router.HandleFunc("/api/v1/settings/logo", settingsHandler.GetLogo).Methods("GET")

	// WhatsApp group settings (public - for displaying join button)
	router.HandleFunc("/api/v1/settings/whatsapp", settingsHandler.GetWhatsAppSettings).Methods("GET")

	// Theme CSS (public - for dynamic styling)
	router.HandleFunc("/api/v1/theme/css", themeHandler.GetCSS).Methods("GET")
	router.HandleFunc("/api/v1/theme/presets", themeHandler.GetPresets).Methods("GET")

	// Tenant registration (public - for self-service signup)
	router.HandleFunc("/api/v1/tenants/register", tenantHandler.Register).Methods("POST")
	router.HandleFunc("/api/v1/tenants/check-slug", tenantHandler.CheckSlug).Methods("GET")

	// Contact form (public - for landing page inquiries)
	router.HandleFunc("/api/v1/contact", contactHandler.Submit).Methods("POST")

	// SaaS Stripe webhook (public - verified by signature)
	router.HandleFunc("/api/v1/billing/webhook", billingHandler.HandleWebhook).Methods("POST")

	// Demo tenant endpoints (public - for demo landing page)
	router.HandleFunc("/api/v1/demo/credentials", demoHandler.GetCredentials).Methods("GET")
	router.HandleFunc("/api/v1/demo/status", demoHandler.GetStatus).Methods("GET")

	// Consent versions (public - for displaying current ToS/Privacy versions)
	router.HandleFunc("/api/v1/consent/versions", consentHandler.GetCurrentConsentVersions).Methods("GET")

	// Protected routes (authenticated users)
	protected := router.PathPrefix("/api/v1").Subrouter()
	protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	protected.Use(middleware.AddVersionHeader)

	// Auth
	protected.HandleFunc("/auth/change-password", authHandler.ChangePassword).Methods("PUT")

	// Users
	protected.HandleFunc("/users/me", userHandler.GetMe).Methods("GET")
	protected.HandleFunc("/users/me", userHandler.UpdateMe).Methods("PUT")
	protected.HandleFunc("/users/me/photo", userHandler.UploadPhoto).Methods("POST")
	protected.HandleFunc("/users/me/export", userHandler.ExportMyData).Methods("GET")
	protected.HandleFunc("/users/me/consent", consentHandler.GetConsentStatus).Methods("GET")
	protected.HandleFunc("/users/me/consent/history", consentHandler.GetConsentHistory).Methods("GET")
	protected.HandleFunc("/users/me/consent", consentHandler.UpdateConsent).Methods("POST")
	protected.HandleFunc("/users/me", userHandler.DeleteAccount).Methods("DELETE")

	// Tenant info (authenticated users)
	protected.HandleFunc("/tenants/me", tenantHandler.GetCurrentTenant).Methods("GET")

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

	// Color requests (authenticated users)
	protected.HandleFunc("/color-requests", colorRequestHandler.CreateRequest).Methods("POST")
	protected.HandleFunc("/color-requests", colorRequestHandler.ListRequests).Methods("GET")
	protected.HandleFunc("/color-requests/{id}", colorRequestHandler.GetRequest).Methods("GET")

	// Walk reports (authenticated users)
	protected.HandleFunc("/walk-reports", walkReportHandler.CreateReport).Methods("POST")
	protected.HandleFunc("/walk-reports/by-booking/{bookingId}", walkReportHandler.GetReportByBooking).Methods("GET")
	protected.HandleFunc("/walk-reports/{id}", walkReportHandler.GetReport).Methods("GET")
	protected.HandleFunc("/walk-reports/{id}", walkReportHandler.UpdateReport).Methods("PUT")
	protected.HandleFunc("/walk-reports/{id}", walkReportHandler.DeleteReport).Methods("DELETE")
	protected.HandleFunc("/walk-reports/{id}/photos", walkReportHandler.UploadPhoto).Methods("POST")
	protected.HandleFunc("/walk-reports/{id}/photos/{photoId}", walkReportHandler.DeletePhoto).Methods("DELETE")
	protected.HandleFunc("/dogs/{id}/walk-reports", walkReportHandler.GetDogWalkReports).Methods("GET")

	// SaaS Billing routes (authenticated users)
	protected.HandleFunc("/billing/subscription", billingHandler.GetSubscription).Methods("GET")
	protected.HandleFunc("/billing/plans", billingHandler.GetPlans).Methods("GET")
	protected.HandleFunc("/billing/usage", billingHandler.GetUsage).Methods("GET")
	protected.HandleFunc("/billing/checkout", billingHandler.CreateCheckout).Methods("POST")
	protected.HandleFunc("/billing/portal", billingHandler.CreateBillingPortal).Methods("POST")
	protected.HandleFunc("/billing/cancel", billingHandler.CancelSubscription).Methods("POST")

	// Feature flags (authenticated users - check if flags are enabled)
	protected.HandleFunc("/feature-flags/{key}/check", featureFlagHandler.CheckFlag).Methods("GET")
	protected.HandleFunc("/feature-flags/check", featureFlagHandler.CheckMultipleFlags).Methods("POST")

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
	admin.HandleFunc("/settings/logo", settingsHandler.UploadLogo).Methods("POST")
	admin.HandleFunc("/settings/logo", settingsHandler.ResetLogo).Methods("DELETE")

	// Experience requests management (admin only)
	admin.HandleFunc("/experience-requests/{id}/approve", experienceHandler.ApproveRequest).Methods("PUT")
	admin.HandleFunc("/experience-requests/{id}/deny", experienceHandler.DenyRequest).Methods("PUT")

	// Color requests management (admin only)
	admin.HandleFunc("/color-requests/{id}/approve", colorRequestHandler.ApproveRequest).Methods("PUT")
	admin.HandleFunc("/color-requests/{id}/deny", colorRequestHandler.DenyRequest).Methods("PUT")

	// User colors management (admin only)
	admin.HandleFunc("/users/{id}/colors", userColorHandler.GetUserColors).Methods("GET")
	admin.HandleFunc("/users/{id}/colors", userColorHandler.AddColorToUser).Methods("POST")
	admin.HandleFunc("/users/{id}/colors", userColorHandler.SetUserColors).Methods("PUT")
	admin.HandleFunc("/users/{id}/colors/{colorId}", userColorHandler.RemoveColorFromUser).Methods("DELETE")

	// User management (admin only)
	admin.HandleFunc("/users", userHandler.ListUsers).Methods("GET")
	admin.HandleFunc("/users", userHandler.AdminCreateUser).Methods("POST")
	admin.HandleFunc("/users/{id}", userHandler.GetUser).Methods("GET")
	admin.HandleFunc("/users/{id}", userHandler.AdminUpdateUser).Methods("PUT")
	admin.HandleFunc("/users/{id}/activate", userHandler.ActivateUser).Methods("PUT")
	admin.HandleFunc("/users/{id}/deactivate", userHandler.DeactivateUser).Methods("PUT")
	admin.HandleFunc("/users/{id}", userHandler.AdminDeleteUser).Methods("DELETE") // Super-admin only

	// Reactivation requests management (admin only)
	admin.HandleFunc("/reactivation-requests", reactivationHandler.ListRequests).Methods("GET")
	admin.HandleFunc("/reactivation-requests/{id}/approve", reactivationHandler.ApproveRequest).Methods("PUT")
	admin.HandleFunc("/reactivation-requests/{id}/deny", reactivationHandler.DenyRequest).Methods("PUT")

	// Admin dashboard (admin only)
	admin.HandleFunc("/admin/stats", dashboardHandler.GetStats).Methods("GET")
	admin.HandleFunc("/admin/activity", dashboardHandler.GetRecentActivity).Methods("GET")

	// Audit logs (admin only)
	admin.HandleFunc("/admin/audit-logs", auditHandler.ListAuditLogs).Methods("GET")
	admin.HandleFunc("/admin/audit-logs/actions", auditHandler.GetAuditLogActions).Methods("GET")
	admin.HandleFunc("/admin/audit-logs/entity-types", auditHandler.GetAuditLogEntityTypes).Methods("GET")

	// Feature flags (admin only - tenant-level management)
	admin.HandleFunc("/admin/feature-flags", featureFlagHandler.GetTenantFlags).Methods("GET")
	admin.HandleFunc("/admin/feature-flags/{id}", featureFlagHandler.SetTenantFlag).Methods("PUT")
	admin.HandleFunc("/admin/feature-flags/{id}", featureFlagHandler.ResetTenantFlag).Methods("DELETE")

	// Cache management (admin only)
	admin.HandleFunc("/admin/cache/stats", cacheHandler.GetStats).Methods("GET")
	admin.HandleFunc("/admin/cache", cacheHandler.Clear).Methods("DELETE")
	admin.HandleFunc("/admin/cache/prefix", cacheHandler.ClearPrefix).Methods("DELETE")

	// Booking time management (admin only)
	admin.HandleFunc("/admin/booking-times/rules", bookingTimeHandler.GetRules).Methods("GET")
	admin.HandleFunc("/admin/booking-times/rules", bookingTimeHandler.UpdateRules).Methods("PUT")
	admin.HandleFunc("/admin/booking-times/rules", bookingTimeHandler.CreateRule).Methods("POST")
	admin.HandleFunc("/admin/booking-times/rules/{id}", bookingTimeHandler.DeleteRule).Methods("DELETE")

	// Holiday management (admin only)
	admin.HandleFunc("/admin/holidays", holidayHandler.CreateHoliday).Methods("POST")
	admin.HandleFunc("/admin/holidays/{id}", holidayHandler.UpdateHoliday).Methods("PUT")
	admin.HandleFunc("/admin/holidays/{id}", holidayHandler.DeleteHoliday).Methods("DELETE")

	// Theme management (admin only)
	admin.HandleFunc("/admin/theme", themeHandler.GetCurrentTheme).Methods("GET")
	admin.HandleFunc("/admin/theme", themeHandler.UpdateTheme).Methods("PUT")

	// Tenant management (admin only)
	admin.HandleFunc("/admin/tenant", tenantHandler.GetCurrentTenant).Methods("GET")
	admin.HandleFunc("/admin/tenant", tenantHandler.UpdateTenant).Methods("PUT")
	admin.HandleFunc("/admin/tenant/stats", tenantHandler.GetTenantStats).Methods("GET")

	// Booking approval management (admin only)
	admin.HandleFunc("/bookings/pending-approvals", bookingHandler.GetPendingApprovals).Methods("GET")
	admin.HandleFunc("/bookings/{id}/approve", bookingHandler.ApprovePendingBooking).Methods("PUT")
	admin.HandleFunc("/bookings/{id}/reject", bookingHandler.RejectPendingBooking).Methods("PUT")

	// DONE: Phase 4 - Super Admin routes (authenticated + admin + super admin)
	superAdmin := admin.PathPrefix("").Subrouter()
	superAdmin.Use(middleware.RequireSuperAdmin)
	superAdmin.HandleFunc("/admin/users/{id}/promote", userHandler.PromoteToAdmin).Methods("POST")
	superAdmin.HandleFunc("/admin/users/{id}/demote", userHandler.DemoteAdmin).Methods("POST")
	superAdmin.HandleFunc("/admin/users/{id}/impersonate", userHandler.ImpersonateUser).Methods("POST")

	// Color category management (super-admin only)
	superAdmin.HandleFunc("/colors", colorCategoryHandler.CreateColor).Methods("POST")
	superAdmin.HandleFunc("/colors/{id}", colorCategoryHandler.GetColor).Methods("GET")
	superAdmin.HandleFunc("/colors/{id}", colorCategoryHandler.UpdateColor).Methods("PUT")
	superAdmin.HandleFunc("/colors/{id}", colorCategoryHandler.DeleteColor).Methods("DELETE")
	superAdmin.HandleFunc("/colors/{id}/stats", colorCategoryHandler.GetColorStats).Methods("GET")
	// NOTE: EndImpersonation is on 'protected' router (not superAdmin) because when
	// impersonating a regular user, the token has is_super_admin=false
	protected.HandleFunc("/end-impersonation", userHandler.EndImpersonation).Methods("POST")

	// SaaS: Central Admin routes (platform-wide administration)
	centralAdmin := protected.PathPrefix("/central-admin").Subrouter()
	centralAdmin.Use(middleware.RequireCentralAdmin)
	centralAdmin.HandleFunc("/stats", centralAdminHandler.GetPlatformStats).Methods("GET")
	centralAdmin.HandleFunc("/tenants", centralAdminHandler.ListTenants).Methods("GET")
	centralAdmin.HandleFunc("/tenants/{id}", centralAdminHandler.GetTenant).Methods("GET")
	centralAdmin.HandleFunc("/tenants/{id}", centralAdminHandler.UpdateTenant).Methods("PUT")
	centralAdmin.HandleFunc("/tenants/{id}/activate", centralAdminHandler.ActivateTenant).Methods("POST")
	centralAdmin.HandleFunc("/tenants/{id}/deactivate", centralAdminHandler.DeactivateTenant).Methods("POST")
	centralAdmin.HandleFunc("/tenants/{id}/users", centralAdminHandler.GetTenantUsers).Methods("GET")
	centralAdmin.HandleFunc("/tenants/{id}/export", centralAdminHandler.ExportTenantData).Methods("GET")
	centralAdmin.HandleFunc("/admins", centralAdminHandler.ListCentralAdmins).Methods("GET")
	centralAdmin.HandleFunc("/admins/{id}/promote", centralAdminHandler.PromoteToCentralAdmin).Methods("POST")
	centralAdmin.HandleFunc("/admins/{id}/demote", centralAdminHandler.DemoteFromCentralAdmin).Methods("POST")
	centralAdmin.HandleFunc("/users/search", centralAdminHandler.SearchUsers).Methods("GET")

	// Feature flags management (central admin only)
	centralAdmin.HandleFunc("/feature-flags", featureFlagHandler.ListFlags).Methods("GET")
	centralAdmin.HandleFunc("/feature-flags", featureFlagHandler.CreateFlag).Methods("POST")
	centralAdmin.HandleFunc("/feature-flags/{id}", featureFlagHandler.GetFlag).Methods("GET")
	centralAdmin.HandleFunc("/feature-flags/{id}", featureFlagHandler.UpdateFlag).Methods("PUT")
	centralAdmin.HandleFunc("/feature-flags/{id}", featureFlagHandler.DeleteFlag).Methods("DELETE")

	// Uploads directory (user photos, dog photos) - must remain on filesystem
	router.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))

	// Get embedded landing page filesystem (for SaaS landing page)
	landingFS, err := static.LandingFS()
	if err != nil {
		log.Fatalf("Failed to get embedded landing pages: %v", err)
	}
	// Serve landing pages at /landing/
	router.PathPrefix("/landing/").Handler(http.StripPrefix("/landing/", http.FileServer(http.FS(landingFS))))

	// Get embedded central admin filesystem (for SaaS platform administration)
	centralFS, err := static.CentralFS()
	if err != nil {
		log.Fatalf("Failed to get embedded central admin pages: %v", err)
	}
	// Serve central admin pages at /central/
	router.PathPrefix("/central/").Handler(http.StripPrefix("/central/", http.FileServer(http.FS(centralFS))))

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

	// Root path: redirect to landing page if no tenant, otherwise serve frontend
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tenantID := middleware.GetTenantID(r)
		if tenantID == 0 {
			// No tenant subdomain - redirect to landing page
			http.Redirect(w, r, "/landing/", http.StatusTemporaryRedirect)
			return
		}
		// Tenant exists - serve the frontend index
		serveEmbeddedFile(w, r, frontendFS, "index.html")
	}).Methods("GET")

	// Static files from embedded frontend
	router.PathPrefix("/").Handler(http.FileServer(http.FS(frontendFS)))

	// Start server with graceful shutdown
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel to listen for shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on port %s...", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Block until shutdown signal received
	sig := <-quit
	log.Printf("Received signal %v, initiating graceful shutdown...", sig)

	// Create context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server gracefully
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped gracefully")
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
