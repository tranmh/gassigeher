package services

import (
	"strconv"
	"testing"
)

// ============================================================================
// BUG #1: CRITICAL - Cache key collision due to wrong int-to-string conversion
// Original bug (line 36): return flagKey + ":" + string(rune(tenantID))
// This converted the int to a unicode character instead of a decimal string!
// Fix: Use strconv.Itoa(tenantID) instead
// ============================================================================

func TestCacheKey_CorrectIntConversion(t *testing.T) {
	// Test that tenant IDs are properly converted to string numbers
	// not ASCII/Unicode characters

	testCases := []struct {
		name     string
		tenantID int
		flagKey  string
		expected string
	}{
		{
			name:     "Tenant 65 should be '65' not 'A'",
			tenantID: 65,
			flagKey:  "test_feature",
			expected: "test_feature:65",
		},
		{
			name:     "Tenant 66 should be '66' not 'B'",
			tenantID: 66,
			flagKey:  "test_feature",
			expected: "test_feature:66",
		},
		{
			name:     "Tenant 0 should be '0' not NUL",
			tenantID: 0,
			flagKey:  "feature",
			expected: "feature:0",
		},
		{
			name:     "Tenant 10 should be '10' not newline",
			tenantID: 10,
			flagKey:  "feature",
			expected: "feature:10",
		},
		{
			name:     "Tenant 32 should be '32' not space",
			tenantID: 32,
			flagKey:  "feature",
			expected: "feature:32",
		},
		{
			name:     "Tenant 58 should be '58' not colon (delimiter collision)",
			tenantID: 58,
			flagKey:  "feature",
			expected: "feature:58",
		},
		{
			name:     "Tenant 1000 should be numeric '1000'",
			tenantID: 1000,
			flagKey:  "test",
			expected: "test:1000",
		},
		{
			name:     "Large tenant ID should work correctly",
			tenantID: 999999,
			flagKey:  "premium_feature",
			expected: "premium_feature:999999",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := cacheKey(tc.tenantID, tc.flagKey)
			if got != tc.expected {
				t.Errorf("cacheKey(%d, %q) = %q, want %q",
					tc.tenantID, tc.flagKey, got, tc.expected)
			}
		})
	}
}

func TestCacheKey_NoCollisionBetweenTenants(t *testing.T) {
	// Ensure different tenants get different cache keys
	// This would fail with the bug: string(rune(65)) == "A"

	tenantIDs := []int{1, 10, 65, 66, 100, 1000}
	flagKey := "same_feature"

	keys := make(map[string]int)
	for _, tid := range tenantIDs {
		key := cacheKey(tid, flagKey)
		if existingTid, exists := keys[key]; exists {
			t.Errorf("Cache key collision! Tenant %d and %d both produce key %q",
				existingTid, tid, key)
		}
		keys[key] = tid
	}
}

func TestCacheKey_UsesStrcovItoa(t *testing.T) {
	// Verify the implementation uses proper int-to-string conversion
	// by checking that cacheKey matches the expected format

	tenantID := 65
	flagKey := "feature"

	expected := flagKey + ":" + strconv.Itoa(tenantID)
	got := cacheKey(tenantID, flagKey)

	if got != expected {
		t.Errorf("cacheKey does not use strconv.Itoa format")
		t.Errorf("Expected: %q", expected)
		t.Errorf("Got: %q", got)

		// Check if it's using the buggy string(rune()) conversion
		buggyResult := flagKey + ":" + string(rune(tenantID))
		if got == buggyResult {
			t.Error("BUG DETECTED: cacheKey is using string(rune(tenantID)) instead of strconv.Itoa(tenantID)")
		}
	}
}

func TestCacheKey_ASCIICollisionRegression(t *testing.T) {
	// Regression test: Tenant IDs 65-90 should NOT produce A-Z
	// This catches the string(rune(tenantID)) bug

	asciiLetters := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	for i, char := range asciiLetters {
		tenantID := 65 + i // ASCII 'A' = 65
		key := cacheKey(tenantID, "flag")

		buggyKey := "flag:" + string(char)
		if key == buggyKey {
			t.Errorf("REGRESSION: Tenant %d produces buggy cache key %q (ASCII char)",
				tenantID, key)
		}

		expectedKey := "flag:" + strconv.Itoa(tenantID)
		if key != expectedKey {
			t.Errorf("Tenant %d: got %q, want %q", tenantID, key, expectedKey)
		}
	}
}
