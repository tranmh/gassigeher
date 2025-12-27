package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
)

// SubscriptionRepository handles subscription database operations
type SubscriptionRepository struct {
	db *sql.DB
}

// NewSubscriptionRepository creates a new subscription repository
func NewSubscriptionRepository(db *sql.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

// GetAllPlans returns all active pricing plans
func (r *SubscriptionRepository) GetAllPlans() ([]*models.PricingPlan, error) {
	query := `
		SELECT id, name, slug, max_dogs, price_monthly, price_yearly, is_active, created_at
		FROM pricing_plans
		WHERE is_active = 1
		ORDER BY price_monthly ASC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query pricing plans: %w", err)
	}
	defer rows.Close()

	plans := []*models.PricingPlan{}
	for rows.Next() {
		plan := &models.PricingPlan{}
		err := rows.Scan(
			&plan.ID,
			&plan.Name,
			&plan.Slug,
			&plan.MaxDogs,
			&plan.PriceMonthly,
			&plan.PriceYearly,
			&plan.IsActive,
			&plan.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pricing plan: %w", err)
		}
		plans = append(plans, plan)
	}

	return plans, nil
}

// GetPlanBySlug returns a pricing plan by its slug
func (r *SubscriptionRepository) GetPlanBySlug(slug string) (*models.PricingPlan, error) {
	query := `
		SELECT id, name, slug, max_dogs, price_monthly, price_yearly, is_active, created_at
		FROM pricing_plans
		WHERE slug = ? AND is_active = 1
	`

	plan := &models.PricingPlan{}
	err := r.db.QueryRow(query, slug).Scan(
		&plan.ID,
		&plan.Name,
		&plan.Slug,
		&plan.MaxDogs,
		&plan.PriceMonthly,
		&plan.PriceYearly,
		&plan.IsActive,
		&plan.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing plan: %w", err)
	}

	return plan, nil
}

// GetPlanByID returns a pricing plan by its ID
func (r *SubscriptionRepository) GetPlanByID(id int) (*models.PricingPlan, error) {
	query := `
		SELECT id, name, slug, max_dogs, price_monthly, price_yearly, is_active, created_at
		FROM pricing_plans
		WHERE id = ?
	`

	plan := &models.PricingPlan{}
	err := r.db.QueryRow(query, id).Scan(
		&plan.ID,
		&plan.Name,
		&plan.Slug,
		&plan.MaxDogs,
		&plan.PriceMonthly,
		&plan.PriceYearly,
		&plan.IsActive,
		&plan.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing plan: %w", err)
	}

	return plan, nil
}

// GetSubscriptionByTenant returns the subscription for a tenant
func (r *SubscriptionRepository) GetSubscriptionByTenant(tenantID int) (*models.TenantSubscription, error) {
	query := `
		SELECT ts.id, ts.tenant_id, ts.plan_id, ts.status, ts.billing_cycle,
		       ts.current_period_start, ts.current_period_end,
		       ts.stripe_customer_id, ts.stripe_subscription_id,
		       ts.cancelled_at, ts.created_at, ts.updated_at,
		       pp.id, pp.name, pp.slug, pp.max_dogs, pp.price_monthly, pp.price_yearly, pp.is_active, pp.created_at
		FROM tenant_subscriptions ts
		JOIN pricing_plans pp ON ts.plan_id = pp.id
		WHERE ts.tenant_id = ?
	`

	sub := &models.TenantSubscription{
		Plan: &models.PricingPlan{},
	}
	var billingCycle, stripeCustomerID, stripeSubscriptionID sql.NullString
	var currentPeriodStart, currentPeriodEnd, cancelledAt sql.NullTime

	err := r.db.QueryRow(query, tenantID).Scan(
		&sub.ID,
		&sub.TenantID,
		&sub.PlanID,
		&sub.Status,
		&billingCycle,
		&currentPeriodStart,
		&currentPeriodEnd,
		&stripeCustomerID,
		&stripeSubscriptionID,
		&cancelledAt,
		&sub.CreatedAt,
		&sub.UpdatedAt,
		&sub.Plan.ID,
		&sub.Plan.Name,
		&sub.Plan.Slug,
		&sub.Plan.MaxDogs,
		&sub.Plan.PriceMonthly,
		&sub.Plan.PriceYearly,
		&sub.Plan.IsActive,
		&sub.Plan.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Set nullable fields
	if billingCycle.Valid {
		sub.BillingCycle = billingCycle.String
	}
	if stripeCustomerID.Valid {
		sub.StripeCustomerID = &stripeCustomerID.String
	}
	if stripeSubscriptionID.Valid {
		sub.StripeSubscriptionID = &stripeSubscriptionID.String
	}
	if currentPeriodStart.Valid {
		sub.CurrentPeriodStart = &currentPeriodStart.Time
	}
	if currentPeriodEnd.Valid {
		sub.CurrentPeriodEnd = &currentPeriodEnd.Time
	}
	if cancelledAt.Valid {
		sub.CancelledAt = &cancelledAt.Time
	}

	return sub, nil
}

// CreateSubscription creates a new subscription for a tenant
func (r *SubscriptionRepository) CreateSubscription(sub *models.TenantSubscription) error {
	query := `
		INSERT INTO tenant_subscriptions (
			tenant_id, plan_id, status, billing_cycle,
			current_period_start, current_period_end,
			stripe_customer_id, stripe_subscription_id,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.Exec(
		query,
		sub.TenantID,
		sub.PlanID,
		sub.Status,
		sub.BillingCycle,
		sub.CurrentPeriodStart,
		sub.CurrentPeriodEnd,
		sub.StripeCustomerID,
		sub.StripeSubscriptionID,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get subscription ID: %w", err)
	}

	sub.ID = int(id)
	sub.CreatedAt = now
	sub.UpdatedAt = now

	return nil
}

// UpdateSubscription updates an existing subscription
func (r *SubscriptionRepository) UpdateSubscription(sub *models.TenantSubscription) error {
	query := `
		UPDATE tenant_subscriptions SET
			plan_id = ?,
			status = ?,
			billing_cycle = ?,
			current_period_start = ?,
			current_period_end = ?,
			stripe_customer_id = ?,
			stripe_subscription_id = ?,
			cancelled_at = ?,
			updated_at = ?
		WHERE id = ?
	`

	now := time.Now()

	// Convert empty billing_cycle to NULL (for CHECK constraint)
	var billingCycle interface{}
	if sub.BillingCycle != "" {
		billingCycle = sub.BillingCycle
	}

	_, err := r.db.Exec(
		query,
		sub.PlanID,
		sub.Status,
		billingCycle,
		sub.CurrentPeriodStart,
		sub.CurrentPeriodEnd,
		sub.StripeCustomerID,
		sub.StripeSubscriptionID,
		sub.CancelledAt,
		now,
		sub.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	sub.UpdatedAt = now
	return nil
}

// GetTenantDogLimit returns the dog limit for a tenant based on their subscription
func (r *SubscriptionRepository) GetTenantDogLimit(tenantID int) (int, error) {
	query := `
		SELECT pp.max_dogs
		FROM tenant_subscriptions ts
		JOIN pricing_plans pp ON ts.plan_id = pp.id
		WHERE ts.tenant_id = ? AND ts.status = 'active'
	`

	var maxDogs int
	err := r.db.QueryRow(query, tenantID).Scan(&maxDogs)

	if err == sql.ErrNoRows {
		// No subscription found, return default free tier limit
		return models.DefaultFreeTierDogLimit, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get dog limit: %w", err)
	}

	return maxDogs, nil
}

// CancelSubscription marks a subscription as cancelled and resets to Free plan
func (r *SubscriptionRepository) CancelSubscription(tenantID int, reason string) error {
	query := `
		UPDATE tenant_subscriptions SET
			plan_id = ?,
			status = ?,
			cancelled_at = ?,
			updated_at = ?
		WHERE tenant_id = ?
	`

	now := time.Now()
	_, err := r.db.Exec(query, models.FreePlanID, models.SubscriptionStatusCancelled, now, now, tenantID)
	if err != nil {
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}

	return nil
}

// SetStripeIDs updates the Stripe customer and subscription IDs
func (r *SubscriptionRepository) SetStripeIDs(tenantID int, customerID, subscriptionID string) error {
	query := `
		UPDATE tenant_subscriptions SET
			stripe_customer_id = ?,
			stripe_subscription_id = ?,
			updated_at = ?
		WHERE tenant_id = ?
	`

	_, err := r.db.Exec(query, customerID, subscriptionID, time.Now(), tenantID)
	if err != nil {
		return fmt.Errorf("failed to set Stripe IDs: %w", err)
	}

	return nil
}

// GetSubscriptionByStripeID returns subscription by Stripe subscription ID
func (r *SubscriptionRepository) GetSubscriptionByStripeID(stripeSubscriptionID string) (*models.TenantSubscription, error) {
	query := `
		SELECT ts.id, ts.tenant_id, ts.plan_id, ts.status, ts.billing_cycle,
		       ts.current_period_start, ts.current_period_end,
		       ts.stripe_customer_id, ts.stripe_subscription_id,
		       ts.cancelled_at, ts.created_at, ts.updated_at
		FROM tenant_subscriptions ts
		WHERE ts.stripe_subscription_id = ?
	`

	sub := &models.TenantSubscription{}
	var billingCycle, stripeCustomerID, stripeSubID sql.NullString
	var currentPeriodStart, currentPeriodEnd, cancelledAt sql.NullTime

	err := r.db.QueryRow(query, stripeSubscriptionID).Scan(
		&sub.ID,
		&sub.TenantID,
		&sub.PlanID,
		&sub.Status,
		&billingCycle,
		&currentPeriodStart,
		&currentPeriodEnd,
		&stripeCustomerID,
		&stripeSubID,
		&cancelledAt,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription by Stripe ID: %w", err)
	}

	// Set nullable fields
	if billingCycle.Valid {
		sub.BillingCycle = billingCycle.String
	}
	if stripeCustomerID.Valid {
		sub.StripeCustomerID = &stripeCustomerID.String
	}
	if stripeSubID.Valid {
		sub.StripeSubscriptionID = &stripeSubID.String
	}
	if currentPeriodStart.Valid {
		sub.CurrentPeriodStart = &currentPeriodStart.Time
	}
	if currentPeriodEnd.Valid {
		sub.CurrentPeriodEnd = &currentPeriodEnd.Time
	}
	if cancelledAt.Valid {
		sub.CancelledAt = &cancelledAt.Time
	}

	return sub, nil
}
