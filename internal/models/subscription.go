package models

import (
	"errors"
	"strings"
	"time"
)

// Subscription status constants
const (
	SubscriptionStatusActive    = "active"
	SubscriptionStatusCancelled = "cancelled"
	SubscriptionStatusPastDue   = "past_due"
	SubscriptionStatusTrialing  = "trialing"
)

// Billing cycle constants
const (
	BillingCycleMonthly = "monthly"
	BillingCycleYearly  = "yearly"
)

// Default dog limit for free tier
const DefaultFreeTierDogLimit = 10

// FreePlanID is the ID of the free plan in the pricing_plans table
// This should match the ID seeded in database migrations
const FreePlanID = 1

// PricingPlan represents a subscription pricing plan
type PricingPlan struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`          // "Free", "Pro"
	Slug         string    `json:"slug"`          // "free", "pro"
	MaxDogs      int       `json:"max_dogs"`      // 10, -1 (unlimited)
	PriceMonthly int       `json:"price_monthly"` // cents: 0, 2900
	PriceYearly  int       `json:"price_yearly"`  // cents: 0, 29000
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
}

// TenantSubscription represents a tenant's subscription to a pricing plan
type TenantSubscription struct {
	ID                   int        `json:"id"`
	TenantID             int        `json:"tenant_id"`
	PlanID               int        `json:"plan_id"`
	Status               string     `json:"status"`        // active, cancelled, past_due, trialing
	BillingCycle         string     `json:"billing_cycle"` // monthly, yearly
	CurrentPeriodStart   *time.Time `json:"current_period_start,omitempty"`
	CurrentPeriodEnd     *time.Time `json:"current_period_end,omitempty"`
	StripeCustomerID     *string    `json:"stripe_customer_id,omitempty"`
	StripeSubscriptionID *string    `json:"stripe_subscription_id,omitempty"`
	CancelledAt          *time.Time `json:"cancelled_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`

	// Related data (loaded separately)
	Plan *PricingPlan `json:"plan,omitempty"`
}

// CreateSubscriptionRequest represents a request to create/upgrade a subscription
type CreateSubscriptionRequest struct {
	PlanSlug     string `json:"plan_slug"`
	BillingCycle string `json:"billing_cycle"` // monthly, yearly
}

// CancelSubscriptionRequest represents a request to cancel a subscription
type CancelSubscriptionRequest struct {
	Reason string `json:"reason,omitempty"`
}

// SubscriptionResponse represents subscription info returned to the frontend
type SubscriptionResponse struct {
	Subscription *TenantSubscription `json:"subscription"`
	Plan         *PricingPlan        `json:"plan"`
	Usage        *UsageInfo          `json:"usage"`
}

// UsageInfo represents current usage stats for a tenant
type UsageInfo struct {
	DogsUsed  int `json:"dogs_used"`
	DogsLimit int `json:"dogs_limit"` // -1 for unlimited
}

// IsUnlimited returns true if the plan has unlimited dogs
func (p *PricingPlan) IsUnlimited() bool {
	return p.MaxDogs == -1
}

// GetYearlySavings returns the amount saved per year in cents when choosing yearly billing
func (p *PricingPlan) GetYearlySavings() int {
	if p.PriceMonthly == 0 {
		return 0
	}
	yearlyFromMonthly := p.PriceMonthly * 12
	return yearlyFromMonthly - p.PriceYearly
}

// IsActive returns true if the subscription is currently active
func (s *TenantSubscription) IsActive() bool {
	return s.Status == SubscriptionStatusActive || s.Status == SubscriptionStatusTrialing
}

// IsExpired returns true if the subscription period has ended
func (s *TenantSubscription) IsExpired() bool {
	if s.CurrentPeriodEnd == nil {
		return false
	}
	return time.Now().After(*s.CurrentPeriodEnd)
}

// GetDogLimit returns the dog limit for this subscription
func (s *TenantSubscription) GetDogLimit() int {
	if s.Plan == nil {
		return DefaultFreeTierDogLimit
	}
	return s.Plan.MaxDogs
}

// Validate validates the CreateSubscriptionRequest
func (r *CreateSubscriptionRequest) Validate() error {
	if strings.TrimSpace(r.PlanSlug) == "" {
		return errors.New("plan slug is required")
	}

	// Default to monthly if not specified
	if r.BillingCycle == "" {
		r.BillingCycle = BillingCycleMonthly
	}

	// Validate billing cycle
	if r.BillingCycle != BillingCycleMonthly && r.BillingCycle != BillingCycleYearly {
		return errors.New("billing cycle must be 'monthly' or 'yearly'")
	}

	return nil
}

// GetDefaultFreePlan returns the default free pricing plan
func GetDefaultFreePlan() *PricingPlan {
	return &PricingPlan{
		ID:           1,
		Name:         "Free",
		Slug:         "free",
		MaxDogs:      DefaultFreeTierDogLimit,
		PriceMonthly: 0,
		PriceYearly:  0,
		IsActive:     true,
	}
}

// GetDefaultProPlan returns the default pro pricing plan
func GetDefaultProPlan() *PricingPlan {
	return &PricingPlan{
		ID:           2,
		Name:         "Pro",
		Slug:         "pro",
		MaxDogs:      -1, // Unlimited
		PriceMonthly: 2900, // €29.00
		PriceYearly:  29000, // €290.00
		IsActive:     true,
	}
}
