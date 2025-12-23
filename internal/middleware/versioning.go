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

// APIVersionRedirect rewrites legacy /api/* routes to /api/v1/*
// This ensures backwards compatibility while transitioning to versioned APIs
// NOTE: Uses URL rewriting (not HTTP redirect) to support POST/PUT/DELETE methods
func APIVersionRedirect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Skip if already versioned (starts with /api/v followed by a number)
		if strings.HasPrefix(path, "/api/v") && len(path) > 6 {
			// Check if character after /api/v is a digit
			if path[6] >= '0' && path[6] <= '9' {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Skip non-API routes
		if !strings.HasPrefix(path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		// Skip specific routes that should remain unversioned
		unversionedRoutes := []string{
			"/api/health",
			"/api/ready",
			"/api/version",
			"/api/metrics",
		}
		for _, route := range unversionedRoutes {
			if strings.HasPrefix(path, route) {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Rewrite URL path from /api/* to /api/v1/* (in-place, no HTTP redirect)
		// This preserves the request method (POST, PUT, DELETE) and body
		newPath := "/api/v1" + strings.TrimPrefix(path, "/api")
		r.URL.Path = newPath
		r.RequestURI = newPath
		if r.URL.RawQuery != "" {
			r.RequestURI = newPath + "?" + r.URL.RawQuery
		}
		next.ServeHTTP(w, r)
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
	Current   string   `json:"current"`
	Supported []string `json:"supported"`
	Deprecated []string `json:"deprecated,omitempty"`
}

// GetVersionInfo returns the current version information
func GetVersionInfo() VersionInfo {
	return VersionInfo{
		Current:   CurrentAPIVersion,
		Supported: SupportedVersions,
	}
}
