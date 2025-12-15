package database

func init() {
	RegisterMigration(&Migration{
		ID:          "026_add_color_id_to_dogs",
		Description: "Add color_id foreign key to dogs table and migrate existing category data",
		Up: map[string]string{
			"sqlite": `
ALTER TABLE dogs ADD COLUMN color_id INTEGER REFERENCES color_categories(id);

UPDATE dogs SET color_id = (
  SELECT id FROM color_categories WHERE name =
    CASE dogs.category
      WHEN 'green' THEN 'gruen'
      WHEN 'orange' THEN 'orange'
      WHEN 'blue' THEN 'dunkelblau'
    END
) WHERE color_id IS NULL AND category IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_dogs_color ON dogs(color_id);
`,
			"mysql": `
ALTER TABLE dogs ADD COLUMN color_id INT,
ADD CONSTRAINT fk_dogs_color FOREIGN KEY (color_id) REFERENCES color_categories(id);

UPDATE dogs SET color_id = (
  SELECT id FROM color_categories WHERE name =
    CASE dogs.category
      WHEN 'green' THEN 'gruen'
      WHEN 'orange' THEN 'orange'
      WHEN 'blue' THEN 'dunkelblau'
    END
) WHERE color_id IS NULL AND category IS NOT NULL;

CREATE INDEX idx_dogs_color ON dogs(color_id);
`,
			"postgres": `
ALTER TABLE dogs ADD COLUMN color_id INTEGER REFERENCES color_categories(id);

UPDATE dogs SET color_id = (
  SELECT id FROM color_categories WHERE name =
    CASE dogs.category
      WHEN 'green' THEN 'gruen'
      WHEN 'orange' THEN 'orange'
      WHEN 'blue' THEN 'dunkelblau'
    END
) WHERE color_id IS NULL AND category IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_dogs_color ON dogs(color_id);
`,
		},
	})
}
