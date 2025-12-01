# Booking Time Restrictions Implementation Plan

**Version:** 1.0
**Date:** 2025-01-23
**Status:** Planning Phase

---

## Executive Summary

This document outlines the implementation of booking time restrictions for the Gassigeher dog walking system, enforcing specific time windows and blocked periods based on day type (weekday, weekend, public holiday).

**Key Features:**
- ✅ Configurable time windows per day type
- ✅ Automatic Baden-Württemberg holiday detection via API
- ✅ Admin-configurable blocked times (feeding periods)
- ✅ Morning walk approval system (configurable)
- ✅ 15-minute booking time granularity
- ✅ Real-time time slot availability

---

## Requirements

### Time Windows

#### Monday to Friday (Weekdays)
```
09:00 - 12:00   Morning walk (nach Absprache - may require approval)
12:00 - 13:00   [Open]
13:00 - 14:00   [BLOCKED - General lunch period]
14:00 - 16:30   Afternoon walk
16:30 - 18:00   [BLOCKED - Feeding time]
18:00 - 19:30   Evening walk
```

#### Saturday, Sunday, Public Holidays (Weekends/Holidays)
```
09:00 - 12:00   Morning walk (nach Absprache - may require approval)
12:00 - 13:00   [BLOCKED - Feeding time]
13:00 - 14:00   [BLOCKED - General lunch period]
14:00 - 17:00   Afternoon/Evening walk (combined)
```

### Holiday Detection

**Public Holidays (Feiertage):**
- Auto-detect Baden-Württemberg public holidays using https://feiertage-api.de/
- Cache API results to minimize external calls
- Admin override: Add custom dates as holidays
- Admin override: Remove specific holidays if needed

**Holiday API Integration:**
```
GET https://feiertage-api.de/api/?jahr=2025&nur_land=BW
```

Response example:
```json
{
  "Neujahrstag": {
    "datum": "2025-01-01",
    "hinweis": ""
  },
  "HeiligeDreiKönige": {
    "datum": "2025-01-06",
    "hinweis": ""
  },
  ...
}
```

### Configurable Settings

1. **Morning Walk Approval** (`morning_walk_requires_approval`)
   - `true`: Morning bookings require admin approval before walk
   - `false`: Morning bookings are auto-confirmed

2. **Feiertage API** (`use_feiertage_api`)
   - `true`: Auto-fetch holidays from API
   - `false`: Use only custom admin-defined holidays

3. **Feiertage State** (`feiertage_state`)
   - Default: `"BW"` (Baden-Württemberg)
   - Future: Support other German states

4. **Time Granularity** (hardcoded initially)
   - 15-minute intervals (14:00, 14:15, 14:30, 14:45, ...)

---

## Architecture

### Database Schema Changes

#### 1. New Table: `booking_time_rules`

Stores configurable time windows per day type.

```sql
CREATE TABLE booking_time_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    day_type TEXT NOT NULL, -- 'weekday', 'weekend', 'holiday'
    rule_name TEXT NOT NULL, -- e.g., 'Afternoon Walk', 'Feeding Block'
    start_time TEXT NOT NULL, -- HH:MM format
    end_time TEXT NOT NULL,   -- HH:MM format
    is_blocked INTEGER NOT NULL DEFAULT 0, -- 0 = allowed, 1 = blocked
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(day_type, rule_name)
);
```

**Seed Data:**
```sql
-- Weekday rules
INSERT INTO booking_time_rules (day_type, rule_name, start_time, end_time, is_blocked) VALUES
('weekday', 'Morning Walk', '09:00', '12:00', 0),
('weekday', 'Lunch Block', '13:00', '14:00', 1),
('weekday', 'Afternoon Walk', '14:00', '16:30', 0),
('weekday', 'Feeding Block', '16:30', '18:00', 1),
('weekday', 'Evening Walk', '18:00', '19:30', 0);

-- Weekend/Holiday rules
INSERT INTO booking_time_rules (day_type, rule_name, start_time, end_time, is_blocked) VALUES
('weekend', 'Morning Walk', '09:00', '12:00', 0),
('weekend', 'Feeding Block', '12:00', '13:00', 1),
('weekend', 'Lunch Block', '13:00', '14:00', 1),
('weekend', 'Afternoon Walk', '14:00', '17:00', 0);
```

**Notes:**
- `day_type = 'holiday'` uses same rules as `'weekend'` initially
- Admins can modify times via settings page
- Multiple non-overlapping rules per day type allowed

#### 2. New Table: `custom_holidays`

Allows admins to add/remove specific holiday dates.

```sql
CREATE TABLE custom_holidays (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date TEXT NOT NULL UNIQUE, -- YYYY-MM-DD format
    name TEXT NOT NULL,
    is_active INTEGER NOT NULL DEFAULT 1, -- 0 = disabled, 1 = enabled
    source TEXT NOT NULL, -- 'api' or 'admin'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by INTEGER, -- admin user_id who added it
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_custom_holidays_date ON custom_holidays(date);
CREATE INDEX idx_custom_holidays_active ON custom_holidays(is_active);
```

**Example Data:**
```sql
INSERT INTO custom_holidays (date, name, source) VALUES
('2025-01-01', 'Neujahrstag', 'api'),
('2025-01-06', 'Heilige Drei Könige', 'api'),
('2025-12-25', 'Weihnachten (Sondertag)', 'admin');
```

#### 3. New Table: `feiertage_cache`

Caches API responses to minimize external calls.

```sql
CREATE TABLE feiertage_cache (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    year INTEGER NOT NULL UNIQUE,
    state TEXT NOT NULL, -- 'BW'
    data TEXT NOT NULL, -- JSON string of holidays
    fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL -- 7 days from fetch
);
```

**Cache Strategy:**
- Cache holidays for each year
- Refresh if older than 7 days
- On startup, pre-fetch current + next year

#### 4. Update Table: `bookings`

Add approval workflow fields for morning walks.

```sql
ALTER TABLE bookings ADD COLUMN requires_approval INTEGER DEFAULT 0;
ALTER TABLE bookings ADD COLUMN approval_status TEXT DEFAULT 'approved'; -- 'pending', 'approved', 'rejected'
ALTER TABLE bookings ADD COLUMN approved_by INTEGER;
ALTER TABLE bookings ADD COLUMN approved_at TIMESTAMP;
ALTER TABLE bookings ADD COLUMN rejection_reason TEXT;

CREATE INDEX idx_bookings_approval_status ON bookings(approval_status);

-- Add foreign key
-- Note: SQLite doesn't support adding FK constraints to existing tables
-- So this would be handled in migration by recreating the table
```

**approval_status values:**
- `'approved'`: Default for non-morning walks or when approval not required
- `'pending'`: Morning walk awaiting admin approval
- `'rejected'`: Morning walk denied by admin

#### 5. Update Table: `system_settings`

Add new settings for booking time features.

```sql
INSERT INTO system_settings (key, value, description) VALUES
('morning_walk_requires_approval', 'true', 'Whether morning walks (9-12) require admin approval'),
('use_feiertage_api', 'true', 'Auto-fetch Baden-Württemberg holidays from API'),
('feiertage_state', 'BW', 'German state code for holiday fetching'),
('booking_time_granularity', '15', 'Time slot intervals in minutes'),
('feiertage_cache_days', '7', 'Days to cache holiday API responses');
```

---

## Backend Implementation

### Phase 1: Database & Models

#### Step 1.1: Create Migration

**File:** `internal/database/012_booking_times.go`

```go
package database

func init() {
    RegisterMigration(&Migration{
        ID: "012_booking_times",
        Up: map[string]string{
            "sqlite": `
                -- Create booking_time_rules table
                CREATE TABLE IF NOT EXISTS booking_time_rules (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    day_type TEXT NOT NULL,
                    rule_name TEXT NOT NULL,
                    start_time TEXT NOT NULL,
                    end_time TEXT NOT NULL,
                    is_blocked INTEGER NOT NULL DEFAULT 0,
                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                    UNIQUE(day_type, rule_name)
                );

                -- Create custom_holidays table
                CREATE TABLE IF NOT EXISTS custom_holidays (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    date TEXT NOT NULL UNIQUE,
                    name TEXT NOT NULL,
                    is_active INTEGER NOT NULL DEFAULT 1,
                    source TEXT NOT NULL,
                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                    created_by INTEGER,
                    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
                );
                CREATE INDEX IF NOT EXISTS idx_custom_holidays_date ON custom_holidays(date);
                CREATE INDEX IF NOT EXISTS idx_custom_holidays_active ON custom_holidays(is_active);

                -- Create feiertage_cache table
                CREATE TABLE IF NOT EXISTS feiertage_cache (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    year INTEGER NOT NULL UNIQUE,
                    state TEXT NOT NULL,
                    data TEXT NOT NULL,
                    fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                    expires_at TIMESTAMP NOT NULL
                );

                -- Add approval columns to bookings (recreate table for SQLite)
                CREATE TABLE IF NOT EXISTS bookings_new (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    user_id INTEGER NOT NULL,
                    dog_id INTEGER NOT NULL,
                    date TEXT NOT NULL,
                    scheduled_time TEXT NOT NULL,
                    walk_type TEXT NOT NULL,
                    status TEXT NOT NULL,
                    notes TEXT,
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
                    UNIQUE(dog_id, date, walk_type)
                );

                INSERT INTO bookings_new SELECT
                    id, user_id, dog_id, date, scheduled_time, walk_type, status,
                    notes, created_at, updated_at,
                    0, 'approved', NULL, NULL, NULL
                FROM bookings;

                DROP TABLE bookings;
                ALTER TABLE bookings_new RENAME TO bookings;

                CREATE INDEX IF NOT EXISTS idx_bookings_user ON bookings(user_id);
                CREATE INDEX IF NOT EXISTS idx_bookings_dog ON bookings(dog_id);
                CREATE INDEX IF NOT EXISTS idx_bookings_date ON bookings(date);
                CREATE INDEX IF NOT EXISTS idx_bookings_status ON bookings(status);
                CREATE INDEX IF NOT EXISTS idx_bookings_approval_status ON bookings(approval_status);

                -- Seed default time rules
                INSERT OR IGNORE INTO booking_time_rules (day_type, rule_name, start_time, end_time, is_blocked) VALUES
                ('weekday', 'Morning Walk', '09:00', '12:00', 0),
                ('weekday', 'Lunch Block', '13:00', '14:00', 1),
                ('weekday', 'Afternoon Walk', '14:00', '16:30', 0),
                ('weekday', 'Feeding Block', '16:30', '18:00', 1),
                ('weekday', 'Evening Walk', '18:00', '19:30', 0),
                ('weekend', 'Morning Walk', '09:00', '12:00', 0),
                ('weekend', 'Feeding Block', '12:00', '13:00', 1),
                ('weekend', 'Lunch Block', '13:00', '14:00', 1),
                ('weekend', 'Afternoon Walk', '14:00', '17:00', 0);

                -- Add new system settings
                INSERT OR IGNORE INTO system_settings (key, value, description) VALUES
                ('morning_walk_requires_approval', 'true', 'Whether morning walks require admin approval'),
                ('use_feiertage_api', 'true', 'Auto-fetch Baden-Württemberg holidays from API'),
                ('feiertage_state', 'BW', 'German state code for holiday fetching'),
                ('booking_time_granularity', '15', 'Time slot intervals in minutes'),
                ('feiertage_cache_days', '7', 'Days to cache holiday API responses');
            `,
            "mysql": `
                -- Similar structure with MySQL syntax (AUTO_INCREMENT, BOOLEAN, etc.)
            `,
            "postgres": `
                -- Similar structure with PostgreSQL syntax (SERIAL, BOOLEAN, etc.)
            `,
        },
    })
}
```

#### Step 1.2: Create Models

**File:** `internal/models/booking_time_rule.go`

```go
package models

import "time"

type BookingTimeRule struct {
    ID        int       `json:"id"`
    DayType   string    `json:"day_type"`   // 'weekday', 'weekend', 'holiday'
    RuleName  string    `json:"rule_name"`
    StartTime string    `json:"start_time"` // HH:MM format
    EndTime   string    `json:"end_time"`   // HH:MM format
    IsBlocked bool      `json:"is_blocked"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// Validate validates booking time rule
func (r *BookingTimeRule) Validate() error {
    if r.DayType != "weekday" && r.DayType != "weekend" && r.DayType != "holiday" {
        return fmt.Errorf("day_type must be 'weekday', 'weekend', or 'holiday'")
    }
    if r.RuleName == "" {
        return fmt.Errorf("rule_name is required")
    }

    // Validate time format
    if !isValidTimeFormat(r.StartTime) {
        return fmt.Errorf("start_time must be in HH:MM format")
    }
    if !isValidTimeFormat(r.EndTime) {
        return fmt.Errorf("end_time must be in HH:MM format")
    }

    // Validate end > start
    if r.EndTime <= r.StartTime {
        return fmt.Errorf("end_time must be after start_time")
    }

    return nil
}

func isValidTimeFormat(t string) bool {
    _, err := time.Parse("15:04", t)
    return err == nil
}
```

**File:** `internal/models/custom_holiday.go`

```go
package models

import "time"

type CustomHoliday struct {
    ID        int       `json:"id"`
    Date      string    `json:"date"` // YYYY-MM-DD
    Name      string    `json:"name"`
    IsActive  bool      `json:"is_active"`
    Source    string    `json:"source"` // 'api' or 'admin'
    CreatedAt time.Time `json:"created_at"`
    CreatedBy *int      `json:"created_by,omitempty"` // Admin user ID
}

func (h *CustomHoliday) Validate() error {
    if h.Date == "" {
        return fmt.Errorf("date is required")
    }

    // Validate date format
    _, err := time.Parse("2006-01-02", h.Date)
    if err != nil {
        return fmt.Errorf("date must be in YYYY-MM-DD format")
    }

    if h.Name == "" {
        return fmt.Errorf("name is required")
    }

    if h.Source != "api" && h.Source != "admin" {
        return fmt.Errorf("source must be 'api' or 'admin'")
    }

    return nil
}
```

**File:** Update `internal/models/booking.go`

```go
// Add new fields to Booking struct
type Booking struct {
    // ... existing fields ...
    RequiresApproval bool      `json:"requires_approval"`
    ApprovalStatus   string    `json:"approval_status"` // 'pending', 'approved', 'rejected'
    ApprovedBy       *int      `json:"approved_by,omitempty"`
    ApprovedAt       *time.Time `json:"approved_at,omitempty"`
    RejectionReason  *string   `json:"rejection_reason,omitempty"`
}
```

// DONE

---

### Phase 2: Repositories

#### Step 2.1: BookingTimeRepository

**File:** `internal/repository/booking_time_repository.go`

```go
package repository

import (
    "database/sql"
    "gassigeher/internal/models"
)

type BookingTimeRepository struct {
    db *sql.DB
}

func NewBookingTimeRepository(db *sql.DB) *BookingTimeRepository {
    return &BookingTimeRepository{db: db}
}

// GetRulesByDayType returns all rules for a specific day type
func (r *BookingTimeRepository) GetRulesByDayType(dayType string) ([]models.BookingTimeRule, error) {
    query := `
        SELECT id, day_type, rule_name, start_time, end_time, is_blocked, created_at, updated_at
        FROM booking_time_rules
        WHERE day_type = ?
        ORDER BY start_time ASC
    `

    rows, err := r.db.Query(query, dayType)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var rules []models.BookingTimeRule
    for rows.Next() {
        var rule models.BookingTimeRule
        var isBlocked int

        err := rows.Scan(
            &rule.ID, &rule.DayType, &rule.RuleName,
            &rule.StartTime, &rule.EndTime, &isBlocked,
            &rule.CreatedAt, &rule.UpdatedAt,
        )
        if err != nil {
            return nil, err
        }

        rule.IsBlocked = isBlocked == 1
        rules = append(rules, rule)
    }

    return rules, nil
}

// GetAllRules returns all time rules grouped by day type
func (r *BookingTimeRepository) GetAllRules() (map[string][]models.BookingTimeRule, error) {
    query := `
        SELECT id, day_type, rule_name, start_time, end_time, is_blocked, created_at, updated_at
        FROM booking_time_rules
        ORDER BY day_type, start_time ASC
    `

    rows, err := r.db.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    result := make(map[string][]models.BookingTimeRule)

    for rows.Next() {
        var rule models.BookingTimeRule
        var isBlocked int

        err := rows.Scan(
            &rule.ID, &rule.DayType, &rule.RuleName,
            &rule.StartTime, &rule.EndTime, &isBlocked,
            &rule.CreatedAt, &rule.UpdatedAt,
        )
        if err != nil {
            return nil, err
        }

        rule.IsBlocked = isBlocked == 1
        result[rule.DayType] = append(result[rule.DayType], rule)
    }

    return result, nil
}

// UpdateRule updates a time rule
func (r *BookingTimeRepository) UpdateRule(id int, rule *models.BookingTimeRule) error {
    query := `
        UPDATE booking_time_rules
        SET start_time = ?, end_time = ?, is_blocked = ?, updated_at = ?
        WHERE id = ?
    `

    isBlocked := 0
    if rule.IsBlocked {
        isBlocked = 1
    }

    _, err := r.db.Exec(query, rule.StartTime, rule.EndTime, isBlocked, time.Now(), id)
    return err
}

// CreateRule creates a new time rule
func (r *BookingTimeRepository) CreateRule(rule *models.BookingTimeRule) error {
    query := `
        INSERT INTO booking_time_rules (day_type, rule_name, start_time, end_time, is_blocked)
        VALUES (?, ?, ?, ?, ?)
    `

    isBlocked := 0
    if rule.IsBlocked {
        isBlocked = 1
    }

    result, err := r.db.Exec(query, rule.DayType, rule.RuleName, rule.StartTime, rule.EndTime, isBlocked)
    if err != nil {
        return err
    }

    id, _ := result.LastInsertId()
    rule.ID = int(id)
    return nil
}

// DeleteRule deletes a time rule
func (r *BookingTimeRepository) DeleteRule(id int) error {
    query := `DELETE FROM booking_time_rules WHERE id = ?`
    _, err := r.db.Exec(query, id)
    return err
}
```

#### Step 2.2: HolidayRepository

**File:** `internal/repository/holiday_repository.go`

```go
package repository

import (
    "database/sql"
    "gassigeher/internal/models"
    "time"
)

type HolidayRepository struct {
    db *sql.DB
}

func NewHolidayRepository(db *sql.DB) *HolidayRepository {
    return &HolidayRepository{db: db}
}

// GetHolidaysByYear returns all active holidays for a specific year
func (r *HolidayRepository) GetHolidaysByYear(year int) ([]models.CustomHoliday, error) {
    query := `
        SELECT id, date, name, is_active, source, created_at, created_by
        FROM custom_holidays
        WHERE is_active = 1
          AND date LIKE ?
        ORDER BY date ASC
    `

    yearPrefix := fmt.Sprintf("%d-%%", year)
    rows, err := r.db.Query(query, yearPrefix)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    return r.scanHolidays(rows)
}

// IsHoliday checks if a specific date is a holiday
func (r *HolidayRepository) IsHoliday(date string) (bool, error) {
    query := `
        SELECT COUNT(*) FROM custom_holidays
        WHERE date = ? AND is_active = 1
    `

    var count int
    err := r.db.QueryRow(query, date).Scan(&count)
    if err != nil {
        return false, err
    }

    return count > 0, nil
}

// CreateHoliday adds a custom holiday
func (r *HolidayRepository) CreateHoliday(holiday *models.CustomHoliday) error {
    query := `
        INSERT INTO custom_holidays (date, name, is_active, source, created_by)
        VALUES (?, ?, ?, ?, ?)
    `

    isActive := 1
    if !holiday.IsActive {
        isActive = 0
    }

    result, err := r.db.Exec(query, holiday.Date, holiday.Name, isActive, holiday.Source, holiday.CreatedBy)
    if err != nil {
        return err
    }

    id, _ := result.LastInsertId()
    holiday.ID = int(id)
    return nil
}

// UpdateHoliday updates a holiday
func (r *HolidayRepository) UpdateHoliday(id int, holiday *models.CustomHoliday) error {
    query := `
        UPDATE custom_holidays
        SET name = ?, is_active = ?
        WHERE id = ?
    `

    isActive := 1
    if !holiday.IsActive {
        isActive = 0
    }

    _, err := r.db.Exec(query, holiday.Name, isActive, id)
    return err
}

// DeleteHoliday deletes a holiday
func (r *HolidayRepository) DeleteHoliday(id int) error {
    query := `DELETE FROM custom_holidays WHERE id = ?`
    _, err := r.db.Exec(query, id)
    return err
}

// GetCachedHolidays retrieves cached holidays from API
func (r *HolidayRepository) GetCachedHolidays(year int, state string) (string, error) {
    query := `
        SELECT data FROM feiertage_cache
        WHERE year = ? AND state = ? AND expires_at > ?
    `

    var data string
    err := r.db.QueryRow(query, year, state, time.Now()).Scan(&data)
    if err == sql.ErrNoRows {
        return "", nil // Cache miss
    }
    if err != nil {
        return "", err
    }

    return data, nil
}

// SetCachedHolidays stores API response in cache
func (r *HolidayRepository) SetCachedHolidays(year int, state string, data string, cacheDays int) error {
    expiresAt := time.Now().AddDate(0, 0, cacheDays)

    query := `
        INSERT OR REPLACE INTO feiertage_cache (year, state, data, fetched_at, expires_at)
        VALUES (?, ?, ?, ?, ?)
    `

    _, err := r.db.Exec(query, year, state, data, time.Now(), expiresAt)
    return err
}

// scanHolidays helper to scan holiday rows
func (r *HolidayRepository) scanHolidays(rows *sql.Rows) ([]models.CustomHoliday, error) {
    var holidays []models.CustomHoliday

    for rows.Next() {
        var h models.CustomHoliday
        var isActive int
        var createdBy sql.NullInt64

        err := rows.Scan(&h.ID, &h.Date, &h.Name, &isActive, &h.Source, &h.CreatedAt, &createdBy)
        if err != nil {
            return nil, err
        }

        h.IsActive = isActive == 1
        if createdBy.Valid {
            id := int(createdBy.Int64)
            h.CreatedBy = &id
        }

        holidays = append(holidays, h)
    }

    return holidays, nil
}
```

#### Step 2.3: Update BookingRepository

**File:** Update `internal/repository/booking_repository.go`

```go
// Add new methods to BookingRepository

// GetPendingApprovalBookings returns all bookings awaiting approval
func (r *BookingRepository) GetPendingApprovalBookings() ([]models.Booking, error) {
    query := `
        SELECT b.id, b.user_id, b.dog_id, b.date, b.scheduled_time, b.walk_type,
               b.status, b.notes, b.created_at, b.updated_at,
               b.requires_approval, b.approval_status, b.approved_by, b.approved_at, b.rejection_reason,
               u.name as user_name, d.name as dog_name
        FROM bookings b
        JOIN users u ON b.user_id = u.id
        JOIN dogs d ON b.dog_id = d.id
        WHERE b.approval_status = 'pending'
        ORDER BY b.date ASC, b.scheduled_time ASC
    `

    rows, err := r.db.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    return r.scanBookingsWithNames(rows)
}

// ApproveBooking approves a pending booking
func (r *BookingRepository) ApproveBooking(bookingID int, adminID int) error {
    query := `
        UPDATE bookings
        SET approval_status = 'approved', approved_by = ?, approved_at = ?
        WHERE id = ? AND approval_status = 'pending'
    `

    result, err := r.db.Exec(query, adminID, time.Now(), bookingID)
    if err != nil {
        return err
    }

    rows, _ := result.RowsAffected()
    if rows == 0 {
        return fmt.Errorf("booking not found or not pending")
    }

    return nil
}

// RejectBooking rejects a pending booking
func (r *BookingRepository) RejectBooking(bookingID int, adminID int, reason string) error {
    query := `
        UPDATE bookings
        SET approval_status = 'rejected', approved_by = ?, approved_at = ?, rejection_reason = ?, status = 'cancelled'
        WHERE id = ? AND approval_status = 'pending'
    `

    result, err := r.db.Exec(query, adminID, time.Now(), reason, bookingID)
    if err != nil {
        return err
    }

    rows, _ := result.RowsAffected()
    if rows == 0 {
        return fmt.Errorf("booking not found or not pending")
    }

    return nil
}
```

// DONE

---

### Phase 3: Services

#### Step 3.1: HolidayService

**File:** `internal/services/holiday_service.go`

```go
package services

import (
    "encoding/json"
    "fmt"
    "gassigeher/internal/models"
    "gassigeher/internal/repository"
    "io"
    "net/http"
    "strconv"
    "time"
)

type HolidayService struct {
    holidayRepo *repository.HolidayRepository
    settingsRepo *repository.SettingsRepository
}

func NewHolidayService(holidayRepo *repository.HolidayRepository, settingsRepo *repository.SettingsRepository) *HolidayService {
    return &HolidayService{
        holidayRepo: holidayRepo,
        settingsRepo: settingsRepo,
    }
}

// FetchAndCacheHolidays fetches holidays from API and stores in DB
func (s *HolidayService) FetchAndCacheHolidays(year int) error {
    // Get state from settings
    state, err := s.settingsRepo.GetSetting("feiertage_state")
    if err != nil || state == "" {
        state = "BW" // Default
    }

    // Check cache first
    cached, err := s.holidayRepo.GetCachedHolidays(year, state)
    if err == nil && cached != "" {
        // Cache hit - populate custom_holidays table
        return s.populateHolidaysFromCache(cached, year)
    }

    // Cache miss - fetch from API
    url := fmt.Sprintf("https://feiertage-api.de/api/?jahr=%d&nur_land=%s", year, state)

    resp, err := http.Get(url)
    if err != nil {
        return fmt.Errorf("failed to fetch holidays: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("holiday API returned status %d", resp.StatusCode)
    }

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return fmt.Errorf("failed to read API response: %w", err)
    }

    // Parse response
    var holidays map[string]struct {
        Datum   string `json:"datum"`
        Hinweis string `json:"hinweis"`
    }

    if err := json.Unmarshal(body, &holidays); err != nil {
        return fmt.Errorf("failed to parse holidays: %w", err)
    }

    // Cache response
    cacheDaysStr, _ := s.settingsRepo.GetSetting("feiertage_cache_days")
    cacheDays, _ := strconv.Atoi(cacheDaysStr)
    if cacheDays == 0 {
        cacheDays = 7 // Default
    }

    if err := s.holidayRepo.SetCachedHolidays(year, state, string(body), cacheDays); err != nil {
        // Log error but continue
        fmt.Printf("Warning: Failed to cache holidays: %v\n", err)
    }

    // Insert holidays into custom_holidays table
    for name, holiday := range holidays {
        h := &models.CustomHoliday{
            Date:     holiday.Datum,
            Name:     name,
            IsActive: true,
            Source:   "api",
        }

        // Insert or ignore if already exists
        _ = s.holidayRepo.CreateHoliday(h)
    }

    return nil
}

// IsHoliday checks if a date is a holiday
func (s *HolidayService) IsHoliday(date string) (bool, error) {
    // Check if API usage is enabled
    useAPI, err := s.settingsRepo.GetSetting("use_feiertage_api")
    if err == nil && useAPI == "true" {
        // Ensure holidays are cached for this year
        dateObj, _ := time.Parse("2006-01-02", date)
        year := dateObj.Year()
        _ = s.FetchAndCacheHolidays(year)
    }

    // Check database
    return s.holidayRepo.IsHoliday(date)
}

// GetHolidaysForYear returns all holidays in a year
func (s *HolidayService) GetHolidaysForYear(year int) ([]models.CustomHoliday, error) {
    // Fetch and cache if API enabled
    useAPI, err := s.settingsRepo.GetSetting("use_feiertage_api")
    if err == nil && useAPI == "true" {
        _ = s.FetchAndCacheHolidays(year)
    }

    return s.holidayRepo.GetHolidaysByYear(year)
}

// populateHolidaysFromCache helper
func (s *HolidayService) populateHolidaysFromCache(cached string, year int) error {
    var holidays map[string]struct {
        Datum string `json:"datum"`
    }

    if err := json.Unmarshal([]byte(cached), &holidays); err != nil {
        return err
    }

    for name, holiday := range holidays {
        h := &models.CustomHoliday{
            Date:     holiday.Datum,
            Name:     name,
            IsActive: true,
            Source:   "api",
        }
        _ = s.holidayRepo.CreateHoliday(h)
    }

    return nil
}
```

#### Step 3.2: BookingTimeService

**File:** `internal/services/booking_time_service.go`

```go
package services

import (
    "fmt"
    "gassigeher/internal/models"
    "gassigeher/internal/repository"
    "time"
)

type BookingTimeService struct {
    bookingTimeRepo *repository.BookingTimeRepository
    holidayService  *HolidayService
    settingsRepo    *repository.SettingsRepository
}

func NewBookingTimeService(
    bookingTimeRepo *repository.BookingTimeRepository,
    holidayService *HolidayService,
    settingsRepo *repository.SettingsRepository,
) *BookingTimeService {
    return &BookingTimeService{
        bookingTimeRepo: bookingTimeRepo,
        holidayService:  holidayService,
        settingsRepo:    settingsRepo,
    }
}

// ValidateBookingTime validates if a time slot is allowed
func (s *BookingTimeService) ValidateBookingTime(date string, scheduledTime string) error {
    // Parse date
    dateObj, err := time.Parse("2006-01-02", date)
    if err != nil {
        return fmt.Errorf("invalid date format")
    }

    // Parse time
    timeObj, err := time.Parse("15:04", scheduledTime)
    if err != nil {
        return fmt.Errorf("invalid time format")
    }

    // Determine day type
    dayType, err := s.getDayType(date, dateObj)
    if err != nil {
        return err
    }

    // Get rules for day type
    rules, err := s.bookingTimeRepo.GetRulesByDayType(dayType)
    if err != nil {
        return fmt.Errorf("failed to load time rules: %w", err)
    }

    // Check if time falls within any allowed window
    inAllowedWindow := false
    inBlockedWindow := false

    for _, rule := range rules {
        startTime, _ := time.Parse("15:04", rule.StartTime)
        endTime, _ := time.Parse("15:04", rule.EndTime)

        // Check if time is within this rule's window
        if !timeObj.Before(startTime) && timeObj.Before(endTime) {
            if rule.IsBlocked {
                inBlockedWindow = true
                return fmt.Errorf("Zeit ist gesperrt: %s (%s-%s)", rule.RuleName, rule.StartTime, rule.EndTime)
            } else {
                inAllowedWindow = true
            }
        }
    }

    if !inAllowedWindow {
        return fmt.Errorf("Zeit ist außerhalb der erlaubten Buchungszeiten")
    }

    if inBlockedWindow {
        return fmt.Errorf("Zeit fällt in eine Sperrzeit")
    }

    return nil
}

// GetAvailableTimeSlots returns all available time slots for a date
func (s *BookingTimeService) GetAvailableTimeSlots(date string) ([]string, error) {
    // Parse date
    dateObj, err := time.Parse("2006-01-02", date)
    if err != nil {
        return nil, fmt.Errorf("invalid date format")
    }

    // Determine day type
    dayType, err := s.getDayType(date, dateObj)
    if err != nil {
        return nil, err
    }

    // Get rules
    rules, err := s.bookingTimeRepo.GetRulesByDayType(dayType)
    if err != nil {
        return nil, err
    }

    // Get granularity
    granularityStr, _ := s.settingsRepo.GetSetting("booking_time_granularity")
    granularity := 15 // Default
    if g, err := strconv.Atoi(granularityStr); err == nil {
        granularity = g
    }

    // Generate time slots
    var slots []string

    for _, rule := range rules {
        if rule.IsBlocked {
            continue // Skip blocked windows
        }

        startTime, _ := time.Parse("15:04", rule.StartTime)
        endTime, _ := time.Parse("15:04", rule.EndTime)

        // Generate slots in granularity intervals
        current := startTime
        for current.Before(endTime) {
            slots = append(slots, current.Format("15:04"))
            current = current.Add(time.Duration(granularity) * time.Minute)
        }
    }

    return slots, nil
}

// RequiresApproval checks if a booking requires admin approval
func (s *BookingTimeService) RequiresApproval(scheduledTime string) (bool, error) {
    // Check setting
    requiresApprovalStr, err := s.settingsRepo.GetSetting("morning_walk_requires_approval")
    if err != nil || requiresApprovalStr != "true" {
        return false, nil // Setting disabled
    }

    // Parse time
    timeObj, err := time.Parse("15:04", scheduledTime)
    if err != nil {
        return false, err
    }

    // Morning window: 09:00 - 12:00
    morningStart, _ := time.Parse("15:04", "09:00")
    morningEnd, _ := time.Parse("15:04", "12:00")

    // Check if time falls in morning window
    if !timeObj.Before(morningStart) && timeObj.Before(morningEnd) {
        return true, nil
    }

    return false, nil
}

// getDayType determines if date is weekday, weekend, or holiday
func (s *BookingTimeService) getDayType(date string, dateObj time.Time) (string, error) {
    // Check if holiday
    isHoliday, err := s.holidayService.IsHoliday(date)
    if err != nil {
        return "", err
    }

    if isHoliday {
        return "weekend", nil // Holidays use weekend rules
    }

    // Check day of week
    weekday := dateObj.Weekday()
    if weekday == time.Saturday || weekday == time.Sunday {
        return "weekend", nil
    }

    return "weekday", nil
}

// GetRulesForDate returns applicable rules for a specific date
func (s *BookingTimeService) GetRulesForDate(date string) ([]models.BookingTimeRule, error) {
    dateObj, err := time.Parse("2006-01-02", date)
    if err != nil {
        return nil, err
    }

    dayType, err := s.getDayType(date, dateObj)
    if err != nil {
        return nil, err
    }

    return s.bookingTimeRepo.GetRulesByDayType(dayType)
}
```

// DONE

---

### Phase 4: Handlers & API Endpoints

#### Step 4.1: BookingTimeHandler

**File:** `internal/handlers/booking_time_handler.go`

```go
package handlers

import (
    "encoding/json"
    "gassigeher/internal/config"
    "gassigeher/internal/middleware"
    "gassigeher/internal/models"
    "gassigeher/internal/repository"
    "gassigeher/internal/services"
    "net/http"
    "strconv"
)

type BookingTimeHandler struct {
    bookingTimeRepo *repository.BookingTimeRepository
    bookingTimeService *services.BookingTimeService
}

func NewBookingTimeHandler(
    bookingTimeRepo *repository.BookingTimeRepository,
    bookingTimeService *services.BookingTimeService,
) *BookingTimeHandler {
    return &BookingTimeHandler{
        bookingTimeRepo: bookingTimeRepo,
        bookingTimeService: bookingTimeService,
    }
}

// GetAvailableSlots returns available time slots for a date
// GET /api/booking-times/available?date=YYYY-MM-DD
func (h *BookingTimeHandler) GetAvailableSlots(w http.ResponseWriter, r *http.Request) {
    date := r.URL.Query().Get("date")
    if date == "" {
        respondError(w, http.StatusBadRequest, "date parameter required")
        return
    }

    slots, err := h.bookingTimeService.GetAvailableTimeSlots(date)
    if err != nil {
        respondError(w, http.StatusBadRequest, err.Error())
        return
    }

    respondJSON(w, http.StatusOK, map[string]interface{}{
        "date": date,
        "slots": slots,
    })
}

// GetRules returns all time rules
// GET /api/booking-times/rules
func (h *BookingTimeHandler) GetRules(w http.ResponseWriter, r *http.Request) {
    rules, err := h.bookingTimeRepo.GetAllRules()
    if err != nil {
        respondError(w, http.StatusInternalServerError, "Failed to load rules")
        return
    }

    respondJSON(w, http.StatusOK, rules)
}

// GetRulesForDate returns applicable rules for a specific date
// GET /api/booking-times/rules-for-date?date=YYYY-MM-DD
func (h *BookingTimeHandler) GetRulesForDate(w http.ResponseWriter, r *http.Request) {
    date := r.URL.Query().Get("date")
    if date == "" {
        respondError(w, http.StatusBadRequest, "date parameter required")
        return
    }

    rules, err := h.bookingTimeService.GetRulesForDate(date)
    if err != nil {
        respondError(w, http.StatusBadRequest, err.Error())
        return
    }

    respondJSON(w, http.StatusOK, rules)
}

// UpdateRules updates time rules (admin only)
// PUT /api/booking-times/rules
func (h *BookingTimeHandler) UpdateRules(w http.ResponseWriter, r *http.Request) {
    // Admin check done by middleware

    var rules []models.BookingTimeRule
    if err := json.NewDecoder(r.Body).Decode(&rules); err != nil {
        respondError(w, http.StatusBadRequest, "Invalid request body")
        return
    }

    // Validate each rule
    for _, rule := range rules {
        if err := rule.Validate(); err != nil {
            respondError(w, http.StatusBadRequest, err.Error())
            return
        }
    }

    // Update each rule
    for _, rule := range rules {
        if err := h.bookingTimeRepo.UpdateRule(rule.ID, &rule); err != nil {
            respondError(w, http.StatusInternalServerError, "Failed to update rule")
            return
        }
    }

    respondJSON(w, http.StatusOK, map[string]string{
        "message": "Rules updated successfully",
    })
}

// CreateRule creates a new time rule (admin only)
// POST /api/booking-times/rules
func (h *BookingTimeHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
    var rule models.BookingTimeRule
    if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
        respondError(w, http.StatusBadRequest, "Invalid request body")
        return
    }

    if err := rule.Validate(); err != nil {
        respondError(w, http.StatusBadRequest, err.Error())
        return
    }

    if err := h.bookingTimeRepo.CreateRule(&rule); err != nil {
        respondError(w, http.StatusInternalServerError, "Failed to create rule")
        return
    }

    respondJSON(w, http.StatusCreated, rule)
}

// DeleteRule deletes a time rule (admin only)
// DELETE /api/booking-times/rules/:id
func (h *BookingTimeHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
    idStr := r.URL.Path[len("/api/booking-times/rules/"):]
    id, err := strconv.Atoi(idStr)
    if err != nil {
        respondError(w, http.StatusBadRequest, "Invalid rule ID")
        return
    }

    if err := h.bookingTimeRepo.DeleteRule(id); err != nil {
        respondError(w, http.StatusInternalServerError, "Failed to delete rule")
        return
    }

    respondJSON(w, http.StatusOK, map[string]string{
        "message": "Rule deleted successfully",
    })
}
```

#### Step 4.2: HolidayHandler

**File:** `internal/handlers/holiday_handler.go`

```go
package handlers

import (
    "encoding/json"
    "gassigeher/internal/middleware"
    "gassigeher/internal/models"
    "gassigeher/internal/repository"
    "gassigeher/internal/services"
    "net/http"
    "strconv"
    "time"
)

type HolidayHandler struct {
    holidayRepo    *repository.HolidayRepository
    holidayService *services.HolidayService
}

func NewHolidayHandler(
    holidayRepo *repository.HolidayRepository,
    holidayService *services.HolidayService,
) *HolidayHandler {
    return &HolidayHandler{
        holidayRepo: holidayRepo,
        holidayService: holidayService,
    }
}

// GetHolidays returns all holidays for a year
// GET /api/holidays?year=2025
func (h *HolidayHandler) GetHolidays(w http.ResponseWriter, r *http.Request) {
    yearStr := r.URL.Query().Get("year")
    year := time.Now().Year() // Default to current year

    if yearStr != "" {
        y, err := strconv.Atoi(yearStr)
        if err == nil {
            year = y
        }
    }

    holidays, err := h.holidayService.GetHolidaysForYear(year)
    if err != nil {
        respondError(w, http.StatusInternalServerError, "Failed to load holidays")
        return
    }

    respondJSON(w, http.StatusOK, holidays)
}

// CreateHoliday adds a custom holiday (admin only)
// POST /api/holidays
func (h *HolidayHandler) CreateHoliday(w http.ResponseWriter, r *http.Request) {
    adminID, _ := r.Context().Value(middleware.UserIDKey).(int)

    var holiday models.CustomHoliday
    if err := json.NewDecoder(r.Body).Decode(&holiday); err != nil {
        respondError(w, http.StatusBadRequest, "Invalid request body")
        return
    }

    if err := holiday.Validate(); err != nil {
        respondError(w, http.StatusBadRequest, err.Error())
        return
    }

    holiday.Source = "admin"
    holiday.CreatedBy = &adminID

    if err := h.holidayRepo.CreateHoliday(&holiday); err != nil {
        respondError(w, http.StatusInternalServerError, "Failed to create holiday")
        return
    }

    respondJSON(w, http.StatusCreated, holiday)
}

// UpdateHoliday updates a holiday (admin only)
// PUT /api/holidays/:id
func (h *HolidayHandler) UpdateHoliday(w http.ResponseWriter, r *http.Request) {
    idStr := r.URL.Path[len("/api/holidays/"):]
    id, err := strconv.Atoi(idStr)
    if err != nil {
        respondError(w, http.StatusBadRequest, "Invalid holiday ID")
        return
    }

    var holiday models.CustomHoliday
    if err := json.NewDecoder(r.Body).Decode(&holiday); err != nil {
        respondError(w, http.StatusBadRequest, "Invalid request body")
        return
    }

    if err := h.holidayRepo.UpdateHoliday(id, &holiday); err != nil {
        respondError(w, http.StatusInternalServerError, "Failed to update holiday")
        return
    }

    respondJSON(w, http.StatusOK, map[string]string{
        "message": "Holiday updated successfully",
    })
}

// DeleteHoliday deletes a holiday (admin only)
// DELETE /api/holidays/:id
func (h *HolidayHandler) DeleteHoliday(w http.ResponseWriter, r *http.Request) {
    idStr := r.URL.Path[len("/api/holidays/"):]
    id, err := strconv.Atoi(idStr)
    if err != nil {
        respondError(w, http.StatusBadRequest, "Invalid holiday ID")
        return
    }

    if err := h.holidayRepo.DeleteHoliday(id); err != nil {
        respondError(w, http.StatusInternalServerError, "Failed to delete holiday")
        return
    }

    respondJSON(w, http.StatusOK, map[string]string{
        "message": "Holiday deleted successfully",
    })
}
```

#### Step 4.3: Update BookingHandler

**File:** Update `internal/handlers/booking_handler.go`

Add validation before creating bookings:

```go
// In CreateBooking method, add after existing validations:

// Validate booking time
if err := h.bookingTimeService.ValidateBookingTime(booking.Date, booking.ScheduledTime); err != nil {
    respondError(w, http.StatusBadRequest, err.Error())
    return
}

// Check if morning walk requires approval
requiresApproval, err := h.bookingTimeService.RequiresApproval(booking.ScheduledTime)
if err != nil {
    respondError(w, http.StatusInternalServerError, "Failed to check approval requirement")
    return
}

booking.RequiresApproval = requiresApproval
if requiresApproval {
    booking.ApprovalStatus = "pending"
} else {
    booking.ApprovalStatus = "approved"
}

// ... continue with existing booking creation logic
```

Add new approval methods:

```go
// ApprovePendingBooking approves a morning walk booking
// PUT /api/bookings/:id/approve
func (h *BookingHandler) ApprovePendingBooking(w http.ResponseWriter, r *http.Request) {
    adminID, _ := r.Context().Value(middleware.UserIDKey).(int)

    idStr := r.URL.Path[len("/api/bookings/"):]
    idStr = idStr[:len(idStr)-len("/approve")]
    id, err := strconv.Atoi(idStr)
    if err != nil {
        respondError(w, http.StatusBadRequest, "Invalid booking ID")
        return
    }

    if err := h.bookingRepo.ApproveBooking(id, adminID); err != nil {
        respondError(w, http.StatusInternalServerError, err.Error())
        return
    }

    // TODO: Send email notification to user

    respondJSON(w, http.StatusOK, map[string]string{
        "message": "Booking approved successfully",
    })
}

// RejectPendingBooking rejects a morning walk booking
// PUT /api/bookings/:id/reject
func (h *BookingHandler) RejectPendingBooking(w http.ResponseWriter, r *http.Request) {
    adminID, _ := r.Context().Value(middleware.UserIDKey).(int)

    idStr := r.URL.Path[len("/api/bookings/"):]
    idStr = idStr[:len(idStr)-len("/reject")]
    id, err := strconv.Atoi(idStr)
    if err != nil {
        respondError(w, http.StatusBadRequest, "Invalid booking ID")
        return
    }

    var req struct {
        Reason string `json:"reason"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, http.StatusBadRequest, "Invalid request body")
        return
    }

    if req.Reason == "" {
        respondError(w, http.StatusBadRequest, "Rejection reason required")
        return
    }

    if err := h.bookingRepo.RejectBooking(id, adminID, req.Reason); err != nil {
        respondError(w, http.StatusInternalServerError, err.Error())
        return
    }

    // TODO: Send email notification to user with reason

    respondJSON(w, http.StatusOK, map[string]string{
        "message": "Booking rejected successfully",
    })
}

// GetPendingApprovals returns all bookings awaiting approval (admin only)
// GET /api/bookings/pending-approvals
func (h *BookingHandler) GetPendingApprovals(w http.ResponseWriter, r *http.Request) {
    bookings, err := h.bookingRepo.GetPendingApprovalBookings()
    if err != nil {
        respondError(w, http.StatusInternalServerError, "Failed to load pending bookings")
        return
    }

    respondJSON(w, http.StatusOK, bookings)
}
```

// DONE

---

### Phase 5: Route Registration

**File:** Update `cmd/server/main.go`

```go
// Initialize new repositories
bookingTimeRepo := repository.NewBookingTimeRepository(db)
holidayRepo := repository.NewHolidayRepository(db)

// Initialize new services
holidayService := services.NewHolidayService(holidayRepo, settingsRepo)
bookingTimeService := services.NewBookingTimeService(bookingTimeRepo, holidayService, settingsRepo)

// Initialize new handlers
bookingTimeHandler := handlers.NewBookingTimeHandler(bookingTimeRepo, bookingTimeService)
holidayHandler := handlers.NewHolidayHandler(holidayRepo, holidayService)

// Update booking handler to include new services
bookingHandler := handlers.NewBookingHandler(db, cfg, bookingTimeService)

// Public routes (available time slots)
router.HandleFunc("/api/booking-times/available", bookingTimeHandler.GetAvailableSlots).Methods("GET")
router.HandleFunc("/api/booking-times/rules-for-date", bookingTimeHandler.GetRulesForDate).Methods("GET")
router.HandleFunc("/api/holidays", holidayHandler.GetHolidays).Methods("GET")

// Protected routes (user bookings with time validation)
// ... existing booking routes ...

// Admin routes (manage time rules and holidays)
admin.HandleFunc("/api/booking-times/rules", bookingTimeHandler.GetRules).Methods("GET")
admin.HandleFunc("/api/booking-times/rules", bookingTimeHandler.UpdateRules).Methods("PUT")
admin.HandleFunc("/api/booking-times/rules", bookingTimeHandler.CreateRule).Methods("POST")
admin.HandleFunc("/api/booking-times/rules/{id}", bookingTimeHandler.DeleteRule).Methods("DELETE")

admin.HandleFunc("/api/holidays", holidayHandler.CreateHoliday).Methods("POST")
admin.HandleFunc("/api/holidays/{id}", holidayHandler.UpdateHoliday).Methods("PUT")
admin.HandleFunc("/api/holidays/{id}", holidayHandler.DeleteHoliday).Methods("DELETE")

admin.HandleFunc("/api/bookings/pending-approvals", bookingHandler.GetPendingApprovals).Methods("GET")
admin.HandleFunc("/api/bookings/{id}/approve", bookingHandler.ApprovePendingBooking).Methods("PUT")
admin.HandleFunc("/api/bookings/{id}/reject", bookingHandler.RejectPendingBooking).Methods("PUT")
```

// DONE

---

## Frontend Implementation

### Phase 6: API Client Updates

**File:** Update `frontend/js/api.js`

```javascript
// Booking time methods
getAvailableTimeSlots(date) {
    return this.request(`/api/booking-times/available?date=${date}`, 'GET');
},

getRulesForDate(date) {
    return this.request(`/api/booking-times/rules-for-date?date=${date}`, 'GET');
},

getBookingTimeRules() {
    return this.request('/api/booking-times/rules', 'GET');
},

updateBookingTimeRules(rules) {
    return this.request('/api/booking-times/rules', 'PUT', rules);
},

createBookingTimeRule(rule) {
    return this.request('/api/booking-times/rules', 'POST', rule);
},

deleteBookingTimeRule(id) {
    return this.request(`/api/booking-times/rules/${id}`, 'DELETE');
},

// Holiday methods
getHolidays(year) {
    return this.request(`/api/holidays?year=${year}`, 'GET');
},

createHoliday(holiday) {
    return this.request('/api/holidays', 'POST', holiday);
},

updateHoliday(id, holiday) {
    return this.request(`/api/holidays/${id}`, 'PUT', holiday);
},

deleteHoliday(id) {
    return this.request(`/api/holidays/${id}`, 'DELETE');
},

// Booking approval methods
getPendingApprovalBookings() {
    return this.request('/api/bookings/pending-approvals', 'GET');
},

approveBooking(id) {
    return this.request(`/api/bookings/${id}/approve`, 'PUT');
},

rejectBooking(id, reason) {
    return this.request(`/api/bookings/${id}/reject`, 'PUT', { reason });
},
```

// DONE

### Phase 7: Update Booking Form

**File:** Update `frontend/dogs.html` (booking form section)

Add time slot selector with validation:

```html
<!-- After date input, add time slot selector -->
<div class="form-group">
    <label data-i18n="bookings.time" for="booking-time">Uhrzeit</label>
    <select id="booking-time" required>
        <option value="">Bitte wählen...</option>
    </select>
    <small id="time-approval-notice" class="text-warning" style="display: none;">
        ⚠️ Vormittagsspaziergänge erfordern eine Admin-Genehmigung
    </small>
</div>

<!-- Add info box showing time windows -->
<div id="time-rules-info" class="alert alert-info" style="display: none;">
    <h6>Erlaubte Buchungszeiten:</h6>
    <ul id="time-rules-list"></ul>
</div>
```

**File:** Update `frontend/js/dogs.js` (or create booking-form.js)

```javascript
// When date is selected, load available time slots
document.getElementById('booking-date').addEventListener('change', async (e) => {
    const date = e.target.value;
    if (!date) return;

    try {
        // Load available slots
        const response = await window.api.getAvailableTimeSlots(date);
        const slots = response.slots || [];

        const timeSelect = document.getElementById('booking-time');
        timeSelect.innerHTML = '<option value="">Bitte wählen...</option>';

        slots.forEach(slot => {
            const option = document.createElement('option');
            option.value = slot;
            option.textContent = slot;
            timeSelect.appendChild(option);
        });

        // Load and display time rules
        const rulesResponse = await window.api.getRulesForDate(date);
        const rules = rulesResponse || [];

        const rulesList = document.getElementById('time-rules-list');
        const rulesInfo = document.getElementById('time-rules-info');

        rulesList.innerHTML = '';
        rules.forEach(rule => {
            if (!rule.is_blocked) {
                const li = document.createElement('li');
                li.textContent = `${rule.rule_name}: ${rule.start_time} - ${rule.end_time}`;
                rulesList.appendChild(li);
            }
        });

        rulesInfo.style.display = 'block';

    } catch (error) {
        console.error('Failed to load time slots:', error);
        alert('Fehler beim Laden der Zeitfenster');
    }
});

// When time is selected, check if approval required
document.getElementById('booking-time').addEventListener('change', (e) => {
    const time = e.target.value;
    const notice = document.getElementById('time-approval-notice');

    if (time >= '09:00' && time < '12:00') {
        // Morning walk - may require approval
        notice.style.display = 'block';
    } else {
        notice.style.display = 'none';
    }
});

// Update booking submission to use selected time
// (modify existing booking form handler)
```

// DONE

---

### Phase 8: New Admin Page - Booking Time Settings

**File:** `frontend/admin-booking-times.html`

```html
<!DOCTYPE html>
<html lang="de">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Buchungszeiten verwalten - Gassigeher Admin</title>
    <link rel="stylesheet" href="/assets/css/main.css">
</head>
<body>
    <!-- Navigation (copy from other admin pages) -->
    <nav class="admin-nav">
        <!-- 8-item navigation -->
    </nav>

    <div class="container">
        <h1>Buchungszeiten verwalten</h1>

        <!-- Settings -->
        <section class="card">
            <h2>Einstellungen</h2>
            <div class="form-group">
                <label>
                    <input type="checkbox" id="morning-approval-toggle">
                    Vormittagsspaziergänge erfordern Admin-Genehmigung
                </label>
            </div>
            <div class="form-group">
                <label>
                    <input type="checkbox" id="use-feiertage-api">
                    Automatische Feiertage-Erkennung (Baden-Württemberg)
                </label>
            </div>
            <button id="save-settings-btn" class="btn btn-primary">Einstellungen speichern</button>
        </section>

        <!-- Time Rules Tabs -->
        <section class="card">
            <h2>Zeitfenster konfigurieren</h2>

            <div class="tabs">
                <button class="tab-btn active" data-tab="weekday">Wochentags</button>
                <button class="tab-btn" data-tab="weekend">Wochenende/Feiertage</button>
            </div>

            <!-- Weekday Rules -->
            <div id="weekday-tab" class="tab-content active">
                <h3>Montag bis Freitag</h3>
                <table class="table">
                    <thead>
                        <tr>
                            <th>Zeitfenster</th>
                            <th>Von</th>
                            <th>Bis</th>
                            <th>Typ</th>
                            <th>Aktionen</th>
                        </tr>
                    </thead>
                    <tbody id="weekday-rules">
                        <!-- Populated by JS -->
                    </tbody>
                </table>
                <button id="add-weekday-rule-btn" class="btn btn-secondary">+ Zeitfenster hinzufügen</button>
            </div>

            <!-- Weekend Rules -->
            <div id="weekend-tab" class="tab-content">
                <h3>Samstag, Sonntag, Feiertage</h3>
                <table class="table">
                    <thead>
                        <tr>
                            <th>Zeitfenster</th>
                            <th>Von</th>
                            <th>Bis</th>
                            <th>Typ</th>
                            <th>Aktionen</th>
                        </tr>
                    </thead>
                    <tbody id="weekend-rules">
                        <!-- Populated by JS -->
                    </tbody>
                </table>
                <button id="add-weekend-rule-btn" class="btn btn-secondary">+ Zeitfenster hinzufügen</button>
            </div>
        </section>

        <!-- Holidays Management -->
        <section class="card">
            <h2>Feiertage verwalten</h2>

            <div class="form-inline">
                <label>Jahr:</label>
                <select id="holiday-year-select">
                    <option value="2025">2025</option>
                    <option value="2026">2026</option>
                </select>
                <button id="load-holidays-btn" class="btn btn-primary">Laden</button>
            </div>

            <table class="table">
                <thead>
                    <tr>
                        <th>Datum</th>
                        <th>Name</th>
                        <th>Quelle</th>
                        <th>Status</th>
                        <th>Aktionen</th>
                    </tr>
                </thead>
                <tbody id="holidays-table">
                    <!-- Populated by JS -->
                </tbody>
            </table>

            <button id="add-holiday-btn" class="btn btn-secondary">+ Feiertag hinzufügen</button>
        </section>
    </div>

    <script src="/js/i18n.js"></script>
    <script src="/js/api.js"></script>
    <script src="/js/admin-booking-times.js"></script>
</body>
</html>
```

**File:** `frontend/js/admin-booking-times.js`

```javascript
(async function() {
    await window.i18n.load();

    // Load settings
    async function loadSettings() {
        try {
            const settings = await window.api.getSettings();

            document.getElementById('morning-approval-toggle').checked =
                settings.morning_walk_requires_approval === 'true';
            document.getElementById('use-feiertage-api').checked =
                settings.use_feiertage_api === 'true';
        } catch (error) {
            console.error('Failed to load settings:', error);
        }
    }

    // Save settings
    document.getElementById('save-settings-btn').addEventListener('click', async () => {
        const morningApproval = document.getElementById('morning-approval-toggle').checked;
        const useFeiertageAPI = document.getElementById('use-feiertage-api').checked;

        try {
            await window.api.updateSettings([
                { key: 'morning_walk_requires_approval', value: morningApproval.toString() },
                { key: 'use_feiertage_api', value: useFeiertageAPI.toString() }
            ]);
            alert('Einstellungen gespeichert!');
        } catch (error) {
            alert('Fehler beim Speichern der Einstellungen');
        }
    });

    // Load time rules
    async function loadTimeRules() {
        try {
            const rules = await window.api.getBookingTimeRules();

            // Populate weekday rules
            const weekdayRules = rules.weekday || [];
            const weekdayTable = document.getElementById('weekday-rules');
            weekdayTable.innerHTML = '';

            weekdayRules.forEach(rule => {
                const row = createRuleRow(rule);
                weekdayTable.appendChild(row);
            });

            // Populate weekend rules
            const weekendRules = rules.weekend || [];
            const weekendTable = document.getElementById('weekend-rules');
            weekendTable.innerHTML = '';

            weekendRules.forEach(rule => {
                const row = createRuleRow(rule);
                weekendTable.appendChild(row);
            });
        } catch (error) {
            console.error('Failed to load rules:', error);
        }
    }

    // Create rule table row
    function createRuleRow(rule) {
        const tr = document.createElement('tr');

        tr.innerHTML = `
            <td>${rule.rule_name}</td>
            <td><input type="time" value="${rule.start_time}" data-field="start"></td>
            <td><input type="time" value="${rule.end_time}" data-field="end"></td>
            <td>
                <select data-field="blocked">
                    <option value="0" ${!rule.is_blocked ? 'selected' : ''}>Erlaubt</option>
                    <option value="1" ${rule.is_blocked ? 'selected' : ''}>Gesperrt</option>
                </select>
            </td>
            <td>
                <button class="btn-save" data-id="${rule.id}">Speichern</button>
                <button class="btn-delete" data-id="${rule.id}">Löschen</button>
            </td>
        `;

        // Save handler
        tr.querySelector('.btn-save').addEventListener('click', async () => {
            const updatedRule = {
                id: rule.id,
                day_type: rule.day_type,
                rule_name: rule.rule_name,
                start_time: tr.querySelector('[data-field="start"]').value,
                end_time: tr.querySelector('[data-field="end"]').value,
                is_blocked: tr.querySelector('[data-field="blocked"]').value === '1'
            };

            try {
                await window.api.updateBookingTimeRules([updatedRule]);
                alert('Zeitfenster gespeichert!');
            } catch (error) {
                alert('Fehler beim Speichern');
            }
        });

        // Delete handler
        tr.querySelector('.btn-delete').addEventListener('click', async () => {
            if (!confirm('Zeitfenster wirklich löschen?')) return;

            try {
                await window.api.deleteBookingTimeRule(rule.id);
                tr.remove();
                alert('Zeitfenster gelöscht!');
            } catch (error) {
                alert('Fehler beim Löschen');
            }
        });

        return tr;
    }

    // Load holidays
    async function loadHolidays(year) {
        try {
            const holidays = await window.api.getHolidays(year);
            const table = document.getElementById('holidays-table');
            table.innerHTML = '';

            holidays.forEach(holiday => {
                const row = createHolidayRow(holiday);
                table.appendChild(row);
            });
        } catch (error) {
            console.error('Failed to load holidays:', error);
        }
    }

    // Create holiday table row
    function createHolidayRow(holiday) {
        const tr = document.createElement('tr');

        tr.innerHTML = `
            <td>${holiday.date}</td>
            <td>${holiday.name}</td>
            <td>${holiday.source === 'api' ? 'Automatisch' : 'Manuell'}</td>
            <td>
                <label>
                    <input type="checkbox" ${holiday.is_active ? 'checked' : ''}
                           data-id="${holiday.id}" class="holiday-active-toggle">
                    Aktiv
                </label>
            </td>
            <td>
                <button class="btn-delete-holiday" data-id="${holiday.id}">Löschen</button>
            </td>
        `;

        // Toggle active status
        tr.querySelector('.holiday-active-toggle').addEventListener('change', async (e) => {
            try {
                await window.api.updateHoliday(holiday.id, {
                    name: holiday.name,
                    is_active: e.target.checked
                });
            } catch (error) {
                alert('Fehler beim Aktualisieren');
                e.target.checked = !e.target.checked;
            }
        });

        // Delete handler
        tr.querySelector('.btn-delete-holiday').addEventListener('click', async () => {
            if (!confirm('Feiertag wirklich löschen?')) return;

            try {
                await window.api.deleteHoliday(holiday.id);
                tr.remove();
            } catch (error) {
                alert('Fehler beim Löschen');
            }
        });

        return tr;
    }

    // Tab switching
    document.querySelectorAll('.tab-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            // Remove active class from all tabs
            document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
            document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));

            // Add active to clicked tab
            btn.classList.add('active');
            const tabId = btn.dataset.tab + '-tab';
            document.getElementById(tabId).classList.add('active');
        });
    });

    // Load holidays button
    document.getElementById('load-holidays-btn').addEventListener('click', () => {
        const year = document.getElementById('holiday-year-select').value;
        loadHolidays(parseInt(year));
    });

    // Initialize
    loadSettings();
    loadTimeRules();
    loadHolidays(new Date().getFullYear());
})();
```

// DONE

---

### Phase 9: Update Admin Bookings Page

**File:** Update `frontend/admin-bookings.html`

Add pending approvals section at top:

```html
<!-- Add before existing bookings table -->
<section class="card" id="pending-approvals-section">
    <h2>Genehmigungsanfragen <span id="pending-count" class="badge">0</span></h2>

    <table class="table">
        <thead>
            <tr>
                <th>Datum</th>
                <th>Uhrzeit</th>
                <th>Benutzer</th>
                <th>Hund</th>
                <th>Aktionen</th>
            </tr>
        </thead>
        <tbody id="pending-approvals-table">
            <!-- Populated by JS -->
        </tbody>
    </table>
</section>
```

**File:** Update `frontend/js/admin-bookings.js`

Add functions to load and handle approvals:

```javascript
// Load pending approval bookings
async function loadPendingApprovals() {
    try {
        const bookings = await window.api.getPendingApprovalBookings();
        const table = document.getElementById('pending-approvals-table');
        const badge = document.getElementById('pending-count');

        badge.textContent = bookings.length;

        if (bookings.length === 0) {
            document.getElementById('pending-approvals-section').style.display = 'none';
            return;
        }

        document.getElementById('pending-approvals-section').style.display = 'block';
        table.innerHTML = '';

        bookings.forEach(booking => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>${booking.date}</td>
                <td>${booking.scheduled_time}</td>
                <td>${booking.user_name}</td>
                <td>${booking.dog_name}</td>
                <td>
                    <button class="btn-approve" data-id="${booking.id}">✓ Genehmigen</button>
                    <button class="btn-reject" data-id="${booking.id}">✗ Ablehnen</button>
                </td>
            `;

            // Approve handler
            row.querySelector('.btn-approve').addEventListener('click', async () => {
                try {
                    await window.api.approveBooking(booking.id);
                    alert('Buchung genehmigt!');
                    loadPendingApprovals();
                    loadAllBookings(); // Refresh main table
                } catch (error) {
                    alert('Fehler beim Genehmigen');
                }
            });

            // Reject handler
            row.querySelector('.btn-reject').addEventListener('click', async () => {
                const reason = prompt('Grund für Ablehnung:');
                if (!reason) return;

                try {
                    await window.api.rejectBooking(booking.id, reason);
                    alert('Buchung abgelehnt');
                    loadPendingApprovals();
                    loadAllBookings();
                } catch (error) {
                    alert('Fehler beim Ablehnen');
                }
            });

            table.appendChild(row);
        });
    } catch (error) {
        console.error('Failed to load pending approvals:', error);
    }
}

// Call on page load
loadPendingApprovals();

// Refresh pending approvals every 30 seconds
setInterval(loadPendingApprovals, 30000);
```

// DONE

---

### Phase 10: Update User Dashboard

**File:** Update `frontend/dashboard.html`

Show approval status for user's bookings:

```html
<!-- In booking card template, add approval status -->
<div class="booking-card" data-status="${booking.approval_status}">
    <!-- Existing booking details -->

    ${booking.approval_status === 'pending' ? `
        <div class="alert alert-warning">
            ⏳ Warte auf Admin-Genehmigung
        </div>
    ` : ''}

    ${booking.approval_status === 'rejected' ? `
        <div class="alert alert-danger">
            ✗ Abgelehnt: ${booking.rejection_reason || 'Keine Begründung'}
        </div>
    ` : ''}
</div>
```

// DONE

---

## Translation Updates

**File:** Update `frontend/i18n/de.json`

```json
{
  "bookings": {
    "time": "Uhrzeit",
    "timeSlots": "Verfügbare Zeitfenster",
    "morningApprovalRequired": "Vormittagsspaziergänge erfordern eine Admin-Genehmigung",
    "timeRulesTitle": "Erlaubte Buchungszeiten",
    "pendingApproval": "Warte auf Genehmigung",
    "approved": "Genehmigt",
    "rejected": "Abgelehnt",
    "rejectionReason": "Ablehnungsgrund"
  },
  "admin": {
    "bookingTimes": "Buchungszeiten",
    "timeRules": "Zeitfenster",
    "holidays": "Feiertage",
    "pendingApprovals": "Genehmigungsanfragen",
    "approveBooking": "Buchung genehmigen",
    "rejectBooking": "Buchung ablehnen",
    "weekday": "Wochentags",
    "weekend": "Wochenende/Feiertage",
    "allowedTime": "Erlaubte Zeit",
    "blockedTime": "Gesperrte Zeit",
    "morningWalkSettings": "Vormittagsspaziergänge-Einstellungen",
    "requireApproval": "Genehmigung erforderlich",
    "holidaySource": "Quelle",
    "holidayAutomatic": "Automatisch (API)",
    "holidayManual": "Manuell hinzugefügt"
  }
}
```

---

## Testing Strategy

### Unit Tests

**File:** `internal/services/booking_time_service_test.go`

```go
package services

import (
    "testing"
    "time"
)

func TestValidateBookingTime_Weekday(t *testing.T) {
    // Test valid afternoon slot
    err := service.ValidateBookingTime("2025-01-27", "15:00") // Monday
    if err != nil {
        t.Errorf("Expected valid time, got error: %v", err)
    }

    // Test blocked feeding time
    err = service.ValidateBookingTime("2025-01-27", "17:00") // In 16:30-18:00 block
    if err == nil {
        t.Error("Expected error for blocked time, got nil")
    }

    // Test outside allowed windows
    err = service.ValidateBookingTime("2025-01-27", "20:00")
    if err == nil {
        t.Error("Expected error for time outside windows, got nil")
    }
}

func TestValidateBookingTime_Weekend(t *testing.T) {
    // Test valid afternoon slot on Saturday
    err := service.ValidateBookingTime("2025-01-25", "15:00") // Saturday
    if err != nil {
        t.Errorf("Expected valid time, got error: %v", err)
    }

    // Test blocked lunch time
    err = service.ValidateBookingTime("2025-01-25", "13:30") // In 13:00-14:00 block
    if err == nil {
        t.Error("Expected error for blocked time, got nil")
    }
}

func TestRequiresApproval_MorningWalk(t *testing.T) {
    // Test morning time
    requires, err := service.RequiresApproval("10:00")
    if err != nil {
        t.Errorf("Unexpected error: %v", err)
    }
    if !requires {
        t.Error("Expected morning walk to require approval")
    }

    // Test afternoon time
    requires, err = service.RequiresApproval("15:00")
    if err != nil {
        t.Errorf("Unexpected error: %v", err)
    }
    if requires {
        t.Error("Expected afternoon walk to NOT require approval")
    }
}

func TestGetAvailableTimeSlots_15MinGranularity(t *testing.T) {
    slots, err := service.GetAvailableTimeSlots("2025-01-27") // Monday
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }

    // Check morning slots present
    if !contains(slots, "09:00") {
        t.Error("Expected 09:00 to be available")
    }
    if !contains(slots, "09:15") {
        t.Error("Expected 09:15 to be available (15-min granularity)")
    }

    // Check blocked times NOT present
    if contains(slots, "17:00") {
        t.Error("Expected 17:00 to be blocked (feeding time)")
    }
}
```

**File:** `internal/services/holiday_service_test.go`

```go
package services

import (
    "testing"
)

func TestIsHoliday_KnownHoliday(t *testing.T) {
    // Seed test holiday
    _ = holidayRepo.CreateHoliday(&models.CustomHoliday{
        Date: "2025-01-01",
        Name: "Neujahrstag",
        Source: "api",
        IsActive: true,
    })

    isHoliday, err := service.IsHoliday("2025-01-01")
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }

    if !isHoliday {
        t.Error("Expected 2025-01-01 to be a holiday")
    }
}

func TestIsHoliday_NotHoliday(t *testing.T) {
    isHoliday, err := service.IsHoliday("2025-01-15") // Regular Wednesday
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }

    if isHoliday {
        t.Error("Expected 2025-01-15 to NOT be a holiday")
    }
}

func TestFetchAndCacheHolidays(t *testing.T) {
    err := service.FetchAndCacheHolidays(2025)
    if err != nil {
        t.Fatalf("Failed to fetch holidays: %v", err)
    }

    // Check cache was created
    cached, err := holidayRepo.GetCachedHolidays(2025, "BW")
    if err != nil {
        t.Fatalf("Cache not created: %v", err)
    }

    if cached == "" {
        t.Error("Expected cached data, got empty string")
    }
}
```

### Integration Tests

**File:** `internal/handlers/booking_time_handler_test.go`

Test API endpoints with real HTTP requests.

### Manual Testing Checklist

**User Flow:**
1. ✅ Navigate to dogs page
2. ✅ Select dog and date (weekday)
3. ✅ Verify afternoon time slots shown (14:00-16:30, 18:00-19:30)
4. ✅ Verify blocked times NOT shown (13:00-14:00, 16:30-18:00)
5. ✅ Select morning time (10:00)
6. ✅ Verify "requires approval" warning shown
7. ✅ Submit booking
8. ✅ Check dashboard - booking shows "Pending Approval"
9. ✅ Try selecting weekend date
10. ✅ Verify different time slots (09:00-12:00, 14:00-17:00)
11. ✅ Verify weekend blocked times (12:00-14:00)

**Admin Flow:**
1. ✅ Login as admin
2. ✅ Navigate to Pending Approvals section
3. ✅ See test user's morning booking
4. ✅ Approve booking
5. ✅ Verify booking disappears from pending list
6. ✅ Check user's dashboard - booking now "Approved"
7. ✅ Create new morning booking as user
8. ✅ Reject booking with reason
9. ✅ Verify user sees rejection reason
10. ✅ Navigate to Booking Times settings
11. ✅ Modify afternoon window (14:00-16:00 instead of 14:00-16:30)
12. ✅ Save changes
13. ✅ Go back to dogs page as user
14. ✅ Verify new time slots reflect changes (16:15, 16:30 no longer available)
15. ✅ Navigate to Holidays management
16. ✅ Load 2025 holidays - verify BW holidays loaded
17. ✅ Disable "Heilige Drei Könige" (Jan 6)
18. ✅ Try booking on Jan 6 - should use weekday rules now
19. ✅ Add custom holiday (e.g., "Shelter Anniversary" on specific date)
20. ✅ Verify that date uses weekend rules
21. ✅ Toggle "Morning walks require approval" OFF
22. ✅ Create morning booking as user
23. ✅ Verify auto-approved (no pending status)

**Holiday API Test:**
1. ✅ Clear `feiertage_cache` table
2. ✅ Restart server
3. ✅ Navigate to holidays page
4. ✅ Load 2025 - verify API called and cached
5. ✅ Check `feiertage_cache` table - verify entry exists
6. ✅ Reload page - verify cache used (no API call)
7. ✅ Wait 7+ days or manually expire cache
8. ✅ Reload - verify API called again

---

## Deployment Checklist

### Pre-Deployment

- [ ] Run all unit tests: `go test ./... -v`
- [ ] Run integration tests
- [ ] Manual testing completed (all flows above)
- [ ] Database migration tested on fresh DB
- [ ] Database migration tested on existing DB with data
- [ ] Translations complete (all German strings)
- [ ] Admin navigation updated (add Booking Times link)
- [ ] API documentation updated (API.md)
- [ ] User guide updated (USER_GUIDE.md)
- [ ] Admin guide updated (ADMIN_GUIDE.md)

### Deployment Steps

1. **Backup database** before deployment
2. **Pull latest code** from repository
3. **Run migration** (automatic on startup)
4. **Verify migration** succeeded (check tables created)
5. **Restart server**
6. **Verify default time rules** seeded
7. **Test basic booking** with time validation
8. **Configure settings** via admin panel
9. **Test holiday API** connectivity
10. **Monitor logs** for errors

### Post-Deployment

- [ ] Verify existing bookings still work
- [ ] Verify new bookings require valid times
- [ ] Verify admin can approve/reject morning bookings
- [ ] Verify holidays auto-fetched from API
- [ ] Verify time rules configurable
- [ ] Monitor error logs for 24 hours
- [ ] Collect user feedback

---

## Future Enhancements

### Phase 11 (Optional)

1. **Email Notifications:**
   - Send email when morning booking approved
   - Send email when morning booking rejected (include reason)
   - Reminder email for admins if pending approvals > 24 hours

2. **Booking Capacity Limits:**
   - Limit total bookings per time slot (e.g., max 5 dogs per hour)
   - Show "fully booked" for popular times
   - Waitlist functionality

3. **Recurring Bookings:**
   - Allow users to book "every Monday at 15:00" for next month
   - Bulk approval for recurring bookings

4. **Calendar View:**
   - Visual calendar showing available/blocked times
   - Color-coded by availability
   - Click to book directly from calendar

5. **Mobile App:**
   - Push notifications for approval/rejection
   - Quick booking widget

6. **Analytics:**
   - Most popular booking times
   - Approval/rejection rates
   - Holiday impact on bookings

---

## Summary

This implementation plan provides complete booking time restrictions with:

✅ **Configurable time windows** per day type (weekday/weekend/holiday)
✅ **Automatic holiday detection** via feiertage-api.de API
✅ **Admin-configurable blocked times** (feeding periods)
✅ **Morning walk approval system** (optional)
✅ **15-minute booking granularity**
✅ **Real-time validation** (frontend + backend)
✅ **Admin UI** for time management
✅ **Custom holiday management**
✅ **Complete German translations**
✅ **Comprehensive testing strategy**
✅ **Production-ready deployment plan**

**Estimated Implementation Time:** 3-5 days for experienced developer

**Files Created:** 15 new files
**Files Modified:** 8 existing files
**Database Tables Added:** 3 tables
**API Endpoints Added:** 15 endpoints
**Frontend Pages Added:** 1 admin page
**Frontend Pages Modified:** 3 pages

This feature seamlessly integrates with the existing Gassigeher system while maintaining code quality, architecture consistency, and German-only UI.
