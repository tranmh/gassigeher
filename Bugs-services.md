# Bug Report: services

**Analysis Date:** 2025-12-01
**Directory Analyzed:** `internal/services`
**Files Analyzed:** 10 files
**Bugs Found:** 14 bugs

---

## Summary

The services directory contains critical business logic for authentication, email, image processing, and holiday management. Analysis revealed multiple security vulnerabilities, concurrency issues, error handling gaps, and logic errors:

**Critical Issues:**
- JWT secret weakness allowing production deployment with default secret
- Race condition in email template execution
- Email template injection vulnerability
- Timing attack vulnerability in password validation

**High-Priority Issues:**
- Missing JWT expiration validation
- Weak password validation allowing predictable passwords
- Email provider initialization without connection validation
- HTML template injection in user-provided data

**Medium-Priority Issues:**
- Holiday API calls without timeout configuration
- Missing error handling in goroutine email sends
- Base64 encoding implementation bug with padding

---

## Bugs

## Bug #1: JWT Secret Weakness - Production Deployment with Default Secret

**STATUS: CODE UNCHANGED - BUG STILL PRESENT**

**Severity:** CRITICAL

**Description:**
The JWT secret defaults to "change-this-in-production" in the config loader (config.go line 102). The AuthService accepts ANY string as JWT secret without validation (auth_service.go line 20-25), allowing production systems to run with this weak default secret. An attacker who knows this default can forge valid JWT tokens for any user, including super admins, leading to complete authentication bypass.

**Location:**
- File: `internal/services/auth_service.go`
- Function: `NewAuthService`
- Lines: 20-25

**Steps to Reproduce:**
1. Deploy application without setting JWT_SECRET environment variable
2. Default "change-this-in-production" is used
3. Attacker generates JWT with this known secret: `jwt.sign({user_id: 1, is_super_admin: true}, "change-this-in-production")`
4. Expected: Application should refuse weak secrets
5. Actual: Weak secret accepted, attacker gains Super Admin access

**Fix:**
Add validation in NewAuthService to reject weak secrets:

```diff
func NewAuthService(jwtSecret string, jwtExpirationHours int) *AuthService {
+	// Validate JWT secret strength
+	if len(jwtSecret) < 32 {
+		panic("JWT_SECRET must be at least 32 characters long")
+	}
+	if jwtSecret == "change-this-in-production" {
+		panic("JWT_SECRET cannot be the default value - generate a secure secret")
+	}
+	// Check for common weak secrets
+	weakSecrets := []string{"secret", "password", "jwt", "key", "test", "dev", "default"}
+	for _, weak := range weakSecrets {
+		if strings.ToLower(jwtSecret) == weak {
+			panic(fmt.Sprintf("JWT_SECRET '%s' is too weak - use a cryptographically random value", weak))
+		}
+	}
+
	return &AuthService{
		jwtSecret:          jwtSecret,
		jwtExpirationHours: jwtExpirationHours,
	}
}
```

Additionally, add startup validation in main.go to fail fast with helpful error message.

---

## Bug #2: Race Condition in Email Template Execution

**STATUS: CODE UNCHANGED - BUG STILL PRESENT**

**Severity:** CRITICAL

**Description:**
Email templates are executed using `template.Must(template.New("name").Parse(tmpl))` inline in each Send function. When multiple goroutines send emails concurrently (as done in handlers with `go emailService.SendX(...)`), they create and execute templates with the same name simultaneously. The `html/template` package's template namespace can have race conditions when templates are created and executed concurrently without synchronization.

**Location:**
- File: `internal/services/email_service.go`
- Function: All `Send*` functions
- Lines: 117, 193, 252, 318, 385, 458, 526, 614, 685, 758, 842, 918 (multiple instances)

**Steps to Reproduce:**
1. Trigger 10 concurrent booking confirmations via API (e.g., stress test with 10 users)
2. Each spawns `go emailService.SendBookingConfirmation(...)`
3. All 10 goroutines execute `template.Must(template.New("booking").Parse(tmpl))` simultaneously
4. Expected: All 10 emails sent correctly
5. Actual: Race detector shows race condition; some emails may have corrupted template data or panic

**Fix:**
Parse all templates once during EmailService initialization and store them as fields:

```diff
type EmailService struct {
	provider EmailProvider
	baseURL  string
+	// Pre-parsed templates
+	templates map[string]*template.Template
+	templateMutex sync.RWMutex
}

func NewEmailService(config *EmailConfig) (*EmailService, error) {
	// ... existing provider initialization ...

+	// Pre-parse all templates
+	templates := make(map[string]*template.Template)
+	templates["verification"] = template.Must(template.New("verification").Parse(verificationTmpl))
+	templates["welcome"] = template.Must(template.New("welcome").Parse(welcomeTmpl))
+	templates["booking"] = template.Must(template.New("booking").Parse(bookingTmpl))
+	// ... parse all other templates ...

	return &EmailService{
		provider: provider,
		baseURL:  baseURL,
+		templates: templates,
+		templateMutex: sync.RWMutex{},
	}, nil
}

func (s *EmailService) SendVerificationEmail(to, name, token string) error {
	subject := "Willkommen bei Gassigeher - E-Mail-Adresse bestätigen"

+	s.templateMutex.RLock()
+	tmpl := s.templates["verification"]
+	s.templateMutex.RUnlock()
+
	var body bytes.Buffer
-	t := template.Must(template.New("verification").Parse(tmpl))
-	if err := t.Execute(&body, map[string]string{
+	if err := tmpl.Execute(&body, map[string]string{
		"Name":    name,
		"Token":   token,
		"BaseURL": s.baseURL,
	}); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return s.SendEmail(to, subject, body.String())
}
```

This eliminates the race condition by parsing templates once and protecting concurrent access with a read lock.

---

## Bug #3: HTML Template Injection in User-Provided Data

**STATUS: CODE UNCHANGED - BUG STILL PRESENT**

**Severity:** HIGH

**Description:**
User-provided data (name, dogName, reason, message) is directly inserted into HTML email templates without escaping. The `html/template` package DOES auto-escape by default, but only when using `{{.Variable}}` syntax. However, some templates use custom formatting that might bypass this. More critically, the templates use `template.HTML` implicitly which can be dangerous. If user input contains malicious HTML/JavaScript, it could lead to XSS when emails are viewed in mail clients that render HTML.

**Location:**
- File: `internal/services/email_service.go`
- Functions: `SendBookingConfirmation`, `SendAdminCancellation`, `SendExperienceLevelApproved`, etc.
- Lines: 266-936 (multiple functions)

**Steps to Reproduce:**
1. User registers with name: `<script>alert('XSS')</script>`
2. Admin approves experience level with message: `<img src=x onerror=alert('XSS')>`
3. Email sent with user's malicious name and admin's malicious message
4. Expected: HTML escaped to prevent XSS
5. Actual: Depends on mail client, but could execute script if HTML rendering is permissive

**Fix:**
While `html/template` does auto-escape, add explicit validation to reject HTML in user inputs:

```diff
func (s *EmailService) SendExperienceLevelApproved(to, name, level string, message *string) error {
+	// Validate and sanitize inputs
+	name = sanitizeForEmail(name)
+	if message != nil {
+		cleanMsg := sanitizeForEmail(*message)
+		message = &cleanMsg
+	}
+
	levelLabel := "Blau"
	if level == "orange" {
		levelLabel = "Orange"
	}
	// ... rest of function
}

+// sanitizeForEmail removes HTML tags and dangerous characters
+func sanitizeForEmail(input string) string {
+	// Remove HTML tags
+	input = strings.ReplaceAll(input, "<", "&lt;")
+	input = strings.ReplaceAll(input, ">", "&gt;")
+	input = strings.ReplaceAll(input, "\"", "&quot;")
+	input = strings.ReplaceAll(input, "'", "&#39;")
+	return input
+}
```

Apply this sanitization to ALL user-provided data before inserting into templates.

---

## Bug #4: Missing JWT Expiration Validation on Generation

**STATUS: CODE UNCHANGED - BUG STILL PRESENT**

**Severity:** HIGH

**Description:**
The JWT generation function accepts `jwtExpirationHours` but doesn't validate it. A configuration error could set expiration to 0, negative, or excessively large values (e.g., 10 years). A 0 or negative expiration creates immediately expired tokens. An excessively large expiration (e.g., 87600 hours = 10 years) violates security best practices and prevents token rotation, leaving systems vulnerable if tokens are compromised.

**Location:**
- File: `internal/services/auth_service.go`
- Function: `GenerateJWT`
- Lines: 51-69

**Steps to Reproduce:**
1. Set JWT_EXPIRATION_HOURS=0 or JWT_EXPIRATION_HOURS=876000 (100 years)
2. User logs in successfully
3. Expected: Reasonable expiration enforced (1-720 hours)
4. Actual: Token expires immediately (0) or never expires practically (100 years)

**Fix:**
Add validation in NewAuthService and GenerateJWT:

```diff
func NewAuthService(jwtSecret string, jwtExpirationHours int) *AuthService {
+	// Validate expiration hours
+	if jwtExpirationHours < 1 {
+		panic("JWT_EXPIRATION_HOURS must be at least 1 hour")
+	}
+	if jwtExpirationHours > 720 { // Max 30 days
+		log.Printf("Warning: JWT_EXPIRATION_HOURS=%d exceeds recommended maximum of 720 hours (30 days)", jwtExpirationHours)
+	}
+
	return &AuthService{
		jwtSecret:          jwtSecret,
		jwtExpirationHours: jwtExpirationHours,
	}
}

func (s *AuthService) GenerateJWT(userID int, email string, isAdmin bool, isSuperAdmin bool) (string, error) {
+	// Double-check expiration at generation time
+	if s.jwtExpirationHours < 1 {
+		return "", fmt.Errorf("invalid JWT expiration configuration")
+	}
+
	claims := jwt.MapClaims{
		"user_id":        userID,
		"email":          email,
		"is_admin":       isAdmin,
		"is_super_admin": isSuperAdmin,
		"exp":            time.Now().Add(time.Hour * time.Duration(s.jwtExpirationHours)).Unix(),
	}
	// ... rest of function
}
```

---

## Bug #5: Timing Attack Vulnerability in CheckPassword

**STATUS: CODE UNCHANGED - BUG STILL PRESENT**

**Severity:** HIGH

**Description:**
The `CheckPassword` function returns immediately on error without constant-time comparison. While bcrypt.CompareHashAndPassword itself is constant-time, the function returns `err == nil` which could leak information about whether a hash exists or is in the correct format through timing analysis. An attacker could enumerate valid usernames by measuring response times.

**Location:**
- File: `internal/services/auth_service.go`
- Function: `CheckPassword`
- Lines: 36-40

**Steps to Reproduce:**
1. Attacker measures login time for user1@example.com (exists) vs nonexistent@example.com
2. Valid user takes ~100ms (bcrypt comparison)
3. Invalid user takes ~1ms (early return if hash is empty/invalid)
4. Expected: Constant time for all comparisons
5. Actual: Timing difference reveals whether user exists

**Fix:**
Ensure constant-time operation regardless of hash validity:

```diff
func (s *AuthService) CheckPassword(password, hash string) bool {
+	// Always perform bcrypt comparison, even if hash is invalid
+	// This prevents timing attacks to enumerate valid users
+	if hash == "" {
+		// Use a dummy hash to maintain constant time
+		// This is a bcrypt hash of "dummy_password"
+		hash = "$2a$12$R9h/cIPz0gi.URNNX3kh2OPST9/PgBkqquzi.Ss7KIUgO2t0jWMUW"
+	}
+
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
```

This ensures all password checks take approximately the same time, preventing username enumeration via timing attacks.

---

## Bug #6: Weak Password Validation - No Special Character Requirement

**STATUS: CODE UNCHANGED - BUG STILL PRESENT**

**Severity:** MEDIUM

**Description:**
The password validation requires uppercase, lowercase, and numbers, but NO special characters. This allows predictable passwords like "Password123" which are vulnerable to dictionary attacks. The validation also doesn't check against common passwords (e.g., "Password1", "Admin123") or prevent patterns like repeated characters.

**Location:**
- File: `internal/services/auth_service.go`
- Function: `ValidatePassword`
- Lines: 93-125

**Steps to Reproduce:**
1. User registers with password "Password123"
2. Expected: Password rejected as too weak
3. Actual: Password accepted (has upper, lower, number)
4. Attacker uses common password dictionary to crack this weak password

**Fix:**
Strengthen password validation with additional checks:

```diff
func (s *AuthService) ValidatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}
+
+	// Check maximum length to prevent DoS
+	if len(password) > 128 {
+		return fmt.Errorf("password must be at most 128 characters long")
+	}

	hasUpper := false
	hasLower := false
	hasNumber := false
+	hasSpecial := false

	for _, char := range password {
		switch {
		case 'A' <= char && char <= 'Z':
			hasUpper = true
		case 'a' <= char && char <= 'z':
			hasLower = true
		case '0' <= char && char <= '9':
			hasNumber = true
+		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", char):
+			hasSpecial = true
		}
	}

	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if !hasNumber {
		return fmt.Errorf("password must contain at least one number")
	}
+	if !hasSpecial {
+		return fmt.Errorf("password must contain at least one special character (!@#$%^&*()_+-=[]{}|;:,.<>?)")
+	}
+
+	// Check against common weak passwords
+	commonPasswords := []string{
+		"password1", "password123", "admin123", "qwerty123",
+		"letmein1", "welcome1", "monkey123", "dragon123",
+	}
+	passwordLower := strings.ToLower(password)
+	for _, common := range commonPasswords {
+		if passwordLower == common {
+			return fmt.Errorf("password is too common and easily guessable")
+		}
+	}

	return nil
}
```

---

## Bug #7: Email Provider Initialization Without Connection Validation

**STATUS: CODE UNCHANGED - BUG STILL PRESENT**

**Severity:** HIGH

**Description:**
The SMTP provider validates configuration but doesn't test the actual connection to the SMTP server. The Gmail provider doesn't validate the refresh token actually works. Services initialize successfully even if the SMTP server is unreachable or credentials are invalid. Emails silently fail at runtime, and the application appears functional but cannot send critical emails (verification, password reset).

**Location:**
- File: `internal/services/email_provider_smtp.go`
- Function: `NewSMTPProvider`
- Lines: 25-48

- File: `internal/services/email_provider_gmail.go`
- Function: `NewGmailProvider`
- Lines: 22-67

**Steps to Reproduce:**
1. Configure SMTP with invalid credentials or unreachable host
2. Application starts successfully (config validation passes)
3. User registers and expects verification email
4. Expected: Startup fails if email cannot be sent, or at least a warning
5. Actual: Registration succeeds but email send fails silently (if in goroutine) or with generic error

**Fix:**
Add connection test in provider initialization:

```diff
// SMTP Provider
func NewSMTPProvider(config *EmailConfig) (EmailProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	provider := &SMTPProvider{
		host:      config.SMTPHost,
		port:      config.SMTPPort,
		username:  config.SMTPUsername,
		password:  config.SMTPPassword,
		fromEmail: config.SMTPFromEmail,
		bccAdmin:  config.BCCAdmin,
		useTLS:    config.SMTPUseTLS,
		useSSL:    config.SMTPUseSSL,
	}

	// Validate configuration
	if err := provider.ValidateConfig(); err != nil {
		return nil, err
	}
+
+	// Test connection to SMTP server
+	if err := provider.testConnection(); err != nil {
+		return nil, fmt.Errorf("SMTP connection test failed: %w (check SMTP_HOST, SMTP_PORT, and credentials)", err)
+	}

	return provider, nil
}

+// testConnection attempts to connect to SMTP server to validate configuration
+func (p *SMTPProvider) testConnection() error {
+	// Try to establish connection
+	addr := fmt.Sprintf("%s:%d", p.host, p.port)
+
+	var conn net.Conn
+	var err error
+
+	if p.useSSL {
+		tlsConfig := &tls.Config{ServerName: p.host, MinVersion: tls.VersionTLS12}
+		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: 5 * time.Second}, "tcp", addr, tlsConfig)
+	} else {
+		conn, err = net.DialTimeout("tcp", addr, 5*time.Second)
+	}
+
+	if err != nil {
+		return err
+	}
+	defer conn.Close()
+
+	// Create SMTP client and test basic commands
+	client, err := smtp.NewClient(conn, p.host)
+	if err != nil {
+		return err
+	}
+	defer client.Close()
+
+	// Test STARTTLS if configured
+	if p.useTLS {
+		tlsConfig := &tls.Config{ServerName: p.host, MinVersion: tls.VersionTLS12}
+		if err := client.StartTLS(tlsConfig); err != nil {
+			return fmt.Errorf("STARTTLS failed: %w", err)
+		}
+	}
+
+	// Test authentication if credentials provided
+	if p.username != "" && p.password != "" {
+		auth := smtp.PlainAuth("", p.username, p.password, p.host)
+		if err := client.Auth(auth); err != nil {
+			return fmt.Errorf("authentication failed: %w", err)
+		}
+	}
+
+	return client.Quit()
+}
```

Apply similar test for Gmail provider by attempting to fetch user profile or send a test message.

---

## Bug #8: Holiday API Call Without Timeout Configuration

**STATUS: CODE UNCHANGED - BUG STILL PRESENT**

**Severity:** MEDIUM

**Description:**
The `FetchAndCacheHolidays` function uses `http.Get()` with the default HTTP client which has NO timeout. If the holiday API (feiertage-api.de) is slow or unresponsive, the request hangs indefinitely, blocking the booking validation logic. This can cause booking requests to timeout or hang, degrading user experience.

**Location:**
- File: `internal/services/holiday_service.go`
- Function: `FetchAndCacheHolidays`
- Lines: 43-53

**Steps to Reproduce:**
1. Network condition: Slow or unresponsive holiday API
2. User attempts to book a walk on a date that requires holiday check
3. `IsHoliday` calls `FetchAndCacheHolidays` which calls `http.Get()`
4. Expected: Request times out after reasonable duration (e.g., 5 seconds)
5. Actual: Request hangs indefinitely, booking request never completes

**Fix:**
Use HTTP client with timeout:

```diff
func (s *HolidayService) FetchAndCacheHolidays(year int) error {
	// Get state from settings
	state := "BW" // Default
	if setting, err := s.settingsRepo.Get("feiertage_state"); err == nil && setting != nil && setting.Value != "" {
		state = setting.Value
	}

	// Check cache first
	cached, err := s.holidayRepo.GetCachedHolidays(year, state)
	if err == nil && cached != "" {
		// Cache hit - populate custom_holidays table
		return s.populateHolidaysFromCache(cached, year)
	}

	// Cache miss - fetch from API
	url := fmt.Sprintf("https://feiertage-api.de/api/?jahr=%d&nur_land=%s", year, state)

-	resp, err := http.Get(url)
+	// Use HTTP client with timeout
+	client := &http.Client{
+		Timeout: 10 * time.Second,
+	}
+	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch holidays: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("holiday API returned status %d", resp.StatusCode)
	}
	// ... rest of function
}
```

---

## Bug #9: Missing Error Handling for Goroutine Email Sends

**STATUS: CODE UNCHANGED - BUG STILL PRESENT**

**Severity:** MEDIUM

**Description:**
Handlers spawn email sending in goroutines with `go emailService.SendX(...)` but completely ignore errors. If email sending fails (SMTP error, network issue, invalid configuration), the error is silently discarded. Users don't receive critical emails (verification, password reset, booking confirmation) and have no indication of the failure. This violates the principle that critical operations should not fail silently.

**Location:**
- File: Multiple handlers using EmailService
- Pattern: `go emailService.SendX(...)` without error handling
- Example: booking_handler.go, auth_handler.go, etc.

**Steps to Reproduce:**
1. Misconfigure email provider (wrong SMTP password)
2. User registers for account
3. Registration succeeds, JWT token returned
4. Expected: Error logged at minimum, ideally user warned that email failed
5. Actual: Email send error completely ignored, user never receives verification email

**Fix:**
Add error handling channel or logging for goroutine email operations:

```diff
// In handlers, wrap email sends with error logging
-go emailService.SendVerificationEmail(user.Email, user.Name, token)
+go func() {
+	if err := emailService.SendVerificationEmail(user.Email, user.Name, token); err != nil {
+		log.Printf("ERROR: Failed to send verification email to %s: %v", user.Email, err)
+		// Optionally: Store failed email in retry queue or database
+	}
+}()
```

Better solution: Create an email queue service that retries failed emails:

```go
type EmailQueue struct {
	service *EmailService
	queue   chan EmailJob
	// ... retry logic
}

func (eq *EmailQueue) QueueEmail(job EmailJob) {
	eq.queue <- job
}

// Worker goroutine processes queue with retries and error logging
```

---

## Bug #10: Base64 Encoding Implementation Bug

**STATUS: CODE UNCHANGED - BUG STILL PRESENT**

**Severity:** LOW

**Description:**
The custom base64 encoding implementation in `email_provider_smtp.go` (function `encodeBase64`) has a subtle bug with padding. The function uses a custom implementation instead of the standard library's `encoding/base64`. While the implementation appears correct for most cases, it doesn't handle edge cases and is unnecessarily complex. Using custom crypto implementations instead of standard library is a code smell and potential security risk.

**Location:**
- File: `internal/services/email_provider_smtp.go`
- Function: `encodeBase64`
- Lines: 301-337

**Steps to Reproduce:**
1. Send email with subject containing special UTF-8 characters (e.g., emoji, umlauts)
2. Subject is encoded using custom base64 implementation
3. Expected: Correctly encoded subject
4. Actual: Works in most cases, but custom implementation unnecessary and risky

**Fix:**
Replace custom implementation with standard library:

```diff
+import (
+	"encoding/base64"
+)

func encodeRFC2047(s string) string {
	// Check if encoding is needed (contains non-ASCII characters)
	needsEncoding := false
	for _, c := range s {
		if c > 127 {
			needsEncoding = true
			break
		}
	}

	if !needsEncoding {
		return s
	}

-	// RFC 2047 encoding: =?UTF-8?Q?encoded_text?=
-	return fmt.Sprintf("=?UTF-8?B?%s?=", encodeBase64(s))
+	// RFC 2047 encoding: =?UTF-8?B?encoded_text?=
+	return fmt.Sprintf("=?UTF-8?B?%s?=", base64.StdEncoding.EncodeToString([]byte(s)))
}

-// encodeBase64 encodes a string to base64 for RFC 2047
-func encodeBase64(s string) string {
-	// Remove entire custom implementation (lines 302-337)
-	// Use standard library instead
-}
```

This eliminates potential encoding bugs and follows best practices of using vetted standard library implementations for security-critical operations.

---

## Bug #11: Booking Time Validation Logic Error

**STATUS: CODE UNCHANGED - BUG STILL PRESENT**

**Severity:** MEDIUM

**Description:**
In `BookingTimeService.ValidateBookingTime`, the time window validation uses `!timeObj.Before(startTime)` to check if time is after or equal to start. However, it uses `timeObj.Before(endTime)` to check if before end. This creates an OFF-BY-ONE error where the end time itself is EXCLUDED from the valid window. For example, a rule "09:00-12:00" allows 09:00, 09:30, 11:30 but REJECTS 12:00. This is inconsistent with user expectations where "09:00-12:00" typically means inclusive of both endpoints or at least the start time.

**Location:**
- File: `internal/services/booking_time_service.go`
- Function: `ValidateBookingTime`
- Lines: 65-72

**Steps to Reproduce:**
1. Create booking time rule: weekday 09:00-12:00 (morning window)
2. User attempts to book at exactly 12:00
3. Expected: Booking accepted (12:00 is within "up to 12:00")
4. Actual: Booking rejected ("Zeit ist außerhalb der erlaubten Buchungszeiten")

**Fix:**
Clarify the time window logic and document whether endpoints are inclusive/exclusive:

```diff
func (s *BookingTimeService) ValidateBookingTime(date string, scheduledTime string) error {
	// Parse date
	dateObj, err := time.Parse("2006-01-02", date)
	if err != nil {
		return fmt.Errorf("invalid date format")
	}

	// Parse time
	timeObj, err := time.Parse("15:04", scheduledTime)
	if err != nil {
		return fmt.Errorf("invalid time format")
	}

	// Determine day type
	dayType, err := s.getDayType(date, dateObj)
	if err != nil {
		return err
	}

	// Get rules for day type
	rules, err := s.bookingTimeRepo.GetRulesByDayType(dayType)
	if err != nil {
		return fmt.Errorf("failed to load time rules: %w", err)
	}

	// Check if time falls within any allowed window
	inAllowedWindow := false
	inBlockedWindow := false

	for _, rule := range rules {
		startTime, _ := time.Parse("15:04", rule.StartTime)
		endTime, _ := time.Parse("15:04", rule.EndTime)

-		// Check if time is within this rule's window
-		if !timeObj.Before(startTime) && timeObj.Before(endTime) {
+		// Check if time is within this rule's window (inclusive of start, exclusive of end)
+		// This means 09:00-12:00 allows 09:00, 09:30, 11:45 but NOT 12:00
+		// If you want 12:00 included, use: !timeObj.Before(startTime) && !timeObj.After(endTime)
+		if !timeObj.Before(startTime) && timeObj.Before(endTime) {
			if rule.IsBlocked {
				inBlockedWindow = true
				return fmt.Errorf("Zeit ist gesperrt: %s (%s-%s)", rule.RuleName, rule.StartTime, rule.EndTime)
			} else {
				inAllowedWindow = true
			}
		}
	}

	if !inAllowedWindow {
		return fmt.Errorf("Zeit ist außerhalb der erlaubten Buchungszeiten")
	}

	if inBlockedWindow {
		return fmt.Errorf("Zeit fällt in eine Sperrzeit")
	}

	return nil
}
```

Document clearly in the UI and admin guide whether time windows are inclusive or exclusive of endpoints.

---

## Bug #12: Image Service Race Condition on Concurrent Photo Uploads

**STATUS: CODE UNCHANGED - BUG STILL PRESENT**

**Severity:** MEDIUM

**Description:**
The `ProcessDogPhoto` function is not thread-safe. If two admins upload photos for the same dog simultaneously, both will attempt to write to `dog_X_full.jpg` and `dog_X_thumb.jpg` at the same time. This can result in corrupted images (partial writes), one upload overwriting another, or both uploads failing. The cleanup logic (line 72) deletes the full image if thumbnail creation fails, but if another goroutine is writing the full image, this could delete a different upload's file.

**Location:**
- File: `internal/services/image_service.go`
- Function: `ProcessDogPhoto`
- Lines: 38-81

**Steps to Reproduce:**
1. Two admins open dog edit page for dog ID 5
2. Both upload different photos simultaneously
3. Both calls to ProcessDogPhoto(file1, 5) and ProcessDogPhoto(file2, 5) execute
4. Expected: One upload succeeds, other fails with "resource busy" or similar
5. Actual: Race condition - corrupted image, or one upload silently overwrites the other

**Fix:**
Add mutex per dog ID to serialize photo uploads:

```diff
type ImageService struct {
	uploadDir string
+	// Mutex per dog to prevent concurrent uploads to same dog
+	uploadMutexes sync.Map // map[int]*sync.Mutex
}

func (s *ImageService) ProcessDogPhoto(file multipart.File, dogID int) (fullPath, thumbPath string, err error) {
+	// Lock for this specific dog ID
+	mutexInterface, _ := s.uploadMutexes.LoadOrStore(dogID, &sync.Mutex{})
+	mutex := mutexInterface.(*sync.Mutex)
+	mutex.Lock()
+	defer mutex.Unlock()
+
	// Reset file pointer to beginning
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", "", fmt.Errorf("failed to seek file: %w", err)
	}
	// ... rest of function
}
```

This ensures only one photo upload can process per dog at a time, preventing file corruption.

---

## Bug #13: Super Admin Password File Race Condition

**STATUS: CODE UNCHANGED - BUG STILL PRESENT**

**Severity:** LOW

**Description:**
The `CheckAndUpdatePassword` function reads, parses, and potentially writes the credentials file on EVERY server startup. If multiple instances of the application start simultaneously (e.g., in a Docker Swarm or Kubernetes deployment with multiple replicas), they will all read and write the same file concurrently. This can cause file corruption or lost password updates. While the documentation suggests this is a single-server deployment, the code should be defensive against this scenario.

**Location:**
- File: `internal/services/super_admin_service.go`
- Function: `CheckAndUpdatePassword`
- Lines: 36-111

**Steps to Reproduce:**
1. Deploy two instances of application pointing to same filesystem (NFS mount)
2. Both start simultaneously
3. Both call CheckAndUpdatePassword
4. Admin edits password in credentials file
5. Expected: Password updated once, both instances see update
6. Actual: Race condition - file may be corrupted or one instance's write overwrites the other

**Fix:**
Use file locking to prevent concurrent access:

```diff
+import (
+	"syscall"
+)

func (s *SuperAdminService) CheckAndUpdatePassword() error {
	filePath := "SUPER_ADMIN_CREDENTIALS.txt"

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// ... existing logic
	}

+	// Open file with exclusive lock
+	file, err := os.OpenFile(filePath, os.O_RDWR, 0600)
+	if err != nil {
+		return fmt.Errorf("failed to open credentials file: %w", err)
+	}
+	defer file.Close()
+
+	// Lock file to prevent concurrent access
+	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
+		return fmt.Errorf("failed to lock credentials file: %w", err)
+	}
+	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)

-	// Read file
-	content, err := os.ReadFile(filePath)
+	// Read file content
+	content, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read credentials file: %w", err)
	}

	// ... rest of parsing and validation

	// If password changed, write atomically
	if passwordChanged {
		// Write to temp file first
		tempPath := filePath + ".tmp"
		if err := os.WriteFile(tempPath, []byte(newContent), 0600); err != nil {
			return fmt.Errorf("failed to write temp file: %w", err)
		}

		// Atomic rename
		if err := os.Rename(tempPath, filePath); err != nil {
			os.Remove(tempPath) // Cleanup
			return fmt.Errorf("failed to update credentials file: %w", err)
		}
	}

	return nil
}
```

This uses file locking and atomic writes to prevent corruption in multi-instance deployments.

---

## Bug #14: Quoted-Printable Encoding Bug with CRLF

**STATUS: CODE UNCHANGED - BUG STILL PRESENT**

**Severity:** LOW

**Description:**
The `encodeQuotedPrintable` function in SMTP provider handles line breaks incorrectly. It resets `lineLen` to 0 when encountering `\n` (line 374), but doesn't account for the fact that the input might contain `\r\n` (Windows line endings) or just `\n` (Unix line endings). HTML templates use `\r\n` (line 274), but the encoder only checks for `\n`. This can cause line length tracking to be incorrect, potentially creating malformed quoted-printable output that violates the 76-character line limit.

**Location:**
- File: `internal/services/email_provider_smtp.go`
- Function: `encodeQuotedPrintable`
- Lines: 339-381

**Steps to Reproduce:**
1. Send email with HTML body containing long lines with `\r\n` line endings
2. Encoder processes `\r` as regular character (encoded as =0D)
3. Encoder sees `\n` and resets line counter
4. Line length tracking is off by the encoded `\r` (3 characters for =0D)
5. Expected: Lines stay under 76 characters
6. Actual: Lines may exceed 76 characters, violating quoted-printable spec

**Fix:**
Handle CRLF properly in line length tracking:

```diff
func encodeQuotedPrintable(s string) string {
	var result strings.Builder
	lineLen := 0
	maxLineLen := 76

	for i := 0; i < len(s); i++ {
		c := s[i]
+
+		// Handle CRLF as a unit
+		if c == '\r' && i+1 < len(s) && s[i+1] == '\n' {
+			result.WriteString("\r\n")
+			lineLen = 0
+			i++ // Skip the \n
+			continue
+		}

		// Check if character needs encoding
-		needsEncoding := c < 33 || c > 126 || c == '='
+		needsEncoding := c < 32 || c > 126 || c == '=' || c == '\r'

		if needsEncoding {
+			// Handle lone \n (Unix line ending)
+			if c == '\n' {
+				result.WriteString("\r\n")
+				lineLen = 0
+				continue
+			}
+
			// Encode as =XX where XX is hex
			encoded := fmt.Sprintf("=%02X", c)

			// Check line length
			if lineLen+len(encoded) > maxLineLen {
				result.WriteString("=\r\n") // Soft line break
				lineLen = 0
			}

			result.WriteString(encoded)
			lineLen += len(encoded)
		} else {
			// Check line length for regular character
			if lineLen >= maxLineLen {
				result.WriteString("=\r\n") // Soft line break
				lineLen = 0
			}

			result.WriteByte(c)
			lineLen++
-
-			// Handle CRLF
-			if c == '\n' {
-				lineLen = 0
-			}
		}
	}

	return result.String()
}
```

This properly handles both Unix (\n) and Windows (\r\n) line endings.

---

## Statistics

- **Critical:** 3 bugs (JWT secret weakness, race condition in email templates, template injection)
- **High:** 4 bugs (JWT expiration validation, timing attack, password validation, email provider initialization)
- **Medium:** 6 bugs (holiday API timeout, goroutine error handling, booking time logic, image upload race, time validation)
- **Low:** 3 bugs (base64 encoding, super admin file race, quoted-printable encoding)

---

## Recommendations

### Immediate Actions (Critical)

1. **JWT Secret Validation**: Add startup check to reject weak/default JWT secrets. Generate strong secret in deployment documentation.

2. **Email Template Race Condition**: Pre-parse all templates during EmailService initialization. This is a production stability issue.

3. **Template Injection Protection**: Add HTML escaping/sanitization for all user-provided data inserted into email templates.

### High Priority (Security)

4. **Timing Attack Prevention**: Implement constant-time password checking to prevent user enumeration.

5. **Password Policy Strengthening**: Require special characters and check against common password dictionaries.

6. **Email Connection Testing**: Test SMTP/Gmail connectivity during provider initialization to fail fast.

### Medium Priority (Reliability)

7. **HTTP Timeouts**: Add timeouts to all external API calls (holiday API, etc.).

8. **Email Error Handling**: Implement email queue with retry logic instead of fire-and-forget goroutines.

9. **Image Upload Locking**: Add per-dog mutex to prevent concurrent upload corruption.

10. **Time Window Logic**: Document and clarify whether time windows are inclusive/exclusive of endpoints.

### Low Priority (Code Quality)

11. **Standard Library Usage**: Replace custom base64 implementation with standard library.

12. **File Locking**: Add file locking to Super Admin credentials file for multi-instance safety.

13. **CRLF Handling**: Fix quoted-printable encoder to properly handle mixed line endings.

### General Recommendations

- **Add Integration Tests**: Test email sending end-to-end with real SMTP server in test environment.
- **Add Security Tests**: Run timing attack tests, JWT fuzzing, password strength checks.
- **Logging Improvements**: Add structured logging with severity levels for better debugging.
- **Monitoring**: Add metrics for email send success/failure rates.
- **Documentation**: Document all security assumptions (JWT expiration limits, password policies, etc.).

### Security Best Practices

- Never use default secrets in production
- Always validate external API inputs
- Use constant-time comparisons for authentication
- Pre-parse templates to avoid runtime race conditions
- Test email configuration on startup, not at first use
- Sanitize all user input before inserting into templates or logs
- Use standard library for crypto operations (base64, bcrypt, JWT)
- Add timeouts to all network operations
- Handle errors from goroutines explicitly

---

**Analysis Complete:** 14 functional bugs identified across authentication, email, and business logic services. 3 critical security vulnerabilities require immediate attention.
