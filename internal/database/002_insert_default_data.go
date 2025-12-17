package database

import (
	"crypto/rand"
)

// generateRegistrationPassword creates a random 8-character alphanumeric password
func generateRegistrationPassword() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	rand.Read(b)
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}

func init() {
	// Generate a unique password for this installation
	password := generateRegistrationPassword()

	RegisterMigration(&Migration{
		ID:          "002_insert_default_data",
		Description: "Insert default color categories, booking time rules, and system settings",
		Up: map[string]string{
			"sqlite": `
-- Insert default color categories
INSERT OR IGNORE INTO color_categories (name, hex_code, pattern_icon, sort_order) VALUES
  ('gruen', '#28a745', 'circle', 1),
  ('gelb', '#ffc107', 'triangle', 2),
  ('orange', '#fd7e14', 'square', 3),
  ('hellblau', '#17a2b8', 'diamond', 4),
  ('dunkelblau', '#007bff', 'pentagon', 5),
  ('helllila', '#e83e8c', 'hexagon', 6),
  ('dunkellila', '#6f42c1', 'star', 7);

-- Insert default booking time rules
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

-- Insert default system settings
INSERT OR IGNORE INTO system_settings (key, value) VALUES
  ('booking_advance_days', '14'),
  ('cancellation_notice_hours', '12'),
  ('auto_deactivation_days', '365'),
  ('morning_walk_requires_approval', 'true'),
  ('use_feiertage_api', 'true'),
  ('feiertage_state', 'BW'),
  ('booking_time_granularity', '15'),
  ('feiertage_cache_days', '7'),
  ('site_logo', 'https://www.tierheim-goeppingen.de/wp-content/uploads/2017/04/Logo_4c_homepagebanner3.png'),
  ('registration_password', '` + password + `'),
  ('whatsapp_group_enabled', 'false'),
  ('whatsapp_group_link', ''),
  ('default_color_for_new_users', '');
`,
			"mysql": `
-- Insert default color categories
INSERT IGNORE INTO color_categories (name, hex_code, pattern_icon, sort_order) VALUES
  ('gruen', '#28a745', 'circle', 1),
  ('gelb', '#ffc107', 'triangle', 2),
  ('orange', '#fd7e14', 'square', 3),
  ('hellblau', '#17a2b8', 'diamond', 4),
  ('dunkelblau', '#007bff', 'pentagon', 5),
  ('helllila', '#e83e8c', 'hexagon', 6),
  ('dunkellila', '#6f42c1', 'star', 7);

-- Insert default booking time rules
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

-- Insert default system settings
INSERT IGNORE INTO system_settings (` + "`key`" + `, value) VALUES
  ('booking_advance_days', '14'),
  ('cancellation_notice_hours', '12'),
  ('auto_deactivation_days', '365'),
  ('morning_walk_requires_approval', 'true'),
  ('use_feiertage_api', 'true'),
  ('feiertage_state', 'BW'),
  ('booking_time_granularity', '15'),
  ('feiertage_cache_days', '7'),
  ('site_logo', 'https://www.tierheim-goeppingen.de/wp-content/uploads/2017/04/Logo_4c_homepagebanner3.png'),
  ('registration_password', '` + password + `'),
  ('whatsapp_group_enabled', 'false'),
  ('whatsapp_group_link', ''),
  ('default_color_for_new_users', '');
`,
			"postgres": `
-- Insert default color categories
INSERT INTO color_categories (name, hex_code, pattern_icon, sort_order) VALUES
  ('gruen', '#28a745', 'circle', 1),
  ('gelb', '#ffc107', 'triangle', 2),
  ('orange', '#fd7e14', 'square', 3),
  ('hellblau', '#17a2b8', 'diamond', 4),
  ('dunkelblau', '#007bff', 'pentagon', 5),
  ('helllila', '#e83e8c', 'hexagon', 6),
  ('dunkellila', '#6f42c1', 'star', 7)
ON CONFLICT (name) DO NOTHING;

-- Insert default booking time rules
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

-- Insert default system settings
INSERT INTO system_settings (key, value) VALUES
  ('booking_advance_days', '14'),
  ('cancellation_notice_hours', '12'),
  ('auto_deactivation_days', '365'),
  ('morning_walk_requires_approval', 'true'),
  ('use_feiertage_api', 'true'),
  ('feiertage_state', 'BW'),
  ('booking_time_granularity', '15'),
  ('feiertage_cache_days', '7'),
  ('site_logo', 'https://www.tierheim-goeppingen.de/wp-content/uploads/2017/04/Logo_4c_homepagebanner3.png'),
  ('registration_password', '` + password + `'),
  ('whatsapp_group_enabled', 'false'),
  ('whatsapp_group_link', ''),
  ('default_color_for_new_users', '')
ON CONFLICT (key) DO NOTHING;
`,
		},
	})
}
