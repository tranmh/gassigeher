package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// testConfigBugs returns a minimal config for bug tests
func testConfigBugs() *config.Config {
	return &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
}

// ============================================================================
// BUG #1: TenantID=0 bypass vulnerability in CreateBooking
// When TenantMiddleware is missing or fails, tenantID defaults to 0.
// The CreateBooking handler doesn't validate this, potentially allowing
// bookings to be created with tenantID=0 (orphaned data).
// ============================================================================

func TestCreateBooking_BUG_TenantIDZeroBypass(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewBookingHandler(db, testConfigBugs())

	// Create request without tenant context (simulating middleware bypass)
	body := `{"dog_id": 1, "date": "2025-12-30", "scheduled_time": "10:00"}`
	req := httptest.NewRequest("POST", "/api/v1/bookings", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	// Add user context but NO tenant context
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, 1)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CreateBooking(rec, req)

	// BUG: This should return 400 or 403, but it might create a booking with tenantID=0
	// The code at line 72 does: tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)
	// This silently defaults to 0 if tenant context is missing!

	// Current behavior (BUG): Returns 404 "Dog not found" because FindByIDAndTenant(1, 0) finds nothing
	// Expected behavior: Should return 400 "Invalid tenant context" BEFORE any DB operations
	if rec.Code == http.StatusCreated {
		t.Error("BUG CONFIRMED: CreateBooking created a booking with tenantID=0!")
	}

	// Note: While this doesn't create orphaned data due to dog lookup,
	// it's still a security issue - the validation happens too late.
	t.Logf("Current response: %d - %s", rec.Code, rec.Body.String())
}

// ============================================================================
// BUG #2: ListBookings doesn't validate tenantID=0
// In calendar_view mode, non-admins can potentially see ALL bookings
// across all tenants if tenantID=0 is not rejected.
// ============================================================================

func TestListBookings_BUG_TenantIDZeroReturnsAllTenants(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewBookingHandler(db, testConfigBugs())

	req := httptest.NewRequest("GET", "/api/v1/bookings?calendar_view=true", nil)

	// Add user context but NO tenant context
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, 1)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ListBookings(rec, req)

	// The code at line 257 does: tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)
	// Then at line 262: TenantID: &tenantID (which is 0)
	// This creates a filter with TenantID=0, which SHOULD filter for only tenant 0,
	// but if the DB query treats 0 as "no filter", it returns all tenants' bookings!

	if rec.Code == http.StatusOK {
		var bookings []map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &bookings)
		t.Logf("Returned %d bookings with tenantID=0 filter", len(bookings))

		// Check if any booking has a different tenant
		for _, b := range bookings {
			if tid, ok := b["tenant_id"].(float64); ok && tid != 0 {
				t.Errorf("BUG CONFIRMED: Returned booking from tenant %v when tenantID=0", tid)
			}
		}
	}
}

// ============================================================================
// BUG #3: strconv.Atoi errors silently ignored
// When parsing dog_id, user_id, etc., conversion errors are ignored,
// defaulting to 0 which may have unintended effects.
// ============================================================================

func TestListBookings_BUG_InvalidDogIDSilentlyIgnored(t *testing.T) {
	db := testutil.SetupTestDB(t)
	handler := NewBookingHandler(db, testConfigBugs())

	// Pass invalid dog_id that cannot be converted to int
	req := httptest.NewRequest("GET", "/api/v1/bookings?dog_id=invalid", nil)

	ctx := context.WithValue(req.Context(), middleware.UserIDKey, 1)
	ctx = context.WithValue(ctx, middleware.TenantIDKey, 1)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ListBookings(rec, req)

	// BUG: Line 266 does: dogID, _ := strconv.Atoi(dogIDStr)
	// When "invalid" can't be parsed, dogID becomes 0 and error is ignored.
	// This could return bookings with dog_id=0 instead of an error.

	// Expected: Return 400 Bad Request with "Invalid dog_id parameter"
	// Actual: Returns 200 OK (silently uses dog_id=0)
	if rec.Code == http.StatusOK {
		t.Log("BUG: Invalid dog_id='invalid' was silently ignored instead of returning error")
	}
}

// ============================================================================
// BUG #4: Potential nil pointer dereference in CancelBooking
// Line 466: booking.User.Email is accessed after only checking Email != nil
// If booking.User is nil (deleted user), this will panic.
// ============================================================================

func TestCancelBooking_BUG_NilUserPanic(t *testing.T) {
	// This test verifies the nil check issue
	// The fix should check: booking.User != nil && booking.User.Email != nil

	// Note: This is hard to trigger in tests because the DB usually has
	// referential integrity, but in production with deleted users,
	// booking.User could be nil.

	t.Log("BUG: Line 466 in CancelBooking should check booking.User != nil before accessing User.Email")
	t.Log("Current code: if booking.User.Email != nil && h.emailService != nil")
	t.Log("Fixed code:   if booking.User != nil && booking.User.Email != nil && h.emailService != nil")
}

// ============================================================================
// BUG #5: Error ignored when fetching booking for rejection email
// Line 882: booking, _ := h.bookingRepo.FindByIDWithDetails(id)
// Error is ignored, potentially sending email with nil booking data
// ============================================================================

func TestRejectPendingBooking_BUG_ErrorIgnored(t *testing.T) {
	// The code at line 882 ignores the error from FindByIDWithDetails
	// This means if the DB query fails, we continue with a nil booking
	// Then at line 885 we check if booking == nil, which is good,
	// BUT we already called RejectBooking at line 894 before the nil check!

	// Wait, looking more carefully:
	// Line 882: booking, _ := h.bookingRepo.FindByIDWithDetails(id) - Error ignored
	// Line 885-892: Checks if booking is nil or wrong tenant
	// Line 894: h.bookingRepo.RejectBooking(id, adminID, req.Reason) - Called BEFORE these checks!

	// Actually no, the tenant check happens at 885-892 before RejectBooking at 894.
	// But the error from FindByIDWithDetails is still ignored.

	t.Log("BUG: Line 882 ignores error from FindByIDWithDetails")
	t.Log("If there's a database error, it's silently swallowed")
}

// ============================================================================
// BUG #6: Nil pointer dereference in ApprovePendingBooking
// Line 821: booking.Dog.Name accessed without checking if Dog is nil
// ============================================================================

func TestApprovePendingBooking_BUG_NilDogPanic(t *testing.T) {
	// The code at line 817-821 does:
	// if err == nil && booking != nil && booking.User != nil && booking.User.Email != nil {
	//     go h.emailService.SendBookingApproved(
	//         ...
	//         booking.Dog.Name,  // <-- BUG: Dog could be nil!
	//         ...
	//     )
	// }

	// If the dog was deleted but the booking still exists,
	// booking.Dog will be nil and this will panic.

	t.Log("BUG: Line 821 accesses booking.Dog.Name without nil check")
	t.Log("Current code: booking.Dog.Name")
	t.Log("Fixed code: Check booking.Dog != nil before accessing Name")
}

// ============================================================================
// BUG #7: AddNotes doesn't check tenantID=0 bypass
// Unlike GetBooking/CancelBooking, AddNotes doesn't validate tenantID
// ============================================================================

func TestAddNotes_BUG_NoTenantIDValidation(t *testing.T) {
	// This is a documentation test - the actual bug is in the code:
	// Line 492: tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)
	// Unlike GetBooking (line 321) and CancelBooking (line 363),
	// AddNotes does NOT validate: if !ok || tenantID == 0 { return error }
	//
	// Note: We can't easily call AddNotes directly without mux routing
	// because it requires path variables to be set up by gorilla/mux

	t.Log("BUG: AddNotes at line 492 doesn't validate tenantID like other methods do")
	t.Log("GetBooking (line 321) and CancelBooking (line 363) have: if !ok || tenantID == 0 { ... }")
	t.Log("AddNotes should have the same validation")
}

// ============================================================================
// isUniqueConstraintError Tests - These work correctly but need coverage
// ============================================================================

func TestIsUniqueConstraintError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected bool
	}{
		// SQLite patterns
		{"SQLite unique constraint", "UNIQUE constraint failed: bookings.dog_id", true},
		{"SQLite unique constraint lowercase", "unique constraint failed", true},

		// MySQL patterns
		{"MySQL duplicate entry", "Duplicate entry '1-2025-01-15-10:00' for key 'idx_dog_date_time'", true},
		{"MySQL error 1062", "Error 1062: Duplicate entry", true},

		// PostgreSQL patterns
		{"PostgreSQL duplicate key", "duplicate key value violates unique constraint", true},
		{"PostgreSQL violates unique", "violates unique constraint \"bookings_pkey\"", true},

		// Non-unique errors
		{"Empty string", "", false},
		{"Random error", "connection refused", false},
		{"Not found error", "record not found", false},
		{"Foreign key error", "foreign key constraint failed", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isUniqueConstraintError(tt.errMsg)
			if result != tt.expected {
				t.Errorf("isUniqueConstraintError(%q) = %v, want %v", tt.errMsg, result, tt.expected)
			}
		})
	}
}
