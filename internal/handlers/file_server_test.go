package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestValidateFilePath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/uploads/dogs/rex.jpg", true},
		{"/", true},
		{"/a.b/c.d", true},
		{"/x\x00.jpg", false},
		{"/x%00.jpg", false},
		{"/x%2500.jpg", false},
		{"/../etc/passwd", false},
		{"/a/..", false},
		{"/%2e%2e/etc", false},
		{"/%2E%2E/etc", false},
		{"/%252e%252e/etc", false},
		{`/a\b`, false},
		{"/a%5cb", false},
		{"/a%5Cb", false},
		{"/a%255cb", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := ValidateFilePath(tt.path); got != tt.want {
				t.Errorf("ValidateFilePath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestSafeFileServer(t *testing.T) {
	fsys := fstest.MapFS{
		"hello.txt":    {Data: []byte("hi")},
		"sub/file.txt": {Data: []byte("nested")},
	}
	h := SafeFileServer(http.FS(fsys))

	tests := []struct {
		name     string
		target   string
		wantCode int
		wantBody string
	}{
		{"regular file", "/hello.txt", http.StatusOK, "hi"},
		{"nested file", "/sub/file.txt", http.StatusOK, "nested"},
		{"root directory", "/", http.StatusForbidden, ""},
		{"sub directory", "/sub", http.StatusForbidden, ""},
		{"sub directory trailing slash", "/sub/", http.StatusForbidden, ""},
		{"missing file", "/nope.txt", http.StatusNotFound, ""},
		{"traversal", "/../hello.txt", http.StatusBadRequest, ""},
		{"encoded traversal", "/%2e%2e/hello.txt", http.StatusBadRequest, ""},
		{"encoded backslash", "/sub%5cfile.txt", http.StatusBadRequest, ""},
		{"double encoded null", "/hello.txt%2500", http.StatusBadRequest, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, tt.target, nil))
			if rr.Code != tt.wantCode {
				t.Fatalf("GET %s: status %d, want %d (body %q)", tt.target, rr.Code, tt.wantCode, rr.Body.String())
			}
			if tt.wantBody != "" && rr.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", rr.Body.String(), tt.wantBody)
			}
		})
	}
}
