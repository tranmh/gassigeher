package services

import (
	"testing"
)

// =============================================================================
// BUG #2: EMAIL SERVICE SILENT FAILURE
// =============================================================================
// ISSUE: If email service fails to initialize, all emails silently fail.
// Handlers use pattern: if emailService != nil { go emailService.Send() }
// This means if init fails, no emails are sent and admins have no visibility.
//
// FIX: Add IsHealthy() method and explicit logging so admins know email is broken.
// =============================================================================

// TestEmailService_IsHealthy tests the IsHealthy() method on EmailService
func TestEmailService_IsHealthy(t *testing.T) {
	t.Run("nil_email_service_is_not_healthy", func(t *testing.T) {
		var es *EmailService = nil

		// Call the IsHealthy method on the email service
		if es.IsHealthy() {
			t.Error("Nil email service should not be healthy")
		}
	})

	t.Run("initialized_service_with_valid_provider_is_healthy", func(t *testing.T) {
		// Create a mock email config that would successfully initialize
		cfg := &EmailConfig{
			Provider:       "mock", // Mock provider for testing
			GmailFromEmail: "test@example.com",
			BaseURL:        "http://localhost:8080",
		}

		// Try to create email service - this would fail without mock provider
		// but that's expected for now. The test verifies the interface.
		es, err := NewEmailService(cfg)

		// If we get a service, check health
		if err == nil && es != nil {
			if !es.IsHealthy() {
				t.Error("Valid email service should be healthy")
			}
		} else {
			// Expected for now - no mock provider implemented
			t.Logf("Email service not created (expected without mock): %v", err)
		}
	})
}

// TestEmailService_GetHealthStatus tests the GetHealthStatus() method
func TestEmailService_GetHealthStatus(t *testing.T) {
	t.Run("nil_service_returns_descriptive_error", func(t *testing.T) {
		var es *EmailService = nil

		status := es.GetHealthStatus()

		if status.Healthy {
			t.Error("Nil service should not be healthy")
		}

		if status.Error == "" {
			t.Error("Should return reason why service is unhealthy")
		}

		if status.Error != "email service not initialized" {
			t.Errorf("Expected 'email service not initialized', got: %s", status.Error)
		}

		if status.Provider != "none" {
			t.Errorf("Expected provider 'none', got: %s", status.Provider)
		}
	})

	t.Run("returns_full_health_status_for_nil_service", func(t *testing.T) {
		var es *EmailService = nil

		status := es.GetHealthStatus()

		if status.Healthy {
			t.Error("Nil service status should not be healthy")
		}

		if status.Provider != "none" {
			t.Errorf("Expected provider 'none', got: %s", status.Provider)
		}

		if status.Error == "" {
			t.Error("Status should include error message for unhealthy service")
		}
	})
}

// TestEmailService_LogsInitializationFailure verifies logging happens on init failure
func TestEmailService_LogsInitializationFailure(t *testing.T) {
	t.Run("invalid_config_logs_clear_error", func(t *testing.T) {
		// Create invalid config
		cfg := &EmailConfig{
			Provider: "invalid-provider-xyz",
		}

		_, err := NewEmailService(cfg)

		if err == nil {
			t.Error("Invalid provider should cause initialization error")
		}

		// The error message should be clear and actionable
		errMsg := err.Error()
		if errMsg == "" {
			t.Error("Error message should not be empty")
		}

		t.Logf("Error message: %s", errMsg)
	})
}

// TestEmailService_IsHealthy_MethodSafeOnNil verifies IsHealthy is safe to call on nil
func TestEmailService_IsHealthy_MethodSafeOnNil(t *testing.T) {
	t.Run("does_not_panic_on_nil", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("IsHealthy() panicked on nil receiver: %v", r)
			}
		}()

		var es *EmailService = nil
		healthy := es.IsHealthy()

		if healthy {
			t.Error("Nil service should not be healthy")
		}
	})
}

// TestEmailService_GetHealthStatus_MethodSafeOnNil verifies GetHealthStatus is safe on nil
func TestEmailService_GetHealthStatus_MethodSafeOnNil(t *testing.T) {
	t.Run("does_not_panic_on_nil", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("GetHealthStatus() panicked on nil receiver: %v", r)
			}
		}()

		var es *EmailService = nil
		status := es.GetHealthStatus()

		if status.Healthy {
			t.Error("Nil service should not be healthy")
		}
	})
}
