package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// TestAPIVersionRedirect_RewritesLegacyRoutes tests that /api/* routes are rewritten to /api/v1/*
// This is a critical test because the middleware must work even for routes that don't exist
// at the /api/* path - they only exist at /api/v1/*
func TestAPIVersionRedirect_RewritesLegacyRoutes(t *testing.T) {
	// Create a router
	router := mux.NewRouter()

	// Register a handler at /api/v1/test (NOT at /api/test)
	var capturedPath string
	router.HandleFunc("/api/v1/test", func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// Wrap the router with version redirect (this runs BEFORE route matching)
	wrappedRouter := WrapWithVersionRedirect(router)

	// Request to /api/test should be rewritten to /api/v1/test
	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()
	wrappedRouter.ServeHTTP(rec, req)

	// Should return 200 (not 404)
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. The wrapper should rewrite /api/test to /api/v1/test", rec.Code)
	}

	// The handler should see the rewritten path
	if capturedPath != "/api/v1/test" {
		t.Errorf("Expected handler to see path '/api/v1/test', got '%s'", capturedPath)
	}
}

// TestAPIVersionRedirect_PreservesQueryParams tests that query parameters are preserved during rewrite
func TestAPIVersionRedirect_PreservesQueryParams(t *testing.T) {
	router := mux.NewRouter()

	var capturedQuery string
	router.HandleFunc("/api/v1/search", func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}).Methods("GET")

	wrappedRouter := WrapWithVersionRedirect(router)

	req := httptest.NewRequest("GET", "/api/search?q=test&page=1", nil)
	rec := httptest.NewRecorder()
	wrappedRouter.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	if capturedQuery != "q=test&page=1" {
		t.Errorf("Expected query 'q=test&page=1', got '%s'", capturedQuery)
	}
}

// TestAPIVersionRedirect_PreservesMethod tests that POST/PUT/DELETE methods work correctly
func TestAPIVersionRedirect_PreservesMethod(t *testing.T) {
	router := mux.NewRouter()

	var capturedMethod string
	router.HandleFunc("/api/v1/resource", func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}).Methods("POST")

	wrappedRouter := WrapWithVersionRedirect(router)

	req := httptest.NewRequest("POST", "/api/resource", nil)
	rec := httptest.NewRecorder()
	wrappedRouter.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	if capturedMethod != "POST" {
		t.Errorf("Expected method 'POST', got '%s'", capturedMethod)
	}
}

// TestAPIVersionRedirect_SkipsVersionedRoutes tests that /api/v1/* routes are not double-prefixed
func TestAPIVersionRedirect_SkipsVersionedRoutes(t *testing.T) {
	router := mux.NewRouter()

	var capturedPath string
	router.HandleFunc("/api/v1/already-versioned", func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}).Methods("GET")

	wrappedRouter := WrapWithVersionRedirect(router)

	req := httptest.NewRequest("GET", "/api/v1/already-versioned", nil)
	rec := httptest.NewRecorder()
	wrappedRouter.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Should NOT be rewritten to /api/v1/v1/already-versioned
	if capturedPath != "/api/v1/already-versioned" {
		t.Errorf("Expected path '/api/v1/already-versioned', got '%s'", capturedPath)
	}
}

// TestAPIVersionRedirect_SkipsUnversionedRoutes tests that specific routes remain unversioned
func TestAPIVersionRedirect_SkipsUnversionedRoutes(t *testing.T) {
	router := mux.NewRouter()

	// Register health at /api/health (NOT /api/v1/health)
	router.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("healthy"))
	}).Methods("GET")

	wrappedRouter := WrapWithVersionRedirect(router)

	req := httptest.NewRequest("GET", "/api/health", nil)
	rec := httptest.NewRecorder()
	wrappedRouter.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

// ============================================================================
// APIVersionRedirect (deprecated middleware) Tests
// ============================================================================

func TestAPIVersionRedirect_Middleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Captured-Path", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	})

	middleware := APIVersionRedirect(handler)

	t.Run("rewrites legacy api path", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/dogs", nil)
		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)

		if capturedPath := rec.Header().Get("X-Captured-Path"); capturedPath != "/api/v1/dogs" {
			t.Errorf("Expected path /api/v1/dogs, got %s", capturedPath)
		}
	})

	t.Run("preserves query params in rewrite", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/search?q=test", nil)
		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)

		if capturedPath := rec.Header().Get("X-Captured-Path"); capturedPath != "/api/v1/search" {
			t.Errorf("Expected path /api/v1/search, got %s", capturedPath)
		}
	})

	t.Run("skips already versioned paths", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/users", nil)
		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)

		if capturedPath := rec.Header().Get("X-Captured-Path"); capturedPath != "/api/v1/users" {
			t.Errorf("Expected path /api/v1/users, got %s", capturedPath)
		}
	})

	t.Run("skips health endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/health", nil)
		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)

		if capturedPath := rec.Header().Get("X-Captured-Path"); capturedPath != "/api/health" {
			t.Errorf("Expected path /api/health, got %s", capturedPath)
		}
	})
}

// ============================================================================
// AddVersionHeader Tests
// ============================================================================

func TestAddVersionHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := AddVersionHeader(handler)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	versionHeader := rec.Header().Get(APIVersionHeader)
	if versionHeader != CurrentAPIVersion {
		t.Errorf("Expected version header '%s', got '%s'", CurrentAPIVersion, versionHeader)
	}
}

// ============================================================================
// GetVersionInfo Tests
// ============================================================================

func TestGetVersionInfo(t *testing.T) {
	info := GetVersionInfo()

	if info.Current != CurrentAPIVersion {
		t.Errorf("Current = %s, want %s", info.Current, CurrentAPIVersion)
	}

	if len(info.Supported) == 0 {
		t.Error("Supported versions should not be empty")
	}

	// Check that current version is in supported list
	found := false
	for _, v := range info.Supported {
		if v == info.Current {
			found = true
			break
		}
	}
	if !found {
		t.Error("Current version should be in supported versions list")
	}
}

// ============================================================================
// rewriteAPIPath Unit Tests
// ============================================================================

func TestRewriteAPIPath(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantPath    string
		wantRewrite bool
	}{
		{"legacy api path", "/api/dogs", "/api/v1/dogs", true},
		{"already versioned v1", "/api/v1/dogs", "/api/v1/dogs", false},
		{"already versioned v2", "/api/v2/dogs", "/api/v2/dogs", false},
		{"non-api path", "/users", "/users", false},
		{"health endpoint", "/api/health", "/api/health", false},
		{"ready endpoint", "/api/ready", "/api/ready", false},
		{"version endpoint", "/api/version", "/api/version", false},
		{"metrics endpoint", "/api/metrics", "/api/metrics", false},
		{"root api path", "/api/", "/api/v1/", true},
		{"nested path", "/api/users/123/bookings", "/api/v1/users/123/bookings", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotRewrite := rewriteAPIPath(tt.input)
			if gotPath != tt.wantPath {
				t.Errorf("rewriteAPIPath(%q) path = %q, want %q", tt.input, gotPath, tt.wantPath)
			}
			if gotRewrite != tt.wantRewrite {
				t.Errorf("rewriteAPIPath(%q) rewrite = %v, want %v", tt.input, gotRewrite, tt.wantRewrite)
			}
		})
	}
}
