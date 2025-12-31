#!/bin/bash
#
# Gassigeher SaaS - Production Deployment Script
# Target: Ubuntu 24.04 LTS with Docker Compose
# Domain: gassigeher.org (wildcard SSL via Hetzner DNS)
#
# Usage:
#   chmod +x install-ubuntu-24.sh
#   sudo ./install-ubuntu-24.sh
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

# Domain configuration
DOMAIN="gassigeher.org"
BASE_DOMAIN="gassigeher.org"

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

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
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
    log_info "Installing Docker..."

    # Remove old versions
    apt-get remove -y docker docker-engine docker.io containerd runc 2>/dev/null || true

    # Install prerequisites
    apt-get update
    apt-get install -y ca-certificates curl gnupg lsb-release

    # Add Docker GPG key
    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    chmod a+r /etc/apt/keyrings/docker.gpg

    # Add Docker repository
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null

    # Install Docker
    apt-get update
    apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

    # Start Docker
    systemctl enable docker
    systemctl start docker

    log_success "Docker installed successfully"
}

create_directories() {
    log_info "Creating directories..."

    mkdir -p "$INSTALL_DIR"/{data,logs,backups,uploads}
    chmod 750 "$INSTALL_DIR"

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

    # Super Admin Email
    read -p "Super Admin email: " SUPER_ADMIN_EMAIL
    echo ""

    log_success "Configuration collected"
}

create_env_file() {
    log_info "Creating .env file..."

    cat > "$INSTALL_DIR/.env" << EOF
# Gassigeher SaaS Production Configuration
# Generated: $(date)

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
BASE_URL=https://${DOMAIN}
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
    log_info "Creating docker-compose.yml..."

    cat > "$INSTALL_DIR/docker-compose.yml" << 'EOF'
# Gassigeher SaaS Production Stack
# PostgreSQL + Caddy (wildcard SSL) + Go App

services:
  app:
    build:
      context: .
      dockerfile: Dockerfile
    restart: unless-stopped
    env_file: .env
    depends_on:
      db:
        condition: service_healthy
    volumes:
      - ./uploads:/app/uploads
      - ./logs:/app/logs
    networks:
      - internal
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/api/health"]
      interval: 30s
      timeout: 3s
      start_period: 10s
      retries: 3

  db:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      POSTGRES_DB: ${DB_NAME:-gassigeher}
      POSTGRES_USER: ${DB_USER:-gassigeher}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - ./data/postgres:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER:-gassigeher} -d ${DB_NAME:-gassigeher}"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - internal

  caddy:
    build:
      context: .
      dockerfile: Dockerfile.caddy
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - ./data/caddy:/data
      - ./data/caddy-config:/config
      - ./logs/caddy:/var/log/caddy
    environment:
      HETZNER_DNS_TOKEN: ${HETZNER_DNS_TOKEN}
    depends_on:
      app:
        condition: service_healthy
    networks:
      - internal
      - external

networks:
  internal:
    driver: bridge
  external:
    driver: bridge
EOF

    log_success "docker-compose.yml created"
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
        docker compose up -d
        ;;
    stop)
        echo "Stopping Gassigeher..."
        docker compose down
        ;;
    restart)
        echo "Restarting Gassigeher..."
        docker compose restart
        ;;
    logs)
        docker compose logs -f ${2:-}
        ;;
    status)
        docker compose ps
        ;;
    backup)
        ./backup.sh
        ;;
    update)
        echo "Updating Gassigeher..."
        git pull 2>/dev/null || echo "Not a git repo, skipping git pull"
        docker compose build --no-cache app
        docker compose up -d
        echo "Update complete!"
        ;;
    shell)
        docker compose exec app sh
        ;;
    db)
        docker compose exec db psql -U gassigeher gassigeher
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
    ln -sf "$INSTALL_DIR/gassigeher" /usr/local/bin/gassigeher
    log_success "Management script created (use: gassigeher [command])"
}

setup_cron_backup() {
    log_info "Setting up daily backup cron job..."

    # Add cron job for daily backup at 3 AM
    (crontab -l 2>/dev/null | grep -v "gassigeher/backup.sh"; echo "0 3 * * * $INSTALL_DIR/backup.sh >> $INSTALL_DIR/logs/backup.log 2>&1") | crontab -

    log_success "Daily backup scheduled at 3:00 AM"
}

setup_firewall() {
    log_info "Configuring firewall..."

    # Install ufw if not present
    apt-get install -y ufw

    # Allow SSH, HTTP, HTTPS
    ufw allow ssh
    ufw allow http
    ufw allow https

    # Enable firewall
    ufw --force enable

    log_success "Firewall configured (SSH, HTTP, HTTPS allowed)"
}

copy_source_code() {
    log_info "Copying source code..."

    # Get the directory where this script is located
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    SOURCE_DIR="$(dirname "$SCRIPT_DIR")"

    if [[ -f "$SOURCE_DIR/go.mod" ]]; then
        cp -r "$SOURCE_DIR"/{cmd,internal,go.mod,go.sum} "$INSTALL_DIR/"
        log_success "Source code copied from $SOURCE_DIR"
    else
        log_warn "Source code not found. You'll need to copy it manually."
        echo "  cp -r /path/to/gassigeher/{cmd,internal,go.mod,go.sum} $INSTALL_DIR/"
    fi
}

start_services() {
    log_info "Building and starting services..."

    cd "$INSTALL_DIR"

    # Build and start
    docker compose build
    docker compose up -d

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
    echo "Your Gassigeher SaaS is now running at:"
    echo ""
    echo -e "  ${BLUE}Landing Page:${NC}    https://${DOMAIN}"
    echo -e "  ${BLUE}Central Admin:${NC}   https://admin.${DOMAIN}"
    echo -e "  ${BLUE}Demo Tenant:${NC}     https://demo.${DOMAIN}"
    echo -e "  ${BLUE}Health Check:${NC}    https://${DOMAIN}/api/health"
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
    echo "Next steps:"
    echo ""
    echo "  1. Configure Stripe webhook:"
    echo "     URL: https://${DOMAIN}/api/v1/billing/webhook"
    echo ""
    echo "  2. Configure Hetzner DNS:"
    echo "     A record:     ${DOMAIN} -> $(curl -s ifconfig.me)"
    echo "     A record:   *.${DOMAIN} -> $(curl -s ifconfig.me)"
    echo ""
    echo "  3. Test the installation:"
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
    echo "  --force       Force reconfiguration (regenerate .env with new passwords)"
    echo "  --help        Show this help message"
    echo ""
    echo "The script is idempotent - safe to run multiple times."
    echo "Existing .env configuration is preserved unless --force is used."
}

main() {
    # Parse arguments
    FORCE_RECONFIGURE=false
    for arg in "$@"; do
        case $arg in
            --force)
                FORCE_RECONFIGURE=true
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

    check_root
    check_ubuntu

    # Install Docker
    if ! command -v docker &> /dev/null; then
        install_docker
    else
        log_success "Docker already installed"
    fi

    # Create directories
    create_directories

    # Configuration handling (idempotent)
    if [[ -f "$INSTALL_DIR/.env" ]] && [[ "$FORCE_RECONFIGURE" == "false" ]]; then
        log_success "Existing .env found - preserving configuration"
        log_info "Use --force to reconfigure with new credentials"
        # Source existing env to get variables for summary
        source "$INSTALL_DIR/.env"
    else
        if [[ -f "$INSTALL_DIR/.env" ]]; then
            log_warn "Existing .env will be overwritten (--force specified)"
            # Backup existing .env
            cp "$INSTALL_DIR/.env" "$INSTALL_DIR/.env.backup.$(date +%Y%m%d_%H%M%S)"
            log_info "Backup created: .env.backup.$(date +%Y%m%d_%H%M%S)"
        fi
        # Collect configuration
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
EOF
