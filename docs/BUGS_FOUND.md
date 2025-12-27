# Bugs Found During Test Coverage Improvement

This document catalogues bugs discovered while improving test coverage. Each bug includes location, severity, description, and recommended fix.

## Summary

| Severity | Count | Category |
|----------|-------|----------|
| CRITICAL | 5 | Security, Data Integrity |
| HIGH | 8 | Silent Errors, Performance |
| MEDIUM | 6 | Error Handling, Validation |
| LOW | 4 | Code Quality |

---

## CRITICAL BUGS

### 1. Cache Key Collision - Wrong Int-to-String Conversion
**File:** `internal/services/feature_flag_service.go:36`

**Issue:** `string(rune(65))` returns "A", not "65". Multiple tenants share cache keys.

**Impact:** Tenant 65 and 66 get keys "feature:A" and "feature:B" (ASCII characters instead of decimal strings)

**Fix:** Use `strconv.Itoa(tenantID)` or `fmt.Sprintf("%d", tenantID)`

---

### 2. Missing HTTP Timeout on External API
**File:** `internal/services/holiday_service.go:45`

**Issue:** `http.Get(url)` has no timeout.

**Impact:** If feiertage-api.de hangs, the entire application hangs indefinitely.

**Fix:** Use `http.Client{Timeout: 10*time.Second}`

---

### 3. TenantID=0 Bypass in Handlers
**File:** `internal/handlers/booking_handler.go:72`

**Issue:** Type assertion `tenantID, _ := r.Context().Value(...).(int)` silently defaults to 0.

**Impact:** If TenantMiddleware fails, requests proceed with tenantID=0.

**Fix:** Use `ok` check pattern and validate tenantID > 0

---

### 4. Missing Tenant Isolation in Delete
**File:** `internal/repository/blocked_date_repository.go:270`

**Issue:** `DELETE FROM blocked_dates WHERE id = ?` has no tenant_id filter.

**Impact:** If handler forgets to verify ownership, any tenant can delete another tenant's data.

**Fix:** Add tenantID parameter: `WHERE id = ? AND tenant_id = ?`

---

### 5. Silent Error in Stripe Metadata Parsing
**File:** `internal/services/stripe_service.go:206`

**Issue:** `fmt.Sscanf(tenantIDStr, "%d", &tenantID)` error is ignored.

**Impact:** Invalid metadata causes billing records with tenantID=0 (orphaned).

**Fix:** Check error from Sscanf

---

## HIGH SEVERITY BUGS

### 6. Nil Pointer Dereference in CancelBooking
**File:** `internal/handlers/booking_handler.go:466`

**Issue:** `booking.User.Email` accessed without checking if User is nil.

**Fix:** Add `booking.User != nil` check

---

### 7. Nil Pointer Dereference in ApprovePendingBooking
**File:** `internal/handlers/booking_handler.go:821`

**Issue:** `booking.Dog.Name` accessed without nil check.

**Fix:** Add `booking.Dog != nil` check

---

### 8. N+1 Query Problem in Walk Reports
**File:** `internal/repository/walk_report_repository.go:190-198`

**Issue:** For each report, calls `GetPhotos()` separately.

**Impact:** 100 reports = 101 queries instead of 1-2.

**Fix:** Use JOIN or batch query

---

### 9. Silent LastInsertId Errors
**Files:** Multiple repository files

**Issue:** `id, _ := result.LastInsertId()` ignores errors.

**Locations:**
- booking_time_repository.go:125
- marketing_repository.go:99, 214, 360
- holiday_repository.go:73

---

### 10. Silent Holiday Creation Errors
**File:** `internal/services/holiday_service.go:96, 143`

**Issue:** `_ = s.holidayRepo.CreateHoliday(...)` ignores errors.

---

### 11. Missing rows.Err() Check
**Files:** Multiple repository files

**Issue:** After `for rows.Next()` loop, `rows.Err()` is never checked.

**Impact:** Partial results if DB connection drops during iteration.

---

### 12. Dangerous Characters in Subdomain
**File:** `internal/middleware/tenant.go`

**Issue:** extractSubdomain allows SQL injection characters like `'`, `;`, null bytes.

**Fix:** Validate subdomain: `[a-z0-9-]+` only

---

### 13. AddNotes Missing TenantID Validation
**File:** `internal/handlers/booking_handler.go:492`

**Issue:** Unlike other methods, doesn't validate tenantID=0.

---

## MEDIUM SEVERITY BUGS

### 14. strconv.Atoi Errors Silently Ignored
**File:** `internal/handlers/booking_handler.go:266`

**Issue:** `dogID, _ := strconv.Atoi(dogIDStr)` ignores errors.

---

### 15. Silent Date Parse Error
**File:** `internal/services/holiday_service.go:107`

**Issue:** `dateObj, _ := time.Parse(...)` - year becomes 1 if parse fails.

---

### 16. Hardcoded Plan ID
**File:** `internal/repository/subscription_repository.go:303`

**Issue:** `plan_id = 1` assumes Free plan is always ID 1.

---

### 17. Ambiguous FindByIDAndTenant Return
**Files:** dog_repository.go, booking_repository.go, user_repository.go

**Issue:** Returns `(nil, nil)` for both "not found" and "wrong tenant".

---

### 18. Potential Nil Pointer in Stripe
**File:** `internal/services/stripe_service.go:198`

**Issue:** `session.Customer.ID` could be nil.

---

### 19. Error Ignored in FindByIDWithDetails
**File:** `internal/handlers/booking_handler.go:882`

**Issue:** `booking, _ := h.bookingRepo.FindByIDWithDetails(id)` ignores error.

---

## LOW SEVERITY BUGS

### 20. Error Message Parsing for Constraints
**File:** `internal/repository/blocked_date_repository.go:50-57`

**Issue:** Parses error strings to detect unique violations - fragile.

---

### 21. Timezone Assumption
**File:** `internal/services/holiday_service.go:107`

**Issue:** `time.Parse` without timezone consideration.

---

### 22. Settings Error Silent Fallback
**File:** `internal/services/holiday_service.go:31`

**Issue:** Silently falls back to default if settings read fails.

---

### 23. Cache Race Condition
**File:** `internal/services/cache_service.go:115-141`

**Issue:** TOCTOU between RLock and Lock.

---

## Test Files Created

The following test files document these bugs:

1. `internal/handlers/booking_handler_bugs_test.go`
2. `internal/middleware/tenant_test.go`
3. `internal/services/feature_flag_service_bugs_test.go`
4. `internal/services/holiday_service_bugs_test.go`
5. `internal/services/stripe_service_bugs_test.go`
6. `internal/repository/repository_bugs_test.go`

Run: `go test ./... -v 2>&1 | grep -i bug`
