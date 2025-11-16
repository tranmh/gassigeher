package models

import (
	"time"
)

// Dog represents a dog in the system
type Dog struct {
	ID                   int        `json:"id"`
	Name                 string     `json:"name"`
	Breed                string     `json:"breed"`
	Size                 string     `json:"size"` // small, medium, large
	Age                  int        `json:"age"`
	Category             string     `json:"category"` // green, blue, orange
	Photo                *string    `json:"photo,omitempty"`
	SpecialNeeds         *string    `json:"special_needs,omitempty"`
	PickupLocation       *string    `json:"pickup_location,omitempty"`
	WalkRoute            *string    `json:"walk_route,omitempty"`
	WalkDuration         *int       `json:"walk_duration,omitempty"` // minutes
	SpecialInstructions  *string    `json:"special_instructions,omitempty"`
	DefaultMorningTime   *string    `json:"default_morning_time,omitempty"` // HH:MM format
	DefaultEveningTime   *string    `json:"default_evening_time,omitempty"` // HH:MM format
	IsAvailable          bool       `json:"is_available"`
	UnavailableReason    *string    `json:"unavailable_reason,omitempty"`
	UnavailableSince     *time.Time `json:"unavailable_since,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// CreateDogRequest represents the request to create a dog
type CreateDogRequest struct {
	Name                string  `json:"name"`
	Breed               string  `json:"breed"`
	Size                string  `json:"size"`
	Age                 int     `json:"age"`
	Category            string  `json:"category"`
	SpecialNeeds        *string `json:"special_needs,omitempty"`
	PickupLocation      *string `json:"pickup_location,omitempty"`
	WalkRoute           *string `json:"walk_route,omitempty"`
	WalkDuration        *int    `json:"walk_duration,omitempty"`
	SpecialInstructions *string `json:"special_instructions,omitempty"`
	DefaultMorningTime  *string `json:"default_morning_time,omitempty"`
	DefaultEveningTime  *string `json:"default_evening_time,omitempty"`
}

// UpdateDogRequest represents the request to update a dog
type UpdateDogRequest struct {
	Name                *string `json:"name,omitempty"`
	Breed               *string `json:"breed,omitempty"`
	Size                *string `json:"size,omitempty"`
	Age                 *int    `json:"age,omitempty"`
	Category            *string `json:"category,omitempty"`
	SpecialNeeds        *string `json:"special_needs,omitempty"`
	PickupLocation      *string `json:"pickup_location,omitempty"`
	WalkRoute           *string `json:"walk_route,omitempty"`
	WalkDuration        *int    `json:"walk_duration,omitempty"`
	SpecialInstructions *string `json:"special_instructions,omitempty"`
	DefaultMorningTime  *string `json:"default_morning_time,omitempty"`
	DefaultEveningTime  *string `json:"default_evening_time,omitempty"`
}

// ToggleAvailabilityRequest represents the request to toggle dog availability
type ToggleAvailabilityRequest struct {
	IsAvailable       bool    `json:"is_available"`
	UnavailableReason *string `json:"unavailable_reason,omitempty"`
}

// DogFilterRequest represents dog filtering parameters
type DogFilterRequest struct {
	Breed       *string `json:"breed,omitempty"`
	Size        *string `json:"size,omitempty"`
	MinAge      *int    `json:"min_age,omitempty"`
	MaxAge      *int    `json:"max_age,omitempty"`
	Category    *string `json:"category,omitempty"`
	Available   *bool   `json:"available,omitempty"`
	Search      *string `json:"search,omitempty"` // Search in name, breed
}
