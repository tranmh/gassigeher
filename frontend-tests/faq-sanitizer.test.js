/**
 * Unit tests for internal/static/shared/js/faq-sanitizer.js (the real file,
 * not a copy): sanitizeFAQAnswer must strip XSS vectors from FAQ answers
 * while keeping the formatting markup the FAQ uses.
 *
 * @jest-environment jsdom
 */
const { readStatic, loadFaqSanitizer, buildFaq } = require('./__tests__/aqaa-doubles');

/** Parse sanitized output so assertions look at the DOM, not at string shapes. */
function parse(html) {
    const container = document.createElement('div');
    container.innerHTML = html;
    return container;
}

describe('sanitizeFAQAnswer', () => {
    const sanitize = loadFaqSanitizer();

    describe('invalid input', () => {
        it.each([
            ['null', null],
            ['undefined', undefined],
            ['empty string', ''],
            ['number', 42],
            ['object', { answer: '<p>x</p>' }],
            ['array', ['<p>x</p>']],
            ['boolean', true],
        ])('returns an empty string for %s', (_label, input) => {
            expect(sanitize(input)).toBe('');
        });
    });

    describe('safe content is preserved', () => {
        it('returns plain text unchanged', () => {
            expect(sanitize('Einfacher Text ohne HTML')).toBe('Einfacher Text ohne HTML');
        });

        it('keeps the FAQ formatting tags', () => {
            const html = '<p><strong>a</strong> <em>b</em> <b>c</b> <i>d</i><br></p>'
                + '<ul><li>1</li></ul><ol><li>2</li></ol>';
            expect(sanitize(html)).toBe(html);
        });

        it('keeps a typical FAQ answer exactly', () => {
            const faq = buildFaq({ answer: '<p>Gehe zu <a href="/dogs.html">Hunde</a> und wähle aus.</p>' });
            expect(sanitize(faq.answer)).toBe(faq.answer);
        });

        it.each([
            'https://example.com/hilfe',
            'http://example.com',
            '/help.html',
            'mailto:info@example.com',
        ])('keeps the safe href %s', (href) => {
            const out = parse(sanitize(`<a href="${href}">Link</a>`));
            expect(out.querySelector('a').getAttribute('href')).toBe(href);
        });

        it('keeps German umlauts', () => {
            expect(sanitize('<p>Größe & Übung</p>')).toBe('<p>Größe &amp; Übung</p>');
        });

        it('keeps harmless style attributes', () => {
            const out = parse(sanitize('<p style="color: red">x</p>'));
            expect(out.querySelector('p').getAttribute('style')).toBe('color: red');
        });

        it('keeps non-event attributes such as class and title', () => {
            const out = parse(sanitize('<p class="hint" title="Tipp">x</p>'));
            const p = out.querySelector('p');
            expect(p.getAttribute('class')).toBe('hint');
            expect(p.getAttribute('title')).toBe('Tipp');
        });
    });

    describe('dangerous elements are removed', () => {
        it.each([
            ['script', '<script>alert(1)</script>'],
            ['iframe', '<iframe src="https://evil.example"></iframe>'],
            ['object', '<object data="x.swf"></object>'],
            ['embed', '<embed src="x.swf">'],
            ['form', '<form action="/steal"></form>'],
            ['input', '<input type="text">'],
            ['button', '<button>x</button>'],
            ['textarea', '<textarea>x</textarea>'],
            ['select', '<select><option>x</option></select>'],
            ['meta', '<meta http-equiv="refresh" content="0;url=https://evil.example">'],
            ['link', '<link rel="stylesheet" href="https://evil.example/x.css">'],
            ['style', '<style>body{display:none}</style>'],
            ['svg', '<svg><circle r="1"></circle></svg>'],
            ['math', '<math><mi>x</mi></math>'],
            ['base', '<base href="https://evil.example/">'],
            ['applet', '<applet code="X.class"></applet>'],
        ])('removes <%s> while keeping surrounding text', (tag, payload) => {
            const out = sanitize(`<p>vor</p>${payload}<p>nach</p>`);
            expect(parse(out).querySelector(tag)).toBeNull();
            expect(out).toBe('<p>vor</p><p>nach</p>');
        });

        it('removes dangerous elements nested inside safe ones', () => {
            const out = sanitize('<ul><li>ok<script>alert(1)</script></li></ul>');
            expect(out).toBe('<ul><li>ok</li></ul>');
        });

        it('removes every occurrence, not only the first', () => {
            const out = sanitize('<script>a()</script><p>x</p><script>b()</script>');
            expect(out).toBe('<p>x</p>');
        });

        it('removes tags regardless of their case', () => {
            expect(sanitize('<SCRIPT>alert(1)</SCRIPT><p>x</p>')).toBe('<p>x</p>');
        });
    });

    describe('event handlers are removed', () => {
        it.each(['onclick', 'onmouseover', 'onfocus', 'ONLOAD', 'OnError'])(
            'removes %s',
            (attr) => {
                const out = parse(sanitize(`<p ${attr}="alert(1)">x</p>`));
                const p = out.querySelector('p');
                expect(p.attributes.length).toBe(0);
                expect(p.textContent).toBe('x');
            },
        );

        it('removes all handlers from one element and keeps the rest', () => {
            const out = parse(sanitize('<a href="/x" onclick="a()" onmouseover="b()">x</a>'));
            const a = out.querySelector('a');
            expect(a.getAttribute('href')).toBe('/x');
            expect([...a.attributes].map((attr) => attr.name)).toEqual(['href']);
        });
    });

    describe('dangerous href/src protocols are removed', () => {
        it.each([
            'javascript:alert(1)',
            'JavaScript:alert(1)',
            '  javascript:alert(1)',
            'data:text/html,<script>alert(1)</script>',
            'DATA:text/html;base64,PHNjcmlwdD4=',
            'vbscript:msgbox(1)',
            '&#106;avascript:alert(1)',
        ])('drops href="%s" but keeps the link text', (href) => {
            const out = parse(sanitize(`<a href="${href}">Klick</a>`));
            const a = out.querySelector('a');
            expect(a.hasAttribute('href')).toBe(false);
            expect(a.textContent).toBe('Klick');
        });
    });

    describe('dangerous styles are removed', () => {
        it.each([
            'background:url(javascript:alert(1))',
            'width: expression(alert(1))',
            'background-image: URL(https://tracker.example/p.gif)',
            'color: javascript',
        ])('drops style="%s"', (style) => {
            const out = parse(sanitize(`<p style="${style}">x</p>`));
            expect(out.querySelector('p').hasAttribute('style')).toBe(false);
        });
    });

    describe('img tags', () => {
        it.each([
            'https://example.com/a.png',
            'http://example.com/a.png',
            '/uploads/a.png',
            './a.png',
        ])('keeps an image with src %s', (src) => {
            const out = parse(sanitize(`<img src="${src}" alt="Hund">`));
            const img = out.querySelector('img');
            expect(img).not.toBeNull();
            expect(img.getAttribute('src')).toBe(src);
            expect(img.getAttribute('alt')).toBe('Hund');
        });

        it.each([
            ['relative without slash', 'x'],
            ['javascript: url', 'javascript:alert(1)'],
            ['data: url', 'data:image/png;base64,AAAA'],
            ['another scheme','ftp://example.com/a.png'],
        ])('removes an image with %s', (_label, src) => {
            const out = sanitize(`<p>a</p><img src="${src}"><p>b</p>`);
            expect(out).toBe('<p>a</p><p>b</p>');
        });

        it('removes an image without src', () => {
            expect(sanitize('<p>a</p><img alt="x">')).toBe('<p>a</p>');
        });

        it('removes the classic onerror payload entirely', () => {
            const out = sanitize('<img src=x onerror=alert("XSS")>');
            expect(out).toBe('');
        });

        it('keeps a safe image but strips its event handler', () => {
            const out = parse(sanitize('<img src="/a.png" onerror="alert(1)">'));
            const img = out.querySelector('img');
            expect(img.getAttribute('src')).toBe('/a.png');
            expect(img.hasAttribute('onerror')).toBe(false);
        });
    });

    it('handles mixed safe and malicious content in one answer', () => {
        const html = '<p onclick="x()">Hallo <a href="javascript:x()">hier</a></p>'
            + '<iframe></iframe><strong>Ende</strong>';
        expect(sanitize(html)).toBe('<p>Hallo <a>hier</a></p><strong>Ende</strong>');
    });

    it('exposes sanitizeFAQAnswer on window', () => {
        const fakeWindow = {};
        // eslint-disable-next-line no-new-func
        new Function('window', 'document', readStatic('shared/js/faq-sanitizer.js'))(fakeWindow, document);
        expect(typeof fakeWindow.sanitizeFAQAnswer).toBe('function');
        expect(fakeWindow.sanitizeFAQAnswer('<script></script><b>x</b>')).toBe('<b>x</b>');
    });

    // The header documents href as allowlisted ("only http, https, mailto, /"),
    // but the code only denylists a few prefixes. Browsers strip tabs/newlines
    // inside URL schemes, so "java\tscript:" still executes when clicked.
    it('aqaa: suspected bug - href allowlist (only http, https, mailto, /) is not enforced; "java&#9;script:" survives', () => {
        const out = parse(sanitize('<a href="java&#9;script:alert(1)">Klick</a>'));
        expect(out.querySelector('a').hasAttribute('href')).toBe(false);

        const out2 = parse(sanitize('<a href="file:///etc/passwd">Datei</a>'));
        expect(out2.querySelector('a').hasAttribute('href')).toBe(false);
    });
});
