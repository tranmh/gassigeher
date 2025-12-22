# Bug Report: services

**Analysis Date:** 2025-12-22
**Directory Analyzed:** `/home/tranmh/work/gassigeher-saas/internal/services`
**Files Analyzed:** 22 files
**Bugs Found:** 9 bugs

---

## Summary

The services directory contains business logic for authentication, email, holidays, booking times, images, provisioning, Stripe payments, and S3 storage. Analysis revealed **9 functional bugs** across multiple services:

- **Critical:** 3 bugs (security vulnerabilities, resource leaks)
- **High:** 4 bugs (logic errors, race conditions, data integrity)
- **Medium:** 2 bugs (error handling gaps, encoding issues)

Most critical issues involve **modulo bias in random password generation** (weakening cryptographic strength), **goroutine leak in brute force service** (memory exhaustion), and **missing path validation in S3 service** (security vulnerability).

---

## Bugs

## Bug #1: Modulo Bias in Temporary Password Generation

**Severity:** CRITICAL

**Description:**
The `GenerateTempPassword()` function in `auth_service.go` uses modulo operator on random bytes to select characters from a charset. This introduces **modulo bias**, making certain characters more likely than others when the charset size doesn't evenly divide 256. This weakens the password entropy and makes brute force attacks slightly easier. The charset has 56 characters, so 256 % 56 = 32, meaning the first 32 characters in the charset get extra probability (~1.6% higher frequency).

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/services/auth_service.go`
- Function: `GenerateTempPassword`
- Lines: 64-65

**Steps to Reproduce:**
1. Generate 1 million temporary passwords
2. Count character frequency distribution
3. Observe: Some characters appear ~1.6% more often than others
4. Expected: Uniform distribution across all characters
5. Actual: Non-uniform distribution (modulo bias)

**Fix:**
Use rejection sampling to eliminate modulo bias:

```diff
func (s *AuthService) GenerateTempPassword() (string, error) {
	const charset = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	const length = 12

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate temp password: %w", err)
	}

	result := make([]byte, length)
	for i := range bytes {
-		result[i] = charset[int(bytes[i])%len(charset)]
+		// Use rejection sampling to eliminate modulo bias
+		for {
+			n := int(bytes[i])
+			if n < (256 - 256%len(charset)) {
+				result[i] = charset[n%len(charset)]
+				break
+			}
+			// Reject and get new random byte
+			if _, err := rand.Read(bytes[i:i+1]); err != nil {
+				return "", fmt.Errorf("failed to generate temp password: %w", err)
+			}
+		}
	}

	// Ensure password meets requirements
	result[0] = "ABCDEFGHJKLMNPQRSTUVWXYZ"[int(bytes[0])%24]
	result[1] = "abcdefghjkmnpqrstuvwxyz"[int(bytes[1])%23]
	result[2] = "23456789"[int(bytes[2])%8]

	return string(result), nil
}
```

Alternatively, use `crypto/rand.Int()` which handles this correctly.

---

## Bug #2: Goroutine Leak in BruteForceService

**Severity:** CRITICAL

**Description:**
The `NewBruteForceService()` function starts a cleanup goroutine (`go bfs.cleanupStaleEntries()`) that runs indefinitely with no way to stop it. If multiple instances of `BruteForceService` are created (e.g., in tests or if the service is reinitialized), these goroutines accumulate and never terminate, causing a **goroutine leak** and eventual memory exhaustion.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/services/brute_force_service.go`
- Function: `NewBruteForceService`, `cleanupStaleEntries`
- Lines: 34, 113-127

**Steps to Reproduce:**
1. Create 100 instances of `BruteForceService` in a loop
2. Check goroutine count with `runtime.NumGoroutine()`
3. Expected: Goroutines are cleaned up when service instances are garbage collected
4. Actual: 100+ goroutines remain running indefinitely (one per instance)
5. Result: Memory leak, goroutine accumulation

**Fix:**
Add a context or stop channel to gracefully shut down the cleanup goroutine:

```diff
type BruteForceService struct {
	failures    map[string]*FailureRecord
	mu          sync.RWMutex
	maxAttempts int
	lockoutBase time.Duration
	maxLockout  time.Duration
+	stopCleanup chan struct{}
}

func NewBruteForceService() *BruteForceService {
	bfs := &BruteForceService{
		failures:    make(map[string]*FailureRecord),
		maxAttempts: 3,
		lockoutBase: 30 * time.Second,
		maxLockout:  30 * time.Minute,
+		stopCleanup: make(chan struct{}),
	}

	// Start cleanup goroutine
	go bfs.cleanupStaleEntries()

	return bfs
}

+// Stop stops the cleanup goroutine
+func (s *BruteForceService) Stop() {
+	close(s.stopCleanup)
+}

func (s *BruteForceService) cleanupStaleEntries() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

-	for range ticker.C {
+	for {
+		select {
+		case <-ticker.C:
+			s.mu.Lock()
+			cutoff := time.Now().Add(-2 * time.Hour)
+			for key, record := range s.failures {
+				if record.LastFailed.Before(cutoff) && time.Now().After(record.LockedUntil) {
+					delete(s.failures, key)
+				}
+			}
+			s.mu.Unlock()
+		case <-s.stopCleanup:
+			return
+		}
-		s.mu.Lock()
-		cutoff := time.Now().Add(-2 * time.Hour)
-		for key, record := range s.failures {
-			if record.LastFailed.Before(cutoff) && time.Now().After(record.LockedUntil) {
-				delete(s.failures, key)
-			}
-		}
-		s.mu.Unlock()
	}
}
```

Then call `bfs.Stop()` when the service is no longer needed (e.g., in `defer` or shutdown handlers).

---

## Bug #3: Missing Validation in BookingTimeService

**Severity:** HIGH

**Description:**
The `ValidateBookingTime()` function does not validate if the date is in the past before checking time slot rules. This allows bookings to be validated for past dates, which should be rejected. While handlers might check this, services should be defensive and validate all inputs independently.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/services/booking_time_service.go`
- Function: `ValidateBookingTime`
- Lines: 31-84

**Steps to Reproduce:**
1. Call `ValidateBookingTime(tenantID, "2020-01-01", "10:00")`
2. Expected: Error "date cannot be in the past"
3. Actual: Validates successfully if time slot rules allow it
4. Impact: Invalid bookings could be created if handler validation is bypassed

**Fix:**
Add date validation at the beginning of the function:

```diff
func (s *BookingTimeService) ValidateBookingTime(tenantID int, date string, scheduledTime string) error {
	// Parse date
	dateObj, err := time.Parse("2006-01-02", date)
	if err != nil {
		return fmt.Errorf("invalid date format")
	}

+	// Validate date is not in the past
+	today := time.Now().Truncate(24 * time.Hour)
+	if dateObj.Before(today) {
+		return fmt.Errorf("date cannot be in the past")
+	}

	// Parse time
	timeObj, err := time.Parse("15:04", scheduledTime)
	if err != nil {
		return fmt.Errorf("invalid time format")
	}

	// ... rest of validation
}
```

---

## Bug #4: Race Condition in HolidayService Cache Population

**Severity:** HIGH

**Description:**
The `FetchAndCacheHolidays()` function has a potential **race condition**. If multiple goroutines call this function simultaneously for the same year/state, they will all miss the cache, fetch from the API concurrently, and insert duplicate holiday records. While the database might handle duplicates via `INSERT OR IGNORE`, this wastes API quota and database resources.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/services/holiday_service.go`
- Function: `FetchAndCacheHolidays`
- Lines: 28-97

**Steps to Reproduce:**
1. Start 10 goroutines simultaneously
2. Each calls `FetchAndCacheHolidays(tenantID, 2025)`
3. Expected: Only one API call is made, others wait or use the cached result
4. Actual: 10 concurrent API calls are made (cache check → API fetch → insert)
5. Impact: API rate limit exceeded, duplicate database operations

**Fix:**
Use a mutex or sync.Map to coordinate concurrent cache fetches:

```diff
type HolidayService struct {
	holidayRepo  *repository.HolidayRepository
	settingsRepo *repository.SettingsRepository
+	fetchMutex   sync.Mutex // Protects concurrent fetches
}

func (s *HolidayService) FetchAndCacheHolidays(tenantID int, year int) error {
+	// Prevent concurrent fetches for the same year/state
+	s.fetchMutex.Lock()
+	defer s.fetchMutex.Unlock()

	// Get state from settings
	state := "BW" // Default
	if setting, err := s.settingsRepo.Get(tenantID, "feiertage_state"); err == nil && setting != nil && setting.Value != "" {
		state = setting.Value
	}

	// Check cache first (now protected by mutex)
	cached, err := s.holidayRepo.GetCachedHolidays(year, state)
	if err == nil && cached != "" {
		return s.populateHolidaysFromCache(tenantID, cached, year)
	}

	// Cache miss - fetch from API (only one goroutine gets here)
	// ... rest of function
}
```

---

## Bug #5: Missing Path Validation in S3Service Upload

**Severity:** CRITICAL

**Description:**
The `Upload()` function in `s3_service.go` does not validate the `path` parameter, allowing potential **path traversal attacks** if user input is passed directly. An attacker could use `../../../` sequences to upload files outside the intended tenant directory in the S3 bucket.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/services/s3_service.go`
- Function: `Upload`
- Lines: 59-72

**Steps to Reproduce:**
1. Call `s3Service.Upload(ctx, "tenant1", "../../../malicious.txt", data, "text/plain")`
2. Expected: Error "invalid path: path traversal not allowed"
3. Actual: File uploaded to `tenants/../../../malicious.txt` (escapes tenant directory)
4. Impact: Cross-tenant data access, potential bucket pollution

**Fix:**
Validate and sanitize the path parameter:

```diff
+import (
+	"path/filepath"
+	"strings"
+)

func (s *S3Service) Upload(ctx context.Context, tenantSlug, path string, data []byte, contentType string) (string, error) {
+	// Validate path (no absolute paths or traversal)
+	if filepath.IsAbs(path) {
+		return "", fmt.Errorf("absolute paths not allowed")
+	}
+	if strings.Contains(path, "..") {
+		return "", fmt.Errorf("path traversal not allowed")
+	}
+	// Clean the path
+	path = filepath.Clean(path)

	// Organize by tenant: tenants/{slug}/{path}
	objectKey := fmt.Sprintf("tenants/%s/%s", tenantSlug, path)

	_, err := s.client.PutObject(ctx, s.bucketName, objectKey, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Return public URL
	return fmt.Sprintf("%s/%s", s.publicURL, objectKey), nil
}
```

Apply the same validation to `DeleteByPath()` as well (line 80).

---

## Bug #6: Missing Transaction Rollback in ProvisioningService

**Severity:** HIGH

**Description:**
The `ProvisionTenant()` function receives a transaction pointer but does not handle rollback on error. If any of the provisioning steps fail (e.g., `CreateDefaultColors` succeeds but `CreateDefaultBookingRules` fails), the transaction is left in an inconsistent state. The caller is responsible for rollback, but this creates a **tight coupling** and error-prone usage pattern.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/services/provisioning_service.go`
- Function: `ProvisionTenant`
- Lines: 94-111

**Steps to Reproduce:**
1. Mock `CreateDefaultBookingRules` to return an error
2. Call `ProvisionTenant(tx, tenantID)`
3. Expected: Transaction rolled back, no partial data inserted
4. Actual: `CreateDefaultColors` has already inserted data, transaction still open
5. Impact: If caller forgets to rollback, partial provisioning data remains

**Fix:**
Let the service manage the transaction internally:

```diff
type ProvisioningService struct {
	db *sql.DB
}

// ProvisionTenant creates all default data for a new tenant
-func (s *ProvisioningService) ProvisionTenant(tx *sql.Tx, tenantID int) error {
+func (s *ProvisioningService) ProvisionTenant(tenantID int) error {
+	// Begin transaction
+	tx, err := s.db.Begin()
+	if err != nil {
+		return fmt.Errorf("failed to begin transaction: %w", err)
+	}
+	defer tx.Rollback() // Rollback if not committed
+
	// Create default color categories
	if err := s.CreateDefaultColors(tx, tenantID); err != nil {
-		return err
+		return fmt.Errorf("failed to create default colors: %w", err)
	}

	// Create default booking time rules
	if err := s.CreateDefaultBookingRules(tx, tenantID); err != nil {
-		return err
+		return fmt.Errorf("failed to create default booking rules: %w", err)
	}

	// Create default system settings
	if err := s.CreateDefaultSettings(tx, tenantID); err != nil {
-		return err
+		return fmt.Errorf("failed to create default settings: %w", err)
	}

+	// Commit transaction
+	if err := tx.Commit(); err != nil {
+		return fmt.Errorf("failed to commit transaction: %w", err)
+	}
+
	return nil
}
```

---

## Bug #7: Integer Overflow Risk in Stripe Metadata

**Severity:** HIGH

**Description:**
The `CreateCheckoutSession()` function converts `tenantID` (int) to string using `fmt.Sprintf("%d", tenantID)`. If `tenantID` is negative or invalid, this could cause issues when parsing back from metadata. While unlikely, defensive validation would prevent edge cases.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/services/stripe_service.go`
- Function: `CreateCheckoutSession`
- Lines: 95-97

**Steps to Reproduce:**
1. Call `CreateCheckoutSession(-1, "pro", "monthly", "test@example.com")`
2. Session created with metadata `tenant_id: "-1"`
3. Webhook parses this back with `fmt.Sscanf` (line 206)
4. Expected: Error "invalid tenant ID"
5. Actual: Session created with invalid tenant ID
6. Impact: Subscription associated with invalid tenant, orphaned payment

**Fix:**
Validate tenant ID before creating the session:

```diff
func (s *StripeService) CreateCheckoutSession(tenantID int, planSlug, billingCycle, customerEmail string) (*stripe.CheckoutSession, error) {
	if s.secretKey == "" {
		return nil, errors.New("stripe API key not configured")
	}

+	// Validate tenant ID
+	if tenantID <= 0 {
+		return nil, fmt.Errorf("invalid tenant ID: must be positive")
+	}

	// Validate plan
	if planSlug != "pro" {
		return nil, errors.New("invalid plan: only 'pro' plan requires payment")
	}

	// ... rest of function
}
```

Also add validation in `ParseCheckoutSessionEvent()`:

```diff
func (s *StripeService) ParseCheckoutSessionEvent(event *stripe.Event) (*CheckoutSessionData, error) {
	// ... existing code ...

	// Extract tenant_id from metadata
	if tenantIDStr, ok := session.Metadata["tenant_id"]; ok {
		var tenantID int
		fmt.Sscanf(tenantIDStr, "%d", &tenantID)
+		if tenantID <= 0 {
+			return nil, fmt.Errorf("invalid tenant_id in metadata: %d", tenantID)
+		}
		data.TenantID = tenantID
	}

	return data, nil
}
```

---

## Bug #8: Missing Email Format Validation in EmailService

**Severity:** MEDIUM

**Description:**
The `SendEmail()` method in `email_service.go` calls the provider's `SendEmail()` without validating the email format. While the SMTP provider validates the format (line 52-54 in `email_provider_smtp.go`), the Gmail provider does not. This allows invalid email addresses to reach the Gmail API, causing failures that could be caught earlier.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/services/email_service.go`
- Function: `SendEmail`
- Lines: 69-71

**Steps to Reproduce:**
1. Use Gmail provider
2. Call `emailService.SendEmail("not-an-email", "Subject", "Body")`
3. Expected: Immediate error "invalid email address"
4. Actual: Gmail API call fails with cryptic error
5. Impact: Unnecessary API calls, confusing error messages

**Fix:**
Add email validation in the `EmailService.SendEmail()` method:

```diff
+import "net/mail"

// SendEmail sends an email using the configured provider
func (s *EmailService) SendEmail(to, subject, body string) error {
+	// Validate email format
+	if _, err := mail.ParseAddress(to); err != nil {
+		return fmt.Errorf("invalid recipient email address: %w", err)
+	}
+
	return s.provider.SendEmail(to, subject, body)
}
```

This ensures consistent validation regardless of the provider.

---

## Bug #9: Weak Encoding in SMTP Quoted-Printable Implementation

**Severity:** MEDIUM

**Description:**
The `encodeQuotedPrintable()` function in `email_provider_smtp.go` has a logic error in handling line breaks. It checks `if c == '\n'` (line 374) but does not handle `\r` or CRLF sequences properly. This can cause **malformed email bodies** when the HTML contains Windows-style line endings (`\r\n`), leading to broken formatting or rejected emails by strict SMTP servers.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/services/email_provider_smtp.go`
- Function: `encodeQuotedPrintable`
- Lines: 340-381

**Steps to Reproduce:**
1. Create HTML body with CRLF line endings: `"<html>\r\n<body>Test</body>\r\n</html>"`
2. Send email via SMTP provider
3. Expected: Email delivered with proper formatting
4. Actual: Line counter doesn't reset on `\r`, causing premature soft line breaks
5. Impact: Malformed email body, potential delivery failures

**Fix:**
Properly handle CRLF sequences in quoted-printable encoding:

```diff
func encodeQuotedPrintable(s string) string {
	var result strings.Builder
	lineLen := 0
	maxLineLen := 76

	for i := 0; i < len(s); i++ {
		c := s[i]

+		// Handle CRLF as a single unit
+		if c == '\r' && i+1 < len(s) && s[i+1] == '\n' {
+			result.WriteString("\r\n")
+			lineLen = 0
+			i++ // Skip the \n
+			continue
+		}

		// Check if character needs encoding
-		needsEncoding := c < 33 || c > 126 || c == '='
+		needsEncoding := c < 33 || c > 126 || c == '=' || c == '\r'

		if needsEncoding {
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

			// Handle newlines
-			if c == '\n' {
+			if c == '\n' || c == '\r' {
				lineLen = 0
			}
		}
	}

	return result.String()
}
```

Alternatively, use Go's standard library `mime/quotedprintable` package instead of implementing it manually.

---

## Statistics

- **Critical:** 3 bugs
  - Bug #1: Modulo bias in password generation (security)
  - Bug #2: Goroutine leak (resource leak)
  - Bug #5: Missing path validation in S3 (security)

- **High:** 4 bugs
  - Bug #3: Missing date validation (logic error)
  - Bug #4: Race condition in holiday service (concurrency)
  - Bug #6: Missing transaction rollback (data integrity)
  - Bug #7: Integer overflow risk in Stripe metadata (logic error)

- **Medium:** 2 bugs
  - Bug #8: Missing email validation (error handling)
  - Bug #9: Weak quoted-printable encoding (data integrity)

---

## Recommendations

### Immediate Actions (Critical Bugs)

1. **Fix modulo bias in password generation** (Bug #1) - Use `crypto/rand.Int()` or rejection sampling
2. **Add Stop() method to BruteForceService** (Bug #2) - Prevent goroutine leaks
3. **Validate S3 paths** (Bug #5) - Prevent path traversal attacks

### High Priority (High Severity Bugs)

4. **Add date validation to booking service** (Bug #3) - Defensive programming
5. **Add mutex to holiday cache fetch** (Bug #4) - Prevent race conditions
6. **Improve provisioning transaction handling** (Bug #6) - Better error handling
7. **Validate Stripe metadata** (Bug #7) - Prevent invalid tenant associations

### Medium Priority (Medium Severity Bugs)

8. **Add email validation to EmailService** (Bug #8) - Consistent error handling
9. **Fix quoted-printable encoding** (Bug #9) - Use standard library or fix CRLF handling

### General Improvements

1. **Add unit tests for cryptographic functions** - Test password generation, token generation
2. **Add integration tests for email providers** - Test both Gmail and SMTP paths
3. **Add concurrency tests** - Test race conditions in HolidayService, BruteForceService
4. **Add resource cleanup patterns** - Ensure all services with goroutines have Stop() methods
5. **Add input validation at service boundaries** - All public methods should validate inputs
6. **Use standard library where possible** - Replace custom encoding with `mime/quotedprintable`
7. **Add metrics/logging for service operations** - Track API calls, email sends, errors
8. **Document transaction ownership** - Clarify which layer manages transactions

### Code Quality

- **Consistency:** Some services validate inputs, others rely on handlers
- **Defensive programming:** Services should be self-contained and validate all inputs
- **Resource management:** Add lifecycle management (Start/Stop) for services with goroutines
- **Error messages:** Include context in all error messages (service name, parameters)
- **Testing:** Add comprehensive unit tests for all edge cases

---

**End of Bug Report**
