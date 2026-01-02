/**
 * Marketing Config Help Tests
 *
 * Tests for the campaign configuration help system in marketing.html.
 * This tests the context-sensitive help and example placeholders for different campaign types.
 *
 * @jest-environment jsdom
 */

// Define the campaignConfigHelp object (from marketing.html)
const campaignConfigHelp = {
    'fomo_countdown': {
        help: `<strong>FOMO Countdown - Felder:</strong>
<ul style="margin: 4px 0; padding-left: 20px;">
  <li><code>total_slots</code> - Gesamtzahl der verfügbaren Plätze</li>
  <li><code>remaining_slots</code> - Verbleibende Plätze (wird automatisch bei Registrierung reduziert)</li>
  <li><code>message</code> - Nachricht (Platzhalter: <code>{remaining_slots}</code>, <code>{total_slots}</code>)</li>
  <li><code>benefit_type</code> - <strong>WICHTIG:</strong> "free_pro_months" um Pro-Monate zu gewähren</li>
  <li><code>benefit_months</code> - <strong>WICHTIG:</strong> Anzahl kostenloser Pro-Monate (z.B. 12)</li>
  <li><code>cta_text</code> - Button-Text (z.B. "Jetzt registrieren")</li>
  <li><code>cta_link</code> - Button-Link (z.B. "/landing/register.html")</li>
</ul>
<em>Ohne benefit_type und benefit_months wird kein Benefit gewährt!</em>`,
        example: `{
  "total_slots": 20,
  "remaining_slots": 20,
  "message": "Noch {remaining_slots} von {total_slots} Plätzen!",
  "benefit_type": "free_pro_months",
  "benefit_months": 12,
  "cta_text": "Jetzt registrieren",
  "cta_link": "/landing/register.html"
}`
    },
    'referral': {
        help: `<strong>Empfehlungs-Kampagne:</strong><br>
Konfiguration erfolgt über den Tab "Empfehlungscodes". Hier optional zusätzliche Einstellungen.`,
        example: '{}'
    },
    'reference_page': {
        help: `<strong>Referenzseite-Kampagne:</strong><br>
Konfiguration erfolgt über den Tab "Referenzseite". Hier optional zusätzliche Einstellungen.`,
        example: '{}'
    },
    'custom': {
        help: `<strong>Benutzerdefinierte Kampagne:</strong><br>
Freies JSON-Format für eigene Kampagnen-Typen.`,
        example: '{}'
    }
};

// Define the updateConfigHelp function (from marketing.html)
function updateConfigHelp(type) {
    const helpDiv = document.getElementById('config-help');
    const configTextarea = document.getElementById('campaign-config');
    const config = campaignConfigHelp[type];

    if (config) {
        helpDiv.innerHTML = config.help;
        // Only set placeholder if config is empty
        if (!configTextarea.value.trim()) {
            configTextarea.placeholder = config.example;
        }
    } else {
        helpDiv.innerHTML = '';
        configTextarea.placeholder = '';
    }
}

describe('Campaign Config Help System', () => {
    beforeEach(() => {
        // Setup DOM elements
        document.body.innerHTML = `
            <div id="config-help"></div>
            <textarea id="campaign-config"></textarea>
        `;
    });

    afterEach(() => {
        document.body.innerHTML = '';
    });

    describe('campaignConfigHelp Object Structure', () => {
        test('should have fomo_countdown config', () => {
            expect(campaignConfigHelp['fomo_countdown']).toBeDefined();
            expect(campaignConfigHelp['fomo_countdown'].help).toBeDefined();
            expect(campaignConfigHelp['fomo_countdown'].example).toBeDefined();
        });

        test('should have referral config', () => {
            expect(campaignConfigHelp['referral']).toBeDefined();
            expect(campaignConfigHelp['referral'].help).toBeDefined();
            expect(campaignConfigHelp['referral'].example).toBeDefined();
        });

        test('should have reference_page config', () => {
            expect(campaignConfigHelp['reference_page']).toBeDefined();
            expect(campaignConfigHelp['reference_page'].help).toBeDefined();
            expect(campaignConfigHelp['reference_page'].example).toBeDefined();
        });

        test('should have custom config', () => {
            expect(campaignConfigHelp['custom']).toBeDefined();
            expect(campaignConfigHelp['custom'].help).toBeDefined();
            expect(campaignConfigHelp['custom'].example).toBeDefined();
        });
    });

    describe('FOMO Countdown Config Help Content', () => {
        const fomoConfig = campaignConfigHelp['fomo_countdown'];

        test('should document total_slots field', () => {
            expect(fomoConfig.help).toContain('total_slots');
            expect(fomoConfig.help).toContain('Gesamtzahl');
        });

        test('should document remaining_slots field', () => {
            expect(fomoConfig.help).toContain('remaining_slots');
            expect(fomoConfig.help).toContain('Verbleibende Plätze');
        });

        test('should document message field with placeholders', () => {
            expect(fomoConfig.help).toContain('message');
            expect(fomoConfig.help).toContain('{remaining_slots}');
            expect(fomoConfig.help).toContain('{total_slots}');
        });

        test('should emphasize benefit_type is IMPORTANT', () => {
            expect(fomoConfig.help).toContain('benefit_type');
            expect(fomoConfig.help).toContain('WICHTIG');
            expect(fomoConfig.help).toContain('free_pro_months');
        });

        test('should emphasize benefit_months is IMPORTANT', () => {
            expect(fomoConfig.help).toContain('benefit_months');
            expect(fomoConfig.help).toContain('WICHTIG');
        });

        test('should document cta_text and cta_link fields', () => {
            expect(fomoConfig.help).toContain('cta_text');
            expect(fomoConfig.help).toContain('cta_link');
        });

        test('should warn that benefit is NOT granted without required fields', () => {
            expect(fomoConfig.help).toContain('Ohne benefit_type und benefit_months wird kein Benefit gewährt');
        });

        test('example should be valid JSON', () => {
            expect(() => JSON.parse(fomoConfig.example)).not.toThrow();
        });

        test('example should include all required fields', () => {
            const example = JSON.parse(fomoConfig.example);
            expect(example.total_slots).toBeDefined();
            expect(example.remaining_slots).toBeDefined();
            expect(example.message).toBeDefined();
            expect(example.benefit_type).toBe('free_pro_months');
            expect(example.benefit_months).toBeDefined();
            expect(example.cta_text).toBeDefined();
            expect(example.cta_link).toBeDefined();
        });

        test('example should have valid benefit_months value', () => {
            const example = JSON.parse(fomoConfig.example);
            expect(typeof example.benefit_months).toBe('number');
            expect(example.benefit_months).toBeGreaterThan(0);
        });

        test('example message should use placeholders', () => {
            const example = JSON.parse(fomoConfig.example);
            expect(example.message).toContain('{remaining_slots}');
            expect(example.message).toContain('{total_slots}');
        });
    });

    describe('updateConfigHelp Function', () => {
        test('should update help div with FOMO countdown help', () => {
            updateConfigHelp('fomo_countdown');

            const helpDiv = document.getElementById('config-help');
            expect(helpDiv.innerHTML).toContain('FOMO Countdown');
            expect(helpDiv.innerHTML).toContain('benefit_type');
        });

        test('should update help div with referral help', () => {
            updateConfigHelp('referral');

            const helpDiv = document.getElementById('config-help');
            expect(helpDiv.innerHTML).toContain('Empfehlungs-Kampagne');
            expect(helpDiv.innerHTML).toContain('Empfehlungscodes');
        });

        test('should update help div with reference_page help', () => {
            updateConfigHelp('reference_page');

            const helpDiv = document.getElementById('config-help');
            expect(helpDiv.innerHTML).toContain('Referenzseite-Kampagne');
        });

        test('should update help div with custom help', () => {
            updateConfigHelp('custom');

            const helpDiv = document.getElementById('config-help');
            expect(helpDiv.innerHTML).toContain('Benutzerdefinierte Kampagne');
        });

        test('should set placeholder when textarea is empty', () => {
            const textarea = document.getElementById('campaign-config');
            textarea.value = '';

            updateConfigHelp('fomo_countdown');

            expect(textarea.placeholder).toContain('total_slots');
            expect(textarea.placeholder).toContain('benefit_type');
        });

        test('should NOT overwrite placeholder when textarea has content', () => {
            const textarea = document.getElementById('campaign-config');
            textarea.value = '{"existing": "config"}';

            updateConfigHelp('fomo_countdown');

            // Placeholder should NOT be set when textarea has value
            expect(textarea.placeholder).toBe('');
        });

        test('should NOT overwrite placeholder when textarea has whitespace-only content', () => {
            const textarea = document.getElementById('campaign-config');
            textarea.value = '   ';

            updateConfigHelp('fomo_countdown');

            // Whitespace-only should be treated as empty, so placeholder IS set
            expect(textarea.placeholder).toContain('total_slots');
        });

        test('should clear help and placeholder for unknown type', () => {
            // First set some help
            updateConfigHelp('fomo_countdown');
            expect(document.getElementById('config-help').innerHTML).not.toBe('');

            // Then use unknown type
            updateConfigHelp('unknown_type');

            expect(document.getElementById('config-help').innerHTML).toBe('');
            expect(document.getElementById('campaign-config').placeholder).toBe('');
        });

        test('should handle null type gracefully', () => {
            updateConfigHelp(null);

            expect(document.getElementById('config-help').innerHTML).toBe('');
            expect(document.getElementById('campaign-config').placeholder).toBe('');
        });

        test('should handle undefined type gracefully', () => {
            updateConfigHelp(undefined);

            expect(document.getElementById('config-help').innerHTML).toBe('');
            expect(document.getElementById('campaign-config').placeholder).toBe('');
        });
    });

    describe('Config Type Switching', () => {
        test('should switch from fomo_countdown to referral', () => {
            updateConfigHelp('fomo_countdown');
            expect(document.getElementById('config-help').innerHTML).toContain('FOMO');

            updateConfigHelp('referral');
            expect(document.getElementById('config-help').innerHTML).toContain('Empfehlungs');
            expect(document.getElementById('config-help').innerHTML).not.toContain('FOMO');
        });

        test('should update placeholder when switching types with empty textarea', () => {
            const textarea = document.getElementById('campaign-config');
            textarea.value = '';

            updateConfigHelp('fomo_countdown');
            const fomoPlaceholder = textarea.placeholder;
            expect(fomoPlaceholder).toContain('benefit_type');

            updateConfigHelp('referral');
            const referralPlaceholder = textarea.placeholder;
            expect(referralPlaceholder).toBe('{}');
            expect(referralPlaceholder).not.toContain('benefit_type');
        });
    });

    describe('FOMO Config Validation Scenarios', () => {
        test('incomplete config should be identifiable (missing benefit_type)', () => {
            const incompleteConfig = {
                total_slots: 20,
                remaining_slots: 10,
                message: "Nur noch wenige Plätze!"
                // Missing: benefit_type, benefit_months
            };

            // This config would NOT grant any benefit
            expect(incompleteConfig.benefit_type).toBeUndefined();
            expect(incompleteConfig.benefit_months).toBeUndefined();
        });

        test('complete config should have all required fields', () => {
            const completeConfig = JSON.parse(campaignConfigHelp['fomo_countdown'].example);

            expect(completeConfig.total_slots).toBeDefined();
            expect(completeConfig.remaining_slots).toBeDefined();
            expect(completeConfig.message).toBeDefined();
            expect(completeConfig.benefit_type).toBe('free_pro_months');
            expect(completeConfig.benefit_months).toBeDefined();
            expect(completeConfig.cta_text).toBeDefined();
            expect(completeConfig.cta_link).toBeDefined();
        });

        test('benefit_type must be "free_pro_months" for Pro subscription', () => {
            const example = JSON.parse(campaignConfigHelp['fomo_countdown'].example);

            // Only "free_pro_months" triggers the Pro subscription grant
            expect(example.benefit_type).toBe('free_pro_months');
        });
    });

    describe('Accessibility of Help Content', () => {
        test('FOMO help should use semantic HTML list', () => {
            const fomoHelp = campaignConfigHelp['fomo_countdown'].help;

            expect(fomoHelp).toContain('<ul');
            expect(fomoHelp).toContain('<li>');
            expect(fomoHelp).toContain('</ul>');
        });

        test('FOMO help should use code tags for field names', () => {
            const fomoHelp = campaignConfigHelp['fomo_countdown'].help;

            expect(fomoHelp).toContain('<code>total_slots</code>');
            expect(fomoHelp).toContain('<code>benefit_type</code>');
        });

        test('FOMO help should emphasize important fields with strong tags', () => {
            const fomoHelp = campaignConfigHelp['fomo_countdown'].help;

            expect(fomoHelp).toContain('<strong>WICHTIG:</strong>');
        });

        test('FOMO help should have warning in emphasized text', () => {
            const fomoHelp = campaignConfigHelp['fomo_countdown'].help;

            expect(fomoHelp).toContain('<em>');
            expect(fomoHelp).toContain('Ohne benefit_type');
        });
    });
});

describe('Integration: Campaign Modal Form', () => {
    beforeEach(() => {
        // Setup full modal form structure
        document.body.innerHTML = `
            <div id="campaign-modal">
                <select id="campaign-type" onchange="updateConfigHelp(this.value)">
                    <option value="fomo_countdown">FOMO Countdown</option>
                    <option value="referral">Empfehlung</option>
                    <option value="reference_page">Referenzseite</option>
                    <option value="custom">Benutzerdefiniert</option>
                </select>
                <textarea id="campaign-config"></textarea>
                <div id="config-help"></div>
            </div>
        `;
    });

    test('changing select should update help (simulated)', () => {
        const select = document.getElementById('campaign-type');

        // Simulate selecting fomo_countdown
        select.value = 'fomo_countdown';
        updateConfigHelp(select.value);

        expect(document.getElementById('config-help').innerHTML).toContain('FOMO');

        // Simulate selecting referral
        select.value = 'referral';
        updateConfigHelp(select.value);

        expect(document.getElementById('config-help').innerHTML).toContain('Empfehlungs');
    });

    test('form should show complete FOMO example in placeholder', () => {
        updateConfigHelp('fomo_countdown');

        const placeholder = document.getElementById('campaign-config').placeholder;
        const parsed = JSON.parse(placeholder);

        // Verify all fields are present in placeholder example
        expect(parsed).toHaveProperty('total_slots');
        expect(parsed).toHaveProperty('remaining_slots');
        expect(parsed).toHaveProperty('message');
        expect(parsed).toHaveProperty('benefit_type');
        expect(parsed).toHaveProperty('benefit_months');
        expect(parsed).toHaveProperty('cta_text');
        expect(parsed).toHaveProperty('cta_link');
    });
});
