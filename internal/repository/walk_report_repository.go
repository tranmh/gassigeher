package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
)

// WalkReportRepository handles walk report database operations
type WalkReportRepository struct {
	db *sql.DB
}

// NewWalkReportRepository creates a new walk report repository
func NewWalkReportRepository(db *sql.DB) *WalkReportRepository {
	return &WalkReportRepository{db: db}
}

// Create creates a new walk report
func (r *WalkReportRepository) Create(report *models.WalkReport) error {
	query := `
		INSERT INTO walk_reports (booking_id, behavior_rating, energy_level, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.Exec(query,
		report.BookingID,
		report.BehaviorRating,
		report.EnergyLevel,
		report.Notes,
		now,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to create walk report: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get walk report ID: %w", err)
	}

	report.ID = int(id)
	report.CreatedAt = now
	report.UpdatedAt = now

	return nil
}

// FindByID finds a walk report by ID
func (r *WalkReportRepository) FindByID(id int) (*models.WalkReport, error) {
	query := `
		SELECT id, booking_id, behavior_rating, energy_level, notes, created_at, updated_at
		FROM walk_reports
		WHERE id = ?
	`

	report := &models.WalkReport{}
	err := r.db.QueryRow(query, id).Scan(
		&report.ID,
		&report.BookingID,
		&report.BehaviorRating,
		&report.EnergyLevel,
		&report.Notes,
		&report.CreatedAt,
		&report.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find walk report: %w", err)
	}

	// Load photos for this report
	photos, err := r.GetPhotos(report.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load photos: %w", err)
	}
	report.Photos = photos

	return report, nil
}

// FindByBookingID finds a walk report by booking ID
func (r *WalkReportRepository) FindByBookingID(bookingID int) (*models.WalkReport, error) {
	query := `
		SELECT id, booking_id, behavior_rating, energy_level, notes, created_at, updated_at
		FROM walk_reports
		WHERE booking_id = ?
	`

	report := &models.WalkReport{}
	err := r.db.QueryRow(query, bookingID).Scan(
		&report.ID,
		&report.BookingID,
		&report.BehaviorRating,
		&report.EnergyLevel,
		&report.Notes,
		&report.CreatedAt,
		&report.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find walk report by booking: %w", err)
	}

	// Load photos for this report
	photos, err := r.GetPhotos(report.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load photos: %w", err)
	}
	report.Photos = photos

	return report, nil
}

// FindByDogID finds all walk reports for a dog with user details
func (r *WalkReportRepository) FindByDogID(dogID int, limit int) ([]*models.WalkReport, error) {
	query := `
		SELECT wr.id, wr.booking_id, wr.behavior_rating, wr.energy_level, wr.notes,
		       wr.created_at, wr.updated_at,
		       b.date, b.scheduled_time,
		       u.id as user_id, u.first_name, u.last_name
		FROM walk_reports wr
		JOIN bookings b ON wr.booking_id = b.id
		JOIN users u ON b.user_id = u.id
		WHERE b.dog_id = ?
		ORDER BY wr.created_at DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, dogID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query walk reports: %w", err)
	}
	defer rows.Close()

	reports := []*models.WalkReport{}
	for rows.Next() {
		report := &models.WalkReport{
			Booking: &models.Booking{},
			User:    &models.User{},
		}
		var userFirstName, userLastName sql.NullString

		err := rows.Scan(
			&report.ID,
			&report.BookingID,
			&report.BehaviorRating,
			&report.EnergyLevel,
			&report.Notes,
			&report.CreatedAt,
			&report.UpdatedAt,
			&report.Booking.Date,
			&report.Booking.ScheduledTime,
			&report.User.ID,
			&userFirstName,
			&userLastName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan walk report: %w", err)
		}

		if userFirstName.Valid {
			report.User.FirstName = userFirstName.String
		}
		if userLastName.Valid {
			report.User.LastName = userLastName.String
		}

		reports = append(reports, report)
	}

	// Load photos for each report AFTER closing the rows cursor
	// (avoids deadlock with SQLite's single connection)
	for _, report := range reports {
		photos, err := r.GetPhotos(report.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load photos: %w", err)
		}
		report.Photos = photos
	}

	return reports, nil
}

// FindByUserID finds all walk reports created by a user
func (r *WalkReportRepository) FindByUserID(userID int, limit int) ([]*models.WalkReport, error) {
	query := `
		SELECT wr.id, wr.booking_id, wr.behavior_rating, wr.energy_level, wr.notes,
		       wr.created_at, wr.updated_at,
		       b.date, b.scheduled_time, b.dog_id,
		       d.name as dog_name, d.breed
		FROM walk_reports wr
		JOIN bookings b ON wr.booking_id = b.id
		JOIN dogs d ON b.dog_id = d.id
		WHERE b.user_id = ?
		ORDER BY wr.created_at DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query walk reports: %w", err)
	}
	defer rows.Close()

	reports := []*models.WalkReport{}
	for rows.Next() {
		report := &models.WalkReport{
			Booking: &models.Booking{},
			Dog:     &models.Dog{},
		}

		err := rows.Scan(
			&report.ID,
			&report.BookingID,
			&report.BehaviorRating,
			&report.EnergyLevel,
			&report.Notes,
			&report.CreatedAt,
			&report.UpdatedAt,
			&report.Booking.Date,
			&report.Booking.ScheduledTime,
			&report.Dog.ID,
			&report.Dog.Name,
			&report.Dog.Breed,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan walk report: %w", err)
		}

		reports = append(reports, report)
	}

	// Load photos for each report AFTER closing the rows cursor
	// (avoids deadlock with SQLite's single connection)
	for _, report := range reports {
		photos, err := r.GetPhotos(report.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load photos: %w", err)
		}
		report.Photos = photos
	}

	return reports, nil
}

// Update updates a walk report
func (r *WalkReportRepository) Update(report *models.WalkReport) error {
	query := `
		UPDATE walk_reports
		SET behavior_rating = ?, energy_level = ?, notes = ?, updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	result, err := r.db.Exec(query,
		report.BehaviorRating,
		report.EnergyLevel,
		report.Notes,
		now,
		report.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update walk report: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("walk report not found")
	}

	report.UpdatedAt = now
	return nil
}

// Delete deletes a walk report (photos are cascade deleted by FK)
func (r *WalkReportRepository) Delete(id int) error {
	query := `DELETE FROM walk_reports WHERE id = ?`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete walk report: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("walk report not found")
	}

	return nil
}

// AddPhoto adds a photo to a walk report
func (r *WalkReportRepository) AddPhoto(reportID int, photoPath, thumbnailPath string, displayOrder int) (*models.WalkReportPhoto, error) {
	query := `
		INSERT INTO walk_report_photos (walk_report_id, photo_path, photo_thumbnail, display_order, created_at)
		VALUES (?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.Exec(query, reportID, photoPath, thumbnailPath, displayOrder, now)
	if err != nil {
		return nil, fmt.Errorf("failed to add photo: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get photo ID: %w", err)
	}

	photo := &models.WalkReportPhoto{
		ID:             int(id),
		WalkReportID:   reportID,
		PhotoPath:      photoPath,
		PhotoThumbnail: thumbnailPath,
		DisplayOrder:   displayOrder,
		CreatedAt:      now,
	}

	return photo, nil
}

// GetPhotos gets all photos for a walk report
func (r *WalkReportRepository) GetPhotos(reportID int) ([]models.WalkReportPhoto, error) {
	query := `
		SELECT id, walk_report_id, photo_path, photo_thumbnail, display_order, created_at
		FROM walk_report_photos
		WHERE walk_report_id = ?
		ORDER BY display_order ASC
	`

	rows, err := r.db.Query(query, reportID)
	if err != nil {
		return nil, fmt.Errorf("failed to query photos: %w", err)
	}
	defer rows.Close()

	photos := []models.WalkReportPhoto{}
	for rows.Next() {
		photo := models.WalkReportPhoto{}
		err := rows.Scan(
			&photo.ID,
			&photo.WalkReportID,
			&photo.PhotoPath,
			&photo.PhotoThumbnail,
			&photo.DisplayOrder,
			&photo.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan photo: %w", err)
		}
		photos = append(photos, photo)
	}

	return photos, nil
}

// DeletePhoto deletes a photo from a walk report
func (r *WalkReportRepository) DeletePhoto(photoID int) error {
	query := `DELETE FROM walk_report_photos WHERE id = ?`

	result, err := r.db.Exec(query, photoID)
	if err != nil {
		return fmt.Errorf("failed to delete photo: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("photo not found")
	}

	return nil
}

// GetPhotoByID gets a photo by its ID
func (r *WalkReportRepository) GetPhotoByID(photoID int) (*models.WalkReportPhoto, error) {
	query := `
		SELECT id, walk_report_id, photo_path, photo_thumbnail, display_order, created_at
		FROM walk_report_photos
		WHERE id = ?
	`

	photo := &models.WalkReportPhoto{}
	err := r.db.QueryRow(query, photoID).Scan(
		&photo.ID,
		&photo.WalkReportID,
		&photo.PhotoPath,
		&photo.PhotoThumbnail,
		&photo.DisplayOrder,
		&photo.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find photo: %w", err)
	}

	return photo, nil
}

// CountPhotos counts the number of photos for a walk report
func (r *WalkReportRepository) CountPhotos(reportID int) (int, error) {
	query := `SELECT COUNT(*) FROM walk_report_photos WHERE walk_report_id = ?`

	var count int
	err := r.db.QueryRow(query, reportID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count photos: %w", err)
	}

	return count, nil
}

// GetReportStats gets aggregated statistics for a dog's walk reports
func (r *WalkReportRepository) GetReportStats(dogID int) (*models.WalkReportStats, error) {
	query := `
		SELECT
			COUNT(*) as total_walks,
			COALESCE(AVG(wr.behavior_rating), 0) as average_rating,
			COUNT(DISTINCT CASE WHEN wrp.id IS NOT NULL THEN wr.id END) as reports_with_photos
		FROM walk_reports wr
		JOIN bookings b ON wr.booking_id = b.id
		LEFT JOIN walk_report_photos wrp ON wr.id = wrp.walk_report_id
		WHERE b.dog_id = ?
	`

	stats := &models.WalkReportStats{}
	err := r.db.QueryRow(query, dogID).Scan(
		&stats.TotalWalks,
		&stats.AverageRating,
		&stats.ReportsWithPhotos,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get report stats: %w", err)
	}

	return stats, nil
}

// GetBookingUserID gets the user ID for a booking (for authorization checks)
func (r *WalkReportRepository) GetBookingUserID(bookingID int) (int, error) {
	query := `SELECT user_id FROM bookings WHERE id = ?`

	var userID int
	err := r.db.QueryRow(query, bookingID).Scan(&userID)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("booking not found")
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get booking user: %w", err)
	}

	return userID, nil
}

// IsBookingCompleted checks if a booking is completed
func (r *WalkReportRepository) IsBookingCompleted(bookingID int) (bool, error) {
	query := `SELECT status FROM bookings WHERE id = ?`

	var status string
	err := r.db.QueryRow(query, bookingID).Scan(&status)
	if err == sql.ErrNoRows {
		return false, fmt.Errorf("booking not found")
	}
	if err != nil {
		return false, fmt.Errorf("failed to check booking status: %w", err)
	}

	return status == "completed", nil
}
