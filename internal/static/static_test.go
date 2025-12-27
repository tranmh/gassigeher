package static

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

// TestFrontendFS_ReturnsValidFilesystem tests that FrontendFS returns a valid filesystem
func TestFrontendFS_ReturnsValidFilesystem(t *testing.T) {
	fsys, err := FrontendFS()
	if err != nil {
		t.Fatalf("FrontendFS() returned error: %v", err)
	}

	if fsys == nil {
		t.Fatal("FrontendFS() returned nil filesystem")
	}
}

// TestFrontendFS_ContainsExpectedFiles tests that critical frontend files are embedded
func TestFrontendFS_ContainsExpectedFiles(t *testing.T) {
	fsys, err := FrontendFS()
	if err != nil {
		t.Fatalf("FrontendFS() returned error: %v", err)
	}

	// List of critical files that must be embedded
	expectedFiles := []string{
		"index.html",
		"login.html",
		"register.html",
		"verify.html",
		"reset-password.html",
		"forgot-password.html",
		"dogs.html",
		"dashboard.html",
		"profile.html",
		"js/api.js",
		"js/i18n.js",
		"assets/css/main.css",
		"i18n/de.json",
	}

	for _, file := range expectedFiles {
		t.Run(file, func(t *testing.T) {
			_, err := fs.Stat(fsys, file)
			if err != nil {
				t.Errorf("Expected file %s not found in embedded filesystem: %v", file, err)
			}
		})
	}
}

// TestFrontendFS_ContainsAdminPages tests that admin pages are embedded
func TestFrontendFS_ContainsAdminPages(t *testing.T) {
	fsys, err := FrontendFS()
	if err != nil {
		t.Fatalf("FrontendFS() returned error: %v", err)
	}

	adminPages := []string{
		"admin-dashboard.html",
		"admin-dogs.html",
		"admin-bookings.html",
		"admin-users.html",
		"admin-settings.html",
		"admin-blocked-dates.html",
		"admin-experience-requests.html",
		"admin-reactivation-requests.html",
	}

	for _, file := range adminPages {
		t.Run(file, func(t *testing.T) {
			_, err := fs.Stat(fsys, file)
			if err != nil {
				t.Errorf("Admin page %s not found in embedded filesystem: %v", file, err)
			}
		})
	}
}

// TestFrontendFile_ReadsFileContent tests that FrontendFile returns actual content
func TestFrontendFile_ReadsFileContent(t *testing.T) {
	content, err := FrontendFile("index.html")
	if err != nil {
		t.Fatalf("FrontendFile(index.html) returned error: %v", err)
	}

	if len(content) == 0 {
		t.Error("FrontendFile(index.html) returned empty content")
	}

	// Verify it's actually HTML
	contentStr := string(content)
	if !strings.Contains(contentStr, "<html") && !strings.Contains(contentStr, "<!DOCTYPE") {
		t.Error("index.html does not appear to be valid HTML")
	}
}

// TestFrontendFile_ReturnsErrorForMissingFile tests error handling for missing files
func TestFrontendFile_ReturnsErrorForMissingFile(t *testing.T) {
	_, err := FrontendFile("nonexistent-file-12345.html")
	if err == nil {
		t.Error("FrontendFile should return error for nonexistent file")
	}
}

// TestFrontendFS_CanReadFileContent tests reading file content through the filesystem
func TestFrontendFS_CanReadFileContent(t *testing.T) {
	fsys, err := FrontendFS()
	if err != nil {
		t.Fatalf("FrontendFS() returned error: %v", err)
	}

	testCases := []struct {
		file            string
		expectedContent string
	}{
		{"verify.html", "<html"},
		{"reset-password.html", "<html"},
		{"forgot-password.html", "<html"},
		{"js/api.js", "class API"},
		{"i18n/de.json", "{"},
	}

	for _, tc := range testCases {
		t.Run(tc.file, func(t *testing.T) {
			content, err := fs.ReadFile(fsys, tc.file)
			if err != nil {
				t.Fatalf("Failed to read %s: %v", tc.file, err)
			}

			if len(content) == 0 {
				t.Errorf("File %s is empty", tc.file)
			}

			if !strings.Contains(string(content), tc.expectedContent) {
				t.Errorf("File %s does not contain expected content %q", tc.file, tc.expectedContent)
			}
		})
	}
}

// TestFrontendFS_DirectoryStructure tests that directories are properly embedded
func TestFrontendFS_DirectoryStructure(t *testing.T) {
	fsys, err := FrontendFS()
	if err != nil {
		t.Fatalf("FrontendFS() returned error: %v", err)
	}

	expectedDirs := []string{
		"js",
		"assets",
		"assets/css",
		"i18n",
	}

	for _, dir := range expectedDirs {
		t.Run(dir, func(t *testing.T) {
			info, err := fs.Stat(fsys, dir)
			if err != nil {
				t.Errorf("Directory %s not found: %v", dir, err)
				return
			}

			if !info.IsDir() {
				t.Errorf("%s should be a directory", dir)
			}
		})
	}
}

// TestFrontendFS_FilesNotEmpty tests that embedded files have actual content
func TestFrontendFS_FilesNotEmpty(t *testing.T) {
	fsys, err := FrontendFS()
	if err != nil {
		t.Fatalf("FrontendFS() returned error: %v", err)
	}

	// Walk through all files and check they're not empty
	err = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			t.Errorf("Failed to get info for %s: %v", path, err)
			return nil
		}

		if info.Size() == 0 {
			t.Errorf("File %s is empty (0 bytes)", path)
		}

		return nil
	})

	if err != nil {
		t.Errorf("Error walking filesystem: %v", err)
	}
}

// TestFrontendFS_Security_NoInlineEventHandlers tests that HTML files don't contain inline event handlers
// SECURITY: GASSI-2025-003 - Required for strict CSP without 'unsafe-inline'
func TestFrontendFS_Security_NoInlineEventHandlers(t *testing.T) {
	fsys, err := FrontendFS()
	if err != nil {
		t.Fatalf("FrontendFS() returned error: %v", err)
	}

	// Inline event handler patterns to detect in HTML attributes
	// These patterns match HTML attributes like onclick="..." but NOT JavaScript assignments like window.onclick = ...
	// The pattern requires = followed by " or ' (HTML attribute syntax)
	inlineHandlerPatterns := []string{
		`\bonclick\s*=\s*["']`,
		`\bonsubmit\s*=\s*["']`,
		`\bonchange\s*=\s*["']`,
		`\bonload\s*=\s*["']`,
		`\bonerror\s*=\s*["']`,
		`\bonkeyup\s*=\s*["']`,
		`\bonkeydown\s*=\s*["']`,
		`\bonkeypress\s*=\s*["']`,
		`\bonfocus\s*=\s*["']`,
		`\bonblur\s*=\s*["']`,
		`\bonmouseover\s*=\s*["']`,
		`\bonmouseout\s*=\s*["']`,
		`\bonmousedown\s*=\s*["']`,
		`\bonmouseup\s*=\s*["']`,
		`\boninput\s*=\s*["']`,
		`\bonmouseenter\s*=\s*["']`,
		`\bonmouseleave\s*=\s*["']`,
	}

	// Files to exclude from this check (test files that intentionally contain XSS patterns)
	excludeFiles := map[string]bool{
		"js/sanitize.test.html": true, // XSS sanitization test file
	}

	// Compile all patterns
	var compiledPatterns []*regexp.Regexp
	for _, pattern := range inlineHandlerPatterns {
		re := regexp.MustCompile(pattern)
		compiledPatterns = append(compiledPatterns, re)
	}

	// Walk through all HTML files
	err = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}

		// Skip excluded files
		if excludeFiles[path] {
			return nil
		}

		content, err := fs.ReadFile(fsys, path)
		if err != nil {
			t.Errorf("Failed to read %s: %v", path, err)
			return nil
		}

		contentStr := string(content)

		// Check for each inline handler pattern
		for i, re := range compiledPatterns {
			matches := re.FindAllString(contentStr, -1)
			if len(matches) > 0 {
				t.Errorf("SECURITY VIOLATION: %s contains %d inline %s handler(s) - breaks strict CSP",
					path, len(matches), inlineHandlerPatterns[i])
			}
		}

		return nil
	})

	if err != nil {
		t.Errorf("Error walking filesystem: %v", err)
	}
}

// ============================================================================
// Landing Page Filesystem Tests
// ============================================================================

// TestLandingFS_ReturnsValidFilesystem tests that LandingFS returns a valid filesystem
func TestLandingFS_ReturnsValidFilesystem(t *testing.T) {
	fsys, err := LandingFS()
	if err != nil {
		t.Fatalf("LandingFS() returned error: %v", err)
	}

	if fsys == nil {
		t.Fatal("LandingFS() returned nil filesystem")
	}
}

// TestLandingFS_ContainsExpectedFiles tests that critical landing page files are embedded
func TestLandingFS_ContainsExpectedFiles(t *testing.T) {
	fsys, err := LandingFS()
	if err != nil {
		t.Fatalf("LandingFS() returned error: %v", err)
	}

	// List of critical files that must be embedded in landing pages
	expectedFiles := []string{
		"index.html",
	}

	for _, file := range expectedFiles {
		t.Run(file, func(t *testing.T) {
			_, err := fs.Stat(fsys, file)
			if err != nil {
				t.Errorf("Expected file %s not found in landing filesystem: %v", file, err)
			}
		})
	}
}

// TestLandingFile_ReadsFileContent tests that LandingFile returns actual content
func TestLandingFile_ReadsFileContent(t *testing.T) {
	content, err := LandingFile("index.html")
	if err != nil {
		t.Fatalf("LandingFile(index.html) returned error: %v", err)
	}

	if len(content) == 0 {
		t.Error("LandingFile(index.html) returned empty content")
	}

	// Verify it's actually HTML
	contentStr := string(content)
	if !strings.Contains(contentStr, "<html") && !strings.Contains(contentStr, "<!DOCTYPE") {
		t.Error("landing/index.html does not appear to be valid HTML")
	}
}

// TestLandingFile_ReturnsErrorForMissingFile tests error handling for missing files
func TestLandingFile_ReturnsErrorForMissingFile(t *testing.T) {
	_, err := LandingFile("nonexistent-file-12345.html")
	if err == nil {
		t.Error("LandingFile should return error for nonexistent file")
	}
}

// TestLandingFS_CanReadFileContent tests reading file content through the filesystem
func TestLandingFS_CanReadFileContent(t *testing.T) {
	fsys, err := LandingFS()
	if err != nil {
		t.Fatalf("LandingFS() returned error: %v", err)
	}

	content, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		t.Fatalf("Failed to read landing index.html: %v", err)
	}

	if len(content) == 0 {
		t.Error("Landing index.html is empty")
	}

	if !strings.Contains(string(content), "<html") {
		t.Error("Landing index.html does not contain expected HTML tag")
	}
}

// ============================================================================
// Central Admin Filesystem Tests
// ============================================================================

// TestCentralFS_ReturnsValidFilesystem tests that CentralFS returns a valid filesystem
func TestCentralFS_ReturnsValidFilesystem(t *testing.T) {
	fsys, err := CentralFS()
	if err != nil {
		t.Fatalf("CentralFS() returned error: %v", err)
	}

	if fsys == nil {
		t.Fatal("CentralFS() returned nil filesystem")
	}
}

// TestCentralFS_ContainsExpectedFiles tests that critical central admin files are embedded
func TestCentralFS_ContainsExpectedFiles(t *testing.T) {
	fsys, err := CentralFS()
	if err != nil {
		t.Fatalf("CentralFS() returned error: %v", err)
	}

	// List of critical files that must be embedded in central admin
	expectedFiles := []string{
		"index.html",
	}

	for _, file := range expectedFiles {
		t.Run(file, func(t *testing.T) {
			_, err := fs.Stat(fsys, file)
			if err != nil {
				t.Errorf("Expected file %s not found in central filesystem: %v", file, err)
			}
		})
	}
}

// TestCentralFile_ReadsFileContent tests that CentralFile returns actual content
func TestCentralFile_ReadsFileContent(t *testing.T) {
	content, err := CentralFile("index.html")
	if err != nil {
		t.Fatalf("CentralFile(index.html) returned error: %v", err)
	}

	if len(content) == 0 {
		t.Error("CentralFile(index.html) returned empty content")
	}

	// Verify it's actually HTML
	contentStr := string(content)
	if !strings.Contains(contentStr, "<html") && !strings.Contains(contentStr, "<!DOCTYPE") {
		t.Error("central/index.html does not appear to be valid HTML")
	}
}

// TestCentralFile_ReturnsErrorForMissingFile tests error handling for missing files
func TestCentralFile_ReturnsErrorForMissingFile(t *testing.T) {
	_, err := CentralFile("nonexistent-file-12345.html")
	if err == nil {
		t.Error("CentralFile should return error for nonexistent file")
	}
}

// TestCentralFS_CanReadFileContent tests reading file content through the filesystem
func TestCentralFS_CanReadFileContent(t *testing.T) {
	fsys, err := CentralFS()
	if err != nil {
		t.Fatalf("CentralFS() returned error: %v", err)
	}

	content, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		t.Fatalf("Failed to read central index.html: %v", err)
	}

	if len(content) == 0 {
		t.Error("Central index.html is empty")
	}

	if !strings.Contains(string(content), "<html") {
		t.Error("Central index.html does not contain expected HTML tag")
	}
}

// ============================================================================
// Cross-Filesystem Consistency Tests
// ============================================================================

// TestAllFilesystems_DoNotContainEmptyFiles tests that no embedded files are empty
func TestAllFilesystems_DoNotContainEmptyFiles(t *testing.T) {
	filesystems := map[string]func() (fs.FS, error){
		"frontend": FrontendFS,
		"landing":  LandingFS,
		"central":  CentralFS,
	}

	for name, getFsys := range filesystems {
		t.Run(name, func(t *testing.T) {
			fsys, err := getFsys()
			if err != nil {
				t.Fatalf("%s FS returned error: %v", name, err)
			}

			err = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				if d.IsDir() {
					return nil
				}

				info, err := d.Info()
				if err != nil {
					t.Errorf("Failed to get info for %s/%s: %v", name, path, err)
					return nil
				}

				if info.Size() == 0 {
					t.Errorf("File %s/%s is empty (0 bytes)", name, path)
				}

				return nil
			})

			if err != nil {
				t.Errorf("Error walking %s filesystem: %v", name, err)
			}
		})
	}
}

// TestAllFilesystems_HTMLFilesAreValid tests that all HTML files contain valid HTML markers
func TestAllFilesystems_HTMLFilesAreValid(t *testing.T) {
	filesystems := map[string]func() (fs.FS, error){
		"frontend": FrontendFS,
		"landing":  LandingFS,
		"central":  CentralFS,
	}

	for name, getFsys := range filesystems {
		t.Run(name, func(t *testing.T) {
			fsys, err := getFsys()
			if err != nil {
				t.Fatalf("%s FS returned error: %v", name, err)
			}

			err = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				if d.IsDir() || !strings.HasSuffix(path, ".html") {
					return nil
				}

				content, err := fs.ReadFile(fsys, path)
				if err != nil {
					t.Errorf("Failed to read %s/%s: %v", name, path, err)
					return nil
				}

				contentStr := string(content)
				hasHTML := strings.Contains(contentStr, "<html") ||
					strings.Contains(contentStr, "<!DOCTYPE") ||
					strings.Contains(contentStr, "<!doctype")

				if !hasHTML {
					t.Errorf("File %s/%s does not appear to be valid HTML", name, path)
				}

				return nil
			})

			if err != nil {
				t.Errorf("Error walking %s filesystem: %v", name, err)
			}
		})
	}
}
