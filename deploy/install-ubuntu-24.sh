#!/bin/bash
#
# Gassigeher SaaS - Production Deployment Script
# Target: Ubuntu 24.04 LTS with Docker Compose
# Domain: gassigeher.org (wildcard SSL via Hetzner DNS)
#
# Usage:
#   chmod +x install-ubuntu-24.sh
#   ./install-ubuntu-24.sh          # Runs without root (Docker must be pre-installed)
#   sudo ./install-ubuntu-24.sh     # Full install including Docker
#
# Prerequisites:
#   - Fresh Ubuntu 24.04 LTS server (Hetzner Cloud recommended)
#   - Domain gassigeher.org pointing to server IP
#   - Hetzner DNS API token (for wildcard SSL)
#   - Stripe API keys (for billing)
#   - SMTP credentials (for email)
#

set -e  # Exit on error
set -o pipefail

# ============================================
# CONFIGURATION - EDIT THESE VALUES
# ============================================

# Domain configuration (override with --local for local testing)
DOMAIN="gassigeher.org"
BASE_DOMAIN="gassigeher.org"
LOCAL_MODE=false

# Installation directory
INSTALL_DIR="/opt/gassigeher"

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
    # Check if we can use sudo (needed for some operations)
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
    if ! grep -q "Ubuntu" /etc/os-release; then
        log_error "This script is designed for Ubuntu 24.04"
        exit 1
    fi
    log_success "Ubuntu detected"
}

generate_password() {
    openssl rand -base64 32 | tr -d '/+=' | cut -c1-32
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
        $SUDO mkdir -p "$INSTALL_DIR"/{data/app,logs,backups,uploads}
        $SUDO chmod 750 "$INSTALL_DIR"
        # Make current user the owner if using sudo
        if [[ -n "$SUDO" ]]; then
            $SUDO chown -R "$(id -u):$(id -g)" "$INSTALL_DIR"
        fi
    else
        mkdir -p "$INSTALL_DIR"/{data/app,logs,backups,uploads}
        chmod 750 "$INSTALL_DIR"
    fi

    log_success "Directories created at $INSTALL_DIR"
}

collect_configuration() {
    log_info "Collecting configuration..."
    echo ""
    echo "============================================"
    echo "  GASSIGEHER SAAS CONFIGURATION"
    echo "============================================"
    echo ""

    # Generate secrets
    JWT_SECRET=$(generate_password)
    DB_PASSWORD=$(generate_password)

    # Hetzner DNS Token (required for wildcard SSL)
    echo -e "${YELLOW}Hetzner DNS API Token${NC}"
    echo "Get it from: https://dns.hetzner.com/settings/api-token"
    read -p "Enter Hetzner DNS API Token: " HETZNER_DNS_TOKEN
    if [[ -z "$HETZNER_DNS_TOKEN" ]]; then
        log_error "Hetzner DNS token is required for wildcard SSL"
        exit 1
    fi
    echo ""

    # Stripe Configuration
    echo -e "${YELLOW}Stripe Configuration${NC}"
    echo "Get keys from: https://dashboard.stripe.com/apikeys"
    read -p "Stripe Secret Key (sk_live_...): " STRIPE_SECRET_KEY
    read -p "Stripe Publishable Key (pk_live_...): " STRIPE_PUBLISHABLE_KEY
    read -p "Stripe Webhook Secret (whsec_...): " STRIPE_WEBHOOK_SECRET
    read -p "Stripe Price ID - Monthly (price_...): " STRIPE_PRICE_MONTHLY
    read -p "Stripe Price ID - Yearly (price_...): " STRIPE_PRICE_YEARLY
    echo ""

    # S3 Configuration (Hetzner Object Storage)
    echo -e "${YELLOW}Hetzner S3 Object Storage (optional - press Enter to skip)${NC}"
    echo "Create bucket at: https://console.hetzner.cloud/"
    read -p "Use S3 storage? (y/N): " USE_S3_INPUT
    if [[ "$USE_S3_INPUT" =~ ^[Yy]$ ]]; then
        USE_S3="true"
        read -p "S3 Endpoint (e.g., fsn1.your-objectstorage.com): " S3_ENDPOINT
        read -p "S3 Access Key: " S3_ACCESS_KEY
        read -p "S3 Secret Key: " S3_SECRET_KEY
        read -p "S3 Bucket Name: " S3_BUCKET_NAME
        read -p "S3 Region (e.g., fsn1): " S3_REGION
        read -p "S3 Public URL (e.g., https://bucket.fsn1.your-objectstorage.com): " S3_PUBLIC_URL
    else
        USE_S3="false"
        S3_ENDPOINT=""
        S3_ACCESS_KEY=""
        S3_SECRET_KEY=""
        S3_BUCKET_NAME=""
        S3_REGION=""
        S3_PUBLIC_URL=""
    fi
    echo ""

    # SMTP Configuration
    echo -e "${YELLOW}SMTP Email Configuration${NC}"
    echo "Examples: smtp.strato.de (port 465), smtp.office365.com (port 587)"
    read -p "SMTP Host: " SMTP_HOST
    read -p "SMTP Port [465]: " SMTP_PORT
    SMTP_PORT=${SMTP_PORT:-465}
    read -p "SMTP Username: " SMTP_USERNAME
    read -p "SMTP Password: " SMTP_PASSWORD
    read -p "SMTP From Email: " SMTP_FROM_EMAIL
    read -p "Use SSL? (Y/n): " SMTP_SSL_INPUT
    if [[ "$SMTP_SSL_INPUT" =~ ^[Nn]$ ]]; then
        SMTP_USE_SSL="false"
    else
        SMTP_USE_SSL="true"
    fi
    read -p "BCC Admin Email (optional): " EMAIL_BCC_ADMIN
    echo ""

    # Contact Email
    read -p "Contact form email [kontakt@$DOMAIN]: " CONTACT_EMAIL
    CONTACT_EMAIL=${CONTACT_EMAIL:-kontakt@$DOMAIN}

    # Super Admin Email (with validation)
    while true; do
        read -p "Super Admin email: " SUPER_ADMIN_EMAIL
        if [[ "$SUPER_ADMIN_EMAIL" =~ ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$ ]]; then
            break
        else
            log_error "Invalid email format. Please enter a valid email address."
        fi
    done
    echo ""

    log_success "Configuration collected"
}

create_env_file() {
    log_info "Creating .env file..."

    # Set BASE_URL based on mode
    if [[ "$LOCAL_MODE" == "true" ]]; then
        COMPUTED_BASE_URL="http://${DOMAIN}:8080"
    else
        COMPUTED_BASE_URL="https://${DOMAIN}"
    fi

    cat > "$INSTALL_DIR/.env" << EOF
# Gassigeher SaaS Configuration
# Generated: $(date)
# Mode: $(if [[ "$LOCAL_MODE" == "true" ]]; then echo "LOCAL"; else echo "PRODUCTION"; fi)

# ============================================
# DATABASE
# ============================================
DB_TYPE=postgres
DB_HOST=db
DB_PORT=5432
DB_NAME=gassigeher
DB_USER=gassigeher
DB_PASSWORD=${DB_PASSWORD}

# ============================================
# APPLICATION
# ============================================
PORT=8080
BASE_URL=${COMPUTED_BASE_URL}
BASE_DOMAIN=${BASE_DOMAIN}

# ============================================
# AUTHENTICATION
# ============================================
JWT_SECRET=${JWT_SECRET}
JWT_EXPIRATION_HOURS=24
SUPER_ADMIN_EMAIL=${SUPER_ADMIN_EMAIL}

# ============================================
# HETZNER DNS (for wildcard SSL)
# ============================================
HETZNER_DNS_TOKEN=${HETZNER_DNS_TOKEN}

# ============================================
# STRIPE BILLING
# ============================================
STRIPE_SECRET_KEY=${STRIPE_SECRET_KEY}
STRIPE_PUBLISHABLE_KEY=${STRIPE_PUBLISHABLE_KEY}
STRIPE_WEBHOOK_SECRET=${STRIPE_WEBHOOK_SECRET}
STRIPE_PRICE_MONTHLY=${STRIPE_PRICE_MONTHLY}
STRIPE_PRICE_YEARLY=${STRIPE_PRICE_YEARLY}

# ============================================
# S3 STORAGE (Hetzner Object Storage)
# ============================================
USE_S3=${USE_S3}
S3_ENDPOINT=${S3_ENDPOINT}
S3_ACCESS_KEY=${S3_ACCESS_KEY}
S3_SECRET_KEY=${S3_SECRET_KEY}
S3_BUCKET_NAME=${S3_BUCKET_NAME}
S3_REGION=${S3_REGION}
S3_PUBLIC_URL=${S3_PUBLIC_URL}

# ============================================
# EMAIL (SMTP)
# ============================================
EMAIL_PROVIDER=smtp
SMTP_HOST=${SMTP_HOST}
SMTP_PORT=${SMTP_PORT}
SMTP_USERNAME=${SMTP_USERNAME}
SMTP_PASSWORD=${SMTP_PASSWORD}
SMTP_FROM_EMAIL=${SMTP_FROM_EMAIL}
SMTP_USE_SSL=${SMTP_USE_SSL}
EMAIL_BCC_ADMIN=${EMAIL_BCC_ADMIN}
CONTACT_EMAIL=${CONTACT_EMAIL}

# ============================================
# LOGGING
# ============================================
LOG_DIR=/app/logs
LOG_MAX_AGE_DAYS=30
LOG_COMPRESS_SIZE_MB=10
LOG_CONSOLE_OUTPUT=true

# ============================================
# SYSTEM SETTINGS
# ============================================
BOOKING_ADVANCE_DAYS=14
CANCELLATION_NOTICE_HOURS=12
AUTO_DEACTIVATION_DAYS=365
EOF

    chmod 600 "$INSTALL_DIR/.env"
    log_success ".env file created"
}

create_dockerfile() {
    log_info "Creating Dockerfile..."

    cat > "$INSTALL_DIR/Dockerfile" << 'EOF'
# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o gassigeher ./cmd/server

# Production stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata wget

WORKDIR /app

COPY --from=builder /app/gassigeher .

RUN mkdir -p /app/uploads /app/logs

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1

CMD ["./gassigeher"]
EOF

    log_success "Dockerfile created"
}

create_caddy_dockerfile() {
    log_info "Creating Caddy Dockerfile with Hetzner DNS module..."

    cat > "$INSTALL_DIR/Dockerfile.caddy" << 'EOF'
# Custom Caddy with Hetzner DNS module for wildcard SSL
FROM caddy:2-builder AS builder

RUN xcaddy build \
    --with github.com/caddy-dns/hetzner

FROM caddy:2-alpine

COPY --from=builder /usr/bin/caddy /usr/bin/caddy
EOF

    log_success "Caddy Dockerfile created"
}

create_caddyfile() {
    log_info "Creating Caddyfile..."

    cat > "$INSTALL_DIR/Caddyfile" << 'EOF'
# Caddyfile for Gassigeher SaaS
# Wildcard SSL for *.gassigeher.org using Hetzner DNS challenge

# Main domain - Landing page
gassigeher.org {
    tls {
        dns hetzner {env.HETZNER_DNS_TOKEN}
    }
    reverse_proxy app:8080
    header {
        X-Frame-Options "DENY"
        X-Content-Type-Options "nosniff"
        X-XSS-Protection "1; mode=block"
        Strict-Transport-Security "max-age=31536000; includeSubDomains"
        Referrer-Policy "strict-origin-when-cross-origin"
    }
    encode gzip zstd
    log {
        output file /var/log/caddy/gassigeher.log
        format json
    }
}

# WWW redirect
www.gassigeher.org {
    redir https://gassigeher.org{uri} permanent
}

# Central admin dashboard
admin.gassigeher.org {
    tls {
        dns hetzner {env.HETZNER_DNS_TOKEN}
    }
    reverse_proxy app:8080
    header {
        X-Frame-Options "DENY"
        X-Content-Type-Options "nosniff"
        Strict-Transport-Security "max-age=31536000; includeSubDomains"
    }
    encode gzip zstd
    log {
        output file /var/log/caddy/admin.log
        format json
    }
}

# Wildcard for all tenant subdomains
*.gassigeher.org {
    tls {
        dns hetzner {env.HETZNER_DNS_TOKEN}
    }
    reverse_proxy app:8080
    header {
        X-Frame-Options "DENY"
        X-Content-Type-Options "nosniff"
        X-XSS-Protection "1; mode=block"
        Strict-Transport-Security "max-age=31536000; includeSubDomains"
        Referrer-Policy "strict-origin-when-cross-origin"
    }
    encode gzip zstd
    log {
        output file /var/log/caddy/tenants.log
        format json
    }
}
EOF

    log_success "Caddyfile created"
}

create_docker_compose() {
    log_info "Copying docker-compose.yml..."
    cp "$INSTALL_DIR/docker-compose.yml" "$INSTALL_DIR/docker-compose.yml.bak" 2>/dev/null || true
    # docker-compose.yml is already in the repo, no need to generate
    log_success "docker-compose.yml ready"
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

source /opt/gassigeher/.env

mkdir -p "$BACKUP_DIR"

echo "[$(date)] Starting PostgreSQL backup..."

# Backup database
docker compose -f /opt/gassigeher/docker-compose.yml exec -T db \
    pg_dump -U "$DB_USER" "$DB_NAME" | gzip > "$BACKUP_DIR/gassigeher_${DATE}.sql.gz"

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

case "$1" in
    start)
        echo "Starting Gassigeher..."
        docker compose --profile production up -d
        ;;
    stop)
        echo "Stopping Gassigeher..."
        docker compose --profile production down
        ;;
    restart)
        echo "Restarting Gassigeher..."
        docker compose --profile production restart
        ;;
    logs)
        docker compose --profile production logs -f ${2:-}
        ;;
    status)
        docker compose --profile production ps
        ;;
    backup)
        ./backup.sh
        ;;
    update)
        echo "Updating Gassigeher..."
        git pull 2>/dev/null || echo "Not a git repo, skipping git pull"
        docker compose --profile production build --no-cache app
        docker compose --profile production up -d
        echo "Update complete!"
        ;;
    shell)
        docker compose --profile production exec app sh
        ;;
    db)
        docker compose --profile production exec db psql -U gassigeher gassigeher
        ;;
    health)
        curl -s http://localhost:8080/api/health | jq .
        curl -s http://localhost:8080/api/ready | jq .
        ;;
    *)
        echo "Gassigeher Management Script"
        echo ""
        echo "Usage: gassigeher [command]"
        echo ""
        echo "Commands:"
        echo "  start     Start all services"
        echo "  stop      Stop all services"
        echo "  restart   Restart all services"
        echo "  logs      View logs (optional: app, db, caddy)"
        echo "  status    Show service status"
        echo "  backup    Run database backup"
        echo "  update    Pull latest code and rebuild"
        echo "  shell     Open shell in app container"
        echo "  db        Open PostgreSQL shell"
        echo "  health    Check application health"
        ;;
esac
EOF

    chmod +x "$INSTALL_DIR/gassigeher"

    # Create symlink in /usr/local/bin if we have sudo access
    if [[ "$CAN_SUDO" == "true" ]]; then
        $SUDO ln -sf "$INSTALL_DIR/gassigeher" /usr/local/bin/gassigeher
        log_success "Management script created (use: gassigeher [command])"
    else
        log_success "Management script created (use: $INSTALL_DIR/gassigeher [command])"
        log_info "Add to PATH: export PATH=\"$INSTALL_DIR:\$PATH\""
    fi
}

setup_cron_backup() {
    log_info "Setting up daily backup cron job..."

    # Add cron job for daily backup at 3 AM
    CRON_JOB="0 3 * * * $INSTALL_DIR/backup.sh >> $INSTALL_DIR/logs/backup.log 2>&1"

    # Get existing crontab (without our job), add our job, then install
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

    # Install ufw if not present
    $SUDO apt-get install -y ufw

    # Allow SSH, HTTP, HTTPS
    $SUDO ufw allow ssh
    $SUDO ufw allow http
    $SUDO ufw allow https

    # Enable firewall
    $SUDO ufw --force enable

    log_success "Firewall configured (SSH, HTTP, HTTPS allowed)"
}

copy_source_code() {
    log_info "Copying source code..."

    # Get the directory where this script is located
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    SOURCE_DIR="$(dirname "$SCRIPT_DIR")"

    if [[ -f "$SOURCE_DIR/go.mod" ]]; then
        cp -r "$SOURCE_DIR"/{cmd,internal,go.mod,go.sum,docker-compose.yml,.dockerignore} "$INSTALL_DIR/"
        log_success "Source code copied from $SOURCE_DIR"
    else
        log_warn "Source code not found. You'll need to copy it manually."
        echo "  cp -r /path/to/gassigeher/{cmd,internal,go.mod,go.sum,docker-compose.yml} $INSTALL_DIR/"
    fi
}

start_services() {
    log_info "Building and starting services..."

    cd "$INSTALL_DIR"

    # Build and start with production profile (includes Caddy for SSL)
    log_info "Running: docker compose --profile production up --build -d"
    docker compose --profile production up --build -d

    # Wait for services to be healthy
    log_info "Waiting for services to start..."
    sleep 10

    # Check status
    docker compose ps

    log_success "Services started!"
}

print_summary() {
    echo ""
    echo "============================================"
    echo -e "${GREEN}  INSTALLATION COMPLETE!${NC}"
    echo "============================================"
    echo ""

    if [[ "$LOCAL_MODE" == "true" ]]; then
        echo "Your Gassigeher SaaS is now running at (LOCAL MODE):"
        echo ""
        echo -e "  ${BLUE}Landing Page:${NC}    http://${DOMAIN}:8080/landing/"
        echo -e "  ${BLUE}Central Admin:${NC}   http://${DOMAIN}:8080/central/"
        echo -e "  ${BLUE}Demo Tenant:${NC}     http://demo.${DOMAIN}:8080"
        echo -e "  ${BLUE}Health Check:${NC}    http://${DOMAIN}:8080/api/health"
        echo ""
        echo "Required /etc/hosts entries:"
        echo "  127.0.0.1 ${DOMAIN} demo.${DOMAIN}"
    else
        echo "Your Gassigeher SaaS is now running at:"
        echo ""
        echo -e "  ${BLUE}Landing Page:${NC}    https://${DOMAIN}"
        echo -e "  ${BLUE}Central Admin:${NC}   https://admin.${DOMAIN}"
        echo -e "  ${BLUE}Demo Tenant:${NC}     https://demo.${DOMAIN}"
        echo -e "  ${BLUE}Health Check:${NC}    https://${DOMAIN}/api/health"
    fi
    echo ""
    echo "Management commands:"
    echo ""
    echo "  gassigeher start    - Start services"
    echo "  gassigeher stop     - Stop services"
    echo "  gassigeher logs     - View logs"
    echo "  gassigeher backup   - Run backup"
    echo "  gassigeher update   - Update application"
    echo "  gassigeher status   - Check status"
    echo ""
    echo "Important files:"
    echo ""
    echo "  Config:   $INSTALL_DIR/.env"
    echo "  Logs:     $INSTALL_DIR/logs/"
    echo "  Backups:  $INSTALL_DIR/backups/"
    echo "  Data:     $INSTALL_DIR/data/"
    echo ""
    echo "Super Admin credentials will be in:"
    echo "  $INSTALL_DIR/logs/ (check app logs)"
    echo ""
    if [[ "$LOCAL_MODE" != "true" ]]; then
        echo "Next steps:"
        echo ""
        echo "  1. Configure Stripe webhook:"
        echo "     URL: https://${DOMAIN}/api/v1/billing/webhook"
        echo ""
        echo "  2. Configure Hetzner DNS:"
        echo "     A record:     ${DOMAIN} -> $(curl -s ifconfig.me)"
        echo "     A record:   *.${DOMAIN} -> $(curl -s ifconfig.me)"
        echo ""
    fi
    echo "  Test the installation:"
    echo "     gassigeher health"
    echo ""
    echo "============================================"
}

# ============================================
# MAIN INSTALLATION
# ============================================

show_help() {
    echo "Gassigeher SaaS Installer"
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --force       Force complete reinstall (stops containers, deletes target dir, rebuilds)"
    echo "  --local       Use gassigeher.local domain for local testing (no SSL)"
    echo "  --help        Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                    # Production install (gassigeher.org)"
    echo "  $0 --local            # Local testing (gassigeher.local:8080)"
    echo "  $0 --force --local    # Fresh local install"
    echo ""
    echo "The script is idempotent - safe to run multiple times."
    echo "Existing .env configuration is preserved unless --force is used."
    echo ""
    echo "WARNING: --force will DELETE all data including database, uploads, and backups!"
}

force_cleanup() {
    log_warn "============================================"
    log_warn "  FORCE MODE - DESTRUCTIVE OPERATION"
    log_warn "============================================"
    echo ""
    log_warn "This will:"
    log_warn "  1. Stop all running containers"
    log_warn "  2. DELETE the entire directory: $INSTALL_DIR"
    log_warn "  3. This includes: database, uploads, backups, logs, .env"
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

    # Stop containers and remove volumes if docker-compose.yml exists
    if [[ -f "$INSTALL_DIR/docker-compose.yml" ]]; then
        log_info "Stopping containers and removing volumes..."
        cd "$INSTALL_DIR"
        docker compose --profile production down -v --remove-orphans || true
        cd - > /dev/null
        log_success "Containers and volumes removed"
    else
        log_info "No docker-compose.yml found, skipping container stop"
    fi

    # Also prune any orphaned volumes with gassigeher in the name
    log_info "Cleaning up orphaned Docker volumes..."
    docker volume ls -q | grep -i gassi | xargs -r docker volume rm 2>/dev/null || true

    # Delete the entire target directory
    if [[ -d "$INSTALL_DIR" ]]; then
        log_info "Deleting $INSTALL_DIR..."
        if [[ "$CAN_SUDO" == "true" ]]; then
            $SUDO rm -rf "$INSTALL_DIR"
        else
            rm -rf "$INSTALL_DIR"
        fi
        log_success "Directory deleted"
    else
        log_info "Directory does not exist, nothing to delete"
    fi

    log_success "Force cleanup completed"
    echo ""
}

main() {
    # Parse arguments
    FORCE_MODE=false
    for arg in "$@"; do
        case $arg in
            --force)
                FORCE_MODE=true
                ;;
            --local)
                LOCAL_MODE=true
                DOMAIN="gassigeher.local"
                BASE_DOMAIN="gassigeher.local"
                log_info "Local mode: Using domain gassigeher.local"
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
        esac
    done

    echo ""
    echo "============================================"
    echo "  GASSIGEHER SAAS INSTALLER"
    echo "  Ubuntu 24.04 + Docker Compose"
    echo "============================================"
    echo ""

    check_sudo
    check_ubuntu

    # Force mode: cleanup before installation
    if [[ "$FORCE_MODE" == "true" ]]; then
        force_cleanup
    fi

    # Install Docker
    if ! command -v docker &> /dev/null; then
        install_docker
    else
        log_success "Docker already installed"
    fi

    # Create directories
    create_directories

    # Configuration handling (idempotent)
    if [[ -f "$INSTALL_DIR/.env" ]]; then
        log_success "Existing .env found - preserving configuration"
        log_info "Use --force to do a complete reinstall"
        # Source existing env to get variables for summary
        source "$INSTALL_DIR/.env"
    else
        # Collect configuration (fresh install or after --force cleanup)
        collect_configuration
        # Create .env file
        create_env_file
    fi

    # Create/update configuration files (always safe to overwrite)
    create_dockerfile
    create_caddy_dockerfile
    create_caddyfile
    create_docker_compose
    create_backup_script
    create_management_script

    # Copy source code (overwrites are safe)
    copy_source_code

    # Setup cron and firewall (idempotent)
    setup_cron_backup
    setup_firewall

    # Start services
    start_services

    # Print summary
    print_summary
}

# Run main function
main "$@"
