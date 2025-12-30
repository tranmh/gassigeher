package models

import (
	"encoding/json"
	"time"
)

// PromoCode represents a promotional code for discounts
type PromoCode struct {
	ID             int        `json:"id"`
	Code           string     `json:"code"`
	Description    string     `json:"description,omitempty"`
	DiscountType   string     `json:"discount_type"` // percentage, fixed, free_months
	DiscountValue  int        `json:"discount_value"` // percentage (0-100), cents, or months
	MaxUses        *int       `json:"max_uses,omitempty"`
	UsesCount      int        `json:"uses_count"`
	ValidForPlans  string     `json:"valid_for_plans,omitempty"` // JSON array: ["pro"] or null for all
	IsActive       bool       `json:"is_active"`
	StripeCouponID *string    `json:"stripe_coupon_id,omitempty"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// PromoCodeUse tracks when a promo code is used by a tenant
type PromoCodeUse struct {
	ID          int       `json:"id"`
	PromoCodeID int       `json:"promo_code_id"`
	TenantID    int       `json:"tenant_id"`
	AppliedAt   time.Time `json:"applied_at"`

	// Joined fields
	Code       *string `json:"code,omitempty"`
	TenantName *string `json:"tenant_name,omitempty"`
}

// Discount type constants
const (
	DiscountTypePercentage = "percentage"
	DiscountTypeFixed      = "fixed"
	DiscountTypeFreeMonths = "free_months"
)

// MaxPromoDiscountMonths is the maximum free months a promo code can grant
const MaxPromoDiscountMonths = 24

// MaxPromoDiscountPercent is the maximum percentage discount (100%)
const MaxPromoDiscountPercent = 100

// IsValid checks if the promo code can be used
func (p *PromoCode) IsValid() bool {
	if !p.IsActive {
		return false
	}
	if p.MaxUses != nil && p.UsesCount >= *p.MaxUses {
		return false // Max uses exceeded
	}
	if p.ExpiresAt != nil && time.Now().After(*p.ExpiresAt) {
		return false // Expired
	}
	return true
}

// GetDiscountDescription returns a human-readable description of the discount
func (p *PromoCode) GetDiscountDescription() string {
	switch p.DiscountType {
	case DiscountTypePercentage:
		return intToStr(p.DiscountValue) + "% Rabatt"
	case DiscountTypeFixed:
		cents := p.DiscountValue
		euros := cents / 100
		return "€" + intToStr(euros) + " Rabatt"
	case DiscountTypeFreeMonths:
		if p.DiscountValue == 1 {
			return "1 Monat kostenlos"
		}
		return intToStr(p.DiscountValue) + " Monate kostenlos"
	default:
		return ""
	}
}

// intToStr converts int to string without importing strconv
func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + intToStr(-n)
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}

// Validate validates the promo code fields
func (p *PromoCode) Validate() error {
	if p.Code == "" {
		return &ValidationError{Field: "code", Message: "Code ist erforderlich"}
	}
	if len(p.Code) < 3 || len(p.Code) > 50 {
		return &ValidationError{Field: "code", Message: "Code muss zwischen 3 und 50 Zeichen lang sein"}
	}

	// Validate discount type
	switch p.DiscountType {
	case DiscountTypePercentage:
		if p.DiscountValue < 1 || p.DiscountValue > MaxPromoDiscountPercent {
			return &ValidationError{Field: "discount_value", Message: "Prozent muss zwischen 1 und 100 liegen"}
		}
	case DiscountTypeFixed:
		if p.DiscountValue < 1 {
			return &ValidationError{Field: "discount_value", Message: "Rabattbetrag muss positiv sein"}
		}
	case DiscountTypeFreeMonths:
		if p.DiscountValue < 1 || p.DiscountValue > MaxPromoDiscountMonths {
			return &ValidationError{Field: "discount_value", Message: "Gratismonate müssen zwischen 1 und 24 liegen"}
		}
	default:
		return &ValidationError{Field: "discount_type", Message: "Ungültiger Rabatttyp"}
	}

	// HIGH-9: Validate MaxUses (must be positive if set)
	if p.MaxUses != nil && *p.MaxUses <= 0 {
		return &ValidationError{Field: "max_uses", Message: "Maximale Nutzungen muss positiv sein"}
	}

	// HIGH-9: Validate ExpiresAt (must not be in the past if set)
	if p.ExpiresAt != nil && time.Now().After(*p.ExpiresAt) {
		return &ValidationError{Field: "expires_at", Message: "Ablaufdatum darf nicht in der Vergangenheit liegen"}
	}

	// HIGH-9: Validate ValidForPlans (must be valid JSON array if set)
	if p.ValidForPlans != "" {
		var plans []interface{}
		if err := json.Unmarshal([]byte(p.ValidForPlans), &plans); err != nil {
			return &ValidationError{Field: "valid_for_plans", Message: "Ungültiges JSON-Format für Pläne"}
		}
	}

	return nil
}
