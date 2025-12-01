# Test Fix Plan

**Date:** 2025-01-24
**Status:** Analysis Complete
**Total Failing Tests:** 28

---

## Executive Summary

This document outlines the fixes required for all failing tests related to the new booking time restrictions feature. The failures fall into four main categories:

1. **Missing Admin Authorization Checks** (21 tests) - Handlers not validating admin permissions
2. **SQL Injection Protection Issues** (1 test) - Missing database verification
3. **Implementation Issues** (6 tests) - Incomplete feature implementation

**Root Cause:** Handlers rely on middleware for authorization but tests call handlers directly without middleware, exposing that handlers don't validate permissions from context.

---

## Category 1: Missing Admin Authorization Checks

### Issue Description

**Problem:** Admin-only handler methods don't check `isAdmin` from request context. They have comments like "Admin check done by middleware" but don't actually validate the permission when called directly.

**Affected Handlers:**
- `BookingTimeHandler`: GetRules, UpdateRules, CreateRule, DeleteRule
- `HolidayHandler`: CreateHoliday, UpdateHoliday, DeleteHoliday
- `BookingHandler`: GetPendingApprovals, ApprovePendingBooking, RejectPendingBooking

### Failing Tests

| Test File | Test Name | Handler | Expected | Current |
|-----------|-----------|---------|----------|---------|
| `booking_handler_test.go` | `TestGetPendingApprovals/TC-3.3.2-C` | GetPendingApprovals | 403 Forbidden | 200 OK |
| `booking_handler_test.go` | `TestApproveBooking/TC-3.3.3-D` | ApprovePendingBooking | 403 Forbidden | 500/200 |
| `booking_handler_test.go` | `TestRejectBooking/TC-3.3.4-D` | RejectPendingBooking | 403 Forbidden | 500/200 |
| `booking_time_handler_security_test.go` | `TestSecurityAdminEndpointProtection/TC-6.1.1-A` | GetRules | 403 Forbidden | 200 OK |
| `booking_time_handler_security_test.go` | `TestSecurityAdminEndpointProtection/TC-6.1.1-B` | UpdateRules | 403 Forbidden | 200/400 |
| `booking_time_handler_security_test.go` | `TestSecurityAdminEndpointProtection/TC-6.1.1-C` | CreateRule | 403 Forbidden | 201/400 |
| `booking_time_handler_security_test.go` | `TestSecurityAdminEndpointProtection/TC-6.1.1-D` | DeleteRule | 403 Forbidden | 200 |
| `booking_time_handler_security_test.go` | `TestSecurityAdminEndpointProtection/TC-6.1.1-E` | CreateHoliday | 403 Forbidden | 201/400 |
| `booking_time_handler_security_test.go` | `TestSecurityAdminEndpointProtection/TC-6.1.1-F` | UpdateHoliday | 403 Forbidden | 200 |
| `booking_time_handler_security_test.go` | `TestSecurityAdminEndpointProtection/TC-6.1.1-G` | DeleteHoliday | 403 Forbidden | 200 |
| `booking_time_handler_security_test.go` | `TestSecurityAdminEndpointProtection/TC-6.1.1-H` | GetRules (admin) | 200 OK | 200 OK ✓ |

### Fix Required

Add admin permission check at the beginning of each admin-only handler method.

**Standard Pattern:**

```go
// Check if user is admin
isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
if !ok || !isAdmin {
    respondError(w, http.StatusForbidden, "Admin access required")
    return
}
```

### Files to Modify

#### 1. `internal/handlers/booking_time_handler.go`

**Methods to fix:**
- `GetRules()` - Line 52
- `UpdateRules()` - Line 82
- `CreateRule()` - Line 114
- `DeleteRule()` - Line 136

**Fix for each method:**

```go
// Add at beginning of GetRules (after line 52)
func (h *BookingTimeHandler) GetRules(w http.ResponseWriter, r *http.Request) {
    // Check admin permission
    isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
    if !ok || !isAdmin {
        respondError(w, http.StatusForbidden, "Admin access required")
        return
    }

    // ... rest of existing code
}
```

Repeat for UpdateRules, CreateRule, and DeleteRule.

#### 2. `internal/handlers/holiday_handler.go`

**Methods to fix:**
- `CreateHoliday()` - Line 55
- `UpdateHoliday()` - Line 82
- `DeleteHoliday()` - Line 115

**Fix:**

```go
// Add at beginning of each method
isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
if !ok || !isAdmin {
    respondError(w, http.StatusForbidden, "Admin access required")
    return
}
```

#### 3. `internal/handlers/booking_handler.go`

**Methods to fix:**
- `GetPendingApprovals()` - Line 681
- `ApprovePendingBooking()` - Line 693
- `RejectPendingBooking()` - Line 725

**Fix:**

```go
// Add at beginning of each method (before line 682, 694, 726)
isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
if !ok || !isAdmin {
    respondError(w, http.StatusForbidden, "Admin access required")
    return
}
```

---

## Category 2: SQL Injection Protection

### Issue Description

**Problem:** SQL injection test expects database verification that custom_holidays table still exists after injection attempt, but test may be failing on the database check itself.

### Failing Test

| Test File | Test Name | Issue |
|-----------|-----------|-------|
| `booking_time_handler_security_test.go` | `TestSecuritySQLInjection/TC-6.2.1-B` | Holiday name SQL injection verification |

### Analysis

The test at line 292-308 in `booking_time_handler_security_test.go`:

```go
{
    name:    "TC-6.2.1-B: SQL injection in holiday name",
    method:  http.MethodPost,
    path:    "/api/holidays",
    handler: holidayHandler.CreateHoliday,
    body:    `{"date":"2025-07-01","name":"Holiday'; DELETE FROM custom_holidays; --","is_active":true,"source":"admin"}`,
    checkDatabase: func(t *testing.T, db *sql.DB) {
        // Verify custom_holidays table still exists (not dropped)
        var tableName string
        err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='custom_holidays'").Scan(&tableName)
        if err != nil {
            t.Error("custom_holidays table was deleted! SQL injection succeeded!")
        }
        // The table should exist even if empty
        if tableName != "custom_holidays" {
            t.Error("custom_holidays table structure compromised!")
        }
    },
    description: "SQL injection in holiday name should be escaped",
},
```

### Possible Issues

1. **Admin check required** - CreateHoliday needs admin check (covered in Category 1)
2. **Table name check** - If query returns no rows, `tableName` will be empty string but error is checked

### Fix Required

The SQL injection is already protected by Go's parameterized queries in the repository. The fix needed is:

1. Add admin check to CreateHoliday (Category 1 fix)
2. Improve database verification in test:

```go
checkDatabase: func(t *testing.T, db *sql.DB) {
    // Verify custom_holidays table still exists
    var tableName string
    err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='custom_holidays'").Scan(&tableName)
    if err != nil {
        t.Errorf("Failed to check table existence: %v (SQL injection may have succeeded!)", err)
        return
    }
    if tableName != "custom_holidays" {
        t.Errorf("Expected table name 'custom_holidays', got '%s'", tableName)
    }
    // Also verify we can query the table
    var count int
    err = db.QueryRow("SELECT COUNT(*) FROM custom_holidays").Scan(&count)
    if err != nil {
        t.Errorf("Cannot query custom_holidays table: %v", err)
    }
},
```

**However**, since repositories use parameterized queries (`?` placeholders), SQL injection is already prevented. The test should pass once admin check is added.

---

## Category 3: Implementation Issues

### Issue Description

**Problem:** Some functionality may not be fully implemented or there are integration issues with the new booking time service.

### Failing Tests

| Test File | Test Name | Likely Issue |
|-----------|-----------|--------------|
| `booking_handler_test.go` | `TestRejectBooking/TC-3.3.4-A` | Rejection logic or database update |
| `booking_time_handler_test.go` | `TestGetAvailableSlots/TC-3.1.1-A` | Service integration |
| `booking_time_handler_test.go` | `TestGetAvailableSlots/TC-3.1.1-B` | Service integration |
| `booking_time_handler_test.go` | `TestGetRules/TC-3.1.2-A` | Admin check needed |
| `booking_time_handler_test.go` | `TestUpdateRules` | Admin check needed |
| `booking_time_handler_test.go` | `TestCreateRule/TC-3.1.4-A` | Admin check needed |
| `booking_time_handler_test.go` | `TestDeleteRule` | Admin check needed |
| `booking_time_handler_test.go` | `TestGetRulesForDate/Valid_weekday` | Service integration |
| `booking_time_handler_test.go` | `TestGetRulesForDate/Valid_weekend` | Service integration |
| `booking_time_handler_test.go` | `TestGetHolidays` | Service integration |
| `booking_time_handler_test.go` | `TestCreateHoliday/TC-3.2.2-A` | Admin check needed |
| `booking_time_handler_test.go` | `TestUpdateHoliday` | Admin check needed |
| `booking_time_handler_test.go` | `TestDeleteHoliday` | Admin check needed |

### Analysis by Test

#### TestRejectBooking/TC-3.3.4-A (Line 1311-1327)

**Expected:** Admin can reject booking with reason, status becomes "cancelled"

**Potential Issues:**
1. Admin check missing (Category 1 fix will help)
2. Repository method may not be working correctly
3. Booking status not being updated

**Investigation needed:**
- Check `internal/repository/booking_repository.go` - `RejectBooking()` method
- Verify the SQL query updates both `status` and `rejection_reason`

#### TestGetAvailableSlots Tests (Line 44-132)

**Expected:** Returns available time slots based on day type rules

**Potential Issues:**
1. Service not initialized properly in test
2. Default rules not seeded in test database
3. Service logic has bugs

**Investigation needed:**
- Check if test database has default time rules seeded
- Verify `BookingTimeService.GetAvailableTimeSlots()` implementation
- Check `BookingTimeRepository.GetRulesByDayType()` returns rules

#### TestGetRules/TestUpdateRules/TestCreateRule/TestDeleteRule

**Expected:** CRUD operations on booking time rules

**Primary Issue:** Admin checks missing (Category 1 fixes these)

**Secondary Check:**
- Verify repository methods work correctly
- Check test database migrations ran

#### TestGetRulesForDate Tests (Line 410-491)

**Expected:** Returns rules applicable to specific date (weekday vs weekend detection)

**Potential Issues:**
1. `BookingTimeService.GetRulesForDate()` not determining day type correctly
2. Holiday service integration issues
3. Rules not being filtered properly

**Investigation needed:**
- Check `getDayType()` method in booking_time_service.go
- Verify weekend detection (Saturday/Sunday)
- Verify holiday detection works

#### TestGetHolidays (Line 46-157)

**Expected:** Returns holidays for specified year

**Potential Issues:**
1. HolidayService not initialized properly in test
2. Year filtering not working
3. Database query issues

**Investigation needed:**
- Check `HolidayRepository.GetHolidaysByYear()` SQL query
- Verify year parameter parsing in handler

#### TestCreateHoliday/TestUpdateHoliday/TestDeleteHoliday

**Primary Issue:** Admin checks missing (Category 1 fixes these)

**Secondary Check:**
- Verify repository methods
- Check model validation

---

## Fix Implementation Order

### Phase 1: Add Admin Checks (Priority: CRITICAL)
**Time Estimate:** 30 minutes

1. Add admin checks to `BookingTimeHandler` (4 methods)
2. Add admin checks to `HolidayHandler` (3 methods)
3. Add admin checks to `BookingHandler` (3 methods)

**Expected Result:** 18 tests should pass after this phase.

### Phase 2: Test Database Seeding (Priority: HIGH)
**Time Estimate:** 1 hour

1. Verify test setup functions seed default booking time rules
2. Verify migrations run in test databases
3. Add explicit seeding if needed in test setup

**Files to check:**
- `internal/handlers/booking_time_handler_test.go` - `setupBookingTimeHandlerTest()`
- `internal/handlers/holiday_handler_test.go` - `setupHolidayHandlerTest()`

**Expected Result:** Available slots and rules tests should pass.

### Phase 3: Service Integration Debugging (Priority: HIGH)
**Time Estimate:** 2 hours

1. Debug `GetAvailableTimeSlots` failures
   - Add logging to see what rules are returned
   - Check granularity calculation
   - Verify time slot generation logic

2. Debug `GetRulesForDate` failures
   - Check day type determination
   - Verify holiday service integration
   - Test weekend detection

3. Debug holiday tests
   - Check year filtering
   - Verify API integration (or mock in tests)

**Expected Result:** Booking time feature tests pass.

### Phase 4: Repository Method Verification (Priority: MEDIUM)
**Time Estimate:** 1 hour

1. Check `RejectBooking()` repository method
   - Verify SQL updates both status and rejection_reason
   - Check return values
   - Add unit test if missing

2. Verify all repository methods handle errors correctly

**Expected Result:** Rejection test passes.

### Phase 5: Test Improvements (Priority: LOW)
**Time Estimate:** 30 minutes

1. Improve SQL injection test database verification
2. Add better error messages in tests
3. Add integration tests if needed

---

## Detailed Fix Steps

### Step 1: Fix Admin Checks

**File:** `internal/handlers/booking_time_handler.go`

```go
// Add import at top if not present
import (
    "github.com/tranm/gassigeher/internal/middleware"
)

// Fix GetRules (line 52)
func (h *BookingTimeHandler) GetRules(w http.ResponseWriter, r *http.Request) {
    // Check admin permission
    isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
    if !ok || !isAdmin {
        respondError(w, http.StatusForbidden, "Admin access required")
        return
    }

    rules, err := h.bookingTimeRepo.GetAllRules()
    // ... rest unchanged
}

// Fix UpdateRules (line 82)
func (h *BookingTimeHandler) UpdateRules(w http.ResponseWriter, r *http.Request) {
    // Check admin permission
    isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
    if !ok || !isAdmin {
        respondError(w, http.StatusForbidden, "Admin access required")
        return
    }

    var rules []models.BookingTimeRule
    // ... rest unchanged
}

// Fix CreateRule (line 114)
func (h *BookingTimeHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
    // Check admin permission
    isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
    if !ok || !isAdmin {
        respondError(w, http.StatusForbidden, "Admin access required")
        return
    }

    var rule models.BookingTimeRule
    // ... rest unchanged
}

// Fix DeleteRule (line 136)
func (h *BookingTimeHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
    // Check admin permission
    isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
    if !ok || !isAdmin {
        respondError(w, http.StatusForbidden, "Admin access required")
        return
    }

    // Extract ID from path
    // ... rest unchanged
}
```

**File:** `internal/handlers/holiday_handler.go`

```go
// Fix CreateHoliday (line 55)
func (h *HolidayHandler) CreateHoliday(w http.ResponseWriter, r *http.Request) {
    // Check admin permission
    isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
    if !ok || !isAdmin {
        respondError(w, http.StatusForbidden, "Admin access required")
        return
    }

    adminID, _ := r.Context().Value(middleware.UserIDKey).(int)
    // ... rest unchanged
}

// Fix UpdateHoliday (line 82)
func (h *HolidayHandler) UpdateHoliday(w http.ResponseWriter, r *http.Request) {
    // Check admin permission
    isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
    if !ok || !isAdmin {
        respondError(w, http.StatusForbidden, "Admin access required")
        return
    }

    // Extract ID from path
    // ... rest unchanged
}

// Fix DeleteHoliday (line 115)
func (h *HolidayHandler) DeleteHoliday(w http.ResponseWriter, r *http.Request) {
    // Check admin permission
    isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
    if !ok || !isAdmin {
        respondError(w, http.StatusForbidden, "Admin access required")
        return
    }

    // Extract ID from path
    // ... rest unchanged
}
```

**File:** `internal/handlers/booking_handler.go`

```go
// Fix GetPendingApprovals (line 681)
func (h *BookingHandler) GetPendingApprovals(w http.ResponseWriter, r *http.Request) {
    // Check admin permission
    isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
    if !ok || !isAdmin {
        respondError(w, http.StatusForbidden, "Admin access required")
        return
    }

    bookings, err := h.bookingRepo.GetPendingApprovalBookings()
    // ... rest unchanged
}

// Fix ApprovePendingBooking (line 693)
func (h *BookingHandler) ApprovePendingBooking(w http.ResponseWriter, r *http.Request) {
    // Check admin permission
    isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
    if !ok || !isAdmin {
        respondError(w, http.StatusForbidden, "Admin access required")
        return
    }

    adminID, _ := r.Context().Value(middleware.UserIDKey).(int)
    // ... rest unchanged
}

// Fix RejectPendingBooking (line 725)
func (h *BookingHandler) RejectPendingBooking(w http.ResponseWriter, r *http.Request) {
    // Check admin permission
    isAdmin, ok := r.Context().Value(middleware.IsAdminKey).(bool)
    if !ok || !isAdmin {
        respondError(w, http.StatusForbidden, "Admin access required")
        return
    }

    adminID, _ := r.Context().Value(middleware.UserIDKey).(int)
    // ... rest unchanged
}
```

### Step 2: Run Tests After Admin Fixes

```bash
go test ./internal/handlers -v -run "TestGetPendingApprovals|TestApproveBooking|TestRejectBooking|TestSecurityAdminEndpointProtection"
```

**Expected:** 18 tests pass (all admin authorization tests).

### Step 3: Debug Service Integration Issues

If remaining tests still fail, add debug logging:

**File:** `internal/services/booking_time_service.go`

```go
// Add logging to GetAvailableTimeSlots
func (s *BookingTimeService) GetAvailableTimeSlots(date string) ([]string, error) {
    // ... existing code ...

    rules, err := s.bookingTimeRepo.GetRulesByDayType(dayType)
    if err != nil {
        return nil, err
    }

    // DEBUG: Log rules found
    fmt.Printf("DEBUG: GetAvailableTimeSlots for %s (dayType=%s): found %d rules\n",
        date, dayType, len(rules))
    for _, rule := range rules {
        fmt.Printf("  - %s: %s-%s (blocked=%v)\n",
            rule.RuleName, rule.StartTime, rule.EndTime, rule.IsBlocked)
    }

    // ... rest of method ...
}
```

Run tests with output:

```bash
go test ./internal/handlers -v -run "TestGetAvailableSlots" 2>&1 | tee test_output.txt
```

Review `test_output.txt` to see what rules are being loaded.

### Step 4: Verify Test Database Setup

Check if default rules are seeded in test:

**File:** `internal/handlers/booking_time_handler_test.go`

```go
func setupBookingTimeHandlerTest(t *testing.T) (*sql.DB, *BookingTimeHandler, func()) {
    // Create in-memory test database
    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        t.Fatalf("Failed to open test database: %v", err)
    }

    // Run migrations
    if err := database.RunMigrations(db); err != nil {
        t.Fatalf("Failed to run migrations: %v", err)
    }

    // VERIFY: Check if default rules were seeded
    var count int
    err = db.QueryRow("SELECT COUNT(*) FROM booking_time_rules").Scan(&count)
    if err != nil {
        t.Fatalf("Failed to query booking_time_rules: %v", err)
    }
    if count == 0 {
        t.Fatalf("No default booking time rules seeded! Migration may have failed.")
    }
    t.Logf("Test database has %d booking time rules", count)

    // ... rest of setup
}
```

### Step 5: Fix Specific Repository Issues

If `RejectBooking` test still fails, check repository:

**File:** `internal/repository/booking_repository.go`

Look for `RejectBooking` method and verify SQL:

```go
func (r *BookingRepository) RejectBooking(bookingID int, adminID int, reason string) error {
    query := `
        UPDATE bookings
        SET approval_status = 'rejected',
            approved_by = ?,
            approved_at = ?,
            rejection_reason = ?,
            status = 'cancelled'
        WHERE id = ? AND approval_status = 'pending'
    `

    result, err := r.db.Exec(query, adminID, time.Now(), reason, bookingID)
    if err != nil {
        return err
    }

    rows, _ := result.RowsAffected()
    if rows == 0 {
        return fmt.Errorf("booking not found or not pending")
    }

    return nil
}
```

Ensure `status = 'cancelled'` is included.

---

## Testing Checklist

After implementing fixes, run tests in this order:

### Phase 1: Admin Authorization Tests
```bash
# Test admin checks
go test ./internal/handlers -v -run "TestSecurityAdminEndpointProtection"
# Expected: All 8 sub-tests pass

go test ./internal/handlers -v -run "TestGetPendingApprovals"
# Expected: TC-3.3.2-A and TC-3.3.2-C both pass

go test ./internal/handlers -v -run "TestApproveBooking"
# Expected: TC-3.3.3-A and TC-3.3.3-D both pass

go test ./internal/handlers -v -run "TestRejectBooking"
# Expected: All 3 sub-tests pass
```

### Phase 2: Booking Time Functionality Tests
```bash
# Test time slots
go test ./internal/handlers -v -run "TestGetAvailableSlots"
# Expected: TC-3.1.1-A and TC-3.1.1-B pass

# Test rules CRUD
go test ./internal/handlers -v -run "TestGetRules"
go test ./internal/handlers -v -run "TestUpdateRules"
go test ./internal/handlers -v -run "TestCreateRule"
go test ./internal/handlers -v -run "TestDeleteRule"

# Test rules for date
go test ./internal/handlers -v -run "TestGetRulesForDate"
```

### Phase 3: Holiday Tests
```bash
go test ./internal/handlers -v -run "TestGetHolidays"
go test ./internal/handlers -v -run "TestCreateHoliday"
go test ./internal/handlers -v -run "TestUpdateHoliday"
go test ./internal/handlers -v -run "TestDeleteHoliday"
```

### Phase 4: Security Tests
```bash
go test ./internal/handlers -v -run "TestSecuritySQLInjection"
```

### Phase 5: Full Test Suite
```bash
# Run all tests
go test ./... -v

# Check for remaining failures
go test ./... | grep FAIL
```

---

## Success Criteria

All tests pass:
- [ ] 0 FAIL messages in test output
- [ ] All authorization tests return 403 for non-admins
- [ ] All authorization tests return 200/201 for admins
- [ ] Available slots tests return time slots
- [ ] Rules CRUD tests work correctly
- [ ] Holiday tests work correctly
- [ ] SQL injection tests verify database integrity

---

## Rollback Plan

If fixes cause regressions:

1. **Revert admin checks** if they break production:
   - Comment out admin checks temporarily
   - Add `TODO` comments
   - Deploy with middleware-only protection
   - Schedule proper fix

2. **Disable feature** if critical bugs found:
   - Set `use_booking_time_restrictions=false` in settings
   - Fall back to old booking logic
   - Fix in development environment

---

## Post-Fix Actions

After all tests pass:

1. **Run full test suite:** `go test ./... -v -coverprofile=coverage.out`
2. **Check coverage:** `go tool cover -html=coverage.out`
3. **Manual testing:** Follow test plan in BookingTimeTestPlan.md
4. **Update documentation:** Mark feature as fully tested
5. **Deploy to staging:** Test in staging environment
6. **Monitor production:** Watch for errors after deployment

---

## Summary

**Primary Issue:** Missing admin authorization checks in handlers
**Primary Fix:** Add 4-line admin check to 10 handler methods
**Estimated Fix Time:** 30 minutes for admin checks + 2-3 hours for integration debugging
**Risk Level:** Low (fixes are straightforward, feature is new)
**Test Coverage:** Will improve from current state to 100% for new feature

**Key Insight:** Handlers should not rely solely on middleware for authorization. Defensive programming requires handlers to validate permissions from context, making them safe to call in any environment (production, testing, or direct invocation).
