package repository

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// TestFormatTimestamp verifies the FormatTimestamp helper produces clean RFC3339 strings
func TestFormatTimestamp(t *testing.T) {
	now := time.Now()
	formatted := FormatTimestamp(now)

	// Should not contain monotonic clock suffix
	if strings.Contains(formatted, " m=") {
		t.Errorf("FormatTimestamp contains monotonic suffix: %s", formatted)
	}

	// Should not contain timezone name like "CET"
	if strings.Contains(formatted, "CET") || strings.Contains(formatted, "CEST") {
		t.Errorf("FormatTimestamp contains timezone name: %s", formatted)
	}

	// Should be valid RFC3339 format
	_, err := time.Parse(time.RFC3339, formatted)
	if err != nil {
		t.Errorf("FormatTimestamp is not valid RFC3339: %s, error: %v", formatted, err)
	}
}

// TestFormatTimestampPtr handles nil pointer
func TestFormatTimestampPtr(t *testing.T) {
	// Nil pointer should return nil
	result := FormatTimestampPtr(nil)
	if result != nil {
		t.Errorf("FormatTimestampPtr(nil) should return nil, got %v", result)
	}

	// Non-nil pointer should return formatted string
	now := time.Now()
	result = FormatTimestampPtr(&now)
	if result == nil {
		t.Error("FormatTimestampPtr(&now) should not return nil")
	}
	if strings.Contains(*result, " m=") {
		t.Errorf("FormatTimestampPtr contains monotonic suffix: %s", *result)
	}
}

// TestTimestampStoredCorrectly verifies timestamps stored via helper can be read back
func TestTimestampStoredCorrectly(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE ts_test (id INTEGER PRIMARY KEY, created_at TEXT)`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert using FormatTimestamp
	now := time.Now()
	formatted := FormatTimestamp(now)
	_, err = db.Exec("INSERT INTO ts_test (created_at) VALUES (?)", formatted)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Read back as string - should be clean
	var storedValue string
	err = db.QueryRow("SELECT created_at FROM ts_test WHERE id = 1").Scan(&storedValue)
	if err != nil {
		t.Fatalf("Failed to read string: %v", err)
	}

	if strings.Contains(storedValue, " m=") {
		t.Errorf("Stored timestamp has monotonic suffix: %s", storedValue)
	}

	// Verify we can parse it back
	_, err = time.Parse(time.RFC3339, storedValue)
	if err != nil {
		t.Errorf("Stored timestamp not valid RFC3339: %s", storedValue)
	}
}

// TestParseTimestamp verifies parsing of various timestamp formats
func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"RFC3339", "2025-12-24T14:30:00+01:00", false},
		{"RFC3339 UTC", "2025-12-24T13:30:00Z", false},
		{"RFC3339 Nano", "2025-12-24T14:30:00.123456789+01:00", false},
		// Legacy format with monotonic clock (backward compatibility)
		{"Legacy with monotonic", "2025-12-24 14:30:00.123456789 +0100 CET m=+123.456", false},
		{"Invalid", "not-a-timestamp", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseTimestamp(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTimestamp(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}
