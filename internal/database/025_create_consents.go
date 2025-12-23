package database

func init() {
	RegisterMigration(&Migration{
		ID:          "025_create_consents",
		Description: "Create consents table for GDPR consent tracking",
		Up: map[string]string{
			"sqlite": `
-- Create consents table for tracking user consent
CREATE TABLE IF NOT EXISTS consents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    tenant_id INTEGER NOT NULL,
    consent_type VARCHAR(50) NOT NULL,
    version VARCHAR(20) NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    accepted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_consents_user_id ON consents(user_id);
CREATE INDEX IF NOT EXISTS idx_consents_tenant_id ON consents(tenant_id);
CREATE INDEX IF NOT EXISTS idx_consents_type ON consents(consent_type);
CREATE INDEX IF NOT EXISTS idx_consents_user_type ON consents(user_id, consent_type);
`,
			"mysql": `
-- Create consents table for tracking user consent
CREATE TABLE IF NOT EXISTS consents (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    tenant_id INT NOT NULL,
    consent_type VARCHAR(50) NOT NULL,
    version VARCHAR(20) NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    accepted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_consents_user_id (user_id),
    INDEX idx_consents_tenant_id (tenant_id),
    INDEX idx_consents_type (consent_type),
    INDEX idx_consents_user_type (user_id, consent_type),
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`,
			"postgres": `
-- Create consents table for tracking user consent
CREATE TABLE IF NOT EXISTS consents (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    tenant_id INTEGER NOT NULL REFERENCES tenants(id),
    consent_type VARCHAR(50) NOT NULL,
    version VARCHAR(20) NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    accepted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_consents_user_id ON consents(user_id);
CREATE INDEX IF NOT EXISTS idx_consents_tenant_id ON consents(tenant_id);
CREATE INDEX IF NOT EXISTS idx_consents_type ON consents(consent_type);
CREATE INDEX IF NOT EXISTS idx_consents_user_type ON consents(user_id, consent_type);
`,
		},
	})
}
