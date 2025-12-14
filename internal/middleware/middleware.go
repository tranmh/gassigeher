package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/tranmh/gassigeher/internal/logging"
	"github.com/tranmh/gassigeher/internal/services"
)

type contextKey string

const UserIDKey contextKey = "userID"
const EmailKey contextKey = "email"
const IsAdminKey contextKey = "isAdmin"
const IsSuperAdminKey contextKey = "isSuperAdmin"       // DONE: Phase 3
const RequestIDKey contextKey = "requestID"
const OriginalUserIDKey contextKey = "originalUserID"   // Impersonation: Super-admin's real ID
const IsImpersonatingKey contextKey = "isImpersonating" // Impersonation: Boolean flag

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

			// Allowed origins for CORS (configurable base + additional domains)
			allowedOrigins := []string{
				baseURL,
				"https://gassi.cuong.net",
				"https://www.gassi.cuong.net",
			}

			origin := r.Header.Get("Origin")
			for _, allowedOrigin := range allowedOrigins {
				if origin == allowedOrigin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}

			// If no origin header or not in allowed list, allow same-origin requests
			if origin == "" {
				w.Header().Set("Access-Control-Allow-Origin", baseURL)
			}

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
				http.Error(w, `{"error":"Missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			// Extract token from "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, `{"error":"Invalid authorization header format"}`, http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			// Validate token
			authService := services.NewAuthService(jwtSecret, 24) // expiration not used here
			claims, err := authService.ValidateJWT(tokenString)
			if err != nil {
			http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized) // BUG FIX #3
				return
			}

			// Extract claims
			userID, ok := (*claims)["user_id"].(float64)
			if !ok {
				http.Error(w, `{"error":"Invalid token claims"}`, http.StatusUnauthorized)
				return
			}

			email, ok := (*claims)["email"].(string)
			if !ok {
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

			// Extract impersonation claims (if present)
			originalUserID := 0
			isImpersonating := false
			if impersonating, ok := (*claims)["impersonating"].(bool); ok && impersonating {
				isImpersonating = true
				if origID, ok := (*claims)["original_user_id"].(float64); ok {
					originalUserID = int(origID)
				}
			}

			// Add to context
			ctx := context.WithValue(r.Context(), UserIDKey, int(userID))
			ctx = context.WithValue(ctx, EmailKey, email)
			ctx = context.WithValue(ctx, IsAdminKey, isAdmin)
			ctx = context.WithValue(ctx, IsSuperAdminKey, isSuperAdmin) // DONE: Phase 3
			ctx = context.WithValue(ctx, IsImpersonatingKey, isImpersonating)
			ctx = context.WithValue(ctx, OriginalUserIDKey, originalUserID)

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

		// Content Security Policy
		// Note: img-src includes tierheim-goeppingen.de for the default site logo
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https://www.tierheim-goeppingen.de")

		next.ServeHTTP(w, r)
	})
}
