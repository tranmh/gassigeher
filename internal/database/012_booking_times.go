package database

func init() {
	RegisterMigration(&Migration{
		ID:          "012_booking_times",
		Description: "Add booking time restrictions with time rules, holidays, and approval workflow",
		Up: map[string]string{
			"sqlite": `
-- Create booking_time_rules table
CREATE TABLE IF NOT EXISTS booking_time_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    day_type TEXT NOT NULL,
    rule_name TEXT NOT NULL,
    start_time TEXT NOT NULL,
    end_time TEXT NOT NULL,
    is_blocked INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(day_type, rule_name)
);

-- Create custom_holidays table
CREATE TABLE IF NOT EXISTS custom_holidays (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    is_active INTEGER NOT NULL DEFAULT 1,
    source TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by INTEGER,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_custom_holidays_date ON custom_holidays(date);
CREATE INDEX IF NOT EXISTS idx_custom_holidays_active ON custom_holidays(is_active);

-- Create feiertage_cache table
CREATE TABLE IF NOT EXISTS feiertage_cache (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    year INTEGER NOT NULL UNIQUE,
    state TEXT NOT NULL,
    data TEXT NOT NULL,
    fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL
);

-- Add approval columns to bookings (recreate table for SQLite)
CREATE TABLE IF NOT EXISTS bookings_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    dog_id INTEGER NOT NULL,
    date DATE NOT NULL,
    scheduled_time TEXT NOT NULL,
    walk_type TEXT CHECK(walk_type IN ('morning', 'evening')),
    status TEXT DEFAULT 'scheduled' CHECK(status IN ('scheduled', 'completed', 'cancelled')),
    completed_at TIMESTAMP,
    user_notes TEXT,
    admin_cancellation_reason TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    requires_approval INTEGER DEFAULT 0,
    approval_status TEXT DEFAULT 'approved',
    approved_by INTEGER,
    approved_at TIMESTAMP,
    rejection_reason TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (dog_id) REFERENCES dogs(id) ON DELETE CASCADE,
    FOREIGN KEY (approved_by) REFERENCES users(id) ON DELETE SET NULL,
    UNIQUE(dog_id, date, walk_type)
);

INSERT INTO bookings_new SELECT
    id, user_id, dog_id, date, scheduled_time, walk_type, status,
    completed_at, user_notes, admin_cancellation_reason, created_at, updated_at,
    0, 'approved', NULL, NULL, NULL
FROM bookings;

DROP TABLE bookings;
ALTER TABLE bookings_new RENAME TO bookings;

CREATE INDEX IF NOT EXISTS idx_bookings_user ON bookings(user_id);
CREATE INDEX IF NOT EXISTS idx_bookings_dog ON bookings(dog_id);
CREATE INDEX IF NOT EXISTS idx_bookings_date ON bookings(date);
CREATE INDEX IF NOT EXISTS idx_bookings_status ON bookings(status);
CREATE INDEX IF NOT EXISTS idx_bookings_approval_status ON bookings(approval_status);

-- Seed default time rules
INSERT OR IGNORE INTO booking_time_rules (day_type, rule_name, start_time, end_time, is_blocked) VALUES
('weekday', 'Morgenspaziergang', '09:00', '12:00', 0),
('weekday', 'Mittagspause', '13:00', '14:00', 1),
('weekday', 'Nachmittagsspaziergang', '14:00', '16:30', 0),
('weekday', 'Fütterungszeit', '16:30', '18:00', 1),
('weekday', 'Abendspaziergang', '18:00', '19:30', 0),
('weekend', 'Morgenspaziergang', '09:00', '12:00', 0),
('weekend', 'Fütterungszeit', '12:00', '13:00', 1),
('weekend', 'Mittagspause', '13:00', '14:00', 1),
('weekend', 'Nachmittagsspaziergang', '14:00', '17:00', 0);

-- Add new system settings
INSERT OR IGNORE INTO system_settings (key, value) VALUES
('morning_walk_requires_approval', 'true'),
('use_feiertage_api', 'true'),
('feiertage_state', 'BW'),
('booking_time_granularity', '15'),
('feiertage_cache_days', '7');
`,
			"mysql": `
-- Create booking_time_rules table
CREATE TABLE IF NOT EXISTS booking_time_rules (
    id INT AUTO_INCREMENT PRIMARY KEY,
    day_type VARCHAR(20) NOT NULL,
    rule_name VARCHAR(100) NOT NULL,
    start_time VARCHAR(10) NOT NULL,
    end_time VARCHAR(10) NOT NULL,
    is_blocked TINYINT(1) NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY unique_day_rule (day_type, rule_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Create custom_holidays table
CREATE TABLE IF NOT EXISTS custom_holidays (
    id INT AUTO_INCREMENT PRIMARY KEY,
    date DATE NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    is_active TINYINT(1) NOT NULL DEFAULT 1,
    source VARCHAR(20) NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    created_by INT,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
    INDEX idx_custom_holidays_date (date),
    INDEX idx_custom_holidays_active (is_active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Create feiertage_cache table
CREATE TABLE IF NOT EXISTS feiertage_cache (
    id INT AUTO_INCREMENT PRIMARY KEY,
    year INT NOT NULL UNIQUE,
    state VARCHAR(10) NOT NULL,
    data TEXT NOT NULL,
    fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Add approval columns to bookings
ALTER TABLE bookings
ADD COLUMN requires_approval TINYINT(1) DEFAULT 0,
ADD COLUMN approval_status VARCHAR(20) DEFAULT 'approved',
ADD COLUMN approved_by INT,
ADD COLUMN approved_at DATETIME,
ADD COLUMN rejection_reason TEXT,
ADD FOREIGN KEY (approved_by) REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX idx_bookings_approval_status ON bookings(approval_status);

-- Seed default time rules
INSERT IGNORE INTO booking_time_rules (day_type, rule_name, start_time, end_time, is_blocked) VALUES
('weekday', 'Morgenspaziergang', '09:00', '12:00', 0),
('weekday', 'Mittagspause', '13:00', '14:00', 1),
('weekday', 'Nachmittagsspaziergang', '14:00', '16:30', 0),
('weekday', 'Fütterungszeit', '16:30', '18:00', 1),
('weekday', 'Abendspaziergang', '18:00', '19:30', 0),
('weekend', 'Morgenspaziergang', '09:00', '12:00', 0),
('weekend', 'Fütterungszeit', '12:00', '13:00', 1),
('weekend', 'Mittagspause', '13:00', '14:00', 1),
('weekend', 'Nachmittagsspaziergang', '14:00', '17:00', 0);

-- Add new system settings
INSERT IGNORE INTO system_settings (key, value) VALUES
('morning_walk_requires_approval', 'true'),
('use_feiertage_api', 'true'),
('feiertage_state', 'BW'),
('booking_time_granularity', '15'),
('feiertage_cache_days', '7');
`,
			"postgres": `
-- Create booking_time_rules table
CREATE TABLE IF NOT EXISTS booking_time_rules (
    id SERIAL PRIMARY KEY,
    day_type VARCHAR(20) NOT NULL,
    rule_name VARCHAR(100) NOT NULL,
    start_time VARCHAR(10) NOT NULL,
    end_time VARCHAR(10) NOT NULL,
    is_blocked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(day_type, rule_name)
);

-- Create custom_holidays table
CREATE TABLE IF NOT EXISTS custom_holidays (
    id SERIAL PRIMARY KEY,
    date DATE NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    source VARCHAR(20) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by INTEGER,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_custom_holidays_date ON custom_holidays(date);
CREATE INDEX IF NOT EXISTS idx_custom_holidays_active ON custom_holidays(is_active);

-- Create feiertage_cache table
CREATE TABLE IF NOT EXISTS feiertage_cache (
    id SERIAL PRIMARY KEY,
    year INTEGER NOT NULL UNIQUE,
    state VARCHAR(10) NOT NULL,
    data TEXT NOT NULL,
    fetched_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Add approval columns to bookings
ALTER TABLE bookings
ADD COLUMN IF NOT EXISTS requires_approval BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS approval_status VARCHAR(20) DEFAULT 'approved',
ADD COLUMN IF NOT EXISTS approved_by INTEGER,
ADD COLUMN IF NOT EXISTS approved_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS rejection_reason TEXT;

-- Add foreign key constraint
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'bookings_approved_by_fkey'
    ) THEN
        ALTER TABLE bookings ADD CONSTRAINT bookings_approved_by_fkey
        FOREIGN KEY (approved_by) REFERENCES users(id) ON DELETE SET NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_bookings_approval_status ON bookings(approval_status);

-- Seed default time rules
INSERT INTO booking_time_rules (day_type, rule_name, start_time, end_time, is_blocked) VALUES
('weekday', 'Morgenspaziergang', '09:00', '12:00', FALSE),
('weekday', 'Mittagspause', '13:00', '14:00', TRUE),
('weekday', 'Nachmittagsspaziergang', '14:00', '16:30', FALSE),
('weekday', 'Fütterungszeit', '16:30', '18:00', TRUE),
('weekday', 'Abendspaziergang', '18:00', '19:30', FALSE),
('weekend', 'Morgenspaziergang', '09:00', '12:00', FALSE),
('weekend', 'Fütterungszeit', '12:00', '13:00', TRUE),
('weekend', 'Mittagspause', '13:00', '14:00', TRUE),
('weekend', 'Nachmittagsspaziergang', '14:00', '17:00', FALSE)
ON CONFLICT (day_type, rule_name) DO NOTHING;

-- Add new system settings
INSERT INTO system_settings (key, value) VALUES
('morning_walk_requires_approval', 'true'),
('use_feiertage_api', 'true'),
('feiertage_state', 'BW'),
('booking_time_granularity', '15'),
('feiertage_cache_days', '7')
ON CONFLICT (key) DO NOTHING;
`,
		},
	})
}

// DONE
