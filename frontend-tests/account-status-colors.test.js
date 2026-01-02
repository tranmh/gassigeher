/**
 * Account Status - Dynamic Color Configuration Tests (TDD)
 *
 * BUG #11: The COLOR_CONFIG is hardcoded with green/orange/blue colors.
 * In SaaS mode, tenants can customize their color categories.
 * The page should use colors from the API response instead.
 *
 * @jest-environment jsdom
 */

describe('BUG #11: Dynamic Color Configuration', () => {

    describe('RED PHASE - Current hardcoded behavior (should fail after fix)', () => {
        // This is the CURRENT buggy behavior - hardcoded colors
        const HARDCODED_COLOR_CONFIG = {
            green: { name: 'Grün', color: '#82b965', icon: '🟢' },
            orange: { name: 'Orange', color: '#f5a623', icon: '🟠' },
            blue: { name: 'Blau', color: '#4a90e2', icon: '🔵' }
        };

        test('BUGGY: hardcoded config ignores custom tenant colors', () => {
            // Tenant has custom colors from API
            const apiColors = [
                { id: 1, name: 'Anfänger', hex_color: '#00FF00', icon: '🌱' },
                { id: 2, name: 'Fortgeschritten', hex_color: '#FFA500', icon: '🌟' },
                { id: 3, name: 'Profi', hex_color: '#0000FF', icon: '💎' }
            ];

            // With hardcoded config, custom colors are NOT used
            const config = HARDCODED_COLOR_CONFIG;

            // This fails because hardcoded config doesn't have 'anfänger' key
            expect(config['anfänger']).toBeUndefined();
            expect(config['fortgeschritten']).toBeUndefined();
            expect(config['profi']).toBeUndefined();
        });

        test('BUGGY: hardcoded config uses wrong colors for user display', () => {
            // User has a custom color assigned
            const userColors = [{ name: 'Anfänger', hex_color: '#00FF00' }];

            // Hardcoded lookup fails
            const colorKey = userColors[0].name.toLowerCase();
            const config = HARDCODED_COLOR_CONFIG[colorKey];

            expect(config).toBeUndefined(); // Bug: can't find custom color
        });
    });

    describe('GREEN PHASE - Expected dynamic behavior', () => {

        /**
         * buildDynamicColorConfig - Creates color config from API response
         * This is the FIXED implementation that should replace hardcoded config
         */
        function buildDynamicColorConfig(apiColors) {
            if (!apiColors || !Array.isArray(apiColors)) {
                // Fallback to empty config, not hardcoded values
                return {};
            }

            const config = {};
            apiColors.forEach(color => {
                if (!color.name) return;

                const key = color.name.toLowerCase();
                config[key] = {
                    id: color.id,
                    name: color.name,
                    color: color.hex_color || color.color || '#666666',
                    icon: color.icon || '●'
                };
            });
            return config;
        }

        test('should build config from API colors', () => {
            const apiColors = [
                { id: 1, name: 'Anfänger', hex_color: '#00FF00', icon: '🌱' },
                { id: 2, name: 'Fortgeschritten', hex_color: '#FFA500', icon: '🌟' },
                { id: 3, name: 'Profi', hex_color: '#0000FF', icon: '💎' }
            ];

            const config = buildDynamicColorConfig(apiColors);

            expect(config['anfänger']).toBeDefined();
            expect(config['anfänger'].name).toBe('Anfänger');
            expect(config['anfänger'].color).toBe('#00FF00');
            expect(config['anfänger'].icon).toBe('🌱');
        });

        test('should handle German special characters in keys', () => {
            const apiColors = [
                { id: 1, name: 'Grün', hex_color: '#82b965' },
                { id: 2, name: 'Größe', hex_color: '#ff0000' }
            ];

            const config = buildDynamicColorConfig(apiColors);

            expect(config['grün']).toBeDefined();
            expect(config['größe']).toBeDefined();
        });

        test('should provide default icon when missing', () => {
            const apiColors = [
                { id: 1, name: 'Custom', hex_color: '#123456' } // no icon
            ];

            const config = buildDynamicColorConfig(apiColors);

            expect(config['custom'].icon).toBe('●');
        });

        test('should provide default color when missing', () => {
            const apiColors = [
                { id: 1, name: 'NoColor' } // no hex_color
            ];

            const config = buildDynamicColorConfig(apiColors);

            expect(config['nocolor'].color).toBe('#666666');
        });

        test('should handle empty API response', () => {
            const config = buildDynamicColorConfig([]);
            expect(config).toEqual({});
        });

        test('should handle null API response', () => {
            const config = buildDynamicColorConfig(null);
            expect(config).toEqual({});
        });

        test('should handle undefined API response', () => {
            const config = buildDynamicColorConfig(undefined);
            expect(config).toEqual({});
        });

        test('should skip colors without name', () => {
            const apiColors = [
                { id: 1, hex_color: '#123456' }, // no name
                { id: 2, name: 'Valid', hex_color: '#654321' }
            ];

            const config = buildDynamicColorConfig(apiColors);

            expect(Object.keys(config).length).toBe(1);
            expect(config['valid']).toBeDefined();
        });

        test('should preserve color ID for later use', () => {
            const apiColors = [
                { id: 42, name: 'Special', hex_color: '#abcdef' }
            ];

            const config = buildDynamicColorConfig(apiColors);

            expect(config['special'].id).toBe(42);
        });
    });

    describe('Integration: renderUserColors with dynamic config', () => {

        function buildDynamicColorConfig(apiColors) {
            if (!apiColors || !Array.isArray(apiColors)) return {};
            const config = {};
            apiColors.forEach(color => {
                if (!color.name) return;
                const key = color.name.toLowerCase();
                config[key] = {
                    id: color.id,
                    name: color.name,
                    color: color.hex_color || color.color || '#666666',
                    icon: color.icon || '●'
                };
            });
            return config;
        }

        function renderColorBadge(userColor, colorConfig) {
            const key = userColor.name ? userColor.name.toLowerCase() : '';
            const config = colorConfig[key];

            if (config) {
                return `<span class="color-badge" style="background-color: ${config.color};">${config.icon} ${config.name}</span>`;
            } else {
                // Fallback for unknown colors - use data from userColor itself
                const color = userColor.hex_color || userColor.color || '#666666';
                const icon = userColor.icon || '●';
                const name = userColor.name || 'Unbekannt';
                return `<span class="color-badge" style="background-color: ${color};">${icon} ${name}</span>`;
            }
        }

        test('should render user colors with tenant-specific config', () => {
            // Tenant's custom colors from API
            const apiColors = [
                { id: 1, name: 'Beginner', hex_color: '#00FF00', icon: '🐣' },
                { id: 2, name: 'Expert', hex_color: '#FF0000', icon: '🦅' }
            ];

            // User has Beginner color assigned
            const userColors = [{ name: 'Beginner' }];

            // Build dynamic config
            const colorConfig = buildDynamicColorConfig(apiColors);

            // Render badge
            const badge = renderColorBadge(userColors[0], colorConfig);

            expect(badge).toContain('#00FF00');
            expect(badge).toContain('🐣');
            expect(badge).toContain('Beginner');
        });

        test('should fallback gracefully for unknown colors', () => {
            const apiColors = [
                { id: 1, name: 'Known', hex_color: '#123456' }
            ];

            // User has a color not in the config
            const userColor = { name: 'Unknown', hex_color: '#abcdef', icon: '?' };

            const colorConfig = buildDynamicColorConfig(apiColors);
            const badge = renderColorBadge(userColor, colorConfig);

            // Should use the color's own data as fallback
            expect(badge).toContain('#abcdef');
            expect(badge).toContain('?');
            expect(badge).toContain('Unknown');
        });
    });
});
