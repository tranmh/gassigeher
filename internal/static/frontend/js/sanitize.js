/**
 * Sanitization utility for preventing XSS attacks
 * Converts user-controlled strings to safe HTML-escaped text
 */

/**
 * Escapes HTML special characters to prevent XSS
 * @param {string} str - The string to sanitize
 * @returns {string} The sanitized string safe for HTML insertion
 */
function sanitizeHTML(str) {
    if (!str) return '';
    if (typeof str !== 'string') str = String(str);

    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
}

/**
 * Safely inserts text content into an HTML element
 * Replaces the element's content with text (not HTML)
 * @param {HTMLElement} element - The element to update
 * @param {string} text - The text to insert
 */
function setTextContent(element, text) {
    if (!element) return;
    element.textContent = text || '';
}

/**
 * Creates a safe HTML element with text content
 * @param {string} tag - HTML tag name (e.g., 'div', 'span', 'p')
 * @param {string} text - Text content (will be safely escaped)
 * @param {Object} attributes - Optional attributes object
 * @returns {HTMLElement} The created element
 */
function createSafeElement(tag, text, attributes = {}) {
    const element = document.createElement(tag);
    element.textContent = text;

    for (const [key, value] of Object.entries(attributes)) {
        if (key === 'class') {
            element.className = value;
        } else if (key === 'style') {
            Object.assign(element.style, value);
        } else {
            element.setAttribute(key, value);
        }
    }

    return element;
}

// Backward compatibility and global export
window.sanitizeHTML = sanitizeHTML;
window.setTextContent = setTextContent;
window.createSafeElement = createSafeElement;
