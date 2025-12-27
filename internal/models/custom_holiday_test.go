package models

import (
	"testing"
)

func TestCustomHoliday_Validate(t *testing.T) {
	t.Run("valid holiday", func(t *testing.T) {
		h := CustomHoliday{
			Date:   "2025-12-25",
			Name:   "Weihnachten",
			Source: "admin",
		}
		if err := h.Validate(); err != nil {
			t.Errorf("Validate() error = %v, want nil", err)
		}
	})

	t.Run("valid api source", func(t *testing.T) {
		h := CustomHoliday{
			Date:   "2025-01-01",
			Name:   "Neujahr",
			Source: "api",
		}
		if err := h.Validate(); err != nil {
			t.Errorf("Validate() error = %v, want nil", err)
		}
	})

	t.Run("empty date", func(t *testing.T) {
		h := CustomHoliday{
			Date:   "",
			Name:   "Test",
			Source: "admin",
		}
		if err := h.Validate(); err == nil {
			t.Error("Validate() expected error for empty date")
		}
	})

	t.Run("invalid date format", func(t *testing.T) {
		invalidDates := []string{
			"25-12-2025",      // DD-MM-YYYY
			"12/25/2025",      // MM/DD/YYYY
			"2025-25-12",      // Invalid month
			"2025-12-32",      // Invalid day
			"not-a-date",      // Not a date
			"20251225",        // No separators
			"2025-1-25",       // Single digit month
			"2025-12-5",       // Single digit day
		}

		for _, date := range invalidDates {
			t.Run(date, func(t *testing.T) {
				h := CustomHoliday{
					Date:   date,
					Name:   "Test",
					Source: "admin",
				}
				if err := h.Validate(); err == nil {
					t.Errorf("Validate() expected error for invalid date format: %s", date)
				}
			})
		}
	})

	t.Run("empty name", func(t *testing.T) {
		h := CustomHoliday{
			Date:   "2025-12-25",
			Name:   "",
			Source: "admin",
		}
		if err := h.Validate(); err == nil {
			t.Error("Validate() expected error for empty name")
		}
	})

	t.Run("invalid source", func(t *testing.T) {
		invalidSources := []string{"", "user", "system", "API", "ADMIN"}
		for _, source := range invalidSources {
			t.Run(source, func(t *testing.T) {
				h := CustomHoliday{
					Date:   "2025-12-25",
					Name:   "Test",
					Source: source,
				}
				if err := h.Validate(); err == nil {
					t.Errorf("Validate() expected error for invalid source: %s", source)
				}
			})
		}
	})
}
