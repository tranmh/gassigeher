package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMarketingCampaign_GetFOMOConfig(t *testing.T) {
	tests := []struct {
		name       string
		config     *string
		wantNil    bool
		wantErr    bool
		wantSlots  int
	}{
		{
			name:    "nil config returns nil",
			config:  nil,
			wantNil: true,
			wantErr: false,
		},
		{
			name:      "valid config parses correctly",
			config:    strPtr(`{"total_slots":10,"remaining_slots":5,"message":"Test","cta_text":"Click","cta_link":"/register"}`),
			wantNil:   false,
			wantErr:   false,
			wantSlots: 10,
		},
		{
			name:    "invalid JSON returns error",
			config:  strPtr(`{invalid json}`),
			wantNil: false,
			wantErr: true,
		},
		{
			name:      "empty JSON object returns zero values",
			config:    strPtr(`{}`),
			wantNil:   false,
			wantErr:   false,
			wantSlots: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &MarketingCampaign{Config: tt.config}
			got, err := c.GetFOMOConfig()

			if (err != nil) != tt.wantErr {
				t.Errorf("GetFOMOConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantNil && got != nil {
				t.Errorf("GetFOMOConfig() = %v, want nil", got)
				return
			}

			if !tt.wantNil && !tt.wantErr && got.TotalSlots != tt.wantSlots {
				t.Errorf("GetFOMOConfig() TotalSlots = %v, want %v", got.TotalSlots, tt.wantSlots)
			}
		})
	}
}

func TestMarketingCampaign_SetFOMOConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *FOMOConfig
		wantErr bool
	}{
		{
			name: "valid config serializes correctly",
			config: &FOMOConfig{
				TotalSlots:     10,
				RemainingSlots: 5,
				Message:        "Nur noch 5 Plätze!",
				CTAText:        "Jetzt registrieren",
				CTALink:        "/register",
			},
			wantErr: false,
		},
		{
			name:    "empty config serializes",
			config:  &FOMOConfig{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &MarketingCampaign{}
			err := c.SetFOMOConfig(tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("SetFOMOConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify we can read it back
				got, err := c.GetFOMOConfig()
				if err != nil {
					t.Errorf("GetFOMOConfig() after SetFOMOConfig() error = %v", err)
					return
				}
				if got.TotalSlots != tt.config.TotalSlots {
					t.Errorf("Round-trip TotalSlots = %v, want %v", got.TotalSlots, tt.config.TotalSlots)
				}
				if got.Message != tt.config.Message {
					t.Errorf("Round-trip Message = %v, want %v", got.Message, tt.config.Message)
				}
			}
		})
	}
}

func TestReferralCode_IsValid(t *testing.T) {
	now := time.Now()
	pastTime := now.Add(-24 * time.Hour)
	futureTime := now.Add(24 * time.Hour)

	tests := []struct {
		name string
		code ReferralCode
		want bool
	}{
		{
			name: "active code with no limits is valid",
			code: ReferralCode{
				IsActive: true,
				MaxUses:  nil,
				ExpiresAt: nil,
			},
			want: true,
		},
		{
			name: "inactive code is invalid",
			code: ReferralCode{
				IsActive: false,
			},
			want: false,
		},
		{
			name: "code at max uses is invalid",
			code: ReferralCode{
				IsActive:  true,
				MaxUses:   intPtr(5),
				UsesCount: 5,
			},
			want: false,
		},
		{
			name: "code over max uses is invalid",
			code: ReferralCode{
				IsActive:  true,
				MaxUses:   intPtr(5),
				UsesCount: 6,
			},
			want: false,
		},
		{
			name: "code under max uses is valid",
			code: ReferralCode{
				IsActive:  true,
				MaxUses:   intPtr(5),
				UsesCount: 3,
			},
			want: true,
		},
		{
			name: "expired code is invalid",
			code: ReferralCode{
				IsActive:  true,
				ExpiresAt: &pastTime,
			},
			want: false,
		},
		{
			name: "code with future expiry is valid",
			code: ReferralCode{
				IsActive:  true,
				ExpiresAt: &futureTime,
			},
			want: true,
		},
		{
			name: "inactive expired code is invalid",
			code: ReferralCode{
				IsActive:  false,
				ExpiresAt: &pastTime,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.code.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFOMOConfig_JSONRoundTrip(t *testing.T) {
	original := FOMOConfig{
		TotalSlots:     10,
		RemainingSlots: 3,
		Message:        "Nur noch 3 Plätze kostenlos!",
		CTAText:        "Jetzt sichern",
		CTALink:        "/register?plan=pro",
	}

	// Serialize
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Deserialize
	var parsed FOMOConfig
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	// Compare
	if parsed.TotalSlots != original.TotalSlots {
		t.Errorf("TotalSlots = %v, want %v", parsed.TotalSlots, original.TotalSlots)
	}
	if parsed.RemainingSlots != original.RemainingSlots {
		t.Errorf("RemainingSlots = %v, want %v", parsed.RemainingSlots, original.RemainingSlots)
	}
	if parsed.Message != original.Message {
		t.Errorf("Message = %v, want %v", parsed.Message, original.Message)
	}
	if parsed.CTAText != original.CTAText {
		t.Errorf("CTAText = %v, want %v", parsed.CTAText, original.CTAText)
	}
	if parsed.CTALink != original.CTALink {
		t.Errorf("CTALink = %v, want %v", parsed.CTALink, original.CTALink)
	}
}

func TestCreateReferralCodeRequest_Validation(t *testing.T) {
	// Test that the struct fields work as expected
	req := CreateReferralCodeRequest{
		Code:                   "TEST123",
		ReferrerEmail:          strPtr("test@example.com"),
		DiscountMonthsReferrer: 1,
		DiscountMonthsReferee:  2,
		MaxUses:                intPtr(10),
		ExpiresAt:              strPtr("2025-12-31"),
	}

	if req.Code != "TEST123" {
		t.Errorf("Code = %v, want TEST123", req.Code)
	}
	if *req.ReferrerEmail != "test@example.com" {
		t.Errorf("ReferrerEmail = %v, want test@example.com", *req.ReferrerEmail)
	}
	if req.DiscountMonthsReferrer != 1 {
		t.Errorf("DiscountMonthsReferrer = %v, want 1", req.DiscountMonthsReferrer)
	}
	if req.DiscountMonthsReferee != 2 {
		t.Errorf("DiscountMonthsReferee = %v, want 2", req.DiscountMonthsReferee)
	}
	if *req.MaxUses != 10 {
		t.Errorf("MaxUses = %v, want 10", *req.MaxUses)
	}
}

func TestMarketingStatsResponse(t *testing.T) {
	stats := MarketingStatsResponse{
		ActiveCampaigns:    2,
		TotalReferralCodes: 10,
		TotalReferralUses:  25,
		ApprovedReferences: 5,
		PendingReferences:  3,
	}

	// Test JSON serialization
	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var parsed MarketingStatsResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if parsed.ActiveCampaigns != stats.ActiveCampaigns {
		t.Errorf("ActiveCampaigns = %v, want %v", parsed.ActiveCampaigns, stats.ActiveCampaigns)
	}
	if parsed.TotalReferralCodes != stats.TotalReferralCodes {
		t.Errorf("TotalReferralCodes = %v, want %v", parsed.TotalReferralCodes, stats.TotalReferralCodes)
	}
}

func TestReferenceEntry_Fields(t *testing.T) {
	entry := ReferenceEntry{
		ID:           1,
		TenantID:     2,
		DisplayName:  "Tierheim Berlin",
		City:         strPtr("Berlin"),
		FederalState: strPtr("BE"),
		WebsiteURL:   strPtr("https://tierheim-berlin.de"),
		Testimonial:  strPtr("Tolle Plattform!"),
		LogoURL:      strPtr("/uploads/logos/berlin.png"),
		IsApproved:   true,
		DisplayOrder: 1,
	}

	if entry.DisplayName != "Tierheim Berlin" {
		t.Errorf("DisplayName = %v, want Tierheim Berlin", entry.DisplayName)
	}
	if *entry.City != "Berlin" {
		t.Errorf("City = %v, want Berlin", *entry.City)
	}
	if !entry.IsApproved {
		t.Error("IsApproved = false, want true")
	}
}

// Helper function - strPtr
// Note: intPtr is already defined in test_helpers.go
func strPtr(s string) *string {
	return &s
}
