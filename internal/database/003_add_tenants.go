package database

func init() {
	RegisterMigration(&Migration{
		ID:          "003_add_tenants",
		Description: "Create tenants and tenant_settings tables for multi-tenancy",
		Up: map[string]string{
			"sqlite": `
-- Tenants table
CREATE TABLE IF NOT EXISTS tenants (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  slug TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  status TEXT DEFAULT 'active' CHECK(status IN ('active', 'suspended', 'deleted')),

  -- Contact Info
  contact_email TEXT NOT NULL,
  contact_phone TEXT,
  address TEXT,
  city TEXT,
  postal_code TEXT,
  federal_state TEXT DEFAULT 'BW',

  -- Timestamps
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  suspended_at TIMESTAMP,
  suspended_reason TEXT,
  deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tenants_slug ON tenants(slug);
CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants(status);

-- Tenant settings table
CREATE TABLE IF NOT EXISTS tenant_settings (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id INTEGER NOT NULL UNIQUE,

  -- Branding (10 presets + custom)
  theme_preset TEXT DEFAULT 'classic',
  color_primary TEXT,
  color_secondary TEXT,
  color_accent TEXT,
  color_background TEXT,
  color_text TEXT,

  -- Logo
  logo_url TEXT,
  favicon_url TEXT,

  -- Content
  welcome_message TEXT,
  footer_text TEXT,

  -- External Links
  website_url TEXT,
  donation_url TEXT,

  -- Timestamps
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tenant_settings_tenant ON tenant_settings(tenant_id);
`,
			"mysql": `
-- Tenants table
CREATE TABLE IF NOT EXISTS tenants (
  id INT AUTO_INCREMENT PRIMARY KEY,
  slug VARCHAR(100) NOT NULL UNIQUE,
  name VARCHAR(255) NOT NULL,
  status ENUM('active', 'suspended', 'deleted') DEFAULT 'active',

  -- Contact Info
  contact_email VARCHAR(255) NOT NULL,
  contact_phone VARCHAR(50),
  address TEXT,
  city VARCHAR(100),
  postal_code VARCHAR(20),
  federal_state VARCHAR(50) DEFAULT 'BW',

  -- Timestamps
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  suspended_at TIMESTAMP NULL,
  suspended_reason TEXT,
  deleted_at TIMESTAMP NULL,

  INDEX idx_tenants_slug (slug),
  INDEX idx_tenants_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Tenant settings table
CREATE TABLE IF NOT EXISTS tenant_settings (
  id INT AUTO_INCREMENT PRIMARY KEY,
  tenant_id INT NOT NULL UNIQUE,

  -- Branding (10 presets + custom)
  theme_preset VARCHAR(50) DEFAULT 'classic',
  color_primary VARCHAR(7),
  color_secondary VARCHAR(7),
  color_accent VARCHAR(7),
  color_background VARCHAR(7),
  color_text VARCHAR(7),

  -- Logo
  logo_url VARCHAR(500),
  favicon_url VARCHAR(500),

  -- Content
  welcome_message TEXT,
  footer_text TEXT,

  -- External Links
  website_url VARCHAR(500),
  donation_url VARCHAR(500),

  -- Timestamps
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
  INDEX idx_tenant_settings_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`,
			"postgres": `
-- Tenants table
CREATE TABLE IF NOT EXISTS tenants (
  id SERIAL PRIMARY KEY,
  slug VARCHAR(100) NOT NULL UNIQUE,
  name VARCHAR(255) NOT NULL,
  status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'deleted')),

  -- Contact Info
  contact_email VARCHAR(255) NOT NULL,
  contact_phone VARCHAR(50),
  address TEXT,
  city VARCHAR(100),
  postal_code VARCHAR(20),
  federal_state VARCHAR(50) DEFAULT 'BW',

  -- Timestamps
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  suspended_at TIMESTAMP WITH TIME ZONE,
  suspended_reason TEXT,
  deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_tenants_slug ON tenants(slug);
CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants(status);

-- Tenant settings table
CREATE TABLE IF NOT EXISTS tenant_settings (
  id SERIAL PRIMARY KEY,
  tenant_id INTEGER NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,

  -- Branding (10 presets + custom)
  theme_preset VARCHAR(50) DEFAULT 'classic',
  color_primary VARCHAR(7),
  color_secondary VARCHAR(7),
  color_accent VARCHAR(7),
  color_background VARCHAR(7),
  color_text VARCHAR(7),

  -- Logo
  logo_url VARCHAR(500),
  favicon_url VARCHAR(500),

  -- Content
  welcome_message TEXT,
  footer_text TEXT,

  -- External Links
  website_url VARCHAR(500),
  donation_url VARCHAR(500),

  -- Timestamps
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tenant_settings_tenant ON tenant_settings(tenant_id);
`,
		},
	})
}
