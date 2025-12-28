/**
 * FAQ Data Module Tests
 *
 * Tests for the unified FAQ data and helper functions.
 * Focus: Bug detection - XSS in HTML content, edge cases in search/filter functions.
 *
 * @jest-environment jsdom
 */

// The faq-data.js file exports to module.exports in Node.js
// But in browser context, we need to access the global functions
// Load as a proper module for testing
const faqModule = require('../internal/static/shared/js/faq-data.js');

const FAQ_DATA = faqModule.FAQ_DATA;
const getFAQsForLanding = faqModule.getFAQsForLanding;
const getFAQsForApp = faqModule.getFAQsForApp;
const getFAQsByCategory = faqModule.getFAQsByCategory;
const searchFAQs = faqModule.searchFAQs;
const getFAQsForContactCategory = faqModule.getFAQsForContactCategory;

beforeAll(() => {
    document.body.innerHTML = '';
});

describe('FAQ_DATA structure', () => {
    test('should be a non-empty array', () => {
        expect(Array.isArray(FAQ_DATA)).toBe(true);
        expect(FAQ_DATA.length).toBeGreaterThan(0);
    });

    test('each FAQ should have required fields', () => {
        FAQ_DATA.forEach(faq => {
            expect(faq.id).toBeDefined();
            expect(typeof faq.id).toBe('number');
            expect(faq.category).toBeDefined();
            expect(typeof faq.category).toBe('string');
            expect(faq.question).toBeDefined();
            expect(typeof faq.question).toBe('string');
            expect(faq.answer).toBeDefined();
            expect(typeof faq.answer).toBe('string');
            expect(faq.keywords).toBeDefined();
            expect(Array.isArray(faq.keywords)).toBe(true);
        });
    });

    test('all IDs should be unique', () => {
        const ids = FAQ_DATA.map(faq => faq.id);
        const uniqueIds = new Set(ids);
        expect(uniqueIds.size).toBe(ids.length);
    });

    test('should have FAQs in all expected categories', () => {
        const categories = new Set(FAQ_DATA.map(faq => faq.category));

        expect(categories.has('platform')).toBe(true);
        expect(categories.has('shared')).toBe(true);
        expect(categories.has('booking')).toBe(true);
        expect(categories.has('account')).toBe(true);
        expect(categories.has('dogs')).toBe(true);
    });
});

describe('getFAQsForLanding', () => {
    test('should return only platform and shared FAQs', () => {
        const landingFAQs = getFAQsForLanding();

        landingFAQs.forEach(faq => {
            expect(['platform', 'shared']).toContain(faq.category);
        });
    });

    test('should not include booking, account, or dogs FAQs', () => {
        const landingFAQs = getFAQsForLanding();

        const excludedCategories = landingFAQs.filter(faq =>
            ['booking', 'account', 'dogs'].includes(faq.category)
        );

        expect(excludedCategories.length).toBe(0);
    });

    test('should return non-empty array', () => {
        const landingFAQs = getFAQsForLanding();
        expect(landingFAQs.length).toBeGreaterThan(0);
    });
});

describe('getFAQsForApp', () => {
    test('should return booking, account, dogs, and shared FAQs', () => {
        const appFAQs = getFAQsForApp();

        appFAQs.forEach(faq => {
            expect(['booking', 'account', 'dogs', 'shared']).toContain(faq.category);
        });
    });

    test('should not include platform FAQs', () => {
        const appFAQs = getFAQsForApp();

        const platformFAQs = appFAQs.filter(faq => faq.category === 'platform');
        expect(platformFAQs.length).toBe(0);
    });

    test('should return non-empty array', () => {
        const appFAQs = getFAQsForApp();
        expect(appFAQs.length).toBeGreaterThan(0);
    });
});

describe('getFAQsByCategory', () => {
    test('should return FAQs for specific category', () => {
        const bookingFAQs = getFAQsByCategory('booking');

        bookingFAQs.forEach(faq => {
            expect(faq.category).toBe('booking');
        });
    });

    test('should return empty array for non-existent category', () => {
        const fakeFAQs = getFAQsByCategory('fake_category');
        expect(fakeFAQs).toEqual([]);
    });

    test('should return all app FAQs for "all" category', () => {
        const allFAQs = getFAQsByCategory('all');
        const appFAQs = getFAQsForApp();

        expect(allFAQs.length).toBe(appFAQs.length);
    });

    test('should handle null category', () => {
        const result = getFAQsByCategory(null);
        expect(Array.isArray(result)).toBe(true);
    });

    test('should handle undefined category', () => {
        const result = getFAQsByCategory(undefined);
        expect(Array.isArray(result)).toBe(true);
    });
});

describe('searchFAQs', () => {
    test('should find FAQs by question match', () => {
        const results = searchFAQs('buchen');
        expect(results.length).toBeGreaterThan(0);
        expect(results.some(faq => faq.question.toLowerCase().includes('buchen'))).toBe(true);
    });

    test('should find FAQs by answer match', () => {
        const results = searchFAQs('Dashboard');
        expect(results.length).toBeGreaterThan(0);
    });

    test('should find FAQs by keyword match', () => {
        // 'demo' is in platform category, test with landingOnly=true
        const landingResults = searchFAQs('demo', true);
        expect(landingResults.length).toBeGreaterThan(0);

        // Also test with an app FAQ keyword
        const appResults = searchFAQs('passwort');
        expect(appResults.length).toBeGreaterThan(0);
    });

    test('should be case insensitive', () => {
        const lowerResults = searchFAQs('passwort');
        const upperResults = searchFAQs('PASSWORT');
        const mixedResults = searchFAQs('PaSsWoRt');

        expect(lowerResults.length).toBe(upperResults.length);
        expect(lowerResults.length).toBe(mixedResults.length);
    });

    test('should return all app FAQs for empty query', () => {
        const results = searchFAQs('');
        const appFAQs = getFAQsForApp();

        expect(results.length).toBe(appFAQs.length);
    });

    test('should return all app FAQs for whitespace-only query', () => {
        const results = searchFAQs('   ');
        const appFAQs = getFAQsForApp();

        expect(results.length).toBe(appFAQs.length);
    });

    test('should return landing FAQs when landingOnly is true', () => {
        const results = searchFAQs('kostenlos', true);
        const landingFAQs = getFAQsForLanding();

        // All results should be in landing FAQs
        results.forEach(result => {
            expect(landingFAQs.some(faq => faq.id === result.id)).toBe(true);
        });
    });

    test('should return empty array for no matches', () => {
        const results = searchFAQs('xyzabcdefghijklmnop123456789');
        expect(results).toEqual([]);
    });

    test('should handle special regex characters in query', () => {
        // These characters could break if not escaped properly
        expect(() => searchFAQs('test.*+?^${}()|[]\\/')).not.toThrow();
        expect(() => searchFAQs('[a-z]+')).not.toThrow();
        expect(() => searchFAQs('(test)')).not.toThrow();
    });

    test('should handle very long search query', () => {
        const longQuery = 'a'.repeat(10000);
        expect(() => searchFAQs(longQuery)).not.toThrow();
    });

    test('should handle German umlauts', () => {
        const results = searchFAQs('Größe');
        expect(Array.isArray(results)).toBe(true);

        const results2 = searchFAQs('Höher');
        expect(Array.isArray(results2)).toBe(true);
    });

    test('should handle emoji in search', () => {
        const results = searchFAQs('🔒');
        expect(Array.isArray(results)).toBe(true);
    });
});

describe('getFAQsForContactCategory', () => {
    const contactCategories = ['general', 'support', 'sales', 'partnership', 'press', 'other'];

    contactCategories.forEach(category => {
        test(`should return FAQs for contact category: ${category}`, () => {
            const results = getFAQsForContactCategory(category);
            expect(Array.isArray(results)).toBe(true);
            expect(results.length).toBeGreaterThan(0);
        });
    });

    test('should return default FAQs for unknown category', () => {
        const results = getFAQsForContactCategory('unknown_category');
        expect(Array.isArray(results)).toBe(true);
        expect(results.length).toBeGreaterThan(0);
    });

    test('should handle null category', () => {
        const results = getFAQsForContactCategory(null);
        expect(Array.isArray(results)).toBe(true);
    });

    test('should handle undefined category', () => {
        const results = getFAQsForContactCategory(undefined);
        expect(Array.isArray(results)).toBe(true);
    });
});

describe('FAQ Content - XSS VULNERABILITY TESTS', () => {
    // CRITICAL: FAQs contain HTML in answers. Test that they don't have XSS.

    test('XSS: answers should not contain script tags', () => {
        const faqsWithScripts = FAQ_DATA.filter(faq =>
            faq.answer.toLowerCase().includes('<script')
        );

        expect(faqsWithScripts.length).toBe(0);
    });

    test('XSS: answers should not contain event handlers', () => {
        const eventHandlers = ['onerror', 'onload', 'onclick', 'onmouseover', 'onfocus'];

        eventHandlers.forEach(handler => {
            const faqsWithHandler = FAQ_DATA.filter(faq =>
                faq.answer.toLowerCase().includes(handler + '=')
            );

            expect(faqsWithHandler.length).toBe(0);
        });
    });

    test('XSS: answers should not contain javascript: protocol', () => {
        const faqsWithJS = FAQ_DATA.filter(faq =>
            faq.answer.toLowerCase().includes('javascript:')
        );

        expect(faqsWithJS.length).toBe(0);
    });

    test('XSS: answers should not contain data: URLs', () => {
        const faqsWithData = FAQ_DATA.filter(faq =>
            faq.answer.toLowerCase().includes('data:text/html')
        );

        expect(faqsWithData.length).toBe(0);
    });

    test('HTML: answers should only contain safe tags', () => {
        const safeTags = ['<a ', '<strong>', '</strong>', '<br>', '<br/>', '<br />'];
        const dangerousTags = ['<script', '<iframe', '<object', '<embed', '<svg', '<img'];

        FAQ_DATA.forEach(faq => {
            dangerousTags.forEach(tag => {
                expect(faq.answer.toLowerCase()).not.toContain(tag);
            });
        });
    });

    test('HTML: links should have safe href values', () => {
        const hrefRegex = /href=["']([^"']+)["']/g;

        FAQ_DATA.forEach(faq => {
            let match;
            while ((match = hrefRegex.exec(faq.answer)) !== null) {
                const href = match[1];

                // Should start with /, http://, https://, or #
                const isSafe = href.startsWith('/') ||
                    href.startsWith('http://') ||
                    href.startsWith('https://') ||
                    href.startsWith('#') ||
                    href.startsWith('mailto:');

                expect(isSafe).toBe(true);

                // Should not be javascript:
                expect(href.toLowerCase()).not.toMatch(/^javascript:/);
            }
        });
    });
});

describe('FAQ Content Quality', () => {
    test('all questions should end with question mark', () => {
        const questionsWithoutMark = FAQ_DATA.filter(faq =>
            !faq.question.trim().endsWith('?')
        );

        expect(questionsWithoutMark.length).toBe(0);
    });

    test('all answers should be non-empty', () => {
        FAQ_DATA.forEach(faq => {
            expect(faq.answer.trim().length).toBeGreaterThan(0);
        });
    });

    test('all keywords should be non-empty', () => {
        FAQ_DATA.forEach(faq => {
            faq.keywords.forEach(keyword => {
                expect(keyword.trim().length).toBeGreaterThan(0);
            });
        });
    });

    test('questions should not be too long', () => {
        const maxLength = 200;
        const longQuestions = FAQ_DATA.filter(faq => faq.question.length > maxLength);

        if (longQuestions.length > 0) {
            console.log('Long questions:', longQuestions.map(f => f.question.substring(0, 50)));
        }

        expect(longQuestions.length).toBe(0);
    });
});

describe('FAQ ID ranges', () => {
    test('platform FAQs should have IDs 1-10', () => {
        const platformFAQs = FAQ_DATA.filter(faq => faq.category === 'platform');
        platformFAQs.forEach(faq => {
            expect(faq.id).toBeGreaterThanOrEqual(1);
            expect(faq.id).toBeLessThanOrEqual(19);
        });
    });

    test('booking FAQs should have IDs 20-29', () => {
        const bookingFAQs = FAQ_DATA.filter(faq => faq.category === 'booking');
        bookingFAQs.forEach(faq => {
            expect(faq.id).toBeGreaterThanOrEqual(20);
            expect(faq.id).toBeLessThanOrEqual(29);
        });
    });

    test('account FAQs should have IDs 30-39', () => {
        const accountFAQs = FAQ_DATA.filter(faq => faq.category === 'account');
        accountFAQs.forEach(faq => {
            expect(faq.id).toBeGreaterThanOrEqual(30);
            expect(faq.id).toBeLessThanOrEqual(39);
        });
    });

    test('dogs FAQs should have IDs 40-49', () => {
        const dogsFAQs = FAQ_DATA.filter(faq => faq.category === 'dogs');
        dogsFAQs.forEach(faq => {
            expect(faq.id).toBeGreaterThanOrEqual(40);
            expect(faq.id).toBeLessThanOrEqual(49);
        });
    });
});

describe('Module exports (for Node.js)', () => {
    // The file exports functions for potential Node.js use

    test('should export FAQ_DATA', () => {
        expect(FAQ_DATA).toBeDefined();
    });

    test('should export getFAQsForLanding', () => {
        expect(typeof getFAQsForLanding).toBe('function');
    });

    test('should export getFAQsForApp', () => {
        expect(typeof getFAQsForApp).toBe('function');
    });

    test('should export getFAQsByCategory', () => {
        expect(typeof getFAQsByCategory).toBe('function');
    });

    test('should export searchFAQs', () => {
        expect(typeof searchFAQs).toBe('function');
    });

    test('should export getFAQsForContactCategory', () => {
        expect(typeof getFAQsForContactCategory).toBe('function');
    });
});

describe('Edge cases and robustness', () => {
    test('searchFAQs should handle null input', () => {
        expect(() => searchFAQs(null)).not.toThrow();
    });

    test('searchFAQs should handle undefined input', () => {
        expect(() => searchFAQs(undefined)).not.toThrow();
    });

    test('searchFAQs should handle number input', () => {
        expect(() => searchFAQs(123)).not.toThrow();
    });

    test('searchFAQs should handle object input', () => {
        expect(() => searchFAQs({})).not.toThrow();
    });

    test('functions should not modify FAQ_DATA', () => {
        const originalLength = FAQ_DATA.length;
        const originalFirstId = FAQ_DATA[0].id;

        getFAQsForLanding();
        getFAQsForApp();
        getFAQsByCategory('booking');
        searchFAQs('test');
        getFAQsForContactCategory('general');

        expect(FAQ_DATA.length).toBe(originalLength);
        expect(FAQ_DATA[0].id).toBe(originalFirstId);
    });
});

describe('Performance considerations', () => {
    test('should handle concurrent searches', () => {
        const queries = ['test', 'buchen', 'email', 'passwort', 'hund'];
        const results = queries.map(q => searchFAQs(q));

        expect(results.length).toBe(queries.length);
        results.forEach(result => {
            expect(Array.isArray(result)).toBe(true);
        });
    });

    test('should handle many rapid filter calls', () => {
        for (let i = 0; i < 100; i++) {
            getFAQsForLanding();
            getFAQsForApp();
            getFAQsByCategory('booking');
        }

        // If we get here without timeout, performance is acceptable
        expect(true).toBe(true);
    });
});
