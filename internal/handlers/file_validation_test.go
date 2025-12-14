package handlers

import (
	"bytes"
	"testing"
)

// JPEG magic bytes (FFD8FF)
var validJPEGHeader = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}

// PNG magic bytes (89504E47)
var validPNGHeader = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

// Plain text content (no magic bytes for image)
var plainTextContent = []byte("This is not an image file, it's plain text content.")

// HTML content (could be XSS attack vector)
var htmlContent = []byte("<html><script>alert('XSS')</script></html>")

func TestValidateImageFile_ValidJPEG(t *testing.T) {
	reader := bytes.NewReader(validJPEGHeader)
	errMsg, valid := ValidateImageFile("test.jpg", reader)
	if !valid {
		t.Errorf("Expected valid JPEG to pass, got error: %s", errMsg)
	}
}

func TestValidateImageFile_ValidPNG(t *testing.T) {
	reader := bytes.NewReader(validPNGHeader)
	errMsg, valid := ValidateImageFile("test.png", reader)
	if !valid {
		t.Errorf("Expected valid PNG to pass, got error: %s", errMsg)
	}
}

func TestValidateImageFile_TextFileWithJPGExtension(t *testing.T) {
	// This is the security test - a text file disguised as JPEG
	reader := bytes.NewReader(plainTextContent)
	errMsg, valid := ValidateImageFile("malicious.jpg", reader)
	if valid {
		t.Error("Expected text file with .jpg extension to be rejected")
	}
	if errMsg != "File content does not match a valid image type" {
		t.Errorf("Expected MIME type mismatch error, got: %s", errMsg)
	}
}

func TestValidateImageFile_HTMLFileWithJPGExtension(t *testing.T) {
	// XSS attack vector - HTML disguised as JPEG
	reader := bytes.NewReader(htmlContent)
	errMsg, valid := ValidateImageFile("xss.jpg", reader)
	if valid {
		t.Error("Expected HTML file with .jpg extension to be rejected")
	}
	if errMsg != "File content does not match a valid image type" {
		t.Errorf("Expected MIME type mismatch error, got: %s", errMsg)
	}
}

func TestValidateImageFile_InvalidExtension(t *testing.T) {
	reader := bytes.NewReader(validJPEGHeader)
	errMsg, valid := ValidateImageFile("test.exe", reader)
	if valid {
		t.Error("Expected .exe extension to be rejected")
	}
	if errMsg != "Only JPEG and PNG files are allowed" {
		t.Errorf("Expected extension error, got: %s", errMsg)
	}
}

func TestValidateImageFile_PHPFileWithJPGExtension(t *testing.T) {
	// PHP backdoor disguised as image
	phpContent := []byte("<?php system($_GET['cmd']); ?>")
	reader := bytes.NewReader(phpContent)
	errMsg, valid := ValidateImageFile("backdoor.jpg", reader)
	if valid {
		t.Error("Expected PHP file with .jpg extension to be rejected")
	}
	if errMsg != "File content does not match a valid image type" {
		t.Errorf("Expected MIME type mismatch error, got: %s", errMsg)
	}
}

func TestValidateImageMIMEType_ValidJPEG(t *testing.T) {
	reader := bytes.NewReader(validJPEGHeader)
	errMsg, valid := ValidateImageMIMEType(reader)
	if !valid {
		t.Errorf("Expected valid JPEG MIME to pass, got error: %s", errMsg)
	}
}

func TestValidateImageMIMEType_ValidPNG(t *testing.T) {
	reader := bytes.NewReader(validPNGHeader)
	errMsg, valid := ValidateImageMIMEType(reader)
	if !valid {
		t.Errorf("Expected valid PNG MIME to pass, got error: %s", errMsg)
	}
}

func TestValidateImageMIMEType_TextContent(t *testing.T) {
	reader := bytes.NewReader(plainTextContent)
	_, valid := ValidateImageMIMEType(reader)
	if valid {
		t.Error("Expected text content to fail MIME validation")
	}
}

func TestValidateImageMIMEType_HTMLContent(t *testing.T) {
	reader := bytes.NewReader(htmlContent)
	_, valid := ValidateImageMIMEType(reader)
	if valid {
		t.Error("Expected HTML content to fail MIME validation")
	}
}
