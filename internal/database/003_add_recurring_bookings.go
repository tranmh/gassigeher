package database

func init() {
	RegisterMigration(&Migration{
		ID:          "003_add_recurring_bookings",
		Description: "Add recurring booking series support with linked bookings",
		Up: map[string]string{
			"sqlite": `
-- Recurring booking series table (per-tenant)
CREATE TABLE IF NOT EXISTS recurring_booking_series (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id INTEGER NOT NULL DEFAULT 0 REFERENCES tenants(id) ON DELETE CASCADE,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  dog_id INTEGER NOT NULL REFERENCES dogs(id) ON DELETE CASCADE,
  recurrence_type TEXT NOT NULL CHECK(recurrence_type IN ('weekly', 'interval')),
  day_of_week INTEGER CHECK(day_of_week >= 0 AND day_of_week <= 6),
  interval_days INTEGER CHECK(interval_days > 0),
  scheduled_time TEXT NOT NULL,
  start_date TEXT NOT NULL,
  end_date TEXT NOT NULL,
  status TEXT DEFAULT 'active' CHECK(status IN ('active', 'cancelled', 'completed')),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_recurring_series_tenant ON recurring_booking_series(tenant_id);
CREATE INDEX IF NOT EXISTS idx_recurring_series_user ON recurring_booking_series(user_id);
CREATE INDEX IF NOT EXISTS idx_recurring_series_dog ON recurring_booking_series(dog_id);
CREATE INDEX IF NOT EXISTS idx_recurring_series_status ON recurring_booking_series(status);

-- Add recurrence_id column to bookings table
ALTER TABLE bookings ADD COLUMN recurrence_id INTEGER REFERENCES recurring_booking_series(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_bookings_recurrence ON bookings(recurrence_id);

-- Seed default recurring booking settings
INSERT OR IGNORE INTO system_settings (tenant_id, key, value) VALUES (0, 'recurring_booking_max_weeks', '8');
INSERT OR IGNORE INTO system_settings (tenant_id, key, value) VALUES (0, 'max_active_recurring_series', '3');
`,
			"postgres": `
-- Recurring booking series table (per-tenant)
CREATE TABLE IF NOT EXISTS recurring_booking_series (
  id SERIAL PRIMARY KEY,
  tenant_id INTEGER NOT NULL DEFAULT 0 REFERENCES tenants(id) ON DELETE CASCADE,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  dog_id INTEGER NOT NULL REFERENCES dogs(id) ON DELETE CASCADE,
  recurrence_type VARCHAR(20) NOT NULL CHECK(recurrence_type IN ('weekly', 'interval')),
  day_of_week INTEGER CHECK(day_of_week >= 0 AND day_of_week <= 6),
  interval_days INTEGER CHECK(interval_days > 0),
  scheduled_time VARCHAR(10) NOT NULL,
  start_date DATE NOT NULL,
  end_date DATE NOT NULL,
  status VARCHAR(20) DEFAULT 'active' CHECK(status IN ('active', 'cancelled', 'completed')),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_recurring_series_tenant ON recurring_booking_series(tenant_id);
CREATE INDEX IF NOT EXISTS idx_recurring_series_user ON recurring_booking_series(user_id);
CREATE INDEX IF NOT EXISTS idx_recurring_series_dog ON recurring_booking_series(dog_id);
CREATE INDEX IF NOT EXISTS idx_recurring_series_status ON recurring_booking_series(status);

-- Add recurrence_id column to bookings table
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS recurrence_id INTEGER REFERENCES recurring_booking_series(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_bookings_recurrence ON bookings(recurrence_id);

-- Seed default recurring booking settings
INSERT INTO system_settings (tenant_id, key, value) VALUES (0, 'recurring_booking_max_weeks', '8') ON CONFLICT DO NOTHING;
INSERT INTO system_settings (tenant_id, key, value) VALUES (0, 'max_active_recurring_series', '3') ON CONFLICT DO NOTHING;
`,
		},
	})
}
