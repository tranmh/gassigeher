package services

import (
	"strings"
	"testing"
)

// TestValidateS3Path tests path validation to prevent path traversal attacks
func TestValidateS3Path(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantValid bool
	}{
		// Valid paths
		{name: "simple filename", path: "photo.jpg", wantValid: true},
		{name: "nested path", path: "dogs/photo.jpg", wantValid: true},
		{name: "deeply nested", path: "uploads/2025/01/photo.jpg", wantValid: true},

		// Invalid paths - path traversal attempts
		{name: "parent directory", path: "../photo.jpg", wantValid: false},
		{name: "double parent", path: "../../photo.jpg", wantValid: false},
		{name: "nested traversal", path: "dogs/../../../photo.jpg", wantValid: false},
		{name: "absolute path", path: "/etc/passwd", wantValid: false},
		{name: "hidden path traversal", path: "dogs/..\\photo.jpg", wantValid: false},
		{name: "null byte", path: "photo.jpg\x00.txt", wantValid: false},
		{name: "empty path", path: "", wantValid: false},
		{name: "only dots", path: "..", wantValid: false},
		{name: "double slash", path: "dogs//photo.jpg", wantValid: true}, // This is OK, just redundant
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateS3Path(tt.path)
			gotValid := err == nil

			if gotValid != tt.wantValid {
				if tt.wantValid {
					t.Errorf("validateS3Path(%q) returned error %v, want valid", tt.path, err)
				} else {
					t.Errorf("validateS3Path(%q) returned nil, want error (path traversal should be rejected)", tt.path)
				}
			}
		})
	}
}

// TestS3ServiceUploadPathValidation tests that Upload rejects path traversal
func TestS3ServiceUploadPathValidation(t *testing.T) {
	// We can't test the actual Upload without mocking S3, but we can test
	// that the validation is applied by testing the validateS3Path function
	// which should be called by Upload

	// This test ensures the validation function exists and works
	err := validateS3Path("../../../malicious.txt")
	if err == nil {
		t.Error("Expected path traversal to be rejected, but it was allowed")
	}
}

// =============================================================================
// TENANT ISOLATION TESTS (SaaS Security Critical)
// =============================================================================

// TestTenantIsolation_ObjectKeyFormat verifies that S3 object keys follow
// the tenant isolation pattern: tenants/{slug}/{path}
func TestTenantIsolation_ObjectKeyFormat(t *testing.T) {
	// Mock service just to test key generation (no actual S3 needed)
	s := &S3Service{
		publicURL:  "https://storage.example.com",
		bucketName: "gassigeher-uploads",
	}

	tests := []struct {
		name           string
		tenantSlug     string
		path           string
		expectedPrefix string
	}{
		{
			name:           "standard tenant path",
			tenantSlug:     "tierheim-goeppingen",
			path:           "dogs/photo.jpg",
			expectedPrefix: "tenants/tierheim-goeppingen/dogs/photo.jpg",
		},
		{
			name:           "user photo path",
			tenantSlug:     "tierheim-munich",
			path:           "users/profile.jpg",
			expectedPrefix: "tenants/tierheim-munich/users/profile.jpg",
		},
		{
			name:           "walk report photo",
			tenantSlug:     "shelter-berlin",
			path:           "walk-reports/2025/01/report1.jpg",
			expectedPrefix: "tenants/shelter-berlin/walk-reports/2025/01/report1.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objectKey, err := s.GetObjectKey(tt.tenantSlug, tt.path)
			if err != nil {
				t.Fatalf("GetObjectKey() unexpected error: %v", err)
			}

			if objectKey != tt.expectedPrefix {
				t.Errorf("GetObjectKey() = %q, want %q", objectKey, tt.expectedPrefix)
			}

			// Verify tenant namespace prefix
			if !strings.HasPrefix(objectKey, "tenants/"+tt.tenantSlug+"/") {
				t.Errorf("Object key %q does not have correct tenant prefix", objectKey)
			}
		})
	}
}

// TestTenantIsolation_CrossTenantAccessPrevention verifies that path traversal
// attacks cannot escape tenant namespace
func TestTenantIsolation_CrossTenantAccessPrevention(t *testing.T) {
	// These paths attempt to escape tenant namespace and access other tenants' files
	attackPaths := []struct {
		name        string
		tenantSlug  string
		malicious   string
		description string
	}{
		{
			name:        "traverse to other tenant via double dot",
			tenantSlug:  "attacker-tenant",
			malicious:   "../victim-tenant/dogs/secret.jpg",
			description: "Attempts to access victim-tenant's files",
		},
		{
			name:        "traverse multiple levels",
			tenantSlug:  "attacker-tenant",
			malicious:   "../../victim-tenant/users/data.json",
			description: "Attempts deeper traversal",
		},
		{
			name:        "escape tenants directory entirely",
			tenantSlug:  "attacker-tenant",
			malicious:   "../../../etc/passwd",
			description: "Attempts to access system files",
		},
		{
			name:        "backslash traversal",
			tenantSlug:  "attacker-tenant",
			malicious:   "..\\victim-tenant\\dogs\\photo.jpg",
			description: "Windows-style path traversal",
		},
		{
			name:        "encoded traversal",
			tenantSlug:  "attacker-tenant",
			malicious:   "%2e%2e/victim-tenant/dogs/photo.jpg",
			description: "URL-encoded path traversal",
		},
		{
			name:        "null byte injection",
			tenantSlug:  "attacker-tenant",
			malicious:   "dogs/photo.jpg\x00../../victim/secret.txt",
			description: "Null byte to bypass extension checks",
		},
	}

	for _, tt := range attackPaths {
		t.Run(tt.name, func(t *testing.T) {
			// Path validation should reject these
			err := validateS3Path(tt.malicious)
			if err == nil {
				t.Errorf("SECURITY: Path validation should reject %q (%s)", tt.malicious, tt.description)
			}
		})
	}
}

// TestTenantIsolation_TenantSlugValidation verifies that tenant slugs themselves
// cannot be used for path traversal
func TestTenantIsolation_TenantSlugValidation(t *testing.T) {
	maliciousSlugs := []struct {
		name   string
		slug   string
		reason string
	}{
		{
			name:   "traversal in slug",
			slug:   "../victim-tenant",
			reason: "Slug with parent directory reference",
		},
		{
			name:   "absolute path slug",
			slug:   "/etc/passwd",
			reason: "Slug starting with /",
		},
		{
			name:   "backslash in slug",
			slug:   "tenant\\..\\victim",
			reason: "Windows-style traversal in slug",
		},
		{
			name:   "null byte in slug",
			slug:   "tenant\x00/../victim",
			reason: "Null byte injection in slug",
		},
	}

	for _, tt := range maliciousSlugs {
		t.Run(tt.name, func(t *testing.T) {
			// validateS3Path is called on both slug and path in Upload
			err := validateS3Path(tt.slug)
			if err == nil {
				t.Errorf("SECURITY: Tenant slug validation should reject %q (%s)", tt.slug, tt.reason)
			}
		})
	}
}

// TestTenantIsolation_PublicURLFormat verifies public URLs maintain tenant isolation
func TestTenantIsolation_PublicURLFormat(t *testing.T) {
	s := &S3Service{
		publicURL:  "https://storage.gassigeher.org",
		bucketName: "gassigeher-uploads",
	}

	tests := []struct {
		tenantSlug  string
		path        string
		expectedURL string
	}{
		{
			tenantSlug:  "tierheim-goeppingen",
			path:        "dogs/photo.jpg",
			expectedURL: "https://storage.gassigeher.org/tenants/tierheim-goeppingen/dogs/photo.jpg",
		},
		{
			tenantSlug:  "shelter-munich",
			path:        "users/123/profile.jpg",
			expectedURL: "https://storage.gassigeher.org/tenants/shelter-munich/users/123/profile.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.tenantSlug+"/"+tt.path, func(t *testing.T) {
			objectKey, err := s.GetObjectKey(tt.tenantSlug, tt.path)
			if err != nil {
				t.Fatalf("GetObjectKey() unexpected error: %v", err)
			}
			publicURL := s.GetPublicURL(objectKey)

			if publicURL != tt.expectedURL {
				t.Errorf("GetPublicURL() = %q, want %q", publicURL, tt.expectedURL)
			}

			// Verify URL contains tenant isolation prefix
			if !strings.Contains(publicURL, "/tenants/"+tt.tenantSlug+"/") {
				t.Errorf("Public URL %q missing tenant isolation prefix", publicURL)
			}
		})
	}
}

// TestTenantIsolation_CannotAccessOtherTenantPaths ensures a tenant's path
// generation can never reference another tenant's namespace
func TestTenantIsolation_CannotAccessOtherTenantPaths(t *testing.T) {
	s := &S3Service{
		publicURL:  "https://storage.example.com",
		bucketName: "test-bucket",
	}

	// Tenant A trying to generate paths that would access Tenant B's files
	tenantA := "tenant-a"
	tenantB := "tenant-b"

	// Generate a normal key for tenant A
	keyA, err := s.GetObjectKey(tenantA, "dogs/photo.jpg")
	if err != nil {
		t.Fatalf("GetObjectKey() unexpected error: %v", err)
	}

	// The key should ONLY contain tenant A's namespace
	if strings.Contains(keyA, tenantB) {
		t.Errorf("Tenant A's object key contains tenant B's namespace: %s", keyA)
	}

	// Verify tenant A's key is properly namespaced
	expectedPrefix := "tenants/" + tenantA + "/"
	if !strings.HasPrefix(keyA, expectedPrefix) {
		t.Errorf("Object key %q doesn't start with expected prefix %q", keyA, expectedPrefix)
	}

	// Even if tenant A provides tenant B's slug in the path, it should stay in A's namespace
	// (validateS3Path would reject this, but let's verify the key generation anyway)
	pathWithTenantB := "dogs/../../../tenants/" + tenantB + "/secret.jpg"
	err = validateS3Path(pathWithTenantB)
	if err == nil {
		t.Error("Path validation should reject paths containing traversal attempts")
	}
}

// TestTenantIsolation_EmptyInputs tests edge cases with empty inputs
func TestTenantIsolation_EmptyInputs(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{name: "empty string", input: "", wantError: true},
		{name: "whitespace only", input: "   ", wantError: false}, // spaces are technically valid chars
		{name: "just a dot", input: ".", wantError: false},        // current dir, not traversal
		{name: "just two dots", input: "..", wantError: true},     // parent dir - blocked
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateS3Path(tt.input)
			gotError := err != nil
			if gotError != tt.wantError {
				t.Errorf("validateS3Path(%q) error=%v, wantError=%v", tt.input, err, tt.wantError)
			}
		})
	}
}
