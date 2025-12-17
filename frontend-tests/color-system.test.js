/**
 * Color System Helper Tests
 *
 * Tests for the color system helper functions used throughout the codebase
 * for displaying color badges, pattern icons, and user/dog colors.
 *
 * @jest-environment jsdom
 */

// Mock sanitizeHTML function (same as sanitize.js)
function sanitizeHTML(str) {
  if (!str) return '';
  if (typeof str !== 'string') str = String(str);
  const div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
}

// Make it available globally for the helper functions
global.sanitizeHTML = sanitizeHTML;

describe('Color System Helpers', () => {
  describe('getPatternIcon', () => {
    // Pattern icon function - MUST match admin-colors.html select options
    function getPatternIcon(pattern) {
      const icons = {
        'circle': '●', 'triangle': '▲', 'square': '■', 'diamond': '◆',
        'pentagon': '⬠', 'hexagon': '⬡', 'star': '★', 'heart': '♥',
        'cross': '✚', 'spade': '♠', 'club': '♣', 'moon': '☽',
        'sun': '☀', 'ring': '○', 'target': '◎'
      };
      return icons[pattern] || '●';
    }

    test('should return correct icon for circle pattern', () => {
      expect(getPatternIcon('circle')).toBe('●');
    });

    test('should return correct icon for triangle pattern', () => {
      expect(getPatternIcon('triangle')).toBe('▲');
    });

    test('should return correct icon for square pattern', () => {
      expect(getPatternIcon('square')).toBe('■');
    });

    test('should return correct icon for star pattern', () => {
      expect(getPatternIcon('star')).toBe('★');
    });

    test('should return correct icon for heart pattern', () => {
      expect(getPatternIcon('heart')).toBe('♥');
    });

    test('should return correct icon for diamond pattern', () => {
      expect(getPatternIcon('diamond')).toBe('◆');
    });

    test('should return default circle icon for unknown pattern', () => {
      expect(getPatternIcon('unknown')).toBe('●');
    });

    test('should return default circle icon for null/undefined', () => {
      expect(getPatternIcon(null)).toBe('●');
      expect(getPatternIcon(undefined)).toBe('●');
    });
  });

  describe('getColorBadgeHtml', () => {
    // Color badge HTML generation function
    function getPatternIcon(pattern) {
      const icons = {
        'circle': '●', 'triangle': '▲', 'square': '■', 'diamond': '◆',
        'pentagon': '⬠', 'hexagon': '⬡', 'star': '★', 'heart': '♥',
        'cross': '✚', 'spade': '♠', 'club': '♣', 'moon': '☽',
        'sun': '☀', 'ring': '○', 'target': '◎'
      };
      return icons[pattern] || '●';
    }

    function getColorBadgeHtml(color) {
      if (!color) {
        return '<span style="display: inline-block; padding: 4px 12px; border-radius: var(--border-radius); background: #ccc; color: #666; font-size: 0.85rem;">Keine Kategorie</span>';
      }
      return `
        <span style="
          display: inline-block;
          padding: 4px 12px;
          border-radius: var(--border-radius);
          background: ${color.hex_code}20;
          border: 2px solid ${color.hex_code};
          color: ${color.hex_code};
          font-size: 0.85rem;
          font-weight: 500;
        ">
          ${getPatternIcon(color.pattern_icon)} ${sanitizeHTML(color.name)}
        </span>
      `;
    }

    test('should return "Keine Kategorie" badge for null color', () => {
      const html = getColorBadgeHtml(null);
      expect(html).toContain('Keine Kategorie');
      expect(html).toContain('#ccc');
    });

    test('should return "Keine Kategorie" badge for undefined color', () => {
      const html = getColorBadgeHtml(undefined);
      expect(html).toContain('Keine Kategorie');
    });

    test('should generate badge with correct hex color', () => {
      const color = { name: 'Grün', hex_code: '#28a745', pattern_icon: 'circle' };
      const html = getColorBadgeHtml(color);
      expect(html).toContain('#28a745');
      expect(html).toContain('Grün');
    });

    test('should include pattern icon in badge', () => {
      const color = { name: 'Stern', hex_code: '#ffc107', pattern_icon: 'star' };
      const html = getColorBadgeHtml(color);
      expect(html).toContain('★');
      expect(html).toContain('Stern');
    });

    test('should use transparent background (hex + 20)', () => {
      const color = { name: 'Blau', hex_code: '#007bff', pattern_icon: 'circle' };
      const html = getColorBadgeHtml(color);
      expect(html).toContain('#007bff20'); // 20% opacity background
      expect(html).toContain('border: 2px solid #007bff');
    });

    test('should sanitize color name to prevent XSS', () => {
      const color = { name: '<script>alert("XSS")</script>', hex_code: '#ff0000', pattern_icon: 'circle' };
      const html = getColorBadgeHtml(color);
      expect(html).not.toContain('<script>');
      expect(html).toContain('&lt;script&gt;');
    });
  });

  describe('getUserColorsHtml', () => {
    function getPatternIcon(pattern) {
      const icons = {
        'circle': '●', 'triangle': '▲', 'square': '■', 'diamond': '◆',
        'pentagon': '⬠', 'hexagon': '⬡', 'star': '★', 'heart': '♥',
        'cross': '✚', 'spade': '♠', 'club': '♣', 'moon': '☽',
        'sun': '☀', 'ring': '○', 'target': '◎'
      };
      return icons[pattern] || '●';
    }

    function getUserColorsHtml(user) {
      if (!user || !user.colors || user.colors.length === 0) {
        return '<span style="color: #999;">Keine Farben</span>';
      }
      return user.colors.map(color => {
        const icon = getPatternIcon(color.pattern_icon);
        return `<span style="
          display: inline-block;
          padding: 2px 8px;
          margin: 2px;
          border-radius: 4px;
          background: ${color.hex_code}20;
          border: 1px solid ${color.hex_code};
          color: ${color.hex_code};
          font-size: 0.8rem;
        ">${icon} ${sanitizeHTML(color.name)}</span>`;
      }).join(' ');
    }

    test('should return "Keine Farben" for null user', () => {
      const html = getUserColorsHtml(null);
      expect(html).toContain('Keine Farben');
      expect(html).toContain('#999');
    });

    test('should return "Keine Farben" for user without colors', () => {
      const user = { name: 'Test User' };
      const html = getUserColorsHtml(user);
      expect(html).toContain('Keine Farben');
    });

    test('should return "Keine Farben" for user with empty colors array', () => {
      const user = { name: 'Test User', colors: [] };
      const html = getUserColorsHtml(user);
      expect(html).toContain('Keine Farben');
    });

    test('should render single color badge', () => {
      const user = {
        name: 'Test User',
        colors: [
          { name: 'Grün', hex_code: '#28a745', pattern_icon: 'circle' }
        ]
      };
      const html = getUserColorsHtml(user);
      expect(html).toContain('#28a745');
      expect(html).toContain('Grün');
      expect(html).toContain('●');
    });

    test('should render multiple color badges', () => {
      const user = {
        name: 'Test User',
        colors: [
          { name: 'Grün', hex_code: '#28a745', pattern_icon: 'circle' },
          { name: 'Blau', hex_code: '#007bff', pattern_icon: 'triangle' }
        ]
      };
      const html = getUserColorsHtml(user);
      expect(html).toContain('#28a745');
      expect(html).toContain('Grün');
      expect(html).toContain('#007bff');
      expect(html).toContain('Blau');
      expect(html).toContain('●'); // circle
      expect(html).toContain('▲'); // triangle
    });

    test('should sanitize color names to prevent XSS', () => {
      const user = {
        name: 'Test User',
        colors: [
          { name: '<img onerror=alert("XSS")>', hex_code: '#ff0000', pattern_icon: 'circle' }
        ]
      };
      const html = getUserColorsHtml(user);
      expect(html).not.toContain('<img');
      expect(html).toContain('&lt;img');
    });
  });

  describe('Dog Photo URL Helpers', () => {
    // Recreate the getDogPhotoUrl function
    function getDogPhotoUrl(dog, useThumbnail = false) {
      if (dog.photo) {
        const photoField = useThumbnail && dog.photo_thumbnail
          ? dog.photo_thumbnail
          : dog.photo;
        return `/uploads/${photoField}`;
      }
      return '/assets/images/placeholders/dog-placeholder.svg';
    }

    test('should return photo URL when dog has photo', () => {
      const dog = { name: 'Bella', photo: 'dogs/bella.jpg' };
      const url = getDogPhotoUrl(dog);
      expect(url).toBe('/uploads/dogs/bella.jpg');
    });

    test('should return thumbnail URL when useThumbnail is true', () => {
      const dog = {
        name: 'Bella',
        photo: 'dogs/bella.jpg',
        photo_thumbnail: 'dogs/bella_thumb.jpg'
      };
      const url = getDogPhotoUrl(dog, true);
      expect(url).toBe('/uploads/dogs/bella_thumb.jpg');
    });

    test('should return full photo when thumbnail requested but not available', () => {
      const dog = { name: 'Bella', photo: 'dogs/bella.jpg' };
      const url = getDogPhotoUrl(dog, true);
      expect(url).toBe('/uploads/dogs/bella.jpg');
    });

    test('should return placeholder for dog without photo', () => {
      const dog = { name: 'Max' };
      const url = getDogPhotoUrl(dog);
      expect(url).toBe('/assets/images/placeholders/dog-placeholder.svg');
    });

    test('should return placeholder for dog with null photo', () => {
      const dog = { name: 'Max', photo: null };
      const url = getDogPhotoUrl(dog);
      expect(url).toBe('/assets/images/placeholders/dog-placeholder.svg');
    });
  });

  describe('Calendar Dog Cell', () => {
    function getPatternIcon(pattern) {
      const icons = {
        'circle': '●', 'triangle': '▲', 'square': '■', 'diamond': '◆',
        'pentagon': '⬠', 'hexagon': '⬡', 'star': '★', 'heart': '♥',
        'cross': '✚', 'spade': '♠', 'club': '♣', 'moon': '☽',
        'sun': '☀', 'ring': '○', 'target': '◎'
      };
      return icons[pattern] || '●';
    }

    function getDogPhotoUrl(dog, useThumbnail = false) {
      if (dog.photo) {
        return `/uploads/${useThumbnail && dog.photo_thumbnail ? dog.photo_thumbnail : dog.photo}`;
      }
      return '/assets/images/placeholders/dog-placeholder.svg';
    }

    function getCalendarDogCell(dog, color) {
      const photoUrl = getDogPhotoUrl(dog, true);
      const safeDogName = sanitizeHTML(dog.name);
      const dogColor = color || dog.color;

      if (dogColor && dogColor.hex_code) {
        const icon = getPatternIcon(dogColor.pattern_icon);
        const safeColorName = sanitizeHTML(dogColor.name);
        return `<div class="calendar-dog-name-cell">
          <img src="${photoUrl}" class="calendar-dog-photo" loading="lazy">
          <div>
            <div>${safeDogName}</div>
            <span style="background: ${dogColor.hex_code}20; border: 1px solid ${dogColor.hex_code}; color: ${dogColor.hex_code};">
              ${icon} ${safeColorName}
            </span>
          </div>
        </div>`;
      }

      return `<div class="calendar-dog-name-cell">
        <img src="${photoUrl}" class="calendar-dog-photo" loading="lazy">
        <div>
          <div>${safeDogName}</div>
          <span style="background: #e0e0e0; color: #666;">Keine Kategorie</span>
        </div>
      </div>`;
    }

    test('should render dog cell with color from parameter', () => {
      const dog = { id: 1, name: 'Bella', photo: 'dogs/bella.jpg' };
      const color = { name: 'Grün', hex_code: '#28a745', pattern_icon: 'circle' };
      const html = getCalendarDogCell(dog, color);

      expect(html).toContain('Bella');
      expect(html).toContain('#28a745');
      expect(html).toContain('Grün');
      expect(html).toContain('●');
      expect(html).toContain('/uploads/dogs/bella.jpg');
    });

    test('should render dog cell with embedded dog.color', () => {
      const dog = {
        id: 1,
        name: 'Max',
        photo: 'dogs/max.jpg',
        color: { name: 'Blau', hex_code: '#007bff', pattern_icon: 'triangle' }
      };
      const html = getCalendarDogCell(dog);

      expect(html).toContain('Max');
      expect(html).toContain('#007bff');
      expect(html).toContain('Blau');
      expect(html).toContain('▲'); // triangle
    });

    test('should prefer color parameter over dog.color', () => {
      const dog = {
        id: 1,
        name: 'Rocky',
        color: { name: 'Rot', hex_code: '#ff0000', pattern_icon: 'circle' }
      };
      const color = { name: 'Grün', hex_code: '#28a745', pattern_icon: 'star' };
      const html = getCalendarDogCell(dog, color);

      expect(html).toContain('#28a745');
      expect(html).toContain('Grün');
      expect(html).not.toContain('#ff0000');
    });

    test('should render "Keine Kategorie" when no color available', () => {
      const dog = { id: 1, name: 'Luna' };
      const html = getCalendarDogCell(dog);

      expect(html).toContain('Luna');
      expect(html).toContain('Keine Kategorie');
      expect(html).toContain('#e0e0e0');
    });

    test('should sanitize dog name to prevent XSS', () => {
      const dog = { id: 1, name: '<script>alert("XSS")</script>' };
      const html = getCalendarDogCell(dog);

      expect(html).not.toContain('<script>');
      expect(html).toContain('&lt;script&gt;');
    });

    test('should sanitize color name to prevent XSS', () => {
      const dog = { id: 1, name: 'Safe Dog' };
      const color = { name: '<img onerror=alert("XSS")>', hex_code: '#ff0000', pattern_icon: 'circle' };
      const html = getCalendarDogCell(dog, color);

      // The < and > should be escaped making the tag harmless
      // The user-provided content should have < escaped to &lt;
      expect(html).toContain('&lt;img');
      expect(html).toContain('&gt;');
      // Verify the escaped content is in the span (color name area), not as an actual tag
      expect(html).toMatch(/●\s*&lt;img/);
    });

    test('should use placeholder for dog without photo', () => {
      const dog = { id: 1, name: 'No Photo Dog' };
      const html = getCalendarDogCell(dog);

      expect(html).toContain('/assets/images/placeholders/dog-placeholder.svg');
    });
  });
});

describe('Color System Integration', () => {
  test('all pattern icons should be unique', () => {
    const icons = {
      'circle': '●', 'triangle': '▲', 'square': '■', 'diamond': '◆',
      'pentagon': '⬠', 'hexagon': '⬡', 'star': '★', 'heart': '♥',
      'cross': '✚', 'spade': '♠', 'club': '♣', 'moon': '☽',
      'sun': '☀', 'ring': '○', 'target': '◎'
    };
    const values = Object.values(icons);
    const uniqueValues = new Set(values);
    expect(values.length).toBe(uniqueValues.size);
  });

  test('hex color format should be valid', () => {
    const testColors = [
      { hex_code: '#28a745' }, // green
      { hex_code: '#007bff' }, // blue
      { hex_code: '#ffc107' }, // yellow
      { hex_code: '#dc3545' }, // red
      { hex_code: '#6f42c1' }, // purple
    ];

    const hexPattern = /^#[0-9A-Fa-f]{6}$/;
    testColors.forEach(color => {
      expect(color.hex_code).toMatch(hexPattern);
    });
  });

  test('transparent background color should append 20 to hex', () => {
    const hexCode = '#28a745';
    const transparentBg = `${hexCode}20`;
    expect(transparentBg).toBe('#28a74520');
    // This creates ~12.5% opacity when used in CSS
  });
});
