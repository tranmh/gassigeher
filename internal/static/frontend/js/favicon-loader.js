/**
 * Favicon Loader - Dynamically loads tenant-specific favicon
 * This script fetches the current favicon URL from the API and updates
 * the browser tab icon accordingly.
 */
(function() {
    'use strict';

    // Placeholder favicon URL for tenants without custom favicon
    const PLACEHOLDER_FAVICON = '/assets/images/placeholders/favicon-placeholder.svg';

    // Favicon link element ID
    const FAVICON_LINK_ID = 'site-favicon';

    /**
     * Gets or creates the favicon link element
     * @returns {HTMLLinkElement} The favicon link element
     */
    function getFaviconLink() {
        let link = document.getElementById(FAVICON_LINK_ID);
        if (!link) {
            link = document.createElement('link');
            link.id = FAVICON_LINK_ID;
            link.rel = 'icon';
            document.head.appendChild(link);
        }
        return link;
    }

    /**
     * Updates the favicon URL
     * @param {string} url - The favicon URL
     */
    function setFavicon(url) {
        const link = getFaviconLink();

        // Determine type based on URL
        if (url.endsWith('.svg')) {
            link.type = 'image/svg+xml';
        } else if (url.endsWith('.png')) {
            link.type = 'image/png';
        } else if (url.endsWith('.ico')) {
            link.type = 'image/x-icon';
        } else {
            link.type = 'image/png'; // Default to PNG
        }

        // Add cache buster for dynamic updates
        const cacheBuster = '?t=' + Date.now();
        link.href = url + cacheBuster;
    }

    /**
     * Fetches and applies the favicon from the API
     */
    async function loadFavicon() {
        try {
            const response = await fetch('/api/v1/settings/favicon');
            if (response.ok) {
                const data = await response.json();
                setFavicon(data.favicon_url || PLACEHOLDER_FAVICON);
            } else {
                setFavicon(PLACEHOLDER_FAVICON);
            }
        } catch (error) {
            console.warn('Failed to fetch favicon setting, using placeholder:', error);
            setFavicon(PLACEHOLDER_FAVICON);
        }
    }

    /**
     * Reloads the favicon from the API
     * This can be called after admin updates the favicon
     */
    window.reloadFavicon = function() {
        loadFavicon();
    };

    // Initialize on DOM ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', loadFavicon);
    } else {
        // DOM is already ready
        loadFavicon();
    }
})();
