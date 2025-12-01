package database

func init() {
	RegisterMigration(&Migration{
		ID:          "015_add_external_link",
		Description: "Add external_link column to dogs table for linking to external dog profiles",
		Up: map[string]string{
			"sqlite": `
-- Add external_link column to dogs table (nullable URL to external profile)
ALTER TABLE dogs ADD COLUMN external_link TEXT;
`,
			"mysql": `
-- Add external_link column to dogs table (nullable URL to external profile)
ALTER TABLE dogs ADD COLUMN external_link VARCHAR(500);
`,
			"postgres": `
-- Add external_link column to dogs table (nullable URL to external profile)
ALTER TABLE dogs ADD COLUMN external_link VARCHAR(500);
`,
		},
	})
}
