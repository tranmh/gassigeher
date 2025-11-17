# Test Strategy - Gassigeher

**Goal**: Achieve 90% line code coverage across all packages
**Current Coverage**: ~15% overall (Models: 50%, Repository: 6.3%, Services: 18.7%, Handlers: 0%)
**Target Date**: [To be defined]

## Executive Summary

This document outlines a comprehensive testing strategy to achieve 90% code coverage and prevent bugs through systematic testing at all levels: Unit, Integration, API, and End-to-End tests.

## Current State Analysis

```
Package                    Current Coverage    Target Coverage    Gap
─────────────────────────────────────────────────────────────────────
internal/models            50.0%              90%                40%
internal/repository        6.3%               90%                83.7%
internal/services          18.7%              90%                71.3%
internal/handlers          0.0%               90%                90%
internal/middleware        0.0%               90%                90%
internal/database          0.0%               85%                85%
internal/cron              0.0%               85%                85%
internal/config            0.0%               80%                80%
cmd/server                 0.0%               70%                70%
─────────────────────────────────────────────────────────────────────
OVERALL                    ~15%               90%                75%
```

## Test Pyramid

```
                    /\
                   /  \
                  / E2E \         ~5-10 tests (Browser automation)
                 /      \
                /________\
               /          \
              /    API     \      ~50 tests (HTTP endpoints)
             /              \
            /________________\
           /                  \
          /    Integration     \  ~100 tests (DB + Services)
         /                      \
        /________________________\
       /                          \
      /         Unit Tests          \ ~300 tests (Models, Utils, Logic)
     /________________________________\
```

## Testing Stack

### 1. Unit Tests
- **Tool**: Go native `testing` package + `testify/assert`
- **Purpose**: Test individual functions, models, business logic
- **Coverage Target**: 95%

### 2. Integration Tests
- **Tool**: Go `testing` + `testify/suite` + in-memory SQLite
- **Purpose**: Test database operations, service interactions
- **Coverage Target**: 90%

### 3. API Tests
- **Tool**: `httptest` + `testify`
- **Purpose**: Test HTTP handlers and middleware
- **Coverage Target**: 90%

### 4. E2E Tests
- **Tool**: Playwright (Go bindings)
- **Purpose**: Test complete user workflows in browser
- **Coverage Target**: Critical paths only

## Test Data Strategy

### Approach 1: In-Memory SQLite (Unit/Integration Tests)
```go
func setupTestDB(t *testing.T) *sql.DB {
    db, err := sql.Open("sqlite3", ":memory:")
    require.NoError(t, err)

    // Run migrations
    database.RunMigrations(db)

    t.Cleanup(func() { db.Close() })
    return db
}
```

### Approach 2: Test Fixtures (Integration Tests)
```go
func seedTestData(db *sql.DB) {
    // Insert known test data
    users := []User{
        {Email: "test@example.com", Name: "Test User"},
    }
    // ... insert fixtures
}
```

### Approach 3: Transaction Rollback (API Tests)
```go
func TestWithRollback(t *testing.T) {
    tx, _ := db.Begin()
    defer tx.Rollback()

    // Run test operations
    // Automatically rolled back
}
```

## Implementation Roadmap

### Phase 1: Foundation (Week 1-2) - Target: 40% Coverage

#### 1.1 Complete Model Tests
**Files**: `internal/models/*_test.go`

```go
// Example: internal/models/booking_test.go
func TestBooking_Validate(t *testing.T) {
    tests := []struct {
        name    string
        booking Booking
        wantErr bool
        errMsg  string
    }{
        {
            name: "valid booking",
            booking: Booking{
                UserID:        1,
                DogID:         1,
                Date:          "2025-12-01",
                WalkType:      "morning",
                ScheduledTime: "09:00",
            },
            wantErr: false,
        },
        {
            name: "invalid date format",
            booking: Booking{
                Date: "01-12-2025", // Wrong format
            },
            wantErr: true,
            errMsg:  "invalid date format",
        },
        // ... more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.booking.Validate()
            if tt.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

**Models to Test**:
- [x] `Booking` (50% done)
- [ ] `User`
- [ ] `Dog`
- [ ] `BlockedDate`
- [ ] `ExperienceRequest`
- [ ] `ReactivationRequest`

#### 1.2 Repository Layer Tests
**Files**: `internal/repository/*_test.go`

```go
// Example: internal/repository/user_repository_test.go
func TestUserRepository(t *testing.T) {
    db := setupTestDB(t)
    repo := NewUserRepository(db)

    t.Run("Create", func(t *testing.T) {
        user := &models.User{
            Email:    "test@example.com",
            Name:     "Test User",
            Password: "hashed_password",
        }

        err := repo.Create(user)
        assert.NoError(t, err)
        assert.NotZero(t, user.ID)
    })

    t.Run("FindByEmail", func(t *testing.T) {
        // Create test user
        user := &models.User{Email: "find@example.com", Name: "Find Me"}
        repo.Create(user)

        // Find by email
        found, err := repo.FindByEmail("find@example.com")
        assert.NoError(t, err)
        assert.Equal(t, user.Email, found.Email)
    })

    t.Run("FindByEmail_NotFound", func(t *testing.T) {
        _, err := repo.FindByEmail("nonexistent@example.com")
        assert.Error(t, err)
        assert.Equal(t, sql.ErrNoRows, err)
    })
}
```

**Repositories to Test**:
- [x] `UserRepository` (6.3% done)
- [ ] `DogRepository`
- [ ] `BookingRepository`
- [ ] `BlockedDateRepository`
- [ ] `ExperienceRequestRepository`
- [ ] `ReactivationRequestRepository`
- [ ] `SystemSettingsRepository`

### Phase 2: Business Logic (Week 3-4) - Target: 65% Coverage

#### 2.1 Service Layer Tests
**Files**: `internal/services/*_test.go`

```go
// Example: internal/services/auth_service_test.go
func TestAuthService(t *testing.T) {
    service := NewAuthService("test-jwt-secret")

    t.Run("HashPassword", func(t *testing.T) {
        password := "test123"
        hash, err := service.HashPassword(password)

        assert.NoError(t, err)
        assert.NotEmpty(t, hash)
        assert.NotEqual(t, password, hash)
    })

    t.Run("ComparePassword_Valid", func(t *testing.T) {
        password := "test123"
        hash, _ := service.HashPassword(password)

        err := service.ComparePassword(hash, password)
        assert.NoError(t, err)
    })

    t.Run("ComparePassword_Invalid", func(t *testing.T) {
        hash, _ := service.HashPassword("test123")

        err := service.ComparePassword(hash, "wrong")
        assert.Error(t, err)
    })

    t.Run("GenerateJWT", func(t *testing.T) {
        user := &models.User{
            ID:    1,
            Email: "test@example.com",
        }

        token, err := service.GenerateJWT(user, false)
        assert.NoError(t, err)
        assert.NotEmpty(t, token)
    })

    t.Run("ValidateJWT", func(t *testing.T) {
        user := &models.User{ID: 1, Email: "test@example.com"}
        token, _ := service.GenerateJWT(user, false)

        claims, err := service.ValidateJWT(token)
        assert.NoError(t, err)
        assert.Equal(t, user.ID, claims.UserID)
        assert.Equal(t, user.Email, claims.Email)
    })
}
```

**Services to Test**:
- [x] `AuthService` (18.7% done)
- [ ] `EmailService` (mock SMTP/Gmail API)
- [ ] Booking validation logic
- [ ] Experience level checks

#### 2.2 Cron Job Tests
**Files**: `internal/cron/cron_test.go`

```go
func TestCronService(t *testing.T) {
    db := setupTestDB(t)
    cronService := NewCronService(db)

    t.Run("AutoCompleteBookings", func(t *testing.T) {
        // Create past booking
        booking := &models.Booking{
            Date:   "2025-01-01", // Past date
            Status: "scheduled",
        }
        // ... insert booking

        // Run auto-complete
        cronService.AutoCompleteBookings()

        // Verify status changed
        updated, _ := bookingRepo.FindByID(booking.ID)
        assert.Equal(t, "completed", updated.Status)
    })
}
```

### Phase 3: HTTP Layer (Week 5-6) - Target: 85% Coverage

#### 3.1 Middleware Tests
**Files**: `internal/middleware/middleware_test.go`

```go
func TestAuthMiddleware(t *testing.T) {
    // Setup
    authService := services.NewAuthService("test-secret")
    middleware := AuthMiddleware(authService)

    t.Run("Valid Token", func(t *testing.T) {
        // Create valid token
        user := &models.User{ID: 1, Email: "test@example.com"}
        token, _ := authService.GenerateJWT(user, false)

        // Create request with token
        req := httptest.NewRequest("GET", "/api/users/me", nil)
        req.Header.Set("Authorization", "Bearer "+token)

        rec := httptest.NewRecorder()
        next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Verify context has user info
            userID := r.Context().Value("userID")
            assert.Equal(t, 1, userID)
            w.WriteHeader(http.StatusOK)
        })

        middleware(next).ServeHTTP(rec, req)
        assert.Equal(t, http.StatusOK, rec.Code)
    })

    t.Run("Missing Token", func(t *testing.T) {
        req := httptest.NewRequest("GET", "/api/users/me", nil)
        rec := httptest.NewRecorder()
        next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            t.Fatal("Should not reach handler")
        })

        middleware(next).ServeHTTP(rec, req)
        assert.Equal(t, http.StatusUnauthorized, rec.Code)
    })
}
```

#### 3.2 Handler Tests
**Files**: `internal/handlers/*_test.go`

```go
// Example: internal/handlers/auth_handler_test.go
func TestAuthHandler_Login(t *testing.T) {
    // Setup
    db := setupTestDB(t)
    cfg := &config.Config{JWTSecret: "test-secret"}
    handler := NewAuthHandler(db, cfg)

    // Create test user
    userRepo := repository.NewUserRepository(db)
    authService := services.NewAuthService(cfg.JWTSecret)
    hash, _ := authService.HashPassword("test123")
    testUser := &models.User{
        Email:        "test@example.com",
        Name:         "Test User",
        PasswordHash: hash,
        IsVerified:   true,
    }
    userRepo.Create(testUser)

    t.Run("Valid Login", func(t *testing.T) {
        body := `{"email":"test@example.com","password":"test123"}`
        req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
        req.Header.Set("Content-Type", "application/json")

        rec := httptest.NewRecorder()
        handler.Login(rec, req)

        assert.Equal(t, http.StatusOK, rec.Code)

        var response map[string]interface{}
        json.Unmarshal(rec.Body.Bytes(), &response)
        assert.NotEmpty(t, response["token"])
    })

    t.Run("Invalid Password", func(t *testing.T) {
        body := `{"email":"test@example.com","password":"wrong"}`
        req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
        req.Header.Set("Content-Type", "application/json")

        rec := httptest.NewRecorder()
        handler.Login(rec, req)

        assert.Equal(t, http.StatusUnauthorized, rec.Code)
    })

    t.Run("User Not Found", func(t *testing.T) {
        body := `{"email":"nonexistent@example.com","password":"test123"}`
        req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))

        rec := httptest.NewRecorder()
        handler.Login(rec, req)

        assert.Equal(t, http.StatusUnauthorized, rec.Code)
    })
}
```

**Handlers to Test** (50+ endpoints):
- [ ] `AuthHandler` (Register, Login, Verify, ForgotPassword, ResetPassword)
- [ ] `UserHandler` (GetMe, UpdateMe, UploadPhoto, DeleteAccount)
- [ ] `DogHandler` (List, Get, Create, Update, Delete, Toggle Availability)
- [ ] `BookingHandler` (List, Create, Get, Cancel, AddNotes, Calendar)
- [ ] `BlockedDateHandler` (List, Create, Delete)
- [ ] `ExperienceRequestHandler` (Create, List, Review)
- [ ] `ReactivationRequestHandler` (Create, List, Review)
- [ ] `DashboardHandler` (GetStats, GetActivity)

### Phase 4: End-to-End Tests (Week 7-8) - Target: 90% Coverage

#### 4.1 E2E Test Setup
**Tool**: Playwright for Go

```bash
go get github.com/playwright-community/playwright-go
```

**File**: `tests/e2e/e2e_test.go`

```go
package e2e

import (
    "testing"
    "github.com/playwright-community/playwright-go"
    "github.com/stretchr/testify/assert"
)

func TestE2E(t *testing.T) {
    // Start test server
    server := startTestServer(t)
    defer server.Close()

    // Setup Playwright
    pw, err := playwright.Run()
    require.NoError(t, err)
    defer pw.Stop()

    browser, err := pw.Chromium.Launch()
    require.NoError(t, err)
    defer browser.Close()

    page, err := browser.NewPage()
    require.NoError(t, err)

    t.Run("User Registration Flow", func(t *testing.T) {
        // Navigate to registration
        _, err := page.Goto(server.URL + "/register.html")
        assert.NoError(t, err)

        // Fill form
        page.Fill("#email", "newuser@example.com")
        page.Fill("#name", "New User")
        page.Fill("#phone", "+49 123 456789")
        page.Fill("#password", "test123")
        page.Check("#terms")

        // Submit
        page.Click("button[type=submit]")

        // Wait for success message
        page.WaitForSelector(".alert-success")

        // Verify alert text
        text, _ := page.TextContent(".alert-success")
        assert.Contains(t, text, "Registrierung erfolgreich")
    })

    t.Run("Login and Booking Flow", func(t *testing.T) {
        // Login
        _, err := page.Goto(server.URL + "/login.html")
        assert.NoError(t, err)

        page.Fill("#email", "test@example.com")
        page.Fill("#password", "test123")
        page.Click("button[type=submit]")

        // Wait for redirect to dashboard
        page.WaitForURL("**/dashboard.html")

        // Navigate to dogs
        page.Click("a[href='/dogs.html']")
        page.WaitForURL("**/dogs.html")

        // Click first available dog
        page.Click(".dog-card:first-child button")

        // Wait for modal
        page.WaitForSelector("#booking-modal")

        // Fill booking form
        page.Fill("#booking-date", "2025-12-25")
        page.SelectOption("#booking-walk-type", "morning")
        page.SelectOption("#booking-time", "09:00")

        // Submit booking
        page.Click("#booking-form button[type=submit]")

        // Wait for success
        page.WaitForSelector(".alert-success")

        // Verify booking appears in dashboard
        page.WaitForURL("**/dashboard.html")
        bookingExists := page.Locator(".booking-card").Count() > 0
        assert.True(t, bookingExists)
    })
}
```

**Critical E2E Test Cases**:
1. User Registration → Email Verification → Login
2. Login → Browse Dogs → Create Booking → View Dashboard
3. Login → Profile → Upload Photo → Update Info
4. Admin Login → Manage Dogs → Toggle Availability
5. Admin → Review Experience Requests → Approve/Deny
6. User → Cancel Booking (within notice period)
7. User → Add Walk Notes (completed booking)
8. User → Request Experience Level Upgrade
9. Login → Calendar View → Quick Book
10. User → Account Deletion (GDPR)

## Coverage Measurement

### Generate Coverage Report
```bash
# Run all tests with coverage
go test ./... -coverprofile=coverage.out -covermode=atomic

# View coverage in terminal
go tool cover -func=coverage.out

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html

# View coverage by package
go test ./internal/handlers -cover
go test ./internal/repository -cover
go test ./internal/services -cover
```

### Coverage Goals by Package
```bash
# Check if target met
go test ./internal/handlers -cover | grep "coverage:" | awk '{if ($4 < 90) exit 1}'
```

### CI/CD Integration (Future)
**File**: `.github/workflows/test.yml`

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    steps:
    - uses: actions/checkout@v3

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'

    - name: Install dependencies
      run: go mod download

    - name: Run tests
      run: go test ./... -coverprofile=coverage.out -covermode=atomic

    - name: Check coverage threshold
      run: |
        coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
        if (( $(echo "$coverage < 90.0" | bc -l) )); then
          echo "Coverage is $coverage%, below 90% threshold"
          exit 1
        fi
        echo "Coverage is $coverage%"

    - name: Upload coverage to Codecov
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out
```

## Test Utilities and Helpers

### Test Helper Package
**File**: `internal/testutil/helpers.go`

```go
package testutil

import (
    "database/sql"
    "testing"
    _ "github.com/mattn/go-sqlite3"
    "github.com/stretchr/testify/require"
    "github.com/tranm/gassigeher/internal/database"
)

// SetupTestDB creates an in-memory SQLite database for testing
func SetupTestDB(t *testing.T) *sql.DB {
    db, err := sql.Open("sqlite3", ":memory:")
    require.NoError(t, err)

    // Run migrations
    err = database.RunMigrations(db)
    require.NoError(t, err)

    t.Cleanup(func() {
        db.Close()
    })

    return db
}

// SeedTestUser creates a test user
func SeedTestUser(t *testing.T, db *sql.DB, email string) int {
    result, err := db.Exec(`
        INSERT INTO users (email, name, password_hash, experience_level, is_verified, terms_accepted_at)
        VALUES (?, 'Test User', 'hash', 'green', 1, datetime('now'))
    `, email)
    require.NoError(t, err)

    id, _ := result.LastInsertId()
    return int(id)
}

// SeedTestDog creates a test dog
func SeedTestDog(t *testing.T, db *sql.DB, name string) int {
    result, err := db.Exec(`
        INSERT INTO dogs (name, breed, category, is_available, created_at)
        VALUES (?, 'Test Breed', 'green', 1, datetime('now'))
    `, name)
    require.NoError(t, err)

    id, _ := result.LastInsertId()
    return int(id)
}
```

## Mocking Strategy

### Mock External Services
**File**: `internal/services/mocks/email_service.go`

```go
package mocks

import (
    "github.com/stretchr/testify/mock"
)

type MockEmailService struct {
    mock.Mock
}

func (m *MockEmailService) SendVerificationEmail(to, name, token string) error {
    args := m.Called(to, name, token)
    return args.Error(0)
}

func (m *MockEmailService) SendBookingConfirmation(to, dogName, date, time string) error {
    args := m.Called(to, dogName, date, time)
    return args.Error(0)
}

// Usage in tests:
// mockEmail := new(mocks.MockEmailService)
// mockEmail.On("SendVerificationEmail", "test@example.com", "Test", mock.Anything).Return(nil)
```

## Performance Testing (Bonus)

### Load Test
**Tool**: `vegeta` or `k6`

```bash
# Install vegeta
go install github.com/tsenart/vegeta@latest

# Run load test
echo "GET http://localhost:8080/api/dogs" | vegeta attack -duration=30s -rate=50 | vegeta report
```

## Best Practices

### 1. Test Naming Convention
```go
// Pattern: Test<FunctionName>_<Scenario>_<ExpectedResult>
func TestUserRepository_Create_Success(t *testing.T) { }
func TestUserRepository_Create_DuplicateEmail_ReturnsError(t *testing.T) { }
func TestBooking_Validate_InvalidDate_ReturnsError(t *testing.T) { }
```

### 2. Table-Driven Tests
```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid", "test@example.com", false},
        {"invalid", "not-an-email", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test logic
        })
    }
}
```

### 3. Test Independence
- Each test should be independent
- Use `t.Run()` for subtests
- Always clean up resources

### 4. Meaningful Assertions
```go
// Bad
assert.True(t, user.ID > 0)

// Good
assert.NotZero(t, user.ID, "User ID should be set after creation")
```

## Progress Tracking

### Weekly Coverage Goals

| Week | Target Coverage | Focus Areas | Deliverables |
|------|----------------|-------------|--------------|
| 1 | 40% | Models + Repositories | Complete model validation tests, Basic CRUD tests |
| 2 | 50% | Repository completion | All repository methods tested |
| 3 | 65% | Services | Auth, Email service tests (mocked) |
| 4 | 75% | Cron + Middleware | Automated job tests, Auth middleware tests |
| 5 | 85% | Handlers (part 1) | Auth, User, Dog handlers |
| 6 | 90% | Handlers (part 2) | Booking, Admin handlers |
| 7 | 90%+ | E2E tests | Critical user flows |
| 8 | 90%+ | Coverage gaps | Fill remaining gaps, refine tests |

### Daily Test Checklist
```bash
#!/bin/bash
# run-tests.sh

echo "Running all tests..."
go test ./... -v -cover

echo ""
echo "Generating coverage report..."
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total

echo ""
echo "Coverage by package:"
go test ./internal/models -cover
go test ./internal/repository -cover
go test ./internal/services -cover
go test ./internal/handlers -cover
go test ./internal/middleware -cover

echo ""
echo "Opening HTML report..."
go tool cover -html=coverage.out
```

## Common Pitfalls to Avoid

1. **Testing Implementation, Not Behavior**
   - ❌ Test internal variables
   - ✅ Test public API and observable behavior

2. **Flaky Tests**
   - ❌ Tests that randomly fail
   - ✅ Use deterministic test data, avoid time.Now() in tests

3. **Slow Tests**
   - ❌ Testing with real database/API calls unnecessarily
   - ✅ Use in-memory DB, mock external services

4. **Over-Mocking**
   - ❌ Mocking everything
   - ✅ Mock only external dependencies (email, payment)

5. **Not Testing Error Cases**
   - ❌ Only testing happy path
   - ✅ Test error conditions, edge cases, validation failures

## Success Metrics

- [ ] 90%+ line coverage across all packages
- [ ] All critical paths have E2E tests
- [ ] All handlers have API tests
- [ ] All repository methods tested
- [ ] All service logic tested
- [ ] All validation logic tested
- [ ] No flaky tests (0% failure rate on reruns)
- [ ] Test suite runs in < 2 minutes
- [ ] Documentation for all test utilities
- [ ] Coverage report generated on each run

## Next Steps

1. **Immediate Actions**:
   - Install testify: `go get github.com/stretchr/testify`
   - Create `internal/testutil/helpers.go`
   - Start with model tests (easiest wins)

2. **Week 1 Goals**:
   - Complete all model validation tests
   - Set up test database helpers
   - Write first repository tests

3. **Documentation**:
   - Update this document after each phase
   - Document test utilities and patterns
   - Share learnings with team

## Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Testify GitHub](https://github.com/stretchr/testify)
- [Table-Driven Tests in Go](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Advanced Testing in Go (Video)](https://www.youtube.com/watch?v=8hQG7QlcLBk)

---

**Last Updated**: [Current Date]
**Owner**: Development Team
**Review Frequency**: Weekly during test implementation phase
