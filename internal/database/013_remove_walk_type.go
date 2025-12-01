package database

func init() {
	RegisterMigration(&Migration{
		ID:          "013_remove_walk_type",
		Description: "Remove walk_type field from bookings, use only scheduled_time",
		Up: map[string]string{
			"sqlite": `
-- SQLite requires table recreation to remove column and change unique constraint
-- Create new table without walk_type column
CREATE TABLE IF NOT EXISTS bookings_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    dog_id INTEGER NOT NULL,
    date DATE NOT NULL,
    scheduled_time TEXT NOT NULL,
    status TEXT DEFAULT 'scheduled' CHECK(status IN ('scheduled', 'completed', 'cancelled')),
    completed_at TIMESTAMP,
    user_notes TEXT,
    admin_cancellation_reason TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    requires_approval INTEGER DEFAULT 0,
    approval_status TEXT DEFAULT 'approved',
    approved_by INTEGER,
    approved_at TIMESTAMP,
    rejection_reason TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (dog_id) REFERENCES dogs(id) ON DELETE CASCADE,
    FOREIGN KEY (approved_by) REFERENCES users(id) ON DELETE SET NULL,
    UNIQUE(dog_id, date, scheduled_time)
);

-- Copy data from old table (excluding walk_type)
INSERT INTO bookings_new (
    id, user_id, dog_id, date, scheduled_time, status,
    completed_at, user_notes, admin_cancellation_reason,
    created_at, updated_at, requires_approval, approval_status,
    approved_by, approved_at, rejection_reason
)
SELECT
    id, user_id, dog_id, date, scheduled_time, status,
    completed_at, user_notes, admin_cancellation_reason,
    created_at, updated_at, requires_approval, approval_status,
    approved_by, approved_at, rejection_reason
FROM bookings;

-- Drop old table and rename new one
DROP TABLE bookings;
ALTER TABLE bookings_new RENAME TO bookings;

-- Recreate indexes
CREATE INDEX IF NOT EXISTS idx_bookings_user ON bookings(user_id);
CREATE INDEX IF NOT EXISTS idx_bookings_dog ON bookings(dog_id);
CREATE INDEX IF NOT EXISTS idx_bookings_date ON bookings(date);
CREATE INDEX IF NOT EXISTS idx_bookings_status ON bookings(status);
CREATE INDEX IF NOT EXISTS idx_bookings_approval_status ON bookings(approval_status);
`,
			"mysql": `
-- MySQL: Drop the unique constraint that includes walk_type, then drop column
ALTER TABLE bookings DROP INDEX IF EXISTS unique_dog_date_walk;
ALTER TABLE bookings DROP INDEX IF EXISTS bookings_dog_id_date_walk_type_key;

-- Drop walk_type column
ALTER TABLE bookings DROP COLUMN IF EXISTS walk_type;

-- Add new unique constraint on dog_id, date, scheduled_time
ALTER TABLE bookings ADD UNIQUE INDEX unique_dog_date_time (dog_id, date, scheduled_time);
`,
			"postgres": `
-- PostgreSQL: Drop constraints and column
ALTER TABLE bookings DROP CONSTRAINT IF EXISTS bookings_dog_id_date_walk_type_key;

-- Drop walk_type column
ALTER TABLE bookings DROP COLUMN IF EXISTS walk_type;

-- Add new unique constraint
ALTER TABLE bookings ADD CONSTRAINT bookings_dog_date_time_unique UNIQUE (dog_id, date, scheduled_time);
`,
		},
	})
}
