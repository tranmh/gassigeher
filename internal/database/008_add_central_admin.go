package database

func init() {
	RegisterMigration(&Migration{
		ID:          "008_add_central_admin",
		Description: "Add is_central_admin column to users table for platform-wide admin access",
		Up: map[string]string{
			"sqlite": `
				ALTER TABLE users ADD COLUMN is_central_admin INTEGER DEFAULT 0;
			`,
			"mysql": `
				ALTER TABLE users ADD COLUMN is_central_admin TINYINT(1) DEFAULT 0;
			`,
			"postgres": `
				ALTER TABLE users ADD COLUMN IF NOT EXISTS is_central_admin BOOLEAN DEFAULT FALSE;
			`,
		},
	})
}
