package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
)

// RecurringBookingRepository handles recurring booking series database operations
type RecurringBookingRepository struct {
	db DBExecutor
}

// NewRecurringBookingRepository creates a new recurring booking repository
func NewRecurringBookingRepository(db DBExecutor) *RecurringBookingRepository {
	return &RecurringBookingRepository{db: db}
}

// Create creates a new recurring booking series
func (r *RecurringBookingRepository) Create(series *models.RecurringBookingSeries) error {
	query := `
		INSERT INTO recurring_booking_series (tenant_id, user_id, dog_id, recurrence_type, day_of_week, interval_days, scheduled_time, start_date, end_date, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	if series.Status == "" {
		series.Status = "active"
	}

	id, err := r.db.InsertReturningID(query,
		series.TenantID,
		series.UserID,
		series.DogID,
		series.RecurrenceType,
		series.DayOfWeek,
		series.IntervalDays,
		series.ScheduledTime,
		series.StartDate,
		series.EndDate,
		series.Status,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to create recurring booking series: %w", err)
	}

	series.ID = int(id)
	series.CreatedAt = now
	series.UpdatedAt = now

	return nil
}

// FindByID finds a recurring booking series by ID
func (r *RecurringBookingRepository) FindByID(id int) (*models.RecurringBookingSeries, error) {
	query := `
		SELECT id, tenant_id, user_id, dog_id, recurrence_type, day_of_week, interval_days,
		       scheduled_time, start_date, end_date, status, created_at, updated_at
		FROM recurring_booking_series
		WHERE id = ?
	`

	series := &models.RecurringBookingSeries{}
	var dayOfWeek, intervalDays sql.NullInt64

	err := r.db.QueryRow(query, id).Scan(
		&series.ID,
		&series.TenantID,
		&series.UserID,
		&series.DogID,
		&series.RecurrenceType,
		&dayOfWeek,
		&intervalDays,
		&series.ScheduledTime,
		&series.StartDate,
		&series.EndDate,
		&series.Status,
		&series.CreatedAt,
		&series.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find recurring booking series: %w", err)
	}

	if dayOfWeek.Valid {
		dow := int(dayOfWeek.Int64)
		series.DayOfWeek = &dow
	}
	if intervalDays.Valid {
		id := int(intervalDays.Int64)
		series.IntervalDays = &id
	}

	series.StartDate = normalizeDate(series.StartDate)
	series.EndDate = normalizeDate(series.EndDate)

	return series, nil
}

// FindByIDAndTenant finds a recurring booking series by ID with tenant isolation
func (r *RecurringBookingRepository) FindByIDAndTenant(id int, tenantID int) (*models.RecurringBookingSeries, error) {
	query := `
		SELECT id, tenant_id, user_id, dog_id, recurrence_type,
		       day_of_week, interval_days, scheduled_time, start_date,
		       end_date, status, created_at, updated_at
		FROM recurring_booking_series
		WHERE id = ? AND tenant_id = ?
	`

	series := &models.RecurringBookingSeries{}
	var dayOfWeek, intervalDays sql.NullInt64
	err := r.db.QueryRow(query, id, tenantID).Scan(
		&series.ID, &series.TenantID, &series.UserID, &series.DogID,
		&series.RecurrenceType, &dayOfWeek, &intervalDays,
		&series.ScheduledTime, &series.StartDate, &series.EndDate,
		&series.Status, &series.CreatedAt, &series.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find recurring series by ID and tenant: %w", err)
	}

	if dayOfWeek.Valid {
		dow := int(dayOfWeek.Int64)
		series.DayOfWeek = &dow
	}
	if intervalDays.Valid {
		intDays := int(intervalDays.Int64)
		series.IntervalDays = &intDays
	}

	series.StartDate = normalizeDate(series.StartDate)
	series.EndDate = normalizeDate(series.EndDate)

	return series, nil
}

// FindByUserID finds all recurring series for a user
func (r *RecurringBookingRepository) FindByUserID(userID, tenantID int) ([]*models.RecurringBookingSeries, error) {
	query := `
		SELECT rbs.id, rbs.tenant_id, rbs.user_id, rbs.dog_id, rbs.recurrence_type,
		       rbs.day_of_week, rbs.interval_days, rbs.scheduled_time, rbs.start_date,
		       rbs.end_date, rbs.status, rbs.created_at, rbs.updated_at,
		       d.name, d.breed, d.photo_thumbnail
		FROM recurring_booking_series rbs
		LEFT JOIN dogs d ON rbs.dog_id = d.id AND d.tenant_id = rbs.tenant_id
		WHERE rbs.user_id = ? AND rbs.tenant_id = ?
		ORDER BY rbs.created_at DESC
	`

	rows, err := r.db.Query(query, userID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to find recurring series by user: %w", err)
	}
	defer rows.Close()

	return r.scanSeriesWithDogRows(rows)
}

// FindAll finds all recurring series for a tenant with optional filters
func (r *RecurringBookingRepository) FindAll(filter *models.RecurringBookingFilterRequest) ([]*models.RecurringBookingSeries, error) {
	query := `
		SELECT rbs.id, rbs.tenant_id, rbs.user_id, rbs.dog_id, rbs.recurrence_type,
		       rbs.day_of_week, rbs.interval_days, rbs.scheduled_time, rbs.start_date,
		       rbs.end_date, rbs.status, rbs.created_at, rbs.updated_at,
		       d.name, d.breed, d.photo_thumbnail
		FROM recurring_booking_series rbs
		LEFT JOIN dogs d ON rbs.dog_id = d.id AND d.tenant_id = rbs.tenant_id
		WHERE 1=1
	`
	args := []interface{}{}

	if filter != nil {
		if filter.TenantID != nil {
			query += " AND rbs.tenant_id = ?"
			args = append(args, *filter.TenantID)
		}
		if filter.UserID != nil {
			query += " AND rbs.user_id = ?"
			args = append(args, *filter.UserID)
		}
		if filter.DogID != nil {
			query += " AND rbs.dog_id = ?"
			args = append(args, *filter.DogID)
		}
		if filter.Status != nil {
			query += " AND rbs.status = ?"
			args = append(args, *filter.Status)
		}
	}

	query += " ORDER BY rbs.created_at DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to find recurring series: %w", err)
	}
	defer rows.Close()

	return r.scanSeriesWithDogRows(rows)
}

// Cancel cancels a recurring booking series
func (r *RecurringBookingRepository) Cancel(id, tenantID int) error {
	query := `
		UPDATE recurring_booking_series
		SET status = 'cancelled', updated_at = ?
		WHERE id = ? AND tenant_id = ?
	`

	result, err := r.db.Exec(query, time.Now(), id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to cancel recurring series: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}

	return nil
}

// MarkCompleted marks a recurring series as completed
func (r *RecurringBookingRepository) MarkCompleted(id, tenantID int) error {
	query := `
		UPDATE recurring_booking_series
		SET status = 'completed', updated_at = ?
		WHERE id = ? AND tenant_id = ?
	`

	result, err := r.db.Exec(query, time.Now(), id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to mark recurring series as completed: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}

	return nil
}

// CountActiveByUser counts active recurring series for a user
func (r *RecurringBookingRepository) CountActiveByUser(userID, tenantID int) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM recurring_booking_series
		WHERE user_id = ? AND tenant_id = ? AND status = 'active'
	`

	var count int
	err := r.db.QueryRow(query, userID, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count active recurring series: %w", err)
	}

	return count, nil
}

// FindActiveByDog finds all active recurring series for a specific dog
func (r *RecurringBookingRepository) FindActiveByDog(dogID, tenantID int) ([]*models.RecurringBookingSeries, error) {
	query := `
		SELECT id, tenant_id, user_id, dog_id, recurrence_type, day_of_week, interval_days,
		       scheduled_time, start_date, end_date, status, created_at, updated_at
		FROM recurring_booking_series
		WHERE dog_id = ? AND tenant_id = ? AND status = 'active'
	`

	rows, err := r.db.Query(query, dogID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to find active recurring series by dog: %w", err)
	}
	defer rows.Close()

	return r.scanSeriesRows(rows)
}

// scanSeriesRows scans rows into recurring booking series (without joins)
func (r *RecurringBookingRepository) scanSeriesRows(rows *sql.Rows) ([]*models.RecurringBookingSeries, error) {
	var seriesList []*models.RecurringBookingSeries

	for rows.Next() {
		series := &models.RecurringBookingSeries{}
		var dayOfWeek, intervalDays sql.NullInt64

		err := rows.Scan(
			&series.ID,
			&series.TenantID,
			&series.UserID,
			&series.DogID,
			&series.RecurrenceType,
			&dayOfWeek,
			&intervalDays,
			&series.ScheduledTime,
			&series.StartDate,
			&series.EndDate,
			&series.Status,
			&series.CreatedAt,
			&series.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan recurring series: %w", err)
		}

		if dayOfWeek.Valid {
			dow := int(dayOfWeek.Int64)
			series.DayOfWeek = &dow
		}
		if intervalDays.Valid {
			id := int(intervalDays.Int64)
			series.IntervalDays = &id
		}

		series.StartDate = normalizeDate(series.StartDate)
		series.EndDate = normalizeDate(series.EndDate)

		seriesList = append(seriesList, series)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recurring series: %w", err)
	}

	return seriesList, nil
}

// scanSeriesWithDogRows scans rows with joined dog data
func (r *RecurringBookingRepository) scanSeriesWithDogRows(rows *sql.Rows) ([]*models.RecurringBookingSeries, error) {
	var seriesList []*models.RecurringBookingSeries

	for rows.Next() {
		series := &models.RecurringBookingSeries{}
		var dayOfWeek, intervalDays sql.NullInt64
		var dogName, dogBreed, dogPhotoThumbnail sql.NullString

		err := rows.Scan(
			&series.ID,
			&series.TenantID,
			&series.UserID,
			&series.DogID,
			&series.RecurrenceType,
			&dayOfWeek,
			&intervalDays,
			&series.ScheduledTime,
			&series.StartDate,
			&series.EndDate,
			&series.Status,
			&series.CreatedAt,
			&series.UpdatedAt,
			&dogName,
			&dogBreed,
			&dogPhotoThumbnail,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan recurring series with dog: %w", err)
		}

		if dayOfWeek.Valid {
			dow := int(dayOfWeek.Int64)
			series.DayOfWeek = &dow
		}
		if intervalDays.Valid {
			id := int(intervalDays.Int64)
			series.IntervalDays = &id
		}

		series.StartDate = normalizeDate(series.StartDate)
		series.EndDate = normalizeDate(series.EndDate)

		// Attach dog data if available
		if dogName.Valid {
			series.Dog = &models.Dog{
				ID:   series.DogID,
				Name: dogName.String,
			}
			if dogBreed.Valid {
				series.Dog.Breed = dogBreed.String
			}
			if dogPhotoThumbnail.Valid {
				thumbnail := dogPhotoThumbnail.String
				series.Dog.PhotoThumbnail = &thumbnail
			}
		}

		seriesList = append(seriesList, series)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recurring series: %w", err)
	}

	return seriesList, nil
}
