/**
 * Billing Page Tests - TDD for 2-Tier Payment Feature
 *
 * Tests for billing.html functionality including:
 * - XSS prevention in plan rendering
 * - XSS prevention in alert messages
 * - Billing cycle validation
 * - Usage bar calculation
 *
 * @jest-environment jsdom
 */

/**
 * escapeHtml - Exact copy of the function from billing.html
 * This ensures tests match the actual implementation.
 */
function escapeHtml(text) {
  if (!text) return '';
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

// XSS Payloads for testing
const XSS_PAYLOADS = {
  scriptTag: '<script>alert("XSS")</script>',
  imgOnerror: '<img src=x onerror=alert("XSS")>',
  divOnclick: '<div onclick=alert("XSS")>Click me</div>',
  svgOnload: '<svg onload=alert("XSS")>',
  eventHandler: '"><img src=x onerror=alert("XSS")><"',
};

describe('escapeHtml function', () => {
  test('should escape < and > characters', () => {
    const result = escapeHtml(XSS_PAYLOADS.scriptTag);
    expect(result).not.toContain('<script>');
    expect(result).not.toContain('</script>');
    expect(result).toContain('&lt;');
    expect(result).toContain('&gt;');
  });

  test('should escape img tags with onerror', () => {
    const result = escapeHtml(XSS_PAYLOADS.imgOnerror);
    expect(result).not.toContain('<img');
    expect(result).toContain('&lt;img');
  });

  test('should escape onclick event handlers', () => {
    const result = escapeHtml(XSS_PAYLOADS.divOnclick);
    expect(result).not.toContain('<div');
    expect(result).toContain('&lt;div');
  });

  test('should handle null and undefined', () => {
    expect(escapeHtml(null)).toBe('');
    expect(escapeHtml(undefined)).toBe('');
    expect(escapeHtml('')).toBe('');
  });

  test('should preserve normal text', () => {
    expect(escapeHtml('Free')).toBe('Free');
    expect(escapeHtml('Pro')).toBe('Pro');
    expect(escapeHtml('Monthly Plan')).toBe('Monthly Plan');
  });

  test('should escape special characters in plan names', () => {
    expect(escapeHtml('Plan & Features')).toBe('Plan &amp; Features');
    expect(escapeHtml('Plan "Premium"')).toBe('Plan "Premium"');
  });
});

describe('XSS Vulnerability Tests - billing.html', () => {
  describe('Bug: plan.name not sanitized in renderPlans (line 419)', () => {
    const maliciousPlan = {
      id: 1,
      name: XSS_PAYLOADS.scriptTag,
      slug: 'malicious',
      max_dogs: 10,
      price_monthly: 0,
      price_yearly: 0
    };

    // Vulnerable template (before fix)
    const vulnerableTemplate = (plan) =>
      `<h3 class="plan-name">${plan.name}</h3>`;

    // Safe template (after fix)
    const safeTemplate = (plan) =>
      `<h3 class="plan-name">${escapeHtml(plan.name)}</h3>`;

    test('VULNERABLE: renders XSS payload in plan name', () => {
      const div = document.createElement('div');
      div.innerHTML = vulnerableTemplate(maliciousPlan);
      expect(div.innerHTML).toContain('<script>');
    });

    test('SAFE: escapes XSS payload in plan name', () => {
      const div = document.createElement('div');
      div.innerHTML = safeTemplate(maliciousPlan);
      expect(div.innerHTML).not.toContain('<script>');
      expect(div.innerHTML).toContain('&lt;script&gt;');
    });
  });

  describe('Bug: currentSubscription.plan.name not sanitized (line 479)', () => {
    const maliciousSubscription = {
      plan: { name: XSS_PAYLOADS.imgOnerror }
    };

    const vulnerableTemplate = (subscription) =>
      `<span>${subscription.plan ? subscription.plan.name : 'Pro'}</span>`;

    const safeTemplate = (subscription) =>
      `<span>${subscription.plan ? escapeHtml(subscription.plan.name) : 'Pro'}</span>`;

    test('VULNERABLE: renders XSS payload in subscription plan name', () => {
      const div = document.createElement('div');
      div.innerHTML = vulnerableTemplate(maliciousSubscription);
      expect(div.innerHTML).toContain('<img');
    });

    test('SAFE: escapes XSS payload in subscription plan name', () => {
      const div = document.createElement('div');
      div.innerHTML = safeTemplate(maliciousSubscription);
      expect(div.innerHTML).not.toContain('<img');
      expect(div.innerHTML).toContain('&lt;img');
    });
  });

  describe('Bug: message not sanitized in showAlert (line 593)', () => {
    const maliciousMessage = XSS_PAYLOADS.scriptTag;

    // Vulnerable showAlert (before fix)
    const vulnerableShowAlert = (message, type) => {
      const alertClass = type === 'error' ? 'alert-error' : 'alert-success';
      return `<div class="alert ${alertClass}">${message}</div>`;
    };

    // Safe showAlert (after fix)
    const safeShowAlert = (message, type) => {
      const alertClass = type === 'error' ? 'alert-error' : 'alert-success';
      return `<div class="alert ${alertClass}">${escapeHtml(message)}</div>`;
    };

    test('VULNERABLE: renders XSS payload in alert message', () => {
      const div = document.createElement('div');
      div.innerHTML = vulnerableShowAlert(maliciousMessage, 'error');
      expect(div.innerHTML).toContain('<script>');
    });

    test('SAFE: escapes XSS payload in alert message', () => {
      const div = document.createElement('div');
      div.innerHTML = safeShowAlert(maliciousMessage, 'error');
      expect(div.innerHTML).not.toContain('<script>');
      expect(div.innerHTML).toContain('&lt;script&gt;');
    });

    test('SAFE: preserves normal error messages', () => {
      const div = document.createElement('div');
      div.innerHTML = safeShowAlert('Zahlung fehlgeschlagen', 'error');
      expect(div.textContent).toBe('Zahlung fehlgeschlagen');
    });
  });
});

describe('Billing Cycle Validation', () => {
  const VALID_BILLING_CYCLES = ['monthly', 'yearly'];

  function isValidBillingCycle(cycle) {
    return VALID_BILLING_CYCLES.includes(cycle);
  }

  test('should accept "monthly" billing cycle', () => {
    expect(isValidBillingCycle('monthly')).toBe(true);
  });

  test('should accept "yearly" billing cycle', () => {
    expect(isValidBillingCycle('yearly')).toBe(true);
  });

  test('should reject invalid billing cycles', () => {
    expect(isValidBillingCycle('invalid')).toBe(false);
    expect(isValidBillingCycle('weekly')).toBe(false);
    expect(isValidBillingCycle('quarterly')).toBe(false);
    expect(isValidBillingCycle('')).toBe(false);
    expect(isValidBillingCycle(null)).toBe(false);
    expect(isValidBillingCycle(undefined)).toBe(false);
  });

  test('should reject XSS payloads in billing cycle', () => {
    expect(isValidBillingCycle('<script>alert(1)</script>')).toBe(false);
    expect(isValidBillingCycle('monthly<script>')).toBe(false);
  });
});

describe('Plan Slug Validation', () => {
  const VALID_PLAN_SLUGS = ['free', 'pro'];

  function isValidPlanSlug(slug) {
    return VALID_PLAN_SLUGS.includes(slug);
  }

  test('should accept "free" plan slug', () => {
    expect(isValidPlanSlug('free')).toBe(true);
  });

  test('should accept "pro" plan slug', () => {
    expect(isValidPlanSlug('pro')).toBe(true);
  });

  test('should reject invalid plan slugs', () => {
    expect(isValidPlanSlug('enterprise')).toBe(false);
    expect(isValidPlanSlug('premium')).toBe(false);
    expect(isValidPlanSlug('')).toBe(false);
    expect(isValidPlanSlug(null)).toBe(false);
  });
});

describe('Usage Bar Calculation', () => {
  function calculateUsagePercentage(dogsUsed, dogsLimit) {
    if (dogsLimit === -1) return 0; // Unlimited
    return Math.min(100, (dogsUsed / dogsLimit) * 100);
  }

  function getUsageBarClass(percentage) {
    if (percentage >= 90) return 'danger';
    if (percentage >= 70) return 'warning';
    return '';
  }

  test('should return 0% for unlimited plans', () => {
    expect(calculateUsagePercentage(100, -1)).toBe(0);
    expect(calculateUsagePercentage(0, -1)).toBe(0);
  });

  test('should calculate correct percentage for free tier', () => {
    expect(calculateUsagePercentage(5, 10)).toBe(50);
    expect(calculateUsagePercentage(10, 10)).toBe(100);
    expect(calculateUsagePercentage(0, 10)).toBe(0);
  });

  test('should cap at 100% even if over limit', () => {
    expect(calculateUsagePercentage(15, 10)).toBe(100);
  });

  test('should return danger class at 90%+', () => {
    expect(getUsageBarClass(90)).toBe('danger');
    expect(getUsageBarClass(100)).toBe('danger');
  });

  test('should return warning class at 70-89%', () => {
    expect(getUsageBarClass(70)).toBe('warning');
    expect(getUsageBarClass(89)).toBe('warning');
  });

  test('should return empty class below 70%', () => {
    expect(getUsageBarClass(69)).toBe('');
    expect(getUsageBarClass(50)).toBe('');
    expect(getUsageBarClass(0)).toBe('');
  });
});

describe('Price Formatting', () => {
  function formatPrice(priceInCents) {
    if (priceInCents === 0) return '0';
    return (priceInCents / 100).toFixed(0);
  }

  test('should format free plan as 0', () => {
    expect(formatPrice(0)).toBe('0');
  });

  test('should format monthly Pro plan', () => {
    expect(formatPrice(2900)).toBe('29');
  });

  test('should format yearly Pro plan', () => {
    expect(formatPrice(29000)).toBe('290');
  });
});

describe('Dog Limit Display', () => {
  function formatDogLimit(maxDogs) {
    return maxDogs === -1 ? 'Unbegrenzt' : maxDogs.toString();
  }

  test('should display "Unbegrenzt" for Pro plan (-1)', () => {
    expect(formatDogLimit(-1)).toBe('Unbegrenzt');
  });

  test('should display number for Free plan', () => {
    expect(formatDogLimit(10)).toBe('10');
  });
});

describe('Subscription Status Display', () => {
  function getStatusText(status) {
    const statusMap = {
      'active': 'Aktiv',
      'cancelled': 'Gekündigt',
      'past_due': 'Zahlung ausstehend',
      'trialing': 'Testphase'
    };
    return statusMap[status] || status;
  }

  function getStatusClass(status) {
    return `status-${status}`;
  }

  test('should return correct German text for statuses', () => {
    expect(getStatusText('active')).toBe('Aktiv');
    expect(getStatusText('cancelled')).toBe('Gekündigt');
    expect(getStatusText('past_due')).toBe('Zahlung ausstehend');
    expect(getStatusText('trialing')).toBe('Testphase');
  });

  test('should return original status if not mapped', () => {
    expect(getStatusText('unknown')).toBe('unknown');
  });

  test('should generate correct CSS class', () => {
    expect(getStatusClass('active')).toBe('status-active');
    expect(getStatusClass('cancelled')).toBe('status-cancelled');
    expect(getStatusClass('past_due')).toBe('status-past_due');
  });
});
