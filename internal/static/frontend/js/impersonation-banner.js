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

                // Check for tenant info from central admin impersonation
                let tenantInfo = null;
                const impersonationJson = localStorage.getItem('gassigeher_impersonating');
                if (impersonationJson) {
                    try {
                        const impersonation = JSON.parse(impersonationJson);
                        if (impersonation.tenantName) {
                            tenantInfo = impersonation.tenantName;
                        }
                    } catch (e) {
                        // Ignore parse errors
                    }
                }

                this.showBanner(userName, tenantInfo);
            }
        } catch (error) {
            // Silently fail - user might not be logged in
            console.debug('Impersonation check failed:', error);
        }
    }

    /**
     * Show the impersonation banner
     * @param {string} userName - Name of the impersonated user
     * @param {string|null} tenantName - Name of the tenant (for central admin impersonation)
     */
    static showBanner(userName, tenantName = null) {
        // Remove existing banner if any
        const existingBanner = document.getElementById('impersonation-banner');
        if (existingBanner) {
            existingBanner.remove();
        }

        // Build banner message
        let message = `<strong>Impersonation aktiv:</strong> Sie sind als <strong>${this.escapeHtml(userName)}</strong>`;
        if (tenantName) {
            message += ` bei <strong>${this.escapeHtml(tenantName)}</strong>`;
        }
        message += ' angemeldet';

        // Create banner element
        const banner = document.createElement('div');
        banner.id = 'impersonation-banner';
        banner.innerHTML = `
            <span>${message}</span>
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
            const centralImpersonationJson = localStorage.getItem('gassigeher_impersonating');
            let response;

            if (centralImpersonationJson) {
                // Central admin impersonation - use central admin endpoint
                response = await this.endCentralImpersonation();

                // Parse impersonation info to get central admin origin
                let centralAdminOrigin = null;
                try {
                    const impersonationInfo = JSON.parse(centralImpersonationJson);
                    centralAdminOrigin = impersonationInfo.centralAdminOrigin;
                } catch (e) {
                    console.error('Failed to parse impersonation info:', e);
                }

                // Clean up localStorage on this tenant origin
                localStorage.removeItem('gassigeher_impersonating');
                localStorage.removeItem('gassigeher_token');

                if (response && response.token) {
                    if (centralAdminOrigin && centralAdminOrigin !== window.location.origin) {
                        // Cross-origin: Pass token back to central admin via URL hash
                        const tokenData = encodeURIComponent(response.token);
                        window.location.href = centralAdminOrigin + '/central/users.html#token=' + tokenData;
                    } else {
                        // Same origin: Just set token and redirect
                        window.api.setToken(response.token);
                        window.location.href = '/central/users.html';
                    }
                }
            } else {
                // Super admin impersonation - use regular endpoint (same origin)
                response = await window.api.endImpersonation();
                if (response && response.token) {
                    window.api.setToken(response.token);
                    window.location.href = '/admin-dashboard.html';
                }
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

        // Get CSRF token from cookie
        const csrfToken = this.getCSRFToken();

        const headers = {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`
        };

        // Add CSRF token if available
        if (csrfToken) {
            headers['X-CSRF-Token'] = csrfToken;
        }

        const response = await fetch('/api/v1/central-admin/end-impersonation', {
            method: 'POST',
            headers: headers
        });

        if (!response.ok) {
            const error = await response.json().catch(() => ({ error: 'Unknown error' }));
            throw new Error(error.error || 'Request failed');
        }

        return response.json();
    }

    /**
     * Get CSRF token from cookie
     */
    static getCSRFToken() {
        const match = document.cookie.match(/(?:^|; )csrf_token=([^;]*)/);
        return match ? decodeURIComponent(match[1]) : null;
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
