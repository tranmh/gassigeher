#!/bin/bash
# Gassigeher Multi-Database Restore Script
# Supports: SQLite, MySQL, PostgreSQL
# Usage: ./restore.sh <backup_file>

set -e

# Load environment variables if .env exists
ENV_FILE="${ENV_FILE:-/var/gassigeher/.env}"
if [ -f "$ENV_FILE" ]; then
    export $(grep -v '^#' "$ENV_FILE" | xargs)
fi

# Configuration with defaults
DB_TYPE="${DB_TYPE:-sqlite}"
LOG_FILE="${LOG_FILE:-/var/gassigeher/logs/restore.log}"

# Ensure log directory exists
mkdir -p "$(dirname "$LOG_FILE")"

# Logging function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# Error handler
error_exit() {
    log "ERROR: $1"
    exit 1
}

# Usage
usage() {
    echo "Usage: $0 <backup_file>"
    echo ""
    echo "Restore a Gassigeher database backup."
    echo ""
    echo "Arguments:"
    echo "  backup_file    Path to the backup file (.gz, .sql.gz, or .dump.gz)"
    echo ""
    echo "Environment variables (or from .env):"
    echo "  DB_TYPE        Database type: sqlite, mysql, postgres (default: sqlite)"
    echo "  DATABASE_PATH  SQLite database path (default: /var/gassigeher/data/gassigeher.db)"
    echo "  DB_HOST        MySQL/PostgreSQL host (default: localhost)"
    echo "  DB_PORT        MySQL (3306) / PostgreSQL (5432) port"
    echo "  DB_USER        Database username"
    echo "  DB_PASSWORD    Database password"
    echo "  DB_NAME        Database name"
    echo ""
    echo "Examples:"
    echo "  DB_TYPE=sqlite ./restore.sh backups/gassigeher_sqlite_20250101.db.gz"
    echo "  DB_TYPE=mysql ./restore.sh backups/gassigeher_mysql_20250101.sql.gz"
    echo "  DB_TYPE=postgres ./restore.sh backups/gassigeher_postgres_20250101.dump.gz"
    exit 1
}

# Check arguments
if [ $# -ne 1 ]; then
    usage
fi

BACKUP_FILE="$1"

if [ ! -f "$BACKUP_FILE" ]; then
    error_exit "Backup file not found: $BACKUP_FILE"
fi

# SQLite restore
restore_sqlite() {
    local db_path="${DATABASE_PATH:-/var/gassigeher/data/gassigeher.db}"
    local backup_file="$1"
    local temp_file="/tmp/gassigeher_restore_$$.db"

    log "Starting SQLite restore to $db_path..."

    # Decompress if needed
    if [[ "$backup_file" == *.gz ]]; then
        log "Decompressing backup..."
        gunzip -c "$backup_file" > "$temp_file" || error_exit "Decompression failed"
    else
        cp "$backup_file" "$temp_file" || error_exit "Copy failed"
    fi

    # Verify the backup is a valid SQLite database
    if ! sqlite3 "$temp_file" "SELECT 1" &>/dev/null; then
        rm -f "$temp_file"
        error_exit "Invalid SQLite database file"
    fi

    # Stop the application if running (optional - depends on your setup)
    # systemctl stop gassigeher || true

    # Create backup of current database
    if [ -f "$db_path" ]; then
        local current_backup="${db_path}.pre_restore_$(date +%Y%m%d_%H%M%S)"
        log "Backing up current database to $current_backup"
        cp "$db_path" "$current_backup"
    fi

    # Ensure directory exists
    mkdir -p "$(dirname "$db_path")"

    # Replace database
    mv "$temp_file" "$db_path" || error_exit "Failed to replace database"
    chmod 644 "$db_path"

    log "SQLite restore completed successfully"
}

# MySQL restore
restore_mysql() {
    local host="${DB_HOST:-localhost}"
    local port="${DB_PORT:-3306}"
    local user="${DB_USER:-gassigeher}"
    local pass="${DB_PASSWORD:-}"
    local name="${DB_NAME:-gassigeher}"
    local backup_file="$1"
    local temp_file="/tmp/gassigeher_restore_$$.sql"

    log "Starting MySQL restore to $name on $host:$port..."

    # Check if mysql client is available
    if ! command -v mysql &> /dev/null; then
        error_exit "mysql client not found. Install mysql-client package."
    fi

    # Decompress if needed
    if [[ "$backup_file" == *.gz ]]; then
        log "Decompressing backup..."
        gunzip -c "$backup_file" > "$temp_file" || error_exit "Decompression failed"
    else
        cp "$backup_file" "$temp_file" || error_exit "Copy failed"
    fi

    # Build connection options
    local conn_opts="-h $host -P $port -u $user"
    if [ -n "$pass" ]; then
        conn_opts="$conn_opts -p$pass"
    fi

    # Drop and recreate database (WARNING: this deletes all data!)
    log "WARNING: This will replace all data in database '$name'"
    log "Dropping and recreating database..."

    mysql $conn_opts -e "DROP DATABASE IF EXISTS $name; CREATE DATABASE $name;" || error_exit "Failed to recreate database"

    # Restore
    log "Restoring data..."
    mysql $conn_opts "$name" < "$temp_file" || error_exit "Restore failed"

    # Cleanup
    rm -f "$temp_file"

    log "MySQL restore completed successfully"
}

# PostgreSQL restore
restore_postgres() {
    local host="${DB_HOST:-localhost}"
    local port="${DB_PORT:-5432}"
    local user="${DB_USER:-gassigeher}"
    local pass="${DB_PASSWORD:-}"
    local name="${DB_NAME:-gassigeher}"
    local backup_file="$1"
    local temp_file="/tmp/gassigeher_restore_$$.dump"

    log "Starting PostgreSQL restore to $name on $host:$port..."

    # Check if pg_restore is available
    if ! command -v pg_restore &> /dev/null; then
        error_exit "pg_restore not found. Install postgresql-client package."
    fi

    # Set password via environment variable
    export PGPASSWORD="$pass"

    # Decompress if needed
    if [[ "$backup_file" == *.gz ]]; then
        log "Decompressing backup..."
        gunzip -c "$backup_file" > "$temp_file" || error_exit "Decompression failed"
    else
        cp "$backup_file" "$temp_file" || error_exit "Copy failed"
    fi

    # Drop and recreate database
    log "WARNING: This will replace all data in database '$name'"
    log "Dropping and recreating database..."

    # Terminate existing connections
    psql -h "$host" -p "$port" -U "$user" -d postgres -c "
        SELECT pg_terminate_backend(pid)
        FROM pg_stat_activity
        WHERE datname = '$name' AND pid <> pg_backend_pid();
    " 2>/dev/null || true

    psql -h "$host" -p "$port" -U "$user" -d postgres -c "DROP DATABASE IF EXISTS $name;" || error_exit "Failed to drop database"
    psql -h "$host" -p "$port" -U "$user" -d postgres -c "CREATE DATABASE $name;" || error_exit "Failed to create database"

    # Restore using pg_restore (for custom format dumps)
    log "Restoring data..."
    pg_restore -h "$host" -p "$port" -U "$user" -d "$name" --no-owner --no-privileges "$temp_file" || {
        # If pg_restore fails, try plain SQL restore
        log "pg_restore failed, trying psql..."
        psql -h "$host" -p "$port" -U "$user" -d "$name" < "$temp_file" || error_exit "Restore failed"
    }

    unset PGPASSWORD

    # Cleanup
    rm -f "$temp_file"

    log "PostgreSQL restore completed successfully"
}

# Confirm restore
confirm_restore() {
    echo ""
    echo "=========================================="
    echo "           WARNING - DATA LOSS           "
    echo "=========================================="
    echo ""
    echo "This will REPLACE ALL DATA in the database!"
    echo ""
    echo "  Database type: $DB_TYPE"
    echo "  Backup file:   $BACKUP_FILE"
    echo ""
    read -p "Are you sure you want to continue? (yes/no): " confirm

    if [ "$confirm" != "yes" ]; then
        log "Restore cancelled by user"
        exit 0
    fi
}

# Main execution
main() {
    log "=========================================="
    log "Starting restore for DB_TYPE=$DB_TYPE"
    log "Backup file: $BACKUP_FILE"

    # Ask for confirmation
    confirm_restore

    case "$DB_TYPE" in
        sqlite)
            restore_sqlite "$BACKUP_FILE"
            ;;
        mysql)
            restore_mysql "$BACKUP_FILE"
            ;;
        postgres|postgresql)
            restore_postgres "$BACKUP_FILE"
            ;;
        *)
            error_exit "Unknown database type: $DB_TYPE. Supported: sqlite, mysql, postgres"
            ;;
    esac

    log "Restore completed successfully"
    log "=========================================="
    echo ""
    echo "Restore complete! You may need to restart the application."
}

# Run main
main "$@"

exit 0
