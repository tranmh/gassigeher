package repository

import (
	"testing"

	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// TestSubscriptionRepository_GetAllPlans tests fetching all pricing plans (TDD RED Phase)
func TestSubscriptionRepository_GetAllPlans(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSubscriptionRepository(db)

	t.Run("returns default plans from migration", func(t *testing.T) {
		plans, err := repo.GetAllPlans()
		if err != nil {
			t.Fatalf("GetAllPlans() error: %v", err)
		}

		if len(plans) < 2 {
			t.Errorf("Expected at least 2 plans (Free, Pro), got %d", len(plans))
		}

		// Verify Free plan exists
		var freePlan *models.PricingPlan
		for _, p := range plans {
			if p.Slug == "free" {
				freePlan = p
				break
			}
		}
		if freePlan == nil {
			t.Error("Expected to find 'free' plan")
		} else {
			if freePlan.MaxDogs != 10 {
				t.Errorf("Free plan MaxDogs = %d, want 10", freePlan.MaxDogs)
			}
			if freePlan.PriceMonthly != 0 {
				t.Errorf("Free plan PriceMonthly = %d, want 0", freePlan.PriceMonthly)
			}
		}

		// Verify Pro plan exists
		var proPlan *models.PricingPlan
		for _, p := range plans {
			if p.Slug == "pro" {
				proPlan = p
				break
			}
		}
		if proPlan == nil {
			t.Error("Expected to find 'pro' plan")
		} else {
			if proPlan.MaxDogs != -1 {
				t.Errorf("Pro plan MaxDogs = %d, want -1 (unlimited)", proPlan.MaxDogs)
			}
			if proPlan.PriceMonthly != 2900 {
				t.Errorf("Pro plan PriceMonthly = %d, want 2900", proPlan.PriceMonthly)
			}
		}
	})
}

// TestSubscriptionRepository_GetPlanBySlug tests fetching a plan by slug
func TestSubscriptionRepository_GetPlanBySlug(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSubscriptionRepository(db)

	t.Run("finds existing plan", func(t *testing.T) {
		plan, err := repo.GetPlanBySlug("free")
		if err != nil {
			t.Fatalf("GetPlanBySlug() error: %v", err)
		}
		if plan == nil {
			t.Fatal("Expected plan, got nil")
		}
		if plan.Slug != "free" {
			t.Errorf("Plan slug = %s, want 'free'", plan.Slug)
		}
	})

	t.Run("returns nil for non-existent plan", func(t *testing.T) {
		plan, err := repo.GetPlanBySlug("nonexistent")
		if err != nil {
			t.Fatalf("GetPlanBySlug() error: %v", err)
		}
		if plan != nil {
			t.Errorf("Expected nil for non-existent plan, got %v", plan)
		}
	})
}

// TestSubscriptionRepository_GetSubscriptionByTenant tests fetching subscription by tenant ID
func TestSubscriptionRepository_GetSubscriptionByTenant(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSubscriptionRepository(db)

	t.Run("returns subscription for tenant", func(t *testing.T) {
		// Test tenant 0 is created by SetupTestDB (default tenant for Simple-Mode)
		sub, err := repo.GetSubscriptionByTenant(0)
		if err != nil {
			t.Fatalf("GetSubscriptionByTenant() error: %v", err)
		}
		if sub == nil {
			t.Fatal("Expected subscription, got nil")
		}
		if sub.TenantID != 0 {
			t.Errorf("Subscription TenantID = %d, want 0", sub.TenantID)
		}
		if sub.Status != models.SubscriptionStatusActive {
			t.Errorf("Subscription Status = %s, want 'active'", sub.Status)
		}
	})

	t.Run("returns nil for non-existent tenant", func(t *testing.T) {
		sub, err := repo.GetSubscriptionByTenant(99999)
		if err != nil {
			t.Fatalf("GetSubscriptionByTenant() error: %v", err)
		}
		if sub != nil {
			t.Errorf("Expected nil for non-existent tenant, got %v", sub)
		}
	})
}

// TestSubscriptionRepository_CreateSubscription tests creating a new subscription
func TestSubscriptionRepository_CreateSubscription(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSubscriptionRepository(db)

	// Create a second tenant for testing
	now := testutil.Now()
	_, err := db.Exec(`
		INSERT INTO tenants (id, slug, name, status, contact_email, created_at, updated_at)
		VALUES (2, 'tenant-2', 'Tenant 2', 'active', 'tenant2@example.com', ?, ?)
	`, now, now)
	if err != nil {
		t.Fatalf("Failed to create test tenant: %v", err)
	}

	t.Run("creates subscription successfully", func(t *testing.T) {
		sub := &models.TenantSubscription{
			TenantID:     2,
			PlanID:       2, // Pro plan
			Status:       models.SubscriptionStatusActive,
			BillingCycle: models.BillingCycleMonthly,
		}

		err := repo.CreateSubscription(sub)
		if err != nil {
			t.Fatalf("CreateSubscription() error: %v", err)
		}

		if sub.ID == 0 {
			t.Error("Expected subscription ID to be set")
		}

		// Verify in database
		retrieved, _ := repo.GetSubscriptionByTenant(2)
		if retrieved == nil {
			t.Fatal("Failed to retrieve created subscription")
		}
		if retrieved.PlanID != 2 {
			t.Errorf("Retrieved PlanID = %d, want 2", retrieved.PlanID)
		}
	})
}

// TestSubscriptionRepository_UpdateSubscription tests updating a subscription
func TestSubscriptionRepository_UpdateSubscription(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSubscriptionRepository(db)

	t.Run("updates subscription status", func(t *testing.T) {
		// Get existing subscription for tenant 0 (default tenant)
		sub, _ := repo.GetSubscriptionByTenant(0)
		if sub == nil {
			t.Fatal("No subscription found for tenant 0")
		}

		// Update to Pro plan
		sub.PlanID = 2
		sub.BillingCycle = models.BillingCycleYearly
		sub.Status = models.SubscriptionStatusActive

		err := repo.UpdateSubscription(sub)
		if err != nil {
			t.Fatalf("UpdateSubscription() error: %v", err)
		}

		// Verify update
		updated, _ := repo.GetSubscriptionByTenant(0)
		if updated.PlanID != 2 {
			t.Errorf("Updated PlanID = %d, want 2", updated.PlanID)
		}
		if updated.BillingCycle != models.BillingCycleYearly {
			t.Errorf("Updated BillingCycle = %s, want 'yearly'", updated.BillingCycle)
		}
	})
}

// TestSubscriptionRepository_GetTenantDogLimit tests getting dog limit for a tenant
func TestSubscriptionRepository_GetTenantDogLimit(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSubscriptionRepository(db)

	t.Run("returns 10 for free tenant", func(t *testing.T) {
		limit, err := repo.GetTenantDogLimit(0)
		if err != nil {
			t.Fatalf("GetTenantDogLimit() error: %v", err)
		}
		if limit != 10 {
			t.Errorf("Dog limit = %d, want 10 for free tier", limit)
		}
	})

	t.Run("returns -1 for pro tenant", func(t *testing.T) {
		// Update tenant 0 to Pro
		sub, err := repo.GetSubscriptionByTenant(0)
		if err != nil {
			t.Fatalf("GetSubscriptionByTenant() error: %v", err)
		}
		if sub == nil {
			t.Fatal("Expected subscription, got nil")
		}
		sub.PlanID = 2
		err = repo.UpdateSubscription(sub)
		if err != nil {
			t.Fatalf("UpdateSubscription() error: %v", err)
		}

		limit, err := repo.GetTenantDogLimit(0)
		if err != nil {
			t.Fatalf("GetTenantDogLimit() error: %v", err)
		}
		if limit != -1 {
			t.Errorf("Dog limit = %d, want -1 (unlimited) for pro tier", limit)
		}
	})

	t.Run("returns default 10 for non-existent tenant", func(t *testing.T) {
		limit, err := repo.GetTenantDogLimit(99999)
		if err != nil {
			t.Fatalf("GetTenantDogLimit() error: %v", err)
		}
		if limit != 10 {
			t.Errorf("Dog limit = %d, want 10 (default) for non-existent tenant", limit)
		}
	})
}

// TestSubscriptionRepository_CancelSubscription_ResetsToPlanFree tests that cancellation resets plan to Free (TDD RED Phase)
// BUG #3: When subscription is cancelled, plan_id should be reset to Free (1)
func TestSubscriptionRepository_CancelSubscription_ResetsToPlanFree(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewSubscriptionRepository(db)

	// First, delete any existing subscription for tenant 0 (default tenant)
	_, _ = db.Exec(`DELETE FROM tenant_subscriptions WHERE tenant_id = 0`)

	// Create a Pro subscription for tenant 0 (default tenant from SetupTestDB)
	_, err := db.Exec(`
		INSERT INTO tenant_subscriptions (tenant_id, plan_id, status, created_at, updated_at)
		VALUES (0, 2, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`)
	if err != nil {
		t.Fatalf("Failed to create Pro subscription: %v", err)
	}

	// Verify it's Pro
	sub, err := repo.GetSubscriptionByTenant(0)
	if err != nil {
		t.Fatalf("GetSubscriptionByTenant() error: %v", err)
	}
	if sub == nil {
		t.Fatal("Expected subscription to exist")
	}
	if sub.PlanID != 2 {
		t.Errorf("Expected PlanID = 2 (Pro), got %d", sub.PlanID)
	}

	// Cancel subscription
	err = repo.CancelSubscription(0, "test cancellation")
	if err != nil {
		t.Fatalf("CancelSubscription() error: %v", err)
	}

	// Verify plan_id is reset to Free (1)
	sub, err = repo.GetSubscriptionByTenant(0)
	if err != nil {
		t.Fatalf("GetSubscriptionByTenant() after cancel error: %v", err)
	}
	if sub == nil {
		t.Fatal("Expected subscription to still exist after cancellation")
	}
	if sub.Status != models.SubscriptionStatusCancelled {
		t.Errorf("Expected status = 'cancelled', got %s", sub.Status)
	}
	if sub.PlanID != 1 {
		t.Errorf("BUG #3: Expected plan_id = 1 (Free) after cancellation, got %d", sub.PlanID)
	}
}
