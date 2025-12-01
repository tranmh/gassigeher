// Mobile navigation menu functionality
// This script handles the hamburger menu toggle and area switcher visibility

// Mobile menu toggle function
function toggleMenu() {
    console.log('toggleMenu called');
    const nav = document.getElementById('main-nav');
    const overlay = document.getElementById('nav-overlay');
    console.log('nav element:', nav);
    console.log('overlay element:', overlay);
    if (nav && overlay) {
        nav.classList.toggle('active');
        overlay.classList.toggle('active');
        console.log('Toggled active class - nav now:', nav.classList.contains('active') ? 'OPEN' : 'CLOSED');
    } else {
        console.error('Could not find nav or overlay elements!');
    }
}

// Close menu when clicking on a link (except logout which is handled separately)
document.addEventListener('click', function(e) {
    if (e.target.tagName === 'A' && e.target.getAttribute('href') !== '#') {
        const nav = document.getElementById('main-nav');
        const overlay = document.getElementById('nav-overlay');
        if (nav && overlay && nav.classList.contains('active')) {
            nav.classList.remove('active');
            overlay.classList.remove('active');
        }
    }
});

// Show admin area link if user is admin (for user pages)
function showAdminLinkIfAdmin(user) {
    if (user && user.is_admin) {
        const adminLink = document.getElementById('admin-area-link');
        if (adminLink) {
            adminLink.style.display = 'list-item';
            // Update translations for the admin link after making it visible
            if (window.i18n && window.i18n.updateElement) {
                window.i18n.updateElement(adminLink);
            }
        }
    }
}

// User area link is always visible on admin pages (no special logic needed)

// Debug logging
console.log('nav-menu.js loaded successfully');
console.log('toggleMenu function:', typeof toggleMenu);
console.log('showAdminLinkIfAdmin function:', typeof showAdminLinkIfAdmin);
