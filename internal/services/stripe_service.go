package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

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

// CheckoutOptions contains optional parameters for checkout session creation
type CheckoutOptions struct {
	TrialDays      int    // Number of trial days (for free months)
	PromoCode      string // Promo code applied
	ReferralCode   string // Referral code applied
	PromoCodeID    int    // Internal promo code ID
	ReferralCodeID int    // Internal referral code ID
}

// CreateCheckoutSession creates a Stripe checkout session for subscription
func (s *StripeService) CreateCheckoutSession(tenantID int, planSlug, billingCycle, customerEmail string) (*stripe.CheckoutSession, error) {
	return s.CreateCheckoutSessionWithOptions(tenantID, planSlug, billingCycle, customerEmail, nil)
}

// CreateCheckoutSessionWithOptions creates a Stripe checkout session with additional options
func (s *StripeService) CreateCheckoutSessionWithOptions(tenantID int, planSlug, billingCycle, customerEmail string, options *CheckoutOptions) (*stripe.CheckoutSession, error) {
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

	// Build metadata with codes
	metadata := map[string]string{
		"tenant_id": fmt.Sprintf("%d", tenantID),
	}
	if options != nil {
		if options.PromoCode != "" {
			metadata["promo_code"] = options.PromoCode
		}
		if options.ReferralCode != "" {
			metadata["referral_code"] = options.ReferralCode
		}
		if options.PromoCodeID > 0 {
			metadata["promo_code_id"] = fmt.Sprintf("%d", options.PromoCodeID)
		}
		if options.ReferralCodeID > 0 {
			metadata["referral_code_id"] = fmt.Sprintf("%d", options.ReferralCodeID)
		}
	}

	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL:    stripe.String(fmt.Sprintf("%s/billing?session_id={CHECKOUT_SESSION_ID}&success=true", s.baseURL)),
		CancelURL:     stripe.String(fmt.Sprintf("%s/billing?cancelled=true", s.baseURL)),
		CustomerEmail: stripe.String(customerEmail),
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: metadata,
		},
		Metadata: metadata,
	}

	// Add trial period if specified (for free months)
	if options != nil && options.TrialDays > 0 {
		params.SubscriptionData.TrialPeriodDays = stripe.Int64(int64(options.TrialDays))
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
	PromoCode      string
	ReferralCode   string
	PromoCodeID    int
	ReferralCodeID int
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
		CustomerEmail: session.CustomerEmail,
	}

	// Safely extract Customer ID (may be nil for guest checkout)
	if session.Customer != nil {
		data.CustomerID = session.Customer.ID
	}

	// Safely extract Subscription ID (may be nil for one-time payments)
	if session.Subscription != nil {
		data.SubscriptionID = session.Subscription.ID
	}

	// Extract tenant_id from metadata with proper error handling
	// BUG FIX: tenant_id is now REQUIRED - return error if missing to prevent TenantID=0
	tenantIDStr, ok := session.Metadata["tenant_id"]
	if !ok || tenantIDStr == "" {
		return nil, errors.New("tenant_id is required in metadata but was missing")
	}

	var tenantID int
	n, err := fmt.Sscanf(tenantIDStr, "%d", &tenantID)
	if err != nil || n != 1 {
		return nil, fmt.Errorf("invalid tenant_id in metadata: %s", tenantIDStr)
	}
	if tenantID <= 0 {
		return nil, fmt.Errorf("tenant_id must be positive, got: %d", tenantID)
	}
	data.TenantID = tenantID

	// Extract promo code from metadata
	if promoCode, ok := session.Metadata["promo_code"]; ok {
		data.PromoCode = promoCode
	}

	// Extract referral code from metadata
	if referralCode, ok := session.Metadata["referral_code"]; ok {
		data.ReferralCode = referralCode
	}

	// Extract promo code ID from metadata
	if promoCodeIDStr, ok := session.Metadata["promo_code_id"]; ok {
		var promoCodeID int
		if n, err := fmt.Sscanf(promoCodeIDStr, "%d", &promoCodeID); err != nil || n != 1 {
			log.Printf("WARNING: Failed to parse promo_code_id '%s': %v", promoCodeIDStr, err)
		} else {
			data.PromoCodeID = promoCodeID
		}
	}

	// Extract referral code ID from metadata
	if referralCodeIDStr, ok := session.Metadata["referral_code_id"]; ok {
		var referralCodeID int
		if n, err := fmt.Sscanf(referralCodeIDStr, "%d", &referralCodeID); err != nil || n != 1 {
			log.Printf("WARNING: Failed to parse referral_code_id '%s': %v", referralCodeIDStr, err)
		} else {
			data.ReferralCodeID = referralCodeID
		}
	}

	return data, nil
}

// SubscriptionEventData represents data from subscription events
type SubscriptionEventData struct {
	SubscriptionID     string
	CustomerID         string
	Status             string
	PlanID             string
	CurrentPeriodStart int64
	CurrentPeriodEnd   int64
}

// InvoiceEventData represents data from invoice events
type InvoiceEventData struct {
	InvoiceID          string
	SubscriptionID     string
	CustomerID         string
	AmountPaid         int64  // in cents
	Currency           string
	Status             string
	InvoicePDF         string // URL to invoice PDF
	PeriodStart        int64
	PeriodEnd          int64
	InvoiceNumber      string
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

// ParseInvoiceEvent extracts data from an invoice event
func (s *StripeService) ParseInvoiceEvent(event *stripe.Event) (*InvoiceEventData, error) {
	var inv stripe.Invoice
	err := json.Unmarshal(event.Data.Raw, &inv)
	if err != nil {
		return nil, fmt.Errorf("failed to parse invoice event: %w", err)
	}

	data := &InvoiceEventData{
		InvoiceID:     inv.ID,
		AmountPaid:    inv.AmountPaid,
		Currency:      string(inv.Currency),
		Status:        string(inv.Status),
		InvoicePDF:    inv.InvoicePDF,
		PeriodStart:   inv.PeriodStart,
		PeriodEnd:     inv.PeriodEnd,
		InvoiceNumber: inv.Number,
	}

	// Safely extract Subscription ID
	if inv.Subscription != nil {
		data.SubscriptionID = inv.Subscription.ID
	}

	// Safely extract Customer ID
	if inv.Customer != nil {
		data.CustomerID = inv.Customer.ID
	}

	return data, nil
}
