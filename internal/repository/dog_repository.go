package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/tranm/gassigeher/internal/models"
)

// DogRepository handles dog database operations
type DogRepository struct {
	db *sql.DB
}

// NewDogRepository creates a new dog repository
func NewDogRepository(db *sql.DB) *DogRepository {
	return &DogRepository{db: db}
}

// Create creates a new dog
func (r *DogRepository) Create(dog *models.Dog) error {
	query := `
		INSERT INTO dogs (
			name, breed, size, age, category, photo, photo_thumbnail, special_needs,
			pickup_location, walk_route, walk_duration, special_instructions,
			default_morning_time, default_evening_time, is_available
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(
		query,
		dog.Name,
		dog.Breed,
		dog.Size,
		dog.Age,
		dog.Category,
		dog.Photo,
		dog.PhotoThumbnail,
		dog.SpecialNeeds,
		dog.PickupLocation,
		dog.WalkRoute,
		dog.WalkDuration,
		dog.SpecialInstructions,
		dog.DefaultMorningTime,
		dog.DefaultEveningTime,
		dog.IsAvailable,
	)
	if err != nil {
		return fmt.Errorf("failed to create dog: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get dog ID: %w", err)
	}

	dog.ID = int(id)
	dog.CreatedAt = time.Now()
	dog.UpdatedAt = time.Now()
	return nil
}

// FindByID finds a dog by ID
func (r *DogRepository) FindByID(id int) (*models.Dog, error) {
	query := `
		SELECT id, name, breed, size, age, category, photo, photo_thumbnail, special_needs,
		       pickup_location, walk_route, walk_duration, special_instructions,
		       default_morning_time, default_evening_time, is_available,
		       unavailable_reason, unavailable_since, created_at, updated_at
		FROM dogs
		WHERE id = ?
	`

	dog := &models.Dog{}
	err := r.db.QueryRow(query, id).Scan(
		&dog.ID,
		&dog.Name,
		&dog.Breed,
		&dog.Size,
		&dog.Age,
		&dog.Category,
		&dog.Photo,
		&dog.PhotoThumbnail,
		&dog.SpecialNeeds,
		&dog.PickupLocation,
		&dog.WalkRoute,
		&dog.WalkDuration,
		&dog.SpecialInstructions,
		&dog.DefaultMorningTime,
		&dog.DefaultEveningTime,
		&dog.IsAvailable,
		&dog.UnavailableReason,
		&dog.UnavailableSince,
		&dog.CreatedAt,
		&dog.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find dog: %w", err)
	}

	return dog, nil
}

// FindAll finds all dogs with optional filtering
func (r *DogRepository) FindAll(filter *models.DogFilterRequest) ([]*models.Dog, error) {
	query := `
		SELECT id, name, breed, size, age, category, photo, photo_thumbnail, special_needs,
		       pickup_location, walk_route, walk_duration, special_instructions,
		       default_morning_time, default_evening_time, is_available,
		       unavailable_reason, unavailable_since, created_at, updated_at
		FROM dogs
		WHERE 1=1
	`

	args := []interface{}{}

	// Apply filters
	if filter != nil {
		if filter.Breed != nil && *filter.Breed != "" {
			query += " AND LOWER(breed) = LOWER(?)"
			args = append(args, *filter.Breed)
		}

		if filter.Size != nil && *filter.Size != "" {
			query += " AND size = ?"
			args = append(args, *filter.Size)
		}

		if filter.MinAge != nil {
			query += " AND age >= ?"
			args = append(args, *filter.MinAge)
		}

		if filter.MaxAge != nil {
			query += " AND age <= ?"
			args = append(args, *filter.MaxAge)
		}

		if filter.Category != nil && *filter.Category != "" {
			query += " AND category = ?"
			args = append(args, *filter.Category)
		}

		if filter.Available != nil {
			query += " AND is_available = ?"
			args = append(args, *filter.Available)
		}

		if filter.Search != nil && *filter.Search != "" {
			query += " AND (LOWER(name) LIKE LOWER(?) OR LOWER(breed) LIKE LOWER(?))"
			searchTerm := "%" + *filter.Search + "%"
			args = append(args, searchTerm, searchTerm)
		}
	}

	query += " ORDER BY name ASC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query dogs: %w", err)
	}
	defer rows.Close()

	dogs := []*models.Dog{}
	for rows.Next() {
		dog := &models.Dog{}
		err := rows.Scan(
			&dog.ID,
			&dog.Name,
			&dog.Breed,
			&dog.Size,
			&dog.Age,
			&dog.Category,
			&dog.Photo,
			&dog.PhotoThumbnail,
			&dog.SpecialNeeds,
			&dog.PickupLocation,
			&dog.WalkRoute,
			&dog.WalkDuration,
			&dog.SpecialInstructions,
			&dog.DefaultMorningTime,
			&dog.DefaultEveningTime,
			&dog.IsAvailable,
			&dog.UnavailableReason,
			&dog.UnavailableSince,
			&dog.CreatedAt,
			&dog.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan dog: %w", err)
		}
		dogs = append(dogs, dog)
	}

	return dogs, nil
}

// Update updates a dog
func (r *DogRepository) Update(dog *models.Dog) error {
	query := `
		UPDATE dogs SET
			name = ?,
			breed = ?,
			size = ?,
			age = ?,
			category = ?,
			photo = ?,
			photo_thumbnail = ?,
			special_needs = ?,
			pickup_location = ?,
			walk_route = ?,
			walk_duration = ?,
			special_instructions = ?,
			default_morning_time = ?,
			default_evening_time = ?,
			is_available = ?,
			unavailable_reason = ?,
			unavailable_since = ?,
			updated_at = ?
		WHERE id = ?
	`

	_, err := r.db.Exec(
		query,
		dog.Name,
		dog.Breed,
		dog.Size,
		dog.Age,
		dog.Category,
		dog.Photo,
		dog.PhotoThumbnail,
		dog.SpecialNeeds,
		dog.PickupLocation,
		dog.WalkRoute,
		dog.WalkDuration,
		dog.SpecialInstructions,
		dog.DefaultMorningTime,
		dog.DefaultEveningTime,
		dog.IsAvailable,
		dog.UnavailableReason,
		dog.UnavailableSince,
		time.Now(),
		dog.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update dog: %w", err)
	}

	return nil
}

// Delete deletes a dog (only if no future bookings exist)
func (r *DogRepository) Delete(id int) error {
	// Check for future bookings
	checkQuery := `
		SELECT COUNT(*) FROM bookings
		WHERE dog_id = ? AND date >= date('now') AND status = 'scheduled'
	`

	var count int
	err := r.db.QueryRow(checkQuery, id).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check bookings: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("cannot delete dog with future bookings")
	}

	// Delete the dog
	deleteQuery := `DELETE FROM dogs WHERE id = ?`
	_, err = r.db.Exec(deleteQuery, id)
	if err != nil {
		return fmt.Errorf("failed to delete dog: %w", err)
	}

	return nil
}

// ToggleAvailability toggles a dog's availability status
func (r *DogRepository) ToggleAvailability(id int, isAvailable bool, reason *string) error {
	var query string
	var args []interface{}

	if isAvailable {
		// Mark as available (clear reason and timestamp)
		query = `
			UPDATE dogs SET
				is_available = 1,
				unavailable_reason = NULL,
				unavailable_since = NULL,
				updated_at = ?
			WHERE id = ?
		`
		args = []interface{}{time.Now(), id}
	} else {
		// Mark as unavailable
		query = `
			UPDATE dogs SET
				is_available = 0,
				unavailable_reason = ?,
				unavailable_since = ?,
				updated_at = ?
			WHERE id = ?
		`
		now := time.Now()
		args = []interface{}{reason, now, now, id}
	}

	_, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to toggle availability: %w", err)
	}

	return nil
}

// GetBreeds returns a list of unique breeds
func (r *DogRepository) GetBreeds() ([]string, error) {
	query := `SELECT DISTINCT breed FROM dogs ORDER BY breed ASC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get breeds: %w", err)
	}
	defer rows.Close()

	breeds := []string{}
	for rows.Next() {
		var breed string
		if err := rows.Scan(&breed); err != nil {
			return nil, fmt.Errorf("failed to scan breed: %w", err)
		}
		breeds = append(breeds, breed)
	}

	return breeds, nil
}

// CanUserAccessDog checks if a user can access a dog based on their experience level
func CanUserAccessDog(userLevel, dogCategory string) bool {
	// Define level hierarchy: green < blue < orange
	levelOrder := map[string]int{
		"green":  1,
		"blue":   2,
		"orange": 3,
	}

	userLevelNum, userOk := levelOrder[strings.ToLower(userLevel)]
	dogLevelNum, dogOk := levelOrder[strings.ToLower(dogCategory)]

	if !userOk || !dogOk {
		return false
	}

	// User can access dog if their level is >= dog's required level
	return userLevelNum >= dogLevelNum
}
