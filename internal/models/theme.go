package models

// GetThemePresetNames returns a list of all available theme preset names
func GetThemePresetNames() []string {
	names := make([]string, 0, len(ThemePresets))
	for name := range ThemePresets {
		names = append(names, name)
	}
	return names
}

// GetThemePreset returns a theme preset by name, or the classic preset if not found
func GetThemePreset(name string) ThemeColors {
	if colors, ok := ThemePresets[name]; ok {
		return colors
	}
	return ThemePresets["classic"]
}

// IsValidPreset checks if a preset name is valid
func IsValidPreset(name string) bool {
	_, ok := ThemePresets[name]
	return ok
}

// ThemePresetInfo contains metadata about a theme preset
type ThemePresetInfo struct {
	Name   string      `json:"name"`
	Colors ThemeColors `json:"colors"`
}

// GetAllPresetInfo returns all theme presets with their metadata
func GetAllPresetInfo() []ThemePresetInfo {
	presets := make([]ThemePresetInfo, 0, len(ThemePresets))
	// Return in a consistent order
	order := []string{"classic", "ocean", "forest", "sunset", "lavender", "coral", "midnight", "emerald", "rose", "slate"}
	for _, name := range order {
		if colors, ok := ThemePresets[name]; ok {
			presets = append(presets, ThemePresetInfo{
				Name:   name,
				Colors: colors,
			})
		}
	}
	return presets
}
