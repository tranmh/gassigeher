/**
 * Logo Banner - Injects site logo above navigation on all pages
 * This script fetches the current logo URL from the API and displays it
 * in a banner at the top of the page, above the navigation header.
 */
(function() {
    'use strict';

    // Placeholder logo URL for tenants without custom logo
    const PLACEHOLDER_LOGO = '/assets/images/placeholders/logo-placeholder.svg';

    /**
     * Creates and injects the logo banner into the page
     */
    async function initLogoBanner() {
        // Create banner container
        const banner = document.createElement('div');
        banner.id = 'logo-banner';
        banner.className = 'logo-banner';

        // Create link wrapper (links to homepage)
        const logoLink = document.createElement('a');
        logoLink.href = '/';
        logoLink.className = 'logo-banner-link';
        logoLink.setAttribute('aria-label', 'Zur Startseite');

        // Create logo image
        const logoImg = document.createElement('img');
        logoImg.className = 'logo-banner-img';
        logoImg.alt = 'Gassigeher';

        // Try to fetch logo URL from API
        try {
            const response = await fetch('/api/v1/settings/logo');
            if (response.ok) {
                const data = await response.json();
                logoImg.src = data.logo_url || PLACEHOLDER_LOGO;
            } else {
                logoImg.src = PLACEHOLDER_LOGO;
            }
        } catch (error) {
            console.warn('Failed to fetch logo setting, using placeholder:', error);
            logoImg.src = PLACEHOLDER_LOGO;
        }

        // Handle image load error - fallback to placeholder
        logoImg.onerror = function() {
            if (this.src !== PLACEHOLDER_LOGO) {
                console.warn('Logo failed to load, falling back to placeholder');
                this.src = PLACEHOLDER_LOGO;
            }
        };

        // Assemble the banner
        logoLink.appendChild(logoImg);
        banner.appendChild(logoLink);

        // Insert at beginning of body (before header)
        const body = document.body;
        const header = document.querySelector('header');
        if (header) {
            body.insertBefore(banner, header);
        } else {
            // Fallback: insert as first child of body
            body.insertBefore(banner, body.firstChild);
        }
    }

    // Initialize on DOM ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', initLogoBanner);
    } else {
        // DOM is already ready
        initLogoBanner();
    }
})();
