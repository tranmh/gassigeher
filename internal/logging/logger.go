package logging

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Logger provides structured logging with daily rotation and compression
type Logger struct {
	mu            sync.Mutex
	logDir        string
	currentFile   *os.File
	currentDate   string
	maxAgeDays    int
	compressSize  int64 // bytes, compress if file exceeds this size
	consoleOutput bool
}

// Config holds logger configuration
type Config struct {
	LogDir        string // Directory for log files
	MaxAgeDays    int    // Days to keep logs (default: 30)
	CompressSizeMB int   // Compress files larger than this (default: 10MB)
	ConsoleOutput bool   // Also output to console (default: true)
}

// DefaultConfig returns default logger configuration
func DefaultConfig() *Config {
	return &Config{
		LogDir:        "./logs",
		MaxAgeDays:    30,
		CompressSizeMB: 10,
		ConsoleOutput: true,
	}
}

// NewLogger creates a new logger with rotation support
func NewLogger(cfg *Config) (*Logger, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Create log directory if not exists
	if err := os.MkdirAll(cfg.LogDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	l := &Logger{
		logDir:        cfg.LogDir,
		maxAgeDays:    cfg.MaxAgeDays,
		compressSize:  int64(cfg.CompressSizeMB) * 1024 * 1024,
		consoleOutput: cfg.ConsoleOutput,
	}

	// Open initial log file
	if err := l.rotate(); err != nil {
		return nil, err
	}

	// Set as default logger output
	log.SetOutput(l)
	log.SetFlags(0) // We handle timestamp ourselves

	// Clean up old logs on startup
	go l.cleanOldLogs()

	return l, nil
}

// Write implements io.Writer interface
func (l *Logger) Write(p []byte) (n int, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if we need to rotate (new day)
	today := time.Now().Format("2006-01-02")
	if today != l.currentDate {
		if err := l.rotateUnlocked(); err != nil {
			// Log to stderr if rotation fails
			fmt.Fprintf(os.Stderr, "Log rotation failed: %v\n", err)
		}
	}

	// Write to file
	if l.currentFile != nil {
		n, err = l.currentFile.Write(p)
	}

	// Also write to console if enabled
	if l.consoleOutput {
		os.Stdout.Write(p)
	}

	return n, err
}

// rotate opens a new log file for today
func (l *Logger) rotate() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.rotateUnlocked()
}

func (l *Logger) rotateUnlocked() error {
	// Close current file
	if l.currentFile != nil {
		oldFile := l.currentFile
		oldDate := l.currentDate
		l.currentFile = nil

		oldFile.Close()

		// Compress old file if needed (in background)
		go l.compressIfNeeded(oldDate)
	}

	// Open new file for today
	today := time.Now().Format("2006-01-02")
	filename := filepath.Join(l.logDir, fmt.Sprintf("gassigeher_%s.log", today))

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	l.currentFile = f
	l.currentDate = today

	return nil
}

// compressIfNeeded compresses a log file if it exceeds the size limit
func (l *Logger) compressIfNeeded(date string) {
	filename := filepath.Join(l.logDir, fmt.Sprintf("gassigeher_%s.log", date))

	info, err := os.Stat(filename)
	if err != nil {
		return // File doesn't exist or can't be accessed
	}

	if info.Size() < l.compressSize {
		return // File is small enough
	}

	// Compress the file
	gzFilename := filename + ".gz"

	// Open source file
	src, err := os.Open(filename)
	if err != nil {
		log.Printf("Failed to open log file for compression: %v", err)
		return
	}
	defer src.Close()

	// Create destination gzip file
	dst, err := os.Create(gzFilename)
	if err != nil {
		log.Printf("Failed to create compressed log file: %v", err)
		return
	}
	defer dst.Close()

	// Create gzip writer
	gz := gzip.NewWriter(dst)
	defer gz.Close()

	// Copy data
	if _, err := io.Copy(gz, src); err != nil {
		log.Printf("Failed to compress log file: %v", err)
		os.Remove(gzFilename) // Clean up partial file
		return
	}

	// Close gzip writer to flush
	gz.Close()
	dst.Close()
	src.Close()

	// Remove original file
	os.Remove(filename)
}

// cleanOldLogs removes log files older than maxAgeDays
func (l *Logger) cleanOldLogs() {
	cutoff := time.Now().AddDate(0, 0, -l.maxAgeDays)

	files, err := filepath.Glob(filepath.Join(l.logDir, "gassigeher_*.log*"))
	if err != nil {
		return
	}

	for _, file := range files {
		// Extract date from filename
		base := filepath.Base(file)
		base = strings.TrimPrefix(base, "gassigeher_")
		base = strings.TrimSuffix(base, ".log")
		base = strings.TrimSuffix(base, ".log.gz")
		base = strings.TrimSuffix(base, ".gz")

		fileDate, err := time.Parse("2006-01-02", base)
		if err != nil {
			continue // Skip files with unexpected names
		}

		if fileDate.Before(cutoff) {
			os.Remove(file)
		}
	}
}

// Close closes the logger
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.currentFile != nil {
		return l.currentFile.Close()
	}
	return nil
}

// GetLogFiles returns list of log files sorted by date (newest first)
func (l *Logger) GetLogFiles() ([]LogFileInfo, error) {
	files, err := filepath.Glob(filepath.Join(l.logDir, "gassigeher_*.log*"))
	if err != nil {
		return nil, err
	}

	var infos []LogFileInfo
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		infos = append(infos, LogFileInfo{
			Name:       filepath.Base(file),
			Path:       file,
			Size:       info.Size(),
			ModTime:    info.ModTime(),
			Compressed: strings.HasSuffix(file, ".gz"),
		})
	}

	// Sort by modification time (newest first)
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].ModTime.After(infos[j].ModTime)
	})

	return infos, nil
}

// LogFileInfo contains information about a log file
type LogFileInfo struct {
	Name       string
	Path       string
	Size       int64
	ModTime    time.Time
	Compressed bool
}

// FormatSize returns human-readable file size
func (i LogFileInfo) FormatSize() string {
	const unit = 1024
	if i.Size < unit {
		return fmt.Sprintf("%d B", i.Size)
	}
	div, exp := int64(unit), 0
	for n := i.Size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(i.Size)/float64(div), "KMGTPE"[exp])
}
