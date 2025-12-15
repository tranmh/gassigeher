package database

func init() {
	RegisterMigration(&Migration{
		ID:          "025_create_user_colors_table",
		Description: "Create user_colors junction table for many-to-many user-color relationship",
		Up: map[string]string{
			"sqlite": `
CREATE TABLE IF NOT EXISTS user_colors (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  color_id INTEGER NOT NULL,
  granted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  granted_by INTEGER,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (color_id) REFERENCES color_categories(id) ON DELETE RESTRICT,
  FOREIGN KEY (granted_by) REFERENCES users(id),
  UNIQUE(user_id, color_id)
);

CREATE INDEX IF NOT EXISTS idx_user_colors_user ON user_colors(user_id);
CREATE INDEX IF NOT EXISTS idx_user_colors_color ON user_colors(color_id);
`,
			"mysql": `
CREATE TABLE IF NOT EXISTS user_colors (
  id INT AUTO_INCREMENT PRIMARY KEY,
  user_id INT NOT NULL,
  color_id INT NOT NULL,
  granted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  granted_by INT,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (color_id) REFERENCES color_categories(id) ON DELETE RESTRICT,
  FOREIGN KEY (granted_by) REFERENCES users(id),
  UNIQUE KEY unique_user_color (user_id, color_id),
  INDEX idx_user_colors_user (user_id),
  INDEX idx_user_colors_color (color_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`,
			"postgres": `
CREATE TABLE IF NOT EXISTS user_colors (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL,
  color_id INTEGER NOT NULL,
  granted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  granted_by INTEGER,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (color_id) REFERENCES color_categories(id) ON DELETE RESTRICT,
  FOREIGN KEY (granted_by) REFERENCES users(id),
  UNIQUE(user_id, color_id)
);

CREATE INDEX IF NOT EXISTS idx_user_colors_user ON user_colors(user_id);
CREATE INDEX IF NOT EXISTS idx_user_colors_color ON user_colors(color_id);
`,
		},
	})
}
