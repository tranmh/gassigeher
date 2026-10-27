package middleware

import (
	"encoding/base64"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// makeCSRFToken builds a token in the documented format with the given timestamp.
func makeCSRFToken(ts time.Time, randomLen int) string {
	b := make([]byte, CSRFTimestampBytes+randomLen)
	binary.BigEndian.PutUint64(b[:CSRFTimestampBytes], uint64(ts.Unix()))
	for i := CSRFTimestampBytes; i < len(b); i++ {
		b[i] = byte(i)
	}
	return base64.URLEncoding.EncodeToString(b)
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func csrfCookie(rr *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rr.Result().Cookies() {
		if c.Name == CSRFCookieName {
			return c
		}
	}
	return nil
}

func TestCSRF_TokenValidity(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		token    string
		wantCode int
	}{
		{"fresh", makeCSRFToken(now, CSRFTokenLength), http.StatusOK},
		{"23h old", makeCSRFToken(now.Add(-23*time.Hour), CSRFTokenLength), http.StatusOK},
		{"25h old", makeCSRFToken(now.Add(-25*time.Hour), CSRFTokenLength), http.StatusForbidden},
		{"1 min in future", makeCSRFToken(now.Add(time.Minute), CSRFTokenLength), http.StatusOK},
		{"10 min in future", makeCSRFToken(now.Add(10*time.Minute), CSRFTokenLength), http.StatusForbidden},
		{"39 bytes", makeCSRFToken(now, CSRFTokenLength-1), http.StatusForbidden},
		{"not base64", "!!!not-base64!!!", http.StatusForbidden},
	}
	h := NewCSRFMiddleware().Middleware(okHandler())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/bookings", strings.NewReader("{}"))
			req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: tt.token})
			req.Header.Set(CSRFHeaderName, tt.token)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != tt.wantCode {
				t.Errorf("status %d, want %d", rr.Code, tt.wantCode)
			}
		})
	}
}

func TestCSRF_RejectionBodyAndMissingParts(t *testing.T) {
	token := makeCSRFToken(time.Now(), CSRFTokenLength)
	tests := []struct {
		name           string
		method         string
		cookie, header string
	}{
		{"cookie only", http.MethodPost, token, ""},
		{"header only", http.MethodPut, "", token},
		{"trace is not safe", "TRACE", "", ""},
		{"delete without token", http.MethodDelete, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			h := NewCSRFMiddleware().Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
			req := httptest.NewRequest(tt.method, "/api/dogs", nil)
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: tt.cookie})
			}
			if tt.header != "" {
				req.Header.Set(CSRFHeaderName, tt.header)
			}
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != http.StatusForbidden || called {
				t.Fatalf("status %d, next called %v; want 403 and not called", rr.Code, called)
			}
			if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q", ct)
			}
			if body := strings.TrimSpace(rr.Body.String()); body != `{"error":"CSRF token validation failed"}` {
				t.Errorf("body = %s", body)
			}
		})
	}
}

func TestCSRF_SkipPaths(t *testing.T) {
	m := NewCSRFMiddleware()
	m.AddSkipPath("/api/custom")
	h := m.Middleware(okHandler())
	tests := []struct {
		path     string
		wantCode int
	}{
		{"/api/health", http.StatusOK},
		{"/api/health/db", http.StatusOK},
		{"/api/healthx", http.StatusForbidden},
		{"/api/v1/billing/webhook", http.StatusOK},
		{"/api/v1/billing/webhooks", http.StatusForbidden},
		{"/api/custom", http.StatusOK},
		{"/api/custom/deep", http.StatusOK},
		{"/api/customer", http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, tt.path, nil))
			if rr.Code != tt.wantCode {
				t.Errorf("POST %s: status %d, want %d", tt.path, rr.Code, tt.wantCode)
			}
		})
	}
}

func TestCSRF_SafeMethodCookie(t *testing.T) {
	t.Run("new cookie attributes", func(t *testing.T) {
		m := NewCSRFMiddleware()
		m.SetSecure(true)
		rr := httptest.NewRecorder()
		m.Middleware(okHandler()).ServeHTTP(rr, httptest.NewRequest(http.MethodHead, "/x", nil))
		c := csrfCookie(rr)
		if c == nil {
			t.Fatal("no csrf cookie set")
		}
		if c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode || c.Path != "/" || c.MaxAge != 86400 {
			t.Errorf("unexpected cookie attributes: %+v", c)
		}
		raw, err := base64.URLEncoding.DecodeString(c.Value)
		if err != nil || len(raw) != CSRFTimestampBytes+CSRFTokenLength {
			t.Fatalf("token not base64url of 40 bytes: len=%d err=%v", len(raw), err)
		}
		ts := time.Unix(int64(binary.BigEndian.Uint64(raw[:8])), 0)
		if d := time.Since(ts); d < -time.Minute || d > time.Minute {
			t.Errorf("embedded timestamp %v not close to now", ts)
		}
	})

	t.Run("valid existing cookie kept", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: makeCSRFToken(time.Now(), CSRFTokenLength)})
		rr := httptest.NewRecorder()
		NewCSRFMiddleware().Middleware(okHandler()).ServeHTTP(rr, req)
		if c := csrfCookie(rr); c != nil {
			t.Errorf("cookie unexpectedly replaced: %v", c.Value)
		}
	})

	t.Run("expired cookie replaced", func(t *testing.T) {
		old := makeCSRFToken(time.Now().Add(-48*time.Hour), CSRFTokenLength)
		req := httptest.NewRequest(http.MethodOptions, "/x", nil)
		req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: old})
		rr := httptest.NewRecorder()
		NewCSRFMiddleware().Middleware(okHandler()).ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("status %d", rr.Code)
		}
		if c := csrfCookie(rr); c == nil || c.Value == old {
			t.Errorf("expected fresh cookie, got %v", c)
		}
	})
}

func loginReq(remote, xff string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.RemoteAddr = remote
	if xff != "" {
		req.Header.Set("X-Forwarded-For", xff)
	}
	return req
}

func TestRateLimitLogin_Threshold(t *testing.T) {
	ResetRateLimiter()
	t.Cleanup(ResetRateLimiter)
	h := RateLimitLogin(okHandler())
	for i := 1; i <= 5; i++ {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, loginReq("203.0.113.1:1000", ""))
		if rr.Code != http.StatusOK {
			t.Fatalf("request %d: status %d, want 200", i, rr.Code)
		}
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, loginReq("203.0.113.1:2000", ""))
	if rr.Code != http.StatusTooManyRequests || !strings.Contains(rr.Body.String(), "Zu viele Anmeldeversuche") {
		t.Errorf("6th request: status %d body %q", rr.Code, rr.Body.String())
	}
	// Another IP is unaffected
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, loginReq("203.0.113.2:1000", ""))
	if rr.Code != http.StatusOK {
		t.Errorf("other IP: status %d", rr.Code)
	}
}

func TestRateLimitLogin_WindowExpiry(t *testing.T) {
	ResetRateLimiter()
	t.Cleanup(ResetRateLimiter)
	old := time.Now().Add(-61 * time.Second)
	loginLimiter.mu.Lock()
	loginLimiter.requests["198.51.100.7"] = []time.Time{old, old, old, old, old}
	loginLimiter.mu.Unlock()

	rr := httptest.NewRecorder()
	RateLimitLogin(okHandler()).ServeHTTP(rr, loginReq("198.51.100.7:1", ""))
	if rr.Code != http.StatusOK {
		t.Errorf("requests older than a minute still counted: status %d", rr.Code)
	}
}

func TestRateLimitLogin_ResetClearsState(t *testing.T) {
	ResetRateLimiter()
	t.Cleanup(ResetRateLimiter)
	h := RateLimitLogin(okHandler())
	for i := 0; i < 5; i++ {
		h.ServeHTTP(httptest.NewRecorder(), loginReq("198.51.100.8:1", ""))
	}
	ResetRateLimiter()
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, loginReq("198.51.100.8:1", ""))
	if rr.Code != http.StatusOK {
		t.Errorf("after reset: status %d", rr.Code)
	}
}

func TestRateLimitLogin_TrustedProxies(t *testing.T) {
	ResetRateLimiter()
	SetTrustedProxies([]string{"10.0.0.1"})
	t.Cleanup(func() { SetTrustedProxies(nil); ResetRateLimiter() })
	h := RateLimitLogin(okHandler())

	t.Run("untrusted peer cannot spoof XFF", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			h.ServeHTTP(httptest.NewRecorder(), loginReq("192.0.2.9:1", "1.1.1."+string(rune('0'+i))))
		}
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, loginReq("192.0.2.9:1", "8.8.8.8"))
		if rr.Code != http.StatusTooManyRequests {
			t.Errorf("status %d, want 429", rr.Code)
		}
	})

	t.Run("trusted proxy uses first XFF entry", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, loginReq("10.0.0.1:443", " 2.2.2.2 , 10.0.0.1"))
			if rr.Code != http.StatusOK {
				t.Fatalf("request %d: status %d", i, rr.Code)
			}
		}
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, loginReq("10.0.0.1:443", "2.2.2.2"))
		if rr.Code != http.StatusTooManyRequests {
			t.Errorf("client 2.2.2.2 not limited: %d", rr.Code)
		}
		rr = httptest.NewRecorder()
		h.ServeHTTP(rr, loginReq("10.0.0.1:443", "3.3.3.3"))
		if rr.Code != http.StatusOK {
			t.Errorf("client 3.3.3.3 behind same proxy limited: %d", rr.Code)
		}
	})
}

func TestRateLimitAuthEndpoint_SharedBucket(t *testing.T) {
	ResetAuthRateLimiter()
	t.Cleanup(ResetAuthRateLimiter)
	h := RateLimitAuthEndpoint(okHandler())
	paths := []string{"/api/auth/register", "/api/auth/forgot-password", "/api/auth/reset-password"}
	for _, p := range paths {
		req := httptest.NewRequest(http.MethodPost, p, nil)
		req.RemoteAddr = "203.0.113.50:1234"
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("%s: status %d", p, rr.Code)
		}
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", nil)
	req.RemoteAddr = "203.0.113.50:9999"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("4th request: status %d, want 429", rr.Code)
	}
	if rr.Header().Get("Content-Type") != "application/json" || rr.Header().Get("Retry-After") != "60" {
		t.Errorf("headers = %v", rr.Header())
	}
	if !strings.Contains(rr.Body.String(), "Zu viele Anfragen") {
		t.Errorf("body = %q", rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/auth/register", nil)
	req.RemoteAddr = "203.0.113.51:1"
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("other IP: status %d", rr.Code)
	}
}

func TestRateLimitAuthEndpoint_WindowExpiry(t *testing.T) {
	ResetAuthRateLimiter()
	t.Cleanup(ResetAuthRateLimiter)
	old := time.Now().Add(-2 * time.Minute)
	authEndpointLimiter.mu.Lock()
	authEndpointLimiter.requests["203.0.113.60"] = []time.Time{old, old, old}
	authEndpointLimiter.mu.Unlock()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", nil)
	req.RemoteAddr = "203.0.113.60:1"
	rr := httptest.NewRecorder()
	RateLimitAuthEndpoint(okHandler()).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("status %d, want 200", rr.Code)
	}
}
