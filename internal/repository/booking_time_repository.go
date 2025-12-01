package repository

import (
	"database/sql"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
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
