/**
 * Unified FAQ Data for Gassigeher
 * Single source of truth for both landing page and in-app help center
 *
 * Categories:
 * - platform: Pre-registration questions (landing page)
 * - booking: Booking-related questions (in-app)
 * - account: Account management questions (in-app)
 * - dogs: Dog and level-related questions (in-app)
 * - shared: Questions shown on both landing and in-app
 */

const FAQ_DATA = [
    // ============================================
    // PLATFORM FAQs (Landing Page)
    // ============================================
    {
        id: 1,
        category: 'platform',
        question: 'Kann ich Gassigeher vor der Registrierung testen?',
        answer: 'Ja! Wir bieten eine vollständige <a href="/landing/demo.html">Live-Demo</a> an. Sie können alle Funktionen ausprobieren - als Administrator, als Gassigeher mit verschiedenen Erfahrungsstufen. Die Demo wird täglich zurückgesetzt, so dass Sie bedenkenlos alles testen können.',
        keywords: ['demo', 'testen', 'ausprobieren', 'vorschau']
    },
    {
        id: 2,
        category: 'platform',
        question: 'Ist Gassigeher wirklich kostenlos?',
        answer: 'Ja! Der <strong>Free-Plan</strong> ist dauerhaft kostenlos und enthält alle Grundfunktionen für bis zu 10 Hunde. Für größere Tierheime mit mehr Hunden bieten wir optional einen <strong>Pro-Plan</strong> ab 29 EUR/Monat an. Es gibt keine versteckten Kosten.',
        keywords: ['kostenlos', 'gratis', 'preis', 'kosten', 'free', 'bezahlen']
    },
    {
        id: 3,
        category: 'platform',
        question: 'Wie lange dauert die Einrichtung?',
        answer: 'Die Registrierung dauert etwa 2-3 Minuten. Danach können Sie sofort Hunde hinzufügen und Ihr System einrichten. Die meisten Tierheime sind innerhalb einer Stunde voll einsatzbereit.',
        keywords: ['einrichtung', 'setup', 'installation', 'starten', 'anfangen']
    },
    {
        id: 4,
        category: 'platform',
        question: 'Welche technischen Voraussetzungen gibt es?',
        answer: 'Keine! Gassigeher läuft komplett im Browser. Sie brauchen nur einen Internetzugang und einen aktuellen Browser (Chrome, Firefox, Safari oder Edge). Es muss nichts installiert werden.',
        keywords: ['browser', 'technisch', 'voraussetzungen', 'installieren', 'computer', 'handy']
    },
    {
        id: 5,
        category: 'platform',
        question: 'Können Freiwillige selbst Termine buchen?',
        answer: 'Ja! Freiwillige registrieren sich mit ihrer E-Mail-Adresse und können dann Spaziergänge selbstständig buchen. Sie sehen in einem Kalender alle verfügbaren Termine und Hunde, die ihrer Erfahrungsstufe entsprechen.',
        keywords: ['selbst', 'buchen', 'freiwillige', 'registrieren', 'termin']
    },
    {
        id: 6,
        category: 'platform',
        question: 'Wie werden Daten geschützt?',
        answer: 'Gassigeher ist DSGVO-konform. Alle Daten werden verschlüsselt übertragen (SSL) und in Deutschland gehostet. Nutzer können ihr Konto jederzeit selbst löschen. Passwörter werden sicher gehasht gespeichert.',
        keywords: ['dsgvo', 'datenschutz', 'sicherheit', 'verschlüsselung', 'daten', 'privat']
    },
    {
        id: 7,
        category: 'platform',
        question: 'Kann ich das Design anpassen?',
        answer: 'Ja! Sie können aus verschiedenen Farbthemen wählen oder eigene Farben definieren. So passt Gassigeher zu Ihrem Tierheim-Branding.',
        keywords: ['design', 'farben', 'theme', 'anpassen', 'branding', 'logo']
    },
    {
        id: 8,
        category: 'platform',
        question: 'Was passiert bei technischen Problemen?',
        answer: 'Bei technischen Fragen können Sie uns jederzeit per E-Mail kontaktieren. Wir bemühen uns, alle Anfragen innerhalb von 24-48 Stunden zu beantworten.',
        keywords: ['support', 'hilfe', 'problem', 'fehler', 'kontakt', 'technisch']
    },
    {
        id: 9,
        category: 'platform',
        question: 'Kann ich meine Daten exportieren?',
        answer: 'Ja, Sie können jederzeit Ihre Daten exportieren. Kontaktieren Sie uns einfach und wir stellen Ihnen einen vollständigen Export zur Verfügung.',
        keywords: ['export', 'daten', 'herunterladen', 'download', 'backup']
    },
    {
        id: 10,
        category: 'platform',
        question: 'Wie kann ich Gassigeher unterstützen?',
        answer: 'Am einfachsten über unsere <a href="https://buymeacoffee.com/gassigeher" target="_blank">Buy Me a Coffee</a> Seite. Jede Spende hilft uns, die Hosting-Kosten zu decken und das System weiterzuentwickeln. Aber am meisten hilft es uns, wenn Sie Gassigeher weiterempfehlen!',
        keywords: ['spende', 'unterstützen', 'helfen', 'donation', 'fördern']
    },

    // ============================================
    // SHARED FAQs (Both Landing & In-App)
    // ============================================
    {
        id: 11,
        category: 'shared',
        question: 'Was sind Farbkategorien?',
        answer: 'Jeder Hund ist einer Farbkategorie zugeordnet. Die Farben und ihre Bedeutung werden von Ihrem Tierheim individuell festgelegt.<br><br>' +
            'Neue Nutzer starten mit einer Standardfarbe. Um weitere Hunde ausführen zu dürfen, können Sie zusätzliche Farben über Ihr Profil beantragen. ' +
            'Ein Administrator prüft Ihren Antrag und schaltet die Farbe bei Genehmigung frei.',
        keywords: ['farbe', 'kategorie', 'farbkategorie', 'stufe', 'level', 'zugang', 'beantragen', 'freischalten']
    },

    // ============================================
    // BOOKING FAQs (In-App)
    // ============================================
    {
        id: 20,
        category: 'booking',
        question: 'Wie buche ich einen Spaziergang?',
        answer: '1. Gehen Sie zur <a href="/dogs.html">Hunde-Seite</a> oder zum <a href="/calendar.html">Kalender</a><br>' +
            '2. Wählen Sie einen Hund aus, der Ihrer Erfahrungsstufe entspricht<br>' +
            '3. Klicken Sie auf "Buchen" und wählen Sie Datum und Uhrzeit<br>' +
            '4. Bestätigen Sie die Buchung<br><br>' +
            'Sie erhalten eine Bestätigungs-E-Mail mit allen Details.',
        keywords: ['buchen', 'reservieren', 'spaziergang', 'termin', 'wie']
    },
    {
        id: 21,
        category: 'booking',
        question: 'Wie storniere ich eine Buchung?',
        answer: 'Gehen Sie zu Ihrem <a href="/dashboard.html">Dashboard</a> und klicken Sie bei der entsprechenden Buchung auf "Stornieren".<br><br>' +
            '<strong>Wichtig:</strong> Bitte stornieren Sie mindestens 12 Stunden vor dem Termin (kann vom Tierheim angepasst werden), damit andere Gassigeher den Termin übernehmen können.',
        keywords: ['stornieren', 'absagen', 'abbrechen', 'cancel', 'termin']
    },
    {
        id: 22,
        category: 'booking',
        question: 'Wo sehe ich meine Buchungshistorie?',
        answer: 'Auf Ihrem <a href="/dashboard.html">Dashboard</a> sehen Sie alle Ihre Buchungen:<br>' +
            '• Kommende Termine<br>' +
            '• Vergangene Spaziergänge<br>' +
            '• Stornierte Buchungen<br><br>' +
            'Sie können auch Notizen zu vergangenen Spaziergängen hinzufügen.',
        keywords: ['historie', 'verlauf', 'vergangen', 'buchungen', 'übersicht']
    },
    {
        id: 23,
        category: 'booking',
        question: 'Was sind die Buchungszeiten?',
        answer: 'Die verfügbaren Buchungszeiten werden vom Tierheim festgelegt und können je nach Wochentag variieren:<br><br>' +
            '• <strong>Werktage:</strong> Meist Vormittag, Nachmittag und Abend<br>' +
            '• <strong>Wochenende:</strong> Oft andere Zeiten<br>' +
            '• <strong>Feiertage:</strong> Können abweichen<br><br>' +
            'Die genauen Zeiten sehen Sie beim Buchen.',
        keywords: ['zeiten', 'uhrzeit', 'wann', 'vormittag', 'nachmittag', 'abend']
    },
    {
        id: 24,
        category: 'booking',
        question: 'Warum ist ein Datum blockiert?',
        answer: 'Ein Datum kann aus verschiedenen Gründen blockiert sein:<br><br>' +
            '• <strong>Feiertag:</strong> An manchen Feiertagen sind keine Spaziergänge möglich<br>' +
            '• <strong>Tierheim-Event:</strong> Veranstaltungen oder Wartungsarbeiten<br>' +
            '• <strong>Einzelner Hund:</strong> Manche Hunde sind zeitweise nicht verfügbar (Tierarzt, Training)<br><br>' +
            'Blockierte Daten werden im Kalender grau angezeigt.',
        keywords: ['blockiert', 'gesperrt', 'nicht verfügbar', 'datum', 'grau']
    },
    {
        id: 25,
        category: 'booking',
        question: 'Wie funktionieren Erinnerungs-E-Mails?',
        answer: 'Sie erhalten automatisch eine Erinnerung per E-Mail etwa 1-2 Stunden vor Ihrem gebuchten Spaziergang.<br><br>' +
            'Die E-Mail enthält:<br>' +
            '• Name des Hundes<br>' +
            '• Datum und Uhrzeit<br>' +
            '• Ggf. besondere Hinweise<br><br>' +
            'Tipp: Prüfen Sie auch Ihren Spam-Ordner.',
        keywords: ['erinnerung', 'email', 'benachrichtigung', 'reminder', 'vergessen']
    },

    // ============================================
    // ACCOUNT FAQs (In-App)
    // ============================================
    {
        id: 30,
        category: 'account',
        question: 'Wie setze ich mein Passwort zurück?',
        answer: '1. Gehen Sie zur <a href="/forgot-password.html">Passwort vergessen</a> Seite<br>' +
            '2. Geben Sie Ihre E-Mail-Adresse ein<br>' +
            '3. Sie erhalten einen Link per E-Mail<br>' +
            '4. Klicken Sie auf den Link und setzen Sie ein neues Passwort<br><br>' +
            '<strong>Hinweis:</strong> Der Link ist aus Sicherheitsgründen nur 24 Stunden gültig.',
        keywords: ['passwort', 'vergessen', 'zurücksetzen', 'reset', 'kennwort', 'login']
    },
    {
        id: 31,
        category: 'account',
        question: 'Wie ändere ich meine E-Mail-Adresse?',
        answer: 'Gehen Sie zu Ihrem <a href="/profile.html">Profil</a> und ändern Sie die E-Mail-Adresse im entsprechenden Feld.<br><br>' +
            '<strong>Wichtig:</strong> Sie müssen die neue E-Mail-Adresse bestätigen, bevor sie aktiv wird. Prüfen Sie Ihr Postfach und klicken Sie auf den Bestätigungslink.',
        keywords: ['email', 'ändern', 'adresse', 'mail', 'postfach']
    },
    {
        id: 32,
        category: 'account',
        question: 'Wie lade ich ein Profilbild hoch?',
        answer: '1. Gehen Sie zu Ihrem <a href="/profile.html">Profil</a><br>' +
            '2. Klicken Sie auf das Profilbild oder "Foto ändern"<br>' +
            '3. Wählen Sie ein Bild von Ihrem Gerät (JPEG oder PNG, max. 5 MB)<br>' +
            '4. Das Bild wird automatisch gespeichert<br><br>' +
            'Ihr Profilbild wird anderen Nutzern und Admins angezeigt.',
        keywords: ['profilbild', 'foto', 'bild', 'avatar', 'hochladen']
    },
    {
        id: 33,
        category: 'account',
        question: 'Warum wurde mein Konto deaktiviert?',
        answer: 'Konten werden automatisch deaktiviert, wenn sie längere Zeit (standardmäßig 365 Tage) nicht genutzt wurden.<br><br>' +
            '<strong>Gründe für Deaktivierung:</strong><br>' +
            '• Lange Inaktivität<br>' +
            '• Manuelle Deaktivierung durch Admin<br>' +
            '• Sicherheitsgründe<br><br>' +
            'Sie können jederzeit eine Reaktivierung beantragen.',
        keywords: ['deaktiviert', 'gesperrt', 'inaktiv', 'konto', 'zugang']
    },
    {
        id: 34,
        category: 'account',
        question: 'Wie beantrage ich eine Reaktivierung?',
        answer: 'Wenn Ihr Konto deaktiviert wurde:<br><br>' +
            '1. Versuchen Sie sich einzuloggen<br>' +
            '2. Sie werden zur Reaktivierungs-Seite weitergeleitet<br>' +
            '3. Geben Sie Ihre E-Mail-Adresse ein<br>' +
            '4. Ein Administrator wird Ihre Anfrage prüfen<br>' +
            '5. Sie erhalten eine E-Mail, sobald Ihr Konto reaktiviert wurde',
        keywords: ['reaktivierung', 'wieder', 'aktivieren', 'entsperren', 'antrag']
    },
    {
        id: 35,
        category: 'account',
        question: 'Wie lösche ich mein Konto (DSGVO)?',
        answer: 'Sie können Ihr Konto jederzeit selbst löschen:<br><br>' +
            '1. Gehen Sie zu Ihrem <a href="/profile.html">Profil</a><br>' +
            '2. Scrollen Sie zum Abschnitt "Konto löschen"<br>' +
            '3. Bestätigen Sie mit Ihrem Passwort<br><br>' +
            '<strong>Wichtig:</strong> Diese Aktion kann nicht rückgängig gemacht werden. Ihre persönlichen Daten werden anonymisiert, die Buchungshistorie bleibt für Tierheim-Statistiken erhalten.',
        keywords: ['löschen', 'konto', 'dsgvo', 'daten', 'entfernen', 'account']
    },
    {
        id: 36,
        category: 'account',
        question: 'Warum erhalte ich keine E-Mails?',
        answer: 'Wenn Sie keine E-Mails erhalten:<br><br>' +
            '1. <strong>Spam-Ordner prüfen:</strong> E-Mails landen manchmal dort<br>' +
            '2. <strong>E-Mail-Adresse prüfen:</strong> Ist sie korrekt im Profil?<br>' +
            '3. <strong>Filter deaktivieren:</strong> Manche E-Mail-Programme filtern zu streng<br>' +
            '4. <strong>Absender whitelisten:</strong> Fügen Sie die Absenderadresse der Gassigeher-E-Mails zu Ihren Kontakten hinzu<br><br>' +
            'Bei weiterhin fehlenden E-Mails kontaktieren Sie bitte den Support.',
        keywords: ['email', 'nicht', 'erhalten', 'spam', 'posteingang', 'mail']
    },
    {
        id: 37,
        category: 'account',
        question: 'Wie bestätige ich meine E-Mail-Adresse?',
        answer: 'Nach der Registrierung erhalten Sie eine Bestätigungs-E-Mail:<br><br>' +
            '1. Öffnen Sie die E-Mail von Gassigeher<br>' +
            '2. Klicken Sie auf den Bestätigungslink<br>' +
            '3. Ihr Konto wird aktiviert<br><br>' +
            '<strong>Keine E-Mail erhalten?</strong> Prüfen Sie Ihren Spam-Ordner oder fordern Sie eine neue Bestätigung an.',
        keywords: ['bestätigen', 'verifizieren', 'email', 'link', 'aktivieren']
    },

    // ============================================
    // DOGS FAQs (In-App)
    // ============================================
    {
        id: 40,
        category: 'dogs',
        question: 'Warum kann ich bestimmte Hunde nicht buchen?',
        answer: 'Hunde mit einem 🔒 Symbol erfordern eine höhere Erfahrungsstufe als Sie aktuell haben.<br><br>' +
            '<strong>Beispiel:</strong> Ein 🟠 Orange-Hund kann nicht von einem 🟢 Grün-Nutzer gebucht werden.<br><br>' +
            'Sie können eine <a href="/profile.html">Stufenerhöhung beantragen</a>, wenn Sie mehr Erfahrung gesammelt haben.',
        keywords: ['gesperrt', 'buchen', 'nicht', 'schloss', 'level', 'stufe']
    },
    {
        id: 41,
        category: 'dogs',
        question: 'Wie beantrage ich eine höhere Erfahrungsstufe?',
        answer: '1. Gehen Sie zu Ihrem <a href="/profile.html">Profil</a><br>' +
            '2. Im Abschnitt "Erfahrungsstufe" klicken Sie auf "Höhere Stufe beantragen"<br>' +
            '3. Wählen Sie die gewünschte Stufe (z.B. von Grün auf Orange)<br>' +
            '4. Warten Sie auf die Prüfung durch einen Administrator<br><br>' +
            '<strong>Tipp:</strong> Sammeln Sie erst einige Spaziergänge auf Ihrer aktuellen Stufe.',
        keywords: ['stufe', 'erhöhen', 'beantragen', 'level', 'aufstieg', 'promotion']
    },
    {
        id: 42,
        category: 'dogs',
        question: 'Wie lange dauert die Prüfung meines Stufenantrags?',
        answer: 'Die Bearbeitungszeit hängt vom Tierheim ab, typischerweise:<br><br>' +
            '• <strong>1-3 Werktage</strong> bei den meisten Tierheimen<br>' +
            '• Bei hohem Aufkommen kann es länger dauern<br><br>' +
            'Sie erhalten eine E-Mail, sobald Ihr Antrag bearbeitet wurde (genehmigt oder abgelehnt mit Begründung).',
        keywords: ['dauer', 'warten', 'prüfung', 'antrag', 'bearbeitung']
    },
    {
        id: 43,
        category: 'dogs',
        question: 'Was bedeutet "Hund nicht verfügbar"?',
        answer: 'Ein Hund kann vorübergehend nicht verfügbar sein wegen:<br><br>' +
            '• <strong>Tierarztbesuch:</strong> Medizinische Untersuchung oder Behandlung<br>' +
            '• <strong>Training:</strong> Verhaltenstherapie oder Schulung<br>' +
            '• <strong>Vermittlung:</strong> Der Hund wurde möglicherweise vermittelt<br>' +
            '• <strong>Quarantäne:</strong> Aus gesundheitlichen Gründen<br><br>' +
            'Nicht verfügbare Hunde werden ausgegraut angezeigt.',
        keywords: ['nicht verfügbar', 'ausgegraut', 'hund', 'warum']
    },
    {
        id: 44,
        category: 'dogs',
        question: 'Was sind "Featured Dogs" / Hervorgehobene Hunde?',
        answer: 'Hervorgehobene Hunde werden prominent auf der Startseite angezeigt. Das sind oft:<br><br>' +
            '• Hunde, die dringend Bewegung brauchen<br>' +
            '• Neue Hunde im Tierheim<br>' +
            '• Hunde, die selten gebucht werden<br><br>' +
            'Diese Markierung wird von Administratoren vergeben.',
        keywords: ['featured', 'hervorgehoben', 'besonders', 'startseite']
    },
    {
        id: 45,
        category: 'dogs',
        question: 'Was ist der Unterschied zwischen den Spaziergang-Typen?',
        answer: 'Je nach Tierheim gibt es verschiedene Spaziergang-Arten:<br><br>' +
            '• <strong>Kurz (15-30 Min):</strong> Schnelle Runde, ideal für ältere oder kranke Hunde<br>' +
            '• <strong>Normal (30-60 Min):</strong> Standard-Spaziergang<br>' +
            '• <strong>Lang (60+ Min):</strong> Ausgedehnter Spaziergang für aktive Hunde<br><br>' +
            'Die verfügbaren Typen werden beim Buchen angezeigt.',
        keywords: ['spaziergang', 'typ', 'dauer', 'kurz', 'lang', 'normal']
    }
];

/**
 * Get FAQs for the landing page (pre-registration questions)
 * @returns {Array} FAQs for landing page
 */
function getFAQsForLanding() {
    return FAQ_DATA.filter(faq => ['platform', 'shared'].includes(faq.category));
}

/**
 * Get FAQs for the in-app help center (operational questions)
 * @returns {Array} FAQs for in-app help
 */
function getFAQsForApp() {
    return FAQ_DATA.filter(faq => ['booking', 'account', 'dogs', 'shared'].includes(faq.category));
}

/**
 * Get FAQs by category
 * @param {string} category - Category to filter by
 * @returns {Array} FAQs in the specified category
 */
function getFAQsByCategory(category) {
    if (category === 'all') {
        return getFAQsForApp();
    }
    return FAQ_DATA.filter(faq => faq.category === category);
}

/**
 * Search FAQs by keyword
 * @param {string} query - Search query
 * @param {boolean} landingOnly - If true, search only landing FAQs
 * @returns {Array} Matching FAQs
 */
function searchFAQs(query, landingOnly = false) {
    // Handle null, undefined, or non-string input
    if (query === null || query === undefined || typeof query !== 'string') {
        return landingOnly ? getFAQsForLanding() : getFAQsForApp();
    }

    const normalizedQuery = query.toLowerCase().trim();
    if (!normalizedQuery) {
        return landingOnly ? getFAQsForLanding() : getFAQsForApp();
    }

    const baseFAQs = landingOnly ? getFAQsForLanding() : getFAQsForApp();

    return baseFAQs.filter(faq => {
        const questionMatch = faq.question.toLowerCase().includes(normalizedQuery);
        const answerMatch = faq.answer.toLowerCase().includes(normalizedQuery);
        const keywordMatch = faq.keywords.some(kw => kw.toLowerCase().includes(normalizedQuery));
        return questionMatch || answerMatch || keywordMatch;
    });
}

/**
 * Get FAQs relevant to contact form categories
 * Maps contact form categories to relevant FAQ categories
 * @param {string} contactCategory - Contact form category
 * @returns {Array} Relevant FAQs
 */
function getFAQsForContactCategory(contactCategory) {
    const categoryMapping = {
        'general': ['platform', 'shared'],
        'support': ['account', 'booking', 'dogs'],
        'sales': ['platform'],
        'partnership': ['platform'],
        'press': ['platform'],
        'other': ['platform', 'shared']
    };

    const relevantCategories = categoryMapping[contactCategory] || ['platform', 'shared'];
    return FAQ_DATA.filter(faq => relevantCategories.includes(faq.category));
}

// Export for use in both browser and potential Node.js testing
if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
        FAQ_DATA,
        getFAQsForLanding,
        getFAQsForApp,
        getFAQsByCategory,
        searchFAQs,
        getFAQsForContactCategory
    };
}
