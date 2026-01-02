package database

func init() {
	RegisterMigration(&Migration{
		ID:          "006_fix_column_sizes",
		Description: "Fix column sizes for PostgreSQL compatibility (password_hash needs to be at least 60 chars for bcrypt)",
		Up: map[string]string{
			"sqlite": `
-- SQLite uses TEXT which has no size limit, no changes needed
SELECT 1;
`,
			"postgres": `
-- Fix password_hash column size to accommodate bcrypt hashes (60 chars) with margin
-- Some databases may have been created with VARCHAR(100) which is too small
ALTER TABLE users ALTER COLUMN password_hash TYPE VARCHAR(255);

-- Also ensure other columns have adequate sizes
ALTER TABLE users ALTER COLUMN first_name TYPE VARCHAR(255);
ALTER TABLE users ALTER COLUMN last_name TYPE VARCHAR(255);
`,
		},
	})
}
