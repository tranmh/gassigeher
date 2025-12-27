// Theme Loader - Injects dynamic theme CSS from tenant settings
// This script should be loaded early in all pages

(function() {
    'use strict';

    // Create and inject the theme CSS link
    function loadThemeCSS() {
        // Check if already loaded
        if (document.getElementById('theme-css')) {
            return;
        }

        const link = document.createElement('link');
        link.id = 'theme-css';
        link.rel = 'stylesheet';
        // Add timestamp to bust cache after branding changes
        const cacheBuster = localStorage.getItem('gassigeher_theme_version') || Date.now();
        link.href = '/api/v1/theme/css?v=' + cacheBuster;

        // Insert after main.css to ensure theme overrides work
        const mainCSS = document.querySelector('link[href*="main.css"]');
        if (mainCSS && mainCSS.parentNode) {
            mainCSS.parentNode.insertBefore(link, mainCSS.nextSibling);
        } else {
            document.head.appendChild(link);
        }
    }

    // Reload theme CSS (call after saving branding)
    window.reloadThemeCSS = function() {
        const newVersion = Date.now();
        localStorage.setItem('gassigeher_theme_version', newVersion);

        const existingLink = document.getElementById('theme-css');
        if (existingLink) {
            existingLink.href = '/api/v1/theme/css?v=' + newVersion;
        } else {
            loadThemeCSS();
        }
    };

    // Load theme immediately
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', loadThemeCSS);
    } else {
        loadThemeCSS();
    }
})();
