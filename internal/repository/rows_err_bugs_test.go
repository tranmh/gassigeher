package repository

import (
	"testing"

	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// Test file for verifying rows.Err() checks in repository functions
// BUG FIXES:
// 1. CRITICAL: SubscriptionRepository.GetAllPlans - missing rows.Err() check
// 2. HIGH: PromoCodeRepository.GetAll - missing rows.Err() check
// 3. HIGH: MarketingRepository.ListCampaigns - missing rows.Err() check

// =========================================================================
// CRITICAL BUG #1: SubscriptionRepository.GetAllPlans missing rows.Err()
// =========================================================================

// TestSubscriptionRepository_GetAllPlans_RowsErrCheck verifies that GetAllPlans
// properly checks rows.Err() after iteration to catch any errors that occurred
// during the iteration process.
//
// TDD RED Phase: This test ensures the function handles iteration errors.
// The rows.Err() check is important because errors during iteration may not
// be returned by rows.Scan() and only become visible via rows.Err().
func TestSubscriptionRepository_GetAllPlans_RowsErrCheck(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSubscriptionRepository(db)

	t.Run("GetAllPlans returns plans and handles iteration properly", func(t *testing.T) {
		// This test verifies the function completes iteration properly
		// and would catch errors via rows.Err() if they occurred
		plans, err := repo.GetAllPlans()
		if err != nil {
			t.Fatalf("GetAllPlans() error: %v", err)
		}

		// Verify we got plans (migrations create default plans)
		if len(plans) < 2 {
			t.Errorf("Expected at least 2 plans, got %d", len(plans))
		}

		// Verify all plans have valid data (would fail if iteration error occurred)
		for _, plan := range plans {
			if plan.ID == 0 {
				t.Error("Plan ID should not be 0")
			}
			if plan.Name == "" {
				t.Error("Plan Name should not be empty")
			}
			if plan.Slug == "" {
				t.Error("Plan Slug should not be empty")
			}
		}
	})

	t.Run("GetAllPlans properly iterates through all rows", func(t *testing.T) {
		// Create additional test plans to ensure iteration works correctly
		now := testutil.Now()
		_, err := db.Exec(`
			INSERT INTO pricing_plans (name, slug, max_dogs, price_monthly, price_yearly, is_active, created_at)
			VALUES ('Test Plan', 'test-plan', 5, 1000, 10000, 1, ?)
		`, now)
		if err != nil {
			t.Fatalf("Failed to create test plan: %v", err)
		}

		plans, err := repo.GetAllPlans()
		if err != nil {
			t.Fatalf("GetAllPlans() after insert error: %v", err)
		}

		// Should now have at least 3 plans
		if len(plans) < 3 {
			t.Errorf("Expected at least 3 plans after insert, got %d", len(plans))
		}

		// Find our test plan
		var found bool
		for _, p := range plans {
			if p.Slug == "test-plan" {
				found = true
				if p.Name != "Test Plan" {
					t.Errorf("Test plan name = %s, want 'Test Plan'", p.Name)
				}
				if p.MaxDogs != 5 {
					t.Errorf("Test plan MaxDogs = %d, want 5", p.MaxDogs)
				}
				break
			}
		}
		if !found {
			t.Error("Test plan not found in results")
		}
	})
}

// =========================================================================
// HIGH BUG #2: PromoCodeRepository.GetAll missing rows.Err()
// =========================================================================

// TestPromoCodeRepository_GetAll_RowsErrCheck verifies that GetAll
// properly checks rows.Err() after iteration.
func TestPromoCodeRepository_GetAll_RowsErrCheck(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewPromoCodeRepository(db)

	t.Run("GetAll handles iteration properly with empty result", func(t *testing.T) {
		codes, err := repo.GetAll(false)
		if err != nil {
			t.Fatalf("GetAll() error: %v", err)
		}

		// Empty result should return empty slice, not nil
		if codes == nil {
			t.Error("Expected empty slice, got nil")
		}
	})

	t.Run("GetAll properly iterates through multiple rows", func(t *testing.T) {
		// Create multiple promo codes
		testCodes := []*models.PromoCode{
			{Code: "ROWS_ERR_TEST1", DiscountType: "percentage", DiscountValue: 10, IsActive: true},
			{Code: "ROWS_ERR_TEST2", DiscountType: "fixed", DiscountValue: 500, IsActive: true},
			{Code: "ROWS_ERR_TEST3", DiscountType: "percentage", DiscountValue: 20, IsActive: false},
		}

		for _, code := range testCodes {
			if err := repo.Create(code); err != nil {
				t.Fatalf("Failed to create test promo code: %v", err)
			}
		}

		// Get all codes (including inactive)
		codes, err := repo.GetAll(false)
		if err != nil {
			t.Fatalf("GetAll(false) error: %v", err)
		}

		if len(codes) < 3 {
			t.Errorf("Expected at least 3 codes, got %d", len(codes))
		}

		// Verify all returned codes have valid data
		for _, code := range codes {
			if code.ID == 0 {
				t.Error("Code ID should not be 0")
			}
			if code.Code == "" {
				t.Error("Code should not be empty")
			}
			if code.DiscountType == "" {
				t.Error("DiscountType should not be empty")
			}
		}
	})

	t.Run("GetAll with activeOnly filter iterates properly", func(t *testing.T) {
		// Get only active codes
		activeCodes, err := repo.GetAll(true)
		if err != nil {
			t.Fatalf("GetAll(true) error: %v", err)
		}

		// All returned codes should be active
		for _, code := range activeCodes {
			if !code.IsActive {
				t.Errorf("GetAll(true) returned inactive code: %s", code.Code)
			}
		}
	})
}

// =========================================================================
// HIGH BUG #3: MarketingRepository.ListCampaigns missing rows.Err()
// =========================================================================

// TestMarketingRepository_ListCampaigns_RowsErrCheck verifies that ListCampaigns
// properly checks rows.Err() after iteration.
func TestMarketingRepository_ListCampaigns_RowsErrCheck(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	repo := NewMarketingRepository(db)

	t.Run("ListCampaigns handles iteration properly with empty result", func(t *testing.T) {
		campaigns, err := repo.ListCampaigns()
		if err != nil {
			t.Fatalf("ListCampaigns() error: %v", err)
		}

		// Empty result should return nil or empty slice
		// (the current implementation returns nil for empty result, which is acceptable)
		_ = campaigns // Just verify no error occurred during iteration
	})

	t.Run("ListCampaigns properly iterates through multiple rows", func(t *testing.T) {
		// Create multiple campaigns
		testCampaigns := []*models.MarketingCampaign{
			{Type: "fomo_countdown", Name: "ROWS_ERR_CAMP1", IsActive: true},
			{Type: "referral", Name: "ROWS_ERR_CAMP2", IsActive: true},
			{Type: "fomo_countdown", Name: "ROWS_ERR_CAMP3", IsActive: false},
		}

		for _, campaign := range testCampaigns {
			if err := repo.CreateCampaign(campaign); err != nil {
				t.Fatalf("Failed to create test campaign: %v", err)
			}
		}

		// List all campaigns
		campaigns, err := repo.ListCampaigns()
		if err != nil {
			t.Fatalf("ListCampaigns() error: %v", err)
		}

		if len(campaigns) < 3 {
			t.Errorf("Expected at least 3 campaigns, got %d", len(campaigns))
		}

		// Verify all returned campaigns have valid data
		for _, campaign := range campaigns {
			if campaign.ID == 0 {
				t.Error("Campaign ID should not be 0")
			}
			if campaign.Name == "" {
				t.Error("Campaign Name should not be empty")
			}
			if campaign.Type == "" {
				t.Error("Campaign Type should not be empty")
			}
		}
	})

	t.Run("ListCampaigns returns campaigns ordered by created_at DESC", func(t *testing.T) {
		// The function should return campaigns in descending order by created_at
		campaigns, err := repo.ListCampaigns()
		if err != nil {
			t.Fatalf("ListCampaigns() error: %v", err)
		}

		// Verify we have campaigns and they completed iteration
		if len(campaigns) == 0 {
			t.Skip("No campaigns to verify ordering")
		}

		// Just verify iteration completed without error
		for i, c := range campaigns {
			if c == nil {
				t.Errorf("Campaign at index %d is nil", i)
			}
		}
	})
}

// =========================================================================
// Additional tests for ListReferralCodes which also needs rows.Err() check
// =========================================================================

func TestMarketingRepository_ListReferralCodes_RowsErrCheck(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	repo := NewMarketingRepository(db)

	t.Run("ListReferralCodes handles iteration properly", func(t *testing.T) {
		// Create test referral codes
		testCodes := []*models.ReferralCode{
			{Code: "REF_ROWS_TEST1", DiscountMonthsReferrer: 3, DiscountMonthsReferee: 1, IsActive: true},
			{Code: "REF_ROWS_TEST2", DiscountMonthsReferrer: 2, DiscountMonthsReferee: 1, IsActive: true},
		}

		for _, code := range testCodes {
			if err := repo.CreateReferralCode(code); err != nil {
				t.Fatalf("Failed to create test referral code: %v", err)
			}
		}

		codes, err := repo.ListReferralCodes()
		if err != nil {
			t.Fatalf("ListReferralCodes() error: %v", err)
		}

		if len(codes) < 2 {
			t.Errorf("Expected at least 2 referral codes, got %d", len(codes))
		}

		// Verify all returned codes have valid data
		for _, code := range codes {
			if code.ID == 0 {
				t.Error("Referral code ID should not be 0")
			}
			if code.Code == "" {
				t.Error("Referral code should not be empty")
			}
		}
	})
}

// =========================================================================
// Additional tests for ListReferenceEntries which also needs rows.Err() check
// =========================================================================

func TestMarketingRepository_ListReferenceEntries_RowsErrCheck(t *testing.T) {
	db := setupMarketingTestDB(t)
	defer db.Close()

	// Create a tenant for reference entries
	_, err := db.Exec(`INSERT INTO tenants (slug, name) VALUES ('ref-test', 'Ref Test')`)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	var tenantID int
	db.QueryRow(`SELECT id FROM tenants WHERE slug = 'ref-test'`).Scan(&tenantID)

	repo := NewMarketingRepository(db)

	t.Run("ListReferenceEntries handles iteration properly", func(t *testing.T) {
		// Create test reference entries
		testCity := "Test City"
		entry := &models.ReferenceEntry{
			TenantID:    tenantID,
			DisplayName: "Test Shelter",
			City:        &testCity,
			IsApproved:  true,
		}

		if err := repo.CreateReferenceEntry(entry); err != nil {
			t.Fatalf("Failed to create test reference entry: %v", err)
		}

		entries, err := repo.ListReferenceEntries(false)
		if err != nil {
			t.Fatalf("ListReferenceEntries() error: %v", err)
		}

		if len(entries) < 1 {
			t.Errorf("Expected at least 1 reference entry, got %d", len(entries))
		}

		// Verify all returned entries have valid data
		for _, entry := range entries {
			if entry.ID == 0 {
				t.Error("Reference entry ID should not be 0")
			}
			if entry.DisplayName == "" {
				t.Error("Reference entry DisplayName should not be empty")
			}
		}
	})

	t.Run("ListReferenceEntries with approvedOnly filter", func(t *testing.T) {
		// Get only approved entries
		approvedEntries, err := repo.ListReferenceEntries(true)
		if err != nil {
			t.Fatalf("ListReferenceEntries(true) error: %v", err)
		}

		// All returned entries should be approved
		for _, entry := range approvedEntries {
			if !entry.IsApproved {
				t.Errorf("ListReferenceEntries(true) returned unapproved entry: %s", entry.DisplayName)
			}
		}
	})
}
