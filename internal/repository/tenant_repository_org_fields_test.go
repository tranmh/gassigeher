package repository

import (
	"testing"

	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// TestTenantRepository_OrganizationFields tests CRUD for organization identity fields
// in tenant_settings (organization_name, organization_address, organization_email,
// organization_phone, privacy_officer_email).
func TestTenantRepository_OrganizationFields(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewTenantRepository(db)

	// Create a test tenant
	tenant := &models.Tenant{
		Slug:         "org-fields-test",
		Name:         "Org Fields Test Tenant",
		Status:       models.TenantStatusActive,
		ContactEmail: "org@test.com",
		FederalState: "BW",
	}
	err := repo.Create(tenant)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	t.Run("creates settings with organization fields", func(t *testing.T) {
		orgName := "Tierschutzverein Göppingen e.V."
		orgAddress := "Tierheimstr. 1, 73033 Göppingen"
		orgEmail := "info@tierheim-goeppingen.de"
		orgPhone := "07161 12345"
		privacyEmail := "datenschutz@tierheim-goeppingen.de"

		settings := &models.TenantSettings{
			TenantID:            tenant.ID,
			ThemePreset:         "classic",
			OrganizationName:    &orgName,
			OrganizationAddress: &orgAddress,
			OrganizationEmail:   &orgEmail,
			OrganizationPhone:   &orgPhone,
			PrivacyOfficerEmail: &privacyEmail,
		}
		err := repo.CreateSettings(settings)
		if err != nil {
			t.Fatalf("CreateSettings with org fields returned error: %v", err)
		}

		// Verify by reading back
		got, err := repo.GetSettings(tenant.ID)
		if err != nil {
			t.Fatalf("GetSettings returned error: %v", err)
		}
		if got == nil {
			t.Fatal("Expected settings to be non-nil")
		}

		if got.OrganizationName == nil || *got.OrganizationName != orgName {
			t.Errorf("OrganizationName: expected %q, got %v", orgName, got.OrganizationName)
		}
		if got.OrganizationAddress == nil || *got.OrganizationAddress != orgAddress {
			t.Errorf("OrganizationAddress: expected %q, got %v", orgAddress, got.OrganizationAddress)
		}
		if got.OrganizationEmail == nil || *got.OrganizationEmail != orgEmail {
			t.Errorf("OrganizationEmail: expected %q, got %v", orgEmail, got.OrganizationEmail)
		}
		if got.OrganizationPhone == nil || *got.OrganizationPhone != orgPhone {
			t.Errorf("OrganizationPhone: expected %q, got %v", orgPhone, got.OrganizationPhone)
		}
		if got.PrivacyOfficerEmail == nil || *got.PrivacyOfficerEmail != privacyEmail {
			t.Errorf("PrivacyOfficerEmail: expected %q, got %v", privacyEmail, got.PrivacyOfficerEmail)
		}
	})

	t.Run("updates organization fields", func(t *testing.T) {
		settings, err := repo.GetSettings(tenant.ID)
		if err != nil {
			t.Fatalf("GetSettings failed: %v", err)
		}

		// Update to new values
		newName := "Tierheim Musterstadt e.V."
		newAddress := "Neue Straße 5, 70000 Stuttgart"
		newEmail := "kontakt@tierheim-musterstadt.de"
		newPhone := "0711 98765"
		newPrivacy := "dsgvo@tierheim-musterstadt.de"

		settings.OrganizationName = &newName
		settings.OrganizationAddress = &newAddress
		settings.OrganizationEmail = &newEmail
		settings.OrganizationPhone = &newPhone
		settings.PrivacyOfficerEmail = &newPrivacy

		err = repo.UpdateSettings(settings)
		if err != nil {
			t.Fatalf("UpdateSettings returned error: %v", err)
		}

		// Verify update
		updated, err := repo.GetSettings(tenant.ID)
		if err != nil {
			t.Fatalf("GetSettings after update failed: %v", err)
		}

		if updated.OrganizationName == nil || *updated.OrganizationName != newName {
			t.Errorf("Updated OrganizationName: expected %q, got %v", newName, updated.OrganizationName)
		}
		if updated.OrganizationAddress == nil || *updated.OrganizationAddress != newAddress {
			t.Errorf("Updated OrganizationAddress: expected %q, got %v", newAddress, updated.OrganizationAddress)
		}
		if updated.OrganizationEmail == nil || *updated.OrganizationEmail != newEmail {
			t.Errorf("Updated OrganizationEmail: expected %q, got %v", newEmail, updated.OrganizationEmail)
		}
		if updated.OrganizationPhone == nil || *updated.OrganizationPhone != newPhone {
			t.Errorf("Updated OrganizationPhone: expected %q, got %v", newPhone, updated.OrganizationPhone)
		}
		if updated.PrivacyOfficerEmail == nil || *updated.PrivacyOfficerEmail != newPrivacy {
			t.Errorf("Updated PrivacyOfficerEmail: expected %q, got %v", newPrivacy, updated.PrivacyOfficerEmail)
		}
	})

	t.Run("handles null organization fields", func(t *testing.T) {
		settings, err := repo.GetSettings(tenant.ID)
		if err != nil {
			t.Fatalf("GetSettings failed: %v", err)
		}

		// Set all org fields to nil
		settings.OrganizationName = nil
		settings.OrganizationAddress = nil
		settings.OrganizationEmail = nil
		settings.OrganizationPhone = nil
		settings.PrivacyOfficerEmail = nil

		err = repo.UpdateSettings(settings)
		if err != nil {
			t.Fatalf("UpdateSettings with nil org fields returned error: %v", err)
		}

		// Verify all nil
		updated, err := repo.GetSettings(tenant.ID)
		if err != nil {
			t.Fatalf("GetSettings after nil update failed: %v", err)
		}

		if updated.OrganizationName != nil {
			t.Errorf("Expected nil OrganizationName, got %v", updated.OrganizationName)
		}
		if updated.OrganizationAddress != nil {
			t.Errorf("Expected nil OrganizationAddress, got %v", updated.OrganizationAddress)
		}
		if updated.OrganizationEmail != nil {
			t.Errorf("Expected nil OrganizationEmail, got %v", updated.OrganizationEmail)
		}
		if updated.OrganizationPhone != nil {
			t.Errorf("Expected nil OrganizationPhone, got %v", updated.OrganizationPhone)
		}
		if updated.PrivacyOfficerEmail != nil {
			t.Errorf("Expected nil PrivacyOfficerEmail, got %v", updated.PrivacyOfficerEmail)
		}
	})
}

// TestTenantRepository_DefaultTenantOrgFields verifies org fields work with tenant_id=0 (Simple-Mode)
func TestTenantRepository_DefaultTenantOrgFields(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewTenantRepository(db)

	// Create tenant_settings for default tenant (id=0) if not exists
	// (SetupTestDB creates the tenant row but not always the settings row)
	settings, err := repo.GetSettings(0)
	if err != nil {
		t.Fatalf("GetSettings for default tenant failed: %v", err)
	}
	if settings == nil {
		// Create settings for default tenant
		newSettings := &models.TenantSettings{
			TenantID:    0,
			ThemePreset: "classic",
		}
		if err := repo.CreateSettings(newSettings); err != nil {
			t.Fatalf("Failed to create default tenant settings: %v", err)
		}
		settings, err = repo.GetSettings(0)
		if err != nil || settings == nil {
			t.Fatalf("GetSettings after create failed: %v", err)
		}
	}

	// Organization fields should be nil by default
	if settings.OrganizationName != nil {
		t.Errorf("Expected nil OrganizationName on fresh default tenant, got %v", settings.OrganizationName)
	}

	// Set organization fields on default tenant
	orgName := "Tierheim Test"
	orgEmail := "info@tierheim-test.de"
	settings.OrganizationName = &orgName
	settings.OrganizationEmail = &orgEmail

	err = repo.UpdateSettings(settings)
	if err != nil {
		t.Fatalf("UpdateSettings on default tenant failed: %v", err)
	}

	// Verify
	updated, err := repo.GetSettings(0)
	if err != nil {
		t.Fatalf("GetSettings after update failed: %v", err)
	}
	if updated.OrganizationName == nil || *updated.OrganizationName != orgName {
		t.Errorf("Expected OrganizationName %q, got %v", orgName, updated.OrganizationName)
	}
	if updated.OrganizationEmail == nil || *updated.OrganizationEmail != orgEmail {
		t.Errorf("Expected OrganizationEmail %q, got %v", orgEmail, updated.OrganizationEmail)
	}
}
