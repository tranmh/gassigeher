// Central Admin API Client

const TOKEN_KEY = 'gassigeher_token';

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

// API Base
async function apiRequest(endpoint, options = {}) {
    const token = getToken();
    if (!token) {
        window.location.href = '/login.html';
        throw new Error('Not authenticated');
    }

    const config = {
        ...options,
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`,
            ...options.headers
        }
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

    // Users
    async searchUsers(query) {
        return apiRequest(`/central-admin/users/search?q=${encodeURIComponent(query)}`);
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
