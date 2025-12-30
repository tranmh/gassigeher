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
func (r *WalkReportRepository) Create(tenantID int, report *models.WalkReport) error {
	query := `
		INSERT INTO walk_reports (tenant_id, booking_id, behavior_rating, energy_level, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.Exec(query,
		tenantID,
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
	report.TenantID = tenantID
	report.CreatedAt = now
	report.UpdatedAt = now

	return nil
}

// FindByID finds a walk report by ID within a tenant
func (r *WalkReportRepository) FindByID(tenantID int, id int) (*models.WalkReport, error) {
	query := `
		SELECT id, tenant_id, booking_id, behavior_rating, energy_level, notes, created_at, updated_at
		FROM walk_reports
		WHERE id = ? AND tenant_id = ?
	`

	report := &models.WalkReport{}
	err := r.db.QueryRow(query, id, tenantID).Scan(
		&report.ID,
		&report.TenantID,
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
	photos, err := r.GetPhotos(tenantID, report.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load photos: %w", err)
	}
	report.Photos = photos

	return report, nil
}

// FindByBookingID finds a walk report by booking ID within a tenant
func (r *WalkReportRepository) FindByBookingID(tenantID int, bookingID int) (*models.WalkReport, error) {
	query := `
		SELECT id, tenant_id, booking_id, behavior_rating, energy_level, notes, created_at, updated_at
		FROM walk_reports
		WHERE booking_id = ? AND tenant_id = ?
	`

	report := &models.WalkReport{}
	err := r.db.QueryRow(query, bookingID, tenantID).Scan(
		&report.ID,
		&report.TenantID,
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
	photos, err := r.GetPhotos(tenantID, report.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load photos: %w", err)
	}
	report.Photos = photos

	return report, nil
}

// FindByDogID finds all walk reports for a dog with user details within a tenant
func (r *WalkReportRepository) FindByDogID(tenantID int, dogID int, limit int) ([]*models.WalkReport, error) {
	query := `
		SELECT wr.id, wr.tenant_id, wr.booking_id, wr.behavior_rating, wr.energy_level, wr.notes,
		       wr.created_at, wr.updated_at,
		       b.date, b.scheduled_time,
		       u.id as user_id, u.first_name, u.last_name
		FROM walk_reports wr
		JOIN bookings b ON wr.booking_id = b.id AND wr.tenant_id = b.tenant_id
		JOIN users u ON b.user_id = u.id AND b.tenant_id = u.tenant_id
		WHERE b.dog_id = ? AND wr.tenant_id = ?
		ORDER BY wr.created_at DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, dogID, tenantID, limit)
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
			&report.TenantID,
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

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating walk reports: %w", err)
	}

	// Batch load photos for all reports in a single query (fixes N+1 query bug)
	if len(reports) > 0 {
		reportIDs := make([]int, len(reports))
		for i, report := range reports {
			reportIDs[i] = report.ID
		}

		photosMap, err := r.GetPhotosByReportIDs(tenantID, reportIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to batch load photos: %w", err)
		}

		for _, report := range reports {
			if photos, ok := photosMap[report.ID]; ok {
				report.Photos = photos
			}
		}
	}

	return reports, nil
}

// FindByUserID finds all walk reports created by a user within a tenant
func (r *WalkReportRepository) FindByUserID(tenantID int, userID int, limit int) ([]*models.WalkReport, error) {
	query := `
		SELECT wr.id, wr.tenant_id, wr.booking_id, wr.behavior_rating, wr.energy_level, wr.notes,
		       wr.created_at, wr.updated_at,
		       b.date, b.scheduled_time, b.dog_id,
		       d.name as dog_name, d.breed
		FROM walk_reports wr
		JOIN bookings b ON wr.booking_id = b.id AND wr.tenant_id = b.tenant_id
		JOIN dogs d ON b.dog_id = d.id AND b.tenant_id = d.tenant_id
		WHERE b.user_id = ? AND wr.tenant_id = ?
		ORDER BY wr.created_at DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, userID, tenantID, limit)
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
			&report.TenantID,
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

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating walk reports: %w", err)
	}

	// Batch load photos for all reports in a single query (fixes N+1 query bug)
	if len(reports) > 0 {
		reportIDs := make([]int, len(reports))
		for i, report := range reports {
			reportIDs[i] = report.ID
		}

		photosMap, err := r.GetPhotosByReportIDs(tenantID, reportIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to batch load photos: %w", err)
		}

		for _, report := range reports {
			if photos, ok := photosMap[report.ID]; ok {
				report.Photos = photos
			}
		}
	}

	return reports, nil
}

// Update updates a walk report within a tenant
func (r *WalkReportRepository) Update(tenantID int, report *models.WalkReport) error {
	query := `
		UPDATE walk_reports
		SET behavior_rating = ?, energy_level = ?, notes = ?, updated_at = ?
		WHERE id = ? AND tenant_id = ?
	`

	now := time.Now()
	result, err := r.db.Exec(query,
		report.BehaviorRating,
		report.EnergyLevel,
		report.Notes,
		now,
		report.ID,
		tenantID,
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

// Delete deletes a walk report (photos are cascade deleted by FK) within a tenant
func (r *WalkReportRepository) Delete(tenantID int, id int) error {
	query := `DELETE FROM walk_reports WHERE id = ? AND tenant_id = ?`

	result, err := r.db.Exec(query, id, tenantID)
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

// AddPhoto adds a photo to a walk report within a tenant
func (r *WalkReportRepository) AddPhoto(tenantID int, reportID int, photoPath, thumbnailPath string, displayOrder int) (*models.WalkReportPhoto, error) {
	query := `
		INSERT INTO walk_report_photos (tenant_id, walk_report_id, photo_path, photo_thumbnail, display_order, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.Exec(query, tenantID, reportID, photoPath, thumbnailPath, displayOrder, now)
	if err != nil {
		return nil, fmt.Errorf("failed to add photo: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get photo ID: %w", err)
	}

	photo := &models.WalkReportPhoto{
		ID:             int(id),
		TenantID:       tenantID,
		WalkReportID:   reportID,
		PhotoPath:      photoPath,
		PhotoThumbnail: thumbnailPath,
		DisplayOrder:   displayOrder,
		CreatedAt:      now,
	}

	return photo, nil
}

// GetPhotos gets all photos for a walk report within a tenant
func (r *WalkReportRepository) GetPhotos(tenantID int, reportID int) ([]models.WalkReportPhoto, error) {
	query := `
		SELECT id, tenant_id, walk_report_id, photo_path, photo_thumbnail, display_order, created_at
		FROM walk_report_photos
		WHERE walk_report_id = ? AND tenant_id = ?
		ORDER BY display_order ASC
	`

	rows, err := r.db.Query(query, reportID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query photos: %w", err)
	}
	defer rows.Close()

	photos := []models.WalkReportPhoto{}
	for rows.Next() {
		photo := models.WalkReportPhoto{}
		err := rows.Scan(
			&photo.ID,
			&photo.TenantID,
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

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating photos: %w", err)
	}

	return photos, nil
}

// DeletePhoto deletes a photo from a walk report within a tenant
func (r *WalkReportRepository) DeletePhoto(tenantID int, photoID int) error {
	query := `DELETE FROM walk_report_photos WHERE id = ? AND tenant_id = ?`

	result, err := r.db.Exec(query, photoID, tenantID)
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

// GetPhotoByID gets a photo by its ID within a tenant
func (r *WalkReportRepository) GetPhotoByID(tenantID int, photoID int) (*models.WalkReportPhoto, error) {
	query := `
		SELECT id, tenant_id, walk_report_id, photo_path, photo_thumbnail, display_order, created_at
		FROM walk_report_photos
		WHERE id = ? AND tenant_id = ?
	`

	photo := &models.WalkReportPhoto{}
	err := r.db.QueryRow(query, photoID, tenantID).Scan(
		&photo.ID,
		&photo.TenantID,
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

// CountPhotos counts the number of photos for a walk report within a tenant
func (r *WalkReportRepository) CountPhotos(tenantID int, reportID int) (int, error) {
	query := `SELECT COUNT(*) FROM walk_report_photos WHERE walk_report_id = ? AND tenant_id = ?`

	var count int
	err := r.db.QueryRow(query, reportID, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count photos: %w", err)
	}

	return count, nil
}

// GetReportStats gets aggregated statistics for a dog's walk reports within a tenant
func (r *WalkReportRepository) GetReportStats(tenantID int, dogID int) (*models.WalkReportStats, error) {
	query := `
		SELECT
			COUNT(*) as total_walks,
			COALESCE(AVG(wr.behavior_rating), 0) as average_rating,
			COUNT(DISTINCT CASE WHEN wrp.id IS NOT NULL THEN wr.id END) as reports_with_photos
		FROM walk_reports wr
		JOIN bookings b ON wr.booking_id = b.id AND wr.tenant_id = b.tenant_id
		LEFT JOIN walk_report_photos wrp ON wr.id = wrp.walk_report_id AND wr.tenant_id = wrp.tenant_id
		WHERE b.dog_id = ? AND wr.tenant_id = ?
	`

	stats := &models.WalkReportStats{}
	err := r.db.QueryRow(query, dogID, tenantID).Scan(
		&stats.TotalWalks,
		&stats.AverageRating,
		&stats.ReportsWithPhotos,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get report stats: %w", err)
	}

	return stats, nil
}

// GetBookingUserID gets the user ID for a booking within a tenant (for authorization checks)
func (r *WalkReportRepository) GetBookingUserID(tenantID int, bookingID int) (int, error) {
	query := `SELECT user_id FROM bookings WHERE id = ? AND tenant_id = ?`

	var userID int
	err := r.db.QueryRow(query, bookingID, tenantID).Scan(&userID)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("booking not found")
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get booking user: %w", err)
	}

	return userID, nil
}

// IsBookingCompleted checks if a booking is completed within a tenant
func (r *WalkReportRepository) IsBookingCompleted(tenantID int, bookingID int) (bool, error) {
	query := `SELECT status FROM bookings WHERE id = ? AND tenant_id = ?`

	var status string
	err := r.db.QueryRow(query, bookingID, tenantID).Scan(&status)
	if err == sql.ErrNoRows {
		return false, fmt.Errorf("booking not found")
	}
	if err != nil {
		return false, fmt.Errorf("failed to check booking status: %w", err)
	}

	return status == "completed", nil
}

// GetPhotosByReportIDs batch loads photos for multiple walk reports in a single query
// This fixes the N+1 query problem where GetPhotos was called individually for each report
// Returns a map of reportID -> photos
func (r *WalkReportRepository) GetPhotosByReportIDs(tenantID int, reportIDs []int) (map[int][]models.WalkReportPhoto, error) {
	if len(reportIDs) == 0 {
		return make(map[int][]models.WalkReportPhoto), nil
	}

	// Build placeholder string for IN clause
	placeholders := make([]string, len(reportIDs))
	args := make([]interface{}, len(reportIDs)+1)
	args[0] = tenantID
	for i, id := range reportIDs {
		placeholders[i] = "?"
		args[i+1] = id
	}

	query := `
		SELECT id, tenant_id, walk_report_id, photo_path, photo_thumbnail, display_order, created_at
		FROM walk_report_photos
		WHERE tenant_id = ? AND walk_report_id IN (` + joinStrings(placeholders, ",") + `)
		ORDER BY walk_report_id, display_order ASC
	`

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query photos: %w", err)
	}
	defer rows.Close()

	photosMap := make(map[int][]models.WalkReportPhoto)
	for rows.Next() {
		photo := models.WalkReportPhoto{}
		err := rows.Scan(
			&photo.ID,
			&photo.TenantID,
			&photo.WalkReportID,
			&photo.PhotoPath,
			&photo.PhotoThumbnail,
			&photo.DisplayOrder,
			&photo.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan photo: %w", err)
		}
		photosMap[photo.WalkReportID] = append(photosMap[photo.WalkReportID], photo)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating photos: %w", err)
	}

	return photosMap, nil
}

// joinStrings joins strings with a separator (avoiding import of strings package)
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
