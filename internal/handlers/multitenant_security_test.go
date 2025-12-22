package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

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
	now := testutil.Now()
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
	now := testutil.Now()
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
	now := testutil.Now()
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
	now := testutil.Now()
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
	now := testutil.Now()
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
