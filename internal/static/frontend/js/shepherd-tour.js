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

// Shepherd.js is loaded via CDN in HTML pages that use tours
// <script src="https://cdn.jsdelivr.net/npm/shepherd.js@13.0.0/dist/js/shepherd.min.js"></script>
// <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/shepherd.js@13.0.0/dist/css/shepherd.css"/>

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
        if (hostname.startsWith('demo.') || hostname === 'demo.gassigeher.org') {
            return true;
        }
        return localStorage.getItem(STORAGE_KEYS.isDemo) === 'true';
    }

    // Check if tour has been completed
    function isTourComplete(tourType) {
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
            attachTo: { element: 'a[href="/dogs.html"]', on: 'bottom' },
            title: 'Unsere Hunde',
            text: 'Hier findest du alle verfügbaren Hunde. Klicke auf einen Hund, um mehr über ihn zu erfahren und einen Spaziergang zu buchen.',
            buttons: [
                defaultButtons.back,
                defaultButtons.next
            ]
        },
        {
            id: 'nav-calendar',
            attachTo: { element: 'a[href="/calendar.html"]', on: 'bottom' },
            title: 'Kalender',
            text: 'Im Kalender siehst du alle deine gebuchten Spaziergänge auf einen Blick.',
            buttons: [
                defaultButtons.back,
                defaultButtons.next
            ]
        },
        {
            id: 'my-bookings',
            attachTo: { element: '#my-bookings, .bookings-section, .upcoming-walks', on: 'top' },
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
            attachTo: { element: 'a[href="/profile.html"]', on: 'bottom' },
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
            id: 'dog-card',
            attachTo: { element: '.dog-card:first-child, .dog-item:first-child', on: 'right' },
            title: 'Hunde-Karte',
            text: 'Jeder Hund hat eine eigene Karte mit Foto, Name und Eigenschaften. Die Farbe zeigt das erforderliche Erfahrungslevel.',
            buttons: [
                defaultButtons.back,
                defaultButtons.next
            ]
        },
        {
            id: 'color-legend',
            attachTo: { element: '.color-legend, .filter-section, #color-filter', on: 'bottom' },
            title: 'Farbkategorien',
            text: 'Grün = Anfänger, Gelb = Etwas Erfahrung, Orange/Blau = Fortgeschritten. Dein Level bestimmt, welche Hunde du buchen kannst.',
            buttons: [
                defaultButtons.back,
                defaultButtons.next
            ]
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
            attachTo: { element: '.stats-grid, .dashboard-stats, #stats-section', on: 'bottom' },
            title: 'Statistik-Übersicht',
            text: 'Hier siehst du wichtige Kennzahlen: Aktive Benutzer, Buchungen, verfügbare Hunde und mehr.',
            buttons: [
                defaultButtons.back,
                defaultButtons.next
            ]
        },
        {
            id: 'admin-nav-dogs',
            attachTo: { element: 'a[href="/admin-dogs.html"]', on: 'bottom' },
            title: 'Hunde verwalten',
            text: 'Hier kannst du Hunde hinzufügen, bearbeiten oder die Verfügbarkeit ändern.',
            buttons: [
                defaultButtons.back,
                defaultButtons.next
            ]
        },
        {
            id: 'admin-nav-users',
            attachTo: { element: 'a[href="/admin-users.html"]', on: 'bottom' },
            title: 'Benutzer verwalten',
            text: 'Verwalte Benutzerkonten, vergebe Berechtigungen und bearbeite Erfahrungslevel.',
            buttons: [
                defaultButtons.back,
                defaultButtons.next
            ]
        },
        {
            id: 'admin-nav-bookings',
            attachTo: { element: '.nav-dropdown:has(a[href="/admin-bookings.html"]), a[href="/admin-bookings.html"]', on: 'bottom' },
            title: 'Buchungen & Termine',
            text: 'Im Buchungsbereich findest du alle Termine, Genehmigungen und gesperrte Tage.',
            buttons: [
                defaultButtons.back,
                defaultButtons.next
            ]
        },
        {
            id: 'admin-nav-settings',
            attachTo: { element: '.nav-dropdown:has(a[href="/admin-settings.html"]), a[href="/admin-settings.html"]', on: 'bottom' },
            title: 'Einstellungen',
            text: 'Konfiguriere Systemeinstellungen, Branding und Buchungszeiten.',
            buttons: [
                defaultButtons.back,
                defaultButtons.done
            ]
        }
    ];

    // Initialize tour for current page
    function initTour() {
        const path = window.location.pathname;
        let tour = null;
        let tourType = 'user';

        // Determine which tour to show based on page
        if (path.includes('admin-dashboard')) {
            if (!isTourComplete('admin')) {
                tour = createTour('admin');
                if (tour) {
                    adminDashboardSteps.forEach(step => tour.addStep(step));
                    tourType = 'admin';
                }
            }
        } else if (path.includes('dashboard') && !path.includes('admin')) {
            if (!isTourComplete('user')) {
                tour = createTour('user');
                if (tour) {
                    userDashboardSteps.forEach(step => tour.addStep(step));
                }
            }
        } else if (path.includes('dogs.html') && !path.includes('admin')) {
            if (!isTourComplete('user')) {
                tour = createTour('user');
                if (tour) {
                    dogsPageSteps.forEach(step => tour.addStep(step));
                }
            }
        }

        // Start tour after a short delay to ensure page is loaded
        if (tour) {
            setTimeout(() => {
                // Only start if elements exist
                const firstStep = tour.steps[0];
                if (firstStep && !firstStep.options.attachTo) {
                    tour.start();
                } else if (firstStep && firstStep.options.attachTo) {
                    const el = document.querySelector(firstStep.options.attachTo.element);
                    if (el || !firstStep.options.attachTo.element) {
                        tour.start();
                    }
                }
            }, 500);
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
