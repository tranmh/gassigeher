package models

import (
	"testing"
)

// TestCreateColorRequestRequest_Validate tests validation for color request creation
func TestCreateColorRequestRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateColorRequestRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			req: CreateColorRequestRequest{
				ColorID: 1,
			},
			wantErr: false,
		},
		{
			name: "valid request - higher color ID",
			req: CreateColorRequestRequest{
				ColorID: 100,
			},
			wantErr: false,
		},
		{
			name: "invalid - zero color ID",
			req: CreateColorRequestRequest{
				ColorID: 0,
			},
			wantErr: true,
			errMsg:  "Farb-ID ist erforderlich",
		},
		{
			name: "invalid - negative color ID",
			req: CreateColorRequestRequest{
				ColorID: -1,
			},
			wantErr: true,
			errMsg:  "Farb-ID ist erforderlich",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, expected to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

// TestReviewColorRequestRequest_Validate tests validation for reviewing color requests
func TestReviewColorRequestRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     ReviewColorRequestRequest
		wantErr bool
	}{
		{
			name: "approved without message",
			req: ReviewColorRequestRequest{
				Approved: true,
				Message:  nil,
			},
			wantErr: false,
		},
		{
			name: "approved with message",
			req: ReviewColorRequestRequest{
				Approved: true,
				Message:  stringPtr("Willkommen!"),
			},
			wantErr: false,
		},
		{
			name: "denied without message",
			req: ReviewColorRequestRequest{
				Approved: false,
				Message:  nil,
			},
			wantErr: false,
		},
		{
			name: "denied with message",
			req: ReviewColorRequestRequest{
				Approved: false,
				Message:  stringPtr("Bitte erst Einweisung absolvieren"),
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
