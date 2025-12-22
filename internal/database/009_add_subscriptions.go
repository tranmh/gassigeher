package database

func init() {
	RegisterMigration(&Migration{
		ID:          "009_add_subscriptions",
		Description: "Create pricing_plans and tenant_subscriptions tables for 2-tier pricing",
		Up: map[string]string{
			"sqlite": `
-- Pricing plans table (static data: Free, Pro)
CREATE TABLE IF NOT EXISTS pricing_plans (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  max_dogs INTEGER NOT NULL,
  price_monthly INTEGER NOT NULL DEFAULT 0,
  price_yearly INTEGER NOT NULL DEFAULT 0,
  is_active INTEGER DEFAULT 1,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_pricing_plans_slug ON pricing_plans(slug);
CREATE INDEX IF NOT EXISTS idx_pricing_plans_active ON pricing_plans(is_active);

-- Seed default pricing plans
INSERT OR IGNORE INTO pricing_plans (id, name, slug, max_dogs, price_monthly, price_yearly, is_active)
VALUES
  (1, 'Free', 'free', 10, 0, 0, 1),
  (2, 'Pro', 'pro', -1, 2900, 29000, 1);

-- Tenant subscriptions table
CREATE TABLE IF NOT EXISTS tenant_subscriptions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id INTEGER NOT NULL,
  plan_id INTEGER NOT NULL DEFAULT 1,
  status TEXT DEFAULT 'active' CHECK(status IN ('active', 'cancelled', 'past_due', 'trialing')),
  billing_cycle TEXT CHECK(billing_cycle IN ('monthly', 'yearly')),
  current_period_start TIMESTAMP,
  current_period_end TIMESTAMP,
  stripe_customer_id TEXT,
  stripe_subscription_id TEXT,
  cancelled_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
  FOREIGN KEY (plan_id) REFERENCES pricing_plans(id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tenant_subscriptions_tenant ON tenant_subscriptions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_status ON tenant_subscriptions(status);
CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_stripe_customer ON tenant_subscriptions(stripe_customer_id);
CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_stripe_sub ON tenant_subscriptions(stripe_subscription_id);

-- Add max_dogs column to tenants table for quick access (denormalized)
ALTER TABLE tenants ADD COLUMN max_dogs INTEGER DEFAULT 10;
ALTER TABLE tenants ADD COLUMN subscription_status TEXT DEFAULT 'free';

-- Create default free subscriptions for existing tenants
INSERT OR IGNORE INTO tenant_subscriptions (tenant_id, plan_id, status)
SELECT id, 1, 'active' FROM tenants WHERE id NOT IN (SELECT tenant_id FROM tenant_subscriptions);
`,
			"mysql": `
-- Pricing plans table (static data: Free, Pro)
CREATE TABLE IF NOT EXISTS pricing_plans (
  id INT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(50) NOT NULL,
  slug VARCHAR(50) NOT NULL UNIQUE,
  max_dogs INT NOT NULL,
  price_monthly INT NOT NULL DEFAULT 0,
  price_yearly INT NOT NULL DEFAULT 0,
  is_active TINYINT(1) DEFAULT 1,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_pricing_plans_slug (slug),
  INDEX idx_pricing_plans_active (is_active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Seed default pricing plans
INSERT IGNORE INTO pricing_plans (id, name, slug, max_dogs, price_monthly, price_yearly, is_active)
VALUES
  (1, 'Free', 'free', 10, 0, 0, 1),
  (2, 'Pro', 'pro', -1, 2900, 29000, 1);

-- Tenant subscriptions table
CREATE TABLE IF NOT EXISTS tenant_subscriptions (
  id INT AUTO_INCREMENT PRIMARY KEY,
  tenant_id INT NOT NULL,
  plan_id INT NOT NULL DEFAULT 1,
  status ENUM('active', 'cancelled', 'past_due', 'trialing') DEFAULT 'active',
  billing_cycle ENUM('monthly', 'yearly'),
  current_period_start TIMESTAMP NULL,
  current_period_end TIMESTAMP NULL,
  stripe_customer_id VARCHAR(100),
  stripe_subscription_id VARCHAR(100),
  cancelled_at TIMESTAMP NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE INDEX idx_tenant_subscriptions_tenant (tenant_id),
  INDEX idx_tenant_subscriptions_status (status),
  INDEX idx_tenant_subscriptions_stripe_customer (stripe_customer_id),
  INDEX idx_tenant_subscriptions_stripe_sub (stripe_subscription_id),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
  FOREIGN KEY (plan_id) REFERENCES pricing_plans(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Add max_dogs column to tenants table for quick access (denormalized)
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS max_dogs INT DEFAULT 10;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS subscription_status VARCHAR(20) DEFAULT 'free';

-- Create default free subscriptions for existing tenants
INSERT IGNORE INTO tenant_subscriptions (tenant_id, plan_id, status)
SELECT id, 1, 'active' FROM tenants WHERE id NOT IN (SELECT tenant_id FROM tenant_subscriptions);
`,
			"postgres": `
-- Pricing plans table (static data: Free, Pro)
CREATE TABLE IF NOT EXISTS pricing_plans (
  id SERIAL PRIMARY KEY,
  name VARCHAR(50) NOT NULL,
  slug VARCHAR(50) NOT NULL UNIQUE,
  max_dogs INTEGER NOT NULL,
  price_monthly INTEGER NOT NULL DEFAULT 0,
  price_yearly INTEGER NOT NULL DEFAULT 0,
  is_active BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_pricing_plans_slug ON pricing_plans(slug);
CREATE INDEX IF NOT EXISTS idx_pricing_plans_active ON pricing_plans(is_active);

-- Seed default pricing plans
INSERT INTO pricing_plans (id, name, slug, max_dogs, price_monthly, price_yearly, is_active)
VALUES
  (1, 'Free', 'free', 10, 0, 0, TRUE),
  (2, 'Pro', 'pro', -1, 2900, 29000, TRUE)
ON CONFLICT (slug) DO NOTHING;

-- Reset sequence to avoid conflicts
SELECT setval('pricing_plans_id_seq', COALESCE((SELECT MAX(id) FROM pricing_plans), 0) + 1, false);

-- Tenant subscriptions table
CREATE TABLE IF NOT EXISTS tenant_subscriptions (
  id SERIAL PRIMARY KEY,
  tenant_id INTEGER NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
  plan_id INTEGER NOT NULL DEFAULT 1 REFERENCES pricing_plans(id),
  status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'cancelled', 'past_due', 'trialing')),
  billing_cycle VARCHAR(10) CHECK (billing_cycle IN ('monthly', 'yearly')),
  current_period_start TIMESTAMP WITH TIME ZONE,
  current_period_end TIMESTAMP WITH TIME ZONE,
  stripe_customer_id VARCHAR(100),
  stripe_subscription_id VARCHAR(100),
  cancelled_at TIMESTAMP WITH TIME ZONE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_status ON tenant_subscriptions(status);
CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_stripe_customer ON tenant_subscriptions(stripe_customer_id);
CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_stripe_sub ON tenant_subscriptions(stripe_subscription_id);

-- Add max_dogs column to tenants table for quick access (denormalized)
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS max_dogs INTEGER DEFAULT 10;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS subscription_status VARCHAR(20) DEFAULT 'free';

-- Create default free subscriptions for existing tenants
INSERT INTO tenant_subscriptions (tenant_id, plan_id, status)
SELECT id, 1, 'active' FROM tenants WHERE id NOT IN (SELECT tenant_id FROM tenant_subscriptions)
ON CONFLICT (tenant_id) DO NOTHING;
`,
		},
	})
}
