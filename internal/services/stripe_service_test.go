package services

import (
	"testing"
)

// TestStripeService_CreateCheckoutSession tests creating a Stripe checkout session (TDD RED Phase)
func TestStripeService_CreateCheckoutSession(t *testing.T) {
	// Test without real Stripe API - use mock or skip if no API key
	t.Run("returns error when API key not configured", func(t *testing.T) {
		service := NewStripeService("", "", "", "", "")
		_, err := service.CreateCheckoutSession(1, "pro", "monthly", "test@example.com")
		if err == nil {
			t.Error("Expected error for missing API key, got nil")
		}
	})

	t.Run("validates plan slug", func(t *testing.T) {
		service := NewStripeService("sk_test_xxx", "pk_test_xxx", "price_monthly", "price_yearly", "http://localhost:8080")
		_, err := service.CreateCheckoutSession(1, "invalid_plan", "monthly", "test@example.com")
		if err == nil {
			t.Error("Expected error for invalid plan slug, got nil")
		}
	})

	t.Run("validates billing cycle", func(t *testing.T) {
		service := NewStripeService("sk_test_xxx", "pk_test_xxx", "price_monthly", "price_yearly", "http://localhost:8080")
		_, err := service.CreateCheckoutSession(1, "pro", "weekly", "test@example.com")
		if err == nil {
			t.Error("Expected error for invalid billing cycle, got nil")
		}
	})
}

// TestStripeService_CreateBillingPortalSession tests creating a billing portal session
func TestStripeService_CreateBillingPortalSession(t *testing.T) {
	t.Run("returns error when API key not configured", func(t *testing.T) {
		service := NewStripeService("", "", "", "", "")
		_, err := service.CreateBillingPortalSession("cus_xxx")
		if err == nil {
			t.Error("Expected error for missing API key, got nil")
		}
	})

	t.Run("returns error when customer ID empty", func(t *testing.T) {
		service := NewStripeService("sk_test_xxx", "pk_test_xxx", "price_monthly", "price_yearly", "http://localhost:8080")
		_, err := service.CreateBillingPortalSession("")
		if err == nil {
			t.Error("Expected error for empty customer ID, got nil")
		}
	})
}

// TestStripeService_CancelSubscription tests canceling a subscription
func TestStripeService_CancelSubscription(t *testing.T) {
	t.Run("returns error when API key not configured", func(t *testing.T) {
		service := NewStripeService("", "", "", "", "")
		err := service.CancelSubscription("sub_xxx")
		if err == nil {
			t.Error("Expected error for missing API key, got nil")
		}
	})

	t.Run("returns error when subscription ID empty", func(t *testing.T) {
		service := NewStripeService("sk_test_xxx", "pk_test_xxx", "price_monthly", "price_yearly", "http://localhost:8080")
		err := service.CancelSubscription("")
		if err == nil {
			t.Error("Expected error for empty subscription ID, got nil")
		}
	})
}

// TestStripeService_VerifyWebhookSignature tests webhook signature verification
func TestStripeService_VerifyWebhookSignature(t *testing.T) {
	t.Run("returns error when webhook secret not configured", func(t *testing.T) {
		service := NewStripeService("sk_test_xxx", "pk_test_xxx", "price_monthly", "price_yearly", "http://localhost:8080")
		_, err := service.VerifyWebhookSignature([]byte("{}"), "sig_xxx")
		if err == nil {
			t.Error("Expected error for missing webhook secret, got nil")
		}
	})
}

// TestStripeService_IsConfigured tests checking if Stripe is properly configured
func TestStripeService_IsConfigured(t *testing.T) {
	tests := []struct {
		name       string
		secretKey  string
		priceMonth string
		priceYear  string
		expected   bool
	}{
		{
			name:       "Not configured - all empty",
			secretKey:  "",
			priceMonth: "",
			priceYear:  "",
			expected:   false,
		},
		{
			name:       "Not configured - missing secret key",
			secretKey:  "",
			priceMonth: "price_xxx",
			priceYear:  "price_yyy",
			expected:   false,
		},
		{
			name:       "Not configured - missing monthly price",
			secretKey:  "sk_test_xxx",
			priceMonth: "",
			priceYear:  "price_yyy",
			expected:   false,
		},
		{
			name:       "Not configured - missing yearly price",
			secretKey:  "sk_test_xxx",
			priceMonth: "price_xxx",
			priceYear:  "",
			expected:   false,
		},
		{
			name:       "Fully configured",
			secretKey:  "sk_test_xxx",
			priceMonth: "price_xxx",
			priceYear:  "price_yyy",
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewStripeService(tt.secretKey, "pk_test_xxx", tt.priceMonth, tt.priceYear, "http://localhost:8080")
			if service.IsConfigured() != tt.expected {
				t.Errorf("IsConfigured() = %v, want %v", service.IsConfigured(), tt.expected)
			}
		})
	}
}

// TestStripeService_GetPriceID tests getting the correct price ID
func TestStripeService_GetPriceID(t *testing.T) {
	service := NewStripeService("sk_test_xxx", "pk_test_xxx", "price_monthly_123", "price_yearly_456", "http://localhost:8080")

	tests := []struct {
		name         string
		billingCycle string
		expected     string
		expectError  bool
	}{
		{
			name:         "Monthly billing",
			billingCycle: "monthly",
			expected:     "price_monthly_123",
			expectError:  false,
		},
		{
			name:         "Yearly billing",
			billingCycle: "yearly",
			expected:     "price_yearly_456",
			expectError:  false,
		},
		{
			name:         "Invalid billing cycle",
			billingCycle: "weekly",
			expected:     "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			priceID, err := service.GetPriceID(tt.billingCycle)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if priceID != tt.expected {
					t.Errorf("GetPriceID() = %s, want %s", priceID, tt.expected)
				}
			}
		})
	}
}
