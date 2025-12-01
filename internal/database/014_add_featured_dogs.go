package database

func init() {
	RegisterMigration(&Migration{
		ID:          "014_add_featured_dogs",
		Description: "Add is_featured column to dogs table for homepage display",
		Up: map[string]string{
			"sqlite": `
-- Add is_featured column to dogs table
ALTER TABLE dogs ADD COLUMN is_featured INTEGER DEFAULT 0;

-- Create index for featured dogs query
CREATE INDEX IF NOT EXISTS idx_dogs_featured ON dogs(is_featured);
`,
			"mysql": `
-- Add is_featured column to dogs table
ALTER TABLE dogs ADD COLUMN is_featured TINYINT(1) DEFAULT 0;

-- Create index for featured dogs query
CREATE INDEX idx_dogs_featured ON dogs(is_featured);
`,
			"postgres": `
-- Add is_featured column to dogs table
ALTER TABLE dogs ADD COLUMN is_featured BOOLEAN DEFAULT FALSE;

-- Create index for featured dogs query
CREATE INDEX IF NOT EXISTS idx_dogs_featured ON dogs(is_featured);
`,
		},
	})
}
