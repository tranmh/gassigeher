#!/bin/bash
#
# Dry-run simulation of install-ubuntu-24.sh
# Shows what WOULD be executed without actually doing it
#

set -e

BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo ""
echo "============================================"
echo "  GASSIGEHER INSTALLER - DRY RUN SIMULATION"
echo "============================================"
echo ""

simulate() {
    echo -e "${YELLOW}[WOULD EXECUTE]${NC} $1"
}

# Simulated installation steps
echo -e "${BLUE}=== Step 1: Check Prerequisites ===${NC}"
simulate "check if running as root"
simulate "check if Ubuntu OS"
echo ""

echo -e "${BLUE}=== Step 2: Install Docker (if not present) ===${NC}"
simulate "apt-get update"
simulate "apt-get install -y ca-certificates curl gnupg lsb-release"
simulate "curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor"
simulate "apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin"
simulate "systemctl enable docker && systemctl start docker"
echo ""

echo -e "${BLUE}=== Step 3: Create Directories ===${NC}"
simulate "mkdir -p /opt/gassigeher/{data/app,data/postgres,data/minio,logs,backups,uploads}"
simulate "chmod 750 /opt/gassigeher"
echo ""

echo -e "${BLUE}=== Step 4: Load Configuration ===${NC}"
echo -e "${GREEN}[AUTO]${NC} Read configuration from .env.example (or .env if exists)"
echo -e "${GREEN}[AUTO]${NC} No user prompts - all settings from config file"
echo ""

echo -e "${BLUE}=== Step 5: Generate Secrets (auto-generated) ===${NC}"
simulate "openssl rand -base64 32  # DB_PASSWORD"
simulate "openssl rand -base64 32  # JWT_SECRET"
simulate "openssl rand -base64 32  # SUPER_ADMIN_PASSWORD"
simulate "openssl rand -base64 32  # CENTRAL_ADMIN_PASSWORD"
simulate "openssl rand -base64 32  # S3_SECRET_KEY"
simulate "openssl rand -base64 32  # METRICS_PASSWORD"
simulate "openssl rand -base64 32  # GRAFANA_ADMIN_PASSWORD"
echo -e "${GREEN}[AUTO]${NC} Secrets saved to /opt/gassigeher/.env.secrets"
echo -e "${GREEN}[AUTO]${NC} Admin credentials saved to /opt/gassigeher/SUPER_ADMIN_CREDENTIALS.txt"
echo ""

echo -e "${BLUE}=== Step 6: Create Configuration Files ===${NC}"
simulate "cat > /opt/gassigeher/.env           # Application config"
simulate "cat > /opt/gassigeher/.env.secrets   # Auto-generated secrets"
echo ""

echo -e "${BLUE}=== Step 7: Copy Source Code ===${NC}"
simulate "cp -r {cmd,internal,go.mod,go.sum,docker-compose.yml} /opt/gassigeher/"
echo ""

echo -e "${BLUE}=== Step 8: Create Helper Scripts ===${NC}"
simulate "cat > /opt/gassigeher/backup.sh      # Database backup"
simulate "cat > /opt/gassigeher/gassigeher     # Management script"
simulate "ln -sf /opt/gassigeher/gassigeher /usr/local/bin/gassigeher"
echo ""

echo -e "${BLUE}=== Step 9: Setup Cron & Firewall ===${NC}"
simulate "crontab: 0 3 * * * /opt/gassigeher/backup.sh"
simulate "ufw allow ssh"
simulate "ufw allow http"
simulate "ufw allow https"
simulate "ufw --force enable"
echo ""

echo -e "${BLUE}=== Step 10: Build & Start Services ===${NC}"
simulate "cd /opt/gassigeher && docker compose --profile production up --build -d"
echo ""

echo -e "${BLUE}=== Files That Would Be Created ===${NC}"
echo "/opt/gassigeher/"
echo "├── .env                    # Configuration (from .env.example)"
echo "├── .env.secrets            # Auto-generated secrets (NEVER commit!)"
echo "├── SUPER_ADMIN_CREDENTIALS.txt  # Admin passwords"
echo "├── docker-compose.yml      # Service definitions"
echo "├── gassigeher              # Management script"
echo "├── backup.sh               # Backup script"
echo "├── go.mod, go.sum          # Go dependencies"
echo "├── cmd/                    # Go source"
echo "├── internal/               # Go source"
echo "├── data/"
echo "│   ├── postgres/           # PostgreSQL data"
echo "│   ├── minio/              # MinIO/S3 data"
echo "│   ├── caddy/              # SSL certificates"
echo "│   └── caddy-config/       # Caddy config"
echo "├── logs/                   # Application logs"
echo "│   └── caddy/              # Caddy logs"
echo "├── backups/                # Database backups"
echo "└── uploads/                # User uploads"
echo ""

echo -e "${BLUE}=== Docker Containers ===${NC}"
echo "┌─────────────────────────────────────────────────────────────────┐"
echo "│  Container           │  Image                    │  Resources  │"
echo "├─────────────────────────────────────────────────────────────────┤"
echo "│  gassigeher-app      │  gassigeher:latest        │  0.5 CPU, 512MB │"
echo "│  gassigeher-db       │  postgres:16.4-alpine     │  0.5 CPU, 1GB   │"
echo "│  gassigeher-minio    │  minio/minio:RELEASE.*    │  0.25 CPU, 512MB│"
echo "│  gassigeher-prometheus│ prom/prometheus:v2.54.1  │  0.25 CPU, 256MB│"
echo "│  gassigeher-grafana  │  grafana/grafana:11.2.2   │  0.25 CPU, 256MB│"
echo "│  gassigeher-caddy    │  caddy:2 + hetzner (prod) │  0.25 CPU, 128MB│"
echo "└─────────────────────────────────────────────────────────────────┘"
echo ""
echo "Total resource limits: ~2 CPU, ~2.7GB RAM (fits 2 vCPU / 4GB server)"
echo ""

echo -e "${BLUE}=== Security Improvements ===${NC}"
echo "✓ No hardcoded passwords in docker-compose.yml"
echo "✓ Secrets auto-generated and stored in .env.secrets"
echo "✓ .env.secrets excluded from version control"
echo "✓ Resource limits prevent container resource exhaustion"
echo "✓ Logging limits prevent disk space exhaustion (10MB x 3 files)"
echo "✓ Graceful shutdown with stop_grace_period"
echo "✓ Pinned image versions for reproducibility"
echo ""

echo -e "${BLUE}=== Configuration Files ===${NC}"
echo "Config (.env):          Application settings, safe to version control"
echo "Secrets (.env.secrets): Passwords/keys, auto-generated, NEVER commit"
echo ""
echo "External services to configure in .env.secrets:"
echo "  - HETZNER_DNS_TOKEN    (for wildcard SSL)"
echo "  - STRIPE_* keys        (for SaaS billing)"
echo "  - SMTP_PASSWORD        (for email)"
echo ""

echo -e "${BLUE}=== URLs After Installation ===${NC}"
echo "Application:      https://gassigeher.org"
echo "Central Admin:    https://admin.gassigeher.org (SaaS-Mode)"
echo "Demo Tenant:      https://demo.gassigeher.org (SaaS-Mode)"
echo "Health Check:     https://gassigeher.org/api/health"
echo "MinIO Console:    http://localhost:9001"
echo "Grafana:          http://localhost:3000"
echo ""

echo -e "${GREEN}=== Dry Run Complete ===${NC}"
echo ""
echo "To actually install, run on your production server:"
echo "  sudo ./deploy/install-ubuntu-24.sh"
echo ""
echo "For local testing:"
echo "  ./deploy/install-ubuntu-24.sh --local"
echo ""
echo "For fresh install (regenerates all secrets):"
echo "  ./deploy/install-ubuntu-24.sh --force"
echo ""
