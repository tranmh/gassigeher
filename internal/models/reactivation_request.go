package models

import "time"

// ReactivationRequest represents a request to reactivate a deactivated account
type ReactivationRequest struct {
	ID           int        `json:"id"`
	TenantID     int        `json:"tenant_id,omitempty"` // SaaS: Tenant this request belongs to
	UserID       int        `json:"user_id"`
	Status       string     `json:"status"`
	AdminMessage *string    `json:"admin_message,omitempty"`
	ReviewedBy   *int       `json:"reviewed_by,omitempty"`
	ReviewedAt   *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`

	// Joined data for responses
	User *User `json:"user,omitempty"`
}

// CreateReactivationRequestRequest represents a request to create a reactivation request
type CreateReactivationRequestRequest struct {
	// No fields needed - user ID comes from auth context
}

// ReviewReactivationRequestRequest represents a request to review a reactivation request
type ReviewReactivationRequestRequest struct {
	Approved bool    `json:"approved"`
	Message  *string `json:"message,omitempty"`
}

// Validate validates the review request
func (r *ReviewReactivationRequestRequest) Validate() error {
	// No specific validation needed
	return nil
}
