package database

func init() {
	RegisterMigration(&Migration{
		ID:          "003_feature_flags",
		Description: "Add feature flags tables for platform-wide feature management",
		Up: map[string]string{
			"sqlite": `
-- Feature flags table for global feature management
CREATE TABLE IF NOT EXISTS feature_flags (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  key TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  description TEXT,
  is_global INTEGER DEFAULT 0,
  is_enabled INTEGER DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_feature_flags_key ON feature_flags(key);

-- Tenant-specific feature flag overrides
CREATE TABLE IF NOT EXISTS tenant_feature_flags (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  feature_flag_id INTEGER NOT NULL REFERENCES feature_flags(id) ON DELETE CASCADE,
  is_enabled INTEGER DEFAULT 0,
  enabled_at TIMESTAMP,
  enabled_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
  UNIQUE(tenant_id, feature_flag_id)
);
CREATE INDEX IF NOT EXISTS idx_tenant_feature_flags_tenant ON tenant_feature_flags(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_feature_flags_flag ON tenant_feature_flags(feature_flag_id);
`,
			"postgres": `
-- Feature flags table for global feature management
CREATE TABLE IF NOT EXISTS feature_flags (
  id SERIAL PRIMARY KEY,
  key VARCHAR(100) NOT NULL UNIQUE,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  is_global BOOLEAN DEFAULT FALSE,
  is_enabled BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_feature_flags_key ON feature_flags(key);

-- Tenant-specific feature flag overrides
CREATE TABLE IF NOT EXISTS tenant_feature_flags (
  id SERIAL PRIMARY KEY,
  tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  feature_flag_id INTEGER NOT NULL REFERENCES feature_flags(id) ON DELETE CASCADE,
  is_enabled BOOLEAN DEFAULT FALSE,
  enabled_at TIMESTAMP,
  enabled_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
  UNIQUE(tenant_id, feature_flag_id)
);
CREATE INDEX IF NOT EXISTS idx_tenant_feature_flags_tenant ON tenant_feature_flags(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_feature_flags_flag ON tenant_feature_flags(feature_flag_id);
`,
		},
	})
}
