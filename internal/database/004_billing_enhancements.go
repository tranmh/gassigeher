package database

func init() {
	RegisterMigration(&Migration{
		ID:          "004_billing_enhancements",
		Description: "Add promo codes, invoices, and free months tracking for billing",
		Up: map[string]string{
			"sqlite": `
-- Promotional codes (separate from referral codes)
-- Used for marketing campaigns like LAUNCH50, BLACKFRIDAY, etc.
CREATE TABLE IF NOT EXISTS promo_codes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL UNIQUE,
  description TEXT,
  discount_type TEXT NOT NULL CHECK(discount_type IN ('percentage', 'fixed', 'free_months')),
  discount_value INTEGER NOT NULL,
  max_uses INTEGER,
  uses_count INTEGER DEFAULT 0,
  valid_for_plans TEXT,
  is_active INTEGER DEFAULT 1,
  stripe_coupon_id TEXT,
  expires_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_promo_codes_code ON promo_codes(code);
CREATE INDEX IF NOT EXISTS idx_promo_codes_active ON promo_codes(is_active);

-- Promo code usage tracking
CREATE TABLE IF NOT EXISTS promo_code_uses (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  promo_code_id INTEGER NOT NULL REFERENCES promo_codes(id) ON DELETE CASCADE,
  tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(promo_code_id, tenant_id)
);
CREATE INDEX IF NOT EXISTS idx_promo_code_uses_code ON promo_code_uses(promo_code_id);
CREATE INDEX IF NOT EXISTS idx_promo_code_uses_tenant ON promo_code_uses(tenant_id);

-- Tenant invoices for billing history
CREATE TABLE IF NOT EXISTS tenant_invoices (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  subscription_id INTEGER REFERENCES tenant_subscriptions(id) ON DELETE SET NULL,
  stripe_invoice_id TEXT UNIQUE,
  invoice_number TEXT NOT NULL,
  status TEXT DEFAULT 'paid' CHECK(status IN ('draft', 'open', 'paid', 'void', 'uncollectible')),
  amount_cents INTEGER NOT NULL,
  currency TEXT DEFAULT 'EUR',
  period_start DATE,
  period_end DATE,
  pdf_path TEXT,
  pdf_url TEXT,
  description TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  paid_at TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_tenant_invoices_tenant ON tenant_invoices(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_invoices_stripe ON tenant_invoices(stripe_invoice_id);
CREATE INDEX IF NOT EXISTS idx_tenant_invoices_status ON tenant_invoices(status);

-- Add free months tracking to tenant_subscriptions
ALTER TABLE tenant_subscriptions ADD COLUMN free_months_remaining INTEGER DEFAULT 0;
ALTER TABLE tenant_subscriptions ADD COLUMN free_months_granted INTEGER DEFAULT 0;
ALTER TABLE tenant_subscriptions ADD COLUMN free_months_source TEXT;
ALTER TABLE tenant_subscriptions ADD COLUMN applied_promo_code_id INTEGER REFERENCES promo_codes(id) ON DELETE SET NULL;
ALTER TABLE tenant_subscriptions ADD COLUMN applied_referral_code_id INTEGER REFERENCES referral_codes(id) ON DELETE SET NULL;
ALTER TABLE tenant_subscriptions ADD COLUMN trial_ends_at TIMESTAMP;
`,
			"postgres": `
-- Promotional codes (separate from referral codes)
CREATE TABLE IF NOT EXISTS promo_codes (
  id SERIAL PRIMARY KEY,
  code VARCHAR(50) NOT NULL UNIQUE,
  description TEXT,
  discount_type VARCHAR(20) NOT NULL CHECK(discount_type IN ('percentage', 'fixed', 'free_months')),
  discount_value INTEGER NOT NULL,
  max_uses INTEGER,
  uses_count INTEGER DEFAULT 0,
  valid_for_plans JSONB,
  is_active BOOLEAN DEFAULT TRUE,
  stripe_coupon_id VARCHAR(255),
  expires_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_promo_codes_code ON promo_codes(code);
CREATE INDEX IF NOT EXISTS idx_promo_codes_active ON promo_codes(is_active);

-- Promo code usage tracking
CREATE TABLE IF NOT EXISTS promo_code_uses (
  id SERIAL PRIMARY KEY,
  promo_code_id INTEGER NOT NULL REFERENCES promo_codes(id) ON DELETE CASCADE,
  tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(promo_code_id, tenant_id)
);
CREATE INDEX IF NOT EXISTS idx_promo_code_uses_code ON promo_code_uses(promo_code_id);
CREATE INDEX IF NOT EXISTS idx_promo_code_uses_tenant ON promo_code_uses(tenant_id);

-- Tenant invoices for billing history
CREATE TABLE IF NOT EXISTS tenant_invoices (
  id SERIAL PRIMARY KEY,
  tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  subscription_id INTEGER REFERENCES tenant_subscriptions(id) ON DELETE SET NULL,
  stripe_invoice_id VARCHAR(255) UNIQUE,
  invoice_number VARCHAR(100) NOT NULL,
  status VARCHAR(20) DEFAULT 'paid' CHECK(status IN ('draft', 'open', 'paid', 'void', 'uncollectible')),
  amount_cents INTEGER NOT NULL,
  currency VARCHAR(3) DEFAULT 'EUR',
  period_start DATE,
  period_end DATE,
  pdf_path VARCHAR(500),
  pdf_url VARCHAR(500),
  description TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  paid_at TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_tenant_invoices_tenant ON tenant_invoices(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_invoices_stripe ON tenant_invoices(stripe_invoice_id);
CREATE INDEX IF NOT EXISTS idx_tenant_invoices_status ON tenant_invoices(status);

-- Add free months tracking to tenant_subscriptions
ALTER TABLE tenant_subscriptions ADD COLUMN IF NOT EXISTS free_months_remaining INTEGER DEFAULT 0;
ALTER TABLE tenant_subscriptions ADD COLUMN IF NOT EXISTS free_months_granted INTEGER DEFAULT 0;
ALTER TABLE tenant_subscriptions ADD COLUMN IF NOT EXISTS free_months_source VARCHAR(50);
ALTER TABLE tenant_subscriptions ADD COLUMN IF NOT EXISTS applied_promo_code_id INTEGER REFERENCES promo_codes(id) ON DELETE SET NULL;
ALTER TABLE tenant_subscriptions ADD COLUMN IF NOT EXISTS applied_referral_code_id INTEGER REFERENCES referral_codes(id) ON DELETE SET NULL;
ALTER TABLE tenant_subscriptions ADD COLUMN IF NOT EXISTS trial_ends_at TIMESTAMP;
`,
		},
	})
}
