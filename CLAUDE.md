# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Gassigeher is a **complete production-ready** dog walking booking system for animal shelters. Built with Go 1.24+ backend (supports SQLite, MySQL, PostgreSQL) and Vanilla JavaScript frontend. All 10 implementation phases complete.

**Status**: ✅ Production ready | 71 API endpoints | 26 pages | 18 email types | 305+ tests | GDPR-compliant

> **Essential Reading**:
> - [ImplementationPlan.md](docs/ImplementationPlan.md) - Complete architecture, all 10 phases
> - [API.md](docs/API.md) - All 71 endpoints with request/response examples
> - [DEPLOYMENT.md](docs/DEPLOYMENT.md) - Production deployment steps

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

**Current Coverage:** 305+ tests passing across all packages (handlers, models, repository, services, middleware, database)

## Architecture Overview

### Three-Layer Backend Architecture

**1. Handlers** (`internal/handlers/`) - 14 handler files
- HTTP request/response handling
- Input validation
- Context extraction (user_id, is_admin)
- Calls services/repositories
- **Pattern**: Each handler owns its dependencies (repos, services, config)

**2. Repositories** (`internal/repository/`) - 12 repository files
- Direct database operations
- SQL query construction
- No business logic
- **Pattern**: One repository per model, returns models only

**3. Services** (`internal/services/`)
- Business logic (auth, email, holidays, booking times)
- Independent of HTTP layer
- **AuthService**: JWT, password hashing, token generation
- **EmailService**: Multi-provider email (Gmail API, SMTP), HTML templates
- **EmailProvider Interface**: Pluggable email providers (Gmail, SMTP)
- **HolidayService**: German public holiday API integration with caching
- **BookingTimeService**: Configurable time slots, approval workflow

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
- Admins defined in database (`users.is_admin` and `users.is_super_admin` flags)
- `RequireAdmin` middleware checks `is_admin` context value from JWT
- `RequireSuperAdmin` middleware checks `is_super_admin` context value from JWT
- Applied to protected routes via subrouter
- Super Admin manages admin privileges via UI (admin-users.html)

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

### Email Service Architecture

**Multi-Provider Support:**

The application supports two email providers:
1. **Gmail API** (OAuth2) - Default, best deliverability
2. **SMTP** (Username/Password) - Universal, works with any provider

**Provider Interface:**
```go
type EmailProvider interface {
    SendEmail(to, subject, body string) error
    ValidateConfig() error
    Close() error
    GetFromEmail() string
}
```

**Supported SMTP Providers:**
- Strato (smtp.strato.de)
- Office365 (smtp.office365.com)
- Gmail SMTP (smtp.gmail.com)
- Any custom SMTP server

**Initialization Pattern:**

Email service can fail gracefully. Pattern used in handlers:

```go
emailService, err := services.NewEmailService(services.ConfigToEmailConfig(cfg))
if err != nil {
    // Log but don't fail - emails will fail gracefully
    fmt.Printf("Warning: Failed to initialize email service: %v\n", err)
}
```

All email sends are in goroutines and check for nil: `if emailService != nil { go emailService.SendX(...) }`

**Provider Selection:**

Set via `EMAIL_PROVIDER` environment variable:
- `gmail` (default) - Uses Gmail API with OAuth2
- `smtp` - Uses standard SMTP

**BCC Admin Copy:**

Optional `EMAIL_BCC_ADMIN` setting sends a blind copy of all emails to admin for audit trail.

**Configuration Examples:**

Gmail API:
```bash
EMAIL_PROVIDER=gmail
GMAIL_CLIENT_ID=...
GMAIL_CLIENT_SECRET=...
GMAIL_REFRESH_TOKEN=...
GMAIL_FROM_EMAIL=noreply@gassigeher.com
EMAIL_BCC_ADMIN=admin@gassigeher.com  # Optional
```

SMTP (Strato):
```bash
EMAIL_PROVIDER=smtp
SMTP_HOST=smtp.strato.de
SMTP_PORT=465
SMTP_USERNAME=noreply@yourdomain.com
SMTP_PASSWORD=your-password
SMTP_FROM_EMAIL=noreply@yourdomain.com
SMTP_USE_SSL=true
EMAIL_BCC_ADMIN=admin@yourdomain.com  # Optional
```

**See Also:**
- [Email Provider Selection Guide](docs/Email_Provider_Selection_Guide.md)
- [SMTP Setup Guides](docs/SMTP_Setup_Guides.md)

### User Activity Tracking

Critical for auto-deactivation (365-day inactivity default):
- Updated on: login, booking creation, booking cancellation
- Method: `userRepo.UpdateLastActivity(userID)`
- **Must call after any user action that counts as "activity"**

### Booking Time Rules

Configurable time slots managed via `BookingTimeService`:
- **Day types**: weekday, weekend, holiday
- **Time slots**: Morning, afternoon, evening with configurable start/end times
- **Blocked periods**: Feeding times, rest periods
- **Approval workflow**: Certain times may require admin approval
- Admin configures via `admin-settings.html`
- API: `GET/PUT /api/booking-times/rules`, `GET /api/booking-times/available-slots`

### German Holiday Integration

Holiday detection via `HolidayService`:
- Uses feiertage-api.de for German public holidays
- Caches results in `feiertage_cache` table
- Default state: BW (Baden-Württemberg), configurable
- Custom holidays can be added by admin
- Holidays use weekend booking rules by default

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
1. **Auto-complete**: Runs every 15 minutes, marks past bookings as completed
2. **Booking reminders**: Runs every 15 minutes, sends email 1-2 hours before walks
3. **Auto-deactivate**: Runs daily at 3am, deactivates inactive users

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

Auto-run on startup via migration system in `internal/database/`:
- 21 migrations defined in `internal/database/00X_*.go` files
- Each migration has SQL for all three databases (SQLite, MySQL, PostgreSQL)
- Creates 11 tables with indexes
- Schema versioning via `schema_migrations` table
- Idempotent (safe to run multiple times)

**When modifying schema:**
1. Create new migration file in `internal/database/`
2. Add SQL for all three databases
3. Use `IF NOT EXISTS` for safety
4. Test with fresh database (delete gassigeher.db)

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
2. **Copy navigation** from any existing admin page
3. **Add translations** for new features
4. **Add route** to `cmd/server/main.go` under `admin` subrouter if needed
5. **Update navigation** in all admin pages to include new page

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
const IsSuperAdminKey contextKey = "isSuperAdmin"
```

Access in handlers:
```go
userID, _ := r.Context().Value(middleware.UserIDKey).(int)
isAdmin, _ := r.Context().Value(middleware.IsAdminKey).(bool)
isSuperAdmin, _ := r.Context().Value(middleware.IsSuperAdminKey).(bool)
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
- `SUPER_ADMIN_EMAIL` - Email for Super Admin account (created automatically on first run)
- `DATABASE_PATH` - SQLite file location
- Gmail API credentials (4 variables)

**Note**: `ADMIN_EMAILS` is no longer used. Admin privileges are now managed via database.

### Super Admin System

**Super Admin vs Regular Admin:**

- **Super Admin** (ID=1): Can promote/demote other admins, cannot be deleted/deactivated
- **Regular Admin**: All admin functions except user promotion, can be demoted by Super Admin
- **Admin privileges stored in database** (not config file)

**First-time installation:**
- Automatic seed data generation
- Super Admin created automatically
- Credentials in `SUPER_ADMIN_CREDENTIALS.txt` and console
- 3 test users created (green, blue, orange levels)
- 5 test dogs created
- 3 test bookings created

**Change Super Admin password:**
1. Edit `SUPER_ADMIN_CREDENTIALS.txt`
2. Update the `PASSWORD:` line
3. Save file
4. Restart server
5. File updated with confirmation

**Promote user to admin:**
- Login as Super Admin
- Go to admin-users.html
- Click "Zu Admin ernennen" button on user row
- User immediately gains admin access

**Demote admin:**
- Login as Super Admin
- Go to admin-users.html
- Click "Admin entfernen" button on admin row
- User immediately loses admin access

**Authentication changes:**
- JWT includes `is_admin` and `is_super_admin` claims
- Middleware: `RequireAdmin` and `RequireSuperAdmin`
- Config-based `ADMIN_EMAILS` removed
- Admin flags stored in database: `users.is_admin`, `users.is_super_admin`

**Protection:**
- Super Admin cannot be deleted (ID=1 protected)
- Admins excluded from auto-deactivation cron
- Cannot promote/demote Super Admin
- Only Super Admin can manage admin privileges

### System Settings (Configurable at Runtime)

Three settings stored in `system_settings` table:
- `booking_advance_days` (default: 14)
- `cancellation_notice_hours` (default: 12)
- `auto_deactivation_days` (default: 365)

Admins can change via settings page → updates take effect immediately.

## Multi-Database Support

### Overview

The application supports **three database backends** with complete feature parity:
- **SQLite** (default) - Zero-config, perfect for development and small deployments (<1,000 users)
- **MySQL** - Web-scale performance for medium deployments (1,000-50,000 users)
- **PostgreSQL** - Enterprise-grade for large deployments (10,000+ users)

**Key Principle**: All SQL is database-agnostic. Repositories use standard SQL that works identically across all three databases.

### Configuration

Set database type via environment variable:

```bash
# SQLite (default)
DB_TYPE=sqlite
DATABASE_PATH=./gassigeher.db

# MySQL
DB_TYPE=mysql
DB_HOST=localhost
DB_PORT=3306
DB_NAME=gassigeher
DB_USER=gassigeher_user
DB_PASSWORD=secure_password
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5

# PostgreSQL
DB_TYPE=postgres
DB_HOST=localhost
DB_PORT=5432
DB_NAME=gassigeher
DB_USER=gassigeher_user
DB_PASSWORD=secure_password
DB_SSLMODE=require
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
```

See `.env.example` for complete configuration options.

### Architecture

**Dialect System** (`internal/database/dialect*.go`):
- `Dialect` interface defines database-specific SQL syntax
- `SQLiteDialect`, `MySQLDialect`, `PostgreSQLDialect` implementations
- Handles differences in: auto-increment, boolean types, text types, placeholders
- Factory pattern creates correct dialect based on `DB_TYPE`

**Migration System** (`internal/database/migrations.go`):
- Migrations defined in `internal/database/00X_*.go` files
- Each migration has SQL for all three databases
- Schema versioning via `schema_migrations` table
- Idempotent - safe to run multiple times
- Auto-runs on application startup

**Repository Layer** (`internal/repository/*.go`):
- Uses **100% standard SQL** (SELECT, INSERT, UPDATE, DELETE)
- **No database-specific functions** in queries
- Parameterized queries with `?` placeholders (works on all databases)
- Date/time operations use Go's `time.Now()` instead of SQL functions

### Database-Agnostic SQL Patterns

**✅ CORRECT - Standard SQL (works everywhere):**

```go
// Use Go for dates
currentDate := time.Now().Format("2006-01-02")
query := `SELECT * FROM bookings WHERE date >= ? AND status = ?`
db.Query(query, currentDate, "scheduled")

// Standard comparison operators
query := `SELECT * FROM users WHERE is_active = ? AND last_activity_at < ?`
db.Query(query, 1, cutoffTime)

// Standard aggregates
query := `SELECT COUNT(*) FROM bookings WHERE dog_id = ?`
```

**❌ INCORRECT - Database-specific SQL:**

```go
// SQLite-specific (don't use!)
query := `SELECT * FROM bookings WHERE date >= date('now')`

// MySQL-specific (don't use!)
query := `SELECT * FROM bookings WHERE date >= CURDATE()`

// PostgreSQL-specific (don't use!)
query := `SELECT * FROM bookings WHERE date >= CURRENT_DATE`
```

### Testing Across Databases

**Run tests on all databases:**

```bash
# SQLite (default)
go test ./... -v

# MySQL (requires running MySQL server)
DB_TYPE=mysql DB_TEST_MYSQL="user:pass@tcp(localhost:3306)/test_db" go test ./... -v

# PostgreSQL (requires running PostgreSQL server)
DB_TYPE=postgres DB_TEST_POSTGRES="postgres://user:pass@localhost:5432/test_db" go test ./... -v
```

**Docker Compose for testing:**

```bash
# Start test databases
docker-compose -f docker-compose.test.yml up -d

# Run tests against all databases
./scripts/test_all_databases.sh  # Linux/Mac
./scripts/test_all_databases.ps1  # Windows
```

See **[MultiDatabase_Testing_Guide.md](docs/MultiDatabase_Testing_Guide.md)** for comprehensive testing instructions.

### When to Add Database-Specific Code

**You DON'T need dialect-specific code if:**
- ✅ Using standard SELECT, INSERT, UPDATE, DELETE
- ✅ Using standard WHERE, JOIN, GROUP BY, ORDER BY
- ✅ Using standard aggregates (COUNT, SUM, AVG, MIN, MAX)
- ✅ Using Go's `time.Now()` for dates/timestamps
- ✅ Using `?` placeholders for parameters

**You NEED dialect-specific code only for:**
- ❌ CREATE TABLE statements (auto-increment syntax varies)
- ❌ ALTER TABLE statements (IF NOT EXISTS support varies)
- ❌ INSERT OR IGNORE / UPSERT logic (syntax varies)
- ❌ Special database functions (rare, avoid if possible)

**For migrations**, add SQL for each database in the migration file:

```go
// internal/database/001_create_table.go
func init() {
    RegisterMigration(&Migration{
        ID: "001_create_table",
        Up: map[string]string{
            "sqlite": `CREATE TABLE IF NOT EXISTS users (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                name TEXT NOT NULL,
                is_active INTEGER DEFAULT 0
            )`,
            "mysql": `CREATE TABLE IF NOT EXISTS users (
                id INT AUTO_INCREMENT PRIMARY KEY,
                name VARCHAR(255) NOT NULL,
                is_active TINYINT(1) DEFAULT 0
            )`,
            "postgres": `CREATE TABLE IF NOT EXISTS users (
                id SERIAL PRIMARY KEY,
                name VARCHAR(255) NOT NULL,
                is_active BOOLEAN DEFAULT FALSE
            )`,
        },
    })
}
```

### Migration Best Practices

1. **Always add SQL for all three databases** in every migration
2. **Use IF NOT EXISTS** for CREATE TABLE (idempotency)
3. **Test migration on all databases** before committing
4. **Keep schema identical** across databases (same tables, columns, constraints)
5. **Use schema_migrations table** for version tracking (automatic)

### Connection Pooling

**SQLite**: No pooling needed (file-based, single connection optimal)

**MySQL/PostgreSQL**: Connection pooling configured automatically
- `DB_MAX_OPEN_CONNS=25` - Maximum simultaneous connections
- `DB_MAX_IDLE_CONNS=5` - Idle connections to keep in pool
- `DB_CONN_MAX_LIFETIME=5` - Connection lifetime in minutes

### Database Selection Guide

**Choose SQLite if:**
- Development or testing environment
- Small shelter (<1,000 users)
- Single server deployment
- Zero setup time required
- File-based backup preferred

**Choose MySQL if:**
- Medium to large shelter (1,000-50,000 users)
- Proven web-scale performance needed
- Replication/clustering required
- Familiar with MySQL administration
- Widely supported hosting

**Choose PostgreSQL if:**
- Enterprise deployment (10,000+ users)
- Advanced features needed (full-text search, JSON columns)
- Complex analytics queries
- Strong ACID compliance critical
- Multiple concurrent writes

See **[Database_Selection_Guide.md](docs/Database_Selection_Guide.md)** for detailed comparison and migration procedures.

### Related Documentation

- **[DatabasesSupportPlan.md](docs/DatabasesSupportPlan.md)** - Complete implementation plan (2,300+ lines)
- **[MySQL_Setup_Guide.md](docs/MySQL_Setup_Guide.md)** - MySQL installation and configuration
- **[PostgreSQL_Setup_Guide.md](docs/PostgreSQL_Setup_Guide.md)** - PostgreSQL installation and configuration
- **[Database_Selection_Guide.md](docs/Database_Selection_Guide.md)** - Choosing the right database
- **[MultiDatabase_Testing_Guide.md](docs/MultiDatabase_Testing_Guide.md)** - Testing across databases

## Database Schema Key Points

### Tables (11 total)
- `users` - User accounts with first_name, last_name, experience level
- `dogs` - Dog info with is_featured, external_link, photo fields
- `bookings` - Walk bookings with approval workflow
- `blocked_dates` - Admin-blocked dates (global or per-dog)
- `experience_requests` - User level promotion requests
- `reactivation_requests` - Account reactivation requests
- `system_settings` - Configurable system settings
- `booking_time_rules` - Configurable time slots per day type
- `custom_holidays` - Custom and API-fetched holidays
- `feiertage_cache` - German holiday API cache
- `schema_migrations` - Migration version tracking

### Users Table GDPR Fields
- `is_deleted` - Flag for deleted accounts
- `anonymous_id` - Generated on deletion (e.g., "anonymous_user_1234567890")
- `is_active` - For deactivation system
- `last_activity_at` - For auto-deactivation (updated on login, booking)
- `deactivated_at`, `deactivation_reason` - Audit trail

### Unique Constraints
- `users.email` - UNIQUE (but can be NULL after deletion)
- `bookings(dog_id, date, walk_type)` - Prevents double-booking
- `blocked_dates.date` - One block per date (global) or per dog_id

## Frontend Structure

### Page Types

**Public pages**: index.html, register.html, login.html, verify.html, forgot-password.html, reset-password.html, terms.html, privacy.html

**Protected pages**: dogs.html, dashboard.html, profile.html

**Admin pages** (10 pages): admin-dashboard.html, admin-dogs.html, admin-bookings.html, admin-blocked-dates.html, admin-experience-requests.html, admin-users.html, admin-reactivation-requests.html, admin-settings.html, admin-holidays.html, admin-booking-times.html

**Pattern**: All admin pages have unified navigation header.

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

### Dog Photo Handling

**Database Schema:**
- `dogs.photo` - Path to full-size photo (e.g., "dogs/dog_1_full.jpg")
- `dogs.photo_thumbnail` - Path to thumbnail (e.g., "dogs/dog_1_thumb.jpg")
- Both fields nullable (dogs can exist without photos)

**Upload Process (Current - Without Phase 1):**
1. Admin selects photo via admin-dogs.html
2. Client-side validation (JPEG/PNG, max 10MB)
3. Photo preview shown via FileReader API
4. On form submit: Dog created/updated first
5. Then photo uploaded via `POST /api/dogs/:id/photo`
6. Backend saves photo to `uploads/dogs/` directory
7. Database updated with photo path
8. Old photo deleted if exists

**Upload Process (With Phase 1 - Future):**
Same as above, but step 6 includes:
- Automatic resizing to 800x800 max
- JPEG compression (quality 85%)
- Thumbnail generation (300x300)
- Saves both full and thumbnail
- ~85% file size reduction

**Frontend Display Pattern:**

```javascript
// Use helper functions (recommended)
${getDogPhotoHtml(dog, true)}  // Uses thumbnail, lazy loading, category placeholder

// Manual pattern (not recommended)
${dog.photo ? `<img src="/uploads/${dog.photo}" ...>` : 'fallback'}
```

**Helper Functions (frontend/js/dog-photo-helpers.js):**
- `getDogPhotoUrl(dog, useThumbnail, useCategoryPlaceholder)` - Get photo URL
- `getDogPhotoHtml(dog, useThumbnail, className, lazyLoad, categoryPlaceholder, withSkeleton)` - Generate img tag
- `getDogPhotoResponsive(dog, className, lazyLoad)` - Generate picture element for mobile/desktop
- `getCalendarDogCell(dog)` - Calendar grid cell with photo
- `preloadCriticalDogImages(dogs, count)` - Preload first N images

**Placeholder Strategy:**
- Dogs without photos show SVG placeholders
- Category-specific colors: green, blue, orange
- Files: `frontend/assets/images/placeholders/dog-placeholder-{category}.svg`
- Fallback: `dog-placeholder.svg` (generic)

**Upload UI (admin-dogs.html):**
- Drag & drop zone with visual feedback
- File validation before upload
- Preview before upload
- Progress indicator during upload
- Edit mode shows current photo with "Change" and "Remove" buttons
- German error messages

**Performance Optimizations:**
- Lazy loading: `loading="lazy"` attribute (95%+ browser support)
- Responsive images: `<picture>` element (mobile gets thumbnails)
- Skeleton loader: Animated shimmer while loading
- Fade-in: Smooth appearance when loaded
- Preload: First 3 images preloaded for instant display
- Calendar: Uses thumbnails in grid (40x40 circles)

**Best Practices:**
- Always use helper functions for consistency
- Use thumbnails in lists/grids (performance)
- Use full-size in detail views
- Enable lazy loading by default
- Provide meaningful alt text
- Handle NULL photo values gracefully

**Common Patterns:**

```javascript
// Dog card in list
${getDogPhotoHtml(dog, true)}  // Thumbnail, lazy load, skeleton

// Dog detail modal
${getDogPhotoHtml(dog, false, 'dog-detail-image', false)}  // Full size, no lazy load

// Calendar view
${getCalendarDogCell(dog)}  // Pre-formatted cell with thumbnail

// Responsive (mobile/desktop)
${getDogPhotoResponsive(dog)}  // Picture element with media queries
```

**Storage**: Photos in `uploads/dogs/`, paths in database (nullable).

Read these for context:
- [ImplementationPlan.md](docs/ImplementationPlan.md) - Complete architecture, all 10 phases
- [API.md](docs/API.md) - All 50+ endpoints with examples
- [USER_GUIDE.md](docs/USER_GUIDE.md) - User features and workflows
- [ADMIN_GUIDE.md](docs/ADMIN_GUIDE.md) - Admin operations and best practices
- [DEPLOYMENT.md](docs/DEPLOYMENT.md) - Production deployment steps
- [PROJECT_SUMMARY.md](docs/PROJECT_SUMMARY.md) - Executive overview

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
- Registers all routes (71 endpoints)
- Starts cron service
- Applies middleware chain

**Database setup:** `internal/database/database.go`
- Auto-migration on startup (21 migrations)
- Creates 11 tables with indexes

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

Defined in `frontend/assets/css/main.css` (compiled from SCSS):
- Primary green: `#82b965`
- Dark background: `#26272b`
- Dark gray: `#33363b`
- Border radius: `6px`
- System fonts only: Arial, sans-serif (no external fonts)

**Styling**: Uses SCSS/SASS for modular styling, compiled to CSS

## German-Only UI

All user-facing text in German via `frontend/i18n/de.json` (400+ translations).

**When adding features:**
1. Add keys to `de.json`
2. Use `data-i18n` attributes in HTML
3. Call `window.i18n.load()` in page scripts

Framework supports other languages (add `en.json` for English), but currently German-only.

## Security Notes

**Admin privileges are database-driven** for ease of management:
- Super Admin (ID=1) manages all admin privileges via UI
- No server restart needed to add/remove admins
- Protected from privilege escalation (only Super Admin can promote)
- Super Admin cannot be deleted or demoted
- Check: `user.IsAdmin` or `user.IsSuperAdmin` flags

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
- Auto-complete bookings: Every 15 minutes
- Booking reminders: Every 15 minutes (1-2 hours before walk)
- Auto-deactivate users: Daily at 3:00 AM

## Email Templates

Located in `internal/services/email_service.go` and `email_account.go`. **18 email types** total.

**Email types:**
- Authentication: verification, welcome, password reset
- Bookings: confirmation, reminder, cancellation, approval/rejection, moved
- Experience: level approved/denied
- Account: deactivated, reactivated, deletion confirmation
- Auto-deactivation: warning and notification

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

### Production Features

- **Standalone binary**: Frontend embedded via `go:embed`
- **Health check**: `GET /api/health` for monitoring
- **Version info**: `GET /api/version` with build-time injection
- **CLI parameter**: `-env /path/to/custom.env` for config location
- **Configurable BASE_URL**: No hardcoded localhost URLs

## Repository Organization

```
cmd/server/main.go              # Entry point
internal/
  config/                        # Env var loading
  cron/                         # Automated jobs (3 cron tasks)
  database/                     # Migrations (21 migration files)
  handlers/                     # HTTP handlers (14 files)
  logging/                      # Production logging with rotation
  middleware/                   # Auth, security, logging
  models/                       # Data structures
  repository/                   # Database ops (12 files)
  services/                     # Business logic (auth, email, holidays, booking times)
  static/frontend/              # Embedded frontend files
  version/                      # Build version information
frontend/
  assets/css/main.css           # All styles (compiled from SCSS)
  i18n/de.json                  # German translations (400+ strings)
  js/api.js                     # API client wrapper
  js/i18n.js                    # Translation system
  [26 HTML pages]               # Complete UI
deploy/                         # Production configs
docs/                           # 15 documentation files
uploads/                        # User and dog photos
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
1. **[ImplementationPlan.md](docs/ImplementationPlan.md)** - See which phase the feature belongs to
2. **[API.md](docs/API.md)** - Check existing endpoint patterns
3. **[ADMIN_GUIDE.md](docs/ADMIN_GUIDE.md)** - Understand admin workflows (if admin feature)
4. **[USER_GUIDE.md](docs/USER_GUIDE.md)** - Understand user workflows (if user feature)

---

## Complete Documentation Index

| Document | Lines | Purpose |
|----------|-------|---------|
| [README.md](README.md) | 800+ | Project overview, setup, quick start |
| [ImplementationPlan.md](docs/ImplementationPlan.md) | 1,500+ | Architecture, all 10 phases, database schema |
| [API.md](docs/API.md) | 600+ | Complete REST API reference (71 endpoints) |
| [DEPLOYMENT.md](docs/DEPLOYMENT.md) | 600+ | Production deployment guide |
| [USER_GUIDE.md](docs/USER_GUIDE.md) | 350+ | User manual (German) |
| [ADMIN_GUIDE.md](docs/ADMIN_GUIDE.md) | 500+ | Administrator handbook |
| [PROJECT_SUMMARY.md](docs/PROJECT_SUMMARY.md) | 500+ | Executive summary |
| [CLAUDE.md](CLAUDE.md) | 1,000+ | This file - AI development guide |

**Total**: 9,500+ lines of documentation across 15 files

**Navigation**: See [DOCUMENTATION_INDEX.md](docs/DOCUMENTATION_INDEX.md) for quick access guide

---

**Status**: All 10 phases complete. Production-ready. Fully documented. Ready to deploy. 🚀
