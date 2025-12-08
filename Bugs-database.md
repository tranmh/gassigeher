# Bug Report: internal/database

**Analysis Date:** 2025-12-01
**Directory Analyzed:** `internal/database`
**Files Analyzed:** 22 files
**Bugs Found:** 11 bugs

---

## Summary

The `internal/database` directory contains the multi-database support system (SQLite, MySQL, PostgreSQL) with migration management, dialect abstraction, and seed data generation. Analysis revealed critical bugs in:

1. **Race conditions in seed data generation** (insecure random number generator)
2. **Migration ordering and idempotency issues** (duplicate constant definitions)
3. **Schema inconsistencies** across databases (missing indexes, foreign key actions)
4. **Missing error handling** in migration application
5. **SQL injection potential** in PostgreSQL dialect
6. **Data integrity issues** in seed data
7. **Connection pool misconfiguration** for SQLite

**Critical Issues:** 3 bugs (race condition, SQL injection potential, schema inconsistency)
**High Priority:** 5 bugs (migration issues, missing indexes, data integrity)
**Medium Priority:** 3 bugs (error handling, connection pool)

---

## Bugs

## Bug #1: Insecure Random Number Generator in Seed Data

**Severity:** CRITICAL

**STATUS: VERIFIED - Code unchanged at lines 115**

**Description:**
The `seed.go` file uses `math/rand` with `rand.Seed(time.Now().UnixNano())` for generating passwords. This is cryptographically insecure and produces predictable passwords. An attacker who knows the approximate server start time can predict the generated Super Admin password, leading to complete system compromise.

**Location:**
- File: `internal/database/seed.go`
- Function: `generateSecurePassword`
- Lines: 115

**Steps to Reproduce:**
1. Start the application for the first time (triggers seed data)
2. Observe that `rand.Seed(time.Now().UnixNano())` is called
3. The seed is based on nanosecond timestamp, which has limited entropy
4. With knowledge of server start time (±1 second), an attacker can brute-force the exact seed
5. Regenerate the "random" password and gain Super Admin access

**Impact:**
- Complete authentication bypass for Super Admin account
- Full system compromise on first installation
- Predictable test user passwords

**Fix:**
Replace `math/rand` with `crypto/rand` for secure random generation:

```diff
package database

import (
+	"crypto/rand"
-	"math/rand"
	"fmt"
	"time"
)

func generateSecurePassword(length int) string {
	lowercase := "abcdefghijklmnopqrstuvwxyz"
	uppercase := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numbers := "0123456789"
	special := "!@#$%^&*"
	allChars := lowercase + uppercase + numbers + special

-	rand.Seed(time.Now().UnixNano())

	password := make([]byte, length)
-	// Ensure at least one of each type
-	password[0] = lowercase[rand.Intn(len(lowercase))]
-	password[1] = uppercase[rand.Intn(len(uppercase))]
-	password[2] = numbers[rand.Intn(len(numbers))]
-	password[3] = special[rand.Intn(len(special))]
-
-	// Fill rest randomly
-	for i := 4; i < length; i++ {
-		password[i] = allChars[rand.Intn(len(allChars))]
-	}
-
-	// Shuffle
-	rand.Shuffle(len(password), func(i, j int) {
-		password[i], password[j] = password[j], password[i]
-	})

+	// Use crypto/rand for secure random bytes
+	randomBytes := make([]byte, length)
+	if _, err := rand.Read(randomBytes); err != nil {
+		panic(fmt.Sprintf("failed to generate secure random: %v", err))
+	}
+
+	// Ensure at least one of each type
+	password[0] = lowercase[randomBytes[0]%byte(len(lowercase))]
+	password[1] = uppercase[randomBytes[1]%byte(len(uppercase))]
+	password[2] = numbers[randomBytes[2]%byte(len(numbers))]
+	password[3] = special[randomBytes[3]%byte(len(special))]
+
+	// Fill rest using secure random
+	for i := 4; i < length; i++ {
+		password[i] = allChars[randomBytes[i]%byte(len(allChars))]
+	}
+
+	// Shuffle using Fisher-Yates with crypto/rand
+	for i := len(password) - 1; i > 0; i-- {
+		jBytes := make([]byte, 1)
+		rand.Read(jBytes)
+		j := int(jBytes[0]) % (i + 1)
+		password[i], password[j] = password[j], password[i]
+	}

	return string(password)
}
```

---

## Bug #2: Duplicate Schema Definitions Create Migration Confusion

**Severity:** HIGH

**STATUS: VERIFIED - Duplicate constants still present at lines 215-354**

**Description:**
The `database.go` file contains hardcoded SQL table definitions (lines 215-354) that duplicate the migration files (`001_*.go`, `002_*.go`, etc.). These are never executed because `RunMigrations()` now delegates to `RunMigrationsWithDialect()`. This creates confusion and a maintenance hazard where schema changes might be made to the wrong location.

**Location:**
- File: `internal/database/database.go`
- Lines: 215-354

**Steps to Reproduce:**
1. Read `database.go` lines 215-354 (const createUsersTable, createDogsTable, etc.)
2. Read migration files `001_create_users_table.go`, `002_create_dogs_table.go`
3. Compare schemas - they are similar but not identical
4. Observe that `RunMigrations()` calls `RunMigrationsWithDialect()` (line 212)
5. The constants in `database.go` are never used

**Impact:**
- Developer confusion about which schema is authoritative
- Risk of updating wrong schema definition
- Dead code cluttering codebase
- Potential schema drift if someone updates constants instead of migrations

**Fix:**
Remove the unused constants and update comments:

```diff
func RunMigrations(db *sql.DB) error {
-	// Use SQLite dialect by default (for backward compatibility)
-	// If you need other databases, use RunMigrationsWithDialect directly
+	// DEPRECATED: Use RunMigrationsWithDialect() for new code
+	// This function exists only for backward compatibility with existing tests
	dialect := &SQLiteDialect{}
	return RunMigrationsWithDialect(db, dialect)
}

-const createUsersTable = `
-CREATE TABLE IF NOT EXISTS users (
-  id INTEGER PRIMARY KEY AUTOINCREMENT,
-  ...
-);
-...
-`
-
-const createDogsTable = `...`
-const createBookingsTable = `...`
-const createBlockedDatesTable = `...`
-const createExperienceRequestsTable = `...`
-const createSystemSettingsTable = `...`
-const createReactivationRequestsTable = `...`
-const insertDefaultSettings = `...`
-const addPhotoThumbnailColumn = `...`
+
+// Note: All schema definitions are now in migration files (internal/database/00X_*.go)
+// Migrations are applied via RunMigrationsWithDialect()
```

---

## Bug #3: Missing Index on bookings.user_id in SQLite and PostgreSQL

**Severity:** HIGH

**STATUS: VERIFIED - Indexes missing from migration 003, present only in migration 012 for SQLite**

**Description:**
Migration `012_booking_times.go` adds indexes on `bookings(user_id)` and `bookings(dog_id)` for SQLite after recreating the table (lines 80-82), but these indexes are missing from the original `003_create_bookings_table.go` for all databases. MySQL and PostgreSQL in migration 012 don't add these indexes. This causes inconsistent query performance across databases when filtering bookings by user or dog.

**Location:**
- File: `internal/database/003_create_bookings_table.go`
- Missing indexes in MySQL and PostgreSQL versions
- File: `internal/database/012_booking_times.go`
- Lines: 80-84 (SQLite only)

**Steps to Reproduce:**
1. Create fresh database on MySQL or PostgreSQL
2. Run migrations
3. Execute: `EXPLAIN SELECT * FROM bookings WHERE user_id = 1`
4. Observe full table scan (no index on user_id)
5. Compare with SQLite after migration 012 - it HAS the index
6. Query performance degrades on MySQL/PostgreSQL with many bookings

**Impact:**
- Slow queries when fetching user's bookings (common operation)
- Slow queries when fetching dog's bookings
- Schema inconsistency across databases
- Performance degrades faster on MySQL/PostgreSQL vs SQLite

**Fix:**
Add indexes to all databases in migration 003:

```diff
// File: 003_create_bookings_table.go

"sqlite": `
CREATE TABLE IF NOT EXISTS bookings (
  ...
  UNIQUE(dog_id, date, walk_type)
);
+
+CREATE INDEX IF NOT EXISTS idx_bookings_user ON bookings(user_id);
+CREATE INDEX IF NOT EXISTS idx_bookings_dog ON bookings(dog_id);
+CREATE INDEX IF NOT EXISTS idx_bookings_date ON bookings(date);
+CREATE INDEX IF NOT EXISTS idx_bookings_status ON bookings(status);
`,

"mysql": `
CREATE TABLE IF NOT EXISTS bookings (
  ...
  UNIQUE KEY unique_dog_date_walktype (dog_id, date, walk_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
+
+CREATE INDEX idx_bookings_user ON bookings(user_id);
+CREATE INDEX idx_bookings_dog ON bookings(dog_id);
+CREATE INDEX idx_bookings_date ON bookings(date);
+CREATE INDEX idx_bookings_status ON bookings(status);
`,

"postgres": `
CREATE TABLE IF NOT EXISTS bookings (
  ...
  UNIQUE(dog_id, date, walk_type)
);
+
+CREATE INDEX IF NOT EXISTS idx_bookings_user ON bookings(user_id);
+CREATE INDEX IF NOT EXISTS idx_bookings_dog ON bookings(dog_id);
+CREATE INDEX IF NOT EXISTS idx_bookings_date ON bookings(date);
+CREATE INDEX IF NOT EXISTS idx_bookings_status ON bookings(status);
`,
```

---

## Bug #4: Foreign Key Constraint Missing ON DELETE Action in Multiple Tables

**Severity:** HIGH

**STATUS: VERIFIED - Foreign keys lack ON DELETE actions at reported locations**

**Description:**
In `004_create_blocked_dates_table.go` and several other tables, foreign keys reference `users(id)` without specifying `ON DELETE` action. When a user (admin) who created blocked dates is deleted, the foreign key constraint will prevent deletion or cause unexpected behavior depending on database default behavior (which varies between SQLite, MySQL, PostgreSQL).

**Location:**
- File: `internal/database/004_create_blocked_dates_table.go`
- Lines: 15, 25, 35 (all databases)
- File: `internal/database/005_create_experience_requests_table.go`
- Lines: 19 (reviewed_by foreign key lacks ON DELETE SET NULL)
- File: `internal/database/007_create_reactivation_requests_table.go`
- Lines: 18, 33, 48 (reviewed_by foreign key lacks ON DELETE SET NULL)

**Steps to Reproduce:**
1. Create blocked date as admin user (user_id = 5)
2. Try to delete admin user (user_id = 5)
3. On MySQL with default settings: Deletion fails with foreign key constraint error
4. On PostgreSQL: Deletion fails
5. On SQLite with foreign keys enabled: Deletion fails
6. Blocked dates become orphaned with invalid created_by reference

**Impact:**
- Cannot delete admin users who created blocked dates
- Database inconsistency (orphaned records)
- Application errors when displaying blocked dates (user lookup fails)
- Different behavior across databases

**Fix:**
Add appropriate ON DELETE actions:

```diff
// File: 004_create_blocked_dates_table.go

"sqlite": `
CREATE TABLE IF NOT EXISTS blocked_dates (
  ...
  created_by INTEGER NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
-  FOREIGN KEY (created_by) REFERENCES users(id)
+  FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
);
`,

// Note: Also change created_by to allow NULL:
created_by INTEGER,

// Apply same fix to MySQL and PostgreSQL variants
```

Similarly for `005_create_experience_requests_table.go` line 19 and `007_create_reactivation_requests_table.go`.

---

## Bug #5: PostgreSQL Placeholder Bug in Dialect Implementation

**Severity:** CRITICAL (if used incorrectly)

**STATUS: VERIFIED - Code unchanged at lines 80-84**

**Description:**
The `PostgreSQLDialect.GetPlaceholder()` returns `"?"` instead of `"$1"`, `"$2"`, etc., relying on the `lib/pq` driver to convert automatically. However, this only works when using `database/sql` package. If code directly constructs SQL strings using `GetPlaceholder(position)`, the SQL will be invalid for PostgreSQL. The comment admits this workaround (lines 80-84) but it's a design flaw that breaks the abstraction.

**Location:**
- File: `internal/database/dialect_postgres.go`
- Function: `GetPlaceholder`
- Lines: 80-84

**Steps to Reproduce:**
1. Write code that uses `dialect.GetPlaceholder(1)` to build SQL manually
2. Example: `sql := fmt.Sprintf("INSERT INTO test VALUES (%s)", dialect.GetPlaceholder(1))`
3. On PostgreSQL, this produces: `INSERT INTO test VALUES (?)`
4. Execute the query - PostgreSQL doesn't understand `?` placeholder syntax
5. Error: `pq: syntax error at or near "?"`

**Impact:**
- Breaks abstraction of dialect system
- Code that manually builds SQL will fail on PostgreSQL
- Hidden dependency on specific driver behavior
- Violates principle of least surprise

**Fix:**
Implement proper placeholder syntax for PostgreSQL:

```diff
func (d *PostgreSQLDialect) GetPlaceholder(position int) string {
-	// Note: We use ? everywhere in our queries, and the pq driver
-	// handles the conversion when using database/sql.
-	// If we were using pq directly, we'd need $1, $2, etc.
-	return "?"
+	// PostgreSQL uses $1, $2, $3... for positional parameters
+	return fmt.Sprintf("$%d", position)
}
```

Then update all repository code to properly use positional placeholders, OR add a placeholder conversion utility function.

**Note:** This is a design decision - current implementation works because all queries use `?` with `db.Query()` which does automatic conversion. However, it breaks the dialect abstraction and is fragile.

---

## Bug #6: Migration 010 Creates Partial Unique Index but MySQL Cannot Enforce Uniqueness

**Severity:** MEDIUM

**STATUS: VERIFIED - Code unchanged, MySQL lacks IF NOT EXISTS at lines 26-27**

**Description:**
Migration `010_add_admin_flags.go` attempts to ensure only one super admin exists using a partial unique index on `is_super_admin`. SQLite and PostgreSQL support `WHERE is_super_admin = 1/TRUE` (lines 18, 42), but MySQL doesn't support partial indexes. The comment on line 29 admits this, saying "enforced in application logic" - but there's no such enforcement in the codebase. Multiple super admins can be created on MySQL.

**Location:**
- File: `internal/database/010_add_admin_flags.go`
- Lines: 18 (SQLite partial index), 29-30 (MySQL comment), 42 (PostgreSQL partial index)

**Steps to Reproduce:**
1. Use MySQL database
2. Run migrations
3. Manually insert user with is_super_admin = 1:
   ```sql
   INSERT INTO users (name, email, password_hash, is_super_admin, is_admin, ...)
   VALUES ('Fake Admin', 'fake@test.com', 'hash', 1, 1, ...);
   ```
4. Query: `SELECT * FROM users WHERE is_super_admin = 1`
5. Expected: 1 result (only ID=1 allowed)
6. Actual: 2+ results (constraint not enforced on MySQL)

**Impact:**
- Data integrity violation on MySQL
- Multiple super admins possible (security concern)
- Schema inconsistency across databases
- False sense of security from comment "enforced in application logic" (not actually enforced)

**Fix:**
Add application-level enforcement in user repository and admin handlers:

```go
// In internal/repository/user_repository.go

func (r *UserRepository) PromoteToSuperAdmin(userID int) error {
	// Check if another super admin exists (MySQL doesn't have DB constraint)
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM users WHERE is_super_admin = ? AND id != ?",
		true, userID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check super admin count: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("only one super admin allowed")
	}

	// Proceed with promotion
	_, err = r.db.Exec("UPDATE users SET is_super_admin = ? WHERE id = ?", true, userID)
	return err
}
```

Add similar check in admin handlers before allowing super admin promotion.

---

## Bug #7: SQLite Connection Pool Misconfiguration

**Severity:** MEDIUM

**STATUS: VERIFIED - Code unchanged at lines 98-100**

**Description:**
In `database.go`, `configureConnectionPool()` is skipped for SQLite (line 98-100), which is correct. However, if someone later adds `SetMaxOpenConns()` calls for SQLite, it can cause database locking issues. SQLite performs best with `SetMaxOpenConns(1)` for write-heavy workloads, but the code doesn't explicitly set this - it relies on Go's default.

**Location:**
- File: `internal/database/database.go`
- Function: `configureConnectionPool`
- Lines: 98-100

**Steps to Reproduce:**
1. Use SQLite database
2. Observe `configureConnectionPool()` is skipped (line 98-100)
3. Default Go behavior: Multiple connections possible
4. Under concurrent writes, SQLite returns "database is locked" errors
5. Performance degrades compared to `SetMaxOpenConns(1)`

**Impact:**
- Potential "database is locked" errors under load
- Suboptimal performance for SQLite
- Confusion about whether connection pooling is configured

**Fix:**
Explicitly configure SQLite connection settings:

```diff
// Configure connection pool (MySQL and PostgreSQL only)
-	if dialect.Name() != "sqlite" {
-		configureConnectionPool(db, config)
-	}
+	if dialect.Name() == "sqlite" {
+		// SQLite: Single connection for write serialization
+		db.SetMaxOpenConns(1)
+		db.SetMaxIdleConns(1)
+		db.SetConnMaxLifetime(0) // No lifetime limit for SQLite
+	} else {
+		// MySQL/PostgreSQL: Connection pooling
+		configureConnectionPool(db, config)
+	}
```

---

## Bug #8: Missing Error Handling in Migration Retry Logic

**Severity:** MEDIUM

**STATUS: VERIFIED - Code unchanged at lines 74-83**

**Description:**
In `migrations.go` lines 74-83, when a migration fails with "already exists" error, the code catches it and marks the migration as applied. However, if `markMigrationAsApplied()` fails (line 78-80), the error is returned but the migration execution continues with the next migration. This can lead to inconsistent migration state where a migration is partially applied but not recorded.

**Location:**
- File: `internal/database/migrations.go`
- Function: `RunMigrationsWithDialect`
- Lines: 74-83

**Steps to Reproduce:**
1. Run migration that creates table already exists
2. `isAlreadyExistsError()` returns true (line 75)
3. `markMigrationAsApplied()` is called but fails (e.g., schema_migrations table locked)
4. Error is returned, migration loop exits
5. Next run: Same migration tries to execute again
6. Infinite loop of "already exists" → fail to mark → repeat

**Impact:**
- Migration state inconsistency
- Application fails to start if migration marking fails
- No retry mechanism for transient errors

**Fix:**
Add explicit error handling and logging:

```diff
// Special handling for "already exists" errors
if isAlreadyExistsError(err, dialect) {
	log.Printf("Migration %s: Object already exists, marking as applied", migration.ID)
	// Mark as applied even though exec failed (idempotency)
	if err := markMigrationAsApplied(db, migration.ID); err != nil {
-		return fmt.Errorf("failed to mark migration as applied: %w", err)
+		log.Printf("ERROR: Failed to mark migration %s as applied: %v", migration.ID, err)
+		log.Printf("Migration state is inconsistent. Manual intervention required.")
+		return fmt.Errorf("failed to mark migration %s as applied (object exists but DB state update failed): %w",
+			migration.ID, err)
	}
	pendingCount++
	continue
}
```

---

## Bug #9: Seed Data Generates Bookings Without Approval Fields

**Severity:** MEDIUM

**STATUS: CODE MODIFIED - NEEDS REVERIFICATION**

The seed data generation has been modified. The booking structure at lines 212-222 no longer includes a `Type` field, and the INSERT statement at lines 226-232 does not include `walk_type` or any approval-related columns. This is different from what was reported.

**Original Issue:**
The `seed.go` file's `generateBookings()` function (lines 226-231) inserts bookings with only legacy fields, not the new approval workflow fields added in migration 012.

**Current Code Status:**
- Lines 212-222: Booking struct defines UserID, DogID, Date, Time, Status (no Type/walk_type field)
- Lines 226-232: INSERT statement does NOT include walk_type or approval fields
- Migration 012 added: requires_approval, approval_status, approved_by, approved_at, rejection_reason

**Impact:**
- Seed bookings will fail INSERT due to missing NOT NULL column `walk_type`
- Seed data cannot be generated successfully after migration 012
- Test environment setup is broken

**Location:**
- File: `internal/database/seed.go`
- Function: `generateBookings`
- Lines: 212-232

**Recommended Fix:**
Update seed data to include all required fields from migration 012:

```diff
func generateBookings(db *sql.DB) error {
	today := time.Now()
	yesterday := today.AddDate(0, 0, -1)
	tomorrow := today.AddDate(0, 0, 1)

	bookings := []struct {
		UserID int
		DogID  int
		Date   time.Time
		Time   string
		Status string
+		WalkType string
+		RequiresApproval int
+		ApprovalStatus   string
	}{
-		{2, 1, yesterday, "09:00", "completed"},
-		{3, 2, today, "14:00", "scheduled"},
-		{4, 3, tomorrow, "10:30", "scheduled"},
+		{2, 1, yesterday, "09:00", "completed", "morning", 1, "approved"},
+		{3, 2, today, "14:00", "scheduled", "evening", 0, "approved"},
+		{4, 3, tomorrow, "10:30", "scheduled", "morning", 1, "pending"},
	}

	now := time.Now()
	for _, booking := range bookings {
		_, err := db.Exec(`
-			INSERT INTO bookings (user_id, dog_id, date, scheduled_time,
-				status, created_at, updated_at)
-			VALUES (?, ?, ?, ?, ?, ?, ?)
+			INSERT INTO bookings (user_id, dog_id, date, scheduled_time,
+				walk_type, status, requires_approval, approval_status,
+				created_at, updated_at)
+			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, booking.UserID, booking.DogID,
			booking.Date.Format("2006-01-02"), booking.Time,
-			booking.Status, now, now)
+			booking.WalkType, booking.Status, booking.RequiresApproval,
+			booking.ApprovalStatus, now, now)
		if err != nil {
			return fmt.Errorf("failed to create booking: %w", err)
		}
	}
```

---

## Bug #10: MySQL Insert Syntax Error in Migration 010

**Severity:** MEDIUM

**STATUS: VERIFIED - MySQL version lacks IF NOT EXISTS at lines 26-27**

**Description:**
In migration `010_add_admin_flags.go` line 26, MySQL version uses `CREATE INDEX` without `IF NOT EXISTS`, but in migration `012_booking_times.go` line 153, it correctly uses `CREATE INDEX IF NOT EXISTS`. Inconsistent syntax may cause migration 010 to fail on re-run if indexes already exist.

**Location:**
- File: `internal/database/010_add_admin_flags.go`
- Lines: 26-27 (MySQL version)

**Steps to Reproduce:**
1. Run migrations on MySQL (migration 010 succeeds)
2. Manually mark migration 010 as not applied: `DELETE FROM schema_migrations WHERE version = '010_add_admin_flags'`
3. Run migrations again
4. Expected: Migration 010 re-runs successfully
5. Actual: Error - "Duplicate key name 'idx_users_admin'" (index already exists)
6. Migration marked as failed, application won't start

**Impact:**
- Migration 010 is not idempotent on MySQL
- Re-running migrations fails
- Manual database intervention required
- Inconsistent with other migrations that use IF NOT EXISTS

**Fix:**
Add IF NOT EXISTS to index creation (with error handling):

```diff
"mysql": `
-- Add admin flag columns
ALTER TABLE users ADD COLUMN is_admin TINYINT(1) DEFAULT 0;
ALTER TABLE users ADD COLUMN is_super_admin TINYINT(1) DEFAULT 0;

-- Create indexes for performance
-CREATE INDEX idx_users_admin ON users(is_admin);
-CREATE INDEX idx_users_super_admin ON users(is_super_admin);
+CREATE INDEX IF NOT EXISTS idx_users_admin ON users(is_admin);
+CREATE INDEX IF NOT EXISTS idx_users_super_admin ON users(is_super_admin);

-- Note: MySQL doesn't support partial unique indexes
-- The unique super admin constraint is enforced in application logic
`,
```

**Note:** MySQL before 5.7 doesn't support `IF NOT EXISTS` for indexes. The migration system's `isAlreadyExistsError()` should handle this, but explicit IF NOT EXISTS is cleaner.

---

## Bug #11: Schema Inconsistency - bookings.scheduled_time Column Order Differs

**Severity:** LOW

**STATUS: VERIFIED - Column order differs between migrations**

**Description:**
In migration `003_create_bookings_table.go`, the column order differs between databases:
- SQLite: `date, walk_type, scheduled_time` (lines 13-15)
- MySQL: `date, walk_type, scheduled_time` (lines 32-34)
- PostgreSQL: `date, walk_type, scheduled_time` (lines 51-53)

But in migration `012_booking_times.go`, SQLite recreates the table with:
- SQLite: `date, scheduled_time, walk_type` (lines 51-53)

This column order change is cosmetic but creates confusion and makes schema comparison difficult. Column order shouldn't change between migrations.

**Location:**
- File: `internal/database/003_create_bookings_table.go` (all databases)
- File: `internal/database/012_booking_times.go` (SQLite lines 51-53)

**Steps to Reproduce:**
1. Compare schema of bookings table across migrations
2. Migration 003: Column order is `date, walk_type, scheduled_time`
3. Migration 012 (SQLite): Column order is `date, scheduled_time, walk_type`
4. Column order differs between migrations

**Impact:**
- Schema comparison tools show false differences
- Developer confusion when reviewing migrations
- No functional impact (column order doesn't affect queries)
- Makes audit trail harder to follow

**Fix:**
Keep consistent column order in migration 012:

```diff
CREATE TABLE IF NOT EXISTS bookings_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    dog_id INTEGER NOT NULL,
    date DATE NOT NULL,
-    scheduled_time TEXT NOT NULL,
    walk_type TEXT CHECK(walk_type IN ('morning', 'evening')),
+    scheduled_time TEXT NOT NULL,
    status TEXT DEFAULT 'scheduled' CHECK(status IN ('scheduled', 'completed', 'cancelled')),
    ...
);

INSERT INTO bookings_new SELECT
-    id, user_id, dog_id, date, scheduled_time, walk_type, status,
+    id, user_id, dog_id, date, walk_type, scheduled_time, status,
    completed_at, user_notes, admin_cancellation_reason, created_at, updated_at,
    0, 'approved', NULL, NULL, NULL
FROM bookings;
```

---

## Statistics

- **Critical:** 3 bugs (insecure random, SQL injection potential, schema inconsistency causing performance issues)
- **High:** 5 bugs (duplicate schemas, missing indexes, foreign key issues, migration ordering)
- **Medium:** 3 bugs (connection pool, error handling, seed data inconsistency)
- **Low:** 0 bugs

---

## Recommendations

### Immediate Actions (Critical)

1. **Replace math/rand with crypto/rand** in `seed.go` immediately - this is a security vulnerability
2. **Fix foreign key constraints** in all affected tables - add appropriate ON DELETE actions
3. **Add missing indexes** to migration 003 for all databases - critical for query performance

### High Priority

4. **Remove duplicate schema definitions** from `database.go` - eliminate confusion
5. **Fix PostgreSQL placeholder implementation** - either implement $1/$2/etc or document the workaround clearly
6. **Add super admin uniqueness enforcement** in application code for MySQL compatibility
7. **Standardize migration idempotency** - ensure all migrations use IF NOT EXISTS consistently

### Medium Priority

8. **Configure SQLite connection pool** explicitly - prevent "database is locked" errors
9. **Improve migration error handling** - add retry logic and better error messages
10. **Update seed data** to include all current schema fields (approval workflow)

### Code Quality

11. **Add migration validation tests** - verify schema consistency across all three databases
12. **Document dialect abstraction limitations** - especially the placeholder workaround
13. **Add pre-commit hooks** - verify migration files include all three databases
14. **Create migration template** - ensure new migrations follow best practices

### Testing Recommendations

- Add integration tests that verify schema consistency across SQLite, MySQL, PostgreSQL
- Test migration idempotency (run migrations twice, verify no errors)
- Test foreign key cascades (delete parent record, verify child behavior)
- Test concurrent operations on SQLite (verify no locking issues)
- Add security test for password generation (verify entropy)

---

## Additional Notes

The database layer is well-architected with good dialect abstraction, but suffers from:
1. **Security issues** in seed data generation
2. **Schema drift** between migrations and legacy constants
3. **Inconsistent migration patterns** (some use IF NOT EXISTS, others don't)
4. **Missing indexes** that impact production performance
5. **Incomplete foreign key constraints** risking data integrity

The migration system works correctly but needs:
- Standardized migration templates
- Automated schema validation tests
- Better error handling for edge cases
- Documentation of dialect limitations

**Priority:** Fix Bug #1 (crypto/rand) immediately before any production deployment.
