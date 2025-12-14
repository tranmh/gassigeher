package database

func init() {
	RegisterMigration(&Migration{
		ID:          "023_add_must_change_password",
		Description: "Add must_change_password flag to users table for admin-created accounts",
		Up: map[string]string{
			"sqlite": `
ALTER TABLE users ADD COLUMN must_change_password INTEGER DEFAULT 0;
`,
			"mysql": `
ALTER TABLE users ADD COLUMN must_change_password TINYINT(1) DEFAULT 0;
`,
			"postgres": `
ALTER TABLE users ADD COLUMN must_change_password BOOLEAN DEFAULT FALSE;
`,
		},
	})
}
