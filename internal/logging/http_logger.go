package logging

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ResponseWriter wraps http.ResponseWriter to capture status code and size
type ResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

// NewResponseWriter creates a new ResponseWriter wrapper
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // Default to 200
	}
}

// WriteHeader captures the status code
func (rw *ResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures bytes written
func (rw *ResponseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += int64(n)
	return n, err
}

// StatusCode returns the captured status code
func (rw *ResponseWriter) StatusCode() int {
	return rw.statusCode
}

// BytesWritten returns the number of bytes written
func (rw *ResponseWriter) BytesWritten() int64 {
	return rw.bytesWritten
}

// HTTPLogEntry represents a single HTTP request log entry
type HTTPLogEntry struct {
	Timestamp    time.Time
	RequestID    string
	Method       string
	Path         string
	Query        string
	StatusCode   int
	Duration     time.Duration
	BytesIn      int64
	BytesOut     int64
	ClientIP     string
	UserAgent    string
	Referer      string
	UserID       int    // 0 if not authenticated
	Error        string // Error message if any
}

// Format returns a formatted log line
func (e *HTTPLogEntry) Format() string {
	// Format: [TIMESTAMP] REQUEST_ID | IP | METHOD PATH | STATUS | DURATION | IN/OUT | USER_AGENT
	// Example: [2025-11-29T19:42:21+01:00] abc123 | 192.168.1.1 | GET /api/dogs | 200 | 45ms | 0B/1.2KB | Mozilla/5.0...

	path := e.Path
	if e.Query != "" {
		// Sanitize sensitive query params
		if strings.Contains(e.Query, "token") {
			path += "?token=REDACTED"
		} else {
			path += "?" + e.Query
		}
	}

	userInfo := ""
	if e.UserID > 0 {
		userInfo = fmt.Sprintf(" | user:%d", e.UserID)
	}

	errorInfo := ""
	if e.Error != "" {
		errorInfo = fmt.Sprintf(" | error:%s", e.Error)
	}

	// Truncate user agent for readability
	userAgent := e.UserAgent
	if len(userAgent) > 50 {
		userAgent = userAgent[:50] + "..."
	}

	return fmt.Sprintf("[%s] %s | %s | %s %s | %d | %s | %s/%s%s%s | %s",
		e.Timestamp.Format(time.RFC3339),
		e.RequestID,
		e.ClientIP,
		e.Method,
		path,
		e.StatusCode,
		formatDuration(e.Duration),
		formatBytes(e.BytesIn),
		formatBytes(e.BytesOut),
		userInfo,
		errorInfo,
		userAgent,
	)
}

// FormatJSON returns a JSON formatted log line (for structured logging)
func (e *HTTPLogEntry) FormatJSON() string {
	path := e.Path
	if e.Query != "" && !strings.Contains(e.Query, "token") {
		path += "?" + e.Query
	}

	return fmt.Sprintf(`{"time":"%s","request_id":"%s","client_ip":"%s","method":"%s","path":"%s","status":%d,"duration_ms":%d,"bytes_in":%d,"bytes_out":%d,"user_id":%d,"user_agent":"%s","error":"%s"}`,
		e.Timestamp.Format(time.RFC3339),
		e.RequestID,
		e.ClientIP,
		e.Method,
		escapeJSON(path),
		e.StatusCode,
		e.Duration.Milliseconds(),
		e.BytesIn,
		e.BytesOut,
		e.UserID,
		escapeJSON(e.UserAgent),
		escapeJSON(e.Error),
	)
}

// generateRequestID generates a unique request ID
func GenerateRequestID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// GetClientIP extracts the real client IP from request
func GetClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (from reverse proxies)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP (original client)
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	// Remove port if present
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// formatDuration formats duration in human-readable form
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// formatBytes formats bytes in human-readable form
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// escapeJSON escapes a string for JSON output
func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}

// StatusText returns a colored status text for terminal output
func StatusText(code int) string {
	switch {
	case code >= 500:
		return fmt.Sprintf("\033[31m%d\033[0m", code) // Red
	case code >= 400:
		return fmt.Sprintf("\033[33m%d\033[0m", code) // Yellow
	case code >= 300:
		return fmt.Sprintf("\033[36m%d\033[0m", code) // Cyan
	case code >= 200:
		return fmt.Sprintf("\033[32m%d\033[0m", code) // Green
	default:
		return fmt.Sprintf("%d", code)
	}
}
