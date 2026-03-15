/**
 * Registration Privacy Checkbox Tests
 *
 * Verifies that register.html has a privacy/Datenschutz checkbox
 * and that the register JS sends accept_privacy in the request.
 *
 * Bug: Backend requires accept_privacy=true but frontend had no
 * privacy checkbox, making registration impossible.
 *
 * @jest-environment jsdom
 */

const fs = require('fs');
const path = require('path');

describe('Registration form privacy checkbox', () => {
  test('register.html contains a privacy/Datenschutz checkbox', () => {
    const html = fs.readFileSync(
      path.join(__dirname, '..', 'internal', 'static', 'frontend', 'register.html'),
      'utf-8'
    );

    // Must have an accept-privacy checkbox input
    expect(html).toContain('id="accept-privacy"');
    expect(html).toContain('type="checkbox"');

    // Must link to privacy.html
    expect(html).toMatch(/href=["']\/privacy\.html["']/);

    // Must mention Datenschutz
    expect(html).toMatch(/Datenschutz/i);
  });

  test('register.js reads accept_privacy checkbox and sends it in request data', () => {
    const js = fs.readFileSync(
      path.join(__dirname, '..', 'internal', 'static', 'frontend', 'js', 'pages', 'register.js'),
      'utf-8'
    );

    // JS must read the accept-privacy checkbox
    expect(js).toContain('accept-privacy');

    // JS must include accept_privacy in the data object sent to the API
    expect(js).toMatch(/accept_privacy\s*:/);
  });

  test('register.js validates that privacy must be accepted', () => {
    const js = fs.readFileSync(
      path.join(__dirname, '..', 'internal', 'static', 'frontend', 'js', 'pages', 'register.js'),
      'utf-8'
    );

    // JS must validate accept_privacy (show error if not checked)
    expect(js).toContain('accept_privacy');
    expect(js).toMatch(/privacy.*error|Datenschutz/i);
  });
});
