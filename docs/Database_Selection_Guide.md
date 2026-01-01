# Database Selection Guide for Gassigeher

**Purpose:** Help you choose the right database for your deployment
**Last Updated:** 2025-12-31
**Supported Databases:** SQLite and PostgreSQL

---

## Quick Decision Matrix

| Your Situation | Recommended Database | Why |
|----------------|---------------------|-----|
| **Development** | SQLite | Zero setup, fast, file-based |
| **Small shelter (<1,000 users)** | SQLite | Simple, reliable, no server needed |
| **Large shelter (1,000+ users)** | PostgreSQL | Enterprise-grade, highly concurrent |
| **Multiple shelters (SaaS-Mode)** | PostgreSQL | Required for Row-Level Security |
| **Cloud deployment** | PostgreSQL | Well-supported on cloud platforms |

---

## Database Comparison

### SQLite

**Best For:** Development, small deployments, simple setups

**Advantages:**
- Zero configuration - No server to install or configure
- File-based - Single file database (easy backup)
- Fast for small datasets - Excellent performance up to 1,000 users
- Portable - Works on Windows, Linux, Mac
- No cost - Free, no licensing
- Easy backup - Just copy the file

**Limitations:**
- Limited concurrency - One writer at a time
- No network access - Cannot connect from remote clients
- Not for large scale - Performance degrades above 1,000 users
- Single server only - Cannot distribute across servers

**Recommended For:**
- Development and testing
- Single-server deployments
- Shelters with <1,000 users
- Simple hosting requirements

**Max Recommended Users:** 1,000

---

### PostgreSQL

**Best For:** Enterprise applications, SaaS-Mode, high concurrency

**Advantages:**
- Advanced features - JSON, full-text search, geospatial
- Excellent concurrency - Best for many simultaneous writers
- ACID compliant - Strong data integrity guarantees
- Extensible - Can add custom functions and types
- Standards compliant - Follows SQL standards closely
- Great for analytics - Complex queries perform well
- Row-Level Security - Required for SaaS multi-tenancy

**Limitations:**
- More complex setup - Requires PostgreSQL server
- Steeper learning curve - More configuration options
- Resource intensive - Needs more RAM than SQLite
- Less common on shared hosting - May need VPS or cloud

**Recommended For:**
- Enterprise deployments
- SaaS-Mode deployments (required)
- Shelters with 1,000+ users
- Applications with complex data requirements
- Cloud deployments (AWS RDS, Google Cloud SQL, etc.)

**Max Recommended Users:** 1,000,000+

---

## Feature Comparison

| Feature | SQLite | PostgreSQL |
|---------|--------|------------|
| **Setup Time** | 0 min | 30-45 min |
| **Maintenance** | None | Medium |
| **Backup** | File copy | pg_dump |
| **Replication** | No | Yes |
| **Clustering** | No | Yes |
| **Full-Text Search** | Yes (FTS5) | Yes (Better) |
| **JSON Support** | Yes | Yes (Better) |
| **Concurrent Writes** | Limited | Excellent |
| **Transaction Performance** | Excellent | Excellent |
| **Storage Limit** | 281 TB | Unlimited |
| **Row-Level Security** | No | Yes |

---

## Performance Comparison

### Expected Response Times (Typical Queries)

| Operation | SQLite | PostgreSQL |
|-----------|--------|------------|
| **Simple SELECT** | 0.1-0.5ms | 0.5-2ms |
| **Complex JOIN** | 1-5ms | 2-8ms |
| **INSERT** | 0.5-1ms | 1-3ms |
| **UPDATE** | 0.5-1ms | 1-3ms |
| **Transaction** | 1-2ms | 2-5ms |

**Note:** Network latency adds ~0.5-1ms for PostgreSQL on remote servers

### Concurrent User Support

| Database | Concurrent Reads | Concurrent Writes | Max Users* |
|----------|------------------|-------------------|------------|
| **SQLite** | Unlimited | 1 at a time | 1,000 |
| **PostgreSQL** | Excellent | Excellent | 1,000,000+ |

*Max users = realistic limit for Gassigeher use case

---

## Cost Comparison

### SQLite

**Server Cost:** $0 (runs on app server)
**Hosting Cost:** Minimal (any server can run it)
**Maintenance Cost:** $0/month
**Backup Cost:** $0 (file copy)

**Total:** ~$5-10/month (app server only)

---

### PostgreSQL

**Server Cost:**
- VPS: $20-100/month
- Cloud (AWS RDS): $25-150/month
- Managed (Digital Ocean): $15-60/month

**Hosting Cost:** Depends on size
**Maintenance Cost:** Low-Medium
**Backup Cost:** Included in hosting

**Total:** ~$30-150/month (varies by hosting)

---

## Migration Paths

### Start with SQLite, Grow as Needed

**Recommended Path:**
```
Development -> SQLite
  |
Small Deployment (< 1,000 users) -> SQLite
  |
Growing (1,000+ users) -> Migrate to PostgreSQL
  |
SaaS-Mode -> PostgreSQL (required)
```

**Migration Difficulty:**
- SQLite -> PostgreSQL: Easy (1-2 hours)

---

## When to Switch Databases

### Signs You've Outgrown SQLite

- Database file > 1 GB
- Frequent "database locked" errors
- Slow queries (>100ms for simple SELECTs)
- More than 10 concurrent users
- Need remote database access
- Plan to exceed 1,000 users
- Want to deploy SaaS-Mode

**Solution:** Migrate to PostgreSQL

---

## Database Selection Flowchart

```
Start Here
   |
Do you need SaaS-Mode?
   |-- Yes -> Use PostgreSQL (required for RLS)
   |
   +-- No -> Do you have < 1,000 users?
               |-- Yes -> Use SQLite (Simple, free, fast)
               |
               +-- No -> Use PostgreSQL (Scalable, enterprise-grade)
```

---

## Detailed Comparison

### SQLite Use Cases

**Perfect For:**
- Local development
- Demo/staging environments
- Small animal shelters (1-50 dogs, <1,000 volunteers)
- Single-server deployments
- Embedded applications

**Example Deployment:**
- Small shelter with 20 dogs
- 100 registered volunteers
- 1-5 concurrent users typically
- ~100 bookings per month
- Single VPS server

**Database Size After 1 Year:**
- Users: 100 x ~1KB = 100KB
- Dogs: 20 x ~2KB = 40KB
- Bookings: 1200 x ~500 bytes = 600KB
- **Total: ~1MB** - SQLite handles this easily

---

### PostgreSQL Use Cases

**Perfect For:**
- Large shelters or shelter networks
- 500+ dogs, 10,000+ volunteers
- High concurrent users (50-500)
- Heavy write load
- Complex reporting needs
- Multi-region deployments
- SaaS-Mode deployments

**Example Deployment:**
- Shelter network with 1,000+ dogs
- 10,000+ registered volunteers
- 50-200 concurrent users
- ~10,000 bookings per month
- Cloud infrastructure
- Advanced analytics

**Database Size After 1 Year:**
- Users: 10,000 x ~1KB = 10MB
- Dogs: 1,000 x ~2KB = 2MB
- Bookings: 120,000 x ~500 bytes = 60MB
- **Total: ~75MB** - PostgreSQL excels at this scale

---

## Configuration Examples

### SQLite (Default)

**.env:**
```bash
# Minimal configuration (or no .env at all)
DB_TYPE=sqlite
DATABASE_PATH=./gassigeher.db
```

**That's it!** No server, no additional configuration.

---

### PostgreSQL

**.env:**
```bash
DB_TYPE=postgres
DB_HOST=localhost
DB_PORT=5432
DB_NAME=gassigeher
DB_USER=gassigeher_user
DB_PASSWORD=your_secure_password
DB_SSLMODE=require
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
```

**Setup Time:** 30-45 minutes (install PostgreSQL, create database, configure)

---

## Hosting Recommendations

### For SQLite

**Best Hosting:**
- Any VPS (DigitalOcean, Linode, Vultr)
- Shared hosting (if Go supported)
- Local server

**Requirements:**
- 512MB RAM minimum
- 10GB disk space
- Linux or Windows server

**Cost:** $5-10/month

---

### For PostgreSQL

**Best Hosting:**
- Managed PostgreSQL (AWS RDS, Google Cloud SQL, DigitalOcean)
- VPS with PostgreSQL installed
- Heroku Postgres

**Requirements:**
- 2GB RAM minimum (4GB recommended)
- 20GB disk space
- Reliable network

**Cost:** $25-100/month (managed), $20-50/month (VPS + self-managed)

---

## Migration Timing

### When to Migrate

**SQLite -> PostgreSQL:**
- Reaching 500-1,000 users
- Need for concurrent write access
- Remote database access needed
- Approaching 1GB database size
- Planning SaaS-Mode deployment

**Timeline:** Plan migration before hitting limits, not after

---

## Decision Factors

### Technical Factors

| Factor | SQLite | PostgreSQL |
|--------|--------|------------|
| **Concurrent Writers** | 1 | 500+ |
| **Query Complexity** | Simple-Medium | Very Complex |
| **Data Size** | <10 GB | Unlimited |
| **Setup Complexity** | Easy | Medium-Hard |
| **Maintenance** | None | Regular |
| **Row-Level Security** | No | Yes |

### Business Factors

| Factor | SQLite | PostgreSQL |
|--------|--------|------------|
| **Initial Cost** | $0 | $$$ |
| **Ongoing Cost** | $ | $$$ |
| **Team Expertise** | Easy | Less Common |
| **Vendor Support** | Limited | Good |
| **Cloud Options** | Limited | Excellent |

---

## Recommendations by Shelter Size

### Tiny Shelter (1-10 dogs, <50 users)

**Recommended:** SQLite
**Why:** Overkill to use anything else
**Cost:** ~$5-10/month (basic VPS)

---

### Small Shelter (10-50 dogs, 50-500 users)

**Recommended:** SQLite
**Why:** Still within SQLite's sweet spot
**Cost:** ~$10-20/month (VPS)

---

### Medium Shelter (50-200 dogs, 500-5,000 users)

**Recommended:** PostgreSQL
**Why:** Better concurrency, scalability, cloud support
**Cost:** ~$30-80/month (managed PostgreSQL or VPS)

---

### Large Shelter (200+ dogs, 5,000+ users)

**Recommended:** PostgreSQL
**Why:** Best performance at scale, advanced features
**Cost:** ~$50-150/month (managed PostgreSQL)

---

### Shelter Network (Multiple Locations / SaaS-Mode)

**Recommended:** PostgreSQL (Required)
**Why:** Row-Level Security for multi-tenancy, scalability
**Cost:** ~$100-500/month (cloud deployment with replication)

---

## Quick Start Guides

### Use SQLite (No Setup)

```bash
# 1. Clone repository
git clone <repo-url>
cd gassigeher

# 2. Configure (or use defaults)
cp .env.example .env
# DATABASE_PATH=./gassigeher.db (default)

# 3. Run
go run cmd/server/main.go

# Done! Database created automatically
```

---

### Use PostgreSQL (With Docker)

```bash
# 1. Start PostgreSQL
docker run --name gassigeher-postgres \
  -e POSTGRES_DB=gassigeher \
  -e POSTGRES_USER=gassigeher_user \
  -e POSTGRES_PASSWORD=gassigeher_pass \
  -p 5432:5432 -d postgres:15

# 2. Configure .env
DB_TYPE=postgres
DB_HOST=localhost
DB_PORT=5432
DB_NAME=gassigeher
DB_USER=gassigeher_user
DB_PASSWORD=gassigeher_pass
DB_SSLMODE=disable

# 3. Run application
go run cmd/server/main.go
# Tables created automatically!
```

---

## FAQ

### Can I switch databases later?

**Yes!** Gassigeher supports both databases. You can migrate from SQLite to PostgreSQL when needed.

---

### What if I'm not sure which to choose?

**Start with SQLite.** You can always migrate later. SQLite is perfect for:
- Getting started quickly
- Development and testing
- Small deployments

When you outgrow SQLite, you'll know (slow queries, database locked errors).

---

### Does the API change with different databases?

**No.** The API is identical regardless of database. All features work the same.

---

### Can I use a different database not listed?

**Not currently.** Gassigeher supports SQLite and PostgreSQL. These cover 99% of use cases.

---

### Which database is most reliable?

Both are **extremely reliable** when properly configured. Choose based on your needs, not reliability concerns.

- SQLite: Billions of deployments, rock-solid
- PostgreSQL: Bank-grade reliability, ACID compliant

---

### Can I use managed database services?

**Yes!** Gassigeher works great with:

**PostgreSQL:**
- AWS RDS for PostgreSQL
- Google Cloud SQL for PostgreSQL
- Azure Database for PostgreSQL
- Heroku Postgres
- DigitalOcean Managed PostgreSQL

Just set `DB_HOST`, `DB_USER`, `DB_PASSWORD` to your managed service credentials.

---

## Summary

**For most users:** Start with **SQLite** (simple, free, fast)

**When growing:** Migrate to **PostgreSQL** (advanced, scalable)

**For SaaS-Mode:** Use **PostgreSQL** (required for Row-Level Security)

**Switching databases:** Easy - just change environment variables and run migrations!

---

**Related Documentation:**
- [PostgreSQL Setup Guide](PostgreSQL_Setup_Guide.md) - Detailed PostgreSQL configuration
- [Multi-Database Testing Guide](MultiDatabase_Testing_Guide.md) - Testing with both databases

---

**Need Help Deciding?** Consider:
1. How many users do you expect?
2. What's your technical expertise?
3. What's your budget?
4. Do you need SaaS-Mode (multi-tenancy)?

Still unsure? **Start with SQLite** - you can always migrate later!
