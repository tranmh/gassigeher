/**
 * Help Tooltips Component
 * Provides contextual help tooltips for UI elements.
 * Usage: Add data-help="key" attribute to any element.
 */

const HelpTooltips = {
    // Track initialization to prevent duplicate event listeners
    _initialized: false,
    _escapeHandler: null,
    _clickOutsideHandler: null,

    /**
     * Escape HTML to prevent XSS attacks
     * @param {string} text - Text to escape
     * @returns {string} Escaped text safe for HTML insertion
     */
    escapeHtml(text) {
        if (text === null || text === undefined) return '';
        const div = document.createElement('div');
        div.textContent = String(text);
        return div.innerHTML;
    },

    // Tooltip content keyed by data-help value
    content: {
        // Experience levels
        'experience_locked': {
            title: 'Gesperrter Hund',
            text: 'Dieser Hund erfordert eine Farbe, die du noch nicht hast. Du kannst die benötigte Farbe über dein Profil beantragen.'
        },

        // Booking
        'booking_advance_days': {
            title: 'Vorlaufzeit',
            text: 'Wie viele Tage im Voraus können Buchungen vorgenommen werden? Standard: 14 Tage.'
        },
        'booking_cancellation': {
            title: 'Stornierungsfrist',
            text: 'Buchungen können bis zu dieser Zeit vor dem Termin kostenlos storniert werden.'
        },
        'booking_approval': {
            title: 'Genehmigung erforderlich',
            text: 'Manche Zeiten oder Hunde erfordern eine Genehmigung durch einen Administrator vor der Buchung.'
        },
        'booking_time_morning': {
            title: 'Vormittag',
            text: 'Morgendliche Spaziergänge. Die genauen Zeiten werden vom Tierheim festgelegt.'
        },
        'booking_time_afternoon': {
            title: 'Nachmittag',
            text: 'Nachmittagsspaziergänge. Oft die beliebteste Zeit.'
        },
        'booking_time_evening': {
            title: 'Abend',
            text: 'Abendspaziergänge. Nicht an allen Tagen verfügbar.'
        },

        // Dog attributes
        'dog_size_small': {
            title: 'Klein',
            text: 'Hunde unter 10 kg. Leicht zu führen, ideal für Anfänger.'
        },
        'dog_size_medium': {
            title: 'Mittel',
            text: 'Hunde zwischen 10-25 kg. Die häufigste Größenkategorie.'
        },
        'dog_size_large': {
            title: 'Groß',
            text: 'Hunde über 25 kg. Erfordert mehr Kraft und Erfahrung.'
        },
        'dog_featured': {
            title: 'Vorgestellter Hund',
            text: 'Dieser Hund wird auf der Startseite hervorgehoben und sucht besonders dringend nach Gassigehern.'
        },
        'dog_external_link': {
            title: 'Externer Link',
            text: 'Link zur Tierheim-Webseite mit weiteren Informationen über diesen Hund.'
        },

        // Account
        'account_verification': {
            title: 'E-Mail-Verifizierung',
            text: 'Deine E-Mail-Adresse muss verifiziert sein, um Buchungen vorzunehmen. Prüfe deinen Posteingang.'
        },
        'account_deactivation': {
            title: 'Auto-Deaktivierung',
            text: 'Konten werden nach längerer Inaktivität automatisch deaktiviert. Du kannst jederzeit eine Reaktivierung beantragen.'
        },
        'account_level_request': {
            title: 'Höherstufung beantragen',
            text: 'Nach einigen erfolgreichen Spaziergängen kannst du eine höhere Erfahrungsstufe beantragen.'
        },

        // Admin settings
        'admin_blocked_dates': {
            title: 'Gesperrte Termine',
            text: 'An diesen Tagen sind keine Buchungen möglich (Feiertage, Veranstaltungen, etc.).'
        },
        'admin_booking_times': {
            title: 'Buchungszeiten',
            text: 'Lege fest, zu welchen Zeiten an welchen Tagen Spaziergänge gebucht werden können.'
        },
        'admin_holidays': {
            title: 'Feiertage',
            text: 'Feiertage werden automatisch erkannt. Du kannst auch eigene Feiertage hinzufügen.'
        },
        'admin_auto_deactivation': {
            title: 'Auto-Deaktivierung',
            text: 'Nach wie vielen Tagen Inaktivität werden Nutzerkonten automatisch deaktiviert?'
        },

        // Colors/Categories
        'color_category': {
            title: 'Farbkategorie',
            text: 'Hunde werden nach Erfahrungsanforderungen in Farbkategorien eingeteilt. Je höher die Stufe, desto anspruchsvoller der Hund.'
        },
        'color_pattern': {
            title: 'Muster-Symbol',
            text: 'Zusätzlich zur Farbe wird ein Symbol verwendet, um auch bei Farbenblindheit die Kategorie zu erkennen.'
        }
    },

    // Active tooltip element
    activeTooltip: null,

    /**
     * Initialize tooltips on page
     * @param {Object} options - Configuration options
     */
    init(options = {}) {
        const defaults = {
            selector: '[data-help]',
            position: 'top',
            trigger: 'click', // 'click' or 'hover'
            showIcon: true
        };

        this.options = { ...defaults, ...options };

        // Add help icons to elements
        if (this.options.showIcon) {
            this.addHelpIcons();
        }

        // Bind event listeners
        this.bindEvents();

        // Only add document-level listeners once to prevent duplicates
        if (!this._initialized) {
            this._initialized = true;

            // Close on escape key
            this._escapeHandler = (e) => {
                if (e.key === 'Escape') {
                    this.hideTooltip();
                }
            };
            document.addEventListener('keydown', this._escapeHandler);

            // Close on click outside
            this._clickOutsideHandler = (e) => {
                if (this.activeTooltip && !e.target.closest('.help-tooltip') && !e.target.closest('[data-help]')) {
                    this.hideTooltip();
                }
            };
            document.addEventListener('click', this._clickOutsideHandler);
        }
    },

    /**
     * Cleanup event listeners (useful for SPA or re-initialization)
     */
    cleanup() {
        if (this._escapeHandler) {
            document.removeEventListener('keydown', this._escapeHandler);
            this._escapeHandler = null;
        }
        if (this._clickOutsideHandler) {
            document.removeEventListener('click', this._clickOutsideHandler);
            this._clickOutsideHandler = null;
        }
        this._initialized = false;
        this.hideTooltip();
    },

    /**
     * Add help icons to elements with data-help attribute
     */
    addHelpIcons() {
        document.querySelectorAll(this.options.selector).forEach(el => {
            // Skip if already has icon
            if (el.querySelector('.help-icon')) return;

            // Skip if element is an icon itself
            if (el.classList.contains('help-icon')) return;

            const icon = document.createElement('span');
            icon.className = 'help-icon';
            icon.innerHTML = '?';
            icon.style.cssText = `
                display: inline-flex;
                align-items: center;
                justify-content: center;
                width: 16px;
                height: 16px;
                border-radius: 50%;
                background: var(--text-gray, #666);
                color: white;
                font-size: 11px;
                font-weight: bold;
                margin-left: 6px;
                cursor: help;
                vertical-align: middle;
                flex-shrink: 0;
            `;
            icon.setAttribute('data-help', el.getAttribute('data-help'));
            icon.setAttribute('aria-label', 'Hilfe');
            icon.setAttribute('role', 'button');
            icon.setAttribute('tabindex', '0');

            el.appendChild(icon);
        });
    },

    /**
     * Bind event listeners
     */
    bindEvents() {
        const trigger = this.options.trigger;

        document.querySelectorAll(this.options.selector + ', .help-icon').forEach(el => {
            if (trigger === 'click') {
                el.addEventListener('click', (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    this.toggleTooltip(el);
                });

                // Keyboard support
                el.addEventListener('keydown', (e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault();
                        this.toggleTooltip(el);
                    }
                });
            } else {
                el.addEventListener('mouseenter', () => this.showTooltip(el));
                el.addEventListener('mouseleave', () => this.hideTooltip());
                el.addEventListener('focus', () => this.showTooltip(el));
                el.addEventListener('blur', () => this.hideTooltip());
            }
        });
    },

    /**
     * Toggle tooltip visibility
     * @param {HTMLElement} el - Target element
     */
    toggleTooltip(el) {
        if (this.activeTooltip && this.activeTooltip.targetEl === el) {
            this.hideTooltip();
        } else {
            this.showTooltip(el);
        }
    },

    /**
     * Show tooltip for element
     * @param {HTMLElement} el - Target element
     */
    showTooltip(el) {
        this.hideTooltip();

        const key = el.getAttribute('data-help');
        const content = this.content[key];

        if (!content) {
            console.warn(`HelpTooltips: No content found for key "${key}"`);
            return;
        }

        // Create tooltip container
        const tooltip = document.createElement('div');
        tooltip.className = 'help-tooltip';
        tooltip.style.cssText = `
            position: fixed;
            z-index: 10001;
            max-width: 280px;
            padding: 12px 16px;
            background: var(--card-bg, white);
            border: 1px solid var(--border-color, #e2e8f0);
            border-radius: var(--border-radius, 6px);
            box-shadow: 0 4px 12px rgba(0,0,0,0.15);
            font-size: 0.875rem;
            line-height: 1.5;
        `;

        // Create title element with textContent to prevent XSS
        const titleEl = document.createElement('div');
        titleEl.className = 'help-tooltip-title';
        titleEl.textContent = content.title; // Safe - uses textContent
        titleEl.style.cssText = `
            font-weight: 600;
            color: var(--text-dark, #1a202c);
            margin-bottom: 6px;
        `;
        tooltip.appendChild(titleEl);

        // Create text element with textContent to prevent XSS
        const textEl = document.createElement('div');
        textEl.className = 'help-tooltip-text';
        textEl.textContent = content.text; // Safe - uses textContent
        textEl.style.cssText = `
            color: var(--text-gray, #666);
        `;
        tooltip.appendChild(textEl);

        document.body.appendChild(tooltip);

        // Position tooltip
        this.positionTooltip(tooltip, el);

        // Store reference
        this.activeTooltip = { element: tooltip, targetEl: el };
    },

    /**
     * Position tooltip relative to target element
     * @param {HTMLElement} tooltip - Tooltip element
     * @param {HTMLElement} target - Target element
     */
    positionTooltip(tooltip, target) {
        const targetRect = target.getBoundingClientRect();
        const tooltipRect = tooltip.getBoundingClientRect();
        const padding = 8;

        let top, left;
        const position = this.options.position;

        // Calculate position
        if (position === 'top') {
            top = targetRect.top - tooltipRect.height - padding;
            left = targetRect.left + (targetRect.width / 2) - (tooltipRect.width / 2);
        } else if (position === 'bottom') {
            top = targetRect.bottom + padding;
            left = targetRect.left + (targetRect.width / 2) - (tooltipRect.width / 2);
        } else if (position === 'left') {
            top = targetRect.top + (targetRect.height / 2) - (tooltipRect.height / 2);
            left = targetRect.left - tooltipRect.width - padding;
        } else if (position === 'right') {
            top = targetRect.top + (targetRect.height / 2) - (tooltipRect.height / 2);
            left = targetRect.right + padding;
        }

        // Ensure tooltip stays within viewport
        const viewportWidth = window.innerWidth;
        const viewportHeight = window.innerHeight;

        // Horizontal bounds
        if (left < padding) {
            left = padding;
        } else if (left + tooltipRect.width > viewportWidth - padding) {
            left = viewportWidth - tooltipRect.width - padding;
        }

        // Vertical bounds - flip if needed
        if (top < padding) {
            // Flip to bottom
            top = targetRect.bottom + padding;
        } else if (top + tooltipRect.height > viewportHeight - padding) {
            // Flip to top
            top = targetRect.top - tooltipRect.height - padding;
        }

        tooltip.style.top = `${top}px`;
        tooltip.style.left = `${left}px`;
    },

    /**
     * Hide active tooltip
     */
    hideTooltip() {
        if (this.activeTooltip) {
            this.activeTooltip.element.remove();
            this.activeTooltip = null;
        }
    },

    /**
     * Update content dynamically (e.g., from i18n)
     * @param {Object} newContent - New content object
     */
    updateContent(newContent) {
        this.content = { ...this.content, ...newContent };
    },

    /**
     * Load content from i18n
     */
    loadFromI18n() {
        if (window.i18n && window.i18n.translations && window.i18n.translations.help_tooltips) {
            this.updateContent(window.i18n.translations.help_tooltips);
        }
    }
};

// Auto-initialize on DOM ready
document.addEventListener('DOMContentLoaded', () => {
    // Initialize with default options
    HelpTooltips.init();

    // Try to load from i18n after it loads
    if (window.i18n) {
        window.i18n.load().then(() => {
            HelpTooltips.loadFromI18n();
        }).catch(() => {
            // i18n not available, use built-in content
        });
    }
});

// Make available globally
window.HelpTooltips = HelpTooltips;
