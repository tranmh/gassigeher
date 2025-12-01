package repository

import (
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/testutil"
	_ "github.com/mattn/go-sqlite3"
)

// ========================================
// Phase 7: Performance Testing
// Test 7.3: Concurrent Request Handling
// ========================================

// Test 7.3.1: Concurrent Booking Creation
// Purpose: Verify system handles concurrent bookings correctly
func TestConcurrentBookingCreation(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	bookingRepo := NewBookingRepository(db)

	// Create test users and dogs
	for i := 1; i <= 50; i++ {
		_, err := db.Exec(`
			INSERT INTO users (id, name, email, password_hash, experience_level, terms_accepted_at)
			VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		`, i, fmt.Sprintf("User %d", i), fmt.Sprintf("user%d@test.com", i), "hash", "green")
		if err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}
	}

	_, err := db.Exec(`
		INSERT INTO dogs (id, name, category, age, breed, is_available)
		VALUES (1, 'Test Dog', 'green', 3, 'Mixed', 1)
	`)
	if err != nil {
		t.Fatalf("Failed to create test dog: %v", err)
	}

	// 50 users try to book the same dog/date/time/walktype simultaneously
	var wg sync.WaitGroup
	errors := make([]error, 50)
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			booking := &models.Booking{
				UserID:         index + 1,
				DogID:          1,
				Date:           "2025-01-27",
				ScheduledTime:  "15:00",
				Status:         "scheduled",
				ApprovalStatus: "approved",
			}
			err := bookingRepo.Create(booking)
			errors[index] = err
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// Verify: Only one booking created due to UNIQUE constraint
	// (dog_id, date, walk_type) must be unique
	if successCount != 1 {
		t.Errorf("Expected exactly 1 successful booking, got %d", successCount)
	}

	// Verify database has exactly 1 booking for this combination
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM bookings
		WHERE dog_id = 1 AND date = '2025-01-27' AND scheduled_time = '15:00'
	`).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query bookings: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 booking in database, got %d", count)
	}

	// Count how many errors were UNIQUE constraint violations
	constraintErrors := 0
	for _, err := range errors {
		if err != nil && (err.Error() == "UNIQUE constraint failed: bookings.dog_id, bookings.date, bookings.walk_type" ||
		   err.Error() == "booking already exists for this time slot") {
			constraintErrors++
		}
	}

	t.Logf("Concurrent booking test: %d attempts, %d success, %d constraint errors",
		50, successCount, constraintErrors)
}

// Test 7.3.1: Concurrent Booking Creation - Different Time Slots
// Purpose: Verify multiple bookings can be created concurrently for different times
func TestConcurrentBookingCreation_DifferentTimeSlots(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	bookingRepo := NewBookingRepository(db)

	// Create test users and dogs
	for i := 1; i <= 20; i++ {
		_, err := db.Exec(`
			INSERT INTO users (id, name, email, password_hash, experience_level, terms_accepted_at)
			VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		`, i, fmt.Sprintf("User %d", i), fmt.Sprintf("user%d@test.com", i), "hash", "green")
		if err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}
	}

	_, err := db.Exec(`
		INSERT INTO dogs (id, name, category, age, breed, is_available)
		VALUES (1, 'Test Dog', 'green', 3, 'Mixed', 1)
	`)
	if err != nil {
		t.Fatalf("Failed to create test dog: %v", err)
	}

	// 20 users book different time slots simultaneously
	timeSlots := []string{
		"09:00", "09:15", "09:30", "09:45",
		"10:00", "10:15", "10:30", "10:45",
		"14:00", "14:15", "14:30", "14:45",
		"15:00", "15:15", "15:30", "15:45",
		"18:00", "18:15", "18:30", "18:45",
	}

	var wg sync.WaitGroup
	errors := make([]error, 20)
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			booking := &models.Booking{
				UserID:         index + 1,
				DogID:          1,
				Date:           "2025-01-27",
				ScheduledTime:  timeSlots[index],
				Status:         "scheduled",
				ApprovalStatus: "approved",
			}
			err := bookingRepo.Create(booking)
			errors[index] = err
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// Verify: All 20 bookings created since each has a unique scheduled_time
	// The constraint is on (dog_id, date, scheduled_time), so different times are allowed
	if successCount != 20 {
		t.Errorf("Expected 20 successful bookings (all different times), got %d", successCount)
		for i, err := range errors {
			if err != nil {
				t.Logf("  Error at index %d: %v", i, err)
			}
		}
	}

	// Verify database has exactly 20 bookings
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM bookings WHERE dog_id = 1`).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query bookings: %v", err)
	}

	if count != 20 {
		t.Errorf("Expected 20 bookings in database, got %d", count)
	}

	t.Logf("Concurrent different slots test: %d attempts, %d success", 20, successCount)
}

// Test 7.3.1: Concurrent Approval Updates
// Purpose: Verify concurrent approval operations work correctly
func TestConcurrentApprovalUpdates(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	bookingRepo := NewBookingRepository(db)

	// Create test user, dog, and admin
	_, err := db.Exec(`
		INSERT INTO users (id, name, email, password_hash, experience_level, is_admin, is_verified, is_active, terms_accepted_at, last_activity_at, created_at)
		VALUES (1, 'User', 'user@test.com', 'hash', 'green', 0, 1, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		       (2, 'Admin', 'admin@test.com', 'hash', 'green', 1, 1, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`)
	if err != nil {
		t.Fatalf("Failed to create test users: %v", err)
	}

	// Create 10 dogs for testing
	for i := 1; i <= 10; i++ {
		_, _ = db.Exec(`
			INSERT INTO dogs (id, name, category, age, breed, is_available)
			VALUES (?, ?, 'green', 3, 'Mixed', 1)
		`, i, fmt.Sprintf("Test Dog %d", i))
	}

	// Create 10 pending bookings (use different dogs to satisfy UNIQUE constraint)
	for i := 1; i <= 10; i++ {
		booking := &models.Booking{
			UserID:           1,
			DogID:            i, // Use different dog IDs to avoid UNIQUE constraint
			Date:             "2025-01-27",
			ScheduledTime:    fmt.Sprintf("%02d:00", 8+i),
			Status:           "scheduled",
			ApprovalStatus:   "pending",
			RequiresApproval: true,
		}
		err = bookingRepo.Create(booking)
		if err != nil {
			t.Fatalf("Failed to create booking %d: %v", i, err)
		}
	}

	// Verify bookings were created
	var createdCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM bookings WHERE approval_status = 'pending'`).Scan(&createdCount)
	if err != nil || createdCount != 10 {
		t.Fatalf("Expected 10 pending bookings to be created, got %d (error: %v)", createdCount, err)
	}

	// 10 concurrent approval attempts (simulating multiple admins)
	var wg sync.WaitGroup
	approvedCount := 0
	var mu sync.Mutex

	bookings, err := bookingRepo.GetPendingApprovalBookings()
	if err != nil {
		t.Fatalf("Failed to get pending approval bookings: %v", err)
	}

	t.Logf("Found %d pending bookings to approve", len(bookings))

	for _, booking := range bookings {
		wg.Add(1)
		go func(b *models.Booking) {
			defer wg.Done()
			err := bookingRepo.ApproveBooking(b.ID, 2) // Admin ID = 2
			if err == nil {
				mu.Lock()
				approvedCount++
				mu.Unlock()
			} else {
				t.Logf("Failed to approve booking %d: %v", b.ID, err)
			}
		}(booking)
	}

	wg.Wait()

	// Verify all bookings approved
	if approvedCount != 10 {
		t.Errorf("Expected 10 approved bookings, got %d", approvedCount)
	}

	// Verify database state
	var pendingCount int
	_ = db.QueryRow(`SELECT COUNT(*) FROM bookings WHERE approval_status = 'pending'`).Scan(&pendingCount)

	if pendingCount != 0 {
		t.Errorf("Expected 0 pending bookings, got %d", pendingCount)
	}

	var approvedCountDB int
	_ = db.QueryRow(`SELECT COUNT(*) FROM bookings WHERE approval_status = 'approved'`).Scan(&approvedCountDB)

	if approvedCountDB != 10 {
		t.Errorf("Expected 10 approved bookings in DB, got %d", approvedCountDB)
	}

	t.Logf("Concurrent approval test: %d approvals processed", approvedCount)
}

// Benchmark: Concurrent Booking Creation Performance
func BenchmarkConcurrentBookingCreation(b *testing.B) {
	db, _ := sql.Open("sqlite3", ":memory:")
	defer db.Close()

	// Setup schema
	testutil.SetupTestDB(nil)

	// Create test data
	for i := 1; i <= 100; i++ {
		_, _ = db.Exec(`
			INSERT INTO users (id, name, email, password_hash, experience_level, is_active, is_verified, terms_accepted_at, last_activity_at, created_at)
			VALUES (?, ?, ?, ?, ?, 1, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, i, fmt.Sprintf("User %d", i), fmt.Sprintf("user%d@test.com", i), "hash", "green")
	}

	_, _ = db.Exec(`
		INSERT INTO dogs (id, name, category, age, breed, is_available)
		VALUES (1, 'Test Dog', 'green', 3, 'Mixed', 1)
	`)

	bookingRepo := NewBookingRepository(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for j := 0; j < 10; j++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				booking := &models.Booking{
					UserID:         (index % 100) + 1,
					DogID:          1,
					Date:           "2025-01-27",
					ScheduledTime:  fmt.Sprintf("%02d:%02d", 9+(index/4), (index%4)*15),
					Status:         "scheduled",
					ApprovalStatus: "approved",
				}
				_ = bookingRepo.Create(booking)
			}(j)
		}
		wg.Wait()

		// Clean up for next iteration
		_, _ = db.Exec(`DELETE FROM bookings`)
	}
}

// Test: Race Condition Detection
func TestBookingCreation_RaceConditions(t *testing.T) {
	// This test should be run with: go test -race
	db := testutil.SetupTestDB(t)
	defer db.Close()

	bookingRepo := NewBookingRepository(db)

	// Create test data
	_, _ = db.Exec(`
		INSERT INTO users (id, name, email, password_hash, experience_level, is_active, is_verified, terms_accepted_at, last_activity_at, created_at)
		VALUES (1, 'User', 'user@test.com', 'hash', 'green', 1, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`)
	_, _ = db.Exec(`
		INSERT INTO dogs (id, name, category, age, breed, is_available)
		VALUES (1, 'Test Dog', 'green', 3, 'Mixed', 1)
	`)

	// Run 100 concurrent booking operations
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// Mix of operations
			switch index % 3 {
			case 0: // Create
				booking := &models.Booking{
					UserID:         1,
					DogID:          1,
					Date:           time.Now().AddDate(0, 0, index).Format("2006-01-02"),
					ScheduledTime:  fmt.Sprintf("%02d:00", (index%10)+9),
					Status:         "scheduled",
					ApprovalStatus: "approved",
				}
				_ = bookingRepo.Create(booking)
			case 1: // Read
				_, _ = bookingRepo.GetUpcoming(1, 10)
			case 2: // GetPending
				_, _ = bookingRepo.GetPendingApprovalBookings()
			}
		}(i)
	}

	wg.Wait()

	t.Log("Race condition test completed successfully (run with -race flag to detect data races)")
}
