package database

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

// WeakMetricsPasswordPatterns contains patterns that indicate a weak/default metrics password
var WeakMetricsPasswordPatterns = []string{
	"change-this",
	"changeme",
	"password",
	"secret",
	"default",
	"metrics",
	"prometheus",
	"admin",
	"test",
}

// IsWeakMetricsPassword checks if the given metrics password is weak or uses a default pattern
func IsWeakMetricsPassword(password string) bool {
	if password == "" {
		return true
	}

	// Check minimum length (at least 32 characters for strong password)
	if len(password) < 32 {
		return true
	}

	passwordLower := strings.ToLower(password)

	// Check for weak patterns
	for _, pattern := range WeakMetricsPasswordPatterns {
		if strings.Contains(passwordLower, pattern) {
			return true
		}
	}

	return false
}

// GenerateSecureMetricsPassword generates a cryptographically secure random metrics password
// Returns a base64-encoded 256-bit (32 bytes) random string
func GenerateSecureMetricsPassword() (string, error) {
	bytes := make([]byte, 32) // 256 bits
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// UpdateEnvFileMetricsPassword updates the METRICS_PASSWORD in the .env file
// Creates a backup before modifying (.env.bak.TIMESTAMP)
func UpdateEnvFileMetricsPassword(envPath string, newPassword string) error {
	// Read current content
	content, err := os.ReadFile(envPath)
	if err != nil {
		return fmt.Errorf("failed to read .env file: %w", err)
	}

	// Create backup with timestamp
	backupPath := fmt.Sprintf("%s.bak.%s", envPath, time.Now().Format("20060102-150405"))
	if err := os.WriteFile(backupPath, content, 0600); err != nil {
		return fmt.Errorf("failed to create backup at %s: %w", backupPath, err)
	}
	log.Printf("Created .env backup: %s", backupPath)

	// Replace METRICS_PASSWORD line using regex
	re := regexp.MustCompile(`(?m)^METRICS_PASSWORD=.*$`)
	newLine := fmt.Sprintf("METRICS_PASSWORD=%s", newPassword)

	var newContent string
	if re.MatchString(string(content)) {
		// Replace existing METRICS_PASSWORD
		newContent = re.ReplaceAllString(string(content), newLine)
	} else {
		// Append METRICS_PASSWORD if not found
		newContent = string(content) + "\n" + newLine + "\n"
	}

	// Write updated content
	if err := os.WriteFile(envPath, []byte(newContent), 0600); err != nil {
		return fmt.Errorf("failed to write updated .env file: %w", err)
	}

	return nil
}

// EnforceStrongMetricsPassword checks and enforces a strong metrics password during startup
// This should be called BEFORE migrations run, passing the fresh install status
// Parameters:
//   - isFreshInstall: true if this is a fresh database (no schema_migrations table yet)
//   - currentPassword: the current METRICS_PASSWORD value from config
//   - envPath: path to the .env file (from -env flag or default)
//
// Returns:
//   - newPassword: the (possibly new) metrics password to use
//   - changed: true if the password was changed
//   - error: any error that occurred
func EnforceStrongMetricsPassword(isFreshInstall bool, currentPassword string, envPath string) (newPassword string, changed bool, err error) {
	// Only enforce on fresh install
	if !isFreshInstall {
		// Not a fresh install - check if password is weak but only warn
		if IsWeakMetricsPassword(currentPassword) {
			log.Println("WARNING: Current METRICS_PASSWORD appears to be weak. Consider changing it for production.")
		}
		return currentPassword, false, nil
	}

	// Fresh install - check if password is weak
	if !IsWeakMetricsPassword(currentPassword) {
		// Password is already strong
		log.Println("METRICS_PASSWORD is strong - no changes needed")
		return currentPassword, false, nil
	}

	// Generate new secure password
	newPassword, err = GenerateSecureMetricsPassword()
	if err != nil {
		return "", false, fmt.Errorf("failed to generate secure metrics password: %w", err)
	}

	// Update .env file if path provided
	if envPath != "" {
		if err := UpdateEnvFileMetricsPassword(envPath, newPassword); err != nil {
			return "", false, fmt.Errorf("failed to update .env file: %w", err)
		}
	}

	// Log and print the change
	log.Println("=" + strings.Repeat("=", 70))
	log.Println("SECURITY: METRICS_PASSWORD was weak/default and has been auto-generated")
	log.Println("=" + strings.Repeat("=", 70))
	log.Printf("New METRICS_PASSWORD: %s", newPassword)
	log.Println("")
	log.Println("The .env file has been updated automatically.")
	log.Println("A backup of the old .env file was created.")
	log.Println("Update your Prometheus scrape config with these credentials:")
	log.Println("  Username: prometheus (or METRICS_USERNAME env var)")
	log.Printf("  Password: %s", newPassword)
	log.Println("=" + strings.Repeat("=", 70))

	// Also print to stdout for visibility
	fmt.Println("")
	fmt.Println(strings.Repeat("=", 72))
	fmt.Println(" SECURITY: METRICS_PASSWORD was weak/default and has been auto-generated")
	fmt.Println(strings.Repeat("=", 72))
	fmt.Printf(" New METRICS_PASSWORD: %s\n", newPassword)
	fmt.Println("")
	fmt.Println(" The .env file has been updated automatically.")
	fmt.Println(" A backup of the old .env file was created.")
	fmt.Println(" Update your Prometheus scrape config with these credentials:")
	fmt.Println("   Username: prometheus (or METRICS_USERNAME env var)")
	fmt.Printf("   Password: %s\n", newPassword)
	fmt.Println(strings.Repeat("=", 72))
	fmt.Println("")

	return newPassword, true, nil
}
