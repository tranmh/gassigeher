package database

import (
	"crypto/rand"
)

// generateRegistrationPassword creates a random 8-character alphanumeric password
func generateRegistrationPassword() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	rand.Read(b)
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}

func init() {
	// Generate a unique password for this installation
	password := generateRegistrationPassword()

	RegisterMigration(&Migration{
		ID:          "018_insert_registration_password",
		Description: "Insert default registration password for user registration",
		Up: map[string]string{
			"sqlite": `
INSERT OR IGNORE INTO system_settings (key, value) VALUES
  ('registration_password', '` + password + `');
`,
			"mysql": "INSERT IGNORE INTO system_settings (`key`, value) VALUES\n" +
				"  ('registration_password', '" + password + "');",
			"postgres": `
INSERT INTO system_settings (key, value) VALUES
  ('registration_password', '` + password + `')
ON CONFLICT (key) DO NOTHING;
`,
		},
	})
}
