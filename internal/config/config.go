package config

import (
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

		// JWT
		JWTSecret:          getEnv("JWT_SECRET", "change-this-in-production"),
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
		Port:             c.DBPort,
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

// DONE
