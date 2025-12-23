package models

import (
	"testing"
)

// TestCreateWalkReportRequest_Validate tests create walk report validation
func TestCreateWalkReportRequest_Validate(t *testing.T) {
	validNotes := "Walk went well"

	t.Run("valid request with all fields", func(t *testing.T) {
		req := CreateWalkReportRequest{
			BookingID:      1,
			BehaviorRating: 5,
			EnergyLevel:    "high",
			Notes:          &validNotes,
		}
		if err := req.Validate(); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("valid request without notes", func(t *testing.T) {
		req := CreateWalkReportRequest{
			BookingID:      1,
			BehaviorRating: 3,
			EnergyLevel:    "medium",
		}
		if err := req.Validate(); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("invalid booking ID zero", func(t *testing.T) {
		req := CreateWalkReportRequest{
			BookingID:      0,
			BehaviorRating: 3,
			EnergyLevel:    "medium",
		}
		err := req.Validate()
		if err == nil {
			t.Error("Expected error for zero booking ID")
		}
		if verr, ok := err.(*ValidationError); ok {
			if verr.Field != "booking_id" {
				t.Errorf("Expected field 'booking_id', got '%s'", verr.Field)
			}
		}
	})

	t.Run("invalid booking ID negative", func(t *testing.T) {
		req := CreateWalkReportRequest{
			BookingID:      -1,
			BehaviorRating: 3,
			EnergyLevel:    "medium",
		}
		if err := req.Validate(); err == nil {
			t.Error("Expected error for negative booking ID")
		}
	})

	t.Run("behavior rating too low", func(t *testing.T) {
		req := CreateWalkReportRequest{
			BookingID:      1,
			BehaviorRating: 0,
			EnergyLevel:    "medium",
		}
		err := req.Validate()
		if err == nil {
			t.Error("Expected error for rating < 1")
		}
		if verr, ok := err.(*ValidationError); ok {
			if verr.Field != "behavior_rating" {
				t.Errorf("Expected field 'behavior_rating', got '%s'", verr.Field)
			}
		}
	})

	t.Run("behavior rating too high", func(t *testing.T) {
		req := CreateWalkReportRequest{
			BookingID:      1,
			BehaviorRating: 6,
			EnergyLevel:    "medium",
		}
		if err := req.Validate(); err == nil {
			t.Error("Expected error for rating > 5")
		}
	})

	t.Run("valid behavior ratings 1-5", func(t *testing.T) {
		for rating := 1; rating <= 5; rating++ {
			req := CreateWalkReportRequest{
				BookingID:      1,
				BehaviorRating: rating,
				EnergyLevel:    "medium",
			}
			if err := req.Validate(); err != nil {
				t.Errorf("Rating %d should be valid, got error: %v", rating, err)
			}
		}
	})

	t.Run("invalid energy level", func(t *testing.T) {
		req := CreateWalkReportRequest{
			BookingID:      1,
			BehaviorRating: 3,
			EnergyLevel:    "invalid",
		}
		err := req.Validate()
		if err == nil {
			t.Error("Expected error for invalid energy level")
		}
		if verr, ok := err.(*ValidationError); ok {
			if verr.Field != "energy_level" {
				t.Errorf("Expected field 'energy_level', got '%s'", verr.Field)
			}
		}
	})

	t.Run("valid energy levels", func(t *testing.T) {
		for _, level := range ValidEnergyLevels {
			req := CreateWalkReportRequest{
				BookingID:      1,
				BehaviorRating: 3,
				EnergyLevel:    level,
			}
			if err := req.Validate(); err != nil {
				t.Errorf("Energy level %q should be valid, got error: %v", level, err)
			}
		}
	})

	t.Run("notes too long", func(t *testing.T) {
		longNotes := string(make([]byte, 2001))
		req := CreateWalkReportRequest{
			BookingID:      1,
			BehaviorRating: 3,
			EnergyLevel:    "medium",
			Notes:          &longNotes,
		}
		err := req.Validate()
		if err == nil {
			t.Error("Expected error for notes > 2000 characters")
		}
		if verr, ok := err.(*ValidationError); ok {
			if verr.Field != "notes" {
				t.Errorf("Expected field 'notes', got '%s'", verr.Field)
			}
		}
	})

	t.Run("notes at max length is valid", func(t *testing.T) {
		maxNotes := string(make([]byte, 2000))
		req := CreateWalkReportRequest{
			BookingID:      1,
			BehaviorRating: 3,
			EnergyLevel:    "medium",
			Notes:          &maxNotes,
		}
		if err := req.Validate(); err != nil {
			t.Errorf("Notes at 2000 characters should be valid, got error: %v", err)
		}
	})
}

// TestUpdateWalkReportRequest_Validate tests update walk report validation
func TestUpdateWalkReportRequest_Validate(t *testing.T) {
	validNotes := "Updated notes"

	t.Run("valid request", func(t *testing.T) {
		req := UpdateWalkReportRequest{
			BehaviorRating: 4,
			EnergyLevel:    "low",
			Notes:          &validNotes,
		}
		if err := req.Validate(); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("behavior rating too low", func(t *testing.T) {
		req := UpdateWalkReportRequest{
			BehaviorRating: 0,
			EnergyLevel:    "medium",
		}
		if err := req.Validate(); err == nil {
			t.Error("Expected error for rating < 1")
		}
	})

	t.Run("behavior rating too high", func(t *testing.T) {
		req := UpdateWalkReportRequest{
			BehaviorRating: 6,
			EnergyLevel:    "medium",
		}
		if err := req.Validate(); err == nil {
			t.Error("Expected error for rating > 5")
		}
	})

	t.Run("invalid energy level", func(t *testing.T) {
		req := UpdateWalkReportRequest{
			BehaviorRating: 3,
			EnergyLevel:    "super-high",
		}
		if err := req.Validate(); err == nil {
			t.Error("Expected error for invalid energy level")
		}
	})

	t.Run("notes too long", func(t *testing.T) {
		longNotes := string(make([]byte, 2001))
		req := UpdateWalkReportRequest{
			BehaviorRating: 3,
			EnergyLevel:    "medium",
			Notes:          &longNotes,
		}
		if err := req.Validate(); err == nil {
			t.Error("Expected error for notes > 2000 characters")
		}
	})
}

// TestValidEnergyLevels tests the valid energy levels slice
func TestValidEnergyLevels(t *testing.T) {
	expected := []string{"low", "medium", "high"}

	if len(ValidEnergyLevels) != len(expected) {
		t.Errorf("Expected %d energy levels, got %d", len(expected), len(ValidEnergyLevels))
	}

	for i, level := range expected {
		if ValidEnergyLevels[i] != level {
			t.Errorf("Expected energy level %d to be %q, got %q", i, level, ValidEnergyLevels[i])
		}
	}
}
