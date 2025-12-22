# 2-Tier Pricing Implementation Plan

> **Methodology:** Test-Driven Development (TDD)
> **Testing:** Go tests (`go test ./...`) + Jest (frontend)
> **Build:** `./bat.sh`

---

## Executive Summary

Implement a simple 2-tier pricing model for Gassigeher SaaS with **maximum transparency**:

| Tier | Dogs | Price | Features |
|------|------|-------|----------|
| **Free** | 10 | €0/month | All features, forever |
| **Pro** | Unlimited | €29/month or €290/year | All features |

**Philosophy:**
- One limit, nothing else (maximum transparency)
- Free tier IS the trial
- Open Source on GitHub = ultimate trial
- Soft downgrade (existing dogs stay, can't add new)

---

## Table of Contents

1. [Current State](#current-state)
2. [Missing Features](#missing-features)
3. [TDD Implementation Phases](#tdd-implementation-phases)
4. [Database Schema](#database-schema)
5. [API Endpoints](#api-endpoints)
6. [File Changes](#file-changes)
7. [Stripe Setup](#stripe-setup)
8. [Testing Strategy](#testing-strategy)

---

## Current State

### What Exists (Ready)
- Multi-tenant architecture with `tenant_id` on 14 tables
- Tenant model with slug, name, status, settings
- Dog model with `TenantID` field
- Repository pattern with tenant filtering
- Middleware extracts tenant from subdomain/JWT
- Go test infrastructure (`internal/*_test.go`)
- Build script (`./bat.sh`)

### Critical Bug Found
**`CreateDog()` handler doesn't set `TenantID` on the dog!**

Location: `internal/handlers/dog_handler.go` line ~133-204

---

## Missing Features

### Phase 1: Dog Limit Foundation
| Feature | Status | Test File |
|---------|--------|-----------|
| `CountByTenant()` repository method | Missing | `dog_repository_test.go` |
| TenantID set in CreateDog handler | Bug | `dog_handler_test.go` |
| 10-dog limit enforcement | Missing | `dog_handler_test.go` |
| `max_dogs` column on tenants | Missing | Migration |

### Phase 2: Subscription Model
| Feature | Status | Test File |
|---------|--------|-----------|
| `PricingPlan` model | Missing | `subscription_test.go` |
| `TenantSubscription` model | Missing | `subscription_test.go` |
| Subscription repository | Missing | `subscription_repository_test.go` |
| Database migrations | Missing | Migration files |

### Phase 3: Stripe Integration
| Feature | Status | Test File |
|---------|--------|-----------|
| Stripe service | Missing | `stripe_service_test.go` |
| Billing handler | Missing | `billing_handler_test.go` |
| Webhook verification | Missing | `billing_handler_test.go` |

### Phase 4-6: Frontend
| Feature | Status | Test File |
|---------|--------|-----------|
| Landing page transparency | Missing | Jest tests |
| Billing page | Missing | Jest tests |
| Usage indicators | Missing | Jest tests |
| API client methods | Missing | Jest tests |

---

## TDD Implementation Phases

### Phase 1: Dog Limit Foundation

#### Step 1.1: CountByTenant (RED → GREEN)

**RED Phase - Write Failing Test:**
```go
// internal/repository/dog_repository_test.go

func TestDogRepository_CountByTenant(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    repo := NewDogRepository(db)
    tenantID := 1

    // Create 3 dogs for tenant 1
    for i := 0; i < 3; i++ {
        dog := &models.Dog{
            Name:     fmt.Sprintf("Dog %d", i),
            TenantID: tenantID,
        }
        _, err := repo.Create(dog)
        require.NoError(t, err)
    }

    // Create 2 dogs for tenant 2 (should not be counted)
    for i := 0; i < 2; i++ {
        dog := &models.Dog{
            Name:     fmt.Sprintf("Other Dog %d", i),
            TenantID: 2,
        }
        _, err := repo.Create(dog)
        require.NoError(t, err)
    }

    // Test count for tenant 1
    count, err := repo.CountByTenant(tenantID)
    require.NoError(t, err)
    assert.Equal(t, 3, count)

    // Test count for tenant 2
    count, err = repo.CountByTenant(2)
    require.NoError(t, err)
    assert.Equal(t, 2, count)

    // Test count for non-existent tenant
    count, err = repo.CountByTenant(999)
    require.NoError(t, err)
    assert.Equal(t, 0, count)
}
```

**GREEN Phase - Implement:**
```go
// internal/repository/dog_repository.go

func (r *DogRepository) CountByTenant(tenantID int) (int, error) {
    var count int
    query := `SELECT COUNT(*) FROM dogs WHERE tenant_id = ?`
    err := r.db.QueryRow(query, tenantID).Scan(&count)
    if err != nil {
        return 0, err
    }
    return count, nil
}
```

**Verify:** `go test ./internal/repository/... -run TestDogRepository_CountByTenant -v`

---

#### Step 1.2: CreateDog TenantID Fix (RED → GREEN)

**RED Phase - Write Failing Test:**
```go
// internal/handlers/dog_handler_test.go

func TestDogHandler_CreateDog_SetsTenantID(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    handler := NewDogHandler(db, testConfig)
    tenantID := 1

    // Create request with tenant context
    body := `{"name": "Bella", "breed": "Labrador", "size": "large", "color": "braun"}`
    req := httptest.NewRequest("POST", "/api/dogs", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    // Add tenant to context
    ctx := context.WithValue(req.Context(), middleware.TenantIDKey, tenantID)
    req = req.WithContext(ctx)

    rr := httptest.NewRecorder()
    handler.CreateDog(rr, req)

    assert.Equal(t, http.StatusCreated, rr.Code)

    // Verify dog was created with correct tenant_id
    var response map[string]interface{}
    json.Unmarshal(rr.Body.Bytes(), &response)

    dogID := int(response["id"].(float64))
    dog, _ := handler.dogRepo.FindByID(dogID)
    assert.Equal(t, tenantID, dog.TenantID)
}
```

**GREEN Phase - Fix Handler:**
```go
// internal/handlers/dog_handler.go - CreateDog method

// Add after line ~150 (after validation):
tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)
dog.TenantID = tenantID
```

**Verify:** `go test ./internal/handlers/... -run TestDogHandler_CreateDog_SetsTenantID -v`

---

#### Step 1.3: Dog Limit Enforcement (RED → GREEN)

**RED Phase - Write Failing Test:**
```go
// internal/handlers/dog_handler_test.go

func TestDogHandler_CreateDog_EnforcesLimit(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    handler := NewDogHandler(db, testConfig)
    tenantID := 1
    maxDogs := 10

    // Create 10 dogs (at limit)
    for i := 0; i < maxDogs; i++ {
        body := fmt.Sprintf(`{"name": "Dog%d", "breed": "Mix", "size": "medium", "color": "schwarz"}`, i)
        req := httptest.NewRequest("POST", "/api/dogs", strings.NewReader(body))
        req.Header.Set("Content-Type", "application/json")
        ctx := context.WithValue(req.Context(), middleware.TenantIDKey, tenantID)
        req = req.WithContext(ctx)

        rr := httptest.NewRecorder()
        handler.CreateDog(rr, req)
        require.Equal(t, http.StatusCreated, rr.Code, "Dog %d should be created", i)
    }

    // Try to create 11th dog - should fail
    body := `{"name": "OverLimit", "breed": "Mix", "size": "medium", "color": "schwarz"}`
    req := httptest.NewRequest("POST", "/api/dogs", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    ctx := context.WithValue(req.Context(), middleware.TenantIDKey, tenantID)
    req = req.WithContext(ctx)

    rr := httptest.NewRecorder()
    handler.CreateDog(rr, req)

    assert.Equal(t, http.StatusConflict, rr.Code)

    var response map[string]string
    json.Unmarshal(rr.Body.Bytes(), &response)
    assert.Contains(t, response["error"], "Maximum")
}

func TestDogHandler_CreateDog_AllowsUnlimitedForPro(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    // Set tenant to Pro (max_dogs = -1)
    _, err := db.Exec("UPDATE tenants SET max_dogs = -1 WHERE id = ?", 1)
    require.NoError(t, err)

    handler := NewDogHandler(db, testConfig)
    tenantID := 1

    // Create 15 dogs (beyond free limit)
    for i := 0; i < 15; i++ {
        body := fmt.Sprintf(`{"name": "ProDog%d", "breed": "Mix", "size": "medium", "color": "schwarz"}`, i)
        req := httptest.NewRequest("POST", "/api/dogs", strings.NewReader(body))
        req.Header.Set("Content-Type", "application/json")
        ctx := context.WithValue(req.Context(), middleware.TenantIDKey, tenantID)
        req = req.WithContext(ctx)

        rr := httptest.NewRecorder()
        handler.CreateDog(rr, req)
        assert.Equal(t, http.StatusCreated, rr.Code, "Pro tenant dog %d should be created", i)
    }
}
```

**GREEN Phase - Implement Limit Check:**
```go
// internal/handlers/dog_handler.go - CreateDog method

// Add after TenantID extraction:
// Check dog limit for tenant
if tenantID > 0 {
    count, err := h.dogRepo.CountByTenant(tenantID)
    if err != nil {
        respondError(w, http.StatusInternalServerError, "Fehler beim Prüfen des Hundelimits")
        return
    }

    maxDogs, err := h.tenantRepo.GetMaxDogs(tenantID)
    if err != nil {
        respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Tenant-Einstellungen")
        return
    }

    // -1 means unlimited (Pro tier)
    if maxDogs != -1 && count >= maxDogs {
        respondError(w, http.StatusConflict, fmt.Sprintf("Maximum %d Hunde erreicht. Upgrade auf Pro für unbegrenzte Hunde.", maxDogs))
        return
    }
}
```

**Verify:** `go test ./internal/handlers/... -run TestDogHandler_CreateDog_EnforcesLimit -v`

---

#### Step 1.4: Migration for max_dogs

**File:** `internal/database/022_add_max_dogs.go`

```go
package database

func init() {
    RegisterMigration(&Migration{
        ID: "022_add_max_dogs",
        Up: map[string]string{
            "sqlite": `
                ALTER TABLE tenants ADD COLUMN max_dogs INTEGER DEFAULT 10;
                ALTER TABLE tenants ADD COLUMN subscription_status VARCHAR(20) DEFAULT 'free';
            `,
            "mysql": `
                ALTER TABLE tenants ADD COLUMN max_dogs INT DEFAULT 10;
                ALTER TABLE tenants ADD COLUMN subscription_status VARCHAR(20) DEFAULT 'free';
            `,
            "postgres": `
                ALTER TABLE tenants ADD COLUMN max_dogs INTEGER DEFAULT 10;
                ALTER TABLE tenants ADD COLUMN subscription_status VARCHAR(20) DEFAULT 'free';
            `,
        },
        Down: map[string]string{
            "sqlite":   `SELECT 1;`, // SQLite doesn't support DROP COLUMN easily
            "mysql":    `ALTER TABLE tenants DROP COLUMN max_dogs, DROP COLUMN subscription_status;`,
            "postgres": `ALTER TABLE tenants DROP COLUMN max_dogs, DROP COLUMN subscription_status;`,
        },
    })
}
```

**Verify:** `./bat.sh` (builds and runs all tests)

---

### Phase 2: Subscription Model

#### Step 2.1: Subscription Models (RED → GREEN)

**RED Phase - Write Failing Test:**
```go
// internal/models/subscription_test.go

func TestPricingPlan_Validate(t *testing.T) {
    tests := []struct {
        name    string
        plan    PricingPlan
        wantErr bool
    }{
        {
            name: "valid free plan",
            plan: PricingPlan{
                Name:         "Free",
                Slug:         "free",
                MaxDogs:      10,
                PriceMonthly: 0,
                PriceYearly:  0,
            },
            wantErr: false,
        },
        {
            name: "valid pro plan",
            plan: PricingPlan{
                Name:         "Pro",
                Slug:         "pro",
                MaxDogs:      -1, // unlimited
                PriceMonthly: 2900,
                PriceYearly:  29000,
            },
            wantErr: false,
        },
        {
            name:    "missing name",
            plan:    PricingPlan{Slug: "test"},
            wantErr: true,
        },
        {
            name:    "missing slug",
            plan:    PricingPlan{Name: "Test"},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.plan.Validate()
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}

func TestTenantSubscription_IsActive(t *testing.T) {
    now := time.Now()

    tests := []struct {
        name   string
        sub    TenantSubscription
        active bool
    }{
        {
            name:   "active subscription",
            sub:    TenantSubscription{Status: "active", CurrentPeriodEnd: now.Add(24 * time.Hour)},
            active: true,
        },
        {
            name:   "cancelled subscription",
            sub:    TenantSubscription{Status: "cancelled", CurrentPeriodEnd: now.Add(24 * time.Hour)},
            active: false,
        },
        {
            name:   "expired subscription",
            sub:    TenantSubscription{Status: "active", CurrentPeriodEnd: now.Add(-24 * time.Hour)},
            active: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            assert.Equal(t, tt.active, tt.sub.IsActive())
        })
    }
}
```

**GREEN Phase - Implement:**
```go
// internal/models/subscription.go

package models

import (
    "errors"
    "time"
)

// PricingPlan represents a subscription plan (Free, Pro)
type PricingPlan struct {
    ID           int       `json:"id"`
    Name         string    `json:"name"`           // "Free", "Pro"
    Slug         string    `json:"slug"`           // "free", "pro"
    MaxDogs      int       `json:"max_dogs"`       // 10, -1 (unlimited)
    PriceMonthly int       `json:"price_monthly"`  // cents: 0, 2900
    PriceYearly  int       `json:"price_yearly"`   // cents: 0, 29000
    IsActive     bool      `json:"is_active"`
    CreatedAt    time.Time `json:"created_at"`
}

func (p *PricingPlan) Validate() error {
    if p.Name == "" {
        return errors.New("name is required")
    }
    if p.Slug == "" {
        return errors.New("slug is required")
    }
    return nil
}

// TenantSubscription represents a tenant's subscription to a plan
type TenantSubscription struct {
    ID                   int        `json:"id"`
    TenantID             int        `json:"tenant_id"`
    PlanID               int        `json:"plan_id"`
    Status               string     `json:"status"`        // "active", "cancelled", "past_due"
    BillingCycle         string     `json:"billing_cycle"` // "monthly", "yearly"
    CurrentPeriodStart   time.Time  `json:"current_period_start"`
    CurrentPeriodEnd     time.Time  `json:"current_period_end"`
    StripeCustomerID     string     `json:"stripe_customer_id,omitempty"`
    StripeSubscriptionID string     `json:"stripe_subscription_id,omitempty"`
    CreatedAt            time.Time  `json:"created_at"`
    UpdatedAt            time.Time  `json:"updated_at"`

    // Joined fields
    Plan *PricingPlan `json:"plan,omitempty"`
}

func (s *TenantSubscription) IsActive() bool {
    return s.Status == "active" && s.CurrentPeriodEnd.After(time.Now())
}

func (s *TenantSubscription) IsPro() bool {
    return s.Plan != nil && s.Plan.Slug == "pro" && s.IsActive()
}
```

**Verify:** `go test ./internal/models/... -run TestPricingPlan -v`

---

### Phase 3: Stripe Integration

#### Step 3.1: Stripe Service (RED → GREEN)

**RED Phase - Write Failing Test:**
```go
// internal/services/stripe_service_test.go

func TestStripeService_CreateCheckoutSession(t *testing.T) {
    // Use Stripe test mode
    cfg := &config.Config{
        StripeSecretKey:    "sk_test_xxx",
        StripePriceMonthly: "price_test_monthly",
        StripePriceYearly:  "price_test_yearly",
    }

    service := NewStripeService(cfg)

    session, err := service.CreateCheckoutSession(1, "monthly", "https://example.com/success", "https://example.com/cancel")

    // In test mode, this should return a valid session
    assert.NoError(t, err)
    assert.NotEmpty(t, session.ID)
    assert.Contains(t, session.URL, "checkout.stripe.com")
}

func TestStripeService_VerifyWebhookSignature(t *testing.T) {
    cfg := &config.Config{
        StripeWebhookSecret: "whsec_test",
    }

    service := NewStripeService(cfg)

    // Test with invalid signature
    _, err := service.VerifyWebhookSignature([]byte("payload"), "invalid_sig")
    assert.Error(t, err)
}
```

---

## Database Schema

### New Tables

```sql
-- pricing_plans (seed data)
CREATE TABLE pricing_plans (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(50) NOT NULL,
    slug VARCHAR(50) UNIQUE NOT NULL,
    max_dogs INTEGER NOT NULL,
    price_monthly INTEGER NOT NULL,
    price_yearly INTEGER NOT NULL,
    is_active INTEGER DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO pricing_plans (name, slug, max_dogs, price_monthly, price_yearly)
VALUES
    ('Free', 'free', 10, 0, 0),
    ('Pro', 'pro', -1, 2900, 29000);

-- tenant_subscriptions
CREATE TABLE tenant_subscriptions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id INTEGER NOT NULL,
    plan_id INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    billing_cycle VARCHAR(10),
    current_period_start TIMESTAMP,
    current_period_end TIMESTAMP,
    stripe_customer_id VARCHAR(100),
    stripe_subscription_id VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    FOREIGN KEY (plan_id) REFERENCES pricing_plans(id)
);

-- tenants table additions
ALTER TABLE tenants ADD COLUMN max_dogs INTEGER DEFAULT 10;
ALTER TABLE tenants ADD COLUMN subscription_status VARCHAR(20) DEFAULT 'free';
```

---

## API Endpoints

### New Endpoints

| Method | Endpoint | Description | Auth | Test |
|--------|----------|-------------|------|------|
| GET | `/api/tenants/usage` | Dog usage stats | Protected | `tenant_handler_test.go` |
| GET | `/api/billing/subscription` | Current subscription | Protected | `billing_handler_test.go` |
| POST | `/api/billing/checkout` | Create Stripe checkout | Protected | `billing_handler_test.go` |
| POST | `/api/billing/portal` | Stripe customer portal | Protected | `billing_handler_test.go` |
| POST | `/api/billing/webhook` | Stripe webhooks | Public | `billing_handler_test.go` |
| GET | `/api/billing/invoices` | Invoice list | Protected | `billing_handler_test.go` |

### Usage Endpoint Response

```json
{
  "dogs_used": 7,
  "dogs_limit": 10,
  "dogs_remaining": 3,
  "plan": "free",
  "can_add_dogs": true
}
```

---

## File Changes

### New Files (Create)

| File | Purpose | Test File |
|------|---------|-----------|
| `internal/database/022_add_max_dogs.go` | Migration | - |
| `internal/database/023_add_subscriptions.go` | Migration | - |
| `internal/models/subscription.go` | Models | `subscription_test.go` |
| `internal/repository/subscription_repository.go` | CRUD | `subscription_repository_test.go` |
| `internal/services/stripe_service.go` | Stripe API | `stripe_service_test.go` |
| `internal/handlers/billing_handler.go` | Endpoints | `billing_handler_test.go` |
| `internal/static/frontend/billing.html` | UI | Jest |
| `internal/static/frontend/js/billing.js` | Logic | Jest |

### Existing Files (Modify)

| File | Changes | Test Coverage |
|------|---------|---------------|
| `internal/repository/dog_repository.go` | Add `CountByTenant()` | `dog_repository_test.go` |
| `internal/handlers/dog_handler.go` | Fix TenantID, add limit | `dog_handler_test.go` |
| `internal/models/tenant.go` | Add `MaxDogs` field | `tenant_test.go` |
| `internal/repository/tenant_repository.go` | Add `GetMaxDogs()` | `tenant_repository_test.go` |
| `cmd/server/main.go` | Register billing routes | - |
| `internal/config/config.go` | Add Stripe config | - |

---

## Stripe Setup

### 1. Create Account
1. Go to https://dashboard.stripe.com/register
2. Complete business verification
3. Enable test mode

### 2. Create Product & Prices
**Product:** Gassigeher Pro
- Monthly: €29.00/month
- Yearly: €290.00/year (2 months free)

### 3. Environment Variables
```bash
STRIPE_SECRET_KEY=sk_test_xxx
STRIPE_PUBLISHABLE_KEY=pk_test_xxx
STRIPE_WEBHOOK_SECRET=whsec_xxx
STRIPE_PRICE_MONTHLY=price_xxx
STRIPE_PRICE_YEARLY=price_xxx
```

### 4. Webhook Events
- `checkout.session.completed`
- `invoice.paid`
- `invoice.payment_failed`
- `customer.subscription.updated`
- `customer.subscription.deleted`

---

## Testing Strategy

### Go Tests

```bash
# Run all tests
go test ./... -v

# Run specific package
go test ./internal/repository/... -v

# Run specific test
go test ./internal/handlers/... -run TestDogHandler_CreateDog_EnforcesLimit -v

# With coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Build and test (recommended)
./bat.sh
```

### Jest Tests (Frontend)

```bash
# Run Jest tests
npm test

# Run specific test file
npm test -- billing.test.js

# Watch mode
npm test -- --watch
```

### TDD Cycle

```
1. RED:   Write failing test
2. GREEN: Write minimal code to pass
3. REFACTOR: Clean up code
4. VERIFY: ./bat.sh
5. REPEAT
```

---

## Implementation Checklist

### Phase 1: Dog Limit Foundation
- [ ] RED: Write `TestDogRepository_CountByTenant`
- [ ] GREEN: Implement `CountByTenant()`
- [ ] RED: Write `TestDogHandler_CreateDog_SetsTenantID`
- [ ] GREEN: Fix `CreateDog` to set TenantID
- [ ] RED: Write `TestDogHandler_CreateDog_EnforcesLimit`
- [ ] GREEN: Implement limit check
- [ ] Create migration `022_add_max_dogs.go`
- [ ] Run `./bat.sh` - all tests pass

### Phase 2: Subscription Model
- [ ] RED: Write `TestPricingPlan_Validate`
- [ ] GREEN: Implement `PricingPlan` model
- [ ] RED: Write `TestTenantSubscription_IsActive`
- [ ] GREEN: Implement `TenantSubscription` model
- [ ] RED: Write `TestSubscriptionRepository_*`
- [ ] GREEN: Implement subscription repository
- [ ] Create migration `023_add_subscriptions.go`
- [ ] Run `./bat.sh` - all tests pass

### Phase 3: Stripe Integration
- [ ] RED: Write `TestStripeService_CreateCheckoutSession`
- [ ] GREEN: Implement Stripe service
- [ ] RED: Write `TestBillingHandler_*`
- [ ] GREEN: Implement billing handler
- [ ] Run `./bat.sh` - all tests pass

### Phase 4-6: Frontend
- [ ] Update landing page
- [ ] Create billing.html
- [ ] Add usage indicators
- [ ] Write Jest tests
- [ ] Run `npm test` - all tests pass

---

## Transparency Messaging

```
┌─────────────────────────────────────────────────────────────┐
│                    GASSIGEHER                               │
│                                                             │
│   Einfach. Ehrlich. Transparent.                           │
│                                                             │
│   Ein Limit. Sonst nichts.                                 │
│   ✓ 10 Hunde kostenlos                                     │
│   ✓ Alle Funktionen inklusive                              │
│   ✓ Keine versteckten Kosten                               │
│   ✓ Keine Zeitbegrenzung                                   │
│                                                             │
│   Lieber selbst hosten?                                    │
│   → github.com/tranmh/gassigeher                           │
│   Unbegrenzte Hunde. Kein Problem.                         │
│                                                             │
│   ┌─────────────┬─────────────┐                            │
│   │    FREE     │     PRO     │                            │
│   ├─────────────┼─────────────┤                            │
│   │  10 Hunde   │ Unbegrenzt  │                            │
│   │ €0/Monat    │ €29/Monat   │                            │
│   │             │ €290/Jahr   │                            │
│   └─────────────┴─────────────┘                            │
│                                                             │
│              [ Kostenlos starten ]                          │
└─────────────────────────────────────────────────────────────┘
```

---

*Document created: December 2024*
*Methodology: Test-Driven Development (TDD)*
*Build: ./bat.sh*
