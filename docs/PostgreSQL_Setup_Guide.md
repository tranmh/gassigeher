# PostgreSQL Setup Guide for Gassigeher

**Purpose:** Step-by-step guide to configure Gassigeher with PostgreSQL
**Difficulty:** Medium-High
**Time Required:** 30-45 minutes
**Last Updated:** 2025-01-22

---

## Prerequisites

- PostgreSQL 12+ (recommended: 15+)
- Administrative access (postgres user)
- Basic knowledge of PostgreSQL commands

---

## Quick Start (Docker - Recommended for Development)

```bash
# 1. Start PostgreSQL container
docker run --name gassigeher-postgres \
  -e POSTGRES_DB=gassigeher \
  -e POSTGRES_USER=gassigeher_user \
  -e POSTGRES_PASSWORD=gassigeher_pass \
  -p 5432:5432 \
  -d postgres:15

# 2. Configure Gassigeher (.env)
DB_TYPE=postgres
DB_HOST=localhost
DB_PORT=5432
DB_NAME=gassigeher
DB_USER=gassigeher_user
DB_PASSWORD=gassigeher_pass
DB_SSLMODE=disable

# 3. Run Gassigeher
go run cmd/server/main.go

# Done! Tables created automatically
```

---

## Production Setup (Step-by-Step)

### Step 1: Install PostgreSQL

#### Ubuntu/Debian:
```bash
# Add PostgreSQL repository
sudo sh -c 'echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list'
wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | sudo apt-key add -

# Install PostgreSQL
sudo apt update
sudo apt install postgresql-15

# Start service
sudo systemctl start postgresql
sudo systemctl enable postgresql
```

#### CentOS/RHEL:
```bash
sudo yum install postgresql-server postgresql-contrib
sudo postgresql-setup initdb
sudo systemctl start postgresql
sudo systemctl enable postgresql
```

#### macOS:
```bash
brew install postgresql@15
brew services start postgresql@15
```

#### Windows:
Download installer from: https://www.postgresql.org/download/windows/

---

### Step 2: Create Database and User

```bash
# Switch to postgres user
sudo -u postgres psql

# Or on Windows/Mac:
psql -U postgres
```

```sql
-- Create user
CREATE USER gassigeher_user WITH PASSWORD 'your_secure_password';

-- Create database with correct encoding
CREATE DATABASE gassigeher
  WITH OWNER gassigeher_user
  ENCODING 'UTF8'
  LC_COLLATE = 'en_US.UTF-8'
  LC_CTYPE = 'en_US.UTF-8'
  TEMPLATE template0;

-- Grant all privileges
GRANT ALL PRIVILEGES ON DATABASE gassigeher TO gassigeher_user;

-- Connect to database
\c gassigeher

-- Grant schema privileges (PostgreSQL 15+)
GRANT ALL ON SCHEMA public TO gassigeher_user;

-- Verify
\l  -- List databases
\du -- List users

-- Exit
\q
```

**Important:** Replace `'your_secure_password'` with a strong password!

---

### Step 3: Configure PostgreSQL for Network Access (If Needed)

**For remote connections:**

#### Edit postgresql.conf
```bash
sudo nano /etc/postgresql/15/main/postgresql.conf
```

Find and change:
```ini
listen_addresses = '*'  # Listen on all interfaces
# Or specific IP: listen_addresses = '192.168.1.10'
```

#### Edit pg_hba.conf
```bash
sudo nano /etc/postgresql/15/main/pg_hba.conf
```

Add at the end:
```
# Allow connections from specific IP
host    gassigeher    gassigeher_user    192.168.1.0/24    md5

# Or allow from anywhere (less secure)
host    gassigeher    gassigeher_user    0.0.0.0/0         md5
```

#### Restart PostgreSQL
```bash
sudo systemctl restart postgresql
```

---

### Step 4: Test Connection

```bash
# Test connection as gassigeher_user
psql -h localhost -U gassigeher_user -d gassigeher

# If successful, you should see:
# gassigeher=>

# Exit
\q
```

---

### Step 5: Configure Gassigeher

Create or edit `.env` file:

```bash
# Database Configuration
DB_TYPE=postgres
DB_HOST=localhost
DB_PORT=5432
DB_NAME=gassigeher
DB_USER=gassigeher_user
DB_PASSWORD=your_secure_password
DB_SSLMODE=disable  # Use 'require' or 'verify-full' for production

# Connection Pool (optional, these are defaults)
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5

# ... other configuration ...
```

---

### Step 6: Run Gassigeher

```bash
# Run application
go run cmd/server/main.go

# Expected output:
# 2025/01/22 12:00:00 Using database: postgres
# 2025/01/22 12:00:00 Applying migration: 001_create_users_table
# ... (9 migrations)
# 2025/01/22 12:00:00 Applied 9 migration(s)
# 2025/01/22 12:00:00 Server starting on port 8080...
```

---

### Step 7: Verify Database Setup

```bash
# Connect to PostgreSQL
psql -h localhost -U gassigeher_user -d gassigeher

# Check tables created
\dt

# Should show:
#              List of relations
#  Schema |         Name          | Type  |     Owner
# --------+-----------------------+-------+----------------
#  public | blocked_dates         | table | gassigeher_user
#  public | bookings              | table | gassigeher_user
#  public | dogs                  | table | gassigeher_user
#  public | experience_requests   | table | gassigeher_user
#  public | reactivation_requests | table | gassigeher_user
#  public | schema_migrations     | table | gassigeher_user
#  public | system_settings       | table | gassigeher_user
#  public | users                 | table | gassigeher_user

# Check migration status
SELECT * FROM schema_migrations ORDER BY applied_at;

# Should show 9 migrations applied

# Check table schema
\d users

# Exit
\q
```

---

## SSL/TLS Configuration (Production)

### Enable SSL in PostgreSQL

#### 1. Generate SSL Certificates

```bash
# Navigate to PostgreSQL data directory
cd /var/lib/postgresql/15/main

# Generate private key
openssl genrsa -out server.key 2048
chmod 600 server.key
chown postgres:postgres server.key

# Generate certificate
openssl req -new -key server.key -out server.csr
openssl x509 -req -in server.csr -signkey server.key -out server.crt -days 365

# Set permissions
chmod 600 server.crt
chown postgres:postgres server.crt
```

#### 2. Edit postgresql.conf

```ini
ssl = on
ssl_cert_file = 'server.crt'
ssl_key_file = 'server.key'
```

#### 3. Restart PostgreSQL

```bash
sudo systemctl restart postgresql
```

#### 4. Configure Gassigeher

```bash
DB_SSLMODE=require  # Or verify-full for maximum security
```

---

## Cloud PostgreSQL

### AWS RDS for PostgreSQL

**1. Create RDS Instance:**
- Go to AWS RDS Console
- Create database → PostgreSQL
- Select version 15
- Choose instance size (t3.micro for dev, t3.small+ for production)
- Set master username and password
- Configure VPC and security groups
- Create database

**2. Get Connection Details:**
- Endpoint: `gassigeher.xxxxx.us-east-1.rds.amazonaws.com`
- Port: `5432`
- Username: Your master username
- Password: Your master password
- Database: `postgres` (default)

**3. Create Gassigeher Database:**
```bash
psql -h gassigeher.xxxxx.us-east-1.rds.amazonaws.com -U postgres

CREATE DATABASE gassigeher WITH ENCODING 'UTF8';
CREATE USER gassigeher_user WITH PASSWORD 'password';
GRANT ALL PRIVILEGES ON DATABASE gassigeher TO gassigeher_user;
\c gassigeher
GRANT ALL ON SCHEMA public TO gassigeher_user;
\q
```

**4. Configure Gassigeher:**
```bash
DB_TYPE=postgres
DB_HOST=gassigeher.xxxxx.us-east-1.rds.amazonaws.com
DB_PORT=5432
DB_NAME=gassigeher
DB_USER=gassigeher_user
DB_PASSWORD=your_password
DB_SSLMODE=require
```

---

### DigitalOcean Managed PostgreSQL

**1. Create Managed Database:**
- Go to DigitalOcean → Databases
- Create → PostgreSQL 15
- Choose datacenter and size
- Create cluster

**2. Get Connection Details:**
- Host: `db-postgresql-xxx.ondigitalocean.com`
- Port: `25060`
- User: `doadmin`
- Password: (shown in UI)
- Database: `defaultdb`

**3. Create Gassigeher Database:**
```bash
psql -h db-postgresql-xxx.ondigitalocean.com -p 25060 -U doadmin defaultdb

CREATE DATABASE gassigeher;
CREATE USER gassigeher_user WITH PASSWORD 'password';
GRANT ALL PRIVILEGES ON DATABASE gassigeher TO gassigeher_user;
\c gassigeher
GRANT ALL ON SCHEMA public TO gassigeher_user;
\q
```

**4. Configure Gassigeher:**
```bash
DB_TYPE=postgres
DB_CONNECTION_STRING=postgres://gassigeher_user:password@db-postgresql-xxx.ondigitalocean.com:25060/gassigeher?sslmode=require
```

---

## Connection String Format

### Standard Format

```
postgres://username:password@host:port/database?parameters
```

### Full Example

```
postgres://gassigeher_user:mypassword@localhost:5432/gassigeher?sslmode=disable
```

### SSL Mode Options

- `disable` - No SSL (development only)
- `require` - SSL required (production recommended)
- `verify-ca` - Verify certificate authority
- `verify-full` - Full SSL verification (most secure)

### Additional Parameters

- `connect_timeout=10` - Connection timeout (seconds)
- `application_name=gassigeher` - Application name in logs
- `pool_max_conns=25` - Max connections (handled by app)

---

## Troubleshooting

### "FATAL: password authentication failed"

**Problem:** Wrong username or password

**Solution:**
```sql
ALTER USER gassigeher_user WITH PASSWORD 'new_password';
```

---

### "FATAL: database 'gassigeher' does not exist"

**Problem:** Database not created

**Solution:**
```sql
CREATE DATABASE gassigeher WITH ENCODING 'UTF8';
```

---

### "could not connect to server"

**Problem:** PostgreSQL not running or firewall blocking

**Solutions:**
```bash
# Check if running
sudo systemctl status postgresql

# Start PostgreSQL
sudo systemctl start postgresql

# Check if listening
sudo netstat -plnt | grep 5432

# Check firewall
sudo ufw allow 5432/tcp
```

---

### "permission denied for schema public"

**Problem:** User doesn't have schema permissions (PostgreSQL 15+)

**Solution:**
```sql
\c gassigeher postgres
GRANT ALL ON SCHEMA public TO gassigeher_user;
```

---

### "SSL connection required"

**Problem:** Server requires SSL but client configured with sslmode=disable

**Solution:**
```bash
DB_SSLMODE=require  # Change in .env
```

---

## Performance Tuning

### Recommended PostgreSQL Configuration

Edit `/etc/postgresql/15/main/postgresql.conf`:

```ini
# Memory Settings
shared_buffers = 256MB          # 25% of RAM (for 1GB RAM)
effective_cache_size = 768MB    # 75% of RAM
maintenance_work_mem = 64MB
work_mem = 16MB

# Checkpoint Settings
checkpoint_completion_target = 0.9
wal_buffers = 16MB

# Connection Settings
max_connections = 100

# Logging (optional, for debugging)
log_statement = 'mod'           # Log modifications
log_duration = on               # Log query duration
log_min_duration_statement = 100  # Log slow queries (>100ms)
```

**Restart after changes:**
```bash
sudo systemctl restart postgresql
```

---

## Backup and Restore

### Backup Database

```bash
# Full backup (custom format, recommended)
pg_dump -h localhost -U gassigeher_user -Fc gassigeher > gassigeher_backup_$(date +%Y%m%d).dump

# SQL format backup
pg_dump -h localhost -U gassigeher_user gassigeher > gassigeher_backup_$(date +%Y%m%d).sql

# Compressed SQL backup
pg_dump -h localhost -U gassigeher_user gassigeher | gzip > gassigeher_backup_$(date +%Y%m%d).sql.gz

# Automated daily backup (crontab)
0 3 * * * pg_dump -h localhost -U gassigeher_user -Fc gassigeher > /backups/gassigeher_$(date +\%Y\%m\%d).dump
```

### Restore Database

```bash
# Restore from custom format
pg_restore -h localhost -U gassigeher_user -d gassigeher gassigeher_backup_20250122.dump

# Restore from SQL format
psql -h localhost -U gassigeher_user -d gassigeher < gassigeher_backup_20250122.sql

# Restore from compressed
gunzip < gassigeher_backup_20250122.sql.gz | psql -h localhost -U gassigeher_user -d gassigeher
```

---

## Migration from SQLite

See [Database_Migration_Guide.md](Database_Migration_Guide.md) for complete migration procedures.

**Quick overview:**
1. Export data from SQLite
2. Transform SQL for PostgreSQL
3. Import into PostgreSQL
4. Update .env to use PostgreSQL
5. Verify application works

---

## Monitoring

### Check Database Size

```sql
SELECT
  pg_database.datname AS database_name,
  pg_size_pretty(pg_database_size(pg_database.datname)) AS size
FROM pg_database
WHERE datname = 'gassigeher';
```

### Check Table Sizes

```sql
SELECT
  schemaname,
  tablename,
  pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

### Check Active Connections

```sql
SELECT
  count(*) AS connection_count,
  state
FROM pg_stat_activity
WHERE datname = 'gassigeher'
GROUP BY state;
```

### Check Slow Queries

```sql
SELECT
  pid,
  now() - pg_stat_activity.query_start AS duration,
  query,
  state
FROM pg_stat_activity
WHERE state != 'idle'
  AND now() - pg_stat_activity.query_start > interval '5 seconds'
ORDER BY duration DESC;
```

---

## Security Recommendations

### 1. Use Strong Passwords

```bash
# Generate secure password
openssl rand -base64 32
```

### 2. Configure pg_hba.conf Properly

```bash
sudo nano /etc/postgresql/15/main/pg_hba.conf
```

**Development (local only):**
```
local   gassigeher    gassigeher_user    md5
host    gassigeher    gassigeher_user    127.0.0.1/32    md5
```

**Production (specific IPs):**
```
host    gassigeher    gassigeher_user    192.168.1.0/24    md5
```

**Never use (insecure):**
```
host    all    all    0.0.0.0/0    trust  # DON'T DO THIS!
```

### 3. Use SSL/TLS in Production

```bash
DB_SSLMODE=require  # Or verify-full
```

### 4. Regular Security Updates

```bash
sudo apt update
sudo apt upgrade postgresql-15
```

### 5. Limit Privileges

```sql
-- Don't grant SUPERUSER unless needed
-- gassigeher_user should only have access to gassigeher database
REVOKE ALL ON DATABASE postgres FROM gassigeher_user;
```

---

## Docker Compose (Recommended for Development)

**File:** `docker-compose.yml`

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    container_name: gassigeher-postgres
    environment:
      POSTGRES_DB: gassigeher
      POSTGRES_USER: gassigeher_user
      POSTGRES_PASSWORD: gassigeher_pass
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./postgres-init:/docker-entrypoint-initdb.d  # Optional: init scripts
    command:
      - postgres
      - -c
      - timezone=UTC

  gassigeher:
    build: .
    ports:
      - "8080:8080"
    depends_on:
      - postgres
    environment:
      DB_TYPE: postgres
      DB_HOST: postgres
      DB_PORT: 5432
      DB_NAME: gassigeher
      DB_USER: gassigeher_user
      DB_PASSWORD: gassigeher_pass
      DB_SSLMODE: disable

volumes:
  postgres_data:
```

**Usage:**
```bash
docker-compose up -d
```

---

## Verification Checklist

After setup, verify:

- [ ] PostgreSQL server is running
  ```bash
  sudo systemctl status postgresql
  ```

- [ ] Database exists
  ```sql
  \l gassigeher
  ```

- [ ] User has correct privileges
  ```sql
  \du gassigeher_user
  ```

- [ ] Encoding is UTF8
  ```sql
  SELECT encoding, datcollate, datctype
  FROM pg_database
  WHERE datname = 'gassigeher';
  ```

- [ ] Gassigeher connects successfully
  ```bash
  go run cmd/server/main.go
  # Should see: "Using database: postgres"
  ```

- [ ] All tables created
  ```sql
  \c gassigeher
  \dt
  # Should show 8 tables
  ```

- [ ] Application accessible
  ```bash
  curl http://localhost:8080/
  # Should return HTML
  ```

---

## Common Issues

### Issue: Encoding/Locale Problems

**Solution:** Recreate database with correct encoding
```sql
DROP DATABASE gassigeher;  -- Backup first!
CREATE DATABASE gassigeher
  WITH ENCODING 'UTF8'
  LC_COLLATE = 'en_US.UTF-8'
  LC_CTYPE = 'en_US.UTF-8'
  TEMPLATE template0;
```

### Issue: Slow Queries

**Solution:** Analyze and optimize
```sql
-- Check query performance
EXPLAIN ANALYZE SELECT * FROM users WHERE email = 'test@example.com';

-- Reindex if needed
REINDEX DATABASE gassigeher;

-- Update statistics
ANALYZE;
```

### Issue: Connection Pool Exhausted

**Solution:** Increase pool size in .env:
```bash
DB_MAX_OPEN_CONNS=50
DB_MAX_IDLE_CONNS=10
```

Or increase PostgreSQL max_connections:
```ini
# postgresql.conf
max_connections = 200
```

---

## PostgreSQL-Specific Features (Optional)

### Full-Text Search

```sql
-- Create full-text search index on dog names/breeds
CREATE INDEX idx_dogs_fulltext ON dogs USING gin(to_tsvector('english', name || ' ' || breed));

-- Search
SELECT * FROM dogs
WHERE to_tsvector('english', name || ' ' || breed) @@ to_tsquery('english', 'labrador');
```

### JSON Columns (Future Enhancement)

```sql
-- PostgreSQL has excellent JSON support
-- Could store dog metadata as JSON in future
ALTER TABLE dogs ADD COLUMN metadata JSONB;
CREATE INDEX idx_dogs_metadata ON dogs USING gin(metadata);
```

---

## Maintenance

### Weekly Tasks

- Check disk space: `df -h`
- Check database size (see Monitoring section)
- Review log files: `tail -100 /var/log/postgresql/postgresql-15-main.log`

### Monthly Tasks

- Backup database
- Vacuum analyze: `VACUUM ANALYZE;`
- Check for updates: `sudo apt list --upgradable | grep postgresql`

### Quarterly Tasks

- Review and optimize slow queries
- Analyze table statistics
- Plan capacity upgrades if needed

### Routine Maintenance Commands

```sql
-- Analyze all tables (update statistics)
ANALYZE;

-- Vacuum and analyze (reclaim space and update stats)
VACUUM ANALYZE;

-- Check for table bloat
SELECT schemaname, tablename, pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename))
FROM pg_tables WHERE schemaname = 'public' ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

---

## Uninstall / Cleanup

### Remove PostgreSQL (if needed)

```bash
# Backup first!
pg_dump -h localhost -U gassigeher_user -Fc gassigeher > final_backup.dump

# Drop database
sudo -u postgres psql
DROP DATABASE gassigeher;
DROP USER gassigeher_user;
\q

# Optionally remove PostgreSQL server
sudo apt remove postgresql-15
```

### Remove Docker Container

```bash
docker stop gassigeher-postgres
docker rm gassigeher-postgres
docker volume rm gassigeher_postgres_data  # Removes data permanently!
```

---

## Advanced: Replication (High Availability)

### Primary-Replica Setup

**Use Case:** High availability, read scaling

**Setup:**
1. Configure primary server
2. Create replica server
3. Set up streaming replication
4. Configure Gassigeher to use primary for writes, replicas for reads

**Complexity:** High - consult PostgreSQL documentation

---

## Performance Comparison

### PostgreSQL vs SQLite for Gassigeher

| Operation | SQLite | PostgreSQL |
|-----------|--------|------------|
| **Simple SELECT** | 0.1-0.5ms | 0.5-2ms |
| **Complex JOIN** | 1-5ms | 2-8ms |
| **INSERT** | 0.5-1ms | 1-3ms |
| **Concurrent Writes** | Limited | Excellent |

**For Gassigeher workload:** PostgreSQL is better for concurrent access and larger deployments.

---

## Next Steps

- **Completed PostgreSQL Setup?** See [Database_Migration_Guide.md](Database_Migration_Guide.md) to migrate data from SQLite
- **Performance optimization?** See DEPLOYMENT.md
- **Questions?** See [Database_Selection_Guide.md](Database_Selection_Guide.md)

---

**Your PostgreSQL database is ready for Gassigeher!** 🎉
