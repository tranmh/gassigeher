package repository

import (
	"database/sql"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
)

// ConsentRepository handles consent database operations
type ConsentRepository struct {
	db DBExecutor
}

// NewConsentRepository creates a new consent repository
func NewConsentRepository(db DBExecutor) *ConsentRepository {
	return &ConsentRepository{db: db}
}

// Create records a new consent
func (r *ConsentRepository) Create(consent *models.Consent) error {
	id, err := r.db.InsertReturningID(`
		INSERT INTO consents (user_id, tenant_id, consent_type, version, ip_address, user_agent, accepted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		consent.UserID, consent.TenantID, consent.ConsentType, consent.Version,
		consent.IPAddress, consent.UserAgent, consent.AcceptedAt,
	)
	if err != nil {
		return err
	}

	consent.ID = int(id)
	return nil
}

// FindByUserID returns all consents for a user
func (r *ConsentRepository) FindByUserID(userID, tenantID int) ([]*models.Consent, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, tenant_id, consent_type, version, ip_address, user_agent, accepted_at
		FROM consents
		WHERE user_id = ? AND tenant_id = ?
		ORDER BY accepted_at DESC`,
		userID, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var consents []*models.Consent
	for rows.Next() {
		c := &models.Consent{}
		err := rows.Scan(&c.ID, &c.UserID, &c.TenantID, &c.ConsentType, &c.Version,
			&c.IPAddress, &c.UserAgent, &c.AcceptedAt)
		if err != nil {
			return nil, err
		}
		consents = append(consents, c)
	}
	return consents, nil
}

// GetLatestByType returns the most recent consent of a given type for a user
func (r *ConsentRepository) GetLatestByType(userID, tenantID int, consentType string) (*models.Consent, error) {
	c := &models.Consent{}
	err := r.db.QueryRow(`
		SELECT id, user_id, tenant_id, consent_type, version, ip_address, user_agent, accepted_at
		FROM consents
		WHERE user_id = ? AND tenant_id = ? AND consent_type = ?
		ORDER BY accepted_at DESC
		LIMIT 1`,
		userID, tenantID, consentType,
	).Scan(&c.ID, &c.UserID, &c.TenantID, &c.ConsentType, &c.Version,
		&c.IPAddress, &c.UserAgent, &c.AcceptedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

// GetConsentStatus returns the current consent status for a user
func (r *ConsentRepository) GetConsentStatus(userID, tenantID int) (*models.ConsentStatus, error) {
	status := &models.ConsentStatus{}

	// Get latest terms consent
	termsConsent, err := r.GetLatestByType(userID, tenantID, models.ConsentTypeTerms)
	if err != nil {
		return nil, err
	}
	if termsConsent != nil {
		status.TermsAccepted = true
		status.TermsVersion = termsConsent.Version
		status.TermsAcceptedAt = &termsConsent.AcceptedAt

		// Check if current version is newer
		if termsConsent.Version != models.CurrentConsentVersions[models.ConsentTypeTerms] {
			status.RequiresUpdate = true
		}
	}

	// Get latest privacy consent
	privacyConsent, err := r.GetLatestByType(userID, tenantID, models.ConsentTypePrivacy)
	if err != nil {
		return nil, err
	}
	if privacyConsent != nil {
		status.PrivacyAccepted = true
		status.PrivacyVersion = privacyConsent.Version
		status.PrivacyAcceptedAt = &privacyConsent.AcceptedAt

		// Check if current version is newer
		if privacyConsent.Version != models.CurrentConsentVersions[models.ConsentTypePrivacy] {
			status.RequiresUpdate = true
		}
	}

	// If either is missing, requires update
	if !status.TermsAccepted || !status.PrivacyAccepted {
		status.RequiresUpdate = true
	}

	return status, nil
}

// HasValidConsent checks if user has valid consent for all required types
func (r *ConsentRepository) HasValidConsent(userID, tenantID int) (bool, error) {
	status, err := r.GetConsentStatus(userID, tenantID)
	if err != nil {
		return false, err
	}
	return !status.RequiresUpdate, nil
}

// RecordConsent records consents for terms and privacy
func (r *ConsentRepository) RecordConsent(userID, tenantID int, ipAddress, userAgent string) error {
	now := time.Now()

	// Record terms consent
	termsConsent := &models.Consent{
		UserID:      userID,
		TenantID:    tenantID,
		ConsentType: models.ConsentTypeTerms,
		Version:     models.CurrentConsentVersions[models.ConsentTypeTerms],
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		AcceptedAt:  now,
	}
	if err := r.Create(termsConsent); err != nil {
		return err
	}

	// Record privacy consent
	privacyConsent := &models.Consent{
		UserID:      userID,
		TenantID:    tenantID,
		ConsentType: models.ConsentTypePrivacy,
		Version:     models.CurrentConsentVersions[models.ConsentTypePrivacy],
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		AcceptedAt:  now,
	}
	return r.Create(privacyConsent)
}
