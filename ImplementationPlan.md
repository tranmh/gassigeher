# Gassigeher - Dog Walking Booking System
## Complete Implementation Plan

---

## Executive Summary

**Gassigeher** is a complete production-ready web-based dog walking booking system that connects dog walkers (Gassigeher) with dogs needing walks. The system features:

- Two user roles: Gassigeher (regular users) and Admins
- Experience-based access levels (Green/Blue/Orange)
- Calendar-based booking interface
- Comprehensive email notifications for all actions
- GDPR-compliant account deletion with data anonymization
- Automatic user lifecycle management (deactivation after 1 year inactivity)
- Dog health status management (temporary unavailability)
- Mobile-first responsive design
- German UI with internationalization support
- 90% test coverage
- Production deployment ready

---

## Technology Stack

### Backend
- **Language**: Go (Golang)
- **Database**: SQLite
- **Authentication**: JWT (JSON Web Tokens)
- **Email**: Gmail API
- **Testing**: Go standard testing library + testify
- **Router**: gorilla/mux or chi

### Frontend
- **Framework**: Vanilla JavaScript (ES6+)
- **Markup**: HTML5
- **Styling**: CSS3 (custom, no frameworks)
- **Internationalization**: Custom i18n JSON files
- **Calendar**: Custom or lightweight library (FullCalendar alternative)
- **Testing**: Jest or similar for JS testing

### DevOps
- **Version Control**: Git
- **Build**: Go build, no frontend bundler (vanilla JS)
- **Testing**: Go test, JS test runner
- **Target Coverage**: 90% line coverage

---

## Color Scheme & Design

Based on Tierheim Göppingen aesthetic:

- **Primary Accent**: `#82b965` (sage green)
- **Dark Background**: `#26272b`, `#33363b` (charcoal gray)
- **Text on Dark**: `#ffffff` (white)
- **Font**: Titillium (with Arial, sans-serif fallbacks)
- **Border Radius**: 6px for consistency
- **Approach**: Clean, functional, accessible, dog-friendly

---

## User Roles & Permissions

### Gassigeher (Regular Users)
**Can:**
- Register with email, phone, name
- Verify email address
- Accept Terms & Conditions
- Receive welcome email with app instructions after verification
- Login/logout
- Browse all dogs (with category indicators and availability status)
- See when dogs are temporarily unavailable (e.g., "Currently unavailable: Health check")
- Filter dogs by breed, size, age, special needs, category
- View dog details (name, breed, size, age, photo, special needs, pick-up location, walk route, duration, instructions)
- Book dogs within their experience level (Green → Blue → Orange)
- View higher-level dogs (disabled, labeled "Requires X level")
- Adjust suggested walk times when booking
- Book multiple dogs for same walk time (unlimited)
- View calendar of bookings (own bookings only)
- Receive email notifications for all actions
- Cancel/reschedule bookings (12 hours notice minimum, admin-adjustable)
- View walk history (own walks only)
- Add optional notes after walk completion (auto-completed)
- Request experience level promotion (self-select, admin approves)
- Upload and update profile photo
- Edit profile (name, email, phone)
- Change password
- Reset forgotten password (self-service)
- Delete account (GDPR-compliant: personal data removed, walk history anonymized)
- Request account reactivation via email if deactivated

**Cannot:**
- Book dogs above their experience level without promotion
- Book temporarily unavailable dogs
- View other users' bookings or history
- Manage dogs
- Access admin features
- Use account if deactivated (auto-deactivated after 1 year of inactivity)

### Admins
**Can:**
- All Gassigeher capabilities
- View admin dashboard with statistics:
  - Total walks completed
  - Most popular dogs
  - Most active users (and inactive users count)
  - Upcoming walks (all users)
  - Recent activity feed
  - Pending reactivation requests
- View all users and their details (active and inactive)
- Manually activate/deactivate user accounts
- Process user reactivation requests
- Approve/deny experience level promotion requests
- Add, edit, delete dogs
- Set dog category (Green/Blue/Orange)
- Mark dogs as temporarily unavailable (e.g., sick) with optional reason
- Mark dogs as available again
- Upload dog photos
- Manage all dog details (breed, size, age, special needs, photo, pick-up location, route preferences, duration, special instructions)
- Set default suggested walk times for dogs
- View all bookings (all users, including anonymized deleted users)
- Cancel any booking (required to provide reason)
- Move/reschedule any booking (required to provide reason)
- Block specific dates with reason (visible to users)
- Adjust system settings:
  - Booking advance limit (default: 14 days)
  - Cancellation notice period (default: 12 hours)
  - Auto-deactivation period (default: 1 year)
- View complete walk history (all users, including anonymized)
- View all walk notes from users (including from deleted/anonymized accounts)

**Admin Creation:**
- Config-based: Admins defined in environment variables or config file
- No UI for admin management (keeps security tight)
- Example: `ADMIN_EMAILS=admin@example.com,admin2@example.com`

---

## Experience Level System

### Categories
1. **Green (Beginner)**
   - Default for new users
   - Can book Green-category dogs only

2. **Blue (Experienced)**
   - Can book Green and Blue dogs
   - Requires admin approval

3. **Orange (Dedicated Experienced)**
   - Can book all dogs (Green, Blue, Orange)
   - Requires admin approval

### Promotion Flow
1. User requests promotion from their profile page
2. Admin receives notification (dashboard alert)
3. Admin reviews user's walk history
4. Admin approves or denies with optional message
5. User receives email notification of decision
6. If approved, user can immediately book higher-level dogs

---

## Core Features

### 1. Authentication & Registration

#### Registration Flow
1. User visits landing page, clicks "Register"
2. Form fields:
   - Name (required)
   - Email (required, validated)
   - Phone number (required)
   - Password (required, min 8 chars, complexity rules)
   - Confirm password
   - Accept Terms & Conditions checkbox (required, links to T&C page)
3. Submit creates unverified account
4. System sends verification email with token link
5. User clicks link, account becomes verified
6. User can now login and access booking area
7. Initial experience level: Green (can request promotion later)

#### Login Flow
1. Email + password
2. JWT token generated (24-hour expiration, configurable)
3. Token stored in localStorage
4. Redirect to dashboard

#### Password Management
1. **Change Password** (logged in):
   - Old password required
   - New password + confirmation

2. **Forgot Password** (logged out):
   - Enter email
   - Receive reset link via email (token expires in 1 hour)
   - Click link, enter new password
   - Redirect to login

### 2. Dog Management (Admin Only)

#### Dog Model
```javascript
{
  id: int,
  name: string,
  breed: string,
  size: enum('small', 'medium', 'large'),
  age: int (years),
  category: enum('green', 'blue', 'orange'),
  photo: string (filename),
  specialNeeds: text (markdown-supported),
  pickupLocation: string (address),
  walkRoute: text (preferences/suggestions),
  walkDuration: int (minutes),
  specialInstructions: text,
  defaultMorningTime: time (suggested),
  defaultEveningTime: time (suggested),
  isAvailable: boolean (default: true),
  unavailableReason: text (optional, e.g., "Health check", "Vet visit"),
  unavailableSince: timestamp (null if available),
  createdAt: timestamp,
  updatedAt: timestamp
}
```

#### CRUD Operations
- **Create**: Admin fills form, uploads photo
- **Read**: All users can view (category restrictions apply, availability status displayed)
- **Update**: Admin edits any field, can replace photo
- **Delete**: Admin can remove dog (prevent if future bookings exist to maintain data integrity)
- **Toggle Availability**: Admin can quickly mark as unavailable/available with optional reason

### 3. Booking System

#### Booking Model
```javascript
{
  id: int,
  userId: int (foreign key),
  dogId: int (foreign key),
  date: date,
  walkType: enum('morning', 'evening'),
  scheduledTime: time (can differ from dog's default),
  status: enum('scheduled', 'completed', 'cancelled'),
  completedAt: timestamp (null until completed),
  userNotes: text (optional, added after completion),
  adminCancellationReason: text (null unless admin cancelled),
  createdAt: timestamp,
  updatedAt: timestamp
}
```

#### Booking Rules
1. Users can book up to 14 days in advance (admin-adjustable)
2. Cannot book past dates
3. Dog must be in user's allowed category (Green/Blue/Orange)
4. Dog must be available (isAvailable = true)
5. User must be active (not deactivated)
6. One user can book same dog for both morning and evening same day
7. Multiple users can book same dog on same day (different walk types)
8. Dog cannot be double-booked for same walk type on same day
9. Cannot book on admin-blocked dates
10. User can book unlimited dogs for same walk time
11. Suggested times from dog profile, user can adjust

#### Booking Flow (User)
1. Navigate to calendar view
2. Select date (within 14-day window)
3. See available dogs for that date
4. Filter by breed, size, age, special needs, category
5. Click dog to see details
6. Select morning or evening walk
7. See suggested time, can adjust
8. Confirm booking
9. Receive email confirmation

#### Cancellation Flow (User)
1. View booking in calendar or list
2. Click cancel (must be 12+ hours before scheduled time)
3. Confirm cancellation
4. Booking status → cancelled
5. Receive email confirmation

#### Admin Actions
1. **Cancel Booking**:
   - Select booking
   - Enter reason (required)
   - Confirm
   - User receives email with reason

2. **Move Booking**:
   - Select booking
   - Choose new date/time
   - Enter reason (required)
   - Confirm
   - User receives email with old/new details and reason

### 4. Blocked Days

#### Blocked Day Model
```javascript
{
  id: int,
  date: date,
  reason: string,
  createdBy: int (admin user ID),
  createdAt: timestamp
}
```

#### Functionality
- Admin selects date(s) to block
- Enters reason (displayed to users)
- Blocked dates show in calendar as unavailable with reason tooltip
- Users cannot create bookings on blocked dates
- Admin can unblock dates

### 5. Walk Completion & Notes

#### Auto-Completion
- Cron job runs every hour
- Checks for scheduled walks where `date + time < now`
- Updates status to 'completed'
- Sets completedAt timestamp

#### User Notes
- After walk completes, user can add optional notes
- Notes visible to admins only
- Edit allowed for 24 hours after completion
- Stored in booking.userNotes field

### 6. Account Deletion & GDPR Compliance

#### Account Deletion Flow
1. User clicks "Delete Account" in profile settings
2. Confirmation modal with warning (action cannot be undone, walk history will be anonymized)
3. User enters password to confirm
4. System performs anonymization:
   - Generate unique anonymous ID (e.g., `anonymous_user_1234567890`)
   - Delete personal data: name → "Deleted User", email → null, phone → null, profile_photo → deleted
   - Set `is_deleted = true`, `deleted_at = NOW()`
   - Keep user_id for foreign key integrity
   - All past bookings and notes remain but show anonymous name
5. User logged out immediately
6. Confirmation email sent to original email (before deletion) as legal proof

#### Data Retention
- **Deleted**: Name, email, phone number, profile photo, password hash
- **Anonymized**: User ID becomes anonymous reference
- **Retained**: Walk history (bookings), walk notes, timestamps
- **Reason**: Walk history needed for dog care records, admin statistics, legal/audit trail

#### GDPR Compliance Notes
- Right to be forgotten: Personal identifiable information removed
- Right to data portability: Can be added as future enhancement (export data before deletion)
- Consent tracking: Terms acceptance timestamp kept
- Legal basis: Legitimate interest (animal care records)

---

### 7. User Lifecycle Management

#### Automatic Deactivation
- **Trigger**: No activity for 1 year (admin-adjustable)
- **Activity Definition**: Last login, last booking created, or last walk completed
- **Process**:
  - Cron job runs daily at 3am
  - Checks for users where `last_activity_at < NOW() - 1 year`
  - Sets `is_active = false`, `deactivated_at = NOW()`, `deactivation_reason = 'auto_inactivity'`
  - Email sent: "Ihr Konto wurde deaktiviert"
- **Effect**: User cannot login, all future bookings cancelled (with notification)

#### Manual Deactivation (Admin)
- Admin selects user from user management page
- Enters reason (required): "Unreliable", "Rule violation", "User request", etc.
- Confirms deactivation
- User receives email with reason
- All future bookings cancelled
- `is_active = false`, `deactivated_at = NOW()`, `deactivation_reason = admin's reason`

#### Reactivation Flow
1. **User-initiated**:
   - Deactivated user tries to login
   - System shows: "Your account is deactivated. Request reactivation?"
   - User clicks "Request Reactivation"
   - Email sent to admin with user info and deactivation history
   - Admin receives dashboard notification

2. **Admin review**:
   - Admin views reactivation request
   - Sees user's walk history, reason for deactivation
   - Approves or denies with optional message
   - If approved: `is_active = true`, `reactivated_at = NOW()`
   - User receives email with decision

3. **Admin-initiated**:
   - Admin can manually activate any user from user management page
   - Optional message to user
   - User receives email: "Ihr Konto wurde wieder aktiviert"

---

### 8. Email Notifications (Gmail API)

#### All Email Types
1. **Registration + Welcome**:
   - Subject: "Willkommen bei Gassigeher"
   - Body: Welcome message + verification link + app instructions
   - Attachments: None
   - Sent: Immediately after registration

2. **Email Verification**:
   - Subject: "E-Mail-Adresse bestätigen"
   - Body: Click link to verify

3. **Welcome Email** (after verification):
   - Subject: "Los geht's! Ihr Konto ist aktiviert"
   - Body:
     - Welcome message
     - How to browse dogs
     - How to make first booking
     - Experience level explanation
     - Contact info for support
   - Sent: Immediately after successful verification

4. **Booking Confirmation** (user creates):
   - Subject: "Buchungsbestätigung - [Dog Name]"
   - Body: Date, time, dog details, pickup location

5. **Booking Reminder** (1 hour before):
   - Subject: "Erinnerung: Gassirunde mit [Dog Name] in 1 Stunde"
   - Body: Reminder with details

6. **User Cancellation**:
   - Subject: "Buchung storniert - [Dog Name]"
   - Body: Confirmation of cancellation

7. **Admin Cancellation**:
   - Subject: "Deine Buchung wurde storniert - [Dog Name]"
   - Body: Reason from admin, apology

8. **Admin Move**:
   - Subject: "Deine Buchung wurde verschoben - [Dog Name]"
   - Body: Old date/time, new date/time, reason

9. **Password Reset**:
   - Subject: "Passwort zurücksetzen"
   - Body: Reset link with token

10. **Email Change** (when user updates email):
    - Subject: "E-Mail-Adresse bestätigen"
    - Body: New verification link

11. **Experience Level Request** (to admin):
    - Dashboard notification, not email (avoid spam)

12. **Experience Level Approved/Denied**:
    - Subject: "Dein Antrag auf [Blue/Orange] Level"
    - Body: Approved/denied, optional admin message

13. **Account Deactivated**:
    - Subject: "Ihr Konto wurde deaktiviert"
    - Body: Reason (inactivity or admin action), how to request reactivation

14. **Account Reactivated**:
    - Subject: "Ihr Konto wurde wieder aktiviert"
    - Body: Welcome back, optional admin message

15. **Reactivation Request Received** (to admin):
    - Subject: "Reaktivierungsanfrage von [User Name]"
    - Body: User details, deactivation reason, link to admin dashboard

16. **Reactivation Denied**:
    - Subject: "Ihre Reaktivierungsanfrage"
    - Body: Request denied, optional admin message

17. **Account Deletion Confirmation**:
    - Subject: "Ihr Konto wurde gelöscht"
    - Body: Confirmation, data deleted, walk history anonymized, legal notice

#### Gmail API Setup
- OAuth 2.0 credentials
- Send emails from configured Gmail account
- Store credentials securely (environment variables)
- HTML email templates with inline CSS

---

## Database Schema

### Users Table
```sql
CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  email TEXT UNIQUE, -- Can be NULL after deletion
  phone TEXT, -- Can be NULL after deletion
  password_hash TEXT, -- Can be NULL after deletion
  experience_level TEXT DEFAULT 'green' CHECK(experience_level IN ('green', 'blue', 'orange')),
  is_verified INTEGER DEFAULT 0,
  is_active INTEGER DEFAULT 1, -- For deactivation
  is_deleted INTEGER DEFAULT 0, -- For GDPR deletion
  verification_token TEXT,
  verification_token_expires TIMESTAMP,
  password_reset_token TEXT,
  password_reset_expires TIMESTAMP,
  profile_photo TEXT,
  anonymous_id TEXT UNIQUE, -- Generated on deletion (e.g., 'anonymous_user_1234567890')
  terms_accepted_at TIMESTAMP NOT NULL,
  last_activity_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, -- For auto-deactivation
  deactivated_at TIMESTAMP,
  deactivation_reason TEXT, -- 'auto_inactivity', 'admin_action', etc.
  reactivated_at TIMESTAMP,
  deleted_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index for auto-deactivation query performance
CREATE INDEX idx_users_last_activity ON users(last_activity_at, is_active);
-- Index for email lookup (login)
CREATE INDEX idx_users_email ON users(email);
```

### Dogs Table
```sql
CREATE TABLE dogs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  breed TEXT NOT NULL,
  size TEXT CHECK(size IN ('small', 'medium', 'large')),
  age INTEGER,
  category TEXT CHECK(category IN ('green', 'blue', 'orange')),
  photo TEXT,
  special_needs TEXT,
  pickup_location TEXT,
  walk_route TEXT,
  walk_duration INTEGER, -- minutes
  special_instructions TEXT,
  default_morning_time TEXT, -- HH:MM format
  default_evening_time TEXT, -- HH:MM format
  is_available INTEGER DEFAULT 1, -- Temporary unavailability toggle
  unavailable_reason TEXT, -- Optional reason (e.g., "Vet visit", "Health check")
  unavailable_since TIMESTAMP, -- When marked unavailable
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index for filtering available dogs
CREATE INDEX idx_dogs_available ON dogs(is_available, category);
```

### Bookings Table
```sql
CREATE TABLE bookings (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  dog_id INTEGER NOT NULL,
  date DATE NOT NULL,
  walk_type TEXT CHECK(walk_type IN ('morning', 'evening')),
  scheduled_time TEXT NOT NULL, -- HH:MM format
  status TEXT DEFAULT 'scheduled' CHECK(status IN ('scheduled', 'completed', 'cancelled')),
  completed_at TIMESTAMP,
  user_notes TEXT,
  admin_cancellation_reason TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (dog_id) REFERENCES dogs(id) ON DELETE CASCADE,
  UNIQUE(dog_id, date, walk_type) -- prevent double-booking same walk
);
```

### Blocked Dates Table
```sql
CREATE TABLE blocked_dates (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  date DATE NOT NULL UNIQUE,
  reason TEXT NOT NULL,
  created_by INTEGER NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (created_by) REFERENCES users(id)
);
```

### Experience Level Requests Table
```sql
CREATE TABLE experience_requests (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  requested_level TEXT CHECK(requested_level IN ('blue', 'orange')),
  status TEXT DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'denied')),
  admin_message TEXT,
  reviewed_by INTEGER,
  reviewed_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (reviewed_by) REFERENCES users(id)
);
```

### System Settings Table
```sql
CREATE TABLE system_settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Default values:
INSERT INTO system_settings (key, value) VALUES
  ('booking_advance_days', '14'),
  ('cancellation_notice_hours', '12'),
  ('auto_deactivation_days', '365'); -- 1 year = 365 days
```

### Reactivation Requests Table
```sql
CREATE TABLE reactivation_requests (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  status TEXT DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'denied')),
  admin_message TEXT,
  reviewed_by INTEGER, -- Admin user ID
  reviewed_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (reviewed_by) REFERENCES users(id)
);

-- Index for pending requests query
CREATE INDEX idx_reactivation_pending ON reactivation_requests(status, created_at);
```

---

## API Endpoints

### Authentication
- `POST /api/auth/register` - Register new user
- `POST /api/auth/verify-email` - Verify email with token
- `POST /api/auth/login` - Login, returns JWT
- `POST /api/auth/logout` - Invalidate token (client-side mostly)
- `POST /api/auth/forgot-password` - Request password reset
- `POST /api/auth/reset-password` - Reset password with token
- `PUT /api/auth/change-password` - Change password (authenticated)

### Users
- `GET /api/users/me` - Get current user profile
- `PUT /api/users/me` - Update own profile
- `POST /api/users/me/photo` - Upload profile photo
- `DELETE /api/users/me` - Delete own account (GDPR, requires password confirmation)
- `GET /api/users` - List all users (admin only, includes active/inactive filter)
- `GET /api/users/:id` - Get user by ID (admin only)
- `PUT /api/users/:id/activate` - Activate user account (admin only)
- `PUT /api/users/:id/deactivate` - Deactivate user account (admin only, requires reason)

### Dogs
- `GET /api/dogs` - List all dogs (with filters: breed, size, age, category, search, availability)
- `GET /api/dogs/:id` - Get dog details
- `POST /api/dogs` - Create dog (admin only)
- `PUT /api/dogs/:id` - Update dog (admin only)
- `DELETE /api/dogs/:id` - Delete dog (admin only, prevents if future bookings exist)
- `POST /api/dogs/:id/photo` - Upload dog photo (admin only)
- `PUT /api/dogs/:id/availability` - Toggle dog availability (admin only, with optional reason)

### Bookings
- `GET /api/bookings` - List bookings (user sees own, admin sees all)
- `GET /api/bookings/:id` - Get booking details
- `POST /api/bookings` - Create booking
- `PUT /api/bookings/:id/cancel` - Cancel booking (user or admin)
- `PUT /api/bookings/:id/move` - Move booking (admin only, requires reason)
- `PUT /api/bookings/:id/notes` - Add/update user notes
- `GET /api/bookings/calendar/:year/:month` - Get calendar data for month

### Blocked Dates
- `GET /api/blocked-dates` - List all blocked dates
- `POST /api/blocked-dates` - Block a date (admin only)
- `DELETE /api/blocked-dates/:id` - Unblock a date (admin only)

### Experience Requests
- `POST /api/experience-requests` - Request level promotion
- `GET /api/experience-requests` - List requests (user sees own, admin sees all pending)
- `PUT /api/experience-requests/:id/approve` - Approve request (admin only)
- `PUT /api/experience-requests/:id/deny` - Deny request (admin only, optional message)

### Reactivation Requests
- `POST /api/reactivation-requests` - Request account reactivation (deactivated users only)
- `GET /api/reactivation-requests` - List requests (user sees own, admin sees all pending)
- `PUT /api/reactivation-requests/:id/approve` - Approve reactivation (admin only, optional message)
- `PUT /api/reactivation-requests/:id/deny` - Deny reactivation (admin only, optional message)

### Admin Dashboard
- `GET /api/admin/stats` - Get dashboard statistics (admin only)
- `GET /api/admin/recent-activity` - Get recent activity feed (admin only)

### System Settings
- `GET /api/settings` - Get all settings (admin only)
- `PUT /api/settings/:key` - Update setting (admin only)

### Terms & Conditions
- `GET /api/terms` - Get current T&C HTML content
- `PUT /api/terms` - Update T&C (admin only)

---

## Frontend Structure

```
/frontend
├── /assets
│   ├── /css
│   │   ├── main.css          # Main styles
│   │   ├── calendar.css      # Calendar-specific
│   │   ├── forms.css         # Form styles
│   │   └── mobile.css        # Mobile overrides
│   ├── /images
│   │   ├── logo.svg
│   │   ├── dog-placeholder.svg
│   │   └── icons/
│   └── /uploads              # User/dog photos (served by backend)
├── /js
│   ├── main.js               # App initialization
│   ├── auth.js               # Auth logic
│   ├── api.js                # API client
│   ├── router.js             # Client-side routing
│   ├── i18n.js               # Internationalization
│   ├── calendar.js           # Calendar component
│   ├── dog-list.js           # Dog listing
│   ├── booking-form.js       # Booking modal
│   ├── profile.js            # User profile
│   ├── account-deletion.js   # Account deletion flow
│   ├── admin-dashboard.js    # Admin dashboard
│   ├── admin-dogs.js         # Dog management
│   ├── admin-users.js        # User management (activate/deactivate)
│   ├── admin-reactivation.js # Reactivation request management
│   └── utils.js              # Utilities
├── /i18n
│   ├── de.json               # German translations
│   └── en.json               # English translations (future)
├── index.html                # Landing page
├── app.html                  # Main app (post-login)
├── terms.html                # Terms & Conditions
└── README.md
```

### Key Frontend Components

#### 1. Calendar View (Main UI)
- Month view with day cells
- Click day → show available dogs modal
- Color-coded indicators:
  - Green/Blue/Orange dots for dog categories
  - Gray for blocked days (hover shows reason)
  - Highlighted for user's bookings
- Mobile: Swipe between months
- Desktop: Month navigation arrows

#### 2. Dog Card Component
```html
<div class="dog-card" data-category="green" data-available="true">
  <img src="/uploads/dogs/dog-123.jpg" alt="Dog Name">
  <div class="dog-info">
    <h3>Dog Name</h3>
    <span class="dog-category green">Alle</span>
    <!-- Availability status (only shown if unavailable) -->
    <div class="dog-unavailable" style="display: none;">
      <span class="status-badge">Momentan nicht verfügbar</span>
      <p class="unavailable-reason">Tierarztbesuch</p>
    </div>
    <p>Rasse: Golden Retriever</p>
    <p>Größe: Groß</p>
    <p>Alter: 3 Jahre</p>
    <button class="btn-book" disabled>Buchen</button>
  </div>
</div>
```

#### 3. Booking Modal
- Shows dog details
- Date/time picker (with suggested time pre-filled)
- Morning/evening toggle
- Special instructions display
- Pickup location map/address
- Confirm button

#### 4. Admin Dashboard Cards
```
┌──────────────────────┬──────────────────────┬──────────────────────┐
│  Total Walks         │  Active Users        │  Inactive Users      │
│  1,234               │  89                  │  12                  │
└──────────────────────┴──────────────────────┴──────────────────────┘
┌──────────────────────┬──────────────────────┬──────────────────────┐
│  Upcoming Today      │  Experience Requests │  Reactivation Reqs   │
│  12                  │  3                   │  2                   │
└──────────────────────┴──────────────────────┴──────────────────────┘
┌──────────────────────┬──────────────────────┐
│  Available Dogs      │  Unavailable Dogs    │
│  15                  │  2                   │
└──────────────────────┴──────────────────────┘
┌─────────────────────────────────────────────┐
│  Recent Activity                            │
│  • User A booked Buddy (10 min ago)        │
│  • User B completed walk with Max          │
│  • User C requested Blue level             │
│  • Dog "Bella" marked unavailable (Vet)    │
│  • User D reactivation approved            │
└─────────────────────────────────────────────┘
```

---

## Internationalization (i18n)

### Structure
All UI strings stored in JSON files:

**de.json** (German):
```json
{
  "nav": {
    "home": "Startseite",
    "calendar": "Kalender",
    "profile": "Profil",
    "logout": "Abmelden"
  },
  "dogs": {
    "category_green": "Alle",
    "category_blue": "Erfahrene",
    "category_orange": "Nur erfahrene",
    "size_small": "Klein",
    "size_medium": "Mittel",
    "size_large": "Groß"
  },
  "booking": {
    "title": "Hund buchen",
    "morning": "Morgen",
    "evening": "Abend",
    "confirm": "Buchung bestätigen",
    "cancel": "Stornieren"
  },
  // ... all strings
}
```

### Usage in JS
```javascript
// i18n.js
class I18n {
  constructor(locale = 'de') {
    this.locale = locale;
    this.translations = {};
  }

  async load() {
    const response = await fetch(`/i18n/${this.locale}.json`);
    this.translations = await response.json();
  }

  t(key) {
    // Access nested keys: "dogs.category_green"
    const keys = key.split('.');
    let value = this.translations;
    for (const k of keys) {
      value = value[k];
    }
    return value || key;
  }
}

// Usage:
const i18n = new I18n('de');
await i18n.load();
document.getElementById('title').textContent = i18n.t('booking.title');
```

### HTML Data Attributes
```html
<button data-i18n="booking.confirm">Booking bestätigen</button>

<!-- JS auto-translates on page load: -->
<script>
  document.querySelectorAll('[data-i18n]').forEach(el => {
    el.textContent = i18n.t(el.dataset.i18n);
  });
</script>
```

---

## Authentication & Security

### JWT Implementation
1. **Token Generation** (server):
   ```go
   token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
     "user_id": user.ID,
     "email": user.Email,
     "is_admin": isAdmin(user.Email),
     "exp": time.Now().Add(24 * time.Hour).Unix(),
   })
   ```

2. **Token Storage** (client):
   - Store in `localStorage` (key: `gassigeher_token`)
   - Send in `Authorization: Bearer <token>` header

3. **Token Validation** (server middleware):
   - Parse and verify signature
   - Check expiration
   - Extract user_id, is_admin
   - Attach to request context

### Password Security
- Hash: bcrypt with cost factor 12
- Requirements: min 8 chars, 1 uppercase, 1 lowercase, 1 number
- Reset tokens: random 32-byte hex, 1-hour expiration

### Email Verification
- Token: random 32-byte hex
- Expiration: 24 hours
- One-time use (deleted after verification)

### File Upload Security
- Validate file types (JPEG, PNG only)
- Max size: 5MB
- Sanitize filenames
- Store outside web root, serve via Go handler with authentication

### Admin Authorization
```go
func isAdmin(email string) bool {
  adminEmails := strings.Split(os.Getenv("ADMIN_EMAILS"), ",")
  for _, admin := range adminEmails {
    if strings.TrimSpace(admin) == email {
      return true
    }
  }
  return false
}
```

---

## Testing Strategy

### Backend Tests (Go)

#### Unit Tests
- **Models**: Test struct methods, validations
- **Handlers**: Test HTTP handlers with mock DB
- **Services**: Test business logic (booking rules, availability checks)
- **Email**: Mock Gmail API, verify correct templates/recipients
- **Auth**: Test JWT generation/validation, password hashing

#### Integration Tests
- **API**: Test full request/response cycles with test DB
- **Database**: Test queries with temporary SQLite file
- **End-to-end**: Test complete flows (register → verify → login → book)

#### Example Test Structure
```go
// handlers_test.go
func TestCreateBooking(t *testing.T) {
  // Setup test DB
  db := setupTestDB()
  defer db.Close()

  // Create test data
  user := createTestUser(db)
  dog := createTestDog(db, "green")

  // Test successful booking
  t.Run("Successful booking", func(t *testing.T) {
    req := createBookingRequest(user.ID, dog.ID, "2025-01-20", "morning", "09:00")
    resp := executeRequest(req)
    assert.Equal(t, http.StatusCreated, resp.Code)
  })

  // Test booking above user level
  t.Run("Booking above level fails", func(t *testing.T) {
    orangeDog := createTestDog(db, "orange")
    req := createBookingRequest(user.ID, orangeDog.ID, "2025-01-20", "morning", "09:00")
    resp := executeRequest(req)
    assert.Equal(t, http.StatusForbidden, resp.Code)
  })

  // Test double-booking prevention
  // Test blocked date prevention
  // Test booking advance limit
  // etc.
}
```

### Frontend Tests (JavaScript)

#### Unit Tests
- **API Client**: Mock fetch, test request formatting
- **i18n**: Test translation loading and key lookup
- **Utilities**: Test date formatting, validation functions
- **Calendar**: Test date calculations, availability logic

#### Integration Tests
- **Components**: Test DOM rendering with test data
- **Forms**: Test validation, submission
- **Navigation**: Test routing and state management

#### Example Test Structure
```javascript
// calendar.test.js
describe('Calendar Component', () => {
  test('renders month correctly', () => {
    const calendar = new Calendar(2025, 0); // January 2025
    const html = calendar.render();
    expect(html).toContain('Januar 2025');
    expect(html.match(/<td/g).length).toBeGreaterThan(28);
  });

  test('highlights blocked dates', () => {
    const calendar = new Calendar(2025, 0);
    calendar.setBlockedDates(['2025-01-15']);
    const html = calendar.render();
    expect(html).toContain('date-blocked');
  });

  test('shows user bookings', () => {
    const calendar = new Calendar(2025, 0);
    calendar.setUserBookings([{date: '2025-01-20', dog: 'Buddy'}]);
    const html = calendar.render();
    expect(html).toContain('user-booking');
  });
});
```

### Coverage Requirements
- **Target**: 90% line coverage
- **Tools**:
  - Go: `go test -coverprofile=coverage.out`
  - JS: `jest --coverage`
- **CI Check**: Fail build if coverage drops below 85%

### Test Data
- Create seed script for test database
- Include variety of dogs (all categories, sizes, breeds)
- Include test users (green, blue, orange levels)
- Include past, current, and future bookings
- Include blocked dates

---

## Development Phases

### Phase 1: Foundation (Week 1-2) // DONE ✅
**Backend:**
- [x] Project setup (Go modules, directory structure)
- [x] SQLite database setup with migrations (all tables including new ones)
- [x] User model and authentication (register, login, JWT)
- [x] Email verification flow (Gmail API setup)
- [x] Welcome email after verification
- [x] Password reset flow
- [x] Admin middleware and config-based admin detection
- [x] Basic API endpoints (auth, users)
- [x] Last activity tracking (update on login, booking)
- [x] Unit tests structure (0% coverage, tests to be written in Phase 9)

**Frontend:**
- [x] HTML/CSS boilerplate with color scheme (Tierheim Göppingen)
- [x] Landing page design
- [x] Registration form with validation
- [x] Login page
- [x] Email verification success page
- [x] Password reset flow pages
- [x] i18n setup (de.json, i18n.js)
- [x] API client setup (api.js)
- [x] Client-side routing (router.js)

**Additional:**
- [x] Build scripts (bat.bat for Windows, bat.sh for Linux/Mac)
- [x] .env configuration file
- [x] README.md with complete setup instructions
- [x] Terms & Conditions placeholder page

**Deliverable:** ✅ Users can register, verify email, receive welcome email, login, and reset password. German UI with i18n foundation. Build compiles successfully.

---

### Phase 2: Dog Management (Week 3) // DONE ✅
**Backend:**
- [x] Dog model and CRUD endpoints
- [x] File upload handling (dog photos)
- [x] Dog filtering and search
- [x] Category-based access control
- [x] Dog availability toggle endpoint (mark unavailable/available)
- [x] Prevent booking unavailable dogs
- [x] Prevent deletion if future bookings exist
- [x] Unit and integration tests structure (tests to be written in Phase 9)

**Frontend:**
- [x] Admin: Dog management UI (list, create, edit, delete)
- [x] Admin: Photo upload for dogs
- [x] Admin: Quick availability toggle button with reason input
- [x] User: Dog browsing page with filters (including availability filter)
- [x] Dog detail modal/page (placeholder link for Phase 3)
- [x] Dog card component with category indicators and availability status
- [x] Visual indicator for unavailable dogs (grayed out, badge) and locked dogs
- [x] Responsive design for mobile

**Deliverable:** ✅ Admins can manage dogs and toggle availability. Users can browse and filter dogs by category and see availability status. Experience level restrictions are visually enforced.

---

### Phase 3: Booking System (Week 4-5)
**Backend:**
- [ ] Booking model and endpoints
- [ ] Booking validation rules (level, double-booking, advance limit, blocked dates)
- [ ] System settings table and endpoints
- [ ] Auto-completion cron job
- [ ] User notes for completed walks
- [ ] Cancellation logic with notice period check
- [ ] Email notifications for bookings (confirmation, reminder, cancellation)
- [ ] Comprehensive tests for booking rules

**Frontend:**
- [ ] Calendar component (month view, day selection)
- [ ] Booking modal (date, time, dog selection)
- [ ] User dashboard (upcoming bookings)
- [ ] Booking list view (alternative to calendar)
- [ ] Cancellation confirmation modal
- [ ] Mobile-optimized calendar (swipe navigation)
- [ ] Tests for calendar logic

**Deliverable:** Users can book dogs via calendar, receive emails, and cancel bookings. Walks auto-complete.

---

### Phase 4: Blocked Dates & Admin Actions (Week 6)
**Backend:**
- [ ] Blocked dates model and endpoints
- [ ] Admin cancel booking with reason
- [ ] Admin move booking with reason
- [ ] Email notifications for admin actions (cancel, move)
- [ ] Tests for admin actions

**Frontend:**
- [ ] Admin: Blocked dates management (add, remove)
- [ ] Admin: Booking list with cancel/move actions
- [ ] Admin: Reason input modals
- [ ] Calendar: Display blocked dates with reason tooltips
- [ ] Tests for admin UI

**Deliverable:** Admins can block dates and manage all bookings. Users see blocked dates on calendar.

---

### Phase 5: Experience Levels (Week 7)
**Backend:**
- [ ] Experience requests model and endpoints
- [ ] Approval/denial logic with email notifications
- [ ] Update booking validation to check user level
- [ ] Tests for experience system

**Frontend:**
- [ ] User: Request level promotion button/page
- [ ] Admin: Pending requests list on dashboard
- [ ] Admin: Approve/deny UI with optional message
- [ ] Dog cards: Show "Requires X level" for inaccessible dogs
- [ ] Tests for level restrictions

**Deliverable:** Users can request promotions. Admins approve/deny. Dog booking respects experience levels.

---

### Phase 6: User Profiles & Photos (Week 8)
**Backend:**
- [ ] Profile update endpoints (name, email, phone)
- [ ] Email re-verification on email change
- [ ] Profile photo upload
- [ ] Tests for profile updates

**Frontend:**
- [ ] User: Profile page (view/edit)
- [ ] User: Photo upload with preview
- [ ] Profile photo display throughout app
- [ ] Tests for profile UI

**Deliverable:** Users can edit profiles and upload photos.

---

### Phase 7: Account Management & GDPR (Week 9)
**Backend:**
- [ ] Account deletion endpoint with GDPR anonymization
- [ ] Auto-deactivation cron job (runs daily, checks inactivity)
- [ ] Manual activation/deactivation endpoints (admin)
- [ ] Reactivation request model and endpoints
- [ ] Email notifications for deactivation/reactivation
- [ ] Update login to check is_active flag
- [ ] Tests for anonymization, deactivation, reactivation flows

**Frontend:**
- [ ] User: Account deletion button in profile with password confirmation modal
- [ ] User: Warning messages about data retention
- [ ] User: Reactivation request page (for deactivated users on login attempt)
- [ ] Admin: User list with active/inactive filter
- [ ] Admin: Activate/deactivate buttons with reason input
- [ ] Admin: Reactivation requests list with approve/deny
- [ ] Tests for account management UI

**Deliverable:** Complete GDPR-compliant account deletion, automatic inactivity deactivation, and reactivation workflow.

---

### Phase 8: Admin Dashboard & Reports (Week 10)
**Backend:**
- [ ] Dashboard stats endpoint (walks, active/inactive users, available/unavailable dogs, reactivation requests, recent activity)
- [ ] Walk history endpoint with filtering (including anonymized users)
- [ ] User list endpoint for admin (with active/inactive status)
- [ ] Tests for admin endpoints

**Frontend:**
- [ ] Admin: Dashboard with stat cards (including inactive users, unavailable dogs, reactivation requests)
- [ ] Admin: Recent activity feed (including deactivations, dog availability changes)
- [ ] Admin: User list with experience levels and active/inactive status
- [ ] Admin: Walk history with filters (display anonymized user names)
- [ ] Admin: System settings page (booking advance, cancellation notice, auto-deactivation period)
- [ ] Tests for dashboard components

**Deliverable:** Comprehensive admin dashboard with complete stats, activity feed, user management, and system settings.

---

### Phase 9: Polish & Testing (Week 11)
**Backend:**
- [ ] Achieve 90% test coverage
- [ ] Performance testing (concurrent bookings, large datasets)
- [ ] Test auto-deactivation cron job
- [ ] Test anonymization thoroughly (GDPR compliance)
- [ ] Security audit (SQL injection, XSS, CSRF)
- [ ] Error handling improvements
- [ ] API documentation (comments, README)

**Frontend:**
- [ ] Achieve 90% test coverage
- [ ] Cross-browser testing (Chrome, Firefox, Safari, Edge)
- [ ] Mobile device testing (iOS, Android)
- [ ] Accessibility audit (keyboard navigation, screen readers)
- [ ] Performance optimization (lazy loading, caching)
- [ ] UI polish (animations, loading states, error messages)
- [ ] Test all new flows (deletion, deactivation, reactivation)

**Deliverable:** Production-ready complete application with 90% test coverage, polished UI, comprehensive tests.

---

### Phase 10: Deployment (Week 12)
- [ ] Production environment setup
- [ ] Environment variables configuration
- [ ] Gmail API production credentials
- [ ] Database backups strategy
- [ ] Cron jobs setup:
  - [ ] Walk auto-completion (hourly)
  - [ ] Booking reminders (every 15 minutes)
  - [ ] Auto-deactivation (daily at 3am)
  - [ ] Database backup (daily at 2am)
- [ ] Monitoring and logging setup
- [ ] Terms & Conditions page content
- [ ] Privacy Policy page (GDPR)
- [ ] User documentation (how to use the app)
- [ ] Admin documentation (how to manage users, dogs, settings)
- [ ] Launch!

---

## Deployment Considerations

### Server Requirements
- **OS**: Linux (Ubuntu 22.04 LTS recommended)
- **Go**: 1.21+
- **SQLite**: 3.35+
- **Reverse Proxy**: nginx (for HTTPS, static files)
- **Process Manager**: systemd or supervisor
- **SSL**: Let's Encrypt (Certbot)

### Environment Variables
```bash
# App
PORT=8080
ENVIRONMENT=production

# Database
DATABASE_PATH=/var/gassigeher/data/gassigeher.db

# JWT
JWT_SECRET=<random-256-bit-secret>
JWT_EXPIRATION_HOURS=24

# Admin
ADMIN_EMAILS=admin@example.com,admin2@example.com

# Gmail API
GMAIL_CLIENT_ID=<google-oauth-client-id>
GMAIL_CLIENT_SECRET=<google-oauth-client-secret>
GMAIL_REFRESH_TOKEN=<google-oauth-refresh-token>
GMAIL_FROM_EMAIL=noreply@gassigeher.com

# Uploads
UPLOAD_DIR=/var/gassigeher/uploads
MAX_UPLOAD_SIZE_MB=5

# System Settings (defaults)
BOOKING_ADVANCE_DAYS=14
CANCELLATION_NOTICE_HOURS=12
AUTO_DEACTIVATION_DAYS=365
```

### Directory Structure
```
/var/gassigeher/
├── bin/
│   └── gassigeher          # Go binary
├── data/
│   └── gassigeher.db       # SQLite database
├── uploads/
│   ├── users/              # User photos
│   └── dogs/               # Dog photos
├── frontend/               # Static files
├── logs/
│   ├── access.log
│   └── error.log
└── config/
    └── .env                # Environment variables
```

### nginx Configuration
```nginx
server {
  listen 80;
  server_name gassigeher.com;
  return 301 https://$server_name$request_uri;
}

server {
  listen 443 ssl http2;
  server_name gassigeher.com;

  ssl_certificate /etc/letsencrypt/live/gassigeher.com/fullchain.pem;
  ssl_certificate_key /etc/letsencrypt/live/gassigeher.com/privkey.pem;

  # Static files
  location / {
    root /var/gassigeher/frontend;
    try_files $uri $uri/ /index.html;
  }

  # API
  location /api/ {
    proxy_pass http://localhost:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
  }

  # Uploads (authenticated via Go handler)
  location /uploads/ {
    proxy_pass http://localhost:8080;
  }
}
```

### Systemd Service
```ini
[Unit]
Description=Gassigeher Dog Walking App
After=network.target

[Service]
Type=simple
User=gassigeher
WorkingDirectory=/var/gassigeher
EnvironmentFile=/var/gassigeher/config/.env
ExecStart=/var/gassigeher/bin/gassigeher
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

### Backup Strategy
- **Database**: Daily automated backups via cron
  ```bash
  0 2 * * * sqlite3 /var/gassigeher/data/gassigeher.db ".backup /var/gassigeher/backups/db-$(date +\%Y\%m\%d).db"
  ```
- **Uploads**: Weekly rsync to backup server
- **Retention**: 30 days for daily DB backups, 90 days for weekly upload backups

### Monitoring
- **Logs**: Rotate with logrotate, monitor with tail/grep or Loki
- **Uptime**: UptimeRobot or similar
- **Errors**: Sentry or custom error reporting
- **Metrics**: Prometheus + Grafana (optional, for advanced monitoring)

---

## Future Enhancements (Post-Launch)

### Potential Features
1. **Push Notifications**: Browser push for booking reminders (alternative to email)
2. **SMS Notifications**: Alternative to email (Twilio integration)
3. **Multi-language Support**: Add English, French, etc. (i18n foundation already in place)
4. **Mobile Apps**: Native iOS/Android apps
5. **Dog Ratings**: Users rate dogs after walks, helps categorization
6. **User Ratings**: Admins rate user reliability for promotion decisions
7. **Walk Reports**: Users upload photos during/after walk
8. **GPS Tracking**: Optional live tracking during walks for safety
9. **Payment Integration**: Charge for premium features or donations
10. **Recurring Bookings**: Book same dog every Monday for 4 weeks
11. **Waiting List**: Join queue if dog is fully booked
12. **Social Features**: Share walk photos, comment on dogs
13. **Admin Reports**: Export CSV of walks, users, dog statistics
14. **Multiple Organizations**: Multi-tenant system for different shelters
15. **Volunteer Hours Tracking**: Gamification, badges, leaderboards
16. **Data Export**: GDPR data portability (export user data as JSON/PDF)
17. **Walk Duration Tracking**: Track actual walk duration vs. scheduled
18. **Weather Integration**: Display weather forecast for scheduled walks
19. **Calendar Integration**: Export bookings to Google Calendar, iCal
20. **In-app Messaging**: Direct messaging between users and admins

---

## Summary

This is a **complete, production-ready** implementation plan for a comprehensive dog walking booking system with:

### Core Features ✅
- **Two user roles** (Gassigeher, Admin) with distinct permissions
- **Experience-based access** (Green/Blue/Orange categories) with promotion workflow
- **Full booking system** with calendar UI, flexible scheduling, and admin controls
- **Dog availability management** with quick health status toggles
- **Comprehensive email notifications** (17 types) for all actions

### User Management & GDPR ✅
- **GDPR-compliant account deletion** with data anonymization
- **Automatic user lifecycle management** (deactivation after 1 year inactivity)
- **Reactivation workflow** with admin approval
- **Manual activation/deactivation** by admins
- **Complete audit trail** for compliance

### Technical Excellence ✅
- **Mobile-first responsive design** with dog-themed aesthetics (Tierheim Göppingen colors)
- **German UI with i18n foundation** for future translations
- **90% test coverage** for backend and frontend
- **Production-ready architecture** with security best practices
- **Automated workflows** (cron jobs for completion, reminders, deactivation)

### Admin Tools ✅
- **Comprehensive dashboard** with real-time statistics
- **User management** (activate, deactivate, promote, view history)
- **Dog management** (CRUD, availability, photos, categories)
- **Booking management** (cancel, move, view all with reasons)
- **System settings** (adjustable thresholds, limits)
- **Activity monitoring** (recent actions, pending requests)

**Total Estimated Timeline**: 12 weeks (including all features, testing, and deployment)

**Technologies**: Go, SQLite, Vanilla JavaScript, HTML5, CSS3, Gmail API, JWT, bcrypt

**Philosophy**: Simple, maintainable, user-focused, GDPR-compliant, production-ready.

Ready to start implementing a complete dog walking management system! 🐕
