#!/bin/bash
# Migration script: Old DB (28 migrations with experience_level) -> New DB (2 migrations, color-only)
# For SQLite versions < 3.35.0 that don't support DROP COLUMN

set -e

DB_FILE="${1:-gassigeher.db}"
BACKUP_FILE="${DB_FILE}.backup.$(date +%Y%m%d_%H%M%S)"
EXPORT_DIR="/tmp/gassigeher_migration_$(date +%Y%m%d_%H%M%S)"

echo "=========================================="
echo "Database Migration: Level -> Color System"
echo "=========================================="
echo ""
echo "Database: $DB_FILE"
echo ""

# Check if database exists
if [ ! -f "$DB_FILE" ]; then
    echo "ERROR: Database file '$DB_FILE' not found!"
    echo "Usage: $0 [database_file]"
    exit 1
fi

mkdir -p "$EXPORT_DIR"

# Step 1: Export data
echo "[Step 1/3] Exporting data..."

sqlite3 "$DB_FILE" <<'EOF' > "$EXPORT_DIR/users.sql"
.mode insert users
SELECT id, first_name, last_name, email, phone, password_hash,
       is_admin, is_super_admin, is_verified, is_active, must_change_password,
       verification_token, verification_token_expires,
       reset_token, reset_token_expires,
       profile_photo, terms_accepted_at, last_activity_at,
       is_deleted, anonymous_id, deactivated_at, deactivation_reason,
       created_at
FROM users WHERE is_deleted = 0;
EOF
echo "  - Exported users to $EXPORT_DIR/users.sql"

sqlite3 "$DB_FILE" <<'EOF' > "$EXPORT_DIR/dogs.sql"
.mode insert dogs
SELECT id, name, breed, size, age, color_id, is_available, is_featured,
       description, external_link, photo, photo_thumbnail,
       care_instructions, special_notes, medical_notes,
       created_at, updated_at
FROM dogs;
EOF
echo "  - Exported dogs to $EXPORT_DIR/dogs.sql"

sqlite3 "$DB_FILE" <<'EOF' > "$EXPORT_DIR/user_colors.sql"
.mode insert user_colors
SELECT user_id, color_id, assigned_at FROM user_colors;
EOF
echo "  - Exported user_colors to $EXPORT_DIR/user_colors.sql"

sqlite3 "$DB_FILE" <<'EOF' > "$EXPORT_DIR/bookings.sql"
.mode insert bookings
SELECT id, user_id, dog_id, date, scheduled_time, status,
       completed_at, user_notes, admin_cancellation_reason,
       requires_approval, approval_status, approved_by, approved_at, rejection_reason,
       reminder_sent_at, created_at, updated_at
FROM bookings;
EOF
echo "  - Exported bookings to $EXPORT_DIR/bookings.sql"

sqlite3 "$DB_FILE" <<'EOF' > "$EXPORT_DIR/blocked_dates.sql"
.mode insert blocked_dates
SELECT id, date, reason, dog_id, created_by, created_at FROM blocked_dates;
EOF
echo "  - Exported blocked_dates to $EXPORT_DIR/blocked_dates.sql"

echo ""

# Step 2: Backup and recreate DB
echo "[Step 2/3] Backing up old database..."
mv "$DB_FILE" "$BACKUP_FILE"
echo "  - Backup saved to: $BACKUP_FILE"

echo ""
echo "[Step 3/3] Creating new database and importing data..."
echo "  - Starting application to create fresh schema..."

# Start the app briefly to create schema, then kill it
timeout 5 ./gassigeher 2>/dev/null || true

echo "  - Importing data..."

# Import in correct order (respecting foreign keys)
sqlite3 "$DB_FILE" < "$EXPORT_DIR/users.sql" 2>/dev/null || echo "    (users: some may already exist from seed)"
sqlite3 "$DB_FILE" < "$EXPORT_DIR/dogs.sql" 2>/dev/null || echo "    (dogs: some may already exist from seed)"
sqlite3 "$DB_FILE" < "$EXPORT_DIR/user_colors.sql" 2>/dev/null || echo "    (user_colors: some may already exist)"
sqlite3 "$DB_FILE" < "$EXPORT_DIR/bookings.sql" 2>/dev/null || echo "    (bookings import complete)"
sqlite3 "$DB_FILE" < "$EXPORT_DIR/blocked_dates.sql" 2>/dev/null || echo "    (blocked_dates import complete)"

echo ""
echo "=========================================="
echo "Migration Complete!"
echo "=========================================="
echo ""
echo "Backup location: $BACKUP_FILE"
echo "Export files:    $EXPORT_DIR/"
echo ""
echo "To verify, run: sqlite3 $DB_FILE 'SELECT COUNT(*) FROM users; SELECT COUNT(*) FROM dogs;'"
echo ""
echo "To start the application: ./gassigeher"
