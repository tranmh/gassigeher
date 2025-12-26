package repository

import (
	"fmt"
	"strings"
	"time"
)

// FormatTimestamp formats a time.Time value for storage in the database.
// It ensures the timestamp is in RFC3339 format without the monotonic clock suffix
// that Go adds internally to time.Time values.
//
// Example output: "2025-12-24T14:30:00+01:00"
func FormatTimestamp(t time.Time) string {
	return t.Format(time.RFC3339)
}

// FormatTimestampPtr formats a *time.Time value, returning nil if the input is nil.
func FormatTimestampPtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

// ParseTimestamp parses a timestamp string from the database.
// It handles multiple formats for backward compatibility:
// - RFC3339: "2025-12-24T14:30:00+01:00"
// - RFC3339Nano: "2025-12-24T14:30:00.123456789+01:00"
// - Legacy Go format with monotonic clock: "2025-12-24 14:30:00.123456789 +0100 CET m=+123.456"
func ParseTimestamp(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}

	// Try RFC3339 first (most common after fix)
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	// Try RFC3339Nano
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}

	// Handle legacy format with monotonic clock suffix
	// Format: "2025-12-24 14:30:00.123456789 +0100 CET m=+123.456"
	if strings.Contains(s, " m=") {
		// Remove the monotonic clock suffix
		parts := strings.Split(s, " m=")
		cleaned := parts[0]

		// Try parsing the cleaned string
		// Format: "2025-12-24 14:30:00.123456789 +0100 CET"
		layouts := []string{
			"2006-01-02 15:04:05.999999999 -0700 MST",
			"2006-01-02 15:04:05.999999999 -0700",
			"2006-01-02 15:04:05 -0700 MST",
			"2006-01-02 15:04:05 -0700",
		}

		for _, layout := range layouts {
			if t, err := time.Parse(layout, cleaned); err == nil {
				return t, nil
			}
		}
	}

	// Try some other common formats
	additionalLayouts := []string{
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, layout := range additionalLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse timestamp: %s", s)
}

// ParseTimestampPtr parses a timestamp string pointer, returning nil if the input is nil or empty.
func ParseTimestampPtr(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := ParseTimestamp(*s)
	if err != nil {
		return nil
	}
	return &t
}

// NowFormatted returns the current time formatted for database storage.
func NowFormatted() string {
	return FormatTimestamp(time.Now())
}
