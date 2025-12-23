package database

func init() {
	RegisterMigration(&Migration{
		ID:          "001_schema",
		Description: "Complete SaaS multi-tenant schema",
		Up: map[string]string{
			"sqlite": `
-- Tenants table (central)
CREATE TABLE IF NOT EXISTS tenants (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  slug TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  contact_email TEXT NOT NULL,
  contact_phone TEXT,
  address TEXT,
  city TEXT,
  postal_code TEXT,
  federal_state TEXT DEFAULT 'BW',
  status TEXT DEFAULT 'active' CHECK(status IN ('active', 'suspended', 'pending')),
  is_demo INTEGER DEFAULT 0,
  suspended_at TIMESTAMP,
  suspended_reason TEXT,
  deleted_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tenant settings
CREATE TABLE IF NOT EXISTS tenant_settings (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id INTEGER NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
  theme_preset TEXT DEFAULT 'classic',
  color_primary TEXT,
  color_secondary TEXT,
  color_accent TEXT,
  color_background TEXT,
  color_text TEXT,
  logo_url TEXT,
  favicon_url TEXT,
  welcome_message TEXT,
  footer_text TEXT,
  website_url TEXT,
  donation_url TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Demo tenant state (for reset functionality)
CREATE TABLE IF NOT EXISTS demo_tenant_state (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id INTEGER NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
  admin_password TEXT NOT NULL,
  last_reset_at TIMESTAMP,
  next_reset_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Users table (per-tenant)
CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
  first_name TEXT,
  last_name TEXT,
  email TEXT,
  phone TEXT,
  password_hash TEXT,
  is_verified INTEGER DEFAULT 0,
  is_active INTEGER DEFAULT 1,
  is_deleted INTEGER DEFAULT 0,
  is_admin INTEGER DEFAULT 0,
  is_super_admin INTEGER DEFAULT 0,
  is_central_admin INTEGER DEFAULT 0,
  must_change_password INTEGER DEFAULT 0,
  verification_token TEXT,
  verification_token_expires TIMESTAMP,
  password_reset_token TEXT,
  password_reset_expires TIMESTAMP,
  profile_photo TEXT,
  anonymous_id TEXT,
  terms_accepted_at TIMESTAMP NOT NULL,
  last_activity_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  deactivated_at TIMESTAMP,
  deactivation_reason TEXT,
  reactivated_at TIMESTAMP,
  deleted_at TIMESTAMP,
  failed_login_attempts INTEGER DEFAULT 0,
  locked_until TEXT,
  last_failed_login TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_users_tenant ON users(tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_last_activity ON users(last_activity_at, is_active);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_tenant_email ON users(COALESCE(tenant_id, 0), email) WHERE email IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_tenant_anonymous ON users(COALESCE(tenant_id, 0), anonymous_id) WHERE anonymous_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_one_super_admin_per_tenant ON users(COALESCE(tenant_id, 0), is_super_admin) WHERE is_super_admin = 1;

-- Color categories (per-tenant)
CREATE TABLE IF NOT EXISTS color_categories (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  hex_code TEXT NOT NULL,
  pattern_icon TEXT,
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_color_categories_tenant ON color_categories(tenant_id);
CREATE INDEX IF NOT EXISTS idx_color_categories_sort ON color_categories(sort_order);
CREATE UNIQUE INDEX IF NOT EXISTS idx_color_categories_tenant_name ON color_categories(COALESCE(tenant_id, 0), name);

-- Dogs table (per-tenant)
CREATE TABLE IF NOT EXISTS dogs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  breed TEXT NOT NULL,
  size TEXT CHECK(size IN ('small', 'medium', 'large')),
  age INTEGER,
  color_id INTEGER REFERENCES color_categories(id),
  photo TEXT,
  photo_thumbnail TEXT,
  special_needs TEXT,
  pickup_location TEXT,
  walk_route TEXT,
  walk_duration INTEGER,
  special_instructions TEXT,
  default_morning_time TEXT,
  default_evening_time TEXT,
  is_available INTEGER DEFAULT 1,
  is_featured INTEGER DEFAULT 0,
  unavailable_reason TEXT,
  unavailable_since TIMESTAMP,
  external_link TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_dogs_tenant ON dogs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_dogs_available ON dogs(is_available);
CREATE INDEX IF NOT EXISTS idx_dogs_color ON dogs(color_id);
CREATE INDEX IF NOT EXISTS idx_dogs_featured ON dogs(is_featured);

-- Bookings table (per-tenant)
CREATE TABLE IF NOT EXISTS bookings (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  dog_id INTEGER NOT NULL REFERENCES dogs(id) ON DELETE CASCADE,
  date DATE NOT NULL,
  scheduled_time TEXT NOT NULL,
  status TEXT DEFAULT 'scheduled' CHECK(status IN ('scheduled', 'completed', 'cancelled')),
  completed_at TIMESTAMP,
  user_notes TEXT,
  admin_cancellation_reason TEXT,
  requires_approval INTEGER DEFAULT 0,
  approval_status TEXT DEFAULT 'approved',
  approved_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
  approved_at TIMESTAMP,
  rejection_reason TEXT,
  reminder_sent_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(dog_id, date, scheduled_time)
);
CREATE INDEX IF NOT EXISTS idx_bookings_tenant ON bookings(tenant_id);
CREATE INDEX IF NOT EXISTS idx_bookings_user ON bookings(user_id);
CREATE INDEX IF NOT EXISTS idx_bookings_dog ON bookings(dog_id);
CREATE INDEX IF NOT EXISTS idx_bookings_date ON bookings(date);
CREATE INDEX IF NOT EXISTS idx_bookings_status ON bookings(status);

-- Blocked dates (per-tenant)
CREATE TABLE IF NOT EXISTS blocked_dates (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
  date DATE NOT NULL,
  dog_id INTEGER REFERENCES dogs(id) ON DELETE CASCADE,
  reason TEXT NOT NULL,
  created_by INTEGER REFERENCES users(id),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_blocked_dates_tenant ON blocked_dates(tenant_id);
CREATE INDEX IF NOT EXISTS idx_blocked_dates_date ON blocked_dates(date);
CREATE UNIQUE INDEX IF NOT EXISTS idx_blocked_dates_tenant_dog_date ON blocked_dates(COALESCE(tenant_id, 0), COALESCE(dog_id, 0), date);

-- System settings (per-tenant)
CREATE TABLE IF NOT EXISTS system_settings (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
  key TEXT NOT NULL,
  value TEXT NOT NULL,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_system_settings_tenant_key ON system_settings(tenant_id, key);

-- User colors junction table (per-tenant)
CREATE TABLE IF NOT EXISTS user_colors (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  color_id INTEGER NOT NULL REFERENCES color_categories(id) ON DELETE RESTRICT,
  granted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  granted_by INTEGER REFERENCES users(id),
  UNIQUE(user_id, color_id)
);
CREATE INDEX IF NOT EXISTS idx_user_colors_tenant ON user_colors(tenant_id);
CREATE INDEX IF NOT EXISTS idx_user_colors_user ON user_colors(user_id);
CREATE INDEX IF NOT EXISTS idx_user_colors_color ON user_colors(color_id);

-- Color requests (per-tenant)
CREATE TABLE IF NOT EXISTS color_requests (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  color_id INTEGER NOT NULL REFERENCES color_categories(id),
  status TEXT DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'denied')),
  admin_message TEXT,
  reviewed_by INTEGER REFERENCES users(id),
  reviewed_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_color_requests_tenant ON color_requests(tenant_id);
CREATE INDEX IF NOT EXISTS idx_color_requests_user ON color_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_color_requests_status ON color_requests(status);

-- Experience requests (per-tenant)
CREATE TABLE IF NOT EXISTS experience_requests (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  requested_level TEXT CHECK(requested_level IN ('blue', 'orange')),
  status TEXT DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'denied')),
  admin_message TEXT,
  reviewed_by INTEGER REFERENCES users(id),
  reviewed_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_experience_requests_tenant ON experience_requests(tenant_id);

-- Reactivation requests (per-tenant)
CREATE TABLE IF NOT EXISTS reactivation_requests (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  status TEXT DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'denied')),
  admin_message TEXT,
  reviewed_by INTEGER REFERENCES users(id),
  reviewed_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_reactivation_requests_tenant ON reactivation_requests(tenant_id);
CREATE INDEX IF NOT EXISTS idx_reactivation_pending ON reactivation_requests(status, created_at);

-- Walk reports (per-tenant)
CREATE TABLE IF NOT EXISTS walk_reports (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
  booking_id INTEGER NOT NULL UNIQUE REFERENCES bookings(id) ON DELETE CASCADE,
  behavior_rating INTEGER NOT NULL CHECK(behavior_rating >= 1 AND behavior_rating <= 5),
  energy_level TEXT NOT NULL CHECK(energy_level IN ('low', 'medium', 'high')),
  notes TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_walk_reports_tenant ON walk_reports(tenant_id);
CREATE INDEX IF NOT EXISTS idx_walk_reports_booking_id ON walk_reports(booking_id);

-- Walk report photos (per-tenant)
CREATE TABLE IF NOT EXISTS walk_report_photos (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
  walk_report_id INTEGER NOT NULL REFERENCES walk_reports(id) ON DELETE CASCADE,
  photo_path TEXT NOT NULL,
  photo_thumbnail TEXT NOT NULL,
  display_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_walk_report_photos_tenant ON walk_report_photos(tenant_id);
CREATE INDEX IF NOT EXISTS idx_walk_report_photos_report_id ON walk_report_photos(walk_report_id);

-- Booking time rules (per-tenant)
CREATE TABLE IF NOT EXISTS booking_time_rules (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
  day_type TEXT NOT NULL,
  rule_name TEXT NOT NULL,
  start_time TEXT NOT NULL,
  end_time TEXT NOT NULL,
  is_blocked INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(tenant_id, day_type, rule_name)
);
CREATE INDEX IF NOT EXISTS idx_booking_time_rules_tenant ON booking_time_rules(tenant_id);

-- Custom holidays (per-tenant)
CREATE TABLE IF NOT EXISTS custom_holidays (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
  date TEXT NOT NULL,
  name TEXT NOT NULL,
  is_active INTEGER NOT NULL DEFAULT 1,
  source TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  created_by INTEGER REFERENCES users(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_custom_holidays_tenant ON custom_holidays(tenant_id);
CREATE INDEX IF NOT EXISTS idx_custom_holidays_date ON custom_holidays(date);
CREATE UNIQUE INDEX IF NOT EXISTS idx_custom_holidays_tenant_date ON custom_holidays(tenant_id, date);

-- Feiertage cache (global - shared across tenants)
CREATE TABLE IF NOT EXISTS feiertage_cache (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  year INTEGER NOT NULL UNIQUE,
  state TEXT NOT NULL,
  data TEXT NOT NULL,
  fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  expires_at TIMESTAMP NOT NULL
);

-- Pricing plans (global)
CREATE TABLE IF NOT EXISTS pricing_plans (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  max_dogs INTEGER NOT NULL DEFAULT -1,
  price_monthly INTEGER NOT NULL DEFAULT 0,
  price_yearly INTEGER NOT NULL DEFAULT 0,
  is_active INTEGER DEFAULT 1,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tenant subscriptions
CREATE TABLE IF NOT EXISTS tenant_subscriptions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  plan_id INTEGER NOT NULL REFERENCES pricing_plans(id),
  status TEXT DEFAULT 'active' CHECK(status IN ('active', 'cancelled', 'past_due', 'trialing')),
  billing_cycle TEXT DEFAULT 'monthly' CHECK(billing_cycle IN ('monthly', 'yearly')),
  current_period_start TIMESTAMP,
  current_period_end TIMESTAMP,
  stripe_customer_id TEXT,
  stripe_subscription_id TEXT,
  cancelled_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_tenant ON tenant_subscriptions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_status ON tenant_subscriptions(status);

-- Insert default pricing plans
INSERT INTO pricing_plans (name, slug, max_dogs, price_monthly, price_yearly, is_active) VALUES
  ('Free', 'free', 10, 0, 0, 1),
  ('Pro', 'pro', -1, 2900, 29000, 1);

-- Insert default color categories (global - tenant_id NULL means shared across all tenants)
INSERT INTO color_categories (id, tenant_id, name, hex_code, sort_order) VALUES
  (1, NULL, 'Gruen', '#22c55e', 1),
  (2, NULL, 'Gelb', '#eab308', 2),
  (3, NULL, 'Orange', '#f97316', 3),
  (4, NULL, 'Hellblau', '#38bdf8', 4),
  (5, NULL, 'Dunkelblau', '#3b82f6', 5);
`,
			"mysql":    `SELECT 1;`,
			"postgres": `SELECT 1;`,
		},
	})
}
