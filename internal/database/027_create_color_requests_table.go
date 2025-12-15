package database

func init() {
	RegisterMigration(&Migration{
		ID:          "027_create_color_requests_table",
		Description: "Create color_requests table for user color category requests",
		Up: map[string]string{
			"sqlite": `
CREATE TABLE IF NOT EXISTS color_requests (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  color_id INTEGER NOT NULL,
  status TEXT DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'denied')),
  admin_message TEXT,
  reviewed_by INTEGER,
  reviewed_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (color_id) REFERENCES color_categories(id),
  FOREIGN KEY (reviewed_by) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_color_requests_user ON color_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_color_requests_status ON color_requests(status);
CREATE INDEX IF NOT EXISTS idx_color_requests_color ON color_requests(color_id);
`,
			"mysql": `
CREATE TABLE IF NOT EXISTS color_requests (
  id INT AUTO_INCREMENT PRIMARY KEY,
  user_id INT NOT NULL,
  color_id INT NOT NULL,
  status VARCHAR(20) DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'denied')),
  admin_message TEXT,
  reviewed_by INT,
  reviewed_at DATETIME,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (color_id) REFERENCES color_categories(id),
  FOREIGN KEY (reviewed_by) REFERENCES users(id),
  INDEX idx_color_requests_user (user_id),
  INDEX idx_color_requests_status (status),
  INDEX idx_color_requests_color (color_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`,
			"postgres": `
CREATE TABLE IF NOT EXISTS color_requests (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL,
  color_id INTEGER NOT NULL,
  status VARCHAR(20) DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'denied')),
  admin_message TEXT,
  reviewed_by INTEGER,
  reviewed_at TIMESTAMP WITH TIME ZONE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (color_id) REFERENCES color_categories(id),
  FOREIGN KEY (reviewed_by) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_color_requests_user ON color_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_color_requests_status ON color_requests(status);
CREATE INDEX IF NOT EXISTS idx_color_requests_color ON color_requests(color_id);
`,
		},
	})
}
