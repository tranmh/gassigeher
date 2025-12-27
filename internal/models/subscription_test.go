package models

import (
	"testing"
	"time"
)

// TestPricingPlan_IsUnlimited tests if a plan has unlimited dogs (TDD RED Phase)
func TestPricingPlan_IsUnlimited(t *testing.T) {
	tests := []struct {
		name     string
		maxDogs  int
		expected bool
	}{
		{
			name:     "Free plan with 10 dogs is limited",
			maxDogs:  10,
			expected: false,
		},
		{
			name:     "Pro plan with -1 dogs is unlimited",
			maxDogs:  -1,
			expected: true,
		},
		{
			name:     "Plan with 0 dogs is limited",
			maxDogs:  0,
			expected: false,
		},
		{
			name:     "Plan with 100 dogs is limited",
			maxDogs:  100,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := &PricingPlan{
				MaxDogs: tt.maxDogs,
			}
			if plan.IsUnlimited() != tt.expected {
				t.Errorf("IsUnlimited() = %v, want %v", plan.IsUnlimited(), tt.expected)
			}
		})
	}
}

// TestPricingPlan_GetYearlySavings tests yearly savings calculation
func TestPricingPlan_GetYearlySavings(t *testing.T) {
	tests := []struct {
		name          string
		priceMonthly  int // in cents
		priceYearly   int // in cents
		expectedSaved int // in cents
	}{
		{
			name:          "Free plan has no savings",
			priceMonthly:  0,
			priceYearly:   0,
			expectedSaved: 0,
		},
		{
			name:          "Pro plan saves 2 months (€58)",
			priceMonthly:  2900, // €29/month
			priceYearly:   29000, // €290/year
			expectedSaved: 5800, // 12*29 - 290 = 348 - 290 = €58
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := &PricingPlan{
				PriceMonthly: tt.priceMonthly,
				PriceYearly:  tt.priceYearly,
			}
			if saved := plan.GetYearlySavings(); saved != tt.expectedSaved {
				t.Errorf("GetYearlySavings() = %d, want %d", saved, tt.expectedSaved)
			}
		})
	}
}

// TestTenantSubscription_IsActive tests subscription active status
func TestTenantSubscription_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{
			name:     "Active subscription",
			status:   SubscriptionStatusActive,
			expected: true,
		},
		{
			name:     "Cancelled subscription",
			status:   SubscriptionStatusCancelled,
			expected: false,
		},
		{
			name:     "Past due subscription",
			status:   SubscriptionStatusPastDue,
			expected: false,
		},
		{
			name:     "Trialing subscription",
			status:   SubscriptionStatusTrialing,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := &TenantSubscription{
				Status: tt.status,
			}
			if sub.IsActive() != tt.expected {
				t.Errorf("IsActive() = %v, want %v", sub.IsActive(), tt.expected)
			}
		})
	}
}

// TestTenantSubscription_IsExpired tests if subscription period has ended
func TestTenantSubscription_IsExpired(t *testing.T) {
	now := time.Now()
	past := now.Add(-24 * time.Hour)
	future := now.Add(24 * time.Hour)

	tests := []struct {
		name             string
		currentPeriodEnd *time.Time
		expected         bool
	}{
		{
			name:             "Period ended yesterday is expired",
			currentPeriodEnd: &past,
			expected:         true,
		},
		{
			name:             "Period ends tomorrow is not expired",
			currentPeriodEnd: &future,
			expected:         false,
		},
		{
			name:             "No period end (free) is not expired",
			currentPeriodEnd: nil,
			expected:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := &TenantSubscription{
				CurrentPeriodEnd: tt.currentPeriodEnd,
			}
			if sub.IsExpired() != tt.expected {
				t.Errorf("IsExpired() = %v, want %v", sub.IsExpired(), tt.expected)
			}
		})
	}
}

// TestTenantSubscription_GetDogLimit tests getting the dog limit from subscription
func TestTenantSubscription_GetDogLimit(t *testing.T) {
	freePlan := &PricingPlan{ID: 1, Slug: "free", MaxDogs: 10}
	proPlan := &PricingPlan{ID: 2, Slug: "pro", MaxDogs: -1}

	tests := []struct {
		name     string
		plan     *PricingPlan
		expected int
	}{
		{
			name:     "Free plan returns 10",
			plan:     freePlan,
			expected: 10,
		},
		{
			name:     "Pro plan returns -1 (unlimited)",
			plan:     proPlan,
			expected: -1,
		},
		{
			name:     "No plan returns default 10",
			plan:     nil,
			expected: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := &TenantSubscription{
				Plan: tt.plan,
			}
			if limit := sub.GetDogLimit(); limit != tt.expected {
				t.Errorf("GetDogLimit() = %d, want %d", limit, tt.expected)
			}
		})
	}
}

// TestSubscriptionRequest_Validate tests subscription request validation
func TestSubscriptionRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateSubscriptionRequest
		wantErr bool
	}{
		{
			name: "Valid monthly subscription",
			req: CreateSubscriptionRequest{
				PlanSlug:     "pro",
				BillingCycle: BillingCycleMonthly,
			},
			wantErr: false,
		},
		{
			name: "Valid yearly subscription",
			req: CreateSubscriptionRequest{
				PlanSlug:     "pro",
				BillingCycle: BillingCycleYearly,
			},
			wantErr: false,
		},
		{
			name: "Missing plan slug",
			req: CreateSubscriptionRequest{
				BillingCycle: BillingCycleMonthly,
			},
			wantErr: true,
		},
		{
			name: "Invalid billing cycle",
			req: CreateSubscriptionRequest{
				PlanSlug:     "pro",
				BillingCycle: "weekly",
			},
			wantErr: true,
		},
		{
			name: "Empty billing cycle defaults to monthly",
			req: CreateSubscriptionRequest{
				PlanSlug: "pro",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestGetDefaultFreePlan tests the GetDefaultFreePlan function
func TestGetDefaultFreePlan(t *testing.T) {
	plan := GetDefaultFreePlan()

	if plan == nil {
		t.Fatal("GetDefaultFreePlan() returned nil")
	}

	if plan.Slug != "free" {
		t.Errorf("Slug = %s, want free", plan.Slug)
	}

	if plan.Name != "Free" {
		t.Errorf("Name = %s, want Free", plan.Name)
	}

	if plan.PriceMonthly != 0 {
		t.Errorf("PriceMonthly = %d, want 0", plan.PriceMonthly)
	}

	if plan.PriceYearly != 0 {
		t.Errorf("PriceYearly = %d, want 0", plan.PriceYearly)
	}

	if plan.MaxDogs != 10 {
		t.Errorf("MaxDogs = %d, want 10", plan.MaxDogs)
	}

	if plan.IsUnlimited() {
		t.Error("Free plan should not be unlimited")
	}
}

// TestGetDefaultProPlan tests the GetDefaultProPlan function
func TestGetDefaultProPlan(t *testing.T) {
	plan := GetDefaultProPlan()

	if plan == nil {
		t.Fatal("GetDefaultProPlan() returned nil")
	}

	if plan.Slug != "pro" {
		t.Errorf("Slug = %s, want pro", plan.Slug)
	}

	if plan.Name != "Pro" {
		t.Errorf("Name = %s, want Pro", plan.Name)
	}

	// Pro plan should have prices set
	if plan.PriceMonthly == 0 {
		t.Error("PriceMonthly should not be 0 for pro plan")
	}

	if plan.PriceYearly == 0 {
		t.Error("PriceYearly should not be 0 for pro plan")
	}

	// Pro plan should have unlimited dogs
	if plan.MaxDogs != -1 {
		t.Errorf("MaxDogs = %d, want -1 (unlimited)", plan.MaxDogs)
	}

	if !plan.IsUnlimited() {
		t.Error("Pro plan should be unlimited")
	}

	// Yearly should be cheaper than 12x monthly
	yearlySavings := plan.GetYearlySavings()
	if yearlySavings <= 0 {
		t.Errorf("GetYearlySavings() = %d, want > 0", yearlySavings)
	}
}
