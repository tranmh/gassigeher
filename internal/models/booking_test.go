package models

import (
	"testing"
)

// DONE: TestCreateBookingRequest_Validate tests CreateBookingRequest validation
func TestCreateBookingRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateBookingRequest
		wantErr bool
	}{
		{
			name: "Valid request",
			req: CreateBookingRequest{
				DogID:         1,
				Date:          "2025-12-01",
				ScheduledTime: "09:00",
			},
			wantErr: false,
		},
		{
			name: "Missing dog ID",
			req: CreateBookingRequest{
				Date:          "2025-12-01",
				ScheduledTime: "09:00",
			},
			wantErr: true,
		},
		{
			name: "Invalid date format",
			req: CreateBookingRequest{
				DogID:         1,
				Date:          "01-12-2025",
				ScheduledTime: "09:00",
			},
			wantErr: true,
		},
		{
			name: "Invalid time format",
			req: CreateBookingRequest{
				DogID:         1,
				Date:          "2025-12-01",
				ScheduledTime: "25:00",
			},
			wantErr: true,
		},
		{
			name: "Empty date",
			req: CreateBookingRequest{
				DogID:         1,
				Date:          "",
				ScheduledTime: "09:00",
			},
			wantErr: true,
		},
		{
			name: "Empty scheduled time",
			req: CreateBookingRequest{
				DogID:         1,
				Date:          "2025-12-01",
				ScheduledTime: "",
			},
			wantErr: true,
		},
		{
			name: "Missing scheduled time",
			req: CreateBookingRequest{
				DogID: 1,
				Date:  "2025-12-01",
			},
			wantErr: true,
		},
		{
			name: "Negative dog ID",
			req: CreateBookingRequest{
				DogID:         -1,
				Date:          "2025-12-01",
				ScheduledTime: "09:00",
			},
			wantErr: true,
		},
		{
			name: "Zero dog ID",
			req: CreateBookingRequest{
				DogID:         0,
				Date:          "2025-12-01",
				ScheduledTime: "09:00",
			},
			wantErr: true,
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

// TestAdminCreateBookingRequest_Validate tests AdminCreateBookingRequest validation
func TestAdminCreateBookingRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     AdminCreateBookingRequest
		wantErr bool
	}{
		{
			name: "Valid request",
			req: AdminCreateBookingRequest{
				UserID:        42,
				DogID:         1,
				Date:          "2025-12-01",
				ScheduledTime: "09:00",
			},
			wantErr: false,
		},
		{
			name: "Missing user ID",
			req: AdminCreateBookingRequest{
				DogID:         1,
				Date:          "2025-12-01",
				ScheduledTime: "09:00",
			},
			wantErr: true,
		},
		{
			name: "Zero user ID",
			req: AdminCreateBookingRequest{
				UserID:        0,
				DogID:         1,
				Date:          "2025-12-01",
				ScheduledTime: "09:00",
			},
			wantErr: true,
		},
		{
			name: "Negative user ID",
			req: AdminCreateBookingRequest{
				UserID:        -5,
				DogID:         1,
				Date:          "2025-12-01",
				ScheduledTime: "09:00",
			},
			wantErr: true,
		},
		{
			name: "Missing dog ID",
			req: AdminCreateBookingRequest{
				UserID:        42,
				Date:          "2025-12-01",
				ScheduledTime: "09:00",
			},
			wantErr: true,
		},
		{
			name: "Empty date",
			req: AdminCreateBookingRequest{
				UserID:        42,
				DogID:         1,
				Date:          "",
				ScheduledTime: "09:00",
			},
			wantErr: true,
		},
		{
			name: "Invalid date format",
			req: AdminCreateBookingRequest{
				UserID:        42,
				DogID:         1,
				Date:          "01-12-2025",
				ScheduledTime: "09:00",
			},
			wantErr: true,
		},
		{
			name: "Empty scheduled time",
			req: AdminCreateBookingRequest{
				UserID:        42,
				DogID:         1,
				Date:          "2025-12-01",
				ScheduledTime: "",
			},
			wantErr: true,
		},
		{
			name: "Invalid time format",
			req: AdminCreateBookingRequest{
				UserID:        42,
				DogID:         1,
				Date:          "2025-12-01",
				ScheduledTime: "25:00",
			},
			wantErr: true,
		},
		{
			name: "Past date is allowed (documentation use-case)",
			req: AdminCreateBookingRequest{
				UserID:        42,
				DogID:         1,
				Date:          "2020-01-01",
				ScheduledTime: "09:00",
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

// DONE: TestMoveBookingRequest_Validate tests MoveBookingRequest validation
func TestMoveBookingRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     MoveBookingRequest
		wantErr bool
	}{
		{
			name: "Valid request",
			req: MoveBookingRequest{
				Date:          "2025-12-01",
				ScheduledTime: "17:00",
				Reason:        "Dog not feeling well",
			},
			wantErr: false,
		},
		{
			name: "Valid morning walk",
			req: MoveBookingRequest{
				Date:          "2025-12-15",
				ScheduledTime: "09:30",
				Reason:        "Owner request",
			},
			wantErr: false,
		},
		{
			name: "Missing reason",
			req: MoveBookingRequest{
				Date:          "2025-12-01",
				ScheduledTime: "17:00",
			},
			wantErr: true,
		},
		{
			name: "Empty reason",
			req: MoveBookingRequest{
				Date:          "2025-12-01",
				ScheduledTime: "17:00",
				Reason:        "",
			},
			wantErr: true,
		},
		{
			name: "Invalid date",
			req: MoveBookingRequest{
				Date:          "invalid",
				ScheduledTime: "17:00",
				Reason:        "Test",
			},
			wantErr: true,
		},
		{
			name: "Empty date",
			req: MoveBookingRequest{
				Date:          "",
				ScheduledTime: "17:00",
				Reason:        "Test",
			},
			wantErr: true,
		},
		{
			name: "Invalid date format",
			req: MoveBookingRequest{
				Date:          "01-12-2025",
				ScheduledTime: "17:00",
				Reason:        "Test",
			},
			wantErr: true,
		},
		{
			name: "Empty scheduled time",
			req: MoveBookingRequest{
				Date:          "2025-12-01",
				ScheduledTime: "",
				Reason:        "Test",
			},
			wantErr: true,
		},
		{
			name: "Invalid time format",
			req: MoveBookingRequest{
				Date:          "2025-12-01",
				ScheduledTime: "25:00",
				Reason:        "Test",
			},
			wantErr: true,
		},
		{
			name: "Invalid time format 2",
			req: MoveBookingRequest{
				Date:          "2025-12-01",
				ScheduledTime: "9:00 AM",
				Reason:        "Test",
			},
			wantErr: true,
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
