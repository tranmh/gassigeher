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
  tagline TEXT,
  description TEXT,
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

-- Marketing campaigns (FOMO countdown, referral promotions, etc.)
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

-- Referral codes for tenant referrals
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

-- Referral code usage tracking
CREATE TABLE IF NOT EXISTS referral_uses (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code_id INTEGER NOT NULL REFERENCES referral_codes(id) ON DELETE CASCADE,
  referee_tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_referral_uses_code ON referral_uses(code_id);
CREATE INDEX IF NOT EXISTS idx_referral_uses_referee ON referral_uses(referee_tenant_id);

-- Reference page entries (shelters displayed publicly)
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
			"mysql": `
-- Tenants table (central)
CREATE TABLE IF NOT EXISTS tenants (
  id INT AUTO_INCREMENT PRIMARY KEY,
  slug VARCHAR(100) NOT NULL UNIQUE,
  name VARCHAR(255) NOT NULL,
  contact_email VARCHAR(255) NOT NULL,
  contact_phone VARCHAR(50),
  address TEXT,
  city VARCHAR(100),
  postal_code VARCHAR(20),
  federal_state VARCHAR(10) DEFAULT 'BW',
  status ENUM('active', 'suspended', 'pending') DEFAULT 'active',
  is_demo TINYINT(1) DEFAULT 0,
  suspended_at DATETIME,
  suspended_reason TEXT,
  deleted_at DATETIME,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Tenant settings
CREATE TABLE IF NOT EXISTS tenant_settings (
  id INT AUTO_INCREMENT PRIMARY KEY,
  tenant_id INT NOT NULL UNIQUE,
  theme_preset VARCHAR(50) DEFAULT 'classic',
  color_primary VARCHAR(20),
  color_secondary VARCHAR(20),
  color_accent VARCHAR(20),
  color_background VARCHAR(20),
  color_text VARCHAR(20),
  logo_url TEXT,
  favicon_url TEXT,
  welcome_message TEXT,
  tagline VARCHAR(255),
  description TEXT,
  footer_text TEXT,
  website_url VARCHAR(255),
  donation_url VARCHAR(255),
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- Demo tenant state
CREATE TABLE IF NOT EXISTS demo_tenant_state (
  id INT AUTO_INCREMENT PRIMARY KEY,
  tenant_id INT NOT NULL UNIQUE,
  admin_password VARCHAR(255) NOT NULL,
  last_reset_at DATETIME,
  next_reset_at DATETIME,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- Users table (per-tenant)
CREATE TABLE IF NOT EXISTS users (
  id INT AUTO_INCREMENT PRIMARY KEY,
  tenant_id INT,
  first_name VARCHAR(100),
  last_name VARCHAR(100),
  email VARCHAR(255),
  phone VARCHAR(50),
  password_hash VARCHAR(255),
  is_verified TINYINT(1) DEFAULT 0,
  is_active TINYINT(1) DEFAULT 1,
  is_deleted TINYINT(1) DEFAULT 0,
  is_admin TINYINT(1) DEFAULT 0,
  is_super_admin TINYINT(1) DEFAULT 0,
  is_central_admin TINYINT(1) DEFAULT 0,
  must_change_password TINYINT(1) DEFAULT 0,
  verification_token VARCHAR(255),
  verification_token_expires DATETIME,
  password_reset_token VARCHAR(255),
  password_reset_expires DATETIME,
  profile_photo VARCHAR(255),
  anonymous_id VARCHAR(100),
  terms_accepted_at DATETIME NOT NULL,
  last_activity_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  deactivated_at DATETIME,
  deactivation_reason TEXT,
  reactivated_at DATETIME,
  deleted_at DATETIME,
  failed_login_attempts INT DEFAULT 0,
  locked_until VARCHAR(50),
  last_failed_login VARCHAR(50),
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_users_tenant (tenant_id),
  INDEX idx_users_email (email),
  INDEX idx_users_last_activity (last_activity_at, is_active),
  UNIQUE KEY idx_users_tenant_email (tenant_id, email),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- Color categories (per-tenant)
CREATE TABLE IF NOT EXISTS color_categories (
  id INT AUTO_INCREMENT PRIMARY KEY,
  tenant_id INT,
  name VARCHAR(100) NOT NULL,
  hex_code VARCHAR(20) NOT NULL,
  pattern_icon VARCHAR(100),
  sort_order INT NOT NULL DEFAULT 0,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_color_categories_tenant (tenant_id),
  INDEX idx_color_categories_sort (sort_order),
  UNIQUE KEY idx_color_categories_tenant_name (tenant_id, name),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- Dogs table (per-tenant)
CREATE TABLE IF NOT EXISTS dogs (
  id INT AUTO_INCREMENT PRIMARY KEY,
  tenant_id INT,
  name VARCHAR(100) NOT NULL,
  breed VARCHAR(100) NOT NULL,
  size ENUM('small', 'medium', 'large'),
  age INT,
  color_id INT,
  photo VARCHAR(255),
  photo_thumbnail VARCHAR(255),
  special_needs TEXT,
  pickup_location TEXT,
  walk_route TEXT,
  walk_duration INT,
  special_instructions TEXT,
  default_morning_time VARCHAR(10),
  default_evening_time VARCHAR(10),
  is_available TINYINT(1) DEFAULT 1,
  is_featured TINYINT(1) DEFAULT 0,
  unavailable_reason TEXT,
  unavailable_since DATETIME,
  external_link VARCHAR(500),
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_dogs_tenant (tenant_id),
  INDEX idx_dogs_available (is_available),
  INDEX idx_dogs_color (color_id),
  INDEX idx_dogs_featured (is_featured),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
  FOREIGN KEY (color_id) REFERENCES color_categories(id)
);

-- Bookings table (per-tenant)
CREATE TABLE IF NOT EXISTS bookings (
  id INT AUTO_INCREMENT PRIMARY KEY,
  tenant_id INT,
  user_id INT NOT NULL,
  dog_id INT NOT NULL,
  date DATE NOT NULL,
  scheduled_time VARCHAR(10) NOT NULL,
  status ENUM('scheduled', 'completed', 'cancelled') DEFAULT 'scheduled',
  completed_at DATETIME,
  user_notes TEXT,
  admin_cancellation_reason TEXT,
  requires_approval TINYINT(1) DEFAULT 0,
  approval_status VARCHAR(20) DEFAULT 'approved',
  approved_by INT,
  approved_at DATETIME,
  rejection_reason TEXT,
  reminder_sent_at DATETIME,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY idx_booking_unique (dog_id, date, scheduled_time),
  INDEX idx_bookings_tenant (tenant_id),
  INDEX idx_bookings_user (user_id),
  INDEX idx_bookings_dog (dog_id),
  INDEX idx_bookings_date (date),
  INDEX idx_bookings_status (status),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (dog_id) REFERENCES dogs(id) ON DELETE CASCADE,
  FOREIGN KEY (approved_by) REFERENCES users(id) ON DELETE SET NULL
);

-- Blocked dates (per-tenant)
CREATE TABLE IF NOT EXISTS blocked_dates (
  id INT AUTO_INCREMENT PRIMARY KEY,
  tenant_id INT,
  date DATE NOT NULL,
  dog_id INT,
  reason TEXT NOT NULL,
  created_by INT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY idx_blocked_dates_unique (tenant_id, dog_id, date),
  INDEX idx_blocked_dates_tenant (tenant_id),
  INDEX idx_blocked_dates_date (date),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
  FOREIGN KEY (dog_id) REFERENCES dogs(id) ON DELETE CASCADE,
  FOREIGN KEY (created_by) REFERENCES users(id)
);

-- System settings (per-tenant)
CREATE TABLE IF NOT EXISTS system_settings (
  id INT AUTO_INCREMENT PRIMARY KEY,
  tenant_id INT,
  ` + "`key`" + ` VARCHAR(100) NOT NULL,
  value TEXT NOT NULL,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY idx_system_settings_tenant_key (tenant_id, ` + "`key`" + `),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- User colors junction table (per-tenant)
CREATE TABLE IF NOT EXISTS user_colors (
  id INT AUTO_INCREMENT PRIMARY KEY,
  tenant_id INT,
  user_id INT NOT NULL,
  color_id INT NOT NULL,
  granted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  granted_by INT,
  UNIQUE KEY idx_user_colors_unique (user_id, color_id),
  INDEX idx_user_colors_tenant (tenant_id),
  INDEX idx_user_colors_user (user_id),
  INDEX idx_user_colors_color (color_id),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (color_id) REFERENCES color_categories(id) ON DELETE RESTRICT,
  FOREIGN KEY (granted_by) REFERENCES users(id)
);

-- Color requests (per-tenant)
CREATE TABLE IF NOT EXISTS color_requests (
  id INT AUTO_INCREMENT PRIMARY KEY,
  tenant_id INT,
  user_id INT NOT NULL,
  color_id INT NOT NULL,
  status ENUM('pending', 'approved', 'denied') DEFAULT 'pending',
  admin_message TEXT,
  reviewed_by INT,
  reviewed_at DATETIME,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_color_requests_tenant (tenant_id),
  INDEX idx_color_requests_user (user_id),
  INDEX idx_color_requests_status (status),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (color_id) REFERENCES color_categories(id),
  FOREIGN KEY (reviewed_by) REFERENCES users(id)
);

-- Experience requests (per-tenant)
CREATE TABLE IF NOT EXISTS experience_requests (
  id INT AUTO_INCREMENT PRIMARY KEY,
  tenant_id INT,
  user_id INT NOT NULL,
  requested_level ENUM('blue', 'orange'),
  status ENUM('pending', 'approved', 'denied') DEFAULT 'pending',
  admin_message TEXT,
  reviewed_by INT,
  reviewed_at DATETIME,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_experience_requests_tenant (tenant_id),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (reviewed_by) REFERENCES users(id)
);

-- Reactivation requests (per-tenant)
CREATE TABLE IF NOT EXISTS reactivation_requests (
  id INT AUTO_INCREMENT PRIMARY KEY,
  tenant_id INT,
  user_id INT NOT NULL,
  status ENUM('pending', 'approved', 'denied') DEFAULT 'pending',
  admin_message TEXT,
  reviewed_by INT,
  reviewed_at DATETIME,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_reactivation_requests_tenant (tenant_id),
  INDEX idx_reactivation_pending (status, created_at),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (reviewed_by) REFERENCES users(id)
);

-- Walk reports (per-tenant)
CREATE TABLE IF NOT EXISTS walk_reports (
  id INT AUTO_INCREMENT PRIMARY KEY,
  tenant_id INT,
  booking_id INT NOT NULL UNIQUE,
  behavior_rating INT NOT NULL,
  energy_level ENUM('low', 'medium', 'high') NOT NULL,
  notes TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_walk_reports_tenant (tenant_id),
  INDEX idx_walk_reports_booking_id (booking_id),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
  FOREIGN KEY (booking_id) REFERENCES bookings(id) ON DELETE CASCADE
);

-- Walk report photos (per-tenant)
CREATE TABLE IF NOT EXISTS walk_report_photos (
  id INT AUTO_INCREMENT PRIMARY KEY,
  tenant_id INT,
  walk_report_id INT NOT NULL,
  photo_path VARCHAR(255) NOT NULL,
  photo_thumbnail VARCHAR(255) NOT NULL,
  display_order INT NOT NULL DEFAULT 0,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_walk_report_photos_tenant (tenant_id),
  INDEX idx_walk_report_photos_report_id (walk_report_id),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
  FOREIGN KEY (walk_report_id) REFERENCES walk_reports(id) ON DELETE CASCADE
);

-- Booking time rules (per-tenant)
CREATE TABLE IF NOT EXISTS booking_time_rules (
  id INT AUTO_INCREMENT PRIMARY KEY,
  tenant_id INT,
  day_type VARCHAR(20) NOT NULL,
  rule_name VARCHAR(100) NOT NULL,
  start_time VARCHAR(10) NOT NULL,
  end_time VARCHAR(10) NOT NULL,
  is_blocked TINYINT(1) NOT NULL DEFAULT 0,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY idx_booking_time_rules_unique (tenant_id, day_type, rule_name),
  INDEX idx_booking_time_rules_tenant (tenant_id),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- Custom holidays (per-tenant)
CREATE TABLE IF NOT EXISTS custom_holidays (
  id INT AUTO_INCREMENT PRIMARY KEY,
  tenant_id INT,
  date VARCHAR(20) NOT NULL,
  name VARCHAR(255) NOT NULL,
  is_active TINYINT(1) NOT NULL DEFAULT 1,
  source VARCHAR(50) NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  created_by INT,
  UNIQUE KEY idx_custom_holidays_tenant_date (tenant_id, date),
  INDEX idx_custom_holidays_tenant (tenant_id),
  INDEX idx_custom_holidays_date (date),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
  FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
);

-- Feiertage cache (global)
CREATE TABLE IF NOT EXISTS feiertage_cache (
  id INT AUTO_INCREMENT PRIMARY KEY,
  year INT NOT NULL UNIQUE,
  state VARCHAR(10) NOT NULL,
  data LONGTEXT NOT NULL,
  fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  expires_at DATETIME NOT NULL
);

-- Pricing plans (global)
CREATE TABLE IF NOT EXISTS pricing_plans (
  id INT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  slug VARCHAR(50) NOT NULL UNIQUE,
  max_dogs INT NOT NULL DEFAULT -1,
  price_monthly INT NOT NULL DEFAULT 0,
  price_yearly INT NOT NULL DEFAULT 0,
  is_active TINYINT(1) DEFAULT 1,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Tenant subscriptions
CREATE TABLE IF NOT EXISTS tenant_subscriptions (
  id INT AUTO_INCREMENT PRIMARY KEY,
  tenant_id INT NOT NULL,
  plan_id INT NOT NULL,
  status ENUM('active', 'cancelled', 'past_due', 'trialing') DEFAULT 'active',
  billing_cycle ENUM('monthly', 'yearly') DEFAULT 'monthly',
  current_period_start DATETIME,
  current_period_end DATETIME,
  stripe_customer_id VARCHAR(255),
  stripe_subscription_id VARCHAR(255),
  cancelled_at DATETIME,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_tenant_subscriptions_tenant (tenant_id),
  INDEX idx_tenant_subscriptions_status (status),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
  FOREIGN KEY (plan_id) REFERENCES pricing_plans(id)
);

-- Marketing campaigns
CREATE TABLE IF NOT EXISTS marketing_campaigns (
  id INT AUTO_INCREMENT PRIMARY KEY,
  type ENUM('fomo_countdown', 'referral', 'reference_page', 'custom') NOT NULL,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  config JSON,
  is_active TINYINT(1) DEFAULT 0,
  start_date DATETIME,
  end_date DATETIME,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Referral codes
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
  INDEX idx_referral_codes_code (code),
  INDEX idx_referral_codes_referrer (referrer_tenant_id),
  FOREIGN KEY (referrer_tenant_id) REFERENCES tenants(id) ON DELETE SET NULL
);

-- Referral uses
CREATE TABLE IF NOT EXISTS referral_uses (
  id INT AUTO_INCREMENT PRIMARY KEY,
  code_id INT NOT NULL,
  referee_tenant_id INT NOT NULL,
  applied_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_referral_uses_code (code_id),
  INDEX idx_referral_uses_referee (referee_tenant_id),
  FOREIGN KEY (code_id) REFERENCES referral_codes(id) ON DELETE CASCADE,
  FOREIGN KEY (referee_tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- Reference entries
CREATE TABLE IF NOT EXISTS reference_entries (
  id INT AUTO_INCREMENT PRIMARY KEY,
  tenant_id INT NOT NULL UNIQUE,
  display_name VARCHAR(255) NOT NULL,
  city VARCHAR(100),
  federal_state VARCHAR(50),
  website_url VARCHAR(255),
  testimonial TEXT,
  logo_url VARCHAR(255),
  is_approved TINYINT(1) DEFAULT 0,
  display_order INT DEFAULT 0,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_reference_entries_approved (is_approved, display_order),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- Insert default pricing plans
INSERT INTO pricing_plans (name, slug, max_dogs, price_monthly, price_yearly, is_active) VALUES
  ('Free', 'free', 10, 0, 0, 1),
  ('Pro', 'pro', -1, 2900, 29000, 1);

-- Insert default color categories
INSERT INTO color_categories (id, tenant_id, name, hex_code, sort_order) VALUES
  (1, NULL, 'Gruen', '#22c55e', 1),
  (2, NULL, 'Gelb', '#eab308', 2),
  (3, NULL, 'Orange', '#f97316', 3),
  (4, NULL, 'Hellblau', '#38bdf8', 4),
  (5, NULL, 'Dunkelblau', '#3b82f6', 5);
`,
			"postgres": `
-- Tenants table (central)
CREATE TABLE IF NOT EXISTS tenants (
  id SERIAL PRIMARY KEY,
  slug VARCHAR(100) NOT NULL UNIQUE,
  name VARCHAR(255) NOT NULL,
  contact_email VARCHAR(255) NOT NULL,
  contact_phone VARCHAR(50),
  address TEXT,
  city VARCHAR(100),
  postal_code VARCHAR(20),
  federal_state VARCHAR(10) DEFAULT 'BW',
  status VARCHAR(20) DEFAULT 'active' CHECK(status IN ('active', 'suspended', 'pending')),
  is_demo BOOLEAN DEFAULT FALSE,
  suspended_at TIMESTAMP,
  suspended_reason TEXT,
  deleted_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tenant settings
CREATE TABLE IF NOT EXISTS tenant_settings (
  id SERIAL PRIMARY KEY,
  tenant_id INTEGER NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
  theme_preset VARCHAR(50) DEFAULT 'classic',
  color_primary VARCHAR(20),
  color_secondary VARCHAR(20),
  color_accent VARCHAR(20),
  color_background VARCHAR(20),
  color_text VARCHAR(20),
  logo_url TEXT,
  favicon_url TEXT,
  welcome_message TEXT,
  tagline VARCHAR(255),
  description TEXT,
  footer_text TEXT,
  website_url VARCHAR(255),
  donation_url VARCHAR(255),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Demo tenant state
CREATE TABLE IF NOT EXISTS demo_tenant_state (
  id SERIAL PRIMARY KEY,
  tenant_id INTEGER NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
  admin_password VARCHAR(255) NOT NULL,
  last_reset_at TIMESTAMP,
  next_reset_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Users table (per-tenant)
CREATE TABLE IF NOT EXISTS users (
  id SERIAL PRIMARY KEY,
  tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
  first_name VARCHAR(100),
  last_name VARCHAR(100),
  email VARCHAR(255),
  phone VARCHAR(50),
  password_hash VARCHAR(255),
  is_verified BOOLEAN DEFAULT FALSE,
  is_active BOOLEAN DEFAULT TRUE,
  is_deleted BOOLEAN DEFAULT FALSE,
  is_admin BOOLEAN DEFAULT FALSE,
  is_super_admin BOOLEAN DEFAULT FALSE,
  is_central_admin BOOLEAN DEFAULT FALSE,
  must_change_password BOOLEAN DEFAULT FALSE,
  verification_token VARCHAR(255),
  verification_token_expires TIMESTAMP,
  password_reset_token VARCHAR(255),
  password_reset_expires TIMESTAMP,
  profile_photo VARCHAR(255),
  anonymous_id VARCHAR(100),
  terms_accepted_at TIMESTAMP NOT NULL,
  last_activity_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  deactivated_at TIMESTAMP,
  deactivation_reason TEXT,
  reactivated_at TIMESTAMP,
  deleted_at TIMESTAMP,
  failed_login_attempts INTEGER DEFAULT 0,
  locked_until VARCHAR(50),
  last_failed_login VARCHAR(50),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_users_tenant ON users(tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_last_activity ON users(last_activity_at, is_active);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_tenant_email ON users(COALESCE(tenant_id, 0), email) WHERE email IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_tenant_anonymous ON users(COALESCE(tenant_id, 0), anonymous_id) WHERE anonymous_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_one_super_admin_per_tenant ON users(COALESCE(tenant_id, 0), is_super_admin) WHERE is_super_admin = TRUE;

-- Color categories (per-tenant)
CREATE TABLE IF NOT EXISTS color_categories (
  id SERIAL PRIMARY KEY,
  tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
  name VARCHAR(100) NOT NULL,
  hex_code VARCHAR(20) NOT NULL,
  pattern_icon VARCHAR(100),
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_color_categories_tenant ON color_categories(tenant_id);
CREATE INDEX IF NOT EXISTS idx_color_categories_sort ON color_categories(sort_order);
CREATE UNIQUE INDEX IF NOT EXISTS idx_color_categories_tenant_name ON color_categories(COALESCE(tenant_id, 0), name);

-- Dogs table (per-tenant)
CREATE TABLE IF NOT EXISTS dogs (
  id SERIAL PRIMARY KEY,
  tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
  name VARCHAR(100) NOT NULL,
  breed VARCHAR(100) NOT NULL,
  size VARCHAR(20) CHECK(size IN ('small', 'medium', 'large')),
  age INTEGER,
  color_id INTEGER REFERENCES color_categories(id),
  photo VARCHAR(255),
  photo_thumbnail VARCHAR(255),
  special_needs TEXT,
  pickup_location TEXT,
  walk_route TEXT,
  walk_duration INTEGER,
  special_instructions TEXT,
  default_morning_time VARCHAR(10),
  default_evening_time VARCHAR(10),
  is_available BOOLEAN DEFAULT TRUE,
  is_featured BOOLEAN DEFAULT FALSE,
  unavailable_reason TEXT,
  unavailable_since TIMESTAMP,
  external_link VARCHAR(500),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_dogs_tenant ON dogs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_dogs_available ON dogs(is_available);
CREATE INDEX IF NOT EXISTS idx_dogs_color ON dogs(color_id);
CREATE INDEX IF NOT EXISTS idx_dogs_featured ON dogs(is_featured);

-- Bookings table (per-tenant)
CREATE TABLE IF NOT EXISTS bookings (
  id SERIAL PRIMARY KEY,
  tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  dog_id INTEGER NOT NULL REFERENCES dogs(id) ON DELETE CASCADE,
  date DATE NOT NULL,
  scheduled_time VARCHAR(10) NOT NULL,
  status VARCHAR(20) DEFAULT 'scheduled' CHECK(status IN ('scheduled', 'completed', 'cancelled')),
  completed_at TIMESTAMP,
  user_notes TEXT,
  admin_cancellation_reason TEXT,
  requires_approval BOOLEAN DEFAULT FALSE,
  approval_status VARCHAR(20) DEFAULT 'approved',
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
  id SERIAL PRIMARY KEY,
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
  id SERIAL PRIMARY KEY,
  tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
  key VARCHAR(100) NOT NULL,
  value TEXT NOT NULL,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_system_settings_tenant_key ON system_settings(tenant_id, key);

-- User colors junction table (per-tenant)
CREATE TABLE IF NOT EXISTS user_colors (
  id SERIAL PRIMARY KEY,
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
  id SERIAL PRIMARY KEY,
  tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  color_id INTEGER NOT NULL REFERENCES color_categories(id),
  status VARCHAR(20) DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'denied')),
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
  id SERIAL PRIMARY KEY,
  tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  requested_level VARCHAR(20) CHECK(requested_level IN ('blue', 'orange')),
  status VARCHAR(20) DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'denied')),
  admin_message TEXT,
  reviewed_by INTEGER REFERENCES users(id),
  reviewed_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_experience_requests_tenant ON experience_requests(tenant_id);

-- Reactivation requests (per-tenant)
CREATE TABLE IF NOT EXISTS reactivation_requests (
  id SERIAL PRIMARY KEY,
  tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  status VARCHAR(20) DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'denied')),
  admin_message TEXT,
  reviewed_by INTEGER REFERENCES users(id),
  reviewed_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_reactivation_requests_tenant ON reactivation_requests(tenant_id);
CREATE INDEX IF NOT EXISTS idx_reactivation_pending ON reactivation_requests(status, created_at);

-- Walk reports (per-tenant)
CREATE TABLE IF NOT EXISTS walk_reports (
  id SERIAL PRIMARY KEY,
  tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
  booking_id INTEGER NOT NULL UNIQUE REFERENCES bookings(id) ON DELETE CASCADE,
  behavior_rating INTEGER NOT NULL CHECK(behavior_rating >= 1 AND behavior_rating <= 5),
  energy_level VARCHAR(20) NOT NULL CHECK(energy_level IN ('low', 'medium', 'high')),
  notes TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_walk_reports_tenant ON walk_reports(tenant_id);
CREATE INDEX IF NOT EXISTS idx_walk_reports_booking_id ON walk_reports(booking_id);

-- Walk report photos (per-tenant)
CREATE TABLE IF NOT EXISTS walk_report_photos (
  id SERIAL PRIMARY KEY,
  tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
  walk_report_id INTEGER NOT NULL REFERENCES walk_reports(id) ON DELETE CASCADE,
  photo_path VARCHAR(255) NOT NULL,
  photo_thumbnail VARCHAR(255) NOT NULL,
  display_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_walk_report_photos_tenant ON walk_report_photos(tenant_id);
CREATE INDEX IF NOT EXISTS idx_walk_report_photos_report_id ON walk_report_photos(walk_report_id);

-- Booking time rules (per-tenant)
CREATE TABLE IF NOT EXISTS booking_time_rules (
  id SERIAL PRIMARY KEY,
  tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
  day_type VARCHAR(20) NOT NULL,
  rule_name VARCHAR(100) NOT NULL,
  start_time VARCHAR(10) NOT NULL,
  end_time VARCHAR(10) NOT NULL,
  is_blocked BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(tenant_id, day_type, rule_name)
);
CREATE INDEX IF NOT EXISTS idx_booking_time_rules_tenant ON booking_time_rules(tenant_id);

-- Custom holidays (per-tenant)
CREATE TABLE IF NOT EXISTS custom_holidays (
  id SERIAL PRIMARY KEY,
  tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
  date VARCHAR(20) NOT NULL,
  name VARCHAR(255) NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  source VARCHAR(50) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  created_by INTEGER REFERENCES users(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_custom_holidays_tenant ON custom_holidays(tenant_id);
CREATE INDEX IF NOT EXISTS idx_custom_holidays_date ON custom_holidays(date);
CREATE UNIQUE INDEX IF NOT EXISTS idx_custom_holidays_tenant_date ON custom_holidays(tenant_id, date);

-- Feiertage cache (global)
CREATE TABLE IF NOT EXISTS feiertage_cache (
  id SERIAL PRIMARY KEY,
  year INTEGER NOT NULL UNIQUE,
  state VARCHAR(10) NOT NULL,
  data TEXT NOT NULL,
  fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  expires_at TIMESTAMP NOT NULL
);

-- Pricing plans (global)
CREATE TABLE IF NOT EXISTS pricing_plans (
  id SERIAL PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  slug VARCHAR(50) NOT NULL UNIQUE,
  max_dogs INTEGER NOT NULL DEFAULT -1,
  price_monthly INTEGER NOT NULL DEFAULT 0,
  price_yearly INTEGER NOT NULL DEFAULT 0,
  is_active BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tenant subscriptions
CREATE TABLE IF NOT EXISTS tenant_subscriptions (
  id SERIAL PRIMARY KEY,
  tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  plan_id INTEGER NOT NULL REFERENCES pricing_plans(id),
  status VARCHAR(20) DEFAULT 'active' CHECK(status IN ('active', 'cancelled', 'past_due', 'trialing')),
  billing_cycle VARCHAR(20) DEFAULT 'monthly' CHECK(billing_cycle IN ('monthly', 'yearly')),
  current_period_start TIMESTAMP,
  current_period_end TIMESTAMP,
  stripe_customer_id VARCHAR(255),
  stripe_subscription_id VARCHAR(255),
  cancelled_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_tenant ON tenant_subscriptions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_status ON tenant_subscriptions(status);

-- Marketing campaigns
CREATE TABLE IF NOT EXISTS marketing_campaigns (
  id SERIAL PRIMARY KEY,
  type VARCHAR(30) NOT NULL CHECK(type IN ('fomo_countdown', 'referral', 'reference_page', 'custom')),
  name VARCHAR(255) NOT NULL,
  description TEXT,
  config JSONB,
  is_active BOOLEAN DEFAULT FALSE,
  start_date TIMESTAMP,
  end_date TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Referral codes
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
CREATE INDEX IF NOT EXISTS idx_referral_codes_referrer ON referral_codes(referrer_tenant_id);

-- Referral uses
CREATE TABLE IF NOT EXISTS referral_uses (
  id SERIAL PRIMARY KEY,
  code_id INTEGER NOT NULL REFERENCES referral_codes(id) ON DELETE CASCADE,
  referee_tenant_id INTEGER NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_referral_uses_code ON referral_uses(code_id);
CREATE INDEX IF NOT EXISTS idx_referral_uses_referee ON referral_uses(referee_tenant_id);

-- Reference entries
CREATE TABLE IF NOT EXISTS reference_entries (
  id SERIAL PRIMARY KEY,
  tenant_id INTEGER NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
  display_name VARCHAR(255) NOT NULL,
  city VARCHAR(100),
  federal_state VARCHAR(50),
  website_url VARCHAR(255),
  testimonial TEXT,
  logo_url VARCHAR(255),
  is_approved BOOLEAN DEFAULT FALSE,
  display_order INTEGER DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_reference_entries_approved ON reference_entries(is_approved, display_order);

-- Insert default pricing plans
INSERT INTO pricing_plans (name, slug, max_dogs, price_monthly, price_yearly, is_active) VALUES
  ('Free', 'free', 10, 0, 0, TRUE),
  ('Pro', 'pro', -1, 2900, 29000, TRUE);

-- Insert default color categories
INSERT INTO color_categories (id, tenant_id, name, hex_code, sort_order) VALUES
  (1, NULL, 'Gruen', '#22c55e', 1),
  (2, NULL, 'Gelb', '#eab308', 2),
  (3, NULL, 'Orange', '#f97316', 3),
  (4, NULL, 'Hellblau', '#38bdf8', 4),
  (5, NULL, 'Dunkelblau', '#3b82f6', 5);
`,
		},
	})
}
