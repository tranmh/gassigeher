# Gassigeher Test Data Generator
# Generates realistic test data for the dog walking booking system
# Usage: .\scripts\gentestdata.ps1

param(
    [string]$DatabasePath = ".\gassigeher.db",
    [string]$EnvFile = ".\.env"
)

# CRITICAL: Set console and output encoding to UTF-8 to handle German umlauts correctly
$OutputEncoding = [System.Text.Encoding]::UTF8
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$PSDefaultParameterValues['*:Encoding'] = 'utf8'

# Color output helpers
function Write-Step { param($msg) Write-Host "==> $msg" -ForegroundColor Cyan }
function Write-Success { param($msg) Write-Host "[OK] $msg" -ForegroundColor Green }
function Write-Info { param($msg) Write-Host "[INFO] $msg" -ForegroundColor Yellow }
function Write-Error { param($msg) Write-Host "[ERROR] $msg" -ForegroundColor Red }

Write-Host @"
================================================================
      Gassigeher Test Data Generator
      Generates realistic test data for next 2 weeks
================================================================
"@ -ForegroundColor Cyan

# Check for sqlite3.exe in bin folder or PATH
$sqlitePath = ".\bin\sqlite3.exe"
if (-not (Test-Path $sqlitePath)) {
    $sqlite3 = Get-Command sqlite3.exe -ErrorAction SilentlyContinue
    if ($sqlite3) {
        $sqlitePath = "sqlite3.exe"
        Write-Success "Found sqlite3.exe in PATH at: $($sqlite3.Source)"
    } else {
        Write-Error "sqlite3.exe not found in bin folder or PATH"
        Write-Info "Please ensure bin/sqlite3.exe exists"
        exit 1
    }
} else {
    Write-Success "Found sqlite3.exe at: $sqlitePath"
}

# Load environment variables from .env if exists
$envVars = @{}
if (Test-Path $EnvFile) {
    Write-Step "Loading environment variables from $EnvFile"
    Get-Content $EnvFile | ForEach-Object {
        if ($_ -match '^([^=#]+)=(.*)$') {
            $key = $matches[1].Trim()
            $value = $matches[2].Trim()
            $envVars[$key] = $value
            [Environment]::SetEnvironmentVariable($key, $value, "Process")
        }
    }
    Write-Success "Environment variables loaded"
}

# Get database path from env or parameter
$dbPath = $env:DATABASE_PATH
if (-not $dbPath -and $envVars.ContainsKey('DATABASE_PATH')) {
    $dbPath = $envVars['DATABASE_PATH']
}
if (-not $dbPath) { $dbPath = $DatabasePath }

if (-not (Test-Path $dbPath)) {
    Write-Error "Database not found at: $dbPath"
    Write-Info "Please run the application first to create the database"
    exit 1
}

Write-Info "Using database: $dbPath"

# Get super admin email from env (must be valid email)
$adminEmail = $env:SUPER_ADMIN_EMAIL
if (-not $adminEmail -and $envVars.ContainsKey('SUPER_ADMIN_EMAIL')) {
    $adminEmail = $envVars['SUPER_ADMIN_EMAIL']
}
if (-not $adminEmail) {
    $adminEmail = "admin@tierheim-goeppingen.de"
    Write-Info "No SUPER_ADMIN_EMAIL found, using default: $adminEmail"
} else {
    # Validate it's a proper email (contains @)
    if ($adminEmail -notmatch '@') {
        Write-Info "SUPER_ADMIN_EMAIL '$adminEmail' is not a valid email address, using default"
        $adminEmail = "admin@tierheim-goeppingen.de"
    } else {
        Write-Info "Using super admin email from .env: $adminEmail"
    }
}

Write-Step "Generating SQL file..."

# Bcrypt hash for "test123"
$TEST_PASSWORD_HASH = '$2a$10$LT4jdYaamd5Sxed9IhHTKuedmp/AvzGH27pJwCFzxAqAuO0c6OqfC'

# Test data arrays
$germanFirstNames = @("Max", "Anna", "Lukas", "Sophie", "Felix", "Emma", "Leon", "Mia", "Paul", "Laura", "Jonas", "Lena", "Tim", "Sarah")
$germanLastNames = @("Müller", "Schmidt", "Schneider", "Fischer", "Weber", "Meyer", "Wagner", "Becker", "Schulz", "Hoffmann")

$dogNames = @("Bella", "Max", "Luna", "Charlie", "Lucy", "Rocky", "Daisy", "Duke", "Molly", "Zeus", "Lola", "Bruno", "Coco", "Buster", "Rosie", "Rex", "Penny", "Oscar")
$dogBreeds = @("Labrador", "Schäferhund", "Golden Retriever", "Bulldogge", "Pudel", "Husky", "Beagle", "Dackel", "Boxer", "Rottweiler", "Mischling", "Collie")

$userNotes = @(
    "Sehr entspannter Spaziergang, Hund hat gut gehört.",
    "Hund war sehr energiegeladen und verspielt.",
    "Kleine Pause am See gemacht, Hund hat getrunken.",
    "Begegnung mit anderen Hunden - alles gut verlaufen.",
    "Hund hat viel geschnüffelt, brauchte etwas Zeit.",
    "Toller Spaziergang im Wald, Hund war glücklich."
)

$specialNeeds = @(
    "Keine Besonderheiten",
    "Verträgt sich gut mit anderen Hunden",
    "Sollte an der Leine geführt werden",
    "Mag keine Katzen",
    "Braucht viel Wasser bei Spaziergängen"
)

# Create SQL file
$sqlFile = "scripts\testdata.sql"
$sql = New-Object System.Text.StringBuilder

# SQL Header
[void]$sql.AppendLine("-- Gassigeher Test Data")
[void]$sql.AppendLine("-- Generated: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')")
[void]$sql.AppendLine("-- WARNING: This will delete all existing data!")
[void]$sql.AppendLine("")
[void]$sql.AppendLine("BEGIN TRANSACTION;")
[void]$sql.AppendLine("")

# Clear existing data in correct order (respecting foreign keys)
[void]$sql.AppendLine("-- Clear existing data")
$tables = @("reactivation_requests", "experience_requests", "custom_holidays", "feiertage_cache", "booking_time_rules", "blocked_dates", "bookings", "dogs", "users", "system_settings")
foreach ($table in $tables) {
    [void]$sql.AppendLine("DELETE FROM $table;")
}
[void]$sql.AppendLine("")

# Reset autoincrement sequences
[void]$sql.AppendLine("-- Reset autoincrement sequences")
[void]$sql.AppendLine("DELETE FROM sqlite_sequence WHERE name IN ('users', 'dogs', 'bookings', 'blocked_dates', 'experience_requests', 'reactivation_requests', 'booking_time_rules', 'custom_holidays', 'feiertage_cache');")
[void]$sql.AppendLine("")

# System settings
[void]$sql.AppendLine("-- System settings")
[void]$sql.AppendLine("INSERT INTO system_settings (key, value) VALUES")
[void]$sql.AppendLine("('booking_advance_days', '14'),")
[void]$sql.AppendLine("('cancellation_notice_hours', '12'),")
[void]$sql.AppendLine("('auto_deactivation_days', '365'),")
[void]$sql.AppendLine("('morning_walk_requires_approval', 'true'),")
[void]$sql.AppendLine("('use_feiertage_api', 'true'),")
[void]$sql.AppendLine("('feiertage_state', 'BW'),")
[void]$sql.AppendLine("('booking_time_granularity', '15'),")
[void]$sql.AppendLine("('feiertage_cache_days', '7');")
[void]$sql.AppendLine("")

# Booking time rules
[void]$sql.AppendLine("-- Booking time rules (weekday and weekend)")
[void]$sql.AppendLine("INSERT INTO booking_time_rules (day_type, rule_name, start_time, end_time, is_blocked) VALUES")
[void]$sql.AppendLine("('weekday', 'Morning Walk', '09:00', '12:00', 0),")
[void]$sql.AppendLine("('weekday', 'Lunch Block', '13:00', '14:00', 1),")
[void]$sql.AppendLine("('weekday', 'Afternoon Walk', '14:00', '16:30', 0),")
[void]$sql.AppendLine("('weekday', 'Feeding Block', '16:30', '18:00', 1),")
[void]$sql.AppendLine("('weekday', 'Evening Walk', '18:00', '19:30', 0),")
[void]$sql.AppendLine("('weekend', 'Morning Walk', '09:00', '12:00', 0),")
[void]$sql.AppendLine("('weekend', 'Feeding Block', '12:00', '13:00', 1),")
[void]$sql.AppendLine("('weekend', 'Lunch Block', '13:00', '14:00', 1),")
[void]$sql.AppendLine("('weekend', 'Afternoon Walk', '14:00', '17:00', 0);")
[void]$sql.AppendLine("")

# Custom holidays (Baden-Württemberg 2025)
[void]$sql.AppendLine("-- Custom holidays (example Baden-Württemberg holidays for 2025)")
[void]$sql.AppendLine("INSERT INTO custom_holidays (date, name, is_active, source) VALUES")
[void]$sql.AppendLine("('2025-01-01', 'Neujahrstag', 1, 'api'),")
[void]$sql.AppendLine("('2025-01-06', 'Heilige Drei Könige', 1, 'api'),")
[void]$sql.AppendLine("('2025-04-18', 'Karfreitag', 1, 'api'),")
[void]$sql.AppendLine("('2025-04-21', 'Ostermontag', 1, 'api'),")
[void]$sql.AppendLine("('2025-05-01', 'Tag der Arbeit', 1, 'api'),")
[void]$sql.AppendLine("('2025-05-29', 'Christi Himmelfahrt', 1, 'api'),")
[void]$sql.AppendLine("('2025-06-09', 'Pfingstmontag', 1, 'api'),")
[void]$sql.AppendLine("('2025-06-19', 'Fronleichnam', 1, 'api'),")
[void]$sql.AppendLine("('2025-10-03', 'Tag der Deutschen Einheit', 1, 'api'),")
[void]$sql.AppendLine("('2025-11-01', 'Allerheiligen', 1, 'api'),")
[void]$sql.AppendLine("('2025-12-25', 'Erster Weihnachtsfeiertag', 1, 'api'),")
[void]$sql.AppendLine("('2025-12-26', 'Zweiter Weihnachtsfeiertag', 1, 'api');")
[void]$sql.AppendLine("")

# Generate users
[void]$sql.AppendLine("-- Users (12 total)")
$userCount = 0
$users = @()
$now = Get-Date -Format "yyyy-MM-dd HH:mm:ss"

# Super Admin user (ID=1)
[void]$sql.AppendLine("INSERT INTO users (email, name, phone, password_hash, experience_level, is_verified, is_active, is_admin, is_super_admin, terms_accepted_at, last_activity_at, created_at) VALUES")
[void]$sql.AppendLine("('$adminEmail', 'Super Admin', '+49 7161 12345', '$TEST_PASSWORD_HASH', 'orange', 1, 1, 1, 1, '$now', '$now', '$now'),")
$users += @{email=$adminEmail; level='orange'}
$userCount++

# Regular users with different experience levels
$experienceLevels = @(@{level='green'; count=4}, @{level='blue'; count=4}, @{level='orange'; count=3})
$totalRegularUsers = 11
$currentUser = 0

foreach ($expLevel in $experienceLevels) {
    for ($i = 0; $i -lt $expLevel.count; $i++) {
        $firstName = $germanFirstNames | Get-Random
        $lastName = $germanLastNames | Get-Random
        $email = "$($firstName.ToLower()).$($lastName.ToLower())$currentUser@example.com"
        $phone = "+49 " + (Get-Random -Minimum 100 -Maximum 999) + " " + (Get-Random -Minimum 1000000 -Maximum 9999999)
        $lastActivity = (Get-Date).AddDays(-(Get-Random -Minimum 1 -Maximum 30)).ToString("yyyy-MM-dd HH:mm:ss")
        $isActive = 1

        # Make user 5 inactive
        if ($currentUser -eq 4) {
            $isActive = 0
            $lastActivity = (Get-Date).AddDays(-400).ToString("yyyy-MM-dd HH:mm:ss")
        }

        $comma = if ($currentUser -eq $totalRegularUsers - 1) { ";" } else { "," }
        [void]$sql.AppendLine("('$email', '$firstName $lastName', '$phone', '$TEST_PASSWORD_HASH', '$($expLevel.level)', 1, $isActive, 0, 0, '$now', '$lastActivity', '$now')$comma")

        $users += @{email=$email; level=$expLevel.level}
        $userCount++
        $currentUser++
    }
}
[void]$sql.AppendLine("")

Write-Info "Generated $userCount users"

# Generate dogs
[void]$sql.AppendLine("-- Dogs (18 total)")
[void]$sql.AppendLine("INSERT INTO dogs (name, breed, size, age, category, photo, photo_thumbnail, special_needs, pickup_location, walk_route, walk_duration, special_instructions, default_morning_time, default_evening_time, is_available, created_at) VALUES")

$dogCategories = @(@{category='green'; count=7}, @{category='blue'; count=6}, @{category='orange'; count=5})
$dogSizes = @('small', 'medium', 'large')
$dogCount = 0
$totalDogs = 18

foreach ($catDef in $dogCategories) {
    for ($i = 0; $i -lt $catDef.count; $i++) {
        $name = $dogNames | Get-Random
        $breed = $dogBreeds | Get-Random
        $size = $dogSizes | Get-Random
        $age = Get-Random -Minimum 1 -Maximum 12
        $needs = $specialNeeds | Get-Random
        $pickup = "Tierheim Eingang"
        $route = "Standard Route $(Get-Random -Minimum 1 -Maximum 5)"
        $duration = @(30, 45, 60) | Get-Random
        $instructions = "Bitte vor dem Spaziergang anmelden"
        $morningTime = "09:00"
        $eveningTime = "15:00"
        $isAvailable = 1

        # Add sample photo paths (will be created by ProcessDogPhoto when real photos uploaded)
        # For test data, we use NULL to demonstrate placeholder functionality
        $photo = "NULL"
        $photoThumb = "NULL"

        # Make dogs 3 and 8 unavailable
        if ($dogCount -eq 3 -or $dogCount -eq 8) { $isAvailable = 0 }

        $comma = if ($dogCount -eq $totalDogs - 1) { ";" } else { "," }
        [void]$sql.AppendLine("('$name', '$breed', '$size', $age, '$($catDef.category)', $photo, $photoThumb, '$needs', '$pickup', '$route', $duration, '$instructions', '$morningTime', '$eveningTime', $isAvailable, '$now')$comma")
        $dogCount++
    }
}
[void]$sql.AppendLine("")

Write-Info "Generated $dogCount dogs"

# Generate blocked dates - need admin user ID (1) as created_by
[void]$sql.AppendLine("-- Blocked dates (3 random dates)")
[void]$sql.AppendLine("INSERT INTO blocked_dates (date, reason, created_by, created_at) VALUES")
$blockedDates = @()
for ($i = 0; $i -lt 3; $i++) {
    $randomDay = 3 + ($i * 4) # Days 3, 7, 11
    $blockedDate = (Get-Date).AddDays($randomDay).ToString("yyyy-MM-dd")
    $blockedDates += $blockedDate
    $comma = if ($i -eq 2) { ";" } else { "," }
    [void]$sql.AppendLine("('$blockedDate', 'Tierheim geschlossen - Testdaten', 1, '$now')$comma")
}
[void]$sql.AppendLine("")

Write-Info "Generated $($blockedDates.Count) blocked dates"

# Generate bookings
[void]$sql.AppendLine("-- Bookings (past, today, and future)")
$bookingValues = New-Object System.Collections.ArrayList
$usedSlots = @{} # Track used combinations to avoid UNIQUE constraint violations
$times = @{
    morning = @("09:00", "09:30", "10:00")
    evening = @("14:00", "14:30", "15:00", "15:30")
}
$walkTypes = @('morning', 'evening')

# Helper function to determine if time requires approval (09:00-12:00)
function Requires-Approval($time) {
    $hour = [int]$time.Substring(0, 2)
    return ($hour -ge 9 -and $hour -lt 12)
}

# Past bookings (completed)
for ($dayOffset = -14; $dayOffset -lt 0; $dayOffset++) {
    $bookDate = (Get-Date).AddDays($dayOffset).ToString("yyyy-MM-dd")
    $dailyBookings = Get-Random -Minimum 2 -Maximum 5

    for ($b = 0; $b -lt $dailyBookings; $b++) {
        # Find available dog/walk_type combination
        $attempts = 0
        do {
            $userId = Get-Random -Minimum 1 -Maximum ($userCount + 1)
            $dogId = Get-Random -Minimum 1 -Maximum ($dogCount + 1)
            $walkType = $walkTypes | Get-Random
            $slotKey = "$dogId|$bookDate|$walkType"
            $attempts++
        } while ($usedSlots.ContainsKey($slotKey) -and $attempts -lt 50)

        if ($attempts -ge 50) { continue } # Skip if no slot available
        $usedSlots[$slotKey] = $true

        $time = $times[$walkType] | Get-Random
        $createdAt = (Get-Date).AddDays($dayOffset).ToString("yyyy-MM-dd HH:mm:ss")
        $completedAt = (Get-Date).AddDays($dayOffset).AddHours(2).ToString("yyyy-MM-dd HH:mm:ss")

        # Add note to some bookings
        $note = if ((Get-Random -Minimum 1 -Maximum 100) -le 60) {
            "'$(($userNotes | Get-Random) -replace "'", "''")'"
        } else {
            "NULL"
        }

        # Approval fields for completed bookings
        $requiresApproval = if (Requires-Approval $time) { 1 } else { 0 }
        $approvalStatus = "'approved'"
        $approvedBy = if ($requiresApproval -eq 1) { 1 } else { "NULL" }
        $approvedAt = if ($requiresApproval -eq 1) { "'$createdAt'" } else { "NULL" }
        $rejectionReason = "NULL"

        [void]$bookingValues.Add("($userId, $dogId, '$bookDate', '$walkType', '$time', 'completed', '$completedAt', $note, '$createdAt', $requiresApproval, $approvalStatus, $approvedBy, $approvedAt, $rejectionReason)")
    }
}

# Today's bookings
$todayDate = (Get-Date).ToString("yyyy-MM-dd")
$pendingApprovalCount = 0
for ($b = 0; $b -lt 3; $b++) {
    $attempts = 0
    do {
        $userId = Get-Random -Minimum 1 -Maximum ($userCount + 1)
        $dogId = Get-Random -Minimum 1 -Maximum ($dogCount + 1)
        $walkType = $walkTypes | Get-Random
        $slotKey = "$dogId|$todayDate|$walkType"
        $attempts++
    } while ($usedSlots.ContainsKey($slotKey) -and $attempts -lt 50)

    if ($attempts -ge 50) { continue }
    $usedSlots[$slotKey] = $true

    $time = $times[$walkType] | Get-Random
    $status = if ($walkType -eq 'morning') { 'completed' } else { 'scheduled' }
    $completedAt = if ($status -eq 'completed') { "'$now'" } else { "NULL" }

    # Approval fields
    $requiresApproval = if (Requires-Approval $time) { 1 } else { 0 }

    if ($status -eq 'completed') {
        # Completed bookings are always approved
        $approvalStatus = "'approved'"
        $approvedBy = if ($requiresApproval -eq 1) { 1 } else { "NULL" }
        $approvedAt = if ($requiresApproval -eq 1) { "'$now'" } else { "NULL" }
    } else {
        # Scheduled bookings might be pending approval
        $approvalStatus = "'approved'"
        $approvedBy = "NULL"
        $approvedAt = "NULL"
    }
    $rejectionReason = "NULL"

    [void]$bookingValues.Add("($userId, $dogId, '$todayDate', '$walkType', '$time', '$status', $completedAt, NULL, '$now', $requiresApproval, $approvalStatus, $approvedBy, $approvedAt, $rejectionReason)")
}

# Future bookings
for ($dayOffset = 1; $dayOffset -le 14; $dayOffset++) {
    $bookDate = (Get-Date).AddDays($dayOffset).ToString("yyyy-MM-dd")

    # Skip blocked dates
    if ($blockedDates -contains $bookDate) { continue }

    $dailyBookings = Get-Random -Minimum 3 -Maximum 7

    for ($b = 0; $b -lt $dailyBookings; $b++) {
        $attempts = 0
        do {
            $userId = Get-Random -Minimum 1 -Maximum ($userCount + 1)
            $dogId = Get-Random -Minimum 1 -Maximum ($dogCount + 1)
            $walkType = $walkTypes | Get-Random
            $slotKey = "$dogId|$bookDate|$walkType"
            $attempts++
        } while ($usedSlots.ContainsKey($slotKey) -and $attempts -lt 50)

        if ($attempts -ge 50) { continue }
        $usedSlots[$slotKey] = $true

        $time = $times[$walkType] | Get-Random
        $status = if ((Get-Random -Minimum 1 -Maximum 100) -le 10) { 'cancelled' } else { 'scheduled' }

        # Approval fields
        $requiresApproval = if (Requires-Approval $time) { 1 } else { 0 }

        if ($status -eq 'cancelled') {
            # Cancelled bookings are approved first
            $approvalStatus = "'approved'"
            $approvedBy = "NULL"
            $approvedAt = "NULL"
        } elseif ($requiresApproval -eq 1 -and $pendingApprovalCount -lt 2 -and $dayOffset -le 7) {
            # Create some pending approval bookings (max 2, only in next week)
            $approvalStatus = "'pending'"
            $approvedBy = "NULL"
            $approvedAt = "NULL"
            $pendingApprovalCount++
        } elseif ($requiresApproval -eq 1 -and $pendingApprovalCount -eq 2 -and $dayOffset -eq 3) {
            # Create one rejected booking for testing
            $approvalStatus = "'rejected'"
            $approvedBy = 1
            $approvedAt = "'$now'"
            $status = 'cancelled' # Rejected bookings are cancelled
        } else {
            # Regular approved bookings
            $approvalStatus = "'approved'"
            $approvedBy = "NULL"
            $approvedAt = "NULL"
        }

        $rejectionReason = if ($approvalStatus -eq "'rejected'") { "'Nicht genügend Personal verfügbar für Vormittagstermine an diesem Tag'" } else { "NULL" }

        [void]$bookingValues.Add("($userId, $dogId, '$bookDate', '$walkType', '$time', '$status', NULL, NULL, '$now', $requiresApproval, $approvalStatus, $approvedBy, $approvedAt, $rejectionReason)")
    }
}

# Write bookings
[void]$sql.AppendLine("INSERT INTO bookings (user_id, dog_id, date, walk_type, scheduled_time, status, completed_at, user_notes, created_at, requires_approval, approval_status, approved_by, approved_at, rejection_reason) VALUES")
for ($i = 0; $i -lt $bookingValues.Count; $i++) {
    $comma = if ($i -eq $bookingValues.Count - 1) { ";" } else { "," }
    [void]$sql.AppendLine("$($bookingValues[$i])$comma")
}
[void]$sql.AppendLine("")

Write-Info "Generated $($bookingValues.Count) bookings"

# Experience requests
[void]$sql.AppendLine("-- Experience level requests")
[void]$sql.AppendLine("INSERT INTO experience_requests (user_id, requested_level, status, admin_message, reviewed_by, created_at, reviewed_at) VALUES")
$req1Date = (Get-Date).AddDays(-5).ToString("yyyy-MM-dd HH:mm:ss")
$req2Date = (Get-Date).AddDays(-3).ToString("yyyy-MM-dd HH:mm:ss")
$req3Date = (Get-Date).AddDays(-10).ToString("yyyy-MM-dd HH:mm:ss")
$req3Review = (Get-Date).AddDays(-8).ToString("yyyy-MM-dd HH:mm:ss")
$req4Date = (Get-Date).AddDays(-7).ToString("yyyy-MM-dd HH:mm:ss")
$req4Review = (Get-Date).AddDays(-6).ToString("yyyy-MM-dd HH:mm:ss")

[void]$sql.AppendLine("(2, 'blue', 'pending', NULL, NULL, '$req1Date', NULL),")
[void]$sql.AppendLine("(6, 'orange', 'pending', NULL, NULL, '$req2Date', NULL),")
[void]$sql.AppendLine("(3, 'blue', 'approved', 'Gute Erfahrung nachgewiesen', 1, '$req3Date', '$req3Review'),")
[void]$sql.AppendLine("(4, 'blue', 'denied', 'Bitte mehr Erfahrung sammeln', 1, '$req4Date', '$req4Review');")
[void]$sql.AppendLine("")

Write-Info "Generated 4 experience requests"

# Commit transaction
[void]$sql.AppendLine("COMMIT;")
[void]$sql.AppendLine("")
[void]$sql.AppendLine("-- Data generation complete")

# Write SQL file with UTF-8 encoding (no BOM) to ensure proper umlaut handling
$utf8NoBom = New-Object System.Text.UTF8Encoding($false)
$sqlFilePath = Join-Path $PSScriptRoot "..\$sqlFile"
[System.IO.File]::WriteAllText($sqlFilePath, $sql.ToString(), $utf8NoBom)
Write-Success "SQL file created with UTF-8 (no BOM): $sqlFile"

# Execute SQL file
Write-Step "Executing SQL file..."
try {
    $output = & $sqlitePath $dbPath ".read $sqlFile" 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Data inserted successfully"
    } else {
        Write-Error "SQLite execution failed: $output"
        exit 1
    }
} catch {
    Write-Error "Failed to execute SQL: $_"
    exit 1
}

# Summary output
Write-Host ""
Write-Host "================================================================" -ForegroundColor Green
Write-Host "         Test Data Generation Complete!" -ForegroundColor Green
Write-Host "================================================================" -ForegroundColor Green
Write-Host ""

Write-Host "Summary:" -ForegroundColor Cyan
Write-Host "  Users:                     $userCount (1 super admin, 1 inactive)" -ForegroundColor White
Write-Host "  Dogs:                      $dogCount (2 unavailable)" -ForegroundColor White
$bookingCountDisplay = $bookingValues.Count
Write-Host "  Bookings:                  $bookingCountDisplay (spanning 28 days)" -ForegroundColor White
Write-Host "  - Pending Approvals:       ~$pendingApprovalCount morning bookings" -ForegroundColor White
Write-Host "  - Rejected:                1 booking (with reason)" -ForegroundColor White
$blockedCountDisplay = $blockedDates.Count
Write-Host "  Blocked Dates:             $blockedCountDisplay" -ForegroundColor White
Write-Host "  Experience Requests:       4 (2 pending, 1 approved, 1 denied)" -ForegroundColor White
Write-Host "  Booking Time Rules:        9 (weekday/weekend windows)" -ForegroundColor White
Write-Host "  Custom Holidays:           12 (BW 2025 holidays)" -ForegroundColor White
Write-Host ""

Write-Host "Login Credentials - all users:" -ForegroundColor Cyan
Write-Host "  Password: test123" -ForegroundColor Yellow
Write-Host ""

Write-Host "Sample User Logins:" -ForegroundColor Cyan
Write-Host "  Super Admin:  $adminEmail" -ForegroundColor White
$sampleUsers = $users | Where-Object { $_.email -ne $adminEmail } | Select-Object -First 3
foreach ($user in $sampleUsers) {
    $levelName = switch ($user.level) {
        'green' { "Green" }
        'blue' { "Blue" }
        'orange' { "Orange" }
    }
    Write-Host "  $levelName User:  $($user.email)" -ForegroundColor White
}
Write-Host ""

Write-Host "Blocked Dates:" -ForegroundColor Cyan
foreach ($date in $blockedDates) {
    Write-Host "  - $date" -ForegroundColor White
}
Write-Host ""

Write-Host "Next Steps:" -ForegroundColor Cyan
Write-Host "  1. Start the application with: go run cmd/server/main.go" -ForegroundColor White
Write-Host "  2. Open browser to: http://localhost:8080" -ForegroundColor White
Write-Host "  3. Login with any of the above credentials" -ForegroundColor White
Write-Host "  4. Test booking flows for next 2 weeks" -ForegroundColor White
Write-Host ""

Write-Success "Database populated successfully!"
Write-Info "SQL file created: $sqlFile"
