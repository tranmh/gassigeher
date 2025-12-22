package services

import (
	"testing"
)

// TestValidateS3Path tests path validation to prevent path traversal attacks
func TestValidateS3Path(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantValid bool
	}{
		// Valid paths
		{name: "simple filename", path: "photo.jpg", wantValid: true},
		{name: "nested path", path: "dogs/photo.jpg", wantValid: true},
		{name: "deeply nested", path: "uploads/2025/01/photo.jpg", wantValid: true},

		// Invalid paths - path traversal attempts
		{name: "parent directory", path: "../photo.jpg", wantValid: false},
		{name: "double parent", path: "../../photo.jpg", wantValid: false},
		{name: "nested traversal", path: "dogs/../../../photo.jpg", wantValid: false},
		{name: "absolute path", path: "/etc/passwd", wantValid: false},
		{name: "hidden path traversal", path: "dogs/..\\photo.jpg", wantValid: false},
		{name: "null byte", path: "photo.jpg\x00.txt", wantValid: false},
		{name: "empty path", path: "", wantValid: false},
		{name: "only dots", path: "..", wantValid: false},
		{name: "double slash", path: "dogs//photo.jpg", wantValid: true}, // This is OK, just redundant
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateS3Path(tt.path)
			gotValid := err == nil

			if gotValid != tt.wantValid {
				if tt.wantValid {
					t.Errorf("validateS3Path(%q) returned error %v, want valid", tt.path, err)
				} else {
					t.Errorf("validateS3Path(%q) returned nil, want error (path traversal should be rejected)", tt.path)
				}
			}
		})
	}
}

// TestS3ServiceUploadPathValidation tests that Upload rejects path traversal
func TestS3ServiceUploadPathValidation(t *testing.T) {
	// We can't test the actual Upload without mocking S3, but we can test
	// that the validation is applied by testing the validateS3Path function
	// which should be called by Upload

	// This test ensures the validation function exists and works
	err := validateS3Path("../../../malicious.txt")
	if err == nil {
		t.Error("Expected path traversal to be rejected, but it was allowed")
	}
}
