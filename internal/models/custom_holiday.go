package models

import (
	"fmt"
	"time"
)

type CustomHoliday struct {
	ID        int       `json:"id"`
	Date      string    `json:"date"` // YYYY-MM-DD
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	Source    string    `json:"source"` // 'api' or 'admin'
	CreatedAt time.Time `json:"created_at"`
	CreatedBy *int      `json:"created_by,omitempty"` // Admin user ID
}

func (h *CustomHoliday) Validate() error {
	if h.Date == "" {
		return fmt.Errorf("date is required")
	}

	// Validate date format
	_, err := time.Parse("2006-01-02", h.Date)
	if err != nil {
		return fmt.Errorf("date must be in YYYY-MM-DD format")
	}

	if h.Name == "" {
		return fmt.Errorf("name is required")
	}

	if h.Source != "api" && h.Source != "admin" {
		return fmt.Errorf("source must be 'api' or 'admin'")
	}

	return nil
}
