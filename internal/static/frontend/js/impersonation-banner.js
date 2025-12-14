/**
 * Impersonation Banner Component
 * Shows a red banner when super-admin is impersonating another user
 */
class ImpersonationBanner {
    /**
     * Initialize the impersonation banner
     * Call this on page load for all protected pages
     */
    static async init() {
        try {
            const response = await window.api.getMe();
            if (response && response.is_impersonating) {
                const userName = `${response.first_name} ${response.last_name}`;
                this.showBanner(userName);
            }
        } catch (error) {
            // Silently fail - user might not be logged in
            console.debug('Impersonation check failed:', error);
        }
    }

    /**
     * Show the impersonation banner
     * @param {string} userName - Name of the impersonated user
     */
    static showBanner(userName) {
        // Remove existing banner if any
        const existingBanner = document.getElementById('impersonation-banner');
        if (existingBanner) {
            existingBanner.remove();
        }

        // Create banner element
        const banner = document.createElement('div');
        banner.id = 'impersonation-banner';
        banner.innerHTML = `
            <span>
                <strong>Impersonation aktiv:</strong> Sie sind als <strong>${this.escapeHtml(userName)}</strong> angemeldet
            </span>
            <button onclick="ImpersonationBanner.endImpersonation()">
                Zurück zum Admin
            </button>
        `;

        // Add banner to top of page
        document.body.prepend(banner);

        // Add class to body for padding adjustment
        document.body.classList.add('impersonating');
    }

    /**
     * End impersonation and return to super-admin
     */
    static async endImpersonation() {
        try {
            const response = await window.api.endImpersonation();
            if (response && response.token) {
                // Set the new token (super-admin's token)
                window.api.setToken(response.token);
                // Redirect to admin dashboard
                window.location.href = '/admin-dashboard.html';
            }
        } catch (error) {
            console.error('Failed to end impersonation:', error);
            alert('Fehler beim Beenden der Impersonation: ' + (error.message || 'Unbekannter Fehler'));
        }
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
window.ImpersonationBanner = ImpersonationBanner;
