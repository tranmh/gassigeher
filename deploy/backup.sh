#!/bin/bash
# Gassigeher Multi-Database Backup Script
# Supports: SQLite, MySQL, PostgreSQL
# Run daily via cron: 0 2 * * * /var/gassigeher/deploy/backup.sh

set -e

# Load environment variables if .env exists
ENV_FILE="${ENV_FILE:-/var/gassigeher/.env}"
if [ -f "$ENV_FILE" ]; then
    export $(grep -v '^#' "$ENV_FILE" | xargs)
fi

# Configuration with defaults
DB_TYPE="${DB_TYPE:-sqlite}"
BACKUP_DIR="${BACKUP_DIR:-/var/gassigeher/backups}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"
DATE=$(date +%Y%m%d_%H%M%S)
LOG_FILE="${LOG_FILE:-/var/gassigeher/logs/backup.log}"

# Ensure directories exist
mkdir -p "$BACKUP_DIR"
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

# SQLite backup
backup_sqlite() {
    local db_path="${DATABASE_PATH:-/var/gassigeher/data/gassigeher.db}"
    local backup_file="${BACKUP_DIR}/gassigeher_sqlite_${DATE}.db"

    log "Starting SQLite backup from $db_path..."

    if [ ! -f "$db_path" ]; then
        error_exit "SQLite database not found: $db_path"
    fi

    # Use SQLite's built-in backup command (safe for concurrent access)
    sqlite3 "$db_path" ".backup '$backup_file'" || error_exit "SQLite backup failed"

    # Compress
    gzip "$backup_file" || error_exit "Compression failed"

    log "SQLite backup completed: ${backup_file}.gz ($(du -h "${backup_file}.gz" | cut -f1))"
}

# MySQL backup
backup_mysql() {
    local host="${DB_HOST:-localhost}"
    local port="${DB_PORT:-3306}"
    local user="${DB_USER:-gassigeher}"
    local pass="${DB_PASSWORD:-}"
    local name="${DB_NAME:-gassigeher}"
    local backup_file="${BACKUP_DIR}/gassigeher_mysql_${DATE}.sql"

    log "Starting MySQL backup of $name from $host:$port..."

    # Check if mysqldump is available
    if ! command -v mysqldump &> /dev/null; then
        error_exit "mysqldump not found. Install mysql-client package."
    fi

    # Build connection options
    local conn_opts="-h $host -P $port -u $user"
    if [ -n "$pass" ]; then
        conn_opts="$conn_opts -p$pass"
    fi

    # Perform backup with best practices:
    # --single-transaction: Consistent snapshot without locking tables
    # --routines: Include stored procedures
    # --triggers: Include triggers
    # --quick: Don't buffer result set in memory
    # --lock-tables=false: No table locking for InnoDB
    mysqldump $conn_opts \
        --single-transaction \
        --routines \
        --triggers \
        --quick \
        --lock-tables=false \
        "$name" > "$backup_file" || error_exit "MySQL backup failed"

    # Compress
    gzip "$backup_file" || error_exit "Compression failed"

    log "MySQL backup completed: ${backup_file}.gz ($(du -h "${backup_file}.gz" | cut -f1))"
}

# PostgreSQL backup
backup_postgres() {
    local host="${DB_HOST:-localhost}"
    local port="${DB_PORT:-5432}"
    local user="${DB_USER:-gassigeher}"
    local pass="${DB_PASSWORD:-}"
    local name="${DB_NAME:-gassigeher}"
    local backup_file="${BACKUP_DIR}/gassigeher_postgres_${DATE}.dump"

    log "Starting PostgreSQL backup of $name from $host:$port..."

    # Check if pg_dump is available
    if ! command -v pg_dump &> /dev/null; then
        error_exit "pg_dump not found. Install postgresql-client package."
    fi

    # Set password via environment variable (more secure than command line)
    export PGPASSWORD="$pass"

    # Perform backup with custom format (allows selective restore)
    # -Fc: Custom format (compressed, allows pg_restore)
    # -b: Include large objects
    # -v: Verbose mode
    pg_dump -h "$host" -p "$port" -U "$user" \
        -Fc -b \
        "$name" > "$backup_file" || error_exit "PostgreSQL backup failed"

    unset PGPASSWORD

    # Custom format is already compressed, but we can gzip for consistency
    gzip "$backup_file" || error_exit "Compression failed"

    log "PostgreSQL backup completed: ${backup_file}.dump.gz ($(du -h "${backup_file}.gz" | cut -f1))"
}

# Cleanup old backups
cleanup_old_backups() {
    log "Cleaning up backups older than $RETENTION_DAYS days..."

    local deleted=$(find "$BACKUP_DIR" -name "gassigeher_*" -type f -mtime +$RETENTION_DAYS -delete -print | wc -l)

    if [ "$deleted" -gt 0 ]; then
        log "Deleted $deleted old backup(s)"
    fi

    # Count and report remaining backups
    local total=$(find "$BACKUP_DIR" -name "gassigeher_*" -type f | wc -l)
    local size=$(du -sh "$BACKUP_DIR" 2>/dev/null | cut -f1)
    log "Total backups: $total files, $size"
}

# Main execution
main() {
    log "=========================================="
    log "Starting backup for DB_TYPE=$DB_TYPE"

    case "$DB_TYPE" in
        sqlite)
            backup_sqlite
            ;;
        mysql)
            backup_mysql
            ;;
        postgres|postgresql)
            backup_postgres
            ;;
        *)
            error_exit "Unknown database type: $DB_TYPE. Supported: sqlite, mysql, postgres"
            ;;
    esac

    cleanup_old_backups

    log "Backup completed successfully"
    log "=========================================="
}

# Run main
main "$@"

exit 0
