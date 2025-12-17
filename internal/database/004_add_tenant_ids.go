package database

func init() {
	RegisterMigration(&Migration{
		ID:          "004_add_tenant_ids",
		Description: "Add tenant_id column to all tenant-scoped tables",
		Up: map[string]string{
			"sqlite": `
-- Add tenant_id to users table
ALTER TABLE users ADD COLUMN tenant_id INTEGER REFERENCES tenants(id);
CREATE INDEX IF NOT EXISTS idx_users_tenant ON users(tenant_id);

-- Add tenant_id to color_categories table
ALTER TABLE color_categories ADD COLUMN tenant_id INTEGER REFERENCES tenants(id);
CREATE INDEX IF NOT EXISTS idx_color_categories_tenant ON color_categories(tenant_id);

-- Add tenant_id to dogs table
ALTER TABLE dogs ADD COLUMN tenant_id INTEGER REFERENCES tenants(id);
CREATE INDEX IF NOT EXISTS idx_dogs_tenant ON dogs(tenant_id);

-- Add tenant_id to bookings table
ALTER TABLE bookings ADD COLUMN tenant_id INTEGER REFERENCES tenants(id);
CREATE INDEX IF NOT EXISTS idx_bookings_tenant ON bookings(tenant_id);

-- Add tenant_id to blocked_dates table
ALTER TABLE blocked_dates ADD COLUMN tenant_id INTEGER REFERENCES tenants(id);
CREATE INDEX IF NOT EXISTS idx_blocked_dates_tenant ON blocked_dates(tenant_id);

-- Add tenant_id to experience_requests table
ALTER TABLE experience_requests ADD COLUMN tenant_id INTEGER REFERENCES tenants(id);
CREATE INDEX IF NOT EXISTS idx_experience_requests_tenant ON experience_requests(tenant_id);

-- Add tenant_id to system_settings table
ALTER TABLE system_settings ADD COLUMN tenant_id INTEGER REFERENCES tenants(id);

-- Add tenant_id to reactivation_requests table
ALTER TABLE reactivation_requests ADD COLUMN tenant_id INTEGER REFERENCES tenants(id);
CREATE INDEX IF NOT EXISTS idx_reactivation_requests_tenant ON reactivation_requests(tenant_id);

-- Add tenant_id to booking_time_rules table
ALTER TABLE booking_time_rules ADD COLUMN tenant_id INTEGER REFERENCES tenants(id);
CREATE INDEX IF NOT EXISTS idx_booking_time_rules_tenant ON booking_time_rules(tenant_id);

-- Add tenant_id to custom_holidays table
ALTER TABLE custom_holidays ADD COLUMN tenant_id INTEGER REFERENCES tenants(id);
CREATE INDEX IF NOT EXISTS idx_custom_holidays_tenant ON custom_holidays(tenant_id);

-- Add tenant_id to walk_reports table
ALTER TABLE walk_reports ADD COLUMN tenant_id INTEGER REFERENCES tenants(id);
CREATE INDEX IF NOT EXISTS idx_walk_reports_tenant ON walk_reports(tenant_id);

-- Add tenant_id to walk_report_photos table
ALTER TABLE walk_report_photos ADD COLUMN tenant_id INTEGER REFERENCES tenants(id);
CREATE INDEX IF NOT EXISTS idx_walk_report_photos_tenant ON walk_report_photos(tenant_id);

-- Add tenant_id to user_colors table
ALTER TABLE user_colors ADD COLUMN tenant_id INTEGER REFERENCES tenants(id);
CREATE INDEX IF NOT EXISTS idx_user_colors_tenant ON user_colors(tenant_id);

-- Add tenant_id to color_requests table
ALTER TABLE color_requests ADD COLUMN tenant_id INTEGER REFERENCES tenants(id);
CREATE INDEX IF NOT EXISTS idx_color_requests_tenant ON color_requests(tenant_id);

-- Note: feiertage_cache is global (shared across tenants), no tenant_id needed
`,
			"mysql": `
-- Add tenant_id to users table
ALTER TABLE users ADD COLUMN tenant_id INT NULL;
ALTER TABLE users ADD INDEX idx_users_tenant (tenant_id);
ALTER TABLE users ADD CONSTRAINT fk_users_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- Add tenant_id to color_categories table
ALTER TABLE color_categories ADD COLUMN tenant_id INT NULL;
ALTER TABLE color_categories ADD INDEX idx_color_categories_tenant (tenant_id);
ALTER TABLE color_categories ADD CONSTRAINT fk_color_categories_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- Add tenant_id to dogs table
ALTER TABLE dogs ADD COLUMN tenant_id INT NULL;
ALTER TABLE dogs ADD INDEX idx_dogs_tenant (tenant_id);
ALTER TABLE dogs ADD CONSTRAINT fk_dogs_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- Add tenant_id to bookings table
ALTER TABLE bookings ADD COLUMN tenant_id INT NULL;
ALTER TABLE bookings ADD INDEX idx_bookings_tenant (tenant_id);
ALTER TABLE bookings ADD CONSTRAINT fk_bookings_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- Add tenant_id to blocked_dates table
ALTER TABLE blocked_dates ADD COLUMN tenant_id INT NULL;
ALTER TABLE blocked_dates ADD INDEX idx_blocked_dates_tenant (tenant_id);
ALTER TABLE blocked_dates ADD CONSTRAINT fk_blocked_dates_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- Add tenant_id to experience_requests table
ALTER TABLE experience_requests ADD COLUMN tenant_id INT NULL;
ALTER TABLE experience_requests ADD INDEX idx_experience_requests_tenant (tenant_id);
ALTER TABLE experience_requests ADD CONSTRAINT fk_experience_requests_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- Add tenant_id to system_settings table
ALTER TABLE system_settings ADD COLUMN tenant_id INT NULL;
ALTER TABLE system_settings ADD CONSTRAINT fk_system_settings_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- Add tenant_id to reactivation_requests table
ALTER TABLE reactivation_requests ADD COLUMN tenant_id INT NULL;
ALTER TABLE reactivation_requests ADD INDEX idx_reactivation_requests_tenant (tenant_id);
ALTER TABLE reactivation_requests ADD CONSTRAINT fk_reactivation_requests_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- Add tenant_id to booking_time_rules table
ALTER TABLE booking_time_rules ADD COLUMN tenant_id INT NULL;
ALTER TABLE booking_time_rules ADD INDEX idx_booking_time_rules_tenant (tenant_id);
ALTER TABLE booking_time_rules ADD CONSTRAINT fk_booking_time_rules_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- Add tenant_id to custom_holidays table
ALTER TABLE custom_holidays ADD COLUMN tenant_id INT NULL;
ALTER TABLE custom_holidays ADD INDEX idx_custom_holidays_tenant (tenant_id);
ALTER TABLE custom_holidays ADD CONSTRAINT fk_custom_holidays_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- Add tenant_id to walk_reports table
ALTER TABLE walk_reports ADD COLUMN tenant_id INT NULL;
ALTER TABLE walk_reports ADD INDEX idx_walk_reports_tenant (tenant_id);
ALTER TABLE walk_reports ADD CONSTRAINT fk_walk_reports_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- Add tenant_id to walk_report_photos table
ALTER TABLE walk_report_photos ADD COLUMN tenant_id INT NULL;
ALTER TABLE walk_report_photos ADD INDEX idx_walk_report_photos_tenant (tenant_id);
ALTER TABLE walk_report_photos ADD CONSTRAINT fk_walk_report_photos_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- Add tenant_id to user_colors table
ALTER TABLE user_colors ADD COLUMN tenant_id INT NULL;
ALTER TABLE user_colors ADD INDEX idx_user_colors_tenant (tenant_id);
ALTER TABLE user_colors ADD CONSTRAINT fk_user_colors_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- Add tenant_id to color_requests table
ALTER TABLE color_requests ADD COLUMN tenant_id INT NULL;
ALTER TABLE color_requests ADD INDEX idx_color_requests_tenant (tenant_id);
ALTER TABLE color_requests ADD CONSTRAINT fk_color_requests_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- Note: feiertage_cache is global (shared across tenants), no tenant_id needed
`,
			"postgres": `
-- Add tenant_id to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_users_tenant ON users(tenant_id);

-- Add tenant_id to color_categories table
ALTER TABLE color_categories ADD COLUMN IF NOT EXISTS tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_color_categories_tenant ON color_categories(tenant_id);

-- Add tenant_id to dogs table
ALTER TABLE dogs ADD COLUMN IF NOT EXISTS tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_dogs_tenant ON dogs(tenant_id);

-- Add tenant_id to bookings table
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_bookings_tenant ON bookings(tenant_id);

-- Add tenant_id to blocked_dates table
ALTER TABLE blocked_dates ADD COLUMN IF NOT EXISTS tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_blocked_dates_tenant ON blocked_dates(tenant_id);

-- Add tenant_id to experience_requests table
ALTER TABLE experience_requests ADD COLUMN IF NOT EXISTS tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_experience_requests_tenant ON experience_requests(tenant_id);

-- Add tenant_id to system_settings table
ALTER TABLE system_settings ADD COLUMN IF NOT EXISTS tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE;

-- Add tenant_id to reactivation_requests table
ALTER TABLE reactivation_requests ADD COLUMN IF NOT EXISTS tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_reactivation_requests_tenant ON reactivation_requests(tenant_id);

-- Add tenant_id to booking_time_rules table
ALTER TABLE booking_time_rules ADD COLUMN IF NOT EXISTS tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_booking_time_rules_tenant ON booking_time_rules(tenant_id);

-- Add tenant_id to custom_holidays table
ALTER TABLE custom_holidays ADD COLUMN IF NOT EXISTS tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_custom_holidays_tenant ON custom_holidays(tenant_id);

-- Add tenant_id to walk_reports table
ALTER TABLE walk_reports ADD COLUMN IF NOT EXISTS tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_walk_reports_tenant ON walk_reports(tenant_id);

-- Add tenant_id to walk_report_photos table
ALTER TABLE walk_report_photos ADD COLUMN IF NOT EXISTS tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_walk_report_photos_tenant ON walk_report_photos(tenant_id);

-- Add tenant_id to user_colors table
ALTER TABLE user_colors ADD COLUMN IF NOT EXISTS tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_user_colors_tenant ON user_colors(tenant_id);

-- Add tenant_id to color_requests table
ALTER TABLE color_requests ADD COLUMN IF NOT EXISTS tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_color_requests_tenant ON color_requests(tenant_id);

-- Note: feiertage_cache is global (shared across tenants), no tenant_id needed
`,
		},
	})
}
