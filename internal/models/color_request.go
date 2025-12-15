package models

import "time"

// ColorRequest represents a request for a user to gain access to a color category
type ColorRequest struct {
	ID           int        `json:"id"`
	UserID       int        `json:"user_id"`
	ColorID      int        `json:"color_id"`
	Status       string     `json:"status"`
	AdminMessage *string    `json:"admin_message,omitempty"`
	ReviewedBy   *int       `json:"reviewed_by,omitempty"`
	ReviewedAt   *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`

	// Joined data for responses
	User  *User          `json:"user,omitempty"`
	Color *ColorCategory `json:"color,omitempty"`
}

// CreateColorRequestRequest represents a request to create a color request
type CreateColorRequestRequest struct {
	ColorID int `json:"color_id"`
}

// ReviewColorRequestRequest represents a request to review a color request
type ReviewColorRequestRequest struct {
	Approved bool    `json:"approved"`
	Message  *string `json:"message,omitempty"`
}

// Validate validates the create color request
func (r *CreateColorRequestRequest) Validate() error {
	if r.ColorID <= 0 {
		return &ValidationError{Field: "color_id", Message: "Farb-ID ist erforderlich"}
	}

	return nil
}

// Validate validates the review request
func (r *ReviewColorRequestRequest) Validate() error {
	// No specific validation needed, approved is a boolean
	return nil
}
