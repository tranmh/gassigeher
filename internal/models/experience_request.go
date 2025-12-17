package models

import "time"

// ExperienceRequest represents a request for experience level promotion
type ExperienceRequest struct {
	ID             int        `json:"id"`
	TenantID       int        `json:"tenant_id,omitempty"` // SaaS: Tenant this request belongs to
	UserID         int        `json:"user_id"`
	RequestedLevel string     `json:"requested_level"`
	Status         string     `json:"status"`
	AdminMessage   *string    `json:"admin_message,omitempty"`
	ReviewedBy     *int       `json:"reviewed_by,omitempty"`
	ReviewedAt     *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`

	// Joined data for responses
	User *User `json:"user,omitempty"`
}

// CreateExperienceRequestRequest represents a request to create an experience level request
type CreateExperienceRequestRequest struct {
	RequestedLevel string `json:"requested_level"`
}

// ReviewExperienceRequestRequest represents a request to review an experience request
type ReviewExperienceRequestRequest struct {
	Approved bool    `json:"approved"`
	Message  *string `json:"message,omitempty"`
}

// Validate validates the create experience request
func (r *CreateExperienceRequestRequest) Validate() error {
	if r.RequestedLevel != "blue" && r.RequestedLevel != "orange" {
		return &ValidationError{Field: "requested_level", Message: "Requested level must be 'blue' or 'orange'"}
	}

	return nil
}

// Validate validates the review request
func (r *ReviewExperienceRequestRequest) Validate() error {
	// No specific validation needed, approved is a boolean
	return nil
}
