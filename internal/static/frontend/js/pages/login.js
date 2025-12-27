/**
 * Login page functionality
 * Extracted from inline script for CSP compliance (no unsafe-inline required)
 */

function showAlert(type, message) {
    const container = document.getElementById('alert-container');
    // Use textContent to prevent XSS - safe against script injection
    container.innerHTML = '';
    const alertDiv = document.createElement('div');
    alertDiv.className = `alert alert-${type}`;
    alertDiv.textContent = message;
    container.appendChild(alertDiv);
}

document.addEventListener('DOMContentLoaded', async () => {
    await window.i18n.load();

    const form = document.getElementById('login-form');
    const submitBtn = document.getElementById('submit-btn');

    form.addEventListener('submit', async (e) => {
        e.preventDefault();

        const email = document.getElementById('email').value.trim();
        const password = document.getElementById('password').value;

        submitBtn.disabled = true;
        submitBtn.innerHTML = '<span data-i18n="common.loading">Laden...</span>';

        try {
            const response = await window.api.login(email, password);

            // Check if user must change password
            if (response.must_change_password) {
                showAlert('info', 'Bitte ändere dein temporäres Passwort.');
                setTimeout(() => {
                    window.location.href = '/profile.html?change_password=true';
                }, 1500);
                return;
            }

            showAlert('success', 'Login erfolgreich!');

            // SaaS: Use redirect_to from server response
            const redirectTo = response.redirect_to || '/dashboard.html';
            setTimeout(() => {
                window.location.href = redirectTo;
            }, 1000);
        } catch (error) {
            showAlert('error', error.message || window.i18n.t('errors.unexpected_error'));
            submitBtn.disabled = false;
            submitBtn.innerHTML = '<span data-i18n="auth.login_button">Anmelden</span>';
        }
    });
});
