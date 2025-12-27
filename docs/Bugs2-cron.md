# Bug Report: internal/cron

**Analysis Date:** 2025-12-27
**Directory Analyzed:** `/home/tranmh/work/gassigeher-saas/internal/cron`
**Files Analyzed:** 4 files
**Bugs Found:** 8 bugs

---

## Summary

The cron package handles critical automated tasks including booking auto-completion, reminder emails, user auto-deactivation, demo tenant reset, and tenant activity tracking. Analysis revealed 8 functional bugs spanning race conditions, timezone inconsistencies, error handling gaps, and logical flaws in activity detection. The most critical issues involve potential timezone bugs affecting reminder delivery, missing error propagation in row iteration, and a misleading function that doesn't actually flag inactive tenants as its name suggests.

**Severity Distribution:**
- **Critical:** 2 bugs (timezone issues causing reminder/completion failures)
- **High:** 3 bugs (missing error handling, goroutine leaks, function doesn't do what it claims)
- **Medium:** 2 bugs (midnight crossing edge case, inconsistent tenant filtering)
- **Low:** 1 bug (misleading function naming)

---

## Bugs

## Bug #1: Timezone Inconsistency Between Auto-Complete and Reminders

**Severity:** CRITICAL

**Description:**
The `autoCompleteBookings()` and `sendBookingReminders()` functions use different time sources for scheduling decisions. Auto-complete uses `time.Now()` (server's local timezone), while the cron scheduler uses Europe/Berlin timezone via `runDaily()`. This can cause bookings to be completed before/after reminders are sent, or reminders to be sent at incorrect times if the server timezone differs from Europe/Berlin.

**Impact:**
- Reminders could be sent AFTER bookings are already auto-completed
- Bookings could be completed BEFORE reminders are sent
- Time-sensitive operations become unpredictable across different server locations
- Users in Europe/Berlin timezone may receive incorrect reminder timing

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/cron/cron.go`
- Functions: `autoCompleteBookings()` (line 126), `sendBookingReminders()` (line 141)
- Lines: 126-138, 141-205

**Steps to Reproduce:**
1. Deploy server in a non-Europe/Berlin timezone (e.g., UTC, US Eastern)
2. Create booking scheduled for 09:00 Europe/Berlin time
3. Observe that auto-complete runs at server local time
4. Observe that reminder calculation uses server local time for `time.Now()`
5. Result: Timing mismatch causes operational issues

**Expected Behavior:**
All time-sensitive cron operations should use a consistent timezone (Europe/Berlin).

**Fix:**
Use Europe/Berlin timezone consistently across all booking-related time operations:

```diff
// autoCompleteBookings marks past scheduled bookings as completed
func (s *CronService) autoCompleteBookings() {
+	berlinLoc := getBerlinLocation()
+	now := time.Now().In(berlinLoc)
-	count, err := s.bookingRepo.AutoComplete()
+	count, err := s.bookingRepo.AutoCompleteWithTimezone(now)
	if err != nil {
		log.Printf("Error auto-completing bookings: %v", err)
		return
	}

	if count > 0 {
		log.Printf("Auto-completed %d booking(s)", count)
	} else {
		log.Println("Auto-complete check: no bookings to complete")
	}
}

// sendBookingReminders sends reminders for upcoming bookings (1-2 hours before)
func (s *CronService) sendBookingReminders() {
	// Check if email service is available
	if s.emailService == nil {
		log.Println("Reminder check: email service not configured, skipping")
		return
	}

+	berlinLoc := getBerlinLocation()
+	now := time.Now().In(berlinLoc)
-	// Get bookings that need reminders
-	bookings, err := s.bookingRepo.GetForReminders()
+	bookings, err := s.bookingRepo.GetForRemindersWithTimezone(now)
	if err != nil {
		log.Printf("Error getting bookings for reminders: %v", err)
		return
	}
	// ... rest of function
}
```

Additionally, update `BookingRepository.AutoComplete()` and `GetForReminders()` to accept a `time.Time` parameter for consistent timezone handling.

---

## Bug #2: Missing rows.Err() Check After Iteration in CheckAndFlagInactiveTenants

**Severity:** HIGH

**Description:**
The `CheckAndFlagInactiveTenants()` function iterates through database rows but does NOT check `rows.Err()` after the loop completes. According to Go database best practices, `rows.Err()` must always be checked after `rows.Next()` returns false, as errors during iteration are deferred until the loop ends. Failure to check this can silently ignore database errors, causing incomplete tenant activity checks.

**Impact:**
- Database errors during row iteration are silently ignored
- Incomplete tenant activity checks - some inactive tenants may not be detected
- No indication to operators that the cron job failed partially
- Data integrity issues where admins don't see all inactive tenants

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/cron/tenant_activity.go`
- Function: `CheckAndFlagInactiveTenants()`
- Lines: 117-166

**Steps to Reproduce:**
1. Simulate a database connection error during row iteration (e.g., network timeout mid-query)
2. Run `CheckAndFlagInactiveTenants()`
3. Observe that function completes successfully with log message "Tenant activity check complete"
4. Verify that `rows.Err()` contains an error but it's never checked
5. Result: Error is silently ignored, incomplete data processing

**Expected Behavior:**
Function should check `rows.Err()` after iteration and return error if present.

**Fix:**
Add `rows.Err()` check after the iteration loop:

```diff
	for rows.Next() {
		var tenantID int
		var slug, name string

		if err := rows.Scan(&tenantID, &slug, &name); err != nil {
			log.Printf("Error scanning tenant: %v", err)
			continue
		}

		// Check last booking date for this tenant
		var lastBooking *time.Time
		bookingQuery := `
			SELECT MAX(created_at)
			FROM bookings
			WHERE tenant_id = ?
		`
		c.db.QueryRow(bookingQuery, tenantID).Scan(&lastBooking)

		// Check last user activity for this tenant
		var lastActivity *time.Time
		activityQuery := `
			SELECT MAX(last_activity_at)
			FROM users
			WHERE tenant_id = ? AND is_active = 1
		`
		c.db.QueryRow(activityQuery, tenantID).Scan(&lastActivity)

		// Determine the most recent activity
		var mostRecentActivity *time.Time
		if lastBooking != nil && lastActivity != nil {
			if lastBooking.After(*lastActivity) {
				mostRecentActivity = lastBooking
			} else {
				mostRecentActivity = lastActivity
			}
		} else if lastBooking != nil {
			mostRecentActivity = lastBooking
		} else if lastActivity != nil {
			mostRecentActivity = lastActivity
		}

		// Check if tenant is inactive
		isInactive := mostRecentActivity == nil || mostRecentActivity.Before(cutoffDate)

		if isInactive {
			inactiveCount++
			log.Printf("Tenant '%s' (ID: %d) flagged as inactive - last activity: %v",
				slug, tenantID, mostRecentActivity)
		}
	}

+	// Check for errors during row iteration
+	if err := rows.Err(); err != nil {
+		return fmt.Errorf("error iterating tenant rows: %w", err)
+	}

	log.Printf("Tenant activity check complete. Found %d inactive tenants", inactiveCount)
	return nil
```

This ensures database errors are properly caught and propagated.

---

## Bug #3: CheckAndFlagInactiveTenants Doesn't Actually Flag Tenants

**Severity:** HIGH

**Description:**
The function `CheckAndFlagInactiveTenants()` has a misleading name and incomplete implementation. Despite the name containing "Flag", the function only LOGS inactive tenants but doesn't actually update the database to persist the inactive flag. The comment says "This is run as a daily cron job" implying it should have persistent effects, but inactive status is recalculated on every run without persistence. This makes it impossible for central admins to track which tenants have been flagged and when.

**Impact:**
- No persistent record of which tenants are inactive
- Cannot track how long tenants have been flagged as inactive
- Central admins must manually track inactive tenants
- Function name is misleading - suggests database update but only logs
- No audit trail for tenant inactivity

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/cron/tenant_activity.go`
- Function: `CheckAndFlagInactiveTenants()`
- Lines: 96-175

**Steps to Reproduce:**
1. Create tenant with no activity for 31+ days (beyond inactivity threshold)
2. Run `CheckAndFlagInactiveTenants()` cron job
3. Check database - verify NO field in `tenants` table is updated
4. Check logs - see "Tenant 'xyz' flagged as inactive" message
5. Query tenant record - no `inactive_flagged_at` or `is_inactive` field set
6. Result: Inconsistency between function name and actual behavior

**Expected Behavior:**
Function should UPDATE the tenants table to set an `inactive_flagged_at` timestamp or `is_inactive` boolean flag.

**Fix:**
Add database update to actually flag tenants:

```diff
		// Check if tenant is inactive
		isInactive := mostRecentActivity == nil || mostRecentActivity.Before(cutoffDate)

		if isInactive {
			inactiveCount++
			log.Printf("Tenant '%s' (ID: %d) flagged as inactive - last activity: %v",
				slug, tenantID, mostRecentActivity)
+
+			// Actually flag the tenant in the database
+			updateQuery := `
+				UPDATE tenants
+				SET inactive_flagged_at = ?, updated_at = ?
+				WHERE id = ?
+			`
+			now := time.Now()
+			if _, err := c.db.Exec(updateQuery, now, now, tenantID); err != nil {
+				log.Printf("Error flagging tenant %d as inactive: %v", tenantID, err)
+				continue
+			}
		}
	}
```

Alternative solution: Rename function to `LogInactiveTenants()` if flagging is not desired.

**Note:** This also requires a database migration to add `inactive_flagged_at` column to `tenants` table if it doesn't exist.

---

## Bug #4: Goroutine Leak in runPeriodically When Stop Called Before First Execution

**Severity:** HIGH

**Description:**
In the `runPeriodically()` function, if `Stop()` is called immediately after `Start()`, the goroutine may be stuck executing the provided function `fn()` before entering the select statement. There's no way to cancel an in-progress function execution, causing a goroutine leak. The function runs immediately on line 108 (`fn()`), BEFORE the ticker and stopChan select loop begins. If this function is long-running and Stop() is called during its execution, the goroutine cannot be stopped until fn() completes.

**Impact:**
- Goroutine leak if Stop() called during initial function execution
- Server shutdown delays if cron functions are long-running
- Resource leak in testing scenarios with rapid start/stop cycles
- No timeout mechanism for hung cron functions

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/cron/cron.go`
- Function: `runPeriodically()`
- Lines: 106-123

**Steps to Reproduce:**
1. Create a cron function that takes 30 seconds to complete
2. Call `Start()` which launches `runPeriodically()` in goroutine
3. Immediately call `Stop()` after 1 second
4. Observe that goroutine continues executing initial `fn()` call
5. Goroutine doesn't exit until fn() completes (29 seconds later)
6. Result: Delayed shutdown, goroutine leak during rapid start/stop

**Expected Behavior:**
The function should check stopChan before executing fn(), or use context cancellation.

**Fix:**
Add context-based cancellation or check stopChan before initial execution:

```diff
-func (s *CronService) runPeriodically(name string, interval time.Duration, fn func()) {
+func (s *CronService) runPeriodically(name string, interval time.Duration, fn func(ctx context.Context)) {
+	ctx, cancel := context.WithCancel(context.Background())
+	defer cancel()
+
+	// Listen for stop signal in separate goroutine
+	go func() {
+		<-s.stopChan
+		cancel()
+	}()
+
-	// Run immediately on start
-	fn()
+	// Run immediately on start, but check if stopped first
+	select {
+	case <-ctx.Done():
+		log.Printf("Cron job '%s' stopped before initial execution", name)
+		return
+	default:
+		fn(ctx)
+	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Printf("Running cron job: %s", name)
-			fn()
+			fn(ctx)
		case <-s.stopChan:
			log.Printf("Stopped cron job: %s", name)
			return
		}
	}
}
```

This requires updating all cron functions to accept `context.Context` and check for cancellation.

---

## Bug #5: Date Comparison String Logic Breaks at Midnight in GetForReminders

**Severity:** MEDIUM

**Description:**
The `GetForReminders()` function calculates reminder windows using string-based time comparison. When the current time is near midnight (23:00-23:59), adding 1-2 hours crosses into the next day. The query compares `b.date = ?` (current date) with `b.scheduled_time >= ?` (which may be 00:xx or 01:xx the NEXT day), causing a mismatch. Bookings scheduled for early morning hours (00:00-02:00) will NEVER receive reminders if checked during the previous evening.

**Impact:**
- Bookings scheduled for 00:00-02:00 never receive reminders
- Affects late-night workers or early morning dog walks
- Silent failure - no error logged, reminders just don't send
- Window: Every day from 23:00-23:59, reminders for next day fail

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/repository/booking_repository.go`
- Function: `GetForReminders()`
- Lines: 400-450

**Steps to Reproduce:**
1. Set server time to 23:30 (11:30 PM) on 2025-12-26
2. Create booking for 01:00 (1:00 AM) on 2025-12-27 (90 minutes away)
3. Run `GetForReminders()` at 23:30
4. Observe that:
   - `currentDate = "2025-12-26"` (today)
   - `oneHourTime = "00:30"` (tomorrow)
   - `twoHoursTime = "01:30"` (tomorrow)
   - Query: `WHERE b.date = '2025-12-26' AND b.scheduled_time >= '00:30'`
5. Result: Booking with date='2025-12-27' is NOT matched (date mismatch)
6. Reminder is never sent

**Expected Behavior:**
Function should handle midnight crossings by checking BOTH today and tomorrow's dates.

**Fix:**
Use proper datetime calculations instead of string-based time comparison:

```diff
func (r *BookingRepository) GetForReminders() ([]*models.Booking, error) {
-	// Get bookings scheduled within the next 1-2 hours
-	now := time.Now()
-	oneHourFromNow := now.Add(1 * time.Hour)
-	twoHoursFromNow := now.Add(2 * time.Hour)
-
-	currentDate := now.Format("2006-01-02")
-	oneHourTime := oneHourFromNow.Format("15:04")
-	twoHoursTime := twoHoursFromNow.Format("15:04")
-
-	// Query with user and dog details, excluding already-sent reminders
-	query := `
-		SELECT b.id, b.tenant_id, b.user_id, b.dog_id, b.date, b.scheduled_time, b.status,
-		       b.completed_at, b.user_notes, b.admin_cancellation_reason, b.created_at, b.updated_at,
-		       u.first_name as user_first_name, u.last_name as user_last_name, u.email as user_email,
-		       d.name as dog_name
-		FROM bookings b
-		LEFT JOIN users u ON b.user_id = u.id
-		LEFT JOIN dogs d ON b.dog_id = d.id
-		WHERE b.status = 'scheduled'
-		AND b.reminder_sent_at IS NULL
-		AND b.date = ?
-		AND b.scheduled_time >= ?
-		AND b.scheduled_time < ?
-	`
-
-	rows, err := r.db.Query(query, currentDate, oneHourTime, twoHoursTime)
+	// Get bookings scheduled within the next 1-2 hours (handle midnight crossing)
+	berlinLoc, _ := time.LoadLocation("Europe/Berlin")
+	now := time.Now().In(berlinLoc)
+	oneHourFromNow := now.Add(1 * time.Hour)
+	twoHoursFromNow := now.Add(2 * time.Hour)
+
+	// Check if window crosses midnight
+	currentDate := now.Format("2006-01-02")
+	nextDate := now.Add(24 * time.Hour).Format("2006-01-02")
+
+	oneHourTime := oneHourFromNow.Format("15:04")
+	twoHoursTime := twoHoursFromNow.Format("15:04")
+
+	// Query that handles midnight crossing
+	query := `
+		SELECT b.id, b.tenant_id, b.user_id, b.dog_id, b.date, b.scheduled_time, b.status,
+		       b.completed_at, b.user_notes, b.admin_cancellation_reason, b.created_at, b.updated_at,
+		       u.first_name as user_first_name, u.last_name as user_last_name, u.email as user_email,
+		       d.name as dog_name
+		FROM bookings b
+		LEFT JOIN users u ON b.user_id = u.id
+		LEFT JOIN dogs d ON b.dog_id = d.id
+		WHERE b.status = 'scheduled'
+		AND b.reminder_sent_at IS NULL
+		AND (
+			(b.date = ? AND b.scheduled_time >= ? AND b.scheduled_time < ?)
+			OR (b.date = ? AND b.scheduled_time >= '00:00' AND b.scheduled_time < ?)
+		)
+	`
+
+	rows, err := r.db.Query(query, currentDate, oneHourTime, twoHoursTime, nextDate, twoHoursTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query bookings for reminders: %w", err)
	}
	defer rows.Close()
	// ... rest of function
}
```

This ensures bookings in the early morning hours receive reminders when checked late at night.

---

## Bug #6: Potential SQL Injection in Sub-Queries (Activity Checker)

**Severity:** MEDIUM

**Description:**
In `CheckAndFlagInactiveTenants()`, the function executes sub-queries using `QueryRow()` with tenant IDs obtained from the main query. While tenant IDs are integers (type-safe), the queries don't use consistent error checking. The `Scan()` calls ignore errors, which means if a query fails (e.g., due to SQL injection via a compromised tenant ID, or database issues), the function silently continues with nil values, potentially incorrectly flagging tenants as inactive.

**Impact:**
- Silent failures when sub-queries fail
- Tenants may be incorrectly flagged as inactive due to query errors
- No visibility into sub-query failures in logs
- Reduced reliability of tenant activity tracking

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/cron/tenant_activity.go`
- Function: `CheckAndFlagInactiveTenants()`
- Lines: 126-142

**Steps to Reproduce:**
1. Simulate a database error for sub-query (e.g., connection timeout)
2. Run `CheckAndFlagInactiveTenants()`
3. Observe that `QueryRow().Scan()` fails silently
4. `lastBooking` and `lastActivity` remain nil
5. Tenant is flagged as inactive due to nil values (mostRecentActivity == nil)
6. Result: False positive - active tenant flagged as inactive due to query error

**Expected Behavior:**
Sub-query errors should be logged and handled appropriately.

**Fix:**
Add error checking for sub-queries:

```diff
		// Check last booking date for this tenant
		var lastBooking *time.Time
		bookingQuery := `
			SELECT MAX(created_at)
			FROM bookings
			WHERE tenant_id = ?
		`
-		c.db.QueryRow(bookingQuery, tenantID).Scan(&lastBooking)
+		if err := c.db.QueryRow(bookingQuery, tenantID).Scan(&lastBooking); err != nil && err != sql.ErrNoRows {
+			log.Printf("Error querying last booking for tenant %d: %v", tenantID, err)
+			// Continue with nil lastBooking - tenant may have no bookings
+		}

		// Check last user activity for this tenant
		var lastActivity *time.Time
		activityQuery := `
			SELECT MAX(last_activity_at)
			FROM users
			WHERE tenant_id = ? AND is_active = 1
		`
-		c.db.QueryRow(activityQuery, tenantID).Scan(&lastActivity)
+		if err := c.db.QueryRow(activityQuery, tenantID).Scan(&lastActivity); err != nil && err != sql.ErrNoRows {
+			log.Printf("Error querying last activity for tenant %d: %v", tenantID, err)
+			// Continue with nil lastActivity - tenant may have no users
+		}
```

This prevents silent failures and provides visibility into query issues.

---

## Bug #7: Inconsistent Tenant Filtering in autoDeactivateInactiveUsers

**Severity:** MEDIUM

**Description:**
The `autoDeactivateInactiveUsers()` function retrieves tenants using `FindAll("active")`, which filters for `status = 'active'`. However, it does NOT filter out demo tenants (where `is_demo = 1`). Demo tenants should NOT have automatic user deactivation, as they are reset daily. This means demo tenant users could be auto-deactivated between daily resets, breaking the demo experience. Users logging into demo tenants may find their accounts deactivated before the next reset.

**Impact:**
- Demo tenant users get auto-deactivated (breaks demo functionality)
- Demo experience is inconsistent - accounts randomly deactivated
- Next demo reset re-creates users, but bookings/data may be lost
- Confusing for users trying out the demo (unexpected deactivation)

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/cron/cron.go`
- Function: `autoDeactivateInactiveUsers()`
- Lines: 241-258

**Steps to Reproduce:**
1. Create demo tenant with `is_demo = 1`
2. Create test user with `last_activity_at` 400 days ago
3. Wait for auto-deactivation cron to run (or trigger manually)
4. Observe that demo tenant users are deactivated
5. Demo tenant users cannot login until next daily reset
6. Result: Broken demo experience

**Expected Behavior:**
Demo tenants should be excluded from auto-deactivation processing.

**Fix:**
Filter out demo tenants when retrieving active tenants:

```diff
func (s *CronService) autoDeactivateInactiveUsers() {
-	// Get all active tenants
-	tenants, err := s.tenantRepo.FindAll("active")
+	// Get all active non-demo tenants (demo tenants are reset daily)
+	tenants, err := s.tenantRepo.FindAllNonDemo("active")
	if err != nil {
		log.Printf("Error getting tenants for auto-deactivation: %v", err)
		return
	}

	if len(tenants) == 0 {
		log.Println("No active tenants found for auto-deactivation")
		return
	}

	// Process each tenant
	for _, tenant := range tenants {
+		// Double-check to skip demo tenants
+		if tenant.IsDemo {
+			log.Printf("Skipping demo tenant %d (%s) for auto-deactivation", tenant.ID, tenant.Slug)
+			continue
+		}
		s.autoDeactivateUsersForTenant(tenant.ID)
	}
}
```

This requires adding a `FindAllNonDemo()` method to `TenantRepository`:

```go
func (r *TenantRepository) FindAllNonDemo(status string) ([]*models.Tenant, error) {
	query := `
		SELECT id, slug, name, contact_email, status, is_demo, created_at, updated_at
		FROM tenants
		WHERE status = ? AND is_demo = 0
	`
	// ... implementation
}
```

---

## Bug #8: Missing Timezone Conversion in runDaily Scheduling Logic

**Severity:** LOW

**Description:**
While `runDaily()` correctly uses Europe/Berlin timezone for scheduling, the initial immediate execution (line 212: `fn()`) runs BEFORE timezone conversion. This means the first run uses server local time context, while subsequent scheduled runs use Europe/Berlin time context. This inconsistency could cause the first execution to behave differently than subsequent executions, especially for functions that rely on time-based calculations.

**Impact:**
- First execution may use different timezone than subsequent runs
- Inconsistent behavior between initial run and scheduled runs
- Difficult to debug timezone-related issues (works after restart, fails initially)
- Edge case: If server timezone differs significantly from Berlin, first run timing is wrong

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/cron/cron.go`
- Function: `runDaily()`
- Lines: 209-237

**Steps to Reproduce:**
1. Deploy server in US Eastern timezone (UTC-5)
2. Schedule daily job for 3:00 AM Berlin time
3. Server starts at 10:00 PM Eastern (4:00 AM Berlin time, past scheduled time)
4. Initial `fn()` executes immediately using Eastern timezone context
5. Next run scheduled for 3:00 AM Berlin time (9:00 PM Eastern)
6. Result: Timing inconsistency between first and subsequent runs

**Expected Behavior:**
Initial execution should use the same timezone context as scheduled runs.

**Fix:**
Pass timezone context to the function or ensure consistent timezone handling:

```diff
func (s *CronService) runDaily(name string, hour, minute int, fn func()) {
+	berlinLoc := getBerlinLocation()
+
	// Run immediately on startup
	log.Printf("Running daily job on startup: %s", name)
	fn()

-	berlinLoc := getBerlinLocation()

	for {
		now := time.Now().In(berlinLoc)
		next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, berlinLoc)

		// If we've passed today's scheduled time, schedule for tomorrow
		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}

		duration := next.Sub(now)
		log.Printf("Scheduling daily job '%s' to run in %v (at %s)", name, duration, next.Format("2006-01-02 15:04:05"))

		select {
		case <-time.After(duration):
			log.Printf("Running daily job: %s", name)
			fn()
		case <-s.stopChan:
			log.Printf("Stopped daily job: %s", name)
			return
		}
	}
}
```

Better approach: Pass timezone-aware time to functions:

```diff
-func (s *CronService) runDaily(name string, hour, minute int, fn func()) {
+func (s *CronService) runDaily(name string, hour, minute int, fn func(time.Time)) {
	berlinLoc := getBerlinLocation()
+	now := time.Now().In(berlinLoc)

	// Run immediately on startup
	log.Printf("Running daily job on startup: %s", name)
-	fn()
+	fn(now)

	for {
		now := time.Now().In(berlinLoc)
		next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, berlinLoc)

		// If we've passed today's scheduled time, schedule for tomorrow
		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}

		duration := next.Sub(now)
		log.Printf("Scheduling daily job '%s' to run in %v (at %s)", name, duration, next.Format("2006-01-02 15:04:05"))

		select {
		case <-time.After(duration):
			log.Printf("Running daily job: %s", name)
-			fn()
+			fn(time.Now().In(berlinLoc))
		case <-s.stopChan:
			log.Printf("Stopped daily job: %s", name)
			return
		}
	}
}
```

This ensures all executions use Europe/Berlin timezone context.

---

## Statistics

- **Critical:** 2 bugs (timezone issues)
- **High:** 3 bugs (missing error checks, goroutine leaks, misleading function)
- **Medium:** 3 bugs (midnight crossing, SQL sub-query errors, demo tenant filtering)
- **Low:** 1 bug (timezone inconsistency)

**Total:** 9 bugs

---

## Recommendations

### Immediate Actions (Critical/High Priority)

1. **Standardize Timezone Handling:** Implement consistent Europe/Berlin timezone usage across ALL cron operations. Create a `GetBerlinTime()` helper that returns `time.Now().In(berlinLoc)` and use it everywhere instead of `time.Now()`.

2. **Add rows.Err() Checks:** Audit ALL database row iterations in the cron package and add `rows.Err()` checks after loops. This is a Go best practice violation that must be fixed.

3. **Fix CheckAndFlagInactiveTenants:** Either implement actual database flagging (add `inactive_flagged_at` column and UPDATE query) OR rename to `LogInactiveTenants()` to match behavior.

4. **Implement Context-Based Cancellation:** Refactor cron functions to accept `context.Context` for graceful shutdown. Use `context.WithTimeout()` for long-running operations to prevent hangs.

### Medium Priority

5. **Fix Midnight Crossing Bug:** Refactor `GetForReminders()` to use proper datetime calculations. Consider storing bookings with full timestamp (date + time) instead of separate fields to simplify queries.

6. **Add Sub-Query Error Handling:** Log all sub-query failures in tenant activity checker. Consider using transactions for consistency.

7. **Exclude Demo Tenants:** Add `is_demo` filter to tenant queries in auto-deactivation. Document which cron jobs apply to demo tenants.

### Code Quality Improvements

8. **Add Integration Tests:** The existing test files are excellent, but add integration tests that verify:
   - Timezone handling across midnight
   - Error propagation from database failures
   - Goroutine cleanup on Stop()
   - Demo tenant exclusion

9. **Add Cron Job Monitoring:** Implement metrics/logging for:
   - Execution duration of each cron job
   - Number of records processed
   - Error counts
   - Last successful run timestamp

10. **Document Timezone Decisions:** Add documentation explaining why Europe/Berlin is used (business requirement: German animal shelters) and the implications for multi-region deployments.

### Architectural Considerations

11. **Consider Cron Library:** Evaluate using a production-grade cron library like `github.com/robfig/cron` which handles timezone, error recovery, and graceful shutdown out of the box.

12. **Separate Scheduling from Execution:** Consider separating the scheduling logic (when to run) from business logic (what to run). This makes testing easier and allows schedule changes without code changes.

13. **Add Circuit Breaker:** For email sending in reminders, implement a circuit breaker to prevent repeated failures from blocking the cron job.

### Testing Recommendations

14. **Timezone Tests:** Add tests that explicitly set different server timezones (UTC, US Eastern, Asia/Tokyo) and verify Europe/Berlin handling works correctly.

15. **Midnight Tests:** Add tests for bookings scheduled at 23:00-01:00 to verify reminder and completion logic across midnight boundary.

16. **Error Injection Tests:** Mock database errors during row iteration to verify `rows.Err()` handling.

---

## Additional Notes

### Design Patterns Observed

The cron package follows a service-oriented design with clear separation of concerns:
- `CronService` orchestrates scheduling
- Repository layer handles database operations
- Service layer (EmailService) handles external integrations

This is good architecture, but timezone handling needs to be more explicit at boundaries.

### Positive Aspects

1. Comprehensive test coverage exists (cron_test.go has 668 lines)
2. Graceful error handling in most places (logs errors, continues processing)
3. Demo tenant reset functionality is well-designed
4. Tenant-specific settings for deactivation days is flexible

### Critical User Impact

The timezone bugs (Bug #1, #5) are the most critical as they directly affect user experience:
- **Reminders not sent:** Users miss their dog walk appointments
- **Late completions:** Users can't add notes promptly after walks
- **Midnight edge case:** Predictable daily failures for early morning bookings

These should be fixed immediately before any production deployment handling European time zones.

---

**Report Generated:** 2025-12-27
**Analyst:** Directory Bug Finder Agent
**Files:** `/home/tranmh/work/gassigeher-saas/internal/cron/*.go`
