package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

// WeakJWTSecretPatterns contains patterns that indicate a weak/default JWT secret
var WeakJWTSecretPatterns = []string{
	"change-this",
	"dev-secret",
	"your-secret",
	"secret-key",
	"jwt-secret",
	"my-secret",
	"test-secret",
	"example",
	"placeholder",
	"changeme",
	"insecure",
	"default",
}

// CommonWeakSecrets contains exact matches for known weak secrets
var CommonWeakSecrets = []string{
	"secret",
	"password",
	"123456",
	"jwt",
	"token",
	"key",
}

// IsWeakJWTSecret checks if the given JWT secret is weak or uses a default pattern
func IsWeakJWTSecret(secret string) bool {
	if secret == "" {
		return true
	}

	// Check minimum length (256-bit = 32 bytes, but base64 encoded is ~43 chars)
	if len(secret) < 32 {
		return true
	}

	secretLower := strings.ToLower(secret)

	// Check for weak patterns
	for _, pattern := range WeakJWTSecretPatterns {
		if strings.Contains(secretLower, pattern) {
			return true
		}
	}

	// Check for exact weak secrets
	for _, weak := range CommonWeakSecrets {
		if secretLower == weak {
			return true
		}
	}

	return false
}

// GenerateSecureJWTSecret generates a cryptographically secure random JWT secret
// Returns a base64-encoded 256-bit (32 bytes) random string
func GenerateSecureJWTSecret() (string, error) {
	bytes := make([]byte, 32) // 256 bits
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// IsFreshInstall checks if this is a fresh database installation
// Returns true if schema_migrations table doesn't exist or is empty
func IsFreshInstall(db *sql.DB) bool {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if err != nil {
		// Table doesn't exist or query failed - fresh install
		return true
	}
	return count == 0
}

// UpdateEnvFile updates the JWT_SECRET in the .env file
// Creates a backup before modifying (.env.bak.TIMESTAMP)
func UpdateEnvFile(envPath string, newSecret string) error {
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

	// Replace JWT_SECRET line using regex
	// Matches: JWT_SECRET=anything (with or without quotes)
	re := regexp.MustCompile(`(?m)^JWT_SECRET=.*$`)
	newLine := fmt.Sprintf("JWT_SECRET=%s", newSecret)

	var newContent string
	if re.MatchString(string(content)) {
		// Replace existing JWT_SECRET
		newContent = re.ReplaceAllString(string(content), newLine)
	} else {
		// Append JWT_SECRET if not found
		newContent = string(content) + "\n" + newLine + "\n"
	}

	// Write updated content
	if err := os.WriteFile(envPath, []byte(newContent), 0600); err != nil {
		return fmt.Errorf("failed to write updated .env file: %w", err)
	}

	return nil
}

// EnforceStrongJWTSecret checks and enforces a strong JWT secret during database initialization
// This should be called BEFORE migrations run, passing the fresh install status
// Parameters:
//   - isFreshInstall: true if this is a fresh database (no schema_migrations table yet)
//   - currentSecret: the current JWT_SECRET value from config
//   - envPath: path to the .env file (from -env flag or default)
//
// Returns:
//   - newSecret: the (possibly new) JWT secret to use
//   - changed: true if the secret was changed
//   - error: any error that occurred
func EnforceStrongJWTSecret(isFreshInstall bool, currentSecret string, envPath string) (newSecret string, changed bool, err error) {
	// Only enforce on fresh install
	if !isFreshInstall {
		// Not a fresh install - check if secret is weak but only warn
		if IsWeakJWTSecret(currentSecret) {
			log.Println("WARNING: Current JWT_SECRET appears to be weak. Consider changing it for production.")
		}
		return currentSecret, false, nil
	}

	// Fresh install - check if secret is weak
	if !IsWeakJWTSecret(currentSecret) {
		// Secret is already strong
		log.Println("JWT_SECRET is strong - no changes needed")
		return currentSecret, false, nil
	}

	// Generate new secure secret
	newSecret, err = GenerateSecureJWTSecret()
	if err != nil {
		return "", false, fmt.Errorf("failed to generate secure JWT secret: %w", err)
	}

	// Update .env file if path provided
	if envPath != "" {
		if err := UpdateEnvFile(envPath, newSecret); err != nil {
			return "", false, fmt.Errorf("failed to update .env file: %w", err)
		}
	}

	// Log and print the change
	log.Println("=" + strings.Repeat("=", 70))
	log.Println("SECURITY: JWT_SECRET was weak/default and has been auto-generated")
	log.Println("=" + strings.Repeat("=", 70))
	log.Printf("New JWT_SECRET: %s", newSecret)
	log.Println("")
	log.Println("The .env file has been updated automatically.")
	log.Println("A backup of the old .env file was created.")
	log.Println("All users will need to re-login after this change.")
	log.Println("=" + strings.Repeat("=", 70))

	// Also print to stdout for visibility
	fmt.Println("")
	fmt.Println(strings.Repeat("=", 72))
	fmt.Println(" SECURITY: JWT_SECRET was weak/default and has been auto-generated")
	fmt.Println(strings.Repeat("=", 72))
	fmt.Printf(" New JWT_SECRET: %s\n", newSecret)
	fmt.Println("")
	fmt.Println(" The .env file has been updated automatically.")
	fmt.Println(" A backup of the old .env file was created.")
	fmt.Println(" All users will need to re-login after this change.")
	fmt.Println(strings.Repeat("=", 72))
	fmt.Println("")

	return newSecret, true, nil
}
