# Bug Report: internal/config

**Analysis Date:** 2025-12-27
**Directory Analyzed:** `/home/tranmh/work/gassigeher-saas/internal/config`
**Files Analyzed:** 2 files (config.go, config_test.go)
**Bugs Found:** 11 bugs

---

## Summary

The configuration module has **11 critical security and validation bugs** affecting authentication security, email delivery, database connections, and data integrity. The most severe issues include:

- **Critical**: Insecure JWT secret allows production deployments with weak authentication
- **Critical**: Missing validation for negative/zero configuration values enables system abuse
- **High**: SMTP/S3 SSL/TLS configuration conflicts silently ignored, breaking email delivery
- **High**: Missing PORT validation allows non-numeric ports to crash server
- **High**: Database port defaults to 0, causing connection failures

These bugs can lead to authentication bypass, email delivery failures, database connection issues, and denial of service attacks through configuration manipulation.

---

## Bugs

## Bug #1: Insecure JWT Secret Allowed in Production

**Severity:** CRITICAL

**Description:**
The `getEnvRequired()` function returns a hardcoded insecure default JWT secret (`"change-this-in-production-INSECURE"`) when the `JWT_SECRET` environment variable is not set. While the function name suggests it's "required," it actually provides a fallback that allows production deployments with a publicly-known secret. This completely breaks JWT authentication security, as attackers can forge valid JWT tokens using this known secret to impersonate any user, including administrators.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/config/config.go`
- Function: `getEnvRequired`
- Lines: 261-268
- Usage: Line 137

**Steps to Reproduce:**
1. Deploy application without setting `JWT_SECRET` environment variable
2. Application starts successfully with default secret
3. Attacker uses known secret `"change-this-in-production-INSECURE"` to forge JWT tokens
4. Expected: Application refuses to start without valid JWT_SECRET
5. Actual: Application runs with publicly-known secret

**Impact:**
- Complete authentication bypass
- Attacker can impersonate any user including super admin
- All JWT tokens can be forged
- No indication to administrators that security is compromised

**Fix:**
Replace the insecure default with a startup validation that fails if JWT_SECRET is not set or is the insecure default:

```diff
func getEnvRequired(key, insecureDefault string) string {
	if value := os.Getenv(key); value != "" {
+		// Reject known insecure values
+		if value == "change-this-in-production-INSECURE" ||
+		   value == "change-this-to-a-random-secret-in-production" {
+			log.Fatalf("SECURITY ERROR: %s is set to an insecure default value. Generate a secure secret with: openssl rand -base64 32", key)
+		}
		return value
	}
-	// Log warning but don't crash - allows development without explicit config
-	// The "INSECURE" suffix makes it clear this is not safe for production
-	return insecureDefault
+	// Fail fast - required values must be explicitly set
+	log.Fatalf("SECURITY ERROR: %s must be set in production. Generate with: openssl rand -base64 32", key)
+	return "" // unreachable
}
```

Add validation in `Load()` function:

```diff
func Load() *Config {
-	return &Config{
+	cfg := &Config{
		// ... all existing fields ...
	}
+
+	// Validate critical security settings at startup
+	if cfg.JWTSecret == "" ||
+	   cfg.JWTSecret == "change-this-in-production-INSECURE" ||
+	   cfg.JWTSecret == "change-this-to-a-random-secret-in-production" {
+		log.Fatal("SECURITY ERROR: JWT_SECRET must be set to a secure random value. Generate with: openssl rand -base64 32")
+	}
+
+	if len(cfg.JWTSecret) < 32 {
+		log.Printf("WARNING: JWT_SECRET should be at least 32 characters (256 bits). Current length: %d", len(cfg.JWTSecret))
+	}
+
+	return cfg
}
```

---

## Bug #2: Missing Validation for Negative Integer Configuration Values

**Severity:** CRITICAL

**Description:**
The `getEnvAsInt()` function parses integer environment variables without validating that they are positive for settings that should never be negative. This allows configuration of negative values for `JWT_EXPIRATION_HOURS`, `MAX_UPLOAD_SIZE_MB`, `BOOKING_ADVANCE_DAYS`, `CANCELLATION_NOTICE_HOURS`, `AUTO_DEACTIVATION_DAYS`, and database connection pool settings. Negative values can cause:
- JWT tokens that expire immediately or in the past (security issue)
- Negative file size limits that bypass upload validation
- Negative booking windows that allow past bookings
- Negative connection pool sizes that crash database connections

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/config/config.go`
- Function: `getEnvAsInt`
- Lines: 243-249
- Affected settings: Lines 132-171

**Steps to Reproduce:**
1. Set environment variable: `JWT_EXPIRATION_HOURS=-1`
2. Application loads config successfully
3. JWT tokens generated with negative expiration
4. Expected: Configuration validation error at startup
5. Actual: Negative value accepted, causing unpredictable behavior

**Impact:**
- JWT tokens that expire immediately (authentication fails)
- Upload size limits bypassed with negative values
- Database connection pool crashes with negative max connections
- Business logic violations (negative booking advance days)

**Fix:**
Add validation for positive-only integer settings:

```diff
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

+// getEnvAsPositiveInt returns a positive integer from environment or default
+// Logs fatal error if value is negative (for settings that must be positive)
+func getEnvAsPositiveInt(key string, defaultValue int, allowZero bool) int {
+	valueStr := os.Getenv(key)
+	if valueStr == "" {
+		return defaultValue
+	}
+
+	value, err := strconv.Atoi(valueStr)
+	if err != nil {
+		log.Printf("Warning: Invalid integer for %s: %s, using default: %d", key, valueStr, defaultValue)
+		return defaultValue
+	}
+
+	minValue := 1
+	if allowZero {
+		minValue = 0
+	}
+
+	if value < minValue {
+		log.Fatalf("Configuration error: %s must be >= %d, got: %d", key, minValue, value)
+	}
+
+	return value
+}
```

Update critical settings to use the new function:

```diff
	// JWT (SECURITY: JWT_SECRET must be explicitly set in production)
	JWTSecret:          getEnvRequired("JWT_SECRET", "change-this-in-production-INSECURE"),
-	JWTExpirationHours: getEnvAsInt("JWT_EXPIRATION_HOURS", 24),
+	JWTExpirationHours: getEnvAsPositiveInt("JWT_EXPIRATION_HOURS", 24, false),

	// Uploads
	UploadDir:       getEnv("UPLOAD_DIR", "./uploads"),
-	MaxUploadSizeMB: getEnvAsInt("MAX_UPLOAD_SIZE_MB", 5),
+	MaxUploadSizeMB: getEnvAsPositiveInt("MAX_UPLOAD_SIZE_MB", 5, false),

	// System Settings
-	BookingAdvanceDays:      getEnvAsInt("BOOKING_ADVANCE_DAYS", 14),
-	CancellationNoticeHours: getEnvAsInt("CANCELLATION_NOTICE_HOURS", 12),
-	AutoDeactivationDays:    getEnvAsInt("AUTO_DEACTIVATION_DAYS", 365),
+	BookingAdvanceDays:      getEnvAsPositiveInt("BOOKING_ADVANCE_DAYS", 14, false),
+	CancellationNoticeHours: getEnvAsPositiveInt("CANCELLATION_NOTICE_HOURS", 12, false),
+	AutoDeactivationDays:    getEnvAsPositiveInt("AUTO_DEACTIVATION_DAYS", 365, false),

	// Connection Pool Configuration (MySQL/PostgreSQL)
-	DBMaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
-	DBMaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
-	DBConnMaxLifetime: getEnvAsInt("DB_CONN_MAX_LIFETIME", 5),
+	DBMaxOpenConns:    getEnvAsPositiveInt("DB_MAX_OPEN_CONNS", 25, false),
+	DBMaxIdleConns:    getEnvAsPositiveInt("DB_MAX_IDLE_CONNS", 5, false),
+	DBConnMaxLifetime: getEnvAsPositiveInt("DB_CONN_MAX_LIFETIME", 5, false),
```

---

## Bug #3: PORT Configuration Not Validated as Numeric

**Severity:** HIGH

**Description:**
The `Port` configuration field is stored as a string without validation that it contains a valid numeric port. Non-numeric values like `"abc"`, empty strings, or invalid port numbers (0, >65535) are accepted during config loading but will cause the HTTP server to fail at startup with a cryptic error. This makes debugging difficult as the error occurs later in the startup sequence rather than during configuration validation.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/config/config.go`
- Field: `Port string`
- Line: 73, 174

**Steps to Reproduce:**
1. Set environment variable: `PORT=invalid`
2. Application loads config successfully
3. Server startup fails with error: `listen tcp: address invalid: invalid port`
4. Expected: Clear validation error during config load
5. Actual: Cryptic error during server startup

**Impact:**
- Server fails to start with unclear error messages
- Difficult to debug in production deployments
- Invalid ports accepted by configuration system

**Fix:**
Add port validation in the `Load()` function:

```diff
func Load() *Config {
	cfg := &Config{
		// ... all existing fields ...
		Port:    getEnv("PORT", "8080"),
		// ... rest of fields ...
	}
+
+	// Validate PORT is a valid number in range 1-65535
+	portNum, err := strconv.Atoi(cfg.Port)
+	if err != nil {
+		log.Fatalf("Configuration error: PORT must be a valid number, got: %s", cfg.Port)
+	}
+	if portNum < 1 || portNum > 65535 {
+		log.Fatalf("Configuration error: PORT must be between 1-65535, got: %d", portNum)
+	}
+
+	// Warn about privileged ports (< 1024) on Unix systems
+	if portNum < 1024 {
+		log.Printf("WARNING: PORT %d is a privileged port. Requires root privileges on Unix systems.", portNum)
+	}

	return cfg
}
```

---

## Bug #4: Database Port Defaults to 0, Causing Connection Failures

**Severity:** HIGH

**Description:**
The `DBPort` configuration defaults to 0 with a comment "0 means use default" but this default is never actually applied. When users configure MySQL or PostgreSQL without explicitly setting `DB_PORT`, the connection string is built with port 0, which is invalid and causes database connection failures. The database connection code must handle the 0 value and substitute the correct default (3306 for MySQL, 5432 for PostgreSQL), but relying on this is fragile and leads to confusing errors.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/config/config.go`
- Field: `DBPort int`
- Line: 21, 124

**Steps to Reproduce:**
1. Set DB_TYPE=mysql without setting DB_PORT
2. DBPort defaults to 0
3. Database connection attempts to connect to port 0
4. Connection fails: "dial tcp: invalid port 0"
5. Expected: Automatic default to 3306 for MySQL
6. Actual: Connection fails with port 0

**Impact:**
- Database connections fail with unclear errors
- Users must explicitly set DB_PORT even though defaults exist
- Poor user experience for standard configurations

**Fix:**
Set correct default ports based on database type:

```diff
func Load() *Config {
-	return &Config{
+	cfg := &Config{
		// Database Type (default: sqlite)
		DBType: getEnv("DB_TYPE", "sqlite"),

		// SQLite Configuration
		DatabasePath: getEnv("DATABASE_PATH", "./gassigeher.db"),

		// MySQL/PostgreSQL Configuration
		DBHost:             getEnv("DB_HOST", "localhost"),
-		DBPort:             getEnvAsInt("DB_PORT", 0), // 0 means use default (3306 for MySQL, 5432 for PostgreSQL)
+		DBPort:             getEnvAsInt("DB_PORT", 0), // Will be set below based on DB_TYPE
		DBName:             getEnv("DB_NAME", "gassigeher"),
		// ... rest of config ...
	}
+
+	// Set default database port based on type if not explicitly configured
+	if cfg.DBPort == 0 {
+		switch cfg.DBType {
+		case "mysql":
+			cfg.DBPort = 3306
+		case "postgres":
+			cfg.DBPort = 5432
+		// sqlite doesn't use a port
+		}
+	}
+
+	// Validate database port if set
+	if cfg.DBPort > 0 && (cfg.DBPort < 1 || cfg.DBPort > 65535) {
+		log.Fatalf("Configuration error: DB_PORT must be between 1-65535, got: %d", cfg.DBPort)
+	}
+
+	return cfg
}
```

---

## Bug #5: SMTP_USE_TLS and SMTP_USE_SSL Can Both Be False

**Severity:** HIGH

**Description:**
The SMTP configuration allows both `SMTPUseTLS` and `SMTPUseSSL` to be false simultaneously, which means emails will be sent over an unencrypted connection. While the `email_provider_factory.go` validates that both cannot be true, it doesn't validate that at least one must be true for production use. Unencrypted SMTP connections expose email contents, credentials, and authentication tokens in plaintext over the network, violating security best practices and compliance requirements (GDPR, etc.).

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/config/config.go`
- Fields: `SMTPUseTLS`, `SMTPUseSSL`
- Lines: 57-59, 158-159

**Steps to Reproduce:**
1. Set EMAIL_PROVIDER=smtp
2. Set SMTP_USE_TLS=false
3. Set SMTP_USE_SSL=false
4. Application starts successfully
5. Emails sent over unencrypted connection
6. Expected: Warning or error about unencrypted email
7. Actual: Silently uses unencrypted connection

**Impact:**
- Email contents exposed in plaintext (PII violations)
- SMTP credentials transmitted unencrypted
- Password reset tokens visible to network sniffers
- Violates GDPR/compliance requirements

**Fix:**
Add validation in `Load()` to warn about unencrypted SMTP:

```diff
func Load() *Config {
	cfg := &Config{
		// ... all fields ...
	}
+
+	// Validate SMTP encryption configuration
+	if cfg.EmailProvider == "smtp" {
+		if !cfg.SMTPUseTLS && !cfg.SMTPUseSSL {
+			log.Printf("SECURITY WARNING: SMTP email is configured without encryption (neither TLS nor SSL enabled)")
+			log.Printf("Email contents, credentials, and tokens will be transmitted in plaintext")
+			log.Printf("Set SMTP_USE_TLS=true (port 587) or SMTP_USE_SSL=true (port 465)")
+		}
+	}

	return cfg
}
```

---

## Bug #6: S3_USE_SSL Defaults to True but S3_ENDPOINT Not Validated

**Severity:** HIGH

**Description:**
The `S3UseSSL` configuration defaults to `true` for security, but there's no validation that the `S3Endpoint` actually supports HTTPS. If a user configures an HTTP-only S3 endpoint (e.g., `http://localhost:9000` for MinIO development), file uploads will fail with SSL errors. Conversely, if S3UseSSL is set to false but the endpoint requires HTTPS, uploads will also fail. The endpoint and SSL setting should be validated for consistency.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/config/config.go`
- Fields: `S3Endpoint`, `S3UseSSL`
- Lines: 82, 88, 183, 189

**Steps to Reproduce:**
1. Set USE_S3=true
2. Set S3_ENDPOINT=http://localhost:9000 (MinIO development)
3. S3UseSSL defaults to true
4. File upload attempts HTTPS connection to HTTP endpoint
5. Upload fails with SSL protocol error
6. Expected: Warning about HTTP endpoint with SSL enabled
7. Actual: Silent failure during upload

**Impact:**
- File uploads fail in production
- Tenant logos/assets cannot be uploaded
- Unclear error messages for S3 configuration issues

**Fix:**
Add S3 endpoint validation in `Load()`:

```diff
func Load() *Config {
	cfg := &Config{
		// ... all fields ...
	}
+
+	// Validate S3 configuration if enabled
+	if cfg.UseS3 {
+		if cfg.S3Endpoint == "" {
+			log.Fatal("Configuration error: S3_ENDPOINT is required when USE_S3=true")
+		}
+
+		// Check endpoint scheme matches SSL setting
+		if strings.HasPrefix(cfg.S3Endpoint, "http://") && cfg.S3UseSSL {
+			log.Printf("WARNING: S3_ENDPOINT uses http:// but S3_USE_SSL=true")
+			log.Printf("This may cause connection failures. Set S3_USE_SSL=false for HTTP endpoints")
+		}
+
+		if strings.HasPrefix(cfg.S3Endpoint, "https://") && !cfg.S3UseSSL {
+			log.Printf("WARNING: S3_ENDPOINT uses https:// but S3_USE_SSL=false")
+			log.Printf("This may cause connection failures. Set S3_USE_SSL=true for HTTPS endpoints")
+		}
+
+		if cfg.S3AccessKey == "" || cfg.S3SecretKey == "" {
+			log.Fatal("Configuration error: S3_ACCESS_KEY and S3_SECRET_KEY are required when USE_S3=true")
+		}
+
+		if cfg.S3BucketName == "" {
+			log.Fatal("Configuration error: S3_BUCKET_NAME is required when USE_S3=true")
+		}
+	}

	return cfg
}
```

---

## Bug #7: Missing Validation for BASE_URL Format

**Severity:** MEDIUM

**Description:**
The `BASE_URL` configuration is critical for generating correct email links (password reset, verification), but there's no validation that it's a valid URL format. Invalid BASE_URL values (missing protocol, trailing slash inconsistency, typos) will generate broken links in emails, preventing users from resetting passwords or verifying accounts. This is especially problematic in production where email links are the primary recovery mechanism.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/config/config.go`
- Field: `BaseURL`
- Line: 74, 175

**Steps to Reproduce:**
1. Set BASE_URL=localhost:8080 (missing protocol)
2. Application loads successfully
3. Password reset email sent with link: `localhost:8080/reset-password?token=...`
4. Browser treats it as relative path, link is broken
5. Expected: Configuration error about missing protocol
6. Actual: Broken links generated in emails

**Impact:**
- Users cannot reset passwords
- Account verification fails
- All email links broken
- Support burden from users unable to access system

**Fix:**
Add BASE_URL validation:

```diff
+import (
+	"net/url"
+	// ... existing imports ...
+)

func Load() *Config {
	cfg := &Config{
		// ... all fields ...
	}
+
+	// Validate BASE_URL format
+	if cfg.BaseURL != "" {
+		parsedURL, err := url.Parse(cfg.BaseURL)
+		if err != nil {
+			log.Fatalf("Configuration error: BASE_URL is not a valid URL: %v", err)
+		}
+
+		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
+			log.Fatalf("Configuration error: BASE_URL must start with http:// or https://, got: %s", cfg.BaseURL)
+		}
+
+		if parsedURL.Host == "" {
+			log.Fatalf("Configuration error: BASE_URL must include a hostname, got: %s", cfg.BaseURL)
+		}
+
+		// Warn about production using HTTP
+		if parsedURL.Scheme == "http" && !strings.Contains(parsedURL.Host, "localhost") &&
+		   !strings.Contains(parsedURL.Host, "127.0.0.1") {
+			log.Printf("WARNING: BASE_URL uses http:// for non-localhost domain: %s", cfg.BaseURL)
+			log.Printf("Production deployments should use https:// for security")
+		}
+
+		// Normalize: remove trailing slash for consistency
+		cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
+	}

	return cfg
}
```

---

## Bug #8: Missing Validation for Stripe Keys in SaaS Mode

**Severity:** MEDIUM

**Description:**
When `SaaSMode` is enabled (or `BASE_DOMAIN` is set), the application requires Stripe configuration for billing, but there's no validation that Stripe keys are configured. The application will start successfully but billing features will fail at runtime when tenants try to subscribe. The `.env.example` shows test keys (`sk_test_...`, `pk_test_...`) which may be copied to production, causing live billing to fail.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/config/config.go`
- Fields: `StripeSecretKey`, `StripePublishableKey`, etc.
- Lines: 91-96, 192-196

**Steps to Reproduce:**
1. Set BASE_DOMAIN=gassigeher.org (enables SaaS mode)
2. Don't set Stripe environment variables
3. Application starts successfully
4. Tenant tries to subscribe
5. Billing fails with "Stripe not configured"
6. Expected: Startup warning about missing Stripe config in SaaS mode
7. Actual: Runtime failure when billing is accessed

**Impact:**
- Billing broken in production
- Revenue loss from failed subscriptions
- Test keys used in production (payment processing fails)
- Poor user experience during subscription

**Fix:**
Add Stripe validation for SaaS mode:

```diff
func Load() *Config {
	cfg := &Config{
		// ... all fields ...
	}
+
+	// Validate Stripe configuration in SaaS mode (unless test mode)
+	if cfg.SaaSMode && cfg.BaseDomain != "" && !cfg.IsBillingTestModeEnabled() {
+		if cfg.StripeSecretKey == "" {
+			log.Printf("WARNING: SaaS mode enabled but STRIPE_SECRET_KEY not set. Billing features will fail.")
+			log.Printf("Set BILLING_TEST_MODE=true for development or configure Stripe for production")
+		}
+
+		// Warn if using test keys in production
+		if cfg.StripeSecretKey != "" && strings.HasPrefix(cfg.StripeSecretKey, "sk_test_") {
+			if !cfg.IsLocalDevelopment() {
+				log.Printf("WARNING: Using Stripe TEST key in production environment")
+				log.Printf("Replace with live key (sk_live_...) for actual payment processing")
+			}
+		}
+
+		// Warn if mixing test and live keys
+		if strings.HasPrefix(cfg.StripeSecretKey, "sk_test_") &&
+		   strings.HasPrefix(cfg.StripePublishableKey, "pk_live_") {
+			log.Fatal("Configuration error: Cannot mix Stripe test secret key with live publishable key")
+		}
+		if strings.HasPrefix(cfg.StripeSecretKey, "sk_live_") &&
+		   strings.HasPrefix(cfg.StripePublishableKey, "pk_test_") {
+			log.Fatal("Configuration error: Cannot mix Stripe live secret key with test publishable key")
+		}
+	}

	return cfg
}
```

---

## Bug #9: DB_TYPE Not Validated Against Supported Values

**Severity:** MEDIUM

**Description:**
The `DBType` configuration accepts any string value without validation. While the database initialization code will fail with an unsupported type, the error occurs later in the startup sequence. Invalid database types like typos (`"postgress"`, `"mysqll"`) or unsupported databases (`"mongodb"`, `"redis"`) are accepted by config loading, causing confusing errors during database initialization.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/config/config.go`
- Field: `DBType`
- Line: 14, 117

**Steps to Reproduce:**
1. Set DB_TYPE=postgress (typo)
2. Config loads successfully
3. Database initialization fails with "unsupported database type: postgress"
4. Expected: Clear config validation error with list of supported types
5. Actual: Error during database initialization

**Impact:**
- Confusing error messages for typos
- Application starts but database doesn't initialize
- Difficult to debug in container deployments

**Fix:**
Add DB_TYPE validation:

```diff
func Load() *Config {
	cfg := &Config{
		// Database Type (default: sqlite)
		DBType: getEnv("DB_TYPE", "sqlite"),
		// ... rest of fields ...
	}
+
+	// Validate database type
+	validDBTypes := []string{"sqlite", "mysql", "postgres"}
+	dbTypeValid := false
+	for _, valid := range validDBTypes {
+		if cfg.DBType == valid {
+			dbTypeValid = true
+			break
+		}
+	}
+
+	if !dbTypeValid {
+		log.Fatalf("Configuration error: DB_TYPE must be one of: %v, got: %s", validDBTypes, cfg.DBType)
+	}
+
+	// Validate PostgreSQL is used for SaaS mode (required for RLS)
+	if cfg.SaaSMode && cfg.BaseDomain != "" && cfg.DBType != "postgres" {
+		log.Fatal("Configuration error: SaaS mode requires DB_TYPE=postgres (for Row-Level Security)")
+	}

	return cfg
}
```

---

## Bug #10: Integer Overflow Not Checked in getEnvAsInt

**Severity:** LOW

**Description:**
The `getEnvAsInt()` function uses `strconv.Atoi()` which parses into Go's `int` type (platform-dependent: 32-bit on 32-bit systems, 64-bit on 64-bit systems). On 32-bit systems, extremely large values will overflow and wrap around to negative numbers without any error. For example, `MAX_UPLOAD_SIZE_MB=2147483648` would overflow to -2147483648 on 32-bit systems, bypassing file size limits.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/config/config.go`
- Function: `getEnvAsInt`
- Lines: 243-249

**Steps to Reproduce:**
1. On 32-bit system, set MAX_UPLOAD_SIZE_MB=2147483648
2. Value overflows to negative number
3. File size validation uses negative limit
4. Expected: Overflow error or warning
5. Actual: Silent overflow to negative value

**Impact:**
- File size limits can be bypassed on 32-bit systems
- Connection pool settings can overflow
- Unpredictable behavior from negative overflows

**Fix:**
Add bounds checking for integer parsing:

```diff
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
+	if valueStr == "" {
+		return defaultValue
+	}
+
-	if value, err := strconv.Atoi(valueStr); err == nil {
+	// Use ParseInt with explicit 32-bit range to detect overflows
+	value64, err := strconv.ParseInt(valueStr, 10, 32)
+	if err != nil {
+		log.Printf("Warning: Invalid integer for %s: %s (using default: %d)", key, valueStr, defaultValue)
		return defaultValue
+	}
+
+	value := int(value64)
+
+	// Warn about suspiciously large values (likely configuration errors)
+	if value > 1000000 { // 1 million
+		log.Printf("WARNING: %s has unusually large value: %d", key, value)
	}
+
	return defaultValue
}
```

---

## Bug #11: Missing Validation for DBMaxIdleConns vs DBMaxOpenConns Relationship

**Severity:** LOW

**Description:**
The database connection pool settings `DBMaxIdleConns` and `DBMaxOpenConns` are validated independently but there's no check that `DBMaxIdleConns <= DBMaxOpenConns`. According to database/sql documentation, idle connections cannot exceed max open connections. If misconfigured (e.g., `DB_MAX_IDLE_CONNS=50` and `DB_MAX_OPEN_CONNS=25`), the database driver will silently cap idle connections at max open, which may not match the administrator's intent and can cause performance issues.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/config/config.go`
- Fields: `DBMaxOpenConns`, `DBMaxIdleConns`
- Lines: 31-32, 132-133

**Steps to Reproduce:**
1. Set DB_MAX_OPEN_CONNS=25
2. Set DB_MAX_IDLE_CONNS=50 (higher than max open)
3. Config loads successfully
4. Database driver caps idle at 25
5. Expected: Warning about misconfiguration
6. Actual: Silent capping with potential performance issues

**Impact:**
- Unexpected connection pool behavior
- Performance issues from misconfigured pools
- Difficult to debug connection exhaustion

**Fix:**
Add relationship validation:

```diff
func Load() *Config {
	cfg := &Config{
		// ... all fields ...
	}
+
+	// Validate database connection pool settings relationship
+	if cfg.DBType == "mysql" || cfg.DBType == "postgres" {
+		if cfg.DBMaxIdleConns > cfg.DBMaxOpenConns {
+			log.Printf("WARNING: DB_MAX_IDLE_CONNS (%d) is greater than DB_MAX_OPEN_CONNS (%d)",
+				cfg.DBMaxIdleConns, cfg.DBMaxOpenConns)
+			log.Printf("Idle connections will be automatically capped at max open connections")
+			log.Printf("Consider setting DB_MAX_IDLE_CONNS <= DB_MAX_OPEN_CONNS for clarity")
+		}
+
+		// Warn about connection pool configuration that may cause issues
+		if cfg.DBMaxOpenConns > 100 {
+			log.Printf("WARNING: DB_MAX_OPEN_CONNS is very high (%d). Ensure database can handle this many connections",
+				cfg.DBMaxOpenConns)
+		}
+	}

	return cfg
}
```

---

## Statistics

- **Critical:** 2 bugs (JWT security, negative values)
- **High:** 4 bugs (PORT validation, DB port defaults, SMTP encryption, S3 SSL)
- **Medium:** 3 bugs (BASE_URL format, Stripe config, DB_TYPE validation)
- **Low:** 2 bugs (integer overflow, connection pool relationship)

---

## Recommendations

### Immediate Actions (Critical/High Priority)

1. **Add comprehensive configuration validation** - Create a `Validate()` method called immediately after `Load()` that checks all critical settings
2. **Fail fast on security issues** - Invalid security configurations (weak JWT secrets, unencrypted SMTP) should prevent server startup
3. **Implement getEnvAsPositiveInt()** - Replace all uses of getEnvAsInt() for positive-only settings
4. **Add startup validation log** - Print a summary of critical configuration values on startup for audit/debugging

### Long-term Improvements

1. **Configuration schema validation** - Consider using a schema validation library (e.g., go-playground/validator)
2. **Environment-specific validation** - Production deployments should have stricter validation than development
3. **Configuration test coverage** - Add tests for invalid configurations to ensure validation works
4. **Documentation** - Document valid ranges and relationships for all config values in `.env.example`

### Security Hardening

1. **Mandatory security settings** - JWT_SECRET, SUPER_ADMIN_EMAIL should fail if not set (not use defaults)
2. **Encryption by default** - SMTP and S3 should require explicit opt-out of encryption
3. **Production detection** - Auto-detect production environment (non-localhost BASE_URL) and enforce stricter validation
4. **Secrets scanning** - Add startup check to warn about known insecure values from .env.example

### Code Quality

1. **Consistent error handling** - Use consistent pattern for config validation errors (all fatal vs all warnings)
2. **Helper function consolidation** - Create a single validation framework instead of ad-hoc checks
3. **Type safety** - Consider using typed config with constraints (e.g., positive integers as separate type)
4. **Unit test coverage** - Add negative test cases for all validation scenarios
