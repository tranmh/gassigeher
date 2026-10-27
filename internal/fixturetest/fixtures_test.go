package fixturetest

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFixtures(t *testing.T) {
	c := &FixedClock{T: BaseTime}
	c.Advance(24 * time.Hour)
	if got := c.Now().Format("2006-01-02"); got != Date(1) {
		t.Fatalf("clock %s != Date(1) %s", got, Date(1))
	}
	req := JSONRequest(t, http.MethodPost, "/x", map[string]string{"a": "b"})
	h := &OKHandler{}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if h.Calls != 1 || rr.Code != http.StatusOK || req.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("unexpected: calls=%d code=%d", h.Calls, rr.Code)
	}
}
