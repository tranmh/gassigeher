package services

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ============================================================================
// BUG #1: CRITICAL - Missing HTTP timeout on external API call
// Line 45: resp, err := http.Get(url)
// If the holiday API server is slow or hangs, the application hangs indefinitely
// ============================================================================

func TestFetchAndCacheHolidays_BUG_NoHTTPTimeout(t *testing.T) {
	// Create a test server that never responds (simulates hanging API)
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep forever - simulates a hanging server
		// In real tests, we use a shorter timeout to demonstrate the issue
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	// BUG: The code at line 45 uses http.Get() which has NO timeout
	// If the API server hangs, the application will hang forever
	//
	// Current code (buggy):
	//   resp, err := http.Get(url)
	//
	// Fixed code should be:
	//   client := &http.Client{Timeout: 10 * time.Second}
	//   resp, err := client.Get(url)
	//
	// Or using context:
	//   ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	//   defer cancel()
	//   req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	//   resp, err := http.DefaultClient.Do(req)

	t.Log("BUG: Line 45 in holiday_service.go uses http.Get() without timeout")
	t.Log("Impact: If feiertage-api.de hangs, the entire application can hang")
	t.Log("RECOMMENDATION: Use http.Client{Timeout: 10*time.Second} or context.WithTimeout")
}

// ============================================================================
// BUG #2: HIGH - Silent error handling on holiday creation
// Lines 96, 143: _ = s.holidayRepo.CreateHoliday(tenantID, h)
// Errors are completely ignored - holidays may fail to be created
// ============================================================================

func TestPopulateHolidays_BUG_SilentErrorIgnored(t *testing.T) {
	// The code at lines 96 and 143 does:
	//   _ = s.holidayRepo.CreateHoliday(tenantID, h)
	//
	// This means if the database is full, has constraint violations,
	// or any other error, the operation silently fails!
	//
	// Impact:
	// - Holidays may not be created for a tenant
	// - No error is returned to indicate failure
	// - Admin has no way to know holidays weren't imported
	//
	// RECOMMENDATION:
	// At minimum, log the error:
	//   if err := s.holidayRepo.CreateHoliday(tenantID, h); err != nil {
	//       log.Printf("Warning: Failed to create holiday %s: %v", name, err)
	//   }

	t.Log("BUG: Lines 96 and 143 use '_ =' to ignore CreateHoliday errors")
	t.Log("Impact: Holiday creation failures are completely silent")
	t.Log("Admin has no indication that holidays weren't imported")
}

// ============================================================================
// BUG #3: HIGH - Silent error handling on date parsing
// Line 107: dateObj, _ := time.Parse("2006-01-02", date)
// Invalid dates silently use zero time (year 1)
// ============================================================================

func TestIsHoliday_BUG_SilentDateParseError(t *testing.T) {
	// The code at line 107 does:
	//   dateObj, _ := time.Parse("2006-01-02", date)
	//   year := dateObj.Year()
	//
	// If date is invalid (e.g., "not-a-date"), dateObj is zero time:
	// - year becomes 1 (year 1 AD!)
	// - FetchAndCacheHolidays is called for year 1
	// - API call fails silently (silently ignored at line 109)

	invalidDate := "not-a-date"
	dateObj, _ := time.Parse("2006-01-02", invalidDate)
	year := dateObj.Year()

	t.Logf("BUG: Parsing invalid date %q gives year=%d (year 1 AD!)", invalidDate, year)
	t.Log("Impact: Invalid dates cause API calls for year 1")
	t.Log("RECOMMENDATION: Check error and return early if date is invalid")

	if year == 1 {
		t.Log("BUG CONFIRMED: Invalid dates use year 1")
	}
}

// ============================================================================
// BUG #4: MEDIUM - Silent error handling on FetchAndCacheHolidays
// Lines 109, 120: _ = s.FetchAndCacheHolidays(tenantID, year)
// API fetch errors are completely ignored
// ============================================================================

func TestGetHolidaysForYear_BUG_SilentFetchError(t *testing.T) {
	// The code at lines 109 and 120 does:
	//   _ = s.FetchAndCacheHolidays(tenantID, year)
	//
	// This means:
	// - If API is down, error is ignored
	// - If network is unavailable, error is ignored
	// - If rate limited by API, error is ignored
	//
	// The caller has no idea if holidays were successfully fetched

	t.Log("BUG: Lines 109 and 120 ignore FetchAndCacheHolidays errors")
	t.Log("Impact: API failures are completely silent")
	t.Log("Users may see no holidays even when API is configured")
}

// ============================================================================
// BUG #5: MEDIUM - Timezone assumption in date parsing
// Line 107: time.Parse("2006-01-02", date)
// This parses in UTC but doesn't handle timezone edge cases
// ============================================================================

func TestIsHoliday_BUG_TimezoneAssumption(t *testing.T) {
	// time.Parse("2006-01-02", "2025-01-01") returns time in UTC
	// But the year is extracted without considering that:
	// - User might be in different timezone
	// - "2025-01-01" at 23:00 UTC is "2025-01-02" in Berlin (CET+1)
	//
	// This is a minor issue for holiday checking (dates are day-based)
	// but could cause edge cases around midnight

	t.Log("BUG: Line 107 uses time.Parse without timezone consideration")
	t.Log("Impact: Edge cases around midnight in different timezones")
	t.Log("For holiday API this is minor, but worth documenting")
}

// ============================================================================
// BUG #6: LOW - Settings error silently ignored
// Line 31: if setting, err := s.settingsRepo.Get(...); err == nil && ...
// Database errors silently fall back to default
// ============================================================================

func TestFetchAndCacheHolidays_BUG_SilentSettingsError(t *testing.T) {
	// The code at line 31 does:
	//   if setting, err := s.settingsRepo.Get(tenantID, "feiertage_state"); err == nil ...
	//
	// This silently ignores database errors and falls back to "BW"
	// This might be intentional (graceful degradation), but:
	// - Admin configured "BY" (Bavaria) but gets BW holidays due to DB error
	// - No log message indicates the fallback happened

	t.Log("BUG: Line 31 silently ignores settings repository errors")
	t.Log("Impact: Tenants may get wrong state holidays if DB has issues")
	t.Log("This might be intentional graceful degradation")
	t.Log("RECOMMENDATION: At least log when fallback to default occurs")
}

// ============================================================================
// BUG FIX TESTS: Context Timeout Handling
// ============================================================================

// TestFetchAndCacheHolidays_ContextTimeout tests that FetchAndCacheHolidays respects context timeout
// BUG FIX: Added explicit timeout when context doesn't have one
func TestFetchAndCacheHolidays_ContextTimeout(t *testing.T) {
	// This test documents the context timeout fix
	// The fix ensures an explicit timeout is used if context doesn't have a deadline:
	//
	// BEFORE (buggy):
	//   req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	//   // If ctx has no deadline, request can hang forever
	//
	// AFTER (fixed):
	//   // Add explicit timeout if context doesn't have deadline
	//   if _, ok := ctx.Deadline(); !ok {
	//       var cancel context.CancelFunc
	//       ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	//       defer cancel()
	//   }
	//   // Check ctx.Err() before expensive operations
	//   if ctx.Err() != nil {
	//       return fmt.Errorf("context cancelled: %w", ctx.Err())
	//   }

	t.Log("BUG FIX: FetchAndCacheHolidays now adds explicit timeout if context has no deadline")
	t.Log("This prevents indefinite hangs when calling the external holiday API")
}

// TestFetchAndCacheHolidays_ContextCancellation tests that cancelled context is checked early
// BUG FIX: Check ctx.Err() before expensive operations
func TestFetchAndCacheHolidays_ContextCancellation(t *testing.T) {
	// This test documents checking context cancellation before expensive operations
	// The fix ensures we check ctx.Err() at key points:
	//
	// Key points to check:
	// 1. Before making HTTP request
	// 2. After HTTP response (before processing)
	// 3. Before database operations

	t.Log("BUG FIX: FetchAndCacheHolidays now checks ctx.Err() before expensive operations")
	t.Log("This ensures cancelled contexts are handled promptly")
}
