#!/bin/bash
# Gassigeher Multi-Database Restore Script
# Supports: SQLite, MySQL, PostgreSQL
# Usage: ./restore.sh <backup_file>

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
LOG_FILE="${LOG_FILE:-$INSTALL_DIR/logs/restore.log}"

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
    echo "Restore a Gassigeher database or file storage backup."
    echo ""
    echo "Arguments:"
    echo "  backup_file    Path to the backup file"
    echo "                 Database: .db.gz, .sql.gz, or .dump.gz"
    echo "                 File storage: _files_*.tar.gz"
    echo ""
    echo "Environment variables (or from .env):"
    echo "  DB_TYPE        Database type: sqlite, mysql, postgres (default: sqlite)"
    echo "  DATABASE_PATH  SQLite database path (default: /var/gassigeher/data/gassigeher.db)"
    echo "  DB_HOST        MySQL/PostgreSQL host (default: localhost)"
    echo "  DB_PORT        MySQL (3306) / PostgreSQL (5432) port"
    echo "  DB_USER        Database username"
    echo "  DB_PASSWORD    Database password"
    echo "  DB_NAME        Database name"
    echo "  UPLOADS_DIR    Uploads directory (default: INSTALL_DIR/uploads)"
    echo "  MINIO_DATA_DIR MinIO data directory (default: INSTALL_DIR/data/minio)"
    echo ""
    echo "Examples:"
    echo "  # Database restore"
    echo "  DB_TYPE=sqlite ./restore.sh backups/gassigeher_sqlite_20250101.db.gz"
    echo "  DB_TYPE=mysql ./restore.sh backups/gassigeher_mysql_20250101.sql.gz"
    echo "  DB_TYPE=postgres ./restore.sh backups/gassigeher_postgres_20250101.dump.gz"
    echo ""
    echo "  # Configuration restore (.env with secrets)"
    echo "  ./restore.sh backups/gassigeher_env_20250101.env.gz"
    echo ""
    echo "  # File storage restore (uploads, MinIO data)"
    echo "  ./restore.sh backups/gassigeher_files_20250101.tar.gz"
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
    local temp_file="/tmp/gassigeher_restore_$$.sql"

    log "Starting PostgreSQL restore to $name..."

    # Decompress if needed
    if [[ "$backup_file" == *.gz ]]; then
        log "Decompressing backup..."
        gunzip -c "$backup_file" > "$temp_file" || error_exit "Decompression failed"
    else
        cp "$backup_file" "$temp_file" || error_exit "Copy failed"
    fi

    # Check if running via Docker Compose
    local docker_compose_file="$INSTALL_DIR/docker-compose.yml"
    if [ -f "$docker_compose_file" ] && docker compose -f "$docker_compose_file" ps db 2>/dev/null | grep -q "running"; then
        log "Using Docker Compose container for restore..."

        # Stop the app to release connections
        log "Stopping app container..."
        docker compose -f "$docker_compose_file" stop app 2>/dev/null || true

        # Drop and recreate database
        log "Dropping and recreating database..."
        docker compose -f "$docker_compose_file" exec -T db psql -U "$user" -d postgres -c "DROP DATABASE IF EXISTS $name;"
        docker compose -f "$docker_compose_file" exec -T db psql -U "$user" -d postgres -c "CREATE DATABASE $name;"

        # Restore
        log "Restoring data..."
        cat "$temp_file" | docker compose -f "$docker_compose_file" exec -T db psql -U "$user" -d "$name" || error_exit "Restore failed"

        # Restart app
        log "Restarting app container..."
        docker compose -f "$docker_compose_file" start app
    else
        # Check if psql is available locally
        if ! command -v psql &> /dev/null; then
            error_exit "psql not found. Install postgresql-client package."
        fi

        log "Using local psql to $host:$port..."
        export PGPASSWORD="$pass"

        # Terminate existing connections
        psql -h "$host" -p "$port" -U "$user" -d postgres -c "
            SELECT pg_terminate_backend(pid)
            FROM pg_stat_activity
            WHERE datname = '$name' AND pid <> pg_backend_pid();
        " 2>/dev/null || true

        psql -h "$host" -p "$port" -U "$user" -d postgres -c "DROP DATABASE IF EXISTS $name;" || error_exit "Failed to drop database"
        psql -h "$host" -p "$port" -U "$user" -d postgres -c "CREATE DATABASE $name;" || error_exit "Failed to create database"

        # Restore
        log "Restoring data..."
        psql -h "$host" -p "$port" -U "$user" -d "$name" < "$temp_file" || error_exit "Restore failed"

        unset PGPASSWORD
    fi

    # Cleanup
    rm -f "$temp_file"

    log "PostgreSQL restore completed successfully"
}

# File storage restore
restore_file_storage() {
    local backup_file="$1"
    local temp_dir="/tmp/gassigeher_restore_files_$$"

    log "Starting file storage restore from $backup_file..."

    # Create temp directory
    mkdir -p "$temp_dir"

    # Decompress if needed and extract
    if [[ "$backup_file" == *.tar.gz ]]; then
        log "Extracting backup archive..."
        tar -xzf "$backup_file" -C "$temp_dir" || error_exit "Extraction failed"
    elif [[ "$backup_file" == *.tar ]]; then
        tar -xf "$backup_file" -C "$temp_dir" || error_exit "Extraction failed"
    else
        error_exit "Unknown file format. Expected .tar or .tar.gz"
    fi

    # Find and restore uploads directory
    local uploads_src=$(find "$temp_dir" -type d -name "uploads" -print -quit)
    if [ -n "$uploads_src" ] && [ -d "$uploads_src" ]; then
        local uploads_dest="${UPLOADS_DIR:-$INSTALL_DIR/uploads}"
        log "Restoring uploads to $uploads_dest..."

        # Backup current uploads if exists
        if [ -d "$uploads_dest" ] && [ "$(ls -A "$uploads_dest" 2>/dev/null)" ]; then
            local uploads_backup="${uploads_dest}.pre_restore_$(date +%Y%m%d_%H%M%S)"
            log "Backing up current uploads to $uploads_backup"
            mv "$uploads_dest" "$uploads_backup"
        fi

        mkdir -p "$(dirname "$uploads_dest")"
        cp -r "$uploads_src" "$uploads_dest"
        log "Uploads restored successfully"
    else
        log "No uploads directory found in backup"
    fi

    # Find and restore MinIO data directory
    local minio_src=$(find "$temp_dir" -type d -name "minio" -print -quit)
    if [ -n "$minio_src" ] && [ -d "$minio_src" ]; then
        local minio_dest="${MINIO_DATA_DIR:-$INSTALL_DIR/data/minio}"
        log "Restoring MinIO data to $minio_dest..."

        # Backup current MinIO data if exists
        if [ -d "$minio_dest" ] && [ "$(ls -A "$minio_dest" 2>/dev/null)" ]; then
            local minio_backup="${minio_dest}.pre_restore_$(date +%Y%m%d_%H%M%S)"
            log "Backing up current MinIO data to $minio_backup"
            mv "$minio_dest" "$minio_backup"
        fi

        mkdir -p "$(dirname "$minio_dest")"
        cp -r "$minio_src" "$minio_dest"
        log "MinIO data restored successfully"
    else
        log "No MinIO data directory found in backup"
    fi

    # Cleanup temp directory
    rm -rf "$temp_dir"

    log "File storage restore completed successfully"
}

# .env configuration restore
restore_env_config() {
    local backup_file="$1"
    local env_dest="${ENV_FILE:-$INSTALL_DIR/.env}"

    log "Starting .env configuration restore from $backup_file..."

    # Decompress if needed
    local temp_file="/tmp/gassigeher_restore_env_$$.env"
    if [[ "$backup_file" == *.gz ]]; then
        log "Decompressing .env backup..."
        gunzip -c "$backup_file" > "$temp_file" || error_exit "Decompression failed"
    else
        cp "$backup_file" "$temp_file" || error_exit "Copy failed"
    fi

    # Backup current .env if exists
    if [ -f "$env_dest" ]; then
        local env_backup="${env_dest}.pre_restore_$(date +%Y%m%d_%H%M%S)"
        log "Backing up current .env to $env_backup"
        cp "$env_dest" "$env_backup"
        chmod 600 "$env_backup"
    fi

    # Restore .env
    mkdir -p "$(dirname "$env_dest")"
    mv "$temp_file" "$env_dest" || error_exit "Failed to restore .env"
    chmod 600 "$env_dest"

    log ".env configuration restore completed successfully"
    log "IMPORTANT: Review restored .env and update any environment-specific values!"
}

# Detect backup type from filename
detect_backup_type() {
    local filename="$1"

    if [[ "$filename" == *"_env_"*.env* ]]; then
        echo "env"
    elif [[ "$filename" == *"_files_"*.tar* ]]; then
        echo "files"
    elif [[ "$filename" == *"_sqlite_"* ]]; then
        echo "sqlite"
    elif [[ "$filename" == *"_mysql_"* ]]; then
        echo "mysql"
    elif [[ "$filename" == *"_postgres_"* ]]; then
        echo "postgres"
    else
        echo "unknown"
    fi
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
    log "Backup file: $BACKUP_FILE"

    # Auto-detect backup type from filename
    local detected_type=$(detect_backup_type "$BACKUP_FILE")
    log "Detected backup type: $detected_type"

    # Handle .env configuration backups
    if [ "$detected_type" = "env" ]; then
        log "Starting .env configuration restore..."

        # Ask for confirmation
        confirm_restore

        restore_env_config "$BACKUP_FILE"

        log "Restore completed successfully"
        log "=========================================="
        echo ""
        echo ".env configuration restore complete!"
        echo "IMPORTANT: Review the restored .env file and restart the application."
        return
    fi

    # Handle file storage backups
    if [ "$detected_type" = "files" ]; then
        log "Starting file storage restore..."

        # Ask for confirmation
        confirm_restore

        restore_file_storage "$BACKUP_FILE"

        log "Restore completed successfully"
        log "=========================================="
        echo ""
        echo "File storage restore complete!"
        echo "If using Docker, you may need to restart MinIO: docker compose restart minio"
        return
    fi

    # Handle database backups
    # Use detected type if available, otherwise fall back to DB_TYPE env var
    if [ "$detected_type" != "unknown" ]; then
        DB_TYPE="$detected_type"
    fi

    log "Starting database restore for DB_TYPE=$DB_TYPE"

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
    echo "Database restore complete! You may need to restart the application."
}

# Run main
main "$@"

exit 0
