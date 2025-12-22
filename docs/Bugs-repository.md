# Bug Report: repository

**Analysis Date:** 2025-12-22
**Directory Analyzed:** `/home/tranmh/work/gassigeher-saas/internal/repository`
**Files Analyzed:** 15 files
**Bugs Found:** 8 bugs

---

## Summary

Analysis of the repository layer revealed 8 functional bugs across multiple files. The issues range from database compatibility problems (SQLite-specific SQL), missing error checks, potential race conditions in multi-tenant operations, to incorrect date calculation logic. Most critical is the SQLite-specific "INSERT OR REPLACE" syntax in holiday_repository.go which breaks MySQL/PostgreSQL compatibility, and the date formatting bug in tenant_repository.go that causes incorrect monthly statistics.

**Severity Distribution:**
- Critical: 1 bug (database compatibility)
- High: 3 bugs (logic errors, missing error checks)
- Medium: 3 bugs (error handling gaps, race conditions)
- Low: 1 bug (inconsistent error handling)

---

## Bugs

## Bug #1: Database Compatibility Issue - SQLite-Specific SQL Syntax

**Severity:** Critical

**Description:**
The `SetCachedHolidays` method in `holiday_repository.go` uses `INSERT OR REPLACE INTO` SQL syntax, which is SQLite-specific and will fail on MySQL/PostgreSQL. According to the codebase documentation (CLAUDE.md), this application supports three database backends (SQLite, MySQL, PostgreSQL) with "100% standard SQL" in repositories. This violates the multi-database support requirement and will cause runtime errors on non-SQLite databases.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/repository/holiday_repository.go`
- Function: `SetCachedHolidays`
- Lines: 126-133

**Steps to Reproduce:**
1. Configure application to use MySQL or PostgreSQL (set `DB_TYPE=mysql` or `DB_TYPE=postgres`)
2. Application fetches German holidays from feiertage-api.de
3. Call `SetCachedHolidays(year, state, data, cacheDays)` to cache the result
4. Expected: Cache successfully stored
5. Actual: SQL syntax error - "INSERT OR REPLACE" is not valid MySQL/PostgreSQL syntax

**Fix:**
Use database-agnostic upsert pattern with standard SQL. Use UPDATE + INSERT pattern:

```diff
  func (r *HolidayRepository) SetCachedHolidays(year int, state string, data string, cacheDays int) error {
  	expiresAt := time.Now().AddDate(0, 0, cacheDays)

- 	query := `
- 		INSERT OR REPLACE INTO feiertage_cache (year, state, data, fetched_at, expires_at)
- 		VALUES (?, ?, ?, ?, ?)
- 	`
-
- 	_, err := r.db.Exec(query, year, state, data, time.Now(), expiresAt)
- 	return err
+ 	// Try UPDATE first (standard SQL)
+ 	updateQuery := `UPDATE feiertage_cache SET data = ?, fetched_at = ?, expires_at = ? WHERE year = ? AND state = ?`
+ 	result, err := r.db.Exec(updateQuery, data, time.Now(), expiresAt, year, state)
+ 	if err != nil {
+ 		return fmt.Errorf("failed to update cache: %w", err)
+ 	}
+
+ 	rows, err := result.RowsAffected()
+ 	if err != nil {
+ 		return fmt.Errorf("failed to check rows affected: %w", err)
+ 	}
+
+ 	if rows == 0 {
+ 		// Row doesn't exist, INSERT it
+ 		insertQuery := `INSERT INTO feiertage_cache (year, state, data, fetched_at, expires_at) VALUES (?, ?, ?, ?, ?)`
+ 		_, err = r.db.Exec(insertQuery, year, state, data, time.Now(), expiresAt)
+ 		if err != nil {
+ 			return fmt.Errorf("failed to insert cache: %w", err)
+ 		}
+ 	}
+ 	return nil
  }
```

This uses standard SQL that works on all three supported databases (SQLite, MySQL, PostgreSQL).

---

## Bug #2: Missing Error Check for RowsAffected in Booking Approval

**Severity:** High

**Description:**
The `ApproveBooking` method checks if `rows == 0` after `RowsAffected()` to detect if a booking was not found or not pending. However, it ignores the error returned by `RowsAffected()` itself (line 719: `rows, _ := result.RowsAffected()`). If `RowsAffected()` fails, `rows` will be 0, causing a misleading "booking not found or not pending" error when the real issue is a database driver error.

This same bug also exists in `RejectBooking` at line 745.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/repository/booking_repository.go`
- Function: `ApproveBooking`
- Lines: 706-725

**Steps to Reproduce:**
1. Simulate a database driver error in `RowsAffected()` (e.g., network interruption)
2. Call `ApproveBooking(bookingID, adminID)`
3. Expected: Return error indicating database issue ("failed to check rows affected")
4. Actual: Returns misleading error "booking not found or not pending"

**Fix:**
Check the error from `RowsAffected()`:

```diff
  func (r *BookingRepository) ApproveBooking(bookingID int, adminID int) error {
  	query := `
  		UPDATE bookings
  		SET approval_status = 'approved', approved_by = ?, approved_at = ?
  		WHERE id = ? AND approval_status = 'pending'
  	`

  	result, err := r.db.Exec(query, adminID, time.Now(), bookingID)
  	if err != nil {
  		return err
  	}

- 	rows, _ := result.RowsAffected()
+ 	rows, err := result.RowsAffected()
+ 	if err != nil {
+ 		return fmt.Errorf("failed to get rows affected: %w", err)
+ 	}
  	if rows == 0 {
  		return fmt.Errorf("booking not found or not pending")
  	}

  	return nil
  }
```

Apply the same fix to `RejectBooking` at line 728-751.

---

## Bug #3: Incorrect Month Calculation in Booking Query

**Severity:** High

**Description:**
In the `FindAll` method, when filtering by year and month, the code calculates the last day of the month using a confusing pattern. Line 178 creates a time.Date with `month+1`, which for December (month=12) becomes 13. While Go's time.Date normalizes this to January of the next year, and the subsequent subtraction of 24 hours works correctly, this logic is fragile and confusing. If someone modifies the subtraction amount, it will break.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/repository/booking_repository.go`
- Function: `FindAll`
- Lines: 174-183

**Steps to Reproduce:**
1. Query bookings with `filter.Year = 2025, filter.Month = 12`
2. Current code: `time.Date(2025, time.Month(13), 1, ...)` → normalizes to `2026-01-01`, then subtract 24h → `2025-12-31`
3. Expected: Get bookings from December 1-31, 2025
4. Actual: Works, but logic is fragile (changing the subtraction breaks it)

**Fix:**
Use explicit calculation for last day of month:

```diff
  		if filter.Year != nil && filter.Month != nil {
  			// Filter by year and month
  			startDate := fmt.Sprintf("%d-%02d-01", *filter.Year, *filter.Month)
- 			// Calculate last day of month
- 			nextMonth := time.Date(*filter.Year, time.Month(*filter.Month+1), 1, 0, 0, 0, 0, time.UTC)
- 			endDate := nextMonth.Add(-24 * time.Hour).Format("2006-01-02")
+ 			// Calculate last day of month correctly
+ 			// Start with first day of current month, add 1 month, subtract 1 day
+ 			firstOfMonth := time.Date(*filter.Year, time.Month(*filter.Month), 1, 0, 0, 0, 0, time.UTC)
+ 			firstOfNextMonth := firstOfMonth.AddDate(0, 1, 0)
+ 			lastDay := firstOfNextMonth.Add(-24 * time.Hour)
+ 			endDate := lastDay.Format("2006-01-02")

  			query += " AND date >= ? AND date <= ?"
  			args = append(args, startDate, endDate)
  		}
```

This makes the logic explicit and less fragile.

---

## Bug #4: Incorrect Date Formatting in Tenant Statistics Query

**Severity:** High

**Description:**
The `GetStats` method in `tenant_repository.go` calculates bookings for the current month using an incorrect date format. Line 367 uses `firstOfMonth := time.Now().Format("2006-01-01")` which always formats to January 1st (month "01" is hardcoded), regardless of the current month. This means it will count bookings from January 1st to today instead of from the 1st of the current month.

For example, if today is December 22, 2025, the code formats to "2025-01-01" instead of "2025-12-01", so it counts all bookings from January 1st, not December 1st. This results in wildly incorrect "Bookings This Month" statistics on the admin dashboard.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/repository/tenant_repository.go`
- Function: `GetStats`
- Lines: 366-371

**Steps to Reproduce:**
1. Today is December 22, 2025
2. Create 5 bookings on December 1-5, 2025
3. Create 100 bookings on January 1-31, 2025
4. Call `GetStats(tenantID)`
5. Expected: `BookingsThisMonth = 5` (December bookings)
6. Actual: `BookingsThisMonth = 105` (all bookings since January 1st)

**Fix:**
Use correct date formatting to get the first day of the current month:

```diff
  	// Bookings this month
- 	firstOfMonth := time.Now().Format("2006-01-01")
+ 	now := time.Now()
+ 	firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
  	err = r.db.QueryRow(`SELECT COUNT(*) FROM bookings WHERE tenant_id = ? AND date >= ?`, tenantID, firstOfMonth).Scan(&stats.BookingsThisMonth)
  	if err != nil {
  		return nil, fmt.Errorf("failed to count monthly bookings: %w", err)
  	}
```

This correctly gets the first day of the current month in "YYYY-MM-DD" format.

---

## Bug #5: Transaction Commit Failure Doesn't Prevent Model Update

**Severity:** Medium

**Description:**
In the `CreateWithLimitCheck` method in `dog_repository.go`, if the transaction commit fails (line 432), the function returns an error but still sets `dog.ID`, `dog.CreatedAt`, and `dog.UpdatedAt` after the failed commit (lines 436-438). This means the dog object will have an ID even though the database insertion failed. If the caller doesn't check the error properly, they might assume the dog was created successfully.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/repository/dog_repository.go`
- Function: `CreateWithLimitCheck`
- Lines: 426-440

**Steps to Reproduce:**
1. Simulate a commit failure (e.g., database connection lost during commit)
2. Call `CreateWithLimitCheck(dog, limit)`
3. Function returns error
4. Check `dog.ID` - it's set to a non-zero value
5. Expected: dog.ID remains 0 (unchanged) on error
6. Actual: dog.ID is set even though database insertion failed

**Fix:**
Get the ID before commit, but only set it on dog object after successful commit:

```diff
  	id, err := result.LastInsertId()
  	if err != nil {
  		return fmt.Errorf("failed to get dog ID: %w", err)
  	}

  	// Commit transaction
  	if err := tx.Commit(); err != nil {
  		return fmt.Errorf("failed to commit transaction: %w", err)
  	}

+ 	// Only set fields after successful commit
  	dog.ID = int(id)
  	dog.CreatedAt = time.Now()
  	dog.UpdatedAt = time.Now()
  	return nil
```

This ensures the dog object is only modified after successful database commit.

---

## Bug #6: Potential Race Condition in User Color Assignment

**Severity:** Medium

**Description:**
The `SetUserColors` method in `user_color_repository.go` uses a transaction to delete all colors and then insert new ones. However, if two concurrent requests try to set colors for the same user, there's a race condition. Both transactions might execute DELETE, then both INSERT their colors, and whichever commits last wins. This violates the expectation that `SetUserColors` is atomic and could result in data loss (the first transaction's colors are overwritten).

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/repository/user_color_repository.go`
- Function: `SetUserColors`
- Lines: 123-154

**Steps to Reproduce:**
1. Start two concurrent goroutines
2. Goroutine A calls `SetUserColors(tenantID, userID, [colorID1, colorID2], adminA)`
3. Goroutine B calls `SetUserColors(tenantID, userID, [colorID3, colorID4], adminB)` simultaneously
4. Expected: One succeeds, other gets conflict error, or both succeed with last one winning
5. Actual: Both may appear to succeed, but only the last commit's colors are saved

**Fix:**
Add row-level locking to prevent concurrent modifications:

```diff
  func (r *UserColorRepository) SetUserColors(tenantID int, userID int, colorIDs []int, grantedBy int) error {
  	// Start transaction
  	tx, err := r.db.Begin()
  	if err != nil {
  		return fmt.Errorf("failed to start transaction: %w", err)
  	}
  	defer tx.Rollback()

+ 	// Lock the user row to prevent concurrent modifications
+ 	// Use SELECT ... FOR UPDATE to acquire a lock
+ 	_, err = tx.Exec("SELECT id FROM users WHERE id = ? AND tenant_id = ? FOR UPDATE", userID, tenantID)
+ 	if err != nil {
+ 		return fmt.Errorf("failed to lock user record: %w", err)
+ 	}
+
  	// Remove all existing colors for this user in this tenant
  	_, err = tx.Exec("DELETE FROM user_colors WHERE user_id = ? AND tenant_id = ?", userID, tenantID)
  	if err != nil {
  		return fmt.Errorf("failed to remove existing colors: %w", err)
  	}

  	// Add new colors
  	now := time.Now()
  	for _, colorID := range colorIDs {
  		_, err = tx.Exec(
  			"INSERT INTO user_colors (tenant_id, user_id, color_id, granted_at, granted_by) VALUES (?, ?, ?, ?, ?)",
  			tenantID, userID, colorID, now, grantedBy,
  		)
  		if err != nil {
  			return fmt.Errorf("failed to add color %d: %w", colorID, err)
  		}
  	}

  	if err := tx.Commit(); err != nil {
  		return fmt.Errorf("failed to commit transaction: %w", err)
  	}

  	return nil
  }
```

This prevents concurrent updates to the same user's colors.

---

## Bug #7: Missing Error Check on LastInsertId in Multiple Repositories

**Severity:** Medium

**Description:**
Several repository methods call `result.LastInsertId()` but ignore the error (using `id, _ := result.LastInsertId()`). While this is generally safe on SQLite, on MySQL/PostgreSQL, `LastInsertId()` can fail if:
- The table doesn't have an auto-increment primary key
- The driver doesn't support LastInsertId (rare but possible)
- There's a database connection issue after the INSERT

Ignoring this error means the method would set an ID of 0 on the model, which could cause downstream issues.

**Affected files and lines:**
1. `booking_time_repository.go` line 125 - `CreateRule`
2. `holiday_repository.go` line 73 - `CreateHoliday`

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/repository/booking_time_repository.go`
- Function: `CreateRule`
- Lines: 108-129

**Steps to Reproduce:**
1. Use a database configuration where LastInsertId might fail
2. Call `CreateRule(tenantID, rule)`
3. INSERT succeeds but LastInsertId fails
4. Expected: Return error
5. Actual: Sets rule.ID = 0, returns success (misleading)

**Fix:**
Check the error from LastInsertId:

```diff
  func (r *BookingTimeRepository) CreateRule(tenantID int, rule *models.BookingTimeRule) error {
  	query := `
  		INSERT INTO booking_time_rules (tenant_id, day_type, rule_name, start_time, end_time, is_blocked)
  		VALUES (?, ?, ?, ?, ?, ?)
  	`

  	isBlocked := 0
  	if rule.IsBlocked {
  		isBlocked = 1
  	}

  	result, err := r.db.Exec(query, tenantID, rule.DayType, rule.RuleName, rule.StartTime, rule.EndTime, isBlocked)
  	if err != nil {
  		return err
  	}

- 	id, _ := result.LastInsertId()
+ 	id, err := result.LastInsertId()
+ 	if err != nil {
+ 		return fmt.Errorf("failed to get inserted rule ID: %w", err)
+ 	}
  	rule.ID = int(id)
  	rule.TenantID = tenantID
  	return nil
  }
```

Apply the same fix to `holiday_repository.go:CreateHoliday` (line 68-77).

---

## Bug #8: Inconsistent Error Handling in Settings Repository Upsert

**Severity:** Low

**Description:**
The `Upsert` method in `settings_repository.go` checks for a specific error string ("setting not found") to determine whether to insert or return an error. This is fragile because:
1. It relies on exact string matching, which breaks if the error message changes
2. It doesn't distinguish between "setting not found" and other errors from Update
3. String comparison is inefficient and error-prone

A better approach would be to check `RowsAffected` directly.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/repository/settings_repository.go`
- Function: `Upsert`
- Lines: 121-131

**Steps to Reproduce:**
1. Call `Upsert` when Update fails due to database error (not "setting not found")
2. Error message is something like "database connection lost"
3. Expected: Return the database error
4. Actual: Returns the database error (works), but via fragile string comparison

**Fix:**
Use a more explicit approach with RowsAffected:

```diff
  // Upsert creates or updates a setting for a tenant
  func (r *SettingsRepository) Upsert(tenantID int, key, value string) error {
- 	// Try to update first
- 	err := r.Update(tenantID, key, value)
- 	if err != nil && err.Error() == "setting not found" {
- 		// Setting doesn't exist, create it
- 		return r.Create(tenantID, key, value)
- 	}
- 	return err
+ 	query := `
+ 		UPDATE system_settings
+ 		SET value = ?, updated_at = ?
+ 		WHERE key = ? AND tenant_id = ?
+ 	`
+
+ 	result, err := r.db.Exec(query, value, time.Now(), key, tenantID)
+ 	if err != nil {
+ 		return fmt.Errorf("failed to update setting: %w", err)
+ 	}
+
+ 	rows, err := result.RowsAffected()
+ 	if err != nil {
+ 		return fmt.Errorf("failed to check rows affected: %w", err)
+ 	}
+
+ 	if rows == 0 {
+ 		// Setting doesn't exist, create it
+ 		return r.Create(tenantID, key, value)
+ 	}
+
+ 	return nil
  }
```

This avoids string comparison and is more robust.

---

## Statistics

- **Critical:** 1 bug (database compatibility issue)
- **High:** 3 bugs (logic errors, missing error checks, incorrect calculations)
- **Medium:** 3 bugs (transaction handling, race conditions, error handling)
- **Low:** 1 bug (code quality, string comparison)

---

## Recommendations

### Immediate Actions (Critical/High Priority)

1. **Fix Bug #1 (Database Compatibility)**: Replace SQLite-specific "INSERT OR REPLACE" with standard SQL in `holiday_repository.go`. This is critical for multi-database support as documented.

2. **Fix Bug #4 (Date Formatting)**: Correct the date formatting in `GetStats` to accurately calculate monthly statistics. This directly affects admin dashboard reporting accuracy.

3. **Fix Bug #2 (Error Handling)**: Add error checks for `RowsAffected()` calls in booking approval/rejection methods to prevent misleading error messages.

4. **Fix Bug #3 (Month Calculation)**: Clarify and correct the month boundary calculation logic to make it more maintainable and less fragile.

### Short-term Improvements (Medium Priority)

5. **Address Bug #5 (Transaction Consistency)**: Only set model fields after successful transaction commit to maintain data consistency and prevent confusion.

6. **Address Bug #6 (Race Conditions)**: Add row-level locking in `SetUserColors` to prevent concurrent modification issues, especially in multi-admin scenarios.

7. **Address Bug #7 (Error Checks)**: Add error checks for all `LastInsertId()` calls across repositories for robustness across all database backends.

### Long-term Code Quality (Low Priority)

8. **Refactor Bug #8**: Improve the Upsert pattern to avoid string-based error checking and use RowsAffected instead.

### General Best Practices

1. **Add Integration Tests**: Create integration tests that run against all three supported databases (SQLite, MySQL, PostgreSQL) to catch database-specific issues early.

2. **Code Review Checklist**: Add items to verify:
   - All `rows.Close()` are deferred immediately after Query
   - All `RowsAffected()` errors are checked
   - All `LastInsertId()` errors are checked
   - No database-specific SQL syntax is used (per CLAUDE.md requirements)

3. **SQL Linting**: Consider adding a SQL linter to detect database-specific syntax at build time.

4. **Concurrency Testing**: Add stress tests that simulate concurrent operations to detect race conditions in multi-tenant scenarios.

5. **Documentation**: Update repository documentation to explicitly state:
   - Transaction isolation level requirements
   - Concurrency guarantees
   - Multi-database compatibility requirements
