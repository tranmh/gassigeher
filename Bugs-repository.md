# Bug Report: internal/repository

**Analysis Date:** 2025-12-01
**Directory Analyzed:** `internal/repository`
**Files Analyzed:** 9 files
**Bugs Found:** 13 bugs

**Verification Date:** 2025-12-01
**Verification Status:** 12 bugs still present with accurate locations, 1 bug partially fixed (code modified)

---

## Summary

The repository layer contains several critical bugs related to error handling, race conditions, SQL logic errors, and potential security issues. The most severe issues include:

- Missing error checks that could lead to silent failures
- Unique constraint violation detection that only works for SQLite
- Missing NULL handling causing potential panics
- Incorrect SQL logic in time comparisons
- Missing row affinity checks leading to incorrect success responses
- Database-specific error handling breaking multi-database compatibility

**Severity Distribution:**
- Critical: 3 bugs (SQL injection potential, data integrity, multi-database compatibility)
- High: 6 bugs (error handling gaps, race conditions, logic errors)
- Medium: 4 bugs (minor logic inconsistencies, missing validations)

---

## Bugs

## Bug #1: Database-Specific Error String Matching Breaks Multi-Database Compatibility

**Description:**
In `blocked_date_repository.go`, the `Create` function checks for unique constraint violations by matching the exact SQLite error message string. This breaks the multi-database compatibility promise of the application, as MySQL and PostgreSQL return different error messages for unique constraint violations.

**Location:**
- File: `/home/jaco/Git-clones/gassigeher/internal/repository/blocked_date_repository.go`
- Function: `Create`
- Lines: 38-40

**Impact:**
- On MySQL/PostgreSQL, unique constraint violations will not be detected
- Users will receive generic "failed to create blocked date" error instead of "date is already blocked"
- Frontend cannot provide helpful error messages
- Breaks documented multi-database support

**Steps to Reproduce:**
1. Configure application to use MySQL or PostgreSQL
2. Create a blocked date for "2025-12-25"
3. Attempt to create another blocked date for "2025-12-25"
4. Expected: "date is already blocked" error
5. Actual: Generic "failed to create blocked date" error

**Fix:**
Use database driver error codes instead of string matching:

```diff
- if err.Error() == "UNIQUE constraint failed: blocked_dates.date" {
-     return fmt.Errorf("date is already blocked")
- }
+ // Check for unique constraint violation across all databases
+ import "github.com/mattn/go-sqlite3"
+ import "github.com/go-sql-driver/mysql"
+ import "github.com/lib/pq"
+
+ if sqliteErr, ok := err.(sqlite3.Error); ok && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
+     return fmt.Errorf("date is already blocked")
+ }
+ if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1062 {
+     return fmt.Errorf("date is already blocked")
+ }
+ if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
+     return fmt.Errorf("date is already blocked")
+ }
```

Alternatively, use a simpler string contains check that works across databases:

```diff
- if err.Error() == "UNIQUE constraint failed: blocked_dates.date" {
+ if strings.Contains(strings.ToLower(err.Error()), "unique") ||
+    strings.Contains(strings.ToLower(err.Error()), "duplicate") {
     return fmt.Errorf("date is already blocked")
 }
```

---

## Bug #2: Missing Error Check After LastInsertId() in Multiple Repositories

**STATUS: CODE PARTIALLY MODIFIED - NEEDS REVERIFICATION**

**Description:**
Several repository methods ignore the error returned from `LastInsertId()` after successful inserts. While rare, this error can occur if the database driver doesn't support returning the last insert ID, or if there's a connection issue immediately after the insert. Ignoring this error leads to objects with ID=0, causing subsequent operations to fail silently.

**Code Changes Detected:**
- `blocked_date_repository.go:44-47` - NOW PROPERLY CHECKS ERROR (FIXED)
- `booking_time_repository.go:124` - STILL UNCHECKED (BUG REMAINS)
- `holiday_repository.go:72` - STILL UNCHECKED (BUG REMAINS)

**Location:**
- File: `/home/jaco/Git-clones/gassigeher/internal/repository/booking_time_repository.go`
- Function: `CreateRule`
- Lines: 124

**Similar Issues:**
- `holiday_repository.go:72` - CreateHoliday
- Both still ignore error from `LastInsertId()`

**Impact:**
- Objects created with ID=0 cannot be referenced later
- Frontend receives invalid response with id=0
- Subsequent operations fail with "not found" errors
- Silent data corruption in client state

**Steps to Reproduce:**
1. Use a database driver that doesn't support LastInsertId in certain conditions
2. Create a new booking time rule
3. Response contains rule with id=0
4. Attempt to update/delete rule by ID
5. Operation fails with "not found"

**Fix:**
Check and handle the error from LastInsertId():

```diff
 result, err := r.db.Exec(query, rule.DayType, rule.RuleName, rule.StartTime, rule.EndTime, isBlocked)
 if err != nil {
     return err
 }

- id, _ := result.LastInsertId()
+ id, err := result.LastInsertId()
+ if err != nil {
+     return fmt.Errorf("failed to get rule ID: %w", err)
+ }
 rule.ID = int(id)
 return nil
```

Apply same fix to `holiday_repository.go:72`.

---

## Bug #3: Missing Rows Affected Check in ApproveBooking

**Description:**
In `booking_repository.go`, the `ApproveBooking` function ignores the error when checking `RowsAffected()` with the blank identifier. This means if the database driver returns an error checking rows affected, the function will silently report success even when the update might have failed.

**Location:**
- File: `/home/jaco/Git-clones/gassigeher/internal/repository/booking_repository.go`
- Function: `ApproveBooking`
- Lines: 626-629

**Similar Issues:**
- `RejectBooking` (line 652) has the same issue

**Impact:**
- Admin approves booking but approval doesn't persist
- Frontend shows success but booking remains pending
- Users never receive approval notification
- Booking remains in pending state indefinitely

**Steps to Reproduce:**
1. Create a booking requiring approval
2. Admin clicks approve
3. Database driver returns error on RowsAffected (rare but possible)
4. Expected: Error response to admin
5. Actual: Success response but booking still pending

**Fix:**
Properly check the error from RowsAffected:

```diff
 result, err := r.db.Exec(query, adminID, time.Now(), bookingID)
 if err != nil {
     return err
 }

- rows, _ := result.RowsAffected()
+ rows, err := result.RowsAffected()
+ if err != nil {
+     return fmt.Errorf("failed to check rows affected: %w", err)
+ }
 if rows == 0 {
     return fmt.Errorf("booking not found or not pending")
 }
```

Apply same fix to `RejectBooking` at line 652.

---

## Bug #4: CheckDoubleBooking Doesn't Check For Pending Approval Bookings

**Description:**
The `CheckDoubleBooking` function only checks for bookings with status='scheduled', but doesn't check for bookings with approval_status='pending'. This allows users to create multiple pending bookings for the same dog/date/time slot, even though only one should be allowed.

**Location:**
- File: `/home/jaco/Git-clones/gassigeher/internal/repository/booking_repository.go`
- Function: `CheckDoubleBooking`
- Lines: 228-242

**Impact:**
- Multiple users can request the same dog at the same time
- Admin must manually reject duplicate requests
- Poor user experience when approvals come back
- Race condition during approval process

**Steps to Reproduce:**
1. Enable booking approval for a dog
2. User A creates booking for Dog X on 2025-12-10 at 09:00 (status=scheduled, approval_status=pending)
3. User B creates booking for Dog X on 2025-12-10 at 09:00
4. Expected: "Dog is already booked for this time" error
5. Actual: Both bookings created, both pending approval

**Fix:**
Include pending approval bookings in the double-booking check:

```diff
 func (r *BookingRepository) CheckDoubleBooking(dogID int, date, scheduledTime string) (bool, error) {
     query := `
         SELECT COUNT(*)
         FROM bookings
-        WHERE dog_id = ? AND date = ? AND walk_type = ? AND status = 'scheduled'
+        WHERE dog_id = ? AND date = ? AND scheduled_time = ?
+          AND status != 'cancelled'
+          AND (approval_status = 'approved' OR approval_status = 'pending')
     `
```

---

## Bug #5: Race Condition in AutoComplete Time Comparison

**Description:**
The `AutoComplete` function reads the current time once at the beginning but uses it in a complex query with date and time comparisons. If the function runs exactly at midnight, bookings scheduled for 23:59 on the previous day might not be completed because the comparison is not atomic. Additionally, there's a subtle race condition where a booking at exactly the current time might not be marked completed.

**Location:**
- File: `/home/jaco/Git-clones/gassigeher/internal/repository/booking_repository.go`
- Function: `AutoComplete`
- Lines: 245-272

**Impact:**
- Bookings at midnight boundary may not auto-complete
- Bookings at exactly current time might be skipped
- Inconsistent completion state
- Users cannot add notes to past bookings

**Steps to Reproduce:**
1. Create booking for 2025-12-01 23:59
2. Run AutoComplete at exactly 2025-12-02 00:00:00
3. Expected: Booking marked complete
4. Actual: Might be skipped due to time comparison edge case

**Fix:**
Use <= for time comparison to include the current minute:

```diff
 func (r *BookingRepository) AutoComplete() (int, error) {
     // Get current date and time
     now := time.Now()
     currentDate := now.Format("2006-01-02")
     currentTime := now.Format("15:04")

     query := `
         UPDATE bookings
         SET status = 'completed', completed_at = ?, updated_at = ?
         WHERE status = 'scheduled'
         AND (
             date < ?
-            OR (date = ? AND scheduled_time < ?)
+            OR (date = ? AND scheduled_time <= ?)
         )
     `
```

This ensures bookings at exactly the current time are also completed.

---

## Bug #6: GetForReminders Has Incorrect Time Window Logic

**STATUS: CODE MODIFIED - NEEDS REVERIFICATION**

**Description:**
The `GetForReminders` function attempts to find bookings 1-2 hours in the future, but the query logic has been simplified. The bug report mentioned complex midnight-crossing logic, but the current code (lines 319-395) now uses a simple same-day check.

**Code Changes Detected:**
- Original complex logic with `nextDate`, `oneHourTime`, `twoHoursTime` has been replaced
- Current code at line 340 uses: `AND b.date = ?` (same-day only)
- Lines 326-327 still calculate `oneHourTime` and `twoHoursTime` but only for same-day comparison
- Midnight-crossing bookings are NO LONGER handled

**Location:**
- File: `/home/jaco/Git-clones/gassigeher/internal/repository/booking_repository.go`
- Function: `GetForReminders`
- Lines: 319-395

**New Impact:**
- Reminders NOT sent for bookings after midnight (e.g., current time 23:30, booking at 00:30)
- Only same-day bookings get reminders
- Time window logic simplified but loses cross-day functionality
- Users miss reminders for early morning bookings when running cron at night

**Current Code Pattern:**
```go
oneHourTime := oneHourFromNow.Format("15:04")
twoHoursTime := twoHoursFromNow.Format("15:04")

query := `
    ...
    WHERE b.status = 'scheduled'
    AND b.reminder_sent_at IS NULL
    AND b.date = ?
    AND b.scheduled_time >= ?
    AND b.scheduled_time < ?
`
```

**Steps to Reproduce:**
1. Current time is 23:30 today
2. Create booking for tomorrow at 00:30 (half hour after midnight)
3. Booking is 1 hour away but on different day
4. Expected: Booking appears in reminder list
5. Actual: Booking NOT in list because `b.date = currentDate` excludes next day

**Fix:**
Convert to timestamp comparison or fetch cross-day bookings:

```diff
 func (r *BookingRepository) GetForReminders() ([]*models.Booking, error) {
     now := time.Now()
     oneHourFromNow := now.Add(1 * time.Hour)
     twoHoursFromNow := now.Add(2 * time.Hour)

-    currentDate := now.Format("2006-01-02")
-    oneHourTime := oneHourFromNow.Format("15:04")
-    twoHoursTime := twoHoursFromNow.Format("15:04")
+    minDate := oneHourFromNow.Format("2006-01-02")
+    maxDate := twoHoursFromNow.Format("2006-01-02")

     query := `
         SELECT ...
         FROM bookings b
         WHERE b.status = 'scheduled'
           AND b.reminder_sent_at IS NULL
-          AND b.date = ?
-          AND b.scheduled_time >= ?
-          AND b.scheduled_time < ?
+          AND b.date >= ? AND b.date <= ?
     `

-    rows, err := r.db.Query(query, currentDate, oneHourTime, twoHoursTime)
+    rows, err := r.db.Query(query, minDate, maxDate)
+    // Then filter in Go code based on combined datetime
```

---

## Bug #7: Missing Error Check on rows.Close() After Query Errors

**Description:**
Throughout the repository files, when a query fails during row iteration (in the `for rows.Next()` loop), the code returns immediately without checking if there's an error from `rows.Err()`. Additionally, the deferred `rows.Close()` might panic if rows is nil due to query failure.

**Location:**
- File: `/home/jaco/Git-clones/gassigeher/internal/repository/booking_repository.go`
- Function: `FindAll` (and many others)
- Lines: 161-183

**Similar Issues:**
- Affects nearly all methods that query multiple rows across all repository files
- Over 20 instances

**Impact:**
- Query errors silently ignored
- Incomplete result sets returned as successful
- Database connection leaks if rows not properly closed
- Panics possible if rows is nil

**Steps to Reproduce:**
1. Database has network interruption during query
2. Query returns partial results then fails
3. Expected: Error returned to caller
4. Actual: Partial results returned as complete success

**Fix:**
Add error check after row iteration:

```diff
 bookings := []*models.Booking{}
 for rows.Next() {
     booking := &models.Booking{}
     err := rows.Scan(...)
     if err != nil {
         return nil, fmt.Errorf("failed to scan booking: %w", err)
     }
     bookings = append(bookings, booking)
 }
+
+ // Check for errors that occurred during iteration
+ if err := rows.Err(); err != nil {
+     return nil, fmt.Errorf("error during row iteration: %w", err)
+ }

 return bookings, nil
```

Apply this pattern to all methods that iterate over rows in all repository files.

---

## Bug #8: FindByIDWithDetails Returns Incomplete Data on JOIN Failure

**Description:**
The `FindByIDWithDetails` function uses LEFT JOINs to fetch user and dog details along with a booking. However, if the user or dog has been deleted (NULL results), the function still populates the booking's User and Dog pointers with empty struct instances. The calling code cannot distinguish between "user exists with empty fields" vs "user was deleted".

**Location:**
- File: `/home/jaco/Git-clones/gassigeher/internal/repository/booking_repository.go`
- Function: `FindByIDWithDetails`
- Lines: 430-504

**Impact:**
- Deleted users appear as users with "Deleted User" name
- Cannot distinguish between missing data and deleted entities
- May cause nil pointer dereferences if caller expects nil for deleted users
- Inconsistent with FindByID which doesn't populate joined data

**Steps to Reproduce:**
1. Create booking for user and dog
2. Delete user account (GDPR anonymization)
3. Call FindByIDWithDetails(bookingID)
4. Expected: booking.User = nil or clear indication of deletion
5. Actual: booking.User = &User{Name: "Deleted User", Email: nil, ...}

**Fix:**
Return nil for User/Dog if the JOIN returned no data:

```diff
 booking := &models.Booking{
-    User: &models.User{},
-    Dog:  &models.Dog{},
 }

 var userName, userEmail, userPhone sql.NullString
 var dogName, breed, size string
 var age int

 err := r.db.QueryRow(query, id).Scan(...)

 if err == sql.ErrNoRows {
     return nil, nil
 }

 if err != nil {
     return nil, fmt.Errorf("failed to find booking with details: %w", err)
 }

 // Populate user details
+ if userName.Valid {
+     booking.User = &models.User{}
-     if userName.Valid {
-         booking.User.Name = userName.String
-     } else {
-         booking.User.Name = "Deleted User"
-     }
+     booking.User.Name = userName.String
      if userEmail.Valid {
          email := userEmail.String
          booking.User.Email = &email
      }
      // ... other user fields
+ }

 // Populate dog details
+ if dogName != "" {
+     booking.Dog = &models.Dog{}
      booking.Dog.Name = dogName
      // ... other dog fields
+ }
```

---

## Bug #9: DogRepository.Delete Race Condition Between Check and Delete

**Description:**
The `Delete` function in `dog_repository.go` has a TOCTOU (Time-Of-Check-Time-Of-Use) race condition. It first checks if there are future bookings, then deletes the dog. Between these two operations, a new booking could be created, resulting in orphaned bookings that reference a deleted dog.

**Location:**
- File: `/home/jaco/Git-clones/gassigeher/internal/repository/dog_repository.go`
- Function: `Delete`
- Lines: 358-385

**Impact:**
- Orphaned bookings with invalid dog_id
- Frontend shows bookings for non-existent dogs
- JOIN queries fail or return NULL
- Data integrity violation

**Steps to Reproduce:**
1. Dog exists with no future bookings
2. Admin clicks delete (triggers check)
3. User creates booking for this dog (between check and delete)
4. Admin's delete executes
5. Expected: Delete fails with "cannot delete dog with future bookings"
6. Actual: Dog deleted, booking orphaned

**Fix:**
Use a transaction with proper isolation level, or use a foreign key constraint with ON DELETE RESTRICT:

```diff
 func (r *DogRepository) Delete(id int) error {
+    // Start transaction
+    tx, err := r.db.Begin()
+    if err != nil {
+        return fmt.Errorf("failed to start transaction: %w", err)
+    }
+    defer tx.Rollback()
+
     // Check for future bookings
     currentDate := time.Now().Format("2006-01-02")
     checkQuery := `
         SELECT COUNT(*) FROM bookings
         WHERE dog_id = ? AND date >= ? AND status = 'scheduled'
+        FOR UPDATE  -- Lock the rows
     `

     var count int
-    err := r.db.QueryRow(checkQuery, id, currentDate).Scan(&count)
+    err = tx.QueryRow(checkQuery, id, currentDate).Scan(&count)
     if err != nil {
         return fmt.Errorf("failed to check bookings: %w", err)
     }

     if count > 0 {
         return fmt.Errorf("cannot delete dog with future bookings")
     }

     // Delete the dog
     deleteQuery := `DELETE FROM dogs WHERE id = ?`
-    _, err = r.db.Exec(deleteQuery, id)
+    _, err = tx.Exec(deleteQuery, id)
     if err != nil {
         return fmt.Errorf("failed to delete dog: %w", err)
     }

+    // Commit transaction
+    if err := tx.Commit(); err != nil {
+        return fmt.Errorf("failed to commit transaction: %w", err)
+    }
+
     return nil
 }
```

Better approach: Add foreign key constraint in schema with ON DELETE RESTRICT.

---

## Bug #10: HolidayRepository SetCachedHolidays Uses SQLite-Specific INSERT OR REPLACE

**Description:**
The `SetCachedHolidays` function uses `INSERT OR REPLACE`, which is SQLite-specific syntax. This breaks multi-database compatibility. MySQL uses `REPLACE INTO` or `INSERT ... ON DUPLICATE KEY UPDATE`, while PostgreSQL uses `INSERT ... ON CONFLICT`.

**Location:**
- File: `/home/jaco/Git-clones/gassigeher/internal/repository/holiday_repository.go`
- Function: `SetCachedHolidays`
- Lines: 121-131

**Impact:**
- Application crashes on MySQL/PostgreSQL with syntax error
- Holiday caching completely broken on non-SQLite databases
- Booking time restrictions don't work properly
- Cannot deploy on production MySQL/PostgreSQL

**Steps to Reproduce:**
1. Configure application to use MySQL
2. System fetches holidays from API
3. Attempts to cache with SetCachedHolidays
4. Expected: Cache stored successfully
5. Actual: SQL syntax error, cache not stored

**Fix:**
Use database-agnostic approach with separate INSERT and UPDATE:

```diff
 func (r *HolidayRepository) SetCachedHolidays(year int, state string, data string, cacheDays int) error {
     expiresAt := time.Now().AddDate(0, 0, cacheDays)

-    query := `
-        INSERT OR REPLACE INTO feiertage_cache (year, state, data, fetched_at, expires_at)
-        VALUES (?, ?, ?, ?, ?)
-    `
-
-    _, err := r.db.Exec(query, year, state, data, time.Now(), expiresAt)
-    return err
+    // Try UPDATE first
+    updateQuery := `
+        UPDATE feiertage_cache
+        SET data = ?, fetched_at = ?, expires_at = ?
+        WHERE year = ? AND state = ?
+    `
+
+    result, err := r.db.Exec(updateQuery, data, time.Now(), expiresAt, year, state)
+    if err != nil {
+        return err
+    }
+
+    rows, _ := result.RowsAffected()
+    if rows > 0 {
+        return nil // Update successful
+    }
+
+    // If no rows updated, INSERT
+    insertQuery := `
+        INSERT INTO feiertage_cache (year, state, data, fetched_at, expires_at)
+        VALUES (?, ?, ?, ?, ?)
+    `
+
+    _, err = r.db.Exec(insertQuery, year, state, data, time.Now(), expiresAt)
+    return err
 }
```

---

## Bug #11: UserRepository.FindInactiveUsers Excludes Admins But Query Might Return Empty Set

**Description:**
The `FindInactiveUsers` function correctly excludes admins and super admins from auto-deactivation, but the query uses `AND is_admin = 0 AND is_super_admin = 0`. If the database stores booleans as TRUE/FALSE (PostgreSQL) instead of 0/1 (SQLite), this query returns no results. Additionally, the function doesn't verify that admins are actually excluded if the database has inconsistent boolean representations.

**Location:**
- File: `/home/jaco/Git-clones/gassigeher/internal/repository/user_repository.go`
- Function: `FindInactiveUsers`
- Lines: 424-486

**Impact:**
- Auto-deactivation doesn't work on PostgreSQL
- Admins could potentially be deactivated on some databases
- Cron job reports 0 users deactivated even when many are inactive
- Compliance issue if inactive accounts not cleaned up

**Steps to Reproduce:**
1. Configure application with PostgreSQL
2. Create users with last_activity > 365 days ago
3. Run auto-deactivation cron job
4. Expected: Inactive users deactivated
5. Actual: No users deactivated (query returns 0 rows)

**Fix:**
Use database-agnostic boolean comparison:

```diff
 query := `
     SELECT id, name, email, phone, password_hash, experience_level,
            is_admin, is_super_admin, is_verified, is_active, is_deleted,
            verification_token, verification_token_expires, password_reset_token,
            password_reset_expires, profile_photo, anonymous_id,
            terms_accepted_at, last_activity_at, deactivated_at,
            deactivation_reason, reactivated_at, deleted_at,
            created_at, updated_at
     FROM users
     WHERE is_active = 1
       AND is_deleted = 0
-      AND is_admin = 0
-      AND is_super_admin = 0
+      AND is_admin != 1
+      AND is_super_admin != 1
       AND last_activity_at < ?
 `
```

Even better, use NOT:

```diff
-      AND is_admin = 0
-      AND is_super_admin = 0
+      AND NOT is_admin
+      AND NOT is_super_admin
```

But this might not work on SQLite. The safest approach that works everywhere:

```diff
+      AND (is_admin = 0 OR is_admin = FALSE OR is_admin IS NULL)
+      AND (is_super_admin = 0 OR is_super_admin = FALSE OR is_super_admin IS NULL)
```

Or just rely on the boolean value being falsy:

```diff
-      AND is_admin = 0
-      AND is_super_admin = 0
+      AND NOT COALESCE(is_admin, 0)
+      AND NOT COALESCE(is_super_admin, 0)
```

---

## Bug #12: PromoteToAdmin and DemoteAdmin Use Boolean Literals Instead of Database-Agnostic Values

**Description:**
The `PromoteToAdmin` and `DemoteAdmin` functions pass Go boolean values (`true` and `false`) directly to `db.Exec()`. While Go's database/sql package converts these to appropriate database types, it's inconsistent with the rest of the codebase which uses integer values (0/1) for boolean fields. This could cause issues if the database driver doesn't properly convert booleans.

**Location:**
- File: `/home/jaco/Git-clones/gassigeher/internal/repository/user_repository.go`
- Function: `PromoteToAdmin`, `DemoteAdmin`
- Lines: 563, 574

**Impact:**
- Inconsistent boolean handling across codebase
- Potential type conversion errors on some database drivers
- Admin promotion/demotion might fail silently
- Testing difficulties due to type inconsistencies

**Steps to Reproduce:**
1. Use custom database driver with strict type checking
2. Attempt to promote user to admin
3. Expected: is_admin set to 1
4. Actual: May fail with type mismatch error (driver-dependent)

**Fix:**
Use integer values consistently with the rest of the codebase:

```diff
 func (r *UserRepository) PromoteToAdmin(userID int) error {
     query := `UPDATE users SET is_admin = ?, updated_at = ? WHERE id = ?`
-    _, err := r.db.Exec(query, true, time.Now(), userID)
+    _, err := r.db.Exec(query, 1, time.Now(), userID)
     if err != nil {
         return fmt.Errorf("failed to promote user to admin: %w", err)
     }
     return nil
 }

 func (r *UserRepository) DemoteAdmin(userID int) error {
     query := `UPDATE users SET is_admin = ?, updated_at = ? WHERE id = ?`
-    _, err := r.db.Exec(query, false, time.Now(), userID)
+    _, err := r.db.Exec(query, 0, time.Now(), userID)
     if err != nil {
         return fmt.Errorf("failed to demote admin: %w", err)
     }
     return nil
 }
```

---

## Bug #13: Missing Transaction in BookingRepository.Update Could Create Orphaned Modifications

**Description:**
The `Update` function in `booking_repository.go` allows admins to move bookings to new dates/times. However, it doesn't check if the new date/time slot is already taken by another booking (double-booking). The `CheckDoubleBooking` function exists but isn't called before the update, creating a race condition where two bookings end up scheduled for the same dog at the same time.

**Location:**
- File: `/home/jaco/Git-clones/gassigeher/internal/repository/booking_repository.go`
- Function: `Update`
- Lines: 408-428

**Impact:**
- Admin can move booking to already-booked slot
- Two users scheduled for same dog at same time
- Conflicts at walk time
- Poor user experience
- Data integrity violation

**Steps to Reproduce:**
1. User A has booking for Dog X on 2025-12-10 morning
2. User B has booking for Dog X on 2025-12-15 morning
3. Admin moves User B's booking to 2025-12-10 morning
4. Expected: Error "time slot already taken"
5. Actual: Both bookings now on 2025-12-10 morning

**Fix:**
Add double-booking check before update:

```diff
 func (r *BookingRepository) Update(booking *models.Booking) error {
+    // Check for double-booking at the new date/time
+    exists, err := r.CheckDoubleBooking(booking.DogID, booking.Date, booking.ScheduledTime)
+    if err != nil {
+        return fmt.Errorf("failed to check double booking: %w", err)
+    }
+
+    if exists {
+        // Need to check if the existing booking is this same booking
+        checkQuery := `
+            SELECT id FROM bookings
+            WHERE dog_id = ? AND date = ? AND scheduled_time = ? AND status = 'scheduled'
+        `
+        var existingID int
+        err := r.db.QueryRow(checkQuery, booking.DogID, booking.Date, booking.ScheduledTime).Scan(&existingID)
+        if err != nil && err != sql.ErrNoRows {
+            return fmt.Errorf("failed to check existing booking: %w", err)
+        }
+
+        // If it's a different booking, return error
+        if existingID != 0 && existingID != booking.ID {
+            return fmt.Errorf("time slot already taken by another booking")
+        }
+    }
+
     query := `
         UPDATE bookings
         SET date = ?, scheduled_time = ?, updated_at = ?
         WHERE id = ?
     `

     _, err := r.db.Exec(query,
         booking.Date,
         booking.ScheduledTime,
         time.Now(),
         booking.ID,
     )

     if err != nil {
         return fmt.Errorf("failed to update booking: %w", err)
     }

     return nil
 }
```

Better: Modify CheckDoubleBooking to accept an optional bookingID to exclude from the check.

---

## Statistics

- **Critical:** 3 bugs
  - Bug #1: Multi-database compatibility broken
  - Bug #9: Race condition causing data integrity violation
  - Bug #10: SQL syntax breaks on MySQL/PostgreSQL

- **High:** 6 bugs
  - Bug #2: Missing error checks on LastInsertId (PARTIALLY FIXED)
  - Bug #3: Missing error checks on RowsAffected
  - Bug #4: Double-booking check incomplete
  - Bug #6: Reminder time calculation flawed (CODE MODIFIED)
  - Bug #11: Boolean comparison breaks on PostgreSQL
  - Bug #13: Missing double-booking check in Update

- **Medium:** 4 bugs
  - Bug #5: Edge case in time comparison
  - Bug #7: Missing rows.Err() checks
  - Bug #8: Ambiguous deleted entity handling
  - Bug #12: Inconsistent boolean value usage

---

## Verification Summary (2025-12-01)

**Bugs Still Present with Accurate Locations:** 11 bugs
- Bug #1: Line 38-40 (UNCHANGED)
- Bug #2: Lines 124 (booking_time_repository.go) and 72 (holiday_repository.go) UNCHANGED, but blocked_date_repository.go FIXED
- Bug #3: Lines 626, 652 (UNCHANGED)
- Bug #4: Line 232 (UNCHANGED)
- Bug #5: Line 257 (UNCHANGED)
- Bug #7: Multiple locations (UNCHANGED)
- Bug #8: Lines 444-503 (UNCHANGED)
- Bug #9: Lines 358-385 (UNCHANGED)
- Bug #10: Line 125 (UNCHANGED)
- Bug #11: Lines 437-438 (UNCHANGED)
- Bug #12: Lines 563, 574 (UNCHANGED)
- Bug #13: Lines 408-428 (UNCHANGED)

**Bugs with Code Changes:** 1 bug
- Bug #6: GetForReminders logic simplified, midnight-crossing functionality removed (lines 319-395)

**Line Number Updates:** None needed - all line numbers remain accurate

---

## Recommendations

### Immediate Actions (Critical Priority)

1. **Fix multi-database compatibility** (Bugs #1, #10, #11, #12)
   - Audit all SQL queries for database-specific syntax
   - Use database-agnostic patterns (separate INSERT/UPDATE vs UPSERT)
   - Standardize boolean value handling (use integers 0/1)
   - Add integration tests for MySQL and PostgreSQL

2. **Implement proper error handling** (Bugs #2, #3, #7)
   - Check all LastInsertId() calls
   - Check all RowsAffected() calls
   - Add rows.Err() checks after all row iterations
   - Add error context with fmt.Errorf wrapping

3. **Fix race conditions and data integrity** (Bugs #4, #9, #13)
   - Use transactions for check-then-modify operations
   - Add database constraints (foreign keys, unique constraints)
   - Implement proper locking where needed

### High Priority Improvements

1. **Add comprehensive repository tests**
   - Test all error paths
   - Test edge cases (midnight boundaries, time zones)
   - Test concurrent operations
   - Test on all three database backends

2. **Implement transaction support**
   - Add BeginTx methods to repositories
   - Use proper isolation levels
   - Add rollback on error

3. **Add input validation**
   - Validate date ranges before queries
   - Validate time formats
   - Validate foreign key references

### Long-term Architectural Improvements

1. **Add database abstraction layer**
   - Create dialect-aware query builders
   - Centralize database-specific logic
   - Make UPSERT operations portable

2. **Implement optimistic locking**
   - Add version column to tables
   - Check version on update
   - Return concurrency error on mismatch

3. **Add monitoring and observability**
   - Log slow queries
   - Track error rates by type
   - Monitor connection pool usage
   - Alert on transaction deadlocks

4. **Documentation**
   - Document expected concurrent behavior
   - Document transaction boundaries
   - Document error handling patterns
   - Add examples for each repository method

---

**Analysis Complete**

This bug report identifies 13 functional bugs in the repository layer, ranging from critical multi-database compatibility issues to edge cases in time handling. The most urgent fixes are the SQL syntax issues that break the documented multi-database support, followed by error handling gaps that could lead to data corruption.
