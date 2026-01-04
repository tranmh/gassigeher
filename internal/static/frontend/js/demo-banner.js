/**
 * Demo Banner Component
 * Shows an info banner when user is on the demo tenant
 */
class DemoBanner {
    /**
     * Initialize the demo banner
     * Call this on page load for all pages
     */
    static async init() {
        try {
            // Check if we're on the demo tenant by checking the subdomain
            const hostname = window.location.hostname;
            const isDemo = hostname.startsWith('demo.') || hostname === 'demo';

            if (isDemo) {
                // Fetch demo status to get next reset time
                const response = await fetch('/api/v1/demo/status');
                if (response.ok) {
                    const data = await response.json();
                    if (data.is_demo) {
                        this.showBanner(data.next_reset_at);
                    }
                } else {
                    // Show banner without reset time
                    this.showBanner(null);
                }
            }
        } catch (error) {
            // Silently fail - still show basic banner if on demo subdomain
            console.debug('Demo status check failed:', error);
            const hostname = window.location.hostname;
            if (hostname.startsWith('demo.') || hostname === 'demo') {
                this.showBanner(null);
            }
        }
    }

    /**
     * Show the demo banner
     * @param {string|null} nextReset - Next reset time (formatted string) or null
     */
    static showBanner(nextReset) {
        // Remove existing banner if any
        const existingBanner = document.getElementById('demo-banner');
        if (existingBanner) {
            existingBanner.remove();
        }

        // Create banner element
        const banner = document.createElement('div');
        banner.id = 'demo-banner';

        let resetText = '';
        if (nextReset) {
            resetText = ` | <span class="reset-info">Reset: ${this.escapeHtml(nextReset)}</span>`;
        }

        // Build absolute URL to main domain for demo page
        // demo.gassigeher.org → gassigeher.org/demo
        // demo.gassigeher.local:8080 → gassigeher.local:8080/demo
        const host = window.location.host; // includes port if present
        const hostname = window.location.hostname;
        const port = window.location.port;
        const hostParts = hostname.split('.');
        // Remove 'demo' subdomain to get base domain
        let baseDomain = hostParts.length > 2
            ? hostParts.slice(-2).join('.')
            : hostParts.join('.');
        // Add port back if present
        if (port) {
            baseDomain += ':' + port;
        }
        const demoPageUrl = `${window.location.protocol}//${baseDomain}/demo`;

        banner.innerHTML = `
            <div class="demo-banner-content">
                <span class="demo-label">DEMO</span>
                <span class="demo-text">
                    Dies ist eine Demo-Umgebung. Alle Daten werden taeglich zurueckgesetzt.${resetText}
                </span>
                <a href="${demoPageUrl}" class="demo-link" target="_blank">Zugangsdaten anzeigen</a>
            </div>
        `;

        // Add banner to top of page
        document.body.prepend(banner);

        // Add class to body for padding adjustment
        document.body.classList.add('demo-mode');
    }

    /**
     * Escape HTML to prevent XSS
     * @param {string} text - Text to escape
     * @returns {string} Escaped text
     */
    static escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

// Make it globally available
window.DemoBanner = DemoBanner;

// Auto-initialize on DOMContentLoaded
document.addEventListener('DOMContentLoaded', () => {
    DemoBanner.init();
});
