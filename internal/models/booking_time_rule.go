package models

import (
	"fmt"
	"time"
)

type BookingTimeRule struct {
	ID        int       `json:"id"`
	DayType   string    `json:"day_type"`   // 'weekday', 'weekend', 'holiday'
	RuleName  string    `json:"rule_name"`
	StartTime string    `json:"start_time"` // HH:MM format
	EndTime   string    `json:"end_time"`   // HH:MM format
	IsBlocked bool      `json:"is_blocked"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Validate validates booking time rule
func (r *BookingTimeRule) Validate() error {
	if r.DayType != "weekday" && r.DayType != "weekend" && r.DayType != "holiday" {
		return fmt.Errorf("day_type must be 'weekday', 'weekend', or 'holiday'")
	}
	if r.RuleName == "" {
		return fmt.Errorf("rule_name is required")
	}

	// Validate time format
	if !isValidTimeFormat(r.StartTime) {
		return fmt.Errorf("start_time must be in HH:MM format")
	}
	if !isValidTimeFormat(r.EndTime) {
		return fmt.Errorf("end_time must be in HH:MM format")
	}

	// Validate end > start
	if r.EndTime <= r.StartTime {
		return fmt.Errorf("end_time must be after start_time")
	}

	return nil
}

func isValidTimeFormat(t string) bool {
	_, err := time.Parse("15:04", t)
	return err == nil
}
