package handlers

import (
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/stripe/stripe-go/v76"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
	"github.com/tranmh/gassigeher/internal/services"
)

// BillingHandler handles billing-related HTTP requests
type BillingHandler struct {
	db               *sql.DB
	cfg              *config.Config
	stripeService    *services.StripeService
	s3Service        *services.S3Service // For S3 pre-signed URL generation
	subscriptionRepo *repository.SubscriptionRepository
	dogRepo          *repository.DogRepository
	invoiceRepo      *repository.InvoiceRepository
	promoCodeRepo    *repository.PromoCodeRepository
	marketingRepo    *repository.MarketingRepository
}

// NewBillingHandler creates a new billing handler
func NewBillingHandler(db *sql.DB, cfg *config.Config, stripeService *services.StripeService) *BillingHandler {
	// Initialize S3 service if configured
	var s3Service *services.S3Service
	if cfg != nil && cfg.UseS3 {
		s3Config := &services.S3Config{
			Endpoint:   cfg.S3Endpoint,
			AccessKey:  cfg.S3AccessKey,
			SecretKey:  cfg.S3SecretKey,
			BucketName: cfg.S3BucketName,
			Region:     cfg.S3Region,
			PublicURL:  cfg.S3PublicURL,
			UseSSL:     cfg.S3UseSSL,
		}
		var err error
		s3Service, err = services.NewS3Service(s3Config)
		if err != nil {
			log.Printf("Warning: Failed to initialize S3 service in BillingHandler: %v", err)
		}
	}

	return &BillingHandler{
		db:               db,
		cfg:              cfg,
		stripeService:    stripeService,
		s3Service:        s3Service,
		subscriptionRepo: repository.NewSubscriptionRepository(db),
		dogRepo:          repository.NewDogRepository(db),
		invoiceRepo:      repository.NewInvoiceRepository(db),
		promoCodeRepo:    repository.NewPromoCodeRepository(db),
		marketingRepo:    repository.NewMarketingRepository(db),
	}
}

// GetSubscription returns the current subscription for the tenant
// GET /api/billing/subscription
func (h *BillingHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok {
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

	// Build enhanced response with free months info
	response := map[string]interface{}{
		"subscription": subscription,
		"plan":         subscription.Plan,
	}

	// Add free months info if applicable
	if subscription.FreeMonthsRemaining > 0 || subscription.FreeMonthsGranted > 0 {
		response["free_months_remaining"] = subscription.FreeMonthsRemaining
		response["free_months_granted"] = subscription.FreeMonthsGranted
		response["free_months_source"] = subscription.GetFreeMonthsSourceLabel()
	}

	// Add trial info if applicable
	if subscription.IsInTrial() {
		response["trial_ends_at"] = subscription.TrialEndsAt
		response["days_until_trial_ends"] = subscription.GetDaysUntilTrialEnds()
	}

	respondJSON(w, http.StatusOK, response)
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

	// Add test mode status
	testModeEnabled := h.cfg != nil && h.cfg.IsBillingTestModeEnabled()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"plans":             plans,
		"stripe_configured": stripeConfigured,
		"publishable_key":   publishableKey,
		"test_mode":         testModeEnabled,
	})
}

// GetUsage returns the current usage for the tenant
// GET /api/billing/usage
func (h *BillingHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok {
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
	PromoCode    string `json:"promo_code,omitempty"`
	ReferralCode string `json:"referral_code,omitempty"`
}

// CreateCheckout creates a Stripe checkout session
// POST /api/billing/checkout
func (h *BillingHandler) CreateCheckout(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok {
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

	// Get user email from context (required for Stripe checkout)
	email, _ := r.Context().Value(middleware.EmailKey).(string)
	if email == "" {
		respondError(w, http.StatusBadRequest, "E-Mail-Adresse nicht verfügbar. Bitte melden Sie sich erneut an.")
		return
	}

	// Build checkout options
	options := &services.CheckoutOptions{}
	trialDays := 0

	// Validate and apply promo code if provided
	if req.PromoCode != "" {
		promoCode, err := h.promoCodeRepo.GetByCode(req.PromoCode)
		if err != nil {
			log.Printf("ERROR: Failed to look up promo code: %v", err)
			respondError(w, http.StatusInternalServerError, "Fehler beim Prüfen des Gutscheincodes")
			return
		}

		if promoCode == nil || !promoCode.IsValid() {
			respondError(w, http.StatusBadRequest, "Ungültiger oder abgelaufener Gutscheincode")
			return
		}

		// Check if promo code is valid for the requested plan
		if promoCode.ValidForPlans != "" {
			// ValidForPlans is stored as JSON array like '["pro"]' or '["free","pro"]'
			if !strings.Contains(promoCode.ValidForPlans, `"`+req.PlanSlug+`"`) {
				respondError(w, http.StatusBadRequest, "Dieser Gutscheincode ist für den gewählten Plan nicht gültig")
				return
			}
		}

		// Check if tenant has already used this promo code
		hasUsed, err := h.promoCodeRepo.HasTenantUsedCode(promoCode.ID, tenantID)
		if err != nil {
			log.Printf("ERROR: Failed to check promo code usage: %v", err)
			respondError(w, http.StatusInternalServerError, "Fehler beim Prüfen des Gutscheincodes")
			return
		}
		if hasUsed {
			respondError(w, http.StatusBadRequest, "Dieser Gutscheincode wurde bereits verwendet")
			return
		}

		options.PromoCode = req.PromoCode
		options.PromoCodeID = promoCode.ID

		// Apply free months as trial days
		if promoCode.DiscountType == models.DiscountTypeFreeMonths {
			trialDays += promoCode.DiscountValue * 30 // 30 days per month
		}
	}

	// Validate and apply referral code if provided
	if req.ReferralCode != "" {
		referralCode, err := h.marketingRepo.GetReferralCodeByCode(req.ReferralCode)
		if err != nil {
			log.Printf("ERROR: Failed to look up referral code: %v", err)
			respondError(w, http.StatusInternalServerError, "Fehler beim Prüfen des Empfehlungscodes")
			return
		}

		if referralCode == nil || !referralCode.IsActive {
			respondError(w, http.StatusBadRequest, "Ungültiger oder inaktiver Empfehlungscode")
			return
		}

		// Check if referral code has expired
		if referralCode.ExpiresAt != nil && referralCode.ExpiresAt.Before(time.Now()) {
			respondError(w, http.StatusBadRequest, "Empfehlungscode ist abgelaufen")
			return
		}

		// Check if max uses reached
		if referralCode.MaxUses != nil && referralCode.UsesCount >= *referralCode.MaxUses {
			respondError(w, http.StatusBadRequest, "Empfehlungscode hat die maximale Anzahl an Verwendungen erreicht")
			return
		}

		// Check if tenant has already used any referral code
		hasUsed, err := h.marketingRepo.HasTenantUsedReferral(tenantID)
		if err != nil {
			log.Printf("ERROR: Failed to check referral usage: %v", err)
			respondError(w, http.StatusInternalServerError, "Fehler beim Prüfen des Empfehlungscodes")
			return
		}
		if hasUsed {
			respondError(w, http.StatusBadRequest, "Sie haben bereits einen Empfehlungscode verwendet")
			return
		}

		options.ReferralCode = req.ReferralCode
		options.ReferralCodeID = referralCode.ID

		// Add referee free months as trial days
		trialDays += referralCode.DiscountMonthsReferee * 30
	}

	// Set trial days if any discounts apply
	if trialDays > 0 {
		options.TrialDays = trialDays
	}

	// Create checkout session with options
	session, err := h.stripeService.CreateCheckoutSessionWithOptions(tenantID, req.PlanSlug, req.BillingCycle, email, options)
	if err != nil {
		log.Printf("ERROR: Failed to create checkout session: %v", err)
		respondError(w, http.StatusInternalServerError, "Fehler beim Erstellen der Checkout-Session")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"session_id":   session.ID,
		"checkout_url": session.URL,
		"trial_days":   trialDays,
	})
}

// CreateBillingPortal creates a Stripe billing portal session
// POST /api/billing/portal
func (h *BillingHandler) CreateBillingPortal(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok {
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
	if !ok {
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

		// Track which codes were applied for combined source
		hasPromo := false
		hasReferral := false
		totalFreeMonths := 0

		// Apply promo code if present
		if data.PromoCodeID > 0 {
			promoCode, err := h.promoCodeRepo.GetByID(data.PromoCodeID)
			if err == nil && promoCode != nil {
				// Record usage
				if err := h.promoCodeRepo.RecordUse(promoCode.ID, data.TenantID); err != nil {
					log.Printf("ERROR: Failed to record promo code use: %v", err)
				}
				if err := h.promoCodeRepo.IncrementUsesCount(promoCode.ID); err != nil {
					log.Printf("ERROR: Failed to increment promo code uses: %v", err)
				}

				// Apply free months to subscription
				if promoCode.DiscountType == models.DiscountTypeFreeMonths {
					subscription.FreeMonthsRemaining += promoCode.DiscountValue
					subscription.FreeMonthsGranted += promoCode.DiscountValue
					subscription.AppliedPromoCodeID = &promoCode.ID
					hasPromo = true
					totalFreeMonths += promoCode.DiscountValue
				}

				log.Printf("Applied promo code %s for tenant %d", promoCode.Code, data.TenantID)
			}
		}

		// Apply referral code if present
		if data.ReferralCodeID > 0 {
			referralCode, err := h.marketingRepo.GetReferralCode(data.ReferralCodeID)
			if err == nil && referralCode != nil {
				// Record referral usage
				if err := h.marketingRepo.RecordReferralUse(referralCode.ID, data.TenantID); err != nil {
					log.Printf("ERROR: Failed to record referral use: %v", err)
				}
				if err := h.marketingRepo.IncrementReferralCodeUses(referralCode.ID); err != nil {
					log.Printf("ERROR: Failed to increment referral uses: %v", err)
				}

				// Apply free months to referee (new tenant)
				if referralCode.DiscountMonthsReferee > 0 {
					subscription.FreeMonthsRemaining += referralCode.DiscountMonthsReferee
					subscription.FreeMonthsGranted += referralCode.DiscountMonthsReferee
					subscription.AppliedReferralCodeID = &referralCode.ID
					hasReferral = true
					totalFreeMonths += referralCode.DiscountMonthsReferee
				}

				// Grant free months to referrer if they have a subscription
				if referralCode.ReferrerTenantID != nil && referralCode.DiscountMonthsReferrer > 0 {
					go h.grantFreeMonthsToReferrer(*referralCode.ReferrerTenantID, referralCode.DiscountMonthsReferrer)
				}

				log.Printf("Applied referral code %s for tenant %d", referralCode.Code, data.TenantID)
			}
		}

		// Set free months source based on which codes were applied
		if hasPromo && hasReferral {
			subscription.FreeMonthsSource = strPtr("promo+referral")
		} else if hasPromo {
			subscription.FreeMonthsSource = strPtr("promo")
		} else if hasReferral {
			subscription.FreeMonthsSource = strPtr("referral")
		}

		// Set trial end date based on total free months
		// Use days (months * 30) to match the trial period sent to Stripe
		if totalFreeMonths > 0 {
			trialDays := totalFreeMonths * 30
			trialEnd := time.Now().AddDate(0, 0, trialDays)
			subscription.TrialEndsAt = &trialEnd
		}

		err = h.subscriptionRepo.UpdateSubscription(subscription)
		if err != nil {
			log.Printf("ERROR: Failed to update subscription for tenant %d: %v", data.TenantID, err)
		}
	}

	log.Printf("Checkout completed for tenant %d, customer %s", data.TenantID, data.CustomerID)
}

// grantFreeMonthsToReferrer grants free months to the referrer's subscription
// Uses atomic increment to prevent race conditions
func (h *BillingHandler) grantFreeMonthsToReferrer(referrerTenantID int, months int) {
	err := h.subscriptionRepo.IncrementFreeMonths(referrerTenantID, months, "referral")
	if err != nil {
		log.Printf("ERROR: Failed to grant free months to referrer tenant %d: %v", referrerTenantID, err)
		return
	}

	log.Printf("Granted %d free months to referrer tenant %d", months, referrerTenantID)
}

// handleInvoicePaid processes invoice.paid events
func (h *BillingHandler) handleInvoicePaid(event *stripe.Event) {
	// Parse invoice data
	invoiceData, err := h.stripeService.ParseInvoiceEvent(event)
	if err != nil {
		log.Printf("ERROR: Failed to parse invoice event: %v", err)
		return
	}

	// Find subscription by Stripe subscription ID
	subscription, err := h.subscriptionRepo.GetSubscriptionByStripeID(invoiceData.SubscriptionID)
	if err != nil || subscription == nil {
		log.Printf("ERROR: Subscription not found for Stripe ID %s", invoiceData.SubscriptionID)
		return
	}

	// Update period dates on subscription
	if invoiceData.PeriodStart > 0 {
		start := time.Unix(invoiceData.PeriodStart, 0)
		subscription.CurrentPeriodStart = &start
	}
	if invoiceData.PeriodEnd > 0 {
		end := time.Unix(invoiceData.PeriodEnd, 0)
		subscription.CurrentPeriodEnd = &end
	}

	subscription.Status = models.SubscriptionStatusActive
	if err := h.subscriptionRepo.UpdateSubscription(subscription); err != nil {
		log.Printf("ERROR: Failed to update subscription %s after invoice paid: %v", invoiceData.SubscriptionID, err)
	}

	// Check if invoice already exists (prevent duplicates)
	existingInvoice, err := h.invoiceRepo.GetByStripeID(invoiceData.InvoiceID)
	if err != nil {
		log.Printf("ERROR: Failed to check existing invoice: %v", err)
	}
	if existingInvoice != nil {
		log.Printf("Invoice %s already exists, skipping creation", invoiceData.InvoiceID)
		return
	}

	// Create invoice record
	subscriptionID := subscription.ID
	invoiceNumber := invoiceData.InvoiceNumber
	if invoiceNumber == "" {
		invoiceNumber = h.invoiceRepo.GenerateNextInvoiceNumber(subscription.TenantID)
	}

	var periodStart, periodEnd *time.Time
	if invoiceData.PeriodStart > 0 {
		t := time.Unix(invoiceData.PeriodStart, 0)
		periodStart = &t
	}
	if invoiceData.PeriodEnd > 0 {
		t := time.Unix(invoiceData.PeriodEnd, 0)
		periodEnd = &t
	}

	now := time.Now()
	invoice := &models.TenantInvoice{
		TenantID:        subscription.TenantID,
		SubscriptionID:  &subscriptionID,
		StripeInvoiceID: &invoiceData.InvoiceID,
		InvoiceNumber:   invoiceNumber,
		Status:          models.InvoiceStatusPaid,
		AmountCents:     int(invoiceData.AmountPaid),
		Currency:        invoiceData.Currency,
		PeriodStart:     periodStart,
		PeriodEnd:       periodEnd,
		Description:     "Pro Abonnement",
		PaidAt:          &now,
	}

	// Set PDF URL if available
	if invoiceData.InvoicePDF != "" {
		invoice.PDFURL = &invoiceData.InvoicePDF
	}

	if err := h.invoiceRepo.Create(invoice); err != nil {
		log.Printf("ERROR: Failed to create invoice record: %v", err)
		return
	}

	log.Printf("Invoice %s created for tenant %d (amount: %d %s)", invoiceData.InvoiceID, subscription.TenantID, invoiceData.AmountPaid, invoiceData.Currency)
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

// TestUpgradeRequest represents a test upgrade request
type TestUpgradeRequest struct {
	PlanSlug string `json:"plan_slug"` // "pro" or "free"
}

// TestUpgrade allows upgrading/downgrading subscriptions without Stripe (TEST MODE ONLY)
// POST /api/billing/test-upgrade
// This endpoint is only available in local development or when BILLING_TEST_MODE=true
func (h *BillingHandler) TestUpgrade(w http.ResponseWriter, r *http.Request) {
	// Security: Only allow in test mode
	if h.cfg == nil || !h.cfg.IsBillingTestModeEnabled() {
		respondError(w, http.StatusForbidden, "Test-Modus nicht aktiviert. Setzen Sie BILLING_TEST_MODE=true oder verwenden Sie eine .local Domain.")
		return
	}

	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Nicht autorisiert")
		return
	}

	// Security: Only admins can upgrade
	isAdmin, _ := r.Context().Value(middleware.IsAdminKey).(bool)
	if !isAdmin {
		respondError(w, http.StatusForbidden, "Nur Administratoren können den Plan ändern")
		return
	}

	var req TestUpgradeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Anfrage")
		return
	}

	// Require explicit plan_slug - don't silently default to pro
	if req.PlanSlug == "" {
		respondError(w, http.StatusBadRequest, "Plan-Slug ist erforderlich. Erlaubt: free, pro")
		return
	}

	// Look up plan by slug from database instead of using hardcoded IDs
	plan, err := h.subscriptionRepo.GetPlanBySlug(req.PlanSlug)
	if err != nil {
		log.Printf("ERROR: Failed to look up plan '%s': %v", req.PlanSlug, err)
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden des Plans")
		return
	}
	if plan == nil {
		respondError(w, http.StatusBadRequest, "Ungültiger Plan. Erlaubt: free, pro")
		return
	}

	planID := plan.ID
	planName := plan.Name

	// Get or create subscription
	subscription, err := h.subscriptionRepo.GetSubscriptionByTenant(tenantID)
	if err != nil {
		log.Printf("ERROR: Failed to get subscription for tenant %d: %v", tenantID, err)
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Subscription")
		return
	}

	if subscription == nil {
		// Create new subscription
		subscription = &models.TenantSubscription{
			TenantID:     tenantID,
			PlanID:       planID,
			Status:       models.SubscriptionStatusActive,
			BillingCycle: models.BillingCycleMonthly,
		}
		err = h.subscriptionRepo.CreateSubscription(subscription)
		if err != nil {
			log.Printf("ERROR: Failed to create subscription for tenant %d: %v", tenantID, err)
			respondError(w, http.StatusInternalServerError, "Fehler beim Erstellen der Subscription")
			return
		}
	} else {
		// Update existing subscription
		subscription.PlanID = planID
		subscription.Status = models.SubscriptionStatusActive
		err = h.subscriptionRepo.UpdateSubscription(subscription)
		if err != nil {
			log.Printf("ERROR: Failed to update subscription for tenant %d: %v", tenantID, err)
			respondError(w, http.StatusInternalServerError, "Fehler beim Aktualisieren der Subscription")
			return
		}
	}

	log.Printf("TEST MODE: Tenant %d upgraded to %s plan", tenantID, planName)

	// Get updated subscription with plan details
	subscription, _ = h.subscriptionRepo.GetSubscriptionByTenant(tenantID)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "Plan erfolgreich geändert (Test-Modus)",
		"plan":         planName,
		"subscription": subscription,
		"test_mode":    true,
	})
}

// GetInvoices returns all invoices for the tenant
// GET /api/billing/invoices
func (h *BillingHandler) GetInvoices(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Nicht autorisiert")
		return
	}

	invoices, err := h.invoiceRepo.GetByTenant(tenantID)
	if err != nil {
		log.Printf("ERROR: Failed to get invoices for tenant %d: %v", tenantID, err)
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Rechnungen")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"invoices": invoices,
		"count":    len(invoices),
	})
}

// GetInvoice returns a single invoice by ID
// GET /api/billing/invoices/{id}
func (h *BillingHandler) GetInvoice(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Nicht autorisiert")
		return
	}

	// Get invoice ID from URL using gorilla/mux
	vars := mux.Vars(r)
	invoiceID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Rechnungs-ID")
		return
	}

	invoice, err := h.invoiceRepo.GetByID(invoiceID)
	if err != nil {
		log.Printf("ERROR: Failed to get invoice %d: %v", invoiceID, err)
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Rechnung")
		return
	}

	if invoice == nil {
		respondError(w, http.StatusNotFound, "Rechnung nicht gefunden")
		return
	}

	// Security: Verify invoice belongs to this tenant
	if invoice.TenantID != tenantID {
		respondError(w, http.StatusForbidden, "Zugriff verweigert")
		return
	}

	respondJSON(w, http.StatusOK, invoice)
}

// DownloadInvoicePDF downloads an invoice PDF
// GET /api/billing/invoices/{id}/pdf
func (h *BillingHandler) DownloadInvoicePDF(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := r.Context().Value(middleware.TenantIDKey).(int)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Nicht autorisiert")
		return
	}

	// Get invoice ID from URL using gorilla/mux
	vars := mux.Vars(r)
	invoiceID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Rechnungs-ID")
		return
	}

	invoice, err := h.invoiceRepo.GetByID(invoiceID)
	if err != nil {
		log.Printf("ERROR: Failed to get invoice %d: %v", invoiceID, err)
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Rechnung")
		return
	}

	if invoice == nil {
		respondError(w, http.StatusNotFound, "Rechnung nicht gefunden")
		return
	}

	// Security: Verify invoice belongs to this tenant
	if invoice.TenantID != tenantID {
		respondError(w, http.StatusForbidden, "Zugriff verweigert")
		return
	}

	// Check if PDF is available
	if invoice.PDFURL == nil && invoice.PDFPath == nil {
		respondError(w, http.StatusNotFound, "PDF nicht verfügbar")
		return
	}

	// If we have a Stripe PDF URL, return JSON with URL (consistent with S3 response)
	if invoice.PDFURL != nil && *invoice.PDFURL != "" {
		// Security: Validate URL is from Stripe domain to prevent open redirect
		pdfURL := *invoice.PDFURL
		if !strings.HasPrefix(pdfURL, "https://pay.stripe.com/") &&
			!strings.HasPrefix(pdfURL, "https://invoice.stripe.com/") &&
			!strings.HasPrefix(pdfURL, "https://files.stripe.com/") {
			log.Printf("SECURITY: Invoice %d has suspicious PDF URL: %s", invoice.ID, pdfURL)
			respondError(w, http.StatusBadRequest, "Ungültige PDF-URL")
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{
			"url": pdfURL,
		})
		return
	}

	// If we have a local PDF path (S3 key), generate a pre-signed URL
	if invoice.PDFPath != nil && *invoice.PDFPath != "" {
		// Check if S3 service is available
		if h.s3Service == nil {
			log.Printf("ERROR: S3 service not configured but invoice %d has PDFPath", invoice.ID)
			respondError(w, http.StatusServiceUnavailable, "PDF-Download nicht verfügbar")
			return
		}

		// Get tenant slug from context for S3 path
		tenantSlug, ok := r.Context().Value(middleware.TenantSlugKey).(string)
		if !ok || tenantSlug == "" {
			// This should not happen - middleware should always set tenant slug
			log.Printf("WARNING: Missing tenant slug for invoice %d S3 download, using 'default'", invoice.ID)
			tenantSlug = "default"
		}

		// Build S3 object key with tenant isolation
		objectKey, err := h.s3Service.GetObjectKey(tenantSlug, *invoice.PDFPath)
		if err != nil {
			log.Printf("ERROR: Invalid S3 path for invoice %d: %v", invoice.ID, err)
			respondError(w, http.StatusInternalServerError, "Ungültiger PDF-Pfad")
			return
		}

		// Calculate expiry time BEFORE generating URL to avoid time drift
		expiry := time.Hour
		expiresAt := time.Now().Add(expiry)

		// Generate pre-signed URL with 1 hour expiry
		presignedURL, err := h.s3Service.GetPresignedURL(r.Context(), objectKey, expiry)
		if err != nil {
			log.Printf("ERROR: Failed to generate pre-signed URL for invoice %d: %v", invoice.ID, err)
			respondError(w, http.StatusInternalServerError, "Fehler beim Generieren des Download-Links")
			return
		}

		respondJSON(w, http.StatusOK, map[string]string{
			"url":     presignedURL,
			"expires": expiresAt.Format(time.RFC3339),
		})
		return
	}

	respondError(w, http.StatusNotFound, "PDF nicht gefunden")
}
