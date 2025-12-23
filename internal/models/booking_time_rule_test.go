package models

import (
	"testing"
)

// TestBookingTimeRule_Validate tests booking time rule validation
func TestBookingTimeRule_Validate(t *testing.T) {
	t.Run("valid weekday rule", func(t *testing.T) {
		rule := BookingTimeRule{
			DayType:   "weekday",
			RuleName:  "Morning walk",
			StartTime: "08:00",
			EndTime:   "12:00",
		}
		if err := rule.Validate(); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("valid weekend rule", func(t *testing.T) {
		rule := BookingTimeRule{
			DayType:   "weekend",
			RuleName:  "Weekend morning",
			StartTime: "09:00",
			EndTime:   "11:00",
		}
		if err := rule.Validate(); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("valid holiday rule", func(t *testing.T) {
		rule := BookingTimeRule{
			DayType:   "holiday",
			RuleName:  "Holiday afternoon",
			StartTime: "14:00",
			EndTime:   "17:00",
		}
		if err := rule.Validate(); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("invalid day type", func(t *testing.T) {
		rule := BookingTimeRule{
			DayType:   "invalid",
			RuleName:  "Test rule",
			StartTime: "08:00",
			EndTime:   "12:00",
		}
		if err := rule.Validate(); err == nil {
			t.Error("Expected error for invalid day type")
		}
	})

	t.Run("empty day type", func(t *testing.T) {
		rule := BookingTimeRule{
			DayType:   "",
			RuleName:  "Test rule",
			StartTime: "08:00",
			EndTime:   "12:00",
		}
		if err := rule.Validate(); err == nil {
			t.Error("Expected error for empty day type")
		}
	})

	t.Run("empty rule name", func(t *testing.T) {
		rule := BookingTimeRule{
			DayType:   "weekday",
			RuleName:  "",
			StartTime: "08:00",
			EndTime:   "12:00",
		}
		if err := rule.Validate(); err == nil {
			t.Error("Expected error for empty rule name")
		}
	})

	t.Run("invalid start time format", func(t *testing.T) {
		rule := BookingTimeRule{
			DayType:   "weekday",
			RuleName:  "Test rule",
			StartTime: "8:00", // Missing leading zero
			EndTime:   "12:00",
		}
		if err := rule.Validate(); err == nil {
			t.Error("Expected error for invalid start time format")
		}
	})

	t.Run("invalid end time format", func(t *testing.T) {
		rule := BookingTimeRule{
			DayType:   "weekday",
			RuleName:  "Test rule",
			StartTime: "08:00",
			EndTime:   "12-00", // Wrong separator
		}
		if err := rule.Validate(); err == nil {
			t.Error("Expected error for invalid end time format")
		}
	})

	t.Run("end time before start time", func(t *testing.T) {
		rule := BookingTimeRule{
			DayType:   "weekday",
			RuleName:  "Test rule",
			StartTime: "12:00",
			EndTime:   "08:00",
		}
		if err := rule.Validate(); err == nil {
			t.Error("Expected error when end time is before start time")
		}
	})

	t.Run("end time equals start time", func(t *testing.T) {
		rule := BookingTimeRule{
			DayType:   "weekday",
			RuleName:  "Test rule",
			StartTime: "10:00",
			EndTime:   "10:00",
		}
		if err := rule.Validate(); err == nil {
			t.Error("Expected error when end time equals start time")
		}
	})

	t.Run("valid edge case times", func(t *testing.T) {
		// Test 00:00 to 23:59
		rule := BookingTimeRule{
			DayType:   "weekday",
			RuleName:  "Full day",
			StartTime: "00:00",
			EndTime:   "23:59",
		}
		if err := rule.Validate(); err != nil {
			t.Errorf("Expected no error for 00:00-23:59, got %v", err)
		}
	})

	t.Run("valid blocked rule", func(t *testing.T) {
		rule := BookingTimeRule{
			DayType:   "weekday",
			RuleName:  "Lunch break",
			StartTime: "12:00",
			EndTime:   "13:00",
			IsBlocked: true,
		}
		if err := rule.Validate(); err != nil {
			t.Errorf("Expected no error for blocked rule, got %v", err)
		}
	})

	t.Run("invalid time 25:00", func(t *testing.T) {
		rule := BookingTimeRule{
			DayType:   "weekday",
			RuleName:  "Test rule",
			StartTime: "25:00",
			EndTime:   "26:00",
		}
		if err := rule.Validate(); err == nil {
			t.Error("Expected error for invalid hour 25")
		}
	})

	t.Run("invalid time 12:60", func(t *testing.T) {
		rule := BookingTimeRule{
			DayType:   "weekday",
			RuleName:  "Test rule",
			StartTime: "12:60",
			EndTime:   "13:00",
		}
		if err := rule.Validate(); err == nil {
			t.Error("Expected error for invalid minute 60")
		}
	})

	t.Run("time format with seconds is invalid", func(t *testing.T) {
		rule := BookingTimeRule{
			DayType:   "weekday",
			RuleName:  "Test rule",
			StartTime: "12:00:00",
			EndTime:   "13:00:00",
		}
		if err := rule.Validate(); err == nil {
			t.Error("Expected error for time with seconds")
		}
	})

	t.Run("whitespace in rule name is allowed", func(t *testing.T) {
		rule := BookingTimeRule{
			DayType:   "weekday",
			RuleName:  "  Morning walk  ",
			StartTime: "08:00",
			EndTime:   "12:00",
		}
		// Note: The current validation doesn't trim whitespace,
		// so this should be valid (just whitespace around name)
		if err := rule.Validate(); err != nil {
			t.Errorf("Expected no error for rule name with whitespace, got %v", err)
		}
	})
}

// TestAllDayTypes tests all valid day types
func TestAllDayTypes(t *testing.T) {
	validDayTypes := []string{"weekday", "weekend", "holiday"}

	for _, dayType := range validDayTypes {
		t.Run(dayType, func(t *testing.T) {
			rule := BookingTimeRule{
				DayType:   dayType,
				RuleName:  "Test rule",
				StartTime: "08:00",
				EndTime:   "12:00",
			}
			if err := rule.Validate(); err != nil {
				t.Errorf("Expected %q to be valid day type, got error: %v", dayType, err)
			}
		})
	}
}
