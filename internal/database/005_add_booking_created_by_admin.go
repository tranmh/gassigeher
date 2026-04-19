package database

func init() {
	RegisterMigration(&Migration{
		ID:          "005_add_booking_created_by_admin",
		Description: "Add created_by_admin column to bookings for admin-created-on-behalf audit trail",
		Up: map[string]string{
			"sqlite": `
-- Add created_by_admin column to bookings table
ALTER TABLE bookings ADD COLUMN created_by_admin INTEGER REFERENCES users(id) ON DELETE SET NULL;
`,
			"postgres": `
-- Add created_by_admin column to bookings table
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS created_by_admin INTEGER REFERENCES users(id) ON DELETE SET NULL;
`,
		},
	})
}
