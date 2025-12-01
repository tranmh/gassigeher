# Booking Time Restrictions Test Plan

**Version:** 1.0
**Date:** 2025-01-23
**Status:** Active
**Related Document:** [BookingTimeImplementationPlan.md](BookingTimeImplementationPlan.md)

---

## Executive Summary

This document outlines comprehensive testing procedures for the booking time restrictions feature in the Gassigeher dog walking system. The test plan covers unit tests, integration tests, API tests, UI tests, end-to-end workflows, security testing, and performance validation.

**Testing Scope:**
- ✅ Time window validation (weekday/weekend/holiday)
- ✅ Holiday detection and caching
- ✅ Morning walk approval workflow
- ✅ Admin time rule management
- ✅ User booking experience
- ✅ API security and authorization
- ✅ Database migrations and integrity
- ✅ Performance under load

**Testing Tools:**
- Go test framework (`go test`)
- HTTP testing (`httptest` package)
- SQLite in-memory database for tests
- Manual browser testing (Chrome, Firefox, Safari, Edge)
- API testing tools (Postman/curl)
- Load testing (optional: Apache Bench)

---

## Phase 1: Unit Testing - Services // DONE

### 1.1 BookingTimeService Tests

**File:** `internal/services/booking_time_service_test.go`

#### Test 1.1.1: ValidateBookingTime - Weekday Allowed Times

**Purpose:** Verify booking validation for allowed weekday time windows

**Test Cases:**

| Test Case | Date | Time | Expected Result |
|-----------|------|------|-----------------|
| TC-1.1.1-A | 2025-01-27 (Mon) | 09:30 | Success (morning window) |
| TC-1.1.1-B | 2025-01-27 (Mon) | 12:15 | Success (open period) |
| TC-1.1.1-C | 2025-01-27 (Mon) | 14:45 | Success (afternoon window) |
| TC-1.1.1-D | 2025-01-27 (Mon) | 18:30 | Success (evening window) |
| TC-1.1.1-E | 2025-01-28 (Tue) | 10:00 | Success (morning window) |

**Code:**
```go
func TestValidateBookingTime_WeekdayAllowed(t *testing.T) {
    testCases := []struct {
        name string
        date string
        time string
        wantErr bool
    }{
        {"Morning window", "2025-01-27", "09:30", false},
        {"Open period", "2025-01-27", "12:15", false},
        {"Afternoon window", "2025-01-27", "14:45", false},
        {"Evening window", "2025-01-27", "18:30", false},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            err := service.ValidateBookingTime(tc.date, tc.time)
            if (err != nil) != tc.wantErr {
                t.Errorf("ValidateBookingTime() error = %v, wantErr %v", err, tc.wantErr)
            }
        })
    }
}
```

#### Test 1.1.2: ValidateBookingTime - Weekday Blocked Times

**Purpose:** Verify booking rejection for blocked weekday periods

**Test Cases:**

| Test Case | Date | Time | Block Period | Expected Error |
|-----------|------|------|--------------|----------------|
| TC-1.1.2-A | 2025-01-27 (Mon) | 13:00 | Lunch Block | "Zeit ist gesperrt: Lunch Block" |
| TC-1.1.2-B | 2025-01-27 (Mon) | 13:45 | Lunch Block | "Zeit ist gesperrt: Lunch Block" |
| TC-1.1.2-C | 2025-01-27 (Mon) | 17:00 | Feeding Block | "Zeit ist gesperrt: Feeding Block" |
| TC-1.1.2-D | 2025-01-27 (Mon) | 17:30 | Feeding Block | "Zeit ist gesperrt: Feeding Block" |
| TC-1.1.2-E | 2025-01-27 (Mon) | 08:00 | Outside windows | "Zeit ist außerhalb" |
| TC-1.1.2-F | 2025-01-27 (Mon) | 20:00 | Outside windows | "Zeit ist außerhalb" |

**Code:**
```go
func TestValidateBookingTime_WeekdayBlocked(t *testing.T) {
    testCases := []struct {
        name string
        date string
        time string
        wantErrContains string
    }{
        {"Lunch block start", "2025-01-27", "13:00", "Lunch Block"},
        {"Lunch block middle", "2025-01-27", "13:45", "Lunch Block"},
        {"Feeding block start", "2025-01-27", "17:00", "Feeding Block"},
        {"Feeding block middle", "2025-01-27", "17:30", "Feeding Block"},
        {"Before opening", "2025-01-27", "08:00", "außerhalb"},
        {"After closing", "2025-01-27", "20:00", "außerhalb"},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            err := service.ValidateBookingTime(tc.date, tc.time)
            if err == nil {
                t.Error("Expected error, got nil")
            }
            if !strings.Contains(err.Error(), tc.wantErrContains) {
                t.Errorf("Error %v should contain %q", err, tc.wantErrContains)
            }
        })
    }
}
```

#### Test 1.1.3: ValidateBookingTime - Weekend Times

**Purpose:** Verify weekend time window validation

**Test Cases:**

| Test Case | Date | Time | Expected Result |
|-----------|------|------|-----------------|
| TC-1.1.3-A | 2025-01-25 (Sat) | 10:00 | Success (morning window) |
| TC-1.1.3-B | 2025-01-25 (Sat) | 15:00 | Success (afternoon window) |
| TC-1.1.3-C | 2025-01-26 (Sun) | 11:30 | Success (morning window) |
| TC-1.1.3-D | 2025-01-26 (Sun) | 16:30 | Success (afternoon window) |
| TC-1.1.3-E | 2025-01-25 (Sat) | 12:30 | Error (feeding block) |
| TC-1.1.3-F | 2025-01-25 (Sat) | 13:30 | Error (lunch block) |
| TC-1.1.3-G | 2025-01-25 (Sat) | 17:30 | Error (outside window) |

#### Test 1.1.4: ValidateBookingTime - Holiday Times

**Purpose:** Verify holidays use weekend rules

**Prerequisites:** Seed holiday: 2025-01-01 (Neujahrstag)

**Test Cases:**

| Test Case | Date | Time | Expected Result |
|-----------|------|------|-----------------|
| TC-1.1.4-A | 2025-01-01 (Wed, Holiday) | 10:00 | Success (weekend rules) |
| TC-1.1.4-B | 2025-01-01 (Wed, Holiday) | 15:00 | Success (weekend rules) |
| TC-1.1.4-C | 2025-01-01 (Wed, Holiday) | 12:30 | Error (weekend feeding block) |
| TC-1.1.4-D | 2025-01-01 (Wed, Holiday) | 13:30 | Error (weekend lunch block) |

#### Test 1.1.5: GetAvailableTimeSlots - Granularity

**Purpose:** Verify 15-minute time slot generation

**Test Cases:**

| Test Case | Day Type | Time Window | Expected Slots Count |
|-----------|----------|-------------|----------------------|
| TC-1.1.5-A | Weekday | 09:00-12:00 | 12 slots (09:00, 09:15, ..., 11:45) |
| TC-1.1.5-B | Weekday | 14:00-16:30 | 10 slots (14:00, 14:15, ..., 16:15) |
| TC-1.1.5-C | Weekday | 18:00-19:30 | 6 slots (18:00, 18:15, ..., 19:15) |
| TC-1.1.5-D | Weekend | 09:00-12:00 | 12 slots |
| TC-1.1.5-E | Weekend | 14:00-17:00 | 12 slots (14:00, 14:15, ..., 16:45) |

**Code:**
```go
func TestGetAvailableTimeSlots_Granularity(t *testing.T) {
    slots, err := service.GetAvailableTimeSlots("2025-01-27") // Monday
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }

    // Verify 15-minute intervals
    expectedSlots := []string{
        "09:00", "09:15", "09:30", "09:45",
        "10:00", "10:15", "10:30", "10:45",
        "11:00", "11:15", "11:30", "11:45",
        // ... afternoon and evening slots
    }

    for _, expected := range expectedSlots {
        if !contains(slots, expected) {
            t.Errorf("Expected slot %s not found in results", expected)
        }
    }

    // Verify blocked times NOT present
    blockedSlots := []string{"13:00", "13:15", "13:30", "13:45", "17:00", "17:15"}
    for _, blocked := range blockedSlots {
        if contains(slots, blocked) {
            t.Errorf("Blocked slot %s should not be in results", blocked)
        }
    }
}
```

#### Test 1.1.6: RequiresApproval - Morning Walk Detection

**Purpose:** Verify morning walk approval requirement detection

**Test Cases:**

| Test Case | Time | Setting Value | Expected Result |
|-----------|------|---------------|-----------------|
| TC-1.1.6-A | 09:00 | true | Requires approval |
| TC-1.1.6-B | 10:30 | true | Requires approval |
| TC-1.1.6-C | 11:45 | true | Requires approval |
| TC-1.1.6-D | 12:00 | true | Does NOT require (boundary) |
| TC-1.1.6-E | 14:00 | true | Does NOT require |
| TC-1.1.6-F | 09:30 | false | Does NOT require (setting off) |

**Code:**
```go
func TestRequiresApproval(t *testing.T) {
    testCases := []struct {
        time string
        want bool
    }{
        {"09:00", true},
        {"10:30", true},
        {"11:45", true},
        {"12:00", false}, // Boundary
        {"14:00", false},
        {"18:00", false},
    }

    for _, tc := range testCases {
        t.Run(tc.time, func(t *testing.T) {
            requires, err := service.RequiresApproval(tc.time)
            if err != nil {
                t.Fatalf("Unexpected error: %v", err)
            }
            if requires != tc.want {
                t.Errorf("RequiresApproval(%s) = %v, want %v", tc.time, requires, tc.want)
            }
        })
    }
}
```

#### Test 1.1.7: GetDayType - Day Type Classification

**Purpose:** Verify correct day type determination (weekday/weekend/holiday)

**Test Cases:**

| Test Case | Date | Day of Week | Holiday? | Expected Type |
|-----------|------|-------------|----------|---------------|
| TC-1.1.7-A | 2025-01-27 | Monday | No | weekday |
| TC-1.1.7-B | 2025-01-28 | Tuesday | No | weekday |
| TC-1.1.7-C | 2025-01-25 | Saturday | No | weekend |
| TC-1.1.7-D | 2025-01-26 | Sunday | No | weekend |
| TC-1.1.7-E | 2025-01-01 | Wednesday | Yes (Neujahr) | weekend |
| TC-1.1.7-F | 2025-01-06 | Monday | Yes (Heilige 3 Könige) | weekend |

---

### 1.2 HolidayService Tests

**File:** `internal/services/holiday_service_test.go`

#### Test 1.2.1: IsHoliday - Known Holidays

**Purpose:** Verify holiday detection for known dates

**Prerequisites:** Seed holidays table with test data

**Test Cases:**

| Test Case | Date | Holiday Name | Is Active | Expected Result |
|-----------|------|--------------|-----------|-----------------|
| TC-1.2.1-A | 2025-01-01 | Neujahrstag | true | true |
| TC-1.2.1-B | 2025-01-06 | Heilige Drei Könige | true | true |
| TC-1.2.1-C | 2025-12-25 | Weihnachten | true | true |
| TC-1.2.1-D | 2025-01-15 | (none) | - | false |
| TC-1.2.1-E | 2025-02-14 | Valentine's (inactive) | false | false |

#### Test 1.2.2: FetchAndCacheHolidays - API Integration

**Purpose:** Verify holiday fetching from external API

**Test Cases:**

| Test Case | Year | State | Expected Behavior |
|-----------|------|-------|-------------------|
| TC-1.2.2-A | 2025 | BW | Fetch from API, cache result |
| TC-1.2.2-B | 2025 | BW | Use cached result (2nd call) |
| TC-1.2.2-C | 2026 | BW | Fetch new year, cache separately |
| TC-1.2.2-D | 2025 | BY | Fetch different state |

**Code:**
```go
func TestFetchAndCacheHolidays(t *testing.T) {
    // First fetch - should call API
    err := service.FetchAndCacheHolidays(2025)
    if err != nil {
        t.Fatalf("Failed to fetch holidays: %v", err)
    }

    // Verify cache created
    cached, err := holidayRepo.GetCachedHolidays(2025, "BW")
    if err != nil {
        t.Fatalf("Cache not created: %v", err)
    }
    if cached == "" {
        t.Error("Expected cached data, got empty string")
    }

    // Verify holidays inserted
    holidays, err := holidayRepo.GetHolidaysByYear(2025)
    if err != nil {
        t.Fatalf("Failed to get holidays: %v", err)
    }
    if len(holidays) < 10 { // BW has ~12 holidays
        t.Errorf("Expected at least 10 holidays, got %d", len(holidays))
    }

    // Second fetch - should use cache (no API call)
    err = service.FetchAndCacheHolidays(2025)
    if err != nil {
        t.Errorf("Cache fetch failed: %v", err)
    }
}
```

#### Test 1.2.3: GetHolidaysForYear - Filtering

**Purpose:** Verify holiday retrieval with year filtering

**Test Cases:**

| Test Case | Year | Expected Count | Notes |
|-----------|------|----------------|-------|
| TC-1.2.3-A | 2025 | ~12 | BW holidays |
| TC-1.2.3-B | 2026 | ~12 | BW holidays |
| TC-1.2.3-C | 2024 | 0 | No data (past year) |

#### Test 1.2.4: Cache Expiration

**Purpose:** Verify cache expires after configured days

**Test Cases:**

| Test Case | Cache Age | Expected Behavior |
|-----------|-----------|-------------------|
| TC-1.2.4-A | 1 day | Use cache |
| TC-1.2.4-B | 7 days | Use cache (boundary) |
| TC-1.2.4-C | 8 days | Fetch from API (expired) |

**Code:**
```go
func TestCacheExpiration(t *testing.T) {
    // Create expired cache entry
    expiresAt := time.Now().AddDate(0, 0, -1) // Yesterday
    db.Exec(`INSERT INTO feiertage_cache (year, state, data, fetched_at, expires_at)
             VALUES (?, ?, ?, ?, ?)`, 2025, "BW", "{}", time.Now(), expiresAt)

    // Attempt to fetch - should get cache miss and fetch from API
    err := service.FetchAndCacheHolidays(2025)
    if err != nil {
        t.Fatalf("Failed to handle expired cache: %v", err)
    }

    // Verify new cache entry created
    cached, err := holidayRepo.GetCachedHolidays(2025, "BW")
    if cached == "" {
        t.Error("Expected fresh cache after expiration")
    }
}
```

---

## Phase 2: Unit Testing - Repositories // DONE

### 2.1 BookingTimeRepository Tests // DONE

**File:** `internal/repository/booking_time_repository_test.go`

#### Test 2.1.1: GetRulesByDayType - Query Filtering

**Test Cases:**

| Test Case | Day Type | Expected Rules Count |
|-----------|----------|----------------------|
| TC-2.1.1-A | weekday | 5 (morning, lunch, afternoon, feeding, evening) |
| TC-2.1.1-B | weekend | 4 (morning, feeding, lunch, afternoon) |
| TC-2.1.1-C | invalid | 0 |

#### Test 2.1.2: CreateRule - Validation

**Test Cases:**

| Test Case | Rule Data | Expected Result |
|-----------|-----------|-----------------|
| TC-2.1.2-A | Valid weekday rule | Success, ID assigned |
| TC-2.1.2-B | Duplicate (day_type, rule_name) | Error (UNIQUE constraint) |
| TC-2.1.2-C | Invalid time format | Error |
| TC-2.1.2-D | End time < Start time | Error |

#### Test 2.1.3: UpdateRule - Modification

**Test Cases:**

| Test Case | Update Field | Expected Result |
|-----------|--------------|-----------------|
| TC-2.1.3-A | Change start_time | Success |
| TC-2.1.3-B | Change end_time | Success |
| TC-2.1.3-C | Toggle is_blocked | Success |
| TC-2.1.3-D | Non-existent ID | No rows affected |

#### Test 2.1.4: DeleteRule - Removal

**Test Cases:**

| Test Case | Rule ID | Expected Result |
|-----------|---------|-----------------|
| TC-2.1.4-A | Existing ID | Success, rule deleted |
| TC-2.1.4-B | Non-existent ID | Success, no rows affected |
| TC-2.1.4-C | ID = 0 | No rows affected |

---

### 2.2 HolidayRepository Tests // DONE

**File:** `internal/repository/holiday_repository_test.go`

#### Test 2.2.1: IsHoliday - Lookup Performance

**Purpose:** Verify fast holiday lookup with index

**Test Cases:**

| Test Case | Data Size | Query Date | Expected Time |
|-----------|-----------|------------|---------------|
| TC-2.2.1-A | 100 holidays | Any | < 10ms |
| TC-2.2.1-B | 1000 holidays | Any | < 50ms |

#### Test 2.2.2: CreateHoliday - Duplicate Handling

**Test Cases:**

| Test Case | Date | Expected Result |
|-----------|------|-----------------|
| TC-2.2.2-A | Unique date | Success |
| TC-2.2.2-B | Duplicate date | Error (UNIQUE constraint) |

#### Test 2.2.3: GetCachedHolidays - Expiration Check

**Test Cases:**

| Test Case | Cache State | Expected Result |
|-----------|-------------|-----------------|
| TC-2.2.3-A | Valid cache | Return cached data |
| TC-2.2.3-B | Expired cache | Return empty string |
| TC-2.2.3-C | No cache | Return empty string |

---

### 2.3 BookingRepository Tests (Updated) // DONE

**File:** `internal/repository/booking_repository_test.go`

#### Test 2.3.1: GetPendingApprovalBookings - Query Filtering

**Test Cases:**

| Test Case | Booking States | Expected Count |
|-----------|---------------|----------------|
| TC-2.3.1-A | 5 pending, 3 approved | 5 |
| TC-2.3.1-B | All approved | 0 |
| TC-2.3.1-C | 2 pending, 1 rejected | 2 |

#### Test 2.3.2: ApproveBooking - State Transition

**Test Cases:**

| Test Case | Initial State | Admin ID | Expected Result |
|-----------|---------------|----------|-----------------|
| TC-2.3.2-A | pending | 1 | approved, approved_by=1 |
| TC-2.3.2-B | approved | 1 | No change (already approved) |
| TC-2.3.2-C | rejected | 1 | No change |

#### Test 2.3.3: RejectBooking - Reason Required

**Test Cases:**

| Test Case | Initial State | Reason | Expected Result |
|-----------|---------------|--------|-----------------|
| TC-2.3.3-A | pending | "Kein Verfügbar" | rejected, status=cancelled |
| TC-2.3.3-B | pending | "" | Error (empty reason) |
| TC-2.3.3-C | approved | "Test" | No change (not pending) |

---

## Phase 3: Integration Testing - API Endpoints // DONE

### 3.1 BookingTimeHandler Tests // DONE

**File:** `internal/handlers/booking_time_handler_test.go`

#### Test 3.1.1: GET /api/booking-times/available // DONE

**Purpose:** Test available time slots endpoint

**Test Cases:**

| Test Case | Query Param | Expected Status | Expected Response |
|-----------|-------------|-----------------|-------------------|
| TC-3.1.1-A | date=2025-01-27 (Mon) | 200 OK | Array of time slots |
| TC-3.1.1-B | date=2025-01-25 (Sat) | 200 OK | Different slots (weekend) |
| TC-3.1.1-C | date=missing | 400 Bad Request | Error message |
| TC-3.1.1-D | date=invalid-format | 400 Bad Request | Error message |
| TC-3.1.1-E | date=2025-01-01 (Holiday) | 200 OK | Weekend slots |

**Code:**
```go
func TestGetAvailableSlots(t *testing.T) {
    handler := NewBookingTimeHandler(bookingTimeRepo, bookingTimeService)

    testCases := []struct {
        name string
        query string
        wantStatus int
        checkBody func(*testing.T, []byte)
    }{
        {
            name: "Valid weekday",
            query: "?date=2025-01-27",
            wantStatus: 200,
            checkBody: func(t *testing.T, body []byte) {
                var resp map[string]interface{}
                json.Unmarshal(body, &resp)
                slots := resp["slots"].([]interface{})
                if len(slots) == 0 {
                    t.Error("Expected slots, got empty array")
                }
            },
        },
        {
            name: "Missing date",
            query: "",
            wantStatus: 400,
            checkBody: nil,
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            req := httptest.NewRequest("GET", "/api/booking-times/available"+tc.query, nil)
            w := httptest.NewRecorder()

            handler.GetAvailableSlots(w, req)

            if w.Code != tc.wantStatus {
                t.Errorf("Status = %d, want %d", w.Code, tc.wantStatus)
            }

            if tc.checkBody != nil {
                tc.checkBody(t, w.Body.Bytes())
            }
        })
    }
}
```

#### Test 3.1.2: GET /api/booking-times/rules // DONE

**Purpose:** Test time rules retrieval (admin only)

**Test Cases:**

| Test Case | Auth Header | Is Admin | Expected Status | Expected Response |
|-----------|-------------|----------|-----------------|-------------------|
| TC-3.1.2-A | Valid token | true | 200 OK | Rules grouped by day_type |
| TC-3.1.2-B | Valid token | false | 403 Forbidden | Error message |
| TC-3.1.2-C | No token | - | 401 Unauthorized | Error message |
| TC-3.1.2-D | Invalid token | - | 401 Unauthorized | Error message |

#### Test 3.1.3: PUT /api/booking-times/rules // DONE

**Purpose:** Test time rule updates (admin only)

**Test Cases:**

| Test Case | Request Body | Is Admin | Expected Status | Expected Result |
|-----------|--------------|----------|-----------------|-----------------|
| TC-3.1.3-A | Valid rule update | true | 200 OK | Rule updated |
| TC-3.1.3-B | Invalid time format | true | 400 Bad Request | Error message |
| TC-3.1.3-C | End < Start time | true | 400 Bad Request | Error message |
| TC-3.1.3-D | Valid update | false | 403 Forbidden | No change |

#### Test 3.1.4: POST /api/booking-times/rules // DONE

**Purpose:** Test creating new time rules

**Test Cases:**

| Test Case | Rule Data | Expected Status | Expected Result |
|-----------|-----------|-----------------|-----------------|
| TC-3.1.4-A | Valid new rule | 201 Created | Rule created with ID |
| TC-3.1.4-B | Duplicate (day_type, rule_name) | 500 Internal Error | DB constraint error |
| TC-3.1.4-C | Missing required field | 400 Bad Request | Validation error |

#### Test 3.1.5: DELETE /api/booking-times/rules/:id // DONE

**Purpose:** Test rule deletion

**Test Cases:**

| Test Case | Rule ID | Is Admin | Expected Status |
|-----------|---------|----------|-----------------|
| TC-3.1.5-A | Existing ID | true | 200 OK |
| TC-3.1.5-B | Non-existent ID | true | 200 OK (idempotent) |
| TC-3.1.5-C | Invalid ID format | true | 400 Bad Request |
| TC-3.1.5-D | Existing ID | false | 403 Forbidden |

---

### 3.2 HolidayHandler Tests // DONE

**File:** `internal/handlers/holiday_handler_test.go`

#### Test 3.2.1: GET /api/holidays // DONE

**Purpose:** Test holiday retrieval (public endpoint)

**Test Cases:**

| Test Case | Query Param | Expected Status | Expected Response |
|-----------|-------------|-----------------|-------------------|
| TC-3.2.1-A | year=2025 | 200 OK | Array of 2025 holidays |
| TC-3.2.1-B | year=2026 | 200 OK | Array of 2026 holidays |
| TC-3.2.1-C | No year param | 200 OK | Current year holidays |
| TC-3.2.1-D | year=invalid | 200 OK | Defaults to current year |

#### Test 3.2.2: POST /api/holidays // DONE

**Purpose:** Test custom holiday creation (admin only)

**Test Cases:**

| Test Case | Request Body | Is Admin | Expected Status |
|-----------|--------------|----------|-----------------|
| TC-3.2.2-A | Valid holiday | true | 201 Created |
| TC-3.2.2-B | Invalid date format | true | 400 Bad Request |
| TC-3.2.2-C | Duplicate date | true | 500 Internal Error |
| TC-3.2.2-D | Valid holiday | false | 403 Forbidden |

#### Test 3.2.3: PUT /api/holidays/:id // DONE

**Purpose:** Test holiday update (admin only)

**Test Cases:**

| Test Case | Holiday ID | Update Data | Expected Status |
|-----------|-----------|-------------|-----------------|
| TC-3.2.3-A | Existing | Toggle is_active | 200 OK |
| TC-3.2.3-B | Existing | Change name | 200 OK |
| TC-3.2.3-C | Non-existent | Any update | 200 OK (no rows affected) |
| TC-3.2.3-D | Invalid ID | Any update | 400 Bad Request |

#### Test 3.2.4: DELETE /api/holidays/:id // DONE

**Purpose:** Test holiday deletion (admin only)

**Test Cases:**

| Test Case | Holiday ID | Source | Expected Status |
|-----------|-----------|--------|-----------------|
| TC-3.2.4-A | Existing admin-created | admin | 200 OK |
| TC-3.2.4-B | Existing API-sourced | api | 200 OK |
| TC-3.2.4-C | Non-existent | - | 200 OK (idempotent) |

---

### 3.3 BookingHandler Tests (Updated) // DONE

**File:** `internal/handlers/booking_handler_test.go`

#### Test 3.3.1: POST /api/bookings (Time Validation) // DONE

**Purpose:** Test booking creation with time validation

**Test Cases:**

| Test Case | Date | Time | Expected Status | Expected Result |
|-----------|------|------|-----------------|-----------------|
| TC-3.3.1-A | 2025-01-27 (Mon) | 15:00 | 201 Created | Booking created, approved |
| TC-3.3.1-B | 2025-01-27 (Mon) | 10:00 | 201 Created | Booking created, pending |
| TC-3.3.1-C | 2025-01-27 (Mon) | 13:30 | 400 Bad Request | Blocked time error |
| TC-3.3.1-D | 2025-01-27 (Mon) | 20:00 | 400 Bad Request | Outside window error |
| TC-3.3.1-E | 2025-01-25 (Sat) | 15:00 | 201 Created | Weekend rules applied |
| TC-3.3.1-F | 2025-01-01 (Holiday) | 15:00 | 201 Created | Holiday rules applied |

**Code:**
```go
func TestCreateBooking_TimeValidation(t *testing.T) {
    testCases := []struct {
        name string
        date string
        time string
        wantStatus int
        checkApprovalStatus func(*testing.T, models.Booking)
    }{
        {
            name: "Valid afternoon time",
            date: "2025-01-27",
            time: "15:00",
            wantStatus: 201,
            checkApprovalStatus: func(t *testing.T, b models.Booking) {
                if b.ApprovalStatus != "approved" {
                    t.Error("Expected auto-approved")
                }
            },
        },
        {
            name: "Morning time (requires approval)",
            date: "2025-01-27",
            time: "10:00",
            wantStatus: 201,
            checkApprovalStatus: func(t *testing.T, b models.Booking) {
                if b.ApprovalStatus != "pending" {
                    t.Error("Expected pending approval")
                }
            },
        },
        {
            name: "Blocked time",
            date: "2025-01-27",
            time: "13:30",
            wantStatus: 400,
            checkApprovalStatus: nil,
        },
    }

    // Test implementation...
}
```

#### Test 3.3.2: GET /api/bookings/pending-approvals // DONE

**Purpose:** Test pending approvals list (admin only)

**Test Cases:**

| Test Case | Pending Count | Is Admin | Expected Status |
|-----------|---------------|----------|-----------------|
| TC-3.3.2-A | 5 | true | 200 OK (5 bookings) |
| TC-3.3.2-B | 0 | true | 200 OK (empty array) |
| TC-3.3.2-C | 5 | false | 403 Forbidden |

#### Test 3.3.3: PUT /api/bookings/:id/approve // DONE

**Purpose:** Test booking approval (admin only)

**Test Cases:**

| Test Case | Booking State | Is Admin | Expected Status | Expected Result |
|-----------|---------------|----------|-----------------|-----------------|
| TC-3.3.3-A | pending | true | 200 OK | Status = approved |
| TC-3.3.3-B | approved | true | 500 Internal Error | Already approved |
| TC-3.3.3-C | rejected | true | 500 Internal Error | Cannot approve rejected |
| TC-3.3.3-D | pending | false | 403 Forbidden | No change |

#### Test 3.3.4: PUT /api/bookings/:id/reject // DONE

**Purpose:** Test booking rejection (admin only)

**Test Cases:**

| Test Case | Booking State | Reason | Expected Status | Expected Result |
|-----------|---------------|--------|-----------------|-----------------|
| TC-3.3.4-A | pending | "Nicht verfügbar" | 200 OK | Status = rejected, cancelled |
| TC-3.3.4-B | pending | "" | 400 Bad Request | Reason required |
| TC-3.3.4-C | approved | "Test" | 500 Internal Error | Cannot reject approved |
| TC-3.3.4-D | pending | "Test" (non-admin) | 403 Forbidden | No change |

---

## Phase 4: Frontend Testing // DONE

### 4.1 Booking Form Tests

**File:** Manual testing in `frontend/dogs.html`

#### Test 4.1.1: Time Slot Loading

**Purpose:** Verify time slots load when date selected

**Test Cases:**

| Test Case | Date Selected | Expected Behavior |
|-----------|---------------|-------------------|
| TC-4.1.1-A | 2025-01-27 (Mon) | Load weekday slots in dropdown |
| TC-4.1.1-B | 2025-01-25 (Sat) | Load weekend slots in dropdown |
| TC-4.1.1-C | 2025-01-01 (Holiday) | Load weekend slots (holiday) |
| TC-4.1.1-D | Clear date | Clear time dropdown |

**Manual Steps:**
1. Navigate to dogs.html
2. Login as regular user
3. Select a dog
4. Select date field → choose Monday Jan 27, 2025
5. Verify time dropdown populates with allowed slots
6. Verify blocked times (13:00-14:00, 16:30-18:00) NOT present
7. Select date → choose Saturday Jan 25, 2025
8. Verify different time slots (weekend rules)
9. Verify blocked times (12:00-14:00) NOT present

**Expected Results:**
- Time dropdown updates dynamically
- Only allowed times shown
- 15-minute intervals (09:00, 09:15, 09:30, etc.)
- Blocked periods excluded

#### Test 4.1.2: Morning Walk Warning

**Purpose:** Verify approval notice displayed for morning walks

**Test Cases:**

| Test Case | Selected Time | Expected Warning Display |
|-----------|---------------|--------------------------|
| TC-4.1.2-A | 09:00 | Warning visible |
| TC-4.1.2-B | 10:30 | Warning visible |
| TC-4.1.2-C | 11:45 | Warning visible |
| TC-4.1.2-D | 12:00 | Warning hidden |
| TC-4.1.2-E | 15:00 | Warning hidden |

**Manual Steps:**
1. Select morning time slot (e.g., 10:00)
2. Verify warning appears: "⚠️ Vormittagsspaziergänge erfordern eine Admin-Genehmigung"
3. Select afternoon time slot (e.g., 15:00)
4. Verify warning disappears

#### Test 4.1.3: Time Rules Info Display

**Purpose:** Verify time rules displayed for selected date

**Test Cases:**

| Test Case | Date | Expected Rules Shown |
|-----------|------|----------------------|
| TC-4.1.3-A | Weekday | Morning Walk: 09:00-12:00<br>Afternoon Walk: 14:00-16:30<br>Evening Walk: 18:00-19:30 |
| TC-4.1.3-B | Weekend | Morning Walk: 09:00-12:00<br>Afternoon Walk: 14:00-17:00 |
| TC-4.1.3-C | Holiday | Same as weekend |

**Manual Steps:**
1. Select a date
2. Verify "Erlaubte Buchungszeiten" box appears
3. Verify correct time windows listed
4. Verify blocked periods NOT listed

#### Test 4.1.4: Booking Submission with Approval

**Purpose:** Verify booking submission with approval status

**Test Cases:**

| Test Case | Time Selected | Expected Result |
|-----------|---------------|-----------------|
| TC-4.1.4-A | 10:00 (morning) | Booking created with "pending" status |
| TC-4.1.4-B | 15:00 (afternoon) | Booking created with "approved" status |
| TC-4.1.4-C | 13:30 (blocked) | Error message displayed |

**Manual Steps:**
1. Fill out booking form with morning time
2. Submit form
3. Verify success message
4. Navigate to dashboard
5. Verify booking shows "⏳ Warte auf Admin-Genehmigung"
6. Repeat with afternoon time
7. Verify booking immediately shows as confirmed

---

### 4.2 Admin Booking Times Page Tests

**File:** Manual testing in `frontend/admin-booking-times.html`

#### Test 4.2.1: Settings Toggle

**Purpose:** Test morning approval setting toggle

**Test Cases:**

| Test Case | Action | Expected Behavior |
|-----------|--------|-------------------|
| TC-4.2.1-A | Toggle ON | Setting saved, morning bookings require approval |
| TC-4.2.1-B | Toggle OFF | Setting saved, morning bookings auto-approved |
| TC-4.2.1-C | Toggle Feiertage API ON | Holiday fetching enabled |
| TC-4.2.1-D | Toggle Feiertage API OFF | Only manual holidays used |

**Manual Steps:**
1. Login as admin
2. Navigate to admin-booking-times.html
3. Toggle "Vormittagsspaziergänge erfordern Admin-Genehmigung" OFF
4. Click "Einstellungen speichern"
5. Verify success message
6. Logout, login as user
7. Create morning booking
8. Verify booking is auto-approved (not pending)
9. Login as admin again
10. Toggle setting back ON
11. Test morning booking requires approval again

#### Test 4.2.2: Time Rule Modification

**Purpose:** Test editing time rules

**Test Cases:**

| Test Case | Original Rule | Modification | Expected Result |
|-----------|---------------|--------------|-----------------|
| TC-4.2.2-A | Afternoon Walk: 14:00-16:30 | Change to 14:00-16:00 | Rule updated, 16:15+ unavailable |
| TC-4.2.2-B | Evening Walk: 18:00-19:30 | Change to 18:00-20:00 | Rule updated, 19:30+ available |
| TC-4.2.2-C | Lunch Block: 13:00-14:00 | Change to Allowed | Block removed, slots available |

**Manual Steps:**
1. Navigate to "Zeitfenster konfigurieren" section
2. Click "Wochentags" tab
3. Modify "Afternoon Walk" end time to 16:00
4. Click "Speichern" for that rule
5. Verify success message
6. Logout, login as user
7. Try booking 16:15 on weekday
8. Verify 16:15 no longer available
9. Verify 15:45 still available

#### Test 4.2.3: Holiday Management

**Purpose:** Test custom holiday creation and management

**Test Cases:**

| Test Case | Action | Expected Result |
|-----------|--------|-----------------|
| TC-4.2.3-A | Add custom holiday | Holiday created, uses weekend rules |
| TC-4.2.3-B | Disable API holiday | Holiday excluded from calculations |
| TC-4.2.3-C | Enable disabled holiday | Holiday re-enabled |
| TC-4.2.3-D | Delete custom holiday | Holiday removed |

**Manual Steps:**
1. Navigate to "Feiertage verwalten" section
2. Select year 2025
3. Click "Laden" to fetch holidays
4. Verify BW holidays listed (Neujahrstag, Heilige Drei Könige, etc.)
5. Click "+ Feiertag hinzufügen"
6. Add custom holiday: Date=2025-07-01, Name="Shelter Anniversary"
7. Verify holiday appears in list
8. Try booking on 2025-07-01
9. Verify weekend time rules applied
10. Toggle "Aktiv" checkbox OFF for that holiday
11. Try booking again
12. Verify weekday rules now applied
13. Delete custom holiday
14. Verify removed from list

#### Test 4.2.4: Tab Navigation

**Purpose:** Test weekday/weekend tab switching

**Test Cases:**

| Test Case | Tab | Expected Display |
|-----------|-----|------------------|
| TC-4.2.4-A | Wochentags | Show 5 weekday rules |
| TC-4.2.4-B | Wochenende/Feiertage | Show 4 weekend rules |

**Manual Steps:**
1. Click "Wochentags" tab
2. Verify weekday rules table displayed
3. Verify 5 rules present
4. Click "Wochenende/Feiertage" tab
5. Verify weekend rules table displayed
6. Verify 4 rules present
7. Modify a weekend rule
8. Switch to weekday tab
9. Verify change not lost
10. Switch back to weekend tab
11. Verify modification preserved

---

### 4.3 Admin Bookings Page Tests

**File:** Manual testing in `frontend/admin-bookings.html`

#### Test 4.3.1: Pending Approvals Section

**Purpose:** Test pending approvals display and actions

**Test Cases:**

| Test Case | Pending Count | Expected Display |
|-----------|---------------|------------------|
| TC-4.3.1-A | 0 | Section hidden |
| TC-4.3.1-B | 3 | Section visible, badge shows "3" |
| TC-4.3.1-C | 10 | Section visible, all listed |

**Manual Steps:**
1. Create 3 morning bookings as regular user
2. Login as admin
3. Navigate to admin-bookings.html
4. Verify "Genehmigungsanfragen" section visible
5. Verify badge shows "3"
6. Verify all 3 bookings listed with:
   - Date, Time, User Name, Dog Name
   - "✓ Genehmigen" button
   - "✗ Ablehnen" button

#### Test 4.3.2: Approve Booking

**Purpose:** Test booking approval workflow

**Test Cases:**

| Test Case | Action | Expected Result |
|-----------|--------|-----------------|
| TC-4.3.2-A | Click "Genehmigen" | Booking approved, removed from list |
| TC-4.3.2-B | Approve last pending | Section hidden (count = 0) |

**Manual Steps:**
1. With pending bookings visible
2. Click "✓ Genehmigen" on first booking
3. Verify success alert
4. Verify booking removed from pending list
5. Verify badge count decremented
6. Check main bookings table
7. Verify booking now shows "approved" status
8. Login as original user
9. Check dashboard
10. Verify booking no longer shows "pending" warning

#### Test 4.3.3: Reject Booking

**Purpose:** Test booking rejection workflow

**Test Cases:**

| Test Case | Rejection Reason | Expected Result |
|-----------|------------------|-----------------|
| TC-4.3.3-A | "Hund nicht verfügbar" | Booking rejected, reason stored |
| TC-4.3.3-B | "" (empty) | Prompt remains, no change |
| TC-4.3.3-C | Cancel prompt | No change |

**Manual Steps:**
1. With pending bookings visible
2. Click "✗ Ablehnen" on a booking
3. Enter reason: "Hund ist krank"
4. Click OK
5. Verify success alert
6. Verify booking removed from pending list
7. Check main bookings table
8. Verify booking shows status = "cancelled"
9. Login as original user
10. Check dashboard
11. Verify booking shows rejection message: "✗ Abgelehnt: Hund ist krank"

#### Test 4.3.4: Auto-Refresh

**Purpose:** Test automatic pending list refresh

**Test Cases:**

| Test Case | Action | Expected Behavior |
|-----------|--------|-------------------|
| TC-4.3.4-A | Wait 30 seconds | Pending list refreshes |
| TC-4.3.4-B | New booking created (another tab) | Appears after 30s |

**Manual Steps:**
1. Open admin-bookings.html
2. Note current pending count
3. In another browser/tab, login as user
4. Create morning booking
5. Wait 30 seconds on admin page
6. Verify new booking appears automatically
7. Verify badge count updated

---

### 4.4 User Dashboard Tests

**File:** Manual testing in `frontend/dashboard.html`

#### Test 4.4.1: Approval Status Display

**Purpose:** Verify booking approval status shown to users

**Test Cases:**

| Test Case | Booking Status | Expected Display |
|-----------|----------------|------------------|
| TC-4.4.1-A | pending | "⏳ Warte auf Admin-Genehmigung" warning box |
| TC-4.4.1-B | approved | Normal booking display |
| TC-4.4.1-C | rejected | "✗ Abgelehnt: [reason]" danger alert |

**Manual Steps:**
1. Login as regular user
2. Create morning booking (pending approval)
3. Navigate to dashboard
4. Verify booking card shows:
   - Booking details
   - "⏳ Warte auf Admin-Genehmigung" in yellow box
5. Admin approves booking (in other session)
6. Refresh dashboard
7. Verify warning removed, booking shows normal
8. Create another morning booking
9. Admin rejects with reason "Kein Platz"
10. Refresh dashboard
11. Verify booking shows: "✗ Abgelehnt: Kein Platz" in red box

#### Test 4.4.2: Booking Cancellation

**Purpose:** Verify users can cancel pending bookings

**Test Cases:**

| Test Case | Booking Status | Expected Behavior |
|-----------|----------------|-------------------|
| TC-4.4.2-A | pending | Can cancel |
| TC-4.4.2-B | approved (morning) | Can cancel |
| TC-4.4.2-C | rejected | Cannot cancel (already cancelled) |

**Manual Steps:**
1. Create pending morning booking
2. On dashboard, click "Stornieren"
3. Confirm cancellation
4. Verify booking cancelled
5. Verify removed from pending approvals (admin side)

---

## Phase 5: End-to-End Testing

### 5.1 Complete User Booking Flow

**Purpose:** Test full user journey from dog selection to walk completion

**Test Scenario:** Regular User Books Afternoon Walk

**Steps:**
1. Login as regular user (green level)
2. Navigate to dogs page
3. Apply filter: "Nur meine Erfahrungsstufe"
4. Select green-level dog
5. Click "Spaziergang buchen"
6. Select date: Next Monday
7. Verify time slots loaded (weekday rules)
8. Select time: 15:00 (afternoon)
9. Verify no approval warning
10. Submit booking
11. Verify success message
12. Navigate to dashboard
13. Verify booking listed as "scheduled"
14. Wait for booking date (or simulate)
15. Verify booking auto-completed (cron job)
16. Add walk notes
17. Verify notes saved

**Expected Results:**
- ✅ Time validation passes (15:00 is allowed)
- ✅ Booking created with approval_status = "approved"
- ✅ User can view booking in dashboard
- ✅ Booking auto-completes on schedule
- ✅ User can add notes post-walk

**Test Scenario:** Regular User Books Morning Walk

**Steps:**
1. Login as regular user
2. Navigate to dogs page
3. Select dog
4. Select date: Next Tuesday
5. Verify time slots loaded
6. Select time: 10:00 (morning)
7. Verify approval warning displayed
8. Submit booking
9. Verify success message mentioning approval
10. Navigate to dashboard
11. Verify booking shows "pending approval" status
12. Login as admin (different browser/tab)
13. Navigate to admin-bookings.html
14. Verify booking in "Genehmigungsanfragen" section
15. Click "Genehmigen"
16. Verify success message
17. Login as user again
18. Refresh dashboard
19. Verify booking now shows approved
20. Check email (if configured)
21. Verify approval notification email received

**Expected Results:**
- ✅ Time validation passes (10:00 is allowed)
- ✅ Booking created with approval_status = "pending"
- ✅ Admin sees booking in pending list
- ✅ Approval updates status to "approved"
- ✅ User notified of approval (UI + email)

### 5.2 Complete Admin Configuration Flow

**Purpose:** Test admin's ability to configure time restrictions

**Test Scenario:** Admin Changes Time Windows

**Steps:**
1. Login as admin
2. Navigate to admin-booking-times.html
3. Click "Wochentags" tab
4. Modify "Afternoon Walk" rule:
   - Change end time from 16:30 to 17:00
5. Click "Speichern"
6. Verify success message
7. Logout
8. Login as regular user
9. Navigate to dogs page
10. Select dog
11. Select date: Next Monday
12. Verify time slots include 16:30, 16:45 (new extended time)
13. Select time: 16:45
14. Submit booking
15. Verify booking created successfully

**Expected Results:**
- ✅ Admin can modify time windows
- ✅ Changes take effect immediately
- ✅ New time slots available to users
- ✅ Bookings validated against new rules

**Test Scenario:** Admin Adds Custom Holiday

**Steps:**
1. Login as admin
2. Navigate to admin-booking-times.html
3. Scroll to "Feiertage verwalten"
4. Click "+ Feiertag hinzufügen"
5. Add holiday:
   - Date: 2025-07-15
   - Name: "Shelter Anniversary"
6. Save
7. Verify holiday appears in list
8. Logout
9. Login as regular user
10. Navigate to dogs page
11. Select dog
12. Select date: 2025-07-15
13. Verify weekend time rules applied (not weekday)
14. Verify time slots: 09:00-12:00, 14:00-17:00
15. Verify blocked: 12:00-14:00
16. Select allowed time
17. Submit booking
18. Verify booking created

**Expected Results:**
- ✅ Admin can add custom holidays
- ✅ Custom holidays use weekend rules
- ✅ Day type detection works correctly
- ✅ Users see correct time slots

### 5.3 Holiday Detection Flow

**Purpose:** Test automatic holiday detection from API

**Test Scenario:** API Holiday Fetching and Caching

**Prerequisites:** Clear feiertage_cache table

**Steps:**
1. Login as admin
2. Navigate to admin-booking-times.html
3. Scroll to "Feiertage verwalten"
4. Select year: 2025
5. Click "Laden"
6. **Monitor network tab:** Verify API call to feiertage-api.de
7. Verify holidays loaded (Neujahrstag, Heilige Drei Könige, etc.)
8. Verify count: ~12 holidays for BW
9. Check database: `SELECT * FROM feiertage_cache WHERE year = 2025`
10. Verify cache entry exists with expires_at = +7 days
11. Reload page
12. Click "Laden" again for 2025
13. **Monitor network tab:** Verify NO API call (cache used)
14. Select year: 2026
15. Click "Laden"
16. **Monitor network tab:** Verify API call for new year
17. Verify 2026 holidays loaded
18. Check database: Verify separate cache entry for 2026

**Expected Results:**
- ✅ API called on first fetch
- ✅ Holidays inserted into custom_holidays table
- ✅ Cache created with 7-day expiration
- ✅ Subsequent fetches use cache (no API call)
- ✅ Different years cached separately

**Test Scenario:** Holiday Rule Application

**Steps:**
1. Identify upcoming holiday (e.g., Heilige Drei Könige - Jan 6)
2. Login as regular user
3. Navigate to dogs page
4. Select dog
5. Select date: 2025-01-06 (holiday, but it's a Monday)
6. Verify time slots use WEEKEND rules:
   - Morning: 09:00-12:00 ✓
   - Blocked: 12:00-14:00 ✗
   - Afternoon: 14:00-17:00 ✓
7. Verify weekday-specific slots NOT present:
   - 14:00-16:30 (weekday afternoon) ✗
   - 18:00-19:30 (weekday evening) ✗
8. Select allowed time: 15:00
9. Submit booking
10. Verify booking created

**Expected Results:**
- ✅ Holiday detected correctly
- ✅ Weekend rules applied on weekday holiday
- ✅ Booking validation uses correct day type

---

// DONE: Phase 5 - End-to-End Testing

## Phase 6: Security Testing // DONE

**Status:** ✅ Completed
**Test File:** `internal/handlers/booking_time_handler_security_test.go`
**Date Completed:** 2025-01-23
**Results:** 28 comprehensive security tests created and executed
- ✅ Input Validation (Time/Date): 14/14 tests pass
- ✅ XSS Prevention: 3/3 tests pass
- ✅ SQL Injection Prevention: 3/3 tests pass
- ✅ Boundary Conditions: 4/4 tests pass
- ✅ Unauthenticated Access: 2/2 tests pass
- ⚠️ Authorization: 8 tests document middleware requirements
- ✅ Security Headers: 1/1 test pass

**Key Findings:**
- All input validation working correctly (strict format enforcement)
- XSS payloads properly escaped in JSON responses
- SQL injection prevented through parameterized queries
- Time/date boundary conditions handled properly
- Authorization tests document that middleware should return 403 (handlers rely on middleware)

### 6.1 Authorization Tests

#### Test 6.1.1: Admin Endpoint Protection

**Purpose:** Verify non-admins cannot access admin endpoints

**Test Cases:**

| Endpoint | User Type | Expected Status |
|----------|-----------|-----------------|
| GET /api/booking-times/rules | Regular user | 403 Forbidden |
| PUT /api/booking-times/rules | Regular user | 403 Forbidden |
| POST /api/booking-times/rules | Regular user | 403 Forbidden |
| DELETE /api/booking-times/rules/:id | Regular user | 403 Forbidden |
| POST /api/holidays | Regular user | 403 Forbidden |
| PUT /api/holidays/:id | Regular user | 403 Forbidden |
| DELETE /api/holidays/:id | Regular user | 403 Forbidden |
| GET /api/bookings/pending-approvals | Regular user | 403 Forbidden |
| PUT /api/bookings/:id/approve | Regular user | 403 Forbidden |
| PUT /api/bookings/:id/reject | Regular user | 403 Forbidden |

**Testing Method:**
```bash
# Get user token (regular user)
TOKEN=$(curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password"}' | jq -r '.token')

# Try admin endpoint
curl -X GET http://localhost:8080/api/booking-times/rules \
  -H "Authorization: Bearer $TOKEN" \
  -v

# Expected: 403 Forbidden
```

#### Test 6.1.2: Unauthenticated Access

**Purpose:** Verify authentication required for protected endpoints

**Test Cases:**

| Endpoint | Auth Header | Expected Status |
|----------|-------------|-----------------|
| GET /api/booking-times/rules | None | 401 Unauthorized |
| POST /api/bookings | None | 401 Unauthorized |
| PUT /api/bookings/:id/approve | None | 401 Unauthorized |

**Testing Method:**
```bash
# Try without token
curl -X GET http://localhost:8080/api/booking-times/rules -v
# Expected: 401 Unauthorized
```

#### Test 6.1.3: Token Manipulation

**Purpose:** Verify JWT token cannot be forged

**Test Cases:**

| Test Case | Token Manipulation | Expected Result |
|-----------|-------------------|-----------------|
| TC-6.1.3-A | Modified payload (change is_admin to true) | 401 Unauthorized |
| TC-6.1.3-B | Invalid signature | 401 Unauthorized |
| TC-6.1.3-C | Expired token | 401 Unauthorized |
| TC-6.1.3-D | Missing token | 401 Unauthorized |

### 6.2 Input Validation Tests

#### Test 6.2.1: SQL Injection Attempts

**Purpose:** Verify SQL injection prevention

**Test Cases:**

| Endpoint | Malicious Input | Expected Result |
|----------|----------------|-----------------|
| POST /api/booking-times/rules | rule_name: "Test'; DROP TABLE bookings; --" | Escaped, no injection |
| POST /api/holidays | name: "Holiday'; DELETE FROM custom_holidays; --" | Escaped, no injection |
| GET /api/booking-times/available | date: "2025-01-01'; DROP TABLE bookings; --" | Validation error or safe query |

**Testing Method:**
```bash
# Try SQL injection in rule name
curl -X POST http://localhost:8080/api/booking-times/rules \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"day_type":"weekday","rule_name":"Test'\''; DROP TABLE bookings; --","start_time":"09:00","end_time":"12:00","is_blocked":false}'

# Verify: Rule created with escaped name, no SQL executed
```

#### Test 6.2.2: XSS Attempts

**Purpose:** Verify XSS prevention in user inputs

**Test Cases:**

| Field | Malicious Input | Expected Result |
|-------|----------------|-----------------|
| holiday.name | `<script>alert('XSS')</script>` | HTML escaped in output |
| rule.rule_name | `<img src=x onerror=alert(1)>` | HTML escaped in output |
| booking rejection_reason | `<script>document.cookie</script>` | HTML escaped in output |

**Testing Method:**
1. Create holiday with name: `<script>alert('XSS')</script>`
2. View holiday in admin panel
3. Verify script not executed
4. Inspect HTML: Verify `&lt;script&gt;` (escaped)

#### Test 6.2.3: Time Format Validation

**Purpose:** Verify strict time format enforcement

**Test Cases:**

| Input | Expected Result |
|-------|-----------------|
| "09:00" | Valid ✓ |
| "9:00" | Invalid (missing leading zero) |
| "25:00" | Invalid (hours > 23) |
| "12:60" | Invalid (minutes > 59) |
| "12:00 PM" | Invalid (not 24-hour format) |
| "abc" | Invalid |
| "09:00:00" | Invalid (seconds not allowed) |

**Testing Method:**
```go
func TestTimeFormatValidation(t *testing.T) {
    testCases := []struct {
        input string
        valid bool
    }{
        {"09:00", true},
        {"9:00", false},
        {"25:00", false},
        {"12:60", false},
        {"12:00 PM", false},
    }

    for _, tc := range testCases {
        rule := models.BookingTimeRule{
            DayType: "weekday",
            RuleName: "Test",
            StartTime: tc.input,
            EndTime: "12:00",
        }
        err := rule.Validate()
        if tc.valid && err != nil {
            t.Errorf("Expected valid, got error: %v", err)
        }
        if !tc.valid && err == nil {
            t.Errorf("Expected error, got valid")
        }
    }
}
```

#### Test 6.2.4: Date Format Validation

**Purpose:** Verify strict date format enforcement

**Test Cases:**

| Input | Expected Result |
|-------|-----------------|
| "2025-01-27" | Valid ✓ |
| "2025-1-27" | Invalid (missing leading zeros) |
| "27-01-2025" | Invalid (wrong order) |
| "2025/01/27" | Invalid (wrong separator) |
| "2025-13-01" | Invalid (month > 12) |
| "2025-02-30" | Invalid (invalid day) |

---

## Phase 7: Performance Testing // DONE

### 7.1 Database Query Performance

#### Test 7.1.1: Holiday Lookup Performance

**Purpose:** Verify holiday check is fast with index

**Test Setup:**
- Insert 1000 holidays into custom_holidays table
- Run query 1000 times

**Performance Targets:**

| Query | Target Time | Acceptable |
|-------|-------------|------------|
| Single holiday lookup | < 1ms | < 5ms |
| Year holiday list | < 10ms | < 50ms |

**Testing Method:**
```go
func BenchmarkIsHoliday(b *testing.B) {
    // Setup: Insert 1000 holidays
    for i := 0; i < 1000; i++ {
        date := time.Now().AddDate(0, 0, i).Format("2006-01-02")
        holidayRepo.CreateHoliday(&models.CustomHoliday{
            Date: date,
            Name: fmt.Sprintf("Holiday %d", i),
            IsActive: true,
            Source: "test",
        })
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        holidayRepo.IsHoliday("2025-06-15")
    }
}
```

**Expected Result:**
- Benchmark shows < 1ms per operation
- Index on `custom_holidays.date` utilized

#### Test 7.1.2: Available Slots Generation

**Purpose:** Verify time slot generation is fast

**Performance Targets:**

| Test Case | Target Time |
|-----------|-------------|
| Generate weekday slots | < 10ms |
| Generate weekend slots | < 10ms |
| Generate with 100 rules | < 50ms |

**Testing Method:**
```go
func BenchmarkGetAvailableTimeSlots(b *testing.B) {
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        service.GetAvailableTimeSlots("2025-01-27")
    }
}
```

#### Test 7.1.3: Booking Validation Performance

**Purpose:** Verify booking validation completes quickly

**Performance Targets:**

| Test Case | Target Time |
|-----------|-------------|
| Validate single booking time | < 5ms |
| Validate 100 bookings | < 500ms |

### 7.2 API Response Time

#### Test 7.2.1: Endpoint Response Times

**Purpose:** Verify all endpoints respond within acceptable time

**Performance Targets:**

| Endpoint | Target | Acceptable |
|----------|--------|------------|
| GET /api/booking-times/available | < 50ms | < 200ms |
| GET /api/booking-times/rules | < 50ms | < 200ms |
| GET /api/holidays | < 50ms | < 200ms |
| POST /api/bookings (with validation) | < 100ms | < 500ms |

**Testing Method:**
```bash
# Use Apache Bench
ab -n 1000 -c 10 -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/booking-times/available?date=2025-01-27

# Verify:
# - Mean response time < 50ms
# - 99th percentile < 200ms
# - No failures
```

### 7.3 Concurrent Request Handling

#### Test 7.3.1: Concurrent Booking Creation

**Purpose:** Verify system handles concurrent bookings correctly

**Test Scenario:**
- 50 users simultaneously create bookings
- Same date/time combinations
- Verify no double-bookings created
- Verify time validation still enforced

**Testing Method:**
```go
func TestConcurrentBookingCreation(t *testing.T) {
    var wg sync.WaitGroup
    errors := make([]error, 50)

    for i := 0; i < 50; i++ {
        wg.Add(1)
        go func(index int) {
            defer wg.Done()
            booking := &models.Booking{
                UserID: index + 1,
                DogID: 1,
                Date: "2025-01-27",
                ScheduledTime: "15:00",
                WalkType: "short",
            }
            errors[index] = bookingRepo.CreateBooking(booking)
        }(i)
    }

    wg.Wait()

    // Verify: Only one booking created (UNIQUE constraint)
    bookings, _ := bookingRepo.GetBookingsByDog(1, "2025-01-27")
    if len(bookings) != 1 {
        t.Errorf("Expected 1 booking, got %d", len(bookings))
    }
}
```

### 7.4 Holiday API Caching Effectiveness

#### Test 7.4.1: Cache Hit Rate

**Purpose:** Verify holiday cache prevents excessive API calls

**Test Scenario:**
- Make 100 requests for 2025 holidays
- Verify only 1 API call made
- Verify 99 cache hits

**Testing Method:**
```go
func TestHolidayCacheEffectiveness(t *testing.T) {
    // Mock HTTP client to count API calls
    apiCallCount := 0
    originalTransport := http.DefaultClient.Transport
    http.DefaultClient.Transport = &mockTransport{
        onRequest: func(req *http.Request) {
            if strings.Contains(req.URL.String(), "feiertage-api.de") {
                apiCallCount++
            }
        },
    }
    defer func() { http.DefaultClient.Transport = originalTransport }()

    // Make 100 requests
    for i := 0; i < 100; i++ {
        service.GetHolidaysForYear(2025)
    }

    // Verify only 1 API call
    if apiCallCount != 1 {
        t.Errorf("Expected 1 API call, got %d", apiCallCount)
    }
}
```

---

## Phase 8: Regression Testing // DONE

### 8.1 Existing Booking Functionality

**Purpose:** Ensure existing booking features still work after time restrictions added

**Status:** Regression test file created at `internal/handlers/booking_handler_regression_test.go`
**Tests Implemented:** Basic booking creation, experience level validation, date restrictions, blocked dates, user dashboard, dog browsing
**Result:** Core functionality verified - existing features continue to work with time restrictions added

#### Test 8.1.1: Basic Booking Creation

**Test Cases:**

| Test Case | Description | Expected Result |
|-----------|-------------|-----------------|
| TC-8.1.1-A | Create booking (no time validation) | Still works if within allowed times |
| TC-8.1.1-B | Cancel booking | Still works |
| TC-8.1.1-C | View bookings in dashboard | Still works, new approval status shown |
| TC-8.1.1-D | Admin view all bookings | Still works |

#### Test 8.1.2: Experience Level Validation

**Test Cases:**

| Test Case | Description | Expected Result |
|-----------|-------------|-----------------|
| TC-8.1.2-A | Green user books green dog | Success (with time validation) |
| TC-8.1.2-B | Green user books blue dog | Error (experience level) |
| TC-8.1.2-C | Blue user books orange dog | Error (experience level) |

#### Test 8.1.3: Date Restrictions

**Test Cases:**

| Test Case | Description | Expected Result |
|-----------|-------------|-----------------|
| TC-8.1.3-A | Book date in past | Error |
| TC-8.1.3-B | Book beyond advance limit | Error |
| TC-8.1.3-C | Book on blocked date | Error |
| TC-8.1.3-D | Double booking same dog/date/time | Error |

### 8.2 Existing Admin Functionality

**Purpose:** Ensure admin features still work

#### Test 8.2.1: Admin Dashboard

**Test Cases:**

| Test Case | Expected Result |
|-----------|-----------------|
| TC-8.2.1-A | View admin stats | Still works |
| TC-8.2.1-B | View bookings list | Still works, new approval columns |
| TC-8.2.1-C | View users list | Still works |
| TC-8.2.1-D | Manage dogs | Still works |

#### Test 8.2.2: Blocked Dates

**Test Cases:**

| Test Case | Expected Result |
|-----------|-----------------|
| TC-8.2.2-A | Add blocked date | Still works, prevents all bookings |
| TC-8.2.2-B | Remove blocked date | Still works |
| TC-8.2.2-C | View blocked dates | Still works |

### 8.3 Existing User Functionality

**Purpose:** Ensure user features still work

#### Test 8.3.1: User Dashboard

**Test Cases:**

| Test Case | Expected Result |
|-----------|-----------------|
| TC-8.3.1-A | View upcoming bookings | Still works, approval status shown |
| TC-8.3.1-B | View completed walks | Still works |
| TC-8.3.1-C | Add walk notes | Still works |
| TC-8.3.1-D | Cancel booking | Still works |

#### Test 8.3.2: Dog Browsing

**Test Cases:**

| Test Case | Expected Result |
|-----------|-----------------|
| TC-8.3.2-A | View all dogs | Still works |
| TC-8.3.2-B | Filter by experience level | Still works |
| TC-8.3.2-C | View dog details | Still works |
| TC-8.3.2-D | Check dog availability | Still works |

---

## Phase 9: Database Testing

### 9.1 Migration Testing

#### Test 9.1.1: Fresh Database Migration

**Purpose:** Verify migration works on fresh database

**Test Steps:**
1. Delete existing database file
2. Start application
3. Verify all tables created
4. Verify seed data inserted

**Expected Results:**
- ✅ booking_time_rules table created with 9 seed rows
- ✅ custom_holidays table created (empty)
- ✅ feiertage_cache table created (empty)
- ✅ bookings table has new columns
- ✅ system_settings has new entries

**Verification SQL:**
```sql
-- Check tables exist
SELECT name FROM sqlite_master WHERE type='table'
  AND name IN ('booking_time_rules', 'custom_holidays', 'feiertage_cache');

-- Check seed data
SELECT COUNT(*) FROM booking_time_rules; -- Should be 9

-- Check bookings columns
PRAGMA table_info(bookings); -- Verify new columns present

-- Check system settings
SELECT * FROM system_settings WHERE key LIKE '%morning%' OR key LIKE '%feiertage%';
```

#### Test 9.1.2: Existing Database Migration

**Purpose:** Verify migration works on database with existing data

**Test Steps:**
1. Use database with existing bookings
2. Run migration
3. Verify new tables created
4. Verify existing data preserved
5. Verify new columns added with defaults

**Expected Results:**
- ✅ Existing bookings preserved
- ✅ New columns have default values (requires_approval=0, approval_status='approved')
- ✅ No data loss

**Verification SQL:**
```sql
-- Check existing bookings count before and after
SELECT COUNT(*) FROM bookings;

-- Verify new columns have defaults
SELECT
  COUNT(*) as total,
  SUM(CASE WHEN requires_approval = 0 THEN 1 ELSE 0 END) as default_requires_approval,
  SUM(CASE WHEN approval_status = 'approved' THEN 1 ELSE 0 END) as default_approved
FROM bookings;
-- All should be equal (all have defaults)
```

#### Test 9.1.3: Idempotency

**Purpose:** Verify migration can run multiple times safely

**Test Steps:**
1. Run migration (first time)
2. Run migration again (second time)
3. Verify no errors
4. Verify no duplicate data

**Expected Results:**
- ✅ No SQL errors
- ✅ Seed data not duplicated (INSERT OR IGNORE)
- ✅ System stable

### 9.2 Data Integrity Testing

#### Test 9.2.1: Foreign Key Constraints

**Purpose:** Verify referential integrity maintained

**Test Cases:**

| Test Case | Action | Expected Result |
|-----------|--------|-----------------|
| TC-9.2.1-A | Delete user with approved bookings | Bookings remain, approved_by NULL |
| TC-9.2.1-B | Delete admin who approved bookings | Bookings remain, approved_by NULL (ON DELETE SET NULL) |
| TC-9.2.1-C | Delete holiday created by admin | Holiday deleted |

#### Test 9.2.2: Unique Constraints

**Purpose:** Verify unique constraints enforced

**Test Cases:**

| Test Case | Action | Expected Result |
|-----------|--------|-----------------|
| TC-9.2.2-A | Create duplicate (day_type, rule_name) | Error |
| TC-9.2.2-B | Create duplicate holiday date | Error |
| TC-9.2.2-C | Create duplicate booking (dog, date, walk_type) | Error |

#### Test 9.2.3: Index Effectiveness

**Purpose:** Verify indexes improve query performance

**Test Cases:**

| Index | Query | Performance Improvement |
|-------|-------|-------------------------|
| idx_custom_holidays_date | SELECT * WHERE date = ? | > 10x faster |
| idx_custom_holidays_active | SELECT * WHERE is_active = 1 | > 5x faster |
| idx_bookings_approval_status | SELECT * WHERE approval_status = 'pending' | > 5x faster |

**Testing Method:**
```sql
-- Without index (simulate)
EXPLAIN QUERY PLAN SELECT * FROM custom_holidays WHERE date = '2025-01-01';
-- Should show: SEARCH using INDEX idx_custom_holidays_date

-- Benchmark
CREATE TABLE temp_holidays AS SELECT * FROM custom_holidays;
DROP INDEX IF EXISTS idx_custom_holidays_date;
-- Time query without index
-- Time query with index
-- Compare
```

---

// DONE: Phase 9 - Database Testing

## Phase 10: Browser Compatibility Testing

### 10.1 Frontend Compatibility

**Purpose:** Verify UI works across browsers

**Browsers to Test:**
- Chrome (latest)
- Firefox (latest)
- Safari (latest)
- Edge (latest)

**Features to Test:**

| Feature | Chrome | Firefox | Safari | Edge |
|---------|--------|---------|--------|------|
| Time slot dropdown | ✓ | ✓ | ✓ | ✓ |
| Time input fields | ✓ | ✓ | ✓ | ✓ |
| Tab switching | ✓ | ✓ | ✓ | ✓ |
| Modal dialogs | ✓ | ✓ | ✓ | ✓ |
| API calls (fetch) | ✓ | ✓ | ✓ | ✓ |
| Date picker | ✓ | ✓ | ✓ | ✓ |

**Common Issues to Check:**
- Time input format (Safari handles differently)
- Fetch API (all modern browsers support)
- CSS Grid/Flexbox (well-supported)
- JavaScript ES6+ (well-supported)

### 10.2 Mobile Responsiveness

**Purpose:** Verify UI works on mobile devices

**Devices to Test:**
- iPhone (Safari)
- Android phone (Chrome)
- Tablet (iPad/Android)

**Features to Test:**

| Feature | Mobile | Tablet |
|---------|--------|--------|
| Booking form | ✓ | ✓ |
| Time slot selection | ✓ | ✓ |
| Admin time rules table | ✓ | ✓ |
| Holiday management | ✓ | ✓ |
| Navigation | ✓ | ✓ |

---

// DONE: Phase 10 - Browser Compatibility Testing

## Phase 11: Documentation Testing

### 11.1 API Documentation Accuracy

**Purpose:** Verify API.md documentation matches implementation

**Test Cases:**

| Endpoint | Documented | Implemented | Match |
|----------|------------|-------------|-------|
| GET /api/booking-times/available | ✓ | ✓ | ✓ |
| GET /api/booking-times/rules | ✓ | ✓ | ✓ |
| PUT /api/booking-times/rules | ✓ | ✓ | ✓ |
| POST /api/booking-times/rules | ✓ | ✓ | ✓ |
| DELETE /api/booking-times/rules/:id | ✓ | ✓ | ✓ |
| GET /api/holidays | ✓ | ✓ | ✓ |
| POST /api/holidays | ✓ | ✓ | ✓ |
| PUT /api/holidays/:id | ✓ | ✓ | ✓ |
| DELETE /api/holidays/:id | ✓ | ✓ | ✓ |
| GET /api/bookings/pending-approvals | ✓ | ✓ | ✓ |
| PUT /api/bookings/:id/approve | ✓ | ✓ | ✓ |
| PUT /api/bookings/:id/reject | ✓ | ✓ | ✓ |

**Verification:**
- Request/response examples match actual behavior
- Error codes documented correctly
- Authorization requirements documented

### 11.2 User Guide Accuracy

**Purpose:** Verify USER_GUIDE.md accurately describes features

**Test Cases:**
- Follow user guide step-by-step for booking with time restrictions
- Verify screenshots (if any) match current UI
- Verify instructions accurate

### 11.3 Admin Guide Accuracy

**Purpose:** Verify ADMIN_GUIDE.md accurately describes admin features

**Test Cases:**
- Follow admin guide for configuring time windows
- Follow admin guide for managing holidays
- Follow admin guide for approving bookings
- Verify all steps work as documented

---

## Test Execution Summary

### Test Coverage Matrix

| Category | Test Count | Status | Pass Rate |
|----------|-----------|--------|-----------|
| Unit Tests - Services | 15 tests | ⏳ Pending | - |
| Unit Tests - Repositories | 12 tests | ⏳ Pending | - |
| Integration Tests - API | 18 tests | ⏳ Pending | - |
| Frontend Tests - Manual | 20 tests | ⏳ Pending | - |
| End-to-End Tests | 5 scenarios | ⏳ Pending | - |
| Security Tests | 10 tests | ⏳ Pending | - |
| Performance Tests | 8 tests | ⏳ Pending | - |
| Regression Tests | 15 tests | ⏳ Pending | - |
| Database Tests | 10 tests | ⏳ Pending | - |
| Browser Compatibility | 6 browsers | ⏳ Pending | - |
| **TOTAL** | **119 tests** | ⏳ Pending | - |

### Priority Testing Order

**Phase 1 (Critical):** Must pass before deployment
1. Unit Tests - Services (booking time validation)
2. Unit Tests - Repositories (CRUD operations)
3. Integration Tests - API (endpoint functionality)
4. Security Tests - Authorization (admin protection)
5. Database Tests - Migration (data integrity)

**Phase 2 (High Priority):** Must pass for production
6. Frontend Tests - Booking form (user experience)
7. Frontend Tests - Admin pages (admin experience)
8. End-to-End Tests - Complete workflows
9. Regression Tests - Existing features (no breaking changes)

**Phase 3 (Medium Priority):** Should pass for quality
10. Performance Tests (acceptable response times)
11. Browser Compatibility (major browsers)
12. Security Tests - Input validation (XSS, SQL injection)

**Phase 4 (Low Priority):** Nice to have
13. Holiday API Tests - Caching (optimization)
14. Mobile Responsiveness (mobile support)
15. Documentation Tests (accuracy)

---

## Test Automation

### Automated Test Execution

**Run All Unit Tests:**
```bash
go test ./internal/services/... -v
go test ./internal/repository/... -v
go test ./internal/handlers/... -v
```

**Run Specific Test Suite:**
```bash
go test ./internal/services/booking_time_service_test.go -v
go test ./internal/services/holiday_service_test.go -v
```

**Run with Coverage:**
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

**Target Coverage:**
- Services: 80%+
- Repositories: 70%+
- Handlers: 60%+

### Continuous Integration

**Recommended CI Pipeline:**
```yaml
# .github/workflows/test.yml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: '1.21'
      - name: Run unit tests
        run: go test ./... -v
      - name: Run integration tests
        run: go test ./internal/handlers/... -v
      - name: Generate coverage
        run: go test ./... -coverprofile=coverage.out
      - name: Upload coverage
        uses: codecov/codecov-action@v2
```

---

## Bug Tracking Template

### Bug Report Format

```
**Bug ID:** BT-001
**Severity:** Critical | High | Medium | Low
**Priority:** P0 | P1 | P2 | P3
**Component:** Service | Repository | Handler | Frontend
**Test Case:** TC-X.X.X-X

**Description:**
[Clear description of the bug]

**Steps to Reproduce:**
1. Step 1
2. Step 2
3. Step 3

**Expected Result:**
[What should happen]

**Actual Result:**
[What actually happens]

**Screenshots/Logs:**
[Attach relevant screenshots or error logs]

**Environment:**
- Browser: Chrome 120
- OS: Windows 11
- Database: SQLite 3.x
- Go Version: 1.21

**Workaround:**
[Temporary workaround if available]

**Fix:**
[Proposed fix if known]
```

---

## Sign-Off Checklist

### Pre-Deployment Verification

**Development Team:**
- [ ] All unit tests pass (100%)
- [ ] All integration tests pass (100%)
- [ ] Code review completed
- [ ] No critical or high-severity bugs
- [ ] Performance benchmarks met
- [ ] Security tests pass

**QA Team:**
- [ ] All manual tests completed
- [ ] End-to-end scenarios verified
- [ ] Browser compatibility confirmed
- [ ] Regression tests pass
- [ ] User acceptance testing completed

**DevOps Team:**
- [ ] Migration tested on production-like environment
- [ ] Backup strategy verified
- [ ] Rollback plan documented
- [ ] Monitoring configured

**Product Owner:**
- [ ] Feature requirements met
- [ ] User documentation updated
- [ ] Admin documentation updated
- [ ] Training materials prepared (if needed)

---

## Appendix

### A. Test Data Sets

**Users:**
- Regular User 1: green level, email: green@test.com
- Regular User 2: blue level, email: blue@test.com
- Regular User 3: orange level, email: orange@test.com
- Admin User: email: admin@test.com

**Dogs:**
- Dog 1: Green level, available
- Dog 2: Blue level, available
- Dog 3: Orange level, available
- Dog 4: Green level, unavailable

**Dates:**
- Weekday: 2025-01-27 (Monday)
- Weekend: 2025-01-25 (Saturday)
- Holiday: 2025-01-01 (Neujahrstag, Wednesday)
- Blocked: 2025-02-14 (if configured)

### B. Test Environment Setup

**Local Development:**
```bash
# Clone repository
git clone <repo-url>
cd gassigeher

# Build
go build -o gassigeher.exe ./cmd/server

# Run tests
go test ./... -v

# Start server
./gassigeher.exe
```

**Test Database:**
```bash
# Use separate test database
export DATABASE_PATH=./test_gassigeher.db

# Seed test data
go run ./scripts/seed_test_data.go

# Run server
./gassigeher.exe
```

### C. Useful SQL Queries for Testing

**Check time rules:**
```sql
SELECT * FROM booking_time_rules ORDER BY day_type, start_time;
```

**Check holidays:**
```sql
SELECT * FROM custom_holidays WHERE is_active = 1 ORDER BY date;
```

**Check pending approvals:**
```sql
SELECT b.*, u.name as user_name, d.name as dog_name
FROM bookings b
JOIN users u ON b.user_id = u.id
JOIN dogs d ON b.dog_id = d.id
WHERE b.approval_status = 'pending'
ORDER BY b.date, b.scheduled_time;
```

**Check cache:**
```sql
SELECT year, state, datetime(expires_at) as expires,
       CASE WHEN expires_at > datetime('now') THEN 'Valid' ELSE 'Expired' END as status
FROM feiertage_cache;
```

### D. Performance Benchmarking

**Apache Bench Commands:**
```bash
# Test available slots endpoint
ab -n 1000 -c 10 \
  -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/booking-times/available?date=2025-01-27

# Test booking creation
ab -n 100 -c 5 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -p booking.json \
  http://localhost:8080/api/bookings
```

---

**Document Version:** 1.0
**Last Updated:** 2025-01-23
**Next Review:** After implementation completion

**Prepared By:** Development Team
**Approved By:** Project Lead
**Status:** Active Testing Document
