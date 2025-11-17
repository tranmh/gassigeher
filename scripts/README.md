# Scripts Documentation

Utility scripts for Gassigeher development and testing.

## Available Scripts

### `gentestdata.ps1` - Test Data Generator

Generates comprehensive test data for the Gassigeher application, including users, dogs, bookings, and all related entities.

**How it works:**
- Generates a SQL file (`scripts/testdata.sql`) with all test data
- Executes the SQL file using `bin/sqlite3.exe`
- Clears all existing data first (fresh start)
- Creates realistic German test data spanning 28 days

**Purpose:**
- Populates a fresh database with realistic test data
- Creates bookings spanning past 2 weeks and future 2 weeks
- Includes edge cases for thorough testing

**Prerequisites:**

1. **sqlite3.exe**: Should already be in `bin/sqlite3.exe` (included with project)
   - If missing, download from: https://www.sqlite.org/download.html
   - Or install via package manager:
     ```powershell
     # Chocolatey
     choco install sqlite

     # Scoop
     scoop install sqlite
     ```

2. **Application Database**: The application must be run at least once to create the database schema
   ```bash
   go run cmd/server/main.go
   # Wait for "Server running on port 8080"
   # Stop the server (Ctrl+C)
   ```

**Usage:**

```powershell
# Basic usage (uses DATABASE_PATH from .env or default gassigeher.db)
.\scripts\gentestdata.ps1

# Custom database path
.\scripts\gentestdata.ps1 -DatabasePath "C:\path\to\custom.db"

# Custom .env file
.\scripts\gentestdata.ps1 -EnvFile "C:\path\to\.env"
```

**What Gets Generated:**

- **System Settings**: Default booking/cancellation rules
- **Users** (12 total):
  - 1 admin user (from ADMIN_EMAILS env var)
  - 4 green-level users (beginner walkers)
  - 4 blue-level users (intermediate walkers)
  - 3 orange-level users (experienced walkers)
  - 1 inactive user (for auto-deactivation testing)
  - 1 deleted user (for GDPR testing)
  - All verified and ready to use
  - Password for all: **test123**

- **Dogs** (18 total):
  - 7 green category dogs (easy to handle)
  - 6 blue category dogs (moderate experience needed)
  - 5 orange category dogs (experienced handlers only)
  - 2 marked as unavailable
  - Realistic German breeds and characteristics

- **Bookings** (~60-80 total):
  - **Historical**: 2-4 bookings/day for past 14 days (all completed)
  - **Today**: 3 bookings (morning completed, afternoon pending)
  - **Future**: 3-6 bookings/day for next 14 days (pending + some cancelled)
  - Mix of morning/afternoon walk types
  - ~10% cancellation rate for future bookings

- **Walk Notes**: Added to ~60% of completed bookings
  - Realistic German comments about each walk

- **Blocked Dates**: 3 random dates in next 2 weeks
  - Tests booking prevention logic

- **Experience Level Requests** (4 total):
  - 2 pending (awaiting admin review)
  - 1 approved (with admin comment)
  - 1 rejected (with reason)

**Output Example:**

```
╔════════════════════════════════════════════════════════════╗
║           Test Data Generation Complete!                   ║
╚════════════════════════════════════════════════════════════╝

Summary:
  Users:                 12 (1 admin, 1 deleted, 1 inactive)
  Dogs:                  18 (2 unavailable)
  Bookings:              73 (spanning 28 days)
  Walk Notes:            28
  Blocked Dates:         3
  Experience Requests:   4

Login Credentials (all users):
  Password: test123

Sample User Logins:
  Admin:  admin@tierheim-goeppingen.de
  Green User:  max.mueller@example.com
  Blue User:  anna.schmidt@example.com
  Orange User:  lukas.fischer@example.com
```

**Testing Scenarios Covered:**

1. **Authentication & Authorization**
   - Admin vs regular user access
   - Email verification flow (some users unverified)
   - Different experience levels

2. **Booking Management**
   - Creating bookings at different experience levels
   - Double-booking prevention (unique constraint)
   - Blocked date prevention
   - Cancellation workflow
   - Auto-completion of past bookings

3. **User Lifecycle**
   - Active users with recent activity
   - Inactive users (365+ days old)
   - Deleted accounts (GDPR anonymization)
   - Experience level progression requests

4. **Admin Workflows**
   - Reviewing experience requests
   - Managing dogs (available/unavailable)
   - Viewing booking statistics
   - Managing blocked dates

5. **Walk History & Notes**
   - Completed bookings with/without notes
   - Historical activity tracking
   - User engagement metrics

**Important Notes:**

⚠️ **Data Clearing**: This script **deletes all existing data** before generating new data. Use with caution in production environments!

🔑 **Admin Access**: Admin user email is taken from `ADMIN_EMAILS` in your `.env` file. If not set, defaults to `admin@tierheim-goeppingen.de`.

📅 **Dynamic Dates**: Bookings are generated relative to today's date, so running the script on different days will produce different date ranges.

🔒 **Password Hash**: The bcrypt hash for "test123" is pre-generated. To change the test password, run:
```bash
go run scripts/genhash.go <new_password>
# Copy the hash output and update $TEST_PASSWORD_HASH in gentestdata.ps1
```

📄 **Generated SQL File**: The script creates `scripts/testdata.sql` which you can inspect or manually execute:
```bash
# View the generated SQL
cat scripts/testdata.sql

# Manually execute if needed (instead of running the PowerShell script)
.\bin\sqlite3.exe gassigeher.db ".read scripts/testdata.sql"
```

---

### `genhash.go` - Password Hash Generator

Utility to generate bcrypt password hashes for test users.

**Usage:**

```bash
go run scripts/genhash.go <password>

# Example:
go run scripts/genhash.go test123
# Output:
# Password: test123
# Bcrypt Hash: $2a$10$LT4jdYaamd5Sxed9IhHTKuedmp/AvzGH27pJwCFzxAqAuO0c6OqfC
```

**Purpose:**
- Generate bcrypt hashes compatible with Go's auth service
- Use when changing default test password in `gentestdata.ps1`
- Quick hash generation for manual database operations

---

## Troubleshooting

### "Database not found"
**Cause**: Application hasn't been run yet to create database schema.

**Solution**:
```bash
go run cmd/server/main.go
# Wait for server to start, then stop (Ctrl+C)
.\scripts\gentestdata.ps1
```

### "SQLite3.exe not found"
**Cause**: No SQLite connectivity method available.

**Solution**:
1. Download sqlite3.exe from https://www.sqlite.org/download.html
2. Place in `C:\Windows\System32\` or in `scripts\` folder
3. Or install System.Data.SQLite NuGet package

### "SQL Error: UNIQUE constraint failed"
**Cause**: Some random data happened to create duplicates (very rare).

**Solution**:
- This is normal due to random generation
- The script will skip duplicate bookings automatically
- No action needed

### "Failed to initialize email service"
**Cause**: Gmail API credentials not configured (expected for testing).

**Solution**:
- This warning is normal for test environments
- Email functionality will be disabled, but app works fine
- Configure Gmail credentials in `.env` if email testing is needed

---

## Development Workflow

**Typical Testing Cycle:**

```bash
# 1. Build application
.\bat.bat

# 2. Generate fresh test data
.\scripts\gentestdata.ps1

# 3. Start application
go run cmd/server/main.go

# 4. Test in browser
# Open http://localhost:8080
# Login with any test user (password: test123)

# 5. Regenerate data when needed
# Stop server (Ctrl+C)
# .\scripts\gentestdata.ps1
# Restart server
```

**CI/CD Integration:**

```powershell
# Automated test setup
.\bat.bat                       # Build & test
.\scripts\gentestdata.ps1       # Load test data
go run cmd/server/main.go &     # Start server in background
# Run integration tests
```

---

## Contributing

When adding new test scenarios:

1. Update `gentestdata.ps1` with new data generation logic
2. Update this README with new test scenarios
3. Ensure data is realistic and covers edge cases
4. Test script on clean database
5. Document any new prerequisites

---

## File Structure

```
scripts/
├── README.md           # This file - comprehensive documentation
├── gentestdata.ps1     # Test data generator (PowerShell)
├── genhash.go          # Password hash utility (Go)
└── testdata.sql        # Generated SQL file (created by gentestdata.ps1)
```

---

**Last Updated**: Phase 10 Complete (Production Ready)
**Compatibility**: Windows PowerShell 5.1+, PowerShell Core 7+
**Database Version**: SQLite 3.x
