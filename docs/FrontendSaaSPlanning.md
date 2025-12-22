# SaaS Frontend Implementation Plan

## Overview

Complete the Gassigeher SaaS frontend with marketing pages, legal compliance, and enhanced onboarding.

**Backend Status:** ✅ DONE (billing handler, dog limits, Stripe integration, subscriptions)
**Frontend Status:** 🚧 IN PROGRESS (this plan)

---

## What's DONE (Backend)

- ✅ Billing handler with all endpoints (`internal/handlers/billing_handler.go`)
- ✅ Dog limit enforcement (10 free / unlimited pro)
- ✅ Stripe service integration (`internal/services/stripe_service.go`)
- ✅ Subscription model and repository
- ✅ Tenant self-registration at `/api/tenants/register`
- ✅ billing.html (CSS/HTML complete)
- ✅ privacy.html (DSGVO - GDPR compliant)
- ✅ terms.html (AGB - basic)

---

## What's MISSING (This Plan)

### Phase 1: Legal Pages (German Law - CRITICAL)

| Page | Requirement | Priority |
|------|-------------|----------|
| `impressum.html` | TMG §5 - MANDATORY | 🔴 CRITICAL |
| `widerrufsbelehrung.html` | BGB §312d - Consumer rights | 🔴 CRITICAL |
| `sla.html` | Service Level Agreement | 🟡 HIGH |

**Note on Cookies/localStorage:** No cookie consent banner needed. The application only uses localStorage for essential functionality (JWT authentication, pending booking state). This is exempt under GDPR/ePrivacy as "technically necessary" storage. The privacy.html already correctly documents this.

### Phase 2: Marketing Pages (Main Domain)

| Page | Purpose | Priority |
|------|---------|----------|
| `landing/index.html` | Marketing frontpage for gassigeher.org | 🔴 HIGH |
| `landing/pricing.html` | Public pricing comparison | 🔴 HIGH |
| `landing/features.html` | Feature showcase | 🟡 MEDIUM |
| `landing/about.html` | Company info, mission | 🟡 MEDIUM |
| `landing/contact.html` | Contact form | 🟡 MEDIUM |
| `landing/faq.html` | Common questions | 🟡 MEDIUM |

### Phase 3: Enhanced Registration

| Feature | Description | Priority |
|---------|-------------|----------|
| Plan selection during signup | Show Free vs Pro pricing, Stripe checkout for Pro | 🔴 HIGH |
| Enhanced `landing/register.html` | Add plan toggle, payment flow | 🔴 HIGH |

### Phase 4: Existing Enhancements

| File | Enhancement | Priority |
|------|-------------|----------|
| `billing.html` | Complete JavaScript (loadBillingData, Stripe) | 🟡 MEDIUM |
| `terms.html` | Enhance AGB with payment/liability clauses | 🟡 MEDIUM |
| `i18n/de.json` | Add all new translations | 🟡 MEDIUM |

---

## File Structure

```
internal/static/
├── landing/                      # Main domain (gassigeher.org)
│   ├── index.html               # Marketing frontpage
│   ├── pricing.html             # Public pricing
│   ├── features.html            # Feature showcase
│   ├── about.html               # About us
│   ├── contact.html             # Contact form
│   ├── faq.html                 # FAQ
│   ├── register.html            # Enhanced with plan selection
│   ├── impressum.html           # Legal notice
│   ├── widerrufsbelehrung.html  # Cancellation policy
│   ├── sla.html                 # Service Level Agreement
│   └── assets/
│       ├── css/landing.css      # Landing-specific styles
│       └── js/landing.js        # Landing-specific JS
│
└── frontend/                     # Tenant subdomains (*.gassigeher.org)
    ├── index.html               # Tenant homepage (keep as-is)
    ├── billing.html             # Complete JS
    ├── terms.html               # Enhance
    ├── privacy.html             # Keep as-is ✅
    └── js/
        └── billing.js           # NEW: Billing page logic
```

---

## Phase 1: Legal Pages

### 1.1 Impressum (impressum.html)

**Location:** `internal/static/landing/impressum.html`

**Required by:** German TMG (Telemediengesetz) §5 & §7 - MANDATORY for all commercial websites
**Penalty for missing:** €5,000 - €50,000 fine

**Required Content (with placeholders):**
```
Angaben gemäß § 5 TMG

[FIRMENNAME]
[RECHTSFORM]
[STRASSE HAUSNUMMER]
[PLZ ORT]

Vertreten durch:
[GESCHÄFTSFÜHRER NAME]

Kontakt:
Telefon: [TELEFON]
E-Mail: [EMAIL]

Registereintrag:
Eintragung im Handelsregister
Registergericht: [AMTSGERICHT]
Registernummer: [HRB NUMMER]

Umsatzsteuer-ID:
Umsatzsteuer-Identifikationsnummer gemäß § 27 a UStG:
[UST-IDNR]

Verantwortlich für den Inhalt nach § 55 Abs. 2 RStV:
[NAME]
[ADRESSE]

Streitschlichtung:
Die Europäische Kommission stellt eine Plattform zur
Online-Streitbeilegung (OS) bereit:
https://ec.europa.eu/consumers/odr/

Wir sind nicht bereit oder verpflichtet, an
Streitbeilegungsverfahren vor einer
Verbraucherschlichtungsstelle teilzunehmen.
```

### 1.2 Widerrufsbelehrung (widerrufsbelehrung.html)

**Location:** `internal/static/landing/widerrufsbelehrung.html`

**Required by:** BGB §312d, German PAngV - Required for B2C digital services
**Applies when:** Selling Pro tier subscriptions to consumers

**Content:**
```
Widerrufsbelehrung

Widerrufsrecht
Sie haben das Recht, binnen vierzehn Tagen ohne Angabe von
Gründen diesen Vertrag zu widerrufen.

Die Widerrufsfrist beträgt vierzehn Tage ab dem Tag des
Vertragsabschlusses.

Um Ihr Widerrufsrecht auszuüben, müssen Sie uns
[FIRMENNAME]
[ADRESSE]
[EMAIL]
mittels einer eindeutigen Erklärung (z. B. ein mit der Post
versandter Brief oder E-Mail) über Ihren Entschluss, diesen
Vertrag zu widerrufen, informieren.

Sie können dafür das beigefügte Muster-Widerrufsformular
verwenden, das jedoch nicht vorgeschrieben ist.

Zur Wahrung der Widerrufsfrist reicht es aus, dass Sie die
Mitteilung über die Ausübung des Widerrufsrechts vor Ablauf
der Widerrufsfrist absenden.

Folgen des Widerrufs
Wenn Sie diesen Vertrag widerrufen, haben wir Ihnen alle
Zahlungen, die wir von Ihnen erhalten haben, unverzüglich
und spätestens binnen vierzehn Tagen ab dem Tag zurückzuzahlen,
an dem die Mitteilung über Ihren Widerruf dieses Vertrags
bei uns eingegangen ist. Für diese Rückzahlung verwenden wir
dasselbe Zahlungsmittel, das Sie bei der ursprünglichen
Transaktion eingesetzt haben.

Besondere Hinweise
Ihr Widerrufsrecht erlischt vorzeitig, wenn wir mit der
Ausführung des Vertrags (Bereitstellung des digitalen
Dienstes) begonnen haben, nachdem Sie:
1. ausdrücklich zugestimmt haben, dass wir mit der Ausführung
   des Vertrags vor Ablauf der Widerrufsfrist beginnen, und
2. Ihre Kenntnis davon bestätigt haben, dass Sie durch Ihre
   Zustimmung mit Beginn der Ausführung des Vertrags Ihr
   Widerrufsrecht verlieren.

Muster-Widerrufsformular
(Wenn Sie den Vertrag widerrufen wollen, dann füllen Sie
bitte dieses Formular aus und senden Sie es zurück.)

An [FIRMENNAME], [ADRESSE], [EMAIL]:

Hiermit widerrufe(n) ich/wir (*) den von mir/uns (*)
abgeschlossenen Vertrag über den Kauf der folgenden Waren (*)/
die Erbringung der folgenden Dienstleistung (*)

Bestellt am (*)/erhalten am (*):
Name des/der Verbraucher(s):
Anschrift des/der Verbraucher(s):
Unterschrift des/der Verbraucher(s) (nur bei Mitteilung auf Papier):
Datum:

(*) Unzutreffendes streichen.
```

### 1.3 SLA (sla.html)

**Location:** `internal/static/landing/sla.html`

**Content Sections:**

```markdown
# Service Level Agreement (SLA)

Gültig ab: [DATUM]
Version: 1.0

## 1. Verfügbarkeit

### 1.1 Verfügbarkeitsziel
Gassigeher strebt eine Systemverfügbarkeit von 99,5% pro Kalendermonat an.

### 1.2 Berechnung
Verfügbarkeit = (Gesamtminuten - Ausfallminuten) / Gesamtminuten × 100

### 1.3 Ausnahmen
Folgende Zeiten zählen nicht als Ausfallzeit:
- Geplante Wartungsfenster (siehe §3)
- Höhere Gewalt (Naturkatastrophen, Krieg, etc.)
- Probleme außerhalb unserer Kontrolle (DNS, Internet-Backbone)
- Kundenverursachte Ausfälle

## 2. Support-Reaktionszeiten

| Plan | Erstreaktion | Kritisch | Standard |
|------|--------------|----------|----------|
| Free | 48 Stunden | 24 Stunden | 48 Stunden |
| Pro  | 4 Stunden | 4 Stunden | 24 Stunden |

### 2.1 Prioritätsstufen
- **Kritisch:** System nicht nutzbar, keine Workaround möglich
- **Standard:** Eingeschränkte Funktionalität, Workaround verfügbar

### 2.2 Support-Kanäle
- E-Mail: support@gassigeher.org
- Antwortzeiten gelten während der Geschäftszeiten (Mo-Fr, 9-17 Uhr MEZ)

## 3. Wartungsfenster

### 3.1 Geplante Wartung
- **Zeitfenster:** Sonntag, 02:00 - 06:00 Uhr MEZ
- **Ankündigung:** Mindestens 48 Stunden im Voraus per E-Mail
- **Häufigkeit:** Maximal 1x pro Woche

### 3.2 Notfallwartung
Bei kritischen Sicherheitsproblemen kann ohne Vorankündigung
gewartet werden. Kunden werden schnellstmöglich informiert.

## 4. Datensicherung

### 4.1 Backup-Frequenz
- Tägliche automatische Backups
- Aufbewahrung: 30 Tage

### 4.2 Datenwiederherstellung
- Pro-Kunden: Kostenlose Wiederherstellung auf Anfrage
- Free-Kunden: Wiederherstellung gegen Gebühr (€50)

### 4.3 Datenexport
Kunden können ihre Daten jederzeit über die API exportieren.

## 5. Haftungsausschluss

### 5.1 Haftungsbegrenzung
Die maximale Haftung ist auf die im letzten Monat gezahlten
Gebühren begrenzt.

### 5.2 Ausschlüsse
Keine Haftung für:
- Indirekte Schäden oder entgangenen Gewinn
- Datenverlust durch Kundenverschulden
- Ausfälle durch Drittanbieter

## 6. Eskalationsprozess

### Stufe 1: Support-Team
E-Mail an support@gassigeher.org

### Stufe 2: Technische Leitung
Wenn Stufe 1 nicht innerhalb der SLA-Zeit reagiert

### Stufe 3: Geschäftsführung
Für geschäftskritische Eskalationen

## 7. SLA-Änderungen

Änderungen werden 30 Tage im Voraus angekündigt.
Wesentliche Verschlechterungen berechtigen zur außerordentlichen
Kündigung.
```

---

## Phase 2: Marketing Pages

### 2.1 Landing Page (landing/index.html)

**Hero Section:**
```
Gassigeher
Die Buchungsplattform für Tierheime

Ehrenamtliche Gassigeher koordinieren.
Hunde glücklich machen.
Zeit sparen.

[Kostenlos starten] [Preise ansehen]
```

**Features Section (6 items):**
- Einfache Buchung (Calendar icon) - Kalenderansicht mit Zeitslots
- Farbsystem für Erfahrungslevel (Shield icon) - Sichere Hundevergabe
- Automatische Benachrichtigungen (Bell icon) - 18 E-Mail-Typen
- Multi-Tenant SaaS (Cloud icon) - Jedes Tierheim eigene Instanz
- DSGVO-konform (Lock icon) - Datenschutz made in Germany
- Open Source Option (GitHub icon) - Selbst hosten möglich

**Social Proof:**
```
"Entwickelt in Zusammenarbeit mit dem Tierheim Göppingen"

[Logo placeholder] [Testimonial placeholder]
```

**CTA Section:**
```
Kostenlos starten
10 Hunde inklusive. Alle Funktionen. Keine Kreditkarte nötig.

[Jetzt registrieren]

Lieber selbst hosten?
Gassigeher ist Open Source. github.com/tranmh/gassigeher
```

**Footer:**
- Links: Preise, Features, FAQ, Kontakt
- Legal: Impressum, Datenschutz, AGB, Widerrufsrecht, SLA
- Social: GitHub

### 2.2 Pricing Page (landing/pricing.html)

**Header:**
```
Unsere Preise
Transparent. Einfach. Ohne versteckte Kosten.
```

**Billing Cycle Toggle:**
```
[Monatlich] [Jährlich - 2 Monate gratis]
```

**Pricing Cards:**
```
┌─────────────────────────┐  ┌─────────────────────────┐
│          FREE           │  │        PRO ⭐           │
│                         │  │       Empfohlen         │
├─────────────────────────┤  ├─────────────────────────┤
│        €0               │  │      €29/Monat          │
│      pro Monat          │  │   oder €290/Jahr        │
│                         │  │   (2 Monate gratis)     │
├─────────────────────────┤  ├─────────────────────────┤
│ ✓ Bis zu 10 Hunde       │  │ ✓ Unbegrenzte Hunde     │
│ ✓ Alle Funktionen       │  │ ✓ Alle Funktionen       │
│ ✓ E-Mail-Support        │  │ ✓ Prioritäts-Support    │
│ ✓ DSGVO-konform         │  │ ✓ DSGVO-konform         │
│ ✓ Für immer kostenlos   │  │ ✓ Erweiterte Statistiken│
│                         │  │ ✓ SLA mit 99,5% Uptime  │
├─────────────────────────┤  ├─────────────────────────┤
│   [Kostenlos starten]   │  │    [Pro wählen]         │
└─────────────────────────┘  └─────────────────────────┘
```

**FAQ Section:**
```
Häufige Fragen zu Preisen

❓ Was passiert, wenn ich mehr als 10 Hunde habe?
→ Sie können weiterhin alle bestehenden Hunde verwalten,
  aber keine neuen hinzufügen. Upgraden Sie auf Pro für
  unbegrenzte Hunde.

❓ Kann ich jederzeit kündigen?
→ Ja, monatlich kündbar. Bei Jahreszahlung wird der
  Restbetrag anteilig erstattet.

❓ Gibt es eine Testphase?
→ Der Free-Plan ist Ihre Testphase - unbegrenzt und mit
  allen Funktionen. Upgraden Sie nur, wenn Sie mehr als
  10 Hunde brauchen.

❓ Welche Zahlungsmethoden akzeptieren Sie?
→ Kreditkarte, SEPA-Lastschrift über unseren Partner Stripe.

❓ Erhalte ich eine Rechnung?
→ Ja, automatisch per E-Mail nach jeder Zahlung.
  Deutsche Rechnungen mit ausgewiesener MwSt.
```

**Trust Badges:**
```
🔒 Sichere Zahlung via Stripe
🇩🇪 Daten in Deutschland
📜 DSGVO-konform
```

### 2.3 Features Page (landing/features.html)

**Sections:**

1. **Buchungssystem**
   - Kalenderansicht mit Drag & Drop
   - Konfigurierbare Zeitslots (Morgen, Mittag, Nachmittag)
   - Automatische Doppelbuchungs-Prüfung
   - Genehmigungs-Workflow für bestimmte Zeiten
   - Geblockte Termine (Feiertage, Fütterungszeiten)

2. **Hundeverwaltung**
   - Hundeprofile mit Fotos
   - Farbkategorien (Grün/Orange/Blau)
   - Verfügbarkeitsstatus
   - Featured Dogs auf der Startseite
   - Externe Links (Vermittlungsseite)

3. **Benutzerverwaltung**
   - Selbstregistrierung mit Registrierungscode
   - Erfahrungslevel-System
   - Level-Aufstiegs-Anfragen
   - Automatische Deaktivierung bei Inaktivität
   - DSGVO-konforme Kontolöschung

4. **Benachrichtigungen**
   - 18 verschiedene E-Mail-Typen
   - Buchungsbestätigung
   - Erinnerung vor dem Spaziergang
   - Genehmigungs-Benachrichtigungen
   - Kontostatus-Updates

5. **Admin-Dashboard**
   - Echtzeit-Statistiken
   - Buchungsübersicht
   - Benutzer-Management
   - Systemeinstellungen
   - Feiertags-Verwaltung

6. **Sicherheit & Datenschutz**
   - DSGVO-konform
   - Verschlüsselte Passwörter (Argon2id)
   - JWT-Authentifizierung
   - Sicherheits-Header
   - Anonymisierung bei Löschung

### 2.4 About Page (landing/about.html)

**Content:**
```
Über Gassigeher

Unsere Mission
Wir helfen Tierheimen, ihre ehrenamtlichen Gassigeher
effizienter zu koordinieren - damit mehr Zeit für die
Tiere bleibt.

Die Geschichte
Gassigeher wurde in Zusammenarbeit mit dem Tierheim
Göppingen entwickelt. Aus der täglichen Praxis eines
echten Tierheims entstand eine Software, die genau die
Probleme löst, die Tierheime wirklich haben.

Open Source
Gassigeher ist Open Source. Tierheime, die ihre eigene
Infrastruktur betreiben möchten, können den Code kostenlos
nutzen und selbst hosten.

→ github.com/tranmh/gassigeher

Das Team
[PLACEHOLDER - Team member photos/bios]

Kontakt
E-Mail: info@gassigeher.org
```

### 2.5 Contact Page (landing/contact.html)

**Form Fields:**
```html
Kontakt

Haben Sie Fragen? Wir helfen gerne!

Name: [________________]
E-Mail: [________________]
Betreff: [Dropdown: Allgemein | Support | Vertrieb | Presse | Partnerschaft]
Nachricht: [________________
            ________________
            ________________]

[Nachricht senden]

Alternative Kontaktwege:
📧 E-Mail: info@gassigeher.org
📍 Adresse: [PLACEHOLDER]
```

**Backend:** `POST /api/contact` → sends email to configured CONTACT_EMAIL

### 2.6 FAQ Page (landing/faq.html)

**Categories with Accordion:**

```
1. Allgemein
   ├─ Was ist Gassigeher?
   ├─ Für wen ist die Software gedacht?
   ├─ Ist Gassigeher kostenlos?
   └─ Kann ich Gassigeher selbst hosten?

2. Preise & Abrechnung
   ├─ Was kostet Gassigeher?
   ├─ Welche Zahlungsmethoden werden akzeptiert?
   ├─ Kann ich jederzeit kündigen?
   ├─ Was passiert mit meinen Daten nach Kündigung?
   └─ Erhalte ich eine Rechnung?

3. Funktionen
   ├─ Was sind Farbkategorien?
   ├─ Wie funktioniert die Buchung?
   ├─ Wie viele Hunde kann ich anlegen?
   ├─ Kann ich mehrere Administratoren haben?
   └─ Werden Erinnerungs-E-Mails verschickt?

4. Datenschutz & Sicherheit
   ├─ Ist Gassigeher DSGVO-konform?
   ├─ Wo werden meine Daten gespeichert?
   ├─ Wie werden Passwörter geschützt?
   ├─ Kann ich meine Daten exportieren?
   └─ Was passiert bei Kontolöschung?

5. Technisch
   ├─ Welche Browser werden unterstützt?
   ├─ Gibt es eine mobile App?
   ├─ Welche Datenbanken werden unterstützt?
   └─ Wie erhalte ich technischen Support?
```

---

## Phase 3: Enhanced Registration

### 3.1 Plan Selection in Registration

**Modify:** `internal/static/landing/register.html`

**Multi-Step Form:**
```
┌──────────────────────────────────────────────────────────┐
│  Schritt 1 von 4: Plan wählen                            │
│  ○────○────○────○                                        │
└──────────────────────────────────────────────────────────┘

┌─────────────────────┐  ┌─────────────────────┐
│        FREE         │  │        PRO          │
│      €0/Monat       │  │     €29/Monat       │
│     10 Hunde        │  │  Unbegrenzte Hunde  │
│                     │  │                     │
│  ○ Auswählen        │  │  ○ Auswählen        │
└─────────────────────┘  └─────────────────────┘

Bei Pro: Billing Cycle Auswahl
[Monatlich €29] [Jährlich €290 - spare €58]

[Weiter →]
```

```
┌──────────────────────────────────────────────────────────┐
│  Schritt 2 von 4: Tierheim-Daten                         │
│  ●────○────○────○                                        │
└──────────────────────────────────────────────────────────┘

Tierheim-Name: [________________]
Subdomain: [________].gassigeher.org
           ✓ Verfügbar

Kontakt-E-Mail: [________________]
Telefon (optional): [________________]
Stadt: [________________]
PLZ: [________________]
Bundesland: [Dropdown: BW, BY, BE, ...]

[← Zurück] [Weiter →]
```

```
┌──────────────────────────────────────────────────────────┐
│  Schritt 3 von 4: Administrator-Konto                    │
│  ●────●────○────○                                        │
└──────────────────────────────────────────────────────────┘

Vorname: [________________]
Nachname: [________________]
E-Mail: [________________]
Passwort: [________________]
Passwort bestätigen: [________________]

☐ Ich akzeptiere die AGB und Datenschutzerklärung
☐ Ich habe die Widerrufsbelehrung zur Kenntnis genommen

[← Zurück] [Weiter →]
```

```
┌──────────────────────────────────────────────────────────┐
│  Schritt 4 von 4: Zahlung (nur bei Pro)                  │
│  ●────●────●────○                                        │
└──────────────────────────────────────────────────────────┘

Ausgewählter Plan: Pro (Jährlich)
Preis: €290,00 / Jahr

┌────────────────────────────────────────┐
│  Stripe Elements Card Input            │
│  [Card Number]                         │
│  [MM/YY] [CVC]                         │
└────────────────────────────────────────┘

🔒 Sichere Zahlung via Stripe

[← Zurück] [Kostenpflichtig bestellen]
```

### 3.2 Backend Enhancement

**Modify:** `internal/handlers/tenant_handler.go`

```go
type TenantRegistrationRequest struct {
    // Existing fields...
    OrganizationName string `json:"organization_name"`
    Slug             string `json:"slug"`
    ContactEmail     string `json:"contact_email"`
    // ...

    // NEW: Plan selection
    PlanSlug        string `json:"plan_slug"`         // "free" or "pro"
    BillingCycle    string `json:"billing_cycle"`     // "monthly" or "yearly" (only for pro)
    PaymentMethodID string `json:"payment_method_id"` // Stripe PaymentMethod ID (only for pro)
}
```

**Registration Flow for Pro:**
1. Validate all form fields
2. Validate plan_slug ("free" or "pro")
3. If Pro:
   a. Validate billing_cycle ("monthly" or "yearly")
   b. Validate payment_method_id exists
   c. Create Stripe Customer with tenant metadata
   d. Create Stripe Subscription with payment method
   e. On payment failure: Return error, don't create tenant
4. Create tenant (in transaction)
5. Create tenant_subscription record
6. Provision default data
7. Send welcome email with login URL

**New Endpoint:**
```
POST /api/tenants/register
Body: TenantRegistrationRequest (with plan fields)
```

---

## Phase 4: Existing Enhancements

### 4.1 billing.js Completion

**Location:** `internal/static/frontend/js/billing.js`

**Functions to implement:**
```javascript
/**
 * Load all billing data on page load
 */
async function loadBillingData() {
    try {
        showLoading();

        const [subscriptionRes, usageRes, plansRes] = await Promise.all([
            api.get('/billing/subscription'),
            api.get('/billing/usage'),
            api.get('/billing/plans')
        ]);

        currentSubscription = subscriptionRes.subscription;
        currentPlan = subscriptionRes.plan;
        usage = usageRes;
        plans = plansRes.plans;
        stripeConfigured = plansRes.stripe_configured;

        renderCurrentPlan();
        renderUsageBar();
        renderPlanCards();
        renderSubscriptionDetails();

    } catch (error) {
        showAlert('Fehler beim Laden der Abrechnungsdaten', 'error');
        console.error(error);
    } finally {
        hideLoading();
    }
}

/**
 * Render usage bar (dogs used / dogs limit)
 */
function renderUsageBar() {
    const percentage = usage.dogs_limit === -1
        ? 0
        : Math.min(100, (usage.dogs_used / usage.dogs_limit) * 100);

    const usageBar = document.getElementById('usage-bar-fill');
    usageBar.style.width = `${percentage}%`;

    // Add warning/danger classes
    usageBar.classList.remove('warning', 'danger');
    if (percentage >= 90) usageBar.classList.add('danger');
    else if (percentage >= 70) usageBar.classList.add('warning');

    // Update text
    document.getElementById('usage-text').textContent =
        usage.dogs_limit === -1
            ? `${usage.dogs_used} Hunde (Unbegrenzt)`
            : `${usage.dogs_used} / ${usage.dogs_limit} Hunde`;
}

/**
 * Handle upgrade to Pro
 */
async function handleUpgrade(billingCycle) {
    if (!stripeConfigured) {
        showAlert('Zahlungssystem nicht verfügbar', 'error');
        return;
    }

    try {
        const response = await api.post('/billing/checkout', {
            plan_slug: 'pro',
            billing_cycle: billingCycle
        });

        // Redirect to Stripe Checkout
        window.location.href = response.checkout_url;

    } catch (error) {
        showAlert('Fehler beim Erstellen der Checkout-Session', 'error');
        console.error(error);
    }
}

/**
 * Handle subscription cancellation
 */
async function handleCancel() {
    const confirmed = confirm(
        'Möchten Sie Ihr Abonnement wirklich kündigen?\n\n' +
        'Ihr Zugang bleibt bis zum Ende des Abrechnungszeitraums aktiv. ' +
        'Danach wird Ihr Konto auf den Free-Plan (10 Hunde) umgestellt.'
    );

    if (!confirmed) return;

    try {
        await api.post('/billing/cancel');
        showAlert('Abonnement erfolgreich gekündigt', 'success');
        setTimeout(() => location.reload(), 2000);

    } catch (error) {
        showAlert('Fehler beim Kündigen des Abonnements', 'error');
        console.error(error);
    }
}

/**
 * Open Stripe Billing Portal
 */
async function openBillingPortal() {
    try {
        const response = await api.post('/billing/portal');
        window.location.href = response.portal_url;

    } catch (error) {
        showAlert('Fehler beim Öffnen des Kundenportals', 'error');
        console.error(error);
    }
}

/**
 * Escape HTML to prevent XSS
 */
function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}
```

### 4.2 terms.html Enhancement

**Add sections:**
```
§ 7. Zahlungsbedingungen

7.1 Der Preis für den Pro-Plan beträgt €29 pro Monat oder
€290 pro Jahr (entspricht €24,17/Monat).

7.2 Die Zahlung erfolgt im Voraus per Kreditkarte oder
SEPA-Lastschrift über unseren Zahlungsdienstleister Stripe.

7.3 Bei Jahreszahlung wird der Gesamtbetrag zu Beginn des
Abrechnungszeitraums fällig.

7.4 Rechnungen werden automatisch per E-Mail zugestellt.

§ 8. Preisänderungen

8.1 Preisänderungen werden mindestens 30 Tage im Voraus
per E-Mail angekündigt.

8.2 Bei Preiserhöhungen haben Sie ein außerordentliches
Kündigungsrecht zum Zeitpunkt des Inkrafttretens.

8.3 Bestehende Jahresabonnements sind bis zum Ende des
bezahlten Zeitraums von Preisänderungen ausgenommen.

§ 9. Haftungsbeschränkung

9.1 Wir haften unbeschränkt für Vorsatz und grobe
Fahrlässigkeit.

9.2 Bei leichter Fahrlässigkeit haften wir nur bei
Verletzung wesentlicher Vertragspflichten, begrenzt auf
den vorhersehbaren, typischen Schaden.

9.3 Die maximale Haftung ist auf die im letzten Jahr
gezahlten Gebühren begrenzt.

9.4 Die vorstehenden Beschränkungen gelten nicht für
Schäden an Leben, Körper oder Gesundheit.

§ 10. Streitbeilegung

10.1 Wir sind nicht bereit oder verpflichtet, an
Streitbeilegungsverfahren vor einer
Verbraucherschlichtungsstelle teilzunehmen.

10.2 Die Europäische Kommission stellt eine Plattform
zur Online-Streitbeilegung (OS) bereit:
https://ec.europa.eu/consumers/odr/

§ 11. Gerichtsstand

11.1 Für Streitigkeiten mit Verbrauchern gilt der
gesetzliche Gerichtsstand.

11.2 Für Streitigkeiten mit Unternehmern ist
[ORT] ausschließlicher Gerichtsstand.

§ 12. Schlussbestimmungen

12.1 Es gilt das Recht der Bundesrepublik Deutschland
unter Ausschluss des UN-Kaufrechts.

12.2 Sollten einzelne Bestimmungen unwirksam sein,
bleibt die Wirksamkeit der übrigen Bestimmungen
unberührt.
```

### 4.3 Translations (de.json)

**Add keys:**
```json
{
  "legal": {
    "impressum": "Impressum",
    "widerrufsrecht": "Widerrufsrecht",
    "sla": "Service Level Agreement",
    "agb": "AGB",
    "datenschutz": "Datenschutz"
  },
  "landing": {
    "hero_title": "Die Buchungsplattform für Tierheime",
    "hero_subtitle": "Ehrenamtliche koordinieren. Hunde glücklich machen. Zeit sparen.",
    "cta_free": "Kostenlos starten",
    "cta_pricing": "Preise ansehen",
    "features_title": "Funktionen",
    "social_proof": "Entwickelt mit dem Tierheim Göppingen",
    "open_source": "Open Source verfügbar"
  },
  "pricing": {
    "title": "Unsere Preise",
    "subtitle": "Transparent. Einfach. Ohne versteckte Kosten.",
    "monthly": "Monatlich",
    "yearly": "Jährlich",
    "save_months": "2 Monate gratis",
    "free_title": "Free",
    "pro_title": "Pro",
    "recommended": "Empfohlen",
    "per_month": "/Monat",
    "per_year": "/Jahr",
    "dogs_limit": "Bis zu {count} Hunde",
    "unlimited_dogs": "Unbegrenzte Hunde",
    "all_features": "Alle Funktionen",
    "email_support": "E-Mail-Support",
    "priority_support": "Prioritäts-Support",
    "gdpr_compliant": "DSGVO-konform",
    "forever_free": "Für immer kostenlos",
    "extended_stats": "Erweiterte Statistiken",
    "sla_uptime": "SLA mit 99,5% Uptime",
    "start_free": "Kostenlos starten",
    "choose_pro": "Pro wählen",
    "faq_title": "Häufige Fragen zu Preisen"
  },
  "features": {
    "booking_title": "Buchungssystem",
    "booking_desc": "Kalenderansicht, Zeitslots, automatische Genehmigungen",
    "dogs_title": "Hundeverwaltung",
    "dogs_desc": "Profile, Fotos, Farbkategorien, Verfügbarkeit",
    "users_title": "Benutzerverwaltung",
    "users_desc": "Registrierung, Erfahrungslevel, Deaktivierung",
    "notifications_title": "Benachrichtigungen",
    "notifications_desc": "18 verschiedene E-Mail-Typen automatisch",
    "admin_title": "Admin-Dashboard",
    "admin_desc": "Statistiken, Übersichten, Einstellungen",
    "security_title": "Sicherheit",
    "security_desc": "DSGVO-konform, verschlüsselt, sicher"
  },
  "faq": {
    "title": "Häufig gestellte Fragen",
    "general": "Allgemein",
    "pricing": "Preise & Abrechnung",
    "features": "Funktionen",
    "security": "Datenschutz & Sicherheit",
    "technical": "Technisch"
  },
  "contact": {
    "title": "Kontakt",
    "subtitle": "Haben Sie Fragen? Wir helfen gerne!",
    "name": "Name",
    "email": "E-Mail",
    "subject": "Betreff",
    "subject_general": "Allgemeine Anfrage",
    "subject_support": "Technischer Support",
    "subject_sales": "Vertrieb",
    "subject_press": "Presse",
    "subject_partnership": "Partnerschaft",
    "message": "Nachricht",
    "send": "Nachricht senden",
    "success": "Nachricht erfolgreich gesendet!",
    "error": "Fehler beim Senden. Bitte versuchen Sie es erneut."
  },
  "registration": {
    "step_plan": "Plan wählen",
    "step_organization": "Tierheim-Daten",
    "step_admin": "Administrator",
    "step_payment": "Zahlung",
    "select_plan": "Plan auswählen",
    "billing_monthly": "Monatlich",
    "billing_yearly": "Jährlich",
    "save_amount": "Spare €{amount}",
    "subdomain_available": "Verfügbar",
    "subdomain_taken": "Bereits vergeben",
    "accept_terms": "Ich akzeptiere die AGB und Datenschutzerklärung",
    "accept_withdrawal": "Ich habe die Widerrufsbelehrung zur Kenntnis genommen",
    "secure_payment": "Sichere Zahlung via Stripe",
    "submit_free": "Kostenlos registrieren",
    "submit_paid": "Kostenpflichtig bestellen"
  },
  "billing": {
    "title": "Abrechnung",
    "current_plan": "Aktueller Plan",
    "usage": "Nutzung",
    "dogs_used": "Hunde verwendet",
    "unlimited": "Unbegrenzt",
    "subscription_status": "Status",
    "status_active": "Aktiv",
    "status_cancelled": "Gekündigt",
    "status_past_due": "Zahlung ausstehend",
    "next_billing": "Nächste Abrechnung",
    "cancel_subscription": "Abonnement kündigen",
    "manage_payment": "Zahlungsmethode verwalten",
    "upgrade": "Upgrade auf Pro",
    "cancel_confirm": "Möchten Sie Ihr Abonnement wirklich kündigen?"
  }
}
```

---

## Routing Configuration

### Main Domain Routes (gassigeher.org)

**Modify:** `cmd/server/main.go`

```go
// Determine if request is for main domain or tenant subdomain
func isMainDomain(host string) bool {
    baseDomain := os.Getenv("BASE_DOMAIN") // e.g., "gassigeher.org"
    return host == baseDomain || host == "www."+baseDomain
}

// In main():
// Landing pages served for main domain
landingFS := http.FileServer(http.FS(landingFiles))

router.PathPrefix("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    if isMainDomain(r.Host) {
        landingFS.ServeHTTP(w, r)
    } else {
        // Tenant subdomain - serve frontend files
        frontendFS.ServeHTTP(w, r)
    }
}))

// Public landing API endpoints
router.HandleFunc("/api/contact", contactHandler.Submit).Methods("POST")
router.HandleFunc("/api/tenants/register", tenantHandler.Register).Methods("POST")
router.HandleFunc("/api/tenants/check-slug", tenantHandler.CheckSlug).Methods("GET")
```

### Tenant Subdomain Routes (*.gassigeher.org)

Keep existing routing - no changes needed. Each tenant subdomain serves files from `frontend/`.

---

## Implementation Order

### Sprint 1: Legal Compliance (CRITICAL)
1. [ ] Create `landing/impressum.html` with placeholders
2. [ ] Create `landing/widerrufsbelehrung.html`
3. [ ] Create `landing/sla.html`
4. [ ] Enhance `frontend/terms.html` with payment clauses
5. [ ] Add legal translations to `de.json`

### Sprint 2: Marketing Frontpage
1. [ ] Create `landing/index.html` (marketing homepage)
2. [ ] Create `landing/pricing.html`
3. [ ] Create `landing/assets/css/landing.css`
4. [ ] Create `landing/assets/js/landing.js`
5. [ ] Configure main domain routing in `main.go`

### Sprint 3: Supporting Marketing Pages
1. [ ] Create `landing/features.html`
2. [ ] Create `landing/about.html`
3. [ ] Create `landing/contact.html`
4. [ ] Create contact handler backend endpoint
5. [ ] Create `landing/faq.html`

### Sprint 4: Enhanced Registration
1. [ ] Enhance `landing/register.html` with multi-step form
2. [ ] Add plan selection step
3. [ ] Integrate Stripe Elements for Pro payment
4. [ ] Modify `tenant_handler.go` for Pro registration with Stripe
5. [ ] Test full Pro registration flow end-to-end

### Sprint 5: Billing Completion
1. [ ] Create `frontend/js/billing.js`
2. [ ] Wire up billing.html to billing.js
3. [ ] Test upgrade/downgrade flows
4. [ ] Test Stripe portal integration
5. [ ] Add remaining translations to `de.json`

---

## Files to Create (New)

| File | Lines (est.) | Priority |
|------|--------------|----------|
| `landing/index.html` | ~300 | HIGH |
| `landing/pricing.html` | ~250 | HIGH |
| `landing/features.html` | ~300 | MEDIUM |
| `landing/about.html` | ~150 | MEDIUM |
| `landing/contact.html` | ~150 | MEDIUM |
| `landing/faq.html` | ~350 | MEDIUM |
| `landing/impressum.html` | ~120 | CRITICAL |
| `landing/widerrufsbelehrung.html` | ~150 | CRITICAL |
| `landing/sla.html` | ~200 | HIGH |
| `landing/assets/css/landing.css` | ~500 | HIGH |
| `landing/assets/js/landing.js` | ~250 | HIGH |
| `frontend/js/billing.js` | ~200 | MEDIUM |
| `internal/handlers/contact_handler.go` | ~100 | MEDIUM |

**Total: ~3,020 lines of new code**

## Files to Modify (Existing)

| File | Changes |
|------|---------|
| `landing/register.html` | Add multi-step form with plan selection |
| `frontend/terms.html` | Add payment/liability/jurisdiction sections |
| `frontend/i18n/de.json` | Add ~150 new translation keys |
| `cmd/server/main.go` | Add main domain routing, contact endpoint |
| `internal/handlers/tenant_handler.go` | Add Pro registration with Stripe |

---

## Success Criteria

- [ ] All legal pages accessible and German-law compliant
- [ ] Marketing pages convert visitors to signups
- [ ] Pro plan purchasable during registration
- [ ] Billing page fully functional with upgrade/cancel
- [ ] All text in German with proper translations
- [ ] Mobile-responsive design (tested on iOS/Android)
- [ ] Lighthouse score > 90 for landing pages
- [ ] All forms have proper validation and error messages
- [ ] Stripe integration working in test mode

---

## Notes

### localStorage Usage (No Cookie Consent Needed)
The application uses localStorage for:
- JWT authentication token (`gassigeher_token`)
- Pending booking state (`pendingBooking`)

Both are **technically necessary** for core functionality and are exempt from consent requirements under GDPR/ePrivacy Directive. The privacy.html already correctly documents this usage.

### German Legal Requirements Summary
| Requirement | Status | File |
|-------------|--------|------|
| Impressum (TMG §5) | 🔴 Missing | `impressum.html` |
| Datenschutz (DSGVO) | ✅ Done | `privacy.html` |
| AGB | ⚠️ Needs enhancement | `terms.html` |
| Widerrufsrecht (BGB §312d) | 🔴 Missing | `widerrufsbelehrung.html` |
| SLA | 🔴 Missing | `sla.html` |
| Cookie Consent | ✅ Not needed | N/A |
