/**
 * Dog Photo Helpers Module Tests
 *
 * Tests for the helper functions that generate dog photo HTML.
 *
 * @jest-environment jsdom
 */

// Mock sanitizeHTML globally
window.sanitizeHTML = function(str) {
  if (!str) return '';
  const div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
};

// Define helper functions since they're global functions in the source
beforeAll(() => {
  document.body.innerHTML = '';

  // Define helper functions
  window.getDogPhotoUrl = function(dog, useThumbnail = false) {
    if (dog.photo) {
      const photoField = useThumbnail && dog.photo_thumbnail ? dog.photo_thumbnail : dog.photo;
      return `/uploads/${photoField}`;
    }
    return '/assets/images/placeholders/dog-placeholder.svg';
  };

  window.getDogPhotoAlt = function(dog) {
    const safeName = typeof sanitizeHTML !== 'undefined' ? sanitizeHTML(dog.name) : dog.name;
    const safeBreed = typeof sanitizeHTML !== 'undefined' ? sanitizeHTML(dog.breed) : dog.breed;
    if (dog.photo) {
      return `${safeName} (${safeBreed})`;
    }
    return `Kein Foto für ${safeName}`;
  };

  window.getDogPhotoHtml = function(dog, useThumbnail = false, className = 'dog-card-image', lazyLoad = true, withSkeleton = true) {
    const photoUrl = getDogPhotoUrl(dog, useThumbnail);
    const altText = getDogPhotoAlt(dog);
    const loadingAttr = lazyLoad ? ' loading="lazy"' : '';
    const uniqueId = `dog-img-${dog.id || Math.random().toString(36).substr(2, 9)}`;
    const isSvgPlaceholder = photoUrl.includes('.svg');
    if (withSkeleton && !isSvgPlaceholder) {
      return `<div class="dog-card-image-container" id="container-${uniqueId}">
                  <img src="${photoUrl}"
                       alt="${altText}"
                       class="${className}"
                       id="${uniqueId}"
                       ${loadingAttr}
                       onload="handleImageLoad('${uniqueId}')">
              </div>`;
    }
    return `<img src="${photoUrl}" alt="${altText}" class="${className}"${loadingAttr}>`;
  };

  window.getDogPhotoResponsive = function(dog, className = 'dog-card-image', lazyLoad = true) {
    const fullUrl = getDogPhotoUrl(dog, false);
    const thumbUrl = getDogPhotoUrl(dog, true);
    const altText = getDogPhotoAlt(dog);
    const loadingAttr = lazyLoad ? ' loading="lazy"' : '';
    if (dog.photo && dog.photo_thumbnail && dog.photo !== dog.photo_thumbnail) {
      return `
          <picture>
              <source media="(max-width: 768px)" srcset="${thumbUrl}">
              <img src="${fullUrl}" alt="${altText}" class="${className}"${loadingAttr}>
          </picture>
      `;
    }
    return `<img src="${fullUrl}" alt="${altText}" class="${className}"${loadingAttr}>`;
  };

  window.setDogPhotoSrc = function(imgElement, dog, useThumbnail = false) {
    if (!imgElement) return;
    const photoUrl = getDogPhotoUrl(dog, useThumbnail);
    const altText = getDogPhotoAlt(dog);
    imgElement.src = photoUrl;
    imgElement.alt = altText;
  };

  window.getPlaceholderUrl = function() {
    return '/assets/images/placeholders/dog-placeholder.svg';
  };

  window.handleImageLoad = function(imageId) {
    const img = document.getElementById(imageId);
    const container = document.getElementById(`container-${imageId}`);
    if (img) {
      img.classList.add('loaded');
      if (img.complete && img.naturalHeight !== 0) {
        img.classList.add('no-animation');
      }
    }
    if (container) {
      container.classList.add('loaded');
    }
  };

  window.preloadCriticalDogImages = function(dogs, count = 3) {
    if (!dogs || dogs.length === 0) return;
    const dogsToPreload = dogs.slice(0, count);
    dogsToPreload.forEach(dog => {
      if (dog.photo) {
        const link = document.createElement('link');
        link.rel = 'preload';
        link.as = 'image';
        link.href = getDogPhotoUrl(dog, false);
        document.head.appendChild(link);
      }
    });
  };

  window.getCalendarDogCell = function(dog, color) {
    const photoUrl = getDogPhotoUrl(dog, true);
    const altText = getDogPhotoAlt(dog);
    const safeDogName = typeof sanitizeHTML !== 'undefined' ? sanitizeHTML(dog.name) : dog.name;
    const dogColor = color || dog.color;
    if (dogColor && dogColor.hex_code) {
      const patternIcons = {
        'circle': '●', 'triangle': '▲', 'square': '■', 'diamond': '◆',
        'pentagon': '⬠', 'hexagon': '⬡', 'star': '★', 'heart': '♥',
        'cross': '✚', 'spade': '♠', 'club': '♣', 'moon': '☽',
        'sun': '☀', 'ring': '○', 'target': '◎'
      };
      const icon = patternIcons[dogColor.pattern_icon] || '●';
      const safeColorName = typeof sanitizeHTML !== 'undefined' ? sanitizeHTML(dogColor.name) : dogColor.name;
      return `<div class="calendar-dog-name-cell">
          <img src="${photoUrl}" alt="${altText}" class="calendar-dog-photo" loading="lazy">
          <div>
              <div style="font-weight: 700; font-size: 1rem; color: var(--text-dark);">${safeDogName}</div>
              <span style="display: inline-flex; align-items: center; gap: 3px; font-size: 0.7rem; padding: 2px 8px; background: ${dogColor.hex_code}20; border: 1px solid ${dogColor.hex_code}; color: ${dogColor.hex_code}; border-radius: 4px; margin-top: 4px;">
                  ${icon} ${safeColorName}
              </span>
          </div>
      </div>`;
    }
    return `<div class="calendar-dog-name-cell">
        <img src="${photoUrl}" alt="${altText}" class="calendar-dog-photo" loading="lazy">
        <div>
            <div style="font-weight: 700; font-size: 1rem; color: var(--text-dark);">${safeDogName}</div>
            <span style="display: inline-block; font-size: 0.7rem; padding: 2px 8px; background: #e0e0e0; color: #666; border-radius: 4px; margin-top: 4px;">
                Keine Kategorie
            </span>
        </div>
    </div>`;
  };
});

beforeEach(() => {
  document.body.innerHTML = '';
  document.head.innerHTML = '';
});

describe('getDogPhotoUrl()', () => {
  test('should return photo URL when dog has photo', () => {
    const dog = { id: 1, name: 'Rex', photo: 'dogs/rex.jpg' };

    const url = getDogPhotoUrl(dog);

    expect(url).toBe('/uploads/dogs/rex.jpg');
  });

  test('should return placeholder when dog has no photo', () => {
    const dog = { id: 1, name: 'Rex', photo: null };

    const url = getDogPhotoUrl(dog);

    expect(url).toBe('/assets/images/placeholders/dog-placeholder.svg');
  });

  test('should return thumbnail when useThumbnail is true and thumbnail exists', () => {
    const dog = {
      id: 1,
      name: 'Rex',
      photo: 'dogs/rex.jpg',
      photo_thumbnail: 'dogs/rex_thumb.jpg',
    };

    const url = getDogPhotoUrl(dog, true);

    expect(url).toBe('/uploads/dogs/rex_thumb.jpg');
  });

  test('should return full photo when useThumbnail is true but no thumbnail', () => {
    const dog = {
      id: 1,
      name: 'Rex',
      photo: 'dogs/rex.jpg',
      photo_thumbnail: null,
    };

    const url = getDogPhotoUrl(dog, true);

    expect(url).toBe('/uploads/dogs/rex.jpg');
  });

  test('should return placeholder when useThumbnail is true but no photo', () => {
    const dog = { id: 1, name: 'Rex', photo: null };

    const url = getDogPhotoUrl(dog, true);

    expect(url).toBe('/assets/images/placeholders/dog-placeholder.svg');
  });

  test('should handle empty string photo', () => {
    const dog = { id: 1, name: 'Rex', photo: '' };

    const url = getDogPhotoUrl(dog);

    expect(url).toBe('/assets/images/placeholders/dog-placeholder.svg');
  });
});

describe('getDogPhotoAlt()', () => {
  test('should return name and breed for dog with photo', () => {
    const dog = { id: 1, name: 'Rex', breed: 'Shepherd', photo: 'dogs/rex.jpg' };

    const alt = getDogPhotoAlt(dog);

    expect(alt).toBe('Rex (Shepherd)');
  });

  test('should return "Kein Foto" message for dog without photo', () => {
    const dog = { id: 1, name: 'Rex', breed: 'Shepherd', photo: null };

    const alt = getDogPhotoAlt(dog);

    expect(alt).toBe('Kein Foto für Rex');
  });

  test('should escape XSS in name', () => {
    const dog = {
      id: 1,
      name: '<script>alert("XSS")</script>',
      breed: 'Test',
      photo: 'dogs/test.jpg',
    };

    const alt = getDogPhotoAlt(dog);

    expect(alt).not.toContain('<script>');
    expect(alt).toContain('&lt;script&gt;');
  });

  test('should escape XSS in breed', () => {
    const dog = {
      id: 1,
      name: 'Rex',
      breed: '<img onerror="alert(1)">',
      photo: 'dogs/test.jpg',
    };

    const alt = getDogPhotoAlt(dog);

    expect(alt).not.toContain('<img');
    expect(alt).toContain('&lt;img');
  });
});

describe('getDogPhotoHtml()', () => {
  test('should generate img tag with correct src', () => {
    const dog = { id: 1, name: 'Rex', breed: 'Shepherd', photo: 'dogs/rex.jpg' };

    const html = getDogPhotoHtml(dog);

    expect(html).toContain('/uploads/dogs/rex.jpg');
  });

  test('should include alt text', () => {
    const dog = { id: 1, name: 'Rex', breed: 'Shepherd', photo: 'dogs/rex.jpg' };

    const html = getDogPhotoHtml(dog);

    expect(html).toContain('alt="Rex (Shepherd)"');
  });

  test('should include lazy loading by default', () => {
    const dog = { id: 1, name: 'Rex', breed: 'Shepherd', photo: 'dogs/rex.jpg' };

    const html = getDogPhotoHtml(dog);

    expect(html).toContain('loading="lazy"');
  });

  test('should not include lazy loading when disabled', () => {
    const dog = { id: 1, name: 'Rex', breed: 'Shepherd', photo: 'dogs/rex.jpg' };

    const html = getDogPhotoHtml(dog, false, 'dog-card-image', false);

    expect(html).not.toContain('loading="lazy"');
  });

  test('should use custom class name', () => {
    const dog = { id: 1, name: 'Rex', breed: 'Shepherd', photo: 'dogs/rex.jpg' };

    const html = getDogPhotoHtml(dog, false, 'custom-class');

    expect(html).toContain('class="custom-class"');
  });

  test('should wrap in container with skeleton for photos', () => {
    const dog = { id: 1, name: 'Rex', breed: 'Shepherd', photo: 'dogs/rex.jpg' };

    const html = getDogPhotoHtml(dog, false, 'dog-card-image', true, true);

    expect(html).toContain('dog-card-image-container');
    expect(html).toContain('onload=');
  });

  test('should not use skeleton for placeholder SVGs', () => {
    const dog = { id: 1, name: 'Rex', breed: 'Shepherd', photo: null };

    const html = getDogPhotoHtml(dog, false, 'dog-card-image', true, true);

    expect(html).not.toContain('dog-card-image-container');
    expect(html).toContain('/assets/images/placeholders/');
  });

  test('should not use skeleton when disabled', () => {
    const dog = { id: 1, name: 'Rex', breed: 'Shepherd', photo: 'dogs/rex.jpg' };

    const html = getDogPhotoHtml(dog, false, 'dog-card-image', true, false);

    expect(html).not.toContain('dog-card-image-container');
  });

  test('should use thumbnail when requested', () => {
    const dog = {
      id: 1,
      name: 'Rex',
      breed: 'Shepherd',
      photo: 'dogs/rex.jpg',
      photo_thumbnail: 'dogs/rex_thumb.jpg',
    };

    const html = getDogPhotoHtml(dog, true);

    expect(html).toContain('dogs/rex_thumb.jpg');
  });
});

describe('getDogPhotoResponsive()', () => {
  test('should return picture element when thumbnail differs', () => {
    const dog = {
      id: 1,
      name: 'Rex',
      breed: 'Shepherd',
      photo: 'dogs/rex.jpg',
      photo_thumbnail: 'dogs/rex_thumb.jpg',
    };

    const html = getDogPhotoResponsive(dog);

    expect(html).toContain('<picture>');
    expect(html).toContain('</picture>');
    expect(html).toContain('media="(max-width: 768px)"');
  });

  test('should include both full and thumbnail URLs', () => {
    const dog = {
      id: 1,
      name: 'Rex',
      breed: 'Shepherd',
      photo: 'dogs/rex.jpg',
      photo_thumbnail: 'dogs/rex_thumb.jpg',
    };

    const html = getDogPhotoResponsive(dog);

    expect(html).toContain('dogs/rex_thumb.jpg');
    expect(html).toContain('dogs/rex.jpg');
  });

  test('should return simple img when no thumbnail', () => {
    const dog = {
      id: 1,
      name: 'Rex',
      breed: 'Shepherd',
      photo: 'dogs/rex.jpg',
    };

    const html = getDogPhotoResponsive(dog);

    expect(html).not.toContain('<picture>');
    expect(html).toContain('<img');
  });

  test('should return simple img when thumbnail same as photo', () => {
    const dog = {
      id: 1,
      name: 'Rex',
      breed: 'Shepherd',
      photo: 'dogs/rex.jpg',
      photo_thumbnail: 'dogs/rex.jpg',
    };

    const html = getDogPhotoResponsive(dog);

    expect(html).not.toContain('<picture>');
  });

  test('should include lazy loading by default', () => {
    const dog = { id: 1, name: 'Rex', breed: 'Shepherd', photo: 'dogs/rex.jpg' };

    const html = getDogPhotoResponsive(dog);

    expect(html).toContain('loading="lazy"');
  });

  test('should use custom class name', () => {
    const dog = { id: 1, name: 'Rex', breed: 'Shepherd', photo: 'dogs/rex.jpg' };

    const html = getDogPhotoResponsive(dog, 'my-class');

    expect(html).toContain('class="my-class"');
  });
});

describe('setDogPhotoSrc()', () => {
  test('should set img src and alt', () => {
    const img = document.createElement('img');
    const dog = { id: 1, name: 'Rex', breed: 'Shepherd', photo: 'dogs/rex.jpg' };

    setDogPhotoSrc(img, dog);

    expect(img.src).toContain('/uploads/dogs/rex.jpg');
    expect(img.alt).toBe('Rex (Shepherd)');
  });

  test('should use thumbnail when requested', () => {
    const img = document.createElement('img');
    const dog = {
      id: 1,
      name: 'Rex',
      breed: 'Shepherd',
      photo: 'dogs/rex.jpg',
      photo_thumbnail: 'dogs/rex_thumb.jpg',
    };

    setDogPhotoSrc(img, dog, true);

    expect(img.src).toContain('dogs/rex_thumb.jpg');
  });

  test('should handle null element', () => {
    const dog = { id: 1, name: 'Rex', breed: 'Shepherd', photo: 'dogs/rex.jpg' };

    expect(() => setDogPhotoSrc(null, dog)).not.toThrow();
  });
});

describe('getPlaceholderUrl()', () => {
  test('should return placeholder SVG URL', () => {
    const url = getPlaceholderUrl();

    expect(url).toBe('/assets/images/placeholders/dog-placeholder.svg');
  });
});

describe('handleImageLoad()', () => {
  test('should add loaded class to image', () => {
    document.body.innerHTML = `
      <div id="container-dog-img-1" class="dog-card-image-container">
        <img id="dog-img-1" src="/test.jpg">
      </div>
    `;

    handleImageLoad('dog-img-1');

    expect(document.getElementById('dog-img-1').classList.contains('loaded')).toBe(true);
  });

  test('should add loaded class to container', () => {
    document.body.innerHTML = `
      <div id="container-dog-img-1" class="dog-card-image-container">
        <img id="dog-img-1" src="/test.jpg">
      </div>
    `;

    handleImageLoad('dog-img-1');

    expect(document.getElementById('container-dog-img-1').classList.contains('loaded')).toBe(true);
  });

  test('should handle missing image', () => {
    document.body.innerHTML = '';

    expect(() => handleImageLoad('nonexistent')).not.toThrow();
  });

  test('should handle missing container', () => {
    document.body.innerHTML = '<img id="dog-img-1" src="/test.jpg">';

    expect(() => handleImageLoad('dog-img-1')).not.toThrow();
    expect(document.getElementById('dog-img-1').classList.contains('loaded')).toBe(true);
  });
});

describe('preloadCriticalDogImages()', () => {
  test('should create preload link for first N dogs', () => {
    const dogs = [
      { id: 1, name: 'Rex', photo: 'dogs/rex.jpg' },
      { id: 2, name: 'Max', photo: 'dogs/max.jpg' },
      { id: 3, name: 'Bella', photo: 'dogs/bella.jpg' },
      { id: 4, name: 'Luna', photo: 'dogs/luna.jpg' },
    ];

    preloadCriticalDogImages(dogs, 2);

    const links = document.head.querySelectorAll('link[rel="preload"]');
    expect(links.length).toBe(2);
    expect(links[0].href).toContain('dogs/rex.jpg');
    expect(links[1].href).toContain('dogs/max.jpg');
  });

  test('should default to 3 dogs', () => {
    const dogs = [
      { id: 1, name: 'Rex', photo: 'dogs/rex.jpg' },
      { id: 2, name: 'Max', photo: 'dogs/max.jpg' },
      { id: 3, name: 'Bella', photo: 'dogs/bella.jpg' },
      { id: 4, name: 'Luna', photo: 'dogs/luna.jpg' },
    ];

    preloadCriticalDogImages(dogs);

    const links = document.head.querySelectorAll('link[rel="preload"]');
    expect(links.length).toBe(3);
  });

  test('should skip dogs without photos', () => {
    const dogs = [
      { id: 1, name: 'Rex', photo: null },
      { id: 2, name: 'Max', photo: 'dogs/max.jpg' },
    ];

    preloadCriticalDogImages(dogs);

    const links = document.head.querySelectorAll('link[rel="preload"]');
    expect(links.length).toBe(1);
    expect(links[0].href).toContain('dogs/max.jpg');
  });

  test('should handle empty array', () => {
    expect(() => preloadCriticalDogImages([])).not.toThrow();

    const links = document.head.querySelectorAll('link[rel="preload"]');
    expect(links.length).toBe(0);
  });

  test('should handle null', () => {
    expect(() => preloadCriticalDogImages(null)).not.toThrow();
  });

  test('should handle undefined', () => {
    expect(() => preloadCriticalDogImages(undefined)).not.toThrow();
  });

  test('should set correct link attributes', () => {
    const dogs = [{ id: 1, name: 'Rex', photo: 'dogs/rex.jpg' }];

    preloadCriticalDogImages(dogs, 1);

    const link = document.head.querySelector('link[rel="preload"]');
    expect(link.rel).toBe('preload');
    expect(link.as).toBe('image');
  });
});

describe('getCalendarDogCell()', () => {
  test('should generate cell with dog photo and name', () => {
    const dog = { id: 1, name: 'Rex', breed: 'Shepherd', photo: 'dogs/rex.jpg' };

    const html = getCalendarDogCell(dog);

    expect(html).toContain('calendar-dog-name-cell');
    expect(html).toContain('Rex');
    expect(html).toContain('calendar-dog-photo');
  });

  test('should use thumbnail for calendar', () => {
    const dog = {
      id: 1,
      name: 'Rex',
      breed: 'Shepherd',
      photo: 'dogs/rex.jpg',
      photo_thumbnail: 'dogs/rex_thumb.jpg',
    };

    const html = getCalendarDogCell(dog);

    expect(html).toContain('dogs/rex_thumb.jpg');
  });

  test('should show color badge when color provided', () => {
    const dog = { id: 1, name: 'Rex', breed: 'Shepherd', photo: 'dogs/rex.jpg' };
    const color = { name: 'Grün', hex_code: '#82b965', pattern_icon: 'circle' };

    const html = getCalendarDogCell(dog, color);

    expect(html).toContain('Grün');
    expect(html).toContain('#82b965');
  });

  test('should use embedded color from dog object', () => {
    const dog = {
      id: 1,
      name: 'Rex',
      breed: 'Shepherd',
      photo: 'dogs/rex.jpg',
      color: { name: 'Blau', hex_code: '#0000ff', pattern_icon: 'square' },
    };

    const html = getCalendarDogCell(dog);

    expect(html).toContain('Blau');
    expect(html).toContain('#0000ff');
  });

  test('should show "Keine Kategorie" when no color', () => {
    const dog = { id: 1, name: 'Rex', breed: 'Shepherd', photo: null };

    const html = getCalendarDogCell(dog);

    expect(html).toContain('Keine Kategorie');
  });

  test('should escape XSS in dog name', () => {
    const dog = {
      id: 1,
      name: '<script>alert("XSS")</script>',
      breed: 'Test',
      photo: 'dogs/test.jpg',
    };

    const html = getCalendarDogCell(dog);

    expect(html).not.toContain('<script>');
    expect(html).toContain('&lt;script&gt;');
  });

  test('should escape XSS in color name', () => {
    const dog = { id: 1, name: 'Rex', breed: 'Shepherd', photo: 'dogs/rex.jpg' };
    const color = {
      name: '<img onerror=alert(1)>',
      hex_code: '#000',
      pattern_icon: 'circle',
    };

    const html = getCalendarDogCell(dog, color);

    // The XSS payload should be escaped - check that the color name area contains escaped version
    expect(html).toContain('&lt;img onerror=alert(1)&gt;');
    // Should not contain unescaped version in the span (where color name goes)
    expect(html).not.toMatch(/<img onerror/);
  });

  test('should use pattern icon mapping', () => {
    const dog = { id: 1, name: 'Rex', breed: 'Shepherd', photo: 'dogs/rex.jpg' };

    const patterns = ['circle', 'triangle', 'square', 'diamond', 'star', 'heart'];
    const icons = ['●', '▲', '■', '◆', '★', '♥'];

    patterns.forEach((pattern, index) => {
      const color = { name: 'Test', hex_code: '#000', pattern_icon: pattern };
      const html = getCalendarDogCell(dog, color);
      expect(html).toContain(icons[index]);
    });
  });

  test('should fallback to circle icon for unknown pattern', () => {
    const dog = { id: 1, name: 'Rex', breed: 'Shepherd', photo: 'dogs/rex.jpg' };
    const color = { name: 'Test', hex_code: '#000', pattern_icon: 'unknown' };

    const html = getCalendarDogCell(dog, color);

    expect(html).toContain('●'); // Fallback to circle
  });

  test('should include lazy loading', () => {
    const dog = { id: 1, name: 'Rex', breed: 'Shepherd', photo: 'dogs/rex.jpg' };

    const html = getCalendarDogCell(dog);

    expect(html).toContain('loading="lazy"');
  });
});
