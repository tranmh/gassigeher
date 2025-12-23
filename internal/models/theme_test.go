package models

import (
	"testing"
)

// TestGetThemePresetNames tests getting all preset names
func TestGetThemePresetNames(t *testing.T) {
	names := GetThemePresetNames()

	// Should have all presets
	if len(names) != len(ThemePresets) {
		t.Errorf("Expected %d preset names, got %d", len(ThemePresets), len(names))
	}

	// All returned names should be valid
	for _, name := range names {
		if _, ok := ThemePresets[name]; !ok {
			t.Errorf("Returned name %q is not a valid preset", name)
		}
	}
}

// TestGetThemePreset tests getting a specific preset
func TestGetThemePreset(t *testing.T) {
	t.Run("returns correct preset for valid name", func(t *testing.T) {
		colors := GetThemePreset("ocean")
		expected := ThemePresets["ocean"]
		if colors.Primary != expected.Primary {
			t.Errorf("Expected Primary %s, got %s", expected.Primary, colors.Primary)
		}
	})

	t.Run("returns classic for invalid name", func(t *testing.T) {
		colors := GetThemePreset("nonexistent")
		classic := ThemePresets["classic"]
		if colors.Primary != classic.Primary {
			t.Errorf("Expected classic Primary %s, got %s", classic.Primary, colors.Primary)
		}
	})

	t.Run("returns classic for empty name", func(t *testing.T) {
		colors := GetThemePreset("")
		classic := ThemePresets["classic"]
		if colors.Primary != classic.Primary {
			t.Errorf("Expected classic Primary %s, got %s", classic.Primary, colors.Primary)
		}
	})
}

// TestIsValidPreset tests preset validation
func TestIsValidPreset(t *testing.T) {
	tests := []struct {
		name   string
		preset string
		want   bool
	}{
		{"valid classic", "classic", true},
		{"valid ocean", "ocean", true},
		{"valid forest", "forest", true},
		{"invalid preset", "invalid", false},
		{"empty string", "", false},
		{"case sensitive", "Classic", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidPreset(tt.preset); got != tt.want {
				t.Errorf("IsValidPreset(%q) = %v, want %v", tt.preset, got, tt.want)
			}
		})
	}
}

// TestGetAllPresetInfo tests getting all preset info
func TestGetAllPresetInfo(t *testing.T) {
	presets := GetAllPresetInfo()

	t.Run("returns correct number of presets", func(t *testing.T) {
		// The order list has 10 presets defined
		if len(presets) != 10 {
			t.Errorf("Expected 10 presets, got %d", len(presets))
		}
	})

	t.Run("returns presets in defined order", func(t *testing.T) {
		expectedOrder := []string{"classic", "ocean", "forest", "sunset", "lavender", "coral", "midnight", "emerald", "rose", "slate"}
		for i, expected := range expectedOrder {
			if presets[i].Name != expected {
				t.Errorf("Expected preset %d to be %q, got %q", i, expected, presets[i].Name)
			}
		}
	})

	t.Run("each preset has correct colors", func(t *testing.T) {
		for _, preset := range presets {
			expected := ThemePresets[preset.Name]
			if preset.Colors.Primary != expected.Primary {
				t.Errorf("Preset %q: expected Primary %s, got %s", preset.Name, expected.Primary, preset.Colors.Primary)
			}
		}
	})
}

// TestThemePresets tests the theme presets map
func TestThemePresets(t *testing.T) {
	// Verify all expected presets exist
	expectedPresets := []string{
		"classic", "ocean", "forest", "sunset", "lavender",
		"coral", "midnight", "emerald", "rose", "slate",
	}

	for _, name := range expectedPresets {
		t.Run(name+" preset exists", func(t *testing.T) {
			colors, ok := ThemePresets[name]
			if !ok {
				t.Errorf("Expected preset %q to exist", name)
				return
			}

			// Verify all color fields are non-empty
			if colors.Primary == "" {
				t.Errorf("Preset %q: Primary color is empty", name)
			}
			if colors.Secondary == "" {
				t.Errorf("Preset %q: Secondary color is empty", name)
			}
			if colors.Accent == "" {
				t.Errorf("Preset %q: Accent color is empty", name)
			}
			if colors.Background == "" {
				t.Errorf("Preset %q: Background color is empty", name)
			}
			if colors.Text == "" {
				t.Errorf("Preset %q: Text color is empty", name)
			}

			// Verify colors are valid hex format
			if !ValidateHexColor(colors.Primary) {
				t.Errorf("Preset %q: Primary color %q is invalid hex", name, colors.Primary)
			}
		})
	}
}
