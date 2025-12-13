# Gassigeher - Dog Walking Booking System

**Status**: **100% COMPLETE** | **PRODUCTION READY** | **READY TO DEPLOY**

A complete, production-ready web-based dog walking booking system built with Go and Vanilla JavaScript.

**Implementation**: All 10 phases complete | 71 API endpoints | 26 pages | 18 email types | GDPR-compliant

---

## Quick Start

```bash
# 1. Clone and setup
git clone <repository-url>
cd gassigeher
cp .env.example .env

# 2. Configure .env (set SUPER_ADMIN_EMAIL and optionally email provider)
nano .env

# 3. Build and run
./bat.sh        # Linux/Mac
# or
bat.bat         # Windows

# 4. Visit http://localhost:8080
```

For production deployment, see **[DEPLOYMENT.md](docs/DEPLOYMENT.md)**.

---

## Features

### User Features
- User registration with email verification and welcome email
- **Registration password requirement** - Shelter-provided code to prevent unauthorized registrations
- JWT-based authentication with secure password requirements
- Self-service password reset and change
- Profile management with photo upload (first name, last name, phone, email)
- Email re-verification on email change
- Experience level system (Green → Orange → Blue)
- Dog browsing with filters and search
- **Featured dogs** on homepage with random selection
- **External dog links** to shelter website for more information
- Booking system with date/time selection
- **Configurable booking time slots** for weekdays and weekends
- **German holiday integration** - Automatic recognition of public holidays
- View and manage bookings (upcoming and past)
- Add notes to completed walks
- Cancel bookings with notice period
- **Booking reminders** - Email notifications 1-2 hours before walks
- Request experience level promotions
- GDPR-compliant account deletion
- **WhatsApp group integration** - Easy onboarding to shelter's WhatsApp community
- German UI with mobile-first responsive design

### Admin Features
- Comprehensive admin dashboard with real-time statistics
- Dog management (CRUD, photos, featured status, external links)
- **Dog-specific date blocking** - Block dates for individual dogs
- Booking management (view all, cancel, move)
- **Booking approval workflow** - Approve/deny bookings for certain time slots
- **Booking time rules** - Configure allowed/blocked time slots per day type
- **Custom holiday management** - Add custom holidays
- Block dates with reasons (global or per-dog)
- User management (activate/deactivate accounts, edit names)
- Experience level request approval workflow
- Reactivation request management
- **Configurable site logo** - Change site branding via settings
- **Registration password management** - Control who can register
- **WhatsApp group settings** - Enable/configure community link
- System settings configuration
- Recent activity feed
- Unified admin navigation

### System Features
- **Multi-provider email support** - Gmail API or SMTP (Strato, Office365, etc.)
- **BCC admin copy** - Audit trail for all emails
- Automatic walk completion via cron jobs (every 15 minutes)
- **Automatic booking reminders** via cron jobs
- Automatic user deactivation after configurable inactivity period
- Email notifications for all major actions (18 types)
- **German public holiday API** (feiertage-api.de) with caching
- Experience-based access control
- Double-booking prevention
- Booking validation rules
- Security headers and XSS protection
- **Standalone binary deployment** - Frontend embedded in executable
- **Version info display** in footer with build-time injection
- **Health check endpoint** for monitoring
- **CLI parameter for .env path** - Custom config file location
- **Configurable BASE_URL** - No hardcoded localhost URLs
- Comprehensive test suite (305+ tests)

## Tech Stack

**Backend:**
- Go 1.24+
- Multi-database support (SQLite, MySQL, PostgreSQL)
- gorilla/mux router
- JWT authentication
- bcrypt password hashing
- Multi-provider email (Gmail API, SMTP)
- Embedded frontend (go:embed)

**Frontend:**
- Vanilla JavaScript (ES6+)
- HTML5 & CSS3
- **SCSS/SASS** for modular styling
- Custom i18n system
- No external dependencies

## Project Structure

```
gassigeher/
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
├── internal/
│   ├── config/               # Configuration management
│   ├── cron/                 # Scheduled jobs (auto-complete, reminders, deactivation)
│   ├── database/             # Database setup and migrations (21 migrations)
│   ├── handlers/             # HTTP request handlers (14 handlers)
│   ├── logging/              # Production logging with rotation
│   ├── middleware/           # Auth, logging, CORS middleware
│   ├── models/               # Data models
│   ├── repository/           # Database operations (12 repositories)
│   ├── services/             # Business logic (auth, email, holidays, booking times)
│   ├── static/               # Embedded frontend files
│   │   └── frontend/         # HTML, JS, CSS, i18n
│   └── version/              # Build version information
├── docs/                     # Documentation files
├── deploy/                   # Production deployment configs
├── uploads/                  # User and dog photos
├── .env                      # Environment variables
├── .env.example              # Environment template
├── go.mod                    # Go dependencies
└── README.md                 # This file
```

## Setup

### 1. Prerequisites

- Go 1.24 or higher
- SQLite3 (default), MySQL, or PostgreSQL

### 2. Clone and Install

```bash
cd gassigeher
go mod download
```

### 3. Configure Environment

Copy `.env.example` to `.env` and configure:

```bash
cp .env.example .env
```

Edit `.env` and set your configuration, especially:
- `JWT_SECRET`: Generate a secure random string
- `SUPER_ADMIN_EMAIL`: Your Super Admin email address (for first-time setup)
- `BASE_URL`: Your production URL (e.g., "https://gassigeher.yourdomain.com")
- Email provider credentials (see Email Configuration below)

**Important:** On first run, the system will automatically:
- Create a Super Admin account with the email from `SUPER_ADMIN_EMAIL`
- Generate a secure random password
- Display credentials in console and save to `SUPER_ADMIN_CREDENTIALS.txt`
- Generate a unique registration password (shown in admin settings)
- Create sample data (3 users, 5 dogs, 3 bookings)

### 4. Email Configuration

The application supports two email providers:

**Option A: Gmail API (Recommended)**
```bash
EMAIL_PROVIDER=gmail
GMAIL_CLIENT_ID=your-client-id
GMAIL_CLIENT_SECRET=your-client-secret
GMAIL_REFRESH_TOKEN=your-refresh-token
GMAIL_FROM_EMAIL=noreply@yourdomain.com
```

**Option B: SMTP (Any Provider)**
```bash
EMAIL_PROVIDER=smtp
SMTP_HOST=smtp.strato.de       # or smtp.office365.com, etc.
SMTP_PORT=465                   # 465 for SSL, 587 for TLS
SMTP_USERNAME=your-email@domain.com
SMTP_PASSWORD=your-password
SMTP_FROM_EMAIL=your-email@domain.com
SMTP_USE_SSL=true              # or SMTP_USE_TLS=true for port 587
```

**Optional: BCC Admin Copy**
```bash
EMAIL_BCC_ADMIN=admin@yourdomain.com  # Receive copies of all emails
```

**Note:** For development, you can skip email setup. The app will run but emails won't be sent.

### 5. Build and Test

**Windows:**
```cmd
bat.bat
```

**Linux/Mac:**
```bash
chmod +x bat.sh
./bat.sh
```

This will:
- Check Go installation
- Download dependencies
- Build the application
- Run all tests

### 6. Run the Application

**Development mode:**
```bash
go run cmd/server/main.go
```

**Using compiled binary:**

Windows:
```cmd
gassigeher.exe
```

Linux/Mac:
```bash
./gassigeher
```

**Custom .env file:**
```bash
./gassigeher -env /path/to/custom.env
```

The server will start on `http://localhost:8080`

### 7. Custom Port

```bash
# Windows
set PORT=3000 && gassigeher.exe

# Linux/Mac
PORT=3000 ./gassigeher
```

## API Endpoints

### Authentication (Public)
- `POST /api/auth/register` - Register new user (requires registration password)
- `POST /api/auth/verify-email` - Verify email with token
- `POST /api/auth/login` - Login and get JWT token
- `POST /api/auth/forgot-password` - Request password reset
- `POST /api/auth/reset-password` - Reset password with token

### Authentication (Protected)
- `PUT /api/auth/change-password` - Change password

### Users (Protected)
- `GET /api/users/me` - Get current user profile
- `PUT /api/users/me` - Update profile (email, phone)
- `POST /api/users/me/photo` - Upload profile photo
- `DELETE /api/users/me` - Delete account (GDPR anonymization)

### Dogs (Protected - Read)
- `GET /api/dogs` - List all dogs with filters (breed, size, age, category, availability, search)
- `GET /api/dogs/:id` - Get dog details
- `GET /api/dogs/breeds` - Get all dog breeds
- `GET /api/dogs/featured` - Get featured dogs for homepage

### Dogs (Admin Only)
- `POST /api/dogs` - Create new dog
- `PUT /api/dogs/:id` - Update dog
- `DELETE /api/dogs/:id` - Delete dog (cancels future bookings)
- `POST /api/dogs/:id/photo` - Upload dog photo
- `PUT /api/dogs/:id/availability` - Toggle dog availability (health status)

### Bookings (Protected)
- `GET /api/bookings` - List bookings (user sees own, admin sees all)
- `GET /api/bookings/:id` - Get booking details
- `POST /api/bookings` - Create booking
- `PUT /api/bookings/:id/cancel` - Cancel booking
- `PUT /api/bookings/:id/notes` - Add notes to completed booking
- `GET /api/bookings/calendar/:year/:month` - Get calendar data

### Bookings (Admin Only)
- `PUT /api/bookings/:id/move` - Move booking to new date/time
- `GET /api/bookings/pending-approval` - List bookings awaiting approval
- `PUT /api/bookings/:id/approve` - Approve booking
- `PUT /api/bookings/:id/reject` - Reject booking

### Booking Time Rules (Admin Only)
- `GET /api/booking-times/rules` - Get all time rules
- `PUT /api/booking-times/rules` - Update time rules
- `GET /api/booking-times/available-slots` - Get available time slots for a date

### Holidays (Admin Only)
- `GET /api/holidays` - List all holidays
- `POST /api/holidays` - Add custom holiday
- `DELETE /api/holidays/:id` - Delete holiday
- `POST /api/holidays/fetch` - Fetch holidays from API

### Blocked Dates (Protected - Read)
- `GET /api/blocked-dates` - List all blocked dates

### Blocked Dates (Admin Only)
- `POST /api/blocked-dates` - Block a date (globally or per-dog)
- `DELETE /api/blocked-dates/:id` - Unblock a date

### Experience Requests (Protected)
- `POST /api/experience-requests` - Request level promotion
- `GET /api/experience-requests` - List requests (user sees own, admin sees all pending)

### Experience Requests (Admin Only)
- `PUT /api/experience-requests/:id/approve` - Approve request
- `PUT /api/experience-requests/:id/deny` - Deny request

### Reactivation Requests (Public)
- `POST /api/reactivation-requests` - Request account reactivation

### Reactivation Requests (Admin Only)
- `GET /api/reactivation-requests` - List all pending requests
- `PUT /api/reactivation-requests/:id/approve` - Approve and reactivate user
- `PUT /api/reactivation-requests/:id/deny` - Deny request

### User Management (Admin Only)
- `GET /api/users` - List all users with filters (active/inactive)
- `GET /api/users/:id` - Get user by ID
- `PUT /api/users/:id` - Update user (name, email, phone)
- `PUT /api/users/:id/activate` - Activate user account
- `PUT /api/users/:id/deactivate` - Deactivate user account
- `PUT /api/users/:id/promote` - Promote user to admin
- `PUT /api/users/:id/demote` - Demote admin to user

### System Settings (Admin Only)
- `GET /api/settings` - Get all settings
- `PUT /api/settings/:key` - Update setting value
- `GET /api/settings/registration-password` - Get registration password

### Admin Dashboard (Admin Only)
- `GET /api/admin/stats` - Get dashboard statistics
- `GET /api/admin/activity` - Get recent activity feed

### System (Public)
- `GET /api/health` - Health check endpoint
- `GET /api/version` - Get version information

## Database

The application supports **three database backends** with automatic migrations and feature parity:

### Supported Databases

| Database | Best For | Max Users | Setup Time | Cost |
|----------|----------|-----------|------------|------|
| **SQLite** (default) | Development, small deployments | <1,000 | 5 min | $0 |
| **MySQL** | Web apps, medium deployments | 10,000+ | 30 min | $ |
| **PostgreSQL** | Enterprise, complex queries | 100,000+ | 45 min | $$ |

### Quick Start - SQLite (Default)

No configuration needed! The database file is created automatically on first run:

```bash
# Just run the application
./gassigeher
```

Database file: `./gassigeher.db` (configurable via `DATABASE_PATH`)

### MySQL Configuration

```bash
# In .env file
DB_TYPE=mysql
DB_HOST=localhost
DB_PORT=3306
DB_NAME=gassigeher
DB_USER=gassigeher_user
DB_PASSWORD=your_secure_password
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
```

See **[MySQL_Setup_Guide.md](docs/MySQL_Setup_Guide.md)** for complete setup instructions.

### PostgreSQL Configuration

```bash
# In .env file
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

See **[PostgreSQL_Setup_Guide.md](docs/PostgreSQL_Setup_Guide.md)** for complete setup instructions.

### Database Selection Guide

**Choose SQLite if:**
- Development or testing
- Small animal shelter (<1,000 users)
- Simple deployment (single server)
- Zero setup time required

**Choose MySQL if:**
- Medium to large shelter (1,000-50,000 users)
- Proven web-scale performance needed
- Replication required
- Familiar with MySQL administration

**Choose PostgreSQL if:**
- Enterprise deployment (10,000+ users)
- Advanced features needed (JSON, full-text search)
- Complex analytics queries
- Strong ACID compliance required

See **[Database_Selection_Guide.md](docs/Database_Selection_Guide.md)** for detailed comparison.

### Tables Created (All Databases)
- `users` - User accounts and profiles (with first_name, last_name)
- `dogs` - Dog information (with is_featured, external_link)
- `bookings` - Walk bookings (with approval workflow)
- `blocked_dates` - Admin-blocked dates (global or per-dog)
- `experience_requests` - User level promotion requests
- `reactivation_requests` - Account reactivation requests
- `system_settings` - Configurable system settings
- `booking_time_rules` - Configurable time slots
- `custom_holidays` - Custom and API-fetched holidays
- `feiertage_cache` - Holiday API cache
- `schema_migrations` - Migration version tracking

All 21 migrations run automatically on application startup.

## Implementation Status

### ALL PHASES COMPLETE (10 of 10)

- **Phase 1**: Foundation (Auth, Database, Email)
- **Phase 2**: Dog Management (CRUD, Photos, Categories)
- **Phase 3**: Booking System (Create, View, Cancel, Auto-complete)
- **Phase 4**: Blocked Dates & Admin Actions (Block dates, Move bookings)
- **Phase 5**: Experience Levels (Request, Approve, Deny workflow)
- **Phase 6**: User Profiles & Photos (Edit, Upload, Email re-verification)
- **Phase 7**: Account Management & GDPR (Delete, Deactivate, Reactivate)
- **Phase 8**: Admin Dashboard & Reports (Stats, Activity, Settings)
- **Phase 9**: Polish & Testing (Test suite, Security, Documentation)
- **Phase 10**: Deployment (Production setup, Documentation)

**Status: PRODUCTION READY**

### Current Coverage
- **Backend Tests**: 305+ tests passing
  - Handlers: Comprehensive coverage
  - Models: Validation tests
  - Repository: CRUD and edge cases
  - Services: Auth, email, holidays, booking times
  - Middleware: Auth, security
  - Database: Migrations, dialect
- **Frontend**: Manual testing complete for all features
- **Security**: Headers, XSS protection, password validation
- **CI/CD**: GitHub Actions with Playwright E2E tests

See [ImplementationPlan.md](docs/ImplementationPlan.md) for complete phase details.

## Development Notes

### Color Scheme (Tierheim Göppingen)
- Primary Green: `#82b965`
- Dark Background: `#26272b`
- Dark Gray: `#33363b`
- Border Radius: `6px`
- System fonts only (Arial, sans-serif)

### Admin Access

**Super Admin System:**
- On first installation, a Super Admin is created automatically using `SUPER_ADMIN_EMAIL` from `.env`
- Super Admin credentials are saved in `SUPER_ADMIN_CREDENTIALS.txt`
- Super Admin can promote/demote other users to/from admin role via the web UI
- Admin privileges are stored in the database (no server restart needed)
- Change Super Admin password by editing `SUPER_ADMIN_CREDENTIALS.txt` and restarting

**Becoming an Admin:**
1. Register as a regular user (requires registration password from shelter)
2. Ask the Super Admin to promote you via "Benutzerverwaltung" page
3. Super Admin clicks "Zu Admin ernennen" on your account
4. You immediately gain admin access

### Experience Level System
- **Green (Beginner)**: Default for all new users, can book green-category dogs
- **Orange (Intermediate)**: Requires admin approval, can book green and orange dogs
- **Blue (Experienced)**: Requires admin approval, can book all dogs

### Booking Time Rules
The system supports configurable time slots:
- **Weekday slots**: Morning, afternoon, evening walks
- **Weekend slots**: Different timing for weekends
- **Blocked periods**: Feeding times, rest periods
- **Holiday handling**: Uses German public holiday API
- **Approval workflow**: Certain times may require admin approval

### Testing

Run all tests:
```bash
go test ./... -v
```

Run tests with coverage:
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Default System Settings
- Booking advance: 14 days
- Cancellation notice: 12 hours
- Auto-deactivation: 365 days (1 year)
- Booking time granularity: 15 minutes
- Morning walks require approval: true
- German state for holidays: BW (Baden-Württemberg)

These can be adjusted by admins in the settings page.

### Automated Tasks (Cron Jobs)

The application runs the following automated tasks:

1. **Auto-complete Bookings** (every 15 minutes)
   - Marks past scheduled bookings as completed
   - Updates booking status automatically

2. **Send Booking Reminders** (every 15 minutes)
   - Sends email reminders 1-2 hours before walks
   - Tracks sent reminders to prevent duplicates

3. **Auto-deactivate Inactive Users** (daily at 3:00 AM)
   - Checks for users inactive beyond configured period
   - Deactivates accounts with "auto_inactivity" reason
   - Sends notification emails

### Email Notifications

The system sends 18 types of email notifications:

**Authentication:**
1. Email verification link
2. Welcome email after verification (with optional WhatsApp link)
3. Password reset link

**Bookings:**
4. Booking confirmation
5. Booking reminder (1-2 hours before)
6. User cancellation confirmation
7. Admin cancellation notification
8. Booking approval notification
9. Booking rejection notification

**Admin Actions:**
10. Booking moved notification

**Experience Levels:**
11. Level promotion approved
12. Level promotion denied

**Account Lifecycle:**
13. Account deactivated notification
14. Account reactivated notification
15. Reactivation request denied
16. Account deletion confirmation

**Auto-deactivation:**
17. Auto-deactivation warning
18. Auto-deactivation notification

All emails use HTML templates with inline CSS for consistent branding.

## Security

The application implements multiple security measures:

- **Authentication**: JWT tokens with configurable expiration
- **Password Security**: bcrypt hashing with cost factor 12
- **Password Requirements**: Min 8 chars, uppercase, lowercase, number
- **Registration Password**: Shelter-controlled access code
- **Email Verification**: Required before account activation
- **Admin Authorization**: Database-stored, managed via UI
- **Security Headers**:
  - X-Frame-Options: DENY (clickjacking protection)
  - X-Content-Type-Options: nosniff (MIME sniffing protection)
  - X-XSS-Protection: enabled
  - Strict-Transport-Security: HTTPS enforcement
  - Content-Security-Policy: XSS protection
- **File Upload Validation**: Type and size checks
- **SQL Injection Protection**: Parameterized queries throughout
- **GDPR Compliance**: Right to deletion, data anonymization
- **XSS Protection**: HTML escaping in user name rendering

## Documentation

**Complete documentation suite: 9,500+ lines across 15 comprehensive guides**

See **[DOCUMENTATION_INDEX.md](docs/DOCUMENTATION_INDEX.md)** for navigation guide.

### Core Documentation

| Document | Lines | Purpose | Audience |
|----------|-------|---------|----------|
| **[README.md](README.md)** | 800+ | Project overview, setup, API list | Developers |
| **[ImplementationPlan.md](docs/ImplementationPlan.md)** | 1,500+ | Complete architecture & all 10 phases | Technical Leads |
| **[API.md](docs/API.md)** | 600+ | Complete REST API reference with examples | Developers/Integrators |
| **[DEPLOYMENT.md](docs/DEPLOYMENT.md)** | 600+ | Production deployment (SQLite, MySQL, PostgreSQL) | DevOps/System Admins |
| **[USER_GUIDE.md](docs/USER_GUIDE.md)** | 350+ | How to use the application (German) | End Users |
| **[ADMIN_GUIDE.md](docs/ADMIN_GUIDE.md)** | 500+ | Administrator operations manual | Administrators |
| **[PROJECT_SUMMARY.md](docs/PROJECT_SUMMARY.md)** | 500+ | Executive summary & statistics | Stakeholders |
| **[CLAUDE.md](CLAUDE.md)** | 600+ | AI assistant development guide | AI Developers |
| **[DOCUMENTATION_INDEX.md](docs/DOCUMENTATION_INDEX.md)** | 200+ | Documentation navigation | Everyone |

### Database Documentation

| Document | Lines | Purpose | Audience |
|----------|-------|---------|----------|
| **[Database_Selection_Guide.md](docs/Database_Selection_Guide.md)** | 300+ | Choosing the right database | Decision Makers |
| **[MySQL_Setup_Guide.md](docs/MySQL_Setup_Guide.md)** | 400+ | Complete MySQL setup and configuration | DevOps/System Admins |
| **[PostgreSQL_Setup_Guide.md](docs/PostgreSQL_Setup_Guide.md)** | 500+ | Complete PostgreSQL setup and configuration | DevOps/System Admins |
| **[MultiDatabase_Testing_Guide.md](docs/MultiDatabase_Testing_Guide.md)** | 300+ | Testing across all database backends | Developers/QA |
| **[DatabasesSupportPlan.md](docs/DatabasesSupportPlan.md)** | 2,300+ | Complete multi-database implementation plan | Technical Leads |

### Email Documentation

| Document | Purpose |
|----------|---------|
| **[Email_Provider_Selection_Guide.md](docs/Email_Provider_Selection_Guide.md)** | Choosing between Gmail API and SMTP |
| **[SMTP_Setup_Guides.md](docs/SMTP_Setup_Guides.md)** | Setup guides for various SMTP providers |

**Not sure where to start?** See [DOCUMENTATION_INDEX.md](docs/DOCUMENTATION_INDEX.md).

## Getting Started Guide

### For Users
1. Get registration password from the animal shelter
2. Visit the application URL
3. Click "Registrieren" to create an account
4. Enter registration password and your details
5. Verify your email (check inbox)
6. Login and join WhatsApp group (if enabled)
7. Start browsing dogs and book your first walk!

**Read**: [USER_GUIDE.md](docs/USER_GUIDE.md) for complete instructions.

### For Administrators

**First-Time Setup (Super Admin):**
1. Set `SUPER_ADMIN_EMAIL` in `.env` before first run
2. Start the application
3. Note the Super Admin credentials displayed in console
4. Credentials also saved in `SUPER_ADMIN_CREDENTIALS.txt`
5. Login with those credentials
6. Go to settings to configure:
   - Site logo
   - Registration password (share with users)
   - WhatsApp group link
   - Booking time rules
   - Holidays
7. Add dogs and start managing bookings

**Additional Administrators:**
1. Register and verify as normal user
2. Ask Super Admin to promote you via "Benutzerverwaltung"
3. Login - you'll be redirected to admin dashboard
4. Start managing dogs, users, and bookings

**Read**: [ADMIN_GUIDE.md](docs/ADMIN_GUIDE.md) for complete operations guide.

### For Developers
1. Clone repository
2. Copy `.env.example` to `.env`
3. Configure email provider (or skip for development)
4. Run `./bat.sh` (Linux/Mac) or `bat.bat` (Windows)
5. Visit `http://localhost:8080`

**Read**: [CLAUDE.md](CLAUDE.md) for development guide and [API.md](docs/API.md) for endpoints.

### For DevOps
1. Provision Ubuntu 22.04 server
2. Follow [DEPLOYMENT.md](docs/DEPLOYMENT.md) step-by-step
3. Configure SSL with Let's Encrypt
4. Setup automated backups
5. Monitor via `/api/health` endpoint

**Read**: [DEPLOYMENT.md](docs/DEPLOYMENT.md) for complete production setup.

## Project Statistics

| Category | Count |
|----------|-------|
| **Implementation Phases** | 10/10 (100%) |
| **Backend Files** | 131 Go files |
| **Frontend Pages** | 26 HTML pages |
| **API Endpoints** | 71 REST endpoints |
| **Database Tables** | 11 with indexes |
| **Database Migrations** | 21 |
| **Email Templates** | 18 HTML templates |
| **Test Cases** | 305+ (all passing) |
| **German Translations** | 400+ strings |
| **Documentation Files** | 15 guides |
| **Deployment Configs** | 3 production files |
| **Security Measures** | 12+ implemented |
| **Cron Jobs** | 3 automated tasks |

## Complete Feature List

**Implemented (60+ features)**:
User registration with password • Email verification • JWT authentication • Password reset • Profile management (first/last name) • Photo uploads • Experience levels (Green/Orange/Blue) • Level promotions • Dog browsing • Featured dogs • External dog links • Advanced filters • Dog booking • Booking approval workflow • Configurable time slots • Holiday integration • Booking cancellation • Booking notes • Booking reminders • Dashboard • GDPR account deletion • Auto-deactivation with notification • Reactivation workflow • Admin dashboard • Dog management • Dog-specific blocking • Availability toggle • Booking management • Move bookings • Block dates (global/per-dog) • User management • Admin promotion/demotion • Experience approvals • Registration password • Site logo configuration • WhatsApp integration • System settings • Real-time statistics • Activity feed • Email notifications (18 types) • Multi-provider email (Gmail/SMTP) • BCC admin copy • Auto-completion • Security headers • German i18n • Mobile-responsive design • SCSS modular styling • Terms & privacy pages • Embedded frontend • Version display • Health check • CI/CD pipeline • E2E testing

## What Makes Gassigeher Special

1. **Complete GDPR Compliance**: Full anonymization on deletion with legal email confirmation
2. **Experience-Based Access**: Progressive skill system (Green→Orange→Blue) with admin approvals
3. **Flexible Time Management**: Configurable booking slots with holiday awareness
4. **Multi-Provider Email**: Gmail API or any SMTP server with audit trail
5. **Controlled Registration**: Shelter-provided password prevents unauthorized sign-ups
6. **WhatsApp Integration**: Easy community onboarding for new users
7. **Dog-Specific Blocking**: Block dates for individual dogs (vet visits, etc.)
8. **Automated Lifecycle**: Auto-deactivation with warnings, reactivation workflow
9. **Health Management**: Quick dog availability toggle with reasons
10. **Comprehensive Admin Tools**: 10 admin pages with unified navigation
11. **Zero Frontend Dependencies**: Pure vanilla JavaScript, instant page loads
12. **Standalone Deployment**: Single binary with embedded frontend
13. **Email-First Communication**: 18 HTML email types for all actions
14. **Production-Ready**: Complete deployment package with systemd, nginx, backups, CI/CD

## Contributing

This is a complete application following the implementation plan. Each phase builds upon the previous one with comprehensive testing and documentation.

**All 10 phases are complete. The application is ready for production deployment.**

## License

This project is licensed under the **GNU General Public License v3.0** (GPL-3.0).

Copyright © 2025 Minh Cuong Tran

This program is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.

See the [gpl-3.0.txt](gpl-3.0.txt) file for the full license text.
