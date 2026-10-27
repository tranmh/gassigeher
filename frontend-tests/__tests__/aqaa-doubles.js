/**
 * Shared test doubles for the frontend scripts (aqaa).
 *
 * The scripts under internal/static are plain browser scripts (no exports),
 * so they are loaded from disk and evaluated with injected globals.
 */
const fs = require('fs');
const path = require('path');

const STATIC_DIR = path.join(__dirname, '..', '..', 'internal', 'static');

function readStatic(relPath) {
    return fs.readFileSync(path.join(STATIC_DIR, relPath), 'utf8');
}

/** Load sanitizeFAQAnswer from shared/js/faq-sanitizer.js using the jsdom document. */
function loadFaqSanitizer() {
    const src = readStatic('shared/js/faq-sanitizer.js');
    const fakeWindow = {};
    // eslint-disable-next-line no-new-func
    const fn = new Function('window', 'document', src + '\nreturn sanitizeFAQAnswer;');
    return fn(fakeWindow, document);
}

/** Fake window.location that records navigations instead of performing them. */
function createFakeLocation({ pathname = '/dashboard.html', search = '' } = {}) {
    return { pathname, search, href: 'http://localhost' + pathname + search };
}

/**
 * Fake of the api client (api.js) surface the guards use.
 * getMe resolves with `user` or rejects with `meError`.
 */
function createFakeApi({ authenticated = true, user = buildUser(), meError = null } = {}) {
    return {
        isAuthenticated: jest.fn(() => authenticated),
        getMe: jest.fn(() => (meError ? Promise.reject(meError) : Promise.resolve(user))),
        setToken: jest.fn(),
    };
}

/** Builder for the user object returned by GET /api/users/me. */
function buildUser(overrides = {}) {
    return {
        id: 1,
        first_name: 'Test',
        last_name: 'Walker',
        email: 'walker@example.com',
        experience_level: 'green',
        is_admin: false,
        is_super_admin: false,
        is_active: true,
        ...overrides,
    };
}

/** Builder for a FAQ entry as used by faq-data.js. */
function buildFaq(overrides = {}) {
    return {
        id: 'faq-1',
        question: 'Wie buche ich?',
        answer: '<p>Mit <strong>einem Klick</strong>.</p>',
        ...overrides,
    };
}

/**
 * Run auth-guard.js with injected doubles.
 * Returns { error } — the guard throws to stop page execution on redirect.
 */
function runAuthGuard({ api, location, scriptAttrs = {}, consoleDouble } = {}) {
    const src = readStatic('frontend/js/auth-guard.js');
    const script = document.createElement('script');
    Object.entries(scriptAttrs).forEach(([k, v]) => script.setAttribute(k, v));
    const fakeDocument = {
        currentScript: script,
        body: document.body,
        createElement: (tag) => document.createElement(tag),
    };
    const fakeWindow = { location };
    const fakeConsole = consoleDouble || { error: jest.fn(), warn: jest.fn(), log: jest.fn() };
    // eslint-disable-next-line no-new-func
    const fn = new Function('window', 'document', 'api', 'console', src);
    try {
        fn(fakeWindow, fakeDocument, api, fakeConsole);
        return { error: null, console: fakeConsole };
    } catch (error) {
        return { error, console: fakeConsole };
    }
}

/** Let pending promise callbacks run. */
async function flushPromises() {
    for (let i = 0; i < 5; i++) await Promise.resolve();
}

module.exports = {
    readStatic,
    loadFaqSanitizer,
    createFakeLocation,
    createFakeApi,
    buildUser,
    buildFaq,
    runAuthGuard,
    flushPromises,
};
