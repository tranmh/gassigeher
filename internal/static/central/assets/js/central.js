// Central Admin API Client

const TOKEN_KEY = 'gassigeher_token';

// Handle token returned from impersonation end (cross-origin redirect)
(function handleReturnToken() {
    const hash = window.location.hash;
    if (!hash || !hash.startsWith('#token=')) {
        return;
    }

    try {
        const encoded = hash.substring('#token='.length);
        const token = decodeURIComponent(encoded);

        // Validate token format (JWT has 3 base64url segments)
        if (!/^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$/.test(token)) {
            console.error('Invalid token format');
            return;
        }

        // Store the token
        localStorage.setItem(TOKEN_KEY, token);

        // Clean up URL (remove hash) without triggering page reload
        history.replaceState(null, '', window.location.pathname + window.location.search);

        console.log('Token received from impersonation end');
    } catch (e) {
        console.error('Failed to parse return token:', e);
    }
})();

// Authentication
function getToken() {
    return localStorage.getItem(TOKEN_KEY);
}

function isAuthenticated() {
    return !!getToken();
}

function logout() {
    localStorage.removeItem(TOKEN_KEY);
    window.location.href = '/login.html';
}

// Get CSRF token from cookie
function getCSRFToken() {
    const match = document.cookie.match(/(?:^|; )csrf_token=([^;]*)/);
    return match ? decodeURIComponent(match[1]) : null;
}

// API Base
async function apiRequest(endpoint, options = {}) {
    const token = getToken();
    if (!token) {
        window.location.href = '/login.html';
        throw new Error('Not authenticated');
    }

    const headers = {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
        ...options.headers
    };

    // Include CSRF token for state-changing requests
    const csrfToken = getCSRFToken();
    if (csrfToken && ['POST', 'PUT', 'DELETE', 'PATCH'].includes(options.method)) {
        headers['X-CSRF-Token'] = csrfToken;
    }

    const config = {
        ...options,
        headers
    };

    const response = await fetch(`/api/v1${endpoint}`, config);

    if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error('Unauthorized');
    }

    if (!response.ok) {
        const error = await response.json().catch(() => ({ error: 'Unknown error' }));
        throw new Error(error.error || 'Request failed');
    }

    // Handle empty responses (204 No Content)
    if (response.status === 204) {
        return null;
    }

    // Try to parse JSON, handle empty or non-JSON responses gracefully
    const text = await response.text();
    if (!text) {
        return null;
    }
    try {
        return JSON.parse(text);
    } catch (e) {
        return null;
    }
}

// Central Admin API
const centralAPI = {
    // Stats
    async getStats() {
        return apiRequest('/central-admin/stats');
    },

    // Tenants
    async getTenants(search = '', activeOnly = false) {
        const params = new URLSearchParams();
        if (search) params.append('search', search);
        if (activeOnly) params.append('active_only', 'true');
        return apiRequest(`/central-admin/tenants?${params}`);
    },

    async getTenant(id) {
        return apiRequest(`/central-admin/tenants/${id}`);
    },

    async updateTenant(id, data) {
        return apiRequest(`/central-admin/tenants/${id}`, {
            method: 'PUT',
            body: JSON.stringify(data)
        });
    },

    async activateTenant(id) {
        return apiRequest(`/central-admin/tenants/${id}/activate`, {
            method: 'POST'
        });
    },

    async deactivateTenant(id) {
        return apiRequest(`/central-admin/tenants/${id}/deactivate`, {
            method: 'POST'
        });
    },

    async getTenantUsers(id) {
        return apiRequest(`/central-admin/tenants/${id}/users`);
    },

    async exportTenant(id) {
        return apiRequest(`/central-admin/tenants/${id}/export`);
    },

    // Tenant Activity
    async getInactiveTenants(days = 30) {
        return apiRequest(`/central-admin/tenants/inactive?days=${days}`);
    },

    async getTenantActivity() {
        return apiRequest('/central-admin/tenants/activity');
    },

    // Impersonation
    async impersonateUser(userId) {
        return apiRequest(`/central-admin/impersonate/${userId}`, {
            method: 'POST'
        });
    },

    async endImpersonation() {
        return apiRequest('/central-admin/end-impersonation', {
            method: 'POST'
        });
    },

    // Users
    async searchUsers(query = '', page = 1, limit = 25) {
        const params = new URLSearchParams();
        if (query) params.append('q', query);
        params.append('page', page);
        params.append('limit', limit);
        return apiRequest(`/central-admin/users/search?${params}`);
    },

    // Admins
    async getAdmins() {
        return apiRequest('/central-admin/admins');
    },

    async promoteToAdmin(userId) {
        return apiRequest(`/central-admin/admins/${userId}/promote`, {
            method: 'POST'
        });
    },

    async demoteAdmin(userId) {
        return apiRequest(`/central-admin/admins/${userId}/demote`, {
            method: 'POST'
        });
    }
};

// Utility Functions
function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function formatDate(dateStr) {
    if (!dateStr) return '-';
    const date = new Date(dateStr);
    return date.toLocaleDateString('de-DE');
}

function formatDateTime(dateStr) {
    if (!dateStr) return '-';
    const date = new Date(dateStr);
    return date.toLocaleDateString('de-DE') + ' ' + date.toLocaleTimeString('de-DE', {
        hour: '2-digit',
        minute: '2-digit'
    });
}

function showAlert(message, type = 'success') {
    const container = document.getElementById('alert-container');
    if (!container) return;

    const alert = document.createElement('div');
    alert.className = `alert alert-${type}`;
    alert.textContent = message;

    container.innerHTML = '';
    container.appendChild(alert);

    // Auto-dismiss after 5 seconds
    setTimeout(() => {
        alert.remove();
    }, 5000);
}

// Marketing API
async function getMarketingStats() {
    return apiRequest('/central-admin/marketing/stats');
}

async function getCampaigns() {
    return apiRequest('/central-admin/marketing/campaigns');
}

async function getCampaign(id) {
    return apiRequest(`/central-admin/marketing/campaigns/${id}`);
}

async function createCampaign(campaign) {
    return apiRequest('/central-admin/marketing/campaigns', {
        method: 'POST',
        body: JSON.stringify(campaign)
    });
}

async function updateCampaign(id, campaign) {
    return apiRequest(`/central-admin/marketing/campaigns/${id}`, {
        method: 'PUT',
        body: JSON.stringify(campaign)
    });
}

async function deleteCampaign(id) {
    return apiRequest(`/central-admin/marketing/campaigns/${id}`, {
        method: 'DELETE'
    });
}

async function getReferralCodes() {
    return apiRequest('/central-admin/marketing/referral-codes');
}

async function getReferralCode(id) {
    return apiRequest(`/central-admin/marketing/referral-codes/${id}`);
}

async function createReferralCode(code) {
    return apiRequest('/central-admin/marketing/referral-codes', {
        method: 'POST',
        body: JSON.stringify(code)
    });
}

async function updateReferralCode(id, code) {
    return apiRequest(`/central-admin/marketing/referral-codes/${id}`, {
        method: 'PUT',
        body: JSON.stringify(code)
    });
}

async function toggleReferralCode(id) {
    return apiRequest(`/central-admin/marketing/referral-codes/${id}/toggle`, {
        method: 'PUT'
    });
}

async function deleteReferralCode(id) {
    return apiRequest(`/central-admin/marketing/referral-codes/${id}`, {
        method: 'DELETE'
    });
}

async function getReferenceEntries() {
    return apiRequest('/central-admin/marketing/references');
}

async function approveReferenceEntry(id) {
    return apiRequest(`/central-admin/marketing/references/${id}/approve`, {
        method: 'PUT'
    });
}

async function deleteReferenceEntry(id) {
    return apiRequest(`/central-admin/marketing/references/${id}`, {
        method: 'DELETE'
    });
}

async function createReferenceEntry(entry) {
    return apiRequest('/central-admin/marketing/references', {
        method: 'POST',
        body: JSON.stringify(entry)
    });
}

async function updateReferenceEntry(id, entry) {
    return apiRequest(`/central-admin/marketing/references/${id}`, {
        method: 'PUT',
        body: JSON.stringify(entry)
    });
}
