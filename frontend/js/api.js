// API Client for backend communication
class API {
    constructor() {
        this.baseURL = '/api';
        this.token = localStorage.getItem('gassigeher_token');
    }

    // Set authentication token
    setToken(token) {
        this.token = token;
        if (token) {
            localStorage.setItem('gassigeher_token', token);
        } else {
            localStorage.removeItem('gassigeher_token');
        }
    }

    // Get authentication token
    getToken() {
        return this.token;
    }

    // Check if user is authenticated
    isAuthenticated() {
        return !!this.token;
    }

    // Make HTTP request
    async request(method, endpoint, data = null) {
        const headers = {
            'Content-Type': 'application/json',
        };

        if (this.token) {
            headers['Authorization'] = `Bearer ${this.token}`;
        }

        const options = {
            method,
            headers,
        };

        if (data && (method === 'POST' || method === 'PUT')) {
            options.body = JSON.stringify(data);
        }

        try {
            const response = await fetch(`${this.baseURL}${endpoint}`, options);
            const responseData = await response.json();

            if (!response.ok) {
                throw new Error(responseData.error || 'Request failed');
            }

            return responseData;
        } catch (error) {
            throw error;
        }
    }

    // Upload file
    async uploadFile(endpoint, formData) {
        const headers = {};

        if (this.token) {
            headers['Authorization'] = `Bearer ${this.token}`;
        }

        try {
            const response = await fetch(`${this.baseURL}${endpoint}`, {
                method: 'POST',
                headers,
                body: formData,
            });

            const responseData = await response.json();

            if (!response.ok) {
                throw new Error(responseData.error || 'Upload failed');
            }

            return responseData;
        } catch (error) {
            throw error;
        }
    }

    // AUTH ENDPOINTS

    async register(data) {
        return this.request('POST', '/auth/register', data);
    }

    async verifyEmail(token) {
        return this.request('POST', '/auth/verify-email', { token });
    }

    async login(email, password) {
        const response = await this.request('POST', '/auth/login', { email, password });
        if (response.token) {
            this.setToken(response.token);
        }
        return response;
    }

    async logout() {
        this.setToken(null);
        window.location.href = '/';
    }

    async forgotPassword(email) {
        return this.request('POST', '/auth/forgot-password', { email });
    }

    async resetPassword(token, password, confirmPassword) {
        return this.request('POST', '/auth/reset-password', {
            token,
            password,
            confirm_password: confirmPassword,
        });
    }

    async changePassword(oldPassword, newPassword, confirmPassword) {
        return this.request('PUT', '/auth/change-password', {
            old_password: oldPassword,
            new_password: newPassword,
            confirm_password: confirmPassword,
        });
    }

    // USER ENDPOINTS

    async getMe() {
        return this.request('GET', '/users/me');
    }

    async updateMe(data) {
        return this.request('PUT', '/users/me', data);
    }

    async uploadPhoto(file) {
        const formData = new FormData();
        formData.append('photo', file);
        return this.uploadFile('/users/me/photo', formData);
    }

    // DOG ENDPOINTS

    async getDogs(filters = {}) {
        const params = new URLSearchParams(filters);
        const endpoint = `/dogs${params.toString() ? '?' + params.toString() : ''}`;
        return this.request('GET', endpoint);
    }

    async getDog(id) {
        return this.request('GET', `/dogs/${id}`);
    }

    async getBreeds() {
        return this.request('GET', '/dogs/breeds');
    }

    async createDog(data) {
        return this.request('POST', '/dogs', data);
    }

    async updateDog(id, data) {
        return this.request('PUT', `/dogs/${id}`, data);
    }

    async deleteDog(id) {
        return this.request('DELETE', `/dogs/${id}`);
    }

    async uploadDogPhoto(dogId, file) {
        const formData = new FormData();
        formData.append('photo', file);
        return this.uploadFile(`/dogs/${dogId}/photo`, formData);
    }

    async toggleDogAvailability(dogId, isAvailable, reason = null) {
        return this.request('PUT', `/dogs/${dogId}/availability`, {
            is_available: isAvailable,
            unavailable_reason: reason,
        });
    }
}

// Global instance
window.api = new API();
