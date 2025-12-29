package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// CSRF token constants
const (
	CSRFCookieName = "csrf_token"
	CSRFHeaderName = "X-CSRF-Token"
	CSRFTokenLength = 32 // 32 bytes = 256 bits of entropy
)

// CSRFMiddleware implements Double-Submit Cookie pattern for CSRF protection
type CSRFMiddleware struct {
	tokenLength int
	cookieName  string
	headerName  string
	safeMethods map[string]bool
	skipPaths   []string
	secure      bool // Set to true in production (HTTPS)
}

// NewCSRFMiddleware creates a new CSRF middleware with default settings
func NewCSRFMiddleware() *CSRFMiddleware {
	return &CSRFMiddleware{
		tokenLength: CSRFTokenLength,
		cookieName:  CSRFCookieName,
		headerName:  CSRFHeaderName,
		safeMethods: map[string]bool{
			"GET":     true,
			"HEAD":    true,
			"OPTIONS": true,
			// TRACE intentionally excluded - blocked by BlockTraceMethod middleware
		},
		skipPaths: []string{
			"/api/v1/billing/webhook", // Stripe webhook endpoint
			"/api/health",              // Health check endpoints
		},
		secure: false, // Set via SetSecure() for production
	}
}

// SetSecure enables secure cookies (should be true in production with HTTPS)
func (m *CSRFMiddleware) SetSecure(secure bool) {
	m.secure = secure
}

// AddSkipPath adds a path prefix that should skip CSRF validation
func (m *CSRFMiddleware) AddSkipPath(path string) {
	m.skipPaths = append(m.skipPaths, path)
}

// GenerateToken creates a cryptographically secure random token
func (m *CSRFMiddleware) GenerateToken() string {
	bytes := make([]byte, m.tokenLength)
	if _, err := rand.Read(bytes); err != nil {
		// This should never happen - crypto/rand uses OS entropy source
		// Log error for debugging if it ever occurs
		log.Printf("CRITICAL: Failed to generate CSRF token: %v", err)
		return ""
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

// Middleware returns the CSRF protection middleware handler
func (m *CSRFMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip CSRF check for safe methods
		if m.safeMethods[r.Method] {
			// Generate token for safe requests if none exists
			m.ensureToken(w, r)
			next.ServeHTTP(w, r)
			return
		}

		// Skip CSRF check for exempt endpoints
		if m.shouldSkip(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Validate CSRF token for state-changing requests
		if !m.validateToken(r) {
			m.respondError(w, http.StatusForbidden, "CSRF token validation failed")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ensureToken sets a CSRF cookie if one doesn't exist
func (m *CSRFMiddleware) ensureToken(w http.ResponseWriter, r *http.Request) {
	// Check if token already exists
	// Base64 encoding of 32 bytes = ceil(32/3)*4 = 44 characters
	if cookie, err := r.Cookie(m.cookieName); err == nil && len(cookie.Value) >= 43 {
		return // Token exists and is valid length
	}

	// Generate new token
	token := m.GenerateToken()
	if token == "" {
		log.Printf("ERROR: CSRF token generation failed, client will not receive token")
		return // Token generation failed
	}

	// Set cookie - NOT HttpOnly so frontend can read it
	http.SetCookie(w, &http.Cookie{
		Name:     m.cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false, // Frontend needs to read this
		Secure:   m.secure,
		SameSite: http.SameSiteLaxMode, // Lax allows cookies on top-level navigation (email links)
		MaxAge:   86400, // 24 hours
	})
}

// validateToken checks that the cookie token matches the header token
func (m *CSRFMiddleware) validateToken(r *http.Request) bool {
	// Get token from cookie
	cookie, err := r.Cookie(m.cookieName)
	if err != nil || cookie.Value == "" {
		return false
	}

	// Get token from header
	headerToken := r.Header.Get(m.headerName)
	if headerToken == "" {
		return false
	}

	// Use constant-time comparison to prevent timing attacks
	// While timing attacks on CSRF are difficult, it's security best practice
	return subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(headerToken)) == 1
}

// shouldSkip checks if the path should skip CSRF validation
func (m *CSRFMiddleware) shouldSkip(path string) bool {
	for _, skipPath := range m.skipPaths {
		// Exact match or path is a subpath (has trailing /)
		if path == skipPath || strings.HasPrefix(path, skipPath+"/") {
			return true
		}
	}
	return false
}

// respondError sends a JSON error response
func (m *CSRFMiddleware) respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		log.Printf("ERROR: Failed to encode CSRF error response: %v", err)
	}
}
