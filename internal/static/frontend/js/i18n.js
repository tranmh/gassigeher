// Internationalization (i18n) system
class I18n {
    constructor(locale = null) {
        // Load from localStorage or default to 'de'
        this.locale = locale || localStorage.getItem('gassigeher_language') || 'de';
        this.translations = {};
        this.availableLocales = ['de', 'en'];
        this.localeNames = {
            'de': 'Deutsch',
            'en': 'English'
        };
    }

    async load() {
        try {
            const response = await fetch(`/i18n/${this.locale}.json`);
            if (!response.ok) {
                // Fallback to German if translation not found
                if (this.locale !== 'de') {
                    console.warn(`Translation for ${this.locale} not found, falling back to German`);
                    this.locale = 'de';
                    return this.load();
                }
                throw new Error(`Failed to load translations: ${response.status}`);
            }
            this.translations = await response.json();
            this.applyTranslations();
        } catch (error) {
            console.error('Failed to load translations:', error);
        }
    }

    // Get translation by key (supports nested keys like "auth.login")
    t(key) {
        const keys = key.split('.');
        let value = this.translations;

        for (const k of keys) {
            if (value && typeof value === 'object') {
                value = value[k];
            } else {
                return key; // Return key if translation not found
            }
        }

        return value || key;
    }

    // Apply translations to elements with data-i18n attribute
    applyTranslations() {
        document.querySelectorAll('[data-i18n]').forEach(el => {
            const key = el.dataset.i18n;
            const translation = this.t(key);

            // Check if element has data-i18n-attr to translate attributes
            if (el.dataset.i18nAttr) {
                el.setAttribute(el.dataset.i18nAttr, translation);
            } else {
                el.textContent = translation;
            }
        });

        // Apply placeholder translations
        document.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
            const key = el.dataset.i18nPlaceholder;
            el.placeholder = this.t(key);
        });
    }

    // Manual method to update translations for a specific element or container
    updateElement(element = document) {
        // Handle null/undefined element gracefully
        if (!element) {
            element = document;
        }
        element.querySelectorAll('[data-i18n]').forEach(el => {
            const key = el.dataset.i18n;
            const translation = this.t(key);

            // Check if element has data-i18n-attr to translate attributes
            if (el.dataset.i18nAttr) {
                el.setAttribute(el.dataset.i18nAttr, translation);
            } else {
                el.textContent = translation;
            }
        });

        // Apply placeholder translations
        element.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
            const key = el.dataset.i18nPlaceholder;
            el.placeholder = this.t(key);
        });
    }

    // Change locale and reload
    async changeLocale(locale) {
        this.locale = locale;
        localStorage.setItem('gassigeher_language', locale);
        await this.load();
    }

    // Get current locale
    getLocale() {
        return this.locale;
    }

    // Get available locales
    getAvailableLocales() {
        return this.availableLocales.map(code => ({
            code,
            name: this.localeNames[code] || code
        }));
    }

    // Get locale display name
    getLocaleName(code) {
        return this.localeNames[code] || code;
    }
}

// Global instance
window.i18n = new I18n();
