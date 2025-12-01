// Internationalization (i18n) system
class I18n {
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
        await this.load();
    }
}

// Global instance
window.i18n = new I18n('de');
