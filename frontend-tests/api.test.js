/**
 * API Client Tests
 *
 * Tests for the API client that handles all backend communication.
 *
 * @jest-environment jsdom
 */

// Mock localStorage
const localStorageMock = (() => {
  let store = {};
  return {
    getItem: jest.fn((key) => store[key] || null),
    setItem: jest.fn((key, value) => { store[key] = value; }),
    removeItem: jest.fn((key) => { delete store[key]; }),
    clear: jest.fn(() => { store = {}; }),
  };
})();

Object.defineProperty(window, 'localStorage', { value: localStorageMock });

// Mock fetch
let fetchMock;

// Helper to create mock response (API now uses text() then JSON.parse)
const mockResponse = (data, ok = true, status = 200) => ({
  ok,
  status,
  text: () => Promise.resolve(data ? JSON.stringify(data) : ''),
});

beforeAll(() => {
  document.body.innerHTML = '';
  loadSourceFile('internal/static/frontend/js/api.js');
});

beforeEach(() => {
  localStorageMock.clear();
  fetchMock = jest.fn();
  global.fetch = fetchMock;
  // Reset the API instance token
  window.api.token = null;
});

afterEach(() => {
  jest.restoreAllMocks();
});

describe('API class - Token Management', () => {
  test('setToken should store token in localStorage', () => {
    window.api.setToken('test-token');
    expect(localStorageMock.setItem).toHaveBeenCalledWith('gassigeher_token', 'test-token');
    expect(window.api.token).toBe('test-token');
  });

  test('setToken(null) should remove token from localStorage', () => {
    window.api.setToken(null);
    expect(localStorageMock.removeItem).toHaveBeenCalledWith('gassigeher_token');
    expect(window.api.token).toBeNull();
  });

  test('getToken should return current token', () => {
    window.api.token = 'my-token';
    expect(window.api.getToken()).toBe('my-token');
  });

  test('isAuthenticated should return true when token exists', () => {
    window.api.token = 'valid-token';
    expect(window.api.isAuthenticated()).toBe(true);
  });

  test('isAuthenticated should return false when no token', () => {
    window.api.token = null;
    expect(window.api.isAuthenticated()).toBe(false);
  });

  test('isAuthenticated should return false for empty string', () => {
    window.api.token = '';
    expect(window.api.isAuthenticated()).toBe(false);
  });
});

describe('API class - Request Method', () => {
  test('should make GET request with correct headers', async () => {
    fetchMock.mockResolvedValue(mockResponse({ data: 'test' }));

    await window.api.request('GET', '/test');

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/test', {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    });
  });

  test('should include Authorization header when token exists', async () => {
    window.api.token = 'bearer-token';
    fetchMock.mockResolvedValue(mockResponse({ data: 'test' }));

    await window.api.request('GET', '/test');

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/test', expect.objectContaining({
      headers: expect.objectContaining({
        'Authorization': 'Bearer bearer-token',
      }),
    }));
  });

  test('should send body for POST requests', async () => {
    fetchMock.mockResolvedValue(mockResponse({ success: true }));

    await window.api.request('POST', '/test', { name: 'test' });

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/test', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ name: 'test' }),
    }));
  });

  test('should send body for PUT requests', async () => {
    fetchMock.mockResolvedValue(mockResponse({ success: true }));

    await window.api.request('PUT', '/users/1', { name: 'updated' });

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/users/1', expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({ name: 'updated' }),
    }));
  });

  test('should not send body for DELETE requests', async () => {
    fetchMock.mockResolvedValue(mockResponse({ success: true }));

    await window.api.request('DELETE', '/test/1');

    const callArgs = fetchMock.mock.calls[0][1];
    expect(callArgs.body).toBeUndefined();
  });

  test('should throw error on failed request', async () => {
    fetchMock.mockResolvedValue(mockResponse({ error: 'Bad Request' }, false, 400));

    await expect(window.api.request('GET', '/test')).rejects.toThrow('Bad Request');
  });

  test('should attach status to error object', async () => {
    fetchMock.mockResolvedValue(mockResponse({ error: 'Not Found' }, false, 404));

    try {
      await window.api.request('GET', '/test');
    } catch (error) {
      expect(error.status).toBe(404);
    }
  });

  test('should return response data on success', async () => {
    fetchMock.mockResolvedValue(mockResponse({ id: 1, name: 'Test' }));

    const result = await window.api.request('GET', '/test');
    expect(result).toEqual({ id: 1, name: 'Test' });
  });
});

describe('API class - Auth Endpoints', () => {
  beforeEach(() => {
    fetchMock.mockResolvedValue(mockResponse({ success: true }));
  });

  test('register should POST to /auth/register', async () => {
    await window.api.register({ email: 'test@example.com', password: 'pass123' });

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/auth/register', expect.objectContaining({
      method: 'POST',
    }));
  });

  test('verifyEmail should POST token', async () => {
    await window.api.verifyEmail('verify-token');

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/auth/verify-email', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ token: 'verify-token' }),
    }));
  });

  test('login should POST credentials and set token', async () => {
    fetchMock.mockResolvedValue(mockResponse({ token: 'new-token', user: { id: 1 } }));

    const result = await window.api.login('test@example.com', 'password');

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/auth/login', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ email: 'test@example.com', password: 'password' }),
    }));
    expect(window.api.token).toBe('new-token');
  });

  test('login should not set token if not returned', async () => {
    fetchMock.mockResolvedValue(mockResponse({ user: { id: 1 } }));

    await window.api.login('test@example.com', 'password');
    expect(window.api.token).toBeNull();
  });

  test('forgotPassword should POST email', async () => {
    await window.api.forgotPassword('test@example.com');

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/auth/forgot-password', expect.objectContaining({
      body: JSON.stringify({ email: 'test@example.com' }),
    }));
  });

  test('resetPassword should POST token and passwords', async () => {
    await window.api.resetPassword('reset-token', 'newpass', 'newpass');

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/auth/reset-password', expect.objectContaining({
      body: JSON.stringify({
        token: 'reset-token',
        password: 'newpass',
        confirm_password: 'newpass',
      }),
    }));
  });

  test('changePassword should PUT password data', async () => {
    await window.api.changePassword('oldpass', 'newpass', 'newpass');

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/auth/change-password', expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({
        old_password: 'oldpass',
        new_password: 'newpass',
        confirm_password: 'newpass',
      }),
    }));
  });
});

describe('API class - User Endpoints', () => {
  beforeEach(() => {
    fetchMock.mockResolvedValue(mockResponse({ id: 1, name: 'Test User' }));
  });

  test('getMe should GET /users/me', async () => {
    await window.api.getMe();
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/users/me', expect.objectContaining({
      method: 'GET',
    }));
  });

  test('updateMe should PUT /users/me', async () => {
    await window.api.updateMe({ first_name: 'John' });
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/users/me', expect.objectContaining({
      method: 'PUT',
    }));
  });

  test('deleteAccount should DELETE /users/me', async () => {
    await window.api.deleteAccount('mypassword');
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/users/me', expect.objectContaining({
      method: 'DELETE',
    }));
  });

  test('getUsers should GET /users', async () => {
    await window.api.getUsers();
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/users', expect.objectContaining({
      method: 'GET',
    }));
  });

  test('getUsers with activeOnly should add query param', async () => {
    await window.api.getUsers(true);
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/users?active=true', expect.anything());
  });

  test('getUser should GET specific user', async () => {
    await window.api.getUser(123);
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/users/123', expect.anything());
  });

  test('deactivateUser should PUT with reason', async () => {
    await window.api.deactivateUser(123, 'Inactive');
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/users/123/deactivate', expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({ reason: 'Inactive' }),
    }));
  });

  test('activateUser should PUT with optional message', async () => {
    await window.api.activateUser(123, 'Welcome back');
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/users/123/activate', expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({ message: 'Welcome back' }),
    }));
  });

  test('createUser should POST to /users', async () => {
    await window.api.createUser({ email: 'new@example.com' });
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/users', expect.objectContaining({
      method: 'POST',
    }));
  });
});

describe('API class - Dog Endpoints', () => {
  beforeEach(() => {
    fetchMock.mockResolvedValue(mockResponse({ id: 1, name: 'Rex' }));
  });

  test('getDogs should GET /dogs', async () => {
    await window.api.getDogs();
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/dogs', expect.anything());
  });

  test('getDogs with filters should add query params', async () => {
    await window.api.getDogs({ available: true, color_id: 2 });
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining('/api/v1/dogs?'),
      expect.anything()
    );
  });

  test('getDog should GET specific dog', async () => {
    await window.api.getDog(5);
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/dogs/5', expect.anything());
  });

  test('getBreeds should GET /dogs/breeds', async () => {
    await window.api.getBreeds();
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/dogs/breeds', expect.anything());
  });

  test('createDog should POST to /dogs', async () => {
    await window.api.createDog({ name: 'Buddy' });
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/dogs', expect.objectContaining({
      method: 'POST',
    }));
  });

  test('updateDog should PUT to /dogs/:id', async () => {
    await window.api.updateDog(5, { name: 'Buddy Updated' });
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/dogs/5', expect.objectContaining({
      method: 'PUT',
    }));
  });

  test('deleteDog should DELETE /dogs/:id', async () => {
    await window.api.deleteDog(5);
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/dogs/5', expect.objectContaining({
      method: 'DELETE',
    }));
  });

  test('deleteDog with force should add query param', async () => {
    await window.api.deleteDog(5, true);
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/dogs/5?force=true', expect.anything());
  });

  test('toggleDogAvailability should PUT availability', async () => {
    await window.api.toggleDogAvailability(5, true, null);
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/dogs/5/availability', expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({ is_available: true, unavailable_reason: null }),
    }));
  });

  test('setDogFeatured should PUT featured status', async () => {
    await window.api.setDogFeatured(5, true);
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/dogs/5/featured', expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({ is_featured: true }),
    }));
  });

  test('getFeaturedDogs should GET /dogs/featured', async () => {
    await window.api.getFeaturedDogs();
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/dogs/featured', expect.anything());
  });
});

describe('API class - Booking Endpoints', () => {
  beforeEach(() => {
    fetchMock.mockResolvedValue(mockResponse({ id: 1 }));
  });

  test('createBooking should POST to /bookings', async () => {
    await window.api.createBooking({ dog_id: 1, date: '2025-01-01' });
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/bookings', expect.objectContaining({
      method: 'POST',
    }));
  });

  test('getBookings should GET /bookings', async () => {
    await window.api.getBookings();
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/bookings', expect.anything());
  });

  test('getBookings with filters should add query params', async () => {
    await window.api.getBookings({ status: 'scheduled' });
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining('/api/v1/bookings?'),
      expect.anything()
    );
  });

  test('getBooking should GET specific booking', async () => {
    await window.api.getBooking(10);
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/bookings/10', expect.anything());
  });

  test('cancelBooking should PUT with optional reason', async () => {
    await window.api.cancelBooking(10, 'No longer needed');
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/bookings/10/cancel', expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({ reason: 'No longer needed' }),
    }));
  });

  test('moveBooking should PUT new date and time', async () => {
    await window.api.moveBooking(10, '2025-02-01', '10:00', 'Rescheduled');
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/bookings/10/move', expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({
        date: '2025-02-01',
        scheduled_time: '10:00',
        reason: 'Rescheduled',
      }),
    }));
  });

  test('addBookingNotes should PUT notes', async () => {
    await window.api.addBookingNotes(10, 'Walk went well');
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/bookings/10/notes', expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({ notes: 'Walk went well' }),
    }));
  });

  test('getCalendarData should GET with year and month', async () => {
    await window.api.getCalendarData(2025, 6);
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/bookings/calendar/2025/6', expect.anything());
  });
});

describe('API class - Color Endpoints', () => {
  beforeEach(() => {
    fetchMock.mockResolvedValue(mockResponse([]));
  });

  test('getColors should GET /colors', async () => {
    await window.api.getColors();
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/colors', expect.anything());
  });

  test('createColor should POST to /colors', async () => {
    await window.api.createColor({ name: 'Green', hex_code: '#00ff00' });
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/colors', expect.objectContaining({
      method: 'POST',
    }));
  });

  test('updateColor should PUT to /colors/:id', async () => {
    await window.api.updateColor(2, { name: 'Blue' });
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/colors/2', expect.objectContaining({
      method: 'PUT',
    }));
  });

  test('deleteColor should DELETE /colors/:id', async () => {
    await window.api.deleteColor(2);
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/colors/2', expect.objectContaining({
      method: 'DELETE',
    }));
  });

  test('getColorStats should GET /colors/:id/stats', async () => {
    await window.api.getColorStats(2);
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/colors/2/stats', expect.anything());
  });
});

describe('API class - Settings Endpoints', () => {
  beforeEach(() => {
    fetchMock.mockResolvedValue(mockResponse([]));
  });

  test('getSettings should GET /settings', async () => {
    await window.api.getSettings();
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/settings', expect.anything());
  });

  test('updateSetting should PUT to /settings/:key', async () => {
    await window.api.updateSetting('booking_advance_days', '14');
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/settings/booking_advance_days', expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({ value: '14' }),
    }));
  });

  test('getLogo should GET /settings/logo', async () => {
    await window.api.getLogo();
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/settings/logo', expect.anything());
  });

  test('resetLogo should DELETE /settings/logo', async () => {
    await window.api.resetLogo();
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/settings/logo', expect.objectContaining({
      method: 'DELETE',
    }));
  });
});

describe('API class - Admin Endpoints', () => {
  beforeEach(() => {
    fetchMock.mockResolvedValue(mockResponse({}));
  });

  test('getAdminStats should GET /admin/stats', async () => {
    await window.api.getAdminStats();
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/admin/stats', expect.anything());
  });

  test('getRecentActivity should GET /admin/activity', async () => {
    await window.api.getRecentActivity();
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/admin/activity', expect.anything());
  });

  test('promoteToAdmin should POST', async () => {
    await window.api.promoteToAdmin(5);
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/admin/users/5/promote', expect.objectContaining({
      method: 'POST',
    }));
  });

  test('demoteAdmin should POST', async () => {
    await window.api.demoteAdmin(5);
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/admin/users/5/demote', expect.objectContaining({
      method: 'POST',
    }));
  });

  test('impersonateUser should POST', async () => {
    await window.api.impersonateUser(5);
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/admin/users/5/impersonate', expect.objectContaining({
      method: 'POST',
    }));
  });

  test('endImpersonation should POST', async () => {
    await window.api.endImpersonation();
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/end-impersonation', expect.objectContaining({
      method: 'POST',
    }));
  });
});

describe('API class - Booking Time Endpoints', () => {
  beforeEach(() => {
    fetchMock.mockResolvedValue(mockResponse({}));
  });

  test('getAvailableTimeSlots should GET with date param', async () => {
    await window.api.getAvailableTimeSlots('2025-01-15');
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/booking-times/available?date=2025-01-15', expect.anything());
  });

  test('getRulesForDate should GET with date param', async () => {
    await window.api.getRulesForDate('2025-01-15');
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/booking-times/rules-for-date?date=2025-01-15', expect.anything());
  });

  test('getBookingTimeRules should GET admin rules', async () => {
    await window.api.getBookingTimeRules();
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/admin/booking-times/rules', expect.anything());
  });

  test('updateBookingTimeRules should PUT rules', async () => {
    const rules = [{ id: 1, start_time: '09:00' }];
    await window.api.updateBookingTimeRules(rules);
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/admin/booking-times/rules', expect.objectContaining({
      method: 'PUT',
    }));
  });

  test('createBookingTimeRule should POST new rule', async () => {
    await window.api.createBookingTimeRule({ day_type: 'weekday', rule_name: 'Morning' });
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/admin/booking-times/rules', expect.objectContaining({
      method: 'POST',
    }));
  });

  test('deleteBookingTimeRule should DELETE rule', async () => {
    await window.api.deleteBookingTimeRule(3);
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/admin/booking-times/rules/3', expect.objectContaining({
      method: 'DELETE',
    }));
  });
});

describe('API class - Upload Methods', () => {
  beforeEach(() => {
    fetchMock.mockResolvedValue(mockResponse({ photo_url: '/uploads/test.jpg' }));
  });

  test('uploadFile should POST FormData without Content-Type header', async () => {
    const formData = new FormData();
    formData.append('file', new Blob(['test']));

    await window.api.uploadFile('/test/upload', formData);

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/test/upload', expect.objectContaining({
      method: 'POST',
      body: formData,
    }));

    // Should not have Content-Type header (browser sets it for FormData)
    const callArgs = fetchMock.mock.calls[0][1];
    expect(callArgs.headers['Content-Type']).toBeUndefined();
  });

  test('uploadFile should include auth header when token exists', async () => {
    window.api.token = 'auth-token';
    const formData = new FormData();

    await window.api.uploadFile('/test/upload', formData);

    const callArgs = fetchMock.mock.calls[0][1];
    expect(callArgs.headers['Authorization']).toBe('Bearer auth-token');
  });

  test('uploadPhoto should use correct endpoint', async () => {
    const file = new File([''], 'photo.jpg', { type: 'image/jpeg' });
    await window.api.uploadPhoto(file);

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/users/me/photo', expect.anything());
  });

  test('uploadDogPhoto should use correct endpoint with dog ID', async () => {
    const file = new File([''], 'dog.jpg', { type: 'image/jpeg' });
    await window.api.uploadDogPhoto(5, file);

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/dogs/5/photo', expect.anything());
  });
});

describe('API class - Walk Report Endpoints', () => {
  beforeEach(() => {
    fetchMock.mockResolvedValue(mockResponse({}));
  });

  test('createWalkReport should POST report', async () => {
    await window.api.createWalkReport({ booking_id: 1, notes: 'Great walk' });
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/walk-reports', expect.objectContaining({
      method: 'POST',
    }));
  });

  test('getWalkReport should GET specific report', async () => {
    await window.api.getWalkReport(5);
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/walk-reports/5', expect.anything());
  });

  test('getWalkReportByBooking should GET by booking ID', async () => {
    await window.api.getWalkReportByBooking(10);
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/walk-reports/by-booking/10', expect.anything());
  });

  test('updateWalkReport should PUT report', async () => {
    await window.api.updateWalkReport(5, { notes: 'Updated' });
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/walk-reports/5', expect.objectContaining({
      method: 'PUT',
    }));
  });

  test('deleteWalkReport should DELETE report', async () => {
    await window.api.deleteWalkReport(5);
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/walk-reports/5', expect.objectContaining({
      method: 'DELETE',
    }));
  });

  test('getDogWalkReports should GET with limit', async () => {
    await window.api.getDogWalkReports(3, 5);
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/dogs/3/walk-reports?limit=5', expect.anything());
  });

  test('deleteWalkReportPhoto should DELETE photo', async () => {
    await window.api.deleteWalkReportPhoto(5, 2);
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/walk-reports/5/photos/2', expect.objectContaining({
      method: 'DELETE',
    }));
  });
});

describe('API class - Recurring Booking Endpoints', () => {
  beforeEach(() => {
    fetchMock.mockResolvedValue(mockResponse({}));
  });

  test('previewRecurringBooking should POST preview data', async () => {
    const data = { dog_id: 1, recurrence_type: 'weekly', day_of_week: 2, scheduled_time: '09:00', start_date: '2026-03-01', weeks: 4 };
    await window.api.previewRecurringBooking(data);
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/bookings/recurring/preview', expect.objectContaining({
      method: 'POST',
    }));
  });

  test('createRecurringBooking should POST booking data', async () => {
    const data = { dog_id: 1, recurrence_type: 'weekly', day_of_week: 2, scheduled_time: '09:00', start_date: '2026-03-01', weeks: 4, selected_dates: ['2026-03-03'] };
    await window.api.createRecurringBooking(data);
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/bookings/recurring', expect.objectContaining({
      method: 'POST',
    }));
  });

  test('getMyRecurringSeries should GET user series', async () => {
    await window.api.getMyRecurringSeries();
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/bookings/recurring', expect.anything());
  });

  test('getRecurringSeries should GET specific series', async () => {
    await window.api.getRecurringSeries(7);
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/bookings/recurring/7', expect.anything());
  });

  test('cancelRecurringSeries should PUT cancel with reason', async () => {
    await window.api.cancelRecurringSeries(7, 'No longer needed');
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/bookings/recurring/7/cancel', expect.objectContaining({
      method: 'PUT',
    }));
  });

  test('cancelRecurringSeries should PUT cancel without reason', async () => {
    await window.api.cancelRecurringSeries(7);
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/bookings/recurring/7/cancel', expect.objectContaining({
      method: 'PUT',
    }));
  });

  test('adminListRecurringSeries should GET with filters', async () => {
    await window.api.adminListRecurringSeries({ status: 'active', dog_id: '3' });
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/admin/bookings/recurring?status=active&dog_id=3', expect.anything());
  });

  test('adminListRecurringSeries should GET without filters', async () => {
    await window.api.adminListRecurringSeries();
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/admin/bookings/recurring', expect.anything());
  });

  test('adminCancelRecurringSeries should PUT cancel', async () => {
    await window.api.adminCancelRecurringSeries(5, 'Admin reason');
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/admin/bookings/recurring/5/cancel', expect.objectContaining({
      method: 'PUT',
    }));
  });

  test('adminApproveRecurringSeries should PUT approve', async () => {
    await window.api.adminApproveRecurringSeries(5);
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/admin/bookings/recurring/5/approve', expect.objectContaining({
      method: 'PUT',
    }));
  });

  test('adminRejectRecurringSeries should PUT reject with reason', async () => {
    await window.api.adminRejectRecurringSeries(5, 'Not suitable');
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/admin/bookings/recurring/5/reject', expect.objectContaining({
      method: 'PUT',
    }));
  });
});
