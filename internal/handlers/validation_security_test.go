package handlers

import (
	"testing"
)

// GREEN PHASE: Tests for Bug #3 - javascript: protocol XSS vulnerability
// These tests verify the security fix is working

// TestSanitizeString_JavascriptProtocol tests that javascript: protocol is blocked
func TestSanitizeString_JavascriptProtocol(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "javascript: protocol should be removed",
			input:    "javascript:alert(1)",
			expected: "",
		},
		{
			name:     "javascript: with mixed case should be removed",
			input:    "JavaScript:alert(1)",
			expected: "",
		},
		{
			name:     "javascript: with spaces at start should be removed",
			input:    "  javascript:alert(1)",
			expected: "",
		},
		{
			name:     "javascript: with encoded colon should be removed",
			input:    "javascript&#58;alert(1)",
			expected: "",
		},
		{
			name:     "data: URI should be removed",
			input:    "data:text/html,alert(1)", // script tags already stripped by StripHTMLTags
			expected: "",
		},
		{
			name:     "vbscript: should be removed",
			input:    "vbscript:msgbox(1)",
			expected: "",
		},
		{
			name:     "valid URL should be preserved",
			input:    "https://example.com",
			expected: "https://example.com",
		},
		{
			name:     "normal text should be preserved",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "text with javascript word should be preserved",
			input:    "I love javascript programming",
			expected: "I love javascript programming",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeString(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestValidateDogName_JavascriptProtocol tests dog name validation blocks javascript:
func TestValidateDogName_JavascriptProtocol(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantError  bool
		errorField string
	}{
		{
			name:      "javascript: protocol should be rejected or sanitized to empty",
			input:     "javascript:alert(1)",
			wantError: true, // Should either error or sanitize to empty (which triggers "Name is required")
		},
		{
			name:      "mixed case javascript: should be rejected",
			input:     "JAVASCRIPT:alert('xss')",
			wantError: true,
		},
		{
			name:      "normal name should be accepted",
			input:     "Bella",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateDogName(tt.input)
			if tt.wantError && err == nil {
				t.Errorf("ValidateDogName(%q) should return error for XSS attempt", tt.input)
			}
			if !tt.wantError && err != nil {
				t.Errorf("ValidateDogName(%q) unexpected error: %v", tt.input, err)
			}
		})
	}
}

// TestValidateDogExternalLink_JavascriptProtocol tests external link blocks dangerous protocols
func TestValidateDogExternalLink_JavascriptProtocol(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:      "javascript: protocol should be rejected",
			input:     "javascript:alert(1)",
			wantError: true,
		},
		{
			name:      "data: URI should be rejected",
			input:     "data:text/html,<script>alert(1)</script>",
			wantError: true,
		},
		{
			name:      "https URL should be accepted",
			input:     "https://example.com/dog/123",
			wantError: false,
		},
		{
			name:      "http URL should be accepted",
			input:     "http://shelter.org/dogs",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.input
			_, err := ValidateDogExternalLink(&input)
			if tt.wantError && err == nil {
				t.Errorf("ValidateDogExternalLink(%q) should return error for dangerous protocol", tt.input)
			}
			if !tt.wantError && err != nil {
				t.Errorf("ValidateDogExternalLink(%q) unexpected error: %v", tt.input, err)
			}
		})
	}
}

// TestStripHTMLTags_XSSVectors tests that all XSS vectors are properly stripped
func TestStripHTMLTags_XSSVectors(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "script tag",
			input:    "<script>alert(1)</script>",
			expected: "alert(1)",
		},
		{
			name:     "img onerror",
			input:    "<img src=x onerror=alert(1)>",
			expected: "",
		},
		{
			name:     "svg onload",
			input:    "<svg onload=alert(1)>",
			expected: "",
		},
		{
			name:     "iframe",
			input:    "<iframe src='javascript:alert(1)'></iframe>",
			expected: "",
		},
		{
			name:     "event handler in div",
			input:    "<div onclick=alert(1)>Click me</div>",
			expected: "Click me",
		},
		{
			name:     "nested tags",
			input:    "<div><script>alert(1)</script></div>",
			expected: "alert(1)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripHTMLTags(tt.input)
			if result != tt.expected {
				t.Errorf("StripHTMLTags(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
