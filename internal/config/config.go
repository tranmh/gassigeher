package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/tranmh/gassigeher/internal/database"
)

// Config holds the application configuration
type Config struct {
	// Database Type (sqlite, mysql, postgres)
	DBType string

	// SQLite Configuration
	DatabasePath string

	// MySQL/PostgreSQL Configuration
	DBHost     string
	DBPort     int
	DBName     string
	DBUser     string
	DBPassword string
	DBSSLMode  string // PostgreSQL: disable, require, verify-full

	// Alternative: Full connection string (overrides individual params if set)
	DBConnectionString string

	// Connection Pool (MySQL/PostgreSQL only)
	DBMaxOpenConns    int // Maximum open connections
	DBMaxIdleConns    int // Maximum idle connections
	DBConnMaxLifetime int // Connection max lifetime in minutes

	// JWT
	JWTSecret          string
	JWTExpirationHours int

	// Super Admin (DONE: replaces ADMIN_EMAILS)
	SuperAdminEmail string

	// Email Provider Selection
	EmailProvider string // "gmail" or "smtp"

	// Gmail API
	GmailClientID     string
	GmailClientSecret string
	GmailRefreshToken string
	GmailFromEmail    string

	// SMTP Configuration
	SMTPHost      string
	SMTPPort      int
	SMTPUsername  string
	SMTPPassword  string
	SMTPFromEmail string
	SMTPUseTLS    bool
	SMTPUseSSL    bool

	// BCC Admin Copy (works with all providers)
	EmailBCCAdmin string

	// Uploads
	UploadDir       string
	MaxUploadSizeMB int

	// System Settings
	BookingAdvanceDays      int
	CancellationNoticeHours int
	AutoDeactivationDays    int

	// Server
	Port    string
	BaseURL string // Base URL for email links (e.g., "https://gassigeher.com")

	// SaaS Multi-Tenancy
	SaaSMode   bool   // Enable multi-tenant mode
	BaseDomain string // Base domain for subdomain extraction (e.g., "gassigeher.org")

	// S3 Storage (Hetzner Object Storage)
	UseS3        bool   // Enable S3 storage (false = local filesystem)
	S3Endpoint   string // e.g., "fsn1.your-objectstorage.com"
	S3AccessKey  string
	S3SecretKey  string
	S3BucketName string
	S3Region     string
	S3PublicURL  string // e.g., "https://gassigeher-uploads.fsn1.your-objectstorage.com"
	S3UseSSL     bool

	// Stripe Payment Configuration (SaaS)
	StripeSecretKey      string // sk_live_xxx or sk_test_xxx
	StripePublishableKey string // pk_live_xxx or pk_test_xxx
	StripeWebhookSecret  string // whsec_xxx
	StripePriceMonthly   string // price_xxx for monthly subscription
	StripePriceYearly    string // price_xxx for yearly subscription

	// Billing Test Mode (for development without Stripe)
	BillingTestMode bool // Enable test upgrade/downgrade without Stripe

	// Contact Form
	ContactEmail string // Email address for contact form submissions

	// Monitoring - Sentry
	SentryDSN         string // Sentry DSN for error tracking (leave empty to disable)
	SentryEnvironment string // Environment name (production, staging, development)
	SentryRelease     string // Release version

	// Monitoring - Prometheus
	PrometheusEnabled bool   // Enable Prometheus metrics endpoint
	PrometheusPath    string // Path for metrics endpoint (default: /metrics)
	MetricsUsername   string // Basic auth username for /metrics endpoint
	MetricsPassword   string // Basic auth password for /metrics endpoint
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		// Database Type (default: sqlite)
		DBType: getEnv("DB_TYPE", "sqlite"),

		// SQLite Configuration
		DatabasePath: getEnv("DATABASE_PATH", "./gassigeher.db"),

		// MySQL/PostgreSQL Configuration
		DBHost:             getEnv("DB_HOST", "localhost"),
		DBPort:             getEnvAsInt("DB_PORT", 0), // 0 means use default (3306 for MySQL, 5432 for PostgreSQL)
		DBName:             getEnv("DB_NAME", "gassigeher"),
		DBUser:             getEnv("DB_USER", ""),
		DBPassword:         getEnv("DB_PASSWORD", ""),
		DBSSLMode:          getEnv("DB_SSLMODE", "disable"), // PostgreSQL SSL mode
		DBConnectionString: getEnv("DB_CONNECTION_STRING", ""),

		// Connection Pool Configuration (MySQL/PostgreSQL)
		DBMaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),  // Default: 25 connections
		DBMaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),   // Default: 5 idle connections
		DBConnMaxLifetime: getEnvAsInt("DB_CONN_MAX_LIFETIME", 5), // Default: 5 minutes

		// JWT (SECURITY: JWT_SECRET must be explicitly set in production)
		JWTSecret:          getEnvRequired("JWT_SECRET", "change-this-in-production-INSECURE"),
		JWTExpirationHours: getEnvAsInt("JWT_EXPIRATION_HOURS", 24),

		// Super Admin (DONE: replaces ADMIN_EMAILS)
		SuperAdminEmail: getEnv("SUPER_ADMIN_EMAIL", ""),

		// Email Provider (default: gmail for backward compatibility)
		EmailProvider: getEnv("EMAIL_PROVIDER", "gmail"),

		// Gmail API
		GmailClientID:     getEnv("GMAIL_CLIENT_ID", ""),
		GmailClientSecret: getEnv("GMAIL_CLIENT_SECRET", ""),
		GmailRefreshToken: getEnv("GMAIL_REFRESH_TOKEN", ""),
		GmailFromEmail:    getEnv("GMAIL_FROM_EMAIL", "noreply@gassigeher.com"),

		// SMTP Configuration
		SMTPHost:      getEnv("SMTP_HOST", ""),
		SMTPPort:      getEnvAsInt("SMTP_PORT", 0),
		SMTPUsername:  getEnv("SMTP_USERNAME", ""),
		SMTPPassword:  getEnv("SMTP_PASSWORD", ""),
		SMTPFromEmail: getEnv("SMTP_FROM_EMAIL", ""),
		SMTPUseTLS:    getEnvAsBool("SMTP_USE_TLS", false),
		SMTPUseSSL:    getEnvAsBool("SMTP_USE_SSL", false),

		// BCC Admin Copy
		EmailBCCAdmin: getEnv("EMAIL_BCC_ADMIN", ""),

		// Uploads
		UploadDir:       getEnv("UPLOAD_DIR", "./uploads"),
		MaxUploadSizeMB: getEnvAsInt("MAX_UPLOAD_SIZE_MB", 5),

		// System Settings
		BookingAdvanceDays:      getEnvAsInt("BOOKING_ADVANCE_DAYS", 14),
		CancellationNoticeHours: getEnvAsInt("CANCELLATION_NOTICE_HOURS", 12),
		AutoDeactivationDays:    getEnvAsInt("AUTO_DEACTIVATION_DAYS", 365),

		// Server
		Port:    getEnv("PORT", "8080"),
		BaseURL: getEnv("BASE_URL", "http://localhost:8080"),

		// SaaS Multi-Tenancy
		SaaSMode:   getEnvAsBool("SAAS_MODE", false),
		BaseDomain: getEnv("BASE_DOMAIN", ""), // e.g., "gassigeher.org"

		// S3 Storage (Hetzner Object Storage)
		UseS3:        getEnvAsBool("USE_S3", false),
		S3Endpoint:   getEnv("S3_ENDPOINT", ""),
		S3AccessKey:  getEnv("S3_ACCESS_KEY", ""),
		S3SecretKey:  getEnv("S3_SECRET_KEY", ""),
		S3BucketName: getEnv("S3_BUCKET_NAME", ""),
		S3Region:     getEnv("S3_REGION", ""),
		S3PublicURL:  getEnv("S3_PUBLIC_URL", ""),
		S3UseSSL:     getEnvAsBool("S3_USE_SSL", true),

		// Stripe Payment Configuration (SaaS)
		StripeSecretKey:      getEnv("STRIPE_SECRET_KEY", ""),
		StripePublishableKey: getEnv("STRIPE_PUBLISHABLE_KEY", ""),
		StripeWebhookSecret:  getEnv("STRIPE_WEBHOOK_SECRET", ""),
		StripePriceMonthly:   getEnv("STRIPE_PRICE_MONTHLY", ""),
		StripePriceYearly:    getEnv("STRIPE_PRICE_YEARLY", ""),

		// Billing Test Mode (auto-enabled for .local domains, or set BILLING_TEST_MODE=true)
		BillingTestMode: getEnvAsBool("BILLING_TEST_MODE", false),

		// Contact Form
		ContactEmail: getEnv("CONTACT_EMAIL", "kontakt@gassigeher.org"),

		// Monitoring - Sentry
		SentryDSN:         getEnv("SENTRY_DSN", ""),
		SentryEnvironment: getEnv("SENTRY_ENVIRONMENT", "development"),
		SentryRelease:     getEnv("SENTRY_RELEASE", ""),

		// Monitoring - Prometheus
		PrometheusEnabled: getEnvAsBool("PROMETHEUS_ENABLED", false),
		PrometheusPath:    getEnv("PROMETHEUS_PATH", "/metrics"),
		MetricsUsername:   getEnv("METRICS_USERNAME", "prometheus"),
		MetricsPassword:   getEnv("METRICS_PASSWORD", ""),
	}
}

// GetDBConfig builds a database configuration from the application config
// This is used to initialize the database connection with the correct parameters
func (c *Config) GetDBConfig() *database.DBConfig {
	return &database.DBConfig{
		Type:             c.DBType,
		ConnectionString: c.DBConnectionString,
		Path:             c.DatabasePath,
		Host:             c.DBHost,
		Port:             c.GetEffectiveDBPort(), // Use effective port with defaults
		Database:         c.DBName,
		Username:         c.DBUser,
		Password:         c.DBPassword,
		SSLMode:          c.DBSSLMode,
		MaxOpenConns:     c.DBMaxOpenConns,
		MaxIdleConns:     c.DBMaxIdleConns,
		ConnMaxLifetime:  c.DBConnMaxLifetime,
	}
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := strings.ToLower(os.Getenv(key))
	if valueStr == "" {
		return defaultValue
	}
	return valueStr == "true" || valueStr == "1" || valueStr == "yes"
}

// getEnvRequired returns the environment variable or a default with a warning
// In production, this should be set explicitly - using default logs a warning
func getEnvRequired(key, insecureDefault string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	// Log warning but don't crash - allows development without explicit config
	// The "INSECURE" suffix makes it clear this is not safe for production
	return insecureDefault
}

// IsLocalDevelopment returns true if running in local development mode
// Detected by checking if BaseDomain ends with .local or .localhost
func (c *Config) IsLocalDevelopment() bool {
	if c.BaseDomain == "" {
		return false
	}
	return strings.HasSuffix(c.BaseDomain, ".local") ||
		strings.HasSuffix(c.BaseDomain, ".localhost") ||
		c.BaseDomain == "localhost"
}

// IsBillingTestModeEnabled returns true if billing test mode is enabled
// This allows testing subscription upgrades without Stripe
// Auto-enabled for local development (.local, .localhost domains) or via BILLING_TEST_MODE=true
func (c *Config) IsBillingTestModeEnabled() bool {
	return c.BillingTestMode || c.IsLocalDevelopment()
}

// Validate validates the configuration and returns an error if invalid
// This should be called after Load() to ensure all critical settings are correct
func (c *Config) Validate() error {
	var errors []string

	// CRITICAL: JWT Secret validation - reject insecure defaults in production
	if !c.IsLocalDevelopment() && c.BaseDomain != "" {
		// Production mode (SaaS) - strict validation
		if c.JWTSecret == "change-this-in-production-INSECURE" ||
			strings.Contains(strings.ToLower(c.JWTSecret), "insecure") ||
			len(c.JWTSecret) < 32 {
			errors = append(errors, "JWT_SECRET must be at least 32 characters and not contain 'insecure' in production")
		}
	} else if c.JWTSecret == "change-this-in-production-INSECURE" {
		// Simple mode - warn but don't fail
		fmt.Println("WARNING: Using insecure JWT secret. Set JWT_SECRET environment variable for production.")
	}

	// CRITICAL: Validate non-negative integer values
	if c.JWTExpirationHours <= 0 {
		errors = append(errors, "JWT_EXPIRATION_HOURS must be a positive integer")
	}
	if c.BookingAdvanceDays < 0 {
		errors = append(errors, "BOOKING_ADVANCE_DAYS cannot be negative")
	}
	if c.CancellationNoticeHours < 0 {
		errors = append(errors, "CANCELLATION_NOTICE_HOURS cannot be negative")
	}
	if c.AutoDeactivationDays < 0 {
		errors = append(errors, "AUTO_DEACTIVATION_DAYS cannot be negative")
	}
	if c.MaxUploadSizeMB <= 0 {
		errors = append(errors, "MAX_UPLOAD_SIZE_MB must be a positive integer")
	}
	if c.DBMaxOpenConns < 0 {
		errors = append(errors, "DB_MAX_OPEN_CONNS cannot be negative")
	}
	if c.DBMaxIdleConns < 0 {
		errors = append(errors, "DB_MAX_IDLE_CONNS cannot be negative")
	}
	if c.DBConnMaxLifetime < 0 {
		errors = append(errors, "DB_CONN_MAX_LIFETIME cannot be negative")
	}

	// HIGH: Validate PORT is numeric
	if c.Port != "" {
		if port, err := strconv.Atoi(c.Port); err != nil {
			errors = append(errors, fmt.Sprintf("PORT must be a valid number, got: %s", c.Port))
		} else if port < 1 || port > 65535 {
			errors = append(errors, fmt.Sprintf("PORT must be between 1 and 65535, got: %d", port))
		}
	}

	// HIGH: Validate DB port when using MySQL/PostgreSQL
	if c.DBType == "mysql" || c.DBType == "postgres" {
		if c.DBPort < 0 {
			errors = append(errors, "DB_PORT cannot be negative")
		}
		// Note: DBPort = 0 is allowed as it means "use default port"
	}

	// HIGH: Validate SMTP configuration
	if c.EmailProvider == "smtp" && c.SMTPHost != "" {
		if c.SMTPPort <= 0 || c.SMTPPort > 65535 {
			errors = append(errors, "SMTP_PORT must be a valid port number (1-65535)")
		}
		// Warn if neither TLS nor SSL is enabled (insecure)
		if !c.SMTPUseTLS && !c.SMTPUseSSL {
			fmt.Println("WARNING: Neither SMTP_USE_TLS nor SMTP_USE_SSL is enabled. Email transmission will be unencrypted.")
		}
	}

	// HIGH: Validate S3 configuration when enabled
	if c.UseS3 {
		if c.S3Endpoint == "" {
			errors = append(errors, "S3_ENDPOINT is required when USE_S3 is enabled")
		}
		if c.S3AccessKey == "" {
			errors = append(errors, "S3_ACCESS_KEY is required when USE_S3 is enabled")
		}
		if c.S3SecretKey == "" {
			errors = append(errors, "S3_SECRET_KEY is required when USE_S3 is enabled")
		}
		if c.S3BucketName == "" {
			errors = append(errors, "S3_BUCKET_NAME is required when USE_S3 is enabled")
		}
	}

	// Validate DBType
	validDBTypes := map[string]bool{"sqlite": true, "mysql": true, "postgres": true}
	if !validDBTypes[c.DBType] {
		errors = append(errors, fmt.Sprintf("DB_TYPE must be 'sqlite', 'mysql', or 'postgres', got: %s", c.DBType))
	}

	// Validate EmailProvider
	validEmailProviders := map[string]bool{"gmail": true, "smtp": true}
	if !validEmailProviders[c.EmailProvider] {
		errors = append(errors, fmt.Sprintf("EMAIL_PROVIDER must be 'gmail' or 'smtp', got: %s", c.EmailProvider))
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n- %s", strings.Join(errors, "\n- "))
	}

	return nil
}

// GetDefaultDBPort returns the default port for the configured database type
func (c *Config) GetDefaultDBPort() int {
	switch c.DBType {
	case "mysql":
		return 3306
	case "postgres":
		return 5432
	default:
		return 0 // SQLite doesn't use a port
	}
}

// GetEffectiveDBPort returns the configured port or default if not set
func (c *Config) GetEffectiveDBPort() int {
	if c.DBPort > 0 {
		return c.DBPort
	}
	return c.GetDefaultDBPort()
}

// DONE
