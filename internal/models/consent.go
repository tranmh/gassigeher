package models

import "time"

// Consent represents a user's consent record for terms/privacy
type Consent struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	TenantID    int       `json:"tenant_id"`
	ConsentType string    `json:"consent_type"` // "terms", "privacy", "marketing"
	Version     string    `json:"version"`      // e.g., "2025-01-01"
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	AcceptedAt  time.Time `json:"accepted_at"`
}

// ConsentType constants
const (
	ConsentTypeTerms     = "terms"
	ConsentTypePrivacy   = "privacy"
	ConsentTypeMarketing = "marketing"
)

// CurrentConsentVersions tracks the current version of each consent type
// When these change, users must re-accept
var CurrentConsentVersions = map[string]string{
	ConsentTypeTerms:   "2025-01-01",
	ConsentTypePrivacy: "2025-01-01",
}

// ConsentStatus represents a user's consent status
type ConsentStatus struct {
	TermsAccepted     bool       `json:"terms_accepted"`
	TermsVersion      string     `json:"terms_version,omitempty"`
	TermsAcceptedAt   *time.Time `json:"terms_accepted_at,omitempty"`
	PrivacyAccepted   bool       `json:"privacy_accepted"`
	PrivacyVersion    string     `json:"privacy_version,omitempty"`
	PrivacyAcceptedAt *time.Time `json:"privacy_accepted_at,omitempty"`
	RequiresUpdate    bool       `json:"requires_update"`
}
