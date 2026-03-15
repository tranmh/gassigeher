/**
 * Calendar Component - Shared calendar logic for user and admin pages
 * Displays a weekly availability calendar for dogs with booking capabilities
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
            navContainerId: 'calendar-nav',
            printHeaderId: 'calendar-print-header',
            isAdmin: false,
            onQuickBook: null,
            ...options
        };

        this.allDogs = [];
        this.bookings = [];
        this.blockedDates = [];
        this.allColors = [];
        this.userColors = [];
        this.currentView = 'grid';
        this.weekOffset = 0; // 0 = current week, -1 = last week, +1 = next week
        this.currentUserId = null;

        this._boundKeyHandler = this._handleKeydown.bind(this);
    }

    /**
     * Initialize the calendar component
     */
    async init() {
        await this.loadColors();
        await this.loadCalendar();
        this._bindKeyboard();
    }

    /**
     * Destroy event listeners
     */
    destroy() {
        document.removeEventListener('keydown', this._boundKeyHandler);
    }

    /**
     * Set user colors (for access filtering)
     */
    setUserColors(colors) {
        this.userColors = colors || [];
    }

    /**
     * Set current user ID (for "my booking" highlights)
     */
    setCurrentUserId(userId) {
        this.currentUserId = userId;
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

            // Fetch bookings for the displayed week
            const dates = this.getWeekDates();
            const startDate = this._formatISO(dates[0]);
            const endDate = this._formatISO(dates[6]);

            this.bookings = await api.getBookings({
                date_from: startDate,
                date_to: endDate,
                calendar_view: 'true',
                include_user: 'true'
            });

            // Fetch blocked dates
            this.blockedDates = await api.getBlockedDates();

            this.updateNavigation();
            this.renderCalendar();
            this.renderMobileView();
        } catch (error) {
            console.error('Failed to load calendar:', error);
            this.showAlert('error', 'Fehler beim Laden der Daten: ' + (error.message || 'Unbekannter Fehler'));
        }
    }

    // ========================
    // Week Navigation
    // ========================

    /**
     * Get the 7 dates (Mon-Sun) for the current weekOffset
     */
    getWeekDates() {
        const today = new Date();
        const dayOfWeek = today.getDay(); // 0=Sun, 1=Mon, ...
        const mondayOffset = dayOfWeek === 0 ? -6 : 1 - dayOfWeek;
        const monday = new Date(today);
        monday.setDate(today.getDate() + mondayOffset + (this.weekOffset * 7));
        // Reset time
        monday.setHours(0, 0, 0, 0);

        const days = [];
        for (let i = 0; i < 7; i++) {
            const d = new Date(monday);
            d.setDate(monday.getDate() + i);
            days.push(d);
        }
        return days;
    }

    /**
     * Navigate to previous/next week
     */
    navigateWeek(direction) {
        this.weekOffset += direction;
        this.loadCalendar();
    }

    /**
     * Go to the current week
     */
    goToToday() {
        this.weekOffset = 0;
        this.loadCalendar();
    }

    /**
     * Jump to a specific date's week
     */
    goToDate(dateStr) {
        if (!dateStr) return;
        const target = new Date(dateStr + 'T00:00:00');
        if (isNaN(target.getTime())) return;

        const today = new Date();
        today.setHours(0, 0, 0, 0);

        // Find Monday of today's week
        const todayDow = today.getDay();
        const todayMondayOffset = todayDow === 0 ? -6 : 1 - todayDow;
        const todayMonday = new Date(today);
        todayMonday.setDate(today.getDate() + todayMondayOffset);

        // Find Monday of target week
        const targetDow = target.getDay();
        const targetMondayOffset = targetDow === 0 ? -6 : 1 - targetDow;
        const targetMonday = new Date(target);
        targetMonday.setDate(target.getDate() + targetMondayOffset);

        // Calculate week offset
        const diffMs = targetMonday.getTime() - todayMonday.getTime();
        this.weekOffset = Math.round(diffMs / (7 * 24 * 60 * 60 * 1000));
        this.loadCalendar();
    }

    /**
     * Get ISO week number
     */
    getISOWeekNumber(date) {
        const d = new Date(Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()));
        const dayNum = d.getUTCDay() || 7;
        d.setUTCDate(d.getUTCDate() + 4 - dayNum);
        const yearStart = new Date(Date.UTC(d.getUTCFullYear(), 0, 1));
        return Math.ceil((((d - yearStart) / 86400000) + 1) / 7);
    }

    /**
     * Get week label string e.g. "KW 12 | 16.03. - 22.03.2026"
     */
    getWeekLabel() {
        const dates = this.getWeekDates();
        const monday = dates[0];
        const sunday = dates[6];
        const kw = this.getISOWeekNumber(monday);
        const monStr = this.formatDate(monday);
        const sunDay = String(sunday.getDate()).padStart(2, '0');
        const sunMonth = String(sunday.getMonth() + 1).padStart(2, '0');
        const sunYear = sunday.getFullYear();
        return `KW ${kw} | ${monStr}. - ${sunDay}.${sunMonth}.${sunYear}`;
    }

    /**
     * Update the navigation bar display
     */
    updateNavigation() {
        const navContainer = document.getElementById(this.options.navContainerId);
        if (!navContainer) return;

        const weekLabel = navContainer.querySelector('.week-label');
        if (weekLabel) {
            weekLabel.textContent = this.getWeekLabel();
        }

        // Update print header
        const printHeader = document.getElementById(this.options.printHeaderId);
        if (printHeader) {
            const dateRange = printHeader.querySelector('.print-date-range');
            if (dateRange) {
                dateRange.textContent = this.getWeekLabel();
            }
        }
    }

    /**
     * Print the calendar
     */
    printCalendar() {
        window.print();
    }

    // ========================
    // Keyboard Navigation
    // ========================

    _bindKeyboard() {
        document.addEventListener('keydown', this._boundKeyHandler);
    }

    _handleKeydown(e) {
        // Only handle arrow keys when not in an input
        if (e.target.tagName === 'INPUT' || e.target.tagName === 'SELECT' || e.target.tagName === 'TEXTAREA') return;

        if (e.key === 'ArrowLeft') {
            e.preventDefault();
            this.navigateWeek(-1);
        } else if (e.key === 'ArrowRight') {
            e.preventDefault();
            this.navigateWeek(1);
        }
    }

    // ========================
    // Rendering
    // ========================

    /**
     * Render the desktop calendar grid
     */
    renderCalendar() {
        const grid = document.getElementById(this.options.gridContainerId);
        if (!grid) return;

        const dates = this.getWeekDates();
        const today = new Date();
        today.setHours(0, 0, 0, 0);

        // Build grid HTML
        let html = '';

        // Header row
        html += '<div class="calendar-header dog-name"><strong>Hunde</strong></div>';
        dates.forEach(date => {
            const dayName = this.getDayName(date);
            const dateStr = this.formatDate(date);
            const isToday = date.toDateString() === today.toDateString();
            const isWeekend = date.getDay() === 0 || date.getDay() === 6;
            const classes = ['calendar-header'];
            if (isWeekend) classes.push('weekend');
            const todayStyle = isToday ? ' style="border: 3px solid var(--accent-orange); box-shadow: 0 0 12px rgba(255, 140, 66, 0.4);"' : '';
            html += `<div class="${classes.join(' ')}"${todayStyle}>
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
                const dateStr = this._formatISO(date);
                const cellData = this.getCellData(dog.id, dateStr);
                const isPast = date < today;
                const isWeekend = date.getDay() === 0 || date.getDay() === 6;
                html += this.renderCell(dog, dateStr, cellData, isPast, isWeekend);
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

        const dates = this.getWeekDates();
        const today = new Date();
        today.setHours(0, 0, 0, 0);
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
                const dateStr = this._formatISO(date);
                const cellData = this.getCellData(dog.id, dateStr);
                const dayName = this.getDayName(date);
                const dateDisplay = this.formatDate(date);
                const isPast = date < today;

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
                    const hasBookings = cellData.dogBookings.length > 0;
                    const hasMyBooking = cellData.dogBookings.some(b => b.user_id === this.currentUserId);
                    const myBookingClass = hasMyBooking ? ' my-booking' : '';

                    if (isPast) {
                        // Past dates - show status but no booking action
                        if (hasBookings) {
                            const bookingLines = cellData.dogBookings.map(b => {
                                const name = this._formatBookerName(b);
                                const isMe = b.user_id === this.currentUserId;
                                const badge = isMe ? ' <span class="my-booking-badge">Du</span>' : '';
                                const status = b.status === 'completed' ? ' <span class="booking-completed">✓</span>' : '';
                                return `${b.scheduled_time} ${name}${badge}${status}`;
                            }).join(', ');
                            html += `<div class="day-slot unavailable${myBookingClass}">
                                <div><strong>${dayName} ${dateDisplay}</strong></div>
                                <div style="color: #856404;">${bookingLines}</div>
                            </div>`;
                        } else {
                            html += `<div class="day-slot unavailable">
                                <div><strong>${dayName} ${dateDisplay}</strong></div>
                                <div style="color: #999;">-</div>
                            </div>`;
                        }
                    } else if (hasBookings) {
                        const bookingLines = cellData.dogBookings.map(b => {
                            const name = this._formatBookerName(b);
                            const isMe = b.user_id === this.currentUserId;
                            const badge = isMe ? ' <span class="my-booking-badge">Du</span>' : '';
                            return `${b.scheduled_time} ${name}${badge}`;
                        }).join(', ');
                        html += `<div class="day-slot available${myBookingClass}" data-action="quick-book" data-id="${dog.id}" data-value="${dateStr}">
                            <div><strong>${dayName} ${dateDisplay}</strong></div>
                            <div style="color: #856404;">${bookingLines}</div>
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
        const date = new Date(dateStr + 'T00:00:00');
        const day = String(date.getDate()).padStart(2, '0');
        const month = String(date.getMonth() + 1).padStart(2, '0');
        const year = date.getFullYear();
        return `${day}.${month}.${year}`;
    }

    /**
     * Format ISO date string YYYY-MM-DD from Date object
     */
    _formatISO(date) {
        const y = date.getFullYear();
        const m = String(date.getMonth() + 1).padStart(2, '0');
        const d = String(date.getDate()).padStart(2, '0');
        return `${y}-${m}-${d}`;
    }

    /**
     * Format booker name as "Max M."
     */
    _formatBookerName(booking) {
        if (!booking.user) return '';
        const first = booking.user.first_name || '';
        const last = booking.user.last_name || '';
        if (!first && !last) return '';
        const lastInitial = last ? ` ${last.charAt(0)}.` : '';
        const safeName = typeof sanitizeHTML === 'function' ? sanitizeHTML(first + lastInitial) : first + lastInitial;
        return safeName;
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
    renderCell(dog, date, data, isPast, isWeekend) {
        const safeDogName = typeof sanitizeHTML === 'function' ? sanitizeHTML(dog.name) : dog.name;
        const weekendClass = isWeekend ? ' weekend' : '';
        const pastClass = isPast ? ' past' : '';

        if (!data.isDogAvailable) {
            return `<div class="calendar-cell unavailable${weekendClass}">
                <span style="font-size: 1.5rem;">❌</span>
                <div style="font-size: 0.7rem; margin-top: 4px;">Hund nicht verfügbar</div>
            </div>`;
        }

        // Check if user has the required color for this dog
        if (!data.canAccess) {
            const dogColor = this.getColorForDog(dog);
            const colorName = dogColor ? dogColor.name : 'unbekannte';
            return `<div class="calendar-cell unavailable${weekendClass}">
                <span style="font-size: 1.5rem;">🔒</span>
                <div style="font-size: 0.7rem; margin-top: 4px;">Farbe ${colorName} erforderlich</div>
            </div>`;
        }

        if (data.isBlocked) {
            const blockMessage = data.isGloballyBlocked ? 'Datum gesperrt' : 'Hund gesperrt';
            return `<div class="calendar-cell unavailable${weekendClass}">
                <span style="font-size: 1.5rem;">🚫</span>
                <div style="font-size: 0.7rem; margin-top: 4px;">${blockMessage}</div>
            </div>`;
        }

        const bookedTimes = data.dogBookings;
        const hasBookings = bookedTimes.length > 0;
        const hasMyBooking = bookedTimes.some(b => b.user_id === this.currentUserId);

        if (isPast) {
            // Past cells: show info but no booking action
            if (hasBookings) {
                let content = '';
                bookedTimes.forEach(b => {
                    const isMe = b.user_id === this.currentUserId;
                    const myClass = isMe ? ' my-booking' : '';
                    const badge = isMe ? '<span class="my-booking-badge">Du</span>' : '';
                    const statusIcon = b.status === 'completed' ? ' <span class="booking-completed">✓</span>' : '';
                    const bookerName = this._formatBookerName(b);
                    content += `<div class="walk-type booked${myClass}">⏰ ${b.scheduled_time}${statusIcon}${badge}</div>`;
                    if (bookerName) {
                        content += `<div class="booker-name">${bookerName}</div>`;
                    }
                });
                return `<div class="calendar-cell booked${pastClass}${weekendClass}">
                    ${content}
                </div>`;
            }
            return `<div class="calendar-cell${pastClass}${weekendClass}" style="color: #ccc;">
                <div style="font-size: 0.7rem;">-</div>
            </div>`;
        }

        if (hasBookings) {
            let content = '';
            bookedTimes.forEach(b => {
                const isMe = b.user_id === this.currentUserId;
                const myClass = isMe ? ' my-booking' : '';
                const badge = isMe ? '<span class="my-booking-badge">Du</span>' : '';
                const bookerName = this._formatBookerName(b);
                content += `<div class="walk-type booked${myClass}">⏰ ${b.scheduled_time}${badge}</div>`;
                if (bookerName) {
                    content += `<div class="booker-name">${bookerName}</div>`;
                }
            });
            content += '<div class="walk-type available">+ Weitere Zeiten</div>';

            return `<div class="calendar-cell booked${weekendClass}" data-action="quick-book" data-id="${dog.id}" data-value="${date}" title="Klicken zum Buchen: ${safeDogName} am ${this.formatDateGerman(date)}">
                ${content}
            </div>`;
        }

        // Fully available
        return `<div class="calendar-cell available${weekendClass}" data-action="quick-book" data-id="${dog.id}" data-value="${date}" title="Klicken zum Buchen: ${safeDogName} am ${this.formatDateGerman(date)}">
            <div class="walk-type available">✅ Verfügbar</div>
            <div style="font-size: 0.7rem; color: var(--text-gray);">Alle Zeiten frei</div>
        </div>`;
    }

    // ========================
    // Access & Color Helpers
    // ========================

    canUserAccessDog(dogColorId) {
        if (!dogColorId) return true;
        return this.userColors.some(c => c.id === dogColorId);
    }

    getColorForDog(dog) {
        if (dog.color) return dog.color;
        return this.allColors.find(c => c.id === dog.color_id);
    }

    getPatternIcon(pattern) {
        const icons = {
            'circle': '●', 'triangle': '▲', 'square': '■', 'diamond': '◆',
            'pentagon': '⬠', 'hexagon': '⬡', 'star': '★', 'heart': '♥',
            'cross': '✚', 'spade': '♠', 'club': '♣', 'moon': '☽',
            'sun': '☀', 'ring': '○', 'target': '◎'
        };
        return icons[pattern] || '●';
    }

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

    getCalendarDogCell(dog, dogColor) {
        const safeDogName = typeof sanitizeHTML === 'function' ? sanitizeHTML(dog.name) : dog.name;
        const colorBadge = dogColor ? this.getColorBadgeHtml(dogColor) : '';

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

    // ========================
    // Actions
    // ========================

    quickBook(dogId, date) {
        const dog = this.allDogs.find(d => d.id === dogId);
        if (!dog) return;

        if (!this.options.isAdmin && !this.canUserAccessDog(dog.color_id)) {
            const dogColor = this.getColorForDog(dog);
            const colorName = dogColor ? dogColor.name : 'erforderliche';
            this.showAlert('error', `Du benötigst die Farbe "${colorName}" um diesen Hund zu buchen.`);
            return;
        }

        if (this.options.onQuickBook) {
            this.options.onQuickBook(dog, date);
            return;
        }

        localStorage.setItem('pendingBooking', JSON.stringify({ dogId, date }));
        window.location.href = '/dogs.html';
    }

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
