package models

import (
	"fmt"
	"time"
)

// RecurringBookingSeries represents a recurring booking pattern
type RecurringBookingSeries struct {
	ID             int       `json:"id"`
	TenantID       int       `json:"tenant_id,omitempty"`
	UserID         int       `json:"user_id"`
	DogID          int       `json:"dog_id"`
	RecurrenceType string    `json:"recurrence_type"`         // 'weekly' or 'interval'
	DayOfWeek      *int      `json:"day_of_week,omitempty"`   // 0=Sunday, 1=Monday, ..., 6=Saturday
	IntervalDays   *int      `json:"interval_days,omitempty"` // e.g. 7, 14
	ScheduledTime  string    `json:"scheduled_time"`          // HH:MM
	StartDate      string    `json:"start_date"`              // YYYY-MM-DD
	EndDate        string    `json:"end_date"`                // YYYY-MM-DD
	Status         string    `json:"status"`                  // 'active', 'cancelled', 'completed'
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// Joined data for responses
	User     *User      `json:"user,omitempty"`
	Dog      *Dog       `json:"dog,omitempty"`
	Bookings []*Booking `json:"bookings,omitempty"`

	// Computed fields for responses
	TotalBookings      int     `json:"total_bookings,omitempty"`
	RemainingBookings  int     `json:"remaining_bookings,omitempty"`
	CancellationReason *string `json:"cancellation_reason,omitempty"`
}

// CreateRecurringBookingRequest represents a request to create a recurring booking series
type CreateRecurringBookingRequest struct {
	DogID          int      `json:"dog_id"`
	RecurrenceType string   `json:"recurrence_type"`          // 'weekly' or 'interval'
	DayOfWeek      *int     `json:"day_of_week,omitempty"`    // 0-6 for weekly
	IntervalDays   *int     `json:"interval_days,omitempty"`  // for interval type
	ScheduledTime  string   `json:"scheduled_time"`           // HH:MM
	StartDate      string   `json:"start_date"`               // YYYY-MM-DD
	EndDate        *string  `json:"end_date,omitempty"`       // YYYY-MM-DD (optional, computed from Weeks)
	Weeks          *int     `json:"weeks,omitempty"`          // alternative to EndDate
	ExcludedDates  []string `json:"excluded_dates,omitempty"` // dates to skip (from preview)
}

// RecurringBookingPreviewRequest represents a request to preview recurring booking dates
type RecurringBookingPreviewRequest struct {
	DogID          int     `json:"dog_id"`
	RecurrenceType string  `json:"recurrence_type"`
	DayOfWeek      *int    `json:"day_of_week,omitempty"`
	IntervalDays   *int    `json:"interval_days,omitempty"`
	ScheduledTime  string  `json:"scheduled_time"`
	StartDate      string  `json:"start_date"`
	EndDate        *string `json:"end_date,omitempty"`
	Weeks          *int    `json:"weeks,omitempty"`
}

// PlannedDate represents a single date in a recurring booking preview
type PlannedDate struct {
	Date   string `json:"date"`             // YYYY-MM-DD
	Status string `json:"status"`           // 'available', 'conflict', 'blocked', 'holiday', 'unavailable', 'limit_reached'
	Reason string `json:"reason,omitempty"` // human-readable explanation
}

// RecurringBookingPreviewResponse contains the preview of planned dates
type RecurringBookingPreviewResponse struct {
	DogID          int            `json:"dog_id"`
	ScheduledTime  string         `json:"scheduled_time"`
	PlannedDates   []*PlannedDate `json:"planned_dates"`
	AvailableCount int            `json:"available_count"`
	ConflictCount  int            `json:"conflict_count"`
}

// RecurringBookingFilterRequest represents filters for listing recurring series
// Note: tenant_id is passed as a mandatory parameter to FindAll(), not as a filter field
type RecurringBookingFilterRequest struct {
	UserID *int    `json:"user_id,omitempty"`
	DogID  *int    `json:"dog_id,omitempty"`
	Status *string `json:"status,omitempty"`
}

// Validate validates the create recurring booking request
func (r *CreateRecurringBookingRequest) Validate() error {
	if r.DogID <= 0 {
		return &ValidationError{Field: "dog_id", Message: "Dog ID is required"}
	}

	// Validate recurrence type
	if r.RecurrenceType != "weekly" && r.RecurrenceType != "interval" {
		return &ValidationError{Field: "recurrence_type", Message: "Recurrence type must be 'weekly' or 'interval'"}
	}

	// Validate type-specific fields
	if r.RecurrenceType == "weekly" {
		if r.DayOfWeek == nil {
			return &ValidationError{Field: "day_of_week", Message: "Day of week is required for weekly recurrence"}
		}
		if *r.DayOfWeek < 0 || *r.DayOfWeek > 6 {
			return &ValidationError{Field: "day_of_week", Message: "Day of week must be between 0 (Sunday) and 6 (Saturday)"}
		}
	}

	if r.RecurrenceType == "interval" {
		if r.IntervalDays == nil {
			return &ValidationError{Field: "interval_days", Message: "Interval days is required for interval recurrence"}
		}
		if *r.IntervalDays < 1 || *r.IntervalDays > 90 {
			return &ValidationError{Field: "interval_days", Message: "Interval days must be between 1 and 90"}
		}
	}

	// Validate scheduled time
	if r.ScheduledTime == "" {
		return &ValidationError{Field: "scheduled_time", Message: "Scheduled time is required"}
	}
	if _, err := time.Parse("15:04", r.ScheduledTime); err != nil {
		return &ValidationError{Field: "scheduled_time", Message: "Scheduled time must be in HH:MM format"}
	}

	// Validate start date
	if r.StartDate == "" {
		return &ValidationError{Field: "start_date", Message: "Start date is required"}
	}
	startDate, err := time.Parse("2006-01-02", r.StartDate)
	if err != nil {
		return &ValidationError{Field: "start_date", Message: "Start date must be in YYYY-MM-DD format"}
	}

	// Must have either end_date or weeks
	if r.EndDate == nil && r.Weeks == nil {
		return &ValidationError{Field: "end_date", Message: "Either end date or number of weeks is required"}
	}

	// Validate end date if provided
	if r.EndDate != nil {
		endDate, err := time.Parse("2006-01-02", *r.EndDate)
		if err != nil {
			return &ValidationError{Field: "end_date", Message: "End date must be in YYYY-MM-DD format"}
		}
		if !endDate.After(startDate) {
			return &ValidationError{Field: "end_date", Message: "End date must be after start date"}
		}
	}

	// Validate weeks if provided
	if r.Weeks != nil {
		if *r.Weeks < 1 || *r.Weeks > 52 {
			return &ValidationError{Field: "weeks", Message: "Number of weeks must be between 1 and 52"}
		}
	}

	// Validate excluded dates format
	for _, d := range r.ExcludedDates {
		if _, err := time.Parse("2006-01-02", d); err != nil {
			return &ValidationError{Field: "excluded_dates", Message: fmt.Sprintf("Invalid excluded date format: %s", d)}
		}
	}

	return nil
}

// Validate validates the preview request
func (r *RecurringBookingPreviewRequest) Validate() error {
	req := &CreateRecurringBookingRequest{
		DogID:          r.DogID,
		RecurrenceType: r.RecurrenceType,
		DayOfWeek:      r.DayOfWeek,
		IntervalDays:   r.IntervalDays,
		ScheduledTime:  r.ScheduledTime,
		StartDate:      r.StartDate,
		EndDate:        r.EndDate,
		Weeks:          r.Weeks,
	}
	return req.Validate()
}

// ComputeEndDate calculates the end date based on Weeks if EndDate is not set
func (r *CreateRecurringBookingRequest) ComputeEndDate() string {
	if r.EndDate != nil {
		return *r.EndDate
	}
	if r.Weeks == nil {
		return ""
	}
	startDate, _ := time.Parse("2006-01-02", r.StartDate)
	endDate := startDate.AddDate(0, 0, *r.Weeks*7-1)
	return endDate.Format("2006-01-02")
}

// GenerateDates generates all dates for this recurring pattern between start and end
func GenerateRecurringDates(recurrenceType string, dayOfWeek *int, intervalDays *int, startDate, endDate string) ([]string, error) {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start date: %w", err)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end date: %w", err)
	}

	var dates []string

	switch recurrenceType {
	case "weekly":
		if dayOfWeek == nil {
			return nil, fmt.Errorf("day_of_week required for weekly recurrence")
		}
		targetDay := time.Weekday(*dayOfWeek)

		// Find the first occurrence of the target day on or after start
		current := start
		for current.Weekday() != targetDay {
			current = current.AddDate(0, 0, 1)
		}

		// Generate all dates at 7-day intervals
		for !current.After(end) {
			dates = append(dates, current.Format("2006-01-02"))
			current = current.AddDate(0, 0, 7)
		}

	case "interval":
		if intervalDays == nil {
			return nil, fmt.Errorf("interval_days required for interval recurrence")
		}
		interval := *intervalDays

		current := start
		for !current.After(end) {
			dates = append(dates, current.Format("2006-01-02"))
			current = current.AddDate(0, 0, interval)
		}

	default:
		return nil, fmt.Errorf("unknown recurrence type: %s", recurrenceType)
	}

	return dates, nil
}
