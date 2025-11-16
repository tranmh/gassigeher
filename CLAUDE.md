# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Gassigeher is a **complete production-ready** dog walking booking system for animal shelters. Built with Go backend + SQLite and Vanilla JavaScript frontend. All 10 implementation phases are complete.

**Status**: ✅ Production ready, fully functional, deployment package included.

> **Essential Reading**:
> - [ImplementationPlan.md](ImplementationPlan.md) - Complete architecture, all 10 phases
> - [API.md](API.md) - All 50+ endpoints with request/response examples
> - [DEPLOYMENT.md](DEPLOYMENT.md) - Production deployment steps

---

## Build & Test Commands

### Build Application

**Windows:**
```cmd
bat.bat
```

**Linux/Mac:**
```bash
chmod +x bat.sh
./bat.sh
```

These scripts will download dependencies, build the binary, and run tests.

**Manual build:**
```bash
go build -o gassigeher.exe ./cmd/server    # Windows
go build -o gassigeher ./cmd/server        # Linux/Mac
```

### Run Application

```bash
# Development mode
go run cmd/server/main.go

# Using compiled binary
./gassigeher.exe    # Windows
./gassigeher        # Linux/Mac
```

Server starts on `http://localhost:8080` (configurable via `PORT` environment variable).

### Testing

```bash
# Run all tests
go test ./... -v

# Run specific package tests
go test ./internal/services/... -v
go test ./internal/models/... -v
go test ./internal/repository/... -v

# Run single test
go test ./internal/services/... -run TestAuthService_HashPassword -v

# Coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

**Current Coverage:**
- Auth service: 18.7% (7 tests passing)
- Models: 50% (9 tests passing)
- Repository: 6.3% (4 tests passing)

## Architecture Overview

### Three-Layer Backend Architecture

**1. Handlers** (`internal/handlers/`)
- HTTP request/response handling
- Input validation
- Context extraction (user_id, is_admin)
- Calls services/repositories
- **Pattern**: Each handler owns its dependencies (repos, services, config)

**2. Repositories** (`internal/repository/`)
- Direct database operations
- SQL query construction
- No business logic
- **Pattern**: One repository per model, returns models only

**3. Services** (`internal/services/`)
- Business logic (auth, email)
- Independent of HTTP layer
- **AuthService**: JWT, password hashing, token generation
- **EmailService**: Gmail API, HTML templates

### Request Flow

```
HTTP Request
    ↓
Middleware (Logging → Security → CORS → Auth → Admin?)
    ↓
Handler (validate input, check auth)
    ↓
Repository (database query)
    ↓
Response (JSON)
```

### Key Patterns

**Authentication Flow:**
1. `AuthMiddleware` extracts JWT from `Authorization: Bearer <token>` header
2. Validates token using `AuthService.ValidateJWT()`
3. Injects into context: `user_id`, `email`, `is_admin`
4. Handlers access via `r.Context().Value(middleware.UserIDKey)`

**Admin Authorization:**
- Admins defined in `ADMIN_EMAILS` env var (config-based, not DB)
- `RequireAdmin` middleware checks `is_admin` context value
- Applied to protected routes via subrouter

**GDPR Anonymization:**
- `UserRepository.DeleteAccount()` sets:
  - `name = "Deleted User"`
  - `email = NULL, phone = NULL, password_hash = NULL`
  - `is_deleted = 1, anonymous_id = "anonymous_user_<timestamp>"`
- Walk history preserved but shows "Deleted User"
- Legal basis: Legitimate interest (dog care records)

**Experience Level Enforcement:**
- Helper: `repository.CanUserAccessDog(userLevel, dogCategory)`
- Levels: green (1) → blue (2) → orange (3)
- Users can only book dogs at or below their level
- Frontend shows locked dogs with 🔒 icon

## Critical Implementation Details

### Email Service Initialization

Email service can fail gracefully. Pattern used in handlers:

```go
emailService, err := services.NewEmailService(cfg.GmailClientID, cfg.GmailClientSecret, cfg.GmailRefreshToken, cfg.GmailFromEmail)
if err != nil {
    // Log but don't fail - emails will fail gracefully
    fmt.Printf("Warning: Failed to initialize email service: %v\n", err)
}
```

All email sends are in goroutines: `go emailService.SendX(...)`

### User Activity Tracking

Critical for auto-deactivation (365-day inactivity default):
- Updated on: login, booking creation, booking cancellation
- Method: `userRepo.UpdateLastActivity(userID)`
- **Must call after any user action that counts as "activity"**

### Booking Validation Chain

When creating bookings, validate in this order:
1. Request format (date, time, walk_type)
2. User is active (`user.IsActive`)
3. Dog exists and is available (`dog.IsAvailable`)
4. User has required level (`CanUserAccessDog()`)
5. Date not in past
6. Date within advance limit (default 14 days from settings)
7. Date not blocked (`blockedDateRepo.IsBlocked()`)
8. No double-booking (`bookingRepo.CheckDoubleBooking()`)

### Cron Jobs

Three automated jobs in `internal/cron/cron.go`:
1. **Auto-complete**: Runs hourly via `runPeriodically()`
2. **Auto-deactivate**: Runs daily at 3am via `runDaily()`
3. **Reminders**: Placeholder exists, currently disabled

Started in `main.go`:
```go
cronService := cron.NewCronService(db)
cronService.Start()
defer cronService.Stop()
```

### Frontend API Client

Global instance: `window.api` (from `/js/api.js`)

**Key methods:**
- Authentication: `api.login()`, `api.register()`, `api.logout()`
- Users: `api.getMe()`, `api.updateMe()`, `api.deleteAccount()`
- Dogs: `api.getDogs(filters)`, `api.createDog()`, `api.toggleDogAvailability()`
- Bookings: `api.createBooking()`, `api.getBookings()`, `api.cancelBooking()`
- Admin: `api.getAdminStats()`, `api.getUsers(activeOnly)`

**Token management:**
- Stored in `localStorage['gassigeher_token']`
- Sent as `Authorization: Bearer <token>` header
- Cleared on logout: `api.setToken(null)`

### i18n System

Global instance: `window.i18n` (from `/js/i18n.js`)

**Usage:**
```javascript
await window.i18n.load();  // Loads de.json
i18n.t('dogs.name')         // Returns "Name"
```

**HTML auto-translation:**
```html
<button data-i18n="common.save">Speichern</button>
```

After load, call: `window.i18n.updateElement(element)`

### Database Migrations

Auto-run on startup in `database/database.go`:
- Migrations in `RunMigrations()` function
- Idempotent (safe to run multiple times)
- Creates all 7 tables with indexes

**When modifying schema:**
1. Add migration to `RunMigrations()`
2. Use `IF NOT EXISTS` for safety
3. Test with fresh database (delete gassigeher.db)

## Common Tasks

### Add New API Endpoint

1. **Create/update model** in `internal/models/`
2. **Add repository method** in `internal/repository/` (if DB access needed)
3. **Create handler method** in existing or new handler
4. **Register route** in `cmd/server/main.go`:
   - Public: `router.HandleFunc(...)`
   - Protected: `protected.HandleFunc(...)`
   - Admin: `admin.HandleFunc(...)`
5. **Update API client** in `frontend/js/api.js`
6. **Add translations** in `frontend/i18n/de.json`

### Add New Email Template

1. **Create method** in `internal/services/email_service.go` or `email_account.go`
2. **Use inline HTML template** with styles
3. **Call in handler** with `go emailService.SendX(...)`
4. **Test** by triggering the action

### Add New Admin Page

1. **Create HTML file**: `frontend/admin-<name>.html`
2. **Copy navigation** from any existing admin page (8-item nav)
3. **Add translations** for new features
4. **Add route** to `cmd/server/main.go` under `admin` subrouter if needed
5. **Update navigation** in all 8 admin pages to include new page

## Important Conventions

### Response Helpers

All handlers use:
- `respondJSON(w, statusCode, data)` - Success responses
- `respondError(w, statusCode, message)` - Error responses

Located in `internal/handlers/auth_handler.go` (bottom of file).

### Context Keys

Defined in `internal/middleware/middleware.go`:
```go
const UserIDKey contextKey = "userID"
const EmailKey contextKey = "email"
const IsAdminKey contextKey = "isAdmin"
```

Access in handlers:
```go
userID, _ := r.Context().Value(middleware.UserIDKey).(int)
isAdmin, _ := r.Context().Value(middleware.IsAdminKey).(bool)
```

### Date/Time Formats

**Strict format requirements:**
- Dates: `YYYY-MM-DD` (e.g., "2025-12-01")
- Times: `HH:MM` 24-hour format (e.g., "09:30")
- Timestamps: ISO 8601 / RFC3339

Validated in model `Validate()` methods using `time.Parse()`.

### Handler Initialization Pattern

All handlers follow this pattern:

```go
func NewXHandler(db *sql.DB, cfg *config.Config) *XHandler {
    // Initialize email service if needed
    emailService, err := services.NewEmailService(...)
    if err != nil {
        println("Warning: Failed to initialize email service:", err.Error())
    }

    return &XHandler{
        db: db,
        cfg: cfg,
        xRepo: repository.NewXRepository(db),
        emailService: emailService,
    }
}
```

## Configuration

### Environment Variables

Critical variables in `.env`:
- `JWT_SECRET` - Must be secure random string (256-bit)
- `ADMIN_EMAILS` - Comma-separated list (no DB admin table)
- `DATABASE_PATH` - SQLite file location
- Gmail API credentials (4 variables)

**Admin access**: Users with emails in `ADMIN_EMAILS` automatically get `is_admin: true` in JWT claims.

### System Settings (Configurable at Runtime)

Three settings stored in `system_settings` table:
- `booking_advance_days` (default: 14)
- `cancellation_notice_hours` (default: 12)
- `auto_deactivation_days` (default: 365)

Admins can change via settings page → updates take effect immediately.

## Database Schema Key Points

### Users Table GDPR Fields
- `is_deleted` - Flag for deleted accounts
- `anonymous_id` - Generated on deletion (e.g., "anonymous_user_1234567890")
- `is_active` - For deactivation system
- `last_activity_at` - For auto-deactivation (updated on login, booking)
- `deactivated_at`, `deactivation_reason` - Audit trail

### Unique Constraints
- `users.email` - UNIQUE (but can be NULL after deletion)
- `bookings(dog_id, date, walk_type)` - Prevents double-booking
- `blocked_dates.date` - One block per date

## Frontend Structure

### Page Types

**Public pages**: index.html, register.html, login.html, verify.html, forgot-password.html, reset-password.html, terms.html, privacy.html

**Protected pages**: dogs.html, dashboard.html, profile.html

**Admin pages**: admin-dashboard.html, admin-dogs.html, admin-bookings.html, admin-blocked-dates.html, admin-experience-requests.html, admin-users.html, admin-reactivation-requests.html, admin-settings.html

**Pattern**: All admin pages have identical 9-item navigation header.

### No Build Step

Pure vanilla JavaScript - no webpack, no npm, no bundler.
- Files loaded directly via `<script>` tags
- CSS loaded directly via `<link>` tags
- Changes take effect immediately (refresh browser)

## Special Considerations

### Email Verification on Email Change

When user updates email in profile:
1. New verification token generated
2. `is_verified` set to `false`
3. Verification email sent to **new** email
4. User must verify before email change takes effect

Implementation in `internal/handlers/user_handler.go` → `UpdateMe()`.

### Booking Auto-Completion

Cron job runs hourly, marks bookings as completed where:
```
date < current_date OR (date = current_date AND scheduled_time < current_time)
```

After completion, users can add notes via `PUT /bookings/:id/notes`.

### Experience Level Progression

**Rules enforced in code:**
- Green users can only request Blue (not Orange directly)
- Cannot request already-owned level
- Cannot have duplicate pending requests
- Approval automatically updates user's `experience_level` field

Implementation: `internal/handlers/experience_request_handler.go` → `CreateRequest()`.

### Profile Photo Handling

**Upload Process:**
1. Validate file type (JPEG/PNG only)
2. Save to `UPLOAD_DIR/users/` with original filename
3. Delete old photo if exists
4. Update user's `profile_photo` field
5. Display via `/uploads/<filename>` route (served by nginx in production)

**Storage**: Photos stored in filesystem, paths in database.

## Documentation Files

Read these for context:
- `ImplementationPlan.md` - Complete architecture, all 10 phases
- `API.md` - All 50+ endpoints with examples
- `USER_GUIDE.md` - User features and workflows
- `ADMIN_GUIDE.md` - Admin operations and best practices
- `DEPLOYMENT.md` - Production deployment steps
- `PROJECT_SUMMARY.md` - Executive overview

## Testing Philosophy

Tests are in `*_test.go` files co-located with code.

**Test structure established for:**
- Services: Business logic validation
- Models: Validation method testing
- Repositories: Database operation testing

**To add tests:** Follow existing patterns in `internal/services/auth_service_test.go` and `internal/models/booking_test.go`.

## Key Files to Understand

**Entry point:** `cmd/server/main.go`
- Initializes all handlers
- Registers all routes (50+ endpoints)
- Starts cron service
- Applies middleware chain

**Database setup:** `internal/database/database.go`
- Auto-migration on startup
- Creates 7 tables with indexes

**Auth middleware:** `internal/middleware/middleware.go`
- JWT validation
- Admin checks
- Security headers (XSS, clickjacking protection)

**API client:** `frontend/js/api.js`
- Global `window.api` instance
- All backend endpoints wrapped
- Token management in localStorage

## Development Workflow

1. **Backend changes**: Modify Go files → rebuild → test
2. **Frontend changes**: Edit HTML/JS/CSS → refresh browser (no build needed)
3. **Database changes**: Add migration in `database.go` → restart server
4. **New features**: Follow handler → repository → model → route → API client → UI pattern

## Color Scheme (Tierheim Göppingen)

Defined in `frontend/assets/css/main.css`:
- Primary green: `#82b965`
- Dark background: `#26272b`
- Dark gray: `#33363b`
- System fonts only: Arial, sans-serif (no external fonts)

## German-Only UI

All user-facing text in German via `frontend/i18n/de.json` (300+ translations).

**When adding features:**
1. Add keys to `de.json`
2. Use `data-i18n` attributes in HTML
3. Call `window.i18n.load()` in page scripts

Framework supports other languages (add `en.json` for English), but currently German-only.

## Security Notes

**Admin emails are config-based** (not in database) for security:
- Prevents privilege escalation attacks
- Requires server restart to add/remove admins
- Check: `config.IsAdmin(email)`

**JWT secret must be strong**:
- Generate: `openssl rand -base64 32`
- Change requires all users to re-login

**File uploads validated**:
- Type: JPEG/PNG only
- Size: Max 5MB
- Sanitized filenames
- Stored outside web root in production (served by Go handler)

## Cron Job Integration

Cron service is **always running** when server is up (started in main.go).

**To add new cron job:**
1. Add method to `internal/cron/cron.go`
2. Call via `runPeriodically()` or `runDaily()` in `Start()`
3. Access repositories via `s.bookingRepo`, `s.userRepo`, etc.

**Existing jobs:**
- Auto-complete bookings: Every hour
- Auto-deactivate users: Daily at 3:00 AM

## Email Templates

Located in `internal/services/email_service.go` and `email_account.go`.

**Pattern:**
```go
func (s *EmailService) SendX(to, name string, ...) error {
    subject := "..."
    tmpl := ` ...HTML template with {{.Variables}}... `
    t := template.Must(template.New("name").Parse(tmpl))
    var body bytes.Buffer
    t.Execute(&body, data)
    return s.SendEmail(to, subject, body.String())
}
```

All templates use inline CSS (no external stylesheets in emails).

## Deployment

Complete production deployment package in `deploy/` folder:
- `gassigeher.service` - systemd service file
- `nginx.conf` - Reverse proxy config with SSL
- `backup.sh` - Daily database backup script

See **DEPLOYMENT.md** for step-by-step production deployment guide.

## Repository Organization

```
cmd/server/main.go              # Entry point
internal/
  config/                        # Env var loading
  cron/                         # Automated jobs
  database/                     # Migrations
  handlers/                     # HTTP handlers (12 files)
  middleware/                   # Auth, security, logging
  models/                       # Data structures (10 files)
  repository/                   # Database ops (9 files)
  services/                     # Business logic (auth, email)
frontend/
  assets/css/main.css           # All styles (500+ lines)
  i18n/de.json                  # German translations (300+ strings)
  js/api.js                     # API client wrapper
  js/i18n.js                    # Translation system
  [23 HTML pages]               # Complete UI
deploy/                         # Production configs
[6 documentation files]         # Comprehensive guides
```

## Notes for Future Development

**When adding features:**
- Keep German translations updated
- Add to appropriate admin page if admin feature
- Update ImplementationPlan.md's "Future Enhancements" section
- Consider email notifications
- Update API.md if new endpoints
- Test GDPR implications (data deletion)

**Experience level changes:**
- Frontend: Update locked dog display logic
- Backend: Validation in `CreateBooking` handler
- Don't forget `CanUserAccessDog()` helper

**Email changes:**
- Test with Gmail API (check quota limits)
- All templates use inline CSS
- German language for all emails
- Include unsubscribe info if required by law

**Database schema changes:**
- Add migration in `database.go`
- Update model structs
- Update repository methods (Create, Update, Find*)
- Rebuild and test

This codebase follows clean architecture principles with clear separation of concerns. All 10 phases are complete and the application is production-ready.

---

## Quick Reference

### Most Common Files to Edit

**Adding features:**
1. Model: `internal/models/<name>.go`
2. Repository: `internal/repository/<name>_repository.go`
3. Handler: `internal/handlers/<name>_handler.go`
4. Routes: `cmd/server/main.go`
5. API client: `frontend/js/api.js`
6. Translations: `frontend/i18n/de.json`
7. UI: `frontend/<page>.html`

**Tests:**
- Service tests: `internal/services/*_test.go`
- Model tests: `internal/models/*_test.go`
- Repository tests: `internal/repository/*_test.go`

### Essential Context Files

Before making changes, read:
1. **ImplementationPlan.md** - See which phase the feature belongs to
2. **API.md** - Check existing endpoint patterns
3. **ADMIN_GUIDE.md** - Understand admin workflows (if admin feature)
4. **USER_GUIDE.md** - Understand user workflows (if user feature)

---

## Complete Documentation Index

| Document | Lines | Purpose |
|----------|-------|---------|
| [README.md](README.md) | 500+ | Project overview, setup, quick start |
| [ImplementationPlan.md](ImplementationPlan.md) | 1,500+ | Architecture, all 10 phases, database schema |
| [API.md](API.md) | 600+ | Complete REST API reference |
| [DEPLOYMENT.md](DEPLOYMENT.md) | 400+ | Production deployment guide |
| [USER_GUIDE.md](USER_GUIDE.md) | 350+ | User manual (German) |
| [ADMIN_GUIDE.md](ADMIN_GUIDE.md) | 500+ | Administrator handbook |
| [PROJECT_SUMMARY.md](PROJECT_SUMMARY.md) | 500+ | Executive summary |
| [CLAUDE.md](CLAUDE.md) | 400+ | This file - AI development guide |

**Total**: 6,150+ lines of documentation across 9 files

**Navigation**: See [DOCUMENTATION_INDEX.md](DOCUMENTATION_INDEX.md) for quick access guide

---

**Status**: All 10 phases complete. Production-ready. Fully documented. Ready to deploy. 🚀
