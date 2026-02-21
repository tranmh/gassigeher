package models

import (
	"testing"
)

func TestCreateRecurringBookingRequest_Validate(t *testing.T) {
	intPtr := func(v int) *int { return &v }
	strPtr := func(v string) *string { return &v }

	tests := []struct {
		name    string
		req     CreateRecurringBookingRequest
		wantErr bool
	}{
		{
			name: "Valid weekly request with weeks",
			req: CreateRecurringBookingRequest{
				DogID:          1,
				RecurrenceType: "weekly",
				DayOfWeek:      intPtr(1), // Monday
				ScheduledTime:  "09:00",
				StartDate:      "2026-03-01",
				Weeks:          intPtr(4),
			},
			wantErr: false,
		},
		{
			name: "Valid weekly request with end date",
			req: CreateRecurringBookingRequest{
				DogID:          1,
				RecurrenceType: "weekly",
				DayOfWeek:      intPtr(3), // Wednesday
				ScheduledTime:  "14:30",
				StartDate:      "2026-03-01",
				EndDate:        strPtr("2026-04-01"),
			},
			wantErr: false,
		},
		{
			name: "Valid interval request",
			req: CreateRecurringBookingRequest{
				DogID:          1,
				RecurrenceType: "interval",
				IntervalDays:   intPtr(14),
				ScheduledTime:  "10:00",
				StartDate:      "2026-03-01",
				Weeks:          intPtr(8),
			},
			wantErr: false,
		},
		{
			name: "Valid with excluded dates",
			req: CreateRecurringBookingRequest{
				DogID:          1,
				RecurrenceType: "weekly",
				DayOfWeek:      intPtr(2),
				ScheduledTime:  "09:00",
				StartDate:      "2026-03-01",
				Weeks:          intPtr(4),
				ExcludedDates:  []string{"2026-03-08", "2026-03-15"},
			},
			wantErr: false,
		},
		{
			name: "Missing dog ID",
			req: CreateRecurringBookingRequest{
				RecurrenceType: "weekly",
				DayOfWeek:      intPtr(1),
				ScheduledTime:  "09:00",
				StartDate:      "2026-03-01",
				Weeks:          intPtr(4),
			},
			wantErr: true,
		},
		{
			name: "Invalid recurrence type",
			req: CreateRecurringBookingRequest{
				DogID:          1,
				RecurrenceType: "daily",
				ScheduledTime:  "09:00",
				StartDate:      "2026-03-01",
				Weeks:          intPtr(4),
			},
			wantErr: true,
		},
		{
			name: "Weekly without day_of_week",
			req: CreateRecurringBookingRequest{
				DogID:          1,
				RecurrenceType: "weekly",
				ScheduledTime:  "09:00",
				StartDate:      "2026-03-01",
				Weeks:          intPtr(4),
			},
			wantErr: true,
		},
		{
			name: "Weekly with invalid day_of_week (7)",
			req: CreateRecurringBookingRequest{
				DogID:          1,
				RecurrenceType: "weekly",
				DayOfWeek:      intPtr(7),
				ScheduledTime:  "09:00",
				StartDate:      "2026-03-01",
				Weeks:          intPtr(4),
			},
			wantErr: true,
		},
		{
			name: "Weekly with negative day_of_week",
			req: CreateRecurringBookingRequest{
				DogID:          1,
				RecurrenceType: "weekly",
				DayOfWeek:      intPtr(-1),
				ScheduledTime:  "09:00",
				StartDate:      "2026-03-01",
				Weeks:          intPtr(4),
			},
			wantErr: true,
		},
		{
			name: "Interval without interval_days",
			req: CreateRecurringBookingRequest{
				DogID:          1,
				RecurrenceType: "interval",
				ScheduledTime:  "09:00",
				StartDate:      "2026-03-01",
				Weeks:          intPtr(4),
			},
			wantErr: true,
		},
		{
			name: "Interval with 0 days",
			req: CreateRecurringBookingRequest{
				DogID:          1,
				RecurrenceType: "interval",
				IntervalDays:   intPtr(0),
				ScheduledTime:  "09:00",
				StartDate:      "2026-03-01",
				Weeks:          intPtr(4),
			},
			wantErr: true,
		},
		{
			name: "Interval with 91 days (too many)",
			req: CreateRecurringBookingRequest{
				DogID:          1,
				RecurrenceType: "interval",
				IntervalDays:   intPtr(91),
				ScheduledTime:  "09:00",
				StartDate:      "2026-03-01",
				Weeks:          intPtr(4),
			},
			wantErr: true,
		},
		{
			name: "Missing scheduled time",
			req: CreateRecurringBookingRequest{
				DogID:          1,
				RecurrenceType: "weekly",
				DayOfWeek:      intPtr(1),
				StartDate:      "2026-03-01",
				Weeks:          intPtr(4),
			},
			wantErr: true,
		},
		{
			name: "Invalid scheduled time format",
			req: CreateRecurringBookingRequest{
				DogID:          1,
				RecurrenceType: "weekly",
				DayOfWeek:      intPtr(1),
				ScheduledTime:  "25:00",
				StartDate:      "2026-03-01",
				Weeks:          intPtr(4),
			},
			wantErr: true,
		},
		{
			name: "Missing start date",
			req: CreateRecurringBookingRequest{
				DogID:          1,
				RecurrenceType: "weekly",
				DayOfWeek:      intPtr(1),
				ScheduledTime:  "09:00",
				Weeks:          intPtr(4),
			},
			wantErr: true,
		},
		{
			name: "Invalid start date format",
			req: CreateRecurringBookingRequest{
				DogID:          1,
				RecurrenceType: "weekly",
				DayOfWeek:      intPtr(1),
				ScheduledTime:  "09:00",
				StartDate:      "01-03-2026",
				Weeks:          intPtr(4),
			},
			wantErr: true,
		},
		{
			name: "Neither end date nor weeks",
			req: CreateRecurringBookingRequest{
				DogID:          1,
				RecurrenceType: "weekly",
				DayOfWeek:      intPtr(1),
				ScheduledTime:  "09:00",
				StartDate:      "2026-03-01",
			},
			wantErr: true,
		},
		{
			name: "End date before start date",
			req: CreateRecurringBookingRequest{
				DogID:          1,
				RecurrenceType: "weekly",
				DayOfWeek:      intPtr(1),
				ScheduledTime:  "09:00",
				StartDate:      "2026-03-01",
				EndDate:        strPtr("2026-02-01"),
			},
			wantErr: true,
		},
		{
			name: "End date same as start date",
			req: CreateRecurringBookingRequest{
				DogID:          1,
				RecurrenceType: "weekly",
				DayOfWeek:      intPtr(1),
				ScheduledTime:  "09:00",
				StartDate:      "2026-03-01",
				EndDate:        strPtr("2026-03-01"),
			},
			wantErr: true,
		},
		{
			name: "Invalid end date format",
			req: CreateRecurringBookingRequest{
				DogID:          1,
				RecurrenceType: "weekly",
				DayOfWeek:      intPtr(1),
				ScheduledTime:  "09:00",
				StartDate:      "2026-03-01",
				EndDate:        strPtr("not-a-date"),
			},
			wantErr: true,
		},
		{
			name: "Weeks too low (0)",
			req: CreateRecurringBookingRequest{
				DogID:          1,
				RecurrenceType: "weekly",
				DayOfWeek:      intPtr(1),
				ScheduledTime:  "09:00",
				StartDate:      "2026-03-01",
				Weeks:          intPtr(0),
			},
			wantErr: true,
		},
		{
			name: "Weeks too high (53)",
			req: CreateRecurringBookingRequest{
				DogID:          1,
				RecurrenceType: "weekly",
				DayOfWeek:      intPtr(1),
				ScheduledTime:  "09:00",
				StartDate:      "2026-03-01",
				Weeks:          intPtr(53),
			},
			wantErr: true,
		},
		{
			name: "Invalid excluded date format",
			req: CreateRecurringBookingRequest{
				DogID:          1,
				RecurrenceType: "weekly",
				DayOfWeek:      intPtr(1),
				ScheduledTime:  "09:00",
				StartDate:      "2026-03-01",
				Weeks:          intPtr(4),
				ExcludedDates:  []string{"bad-date"},
			},
			wantErr: true,
		},
		{
			name: "Day of week Sunday (0) is valid",
			req: CreateRecurringBookingRequest{
				DogID:          1,
				RecurrenceType: "weekly",
				DayOfWeek:      intPtr(0),
				ScheduledTime:  "09:00",
				StartDate:      "2026-03-01",
				Weeks:          intPtr(4),
			},
			wantErr: false,
		},
		{
			name: "Day of week Saturday (6) is valid",
			req: CreateRecurringBookingRequest{
				DogID:          1,
				RecurrenceType: "weekly",
				DayOfWeek:      intPtr(6),
				ScheduledTime:  "09:00",
				StartDate:      "2026-03-01",
				Weeks:          intPtr(4),
			},
			wantErr: false,
		},
		{
			name: "Interval 1 day is valid",
			req: CreateRecurringBookingRequest{
				DogID:          1,
				RecurrenceType: "interval",
				IntervalDays:   intPtr(1),
				ScheduledTime:  "09:00",
				StartDate:      "2026-03-01",
				Weeks:          intPtr(1),
			},
			wantErr: false,
		},
		{
			name: "Interval 90 days is valid",
			req: CreateRecurringBookingRequest{
				DogID:          1,
				RecurrenceType: "interval",
				IntervalDays:   intPtr(90),
				ScheduledTime:  "09:00",
				StartDate:      "2026-03-01",
				Weeks:          intPtr(52),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRecurringBookingPreviewRequest_Validate(t *testing.T) {
	intPtr := func(v int) *int { return &v }

	// Valid preview request
	req := &RecurringBookingPreviewRequest{
		DogID:          1,
		RecurrenceType: "weekly",
		DayOfWeek:      intPtr(1),
		ScheduledTime:  "09:00",
		StartDate:      "2026-03-01",
		Weeks:          intPtr(4),
	}
	if err := req.Validate(); err != nil {
		t.Errorf("Valid preview request should not error, got: %v", err)
	}

	// Invalid preview request
	reqInvalid := &RecurringBookingPreviewRequest{
		DogID:          0,
		RecurrenceType: "weekly",
		ScheduledTime:  "09:00",
		StartDate:      "2026-03-01",
		Weeks:          intPtr(4),
	}
	if err := reqInvalid.Validate(); err == nil {
		t.Error("Invalid preview request (dog_id=0) should error")
	}
}

func TestCreateRecurringBookingRequest_ComputeEndDate(t *testing.T) {
	intPtr := func(v int) *int { return &v }
	strPtr := func(v string) *string { return &v }

	tests := []struct {
		name     string
		req      CreateRecurringBookingRequest
		expected string
	}{
		{
			name: "With explicit end date",
			req: CreateRecurringBookingRequest{
				StartDate: "2026-03-01",
				EndDate:   strPtr("2026-04-15"),
				Weeks:     intPtr(8), // should be ignored
			},
			expected: "2026-04-15",
		},
		{
			name: "Computed from weeks (4 weeks)",
			req: CreateRecurringBookingRequest{
				StartDate: "2026-03-01",
				Weeks:     intPtr(4),
			},
			expected: "2026-03-29",
		},
		{
			name: "Computed from weeks (8 weeks)",
			req: CreateRecurringBookingRequest{
				StartDate: "2026-01-01",
				Weeks:     intPtr(8),
			},
			expected: "2026-02-26",
		},
		{
			name: "Computed from weeks (1 week)",
			req: CreateRecurringBookingRequest{
				StartDate: "2026-06-15",
				Weeks:     intPtr(1),
			},
			expected: "2026-06-22",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.req.ComputeEndDate()
			if result != tt.expected {
				t.Errorf("ComputeEndDate() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGenerateRecurringDates_Weekly(t *testing.T) {
	intPtr := func(v int) *int { return &v }

	tests := []struct {
		name      string
		dayOfWeek int
		startDate string
		endDate   string
		expected  []string
	}{
		{
			name:      "Mondays for 4 weeks starting Mon",
			dayOfWeek: 1, // Monday
			startDate: "2026-03-02",
			endDate:   "2026-03-30",
			expected:  []string{"2026-03-02", "2026-03-09", "2026-03-16", "2026-03-23", "2026-03-30"},
		},
		{
			name:      "Tuesdays starting on a Monday",
			dayOfWeek: 2, // Tuesday
			startDate: "2026-03-02",
			endDate:   "2026-03-24",
			expected:  []string{"2026-03-03", "2026-03-10", "2026-03-17", "2026-03-24"},
		},
		{
			name:      "Sunday (0) in March",
			dayOfWeek: 0, // Sunday
			startDate: "2026-03-01",
			endDate:   "2026-03-31",
			expected:  []string{"2026-03-01", "2026-03-08", "2026-03-15", "2026-03-22", "2026-03-29"},
		},
		{
			name:      "Saturday start on Thursday",
			dayOfWeek: 6, // Saturday
			startDate: "2026-03-05",
			endDate:   "2026-03-22",
			expected:  []string{"2026-03-07", "2026-03-14", "2026-03-21"},
		},
		{
			name:      "End date too soon for any match",
			dayOfWeek: 5, // Friday
			startDate: "2026-03-02",
			endDate:   "2026-03-05",
			expected:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dow := tt.dayOfWeek
			dates, err := GenerateRecurringDates("weekly", intPtr(dow), nil, tt.startDate, tt.endDate)
			if err != nil {
				t.Fatalf("GenerateRecurringDates() error: %v", err)
			}

			if len(dates) != len(tt.expected) {
				t.Fatalf("Expected %d dates, got %d: %v", len(tt.expected), len(dates), dates)
			}

			for i, d := range dates {
				if d != tt.expected[i] {
					t.Errorf("Date[%d] = %q, want %q", i, d, tt.expected[i])
				}
			}
		})
	}
}

func TestGenerateRecurringDates_Interval(t *testing.T) {
	intPtr := func(v int) *int { return &v }

	tests := []struct {
		name         string
		intervalDays int
		startDate    string
		endDate      string
		expected     []string
	}{
		{
			name:         "Every 7 days",
			intervalDays: 7,
			startDate:    "2026-03-01",
			endDate:      "2026-03-22",
			expected:     []string{"2026-03-01", "2026-03-08", "2026-03-15", "2026-03-22"},
		},
		{
			name:         "Every 14 days",
			intervalDays: 14,
			startDate:    "2026-03-01",
			endDate:      "2026-04-12",
			expected:     []string{"2026-03-01", "2026-03-15", "2026-03-29", "2026-04-12"},
		},
		{
			name:         "Every 3 days",
			intervalDays: 3,
			startDate:    "2026-03-01",
			endDate:      "2026-03-10",
			expected:     []string{"2026-03-01", "2026-03-04", "2026-03-07", "2026-03-10"},
		},
		{
			name:         "Single occurrence (end before next)",
			intervalDays: 30,
			startDate:    "2026-03-01",
			endDate:      "2026-03-15",
			expected:     []string{"2026-03-01"},
		},
		{
			name:         "Daily interval",
			intervalDays: 1,
			startDate:    "2026-03-01",
			endDate:      "2026-03-03",
			expected:     []string{"2026-03-01", "2026-03-02", "2026-03-03"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interval := tt.intervalDays
			dates, err := GenerateRecurringDates("interval", nil, intPtr(interval), tt.startDate, tt.endDate)
			if err != nil {
				t.Fatalf("GenerateRecurringDates() error: %v", err)
			}

			if len(dates) != len(tt.expected) {
				t.Fatalf("Expected %d dates, got %d: %v", len(tt.expected), len(dates), dates)
			}

			for i, d := range dates {
				if d != tt.expected[i] {
					t.Errorf("Date[%d] = %q, want %q", i, d, tt.expected[i])
				}
			}
		})
	}
}

func TestGenerateRecurringDates_Errors(t *testing.T) {
	intPtr := func(v int) *int { return &v }

	// Unknown recurrence type
	_, err := GenerateRecurringDates("monthly", intPtr(1), nil, "2026-03-01", "2026-04-01")
	if err == nil {
		t.Error("Expected error for unknown recurrence type")
	}

	// Weekly without day_of_week
	_, err = GenerateRecurringDates("weekly", nil, nil, "2026-03-01", "2026-04-01")
	if err == nil {
		t.Error("Expected error for weekly without day_of_week")
	}

	// Interval without interval_days
	_, err = GenerateRecurringDates("interval", nil, nil, "2026-03-01", "2026-04-01")
	if err == nil {
		t.Error("Expected error for interval without interval_days")
	}

	// Invalid start date
	_, err = GenerateRecurringDates("weekly", intPtr(1), nil, "bad-date", "2026-04-01")
	if err == nil {
		t.Error("Expected error for invalid start date")
	}

	// Invalid end date
	_, err = GenerateRecurringDates("weekly", intPtr(1), nil, "2026-03-01", "bad-date")
	if err == nil {
		t.Error("Expected error for invalid end date")
	}
}
