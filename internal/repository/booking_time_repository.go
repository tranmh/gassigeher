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

// GetRulesByDayType returns all rules for a specific day type within a tenant
func (r *BookingTimeRepository) GetRulesByDayType(tenantID int, dayType string) ([]models.BookingTimeRule, error) {
	query := `
		SELECT id, tenant_id, day_type, rule_name, start_time, end_time, is_blocked, created_at, updated_at
		FROM booking_time_rules
		WHERE day_type = ? AND tenant_id = ?
		ORDER BY start_time ASC
	`

	rows, err := r.db.Query(query, dayType, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []models.BookingTimeRule
	for rows.Next() {
		var rule models.BookingTimeRule
		var isBlocked int

		err := rows.Scan(
			&rule.ID, &rule.TenantID, &rule.DayType, &rule.RuleName,
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

// GetAllRules returns all time rules grouped by day type for a tenant
func (r *BookingTimeRepository) GetAllRules(tenantID int) (map[string][]models.BookingTimeRule, error) {
	query := `
		SELECT id, tenant_id, day_type, rule_name, start_time, end_time, is_blocked, created_at, updated_at
		FROM booking_time_rules
		WHERE tenant_id = ?
		ORDER BY day_type, start_time ASC
	`

	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]models.BookingTimeRule)

	for rows.Next() {
		var rule models.BookingTimeRule
		var isBlocked int

		err := rows.Scan(
			&rule.ID, &rule.TenantID, &rule.DayType, &rule.RuleName,
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

// UpdateRule updates a time rule within a tenant
func (r *BookingTimeRepository) UpdateRule(tenantID int, id int, rule *models.BookingTimeRule) error {
	query := `
		UPDATE booking_time_rules
		SET start_time = ?, end_time = ?, is_blocked = ?, updated_at = ?
		WHERE id = ? AND tenant_id = ?
	`

	isBlocked := 0
	if rule.IsBlocked {
		isBlocked = 1
	}

	_, err := r.db.Exec(query, rule.StartTime, rule.EndTime, isBlocked, time.Now(), id, tenantID)
	return err
}

// CreateRule creates a new time rule for a tenant
func (r *BookingTimeRepository) CreateRule(tenantID int, rule *models.BookingTimeRule) error {
	query := `
		INSERT INTO booking_time_rules (tenant_id, day_type, rule_name, start_time, end_time, is_blocked)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	isBlocked := 0
	if rule.IsBlocked {
		isBlocked = 1
	}

	result, err := r.db.Exec(query, tenantID, rule.DayType, rule.RuleName, rule.StartTime, rule.EndTime, isBlocked)
	if err != nil {
		return err
	}

	id, _ := result.LastInsertId()
	rule.ID = int(id)
	rule.TenantID = tenantID
	return nil
}

// DeleteRule deletes a time rule within a tenant
func (r *BookingTimeRepository) DeleteRule(tenantID int, id int) error {
	query := `DELETE FROM booking_time_rules WHERE id = ? AND tenant_id = ?`
	_, err := r.db.Exec(query, id, tenantID)
	return err
}
