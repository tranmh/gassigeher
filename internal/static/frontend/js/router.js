// Simple client-side router
class Router {
    constructor() {
        this.routes = {};
        this.init();
    }

    // Register a route
    on(path, handler) {
        this.routes[path] = handler;
    }

    // Initialize router
    init() {
        // Handle popstate (back/forward buttons)
        window.addEventListener('popstate', () => {
            this.navigate(window.location.pathname, false);
        });

        // Handle clicks on links
        document.addEventListener('click', (e) => {
            if (e.target.matches('[data-route]')) {
                e.preventDefault();
                const path = e.target.getAttribute('href');
                this.navigate(path);
            }
        });

        // Navigate to current path on init
        this.navigate(window.location.pathname, false);
    }

    // Navigate to a path
    navigate(path, pushState = true) {
        // Find matching route
        let handler = this.routes[path];
        let params = {};

        // Try exact match first, then wildcard
        if (!handler) {
            // Check for wildcard routes
            for (const route in this.routes) {
                if (route.includes(':')) {
                    const pattern = new RegExp('^' + route.replace(/:[^\s/]+/g, '([^/]+)') + '$');
                    const match = pattern.exec(path);
                    if (match) {
                        handler = this.routes[route];
                        // Extract param names and values
                        const paramNames = (route.match(/:[^\s/]+/g) || []).map(p => p.substring(1));
                        paramNames.forEach((name, index) => {
                            params[name] = match[index + 1];
                        });
                        break;
                    }
                }
            }
        }

        // Default to 404 if no match
        if (!handler) {
            handler = this.routes['/404'] || (() => {
                document.body.innerHTML = '<h1>404 - Page Not Found</h1>';
            });
        }

        // Update browser history
        if (pushState) {
            window.history.pushState({}, '', path);
        }

        // Call route handler with extracted params
        handler(params);
    }

    // Get query parameters
    getQueryParams() {
        const params = {};
        const queryString = window.location.search.substring(1);
        const pairs = queryString.split('&');

        for (const pair of pairs) {
            const [key, value] = pair.split('=');
            if (key) {
                try {
                    params[decodeURIComponent(key)] = decodeURIComponent(value || '');
                } catch (e) {
                    // Handle malformed URI encoding gracefully
                    params[key] = value || '';
                }
            }
        }

        return params;
    }

    // Redirect to path
    redirect(path) {
        this.navigate(path);
    }

    // Check if user is authenticated, redirect to login if not
    requireAuth() {
        if (!window.api.isAuthenticated()) {
            this.redirect('/login.html');
            return false;
        }
        return true;
    }
}

// Global instance
window.router = new Router();
