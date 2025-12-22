package database

func init() {
	RegisterMigration(&Migration{
		ID:          "006_update_constraints",
		Description: "Update unique constraints to include tenant_id for multi-tenancy",
		Up: map[string]string{
			// SQLite: Cannot modify constraints easily, create new indexes
			// Note: SQLite doesn't support dropping constraints, so we add tenant-scoped indexes
			"sqlite": `
-- Users: email should be unique per tenant (not globally)
-- SQLite doesn't support DROP CONSTRAINT, so we create a new partial index
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_tenant_email ON users(tenant_id, email) WHERE email IS NOT NULL AND tenant_id IS NOT NULL;

-- Color categories: name should be unique per tenant
CREATE UNIQUE INDEX IF NOT EXISTS idx_color_categories_tenant_name ON color_categories(tenant_id, name) WHERE tenant_id IS NOT NULL;

-- System settings: key should be unique per tenant
CREATE UNIQUE INDEX IF NOT EXISTS idx_system_settings_tenant_key ON system_settings(tenant_id, key) WHERE tenant_id IS NOT NULL;

-- Booking time rules: day_type + rule_name should be unique per tenant
CREATE UNIQUE INDEX IF NOT EXISTS idx_booking_time_rules_tenant_unique ON booking_time_rules(tenant_id, day_type, rule_name) WHERE tenant_id IS NOT NULL;

-- Custom holidays: date should be unique per tenant
CREATE UNIQUE INDEX IF NOT EXISTS idx_custom_holidays_tenant_date ON custom_holidays(tenant_id, date) WHERE tenant_id IS NOT NULL;

-- User colors: user_id + color_id should be unique per tenant
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_colors_tenant_unique ON user_colors(tenant_id, user_id, color_id) WHERE tenant_id IS NOT NULL;

-- Bookings: dog_id + date + scheduled_time should be unique per tenant
CREATE UNIQUE INDEX IF NOT EXISTS idx_bookings_tenant_unique ON bookings(tenant_id, dog_id, date, scheduled_time) WHERE tenant_id IS NOT NULL;

-- Walk reports: booking_id should be unique per tenant
CREATE UNIQUE INDEX IF NOT EXISTS idx_walk_reports_tenant_booking ON walk_reports(tenant_id, booking_id) WHERE tenant_id IS NOT NULL;
`,
			"mysql": `
-- Users: email should be unique per tenant
-- First drop the old unique constraint, then add tenant-scoped one
ALTER TABLE users DROP INDEX email;
ALTER TABLE users ADD UNIQUE INDEX idx_users_tenant_email (tenant_id, email);

-- Color categories: name should be unique per tenant
ALTER TABLE color_categories DROP INDEX name;
ALTER TABLE color_categories ADD UNIQUE INDEX idx_color_categories_tenant_name (tenant_id, name);

-- System settings: Drop old PK and create composite PK with tenant_id
-- MySQL requires special handling for primary keys
-- Temporarily disable FK checks for this operation
SET FOREIGN_KEY_CHECKS = 0;
-- First update any NULL tenant_id values to 1 (default tenant)
UPDATE system_settings SET tenant_id = 1 WHERE tenant_id IS NULL;
ALTER TABLE system_settings DROP PRIMARY KEY;
ALTER TABLE system_settings MODIFY COLUMN tenant_id INT NOT NULL DEFAULT 1;
ALTER TABLE system_settings ADD PRIMARY KEY (tenant_id, ` + "`key`" + `);
SET FOREIGN_KEY_CHECKS = 1;

-- Booking time rules: day_type + rule_name should be unique per tenant
ALTER TABLE booking_time_rules DROP INDEX unique_day_rule;
ALTER TABLE booking_time_rules ADD UNIQUE INDEX idx_booking_time_rules_tenant_unique (tenant_id, day_type, rule_name);

-- Custom holidays: date should be unique per tenant
ALTER TABLE custom_holidays DROP INDEX date;
ALTER TABLE custom_holidays ADD UNIQUE INDEX idx_custom_holidays_tenant_date (tenant_id, date);

-- User colors: user_id + color_id should be unique per tenant
ALTER TABLE user_colors DROP INDEX unique_user_color;
ALTER TABLE user_colors ADD UNIQUE INDEX idx_user_colors_tenant_unique (tenant_id, user_id, color_id);

-- Bookings: dog_id + date + scheduled_time should be unique per tenant
ALTER TABLE bookings DROP INDEX unique_dog_date_time;
ALTER TABLE bookings ADD UNIQUE INDEX idx_bookings_tenant_unique (tenant_id, dog_id, date, scheduled_time);

-- Walk reports: booking_id should be unique per tenant
ALTER TABLE walk_reports DROP INDEX booking_id;
ALTER TABLE walk_reports ADD UNIQUE INDEX idx_walk_reports_tenant_booking (tenant_id, booking_id);
`,
			"postgres": `
-- Users: email should be unique per tenant (not globally)
-- Drop constraint first (this also removes the implicit index)
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_tenant_email ON users(tenant_id, email) WHERE email IS NOT NULL;

-- Color categories: name should be unique per tenant
ALTER TABLE color_categories DROP CONSTRAINT IF EXISTS color_categories_name_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_color_categories_tenant_name ON color_categories(tenant_id, name);

-- System settings: key should be unique per tenant
ALTER TABLE system_settings DROP CONSTRAINT IF EXISTS system_settings_pkey;
ALTER TABLE system_settings ADD PRIMARY KEY (tenant_id, key);

-- Booking time rules: day_type + rule_name should be unique per tenant
ALTER TABLE booking_time_rules DROP CONSTRAINT IF EXISTS booking_time_rules_day_type_rule_name_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_booking_time_rules_tenant_unique ON booking_time_rules(tenant_id, day_type, rule_name);

-- Custom holidays: date should be unique per tenant
ALTER TABLE custom_holidays DROP CONSTRAINT IF EXISTS custom_holidays_date_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_custom_holidays_tenant_date ON custom_holidays(tenant_id, date);

-- User colors: user_id + color_id should be unique per tenant
ALTER TABLE user_colors DROP CONSTRAINT IF EXISTS user_colors_user_id_color_id_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_colors_tenant_unique ON user_colors(tenant_id, user_id, color_id);

-- Bookings: dog_id + date + scheduled_time should be unique per tenant
ALTER TABLE bookings DROP CONSTRAINT IF EXISTS bookings_dog_id_date_scheduled_time_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_bookings_tenant_unique ON bookings(tenant_id, dog_id, date, scheduled_time);

-- Walk reports: booking_id should be unique per tenant
ALTER TABLE walk_reports DROP CONSTRAINT IF EXISTS walk_reports_booking_id_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_walk_reports_tenant_booking ON walk_reports(tenant_id, booking_id);
`,
		},
	})
}
