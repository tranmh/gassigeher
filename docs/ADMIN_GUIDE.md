# Gassigeher - Administrator-Handbuch

**Umfassende Anleitung für Administratoren zur Verwaltung der Gassigeher-Plattform.**

**🎯 Verwaltung**: 8 Admin-Seiten | Dashboard mit Live-Statistiken | Vollständige Kontrolle
**🔧 Funktionen**: Hunde, Buchungen, Benutzer, Einstellungen, Level-Anfragen, Reaktivierungen

> **Für Benutzer**: Siehe [USER_GUIDE.md](USER_GUIDE.md)
> **Für Deployment**: Siehe [DEPLOYMENT.md](DEPLOYMENT.md)
> **API-Referenz**: Siehe [API.md](API.md)

---

## Administrator-Zugang

### Wie werde ich Administrator?

Administratoren werden über die Umgebungsvariable `ADMIN_EMAILS` definiert:

```bash
ADMIN_EMAILS=admin@example.com,admin2@example.com
```

**Wichtig**:
- Änderungen erfordern Neustart des Servers
- Verwenden Sie nur vertrauenswürdige E-Mail-Adressen
- Admins haben volle Kontrolle über das System

### Anmelden

1. Registrieren Sie sich mit Ihrer Admin-E-Mail
2. Verifizieren Sie Ihre E-Mail wie normale Nutzer
3. Melden Sie sich an
4. Sie werden automatisch zur Admin-Seite weitergeleitet

---

## Admin Dashboard

### Dashboard-Übersicht

Das Dashboard zeigt Ihnen auf einen Blick:

**Statistiken:**
- 📊 Gesamt abgeschlossene Spaziergänge
- 📅 Heute anstehende Spaziergänge
- 👥 Anzahl aktiver Benutzer
- ⚠️ Anzahl inaktiver Benutzer
- 🐕 Verfügbare Hunde
- 🚫 Nicht verfügbare Hunde
- ⭐ Ausstehende Level-Anfragen
- 🔄 Ausstehende Reaktivierungsanfragen

**Letzte Aktivitäten:**
- Neue Buchungen
- Abgeschlossene Spaziergänge
- Stornierungen

**Schnellzugriff:**
- Links zu allen Verwaltungsseiten

---

## Hundeverwaltung

### Neuen Hund hinzufügen

1. Gehen Sie zu "Hunde verwalten"
2. Klicken Sie auf "Hund hinzufügen"
3. Füllen Sie das Formular aus:
   - **Name**: Name des Hundes
   - **Rasse**: Rasse (wird für Filter verwendet)
   - **Größe**: Klein, Mittel oder Groß
   - **Alter**: In Jahren
   - **Kategorie**: Grün, Blau oder Orange
   - **Besondere Bedürfnisse** (optional)
   - **Abholort**: Wo der Hund abgeholt wird
   - **Spazierweg** (optional): Bevorzugte Routen
   - **Spazierdauer**: In Minuten
   - **Besondere Anweisungen**: Wichtige Hinweise
   - **Standard Morgenzeit**: z.B. 09:00
   - **Standard Abendzeit**: z.B. 17:00
4. Klicken Sie auf "Speichern"

### Hund bearbeiten

1. Finden Sie den Hund in der Liste
2. Klicken Sie auf das Bearbeiten-Symbol (✏️)
3. Ändern Sie die gewünschten Felder
4. Klicken Sie auf "Speichern"

### Hundefoto hochladen

**Beim Hinzufügen eines neuen Hundes:**

1. Füllen Sie das Formular aus (Name, Rasse, etc.)
2. Im Abschnitt "Foto":
   - Klicken Sie auf die Upload-Zone **oder**
   - Ziehen Sie eine Datei per Drag & Drop in die Zone
3. Vorschau wird angezeigt
4. Klicken Sie "Speichern" - Hund und Foto werden hochgeladen
5. Das Foto erscheint in der Hundeliste

**Beim Bearbeiten eines bestehenden Hundes:**

**Ohne Foto (Foto hinzufügen):**
1. Klicken Sie auf ✏️ beim Hund
2. Im Abschnitt "Foto":
   - Klicken Sie auf die Upload-Zone **oder**
   - Ziehen Sie eine Datei in die Zone
3. Vorschau wird angezeigt
4. Klicken Sie "Speichern"

**Mit bestehendem Foto (Foto ändern):**
1. Klicken Sie auf ✏️ beim Hund
2. Aktuelles Foto wird angezeigt
3. Klicken Sie "Foto ändern"
4. Wählen Sie neue Datei aus
5. Vorschau wird angezeigt
6. Klicken Sie "Speichern"

**Unterstützte Formate:**
- ✅ JPEG (.jpg, .jpeg)
- ✅ PNG (.png)
- ❌ Andere Formate (GIF, BMP, etc.) nicht erlaubt

**Maximale Dateigröße:** 10MB

**Hinweise:**
- Drag & Drop funktioniert in allen modernen Browsern
- Eine Vorschau wird vor dem Upload angezeigt
- Das × Symbol entfernt die Vorschau (Datei wird nicht hochgeladen)
- Bei Fehlern erscheint eine deutsche Fehlermeldung
- Alte Fotos werden automatisch beim Upload neuer Fotos gelöscht

**Platzhalterbild:**
Hunde ohne Foto zeigen ein professionelles Platzbild in der Farbe ihrer Kategorie:
- 🟢 Grüne Hunde: Grünes Platzhalterbild
- 🔵 Blaue Hunde: Blaues Platzhalterbild
- 🟠 Orange Hunde: Oranges Platzhalterbild

### Hund als nicht verfügbar markieren

**Wann nutzen:**
- Hund ist krank
- Tierarztbesuch
- Vorübergehende Gründe

**Vorgang:**
1. Klicken Sie auf das 🚫-Symbol beim Hund
2. Geben Sie einen Grund ein (z.B. "Tierarztbesuch")
3. Der Hund wird als nicht verfügbar angezeigt
4. Nutzer können ihn nicht buchen

**Wieder verfügbar machen:**
1. Klicken Sie auf das ✅-Symbol
2. Hund ist sofort wieder buchbar

### Hund löschen

**Vorsicht**: Hunde mit zukünftigen Buchungen können nicht gelöscht werden!

1. Klicken Sie auf das 🗑️-Symbol
2. Bestätigen Sie die Löschung
3. Hund wird permanent entfernt

---

## Buchungsverwaltung

### Alle Buchungen anzeigen

1. Gehen Sie zu "Buchungen verwalten"
2. Sehen Sie alle Buchungen aller Nutzer

### Buchungen filtern

Nutzen Sie Filter:
- **Status**: Geplant, Abgeschlossen, Storniert
- **Datum ab**: Startdatum
- **Datum bis**: Enddatum

### Buchung stornieren (Admin)

1. Finden Sie die Buchung
2. Klicken Sie auf "Stornieren"
3. Geben Sie einen Grund ein (Pflicht!)
4. Bestätigen Sie
5. Der Nutzer erhält eine E-Mail mit dem Grund

**Beispiel-Gründe:**
- "Hund ist krank"
- "Unvorhergesehener Notfall"
- "Wetterbedingungen zu schlecht"

### Buchung verschieben

1. Finden Sie die Buchung
2. Klicken Sie auf "Verschieben"
3. Geben Sie ein:
   - Neues Datum (JJJJ-MM-TT)
   - Spaziergang (Morgen/Abend)
   - Neue Uhrzeit (HH:MM)
   - Grund (Pflicht!)
4. Bestätigen Sie
5. Der Nutzer erhält eine E-Mail mit alten und neuen Details

---

## Gesperrte Tage verwalten

### Tag sperren

**Wann nutzen:**
- Feiertage
- Wetterwarnungen
- Veranstaltungen im Tierheim
- Personalmangel

**Vorgang:**
1. Gehen Sie zu "Gesperrte Tage verwalten"
2. Klicken Sie auf "Tag sperren"
3. Wählen Sie das Datum
4. Geben Sie einen Grund ein (wird Nutzern angezeigt)
5. Speichern

**Beispiel-Gründe:**
- "Feiertag - Tierheim geschlossen"
- "Unwetterwarnung"
- "Tierheim-Veranstaltung"

### Sperrung aufheben

1. Finden Sie den gesperrten Tag
2. Klicken Sie auf "Aufheben"
3. Bestätigen Sie
4. Tag ist sofort wieder buchbar

---

## Erfahrungslevel-Anfragen

### Anfragen prüfen

1. Gehen Sie zu "Level-Anfragen verwalten"
2. Sehen Sie alle ausstehenden Anfragen
3. Für jeden Nutzer sehen Sie:
   - Name und E-Mail
   - Aktuelles Level
   - Angefragtes Level
   - Antragsdatum

### Anfrage genehmigen

**Prüfkriterien:**
- Anzahl abgeschlossener Spaziergänge
- Qualität der Notizen
- Zuverlässigkeit (Stornierungen)
- Feedback von Mitarbeitern

**Vorgang:**
1. Klicken Sie auf "Genehmigen"
2. Optional: Geben Sie eine Nachricht ein
3. Das Level des Nutzers wird automatisch erhöht
4. Nutzer erhält E-Mail-Benachrichtigung

### Anfrage ablehnen

1. Klicken Sie auf "Ablehnen"
2. Optional: Geben Sie einen Grund ein (empfohlen)
3. Nutzer erhält E-Mail
4. Nutzer kann später erneut anfragen

**Beispiel-Nachrichten:**
- "Bitte sammeln Sie mehr Erfahrung mit 10+ Spaziergängen"
- "Genehmigt! Sie haben großartige Arbeit geleistet"

---

## Benutzerverwaltung

### Alle Benutzer anzeigen

1. Gehen Sie zu "Benutzer"
2. Sehen Sie Liste aller Nutzer
3. Filtern Sie nach "Aktiv" oder "Inaktiv"

### Benutzer deaktivieren

**Wann nutzen:**
- Wiederholte Unzuverlässigkeit
- Verstoß gegen AGB
- Auf Nutzerwunsch

**Vorgang:**
1. Finden Sie den Nutzer
2. Klicken Sie auf "Deaktivieren"
3. Geben Sie einen Grund ein (Pflicht!)
4. Bestätigen Sie
5. Der Nutzer erhält eine E-Mail
6. Alle zukünftigen Buchungen werden storniert

### Benutzer aktivieren

1. Finden Sie den deaktivierten Nutzer
2. Klicken Sie auf "Aktivieren"
3. Optional: Geben Sie eine Willkommensnachricht ein
4. Bestätigen Sie
5. Der Nutzer erhält eine E-Mail

---

## Reaktivierungsanfragen

### Anfragen prüfen

1. Gehen Sie zu "Reaktivierungen"
2. Sehen Sie alle ausstehenden Anfragen
3. Für jeden Nutzer sehen Sie:
   - Deaktivierungsgrund
   - Deaktivierungsdatum
   - Spaziergangshistorie

### Anfrage genehmigen

1. Klicken Sie auf "Genehmigen"
2. Optional: Nachricht eingeben
3. Der Nutzer wird automatisch reaktiviert
4. Nutzer erhält E-Mail

### Anfrage ablehnen

1. Klicken Sie auf "Ablehnen"
2. Optional: Begründung eingeben (empfohlen)
3. Nutzer erhält E-Mail
4. Konto bleibt deaktiviert

---

## Systemeinstellungen

### Einstellungen anpassen

Gehen Sie zu "Einstellungen" und konfigurieren Sie:

**Buchungsvorlauf (Tage)**
- Standard: 14 Tage
- Bereich: 1-90 Tage
- Wie weit im Voraus können Nutzer buchen?

**Stornierungsfrist (Stunden)**
- Standard: 12 Stunden
- Bereich: 1-72 Stunden
- Wie viele Stunden vor dem Spaziergang können Nutzer stornieren?

**Auto-Deaktivierung (Tage)**
- Standard: 365 Tage (1 Jahr)
- Bereich: 30-730 Tage
- Nach wie vielen Tagen Inaktivität werden Nutzer automatisch deaktiviert?

**Nach jeder Änderung:**
- Klicken Sie auf "Speichern" für die jeweilige Einstellung
- Die Änderung gilt sofort

---

## Automatisierte Prozesse

### Automatische Spaziergangs-Abschlüsse

**Was**: Vergangene Spaziergänge werden automatisch als "Abgeschlossen" markiert
**Wann**: Stündlich
**Nutzen**: Nutzer können danach Notizen hinzufügen

### Automatische Deaktivierung

**Was**: Inaktive Nutzer werden deaktiviert
**Wann**: Täglich um 3:00 Uhr morgens
**Kriterium**: Keine Aktivität für konfigurierte Anzahl Tage (Standard: 365)
**E-Mail**: Nutzer erhalten Benachrichtigung mit Reaktivierungshinweis

### Datenbank-Backups

**Was**: Komplettes Datenbank-Backup
**Wann**: Täglich um 2:00 Uhr morgens
**Speicherort**: `/var/gassigeher/backups/`
**Aufbewahrung**: 30 Tage
**Format**: Komprimiert (.gz)

---

## Best Practices

### Hundekategorien zuweisen

**Grün** - Nutzen Sie für:
- Ruhige, gut erzogene Hunde
- Kleine bis mittelgroße Hunde
- Hunde ohne besondere Anforderungen

**Blau** - Nutzen Sie für:
- Energiegeladene Hunde
- Große Hunde
- Hunde mit leichten besonderen Bedürfnissen

**Orange** - Nutzen Sie für:
- Sehr große oder kräftige Hunde
- Hunde mit Verhaltensproblemen
- Hunde, die besondere Erfahrung erfordern

### Nutzer-Level genehmigen

**Empfohlene Kriterien für Blau:**
- Mindestens 10 abgeschlossene Spaziergänge
- Keine Stornierungen in letzter Minute
- Positive Notizen

**Empfohlene Kriterien für Orange:**
- Mindestens 25 abgeschlossene Spaziergänge
- Davon mindestens 10 mit blauen Hunden
- Ausgezeichnete Zuverlässigkeit
- Detaillierte, hilfreiche Notizen

### Kommunikation

**Bei Stornierungen:**
- Seien Sie transparent über den Grund
- Bieten Sie Alternativen an, wenn möglich
- Entschuldigen Sie sich für Unannehmlichkeiten

**Bei Deaktivierungen:**
- Erklären Sie klar den Grund
- Geben Sie Informationen zur Reaktivierung
- Seien Sie fair und respektvoll

**Bei Ablehnungen:**
- Geben Sie konstruktives Feedback
- Ermutigen Sie zu weiteren Versuchen
- Seien Sie unterstützend

---

## Tägliche Aufgaben

### Morgen-Check (täglich)

1. **Dashboard prüfen**:
   - Heutige Spaziergänge ansehen
   - Ausstehende Anfragen prüfen

2. **Hunde-Status prüfen**:
   - Kranke Hunde als nicht verfügbar markieren
   - Genesene Hunde wieder freigeben

3. **E-Mails prüfen**:
   - Nutzer-Anfragen beantworten
   - Probleme bearbeiten

### Wöchentliche Aufgaben

1. **Nutzer-Aktivität prüfen**:
   - Inaktive Nutzer identifizieren
   - Bei Bedarf kontaktieren

2. **Level-Anfragen bearbeiten**:
   - Alle ausstehenden Anfragen prüfen
   - Spaziergangshistorie bewerten

3. **Statistiken analysieren**:
   - Beliebte Hunde identifizieren
   - Buchungstrends erkennen

### Monatliche Aufgaben

1. **Backup prüfen**:
   - Backup-Integrität verifizieren
   - Test-Wiederherstellung durchführen

2. **System-Performance**:
   - Datenbankgröße prüfen
   - Serverleistung überwachen

3. **Berichte erstellen**:
   - Spaziergangsstatistiken
   - Nutzer-Engagement
   - Hunde-Auslastung

---

## Fehlerbehebung

### Nutzer kann sich nicht anmelden

**Mögliche Ursachen:**
1. **E-Mail nicht verifiziert**
   - Lösung: Neuen Verifizierungslink senden

2. **Konto deaktiviert**
   - Prüfen: Nutzerliste → Inaktiv-Filter
   - Lösung: Reaktivieren oder Reaktivierungsanfrage genehmigen

3. **Falsches Passwort**
   - Lösung: Nutzer soll "Passwort vergessen" verwenden

### Buchung kann nicht erstellt werden

**Mögliche Ursachen:**
1. **Hund nicht verfügbar**
   - Prüfen: Hunde-Status
   - Lösung: Hund wieder verfügbar machen

2. **Nutzer-Level zu niedrig**
   - Prüfen: Nutzer-Level und Hund-Kategorie
   - Lösung: Level-Anfrage genehmigen oder Hund-Kategorie anpassen

3. **Datum gesperrt**
   - Prüfen: Gesperrte Tage
   - Lösung: Sperrung aufheben, falls angebracht

4. **Doppelbuchung**
   - Prüfen: Buchungen für das Datum
   - Lösung: Anderer Zeitpunkt vorschlagen

### E-Mails werden nicht versendet

**Prüfen:**
1. Gmail API Konfiguration
2. Serverprotokolle: `journalctl -u gassigeher | grep -i email`
3. Gmail API Quota

**Lösung:**
- Refresh Token erneuern
- Gmail API aktivieren in Google Cloud Console
- Quota-Limits prüfen

---

## Sicherheit

### Admin-Konto schützen

1. **Starkes Passwort** verwenden (12+ Zeichen)
2. **Passwort regelmäßig ändern** (alle 90 Tage)
3. **Nicht vom öffentlichen Computer** anmelden
4. **Bei Verdacht** sofort Passwort ändern

### Verdächtige Aktivitäten

**Achten Sie auf:**
- Ungewöhnlich viele Registrierungen
- Spam-Buchungen
- Verdächtige Nutzer-Anfragen

**Bei Verdacht:**
1. Betroffenes Konto deaktivieren
2. Spammy Buchungen stornieren
3. Serverprotokolle prüfen

### Datenbank-Sicherheit

- Regelmäßige Backups prüfen
- Datenbankgröße überwachen
- Bei Verdacht: Datenbankintegrität prüfen

---

## Berichte und Analysen

### Verfügbare Daten

Das Dashboard zeigt:
- Gesamtzahl Spaziergänge
- Nutzer-Statistiken
- Hunde-Verfügbarkeit
- Ausstehende Anfragen

### Erweiterte Analysen

Für detaillierte Analysen:
1. Exportieren Sie Daten aus der Datenbank
2. Nutzen Sie SQL-Abfragen
3. Erstellen Sie Custom-Reports

**Beispiel SQL:**
```sql
-- Beliebteste Hunde
SELECT dogs.name, COUNT(*) as walk_count
FROM bookings
JOIN dogs ON bookings.dog_id = dogs.id
WHERE bookings.status = 'completed'
GROUP BY dogs.id
ORDER BY walk_count DESC
LIMIT 10;

-- Aktivste Nutzer
SELECT users.name, COUNT(*) as booking_count
FROM bookings
JOIN users ON bookings.user_id = users.id
WHERE bookings.status = 'completed'
  AND users.is_deleted = 0
GROUP BY users.id
ORDER BY booking_count DESC
LIMIT 10;
```

---

## Notfallverfahren

### Systemausfall

1. Prüfen Sie Serverstatus: `systemctl status gassigeher`
2. Prüfen Sie Logs: `journalctl -u gassigeher -n 100`
3. Starten Sie neu: `systemctl restart gassigeher`
4. Bei anhaltenden Problemen: Siehe DEPLOYMENT.md

### Datenbank-Korruption

1. Stoppen Sie den Service: `systemctl stop gassigeher`
2. Prüfen Sie Integrität: `sqlite3 gassigeher.db "PRAGMA integrity_check;"`
3. Falls korrupt: Wiederherstellung aus Backup
4. Siehe DEPLOYMENT.md für Details

### Backup-Wiederherstellung

1. Identifizieren Sie das richtige Backup
2. Stoppen Sie den Service
3. Stellen Sie Datenbank wieder her
4. Starten Sie den Service
5. Testen Sie die Funktionalität

---

## Wichtige Hinweise

### Rechtliches

- Sie sind verantwortlich für die Einhaltung lokaler Gesetze
- GDPR-Compliance ist eingebaut, aber überprüfen Sie lokale Anforderungen
- Dokumentieren Sie wichtige Entscheidungen

### Kommunikation mit Nutzern

- Seien Sie höflich und professionell
- Antworten Sie zeitnah auf Anfragen
- Nutzen Sie optionale Nachrichten bei Entscheidungen
- Erklären Sie Ablehnungen konstruktiv

### Datenschutz

- Teilen Sie Nutzerdaten NIEMALS
- Behandeln Sie persönliche Informationen vertraulich
- Folgen Sie GDPR-Richtlinien
- Dokumentieren Sie Datenzugriffe bei Bedarf

---

## Kontakte und Support

### Technischer Support

- **Serverprobleme**: Siehe DEPLOYMENT.md
- **Datenbank**: Siehe DEPLOYMENT.md
- **API-Fragen**: Siehe API.md

### Entwickler-Kontakt

Bei Bugs oder Feature-Anfragen:
- GitHub Issues (wenn Repository öffentlich)
- E-Mail an Entwickler

### Dokumentation

- **Nutzer-Guide**: USER_GUIDE.md
- **API-Dokumentation**: API.md
- **Deployment**: DEPLOYMENT.md
- **Implementierungsplan**: ImplementationPlan.md

---

## Checkliste für neue Administratoren

- [ ] Zugang mit Admin-E-Mail erhalten
- [ ] Dashboard erkundet
- [ ] Testbuchung erstellt und verwaltet
- [ ] Testhund erstellt
- [ ] Tag gesperrt und entsperrt
- [ ] Level-Anfrage genehmigt (Test)
- [ ] Systemeinstellungen verstanden
- [ ] Backup-Prozess geprüft
- [ ] Kontaktinformationen notiert
- [ ] Diese Dokumentation gelesen

---

**Viel Erfolg bei der Verwaltung von Gassigeher! 🐕**

Bei Fragen: support@gassigeher.example.com

---

## Related Documentation

**Essential Guides:**
- [USER_GUIDE.md](USER_GUIDE.md) - User manual (share with users)
- [DEPLOYMENT.md](DEPLOYMENT.md) - Production deployment and troubleshooting
- [API.md](API.md) - Complete API reference

**Technical Documentation:**
- [README.md](../README.md) - Project overview and setup
- [ImplementationPlan.md](ImplementationPlan.md) - Complete architecture
- [CLAUDE.md](../CLAUDE.md) - Development guide

**For Emergencies:**
- Check server logs: `journalctl -u gassigeher -f`
- Database issues: See DEPLOYMENT.md "Troubleshooting"
- Email problems: Check Gmail API credentials in .env

---

**📞 Support Contact**: support@gassigeher.example.com
