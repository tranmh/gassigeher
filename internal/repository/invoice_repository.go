package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
)

// InvoiceRepository handles invoice database operations
type InvoiceRepository struct {
	db *sql.DB
}

// NewInvoiceRepository creates a new invoice repository
func NewInvoiceRepository(db *sql.DB) *InvoiceRepository {
	return &InvoiceRepository{db: db}
}

// Create creates a new invoice
func (r *InvoiceRepository) Create(invoice *models.TenantInvoice) error {
	query := `
		INSERT INTO tenant_invoices (
			tenant_id, subscription_id, stripe_invoice_id, invoice_number,
			status, amount_cents, currency, period_start, period_end,
			pdf_path, pdf_url, description, created_at, paid_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.Exec(query,
		invoice.TenantID,
		invoice.SubscriptionID,
		invoice.StripeInvoiceID,
		invoice.InvoiceNumber,
		invoice.Status,
		invoice.AmountCents,
		invoice.Currency,
		invoice.PeriodStart,
		invoice.PeriodEnd,
		invoice.PDFPath,
		invoice.PDFURL,
		invoice.Description,
		FormatTimestamp(now),
		invoice.PaidAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create invoice: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get invoice ID: %w", err)
	}
	invoice.ID = int(id)
	invoice.CreatedAt = now

	return nil
}

// GetByID returns an invoice by ID
func (r *InvoiceRepository) GetByID(id int) (*models.TenantInvoice, error) {
	query := `
		SELECT i.id, i.tenant_id, i.subscription_id, i.stripe_invoice_id,
		       i.invoice_number, i.status, i.amount_cents, i.currency,
		       i.period_start, i.period_end, i.pdf_path, i.pdf_url,
		       i.description, i.created_at, i.paid_at, t.name, t.slug
		FROM tenant_invoices i
		JOIN tenants t ON i.tenant_id = t.id
		WHERE i.id = ?
	`

	invoice := &models.TenantInvoice{}
	var stripeInvoiceID, pdfPath, pdfURL, description sql.NullString
	var periodStart, periodEnd, paidAt sql.NullTime
	var tenantName, tenantSlug sql.NullString
	var subscriptionIDInt sql.NullInt64

	err := r.db.QueryRow(query, id).Scan(
		&invoice.ID,
		&invoice.TenantID,
		&subscriptionIDInt,
		&stripeInvoiceID,
		&invoice.InvoiceNumber,
		&invoice.Status,
		&invoice.AmountCents,
		&invoice.Currency,
		&periodStart,
		&periodEnd,
		&pdfPath,
		&pdfURL,
		&description,
		&invoice.CreatedAt,
		&paidAt,
		&tenantName,
		&tenantSlug,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}

	// Set nullable fields
	if subscriptionIDInt.Valid {
		id := int(subscriptionIDInt.Int64)
		invoice.SubscriptionID = &id
	}
	if stripeInvoiceID.Valid {
		invoice.StripeInvoiceID = &stripeInvoiceID.String
	}
	if pdfPath.Valid {
		invoice.PDFPath = &pdfPath.String
	}
	if pdfURL.Valid {
		invoice.PDFURL = &pdfURL.String
	}
	if description.Valid {
		invoice.Description = description.String
	}
	if periodStart.Valid {
		invoice.PeriodStart = &periodStart.Time
	}
	if periodEnd.Valid {
		invoice.PeriodEnd = &periodEnd.Time
	}
	if paidAt.Valid {
		invoice.PaidAt = &paidAt.Time
	}
	if tenantName.Valid {
		invoice.TenantName = &tenantName.String
	}
	if tenantSlug.Valid {
		invoice.TenantSlug = &tenantSlug.String
	}

	return invoice, nil
}

// GetByStripeID returns an invoice by Stripe invoice ID
func (r *InvoiceRepository) GetByStripeID(stripeInvoiceID string) (*models.TenantInvoice, error) {
	query := `
		SELECT id FROM tenant_invoices WHERE stripe_invoice_id = ?
	`

	var id int
	err := r.db.QueryRow(query, stripeInvoiceID).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice by Stripe ID: %w", err)
	}

	return r.GetByID(id)
}

// GetByTenant returns all invoices for a tenant
func (r *InvoiceRepository) GetByTenant(tenantID int) ([]*models.TenantInvoice, error) {
	query := `
		SELECT i.id, i.tenant_id, i.subscription_id, i.stripe_invoice_id,
		       i.invoice_number, i.status, i.amount_cents, i.currency,
		       i.period_start, i.period_end, i.pdf_path, i.pdf_url,
		       i.description, i.created_at, i.paid_at
		FROM tenant_invoices i
		WHERE i.tenant_id = ?
		ORDER BY i.created_at DESC
	`

	rows, err := r.db.Query(query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query invoices: %w", err)
	}
	defer rows.Close()

	invoices := []*models.TenantInvoice{}
	for rows.Next() {
		invoice := &models.TenantInvoice{}
		var subscriptionIDInt sql.NullInt64
		var stripeInvoiceID, pdfPath, pdfURL, description sql.NullString
		var periodStart, periodEnd, paidAt sql.NullTime

		err := rows.Scan(
			&invoice.ID,
			&invoice.TenantID,
			&subscriptionIDInt,
			&stripeInvoiceID,
			&invoice.InvoiceNumber,
			&invoice.Status,
			&invoice.AmountCents,
			&invoice.Currency,
			&periodStart,
			&periodEnd,
			&pdfPath,
			&pdfURL,
			&description,
			&invoice.CreatedAt,
			&paidAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan invoice: %w", err)
		}

		// Set nullable fields
		if subscriptionIDInt.Valid {
			id := int(subscriptionIDInt.Int64)
			invoice.SubscriptionID = &id
		}
		if stripeInvoiceID.Valid {
			invoice.StripeInvoiceID = &stripeInvoiceID.String
		}
		if pdfPath.Valid {
			invoice.PDFPath = &pdfPath.String
		}
		if pdfURL.Valid {
			invoice.PDFURL = &pdfURL.String
		}
		if description.Valid {
			invoice.Description = description.String
		}
		if periodStart.Valid {
			invoice.PeriodStart = &periodStart.Time
		}
		if periodEnd.Valid {
			invoice.PeriodEnd = &periodEnd.Time
		}
		if paidAt.Valid {
			invoice.PaidAt = &paidAt.Time
		}

		invoices = append(invoices, invoice)
	}

	return invoices, nil
}

// GetAll returns all invoices (for central admin)
func (r *InvoiceRepository) GetAll(limit, offset int) ([]*models.TenantInvoice, int, error) {
	// Get total count
	var total int
	err := r.db.QueryRow("SELECT COUNT(*) FROM tenant_invoices").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count invoices: %w", err)
	}

	query := `
		SELECT i.id, i.tenant_id, i.subscription_id, i.stripe_invoice_id,
		       i.invoice_number, i.status, i.amount_cents, i.currency,
		       i.period_start, i.period_end, i.pdf_path, i.pdf_url,
		       i.description, i.created_at, i.paid_at, t.name, t.slug
		FROM tenant_invoices i
		JOIN tenants t ON i.tenant_id = t.id
		ORDER BY i.created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query invoices: %w", err)
	}
	defer rows.Close()

	invoices := []*models.TenantInvoice{}
	for rows.Next() {
		invoice := &models.TenantInvoice{}
		var subscriptionIDInt sql.NullInt64
		var stripeInvoiceID, pdfPath, pdfURL, description sql.NullString
		var periodStart, periodEnd, paidAt sql.NullTime
		var tenantName, tenantSlug sql.NullString

		err := rows.Scan(
			&invoice.ID,
			&invoice.TenantID,
			&subscriptionIDInt,
			&stripeInvoiceID,
			&invoice.InvoiceNumber,
			&invoice.Status,
			&invoice.AmountCents,
			&invoice.Currency,
			&periodStart,
			&periodEnd,
			&pdfPath,
			&pdfURL,
			&description,
			&invoice.CreatedAt,
			&paidAt,
			&tenantName,
			&tenantSlug,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan invoice: %w", err)
		}

		// Set nullable fields
		if subscriptionIDInt.Valid {
			id := int(subscriptionIDInt.Int64)
			invoice.SubscriptionID = &id
		}
		if stripeInvoiceID.Valid {
			invoice.StripeInvoiceID = &stripeInvoiceID.String
		}
		if pdfPath.Valid {
			invoice.PDFPath = &pdfPath.String
		}
		if pdfURL.Valid {
			invoice.PDFURL = &pdfURL.String
		}
		if description.Valid {
			invoice.Description = description.String
		}
		if periodStart.Valid {
			invoice.PeriodStart = &periodStart.Time
		}
		if periodEnd.Valid {
			invoice.PeriodEnd = &periodEnd.Time
		}
		if paidAt.Valid {
			invoice.PaidAt = &paidAt.Time
		}
		if tenantName.Valid {
			invoice.TenantName = &tenantName.String
		}
		if tenantSlug.Valid {
			invoice.TenantSlug = &tenantSlug.String
		}

		invoices = append(invoices, invoice)
	}

	return invoices, total, nil
}

// Update updates an invoice
func (r *InvoiceRepository) Update(invoice *models.TenantInvoice) error {
	query := `
		UPDATE tenant_invoices SET
			status = ?,
			pdf_path = ?,
			pdf_url = ?,
			paid_at = ?
		WHERE id = ?
	`

	_, err := r.db.Exec(query,
		invoice.Status,
		invoice.PDFPath,
		invoice.PDFURL,
		invoice.PaidAt,
		invoice.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update invoice: %w", err)
	}

	return nil
}

// GenerateNextInvoiceNumber generates the next invoice number for a tenant
func (r *InvoiceRepository) GenerateNextInvoiceNumber(tenantID int) string {
	// Format: INV-{YEAR}-{TENANT_ID}-{SEQUENCE}
	year := time.Now().Year()

	var count int
	r.db.QueryRow(`
		SELECT COUNT(*) FROM tenant_invoices
		WHERE tenant_id = ? AND strftime('%Y', created_at) = ?
	`, tenantID, fmt.Sprintf("%d", year)).Scan(&count)

	return fmt.Sprintf("INV-%d-%d-%04d", year, tenantID, count+1)
}
