/**
 * Register page functionality
 * Extracted from inline script for CSP compliance (no unsafe-inline required)
 */

function showError(fieldId, message) {
    const errorEl = document.getElementById(`${fieldId}-error`);
    const inputEl = document.getElementById(fieldId);

    if (errorEl) errorEl.textContent = message;
    if (inputEl) inputEl.classList.add('error');
}

function showAlert(type, message) {
    const container = document.getElementById('alert-container');
    // Use textContent to prevent XSS - safe against script injection
    container.innerHTML = '';
    const alertDiv = document.createElement('div');
    alertDiv.className = `alert alert-${type}`;
    alertDiv.textContent = message;
    container.appendChild(alertDiv);
}

function showRegistrationSuccess(message, whatsappLink) {
    // Hide the form
    document.getElementById('register-form').style.display = 'none';

    // Validate WhatsApp link to prevent javascript: protocol injection
    let safeWhatsappLink = '';
    try {
        const url = new URL(whatsappLink);
        if (url.protocol === 'https:' &&
            (url.hostname === 'chat.whatsapp.com' ||
             url.hostname === 'wa.me' ||
             url.hostname === 'api.whatsapp.com')) {
            safeWhatsappLink = url.href;
        }
    } catch (e) {
        console.warn('Invalid WhatsApp link');
    }

    // Show success message with WhatsApp button
    const container = document.getElementById('alert-container');
    container.innerHTML = '';

    const wrapper = document.createElement('div');
    wrapper.style.cssText = 'text-align: center; padding: 30px 0;';

    const icon = document.createElement('div');
    icon.style.cssText = 'font-size: 4rem; margin-bottom: 20px;';
    icon.textContent = '✅';

    const title = document.createElement('h2');
    title.style.marginBottom = '15px';
    title.textContent = 'Registrierung erfolgreich!';

    const messageEl = document.createElement('p');
    messageEl.style.cssText = 'margin-bottom: 25px; color: #666;';
    messageEl.textContent = message; // Safe - uses textContent

    wrapper.appendChild(icon);
    wrapper.appendChild(title);
    wrapper.appendChild(messageEl);

    // Only show WhatsApp section if we have a valid link
    if (safeWhatsappLink) {
        const whatsappBox = document.createElement('div');
        whatsappBox.style.cssText = 'background: #dcf8c6; padding: 20px; border-radius: 8px; margin-bottom: 25px; border-left: 4px solid #25d366;';

        const whatsappTitle = document.createElement('p');
        whatsappTitle.style.cssText = 'margin: 0 0 15px 0; font-weight: 600;';
        whatsappTitle.innerHTML = '<span style="font-size: 1.5rem; margin-right: 8px;">💬</span>Unserer WhatsApp-Gruppe beitreten';

        const whatsappDesc = document.createElement('p');
        whatsappDesc.style.cssText = 'margin: 0 0 15px 0; font-size: 0.9rem; color: #666;';
        whatsappDesc.textContent = 'Bleibe auf dem Laufenden und tausche dich mit anderen Gassigehern aus!';

        whatsappBox.appendChild(whatsappTitle);
        whatsappBox.appendChild(whatsappDesc);

        const whatsappBtn = document.createElement('a');
        whatsappBtn.href = safeWhatsappLink;
        whatsappBtn.target = '_blank';
        whatsappBtn.rel = 'noopener noreferrer';
        whatsappBtn.style.cssText = 'display: inline-block; padding: 12px 30px; background-color: #25d366; color: white; text-decoration: none; border-radius: 6px; font-weight: 600;';
        whatsappBtn.textContent = 'WhatsApp-Gruppe beitreten';
        whatsappBox.appendChild(whatsappBtn);
        wrapper.appendChild(whatsappBox);
    }

    const loginBtn = document.createElement('a');
    loginBtn.href = '/login.html';
    loginBtn.className = 'btn';
    loginBtn.style.marginTop = '10px';
    loginBtn.textContent = 'Zur Anmeldung';
    wrapper.appendChild(loginBtn);

    container.appendChild(wrapper);
}

document.addEventListener('DOMContentLoaded', async () => {
    await window.i18n.load();

    const form = document.getElementById('register-form');
    const submitBtn = document.getElementById('submit-btn');

    form.addEventListener('submit', async (e) => {
        e.preventDefault();

        // Clear previous errors
        document.querySelectorAll('.form-error').forEach(el => el.textContent = '');
        document.querySelectorAll('input').forEach(el => el.classList.remove('error'));

        // Get form data
        const data = {
            first_name: document.getElementById('first-name').value.trim(),
            last_name: document.getElementById('last-name').value.trim(),
            email: document.getElementById('email').value.trim(),
            phone: document.getElementById('phone').value.trim(),
            password: document.getElementById('password').value,
            confirm_password: document.getElementById('confirm-password').value,
            accept_terms: document.getElementById('accept-terms').checked,
            accept_privacy: document.getElementById('accept-privacy').checked,
            registration_password: document.getElementById('registration-password').value.trim().toUpperCase(),
        };

        // Client-side validation
        let hasError = false;

        if (!data.first_name) {
            showError('first-name', 'Vorname ist erforderlich');
            hasError = true;
        }

        if (!data.last_name) {
            showError('last-name', 'Nachname ist erforderlich');
            hasError = true;
        }

        if (!data.email || !data.email.includes('@')) {
            showError('email', window.i18n.t('errors.invalid_email'));
            hasError = true;
        }

        if (!data.phone) {
            showError('phone', window.i18n.t('errors.required_field'));
            hasError = true;
        } else {
            // Validate phone number format
            const phonePattern = /^[\+]?[(]?[0-9]{1,4}[)]?[-\s\.]?[(]?[0-9]{1,4}[)]?[-\s\.]?[0-9]{1,9}$/;
            if (!phonePattern.test(data.phone)) {
                showError('phone', 'Bitte gib eine gültige Telefonnummer ein (z.B. 0123 456789 oder +49 123 456789)');
                hasError = true;
            }
        }

        if (data.password.length < 8) {
            showError('password', window.i18n.t('errors.password_too_short'));
            hasError = true;
        }

        if (data.password !== data.confirm_password) {
            showError('confirm-password', window.i18n.t('errors.password_mismatch'));
            hasError = true;
        }

        // Validate registration password
        if (!data.registration_password) {
            showError('registration-password', 'Registrierungspasswort ist erforderlich');
            hasError = true;
        } else if (!/^[a-zA-Z0-9]{8}$/.test(data.registration_password)) {
            showError('registration-password', 'Muss genau 8 alphanumerische Zeichen sein');
            hasError = true;
        }

        if (!data.accept_terms) {
            showError('terms', 'Du musst die AGB akzeptieren');
            hasError = true;
        }

        if (!data.accept_privacy) {
            showError('privacy', 'Du musst die Datenschutzerklärung akzeptieren');
            hasError = true;
        }

        if (hasError) return;

        // Submit
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<span data-i18n="common.loading">Laden...</span>';

        try {
            const response = await window.api.register(data);

            // Check if WhatsApp group is enabled and show success with WhatsApp button
            const whatsappData = await window.api.getWhatsAppSettings();
            if (whatsappData.enabled && whatsappData.link) {
                showRegistrationSuccess(response.message || 'Registrierung erfolgreich! Bitte prüfe deine E-Mails.', whatsappData.link);
            } else {
                showAlert('success', response.message || 'Registrierung erfolgreich! Bitte prüfe deine E-Mails.');
                setTimeout(() => {
                    window.location.href = '/login.html';
                }, 3000);
            }
        } catch (error) {
            showAlert('error', error.message || window.i18n.t('errors.unexpected_error'));
            submitBtn.disabled = false;
            submitBtn.innerHTML = '<span data-i18n="auth.register_button">Konto erstellen</span>';
        }
    });
});
