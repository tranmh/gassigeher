package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tranmh/gassigeher/internal/config"
)

// TestContactHandler_Submit tests contact form submissions
func TestContactHandler_Submit(t *testing.T) {
	cfg := &config.Config{
		ContactEmail: "test@example.com",
		// Email service will be nil (no email sent in tests)
	}
	handler := NewContactHandler(cfg)

	t.Run("successful submission", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":         "Test User",
			"email":        "user@example.com",
			"subject":      "general",
			"organization": "Test Org",
			"message":      "This is a test message",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/contact", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.Submit(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if response["message"] == nil {
			t.Error("Expected message in response")
		}
	})

	t.Run("successful submission without organization", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":    "Test User",
			"email":   "user@example.com",
			"subject": "support",
			"message": "This is a support request",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/contact", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.Submit(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}
	})

	t.Run("missing required fields", func(t *testing.T) {
		tests := []struct {
			name     string
			reqBody  map[string]interface{}
			expected string
		}{
			{
				name: "missing name",
				reqBody: map[string]interface{}{
					"email":   "test@example.com",
					"subject": "general",
					"message": "Test message",
				},
				expected: "Name ist erforderlich",
			},
			{
				name: "missing email",
				reqBody: map[string]interface{}{
					"name":    "Test User",
					"subject": "general",
					"message": "Test message",
				},
				expected: "E-Mail ist erforderlich",
			},
			{
				name: "missing subject",
				reqBody: map[string]interface{}{
					"name":    "Test User",
					"email":   "test@example.com",
					"message": "Test message",
				},
				expected: "Betreff ist erforderlich",
			},
			{
				name: "missing message",
				reqBody: map[string]interface{}{
					"name":    "Test User",
					"email":   "test@example.com",
					"subject": "general",
				},
				expected: "Nachricht ist erforderlich",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				body, _ := json.Marshal(tt.reqBody)
				req := httptest.NewRequest("POST", "/api/contact", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")

				rec := httptest.NewRecorder()
				handler.Submit(rec, req)

				if rec.Code != http.StatusBadRequest {
					t.Errorf("Expected status 400, got %d", rec.Code)
				}

				var response map[string]interface{}
				json.Unmarshal(rec.Body.Bytes(), &response)

				if errMsg, ok := response["error"].(string); !ok || errMsg != tt.expected {
					t.Errorf("Expected error '%s', got '%v'", tt.expected, response["error"])
				}
			})
		}
	})

	t.Run("invalid email format", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":    "Test User",
			"email":   "not-an-email",
			"subject": "general",
			"message": "Test message",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/contact", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.Submit(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)

		if errMsg, ok := response["error"].(string); !ok || errMsg != "Ungültige E-Mail-Adresse" {
			t.Errorf("Expected 'Ungültige E-Mail-Adresse', got '%v'", response["error"])
		}
	})

	t.Run("field length limits", func(t *testing.T) {
		t.Run("name too long", func(t *testing.T) {
			reqBody := map[string]interface{}{
				"name":    strings.Repeat("a", 201),
				"email":   "test@example.com",
				"subject": "general",
				"message": "Test message",
			}

			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest("POST", "/api/contact", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			handler.Submit(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", rec.Code)
			}

			var response map[string]interface{}
			json.Unmarshal(rec.Body.Bytes(), &response)

			if errMsg, ok := response["error"].(string); !ok || errMsg != "Name ist zu lang" {
				t.Errorf("Expected 'Name ist zu lang', got '%v'", response["error"])
			}
		})

		t.Run("email too long", func(t *testing.T) {
			reqBody := map[string]interface{}{
				"name":    "Test User",
				"email":   strings.Repeat("a", 200) + "@example.com",
				"subject": "general",
				"message": "Test message",
			}

			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest("POST", "/api/contact", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			handler.Submit(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", rec.Code)
			}

			var response map[string]interface{}
			json.Unmarshal(rec.Body.Bytes(), &response)

			if errMsg, ok := response["error"].(string); !ok || errMsg != "E-Mail ist zu lang" {
				t.Errorf("Expected 'E-Mail ist zu lang', got '%v'", response["error"])
			}
		})

		t.Run("message too long", func(t *testing.T) {
			reqBody := map[string]interface{}{
				"name":    "Test User",
				"email":   "test@example.com",
				"subject": "general",
				"message": strings.Repeat("a", 10001),
			}

			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest("POST", "/api/contact", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			handler.Submit(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", rec.Code)
			}

			var response map[string]interface{}
			json.Unmarshal(rec.Body.Bytes(), &response)

			if errMsg, ok := response["error"].(string); !ok || errMsg != "Nachricht ist zu lang (max. 10000 Zeichen)" {
				t.Errorf("Expected message too long error, got '%v'", response["error"])
			}
		})
	})

	t.Run("invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/contact", strings.NewReader("not json"))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.Submit(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})

	t.Run("whitespace trimming", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":    "  Test User  ",
			"email":   "  test@example.com  ",
			"subject": "  general  ",
			"message": "  Test message  ",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/contact", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.Submit(rec, req)

		// Should succeed - whitespace should be trimmed
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("all subject types", func(t *testing.T) {
		subjects := []string{"general", "support", "sales", "partnership", "press", "other"}

		for _, subject := range subjects {
			t.Run(subject, func(t *testing.T) {
				reqBody := map[string]interface{}{
					"name":    "Test User",
					"email":   "test@example.com",
					"subject": subject,
					"message": "Test message",
				}

				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest("POST", "/api/contact", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")

				rec := httptest.NewRecorder()
				handler.Submit(rec, req)

				if rec.Code != http.StatusOK {
					t.Errorf("Expected status 200 for subject '%s', got %d", subject, rec.Code)
				}
			})
		}
	})
}

// TestContactRequest_Validate tests the validation logic directly
func TestContactRequest_Validate(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		req := &ContactRequest{
			Name:    "Test User",
			Email:   "test@example.com",
			Subject: "general",
			Message: "Test message",
		}

		err := req.Validate()
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("trims whitespace", func(t *testing.T) {
		req := &ContactRequest{
			Name:         "  Test User  ",
			Email:        "  test@example.com  ",
			Subject:      "  general  ",
			Organization: "  Test Org  ",
			Message:      "  Test message  ",
		}

		err := req.Validate()
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if req.Name != "Test User" {
			t.Errorf("Expected name to be trimmed, got: '%s'", req.Name)
		}
		if req.Email != "test@example.com" {
			t.Errorf("Expected email to be trimmed, got: '%s'", req.Email)
		}
		if req.Subject != "general" {
			t.Errorf("Expected subject to be trimmed, got: '%s'", req.Subject)
		}
		if req.Organization != "Test Org" {
			t.Errorf("Expected organization to be trimmed, got: '%s'", req.Organization)
		}
		if req.Message != "Test message" {
			t.Errorf("Expected message to be trimmed, got: '%s'", req.Message)
		}
	})

	t.Run("empty fields after trimming", func(t *testing.T) {
		req := &ContactRequest{
			Name:    "   ",
			Email:   "test@example.com",
			Subject: "general",
			Message: "Test message",
		}

		err := req.Validate()
		if err == nil {
			t.Error("Expected error for whitespace-only name")
		}
	})
}

// TestEscapeHTML tests the HTML escaping function
func TestEscapeHTML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello", "Hello"},
		{"<script>alert('xss')</script>", "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"},
		{"<img src=x onerror=alert('xss')>", "&lt;img src=x onerror=alert(&#39;xss&#39;)&gt;"},
		{"User & Co.", "User &amp; Co."},
		{"\"quoted\"", "&quot;quoted&quot;"},
		{"It's good", "It&#39;s good"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := escapeHTML(tt.input)
			if result != tt.expected {
				t.Errorf("escapeHTML(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestIsValidEmail tests the email validation function
func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		email    string
		expected bool
	}{
		// Valid emails
		{"test@example.com", true},
		{"user.name@domain.org", true},
		{"user+tag@example.co.uk", true},
		{"test123@test.de", true},

		// Invalid emails
		{"", false},
		{"@", false},
		{"test@", false},
		{"@example.com", false},
		{"test", false},
		{"test@.com", false},
		{"test@example", false},

		// Header injection attempts
		{"test@example.com\nBcc: victim@evil.com", false},
		{"test@example.com\r\nBcc: victim@evil.com", false},

		// Multiple emails
		{"test1@example.com,test2@example.com", false},
		{"test1@example.com;test2@example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			result := isValidEmail(tt.email)
			if result != tt.expected {
				t.Errorf("isValidEmail(%q) = %v, want %v", tt.email, result, tt.expected)
			}
		})
	}
}

// TestFormatMessage tests the message formatting function
func TestFormatMessage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "Hello World"},
		{"Line 1\nLine 2", "Line 1<br>Line 2"},
		{"Line 1\nLine 2\nLine 3", "Line 1<br>Line 2<br>Line 3"},
		{"<script>alert('xss')</script>", "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"},
		{"Hello\n<script>", "Hello<br>&lt;script&gt;"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := formatMessage(tt.input)
			if result != tt.expected {
				t.Errorf("formatMessage(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
