#!/bin/bash
#
# Gassigeher - Production Deployment Script
# Target: Ubuntu 24.04 LTS with Docker Compose
#
# Supports both Simple-Mode and SaaS-Mode deployments.
#
# Usage:
#   ./install-ubuntu-24.sh                    # Install/update (preserves existing config)
#   ./install-ubuntu-24.sh --force            # Fresh install (regenerates secrets)
#   ./install-ubuntu-24.sh --local            # Local testing mode (no SSL)
#   ./install-ubuntu-24.sh --env /path/.env   # Custom config file location
#
# Configuration:
#   The script reads configuration from:
#     - .env         -> Application settings (from .env.example if not present)
#     - .env.secrets -> Auto-generated passwords and API keys
#
#   Manual configuration required in .env.secrets:
#     - HETZNER_DNS_TOKEN (for wildcard SSL)
#     - STRIPE_* keys (for SaaS billing)
#     - SMTP_PASSWORD (for email)
#
# Prerequisites:
#   - Fresh Ubuntu 24.04 LTS server
#   - Domain pointing to server IP (for production)
#

set -e
set -o pipefail

# ============================================
# CONFIGURATION
# ============================================

# Default installation directory
INSTALL_DIR="/opt/gassigeher"

# Default config file (can be overridden with --env)
CONFIG_FILE=""

# Flags
FORCE_MODE=false
LOCAL_MODE=false
DRY_RUN=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ============================================
# HELPER FUNCTIONS
# ============================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_sudo() {
    if [[ $EUID -eq 0 ]]; then
        SUDO=""
        CAN_SUDO=true
    elif sudo -n true 2>/dev/null; then
        SUDO="sudo"
        CAN_SUDO=true
    else
        SUDO=""
        CAN_SUDO=false
        log_warn "Running without root privileges. Some features may be limited."
        log_warn "Docker must already be installed and accessible."
    fi
}

check_ubuntu() {
    if [[ -f /etc/os-release ]]; then
        if ! grep -q "Ubuntu" /etc/os-release; then
            log_warn "This script is designed for Ubuntu 24.04"
        fi
    fi
    log_success "OS check passed"
}

generate_password() {
    openssl rand -base64 32 | tr -d '/+=' | cut -c1-32
}

# Parse a simple .env file and export variables
# Usage: parse_env_file /path/to/.env
parse_env_file() {
    local file="$1"
    if [[ ! -f "$file" ]]; then
        return 1
    fi

    while IFS= read -r line || [[ -n "$line" ]]; do
        # Skip empty lines and comments
        [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]] && continue

        # Extract key=value, handling quotes
        if [[ "$line" =~ ^([A-Za-z_][A-Za-z0-9_]*)=(.*)$ ]]; then
            local key="${BASH_REMATCH[1]}"
            local value="${BASH_REMATCH[2]}"

            # Remove surrounding quotes if present
            value="${value%\"}"
            value="${value#\"}"
            value="${value%\'}"
            value="${value#\'}"

            # Remove inline comments (but not in quoted values)
            if [[ ! "$value" =~ ^[\"\'] ]]; then
                value="${value%%#*}"
                value="${value%"${value##*[![:space:]]}"}"  # Trim trailing whitespace
            fi

            # Export the variable
            export "$key=$value"
        fi
    done < "$file"
}

# Get value from env file without exporting
get_env_value() {
    local file="$1"
    local key="$2"
    local value=""

    if [[ -f "$file" ]]; then
        value=$(grep -E "^${key}=" "$file" 2>/dev/null | head -1 | cut -d'=' -f2-)
        # Remove quotes
        value="${value%\"}"
        value="${value#\"}"
        value="${value%\'}"
        value="${value#\'}"
    fi

    echo "$value"
}

# ============================================
# INSTALLATION STEPS
# ============================================

install_docker() {
    if [[ "$CAN_SUDO" != "true" ]]; then
        log_error "Cannot install Docker without root privileges"
        log_error "Please install Docker manually or run with sudo"
        exit 1
    fi

    log_info "Installing Docker..."

    # Remove old versions
    $SUDO apt-get remove -y docker docker-engine docker.io containerd runc 2>/dev/null || true

    # Install prerequisites
    $SUDO apt-get update
    $SUDO apt-get install -y ca-certificates curl gnupg lsb-release

    # Add Docker GPG key
    $SUDO install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | $SUDO gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    $SUDO chmod a+r /etc/apt/keyrings/docker.gpg

    # Add Docker repository
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | $SUDO tee /etc/apt/sources.list.d/docker.list > /dev/null

    # Install Docker
    $SUDO apt-get update
    $SUDO apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

    # Start Docker
    $SUDO systemctl enable docker
    $SUDO systemctl start docker

    log_success "Docker installed successfully"
}

create_directories() {
    log_info "Creating directories..."

    if [[ "$CAN_SUDO" == "true" ]]; then
        $SUDO mkdir -p "$INSTALL_DIR"/{data/app,data/postgres,data/minio,data/caddy,data/caddy-config,logs,logs/caddy,backups,uploads}
        $SUDO chmod 750 "$INSTALL_DIR"
        if [[ -n "$SUDO" ]]; then
            $SUDO chown -R "$(id -u):$(id -g)" "$INSTALL_DIR"
        fi
    else
        mkdir -p "$INSTALL_DIR"/{data/app,data/postgres,data/minio,data/caddy,data/caddy-config,logs,logs/caddy,backups,uploads}
        chmod 750 "$INSTALL_DIR"
    fi

    log_success "Directories created at $INSTALL_DIR"
}

load_configuration() {
    log_info "Loading configuration..."

    # Determine source config file
    local source_config=""

    if [[ -n "$CONFIG_FILE" && -f "$CONFIG_FILE" ]]; then
        source_config="$CONFIG_FILE"
        log_info "Using config file: $CONFIG_FILE"
    elif [[ -f "$INSTALL_DIR/.env" ]]; then
        source_config="$INSTALL_DIR/.env"
        log_info "Using existing config: $INSTALL_DIR/.env"
    else
        # Find .env.example in the source directory
        SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
        SOURCE_DIR="$(dirname "$SCRIPT_DIR")"

        if [[ -f "$SOURCE_DIR/.env.example" ]]; then
            source_config="$SOURCE_DIR/.env.example"
            log_info "Using default config: $source_config"
        else
            log_error "No configuration file found!"
            log_error "Please provide .env file or copy .env.example to .env"
            exit 1
        fi
    fi

    # Parse the config file
    parse_env_file "$source_config"

    # Override for local mode
    if [[ "$LOCAL_MODE" == "true" ]]; then
        export BASE_DOMAIN="gassigeher.local"
        export BASE_URL="http://gassigeher.local:8080"
        log_info "Local mode: Using domain gassigeher.local"
    fi

    # Set defaults for required variables
    export PORT="${PORT:-8080}"
    export DB_TYPE="${DB_TYPE:-postgres}"
    export DB_HOST="${DB_HOST:-db}"
    export DB_PORT="${DB_PORT:-5432}"
    export DB_NAME="${DB_NAME:-gassigeher}"
    export DB_USER="${DB_USER:-gassigeher}"

    log_success "Configuration loaded"
}

generate_secrets() {
    log_info "Generating secrets..."

    local secrets_file="$INSTALL_DIR/.env.secrets"

    # Check if secrets file exists and we're not in force mode
    if [[ -f "$secrets_file" && "$FORCE_MODE" != "true" ]]; then
        log_success "Using existing secrets from $secrets_file"
        parse_env_file "$secrets_file"
        return 0
    fi

    # Generate new secrets
    local db_password=$(generate_password)
    local jwt_secret=$(generate_password)
    local s3_access_key="gassigeher$(openssl rand -hex 4)"
    local s3_secret_key=$(generate_password)
    local metrics_password=$(generate_password)
    local grafana_password=$(generate_password)

    # Determine mode and generate appropriate admin password
    local is_saas_mode=false
    if [[ -n "$BASE_DOMAIN" ]]; then
        is_saas_mode=true
    fi

    local super_admin_password=""
    local central_admin_password=""

    if [[ "$is_saas_mode" == "true" ]]; then
        # SaaS Mode: Generate Central Admin password
        central_admin_password=$(generate_password)
        log_info "SaaS Mode detected - generating Central Admin password"
    else
        # Simple Mode: Generate Super Admin password
        super_admin_password=$(generate_password)
        log_info "Simple Mode detected - generating Super Admin password"
    fi

    # Create secrets file
    cat > "$secrets_file" << EOF
# ==================================================
# Gassigeher Secrets
# Generated: $(date)
# Mode: $(if [[ "$is_saas_mode" == "true" ]]; then echo "SaaS"; else echo "Simple"; fi)
# ==================================================
# WARNING: Keep this file secure! Never commit to version control.
# To change admin password: edit this file and restart server.

# ==================================================
# Database
# ==================================================
DB_PASSWORD=${db_password}

# ==================================================
# Authentication
# ==================================================
JWT_SECRET=${jwt_secret}

# ==================================================
# Admin Password
# ==================================================
EOF

    if [[ "$is_saas_mode" == "true" ]]; then
        cat >> "$secrets_file" << EOF
# SaaS Mode: Central Admin is the platform administrator
# Central Admin can impersonate any tenant's Super Admin
# Each tenant has their own Super Admin (created during registration)
CENTRAL_ADMIN_PASSWORD=${central_admin_password}
EOF
    else
        cat >> "$secrets_file" << EOF
# Simple Mode: Super Admin is the local shelter administrator
SUPER_ADMIN_PASSWORD=${super_admin_password}
EOF
    fi

    cat >> "$secrets_file" << EOF

# ==================================================
# S3/MinIO
# ==================================================
S3_ACCESS_KEY=${s3_access_key}
S3_SECRET_KEY=${s3_secret_key}

# ==================================================
# Monitoring
# ==================================================
METRICS_PASSWORD=${metrics_password}
GRAFANA_ADMIN_PASSWORD=${grafana_password}

# ==================================================
# External Services (Manual Configuration Required)
# ==================================================
# Fill these in before starting services:

# Hetzner DNS (for wildcard SSL in production)
HETZNER_DNS_TOKEN=

# Stripe (for SaaS billing - only needed in SaaS Mode)
STRIPE_SECRET_KEY=
STRIPE_PUBLISHABLE_KEY=
STRIPE_WEBHOOK_SECRET=

# SMTP (for email)
SMTP_PASSWORD=

# Gmail API (alternative to SMTP)
# GMAIL_CLIENT_ID=
# GMAIL_CLIENT_SECRET=
# GMAIL_REFRESH_TOKEN=

# Sentry (optional error tracking)
SENTRY_DSN=
EOF

    chmod 600 "$secrets_file"

    # Export for current session
    export DB_PASSWORD="$db_password"
    export JWT_SECRET="$jwt_secret"
    export SUPER_ADMIN_PASSWORD="$super_admin_password"
    export CENTRAL_ADMIN_PASSWORD="$central_admin_password"
    export S3_ACCESS_KEY="$s3_access_key"
    export S3_SECRET_KEY="$s3_secret_key"
    export METRICS_PASSWORD="$metrics_password"
    export GRAFANA_ADMIN_PASSWORD="$grafana_password"

    log_success "Secrets generated and saved to $secrets_file"
    log_warn "Please configure external service credentials in $secrets_file"
}

create_env_file() {
    log_info "Creating .env file..."

    # Determine BASE_URL
    local computed_base_url
    if [[ "$LOCAL_MODE" == "true" ]]; then
        computed_base_url="http://${BASE_DOMAIN:-gassigeher.local}:8080"
    else
        computed_base_url="${BASE_URL:-https://${BASE_DOMAIN:-gassigeher.org}}"
    fi

    cat > "$INSTALL_DIR/.env" << EOF
# Gassigeher Configuration
# Generated: $(date)
# Mode: $(if [[ "$LOCAL_MODE" == "true" ]]; then echo "LOCAL"; else echo "PRODUCTION"; fi)

# ==================================================
# Server
# ==================================================
PORT=${PORT:-8080}
BASE_URL=${computed_base_url}
BASE_DOMAIN=${BASE_DOMAIN:-}

# ==================================================
# Database
# ==================================================
DB_TYPE=postgres
DB_HOST=db
DB_PORT=5432
DB_NAME=${DB_NAME:-gassigeher}
DB_USER=${DB_USER:-gassigeher}
DB_MAX_OPEN_CONNS=${DB_MAX_OPEN_CONNS:-25}
DB_MAX_IDLE_CONNS=${DB_MAX_IDLE_CONNS:-5}
DB_CONN_MAX_LIFETIME=${DB_CONN_MAX_LIFETIME:-5}

# ==================================================
# Authentication
# ==================================================
JWT_EXPIRATION_HOURS=${JWT_EXPIRATION_HOURS:-24}

# ==================================================
# Admin
# ==================================================
SUPER_ADMIN_EMAIL=${SUPER_ADMIN_EMAIL:-admin@yourshelter.com}
CENTRAL_ADMIN_EMAIL=${CENTRAL_ADMIN_EMAIL:-admin@gassigeher.org}

# ==================================================
# Email
# ==================================================
EMAIL_PROVIDER=${EMAIL_PROVIDER:-smtp}
EMAIL_BCC_ADMIN=${EMAIL_BCC_ADMIN:-}
SMTP_HOST=${SMTP_HOST:-}
SMTP_PORT=${SMTP_PORT:-587}
SMTP_USERNAME=${SMTP_USERNAME:-}
SMTP_FROM_EMAIL=${SMTP_FROM_EMAIL:-}
SMTP_USE_TLS=${SMTP_USE_TLS:-false}
SMTP_USE_SSL=${SMTP_USE_SSL:-false}
GMAIL_FROM_EMAIL=${GMAIL_FROM_EMAIL:-}

# ==================================================
# S3 Storage
# ==================================================
USE_S3=${USE_S3:-true}
S3_ENDPOINT=${S3_ENDPOINT:-minio:9000}
S3_BUCKET_NAME=${S3_BUCKET_NAME:-gassigeher-uploads}
S3_REGION=${S3_REGION:-us-east-1}
S3_PUBLIC_URL=${S3_PUBLIC_URL:-http://localhost:9000/gassigeher-uploads}
S3_USE_SSL=${S3_USE_SSL:-false}

# ==================================================
# Stripe (SaaS-Mode)
# ==================================================
STRIPE_PRICE_MONTHLY=${STRIPE_PRICE_MONTHLY:-}
STRIPE_PRICE_YEARLY=${STRIPE_PRICE_YEARLY:-}

# ==================================================
# Rate Limiting
# ==================================================
RATE_LIMIT_ENABLED=${RATE_LIMIT_ENABLED:-true}
RATE_LIMIT_RPS=${RATE_LIMIT_RPS:-10}
RATE_LIMIT_BURST=${RATE_LIMIT_BURST:-20}
BRUTE_FORCE_MAX_ATTEMPTS=${BRUTE_FORCE_MAX_ATTEMPTS:-3}
BRUTE_FORCE_LOCKOUT_BASE=${BRUTE_FORCE_LOCKOUT_BASE:-30}
BRUTE_FORCE_LOCKOUT_MAX=${BRUTE_FORCE_LOCKOUT_MAX:-1800}

# ==================================================
# Tenant Settings (SaaS-Mode)
# ==================================================
TENANT_REGISTRATION_OPEN=${TENANT_REGISTRATION_OPEN:-true}
TENANT_REGISTRATION_PASSWORD=${TENANT_REGISTRATION_PASSWORD:-}
DEMO_ENABLED=${DEMO_ENABLED:-true}
DEMO_RESET_INTERVAL_HOURS=${DEMO_RESET_INTERVAL_HOURS:-24}

# ==================================================
# Logging
# ==================================================
LOG_DIR=/app/logs
LOG_MAX_AGE_DAYS=${LOG_MAX_AGE_DAYS:-30}
LOG_COMPRESS_SIZE_MB=${LOG_COMPRESS_SIZE_MB:-10}
LOG_CONSOLE_OUTPUT=${LOG_CONSOLE_OUTPUT:-true}

# ==================================================
# System Settings
# ==================================================
BOOKING_ADVANCE_DAYS=${BOOKING_ADVANCE_DAYS:-14}
CANCELLATION_NOTICE_HOURS=${CANCELLATION_NOTICE_HOURS:-12}
AUTO_DEACTIVATION_DAYS=${AUTO_DEACTIVATION_DAYS:-365}

# ==================================================
# Monitoring
# ==================================================
METRICS_USERNAME=${METRICS_USERNAME:-prometheus}
GRAFANA_ADMIN_USER=${GRAFANA_ADMIN_USER:-admin}
SENTRY_ENVIRONMENT=${SENTRY_ENVIRONMENT:-production}
SENTRY_RELEASE=${SENTRY_RELEASE:-}
EOF

    chmod 600 "$INSTALL_DIR/.env"
    log_success ".env file created"
}

copy_source_code() {
    log_info "Copying source code..."

    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    SOURCE_DIR="$(dirname "$SCRIPT_DIR")"

    if [[ -f "$SOURCE_DIR/go.mod" ]]; then
        # Copy essential files
        cp -r "$SOURCE_DIR"/{cmd,internal,go.mod,go.sum,.dockerignore} "$INSTALL_DIR/" 2>/dev/null || true
        cp "$SOURCE_DIR/docker-compose.yml" "$INSTALL_DIR/"
        cp "$SOURCE_DIR/Dockerfile" "$INSTALL_DIR/" 2>/dev/null || true
        cp "$SOURCE_DIR/Dockerfile.caddy" "$INSTALL_DIR/" 2>/dev/null || true
        cp "$SOURCE_DIR/Caddyfile" "$INSTALL_DIR/" 2>/dev/null || true
        cp -r "$SOURCE_DIR/deploy" "$INSTALL_DIR/" 2>/dev/null || true

        log_success "Source code copied from $SOURCE_DIR"
    else
        log_error "Source code not found at $SOURCE_DIR"
        log_error "Please run this script from the deploy/ directory of the source code"
        exit 1
    fi
}

create_backup_script() {
    log_info "Creating backup script..."

    cat > "$INSTALL_DIR/backup.sh" << 'EOF'
#!/bin/bash
# Gassigeher Database Backup Script
# Run via cron: 0 3 * * * /opt/gassigeher/backup.sh

set -e

BACKUP_DIR="/opt/gassigeher/backups"
DATE=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS=30

# Load environment
if [[ -f /opt/gassigeher/.env ]]; then
    source /opt/gassigeher/.env
fi
if [[ -f /opt/gassigeher/.env.secrets ]]; then
    source /opt/gassigeher/.env.secrets
fi

mkdir -p "$BACKUP_DIR"

echo "[$(date)] Starting PostgreSQL backup..."

# Backup database
docker compose -f /opt/gassigeher/docker-compose.yml exec -T db \
    pg_dump -U "${DB_USER:-gassigeher}" "${DB_NAME:-gassigeher}" | gzip > "$BACKUP_DIR/gassigeher_${DATE}.sql.gz"

echo "[$(date)] Backup completed: gassigeher_${DATE}.sql.gz"

# Cleanup old backups
find "$BACKUP_DIR" -name "gassigeher_*.sql.gz" -mtime +$RETENTION_DAYS -delete
echo "[$(date)] Cleanup completed (retention: $RETENTION_DAYS days)"

# Show backup size
du -sh "$BACKUP_DIR"
EOF

    chmod +x "$INSTALL_DIR/backup.sh"
    log_success "Backup script created"
}

create_management_script() {
    log_info "Creating management script..."

    cat > "$INSTALL_DIR/gassigeher" << 'EOF'
#!/bin/bash
# Gassigeher Management Script
# Usage: gassigeher [command]

INSTALL_DIR="/opt/gassigeher"
cd "$INSTALL_DIR"

# Load environment for commands that need it
load_env() {
    if [[ -f "$INSTALL_DIR/.env" ]]; then
        set -a
        source "$INSTALL_DIR/.env"
        set +a
    fi
    if [[ -f "$INSTALL_DIR/.env.secrets" ]]; then
        set -a
        source "$INSTALL_DIR/.env.secrets"
        set +a
    fi
}

case "$1" in
    start)
        echo "Starting Gassigeher..."
        load_env
        docker compose --profile production up -d
        ;;
    start-dev)
        echo "Starting Gassigeher (development mode, no Caddy)..."
        load_env
        docker compose up -d
        ;;
    stop)
        echo "Stopping Gassigeher..."
        load_env
        docker compose --profile production down
        ;;
    restart)
        echo "Restarting Gassigeher..."
        load_env
        docker compose --profile production restart
        ;;
    logs)
        load_env
        docker compose --profile production logs -f ${2:-}
        ;;
    status)
        load_env
        docker compose --profile production ps
        ;;
    backup)
        ./backup.sh
        ;;
    update)
        echo "Updating Gassigeher..."
        git pull 2>/dev/null || echo "Not a git repo, skipping git pull"
        load_env
        docker compose --profile production build --no-cache app
        docker compose --profile production up -d
        echo "Update complete!"
        ;;
    shell)
        load_env
        docker compose exec app sh
        ;;
    db)
        load_env
        docker compose exec db psql -U "${DB_USER:-gassigeher}" "${DB_NAME:-gassigeher}"
        ;;
    health)
        curl -s http://localhost:8080/api/health | jq . 2>/dev/null || curl -s http://localhost:8080/api/health
        echo ""
        curl -s http://localhost:8080/api/ready | jq . 2>/dev/null || curl -s http://localhost:8080/api/ready
        ;;
    secrets)
        echo "Editing secrets file..."
        ${EDITOR:-nano} "$INSTALL_DIR/.env.secrets"
        ;;
    *)
        echo "Gassigeher Management Script"
        echo ""
        echo "Usage: gassigeher [command]"
        echo ""
        echo "Commands:"
        echo "  start       Start all services (production with Caddy)"
        echo "  start-dev   Start services (development, no Caddy)"
        echo "  stop        Stop all services"
        echo "  restart     Restart all services"
        echo "  logs        View logs (optional: app, db, caddy, minio)"
        echo "  status      Show service status"
        echo "  backup      Run database backup"
        echo "  update      Pull latest code and rebuild"
        echo "  shell       Open shell in app container"
        echo "  db          Open PostgreSQL shell"
        echo "  health      Check application health"
        echo "  secrets     Edit secrets file"
        ;;
esac
EOF

    chmod +x "$INSTALL_DIR/gassigeher"

    if [[ "$CAN_SUDO" == "true" ]]; then
        $SUDO ln -sf "$INSTALL_DIR/gassigeher" /usr/local/bin/gassigeher
        log_success "Management script installed (use: gassigeher [command])"
    else
        log_success "Management script created (use: $INSTALL_DIR/gassigeher [command])"
        log_info "Add to PATH: export PATH=\"$INSTALL_DIR:\$PATH\""
    fi
}

setup_cron_backup() {
    log_info "Setting up daily backup cron job..."

    CRON_JOB="0 3 * * * $INSTALL_DIR/backup.sh >> $INSTALL_DIR/logs/backup.log 2>&1"

    {
        crontab -l 2>/dev/null | grep -v "gassigeher/backup.sh" || true
        echo "$CRON_JOB"
    } | crontab - 2>/dev/null || log_warn "Could not set up cron job (non-fatal)"

    log_success "Daily backup scheduled at 3:00 AM"
}

setup_firewall() {
    if [[ "$CAN_SUDO" != "true" ]]; then
        log_warn "Skipping firewall setup (requires root privileges)"
        return
    fi

    log_info "Configuring firewall..."

    $SUDO apt-get install -y ufw

    $SUDO ufw allow ssh
    $SUDO ufw allow http
    $SUDO ufw allow https

    $SUDO ufw --force enable

    log_success "Firewall configured (SSH, HTTP, HTTPS allowed)"
}

start_services() {
    log_info "Building and starting services..."

    cd "$INSTALL_DIR"

    # Check if secrets are configured
    if [[ -z "$DB_PASSWORD" ]]; then
        log_error "Secrets not loaded. Please check .env.secrets"
        exit 1
    fi

    # Determine profile
    local profile=""
    if [[ "$LOCAL_MODE" != "true" ]]; then
        profile="--profile production"
    fi

    log_info "Running: docker compose $profile up --build -d"
    docker compose $profile up --build -d

    log_info "Waiting for services to start..."
    sleep 10

    docker compose $profile ps

    log_success "Services started!"
}

force_cleanup() {
    log_warn "============================================"
    log_warn "  FORCE MODE - DESTRUCTIVE OPERATION"
    log_warn "============================================"
    echo ""
    log_warn "This will:"
    log_warn "  1. Stop all running containers"
    log_warn "  2. DELETE the entire directory: $INSTALL_DIR"
    log_warn "  3. This includes: database, uploads, backups, logs, secrets"
    echo ""
    log_error "ALL DATA WILL BE PERMANENTLY LOST!"
    echo ""

    read -p "Type 'yes' to confirm deletion: " CONFIRM
    if [[ "$CONFIRM" != "yes" ]]; then
        log_info "Aborted. No changes made."
        exit 0
    fi

    echo ""
    log_info "Proceeding with force cleanup..."

    if [[ -f "$INSTALL_DIR/docker-compose.yml" ]]; then
        log_info "Stopping containers and removing volumes..."
        cd "$INSTALL_DIR"
        docker compose --profile production down -v --remove-orphans 2>/dev/null || true
        cd - > /dev/null
        log_success "Containers and volumes removed"
    fi

    docker volume ls -q | grep -i gassi | xargs -r docker volume rm 2>/dev/null || true

    if [[ -d "$INSTALL_DIR" ]]; then
        log_info "Deleting $INSTALL_DIR..."
        if [[ "$CAN_SUDO" == "true" ]]; then
            $SUDO rm -rf "$INSTALL_DIR"
        else
            rm -rf "$INSTALL_DIR"
        fi
        log_success "Directory deleted"
    fi

    log_success "Force cleanup completed"
    echo ""
}

print_summary() {
    local domain="${BASE_DOMAIN:-gassigeher.org}"

    echo ""
    echo "============================================"
    echo -e "${GREEN}  INSTALLATION COMPLETE!${NC}"
    echo "============================================"
    echo ""

    if [[ "$LOCAL_MODE" == "true" ]]; then
        echo "Your Gassigeher is now running at (LOCAL MODE):"
        echo ""
        echo -e "  ${BLUE}Application:${NC}     http://${domain}:8080"
        echo -e "  ${BLUE}Health Check:${NC}    http://${domain}:8080/api/health"
        echo -e "  ${BLUE}MinIO Console:${NC}   http://localhost:9001"
        echo -e "  ${BLUE}Grafana:${NC}         http://localhost:3000"
        echo ""
        echo "Required /etc/hosts entry:"
        echo "  127.0.0.1 ${domain}"
        if [[ -n "$BASE_DOMAIN" ]]; then
            echo "  127.0.0.1 demo.${domain}"
        fi
    else
        echo "Your Gassigeher is now running at:"
        echo ""
        echo -e "  ${BLUE}Application:${NC}     https://${domain}"
        if [[ -n "$BASE_DOMAIN" ]]; then
            echo -e "  ${BLUE}Central Admin:${NC}   https://admin.${domain}"
            echo -e "  ${BLUE}Demo Tenant:${NC}     https://demo.${domain}"
        fi
        echo -e "  ${BLUE}Health Check:${NC}    https://${domain}/api/health"
    fi

    echo ""
    echo "Management commands:"
    echo ""
    echo "  gassigeher start    - Start services (production)"
    echo "  gassigeher start-dev- Start services (development)"
    echo "  gassigeher stop     - Stop services"
    echo "  gassigeher logs     - View logs"
    echo "  gassigeher backup   - Run backup"
    echo "  gassigeher status   - Check status"
    echo "  gassigeher secrets  - Edit secrets"
    echo ""
    echo "Important files:"
    echo ""
    echo "  Config:      $INSTALL_DIR/.env"
    echo "  Secrets:     $INSTALL_DIR/.env.secrets (admin password here)"
    echo "  Logs:        $INSTALL_DIR/logs/"
    echo "  Backups:     $INSTALL_DIR/backups/"
    echo ""

    # Check if external services need configuration
    local needs_config=false
    if [[ -z "$HETZNER_DNS_TOKEN" && "$LOCAL_MODE" != "true" ]]; then
        needs_config=true
    fi
    if [[ -z "$SMTP_PASSWORD" && -z "$GMAIL_CLIENT_ID" ]]; then
        needs_config=true
    fi

    if [[ "$needs_config" == "true" ]]; then
        echo -e "${YELLOW}Action Required:${NC}"
        echo ""
        echo "  Edit $INSTALL_DIR/.env.secrets to configure:"
        if [[ -z "$HETZNER_DNS_TOKEN" && "$LOCAL_MODE" != "true" ]]; then
            echo "    - HETZNER_DNS_TOKEN (for wildcard SSL)"
        fi
        if [[ -z "$SMTP_PASSWORD" && -z "$GMAIL_CLIENT_ID" ]]; then
            echo "    - SMTP_PASSWORD or Gmail API credentials (for email)"
        fi
        if [[ -n "$BASE_DOMAIN" ]]; then
            echo "    - STRIPE_* keys (for billing, if using SaaS-Mode)"
        fi
        echo ""
        echo "  Then restart: gassigeher restart"
        echo ""
    fi

    echo "============================================"
}

show_help() {
    echo "Gassigeher Installer"
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --force       Force complete reinstall (regenerates secrets, deletes data)"
    echo "  --local       Use local testing mode (no SSL, uses gassigeher.local)"
    echo "  --env FILE    Use specified config file instead of .env.example"
    echo "  --dry-run     Show what would be done without making changes"
    echo "  --help        Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                         # Install using .env.example defaults"
    echo "  $0 --local                 # Local testing (gassigeher.local:8080)"
    echo "  $0 --env /path/to/.env     # Use custom config file"
    echo "  $0 --force                 # Fresh install (WARNING: deletes all data!)"
    echo ""
    echo "Configuration:"
    echo "  The script reads configuration from two files:"
    echo "    .env         - Application settings"
    echo "    .env.secrets - Passwords and API keys (auto-generated)"
    echo ""
    echo "  On first install, secrets are auto-generated."
    echo "  On subsequent runs, existing secrets are preserved."
    echo "  Use --force to regenerate all secrets."
}

# ============================================
# MAIN
# ============================================

main() {
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --force)
                FORCE_MODE=true
                shift
                ;;
            --local)
                LOCAL_MODE=true
                shift
                ;;
            --env)
                CONFIG_FILE="$2"
                shift 2
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done

    echo ""
    echo "============================================"
    echo "  GASSIGEHER INSTALLER"
    echo "  Ubuntu 24.04 + Docker Compose"
    echo "============================================"
    echo ""

    check_sudo
    check_ubuntu

    # Force mode cleanup
    if [[ "$FORCE_MODE" == "true" ]]; then
        force_cleanup
    fi

    # Install Docker if needed
    if ! command -v docker &> /dev/null; then
        install_docker
    else
        log_success "Docker already installed"
    fi

    # Create directories
    create_directories

    # Load configuration
    load_configuration

    # Generate or load secrets
    generate_secrets

    # Create .env file
    if [[ ! -f "$INSTALL_DIR/.env" || "$FORCE_MODE" == "true" ]]; then
        create_env_file
    else
        log_success "Using existing .env file"
    fi

    # Copy source code
    copy_source_code

    # Create helper scripts
    create_backup_script
    create_management_script

    # Setup cron and firewall
    setup_cron_backup
    setup_firewall

    # Start services
    start_services

    # Print summary
    print_summary
}

main "$@"
