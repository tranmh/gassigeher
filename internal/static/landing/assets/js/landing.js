// Landing Page JavaScript for Gassigeher SaaS

document.addEventListener('DOMContentLoaded', function() {
    // Initialize components
    initSlugChecker();
    initPlanSelection();
    initBillingCycleToggle();
    initRegistrationForm();
    initFAQ();

    // Check URL params for pre-selected plan
    const urlParams = new URLSearchParams(window.location.search);
    const planParam = urlParams.get('plan');
    if (planParam === 'pro') {
        selectPlan('pro');
    }
});

// Plan selection state
let selectedPlan = 'free';
let billingCycle = 'monthly';
let registrationResult = null;

// Plan selection handler
function initPlanSelection() {
    const planCards = document.querySelectorAll('.plan-card');

    planCards.forEach(card => {
        card.addEventListener('click', function() {
            const plan = this.dataset.plan;
            selectPlan(plan);
        });
    });
}

function selectPlan(plan) {
    selectedPlan = plan;

    // Update hidden input
    const selectedPlanInput = document.getElementById('selected_plan');
    if (selectedPlanInput) selectedPlanInput.value = plan;

    // Update card styling
    document.querySelectorAll('.plan-card').forEach(card => {
        card.classList.remove('selected');
        if (card.dataset.plan === plan) {
            card.classList.add('selected');
            card.querySelector('input[type="radio"]').checked = true;
        }
    });

    // Show/hide billing toggle for Pro
    const billingContainer = document.getElementById('billing-toggle-container');
    if (billingContainer) {
        if (plan === 'pro') {
            billingContainer.classList.add('show');
        } else {
            billingContainer.classList.remove('show');
        }
    }

    // Update plan note
    const planNote = document.getElementById('plan-note');
    if (planNote) {
        if (plan === 'pro') {
            planNote.textContent = 'Nach der Registrierung werden Sie zur Zahlung weitergeleitet.';
            planNote.classList.add('pro-note');
        } else {
            planNote.textContent = 'Sie können jederzeit auf Pro upgraden, wenn Sie mehr als 10 Hunde verwalten möchten.';
            planNote.classList.remove('pro-note');
        }
    }

    // Update submit button text
    const submitBtn = document.getElementById('submit-btn');
    if (submitBtn) {
        if (plan === 'pro') {
            submitBtn.textContent = 'Registrieren & zur Zahlung';
        } else {
            submitBtn.textContent = 'Tierheim registrieren';
        }
    }
}

// Billing cycle toggle handler
function initBillingCycleToggle() {
    const billingButtons = document.querySelectorAll('.billing-toggle button');

    billingButtons.forEach(button => {
        button.addEventListener('click', function() {
            const cycle = this.dataset.cycle;
            setBillingCycle(cycle);
        });
    });
}

function setBillingCycle(cycle) {
    billingCycle = cycle;

    // Update hidden input
    const billingCycleInput = document.getElementById('billing_cycle');
    if (billingCycleInput) billingCycleInput.value = cycle;

    // Update button styling
    document.querySelectorAll('.billing-toggle button').forEach(btn => {
        btn.classList.remove('active');
        if (btn.dataset.cycle === cycle) {
            btn.classList.add('active');
        }
    });

    // Update price display
    const proPriceDisplay = document.getElementById('pro-price-display');
    const proPriceNote = document.getElementById('pro-price-note');

    if (cycle === 'yearly') {
        if (proPriceDisplay) proPriceDisplay.innerHTML = '290 <span>EUR/Jahr</span>';
        if (proPriceNote) proPriceNote.innerHTML = '<strong>24,17 EUR/Monat</strong> - 2 Monate gratis';
    } else {
        if (proPriceDisplay) proPriceDisplay.innerHTML = '29 <span>EUR/Monat</span>';
        if (proPriceNote) proPriceNote.innerHTML = '<strong>29 EUR/Monat</strong> - Monatlich kündbar';
    }
}

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
    const successMessageFree = document.getElementById('success-message-free');
    const successMessagePro = document.getElementById('success-message-pro');

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
            admin_password: formData.get('admin_password'),
            plan: selectedPlan,
            billing_cycle: billingCycle
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

            // Store registration result for checkout (use sessionStorage, cleared on tab close)
            // Note: Password storage is temporary and cleared immediately after checkout
            sessionStorage.setItem('gassigeher_checkout_data', JSON.stringify({
                login_url: result.login_url,
                slug: result.slug,
                adminEmail: data.admin_email,
                adminPassword: data.admin_password
            }));
            registrationResult = result;

            // Hide form
            form.style.display = 'none';

            // Show appropriate success message based on plan
            if (selectedPlan === 'pro') {
                // Show Pro success message with checkout button
                if (successMessagePro) {
                    successMessagePro.style.display = 'block';

                    // Update checkout info
                    const checkoutBillingCycle = document.getElementById('checkout-billing-cycle');
                    const checkoutPrice = document.getElementById('checkout-price');

                    if (checkoutBillingCycle) {
                        checkoutBillingCycle.textContent = billingCycle === 'yearly' ? 'jährlich' : 'monatlich';
                    }
                    if (checkoutPrice) {
                        checkoutPrice.textContent = billingCycle === 'yearly' ? '290 EUR/Jahr' : '29 EUR/Monat';
                    }

                    // Setup checkout button
                    const checkoutBtn = document.getElementById('checkout-btn');
                    if (checkoutBtn) {
                        checkoutBtn.addEventListener('click', () => initiateProCheckout(result));
                    }

                    // Setup skip payment link
                    const skipLink = document.getElementById('skip-payment-link');
                    if (skipLink) {
                        skipLink.addEventListener('click', (e) => {
                            e.preventDefault();
                            // Redirect to login page
                            window.location.href = result.login_url;
                        });
                    }
                }
            } else {
                // Show Free success message
                if (successMessageFree) {
                    successMessageFree.style.display = 'block';

                    const loginLink = document.getElementById('login-link-free');
                    if (loginLink && result.login_url) {
                        loginLink.href = result.login_url;
                        loginLink.textContent = `Anmelden bei ${result.slug}.gassigeher.org`;
                    }
                }
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

// Initiate Pro checkout after registration
async function initiateProCheckout(registrationResult) {
    const checkoutBtn = document.getElementById('checkout-btn');
    if (checkoutBtn) {
        checkoutBtn.disabled = true;
        checkoutBtn.textContent = 'Wird geladen...';
    }

    // Get checkout data from sessionStorage
    const checkoutDataStr = sessionStorage.getItem('gassigeher_checkout_data');
    if (!checkoutDataStr) {
        showFormError(document.getElementById('success-message-pro'),
            'Sitzung abgelaufen. Bitte registrieren Sie sich erneut.');
        return;
    }

    let checkoutData;
    try {
        checkoutData = JSON.parse(checkoutDataStr);
    } catch (e) {
        sessionStorage.removeItem('gassigeher_checkout_data');
        showFormError(document.getElementById('success-message-pro'),
            'Sitzung abgelaufen. Bitte registrieren Sie sich erneut.');
        return;
    }

    try {
        // First, we need to authenticate to get a token
        // The tenant was just created, so we need to login to the tenant subdomain
        const baseUrl = checkoutData.login_url.replace(/\/login\/?$/, '');
        const loginResponse = await fetch(`${baseUrl}/api/auth/login`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                email: checkoutData.adminEmail,
                password: checkoutData.adminPassword
            })
        });

        // Clear sensitive data immediately after use
        sessionStorage.removeItem('gassigeher_checkout_data');

        if (!loginResponse.ok) {
            throw new Error('Login fehlgeschlagen');
        }

        const loginData = await loginResponse.json();
        const token = loginData.token;

        // Now create checkout session with the token
        const billingResponse = await fetch(`${baseUrl}/api/billing/checkout`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            },
            body: JSON.stringify({
                plan_slug: 'pro',
                billing_cycle: billingCycle
            })
        });

        if (!billingResponse.ok) {
            const errorData = await billingResponse.json();
            throw new Error(errorData.error || 'Checkout fehlgeschlagen');
        }

        const billingData = await billingResponse.json();

        // Redirect to Stripe checkout
        if (billingData.checkout_url) {
            window.location.href = billingData.checkout_url;
        } else {
            throw new Error('Keine Checkout-URL erhalten');
        }

    } catch (error) {
        console.error('Checkout error:', error);
        // Clear sensitive data on error too
        sessionStorage.removeItem('gassigeher_checkout_data');
        showFormError(document.getElementById('success-message-pro'),
            'Checkout konnte nicht gestartet werden. Sie können später im Dashboard upgraden.');

        if (checkoutBtn) {
            checkoutBtn.disabled = false;
            checkoutBtn.textContent = 'Erneut versuchen';
        }
    }
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
