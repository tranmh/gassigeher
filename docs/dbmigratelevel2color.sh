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

# Export users with explicit column mapping
sqlite3 "$DB_FILE" <<'EOF' > "$EXPORT_DIR/users.sql"
SELECT 'INSERT INTO users (id, first_name, last_name, email, phone, password_hash, ' ||
       'is_verified, is_active, is_deleted, is_admin, is_super_admin, must_change_password, ' ||
       'verification_token, verification_token_expires, password_reset_token, password_reset_expires, ' ||
       'profile_photo, anonymous_id, terms_accepted_at, last_activity_at, ' ||
       'deactivated_at, deactivation_reason, reactivated_at, deleted_at, created_at, updated_at) VALUES (' ||
       id || ', ' ||
       COALESCE('''' || REPLACE(first_name, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(last_name, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(email, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(phone, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(password_hash, '''', '''''') || '''', 'NULL') || ', ' ||
       is_verified || ', ' ||
       is_active || ', ' ||
       is_deleted || ', ' ||
       is_admin || ', ' ||
       is_super_admin || ', ' ||
       must_change_password || ', ' ||
       COALESCE('''' || REPLACE(verification_token, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(verification_token_expires, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(password_reset_token, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(password_reset_expires, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(profile_photo, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(anonymous_id, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(terms_accepted_at, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(last_activity_at, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(deactivated_at, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(deactivation_reason, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(reactivated_at, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(deleted_at, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(created_at, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(updated_at, '''', '''''') || '''', 'NULL') ||
       ');'
FROM users WHERE is_deleted = 0;
EOF
echo "  - Exported users to $EXPORT_DIR/users.sql"

# Export dogs with explicit column mapping
sqlite3 "$DB_FILE" <<'EOF' > "$EXPORT_DIR/dogs.sql"
SELECT 'INSERT INTO dogs (id, name, breed, size, age, color_id, photo, photo_thumbnail, ' ||
       'special_needs, pickup_location, walk_route, walk_duration, special_instructions, ' ||
       'default_morning_time, default_evening_time, is_available, is_featured, ' ||
       'unavailable_reason, unavailable_since, external_link, created_at, updated_at) VALUES (' ||
       id || ', ' ||
       COALESCE('''' || REPLACE(name, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(breed, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(size, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE(age, 'NULL') || ', ' ||
       COALESCE(color_id, 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(photo, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(photo_thumbnail, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(special_needs, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(pickup_location, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(walk_route, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE(walk_duration, 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(special_instructions, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(default_morning_time, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(default_evening_time, '''', '''''') || '''', 'NULL') || ', ' ||
       is_available || ', ' ||
       is_featured || ', ' ||
       COALESCE('''' || REPLACE(unavailable_reason, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(unavailable_since, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(external_link, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(created_at, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(updated_at, '''', '''''') || '''', 'NULL') ||
       ');'
FROM dogs;
EOF
echo "  - Exported dogs to $EXPORT_DIR/dogs.sql"

# Export user_colors with explicit column mapping
sqlite3 "$DB_FILE" <<'EOF' > "$EXPORT_DIR/user_colors.sql"
SELECT 'INSERT INTO user_colors (id, user_id, color_id, granted_at, granted_by) VALUES (' ||
       id || ', ' ||
       user_id || ', ' ||
       color_id || ', ' ||
       COALESCE('''' || REPLACE(granted_at, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE(granted_by, 'NULL') ||
       ');'
FROM user_colors;
EOF
echo "  - Exported user_colors to $EXPORT_DIR/user_colors.sql"

# Export bookings with explicit column mapping
sqlite3 "$DB_FILE" <<'EOF' > "$EXPORT_DIR/bookings.sql"
SELECT 'INSERT INTO bookings (id, user_id, dog_id, date, scheduled_time, status, ' ||
       'completed_at, user_notes, admin_cancellation_reason, requires_approval, ' ||
       'approval_status, approved_by, approved_at, rejection_reason, reminder_sent_at, ' ||
       'created_at, updated_at) VALUES (' ||
       id || ', ' ||
       user_id || ', ' ||
       dog_id || ', ' ||
       COALESCE('''' || REPLACE(date, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(scheduled_time, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(status, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(completed_at, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(user_notes, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(admin_cancellation_reason, '''', '''''') || '''', 'NULL') || ', ' ||
       requires_approval || ', ' ||
       COALESCE('''' || REPLACE(approval_status, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE(approved_by, 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(approved_at, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(rejection_reason, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(reminder_sent_at, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(created_at, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(updated_at, '''', '''''') || '''', 'NULL') ||
       ');'
FROM bookings;
EOF
echo "  - Exported bookings to $EXPORT_DIR/bookings.sql"

# Export blocked_dates with explicit column mapping
sqlite3 "$DB_FILE" <<'EOF' > "$EXPORT_DIR/blocked_dates.sql"
SELECT 'INSERT INTO blocked_dates (id, date, dog_id, reason, created_by, created_at) VALUES (' ||
       id || ', ' ||
       COALESCE('''' || REPLACE(date, '''', '''''') || '''', 'NULL') || ', ' ||
       COALESCE(dog_id, 'NULL') || ', ' ||
       COALESCE('''' || REPLACE(reason, '''', '''''') || '''', 'NULL') || ', ' ||
       created_by || ', ' ||
       COALESCE('''' || REPLACE(created_at, '''', '''''') || '''', 'NULL') ||
       ');'
FROM blocked_dates;
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

echo "  - Clearing seed data before import..."
# Clear seed data (app creates test data on empty DB)
# Order matters due to foreign key constraints
sqlite3 "$DB_FILE" "DELETE FROM bookings;"
sqlite3 "$DB_FILE" "DELETE FROM user_colors;"
sqlite3 "$DB_FILE" "DELETE FROM dogs;"
sqlite3 "$DB_FILE" "DELETE FROM users;"
# Reset auto-increment counters
sqlite3 "$DB_FILE" "DELETE FROM sqlite_sequence WHERE name IN ('users', 'dogs', 'bookings', 'user_colors');"

echo "  - Importing data..."

# Import in correct order (respecting foreign keys)
sqlite3 "$DB_FILE" < "$EXPORT_DIR/users.sql" || echo "    (users import failed)"
sqlite3 "$DB_FILE" < "$EXPORT_DIR/dogs.sql" || echo "    (dogs import failed)"
sqlite3 "$DB_FILE" < "$EXPORT_DIR/user_colors.sql" || echo "    (user_colors import failed)"
sqlite3 "$DB_FILE" < "$EXPORT_DIR/bookings.sql" || echo "    (bookings import failed)"
sqlite3 "$DB_FILE" < "$EXPORT_DIR/blocked_dates.sql" || echo "    (blocked_dates import failed)"

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
