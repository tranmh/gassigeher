# BASE_URL Configuration Changes

## Summary

All hardcoded `localhost:8080` URLs have been replaced with a configurable `BASE_URL` environment variable. This allows the application to work correctly in production environments.

## Changes Made

### 1. Configuration Layer

**File: `internal/config/config.go`**
- Added `BaseURL string` field to `Config` struct
- Default value: `"http://localhost:8080"` (for development)
- Loaded from `BASE_URL` environment variable

**File: `.env.example`**
- Added `BASE_URL=http://localhost:8080` with documentation comment

### 2. Email Service

**File: `internal/services/email_provider.go`**
- Added `BaseURL string` field to `EmailConfig` struct

**File: `internal/services/email_provider_factory.go`**
- Updated `ConfigToEmailConfig()` to pass `BaseURL` from config

**File: `internal/services/email_service.go`**
- Added `baseURL string` field to `EmailService` struct
- Updated `NewEmailService()` to store and use `baseURL`
- Updated 4 email templates to use `{{.BaseURL}}` instead of hardcoded URLs:
  - `SendVerificationEmail()` - Verification link
  - `SendWelcomeEmail()` - Welcome link
  - `SendPasswordResetEmail()` - Reset password link
  - `SendExperienceLevelApproved()` - Dogs page link

**File: `internal/services/email_account.go`**
- Updated 1 email template to use `{{.BaseURL}}`:
  - `SendAccountReactivated()` - Login link

### 3. CORS Middleware

**File: `internal/middleware/middleware.go`**
- Changed `CORSMiddleware` signature from `func(http.Handler) http.Handler` to `func(baseURL string) func(http.Handler) http.Handler`
- Made CORS allowed origins configurable based on `baseURL`
- Removed hardcoded `http://localhost:8080` from CORS origins
- Added fallback to `http://localhost:8080` if `baseURL` is empty

**File: `cmd/server/main.go`**
- Updated middleware initialization: `middleware.CORSMiddleware(cfg.BaseURL)`

**File: `internal/middleware/middleware_test.go`**
- Updated test to use new signature: `CORSMiddleware("http://localhost:8080")(testHandler)`

## Usage

### Development (default)
```bash
# No configuration needed - defaults to localhost:8080
BASE_URL=http://localhost:8080
```

Or simply omit the variable - it will default to `http://localhost:8080`.

### Production
```bash
# Set in .env file
BASE_URL=https://gassigeher.tierheim-goeppingen.de
```

Or set as environment variable:
```bash
export BASE_URL=https://gassigeher.tierheim-goeppingen.de
```

## Remaining Localhost References

All remaining `localhost` references in production code are **appropriate defaults**:

| File | Line | Purpose | Status |
|------|------|---------|--------|
| `config/config.go` | 88 | Database host default | ✅ Correct |
| `config/config.go` | 141 | BASE_URL env var default | ✅ Correct |
| `database/database.go` | 120, 152 | Database connection defaults | ✅ Correct |
| `middleware/middleware.go` | 51 | CORS fallback if baseURL empty | ✅ Correct |
| `services/email_service.go` | 46 | Email service fallback | ✅ Correct |

## Testing

All tests pass:
```bash
go test ./internal/middleware/... -v
# PASS (all CORS, Auth, Security, Logging tests)

go build -o gassigeher.exe ./cmd/server
# Build successful
```

## Benefits

1. ✅ **Production Ready** - Works with any domain
2. ✅ **Email Links** - All email links use correct domain
3. ✅ **CORS** - CORS origins match deployment URL
4. ✅ **Backward Compatible** - Defaults to localhost for development
5. ✅ **Single Source of Truth** - One environment variable controls all URLs

## Migration Guide

For existing deployments:

1. Add to your `.env` file or environment:
   ```bash
   BASE_URL=https://your-domain.com
   ```

2. Restart the application:
   ```bash
   ./gassigeher
   ```

3. Verify:
   - Email links point to your domain
   - CORS allows requests from your domain
   - No hardcoded localhost references in emails

## Example Production Configuration

```bash
# Production .env file
PORT=8080
BASE_URL=https://gassigeher.tierheim-goeppingen.de

# Database
DB_TYPE=postgres
DB_HOST=localhost
DB_PORT=5432
DB_NAME=gassigeher
DB_USER=gassigeher_user
DB_PASSWORD=secure_password

# Email (SMTP)
EMAIL_PROVIDER=smtp
SMTP_HOST=smtp.strato.de
SMTP_PORT=465
SMTP_USERNAME=noreply@tierheim-goeppingen.de
SMTP_PASSWORD=your-password
SMTP_FROM_EMAIL=noreply@tierheim-goeppingen.de
SMTP_USE_SSL=true

# JWT
JWT_SECRET=your-secure-jwt-secret

# Super Admin
SUPER_ADMIN_EMAIL=admin@tierheim-goeppingen.de
```

## Notes

- The BASE_URL should **not** have a trailing slash
- Use `https://` in production for security
- Email links will use exactly the BASE_URL you configure
- CORS will allow requests from BASE_URL origin
