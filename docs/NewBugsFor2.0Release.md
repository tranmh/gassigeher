# New Bugs for 2.0 Release

This document tracks critical and high priority bugs discovered during the security audit for the 2.0 release.

**Audit Date:** 2025-12-30
**Total Bugs Found:** 18 (7 CRITICAL + 11 HIGH)

---

## CRITICAL BUGS (7)

### CRITICAL-1: Database Query Errors Silently Ignored
**File:** `internal/handlers/central_admin_handler.go:214-216`
**Severity:** CRITICAL
**Type:** Database Error Handling

**Issue:** Three QueryRow calls ignore errors completely:
```go
h.db.QueryRow(`SELECT COUNT(*) FROM users WHERE tenant_id = ? AND is_deleted = 0`, tenantID).Scan(&userCount)
h.db.QueryRow(`SELECT COUNT(*) FROM dogs WHERE tenant_id = ?`, tenantID).Scan(&dogCount)
h.db.QueryRow(`SELECT COUNT(*) FROM bookings WHERE tenant_id = ?`, tenantID).Scan(&bookingCount)
```

**Impact:** If queries fail (DB connection down, permission issues), count variables remain 0 and handler returns incorrect data. No error logged or reported.

**Fix Required:** Check error from Scan() and handle appropriately.

---

### CRITICAL-2: Missing rows.Err() in ListCentralAdmins
**File:** `internal/handlers/central_admin_handler.go:370-377`
**Severity:** CRITICAL
**Type:** Database Row Iteration

**Issue:** After iterating rows with `for rows.Next()`, `rows.Err()` is never checked.

**Impact:** If database connection drops mid-iteration, incomplete admin list returned without error notification.

**Fix Required:** Add `if err := rows.Err(); err != nil { ... }` after loop.

---

### CRITICAL-3: Missing rows.Err() in SearchUsers
**File:** `internal/handlers/central_admin_handler.go:585-593`
**Severity:** CRITICAL
**Type:** Database Row Iteration

**Issue:** Same pattern as CRITICAL-2 - missing `rows.Err()` check after iteration.

**Impact:** Incomplete user search results without error notification.

**Fix Required:** Add `if err := rows.Err(); err != nil { ... }` after loop.

---

### CRITICAL-4: Goroutine Leak - TenantRateLimiter Not Closed
**File:** `cmd/server/main.go:223` + `internal/middleware/ratelimit_tenant.go:91-92`
**Severity:** CRITICAL
**Type:** Resource Leak / Goroutine Management

**Issue:** The `TenantRateLimiter` starts a cleanup goroutine that runs indefinitely:
```go
go trl.cleanupStaleEntries()
```
This goroutine is NEVER stopped because there's no call to `Close()` on server shutdown.

**Current State:**
- `cronService.Stop()` is called ✓
- `authHandler.Close()` is called ✓
- `tenantRateLimiterInstance.Close()` is NOT called ✗

**Impact:** Goroutine leak on every restart, memory leak for stale limiter entries.

**Fix Required:** Add `defer` call to close TenantRateLimiter on shutdown.

---

### CRITICAL-5: time.Parse Errors Ignored in Marketing Handler
**File:** `internal/handlers/marketing_handler.go:163,167`
**Severity:** CRITICAL
**Type:** Silent Error / Data Corruption

**Issue:** time.Parse errors silently ignored:
```go
t, _ := time.Parse("2006-01-02", *req.StartDate)  // ERROR IGNORED!
t, _ := time.Parse("2006-01-02", *req.EndDate)    // ERROR IGNORED!
```

**Impact:** Invalid dates like "2025-99-99" accepted, zero time.Time stored in database, campaign validity logic breaks.

**Fix Required:** Check parse error and return validation error to client.

---

### CRITICAL-6: SQLite-Only SQL Syntax in Provisioning
**File:** `internal/services/provisioning_service.go:136`
**Severity:** CRITICAL
**Type:** Database Compatibility

**Issue:** Uses SQLite-specific `INSERT OR REPLACE` syntax:
```go
_, err := tx.Exec(
    "INSERT OR REPLACE INTO system_settings (tenant_id, `key`, value) VALUES (?, ?, ?)",
    tenantID, key, value,
)
```

**Impact:**
- MySQL: Fails with syntax error (uses `REPLACE INTO`)
- PostgreSQL: Fails completely (uses `ON CONFLICT DO UPDATE`)
- New SaaS tenants cannot be created on production PostgreSQL

**Fix Required:** Use database-agnostic upsert pattern or dialect-specific SQL.

---

### CRITICAL-7: time.Parse Errors in PromoCode Repository
**File:** `internal/repository/promo_code_repository.go:119,178,244`
**Severity:** CRITICAL
**Type:** Silent Error / Data Corruption

**Issue:** time.Parse errors silently ignored in three locations:
```go
t, _ := time.Parse(time.RFC3339, expiresAt.String)
code.ExpiresAt = &t
```

**Impact:** Malformed timestamps parse as zero values, promo codes treated as expired/active incorrectly, billing errors.

**Fix Required:** Handle parse errors properly.

---

## HIGH PRIORITY BUGS (11)

### HIGH-1: File Seek Error Ignored After MIME Validation
**Files:**
- `internal/handlers/dog_handler.go:627`
- `internal/handlers/user_handler.go:287`
- `internal/handlers/settings_handler.go:200`

**Severity:** HIGH
**Type:** File Upload / Error Handling

**Issue:** `file.Seek(0, 0)` error ignored:
```go
file.Seek(0, 0)  // error ignored
```

**Impact:** If seek fails, handler reads from wrong position causing empty file reads or corrupted image processing.

**Fix Required:** Check seek error: `if _, err := file.Seek(0, 0); err != nil { ... }`

---

### HIGH-2: Missing rows.Err() in ExportTenantData
**File:** `internal/handlers/tenant_handler.go:938-978, 994-1029, 1057-1077`
**Severity:** HIGH
**Type:** Database Row Iteration

**Issue:** Three Query() operations in GDPR export don't check `rows.Err()` after iteration.

**Impact:** GDPR exports incomplete without notification - compliance violation.

**Fix Required:** Add `rows.Err()` checks after each loop.

---

### HIGH-3: Missing Email Validation for ReferrerEmail
**File:** `internal/handlers/marketing_handler.go:305,396`
**Severity:** HIGH
**Type:** Input Validation

**Issue:** ReferrerEmail never validated in CreateReferralCode and UpdateReferralCode:
```go
code.ReferrerEmail = req.ReferrerEmail  // NEVER VALIDATED!
```

**Impact:** Email header injection possible, could be used for phishing campaigns.

**Fix Required:** Validate email format using `models.ValidateEmail()`.

---

### HIGH-4: Missing Name Validation on Campaign Update
**File:** `internal/handlers/marketing_handler.go:150-151`
**Severity:** HIGH
**Type:** Input Validation

**Issue:** Campaign name not re-validated on update (only validated in CreateCampaign):
```go
if req.Name != nil {
    campaign.Name = *req.Name  // NEVER VALIDATED!
}
```

**Impact:** Admin could inject 10MB+ string causing memory exhaustion, potential XSS.

**Fix Required:** Add length validation (max 255 chars).

---

### HIGH-5: Unbounded Description in Campaign Update
**File:** `internal/handlers/marketing_handler.go:153-155`
**Severity:** HIGH
**Type:** Resource Exhaustion

**Issue:** Description has no length validation:
```go
if req.Description != nil {
    campaign.Description = req.Description  // NO SIZE LIMIT!
}
```

**Impact:** Multi-gigabyte description could cause database bloat and DoS.

**Fix Required:** Add max length validation.

---

### HIGH-6: Missing Validate() in FeatureFlag Requests
**File:** `internal/models/feature_flag.go:47-62`
**Severity:** HIGH
**Type:** Missing Validation

**Issue:** `CreateFeatureFlagRequest` and `UpdateFeatureFlagRequest` have NO `Validate()` methods.

**Impact:** Inconsistent validation pattern, potential for invalid data.

**Fix Required:** Add Validate() methods with proper field validation.

---

### HIGH-7: Incomplete Settings Validation
**File:** `internal/models/settings.go:14-25`
**Severity:** HIGH
**Type:** Incomplete Validation

**Issue:** `UpdateSettingRequest.Validate()` only checks if value is non-empty, no range validation:
```go
func (r *UpdateSettingRequest) Validate() error {
    if r.Value == "" {
        return &ValidationError{Field: "value", Message: "Value is required"}
    }
    return nil
}
```

**Impact:** Invalid settings (negative booking_advance_days, etc.) can break application logic.

**Fix Required:** Add type-specific and range validation.

---

### HIGH-8: Unsafe int64→int Conversion in PromoCode
**File:** `internal/repository/promo_code_repository.go:115,174,240`
**Severity:** HIGH
**Type:** Unsafe Type Conversion

**Issue:** int64 cast to int without bounds checking:
```go
maxUsesInt := int(maxUses.Int64)
```

**Impact:** Integer overflow on 32-bit systems, potential negative MaxUses values.

**Fix Required:** Add bounds checking before conversion.

---

### HIGH-9: Missing PromoCode Constraint Validation
**File:** `internal/models/promo_code.go:99-126`
**Severity:** HIGH
**Type:** Incomplete Validation

**Issue:** PromoCode.Validate() doesn't check:
- MaxUses for negative values
- ExpiresAt for past dates
- ValidForPlans for valid JSON

**Impact:** Invalid data persisted to database.

**Fix Required:** Add constraint validation in Validate() method.

---

### HIGH-10: Cross-Tenant Export Authorization Risk
**File:** `internal/handlers/tenant_handler.go:868-888`
**Severity:** HIGH
**Type:** Authorization

**Issue:** ExportTenantData verifies admin but NOT that user is admin of THIS tenant:
```go
isAdmin, _ := r.Context().Value(middleware.IsAdminKey).(bool)
if !isAdmin {
    respondError(w, http.StatusForbidden, "...")
    return
}
// MISSING: Check admin belongs to THIS tenant
```

**Impact:** In SaaS-Mode, admin of Tenant A could potentially export Tenant B's data.

**Fix Required:** Verify admin's tenant_id matches requested tenant.

---

### HIGH-11: QueryRow Error Ignored in Export Stats
**File:** `internal/handlers/central_admin_handler.go:657`
**Severity:** HIGH
**Type:** Database Error Handling

**Issue:** QueryRow.Scan() error ignored in ExportTenantData:
```go
h.db.QueryRow(`SELECT COUNT(*) FROM bookings WHERE tenant_id = ?`, tenantID).Scan(&bookingCount)
```

**Impact:** GDPR export includes incorrect booking counts.

**Fix Required:** Check Scan() error.

---

## Fix Status

| Bug ID | Status | Fixed In Commit |
|--------|--------|-----------------|
| CRITICAL-1 | **DONE** | Added error logging for QueryRow calls |
| CRITICAL-2 | **DONE** | Added rows.Err() check after iteration |
| CRITICAL-3 | **DONE** | Added rows.Err() check after iteration |
| CRITICAL-4 | **DONE** | Added CloseTenantRateLimiter() + defer in main.go |
| CRITICAL-5 | **DONE** | Added time.Parse error validation with proper response |
| CRITICAL-6 | **DONE** | Changed to UPDATE-then-INSERT pattern |
| CRITICAL-7 | **DONE** | Added time.Parse error logging in 3 locations |
| HIGH-1 | **DONE** | Added file.Seek error handling in 4 handlers |
| HIGH-2 | **DONE** | Added rows.Err() checks in 3 loops (dogs, bookings, blocked_dates) |
| HIGH-3 | **DONE** | Added ValidateEmail() for ReferrerEmail in Create/Update |
| HIGH-4 | **DONE** | Added name length validation (max 255 chars) |
| HIGH-5 | **DONE** | Added description length validation (max 10000 chars) |
| HIGH-6 | **DONE** | Added Validate() methods and validation calls in handler |
| HIGH-7 | Pending | - |
| HIGH-8 | Pending | - |
| HIGH-9 | Pending | - |
| HIGH-10 | Pending | - |
| HIGH-11 | Pending | - |

---

## Testing Checklist

- [ ] All CRITICAL bugs have failing tests (RED phase)
- [ ] All CRITICAL bugs fixed (GREEN phase)
- [ ] All HIGH bugs have failing tests (RED phase)
- [ ] All HIGH bugs fixed (GREEN phase)
- [ ] Full test suite passes
- [ ] Build succeeds on all platforms
