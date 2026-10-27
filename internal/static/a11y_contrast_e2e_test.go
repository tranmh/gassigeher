//go:build e2e

package static

import (
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// relativeLuminance returns the WCAG relative luminance of a #rrggbb colour
func relativeLuminance(t *testing.T, hex string) float64 {
	t.Helper()
	hex = strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	if len(hex) != 6 {
		t.Fatalf("unsupported colour %q", hex)
	}
	channel := func(s string) float64 {
		v, err := strconv.ParseUint(s, 16, 8)
		if err != nil {
			t.Fatalf("invalid colour %q: %v", hex, err)
		}
		c := float64(v) / 255
		if c <= 0.03928 {
			return c / 12.92
		}
		return math.Pow((c+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(hex[0:2]) + 0.7152*channel(hex[2:4]) + 0.0722*channel(hex[4:6])
}

func contrastRatio(t *testing.T, fg, bg string) float64 {
	l1, l2 := relativeLuminance(t, fg), relativeLuminance(t, bg)
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

// resolveCSSColor resolves var(--name, fallback) against :root custom properties
func resolveCSSColor(css, value string) string {
	value = strings.TrimSpace(value)
	varRe := regexp.MustCompile(`^var\(\s*(--[\w-]+)\s*(?:,\s*([^)]+))?\)$`)
	m := varRe.FindStringSubmatch(value)
	if m == nil {
		return value
	}
	propRe := regexp.MustCompile(`:root\s*\{[^}]*` + regexp.QuoteMeta(m[1]) + `\s*:\s*([^;}]+)`)
	if p := propRe.FindStringSubmatch(css); p != nil {
		return resolveCSSColor(css, p[1])
	}
	return strings.TrimSpace(m[2])
}

// TestE2E_ProfilePage_ForgotPasswordLinkContrast checks WCAG 1.4.3 (color-contrast) for the
// "Passwort vergessen?" link shown on /profile.html (login form) on its white card background.
func TestE2E_ProfilePage_ForgotPasswordLinkContrast(t *testing.T) {
	fsys, err := FrontendFS()
	if err != nil {
		t.Fatalf("FrontendFS() returned error: %v", err)
	}
	srv := httptest.NewServer(http.FileServer(http.FS(fsys)))
	defer srv.Close()

	get := func(path string) string {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET %s: status %d", path, resp.StatusCode)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		return string(body)
	}

	get("/profile.html")
	login := get("/login.html")
	if !strings.Contains(login, `<a href="/forgot-password.html" data-i18n="auth.forgot_password">`) {
		t.Fatal("login form has no forgot-password link")
	}
	if !strings.Contains(login, `/assets/css/main.css`) {
		t.Fatal("login form does not load main.css")
	}

	css := get("/assets/css/main.css")
	ruleRe := regexp.MustCompile(`(?:^|[}\s])a\s*\{([^}]*)\}`)
	rule := ruleRe.FindStringSubmatch(strings.TrimPrefix(css, string(rune(0xFEFF))))
	if rule == nil {
		t.Fatal("main.css has no rule for links (a{...})")
	}
	colorRe := regexp.MustCompile(`(?:^|;)\s*color\s*:\s*([^;]+)`)
	c := colorRe.FindStringSubmatch(rule[1])
	if c == nil {
		t.Fatal("link rule in main.css sets no colour")
	}
	fg := resolveCSSColor(css, c[1])

	ratio := contrastRatio(t, fg, "#ffffff")
	if ratio < 4.5 {
		t.Errorf("forgot-password link colour %s on #ffffff has contrast %.2f:1, WCAG 1.4.3 requires at least 4.5:1", fg, ratio)
	}
}
