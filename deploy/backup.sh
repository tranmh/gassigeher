#!/bin/bash
# Gassigeher Multi-Database Backup Script
# Supports: SQLite, MySQL, PostgreSQL
# Run daily via cron: 0 3 * * * /opt/gassigeher/backup.sh >> /opt/gassigeher/logs/backup.log 2>&1

set -e

# Installation directory (Docker Compose default)
INSTALL_DIR="${INSTALL_DIR:-/opt/gassigeher}"

# Load environment variables if .env exists
ENV_FILE="${ENV_FILE:-$INSTALL_DIR/.env}"
if [ -f "$ENV_FILE" ]; then
    export $(grep -v '^#' "$ENV_FILE" | xargs)
fi

# Configuration with defaults
DB_TYPE="${DB_TYPE:-postgres}"
BACKUP_DIR="${BACKUP_DIR:-$INSTALL_DIR/backups}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"
DATE=$(date +%Y%m%d_%H%M%S)
LOG_FILE="${LOG_FILE:-$INSTALL_DIR/logs/backup.log}"

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
    local backup_file="${BACKUP_DIR}/gassigeher_postgres_${DATE}.sql"

    log "Starting PostgreSQL backup of $name..."

    # Check if running via Docker Compose
    local docker_compose_file="${INSTALL_DIR:-/opt/gassigeher}/docker-compose.yml"
    if [ -f "$docker_compose_file" ] && docker compose -f "$docker_compose_file" ps db 2>/dev/null | grep -q "running"; then
        log "Using Docker Compose container for backup..."
        docker compose -f "$docker_compose_file" exec -T db \
            pg_dump -U "$user" "$name" > "$backup_file" || error_exit "PostgreSQL backup failed"
    else
        # Check if pg_dump is available locally
        if ! command -v pg_dump &> /dev/null; then
            error_exit "pg_dump not found. Install postgresql-client package."
        fi

        log "Using local pg_dump from $host:$port..."
        # Set password via environment variable (more secure than command line)
        export PGPASSWORD="$pass"

        # Perform backup with custom format (allows selective restore)
        pg_dump -h "$host" -p "$port" -U "$user" \
            -Fc -b \
            "$name" > "${backup_file%.sql}.dump" || error_exit "PostgreSQL backup failed"
        backup_file="${backup_file%.sql}.dump"

        unset PGPASSWORD
    fi

    # Compress
    gzip "$backup_file" || error_exit "Compression failed"

    log "PostgreSQL backup completed: ${backup_file}.gz ($(du -h "${backup_file}.gz" | cut -f1))"
}

# Backup .env configuration file
backup_env_config() {
    log "Starting .env configuration backup..."

    local env_file="${ENV_FILE:-$INSTALL_DIR/.env}"
    local backup_file="${BACKUP_DIR}/gassigeher_env_${DATE}.env"

    if [ ! -f "$env_file" ]; then
        log "Warning: .env file not found at $env_file, skipping"
        return 0
    fi

    # Copy .env file (contains secrets - handle carefully!)
    cp "$env_file" "$backup_file" || error_exit ".env backup failed"

    # Set restrictive permissions on backup
    chmod 600 "$backup_file"

    # Compress
    gzip "$backup_file" || error_exit ".env compression failed"

    # Set restrictive permissions on compressed file
    chmod 600 "${backup_file}.gz"

    log ".env backup completed: ${backup_file}.gz (contains secrets - store securely!)"
}

# Backup file storage (uploads, MinIO data)
backup_file_storage() {
    log "Starting file storage backup..."

    local backup_file="${BACKUP_DIR}/gassigeher_files_${DATE}.tar"
    local dirs_to_backup=""

    # Check for uploads directory (Simple-Mode filesystem storage)
    local uploads_dir="${UPLOADS_DIR:-$INSTALL_DIR/uploads}"
    if [ -d "$uploads_dir" ] && [ "$(ls -A "$uploads_dir" 2>/dev/null)" ]; then
        dirs_to_backup="$uploads_dir"
        log "Found uploads directory: $uploads_dir"
    fi

    # Check for MinIO data directory (Docker mode S3 storage)
    local minio_dir="${MINIO_DATA_DIR:-$INSTALL_DIR/data/minio}"
    if [ -d "$minio_dir" ] && [ "$(ls -A "$minio_dir" 2>/dev/null)" ]; then
        dirs_to_backup="$dirs_to_backup $minio_dir"
        log "Found MinIO data directory: $minio_dir"
    fi

    # Skip if no directories to backup
    if [ -z "$dirs_to_backup" ]; then
        log "No file storage directories found to backup (uploads or MinIO data)"
        return 0
    fi

    # Create tar archive
    tar -cf "$backup_file" $dirs_to_backup 2>/dev/null || {
        log "Warning: Some files could not be archived (may be in use)"
    }

    # Check if archive was created
    if [ ! -f "$backup_file" ] || [ ! -s "$backup_file" ]; then
        log "Warning: File storage backup appears empty or failed"
        rm -f "$backup_file" 2>/dev/null
        return 0
    fi

    # Compress
    gzip "$backup_file" || error_exit "File storage compression failed"

    log "File storage backup completed: ${backup_file}.gz ($(du -h "${backup_file}.gz" | cut -f1))"
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

    # Step 1: Database backup
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

    # Step 2: Configuration backup (.env with secrets)
    backup_env_config

    # Step 3: File storage backup (uploads, MinIO data)
    backup_file_storage

    # Step 4: Cleanup old backups
    cleanup_old_backups

    log "Backup completed successfully"
    log "=========================================="
}

# Run main
main "$@"

exit 0
