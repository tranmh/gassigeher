package middleware

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/tranmh/gassigeher/internal/logging"
	"github.com/tranmh/gassigeher/internal/services"
)

type contextKey string

const UserIDKey contextKey = "userID"
const EmailKey contextKey = "email"
const IsAdminKey contextKey = "isAdmin"
const IsSuperAdminKey contextKey = "isSuperAdmin"         // DONE: Phase 3
const IsCentralAdminKey contextKey = "isCentralAdmin"     // SaaS: Platform-wide admin
const RequestIDKey contextKey = "requestID"
const OriginalUserIDKey contextKey = "originalUserID"   // Impersonation: Super-admin's real ID
const IsImpersonatingKey contextKey = "isImpersonating" // Impersonation: Boolean flag

// SaaS multi-tenancy context keys
const TenantIDKey contextKey = "tenantID"     // Tenant ID from subdomain or JWT
const TenantSlugKey contextKey = "tenantSlug" // Tenant slug (subdomain)
const IsDemoKey contextKey = "isDemo"         // Demo tenant flag

// LoggingMiddleware logs HTTP requests with comprehensive information
// Includes: timestamp, request ID, client IP, method, path, status code,
// duration, bytes in/out, user agent, and user ID (if authenticated)
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Generate request ID for tracing
		requestID := logging.GenerateRequestID()

		// Add request ID to context
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
		r = r.WithContext(ctx)

		// Wrap response writer to capture status code and bytes
		wrapped := logging.NewResponseWriter(w)

		// Add request ID to response headers for client-side debugging
		wrapped.Header().Set("X-Request-ID", requestID)

		// Get request body size
		var bytesIn int64
		if r.ContentLength > 0 {
			bytesIn = r.ContentLength
		}

		// Call next handler
		next.ServeHTTP(wrapped, r)

		// Build log entry
		entry := &logging.HTTPLogEntry{
			Timestamp:  start,
			RequestID:  requestID,
			Method:     r.Method,
			Path:       r.URL.Path,
			Query:      r.URL.RawQuery,
			StatusCode: wrapped.StatusCode(),
			Duration:   time.Since(start),
			BytesIn:    bytesIn,
			BytesOut:   wrapped.BytesWritten(),
			ClientIP:   logging.GetClientIP(r),
			UserAgent:  r.UserAgent(),
			Referer:    r.Referer(),
		}

		// Try to get user ID from context (set by AuthMiddleware)
		if userID, ok := r.Context().Value(UserIDKey).(int); ok {
			entry.UserID = userID
		}

		// Log the entry
		log.Println(entry.Format())
	})
}
// DONE: Enhanced logging with status code, request ID, client IP, etc.

// CORSMiddleware adds CORS headers
// BUG FIX #1: Restrict CORS to specific origins instead of "*"
// Accepts baseURL from config for dynamic CORS origin configuration
func CORSMiddleware(baseURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Default to localhost if baseURL not provided
			if baseURL == "" {
				baseURL = "http://localhost:8080"
			}

			// Allowed origins for CORS (configurable via BASE_URL and CORS_ALLOWED_ORIGINS)
			allowedOrigins := []string{
				baseURL,
			}
			// Add additional origins from environment variable if set (comma-separated)
			if extraOrigins := os.Getenv("CORS_ALLOWED_ORIGINS"); extraOrigins != "" {
				for _, origin := range strings.Split(extraOrigins, ",") {
					origin = strings.TrimSpace(origin)
					if origin != "" {
						allowedOrigins = append(allowedOrigins, origin)
					}
				}
			}

			origin := r.Header.Get("Origin")
			for _, allowedOrigin := range allowedOrigins {
				if origin == allowedOrigin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}

			// If no origin header, this is a same-origin request - no CORS headers needed
			// Only set CORS headers for cross-origin requests from allowed origins

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// DONE: BUG #1 FIXED - CORS now restricted to specific allowed origins

// AuthMiddleware validates JWT tokens
func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error":"Missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			// Extract token from "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error":"Invalid authorization header format"}`, http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			// Validate token
			authService := services.NewAuthService(jwtSecret, 24) // expiration not used here
			claims, err := authService.ValidateJWT(tokenString)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
				return
			}

			// Extract claims
			userID, ok := (*claims)["user_id"].(float64)
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error":"Invalid token claims"}`, http.StatusUnauthorized)
				return
			}

			email, ok := (*claims)["email"].(string)
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error":"Invalid token claims"}`, http.StatusUnauthorized)
				return
			}

			isAdmin, ok := (*claims)["is_admin"].(bool)
			if !ok {
				isAdmin = false
			}

			// DONE: Phase 3 - Extract is_super_admin claim
			isSuperAdmin, ok := (*claims)["is_super_admin"].(bool)
			if !ok {
				isSuperAdmin = false
			}

			// SaaS: Extract is_central_admin claim
			isCentralAdmin, ok := (*claims)["is_central_admin"].(bool)
			if !ok {
				isCentralAdmin = false
			}

			// Extract impersonation claims (if present)
			originalUserID := 0
			isImpersonating := false
			if impersonating, ok := (*claims)["impersonating"].(bool); ok && impersonating {
				isImpersonating = true
				if origID, ok := (*claims)["original_user_id"].(float64); ok {
					originalUserID = int(origID)
				}
			}

			// SaaS: Extract tenant_id from JWT
			jwtTenantID := 0
			if tid, ok := (*claims)["tenant_id"].(float64); ok {
				jwtTenantID = int(tid)
			}

			// SaaS: Validate JWT tenant_id matches subdomain tenant (if subdomain tenant is set)
			subdomainTenantID, _ := r.Context().Value(TenantIDKey).(int)
			if subdomainTenantID != 0 {
				// Subdomain tenant is set - JWT must have matching tenant_id
				if jwtTenantID == 0 {
					w.Header().Set("Content-Type", "application/json")
					http.Error(w, `{"error":"Token ohne Tierheim-ID ungültig"}`, http.StatusUnauthorized)
					return
				}
				if subdomainTenantID != jwtTenantID {
					w.Header().Set("Content-Type", "application/json")
					http.Error(w, `{"error":"Token für anderes Tierheim ungültig"}`, http.StatusUnauthorized)
					return
				}
			}

			// Add to context
			ctx := context.WithValue(r.Context(), UserIDKey, int(userID))
			ctx = context.WithValue(ctx, EmailKey, email)
			ctx = context.WithValue(ctx, IsAdminKey, isAdmin)
			ctx = context.WithValue(ctx, IsSuperAdminKey, isSuperAdmin)     // DONE: Phase 3
			ctx = context.WithValue(ctx, IsCentralAdminKey, isCentralAdmin) // SaaS: Central admin
			ctx = context.WithValue(ctx, IsImpersonatingKey, isImpersonating)
			ctx = context.WithValue(ctx, OriginalUserIDKey, originalUserID)
			// SaaS: Add tenant_id to context (prefer subdomain, fallback to JWT)
			if subdomainTenantID != 0 {
				ctx = context.WithValue(ctx, TenantIDKey, subdomainTenantID)
			} else if jwtTenantID != 0 {
				ctx = context.WithValue(ctx, TenantIDKey, jwtTenantID)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAdmin middleware checks if user is an admin
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isAdmin, ok := r.Context().Value(IsAdminKey).(bool)
		if !ok || !isAdmin {
			http.Error(w, `{"error":"Admin access required"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireSuperAdmin middleware checks if user is a super admin
// DONE: Phase 3 - New middleware for Super Admin only operations
func RequireSuperAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isSuperAdmin, ok := r.Context().Value(IsSuperAdminKey).(bool)
		if !ok || !isSuperAdmin {
			http.Error(w, `{"error":"Super Admin access required"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// DONE: Phase 3 - Middleware updates complete

// RequireCentralAdmin requires central admin access (platform-wide admin, not tied to tenant)
func RequireCentralAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isCentralAdmin, ok := r.Context().Value(IsCentralAdminKey).(bool)
		if !ok || !isCentralAdmin {
			http.Error(w, `{"error":"Central Admin access required"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// AllowImpersonationEnd is a middleware that allows impersonation tokens to access
// the end-impersonation endpoint. It checks if either:
// 1. The user is a central admin, OR
// 2. The request is from an impersonation session (has IsImpersonatingKey = true)
// This solves the problem where impersonation tokens couldn't end their own session
// because they don't have central admin privileges.
func AllowImpersonationEnd(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if user is central admin - always allowed
		isCentralAdmin, _ := r.Context().Value(IsCentralAdminKey).(bool)
		if isCentralAdmin {
			next.ServeHTTP(w, r)
			return
		}

		// Check if this is an impersonation session - allowed to end own session
		isImpersonating, _ := r.Context().Value(IsImpersonatingKey).(bool)
		if isImpersonating {
			// Verify there's an original user ID (extra safety check)
			originalUserID, ok := r.Context().Value(OriginalUserIDKey).(int)
			if ok && originalUserID > 0 {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Neither central admin nor impersonating - deny access
		http.Error(w, `{"error":"Access denied - requires central admin or active impersonation session"}`, http.StatusForbidden)
	})
}

// BlockDangerousMethods middleware blocks potentially dangerous HTTP methods
// SECURITY: GASSI-2025-004 - Block TRACE and TRACK methods to prevent XST attacks
func BlockDangerousMethods(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := strings.ToUpper(r.Method)
		if method == "TRACE" || method == "TRACK" {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("Method Not Allowed"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SecurityHeadersMiddleware adds security headers
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// Prevent MIME sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Enable XSS protection
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Enforce HTTPS in production
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Content Security Policy (Enhanced for XSS protection)
		// SECURITY: GASSI-2025-003 - 'unsafe-inline' still required in script-src
		// PROGRESS: login.html and register.html migrated to external JS (js/pages/*.js)
		// REMAINING: ~25 other HTML files still use inline scripts for page-specific logic
		// MITIGATION: All user-input handling uses textContent (not innerHTML) to prevent XSS
		// TODO: Continue migrating inline scripts to js/pages/*.js, then remove 'unsafe-inline'
		// Note: img-src includes tierheim-goeppingen.de for the default site logo
		// style-src has 'unsafe-inline' for inline styles (lower risk than scripts)
		// Note: cdn.jsdelivr.net is allowed for Shepherd.js guided tours library
		csp := strings.Join([]string{
			"default-src 'self'",
			"script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net",
			"style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net",
			"img-src 'self' data: https://www.tierheim-goeppingen.de",
			"font-src 'self' data:",
			"connect-src 'self'",
			"frame-ancestors 'none'",
			"form-action 'self'",
			"base-uri 'self'",
			"object-src 'none'",
		}, "; ")
		w.Header().Set("Content-Security-Policy", csp)

		next.ServeHTTP(w, r)
	})
}
