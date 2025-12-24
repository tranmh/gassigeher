package services

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewSentryService_DisabledWhenNoDSN(t *testing.T) {
	config := &SentryConfig{
		DSN:         "",
		Environment: "test",
		Release:     "1.0.0",
	}

	s, err := NewSentryService(config)
	if err != nil {
		t.Fatalf("NewSentryService() error = %v", err)
	}

	if s.IsEnabled() {
		t.Error("IsEnabled() = true, want false when DSN is empty")
	}
}

func TestNewSentryService_EnabledWhenDSNProvided(t *testing.T) {
	config := &SentryConfig{
		DSN:         "https://test@sentry.io/123",
		Environment: "production",
		Release:     "1.0.0",
		ServerName:  "test-server",
	}

	s, err := NewSentryService(config)
	if err != nil {
		t.Fatalf("NewSentryService() error = %v", err)
	}

	if !s.IsEnabled() {
		t.Error("IsEnabled() = false, want true when DSN is provided")
	}
}

func TestSentryService_CaptureException_Disabled(t *testing.T) {
	s := &SentryService{
		config:  &SentryConfig{},
		enabled: false,
	}

	// Should not panic when disabled
	s.CaptureException(errors.New("test error"))
}

func TestSentryService_CaptureException_NilError(t *testing.T) {
	s := &SentryService{
		config:  &SentryConfig{DSN: "test"},
		enabled: true,
	}

	// Should not panic with nil error
	s.CaptureException(nil)
}

func TestSentryService_CaptureException_Enabled(t *testing.T) {
	s := &SentryService{
		config:  &SentryConfig{DSN: "test"},
		enabled: true,
	}

	// Should not panic - just logs locally in stub implementation
	s.CaptureException(errors.New("test error"))
}

func TestSentryService_CaptureMessage_Disabled(t *testing.T) {
	s := &SentryService{
		config:  &SentryConfig{},
		enabled: false,
	}

	// Should not panic when disabled
	s.CaptureMessage("test message")
}

func TestSentryService_CaptureMessage_Enabled(t *testing.T) {
	s := &SentryService{
		config:  &SentryConfig{DSN: "test"},
		enabled: true,
	}

	// Should not panic
	s.CaptureMessage("test message")
}

func TestSentryService_CaptureExceptionWithContext_Disabled(t *testing.T) {
	s := &SentryService{
		config:  &SentryConfig{},
		enabled: false,
	}

	// Should not panic when disabled
	s.CaptureExceptionWithContext(
		errors.New("test"),
		map[string]string{"key": "value"},
		map[string]interface{}{"extra": "data"},
	)
}

func TestSentryService_CaptureExceptionWithContext_NilError(t *testing.T) {
	s := &SentryService{
		config:  &SentryConfig{DSN: "test"},
		enabled: true,
	}

	// Should not panic with nil error
	s.CaptureExceptionWithContext(
		nil,
		map[string]string{"key": "value"},
		map[string]interface{}{"extra": "data"},
	)
}

func TestSentryService_CaptureExceptionWithContext_Enabled(t *testing.T) {
	s := &SentryService{
		config:  &SentryConfig{DSN: "test"},
		enabled: true,
	}

	// Should not panic
	s.CaptureExceptionWithContext(
		errors.New("test error"),
		map[string]string{"tenant_id": "123"},
		map[string]interface{}{"user_id": 456},
	)
}

func TestSentryService_SetUser_Disabled(t *testing.T) {
	s := &SentryService{
		config:  &SentryConfig{},
		enabled: false,
	}

	// Should not panic when disabled
	s.SetUser(1, "test@example.com", 2, "test-tenant")
}

func TestSentryService_SetUser_Enabled(t *testing.T) {
	s := &SentryService{
		config:  &SentryConfig{DSN: "test"},
		enabled: true,
	}

	// Should not panic
	s.SetUser(1, "test@example.com", 2, "test-tenant")
}

func TestSentryService_Flush_Disabled(t *testing.T) {
	s := &SentryService{
		config:  &SentryConfig{},
		enabled: false,
	}

	// Should not block when disabled
	s.Flush(2 * time.Second)
}

func TestSentryService_Flush_Enabled(t *testing.T) {
	s := &SentryService{
		config:  &SentryConfig{DSN: "test"},
		enabled: true,
	}

	// Should not block (stub implementation)
	s.Flush(2 * time.Second)
}

func TestSentryService_RecoveryMiddleware_NoPanic(t *testing.T) {
	s := &SentryService{
		config:  &SentryConfig{DSN: "test"},
		enabled: true,
	}

	// Handler that doesn't panic
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	wrapped := s.RecoveryMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestSentryService_RecoveryMiddleware_WithPanic(t *testing.T) {
	s := &SentryService{
		config:  &SentryConfig{DSN: "test"},
		enabled: true,
	}

	// Handler that panics
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	wrapped := s.RecoveryMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Should not panic - middleware catches it
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Status code = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestSentryService_RecoveryMiddleware_WithErrorPanic(t *testing.T) {
	s := &SentryService{
		config:  &SentryConfig{DSN: "test"},
		enabled: true,
	}

	// Handler that panics with error
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(errors.New("error panic"))
	})

	wrapped := s.RecoveryMiddleware(handler)

	req := httptest.NewRequest("GET", "/test/path", nil)
	rec := httptest.NewRecorder()

	// Should not panic - middleware catches it
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Status code = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestSentryService_RecoveryMiddleware_Disabled(t *testing.T) {
	s := &SentryService{
		config:  &SentryConfig{},
		enabled: false,
	}

	// Handler that panics
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	wrapped := s.RecoveryMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Should still recover even when disabled
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Status code = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestSentryConfig_Fields(t *testing.T) {
	config := SentryConfig{
		DSN:         "https://key@sentry.io/123",
		Environment: "production",
		Release:     "v1.2.3",
		ServerName:  "web-01",
	}

	if config.DSN != "https://key@sentry.io/123" {
		t.Errorf("DSN = %s, want https://key@sentry.io/123", config.DSN)
	}
	if config.Environment != "production" {
		t.Errorf("Environment = %s, want production", config.Environment)
	}
	if config.Release != "v1.2.3" {
		t.Errorf("Release = %s, want v1.2.3", config.Release)
	}
	if config.ServerName != "web-01" {
		t.Errorf("ServerName = %s, want web-01", config.ServerName)
	}
}
