package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
	_ "modernc.org/sqlite"
)

// setupTestDBForBookingTime creates a test database with booking_time_rules table
func setupTestDBForBookingTime(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create booking_time_rules table
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS booking_time_rules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		day_type TEXT NOT NULL,
		rule_name TEXT NOT NULL,
		start_time TEXT NOT NULL,
		end_time TEXT NOT NULL,
		is_blocked INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(day_type, rule_name)
	);
	`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	return db
}

// seedBookingTimeRules seeds the database with test data
func seedBookingTimeRules(t *testing.T, db *sql.DB) {
	rules := []struct {
		dayType   string
		ruleName  string
		startTime string
		endTime   string
		isBlocked int
	}{
		{"weekday", "Morgenspaziergang", "09:00", "12:00", 0},
		{"weekday", "Mittagspause", "13:00", "14:00", 1},
		{"weekday", "Nachmittagsspaziergang", "14:00", "16:30", 0},
		{"weekday", "Fütterungszeit", "17:00", "18:00", 1},
		{"weekday", "Abendspaziergang", "18:00", "19:30", 0},
		{"weekend", "Morgenspaziergang", "09:00", "12:00", 0},
		{"weekend", "Fütterungszeit", "12:00", "13:00", 1},
		{"weekend", "Mittagspause", "13:00", "14:00", 1},
		{"weekend", "Nachmittagsspaziergang", "14:00", "17:00", 0},
	}

	for _, r := range rules {
		_, err := db.Exec(`
			INSERT INTO booking_time_rules (day_type, rule_name, start_time, end_time, is_blocked)
			VALUES (?, ?, ?, ?, ?)
		`, r.dayType, r.ruleName, r.startTime, r.endTime, r.isBlocked)
		if err != nil {
			t.Fatalf("Failed to seed rule %s: %v", r.ruleName, err)
		}
	}
}

// Test 2.1.1: GetRulesByDayType - Query Filtering
func TestGetRulesByDayType_Weekday(t *testing.T) {
	db := setupTestDBForBookingTime(t)
	defer db.Close()

	seedBookingTimeRules(t, db)
	repo := NewBookingTimeRepository(db)

	// Test weekday rules
	rules, err := repo.GetRulesByDayType("weekday")
	if err != nil {
		t.Fatalf("GetRulesByDayType failed: %v", err)
	}

	expectedCount := 5
	if len(rules) != expectedCount {
		t.Errorf("Expected %d weekday rules, got %d", expectedCount, len(rules))
	}

	// Verify all rules are weekday type
	for _, rule := range rules {
		if rule.DayType != "weekday" {
			t.Errorf("Expected day_type 'weekday', got '%s'", rule.DayType)
		}
	}

	// Verify rules are ordered by start_time
	for i := 1; i < len(rules); i++ {
		if rules[i-1].StartTime > rules[i].StartTime {
			t.Errorf("Rules not ordered by start_time: %s > %s", rules[i-1].StartTime, rules[i].StartTime)
		}
	}
}

func TestGetRulesByDayType_Weekend(t *testing.T) {
	db := setupTestDBForBookingTime(t)
	defer db.Close()

	seedBookingTimeRules(t, db)
	repo := NewBookingTimeRepository(db)

	// Test weekend rules
	rules, err := repo.GetRulesByDayType("weekend")
	if err != nil {
		t.Fatalf("GetRulesByDayType failed: %v", err)
	}

	expectedCount := 4
	if len(rules) != expectedCount {
		t.Errorf("Expected %d weekend rules, got %d", expectedCount, len(rules))
	}

	// Verify all rules are weekend type
	for _, rule := range rules {
		if rule.DayType != "weekend" {
			t.Errorf("Expected day_type 'weekend', got '%s'", rule.DayType)
		}
	}
}

func TestGetRulesByDayType_Invalid(t *testing.T) {
	db := setupTestDBForBookingTime(t)
	defer db.Close()

	seedBookingTimeRules(t, db)
	repo := NewBookingTimeRepository(db)

	// Test invalid day type
	rules, err := repo.GetRulesByDayType("invalid")
	if err != nil {
		t.Fatalf("GetRulesByDayType failed: %v", err)
	}

	if len(rules) != 0 {
		t.Errorf("Expected 0 rules for invalid day_type, got %d", len(rules))
	}
}

// Test 2.1.2: CreateRule - Validation
func TestCreateRule_ValidWeekdayRule(t *testing.T) {
	db := setupTestDBForBookingTime(t)
	defer db.Close()

	repo := NewBookingTimeRepository(db)

	rule := &models.BookingTimeRule{
		DayType:   "weekday",
		RuleName:  "Test Rule",
		StartTime: "10:00",
		EndTime:   "11:00",
		IsBlocked: false,
	}

	err := repo.CreateRule(rule)
	if err != nil {
		t.Fatalf("CreateRule failed: %v", err)
	}

	// Verify ID was assigned
	if rule.ID == 0 {
		t.Error("Expected ID to be assigned, got 0")
	}

	// Verify rule was created
	rules, err := repo.GetRulesByDayType("weekday")
	if err != nil {
		t.Fatalf("GetRulesByDayType failed: %v", err)
	}

	if len(rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(rules))
	}

	if rules[0].RuleName != "Test Rule" {
		t.Errorf("Expected rule name 'Test Rule', got '%s'", rules[0].RuleName)
	}
}

func TestCreateRule_DuplicateDayTypeAndName(t *testing.T) {
	db := setupTestDBForBookingTime(t)
	defer db.Close()

	seedBookingTimeRules(t, db)
	repo := NewBookingTimeRepository(db)

	// Try to create duplicate (weekday, Morgenspaziergang)
	rule := &models.BookingTimeRule{
		DayType:   "weekday",
		RuleName:  "Morgenspaziergang",
		StartTime: "08:00",
		EndTime:   "10:00",
		IsBlocked: false,
	}

	err := repo.CreateRule(rule)
	if err == nil {
		t.Error("Expected error for duplicate (day_type, rule_name), got nil")
	}
}

// Test 2.1.3: UpdateRule - Modification
func TestUpdateRule_ChangeStartTime(t *testing.T) {
	db := setupTestDBForBookingTime(t)
	defer db.Close()

	seedBookingTimeRules(t, db)
	repo := NewBookingTimeRepository(db)

	// Get first rule
	rules, _ := repo.GetRulesByDayType("weekday")
	if len(rules) == 0 {
		t.Fatal("No rules found")
	}

	originalRule := rules[0]

	// Update start time
	updatedRule := &models.BookingTimeRule{
		StartTime: "08:30",
		EndTime:   originalRule.EndTime,
		IsBlocked: originalRule.IsBlocked,
	}

	err := repo.UpdateRule(originalRule.ID, updatedRule)
	if err != nil {
		t.Fatalf("UpdateRule failed: %v", err)
	}

	// Verify update
	rules, _ = repo.GetRulesByDayType("weekday")
	found := false
	for _, rule := range rules {
		if rule.ID == originalRule.ID {
			found = true
			if rule.StartTime != "08:30" {
				t.Errorf("Expected start_time '08:30', got '%s'", rule.StartTime)
			}
		}
	}

	if !found {
		t.Error("Updated rule not found")
	}
}

func TestUpdateRule_ToggleIsBlocked(t *testing.T) {
	db := setupTestDBForBookingTime(t)
	defer db.Close()

	seedBookingTimeRules(t, db)
	repo := NewBookingTimeRepository(db)

	// Get a blocked rule
	rules, _ := repo.GetRulesByDayType("weekday")
	var blockedRule *models.BookingTimeRule
	for i, rule := range rules {
		if rule.IsBlocked {
			blockedRule = &rules[i]
			break
		}
	}

	if blockedRule == nil {
		t.Fatal("No blocked rule found")
	}

	// Toggle is_blocked
	updatedRule := &models.BookingTimeRule{
		StartTime: blockedRule.StartTime,
		EndTime:   blockedRule.EndTime,
		IsBlocked: false, // Toggle to false
	}

	err := repo.UpdateRule(blockedRule.ID, updatedRule)
	if err != nil {
		t.Fatalf("UpdateRule failed: %v", err)
	}

	// Verify update
	rules, _ = repo.GetRulesByDayType("weekday")
	for _, rule := range rules {
		if rule.ID == blockedRule.ID {
			if rule.IsBlocked {
				t.Error("Expected is_blocked to be false, got true")
			}
		}
	}
}

func TestUpdateRule_NonExistentID(t *testing.T) {
	db := setupTestDBForBookingTime(t)
	defer db.Close()

	seedBookingTimeRules(t, db)
	repo := NewBookingTimeRepository(db)

	// Try to update non-existent rule
	updatedRule := &models.BookingTimeRule{
		StartTime: "10:00",
		EndTime:   "11:00",
		IsBlocked: false,
	}

	err := repo.UpdateRule(9999, updatedRule)
	if err != nil {
		t.Fatalf("UpdateRule with non-existent ID should not error, got: %v", err)
	}

	// Verify no rows affected (but no error)
	// This is expected SQLite behavior - UPDATE with no matching rows succeeds
}

// Test 2.1.4: DeleteRule - Removal
func TestDeleteRule_ExistingID(t *testing.T) {
	db := setupTestDBForBookingTime(t)
	defer db.Close()

	seedBookingTimeRules(t, db)
	repo := NewBookingTimeRepository(db)

	// Get count before delete
	rulesBefore, _ := repo.GetRulesByDayType("weekday")
	countBefore := len(rulesBefore)

	if countBefore == 0 {
		t.Fatal("No rules to delete")
	}

	// Delete first rule
	err := repo.DeleteRule(rulesBefore[0].ID)
	if err != nil {
		t.Fatalf("DeleteRule failed: %v", err)
	}

	// Verify deletion
	rulesAfter, _ := repo.GetRulesByDayType("weekday")
	countAfter := len(rulesAfter)

	if countAfter != countBefore-1 {
		t.Errorf("Expected %d rules after delete, got %d", countBefore-1, countAfter)
	}

	// Verify rule no longer exists
	for _, rule := range rulesAfter {
		if rule.ID == rulesBefore[0].ID {
			t.Error("Deleted rule still exists")
		}
	}
}

func TestDeleteRule_NonExistentID(t *testing.T) {
	db := setupTestDBForBookingTime(t)
	defer db.Close()

	seedBookingTimeRules(t, db)
	repo := NewBookingTimeRepository(db)

	// Delete non-existent rule
	err := repo.DeleteRule(9999)
	if err != nil {
		t.Fatalf("DeleteRule with non-existent ID should not error, got: %v", err)
	}

	// Verify count unchanged
	rules, _ := repo.GetRulesByDayType("weekday")
	if len(rules) != 5 {
		t.Errorf("Expected 5 rules (unchanged), got %d", len(rules))
	}
}

func TestDeleteRule_IDZero(t *testing.T) {
	db := setupTestDBForBookingTime(t)
	defer db.Close()

	seedBookingTimeRules(t, db)
	repo := NewBookingTimeRepository(db)

	// Delete with ID = 0
	err := repo.DeleteRule(0)
	if err != nil {
		t.Fatalf("DeleteRule with ID=0 should not error, got: %v", err)
	}

	// Verify count unchanged
	rules, _ := repo.GetRulesByDayType("weekday")
	if len(rules) != 5 {
		t.Errorf("Expected 5 rules (unchanged), got %d", len(rules))
	}
}

// Test GetAllRules - Additional coverage
func TestGetAllRules_GroupedByDayType(t *testing.T) {
	db := setupTestDBForBookingTime(t)
	defer db.Close()

	seedBookingTimeRules(t, db)
	repo := NewBookingTimeRepository(db)

	rules, err := repo.GetAllRules()
	if err != nil {
		t.Fatalf("GetAllRules failed: %v", err)
	}

	// Verify grouped by day type
	if _, hasWeekday := rules["weekday"]; !hasWeekday {
		t.Error("Expected 'weekday' key in results")
	}

	if _, hasWeekend := rules["weekend"]; !hasWeekend {
		t.Error("Expected 'weekend' key in results")
	}

	// Verify counts
	if len(rules["weekday"]) != 5 {
		t.Errorf("Expected 5 weekday rules, got %d", len(rules["weekday"]))
	}

	if len(rules["weekend"]) != 4 {
		t.Errorf("Expected 4 weekend rules, got %d", len(rules["weekend"]))
	}
}

// Test timestamps
func TestCreateRule_TimestampsSet(t *testing.T) {
	db := setupTestDBForBookingTime(t)
	defer db.Close()

	repo := NewBookingTimeRepository(db)

	before := time.Now().Add(-1 * time.Second) // Allow 1 second buffer

	rule := &models.BookingTimeRule{
		DayType:   "weekday",
		RuleName:  "Timestamp Test",
		StartTime: "10:00",
		EndTime:   "11:00",
		IsBlocked: false,
	}

	err := repo.CreateRule(rule)
	if err != nil {
		t.Fatalf("CreateRule failed: %v", err)
	}

	after := time.Now().Add(1 * time.Second) // Allow 1 second buffer

	// Retrieve the rule to check timestamps via database query
	var createdAt, updatedAt time.Time
	err = db.QueryRow(`
		SELECT created_at, updated_at FROM booking_time_rules WHERE id = ?
	`, rule.ID).Scan(&createdAt, &updatedAt)

	if err != nil {
		t.Fatalf("Failed to query timestamps: %v", err)
	}

	// Verify timestamps are set (within reasonable bounds with buffer)
	if createdAt.Before(before) || createdAt.After(after) {
		t.Errorf("CreatedAt timestamp not in expected range. Expected between %v and %v, got %v", before, after, createdAt)
	}

	if updatedAt.Before(before) || updatedAt.After(after) {
		t.Errorf("UpdatedAt timestamp not in expected range. Expected between %v and %v, got %v", before, after, updatedAt)
	}
}
