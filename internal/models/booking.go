package models

import "time"

// Booking represents a dog walking booking
type Booking struct {
	ID                      int        `json:"id"`
	UserID                  int        `json:"user_id"`
	DogID                   int        `json:"dog_id"`
	Date                    string     `json:"date"` // YYYY-MM-DD format
	ScheduledTime           string     `json:"scheduled_time"` // HH:MM format
	Status                  string     `json:"status"`
	CompletedAt             *time.Time `json:"completed_at,omitempty"`
	ReminderSentAt          *time.Time `json:"reminder_sent_at,omitempty"`
	UserNotes               *string    `json:"user_notes,omitempty"`
	AdminCancellationReason *string    `json:"admin_cancellation_reason,omitempty"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`

	// Approval workflow fields
	RequiresApproval bool       `json:"requires_approval"`
	ApprovalStatus   string     `json:"approval_status"` // 'pending', 'approved', 'rejected'
	ApprovedBy       *int       `json:"approved_by,omitempty"`
	ApprovedAt       *time.Time `json:"approved_at,omitempty"`
	RejectionReason  *string    `json:"rejection_reason,omitempty"`

	// Joined data for responses
	User *User `json:"user,omitempty"`
	Dog  *Dog  `json:"dog,omitempty"`
}

// CreateBookingRequest represents a request to create a booking
type CreateBookingRequest struct {
	DogID         int    `json:"dog_id"`
	Date          string `json:"date"` // YYYY-MM-DD
	ScheduledTime string `json:"scheduled_time"` // HH:MM
}

// CancelBookingRequest represents a request to cancel a booking
type CancelBookingRequest struct {
	Reason *string `json:"reason,omitempty"` // Optional for users, required for admins
}

// AddNotesRequest represents a request to add notes to a completed booking
type AddNotesRequest struct {
	Notes string `json:"notes"`
}

// MoveBookingRequest represents a request to move a booking to a new date/time
type MoveBookingRequest struct {
	Date          string `json:"date"`
	ScheduledTime string `json:"scheduled_time"`
	Reason        string `json:"reason"`
}

// Validate validates the move booking request
func (r *MoveBookingRequest) Validate() error {
	if r.Date == "" {
		return &ValidationError{Field: "date", Message: "Date is required"}
	}

	if _, err := time.Parse("2006-01-02", r.Date); err != nil {
		return &ValidationError{Field: "date", Message: "Date must be in YYYY-MM-DD format"}
	}

	if r.ScheduledTime == "" {
		return &ValidationError{Field: "scheduled_time", Message: "Scheduled time is required"}
	}

	if _, err := time.Parse("15:04", r.ScheduledTime); err != nil {
		return &ValidationError{Field: "scheduled_time", Message: "Scheduled time must be in HH:MM format"}
	}

	if r.Reason == "" {
		return &ValidationError{Field: "reason", Message: "Reason is required"}
	}

	return nil
}

// BookingFilterRequest represents filters for listing bookings
type BookingFilterRequest struct {
	UserID   *int    `json:"user_id,omitempty"`
	DogID    *int    `json:"dog_id,omitempty"`
	DateFrom *string `json:"date_from,omitempty"`
	DateTo   *string `json:"date_to,omitempty"`
	Status   *string `json:"status,omitempty"`
	Year     *int    `json:"year,omitempty"`
	Month    *int    `json:"month,omitempty"`
}

// CalendarDay represents a day in the calendar with bookings
type CalendarDay struct {
	Date     string     `json:"date"`
	Bookings []*Booking `json:"bookings"`
	IsBlocked bool      `json:"is_blocked"`
	BlockedReason *string `json:"blocked_reason,omitempty"`
}

// CalendarResponse represents a month view of the calendar
type CalendarResponse struct {
	Year  int            `json:"year"`
	Month int            `json:"month"`
	Days  []*CalendarDay `json:"days"`
}

// Validate validates the create booking request
func (r *CreateBookingRequest) Validate() error {
	if r.DogID <= 0 {
		return &ValidationError{Field: "dog_id", Message: "Dog ID is required"}
	}

	if r.Date == "" {
		return &ValidationError{Field: "date", Message: "Date is required"}
	}

	// Validate date format (YYYY-MM-DD)
	if _, err := time.Parse("2006-01-02", r.Date); err != nil {
		return &ValidationError{Field: "date", Message: "Date must be in YYYY-MM-DD format"}
	}

	if r.ScheduledTime == "" {
		return &ValidationError{Field: "scheduled_time", Message: "Scheduled time is required"}
	}

	// Validate time format (HH:MM)
	if _, err := time.Parse("15:04", r.ScheduledTime); err != nil {
		return &ValidationError{Field: "scheduled_time", Message: "Scheduled time must be in HH:MM format"}
	}

	return nil
}
