package database

func init() {
	RegisterMigration(&Migration{
		ID:          "022_create_walk_reports_table",
		Description: "Create walk_reports and walk_report_photos tables for walk feedback system",
		Up: map[string]string{
			"sqlite": `
CREATE TABLE IF NOT EXISTS walk_reports (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  booking_id INTEGER NOT NULL UNIQUE,
  behavior_rating INTEGER NOT NULL CHECK(behavior_rating >= 1 AND behavior_rating <= 5),
  energy_level TEXT NOT NULL CHECK(energy_level IN ('low', 'medium', 'high')),
  notes TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (booking_id) REFERENCES bookings(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS walk_report_photos (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  walk_report_id INTEGER NOT NULL,
  photo_path TEXT NOT NULL,
  photo_thumbnail TEXT NOT NULL,
  display_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (walk_report_id) REFERENCES walk_reports(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_walk_reports_booking_id ON walk_reports(booking_id);
CREATE INDEX IF NOT EXISTS idx_walk_report_photos_report_id ON walk_report_photos(walk_report_id);
`,
			"mysql": `
CREATE TABLE IF NOT EXISTS walk_reports (
  id INT AUTO_INCREMENT PRIMARY KEY,
  booking_id INT NOT NULL UNIQUE,
  behavior_rating INT NOT NULL CHECK(behavior_rating >= 1 AND behavior_rating <= 5),
  energy_level VARCHAR(10) NOT NULL CHECK(energy_level IN ('low', 'medium', 'high')),
  notes TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (booking_id) REFERENCES bookings(id) ON DELETE CASCADE,
  INDEX idx_walk_reports_booking_id (booking_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS walk_report_photos (
  id INT AUTO_INCREMENT PRIMARY KEY,
  walk_report_id INT NOT NULL,
  photo_path VARCHAR(255) NOT NULL,
  photo_thumbnail VARCHAR(255) NOT NULL,
  display_order INT NOT NULL DEFAULT 0,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (walk_report_id) REFERENCES walk_reports(id) ON DELETE CASCADE,
  INDEX idx_walk_report_photos_report_id (walk_report_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`,
			"postgres": `
CREATE TABLE IF NOT EXISTS walk_reports (
  id SERIAL PRIMARY KEY,
  booking_id INTEGER NOT NULL UNIQUE,
  behavior_rating INTEGER NOT NULL CHECK(behavior_rating >= 1 AND behavior_rating <= 5),
  energy_level VARCHAR(10) NOT NULL CHECK(energy_level IN ('low', 'medium', 'high')),
  notes TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (booking_id) REFERENCES bookings(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS walk_report_photos (
  id SERIAL PRIMARY KEY,
  walk_report_id INTEGER NOT NULL,
  photo_path VARCHAR(255) NOT NULL,
  photo_thumbnail VARCHAR(255) NOT NULL,
  display_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (walk_report_id) REFERENCES walk_reports(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_walk_reports_booking_id ON walk_reports(booking_id);
CREATE INDEX IF NOT EXISTS idx_walk_report_photos_report_id ON walk_report_photos(walk_report_id);
`,
		},
	})
}
