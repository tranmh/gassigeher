package static

import (
	"io/fs"
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
