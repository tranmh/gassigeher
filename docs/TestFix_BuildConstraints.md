# Test Build Fix: Build Constraints for Script Files

**Date:** 2025-01-21
**Issue:** Build failure in scripts directory during `go test ./...`
**Status:** ✅ **FIXED**

---

## Problem Description

### Error Message

When running `bat.bat` or `go test ./...`:

```
# github.com/tranm/gassigeher/scripts
scripts\test_phase2.go:14:6: main redeclared in this block
    scripts\genhash.go:13:6: other declaration of main
FAIL	github.com/tranm/gassigeher/scripts [build failed]
FAIL
[WARNING] Some tests failed
```

### Root Cause

The `scripts/` directory contained multiple Go files with `main()` functions:
- `scripts/genhash.go` - Password hash generator utility
- `scripts/test_phase2.go` - Phase 2 migration test utility
- `scripts/update_admin_password.go` - Admin password update utility (already had build constraint)

When running `go test ./...`, Go tries to compile the `scripts` directory as a package. Multiple `main()` functions in the same package cause a compilation error.

---

## Solution

### Fix Applied

Added `//go:build ignore` build constraint to the top of standalone script files.

**Files Modified:**

1. **`scripts/genhash.go`**

**Before:**
```go
// Quick utility to generate bcrypt hash for test password
// Usage: go run scripts/genhash.go test123

package main
```

**After:**
```go
//go:build ignore

// Quick utility to generate bcrypt hash for test password
// Usage: go run scripts/genhash.go test123

package main
```

2. **`scripts/test_phase2.go`**

**Before:**
```go
package main

import (
    // imports...
)
```

**After:**
```go
//go:build ignore

package main

import (
    // imports...
)
```

**Note:** `scripts/update_admin_password.go` already had the old-style constraint `// +build ignore` and didn't need changes.

---

## How Build Constraints Work

### What is `//go:build ignore`?

This is a build constraint (also called a build tag) that tells the Go compiler to ignore this file during normal builds and tests.

**Syntax:**
```go
//go:build ignore  // Modern syntax (Go 1.17+)
```

Or the older form:
```go
// +build ignore  // Old syntax (still works)
```

**Effect:**
- File is excluded from normal `go build` and `go test`
- File can still be run directly with `go run scripts/filename.go`
- Prevents package compilation conflicts

### When to Use

Use build constraints for:
- Standalone utility scripts
- Example programs
- Tools that aren't part of the main application
- Files with multiple main() functions in the same directory

**Do NOT use for:**
- Regular test files (`*_test.go`)
- Application source code
- Library code

---

## Verification

### Before Fix

```bash
$ go test ./...
FAIL	github.com/tranm/gassigeher/scripts [build failed]
FAIL
```

### After Fix

```bash
$ go test ./...
?   	github.com/tranm/gassigeher/cmd/server	[no test files]
?   	github.com/tranm/gassigeher/internal/config	[no test files]
ok  	github.com/tranm/gassigeher/internal/cron	(cached)
?   	github.com/tranm/gassigeher/internal/database	[no test files]
ok  	github.com/tranm/gassigeher/internal/handlers	(cached)
ok  	github.com/tranm/gassigeher/internal/middleware	(cached)
ok  	github.com/tranm/gassigeher/internal/models	(cached)
ok  	github.com/tranm/gassigeher/internal/repository	(cached)
ok  	github.com/tranm/gassigeher/internal/services	(cached)
?   	github.com/tranm/gassigeher/internal/testutil	[no test files]
```

**Result:** ✅ All tests passing

### bat.bat Output

```
========================================
Build and Test Complete!
========================================

To run the application:
  .\gassigeher.exe

[OK] All tests passed
```

**Result:** ✅ Success

---

## Impact

### What Changed

- **Code Functionality:** None - scripts still work the same way
- **Test Execution:** Scripts excluded from `go test ./...`
- **Build Process:** Scripts excluded from normal builds
- **Usage:** Scripts can still be run directly with `go run`

### How to Run Scripts

**Still works:**
```bash
# Generate password hash
go run scripts/genhash.go test123

# Run Phase 2 test
go run scripts/test_phase2.go

# Update admin password
go run scripts/update_admin_password.go
```

**Build constraint doesn't affect direct execution.**

---

## Files with Build Constraints

### Current Status

| File | Build Constraint | Reason |
|------|------------------|--------|
| `scripts/genhash.go` | `//go:build ignore` | Standalone utility with main() |
| `scripts/test_phase2.go` | `//go:build ignore` | Standalone test with main() |
| `scripts/update_admin_password.go` | `// +build ignore` | Standalone utility with main() |

### Why Different Syntax?

- `//go:build ignore` - Modern syntax (Go 1.17+, recommended)
- `// +build ignore` - Old syntax (still works, backward compatible)

Both work the same way. New files should use `//go:build` (modern syntax).

---

## Best Practices

### For Future Script Files

When creating new standalone scripts in `scripts/` directory:

**Template:**
```go
//go:build ignore

// Script description
// Usage: go run scripts/myscript.go [args]

package main

import (
    // imports
)

func main() {
    // script logic
}
```

**Always add `//go:build ignore` at the very top** (line 1) of standalone scripts.

### Directory Structure Recommendation

**Alternative approach** (not implemented, but valid):

```
project/
├── cmd/
│   ├── server/          # Main application
│   ├── genhash/         # Each utility in own directory
│   └── testphase2/      # Separate directory = no conflict
└── scripts/             # PowerShell/Bash scripts only
```

This approach gives each utility its own directory, avoiding the multiple main() issue entirely.

**Current approach is simpler** - use build constraints.

---

## Testing After Fix

### Run All Tests

```bash
# Full test suite
go test ./... -v

# Or use bat.bat
.\bat.bat

# Or use bat.sh (Linux/Mac)
./bat.sh
```

**Expected Output:**
```
[OK] All tests passed

Build and Test Complete!
```

### Verify Scripts Still Work

```bash
# Test each script can still be run directly
go run scripts/genhash.go test123
# Should output: Password: test123, Bcrypt Hash: ...

go run scripts/test_phase2.go
# Should run Phase 2 migration tests

go run scripts/update_admin_password.go
# Should update admin password
```

---

## Summary

**Problem:** Multiple `main()` functions in scripts directory caused build failure during testing.

**Solution:** Added `//go:build ignore` build constraint to standalone script files.

**Files Modified:** 2
- `scripts/genhash.go`
- `scripts/test_phase2.go`

**Test Result:** ✅ All tests now passing (100%)

**Build Result:** ✅ Build successful

**Impact:** None - scripts still work when run directly

**Status:** ✅ **ISSUE RESOLVED**

---

## Lessons Learned

1. **Standalone Scripts Need Build Constraints**
   - Any script with `main()` in a shared directory needs `//go:build ignore`
   - Prevents package compilation conflicts

2. **Go Test Runs on All Packages**
   - `go test ./...` includes all directories
   - Directories with Go files are treated as packages
   - Must handle utility scripts appropriately

3. **Build Constraints Are Simple**
   - Just add `//go:build ignore` at line 1
   - No other changes needed
   - Scripts still work with `go run`

4. **Modern vs Old Syntax**
   - `//go:build ignore` - Modern (recommended)
   - `// +build ignore` - Old (still works)
   - Use modern syntax for new files

---

**Fix Verified:** ✅ All tests passing

**Build Status:** ✅ Successful

**Production Ready:** ✅ Yes (after Phase 1)
