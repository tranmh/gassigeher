package database

func init() {
	RegisterMigration(&Migration{
		ID:          "017_insert_site_logo_setting",
		Description: "Insert default site logo setting for configurable logo banner",
		Up: map[string]string{
			"sqlite": `
INSERT OR IGNORE INTO system_settings (key, value) VALUES
  ('site_logo', 'https://www.tierheim-goeppingen.de/wp-content/uploads/2017/04/Logo_4c_homepagebanner3.png');
`,
			"mysql": "INSERT IGNORE INTO system_settings (`key`, value) VALUES\n" +
				"  ('site_logo', 'https://www.tierheim-goeppingen.de/wp-content/uploads/2017/04/Logo_4c_homepagebanner3.png');",
			"postgres": `
INSERT INTO system_settings (key, value) VALUES
  ('site_logo', 'https://www.tierheim-goeppingen.de/wp-content/uploads/2017/04/Logo_4c_homepagebanner3.png')
ON CONFLICT (key) DO NOTHING;
`,
		},
	})
}
