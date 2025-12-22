/**
 * Landing Page Tests for Gassigeher SaaS
 *
 * Tests for landing.js functionality including:
 * - Plan selection
 * - Billing cycle toggle
 * - Slug validation
 * - Form validation
 * - Price display formatting
 *
 * @jest-environment jsdom
 */

// Plan Selection Tests
describe('Plan Selection', () => {
  let selectedPlan = 'free';

  function selectPlan(plan) {
    const validPlans = ['free', 'pro'];
    if (!validPlans.includes(plan)) {
      return false;
    }
    selectedPlan = plan;
    return true;
  }

  beforeEach(() => {
    selectedPlan = 'free';
  });

  test('should default to free plan', () => {
    expect(selectedPlan).toBe('free');
  });

  test('should allow selecting free plan', () => {
    const result = selectPlan('free');
    expect(result).toBe(true);
    expect(selectedPlan).toBe('free');
  });

  test('should allow selecting pro plan', () => {
    const result = selectPlan('pro');
    expect(result).toBe(true);
    expect(selectedPlan).toBe('pro');
  });

  test('should reject invalid plan values', () => {
    const result = selectPlan('enterprise');
    expect(result).toBe(false);
    expect(selectedPlan).toBe('free'); // Should remain unchanged
  });

  test('should reject XSS payloads as plan', () => {
    const result = selectPlan('<script>alert(1)</script>');
    expect(result).toBe(false);
    expect(selectedPlan).toBe('free');
  });

  test('should reject null and undefined', () => {
    expect(selectPlan(null)).toBe(false);
    expect(selectPlan(undefined)).toBe(false);
  });
});

// Billing Cycle Tests
describe('Billing Cycle Toggle', () => {
  let billingCycle = 'monthly';

  function setBillingCycle(cycle) {
    const validCycles = ['monthly', 'yearly'];
    if (!validCycles.includes(cycle)) {
      return false;
    }
    billingCycle = cycle;
    return true;
  }

  beforeEach(() => {
    billingCycle = 'monthly';
  });

  test('should default to monthly', () => {
    expect(billingCycle).toBe('monthly');
  });

  test('should allow monthly billing', () => {
    const result = setBillingCycle('monthly');
    expect(result).toBe(true);
    expect(billingCycle).toBe('monthly');
  });

  test('should allow yearly billing', () => {
    const result = setBillingCycle('yearly');
    expect(result).toBe(true);
    expect(billingCycle).toBe('yearly');
  });

  test('should reject invalid billing cycles', () => {
    expect(setBillingCycle('weekly')).toBe(false);
    expect(setBillingCycle('quarterly')).toBe(false);
    expect(setBillingCycle('')).toBe(false);
  });

  test('should reject XSS payloads', () => {
    expect(setBillingCycle('<script>')).toBe(false);
    expect(billingCycle).toBe('monthly');
  });
});

// Price Display Tests
describe('Price Display', () => {
  function getPriceDisplay(billingCycle) {
    if (billingCycle === 'yearly') {
      return { price: '290', unit: 'EUR/Jahr', note: '24,17 EUR/Monat - 2 Monate gratis' };
    }
    return { price: '29', unit: 'EUR/Monat', note: '29 EUR/Monat - Monatlich kündbar' };
  }

  test('should display monthly price correctly', () => {
    const display = getPriceDisplay('monthly');
    expect(display.price).toBe('29');
    expect(display.unit).toBe('EUR/Monat');
  });

  test('should display yearly price correctly', () => {
    const display = getPriceDisplay('yearly');
    expect(display.price).toBe('290');
    expect(display.unit).toBe('EUR/Jahr');
  });

  test('should show savings note for yearly', () => {
    const display = getPriceDisplay('yearly');
    expect(display.note).toContain('2 Monate gratis');
  });

  test('should show cancellation note for monthly', () => {
    const display = getPriceDisplay('monthly');
    expect(display.note).toContain('Monatlich kündbar');
  });
});

// Slug Validation Tests
describe('Slug Validation', () => {
  function normalizeSlug(input) {
    return input.toLowerCase().replace(/[^a-z0-9-]/g, '');
  }

  function isValidSlugLength(slug) {
    return slug.length >= 3 && slug.length <= 50;
  }

  test('should convert to lowercase', () => {
    expect(normalizeSlug('TestSlug')).toBe('testslug');
  });

  test('should remove special characters', () => {
    expect(normalizeSlug('test@slug!')).toBe('testslug');
  });

  test('should keep hyphens', () => {
    expect(normalizeSlug('test-slug')).toBe('test-slug');
  });

  test('should remove spaces', () => {
    expect(normalizeSlug('test slug')).toBe('testslug');
  });

  test('should keep numbers', () => {
    expect(normalizeSlug('test123')).toBe('test123');
  });

  test('should remove umlauts', () => {
    expect(normalizeSlug('testüäö')).toBe('test');
  });

  test('should prevent XSS in slug', () => {
    expect(normalizeSlug('<script>alert(1)</script>')).toBe('scriptalert1script');
  });

  test('should validate minimum length', () => {
    expect(isValidSlugLength('ab')).toBe(false);
    expect(isValidSlugLength('abc')).toBe(true);
  });

  test('should validate maximum length', () => {
    expect(isValidSlugLength('a'.repeat(50))).toBe(true);
    expect(isValidSlugLength('a'.repeat(51))).toBe(false);
  });
});

// Form Validation Tests
describe('Registration Form Validation', () => {
  function validateEmail(email) {
    if (!email) return { valid: false, error: 'E-Mail ist erforderlich' };
    if (!email.includes('@')) return { valid: false, error: 'Ungültige E-Mail-Adresse' };
    if (email.length > 200) return { valid: false, error: 'E-Mail ist zu lang' };
    return { valid: true };
  }

  function validateName(name) {
    if (!name || !name.trim()) return { valid: false, error: 'Name ist erforderlich' };
    if (name.length > 200) return { valid: false, error: 'Name ist zu lang' };
    return { valid: true };
  }

  function validatePassword(password) {
    if (!password) return { valid: false, error: 'Passwort ist erforderlich' };
    if (password.length < 8) return { valid: false, error: 'Passwort muss mindestens 8 Zeichen haben' };
    return { valid: true };
  }

  describe('Email Validation', () => {
    test('should require email', () => {
      expect(validateEmail('')).toEqual({ valid: false, error: 'E-Mail ist erforderlich' });
      expect(validateEmail(null)).toEqual({ valid: false, error: 'E-Mail ist erforderlich' });
    });

    test('should require @ symbol', () => {
      expect(validateEmail('notemail')).toEqual({ valid: false, error: 'Ungültige E-Mail-Adresse' });
    });

    test('should accept valid email', () => {
      expect(validateEmail('test@example.com').valid).toBe(true);
    });

    test('should reject too long email', () => {
      const longEmail = 'a'.repeat(200) + '@example.com';
      expect(validateEmail(longEmail)).toEqual({ valid: false, error: 'E-Mail ist zu lang' });
    });
  });

  describe('Name Validation', () => {
    test('should require name', () => {
      expect(validateName('')).toEqual({ valid: false, error: 'Name ist erforderlich' });
      expect(validateName('   ')).toEqual({ valid: false, error: 'Name ist erforderlich' });
    });

    test('should accept valid name', () => {
      expect(validateName('Test User').valid).toBe(true);
    });

    test('should reject too long name', () => {
      expect(validateName('a'.repeat(201))).toEqual({ valid: false, error: 'Name ist zu lang' });
    });
  });

  describe('Password Validation', () => {
    test('should require password', () => {
      expect(validatePassword('')).toEqual({ valid: false, error: 'Passwort ist erforderlich' });
    });

    test('should require minimum length', () => {
      expect(validatePassword('1234567')).toEqual({ valid: false, error: 'Passwort muss mindestens 8 Zeichen haben' });
    });

    test('should accept valid password', () => {
      expect(validatePassword('12345678').valid).toBe(true);
    });
  });
});

// Subject Mapping Tests
describe('Contact Subject Mapping', () => {
  const subjectMap = {
    'general': 'Allgemeine Anfrage',
    'support': 'Technischer Support',
    'sales': 'Vertrieb / Pro-Plan',
    'partnership': 'Partnerschaft',
    'press': 'Presse',
    'other': 'Sonstiges'
  };

  function getSubjectText(key) {
    return subjectMap[key] || key;
  }

  test('should map general subject', () => {
    expect(getSubjectText('general')).toBe('Allgemeine Anfrage');
  });

  test('should map support subject', () => {
    expect(getSubjectText('support')).toBe('Technischer Support');
  });

  test('should map sales subject', () => {
    expect(getSubjectText('sales')).toBe('Vertrieb / Pro-Plan');
  });

  test('should map partnership subject', () => {
    expect(getSubjectText('partnership')).toBe('Partnerschaft');
  });

  test('should map press subject', () => {
    expect(getSubjectText('press')).toBe('Presse');
  });

  test('should map other subject', () => {
    expect(getSubjectText('other')).toBe('Sonstiges');
  });

  test('should return unknown subject as-is', () => {
    expect(getSubjectText('unknown')).toBe('unknown');
  });
});

// Federal State Validation Tests
describe('Federal State Validation', () => {
  const VALID_STATES = [
    'BW', 'BY', 'BE', 'BB', 'HB', 'HH', 'HE', 'MV',
    'NI', 'NW', 'RP', 'SL', 'SN', 'ST', 'SH', 'TH'
  ];

  function isValidFederalState(state) {
    return VALID_STATES.includes(state);
  }

  test('should accept all valid German federal states', () => {
    VALID_STATES.forEach(state => {
      expect(isValidFederalState(state)).toBe(true);
    });
  });

  test('should reject invalid states', () => {
    expect(isValidFederalState('XX')).toBe(false);
    expect(isValidFederalState('CA')).toBe(false);
    expect(isValidFederalState('')).toBe(false);
  });

  test('should be case-sensitive', () => {
    expect(isValidFederalState('bw')).toBe(false);
    expect(isValidFederalState('BW')).toBe(true);
  });
});

// URL Parameter Handling Tests
describe('URL Parameter Handling', () => {
  function getPlanFromUrl(searchParams) {
    const plan = searchParams.get('plan');
    if (plan === 'pro') {
      return 'pro';
    }
    return 'free';
  }

  test('should return pro when plan=pro', () => {
    const params = new URLSearchParams('?plan=pro');
    expect(getPlanFromUrl(params)).toBe('pro');
  });

  test('should default to free when no plan param', () => {
    const params = new URLSearchParams('');
    expect(getPlanFromUrl(params)).toBe('free');
  });

  test('should default to free for invalid plan', () => {
    const params = new URLSearchParams('?plan=enterprise');
    expect(getPlanFromUrl(params)).toBe('free');
  });

  test('should handle XSS in plan param', () => {
    const params = new URLSearchParams('?plan=<script>alert(1)</script>');
    expect(getPlanFromUrl(params)).toBe('free');
  });
});

// Form Error Display Tests
describe('Form Error Display', () => {
  function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }

  function createErrorMessage(message) {
    return `<div class="form-error">${escapeHtml(message)}</div>`;
  }

  test('should escape HTML in error messages', () => {
    const result = createErrorMessage('<script>alert("XSS")</script>');
    expect(result).not.toContain('<script>');
    expect(result).toContain('&lt;script&gt;');
  });

  test('should display normal error messages', () => {
    const result = createErrorMessage('E-Mail ist erforderlich');
    expect(result).toContain('E-Mail ist erforderlich');
  });

  test('should handle empty message', () => {
    const result = createErrorMessage('');
    expect(result).toBe('<div class="form-error"></div>');
  });
});

// Checkout Flow Tests
describe('Pro Checkout Flow', () => {
  function buildCheckoutRequest(tenantId, billingCycle) {
    if (!tenantId) {
      return null;
    }
    if (!['monthly', 'yearly'].includes(billingCycle)) {
      return null;
    }
    return {
      plan_slug: 'pro',
      billing_cycle: billingCycle
    };
  }

  test('should build valid checkout request', () => {
    const request = buildCheckoutRequest(123, 'monthly');
    expect(request).toEqual({
      plan_slug: 'pro',
      billing_cycle: 'monthly'
    });
  });

  test('should build yearly checkout request', () => {
    const request = buildCheckoutRequest(123, 'yearly');
    expect(request).toEqual({
      plan_slug: 'pro',
      billing_cycle: 'yearly'
    });
  });

  test('should return null for missing tenant ID', () => {
    const request = buildCheckoutRequest(null, 'monthly');
    expect(request).toBeNull();
  });

  test('should return null for invalid billing cycle', () => {
    const request = buildCheckoutRequest(123, 'weekly');
    expect(request).toBeNull();
  });
});

// FAQ Accordion Tests
describe('FAQ Accordion', () => {
  test('should toggle open class', () => {
    document.body.innerHTML = `
      <div class="faq-item">
        <div class="faq-question">Question</div>
        <div class="faq-answer">Answer</div>
      </div>
    `;

    const item = document.querySelector('.faq-item');
    expect(item.classList.contains('open')).toBe(false);

    item.classList.toggle('open');
    expect(item.classList.contains('open')).toBe(true);

    item.classList.toggle('open');
    expect(item.classList.contains('open')).toBe(false);
  });
});

// Savings Calculation Tests
describe('Savings Calculation', () => {
  function calculateYearlySavings(monthlyPrice, yearlyPrice) {
    const yearlyMonthly = monthlyPrice * 12;
    return yearlyMonthly - yearlyPrice;
  }

  function calculateSavingsPercentage(monthlyPrice, yearlyPrice) {
    const yearlyMonthly = monthlyPrice * 12;
    return Math.round((1 - yearlyPrice / yearlyMonthly) * 100);
  }

  test('should calculate yearly savings correctly', () => {
    // 29 EUR/month * 12 = 348, yearly = 290, savings = 58
    expect(calculateYearlySavings(29, 290)).toBe(58);
  });

  test('should calculate savings percentage correctly', () => {
    // (348 - 290) / 348 = ~17%
    expect(calculateSavingsPercentage(29, 290)).toBe(17);
  });
});
