package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
)

// setupTestRouter creates a router with the HTML page routes for testing
func setupTestRouter(t *testing.T, frontendDir string) *mux.Router {
	router := mux.NewRouter()

	// Serve specific HTML pages without .html extension
	router.HandleFunc("/verify", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(frontendDir, "verify.html"))
	}).Methods("GET")
	router.HandleFunc("/reset-password", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(frontendDir, "reset-password.html"))
	}).Methods("GET")
	router.HandleFunc("/forgot-password", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(frontendDir, "forgot-password.html"))
	}).Methods("GET")

	// Static files
	router.PathPrefix("/").Handler(http.FileServer(http.Dir(frontendDir)))

	return router
}

// TestHTMLPageRoutes tests that HTML pages are accessible without .html extension
func TestHTMLPageRoutes(t *testing.T) {
	// Create temporary frontend directory with test HTML files
	tempDir := t.TempDir()

	// Create test HTML files
	testPages := map[string]string{
		"verify.html":          "<html><body>Verify Page</body></html>",
		"reset-password.html":  "<html><body>Reset Password Page</body></html>",
		"forgot-password.html": "<html><body>Forgot Password Page</body></html>",
	}

	for filename, content := range testPages {
		err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	router := setupTestRouter(t, tempDir)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "verify page without extension",
			path:           "/verify",
			expectedStatus: http.StatusOK,
			expectedBody:   "Verify Page",
		},
		{
			name:           "verify page with token query param",
			path:           "/verify?token=abc123",
			expectedStatus: http.StatusOK,
			expectedBody:   "Verify Page",
		},
		{
			name:           "reset-password page without extension",
			path:           "/reset-password",
			expectedStatus: http.StatusOK,
			expectedBody:   "Reset Password Page",
		},
		{
			name:           "reset-password page with token query param",
			path:           "/reset-password?token=abc123def456",
			expectedStatus: http.StatusOK,
			expectedBody:   "Reset Password Page",
		},
		{
			name:           "forgot-password page without extension",
			path:           "/forgot-password",
			expectedStatus: http.StatusOK,
			expectedBody:   "Forgot Password Page",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.path, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			if rr.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d for path %s", tc.expectedStatus, rr.Code, tc.path)
			}

			if tc.expectedBody != "" && rr.Code == http.StatusOK {
				body := rr.Body.String()
				if body == "" || !contains(body, tc.expectedBody) {
					t.Errorf("Expected body to contain %q, got %q", tc.expectedBody, body)
				}
			}
		})
	}
}

// TestHTMLPageRoutes_Without_RouteHandlers demonstrates the bug
// This test shows what happens when route handlers are missing
func TestHTMLPageRoutes_Without_RouteHandlers(t *testing.T) {
	// Create temporary frontend directory
	tempDir := t.TempDir()

	// Create reset-password.html file
	err := os.WriteFile(filepath.Join(tempDir, "reset-password.html"), []byte("<html>Reset</html>"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Router WITHOUT the route handler (simulating the bug)
	routerWithoutHandler := mux.NewRouter()
	routerWithoutHandler.PathPrefix("/").Handler(http.FileServer(http.Dir(tempDir)))

	// Router WITH the route handler (the fix)
	routerWithHandler := mux.NewRouter()
	routerWithHandler.HandleFunc("/reset-password", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(tempDir, "reset-password.html"))
	}).Methods("GET")
	routerWithHandler.PathPrefix("/").Handler(http.FileServer(http.Dir(tempDir)))

	t.Run("without route handler returns 404", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/reset-password?token=abc", nil)
		rr := httptest.NewRecorder()
		routerWithoutHandler.ServeHTTP(rr, req)

		// Static file server returns 404 for /reset-password (no .html)
		if rr.Code != http.StatusNotFound {
			t.Errorf("Expected 404 without route handler, got %d", rr.Code)
		}
	})

	t.Run("with route handler returns 200", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/reset-password?token=abc", nil)
		rr := httptest.NewRecorder()
		routerWithHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected 200 with route handler, got %d", rr.Code)
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
