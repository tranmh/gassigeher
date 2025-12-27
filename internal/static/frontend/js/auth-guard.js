/**
 * Auth Guard - External authentication check for protected pages
 *
 * This script should be included on ALL protected pages to ensure
 * unauthenticated users are redirected to the login page.
 *
 * SECURITY: This script must be external (not inline) to comply with
 * Content Security Policy (CSP) that blocks inline scripts.
 *
 * Usage: Add <script src="/js/auth-guard.js"></script> after api.js
 *
 * Options (via data attributes on the script tag):
 *   data-require-admin="true"  - Also require admin privileges
 *   data-require-super-admin="true" - Also require super admin privileges
 */

(function() {
    'use strict';

    // Check if API is loaded
    if (typeof api === 'undefined') {
        console.error('auth-guard.js: API not loaded. Make sure api.js is included before auth-guard.js');
        return;
    }

    // Get options from script tag
    const currentScript = document.currentScript;
    const requireAdmin = currentScript?.getAttribute('data-require-admin') === 'true';
    const requireSuperAdmin = currentScript?.getAttribute('data-require-super-admin') === 'true';

    // Check authentication
    if (!api.isAuthenticated()) {
        // Not authenticated - redirect to login
        const returnUrl = encodeURIComponent(window.location.pathname + window.location.search);
        window.location.href = '/login.html?redirect=' + returnUrl;

        // Stop page execution by throwing (prevents flash of content)
        throw new Error('AuthGuard: Redirecting to login');
    }

    // If admin check is required, we need to verify with the server
    if (requireAdmin || requireSuperAdmin) {
        // Create a synchronous check by blocking with a promise
        // Note: This is handled asynchronously, page may briefly show
        // Consider using a loading overlay for better UX

        api.getMe().then(function(user) {
            if (requireSuperAdmin && !user.is_super_admin) {
                console.warn('AuthGuard: Super admin access required');
                window.location.href = '/dashboard.html';
            } else if (requireAdmin && !user.is_admin) {
                console.warn('AuthGuard: Admin access required');
                window.location.href = '/dashboard.html';
            }
        }).catch(function(error) {
            console.error('AuthGuard: Failed to verify user:', error);
            // Token might be invalid - redirect to login
            api.setToken(null);
            window.location.href = '/login.html';
        });
    }

    console.log('AuthGuard: User authenticated');
})();
