package models

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

// Dog validation constants
const (
	DogAgeMin = 0
	DogAgeMax = 30
)

// ValidDogSizes contains valid dog size values
var ValidDogSizes = []string{"small", "medium", "large"}

// ValidDogCategories contains valid legacy dog categories
var ValidDogCategories = []string{"green", "orange", "blue"}

// timeFormatRegex validates HH:MM format
var timeFormatRegex = regexp.MustCompile(`^([01]?[0-9]|2[0-3]):[0-5][0-9]$`)

// Dog represents a dog in the system
type Dog struct {
	ID                   int            `json:"id"`
	TenantID             int            `json:"tenant_id"` // SaaS: Tenant this dog belongs to (0 = Simple-Mode default)
	Name                 string         `json:"name"`
	Breed                string         `json:"breed"`
	Size                 string         `json:"size"` // small, medium, large
	Age                  int            `json:"age"`
	Category             string         `json:"category"` // green, blue, orange (legacy, use color_id)
	ColorID              *int           `json:"color_id,omitempty"`
	Color                *ColorCategory `json:"color,omitempty"`
	Photo                *string        `json:"photo,omitempty"`
	PhotoThumbnail       *string        `json:"photo_thumbnail,omitempty"`
	SpecialNeeds         *string        `json:"special_needs,omitempty"`
	PickupLocation       *string        `json:"pickup_location,omitempty"`
	WalkRoute            *string        `json:"walk_route,omitempty"`
	WalkDuration         *int           `json:"walk_duration,omitempty"` // minutes
	SpecialInstructions  *string        `json:"special_instructions,omitempty"`
	DefaultMorningTime   *string        `json:"default_morning_time,omitempty"` // HH:MM format
	DefaultEveningTime   *string        `json:"default_evening_time,omitempty"` // HH:MM format
	IsAvailable          bool           `json:"is_available"`
	IsFeatured           bool           `json:"is_featured"`
	ExternalLink         *string        `json:"external_link,omitempty"`
	UnavailableReason    *string        `json:"unavailable_reason,omitempty"`
	UnavailableSince     *time.Time     `json:"unavailable_since,omitempty"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
}

// CreateDogRequest represents the request to create a dog
type CreateDogRequest struct {
	Name                string  `json:"name"`
	Breed               string  `json:"breed"`
	Size                string  `json:"size"`
	Age                 int     `json:"age"`
	Category            string  `json:"category"`
	ColorID             *int    `json:"color_id,omitempty"`
	SpecialNeeds        *string `json:"special_needs,omitempty"`
	PickupLocation      *string `json:"pickup_location,omitempty"`
	WalkRoute           *string `json:"walk_route,omitempty"`
	WalkDuration        *int    `json:"walk_duration,omitempty"`
	SpecialInstructions *string `json:"special_instructions,omitempty"`
	DefaultMorningTime  *string `json:"default_morning_time,omitempty"`
	DefaultEveningTime  *string `json:"default_evening_time,omitempty"`
	ExternalLink        *string `json:"external_link,omitempty"`
}

// UpdateDogRequest represents the request to update a dog
type UpdateDogRequest struct {
	Name                *string `json:"name,omitempty"`
	Breed               *string `json:"breed,omitempty"`
	Size                *string `json:"size,omitempty"`
	Age                 *int    `json:"age,omitempty"`
	Category            *string `json:"category,omitempty"`
	ColorID             *int    `json:"color_id,omitempty"`
	SpecialNeeds        *string `json:"special_needs,omitempty"`
	PickupLocation      *string `json:"pickup_location,omitempty"`
	WalkRoute           *string `json:"walk_route,omitempty"`
	WalkDuration        *int    `json:"walk_duration,omitempty"`
	SpecialInstructions *string `json:"special_instructions,omitempty"`
	DefaultMorningTime  *string `json:"default_morning_time,omitempty"`
	DefaultEveningTime  *string `json:"default_evening_time,omitempty"`
	ExternalLink        *string `json:"external_link,omitempty"`
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
	Category    *string `json:"category,omitempty"`    // Legacy filter (green, blue, orange)
	ColorID     *int    `json:"color_id,omitempty"`    // New color system filter
	Available   *bool   `json:"available,omitempty"`
	Search      *string `json:"search,omitempty"` // Search in name, breed
}

// isValidDogSize checks if the size is valid
func isValidDogSize(size string) bool {
	size = strings.ToLower(strings.TrimSpace(size))
	for _, valid := range ValidDogSizes {
		if size == valid {
			return true
		}
	}
	return false
}

// isValidDogCategory checks if the category is valid
func isValidDogCategory(category string) bool {
	category = strings.ToLower(strings.TrimSpace(category))
	for _, valid := range ValidDogCategories {
		if category == valid {
			return true
		}
	}
	return false
}

// isValidDogTimeFormat checks if the time is in HH:MM format (for dog.go)
func isValidDogTimeFormat(timeStr string) bool {
	return timeFormatRegex.MatchString(timeStr)
}

// Validate validates the CreateDogRequest
// SECURITY FIX: Missing validation for CreateDogRequest
func (r *CreateDogRequest) Validate() error {
	// Required field validation
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("Name ist erforderlich")
	}
	if len(r.Name) > 100 {
		return errors.New("Name darf maximal 100 Zeichen lang sein")
	}

	if strings.TrimSpace(r.Breed) == "" {
		return errors.New("Rasse ist erforderlich")
	}
	if len(r.Breed) > 100 {
		return errors.New("Rasse darf maximal 100 Zeichen lang sein")
	}

	// Size validation
	if !isValidDogSize(r.Size) {
		return errors.New("Größe muss 'small', 'medium' oder 'large' sein")
	}

	// Age validation
	if r.Age < DogAgeMin || r.Age > DogAgeMax {
		return errors.New("Alter muss zwischen 0 und 30 Jahren liegen")
	}

	// Category validation (legacy field)
	if r.Category != "" && !isValidDogCategory(r.Category) {
		return errors.New("Kategorie muss 'green', 'orange' oder 'blue' sein")
	}

	// External link URL validation
	if r.ExternalLink != nil && *r.ExternalLink != "" {
		if err := ValidateURL(*r.ExternalLink); err != nil {
			return errors.New("Ungültiger externer Link: " + err.Error())
		}
	}

	// Time format validation
	if r.DefaultMorningTime != nil && *r.DefaultMorningTime != "" {
		if !isValidDogTimeFormat(*r.DefaultMorningTime) {
			return errors.New("Morgenzeit muss im Format HH:MM sein")
		}
	}
	if r.DefaultEveningTime != nil && *r.DefaultEveningTime != "" {
		if !isValidDogTimeFormat(*r.DefaultEveningTime) {
			return errors.New("Abendzeit muss im Format HH:MM sein")
		}
	}

	// Walk duration validation
	if r.WalkDuration != nil && (*r.WalkDuration < 0 || *r.WalkDuration > 480) {
		return errors.New("Spaziergangszeit muss zwischen 0 und 480 Minuten liegen")
	}

	// Text field length limits
	if r.SpecialNeeds != nil && len(*r.SpecialNeeds) > 1000 {
		return errors.New("Besondere Bedürfnisse darf maximal 1000 Zeichen lang sein")
	}
	if r.PickupLocation != nil && len(*r.PickupLocation) > 500 {
		return errors.New("Abholort darf maximal 500 Zeichen lang sein")
	}
	if r.WalkRoute != nil && len(*r.WalkRoute) > 1000 {
		return errors.New("Laufroute darf maximal 1000 Zeichen lang sein")
	}
	if r.SpecialInstructions != nil && len(*r.SpecialInstructions) > 2000 {
		return errors.New("Besondere Anweisungen darf maximal 2000 Zeichen lang sein")
	}

	return nil
}

// Validate validates the UpdateDogRequest
// SECURITY FIX: Missing validation for UpdateDogRequest
func (r *UpdateDogRequest) Validate() error {
	// Name validation (if provided)
	if r.Name != nil {
		if strings.TrimSpace(*r.Name) == "" {
			return errors.New("Name darf nicht leer sein")
		}
		if len(*r.Name) > 100 {
			return errors.New("Name darf maximal 100 Zeichen lang sein")
		}
	}

	// Breed validation (if provided)
	if r.Breed != nil {
		if strings.TrimSpace(*r.Breed) == "" {
			return errors.New("Rasse darf nicht leer sein")
		}
		if len(*r.Breed) > 100 {
			return errors.New("Rasse darf maximal 100 Zeichen lang sein")
		}
	}

	// Size validation (if provided)
	if r.Size != nil && !isValidDogSize(*r.Size) {
		return errors.New("Größe muss 'small', 'medium' oder 'large' sein")
	}

	// Age validation (if provided)
	if r.Age != nil && (*r.Age < DogAgeMin || *r.Age > DogAgeMax) {
		return errors.New("Alter muss zwischen 0 und 30 Jahren liegen")
	}

	// Category validation (if provided)
	if r.Category != nil && *r.Category != "" && !isValidDogCategory(*r.Category) {
		return errors.New("Kategorie muss 'green', 'orange' oder 'blue' sein")
	}

	// External link URL validation
	if r.ExternalLink != nil && *r.ExternalLink != "" {
		if err := ValidateURL(*r.ExternalLink); err != nil {
			return errors.New("Ungültiger externer Link: " + err.Error())
		}
	}

	// Time format validation
	if r.DefaultMorningTime != nil && *r.DefaultMorningTime != "" {
		if !isValidDogTimeFormat(*r.DefaultMorningTime) {
			return errors.New("Morgenzeit muss im Format HH:MM sein")
		}
	}
	if r.DefaultEveningTime != nil && *r.DefaultEveningTime != "" {
		if !isValidDogTimeFormat(*r.DefaultEveningTime) {
			return errors.New("Abendzeit muss im Format HH:MM sein")
		}
	}

	// Walk duration validation
	if r.WalkDuration != nil && (*r.WalkDuration < 0 || *r.WalkDuration > 480) {
		return errors.New("Spaziergangszeit muss zwischen 0 und 480 Minuten liegen")
	}

	// Text field length limits
	if r.SpecialNeeds != nil && len(*r.SpecialNeeds) > 1000 {
		return errors.New("Besondere Bedürfnisse darf maximal 1000 Zeichen lang sein")
	}
	if r.PickupLocation != nil && len(*r.PickupLocation) > 500 {
		return errors.New("Abholort darf maximal 500 Zeichen lang sein")
	}
	if r.WalkRoute != nil && len(*r.WalkRoute) > 1000 {
		return errors.New("Laufroute darf maximal 1000 Zeichen lang sein")
	}
	if r.SpecialInstructions != nil && len(*r.SpecialInstructions) > 2000 {
		return errors.New("Besondere Anweisungen darf maximal 2000 Zeichen lang sein")
	}

	return nil
}
