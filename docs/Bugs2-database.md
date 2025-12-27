# Bug Report: database

**Analysis Date:** 2025-12-27
**Directory Analyzed:** `/home/tranmh/work/gassigeher-saas/internal/database`
**Files Analyzed:** 16 files
**Bugs Found:** 12 bugs

---

## Summary

The database directory contains critical bugs in migration handling, schema consistency, and SQL errors across the three database backends (SQLite, MySQL, PostgreSQL). Key issues include:

- **Critical**: SQLite ALTER TABLE migration failures (will always fail on existing databases)
- **Critical**: PostgreSQL placeholder syntax incompatibility
- **High**: MySQL index creation race conditions
- **High**: Schema inconsistencies between migration 001 and 002
- **High**: Missing TIMESTAMP type handling in PostgreSQL
- **Medium**: Seed data assumes tenant context incorrectly

Most critical are the migration bugs that will cause deployment failures and data integrity issues when running migrations on existing databases.

---

## Bugs

## Bug #1: SQLite ALTER TABLE Always Fails in Migration 002

**Severity:** CRITICAL

**Description:**
Migration `002_marketing_and_branding.go` uses `ALTER TABLE tenant_settings ADD COLUMN` without the `IF NOT EXISTS` clause for SQLite. The migration comments acknowledge this issue ("SQLite doesn't support IF NOT EXISTS for columns") but still uses unconditional ALTER TABLE. This **will always fail** when running migration 002 on a database where migration 001 already created the `tenant_settings` table with these columns.

The migration system catches the error and marks it as applied (lines 74-83 in `migrations.go`), but this masks real problems and creates inconsistent state.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/database/002_marketing_and_branding.go`
- Lines: 17-21

**Impact:**
- Migration 002 always throws errors on existing databases
- Error is silently caught and logged as "already exists"
- Actual schema problems could be hidden
- Logs filled with spurious errors
- Confusion for developers and operators

**Steps to Reproduce:**
1. Initialize fresh database (runs migration 001)
2. Schema includes `tenant_settings` with `tagline` and `description` from 001
3. Run migration 002
4. ALTER TABLE tries to add columns that already exist
5. SQLite error: "duplicate column name"
6. Error is caught and migration marked as applied (but didn't actually run)

**Fix:**
Remove the redundant ALTER TABLE statements from migration 002 for SQLite. The columns already exist from migration 001. If they're needed for older databases, check column existence first:

```diff
"sqlite": `
--- Add tagline column if not exists
-ALTER TABLE tenant_settings ADD COLUMN tagline TEXT;
-
--- Add description column if not exists
-ALTER TABLE tenant_settings ADD COLUMN description TEXT;
+-- Note: These columns already exist from migration 001
+-- No need to add them again

-- Marketing campaigns table (if not exists from 001)
CREATE TABLE IF NOT EXISTS marketing_campaigns (
```

Alternatively, use a proper idempotent approach:

```sql
-- Check if column exists before adding (SQLite-specific)
SELECT CASE
  WHEN COUNT(*) = 0 THEN
    'ALTER TABLE tenant_settings ADD COLUMN tagline TEXT'
  ELSE 'SELECT 1'
END as sql_statement
FROM pragma_table_info('tenant_settings')
WHERE name = 'tagline';
```

---

## Bug #2: PostgreSQL Placeholder Syntax Mismatch

**Severity:** CRITICAL

**Description:**
The PostgreSQL dialect's `GetPlaceholder()` method returns `"?"` (line 84 in `dialect_postgres.go`), claiming the `lib/pq` driver converts `?` to `$1`, `$2`, etc. automatically. **This is FALSE**. The `lib/pq` driver does **NOT** automatically convert question mark placeholders to dollar sign placeholders.

The comment on lines 77-83 is misleading. While Go's `database/sql` package provides parameter handling, the `lib/pq` driver specifically requires `$1`, `$2` syntax. Using `?` will cause SQL syntax errors.

This bug affects the `markMigrationAsApplied()` function which uses a hardcoded switch statement (lines 156-163 in `migrations.go`) to work around this, but it means all other code using the dialect interface will fail.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/database/dialect_postgres.go`
- Function: `GetPlaceholder`
- Lines: 80-85

**Impact:**
- Any code using `dialect.GetPlaceholder()` with PostgreSQL will generate invalid SQL
- Migration system works around this with hardcoded logic (bad design)
- Dialect interface is broken - violates its contract
- PostgreSQL queries will fail with syntax errors

**Steps to Reproduce:**
1. Use PostgreSQL dialect
2. Build query using `dialect.GetPlaceholder(1)` → Returns `"?"`
3. Execute query: `INSERT INTO table (col) VALUES (?)`
4. PostgreSQL error: `syntax error at or near "?"`
5. Expected: Should use `$1` syntax

**Fix:**
Correct the `GetPlaceholder()` method to return PostgreSQL's actual syntax:

```diff
func (d *PostgreSQLDialect) GetPlaceholder(position int) string {
-	// Note: We use ? everywhere in our queries, and the pq driver
-	// handles the conversion when using database/sql.
-	// If we were using pq directly, we'd need $1, $2, etc.
-	return "?"
+	// PostgreSQL requires $1, $2, $3, ... syntax
+	// The lib/pq driver does NOT convert ? to $n
+	return fmt.Sprintf("$%d", position)
}
```

Then update all calling code to use the dialect's placeholder correctly, or implement a query builder that handles substitution.

**Note:** The hardcoded workaround in `markMigrationAsApplied()` (lines 156-163) should be removed and use the dialect method instead.

---

## Bug #3: MySQL Index Creation Race Condition

**Severity:** HIGH

**Description:**
The `createIndexIfNotExists()` function (lines 255-288 in `migrations.go`) checks if a MySQL index exists by querying `information_schema.statistics`, then creates it if not found. This is a classic **time-of-check-time-of-use (TOCTOU)** race condition.

If two migration processes run simultaneously (e.g., in a cluster deployment or during a race in startup), both might check at the same time, both see no index, and both try to create it. The second will fail with a duplicate key error.

Additionally, the function returns non-fatal errors with `fmt.Errorf()` (line 268), but the caller treats ANY error as non-fatal (line 126). This could hide real permission issues or syntax errors.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/database/migrations.go`
- Function: `createIndexIfNotExists`
- Lines: 255-277

**Impact:**
- Race condition in concurrent deployments
- Second process gets spurious error
- Could cause deployment to fail if error handling changes
- Hides real errors (permission denied, syntax errors, etc.)

**Steps to Reproduce:**
1. Start two server instances simultaneously against same MySQL database
2. Both run migrations at same time
3. Both check for index existence
4. Both see COUNT(*) = 0
5. Both try CREATE INDEX
6. Second fails with "Duplicate key name"

**Fix:**
Use MySQL's `CREATE INDEX ... IF NOT EXISTS` syntax (available in MySQL 5.7.26+) or catch the duplicate key error:

```diff
func createIndexIfNotExists(db *sql.DB, dialect Dialect, indexName, tableName, columnName string) error {
	switch dialect.Name() {
	case "mysql":
-		// MySQL: Check if index exists first using information_schema
-		var count int
-		err := db.QueryRow(`
-			SELECT COUNT(*) FROM information_schema.statistics
-			WHERE table_schema = DATABASE()
-			AND table_name = ?
-			AND index_name = ?`, tableName, indexName).Scan(&count)
-		if err != nil {
-			return fmt.Errorf("failed to check index existence: %w", err)
-		}
-		if count > 0 {
-			// Index already exists
-			return nil
-		}
-		// Create the index
-		_, err = db.Exec(fmt.Sprintf("CREATE INDEX %s ON %s(%s)", indexName, tableName, columnName))
-		return err
+		// MySQL: Try to create, ignore duplicate error
+		_, err := db.Exec(fmt.Sprintf("CREATE INDEX %s ON %s(%s)", indexName, tableName, columnName))
+		if err != nil && strings.Contains(err.Error(), "Duplicate key name") {
+			// Index already exists - this is fine
+			return nil
+		}
+		return err
```

Or if you need MySQL < 5.7.26 compatibility, wrap in proper error handling:

```go
case "mysql":
	_, err := db.Exec(fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s(%s)", indexName, tableName, columnName))
	if err != nil {
		// Check if it's a "syntax error" (IF NOT EXISTS not supported)
		if strings.Contains(err.Error(), "syntax error") {
			// Fall back to check-then-create with better error handling
			// ... (but still has race condition)
		}
		return err
	}
	return nil
```

---

## Bug #4: Schema Inconsistency Between Migrations 001 and 002

**Severity:** HIGH

**Description:**
Migration `001_schema.go` defines `tenant_settings` table with columns `tagline` (line 42) and `description` (line 43) for all three databases. Migration `002_marketing_and_branding.go` then tries to ADD these same columns again (lines 18-21 for SQLite, lines 84-85 for MySQL, lines 149-150 for PostgreSQL).

This creates confusion: which migration owns these columns? If migration 001 runs, the columns exist. If only migration 002 runs (impossible, but architecturally confusing), it would try to add them.

For MySQL and PostgreSQL, the `ADD COLUMN IF NOT EXISTS` syntax handles this gracefully, but it's still a schema design smell indicating these columns should only be defined in ONE place.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/database/001_schema.go`
- Lines: 42-43 (and equivalents in MySQL/Postgres sections)
- File: `/home/tranmh/work/gassigeher-saas/internal/database/002_marketing_and_branding.go`
- Lines: 18-21, 84-85, 149-150

**Impact:**
- Schema ownership unclear
- Migration 002 comments suggest these are "missing" columns, but they exist from 001
- Confusing for developers maintaining migrations
- SQLite version fails (Bug #1)
- If migration 001 is ever modified to remove these, migration 002 becomes the owner, but it will fail on existing DBs

**Steps to Reproduce:**
1. Read migration 001 → `tenant_settings` has `tagline` and `description`
2. Read migration 002 → Tries to add `tagline` and `description`
3. Question: Which migration owns these columns?
4. Answer: Unclear, depends on database type and timing

**Fix:**
Remove the duplicate column additions from migration 002. These columns already exist from migration 001:

```diff
# File: 002_marketing_and_branding.go

"sqlite": `
--- Add missing columns to tenant_settings (tagline and description)
--- SQLite doesn't support IF NOT EXISTS for columns, so we use a trick
--- by creating a new table if the column doesn't exist
-
--- First check if tagline exists by trying to select it
--- If this migration runs on a fresh DB, these tables already exist from 001
--- If it runs on an existing DB, we need to add them
-
--- Add tagline column if not exists
-ALTER TABLE tenant_settings ADD COLUMN tagline TEXT;
-
--- Add description column if not exists
-ALTER TABLE tenant_settings ADD COLUMN description TEXT;
+-- Note: tagline and description columns already exist from migration 001
+-- No action needed

-- Marketing campaigns table (if not exists from 001)
CREATE TABLE IF NOT EXISTS marketing_campaigns (
```

Same for MySQL and PostgreSQL sections. The comments should clarify that migration 002 is ONLY for marketing tables, not branding columns.

---

## Bug #5: PostgreSQL TIMESTAMP Type Inconsistency

**Severity:** HIGH

**Description:**
The PostgreSQL dialect's `GetTimestampType()` returns `"TIMESTAMP WITH TIME ZONE"` (line 62 in `dialect_postgres.go`), but migration `001_schema.go` uses plain `TIMESTAMP` in multiple places (e.g., line 904, 935, 964, etc.).

This inconsistency means:
1. Migrations create columns with plain `TIMESTAMP` (no timezone)
2. Dialect interface claims to use `TIMESTAMP WITH TIME ZONE`
3. Any code using the dialect to build DDL will use the wrong type
4. Data stored in existing columns won't have timezone info

PostgreSQL's best practice is to use `TIMESTAMP WITH TIME ZONE` for UTC storage, which the dialect correctly recommends. But the migrations don't follow this.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/database/dialect_postgres.go`
- Lines: 60-63
- File: `/home/tranmh/work/gassigeher-saas/internal/database/001_schema.go`
- Lines: 904-905, 935-936, 964-965 (and many more)

**Impact:**
- Timezone information lost in PostgreSQL timestamps
- Dialect interface lies about column types
- Inconsistent behavior between migration-created tables and code-created tables
- Queries involving time comparisons may have subtle bugs
- Daylight saving time handling incorrect

**Steps to Reproduce:**
1. Run migration 001 on PostgreSQL
2. Inspect `tenants` table: `created_at TIMESTAMP` (no timezone)
3. Call `dialect.GetTimestampType()` → Returns `"TIMESTAMP WITH TIME ZONE"`
4. Discrepancy: Migration uses one type, dialect claims another

**Fix:**
Update migration 001 to use `TIMESTAMP WITH TIME ZONE` for all timestamp columns in PostgreSQL:

```diff
# File: 001_schema.go, postgres section

CREATE TABLE IF NOT EXISTS tenants (
  id SERIAL PRIMARY KEY,
  slug VARCHAR(100) NOT NULL UNIQUE,
  ...
-  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
-  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
+  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
+  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

Apply this change to ALL timestamp columns in the PostgreSQL section of migration 001. This is a breaking change for existing databases, so you may need a migration to ALTER existing columns:

```sql
-- New migration: 00X_fix_postgres_timestamps.go (postgres only)
ALTER TABLE tenants ALTER COLUMN created_at TYPE TIMESTAMP WITH TIME ZONE;
ALTER TABLE tenants ALTER COLUMN updated_at TYPE TIMESTAMP WITH TIME ZONE;
-- (repeat for all tables with timestamp columns)
```

---

## Bug #6: Migration Tracking Uses Wrong Placeholder Syntax

**Severity:** HIGH

**Description:**
The `markMigrationAsApplied()` function (lines 153-168 in `migrations.go`) uses a hardcoded switch statement to generate PostgreSQL-specific queries with `$1, $2` placeholders. This **duplicates logic** already in the dialect interface and **violates the dialect abstraction**.

If the dialect interface is fixed (Bug #2), this function should use `dialect.GetPlaceholder()` instead of hardcoding the logic. The current implementation means the dialect interface is useless - all code must hardcode database-specific behavior.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/database/migrations.go`
- Function: `markMigrationAsApplied`
- Lines: 153-168

**Impact:**
- Dialect abstraction is broken
- Code duplication (placeholder logic exists in two places)
- If dialect is fixed, this code must also be updated
- Adding a new database requires updating multiple places
- Maintenance burden

**Steps to Reproduce:**
1. Review `markMigrationAsApplied()` function
2. Notice hardcoded switch on `dialect.Name()`
3. Notice PostgreSQL uses `$1, $2` syntax
4. Compare to `dialect.GetPlaceholder()` which returns `"?"`
5. Realize dialect interface is not being used correctly

**Fix:**
Remove hardcoded switch and use dialect interface properly:

```diff
func markMigrationAsApplied(db *sql.DB, dialect Dialect, migrationID string) error {
-	var query string
-	switch dialect.Name() {
-	case "postgres":
-		query = "INSERT INTO schema_migrations (version, applied_at) VALUES ($1, $2)"
-	default:
-		query = "INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)"
-	}
-	_, err := db.Exec(query, migrationID, time.Now())
+	// Build query using dialect placeholders
+	query := fmt.Sprintf("INSERT INTO schema_migrations (version, applied_at) VALUES (%s, %s)",
+		dialect.GetPlaceholder(1),
+		dialect.GetPlaceholder(2))
+
+	_, err := db.Exec(query, migrationID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to insert migration record: %w", err)
	}
	return nil
}
```

**Note:** This fix depends on fixing Bug #2 first (PostgreSQL placeholder syntax).

---

## Bug #7: Seed Data Assumes Tenant Context Incorrectly

**Severity:** MEDIUM

**Description:**
The `SeedDatabase()` function (lines 28-101 in `seed.go`) creates a "Central Admin" user and then skips creating test users, dogs, and bookings with the comment "In SaaS mode, we skip test users/dogs/bookings since they don't belong to any tenant" (line 81-82).

However, the seed functions (`generateTestUsers`, `generateDogs`, `generateBookings`) are still defined and insert data **without setting `tenant_id`**. If these functions were accidentally called, they would insert data with `NULL` tenant_id, violating multi-tenancy isolation.

Additionally, the comment on line 87 says "Demo tenant will be created by DemoSeedService on startup", but there's no error handling if DemoSeedService fails. The seed function should either:
1. Create the demo tenant itself, OR
2. Verify it was created, OR
3. Not reference it at all

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/database/seed.go`
- Function: `SeedDatabase`
- Lines: 81-88
- Functions: `generateTestUsers` (148-178), `generateDogs` (182-288), `generateBookings` (373-407)

**Impact:**
- If seed functions are called by mistake, data created without tenant_id
- Orphaned data in multi-tenant system
- Violates tenant isolation
- Comment references external dependency (DemoSeedService) without verification
- Unclear ownership of demo tenant creation

**Steps to Reproduce:**
1. Review `generateTestUsers()` function
2. Notice INSERT statement on lines 163-168 does not set `tenant_id`
3. If this function were called, it would insert users with `tenant_id = NULL`
4. Same for `generateDogs()` (line 271-280) and `generateBookings()` (line 393-398)
5. Multi-tenancy broken

**Fix:**
Either remove the unused seed functions entirely, or add tenant_id parameters:

```diff
# Option 1: Remove unused functions (since they're not called)
-// generateTestUsers creates 3 test users with different color levels
-// Level field is used for assigning colors after creation
-// DONE
-func generateTestUsers(db *sql.DB) ([]TestUser, error) {
-	// ... entire function ...
-}

# Option 2: Add tenant_id parameter if they might be used
-func generateTestUsers(db *sql.DB) ([]TestUser, error) {
+func generateTestUsers(db *sql.DB, tenantID int) ([]TestUser, error) {
	users := []TestUser{
		// ...
	}

	now := time.Now()
	for i := range users {
		// ...
		_, err = db.Exec(`
			INSERT INTO users (
+				tenant_id,
				first_name, last_name, email, password_hash,
				is_admin, is_super_admin, is_verified, is_active,
				terms_accepted_at, last_activity_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
-		`, users[i].FirstName, users[i].LastName, users[i].Email, string(hashedPassword),
+		`, tenantID, users[i].FirstName, users[i].LastName, users[i].Email, string(hashedPassword),
			false, false, true, true, now, now, now, now)
```

Apply same fix to `generateDogs()` and `generateBookings()`.

**Recommendation:** Since these functions are never called (line 83 skips them), remove them entirely to avoid confusion.

---

## Bug #8: Missing Unique Constraint Validation in Migration 001

**Severity:** MEDIUM

**Description:**
Migration `001_schema.go` defines several unique constraints using the `UNIQUE(columns)` syntax, but some of these are inconsistent across database types. Specifically:

**MySQL unique constraints** (lines 520, 537, 557, 593, 614, 629, etc.):
- Uses `UNIQUE KEY idx_name (columns)` syntax
- Gives index a name explicitly

**PostgreSQL unique constraints** (lines 978, 995, 1048, 1068, 1078, etc.):
- Uses `UNIQUE(columns)` inline in table definition
- No explicit index name
- Then creates a named unique index separately (e.g., line 978)

**SQLite unique constraints** (lines 99, 116, 169, 189, 199, etc.):
- Uses `UNIQUE(columns)` inline
- Then creates named unique index separately

This inconsistency means:
1. PostgreSQL and SQLite create the constraint twice (inline + named index)
2. Some unique constraints might not be enforced correctly
3. Index names differ across databases (PostgreSQL auto-generates, others use explicit names)

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/database/001_schema.go`
- Lines: Various (99, 116, 169, 520, 978, etc.)

**Impact:**
- Duplicate unique constraints (waste of space)
- Inconsistent index names across databases
- Queries optimized for one database might not work on another
- Confusion when inspecting schema

**Steps to Reproduce:**
1. Run migration 001 on PostgreSQL
2. Inspect `users` table constraints
3. Find both inline UNIQUE constraint AND separate unique index
4. Compare to MySQL which only has one

**Fix:**
Standardize unique constraint definitions. Either:

**Option 1:** Use inline UNIQUE without separate index:

```diff
# PostgreSQL
CREATE TABLE IF NOT EXISTS users (
  id SERIAL PRIMARY KEY,
  ...
  UNIQUE(tenant_id, email)
);
-CREATE UNIQUE INDEX IF NOT EXISTS idx_users_tenant_email ON users(COALESCE(tenant_id, 0), email) WHERE email IS NOT NULL;
+-- Unique constraint already defined inline
```

**Option 2:** Use only named unique indexes (preferred for PostgreSQL partial indexes):

```diff
# PostgreSQL
CREATE TABLE IF NOT EXISTS users (
  id SERIAL PRIMARY KEY,
  ...
-  UNIQUE(tenant_id, email)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_tenant_email ON users(COALESCE(tenant_id, 0), email) WHERE email IS NOT NULL;
```

Apply consistently across all three databases.

---

## Bug #9: Foreign Key Cascade Behavior Inconsistency

**Severity:** MEDIUM

**Description:**
Foreign key cascade behaviors differ between database types in migration `001_schema.go`. For example, the `user_colors` table:

**SQLite** (line 206):
```sql
color_id INTEGER NOT NULL REFERENCES color_categories(id) ON DELETE RESTRICT,
```

**MySQL** (line 647):
```sql
FOREIGN KEY (color_id) REFERENCES color_categories(id) ON DELETE RESTRICT,
```

**PostgreSQL** (line 1085):
```sql
color_id INTEGER NOT NULL REFERENCES color_categories(id) ON DELETE RESTRICT,
```

This is consistent, BUT other tables have inconsistencies. For example, `walk_report_photos`:

**SQLite** (line 277):
```sql
walk_report_id INTEGER NOT NULL REFERENCES walk_reports(id) ON DELETE CASCADE,
```

**MySQL** (line 733):
```sql
FOREIGN KEY (walk_report_id) REFERENCES walk_reports(id) ON DELETE CASCADE
```

**PostgreSQL** (line 1156):
```sql
walk_report_id INTEGER NOT NULL REFERENCES walk_reports(id) ON DELETE CASCADE,
```

All are CASCADE here, which is correct. However, there's no systematic verification that all FK cascade behaviors match across the three schemas. A diff would likely reveal discrepancies.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/database/001_schema.go`
- Lines: Throughout schema definitions

**Impact:**
- Data deletion behavior differs across databases
- In SQLite, deleting a color_category might fail (RESTRICT)
- In PostgreSQL, same deletion might cascade and delete user_colors
- Unexpected data loss or FK constraint violations
- Tests passing on one DB, failing on another

**Steps to Reproduce:**
1. Create database on SQLite with migration 001
2. Insert: tenant, color_category, user, user_colors
3. Try to DELETE color_category
4. Result: RESTRICT prevents deletion
5. Repeat on PostgreSQL
6. If CASCADE is used instead of RESTRICT, deletion succeeds and cascades
7. Different behaviors

**Fix:**
Systematically review ALL foreign keys in migration 001 and ensure cascade behaviors match. Create a checklist:

- `ON DELETE CASCADE` - Used when child records should be deleted with parent
- `ON DELETE RESTRICT` - Used when deletion should fail if children exist
- `ON DELETE SET NULL` - Used when child records should be orphaned

Document the intended behavior for each relationship, then verify all three database definitions match.

**Recommendation:** Use a schema diff tool to compare SQLite, MySQL, and PostgreSQL versions of the migration and flag any differences in FK behavior.

---

## Bug #10: Boolean Default Values Inconsistent in Migration 001

**Severity:** MEDIUM

**Description:**
Boolean columns in migration `001_schema.go` use different default value syntaxes across databases:

**SQLite** (line 21):
```sql
is_demo INTEGER DEFAULT 0,
```

**MySQL** (line 441):
```sql
is_demo TINYINT(1) DEFAULT 0,
```

**PostgreSQL** (line 900):
```sql
is_demo BOOLEAN DEFAULT FALSE,
```

This is correct for the data types, BUT the issue is that some boolean defaults are numeric (`0`, `1`) and others are literal (`FALSE`, `TRUE`, `false`, `true`). For example:

**SQLite** (line 74):
```sql
is_admin INTEGER DEFAULT 0,
```

**PostgreSQL** (line 953):
```sql
is_admin BOOLEAN DEFAULT FALSE,
```

Both are correct, but mixing numeric and literal boolean representations in the same schema (e.g., some use `DEFAULT 0`, others use `DEFAULT FALSE` in PostgreSQL) creates confusion.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/database/001_schema.go`
- Lines: Throughout (21, 74, 441, 900, 953, etc.)

**Impact:**
- Inconsistent schema style
- Developers unsure which syntax to use
- Minor: No functional bug, but readability issue
- Could cause confusion when manually writing queries

**Steps to Reproduce:**
1. Review PostgreSQL section of migration 001
2. Find boolean columns with `DEFAULT FALSE`
3. Find others with `DEFAULT 0` (if any)
4. Inconsistency

**Fix:**
Standardize boolean defaults:

- **SQLite**: Always use `DEFAULT 0` or `DEFAULT 1` (INTEGER)
- **MySQL**: Always use `DEFAULT 0` or `DEFAULT 1` (TINYINT)
- **PostgreSQL**: Always use `DEFAULT FALSE` or `DEFAULT TRUE` (BOOLEAN)

Update dialect methods to generate these consistently:

```go
func (d *PostgreSQLDialect) GetBooleanDefault(value bool) string {
	if value {
		return "TRUE"  // Not "1"
	}
	return "FALSE"  // Not "0"
}
```

Then review migration 001 and ensure all boolean defaults use the correct syntax for each database.

---

## Bug #11: Missing Error Handling in Database Connection String Building

**Severity:** MEDIUM

**Description:**
The `buildMySQLDSN()` and `buildPostgreSQLDSN()` functions (lines 124-190 in `database.go`) build connection strings by simple string concatenation without validating or escaping the input values.

If `config.Username` or `config.Password` contain special characters (e.g., `@`, `:`, `/`, `?`, `&`), the connection string will be malformed and fail to parse. For example:

- Username: `user@domain`
- Password: `pass:word`
- Resulting DSN: `user@domain:pass:word@tcp(host:3306)/db`

This is invalid because the `@` in username is interpreted as the host delimiter, and the `:` in password is interpreted as a port delimiter.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/database/database.go`
- Functions: `buildMySQLDSN`, `buildPostgreSQLDSN`
- Lines: 124-154 (MySQL), 156-190 (PostgreSQL)

**Impact:**
- Connection fails if username/password contain special characters
- Cryptic error messages (e.g., "invalid DSN: missing @")
- Security risk: Passwords containing `&` might be parsed as query parameters
- No validation or escaping

**Steps to Reproduce:**
1. Set MySQL username to `user@domain`
2. Set password to `pass:word`
3. Build DSN using `buildMySQLDSN()`
4. Result: `user@domain:pass:word@tcp(localhost:3306)/gassigeher?...`
5. Parse error: Cannot determine where username ends and host begins

**Fix:**
Use URL encoding for username and password:

```diff
import (
	"database/sql"
	"fmt"
+	"net/url"
	"strings"
	"time"
	// ...
)

func buildMySQLDSN(config *DBConfig) string {
	// ... existing code ...

+	// URL-encode username and password to handle special characters
+	username := url.QueryEscape(config.Username)
+	password := url.QueryEscape(config.Password)
+
	// Build DSN
	// parseTime=true is required for scanning time.Time fields
	// charset=utf8mb4 for full Unicode support (including emoji)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci",
-		config.Username,
-		config.Password,
+		username,
+		password,
		host,
		port,
		database,
	)

	return dsn
}
```

Similar fix for PostgreSQL:

```diff
func buildPostgreSQLDSN(config *DBConfig) string {
	// ... existing code ...

+	// URL-encode username and password
+	username := url.QueryEscape(config.Username)
+	password := url.QueryEscape(config.Password)
+
	// Build PostgreSQL connection string
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
-		config.Username,
-		config.Password,
+		username,
+		password,
		host,
		port,
		database,
		sslMode,
	)

	return dsn
}
```

---

## Bug #12: Seeding Doesn't Verify Prerequisites

**Severity:** LOW

**Description:**
The `SeedDatabase()` function (lines 28-101 in `seed.go`) inserts a Central Admin user with hardcoded ID=1 (line 70), but doesn't verify that:

1. The ID isn't already taken (e.g., if seeding runs twice due to a bug)
2. The email isn't already registered (though it checks user count, race conditions exist)
3. The `users` table structure matches expectations

If the INSERT fails (e.g., due to unique constraint violation), the error is returned (line 73-75), but the partially-created state isn't rolled back. The credentials file might be written (lines 91-94) even though the user wasn't created.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/database/seed.go`
- Function: `SeedDatabase`
- Lines: 64-75, 91-94

**Impact:**
- Race condition if seeding runs twice
- Credentials file written even if user creation fails
- No transactional guarantee
- Could lead to inconsistent state

**Steps to Reproduce:**
1. Start two server instances simultaneously (race condition)
2. Both check user count (line 37-40) → both see 0
3. Both try to insert Central Admin with ID=1
4. Second fails with "UNIQUE constraint failed: users.id"
5. Second might still write credentials file

**Fix:**
Use a transaction and verify state:

```diff
func SeedDatabase(db *sql.DB, superAdminEmail string) error {
	// ... existing checks ...

+	// Begin transaction for atomic seeding
+	tx, err := db.Begin()
+	if err != nil {
+		return fmt.Errorf("failed to begin transaction: %w", err)
+	}
+	defer tx.Rollback() // Rollback if not committed
+
	// Generate Central Admin
	centralAdminPassword := generateSecurePassword(20)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(centralAdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash central admin password: %w", err)
	}

	now := time.Now()
-	_, err = db.Exec(`
+	_, err = tx.Exec(`
		INSERT INTO users (
			id, first_name, last_name, email, password_hash,
			is_admin, is_super_admin, is_central_admin, is_verified, is_active,
			terms_accepted_at, last_activity_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, 1, "Central", "Admin", superAdminEmail, string(hashedPassword),
		true, true, true, true, true, now, now, now, now)

	if err != nil {
		return fmt.Errorf("failed to create Central Admin: %w", err)
	}

+	// Commit transaction
+	if err := tx.Commit(); err != nil {
+		return fmt.Errorf("failed to commit seed transaction: %w", err)
+	}
+
	superAdminPassword := centralAdminPassword
	log.Println("✓ Central Admin created (ID: 1)")

	// ... rest of function ...
}
```

This ensures either all seeding succeeds, or none of it does (transactional consistency).

---

## Statistics

- **Critical:** 3 bugs (Migration 002 failures, PostgreSQL placeholder bug, timestamp inconsistency)
- **High:** 4 bugs (Index race condition, schema inconsistency, missing unique constraint validation, FK cascade inconsistency)
- **Medium:** 4 bugs (Seed tenant context, dialect abstraction violation, boolean defaults, connection string escaping)
- **Low:** 1 bug (Seed prerequisite verification)

---

## Recommendations

### Immediate Actions (Critical)

1. **Fix Migration 002 for SQLite** - Remove redundant ALTER TABLE statements that fail on existing databases
2. **Fix PostgreSQL placeholder syntax** - Update `GetPlaceholder()` to return `$n` syntax instead of `?`
3. **Fix timestamp types in PostgreSQL** - Use `TIMESTAMP WITH TIME ZONE` consistently

### Short-term Actions (High Priority)

4. **Fix index creation race condition** - Use idempotent CREATE INDEX with error handling
5. **Remove schema duplication** - Clarify which migration owns which columns
6. **Audit foreign key behaviors** - Ensure CASCADE/RESTRICT settings match across databases
7. **Standardize unique constraints** - Use consistent syntax for all databases

### Medium-term Actions (Maintenance)

8. **Use dialect interface correctly** - Remove hardcoded database-specific logic from `markMigrationAsApplied()`
9. **Remove unused seed functions** - Delete `generateTestUsers`, `generateDogs`, `generateBookings` since they're not called
10. **Add connection string escaping** - URL-encode usernames and passwords in DSN builders
11. **Standardize boolean syntax** - Use consistent defaults across schema

### Long-term Actions (Quality)

12. **Add schema validation tests** - Automated tests to verify schema consistency across databases
13. **Add migration integration tests** - Test migrations on all three databases in CI/CD
14. **Document foreign key cascade behaviors** - Create a design document explaining each FK relationship
15. **Use a migration framework** - Consider using established tools like `golang-migrate` or `goose` instead of custom solution

### Testing Recommendations

1. **Test migration 002 on existing databases** - Verify it doesn't fail with column already exists errors
2. **Test PostgreSQL with `$n` placeholders** - Ensure query execution works correctly
3. **Test connection strings with special characters** - Verify escaping works for `@`, `:`, `&`, etc. in passwords
4. **Test concurrent deployments** - Ensure migrations are idempotent in race conditions
5. **Compare schemas** - Use `pg_dump`, `mysqldump`, and `.schema` to compare final schemas across databases

---

## Additional Notes

### Architecture Concerns

The custom migration system has several design issues:

1. **Error masking** - The `isAlreadyExistsError()` function (lines 170-200 in `migrations.go`) catches errors and marks migrations as applied even when they fail. This can hide real problems.

2. **No rollback support** - Migrations only have `Up`, no `Down`. This makes it impossible to undo migrations.

3. **No dry-run mode** - Cannot preview migrations before applying them.

4. **Limited validation** - No checks that migration SQL is valid before attempting to execute.

### Security Concerns

1. **Password escaping** - Bug #11 could lead to SQL injection if connection string values are attacker-controlled (unlikely, but possible in multi-tenant scenarios).

2. **Credentials file** - `SUPER_ADMIN_CREDENTIALS.txt` is world-readable (mode 0600 is correct, but documentation should emphasize security).

### Performance Concerns

1. **Duplicate indexes** - Bug #8 creates duplicate unique constraints, wasting disk space and slowing down writes.

2. **Wrong timestamp types** - Bug #5 in PostgreSQL means timezone conversions happen in application layer instead of database, reducing performance.

### Documentation Gaps

1. **No migration guide** - No documentation on how to create new migrations, test them, or roll them back.

2. **No schema diagram** - Would help visualize foreign key relationships and catch inconsistencies.

3. **Dialect interface docs** - Javadoc comments exist, but no usage examples or pitfalls documented.

The database migration system needs significant hardening before being production-ready for a SaaS platform with multiple tenants.
