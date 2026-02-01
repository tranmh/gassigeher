package middleware

import (
	"net/http"
	"strings"
)

// APIVersionHeader is the header used to specify API version
const APIVersionHeader = "X-API-Version"

// CurrentAPIVersion is the current API version
const CurrentAPIVersion = "v1"

// SupportedVersions lists all supported API versions
var SupportedVersions = []string{"v1"}

// rewriteAPIPath rewrites legacy /api/* paths to /api/v1/* paths
// Returns the rewritten path, or the original path if no rewrite is needed
func rewriteAPIPath(path string) (string, bool) {
	// Skip if already versioned (starts with /api/v followed by a number)
	if strings.HasPrefix(path, "/api/v") && len(path) > 6 {
		// Check if character after /api/v is a digit
		if path[6] >= '0' && path[6] <= '9' {
			return path, false
		}
	}

	// Skip non-API routes
	if !strings.HasPrefix(path, "/api/") {
		return path, false
	}

	// Skip specific routes that should remain unversioned
	unversionedRoutes := []string{
		"/api/health",
		"/api/ready",
		"/api/version",
		"/api/metrics",
		"/api/calendar/feed", // Calendar iCal feed (public, auth via token in URL)
	}
	for _, route := range unversionedRoutes {
		if strings.HasPrefix(path, route) {
			return path, false
		}
	}

	// Rewrite URL path from /api/* to /api/v1/*
	newPath := "/api/v1" + strings.TrimPrefix(path, "/api")
	return newPath, true
}

// APIVersionRedirect rewrites legacy /api/* routes to /api/v1/*
// This ensures backwards compatibility while transitioning to versioned APIs
// NOTE: This middleware MUST be applied via WrapWithVersionRedirect() to work correctly.
// Using router.Use() will NOT work because gorilla/mux runs middleware AFTER route matching.
// DEPRECATED: Use WrapWithVersionRedirect instead for correct behavior.
func APIVersionRedirect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		newPath, rewritten := rewriteAPIPath(r.URL.Path)
		if rewritten {
			r.URL.Path = newPath
			r.RequestURI = newPath
			if r.URL.RawQuery != "" {
				r.RequestURI = newPath + "?" + r.URL.RawQuery
			}
		}
		next.ServeHTTP(w, r)
	})
}

// WrapWithVersionRedirect wraps an http.Handler (typically a router) to rewrite
// legacy /api/* routes to /api/v1/* BEFORE route matching occurs.
//
// This is different from APIVersionRedirect middleware because it runs BEFORE
// the router sees the request, allowing the rewrite to affect route matching.
//
// Usage in main.go:
//
//	router := mux.NewRouter()
//	// ... register routes ...
//	wrappedRouter := middleware.WrapWithVersionRedirect(router)
//	http.ListenAndServe(":8080", wrappedRouter)
func WrapWithVersionRedirect(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		newPath, rewritten := rewriteAPIPath(r.URL.Path)
		if rewritten {
			r.URL.Path = newPath
			r.RequestURI = newPath
			if r.URL.RawQuery != "" {
				r.RequestURI = newPath + "?" + r.URL.RawQuery
			}
		}
		handler.ServeHTTP(w, r)
	})
}

// AddVersionHeader adds the API version header to responses
func AddVersionHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(APIVersionHeader, CurrentAPIVersion)
		next.ServeHTTP(w, r)
	})
}

// VersionInfo returns information about the API version
type VersionInfo struct {
	Current    string   `json:"current"`
	Supported  []string `json:"supported"`
	Deprecated []string `json:"deprecated,omitempty"`
}

// GetVersionInfo returns the current version information
func GetVersionInfo() VersionInfo {
	return VersionInfo{
		Current:   CurrentAPIVersion,
		Supported: SupportedVersions,
	}
}
