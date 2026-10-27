package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/tranmh/gassigeher/internal/database"
)

var benchSinkAqaa interface{}

var benchSizesAqaa = []int{1, 10, 100}

func newBenchTenantRateLimiterAqaa(b *testing.B) *TenantRateLimiter {
	b.Helper()
	sqlxDB, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		b.Fatalf("open sqlite: %v", err)
	}
	sqlxDB.SetMaxOpenConns(1)
	db := database.WrapDB(sqlxDB, database.NewSQLiteDialect())
	trl := NewTenantRateLimiter(db)
	b.Cleanup(func() {
		trl.Close()
		sqlxDB.Close()
	})
	return trl
}

// BenchmarkTenantRateLimiter_GetLimiters measures getTenantLimiter+getIPLimiter per
// request for n tenants, with fresh cache entries versus entries older than the
// 5-minute tier cache (which are never refreshed and hit the DB on every call).
func BenchmarkTenantRateLimiter_GetLimiters(b *testing.B) {
	for _, state := range []string{"fresh", "stale"} {
		for _, n := range benchSizesAqaa {
			b.Run(fmt.Sprintf("%s/tenants=%d", state, n), func(b *testing.B) {
				trl := newBenchTenantRateLimiterAqaa(b)
				for id := 1; id <= n; id++ {
					trl.getTenantLimiter(id)
					trl.getIPLimiter(id, "10.0.0.1")
				}
				if state == "stale" {
					trl.tenantMu.Lock()
					for _, e := range trl.tenantLimiters {
						e.cacheTime = time.Now().Add(-10 * time.Minute)
					}
					trl.tenantMu.Unlock()
				}
				b.ReportAllocs()
				for b.Loop() {
					for id := 1; id <= n; id++ {
						benchSinkAqaa = trl.getTenantLimiter(id)
						benchSinkAqaa = trl.getIPLimiter(id, "10.0.0.1")
					}
				}
			})
		}
	}
}

// BenchmarkRateLimitAuthEndpoint serves one request after n distinct client IPs
// have been seen; the map keeps an entry per IP (reported as entries).
func BenchmarkRateLimitAuthEndpoint(b *testing.B) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := RateLimitAuthEndpoint(next)
	for _, n := range []int{100, 10000, 100000} {
		b.Run(fmt.Sprintf("ips=%d", n), func(b *testing.B) {
			ResetAuthRateLimiter()
			b.Cleanup(ResetAuthRateLimiter)
			for i := 0; i < n; i++ {
				req := httptest.NewRequest(http.MethodPost, "/api/auth/register", nil)
				req.RemoteAddr = fmt.Sprintf("10.%d.%d.%d:1234", (i>>16)&255, (i>>8)&255, i&255)
				handler.ServeHTTP(httptest.NewRecorder(), req)
			}
			req := httptest.NewRequest(http.MethodPost, "/api/auth/register", nil)
			req.RemoteAddr = "192.168.200.1:1234"
			b.ReportAllocs()
			for b.Loop() {
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
				benchSinkAqaa = rec.Code
			}
			authEndpointLimiter.mu.Lock()
			b.ReportMetric(float64(len(authEndpointLimiter.requests)), "entries")
			authEndpointLimiter.mu.Unlock()
		})
	}
}

// BenchmarkRateLimitLogin serves requests from n distinct client IPs per op.
func BenchmarkRateLimitLogin(b *testing.B) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := RateLimitLogin(next)
	for _, n := range benchSizesAqaa {
		b.Run(fmt.Sprintf("ips=%d", n), func(b *testing.B) {
			ResetRateLimiter()
			b.Cleanup(ResetRateLimiter)
			reqs := make([]*http.Request, n)
			for i := range reqs {
				reqs[i] = httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
				reqs[i].RemoteAddr = fmt.Sprintf("10.1.%d.%d:1234", (i>>8)&255, i&255)
			}
			b.ReportAllocs()
			for b.Loop() {
				for _, req := range reqs {
					rec := httptest.NewRecorder()
					handler.ServeHTTP(rec, req)
					benchSinkAqaa = rec.Code
				}
			}
		})
	}
}

// BenchmarkCSRFMiddleware covers the safe-method, skip-path and token-check paths.
func BenchmarkCSRFMiddleware(b *testing.B) {
	m := NewCSRFMiddleware()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := m.Middleware(next)
	token := m.GenerateToken()

	get := httptest.NewRequest(http.MethodGet, "/api/dogs", nil)
	getWithCookie := httptest.NewRequest(http.MethodGet, "/api/dogs", nil)
	getWithCookie.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: token})
	skip := httptest.NewRequest(http.MethodPost, "/api/health", nil)
	valid := httptest.NewRequest(http.MethodPost, "/api/bookings", nil)
	valid.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: token})
	valid.Header.Set(CSRFHeaderName, token)
	missing := httptest.NewRequest(http.MethodPost, "/api/bookings", nil)

	cases := []struct {
		name string
		req  *http.Request
	}{
		{"get_no_cookie", get},
		{"get_with_cookie", getWithCookie},
		{"post_skip_path", skip},
		{"post_valid_token", valid},
		{"post_missing_token", missing},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, c.req)
				benchSinkAqaa = rec.Code
			}
		})
	}
}
