package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ============================================================================
// extractSubdomain Tests - Find edge cases and bugs
// ============================================================================

func TestExtractSubdomain_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		host       string
		baseDomain string
		want       string
	}{
		// Normal cases
		{"normal subdomain", "tierheim.gassigeher.org", "gassigeher.org", "tierheim"},
		{"subdomain with hyphen", "tierheim-goeppingen.gassigeher.org", "gassigeher.org", "tierheim-goeppingen"},

		// Base domain cases
		{"base domain only", "gassigeher.org", "gassigeher.org", ""},
		{"www subdomain", "www.gassigeher.org", "gassigeher.org", "www"},

		// Port handling
		{"host with port", "tierheim.gassigeher.org:8080", "gassigeher.org", "tierheim"},
		{"baseDomain with port", "tierheim.gassigeher.org", "gassigeher.org:8080", "tierheim"},
		{"both with port", "tierheim.gassigeher.org:8080", "gassigeher.org:8080", "tierheim"},

		// Localhost handling
		{"localhost", "localhost", "gassigeher.org", ""},
		{"localhost with port", "localhost:8080", "gassigeher.org", ""},
		{"127.0.0.1", "127.0.0.1", "gassigeher.org", ""},
		{"127.0.0.1 with port", "127.0.0.1:3000", "gassigeher.org", ""},

		// Edge cases
		{"empty baseDomain", "tierheim.gassigeher.org", "", ""},
		{"different domain", "tierheim.example.com", "gassigeher.org", ""},

		// Multi-level subdomain (should be rejected)
		{"multi-level subdomain", "sub.tierheim.gassigeher.org", "gassigeher.org", ""},

		// Security test: Multi-level subdomains correctly rejected
		{"host injection attempt correctly blocked", "evil.com.gassigeher.org", "gassigeher.org", ""},
		// Unicode subdomains are rejected to prevent homograph attacks (e.g., Cyrillic "а" vs Latin "a")
		{"unicode in subdomain rejected", "tïerheim.gassigeher.org", "gassigeher.org", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSubdomain(tt.host, tt.baseDomain)
			if got != tt.want {
				t.Errorf("extractSubdomain(%q, %q) = %q, want %q", tt.host, tt.baseDomain, got, tt.want)
			}
		})
	}
}

// ============================================================================
// BUG #1: "admin" subdomain bypasses tenant check
// This could allow unauthenticated access to admin.gassigeher.org
// ============================================================================

func TestTenantMiddleware_BUG_AdminSubdomainBypass(t *testing.T) {
	// The TenantMiddleware at line 23 checks: slug == "admin"
	// If true, it skips tenant resolution entirely.
	// This means requests to admin.gassigeher.org have NO tenant context.
	//
	// Potential security issue: If any handler assumes tenant=0 means
	// "super admin" or "all tenants", this could be exploited.

	t.Log("BUG: Line 23 allows 'admin' subdomain to bypass tenant resolution")
	t.Log("Requests to admin.gassigeher.org will have tenantID=0")
	t.Log("This is intentional for central admin, but handlers must validate this")
}

// ============================================================================
// BUG #2: GetTenantID returns 0 for missing context (silent failure)
// ============================================================================

func TestGetTenantID_NoTenantContext(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/dogs", nil)

	// No tenant context set
	tenantID := GetTenantID(req)

	if tenantID != 0 {
		t.Errorf("GetTenantID without context = %d, want 0", tenantID)
	}

	// BUG: This returns 0 silently instead of indicating an error.
	// Callers must explicitly check for 0, or they'll perform operations
	// with tenantID=0 which could affect all tenants or fail silently.
	t.Log("Note: GetTenantID returns 0 for missing tenant - callers must validate this")
}

// ============================================================================
// BUG #3: GetTenantSlug returns empty string for missing context
// ============================================================================

func TestGetTenantSlug_NoTenantContext(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/dogs", nil)

	slug := GetTenantSlug(req)

	if slug != "" {
		t.Errorf("GetTenantSlug without context = %q, want empty", slug)
	}
}

// ============================================================================
// RequireTenant Tests
// ============================================================================

func TestRequireTenant_BlocksZeroTenant(t *testing.T) {
	handler := RequireTenant(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/dogs", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("RequireTenant without tenant = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestRequireTenant_AllowsValidTenant(t *testing.T) {
	handler := RequireTenant(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/dogs", nil)
	ctx := context.WithValue(req.Context(), TenantIDKey, 1)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("RequireTenant with valid tenant = %d, want %d", rec.Code, http.StatusOK)
	}
}

// ============================================================================
// BUG #4: Subdomain validation allows potentially dangerous characters
// ============================================================================

func TestExtractSubdomain_DangerousCharactersRejected(t *testing.T) {
	// These subdomains contain dangerous characters that should be rejected
	// by the security validation in extractSubdomain
	dangerousInputs := []struct {
		name       string
		host       string
		baseDomain string
		concern    string
	}{
		{
			name:       "single quote in subdomain",
			host:       "tier'heim.gassigeher.org",
			baseDomain: "gassigeher.org",
			concern:    "SQL injection if used in raw query",
		},
		{
			name:       "semicolon in subdomain",
			host:       "tier;heim.gassigeher.org",
			baseDomain: "gassigeher.org",
			concern:    "Command injection if used in shell",
		},
		{
			name:       "null byte in subdomain",
			host:       "tier\x00heim.gassigeher.org",
			baseDomain: "gassigeher.org",
			concern:    "Null byte injection",
		},
		{
			name:       "CRLF in subdomain",
			host:       "tier\r\nheim.gassigeher.org",
			baseDomain: "gassigeher.org",
			concern:    "HTTP header injection",
		},
	}

	for _, tt := range dangerousInputs {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSubdomain(tt.host, tt.baseDomain)
			if got != "" {
				t.Errorf("extractSubdomain should reject dangerous characters, got %q (concern: %s)", got, tt.concern)
			}
		})
	}
}

// ============================================================================
// Context Key Type Tests
// ============================================================================

func TestContextKeys_AreDistinct(t *testing.T) {
	// Verify context keys are distinct and don't collide
	if TenantIDKey == TenantSlugKey {
		t.Error("TenantIDKey and TenantSlugKey should be distinct")
	}
	if TenantIDKey == IsDemoKey {
		t.Error("TenantIDKey and IsDemoKey should be distinct")
	}
}

func TestContextKeys_TypeSafety(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)

	// Set tenant ID
	ctx := context.WithValue(req.Context(), TenantIDKey, 42)
	req = req.WithContext(ctx)

	// Correct type assertion
	tenantID, ok := req.Context().Value(TenantIDKey).(int)
	if !ok || tenantID != 42 {
		t.Errorf("TenantIDKey type assertion failed: got %v, %v", tenantID, ok)
	}

	// Wrong type assertion (simulating bug)
	tenantStr, ok := req.Context().Value(TenantIDKey).(string)
	if ok {
		t.Errorf("TenantIDKey should not be extractable as string, got %q", tenantStr)
	}
}
