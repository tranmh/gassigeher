# Booking Time Implementation Fixes Plan

## Overview

This document outlines the fixes required for the booking time restrictions feature introduced in commit `9d9d5e1`. The implementation has several critical bugs and inconsistencies that need to be addressed.

**Decision Summary (from user clarification):**
1. **Remove `walk_type` field entirely** - Use only `scheduled_time` for bookings
2. **Dynamic time slots** - Calendar and booking forms should show actual configured time slots
3. **Implement email notifications** - For approval/rejection workflows
4. **Navigation updates** - All admin pages include Booking Times + new Booking Approvals link

---

## Critical Issues Summary

| # | Issue | Severity | Effort |
|---|-------|----------|--------|
| 1 | Navigation missing from 8 admin pages | Critical | Low |
| 2 | API routing mismatch (404 errors) | Critical | Low |
| 3 | `walk_type` field needs removal | Critical | High |
| 4 | Calendar hardcoded to 2 slots | Critical | Medium |
| 5 | Booking form hardcoded to 2 slots | Critical | Medium |
| 6 | Missing approval/rejection emails | Medium | Medium |
| 7 | Missing i18n keys | Low | Low |
| 8 | UX improvements for approval workflow | Low | Low |

---

## Phase 1: Critical Navigation & API Fixes

### 1.1 Fix Admin Navigation (All 8 Pages)

**Problem:** `admin-booking-times.html` is not accessible from any other admin page.

**Files to modify:**
- `frontend/admin-dashboard.html`
- `frontend/admin-dogs.html`
- `frontend/admin-bookings.html`
- `frontend/admin-blocked-dates.html`
- `frontend/admin-experience-requests.html`
- `frontend/admin-users.html`
- `frontend/admin-reactivation-requests.html`
- `frontend/admin-settings.html`

**Change:** Add two new navigation items before "Einstellungen":

```html
<li><a href="/admin-booking-times.html" data-i18n="admin.booking_times">Buchungszeiten</a></li>
<li><a href="/admin-booking-approvals.html" data-i18n="admin.booking_approvals">Buchungs-Genehmigungen</a></li>
```

**Also update `admin-booking-times.html`** to include the new approval link.

### 1.2 Add Quick Link in Admin Dashboard

**File:** `frontend/admin-dashboard.html`

**Change:** Add to quick links section (around line 93):

```html
<a href="/admin-booking-times.html" class="btn">⏰ Buchungszeiten</a>
<a href="/admin-booking-approvals.html" class="btn">✓ Genehmigungen</a>
```

### 1.3 Fix API Routing Mismatch

**Problem:** `api.js` calls `/booking-times/rules` but `main.go` registers under `/admin/booking-times/rules`.

**File:** `frontend/js/api.js`

**Changes:**
```javascript
// Line 348 - WRONG
async getBookingTimeRules() {
    return this.request('GET', '/booking-times/rules');
}

// Should be:
async getBookingTimeRules() {
    return this.request('GET', '/admin/booking-times/rules');
}

// Similarly fix lines 351-360:
async updateBookingTimeRules(rules) {
    return this.request('PUT', '/admin/booking-times/rules', rules);
}

async createBookingTimeRule(rule) {
    return this.request('POST', '/admin/booking-times/rules', rule);
}

async deleteBookingTimeRule(id) {
    return this.request('DELETE', `/admin/booking-times/rules/${id}`);
}

// Also fix holiday endpoints (lines 369-378):
async createHoliday(holiday) {
    return this.request('POST', '/admin/holidays', holiday);
}

async updateHoliday(id, holiday) {
    return this.request('PUT', `/admin/holidays/${id}`, holiday);
}

async deleteHoliday(id) {
    return this.request('DELETE', `/admin/holidays/${id}`);
}

// Also fix approval endpoints (lines 383-392):
async getPendingApprovalBookings() {
    return this.request('GET', '/admin/bookings/pending-approvals');
}

async approveBooking(id) {
    return this.request('PUT', `/admin/bookings/${id}/approve`);
}

async rejectBooking(id, reason) {
    return this.request('PUT', `/admin/bookings/${id}/reject`, { reason });
}
```

### 1.4 Create New Admin Booking Approvals Page

**New file:** `frontend/admin-booking-approvals.html`

This page provides a dedicated view for managing booking approvals (moved from the section in admin-bookings.html for better visibility).

**Features:**
- List all pending approval bookings
- Approve/Reject buttons with reason input
- Filter by date range
- Show user and dog details
- Auto-refresh every 30 seconds

---

## Phase 2: Remove `walk_type` Field (Breaking Change)

### 2.1 Database Migration

**New file:** `internal/database/013_remove_walk_type.go`

```go
package database

func init() {
    RegisterMigration(&Migration{
        ID:          "013_remove_walk_type",
        Description: "Remove walk_type field, use scheduled_time only",
        Up: map[string]string{
            "sqlite": `
-- Remove walk_type from unique constraint and column
-- SQLite requires table recreation

CREATE TABLE IF NOT EXISTS bookings_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    dog_id INTEGER NOT NULL,
    date DATE NOT NULL,
    scheduled_time TEXT NOT NULL,
    status TEXT DEFAULT 'scheduled' CHECK(status IN ('scheduled', 'completed', 'cancelled')),
    completed_at TIMESTAMP,
    user_notes TEXT,
    admin_cancellation_reason TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    requires_approval INTEGER DEFAULT 0,
    approval_status TEXT DEFAULT 'approved',
    approved_by INTEGER,
    approved_at TIMESTAMP,
    rejection_reason TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (dog_id) REFERENCES dogs(id) ON DELETE CASCADE,
    FOREIGN KEY (approved_by) REFERENCES users(id) ON DELETE SET NULL,
    UNIQUE(dog_id, date, scheduled_time)
);

INSERT INTO bookings_new
SELECT id, user_id, dog_id, date, scheduled_time, status,
       completed_at, user_notes, admin_cancellation_reason,
       created_at, updated_at, requires_approval, approval_status,
       approved_by, approved_at, rejection_reason
FROM bookings;

DROP TABLE bookings;
ALTER TABLE bookings_new RENAME TO bookings;

CREATE INDEX IF NOT EXISTS idx_bookings_user ON bookings(user_id);
CREATE INDEX IF NOT EXISTS idx_bookings_dog ON bookings(dog_id);
CREATE INDEX IF NOT EXISTS idx_bookings_date ON bookings(date);
CREATE INDEX IF NOT EXISTS idx_bookings_status ON bookings(status);
CREATE INDEX IF NOT EXISTS idx_bookings_approval_status ON bookings(approval_status);
`,
            "mysql": `
ALTER TABLE bookings DROP INDEX unique_dog_date_walk;
ALTER TABLE bookings DROP COLUMN walk_type;
ALTER TABLE bookings ADD UNIQUE INDEX unique_dog_date_time (dog_id, date, scheduled_time);
`,
            "postgres": `
ALTER TABLE bookings DROP CONSTRAINT IF EXISTS bookings_dog_id_date_walk_type_key;
ALTER TABLE bookings DROP COLUMN IF EXISTS walk_type;
ALTER TABLE bookings ADD CONSTRAINT bookings_dog_date_time_unique UNIQUE (dog_id, date, scheduled_time);
`,
        },
    })
}
```

### 2.2 Update Model

**File:** `internal/models/booking.go`

**Changes:**
1. Remove `WalkType` field from `Booking` struct
2. Remove `WalkType` from `CreateBookingRequest`
3. Remove `WalkType` from `MoveBookingRequest`
4. Remove `WalkType` validation
5. Update `BookingFilterRequest` to remove `WalkType`

```go
// Booking struct - remove WalkType field
type Booking struct {
    ID                      int        `json:"id"`
    UserID                  int        `json:"user_id"`
    DogID                   int        `json:"dog_id"`
    Date                    string     `json:"date"`
    ScheduledTime           string     `json:"scheduled_time"`
    Status                  string     `json:"status"`
    // ... rest unchanged, but NO WalkType
}

// CreateBookingRequest - remove WalkType
type CreateBookingRequest struct {
    DogID         int    `json:"dog_id"`
    Date          string `json:"date"`
    ScheduledTime string `json:"scheduled_time"`
}

// Validate - remove walk_type validation
func (r *CreateBookingRequest) Validate() error {
    // Remove the walk_type validation block entirely
}
```

### 2.3 Update Repository

**File:** `internal/repository/booking_repository.go`

**Changes:**
1. Remove `walk_type` from all INSERT/SELECT/UPDATE queries
2. Update `CheckDoubleBooking` to check by `scheduled_time` instead of `walk_type`

```go
// CheckDoubleBooking - now checks by scheduled_time
func (r *BookingRepository) CheckDoubleBooking(dogID int, date string, scheduledTime string) (bool, error) {
    var count int
    err := r.db.QueryRow(`
        SELECT COUNT(*) FROM bookings
        WHERE dog_id = ? AND date = ? AND scheduled_time = ? AND status != 'cancelled'
    `, dogID, date, scheduledTime).Scan(&count)
    return count > 0, err
}
```

### 2.4 Update Handlers

**File:** `internal/handlers/booking_handler.go`

**Changes:**
1. Remove `WalkType` from booking creation
2. Update `MoveBooking` to not use `WalkType`
3. Update email notifications to not reference `WalkType`

### 2.5 Update Email Service

**File:** `internal/services/email_service.go`

**Changes:**
- Remove `walkType` parameter from email templates
- Update `SendBookingConfirmation`, `SendBookingCancellation`, `SendAdminCancellation`, `SendBookingMoved`
- Show only date and time in emails

---

## Phase 3: Dynamic Time Slots in Frontend

### 3.1 Update Calendar.html

**File:** `frontend/calendar.html`

**Major Changes:**

1. **Fetch time rules on load:**
```javascript
let timeRules = [];

async function loadTimeRules() {
    try {
        // Get rules for today to determine slot structure
        const today = new Date().toISOString().split('T')[0];
        timeRules = await api.getRulesForDate(today);
    } catch (error) {
        console.error('Failed to load time rules:', error);
        // Fallback to default 2 slots
        timeRules = [
            { rule_name: 'Morgen', start_time: '09:00', end_time: '12:00', is_blocked: false },
            { rule_name: 'Nachmittag', start_time: '14:00', end_time: '17:00', is_blocked: false }
        ];
    }
}
```

2. **Dynamic slot rendering in `renderCell`:**
```javascript
function renderCell(dog, date, data) {
    // Get rules for this specific date (weekday vs weekend)
    const dateRules = getActiveRulesForDate(date);

    // Build slot display dynamically
    let content = '';
    let allBooked = true;

    dateRules.forEach(rule => {
        if (rule.is_blocked) return; // Skip blocked periods

        const isBooked = data.dogBookings.some(b =>
            isTimeInSlot(b.scheduled_time, rule.start_time, rule.end_time)
        );

        if (!isBooked) allBooked = false;

        const icon = getSlotIcon(rule.rule_name);
        const statusClass = isBooked ? 'booked' : 'available';
        content += `<div class="walk-type ${statusClass}">${icon} ${rule.rule_name}</div>`;
    });

    // ... rest of cell rendering
}

function getSlotIcon(ruleName) {
    const icons = {
        'Morning Walk': '🌅',
        'Afternoon Walk': '☀️',
        'Evening Walk': '🌆'
    };
    return icons[ruleName] || '🕐';
}
```

3. **Update `quickBook` function:**
```javascript
function quickBook(dogId, date, availableSlots) {
    // availableSlots is now an array of {rule_name, start_time, end_time}
    const dog = allDogs.find(d => d.id === dogId);
    if (!dog || availableSlots.length === 0) return;

    let selectedSlot;
    if (availableSlots.length === 1) {
        selectedSlot = availableSlots[0];
    } else {
        // Show slot selection dialog
        const options = availableSlots.map((s, i) => `${i + 1}. ${s.rule_name} (${s.start_time}-${s.end_time})`).join('\n');
        const choice = prompt(`Zeitfenster wählen:\n${options}\n\nNummer eingeben:`);
        const idx = parseInt(choice) - 1;
        if (isNaN(idx) || idx < 0 || idx >= availableSlots.length) return;
        selectedSlot = availableSlots[idx];
    }

    // Navigate to dogs.html with pre-filled data
    const pendingBooking = {
        dogId: dogId,
        date: date,
        slotStartTime: selectedSlot.start_time
    };
    localStorage.setItem('pendingBooking', JSON.stringify(pendingBooking));
    window.location.href = '/dogs.html';
}
```

### 3.2 Update Dogs.html Booking Modal

**File:** `frontend/dogs.html`

**Major Changes:**

1. **Remove walk_type dropdown entirely**
2. **Load time slots dynamically:**

```javascript
async function loadAvailableTimeSlots(date) {
    if (!date) return;

    try {
        const response = await api.getAvailableTimeSlots(date);
        const slots = response.slots || [];

        const timeSelect = document.getElementById('booking-time');
        timeSelect.innerHTML = '<option value="">Bitte wählen...</option>';

        // Get rules to show slot names
        const rules = await api.getRulesForDate(date);

        // Group slots by rule
        rules.forEach(rule => {
            if (rule.is_blocked) return;

            // Add optgroup for each time period
            const group = document.createElement('optgroup');
            group.label = `${rule.rule_name} (${rule.start_time}-${rule.end_time})`;

            // Add individual time slots
            slots.forEach(slot => {
                if (isTimeInRange(slot, rule.start_time, rule.end_time)) {
                    const option = document.createElement('option');
                    option.value = slot;
                    option.textContent = slot;
                    group.appendChild(option);
                }
            });

            if (group.children.length > 0) {
                timeSelect.appendChild(group);
            }
        });

        // Show rules info
        displayTimeRulesInfo(rules);

    } catch (error) {
        console.error('Failed to load time slots:', error);
        // Fallback handled by updateTimeOptionsLegacy()
    }
}
```

3. **Update form submission:**
```javascript
async function submitBooking(event) {
    event.preventDefault();

    const dogId = parseInt(document.getElementById('booking-dog-id').value);
    const date = document.getElementById('booking-date').value;
    const time = document.getElementById('booking-time').value;

    // No walk_type anymore
    const bookingData = {
        dog_id: dogId,
        date: date,
        scheduled_time: time
    };

    try {
        const booking = await api.createBooking(bookingData);
        // ... success handling
    } catch (error) {
        // ... error handling
    }
}
```

4. **Remove walk_type from HTML:**
```html
<!-- REMOVE this entire form-group -->
<div class="form-group">
    <label for="booking-walk-type">Spaziergangszeit *</label>
    <select id="booking-walk-type" required onchange="updateTimeOptions()">
        <option value="morning">Morgen (09:00-12:00)</option>
        <option value="evening">Abend (14:00-17:00)</option>
    </select>
</div>
```

### 3.3 Update Dashboard.html

**File:** `frontend/dashboard.html`

**Changes:**
1. Remove walk_type display from bookings
2. Show scheduled_time with slot name derived from time

```javascript
// In loadUpcomingBookings():
const walkSlotName = getSlotNameForTime(booking.scheduled_time);

// Helper function
function getSlotNameForTime(time) {
    if (time >= '09:00' && time < '12:00') return '🌅 Morgen';
    if (time >= '14:00' && time < '17:00') return '☀️ Nachmittag';
    if (time >= '18:00' && time < '20:00') return '🌆 Abend';
    return '🕐 ' + time;
}
```

---

## Phase 4: Email Notifications for Approvals

### 4.1 Add Email Methods

**File:** `internal/services/email_service.go`

**Add new methods:**

```go
// SendBookingApproved sends notification when booking is approved
func (s *EmailService) SendBookingApproved(to, userName, dogName, date, scheduledTime string) error {
    subject := "Buchung genehmigt - Gassigeher"

    tmpl := `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #82b965; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f9f9f9; }
        .success { color: #28a745; font-weight: bold; }
        .details { background: white; padding: 15px; border-radius: 8px; margin: 15px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🐕 Gassigeher</h1>
        </div>
        <div class="content">
            <p>Hallo {{.UserName}},</p>
            <p class="success">✓ Ihre Buchung wurde genehmigt!</p>
            <div class="details">
                <p><strong>Hund:</strong> {{.DogName}}</p>
                <p><strong>Datum:</strong> {{.Date}}</p>
                <p><strong>Uhrzeit:</strong> {{.ScheduledTime}} Uhr</p>
            </div>
            <p>Wir freuen uns auf Ihren Besuch!</p>
            <p>Mit freundlichen Grüßen,<br>Ihr Gassigeher-Team</p>
        </div>
    </div>
</body>
</html>`

    data := struct {
        UserName      string
        DogName       string
        Date          string
        ScheduledTime string
    }{userName, dogName, date, scheduledTime}

    t := template.Must(template.New("approved").Parse(tmpl))
    var body bytes.Buffer
    if err := t.Execute(&body, data); err != nil {
        return err
    }

    return s.SendEmail(to, subject, body.String())
}

// SendBookingRejected sends notification when booking is rejected
func (s *EmailService) SendBookingRejected(to, userName, dogName, date, scheduledTime, reason string) error {
    subject := "Buchung abgelehnt - Gassigeher"

    tmpl := `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #82b965; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f9f9f9; }
        .rejected { color: #dc3545; font-weight: bold; }
        .details { background: white; padding: 15px; border-radius: 8px; margin: 15px 0; }
        .reason { background: #fff3cd; padding: 15px; border-radius: 8px; border-left: 4px solid #ffc107; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🐕 Gassigeher</h1>
        </div>
        <div class="content">
            <p>Hallo {{.UserName}},</p>
            <p class="rejected">✗ Ihre Buchung wurde leider abgelehnt.</p>
            <div class="details">
                <p><strong>Hund:</strong> {{.DogName}}</p>
                <p><strong>Datum:</strong> {{.Date}}</p>
                <p><strong>Uhrzeit:</strong> {{.ScheduledTime}} Uhr</p>
            </div>
            <div class="reason">
                <p><strong>Grund:</strong> {{.Reason}}</p>
            </div>
            <p>Sie können gerne einen anderen Termin buchen.</p>
            <p>Mit freundlichen Grüßen,<br>Ihr Gassigeher-Team</p>
        </div>
    </div>
</body>
</html>`

    data := struct {
        UserName      string
        DogName       string
        Date          string
        ScheduledTime string
        Reason        string
    }{userName, dogName, date, scheduledTime, reason}

    t := template.Must(template.New("rejected").Parse(tmpl))
    var body bytes.Buffer
    if err := t.Execute(&body, data); err != nil {
        return err
    }

    return s.SendEmail(to, subject, body.String())
}
```

### 4.2 Update Booking Handler

**File:** `internal/handlers/booking_handler.go`

**Update `ApprovePendingBooking`:**
```go
func (h *BookingHandler) ApprovePendingBooking(w http.ResponseWriter, r *http.Request) {
    // ... existing code ...

    if err := h.bookingRepo.ApproveBooking(id, adminID); err != nil {
        respondError(w, http.StatusInternalServerError, err.Error())
        return
    }

    // Get booking details for email
    booking, _ := h.bookingRepo.FindByIDWithDetails(id)
    if booking != nil && booking.User.Email != nil && h.emailService != nil {
        go h.emailService.SendBookingApproved(
            *booking.User.Email,
            booking.User.Name,
            booking.Dog.Name,
            booking.Date,
            booking.ScheduledTime,
        )
    }

    respondJSON(w, http.StatusOK, map[string]string{
        "message": "Booking approved successfully",
    })
}
```

**Update `RejectPendingBooking`:**
```go
func (h *BookingHandler) RejectPendingBooking(w http.ResponseWriter, r *http.Request) {
    // ... existing code ...

    if err := h.bookingRepo.RejectBooking(id, adminID, req.Reason); err != nil {
        respondError(w, http.StatusInternalServerError, err.Error())
        return
    }

    // Get booking details for email
    booking, _ := h.bookingRepo.FindByIDWithDetails(id)
    if booking != nil && booking.User.Email != nil && h.emailService != nil {
        go h.emailService.SendBookingRejected(
            *booking.User.Email,
            booking.User.Name,
            booking.Dog.Name,
            booking.Date,
            booking.ScheduledTime,
            req.Reason,
        )
    }

    respondJSON(w, http.StatusOK, map[string]string{
        "message": "Booking rejected successfully",
    })
}
```

---

## Phase 5: i18n Updates

### 5.1 Add Missing Translation Keys

**File:** `frontend/i18n/de.json`

**Add:**
```json
{
  "admin": {
    "booking_times": "Buchungszeiten",
    "booking_approvals": "Buchungs-Genehmigungen",
    "manage_booking_times": "Buchungszeiten verwalten",
    "manage_booking_approvals": "Buchungs-Genehmigungen verwalten"
  },
  "booking_times": {
    "title": "Buchungszeiten verwalten",
    "settings": "Einstellungen",
    "time_slots": "Zeitfenster konfigurieren",
    "weekday": "Wochentags (Mo-Fr)",
    "weekend": "Wochenende/Feiertage",
    "morning_approval": "Vormittagsspaziergänge erfordern Admin-Genehmigung",
    "morning_approval_desc": "Wenn aktiviert, müssen Buchungen zwischen 09:00 und 12:00 Uhr von einem Admin genehmigt werden.",
    "auto_holidays": "Automatische Feiertage-Erkennung (Baden-Württemberg)",
    "auto_holidays_desc": "Lädt automatisch gesetzliche Feiertage aus der feiertage-api.de. An Feiertagen gelten Wochenendregeln.",
    "save_settings": "Einstellungen speichern",
    "slot_name": "Zeitfenster",
    "from": "Von",
    "to": "Bis",
    "type": "Typ",
    "actions": "Aktionen",
    "add_slot": "Zeitfenster hinzufügen",
    "holidays_title": "Feiertage verwalten",
    "holidays_desc": "Verwalten Sie gesetzliche Feiertage und fügen Sie eigene Feiertage hinzu.",
    "year": "Jahr",
    "load": "Laden",
    "date": "Datum",
    "name": "Name",
    "source": "Quelle",
    "status": "Status",
    "add_holiday": "Feiertag hinzufügen",
    "type_allowed": "Erlaubt",
    "type_blocked": "Gesperrt"
  },
  "booking_approvals": {
    "title": "Buchungs-Genehmigungen",
    "pending": "Ausstehende Genehmigungen",
    "no_pending": "Keine ausstehenden Genehmigungen",
    "approve": "Genehmigen",
    "reject": "Ablehnen",
    "reject_reason": "Grund für Ablehnung",
    "approved": "Genehmigt",
    "rejected": "Abgelehnt",
    "pending_status": "Ausstehend"
  },
  "nav": {
    "calendar": "Kalender"
  }
}
```

### 5.2 Update HTML Files with data-i18n

**File:** `frontend/admin-booking-times.html`

Add `data-i18n` attributes to all hardcoded German text.

**File:** `frontend/calendar.html`

```html
<li><a href="/calendar.html" data-i18n="nav.calendar">Kalender</a></li>
```

---

## Phase 6: UX Improvements

### 6.1 Better Approval Status Display in Dashboard

**File:** `frontend/dashboard.html`

**Improve the pending approval message:**
```html
${booking.approval_status === 'pending' ? `
    <div class="alert alert-warning" style="margin-top: 10px; padding: 15px;">
        <strong>⏳ Warte auf Admin-Genehmigung</strong>
        <p style="margin: 10px 0 0 0; font-size: 0.9rem;">
            Ihr Termin wurde vorgemerkt. Sie erhalten eine E-Mail,
            sobald ein Administrator Ihre Anfrage bearbeitet hat.
        </p>
    </div>
` : ''}
```

### 6.2 Show Approval Status in Admin Bookings List

**File:** `frontend/admin-bookings.html`

**Add approval status badge to booking cards:**
```javascript
// In renderBookings(), add after status badge:
${booking.approval_status === 'pending' ? '<span class="badge" style="background: #ffc107; color: #000;">⏳ Genehmigung ausstehend</span>' : ''}
${booking.approval_status === 'rejected' ? '<span class="badge" style="background: #dc3545; color: #fff;">✗ Abgelehnt</span>' : ''}
```

### 6.3 Add Pending Approvals Count to Admin Dashboard

**File:** `frontend/admin-dashboard.html`

**Add a new stat card for pending approvals:**
```html
<div class="card" style="text-align: center; padding: 25px;">
    <h2 style="margin: 0 0 10px 0; font-size: 2.5rem; color: #ffc107;" id="stat-pending-approvals">-</h2>
    <p style="margin: 0; font-size: 0.9rem; color: #666;">Ausstehende Genehmigungen</p>
</div>
```

**Add to loadStats():**
```javascript
// Fetch pending approvals count
try {
    const pending = await api.getPendingApprovalBookings();
    document.getElementById('stat-pending-approvals').textContent = pending.length;
} catch (e) {
    document.getElementById('stat-pending-approvals').textContent = '-';
}
```

---

## Implementation Order

### Sprint 1: Critical Fixes (Blocking Issues)
1. ✅ Fix API routing mismatch (Phase 1.3)
2. ✅ Fix admin navigation (Phase 1.1, 1.2)
3. ✅ Create admin-booking-approvals.html (Phase 1.4)

### Sprint 2: walk_type Removal (Breaking Change)
1. ✅ Create database migration (Phase 2.1)
2. ✅ Update model (Phase 2.2)
3. ✅ Update repository (Phase 2.3)
4. ✅ Update handlers (Phase 2.4)
5. ✅ Update email service (Phase 2.5)

### Sprint 3: Frontend Dynamic Slots
1. ✅ Update calendar.html (Phase 3.1)
2. ✅ Update dogs.html booking modal (Phase 3.2)
3. ✅ Update dashboard.html (Phase 3.3)

### Sprint 4: Email & Polish
1. ✅ Add approval email methods (Phase 4.1)
2. ✅ Update booking handler with emails (Phase 4.2)
3. ✅ Add i18n keys (Phase 5.1, 5.2)
4. ✅ UX improvements (Phase 6)

---

## Testing Checklist

### Navigation Tests
- [ ] All 9 admin pages show Booking Times link
- [ ] All 9 admin pages show Booking Approvals link
- [ ] Quick links in admin dashboard include both new links
- [ ] Navigation works correctly on mobile

### API Tests
- [ ] GET /admin/booking-times/rules returns 200
- [ ] POST /admin/booking-times/rules creates rule
- [ ] PUT /admin/booking-times/rules updates rules
- [ ] DELETE /admin/booking-times/rules/{id} deletes rule
- [ ] GET /admin/bookings/pending-approvals returns pending bookings
- [ ] PUT /admin/bookings/{id}/approve approves booking
- [ ] PUT /admin/bookings/{id}/reject rejects booking with reason

### Booking Flow Tests (after walk_type removal)
- [ ] Create booking with only scheduled_time works
- [ ] Double-booking prevented by scheduled_time uniqueness
- [ ] Calendar shows dynamic time slots
- [ ] Booking form shows grouped time options
- [ ] Pending approval shown on dashboard

### Email Tests
- [ ] Approval email sent when booking approved
- [ ] Rejection email sent with reason when rejected
- [ ] Booking confirmation email works without walk_type

### Migration Tests
- [ ] SQLite migration runs without errors
- [ ] MySQL migration runs without errors
- [ ] PostgreSQL migration runs without errors
- [ ] Existing bookings preserved after migration

---

## Rollback Plan

If issues arise after deployment:

1. **Database:** Keep backup before migration 013
2. **Frontend:** Revert to commit before changes
3. **API:** The walk_type removal is breaking - requires coordinated rollback of:
   - Database migration (restore from backup)
   - Backend code
   - Frontend code

---

## Notes

- The removal of `walk_type` is a **breaking change** that requires careful coordination
- All existing bookings will retain their `scheduled_time` values
- The `walk_type` data will be lost (but can be derived from `scheduled_time` if needed)
- Consider adding a "slot_name" computed field in responses for display purposes
