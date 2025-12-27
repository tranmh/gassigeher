/**
 * Debug script to investigate auth redirect issue
 */
const { chromium } = require('playwright');

async function debugAuth() {
    const browser = await chromium.launch({ headless: true });
    const context = await browser.newContext();
    const page = await context.newPage();

    // Capture console messages
    const consoleLogs = [];
    page.on('console', msg => {
        consoleLogs.push({ type: msg.type(), text: msg.text() });
    });

    // Capture page errors
    const pageErrors = [];
    page.on('pageerror', error => {
        pageErrors.push(error.message);
    });

    console.log('=== AUTH REDIRECT DEBUG ===\n');

    // Step 1: Go to homepage and clear token
    console.log('1. Navigating to homepage to clear localStorage...');
    await page.goto('http://demo.gassigeher.local:8080/');
    await page.waitForLoadState('networkidle');

    const tokenBefore = await page.evaluate(() => localStorage.getItem('gassigeher_token'));
    console.log('   Token before clear:', tokenBefore ? 'EXISTS' : 'null');

    await page.evaluate(() => localStorage.removeItem('gassigeher_token'));

    const tokenAfter = await page.evaluate(() => localStorage.getItem('gassigeher_token'));
    console.log('   Token after clear:', tokenAfter ? 'EXISTS' : 'null');

    // Step 2: Navigate to dashboard
    console.log('\n2. Navigating to dashboard.html...');
    consoleLogs.length = 0;
    pageErrors.length = 0;

    await page.goto('http://demo.gassigeher.local:8080/dashboard.html');

    // Wait for redirect or timeout
    console.log('   Waiting for potential redirect (5s)...');
    try {
        await page.waitForURL('**/login.html', { timeout: 5000 });
        console.log('   ✓ Redirected to login!');
    } catch {
        console.log('   ✗ No redirect to login within 5s');
    }

    console.log('\n   Final URL:', page.url());

    // Step 3: Check console logs
    console.log('\n3. Browser Console Logs:');
    if (consoleLogs.length === 0) {
        console.log('   (no console output)');
    } else {
        consoleLogs.forEach(log => {
            console.log(`   [${log.type}] ${log.text}`);
        });
    }

    // Step 4: Check for errors
    console.log('\n4. Page Errors:');
    if (pageErrors.length === 0) {
        console.log('   (no errors)');
    } else {
        pageErrors.forEach(err => {
            console.log('   ERROR:', err);
        });
    }

    // Step 5: Check if api object exists
    console.log('\n5. Checking API object...');
    const apiExists = await page.evaluate(() => typeof window.api !== 'undefined');
    console.log('   window.api exists:', apiExists);

    if (apiExists) {
        const isAuth = await page.evaluate(() => window.api.isAuthenticated());
        console.log('   api.isAuthenticated():', isAuth);

        const token = await page.evaluate(() => window.api.getToken());
        console.log('   api.getToken():', token ? 'exists' : 'null');
    }

    // Step 6: Check CSP header
    console.log('\n6. CSP Header Check...');
    const response = await page.goto('http://demo.gassigeher.local:8080/dashboard.html');
    const csp = response.headers()['content-security-policy'];
    if (csp) {
        const scriptSrc = csp.match(/script-src[^;]+/);
        console.log('   script-src:', scriptSrc ? scriptSrc[0] : 'not found');
    } else {
        console.log('   No CSP header found');
    }

    await browser.close();
    console.log('\n=== END DEBUG ===');
}

debugAuth().catch(console.error);
