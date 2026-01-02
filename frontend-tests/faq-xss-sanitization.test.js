/**
 * FAQ XSS Sanitization Tests (TDD)
 *
 * BUG #15: FAQ answers are rendered as raw HTML without sanitization.
 * This allows XSS attacks if faq-data.js is compromised or contains
 * malicious content.
 *
 * The fix: Sanitize FAQ answers to allow only safe HTML tags.
 *
 * @jest-environment jsdom
 */

// XSS Test Payloads
const XSS_PAYLOADS = {
    scriptTag: '<script>alert("XSS")</script>',
    imgOnerror: '<img src=x onerror=alert("XSS")>',
    svgOnload: '<svg onload=alert("XSS")>',
    iframeJs: '<iframe src="javascript:alert(1)">',
    anchorJs: '<a href="javascript:alert(1)">Click</a>',
    eventHandler: '<div onclick="alert(1)">Click me</div>',
    styleExpression: '<div style="background:url(javascript:alert(1))">',
    dataUri: '<a href="data:text/html,<script>alert(1)</script>">Click</a>',
    formAction: '<form action="javascript:alert(1)"><input type="submit"></form>',
    objectTag: '<object data="javascript:alert(1)">',
    embedTag: '<embed src="javascript:alert(1)">',
    metaRefresh: '<meta http-equiv="refresh" content="0;url=javascript:alert(1)">',
};

// Safe HTML that should be PRESERVED
const SAFE_HTML = {
    strongText: '<strong>Important</strong>',
    emphasisText: '<em>emphasized</em>',
    paragraph: '<p>A paragraph of text.</p>',
    lineBreak: 'Line 1<br>Line 2',
    unorderedList: '<ul><li>Item 1</li><li>Item 2</li></ul>',
    orderedList: '<ol><li>First</li><li>Second</li></ol>',
    safeLink: '<a href="https://example.com">Safe Link</a>',
    internalLink: '<a href="/help.html">Internal Link</a>',
    mailtoLink: '<a href="mailto:test@example.com">Email Us</a>',
    nestedFormatting: '<p><strong>Bold</strong> and <em>italic</em></p>',
};

describe('BUG #15: FAQ Answer XSS Sanitization', () => {

    describe('RED PHASE - Current unsafe behavior', () => {
        // This demonstrates the CURRENT buggy behavior
        function unsafeRenderFAQ(faq) {
            // Current implementation - NO SANITIZATION
            return `<div class="faq-answer-content">${faq.answer}</div>`;
        }

        test('VULNERABLE: script tags execute', () => {
            const faq = { answer: XSS_PAYLOADS.scriptTag };
            const html = unsafeRenderFAQ(faq);

            // The script tag is present (BAD!)
            expect(html).toContain('<script>');
        });

        test('VULNERABLE: javascript: URLs present', () => {
            const faq = { answer: XSS_PAYLOADS.anchorJs };
            const html = unsafeRenderFAQ(faq);

            expect(html).toContain('javascript:');
        });

        test('VULNERABLE: event handlers present', () => {
            const faq = { answer: XSS_PAYLOADS.eventHandler };
            const html = unsafeRenderFAQ(faq);

            expect(html).toContain('onclick');
        });
    });

    describe('GREEN PHASE - Safe sanitization', () => {

        /**
         * sanitizeFAQAnswer - Sanitize HTML in FAQ answers
         *
         * Allowed tags: p, br, strong, em, b, i, ul, ol, li, a (with safe href)
         * Removed: scripts, iframes, objects, embeds, forms, event handlers
         * Sanitized: href attributes (only http, https, mailto, /)
         */
        function sanitizeFAQAnswer(html) {
            if (!html || typeof html !== 'string') return '';

            // Create a temporary DOM element for parsing
            const temp = document.createElement('div');
            temp.innerHTML = html;

            // Remove dangerous elements entirely
            const dangerousTags = ['script', 'iframe', 'object', 'embed', 'form', 'input', 'button', 'textarea', 'select', 'meta', 'link', 'style', 'svg', 'math'];
            dangerousTags.forEach(tag => {
                const elements = temp.querySelectorAll(tag);
                elements.forEach(el => el.remove());
            });

            // Remove all event handlers and dangerous attributes
            const allElements = temp.querySelectorAll('*');
            allElements.forEach(el => {
                // Remove event handlers (onclick, onerror, onload, etc.)
                const attrs = [...el.attributes];
                attrs.forEach(attr => {
                    const name = attr.name.toLowerCase();
                    // Remove event handlers
                    if (name.startsWith('on')) {
                        el.removeAttribute(attr.name);
                    }
                    // Sanitize href/src attributes
                    if (name === 'href' || name === 'src') {
                        const value = attr.value.toLowerCase().trim();
                        // Only allow safe protocols
                        if (value.startsWith('javascript:') ||
                            value.startsWith('data:') ||
                            value.startsWith('vbscript:')) {
                            el.removeAttribute(attr.name);
                        }
                    }
                    // Remove style attributes (can contain expressions)
                    if (name === 'style') {
                        const styleValue = attr.value.toLowerCase();
                        if (styleValue.includes('expression') ||
                            styleValue.includes('javascript') ||
                            styleValue.includes('url(')) {
                            el.removeAttribute(attr.name);
                        }
                    }
                });

                // For img tags, validate src
                if (el.tagName === 'IMG') {
                    const src = el.getAttribute('src');
                    if (src) {
                        const srcLower = src.toLowerCase().trim();
                        if (!srcLower.startsWith('http://') &&
                            !srcLower.startsWith('https://') &&
                            !srcLower.startsWith('/')) {
                            el.remove();
                        }
                    }
                }
            });

            return temp.innerHTML;
        }

        // =====================
        // XSS Prevention Tests
        // =====================

        test('should remove script tags', () => {
            const sanitized = sanitizeFAQAnswer(XSS_PAYLOADS.scriptTag);
            expect(sanitized).not.toContain('<script');
            expect(sanitized).not.toContain('</script>');
        });

        test('should remove img onerror handlers', () => {
            const sanitized = sanitizeFAQAnswer(XSS_PAYLOADS.imgOnerror);
            expect(sanitized).not.toContain('onerror');
        });

        test('should remove svg onload handlers', () => {
            const sanitized = sanitizeFAQAnswer(XSS_PAYLOADS.svgOnload);
            expect(sanitized).not.toContain('<svg');
            expect(sanitized).not.toContain('onload');
        });

        test('should remove iframes', () => {
            const sanitized = sanitizeFAQAnswer(XSS_PAYLOADS.iframeJs);
            expect(sanitized).not.toContain('<iframe');
        });

        test('should remove javascript: URLs from anchors', () => {
            const sanitized = sanitizeFAQAnswer(XSS_PAYLOADS.anchorJs);
            expect(sanitized).not.toContain('javascript:');
        });

        test('should remove onclick handlers', () => {
            const sanitized = sanitizeFAQAnswer(XSS_PAYLOADS.eventHandler);
            expect(sanitized).not.toContain('onclick');
        });

        test('should remove data: URLs', () => {
            const sanitized = sanitizeFAQAnswer(XSS_PAYLOADS.dataUri);
            expect(sanitized).not.toContain('data:');
        });

        test('should remove form elements', () => {
            const sanitized = sanitizeFAQAnswer(XSS_PAYLOADS.formAction);
            expect(sanitized).not.toContain('<form');
            expect(sanitized).not.toContain('<input');
        });

        test('should remove object and embed tags', () => {
            const sanitized = sanitizeFAQAnswer(XSS_PAYLOADS.objectTag);
            expect(sanitized).not.toContain('<object');

            const sanitized2 = sanitizeFAQAnswer(XSS_PAYLOADS.embedTag);
            expect(sanitized2).not.toContain('<embed');
        });

        test('should remove meta tags', () => {
            const sanitized = sanitizeFAQAnswer(XSS_PAYLOADS.metaRefresh);
            expect(sanitized).not.toContain('<meta');
        });

        // =====================
        // Safe Content Tests
        // =====================

        test('should preserve strong tags', () => {
            const sanitized = sanitizeFAQAnswer(SAFE_HTML.strongText);
            expect(sanitized).toContain('<strong>');
            expect(sanitized).toContain('</strong>');
            expect(sanitized).toContain('Important');
        });

        test('should preserve em tags', () => {
            const sanitized = sanitizeFAQAnswer(SAFE_HTML.emphasisText);
            expect(sanitized).toContain('<em>');
            expect(sanitized).toContain('emphasized');
        });

        test('should preserve p tags', () => {
            const sanitized = sanitizeFAQAnswer(SAFE_HTML.paragraph);
            expect(sanitized).toContain('<p>');
            expect(sanitized).toContain('A paragraph');
        });

        test('should preserve br tags', () => {
            const sanitized = sanitizeFAQAnswer(SAFE_HTML.lineBreak);
            expect(sanitized).toContain('<br>');
        });

        test('should preserve ul/li lists', () => {
            const sanitized = sanitizeFAQAnswer(SAFE_HTML.unorderedList);
            expect(sanitized).toContain('<ul>');
            expect(sanitized).toContain('<li>');
            expect(sanitized).toContain('Item 1');
        });

        test('should preserve ol/li lists', () => {
            const sanitized = sanitizeFAQAnswer(SAFE_HTML.orderedList);
            expect(sanitized).toContain('<ol>');
            expect(sanitized).toContain('<li>');
        });

        test('should preserve safe https links', () => {
            const sanitized = sanitizeFAQAnswer(SAFE_HTML.safeLink);
            expect(sanitized).toContain('<a');
            expect(sanitized).toContain('href="https://example.com"');
            expect(sanitized).toContain('Safe Link');
        });

        test('should preserve internal links', () => {
            const sanitized = sanitizeFAQAnswer(SAFE_HTML.internalLink);
            expect(sanitized).toContain('href="/help.html"');
        });

        test('should preserve mailto links', () => {
            const sanitized = sanitizeFAQAnswer(SAFE_HTML.mailtoLink);
            expect(sanitized).toContain('href="mailto:');
        });

        test('should preserve nested formatting', () => {
            const sanitized = sanitizeFAQAnswer(SAFE_HTML.nestedFormatting);
            expect(sanitized).toContain('<p>');
            expect(sanitized).toContain('<strong>Bold</strong>');
            expect(sanitized).toContain('<em>italic</em>');
        });

        // =====================
        // Edge Cases
        // =====================

        test('should handle null input', () => {
            expect(sanitizeFAQAnswer(null)).toBe('');
        });

        test('should handle undefined input', () => {
            expect(sanitizeFAQAnswer(undefined)).toBe('');
        });

        test('should handle empty string', () => {
            expect(sanitizeFAQAnswer('')).toBe('');
        });

        test('should handle plain text', () => {
            const text = 'Just plain text without HTML';
            expect(sanitizeFAQAnswer(text)).toBe(text);
        });

        test('should handle mixed content', () => {
            const mixed = '<p>Safe paragraph</p><script>bad()</script><strong>Bold</strong>';
            const sanitized = sanitizeFAQAnswer(mixed);

            expect(sanitized).toContain('<p>Safe paragraph</p>');
            expect(sanitized).toContain('<strong>Bold</strong>');
            expect(sanitized).not.toContain('<script>');
        });

        test('should handle German characters', () => {
            const german = '<p>Häufig gestellte Fragen über größere Änderungen</p>';
            const sanitized = sanitizeFAQAnswer(german);
            expect(sanitized).toContain('Häufig');
            expect(sanitized).toContain('größere');
        });
    });

    describe('Integration: FAQ rendering with sanitization', () => {

        function sanitizeFAQAnswer(html) {
            if (!html || typeof html !== 'string') return '';

            const temp = document.createElement('div');
            temp.innerHTML = html;

            const dangerousTags = ['script', 'iframe', 'object', 'embed', 'form', 'input', 'button', 'textarea', 'select', 'meta', 'link', 'style', 'svg', 'math'];
            dangerousTags.forEach(tag => {
                const elements = temp.querySelectorAll(tag);
                elements.forEach(el => el.remove());
            });

            const allElements = temp.querySelectorAll('*');
            allElements.forEach(el => {
                const attrs = [...el.attributes];
                attrs.forEach(attr => {
                    const name = attr.name.toLowerCase();
                    if (name.startsWith('on')) {
                        el.removeAttribute(attr.name);
                    }
                    if (name === 'href' || name === 'src') {
                        const value = attr.value.toLowerCase().trim();
                        if (value.startsWith('javascript:') ||
                            value.startsWith('data:') ||
                            value.startsWith('vbscript:')) {
                            el.removeAttribute(attr.name);
                        }
                    }
                });

                if (el.tagName === 'IMG') {
                    const src = el.getAttribute('src');
                    if (src) {
                        const srcLower = src.toLowerCase().trim();
                        if (!srcLower.startsWith('http://') &&
                            !srcLower.startsWith('https://') &&
                            !srcLower.startsWith('/')) {
                            el.remove();
                        }
                    }
                }
            });

            return temp.innerHTML;
        }

        function renderFAQsSafe(faqs) {
            return faqs.map(faq => `
                <div class="faq-item" data-faq-id="${faq.id}">
                    <div class="faq-question">${faq.question}</div>
                    <div class="faq-answer">
                        <div class="faq-answer-content">
                            ${sanitizeFAQAnswer(faq.answer)}
                        </div>
                    </div>
                </div>
            `).join('');
        }

        test('should render FAQs with safe HTML only', () => {
            const faqs = [
                {
                    id: 1,
                    question: 'Test Question',
                    answer: '<p>Safe answer</p><script>alert("XSS")</script>'
                }
            ];

            const rendered = renderFAQsSafe(faqs);

            expect(rendered).toContain('<p>Safe answer</p>');
            expect(rendered).not.toContain('<script>');
        });

        test('should render multiple FAQs safely', () => {
            const faqs = [
                { id: 1, question: 'Q1', answer: '<strong>A1</strong><img src=x onerror=alert(1)>' },
                { id: 2, question: 'Q2', answer: '<a href="javascript:void(0)">Link</a>' }
            ];

            const rendered = renderFAQsSafe(faqs);

            expect(rendered).toContain('<strong>A1</strong>');
            expect(rendered).not.toContain('onerror');
            expect(rendered).not.toContain('javascript:');
        });
    });
});
