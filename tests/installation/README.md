# Installation Tests

This directory contains automated tests for verifying the installation and first-run behavior of Gassigeher.

## Files

### `test_all_scenarios.go`
Comprehensive Go test program that validates all 3 Super Admin credential scenarios:

1. **First-time installation** (empty database)
2. **Existing database + missing credentials file**
3. **Normal startup** (both database and credentials exist)

**Usage:**
```bash
# Build the test
go build -o test_all_scenarios.exe test_all_scenarios.go

# Run the test
./test_all_scenarios.exe
```

**What it does:**
- Backs up your existing database and credentials file
- Tests all 3 scenarios in sequence
- Verifies Super Admin creation and credentials file generation
- Restores original files after testing
- Provides detailed pass/fail output

**Safe to run:** Yes - automatically backs up and restores your data.

### `test_credentials_fix.bat`
Simple Windows batch script to test the credentials regeneration feature.

**Usage:**
```cmd
test_credentials_fix.bat
```

**What it does:**
- Deletes the `SUPER_ADMIN_CREDENTIALS.txt` file (if exists)
- Starts the Gassigeher server
- Server will detect missing credentials and regenerate them
- Displays the new password in console

**Note:** Press Ctrl+C to stop the server after seeing the credentials.

## When to Use These Tests

### For Developers
- After modifying Super Admin initialization code
- After changing database migrations
- Before releasing a new version
- When troubleshooting installation issues

### For System Administrators
- To verify first-time installation works correctly
- To test credential recovery (scenario 2)
- To understand the installation process

## Test Results

All tests are passing as of 2025-11-23. See `/docs/SUPER_ADMIN_FIX_SUMMARY.md` for detailed test results.

## Related Documentation

- [SUPER_ADMIN_FIX_SUMMARY.md](../../docs/SUPER_ADMIN_FIX_SUMMARY.md) - Complete test results and implementation details
- [DEPLOYMENT.md](../../docs/DEPLOYMENT.md) - Production deployment guide
- [InstallationAndSelfServices.md](../../docs/InstallationAndSelfServices.md) - Installation and self-service features

## Building from Source

To run these tests, you must first ensure all dependencies are installed:

```bash
# From project root
go mod download

# Build test
cd tests/installation
go build -o test_all_scenarios.exe test_all_scenarios.go
```

## Troubleshooting

**Test fails with "module not found":**
- Run from project root with full path: `go run tests/installation/test_all_scenarios.go`

**Test fails with database errors:**
- Ensure no other instance of Gassigeher is running
- Check that .env file exists in project root
- Verify SQLite3 is available on your system

**Credentials file not created:**
- Check file permissions in current directory
- Verify `SUPER_ADMIN_EMAIL` is set in .env
- Check logs for error messages
