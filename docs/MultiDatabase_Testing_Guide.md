# Multi-Database Testing Guide

**Created:** 2025-01-22
**Updated:** 2025-12-31
**Purpose:** Guide for testing Gassigeher with SQLite and PostgreSQL
**Audience:** Developers, QA, CI/CD Engineers

---

## Quick Start

### Test with SQLite (Default - Always Available)

```bash
# Just run tests normally
go test ./...

# Or use bat.bat
.\bat.bat

# Result: Uses in-memory SQLite, fast, no setup needed
```

**Expected:** All tests passing

---

### Test with PostgreSQL (Requires Docker)

```bash
# 1. Start PostgreSQL test database
docker-compose -f docker-compose.test.yml up -d postgres-test

# 2. Wait for ready (5-10 seconds)

# 3. Set connection string
export DB_TEST_POSTGRES="postgres://gassigeher_test:testpass@localhost:5433/gassigeher_test?sslmode=disable"

# Windows PowerShell:
$env:DB_TEST_POSTGRES="postgres://gassigeher_test:testpass@localhost:5433/gassigeher_test?sslmode=disable"

# 4. Run tests
go test ./...

# 5. Stop database
docker-compose -f docker-compose.test.yml down
```

---

### Test Both Databases (Automated)

```bash
# PowerShell (Windows):
.\scripts\test_all_databases.ps1

# Bash (Linux/Mac):
chmod +x scripts/test_all_databases.sh
./scripts/test_all_databases.sh
```

**What It Does:**
1. Runs tests with SQLite
2. Starts PostgreSQL via Docker
3. Runs tests with PostgreSQL
4. Reports summary

**Duration:** ~1-2 minutes total

---

## Docker Compose Reference

### File: `docker-compose.test.yml`

**Services:**
- `postgres-test` - PostgreSQL 15 on port 5433
- `adminer` - Web UI for database management (port 8081)

### Commands

**Start test database:**
```bash
docker-compose -f docker-compose.test.yml up -d postgres-test
```

**View logs:**
```bash
docker-compose -f docker-compose.test.yml logs -f postgres-test
```

**Check health:**
```bash
docker-compose -f docker-compose.test.yml ps
```

**Stop database:**
```bash
docker-compose -f docker-compose.test.yml down
```

**Clean up (remove data):**
```bash
docker-compose -f docker-compose.test.yml down -v
```

---

## Test Database Credentials

### PostgreSQL

- **Host:** localhost
- **Port:** 5433 (mapped from 5432)
- **Database:** gassigeher_test
- **User:** gassigeher_test
- **Password:** testpass

**Connection String:**
```
postgres://gassigeher_test:testpass@localhost:5433/gassigeher_test?sslmode=disable
```

---

## Adminer Web UI

**Access:** http://localhost:8081

**PostgreSQL Login:**
- System: PostgreSQL
- Server: postgres-test
- Username: gassigeher_test
- Password: testpass
- Database: gassigeher_test

**Features:**
- Browse tables
- Run SQL queries
- View database structure
- Import/export data

---

## Test Categories

### 1. Unit Tests (Automatic)

**Run:** `go test ./internal/database -v`

**Tests:**
- Dialect implementations
- Migration registry
- Migration runner
- Integration tests

**Databases:** Runs on SQLite by default, PostgreSQL if env var set

---

### 2. Repository Tests (Automatic)

**Run:** `go test ./internal/repository -v`

**Tests:**
- Dog repository
- Booking repository
- User repository
- Other repositories

**Databases:** Uses test helper - SQLite default, can use PostgreSQL

---

### 3. Handler Tests (Automatic)

**Run:** `go test ./internal/handlers -v`

**Tests:**
- Auth handler
- Dog handler
- Booking handler
- Other handlers

**Databases:** Uses test helper - SQLite default

---

### 4. Integration Tests (Manual with Docker)

**Run:**
```bash
# Start databases
docker-compose -f docker-compose.test.yml up -d

# Set env var
export DB_TEST_POSTGRES="..."

# Run tests
go test ./internal/database -v -run TestInitializeWithConfig

# Clean up
docker-compose -f docker-compose.test.yml down
```

**Tests:**
- SQLite initialization
- PostgreSQL initialization (if available)
- Connection pool configuration
- DSN builders

---

## Test Execution Strategies

### Strategy 1: Quick Test (SQLite Only)

**Use Case:** Local development, quick verification

**Command:**
```bash
go test ./...
```

**Duration:** ~10-15 seconds
**Databases:** SQLite only
**Coverage:** All tests

---

### Strategy 2: Full Test (Both Databases)

**Use Case:** Pre-commit, comprehensive verification

**Command:**
```bash
# Windows
.\scripts\test_all_databases.ps1

# Linux/Mac
./scripts/test_all_databases.sh
```

**Duration:** ~1-2 minutes (includes Docker startup)
**Databases:** SQLite and PostgreSQL
**Coverage:** All tests x 2 databases

---

### Strategy 3: CI/CD Test (Matrix)

**Use Case:** Automated testing in CI/CD pipeline

**GitHub Actions Example:**
```yaml
jobs:
  test-sqlite:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - run: go test ./... -v

  test-postgres:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_DB: gassigeher_test
          POSTGRES_USER: gassigeher_test
          POSTGRES_PASSWORD: testpass
        ports:
          - 5432:5432
        options: >-
          --health-cmd="pg_isready -U gassigeher_test"
          --health-interval=10s
          --health-timeout=5s
          --health-retries=5
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - run: go test ./... -v
        env:
          DB_TEST_POSTGRES: postgres://gassigeher_test:testpass@localhost:5432/gassigeher_test?sslmode=disable
```

**Duration:** ~1-2 minutes per database (parallel execution)
**Coverage:** Complete test matrix

---

## Troubleshooting

### PostgreSQL Container Won't Start

**Symptom:** Port 5432 already in use

**Solution:**
```bash
# Check if PostgreSQL running locally
netstat -an | grep 5432

# Stop local PostgreSQL or edit docker-compose.test.yml to use port 5433
```

---

### Tests Timeout Waiting for Database

**Symptom:** "Failed to become ready within X seconds"

**Solutions:**
1. **Increase timeout** in test script
2. **Check Docker resources** (increase memory/CPU)
3. **Check health checks** in docker-compose.test.yml
4. **View logs:** `docker-compose -f docker-compose.test.yml logs postgres-test`

---

### Connection Refused Errors

**Symptom:** "connection refused" when running tests

**Solutions:**
1. **Verify containers running:**
   ```bash
   docker-compose -f docker-compose.test.yml ps
   ```

2. **Check health status:**
   ```bash
   docker-compose -f docker-compose.test.yml ps
   # Status should be "healthy"
   ```

3. **Restart containers:**
   ```bash
   docker-compose -f docker-compose.test.yml restart
   ```

---

### Tests Fail on PostgreSQL but Pass on SQLite

**Possible Causes:**
1. **Type differences** - Check boolean handling
2. **Date/time differences** - Check timestamp comparisons
3. **Transaction behavior** - Check isolation levels
4. **Charset issues** - Verify UTF-8 encoding

**Debug:**
```bash
# Run specific test with verbose output
go test ./internal/repository -v -run TestSpecificTest

# Check test result files
cat test_results_postgres.txt | grep FAIL
```

---

## Performance Comparison

### Expected Performance Characteristics

| Operation | SQLite | PostgreSQL |
|-----------|--------|------------|
| **INSERT** | Fast | Fast |
| **SELECT** | Fast | Fast |
| **UPDATE** | Fast | Moderate |
| **DELETE** | Fast | Moderate |
| **Concurrent Writes** | Limited | Excellent |
| **Concurrent Reads** | Excellent | Excellent |

### Running Benchmarks

```bash
# Benchmark database operations
go test ./internal/repository -bench=. -benchmem

# Compare across databases
DB_TEST_POSTGRES="..." go test ./internal/repository -bench=.
```

---

## CI/CD Integration

### GitHub Actions Example

**File:** `.github/workflows/test.yml`

```yaml
name: Multi-Database Tests

on: [push, pull_request]

jobs:
  test-sqlite:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - run: go test ./... -v

  test-postgres:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_DB: gassigeher_test
          POSTGRES_USER: gassigeher_test
          POSTGRES_PASSWORD: testpass
        ports:
          - 5432:5432
        options: >-
          --health-cmd="pg_isready -U gassigeher_test"
          --health-interval=10s
          --health-timeout=5s
          --health-retries=5
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - run: go test ./... -v
        env:
          DB_TEST_POSTGRES: postgres://gassigeher_test:testpass@localhost:5432/gassigeher_test?sslmode=disable
```

---

## Manual Testing Checklist

### Before Committing Changes

- [ ] Run `go test ./...` (SQLite tests)
- [ ] Verify all tests pass
- [ ] No new warnings or errors

### Before Releasing

- [ ] Start Docker test database
- [ ] Run `./scripts/test_all_databases.sh` (or .ps1)
- [ ] Verify both databases pass
- [ ] Check test result files
- [ ] Review any skipped tests
- [ ] Performance acceptable

### Before Production Deployment

- [ ] Full test suite on both databases
- [ ] Load testing on target database
- [ ] Migration testing (if switching databases)
- [ ] Backup/restore testing
- [ ] Performance benchmarks

---

## Test Data Management

### SQLite

**Storage:** In-memory (`:memory:`)
**Persistence:** None (cleaned after each test)
**Speed:** Fastest (no disk I/O)

### PostgreSQL

**Storage:** Docker volume `postgres_test_data`
**Persistence:** Persists between test runs
**Cleanup:** `cleanPostgreSQLTestDB()` drops all tables with CASCADE
**Speed:** Moderate (Docker overhead)

---

## Common Test Patterns

### Test Specific Repository with Specific Database

```go
func TestDogRepository_PostgreSQL(t *testing.T) {
    db := testutil.SetupTestDBWithType(t, "postgres")
    if db == nil {
        return // Skipped if PostgreSQL not available
    }

    // Test runs on PostgreSQL
    repo := repository.NewDogRepository(db)
    // ... test logic ...
}
```

### Test All Repositories with Both Databases

```go
func TestRepositories_AllDatabases(t *testing.T) {
    databases := []string{"sqlite", "postgres"}

    for _, dbType := range databases {
        t.Run(dbType, func(t *testing.T) {
            db := testutil.SetupTestDBWithType(t, dbType)
            if db == nil {
                return // Skipped if DB not available
            }

            // Run all repository tests on this database
            testUserRepository(t, db)
            testDogRepository(t, db)
            testBookingRepository(t, db)
            // ...
        })
    }
}
```

---

## Debugging Tips

### View PostgreSQL Logs

```bash
docker-compose -f docker-compose.test.yml logs -f postgres-test
```

### Connect to Database Directly

**PostgreSQL:**
```bash
docker exec -it gassigeher-postgres-test psql -U gassigeher_test -d gassigeher_test
```

### Reset Test Databases

```bash
# Complete reset
docker-compose -f docker-compose.test.yml down -v
docker-compose -f docker-compose.test.yml up -d

# Or just restart
docker-compose -f docker-compose.test.yml restart
```

---

## Test Results Files

### Output Files

- `test_results_sqlite.txt` - SQLite test output
- `test_results_postgres.txt` - PostgreSQL test output (if run)

### Analyzing Results

**Count passing tests:**
```bash
grep -c "^--- PASS:" test_results_sqlite.txt
```

**Find failures:**
```bash
grep "^--- FAIL:" test_results_sqlite.txt
grep -A 10 "FAIL:" test_results_sqlite.txt
```

**Search for specific test:**
```bash
grep -A 20 "TestDogRepository_Create" test_results_sqlite.txt
```

---

## Best Practices

### 1. Always Test with SQLite First

SQLite tests are fast and don't require setup. Run these first to catch obvious issues.

### 2. Use Docker for PostgreSQL

Don't install PostgreSQL locally for testing. Use Docker for:
- Consistent environment
- Easy cleanup
- Version control
- Parallel testing

### 3. Clean Databases Between Tests

Test helpers automatically clean databases, but for manual testing:

**PostgreSQL:**
```sql
DROP TABLE IF EXISTS bookings CASCADE;
DROP TABLE IF EXISTS dogs CASCADE;
DROP TABLE IF EXISTS users CASCADE;
...
```

### 4. Check for Skipped Tests

```bash
grep "SKIP:" test_results_*.txt
```

Skipped tests indicate missing test database configuration.

---

## Performance Testing

### Benchmark Commands

```bash
# Benchmark all packages
go test ./... -bench=. -benchmem -benchtime=10s

# Benchmark specific package
go test ./internal/repository -bench=. -benchmem

# Save results for comparison
go test ./... -bench=. > bench_sqlite.txt
DB_TEST_POSTGRES="..." go test ./... -bench=. > bench_postgres.txt

# Compare results
benchcmp bench_sqlite.txt bench_postgres.txt
```

### Expected Benchmarks

**SQLite (baseline):**
- INSERT: ~0.5ms
- SELECT: ~0.1ms
- UPDATE: ~0.5ms

**PostgreSQL:**
- INSERT: ~1-2ms
- SELECT: ~0.5ms
- UPDATE: ~1-2ms

**Note:** Docker adds ~0.5-1ms overhead. Production databases on same network would be faster.

---

## Conclusion

Multi-database testing ensures Gassigeher works correctly with SQLite and PostgreSQL. Use the provided Docker Compose and test scripts for comprehensive testing before releases.

**Quick Commands:**

```bash
# Local development
go test ./...

# Pre-release testing
.\scripts\test_all_databases.ps1

# Clean up
docker-compose -f docker-compose.test.yml down -v
```

---

**Questions?** See `docs/DatabasesSupportPlan.md` for full implementation details.
