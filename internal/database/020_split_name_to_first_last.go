package database

func init() {
	RegisterMigration(&Migration{
		ID:          "020_split_name_to_first_last",
		Description: "Split name column into first_name and last_name",
		Up: map[string]string{
			"sqlite": `
-- Add new columns
ALTER TABLE users ADD COLUMN first_name TEXT;
ALTER TABLE users ADD COLUMN last_name TEXT;

-- Migrate existing data: first word = first_name, rest = last_name
-- For single word names, put it in first_name
UPDATE users SET
    first_name = CASE
        WHEN INSTR(name, ' ') > 0 THEN SUBSTR(name, 1, INSTR(name, ' ') - 1)
        ELSE name
    END,
    last_name = CASE
        WHEN INSTR(name, ' ') > 0 THEN TRIM(SUBSTR(name, INSTR(name, ' ') + 1))
        ELSE ''
    END
WHERE name IS NOT NULL;

-- Handle special case for deleted users
UPDATE users SET first_name = 'Deleted', last_name = 'User' WHERE name = 'Deleted User';
`,
			"mysql": `
-- Add new columns
ALTER TABLE users ADD COLUMN first_name VARCHAR(255);
ALTER TABLE users ADD COLUMN last_name VARCHAR(255);

-- Migrate existing data: first word = first_name, rest = last_name
UPDATE users SET
    first_name = CASE
        WHEN LOCATE(' ', name) > 0 THEN SUBSTRING_INDEX(name, ' ', 1)
        ELSE name
    END,
    last_name = CASE
        WHEN LOCATE(' ', name) > 0 THEN TRIM(SUBSTRING(name, LOCATE(' ', name) + 1))
        ELSE ''
    END
WHERE name IS NOT NULL;

-- Handle special case for deleted users
UPDATE users SET first_name = 'Deleted', last_name = 'User' WHERE name = 'Deleted User';
`,
			"postgres": `
-- Add new columns
ALTER TABLE users ADD COLUMN IF NOT EXISTS first_name VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_name VARCHAR(255);

-- Migrate existing data: first word = first_name, rest = last_name
UPDATE users SET
    first_name = CASE
        WHEN POSITION(' ' IN name) > 0 THEN SPLIT_PART(name, ' ', 1)
        ELSE name
    END,
    last_name = CASE
        WHEN POSITION(' ' IN name) > 0 THEN TRIM(SUBSTRING(name FROM POSITION(' ' IN name) + 1))
        ELSE ''
    END
WHERE name IS NOT NULL;

-- Handle special case for deleted users
UPDATE users SET first_name = 'Deleted', last_name = 'User' WHERE name = 'Deleted User';
`,
		},
	})
}
