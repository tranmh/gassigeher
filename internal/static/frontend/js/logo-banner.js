/**
 * Logo Banner - Injects site logo above navigation on all pages
 * This script fetches the current logo URL from the API and displays it
 * in a banner at the top of the page, above the navigation header.
 */
(function() {
    'use strict';

    // Default logo URL (Tierheim Goeppingen)
    const DEFAULT_LOGO = 'https://www.tierheim-goeppingen.de/wp-content/uploads/2017/04/Logo_4c_homepagebanner3.png';

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
        logoImg.alt = 'Gassigeher - Tierheim Goeppingen';

        // Try to fetch logo URL from API
        try {
            const response = await fetch('/api/settings/logo');
            if (response.ok) {
                const data = await response.json();
                logoImg.src = data.logo_url || DEFAULT_LOGO;
            } else {
                logoImg.src = DEFAULT_LOGO;
            }
        } catch (error) {
            console.warn('Failed to fetch logo setting, using default:', error);
            logoImg.src = DEFAULT_LOGO;
        }

        // Handle image load error - fallback to default
        logoImg.onerror = function() {
            if (this.src !== DEFAULT_LOGO) {
                console.warn('Logo failed to load, falling back to default');
                this.src = DEFAULT_LOGO;
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
