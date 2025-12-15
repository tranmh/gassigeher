package models

import "time"

// UserColor represents the many-to-many relationship between users and color categories
type UserColor struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	ColorID   int       `json:"color_id"`
	GrantedAt time.Time `json:"granted_at"`
	GrantedBy *int      `json:"granted_by,omitempty"`

	// Joined data for responses
	Color *ColorCategory `json:"color,omitempty"`
}

// AddColorToUserRequest represents a request to add a color to a user
type AddColorToUserRequest struct {
	ColorID int `json:"color_id"`
}

// SetUserColorsRequest represents a request to set all colors for a user
type SetUserColorsRequest struct {
	ColorIDs []int `json:"color_ids"`
}

// Validate validates the add color to user request
func (r *AddColorToUserRequest) Validate() error {
	if r.ColorID <= 0 {
		return &ValidationError{Field: "color_id", Message: "Farb-ID ist erforderlich"}
	}

	return nil
}

// Validate validates the set user colors request
func (r *SetUserColorsRequest) Validate() error {
	// Empty list is valid (removes all colors)
	for _, id := range r.ColorIDs {
		if id <= 0 {
			return &ValidationError{Field: "color_ids", Message: "Ungueltige Farb-ID in der Liste"}
		}
	}

	return nil
}
