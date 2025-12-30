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
	"strings"
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
	resetTenant := flag.String("reset-tenant", "", "Reset a specific local dev tenant by slug (local dev only)")
	resetAllTenants := flag.Bool("reset-all-tenants", false, "Reset all local dev tenants (local dev only)")
	flag.Parse()

	// Load environment variables from .env file if it exists
	if _, err := os.Stat(*envPath); os.IsNotExist(err) {
		log.Printf("No .env file found at %s, using environment variables", *envPath)
	} else {
		if err := godotenv.Load(*envPath); err != nil {
			log.Fatalf("Error loading .env file from %s: %v", *envPath, err)
		}
		log.Printf("Loaded configuration from: %s", *envPath)
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

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Initialize database with multi-database support
	dbConfig := cfg.GetDBConfig()
	db, dialect, err := database.InitializeWithConfig(dbConfig)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Log database type for transparency
	log.Printf("Using database: %s", dialect.Name())

	// Check if this is a fresh install BEFORE running migrations
	// (After migrations, schema_migrations table will have entries)
	isFreshInstall := database.IsFreshInstall(db)
	if isFreshInstall {
		log.Println("Detected fresh installation (no existing schema)")
	}

	// Run migrations with dialect support
	if err := database.RunMigrationsWithDialect(db, dialect); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Enforce strong JWT secret on fresh installations
	// This auto-generates a secure secret if the current one is weak/default
	newSecret, changed, err := database.EnforceStrongJWTSecret(isFreshInstall, cfg.JWTSecret, *envPath)
	if err != nil {
		log.Fatalf("Failed to enforce strong JWT secret: %v", err)
	}
	if changed {
		// Update the config with the new secret for this session
		cfg.JWTSecret = newSecret
		log.Println("JWT secret has been updated for this session")
	}

	// Enforce strong metrics password on fresh installations
	// This auto-generates a secure password for Prometheus /metrics Basic Auth
	newMetricsPassword, metricsChanged, err := database.EnforceStrongMetricsPassword(isFreshInstall, cfg.MetricsPassword, *envPath)
	if err != nil {
		log.Fatalf("Failed to enforce strong metrics password: %v", err)
	}
	if metricsChanged {
		// Update the config with the new password for this session
		cfg.MetricsPassword = newMetricsPassword
		log.Println("Metrics password has been updated for this session")
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

	// Initialize Sentry for error tracking (optional - enabled via SENTRY_DSN)
	var sentryService *services.SentryService
	if cfg.SentryDSN != "" {
		sentryService, err = services.NewSentryService(&services.SentryConfig{
			DSN:         cfg.SentryDSN,
			Environment: cfg.SentryEnvironment,
			Release:     cfg.SentryRelease,
			ServerName:  "gassigeher",
		})
		if err != nil {
			log.Printf("Warning: Failed to initialize Sentry: %v", err)
			// Don't exit - Sentry is optional
		} else {
			defer sentryService.Flush(2 * time.Second)
		}
	}

	// Initialize business metrics collection
	middleware.InitBusinessMetrics(db)

	// Demo tenant: Ensure demo tenant exists with sample data
	demoSeedService := services.NewDemoSeedService(db)
	if err := demoSeedService.EnsureDemoTenant(); err != nil {
		log.Printf("Warning: Failed to ensure demo tenant: %v", err)
		// Don't exit - demo is optional
	}

	// Local development: Handle tenant reset commands (only in local dev mode)
	if cfg.IsLocalDevelopment() {
		localDevService := services.NewLocalDevSeedService(db)

		// Handle reset commands
		if *resetTenant != "" {
			log.Printf("Resetting local dev tenant: %s", *resetTenant)
			if err := localDevService.ResetTenant(*resetTenant); err != nil {
				log.Fatalf("Failed to reset tenant %s: %v", *resetTenant, err)
			}
			log.Printf("Tenant '%s' reset successfully", *resetTenant)
			return
		}

		if *resetAllTenants {
			log.Println("Resetting all local dev tenants...")
			if err := localDevService.ResetAllTenants(); err != nil {
				log.Fatalf("Failed to reset all tenants: %v", err)
			}
			log.Println("All local dev tenants reset successfully")
			return
		}

		// Ensure local dev tenants exist
		if err := localDevService.EnsureLocalDevTenants(); err != nil {
			log.Printf("Warning: Failed to ensure local dev tenants: %v", err)
			// Don't exit - local dev tenants are optional
		}
		log.Println("Local development mode: tenants demo1-4 available")
	} else if *resetTenant != "" || *resetAllTenants {
		log.Fatal("Tenant reset is only available in local development mode (BASE_DOMAIN must end with .local)")
	}

	// Initialize router
	router := mux.NewRouter()

	// Apply global middleware
	// Sentry recovery middleware (captures panics and sends to Sentry)
	if sentryService != nil && sentryService.IsEnabled() {
		router.Use(sentryService.RecoveryMiddleware)
		log.Println("Sentry panic recovery middleware enabled")
	}

	// SaaS Phase 5: Global rate limiter (100 requests/second burst 200 per IP)
	router.Use(middleware.GlobalRateLimit(100, 200))
	router.Use(middleware.MetricsMiddleware) // Collect request metrics
	router.Use(middleware.LoggingMiddleware)
	router.Use(middleware.BlockDangerousMethods) // SECURITY: GASSI-2025-004 - Block TRACE/TRACK
	router.Use(middleware.SecurityHeadersMiddleware)
	router.Use(middleware.CORSMiddleware(cfg.BaseURL))

	// Tenant resolution middleware
	// Must be after CORS but before auth middleware
	tenantRepo := repository.NewTenantRepository(db)
	if cfg.BaseDomain != "" {
		// SaaS-Mode: Resolve subdomain to tenant_id
		router.Use(middleware.TenantMiddleware(tenantRepo, cfg.BaseDomain))
		log.Printf("SaaS mode: Tenant middleware enabled for domain *.%s", cfg.BaseDomain)

		// SaaS Phase: Per-tenant rate limiting (after tenant is resolved)
		// Free tier: 30 req/s tenant-wide, 20 req/s per-IP
		// Pro tier: 100 req/s tenant-wide, 50 req/s per-IP
		router.Use(middleware.TenantRateLimit(db))
		log.Println("SaaS mode: Per-tenant rate limiting enabled")
	} else {
		// Simple-Mode: Inject default tenant (id=0) for all requests
		// This ensures all repository queries always filter by tenant_id
		router.Use(middleware.SimpleModeMiddleware)
		log.Println("Simple mode: Using default tenant (id=0)")
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
	marketingHandler := handlers.NewMarketingHandler(db)
	promoCodeHandler := handlers.NewPromoCodeHandler(db, cfg)
	importHandler := handlers.NewImportHandler(db)

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
	billingHandler := handlers.NewBillingHandler(db, cfg, stripeService)

	// Infrastructure endpoints (unversioned - for monitoring tools)
	router.HandleFunc("/api/health", healthHandler.Health).Methods("GET")
	router.HandleFunc("/api/ready", healthHandler.Ready).Methods("GET")
	router.HandleFunc("/api/health/detailed", healthHandler.DetailedHealth).Methods("GET")

	// Metrics endpoints protected with Basic Auth (standard Prometheus pattern)
	router.HandleFunc("/metrics", middleware.WrapWithMetricsAuth(
		metricsHandler.GetPrometheusMetrics, cfg.MetricsUsername, cfg.MetricsPassword,
	)).Methods("GET")
	router.HandleFunc("/api/metrics", middleware.WrapWithMetricsAuth(
		metricsHandler.GetMetrics, cfg.MetricsUsername, cfg.MetricsPassword,
	)).Methods("GET")

	// NOTE: API Version redirect is applied via WrapWithVersionRedirect at server startup
	// (router.Use() doesn't work because gorilla/mux runs middleware AFTER route matching)

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

	// Tenant branding (public - for dynamic content on index.html)
	router.HandleFunc("/api/v1/tenant/branding", tenantHandler.GetBranding).Methods("GET")

	// Tenant registration (public - for self-service signup)
	router.HandleFunc("/api/v1/tenants/register", tenantHandler.Register).Methods("POST")

	// Marketing public routes
	router.HandleFunc("/api/v1/marketing/fomo", marketingHandler.GetActiveFOMO).Methods("GET")
	router.HandleFunc("/api/v1/marketing/referral/{code}", marketingHandler.ValidateReferralCode).Methods("GET")
	router.HandleFunc("/api/v1/marketing/references", marketingHandler.ListReferenceEntries).Methods("GET")
	router.HandleFunc("/api/v1/promo-codes/validate/{code}", promoCodeHandler.ValidatePromoCode).Methods("GET")
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

	// CSRF protection for state-changing requests
	csrfMiddleware := middleware.NewCSRFMiddleware()
	csrfMiddleware.SetSecure(strings.HasPrefix(cfg.BaseURL, "https"))
	protected.Use(csrfMiddleware.Middleware)

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

	// SaaS Billing routes (super-admin or central-admin only)
	// Access restricted to tenant super-admin for their own billing, or central admin for impersonation
	billing := protected.PathPrefix("/billing").Subrouter()
	billing.Use(middleware.RequireTenantSuperAdminOrCentralAdmin)
	billing.HandleFunc("/subscription", billingHandler.GetSubscription).Methods("GET")
	billing.HandleFunc("/plans", billingHandler.GetPlans).Methods("GET")
	billing.HandleFunc("/usage", billingHandler.GetUsage).Methods("GET")
	billing.HandleFunc("/checkout", billingHandler.CreateCheckout).Methods("POST")
	billing.HandleFunc("/portal", billingHandler.CreateBillingPortal).Methods("POST")
	billing.HandleFunc("/cancel", billingHandler.CancelSubscription).Methods("POST")
	billing.HandleFunc("/test-upgrade", billingHandler.TestUpgrade).Methods("POST") // Test mode only
	billing.HandleFunc("/invoices", billingHandler.GetInvoices).Methods("GET")
	billing.HandleFunc("/invoices/{id:[0-9]+}", billingHandler.GetInvoice).Methods("GET")
	billing.HandleFunc("/invoices/{id:[0-9]+}/pdf", billingHandler.DownloadInvoicePDF).Methods("GET")

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
	admin.HandleFunc("/dogs/{id}/photo", dogHandler.DeleteDogPhoto).Methods("DELETE")
	admin.HandleFunc("/dogs/{id}/availability", dogHandler.ToggleAvailability).Methods("PUT")
	admin.HandleFunc("/dogs/{id}/featured", dogHandler.SetFeatured).Methods("PUT")

	// Blocked dates management (admin only)
	admin.HandleFunc("/blocked-dates", blockedDateHandler.CreateBlockedDate).Methods("POST")
	admin.HandleFunc("/blocked-dates/{id}", blockedDateHandler.DeleteBlockedDate).Methods("DELETE")

	// Booking management (admin only)
	admin.HandleFunc("/bookings/{id}/move", bookingHandler.MoveBooking).Methods("PUT")

	// System settings (admin only)
	admin.HandleFunc("/settings", settingsHandler.GetAllSettings).Methods("GET")
	// Logo routes must be registered BEFORE {key} to avoid being matched by the wildcard
	admin.HandleFunc("/settings/logo", settingsHandler.UploadLogo).Methods("POST")
	admin.HandleFunc("/settings/logo", settingsHandler.ResetLogo).Methods("DELETE")
	admin.HandleFunc("/settings/{key}", settingsHandler.UpdateSetting).Methods("PUT")

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
	admin.HandleFunc("/admin/tenant/branding", tenantHandler.UpdateBranding).Methods("PUT")
	admin.HandleFunc("/admin/tenant/export", tenantHandler.ExportTenantData).Methods("GET")

	// Import management (admin only)
	admin.HandleFunc("/admin/import/dogs/preview", importHandler.PreviewImport).Methods("POST")
	admin.HandleFunc("/admin/import/dogs", importHandler.ExecuteImport).Methods("POST")
	admin.HandleFunc("/admin/import/dogs/template", importHandler.GetImportTemplate).Methods("GET")

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
	centralAdmin.HandleFunc("/tenants/inactive", centralAdminHandler.GetInactiveTenants).Methods("GET")
	centralAdmin.HandleFunc("/tenants/activity", centralAdminHandler.GetTenantActivity).Methods("GET")
	centralAdmin.HandleFunc("/impersonate/{userId}", centralAdminHandler.ImpersonateTenantUser).Methods("POST")

	// End impersonation uses special middleware that allows impersonation tokens
	// (not just central admin) to end their own session - BUG FIX #4
	endImpersonationHandler := middleware.AllowImpersonationEnd(http.HandlerFunc(centralAdminHandler.EndCentralImpersonation))
	protected.Handle("/central-admin/end-impersonation", endImpersonationHandler).Methods("POST")
	centralAdmin.HandleFunc("/tenants/{id}", centralAdminHandler.GetTenant).Methods("GET")
	centralAdmin.HandleFunc("/tenants/{id}", centralAdminHandler.UpdateTenant).Methods("PUT")
	centralAdmin.HandleFunc("/tenants/{id}/activate", centralAdminHandler.ActivateTenant).Methods("POST")
	centralAdmin.HandleFunc("/tenants/{id}/deactivate", centralAdminHandler.DeactivateTenant).Methods("POST")
	centralAdmin.HandleFunc("/tenants/{id}/users", centralAdminHandler.GetTenantUsers).Methods("GET")
	centralAdmin.HandleFunc("/tenants/{id}/export", centralAdminHandler.ExportTenantData).Methods("GET")
	centralAdmin.HandleFunc("/tenants/{id}/reset", centralAdminHandler.ResetLocalDevTenant).Methods("POST")
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

	// Marketing management (central admin only)
	centralAdmin.HandleFunc("/marketing/stats", marketingHandler.GetMarketingStats).Methods("GET")
	centralAdmin.HandleFunc("/marketing/campaigns", marketingHandler.ListCampaigns).Methods("GET")
	centralAdmin.HandleFunc("/marketing/campaigns", marketingHandler.CreateCampaign).Methods("POST")
	centralAdmin.HandleFunc("/marketing/campaigns/{id}", marketingHandler.GetCampaign).Methods("GET")
	centralAdmin.HandleFunc("/marketing/campaigns/{id}", marketingHandler.UpdateCampaign).Methods("PUT")
	centralAdmin.HandleFunc("/marketing/campaigns/{id}", marketingHandler.DeleteCampaign).Methods("DELETE")
	centralAdmin.HandleFunc("/marketing/referral-codes", marketingHandler.ListReferralCodes).Methods("GET")
	centralAdmin.HandleFunc("/marketing/referral-codes", marketingHandler.CreateReferralCode).Methods("POST")
	centralAdmin.HandleFunc("/marketing/referral-codes/{id}", marketingHandler.GetReferralCode).Methods("GET")
	centralAdmin.HandleFunc("/marketing/referral-codes/{id}", marketingHandler.UpdateReferralCode).Methods("PUT")
	centralAdmin.HandleFunc("/marketing/referral-codes/{id}/toggle", marketingHandler.ToggleReferralCode).Methods("PUT")
	centralAdmin.HandleFunc("/marketing/referral-codes/{id}", marketingHandler.DeleteReferralCode).Methods("DELETE")
	centralAdmin.HandleFunc("/marketing/references", marketingHandler.ListReferenceEntries).Methods("GET")
	centralAdmin.HandleFunc("/marketing/references/{id}", marketingHandler.GetReferenceEntry).Methods("GET")
	centralAdmin.HandleFunc("/marketing/references/{id}/approve", marketingHandler.ApproveReferenceEntry).Methods("PUT")
	centralAdmin.HandleFunc("/marketing/references/{id}", marketingHandler.DeleteReferenceEntry).Methods("DELETE")

	// Promo codes (central admin)
	centralAdmin.HandleFunc("/promo-codes", promoCodeHandler.GetAllPromoCodes).Methods("GET")
	centralAdmin.HandleFunc("/promo-codes", promoCodeHandler.CreatePromoCode).Methods("POST")
	centralAdmin.HandleFunc("/promo-codes/{id}", promoCodeHandler.GetPromoCode).Methods("GET")
	centralAdmin.HandleFunc("/promo-codes/{id}", promoCodeHandler.UpdatePromoCode).Methods("PUT")
	centralAdmin.HandleFunc("/promo-codes/{id}", promoCodeHandler.DeletePromoCode).Methods("DELETE")

	// Uploads directory (user photos, dog photos) - must remain on filesystem
	// BUG FIX #4: Use SafeFileServer to prevent null byte injection and path traversal
	router.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", handlers.SafeFileServer(http.Dir("./uploads"))))

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

	// Root path: serve frontend or redirect to landing page
	// Simple mode (no BASE_DOMAIN): always serve frontend index.html
	// SaaS mode (BASE_DOMAIN set): redirect to landing if no tenant subdomain
	saasMode := cfg.BaseDomain != ""
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !saasMode {
			// Simple mode - serve frontend directly
			serveEmbeddedFile(w, r, frontendFS, "index.html")
			return
		}
		// SaaS mode - check for tenant
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

	// Wrap router with API version redirect (rewrites /api/* to /api/v1/*)
	// This must be done as a wrapper, not middleware, because gorilla/mux
	// runs middleware AFTER route matching (so unversioned routes would 404)
	wrappedRouter := middleware.WrapWithVersionRedirect(router)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      wrappedRouter,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel to listen for shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Channel for server errors
	serverErr := make(chan error, 1)

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on port %s...", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Block until shutdown signal or server error
	select {
	case err := <-serverErr:
		log.Printf("Server failed to start: %v", err)
		// Allow cleanup to proceed
	case sig := <-quit:
		log.Printf("Received signal %v, initiating graceful shutdown...", sig)
	}

	// Stop cron service first (before HTTP server shutdown)
	cronService.Stop()
	log.Println("Cron service stopped")

	// BUG FIX: Stop AuthHandler's BruteForceService goroutine to prevent leak
	authHandler.Close()
	log.Println("Auth handler cleanup completed")

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
