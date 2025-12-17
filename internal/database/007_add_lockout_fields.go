package database

func init() {
	RegisterMigration(&Migration{
		ID:          "007_add_lockout_fields",
		Description: "Add lockout fields to users table for brute force protection",
		Up: map[string]string{
			"sqlite": `
				ALTER TABLE users ADD COLUMN failed_login_attempts INTEGER DEFAULT 0;
				ALTER TABLE users ADD COLUMN locked_until TEXT;
				ALTER TABLE users ADD COLUMN last_failed_login TEXT;
			`,
			"mysql": `
				ALTER TABLE users
				ADD COLUMN failed_login_attempts INT DEFAULT 0,
				ADD COLUMN locked_until DATETIME,
				ADD COLUMN last_failed_login DATETIME;
			`,
			"postgres": `
				ALTER TABLE users ADD COLUMN IF NOT EXISTS failed_login_attempts INTEGER DEFAULT 0;
				ALTER TABLE users ADD COLUMN IF NOT EXISTS locked_until TIMESTAMP;
				ALTER TABLE users ADD COLUMN IF NOT EXISTS last_failed_login TIMESTAMP;
			`,
		},
	})
}
