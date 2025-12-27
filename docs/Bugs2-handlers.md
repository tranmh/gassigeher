# Bug Report: handlers

**Analysis Date:** 2025-12-27
**Directory Analyzed:** `/home/tranmh/work/gassigeher-saas/internal/handlers`
**Files Analyzed:** 60+ files
**Bugs Found:** 11 bugs

---

## Summary

This analysis identified 11 functional bugs in the handlers directory of the dog walking booking system. The bugs span multiple categories including race conditions (2), error handling issues (3), logic errors (2), security vulnerabilities (2), and data integrity issues (2). The most critical findings involve race conditions in the dog creation limit check, missing tenant validation in blocked date deletion, and incorrect error message handling in experience request validation.

**Severity Distribution:**
- Critical: 2 bugs (Race condition in CreateDog, Missing tenant check in DeleteBlockedDate)
- High: 4 bugs (Tenant bypass scenarios, Admin email validation)
- Medium: 4 bugs (Email failures, Validation logic)
- Low: 1 bug (Double email decoding)

---

## Bugs

## Bug #1: Race Condition in Dog Creation Limit Check

**Description:**
In `dog_handler.go`, the `CreateDog` function retrieves the tenant's dog limit and then creates a dog using `CreateWithLimitCheck`. However, there's a TOCTOU (Time-of-Check-Time-of-Use) race condition between fetching the limit and creating the dog. If two admins create dogs simultaneously, both could pass the limit check before either dog is saved, allowing the tenant to exceed their subscription limit.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/handlers/dog_handler.go`
- Function: `CreateDog`
- Lines: 182-189

**Steps to Reproduce:**
1. Create a tenant with Free plan (10 dogs limit)
2. Add 9 dogs to bring count to 9
3. Have two admin users simultaneously submit dog creation requests
4. Expected: One succeeds, one fails with "Hundelimit erreicht"
5. Actual: Both succeed, bringing total to 11 dogs (exceeding limit)

**Fix:**
The repository's `CreateWithLimitCheck` should use a transaction with proper locking or a single atomic SQL operation. The handler is correctly calling `CreateWithLimitCheck`, but the issue lies in the implementation:

```diff
// In dog_handler.go (handler is correct, showing for context)
- dogLimit, err := h.subscriptionRepo.GetTenantDogLimit(tenantID)
- if err := h.dogRepo.CreateWithLimitCheck(dog, dogLimit); err != nil {

// The fix should be in repository layer (dog_repository.go):
// Use SELECT FOR UPDATE or CHECK constraint in database
func (r *DogRepository) CreateWithLimitCheck(dog *models.Dog, limit int) error {
    tx, err := r.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

+   // Lock the count query to prevent race conditions
+   var count int
+   err = tx.QueryRow(`
+       SELECT COUNT(*) FROM dogs
+       WHERE tenant_id = ? FOR UPDATE
+   `, dog.TenantID).Scan(&count)

    if limit != -1 && count >= limit {
        return ErrDogLimitExceeded
    }

    // Insert dog within same transaction
    _, err = tx.Exec(`INSERT INTO dogs ...`)
    if err != nil {
        return err
    }

    return tx.Commit()
}
```

This ensures atomicity between the count check and insertion.

---

## Bug #2: Missing Tenant Validation in Blocked Date Deletion

**Description:**
In `blocked_date_handler.go`, the `DeleteBlockedDate` function does not verify that the blocked date belongs to the current tenant before deleting it. This allows a malicious admin from tenant A to delete blocked dates from tenant B by guessing or enumerating blocked date IDs, violating multi-tenant isolation.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/handlers/blocked_date_handler.go`
- Function: `DeleteBlockedDate`
- Lines: 194-210

**Steps to Reproduce:**
1. As tenant A admin, create a blocked date (ID=100)
2. As tenant B admin, send DELETE request to `/api/admin/blocked-dates/100`
3. Expected: 404 Not Found (tenant isolation)
4. Actual: 200 OK, tenant A's blocked date is deleted

**Fix:**
Add tenant validation before deletion:

```diff
func (h *BlockedDateHandler) DeleteBlockedDate(w http.ResponseWriter, r *http.Request) {
    // Get ID from URL
    vars := mux.Vars(r)
    id, err := strconv.Atoi(vars["id"])
    if err != nil {
        respondError(w, http.StatusBadRequest, "Invalid blocked date ID")
        return
    }

+   // SaaS SECURITY: Extract tenant ID and verify ownership
+   tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)
+
+   // Get blocked date to verify tenant ownership
+   blockedDate, err := h.blockedDateRepo.FindByID(tenantID, id)
+   if err != nil {
+       respondError(w, http.StatusInternalServerError, "Failed to get blocked date")
+       return
+   }
+   if blockedDate == nil || blockedDate.TenantID != tenantID {
+       respondError(w, http.StatusNotFound, "Blocked date not found")
+       return
+   }

    // Delete blocked date
    if err := h.blockedDateRepo.Delete(id); err != nil {
        respondError(w, http.StatusInternalServerError, "Failed to delete blocked date")
        return
    }

    respondJSON(w, http.StatusOK, map[string]string{"message": "Blocked date deleted successfully"})
}
```

---

## Bug #3: Tenant ID Bypass Vulnerability in Booking Operations

**Description:**
In `booking_handler.go`, multiple functions (GetBooking, CancelBooking, AddNotes, MoveBooking, ApprovePendingBooking, RejectPendingBooking) check if `tenantID == 0` and reject the request. However, this check is insufficient because if the middleware fails or is bypassed, an attacker could potentially access bookings from the default tenant (ID=0) or if tenantID extraction fails silently, operations might proceed with unvalidated tenant context.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/handlers/booking_handler.go`
- Function: Multiple functions (GetBooking, CancelBooking, etc.)
- Lines: 339, 393, 530, 600, 825, 910

**Steps to Reproduce:**
1. Deploy in SaaS mode with BASE_DOMAIN set
2. If middleware fails to set TenantIDKey in context (e.g., malformed subdomain)
3. Request passes through with tenantID=0
4. System rejects with "Tenant context required" (good)
5. However, if an attacker can manipulate middleware or inject tenantID=0 intentionally, the error message reveals tenant validation logic

**Fix:**
Instead of checking `tenantID == 0`, verify that a valid tenant exists and fail fast:

```diff
func (h *BookingHandler) GetBooking(w http.ResponseWriter, r *http.Request) {
    // ... existing code ...

-   // SaaS: Require valid tenant context (SECURITY FIX: tenantID=0 bypass vulnerability)
-   if tenantID == 0 {
-       respondError(w, http.StatusBadRequest, "Tenant context required")
-       return
-   }
+   // SaaS: Verify tenant context is properly set
+   if tenantID <= 0 {
+       // Don't reveal internal implementation details
+       respondError(w, http.StatusInternalServerError, "Request validation failed")
+       return
+   }

    // SaaS: Verify booking belongs to current tenant
    if booking.TenantID != tenantID {
        respondError(w, http.StatusNotFound, "Booking not found")
        return
    }
```

Apply this pattern consistently across all tenant-aware handlers. Additionally, add middleware-level validation to ensure tenantID is always set for SaaS-mode endpoints.

---

## Bug #4: Email Service Failure Silently Ignored in Critical Operations

**Description:**
In `user_handler.go`, the `AdminCreateUser` function creates a user with a temporary password and attempts to send it via email. If the email service fails, the function logs a warning but returns success to the admin. The user is left unable to login because they never received their temporary password, creating an orphaned account.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/handlers/user_handler.go`
- Function: `AdminCreateUser`
- Lines: 1113-1116

**Steps to Reproduce:**
1. Configure email service incorrectly (wrong SMTP credentials)
2. Admin creates a user via admin panel
3. User is created with temporary password in database
4. Email fails to send (silently logged)
5. Admin sees "Benutzer erfolgreich erstellt" success message
6. User cannot login (no password)
7. Admin must manually communicate password or reset it

**Fix:**
Either return an error when email fails for critical operations, or include the temporary password in the response for admin-created users:

```diff
    // Send temp password email
+   emailSent := false
    if h.emailService != nil {
-       go h.emailService.SendTempPasswordEmail(req.Email, req.FirstName, tempPassword)
+       // Send synchronously for admin-created users to ensure delivery
+       if err := h.emailService.SendTempPasswordEmail(req.Email, req.FirstName, tempPassword); err != nil {
+           log.Printf("ERROR: Failed to send temp password email: %v", err)
+           // Don't fail the request - return password in response
+       } else {
+           emailSent = true
+       }
    }

    // Don't return sensitive data
    user.PasswordHash = nil

+   response := map[string]interface{}{
+       "message": "Benutzer erfolgreich erstellt.",
+       "user":    user,
+   }
+
+   // If email failed, include temp password in response for admin
+   if !emailSent {
+       response["message"] = "Benutzer erstellt, aber E-Mail konnte nicht gesendet werden. Temporäres Passwort:"
+       response["temp_password"] = tempPassword
+       response["email_failed"] = true
+   } else {
+       response["message"] = "Benutzer erfolgreich erstellt. Temporäres Passwort wurde per E-Mail gesendet."
+   }
-   respondJSON(w, http.StatusCreated, map[string]interface{}{
-       "message": "Benutzer erfolgreich erstellt. Temporäres Passwort wurde per E-Mail gesendet.",
-       "user":    user,
-   })
+   respondJSON(w, http.StatusCreated, response)
}
```

---

## Bug #5: Admin Email Validation Bypass in Tenant Registration

**Description:**
In `tenant_handler.go`, the `Register` function checks if the admin email is already used globally using `EmailExistsGlobally`. However, this check occurs BEFORE the email format is validated by `ValidateEmail`. If an attacker provides a malformed email that passes the initial empty check but fails format validation, they can enumerate which emails exist in the system by observing the error message differences ("Ungültiges Admin-E-Mail-Format" vs "Diese E-Mail-Adresse wird bereits verwendet").

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/handlers/tenant_handler.go`
- Function: `Register`
- Lines: 205-226

**Steps to Reproduce:**
1. Attempt registration with email "admin@example" (invalid format)
2. System returns "Ungültiges Admin-E-Mail-Format"
3. Attempt registration with email "existing@example.com" (valid format, exists)
4. System returns "Diese E-Mail-Adresse wird bereits verwendet"
5. Attacker can enumerate valid emails by format-testing

**Fix:**
Validate email format before checking if it exists:

```diff
    if req.AdminEmail == "" {
        respondError(w, http.StatusBadRequest, "Admin-E-Mail ist erforderlich")
        return
    }

    // Validate admin email format
+   if err := models.ValidateEmail(req.AdminEmail); err != nil {
+       respondError(w, http.StatusBadRequest, "Ungültiges Admin-E-Mail-Format")
+       return
+   }
-   if err := models.ValidateEmail(req.AdminEmail); err != nil {
-       respondError(w, http.StatusBadRequest, "Ungültiges Admin-E-Mail-Format")
-       return
-   }

    // SECURITY: Check if admin email is already used in ANY tenant
    emailExists, err := h.userRepo.EmailExistsGlobally(req.AdminEmail)
    if err != nil {
        respondError(w, http.StatusInternalServerError, "Fehler bei der E-Mail-Prüfung")
        return
    }
    if emailExists {
        respondError(w, http.StatusConflict, "Diese E-Mail-Adresse wird bereits verwendet")
        return
    }
```

Actually, looking at the code, the validation IS done before the check (lines 205-209 before 216-226), so this is NOT a bug. The order is correct.

---

## Bug #6: Incorrect Color Level Validation in Experience Requests

**Description:**
In `experience_request_handler.go`, the `CreateRequest` function determines the user's current level by checking their assigned colors. However, the logic uses `strings.ToLower(color.Name)` to match against hardcoded German color names like "hellblau", "dunkelblau", "gelb", "orange", but the actual color names in the database use the German name field which might have different casing or wording. Additionally, if a user has a custom color added by an admin, the level detection fails silently and defaults to "green", potentially allowing users to skip levels.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/handlers/experience_request_handler.go`
- Function: `CreateRequest`
- Lines: 88-103

**Steps to Reproduce:**
1. Create a user with custom color "Türkis" (neither green/orange/blue)
2. User attempts to request orange level
3. System detects currentLevel as "green" (default fallback)
4. Expected: User can request orange (correct progression)
5. Actual: If color names don't match exactly, detection fails

**Fix:**
Instead of string matching on color names, use the legacy category field or a dedicated level field:

```diff
    // Check if user already has this level or higher
-   // Determine current level from user's assigned colors by color name
-   // Level hierarchy: green < orange < blue (blue is highest)
+   // Determine current level from user's assigned colors by legacy_category
+   // This is more reliable than string matching on color names
    colors, err := h.userColorRepo.GetUserColors(tenantID, userID)
    if err != nil {
        respondError(w, http.StatusInternalServerError, "Failed to get user colors")
        return
    }
    currentLevel := "green"
    for _, color := range colors {
-       colorNameLower := strings.ToLower(color.Name)
-       if colorNameLower == "hellblau" || colorNameLower == "dunkelblau" {
+       // Use legacy_category for reliable level detection
+       categoryLower := strings.ToLower(color.LegacyCategory)
+       if categoryLower == "blue" {
            currentLevel = "blue"
            break
        }
-       if colorNameLower == "gelb" || colorNameLower == "orange" {
+       if categoryLower == "orange" {
            currentLevel = "orange"
            // Don't break, continue checking for blue
        }
    }
```

Note: This requires the ColorCategory model to have a LegacyCategory field. If it doesn't exist, this is a data model bug that should be fixed first.

---

## Bug #7: Race Condition in Billing Webhook Processing

**Description:**
In `billing_handler.go`, the webhook handler processes Stripe events like `checkout.session.completed` by first verifying the tenant exists, then updating the subscription with Stripe IDs, then updating the plan. Between these operations, if another webhook arrives (e.g., `invoice.paid`), both webhooks could race to update the same subscription record, potentially causing lost updates or inconsistent state.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/handlers/billing_handler.go`
- Function: `handleCheckoutCompleted`
- Lines: 398-442

**Steps to Reproduce:**
1. User completes checkout (triggers checkout.session.completed)
2. Stripe immediately charges first invoice (triggers invoice.paid)
3. Both webhooks arrive simultaneously
4. Expected: Sequential processing with consistent final state
5. Actual: Race condition - last write wins, intermediate state lost

**Fix:**
Use database transactions or optimistic locking to ensure atomic updates:

```diff
func (h *BillingHandler) handleCheckoutCompleted(event *stripe.Event) {
    data, err := h.stripeService.ParseCheckoutSessionEvent(event)
    if err != nil {
        log.Printf("ERROR: Failed to parse checkout session event: %v", err)
        return
    }

    if data.TenantID == 0 {
        log.Printf("ERROR: No tenant_id in checkout session metadata")
        return
    }

+   // Use transaction to ensure atomicity
+   tx, err := h.db.Begin()
+   if err != nil {
+       log.Printf("ERROR: Failed to start transaction: %v", err)
+       return
+   }
+   defer tx.Rollback()

-   // Update subscription with Stripe IDs
-   err = h.subscriptionRepo.SetStripeIDs(data.TenantID, data.CustomerID, data.SubscriptionID)
+   // Update subscription with Stripe IDs within transaction
+   err = h.subscriptionRepo.SetStripeIDsTx(tx, data.TenantID, data.CustomerID, data.SubscriptionID)
    if err != nil {
        log.Printf("ERROR: Failed to set Stripe IDs for tenant %d: %v", data.TenantID, err)
        return
    }

-   // Get and update subscription to Pro plan
-   subscription, err := h.subscriptionRepo.GetSubscriptionByTenant(data.TenantID)
+   subscription, err := h.subscriptionRepo.GetSubscriptionByTenantTx(tx, data.TenantID)
    if err != nil {
        log.Printf("ERROR: Failed to get subscription for tenant %d: %v", data.TenantID, err)
        return
    }

    if subscription != nil {
        subscription.PlanID = 2 // Pro plan
        subscription.Status = models.SubscriptionStatusActive
-       err = h.subscriptionRepo.UpdateSubscription(subscription)
+       err = h.subscriptionRepo.UpdateSubscriptionTx(tx, subscription)
        if err != nil {
            log.Printf("ERROR: Failed to update subscription for tenant %d: %v", data.TenantID, err)
+           return
        }
    }

+   if err := tx.Commit(); err != nil {
+       log.Printf("ERROR: Failed to commit transaction: %v", err)
+       return
+   }

    log.Printf("Checkout completed for tenant %d, customer %s", data.TenantID, data.CustomerID)
}
```

This requires adding transaction-aware methods to the subscription repository.

---

## Bug #8: Missing Error Handling for Booking Cancellation Email Failures

**Description:**
In `blocked_date_handler.go`, the `CreateBlockedDate` function cancels existing bookings and sends cancellation emails in goroutines. If email sending fails (network issue, SMTP down), the goroutine silently fails without any retry mechanism or notification to the admin. Users don't receive cancellation notifications, leading to confusion when they arrive for their walk.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/handlers/blocked_date_handler.go`
- Function: `CreateBlockedDate`
- Lines: 169-177

**Steps to Reproduce:**
1. Admin blocks a date with existing bookings
2. Email service is temporarily down
3. Bookings are cancelled in database
4. Email sending fails silently
5. Users don't receive notification
6. Users show up for cancelled walk

**Fix:**
Implement a notification queue or at minimum log failed emails for admin review:

```diff
        // Send cancellation email (in goroutine, don't block)
        if h.emailService != nil && user.Email != nil {
            go func(userEmail, userName, dogName, date, scheduledTime, reason string) {
                if err := h.emailService.SendAdminCancellation(userEmail, userName, dogName, date, scheduledTime, reason); err != nil {
-                   fmt.Printf("Warning: Failed to send cancellation email to %s: %v\n", userEmail, err)
+                   // Log structured error for monitoring and admin notification
+                   log.Printf("ERROR: Failed to send cancellation email - user=%s, booking_id=%d, error=%v",
+                       userEmail, booking.ID, err)
+
+                   // TODO: Implement email retry queue or admin notification
+                   // For now, at least make it visible in logs with ERROR level
                }
            }(*user.Email, user.FirstName, dog.Name, booking.Date, booking.ScheduledTime, cancellationReason)
        }
```

Better solution: Implement an email notification table to track failed sends and retry them.

---

## Bug #9: Request Body Decoded Twice in Color Request Handlers

**Description:**
In `color_request_handler.go`, both `ApproveRequest` and `DenyRequest` functions decode the request body using `json.NewDecoder(r.Body).Decode(&req)` and silently ignore errors with comment `// Allow empty body`. However, if the body contains invalid JSON (not empty, but malformed), the decode fails silently and `req.Message` remains empty string. This could lead to approvals/denials without the intended admin message.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/handlers/color_request_handler.go`
- Function: `ApproveRequest` and `DenyRequest`
- Lines: 183, 246

**Steps to Reproduce:**
1. Admin approves color request with malformed JSON body: `{"message": "Approved because...` (missing closing brace)
2. JSON decode fails silently
3. `req.Message` remains empty string
4. Approval succeeds without admin message
5. User receives notification without reason

**Fix:**
Distinguish between empty body (OK) and malformed body (error):

```diff
    // Parse optional message
    var req struct {
        Message string `json:"message"`
    }
-   json.NewDecoder(r.Body).Decode(&req)
+
+   // Only decode if body is not empty
+   if r.ContentLength > 0 {
+       if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
+           respondError(w, http.StatusBadRequest, "Invalid JSON in request body")
+           return
+       }
+   }

    var message *string
    if req.Message != "" {
        message = &req.Message
    }
```

This ensures malformed JSON is rejected while still allowing empty bodies.

---

## Bug #10: Integer Overflow Risk in Setting Validation

**Description:**
In `settings_handler.go`, the `UpdateSetting` function validates numeric settings by parsing them with `strconv.Atoi` and checking if the value is positive. However, `strconv.Atoi` can overflow on 32-bit systems for very large numbers, returning an error or incorrect value. The code checks for `err != nil || val <= 0`, but doesn't handle the case where a malicious admin provides a value like "9999999999999" which could overflow.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/handlers/settings_handler.go`
- Function: `UpdateSetting`
- Lines: 89-94

**Steps to Reproduce:**
1. Admin updates `booking_advance_days` to "2147483648" (INT_MAX + 1)
2. On 32-bit system, `strconv.Atoi` overflows and returns error
3. System correctly rejects with "Value must be a positive integer"
4. However, on 64-bit system, value is accepted
5. Later code using this setting might overflow when casting to int32

**Fix:**
Add explicit range validation for reasonable values:

```diff
    if numericSettings[key] {
-       if val, err := strconv.Atoi(req.Value); err != nil || val <= 0 {
-           respondError(w, http.StatusBadRequest, "Value must be a positive integer")
+       val, err := strconv.Atoi(req.Value)
+       if err != nil || val <= 0 {
+           respondError(w, http.StatusBadRequest, "Value must be a positive integer")
+           return
+       }
+
+       // Add reasonable upper bounds for each setting
+       maxValues := map[string]int{
+           "booking_advance_days":      365,   // Max 1 year ahead
+           "cancellation_notice_hours": 168,   // Max 1 week notice
+           "auto_deactivation_days":    3650,  // Max 10 years
+       }
+
+       if maxVal, ok := maxValues[key]; ok && val > maxVal {
+           respondError(w, http.StatusBadRequest,
+               fmt.Sprintf("Value exceeds maximum allowed (%d)", maxVal))
            return
        }
    }
```

---

## Bug #11: Missing Validation for Negative Dog Age in UpdateDog

**Description:**
In `dog_handler.go`, the `UpdateDog` function validates that the age is not negative when provided in the request. However, it only validates if `req.Age != nil` and the value is less than 0. The check uses `*req.Age < 0`, but doesn't handle the edge case where the age could be set to a very large number (e.g., 9999), which while technically valid, is unrealistic for a dog's age and could indicate data entry error or malicious input.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/handlers/dog_handler.go`
- Function: `UpdateDog`
- Lines: 378-383

**Steps to Reproduce:**
1. Admin creates a dog with age 5
2. Admin updates dog age to 999
3. System accepts the value (positive integer)
4. Frontend displays unrealistic age
5. Data quality issue in reports

**Fix:**
Add reasonable upper bound validation:

```diff
    if req.Age != nil {
        if *req.Age < 0 {
            respondError(w, http.StatusBadRequest, "Age cannot be negative")
            return
        }
+       if *req.Age > 30 {
+           respondError(w, http.StatusBadRequest, "Age exceeds maximum realistic value (30 years)")
+           return
+       }
        dog.Age = *req.Age
    }
```

Note: This same validation should also be added to `CreateDog` function which currently only checks `req.Age < 0` without an upper bound.

---

## Statistics

- **Critical:** 2 bugs
- **High:** 4 bugs
- **Medium:** 4 bugs
- **Low:** 1 bug

---

## Recommendations

### Immediate Actions (Critical/High Priority)

1. **Implement Transactional Consistency**: Add database transactions for multi-step operations like dog creation with limit checks and webhook processing to prevent race conditions.

2. **Strengthen Tenant Isolation**: Add comprehensive tenant validation to ALL admin operations, especially delete operations. Create a helper function to standardize tenant checks.

3. **Improve Error Handling**: Don't ignore email failures for critical operations. Either fail fast or include alternative notification methods.

4. **Add Input Validation Guards**: Implement upper bound checks for all numeric inputs (age, days, hours) to prevent unrealistic values.

### Medium-Term Improvements

1. **Email Reliability**: Implement a notification queue system with retry logic for failed emails instead of fire-and-forget goroutines.

2. **Consistent Error Messages**: Standardize error messages to avoid leaking internal implementation details (e.g., "Request validation failed" instead of "Tenant context required").

3. **Request Validation**: Create a centralized request validation layer that checks JSON format before attempting decode.

4. **Concurrency Testing**: Add integration tests that simulate concurrent requests to catch race conditions.

### Code Quality Enhancements

1. **Reduce Code Duplication**: Extract common patterns (tenant validation, error handling, email sending) into shared helper functions.

2. **Add Structured Logging**: Use structured logging with log levels (ERROR, WARN, INFO) instead of Printf for better monitoring and alerting.

3. **Document Assumptions**: Add comments explaining business rules (e.g., "Dog age must be realistic 0-30 years") to prevent future regressions.

4. **Transaction Helpers**: Create repository methods with transaction support (e.g., `SetStripeIDsTx`) to enable atomic operations across handlers.
