package models

import (
	"strings"
	"testing"
	"time"
)

// HIGH-9: Test PromoCode validation for additional constraints
func TestPromoCode_Validate(t *testing.T) {
	// Existing validation tests
	t.Run("valid promo code passes", func(t *testing.T) {
		code := &PromoCode{
			Code:          "WELCOME10",
			DiscountType:  DiscountTypePercentage,
			DiscountValue: 10,
		}
		if err := code.Validate(); err != nil {
			t.Errorf("Valid promo code should pass: %v", err)
		}
	})

	t.Run("empty code fails", func(t *testing.T) {
		code := &PromoCode{
			Code:          "",
			DiscountType:  DiscountTypePercentage,
			DiscountValue: 10,
		}
		if err := code.Validate(); err == nil {
			t.Error("Empty code should fail validation")
		}
	})

	t.Run("code too short fails", func(t *testing.T) {
		code := &PromoCode{
			Code:          "AB",
			DiscountType:  DiscountTypePercentage,
			DiscountValue: 10,
		}
		if err := code.Validate(); err == nil {
			t.Error("Code < 3 chars should fail validation")
		}
	})

	t.Run("code too long fails", func(t *testing.T) {
		code := &PromoCode{
			Code:          strings.Repeat("A", 51),
			DiscountType:  DiscountTypePercentage,
			DiscountValue: 10,
		}
		if err := code.Validate(); err == nil {
			t.Error("Code > 50 chars should fail validation")
		}
	})

	// HIGH-9: New constraint validation tests

	// MaxUses validation
	t.Run("HIGH-9: negative MaxUses fails", func(t *testing.T) {
		negativeMaxUses := -1
		code := &PromoCode{
			Code:          "NEGMAX",
			DiscountType:  DiscountTypePercentage,
			DiscountValue: 10,
			MaxUses:       &negativeMaxUses,
		}
		if err := code.Validate(); err == nil {
			t.Error("Negative MaxUses should fail validation")
		}
	})

	t.Run("HIGH-9: zero MaxUses fails", func(t *testing.T) {
		zeroMaxUses := 0
		code := &PromoCode{
			Code:          "ZEROMAX",
			DiscountType:  DiscountTypePercentage,
			DiscountValue: 10,
			MaxUses:       &zeroMaxUses,
		}
		if err := code.Validate(); err == nil {
			t.Error("Zero MaxUses should fail validation")
		}
	})

	t.Run("HIGH-9: positive MaxUses passes", func(t *testing.T) {
		positiveMaxUses := 100
		code := &PromoCode{
			Code:          "POSMAX",
			DiscountType:  DiscountTypePercentage,
			DiscountValue: 10,
			MaxUses:       &positiveMaxUses,
		}
		if err := code.Validate(); err != nil {
			t.Errorf("Positive MaxUses should pass: %v", err)
		}
	})

	t.Run("HIGH-9: nil MaxUses (unlimited) passes", func(t *testing.T) {
		code := &PromoCode{
			Code:          "UNLIMITED",
			DiscountType:  DiscountTypePercentage,
			DiscountValue: 10,
			MaxUses:       nil,
		}
		if err := code.Validate(); err != nil {
			t.Errorf("Nil MaxUses should pass: %v", err)
		}
	})

	// ExpiresAt validation
	t.Run("HIGH-9: past ExpiresAt fails", func(t *testing.T) {
		pastDate := time.Now().Add(-24 * time.Hour)
		code := &PromoCode{
			Code:          "PASTEXP",
			DiscountType:  DiscountTypePercentage,
			DiscountValue: 10,
			ExpiresAt:     &pastDate,
		}
		if err := code.Validate(); err == nil {
			t.Error("Past ExpiresAt should fail validation")
		}
	})

	t.Run("HIGH-9: future ExpiresAt passes", func(t *testing.T) {
		futureDate := time.Now().Add(24 * time.Hour)
		code := &PromoCode{
			Code:          "FUTEXP",
			DiscountType:  DiscountTypePercentage,
			DiscountValue: 10,
			ExpiresAt:     &futureDate,
		}
		if err := code.Validate(); err != nil {
			t.Errorf("Future ExpiresAt should pass: %v", err)
		}
	})

	t.Run("HIGH-9: nil ExpiresAt (no expiration) passes", func(t *testing.T) {
		code := &PromoCode{
			Code:          "NOEXP",
			DiscountType:  DiscountTypePercentage,
			DiscountValue: 10,
			ExpiresAt:     nil,
		}
		if err := code.Validate(); err != nil {
			t.Errorf("Nil ExpiresAt should pass: %v", err)
		}
	})

	// ValidForPlans validation
	t.Run("HIGH-9: invalid JSON in ValidForPlans fails", func(t *testing.T) {
		code := &PromoCode{
			Code:          "BADJSON",
			DiscountType:  DiscountTypePercentage,
			DiscountValue: 10,
			ValidForPlans: "not valid json{[",
		}
		if err := code.Validate(); err == nil {
			t.Error("Invalid JSON in ValidForPlans should fail validation")
		}
	})

	t.Run("HIGH-9: valid JSON array in ValidForPlans passes", func(t *testing.T) {
		code := &PromoCode{
			Code:          "GOODJSON",
			DiscountType:  DiscountTypePercentage,
			DiscountValue: 10,
			ValidForPlans: `["pro", "enterprise"]`,
		}
		if err := code.Validate(); err != nil {
			t.Errorf("Valid JSON array should pass: %v", err)
		}
	})

	t.Run("HIGH-9: empty ValidForPlans passes", func(t *testing.T) {
		code := &PromoCode{
			Code:          "NOPLANS",
			DiscountType:  DiscountTypePercentage,
			DiscountValue: 10,
			ValidForPlans: "",
		}
		if err := code.Validate(); err != nil {
			t.Errorf("Empty ValidForPlans should pass: %v", err)
		}
	})

	t.Run("HIGH-9: ValidForPlans must be array not object", func(t *testing.T) {
		code := &PromoCode{
			Code:          "OBJJSON",
			DiscountType:  DiscountTypePercentage,
			DiscountValue: 10,
			ValidForPlans: `{"plan": "pro"}`,
		}
		if err := code.Validate(); err == nil {
			t.Error("JSON object (not array) in ValidForPlans should fail validation")
		}
	})

	// Discount type validation (existing tests)
	t.Run("percentage discount over 100 fails", func(t *testing.T) {
		code := &PromoCode{
			Code:          "OVER100",
			DiscountType:  DiscountTypePercentage,
			DiscountValue: 101,
		}
		if err := code.Validate(); err == nil {
			t.Error("Percentage > 100 should fail validation")
		}
	})

	t.Run("free months over 24 fails", func(t *testing.T) {
		code := &PromoCode{
			Code:          "OVER24",
			DiscountType:  DiscountTypeFreeMonths,
			DiscountValue: 25,
		}
		if err := code.Validate(); err == nil {
			t.Error("Free months > 24 should fail validation")
		}
	})

	t.Run("invalid discount type fails", func(t *testing.T) {
		code := &PromoCode{
			Code:          "BADTYPE",
			DiscountType:  "invalid_type",
			DiscountValue: 10,
		}
		if err := code.Validate(); err == nil {
			t.Error("Invalid discount type should fail validation")
		}
	})
}

func TestPromoCode_IsValid(t *testing.T) {
	t.Run("inactive code is not valid", func(t *testing.T) {
		code := &PromoCode{
			Code:     "INACTIVE",
			IsActive: false,
		}
		if code.IsValid() {
			t.Error("Inactive code should not be valid")
		}
	})

	t.Run("expired code is not valid", func(t *testing.T) {
		pastDate := time.Now().Add(-24 * time.Hour)
		code := &PromoCode{
			Code:      "EXPIRED",
			IsActive:  true,
			ExpiresAt: &pastDate,
		}
		if code.IsValid() {
			t.Error("Expired code should not be valid")
		}
	})

	t.Run("max uses exceeded is not valid", func(t *testing.T) {
		maxUses := 5
		code := &PromoCode{
			Code:      "MAXED",
			IsActive:  true,
			MaxUses:   &maxUses,
			UsesCount: 5,
		}
		if code.IsValid() {
			t.Error("Code with max uses reached should not be valid")
		}
	})

	t.Run("valid active code is valid", func(t *testing.T) {
		futureDate := time.Now().Add(24 * time.Hour)
		maxUses := 10
		code := &PromoCode{
			Code:      "VALID",
			IsActive:  true,
			ExpiresAt: &futureDate,
			MaxUses:   &maxUses,
			UsesCount: 3,
		}
		if !code.IsValid() {
			t.Error("Valid active code should be valid")
		}
	})
}
