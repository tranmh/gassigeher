package handlers

import (
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

// SafeFileServer wraps http.FileServer with security checks
// BUG FIX #4: Prevents null byte injection and path traversal attacks
func SafeFileServer(root http.FileSystem) http.Handler {
	fileServer := http.FileServer(root)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate the path before serving
		if !ValidateFilePath(r.URL.Path) {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// URL decode the path to catch encoded attacks
		decodedPath, err := url.PathUnescape(r.URL.Path)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Validate the decoded path as well
		if !ValidateFilePath(decodedPath) {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Additional check: ensure the path is clean and doesn't escape root
		cleanPath := filepath.Clean(decodedPath)
		if strings.HasPrefix(cleanPath, "..") || strings.Contains(cleanPath, "..") {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Serve the file
		fileServer.ServeHTTP(w, r)
	})
}

// ValidateFilePath checks if a file path is safe (no null bytes, path traversal, etc.)
// BUG FIX #4: Comprehensive file path validation
func ValidateFilePath(path string) bool {
	// Check for null bytes (primary Bug #4 fix)
	if strings.ContainsRune(path, 0) {
		return false
	}

	// Check for URL-encoded null bytes
	if strings.Contains(path, "%00") {
		return false
	}

	// Check for double URL-encoded null bytes
	if strings.Contains(path, "%2500") {
		return false
	}

	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		return false
	}

	// Check for URL-encoded path traversal
	lowerPath := strings.ToLower(path)
	if strings.Contains(lowerPath, "%2e%2e") || strings.Contains(lowerPath, "%252e") {
		return false
	}

	// Check for backslash (Windows-style path traversal)
	if strings.Contains(path, "\\") {
		return false
	}

	// Check for URL-encoded backslash
	if strings.Contains(lowerPath, "%5c") || strings.Contains(lowerPath, "%255c") {
		return false
	}

	return true
}
