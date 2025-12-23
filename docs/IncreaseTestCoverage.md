# Test Coverage Improvement Plan: 45% to 80%

## Objective
Increase Go test coverage from ~45% to industry-standard 80% with high-quality tests designed to find bugs.

## Current State

| Package | Coverage | Target | Action |
|---------|----------|--------|--------|
| `internal/cron` | 23.5% | 80% | CRITICAL - Add ~15 tests |
| `internal/handlers` | 44.0% | 80% | 6 handlers have NO tests |
| `internal/services` | 44.8% | 75% | Expand existing tests |
| `internal/models` | 47.9% | 75% | Add validation tests |
| `internal/middleware` | 53.7% | 80% | Add multi-tenant tests |
| `internal/database` | 53.0% | 70% | Low priority |
| `internal/repository` | 65.3% | 80% | Add edge cases |
| `internal/config` | 0% | 70% | NEW test file |
| `internal/logging` | 0% | 70% | NEW test file |
| `internal/version` | 100% | 100% | DONE |

## Execution Phases

### Phase 1: Cron Package (23.5% -> 80%)
**File**: `internal/cron/cron_test.go`

Tests to add:
- `TestCronService_SendBookingReminders` - reminders 1-2 hours before walk
- `TestCronService_SendBookingReminders_SkipsAlreadyReminded`
- `TestCronService_SendBookingReminders_SkipsNoEmail`
- `TestCronService_SendBookingReminders_NilEmailService`
- `TestCronService_ResetDemoTenant` - demo reset at midnight
- `TestCronService_ResetDemoTenant_SkipsNoDemoTenant`
- `TestCronService_ResetDemoTenant_RespectsNextResetAt`
- `TestCronService_AutoDeactivateUsersForTenant_RespectsTenantSettings`
- `TestCronService_AutoDeactivateUsersForTenant_DefaultDays`
- `TestCronService_RunPeriodically_ExecutesImmediately`
- `TestCronService_RunPeriodically_StopsOnSignal`
- `TestCronService_RunDaily_SchedulesCorrectly`
- `TestCronService_RunDaily_BerlinTimezone`

**Bug categories**: Timezone edge cases, nil pointer dereference, goroutine leaks

---

### Phase 2: Handler Tests (6 NEW test files)

#### 2.1 `central_admin_handler_test.go` (NEW)
- `TestCentralAdminHandler_GetPlatformStats` - all stats
- `TestCentralAdminHandler_ListTenants` - filtering, search
- `TestCentralAdminHandler_GetTenant` - 404 for non-existent
- `TestCentralAdminHandler_UpdateTenant` - validation
- `TestCentralAdminHandler_ActivateTenant`
- `TestCentralAdminHandler_DeactivateTenant`
- `TestCentralAdminHandler_ListCentralAdmins`
- `TestCentralAdminHandler_PromoteToCentralAdmin` - already admin
- `TestCentralAdminHandler_DemoteFromCentralAdmin` - self-demotion blocked
- `TestCentralAdminHandler_GetTenantUsers` - sensitive data removed
- `TestCentralAdminHandler_SearchUsers` - across tenants
- `TestCentralAdminHandler_ExportTenantData` - GDPR

**Bug categories**: Authorization bypass, self-demotion, cross-tenant access

#### 2.2 `tenant_handler_test.go` (NEW)
- `TestTenantHandler_Register` - creates tenant + super admin
- `TestTenantHandler_Register_RejectsReservedSlug` - security
- `TestTenantHandler_Register_ValidatesSlugFormat`
- `TestTenantHandler_Register_RejectsDuplicateSlug`
- `TestTenantHandler_CheckSlug` - availability check
- `TestTenantHandler_GetCurrentTenant`
- `TestTenantHandler_UpdateTenant` - requires admin
- `TestTenantHandler_GetTenantStats`

**Bug categories**: Reserved slug bypass, duplicate detection

#### 2.3 `theme_handler_test.go` (NEW)
- `TestThemeHandler_GetCSS` - custom/preset colors
- `TestThemeHandler_GetCSS_DefaultTheme`
- `TestThemeHandler_GetPresets` - all presets
- `TestThemeHandler_GetCurrentTheme`
- `TestThemeHandler_UpdateTheme` - validation

#### 2.4 `user_color_handler_test.go` (NEW)
- `TestUserColorHandler_GetUserColors` - requires admin
- `TestUserColorHandler_AddColorToUser` - conflict if exists
- `TestUserColorHandler_RemoveColorFromUser`
- `TestUserColorHandler_SetUserColors` - replaces all

#### 2.5 `walk_report_handler_test.go` (NEW)
- `TestWalkReportHandler_CreateReport` - booking must be completed
- `TestWalkReportHandler_CreateReport_PreventsDuplicate`
- `TestWalkReportHandler_CreateReport_RequiresOwnership`
- `TestWalkReportHandler_GetReport`
- `TestWalkReportHandler_GetDogWalkReports` - with stats
- `TestWalkReportHandler_UpdateReport`
- `TestWalkReportHandler_DeleteReport`
- `TestWalkReportHandler_UploadPhoto` - MIME validation
- `TestWalkReportHandler_UploadPhoto_EnforcesLimit`
- `TestWalkReportHandler_DeletePhoto`

**Bug categories**: Authorization bypass, file upload security

#### 2.6 `health_handler_test.go` (NEW)
- `TestHealthHandler_Health` - returns ok status

---

### Phase 3: Services Package (44.8% -> 75%)

**Files to expand**:
- `auth_service_test.go`:
  - `TestAuthService_GenerateJWT_IncludesTenantID`
  - `TestAuthService_ValidateJWT_ExpiredToken`
  - `TestAuthService_ValidateJWT_InvalidSecret`

- `email_service_test.go`:
  - `TestEmailService_SendBookingReminder`
  - `TestEmailService_SendAccountDeactivated`
  - `TestEmailService_BCCAdmin`
  - `TestEmailService_NilServiceHandling`

- `booking_time_service_test.go`:
  - `TestBookingTimeService_GetAvailableSlots_BlockedPeriods`
  - `TestBookingTimeService_EdgeCaseTimes` (23:59, 00:00)

- **NEW**: `provisioning_service_test.go`:
  - `TestProvisioningService_ProvisionTenant`
  - `TestProvisioningService_CreatesDefaultColors`
  - `TestProvisioningService_CreatesDefaultSettings`

---

### Phase 4: Middleware & Models

**`middleware_test.go`** additions:
- `TestCORSMiddleware_AllowsConfiguredOrigins`
- `TestCORSMiddleware_RejectsUnconfigured`
- `TestAuthMiddleware_ValidatesTenantIDMatch`
- `TestRequireCentralAdmin`
- `TestSecurityHeadersMiddleware`

**Model test additions**:
- `TestDog_Validate`
- `TestTenant_Validate`
- `TestTenantSettings_Validate`
- `TestThemeColors_Validate`
- `TestCreateWalkReportRequest_Validate`

---

### Phase 5: Config & Logging (0% -> 70%)

**NEW**: `internal/config/config_test.go`
- `TestConfig_Load`
- `TestConfig_GetDBConfig_SQLite`
- `TestConfig_GetDBConfig_MySQL`
- `TestConfig_GetDBConfig_PostgreSQL`
- `TestGetEnv_Default`
- `TestGetEnvAsInt`
- `TestGetEnvAsBool`

**NEW**: `internal/logging/logger_test.go`
- `TestLogger_NewLogger`
- `TestLogger_Write`
- `TestLogger_Rotation`
- `TestLogger_CleanOldLogs`

---

### Phase 6: Repository Edge Cases (65.3% -> 80%)

- `TestBookingRepository_GetForReminders_TimezoneEdgeCases`
- `TestUserRepository_FindInactiveUsers_ExcludesAdmins`
- `TestUserRepository_DeleteAccount_GDPR_Completeness`
- `TestTenantRepository_FindBySlug_CaseInsensitive`

---

## Bug Categories to Target

| Category | Tests | Examples |
|----------|-------|----------|
| **Security** | 15+ | CORS bypass, JWT tenant mismatch, file upload MIME spoofing |
| **Authorization** | 20+ | Cross-tenant access, self-demotion, owner vs admin |
| **Data Integrity** | 10+ | Double booking, duplicate reports, GDPR anonymization |
| **Edge Cases** | 15+ | Timezone/DST, boundary dates, nil/empty data, Unicode |
| **Concurrency** | 5+ | Goroutine leaks, race conditions |

---

## Test Pattern (from existing codebase)

```go
func TestHandler_Method(t *testing.T) {
    db := testutil.SetupTestDB(t)
    cfg := &config.Config{JWTSecret: "test-secret", JWTExpirationHours: 24}
    handler := NewHandler(db, cfg)

    // Seed test data
    userID := testutil.SeedTestUser(t, db, "test@example.com", "Test", "green")

    t.Run("success case", func(t *testing.T) {
        req := httptest.NewRequest("GET", "/api/endpoint", nil)
        ctx := context.WithValue(req.Context(), middleware.TenantIDKey, 1)
        ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
        req = req.WithContext(ctx)

        rec := httptest.NewRecorder()
        handler.Method(rec, req)

        if rec.Code != http.StatusOK {
            t.Errorf("Expected 200, got %d: %s", rec.Code, rec.Body.String())
        }
    })

    t.Run("error case", func(t *testing.T) {
        // Test error conditions
    })
}
```

---

## Estimation

| Phase | New Test Functions | Estimated Coverage Impact |
|-------|-------------------|---------------------------|
| 1. Cron | ~15 | 45% -> 50% |
| 2. Handlers | ~50 | 50% -> 65% |
| 3. Services | ~20 | 65% -> 72% |
| 4. Middleware/Models | ~15 | 72% -> 76% |
| 5. Config/Logging | ~15 | 76% -> 79% |
| 6. Repository | ~10 | 79% -> 82% |

**Total**: ~125 new test functions

---

## Files to Create/Modify

### NEW Files:
1. `internal/handlers/central_admin_handler_test.go`
2. `internal/handlers/tenant_handler_test.go`
3. `internal/handlers/theme_handler_test.go`
4. `internal/handlers/user_color_handler_test.go`
5. `internal/handlers/walk_report_handler_test.go`
6. `internal/handlers/health_handler_test.go`
7. `internal/config/config_test.go`
8. `internal/logging/logger_test.go`
9. `internal/services/provisioning_service_test.go`

### Expand Existing:
1. `internal/cron/cron_test.go`
2. `internal/services/auth_service_test.go`
3. `internal/services/email_service_test.go`
4. `internal/services/booking_time_service_test.go`
5. `internal/middleware/middleware_test.go`
6. `internal/models/*_test.go`
7. `internal/repository/*_test.go`

---

## Validation Commands

```bash
# Run all tests
go test ./... -v

# Coverage report
go test ./... -cover

# Detailed coverage per package
go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out

# Run specific package
go test ./internal/handlers/... -v

# Run build script
./bat.sh
```

---

## Success Criteria

1. Overall coverage >= 80%
2. All tests pass (`go test ./...`)
3. Build succeeds (`./bat.sh`)
4. No flaky tests (run 3x to verify)
5. Bugs found during test writing documented
