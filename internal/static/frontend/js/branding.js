/**
 * Shared Branding Loader - Fetches tenant branding and applies it across all pages.
 * Replaces hardcoded "Gassigeher" in page titles, logo text, and footer copyright
 * with the actual tenant name configured in admin-branding.html.
 *
 * Include this script in <head> after theme-loader.js on every page.
 */
(function() {
    'use strict';

    var CACHE_KEY = 'gassigeher_branding';
    var CACHE_TTL = 5 * 60 * 1000; // 5 minutes

    /**
     * Escape HTML to prevent XSS when inserting into innerHTML
     */
    function escapeHtml(str) {
        var div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    }

    /**
     * Escape string for use as .replace() replacement ($ is special in replacement patterns)
     */
    function escapeReplacement(str) {
        return str.replace(/\$/g, '$$$$');
    }

    /**
     * Apply branding data to current page DOM
     */
    function applyBranding(branding) {
        if (!branding || !branding.tenant_name) return;

        var name = branding.tenant_name;

        // Update page title: replace "Gassigeher" with tenant name
        document.title = document.title
            .replace(/Gassigeher Admin/g, name + ' Admin')
            .replace(/Gassigeher/g, name);

        // Update logo links (class="logo")
        var logos = document.querySelectorAll('a.logo');
        for (var i = 0; i < logos.length; i++) {
            var el = logos[i];
            // Skip logos with id="site-logo" - index.html manages its own
            if (el.id === 'site-logo') continue;
            var text = el.textContent;
            el.textContent = text
                .replace(/Gassigeher Admin/g, name + ' Admin')
                .replace(/Gassigeher/g, name);
        }

        // Update footer copyright text (skip elements with data-no-branding)
        var footerPs = document.querySelectorAll('footer p:not([data-no-branding])');
        var year = new Date().getFullYear();
        for (var j = 0; j < footerPs.length; j++) {
            var p = footerPs[j];
            // Skip index.html's footer-copyright (it handles its own branding)
            if (p.id === 'footer-copyright') continue;
            var safeName = escapeReplacement(escapeHtml(name));
            p.innerHTML = p.innerHTML
                .replace(/\d{4}\s+Gassigeher\b/g, year + ' ' + safeName)
                .replace(/Gassigeher Admin/g, safeName + ' Admin')
                .replace(/Gassigeher/g, safeName);
        }

        // Expose branding data globally for page-specific use
        window.branding = branding;
    }

    /**
     * Fetch branding from API with sessionStorage caching
     */
    async function loadBranding() {
        // Check sessionStorage cache first
        try {
            var cached = sessionStorage.getItem(CACHE_KEY);
            if (cached) {
                var parsed = JSON.parse(cached);
                if (Date.now() - parsed.timestamp < CACHE_TTL) {
                    applyBranding(parsed.data);
                    window.branding = parsed.data;
                    return parsed.data;
                }
            }
        } catch (e) { /* ignore parse errors */ }

        // Fetch from API
        try {
            var response = await fetch('/api/v1/tenant/branding');
            if (!response.ok) return null;
            var branding = await response.json();

            // Cache in sessionStorage
            try {
                sessionStorage.setItem(CACHE_KEY, JSON.stringify({
                    data: branding,
                    timestamp: Date.now()
                }));
            } catch (e) { /* sessionStorage may be full or unavailable */ }

            applyBranding(branding);
            return branding;
        } catch (error) {
            console.warn('Failed to load branding:', error);
            return null;
        }
    }

    /**
     * Invalidate branding cache (call after saving branding in admin)
     */
    window.invalidateBrandingCache = function() {
        try {
            sessionStorage.removeItem(CACHE_KEY);
        } catch (e) { /* ignore */ }
    };

    /**
     * Public API: load branding data (returns promise)
     */
    window.loadBrandingData = loadBranding;

    // Run on DOM ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', loadBranding);
    } else {
        loadBranding();
    }
})();
