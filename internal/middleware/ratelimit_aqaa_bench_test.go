package middleware

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"sync/atomic"
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

// countingConnectorAqaa wraps the sqlite driver and counts prepared statements
// (every query goes through Prepare because the wrapped conn has no Queryer).
type countingConnectorAqaa struct {
	drv     driver.Driver
	queries *atomic.Int64
}

func (c *countingConnectorAqaa) Connect(context.Context) (driver.Conn, error) {
	conn, err := c.drv.Open(":memory:")
	if err != nil {
		return nil, err
	}
	return &countingConnAqaa{conn: conn, queries: c.queries}, nil
}

func (c *countingConnectorAqaa) Driver() driver.Driver { return c.drv }

type countingConnAqaa struct {
	conn    driver.Conn
	queries *atomic.Int64
}

func (c *countingConnAqaa) Prepare(query string) (driver.Stmt, error) {
	c.queries.Add(1)
	return c.conn.Prepare(query)
}
func (c *countingConnAqaa) Close() error              { return c.conn.Close() }
func (c *countingConnAqaa) Begin() (driver.Tx, error) { return c.conn.Begin() } //nolint:staticcheck

// TestTenantRateLimiter_StaleTierCacheQueryCount: after the 5-minute tier cache of a
// tenant expires, one refresh should serve later requests again; the number of
// subscription queries must not grow with the number of requests.
func TestTenantRateLimiter_StaleTierCacheQueryCount(t *testing.T) {
	probe, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	drv := probe.Driver()
	probe.Close()

	queries := &atomic.Int64{}
	sqlDB := sql.OpenDB(&countingConnectorAqaa{drv: drv, queries: queries})
	sqlDB.SetMaxOpenConns(1)
	sqlxDB := sqlx.NewDb(sqlDB, "sqlite")
	db := database.WrapDB(sqlxDB, database.NewSQLiteDialect())
	trl := NewTenantRateLimiter(db)
	t.Cleanup(func() {
		trl.Close()
		sqlxDB.Close()
	})

	const tenantID = 1
	trl.getTenantLimiter(tenantID)
	trl.getIPLimiter(tenantID, "10.0.0.1")

	countFor := func(n int) int64 {
		trl.tenantMu.Lock()
		trl.tenantLimiters[tenantID].cacheTime = time.Now().Add(-10 * time.Minute)
		trl.tenantMu.Unlock()
		queries.Store(0)
		for i := 0; i < n; i++ {
			trl.getTenantLimiter(tenantID)
			trl.getIPLimiter(tenantID, "10.0.0.1")
		}
		return queries.Load()
	}

	n := 50
	qn := countFor(n)
	q2n := countFor(2 * n)
	if qn == 0 {
		t.Fatalf("expected the expired tier cache to be refreshed from the DB, got 0 queries")
	}
	if q2n >= 2*qn {
		t.Errorf("subscription queries grow with requests after cache expiry: %d requests -> %d queries, %d requests -> %d queries",
			n, qn, 2*n, q2n)
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
