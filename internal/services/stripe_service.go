package services

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/stripe/stripe-go/v76"
	portalsession "github.com/stripe/stripe-go/v76/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/subscription"
	"github.com/stripe/stripe-go/v76/webhook"
)

// StripeService handles Stripe payment operations
type StripeService struct {
	secretKey      string
	publishableKey string
	priceMonthly   string
	priceYearly    string
	baseURL        string
	webhookSecret  string
}

// NewStripeService creates a new Stripe service
func NewStripeService(secretKey, publishableKey, priceMonthly, priceYearly, baseURL string) *StripeService {
	if secretKey != "" {
		stripe.Key = secretKey
	}
	return &StripeService{
		secretKey:      secretKey,
		publishableKey: publishableKey,
		priceMonthly:   priceMonthly,
		priceYearly:    priceYearly,
		baseURL:        baseURL,
	}
}

// SetWebhookSecret sets the webhook secret for signature verification
func (s *StripeService) SetWebhookSecret(secret string) {
	s.webhookSecret = secret
}

// IsConfigured returns true if Stripe is properly configured
func (s *StripeService) IsConfigured() bool {
	return s.secretKey != "" && s.priceMonthly != "" && s.priceYearly != ""
}

// GetPriceID returns the Stripe price ID for the given billing cycle
func (s *StripeService) GetPriceID(billingCycle string) (string, error) {
	switch billingCycle {
	case "monthly":
		return s.priceMonthly, nil
	case "yearly":
		return s.priceYearly, nil
	default:
		return "", errors.New("invalid billing cycle: must be 'monthly' or 'yearly'")
	}
}

// GetPublishableKey returns the publishable key for frontend use
func (s *StripeService) GetPublishableKey() string {
	return s.publishableKey
}

// CreateCheckoutSession creates a Stripe checkout session for subscription
func (s *StripeService) CreateCheckoutSession(tenantID int, planSlug, billingCycle, customerEmail string) (*stripe.CheckoutSession, error) {
	if s.secretKey == "" {
		return nil, errors.New("stripe API key not configured")
	}

	// Validate plan - only "pro" is valid for checkout (free doesn't need payment)
	if planSlug != "pro" {
		return nil, errors.New("invalid plan: only 'pro' plan requires payment")
	}

	// Get the price ID for the billing cycle
	priceID, err := s.GetPriceID(billingCycle)
	if err != nil {
		return nil, err
	}

	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(fmt.Sprintf("%s/billing?session_id={CHECKOUT_SESSION_ID}&success=true", s.baseURL)),
		CancelURL:  stripe.String(fmt.Sprintf("%s/billing?cancelled=true", s.baseURL)),
		CustomerEmail: stripe.String(customerEmail),
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: map[string]string{
				"tenant_id": fmt.Sprintf("%d", tenantID),
			},
		},
		Metadata: map[string]string{
			"tenant_id": fmt.Sprintf("%d", tenantID),
		},
	}

	result, err := checkoutsession.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create checkout session: %w", err)
	}

	return result, nil
}

// CreateBillingPortalSession creates a Stripe billing portal session
func (s *StripeService) CreateBillingPortalSession(customerID string) (*stripe.BillingPortalSession, error) {
	if s.secretKey == "" {
		return nil, errors.New("stripe API key not configured")
	}

	if customerID == "" {
		return nil, errors.New("customer ID is required")
	}

	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(fmt.Sprintf("%s/billing", s.baseURL)),
	}

	result, err := portalsession.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create billing portal session: %w", err)
	}

	return result, nil
}

// CancelSubscription cancels a Stripe subscription
func (s *StripeService) CancelSubscription(subscriptionID string) error {
	if s.secretKey == "" {
		return errors.New("stripe API key not configured")
	}

	if subscriptionID == "" {
		return errors.New("subscription ID is required")
	}

	params := &stripe.SubscriptionCancelParams{}
	_, err := subscription.Cancel(subscriptionID, params)
	if err != nil {
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}

	return nil
}

// VerifyWebhookSignature verifies the Stripe webhook signature and returns the event
func (s *StripeService) VerifyWebhookSignature(payload []byte, signature string) (*stripe.Event, error) {
	if s.webhookSecret == "" {
		return nil, errors.New("webhook secret not configured")
	}

	event, err := webhook.ConstructEvent(payload, signature, s.webhookSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to verify webhook signature: %w", err)
	}

	return &event, nil
}

// WebhookEvent types
const (
	WebhookEventCheckoutCompleted      = "checkout.session.completed"
	WebhookEventInvoicePaid            = "invoice.paid"
	WebhookEventInvoicePaymentFailed   = "invoice.payment_failed"
	WebhookEventSubscriptionUpdated    = "customer.subscription.updated"
	WebhookEventSubscriptionDeleted    = "customer.subscription.deleted"
)

// CheckoutSessionData represents data from a checkout.session.completed event
type CheckoutSessionData struct {
	CustomerID     string
	SubscriptionID string
	TenantID       int
	CustomerEmail  string
}

// ParseCheckoutSessionEvent extracts data from a checkout.session.completed event
func (s *StripeService) ParseCheckoutSessionEvent(event *stripe.Event) (*CheckoutSessionData, error) {
	if event.Type != WebhookEventCheckoutCompleted {
		return nil, errors.New("invalid event type for checkout session")
	}

	var session stripe.CheckoutSession
	err := json.Unmarshal(event.Data.Raw, &session)
	if err != nil {
		return nil, fmt.Errorf("failed to parse checkout session: %w", err)
	}

	data := &CheckoutSessionData{
		CustomerID:     session.Customer.ID,
		SubscriptionID: session.Subscription.ID,
		CustomerEmail:  session.CustomerEmail,
	}

	// Extract tenant_id from metadata
	if tenantIDStr, ok := session.Metadata["tenant_id"]; ok {
		var tenantID int
		fmt.Sscanf(tenantIDStr, "%d", &tenantID)
		data.TenantID = tenantID
	}

	return data, nil
}

// SubscriptionEventData represents data from subscription events
type SubscriptionEventData struct {
	SubscriptionID string
	CustomerID     string
	Status         string
	PlanID         string
	CurrentPeriodStart int64
	CurrentPeriodEnd   int64
}

// ParseSubscriptionEvent extracts data from a subscription event
func (s *StripeService) ParseSubscriptionEvent(event *stripe.Event) (*SubscriptionEventData, error) {
	var sub stripe.Subscription
	err := json.Unmarshal(event.Data.Raw, &sub)
	if err != nil {
		return nil, fmt.Errorf("failed to parse subscription event: %w", err)
	}

	data := &SubscriptionEventData{
		SubscriptionID:     sub.ID,
		CustomerID:         sub.Customer.ID,
		Status:             string(sub.Status),
		CurrentPeriodStart: sub.CurrentPeriodStart,
		CurrentPeriodEnd:   sub.CurrentPeriodEnd,
	}

	// Get plan ID from the first item
	if len(sub.Items.Data) > 0 {
		data.PlanID = sub.Items.Data[0].Price.ID
	}

	return data, nil
}
