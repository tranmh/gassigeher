package repository

import (
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
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
			name, breed, size, age, category, color_id, photo, photo_thumbnail, special_needs,
			pickup_location, walk_route, walk_duration, special_instructions,
			default_morning_time, default_evening_time, is_available, external_link
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(
		query,
		dog.Name,
		dog.Breed,
		dog.Size,
		dog.Age,
		dog.Category,
		dog.ColorID,
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
		dog.ExternalLink,
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
		SELECT id, name, breed, size, age, category, color_id, photo, photo_thumbnail, special_needs,
		       pickup_location, walk_route, walk_duration, special_instructions,
		       default_morning_time, default_evening_time, is_available, is_featured,
		       external_link, unavailable_reason, unavailable_since, created_at, updated_at
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
		&dog.ColorID,
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
		&dog.IsFeatured,
		&dog.ExternalLink,
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
		SELECT id, name, breed, size, age, category, color_id, photo, photo_thumbnail, special_needs,
		       pickup_location, walk_route, walk_duration, special_instructions,
		       default_morning_time, default_evening_time, is_available, is_featured,
		       external_link, unavailable_reason, unavailable_since, created_at, updated_at
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
			&dog.ColorID,
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
			&dog.IsFeatured,
			&dog.ExternalLink,
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

// GetFeatured returns up to 3 randomly selected featured dogs that are available
// If more than 3 dogs are featured, a random selection of 3 is returned
func (r *DogRepository) GetFeatured() ([]*models.Dog, error) {
	query := `
		SELECT id, name, breed, size, age, category, color_id, photo, photo_thumbnail, special_needs,
		       pickup_location, walk_route, walk_duration, special_instructions,
		       default_morning_time, default_evening_time, is_available, is_featured,
		       external_link, unavailable_reason, unavailable_since, created_at, updated_at
		FROM dogs
		WHERE is_featured = 1 AND is_available = 1
		ORDER BY name ASC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query featured dogs: %w", err)
	}
	defer rows.Close()

	allFeatured := []*models.Dog{}
	for rows.Next() {
		dog := &models.Dog{}
		err := rows.Scan(
			&dog.ID,
			&dog.Name,
			&dog.Breed,
			&dog.Size,
			&dog.Age,
			&dog.Category,
			&dog.ColorID,
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
			&dog.IsFeatured,
			&dog.ExternalLink,
			&dog.UnavailableReason,
			&dog.UnavailableSince,
			&dog.CreatedAt,
			&dog.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan featured dog: %w", err)
		}
		allFeatured = append(allFeatured, dog)
	}

	// If 3 or fewer, return all
	if len(allFeatured) <= 3 {
		return allFeatured, nil
	}

	// Randomly select 3 from all featured dogs
	rand.Shuffle(len(allFeatured), func(i, j int) {
		allFeatured[i], allFeatured[j] = allFeatured[j], allFeatured[i]
	})

	return allFeatured[:3], nil
}

// SetFeatured sets the featured status for a dog
func (r *DogRepository) SetFeatured(id int, isFeatured bool) error {
	query := `UPDATE dogs SET is_featured = ?, updated_at = ? WHERE id = ?`

	_, err := r.db.Exec(query, isFeatured, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to set featured status: %w", err)
	}

	return nil
}

// CountFeatured returns the number of featured dogs
func (r *DogRepository) CountFeatured() (int, error) {
	query := `SELECT COUNT(*) FROM dogs WHERE is_featured = 1`

	var count int
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count featured dogs: %w", err)
	}

	return count, nil
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
			color_id = ?,
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
			external_link = ?,
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
		dog.ColorID,
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
		dog.ExternalLink,
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
	// Use Go time instead of database-specific date('now') for portability
	currentDate := time.Now().Format("2006-01-02")
	checkQuery := `
		SELECT COUNT(*) FROM bookings
		WHERE dog_id = ? AND date >= ? AND status = 'scheduled'
	`

	var count int
	err := r.db.QueryRow(checkQuery, id, currentDate).Scan(&count)
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

// ForceDelete deletes a dog and cancels all future bookings
func (r *DogRepository) ForceDelete(id int) error {
	// Delete the dog (bookings will remain but dog will be gone)
	deleteQuery := `DELETE FROM dogs WHERE id = ?`
	_, err := r.db.Exec(deleteQuery, id)
	if err != nil {
		return fmt.Errorf("failed to delete dog: %w", err)
	}

	return nil
}

// GetFutureBookings returns all future bookings for a dog with user details
func (r *DogRepository) GetFutureBookings(dogID int) ([]*models.Booking, error) {
	currentDate := time.Now().Format("2006-01-02")
	query := `
		SELECT
			b.id, b.user_id, b.dog_id, b.date, b.scheduled_time, b.status,
			b.completed_at, b.user_notes, b.admin_cancellation_reason, b.created_at, b.updated_at,
			u.first_name as user_first_name, u.last_name as user_last_name, u.email as user_email
		FROM bookings b
		LEFT JOIN users u ON b.user_id = u.id
		WHERE b.dog_id = ? AND b.date >= ? AND b.status = 'scheduled'
		ORDER BY b.date ASC, b.scheduled_time ASC
	`

	rows, err := r.db.Query(query, dogID, currentDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query future bookings: %w", err)
	}
	defer rows.Close()

	bookings := []*models.Booking{}
	for rows.Next() {
		booking := &models.Booking{
			User: &models.User{},
		}
		var userFirstName, userLastName, userEmail sql.NullString

		err := rows.Scan(
			&booking.ID,
			&booking.UserID,
			&booking.DogID,
			&booking.Date,
			&booking.ScheduledTime,
			&booking.Status,
			&booking.CompletedAt,
			&booking.UserNotes,
			&booking.AdminCancellationReason,
			&booking.CreatedAt,
			&booking.UpdatedAt,
			&userFirstName,
			&userLastName,
			&userEmail,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan booking: %w", err)
		}

		// Populate user details
		if userFirstName.Valid {
			booking.User.FirstName = userFirstName.String
		} else {
			booking.User.FirstName = "Deleted"
		}
		if userLastName.Valid {
			booking.User.LastName = userLastName.String
		} else {
			booking.User.LastName = "User"
		}
		if userEmail.Valid {
			email := userEmail.String
			booking.User.Email = &email
		}

		bookings = append(bookings, booking)
	}

	return bookings, nil
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
// DEPRECATED: Use CanUserAccessDogByColor for the new color-based system
func CanUserAccessDog(userLevel, dogCategory string) bool {
	// Define level hierarchy: green < orange < blue
	levelOrder := map[string]int{
		"green":  1,
		"orange": 2,
		"blue":   3,
	}

	userLevelNum, userOk := levelOrder[strings.ToLower(userLevel)]
	dogLevelNum, dogOk := levelOrder[strings.ToLower(dogCategory)]

	if !userOk || !dogOk {
		return false
	}

	// User can access dog if their level is >= dog's required level
	return userLevelNum >= dogLevelNum
}

// CanUserAccessDogByColor checks if a user can access a dog based on their assigned colors
// This is the new non-hierarchical color-based access control system
// Returns true if the user has the dog's required color
func CanUserAccessDogByColor(userColorIDs []int, dogColorID int) bool {
	// Dog must have a valid color ID
	if dogColorID <= 0 {
		return false
	}

	// User must have at least one color
	if userColorIDs == nil || len(userColorIDs) == 0 {
		return false
	}

	// Check if user has the dog's color
	for _, colorID := range userColorIDs {
		if colorID == dogColorID {
			return true
		}
	}

	return false
}
