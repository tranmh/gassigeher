package models

import "time"

// BlockedDate represents a date that is blocked from bookings
// If DogID is nil, the block applies to ALL dogs (global block)
// If DogID is set, the block applies only to that specific dog
type BlockedDate struct {
	ID        int       `json:"id"`
	Date      string    `json:"date"`                    // YYYY-MM-DD format
	DogID     *int      `json:"dog_id"`                  // NULL = all dogs, specific ID = that dog only
	DogName   *string   `json:"dog_name,omitempty"`      // Populated via JOIN for display
	Reason    string    `json:"reason"`
	CreatedBy int       `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateBlockedDateRequest represents a request to block a date
type CreateBlockedDateRequest struct {
	Date   string `json:"date"`
	DogID  *int   `json:"dog_id,omitempty"` // Optional: NULL means all dogs
	Reason string `json:"reason"`
}

// Validate validates the create blocked date request
func (r *CreateBlockedDateRequest) Validate() error {
	if r.Date == "" {
		return &ValidationError{Field: "date", Message: "Date is required"}
	}

	// Validate date format (YYYY-MM-DD)
	if _, err := time.Parse("2006-01-02", r.Date); err != nil {
		return &ValidationError{Field: "date", Message: "Date must be in YYYY-MM-DD format"}
	}

	if r.Reason == "" {
		return &ValidationError{Field: "reason", Message: "Reason is required"}
	}

	// DogID validation: if provided, must be positive
	if r.DogID != nil && *r.DogID <= 0 {
		return &ValidationError{Field: "dog_id", Message: "Dog ID must be a positive integer"}
	}

	return nil
}
