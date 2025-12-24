package handlers

import (
	"testing"
)

func TestGetColumnValue(t *testing.T) {
	tests := []struct {
		name     string
		row      []string
		colIndex string
		want     string
	}{
		{
			name:     "valid index",
			row:      []string{"Bella", "Labrador", "3 Jahre"},
			colIndex: "0",
			want:     "Bella",
		},
		{
			name:     "valid index middle",
			row:      []string{"Bella", "Labrador", "3 Jahre"},
			colIndex: "1",
			want:     "Labrador",
		},
		{
			name:     "valid index last",
			row:      []string{"Bella", "Labrador", "3 Jahre"},
			colIndex: "2",
			want:     "3 Jahre",
		},
		{
			name:     "empty column index",
			row:      []string{"Bella", "Labrador", "3 Jahre"},
			colIndex: "",
			want:     "",
		},
		{
			name:     "index out of range",
			row:      []string{"Bella", "Labrador"},
			colIndex: "5",
			want:     "",
		},
		{
			name:     "negative index",
			row:      []string{"Bella", "Labrador"},
			colIndex: "-1",
			want:     "",
		},
		{
			name:     "invalid index",
			row:      []string{"Bella", "Labrador"},
			colIndex: "abc",
			want:     "",
		},
		{
			name:     "value with whitespace",
			row:      []string{"  Bella  ", "Labrador"},
			colIndex: "0",
			want:     "Bella",
		},
		{
			name:     "empty row",
			row:      []string{},
			colIndex: "0",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getColumnValue(tt.row, tt.colIndex)
			if got != tt.want {
				t.Errorf("getColumnValue() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Small variations
		{name: "klein", input: "klein", want: "small"},
		{name: "Klein uppercase", input: "Klein", want: "small"},
		{name: "small", input: "small", want: "small"},
		{name: "s", input: "s", want: "small"},
		{name: "S uppercase", input: "S", want: "small"},

		// Medium variations
		{name: "mittel", input: "mittel", want: "medium"},
		{name: "Mittel uppercase", input: "Mittel", want: "medium"},
		{name: "medium", input: "medium", want: "medium"},
		{name: "m", input: "m", want: "medium"},
		{name: "M uppercase", input: "M", want: "medium"},

		// Large variations
		{name: "gross", input: "gross", want: "large"},
		{name: "groß", input: "groß", want: "large"},
		{name: "Groß uppercase", input: "Groß", want: "large"},
		{name: "large", input: "large", want: "large"},
		{name: "l", input: "l", want: "large"},
		{name: "L uppercase", input: "L", want: "large"},

		// Invalid/empty
		{name: "empty string", input: "", want: ""},
		{name: "unknown value", input: "huge", want: ""},
		{name: "whitespace", input: "   ", want: ""},

		// With whitespace
		{name: "klein with whitespace", input: "  klein  ", want: "small"},
		{name: "medium with whitespace", input: "\tmedium\t", want: "medium"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSize(tt.input)
			if got != tt.want {
				t.Errorf("parseSize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseAge(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		// Simple numbers
		{name: "simple number", input: "3", want: 3, wantErr: false},
		{name: "two digits", input: "12", want: 12, wantErr: false},

		// German format
		{name: "Jahre suffix", input: "3 Jahre", want: 3, wantErr: false},
		{name: "Jahr suffix", input: "1 Jahr", want: 1, wantErr: false},

		// English format
		{name: "years suffix", input: "5 years", want: 5, wantErr: false},
		{name: "year suffix", input: "1 year", want: 1, wantErr: false},

		// With whitespace
		{name: "whitespace", input: "  4 Jahre  ", want: 4, wantErr: false},

		// Edge cases
		{name: "zero", input: "0", want: 0, wantErr: false},
		{name: "large number", input: "15", want: 15, wantErr: false},

		// Invalid cases
		{name: "empty string", input: "", want: 0, wantErr: true},
		{name: "no numbers", input: "Jahre", want: 0, wantErr: true},
		{name: "only text", input: "alt", want: 0, wantErr: true},

		// BUG 2 RED PHASE: These should fail until bug is fixed
		{name: "negative age should error", input: "-3 years", want: 0, wantErr: true},
		{name: "unrealistic age should error", input: "999 years", want: 0, wantErr: true},
		{name: "mixed letters numbers should error", input: "a1b2c3", want: 0, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAge(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAge(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseAge(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseAvailable(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// German positive values
		{name: "ja", input: "ja", want: true},
		{name: "Ja uppercase", input: "Ja", want: true},
		{name: "JA caps", input: "JA", want: true},
		{name: "aktiv", input: "aktiv", want: true},
		{name: "verfügbar", input: "verfügbar", want: true},

		// English positive values
		{name: "yes", input: "yes", want: true},
		{name: "true", input: "true", want: true},
		{name: "active", input: "active", want: true},
		{name: "available", input: "available", want: true},

		// Numeric positive
		{name: "1", input: "1", want: true},

		// German negative values
		{name: "nein", input: "nein", want: false},
		{name: "Nein uppercase", input: "Nein", want: false},
		{name: "inaktiv", input: "inaktiv", want: false},

		// English negative values
		{name: "no", input: "no", want: false},
		{name: "false", input: "false", want: false},
		{name: "inactive", input: "inactive", want: false},

		// Numeric negative
		{name: "0", input: "0", want: false},

		// Default to true
		{name: "empty string defaults to true", input: "", want: true},
		{name: "unknown value defaults to true", input: "maybe", want: true},

		// With whitespace
		{name: "ja with whitespace", input: "  ja  ", want: true},
		{name: "nein with whitespace", input: "\tnein\t", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAvailable(tt.input)
			if got != tt.want {
				t.Errorf("parseAvailable(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestFieldMapping_Fields(t *testing.T) {
	mapping := FieldMapping{
		Name:                "0",
		Breed:               "1",
		Age:                 "2",
		Size:                "3",
		Color:               "4",
		SpecialInstructions: "5",
		PickupLocation:      "6",
		IsAvailable:         "7",
	}

	if mapping.Name != "0" {
		t.Errorf("Name = %s, want 0", mapping.Name)
	}
	if mapping.Breed != "1" {
		t.Errorf("Breed = %s, want 1", mapping.Breed)
	}
	if mapping.Age != "2" {
		t.Errorf("Age = %s, want 2", mapping.Age)
	}
	if mapping.Size != "3" {
		t.Errorf("Size = %s, want 3", mapping.Size)
	}
	if mapping.Color != "4" {
		t.Errorf("Color = %s, want 4", mapping.Color)
	}
	if mapping.SpecialInstructions != "5" {
		t.Errorf("SpecialInstructions = %s, want 5", mapping.SpecialInstructions)
	}
	if mapping.PickupLocation != "6" {
		t.Errorf("PickupLocation = %s, want 6", mapping.PickupLocation)
	}
	if mapping.IsAvailable != "7" {
		t.Errorf("IsAvailable = %s, want 7", mapping.IsAvailable)
	}
}

func TestImportPreview_Fields(t *testing.T) {
	preview := ImportPreview{
		Headers:    []string{"Name", "Breed", "Age"},
		SampleRows: [][]string{{"Bella", "Labrador", "3"}},
		TotalRows:  10,
		Mapping:    map[string]string{"0": "name"},
		Suggestions: map[string]string{
			"name":  "0",
			"breed": "1",
		},
	}

	if len(preview.Headers) != 3 {
		t.Errorf("Headers len = %d, want 3", len(preview.Headers))
	}
	if len(preview.SampleRows) != 1 {
		t.Errorf("SampleRows len = %d, want 1", len(preview.SampleRows))
	}
	if preview.TotalRows != 10 {
		t.Errorf("TotalRows = %d, want 10", preview.TotalRows)
	}
	if len(preview.Suggestions) != 2 {
		t.Errorf("Suggestions len = %d, want 2", len(preview.Suggestions))
	}
}

func TestImportResult_Fields(t *testing.T) {
	result := ImportResult{
		TotalRows: 100,
		Imported:  95,
		Skipped:   5,
		Errors:    []string{"Row 5: Name missing", "Row 10: Invalid size"},
	}

	if result.TotalRows != 100 {
		t.Errorf("TotalRows = %d, want 100", result.TotalRows)
	}
	if result.Imported != 95 {
		t.Errorf("Imported = %d, want 95", result.Imported)
	}
	if result.Skipped != 5 {
		t.Errorf("Skipped = %d, want 5", result.Skipped)
	}
	if len(result.Errors) != 2 {
		t.Errorf("Errors len = %d, want 2", len(result.Errors))
	}
}

func TestImportHandler_suggestMappings(t *testing.T) {
	h := &ImportHandler{}

	tests := []struct {
		name     string
		headers  []string
		expected map[string]string // field -> expected column index
	}{
		{
			name:    "German headers",
			headers: []string{"Name", "Rasse", "Alter", "Größe", "Farbe", "Hinweis", "Standort", "Verfügbar"},
			expected: map[string]string{
				"name":                 "0",
				"breed":                "1",
				"age":                  "2",
				"size":                 "3",
				"color":                "4",
				"special_instructions": "5",
				"pickup_location":      "6",
				"is_available":         "7",
			},
		},
		{
			name:    "English headers",
			headers: []string{"Name", "Breed", "Age", "Size", "Color", "Special Notes", "Location", "Available"},
			expected: map[string]string{
				"name":                 "0",
				"breed":                "1",
				"age":                  "2",
				"size":                 "3",
				"color":                "4",
				"special_instructions": "5",
				"pickup_location":      "6",
				"is_available":         "7",
			},
		},
		{
			name:    "Mixed case headers",
			headers: []string{"NAME", "Breed", "alter", "GROESSE"},
			expected: map[string]string{
				"name":  "0",
				"breed": "1",
				"age":   "2",
				"size":  "3",
			},
		},
		{
			name:     "Empty headers",
			headers:  []string{},
			expected: map[string]string{},
		},
		{
			name:    "Partial match - Hundename",
			headers: []string{"Hundename", "ID"},
			expected: map[string]string{
				"name": "0",
			},
		},
		{
			name:    "Partial match - Kategorie for color",
			headers: []string{"ID", "Kategorie"},
			expected: map[string]string{
				"color": "1",
			},
		},
		// BUG 5 RED PHASE: First match should win, not be overwritten by later matches
		{
			name:    "BUG5: First name match should win",
			headers: []string{"Hundename", "Tiername", "Bezeichnung"},
			expected: map[string]string{
				"name": "0", // "Hundename" matches first, should NOT be overwritten by "Tiername" or "Bezeichnung"
			},
		},
		{
			name:    "BUG5: First breed match should win",
			headers: []string{"Rasse", "Tierart"},
			expected: map[string]string{
				"breed": "0", // "Rasse" matches first
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := h.suggestMappings(tt.headers, 0)

			for field, expectedIdx := range tt.expected {
				if suggestions[field] != expectedIdx {
					t.Errorf("suggestMappings()[%s] = %q, want %q", field, suggestions[field], expectedIdx)
				}
			}
		})
	}
}
