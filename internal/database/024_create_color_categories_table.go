package database

func init() {
	RegisterMigration(&Migration{
		ID:          "024_create_color_categories_table",
		Description: "Create color_categories table for configurable dog color categories",
		Up: map[string]string{
			"sqlite": `
CREATE TABLE IF NOT EXISTS color_categories (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  hex_code TEXT NOT NULL,
  pattern_icon TEXT,
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_color_categories_sort ON color_categories(sort_order);

INSERT OR IGNORE INTO color_categories (name, hex_code, pattern_icon, sort_order) VALUES
  ('gruen', '#28a745', 'circle', 1),
  ('gelb', '#ffc107', 'triangle', 2),
  ('orange', '#fd7e14', 'square', 3),
  ('hellblau', '#17a2b8', 'diamond', 4),
  ('dunkelblau', '#007bff', 'pentagon', 5),
  ('helllila', '#e83e8c', 'hexagon', 6),
  ('dunkellila', '#6f42c1', 'star', 7);
`,
			"mysql": `
CREATE TABLE IF NOT EXISTS color_categories (
  id INT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(100) NOT NULL UNIQUE,
  hex_code VARCHAR(20) NOT NULL,
  pattern_icon VARCHAR(50),
  sort_order INT NOT NULL DEFAULT 0,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_color_categories_sort (sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO color_categories (name, hex_code, pattern_icon, sort_order) VALUES
  ('gruen', '#28a745', 'circle', 1),
  ('gelb', '#ffc107', 'triangle', 2),
  ('orange', '#fd7e14', 'square', 3),
  ('hellblau', '#17a2b8', 'diamond', 4),
  ('dunkelblau', '#007bff', 'pentagon', 5),
  ('helllila', '#e83e8c', 'hexagon', 6),
  ('dunkellila', '#6f42c1', 'star', 7);
`,
			"postgres": `
CREATE TABLE IF NOT EXISTS color_categories (
  id SERIAL PRIMARY KEY,
  name VARCHAR(100) NOT NULL UNIQUE,
  hex_code VARCHAR(20) NOT NULL,
  pattern_icon VARCHAR(50),
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_color_categories_sort ON color_categories(sort_order);

INSERT INTO color_categories (name, hex_code, pattern_icon, sort_order) VALUES
  ('gruen', '#28a745', 'circle', 1),
  ('gelb', '#ffc107', 'triangle', 2),
  ('orange', '#fd7e14', 'square', 3),
  ('hellblau', '#17a2b8', 'diamond', 4),
  ('dunkelblau', '#007bff', 'pentagon', 5),
  ('helllila', '#e83e8c', 'hexagon', 6),
  ('dunkellila', '#6f42c1', 'star', 7)
ON CONFLICT (name) DO NOTHING;
`,
		},
	})
}
