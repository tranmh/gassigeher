package handlers

import (
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/stripe/stripe-go/v76"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
	"github.com/tranmh/gassigeher/internal/services"
)

// BillingHandler handles billing-related HTTP requests
type BillingHandler struct {
	db              *sql.DB
	stripeService   *services.StripeService
	subscriptionRepo *repository.SubscriptionRepository
	dogRepo         *repository.DogRepository
}

// NewBillingHandler creates a new billing handler
func NewBillingHandler(db *sql.DB, stripeService *services.StripeService) *BillingHandler {
	return &BillingHandler{
		db:              db,
		stripeService:   stripeService,
		subscriptionRepo: repository.NewSubscriptionRepository(db),
		dogRepo:         repository.NewDogRepository(db),
	}
}

// GetSubscription returns the current subscription for the tenant
// GET /api/billing/subscription
func (h *BillingHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok || tenantID == 0 {
		respondError(w, http.StatusUnauthorized, "Nicht autorisiert")
		return
	}

	subscription, err := h.subscriptionRepo.GetSubscriptionByTenant(tenantID)
	if err != nil {
		log.Printf("ERROR: Failed to get subscription for tenant %d: %v", tenantID, err)
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Subscription")
		return
	}

	// If no subscription exists, create a default free one
	if subscription == nil {
		subscription = &models.TenantSubscription{
			TenantID: tenantID,
			PlanID:   1, // Free plan
			Status:   models.SubscriptionStatusActive,
			Plan:     models.GetDefaultFreePlan(),
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"subscription": subscription,
		"plan":         subscription.Plan,
	})
}

// GetPlans returns all available pricing plans
// GET /api/billing/plans
func (h *BillingHandler) GetPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.subscriptionRepo.GetAllPlans()
	if err != nil {
		log.Printf("ERROR: Failed to get plans: %v", err)
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Pläne")
		return
	}

	// Add Stripe configured status
	stripeConfigured := h.stripeService != nil && h.stripeService.IsConfigured()
	publishableKey := ""
	if stripeConfigured {
		publishableKey = h.stripeService.GetPublishableKey()
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"plans":            plans,
		"stripe_configured": stripeConfigured,
		"publishable_key":   publishableKey,
	})
}

// GetUsage returns the current usage for the tenant
// GET /api/billing/usage
func (h *BillingHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok || tenantID == 0 {
		respondError(w, http.StatusUnauthorized, "Nicht autorisiert")
		return
	}

	// Get dog count
	dogsUsed, err := h.dogRepo.CountByTenant(tenantID)
	if err != nil {
		log.Printf("ERROR: Failed to count dogs for tenant %d: %v", tenantID, err)
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Nutzung")
		return
	}

	// Get dog limit from subscription
	dogsLimit, err := h.subscriptionRepo.GetTenantDogLimit(tenantID)
	if err != nil {
		log.Printf("ERROR: Failed to get dog limit for tenant %d: %v", tenantID, err)
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden des Limits")
		return
	}

	// Calculate over-limit status
	overLimit := false
	excessCount := 0
	if dogsLimit != -1 && dogsUsed > dogsLimit {
		overLimit = true
		excessCount = dogsUsed - dogsLimit
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"dogs_used":    dogsUsed,
		"dogs_limit":   dogsLimit,
		"over_limit":   overLimit,
		"excess_count": excessCount,
	})
}

// CreateCheckoutRequest represents a checkout session request
type CreateCheckoutRequest struct {
	PlanSlug     string `json:"plan_slug"`
	BillingCycle string `json:"billing_cycle"`
}

// CreateCheckout creates a Stripe checkout session
// POST /api/billing/checkout
func (h *BillingHandler) CreateCheckout(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok || tenantID == 0 {
		respondError(w, http.StatusUnauthorized, "Nicht autorisiert")
		return
	}

	// Security: Only admins can initiate checkout (financial decisions)
	isAdmin, _ := r.Context().Value(middleware.IsAdminKey).(bool)
	if !isAdmin {
		respondError(w, http.StatusForbidden, "Nur Administratoren können Zahlungen verwalten")
		return
	}

	// Parse request first for early validation
	var req CreateCheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Anfrage")
		return
	}

	// Default to pro plan and monthly billing
	if req.PlanSlug == "" {
		req.PlanSlug = "pro"
	}
	if req.BillingCycle == "" {
		req.BillingCycle = models.BillingCycleMonthly
	}

	// Validate billing_cycle - must be "monthly" or "yearly"
	if req.BillingCycle != models.BillingCycleMonthly && req.BillingCycle != models.BillingCycleYearly {
		respondError(w, http.StatusBadRequest, "Ungültiger Abrechnungszeitraum. Erlaubt: monthly, yearly")
		return
	}

	// Validate plan_slug - must be a known plan
	if req.PlanSlug != "free" && req.PlanSlug != "pro" {
		respondError(w, http.StatusBadRequest, "Ungültiger Plan. Erlaubt: free, pro")
		return
	}

	// Check if Stripe is configured (after validation, before external call)
	if h.stripeService == nil || !h.stripeService.IsConfigured() {
		respondError(w, http.StatusServiceUnavailable, "Zahlungssystem nicht konfiguriert")
		return
	}

	// Get user email from context (needed for Stripe)
	email, _ := r.Context().Value(middleware.EmailKey).(string)
	if email == "" {
		email = "customer@example.com" // Fallback - Stripe will ask for email
	}

	// Create checkout session
	session, err := h.stripeService.CreateCheckoutSession(tenantID, req.PlanSlug, req.BillingCycle, email)
	if err != nil {
		log.Printf("ERROR: Failed to create checkout session: %v", err)
		respondError(w, http.StatusInternalServerError, "Fehler beim Erstellen der Checkout-Session")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"session_id":  session.ID,
		"checkout_url": session.URL,
	})
}

// CreateBillingPortal creates a Stripe billing portal session
// POST /api/billing/portal
func (h *BillingHandler) CreateBillingPortal(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok || tenantID == 0 {
		respondError(w, http.StatusUnauthorized, "Nicht autorisiert")
		return
	}

	// Security: Only admins can access billing portal
	isAdmin, _ := r.Context().Value(middleware.IsAdminKey).(bool)
	if !isAdmin {
		respondError(w, http.StatusForbidden, "Nur Administratoren können Zahlungen verwalten")
		return
	}

	// Check if Stripe is configured
	if h.stripeService == nil || !h.stripeService.IsConfigured() {
		respondError(w, http.StatusServiceUnavailable, "Zahlungssystem nicht konfiguriert")
		return
	}

	// Get subscription to find Stripe customer ID
	subscription, err := h.subscriptionRepo.GetSubscriptionByTenant(tenantID)
	if err != nil {
		log.Printf("ERROR: Failed to get subscription for tenant %d: %v", tenantID, err)
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Subscription")
		return
	}

	if subscription == nil || subscription.StripeCustomerID == nil {
		respondError(w, http.StatusBadRequest, "Keine Stripe-Kundenverbindung vorhanden")
		return
	}

	// Create portal session
	session, err := h.stripeService.CreateBillingPortalSession(*subscription.StripeCustomerID)
	if err != nil {
		log.Printf("ERROR: Failed to create billing portal session: %v", err)
		respondError(w, http.StatusInternalServerError, "Fehler beim Erstellen der Portal-Session")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"portal_url": session.URL,
	})
}

// CancelSubscriptionRequest represents a cancel request
type CancelSubscriptionRequest struct {
	Reason string `json:"reason,omitempty"`
}

// CancelSubscription cancels the tenant's subscription
// POST /api/billing/cancel
func (h *BillingHandler) CancelSubscription(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok || tenantID == 0 {
		respondError(w, http.StatusUnauthorized, "Nicht autorisiert")
		return
	}

	// Security: Only admins can cancel subscriptions
	isAdmin, _ := r.Context().Value(middleware.IsAdminKey).(bool)
	if !isAdmin {
		respondError(w, http.StatusForbidden, "Nur Administratoren können Abonnements kündigen")
		return
	}

	// Parse request (optional reason)
	var req CancelSubscriptionRequest
	json.NewDecoder(r.Body).Decode(&req) // Ignore error - reason is optional

	// Get subscription
	subscription, err := h.subscriptionRepo.GetSubscriptionByTenant(tenantID)
	if err != nil {
		log.Printf("ERROR: Failed to get subscription for tenant %d: %v", tenantID, err)
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Subscription")
		return
	}

	if subscription == nil {
		respondError(w, http.StatusBadRequest, "Keine Subscription vorhanden")
		return
	}

	// Cancel in Stripe if configured and has Stripe subscription
	if h.stripeService != nil && h.stripeService.IsConfigured() && subscription.StripeSubscriptionID != nil {
		err := h.stripeService.CancelSubscription(*subscription.StripeSubscriptionID)
		if err != nil {
			log.Printf("ERROR: Failed to cancel Stripe subscription: %v", err)
			// Continue anyway - mark as cancelled locally
		}
	}

	// Mark as cancelled in database
	err = h.subscriptionRepo.CancelSubscription(tenantID, req.Reason)
	if err != nil {
		log.Printf("ERROR: Failed to cancel subscription in database: %v", err)
		respondError(w, http.StatusInternalServerError, "Fehler beim Kündigen der Subscription")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Subscription erfolgreich gekündigt",
		"status":  models.SubscriptionStatusCancelled,
	})
}

// MaxWebhookBodySize is the maximum allowed webhook payload size (64KB)
// Stripe webhooks are typically small JSON payloads
const MaxWebhookBodySize = 64 * 1024

// HandleWebhook handles Stripe webhook events
// POST /api/billing/webhook
func (h *BillingHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Security: Limit body size to prevent DoS attacks
	// Check Content-Length header first for early rejection
	if r.ContentLength > MaxWebhookBodySize {
		respondError(w, http.StatusRequestEntityTooLarge, "Request body too large")
		return
	}

	// Wrap body with size limit reader
	limitedReader := io.LimitReader(r.Body, MaxWebhookBodySize+1)
	payload, err := io.ReadAll(limitedReader)
	if err != nil {
		log.Printf("ERROR: Failed to read webhook payload: %v", err)
		respondError(w, http.StatusBadRequest, "Fehler beim Lesen des Requests")
		return
	}

	// Check if body exceeded limit (LimitReader reads up to limit+1)
	if int64(len(payload)) > MaxWebhookBodySize {
		respondError(w, http.StatusRequestEntityTooLarge, "Request body too large")
		return
	}

	if h.stripeService == nil {
		respondError(w, http.StatusServiceUnavailable, "Stripe nicht konfiguriert")
		return
	}

	// Get Stripe signature
	signature := r.Header.Get("Stripe-Signature")
	if signature == "" {
		respondError(w, http.StatusBadRequest, "Stripe-Signatur fehlt")
		return
	}

	// Verify webhook signature
	event, err := h.stripeService.VerifyWebhookSignature(payload, signature)
	if err != nil {
		log.Printf("ERROR: Failed to verify webhook signature: %v", err)
		respondError(w, http.StatusBadRequest, "Ungültige Signatur")
		return
	}

	// Handle different event types
	switch event.Type {
	case services.WebhookEventCheckoutCompleted:
		h.handleCheckoutCompleted(event)

	case services.WebhookEventInvoicePaid:
		h.handleInvoicePaid(event)

	case services.WebhookEventInvoicePaymentFailed:
		h.handleInvoicePaymentFailed(event)

	case services.WebhookEventSubscriptionUpdated:
		h.handleSubscriptionUpdated(event)

	case services.WebhookEventSubscriptionDeleted:
		h.handleSubscriptionDeleted(event)

	default:
		log.Printf("Unhandled webhook event type: %s", event.Type)
	}

	// Always return 200 to acknowledge receipt
	w.WriteHeader(http.StatusOK)
}

// handleCheckoutCompleted processes checkout.session.completed events
func (h *BillingHandler) handleCheckoutCompleted(event *stripe.Event) {
	data, err := h.stripeService.ParseCheckoutSessionEvent(event)
	if err != nil {
		log.Printf("ERROR: Failed to parse checkout session event: %v", err)
		return
	}

	if data.TenantID == 0 {
		log.Printf("ERROR: No tenant_id in checkout session metadata")
		return
	}

	// Verify tenant exists before updating (security: prevent orphan subscriptions)
	existingSubscription, err := h.subscriptionRepo.GetSubscriptionByTenant(data.TenantID)
	if err != nil || existingSubscription == nil {
		log.Printf("ERROR: Tenant %d not found or has no subscription - ignoring checkout event", data.TenantID)
		return
	}

	// Update subscription with Stripe IDs
	err = h.subscriptionRepo.SetStripeIDs(data.TenantID, data.CustomerID, data.SubscriptionID)
	if err != nil {
		log.Printf("ERROR: Failed to set Stripe IDs for tenant %d: %v", data.TenantID, err)
		return
	}

	// Get and update subscription to Pro plan
	subscription, err := h.subscriptionRepo.GetSubscriptionByTenant(data.TenantID)
	if err != nil {
		log.Printf("ERROR: Failed to get subscription for tenant %d: %v", data.TenantID, err)
		return
	}

	if subscription != nil {
		subscription.PlanID = 2 // Pro plan
		subscription.Status = models.SubscriptionStatusActive
		err = h.subscriptionRepo.UpdateSubscription(subscription)
		if err != nil {
			log.Printf("ERROR: Failed to update subscription for tenant %d: %v", data.TenantID, err)
		}
	}

	log.Printf("Checkout completed for tenant %d, customer %s", data.TenantID, data.CustomerID)
}

// handleInvoicePaid processes invoice.paid events
func (h *BillingHandler) handleInvoicePaid(event *stripe.Event) {
	data, err := h.stripeService.ParseSubscriptionEvent(event)
	if err != nil {
		log.Printf("ERROR: Failed to parse invoice event: %v", err)
		return
	}

	// Find subscription by Stripe subscription ID
	subscription, err := h.subscriptionRepo.GetSubscriptionByStripeID(data.SubscriptionID)
	if err != nil || subscription == nil {
		log.Printf("ERROR: Subscription not found for Stripe ID %s", data.SubscriptionID)
		return
	}

	// Update period dates
	if data.CurrentPeriodStart > 0 {
		start := time.Unix(data.CurrentPeriodStart, 0)
		subscription.CurrentPeriodStart = &start
	}
	if data.CurrentPeriodEnd > 0 {
		end := time.Unix(data.CurrentPeriodEnd, 0)
		subscription.CurrentPeriodEnd = &end
	}

	subscription.Status = models.SubscriptionStatusActive
	if err := h.subscriptionRepo.UpdateSubscription(subscription); err != nil {
		log.Printf("ERROR: Failed to update subscription %s after invoice paid: %v", data.SubscriptionID, err)
	}

	log.Printf("Invoice paid for subscription %s", data.SubscriptionID)
}

// handleInvoicePaymentFailed processes invoice.payment_failed events
func (h *BillingHandler) handleInvoicePaymentFailed(event *stripe.Event) {
	data, err := h.stripeService.ParseSubscriptionEvent(event)
	if err != nil {
		log.Printf("ERROR: Failed to parse payment failed event: %v", err)
		return
	}

	// Find subscription
	subscription, err := h.subscriptionRepo.GetSubscriptionByStripeID(data.SubscriptionID)
	if err != nil || subscription == nil {
		log.Printf("ERROR: Subscription not found for Stripe ID %s", data.SubscriptionID)
		return
	}

	// Mark as past_due
	subscription.Status = models.SubscriptionStatusPastDue
	if err := h.subscriptionRepo.UpdateSubscription(subscription); err != nil {
		log.Printf("ERROR: Failed to mark subscription %s as past_due: %v", data.SubscriptionID, err)
	}

	log.Printf("Payment failed for subscription %s", data.SubscriptionID)
}

// handleSubscriptionUpdated processes customer.subscription.updated events
func (h *BillingHandler) handleSubscriptionUpdated(event *stripe.Event) {
	data, err := h.stripeService.ParseSubscriptionEvent(event)
	if err != nil {
		log.Printf("ERROR: Failed to parse subscription updated event: %v", err)
		return
	}

	// Find subscription
	subscription, err := h.subscriptionRepo.GetSubscriptionByStripeID(data.SubscriptionID)
	if err != nil || subscription == nil {
		log.Printf("ERROR: Subscription not found for Stripe ID %s", data.SubscriptionID)
		return
	}

	// Update status based on Stripe status
	switch data.Status {
	case "active":
		subscription.Status = models.SubscriptionStatusActive
	case "past_due":
		subscription.Status = models.SubscriptionStatusPastDue
	case "canceled":
		subscription.Status = models.SubscriptionStatusCancelled
	case "trialing":
		subscription.Status = models.SubscriptionStatusTrialing
	}

	if err := h.subscriptionRepo.UpdateSubscription(subscription); err != nil {
		log.Printf("ERROR: Failed to update subscription %s status: %v", data.SubscriptionID, err)
	}

	log.Printf("Subscription %s updated to status %s", data.SubscriptionID, data.Status)
}

// handleSubscriptionDeleted processes customer.subscription.deleted events
func (h *BillingHandler) handleSubscriptionDeleted(event *stripe.Event) {
	data, err := h.stripeService.ParseSubscriptionEvent(event)
	if err != nil {
		log.Printf("ERROR: Failed to parse subscription deleted event: %v", err)
		return
	}

	// Find subscription
	subscription, err := h.subscriptionRepo.GetSubscriptionByStripeID(data.SubscriptionID)
	if err != nil || subscription == nil {
		log.Printf("ERROR: Subscription not found for Stripe ID %s", data.SubscriptionID)
		return
	}

	// Mark as cancelled and downgrade to free
	subscription.Status = models.SubscriptionStatusCancelled
	subscription.PlanID = 1 // Free plan
	if err := h.subscriptionRepo.UpdateSubscription(subscription); err != nil {
		log.Printf("ERROR: Failed to downgrade subscription %s to free: %v", data.SubscriptionID, err)
	}

	log.Printf("Subscription %s deleted, downgraded to free", data.SubscriptionID)
}
