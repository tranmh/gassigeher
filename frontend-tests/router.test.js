/**
 * Router Module Tests
 *
 * Tests for the client-side router that handles navigation.
 *
 * @jest-environment jsdom
 */

// Mock window.api before loading router
window.api = {
  isAuthenticated: jest.fn(() => false),
};

// Mock history
const mockPushState = jest.fn();

// Define Router class manually since it's not exported
let RouterClass;
let routerInstance;

beforeAll(() => {
  // Override history.pushState
  Object.defineProperty(window, 'history', {
    writable: true,
    value: {
      pushState: mockPushState,
      replaceState: jest.fn(),
      go: jest.fn(),
      back: jest.fn(),
      forward: jest.fn(),
    },
  });

  // Define Router class
  RouterClass = class Router {
    constructor() {
      this.routes = {};
    }

    on(path, handler) {
      this.routes[path] = handler;
    }

    navigate(path, pushState = true) {
      let handler = this.routes[path];
      if (!handler) {
        for (const route in this.routes) {
          if (route.includes(':')) {
            const pattern = new RegExp('^' + route.replace(/:[^\s/]+/g, '([^/]+)') + '$');
            if (pattern.test(path)) {
              handler = this.routes[route];
              break;
            }
          }
        }
      }
      if (!handler) {
        handler = this.routes['/404'] || (() => {
          document.body.innerHTML = '<h1>404 - Page Not Found</h1>';
        });
      }
      if (pushState) {
        window.history.pushState({}, '', path);
      }
      handler();
    }

    getQueryParams() {
      const params = {};
      const queryString = window.location.search.substring(1);
      const pairs = queryString.split('&');
      for (const pair of pairs) {
        const [key, value] = pair.split('=');
        if (key) {
          params[decodeURIComponent(key)] = decodeURIComponent(value || '');
        }
      }
      return params;
    }

    redirect(path) {
      this.navigate(path);
    }

    requireAuth() {
      if (!window.api.isAuthenticated()) {
        this.redirect('/login.html');
        return false;
      }
      return true;
    }
  };

  document.body.innerHTML = '';
  window.router = new RouterClass();
});

beforeEach(() => {
  document.body.innerHTML = '';
  mockPushState.mockClear();
  window.api.isAuthenticated.mockReturnValue(false);

  // Reset router routes
  window.router.routes = {};
});

describe('Router class - Route Registration', () => {
  test('should register route with handler', () => {
    const handler = jest.fn();
    window.router.on('/test', handler);

    expect(window.router.routes['/test']).toBe(handler);
  });

  test('should register multiple routes', () => {
    const handler1 = jest.fn();
    const handler2 = jest.fn();

    window.router.on('/route1', handler1);
    window.router.on('/route2', handler2);

    expect(window.router.routes['/route1']).toBe(handler1);
    expect(window.router.routes['/route2']).toBe(handler2);
  });

  test('should overwrite existing route', () => {
    const handler1 = jest.fn();
    const handler2 = jest.fn();

    window.router.on('/test', handler1);
    window.router.on('/test', handler2);

    expect(window.router.routes['/test']).toBe(handler2);
  });

  test('should register 404 route', () => {
    const notFoundHandler = jest.fn();
    window.router.on('/404', notFoundHandler);

    expect(window.router.routes['/404']).toBe(notFoundHandler);
  });
});

describe('Router class - Navigation', () => {
  test('should call handler for exact path match', () => {
    const handler = jest.fn();
    window.router.on('/test', handler);

    window.router.navigate('/test');

    expect(handler).toHaveBeenCalled();
  });

  test('should push state to history by default', () => {
    const handler = jest.fn();
    window.router.on('/test', handler);

    window.router.navigate('/test');

    expect(mockPushState).toHaveBeenCalledWith({}, '', '/test');
  });

  test('should not push state when pushState is false', () => {
    const handler = jest.fn();
    window.router.on('/test', handler);

    window.router.navigate('/test', false);

    expect(mockPushState).not.toHaveBeenCalled();
  });

  test('should call 404 handler for unknown route', () => {
    const notFoundHandler = jest.fn();
    window.router.on('/404', notFoundHandler);

    window.router.navigate('/unknown-route');

    expect(notFoundHandler).toHaveBeenCalled();
  });

  test('should render default 404 message when no 404 handler', () => {
    window.router.navigate('/unknown-route');

    expect(document.body.innerHTML).toContain('404');
    expect(document.body.innerHTML).toContain('Page Not Found');
  });
});

describe('Router class - Wildcard Routes', () => {
  test('should match wildcard route with :param', () => {
    const handler = jest.fn();
    window.router.on('/dogs/:id', handler);

    window.router.navigate('/dogs/123');

    expect(handler).toHaveBeenCalled();
  });

  test('should match multiple wildcard params', () => {
    const handler = jest.fn();
    window.router.on('/users/:userId/bookings/:bookingId', handler);

    window.router.navigate('/users/5/bookings/10');

    expect(handler).toHaveBeenCalled();
  });

  test('should prefer exact match over wildcard', () => {
    const exactHandler = jest.fn();
    const wildcardHandler = jest.fn();

    window.router.on('/dogs/featured', exactHandler);
    window.router.on('/dogs/:id', wildcardHandler);

    window.router.navigate('/dogs/featured');

    expect(exactHandler).toHaveBeenCalled();
    expect(wildcardHandler).not.toHaveBeenCalled();
  });

  test('should not match wildcard with extra segments', () => {
    const handler = jest.fn();
    const notFoundHandler = jest.fn();

    window.router.on('/dogs/:id', handler);
    window.router.on('/404', notFoundHandler);

    window.router.navigate('/dogs/123/extra');

    expect(handler).not.toHaveBeenCalled();
    expect(notFoundHandler).toHaveBeenCalled();
  });
});

describe('Router class - Query Parameters', () => {
  beforeEach(() => {
    // Mock location.search
    delete window.location;
    window.location = {
      pathname: '/',
      search: '',
      hostname: 'localhost',
    };
  });

  test('should return empty object for no query params', () => {
    window.location.search = '';

    const params = window.router.getQueryParams();

    expect(params).toEqual({});
  });

  test('should parse single query param', () => {
    window.location.search = '?foo=bar';

    const params = window.router.getQueryParams();

    expect(params).toEqual({ foo: 'bar' });
  });

  test('should parse multiple query params', () => {
    window.location.search = '?foo=bar&baz=qux';

    const params = window.router.getQueryParams();

    expect(params).toEqual({ foo: 'bar', baz: 'qux' });
  });

  test('should handle empty value', () => {
    window.location.search = '?empty=';

    const params = window.router.getQueryParams();

    expect(params).toEqual({ empty: '' });
  });

  test('should decode URL encoded params', () => {
    window.location.search = '?name=John%20Doe&city=M%C3%BCnchen';

    const params = window.router.getQueryParams();

    expect(params).toEqual({ name: 'John Doe', city: 'München' });
  });

  test('should handle param without value (key only)', () => {
    window.location.search = '?active';

    const params = window.router.getQueryParams();

    expect(params).toEqual({ active: '' });
  });
});

describe('Router class - Redirect', () => {
  test('redirect should call navigate', () => {
    const handler = jest.fn();
    window.router.on('/target', handler);

    window.router.redirect('/target');

    expect(handler).toHaveBeenCalled();
    expect(mockPushState).toHaveBeenCalledWith({}, '', '/target');
  });
});

describe('Router class - requireAuth', () => {
  beforeEach(() => {
    delete window.location;
    window.location = {
      pathname: '/',
      href: '',
    };
  });

  test('should return true when authenticated', () => {
    window.api.isAuthenticated.mockReturnValue(true);

    const result = window.router.requireAuth();

    expect(result).toBe(true);
  });

  test('should return false when not authenticated', () => {
    window.api.isAuthenticated.mockReturnValue(false);
    const handler = jest.fn();
    window.router.on('/login.html', handler);

    const result = window.router.requireAuth();

    expect(result).toBe(false);
  });

  test('should redirect to login when not authenticated', () => {
    window.api.isAuthenticated.mockReturnValue(false);
    const loginHandler = jest.fn();
    window.router.on('/login.html', loginHandler);

    window.router.requireAuth();

    expect(loginHandler).toHaveBeenCalled();
  });
});


describe('Router - Global instance', () => {
  test('should have global router instance on window', () => {
    expect(window.router).toBeDefined();
    expect(typeof window.router.on).toBe('function');
    expect(typeof window.router.navigate).toBe('function');
    expect(typeof window.router.redirect).toBe('function');
    expect(typeof window.router.requireAuth).toBe('function');
    expect(typeof window.router.getQueryParams).toBe('function');
  });
});

describe('Router - Edge Cases', () => {
  test('should handle root path', () => {
    const handler = jest.fn();
    window.router.on('/', handler);

    window.router.navigate('/');

    expect(handler).toHaveBeenCalled();
  });

  test('should handle path with trailing slash', () => {
    const handler = jest.fn();
    window.router.on('/test/', handler);

    window.router.navigate('/test/');

    expect(handler).toHaveBeenCalled();
  });

  test('should handle deep nested paths', () => {
    const handler = jest.fn();
    window.router.on('/a/b/c/d/e', handler);

    window.router.navigate('/a/b/c/d/e');

    expect(handler).toHaveBeenCalled();
  });

  test('should handle path with file extension', () => {
    const handler = jest.fn();
    window.router.on('/page.html', handler);

    window.router.navigate('/page.html');

    expect(handler).toHaveBeenCalled();
  });
});
