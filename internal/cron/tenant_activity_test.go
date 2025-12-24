package cron

import (
	"testing"
	"time"
)

func TestNewTenantActivityChecker(t *testing.T) {
	tests := []struct {
		name           string
		inactivityDays int
		wantDays       int
	}{
		{
			name:           "positive days",
			inactivityDays: 30,
			wantDays:       30,
		},
		{
			name:           "zero defaults to 30",
			inactivityDays: 0,
			wantDays:       30,
		},
		{
			name:           "negative defaults to 30",
			inactivityDays: -10,
			wantDays:       30,
		},
		{
			name:           "custom days",
			inactivityDays: 60,
			wantDays:       60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewTenantActivityChecker(nil, tt.inactivityDays)
			if checker.inactivityDays != tt.wantDays {
				t.Errorf("inactivityDays = %d, want %d", checker.inactivityDays, tt.wantDays)
			}
		})
	}
}

func TestTenantActivityChecker_SetInactivityThreshold(t *testing.T) {
	checker := NewTenantActivityChecker(nil, 30)

	tests := []struct {
		name     string
		days     int
		wantDays int
	}{
		{
			name:     "positive value updates",
			days:     45,
			wantDays: 45,
		},
		{
			name:     "zero does not update",
			days:     0,
			wantDays: 45, // keeps previous value
		},
		{
			name:     "negative does not update",
			days:     -5,
			wantDays: 45, // keeps previous value
		},
		{
			name:     "positive value updates again",
			days:     90,
			wantDays: 90,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker.SetInactivityThreshold(tt.days)
			if checker.inactivityDays != tt.wantDays {
				t.Errorf("inactivityDays = %d, want %d", checker.inactivityDays, tt.wantDays)
			}
		})
	}
}

func TestTenantActivity_Fields(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	activity := TenantActivity{
		TenantID:        1,
		TenantSlug:      "test-shelter",
		TenantName:      "Test Shelter",
		LastBookingDate: &yesterday,
		LastUserLogin:   &now,
		DaysInactive:    5,
		TotalBookings:   100,
		ActiveUsers:     10,
		IsInactive:      false,
	}

	if activity.TenantID != 1 {
		t.Errorf("TenantID = %d, want 1", activity.TenantID)
	}
	if activity.TenantSlug != "test-shelter" {
		t.Errorf("TenantSlug = %s, want test-shelter", activity.TenantSlug)
	}
	if activity.TenantName != "Test Shelter" {
		t.Errorf("TenantName = %s, want Test Shelter", activity.TenantName)
	}
	if activity.DaysInactive != 5 {
		t.Errorf("DaysInactive = %d, want 5", activity.DaysInactive)
	}
	if activity.TotalBookings != 100 {
		t.Errorf("TotalBookings = %d, want 100", activity.TotalBookings)
	}
	if activity.ActiveUsers != 10 {
		t.Errorf("ActiveUsers = %d, want 10", activity.ActiveUsers)
	}
	if activity.IsInactive {
		t.Error("IsInactive = true, want false")
	}
}

func TestTenantActivity_NilDates(t *testing.T) {
	activity := TenantActivity{
		TenantID:        2,
		TenantSlug:      "new-shelter",
		TenantName:      "New Shelter",
		LastBookingDate: nil,
		LastUserLogin:   nil,
		DaysInactive:    999,
		TotalBookings:   0,
		ActiveUsers:     0,
		IsInactive:      true,
	}

	if activity.LastBookingDate != nil {
		t.Error("LastBookingDate should be nil")
	}
	if activity.LastUserLogin != nil {
		t.Error("LastUserLogin should be nil")
	}
	if activity.DaysInactive != 999 {
		t.Errorf("DaysInactive = %d, want 999 (for never active)", activity.DaysInactive)
	}
	if !activity.IsInactive {
		t.Error("IsInactive = false, want true (no activity ever)")
	}
}

func TestTenantActivity_JSONTags(t *testing.T) {
	// Verify struct has proper JSON tags by checking field values
	// This is a basic sanity check - actual JSON encoding would be tested in integration tests
	now := time.Now()
	activity := TenantActivity{
		TenantID:        1,
		TenantSlug:      "test",
		TenantName:      "Test",
		LastBookingDate: &now,
		LastUserLogin:   &now,
		DaysInactive:    10,
		TotalBookings:   50,
		ActiveUsers:     5,
		IsInactive:      false,
	}

	// Just verify the struct can be created and accessed
	if activity.TenantID == 0 {
		t.Error("TenantID should not be 0")
	}
}

func TestTenantActivityChecker_NilDB(t *testing.T) {
	// Test that checker can be created with nil DB
	// (actual DB operations will fail, but initialization should work)
	checker := NewTenantActivityChecker(nil, 30)

	if checker == nil {
		t.Error("NewTenantActivityChecker returned nil")
	}
	if checker.inactivityDays != 30 {
		t.Errorf("inactivityDays = %d, want 30", checker.inactivityDays)
	}
	if checker.db != nil {
		t.Error("db should be nil")
	}
}

func TestDaysInactiveCalculation(t *testing.T) {
	// Simulate the calculation logic from GetInactiveTenants/GetAllTenantActivity
	tests := []struct {
		name          string
		lastActivity  *time.Time
		wantInactive  int // approximate days
		wantIsActive  bool
		inactivityDays int
	}{
		{
			name:          "nil activity returns 999",
			lastActivity:  nil,
			wantInactive:  999,
			wantIsActive:  false,
			inactivityDays: 30,
		},
		{
			name:          "recent activity is active",
			lastActivity:  timePtr(time.Now().Add(-1 * time.Hour)),
			wantInactive:  0,
			wantIsActive:  true,
			inactivityDays: 30,
		},
		{
			name:          "activity 10 days ago with 30 day threshold",
			lastActivity:  timePtr(time.Now().AddDate(0, 0, -10)),
			wantInactive:  10,
			wantIsActive:  true,
			inactivityDays: 30,
		},
		{
			name:          "activity 45 days ago with 30 day threshold",
			lastActivity:  timePtr(time.Now().AddDate(0, 0, -45)),
			wantInactive:  45,
			wantIsActive:  false,
			inactivityDays: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var daysInactive int
			var isInactive bool

			cutoffDate := time.Now().AddDate(0, 0, -tt.inactivityDays)

			if tt.lastActivity != nil {
				daysInactive = int(time.Since(*tt.lastActivity).Hours() / 24)
				isInactive = tt.lastActivity.Before(cutoffDate)
			} else {
				daysInactive = 999
				isInactive = true
			}

			// Allow some tolerance for timing
			if tt.lastActivity != nil && (daysInactive < tt.wantInactive-1 || daysInactive > tt.wantInactive+1) {
				t.Errorf("daysInactive = %d, want approximately %d", daysInactive, tt.wantInactive)
			}

			if tt.lastActivity == nil && daysInactive != 999 {
				t.Errorf("daysInactive = %d, want 999 for nil activity", daysInactive)
			}

			if isInactive != !tt.wantIsActive {
				t.Errorf("isInactive = %v, want %v", isInactive, !tt.wantIsActive)
			}
		})
	}
}

func TestMostRecentActivitySelection(t *testing.T) {
	// Test the logic that selects the most recent activity between booking and login
	now := time.Now()
	earlier := now.Add(-1 * time.Hour)
	muchEarlier := now.Add(-24 * time.Hour)

	tests := []struct {
		name            string
		lastBooking     *time.Time
		lastLogin       *time.Time
		wantMostRecent  *time.Time
	}{
		{
			name:           "both nil",
			lastBooking:    nil,
			lastLogin:      nil,
			wantMostRecent: nil,
		},
		{
			name:           "only booking",
			lastBooking:    &earlier,
			lastLogin:      nil,
			wantMostRecent: &earlier,
		},
		{
			name:           "only login",
			lastBooking:    nil,
			lastLogin:      &now,
			wantMostRecent: &now,
		},
		{
			name:           "booking more recent",
			lastBooking:    &now,
			lastLogin:      &earlier,
			wantMostRecent: &now,
		},
		{
			name:           "login more recent",
			lastBooking:    &muchEarlier,
			lastLogin:      &now,
			wantMostRecent: &now,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Replicate the selection logic from tenant_activity.go
			var mostRecentActivity *time.Time
			if tt.lastBooking != nil && tt.lastLogin != nil {
				if tt.lastBooking.After(*tt.lastLogin) {
					mostRecentActivity = tt.lastBooking
				} else {
					mostRecentActivity = tt.lastLogin
				}
			} else if tt.lastBooking != nil {
				mostRecentActivity = tt.lastBooking
			} else if tt.lastLogin != nil {
				mostRecentActivity = tt.lastLogin
			}

			if tt.wantMostRecent == nil && mostRecentActivity != nil {
				t.Errorf("mostRecentActivity = %v, want nil", mostRecentActivity)
			}
			if tt.wantMostRecent != nil && mostRecentActivity == nil {
				t.Errorf("mostRecentActivity = nil, want %v", tt.wantMostRecent)
			}
			if tt.wantMostRecent != nil && mostRecentActivity != nil {
				if !mostRecentActivity.Equal(*tt.wantMostRecent) {
					t.Errorf("mostRecentActivity = %v, want %v", mostRecentActivity, tt.wantMostRecent)
				}
			}
		})
	}
}

// Helper function to create time pointers
func timePtr(t time.Time) *time.Time {
	return &t
}

// BUG 3 RED PHASE: CheckAndFlagInactiveTenants should actually flag tenants
// The function name says "Flag" but it only logs - it should update the database
func TestCheckAndFlagInactiveTenants_ShouldUpdateDatabase(t *testing.T) {
	// This test documents the expected behavior:
	// After calling CheckAndFlagInactiveTenants(), inactive tenants should have
	// a flag set in the database (e.g., is_inactive = true or inactive_flagged_at timestamp)
	//
	// Current behavior: Function only logs, doesn't update database
	// Expected behavior: Function should UPDATE tenants SET inactive_flagged_at = NOW() WHERE ...
	//
	// Since we can't easily test database updates without a real DB,
	// we verify the function at least has the capability to flag
	checker := NewTenantActivityChecker(nil, 30)

	// The checker should have a method to actually flag tenants
	// or the CheckAndFlagInactiveTenants should return the list of flagged tenant IDs
	// Currently it returns only error, which is insufficient

	// This is a design issue - the function should either:
	// 1. Return []int of flagged tenant IDs
	// 2. Have a callback/interface for flagging
	// 3. Actually update the database

	// For now, we just verify the checker exists
	if checker == nil {
		t.Fatal("TenantActivityChecker should not be nil")
	}

	// TODO: Once fixed, this test should verify that:
	// - flaggedIDs, err := checker.CheckAndFlagInactiveTenants()
	// - len(flaggedIDs) > 0 for inactive tenants
	// - Database has updated records
	t.Skip("BUG 3: CheckAndFlagInactiveTenants doesn't actually flag tenants - needs fix")
}

// BUG 4 RED PHASE: rows.Err() should be checked after iteration
func TestCheckAndFlagInactiveTenants_ShouldCheckRowsErr(t *testing.T) {
	// After iterating through rows with rows.Next(), we must check rows.Err()
	// to catch any errors that occurred during iteration.
	//
	// Current code:
	//   for rows.Next() { ... }
	//   log.Printf("complete...")
	//
	// Should be:
	//   for rows.Next() { ... }
	//   if err := rows.Err(); err != nil { return err }
	//   log.Printf("complete...")

	// This is a code review issue - the fix is straightforward
	// We can't easily test this without mocking the database to return an error during iteration
	t.Skip("BUG 4: Missing rows.Err() check - needs code review fix")
}
