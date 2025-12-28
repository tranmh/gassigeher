# Gassigeher - Complete Feature Reference

**Comprehensive documentation of all features in both Simple-Mode and SaaS-Mode.**

> **Quick Links**: [README](../README.md) | [API](API.md) | [Admin Guide](ADMIN_GUIDE.md) | [User Guide](USER_GUIDE.md)

---

## Table of Contents

1. [Deployment Modes](#deployment-modes)
2. [User Roles & Permissions](#user-roles--permissions)
3. [Color-Based Access System](#color-based-access-system)
4. [Booking System](#booking-system)
5. [Central Admin & Impersonation](#central-admin--impersonation-saas)
6. [Marketing & Landing Pages](#marketing--landing-pages-saas)
7. [Demo System](#demo-system-saas)
8. [Storage System (S3 vs Local)](#storage-system-s3-vs-local)
9. [Testing Infrastructure](#testing-infrastructure)
10. [Monitoring & Observability](#monitoring--observability)
11. [Billing & Subscriptions](#billing--subscriptions-saas)
12. [DSGVO/GDPR Compliance](#dsgvogdpr-compliance)
13. [Branding & Theming](#branding--theming-saas)
14. [Feature Flags](#feature-flags)
15. [Audit Logging](#audit-logging)
16. [Cron Jobs & Automation](#cron-jobs--automation)
17. [Data Import/Export](#data-importexport)
18. [Holiday Management](#holiday-management)
19. [Walk Reports](#walk-reports)
20. [External Integrations](#external-integrations)

---

## Deployment Modes

Gassigeher supports two deployment modes:

| Feature | Simple-Mode | SaaS-Mode |
|---------|-------------|-----------|
| Tenants | Single | Unlimited (subdomains) |
| Database | SQLite/MySQL/PostgreSQL | PostgreSQL with RLS |
| Storage | Local filesystem | S3-compatible (Hetzner) |
| Billing | N/A | Stripe integration |
| Central Admin | N/A | Full platform management |
| Theming | Basic | 10 presets + custom colors |
| Demo Tenant | N/A | Auto-reset demo |

### Simple-Mode
- Ideal for individual animal shelters
- Single-tenant deployment
- All features except billing and central admin
- Supports SQLite, MySQL, or PostgreSQL

### SaaS-Mode
- Multi-tenant platform via subdomains (e.g., `shelter1.gassigeher.de`)
- PostgreSQL with Row-Level Security for data isolation
- Full billing integration with Stripe
- Central admin dashboard for platform management
- Per-tenant customization (branding, themes, settings)

---

## User Roles & Permissions

### Regular User
- Browse and filter dogs
- Book walks (based on assigned colors)
- Manage profile and photo
- Request additional colors
- Submit walk reports
- Delete account (GDPR)

### Admin (Tenant Admin)
- All user permissions
- Manage dogs (CRUD, photos, availability)
- Manage bookings (view all, cancel, move, approve)
- Approve/deny color requests
- Manage users (activate/deactivate, assign colors)
- Configure booking time rules and holidays
- View audit logs
- System settings

### Super Admin
- All admin permissions
- Promote/demote admins
- **Impersonate users** within own tenant
- Cannot be deleted or demoted
- One per tenant (ID=1)

### Central Admin (SaaS only)
- Platform-wide management
- View all tenants and statistics
- **Impersonate any user** across all tenants
- Manage billing and subscriptions
- Create/manage promo codes
- Marketing campaign management
- Feature flag management

---

## Color-Based Access System

Colors control which dogs users can book. This system is fully customizable per tenant.

### How It Works
1. **Color Categories**: Admins define colors (e.g., "Grün", "Blau", "Orange")
2. **Dog Assignment**: Each dog is assigned one required color
3. **User Colors**: Users can have multiple colors assigned
4. **Access Control**: Users can only book dogs matching their colors
5. **Color Requests**: Users can request additional colors

### Default Setup
Most tenants use three colors:
- 🟢 **Grün** (Green): Default for new users, beginner-friendly dogs
- 🔵 **Blau** (Blue): Intermediate, requires admin approval
- 🟠 **Orange**: Advanced, requires admin approval

### Color Request Workflow
1. User requests a new color from their profile
2. Request appears in admin dashboard
3. Admin reviews user's walk history
4. Admin approves or denies with optional message
5. User receives email notification
6. If approved, color is automatically added

### API Endpoints
```
GET  /api/colors                    # List all colors
POST /api/color-requests            # Request new color
GET  /api/color-requests            # List requests (user: own, admin: all)
PUT  /api/color-requests/:id/approve  # Approve request
PUT  /api/color-requests/:id/deny     # Deny request
```

---

## Booking System

### Booking Time Rules
Admins can configure when bookings are allowed:

- **Weekday Rules**: Define morning, afternoon, evening slots
- **Weekend/Holiday Rules**: Different times for weekends and holidays
- **Slot Types**:
  - **Allowed**: Bookings permitted
  - **Blocked**: No bookings (e.g., feeding times)
  - **Approval Required**: Admin must approve booking

### Booking Workflow
1. User selects dog and date
2. System shows available time slots
3. User selects slot and confirms
4. If approval required: booking goes to pending
5. Admin approves/rejects with optional message
6. User receives confirmation email

### Booking Reminders
- Automated email reminders 1-2 hours before walk
- Cron job runs every 15 minutes
- Configurable reminder window

### API Endpoints
```
POST /api/bookings                  # Create booking
GET  /api/bookings                  # List bookings
PUT  /api/bookings/:id/cancel       # Cancel booking
PUT  /api/bookings/:id/approve      # Approve booking (admin)
PUT  /api/bookings/:id/reject       # Reject booking (admin)
PUT  /api/bookings/:id/move         # Move booking (admin)
```

---

## Central Admin & Impersonation (SaaS)

### Central Admin Dashboard
Platform-wide management interface at `admin.gassigeher.de`:

- **Platform Statistics**: Total tenants, users, dogs, bookings
- **Tenant Management**: View, activate, deactivate tenants
- **User Overview**: See all users across tenants
- **Billing Management**: Subscriptions, invoices, promo codes
- **Marketing Campaigns**: FOMO, referrals, campaigns

### Impersonation Feature

Both Super Admins and Central Admins can impersonate users for support and debugging.

**Super Admin Impersonation** (within own tenant):
```
POST /api/admin/users/:id/impersonate
```
- Can impersonate any non-admin user in their tenant
- Session marked with `impersonating=true`
- Original user ID preserved for audit

**Central Admin Impersonation** (any tenant):
```
POST /api/central-admin/impersonate/:userId
```
- Can impersonate any user across all tenants
- Full audit trail of impersonation actions
- Special JWT claims: `is_central_admin`, `original_user_id`

**Ending Impersonation**:
```
POST /api/auth/end-impersonation
```
- Returns to original admin session
- Logged in audit trail

### API Endpoints
```
GET  /api/central-admin/stats       # Platform statistics
GET  /api/central-admin/tenants     # List all tenants
GET  /api/central-admin/users       # List all users
POST /api/central-admin/impersonate/:userId  # Impersonate user
```

---

## Marketing & Landing Pages (SaaS)

### Landing Pages
Marketing website at root domain:

| Path | Description |
|------|-------------|
| `/` | Hero section with features |
| `/about` | About the product |
| `/pricing` | Free vs Pro comparison |
| `/faq` | Frequently asked questions |
| `/contact` | Contact form |
| `/demo` | Demo tenant information |
| `/register` | Tenant registration |
| `/imprint` | Legal imprint (Impressum) |
| `/privacy` | Privacy policy |
| `/agb` | Terms of service |
| `/sla` | Service level agreement |
| `/widerrufsbelehrung` | Right of withdrawal |

### Marketing Campaigns

**FOMO Countdown Campaigns**:
- Show limited availability (e.g., "Only 5 free Pro plans left!")
- Creates urgency for sign-ups
- Configurable via Central Admin

**Referral System**:
```
POST /api/central-admin/marketing/referrals
GET  /api/central-admin/marketing/referrals
```
- Users can share referral codes
- Discounts for both referrer and referee
- Maximum 24 free months per referral
- Tracks referral usage and conversions

### Contact Form
```
POST /api/contact
```
- Email validation with header injection prevention
- Auto-sends to configured support email
- Rate limited to prevent spam

---

## Demo System (SaaS)

### Demo Tenant
A special tenant for prospects to try the system:

- **Subdomain**: `demo.gassigeher.de`
- **Auto-created** on system startup
- **Pre-seeded** with users, dogs, and bookings
- **Daily reset** at midnight (configurable)

### Demo Credentials API
```
GET /api/demo/credentials   # Returns demo login info
GET /api/demo/status        # Returns demo tenant status
```

### Configuration
```env
DEMO_ENABLED=true
DEMO_RESET_INTERVAL_HOURS=24
```

### Demo Reset Process
1. Cron job runs at configured interval
2. All demo data is deleted
3. Fresh seed data is created
4. New admin password is generated
5. Credentials stored in demo tenant state

---

## Storage System (S3 vs Local)

### Local Filesystem (Default)
- Used when `USE_S3=false`
- Files stored in `UPLOAD_DIR` (default: `./uploads`)
- Suitable for Simple-Mode and development

### S3-Compatible Storage
- Used when `USE_S3=true`
- Supports any S3-compatible storage (AWS, Hetzner, MinIO)
- Tenant-organized file paths for isolation
- Required for production SaaS deployments

### Configuration
```env
# Local storage
USE_S3=false
UPLOAD_DIR=./uploads

# S3 storage
USE_S3=true
S3_ENDPOINT=fsn1.your-objectstorage.com
S3_REGION=fsn1
S3_BUCKET=gassigeher-uploads
S3_ACCESS_KEY=your-access-key
S3_SECRET_KEY=your-secret-key
```

### Features
- **Path Traversal Prevention**: Strict validation prevents directory attacks
- **Timeout Handling**: 30-second default for S3 operations
- **Thumbnail Generation**: Automatic thumbnails for images
- **Tenant Isolation**: SaaS files organized by tenant ID

---

## Testing Infrastructure

### Backend Testing (Go)
```bash
go test ./...                       # All tests
go test ./internal/services/... -v  # Service tests
go test -coverprofile=coverage.out  # With coverage
go tool cover -html=coverage.out    # View coverage report
```

### Frontend Testing (Jest)
```bash
npm test                # Run Jest unit tests
npm run test:watch      # Watch mode
npm run test:coverage   # With coverage report
```

### E2E Testing (Playwright)
Located in `/e2e-tests/`:
```bash
cd e2e-tests
npm install
npm run test            # Headless tests
npm run test:headed     # With visible browser
npm run test:debug      # Debug mode
npm run test:ui         # Interactive UI mode
```

**Page Object Models**:
- `LandingRegisterPage` - Tenant registration
- `AdminDogsPage` - Dog management
- `BillingPage` - Subscription management

### Test Data Seeding
```bash
./test-overview.sh      # Comprehensive test suite with reporting
```

The seeding system creates:
- Test users with various roles
- Test dogs with different colors
- Sample bookings in various states
- Demo tenant with reset capability

### Code Coverage Goals
- **Target**: 90% code coverage
- **CI Integration**: Coverage reports in GitHub Actions
- **TDD Approach**: Tests written alongside features

---

## Monitoring & Observability

### Prometheus Metrics
```
GET /metrics            # Prometheus text format
GET /api/metrics        # JSON format
```

**Tracked Metrics**:
- HTTP request count by endpoint
- Request latency histograms
- Error rates by status code
- Active connections

### Sentry Error Tracking
```env
SENTRY_DSN=https://your-sentry-dsn
```

**Features**:
- Automatic exception capture
- Custom message logging
- Environment and release tracking
- User context attachment

### Health Check
```
GET /health             # Returns {"status": "ok"}
```

### Logging
- Structured JSON logging
- Request ID for tracing
- Configurable log levels
- Audit trail for compliance

---

## Billing & Subscriptions (SaaS)

### Subscription Tiers

| Feature | Free | Pro |
|---------|------|-----|
| Price | €0/month | €29/month or €290/year |
| Dogs | **10 maximum** | Unlimited |
| Users | Unlimited | Unlimited |
| Storage | 100MB | 10GB |
| Support | Community | Priority |
| Branding | Basic | Full customization |

### Stripe Integration
```env
STRIPE_SECRET_KEY=sk_live_xxx
STRIPE_WEBHOOK_SECRET=whsec_xxx
STRIPE_PRICE_MONTHLY=price_xxx
STRIPE_PRICE_YEARLY=price_xxx
```

### API Endpoints
```
POST /api/billing/checkout          # Create checkout session
POST /api/billing/portal            # Customer billing portal
GET  /api/billing/subscription      # Current subscription
POST /api/billing/webhook           # Stripe webhooks
```

### Promo Codes
```
POST /api/central-admin/promo-codes
GET  /api/central-admin/promo-codes
DELETE /api/central-admin/promo-codes/:id
```

**Discount Types**:
- **Percentage**: 0-100% off
- **Fixed Amount**: Specific amount in cents
- **Free Months**: 1-24 months free

**Promo Code Features**:
- Per-plan validity (Free only, Pro only, or all)
- Maximum uses limit
- Expiration dates
- Stripe coupon integration

### Free Months Tracking
Sources of free months:
- `promo` - From promo codes
- `referral` - From referral program
- `admin_grant` - Manual admin grant
- `trial` - Initial trial period

### Invoices
```
GET /api/billing/invoices           # List invoices
GET /api/billing/invoices/:id/pdf   # Download PDF
```

**Invoice Statuses**: draft, open, paid, void, uncollectible

---

## DSGVO/GDPR Compliance

### User Consent Management
```
GET  /api/users/me/consent          # Current consent status
GET  /api/users/me/consent/history  # Consent history
POST /api/users/me/consent          # Record consent
```

**Tracked Information**:
- Consent timestamp
- IP address
- User agent
- Consent type (terms, privacy, marketing)

### Account Deletion (Right to Erasure)
When a user deletes their account:

**Permanently Deleted**:
- Email address
- Phone number
- First name, last name
- Password hash
- Profile photo
- Verification tokens

**Anonymized (Retained)**:
- Walk history (as "Deleted User")
- Walk notes (for dog care)
- Booking records

**Legal Confirmation**:
- Email sent as legal proof
- Timestamp recorded
- Audit log entry

### Data Export
Users can request their data export:
```
GET /api/users/me/export            # Download personal data
```

### Audit Trail
All data access and modifications are logged for compliance:
- Who accessed what data
- When the access occurred
- What changes were made
- IP address and user agent

---

## Branding & Theming (SaaS)

### Theme Presets
10 built-in themes available:

| Preset | Primary Color | Description |
|--------|---------------|-------------|
| `classic` | #82b965 | Default green |
| `ocean` | #3498db | Blue ocean |
| `forest` | #27ae60 | Deep forest green |
| `sunset` | #e74c3c | Warm sunset |
| `lavender` | #9b59b6 | Purple lavender |
| `coral` | #e91e63 | Coral pink |
| `midnight` | #2c3e50 | Dark midnight |
| `emerald` | #1abc9c | Teal emerald |
| `rose` | #ff6b6b | Soft rose |
| `slate` | #607d8b | Gray slate |

### Custom Branding
```
PUT /api/admin/branding
```

**Customizable Elements**:
- Logo (header and favicon)
- Primary, secondary, accent colors
- Background and text colors
- Shelter name and contact info

### Dynamic CSS
```
GET /api/theme/css                  # Returns CSS variables
GET /api/theme/presets              # List all presets
```

CSS variables are generated dynamically based on tenant settings.

---

## Feature Flags

Enable/disable features per tenant or platform-wide.

### Available Flags
| Flag | Description |
|------|-------------|
| `new_booking_ui` | New booking interface |
| `advanced_search` | Advanced dog search |
| `bulk_operations` | Bulk admin operations |
| `experimental_api` | Experimental endpoints |
| `dark_mode` | Dark mode support |
| `mobile_app_integration` | Mobile app APIs |
| `calendar_sync` | Calendar integration |
| `sms_notifications` | SMS notifications |

### API Endpoints
```
GET  /api/admin/feature-flags       # List all flags
PUT  /api/admin/feature-flags/:name # Toggle flag
GET  /api/feature-flags             # Public flag status
```

### Usage
```javascript
if (await api.isFeatureEnabled('dark_mode')) {
  enableDarkMode();
}
```

---

## Audit Logging

Comprehensive audit trail for compliance and debugging.

### Tracked Actions (30+)

**Booking Actions**:
- `booking.created`, `booking.cancelled`, `booking.approved`
- `booking.rejected`, `booking.moved`, `booking.completed`

**User Actions**:
- `user.created`, `user.updated`, `user.deleted`
- `user.promoted`, `user.demoted`
- `user.activated`, `user.deactivated`
- `user.login`, `user.logout`, `user.impersonated`

**Dog Actions**:
- `dog.created`, `dog.updated`, `dog.deleted`

**Request Actions**:
- `color_request.requested`, `color_request.approved`, `color_request.denied`
- `experience_request.requested`, `experience_request.approved`, `experience_request.denied`

**System Actions**:
- `settings.changed`, `theme.changed`
- `data.exported`, `data.imported`
- `tenant.created`, `tenant.updated`, `tenant.activated`, `tenant.deactivated`

### API Endpoints
```
GET /api/admin/audit-logs           # Paginated logs with filters
GET /api/admin/audit-logs/actions   # Available action types
```

### Filters
- `action` - Filter by action type
- `entity_type` - Filter by entity (user, dog, booking)
- `entity_id` - Filter by specific entity
- `user_id` - Filter by acting user
- `date_from`, `date_to` - Date range

---

## Cron Jobs & Automation

### Scheduled Jobs

| Job | Schedule | Description |
|-----|----------|-------------|
| Auto-Complete Bookings | Every 15 min | Mark past bookings as completed |
| Booking Reminders | Every 15 min | Send reminder emails |
| Auto-Deactivate Users | Daily 3 AM | Deactivate inactive users (90 days) |
| Database Backup | Daily 2 AM | Create compressed backup |
| Demo Reset | Daily midnight | Reset demo tenant data |
| Tenant Activity Check | Daily 4 AM | Flag inactive tenants (30 days) |

### Configuration
```env
AUTO_DEACTIVATE_DAYS=90
DEMO_RESET_INTERVAL_HOURS=24
BACKUP_RETENTION_DAYS=30
```

### Manual Triggers (Admin)
Some jobs can be triggered manually:
```
POST /api/admin/cron/complete-bookings
POST /api/admin/cron/send-reminders
```

---

## Data Import/Export

### Dog Import (CSV)
```
POST /api/admin/import/dogs
```

**Features**:
- CSV file upload (max 10MB)
- Column mapping and auto-detection
- Preview before import
- Batch creation with validation

**Required Columns**:
- `name` - Dog name
- `breed` - Breed
- `size` - small/medium/large
- `color` - Color category name

### Data Export
```
GET /api/admin/export/dogs          # Export all dogs
GET /api/admin/export/users         # Export all users
GET /api/admin/export/bookings      # Export bookings
GET /api/users/me/export            # User self-export (GDPR)
```

---

## Holiday Management

### German Public Holidays
Automatic integration with `feiertage-api.de`:

```
GET /api/holidays                   # List holidays
POST /api/admin/holidays/sync       # Sync from API
```

**Features**:
- Federal state awareness (e.g., Baden-Württemberg, Bayern)
- Automatic holiday detection
- Holiday-specific booking rules

### Custom Holidays
```
POST /api/admin/holidays            # Create custom holiday
PUT  /api/admin/holidays/:id        # Update holiday
DELETE /api/admin/holidays/:id      # Delete holiday
```

**Custom Holiday Fields**:
- Name
- Date
- Recurring (yearly)
- Blocked (no bookings)

---

## Walk Reports

Users can submit reports after completing walks.

### Report Fields
- **Behavior Rating**: 1-5 stars
- **Energy Level**: low, medium, high
- **Notes**: Free text observations
- **Photos**: Optional walk photos

### API Endpoints
```
POST /api/walk-reports              # Submit report
GET  /api/walk-reports              # List reports
GET  /api/walk-reports/:id          # Get report details
GET  /api/dogs/:id/walk-reports     # Reports for specific dog
GET  /api/walk-reports/stats        # Aggregate statistics
```

### Photo Support
- Multiple photos per report
- Automatic thumbnail generation
- S3 or local storage
- Max file size: 5MB per photo

### Statistics
```json
{
  "total_reports": 150,
  "average_behavior_rating": 4.2,
  "average_energy_level": "medium",
  "total_photos": 89
}
```

---

## External Integrations

### TEO Dog Management
Integration with external shelter management systems:

```
POST /api/admin/dogs/sync           # Sync from external system
GET  /api/admin/dogs/external/:id   # Get external dog info
```

**Sync Features**:
- Import dogs from external systems
- Update existing dogs
- External link to shelter website
- External ID mapping

### WhatsApp Community
```
PUT /api/admin/settings/whatsapp
```
- Configure WhatsApp group link
- Include in welcome emails
- Easy community onboarding

### Calendar Integration (Feature Flag)
When `calendar_sync` is enabled:
```
GET /api/users/me/calendar          # iCal feed URL
```
- Subscribable calendar feed
- Automatic booking sync
- Reminder integration

---

## Security Features

### Rate Limiting (3 Layers)

1. **Global IP Limiting**: Max requests per IP per minute
2. **Auth Brute Force Protection**: Exponential backoff on failed logins
3. **Per-Tenant Limiting** (SaaS): Tenant-specific rate limits

### Security Headers
```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Content-Security-Policy: default-src 'self'
Strict-Transport-Security: max-age=31536000
```

### Password Requirements
- Minimum 8 characters
- At least one uppercase letter
- At least one lowercase letter
- At least one number
- bcrypt hashing (cost 12)

### JWT Authentication
- 24-hour token expiration
- Refresh token support
- Tenant ID in claims (SaaS)
- Impersonation flags

---

## Configuration Reference

### Essential Environment Variables

```env
# Mode Selection
BASE_DOMAIN=                        # Empty = Simple-Mode, Set = SaaS-Mode

# Database
DB_TYPE=sqlite                      # sqlite, mysql, postgres
DATABASE_PATH=./gassigeher.db       # For SQLite
DATABASE_URL=                       # For MySQL/PostgreSQL

# Email
EMAIL_PROVIDER=smtp                 # gmail or smtp
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=
SMTP_PASSWORD=

# Storage
USE_S3=false
S3_ENDPOINT=
S3_BUCKET=
S3_ACCESS_KEY=
S3_SECRET_KEY=

# SaaS Features
STRIPE_SECRET_KEY=
STRIPE_WEBHOOK_SECRET=
DEMO_ENABLED=true
DEMO_RESET_INTERVAL_HOURS=24

# Monitoring
SENTRY_DSN=
```

---

## Summary

Gassigeher is a comprehensive dog walking booking system with:

- **60+ features** in Simple-Mode
- **80+ features** in SaaS-Mode
- **85+ API endpoints**
- **46+ frontend pages**
- **21 database repositories**
- **30+ tracked audit actions**
- **18+ email notification types**
- **10 theme presets**

For technical implementation details, see [CLAUDE.md](../CLAUDE.md).
For API reference, see [API.md](API.md).
