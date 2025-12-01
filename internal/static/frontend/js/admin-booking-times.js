(async function() {
    // Check authentication and admin status
    if (!api.isAuthenticated()) {
        window.location.href = '/login.html';
        return;
    }

    try {
        const userData = await api.getMe();
        if (!userData.is_admin) {
            alert('Zugriff verweigert: Diese Seite ist nur für Administratoren zugänglich.');
            window.location.href = '/dashboard.html';
            return;
        }
    } catch (error) {
        console.error('Failed to verify admin status:', error);
        window.location.href = '/dashboard.html';
        return;
    }

    await window.i18n.load();
    window.i18n.updateElement(document.body);

    // Load settings
    async function loadSettings() {
        try {
            const settingsArray = await api.getSettings();
            const settings = {};

            settingsArray.forEach(setting => {
                settings[setting.key] = setting.value;
            });

            document.getElementById('morning-approval-toggle').checked =
                settings.morning_walk_requires_approval === 'true';
            document.getElementById('use-feiertage-api').checked =
                settings.use_feiertage_api === 'true';
        } catch (error) {
            console.error('Failed to load settings:', error);
            showAlert('error', 'Fehler beim Laden der Einstellungen');
        }
    }

    // Save settings
    document.getElementById('save-settings-btn').addEventListener('click', async () => {
        const morningApproval = document.getElementById('morning-approval-toggle').checked;
        const useFeiertageAPI = document.getElementById('use-feiertage-api').checked;

        try {
            await api.updateSetting('morning_walk_requires_approval', morningApproval.toString());
            await api.updateSetting('use_feiertage_api', useFeiertageAPI.toString());
            showAlert('success', 'Einstellungen gespeichert!');
        } catch (error) {
            showAlert('error', error.message || 'Fehler beim Speichern der Einstellungen');
        }
    });

    // Load time rules
    async function loadTimeRules() {
        try {
            const rules = await api.getBookingTimeRules();

            // Populate weekday rules
            const weekdayRules = rules.weekday || [];
            const weekdayTable = document.getElementById('weekday-rules');
            weekdayTable.innerHTML = '';

            weekdayRules.forEach(rule => {
                const row = createRuleRow(rule);
                weekdayTable.appendChild(row);
            });

            // Populate weekend rules
            const weekendRules = rules.weekend || [];
            const weekendTable = document.getElementById('weekend-rules');
            weekendTable.innerHTML = '';

            weekendRules.forEach(rule => {
                const row = createRuleRow(rule);
                weekendTable.appendChild(row);
            });
        } catch (error) {
            console.error('Failed to load rules:', error);
            showAlert('error', 'Fehler beim Laden der Zeitregeln');
        }
    }

    // Create rule table row
    function createRuleRow(rule) {
        const tr = document.createElement('tr');

        tr.innerHTML = `
            <td>${rule.rule_name}</td>
            <td><input type="time" value="${rule.start_time}" data-field="start"></td>
            <td><input type="time" value="${rule.end_time}" data-field="end"></td>
            <td>
                <select data-field="blocked">
                    <option value="0" ${!rule.is_blocked ? 'selected' : ''}>Erlaubt</option>
                    <option value="1" ${rule.is_blocked ? 'selected' : ''}>Gesperrt</option>
                </select>
            </td>
            <td>
                <button class="btn-save" data-id="${rule.id}">Speichern</button>
                <button class="btn-delete" data-id="${rule.id}">Löschen</button>
            </td>
        `;

        // Save handler
        tr.querySelector('.btn-save').addEventListener('click', async () => {
            const updatedRule = {
                id: rule.id,
                day_type: rule.day_type,
                rule_name: rule.rule_name,
                start_time: tr.querySelector('[data-field="start"]').value,
                end_time: tr.querySelector('[data-field="end"]').value,
                is_blocked: tr.querySelector('[data-field="blocked"]').value === '1'
            };

            try {
                await api.updateBookingTimeRules([updatedRule]);
                showAlert('success', 'Zeitfenster gespeichert!');
                // Update the rule object for next save
                rule.start_time = updatedRule.start_time;
                rule.end_time = updatedRule.end_time;
                rule.is_blocked = updatedRule.is_blocked;
            } catch (error) {
                showAlert('error', error.message || 'Fehler beim Speichern');
            }
        });

        // Delete handler
        tr.querySelector('.btn-delete').addEventListener('click', async () => {
            if (!confirm('Zeitfenster wirklich löschen?')) return;

            try {
                await api.deleteBookingTimeRule(rule.id);
                tr.remove();
                showAlert('success', 'Zeitfenster gelöscht!');
            } catch (error) {
                showAlert('error', error.message || 'Fehler beim Löschen');
            }
        });

        return tr;
    }

    // Add rule buttons
    document.getElementById('add-weekday-rule-btn').addEventListener('click', async () => {
        const ruleName = prompt('Name des Zeitfensters:');
        if (!ruleName) return;

        const startTime = prompt('Startzeit (HH:MM):', '09:00');
        if (!startTime) return;

        const endTime = prompt('Endzeit (HH:MM):', '12:00');
        if (!endTime) return;

        const isBlocked = confirm('Ist dieses Zeitfenster gesperrt?');

        try {
            await api.createBookingTimeRule({
                day_type: 'weekday',
                rule_name: ruleName,
                start_time: startTime,
                end_time: endTime,
                is_blocked: isBlocked
            });
            showAlert('success', 'Zeitfenster hinzugefügt!');
            loadTimeRules();
        } catch (error) {
            showAlert('error', error.message || 'Fehler beim Hinzufügen');
        }
    });

    document.getElementById('add-weekend-rule-btn').addEventListener('click', async () => {
        const ruleName = prompt('Name des Zeitfensters:');
        if (!ruleName) return;

        const startTime = prompt('Startzeit (HH:MM):', '09:00');
        if (!startTime) return;

        const endTime = prompt('Endzeit (HH:MM):', '12:00');
        if (!endTime) return;

        const isBlocked = confirm('Ist dieses Zeitfenster gesperrt?');

        try {
            await api.createBookingTimeRule({
                day_type: 'weekend',
                rule_name: ruleName,
                start_time: startTime,
                end_time: endTime,
                is_blocked: isBlocked
            });
            showAlert('success', 'Zeitfenster hinzugefügt!');
            loadTimeRules();
        } catch (error) {
            showAlert('error', error.message || 'Fehler beim Hinzufügen');
        }
    });

    // Load holidays
    async function loadHolidays(year) {
        try {
            const holidays = await api.getHolidays(year);
            const table = document.getElementById('holidays-table');
            table.innerHTML = '';

            if (holidays.length === 0) {
                table.innerHTML = '<tr><td colspan="5" style="text-align: center;">Keine Feiertage gefunden</td></tr>';
                return;
            }

            holidays.forEach(holiday => {
                const row = createHolidayRow(holiday);
                table.appendChild(row);
            });
        } catch (error) {
            console.error('Failed to load holidays:', error);
            showAlert('error', 'Fehler beim Laden der Feiertage');
        }
    }

    // Create holiday table row
    function createHolidayRow(holiday) {
        const tr = document.createElement('tr');

        tr.innerHTML = `
            <td>${holiday.date}</td>
            <td>${holiday.name}</td>
            <td>${holiday.source === 'api' ? 'Automatisch' : 'Manuell'}</td>
            <td>
                <label>
                    <input type="checkbox" ${holiday.is_active ? 'checked' : ''}
                           data-id="${holiday.id}" class="holiday-active-toggle">
                    Aktiv
                </label>
            </td>
            <td>
                <button class="btn-delete-holiday" data-id="${holiday.id}">Löschen</button>
            </td>
        `;

        // Toggle active status
        tr.querySelector('.holiday-active-toggle').addEventListener('change', async (e) => {
            try {
                await api.updateHoliday(holiday.id, {
                    name: holiday.name,
                    is_active: e.target.checked
                });
                showAlert('success', 'Feiertag aktualisiert');
            } catch (error) {
                showAlert('error', error.message || 'Fehler beim Aktualisieren');
                e.target.checked = !e.target.checked;
            }
        });

        // Delete handler
        tr.querySelector('.btn-delete-holiday').addEventListener('click', async () => {
            if (!confirm('Feiertag wirklich löschen?')) return;

            try {
                await api.deleteHoliday(holiday.id);
                tr.remove();
                showAlert('success', 'Feiertag gelöscht');
            } catch (error) {
                showAlert('error', error.message || 'Fehler beim Löschen');
            }
        });

        return tr;
    }

    // Add holiday button
    document.getElementById('add-holiday-btn').addEventListener('click', async () => {
        const date = prompt('Datum (YYYY-MM-DD):', '2025-12-25');
        if (!date) return;

        const name = prompt('Name des Feiertags:', 'Sondertag');
        if (!name) return;

        try {
            await api.createHoliday({
                date: date,
                name: name,
                source: 'admin',
                is_active: true
            });
            showAlert('success', 'Feiertag hinzugefügt!');
            const year = document.getElementById('holiday-year-select').value;
            loadHolidays(parseInt(year));
        } catch (error) {
            showAlert('error', error.message || 'Fehler beim Hinzufügen');
        }
    });

    // Tab switching
    document.querySelectorAll('.tab-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            // Remove active class from all tabs
            document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
            document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));

            // Add active to clicked tab
            btn.classList.add('active');
            const tabId = btn.dataset.tab + '-tab';
            document.getElementById(tabId).classList.add('active');
        });
    });

    // Load holidays button
    document.getElementById('load-holidays-btn').addEventListener('click', () => {
        const year = document.getElementById('holiday-year-select').value;
        loadHolidays(parseInt(year));
    });

    // Show alert function
    function showAlert(type, message) {
        const container = document.getElementById('alert-container');
        container.innerHTML = `<div class="alert alert-${type}">${message}</div>`;
        setTimeout(() => container.innerHTML = '', 5000);
    }

    // Initialize - set default year to current year
    const currentYear = new Date().getFullYear();
    const yearSelect = document.getElementById('holiday-year-select');
    yearSelect.value = currentYear;

    // Load initial data
    loadSettings();
    loadTimeRules();
    loadHolidays(currentYear);
})();
