/**
 * Booking Error Explainer
 * Provides user-friendly explanations for booking failures with actionable solutions.
 * Replaces generic error messages with specific guidance.
 */

const BookingErrors = {
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

    /**
     * Validate and sanitize href to prevent javascript: XSS
     * @param {string} href - URL to validate
     * @returns {string} Safe URL or '#' if invalid
     */
    sanitizeHref(href) {
        if (!href || typeof href !== 'string') return '#';
        const trimmed = href.trim().toLowerCase();
        // Only allow safe protocols
        if (trimmed.startsWith('/') ||
            trimmed.startsWith('http://') ||
            trimmed.startsWith('https://') ||
            trimmed.startsWith('#')) {
            return href;
        }
        // Block javascript:, data:, vbscript:, etc.
        return '#';
    },

    // Error code to explanation mapping
    errorMap: {
        // Level/Permission errors
        'level_too_low': {
            title: 'Erfahrungsstufe nicht ausreichend',
            message: 'Dieser Hund erfordert eine höhere Erfahrungsstufe.',
            solution: 'Du kannst eine Höherstufung beantragen, nachdem du genügend Spaziergänge absolviert hast.',
            action: {
                text: 'Höherstufung beantragen',
                href: '/profile.html#experience'
            },
            icon: '🔒'
        },
        'user_not_verified': {
            title: 'E-Mail nicht verifiziert',
            message: 'Dein Konto muss verifiziert sein, um Buchungen vorzunehmen.',
            solution: 'Bitte überprüfe deinen Posteingang und klicke auf den Bestätigungslink.',
            action: {
                text: 'Bestätigungsmail erneut senden',
                handler: 'resendVerification'
            },
            icon: '📧'
        },
        'user_inactive': {
            title: 'Konto deaktiviert',
            message: 'Dein Konto ist derzeit deaktiviert.',
            solution: 'Du kannst eine Reaktivierung beantragen.',
            action: {
                text: 'Reaktivierung beantragen',
                href: '/profile.html#reactivation'
            },
            icon: '⏸️'
        },

        // Dog availability errors
        'dog_unavailable': {
            title: 'Hund nicht verfügbar',
            message: 'Dieser Hund ist derzeit nicht für Spaziergänge verfügbar.',
            solution: 'Der Hund wurde möglicherweise temporär deaktiviert (z.B. wegen Krankheit oder Adoption).',
            action: {
                text: 'Andere Hunde ansehen',
                href: '/dogs.html'
            },
            icon: '🐕'
        },
        'dog_not_found': {
            title: 'Hund nicht gefunden',
            message: 'Dieser Hund existiert nicht mehr in unserem System.',
            solution: 'Der Hund wurde möglicherweise adoptiert oder aus dem System entfernt.',
            action: {
                text: 'Verfügbare Hunde ansehen',
                href: '/dogs.html'
            },
            icon: '❓'
        },

        // Date/Time errors
        'date_in_past': {
            title: 'Datum liegt in der Vergangenheit',
            message: 'Du kannst keine Buchungen für vergangene Termine vornehmen.',
            solution: 'Bitte wähle ein Datum ab heute.',
            icon: '📅'
        },
        'date_too_far_ahead': {
            title: 'Datum zu weit in der Zukunft',
            message: 'Buchungen können nur bis zu {days} Tage im Voraus erfolgen.',
            solution: 'Bitte wähle ein Datum innerhalb des erlaubten Zeitraums.',
            icon: '📆'
        },
        'date_blocked': {
            title: 'Datum gesperrt',
            message: 'An diesem Datum sind keine Buchungen möglich.',
            solution: 'Das Tierheim ist an diesem Tag geschlossen (Feiertag, Veranstaltung o.ä.).',
            action: {
                text: 'Nächsten verfügbaren Tag finden',
                handler: 'findNextAvailable'
            },
            icon: '🚫'
        },
        'time_blocked': {
            title: 'Zeitraum nicht verfügbar',
            message: 'Dieser Zeitraum ist nicht für Buchungen freigegeben.',
            solution: 'Bitte wähle einen anderen Zeitraum.',
            icon: '⏰'
        },

        // Booking conflicts
        'already_booked': {
            title: 'Bereits gebucht',
            message: 'Dieser Hund ist zu diesem Zeitpunkt bereits gebucht.',
            solution: 'Ein anderer Gassigeher hat diesen Termin bereits reserviert.',
            action: {
                text: 'Meine Buchungen ansehen',
                href: '/dashboard.html'
            },
            icon: '📋'
        },
        'double_booking': {
            title: 'Doppelbuchung',
            message: 'Du hast zu diesem Zeitpunkt bereits eine andere Buchung.',
            solution: 'Bitte wähle einen anderen Termin oder storniere die bestehende Buchung.',
            action: {
                text: 'Meine Buchungen verwalten',
                href: '/dashboard.html'
            },
            icon: '⚠️'
        },
        'max_bookings_reached': {
            title: 'Buchungslimit erreicht',
            message: 'Du hast die maximale Anzahl gleichzeitiger Buchungen erreicht.',
            solution: 'Bitte warte bis bestehende Buchungen abgeschlossen sind oder storniere eine.',
            action: {
                text: 'Buchungen verwalten',
                href: '/dashboard.html'
            },
            icon: '🔢'
        },
        'daily_dog_limit': {
            title: 'Tageslimit erreicht',
            // message and solution are dynamically generated based on existing bookings
            message: '',
            solution: '',
            icon: '📅',
            customRender: true  // Flag to use custom rendering
        },

        // Approval errors
        'approval_required': {
            title: 'Genehmigung erforderlich',
            message: 'Diese Buchung erfordert eine Genehmigung durch einen Administrator.',
            solution: 'Deine Anfrage wurde eingereicht. Du erhältst eine Benachrichtigung, sobald sie bearbeitet wurde.',
            icon: '⏳'
        },

        // Generic errors
        'validation_error': {
            title: 'Ungültige Eingabe',
            message: 'Bitte überprüfe deine Eingaben.',
            solution: 'Stelle sicher, dass alle Pflichtfelder korrekt ausgefüllt sind.',
            icon: '❌'
        },
        'server_error': {
            title: 'Serverfehler',
            message: 'Bei der Verarbeitung ist ein Fehler aufgetreten.',
            solution: 'Bitte versuche es später erneut. Falls das Problem weiterhin besteht, kontaktiere uns.',
            action: {
                text: 'Hilfe-Center',
                href: '/help.html'
            },
            icon: '🔧'
        },
        'network_error': {
            title: 'Verbindungsfehler',
            message: 'Die Verbindung zum Server konnte nicht hergestellt werden.',
            solution: 'Bitte überprüfe deine Internetverbindung und versuche es erneut.',
            icon: '📡'
        }
    },

    /**
     * Parse error response and return structured error info
     * @param {Object|string} error - Error from API or error message
     * @param {Object} context - Additional context (dog, date, etc.)
     * @returns {Object} Structured error information
     */
    parseError(error, context = {}) {
        let errorCode = 'server_error';
        let serverMessage = '';

        // Check for structured daily_dog_limit error from backend
        // API client attaches response data to error.data
        const errorData = error?.data || error;
        if (errorData?.error_type === 'daily_dog_limit') {
            return this.getDailyDogLimitErrorInfo(errorData, context);
        }

        // Extract error message from various formats
        if (typeof error === 'string') {
            serverMessage = error.toLowerCase();
        } else if (error?.message) {
            serverMessage = error.message.toLowerCase();
        } else if (error?.error) {
            serverMessage = error.error.toLowerCase();
        }

        // Map server messages to error codes
        if (serverMessage.includes('experience') || serverMessage.includes('level') || serverMessage.includes('erfahrung')) {
            errorCode = 'level_too_low';
        } else if (serverMessage.includes('verified') || serverMessage.includes('verifizier')) {
            errorCode = 'user_not_verified';
        } else if (serverMessage.includes('inactive') || serverMessage.includes('deaktiv')) {
            errorCode = 'user_inactive';
        } else if (serverMessage.includes('unavailable') || serverMessage.includes('nicht verfügbar')) {
            errorCode = 'dog_unavailable';
        } else if (serverMessage.includes('not found') || serverMessage.includes('nicht gefunden')) {
            errorCode = 'dog_not_found';
        } else if (serverMessage.includes('past') || serverMessage.includes('vergangenheit')) {
            errorCode = 'date_in_past';
        } else if (serverMessage.includes('advance') || serverMessage.includes('voraus') || serverMessage.includes('too far')) {
            errorCode = 'date_too_far_ahead';
        } else if ((serverMessage.includes('blocked') && serverMessage.includes('time')) ||
                   (serverMessage.includes('gesperrt') && serverMessage.includes('zeit'))) {
            // Check time_blocked BEFORE date_blocked to handle "Zeit gesperrt" correctly
            errorCode = 'time_blocked';
        } else if ((serverMessage.includes('blocked') && serverMessage.includes('date')) ||
                   (serverMessage.includes('gesperrt') && (serverMessage.includes('datum') || !serverMessage.includes('zeit')))) {
            // "gesperrt" without "zeit" defaults to date_blocked
            errorCode = 'date_blocked';
        } else if (serverMessage.includes('already booked') || serverMessage.includes('bereits gebucht')) {
            errorCode = 'already_booked';
        } else if (serverMessage.includes('double') || serverMessage.includes('doppel')) {
            errorCode = 'double_booking';
        } else if (serverMessage.includes('maximum') || serverMessage.includes('limit')) {
            errorCode = 'max_bookings_reached';
        } else if (serverMessage.includes('approval') || serverMessage.includes('genehmigung')) {
            errorCode = 'approval_required';
        } else if (serverMessage.includes('validation') || serverMessage.includes('invalid')) {
            errorCode = 'validation_error';
        } else if (serverMessage.includes('network') || serverMessage.includes('fetch')) {
            errorCode = 'network_error';
        }

        return this.getErrorInfo(errorCode, context);
    },

    /**
     * Get error information by code
     * @param {string} code - Error code
     * @param {Object} context - Additional context for template replacement
     * @returns {Object} Error information
     */
    getErrorInfo(code, context = {}) {
        const errorDef = this.errorMap[code] || this.errorMap['server_error'];

        // Deep clone to avoid mutating the original (especially action object)
        const errorInfo = {
            ...errorDef,
            code,
            // Deep clone action if it exists
            action: errorDef.action ? { ...errorDef.action } : undefined
        };

        // Replace template variables (use String() to handle non-string context values)
        if (context.days !== undefined) {
            errorInfo.message = errorInfo.message.replace('{days}', String(context.days));
        }
        if (context.dog !== undefined) {
            errorInfo.message = errorInfo.message.replace('{dog}', String(context.dog));
        }
        if (context.date !== undefined) {
            errorInfo.message = errorInfo.message.replace('{date}', String(context.date));
        }

        return errorInfo;
    },

    /**
     * Get error info for daily dog booking limit exceeded
     * Creates custom message with existing bookings list
     * @param {Object} errorData - Structured error from backend with existing_bookings
     * @param {Object} context - Additional context (dog name, date)
     * @returns {Object} Error information with custom HTML content
     */
    getDailyDogLimitErrorInfo(errorData, context = {}) {
        const errorDef = this.errorMap['daily_dog_limit'];
        const existingBookings = errorData.existing_bookings || [];
        const maxAllowed = errorData.max_allowed || 2;
        const currentCount = errorData.current_count || existingBookings.length;

        // Build the existing bookings list HTML
        let bookingsListHtml = '';
        if (existingBookings.length > 0) {
            bookingsListHtml = '<div style="text-align: left; margin: 12px 0; padding: 12px; background: var(--bg-light, #f7fafc); border-radius: 6px;">';
            bookingsListHtml += '<strong>Dieser Hund ist bereits gebucht für:</strong><ul style="margin: 8px 0 0 0; padding-left: 20px;">';
            for (const booking of existingBookings) {
                const periodName = this.escapeHtml(booking.period_name);
                const startTime = this.escapeHtml(booking.start_time);
                const endTime = this.escapeHtml(booking.end_time);
                if (endTime) {
                    bookingsListHtml += `<li>${periodName}: ${startTime} - ${endTime}</li>`;
                } else {
                    bookingsListHtml += `<li>${periodName}: ${startTime}</li>`;
                }
            }
            bookingsListHtml += '</ul></div>';
        }

        // Build the main message
        const message = `Dieser Hund wurde für heute bereits ${currentCount} Mal gebucht.`;
        const solution = `Das Tageslimit von ${maxAllowed} Buchungen pro Hund ist erreicht. Weitere Buchungen sind für diesen Tag nicht möglich.`;

        return {
            ...errorDef,
            code: 'daily_dog_limit',
            message: message,
            solution: solution,
            customHtml: bookingsListHtml
        };
    },

    /**
     * Show error modal with explanation and action button
     * @param {Object} errorInfo - Structured error information
     * @param {HTMLElement} container - Optional container for inline display
     */
    showError(errorInfo, container = null) {
        const html = this.renderError(errorInfo);

        if (container) {
            // Inline display
            container.innerHTML = html;
            container.style.display = 'block';
        } else {
            // Modal display
            this.showModal(errorInfo);
        }
    },

    /**
     * Render error HTML
     * @param {Object} errorInfo - Error information
     * @returns {string} HTML string
     */
    renderError(errorInfo) {
        // Escape all text content to prevent XSS
        const safeTitle = this.escapeHtml(errorInfo.title);
        const safeMessage = this.escapeHtml(errorInfo.message);
        const safeSolution = this.escapeHtml(errorInfo.solution);
        const safeIcon = this.escapeHtml(errorInfo.icon);

        let actionHtml = '';
        if (errorInfo.action) {
            const safeActionText = this.escapeHtml(errorInfo.action.text);
            if (errorInfo.action.href) {
                // Sanitize href to prevent javascript: XSS
                const safeHref = this.sanitizeHref(errorInfo.action.href);
                actionHtml = `
                    <a href="${safeHref}" class="btn" style="margin-top: 16px;">
                        ${safeActionText}
                    </a>
                `;
            } else if (errorInfo.action.handler) {
                // Handler is from our own code, but escape anyway for safety
                const safeHandler = this.escapeHtml(errorInfo.action.handler);
                actionHtml = `
                    <button type="button" class="btn" style="margin-top: 16px;"
                            data-action="booking-error-action" data-handler="${safeHandler}">
                        ${safeActionText}
                    </button>
                `;
            }
        }

        // Include custom HTML content if present (already sanitized in getDailyDogLimitErrorInfo)
        const customHtml = errorInfo.customHtml || '';

        return `
            <div class="booking-error" style="
                padding: 20px;
                background: var(--error-bg, #fff5f5);
                border: 1px solid var(--error-border, #fed7d7);
                border-radius: var(--border-radius);
                text-align: center;
            ">
                <div style="font-size: 2rem; margin-bottom: 12px;">${safeIcon}</div>
                <h3 style="margin: 0 0 8px; color: var(--error-title, #c53030);">
                    ${safeTitle}
                </h3>
                ${customHtml}
                <p style="margin: 0 0 12px; color: var(--text-dark);">
                    ${safeMessage}
                </p>
                <p style="margin: 0; color: var(--text-gray); font-size: 0.9rem;">
                    💡 ${safeSolution}
                </p>
                ${actionHtml}
            </div>
        `;
    },

    /**
     * Show error in a modal dialog
     * @param {Object} errorInfo - Error information
     */
    showModal(errorInfo) {
        // Remove existing modal
        const existing = document.getElementById('booking-error-modal');
        if (existing) existing.remove();

        const modal = document.createElement('div');
        modal.id = 'booking-error-modal';
        modal.style.cssText = `
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: rgba(0,0,0,0.5);
            display: flex;
            align-items: center;
            justify-content: center;
            z-index: 10000;
            padding: 20px;
        `;

        modal.innerHTML = `
            <div style="
                background: var(--card-bg, white);
                border-radius: var(--border-radius);
                max-width: 450px;
                width: 100%;
                position: relative;
                box-shadow: 0 4px 20px rgba(0,0,0,0.15);
            ">
                <button type="button" data-action="close-booking-error" style="
                    position: absolute;
                    top: 12px;
                    right: 12px;
                    background: none;
                    border: none;
                    font-size: 1.5rem;
                    cursor: pointer;
                    color: var(--text-gray);
                    line-height: 1;
                    padding: 4px;
                " aria-label="Schließen">&times;</button>
                ${this.renderError(errorInfo)}
            </div>
        `;

        // Close on backdrop click
        modal.addEventListener('click', (e) => {
            if (e.target === modal) {
                modal.remove();
            }
        });

        // Close on button click
        modal.querySelector('[data-action="close-booking-error"]').addEventListener('click', () => {
            modal.remove();
        });

        // Handle action buttons
        const actionBtn = modal.querySelector('[data-action="booking-error-action"]');
        if (actionBtn) {
            actionBtn.addEventListener('click', () => {
                const handler = actionBtn.dataset.handler;
                if (this.actionHandlers[handler]) {
                    this.actionHandlers[handler]();
                }
                modal.remove();
            });
        }

        document.body.appendChild(modal);

        // Close on escape
        const escHandler = (e) => {
            if (e.key === 'Escape') {
                modal.remove();
                document.removeEventListener('keydown', escHandler);
            }
        };
        document.addEventListener('keydown', escHandler);
    },

    /**
     * Action handlers for error action buttons
     */
    actionHandlers: {
        resendVerification: async function() {
            try {
                if (window.api && window.api.resendVerification) {
                    await window.api.resendVerification();
                    alert('Bestätigungsmail wurde erneut gesendet!');
                } else {
                    window.location.href = '/profile.html';
                }
            } catch (err) {
                console.error('Failed to resend verification:', err);
                alert('Fehler beim Senden. Bitte versuche es später erneut.');
            }
        },

        findNextAvailable: function() {
            // Navigate to calendar with next available date
            window.location.href = '/calendar.html';
        }
    },

    /**
     * Quick helper to show a parsed error
     * @param {Object|string} error - Error from API
     * @param {Object} context - Additional context
     */
    show(error, context = {}) {
        const errorInfo = this.parseError(error, context);
        this.showError(errorInfo);
    },

    /**
     * Quick helper to show inline error
     * @param {Object|string} error - Error from API
     * @param {HTMLElement} container - Container element
     * @param {Object} context - Additional context
     */
    showInline(error, container, context = {}) {
        const errorInfo = this.parseError(error, context);
        this.showError(errorInfo, container);
    },

    /**
     * Hide inline error
     * @param {HTMLElement} container - Container element
     */
    hideInline(container) {
        if (container) {
            container.innerHTML = '';
            container.style.display = 'none';
        }
    }
};

// Make available globally
window.BookingErrors = BookingErrors;
