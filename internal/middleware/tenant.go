package middleware

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/tranmh/gassigeher/internal/repository"
)

// validSubdomainRegex validates that subdomains only contain safe characters
// Allowed: lowercase letters, digits, and hyphens (no starting/ending with hyphen)
var validSubdomainRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// TenantMiddleware resolves the tenant from the subdomain and adds it to the request context.
// Example: "tierheim-goeppingen.gassigeher.org" → tenant_id for "tierheim-goeppingen"
func TenantMiddleware(tenantRepo *repository.TenantRepository, baseDomain string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host := r.Host

			// Extract subdomain from host
			slug := extractSubdomain(host, baseDomain)

			// If no subdomain or it's "www", skip tenant resolution
			// This allows the landing page and central admin to work without a tenant
			if slug == "" || slug == "www" || slug == "admin" {
				next.ServeHTTP(w, r)
				return
			}

			// Lookup tenant by slug
			tenant, err := tenantRepo.FindBySlug(slug)
			if err != nil {
				http.Error(w, `{"error":"Interner Serverfehler"}`, http.StatusInternalServerError)
				return
			}

			if tenant == nil {
				http.Error(w, `{"error":"Tierheim nicht gefunden"}`, http.StatusNotFound)
				return
			}

			if tenant.Status != "active" {
				if tenant.Status == "suspended" {
					http.Error(w, `{"error":"Dieses Tierheim ist vorübergehend gesperrt"}`, http.StatusForbidden)
					return
				}
				http.Error(w, `{"error":"Tierheim nicht verfügbar"}`, http.StatusForbidden)
				return
			}

			// Inject tenant info into context
			ctx := context.WithValue(r.Context(), TenantIDKey, tenant.ID)
			ctx = context.WithValue(ctx, TenantSlugKey, tenant.Slug)
			ctx = context.WithValue(ctx, IsDemoKey, tenant.IsDemo)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractSubdomain extracts the subdomain from a host string.
// Examples:
//   - "tierheim-goeppingen.gassigeher.org" with baseDomain "gassigeher.org" → "tierheim-goeppingen"
//   - "gassigeher.org" with baseDomain "gassigeher.org" → ""
//   - "localhost:8080" with baseDomain "localhost" → ""
func extractSubdomain(host, baseDomain string) string {
	// Remove port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Handle localhost for development
	if host == "localhost" || host == "127.0.0.1" {
		return ""
	}

	// Check if baseDomain is set
	if baseDomain == "" {
		return ""
	}

	// Remove port from baseDomain if present
	if idx := strings.Index(baseDomain, ":"); idx != -1 {
		baseDomain = baseDomain[:idx]
	}

	// Check if host ends with baseDomain
	if !strings.HasSuffix(host, "."+baseDomain) {
		// Host doesn't have a subdomain (it IS the base domain)
		if host == baseDomain {
			return ""
		}
		return ""
	}

	// Extract subdomain
	subdomain := strings.TrimSuffix(host, "."+baseDomain)

	// Validate subdomain (no dots allowed - only first-level subdomains)
	if strings.Contains(subdomain, ".") {
		return ""
	}

	// Security: Validate subdomain contains only safe characters
	// This prevents SQL injection, null bytes, and other attacks
	// Allowed: lowercase letters, digits, and hyphens (DNS-safe characters)
	if !validSubdomainRegex.MatchString(subdomain) {
		return ""
	}

	return subdomain
}

// GetTenantID extracts the tenant ID from the request context.
// Returns 0 if no tenant is set (e.g., landing page, central admin).
func GetTenantID(r *http.Request) int {
	tenantID, _ := r.Context().Value(TenantIDKey).(int)
	return tenantID
}

// GetTenantSlug extracts the tenant slug from the request context.
// Returns empty string if no tenant is set.
func GetTenantSlug(r *http.Request) string {
	slug, _ := r.Context().Value(TenantSlugKey).(string)
	return slug
}

// RequireTenant is a middleware that ensures a tenant is present in the context.
// Use this for routes that must have a tenant (most API routes).
func RequireTenant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := GetTenantID(r)
		if tenantID == 0 {
			http.Error(w, `{"error":"Kein Tierheim ausgewählt"}`, http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// OptionalTenant is a middleware that allows requests without a tenant.
// Use this for routes that can work with or without a tenant (e.g., central admin).
func OptionalTenant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Just pass through - tenant may or may not be set
		next.ServeHTTP(w, r)
	})
}
