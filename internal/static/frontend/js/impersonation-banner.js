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
     * End impersonation and return to super-admin or central admin
     */
    static async endImpersonation() {
        try {
            // Check if this is a central admin impersonation
            const centralImpersonation = localStorage.getItem('gassigeher_impersonating');
            let response;
            let redirectUrl;

            if (centralImpersonation) {
                // Central admin impersonation - use central admin endpoint
                response = await this.endCentralImpersonation();
                redirectUrl = '/central/users.html';
                // Clean up localStorage
                localStorage.removeItem('gassigeher_impersonating');
            } else {
                // Super admin impersonation - use regular endpoint
                response = await window.api.endImpersonation();
                redirectUrl = '/admin-dashboard.html';
            }

            if (response && response.token) {
                // Set the new token
                window.api.setToken(response.token);
                // Redirect to appropriate admin dashboard
                window.location.href = redirectUrl;
            }
        } catch (error) {
            console.error('Failed to end impersonation:', error);
            alert('Fehler beim Beenden der Impersonation: ' + (error.message || 'Unbekannter Fehler'));
        }
    }

    /**
     * End central admin impersonation
     */
    static async endCentralImpersonation() {
        const token = window.api.getToken();
        if (!token) {
            throw new Error('Not authenticated');
        }

        const response = await fetch('/api/v1/central-admin/end-impersonation', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            }
        });

        if (!response.ok) {
            const error = await response.json().catch(() => ({ error: 'Unknown error' }));
            throw new Error(error.error || 'Request failed');
        }

        return response.json();
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
