/**
 * Gassigeher Guided Tour System
 * Uses Shepherd.js for interactive guided tours
 *
 * Features:
 * - User tour: Dashboard → Dogs → Booking flow
 * - Admin tour: Dashboard → Dogs management → Users → Settings
 * - Skip after first completion (localStorage)
 * - Always active on demo tenant
 * - "Replay tour" button in profile/settings
 */

// Shepherd.js is loaded locally in HTML pages that use tours
// <script src="/vendor/shepherd.min.js"></script>
// <link rel="stylesheet" href="/vendor/shepherd.css"/>

(function() {
    'use strict';

    // Tour storage keys
    const STORAGE_KEYS = {
        userTourComplete: 'gassigeher_user_tour_complete',
        adminTourComplete: 'gassigeher_admin_tour_complete',
        isDemo: 'gassigeher_is_demo'
    };

    // Check if we're on demo tenant
    function isDemoTenant() {
        // Check subdomain or cached value
        const hostname = window.location.hostname;
        // Demo tenant always uses "demo." subdomain regardless of base domain
        if (hostname.startsWith('demo.')) {
            return true;
        }
        return localStorage.getItem(STORAGE_KEYS.isDemo) === 'true';
    }

    // Check if tour should be skipped (for e2e tests)
    function shouldSkipTour() {
        // URL param: ?skipTour=true
        const urlParams = new URLSearchParams(window.location.search);
        if (urlParams.get('skipTour') === 'true') {
            return true;
        }
        // localStorage flag (for e2e test setup)
        if (localStorage.getItem('gassigeher_skip_tour') === 'true') {
            return true;
        }
        return false;
    }

    // Check if tour has been completed
    function isTourComplete(tourType) {
        // Allow skipping for e2e tests
        if (shouldSkipTour()) {
            return true;
        }
        if (isDemoTenant()) {
            return false; // Always show on demo
        }
        const key = tourType === 'admin' ? STORAGE_KEYS.adminTourComplete : STORAGE_KEYS.userTourComplete;
        return localStorage.getItem(key) === 'true';
    }

    // Mark tour as complete
    function markTourComplete(tourType) {
        if (isDemoTenant()) {
            return; // Don't mark complete on demo
        }
        const key = tourType === 'admin' ? STORAGE_KEYS.adminTourComplete : STORAGE_KEYS.userTourComplete;
        localStorage.setItem(key, 'true');
    }

    // Reset tour completion (for replay)
    function resetTour(tourType) {
        const key = tourType === 'admin' ? STORAGE_KEYS.adminTourComplete : STORAGE_KEYS.userTourComplete;
        localStorage.removeItem(key);
    }

    // Default button config
    const defaultButtons = {
        cancel: {
            classes: 'shepherd-button-secondary',
            text: 'Überspringen',
            action: function() {
                this.complete();
            }
        },
        next: {
            classes: 'shepherd-button-primary',
            text: 'Weiter',
            action: function() {
                this.next();
            }
        },
        back: {
            classes: 'shepherd-button-secondary',
            text: 'Zurück',
            action: function() {
                this.back();
            }
        },
        done: {
            classes: 'shepherd-button-primary',
            text: 'Fertig!',
            action: function() {
                this.complete();
            }
        }
    };

    // Create a new Shepherd tour
    function createTour(tourType) {
        if (typeof Shepherd === 'undefined') {
            console.warn('Shepherd.js not loaded. Tour disabled.');
            return null;
        }

        const tour = new Shepherd.Tour({
            useModalOverlay: true,
            defaultStepOptions: {
                classes: 'gassigeher-tour-step',
                scrollTo: { behavior: 'smooth', block: 'center' },
                cancelIcon: {
                    enabled: true
                }
            }
        });

        // Mark complete when tour ends
        tour.on('complete', () => {
            markTourComplete(tourType);
        });

        tour.on('cancel', () => {
            markTourComplete(tourType);
        });

        return tour;
    }

    // User Dashboard Tour Steps
    const userDashboardSteps = [
        {
            id: 'welcome',
            title: 'Willkommen bei Gassigeher!',
            text: 'Diese kurze Tour zeigt dir die wichtigsten Funktionen. Du kannst sie jederzeit überspringen.',
            buttons: [
                defaultButtons.cancel,
                defaultButtons.next
            ]
        },
        {
            id: 'nav-dogs',
            attachTo: { element: 'nav a[href="/dogs.html"], #main-nav a[href="/dogs.html"]', on: 'bottom' },
            title: 'Unsere Hunde',
            text: 'Hier findest du alle verfügbaren Hunde. Klicke auf einen Hund, um mehr über ihn zu erfahren und einen Spaziergang zu buchen.',
            buttons: [
                defaultButtons.back,
                defaultButtons.next
            ]
        },
        {
            id: 'nav-calendar',
            attachTo: { element: 'nav a[href="/calendar.html"], #main-nav a[href="/calendar.html"]', on: 'bottom' },
            title: 'Kalender',
            text: 'Im Kalender siehst du alle deine gebuchten Spaziergänge auf einen Blick.',
            buttons: [
                defaultButtons.back,
                defaultButtons.next
            ]
        },
        {
            id: 'my-bookings',
            attachTo: { element: '#upcoming-bookings', on: 'top' },
            title: 'Deine Buchungen',
            text: 'Hier siehst du deine anstehenden Spaziergänge. Du kannst sie bei Bedarf auch stornieren.',
            buttons: [
                defaultButtons.back,
                defaultButtons.next
            ],
            beforeShowPromise: function() {
                return new Promise(resolve => {
                    // Wait for element to exist
                    setTimeout(resolve, 100);
                });
            }
        },
        {
            id: 'profile',
            attachTo: { element: 'nav a[href="/profile.html"], #main-nav a[href="/profile.html"]', on: 'bottom' },
            title: 'Dein Profil',
            text: 'In deinem Profil kannst du deine Daten bearbeiten und dein Erfahrungslevel einsehen.',
            buttons: [
                defaultButtons.back,
                defaultButtons.done
            ]
        }
    ];

    // Dogs Page Tour Steps
    const dogsPageSteps = [
        {
            id: 'dogs-intro',
            title: 'Unsere Hunde',
            text: 'Hier findest du alle Hunde, die auf einen Spaziergang mit dir warten!',
            buttons: [
                defaultButtons.cancel,
                defaultButtons.next
            ]
        },
        {
            id: 'colors-info',
            attachTo: { element: '#colors-info', on: 'bottom' },
            title: 'Deine Farben',
            text: 'Hier siehst du, welche Farbkategorien du hast. Du kannst nur Hunde mit passenden Farben buchen.',
            buttons: [
                defaultButtons.back,
                defaultButtons.next
            ]
        },
        {
            id: 'filter-bar',
            attachTo: { element: '.filter-bar', on: 'bottom' },
            title: 'Filter & Suche',
            text: 'Nutze die Filter, um Hunde nach Rasse, Größe oder Farbe zu filtern.',
            buttons: [
                defaultButtons.back,
                defaultButtons.next
            ]
        },
        {
            id: 'dog-card',
            attachTo: { element: '.dog-grid .card:first-child, #dogs-list .card:first-child', on: 'right' },
            title: 'Hunde-Karte',
            text: 'Jeder Hund hat eine eigene Karte mit Foto, Name und Eigenschaften. Die Farbe zeigt das erforderliche Erfahrungslevel.',
            buttons: [
                defaultButtons.back,
                defaultButtons.next
            ],
            beforeShowPromise: function() {
                return new Promise(resolve => {
                    // Wait for dogs to load
                    setTimeout(resolve, 500);
                });
            }
        },
        {
            id: 'booking-hint',
            title: 'Buchung starten',
            text: 'Klicke auf einen Hund, um Details zu sehen und einen Spaziergang zu buchen. Viel Spaß!',
            buttons: [
                defaultButtons.back,
                defaultButtons.done
            ]
        }
    ];

    // Admin Dashboard Tour Steps
    const adminDashboardSteps = [
        {
            id: 'admin-welcome',
            title: 'Willkommen im Admin-Bereich!',
            text: 'Diese Tour zeigt dir die wichtigsten Verwaltungsfunktionen.',
            buttons: [
                defaultButtons.cancel,
                defaultButtons.next
            ]
        },
        {
            id: 'admin-stats',
            attachTo: { element: '#stats-grid', on: 'bottom' },
            title: 'Statistik-Übersicht',
            text: 'Hier siehst du wichtige Kennzahlen: Aktive Benutzer, Buchungen, verfügbare Hunde und mehr.',
            buttons: [
                defaultButtons.back,
                defaultButtons.next
            ]
        },
        {
            id: 'admin-nav-dogs',
            attachTo: { element: 'nav a[href="/admin-dogs.html"], #main-nav a[href="/admin-dogs.html"]', on: 'bottom' },
            title: 'Hunde verwalten',
            text: 'Hier kannst du Hunde hinzufügen, bearbeiten oder die Verfügbarkeit ändern.',
            buttons: [
                defaultButtons.back,
                defaultButtons.next
            ]
        },
        {
            id: 'admin-nav-users',
            attachTo: { element: '.nav-dropdown-menu a[href="/admin-users.html"]', on: 'bottom' },
            title: 'Benutzer verwalten',
            text: 'Verwalte Benutzerkonten, vergebe Berechtigungen und bearbeite Erfahrungslevel.',
            buttons: [
                defaultButtons.back,
                defaultButtons.next
            ]
        },
        {
            id: 'admin-nav-bookings',
            attachTo: { element: '.nav-dropdown-menu a[href="/admin-bookings.html"]', on: 'bottom' },
            title: 'Buchungen & Termine',
            text: 'Im Buchungsbereich findest du alle Termine, Genehmigungen und gesperrte Tage.',
            buttons: [
                defaultButtons.back,
                defaultButtons.next
            ]
        },
        {
            id: 'admin-nav-settings',
            attachTo: { element: '.nav-dropdown-menu a[href="/admin-settings.html"]', on: 'bottom' },
            title: 'Einstellungen',
            text: 'Konfiguriere Systemeinstellungen, Branding und Buchungszeiten.',
            buttons: [
                defaultButtons.back,
                defaultButtons.done
            ]
        }
    ];

    // Track retry attempts
    let shepherdRetryCount = 0;
    const MAX_SHEPHERD_RETRIES = 10; // Max 2 seconds of retries

    // Initialize tour for current page
    function initTour() {
        const path = window.location.pathname;
        let tour = null;
        let tourType = 'user';

        // Wait for Shepherd to be available (with max retries)
        if (typeof Shepherd === 'undefined') {
            shepherdRetryCount++;
            if (shepherdRetryCount <= MAX_SHEPHERD_RETRIES) {
                console.log(`[GassigeherTour] Shepherd.js not yet loaded, retry ${shepherdRetryCount}/${MAX_SHEPHERD_RETRIES}...`);
                setTimeout(initTour, 200);
                return null;
            } else {
                console.warn('[GassigeherTour] Shepherd.js failed to load after max retries. Tour disabled.');
                return null;
            }
        }

        console.log('[GassigeherTour] Initializing tour for path:', path);
        console.log('[GassigeherTour] Demo tenant:', isDemoTenant());
        console.log('[GassigeherTour] User tour complete:', isTourComplete('user'));
        console.log('[GassigeherTour] Admin tour complete:', isTourComplete('admin'));

        // Determine which tour to show based on page
        if (path.includes('admin-dashboard')) {
            if (!isTourComplete('admin')) {
                console.log('[GassigeherTour] Starting admin tour...');
                tour = createTour('admin');
                if (tour) {
                    adminDashboardSteps.forEach(step => tour.addStep(step));
                    tourType = 'admin';
                }
            }
        } else if (path.includes('dashboard') && !path.includes('admin')) {
            if (!isTourComplete('user')) {
                console.log('[GassigeherTour] Starting user dashboard tour...');
                tour = createTour('user');
                if (tour) {
                    userDashboardSteps.forEach(step => tour.addStep(step));
                }
            }
        } else if (path.includes('dogs.html') && !path.includes('admin')) {
            if (!isTourComplete('user')) {
                console.log('[GassigeherTour] Starting dogs page tour...');
                tour = createTour('user');
                if (tour) {
                    dogsPageSteps.forEach(step => tour.addStep(step));
                }
            }
        }

        // Start tour after a short delay to ensure page is loaded
        if (tour) {
            setTimeout(() => {
                try {
                    // The first step (welcome) has no attachTo, so just start
                    const firstStep = tour.steps[0];
                    if (firstStep) {
                        console.log('[GassigeherTour] Starting tour with first step:', firstStep.id);
                        tour.start();
                    }
                } catch (e) {
                    console.error('[GassigeherTour] Error starting tour:', e);
                }
            }, 800);
        } else {
            console.log('[GassigeherTour] No tour to start (already complete or wrong page)');
        }

        return tour;
    }

    // Create replay button
    function createReplayButton(tourType) {
        const btn = document.createElement('button');
        btn.className = 'btn btn-secondary';
        btn.innerHTML = '🎓 Tour wiederholen';
        btn.onclick = function() {
            resetTour(tourType);
            location.reload();
        };
        return btn;
    }

    // Export functions to global scope
    window.GassigeherTour = {
        init: initTour,
        isDemoTenant: isDemoTenant,
        isTourComplete: isTourComplete,
        resetTour: resetTour,
        createReplayButton: createReplayButton,
        // Step definitions for customization
        steps: {
            userDashboard: userDashboardSteps,
            dogsPage: dogsPageSteps,
            adminDashboard: adminDashboardSteps
        }
    };

    // Auto-initialize when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', initTour);
    } else {
        // DOM already loaded
        setTimeout(initTour, 100);
    }
})();
