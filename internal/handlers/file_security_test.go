package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// GREEN PHASE: Tests for Bug #4 - Null byte injection in file paths
// These tests verify the security fix is working

// TestSafeFileServer_NullByteInjection tests that null bytes in paths are rejected
func TestSafeFileServer_NullByteInjection(t *testing.T) {
	// Create a temp directory with a test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.jpg")
	if err := os.WriteFile(testFile, []byte("test image content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create the safe file server
	handler := SafeFileServer(http.Dir(tempDir))

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{
			name:           "URL-encoded null byte in path should return 400",
			path:           "/test.jpg%00.txt",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "URL-encoded null byte at start should return 400",
			path:           "/%00test.jpg",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "URL-encoded null byte alone should return 400",
			path:           "/%00",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "double URL-encoded null byte should return 400",
			path:           "/test%00%00.jpg",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "valid file path should return 200",
			path:           "/test.jpg",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "non-existent file should return 404",
			path:           "/nonexistent.jpg",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("SafeFileServer(%q) status = %d, want %d", tt.path, rec.Code, tt.expectedStatus)
			}
		})
	}
}

// TestSafeFileServer_PathTraversal tests that path traversal attempts are blocked
func TestSafeFileServer_PathTraversal(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.jpg")
	if err := os.WriteFile(testFile, []byte("test image content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	handler := SafeFileServer(http.Dir(tempDir))

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{
			name:           "path traversal with .. should be blocked",
			path:           "/../etc/passwd",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "encoded path traversal should be blocked",
			path:           "/%2e%2e/etc/passwd",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "double encoded should be blocked",
			path:           "/%252e%252e/etc/passwd",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "backslash traversal should be blocked",
			path:           "/..\\etc\\passwd",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("SafeFileServer(%q) status = %d, want %d", tt.path, rec.Code, tt.expectedStatus)
			}
		})
	}
}

// TestValidateFilePath tests the file path validation function
func TestValidateFilePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		isValid bool
	}{
		{
			name:    "normal file path",
			path:    "/images/dog.jpg",
			isValid: true,
		},
		{
			name:    "path with spaces",
			path:    "/images/my dog.jpg",
			isValid: true,
		},
		{
			name:    "null byte injection",
			path:    "/images/dog.jpg\x00.txt",
			isValid: false,
		},
		{
			name:    "path traversal",
			path:    "/images/../../../etc/passwd",
			isValid: false,
		},
		{
			name:    "backslash",
			path:    "/images\\..\\passwd",
			isValid: false,
		},
		{
			name:    "encoded null byte",
			path:    "/images/dog%00.jpg",
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateFilePath(tt.path)
			if result != tt.isValid {
				t.Errorf("ValidateFilePath(%q) = %v, want %v", tt.path, result, tt.isValid)
			}
		})
	}
}
