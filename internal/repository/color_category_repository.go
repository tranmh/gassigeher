package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
)

// ColorCategoryRepository handles color category database operations
type ColorCategoryRepository struct {
	db *sql.DB
}

// NewColorCategoryRepository creates a new color category repository
func NewColorCategoryRepository(db *sql.DB) *ColorCategoryRepository {
	return &ColorCategoryRepository{db: db}
}

// Create creates a new color category
func (r *ColorCategoryRepository) Create(color *models.ColorCategory) error {
	query := `
		INSERT INTO color_categories (name, hex_code, pattern_icon, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.Exec(query, color.Name, color.HexCode, color.PatternIcon, color.SortOrder, now, now)
	if err != nil {
		return fmt.Errorf("failed to create color category: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get color ID: %w", err)
	}

	color.ID = int(id)
	color.CreatedAt = now
	color.UpdatedAt = now

	return nil
}

// FindByID finds a color category by ID
func (r *ColorCategoryRepository) FindByID(id int) (*models.ColorCategory, error) {
	query := `
		SELECT id, name, hex_code, pattern_icon, sort_order, created_at, updated_at
		FROM color_categories
		WHERE id = ?
	`

	color := &models.ColorCategory{}
	err := r.db.QueryRow(query, id).Scan(
		&color.ID,
		&color.Name,
		&color.HexCode,
		&color.PatternIcon,
		&color.SortOrder,
		&color.CreatedAt,
		&color.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find color category: %w", err)
	}

	return color, nil
}

// FindByName finds a color category by name
func (r *ColorCategoryRepository) FindByName(name string) (*models.ColorCategory, error) {
	query := `
		SELECT id, name, hex_code, pattern_icon, sort_order, created_at, updated_at
		FROM color_categories
		WHERE name = ?
	`

	color := &models.ColorCategory{}
	err := r.db.QueryRow(query, name).Scan(
		&color.ID,
		&color.Name,
		&color.HexCode,
		&color.PatternIcon,
		&color.SortOrder,
		&color.CreatedAt,
		&color.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find color category by name: %w", err)
	}

	return color, nil
}

// FindAll returns all color categories ordered by sort_order
func (r *ColorCategoryRepository) FindAll() ([]*models.ColorCategory, error) {
	query := `
		SELECT id, name, hex_code, pattern_icon, sort_order, created_at, updated_at
		FROM color_categories
		ORDER BY sort_order ASC, name ASC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query color categories: %w", err)
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

// Update updates a color category
func (r *ColorCategoryRepository) Update(color *models.ColorCategory) error {
	query := `
		UPDATE color_categories
		SET name = ?, hex_code = ?, pattern_icon = ?, sort_order = ?, updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	_, err := r.db.Exec(query, color.Name, color.HexCode, color.PatternIcon, color.SortOrder, now, color.ID)
	if err != nil {
		return fmt.Errorf("failed to update color category: %w", err)
	}

	color.UpdatedAt = now
	return nil
}

// Delete deletes a color category (fails if dogs are assigned)
func (r *ColorCategoryRepository) Delete(id int) error {
	// Check if any dogs are assigned to this color
	count, err := r.CountDogsWithColor(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("cannot delete color category: %d dogs are assigned to this color", count)
	}

	query := `DELETE FROM color_categories WHERE id = ?`
	_, err = r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete color category: %w", err)
	}

	return nil
}

// Count returns the total number of color categories
func (r *ColorCategoryRepository) Count() (int, error) {
	query := `SELECT COUNT(*) FROM color_categories`

	var count int
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count color categories: %w", err)
	}

	return count, nil
}

// CountDogsWithColor returns the number of dogs assigned to a color
func (r *ColorCategoryRepository) CountDogsWithColor(colorID int) (int, error) {
	query := `SELECT COUNT(*) FROM dogs WHERE color_id = ?`

	var count int
	err := r.db.QueryRow(query, colorID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count dogs with color: %w", err)
	}

	return count, nil
}

// CountUsersWithColor returns the number of users who have a color
func (r *ColorCategoryRepository) CountUsersWithColor(colorID int) (int, error) {
	query := `SELECT COUNT(*) FROM user_colors WHERE color_id = ?`

	var count int
	err := r.db.QueryRow(query, colorID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users with color: %w", err)
	}

	return count, nil
}

// GetNextSortOrder returns the next available sort order
func (r *ColorCategoryRepository) GetNextSortOrder() (int, error) {
	query := `SELECT COALESCE(MAX(sort_order), 0) + 1 FROM color_categories`

	var nextOrder int
	err := r.db.QueryRow(query).Scan(&nextOrder)
	if err != nil {
		return 0, fmt.Errorf("failed to get next sort order: %w", err)
	}

	return nextOrder, nil
}
