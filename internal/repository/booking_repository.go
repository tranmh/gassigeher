package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/tranm/gassigeher/internal/models"
)

// BookingRepository handles booking database operations
type BookingRepository struct {
	db *sql.DB
}

// NewBookingRepository creates a new booking repository
func NewBookingRepository(db *sql.DB) *BookingRepository {
	return &BookingRepository{db: db}
}

// Create creates a new booking
func (r *BookingRepository) Create(booking *models.Booking) error {
	query := `
		INSERT INTO bookings (user_id, dog_id, date, walk_type, scheduled_time, status, requires_approval, approval_status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()

	// Set default values if not provided
	if booking.Status == "" {
		booking.Status = "scheduled"
	}
	if booking.ApprovalStatus == "" {
		if booking.RequiresApproval {
			booking.ApprovalStatus = "pending"
		} else {
			booking.ApprovalStatus = "approved"
		}
	}

	result, err := r.db.Exec(query,
		booking.UserID,
		booking.DogID,
		booking.Date,
		booking.WalkType,
		booking.ScheduledTime,
		booking.Status,
		booking.RequiresApproval,
		booking.ApprovalStatus,
		now,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to create booking: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get booking ID: %w", err)
	}

	booking.ID = int(id)
	booking.CreatedAt = now
	booking.UpdatedAt = now

	return nil
}

// FindByID finds a booking by ID
func (r *BookingRepository) FindByID(id int) (*models.Booking, error) {
	query := `
		SELECT id, user_id, dog_id, date, walk_type, scheduled_time, status,
		       completed_at, user_notes, admin_cancellation_reason, created_at, updated_at
		FROM bookings
		WHERE id = ?
	`

	booking := &models.Booking{}
	err := r.db.QueryRow(query, id).Scan(
		&booking.ID,
		&booking.UserID,
		&booking.DogID,
		&booking.Date,
		&booking.WalkType,
		&booking.ScheduledTime,
		&booking.Status,
		&booking.CompletedAt,
		&booking.UserNotes,
		&booking.AdminCancellationReason,
		&booking.CreatedAt,
		&booking.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find booking: %w", err)
	}

	return booking, nil
}

// FindAll finds all bookings with optional filters
func (r *BookingRepository) FindAll(filter *models.BookingFilterRequest) ([]*models.Booking, error) {
	query := `
		SELECT id, user_id, dog_id, date, walk_type, scheduled_time, status,
		       completed_at, user_notes, admin_cancellation_reason, created_at, updated_at
		FROM bookings
		WHERE 1=1
	`
	args := []interface{}{}

	if filter != nil {
		if filter.UserID != nil {
			query += " AND user_id = ?"
			args = append(args, *filter.UserID)
		}

		if filter.DogID != nil {
			query += " AND dog_id = ?"
			args = append(args, *filter.DogID)
		}

		if filter.DateFrom != nil {
			query += " AND date >= ?"
			args = append(args, *filter.DateFrom)
		}

		if filter.DateTo != nil {
			query += " AND date <= ?"
			args = append(args, *filter.DateTo)
		}

		if filter.Status != nil {
			query += " AND status = ?"
			args = append(args, *filter.Status)
		}

		if filter.WalkType != nil {
			query += " AND walk_type = ?"
			args = append(args, *filter.WalkType)
		}

		if filter.Year != nil && filter.Month != nil {
			// Filter by year and month
			startDate := fmt.Sprintf("%d-%02d-01", *filter.Year, *filter.Month)
			// Calculate last day of month
			nextMonth := time.Date(*filter.Year, time.Month(*filter.Month+1), 1, 0, 0, 0, 0, time.UTC)
			endDate := nextMonth.Add(-24 * time.Hour).Format("2006-01-02")

			query += " AND date >= ? AND date <= ?"
			args = append(args, startDate, endDate)
		}
	}

	query += " ORDER BY date ASC, scheduled_time ASC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query bookings: %w", err)
	}
	defer rows.Close()

	bookings := []*models.Booking{}
	for rows.Next() {
		booking := &models.Booking{}
		err := rows.Scan(
			&booking.ID,
			&booking.UserID,
			&booking.DogID,
			&booking.Date,
			&booking.WalkType,
			&booking.ScheduledTime,
			&booking.Status,
			&booking.CompletedAt,
			&booking.UserNotes,
			&booking.AdminCancellationReason,
			&booking.CreatedAt,
			&booking.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan booking: %w", err)
		}
		bookings = append(bookings, booking)
	}

	return bookings, nil
}

// Cancel cancels a booking
func (r *BookingRepository) Cancel(id int, reason *string) error {
	query := `
		UPDATE bookings
		SET status = ?, admin_cancellation_reason = ?, updated_at = ?
		WHERE id = ?
	`

	_, err := r.db.Exec(query, "cancelled", reason, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to cancel booking: %w", err)
	}

	return nil
}

// AddNotes adds notes to a completed booking
func (r *BookingRepository) AddNotes(id int, notes string) error {
	query := `
		UPDATE bookings
		SET user_notes = ?, updated_at = ?
		WHERE id = ? AND status = 'completed'
	`

	result, err := r.db.Exec(query, notes, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to add notes: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("booking not found or not completed")
	}

	return nil
}

// CheckDoubleBooking checks if a dog is already booked for the given date and walk type
func (r *BookingRepository) CheckDoubleBooking(dogID int, date, walkType string) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM bookings
		WHERE dog_id = ? AND date = ? AND walk_type = ? AND status = 'scheduled'
	`

	var count int
	err := r.db.QueryRow(query, dogID, date, walkType).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check double booking: %w", err)
	}

	return count > 0, nil
}

// AutoComplete marks all past scheduled bookings as completed
func (r *BookingRepository) AutoComplete() (int, error) {
	// Get current date and time
	now := time.Now()
	currentDate := now.Format("2006-01-02")
	currentTime := now.Format("15:04")

	query := `
		UPDATE bookings
		SET status = 'completed', completed_at = ?, updated_at = ?
		WHERE status = 'scheduled'
		AND (
			date < ?
			OR (date = ? AND scheduled_time < ?)
		)
	`

	result, err := r.db.Exec(query, now, now, currentDate, currentDate, currentTime)
	if err != nil {
		return 0, fmt.Errorf("failed to auto-complete bookings: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rows), nil
}

// GetUpcoming gets upcoming bookings for a user
func (r *BookingRepository) GetUpcoming(userID int, limit int) ([]*models.Booking, error) {
	query := `
		SELECT id, user_id, dog_id, date, walk_type, scheduled_time, status,
		       completed_at, user_notes, admin_cancellation_reason, created_at, updated_at
		FROM bookings
		WHERE user_id = ? AND status = 'scheduled' AND date >= ?
		ORDER BY date ASC, scheduled_time ASC
		LIMIT ?
	`

	currentDate := time.Now().Format("2006-01-02")
	rows, err := r.db.Query(query, userID, currentDate, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query upcoming bookings: %w", err)
	}
	defer rows.Close()

	bookings := []*models.Booking{}
	for rows.Next() {
		booking := &models.Booking{}
		err := rows.Scan(
			&booking.ID,
			&booking.UserID,
			&booking.DogID,
			&booking.Date,
			&booking.WalkType,
			&booking.ScheduledTime,
			&booking.Status,
			&booking.CompletedAt,
			&booking.UserNotes,
			&booking.AdminCancellationReason,
			&booking.CreatedAt,
			&booking.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan booking: %w", err)
		}
		bookings = append(bookings, booking)
	}

	return bookings, nil
}

// GetForReminders gets bookings that need reminders (1 hour before scheduled time)
func (r *BookingRepository) GetForReminders() ([]*models.Booking, error) {
	// Get bookings scheduled within the next 1-2 hours
	now := time.Now()
	oneHourFromNow := now.Add(1 * time.Hour)
	twoHoursFromNow := now.Add(2 * time.Hour)

	// Need to handle bookings that might be on different dates
	// (e.g., if it's 23:30 now, a booking at 00:30 tomorrow is 1 hour away)
	currentDate := now.Format("2006-01-02")
	nextDate := now.Add(24 * time.Hour).Format("2006-01-02")

	oneHourTime := oneHourFromNow.Format("15:04")
	twoHoursTime := twoHoursFromNow.Format("15:04")

	// Query handles both same-day and next-day bookings
	query := `
		SELECT id, user_id, dog_id, date, walk_type, scheduled_time, status,
		       completed_at, user_notes, admin_cancellation_reason, created_at, updated_at
		FROM bookings
		WHERE status = 'scheduled'
		AND (
			(date = ? AND scheduled_time >= ? AND scheduled_time < ?)
			OR (date = ? AND scheduled_time >= '00:00' AND scheduled_time < ?)
		)
	`

	// For same-day bookings: use oneHourTime to twoHoursTime
	// For next-day bookings: check if twoHoursFromNow crosses midnight
	var rows *sql.Rows
	var err error

	if twoHoursFromNow.Format("2006-01-02") != currentDate {
		// Crosses midnight - need to check both today and tomorrow
		nextDayTime := twoHoursFromNow.Format("15:04")
		rows, err = r.db.Query(query, currentDate, oneHourTime, "23:59", nextDate, nextDayTime)
	} else {
		// Same day - simpler query
		simpleQuery := `
			SELECT id, user_id, dog_id, date, walk_type, scheduled_time, status,
			       completed_at, user_notes, admin_cancellation_reason, created_at, updated_at
			FROM bookings
			WHERE status = 'scheduled'
			AND date = ?
			AND scheduled_time >= ?
			AND scheduled_time < ?
		`
		rows, err = r.db.Query(simpleQuery, currentDate, oneHourTime, twoHoursTime)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query bookings for reminders: %w", err)
	}
	defer rows.Close()

	bookings := []*models.Booking{}
	for rows.Next() {
		booking := &models.Booking{}
		err := rows.Scan(
			&booking.ID,
			&booking.UserID,
			&booking.DogID,
			&booking.Date,
			&booking.WalkType,
			&booking.ScheduledTime,
			&booking.Status,
			&booking.CompletedAt,
			&booking.UserNotes,
			&booking.AdminCancellationReason,
			&booking.CreatedAt,
			&booking.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan booking: %w", err)
		}
		bookings = append(bookings, booking)
	}

	return bookings, nil
}

// Update updates a booking (for admin to move bookings)
func (r *BookingRepository) Update(booking *models.Booking) error {
	query := `
		UPDATE bookings
		SET date = ?, walk_type = ?, scheduled_time = ?, updated_at = ?
		WHERE id = ?
	`

	_, err := r.db.Exec(query,
		booking.Date,
		booking.WalkType,
		booking.ScheduledTime,
		time.Now(),
		booking.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update booking: %w", err)
	}

	return nil
}

// FindByIDWithDetails finds a booking by ID with user and dog details
func (r *BookingRepository) FindByIDWithDetails(id int) (*models.Booking, error) {
	query := `
		SELECT
			b.id, b.user_id, b.dog_id, b.date, b.walk_type, b.scheduled_time, b.status,
			b.completed_at, b.user_notes, b.admin_cancellation_reason, b.created_at, b.updated_at,
			u.name as user_name, u.email as user_email, u.phone as user_phone,
			d.name as dog_name, d.breed, d.size, d.age
		FROM bookings b
		LEFT JOIN users u ON b.user_id = u.id
		LEFT JOIN dogs d ON b.dog_id = d.id
		WHERE b.id = ?
	`

	booking := &models.Booking{
		User: &models.User{},
		Dog:  &models.Dog{},
	}

	var userName, userEmail, userPhone sql.NullString
	var dogName, breed, size string
	var age int

	err := r.db.QueryRow(query, id).Scan(
		&booking.ID,
		&booking.UserID,
		&booking.DogID,
		&booking.Date,
		&booking.WalkType,
		&booking.ScheduledTime,
		&booking.Status,
		&booking.CompletedAt,
		&booking.UserNotes,
		&booking.AdminCancellationReason,
		&booking.CreatedAt,
		&booking.UpdatedAt,
		&userName,
		&userEmail,
		&userPhone,
		&dogName,
		&breed,
		&size,
		&age,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find booking with details: %w", err)
	}

	// Populate user details
	if userName.Valid {
		booking.User.Name = userName.String
	} else {
		booking.User.Name = "Deleted User"
	}
	if userEmail.Valid {
		email := userEmail.String
		booking.User.Email = &email
	}
	if userPhone.Valid {
		phone := userPhone.String
		booking.User.Phone = &phone
	}

	// Populate dog details
	booking.Dog.Name = dogName
	booking.Dog.Breed = breed
	booking.Dog.Size = size
	booking.Dog.Age = age

	return booking, nil
}

// GetPendingApprovalBookings returns all bookings awaiting approval
func (r *BookingRepository) GetPendingApprovalBookings() ([]*models.Booking, error) {
	query := `
		SELECT b.id, b.user_id, b.dog_id, b.date, b.scheduled_time, b.walk_type,
		       b.status, b.completed_at, b.user_notes, b.admin_cancellation_reason,
		       b.created_at, b.updated_at,
		       b.requires_approval, b.approval_status, b.approved_by, b.approved_at, b.rejection_reason,
		       u.name as user_name, u.email as user_email, u.phone as user_phone,
		       d.name as dog_name, d.breed, d.size, d.age
		FROM bookings b
		JOIN users u ON b.user_id = u.id
		JOIN dogs d ON b.dog_id = d.id
		WHERE b.approval_status = 'pending'
		ORDER BY b.date ASC, b.scheduled_time ASC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []*models.Booking
	for rows.Next() {
		booking := &models.Booking{
			User: &models.User{},
			Dog:  &models.Dog{},
		}

		var completedAt, approvedAt sql.NullTime
		var userNotes, adminCancellationReason, rejectionReason sql.NullString
		var userEmail, userPhone sql.NullString
		var approvedBy sql.NullInt64
		var requiresApproval int
		var userName, dogName, breed string
		var size sql.NullString
		var age sql.NullInt64

		err := rows.Scan(
			&booking.ID, &booking.UserID, &booking.DogID,
			&booking.Date, &booking.ScheduledTime, &booking.WalkType,
			&booking.Status, &completedAt, &userNotes, &adminCancellationReason,
			&booking.CreatedAt, &booking.UpdatedAt,
			&requiresApproval, &booking.ApprovalStatus, &approvedBy, &approvedAt, &rejectionReason,
			&userName, &userEmail, &userPhone,
			&dogName, &breed, &size, &age,
		)
		if err != nil {
			return nil, err
		}

		// Convert nullable fields
		booking.RequiresApproval = requiresApproval == 1
		if completedAt.Valid {
			t := completedAt.Time
			booking.CompletedAt = &t
		}
		if userNotes.Valid {
			n := userNotes.String
			booking.UserNotes = &n
		}
		if adminCancellationReason.Valid {
			r := adminCancellationReason.String
			booking.AdminCancellationReason = &r
		}
		if approvedBy.Valid {
			id := int(approvedBy.Int64)
			booking.ApprovedBy = &id
		}
		if approvedAt.Valid {
			t := approvedAt.Time
			booking.ApprovedAt = &t
		}
		if rejectionReason.Valid {
			r := rejectionReason.String
			booking.RejectionReason = &r
		}

		// Populate user details
		booking.User.ID = booking.UserID
		booking.User.Name = userName
		if userEmail.Valid {
			email := userEmail.String
			booking.User.Email = &email
		}
		if userPhone.Valid {
			phone := userPhone.String
			booking.User.Phone = &phone
		}

		// Populate dog details
		booking.Dog.ID = booking.DogID
		booking.Dog.Name = dogName
		booking.Dog.Breed = breed
		if size.Valid {
			booking.Dog.Size = size.String
		}
		if age.Valid {
			booking.Dog.Age = int(age.Int64)
		}

		bookings = append(bookings, booking)
	}

	return bookings, nil
}

// ApproveBooking approves a pending booking
func (r *BookingRepository) ApproveBooking(bookingID int, adminID int) error {
	query := `
		UPDATE bookings
		SET approval_status = 'approved', approved_by = ?, approved_at = ?
		WHERE id = ? AND approval_status = 'pending'
	`

	result, err := r.db.Exec(query, adminID, time.Now(), bookingID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("booking not found or not pending")
	}

	return nil
}

// RejectBooking rejects a pending booking
func (r *BookingRepository) RejectBooking(bookingID int, adminID int, reason string) error {
	// Validate that reason is not empty
	if reason == "" {
		return fmt.Errorf("rejection reason is required")
	}

	query := `
		UPDATE bookings
		SET approval_status = 'rejected', approved_by = ?, approved_at = ?, rejection_reason = ?, status = 'cancelled'
		WHERE id = ? AND approval_status = 'pending'
	`

	result, err := r.db.Exec(query, adminID, time.Now(), reason, bookingID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("booking not found or not pending")
	}

	return nil
}
