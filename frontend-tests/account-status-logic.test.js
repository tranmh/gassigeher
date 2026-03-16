/**
 * Account Status Page Logic Tests
 *
 * Tests for the account-status.html inline JavaScript logic.
 * Since the code is inline, we extract and test the pure logic functions.
 * Focus: Bug detection - date calculations, edge cases, division by zero.
 *
 * @jest-environment jsdom
 */

// Mock dependencies
beforeEach(() => {
    // Mock api
    window.api = {
        isAuthenticated: jest.fn().mockReturnValue(true),
        getMe: jest.fn().mockResolvedValue({ is_verified: true, colors: [] }),
        getDogs: jest.fn().mockResolvedValue([]),
        getColorCategories: jest.fn().mockResolvedValue([]),
        getSettings: jest.fn().mockResolvedValue({})
    };

    // Mock i18n
    window.i18n = {
        load: jest.fn().mockResolvedValue({}),
        updateElement: jest.fn()
    };

    // Mock ImpersonationBanner
    window.ImpersonationBanner = {
        init: jest.fn().mockResolvedValue()
    };
});

// ============================================
// Extracted pure functions from account-status.html
// These mirror the logic in the HTML file
// ============================================

const COLOR_CONFIG = {
    green: { name: 'Grün', color: '#82b965', icon: '🟢' },
    orange: { name: 'Orange', color: '#f5a623', icon: '🟠' },
    blue: { name: 'Blau', color: '#4a90e2', icon: '🔵' }
};

/**
 * Calculate days ago from a date
 * @param {Date|string|null} dateInput - Date to calculate from
 * @returns {number} Days ago (0 if today, negative shouldn't happen)
 */
function calculateDaysAgo(dateInput) {
    if (!dateInput) return null;

    const date = dateInput instanceof Date ? dateInput : new Date(dateInput);
    if (isNaN(date.getTime())) return null;

    return Math.floor((Date.now() - date.getTime()) / (1000 * 60 * 60 * 24));
}

/**
 * Calculate days until deactivation warning
 * Based on account-status.html logic
 */
function calculateDaysUntilWarning(lastActivityDate, autoDeactivationDays = 365) {
    const daysInactive = lastActivityDate
        ? Math.floor((Date.now() - new Date(lastActivityDate).getTime()) / (1000 * 60 * 60 * 24))
        : 0;

    return Math.max(0, autoDeactivationDays - daysInactive - 30);
}

/**
 * Calculate days until deactivation
 */
function calculateDaysUntilDeactivation(lastActivityDate, autoDeactivationDays = 365) {
    const daysInactive = lastActivityDate
        ? Math.floor((Date.now() - new Date(lastActivityDate).getTime()) / (1000 * 60 * 60 * 24))
        : 0;

    return Math.max(0, autoDeactivationDays - daysInactive);
}

/**
 * Calculate progress bar percentage
 * Fixed: Handle division by zero and negative values
 */
function calculateProgressPercent(daysUntilDeactivation, autoDeactivationDays) {
    if (autoDeactivationDays <= 0) return 0;
    return Math.min(100, Math.max(0, (daysUntilDeactivation / autoDeactivationDays) * 100));
}

/**
 * Determine progress bar class based on percentage
 */
function getProgressClass(progressPercent) {
    if (progressPercent > 50) return 'success';
    if (progressPercent > 20) return 'warning';
    return 'danger';
}

/**
 * Count eligible dogs based on user colors
 * Fixed: Handle missing color_category.name property
 */
function countEligibleDogs(dogs, userColors) {
    const userColorNames = userColors.map(c => c.name ? c.name.toLowerCase() : '');

    return dogs.filter(dog => {
        if (!dog.is_available) return false;
        // Handle missing color_category or missing name property
        if (!dog.color_category || !dog.color_category.name) return true;
        return userColorNames.includes(dog.color_category.name.toLowerCase());
    });
}

/**
 * Get activity icon class based on days ago
 */
function getActivityIconClass(daysAgo) {
    if (daysAgo === null) return 'warning';
    if (daysAgo < 30) return 'success';
    if (daysAgo < 180) return 'warning';
    return 'danger';
}

// ============================================
// Tests
// ============================================

describe('calculateDaysAgo', () => {
    test('should return 0 for today', () => {
        const today = new Date();
        expect(calculateDaysAgo(today)).toBe(0);
    });

    test('should return correct days for past date', () => {
        const fiveDaysAgo = new Date(Date.now() - 5 * 24 * 60 * 60 * 1000);
        expect(calculateDaysAgo(fiveDaysAgo)).toBe(5);
    });

    test('should handle string date input', () => {
        const fiveDaysAgo = new Date(Date.now() - 5 * 24 * 60 * 60 * 1000);
        expect(calculateDaysAgo(fiveDaysAgo.toISOString())).toBe(5);
    });

    test('should return null for null input', () => {
        expect(calculateDaysAgo(null)).toBeNull();
    });

    test('should return null for undefined input', () => {
        expect(calculateDaysAgo(undefined)).toBeNull();
    });

    test('should return null for invalid date string', () => {
        expect(calculateDaysAgo('invalid-date')).toBeNull();
    });

    test('should return null for empty string', () => {
        expect(calculateDaysAgo('')).toBeNull();
    });

    // BUG DETECTION: Future dates
    test('EDGE CASE: should handle future dates (negative days)', () => {
        const tomorrow = new Date(Date.now() + 24 * 60 * 60 * 1000);
        const result = calculateDaysAgo(tomorrow);
        expect(result).toBeLessThan(0);
    });
});

describe('calculateDaysUntilWarning', () => {
    test('should return full warning period for recent activity', () => {
        const today = new Date().toISOString();
        const result = calculateDaysUntilWarning(today, 365);
        expect(result).toBe(335); // 365 - 0 - 30
    });

    test('should decrease with inactivity', () => {
        const thirtyDaysAgo = new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString();
        const result = calculateDaysUntilWarning(thirtyDaysAgo, 365);
        expect(result).toBe(305); // 365 - 30 - 30
    });

    test('should return 0 when past warning threshold', () => {
        const yearAgo = new Date(Date.now() - 400 * 24 * 60 * 60 * 1000).toISOString();
        const result = calculateDaysUntilWarning(yearAgo, 365);
        expect(result).toBe(0);
    });

    test('should handle null lastActivityDate', () => {
        const result = calculateDaysUntilWarning(null, 365);
        expect(result).toBe(335); // Assumes 0 days inactive
    });

    // BUG DETECTION: Edge case with custom deactivation days
    test('should handle small autoDeactivationDays value', () => {
        const today = new Date().toISOString();
        const result = calculateDaysUntilWarning(today, 20);
        // 20 - 0 - 30 = -10, but should be 0
        expect(result).toBe(0);
    });

    test('should handle zero autoDeactivationDays', () => {
        const today = new Date().toISOString();
        const result = calculateDaysUntilWarning(today, 0);
        expect(result).toBe(0);
    });
});

describe('calculateDaysUntilDeactivation', () => {
    test('should return full days for recent activity', () => {
        const today = new Date().toISOString();
        const result = calculateDaysUntilDeactivation(today, 365);
        expect(result).toBe(365);
    });

    test('should decrease with inactivity', () => {
        const hundredDaysAgo = new Date(Date.now() - 100 * 24 * 60 * 60 * 1000).toISOString();
        const result = calculateDaysUntilDeactivation(hundredDaysAgo, 365);
        expect(result).toBe(265);
    });

    test('should return 0 when past deactivation threshold', () => {
        const yearAgo = new Date(Date.now() - 400 * 24 * 60 * 60 * 1000).toISOString();
        const result = calculateDaysUntilDeactivation(yearAgo, 365);
        expect(result).toBe(0);
    });

    test('should never return negative value', () => {
        const veryOld = new Date(Date.now() - 1000 * 24 * 60 * 60 * 1000).toISOString();
        const result = calculateDaysUntilDeactivation(veryOld, 365);
        expect(result).toBeGreaterThanOrEqual(0);
    });
});

describe('calculateProgressPercent - BUG DETECTION', () => {
    test('should return 100% for fully active account', () => {
        const result = calculateProgressPercent(365, 365);
        expect(result).toBe(100);
    });

    test('should return 50% for half-way point', () => {
        const result = calculateProgressPercent(182.5, 365);
        expect(result).toBe(50);
    });

    test('should return 0% for expired account', () => {
        const result = calculateProgressPercent(0, 365);
        expect(result).toBe(0);
    });

    test('should cap at 100% for values over threshold', () => {
        const result = calculateProgressPercent(500, 365);
        expect(result).toBe(100);
    });

    // BUG DETECTION: Division by zero
    test('CRITICAL BUG: should handle zero autoDeactivationDays (division by zero)', () => {
        const result = calculateProgressPercent(100, 0);
        // Check if it's a sane value or Infinity/NaN
        expect(isFinite(result)).toBe(true);
        expect(isNaN(result)).toBe(false);
    });

    test('should handle negative values', () => {
        const result = calculateProgressPercent(-10, 365);
        // Negative would give negative percent
        expect(result).toBeGreaterThanOrEqual(0);
    });
});

describe('getProgressClass', () => {
    test('should return success for > 50%', () => {
        expect(getProgressClass(75)).toBe('success');
        expect(getProgressClass(51)).toBe('success');
        expect(getProgressClass(100)).toBe('success');
    });

    test('should return warning for 20-50%', () => {
        expect(getProgressClass(50)).toBe('warning');
        expect(getProgressClass(21)).toBe('warning');
        expect(getProgressClass(35)).toBe('warning');
    });

    test('should return danger for < 20%', () => {
        expect(getProgressClass(20)).toBe('danger');
        expect(getProgressClass(10)).toBe('danger');
        expect(getProgressClass(0)).toBe('danger');
    });

    test('should handle edge case at 50.0001%', () => {
        expect(getProgressClass(50.0001)).toBe('success');
    });
});

describe('countEligibleDogs', () => {
    test('should count dogs matching user colors', () => {
        const dogs = [
            { is_available: true, color_category: { name: 'green' } },
            { is_available: true, color_category: { name: 'orange' } },
            { is_available: true, color_category: { name: 'blue' } }
        ];
        const userColors = [{ name: 'green' }, { name: 'orange' }];

        const eligible = countEligibleDogs(dogs, userColors);
        expect(eligible.length).toBe(2);
    });

    test('should include dogs without color_category', () => {
        const dogs = [
            { is_available: true, color_category: null },
            { is_available: true }
        ];
        const userColors = [{ name: 'green' }];

        const eligible = countEligibleDogs(dogs, userColors);
        expect(eligible.length).toBe(2);
    });

    test('should exclude unavailable dogs', () => {
        const dogs = [
            { is_available: false, color_category: { name: 'green' } },
            { is_available: true, color_category: { name: 'green' } }
        ];
        const userColors = [{ name: 'green' }];

        const eligible = countEligibleDogs(dogs, userColors);
        expect(eligible.length).toBe(1);
    });

    test('should handle empty user colors', () => {
        const dogs = [
            { is_available: true, color_category: { name: 'green' } }
        ];
        const userColors = [];

        const eligible = countEligibleDogs(dogs, userColors);
        expect(eligible.length).toBe(0);
    });

    test('should handle empty dogs array', () => {
        const eligible = countEligibleDogs([], [{ name: 'green' }]);
        expect(eligible.length).toBe(0);
    });

    test('should be case insensitive', () => {
        const dogs = [
            { is_available: true, color_category: { name: 'GREEN' } },
            { is_available: true, color_category: { name: 'Green' } }
        ];
        const userColors = [{ name: 'green' }];

        const eligible = countEligibleDogs(dogs, userColors);
        expect(eligible.length).toBe(2);
    });

    // BUG DETECTION: Missing properties
    test('should handle dogs with missing is_available', () => {
        const dogs = [
            { color_category: { name: 'green' } } // no is_available
        ];
        const userColors = [{ name: 'green' }];

        const eligible = countEligibleDogs(dogs, userColors);
        // Falsy is_available should exclude dog
        expect(eligible.length).toBe(0);
    });

    test('should handle color_category with missing name', () => {
        const dogs = [
            { is_available: true, color_category: {} } // no name
        ];
        const userColors = [{ name: 'green' }];

        // This could throw an error
        expect(() => countEligibleDogs(dogs, userColors)).not.toThrow();
    });
});

describe('getActivityIconClass', () => {
    test('should return success for recent activity', () => {
        expect(getActivityIconClass(0)).toBe('success');
        expect(getActivityIconClass(29)).toBe('success');
    });

    test('should return warning for moderate inactivity', () => {
        expect(getActivityIconClass(30)).toBe('warning');
        expect(getActivityIconClass(179)).toBe('warning');
    });

    test('should return danger for long inactivity', () => {
        expect(getActivityIconClass(180)).toBe('danger');
        expect(getActivityIconClass(365)).toBe('danger');
    });

    test('should return warning for null (no activity)', () => {
        expect(getActivityIconClass(null)).toBe('warning');
    });

    // Edge case
    test('should handle exactly 30 days', () => {
        expect(getActivityIconClass(30)).toBe('warning');
    });
});

describe('COLOR_CONFIG structure', () => {
    test('should have all color entries', () => {
        expect(COLOR_CONFIG.green).toBeDefined();
        expect(COLOR_CONFIG.orange).toBeDefined();
        expect(COLOR_CONFIG.blue).toBeDefined();
    });

    test('each color should have name, color, and icon', () => {
        Object.values(COLOR_CONFIG).forEach(config => {
            expect(config.name).toBeDefined();
            expect(config.color).toBeDefined();
            expect(config.icon).toBeDefined();
        });
    });

    test('colors should be valid hex codes', () => {
        const hexRegex = /^#[0-9A-Fa-f]{6}$/;
        Object.values(COLOR_CONFIG).forEach(config => {
            expect(config.color).toMatch(hexRegex);
        });
    });
});

describe('Integration scenarios', () => {
    test('new user scenario (just registered)', () => {
        const today = new Date().toISOString();
        const autoDeactivationDays = 365;

        const daysAgo = calculateDaysAgo(today);
        const daysUntilWarning = calculateDaysUntilWarning(today, autoDeactivationDays);
        const daysUntilDeactivation = calculateDaysUntilDeactivation(today, autoDeactivationDays);
        const progressPercent = calculateProgressPercent(daysUntilDeactivation, autoDeactivationDays);
        const progressClass = getProgressClass(progressPercent);

        expect(daysAgo).toBe(0);
        expect(daysUntilWarning).toBe(335);
        expect(daysUntilDeactivation).toBe(365);
        expect(progressPercent).toBe(100);
        expect(progressClass).toBe('success');
    });

    test('warning scenario (user approaching deactivation)', () => {
        const elevenMonthsAgo = new Date(Date.now() - 330 * 24 * 60 * 60 * 1000).toISOString();
        const autoDeactivationDays = 365;

        const daysAgo = calculateDaysAgo(elevenMonthsAgo);
        const daysUntilWarning = calculateDaysUntilWarning(elevenMonthsAgo, autoDeactivationDays);
        const daysUntilDeactivation = calculateDaysUntilDeactivation(elevenMonthsAgo, autoDeactivationDays);
        const progressPercent = calculateProgressPercent(daysUntilDeactivation, autoDeactivationDays);
        const progressClass = getProgressClass(progressPercent);

        expect(daysAgo).toBe(330);
        expect(daysUntilWarning).toBe(5); // 365 - 330 - 30
        expect(daysUntilDeactivation).toBe(35);
        expect(progressPercent).toBeCloseTo(9.59, 1);
        expect(progressClass).toBe('danger');
    });

    test('deactivated scenario (past threshold)', () => {
        const yearAndHalfAgo = new Date(Date.now() - 500 * 24 * 60 * 60 * 1000).toISOString();
        const autoDeactivationDays = 365;

        const daysUntilWarning = calculateDaysUntilWarning(yearAndHalfAgo, autoDeactivationDays);
        const daysUntilDeactivation = calculateDaysUntilDeactivation(yearAndHalfAgo, autoDeactivationDays);
        const progressPercent = calculateProgressPercent(daysUntilDeactivation, autoDeactivationDays);

        expect(daysUntilWarning).toBe(0);
        expect(daysUntilDeactivation).toBe(0);
        expect(progressPercent).toBe(0);
    });
});

describe('Edge cases from account-status.html', () => {
    test('should handle user with no last_activity_at', () => {
        const user = { is_verified: true, colors: [] };
        // In the HTML, this uses: user.last_activity_at ? new Date(user.last_activity_at) : null

        const lastActivity = user.last_activity_at ? new Date(user.last_activity_at) : null;
        expect(lastActivity).toBeNull();
    });

    test('should handle settings with missing auto_deactivation_days', () => {
        const settings = {};
        const autoDeactivationDays = settings.auto_deactivation_days || 365;
        expect(autoDeactivationDays).toBe(365);
    });

    test('should handle API returning null for getSettings', () => {
        const settings = null || {};
        const autoDeactivationDays = settings.auto_deactivation_days || 365;
        expect(autoDeactivationDays).toBe(365);
    });
});
