// Package fixturetest holds deterministic test doubles and builders shared by test packages.
package fixturetest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// BaseTime is the fixed "now" all fixtures derive dates from.
var BaseTime = time.Date(2030, time.June, 3, 9, 0, 0, 0, time.UTC) // a Monday

// FixedClock is a controllable clock.
type FixedClock struct{ T time.Time }

// Now returns the clock's current time.
func (c *FixedClock) Now() time.Time { return c.T }

// Advance moves the clock forward.
func (c *FixedClock) Advance(d time.Duration) { c.T = c.T.Add(d) }

// Date returns BaseTime shifted by days, formatted as YYYY-MM-DD.
func Date(days int) string { return BaseTime.AddDate(0, 0, days).Format("2006-01-02") }

// JSONRequest builds a request with a JSON body.
func JSONRequest(t *testing.T, method, path string, body interface{}) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	return req
}

// OKHandler is a next handler that records whether it was called.
type OKHandler struct{ Calls int }

func (h *OKHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	h.Calls++
	w.WriteHeader(http.StatusOK)
}
