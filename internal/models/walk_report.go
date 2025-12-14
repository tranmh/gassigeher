package models

import "time"

// WalkReport represents a user's report after completing a walk
type WalkReport struct {
	ID             int       `json:"id"`
	BookingID      int       `json:"booking_id"`
	BehaviorRating int       `json:"behavior_rating"` // 1-5 scale
	EnergyLevel    string    `json:"energy_level"`    // low, medium, high
	Notes          *string   `json:"notes,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// Photos attached to this report
	Photos []WalkReportPhoto `json:"photos,omitempty"`

	// Joined data for responses
	Booking *Booking `json:"booking,omitempty"`
	Dog     *Dog     `json:"dog,omitempty"`
	User    *User    `json:"user,omitempty"`
}

// WalkReportPhoto represents a photo attached to a walk report
type WalkReportPhoto struct {
	ID             int       `json:"id"`
	WalkReportID   int       `json:"walk_report_id"`
	PhotoPath      string    `json:"photo_path"`
	PhotoThumbnail string    `json:"photo_thumbnail"`
	DisplayOrder   int       `json:"display_order"`
	CreatedAt      time.Time `json:"created_at"`
}

// CreateWalkReportRequest represents a request to create a walk report
type CreateWalkReportRequest struct {
	BookingID      int     `json:"booking_id"`
	BehaviorRating int     `json:"behavior_rating"`
	EnergyLevel    string  `json:"energy_level"`
	Notes          *string `json:"notes,omitempty"`
}

// UpdateWalkReportRequest represents a request to update a walk report
type UpdateWalkReportRequest struct {
	BehaviorRating int     `json:"behavior_rating"`
	EnergyLevel    string  `json:"energy_level"`
	Notes          *string `json:"notes,omitempty"`
}

// WalkReportStats represents aggregated statistics for a dog's walk reports
type WalkReportStats struct {
	TotalWalks        int     `json:"total_walks"`
	AverageRating     float64 `json:"average_rating"`
	ReportsWithPhotos int     `json:"reports_with_photos"`
}

// DogWalkReportsResponse represents the response for a dog's walk history
type DogWalkReportsResponse struct {
	Dog     *Dog             `json:"dog"`
	Stats   *WalkReportStats `json:"stats"`
	Reports []*WalkReport    `json:"reports"`
}

// ValidEnergyLevels contains the allowed energy level values
var ValidEnergyLevels = []string{"low", "medium", "high"}

// Validate validates the create walk report request
func (r *CreateWalkReportRequest) Validate() error {
	if r.BookingID <= 0 {
		return &ValidationError{Field: "booking_id", Message: "Booking ID is required"}
	}

	if r.BehaviorRating < 1 || r.BehaviorRating > 5 {
		return &ValidationError{Field: "behavior_rating", Message: "Behavior rating must be between 1 and 5"}
	}

	if !isValidEnergyLevel(r.EnergyLevel) {
		return &ValidationError{Field: "energy_level", Message: "Energy level must be 'low', 'medium', or 'high'"}
	}

	// Notes are optional but have a max length
	if r.Notes != nil && len(*r.Notes) > 2000 {
		return &ValidationError{Field: "notes", Message: "Notes must be 2000 characters or less"}
	}

	return nil
}

// Validate validates the update walk report request
func (r *UpdateWalkReportRequest) Validate() error {
	if r.BehaviorRating < 1 || r.BehaviorRating > 5 {
		return &ValidationError{Field: "behavior_rating", Message: "Behavior rating must be between 1 and 5"}
	}

	if !isValidEnergyLevel(r.EnergyLevel) {
		return &ValidationError{Field: "energy_level", Message: "Energy level must be 'low', 'medium', or 'high'"}
	}

	// Notes are optional but have a max length
	if r.Notes != nil && len(*r.Notes) > 2000 {
		return &ValidationError{Field: "notes", Message: "Notes must be 2000 characters or less"}
	}

	return nil
}

// isValidEnergyLevel checks if the given energy level is valid
func isValidEnergyLevel(level string) bool {
	for _, valid := range ValidEnergyLevels {
		if level == valid {
			return true
		}
	}
	return false
}
