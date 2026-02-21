package repository

import (
	"testing"

	"github.com/tranmh/gassigeher/internal/models"
	_ "modernc.org/sqlite"
)

func TestRecurringBookingRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRecurringBookingRepository(db)

	dayOfWeek := 1 // Monday
	series := &models.RecurringBookingSeries{
		TenantID:       0,
		UserID:         1,
		DogID:          1,
		RecurrenceType: "weekly",
		DayOfWeek:      &dayOfWeek,
		ScheduledTime:  "09:00",
		StartDate:      "2026-03-02",
		EndDate:        "2026-03-30",
	}

	err := repo.Create(series)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if series.ID == 0 {
		t.Error("Expected non-zero ID after create")
	}
	if series.Status != "active" {
		t.Errorf("Expected status 'active', got %q", series.Status)
	}
	if series.CreatedAt.IsZero() {
		t.Error("Expected non-zero CreatedAt")
	}
}

func TestRecurringBookingRepository_Create_Interval(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRecurringBookingRepository(db)

	intervalDays := 14
	series := &models.RecurringBookingSeries{
		TenantID:       0,
		UserID:         1,
		DogID:          2,
		RecurrenceType: "interval",
		IntervalDays:   &intervalDays,
		ScheduledTime:  "14:00",
		StartDate:      "2026-03-01",
		EndDate:        "2026-04-26",
	}

	err := repo.Create(series)
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if series.ID == 0 {
		t.Error("Expected non-zero ID after create")
	}
}

// Bug #2: Tests now use FindByIDAndTenant (FindByID without tenant removed for tenant safety)
func TestRecurringBookingRepository_CreateAndFindByIDAndTenant(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRecurringBookingRepository(db)

	dayOfWeek := 3 // Wednesday
	series := &models.RecurringBookingSeries{
		TenantID:       0,
		UserID:         1,
		DogID:          1,
		RecurrenceType: "weekly",
		DayOfWeek:      &dayOfWeek,
		ScheduledTime:  "10:30",
		StartDate:      "2026-04-01",
		EndDate:        "2026-05-01",
	}

	if err := repo.Create(series); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	found, err := repo.FindByIDAndTenant(series.ID, 0)
	if err != nil {
		t.Fatalf("FindByIDAndTenant() error: %v", err)
	}
	if found == nil {
		t.Fatal("FindByIDAndTenant() returned nil")
	}

	if found.UserID != 1 {
		t.Errorf("Expected UserID 1, got %d", found.UserID)
	}
	if found.DogID != 1 {
		t.Errorf("Expected DogID 1, got %d", found.DogID)
	}
	if found.RecurrenceType != "weekly" {
		t.Errorf("Expected recurrence_type 'weekly', got %q", found.RecurrenceType)
	}
	if found.DayOfWeek == nil || *found.DayOfWeek != 3 {
		t.Errorf("Expected day_of_week 3, got %v", found.DayOfWeek)
	}
	if found.ScheduledTime != "10:30" {
		t.Errorf("Expected scheduled_time '10:30', got %q", found.ScheduledTime)
	}
	if found.Status != "active" {
		t.Errorf("Expected status 'active', got %q", found.Status)
	}
	if found.StartDate != "2026-04-01" {
		t.Errorf("Expected start_date '2026-04-01', got %q", found.StartDate)
	}
	if found.EndDate != "2026-05-01" {
		t.Errorf("Expected end_date '2026-05-01', got %q", found.EndDate)
	}
}

func TestRecurringBookingRepository_FindByIDAndTenant_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRecurringBookingRepository(db)

	found, err := repo.FindByIDAndTenant(99999, 0)
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got: %v", err)
	}
	if found != nil {
		t.Error("Expected nil for non-existent ID")
	}
}

func TestRecurringBookingRepository_FindByIDAndTenant(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRecurringBookingRepository(db)

	dayOfWeek := 1
	series := &models.RecurringBookingSeries{
		TenantID:       0,
		UserID:         1,
		DogID:          1,
		RecurrenceType: "weekly",
		DayOfWeek:      &dayOfWeek,
		ScheduledTime:  "09:00",
		StartDate:      "2026-03-01",
		EndDate:        "2026-04-01",
	}

	if err := repo.Create(series); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Should find with correct tenant
	found, err := repo.FindByIDAndTenant(series.ID, 0)
	if err != nil {
		t.Fatalf("FindByIDAndTenant() error: %v", err)
	}
	if found == nil {
		t.Fatal("FindByIDAndTenant() returned nil for correct tenant")
	}

	// Should error with wrong tenant (returns ErrNotFound to prevent ID enumeration)
	_, err = repo.FindByIDAndTenant(series.ID, 999)
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound for wrong tenant, got: %v", err)
	}

	// Should error for non-existent ID
	_, err = repo.FindByIDAndTenant(99999, 0)
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound for non-existent ID, got: %v", err)
	}
}

func TestRecurringBookingRepository_FindByUserID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRecurringBookingRepository(db)

	// Create series for user 1
	dayOfWeek := 1
	for i := 0; i < 3; i++ {
		series := &models.RecurringBookingSeries{
			TenantID:       0,
			UserID:         1,
			DogID:          i + 1,
			RecurrenceType: "weekly",
			DayOfWeek:      &dayOfWeek,
			ScheduledTime:  "09:00",
			StartDate:      "2026-03-01",
			EndDate:        "2026-04-01",
		}
		if err := repo.Create(series); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	// Create series for user 2
	series2 := &models.RecurringBookingSeries{
		TenantID:       0,
		UserID:         2,
		DogID:          1,
		RecurrenceType: "weekly",
		DayOfWeek:      &dayOfWeek,
		ScheduledTime:  "09:00",
		StartDate:      "2026-03-01",
		EndDate:        "2026-04-01",
	}
	if err := repo.Create(series2); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Find for user 1
	seriesList, err := repo.FindByUserID(1, 0)
	if err != nil {
		t.Fatalf("FindByUserID() error: %v", err)
	}
	if len(seriesList) != 3 {
		t.Errorf("Expected 3 series for user 1, got %d", len(seriesList))
	}

	// Check that dog data is joined
	for _, s := range seriesList {
		if s.Dog == nil {
			t.Error("Expected Dog to be joined")
		}
	}

	// Find for user 2
	seriesList, err = repo.FindByUserID(2, 0)
	if err != nil {
		t.Fatalf("FindByUserID() error: %v", err)
	}
	if len(seriesList) != 1 {
		t.Errorf("Expected 1 series for user 2, got %d", len(seriesList))
	}

	// Find for user with no series
	seriesList, err = repo.FindByUserID(3, 0)
	if err != nil {
		t.Fatalf("FindByUserID() error: %v", err)
	}
	if len(seriesList) != 0 {
		t.Errorf("Expected 0 series for user 3, got %d", len(seriesList))
	}
}

func TestRecurringBookingRepository_Cancel(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRecurringBookingRepository(db)

	dayOfWeek := 2
	series := &models.RecurringBookingSeries{
		TenantID:       0,
		UserID:         1,
		DogID:          1,
		RecurrenceType: "weekly",
		DayOfWeek:      &dayOfWeek,
		ScheduledTime:  "09:00",
		StartDate:      "2026-03-01",
		EndDate:        "2026-04-01",
	}

	if err := repo.Create(series); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Cancel the series
	err := repo.Cancel(series.ID, 0)
	if err != nil {
		t.Fatalf("Cancel() error: %v", err)
	}

	// Verify status changed
	found, err := repo.FindByIDAndTenant(series.ID, 0)
	if err != nil {
		t.Fatalf("FindByIDAndTenant() error: %v", err)
	}
	if found.Status != "cancelled" {
		t.Errorf("Expected status 'cancelled', got %q", found.Status)
	}
}

func TestRecurringBookingRepository_Cancel_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRecurringBookingRepository(db)

	err := repo.Cancel(99999, 0)
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got: %v", err)
	}
}

func TestRecurringBookingRepository_Cancel_WrongTenant(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRecurringBookingRepository(db)

	dayOfWeek := 1
	series := &models.RecurringBookingSeries{
		TenantID:       0,
		UserID:         1,
		DogID:          1,
		RecurrenceType: "weekly",
		DayOfWeek:      &dayOfWeek,
		ScheduledTime:  "09:00",
		StartDate:      "2026-03-01",
		EndDate:        "2026-04-01",
	}

	if err := repo.Create(series); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Cancel with wrong tenant should fail
	err := repo.Cancel(series.ID, 999)
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound for wrong tenant, got: %v", err)
	}

	// Verify series is still active
	found, err := repo.FindByIDAndTenant(series.ID, 0)
	if err != nil {
		t.Fatalf("FindByIDAndTenant() error: %v", err)
	}
	if found.Status != "active" {
		t.Errorf("Expected status 'active' (unchanged), got %q", found.Status)
	}
}

func TestRecurringBookingRepository_MarkCompleted(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRecurringBookingRepository(db)

	dayOfWeek := 4
	series := &models.RecurringBookingSeries{
		TenantID:       0,
		UserID:         1,
		DogID:          1,
		RecurrenceType: "weekly",
		DayOfWeek:      &dayOfWeek,
		ScheduledTime:  "09:00",
		StartDate:      "2026-03-01",
		EndDate:        "2026-04-01",
	}

	if err := repo.Create(series); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	err := repo.MarkCompleted(series.ID, series.TenantID)
	if err != nil {
		t.Fatalf("MarkCompleted() error: %v", err)
	}

	found, err := repo.FindByIDAndTenant(series.ID, 0)
	if err != nil {
		t.Fatalf("FindByIDAndTenant() error: %v", err)
	}
	if found.Status != "completed" {
		t.Errorf("Expected status 'completed', got %q", found.Status)
	}
}

func TestRecurringBookingRepository_CountActiveByUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRecurringBookingRepository(db)

	dayOfWeek := 1

	// Create 2 active series for user 1
	for i := 0; i < 2; i++ {
		series := &models.RecurringBookingSeries{
			TenantID:       0,
			UserID:         1,
			DogID:          i + 1,
			RecurrenceType: "weekly",
			DayOfWeek:      &dayOfWeek,
			ScheduledTime:  "09:00",
			StartDate:      "2026-03-01",
			EndDate:        "2026-04-01",
		}
		if err := repo.Create(series); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	// Create 1 cancelled series for user 1
	cancelledSeries := &models.RecurringBookingSeries{
		TenantID:       0,
		UserID:         1,
		DogID:          3,
		RecurrenceType: "weekly",
		DayOfWeek:      &dayOfWeek,
		ScheduledTime:  "09:00",
		StartDate:      "2026-03-01",
		EndDate:        "2026-04-01",
	}
	if err := repo.Create(cancelledSeries); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := repo.Cancel(cancelledSeries.ID, 0); err != nil {
		t.Fatalf("Cancel() error: %v", err)
	}

	// Count should be 2 (only active)
	count, err := repo.CountActiveByUser(1, 0)
	if err != nil {
		t.Fatalf("CountActiveByUser() error: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 active series, got %d", count)
	}

	// User 2 should have 0
	count, err = repo.CountActiveByUser(2, 0)
	if err != nil {
		t.Fatalf("CountActiveByUser() error: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 active series for user 2, got %d", count)
	}
}

func TestRecurringBookingRepository_FindActiveByDog(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRecurringBookingRepository(db)

	dayOfWeek := 1

	// Create 2 active series for dog 1
	for i := 0; i < 2; i++ {
		series := &models.RecurringBookingSeries{
			TenantID:       0,
			UserID:         i + 1,
			DogID:          1,
			RecurrenceType: "weekly",
			DayOfWeek:      &dayOfWeek,
			ScheduledTime:  "09:00",
			StartDate:      "2026-03-01",
			EndDate:        "2026-04-01",
		}
		if err := repo.Create(series); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	// Create 1 cancelled series for dog 1
	cancelledSeries := &models.RecurringBookingSeries{
		TenantID:       0,
		UserID:         3,
		DogID:          1,
		RecurrenceType: "weekly",
		DayOfWeek:      &dayOfWeek,
		ScheduledTime:  "09:00",
		StartDate:      "2026-03-01",
		EndDate:        "2026-04-01",
	}
	if err := repo.Create(cancelledSeries); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := repo.Cancel(cancelledSeries.ID, 0); err != nil {
		t.Fatalf("Cancel() error: %v", err)
	}

	// Should find 2 active series for dog 1
	seriesList, err := repo.FindActiveByDog(1, 0)
	if err != nil {
		t.Fatalf("FindActiveByDog() error: %v", err)
	}
	if len(seriesList) != 2 {
		t.Errorf("Expected 2 active series for dog 1, got %d", len(seriesList))
	}

	// Dog 2 should have 0
	seriesList, err = repo.FindActiveByDog(2, 0)
	if err != nil {
		t.Fatalf("FindActiveByDog() error: %v", err)
	}
	if len(seriesList) != 0 {
		t.Errorf("Expected 0 active series for dog 2, got %d", len(seriesList))
	}
}

func TestRecurringBookingRepository_FindAll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRecurringBookingRepository(db)

	dayOfWeek := 1

	// Create mix of series
	for i := 0; i < 3; i++ {
		series := &models.RecurringBookingSeries{
			TenantID:       0,
			UserID:         i + 1,
			DogID:          i + 1,
			RecurrenceType: "weekly",
			DayOfWeek:      &dayOfWeek,
			ScheduledTime:  "09:00",
			StartDate:      "2026-03-01",
			EndDate:        "2026-04-01",
		}
		if err := repo.Create(series); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
		// Cancel the third one
		if i == 2 {
			if err := repo.Cancel(series.ID, 0); err != nil {
				t.Fatalf("Cancel() error: %v", err)
			}
		}
	}

	// Find all with no filter — tenantID is now a mandatory parameter
	all, err := repo.FindAll(0, nil)
	if err != nil {
		t.Fatalf("FindAll() error: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("Expected 3 total series, got %d", len(all))
	}

	// Filter by status = active
	activeStatus := "active"
	active, err := repo.FindAll(0, &models.RecurringBookingFilterRequest{
		Status: &activeStatus,
	})
	if err != nil {
		t.Fatalf("FindAll(active) error: %v", err)
	}
	if len(active) != 2 {
		t.Errorf("Expected 2 active series, got %d", len(active))
	}

	// Filter by user_id
	userID := 1
	byUser, err := repo.FindAll(0, &models.RecurringBookingFilterRequest{
		UserID: &userID,
	})
	if err != nil {
		t.Fatalf("FindAll(user_id=1) error: %v", err)
	}
	if len(byUser) != 1 {
		t.Errorf("Expected 1 series for user 1, got %d", len(byUser))
	}

	// Filter by dog_id
	dogID := 2
	byDog, err := repo.FindAll(0, &models.RecurringBookingFilterRequest{
		DogID: &dogID,
	})
	if err != nil {
		t.Fatalf("FindAll(dog_id=2) error: %v", err)
	}
	if len(byDog) != 1 {
		t.Errorf("Expected 1 series for dog 2, got %d", len(byDog))
	}
}

func TestBookingRepository_CancelFutureByRecurrenceID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bookingRepo := NewBookingRepository(db)
	recurringRepo := NewRecurringBookingRepository(db)

	// Create a series
	dayOfWeek := 1
	series := &models.RecurringBookingSeries{
		TenantID:       0,
		UserID:         1,
		DogID:          1,
		RecurrenceType: "weekly",
		DayOfWeek:      &dayOfWeek,
		ScheduledTime:  "09:00",
		StartDate:      "2026-03-01",
		EndDate:        "2026-04-01",
	}
	if err := recurringRepo.Create(series); err != nil {
		t.Fatalf("Create series error: %v", err)
	}

	// Create bookings: some in future, some completed
	futureDate := "2099-12-01"
	pastDate := "2020-01-01"

	futureBooking := &models.Booking{
		UserID:        1,
		DogID:         1,
		Date:          futureDate,
		ScheduledTime: "09:00",
		RecurrenceID:  &series.ID,
	}
	if err := bookingRepo.Create(futureBooking); err != nil {
		t.Fatalf("Create future booking error: %v", err)
	}

	pastBooking := &models.Booking{
		UserID:        1,
		DogID:         1,
		Date:          pastDate,
		ScheduledTime: "09:00",
		Status:        "completed",
		RecurrenceID:  &series.ID,
	}
	if err := bookingRepo.Create(pastBooking); err != nil {
		t.Fatalf("Create past booking error: %v", err)
	}

	// Cancel future bookings
	count, err := bookingRepo.CancelFutureByRecurrenceID(series.ID, 0, "Test cancellation")
	if err != nil {
		t.Fatalf("CancelFutureByRecurrenceID() error: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 cancelled booking, got %d", count)
	}

	// Verify future booking is cancelled
	bookings, err := bookingRepo.FindByRecurrenceID(series.ID, 0)
	if err != nil {
		t.Fatalf("FindByRecurrenceID() error: %v", err)
	}

	for _, b := range bookings {
		if b.Date == futureDate && b.Status != "cancelled" {
			t.Errorf("Expected future booking to be cancelled, got status %q", b.Status)
		}
		if b.Date == pastDate && b.Status != "completed" {
			t.Errorf("Expected past booking to remain completed, got status %q", b.Status)
		}
	}
}

func TestBookingRepository_FindByRecurrenceID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bookingRepo := NewBookingRepository(db)
	recurringRepo := NewRecurringBookingRepository(db)

	// Create a series
	dayOfWeek := 1
	series := &models.RecurringBookingSeries{
		TenantID:       0,
		UserID:         1,
		DogID:          1,
		RecurrenceType: "weekly",
		DayOfWeek:      &dayOfWeek,
		ScheduledTime:  "09:00",
		StartDate:      "2026-03-01",
		EndDate:        "2026-04-01",
	}
	if err := recurringRepo.Create(series); err != nil {
		t.Fatalf("Create series error: %v", err)
	}

	// Create 3 bookings for this series
	dates := []string{"2099-03-02", "2099-03-09", "2099-03-16"}
	for _, date := range dates {
		booking := &models.Booking{
			UserID:        1,
			DogID:         1,
			Date:          date,
			ScheduledTime: "09:00",
			RecurrenceID:  &series.ID,
		}
		if err := bookingRepo.Create(booking); err != nil {
			t.Fatalf("Create booking error: %v", err)
		}
	}

	// Create a booking without recurrence (should not be found)
	booking := &models.Booking{
		UserID:        1,
		DogID:         2,
		Date:          "2099-03-02",
		ScheduledTime: "09:00",
	}
	if err := bookingRepo.Create(booking); err != nil {
		t.Fatalf("Create booking error: %v", err)
	}

	// Find by recurrence ID
	bookings, err := bookingRepo.FindByRecurrenceID(series.ID, 0)
	if err != nil {
		t.Fatalf("FindByRecurrenceID() error: %v", err)
	}

	if len(bookings) != 3 {
		t.Errorf("Expected 3 bookings, got %d", len(bookings))
	}
}

func TestBookingRepository_ApproveByRecurrenceID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bookingRepo := NewBookingRepository(db)
	recurringRepo := NewRecurringBookingRepository(db)

	// Create a series
	dayOfWeek := 1
	series := &models.RecurringBookingSeries{
		TenantID:       0,
		UserID:         1,
		DogID:          1,
		RecurrenceType: "weekly",
		DayOfWeek:      &dayOfWeek,
		ScheduledTime:  "09:00",
		StartDate:      "2026-03-01",
		EndDate:        "2026-04-01",
	}
	if err := recurringRepo.Create(series); err != nil {
		t.Fatalf("Create series error: %v", err)
	}

	// Create bookings with pending approval
	dates := []string{"2099-04-06", "2099-04-13"}
	for _, date := range dates {
		booking := &models.Booking{
			UserID:           1,
			DogID:            1,
			Date:             date,
			ScheduledTime:    "09:00",
			RecurrenceID:     &series.ID,
			RequiresApproval: true,
			ApprovalStatus:   "pending",
		}
		if err := bookingRepo.Create(booking); err != nil {
			t.Fatalf("Create booking error: %v", err)
		}
	}

	// Approve all
	adminID := 2
	count, err := bookingRepo.ApproveByRecurrenceID(series.ID, 0, adminID)
	if err != nil {
		t.Fatalf("ApproveByRecurrenceID() error: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 approved bookings, got %d", count)
	}

	// Verify
	bookings, err := bookingRepo.FindByRecurrenceID(series.ID, 0)
	if err != nil {
		t.Fatalf("FindByRecurrenceID() error: %v", err)
	}
	for _, b := range bookings {
		if b.ApprovalStatus != "approved" {
			t.Errorf("Expected approval_status 'approved', got %q", b.ApprovalStatus)
		}
	}
}

func TestBookingRepository_RejectByRecurrenceID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bookingRepo := NewBookingRepository(db)
	recurringRepo := NewRecurringBookingRepository(db)

	// Create a series
	dayOfWeek := 1
	series := &models.RecurringBookingSeries{
		TenantID:       0,
		UserID:         1,
		DogID:          1,
		RecurrenceType: "weekly",
		DayOfWeek:      &dayOfWeek,
		ScheduledTime:  "09:00",
		StartDate:      "2026-03-01",
		EndDate:        "2026-04-01",
	}
	if err := recurringRepo.Create(series); err != nil {
		t.Fatalf("Create series error: %v", err)
	}

	// Create bookings with pending approval
	dates := []string{"2099-05-04", "2099-05-11"}
	for _, date := range dates {
		booking := &models.Booking{
			UserID:           1,
			DogID:            1,
			Date:             date,
			ScheduledTime:    "09:00",
			RecurrenceID:     &series.ID,
			RequiresApproval: true,
			ApprovalStatus:   "pending",
		}
		if err := bookingRepo.Create(booking); err != nil {
			t.Fatalf("Create booking error: %v", err)
		}
	}

	// Reject all
	adminID := 2
	count, err := bookingRepo.RejectByRecurrenceID(series.ID, 0, adminID, "Not allowed")
	if err != nil {
		t.Fatalf("RejectByRecurrenceID() error: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 rejected bookings, got %d", count)
	}

	// Verify
	bookings, err := bookingRepo.FindByRecurrenceID(series.ID, 0)
	if err != nil {
		t.Fatalf("FindByRecurrenceID() error: %v", err)
	}
	for _, b := range bookings {
		if b.ApprovalStatus != "rejected" {
			t.Errorf("Expected approval_status 'rejected', got %q", b.ApprovalStatus)
		}
		if b.Status != "cancelled" {
			t.Errorf("Expected status 'cancelled', got %q", b.Status)
		}
	}
}

// Bug #2: FindByID without tenant_id should not exist — use FindByIDAndTenant instead
// This test verifies that FindByID is no longer exposed (compile-time check via test usage)
// and that FindByIDAndTenant enforces tenant isolation
func TestRecurringBookingRepository_FindByIDAndTenant_TenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRecurringBookingRepository(db)

	dayOfWeek := 1
	series := &models.RecurringBookingSeries{
		TenantID:       0,
		UserID:         1,
		DogID:          1,
		RecurrenceType: "weekly",
		DayOfWeek:      &dayOfWeek,
		ScheduledTime:  "09:00",
		StartDate:      "2026-03-01",
		EndDate:        "2026-04-01",
	}
	if err := repo.Create(series); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Should find with correct tenant
	found, err := repo.FindByIDAndTenant(series.ID, 0)
	if err != nil {
		t.Fatalf("FindByIDAndTenant(correct tenant) error: %v", err)
	}
	if found == nil {
		t.Fatal("Expected to find series with correct tenant_id=0")
	}

	// Should NOT find with wrong tenant — tenant isolation enforced
	found, err = repo.FindByIDAndTenant(series.ID, 999)
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound for wrong tenant, got: %v", err)
	}
	if found != nil {
		t.Error("Expected nil for wrong tenant")
	}
}

// Bug #3: FindAll must require tenant_id — calling with nil filter should error or return empty
func TestRecurringBookingRepository_FindAll_RequiresTenantID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRecurringBookingRepository(db)

	dayOfWeek := 1
	series := &models.RecurringBookingSeries{
		TenantID:       0,
		UserID:         1,
		DogID:          1,
		RecurrenceType: "weekly",
		DayOfWeek:      &dayOfWeek,
		ScheduledTime:  "09:00",
		StartDate:      "2026-03-01",
		EndDate:        "2026-04-01",
	}
	if err := repo.Create(series); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Calling FindAll with mandatory tenantID parameter should work
	tenantID := 0
	results, err := repo.FindAll(tenantID, &models.RecurringBookingFilterRequest{
		Status: nil,
	})
	if err != nil {
		t.Fatalf("FindAll(tenantID=0) error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for tenant 0, got %d", len(results))
	}

	// Wrong tenant should return 0 results
	results, err = repo.FindAll(999, nil)
	if err != nil {
		t.Fatalf("FindAll(tenantID=999) error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results for tenant 999, got %d", len(results))
	}
}
