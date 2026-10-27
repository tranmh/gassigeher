//go:build integration

package repository

import (
	"os"
	"testing"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
)

func setupConsentIntegrationDB(t *testing.T) *ConsentRepository {
	t.Helper()
	if os.Getenv("AQAA_SQLITE_PATH") == "" {
		t.Skip("AQAA_SQLITE_PATH not set")
	}
	db := setupTestDB(t)
	return NewConsentRepository(db)
}

func TestIntegrationConsentRepository_RecordAndStatus(t *testing.T) {
	repo := setupConsentIntegrationDB(t)

	// CON-1: no consents -> requires update
	status, err := repo.GetConsentStatus(1, 0)
	if err != nil {
		t.Fatalf("GetConsentStatus: %v", err)
	}
	if status.TermsAccepted || status.PrivacyAccepted || !status.RequiresUpdate {
		t.Errorf("empty status = %+v, want nothing accepted and RequiresUpdate", status)
	}
	valid, err := repo.HasValidConsent(1, 0)
	if err != nil || valid {
		t.Errorf("HasValidConsent = %v, %v; want false, nil", valid, err)
	}

	// CON-2: RecordConsent creates terms + privacy at current versions
	if err := repo.RecordConsent(1, 0, "10.0.0.1", "agent/1.0"); err != nil {
		t.Fatalf("RecordConsent: %v", err)
	}
	consents, err := repo.FindByUserID(1, 0)
	if err != nil {
		t.Fatalf("FindByUserID: %v", err)
	}
	if len(consents) != 2 {
		t.Fatalf("got %d consents, want 2", len(consents))
	}
	types := map[string]bool{}
	for _, c := range consents {
		types[c.ConsentType] = true
		if c.Version != models.CurrentConsentVersions[c.ConsentType] {
			t.Errorf("%s version = %q, want %q", c.ConsentType, c.Version, models.CurrentConsentVersions[c.ConsentType])
		}
		if c.IPAddress != "10.0.0.1" || c.UserAgent != "agent/1.0" {
			t.Errorf("ip/ua = %q/%q", c.IPAddress, c.UserAgent)
		}
		if c.ID == 0 {
			t.Errorf("consent ID not set")
		}
	}
	if !types[models.ConsentTypeTerms] || !types[models.ConsentTypePrivacy] {
		t.Errorf("consent types = %v, want terms and privacy", types)
	}

	status, err = repo.GetConsentStatus(1, 0)
	if err != nil {
		t.Fatalf("GetConsentStatus: %v", err)
	}
	if !status.TermsAccepted || !status.PrivacyAccepted || status.RequiresUpdate {
		t.Errorf("status after record = %+v", status)
	}
	if status.TermsVersion != models.CurrentConsentVersions[models.ConsentTypeTerms] ||
		status.PrivacyVersion != models.CurrentConsentVersions[models.ConsentTypePrivacy] {
		t.Errorf("versions = %q/%q", status.TermsVersion, status.PrivacyVersion)
	}
	valid, err = repo.HasValidConsent(1, 0)
	if err != nil || !valid {
		t.Errorf("HasValidConsent = %v, %v; want true, nil", valid, err)
	}

	// Other users are unaffected
	if valid, _ := repo.HasValidConsent(2, 0); valid {
		t.Errorf("user 2 should not have consent")
	}
}

func TestIntegrationConsentRepository_OutdatedVersion(t *testing.T) {
	repo := setupConsentIntegrationDB(t)

	now := time.Now()
	tests := []struct {
		name        string
		consentType string
		other       string
	}{
		{"outdated terms", models.ConsentTypeTerms, models.ConsentTypePrivacy},
		{"outdated privacy", models.ConsentTypePrivacy, models.ConsentTypeTerms},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := i + 2
			if err := repo.Create(&models.Consent{UserID: userID, TenantID: 0, ConsentType: tt.consentType,
				Version: "0.0-old", IPAddress: "1.1.1.1", UserAgent: "ua", AcceptedAt: now}); err != nil {
				t.Fatalf("Create: %v", err)
			}
			if err := repo.Create(&models.Consent{UserID: userID, TenantID: 0, ConsentType: tt.other,
				Version: models.CurrentConsentVersions[tt.other], IPAddress: "1.1.1.1", UserAgent: "ua", AcceptedAt: now}); err != nil {
				t.Fatalf("Create: %v", err)
			}
			status, err := repo.GetConsentStatus(userID, 0)
			if err != nil {
				t.Fatalf("GetConsentStatus: %v", err)
			}
			if !status.TermsAccepted || !status.PrivacyAccepted || !status.RequiresUpdate {
				t.Errorf("status = %+v, want both accepted and RequiresUpdate", status)
			}
			if valid, _ := repo.HasValidConsent(userID, 0); valid {
				t.Errorf("HasValidConsent = true, want false for outdated version")
			}

			// Accepting the current versions later makes the latest consent valid
			if err := repo.Create(&models.Consent{UserID: userID, TenantID: 0, ConsentType: tt.consentType,
				Version: models.CurrentConsentVersions[tt.consentType], IPAddress: "1.1.1.1", UserAgent: "ua",
				AcceptedAt: now.Add(time.Hour)}); err != nil {
				t.Fatalf("Create: %v", err)
			}
			latest, err := repo.GetLatestByType(userID, 0, tt.consentType)
			if err != nil || latest == nil {
				t.Fatalf("GetLatestByType = %v, %v", latest, err)
			}
			if latest.Version != models.CurrentConsentVersions[tt.consentType] {
				t.Errorf("latest version = %q", latest.Version)
			}
			if valid, _ := repo.HasValidConsent(userID, 0); !valid {
				t.Errorf("HasValidConsent = false after accepting current version")
			}
		})
	}
}

func TestIntegrationConsentRepository_GetLatestByTypeMissing(t *testing.T) {
	repo := setupConsentIntegrationDB(t)
	c, err := repo.GetLatestByType(1, 0, models.ConsentTypeTerms)
	if err != nil || c != nil {
		t.Errorf("GetLatestByType = %v, %v; want nil, nil", c, err)
	}
}
