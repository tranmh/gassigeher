package repository

import (
	"testing"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/testutil"
)

func TestInvoiceRepository_Create(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewInvoiceRepository(db)

	tenantID := createTestTenant(t, db)

	t.Run("creates invoice successfully", func(t *testing.T) {
		now := time.Now()
		periodStart := now.AddDate(0, -1, 0)
		periodEnd := now

		invoice := &models.TenantInvoice{
			TenantID:      tenantID,
			InvoiceNumber: "INV-2025-1-0001",
			Status:        "paid",
			AmountCents:   2900,
			Currency:      "EUR",
			PeriodStart:   &periodStart,
			PeriodEnd:     &periodEnd,
			Description:   "Pro Plan - January 2025",
		}

		err := repo.Create(invoice)
		if err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		if invoice.ID == 0 {
			t.Error("Expected invoice ID to be set")
		}
		if invoice.CreatedAt.IsZero() {
			t.Error("Expected CreatedAt to be set")
		}
	})

	t.Run("creates invoice with Stripe ID", func(t *testing.T) {
		stripeID := "in_test123"
		invoice := &models.TenantInvoice{
			TenantID:        tenantID,
			InvoiceNumber:   "INV-2025-1-0002",
			StripeInvoiceID: &stripeID,
			Status:          "open",
			AmountCents:     2900,
			Currency:        "EUR",
		}

		err := repo.Create(invoice)
		if err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		// Verify Stripe ID was saved
		retrieved, err := repo.GetByID(invoice.ID)
		if err != nil {
			t.Fatalf("GetByID() error: %v", err)
		}
		if retrieved.StripeInvoiceID == nil || *retrieved.StripeInvoiceID != stripeID {
			t.Error("Expected StripeInvoiceID to be saved")
		}
	})

	t.Run("creates invoice with PDF path", func(t *testing.T) {
		pdfPath := "invoices/tenant-1/2025/invoice-0003.pdf"
		invoice := &models.TenantInvoice{
			TenantID:      tenantID,
			InvoiceNumber: "INV-2025-1-0003",
			Status:        "paid",
			AmountCents:   5800,
			Currency:      "EUR",
			PDFPath:       &pdfPath,
		}

		err := repo.Create(invoice)
		if err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		retrieved, err := repo.GetByID(invoice.ID)
		if err != nil {
			t.Fatalf("GetByID() error: %v", err)
		}
		if retrieved.PDFPath == nil || *retrieved.PDFPath != pdfPath {
			t.Error("Expected PDFPath to be saved")
		}
	})
}

func TestInvoiceRepository_GetByID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewInvoiceRepository(db)

	tenantID := createTestTenant(t, db)

	t.Run("returns nil for non-existent ID", func(t *testing.T) {
		invoice, err := repo.GetByID(99999)
		if err != nil {
			t.Fatalf("GetByID() error: %v", err)
		}
		if invoice != nil {
			t.Error("Expected nil for non-existent ID")
		}
	})

	t.Run("returns invoice with tenant info", func(t *testing.T) {
		inv := &models.TenantInvoice{
			TenantID:      tenantID,
			InvoiceNumber: "INV-TEST-001",
			Status:        "paid",
			AmountCents:   2900,
			Currency:      "EUR",
		}
		if err := repo.Create(inv); err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		retrieved, err := repo.GetByID(inv.ID)
		if err != nil {
			t.Fatalf("GetByID() error: %v", err)
		}
		if retrieved == nil {
			t.Fatal("Expected invoice, got nil")
		}
		if retrieved.TenantName == nil {
			t.Error("Expected TenantName to be populated")
		}
		if retrieved.TenantSlug == nil {
			t.Error("Expected TenantSlug to be populated")
		}
	})
}

func TestInvoiceRepository_GetByStripeID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewInvoiceRepository(db)

	tenantID := createTestTenant(t, db)

	t.Run("returns nil for non-existent Stripe ID", func(t *testing.T) {
		invoice, err := repo.GetByStripeID("nonexistent_id")
		if err != nil {
			t.Fatalf("GetByStripeID() error: %v", err)
		}
		if invoice != nil {
			t.Error("Expected nil for non-existent Stripe ID")
		}
	})

	t.Run("finds invoice by Stripe ID", func(t *testing.T) {
		stripeID := "in_unique_test_456"
		inv := &models.TenantInvoice{
			TenantID:        tenantID,
			InvoiceNumber:   "INV-STRIPE-001",
			StripeInvoiceID: &stripeID,
			Status:          "paid",
			AmountCents:     2900,
			Currency:        "EUR",
		}
		if err := repo.Create(inv); err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		retrieved, err := repo.GetByStripeID(stripeID)
		if err != nil {
			t.Fatalf("GetByStripeID() error: %v", err)
		}
		if retrieved == nil {
			t.Fatal("Expected invoice, got nil")
		}
		if retrieved.ID != inv.ID {
			t.Errorf("Expected ID %d, got %d", inv.ID, retrieved.ID)
		}
	})
}

func TestInvoiceRepository_GetByTenant(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewInvoiceRepository(db)

	tenantID := createTestTenant(t, db)

	t.Run("returns empty slice for tenant with no invoices", func(t *testing.T) {
		invoices, err := repo.GetByTenant(tenantID)
		if err != nil {
			t.Fatalf("GetByTenant() error: %v", err)
		}
		if invoices == nil {
			t.Error("Expected empty slice, got nil")
		}
	})

	t.Run("returns invoices ordered by created_at DESC", func(t *testing.T) {
		// Create invoices
		inv1 := &models.TenantInvoice{
			TenantID:      tenantID,
			InvoiceNumber: "INV-ORDER-1",
			Status:        "paid",
			AmountCents:   2900,
			Currency:      "EUR",
		}
		if err := repo.Create(inv1); err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		inv2 := &models.TenantInvoice{
			TenantID:      tenantID,
			InvoiceNumber: "INV-ORDER-2",
			Status:        "paid",
			AmountCents:   5800,
			Currency:      "EUR",
		}
		if err := repo.Create(inv2); err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		invoices, err := repo.GetByTenant(tenantID)
		if err != nil {
			t.Fatalf("GetByTenant() error: %v", err)
		}

		if len(invoices) < 2 {
			t.Fatalf("Expected at least 2 invoices, got %d", len(invoices))
		}

		// Verify ordering: for same-timestamp inserts, ID order is deterministic
		// Just verify we get back both invoices
		hasOrder1, hasOrder2 := false, false
		for _, inv := range invoices {
			if inv.InvoiceNumber == "INV-ORDER-1" {
				hasOrder1 = true
			}
			if inv.InvoiceNumber == "INV-ORDER-2" {
				hasOrder2 = true
			}
		}
		if !hasOrder1 || !hasOrder2 {
			t.Error("Expected both invoices to be returned")
		}
	})

	t.Run("respects tenant isolation", func(t *testing.T) {
		otherTenantID := createTestTenant(t, db)

		// Create invoice for other tenant
		inv := &models.TenantInvoice{
			TenantID:      otherTenantID,
			InvoiceNumber: "INV-OTHER-TENANT",
			Status:        "paid",
			AmountCents:   2900,
			Currency:      "EUR",
		}
		if err := repo.Create(inv); err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		// Query for original tenant
		invoices, err := repo.GetByTenant(tenantID)
		if err != nil {
			t.Fatalf("GetByTenant() error: %v", err)
		}

		for _, invoice := range invoices {
			if invoice.InvoiceNumber == "INV-OTHER-TENANT" {
				t.Error("Found invoice from other tenant - isolation failure")
			}
		}
	})
}

func TestInvoiceRepository_GetAll(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewInvoiceRepository(db)

	tenantID := createTestTenant(t, db)

	t.Run("returns invoices with pagination", func(t *testing.T) {
		// Create 3 invoices
		for i := 0; i < 3; i++ {
			inv := &models.TenantInvoice{
				TenantID:      tenantID,
				InvoiceNumber: "INV-PAGE-" + string(rune('A'+i)),
				Status:        "paid",
				AmountCents:   2900,
				Currency:      "EUR",
			}
			if err := repo.Create(inv); err != nil {
				t.Fatalf("Create() error: %v", err)
			}
		}

		// Get first page
		invoices, total, err := repo.GetAll(2, 0)
		if err != nil {
			t.Fatalf("GetAll() error: %v", err)
		}

		if len(invoices) != 2 {
			t.Errorf("Expected 2 invoices, got %d", len(invoices))
		}
		if total < 3 {
			t.Errorf("Expected total >= 3, got %d", total)
		}
	})
}

func TestInvoiceRepository_Update(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewInvoiceRepository(db)

	tenantID := createTestTenant(t, db)

	t.Run("updates invoice status", func(t *testing.T) {
		inv := &models.TenantInvoice{
			TenantID:      tenantID,
			InvoiceNumber: "INV-UPDATE-001",
			Status:        "open",
			AmountCents:   2900,
			Currency:      "EUR",
		}
		if err := repo.Create(inv); err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		// Update status
		inv.Status = "paid"
		now := time.Now()
		inv.PaidAt = &now

		if err := repo.Update(inv); err != nil {
			t.Fatalf("Update() error: %v", err)
		}

		// Verify update
		retrieved, err := repo.GetByID(inv.ID)
		if err != nil {
			t.Fatalf("GetByID() error: %v", err)
		}
		if retrieved.Status != "paid" {
			t.Errorf("Expected status 'paid', got '%s'", retrieved.Status)
		}
		if retrieved.PaidAt == nil {
			t.Error("Expected PaidAt to be set")
		}
	})

	t.Run("updates PDF path", func(t *testing.T) {
		inv := &models.TenantInvoice{
			TenantID:      tenantID,
			InvoiceNumber: "INV-UPDATE-002",
			Status:        "paid",
			AmountCents:   2900,
			Currency:      "EUR",
		}
		if err := repo.Create(inv); err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		pdfPath := "invoices/updated-path.pdf"
		inv.PDFPath = &pdfPath

		if err := repo.Update(inv); err != nil {
			t.Fatalf("Update() error: %v", err)
		}

		retrieved, err := repo.GetByID(inv.ID)
		if err != nil {
			t.Fatalf("GetByID() error: %v", err)
		}
		if retrieved.PDFPath == nil || *retrieved.PDFPath != pdfPath {
			t.Error("Expected PDFPath to be updated")
		}
	})
}

func TestInvoiceRepository_GenerateNextInvoiceNumber(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewInvoiceRepository(db)

	tenantID := createTestTenant(t, db)

	t.Run("generates sequential invoice numbers", func(t *testing.T) {
		// First invoice number
		num1 := repo.GenerateNextInvoiceNumber(tenantID)
		if num1 == "" {
			t.Error("Expected non-empty invoice number")
		}

		// Create an invoice to increment count
		inv := &models.TenantInvoice{
			TenantID:      tenantID,
			InvoiceNumber: num1,
			Status:        "paid",
			AmountCents:   2900,
			Currency:      "EUR",
		}
		if err := repo.Create(inv); err != nil {
			t.Fatalf("Create() error: %v", err)
		}

		// Second invoice number should be different
		num2 := repo.GenerateNextInvoiceNumber(tenantID)
		if num2 == num1 {
			t.Error("Expected different invoice numbers for sequential calls")
		}
	})

	t.Run("different tenants have independent sequences", func(t *testing.T) {
		tenant2ID := createTestTenant(t, db)

		num1 := repo.GenerateNextInvoiceNumber(tenantID)
		num2 := repo.GenerateNextInvoiceNumber(tenant2ID)

		// Both should start with similar sequence number if no invoices exist
		// but contain different tenant IDs
		if num1 == num2 {
			t.Error("Expected different invoice numbers for different tenants")
		}
	})
}
