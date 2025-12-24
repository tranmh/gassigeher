package handlers

import (
	"regexp"
	"strings"
)

// Input length limits for dog fields
const (
	MaxDogNameLength               = 100
	MaxDogBreedLength              = 100
	MaxDogSpecialNeedsLength       = 1000
	MaxDogPickupLocationLength     = 500
	MaxDogWalkRouteLength          = 1000
	MaxDogSpecialInstructionsLength = 1000
	MaxDogExternalLinkLength       = 500
)

// ValidationError represents a field validation error
type ValidationError struct {
	Field   string
	Message string
}

// htmlTagRegex matches HTML tags including script, img, etc.
var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

// StripHTMLTags removes all HTML tags from a string, preserving the text content
func StripHTMLTags(s string) string {
	return htmlTagRegex.ReplaceAllString(s, "")
}

// SanitizeString removes HTML tags and trims whitespace
func SanitizeString(s string) string {
	// Strip HTML tags
	s = StripHTMLTags(s)
	// Trim whitespace
	s = strings.TrimSpace(s)
	return s
}

// SanitizeStringPtr sanitizes a string pointer, returns nil if input is nil
func SanitizeStringPtr(s *string) *string {
	if s == nil {
		return nil
	}
	sanitized := SanitizeString(*s)
	return &sanitized
}

// ValidateDogName validates dog name length and sanitizes XSS
func ValidateDogName(name string) (string, *ValidationError) {
	sanitized := SanitizeString(name)
	if len(sanitized) == 0 {
		return "", &ValidationError{Field: "name", Message: "Name is required"}
	}
	if len(sanitized) > MaxDogNameLength {
		return "", &ValidationError{Field: "name", Message: "Name must be 100 characters or less"}
	}
	return sanitized, nil
}

// ValidateDogBreed validates dog breed length and sanitizes XSS
func ValidateDogBreed(breed string) (string, *ValidationError) {
	sanitized := SanitizeString(breed)
	if len(sanitized) == 0 {
		return "", &ValidationError{Field: "breed", Message: "Breed is required"}
	}
	if len(sanitized) > MaxDogBreedLength {
		return "", &ValidationError{Field: "breed", Message: "Breed must be 100 characters or less"}
	}
	return sanitized, nil
}

// ValidateDogSpecialNeeds validates and sanitizes special_needs field
func ValidateDogSpecialNeeds(needs *string) (*string, *ValidationError) {
	if needs == nil {
		return nil, nil
	}
	sanitized := SanitizeString(*needs)
	if len(sanitized) > MaxDogSpecialNeedsLength {
		return nil, &ValidationError{Field: "special_needs", Message: "Special needs must be 1000 characters or less"}
	}
	if sanitized == "" {
		return nil, nil
	}
	return &sanitized, nil
}

// ValidateDogPickupLocation validates and sanitizes pickup_location field
func ValidateDogPickupLocation(location *string) (*string, *ValidationError) {
	if location == nil {
		return nil, nil
	}
	sanitized := SanitizeString(*location)
	if len(sanitized) > MaxDogPickupLocationLength {
		return nil, &ValidationError{Field: "pickup_location", Message: "Pickup location must be 500 characters or less"}
	}
	if sanitized == "" {
		return nil, nil
	}
	return &sanitized, nil
}

// ValidateDogWalkRoute validates and sanitizes walk_route field
func ValidateDogWalkRoute(route *string) (*string, *ValidationError) {
	if route == nil {
		return nil, nil
	}
	sanitized := SanitizeString(*route)
	if len(sanitized) > MaxDogWalkRouteLength {
		return nil, &ValidationError{Field: "walk_route", Message: "Walk route must be 1000 characters or less"}
	}
	if sanitized == "" {
		return nil, nil
	}
	return &sanitized, nil
}

// ValidateDogSpecialInstructions validates and sanitizes special_instructions field
func ValidateDogSpecialInstructions(instructions *string) (*string, *ValidationError) {
	if instructions == nil {
		return nil, nil
	}
	sanitized := SanitizeString(*instructions)
	if len(sanitized) > MaxDogSpecialInstructionsLength {
		return nil, &ValidationError{Field: "special_instructions", Message: "Special instructions must be 1000 characters or less"}
	}
	if sanitized == "" {
		return nil, nil
	}
	return &sanitized, nil
}

// ValidateDogExternalLink validates and sanitizes external_link field
func ValidateDogExternalLink(link *string) (*string, *ValidationError) {
	if link == nil {
		return nil, nil
	}
	sanitized := SanitizeString(*link)
	if len(sanitized) > MaxDogExternalLinkLength {
		return nil, &ValidationError{Field: "external_link", Message: "External link must be 500 characters or less"}
	}
	if sanitized == "" {
		return nil, nil
	}
	return &sanitized, nil
}

// Tenant registration validation constants
const (
	MaxOrganizationNameLength = 200
	MaxPersonNameLength       = 100
)

// ValidateOrganizationName validates and sanitizes organization name (XSS prevention)
func ValidateOrganizationName(name string) (string, *ValidationError) {
	sanitized := SanitizeString(name)
	if len(sanitized) == 0 {
		return "", &ValidationError{Field: "organization_name", Message: "Organisationsname ist erforderlich"}
	}
	if len(sanitized) > MaxOrganizationNameLength {
		return "", &ValidationError{Field: "organization_name", Message: "Organisationsname darf maximal 200 Zeichen haben"}
	}
	return sanitized, nil
}

// ValidatePersonName validates and sanitizes a person's name (XSS prevention)
func ValidatePersonName(name string, fieldName string) (string, *ValidationError) {
	sanitized := SanitizeString(name)
	if len(sanitized) == 0 {
		return "", &ValidationError{Field: fieldName, Message: fieldName + " ist erforderlich"}
	}
	if len(sanitized) > MaxPersonNameLength {
		return "", &ValidationError{Field: fieldName, Message: fieldName + " darf maximal 100 Zeichen haben"}
	}
	return sanitized, nil
}
