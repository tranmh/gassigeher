package database

func init() {
	RegisterMigration(&Migration{
		ID:          "019_add_dog_id_to_blocked_dates",
		Description: "Add dog_id column to blocked_dates for dog-specific date blocking",
		Up: map[string]string{
			"sqlite": `
-- SQLite requires recreating the table to change constraints
-- Step 1: Create new table with updated schema
CREATE TABLE IF NOT EXISTS blocked_dates_new (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  date DATE NOT NULL,
  dog_id INTEGER,
  reason TEXT NOT NULL,
  created_by INTEGER NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (created_by) REFERENCES users(id),
  FOREIGN KEY (dog_id) REFERENCES dogs(id) ON DELETE CASCADE
);

-- Step 2: Copy existing data (all existing blocks become global blocks with dog_id=NULL)
INSERT INTO blocked_dates_new (id, date, dog_id, reason, created_by, created_at)
SELECT id, date, NULL, reason, created_by, created_at FROM blocked_dates;

-- Step 3: Drop old table
DROP TABLE blocked_dates;

-- Step 4: Rename new table
ALTER TABLE blocked_dates_new RENAME TO blocked_dates;

-- Step 5: Create unique index using COALESCE to handle NULL dog_id
-- This allows: one global block per date (dog_id=NULL -> 0) and one block per dog per date
CREATE UNIQUE INDEX IF NOT EXISTS idx_blocked_dates_dog_date
ON blocked_dates (COALESCE(dog_id, 0), date);

-- Step 6: Index for faster lookups by date
CREATE INDEX IF NOT EXISTS idx_blocked_dates_date ON blocked_dates (date);
`,
			"mysql": `
-- Add dog_id column (nullable for global blocks)
ALTER TABLE blocked_dates ADD COLUMN dog_id INT NULL;

-- Add foreign key constraint with cascade delete
ALTER TABLE blocked_dates ADD CONSTRAINT fk_blocked_dates_dog
FOREIGN KEY (dog_id) REFERENCES dogs(id) ON DELETE CASCADE;

-- Drop the old unique constraint on date
ALTER TABLE blocked_dates DROP INDEX date;

-- Add computed column for unique constraint (handles NULL as 0)
ALTER TABLE blocked_dates ADD COLUMN dog_id_unique INT AS (COALESCE(dog_id, 0)) STORED;

-- Create new unique constraint on computed column + date
ALTER TABLE blocked_dates ADD UNIQUE KEY idx_blocked_dates_dog_date (dog_id_unique, date);

-- Index for faster lookups by date
CREATE INDEX idx_blocked_dates_date ON blocked_dates (date);
`,
			"postgres": `
-- Add dog_id column (nullable for global blocks)
ALTER TABLE blocked_dates ADD COLUMN IF NOT EXISTS dog_id INTEGER;

-- Add foreign key constraint with cascade delete
ALTER TABLE blocked_dates ADD CONSTRAINT fk_blocked_dates_dog
FOREIGN KEY (dog_id) REFERENCES dogs(id) ON DELETE CASCADE;

-- Drop the old unique constraint on date (PostgreSQL uses _key suffix)
ALTER TABLE blocked_dates DROP CONSTRAINT IF EXISTS blocked_dates_date_key;

-- Create unique index using COALESCE to handle NULL
-- COALESCE(dog_id, 0) means NULL becomes 0 for uniqueness check
CREATE UNIQUE INDEX IF NOT EXISTS idx_blocked_dates_dog_date
ON blocked_dates (COALESCE(dog_id, 0), date);

-- Index for faster lookups by date
CREATE INDEX IF NOT EXISTS idx_blocked_dates_date ON blocked_dates (date);
`,
		},
	})
}
