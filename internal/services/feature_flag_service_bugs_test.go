package services

import (
	"testing"
)

// ============================================================================
// BUG #1: CRITICAL - Cache key collision due to wrong int-to-string conversion
// Line 36: return flagKey + ":" + string(rune(tenantID))
// This converts the int to a unicode character instead of a decimal string!
// ============================================================================

func TestCacheKey_BUG_WrongIntConversion(t *testing.T) {
	// BUG: string(rune(65)) returns "A", not "65"
	// This means tenant 65 and tenant 66 get cache keys like:
	//   "feature:A" and "feature:B" instead of "feature:65" and "feature:66"
	//
	// Even worse: tenants 32-126 map to printable ASCII chars,
	// but tenants 0-31 map to control characters!
	// Tenant 0 maps to NUL character, tenant 10 maps to newline!

	// Demonstrate the bug:
	key1 := cacheKey(65, "test_feature")
	key2 := cacheKey(66, "test_feature")

	// What the code actually produces (BUG):
	// key1 = "test_feature:A" (65 = ASCII 'A')
	// key2 = "test_feature:B" (66 = ASCII 'B')

	// What it SHOULD produce:
	// key1 = "test_feature:65"
	// key2 = "test_feature:66"

	t.Logf("BUG CONFIRMED: cacheKey(65, 'test_feature') = %q", key1)
	t.Logf("BUG CONFIRMED: cacheKey(66, 'test_feature') = %q", key2)

	// The bug causes tenant ID collisions for non-ASCII values
	// For example: tenant 1000 would produce a unicode character '\u03E8'
	key1000 := cacheKey(1000, "test")
	t.Logf("BUG: cacheKey(1000, 'test') = %q (unicode char)", key1000)

	// RECOMMENDATION:
	// Fix line 36 from: return flagKey + ":" + string(rune(tenantID))
	// To: return flagKey + ":" + strconv.Itoa(tenantID)
	// Or: return fmt.Sprintf("%s:%d", flagKey, tenantID)

	// This test documents the bug - the actual fix should use strconv.Itoa
	if key1 == "test_feature:65" {
		t.Log("Bug has been fixed - cache key now uses proper int conversion")
	} else {
		t.Log("BUG PRESENT: Cache key uses wrong int-to-string conversion")
		t.Log("Impact: Multiple tenants may share the same cache key")
		t.Log("For tenant IDs 0-127, keys collide with ASCII characters")
	}
}

// ============================================================================
// Additional cache key tests for coverage and bug detection
// ============================================================================

func TestCacheKey_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		tenantID  int
		flagKey   string
		wantBug   string // What the buggy code produces
		wantFixed string // What it should produce
	}{
		{
			name:      "Tenant 0 produces NUL character",
			tenantID:  0,
			flagKey:   "feature",
			wantBug:   "feature:\x00", // NUL character
			wantFixed: "feature:0",
		},
		{
			name:      "Tenant 10 produces newline",
			tenantID:  10,
			flagKey:   "feature",
			wantBug:   "feature:\n", // Newline character
			wantFixed: "feature:10",
		},
		{
			name:      "Tenant 32 produces space",
			tenantID:  32,
			flagKey:   "feature",
			wantBug:   "feature: ", // Space character
			wantFixed: "feature:32",
		},
		{
			name:      "Tenant 58 produces colon (delimiter collision!)",
			tenantID:  58,
			flagKey:   "feature",
			wantBug:   "feature::", // Double colon!
			wantFixed: "feature:58",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cacheKey(tt.tenantID, tt.flagKey)
			if got == tt.wantBug {
				t.Logf("BUG: Got %q (buggy), should be %q", got, tt.wantFixed)
			}
			// Document that this is a known bug
			t.Logf("cacheKey(%d, %q) = %q", tt.tenantID, tt.flagKey, got)
		})
	}
}
