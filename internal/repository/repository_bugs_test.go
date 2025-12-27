package repository

import (
	"testing"
)

// ============================================================================
// BUG #1: CRITICAL - Missing tenant isolation in Delete method
// File: blocked_date_repository.go, Line 270-279
// The Delete method does not filter by tenant_id!
// ============================================================================

func TestBlockedDateRepository_BUG_DeleteMissingTenantIsolation(t *testing.T) {
	// The Delete method at line 270 does:
	//   query := `DELETE FROM blocked_dates WHERE id = ?`
	//
	// BUG: No tenant_id filter!
	// An attacker could delete another tenant's blocked dates
	// if the handler fails to verify ownership.
	//
	// Current code relies on a comment:
	//   "SaaS SECURITY: Caller MUST verify tenant ownership before calling"
	// This is a security antipattern - the repository should enforce it!
	//
	// RECOMMENDATION:
	// Change to:
	//   func (r *BlockedDateRepository) Delete(id int, tenantID int) error {
	//       query := `DELETE FROM blocked_dates WHERE id = ? AND tenant_id = ?`
	//       result, err := r.db.Exec(query, id, tenantID)
	//       affected, _ := result.RowsAffected()
	//       if affected == 0 {
	//           return fmt.Errorf("blocked date not found or wrong tenant")
	//       }
	//   }

	t.Log("BUG: blocked_date_repository.go:270 Delete() lacks tenant_id filter")
	t.Log("Impact: Cross-tenant data deletion possible if handler forgets to check")
	t.Log("Severity: CRITICAL for multi-tenant SaaS")
}

// ============================================================================
// BUG #2: HIGH - Silent error handling on LastInsertId
// Multiple files ignore LastInsertId() errors
// ============================================================================

func TestRepository_BUG_SilentLastInsertIdError(t *testing.T) {
	// Multiple files have this pattern:
	//   result, err := r.db.Exec(query, ...)
	//   if err != nil { return err }
	//   id, _ := result.LastInsertId()  // ERROR IGNORED!
	//   obj.ID = int(id)
	//
	// Affected files and lines:
	// - booking_time_repository.go:125
	// - marketing_repository.go:99, 214, 360
	// - holiday_repository.go:73
	//
	// Impact: If LastInsertId() fails:
	// - Object is assigned ID=0
	// - Caller thinks creation succeeded
	// - Subsequent operations use wrong ID
	//
	// RECOMMENDATION:
	//   id, err := result.LastInsertId()
	//   if err != nil {
	//       return fmt.Errorf("failed to get inserted ID: %w", err)
	//   }

	t.Log("BUG: Multiple files use '_ =' with LastInsertId()")
	t.Log("Locations:")
	t.Log("  - booking_time_repository.go:125")
	t.Log("  - marketing_repository.go:99, 214, 360")
	t.Log("  - holiday_repository.go:73")
	t.Log("Impact: Objects may get ID=0 silently")
}

// ============================================================================
// BUG #3: HIGH - Missing rows.Err() check after iteration
// Multiple files don't check for database errors after loop
// ============================================================================

func TestRepository_BUG_MissingRowsErrCheck(t *testing.T) {
	// The pattern used is:
	//   rows, err := r.db.Query(...)
	//   if err != nil { return nil, err }
	//   defer rows.Close()
	//   for rows.Next() { /* scan rows */ }
	//   return results, nil  // BUG: Never calls rows.Err()!
	//
	// If database connection drops DURING iteration:
	// - Some rows are successfully scanned
	// - rows.Next() returns false (due to error)
	// - Error is stored in rows.Err()
	// - Without checking, partial results returned silently!
	//
	// Affected locations (partial list):
	// - user_repository.go:722-769 (FindAll)
	// - user_repository.go:630-677 (FindInactiveUsers)
	// - booking_repository.go:232-257 (FindAll)
	// - dog_repository.go:290-325 (FindAll)
	//
	// RECOMMENDATION:
	//   for rows.Next() { /* scan */ }
	//   if err := rows.Err(); err != nil {
	//       return nil, fmt.Errorf("error iterating rows: %w", err)
	//   }

	t.Log("BUG: Multiple Find* methods don't call rows.Err()")
	t.Log("Impact: Partial results returned if DB connection drops during iteration")
	t.Log("Affected methods include FindAll, FindInactiveUsers, GetFeatured, etc.")
}

// ============================================================================
// BUG #4: HIGH - N+1 Query Problem in Walk Report Repository
// File: walk_report_repository.go, Lines 190-198 and 253-261
// ============================================================================

func TestWalkReportRepository_BUG_NPlusOneQueries(t *testing.T) {
	// The FindByDogID and FindByUserID methods do:
	//   1. Execute query to get N walk reports
	//   2. For EACH report, call GetPhotos() separately
	//
	// This is the classic N+1 query problem:
	// - 1 query to get reports
	// - N queries to get photos (one per report)
	// = N+1 total queries
	//
	// For 100 reports, that's 101 database queries!
	//
	// RECOMMENDATION:
	// Option 1: Use a JOIN in the main query
	// Option 2: Use a single query to fetch all photos:
	//   photoQuery := `SELECT * FROM walk_report_photos WHERE report_id IN (?)`
	//   // Then match photos to reports in Go

	t.Log("BUG: walk_report_repository.go has N+1 query problem")
	t.Log("Lines: 190-198 (FindByDogID), 253-261 (FindByUserID)")
	t.Log("Impact: Performance degrades linearly with number of reports")
	t.Log("100 reports = 101 queries instead of 1-2 queries")
}

// ============================================================================
// BUG #5: MEDIUM - Hardcoded plan_id in subscription cancellation
// File: subscription_repository.go, Line 303
// ============================================================================

func TestSubscriptionRepository_BUG_HardcodedPlanID(t *testing.T) {
	// The CancelSubscription method does:
	//   query := `UPDATE tenant_subscriptions SET plan_id = 1 ...`
	//
	// BUG: Assumes plan_id=1 is always the Free plan!
	// If pricing plans table is modified (e.g., new plan added first),
	// cancelled subscriptions get wrong plan.
	//
	// RECOMMENDATION:
	// 1. Query for Free plan ID first:
	//    var freePlanID int
	//    r.db.QueryRow(`SELECT id FROM pricing_plans WHERE slug = 'free'`).Scan(&freePlanID)
	//
	// 2. Or use the slug in join:
	//    UPDATE tenant_subscriptions SET
	//      plan_id = (SELECT id FROM pricing_plans WHERE slug = 'free')

	t.Log("BUG: subscription_repository.go:303 hardcodes plan_id = 1")
	t.Log("Impact: If plan IDs change, cancellations assign wrong plan")
}

// ============================================================================
// BUG #6: MEDIUM - Ambiguous return value in FindByIDAndTenant
// Files: dog_repository.go, booking_repository.go, user_repository.go
// ============================================================================

func TestRepository_BUG_AmbiguousTenantMismatch(t *testing.T) {
	// FindByIDAndTenant methods return (nil, nil) for two cases:
	// 1. Record doesn't exist
	// 2. Record exists but belongs to different tenant
	//
	// Callers can't distinguish these cases!
	// This can mask security bugs or confuse debugging.
	//
	// Example from dog_repository.go:195-207:
	//   if dog == nil {
	//       return nil, nil  // Not found
	//   }
	//   if tenantID > 0 && dog.TenantID != tenantID {
	//       return nil, nil  // BUG: Same return value as "not found"!
	//   }
	//
	// RECOMMENDATION:
	// Use custom error types:
	//   var ErrNotFound = errors.New("not found")
	//   var ErrWrongTenant = errors.New("access denied")

	t.Log("BUG: FindByIDAndTenant returns (nil, nil) for both 'not found' and 'wrong tenant'")
	t.Log("Impact: Cannot distinguish legitimate missing records from tenant violations")
	t.Log("Affected: dog_repository, booking_repository, user_repository")
}

// ============================================================================
// BUG #7: LOW - Error message parsing for constraint violations
// File: blocked_date_repository.go, Lines 50-57
// ============================================================================

func TestBlockedDateRepository_BUG_ErrorMessageParsing(t *testing.T) {
	// The code parses error messages to detect unique constraint violations:
	//   errStr := strings.ToLower(err.Error())
	//   if strings.Contains(errStr, "unique") || strings.Contains(errStr, "duplicate") {
	//       return fmt.Errorf("friendly error message")
	//   }
	//
	// Issues:
	// 1. Database-version dependent (error messages change)
	// 2. Could match false positives (e.g., column named "unique")
	// 3. Not all DB drivers use same error format
	//
	// RECOMMENDATION:
	// Use database driver's error types to detect constraint violations:
	//   if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
	//       // Unique violation in PostgreSQL
	//   }

	t.Log("BUG: blocked_date_repository.go:50-57 parses error strings")
	t.Log("Impact: Fragile detection of constraint violations")
	t.Log("May break with different database versions")
}
