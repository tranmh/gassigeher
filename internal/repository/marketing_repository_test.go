package repository

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/tranmh/gassigeher/internal/models"
)

func setupMarketingTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Create marketing tables
	_, err = db.Exec(`
		CREATE TABLE marketing_campaigns (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			config TEXT,
			is_active INTEGER DEFAULT 0,
			start_date TIMESTAMP,
			end_date TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE referral_codes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL UNIQUE,
			referrer_tenant_id INTEGER,
			referrer_email TEXT,
			discount_months_referrer INTEGER DEFAULT 3,
			discount_months_referee INTEGER DEFAULT 1,
			uses_count INTEGER DEFAULT 0,
			max_uses INTEGER,
			is_active INTEGER DEFAULT 1,
			expires_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE tenants (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			slug TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			status TEXT DEFAULT 'active',
			contact_email TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE referral_uses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code_id INTEGER NOT NULL,
			referee_tenant_id INTEGER NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE reference_entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tenant_id INTEGER NOT NULL UNIQUE,
			display_name TEXT NOT NULL,
			city TEXT,
			federal_state TEXT,
			website_url TEXT,
			testimonial TEXT,
			logo_url TEXT,
			is_approved INTEGER DEFAULT 0,
			display_order INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	return db
}

// ========== BUG #1 & #3: Test that CreateCampaign populates timestamps correctly ==========

func TestCreateCampaign_SetsTimestamps(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	repo := NewMarketingRepository(db)

	campaign := &models.MarketingCampaign{
		Type:     "fomo_countdown",
		Name:     "Test Campaign",
		IsActive: true,
	}

	err := repo.CreateCampaign(campaign)
	if err != nil {
		t.Fatalf("CreateCampaign failed: %v", err)
	}

	// BUG #1: After create, the campaign should have proper timestamps set
	// Currently it returns zero timestamps because it doesn't fetch back from DB
	if campaign.CreatedAt.IsZero() {
		t.Errorf("CreateCampaign should set CreatedAt, got zero time")
	}
	if campaign.UpdatedAt.IsZero() {
		t.Errorf("CreateCampaign should set UpdatedAt, got zero time")
	}

	// Verify timestamps are recent (within last minute)
	now := time.Now()
	if now.Sub(campaign.CreatedAt) > time.Minute {
		t.Errorf("CreatedAt should be recent, got %v", campaign.CreatedAt)
	}
}

func TestCreateCampaign_TimestampsInRFC3339Format(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	repo := NewMarketingRepository(db)

	campaign := &models.MarketingCampaign{
		Type:     "fomo_countdown",
		Name:     "Test Campaign",
		IsActive: true,
	}

	err := repo.CreateCampaign(campaign)
	if err != nil {
		t.Fatalf("CreateCampaign failed: %v", err)
	}

	// BUG #3: Check database value is proper RFC3339 format
	var storedCreatedAt string
	err = db.QueryRow("SELECT created_at FROM marketing_campaigns WHERE id = ?", campaign.ID).Scan(&storedCreatedAt)
	if err != nil {
		t.Fatalf("Failed to read created_at: %v", err)
	}

	// Should not have monotonic clock suffix
	if strings.Contains(storedCreatedAt, " m=") {
		t.Errorf("Stored timestamp has monotonic suffix: %s", storedCreatedAt)
	}

	// Should contain 'T' separator (RFC3339 format)
	if !strings.Contains(storedCreatedAt, "T") {
		t.Errorf("Stored timestamp should be RFC3339 format with 'T' separator, got: %s", storedCreatedAt)
	}

	// Should be valid RFC3339 format
	_, err = time.Parse(time.RFC3339, storedCreatedAt)
	if err != nil {
		t.Errorf("Stored timestamp is not RFC3339: %s, error: %v", storedCreatedAt, err)
	}
}

// ========== BUG #2: Test that expires_at is saved correctly ==========

func TestCreateReferralCode_SavesExpiresAt(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	repo := NewMarketingRepository(db)

	// Set expiry to a future date
	expiresAt := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)

	code := &models.ReferralCode{
		Code:                   "TESTCODE2025",
		ReferrerEmail:          stringPtr("test@example.com"),
		DiscountMonthsReferrer: 3,
		DiscountMonthsReferee:  1,
		MaxUses:                intPtr(100),
		IsActive:               true,
		ExpiresAt:              &expiresAt,
	}

	err := repo.CreateReferralCode(code)
	if err != nil {
		t.Fatalf("CreateReferralCode failed: %v", err)
	}

	// Fetch back from DB
	fetched, err := repo.GetReferralCode(code.ID)
	if err != nil {
		t.Fatalf("GetReferralCode failed: %v", err)
	}

	// BUG #2: expires_at should be saved and returned correctly
	if fetched.ExpiresAt.IsZero() {
		t.Errorf("ExpiresAt should not be zero, got: %v", fetched.ExpiresAt)
	}

	// Should match what we set (within 1 second tolerance for DB rounding)
	if fetched.ExpiresAt.Year() != 2025 || fetched.ExpiresAt.Month() != 12 || fetched.ExpiresAt.Day() != 31 {
		t.Errorf("ExpiresAt date mismatch. Expected 2025-12-31, got: %v", fetched.ExpiresAt)
	}
}

func TestCreateReferralCode_SetsTimestamps(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	repo := NewMarketingRepository(db)

	code := &models.ReferralCode{
		Code:                   "TESTCODE",
		DiscountMonthsReferrer: 3,
		DiscountMonthsReferee:  1,
		IsActive:               true,
	}

	err := repo.CreateReferralCode(code)
	if err != nil {
		t.Fatalf("CreateReferralCode failed: %v", err)
	}

	// After create, should have proper timestamps
	if code.CreatedAt.IsZero() {
		t.Errorf("CreateReferralCode should set CreatedAt, got zero time")
	}
	if code.UpdatedAt.IsZero() {
		t.Errorf("CreateReferralCode should set UpdatedAt, got zero time")
	}
}

func TestCreateReferralCode_TimestampsInRFC3339Format(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	repo := NewMarketingRepository(db)

	code := &models.ReferralCode{
		Code:                   "TESTCODE",
		DiscountMonthsReferrer: 3,
		DiscountMonthsReferee:  1,
		IsActive:               true,
	}

	err := repo.CreateReferralCode(code)
	if err != nil {
		t.Fatalf("CreateReferralCode failed: %v", err)
	}

	// Check database value is proper RFC3339 format
	var storedCreatedAt string
	err = db.QueryRow("SELECT created_at FROM referral_codes WHERE id = ?", code.ID).Scan(&storedCreatedAt)
	if err != nil {
		t.Fatalf("Failed to read created_at: %v", err)
	}

	// Should not have monotonic clock suffix
	if strings.Contains(storedCreatedAt, " m=") {
		t.Errorf("Stored timestamp has monotonic suffix: %s", storedCreatedAt)
	}

	// Should contain 'T' separator (RFC3339 format)
	if !strings.Contains(storedCreatedAt, "T") {
		t.Errorf("Stored timestamp should be RFC3339 format with 'T' separator, got: %s", storedCreatedAt)
	}
}

// stringPtr is defined in user_repository_test.go and shared within the package
// intPtr is defined here as it's not in user_repository_test.go
func intPtr(i int) *int {
	return &i
}

// ========== BUG FIX: GetActiveCampaignByType date comparison ==========

func TestGetActiveCampaignByType_FindsActiveCampaign(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	repo := NewMarketingRepository(db)

	// Create an active campaign with dates that include today
	yesterday := time.Now().AddDate(0, 0, -1)
	tomorrow := time.Now().AddDate(0, 0, 1)

	campaign := &models.MarketingCampaign{
		Type:      "fomo_countdown",
		Name:      "Active FOMO Campaign",
		IsActive:  true,
		StartDate: &yesterday,
		EndDate:   &tomorrow,
	}

	err := repo.CreateCampaign(campaign)
	if err != nil {
		t.Fatalf("CreateCampaign failed: %v", err)
	}

	// Get active campaign by type - this was broken due to date comparison bug
	found, err := repo.GetActiveCampaignByType("fomo_countdown")
	if err != nil {
		t.Fatalf("GetActiveCampaignByType failed: %v", err)
	}

	if found == nil {
		t.Fatal("GetActiveCampaignByType should find the active campaign, got nil")
	}

	if found.ID != campaign.ID {
		t.Errorf("GetActiveCampaignByType returned wrong campaign. Expected ID %d, got %d", campaign.ID, found.ID)
	}
}

func TestGetActiveCampaignByType_ReturnsNilForExpiredCampaign(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	repo := NewMarketingRepository(db)

	// Create an expired campaign
	twoDaysAgo := time.Now().AddDate(0, 0, -2)
	yesterday := time.Now().AddDate(0, 0, -1)

	campaign := &models.MarketingCampaign{
		Type:      "fomo_countdown",
		Name:      "Expired Campaign",
		IsActive:  true,
		StartDate: &twoDaysAgo,
		EndDate:   &yesterday,
	}

	err := repo.CreateCampaign(campaign)
	if err != nil {
		t.Fatalf("CreateCampaign failed: %v", err)
	}

	// Should NOT find expired campaign
	found, err := repo.GetActiveCampaignByType("fomo_countdown")
	if err != nil {
		t.Fatalf("GetActiveCampaignByType failed: %v", err)
	}

	if found != nil {
		t.Error("GetActiveCampaignByType should return nil for expired campaign")
	}
}

func TestGetActiveCampaignByType_ReturnsNilForFutureCampaign(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	repo := NewMarketingRepository(db)

	// Create a future campaign
	tomorrow := time.Now().AddDate(0, 0, 1)
	nextWeek := time.Now().AddDate(0, 0, 7)

	campaign := &models.MarketingCampaign{
		Type:      "fomo_countdown",
		Name:      "Future Campaign",
		IsActive:  true,
		StartDate: &tomorrow,
		EndDate:   &nextWeek,
	}

	err := repo.CreateCampaign(campaign)
	if err != nil {
		t.Fatalf("CreateCampaign failed: %v", err)
	}

	// Should NOT find future campaign
	found, err := repo.GetActiveCampaignByType("fomo_countdown")
	if err != nil {
		t.Fatalf("GetActiveCampaignByType failed: %v", err)
	}

	if found != nil {
		t.Error("GetActiveCampaignByType should return nil for campaign that hasn't started yet")
	}
}

func TestGetActiveCampaignByType_ReturnsNilForInactiveCampaign(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	repo := NewMarketingRepository(db)

	// Create an inactive campaign with valid dates
	yesterday := time.Now().AddDate(0, 0, -1)
	tomorrow := time.Now().AddDate(0, 0, 1)

	campaign := &models.MarketingCampaign{
		Type:      "fomo_countdown",
		Name:      "Inactive Campaign",
		IsActive:  false, // Inactive
		StartDate: &yesterday,
		EndDate:   &tomorrow,
	}

	err := repo.CreateCampaign(campaign)
	if err != nil {
		t.Fatalf("CreateCampaign failed: %v", err)
	}

	// Should NOT find inactive campaign
	found, err := repo.GetActiveCampaignByType("fomo_countdown")
	if err != nil {
		t.Fatalf("GetActiveCampaignByType failed: %v", err)
	}

	if found != nil {
		t.Error("GetActiveCampaignByType should return nil for inactive campaign")
	}
}

// ========== BUG FIX: IncrementReferralCodeUses and RecordReferralUse ==========

func TestIncrementReferralCodeUses_IncrementsCount(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	repo := NewMarketingRepository(db)

	// Create a referral code with 0 uses
	code := &models.ReferralCode{
		Code:                   "TESTINCREMENT",
		DiscountMonthsReferrer: 3,
		DiscountMonthsReferee:  1,
		UsesCount:              0,
		IsActive:               true,
	}

	err := repo.CreateReferralCode(code)
	if err != nil {
		t.Fatalf("CreateReferralCode failed: %v", err)
	}

	// Increment uses
	err = repo.IncrementReferralCodeUses(code.ID)
	if err != nil {
		t.Fatalf("IncrementReferralCodeUses failed: %v", err)
	}

	// Fetch and verify count increased
	fetched, err := repo.GetReferralCode(code.ID)
	if err != nil {
		t.Fatalf("GetReferralCode failed: %v", err)
	}

	if fetched.UsesCount != 1 {
		t.Errorf("Expected uses_count to be 1, got %d", fetched.UsesCount)
	}

	// Increment again
	err = repo.IncrementReferralCodeUses(code.ID)
	if err != nil {
		t.Fatalf("Second IncrementReferralCodeUses failed: %v", err)
	}

	fetched, _ = repo.GetReferralCode(code.ID)
	if fetched.UsesCount != 2 {
		t.Errorf("Expected uses_count to be 2 after second increment, got %d", fetched.UsesCount)
	}
}

func TestRecordReferralUse_RecordsUse(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	repo := NewMarketingRepository(db)

	// Create a tenant first
	_, err := db.Exec(`INSERT INTO tenants (slug, name) VALUES ('test-tenant', 'Test Tenant')`)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	// Get the tenant ID
	var tenantID int
	err = db.QueryRow(`SELECT id FROM tenants WHERE slug = 'test-tenant'`).Scan(&tenantID)
	if err != nil {
		t.Fatalf("Failed to get tenant ID: %v", err)
	}

	// Create a referral code
	code := &models.ReferralCode{
		Code:                   "TESTRECORD",
		DiscountMonthsReferrer: 3,
		DiscountMonthsReferee:  1,
		IsActive:               true,
	}

	err = repo.CreateReferralCode(code)
	if err != nil {
		t.Fatalf("CreateReferralCode failed: %v", err)
	}

	// Record the referral use
	err = repo.RecordReferralUse(code.ID, tenantID)
	if err != nil {
		t.Fatalf("RecordReferralUse failed: %v", err)
	}

	// Verify the use was recorded
	uses, err := repo.GetReferralUses(code.ID)
	if err != nil {
		t.Fatalf("GetReferralUses failed: %v", err)
	}

	if len(uses) != 1 {
		t.Fatalf("Expected 1 referral use, got %d", len(uses))
	}

	if uses[0].CodeID != code.ID {
		t.Errorf("Expected code_id to be %d, got %d", code.ID, uses[0].CodeID)
	}

	if uses[0].RefereeTenantID != tenantID {
		t.Errorf("Expected referee_tenant_id to be %d, got %d", tenantID, uses[0].RefereeTenantID)
	}
}

func TestHasTenantUsedReferral_ReturnsFalseForNewTenant(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	repo := NewMarketingRepository(db)

	// Create a tenant
	_, err := db.Exec(`INSERT INTO tenants (slug, name) VALUES ('new-tenant', 'New Tenant')`)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	var tenantID int
	db.QueryRow(`SELECT id FROM tenants WHERE slug = 'new-tenant'`).Scan(&tenantID)

	// Check if tenant has used a referral - should be false
	hasUsed, err := repo.HasTenantUsedReferral(tenantID)
	if err != nil {
		t.Fatalf("HasTenantUsedReferral failed: %v", err)
	}

	if hasUsed {
		t.Error("New tenant should not have used a referral")
	}
}

func TestHasTenantUsedReferral_ReturnsTrueAfterUsing(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	repo := NewMarketingRepository(db)

	// Create a tenant
	_, err := db.Exec(`INSERT INTO tenants (slug, name) VALUES ('existing-tenant', 'Existing Tenant')`)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	var tenantID int
	db.QueryRow(`SELECT id FROM tenants WHERE slug = 'existing-tenant'`).Scan(&tenantID)

	// Create a referral code
	code := &models.ReferralCode{
		Code:                   "TESTHASUSED",
		DiscountMonthsReferrer: 3,
		DiscountMonthsReferee:  1,
		IsActive:               true,
	}
	repo.CreateReferralCode(code)

	// Record the use
	repo.RecordReferralUse(code.ID, tenantID)

	// Check if tenant has used a referral - should be true now
	hasUsed, err := repo.HasTenantUsedReferral(tenantID)
	if err != nil {
		t.Fatalf("HasTenantUsedReferral failed: %v", err)
	}

	if !hasUsed {
		t.Error("Tenant should have used a referral after RecordReferralUse")
	}
}
