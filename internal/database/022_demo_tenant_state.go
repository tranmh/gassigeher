package database

func init() {
	RegisterMigration(&Migration{
		ID:          "022_demo_tenant_state",
		Description: "Create demo_tenant_state table for demo tenant management",
		Up: map[string]string{
			"sqlite": `
-- Demo tenant state table for tracking demo credentials and reset schedule
CREATE TABLE IF NOT EXISTS demo_tenant_state (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id INTEGER NOT NULL UNIQUE,
  admin_password TEXT NOT NULL,
  last_reset_at TIMESTAMP,
  next_reset_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_demo_tenant_state_tenant ON demo_tenant_state(tenant_id);
CREATE INDEX IF NOT EXISTS idx_demo_tenant_state_next_reset ON demo_tenant_state(next_reset_at);
`,
			"mysql": `
-- Demo tenant state table for tracking demo credentials and reset schedule
CREATE TABLE IF NOT EXISTS demo_tenant_state (
  id INT AUTO_INCREMENT PRIMARY KEY,
  tenant_id INT NOT NULL UNIQUE,
  admin_password VARCHAR(255) NOT NULL,
  last_reset_at TIMESTAMP NULL,
  next_reset_at TIMESTAMP NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_demo_tenant_state_tenant (tenant_id),
  INDEX idx_demo_tenant_state_next_reset (next_reset_at),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`,
			"postgres": `
-- Demo tenant state table for tracking demo credentials and reset schedule
CREATE TABLE IF NOT EXISTS demo_tenant_state (
  id SERIAL PRIMARY KEY,
  tenant_id INTEGER NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
  admin_password VARCHAR(255) NOT NULL,
  last_reset_at TIMESTAMP WITH TIME ZONE,
  next_reset_at TIMESTAMP WITH TIME ZONE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_demo_tenant_state_tenant ON demo_tenant_state(tenant_id);
CREATE INDEX IF NOT EXISTS idx_demo_tenant_state_next_reset ON demo_tenant_state(next_reset_at);
`,
		},
	})
}
