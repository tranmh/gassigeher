package database

func init() {
	RegisterMigration(&Migration{
		ID:          "016_add_reminder_sent",
		Description: "Add reminder_sent_at column to bookings table for tracking sent reminders",
		Up: map[string]string{
			"sqlite": `
-- Add reminder_sent_at column to bookings table
ALTER TABLE bookings ADD COLUMN reminder_sent_at DATETIME;
`,
			"mysql": `
-- Add reminder_sent_at column to bookings table
ALTER TABLE bookings ADD COLUMN reminder_sent_at DATETIME;
`,
			"postgres": `
-- Add reminder_sent_at column to bookings table
ALTER TABLE bookings ADD COLUMN reminder_sent_at TIMESTAMP;
`,
		},
	})
}
