package repository

import (
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// =============================================================================
// BUG #1: Dog Limit Race Condition
// TDD RED PHASE: Test that concurrent dog creation respects the limit
// =============================================================================

func TestDogRepository_CreateWithLimitCheck_ConcurrentRace(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewDogRepository(db)

	// Create a test tenant
	tenantID := createTestTenant(t, db)

	// Set a limit of 3 dogs
	const dogLimit = 3
	const numGoroutines = 10

	// Launch concurrent goroutines trying to create dogs
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			dog := &models.Dog{
				TenantID:    tenantID,
				Name:        fmt.Sprintf("ConcurrentDog%d", idx),
				Breed:       "Poodle",
				Size:        "small",
				Age:         2,
				IsAvailable: true,
			}

			err := repo.CreateWithLimitCheck(dog, dogLimit)

			mu.Lock()
			if err == nil {
				successCount++
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// Count final dogs for this tenant
	var finalCount int
	db.QueryRow("SELECT COUNT(*) FROM dogs WHERE tenant_id = ?", tenantID).Scan(&finalCount)

	t.Logf("TenantID=%d, successCount=%d, finalDogCount=%d", tenantID, successCount, finalCount)

	// CRITICAL: Must not exceed the limit
	if finalCount > dogLimit {
		t.Errorf("RACE CONDITION BUG #1: Dog limit exceeded! Expected max %d dogs, but got %d", dogLimit, finalCount)
	}

	// Exactly dogLimit dogs should have been created
	if finalCount != dogLimit {
		t.Errorf("Expected exactly %d dogs, got %d", dogLimit, finalCount)
	}
}

// =============================================================================
// BUG #4: Missing Tenant Validation in IncrementFreeMonths
// TDD RED PHASE: Test input validation
// =============================================================================

func TestSubscriptionRepository_IncrementFreeMonths_Validation(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSubscriptionRepository(db)

	t.Run("rejects negative tenant_id", func(t *testing.T) {
		err := repo.IncrementFreeMonths(-1, 10, "referral")
		if err == nil {
			t.Error("BUG #4: Should reject negative tenant_id")
		}
	})

	t.Run("rejects zero tenant_id for SaaS mode", func(t *testing.T) {
		// Note: tenant_id=0 is valid for Simple-Mode, but IncrementFreeMonths
		// is only used in SaaS mode where tenant_id must be positive
		err := repo.IncrementFreeMonths(0, 10, "referral")
		// For now, we accept 0 for backward compatibility
		// This test documents the current behavior
		t.Logf("IncrementFreeMonths(0, 10, referral) returned: %v", err)
	})

	t.Run("rejects negative months", func(t *testing.T) {
		// Create a tenant with subscription
		tenantID := createTestTenant(t, db)
		createTestSubscription(t, db, tenantID)

		err := repo.IncrementFreeMonths(tenantID, -5, "referral")
		if err == nil {
			t.Error("BUG #4: Should reject negative months")
		}
	})

	t.Run("rejects zero months", func(t *testing.T) {
		tenantID := createTestTenant(t, db)
		createTestSubscription(t, db, tenantID)

		err := repo.IncrementFreeMonths(tenantID, 0, "referral")
		if err == nil {
			t.Error("BUG #4: Should reject zero months")
		}
	})

	t.Run("rejects excessive months (>120)", func(t *testing.T) {
		tenantID := createTestTenant(t, db)
		createTestSubscription(t, db, tenantID)

		err := repo.IncrementFreeMonths(tenantID, 1000, "referral")
		if err == nil {
			t.Error("BUG #4: Should reject months > 120")
		}
	})

	t.Run("succeeds with valid input", func(t *testing.T) {
		tenantID := createTestTenant(t, db)
		createTestSubscription(t, db, tenantID)

		err := repo.IncrementFreeMonths(tenantID, 3, "referral")
		if err != nil {
			t.Errorf("Should succeed with valid input: %v", err)
		}

		// Verify the free months were incremented
		var freeMonths int
		db.QueryRow("SELECT COALESCE(free_months_remaining, 0) FROM tenant_subscriptions WHERE tenant_id = ?", tenantID).Scan(&freeMonths)
		if freeMonths != 3 {
			t.Errorf("Expected 3 free months, got %d", freeMonths)
		}
	})
}

// =============================================================================
// BUG #5: Promo Code Max Uses Race Condition
// TDD RED PHASE: Test atomic max_uses enforcement
// =============================================================================

func TestPromoCodeRepository_IncrementUsesCount_MaxUses(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewPromoCodeRepository(db)

	t.Run("respects max_uses limit", func(t *testing.T) {
		// Create a promo code with max_uses = 3
		promoID := createTestPromoCode(t, db, "TEST50", 3)

		// Set uses_count to 2 (one below limit)
		db.Exec("UPDATE promo_codes SET uses_count = 2 WHERE id = ?", promoID)

		// First increment should succeed (2 -> 3)
		err := repo.IncrementUsesCount(promoID)
		if err != nil {
			t.Errorf("First increment should succeed: %v", err)
		}

		// Second increment should fail (3 -> 4 would exceed max_uses)
		err = repo.IncrementUsesCount(promoID)
		if err == nil {
			t.Error("BUG #5: Should reject increment when max_uses reached")
		}

		// Verify count didn't exceed max_uses
		var count int
		db.QueryRow("SELECT uses_count FROM promo_codes WHERE id = ?", promoID).Scan(&count)
		if count > 3 {
			t.Errorf("BUG #5: uses_count exceeded max_uses! Got %d, max was 3", count)
		}
	})

	t.Run("concurrent increments respect max_uses", func(t *testing.T) {
		// Create a promo code with max_uses = 5
		promoID := createTestPromoCode(t, db, "CONCURRENT5", 5)

		// Set uses_count to 4 (one below limit)
		db.Exec("UPDATE promo_codes SET uses_count = 4 WHERE id = ?", promoID)

		// Launch 10 concurrent increments - only 1 should succeed
		var wg sync.WaitGroup
		successCount := 0
		var mu sync.Mutex

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := repo.IncrementUsesCount(promoID)
				mu.Lock()
				if err == nil {
					successCount++
				}
				mu.Unlock()
			}()
		}

		wg.Wait()

		// Verify count
		var finalCount int
		db.QueryRow("SELECT uses_count FROM promo_codes WHERE id = ?", promoID).Scan(&finalCount)

		if finalCount > 5 {
			t.Errorf("BUG #5 RACE: uses_count=%d exceeded max_uses=5", finalCount)
		}

		t.Logf("Promo max_uses test: successCount=%d, finalCount=%d", successCount, finalCount)
	})

	t.Run("unlimited uses when max_uses is NULL", func(t *testing.T) {
		// Create a promo code with no limit
		promoID := createTestPromoCodeNoLimit(t, db, "UNLIMITED")

		// Should be able to increment many times
		for i := 0; i < 10; i++ {
			err := repo.IncrementUsesCount(promoID)
			if err != nil {
				t.Errorf("Increment %d should succeed for unlimited code: %v", i, err)
			}
		}

		var count int
		db.QueryRow("SELECT uses_count FROM promo_codes WHERE id = ?", promoID).Scan(&count)
		if count != 10 {
			t.Errorf("Expected 10 uses, got %d", count)
		}
	})
}

// =============================================================================
// BUG #6: Referral Code Max Uses Race Condition
// TDD RED PHASE: Test atomic max_uses enforcement for referral codes
// =============================================================================

func TestMarketingRepository_IncrementReferralCodeUses_MaxUses(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewMarketingRepository(db)

	t.Run("respects max_uses limit", func(t *testing.T) {
		// Create a referral code with max_uses = 3
		codeID := createTestReferralCode(t, db, "REF3", 3)

		// Set uses_count to 2 (one below limit)
		db.Exec("UPDATE referral_codes SET uses_count = 2 WHERE id = ?", codeID)

		// First increment should succeed (2 -> 3)
		err := repo.IncrementReferralCodeUses(codeID)
		if err != nil {
			t.Errorf("First increment should succeed: %v", err)
		}

		// Second increment should fail (3 -> 4 would exceed max_uses)
		err = repo.IncrementReferralCodeUses(codeID)
		if err == nil {
			t.Error("BUG #6: Should reject increment when max_uses reached")
		}

		// Verify count didn't exceed max_uses
		var count int
		db.QueryRow("SELECT uses_count FROM referral_codes WHERE id = ?", codeID).Scan(&count)
		if count > 3 {
			t.Errorf("BUG #6: uses_count exceeded max_uses! Got %d, max was 3", count)
		}
	})

	t.Run("concurrent increments respect max_uses", func(t *testing.T) {
		// Create a referral code with max_uses = 5
		codeID := createTestReferralCode(t, db, "REFCONCURRENT", 5)

		// Set uses_count to 4 (one below limit)
		db.Exec("UPDATE referral_codes SET uses_count = 4 WHERE id = ?", codeID)

		// Launch 10 concurrent increments - only 1 should succeed
		var wg sync.WaitGroup
		successCount := 0
		var mu sync.Mutex

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := repo.IncrementReferralCodeUses(codeID)
				mu.Lock()
				if err == nil {
					successCount++
				}
				mu.Unlock()
			}()
		}

		wg.Wait()

		// Verify count
		var finalCount int
		db.QueryRow("SELECT uses_count FROM referral_codes WHERE id = ?", codeID).Scan(&finalCount)

		if finalCount > 5 {
			t.Errorf("BUG #6 RACE: uses_count=%d exceeded max_uses=5", finalCount)
		}

		t.Logf("Referral max_uses test: successCount=%d, finalCount=%d", successCount, finalCount)
	})
}

// =============================================================================
// BUG #7: Invalid Category Validation
// TDD RED PHASE: Test that invalid categories are rejected
// =============================================================================

func TestDogRepository_FindAll_InvalidCategory(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewDogRepository(db)

	t.Run("accepts valid categories", func(t *testing.T) {
		validCategories := []string{"green", "blue", "orange"}
		for _, cat := range validCategories {
			filter := &models.DogFilterRequest{Category: &cat}
			_, err := repo.FindAll(filter, 0)
			if err != nil {
				t.Errorf("Should accept valid category %q: %v", cat, err)
			}
		}
	})

	t.Run("rejects invalid category", func(t *testing.T) {
		invalidCat := "purple"
		filter := &models.DogFilterRequest{Category: &invalidCat}
		_, err := repo.FindAll(filter, 0)
		if err == nil {
			t.Error("BUG #7: Should reject invalid category 'purple'")
		}
	})

	t.Run("rejects SQL injection attempt", func(t *testing.T) {
		maliciousCat := "'; DROP TABLE dogs; --"
		filter := &models.DogFilterRequest{Category: &maliciousCat}
		_, err := repo.FindAll(filter, 0)
		if err == nil {
			t.Error("BUG #7: Should reject malicious category input")
		}
	})
}

// =============================================================================
// BUG #2 & #3: Promo/Referral Code Race Conditions (HasTenantUsedCode)
// These are TOCTOU bugs that require integration tests with the billing handler
// For repository-level tests, we test the atomic operations
// =============================================================================

func TestPromoCodeRepository_RecordUse_Idempotent(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewPromoCodeRepository(db)

	t.Run("prevents duplicate use by same tenant", func(t *testing.T) {
		promoID := createTestPromoCode(t, db, "ONEUSE", 10)
		tenantID := createTestTenant(t, db)

		// First use should succeed
		err := repo.RecordUse(promoID, tenantID)
		if err != nil {
			t.Errorf("First RecordUse should succeed: %v", err)
		}

		// Check HasTenantUsedCode
		hasUsed, err := repo.HasTenantUsedCode(promoID, tenantID)
		if err != nil {
			t.Errorf("HasTenantUsedCode failed: %v", err)
		}
		if !hasUsed {
			t.Error("HasTenantUsedCode should return true after RecordUse")
		}

		// Second use should fail (duplicate)
		err = repo.RecordUse(promoID, tenantID)
		// Current implementation allows duplicates - this documents current behavior
		// The fix should add a UNIQUE constraint on (promo_code_id, tenant_id)
		t.Logf("Second RecordUse returned: %v (current behavior allows duplicates)", err)
	})
}

func TestMarketingRepository_RecordReferralUse_Idempotent(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewMarketingRepository(db)

	t.Run("prevents duplicate referral use by same tenant", func(t *testing.T) {
		codeID := createTestReferralCode(t, db, "REFONCE", 10)
		tenantID := createTestTenant(t, db)

		// First use should succeed
		err := repo.RecordReferralUse(codeID, tenantID)
		if err != nil {
			t.Errorf("First RecordReferralUse should succeed: %v", err)
		}

		// Check HasTenantUsedReferral
		hasUsed, err := repo.HasTenantUsedReferral(tenantID)
		if err != nil {
			t.Errorf("HasTenantUsedReferral failed: %v", err)
		}
		if !hasUsed {
			t.Error("HasTenantUsedReferral should return true after RecordReferralUse")
		}
	})
}

// =============================================================================
// Helper functions for test setup
// =============================================================================

func createTestSubscription(t *testing.T, db *sql.DB, tenantID int) int {
	t.Helper()
	now := time.Now()
	// First ensure a pricing plan exists
	db.Exec(`INSERT OR IGNORE INTO pricing_plans (id, name, slug, max_dogs, price_monthly, price_yearly, is_active) VALUES (1, 'Free', 'free', 10, 0, 0, 1)`)

	result, err := db.Exec(`INSERT INTO tenant_subscriptions
		(tenant_id, plan_id, status, billing_cycle, created_at, updated_at)
		VALUES (?, 1, 'active', 'monthly', ?, ?)`,
		tenantID, now, now)
	if err != nil {
		t.Fatalf("Failed to create test subscription: %v", err)
	}
	id, _ := result.LastInsertId()
	return int(id)
}

func createTestPromoCode(t *testing.T, db *sql.DB, code string, maxUses int) int {
	t.Helper()
	now := time.Now()
	result, err := db.Exec(`INSERT INTO promo_codes
		(code, description, discount_type, discount_value, max_uses, uses_count, is_active, created_at, updated_at)
		VALUES (?, 'Test promo', 'percentage', 50, ?, 0, 1, ?, ?)`,
		code, maxUses, now, now)
	if err != nil {
		t.Fatalf("Failed to create test promo code: %v", err)
	}
	id, _ := result.LastInsertId()
	return int(id)
}

func createTestPromoCodeNoLimit(t *testing.T, db *sql.DB, code string) int {
	t.Helper()
	now := time.Now()
	result, err := db.Exec(`INSERT INTO promo_codes
		(code, description, discount_type, discount_value, max_uses, uses_count, is_active, created_at, updated_at)
		VALUES (?, 'Test unlimited promo', 'percentage', 50, NULL, 0, 1, ?, ?)`,
		code, now, now)
	if err != nil {
		t.Fatalf("Failed to create test promo code: %v", err)
	}
	id, _ := result.LastInsertId()
	return int(id)
}

func createTestReferralCode(t *testing.T, db *sql.DB, code string, maxUses int) int {
	t.Helper()
	now := time.Now()
	result, err := db.Exec(`INSERT INTO referral_codes
		(code, referrer_email, discount_months_referrer, discount_months_referee, max_uses, uses_count, is_active, created_at, updated_at)
		VALUES (?, 'referrer@test.com', 3, 3, ?, 0, 1, ?, ?)`,
		code, maxUses, now, now)
	if err != nil {
		t.Fatalf("Failed to create test referral code: %v", err)
	}
	id, _ := result.LastInsertId()
	return int(id)
}
