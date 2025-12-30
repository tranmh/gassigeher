package database

func init() {
	RegisterMigration(&Migration{
		ID:          "002_marketing_and_branding",
		Description: "Add marketing tables and branding columns to tenant_settings",
		Up: map[string]string{
			"sqlite": `
-- Add missing columns to tenant_settings (tagline and description)
-- SQLite doesn't support IF NOT EXISTS for ALTER TABLE ADD COLUMN
-- The migration runner handles "duplicate column name" errors gracefully
-- by marking the migration as applied and continuing.
-- All CREATE TABLE statements use IF NOT EXISTS for idempotency.

-- Add tagline column (will fail gracefully if exists)
ALTER TABLE tenant_settings ADD COLUMN tagline TEXT;

-- Add description column (will fail gracefully if exists)
ALTER TABLE tenant_settings ADD COLUMN description TEXT;

-- Marketing campaigns table (if not exists from 001)
CREATE TABLE IF NOT EXISTS marketing_campaigns (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  type TEXT NOT NULL CHECK(type IN ('fomo_countdown', 'referral', 'reference_page', 'custom')),
  name TEXT NOT NULL,
  description TEXT,
  config TEXT,
  is_active INTEGER DEFAULT 0,
  start_date TIMESTAMP,
  end_date TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Referral codes table (if not exists from 001)
CREATE TABLE IF NOT EXISTS referral_codes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL UNIQUE,
  referrer_tenant_id INTEGER REFERENCES tenants(id) ON DELETE SET NULL,
  referrer_email TEXT,
  discount_months_referrer INTEGER DEFAULT 3,
  discount_months_referee INTEGER DEFAULT 1,
  uses_count INTEGER DEFAULT 0,
  max_uses INTEGER,
  is_active INTEGER DEFAULT 1,
  expires_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_referral_codes_code ON referral_codes(code);
CREATE INDEX IF NOT EXISTS idx_referral_codes_referrer ON referral_codes(referrer_tenant_id);

-- Referral uses table (if not exists from 001)
CREATE TABLE IF NOT EXISTS referral_uses (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code_id INTEGER NOT NULL REFERENCES referral_codes(id) ON DELETE CASCADE,
  referee_tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_referral_uses_code ON referral_uses(code_id);
CREATE INDEX IF NOT EXISTS idx_referral_uses_referee ON referral_uses(referee_tenant_id);

-- Reference entries table (if not exists from 001)
CREATE TABLE IF NOT EXISTS reference_entries (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id INTEGER NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
  display_name TEXT NOT NULL,
  city TEXT,
  federal_state TEXT,
  website_url TEXT,
  testimonial TEXT,
  logo_url TEXT,
  is_approved INTEGER DEFAULT 0,
  display_order INTEGER DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_reference_entries_approved ON reference_entries(is_approved, display_order);
`,
			"mysql": `
-- Add missing columns to tenant_settings
ALTER TABLE tenant_settings ADD COLUMN IF NOT EXISTS tagline TEXT;
ALTER TABLE tenant_settings ADD COLUMN IF NOT EXISTS description TEXT;

-- Marketing campaigns table
CREATE TABLE IF NOT EXISTS marketing_campaigns (
  id INT AUTO_INCREMENT PRIMARY KEY,
  type VARCHAR(50) NOT NULL,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  config TEXT,
  is_active TINYINT(1) DEFAULT 0,
  start_date DATETIME,
  end_date DATETIME,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  CONSTRAINT chk_campaign_type CHECK (type IN ('fomo_countdown', 'referral', 'reference_page', 'custom'))
);

-- Referral codes table
CREATE TABLE IF NOT EXISTS referral_codes (
  id INT AUTO_INCREMENT PRIMARY KEY,
  code VARCHAR(50) NOT NULL UNIQUE,
  referrer_tenant_id INT,
  referrer_email VARCHAR(255),
  discount_months_referrer INT DEFAULT 3,
  discount_months_referee INT DEFAULT 1,
  uses_count INT DEFAULT 0,
  max_uses INT,
  is_active TINYINT(1) DEFAULT 1,
  expires_at DATETIME,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (referrer_tenant_id) REFERENCES tenants(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_referral_codes_code ON referral_codes(code);

-- Referral uses table
CREATE TABLE IF NOT EXISTS referral_uses (
  id INT AUTO_INCREMENT PRIMARY KEY,
  code_id INT NOT NULL,
  referee_tenant_id INT NOT NULL,
  applied_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (code_id) REFERENCES referral_codes(id) ON DELETE CASCADE,
  FOREIGN KEY (referee_tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- Reference entries table
CREATE TABLE IF NOT EXISTS reference_entries (
  id INT AUTO_INCREMENT PRIMARY KEY,
  tenant_id INT NOT NULL UNIQUE,
  display_name VARCHAR(255) NOT NULL,
  city VARCHAR(100),
  federal_state VARCHAR(50),
  website_url VARCHAR(500),
  testimonial TEXT,
  logo_url VARCHAR(500),
  is_approved TINYINT(1) DEFAULT 0,
  display_order INT DEFAULT 0,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);
`,
			"postgres": `
-- Add missing columns to tenant_settings
ALTER TABLE tenant_settings ADD COLUMN IF NOT EXISTS tagline TEXT;
ALTER TABLE tenant_settings ADD COLUMN IF NOT EXISTS description TEXT;

-- Marketing campaigns table
CREATE TABLE IF NOT EXISTS marketing_campaigns (
  id SERIAL PRIMARY KEY,
  type VARCHAR(50) NOT NULL CHECK (type IN ('fomo_countdown', 'referral', 'reference_page', 'custom')),
  name VARCHAR(255) NOT NULL,
  description TEXT,
  config TEXT,
  is_active BOOLEAN DEFAULT FALSE,
  start_date TIMESTAMP,
  end_date TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Referral codes table
CREATE TABLE IF NOT EXISTS referral_codes (
  id SERIAL PRIMARY KEY,
  code VARCHAR(50) NOT NULL UNIQUE,
  referrer_tenant_id INTEGER REFERENCES tenants(id) ON DELETE SET NULL,
  referrer_email VARCHAR(255),
  discount_months_referrer INTEGER DEFAULT 3,
  discount_months_referee INTEGER DEFAULT 1,
  uses_count INTEGER DEFAULT 0,
  max_uses INTEGER,
  is_active BOOLEAN DEFAULT TRUE,
  expires_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_referral_codes_code ON referral_codes(code);

-- Referral uses table
CREATE TABLE IF NOT EXISTS referral_uses (
  id SERIAL PRIMARY KEY,
  code_id INTEGER NOT NULL REFERENCES referral_codes(id) ON DELETE CASCADE,
  referee_tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Reference entries table
CREATE TABLE IF NOT EXISTS reference_entries (
  id SERIAL PRIMARY KEY,
  tenant_id INTEGER NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
  display_name VARCHAR(255) NOT NULL,
  city VARCHAR(100),
  federal_state VARCHAR(50),
  website_url VARCHAR(500),
  testimonial TEXT,
  logo_url VARCHAR(500),
  is_approved BOOLEAN DEFAULT FALSE,
  display_order INTEGER DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`,
		},
	})
}
