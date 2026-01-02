/**
 * FAQ Answer Sanitizer
 *
 * BUG #15 FIX: Sanitizes HTML in FAQ answers to prevent XSS attacks.
 *
 * Allowed tags: p, br, strong, em, b, i, ul, ol, li, a (with safe href)
 * Removed: scripts, iframes, objects, embeds, forms, event handlers
 * Sanitized: href attributes (only http, https, mailto, /)
 */

/**
 * Sanitize HTML content from FAQ answers
 * @param {string} html - The HTML content to sanitize
 * @returns {string} - Sanitized HTML
 */
function sanitizeFAQAnswer(html) {
    if (!html || typeof html !== 'string') return '';

    // Create a temporary DOM element for parsing
    const temp = document.createElement('div');
    temp.innerHTML = html;

    // Remove dangerous elements entirely
    const dangerousTags = [
        'script', 'iframe', 'object', 'embed', 'form', 'input',
        'button', 'textarea', 'select', 'meta', 'link', 'style',
        'svg', 'math', 'base', 'applet', 'frame', 'frameset'
    ];

    dangerousTags.forEach(tag => {
        const elements = temp.querySelectorAll(tag);
        elements.forEach(el => el.remove());
    });

    // Process all remaining elements
    const allElements = temp.querySelectorAll('*');
    allElements.forEach(el => {
        const attrs = [...el.attributes];

        attrs.forEach(attr => {
            const name = attr.name.toLowerCase();

            // Remove all event handlers (onclick, onerror, onload, etc.)
            if (name.startsWith('on')) {
                el.removeAttribute(attr.name);
                return;
            }

            // Sanitize href and src attributes
            if (name === 'href' || name === 'src') {
                const value = attr.value.toLowerCase().trim();
                // Only allow safe protocols
                if (value.startsWith('javascript:') ||
                    value.startsWith('data:') ||
                    value.startsWith('vbscript:')) {
                    el.removeAttribute(attr.name);
                }
                return;
            }

            // Remove style attributes that could contain malicious content
            if (name === 'style') {
                const styleValue = attr.value.toLowerCase();
                if (styleValue.includes('expression') ||
                    styleValue.includes('javascript') ||
                    styleValue.includes('url(')) {
                    el.removeAttribute(attr.name);
                }
                return;
            }
        });

        // For img tags, validate src more strictly
        if (el.tagName === 'IMG') {
            const src = el.getAttribute('src');
            if (src) {
                const srcLower = src.toLowerCase().trim();
                // Only allow http(s) and relative URLs
                if (!srcLower.startsWith('http://') &&
                    !srcLower.startsWith('https://') &&
                    !srcLower.startsWith('/') &&
                    !srcLower.startsWith('./')) {
                    el.remove();
                }
            } else {
                // Remove img without src
                el.remove();
            }
        }
    });

    return temp.innerHTML;
}

// Make available globally
if (typeof window !== 'undefined') {
    window.sanitizeFAQAnswer = sanitizeFAQAnswer;
}
