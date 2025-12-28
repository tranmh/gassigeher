package models

import (
	"time"
)

// TenantInvoice represents an invoice for a tenant's subscription
type TenantInvoice struct {
	ID              int        `json:"id"`
	TenantID        int        `json:"tenant_id"`
	SubscriptionID  *int       `json:"subscription_id,omitempty"`
	StripeInvoiceID *string    `json:"stripe_invoice_id,omitempty"`
	InvoiceNumber   string     `json:"invoice_number"`
	Status          string     `json:"status"` // draft, open, paid, void, uncollectible
	AmountCents     int        `json:"amount_cents"`
	Currency        string     `json:"currency"`
	PeriodStart     *time.Time `json:"period_start,omitempty"`
	PeriodEnd       *time.Time `json:"period_end,omitempty"`
	PDFPath         *string    `json:"pdf_path,omitempty"` // S3 path or local path
	PDFURL          *string    `json:"pdf_url,omitempty"`  // Full URL for download
	Description     string     `json:"description,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	PaidAt          *time.Time `json:"paid_at,omitempty"`

	// Joined fields
	TenantName *string `json:"tenant_name,omitempty"`
	TenantSlug *string `json:"tenant_slug,omitempty"`
}

// Invoice status constants
const (
	InvoiceStatusDraft         = "draft"
	InvoiceStatusOpen          = "open"
	InvoiceStatusPaid          = "paid"
	InvoiceStatusVoid          = "void"
	InvoiceStatusUncollectible = "uncollectible"
)

// GetAmountFormatted returns the amount in euros (formatted)
func (i *TenantInvoice) GetAmountFormatted() string {
	euros := i.AmountCents / 100
	cents := i.AmountCents % 100
	if cents == 0 {
		return "€" + intToStr(euros)
	}
	centStr := intToStr(cents)
	if cents < 10 {
		centStr = "0" + centStr
	}
	return "€" + intToStr(euros) + "," + centStr
}

// GetStatusLabel returns a German label for the invoice status
func (i *TenantInvoice) GetStatusLabel() string {
	switch i.Status {
	case InvoiceStatusDraft:
		return "Entwurf"
	case InvoiceStatusOpen:
		return "Offen"
	case InvoiceStatusPaid:
		return "Bezahlt"
	case InvoiceStatusVoid:
		return "Storniert"
	case InvoiceStatusUncollectible:
		return "Uneinbringlich"
	default:
		return i.Status
	}
}

// IsPaid returns true if the invoice is paid
func (i *TenantInvoice) IsPaid() bool {
	return i.Status == InvoiceStatusPaid
}

// Validate validates the invoice fields
func (i *TenantInvoice) Validate() error {
	if i.TenantID == 0 {
		return &ValidationError{Field: "tenant_id", Message: "Tenant-ID ist erforderlich"}
	}
	if i.InvoiceNumber == "" {
		return &ValidationError{Field: "invoice_number", Message: "Rechnungsnummer ist erforderlich"}
	}
	if i.AmountCents < 0 {
		return &ValidationError{Field: "amount_cents", Message: "Betrag darf nicht negativ sein"}
	}

	// Validate status
	switch i.Status {
	case InvoiceStatusDraft, InvoiceStatusOpen, InvoiceStatusPaid, InvoiceStatusVoid, InvoiceStatusUncollectible:
		// Valid
	default:
		return &ValidationError{Field: "status", Message: "Ungültiger Status"}
	}

	return nil
}
