package database

func init() {
	RegisterMigration(&Migration{
		ID:          "026_create_feature_flags",
		Description: "Create feature flags tables for gradual rollout",
		Up: map[string]string{
			"sqlite": `
-- Feature flags table (global flag definitions)
CREATE TABLE IF NOT EXISTS feature_flags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_global INTEGER DEFAULT 1,
    is_enabled INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tenant-specific feature flag overrides
CREATE TABLE IF NOT EXISTS tenant_feature_flags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id INTEGER NOT NULL,
    feature_flag_id INTEGER NOT NULL,
    is_enabled INTEGER DEFAULT 0,
    enabled_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    enabled_by INTEGER,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    FOREIGN KEY (feature_flag_id) REFERENCES feature_flags(id),
    FOREIGN KEY (enabled_by) REFERENCES users(id),
    UNIQUE(tenant_id, feature_flag_id)
);

-- Indexes for efficient lookups
CREATE INDEX IF NOT EXISTS idx_feature_flags_key ON feature_flags(key);
CREATE INDEX IF NOT EXISTS idx_tenant_feature_flags_tenant ON tenant_feature_flags(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_feature_flags_flag ON tenant_feature_flags(feature_flag_id);

-- Insert default feature flags
INSERT OR IGNORE INTO feature_flags (key, name, description, is_global, is_enabled) VALUES
    ('new_booking_ui', 'Neues Buchungs-UI', 'Aktiviert das neue, verbesserte Buchungsinterface', 1, 0),
    ('advanced_search', 'Erweiterte Suche', 'Aktiviert erweiterte Suchfunktionen fuer Hunde und Buchungen', 1, 0),
    ('bulk_operations', 'Massenoperationen', 'Ermoeglicht Admin-Massenaktionen (z.B. mehrere Buchungen absagen)', 1, 0),
    ('dark_mode', 'Dunkelmodus', 'Aktiviert den Dunkelmodus in der Benutzeroberflaeche', 1, 0),
    ('calendar_sync', 'Kalender-Synchronisation', 'Ermoeglicht Synchronisation mit externen Kalendern (Google, Outlook)', 0, 0),
    ('sms_notifications', 'SMS-Benachrichtigungen', 'Sendet Buchungsbestaetigung und Erinnerungen per SMS', 0, 0);
`,
			"mysql": `
-- Feature flags table (global flag definitions)
CREATE TABLE IF NOT EXISTS feature_flags (
    id INT AUTO_INCREMENT PRIMARY KEY,
    ` + "`key`" + ` VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_global TINYINT(1) DEFAULT 1,
    is_enabled TINYINT(1) DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_feature_flags_key (` + "`key`" + `)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Tenant-specific feature flag overrides
CREATE TABLE IF NOT EXISTS tenant_feature_flags (
    id INT AUTO_INCREMENT PRIMARY KEY,
    tenant_id INT NOT NULL,
    feature_flag_id INT NOT NULL,
    is_enabled TINYINT(1) DEFAULT 0,
    enabled_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    enabled_by INT,
    UNIQUE KEY unique_tenant_flag (tenant_id, feature_flag_id),
    INDEX idx_tenant_feature_flags_tenant (tenant_id),
    INDEX idx_tenant_feature_flags_flag (feature_flag_id),
    FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    FOREIGN KEY (feature_flag_id) REFERENCES feature_flags(id),
    FOREIGN KEY (enabled_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Insert default feature flags
INSERT IGNORE INTO feature_flags (` + "`key`" + `, name, description, is_global, is_enabled) VALUES
    ('new_booking_ui', 'Neues Buchungs-UI', 'Aktiviert das neue, verbesserte Buchungsinterface', 1, 0),
    ('advanced_search', 'Erweiterte Suche', 'Aktiviert erweiterte Suchfunktionen fuer Hunde und Buchungen', 1, 0),
    ('bulk_operations', 'Massenoperationen', 'Ermoeglicht Admin-Massenaktionen (z.B. mehrere Buchungen absagen)', 1, 0),
    ('dark_mode', 'Dunkelmodus', 'Aktiviert den Dunkelmodus in der Benutzeroberflaeche', 1, 0),
    ('calendar_sync', 'Kalender-Synchronisation', 'Ermoeglicht Synchronisation mit externen Kalendern (Google, Outlook)', 0, 0),
    ('sms_notifications', 'SMS-Benachrichtigungen', 'Sendet Buchungsbestaetigung und Erinnerungen per SMS', 0, 0);
`,
			"postgres": `
-- Feature flags table (global flag definitions)
CREATE TABLE IF NOT EXISTS feature_flags (
    id SERIAL PRIMARY KEY,
    key VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_global BOOLEAN DEFAULT TRUE,
    is_enabled BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tenant-specific feature flag overrides
CREATE TABLE IF NOT EXISTS tenant_feature_flags (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id),
    feature_flag_id INTEGER NOT NULL REFERENCES feature_flags(id),
    is_enabled BOOLEAN DEFAULT FALSE,
    enabled_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    enabled_by INTEGER REFERENCES users(id),
    UNIQUE(tenant_id, feature_flag_id)
);

-- Indexes for efficient lookups
CREATE INDEX IF NOT EXISTS idx_feature_flags_key ON feature_flags(key);
CREATE INDEX IF NOT EXISTS idx_tenant_feature_flags_tenant ON tenant_feature_flags(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_feature_flags_flag ON tenant_feature_flags(feature_flag_id);

-- Insert default feature flags
INSERT INTO feature_flags (key, name, description, is_global, is_enabled) VALUES
    ('new_booking_ui', 'Neues Buchungs-UI', 'Aktiviert das neue, verbesserte Buchungsinterface', TRUE, FALSE),
    ('advanced_search', 'Erweiterte Suche', 'Aktiviert erweiterte Suchfunktionen fuer Hunde und Buchungen', TRUE, FALSE),
    ('bulk_operations', 'Massenoperationen', 'Ermoeglicht Admin-Massenaktionen (z.B. mehrere Buchungen absagen)', TRUE, FALSE),
    ('dark_mode', 'Dunkelmodus', 'Aktiviert den Dunkelmodus in der Benutzeroberflaeche', TRUE, FALSE),
    ('calendar_sync', 'Kalender-Synchronisation', 'Ermoeglicht Synchronisation mit externen Kalendern (Google, Outlook)', FALSE, FALSE),
    ('sms_notifications', 'SMS-Benachrichtigungen', 'Sendet Buchungsbestaetigung und Erinnerungen per SMS', FALSE, FALSE)
ON CONFLICT (key) DO NOTHING;
`,
		},
	})
}
