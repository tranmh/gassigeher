package models

import (
	"regexp"
	"time"
)

// ColorCategory represents a configurable dog color category
type ColorCategory struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	HexCode     string    `json:"hex_code"`
	PatternIcon *string   `json:"pattern_icon,omitempty"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateColorCategoryRequest represents a request to create a color category
type CreateColorCategoryRequest struct {
	Name        string  `json:"name"`
	HexCode     string  `json:"hex_code"`
	PatternIcon *string `json:"pattern_icon,omitempty"`
}

// UpdateColorCategoryRequest represents a request to update a color category
type UpdateColorCategoryRequest struct {
	Name        *string `json:"name,omitempty"`
	HexCode     *string `json:"hex_code,omitempty"`
	PatternIcon *string `json:"pattern_icon,omitempty"`
	SortOrder   *int    `json:"sort_order,omitempty"`
}

// hexCodeRegex validates hex color codes in format #XXXXXX
var hexCodeRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// Validate validates the create color category request
func (r *CreateColorCategoryRequest) Validate() error {
	if r.Name == "" {
		return &ValidationError{Field: "name", Message: "Name ist erforderlich"}
	}

	if r.HexCode == "" {
		return &ValidationError{Field: "hex_code", Message: "Farbcode ist erforderlich"}
	}

	if !hexCodeRegex.MatchString(r.HexCode) {
		return &ValidationError{Field: "hex_code", Message: "Farbcode muss im Format #XXXXXX sein"}
	}

	return nil
}

// Validate validates the update color category request
func (r *UpdateColorCategoryRequest) Validate() error {
	if r.HexCode != nil && *r.HexCode != "" {
		if !hexCodeRegex.MatchString(*r.HexCode) {
			return &ValidationError{Field: "hex_code", Message: "Farbcode muss im Format #XXXXXX sein"}
		}
	}

	return nil
}
