/**
 * Smoke tests for the shared aqaa test doubles.
 *
 * @jest-environment jsdom
 */
const {
    loadFaqSanitizer,
    createFakeLocation,
    createFakeApi,
    buildUser,
    buildFaq,
    runAuthGuard,
    flushPromises,
} = require('./__tests__/aqaa-doubles');

describe('aqaa doubles: faq sanitizer', () => {
    const sanitize = loadFaqSanitizer();

    it('keeps safe FAQ markup and strips scripts', () => {
        expect(sanitize(buildFaq().answer)).toBe('<p>Mit <strong>einem Klick</strong>.</p>');
        expect(sanitize('<p>a</p><script>alert(1)</script>')).toBe('<p>a</p>');
    });

    it('removes event handlers and javascript: hrefs', () => {
        const out = sanitize('<a href="javascript:alert(1)" onclick="x()">c</a>');
        expect(out).toBe('<a>c</a>');
    });
});

describe('aqaa doubles: auth guard', () => {
    beforeEach(() => {
        document.body.innerHTML = '';
        document.body.style.visibility = '';
    });

    it('redirects unauthenticated users to login with return url', () => {
        const location = createFakeLocation({ pathname: '/bookings.html', search: '?x=1' });
        const { error } = runAuthGuard({ api: createFakeApi({ authenticated: false }), location });
        expect(error).toBeInstanceOf(Error);
        expect(location.href).toBe('/login.html?redirect=' + encodeURIComponent('/bookings.html?x=1'));
    });

    it('sends non-admins to the dashboard when admin is required', async () => {
        const location = createFakeLocation({ pathname: '/admin.html' });
        const api = createFakeApi({ user: buildUser({ is_admin: false }) });
        runAuthGuard({ api, location, scriptAttrs: { 'data-require-admin': 'true' } });
        expect(document.body.style.visibility).toBe('hidden');
        await flushPromises();
        expect(location.href).toBe('/dashboard.html');
    });

    it('shows the page to admins', async () => {
        const location = createFakeLocation({ pathname: '/admin.html' });
        const api = createFakeApi({ user: buildUser({ is_admin: true }) });
        runAuthGuard({ api, location, scriptAttrs: { 'data-require-admin': 'true' } });
        await flushPromises();
        expect(document.body.style.visibility).toBe('');
        expect(document.getElementById('auth-check-loader')).toBeNull();
        expect(location.href).toBe('http://localhost/admin.html');
    });

    it('clears the token and redirects to login when getMe fails', async () => {
        const location = createFakeLocation();
        const api = createFakeApi({ meError: new Error('401') });
        runAuthGuard({ api, location, scriptAttrs: { 'data-require-super-admin': 'true' } });
        await flushPromises();
        expect(api.setToken).toHaveBeenCalledWith(null);
        expect(location.href).toBe('/login.html');
    });
});
