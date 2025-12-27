package services

import (
	"fmt"
	"testing"
)

// ============================================================================
// BUG #1: CRITICAL - Silent error in tenant ID parsing
// Line 206: fmt.Sscanf(tenantIDStr, "%d", &tenantID)
// Error from Sscanf is completely ignored
// ============================================================================

func TestParseCheckoutSession_BUG_SilentSscanfError(t *testing.T) {
	// The code at line 206 does:
	//   fmt.Sscanf(tenantIDStr, "%d", &tenantID)
	//
	// If metadata contains invalid tenant_id like "not-a-number":
	// - Sscanf returns error and scans 0 items
	// - tenantID remains 0 (zero value)
	// - Error is completely ignored
	// - Billing record is created with tenantID=0 (orphaned!)

	// Demonstrate the bug:
	testCases := []struct {
		input    string
		expected int
		hasError bool
	}{
		{"123", 123, false},
		{"not-a-number", 0, true},   // BUG: returns 0, error ignored
		{"", 0, true},               // BUG: returns 0, error ignored
		{"12.34", 12, false},        // Partially parses, stops at '.'
		{"-1", -1, false},           // Negative tenant ID!
		{"99999999999999999999", 0, true}, // Overflow
	}

	for _, tc := range testCases {
		var tenantID int
		n, err := fmt.Sscanf(tc.input, "%d", &tenantID)

		t.Logf("Input: %q -> tenantID=%d, items=%d, err=%v", tc.input, tenantID, n, err)

		if tc.hasError && err == nil && n > 0 {
			t.Logf("Unexpected: %q parsed successfully", tc.input)
		}
	}

	t.Log("")
	t.Log("BUG: Line 206 ignores Sscanf error")
	t.Log("Impact: Invalid metadata causes billing records with tenantID=0")
	t.Log("This orphans the billing record - no tenant can access it!")
	t.Log("")
	t.Log("RECOMMENDATION:")
	t.Log("  n, err := fmt.Sscanf(tenantIDStr, <format>, &tenantID)")
	t.Log("  if err != nil || n != 1 {")
	t.Log("    return nil, fmt.Errorf(<error message>, tenantIDStr)")
	t.Log("  }")
}

// ============================================================================
// BUG #2: MEDIUM - Potential nil pointer dereference
// Line 198: session.Customer.ID
// If Stripe returns session without Customer, this will panic
// ============================================================================

func TestParseCheckoutSession_BUG_NilCustomerPanic(t *testing.T) {
	// The code at lines 198-199 does:
	//   data := &CheckoutSessionData{
	//       CustomerID:     session.Customer.ID,      // <-- BUG: Customer could be nil
	//       SubscriptionID: session.Subscription.ID,  // <-- BUG: Subscription could be nil
	//   }
	//
	// If Stripe returns a session where:
	// - session.Customer is nil (e.g., guest checkout)
	// - session.Subscription is nil (e.g., one-time payment)
	// The code will panic with nil pointer dereference!

	t.Log("BUG: Lines 198-199 access session.Customer.ID and session.Subscription.ID")
	t.Log("Impact: If Stripe returns incomplete data, the application will panic")
	t.Log("")
	t.Log("RECOMMENDATION:")
	t.Log("  var customerID, subscriptionID string")
	t.Log("  if session.Customer != nil {")
	t.Log("    customerID = session.Customer.ID")
	t.Log("  }")
	t.Log("  if session.Subscription != nil {")
	t.Log("    subscriptionID = session.Subscription.ID")
	t.Log("  }")
}

// ============================================================================
// BUG #3: LOW - Negative tenant IDs are accepted
// The Sscanf at line 206 happily parses negative numbers
// ============================================================================

func TestParseCheckoutSession_BUG_NegativeTenantID(t *testing.T) {
	// If metadata contains "tenant_id": "-1", Sscanf will parse it as -1
	// This creates a billing record with tenantID=-1
	// This is likely an injection attempt and should be rejected

	var tenantID int
	fmt.Sscanf("-1", "%d", &tenantID)

	if tenantID < 0 {
		t.Log("BUG: Negative tenant IDs are accepted")
		t.Logf("Sscanf(\"-1\") -> tenantID=%d", tenantID)
		t.Log("Impact: Could be used for injection attacks")
		t.Log("RECOMMENDATION: Validate tenantID > 0 after parsing")
	}
}
