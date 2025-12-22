package database

func init() {
	RegisterMigration(&Migration{
		ID:          "023_add_tenant_is_demo",
		Description: "Add is_demo column to tenants table",
		Up: map[string]string{
			"sqlite": `
-- Add is_demo flag to tenants table
ALTER TABLE tenants ADD COLUMN is_demo INTEGER DEFAULT 0;

-- Create index for demo tenant lookup
CREATE INDEX IF NOT EXISTS idx_tenants_is_demo ON tenants(is_demo);
`,
			"mysql": `
-- Add is_demo flag to tenants table
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS is_demo TINYINT(1) DEFAULT 0;

-- Create index for demo tenant lookup (drop first for idempotency on MySQL < 8.0)
DROP INDEX IF EXISTS idx_tenants_is_demo ON tenants;
CREATE INDEX idx_tenants_is_demo ON tenants(is_demo);
`,
			"postgres": `
-- Add is_demo flag to tenants table
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS is_demo BOOLEAN DEFAULT FALSE;

-- Create index for demo tenant lookup
CREATE INDEX IF NOT EXISTS idx_tenants_is_demo ON tenants(is_demo);
`,
		},
	})
}
