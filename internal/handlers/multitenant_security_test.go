package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// TestMultiTenant_CreateBooking_IncludesTenantID tests that bookings are created with the correct tenant_id
// TDD RED PHASE: This test should FAIL until we fix CreateBooking to include tenant_id
func TestMultiTenant_CreateBooking_IncludesTenantID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewBookingHandler(db, cfg)

	// Create test user and dog for tenant 1
	// SeedTestUser already assigns colors based on level, so don't add colors again
	userID := testutil.SeedTestUser(t, db, "user@tenant1.com", "Test User", "green")
	dogID := testutil.SeedTestDog(t, db, "Buddy", "Labrador", "green") // category=green assigns colorID=1

	// Create booking request
	futureDate := testutil.GetFutureDate(7) // 7 days from now
	reqBody := map[string]interface{}{
		"dog_id":         dogID,
		"date":           futureDate,
		"scheduled_time": "14:00",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/bookings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Set context with tenant_id = 1
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, 1)
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, middleware.EmailKey, "user@tenant1.com")
	ctx = context.WithValue(ctx, middleware.IsAdminKey, false)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CreateBooking(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// Parse response to get booking ID
	var booking models.Booking
	if err := json.Unmarshal(rec.Body.Bytes(), &booking); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify booking was created with correct tenant_id
	bookingRepo := repository.NewBookingRepository(db)
	savedBooking, err := bookingRepo.FindByID(booking.ID)
	if err != nil {
		t.Fatalf("Failed to find booking: %v", err)
	}

	// BUG: The booking should have tenant_id = 1, but it's likely 0 (NULL)
	if savedBooking.TenantID != 1 {
		t.Errorf("BUG: Booking created with tenant_id = %d, expected 1", savedBooking.TenantID)
	}
}

// TestMultiTenant_CheckDoubleBooking_IsolatedByTenant tests that double-booking check is tenant-isolated
// TDD RED PHASE: This test should FAIL until we fix CheckDoubleBooking to filter by tenant
func TestMultiTenant_CheckDoubleBooking_IsolatedByTenant(t *testing.T) {
	db := testutil.SetupTestDB(t)

	// Create user and dog for tenant 1 to satisfy foreign key
	userID := testutil.SeedTestUser(t, db, "user@tenant1.com", "Test User", "green")
	dogID := testutil.SeedTestDog(t, db, "Buddy", "Labrador", "green")

	bookingRepo := repository.NewBookingRepository(db)

	// Create a booking for tenant 1
	booking1 := &models.Booking{
		TenantID:       1,
		UserID:         userID,
		DogID:          dogID,
		Date:           "2025-12-25",
		ScheduledTime:  "14:00",
		Status:         "scheduled",
		ApprovalStatus: "approved",
	}
	if err := bookingRepo.Create(booking1); err != nil {
		t.Fatalf("Failed to create booking for tenant 1: %v", err)
	}

	// Now check if the same dog_id/date/time is considered double-booked
	// BUG: CheckDoubleBooking doesn't filter by tenant_id
	isDoubleBooked, err := bookingRepo.CheckDoubleBooking(dogID, "2025-12-25", "14:00")
	if err != nil {
		t.Fatalf("Failed to check double booking: %v", err)
	}

	// This documents the current behavior - the method should ideally accept tenant_id parameter
	if isDoubleBooked {
		t.Log("NOTE: CheckDoubleBooking found double-booking (expected behavior)")
		t.Log("However, the method lacks tenant isolation - it should accept tenant_id parameter")
	}
}

// TestMultiTenant_ListBookings_FilterByTenant tests that ListBookings only returns current tenant's bookings
// TDD RED PHASE: This test should FAIL until we fix ListBookings to filter by tenant
func TestMultiTenant_ListBookings_FilterByTenant(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewBookingHandler(db, cfg)

	// Create user and dog for tenant 1
	userID := testutil.SeedTestUser(t, db, "user@tenant1.com", "Test User", "green")
	dogID := testutil.SeedTestDog(t, db, "Buddy", "Labrador", "green")

	// Create tenant 2 in the database
	now := testutil.NowTime()
	_, err := db.Exec(`INSERT INTO tenants (id, slug, name, status, contact_email, created_at, updated_at)
		VALUES (2, 'tenant2', 'Tenant 2', 'active', 'tenant2@example.com', ?, ?)`, now, now)
	if err != nil {
		t.Fatalf("Failed to create tenant 2: %v", err)
	}

	bookingRepo := repository.NewBookingRepository(db)

	// Create booking for tenant 1
	booking1 := &models.Booking{
		TenantID:       1,
		UserID:         userID,
		DogID:          dogID,
		Date:           "2025-12-20",
		ScheduledTime:  "10:00",
		Status:         "scheduled",
		ApprovalStatus: "approved",
	}
	if err := bookingRepo.Create(booking1); err != nil {
		t.Fatalf("Failed to create booking for tenant 1: %v", err)
	}

	// Create booking for tenant 2 (using same user/dog IDs - allowed because different tenant)
	booking2 := &models.Booking{
		TenantID:       2,
		UserID:         userID, // same user ID but different tenant context
		DogID:          dogID,  // same dog ID but different tenant context
		Date:           "2025-12-21",
		ScheduledTime:  "11:00",
		Status:         "scheduled",
		ApprovalStatus: "approved",
	}
	if err := bookingRepo.Create(booking2); err != nil {
		t.Fatalf("Failed to create booking for tenant 2: %v", err)
	}

	// Request as admin from tenant 1
	req := httptest.NewRequest("GET", "/api/bookings", nil)
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, 1)
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, middleware.IsAdminKey, true)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ListBookings(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var bookings []*models.Booking
	if err := json.Unmarshal(rec.Body.Bytes(), &bookings); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// BUG: Should only return tenant 1's booking (1 booking), but likely returns both (2 bookings)
	tenant2Count := 0
	for _, b := range bookings {
		if b.TenantID == 2 {
			tenant2Count++
		}
	}

	if tenant2Count > 0 {
		t.Errorf("BUG: ListBookings returned %d booking(s) from tenant 2 when requesting as tenant 1", tenant2Count)
	}
}

// TestMultiTenant_GetBooking_CrossTenantBlocked tests that users can't access bookings from other tenants
// TDD RED PHASE: This test should FAIL until we add tenant verification to GetBooking
func TestMultiTenant_GetBooking_CrossTenantBlocked(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewBookingHandler(db, cfg)

	// Create user and dog for tenant 1
	userID := testutil.SeedTestUser(t, db, "user@tenant1.com", "Test User", "green")
	dogID := testutil.SeedTestDog(t, db, "Buddy", "Labrador", "green")

	// Create tenant 2
	now := testutil.NowTime()
	_, err := db.Exec(`INSERT INTO tenants (id, slug, name, status, contact_email, created_at, updated_at)
		VALUES (2, 'tenant2', 'Tenant 2', 'active', 'tenant2@example.com', ?, ?)`, now, now)
	if err != nil {
		t.Fatalf("Failed to create tenant 2: %v", err)
	}

	bookingRepo := repository.NewBookingRepository(db)

	// Create booking for tenant 2
	booking := &models.Booking{
		TenantID:       2,
		UserID:         userID,
		DogID:          dogID,
		Date:           "2025-12-22",
		ScheduledTime:  "12:00",
		Status:         "scheduled",
		ApprovalStatus: "approved",
	}
	if err := bookingRepo.Create(booking); err != nil {
		t.Fatalf("Failed to create booking: %v", err)
	}

	// Try to access as admin from tenant 1
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/bookings/%d", booking.ID), nil)
	req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", booking.ID)})
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, 1) // Tenant 1!
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, middleware.IsAdminKey, true) // Even admin should be blocked
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.GetBooking(rec, req)

	// BUG: Should return 404 or 403, but likely returns 200 with the booking
	// because GetBooking doesn't verify tenant_id
	if rec.Code == http.StatusOK {
		t.Error("BUG: GetBooking allowed cross-tenant access - admin from tenant 1 accessed tenant 2's booking")
	}
}

// TestMultiTenant_CancelBooking_CrossTenantBlocked tests that admins can't cancel bookings from other tenants
// TDD RED PHASE: This test should FAIL until we add tenant verification to CancelBooking
func TestMultiTenant_CancelBooking_CrossTenantBlocked(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewBookingHandler(db, cfg)

	// Create user and dog for tenant 1
	userID := testutil.SeedTestUser(t, db, "user@tenant1.com", "Test User", "green")
	dogID := testutil.SeedTestDog(t, db, "Buddy", "Labrador", "green")

	// Create tenant 2
	now := testutil.NowTime()
	_, err := db.Exec(`INSERT INTO tenants (id, slug, name, status, contact_email, created_at, updated_at)
		VALUES (2, 'tenant2', 'Tenant 2', 'active', 'tenant2@example.com', ?, ?)`, now, now)
	if err != nil {
		t.Fatalf("Failed to create tenant 2: %v", err)
	}

	bookingRepo := repository.NewBookingRepository(db)

	// Create booking for tenant 2
	booking := &models.Booking{
		TenantID:       2,
		UserID:         userID,
		DogID:          dogID,
		Date:           "2025-12-23",
		ScheduledTime:  "13:00",
		Status:         "scheduled",
		ApprovalStatus: "approved",
	}
	if err := bookingRepo.Create(booking); err != nil {
		t.Fatalf("Failed to create booking: %v", err)
	}

	// Try to cancel as admin from tenant 1
	reqBody := map[string]interface{}{
		"reason": "Testing cross-tenant cancel",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/bookings/%d", booking.ID), bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", booking.ID)})
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, 1) // Tenant 1!
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, middleware.IsAdminKey, true)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CancelBooking(rec, req)

	// BUG: Should return 404 or 403, but likely returns 200 (success)
	if rec.Code == http.StatusOK {
		t.Error("BUG: CancelBooking allowed cross-tenant cancellation - admin from tenant 1 cancelled tenant 2's booking")
	}

	// Verify booking was NOT cancelled
	savedBooking, _ := bookingRepo.FindByID(booking.ID)
	if savedBooking != nil && savedBooking.Status == "cancelled" {
		t.Error("BUG: Cross-tenant cancellation actually modified the booking!")
	}
}

// TestMultiTenant_ApproveBooking_CrossTenantBlocked tests that admins can't approve bookings from other tenants
// TDD RED PHASE: This test should FAIL until we add tenant verification to ApprovePendingBooking
func TestMultiTenant_ApproveBooking_CrossTenantBlocked(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewBookingHandler(db, cfg)

	// Create user and dog for tenant 1
	userID := testutil.SeedTestUser(t, db, "user@tenant1.com", "Test User", "green")
	dogID := testutil.SeedTestDog(t, db, "Buddy", "Labrador", "green")

	// Create tenant 2
	now := testutil.NowTime()
	_, err := db.Exec(`INSERT INTO tenants (id, slug, name, status, contact_email, created_at, updated_at)
		VALUES (2, 'tenant2', 'Tenant 2', 'active', 'tenant2@example.com', ?, ?)`, now, now)
	if err != nil {
		t.Fatalf("Failed to create tenant 2: %v", err)
	}

	bookingRepo := repository.NewBookingRepository(db)

	// Create pending booking for tenant 2
	booking := &models.Booking{
		TenantID:         2,
		UserID:           userID,
		DogID:            dogID,
		Date:             "2025-12-24",
		ScheduledTime:    "08:00",
		Status:           "scheduled",
		RequiresApproval: true,
		ApprovalStatus:   "pending",
	}
	if err := bookingRepo.Create(booking); err != nil {
		t.Fatalf("Failed to create booking: %v", err)
	}

	// Try to approve as admin from tenant 1
	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/bookings/%d/approve", booking.ID), nil)
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, 1) // Tenant 1!
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, middleware.IsAdminKey, true)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ApprovePendingBooking(rec, req)

	// BUG: Should return 404 or 403, but likely returns 200 (success)
	if rec.Code == http.StatusOK {
		t.Error("BUG: ApprovePendingBooking allowed cross-tenant approval - admin from tenant 1 approved tenant 2's booking")
	}

	// Verify booking was NOT approved
	savedBooking, _ := bookingRepo.FindByID(booking.ID)
	if savedBooking != nil && savedBooking.ApprovalStatus == "approved" {
		t.Error("BUG: Cross-tenant approval actually modified the booking!")
	}
}

// TestMultiTenant_MoveBooking_CrossTenantBlocked tests that admins can't move bookings from other tenants
// TDD RED PHASE: This test should FAIL until we add tenant verification to MoveBooking
func TestMultiTenant_MoveBooking_CrossTenantBlocked(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewBookingHandler(db, cfg)

	// Create user and dog for tenant 1
	userID := testutil.SeedTestUser(t, db, "user@tenant1.com", "Test User", "green")
	dogID := testutil.SeedTestDog(t, db, "Buddy", "Labrador", "green")

	// Create tenant 2
	now := testutil.NowTime()
	_, err := db.Exec(`INSERT INTO tenants (id, slug, name, status, contact_email, created_at, updated_at)
		VALUES (2, 'tenant2', 'Tenant 2', 'active', 'tenant2@example.com', ?, ?)`, now, now)
	if err != nil {
		t.Fatalf("Failed to create tenant 2: %v", err)
	}

	bookingRepo := repository.NewBookingRepository(db)

	// Create booking for tenant 2
	booking := &models.Booking{
		TenantID:       2,
		UserID:         userID,
		DogID:          dogID,
		Date:           "2025-12-25",
		ScheduledTime:  "15:00",
		Status:         "scheduled",
		ApprovalStatus: "approved",
	}
	if err := bookingRepo.Create(booking); err != nil {
		t.Fatalf("Failed to create booking: %v", err)
	}

	// Try to move as admin from tenant 1
	reqBody := map[string]interface{}{
		"date":           "2025-12-26",
		"scheduled_time": "16:00",
		"reason":         "Testing cross-tenant move",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/bookings/%d/move", booking.ID), bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", booking.ID)})
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, 1) // Tenant 1!
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, middleware.IsAdminKey, true)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.MoveBooking(rec, req)

	// BUG: Should return 404 or 403, but likely returns 200 (success)
	if rec.Code == http.StatusOK {
		t.Error("BUG: MoveBooking allowed cross-tenant move - admin from tenant 1 moved tenant 2's booking")
	}

	// Verify booking was NOT moved
	savedBooking, _ := bookingRepo.FindByID(booking.ID)
	if savedBooking != nil && savedBooking.Date == "2025-12-26" {
		t.Error("BUG: Cross-tenant move actually modified the booking!")
	}
}

// TestMultiTenant_AddNotes_CrossTenantBlocked tests that users cannot add notes to another tenant's booking
// TDD RED PHASE: This test should FAIL until we add tenant verification
func TestMultiTenant_AddNotes_CrossTenantBlocked(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewBookingHandler(db, cfg)
	bookingRepo := repository.NewBookingRepository(db)

	// Create user and dog in tenant 1 (test tenant)
	userID := testutil.SeedTestUser(t, db, "user1@tenant1.com", "User One", "green")
	dogID := testutil.SeedTestDog(t, db, "Buddy", "Labrador", "green")

	// Create a completed booking in tenant 1 (owned by userID)
	// But we'll try to access it from tenant 2's context
	booking := &models.Booking{
		TenantID:      1, // Tenant 1 owns this booking
		UserID:        userID,
		DogID:         dogID,
		Date:          "2025-12-15", // Past date
		ScheduledTime: "14:00",
		Status:        "completed",
	}
	if err := bookingRepo.Create(booking); err != nil {
		t.Fatalf("Failed to create booking: %v", err)
	}

	// User from tenant 2 tries to add notes to tenant 1's booking
	reqBody := map[string]interface{}{
		"notes": "Some notes from tenant 2 user",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/bookings/%d/notes", booking.ID), bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", booking.ID)})
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, 2) // Tenant 2 - different tenant!
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)         // Same user ID but different tenant context
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.AddNotes(rec, req)

	// BUG: Should return 404 or 403, but might allow cross-tenant access
	if rec.Code == http.StatusOK {
		t.Errorf("BUG: AddNotes allowed cross-tenant access - user from tenant 2 added notes to tenant 1's booking. Status: %d", rec.Code)
	}

	// Expect either 404 (not found) or 403 (forbidden)
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusForbidden {
		t.Errorf("Expected 404 or 403, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// TestMultiTenant_RejectBooking_CrossTenantBlocked tests that admins cannot reject another tenant's booking
// TDD RED PHASE: This test should FAIL until we add tenant verification
func TestMultiTenant_RejectBooking_CrossTenantBlocked(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewBookingHandler(db, cfg)
	bookingRepo := repository.NewBookingRepository(db)

	// Create user and dog in tenant 1 (test tenant)
	userID := testutil.SeedTestUser(t, db, "user@tenant1.com", "Tenant1 User", "green")
	dogID := testutil.SeedTestDog(t, db, "Rex", "German Shepherd", "green")

	// Create a scheduled booking in tenant 1 that requires approval
	futureDate := testutil.GetFutureDate(7)
	booking := &models.Booking{
		TenantID:         1, // Tenant 1 owns this booking
		UserID:           userID,
		DogID:            dogID,
		Date:             futureDate,
		ScheduledTime:    "14:00",
		Status:           "scheduled", // Valid status
		RequiresApproval: true,
	}
	if err := bookingRepo.Create(booking); err != nil {
		t.Fatalf("Failed to create booking: %v", err)
	}

	// Admin from tenant 2 tries to reject tenant 1's booking
	reqBody := map[string]interface{}{
		"reason": "Rejected by wrong tenant admin",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/bookings/%d/reject", booking.ID), bytes.NewReader(body))
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, 2) // Tenant 2 - different tenant!
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, middleware.IsAdminKey, true)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.RejectPendingBooking(rec, req)

	// BUG: Should return 404 or 403, but likely returns 200 (success)
	if rec.Code == http.StatusOK {
		t.Errorf("BUG: RejectPendingBooking allowed cross-tenant rejection - admin from tenant 2 rejected tenant 1's booking")
	}

	// Verify booking was NOT rejected
	savedBooking, _ := bookingRepo.FindByID(booking.ID)
	if savedBooking != nil && savedBooking.Status == "rejected" {
		t.Error("BUG: Cross-tenant rejection actually changed the booking status!")
	}
}

// ====================================================================================
// NEW BUG: TenantID == 0 BYPASS VULNERABILITY
// ====================================================================================
// The pattern `if tenantID > 0 && booking.TenantID != tenantID` has a critical flaw:
// When tenantID is 0 (not set in context), the ENTIRE tenant check is bypassed!
// This means ANY request without proper tenant context can access ANY booking.
// ====================================================================================

// TestMultiTenant_GetBooking_ZeroTenantID_Blocked tests that tenantID=0 is blocked
// TDD RED PHASE: This test should FAIL - exposes the tenantID==0 bypass bug
func TestMultiTenant_GetBooking_ZeroTenantID_Blocked(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewBookingHandler(db, cfg)

	// Create user and dog for tenant 1
	userID := testutil.SeedTestUser(t, db, "user@tenant1.com", "Test User", "green")
	dogID := testutil.SeedTestDog(t, db, "Buddy", "Labrador", "green")

	bookingRepo := repository.NewBookingRepository(db)

	// Create booking for tenant 1
	booking := &models.Booking{
		TenantID:       1,
		UserID:         userID,
		DogID:          dogID,
		Date:           "2025-12-26",
		ScheduledTime:  "10:00",
		Status:         "scheduled",
		ApprovalStatus: "approved",
	}
	if err := bookingRepo.Create(booking); err != nil {
		t.Fatalf("Failed to create booking: %v", err)
	}

	// Try to access with tenantID = 0 (not set) - this should be BLOCKED
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/bookings/%d", booking.ID), nil)
	req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", booking.ID)})
	// CRITICAL: TenantIDKey = 0 (default when not set) simulates missing tenant context
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, 0) // tenantID = 0!
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, middleware.IsAdminKey, true)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.GetBooking(rec, req)

	// BUG: The current code allows this because `if tenantID > 0 && ...` is false when tenantID=0
	// This bypasses ALL tenant isolation!
	if rec.Code == http.StatusOK {
		t.Errorf("SECURITY BUG: GetBooking allowed access with tenantID=0 - tenant isolation bypassed!")
		t.Errorf("Current code: 'if tenantID > 0 && booking.TenantID != tenantID' - condition is false when tenantID=0")
	}

	// Should return 400 (bad request - no tenant) or 403 (forbidden)
	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusForbidden && rec.Code != http.StatusNotFound {
		t.Errorf("Expected 400/403/404, got %d. Response: %s", rec.Code, rec.Body.String())
	}
}

// TestMultiTenant_CancelBooking_ZeroTenantID_Blocked tests that cancellation fails with tenantID=0
// TDD RED PHASE: This test should FAIL - exposes the tenantID==0 bypass bug
func TestMultiTenant_CancelBooking_ZeroTenantID_Blocked(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewBookingHandler(db, cfg)

	// Create user and dog for tenant 1
	userID := testutil.SeedTestUser(t, db, "user@tenant1.com", "Test User", "green")
	dogID := testutil.SeedTestDog(t, db, "Buddy", "Labrador", "green")

	bookingRepo := repository.NewBookingRepository(db)

	// Create booking for tenant 1
	booking := &models.Booking{
		TenantID:       1,
		UserID:         userID,
		DogID:          dogID,
		Date:           testutil.GetFutureDate(14),
		ScheduledTime:  "11:00",
		Status:         "scheduled",
		ApprovalStatus: "approved",
	}
	if err := bookingRepo.Create(booking); err != nil {
		t.Fatalf("Failed to create booking: %v", err)
	}

	reqBody := map[string]interface{}{
		"reason": "Testing zero tenant ID bypass",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/bookings/%d", booking.ID), bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", booking.ID)})
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, 0) // tenantID = 0!
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, middleware.IsAdminKey, true)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CancelBooking(rec, req)

	// BUG: This should fail but likely succeeds due to the bypass
	if rec.Code == http.StatusOK {
		t.Errorf("SECURITY BUG: CancelBooking allowed with tenantID=0 - tenant isolation bypassed!")
	}

	// Verify booking wasn't actually cancelled
	savedBooking, _ := bookingRepo.FindByID(booking.ID)
	if savedBooking != nil && savedBooking.Status == "cancelled" {
		t.Errorf("SECURITY BUG: Booking was actually cancelled with tenantID=0!")
	}
}

// TestMultiTenant_ApproveBooking_ZeroTenantID_Blocked tests approval fails with tenantID=0
// TDD RED PHASE: This test should FAIL
func TestMultiTenant_ApproveBooking_ZeroTenantID_Blocked(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewBookingHandler(db, cfg)

	// Create user and dog for tenant 1
	userID := testutil.SeedTestUser(t, db, "user@tenant1.com", "Test User", "green")
	dogID := testutil.SeedTestDog(t, db, "Buddy", "Labrador", "green")

	bookingRepo := repository.NewBookingRepository(db)

	// Create pending booking for tenant 1
	booking := &models.Booking{
		TenantID:         1,
		UserID:           userID,
		DogID:            dogID,
		Date:             testutil.GetFutureDate(7),
		ScheduledTime:    "09:00",
		Status:           "scheduled",
		RequiresApproval: true,
		ApprovalStatus:   "pending",
	}
	if err := bookingRepo.Create(booking); err != nil {
		t.Fatalf("Failed to create booking: %v", err)
	}

	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/bookings/%d/approve", booking.ID), nil)
	req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", booking.ID)})
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, 0) // tenantID = 0!
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, middleware.IsAdminKey, true)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ApprovePendingBooking(rec, req)

	// BUG: This should fail but likely succeeds
	if rec.Code == http.StatusOK {
		t.Errorf("SECURITY BUG: ApprovePendingBooking allowed with tenantID=0!")
	}

	// Verify booking wasn't actually approved
	savedBooking, _ := bookingRepo.FindByID(booking.ID)
	if savedBooking != nil && savedBooking.ApprovalStatus == "approved" {
		t.Errorf("SECURITY BUG: Booking was actually approved with tenantID=0!")
	}
}

// ====================================================================================
// BUG #1: CROSS-TENANT DOG ACCESS IN CreateBooking - INFORMATION LEAK
// ====================================================================================
// When booking a dog from another tenant, the system returns a color category error
// instead of "Dog not found". This reveals the existence of dogs in other tenants.
// ====================================================================================

// TestMultiTenant_CreateBooking_CrossTenantDog_ReturnsNotFound tests that booking
// a dog from another tenant returns "Dog not found" not "color category error"
// TDD RED PHASE: This test should FAIL until we add tenant check in CreateBooking
func TestMultiTenant_CreateBooking_CrossTenantDog_ReturnsNotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewBookingHandler(db, cfg)

	// Create user in tenant 1
	userID := testutil.SeedTestUser(t, db, "user@tenant1.com", "Test User", "green")

	// Create tenant 2
	now := testutil.NowTime()
	_, err := db.Exec(`INSERT INTO tenants (id, slug, name, status, contact_email, created_at, updated_at)
		VALUES (2, 'tenant2', 'Tenant 2', 'active', 'tenant2@example.com', ?, ?)`, now, now)
	if err != nil {
		t.Fatalf("Failed to create tenant 2: %v", err)
	}

	// Create color for tenant 2
	_, err = db.Exec(`INSERT INTO color_categories (id, tenant_id, name, hex_code, sort_order, created_at, updated_at)
		VALUES (100, 2, 'Green', '#00FF00', 1, ?, ?)`, now, now)
	if err != nil {
		t.Fatalf("Failed to create color for tenant 2: %v", err)
	}

	// Create dog in tenant 2
	_, err = db.Exec(`INSERT INTO dogs (id, tenant_id, name, breed, size, age, color_id, is_available, created_at, updated_at)
		VALUES (100, 2, 'OtherTenantDog', 'Labrador', 'large', 3, 100, 1, ?, ?)`, now, now)
	if err != nil {
		t.Fatalf("Failed to create dog in tenant 2: %v", err)
	}

	// User from tenant 1 tries to book dog from tenant 2
	futureDate := testutil.GetFutureDate(7)
	reqBody := map[string]interface{}{
		"dog_id":         100, // Dog from tenant 2
		"date":           futureDate,
		"scheduled_time": "14:00",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/bookings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, 1) // User is in tenant 1
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, middleware.EmailKey, "user@tenant1.com")
	ctx = context.WithValue(ctx, middleware.IsAdminKey, false)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CreateBooking(rec, req)

	// BUG: Currently returns 403 "Du hast nicht die erforderliche Farbkategorie"
	// Should return 404 "Dog not found" to avoid information leakage
	if rec.Code != http.StatusNotFound {
		t.Errorf("BUG: Expected 404 'Dog not found' for cross-tenant dog, got %d: %s",
			rec.Code, rec.Body.String())
	}

	// Verify the error message is "Dog not found", not color-related
	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err == nil {
		if resp["error"] != "Dog not found" {
			t.Errorf("BUG: Expected error 'Dog not found', got '%s' - information leak!", resp["error"])
		}
	}
}

// TestMultiTenant_CreateBooking_CrossTenantDog_AdminAlsoBlocked tests that even admins
// cannot book dogs from other tenants
// TDD RED PHASE: This test should FAIL
func TestMultiTenant_CreateBooking_CrossTenantDog_AdminAlsoBlocked(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewBookingHandler(db, cfg)

	// Create admin user in tenant 1
	userID := testutil.SeedTestUser(t, db, "admin@tenant1.com", "Admin User", "green")
	// Make them admin
	_, err := db.Exec(`UPDATE users SET is_admin = 1 WHERE id = ?`, userID)
	if err != nil {
		t.Fatalf("Failed to make user admin: %v", err)
	}

	// Create tenant 2
	now := testutil.NowTime()
	_, err = db.Exec(`INSERT INTO tenants (id, slug, name, status, contact_email, created_at, updated_at)
		VALUES (2, 'tenant2', 'Tenant 2', 'active', 'tenant2@example.com', ?, ?)`, now, now)
	if err != nil {
		t.Fatalf("Failed to create tenant 2: %v", err)
	}

	// Create color for tenant 2
	_, err = db.Exec(`INSERT INTO color_categories (id, tenant_id, name, hex_code, sort_order, created_at, updated_at)
		VALUES (100, 2, 'Green', '#00FF00', 1, ?, ?)`, now, now)
	if err != nil {
		t.Fatalf("Failed to create color for tenant 2: %v", err)
	}

	// Create dog in tenant 2
	_, err = db.Exec(`INSERT INTO dogs (id, tenant_id, name, breed, size, age, color_id, is_available, created_at, updated_at)
		VALUES (100, 2, 'OtherTenantDog', 'Labrador', 'large', 3, 100, 1, ?, ?)`, now, now)
	if err != nil {
		t.Fatalf("Failed to create dog in tenant 2: %v", err)
	}

	// Admin from tenant 1 tries to book dog from tenant 2
	futureDate := testutil.GetFutureDate(7)
	reqBody := map[string]interface{}{
		"dog_id":         100, // Dog from tenant 2
		"date":           futureDate,
		"scheduled_time": "14:00",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/bookings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, 1) // Admin is in tenant 1
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, middleware.EmailKey, "admin@tenant1.com")
	ctx = context.WithValue(ctx, middleware.IsAdminKey, true) // Is admin!
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.CreateBooking(rec, req)

	// BUG: Admin bypasses color check but should still get "Dog not found"
	if rec.Code != http.StatusNotFound {
		t.Errorf("BUG: Admin from tenant 1 should not access dogs from tenant 2. Got %d: %s",
			rec.Code, rec.Body.String())
	}
}

// =============================================================================
// USER HANDLER CROSS-TENANT SECURITY TESTS
// =============================================================================

// TestMultiTenant_GetUser_CrossTenantBlocked tests that admins can't read users from other tenants
// TDD RED PHASE: This test should FAIL until we add tenant verification to GetUser
func TestMultiTenant_GetUser_CrossTenantBlocked(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewUserHandler(db, cfg)

	// Create tenant 2
	nowStr := testutil.Now()
	_, err := db.Exec(`INSERT INTO tenants (id, slug, name, status, contact_email, created_at, updated_at)
		VALUES (2, 'tenant2', 'Tenant 2', 'active', 'tenant2@example.com', ?, ?)`, nowStr, nowStr)
	if err != nil {
		t.Fatalf("Failed to create tenant 2: %v", err)
	}

	// Create user in tenant 2
	userRepo := repository.NewUserRepository(db)
	email := "user@tenant2.com"
	hash := "hashedpassword"
	nowTime := time.Now()
	tenant2User := &models.User{
		TenantID:        2,
		FirstName:       "Tenant2",
		LastName:        "User",
		Email:           &email,
		PasswordHash:    &hash,
		IsVerified:      true,
		IsActive:        true,
		TermsAcceptedAt: nowTime,
		LastActivityAt:  nowTime,
	}
	if err := userRepo.Create(tenant2User); err != nil {
		t.Fatalf("Failed to create tenant 2 user: %v", err)
	}

	// Try to access as admin from tenant 1
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/admin/users/%d", tenant2User.ID), nil)
	req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", tenant2User.ID)})
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, 1) // Tenant 1!
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	ctx = context.WithValue(ctx, middleware.EmailKey, "admin@tenant1.com")
	ctx = context.WithValue(ctx, middleware.IsAdminKey, true)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.GetUser(rec, req)

	// BUG: Should return 404, but returns 200 with full user data
	if rec.Code == http.StatusOK {
		t.Error("BUG: GetUser allowed cross-tenant access - admin from tenant 1 accessed tenant 2's user")
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected 404 Not Found, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestMultiTenant_AdminUpdateUser_CrossTenantBlocked tests that admins can't modify users from other tenants
// TDD RED PHASE: This test should FAIL until we add tenant verification to AdminUpdateUser
func TestMultiTenant_AdminUpdateUser_CrossTenantBlocked(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewUserHandler(db, cfg)

	// Create tenant 2
	nowStr := testutil.Now()
	_, err := db.Exec(`INSERT INTO tenants (id, slug, name, status, contact_email, created_at, updated_at)
		VALUES (2, 'tenant2', 'Tenant 2', 'active', 'tenant2@example.com', ?, ?)`, nowStr, nowStr)
	if err != nil {
		t.Fatalf("Failed to create tenant 2: %v", err)
	}

	// Create user in tenant 2
	userRepo := repository.NewUserRepository(db)
	email := "victim@tenant2.com"
	hash := "hashedpassword"
	nowTime := time.Now()
	tenant2User := &models.User{
		TenantID:        2,
		FirstName:       "Original",
		LastName:        "Name",
		Email:           &email,
		PasswordHash:    &hash,
		IsVerified:      true,
		IsActive:        true,
		TermsAcceptedAt: nowTime,
		LastActivityAt:  nowTime,
	}
	if err := userRepo.Create(tenant2User); err != nil {
		t.Fatalf("Failed to create tenant 2 user: %v", err)
	}

	// Try to update as admin from tenant 1
	newFirstName := "HACKED"
	reqBody := map[string]interface{}{
		"first_name": newFirstName,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/admin/users/%d", tenant2User.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", tenant2User.ID)})
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, 1) // Tenant 1!
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	ctx = context.WithValue(ctx, middleware.EmailKey, "admin@tenant1.com")
	ctx = context.WithValue(ctx, middleware.IsAdminKey, true)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.AdminUpdateUser(rec, req)

	// BUG: Should return 404, but returns 200 (success)
	if rec.Code == http.StatusOK {
		t.Error("BUG: AdminUpdateUser allowed cross-tenant modification - admin from tenant 1 modified tenant 2's user")
	}

	// Verify user was NOT modified
	savedUser, _ := userRepo.FindByID(tenant2User.ID)
	if savedUser != nil && savedUser.FirstName == newFirstName {
		t.Error("BUG: Cross-tenant update actually modified the user!")
	}
}

// TestMultiTenant_DeactivateUser_CrossTenantBlocked tests that admins can't deactivate users from other tenants
// TDD RED PHASE: This test should FAIL until we add tenant verification to DeactivateUser
func TestMultiTenant_DeactivateUser_CrossTenantBlocked(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewUserHandler(db, cfg)

	// Create tenant 2
	nowStr := testutil.Now()
	_, err := db.Exec(`INSERT INTO tenants (id, slug, name, status, contact_email, created_at, updated_at)
		VALUES (2, 'tenant2', 'Tenant 2', 'active', 'tenant2@example.com', ?, ?)`, nowStr, nowStr)
	if err != nil {
		t.Fatalf("Failed to create tenant 2: %v", err)
	}

	// Create user in tenant 2
	userRepo := repository.NewUserRepository(db)
	email := "active@tenant2.com"
	hash := "hashedpassword"
	nowTime := time.Now()
	tenant2User := &models.User{
		TenantID:        2,
		FirstName:       "Active",
		LastName:        "User",
		Email:           &email,
		PasswordHash:    &hash,
		IsVerified:      true,
		IsActive:        true,
		TermsAcceptedAt: nowTime,
		LastActivityAt:  nowTime,
	}
	if err := userRepo.Create(tenant2User); err != nil {
		t.Fatalf("Failed to create tenant 2 user: %v", err)
	}

	// Try to deactivate as admin from tenant 1
	reqBody := map[string]interface{}{
		"reason": "Cross-tenant attack test",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", fmt.Sprintf("/api/admin/users/%d/deactivate", tenant2User.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", tenant2User.ID)})
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, 1) // Tenant 1!
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	ctx = context.WithValue(ctx, middleware.EmailKey, "admin@tenant1.com")
	ctx = context.WithValue(ctx, middleware.IsAdminKey, true)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.DeactivateUser(rec, req)

	// BUG: Should return 404, but likely returns 200 (success)
	if rec.Code == http.StatusOK {
		t.Error("BUG: DeactivateUser allowed cross-tenant deactivation - admin from tenant 1 deactivated tenant 2's user")
	}

	// Verify user was NOT deactivated
	savedUser, _ := userRepo.FindByID(tenant2User.ID)
	if savedUser != nil && !savedUser.IsActive {
		t.Error("BUG: Cross-tenant deactivation actually deactivated the user!")
	}
}

// TestMultiTenant_ActivateUser_CrossTenantBlocked tests that admins can't activate users from other tenants
// TDD RED PHASE: This test should FAIL until we add tenant verification to ActivateUser
func TestMultiTenant_ActivateUser_CrossTenantBlocked(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewUserHandler(db, cfg)

	// Create tenant 2
	nowStr := testutil.Now()
	_, err := db.Exec(`INSERT INTO tenants (id, slug, name, status, contact_email, created_at, updated_at)
		VALUES (2, 'tenant2', 'Tenant 2', 'active', 'tenant2@example.com', ?, ?)`, nowStr, nowStr)
	if err != nil {
		t.Fatalf("Failed to create tenant 2: %v", err)
	}

	// Create inactive user in tenant 2
	userRepo := repository.NewUserRepository(db)
	email := "inactive@tenant2.com"
	hash := "hashedpassword"
	nowTime := time.Now()
	tenant2User := &models.User{
		TenantID:        2,
		FirstName:       "Inactive",
		LastName:        "User",
		Email:           &email,
		PasswordHash:    &hash,
		IsVerified:      true,
		IsActive:        false, // Inactive
		TermsAcceptedAt: nowTime,
		LastActivityAt:  nowTime,
	}
	if err := userRepo.Create(tenant2User); err != nil {
		t.Fatalf("Failed to create tenant 2 user: %v", err)
	}

	// Try to activate as admin from tenant 1
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/admin/users/%d/activate", tenant2User.ID), nil)
	req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", tenant2User.ID)})
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, 1) // Tenant 1!
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	ctx = context.WithValue(ctx, middleware.EmailKey, "admin@tenant1.com")
	ctx = context.WithValue(ctx, middleware.IsAdminKey, true)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ActivateUser(rec, req)

	// BUG: Should return 404, but likely returns 200 (success)
	if rec.Code == http.StatusOK {
		t.Error("BUG: ActivateUser allowed cross-tenant activation - admin from tenant 1 activated tenant 2's user")
	}

	// Verify user was NOT activated
	savedUser, _ := userRepo.FindByID(tenant2User.ID)
	if savedUser != nil && savedUser.IsActive {
		t.Error("BUG: Cross-tenant activation actually activated the user!")
	}
}

// TestMultiTenant_AdminDeleteUser_CrossTenantBlocked tests that admins can't delete users from other tenants
// TDD RED PHASE: This test should FAIL until we add tenant verification to AdminDeleteUser
func TestMultiTenant_AdminDeleteUser_CrossTenantBlocked(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewUserHandler(db, cfg)

	// Create tenant 2
	now := testutil.NowTime()
	_, err := db.Exec(`INSERT INTO tenants (id, slug, name, status, contact_email, created_at, updated_at)
		VALUES (2, 'tenant2', 'Tenant 2', 'active', 'tenant2@example.com', ?, ?)`, now, now)
	if err != nil {
		t.Fatalf("Failed to create tenant 2: %v", err)
	}

	// Create user in tenant 2
	userRepo := repository.NewUserRepository(db)
	email := "delete@tenant2.com"
	hash := "hashedpassword"
	tenant2User := &models.User{
		TenantID:        2,
		FirstName:       "Delete",
		LastName:        "Me",
		Email:           &email,
		PasswordHash:    &hash,
		IsVerified:      true,
		IsActive:        true,
		TermsAcceptedAt: time.Now(),
		LastActivityAt:  time.Now(),
	}
	if err := userRepo.Create(tenant2User); err != nil {
		t.Fatalf("Failed to create tenant 2 user: %v", err)
	}

	// Try to delete as admin from tenant 1
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/admin/users/%d", tenant2User.ID), nil)
	req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", tenant2User.ID)})
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, 1) // Tenant 1!
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	ctx = context.WithValue(ctx, middleware.EmailKey, "admin@tenant1.com")
	ctx = context.WithValue(ctx, middleware.IsAdminKey, true)
	ctx = context.WithValue(ctx, middleware.IsSuperAdminKey, true) // Even super admin
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.AdminDeleteUser(rec, req)

	// BUG: Should return 404, but may return 200 (success) or leak info
	if rec.Code == http.StatusOK {
		t.Error("BUG: AdminDeleteUser allowed cross-tenant deletion - admin from tenant 1 deleted tenant 2's user")
	}

	// Verify user still exists and is not deleted
	savedUser, _ := userRepo.FindByID(tenant2User.ID)
	if savedUser == nil || savedUser.IsDeleted {
		t.Error("BUG: Cross-tenant deletion actually deleted the user!")
	}
}

// TestMultiTenant_PromoteToAdmin_CrossTenantBlocked tests that super admins can't promote users from other tenants
// TDD RED PHASE: This test should FAIL until we add tenant verification to PromoteToAdmin
func TestMultiTenant_PromoteToAdmin_CrossTenantBlocked(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewUserHandler(db, cfg)

	// Create tenant 2
	now := testutil.NowTime()
	_, err := db.Exec(`INSERT INTO tenants (id, slug, name, status, contact_email, created_at, updated_at)
		VALUES (2, 'tenant2', 'Tenant 2', 'active', 'tenant2@example.com', ?, ?)`, now, now)
	if err != nil {
		t.Fatalf("Failed to create tenant 2: %v", err)
	}

	// Create regular user in tenant 2
	userRepo := repository.NewUserRepository(db)
	email := "regular@tenant2.com"
	hash := "hashedpassword"
	tenant2User := &models.User{
		TenantID:        2,
		FirstName:       "Regular",
		LastName:        "User",
		Email:           &email,
		PasswordHash:    &hash,
		IsVerified:      true,
		IsActive:        true,
		IsAdmin:         false, // Not an admin
		TermsAcceptedAt: time.Now(),
		LastActivityAt:  time.Now(),
	}
	if err := userRepo.Create(tenant2User); err != nil {
		t.Fatalf("Failed to create tenant 2 user: %v", err)
	}

	// Try to promote as super admin from tenant 1
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/admin/users/%d/promote", tenant2User.ID), nil)
	req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", tenant2User.ID)})
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, 1) // Tenant 1!
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	ctx = context.WithValue(ctx, middleware.EmailKey, "superadmin@tenant1.com")
	ctx = context.WithValue(ctx, middleware.IsAdminKey, true)
	ctx = context.WithValue(ctx, middleware.IsSuperAdminKey, true)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.PromoteToAdmin(rec, req)

	// BUG: Should return 404, but may return 200 (success)
	if rec.Code == http.StatusOK {
		t.Error("BUG: PromoteToAdmin allowed cross-tenant promotion - super admin from tenant 1 promoted tenant 2's user")
	}

	// Verify user was NOT promoted
	savedUser, _ := userRepo.FindByID(tenant2User.ID)
	if savedUser != nil && savedUser.IsAdmin {
		t.Error("BUG: Cross-tenant promotion actually promoted the user!")
	}
}

// TestMultiTenant_DemoteAdmin_CrossTenantBlocked tests that super admins can't demote admins from other tenants
// TDD RED PHASE: This test should FAIL until we add tenant verification to DemoteAdmin
func TestMultiTenant_DemoteAdmin_CrossTenantBlocked(t *testing.T) {
	db := testutil.SetupTestDB(t)
	cfg := &config.Config{
		JWTSecret:          "test-secret",
		JWTExpirationHours: 24,
	}
	handler := NewUserHandler(db, cfg)

	// Create tenant 2
	now := testutil.NowTime()
	_, err := db.Exec(`INSERT INTO tenants (id, slug, name, status, contact_email, created_at, updated_at)
		VALUES (2, 'tenant2', 'Tenant 2', 'active', 'tenant2@example.com', ?, ?)`, now, now)
	if err != nil {
		t.Fatalf("Failed to create tenant 2: %v", err)
	}

	// Create admin user in tenant 2
	userRepo := repository.NewUserRepository(db)
	email := "admin@tenant2.com"
	hash := "hashedpassword"
	tenant2Admin := &models.User{
		TenantID:        2,
		FirstName:       "Admin",
		LastName:        "User",
		Email:           &email,
		PasswordHash:    &hash,
		IsVerified:      true,
		IsActive:        true,
		IsAdmin:         true, // Is an admin
		TermsAcceptedAt: time.Now(),
		LastActivityAt:  time.Now(),
	}
	if err := userRepo.Create(tenant2Admin); err != nil {
		t.Fatalf("Failed to create tenant 2 admin: %v", err)
	}

	// Try to demote as super admin from tenant 1
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/admin/users/%d/demote", tenant2Admin.ID), nil)
	req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", tenant2Admin.ID)})
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, 1) // Tenant 1!
	ctx = context.WithValue(ctx, middleware.UserIDKey, 1)
	ctx = context.WithValue(ctx, middleware.EmailKey, "superadmin@tenant1.com")
	ctx = context.WithValue(ctx, middleware.IsAdminKey, true)
	ctx = context.WithValue(ctx, middleware.IsSuperAdminKey, true)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.DemoteAdmin(rec, req)

	// BUG: Should return 404, but may return 200 (success)
	if rec.Code == http.StatusOK {
		t.Error("BUG: DemoteAdmin allowed cross-tenant demotion - super admin from tenant 1 demoted tenant 2's admin")
	}

	// Verify admin was NOT demoted
	savedUser, _ := userRepo.FindByID(tenant2Admin.ID)
	if savedUser != nil && !savedUser.IsAdmin {
		t.Error("BUG: Cross-tenant demotion actually demoted the admin!")
	}
}
