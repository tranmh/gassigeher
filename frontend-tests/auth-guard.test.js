/**
 * Unit tests for internal/static/frontend/js/auth-guard.js: redirects for
 * unauthenticated users and the admin / super admin checks.
 *
 * @jest-environment jsdom
 */
const {
    createFakeLocation,
    createFakeApi,
    buildUser,
    runAuthGuard,
    flushPromises,
} = require('./__tests__/aqaa-doubles');

const ADMIN = { 'data-require-admin': 'true' };
const SUPER_ADMIN = { 'data-require-super-admin': 'true' };

function loader() {
    return document.getElementById('auth-check-loader');
}

describe('auth-guard.js', () => {
    beforeEach(() => {
        document.body.innerHTML = '';
        document.body.style.visibility = '';
    });

    describe('missing api', () => {
        it('logs an error and does nothing else when api.js is not loaded', () => {
            const location = createFakeLocation();
            const { error, console } = runAuthGuard({ api: undefined, location });
            expect(error).toBeNull();
            expect(console.error).toHaveBeenCalledWith(expect.stringContaining('API not loaded'));
            expect(location.href).toBe('http://localhost/dashboard.html');
            expect(loader()).toBeNull();
        });
    });

    describe('unauthenticated', () => {
        it('redirects to login with the encoded return url and stops execution', () => {
            const location = createFakeLocation({ pathname: '/bookings.html', search: '?dog=3&date=2026-01-01' });
            const api = createFakeApi({ authenticated: false });
            const { error } = runAuthGuard({ api, location });
            expect(error).toBeInstanceOf(Error);
            expect(error.message).toBe('AuthGuard: Redirecting to login');
            expect(location.href).toBe('/login.html?redirect=%2Fbookings.html%3Fdog%3D3%26date%3D2026-01-01');
        });

        it('uses only the path when there is no query string', () => {
            const location = createFakeLocation({ pathname: '/profile.html' });
            runAuthGuard({ api: createFakeApi({ authenticated: false }), location });
            expect(location.href).toBe('/login.html?redirect=%2Fprofile.html');
        });

        it('does not query the server or show the loader even when admin is required', () => {
            const location = createFakeLocation({ pathname: '/admin-dashboard.html' });
            const api = createFakeApi({ authenticated: false });
            runAuthGuard({ api, location, scriptAttrs: ADMIN });
            expect(api.getMe).not.toHaveBeenCalled();
            expect(loader()).toBeNull();
            expect(location.href).toBe('/login.html?redirect=%2Fadmin-dashboard.html');
        });
    });

    describe('authenticated without role requirement', () => {
        it('lets the page render without a server check', () => {
            const location = createFakeLocation();
            const api = createFakeApi();
            const { error } = runAuthGuard({ api, location });
            expect(error).toBeNull();
            expect(api.getMe).not.toHaveBeenCalled();
            expect(document.body.style.visibility).toBe('');
            expect(loader()).toBeNull();
            expect(location.href).toBe('http://localhost/dashboard.html');
        });

        it.each(['false', 'TRUE', '1', ''])('ignores data-require-admin="%s"', (value) => {
            const api = createFakeApi();
            runAuthGuard({ api, location: createFakeLocation(), scriptAttrs: { 'data-require-admin': value } });
            expect(api.getMe).not.toHaveBeenCalled();
            expect(loader()).toBeNull();
        });
    });

    describe('admin required', () => {
        it('hides the page and shows a loader while verifying', () => {
            const api = createFakeApi({ user: buildUser({ is_admin: true }) });
            runAuthGuard({ api, location: createFakeLocation(), scriptAttrs: ADMIN });
            expect(api.getMe).toHaveBeenCalledTimes(1);
            expect(document.body.style.visibility).toBe('hidden');
            expect(loader()).not.toBeNull();
            expect(loader().textContent).toBe('Berechtigung wird überprüft...');
        });

        it('shows the page to admins and removes the loader', async () => {
            const location = createFakeLocation({ pathname: '/admin-users.html' });
            runAuthGuard({ api: createFakeApi({ user: buildUser({ is_admin: true }) }), location, scriptAttrs: ADMIN });
            await flushPromises();
            expect(document.body.style.visibility).toBe('');
            expect(loader()).toBeNull();
            expect(location.href).toBe('http://localhost/admin-users.html');
        });

        it('sends non-admins to the dashboard and keeps the page hidden', async () => {
            const location = createFakeLocation({ pathname: '/admin-users.html' });
            const { console } = runAuthGuard({
                api: createFakeApi({ user: buildUser({ is_admin: false }) }),
                location,
                scriptAttrs: ADMIN,
            });
            await flushPromises();
            expect(location.href).toBe('/dashboard.html');
            expect(console.warn).toHaveBeenCalledWith('AuthGuard: Admin access required');
            expect(document.body.style.visibility).toBe('hidden');
        });

        it('treats a missing is_admin flag as not admin', async () => {
            const location = createFakeLocation({ pathname: '/admin-users.html' });
            runAuthGuard({ api: createFakeApi({ user: { id: 2 } }), location, scriptAttrs: ADMIN });
            await flushPromises();
            expect(location.href).toBe('/dashboard.html');
        });

        it('clears the token and redirects to login when the user cannot be loaded', async () => {
            const location = createFakeLocation({ pathname: '/admin-users.html' });
            const api = createFakeApi({ meError: new Error('Unauthorized') });
            const { console } = runAuthGuard({ api, location, scriptAttrs: ADMIN });
            await flushPromises();
            expect(api.setToken).toHaveBeenCalledWith(null);
            expect(location.href).toBe('/login.html');
            expect(console.error).toHaveBeenCalledWith('AuthGuard: Failed to verify user:', expect.any(Error));
        });
    });

    describe('super admin required', () => {
        it('sends a plain admin to the dashboard', async () => {
            const location = createFakeLocation({ pathname: '/central.html' });
            const { console } = runAuthGuard({
                api: createFakeApi({ user: buildUser({ is_admin: true, is_super_admin: false }) }),
                location,
                scriptAttrs: SUPER_ADMIN,
            });
            await flushPromises();
            expect(location.href).toBe('/dashboard.html');
            expect(console.warn).toHaveBeenCalledWith('AuthGuard: Super admin access required');
        });

        it('shows the page to super admins', async () => {
            const location = createFakeLocation({ pathname: '/central.html' });
            runAuthGuard({
                api: createFakeApi({ user: buildUser({ is_admin: true, is_super_admin: true }) }),
                location,
                scriptAttrs: SUPER_ADMIN,
            });
            await flushPromises();
            expect(location.href).toBe('http://localhost/central.html');
            expect(document.body.style.visibility).toBe('');
            expect(loader()).toBeNull();
        });

        it('requires both roles when both attributes are set', async () => {
            const location = createFakeLocation({ pathname: '/central.html' });
            runAuthGuard({
                api: createFakeApi({ user: buildUser({ is_admin: false, is_super_admin: true }) }),
                location,
                scriptAttrs: { ...ADMIN, ...SUPER_ADMIN },
            });
            await flushPromises();
            expect(location.href).toBe('/dashboard.html');
        });

        it('redirects to login when verification fails', async () => {
            const location = createFakeLocation({ pathname: '/central.html' });
            const api = createFakeApi({ meError: new Error('network') });
            runAuthGuard({ api, location, scriptAttrs: SUPER_ADMIN });
            await flushPromises();
            expect(api.setToken).toHaveBeenCalledWith(null);
            expect(location.href).toBe('/login.html');
        });
    });
});
