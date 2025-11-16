# Gassigeher - Production Deployment Guide

**Complete step-by-step guide for deploying Gassigeher to a production Linux server.**

**Status**: ✅ Deployment package ready | systemd service | nginx config | SSL setup | Automated backups

> **Prerequisites**: Ubuntu 22.04 LTS, root access, domain name, Gmail API credentials
> **Deployment Time**: ~1-2 hours for complete setup
> **Quick Links**: [README](README.md) | [API Docs](API.md) | [Admin Guide](ADMIN_GUIDE.md)

---

## Prerequisites

- Ubuntu 22.04 LTS (or similar Linux distribution)
- Root or sudo access
- Domain name pointing to your server
- Gmail account for email notifications

## Server Requirements

- **CPU**: 1 core minimum, 2+ cores recommended
- **RAM**: 512MB minimum, 1GB+ recommended
- **Disk**: 10GB minimum, 20GB+ recommended
- **Go**: 1.24 or higher
- **SQLite**: 3.35 or higher
- **nginx**: Latest stable version

## Step-by-Step Deployment

### 1. Server Setup

```bash
# Update system
sudo apt update
sudo apt upgrade -y

# Install required packages
sudo apt install -y golang sqlite3 nginx certbot python3-certbot-nginx git

# Verify Go installation
go version
```

### 2. Create Application User

```bash
# Create gassigeher user
sudo useradd -r -m -d /var/gassigeher -s /bin/bash gassigeher

# Create directory structure
sudo mkdir -p /var/gassigeher/{bin,data,uploads,logs,backups,config,frontend}
sudo chown -R gassigeher:gassigeher /var/gassigeher
```

### 3. Deploy Application Files

```bash
# Switch to gassigeher user
sudo su - gassigeher

# Clone repository (or upload files)
cd /var/gassigeher
git clone https://github.com/yourusername/gassigeher.git source
# OR upload via SCP/SFTP

# Build application
cd source
go build -o /var/gassigeher/bin/gassigeher ./cmd/server

# Copy frontend files
cp -r frontend/* /var/gassigeher/frontend/

# Copy deployment files
cp deploy/*.sh /var/gassigeher/

# Make scripts executable
chmod +x /var/gassigeher/*.sh
```

### 4. Configure Environment Variables

```bash
# Create .env file
sudo nano /var/gassigeher/config/.env
```

Copy and configure:

```bash
# Application
PORT=8080
ENVIRONMENT=production

# Database
DATABASE_PATH=/var/gassigeher/data/gassigeher.db

# JWT (Generate secure random string)
JWT_SECRET=your-super-secret-256-bit-random-string-here
JWT_EXPIRATION_HOURS=24

# Admin (Comma-separated admin emails)
ADMIN_EMAILS=admin@yourdomain.com

# Gmail API (from Google Cloud Console)
GMAIL_CLIENT_ID=your-client-id.apps.googleusercontent.com
GMAIL_CLIENT_SECRET=your-client-secret
GMAIL_REFRESH_TOKEN=your-refresh-token
GMAIL_FROM_EMAIL=noreply@yourdomain.com

# Uploads
UPLOAD_DIR=/var/gassigeher/uploads
MAX_UPLOAD_SIZE_MB=5

# System Settings (defaults)
BOOKING_ADVANCE_DAYS=14
CANCELLATION_NOTICE_HOURS=12
AUTO_DEACTIVATION_DAYS=365
```

**Secure the .env file:**
```bash
sudo chmod 600 /var/gassigeher/config/.env
sudo chown gassigeher:gassigeher /var/gassigeher/config/.env
```

### 5. Initialize Database

```bash
# The database will be created automatically on first run
# Migrations run automatically

# Test the application manually first
cd /var/gassigeher
./bin/gassigeher

# If it starts successfully, press Ctrl+C and continue
```

### 6. Setup systemd Service

```bash
# Copy service file
sudo cp /var/gassigeher/source/deploy/gassigeher.service /etc/systemd/system/

# Reload systemd
sudo systemctl daemon-reload

# Enable service (start on boot)
sudo systemctl enable gassigeher

# Start service
sudo systemctl start gassigeher

# Check status
sudo systemctl status gassigeher

# View logs
sudo journalctl -u gassigeher -f
```

### 7. Configure nginx

```bash
# Copy nginx configuration
sudo cp /var/gassigeher/source/deploy/nginx.conf /etc/nginx/sites-available/gassigeher

# Update server_name in the file
sudo nano /etc/nginx/sites-available/gassigeher
# Replace gassigeher.example.com with your domain

# Create symlink
sudo ln -s /etc/nginx/sites-available/gassigeher /etc/nginx/sites-enabled/

# Test nginx configuration
sudo nginx -t

# If test passes, reload nginx
sudo systemctl reload nginx
```

### 8. Setup SSL Certificate (Let's Encrypt)

```bash
# Stop nginx temporarily
sudo systemctl stop nginx

# Get certificate
sudo certbot certonly --standalone -d gassigeher.example.com -d www.gassigeher.example.com

# Update nginx config with certificate paths (already configured in nginx.conf)

# Start nginx
sudo systemctl start nginx

# Setup auto-renewal
sudo certbot renew --dry-run

# Certbot will auto-renew via systemd timer
```

### 9. Setup Automated Backups

```bash
# Make backup script executable
chmod +x /var/gassigeher/backup.sh

# Add to crontab
crontab -e
```

Add this line:
```
# Daily backup at 2:00 AM
0 2 * * * /var/gassigeher/backup.sh

# Weekly upload backup cleanup (optional)
0 3 * * 0 find /var/gassigeher/backups -name "*.gz" -mtime +90 -delete
```

### 10. Setup Log Rotation

```bash
# Create logrotate configuration
sudo nano /etc/logrotate.d/gassigeher
```

Add:
```
/var/gassigeher/logs/*.log {
    daily
    rotate 14
    compress
    delaycompress
    notifempty
    create 0640 gassigeher gassigeher
    sharedscripts
    postrotate
        systemctl reload gassigeher > /dev/null 2>&1 || true
    endscript
}
```

### 11. Configure Firewall

```bash
# Enable UFW
sudo ufw allow OpenSSH
sudo ufw allow 'Nginx Full'
sudo ufw enable

# Verify
sudo ufw status
```

### 12. Verify Deployment

1. **Test website**: Visit https://gassigeher.example.com
2. **Register account**: Create a test user
3. **Check emails**: Verify email notifications work
4. **Test booking flow**: Create a booking
5. **Test admin access**: Login with admin email
6. **Check cron jobs**: Verify auto-completion runs
7. **Check backups**: Verify daily backup creates files

### 13. Monitoring Setup (Optional but Recommended)

#### Basic Monitoring

```bash
# Check service status
sudo systemctl status gassigeher

# Check logs
sudo journalctl -u gassigeher -n 100

# Check nginx logs
sudo tail -f /var/log/nginx/gassigeher.access.log
sudo tail -f /var/log/nginx/gassigeher.error.log

# Check database size
du -h /var/gassigeher/data/gassigeher.db
```

#### Advanced Monitoring (Optional)

Consider setting up:
- **Uptime monitoring**: UptimeRobot, Pingdom, or StatusCake
- **Error tracking**: Sentry
- **Log aggregation**: ELK Stack or Loki
- **Metrics**: Prometheus + Grafana

### 14. Performance Tuning

#### nginx Performance

```bash
# Edit nginx.conf
sudo nano /etc/nginx/nginx.conf
```

Add to http block:
```nginx
# Worker processes (set to CPU count)
worker_processes auto;
worker_connections 1024;

# Gzip compression
gzip on;
gzip_vary on;
gzip_min_length 1024;
gzip_types text/plain text/css text/xml text/javascript application/x-javascript application/json application/xml+rss;

# Buffers
client_body_buffer_size 10K;
client_header_buffer_size 1k;
large_client_header_buffers 2 1k;
```

#### Application Performance

The Go application is optimized by default. Monitor:
- Response times
- Memory usage: `systemctl status gassigeher`
- Connection counts

## Maintenance

### Update Application

```bash
# Stop service
sudo systemctl stop gassigeher

# Backup current version
sudo cp /var/gassigeher/bin/gassigeher /var/gassigeher/bin/gassigeher.backup

# Deploy new version
cd /var/gassigeher/source
git pull
go build -o /var/gassigeher/bin/gassigeher ./cmd/server

# Copy updated frontend files
cp -r frontend/* /var/gassigeher/frontend/

# Restart service
sudo systemctl start gassigeher

# Check status
sudo systemctl status gassigeher
```

### Database Maintenance

```bash
# Vacuum database (optimize)
sqlite3 /var/gassigeher/data/gassigeher.db "VACUUM;"

# Check integrity
sqlite3 /var/gassigeher/data/gassigeher.db "PRAGMA integrity_check;"

# View database size
du -h /var/gassigeher/data/gassigeher.db
```

### Restore from Backup

```bash
# Stop application
sudo systemctl stop gassigeher

# Restore database
gunzip -c /var/gassigeher/backups/gassigeher_YYYYMMDD_HHMMSS.db.gz > /var/gassigeher/data/gassigeher.db

# Set permissions
sudo chown gassigeher:gassigeher /var/gassigeher/data/gassigeher.db

# Start application
sudo systemctl start gassigeher
```

## Troubleshooting

### Service Won't Start

```bash
# Check logs
sudo journalctl -u gassigeher -n 50 --no-pager

# Check environment variables
sudo cat /var/gassigeher/config/.env

# Test manually
sudo su - gassigeher
cd /var/gassigeher
./bin/gassigeher
```

### Database Locked

```bash
# Check for other processes
sudo lsof /var/gassigeher/data/gassigeher.db

# Kill if needed and restart
sudo systemctl restart gassigeher
```

### Email Not Sending

```bash
# Check Gmail API credentials
# Verify refresh token hasn't expired
# Check application logs for email errors
sudo journalctl -u gassigeher | grep -i email
```

### High Memory Usage

```bash
# Check memory usage
sudo systemctl status gassigeher

# Restart service
sudo systemctl restart gassigeher

# Consider adding memory limits to service file
```

## Security Checklist

- [ ] Firewall configured (UFW or iptables)
- [ ] SSL certificate installed and auto-renewing
- [ ] Strong JWT secret (256-bit random)
- [ ] Secure .env file permissions (600)
- [ ] Admin emails configured correctly
- [ ] Database file permissions (640)
- [ ] Regular backups running
- [ ] Log rotation configured
- [ ] nginx security headers enabled
- [ ] Application user has minimal permissions

## Backup Strategy

**Daily Backups:**
- Automated via cron (2:00 AM)
- Compressed with gzip
- 30-day retention on server
- Optional: Upload to remote storage

**Weekly Verification:**
- Test backup restoration
- Verify backup integrity
- Check backup sizes

**Disaster Recovery:**
1. Keep .env file backup securely offline
2. Document Gmail API credentials separately
3. Keep admin email list backup
4. Have deployment guide accessible

## Post-Deployment

1. **Monitor for 24 hours**: Watch logs for errors
2. **Test all features**: Registration, booking, admin functions
3. **Verify emails**: Ensure all 14 email types send correctly
4. **Check cron jobs**: Verify auto-completion and auto-deactivation
5. **Test backup restore**: Ensure backups work
6. **Performance test**: Monitor response times
7. **User documentation**: Share with users
8. **Admin training**: Train administrators

## Production Environment Variables

See `.env.production.example` for complete production configuration template.

## Support

For issues or questions:
- Check logs: `sudo journalctl -u gassigeher -f`
- Review API.md for endpoint documentation
- Review ImplementationPlan.md for architecture details

## Scaling Considerations

For high traffic:
- Use connection pooling for database
- Consider PostgreSQL instead of SQLite
- Add Redis for session caching
- Use CDN for static assets
- Load balancer for multiple instances
- Separate cron jobs to different server

---

**Deployment Status**: Ready for production deployment ✅

---

## Related Documentation

**After Deployment:**
- [USER_GUIDE.md](USER_GUIDE.md) - Share with end users
- [ADMIN_GUIDE.md](ADMIN_GUIDE.md) - Train administrators
- [API.md](API.md) - For developers/integrations

**Technical Reference:**
- [README.md](README.md) - Project overview
- [ImplementationPlan.md](ImplementationPlan.md) - Complete architecture
- [PROJECT_SUMMARY.md](PROJECT_SUMMARY.md) - Executive summary

**For Developers:**
- [CLAUDE.md](CLAUDE.md) - Development guide

---

**🚀 Ready to deploy Gassigeher and help shelter dogs get the walks they need!**
