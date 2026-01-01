package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
)

// ColorCategoryRepository handles color category database operations
type ColorCategoryRepository struct {
	db DBExecutor
}

// NewColorCategoryRepository creates a new color category repository
func NewColorCategoryRepository(db DBExecutor) *ColorCategoryRepository {
	return &ColorCategoryRepository{db: db}
}

// Create creates a new color category
func (r *ColorCategoryRepository) Create(tenantID int, color *models.ColorCategory) error {
	query := `
		INSERT INTO color_categories (tenant_id, name, hex_code, pattern_icon, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	id, err := r.db.InsertReturningID(query, tenantID, color.Name, color.HexCode, color.PatternIcon, color.SortOrder, now, now)
	if err != nil {
		return fmt.Errorf("failed to create color category: %w", err)
	}

	color.ID = int(id)
	color.TenantID = tenantID
	color.CreatedAt = now
	color.UpdatedAt = now

	return nil
}

// FindByID finds a color category by ID within a tenant
func (r *ColorCategoryRepository) FindByID(tenantID int, id int) (*models.ColorCategory, error) {
	query := `
		SELECT id, tenant_id, name, hex_code, pattern_icon, sort_order, created_at, updated_at
		FROM color_categories
		WHERE id = ? AND tenant_id = ?
	`

	color := &models.ColorCategory{}
	err := r.db.QueryRow(query, id, tenantID).Scan(
		&color.ID,
		&color.TenantID,
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

// FindByName finds a color category by name within a tenant (case-insensitive)
func (r *ColorCategoryRepository) FindByName(tenantID int, name string) (*models.ColorCategory, error) {
	query := `
		SELECT id, tenant_id, name, hex_code, pattern_icon, sort_order, created_at, updated_at
		FROM color_categories
		WHERE LOWER(name) = LOWER(?) AND tenant_id = ?
	`

	color := &models.ColorCategory{}
	err := r.db.QueryRow(query, name, tenantID).Scan(
		&color.ID,
		&color.TenantID,
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

// legacyCategoryToColorNames maps legacy category names (green/blue/orange) to German color names
// This supports both uppercase and lowercase variations found in the database
var legacyCategoryToColorNames = map[string][]string{
	"green":  {"Grün", "Gruen", "grün", "gruen"},
	"blue":   {"Blau", "blau", "Dunkelblau", "dunkelblau"},
	"orange": {"Orange", "orange"},
}

// FindByLegacyCategory finds a color category by legacy category name (green/blue/orange)
// This is used to map the legacy category field to the new color_id system
func (r *ColorCategoryRepository) FindByLegacyCategory(tenantID int, category string) (*models.ColorCategory, error) {
	colorNames, ok := legacyCategoryToColorNames[category]
	if !ok {
		return nil, nil // Unknown category
	}

	// Try each possible color name
	for _, name := range colorNames {
		color, err := r.FindByName(tenantID, name)
		if err != nil {
			return nil, err
		}
		if color != nil {
			return color, nil
		}
	}

	return nil, nil // No matching color found
}

// FindAll returns all color categories for a tenant ordered by sort_order
func (r *ColorCategoryRepository) FindAll(tenantID int) ([]*models.ColorCategory, error) {
	query := `
		SELECT id, tenant_id, name, hex_code, pattern_icon, sort_order, created_at, updated_at
		FROM color_categories
		WHERE tenant_id = ?
		ORDER BY sort_order ASC, name ASC
	`

	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query color categories: %w", err)
	}
	defer rows.Close()

	colors := []*models.ColorCategory{}
	for rows.Next() {
		color := &models.ColorCategory{}
		err := rows.Scan(
			&color.ID,
			&color.TenantID,
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

// Update updates a color category within a tenant
func (r *ColorCategoryRepository) Update(tenantID int, color *models.ColorCategory) error {
	query := `
		UPDATE color_categories
		SET name = ?, hex_code = ?, pattern_icon = ?, sort_order = ?, updated_at = ?
		WHERE id = ? AND tenant_id = ?
	`

	now := time.Now()
	_, err := r.db.Exec(query, color.Name, color.HexCode, color.PatternIcon, color.SortOrder, now, color.ID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update color category: %w", err)
	}

	color.UpdatedAt = now
	return nil
}

// Delete deletes a color category (fails if dogs are assigned) within a tenant
func (r *ColorCategoryRepository) Delete(tenantID int, id int) error {
	// Check if any dogs are assigned to this color
	count, err := r.CountDogsWithColor(tenantID, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("cannot delete color category: %d dogs are assigned to this color", count)
	}

	query := `DELETE FROM color_categories WHERE id = ? AND tenant_id = ?`
	_, err = r.db.Exec(query, id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete color category: %w", err)
	}

	return nil
}

// Count returns the total number of color categories for a tenant
func (r *ColorCategoryRepository) Count(tenantID int) (int, error) {
	query := `SELECT COUNT(*) FROM color_categories WHERE tenant_id = ?`

	var count int
	err := r.db.QueryRow(query, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count color categories: %w", err)
	}

	return count, nil
}

// CountDogsWithColor returns the number of dogs assigned to a color within a tenant
func (r *ColorCategoryRepository) CountDogsWithColor(tenantID int, colorID int) (int, error) {
	query := `SELECT COUNT(*) FROM dogs WHERE color_id = ? AND tenant_id = ?`

	var count int
	err := r.db.QueryRow(query, colorID, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count dogs with color: %w", err)
	}

	return count, nil
}

// CountUsersWithColor returns the number of users who have a color within a tenant
func (r *ColorCategoryRepository) CountUsersWithColor(tenantID int, colorID int) (int, error) {
	query := `SELECT COUNT(*) FROM user_colors WHERE color_id = ? AND tenant_id = ?`

	var count int
	err := r.db.QueryRow(query, colorID, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users with color: %w", err)
	}

	return count, nil
}

// GetNextSortOrder returns the next available sort order for a tenant
func (r *ColorCategoryRepository) GetNextSortOrder(tenantID int) (int, error) {
	query := `SELECT COALESCE(MAX(sort_order), 0) + 1 FROM color_categories WHERE tenant_id = ?`

	var nextOrder int
	err := r.db.QueryRow(query, tenantID).Scan(&nextOrder)
	if err != nil {
		return 0, fmt.Errorf("failed to get next sort order: %w", err)
	}

	return nextOrder, nil
}
