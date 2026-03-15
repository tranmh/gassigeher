/**
 * Calendar Component Tests
 *
 * Tests for the CalendarComponent class (week navigation, user names, highlights).
 *
 * @jest-environment jsdom
 */

// Mock sanitizeHTML globally before loading source
beforeAll(() => {
    // Set up minimal DOM
    document.body.innerHTML = `
        <div id="calendar-grid" class="calendar-grid"></div>
        <div id="calendar-mobile" class="calendar-mobile"></div>
        <div id="calendar-nav" class="calendar-nav"><span class="week-label"></span></div>
        <div id="calendar-print-header" class="calendar-print-header"><div class="print-date-range"></div></div>
        <div id="alert-container"></div>
        <select id="color-filter"><option value="">Alle</option></select>
        <select id="availability-filter"><option value="available" selected>Nur verfügbare</option></select>
        <select id="access-filter"><option value="all" selected>Alle</option></select>
    `;

    // Mock sanitizeHTML
    window.sanitizeHTML = (str) => str;

    // Mock api
    window.api = {
        getColors: jest.fn().mockResolvedValue({ colors: [] }),
        getDogs: jest.fn().mockResolvedValue([]),
        getBookings: jest.fn().mockResolvedValue([]),
        getBlockedDates: jest.fn().mockResolvedValue([])
    };

    // Mock getCalendarDogCell from dog-photo-helpers
    window.getCalendarDogCell = jest.fn().mockReturnValue('<div>🐕</div>');

    // Load the calendar component
    loadSourceFile('internal/static/frontend/js/calendar-component.js');
});

afterEach(() => {
    jest.restoreAllMocks();
});

describe('CalendarComponent', () => {
    let component;

    beforeEach(() => {
        component = new CalendarComponent({ isAdmin: false });
        component.currentUserId = 1;
    });

    describe('getWeekDates()', () => {
        test('returns 7 dates', () => {
            component.weekOffset = 0;
            const dates = component.getWeekDates();
            expect(dates).toHaveLength(7);
        });

        test('first date is Monday (day 1)', () => {
            component.weekOffset = 0;
            const dates = component.getWeekDates();
            expect(dates[0].getDay()).toBe(1);
        });

        test('last date is Sunday (day 0)', () => {
            component.weekOffset = 0;
            const dates = component.getWeekDates();
            expect(dates[6].getDay()).toBe(0);
        });

        test('weekOffset -1 returns previous week', () => {
            const comp0 = new CalendarComponent({});
            comp0.weekOffset = 0;
            const thisWeek = comp0.getWeekDates();

            const compPrev = new CalendarComponent({});
            compPrev.weekOffset = -1;
            const prevWeek = compPrev.getWeekDates();

            const diffDays = (thisWeek[0].getTime() - prevWeek[0].getTime()) / (24 * 60 * 60 * 1000);
            expect(diffDays).toBe(7);
        });

        test('weekOffset +1 returns next week', () => {
            const comp0 = new CalendarComponent({});
            comp0.weekOffset = 0;
            const thisWeek = comp0.getWeekDates();

            const compNext = new CalendarComponent({});
            compNext.weekOffset = 1;
            const nextWeek = compNext.getWeekDates();

            const diffDays = (nextWeek[0].getTime() - thisWeek[0].getTime()) / (24 * 60 * 60 * 1000);
            expect(diffDays).toBe(7);
        });

        test('dates are consecutive', () => {
            const dates = component.getWeekDates();
            for (let i = 1; i < dates.length; i++) {
                const diff = (dates[i].getTime() - dates[i - 1].getTime()) / (24 * 60 * 60 * 1000);
                expect(diff).toBe(1);
            }
        });
    });

    describe('getISOWeekNumber()', () => {
        test('returns correct week for known date', () => {
            // Jan 1, 2026 is a Thursday, so ISO week 1
            expect(component.getISOWeekNumber(new Date(2026, 0, 1))).toBe(1);
        });

        test('returns week 53 for Dec 31 2026 (which is Thursday)', () => {
            // Dec 31, 2026 is a Thursday, ISO week 53 of 2026
            expect(component.getISOWeekNumber(new Date(2026, 11, 31))).toBe(53);
        });
    });

    describe('getWeekLabel()', () => {
        test('format matches KW pattern', () => {
            component.weekOffset = 0;
            const label = component.getWeekLabel();
            expect(label).toMatch(/^KW \d+ \| \d{2}\.\d{2}\. - \d{2}\.\d{2}\.\d{4}$/);
        });
    });

    describe('navigateWeek()', () => {
        test('increments weekOffset on next', () => {
            component.weekOffset = 0;
            // Mock loadCalendar to avoid API calls
            component.loadCalendar = jest.fn();
            component.navigateWeek(1);
            expect(component.weekOffset).toBe(1);
        });

        test('decrements weekOffset on prev', () => {
            component.weekOffset = 0;
            component.loadCalendar = jest.fn();
            component.navigateWeek(-1);
            expect(component.weekOffset).toBe(-1);
        });
    });

    describe('goToToday()', () => {
        test('resets weekOffset to 0', () => {
            component.weekOffset = 5;
            component.loadCalendar = jest.fn();
            component.goToToday();
            expect(component.weekOffset).toBe(0);
        });
    });

    describe('goToDate()', () => {
        test('calculates correct weekOffset for a future date', () => {
            component.loadCalendar = jest.fn();
            // Go to a date 14 days from now - should be weekOffset 2
            const future = new Date();
            future.setDate(future.getDate() + 14);
            const dateStr = future.toISOString().split('T')[0];
            component.goToDate(dateStr);
            expect(component.weekOffset).toBe(2);
        });

        test('handles invalid date gracefully', () => {
            component.loadCalendar = jest.fn();
            component.weekOffset = 0;
            component.goToDate('not-a-date');
            // Should not change offset
            expect(component.weekOffset).toBe(0);
        });

        test('handles empty string', () => {
            component.loadCalendar = jest.fn();
            component.weekOffset = 3;
            component.goToDate('');
            // Should not change offset
            expect(component.weekOffset).toBe(3);
        });
    });

    describe('_formatBookerName()', () => {
        test('formats as "Max M."', () => {
            const booking = { user: { first_name: 'Max', last_name: 'Mustermann' } };
            expect(component._formatBookerName(booking)).toBe('Max M.');
        });

        test('handles missing last name', () => {
            const booking = { user: { first_name: 'Anna', last_name: '' } };
            expect(component._formatBookerName(booking)).toBe('Anna');
        });

        test('handles missing user gracefully', () => {
            expect(component._formatBookerName({ user: null })).toBe('');
            expect(component._formatBookerName({})).toBe('');
        });

        test('handles empty names', () => {
            const booking = { user: { first_name: '', last_name: '' } };
            expect(component._formatBookerName(booking)).toBe('');
        });
    });

    describe('renderCell() with user names', () => {
        test('shows booker name in booked cell', () => {
            const dog = { id: 1, name: 'Rex', is_available: true, color_id: null };
            const data = {
                dogBookings: [{ scheduled_time: '09:00', user_id: 2, status: 'scheduled', user: { first_name: 'Max', last_name: 'Mustermann' } }],
                isBlocked: false, isDogAvailable: true, canAccess: true
            };
            const html = component.renderCell(dog, '2026-03-16', data, false, false);
            expect(html).toContain('Max M.');
            expect(html).toContain('09:00');
        });

        test('shows "Du" badge for current user booking', () => {
            component.currentUserId = 5;
            const data = {
                dogBookings: [{ scheduled_time: '09:00', user_id: 5, status: 'scheduled', user: { first_name: 'Anna', last_name: 'Schmidt' } }],
                isBlocked: false, isDogAvailable: true, canAccess: true
            };
            const html = component.renderCell({ id: 1, name: 'Rex', is_available: true }, '2026-03-16', data, false, false);
            expect(html).toContain('my-booking');
            expect(html).toContain('Du');
        });

        test('handles missing user gracefully', () => {
            const data = {
                dogBookings: [{ scheduled_time: '09:00', user_id: 99, status: 'scheduled', user: null }],
                isBlocked: false, isDogAvailable: true, canAccess: true
            };
            const html = component.renderCell({ id: 1, name: 'Rex', is_available: true }, '2026-03-16', data, false, false);
            expect(html).toContain('09:00');
            // Should not crash
        });

        test('adds weekend class for weekend dates', () => {
            const data = {
                dogBookings: [],
                isBlocked: false, isDogAvailable: true, canAccess: true
            };
            const html = component.renderCell({ id: 1, name: 'Rex', is_available: true }, '2026-03-16', data, false, true);
            expect(html).toContain('weekend');
        });
    });

    describe('past dates handling', () => {
        test('past cells do not have quick-book action', () => {
            const data = {
                dogBookings: [],
                isBlocked: false, isDogAvailable: true, canAccess: true
            };
            const html = component.renderCell({ id: 1, name: 'Rex', is_available: true }, '2020-01-01', data, true, false);
            expect(html).not.toContain('data-action="quick-book"');
            expect(html).toContain('past');
        });

        test('past booked cells show completed checkmark', () => {
            const data = {
                dogBookings: [{ scheduled_time: '09:00', user_id: 2, status: 'completed', user: { first_name: 'Max', last_name: 'M' } }],
                isBlocked: false, isDogAvailable: true, canAccess: true
            };
            const html = component.renderCell({ id: 1, name: 'Rex', is_available: true }, '2020-01-01', data, true, false);
            expect(html).toContain('booking-completed');
            expect(html).not.toContain('data-action="quick-book"');
        });
    });

    describe('getDayName()', () => {
        test('returns German day abbreviations', () => {
            // Monday
            expect(component.getDayName(new Date(2026, 2, 16))).toBe('Mo');
            // Sunday
            expect(component.getDayName(new Date(2026, 2, 22))).toBe('So');
            // Saturday
            expect(component.getDayName(new Date(2026, 2, 21))).toBe('Sa');
        });
    });

    describe('formatDate()', () => {
        test('formats as DD.MM', () => {
            expect(component.formatDate(new Date(2026, 2, 5))).toBe('05.03');
            expect(component.formatDate(new Date(2026, 11, 25))).toBe('25.12');
        });
    });

    describe('_formatISO()', () => {
        test('returns YYYY-MM-DD format', () => {
            expect(component._formatISO(new Date(2026, 2, 16))).toBe('2026-03-16');
            expect(component._formatISO(new Date(2026, 0, 1))).toBe('2026-01-01');
        });
    });

    describe('setCurrentUserId()', () => {
        test('sets the current user ID', () => {
            component.setCurrentUserId(42);
            expect(component.currentUserId).toBe(42);
        });
    });

    describe('setUserColors()', () => {
        test('sets user colors array', () => {
            component.setUserColors([{ id: 1 }, { id: 2 }]);
            expect(component.userColors).toHaveLength(2);
        });

        test('handles null input', () => {
            component.setUserColors(null);
            expect(component.userColors).toEqual([]);
        });
    });
});
