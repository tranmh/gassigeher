package database

func init() {
	RegisterMigration(&Migration{
		ID:          "021_insert_whatsapp_settings",
		Description: "Insert WhatsApp group settings (enabled flag and group link)",
		Up: map[string]string{
			"sqlite": `
INSERT OR IGNORE INTO system_settings (key, value) VALUES
  ('whatsapp_group_enabled', 'false'),
  ('whatsapp_group_link', '');
`,
			"mysql": "INSERT IGNORE INTO system_settings (`key`, value) VALUES\n" +
				"  ('whatsapp_group_enabled', 'false'),\n" +
				"  ('whatsapp_group_link', '');",
			"postgres": `
INSERT INTO system_settings (key, value) VALUES
  ('whatsapp_group_enabled', 'false'),
  ('whatsapp_group_link', '')
ON CONFLICT (key) DO NOTHING;
`,
		},
	})
}
