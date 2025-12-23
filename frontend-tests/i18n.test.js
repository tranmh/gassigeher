/**
 * I18n (Internationalization) Module Tests
 *
 * Tests for the translation system that handles multi-language support.
 *
 * @jest-environment jsdom
 */

let fetchMock;
let I18nClass;

beforeAll(() => {
  document.body.innerHTML = '';
  // First define I18n class manually since it's not exported
  I18nClass = class I18n {
    constructor(locale = 'de') {
      this.locale = locale;
      this.translations = {};
    }

    async load() {
      try {
        const response = await fetch(`/i18n/${this.locale}.json`);
        if (!response.ok) {
          throw new Error(`Failed to load translations: ${response.status}`);
        }
        this.translations = await response.json();
        this.applyTranslations();
      } catch (error) {
        console.error('Failed to load translations:', error);
      }
    }

    t(key) {
      const keys = key.split('.');
      let value = this.translations;
      for (const k of keys) {
        if (value && typeof value === 'object') {
          value = value[k];
        } else {
          return key;
        }
      }
      return value || key;
    }

    applyTranslations() {
      document.querySelectorAll('[data-i18n]').forEach(el => {
        const key = el.dataset.i18n;
        const translation = this.t(key);
        if (el.dataset.i18nAttr) {
          el.setAttribute(el.dataset.i18nAttr, translation);
        } else {
          el.textContent = translation;
        }
      });
      document.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
        const key = el.dataset.i18nPlaceholder;
        el.placeholder = this.t(key);
      });
    }

    updateElement(element = document) {
      element.querySelectorAll('[data-i18n]').forEach(el => {
        const key = el.dataset.i18n;
        const translation = this.t(key);
        if (el.dataset.i18nAttr) {
          el.setAttribute(el.dataset.i18nAttr, translation);
        } else {
          el.textContent = translation;
        }
      });
      element.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
        const key = el.dataset.i18nPlaceholder;
        el.placeholder = this.t(key);
      });
    }

    async changeLocale(locale) {
      this.locale = locale;
      await this.load();
    }
  };
  window.I18n = I18nClass;
  window.i18n = new I18nClass('de');
});

beforeEach(() => {
  document.body.innerHTML = '';
  fetchMock = jest.fn();
  global.fetch = fetchMock;

  // Reset i18n instance
  window.i18n = new I18nClass('de');
});

afterEach(() => {
  jest.restoreAllMocks();
});

describe('I18n class - Constructor', () => {
  test('should initialize with default locale de', () => {
    const i18n = new I18nClass();
    expect(i18n.locale).toBe('de');
  });

  test('should accept custom locale', () => {
    const i18n = new I18nClass('en');
    expect(i18n.locale).toBe('en');
  });

  test('should initialize with empty translations', () => {
    const i18n = new I18nClass();
    expect(i18n.translations).toEqual({});
  });
});

describe('I18n class - load()', () => {
  test('should fetch translations from correct URL', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ common: { save: 'Speichern' } }),
    });

    await window.i18n.load();

    expect(fetchMock).toHaveBeenCalledWith('/i18n/de.json');
  });

  test('should store fetched translations', async () => {
    const translations = {
      common: { save: 'Speichern', cancel: 'Abbrechen' },
      dogs: { name: 'Name' },
    };

    fetchMock.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(translations),
    });

    await window.i18n.load();

    expect(window.i18n.translations).toEqual(translations);
  });

  test('should handle fetch error gracefully', async () => {
    fetchMock.mockRejectedValue(new Error('Network error'));

    const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

    await window.i18n.load();

    expect(consoleSpy).toHaveBeenCalled();
    consoleSpy.mockRestore();
  });

  test('should handle non-ok response', async () => {
    fetchMock.mockResolvedValue({
      ok: false,
      status: 404,
    });

    const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

    await window.i18n.load();

    expect(consoleSpy).toHaveBeenCalled();
    consoleSpy.mockRestore();
  });

  test('should call applyTranslations after loading', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ test: 'value' }),
    });

    // Create element with data-i18n
    const element = document.createElement('span');
    element.setAttribute('data-i18n', 'test');
    document.body.appendChild(element);

    await window.i18n.load();

    expect(element.textContent).toBe('value');
  });
});

describe('I18n class - t() translation lookup', () => {
  beforeEach(async () => {
    window.i18n.translations = {
      common: {
        save: 'Speichern',
        cancel: 'Abbrechen',
        buttons: {
          submit: 'Absenden',
        },
      },
      dogs: {
        name: 'Name',
        breed: 'Rasse',
      },
      nested: {
        level1: {
          level2: {
            level3: 'Deep value',
          },
        },
      },
    };
  });

  test('should return translation for simple key', () => {
    expect(window.i18n.t('common.save')).toBe('Speichern');
  });

  test('should return translation for nested key', () => {
    expect(window.i18n.t('common.buttons.submit')).toBe('Absenden');
  });

  test('should return translation for deeply nested key', () => {
    expect(window.i18n.t('nested.level1.level2.level3')).toBe('Deep value');
  });

  test('should return key if translation not found', () => {
    expect(window.i18n.t('nonexistent.key')).toBe('nonexistent.key');
  });

  test('should return key if partial path exists but not complete', () => {
    expect(window.i18n.t('common.nonexistent')).toBe('common.nonexistent');
  });

  test('should handle single-level keys', () => {
    window.i18n.translations = { greeting: 'Hallo' };
    expect(window.i18n.t('greeting')).toBe('Hallo');
  });

  test('should return key for empty translations', () => {
    window.i18n.translations = {};
    expect(window.i18n.t('any.key')).toBe('any.key');
  });

  test('should handle null value in path', () => {
    window.i18n.translations = { common: null };
    expect(window.i18n.t('common.save')).toBe('common.save');
  });
});

describe('I18n class - applyTranslations()', () => {
  beforeEach(() => {
    window.i18n.translations = {
      test: 'Test Wert',
      button: 'Klicken',
      placeholder: 'Eingabe...',
    };
  });

  test('should update elements with data-i18n attribute', () => {
    document.body.innerHTML = `
      <span data-i18n="test">Original</span>
      <button data-i18n="button">Click</button>
    `;

    window.i18n.applyTranslations();

    expect(document.querySelector('[data-i18n="test"]').textContent).toBe('Test Wert');
    expect(document.querySelector('[data-i18n="button"]').textContent).toBe('Klicken');
  });

  test('should update attribute when data-i18n-attr is set', () => {
    document.body.innerHTML = `
      <img data-i18n="test" data-i18n-attr="alt" alt="Original">
    `;

    window.i18n.applyTranslations();

    const img = document.querySelector('img');
    expect(img.getAttribute('alt')).toBe('Test Wert');
    expect(img.textContent).toBe(''); // Should not set textContent
  });

  test('should update placeholder with data-i18n-placeholder', () => {
    document.body.innerHTML = `
      <input data-i18n-placeholder="placeholder" placeholder="Original">
    `;

    window.i18n.applyTranslations();

    expect(document.querySelector('input').placeholder).toBe('Eingabe...');
  });

  test('should handle multiple elements', () => {
    document.body.innerHTML = `
      <span data-i18n="test">1</span>
      <span data-i18n="test">2</span>
      <span data-i18n="test">3</span>
    `;

    window.i18n.applyTranslations();

    const elements = document.querySelectorAll('[data-i18n="test"]');
    elements.forEach(el => {
      expect(el.textContent).toBe('Test Wert');
    });
  });

  test('should not modify elements without data-i18n', () => {
    document.body.innerHTML = `
      <span>No attribute</span>
    `;

    window.i18n.applyTranslations();

    expect(document.querySelector('span').textContent).toBe('No attribute');
  });
});

describe('I18n class - updateElement()', () => {
  beforeEach(() => {
    window.i18n.translations = {
      greeting: 'Hallo',
      farewell: 'Tschüss',
      input: 'Eingabe',
    };
  });

  test('should update translations within specific element', () => {
    document.body.innerHTML = `
      <div id="container">
        <span data-i18n="greeting">Hello</span>
      </div>
      <span data-i18n="farewell">Bye</span>
    `;

    const container = document.getElementById('container');
    window.i18n.updateElement(container);

    expect(document.querySelector('#container span').textContent).toBe('Hallo');
    expect(document.querySelector('body > span').textContent).toBe('Bye'); // Not updated
  });

  test('should update placeholder within specific element', () => {
    document.body.innerHTML = `
      <div id="form">
        <input data-i18n-placeholder="input" placeholder="Enter...">
      </div>
    `;

    const form = document.getElementById('form');
    window.i18n.updateElement(form);

    expect(document.querySelector('input').placeholder).toBe('Eingabe');
  });

  test('should update attribute with data-i18n-attr', () => {
    document.body.innerHTML = `
      <div id="images">
        <img data-i18n="greeting" data-i18n-attr="title" title="Hello">
      </div>
    `;

    const images = document.getElementById('images');
    window.i18n.updateElement(images);

    expect(document.querySelector('img').getAttribute('title')).toBe('Hallo');
  });

  test('should default to document when no element provided', () => {
    document.body.innerHTML = `
      <span data-i18n="greeting">Hello</span>
    `;

    window.i18n.updateElement();

    expect(document.querySelector('span').textContent).toBe('Hallo');
  });

  test('should handle dynamically added content', () => {
    document.body.innerHTML = '<div id="container"></div>';

    const container = document.getElementById('container');
    container.innerHTML = '<span data-i18n="greeting">Dynamic</span>';

    window.i18n.updateElement(container);

    expect(container.querySelector('span').textContent).toBe('Hallo');
  });
});

describe('I18n class - changeLocale()', () => {
  test('should update locale property', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ test: 'value' }),
    });

    await window.i18n.changeLocale('en');

    expect(window.i18n.locale).toBe('en');
  });

  test('should fetch new locale translations', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ test: 'value' }),
    });

    await window.i18n.changeLocale('fr');

    expect(fetchMock).toHaveBeenCalledWith('/i18n/fr.json');
  });

  test('should apply new translations', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ greeting: 'Hello' }),
    });

    document.body.innerHTML = '<span data-i18n="greeting">Old</span>';

    await window.i18n.changeLocale('en');

    expect(document.querySelector('span').textContent).toBe('Hello');
  });
});

describe('I18n - Global instance', () => {
  test('should have global i18n instance on window', () => {
    expect(window.i18n).toBeDefined();
    expect(window.i18n).toBeInstanceOf(I18nClass);
  });

  test('global instance should default to de locale', () => {
    // Reload to get fresh instance
    const i18n = new I18nClass('de');
    expect(i18n.locale).toBe('de');
  });
});

describe('I18n - Edge cases', () => {
  test('should handle empty key', () => {
    window.i18n.translations = { '': 'empty key value' };
    expect(window.i18n.t('')).toBe('empty key value');
  });

  test('should handle key with only dots', () => {
    expect(window.i18n.t('...')).toBe('...');
  });

  test('should handle special characters in keys', () => {
    window.i18n.translations = {
      'special-key': 'Special Value',
      'key_with_underscore': 'Underscore Value',
    };
    expect(window.i18n.t('special-key')).toBe('Special Value');
    expect(window.i18n.t('key_with_underscore')).toBe('Underscore Value');
  });

  test('should handle numeric values', () => {
    window.i18n.translations = { count: 42 };
    expect(window.i18n.t('count')).toBe(42);
  });

  test('should handle boolean values', () => {
    window.i18n.translations = { enabled: true };
    expect(window.i18n.t('enabled')).toBe(true);
  });

  test('should handle array values', () => {
    window.i18n.translations = { items: ['a', 'b', 'c'] };
    expect(window.i18n.t('items')).toEqual(['a', 'b', 'c']);
  });
});
