package models

import (
	"testing"
)

// TestCreateColorCategoryRequest_Validate tests validation for color category creation
func TestCreateColorCategoryRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateColorCategoryRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request with all fields",
			req: CreateColorCategoryRequest{
				Name:        "test-color",
				HexCode:     "#ff5500",
				PatternIcon: stringPtr("circle"),
			},
			wantErr: false,
		},
		{
			name: "valid request without pattern icon",
			req: CreateColorCategoryRequest{
				Name:    "another-color",
				HexCode: "#123456",
			},
			wantErr: false,
		},
		{
			name: "valid hex code - uppercase",
			req: CreateColorCategoryRequest{
				Name:    "color",
				HexCode: "#AABBCC",
			},
			wantErr: false,
		},
		{
			name: "invalid - empty name",
			req: CreateColorCategoryRequest{
				Name:    "",
				HexCode: "#ff5500",
			},
			wantErr: true,
			errMsg:  "Name ist erforderlich",
		},
		{
			name: "invalid - empty hex code",
			req: CreateColorCategoryRequest{
				Name:    "test",
				HexCode: "",
			},
			wantErr: true,
			errMsg:  "Farbcode ist erforderlich",
		},
		{
			name: "invalid hex code - missing hash",
			req: CreateColorCategoryRequest{
				Name:    "test",
				HexCode: "ff5500",
			},
			wantErr: true,
			errMsg:  "Farbcode muss im Format #XXXXXX sein",
		},
		{
			name: "invalid hex code - too short",
			req: CreateColorCategoryRequest{
				Name:    "test",
				HexCode: "#fff",
			},
			wantErr: true,
			errMsg:  "Farbcode muss im Format #XXXXXX sein",
		},
		{
			name: "invalid hex code - too long",
			req: CreateColorCategoryRequest{
				Name:    "test",
				HexCode: "#ff55001",
			},
			wantErr: true,
			errMsg:  "Farbcode muss im Format #XXXXXX sein",
		},
		{
			name: "invalid hex code - invalid characters",
			req: CreateColorCategoryRequest{
				Name:    "test",
				HexCode: "#gggggg",
			},
			wantErr: true,
			errMsg:  "Farbcode muss im Format #XXXXXX sein",
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

// TestUpdateColorCategoryRequest_Validate tests validation for color category updates
func TestUpdateColorCategoryRequest_Validate(t *testing.T) {
	validName := "updated-name"
	validHex := "#aabbcc"
	invalidHex := "invalid"

	tests := []struct {
		name    string
		req     UpdateColorCategoryRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid - update name only",
			req: UpdateColorCategoryRequest{
				Name: &validName,
			},
			wantErr: false,
		},
		{
			name: "valid - update hex code only",
			req: UpdateColorCategoryRequest{
				HexCode: &validHex,
			},
			wantErr: false,
		},
		{
			name: "valid - update both",
			req: UpdateColorCategoryRequest{
				Name:    &validName,
				HexCode: &validHex,
			},
			wantErr: false,
		},
		{
			name: "valid - empty request (no updates)",
			req:     UpdateColorCategoryRequest{},
			wantErr: false,
		},
		{
			name: "invalid - bad hex code format",
			req: UpdateColorCategoryRequest{
				HexCode: &invalidHex,
			},
			wantErr: true,
			errMsg:  "Farbcode muss im Format #XXXXXX sein",
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
