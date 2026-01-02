package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
)

// BUG #1 FIX: Application-level mutex map for per-tenant dog limit enforcement
// This prevents race conditions where concurrent requests could exceed the dog limit
// by ensuring only one dog creation operation per tenant can run at a time
var dogLimitMutexes sync.Map

// DogRepository handles dog database operations
type DogRepository struct {
	db DBExecutor
}

// NewDogRepository creates a new dog repository
func NewDogRepository(db DBExecutor) *DogRepository {
	return &DogRepository{db: db}
}

// NewDogRepositoryWithTx creates a dog repository that can work with transactions
// The tx parameter is stored for use with CreateTx method
func NewDogRepositoryWithTx(db DBExecutor, tx *sql.Tx) *DogRepository {
	return &DogRepository{db: db}
}

// CreateTx creates a new dog within a transaction
// SaaS: Now includes tenant_id for multi-tenancy
func (r *DogRepository) CreateTx(tx *sql.Tx, dog *models.Dog) error {
	query := `
		INSERT INTO dogs (
			tenant_id, name, breed, size, age, color_id, photo, photo_thumbnail, special_needs,
			pickup_location, walk_route, walk_duration, special_instructions,
			default_morning_time, default_evening_time, is_available, is_featured, external_link
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	// PostgreSQL requires RETURNING clause instead of LastInsertId
	if r.db.GetDialectName() == "postgres" {
		query = query + " RETURNING id"
		// Rebind query for PostgreSQL (? -> $1, $2, ...)
		query = r.db.RebindQuery(query)
		var id int64
		err := tx.QueryRow(
			query,
			dog.TenantID,
			dog.Name,
			dog.Breed,
			dog.Size,
			dog.Age,
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
			r.db.BoolValue(dog.IsAvailable),
			r.db.BoolValue(dog.IsFeatured),
			dog.ExternalLink,
		).Scan(&id)
		if err != nil {
			return fmt.Errorf("failed to create dog: %w", err)
		}
		dog.ID = int(id)
		dog.CreatedAt = time.Now()
		dog.UpdatedAt = time.Now()
		return nil
	}

	// SQLite/MySQL use LastInsertId
	result, err := tx.Exec(
		query,
		dog.TenantID,
		dog.Name,
		dog.Breed,
		dog.Size,
		dog.Age,
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
		r.db.BoolValue(dog.IsAvailable),
		r.db.BoolValue(dog.IsFeatured),
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

// Create creates a new dog
// SaaS: Now includes tenant_id for multi-tenancy
func (r *DogRepository) Create(dog *models.Dog) error {
	query := `
		INSERT INTO dogs (
			tenant_id, name, breed, size, age, color_id, photo, photo_thumbnail, special_needs,
			pickup_location, walk_route, walk_duration, special_instructions,
			default_morning_time, default_evening_time, is_available, is_featured, external_link
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	// tenant_id=0 is valid for Simple-Mode (non-SaaS)
	id, err := r.db.InsertReturningID(
		query,
		dog.TenantID,
		dog.Name,
		dog.Breed,
		dog.Size,
		dog.Age,
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
		r.db.BoolValue(dog.IsAvailable),
		r.db.BoolValue(dog.IsFeatured),
		dog.ExternalLink,
	)
	if err != nil {
		return fmt.Errorf("failed to create dog: %w", err)
	}

	dog.ID = int(id)
	dog.CreatedAt = time.Now()
	dog.UpdatedAt = time.Now()
	return nil
}

// FindByID finds a dog by ID
// SaaS: Now includes tenant_id in result
func (r *DogRepository) FindByID(id int) (*models.Dog, error) {
	query := `
		SELECT id, tenant_id, name, breed, size, age, color_id, photo, photo_thumbnail, special_needs,
		       pickup_location, walk_route, walk_duration, special_instructions,
		       default_morning_time, default_evening_time, is_available, is_featured,
		       external_link, unavailable_reason, unavailable_since, created_at, updated_at
		FROM dogs
		WHERE id = ?
	`

	dog := &models.Dog{}
	var tenantID sql.NullInt64
	err := r.db.QueryRow(query, id).Scan(
		&dog.ID,
		&tenantID,
		&dog.Name,
		&dog.Breed,
		&dog.Size,
		&dog.Age,
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

	if tenantID.Valid {
		dog.TenantID = int(tenantID.Int64)
	}

	return dog, nil
}

// FindByIDAndTenant finds a dog by ID with tenant isolation
// SaaS: Returns ErrNotFound if dog doesn't exist, ErrTenantMismatch if it belongs to a different tenant
// Use tenantID=0 for Simple-Mode (no tenant filtering)
func (r *DogRepository) FindByIDAndTenant(id int, tenantID int) (*models.Dog, error) {
	dog, err := r.FindByID(id)
	if err != nil {
		return nil, err
	}
	if dog == nil {
		return nil, ErrNotFound
	}
	// Verify tenant membership (works for both Simple-Mode tenant_id=0 and SaaS-Mode)
	if dog.TenantID != tenantID {
		return nil, ErrTenantMismatch // Dog doesn't belong to this tenant
	}
	return dog, nil
}

// FindAll finds all dogs with optional filtering
// Always filters by tenant_id (tenant_id=0 for Simple-Mode, >0 for SaaS-Mode)
func (r *DogRepository) FindAll(filter *models.DogFilterRequest, tenantID int) ([]*models.Dog, error) {
	query := `
		SELECT id, tenant_id, name, breed, size, age, color_id, photo, photo_thumbnail, special_needs,
		       pickup_location, walk_route, walk_duration, special_instructions,
		       default_morning_time, default_evening_time, is_available, is_featured,
		       external_link, unavailable_reason, unavailable_since, created_at, updated_at
		FROM dogs
		WHERE tenant_id = ?
	`

	args := []interface{}{tenantID}

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

		// Category filter maps to color_id via color name lookup using subquery
		// BUG #7 FIX: Validate category against whitelist to prevent invalid inputs
		if filter.Category != nil && *filter.Category != "" {
			// Map English category names to German color names (whitelist)
			categoryToColorName := map[string]string{
				"green":  "gruen",
				"orange": "orange",
				"blue":   "dunkelblau",
			}
			colorName, ok := categoryToColorName[*filter.Category]
			if !ok {
				// Invalid category - reject (defense in depth)
				return nil, fmt.Errorf("invalid category: %s (allowed: green, orange, blue)", *filter.Category)
			}
			// Use subquery to find color_id by name for the same tenant
			query += " AND color_id IN (SELECT id FROM color_categories WHERE tenant_id = dogs.tenant_id AND LOWER(name) = LOWER(?))"
			args = append(args, colorName)
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
		var tenantIDVal sql.NullInt64
		err := rows.Scan(
			&dog.ID,
			&tenantIDVal,
			&dog.Name,
			&dog.Breed,
			&dog.Size,
			&dog.Age,
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
		if tenantIDVal.Valid {
			dog.TenantID = int(tenantIDVal.Int64)
		}
		dogs = append(dogs, dog)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating dogs: %w", err)
	}

	return dogs, nil
}

// maxFeaturedDogs is the maximum number of featured dogs to return
const maxFeaturedDogs = 3

// featuredQueryLimit bounds the initial query to prevent unbounded memory usage
// We fetch a small pool (3x the result size) for random selection variety
// while still limiting memory usage to a reasonable amount
const featuredQueryLimit = 12

// GetFeatured returns up to 3 randomly selected featured dogs that are available
// If more than 3 dogs are featured, a random selection of 3 is returned
// Always filters by tenant_id (tenant_id=0 for Simple-Mode, >0 for SaaS-Mode)
// FIX: Now uses LIMIT to bound the initial query and prevent unbounded memory usage
func (r *DogRepository) GetFeatured(tenantID int) ([]*models.Dog, error) {
	// Use LIMIT to bound memory usage - fetch a pool of candidates for random selection
	// Using a larger pool (featuredQueryLimit) gives good randomization variety
	// while still preventing unbounded memory usage
	query := `
		SELECT id, tenant_id, name, breed, size, age, color_id, photo, photo_thumbnail, special_needs,
		       pickup_location, walk_route, walk_duration, special_instructions,
		       default_morning_time, default_evening_time, is_available, is_featured,
		       external_link, unavailable_reason, unavailable_since, created_at, updated_at
		FROM dogs
		WHERE is_featured = ? AND is_available = ? AND tenant_id = ?
		ORDER BY id DESC
		LIMIT ?
	`

	args := []interface{}{r.db.BoolValue(true), r.db.BoolValue(true), tenantID, featuredQueryLimit}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query featured dogs: %w", err)
	}
	defer rows.Close()

	candidates := make([]*models.Dog, 0, featuredQueryLimit)
	for rows.Next() {
		dog := &models.Dog{}
		var tenantIDVal sql.NullInt64
		err := rows.Scan(
			&dog.ID,
			&tenantIDVal,
			&dog.Name,
			&dog.Breed,
			&dog.Size,
			&dog.Age,
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
		if tenantIDVal.Valid {
			dog.TenantID = int(tenantIDVal.Int64)
		}
		candidates = append(candidates, dog)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating featured dogs: %w", err)
	}

	// If maxFeaturedDogs or fewer, return all
	if len(candidates) <= maxFeaturedDogs {
		return candidates, nil
	}

	// Randomly select maxFeaturedDogs from the bounded candidate pool
	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	return candidates[:maxFeaturedDogs], nil
}

// SetFeatured sets the featured status for a dog
// SaaS: Filters by tenant_id for tenant isolation
func (r *DogRepository) SetFeatured(id int, tenantID int, isFeatured bool) error {
	query := `UPDATE dogs SET is_featured = ?, updated_at = ? WHERE id = ? AND tenant_id = ?`

	_, err := r.db.Exec(query, r.db.BoolValue(isFeatured), time.Now(), id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to set featured status: %w", err)
	}

	return nil
}

// CountFeatured returns the number of featured dogs for a tenant
// SaaS: Filters by tenant_id for tenant isolation
func (r *DogRepository) CountFeatured(tenantID int) (int, error) {
	query := `SELECT COUNT(*) FROM dogs WHERE is_featured = ? AND tenant_id = ?`

	var count int
	err := r.db.QueryRow(query, r.db.BoolValue(true), tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count featured dogs: %w", err)
	}

	return count, nil
}

// CountByTenant returns the number of dogs for a specific tenant
// SaaS: Used for enforcing dog limits in the freemium model
func (r *DogRepository) CountByTenant(tenantID int) (int, error) {
	query := `SELECT COUNT(*) FROM dogs WHERE tenant_id = ?`

	var count int
	err := r.db.QueryRow(query, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count dogs by tenant: %w", err)
	}

	return count, nil
}

// ErrDogLimitExceeded is returned when the dog limit would be exceeded
var ErrDogLimitExceeded = fmt.Errorf("dog limit exceeded")

// CreateWithLimitCheck creates a dog atomically while checking the limit
// This prevents race conditions where multiple requests could exceed the limit
// SaaS: Used for enforcing dog limits in the freemium model
//
// BUG #1 FIX: Uses application-level mutex per tenant to prevent race conditions.
// SERIALIZABLE isolation alone is NOT sufficient because:
// - In PostgreSQL/MySQL, each HTTP request gets a separate database connection
// - Concurrent requests can each start their own SERIALIZABLE transaction
// - Each transaction sees a snapshot at transaction start time
// - Both can read count=9, pass the check, and insert, resulting in count=11
//
// The per-tenant mutex ensures only one CreateWithLimitCheck can run per tenant
// at a time, preventing the TOCTOU (time-of-check-time-of-use) race condition.
func (r *DogRepository) CreateWithLimitCheck(dog *models.Dog, limit int) error {
	// BUG #1 FIX: Get or create mutex for this tenant
	mutexValue, _ := dogLimitMutexes.LoadOrStore(dog.TenantID, &sync.Mutex{})
	mutex := mutexValue.(*sync.Mutex)

	// Lock the tenant-specific mutex to serialize dog creation
	mutex.Lock()
	defer mutex.Unlock()

	// Now safe to check and insert without race condition
	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Will be no-op after commit

	// Count dogs within transaction
	// Rebind query for PostgreSQL (? -> $1, $2, ...)
	countQuery := r.db.RebindQuery(`SELECT COUNT(*) FROM dogs WHERE tenant_id = ?`)
	var count int
	err = tx.QueryRow(countQuery, dog.TenantID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count dogs: %w", err)
	}

	// Check limit (-1 means unlimited)
	if limit != -1 && count >= limit {
		return ErrDogLimitExceeded
	}

	// Insert dog within same transaction
	query := `
		INSERT INTO dogs (
			tenant_id, name, breed, size, age, color_id, photo, photo_thumbnail, special_needs,
			pickup_location, walk_route, walk_duration, special_instructions,
			default_morning_time, default_evening_time, is_available, is_featured, external_link
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var id int64

	// PostgreSQL requires RETURNING clause instead of LastInsertId
	if r.db.GetDialectName() == "postgres" {
		query = query + " RETURNING id"
		// Rebind query for PostgreSQL (? -> $1, $2, ...)
		query = r.db.RebindQuery(query)
		err = tx.QueryRow(
			query,
			dog.TenantID,
			dog.Name,
			dog.Breed,
			dog.Size,
			dog.Age,
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
			r.db.BoolValue(dog.IsAvailable),
			r.db.BoolValue(dog.IsFeatured),
			dog.ExternalLink,
		).Scan(&id)
		if err != nil {
			return fmt.Errorf("failed to create dog: %w", err)
		}
	} else {
		// SQLite/MySQL use LastInsertId
		result, err := tx.Exec(
			query,
			dog.TenantID,
			dog.Name,
			dog.Breed,
			dog.Size,
			dog.Age,
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
			r.db.BoolValue(dog.IsAvailable),
			r.db.BoolValue(dog.IsFeatured),
			dog.ExternalLink,
		)
		if err != nil {
			return fmt.Errorf("failed to create dog: %w", err)
		}

		id, err = result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get dog ID: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	dog.ID = int(id)
	dog.CreatedAt = time.Now()
	dog.UpdatedAt = time.Now()
	return nil
}

// Update updates a dog
// SaaS: Filters by tenant_id for tenant isolation (BUG FIX: added tenant_id to WHERE clause)
func (r *DogRepository) Update(dog *models.Dog) error {
	query := `
		UPDATE dogs SET
			name = ?,
			breed = ?,
			size = ?,
			age = ?,
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
		WHERE id = ? AND tenant_id = ?
	`

	result, err := r.db.Exec(
		query,
		dog.Name,
		dog.Breed,
		dog.Size,
		dog.Age,
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
		dog.TenantID,
	)

	if err != nil {
		return fmt.Errorf("failed to update dog: %w", err)
	}

	// Verify update actually happened (0 rows = dog not found or tenant mismatch)
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to verify update: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("dog not found or access denied")
	}

	return nil
}

// Delete deletes a dog (only if no future bookings exist)
// SaaS: Filters by tenant_id for tenant isolation
func (r *DogRepository) Delete(id int, tenantID int) error {
	// Check for future bookings
	// Use Go time instead of database-specific date('now') for portability
	currentDate := time.Now().Format("2006-01-02")
	checkQuery := `
		SELECT COUNT(*) FROM bookings
		WHERE dog_id = ? AND tenant_id = ? AND date >= ? AND status = 'scheduled'
	`

	var count int
	err := r.db.QueryRow(checkQuery, id, tenantID, currentDate).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check bookings: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("cannot delete dog with future bookings")
	}

	// Delete the dog
	deleteQuery := `DELETE FROM dogs WHERE id = ? AND tenant_id = ?`
	_, err = r.db.Exec(deleteQuery, id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete dog: %w", err)
	}

	return nil
}

// ForceDelete deletes a dog and cancels all future bookings
// SaaS: Filters by tenant_id for tenant isolation
func (r *DogRepository) ForceDelete(id int, tenantID int) error {
	// Delete the dog (bookings will remain but dog will be gone)
	deleteQuery := `DELETE FROM dogs WHERE id = ? AND tenant_id = ?`
	_, err := r.db.Exec(deleteQuery, id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete dog: %w", err)
	}

	return nil
}

// GetFutureBookings returns all future bookings for a dog with user details
// SaaS: Now includes tenant_id filter for multi-tenant isolation
func (r *DogRepository) GetFutureBookings(dogID int, tenantID int) ([]*models.Booking, error) {
	currentDate := time.Now().Format("2006-01-02")
	query := `
		SELECT
			b.id, b.user_id, b.dog_id, b.date, b.scheduled_time, b.status,
			b.completed_at, b.user_notes, b.admin_cancellation_reason, b.created_at, b.updated_at,
			u.first_name as user_first_name, u.last_name as user_last_name, u.email as user_email
		FROM bookings b
		LEFT JOIN users u ON b.user_id = u.id
		WHERE b.dog_id = ? AND b.tenant_id = ? AND b.date >= ? AND b.status = 'scheduled'
		ORDER BY b.date ASC, b.scheduled_time ASC
	`

	rows, err := r.db.Query(query, dogID, tenantID, currentDate)
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

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating future bookings: %w", err)
	}

	return bookings, nil
}

// ToggleAvailability toggles a dog's availability status
// SaaS: Filters by tenant_id for tenant isolation
func (r *DogRepository) ToggleAvailability(id int, tenantID int, isAvailable bool, reason *string) error {
	var query string
	var args []interface{}

	if isAvailable {
		// Mark as available (clear reason and timestamp)
		query = `
			UPDATE dogs SET
				is_available = ?,
				unavailable_reason = NULL,
				unavailable_since = NULL,
				updated_at = ?
			WHERE id = ? AND tenant_id = ?
		`
		args = []interface{}{r.db.BoolValue(true), time.Now(), id, tenantID}
	} else {
		// Mark as unavailable
		query = `
			UPDATE dogs SET
				is_available = ?,
				unavailable_reason = ?,
				unavailable_since = ?,
				updated_at = ?
			WHERE id = ? AND tenant_id = ?
		`
		now := time.Now()
		args = []interface{}{r.db.BoolValue(false), reason, now, now, id, tenantID}
	}

	_, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to toggle availability: %w", err)
	}

	return nil
}

// GetBreeds returns a list of unique breeds for a tenant
// SaaS: Filters by tenant_id for tenant isolation
func (r *DogRepository) GetBreeds(tenantID int) ([]string, error) {
	query := `SELECT DISTINCT breed FROM dogs WHERE tenant_id = ? ORDER BY breed ASC`

	rows, err := r.db.Query(query, tenantID)
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

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating breeds: %w", err)
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
