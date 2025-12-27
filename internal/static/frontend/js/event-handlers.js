/**
 * Event Handlers Module
 * SECURITY: GASSI-2025-003 - Externalized event handlers for strict CSP compliance
 *
 * This module provides:
 * 1. Event delegation for common actions (data-action attributes)
 * 2. Common handler functions used across multiple pages
 * 3. Safe image fallback handling
 */

(function() {
    'use strict';

    // ============================================================
    // Common Handler Functions
    // ============================================================

    /**
     * Toggle mobile menu visibility
     */
    window.toggleMenu = function() {
        const nav = document.querySelector('#main-nav, #public-nav:not([style*="display: none"])');
        const overlay = document.getElementById('nav-overlay');

        if (nav && overlay) {
            nav.classList.toggle('active');
            overlay.classList.toggle('active');
        }
    };

    /**
     * Close mobile menu
     */
    window.closeMenu = function() {
        const mainNav = document.getElementById('main-nav');
        const publicNav = document.getElementById('public-nav');
        const overlay = document.getElementById('nav-overlay');

        if (mainNav) mainNav.classList.remove('active');
        if (publicNav) publicNav.classList.remove('active');
        if (overlay) overlay.classList.remove('active');
    };

    /**
     * Handle image load errors with fallback
     * @param {HTMLImageElement} img - The image element
     * @param {string} fallback - Fallback image URL
     */
    window.handleImageError = function(img, fallback) {
        if (img && img.src !== fallback) {
            img.src = fallback || '/assets/images/placeholders/dog-placeholder.svg';
        }
    };

    // ============================================================
    // Event Delegation System
    // ============================================================

    /**
     * Action handlers map - maps data-action values to handler functions
     * Pages can extend this by calling registerAction()
     */
    const actionHandlers = {
        'toggle-menu': function(e) {
            e.preventDefault();
            window.toggleMenu();
        },
        'close-menu': function(e) {
            e.preventDefault();
            window.closeMenu();
        },
        'logout': function(e) {
            e.preventDefault();
            if (window.api && typeof window.api.logout === 'function') {
                window.api.logout();
            }
        }
    };

    /**
     * Register a custom action handler
     * @param {string} action - The action name (used in data-action attribute)
     * @param {Function} handler - The handler function
     */
    window.registerAction = function(action, handler) {
        actionHandlers[action] = handler;
    };

    /**
     * Register multiple action handlers at once
     * @param {Object} handlers - Map of action names to handler functions
     */
    window.registerActions = function(handlers) {
        Object.assign(actionHandlers, handlers);
    };

    /**
     * Register change event handlers
     * These are triggered when elements with data-action-change attribute change
     * @param {Object} handlers - Map of action names to handler functions
     */
    window.registerChangeActions = function(handlers) {
        Object.assign(actionHandlers, handlers);
    };

    /**
     * Register submit event handlers
     * These are triggered when forms with data-action-submit attribute are submitted
     * @param {Object} handlers - Map of action names to handler functions
     */
    window.registerSubmitActions = function(handlers) {
        Object.assign(actionHandlers, handlers);
    };

    /**
     * Main click event delegation handler
     */
    function handleDelegatedClick(e) {
        // Find the element with data-action (could be the target or an ancestor)
        let target = e.target;
        while (target && target !== document) {
            const action = target.getAttribute('data-action');
            if (action && actionHandlers[action]) {
                // Get optional data attributes
                const actionId = target.getAttribute('data-id');
                const actionData = target.getAttribute('data-value');

                // Call the handler with context
                actionHandlers[action].call(target, e, actionId, actionData);
                return;
            }
            target = target.parentElement;
        }
    }

    /**
     * Handle change events for data-action-change elements
     */
    function handleDelegatedChange(e) {
        const target = e.target;
        const action = target.getAttribute('data-action-change');
        if (action && actionHandlers[action]) {
            const actionId = target.getAttribute('data-id');
            actionHandlers[action].call(target, e, actionId, target.value);
        }
    }

    /**
     * Handle submit events for forms with data-action-submit
     */
    function handleDelegatedSubmit(e) {
        const form = e.target;
        const action = form.getAttribute('data-action-submit');
        if (action && actionHandlers[action]) {
            e.preventDefault();
            actionHandlers[action].call(form, e);
        }
    }

    /**
     * Handle image errors using event delegation
     */
    function handleImageErrors(e) {
        if (e.target.tagName === 'IMG') {
            const fallback = e.target.getAttribute('data-fallback');
            if (fallback && e.target.src !== fallback) {
                e.target.src = fallback;
            }
        }
    }

    // ============================================================
    // Initialization
    // ============================================================

    /**
     * Initialize event delegation when DOM is ready
     */
    function init() {
        // Click delegation
        document.addEventListener('click', handleDelegatedClick);

        // Change delegation
        document.addEventListener('change', handleDelegatedChange);

        // Submit delegation
        document.addEventListener('submit', handleDelegatedSubmit);

        // Image error handling
        document.addEventListener('error', handleImageErrors, true);

        // Close menu when clicking nav links
        document.addEventListener('click', function(e) {
            if (e.target.tagName === 'A' && e.target.closest('nav')) {
                const href = e.target.getAttribute('href');
                // Don't close if it's a hash link or has data-action
                if (href && href !== '#' && !e.target.hasAttribute('data-action')) {
                    window.closeMenu();
                }
            }
        });
    }

    // Initialize on DOMContentLoaded
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

})();
