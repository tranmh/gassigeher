/**
 * Dog Care Info Form Tests
 *
 * Tests for the dog care information form functionality in admin-dogs.html
 * Verifies that care info fields are correctly collected, populated, and displayed.
 *
 * @jest-environment jsdom
 */

describe('Dog Care Info Form', () => {
  let document;

  beforeEach(() => {
    // Set up DOM structure matching admin-dogs.html
    document = global.document;
    document.body.innerHTML = `
      <form id="dog-form">
        <input type="hidden" id="dog-id">
        <input type="text" id="dog-name">
        <input type="text" id="dog-breed">
        <select id="dog-size">
          <option value="small">Klein</option>
          <option value="medium">Mittel</option>
          <option value="large">Groß</option>
        </select>
        <input type="number" id="dog-age">
        <select id="dog-color">
          <option value="1">Grün</option>
          <option value="2">Blau</option>
        </select>
        <input type="url" id="dog-external-link">

        <!-- Care Info Fields -->
        <textarea id="dog-special-needs"></textarea>
        <input type="text" id="dog-pickup-location">
        <textarea id="dog-walk-route"></textarea>
        <input type="number" id="dog-walk-duration">
        <textarea id="dog-special-instructions"></textarea>
        <input type="time" id="dog-default-morning-time">
        <input type="time" id="dog-default-evening-time">
      </form>
    `;
  });

  describe('collectCareInfoFields', () => {
    /**
     * Helper function that mirrors the form data collection logic from admin-dogs.html
     */
    function collectFormData() {
      const specialNeeds = document.getElementById('dog-special-needs').value.trim();
      const pickupLocation = document.getElementById('dog-pickup-location').value.trim();
      const walkRoute = document.getElementById('dog-walk-route').value.trim();
      const walkDurationVal = document.getElementById('dog-walk-duration').value;
      const specialInstructions = document.getElementById('dog-special-instructions').value.trim();
      const defaultMorningTime = document.getElementById('dog-default-morning-time').value;
      const defaultEveningTime = document.getElementById('dog-default-evening-time').value;

      return {
        name: document.getElementById('dog-name').value,
        breed: document.getElementById('dog-breed').value,
        size: document.getElementById('dog-size').value,
        age: parseInt(document.getElementById('dog-age').value) || 0,
        color_id: parseInt(document.getElementById('dog-color').value) || null,
        external_link: document.getElementById('dog-external-link').value.trim() || null,
        special_needs: specialNeeds || null,
        pickup_location: pickupLocation || null,
        walk_route: walkRoute || null,
        walk_duration: walkDurationVal ? parseInt(walkDurationVal) : null,
        special_instructions: specialInstructions || null,
        default_morning_time: defaultMorningTime || null,
        default_evening_time: defaultEveningTime || null,
      };
    }

    test('should collect all care info fields when filled', () => {
      // Fill in all fields
      document.getElementById('dog-name').value = 'Bella';
      document.getElementById('dog-breed').value = 'Labrador';
      document.getElementById('dog-size').value = 'large';
      document.getElementById('dog-age').value = '5';
      document.getElementById('dog-color').value = '1';
      document.getElementById('dog-special-needs').value = 'Needs gentle handling';
      document.getElementById('dog-pickup-location').value = 'Zwinger 3';
      document.getElementById('dog-walk-route').value = 'Waldweg';
      document.getElementById('dog-walk-duration').value = '45';
      document.getElementById('dog-special-instructions').value = 'Keep away from other dogs';
      document.getElementById('dog-default-morning-time').value = '09:00';
      document.getElementById('dog-default-evening-time').value = '17:00';

      const data = collectFormData();

      expect(data.special_needs).toBe('Needs gentle handling');
      expect(data.pickup_location).toBe('Zwinger 3');
      expect(data.walk_route).toBe('Waldweg');
      expect(data.walk_duration).toBe(45);
      expect(data.special_instructions).toBe('Keep away from other dogs');
      expect(data.default_morning_time).toBe('09:00');
      expect(data.default_evening_time).toBe('17:00');
    });

    test('should return null for empty care info fields', () => {
      document.getElementById('dog-name').value = 'Max';
      document.getElementById('dog-breed').value = 'Beagle';
      document.getElementById('dog-size').value = 'medium';
      document.getElementById('dog-age').value = '3';
      document.getElementById('dog-color').value = '2';

      const data = collectFormData();

      expect(data.special_needs).toBeNull();
      expect(data.pickup_location).toBeNull();
      expect(data.walk_route).toBeNull();
      expect(data.walk_duration).toBeNull();
      expect(data.special_instructions).toBeNull();
      expect(data.default_morning_time).toBeNull();
      expect(data.default_evening_time).toBeNull();
    });

    test('should handle partial care info', () => {
      document.getElementById('dog-name').value = 'Rocky';
      document.getElementById('dog-breed').value = 'Shepherd';
      document.getElementById('dog-size').value = 'large';
      document.getElementById('dog-age').value = '4';
      document.getElementById('dog-color').value = '1';
      document.getElementById('dog-pickup-location').value = 'Main entrance';
      document.getElementById('dog-walk-duration').value = '30';

      const data = collectFormData();

      expect(data.pickup_location).toBe('Main entrance');
      expect(data.walk_duration).toBe(30);
      expect(data.special_needs).toBeNull();
      expect(data.walk_route).toBeNull();
      expect(data.special_instructions).toBeNull();
    });

    test('should trim whitespace from text fields', () => {
      document.getElementById('dog-name').value = 'Test';
      document.getElementById('dog-breed').value = 'Test';
      document.getElementById('dog-size').value = 'small';
      document.getElementById('dog-age').value = '1';
      document.getElementById('dog-special-needs').value = '  Needs attention  ';
      document.getElementById('dog-pickup-location').value = '   Building A   ';

      const data = collectFormData();

      expect(data.special_needs).toBe('Needs attention');
      expect(data.pickup_location).toBe('Building A');
    });

    test('should handle whitespace-only fields as null', () => {
      document.getElementById('dog-name').value = 'Test';
      document.getElementById('dog-breed').value = 'Test';
      document.getElementById('dog-size').value = 'small';
      document.getElementById('dog-age').value = '1';
      document.getElementById('dog-special-needs').value = '   ';
      document.getElementById('dog-pickup-location').value = '  ';

      const data = collectFormData();

      expect(data.special_needs).toBeNull();
      expect(data.pickup_location).toBeNull();
    });
  });

  describe('populateCareInfoFields', () => {
    /**
     * Helper function that mirrors the editDog population logic from admin-dogs.html
     */
    function populateDogForm(dog) {
      document.getElementById('dog-id').value = dog.id;
      document.getElementById('dog-name').value = dog.name;
      document.getElementById('dog-breed').value = dog.breed;
      document.getElementById('dog-size').value = dog.size;
      document.getElementById('dog-age').value = dog.age;
      document.getElementById('dog-color').value = dog.color_id || '';
      document.getElementById('dog-external-link').value = dog.external_link || '';

      // Populate care info fields
      document.getElementById('dog-special-needs').value = dog.special_needs || '';
      document.getElementById('dog-pickup-location').value = dog.pickup_location || '';
      document.getElementById('dog-walk-route').value = dog.walk_route || '';
      document.getElementById('dog-walk-duration').value = dog.walk_duration || '';
      document.getElementById('dog-special-instructions').value = dog.special_instructions || '';
      document.getElementById('dog-default-morning-time').value = dog.default_morning_time || '';
      document.getElementById('dog-default-evening-time').value = dog.default_evening_time || '';
    }

    test('should populate all care info fields', () => {
      const dog = {
        id: 1,
        name: 'Bella',
        breed: 'Labrador',
        size: 'large',
        age: 5,
        color_id: 1,
        special_needs: 'Needs gentle handling',
        pickup_location: 'Zwinger 3',
        walk_route: 'Forest path',
        walk_duration: 45,
        special_instructions: 'Do not approach other dogs',
        default_morning_time: '09:00',
        default_evening_time: '17:00',
      };

      populateDogForm(dog);

      expect(document.getElementById('dog-special-needs').value).toBe('Needs gentle handling');
      expect(document.getElementById('dog-pickup-location').value).toBe('Zwinger 3');
      expect(document.getElementById('dog-walk-route').value).toBe('Forest path');
      expect(document.getElementById('dog-walk-duration').value).toBe('45');
      expect(document.getElementById('dog-special-instructions').value).toBe('Do not approach other dogs');
      expect(document.getElementById('dog-default-morning-time').value).toBe('09:00');
      expect(document.getElementById('dog-default-evening-time').value).toBe('17:00');
    });

    test('should handle null care info fields', () => {
      const dog = {
        id: 2,
        name: 'Max',
        breed: 'Beagle',
        size: 'medium',
        age: 3,
        color_id: 2,
        special_needs: null,
        pickup_location: null,
        walk_route: null,
        walk_duration: null,
        special_instructions: null,
        default_morning_time: null,
        default_evening_time: null,
      };

      populateDogForm(dog);

      expect(document.getElementById('dog-special-needs').value).toBe('');
      expect(document.getElementById('dog-pickup-location').value).toBe('');
      expect(document.getElementById('dog-walk-route').value).toBe('');
      expect(document.getElementById('dog-walk-duration').value).toBe('');
      expect(document.getElementById('dog-special-instructions').value).toBe('');
      expect(document.getElementById('dog-default-morning-time').value).toBe('');
      expect(document.getElementById('dog-default-evening-time').value).toBe('');
    });

    test('should handle undefined care info fields', () => {
      const dog = {
        id: 3,
        name: 'Rocky',
        breed: 'Shepherd',
        size: 'large',
        age: 4,
        color_id: 1,
        // Care info fields not present (undefined)
      };

      populateDogForm(dog);

      expect(document.getElementById('dog-special-needs').value).toBe('');
      expect(document.getElementById('dog-pickup-location').value).toBe('');
      expect(document.getElementById('dog-walk-route').value).toBe('');
      expect(document.getElementById('dog-walk-duration').value).toBe('');
      expect(document.getElementById('dog-special-instructions').value).toBe('');
      expect(document.getElementById('dog-default-morning-time').value).toBe('');
      expect(document.getElementById('dog-default-evening-time').value).toBe('');
    });
  });

  describe('form reset', () => {
    test('should clear all fields including care info on form reset', () => {
      // Fill in fields
      document.getElementById('dog-name').value = 'Test';
      document.getElementById('dog-special-needs').value = 'Test needs';
      document.getElementById('dog-pickup-location').value = 'Test location';
      document.getElementById('dog-walk-duration').value = '30';

      // Reset the form
      document.getElementById('dog-form').reset();

      // Verify fields are cleared
      expect(document.getElementById('dog-name').value).toBe('');
      expect(document.getElementById('dog-special-needs').value).toBe('');
      expect(document.getElementById('dog-pickup-location').value).toBe('');
      expect(document.getElementById('dog-walk-duration').value).toBe('');
    });
  });
});

describe('Dog Care Info Display', () => {
  beforeEach(() => {
    document.body.innerHTML = '<div id="dog-detail-content"></div>';
  });

  /**
   * Helper function that mirrors the care info HTML generation from dogs.html
   */
  function generateCareInfoHtml(dog) {
    const safePickupLocation = dog.pickup_location || '';
    const safeWalkRoute = dog.walk_route || '';
    const safeSpecialInstructions = dog.special_instructions || '';

    const hasCareInfo = dog.pickup_location || dog.walk_route || dog.walk_duration || dog.special_instructions;

    if (!hasCareInfo) {
      return '';
    }

    return `
      <div class="care-info-section">
        <h4>📋 Spaziergang-Informationen</h4>
        <div class="care-info-grid">
          ${safePickupLocation ? `<div><strong>📍 Abholort:</strong> ${safePickupLocation}</div>` : ''}
          ${dog.walk_duration ? `<div><strong>⏱️ Dauer:</strong> ca. ${dog.walk_duration} Minuten</div>` : ''}
          ${safeWalkRoute ? `<div><strong>🚶 Route:</strong> ${safeWalkRoute}</div>` : ''}
          ${safeSpecialInstructions ? `<div><strong>⚠️ Hinweise:</strong><br>${safeSpecialInstructions}</div>` : ''}
        </div>
      </div>
    `;
  }

  test('should generate care info HTML when all fields present', () => {
    const dog = {
      pickup_location: 'Zwinger 3',
      walk_route: 'Waldweg',
      walk_duration: 45,
      special_instructions: 'Keep calm',
    };

    const html = generateCareInfoHtml(dog);

    expect(html).toContain('📍 Abholort:');
    expect(html).toContain('Zwinger 3');
    expect(html).toContain('⏱️ Dauer:');
    expect(html).toContain('45 Minuten');
    expect(html).toContain('🚶 Route:');
    expect(html).toContain('Waldweg');
    expect(html).toContain('⚠️ Hinweise:');
    expect(html).toContain('Keep calm');
  });

  test('should generate partial care info HTML', () => {
    const dog = {
      pickup_location: 'Main entrance',
      walk_duration: 30,
    };

    const html = generateCareInfoHtml(dog);

    expect(html).toContain('📍 Abholort:');
    expect(html).toContain('Main entrance');
    expect(html).toContain('30 Minuten');
    expect(html).not.toContain('🚶 Route:');
    expect(html).not.toContain('⚠️ Hinweise:');
  });

  test('should return empty string when no care info present', () => {
    const dog = {
      name: 'Bella',
      breed: 'Labrador',
    };

    const html = generateCareInfoHtml(dog);

    expect(html).toBe('');
  });

  test('should handle null values gracefully', () => {
    const dog = {
      pickup_location: null,
      walk_route: null,
      walk_duration: null,
      special_instructions: null,
    };

    const html = generateCareInfoHtml(dog);

    expect(html).toBe('');
  });

  test('should only show care info section if at least one field is set', () => {
    const dogWithOneField = {
      walk_duration: 30,
    };

    const html = generateCareInfoHtml(dogWithOneField);

    expect(html).toContain('Spaziergang-Informationen');
    expect(html).toContain('30 Minuten');
  });
});
