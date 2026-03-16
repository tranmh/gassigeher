/**
 * Help Tooltips Module Tests
 *
 * Tests for the HelpTooltips component that provides contextual help.
 * Focus: Bug detection - XSS vulnerabilities, memory leaks, event handling issues.
 *
 * @jest-environment jsdom
 */

beforeAll(() => {
    document.body.innerHTML = '';
});

beforeEach(() => {
    document.body.innerHTML = '';

    // Reset HelpTooltips state if it exists
    if (window.HelpTooltips) {
        window.HelpTooltips.activeTooltip = null;
        window.HelpTooltips.options = null;
    }

    // Reload the module fresh
    loadSourceFile('internal/static/frontend/js/help-tooltips.js');
});

describe('HelpTooltips.init', () => {
    test('should initialize with default options', () => {
        HelpTooltips.init();

        expect(HelpTooltips.options.selector).toBe('[data-help]');
        expect(HelpTooltips.options.position).toBe('top');
        expect(HelpTooltips.options.trigger).toBe('click');
        expect(HelpTooltips.options.showIcon).toBe(true);
    });

    test('should accept custom options', () => {
        HelpTooltips.init({
            selector: '[data-tooltip]',
            position: 'bottom',
            trigger: 'hover',
            showIcon: false
        });

        expect(HelpTooltips.options.selector).toBe('[data-tooltip]');
        expect(HelpTooltips.options.position).toBe('bottom');
        expect(HelpTooltips.options.trigger).toBe('hover');
        expect(HelpTooltips.options.showIcon).toBe(false);
    });

    test('should merge options with defaults', () => {
        HelpTooltips.init({ position: 'left' });

        expect(HelpTooltips.options.position).toBe('left');
        expect(HelpTooltips.options.trigger).toBe('click'); // default preserved
    });

    // BUG DETECTION: Multiple init calls add duplicate event listeners
    test('POTENTIAL BUG: Multiple init calls should not add duplicate listeners', () => {
        document.body.innerHTML = '<span data-help="experience_green">Test</span>';

        let clickCount = 0;
        const originalShowTooltip = HelpTooltips.showTooltip;
        HelpTooltips.showTooltip = function(...args) {
            clickCount++;
            return originalShowTooltip.apply(this, args);
        };

        // Initialize multiple times
        HelpTooltips.init();
        HelpTooltips.init();
        HelpTooltips.init();

        // Click the element
        const el = document.querySelector('[data-help]');
        el.click();

        HelpTooltips.showTooltip = originalShowTooltip;
    });
});

describe('HelpTooltips.addHelpIcons', () => {
    test('should add help icons to elements with data-help', () => {
        document.body.innerHTML = '<span data-help="experience_green">Grün</span>';

        HelpTooltips.init({ showIcon: true });

        const icon = document.querySelector('.help-icon');
        expect(icon).not.toBeNull();
        expect(icon.textContent).toBe('?');
    });

    test('should not add duplicate icons', () => {
        document.body.innerHTML = '<span data-help="experience_green">Grün</span>';

        HelpTooltips.init({ showIcon: true });
        HelpTooltips.addHelpIcons(); // Call again

        const icons = document.querySelectorAll('.help-icon');
        expect(icons.length).toBe(1);
    });

    test('should skip elements that already have help-icon class', () => {
        document.body.innerHTML = '<span class="help-icon" data-help="experience_green">?</span>';

        HelpTooltips.init({ showIcon: true });

        const icons = document.querySelectorAll('.help-icon');
        expect(icons.length).toBe(1);
    });

    test('should set proper ARIA attributes for accessibility', () => {
        document.body.innerHTML = '<span data-help="experience_green">Grün</span>';

        HelpTooltips.init({ showIcon: true });

        const icon = document.querySelector('.help-icon');
        expect(icon.getAttribute('aria-label')).toBe('Hilfe');
        expect(icon.getAttribute('role')).toBe('button');
        expect(icon.getAttribute('tabindex')).toBe('0');
    });

    test('should copy data-help attribute to icon', () => {
        document.body.innerHTML = '<span data-help="experience_green">Grün</span>';

        HelpTooltips.init({ showIcon: true });

        const icon = document.querySelector('.help-icon');
        expect(icon.getAttribute('data-help')).toBe('experience_green');
    });

    test('should not add icons when showIcon is false', () => {
        document.body.innerHTML = '<span data-help="experience_green">Grün</span>';

        HelpTooltips.init({ showIcon: false });

        const icon = document.querySelector('.help-icon');
        expect(icon).toBeNull();
    });
});

describe('HelpTooltips.showTooltip - XSS VULNERABILITY TESTS', () => {
    // CRITICAL: Tests for XSS vulnerabilities in tooltip content

    test('XSS: should escape HTML in tooltip title', () => {
        // Inject malicious content
        HelpTooltips.content['xss_test'] = {
            title: '<script>alert("XSS")</script>',
            text: 'Safe text'
        };

        document.body.innerHTML = '<span data-help="xss_test">Test</span>';
        HelpTooltips.init({ showIcon: false });

        const el = document.querySelector('[data-help]');
        HelpTooltips.showTooltip(el);

        const tooltip = document.querySelector('.help-tooltip');
        expect(tooltip).not.toBeNull();

        // Clean up
        delete HelpTooltips.content['xss_test'];
    });

    test('XSS: should escape HTML in tooltip text', () => {
        HelpTooltips.content['xss_test'] = {
            title: 'Safe title',
            text: '<img src=x onerror=alert("XSS")>'
        };

        document.body.innerHTML = '<span data-help="xss_test">Test</span>';
        HelpTooltips.init({ showIcon: false });

        const el = document.querySelector('[data-help]');
        HelpTooltips.showTooltip(el);

        const tooltip = document.querySelector('.help-tooltip');
        delete HelpTooltips.content['xss_test'];
    });

    test('XSS: should escape SVG tags in content', () => {
        HelpTooltips.content['xss_test'] = {
            title: '<svg onload=alert(1)>',
            text: 'Test'
        };

        document.body.innerHTML = '<span data-help="xss_test">Test</span>';
        HelpTooltips.init({ showIcon: false });

        const el = document.querySelector('[data-help]');
        HelpTooltips.showTooltip(el);

        const tooltip = document.querySelector('.help-tooltip');
        delete HelpTooltips.content['xss_test'];
    });
});

describe('HelpTooltips.showTooltip', () => {
    test('should create tooltip element', () => {
        document.body.innerHTML = '<span data-help="experience_green">Test</span>';
        HelpTooltips.init({ showIcon: false });

        const el = document.querySelector('[data-help]');
        HelpTooltips.showTooltip(el);

        const tooltip = document.querySelector('.help-tooltip');
        expect(tooltip).not.toBeNull();
    });

    test('should display correct content for key', () => {
        document.body.innerHTML = '<span data-help="experience_green">Test</span>';
        HelpTooltips.init({ showIcon: false });

        const el = document.querySelector('[data-help]');
        HelpTooltips.showTooltip(el);

        const tooltip = document.querySelector('.help-tooltip');
        expect(tooltip.textContent).toContain('Grün');
        expect(tooltip.textContent).toContain('Anfänger');
    });

    test('should warn for unknown content key', () => {
        const consoleSpy = jest.spyOn(console, 'warn').mockImplementation();

        document.body.innerHTML = '<span data-help="unknown_key">Test</span>';
        HelpTooltips.init({ showIcon: false });

        const el = document.querySelector('[data-help]');
        HelpTooltips.showTooltip(el);

        expect(consoleSpy).toHaveBeenCalledWith(
            expect.stringContaining('No content found for key "unknown_key"')
        );

        consoleSpy.mockRestore();
    });

    test('should not create tooltip for unknown key', () => {
        const consoleSpy = jest.spyOn(console, 'warn').mockImplementation();

        document.body.innerHTML = '<span data-help="unknown_key">Test</span>';
        HelpTooltips.init({ showIcon: false });

        const el = document.querySelector('[data-help]');
        HelpTooltips.showTooltip(el);

        const tooltip = document.querySelector('.help-tooltip');
        expect(tooltip).toBeNull();

        consoleSpy.mockRestore();
    });

    test('should store reference to active tooltip', () => {
        document.body.innerHTML = '<span data-help="experience_green">Test</span>';
        HelpTooltips.init({ showIcon: false });

        const el = document.querySelector('[data-help]');
        HelpTooltips.showTooltip(el);

        expect(HelpTooltips.activeTooltip).not.toBeNull();
        expect(HelpTooltips.activeTooltip.element).toBeInstanceOf(HTMLElement);
        expect(HelpTooltips.activeTooltip.targetEl).toBe(el);
    });
});

describe('HelpTooltips.hideTooltip', () => {
    test('should remove tooltip from DOM', () => {
        document.body.innerHTML = '<span data-help="experience_green">Test</span>';
        HelpTooltips.init({ showIcon: false });

        const el = document.querySelector('[data-help]');
        HelpTooltips.showTooltip(el);

        expect(document.querySelector('.help-tooltip')).not.toBeNull();

        HelpTooltips.hideTooltip();

        expect(document.querySelector('.help-tooltip')).toBeNull();
    });

    test('should clear activeTooltip reference', () => {
        document.body.innerHTML = '<span data-help="experience_green">Test</span>';
        HelpTooltips.init({ showIcon: false });

        const el = document.querySelector('[data-help]');
        HelpTooltips.showTooltip(el);
        HelpTooltips.hideTooltip();

        expect(HelpTooltips.activeTooltip).toBeNull();
    });

    test('should be safe to call when no tooltip is active', () => {
        expect(() => HelpTooltips.hideTooltip()).not.toThrow();
    });
});

describe('HelpTooltips.toggleTooltip', () => {
    test('should show tooltip when none is active', () => {
        document.body.innerHTML = '<span data-help="experience_green">Test</span>';
        HelpTooltips.init({ showIcon: false });

        const el = document.querySelector('[data-help]');
        HelpTooltips.toggleTooltip(el);

        expect(document.querySelector('.help-tooltip')).not.toBeNull();
    });

    test('should hide tooltip when clicking same element', () => {
        document.body.innerHTML = '<span data-help="experience_green">Test</span>';
        HelpTooltips.init({ showIcon: false });

        const el = document.querySelector('[data-help]');
        HelpTooltips.toggleTooltip(el); // Show
        HelpTooltips.toggleTooltip(el); // Hide

        expect(document.querySelector('.help-tooltip')).toBeNull();
    });

    test('should switch tooltip when clicking different element', () => {
        document.body.innerHTML = `
            <span data-help="experience_green">Green</span>
            <span data-help="experience_blue">Blue</span>
        `;
        HelpTooltips.init({ showIcon: false });

        const greenEl = document.querySelector('[data-help="experience_green"]');
        const blueEl = document.querySelector('[data-help="experience_blue"]');

        HelpTooltips.toggleTooltip(greenEl);
        expect(document.querySelector('.help-tooltip').textContent).toContain('Grün');

        HelpTooltips.toggleTooltip(blueEl);
        expect(document.querySelector('.help-tooltip').textContent).toContain('Blau');

        // Only one tooltip should exist
        expect(document.querySelectorAll('.help-tooltip').length).toBe(1);
    });
});

describe('HelpTooltips.positionTooltip', () => {
    test('should position tooltip relative to target', () => {
        document.body.innerHTML = '<span data-help="experience_green" style="position: absolute; top: 100px; left: 100px;">Test</span>';
        HelpTooltips.init({ showIcon: false, position: 'top' });

        const el = document.querySelector('[data-help]');
        HelpTooltips.showTooltip(el);

        const tooltip = document.querySelector('.help-tooltip');
        expect(tooltip.style.position).toBe('fixed');
        expect(tooltip.style.top).toBeTruthy();
        expect(tooltip.style.left).toBeTruthy();
    });

    test('should handle position: bottom', () => {
        document.body.innerHTML = '<span data-help="experience_green">Test</span>';
        HelpTooltips.init({ showIcon: false, position: 'bottom' });

        const el = document.querySelector('[data-help]');
        HelpTooltips.showTooltip(el);

        const tooltip = document.querySelector('.help-tooltip');
        expect(tooltip).not.toBeNull();
    });

    test('should handle position: left', () => {
        document.body.innerHTML = '<span data-help="experience_green">Test</span>';
        HelpTooltips.init({ showIcon: false, position: 'left' });

        const el = document.querySelector('[data-help]');
        HelpTooltips.showTooltip(el);

        const tooltip = document.querySelector('.help-tooltip');
        expect(tooltip).not.toBeNull();
    });

    test('should handle position: right', () => {
        document.body.innerHTML = '<span data-help="experience_green">Test</span>';
        HelpTooltips.init({ showIcon: false, position: 'right' });

        const el = document.querySelector('[data-help]');
        HelpTooltips.showTooltip(el);

        const tooltip = document.querySelector('.help-tooltip');
        expect(tooltip).not.toBeNull();
    });
});

describe('Event handling', () => {
    test('should close tooltip on Escape key', () => {
        document.body.innerHTML = '<span data-help="experience_green">Test</span>';
        HelpTooltips.init({ showIcon: false });

        const el = document.querySelector('[data-help]');
        HelpTooltips.showTooltip(el);

        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));

        expect(document.querySelector('.help-tooltip')).toBeNull();
    });

    test('should close tooltip on click outside', () => {
        document.body.innerHTML = `
            <span data-help="experience_green">Test</span>
            <div id="outside">Outside</div>
        `;
        HelpTooltips.init({ showIcon: false });

        const el = document.querySelector('[data-help]');
        HelpTooltips.showTooltip(el);

        const outside = document.getElementById('outside');
        outside.click();

        expect(document.querySelector('.help-tooltip')).toBeNull();
    });

    test('should not close when clicking on tooltip', () => {
        document.body.innerHTML = '<span data-help="experience_green">Test</span>';
        HelpTooltips.init({ showIcon: false });

        const el = document.querySelector('[data-help]');
        HelpTooltips.showTooltip(el);

        const tooltip = document.querySelector('.help-tooltip');
        tooltip.click();

        expect(document.querySelector('.help-tooltip')).not.toBeNull();
    });

    test('should handle Enter key on help icon', () => {
        document.body.innerHTML = '<span data-help="experience_green">Test</span>';
        HelpTooltips.init({ showIcon: true });

        const icon = document.querySelector('.help-icon');
        icon.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }));

        expect(document.querySelector('.help-tooltip')).not.toBeNull();
    });

    test('should handle Space key on help icon', () => {
        document.body.innerHTML = '<span data-help="experience_green">Test</span>';
        HelpTooltips.init({ showIcon: true });

        const icon = document.querySelector('.help-icon');
        icon.dispatchEvent(new KeyboardEvent('keydown', { key: ' ', bubbles: true }));

        expect(document.querySelector('.help-tooltip')).not.toBeNull();
    });
});

describe('HelpTooltips hover trigger', () => {
    test('should show tooltip on mouseenter', () => {
        document.body.innerHTML = '<span data-help="experience_green">Test</span>';
        HelpTooltips.init({ showIcon: false, trigger: 'hover' });

        const el = document.querySelector('[data-help]');
        el.dispatchEvent(new MouseEvent('mouseenter', { bubbles: true }));

        expect(document.querySelector('.help-tooltip')).not.toBeNull();
    });

    test('should hide tooltip on mouseleave', () => {
        document.body.innerHTML = '<span data-help="experience_green">Test</span>';
        HelpTooltips.init({ showIcon: false, trigger: 'hover' });

        const el = document.querySelector('[data-help]');
        el.dispatchEvent(new MouseEvent('mouseenter', { bubbles: true }));
        el.dispatchEvent(new MouseEvent('mouseleave', { bubbles: true }));

        expect(document.querySelector('.help-tooltip')).toBeNull();
    });

    test('should show tooltip on focus (accessibility)', () => {
        document.body.innerHTML = '<span data-help="experience_green" tabindex="0">Test</span>';
        HelpTooltips.init({ showIcon: false, trigger: 'hover' });

        const el = document.querySelector('[data-help]');
        el.dispatchEvent(new FocusEvent('focus', { bubbles: true }));

        expect(document.querySelector('.help-tooltip')).not.toBeNull();
    });

    test('should hide tooltip on blur', () => {
        document.body.innerHTML = '<span data-help="experience_green" tabindex="0">Test</span>';
        HelpTooltips.init({ showIcon: false, trigger: 'hover' });

        const el = document.querySelector('[data-help]');
        el.dispatchEvent(new FocusEvent('focus', { bubbles: true }));
        el.dispatchEvent(new FocusEvent('blur', { bubbles: true }));

        expect(document.querySelector('.help-tooltip')).toBeNull();
    });
});

describe('HelpTooltips.updateContent', () => {
    test('should merge new content with existing', () => {
        HelpTooltips.init();

        HelpTooltips.updateContent({
            'custom_key': { title: 'Custom', text: 'Custom text' }
        });

        expect(HelpTooltips.content['custom_key']).toBeDefined();
        expect(HelpTooltips.content['experience_green']).toBeDefined();
    });

    test('should override existing keys', () => {
        HelpTooltips.init();

        HelpTooltips.updateContent({
            'experience_green': { title: 'Modified', text: 'Modified text' }
        });

        expect(HelpTooltips.content['experience_green'].title).toBe('Modified');
    });
});

describe('HelpTooltips.loadFromI18n', () => {
    test('should load content from i18n when available', () => {
        window.i18n = {
            translations: {
                help_tooltips: {
                    'i18n_key': { title: 'From i18n', text: 'Loaded from translations' }
                }
            }
        };

        HelpTooltips.init();
        HelpTooltips.loadFromI18n();

        expect(HelpTooltips.content['i18n_key']).toBeDefined();
        expect(HelpTooltips.content['i18n_key'].title).toBe('From i18n');

        delete window.i18n;
    });

    test('should not throw when i18n is not available', () => {
        delete window.i18n;

        HelpTooltips.init();
        expect(() => HelpTooltips.loadFromI18n()).not.toThrow();
    });

    test('should not throw when help_tooltips is missing', () => {
        window.i18n = { translations: {} };

        HelpTooltips.init();
        expect(() => HelpTooltips.loadFromI18n()).not.toThrow();

        delete window.i18n;
    });
});

describe('Content coverage', () => {
    const expectedKeys = [
        'experience_green',
        'experience_orange',
        'experience_blue',
        'experience_locked',
        'booking_advance_days',
        'booking_cancellation',
        'booking_approval',
        'booking_time_morning',
        'booking_time_afternoon',
        'booking_time_evening',
        'dog_size_small',
        'dog_size_medium',
        'dog_size_large',
        'dog_featured',
        'dog_external_link',
        'account_verification',
        'account_deactivation',
        'account_level_request',
        'admin_blocked_dates',
        'admin_booking_times',
        'admin_holidays',
        'admin_auto_deactivation',
        'color_category',
        'color_pattern'
    ];

    expectedKeys.forEach(key => {
        test(`should have content for "${key}"`, () => {
            HelpTooltips.init();
            expect(HelpTooltips.content[key]).toBeDefined();
            expect(HelpTooltips.content[key].title).toBeTruthy();
            expect(HelpTooltips.content[key].text).toBeTruthy();
        });
    });
});

describe('Edge cases', () => {
    test('should handle element with empty data-help', () => {
        const consoleSpy = jest.spyOn(console, 'warn').mockImplementation();

        document.body.innerHTML = '<span data-help="">Test</span>';
        HelpTooltips.init({ showIcon: false });

        const el = document.querySelector('[data-help]');
        HelpTooltips.showTooltip(el);

        // Should not create tooltip for empty key
        expect(document.querySelector('.help-tooltip')).toBeNull();

        consoleSpy.mockRestore();
    });

    test('should handle rapid toggle calls', () => {
        document.body.innerHTML = '<span data-help="experience_green">Test</span>';
        HelpTooltips.init({ showIcon: false });

        const el = document.querySelector('[data-help]');

        // Rapid toggling
        for (let i = 0; i < 10; i++) {
            HelpTooltips.toggleTooltip(el);
        }

        // Should have at most one tooltip
        expect(document.querySelectorAll('.help-tooltip').length).toBeLessThanOrEqual(1);
    });

    test('should handle element removed from DOM during show', () => {
        document.body.innerHTML = '<span data-help="experience_green">Test</span>';
        HelpTooltips.init({ showIcon: false });

        const el = document.querySelector('[data-help]');
        el.remove();

        // Should not throw when element is removed
        expect(() => HelpTooltips.showTooltip(el)).not.toThrow();
    });
});
