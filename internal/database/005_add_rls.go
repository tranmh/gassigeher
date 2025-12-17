package database

func init() {
	RegisterMigration(&Migration{
		ID:          "005_add_rls",
		Description: "Add PostgreSQL Row Level Security policies for tenant isolation",
		Up: map[string]string{
			// SQLite doesn't support RLS - tenant isolation must be enforced at application level
			"sqlite": `
-- SQLite does not support Row Level Security
-- Tenant isolation is enforced at the application/repository layer
-- This migration is a no-op for SQLite
SELECT 1;
`,
			// MySQL doesn't support RLS - tenant isolation must be enforced at application level
			"mysql": `
-- MySQL does not support Row Level Security
-- Tenant isolation is enforced at the application/repository layer
-- This migration is a no-op for MySQL
SELECT 1;
`,
			// PostgreSQL RLS provides database-level tenant isolation
			"postgres": `
-- Enable Row Level Security on all tenant-scoped tables
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE color_categories ENABLE ROW LEVEL SECURITY;
ALTER TABLE dogs ENABLE ROW LEVEL SECURITY;
ALTER TABLE bookings ENABLE ROW LEVEL SECURITY;
ALTER TABLE blocked_dates ENABLE ROW LEVEL SECURITY;
ALTER TABLE experience_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE system_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE reactivation_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE booking_time_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE custom_holidays ENABLE ROW LEVEL SECURITY;
ALTER TABLE walk_reports ENABLE ROW LEVEL SECURITY;
ALTER TABLE walk_report_photos ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_colors ENABLE ROW LEVEL SECURITY;
ALTER TABLE color_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_settings ENABLE ROW LEVEL SECURITY;

-- Create RLS policies for each table
-- Policy: Users can only access rows where tenant_id matches app.tenant_id session variable

-- Users policy
DROP POLICY IF EXISTS tenant_isolation_users ON users;
CREATE POLICY tenant_isolation_users ON users
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::INTEGER OR tenant_id IS NULL);

-- Color categories policy
DROP POLICY IF EXISTS tenant_isolation_color_categories ON color_categories;
CREATE POLICY tenant_isolation_color_categories ON color_categories
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::INTEGER OR tenant_id IS NULL);

-- Dogs policy
DROP POLICY IF EXISTS tenant_isolation_dogs ON dogs;
CREATE POLICY tenant_isolation_dogs ON dogs
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::INTEGER OR tenant_id IS NULL);

-- Bookings policy
DROP POLICY IF EXISTS tenant_isolation_bookings ON bookings;
CREATE POLICY tenant_isolation_bookings ON bookings
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::INTEGER OR tenant_id IS NULL);

-- Blocked dates policy
DROP POLICY IF EXISTS tenant_isolation_blocked_dates ON blocked_dates;
CREATE POLICY tenant_isolation_blocked_dates ON blocked_dates
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::INTEGER OR tenant_id IS NULL);

-- Experience requests policy
DROP POLICY IF EXISTS tenant_isolation_experience_requests ON experience_requests;
CREATE POLICY tenant_isolation_experience_requests ON experience_requests
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::INTEGER OR tenant_id IS NULL);

-- System settings policy
DROP POLICY IF EXISTS tenant_isolation_system_settings ON system_settings;
CREATE POLICY tenant_isolation_system_settings ON system_settings
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::INTEGER OR tenant_id IS NULL);

-- Reactivation requests policy
DROP POLICY IF EXISTS tenant_isolation_reactivation_requests ON reactivation_requests;
CREATE POLICY tenant_isolation_reactivation_requests ON reactivation_requests
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::INTEGER OR tenant_id IS NULL);

-- Booking time rules policy
DROP POLICY IF EXISTS tenant_isolation_booking_time_rules ON booking_time_rules;
CREATE POLICY tenant_isolation_booking_time_rules ON booking_time_rules
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::INTEGER OR tenant_id IS NULL);

-- Custom holidays policy
DROP POLICY IF EXISTS tenant_isolation_custom_holidays ON custom_holidays;
CREATE POLICY tenant_isolation_custom_holidays ON custom_holidays
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::INTEGER OR tenant_id IS NULL);

-- Walk reports policy
DROP POLICY IF EXISTS tenant_isolation_walk_reports ON walk_reports;
CREATE POLICY tenant_isolation_walk_reports ON walk_reports
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::INTEGER OR tenant_id IS NULL);

-- Walk report photos policy
DROP POLICY IF EXISTS tenant_isolation_walk_report_photos ON walk_report_photos;
CREATE POLICY tenant_isolation_walk_report_photos ON walk_report_photos
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::INTEGER OR tenant_id IS NULL);

-- User colors policy
DROP POLICY IF EXISTS tenant_isolation_user_colors ON user_colors;
CREATE POLICY tenant_isolation_user_colors ON user_colors
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::INTEGER OR tenant_id IS NULL);

-- Color requests policy
DROP POLICY IF EXISTS tenant_isolation_color_requests ON color_requests;
CREATE POLICY tenant_isolation_color_requests ON color_requests
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::INTEGER OR tenant_id IS NULL);

-- Tenant settings policy (tenant can only see their own settings)
DROP POLICY IF EXISTS tenant_isolation_tenant_settings ON tenant_settings;
CREATE POLICY tenant_isolation_tenant_settings ON tenant_settings
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::INTEGER);

-- Note: The application must set the session variable before queries:
-- SET LOCAL app.tenant_id = '123';
-- This is done in the repository layer via SetTenantSession()

-- Note: feiertage_cache does not have RLS as it's shared globally across tenants
`,
		},
	})
}
