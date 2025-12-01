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
                // Create error with response data attached
                const error = new Error(responseData.error || 'Request failed');
                error.status = response.status;
                error.data = responseData;
                throw error;
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

    async deleteDog(id, force = false) {
        const endpoint = force ? `/dogs/${id}?force=true` : `/dogs/${id}`;
        return this.request('DELETE', endpoint);
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

    async setDogFeatured(dogId, isFeatured) {
        return this.request('PUT', `/dogs/${dogId}/featured`, {
            is_featured: isFeatured,
        });
    }

    async getFeaturedDogs() {
        return this.request('GET', '/dogs/featured');
    }

    // BOOKING ENDPOINTS

    async createBooking(data) {
        return this.request('POST', '/bookings', data);
    }

    async getBookings(filters = {}) {
        const params = new URLSearchParams(filters);
        const endpoint = `/bookings${params.toString() ? '?' + params.toString() : ''}`;
        return this.request('GET', endpoint);
    }

    async getBooking(id) {
        return this.request('GET', `/bookings/${id}`);
    }

    async cancelBooking(id, reason = null) {
        return this.request('PUT', `/bookings/${id}/cancel`, { reason });
    }

    async moveBooking(id, date, scheduledTime, reason) {
        return this.request('PUT', `/bookings/${id}/move`, {
            date,
            scheduled_time: scheduledTime,
            reason
        });
    }

    async addBookingNotes(id, notes) {
        return this.request('PUT', `/bookings/${id}/notes`, { notes });
    }

    async getCalendarData(year, month) {
        return this.request('GET', `/bookings/calendar/${year}/${month}`);
    }

    // BLOCKED DATES ENDPOINTS

    async getBlockedDates() {
        return this.request('GET', '/blocked-dates');
    }

    async createBlockedDate(date, reason) {
        return this.request('POST', '/blocked-dates', { date, reason });
    }

    async deleteBlockedDate(id) {
        return this.request('DELETE', `/blocked-dates/${id}`);
    }

    // SETTINGS ENDPOINTS (Admin only)

    async getSettings() {
        return this.request('GET', '/settings');
    }

    async updateSetting(key, value) {
        return this.request('PUT', `/settings/${key}`, { value });
    }

    // EXPERIENCE REQUEST ENDPOINTS

    async createExperienceRequest(requestedLevel) {
        return this.request('POST', '/experience-requests', { requested_level: requestedLevel });
    }

    async getExperienceRequests() {
        return this.request('GET', '/experience-requests');
    }

    async approveExperienceRequest(id, message = null) {
        return this.request('PUT', `/experience-requests/${id}/approve`, { message });
    }

    async denyExperienceRequest(id, message = null) {
        return this.request('PUT', `/experience-requests/${id}/deny`, { message });
    }

    // ACCOUNT DELETION & GDPR ENDPOINTS

    async deleteAccount(password) {
        return this.request('DELETE', '/users/me', { password });
    }

    // USER MANAGEMENT ENDPOINTS (Admin only)

    async getUsers(activeOnly = null) {
        const params = new URLSearchParams();
        if (activeOnly !== null) {
            params.append('active', activeOnly ? 'true' : 'false');
        }
        const endpoint = `/users${params.toString() ? '?' + params.toString() : ''}`;
        return this.request('GET', endpoint);
    }

    async getUser(id) {
        return this.request('GET', `/users/${id}`);
    }

    async deactivateUser(id, reason) {
        return this.request('PUT', `/users/${id}/deactivate`, { reason });
    }

    async activateUser(id, message = null) {
        return this.request('PUT', `/users/${id}/activate`, { message });
    }

    async promoteToAdmin(userId) {
        return this.request('POST', `/admin/users/${userId}/promote`);
    }

    async demoteAdmin(userId) {
        return this.request('POST', `/admin/users/${userId}/demote`);
    }

    // REACTIVATION REQUEST ENDPOINTS

    async createReactivationRequest(email) {
        return this.request('POST', '/reactivation-requests', { email });
    }

    async getReactivationRequests() {
        return this.request('GET', '/reactivation-requests');
    }

    async approveReactivationRequest(id, message = null) {
        return this.request('PUT', `/reactivation-requests/${id}/approve`, { message });
    }

    async denyReactivationRequest(id, message = null) {
        return this.request('PUT', `/reactivation-requests/${id}/deny`, { message });
    }

    // ADMIN DASHBOARD ENDPOINTS

    async getAdminStats() {
        return this.request('GET', '/admin/stats');
    }

    async getRecentActivity() {
        return this.request('GET', '/admin/activity');
    }

    // BOOKING TIME ENDPOINTS

    async getAvailableTimeSlots(date) {
        return this.request('GET', `/booking-times/available?date=${date}`);
    }

    async getRulesForDate(date) {
        return this.request('GET', `/booking-times/rules-for-date?date=${date}`);
    }

    async getBookingTimeRules() {
        return this.request('GET', '/admin/booking-times/rules');
    }

    async updateBookingTimeRules(rules) {
        return this.request('PUT', '/admin/booking-times/rules', rules);
    }

    async createBookingTimeRule(rule) {
        return this.request('POST', '/admin/booking-times/rules', rule);
    }

    async deleteBookingTimeRule(id) {
        return this.request('DELETE', `/admin/booking-times/rules/${id}`);
    }

    // HOLIDAY ENDPOINTS

    async getHolidays(year) {
        return this.request('GET', `/holidays?year=${year}`);
    }

    async createHoliday(holiday) {
        return this.request('POST', '/admin/holidays', holiday);
    }

    async updateHoliday(id, holiday) {
        return this.request('PUT', `/admin/holidays/${id}`, holiday);
    }

    async deleteHoliday(id) {
        return this.request('DELETE', `/admin/holidays/${id}`);
    }

    // BOOKING APPROVAL ENDPOINTS

    async getPendingApprovalBookings() {
        return this.request('GET', '/admin/bookings/pending-approvals');
    }

    async approveBooking(id) {
        return this.request('PUT', `/admin/bookings/${id}/approve`);
    }

    async rejectBooking(id, reason) {
        return this.request('PUT', `/admin/bookings/${id}/reject`, { reason });
    }
}

// Global instance
window.api = new API();
