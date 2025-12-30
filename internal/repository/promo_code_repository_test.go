package repository

import (
	"math"
	"testing"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// HIGH-8: Test SafeInt64ToInt for bounds checking
func TestSafeInt64ToInt(t *testing.T) {
	t.Run("converts valid positive value", func(t *testing.T) {
		val, err := SafeInt64ToInt(100)
		if err != nil {
			t.Fatalf("SafeInt64ToInt(100) error: %v", err)
		}
		if val != 100 {
			t.Errorf("Expected 100, got %d", val)
		}
	})

	t.Run("converts zero", func(t *testing.T) {
		val, err := SafeInt64ToInt(0)
		if err != nil {
			t.Fatalf("SafeInt64ToInt(0) error: %v", err)
		}
		if val != 0 {
			t.Errorf("Expected 0, got %d", val)
		}
	})

	t.Run("converts valid negative value", func(t *testing.T) {
		val, err := SafeInt64ToInt(-100)
		if err != nil {
			t.Fatalf("SafeInt64ToInt(-100) error: %v", err)
		}
		if val != -100 {
			t.Errorf("Expected -100, got %d", val)
		}
	})

	t.Run("converts MaxInt32 successfully", func(t *testing.T) {
		val, err := SafeInt64ToInt(math.MaxInt32)
		if err != nil {
			t.Fatalf("SafeInt64ToInt(MaxInt32) error: %v", err)
		}
		if val != math.MaxInt32 {
			t.Errorf("Expected MaxInt32, got %d", val)
		}
	})

	t.Run("converts MinInt32 successfully", func(t *testing.T) {
		val, err := SafeInt64ToInt(math.MinInt32)
		if err != nil {
			t.Fatalf("SafeInt64ToInt(MinInt32) error: %v", err)
		}
		if val != math.MinInt32 {
			t.Errorf("Expected MinInt32, got %d", val)
		}
	})

	t.Run("rejects value exceeding MaxInt32", func(t *testing.T) {
		_, err := SafeInt64ToInt(math.MaxInt32 + 1)
		if err == nil {
			t.Error("SafeInt64ToInt should reject value > MaxInt32")
		}
	})

	t.Run("rejects value below MinInt32", func(t *testing.T) {
		_, err := SafeInt64ToInt(math.MinInt32 - 1)
		if err == nil {
			t.Error("SafeInt64ToInt should reject value < MinInt32")
		}
	})

	t.Run("rejects very large int64", func(t *testing.T) {
		_, err := SafeInt64ToInt(math.MaxInt64)
		if err == nil {
			t.Error("SafeInt64ToInt should reject MaxInt64")
		}
	})

	t.Run("rejects very negative int64", func(t *testing.T) {
		_, err := SafeInt64ToInt(math.MinInt64)
		if err == nil {
			t.Error("SafeInt64ToInt should reject MinInt64")
		}
	})
}

func TestPromoCodeRepository_Create(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewPromoCodeRepository(db)

	t.Run("creates promo code successfully", func(t *testing.T) {
		code := &models.PromoCode{
			Code:          "WELCOME10",
			Description:   "Welcome discount",
			DiscountType:  "percentage",
			DiscountValue: 10,
			IsActive:      true,
		}

		err := repo.Create(code)
		if err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		if code.ID == 0 {
			t.Error("Expected ID to be set")
		}
		if code.CreatedAt.IsZero() {
			t.Error("Expected CreatedAt to be set")
		}
	})

	t.Run("uppercases code on create", func(t *testing.T) {
		code := &models.PromoCode{
			Code:          "lowercase",
			DiscountType:  "percentage",
			DiscountValue: 5,
			IsActive:      true,
		}

		if err := repo.Create(code); err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		retrieved, err := repo.GetByID(code.ID)
		if err != nil {
			t.Fatalf("GetByID() error: %v", err)
		}
		if retrieved.Code != "LOWERCASE" {
			t.Errorf("Expected code 'LOWERCASE', got '%s'", retrieved.Code)
		}
	})

	t.Run("creates code with max uses limit", func(t *testing.T) {
		maxUses := 100
		code := &models.PromoCode{
			Code:          "LIMITED100",
			DiscountType:  "fixed",
			DiscountValue: 500,
			MaxUses:       &maxUses,
			IsActive:      true,
		}

		if err := repo.Create(code); err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		retrieved, err := repo.GetByID(code.ID)
		if err != nil {
			t.Fatalf("GetByID() error: %v", err)
		}
		if retrieved.MaxUses == nil || *retrieved.MaxUses != 100 {
			t.Error("Expected MaxUses to be 100")
		}
	})

	t.Run("creates code with expiration date", func(t *testing.T) {
		expires := time.Now().Add(30 * 24 * time.Hour)
		code := &models.PromoCode{
			Code:          "EXPIRES30",
			DiscountType:  "percentage",
			DiscountValue: 20,
			ExpiresAt:     &expires,
			IsActive:      true,
		}

		if err := repo.Create(code); err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		retrieved, err := repo.GetByID(code.ID)
		if err != nil {
			t.Fatalf("GetByID() error: %v", err)
		}
		if retrieved.ExpiresAt == nil {
			t.Error("Expected ExpiresAt to be set")
		}
	})

	t.Run("creates code with Stripe coupon ID", func(t *testing.T) {
		couponID := "coupon_test_123"
		code := &models.PromoCode{
			Code:           "STRIPECOUPON",
			DiscountType:   "percentage",
			DiscountValue:  15,
			StripeCouponID: &couponID,
			IsActive:       true,
		}

		if err := repo.Create(code); err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		retrieved, err := repo.GetByID(code.ID)
		if err != nil {
			t.Fatalf("GetByID() error: %v", err)
		}
		if retrieved.StripeCouponID == nil || *retrieved.StripeCouponID != couponID {
			t.Error("Expected StripeCouponID to be saved")
		}
	})
}

func TestPromoCodeRepository_GetByID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewPromoCodeRepository(db)

	t.Run("returns nil for non-existent ID", func(t *testing.T) {
		code, err := repo.GetByID(99999)
		if err != nil {
			t.Fatalf("GetByID() error: %v", err)
		}
		if code != nil {
			t.Error("Expected nil for non-existent ID")
		}
	})

	t.Run("returns code with all fields", func(t *testing.T) {
		maxUses := 50
		expires := time.Now().Add(7 * 24 * time.Hour)
		couponID := "stripe_coupon_abc"

		original := &models.PromoCode{
			Code:           "FULLTEST",
			Description:    "Full test code",
			DiscountType:   "percentage",
			DiscountValue:  25,
			MaxUses:        &maxUses,
			ValidForPlans:  "pro,enterprise",
			IsActive:       true,
			StripeCouponID: &couponID,
			ExpiresAt:      &expires,
		}
		if err := repo.Create(original); err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		retrieved, err := repo.GetByID(original.ID)
		if err != nil {
			t.Fatalf("GetByID() error: %v", err)
		}
		if retrieved == nil {
			t.Fatal("Expected code, got nil")
		}

		if retrieved.Code != "FULLTEST" {
			t.Errorf("Expected code 'FULLTEST', got '%s'", retrieved.Code)
		}
		if retrieved.Description != "Full test code" {
			t.Errorf("Expected description 'Full test code', got '%s'", retrieved.Description)
		}
		if retrieved.DiscountType != "percentage" {
			t.Errorf("Expected discount type 'percentage', got '%s'", retrieved.DiscountType)
		}
		if retrieved.DiscountValue != 25 {
			t.Errorf("Expected discount value 25, got %d", retrieved.DiscountValue)
		}
		if retrieved.MaxUses == nil || *retrieved.MaxUses != 50 {
			t.Error("Expected MaxUses 50")
		}
		if retrieved.ValidForPlans != "pro,enterprise" {
			t.Errorf("Expected ValidForPlans 'pro,enterprise', got '%s'", retrieved.ValidForPlans)
		}
		if !retrieved.IsActive {
			t.Error("Expected IsActive true")
		}
	})
}

func TestPromoCodeRepository_GetByCode(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewPromoCodeRepository(db)

	t.Run("returns nil for non-existent code", func(t *testing.T) {
		code, err := repo.GetByCode("NONEXISTENT")
		if err != nil {
			t.Fatalf("GetByCode() error: %v", err)
		}
		if code != nil {
			t.Error("Expected nil for non-existent code")
		}
	})

	t.Run("finds code case-insensitively", func(t *testing.T) {
		original := &models.PromoCode{
			Code:          "CASETEST",
			DiscountType:  "percentage",
			DiscountValue: 10,
			IsActive:      true,
		}
		if err := repo.Create(original); err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		// Search with lowercase
		retrieved, err := repo.GetByCode("casetest")
		if err != nil {
			t.Fatalf("GetByCode() error: %v", err)
		}
		if retrieved == nil {
			t.Fatal("Expected to find code with lowercase search")
		}
		if retrieved.ID != original.ID {
			t.Errorf("Expected ID %d, got %d", original.ID, retrieved.ID)
		}
	})
}

func TestPromoCodeRepository_GetAll(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewPromoCodeRepository(db)

	t.Run("returns empty slice when no codes exist", func(t *testing.T) {
		codes, err := repo.GetAll(false)
		if err != nil {
			t.Fatalf("GetAll() error: %v", err)
		}
		if codes == nil {
			t.Error("Expected empty slice, got nil")
		}
	})

	t.Run("filters by active status", func(t *testing.T) {
		// Create active code
		active := &models.PromoCode{
			Code:          "ACTIVE1",
			DiscountType:  "percentage",
			DiscountValue: 10,
			IsActive:      true,
		}
		if err := repo.Create(active); err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		// Create inactive code
		inactive := &models.PromoCode{
			Code:          "INACTIVE1",
			DiscountType:  "percentage",
			DiscountValue: 5,
			IsActive:      false,
		}
		if err := repo.Create(inactive); err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		// Get all
		all, err := repo.GetAll(false)
		if err != nil {
			t.Fatalf("GetAll(false) error: %v", err)
		}

		// Get active only
		activeOnly, err := repo.GetAll(true)
		if err != nil {
			t.Fatalf("GetAll(true) error: %v", err)
		}

		if len(all) < len(activeOnly) {
			t.Error("Expected all codes >= active codes")
		}

		// Verify no inactive in active-only results
		for _, c := range activeOnly {
			if !c.IsActive {
				t.Error("Found inactive code in active-only results")
			}
		}
	})
}

func TestPromoCodeRepository_Update(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewPromoCodeRepository(db)

	t.Run("updates code successfully", func(t *testing.T) {
		code := &models.PromoCode{
			Code:          "UPDATE1",
			Description:   "Original description",
			DiscountType:  "percentage",
			DiscountValue: 10,
			IsActive:      true,
		}
		if err := repo.Create(code); err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		// Update
		code.Description = "Updated description"
		code.DiscountValue = 15
		code.IsActive = false

		if err := repo.Update(code); err != nil {
			t.Fatalf("Update() error: %v", err)
		}

		// Verify
		retrieved, err := repo.GetByID(code.ID)
		if err != nil {
			t.Fatalf("GetByID() error: %v", err)
		}
		if retrieved.Description != "Updated description" {
			t.Errorf("Expected description 'Updated description', got '%s'", retrieved.Description)
		}
		if retrieved.DiscountValue != 15 {
			t.Errorf("Expected discount value 15, got %d", retrieved.DiscountValue)
		}
		if retrieved.IsActive {
			t.Error("Expected IsActive false")
		}
	})

	t.Run("uppercases code on update", func(t *testing.T) {
		code := &models.PromoCode{
			Code:          "UPPER1",
			DiscountType:  "percentage",
			DiscountValue: 10,
			IsActive:      true,
		}
		if err := repo.Create(code); err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		code.Code = "lowercase_update"
		if err := repo.Update(code); err != nil {
			t.Fatalf("Update() error: %v", err)
		}

		retrieved, err := repo.GetByID(code.ID)
		if err != nil {
			t.Fatalf("GetByID() error: %v", err)
		}
		if retrieved.Code != "LOWERCASE_UPDATE" {
			t.Errorf("Expected code 'LOWERCASE_UPDATE', got '%s'", retrieved.Code)
		}
	})
}

func TestPromoCodeRepository_Delete(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewPromoCodeRepository(db)

	t.Run("deletes code successfully", func(t *testing.T) {
		code := &models.PromoCode{
			Code:          "DELETE1",
			DiscountType:  "percentage",
			DiscountValue: 10,
			IsActive:      true,
		}
		if err := repo.Create(code); err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		if err := repo.Delete(code.ID); err != nil {
			t.Fatalf("Delete() error: %v", err)
		}

		// Verify deleted
		retrieved, err := repo.GetByID(code.ID)
		if err != nil {
			t.Fatalf("GetByID() error: %v", err)
		}
		if retrieved != nil {
			t.Error("Expected code to be deleted")
		}
	})

	t.Run("no error for non-existent ID", func(t *testing.T) {
		err := repo.Delete(99999)
		if err != nil {
			t.Fatalf("Delete() should not error for non-existent ID: %v", err)
		}
	})
}

func TestPromoCodeRepository_IncrementUsesCount(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewPromoCodeRepository(db)

	t.Run("increments uses count", func(t *testing.T) {
		code := &models.PromoCode{
			Code:          "INCREMENT1",
			DiscountType:  "percentage",
			DiscountValue: 10,
			UsesCount:     0,
			IsActive:      true,
		}
		if err := repo.Create(code); err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		// Increment
		if err := repo.IncrementUsesCount(code.ID); err != nil {
			t.Fatalf("IncrementUsesCount() error: %v", err)
		}

		// Verify
		retrieved, err := repo.GetByID(code.ID)
		if err != nil {
			t.Fatalf("GetByID() error: %v", err)
		}
		if retrieved.UsesCount != 1 {
			t.Errorf("Expected uses count 1, got %d", retrieved.UsesCount)
		}

		// Increment again
		if err := repo.IncrementUsesCount(code.ID); err != nil {
			t.Fatalf("IncrementUsesCount() error: %v", err)
		}

		retrieved, err = repo.GetByID(code.ID)
		if err != nil {
			t.Fatalf("GetByID() error: %v", err)
		}
		if retrieved.UsesCount != 2 {
			t.Errorf("Expected uses count 2, got %d", retrieved.UsesCount)
		}
	})
}

func TestPromoCodeRepository_RecordUse(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewPromoCodeRepository(db)
	tenantID := createTestTenant(t, db)

	t.Run("records promo code use", func(t *testing.T) {
		code := &models.PromoCode{
			Code:          "RECORD1",
			DiscountType:  "percentage",
			DiscountValue: 10,
			IsActive:      true,
		}
		if err := repo.Create(code); err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		err := repo.RecordUse(code.ID, tenantID)
		if err != nil {
			t.Fatalf("RecordUse() error: %v", err)
		}

		// Verify usage was recorded
		hasUsed, err := repo.HasTenantUsedCode(code.ID, tenantID)
		if err != nil {
			t.Fatalf("HasTenantUsedCode() error: %v", err)
		}
		if !hasUsed {
			t.Error("Expected tenant to have used code")
		}
	})
}

func TestPromoCodeRepository_HasTenantUsedCode(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewPromoCodeRepository(db)
	tenantID := createTestTenant(t, db)

	t.Run("returns false when not used", func(t *testing.T) {
		code := &models.PromoCode{
			Code:          "NOTUSED",
			DiscountType:  "percentage",
			DiscountValue: 10,
			IsActive:      true,
		}
		if err := repo.Create(code); err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		hasUsed, err := repo.HasTenantUsedCode(code.ID, tenantID)
		if err != nil {
			t.Fatalf("HasTenantUsedCode() error: %v", err)
		}
		if hasUsed {
			t.Error("Expected false for unused code")
		}
	})

	t.Run("returns true after use", func(t *testing.T) {
		code := &models.PromoCode{
			Code:          "WILLUSE",
			DiscountType:  "percentage",
			DiscountValue: 10,
			IsActive:      true,
		}
		if err := repo.Create(code); err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		if err := repo.RecordUse(code.ID, tenantID); err != nil {
			t.Fatalf("RecordUse() error: %v", err)
		}

		hasUsed, err := repo.HasTenantUsedCode(code.ID, tenantID)
		if err != nil {
			t.Fatalf("HasTenantUsedCode() error: %v", err)
		}
		if !hasUsed {
			t.Error("Expected true after use")
		}
	})

	t.Run("different tenants have independent usage", func(t *testing.T) {
		code := &models.PromoCode{
			Code:          "MULTITENANT",
			DiscountType:  "percentage",
			DiscountValue: 10,
			IsActive:      true,
		}
		if err := repo.Create(code); err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		// Record use for first tenant
		if err := repo.RecordUse(code.ID, tenantID); err != nil {
			t.Fatalf("RecordUse() error: %v", err)
		}

		// Check second tenant
		tenant2ID := createTestTenant(t, db)
		hasUsed, err := repo.HasTenantUsedCode(code.ID, tenant2ID)
		if err != nil {
			t.Fatalf("HasTenantUsedCode() error: %v", err)
		}
		if hasUsed {
			t.Error("Expected false for different tenant")
		}
	})
}

func TestPromoCodeRepository_GetCodeUses(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewPromoCodeRepository(db)
	tenantID := createTestTenant(t, db)

	t.Run("returns empty slice for unused code", func(t *testing.T) {
		code := &models.PromoCode{
			Code:          "NOUSES",
			DiscountType:  "percentage",
			DiscountValue: 10,
			IsActive:      true,
		}
		if err := repo.Create(code); err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		uses, err := repo.GetCodeUses(code.ID)
		if err != nil {
			t.Fatalf("GetCodeUses() error: %v", err)
		}
		if uses == nil {
			t.Error("Expected empty slice, got nil")
		}
		if len(uses) != 0 {
			t.Errorf("Expected 0 uses, got %d", len(uses))
		}
	})

	t.Run("returns uses with tenant info", func(t *testing.T) {
		code := &models.PromoCode{
			Code:          "WITHUSES",
			DiscountType:  "percentage",
			DiscountValue: 10,
			IsActive:      true,
		}
		if err := repo.Create(code); err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		if err := repo.RecordUse(code.ID, tenantID); err != nil {
			t.Fatalf("RecordUse() error: %v", err)
		}

		uses, err := repo.GetCodeUses(code.ID)
		if err != nil {
			t.Fatalf("GetCodeUses() error: %v", err)
		}
		if len(uses) != 1 {
			t.Fatalf("Expected 1 use, got %d", len(uses))
		}

		use := uses[0]
		if use.PromoCodeID != code.ID {
			t.Errorf("Expected PromoCodeID %d, got %d", code.ID, use.PromoCodeID)
		}
		if use.TenantID != tenantID {
			t.Errorf("Expected TenantID %d, got %d", tenantID, use.TenantID)
		}
		if use.TenantName == nil {
			t.Error("Expected TenantName to be populated")
		}
		if use.Code == nil {
			t.Error("Expected Code to be populated")
		}
	})
}
