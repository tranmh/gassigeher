// Landing Page JavaScript for Gassigeher SaaS

document.addEventListener('DOMContentLoaded', function() {
    // Initialize components
    initSlugChecker();
    initRegistrationForm();
    initFAQ();
});

// Slug availability checker
function initSlugChecker() {
    const slugInput = document.getElementById('slug');
    const slugStatus = document.getElementById('slug-status');

    if (!slugInput || !slugStatus) return;

    let debounceTimer;

    slugInput.addEventListener('input', function() {
        const slug = this.value.toLowerCase().replace(/[^a-z0-9-]/g, '');
        this.value = slug;

        clearTimeout(debounceTimer);

        if (slug.length < 3) {
            slugStatus.textContent = 'Mindestens 3 Zeichen erforderlich';
            slugStatus.className = '';
            return;
        }

        slugStatus.textContent = 'Wird überprüft...';
        slugStatus.className = '';

        debounceTimer = setTimeout(async () => {
            try {
                const response = await fetch(`/api/tenants/check-slug?slug=${encodeURIComponent(slug)}`);
                const data = await response.json();

                if (data.available) {
                    slugStatus.textContent = '✓ Verfügbar';
                    slugStatus.className = 'available';
                } else {
                    slugStatus.textContent = data.reason ? `✗ ${data.reason}` : '✗ Nicht verfügbar';
                    slugStatus.className = 'unavailable';
                }
            } catch (error) {
                slugStatus.textContent = 'Fehler bei der Überprüfung';
                slugStatus.className = 'unavailable';
            }
        }, 500);
    });
}

// Registration form handler
function initRegistrationForm() {
    const form = document.getElementById('register-form');
    const successMessage = document.getElementById('success-message');

    if (!form) return;

    form.addEventListener('submit', async function(e) {
        e.preventDefault();

        const submitBtn = form.querySelector('button[type="submit"]');
        const originalText = submitBtn.textContent;

        // Clear previous errors
        form.querySelectorAll('.form-group.error').forEach(g => g.classList.remove('error'));
        form.querySelectorAll('.error-message').forEach(e => e.remove());

        // Disable submit button
        submitBtn.disabled = true;
        submitBtn.classList.add('loading');
        submitBtn.textContent = 'Wird registriert...';

        // Collect form data
        const formData = new FormData(form);
        const data = {
            organization_name: formData.get('organization_name'),
            slug: formData.get('slug'),
            contact_email: formData.get('contact_email'),
            contact_phone: formData.get('contact_phone') || '',
            address: formData.get('address') || '',
            city: formData.get('city'),
            postal_code: formData.get('postal_code'),
            federal_state: formData.get('federal_state'),
            admin_first_name: formData.get('admin_first_name'),
            admin_last_name: formData.get('admin_last_name'),
            admin_email: formData.get('admin_email'),
            admin_password: formData.get('admin_password')
        };

        try {
            const response = await fetch('/api/tenants/register', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(data)
            });

            const result = await response.json();

            if (!response.ok) {
                throw new Error(result.error || 'Registrierung fehlgeschlagen');
            }

            // Success! Show success message
            form.style.display = 'none';
            successMessage.style.display = 'block';

            const loginLink = document.getElementById('login-link');
            if (loginLink && result.login_url) {
                loginLink.href = result.login_url;
                loginLink.textContent = `Anmelden bei ${result.slug}.gassigeher.org`;
            }

        } catch (error) {
            // Show error
            showFormError(form, error.message);
        } finally {
            submitBtn.disabled = false;
            submitBtn.classList.remove('loading');
            submitBtn.textContent = originalText;
        }
    });
}

// Show form error
function showFormError(form, message) {
    // Remove existing error message
    const existingError = form.querySelector('.form-error');
    if (existingError) existingError.remove();

    // Create error element
    const errorDiv = document.createElement('div');
    errorDiv.className = 'form-error';
    errorDiv.style.cssText = `
        background: #fef2f2;
        border: 1px solid #ef4444;
        color: #ef4444;
        padding: 1rem;
        margin-bottom: 1rem;
        border-radius: 6px;
        text-align: center;
    `;
    errorDiv.textContent = message;

    // Insert at top of form
    form.insertBefore(errorDiv, form.firstChild);

    // Scroll to error
    errorDiv.scrollIntoView({ behavior: 'smooth', block: 'center' });
}

// FAQ accordion
function initFAQ() {
    const faqItems = document.querySelectorAll('.faq-item');

    faqItems.forEach(item => {
        const question = item.querySelector('.faq-question');
        if (question) {
            question.addEventListener('click', () => {
                item.classList.toggle('open');
            });
        }
    });
}

// Smooth scroll for anchor links
document.querySelectorAll('a[href^="#"]').forEach(anchor => {
    anchor.addEventListener('click', function(e) {
        e.preventDefault();
        const target = document.querySelector(this.getAttribute('href'));
        if (target) {
            target.scrollIntoView({ behavior: 'smooth' });
        }
    });
});
