# Bug Report: internal/cron

**Analysis Date:** 2025-12-01
**Directory Analyzed:** `internal/cron`
**Files Analyzed:** 2 files
**Bugs Found:** 6 bugs

---

## Summary

The cron service contains **6 functional bugs** affecting scheduled jobs for auto-completion of bookings and auto-deactivation of users. Critical issues include:

- **Missing email notifications** for auto-deactivated users (breaks user communication)
- **Race condition** when stopping cron service (potential goroutine leak)
- **Unsafe concurrent execution** of auto-complete job (no protection against overlapping runs)
- **Missing transaction boundaries** in multi-user deactivation (partial failure risk)
- **Logic error** in daily job scheduling (misses runs after Stop() call)
- **Incomplete resource cleanup** when stopping periodic jobs

**Severity Distribution:**
- **Critical:** 1 bug (missing email notifications)
- **High:** 3 bugs (race condition, unsafe execution, transaction safety)
- **Medium:** 2 bugs (scheduling logic, resource cleanup)

---

## Bugs

## Bug #1: Missing Email Notifications for Auto-Deactivated Users

**STATUS: CODE MODIFIED - NEEDS REVERIFICATION**

**Description:**
The `autoDeactivateInactiveUsers()` function deactivates users but **never sends email notifications** to inform them of the deactivation. This is a critical user experience and compliance issue. Users have no way to know they were auto-deactivated and won't understand why they can't log in.

The handler `user_handler.go` line 425 sends emails for manual admin deactivations via `SendAccountDeactivated()`, but the cron job does not have access to the email service and doesn't call it.

**Location:**
- File: `/home/jaco/Git-clones/gassigeher/internal/cron/cron.go`
- Function: `autoDeactivateInactiveUsers`
- Lines: 227-235 (UPDATED - was 162-170)

**What Changed:**
- EmailService is now part of CronService struct (line 20)
- NewCronService now accepts cfg *config.Config and initializes emailService (lines 25-34)
- However, the autoDeactivateInactiveUsers() function STILL does not send email notifications (lines 227-235)
- The email service is available but not being used for deactivation notifications

**Steps to Reproduce:**
1. Create a user with `last_activity_at` older than 365 days
2. Wait for daily cron job to run (or manually call `autoDeactivateInactiveUsers()`)
3. User is deactivated in database
4. Expected: User receives email notification about deactivation
5. Actual: No email is sent, user is unaware of deactivation

**Fix:**
Add email notification call after successful deactivation (after line 234):

```diff
// Deactivate each user
for _, user := range users {
	if err := s.userRepo.Deactivate(user.ID, "auto_inactivity"); err != nil {
		log.Printf("Error deactivating user %d: %v", user.ID, err)
		continue
	}

	log.Printf("Auto-deactivated user %d (inactive for %d days)", user.ID, days)

+	// Send email notification
+	if s.emailService != nil && user.Email != nil {
+		reason := fmt.Sprintf("Keine Aktivität seit %d Tagen", days)
+		go s.emailService.SendAccountDeactivated(*user.Email, user.Name, reason)
+	}
}
```

---

## Bug #2: Race Condition on stopChan When Stopping Service

**Description:**
The `Stop()` method closes `stopChan` without any synchronization, while multiple goroutines (periodic and daily jobs) are reading from this channel. This creates a race condition where:

1. Multiple goroutines may receive from closed channel simultaneously
2. No guarantee that all goroutines have actually stopped before `Stop()` returns
3. Potential for goroutine leaks if Stop() is called multiple times

Closing a channel that's already closed causes a panic. If `Stop()` is called twice, the application crashes.

**Location:**
- File: `/home/jaco/Git-clones/gassigeher/internal/cron/cron.go`
- Function: `Stop`
- Lines: 61-64 (UPDATED - was 48-51)

**Steps to Reproduce:**
1. Start the cron service via `cronService.Start()`
2. Call `cronService.Stop()` twice in quick succession
3. Expected: Service stops gracefully both times
4. Actual: Second call panics with "close of closed channel"

**Fix:**
Use sync.Once to ensure stopChan is closed only once, and add WaitGroup to ensure all goroutines have stopped:

```diff
type CronService struct {
	db           *sql.DB
	bookingRepo  *repository.BookingRepository
	userRepo     *repository.UserRepository
	settingsRepo *repository.SettingsRepository
+	emailService *services.EmailService
	stopChan     chan bool
+	wg           sync.WaitGroup
+	stopOnce     sync.Once
}

func NewCronService(db *sql.DB, cfg *config.Config) *CronService {
	// Initialize email service for reminders (fail gracefully if not configured)
	var emailService *services.EmailService
	if cfg != nil {
		var err error
		emailService, err = services.NewEmailService(services.ConfigToEmailConfig(cfg))
		if err != nil {
			log.Printf("Warning: Email service not available for cron jobs: %v", err)
		}
	}

	return &CronService{
		db:           db,
		bookingRepo:  repository.NewBookingRepository(db),
		userRepo:     repository.NewUserRepository(db),
		settingsRepo: repository.NewSettingsRepository(db),
		emailService: emailService,
		stopChan:     make(chan bool),
	}
}

func (s *CronService) Start() {
	log.Println("Starting cron service...")

+	s.wg.Add(3)
	// Run auto-complete job every 15 minutes
-	go s.runPeriodically("Auto-complete bookings", 15*time.Minute, s.autoCompleteBookings)
+	go func() {
+		defer s.wg.Done()
+		s.runPeriodically("Auto-complete bookings", 15*time.Minute, s.autoCompleteBookings)
+	}()

	// Run auto-deactivation job daily at 3am
-	go s.runDaily("Auto-deactivate inactive users", 3, 0, s.autoDeactivateInactiveUsers)
+	go func() {
+		defer s.wg.Done()
+		s.runDaily("Auto-deactivate inactive users", 3, 0, s.autoDeactivateInactiveUsers)
+	}()

	// Run booking reminder job every 15 minutes
-	go s.runPeriodically("Send booking reminders", 15*time.Minute, s.sendBookingReminders)
+	go func() {
+		defer s.wg.Done()
+		s.runPeriodically("Send booking reminders", 15*time.Minute, s.sendBookingReminders)
+	}()
}

func (s *CronService) Stop() {
	log.Println("Stopping cron service...")
-	close(s.stopChan)
+	s.stopOnce.Do(func() {
+		close(s.stopChan)
+	})
+	s.wg.Wait() // Wait for all goroutines to finish
+	log.Println("Cron service stopped")
}
```

---

## Bug #3: Unsafe Concurrent Execution of Auto-Complete Job

**Description:**
The `autoCompleteBookings()` function runs every 15 minutes (changed from 1 hour) with no protection against overlapping executions. If the database query or update takes longer than expected (due to high load, slow disk, or large result set), the next scheduled run will start before the previous one finishes.

This can cause:
- **Duplicate processing** of the same bookings
- **Race conditions** on the same database rows
- **Wasted resources** processing the same data twice
- **Inconsistent completed_at timestamps** if two runs update the same booking

The same issue exists for `autoDeactivateInactiveUsers()` and `sendBookingReminders()` but is less likely since one runs daily and the other processes smaller datasets.

**Location:**
- File: `/home/jaco/Git-clones/gassigeher/internal/cron/cron.go`
- Function: `runPeriodically`
- Lines: 67-84 (UPDATED - was 54-70)

**Steps to Reproduce:**
1. Insert 10,000 bookings scheduled for completion
2. Slow down database with `time.Sleep(2 * time.Hour)` in AutoComplete()
3. Start cron service with 15-minute interval
4. Expected: Second run waits for first to complete
5. Actual: Both runs execute simultaneously, processing same bookings

**Fix:**
Add mutex protection to prevent concurrent execution of the same job:

```diff
type CronService struct {
	db           *sql.DB
	bookingRepo  *repository.BookingRepository
	userRepo     *repository.UserRepository
	settingsRepo *repository.SettingsRepository
	emailService *services.EmailService
	stopChan     chan bool
+	jobMutex     sync.Mutex
}

func (s *CronService) runPeriodically(name string, interval time.Duration, fn func()) {
	// Run immediately on start
+	s.runJobSafely(name, fn)
-	fn()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Printf("Running cron job: %s", name)
+			s.runJobSafely(name, fn)
-			fn()
		case <-s.stopChan:
			log.Printf("Stopped cron job: %s", name)
			return
		}
	}
}

+// runJobSafely ensures only one instance of a job runs at a time
+func (s *CronService) runJobSafely(name string, fn func()) {
+	if !s.jobMutex.TryLock() {
+		log.Printf("Skipping cron job '%s' - previous execution still running", name)
+		return
+	}
+	defer s.jobMutex.Unlock()
+
+	fn()
+}
```

Note: This uses a single mutex for all jobs. For true per-job locking, use `map[string]*sync.Mutex`.

---

## Bug #4: Missing Transaction Boundaries in Batch User Deactivation

**Description:**
The `autoDeactivateInactiveUsers()` function deactivates multiple users in a loop (lines 228-235) without using database transactions. If the process crashes or is interrupted partway through (e.g., server shutdown during iteration), some users will be deactivated while others won't, despite all meeting the same criteria.

This violates the principle of atomic batch operations and can lead to:
- **Inconsistent state**: Some users deactivated, others not
- **Unfair treatment**: Users with IDs processed first get deactivated, others don't
- **Difficult recovery**: No way to know which users should have been deactivated

The error handling uses `continue`, which is correct for skipping individual failures, but there's no transaction rollback on system-level failures.

**Location:**
- File: `/home/jaco/Git-clones/gassigeher/internal/cron/cron.go`
- Function: `autoDeactivateInactiveUsers`
- Lines: 228-235 (UPDATED - was 163-170)

**Steps to Reproduce:**
1. Create 100 inactive users meeting deactivation criteria
2. Start deactivation cron job
3. Kill server process (SIGKILL) after 50 users are processed
4. Expected: Either all 100 users deactivated, or none
5. Actual: 50 users deactivated, 50 remain active despite being equally inactive

**Fix:**
Use a database transaction to ensure atomic batch processing. Allow individual user failures but commit the transaction only if no critical errors occur:

```diff
func (s *CronService) autoDeactivateInactiveUsers() {
	// Get deactivation period from settings
	setting, err := s.settingsRepo.Get("auto_deactivation_days")
	if err != nil {
		log.Printf("Error getting auto_deactivation_days setting: %v", err)
		return
	}

	days := 365 // default 1 year
	if setting != nil {
		if d, err := strconv.Atoi(setting.Value); err == nil {
			days = d
		}
	}

	// Find inactive users
	users, err := s.userRepo.FindInactiveUsers(days)
	if err != nil {
		log.Printf("Error finding inactive users: %v", err)
		return
	}

	if len(users) == 0 {
		log.Println("No inactive users to deactivate")
		return
	}

	log.Printf("Found %d inactive user(s) to deactivate", len(users))

+	// Start transaction for batch operation
+	tx, err := s.db.Begin()
+	if err != nil {
+		log.Printf("Error starting transaction for user deactivation: %v", err)
+		return
+	}
+	defer tx.Rollback() // Rollback if not committed

+	successCount := 0
+	var deactivationErrors []error

	// Deactivate each user
	for _, user := range users {
-		if err := s.userRepo.Deactivate(user.ID, "auto_inactivity"); err != nil {
+		// Use transaction-aware deactivation
+		query := `
+			UPDATE users SET
+				is_active = 0,
+				deactivated_at = ?,
+				deactivation_reason = ?,
+				updated_at = ?
+			WHERE id = ?
+		`
+		now := time.Now()
+		_, err := tx.Exec(query, now, "auto_inactivity", now, user.ID)
+		if err != nil {
			log.Printf("Error deactivating user %d: %v", user.ID, err)
+			deactivationErrors = append(deactivationErrors, err)
			continue
		}

+		successCount++
		log.Printf("Auto-deactivated user %d (inactive for %d days)", user.ID, days)
	}

+	// Commit transaction if at least some users were processed
+	if successCount > 0 {
+		if err := tx.Commit(); err != nil {
+			log.Printf("Error committing deactivation transaction: %v", err)
+			return
+		}
+		log.Printf("Successfully deactivated %d/%d users", successCount, len(users))
+	} else {
+		log.Printf("No users were successfully deactivated (%d errors)", len(deactivationErrors))
+	}
}
```

**Alternative approach**: Keep per-user transactions for better isolation, but add proper recovery logging.

---

## Bug #5: Incorrect Daily Job Scheduling Logic After Stop()

**Description:**
The `runDaily()` function has a logic error in its infinite loop structure (lines 169-195). The loop blocks on `time.After(duration)` or `stopChan` indefinitely. Once `stopChan` is closed and the function returns, there's no way to resume it.

More critically, after the scheduled time arrives and the job runs (line 189), the loop **immediately restarts** and calculates the next run time. However, there's no check if `stopChan` was closed during job execution. If `Stop()` is called while the job is running, it won't be detected until the next 24-hour wait begins.

This means:
- Job can run for hours after `Stop()` is called
- No graceful interruption of long-running jobs
- User expects "stop" to mean immediate termination

**Location:**
- File: `/home/jaco/Git-clones/gassigeher/internal/cron/cron.go`
- Function: `runDaily`
- Lines: 169-195 (UPDATED - was 107-129)

**Steps to Reproduce:**
1. Start cron service
2. Manually trigger `autoDeactivateInactiveUsers()` with slow deactivation logic (10 seconds per user)
3. Call `Stop()` during execution
4. Expected: Job stops within seconds
5. Actual: Job continues running until completion, then stops

**Fix:**
Add context-based cancellation to allow graceful shutdown during job execution:

```diff
func (s *CronService) runDaily(name string, hour, minute int, fn func()) {
+	ctx, cancel := context.WithCancel(context.Background())
+	defer cancel()
+
+	// Listen for stop signal in separate goroutine
+	go func() {
+		<-s.stopChan
+		cancel() // Cancel context when stop is requested
+	}()

	// Run immediately on startup
	log.Printf("Running daily job on startup: %s", name)
	fn()

	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())

		// If we've passed today's scheduled time, schedule for tomorrow
		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}

		duration := next.Sub(now)
		log.Printf("Scheduling daily job '%s' to run in %v (at %s)", name, duration, next.Format("2006-01-02 15:04:05"))

		select {
-		case <-time.After(duration):
+		case <-time.After(duration):
			log.Printf("Running daily job: %s", name)
+
+			// Check if stop was requested before running
+			select {
+			case <-ctx.Done():
+				log.Printf("Stopped daily job before execution: %s", name)
+				return
+			default:
+			}
+
			fn()
+
+			// Check if stop was requested after running
+			select {
+			case <-ctx.Done():
+				log.Printf("Stopped daily job after execution: %s", name)
+				return
+			default:
+			}
		case <-s.stopChan:
			log.Printf("Stopped daily job: %s", name)
			return
		}
	}
}
```

**Note**: This doesn't interrupt the job function itself. For true cancellation, pass context to `fn(ctx context.Context)`.

---

## Bug #6: Resource Leak - Ticker Not Stopped in Periodic Jobs

**Description:**
The `runPeriodically()` function creates a ticker (line 71) and properly defers `ticker.Stop()` (line 72). However, if the goroutine running this function panics before reaching the defer statement, the ticker will never be stopped, causing a resource leak.

While Go's defer mechanism usually protects against this, there's a subtle issue: if multiple periodic jobs are started and one panics during initialization (before the defer), its ticker continues running forever, sending events to a channel that's no longer being read.

Additionally, the ticker continues running even during job execution (lines 78, 69), which means if a job takes 2 hours and the interval is 15 minutes, ticks accumulate in the channel buffer. This is acceptable but wasteful.

**Location:**
- File: `/home/jaco/Git-clones/gassigeher/internal/cron/cron.go`
- Function: `runPeriodically`
- Lines: 67-84 (UPDATED - was 54-70)

**Steps to Reproduce:**
1. Modify `autoCompleteBookings()` to panic on first call
2. Start cron service
3. Check goroutine count: `runtime.NumGoroutine()`
4. Expected: Goroutine exits, ticker stops
5. Actual: Goroutine exits, ticker continues running (detectable via increasing memory)

**Fix:**
Add panic recovery to ensure ticker cleanup and improve resource management:

```diff
func (s *CronService) runPeriodically(name string, interval time.Duration, fn func()) {
+	defer func() {
+		if r := recover(); r != nil {
+			log.Printf("PANIC in cron job '%s': %v", name, r)
+		}
+	}()
+
	// Run immediately on start
+	s.safeRunJob(name, fn)
-	fn()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Printf("Running cron job: %s", name)
+			s.safeRunJob(name, fn)
-			fn()
		case <-s.stopChan:
			log.Printf("Stopped cron job: %s", name)
			return
		}
	}
}

+// safeRunJob runs a job with panic recovery
+func (s *CronService) safeRunJob(name string, fn func()) {
+	defer func() {
+		if r := recover(); r != nil {
+			log.Printf("PANIC in job function '%s': %v", name, r)
+		}
+	}()
+	fn()
+}
```

This ensures that:
1. Panics in job functions don't crash the cron service
2. Ticker cleanup always happens via the outer defer
3. Errors are logged for debugging

---

## Statistics

- **Critical:** 1 bug (missing email notifications - #1)
- **High:** 3 bugs (race condition - #2, unsafe execution - #3, transaction safety - #4)
- **Medium:** 2 bugs (scheduling logic - #5, resource leak - #6)

---

## Recommendations

### Immediate Actions (Critical/High Priority)

1. **Complete email service integration** for cron service (Bug #1)
   - Email service is initialized but not used in autoDeactivateInactiveUsers()
   - Send notifications after auto-deactivation
   - Test email delivery for deactivation notices

2. **Fix race condition in Stop()** (Bug #2)
   - Add sync.Once for channel closing
   - Add WaitGroup for graceful shutdown
   - Test with multiple Stop() calls

3. **Add mutex protection for jobs** (Bug #3)
   - Prevent concurrent execution of same job
   - Consider per-job mutexes for better parallelism
   - Add logging for skipped runs

4. **Use transactions for batch operations** (Bug #4)
   - Wrap deactivation loop in transaction
   - Add proper error handling and rollback
   - Log success/failure counts

### Medium Priority Improvements

5. **Improve daily job cancellation** (Bug #5)
   - Add context-based cancellation
   - Allow jobs to respect stop signals during execution
   - Consider passing context to job functions

6. **Add panic recovery** (Bug #6)
   - Protect ticker cleanup with panic handlers
   - Log panics for debugging
   - Ensure cron service remains stable

### Code Quality Improvements

- **Add integration tests** for cron service Start/Stop lifecycle
- **Add metrics/monitoring** for job execution times and success rates
- **Consider using a cron library** (e.g., robfig/cron) for better scheduling
- **Add health check endpoint** to verify cron jobs are running
- **Implement job execution history** (last run time, duration, errors)
- **Add configuration for job intervals** (currently hardcoded)

### Documentation Needs

- Document expected behavior when Stop() is called during job execution
- Add examples of testing cron jobs in unit tests
- Document resource requirements (goroutines, memory) for cron service
- Add runbook for troubleshooting failed cron jobs

### Testing Gaps

Current tests (cron_test.go) only test:
- Individual job logic (autoCompleteBookings, autoDeactivateInactiveUsers)
- Service initialization

Missing tests for:
- Concurrent Stop() calls
- Long-running jobs with Stop() signal
- Job execution overlap scenarios
- Email notification sending
- Transaction rollback on failures
- Panic recovery
- Resource cleanup verification

---

## Related Files to Review

The following files interact with the cron service and may need updates:

1. **`cmd/server/main.go`** - Cron service initialization (line 122)
   - Already passes cfg to NewCronService (CORRECT)
   - Has proper shutdown handling with defer

2. **`internal/repository/booking_repository.go`** - AutoComplete method
   - Consider adding query timeout
   - Add index on (status, date, scheduled_time) for performance

3. **`internal/repository/user_repository.go`** - Deactivate and FindInactiveUsers methods
   - Make Deactivate transaction-aware (accept *sql.Tx)
   - Add index on (is_active, is_deleted, is_admin, last_activity_at)

4. **`internal/handlers/user_handler.go`** - Manual deactivation logic
   - Extract email notification logic to shared function
   - Ensure consistency between manual and auto deactivation

5. **`internal/services/email_service.go`** - Email service interface
   - Verify thread-safety for concurrent goroutine calls
   - Add retry logic for failed email sends in cron context

---

## Final Notes

The cron service is functionally working for its core operations (auto-complete, auto-deactivate, and reminders), but has several production-readiness issues:

- **Missing observability**: No metrics, no execution logs to database, no alerts
- **Limited error handling**: Errors are logged but not reported elsewhere
- **No idempotency guarantees**: Jobs assume they're the only instance running
- **Hardcoded configuration**: Intervals are hardcoded in code, not configurable

These bugs should be fixed before scaling the application beyond a single server instance. In a multi-server deployment, additional work is needed to ensure only one instance runs cron jobs (e.g., distributed locks).
