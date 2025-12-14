package handlers

import (
	"io"
	"net/http"
	"path/filepath"
	"strings"
)

// AllowedImageTypes defines valid MIME types for image uploads
var AllowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
}

// AllowedImageExtensions defines valid file extensions for image uploads
var AllowedImageExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
}

// ValidateImageFile validates both file extension and MIME type (magic bytes)
// Returns nil if valid, error message string if invalid
func ValidateImageFile(filename string, fileReader io.Reader) (string, bool) {
	// First check extension
	ext := strings.ToLower(filepath.Ext(filename))
	if !AllowedImageExtensions[ext] {
		return "Only JPEG and PNG files are allowed", false
	}

	// Read first 512 bytes to detect MIME type
	header := make([]byte, 512)
	n, err := fileReader.Read(header)
	if err != nil && err != io.EOF {
		return "Failed to read file", false
	}

	// Detect content type from magic bytes
	contentType := http.DetectContentType(header[:n])

	// Check if detected type matches allowed types
	if !AllowedImageTypes[contentType] {
		return "File content does not match a valid image type", false
	}

	return "", true
}

// ValidateImageMIMEType validates only the MIME type (for use when extension already checked)
func ValidateImageMIMEType(fileReader io.Reader) (string, bool) {
	// Read first 512 bytes to detect MIME type
	header := make([]byte, 512)
	n, err := fileReader.Read(header)
	if err != nil && err != io.EOF {
		return "Failed to read file", false
	}

	// Detect content type from magic bytes
	contentType := http.DetectContentType(header[:n])

	// Check if detected type matches allowed types
	if !AllowedImageTypes[contentType] {
		return "File content does not match a valid image type", false
	}

	return "", true
}
