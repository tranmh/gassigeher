package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
)

// UserRepository handles user database operations
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(user *models.User) error {
	// Include name column for backward compatibility (NOT NULL constraint)
	query := `
		INSERT INTO users (
			name, first_name, last_name, email, phone, password_hash, experience_level,
			is_admin, is_super_admin, is_verified, is_active,
			verification_token, verification_token_expires,
			terms_accepted_at, last_activity_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(
		query,
		user.FullName(),      // Legacy name column for backward compatibility
		user.FirstName,
		user.LastName,
		user.Email,
		user.Phone,
		user.PasswordHash,
		user.ExperienceLevel,
		user.IsAdmin,
		user.IsSuperAdmin,
		user.IsVerified,
		user.IsActive,
		user.VerificationToken,
		user.VerificationTokenExpires,
		user.TermsAcceptedAt,
		user.LastActivityAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get user ID: %w", err)
	}

	user.ID = int(id)
	return nil
}

// FindByEmail finds a user by email
func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
	query := `
		SELECT id, first_name, last_name, email, phone, password_hash, experience_level,
		       is_admin, is_super_admin, is_verified, is_active, is_deleted,
		       verification_token, verification_token_expires, password_reset_token,
		       password_reset_expires, profile_photo, anonymous_id,
		       terms_accepted_at, last_activity_at, deactivated_at,
		       deactivation_reason, reactivated_at, deleted_at,
		       created_at, updated_at
		FROM users
		WHERE email = ? AND is_deleted = 0
	`

	user := &models.User{}
	var firstName, lastName sql.NullString
	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&firstName,
		&lastName,
		&user.Email,
		&user.Phone,
		&user.PasswordHash,
		&user.ExperienceLevel,
		&user.IsAdmin,         // DONE: Added
		&user.IsSuperAdmin,    // DONE: Added
		&user.IsVerified,
		&user.IsActive,
		&user.IsDeleted,
		&user.VerificationToken,
		&user.VerificationTokenExpires,
		&user.PasswordResetToken,
		&user.PasswordResetExpires,
		&user.ProfilePhoto,
		&user.AnonymousID,
		&user.TermsAcceptedAt,
		&user.LastActivityAt,
		&user.DeactivatedAt,
		&user.DeactivationReason,
		&user.ReactivatedAt,
		&user.DeletedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if firstName.Valid {
		user.FirstName = firstName.String
	}
	if lastName.Valid {
		user.LastName = lastName.String
	}

	return user, nil
}

// FindByID finds a user by ID
func (r *UserRepository) FindByID(id int) (*models.User, error) {
	query := `
		SELECT id, first_name, last_name, email, phone, password_hash, experience_level,
		       is_admin, is_super_admin, is_verified, is_active, is_deleted,
		       verification_token, verification_token_expires, password_reset_token,
		       password_reset_expires, profile_photo, anonymous_id,
		       terms_accepted_at, last_activity_at, deactivated_at,
		       deactivation_reason, reactivated_at, deleted_at,
		       created_at, updated_at
		FROM users
		WHERE id = ?
	`

	user := &models.User{}
	var firstName, lastName sql.NullString
	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&firstName,
		&lastName,
		&user.Email,
		&user.Phone,
		&user.PasswordHash,
		&user.ExperienceLevel,
		&user.IsAdmin,         // DONE: Added
		&user.IsSuperAdmin,    // DONE: Added
		&user.IsVerified,
		&user.IsActive,
		&user.IsDeleted,
		&user.VerificationToken,
		&user.VerificationTokenExpires,
		&user.PasswordResetToken,
		&user.PasswordResetExpires,
		&user.ProfilePhoto,
		&user.AnonymousID,
		&user.TermsAcceptedAt,
		&user.LastActivityAt,
		&user.DeactivatedAt,
		&user.DeactivationReason,
		&user.ReactivatedAt,
		&user.DeletedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if firstName.Valid {
		user.FirstName = firstName.String
	}
	if lastName.Valid {
		user.LastName = lastName.String
	}

	return user, nil
}

// FindByVerificationToken finds a user by verification token
func (r *UserRepository) FindByVerificationToken(token string) (*models.User, error) {
	query := `
		SELECT id, first_name, last_name, email, phone, password_hash, experience_level,
		       is_admin, is_super_admin, is_verified, is_active, is_deleted,
		       verification_token, verification_token_expires, password_reset_token,
		       password_reset_expires, profile_photo, anonymous_id,
		       terms_accepted_at, last_activity_at, deactivated_at,
		       deactivation_reason, reactivated_at, deleted_at,
		       created_at, updated_at
		FROM users
		WHERE verification_token = ? AND is_deleted = 0
	`

	user := &models.User{}
	var firstName, lastName sql.NullString
	err := r.db.QueryRow(query, token).Scan(
		&user.ID,
		&firstName,
		&lastName,
		&user.Email,
		&user.Phone,
		&user.PasswordHash,
		&user.ExperienceLevel,
		&user.IsAdmin,         // DONE: Added
		&user.IsSuperAdmin,    // DONE: Added
		&user.IsVerified,
		&user.IsActive,
		&user.IsDeleted,
		&user.VerificationToken,
		&user.VerificationTokenExpires,
		&user.PasswordResetToken,
		&user.PasswordResetExpires,
		&user.ProfilePhoto,
		&user.AnonymousID,
		&user.TermsAcceptedAt,
		&user.LastActivityAt,
		&user.DeactivatedAt,
		&user.DeactivationReason,
		&user.ReactivatedAt,
		&user.DeletedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if firstName.Valid {
		user.FirstName = firstName.String
	}
	if lastName.Valid {
		user.LastName = lastName.String
	}

	return user, nil
}

// FindByPasswordResetToken finds a user by password reset token
func (r *UserRepository) FindByPasswordResetToken(token string) (*models.User, error) {
	query := `
		SELECT id, first_name, last_name, email, phone, password_hash, experience_level,
		       is_admin, is_super_admin, is_verified, is_active, is_deleted,
		       verification_token, verification_token_expires, password_reset_token,
		       password_reset_expires, profile_photo, anonymous_id,
		       terms_accepted_at, last_activity_at, deactivated_at,
		       deactivation_reason, reactivated_at, deleted_at,
		       created_at, updated_at
		FROM users
		WHERE password_reset_token = ? AND is_deleted = 0
	`

	user := &models.User{}
	var firstName, lastName sql.NullString
	err := r.db.QueryRow(query, token).Scan(
		&user.ID,
		&firstName,
		&lastName,
		&user.Email,
		&user.Phone,
		&user.PasswordHash,
		&user.ExperienceLevel,
		&user.IsAdmin,         // DONE: Added
		&user.IsSuperAdmin,    // DONE: Added
		&user.IsVerified,
		&user.IsActive,
		&user.IsDeleted,
		&user.VerificationToken,
		&user.VerificationTokenExpires,
		&user.PasswordResetToken,
		&user.PasswordResetExpires,
		&user.ProfilePhoto,
		&user.AnonymousID,
		&user.TermsAcceptedAt,
		&user.LastActivityAt,
		&user.DeactivatedAt,
		&user.DeactivationReason,
		&user.ReactivatedAt,
		&user.DeletedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if firstName.Valid {
		user.FirstName = firstName.String
	}
	if lastName.Valid {
		user.LastName = lastName.String
	}

	return user, nil
}

// Update updates a user
func (r *UserRepository) Update(user *models.User) error {
	query := `
		UPDATE users SET
			first_name = ?,
			last_name = ?,
			email = ?,
			phone = ?,
			password_hash = ?,
			experience_level = ?,
			is_admin = ?,
			is_super_admin = ?,
			is_verified = ?,
			is_active = ?,
			is_deleted = ?,
			verification_token = ?,
			verification_token_expires = ?,
			password_reset_token = ?,
			password_reset_expires = ?,
			profile_photo = ?,
			anonymous_id = ?,
			last_activity_at = ?,
			deactivated_at = ?,
			deactivation_reason = ?,
			reactivated_at = ?,
			deleted_at = ?,
			updated_at = ?
		WHERE id = ?
	`

	_, err := r.db.Exec(
		query,
		user.FirstName,
		user.LastName,
		user.Email,
		user.Phone,
		user.PasswordHash,
		user.ExperienceLevel,
		user.IsAdmin,          // DONE: Added
		user.IsSuperAdmin,     // DONE: Added
		user.IsVerified,
		user.IsActive,
		user.IsDeleted,
		user.VerificationToken,
		user.VerificationTokenExpires,
		user.PasswordResetToken,
		user.PasswordResetExpires,
		user.ProfilePhoto,
		user.AnonymousID,
		user.LastActivityAt,
		user.DeactivatedAt,
		user.DeactivationReason,
		user.ReactivatedAt,
		user.DeletedAt,
		time.Now(),
		user.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// UpdateLastActivity updates the last activity timestamp
func (r *UserRepository) UpdateLastActivity(userID int) error {
	query := `UPDATE users SET last_activity_at = ? WHERE id = ?`
	_, err := r.db.Exec(query, time.Now(), userID)
	return err
}

// DeleteAccount performs GDPR-compliant account deletion (anonymization)
func (r *UserRepository) DeleteAccount(userID int) error {
	// Check if user is Super Admin by querying the database
	var isSuperAdmin int
	err := r.db.QueryRow("SELECT is_super_admin FROM users WHERE id = ?", userID).Scan(&isSuperAdmin)
	if err != nil {
		return fmt.Errorf("failed to check user status: %w", err)
	}

	// Prevent Super Admin deletion
	if isSuperAdmin == 1 {
		return fmt.Errorf("cannot delete Super Admin account")
	}

	// Generate anonymous ID
	anonymousID := fmt.Sprintf("anonymous_user_%d", time.Now().Unix())

	query := `
		UPDATE users SET
			first_name = 'Deleted',
			last_name = 'User',
			email = NULL,
			phone = NULL,
			password_hash = NULL,
			profile_photo = NULL,
			is_deleted = 1,
			anonymous_id = ?,
			deleted_at = ?,
			updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	_, err = r.db.Exec(query, anonymousID, now, now, userID)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	return nil
}

// Deactivate deactivates a user account
func (r *UserRepository) Deactivate(userID int, reason string) error {
	query := `
		UPDATE users SET
			is_active = 0,
			deactivated_at = ?,
			deactivation_reason = ?,
			updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	_, err := r.db.Exec(query, now, reason, now, userID)
	if err != nil {
		return fmt.Errorf("failed to deactivate user: %w", err)
	}

	return nil
}

// Activate activates a user account
func (r *UserRepository) Activate(userID int) error {
	query := `
		UPDATE users SET
			is_active = 1,
			reactivated_at = ?,
			deactivated_at = NULL,
			deactivation_reason = NULL,
			updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	_, err := r.db.Exec(query, now, now, userID)
	if err != nil {
		return fmt.Errorf("failed to activate user: %w", err)
	}

	return nil
}

// FindInactiveUsers finds users who haven't been active for the specified number of days
func (r *UserRepository) FindInactiveUsers(days int) ([]*models.User, error) {
	query := `
		SELECT id, first_name, last_name, email, phone, password_hash, experience_level,
		       is_admin, is_super_admin, is_verified, is_active, is_deleted,
		       verification_token, verification_token_expires, password_reset_token,
		       password_reset_expires, profile_photo, anonymous_id,
		       terms_accepted_at, last_activity_at, deactivated_at,
		       deactivation_reason, reactivated_at, deleted_at,
		       created_at, updated_at
		FROM users
		WHERE is_active = 1
		  AND is_deleted = 0
		  AND is_admin = 0
		  AND is_super_admin = 0
		  AND last_activity_at < ?
	`

	cutoffDate := time.Now().AddDate(0, 0, -days)
	rows, err := r.db.Query(query, cutoffDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query inactive users: %w", err)
	}
	defer rows.Close()

	users := []*models.User{}
	for rows.Next() {
		user := &models.User{}
		var firstName, lastName sql.NullString
		err := rows.Scan(
			&user.ID,
			&firstName,
			&lastName,
			&user.Email,
			&user.Phone,
			&user.PasswordHash,
			&user.ExperienceLevel,
			&user.IsAdmin,         // DONE: Added
			&user.IsSuperAdmin,    // DONE: Added
			&user.IsVerified,
			&user.IsActive,
			&user.IsDeleted,
			&user.VerificationToken,
			&user.VerificationTokenExpires,
			&user.PasswordResetToken,
			&user.PasswordResetExpires,
			&user.ProfilePhoto,
			&user.AnonymousID,
			&user.TermsAcceptedAt,
			&user.LastActivityAt,
			&user.DeactivatedAt,
			&user.DeactivationReason,
			&user.ReactivatedAt,
			&user.DeletedAt,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		if firstName.Valid {
			user.FirstName = firstName.String
		}
		if lastName.Valid {
			user.LastName = lastName.String
		}
		users = append(users, user)
	}

	return users, nil
}

// FindAll finds all users with optional filters
func (r *UserRepository) FindAll(activeOnly *bool) ([]*models.User, error) {
	query := `
		SELECT id, first_name, last_name, email, phone, password_hash, experience_level,
		       is_admin, is_super_admin, is_verified, is_active, is_deleted,
		       verification_token, verification_token_expires, password_reset_token,
		       password_reset_expires, profile_photo, anonymous_id,
		       terms_accepted_at, last_activity_at, deactivated_at,
		       deactivation_reason, reactivated_at, deleted_at,
		       created_at, updated_at
		FROM users
		WHERE is_deleted = 0
	`

	args := []interface{}{}

	if activeOnly != nil {
		if *activeOnly {
			query += " AND is_active = 1"
		} else {
			query += " AND is_active = 0"
		}
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	users := []*models.User{}
	for rows.Next() {
		user := &models.User{}
		var firstName, lastName sql.NullString
		err := rows.Scan(
			&user.ID,
			&firstName,
			&lastName,
			&user.Email,
			&user.Phone,
			&user.PasswordHash,
			&user.ExperienceLevel,
			&user.IsAdmin,         // DONE: Added
			&user.IsSuperAdmin,    // DONE: Added
			&user.IsVerified,
			&user.IsActive,
			&user.IsDeleted,
			&user.VerificationToken,
			&user.VerificationTokenExpires,
			&user.PasswordResetToken,
			&user.PasswordResetExpires,
			&user.ProfilePhoto,
			&user.AnonymousID,
			&user.TermsAcceptedAt,
			&user.LastActivityAt,
			&user.DeactivatedAt,
			&user.DeactivationReason,
			&user.ReactivatedAt,
			&user.DeletedAt,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		if firstName.Valid {
			user.FirstName = firstName.String
		}
		if lastName.Valid {
			user.LastName = lastName.String
		}
		users = append(users, user)
	}

	return users, nil
}

// PromoteToAdmin promotes a user to admin role
// Also sets experience_level to 'blue' (highest level) so admins can book any dog
// DONE
func (r *UserRepository) PromoteToAdmin(userID int) error {
	query := `UPDATE users SET is_admin = ?, experience_level = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.Exec(query, true, "blue", time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to promote user to admin: %w", err)
	}
	return nil
}

// DemoteAdmin revokes admin privileges from a user
// DONE
func (r *UserRepository) DemoteAdmin(userID int) error {
	query := `UPDATE users SET is_admin = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.Exec(query, false, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to demote admin: %w", err)
	}
	return nil
}

// IsSuperAdmin checks if a user is a super admin
// DONE
func (r *UserRepository) IsSuperAdmin(userID int) (bool, error) {
	var isSuperAdmin bool
	query := `SELECT is_super_admin FROM users WHERE id = ?`
	err := r.db.QueryRow(query, userID).Scan(&isSuperAdmin)
	if err == sql.ErrNoRows {
		return false, fmt.Errorf("user not found")
	}
	if err != nil {
		return false, fmt.Errorf("failed to check super admin status: %w", err)
	}
	return isSuperAdmin, nil
}

// DONE
