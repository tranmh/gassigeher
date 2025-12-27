package models

import (
	"strings"
	"testing"
)

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestIsValidDogSize(t *testing.T) {
	validSizes := []string{"small", "medium", "large", "SMALL", "Medium", "LARGE", " small ", " medium ", " large "}
	for _, size := range validSizes {
		t.Run(size, func(t *testing.T) {
			if !isValidDogSize(size) {
				t.Errorf("isValidDogSize(%q) = false, want true", size)
			}
		})
	}

	invalidSizes := []string{"", "tiny", "huge", "extra-large", "xl", "xs", "123", "smalll"}
	for _, size := range invalidSizes {
		t.Run(size, func(t *testing.T) {
			if isValidDogSize(size) {
				t.Errorf("isValidDogSize(%q) = true, want false", size)
			}
		})
	}
}

func TestIsValidDogCategory(t *testing.T) {
	validCategories := []string{"green", "orange", "blue", "GREEN", "Orange", "BLUE", " green ", " orange ", " blue "}
	for _, cat := range validCategories {
		t.Run(cat, func(t *testing.T) {
			if !isValidDogCategory(cat) {
				t.Errorf("isValidDogCategory(%q) = false, want true", cat)
			}
		})
	}

	invalidCategories := []string{"", "red", "yellow", "purple", "level1", "beginner", "123"}
	for _, cat := range invalidCategories {
		t.Run(cat, func(t *testing.T) {
			if isValidDogCategory(cat) {
				t.Errorf("isValidDogCategory(%q) = true, want false", cat)
			}
		})
	}
}

func TestIsValidDogTimeFormat(t *testing.T) {
	validTimes := []string{"00:00", "09:30", "12:00", "15:45", "23:59", "9:30", "0:00"}
	for _, time := range validTimes {
		t.Run(time, func(t *testing.T) {
			if !isValidDogTimeFormat(time) {
				t.Errorf("isValidDogTimeFormat(%q) = false, want true", time)
			}
		})
	}

	invalidTimes := []string{"", "24:00", "25:00", "12:60", "12:99", "abc", "12", "12:", ":30", "1230", "12-30"}
	for _, time := range invalidTimes {
		t.Run(time, func(t *testing.T) {
			if isValidDogTimeFormat(time) {
				t.Errorf("isValidDogTimeFormat(%q) = true, want false", time)
			}
		})
	}
}

// ============================================================================
// CreateDogRequest Validation Tests
// ============================================================================

func TestCreateDogRequest_Validate(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		req := CreateDogRequest{
			Name:     "Max",
			Breed:    "Labrador",
			Size:     "medium",
			Age:      5,
			Category: "green",
		}
		if err := req.Validate(); err != nil {
			t.Errorf("Validate() error = %v, want nil", err)
		}
	})

	t.Run("empty name", func(t *testing.T) {
		req := CreateDogRequest{
			Name:     "",
			Breed:    "Labrador",
			Size:     "medium",
			Age:      5,
			Category: "green",
		}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for empty name")
		}
	})

	t.Run("whitespace only name", func(t *testing.T) {
		req := CreateDogRequest{
			Name:     "   ",
			Breed:    "Labrador",
			Size:     "medium",
			Age:      5,
			Category: "green",
		}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for whitespace-only name")
		}
	})

	t.Run("name too long", func(t *testing.T) {
		req := CreateDogRequest{
			Name:     strings.Repeat("a", 101),
			Breed:    "Labrador",
			Size:     "medium",
			Age:      5,
			Category: "green",
		}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for name > 100 chars")
		}
	})

	t.Run("empty breed", func(t *testing.T) {
		req := CreateDogRequest{
			Name:     "Max",
			Breed:    "",
			Size:     "medium",
			Age:      5,
			Category: "green",
		}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for empty breed")
		}
	})

	t.Run("breed too long", func(t *testing.T) {
		req := CreateDogRequest{
			Name:     "Max",
			Breed:    strings.Repeat("a", 101),
			Size:     "medium",
			Age:      5,
			Category: "green",
		}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for breed > 100 chars")
		}
	})

	t.Run("invalid size", func(t *testing.T) {
		req := CreateDogRequest{
			Name:     "Max",
			Breed:    "Labrador",
			Size:     "huge",
			Age:      5,
			Category: "green",
		}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for invalid size")
		}
	})

	t.Run("negative age", func(t *testing.T) {
		req := CreateDogRequest{
			Name:     "Max",
			Breed:    "Labrador",
			Size:     "medium",
			Age:      -1,
			Category: "green",
		}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for negative age")
		}
	})

	t.Run("age too high", func(t *testing.T) {
		req := CreateDogRequest{
			Name:     "Max",
			Breed:    "Labrador",
			Size:     "medium",
			Age:      31,
			Category: "green",
		}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for age > 30")
		}
	})

	t.Run("invalid category", func(t *testing.T) {
		req := CreateDogRequest{
			Name:     "Max",
			Breed:    "Labrador",
			Size:     "medium",
			Age:      5,
			Category: "red",
		}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for invalid category")
		}
	})

	t.Run("empty category allowed", func(t *testing.T) {
		req := CreateDogRequest{
			Name:  "Max",
			Breed: "Labrador",
			Size:  "medium",
			Age:   5,
		}
		if err := req.Validate(); err != nil {
			t.Errorf("Validate() error = %v, want nil for empty category", err)
		}
	})

	t.Run("invalid external link", func(t *testing.T) {
		invalidURL := "not-a-url"
		req := CreateDogRequest{
			Name:         "Max",
			Breed:        "Labrador",
			Size:         "medium",
			Age:          5,
			ExternalLink: &invalidURL,
		}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for invalid external link")
		}
	})

	t.Run("valid external link", func(t *testing.T) {
		validURL := "https://example.com/dog/max"
		req := CreateDogRequest{
			Name:         "Max",
			Breed:        "Labrador",
			Size:         "medium",
			Age:          5,
			ExternalLink: &validURL,
		}
		if err := req.Validate(); err != nil {
			t.Errorf("Validate() error = %v, want nil for valid URL", err)
		}
	})

	t.Run("invalid morning time format", func(t *testing.T) {
		invalidTime := "25:00"
		req := CreateDogRequest{
			Name:               "Max",
			Breed:              "Labrador",
			Size:               "medium",
			Age:                5,
			DefaultMorningTime: &invalidTime,
		}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for invalid morning time")
		}
	})

	t.Run("valid morning time", func(t *testing.T) {
		validTime := "09:30"
		req := CreateDogRequest{
			Name:               "Max",
			Breed:              "Labrador",
			Size:               "medium",
			Age:                5,
			DefaultMorningTime: &validTime,
		}
		if err := req.Validate(); err != nil {
			t.Errorf("Validate() error = %v, want nil", err)
		}
	})

	t.Run("invalid evening time format", func(t *testing.T) {
		invalidTime := "abc"
		req := CreateDogRequest{
			Name:               "Max",
			Breed:              "Labrador",
			Size:               "medium",
			Age:                5,
			DefaultEveningTime: &invalidTime,
		}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for invalid evening time")
		}
	})

	t.Run("walk duration negative", func(t *testing.T) {
		duration := -1
		req := CreateDogRequest{
			Name:         "Max",
			Breed:        "Labrador",
			Size:         "medium",
			Age:          5,
			WalkDuration: &duration,
		}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for negative walk duration")
		}
	})

	t.Run("walk duration too high", func(t *testing.T) {
		duration := 500
		req := CreateDogRequest{
			Name:         "Max",
			Breed:        "Labrador",
			Size:         "medium",
			Age:          5,
			WalkDuration: &duration,
		}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for walk duration > 480")
		}
	})

	t.Run("special needs too long", func(t *testing.T) {
		longText := strings.Repeat("a", 1001)
		req := CreateDogRequest{
			Name:         "Max",
			Breed:        "Labrador",
			Size:         "medium",
			Age:          5,
			SpecialNeeds: &longText,
		}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for special needs > 1000 chars")
		}
	})

	t.Run("pickup location too long", func(t *testing.T) {
		longText := strings.Repeat("a", 501)
		req := CreateDogRequest{
			Name:           "Max",
			Breed:          "Labrador",
			Size:           "medium",
			Age:            5,
			PickupLocation: &longText,
		}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for pickup location > 500 chars")
		}
	})

	t.Run("walk route too long", func(t *testing.T) {
		longText := strings.Repeat("a", 1001)
		req := CreateDogRequest{
			Name:      "Max",
			Breed:     "Labrador",
			Size:      "medium",
			Age:       5,
			WalkRoute: &longText,
		}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for walk route > 1000 chars")
		}
	})

	t.Run("special instructions too long", func(t *testing.T) {
		longText := strings.Repeat("a", 2001)
		req := CreateDogRequest{
			Name:                "Max",
			Breed:               "Labrador",
			Size:                "medium",
			Age:                 5,
			SpecialInstructions: &longText,
		}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for special instructions > 2000 chars")
		}
	})
}

// ============================================================================
// UpdateDogRequest Validation Tests
// ============================================================================

func TestUpdateDogRequest_Validate(t *testing.T) {
	t.Run("empty request is valid", func(t *testing.T) {
		req := UpdateDogRequest{}
		if err := req.Validate(); err != nil {
			t.Errorf("Validate() error = %v, want nil for empty update", err)
		}
	})

	t.Run("valid name update", func(t *testing.T) {
		name := "Bella"
		req := UpdateDogRequest{Name: &name}
		if err := req.Validate(); err != nil {
			t.Errorf("Validate() error = %v, want nil", err)
		}
	})

	t.Run("empty name", func(t *testing.T) {
		name := ""
		req := UpdateDogRequest{Name: &name}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for empty name")
		}
	})

	t.Run("whitespace name", func(t *testing.T) {
		name := "   "
		req := UpdateDogRequest{Name: &name}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for whitespace name")
		}
	})

	t.Run("name too long", func(t *testing.T) {
		name := strings.Repeat("a", 101)
		req := UpdateDogRequest{Name: &name}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for name > 100 chars")
		}
	})

	t.Run("empty breed", func(t *testing.T) {
		breed := ""
		req := UpdateDogRequest{Breed: &breed}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for empty breed")
		}
	})

	t.Run("breed too long", func(t *testing.T) {
		breed := strings.Repeat("a", 101)
		req := UpdateDogRequest{Breed: &breed}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for breed > 100 chars")
		}
	})

	t.Run("invalid size", func(t *testing.T) {
		size := "huge"
		req := UpdateDogRequest{Size: &size}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for invalid size")
		}
	})

	t.Run("valid size", func(t *testing.T) {
		size := "large"
		req := UpdateDogRequest{Size: &size}
		if err := req.Validate(); err != nil {
			t.Errorf("Validate() error = %v, want nil", err)
		}
	})

	t.Run("negative age", func(t *testing.T) {
		age := -1
		req := UpdateDogRequest{Age: &age}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for negative age")
		}
	})

	t.Run("age too high", func(t *testing.T) {
		age := 31
		req := UpdateDogRequest{Age: &age}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for age > 30")
		}
	})

	t.Run("invalid category", func(t *testing.T) {
		cat := "red"
		req := UpdateDogRequest{Category: &cat}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for invalid category")
		}
	})

	t.Run("valid category", func(t *testing.T) {
		cat := "blue"
		req := UpdateDogRequest{Category: &cat}
		if err := req.Validate(); err != nil {
			t.Errorf("Validate() error = %v, want nil", err)
		}
	})

	t.Run("empty category allowed", func(t *testing.T) {
		cat := ""
		req := UpdateDogRequest{Category: &cat}
		if err := req.Validate(); err != nil {
			t.Errorf("Validate() error = %v, want nil for empty category", err)
		}
	})

	t.Run("invalid external link", func(t *testing.T) {
		link := "not-a-url"
		req := UpdateDogRequest{ExternalLink: &link}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for invalid URL")
		}
	})

	t.Run("valid external link", func(t *testing.T) {
		link := "https://example.com/dog"
		req := UpdateDogRequest{ExternalLink: &link}
		if err := req.Validate(); err != nil {
			t.Errorf("Validate() error = %v, want nil", err)
		}
	})

	t.Run("invalid morning time", func(t *testing.T) {
		time := "25:00"
		req := UpdateDogRequest{DefaultMorningTime: &time}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for invalid morning time")
		}
	})

	t.Run("invalid evening time", func(t *testing.T) {
		time := "abc"
		req := UpdateDogRequest{DefaultEveningTime: &time}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for invalid evening time")
		}
	})

	t.Run("walk duration negative", func(t *testing.T) {
		duration := -1
		req := UpdateDogRequest{WalkDuration: &duration}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for negative duration")
		}
	})

	t.Run("walk duration too high", func(t *testing.T) {
		duration := 500
		req := UpdateDogRequest{WalkDuration: &duration}
		if err := req.Validate(); err == nil {
			t.Error("Validate() expected error for duration > 480")
		}
	})

	t.Run("text fields too long", func(t *testing.T) {
		longText1001 := strings.Repeat("a", 1001)
		longText501 := strings.Repeat("a", 501)
		longText2001 := strings.Repeat("a", 2001)

		tests := []struct {
			name string
			req  UpdateDogRequest
		}{
			{"special_needs", UpdateDogRequest{SpecialNeeds: &longText1001}},
			{"pickup_location", UpdateDogRequest{PickupLocation: &longText501}},
			{"walk_route", UpdateDogRequest{WalkRoute: &longText1001}},
			{"special_instructions", UpdateDogRequest{SpecialInstructions: &longText2001}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := tt.req.Validate(); err == nil {
					t.Errorf("Validate() expected error for %s too long", tt.name)
				}
			})
		}
	})
}

// ============================================================================
// Dog Struct Tests
// ============================================================================

func TestDogConstants(t *testing.T) {
	if DogAgeMin != 0 {
		t.Errorf("DogAgeMin = %d, want 0", DogAgeMin)
	}
	if DogAgeMax != 30 {
		t.Errorf("DogAgeMax = %d, want 30", DogAgeMax)
	}
}

func TestValidDogSizesConstant(t *testing.T) {
	expected := []string{"small", "medium", "large"}
	if len(ValidDogSizes) != len(expected) {
		t.Errorf("ValidDogSizes length = %d, want %d", len(ValidDogSizes), len(expected))
	}
	for i, s := range expected {
		if ValidDogSizes[i] != s {
			t.Errorf("ValidDogSizes[%d] = %s, want %s", i, ValidDogSizes[i], s)
		}
	}
}

func TestValidDogCategoriesConstant(t *testing.T) {
	expected := []string{"green", "orange", "blue"}
	if len(ValidDogCategories) != len(expected) {
		t.Errorf("ValidDogCategories length = %d, want %d", len(ValidDogCategories), len(expected))
	}
	for i, c := range expected {
		if ValidDogCategories[i] != c {
			t.Errorf("ValidDogCategories[%d] = %s, want %s", i, ValidDogCategories[i], c)
		}
	}
}
