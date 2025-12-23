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
