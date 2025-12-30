package services

import (
	"encoding/json"
	"testing"

	"github.com/stripe/stripe-go/v76"
)

// ============================================================================
// BUG #2: CRITICAL - Silent error in tenant ID parsing
// Original bug (line 252-261): fmt.Sscanf error was ignored, tenantID defaulted to 0
// Fix: Add proper error handling and validate tenantID > 0
// ============================================================================

func TestParseCheckoutSessionEvent_ValidTenantID(t *testing.T) {
	// Test that valid tenant IDs are parsed correctly
	service := NewStripeService("sk_test", "pk_test", "price_monthly", "price_yearly", "https://example.com")

	testCases := []struct {
		name            string
		tenantIDValue   string
		expectedTenantID int
		shouldError     bool
	}{
		{
			name:            "Valid tenant ID",
			tenantIDValue:   "123",
			expectedTenantID: 123,
			shouldError:     false,
		},
		{
			name:            "Large tenant ID",
			tenantIDValue:   "999999",
			expectedTenantID: 999999,
			shouldError:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			event := createTestCheckoutEvent(tc.tenantIDValue)
			data, err := service.ParseCheckoutSessionEvent(event)

			if tc.shouldError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if data.TenantID != tc.expectedTenantID {
				t.Errorf("TenantID = %d, want %d", data.TenantID, tc.expectedTenantID)
			}
		})
	}
}

func TestParseCheckoutSessionEvent_InvalidTenantID_ReturnsError(t *testing.T) {
	// This is the critical test - invalid tenant IDs must return an error
	// The original bug silently defaulted to tenantID=0
	service := NewStripeService("sk_test", "pk_test", "price_monthly", "price_yearly", "https://example.com")

	testCases := []struct {
		name          string
		tenantIDValue string
		errorMessage  string
	}{
		{
			name:          "Non-numeric tenant ID",
			tenantIDValue: "not-a-number",
			errorMessage:  "invalid tenant_id",
		},
		{
			name:          "Empty tenant ID",
			tenantIDValue: "",
			errorMessage:  "invalid tenant_id",
		},
		{
			name:          "Negative tenant ID",
			tenantIDValue: "-1",
			errorMessage:  "tenant_id must be positive",
		},
		{
			name:          "Zero tenant ID",
			tenantIDValue: "0",
			errorMessage:  "tenant_id must be positive",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			event := createTestCheckoutEvent(tc.tenantIDValue)
			data, err := service.ParseCheckoutSessionEvent(event)

			if err == nil {
				t.Errorf("Expected error for tenant_id=%q but got none (TenantID=%d)",
					tc.tenantIDValue, data.TenantID)
				return
			}

			// Verify error message contains expected text
			if tc.errorMessage != "" {
				if !stripeContainsSubstring(err.Error(), tc.errorMessage) {
					t.Errorf("Error message %q should contain %q", err.Error(), tc.errorMessage)
				}
			}
		})
	}
}

func TestParseCheckoutSessionEvent_MissingTenantID_AllowedButZero(t *testing.T) {
	// When tenant_id is completely missing from metadata, it's allowed
	// but TenantID will be 0 (the zero value)
	// The caller should handle this case
	service := NewStripeService("sk_test", "pk_test", "price_monthly", "price_yearly", "https://example.com")

	// Create event without tenant_id in metadata
	sessionData := map[string]interface{}{
		"customer_email": "test@example.com",
		"metadata":       map[string]string{}, // No tenant_id
	}

	raw, _ := json.Marshal(sessionData)
	event := &stripe.Event{
		Type: WebhookEventCheckoutCompleted,
		Data: &stripe.EventData{
			Raw: raw,
		},
	}

	data, err := service.ParseCheckoutSessionEvent(event)

	// No error expected when metadata is missing (different from invalid value)
	if err != nil {
		t.Errorf("Unexpected error for missing tenant_id: %v", err)
		return
	}

	if data.TenantID != 0 {
		t.Errorf("Expected TenantID=0 when metadata is missing, got %d", data.TenantID)
	}
}

func TestParseCheckoutSessionEvent_NilCustomerSafe(t *testing.T) {
	// Test that nil Customer doesn't cause a panic
	service := NewStripeService("sk_test", "pk_test", "price_monthly", "price_yearly", "https://example.com")

	sessionData := map[string]interface{}{
		"customer_email": "test@example.com",
		"customer":       nil, // Explicitly nil
		"metadata": map[string]string{
			"tenant_id": "123",
		},
	}

	raw, _ := json.Marshal(sessionData)
	event := &stripe.Event{
		Type: WebhookEventCheckoutCompleted,
		Data: &stripe.EventData{
			Raw: raw,
		},
	}

	// Should not panic
	data, err := service.ParseCheckoutSessionEvent(event)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	// CustomerID should be empty string, not crash
	if data.CustomerID != "" {
		t.Errorf("Expected empty CustomerID for nil customer, got %q", data.CustomerID)
	}
}

func TestParseCheckoutSessionEvent_NilSubscriptionSafe(t *testing.T) {
	// Test that nil Subscription doesn't cause a panic
	service := NewStripeService("sk_test", "pk_test", "price_monthly", "price_yearly", "https://example.com")

	sessionData := map[string]interface{}{
		"customer_email": "test@example.com",
		"subscription":   nil, // Explicitly nil
		"metadata": map[string]string{
			"tenant_id": "123",
		},
	}

	raw, _ := json.Marshal(sessionData)
	event := &stripe.Event{
		Type: WebhookEventCheckoutCompleted,
		Data: &stripe.EventData{
			Raw: raw,
		},
	}

	// Should not panic
	data, err := service.ParseCheckoutSessionEvent(event)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	// SubscriptionID should be empty string, not crash
	if data.SubscriptionID != "" {
		t.Errorf("Expected empty SubscriptionID for nil subscription, got %q", data.SubscriptionID)
	}
}

func TestParseCheckoutSessionEvent_PromoAndReferralCodes(t *testing.T) {
	// Test that promo and referral codes are extracted correctly
	service := NewStripeService("sk_test", "pk_test", "price_monthly", "price_yearly", "https://example.com")

	sessionData := map[string]interface{}{
		"customer_email": "test@example.com",
		"metadata": map[string]string{
			"tenant_id":        "123",
			"promo_code":       "SUMMER2025",
			"referral_code":    "REF123",
			"promo_code_id":    "456",
			"referral_code_id": "789",
		},
	}

	raw, _ := json.Marshal(sessionData)
	event := &stripe.Event{
		Type: WebhookEventCheckoutCompleted,
		Data: &stripe.EventData{
			Raw: raw,
		},
	}

	data, err := service.ParseCheckoutSessionEvent(event)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if data.PromoCode != "SUMMER2025" {
		t.Errorf("PromoCode = %q, want %q", data.PromoCode, "SUMMER2025")
	}
	if data.ReferralCode != "REF123" {
		t.Errorf("ReferralCode = %q, want %q", data.ReferralCode, "REF123")
	}
	if data.PromoCodeID != 456 {
		t.Errorf("PromoCodeID = %d, want %d", data.PromoCodeID, 456)
	}
	if data.ReferralCodeID != 789 {
		t.Errorf("ReferralCodeID = %d, want %d", data.ReferralCodeID, 789)
	}
}

// ============================================================================
// Helper functions
// ============================================================================

func createTestCheckoutEvent(tenantIDValue string) *stripe.Event {
	sessionData := map[string]interface{}{
		"customer_email": "test@example.com",
		"metadata": map[string]string{
			"tenant_id": tenantIDValue,
		},
	}

	raw, _ := json.Marshal(sessionData)
	return &stripe.Event{
		Type: WebhookEventCheckoutCompleted,
		Data: &stripe.EventData{
			Raw: raw,
		},
	}
}

func stripeContainsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
