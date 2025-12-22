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
-- MySQL doesn't support ADD COLUMN IF NOT EXISTS, use dynamic SQL
SET @col_exists = (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'tenants' AND column_name = 'is_demo');
SET @sql = IF(@col_exists = 0, 'ALTER TABLE tenants ADD COLUMN is_demo TINYINT(1) DEFAULT 0', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Index creation handled by migration system (uses dynamic SQL for idempotency)
SET @idx_exists = (SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'tenants' AND index_name = 'idx_tenants_is_demo');
SET @sql = IF(@idx_exists = 0, 'CREATE INDEX idx_tenants_is_demo ON tenants(is_demo)', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
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
