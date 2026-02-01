package database

func init() {
	RegisterMigration(&Migration{
		ID:          "002_add_calendar_token",
		Description: "Add calendar_token column to users table for iCal feed authentication",
		Up: map[string]string{
			"sqlite": `
-- Add calendar_token column to users table
ALTER TABLE users ADD COLUMN calendar_token TEXT;

-- Create unique index on calendar_token for fast lookups
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_calendar_token ON users(calendar_token) WHERE calendar_token IS NOT NULL;
`,
			"postgres": `
-- Add calendar_token column to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS calendar_token VARCHAR(64);

-- Create unique index on calendar_token for fast lookups
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_calendar_token ON users(calendar_token) WHERE calendar_token IS NOT NULL;
`,
		},
	})
}
