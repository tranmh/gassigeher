package logging

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestDefaultConfig tests the DefaultConfig function
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.LogDir != "./logs" {
		t.Errorf("Expected LogDir './logs', got '%s'", cfg.LogDir)
	}
	if cfg.MaxAgeDays != 30 {
		t.Errorf("Expected MaxAgeDays 30, got %d", cfg.MaxAgeDays)
	}
	if cfg.CompressSizeMB != 10 {
		t.Errorf("Expected CompressSizeMB 10, got %d", cfg.CompressSizeMB)
	}
	if !cfg.ConsoleOutput {
		t.Error("Expected ConsoleOutput true, got false")
	}
}

// TestNewLogger tests creating a new logger
func TestNewLogger(t *testing.T) {
	t.Run("creates logger with default config", func(t *testing.T) {
		// Create temporary log directory
		tmpDir, err := os.MkdirTemp("", "logger-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		cfg := &Config{
			LogDir:        tmpDir,
			MaxAgeDays:    30,
			CompressSizeMB: 10,
			ConsoleOutput: false, // Disable console for tests
		}

		logger, err := NewLogger(cfg)
		if err != nil {
			t.Fatalf("NewLogger failed: %v", err)
		}
		defer logger.Close()

		// Verify log file was created
		today := time.Now().Format("2006-01-02")
		expectedFile := filepath.Join(tmpDir, "gassigeher_"+today+".log")
		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			t.Errorf("Expected log file %s to exist", expectedFile)
		}
	})

	t.Run("creates logger with nil config uses defaults", func(t *testing.T) {
		// Clean up default logs directory after test
		defer os.RemoveAll("./logs")

		logger, err := NewLogger(nil)
		if err != nil {
			t.Fatalf("NewLogger with nil config failed: %v", err)
		}
		defer logger.Close()

		if logger.logDir != "./logs" {
			t.Errorf("Expected logDir './logs', got '%s'", logger.logDir)
		}
	})

	t.Run("creates log directory if not exists", func(t *testing.T) {
		tmpDir := filepath.Join(os.TempDir(), "logger-test-new-dir-"+time.Now().Format("20060102150405"))
		defer os.RemoveAll(tmpDir)

		cfg := &Config{
			LogDir:        tmpDir,
			MaxAgeDays:    30,
			CompressSizeMB: 10,
			ConsoleOutput: false,
		}

		logger, err := NewLogger(cfg)
		if err != nil {
			t.Fatalf("NewLogger failed: %v", err)
		}
		defer logger.Close()

		// Verify directory was created
		if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
			t.Errorf("Expected log directory %s to exist", tmpDir)
		}
	})
}

// TestLoggerWrite tests the Write method
func TestLoggerWrite(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger-write-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &Config{
		LogDir:        tmpDir,
		MaxAgeDays:    30,
		CompressSizeMB: 10,
		ConsoleOutput: false,
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	defer logger.Close()

	t.Run("writes log message to file", func(t *testing.T) {
		message := "Test log message\n"
		n, err := logger.Write([]byte(message))
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}
		if n != len(message) {
			t.Errorf("Expected to write %d bytes, wrote %d", len(message), n)
		}

		// Verify message was written
		today := time.Now().Format("2006-01-02")
		logFile := filepath.Join(tmpDir, "gassigeher_"+today+".log")
		content, err := os.ReadFile(logFile)
		if err != nil {
			t.Fatalf("Failed to read log file: %v", err)
		}
		if !strings.Contains(string(content), "Test log message") {
			t.Errorf("Expected log file to contain 'Test log message', got '%s'", string(content))
		}
	})
}

// TestLoggerClose tests the Close method
func TestLoggerClose(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger-close-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &Config{
		LogDir:        tmpDir,
		MaxAgeDays:    30,
		CompressSizeMB: 10,
		ConsoleOutput: false,
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}

	err = logger.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Verify file is closed (writing should fail or create a new file)
	// After close, the currentFile should be nil or closed
}

// TestLoggerGetLogFiles tests the GetLogFiles method
func TestLoggerGetLogFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger-files-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create recent log files (within maxAgeDays to avoid cleanup)
	now := time.Now()
	dates := []string{
		now.AddDate(0, 0, -1).Format("2006-01-02"),
		now.AddDate(0, 0, -2).Format("2006-01-02"),
		now.AddDate(0, 0, -3).Format("2006-01-02"),
	}
	for _, date := range dates {
		filename := filepath.Join(tmpDir, "gassigeher_"+date+".log")
		os.WriteFile(filename, []byte("test content"), 0644)
	}

	// Create a compressed file (also recent)
	compressedDate := now.AddDate(0, 0, -4).Format("2006-01-02")
	os.WriteFile(filepath.Join(tmpDir, "gassigeher_"+compressedDate+".log.gz"), []byte("compressed"), 0644)

	cfg := &Config{
		LogDir:        tmpDir,
		MaxAgeDays:    30, // Keep files for 30 days
		CompressSizeMB: 10,
		ConsoleOutput: false,
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	defer logger.Close()

	// Wait for cleanup goroutine
	time.Sleep(50 * time.Millisecond)

	files, err := logger.GetLogFiles()
	if err != nil {
		t.Fatalf("GetLogFiles failed: %v", err)
	}

	// Should have at least 4 files (3 we created + compressed + today's log)
	if len(files) < 4 {
		t.Errorf("Expected at least 4 log files, got %d", len(files))
		for _, f := range files {
			t.Logf("  - %s", f.Name)
		}
	}

	// Check that compressed file is marked as compressed
	foundCompressed := false
	for _, f := range files {
		if strings.HasSuffix(f.Name, ".gz") {
			foundCompressed = true
			if !f.Compressed {
				t.Error("Expected file ending in .gz to be marked as compressed")
			}
		}
	}
	if !foundCompressed {
		t.Error("Expected to find compressed log file")
	}
}

// TestLogFileInfoFormatSize tests the FormatSize method
func TestLogFileInfoFormatSize(t *testing.T) {
	tests := []struct {
		size     int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			info := LogFileInfo{Size: tt.size}
			result := info.FormatSize()
			if result != tt.expected {
				t.Errorf("FormatSize(%d) = %s, want %s", tt.size, result, tt.expected)
			}
		})
	}
}

// TestLoggerCleanOldLogs tests the cleanOldLogs method
func TestLoggerCleanOldLogs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger-clean-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create old log files (older than 30 days)
	oldDate := time.Now().AddDate(0, 0, -35).Format("2006-01-02")
	oldFile := filepath.Join(tmpDir, "gassigeher_"+oldDate+".log")
	os.WriteFile(oldFile, []byte("old log"), 0644)

	// Create recent log file
	recentDate := time.Now().AddDate(0, 0, -5).Format("2006-01-02")
	recentFile := filepath.Join(tmpDir, "gassigeher_"+recentDate+".log")
	os.WriteFile(recentFile, []byte("recent log"), 0644)

	cfg := &Config{
		LogDir:        tmpDir,
		MaxAgeDays:    30,
		CompressSizeMB: 10,
		ConsoleOutput: false,
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	defer logger.Close()

	// Wait a bit for the goroutine to run
	time.Sleep(100 * time.Millisecond)

	// Old file should be deleted
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Error("Expected old log file to be deleted")
	}

	// Recent file should still exist
	if _, err := os.Stat(recentFile); os.IsNotExist(err) {
		t.Error("Expected recent log file to still exist")
	}
}

// HTTP Logger Tests

// TestNewResponseWriter tests creating a new ResponseWriter wrapper
func TestNewResponseWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := NewResponseWriter(rec)

	if rw.StatusCode() != http.StatusOK {
		t.Errorf("Expected default status 200, got %d", rw.StatusCode())
	}

	if rw.BytesWritten() != 0 {
		t.Errorf("Expected 0 bytes written initially, got %d", rw.BytesWritten())
	}
}

// TestResponseWriter_WriteHeader tests status code capture
func TestResponseWriter_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := NewResponseWriter(rec)

	rw.WriteHeader(http.StatusNotFound)

	if rw.StatusCode() != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rw.StatusCode())
	}
}

// TestResponseWriter_Write tests bytes written capture
func TestResponseWriter_Write(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := NewResponseWriter(rec)

	data := []byte("Hello, World!")
	n, err := rw.Write(data)

	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected %d bytes written, got %d", len(data), n)
	}
	if rw.BytesWritten() != int64(len(data)) {
		t.Errorf("Expected BytesWritten %d, got %d", len(data), rw.BytesWritten())
	}
}

// TestHTTPLogEntry_Format tests log entry formatting
func TestHTTPLogEntry_Format(t *testing.T) {
	t.Run("basic format", func(t *testing.T) {
		entry := &HTTPLogEntry{
			Timestamp:  time.Date(2025, 12, 23, 10, 0, 0, 0, time.UTC),
			RequestID:  "abc123",
			Method:     "GET",
			Path:       "/api/dogs",
			StatusCode: 200,
			Duration:   45 * time.Millisecond,
			BytesIn:    0,
			BytesOut:   1024,
			ClientIP:   "192.168.1.1",
			UserAgent:  "Mozilla/5.0",
		}

		formatted := entry.Format()
		if !strings.Contains(formatted, "abc123") {
			t.Error("Expected request ID in formatted output")
		}
		if !strings.Contains(formatted, "GET /api/dogs") {
			t.Error("Expected method and path in formatted output")
		}
		if !strings.Contains(formatted, "192.168.1.1") {
			t.Error("Expected client IP in formatted output")
		}
	})

	t.Run("with query params", func(t *testing.T) {
		entry := &HTTPLogEntry{
			Timestamp:  time.Now(),
			RequestID:  "def456",
			Method:     "GET",
			Path:       "/api/dogs",
			Query:      "page=1&limit=10",
			StatusCode: 200,
			Duration:   10 * time.Millisecond,
		}

		formatted := entry.Format()
		if !strings.Contains(formatted, "page=1&limit=10") {
			t.Error("Expected query params in formatted output")
		}
	})

	t.Run("redacts token in query", func(t *testing.T) {
		entry := &HTTPLogEntry{
			Timestamp:  time.Now(),
			RequestID:  "ghi789",
			Method:     "GET",
			Path:       "/api/verify",
			Query:      "token=secret123",
			StatusCode: 200,
			Duration:   5 * time.Millisecond,
		}

		formatted := entry.Format()
		if strings.Contains(formatted, "secret123") {
			t.Error("Token should be redacted")
		}
		if !strings.Contains(formatted, "REDACTED") {
			t.Error("Expected REDACTED placeholder for token")
		}
	})

	t.Run("with user ID", func(t *testing.T) {
		entry := &HTTPLogEntry{
			Timestamp:  time.Now(),
			RequestID:  "jkl012",
			Method:     "GET",
			Path:       "/api/me",
			StatusCode: 200,
			Duration:   5 * time.Millisecond,
			UserID:     42,
		}

		formatted := entry.Format()
		if !strings.Contains(formatted, "user:42") {
			t.Error("Expected user ID in formatted output")
		}
	})

	t.Run("with error", func(t *testing.T) {
		entry := &HTTPLogEntry{
			Timestamp:  time.Now(),
			RequestID:  "mno345",
			Method:     "POST",
			Path:       "/api/login",
			StatusCode: 401,
			Duration:   10 * time.Millisecond,
			Error:      "invalid credentials",
		}

		formatted := entry.Format()
		if !strings.Contains(formatted, "error:invalid credentials") {
			t.Error("Expected error in formatted output")
		}
	})

	t.Run("truncates long user agent", func(t *testing.T) {
		entry := &HTTPLogEntry{
			Timestamp:  time.Now(),
			RequestID:  "pqr678",
			Method:     "GET",
			Path:       "/api/health",
			StatusCode: 200,
			Duration:   1 * time.Millisecond,
			UserAgent:  strings.Repeat("A", 100),
		}

		formatted := entry.Format()
		if !strings.Contains(formatted, "...") {
			t.Error("Expected truncated user agent")
		}
	})
}

// TestHTTPLogEntry_FormatJSON tests JSON log entry formatting
func TestHTTPLogEntry_FormatJSON(t *testing.T) {
	entry := &HTTPLogEntry{
		Timestamp:  time.Date(2025, 12, 23, 10, 0, 0, 0, time.UTC),
		RequestID:  "abc123",
		Method:     "GET",
		Path:       "/api/dogs",
		StatusCode: 200,
		Duration:   45 * time.Millisecond,
		BytesIn:    100,
		BytesOut:   1024,
		ClientIP:   "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
		UserID:     42,
	}

	formatted := entry.FormatJSON()

	if !strings.Contains(formatted, `"request_id":"abc123"`) {
		t.Error("Expected request_id in JSON output")
	}
	if !strings.Contains(formatted, `"status":200`) {
		t.Error("Expected status in JSON output")
	}
	if !strings.Contains(formatted, `"user_id":42`) {
		t.Error("Expected user_id in JSON output")
	}
}

// TestGenerateRequestID tests request ID generation
func TestGenerateRequestID(t *testing.T) {
	id1 := GenerateRequestID()
	id2 := GenerateRequestID()

	if len(id1) != 16 {
		t.Errorf("Expected 16 character ID, got %d", len(id1))
	}
	if id1 == id2 {
		t.Error("Expected unique request IDs")
	}
}

// TestGetClientIP tests client IP extraction
func TestGetClientIP(t *testing.T) {
	t.Run("from X-Forwarded-For", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("X-Forwarded-For", "10.0.0.1, 192.168.1.1")

		ip := GetClientIP(req)
		if ip != "10.0.0.1" {
			t.Errorf("Expected IP 10.0.0.1, got %s", ip)
		}
	})

	t.Run("from X-Real-IP", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("X-Real-IP", "10.0.0.2")

		ip := GetClientIP(req)
		if ip != "10.0.0.2" {
			t.Errorf("Expected IP 10.0.0.2, got %s", ip)
		}
	})

	t.Run("from RemoteAddr", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.RemoteAddr = "192.168.1.100:12345"

		ip := GetClientIP(req)
		if ip != "192.168.1.100" {
			t.Errorf("Expected IP 192.168.1.100, got %s", ip)
		}
	})

	t.Run("prefers X-Forwarded-For over X-Real-IP", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("X-Forwarded-For", "10.0.0.1")
		req.Header.Set("X-Real-IP", "10.0.0.2")

		ip := GetClientIP(req)
		if ip != "10.0.0.1" {
			t.Errorf("Expected X-Forwarded-For IP 10.0.0.1, got %s", ip)
		}
	})
}

// TestStatusText tests colored status text
func TestStatusText(t *testing.T) {
	tests := []struct {
		code     int
		contains string
	}{
		{200, "32m"}, // Green
		{201, "32m"}, // Green
		{301, "36m"}, // Cyan
		{302, "36m"}, // Cyan
		{400, "33m"}, // Yellow
		{404, "33m"}, // Yellow
		{500, "31m"}, // Red
		{503, "31m"}, // Red
		{100, "100"}, // No color
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.code), func(t *testing.T) {
			result := StatusText(tt.code)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("StatusText(%d) = %q, expected to contain %q", tt.code, result, tt.contains)
			}
		})
	}
}

// TestFormatDuration tests duration formatting
func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{500 * time.Microsecond, "500µs"},
		{50 * time.Millisecond, "50ms"},
		{2 * time.Second, "2.00s"},
		{100 * time.Nanosecond, "0µs"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			entry := &HTTPLogEntry{Duration: tt.duration}
			formatted := entry.Format()
			if !strings.Contains(formatted, tt.expected) {
				t.Errorf("Expected duration %s in output, got %s", tt.expected, formatted)
			}
		})
	}
}

// TestFormatBytes tests byte formatting
func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0B"},
		{512, "512B"},
		{1024, "1.0KB"},
		{1048576, "1.0MB"},
		{1073741824, "1.0GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			entry := &HTTPLogEntry{BytesOut: tt.bytes}
			formatted := entry.Format()
			if !strings.Contains(formatted, tt.expected) {
				t.Errorf("Expected %s in output for %d bytes", tt.expected, tt.bytes)
			}
		})
	}
}
