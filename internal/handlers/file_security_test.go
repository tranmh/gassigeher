package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

// TestSafeFileServer_Security_NoDirectoryListing tests that directory listing is blocked
// SECURITY: GASSI-2025-005 - Prevent information disclosure via directory listing
func TestSafeFileServer_Security_NoDirectoryListing(t *testing.T) {
	// Create a temp directory structure
	tempDir := t.TempDir()

	// Create a subdirectory with files
	subDir := filepath.Join(tempDir, "dogs")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create test files in subdirectory
	testFile1 := filepath.Join(subDir, "dog1.jpg")
	testFile2 := filepath.Join(subDir, "dog2.jpg")
	if err := os.WriteFile(testFile1, []byte("dog1 image"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(testFile2, []byte("dog2 image"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	handler := SafeFileServer(http.Dir(tempDir))

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		description    string
	}{
		{
			name:           "directory_with_trailing_slash_blocked",
			path:           "/dogs/",
			expectedStatus: http.StatusForbidden,
			description:    "Directory listing with trailing slash should be blocked",
		},
		{
			name:           "directory_without_trailing_slash_blocked",
			path:           "/dogs",
			expectedStatus: http.StatusForbidden,
			description:    "Directory listing without trailing slash should be blocked",
		},
		{
			name:           "root_directory_blocked",
			path:           "/",
			expectedStatus: http.StatusForbidden,
			description:    "Root directory listing should be blocked",
		},
		{
			name:           "file_in_subdirectory_allowed",
			path:           "/dogs/dog1.jpg",
			expectedStatus: http.StatusOK,
			description:    "Individual files should still be accessible",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("SECURITY VIOLATION: %s\nSafeFileServer(%q) status = %d, want %d",
					tt.description, tt.path, rec.Code, tt.expectedStatus)
			}

			// Extra check: ensure no file listing is returned for directory requests
			if tt.expectedStatus == http.StatusForbidden {
				body := rec.Body.String()
				if strings.Contains(body, "dog1.jpg") || strings.Contains(body, "dog2.jpg") {
					t.Errorf("SECURITY VIOLATION: Directory listing exposed file names in response body")
				}
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
