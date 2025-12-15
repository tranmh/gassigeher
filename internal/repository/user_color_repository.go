package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
)

// UserColorRepository handles user-color relationship database operations
type UserColorRepository struct {
	db *sql.DB
}

// NewUserColorRepository creates a new user color repository
func NewUserColorRepository(db *sql.DB) *UserColorRepository {
	return &UserColorRepository{db: db}
}

// AddColorToUser adds a color to a user
func (r *UserColorRepository) AddColorToUser(userID, colorID, grantedBy int) error {
	query := `
		INSERT INTO user_colors (user_id, color_id, granted_at, granted_by)
		VALUES (?, ?, ?, ?)
	`

	now := time.Now()
	_, err := r.db.Exec(query, userID, colorID, now, grantedBy)
	if err != nil {
		return fmt.Errorf("failed to add color to user: %w", err)
	}

	return nil
}

// RemoveColorFromUser removes a color from a user
func (r *UserColorRepository) RemoveColorFromUser(userID, colorID int) error {
	query := `DELETE FROM user_colors WHERE user_id = ? AND color_id = ?`

	_, err := r.db.Exec(query, userID, colorID)
	if err != nil {
		return fmt.Errorf("failed to remove color from user: %w", err)
	}

	return nil
}

// GetUserColors returns all colors assigned to a user
func (r *UserColorRepository) GetUserColors(userID int) ([]*models.ColorCategory, error) {
	query := `
		SELECT c.id, c.name, c.hex_code, c.pattern_icon, c.sort_order, c.created_at, c.updated_at
		FROM color_categories c
		INNER JOIN user_colors uc ON c.id = uc.color_id
		WHERE uc.user_id = ?
		ORDER BY c.sort_order ASC, c.name ASC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user colors: %w", err)
	}
	defer rows.Close()

	colors := []*models.ColorCategory{}
	for rows.Next() {
		color := &models.ColorCategory{}
		err := rows.Scan(
			&color.ID,
			&color.Name,
			&color.HexCode,
			&color.PatternIcon,
			&color.SortOrder,
			&color.CreatedAt,
			&color.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan color category: %w", err)
		}
		colors = append(colors, color)
	}

	return colors, nil
}

// GetUserColorIDs returns all color IDs assigned to a user
func (r *UserColorRepository) GetUserColorIDs(userID int) ([]int, error) {
	query := `SELECT color_id FROM user_colors WHERE user_id = ?`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user color IDs: %w", err)
	}
	defer rows.Close()

	colorIDs := []int{}
	for rows.Next() {
		var colorID int
		if err := rows.Scan(&colorID); err != nil {
			return nil, fmt.Errorf("failed to scan color ID: %w", err)
		}
		colorIDs = append(colorIDs, colorID)
	}

	return colorIDs, nil
}

// HasColor checks if a user has a specific color
func (r *UserColorRepository) HasColor(userID, colorID int) (bool, error) {
	query := `SELECT COUNT(*) FROM user_colors WHERE user_id = ? AND color_id = ?`

	var count int
	err := r.db.QueryRow(query, userID, colorID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check user color: %w", err)
	}

	return count > 0, nil
}

// SetUserColors replaces all colors for a user with the given list
func (r *UserColorRepository) SetUserColors(userID int, colorIDs []int, grantedBy int) error {
	// Start transaction
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Remove all existing colors
	_, err = tx.Exec("DELETE FROM user_colors WHERE user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("failed to remove existing colors: %w", err)
	}

	// Add new colors
	now := time.Now()
	for _, colorID := range colorIDs {
		_, err = tx.Exec(
			"INSERT INTO user_colors (user_id, color_id, granted_at, granted_by) VALUES (?, ?, ?, ?)",
			userID, colorID, now, grantedBy,
		)
		if err != nil {
			return fmt.Errorf("failed to add color %d: %w", colorID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetUserColorsWithDetails returns detailed user-color assignments
func (r *UserColorRepository) GetUserColorsWithDetails(userID int) ([]*models.UserColor, error) {
	query := `
		SELECT uc.id, uc.user_id, uc.color_id, uc.granted_at, uc.granted_by,
		       c.id, c.name, c.hex_code, c.pattern_icon, c.sort_order, c.created_at, c.updated_at
		FROM user_colors uc
		INNER JOIN color_categories c ON c.id = uc.color_id
		WHERE uc.user_id = ?
		ORDER BY c.sort_order ASC, c.name ASC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user colors with details: %w", err)
	}
	defer rows.Close()

	userColors := []*models.UserColor{}
	for rows.Next() {
		uc := &models.UserColor{
			Color: &models.ColorCategory{},
		}
		err := rows.Scan(
			&uc.ID,
			&uc.UserID,
			&uc.ColorID,
			&uc.GrantedAt,
			&uc.GrantedBy,
			&uc.Color.ID,
			&uc.Color.Name,
			&uc.Color.HexCode,
			&uc.Color.PatternIcon,
			&uc.Color.SortOrder,
			&uc.Color.CreatedAt,
			&uc.Color.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user color: %w", err)
		}
		userColors = append(userColors, uc)
	}

	return userColors, nil
}
