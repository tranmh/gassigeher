/**
 * Calendar Component - Shared calendar logic for user and admin pages
 * Displays a 14-day availability calendar for dogs with booking capabilities
 */

class CalendarComponent {
    constructor(options = {}) {
        this.options = {
            gridContainerId: 'calendar-grid',
            mobileContainerId: 'calendar-mobile',
            colorFilterId: 'color-filter',
            availabilityFilterId: 'availability-filter',
            accessFilterId: 'access-filter',
            alertContainerId: 'alert-container',
            isAdmin: false,
            onQuickBook: null, // Callback for booking action
            ...options
        };

        this.allDogs = [];
        this.bookings = [];
        this.blockedDates = [];
        this.allColors = [];
        this.userColors = [];
        this.currentView = 'grid';
    }

    /**
     * Initialize the calendar component
     */
    async init() {
        await this.loadColors();
        await this.loadCalendar();
    }

    /**
     * Set user colors (for access filtering)
     */
    setUserColors(colors) {
        this.userColors = colors || [];
    }

    /**
     * Load color categories from API
     */
    async loadColors() {
        try {
            const response = await api.getColors();
            this.allColors = response.colors || [];
            const colorSelect = document.getElementById(this.options.colorFilterId);
            if (colorSelect) {
                colorSelect.innerHTML = '<option value="">Alle Farben</option>';
                this.allColors.forEach(color => {
                    const option = document.createElement('option');
                    option.value = color.id;
                    option.textContent = `${this.getPatternIcon(color.pattern_icon)} ${color.name}`;
                    option.style.color = color.hex_code;
                    colorSelect.appendChild(option);
                });
            }
        } catch (error) {
            console.error('Failed to load colors:', error);
        }
    }

    /**
     * Load calendar data (dogs, bookings, blocked dates)
     */
    async loadCalendar() {
        try {
            const colorFilter = document.getElementById(this.options.colorFilterId)?.value || '';
            const availabilityFilter = document.getElementById(this.options.availabilityFilterId)?.value || 'available';

            // Fetch dogs with filters
            const params = {};
            if (colorFilter) {
                params.color_id = colorFilter;
            }
            if (availabilityFilter === 'available') {
                params.is_available = 'true';
            }
            this.allDogs = await api.getDogs(params);

            // Fetch bookings for next 14 days
            const today = new Date();
            const twoWeeksLater = new Date();
            twoWeeksLater.setDate(today.getDate() + 14);

            const startDate = today.toISOString().split('T')[0];
            const endDate = twoWeeksLater.toISOString().split('T')[0];

            this.bookings = await api.getBookings({
                date_from: startDate,
                date_to: endDate,
                calendar_view: 'true'
            });

            // Fetch blocked dates
            this.blockedDates = await api.getBlockedDates();

            this.renderCalendar();
            this.renderMobileView();
        } catch (error) {
            console.error('Failed to load calendar:', error);
            this.showAlert('error', 'Fehler beim Laden der Daten: ' + (error.message || 'Unbekannter Fehler'));
        }
    }

    /**
     * Render the desktop calendar grid
     */
    renderCalendar() {
        const grid = document.getElementById(this.options.gridContainerId);
        if (!grid) return;

        const dates = this.getNext14Days();

        // Build grid HTML
        let html = '';

        // Header row
        html += '<div class="calendar-header dog-name"><strong>Hunde</strong></div>';
        dates.forEach(date => {
            const dayName = this.getDayName(date);
            const dateStr = this.formatDate(date);
            const isToday = date.toDateString() === new Date().toDateString();
            const todayStyle = isToday ? ' style="border: 3px solid var(--accent-orange); box-shadow: 0 0 12px rgba(255, 140, 66, 0.4);"' : '';
            html += `<div class="calendar-header"${todayStyle}>
                <div style="font-size: 1rem;">${dayName}</div>
                <div class="date-display" style="color: rgba(255,255,255,0.9);">${dateStr}</div>
                ${isToday ? '<div style="font-size: 0.7rem; margin-top: 4px; color: #fff; background: #f59e0b; padding: 2px 6px; border-radius: 3px; font-weight: 700;">HEUTE</div>' : ''}
            </div>`;
        });

        // Filter dogs with valid names
        const validDogs = this.getFilteredDogs();

        // Dog rows
        validDogs.forEach(dog => {
            const dogColor = this.getColorForDog(dog);
            html += `<div class="calendar-cell dog-name" data-dog-id="${dog.id}">${this.getCalendarDogCell(dog, dogColor)}</div>`;

            dates.forEach(date => {
                const dateStr = date.toISOString().split('T')[0];
                const cellData = this.getCellData(dog.id, dateStr);
                html += this.renderCell(dog, dateStr, cellData);
            });
        });

        grid.innerHTML = html;
    }

    /**
     * Render mobile view
     */
    renderMobileView() {
        const mobileContainer = document.getElementById(this.options.mobileContainerId);
        if (!mobileContainer) return;

        const dates = this.getNext14Days();
        let html = '';

        // Filter dogs with valid names
        const validDogs = this.getFilteredDogs();

        validDogs.forEach(dog => {
            const dogColor = this.getColorForDog(dog);
            const colorBadge = dogColor ? this.getColorBadgeHtml(dogColor) : '';
            const safeDogName = typeof sanitizeHTML === 'function' ? sanitizeHTML(dog.name) : dog.name;
            html += `<div class="dog-card-mobile">
                <h3>${safeDogName} ${colorBadge}</h3>`;

            dates.forEach(date => {
                const dateStr = date.toISOString().split('T')[0];
                const cellData = this.getCellData(dog.id, dateStr);
                const dayName = this.getDayName(date);
                const dateDisplay = this.formatDate(date);

                if (!cellData.isDogAvailable) {
                    html += `<div class="day-slot unavailable">
                        <div><strong>${dayName} ${dateDisplay}</strong></div>
                        <div style="color: #999;">Nicht verfügbar</div>
                    </div>`;
                } else if (!cellData.canAccess) {
                    const colorName = dogColor ? dogColor.name : 'unbekannte';
                    html += `<div class="day-slot unavailable">
                        <div><strong>${dayName} ${dateDisplay}</strong></div>
                        <div style="color: #999;">Farbe ${colorName} erforderlich</div>
                    </div>`;
                } else if (cellData.isBlocked) {
                    const blockMessage = cellData.isGloballyBlocked ? 'Datum gesperrt' : 'Hund gesperrt';
                    html += `<div class="day-slot unavailable">
                        <div><strong>${dayName} ${dateDisplay}</strong></div>
                        <div style="color: #999;">${blockMessage}</div>
                    </div>`;
                } else {
                    const bookedTimes = cellData.dogBookings.map(b => b.scheduled_time);
                    const hasBookings = bookedTimes.length > 0;

                    if (hasBookings) {
                        html += `<div class="day-slot available" data-action="quick-book" data-id="${dog.id}" data-value="${dateStr}">
                            <div><strong>${dayName} ${dateDisplay}</strong></div>
                            <div style="color: #856404;">Gebucht: ${bookedTimes.join(', ')}</div>
                            <div style="color: var(--color-primary, #82b965); font-size: 0.8rem;">+ Weitere Zeiten</div>
                        </div>`;
                    } else {
                        html += `<div class="day-slot available" data-action="quick-book" data-id="${dog.id}" data-value="${dateStr}">
                            <div><strong>${dayName} ${dateDisplay}</strong></div>
                            <div style="color: var(--color-primary, #82b965); font-weight: 600;">Alle Zeiten verfügbar</div>
                        </div>`;
                    }
                }
            });

            html += '</div>';
        });

        mobileContainer.innerHTML = html;
    }

    /**
     * Get filtered dogs based on current filter settings
     */
    getFilteredDogs() {
        const accessFilter = document.getElementById(this.options.accessFilterId)?.value || 'all';

        return this.allDogs.filter(dog => {
            if (!dog || !dog.id || !dog.name) return false;
            const visibleName = dog.name.replace(/[\s\u00A0\u2000-\u200B\u2028\u2029\u3000]/g, '');
            if (visibleName.length === 0) return false;

            // Admin mode shows all dogs regardless of color access
            if (this.options.isAdmin) return true;

            // Filter by access if "mine" is selected
            if (accessFilter === 'mine' && !this.canUserAccessDog(dog.color_id)) {
                return false;
            }
            return true;
        });
    }

    /**
     * Get next 14 days from today
     */
    getNext14Days() {
        const days = [];
        const today = new Date();
        for (let i = 0; i < 14; i++) {
            const date = new Date(today);
            date.setDate(today.getDate() + i);
            days.push(date);
        }
        return days;
    }

    /**
     * Get short day name in German
     */
    getDayName(date) {
        const days = ['So', 'Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa'];
        return days[date.getDay()];
    }

    /**
     * Format date as DD.MM
     */
    formatDate(date) {
        const day = String(date.getDate()).padStart(2, '0');
        const month = String(date.getMonth() + 1).padStart(2, '0');
        return `${day}.${month}`;
    }

    /**
     * Format date in German format (DD.MM.YYYY)
     */
    formatDateGerman(dateStr) {
        const date = new Date(dateStr);
        const day = String(date.getDate()).padStart(2, '0');
        const month = String(date.getMonth() + 1).padStart(2, '0');
        const year = date.getFullYear();
        return `${day}.${month}.${year}`;
    }

    /**
     * Get cell data for a specific dog and date
     */
    getCellData(dogId, date) {
        // Filter bookings for this dog and date
        const dogBookings = this.bookings.filter(b => {
            const bookingDate = b.date.split('T')[0];
            return b.dog_id === dogId && bookingDate === date && b.status !== 'cancelled';
        });

        // Check for blocked dates
        const isGloballyBlocked = this.blockedDates.some(bd => {
            const blockedDate = bd.date.split('T')[0];
            return blockedDate === date && bd.dog_id === null;
        });

        const isDogSpecificBlocked = this.blockedDates.some(bd => {
            const blockedDate = bd.date.split('T')[0];
            return blockedDate === date && bd.dog_id === dogId;
        });

        const isBlocked = isGloballyBlocked || isDogSpecificBlocked;

        const dog = this.allDogs.find(d => d.id === dogId);

        // Admin mode: always has access; User mode: check color
        const canAccess = this.options.isAdmin ? true : this.canUserAccessDog(dog?.color_id);

        return {
            dogBookings,
            isBlocked,
            isGloballyBlocked,
            isDogSpecificBlocked,
            isDogAvailable: dog?.is_available,
            canAccess
        };
    }

    /**
     * Render a single calendar cell
     */
    renderCell(dog, date, data) {
        const safeDogName = typeof sanitizeHTML === 'function' ? sanitizeHTML(dog.name) : dog.name;

        if (!data.isDogAvailable) {
            return `<div class="calendar-cell unavailable">
                <span style="font-size: 1.5rem;">❌</span>
                <div style="font-size: 0.7rem; margin-top: 4px;">Hund nicht verfügbar</div>
            </div>`;
        }

        // Check if user has the required color for this dog
        if (!data.canAccess) {
            const dogColor = this.getColorForDog(dog);
            const colorName = dogColor ? dogColor.name : 'unbekannte';
            return `<div class="calendar-cell unavailable">
                <span style="font-size: 1.5rem;">🔒</span>
                <div style="font-size: 0.7rem; margin-top: 4px;">Farbe ${colorName} erforderlich</div>
            </div>`;
        }

        if (data.isBlocked) {
            const blockMessage = data.isGloballyBlocked ? 'Datum gesperrt' : 'Hund gesperrt';
            return `<div class="calendar-cell unavailable">
                <span style="font-size: 1.5rem;">🚫</span>
                <div style="font-size: 0.7rem; margin-top: 4px;">${blockMessage}</div>
            </div>`;
        }

        const bookedTimes = data.dogBookings.map(b => b.scheduled_time);
        const hasBookings = bookedTimes.length > 0;

        if (hasBookings) {
            let content = '';
            bookedTimes.forEach(time => {
                content += `<div class="walk-type booked">⏰ ${time}</div>`;
            });
            content += '<div class="walk-type available">+ Weitere Zeiten</div>';

            return `<div class="calendar-cell booked" data-action="quick-book" data-id="${dog.id}" data-value="${date}" title="Klicken zum Buchen: ${safeDogName} am ${this.formatDateGerman(date)}">
                ${content}
            </div>`;
        }

        // Fully available
        return `<div class="calendar-cell available" data-action="quick-book" data-id="${dog.id}" data-value="${date}" title="Klicken zum Buchen: ${safeDogName} am ${this.formatDateGerman(date)}">
            <div class="walk-type available">✅ Verfügbar</div>
            <div style="font-size: 0.7rem; color: var(--text-gray);">Alle Zeiten frei</div>
        </div>`;
    }

    /**
     * Check if user can access a dog based on color
     */
    canUserAccessDog(dogColorId) {
        if (!dogColorId) return true; // Dogs without color are accessible to all
        return this.userColors.some(c => c.id === dogColorId);
    }

    /**
     * Get color object for a dog
     */
    getColorForDog(dog) {
        if (dog.color) return dog.color;
        return this.allColors.find(c => c.id === dog.color_id);
    }

    /**
     * Get pattern icon character
     */
    getPatternIcon(pattern) {
        const icons = {
            'circle': '●',
            'triangle': '▲',
            'square': '■',
            'diamond': '◆',
            'pentagon': '⬠',
            'hexagon': '⬡',
            'star': '★',
            'heart': '♥',
            'cross': '✚',
            'spade': '♠',
            'club': '♣',
            'moon': '☽',
            'sun': '☀',
            'ring': '○',
            'target': '◎'
        };
        return icons[pattern] || '●';
    }

    /**
     * Get color badge HTML
     */
    getColorBadgeHtml(color) {
        if (!color) return '';
        return `<span style="
            display: inline-flex;
            align-items: center;
            gap: 3px;
            padding: 2px 6px;
            border-radius: 8px;
            font-size: 0.7rem;
            font-weight: 500;
            background: ${color.hex_code}20;
            border: 1px solid ${color.hex_code};
            color: ${color.hex_code};
        ">${this.getPatternIcon(color.pattern_icon)} ${color.name}</span>`;
    }

    /**
     * Get calendar dog cell HTML (for the first column)
     */
    getCalendarDogCell(dog, dogColor) {
        const safeDogName = typeof sanitizeHTML === 'function' ? sanitizeHTML(dog.name) : dog.name;
        const colorBadge = dogColor ? this.getColorBadgeHtml(dogColor) : '';

        // Use dog photo helper if available
        const photoHtml = typeof getCalendarDogCell === 'function'
            ? getCalendarDogCell(dog)
            : `<div style="width: 40px; height: 40px; background: #ddd; border-radius: 50%; display: flex; align-items: center; justify-content: center;">🐕</div>`;

        return `
            <div class="calendar-dog-name-cell" style="display: flex; align-items: center; gap: 10px;">
                ${photoHtml}
                <div>
                    <div style="font-weight: 600;">${safeDogName}</div>
                    ${colorBadge}
                </div>
            </div>
        `;
    }

    /**
     * Handle quick book action
     */
    quickBook(dogId, date) {
        const dog = this.allDogs.find(d => d.id === dogId);
        if (!dog) return;

        // Check if user has access to this dog's color (only for non-admin)
        if (!this.options.isAdmin && !this.canUserAccessDog(dog.color_id)) {
            const dogColor = this.getColorForDog(dog);
            const colorName = dogColor ? dogColor.name : 'erforderliche';
            this.showAlert('error', `Du benötigst die Farbe "${colorName}" um diesen Hund zu buchen.`);
            return;
        }

        // Use callback if provided
        if (this.options.onQuickBook) {
            this.options.onQuickBook(dog, date);
            return;
        }

        // Default behavior: redirect to dogs page with pre-filled booking
        localStorage.setItem('pendingBooking', JSON.stringify({ dogId, date }));
        window.location.href = '/dogs.html';
    }

    /**
     * Switch view between grid and list
     */
    switchView(view) {
        this.currentView = view;
        const gridView = document.getElementById('grid-view');
        const listView = document.getElementById('list-view');
        const calendarGrid = document.querySelector('.calendar-wrapper');
        const calendarMobile = document.getElementById(this.options.mobileContainerId);

        if (view === 'grid') {
            if (gridView) gridView.classList.add('active');
            if (listView) listView.classList.remove('active');
            if (calendarGrid) calendarGrid.style.display = 'block';
            if (calendarMobile) calendarMobile.style.display = 'none';
        } else {
            if (gridView) gridView.classList.remove('active');
            if (listView) listView.classList.add('active');
            if (calendarGrid) calendarGrid.style.display = 'none';
            if (calendarMobile) calendarMobile.style.display = 'block';
            this.renderMobileView();
        }
    }

    /**
     * Show alert message
     */
    showAlert(type, message) {
        const container = document.getElementById(this.options.alertContainerId);
        if (container) {
            container.innerHTML = `<div class="alert alert-${type}">${message}</div>`;
            setTimeout(() => container.innerHTML = '', 5000);
        }
    }
}

// Export for global use
window.CalendarComponent = CalendarComponent;
