package database

func init() {
	RegisterMigration(&Migration{
		ID:          "028_add_default_color_setting",
		Description: "Add default_color_for_new_users system setting",
		Up: map[string]string{
			"sqlite": `
INSERT OR IGNORE INTO system_settings (key, value, updated_at)
VALUES ('default_color_for_new_users', '', CURRENT_TIMESTAMP);
`,
			"mysql": `
INSERT IGNORE INTO system_settings (` + "`key`" + `, value, updated_at)
VALUES ('default_color_for_new_users', '', CURRENT_TIMESTAMP);
`,
			"postgres": `
INSERT INTO system_settings (key, value, updated_at)
VALUES ('default_color_for_new_users', '', CURRENT_TIMESTAMP)
ON CONFLICT (key) DO NOTHING;
`,
		},
	})
}
