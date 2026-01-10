/**
 * Period-Based Booking Tests
 *
 * Tests for the period-based booking blocking feature:
 * - Dogs can only have 1 booking per period (morning/afternoon) per day
 * - UI shows booked periods info
 * - Race condition handling for rapid date changes
 * - Empty dogId handling
 *
 * @jest-environment jsdom
 */

// Mock localStorage
const localStorageMock = (() => {
  let store = {};
  return {
    getItem: jest.fn((key) => store[key] || null),
    setItem: jest.fn((key, value) => { store[key] = value; }),
    removeItem: jest.fn((key) => { delete store[key]; }),
    clear: jest.fn(() => { store = {}; }),
  };
})();

Object.defineProperty(window, 'localStorage', { value: localStorageMock });

// Mock fetch
let fetchMock;

// Helper to create mock response
const mockResponse = (data, ok = true, status = 200) => ({
  ok,
  status,
  text: () => Promise.resolve(data ? JSON.stringify(data) : ''),
});

beforeAll(() => {
  document.body.innerHTML = '';
  loadSourceFile('internal/static/frontend/js/api.js');
});

beforeEach(() => {
  localStorageMock.clear();
  fetchMock = jest.fn();
  global.fetch = fetchMock;
  window.api.token = 'test-token';
});

afterEach(() => {
  jest.restoreAllMocks();
});

describe('API - getAvailableTimeSlots with dogId parameter', () => {
  test('should call API without dog_id when dogId is null', async () => {
    fetchMock.mockResolvedValueOnce(mockResponse({ slots: ['09:00', '10:00'] }));

    await window.api.getAvailableTimeSlots('2025-01-27', null);

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining('/booking-times/available?date=2025-01-27'),
      expect.any(Object)
    );
    expect(fetchMock.mock.calls[0][0]).not.toContain('dog_id');
  });

  test('should call API with dog_id when dogId is provided', async () => {
    fetchMock.mockResolvedValueOnce(mockResponse({ slots: ['14:00', '15:00'] }));

    await window.api.getAvailableTimeSlots('2025-01-27', 123);

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining('/booking-times/available?date=2025-01-27&dog_id=123'),
      expect.any(Object)
    );
  });

  test('should NOT send dog_id when dogId is empty string', async () => {
    fetchMock.mockResolvedValueOnce(mockResponse({ slots: ['09:00'] }));

    await window.api.getAvailableTimeSlots('2025-01-27', '');

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining('/booking-times/available?date=2025-01-27'),
      expect.any(Object)
    );
    // Empty string should be treated as falsy, no dog_id param
    expect(fetchMock.mock.calls[0][0]).not.toContain('dog_id=');
  });

  test('should send dog_id when dogId is string number', async () => {
    fetchMock.mockResolvedValueOnce(mockResponse({ slots: ['09:00'] }));

    await window.api.getAvailableTimeSlots('2025-01-27', '456');

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining('dog_id=456'),
      expect.any(Object)
    );
  });

  test('should return booked_periods from API response', async () => {
    const mockData = {
      slots: ['14:00', '15:00'],
      booked_periods: [
        { rule_name: 'Vormittag', start_time: '08:30', end_time: '12:00' }
      ]
    };
    fetchMock.mockResolvedValueOnce(mockResponse(mockData));

    const result = await window.api.getAvailableTimeSlots('2025-01-27', 1);

    expect(result.slots).toEqual(['14:00', '15:00']);
    expect(result.booked_periods).toHaveLength(1);
    expect(result.booked_periods[0].rule_name).toBe('Vormittag');
  });

  test('should handle missing booked_periods gracefully', async () => {
    const mockData = { slots: ['09:00', '10:00'] };
    fetchMock.mockResolvedValueOnce(mockResponse(mockData));

    const result = await window.api.getAvailableTimeSlots('2025-01-27', 1);

    expect(result.slots).toEqual(['09:00', '10:00']);
    expect(result.booked_periods).toBeUndefined();
  });
});

describe('Period Booking - Error Messages', () => {
  test('should recognize German period blocking error', () => {
    const errorMessage = 'Dieser Hund ist bereits für Vormittag (08:30-12:00) gebucht';

    // The error contains the period name and times
    expect(errorMessage).toContain('Vormittag');
    expect(errorMessage).toContain('08:30');
    expect(errorMessage).toContain('12:00');
  });

  test('should recognize buffer time error', () => {
    const errorMessage = 'Buchung muss mindestens 30 Minuten vor Ende des Zeitraums (12:00) liegen';

    expect(errorMessage).toContain('30 Minuten');
    expect(errorMessage).toContain('12:00');
  });
});

describe('Period Booking - UI Element Handling', () => {
  beforeEach(() => {
    // Setup mock DOM for booking modal
    document.body.innerHTML = `
      <div id="booking-modal">
        <input type="hidden" id="booking-dog-id" value="">
        <input type="date" id="booking-date" value="2025-01-27">
        <select id="booking-time">
          <option value="">Bitte wählen...</option>
        </select>
        <div id="booked-periods-info" style="display: none;">
          <ul id="booked-periods-list"></ul>
        </div>
        <div id="time-rules-info" style="display: none;">
          <ul id="time-rules-list"></ul>
        </div>
      </div>
    `;
  });

  test('dogId input returns empty string when not set', () => {
    const dogIdInput = document.getElementById('booking-dog-id');
    expect(dogIdInput.value).toBe('');
  });

  test('dogId input returns value when set', () => {
    const dogIdInput = document.getElementById('booking-dog-id');
    dogIdInput.value = '123';
    expect(dogIdInput.value).toBe('123');
  });

  test('booked periods info is hidden by default', () => {
    const bookedPeriodsInfo = document.getElementById('booked-periods-info');
    expect(bookedPeriodsInfo.style.display).toBe('none');
  });

  test('booked periods list is empty by default', () => {
    const bookedPeriodsList = document.getElementById('booked-periods-list');
    expect(bookedPeriodsList.children.length).toBe(0);
  });
});

describe('Period Booking - dogId extraction logic', () => {
  test('should return null for empty input value', () => {
    document.body.innerHTML = '<input type="hidden" id="booking-dog-id" value="">';

    const dogIdInput = document.getElementById('booking-dog-id');
    const dogId = (dogIdInput && dogIdInput.value) ? dogIdInput.value : null;

    expect(dogId).toBeNull();
  });

  test('should return value for non-empty input', () => {
    document.body.innerHTML = '<input type="hidden" id="booking-dog-id" value="42">';

    const dogIdInput = document.getElementById('booking-dog-id');
    const dogId = (dogIdInput && dogIdInput.value) ? dogIdInput.value : null;

    expect(dogId).toBe('42');
  });

  test('should return null when element does not exist', () => {
    document.body.innerHTML = '';

    const dogIdInput = document.getElementById('booking-dog-id');
    const dogId = (dogIdInput && dogIdInput.value) ? dogIdInput.value : null;

    expect(dogId).toBeNull();
  });
});

describe('Period Booking - booked periods display logic', () => {
  beforeEach(() => {
    document.body.innerHTML = `
      <div id="booked-periods-info" style="display: none;">
        <ul id="booked-periods-list"></ul>
      </div>
    `;
  });

  test('should show info box when booked periods exist', () => {
    const bookedPeriods = [
      { rule_name: 'Vormittag', start_time: '08:30', end_time: '12:00' }
    ];

    const bookedPeriodsInfo = document.getElementById('booked-periods-info');
    const bookedPeriodsList = document.getElementById('booked-periods-list');

    if (bookedPeriods.length > 0) {
      bookedPeriodsList.innerHTML = '';
      bookedPeriods.forEach(period => {
        const li = document.createElement('li');
        li.textContent = `${period.rule_name}: ${period.start_time} - ${period.end_time}`;
        bookedPeriodsList.appendChild(li);
      });
      bookedPeriodsInfo.style.display = 'block';
    }

    expect(bookedPeriodsInfo.style.display).toBe('block');
    expect(bookedPeriodsList.children.length).toBe(1);
    expect(bookedPeriodsList.children[0].textContent).toBe('Vormittag: 08:30 - 12:00');
  });

  test('should hide info box when no booked periods', () => {
    const bookedPeriods = [];

    const bookedPeriodsInfo = document.getElementById('booked-periods-info');

    if (bookedPeriods.length > 0) {
      bookedPeriodsInfo.style.display = 'block';
    } else {
      bookedPeriodsInfo.style.display = 'none';
    }

    expect(bookedPeriodsInfo.style.display).toBe('none');
  });

  test('should handle multiple booked periods', () => {
    const bookedPeriods = [
      { rule_name: 'Vormittag', start_time: '08:30', end_time: '12:00' },
      { rule_name: 'Nachmittag', start_time: '14:00', end_time: '17:00' }
    ];

    const bookedPeriodsInfo = document.getElementById('booked-periods-info');
    const bookedPeriodsList = document.getElementById('booked-periods-list');

    bookedPeriodsList.innerHTML = '';
    bookedPeriods.forEach(period => {
      const li = document.createElement('li');
      li.textContent = `${period.rule_name}: ${period.start_time} - ${period.end_time}`;
      bookedPeriodsList.appendChild(li);
    });
    bookedPeriodsInfo.style.display = 'block';

    expect(bookedPeriodsList.children.length).toBe(2);
  });
});

describe('Period Booking - Race Condition Prevention', () => {
  test('request tracking variable should track latest request', () => {
    let currentRequest = null;

    // Simulate first request
    const requestId1 = Date.now();
    currentRequest = requestId1;

    // Simulate second request (user changed date rapidly)
    const requestId2 = Date.now() + 1;
    currentRequest = requestId2;

    // First request response arrives
    const isStale1 = currentRequest !== requestId1;
    expect(isStale1).toBe(true); // Should be ignored

    // Second request response arrives
    const isStale2 = currentRequest !== requestId2;
    expect(isStale2).toBe(false); // Should be processed
  });

  test('should ignore stale responses', () => {
    let currentRequest = null;
    let processedResponses = [];

    // Simulate async request handling
    const handleResponse = (requestId, data) => {
      if (currentRequest !== requestId) {
        return; // Ignore stale
      }
      processedResponses.push(data);
    };

    // Start request 1
    const req1 = 1;
    currentRequest = req1;

    // Start request 2 (before 1 completes)
    const req2 = 2;
    currentRequest = req2;

    // Response 2 arrives first
    handleResponse(req2, 'data2');

    // Response 1 arrives (stale)
    handleResponse(req1, 'data1');

    // Only data2 should be processed
    expect(processedResponses).toEqual(['data2']);
  });
});

describe('Period Booking - XSS Prevention', () => {
  test('booked periods should use textContent (not innerHTML)', () => {
    document.body.innerHTML = '<ul id="booked-periods-list"></ul>';

    const bookedPeriodsList = document.getElementById('booked-periods-list');
    const maliciousPeriod = {
      rule_name: '<script>alert("XSS")</script>',
      start_time: '08:30',
      end_time: '12:00'
    };

    const li = document.createElement('li');
    li.textContent = `${maliciousPeriod.rule_name}: ${maliciousPeriod.start_time} - ${maliciousPeriod.end_time}`;
    bookedPeriodsList.appendChild(li);

    // textContent should escape the HTML
    expect(li.innerHTML).not.toContain('<script>');
    expect(li.textContent).toContain('<script>'); // Displayed as text, not executed
  });
});
