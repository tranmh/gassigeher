# Gassigeher - Administrator-Handbuch

**Umfassende Anleitung für Administratoren zur Verwaltung der Gassigeher-Plattform.**

**🎯 Verwaltung**: 10+ Admin-Seiten | Dashboard mit Live-Statistiken | Vollständige Kontrolle
**🔧 Funktionen**: Hunde, Buchungen, Benutzer, Farben, Zeitregeln, Feiertage, Einstellungen

> **Für Benutzer**: Siehe [USER_GUIDE.md](USER_GUIDE.md)
> **Für Deployment**: Siehe [DEPLOYMENT.md](DEPLOYMENT.md)
> **API-Referenz**: Siehe [API.md](API.md)

---

## Administrator-Zugang

### Wie werde ich Administrator?

**Erste Installation:**

Bei der ersten Installation wird automatisch ein **Super Admin** erstellt:
1. Das System erkennt eine leere Datenbank
2. Super Admin wird mit den Zugangsdaten aus `.env` (`SUPER_ADMIN_EMAIL`) erstellt
3. Zufälliges, sicheres Passwort wird generiert
4. Zugangsdaten werden angezeigt:
   - In der Konsole beim Start
   - In der Datei `SUPER_ADMIN_CREDENTIALS.txt`
5. 3 Test-Benutzer, 5 Test-Hunde, 3 Test-Buchungen werden erstellt

**Als zusätzlicher Administrator:**

Nur der **Super Admin** kann weitere Administratoren ernennen:
1. Sie müssen sich zunächst als normaler Benutzer registrieren
2. Verifizieren Sie Ihre E-Mail-Adresse
3. Der Super Admin geht zu "Benutzerverwaltung"
4. Der Super Admin klickt bei Ihrem Account auf "Zu Admin ernennen"
5. Sie erhalten Admin-Rechte sofort (keine E-Mail)
6. Beim nächsten Login haben Sie Zugriff auf alle Admin-Funktionen

**Wichtig**:
- Es gibt nur **einen** Super Admin (ID=1)
- Super Admin kann nicht gelöscht oder herabgestuft werden
- Reguläre Admins können KEINE anderen Admins ernennen
- Weitere Details siehe Abschnitt "Administrator-Verwaltung"

### Anmelden

1. Gehen Sie zur Login-Seite
2. Melden Sie sich mit Ihrer E-Mail und Passwort an
3. Wenn Sie Admin-Rechte haben, werden Sie automatisch zur Admin-Seite weitergeleitet
4. Super Admin sieht zusätzliche Funktionen (z.B. "Zu Admin ernennen" Buttons)

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
- 🎨 Ausstehende Farbanfragen
- 🔄 Ausstehende Reaktivierungsanfragen
- ⏰ Buchungen zur Genehmigung (bei genehmigungspflichtigen Zeiten)

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
   - **Farbe**: Wählen Sie eine Farbe (z.B. Grün, Blau, Orange)
   - **Besondere Bedürfnisse** (optional)
   - **Abholort**: Wo der Hund abgeholt wird
   - **Spazierweg** (optional): Bevorzugte Routen
   - **Spazierdauer**: In Minuten
   - **Besondere Anweisungen**: Wichtige Hinweise
   - **Externer Link** (optional): Link zur Tierheim-Website
   - **Hervorgehoben**: Auf Startseite anzeigen
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
Hunde ohne Foto zeigen ein professionelles Platzhalterbild in der Farbe ihrer zugewiesenen Farbkategorie.

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

## Farbanfragen verwalten

### Anfragen prüfen

1. Gehen Sie zu "Farbanfragen verwalten"
2. Sehen Sie alle ausstehenden Anfragen
3. Für jeden Nutzer sehen Sie:
   - Name und E-Mail
   - Aktuelle Farben
   - Angeforderte Farbe
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
3. Die Farbe wird dem Nutzer automatisch hinzugefügt
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

## Farbverwaltung

### Farben definieren

Sie können eigene Farben für Ihr Tierheim definieren:

1. Gehen Sie zu "Farben verwalten"
2. Klicken Sie auf "Farbe hinzufügen"
3. Geben Sie ein:
   - **Name**: z.B. "Grün", "Anfänger", "Stufe 1"
   - **Hex-Code**: Farbcode (z.B. #4CAF50)
   - **Reihenfolge**: Sortierung in der Liste
4. Klicken Sie auf "Speichern"

### Farben bearbeiten

1. Finden Sie die Farbe in der Liste
2. Klicken Sie auf das Bearbeiten-Symbol
3. Ändern Sie Name, Farbe oder Reihenfolge
4. Speichern Sie

### Farbe löschen

**Vorsicht**: Farben können nur gelöscht werden, wenn keine Hunde oder Nutzer zugeordnet sind!

1. Ordnen Sie alle Hunde/Nutzer anderen Farben zu
2. Klicken Sie auf "Löschen"
3. Bestätigen Sie

### Nutzer-Farben verwalten

Als Admin können Sie Nutzern direkt Farben zuweisen:

1. Gehen Sie zu "Benutzer verwalten"
2. Finden Sie den Nutzer
3. Klicken Sie auf "Farben bearbeiten"
4. Wählen Sie die gewünschten Farben aus
5. Speichern Sie

---

## Buchungszeiten verwalten

### Zeitregeln konfigurieren

Sie können festlegen, wann Buchungen möglich sind:

1. Gehen Sie zu "Buchungszeiten"
2. Definieren Sie Regeln für:
   - **Wochentage**: Morgen-, Mittags-, Abendzeiten
   - **Wochenende/Feiertage**: Angepasste Zeiten
3. Markieren Sie Zeitblöcke als:
   - **Erlaubt**: Buchungen möglich
   - **Gesperrt**: Keine Buchungen (z.B. Fütterungszeiten)
   - **Genehmigungspflichtig**: Admin muss Buchung genehmigen

### Feiertage verwalten

1. Gehen Sie zu "Feiertage"
2. Sie können:
   - Feiertage aus der API abrufen (deutsche Feiertage)
   - Eigene Feiertage hinzufügen
   - Feiertage aktivieren/deaktivieren

### Buchungen genehmigen

Wenn Buchungen in genehmigungspflichtigen Zeitfenstern erstellt werden:

1. Gehen Sie zu "Buchungen zur Genehmigung"
2. Prüfen Sie die ausstehenden Buchungen
3. Klicken Sie "Genehmigen" oder "Ablehnen"
4. Optional: Fügen Sie eine Nachricht hinzu
5. Nutzer erhält E-Mail-Benachrichtigung

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

## Administrator-Verwaltung (Nur Super Admin)

**Wichtig:** Nur der Super Admin kann andere Benutzer zu Administratoren ernennen oder Admin-Rechte entziehen.

### Unterschied: Super Admin vs. Admin

- **Super Admin** (ID=1):
  - Kann andere Admins ernennen und herabstufen
  - Kann nicht gelöscht oder deaktiviert werden
  - Wird beim ersten Start automatisch erstellt
  - Nur eine Person kann Super Admin sein

- **Regulärer Admin**:
  - Hat Zugriff auf alle Admin-Funktionen
  - Kann KEINE anderen Admins verwalten
  - Kann vom Super Admin herabgestuft werden

### Benutzer zum Admin ernennen

**Voraussetzungen:**
- Sie müssen als Super Admin angemeldet sein
- Der Benutzer muss aktiv und verifiziert sein
- Der Benutzer darf noch kein Admin sein

**Vorgang:**
1. Gehen Sie zu **"Benutzerverwaltung"** (admin-users.html)
2. Finden Sie den Benutzer in der Liste
3. In der Spalte "Rolle" sehen Sie "Benutzer"
4. Klicken Sie auf **"Zu Admin ernennen"**
5. Bestätigen Sie die Aktion im Dialog:
   ```
   Möchten Sie [Name] wirklich zum Admin ernennen?

   Admins haben Zugriff auf alle Verwaltungsfunktionen.
   ```
6. Nach Bestätigung:
   - Benutzer erhält sofort Admin-Rechte
   - Badge ändert sich zu "Admin"
   - Benutzer kann sich sofort mit Admin-Rechten anmelden
   - Alle Admin-Seiten sind zugänglich

**Wichtig:** Es gibt KEINE automatische E-Mail-Benachrichtigung. Informieren Sie den neuen Admin persönlich!

### Admin-Rechte entziehen

**Voraussetzungen:**
- Sie müssen als Super Admin angemeldet sein
- Der Benutzer muss ein regulärer Admin sein (nicht Super Admin)
- Sie können sich selbst nicht herabstufen

**Vorgang:**
1. Gehen Sie zu **"Benutzerverwaltung"**
2. Finden Sie den Admin in der Liste
3. In der Spalte "Rolle" sehen Sie "Admin"
4. Klicken Sie auf **"Admin entfernen"**
5. Bestätigen Sie die Aktion im Dialog:
   ```
   Möchten Sie [Name] wirklich die Admin-Rechte entziehen?

   Der Benutzer wird zu einem normalen Benutzer herabgestuft.
   ```
6. Nach Bestätigung:
   - Benutzer verliert sofort Admin-Rechte
   - Badge ändert sich zu "Benutzer"
   - Zugriff auf Admin-Seiten wird gesperrt
   - Aktive Admin-Session wird ungültig

**Wichtig:** Informieren Sie den betroffenen Benutzer über die Änderung!

### Super Admin Passwort ändern

**Wichtig:** Das Super Admin Passwort kann NICHT über die Web-Oberfläche geändert werden!

**Vorgang:**
1. Öffnen Sie die Datei **`SUPER_ADMIN_CREDENTIALS.txt`** im Hauptverzeichnis
2. Die Datei sieht so aus:
   ```
   =============================================================
   GASSIGEHER - SUPER ADMIN CREDENTIALS
   =============================================================

   EMAIL: admin@yourshelter.com
   PASSWORD: aktuelles-passwort-hier

   CREATED: 2025-01-23 10:00:00
   LAST UPDATED: 2025-01-23 10:00:00
   =============================================================
   ```
3. Ändern Sie die Zeile `PASSWORD:` zu Ihrem neuen Passwort
4. Speichern Sie die Datei
5. Starten Sie den Gassigeher-Server neu:
   ```bash
   systemctl restart gassigeher  # Linux
   ```
6. Die Datei wird automatisch aktualisiert mit:
   ```
   LAST UPDATED: [neues Datum]
   PASSWORD CHANGE CONFIRMED: ✓
   ```
7. Melden Sie sich mit dem neuen Passwort an

**Sicherheitshinweise:**
- Verwenden Sie ein starkes Passwort (mind. 12 Zeichen)
- Speichern Sie die Datei sicher (wird automatisch mit 600 Rechten erstellt)
- Die Datei ist in `.gitignore` und wird NICHT ins Git übertragen
- Bewahren Sie eine Kopie an sicherem Ort auf

### Wer ist Super Admin?

**Bei frischer Installation:**
- Der Super Admin wird automatisch erstellt
- E-Mail-Adresse kommt aus `.env` Datei: `SUPER_ADMIN_EMAIL`
- Passwort wird zufällig generiert und angezeigt:
  - In der Konsole beim ersten Start
  - In der Datei `SUPER_ADMIN_CREDENTIALS.txt`

**Bei bestehender Installation:**
- Super Admin ist der Benutzer mit ID=1
- Wurde manuell in der Datenbank gesetzt

**So prüfen Sie, ob Sie Super Admin sind:**
1. Melden Sie sich an
2. Gehen Sie zu "Benutzerverwaltung"
3. Wenn Sie die Buttons "Zu Admin ernennen" und "Admin entfernen" sehen, sind Sie Super Admin
4. Wenn nicht, sind Sie regulärer Admin

### Tipps zur Admin-Verwaltung

**Wie viele Admins brauchen Sie?**
- **Kleines Tierheim (< 50 Nutzer):** 1-2 Admins reichen
- **Mittleres Tierheim (50-200 Nutzer):** 2-4 Admins empfohlen
- **Großes Tierheim (> 200 Nutzer):** 3-6 Admins je nach Arbeitsaufwand

**Wer sollte Admin werden?**
- ✅ Vertrauenswürdige Mitarbeiter
- ✅ Langfristig beim Tierheim tätig
- ✅ Technisch versiert (grundlegendes Computer-Wissen)
- ✅ Verantwortungsbewusst im Umgang mit Nutzerdaten
- ❌ Nicht: Ehrenamtliche ohne feste Bindung
- ❌ Nicht: Unerfahrene Nutzer

**Best Practices:**
- Ernennen Sie Admins nur nach Bedarf
- Dokumentieren Sie, wer Admin ist und warum
- Überprüfen Sie regelmäßig (alle 6 Monate), ob alle Admins noch benötigt werden
- Entziehen Sie Admin-Rechte sofort, wenn jemand das Tierheim verlässt
- Informieren Sie neue Admins über ihre Verantwortung
- Geben Sie neuen Admins eine Einführung in die Admin-Funktionen

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

### Hundefarben zuweisen

Definieren Sie Farben basierend auf den Anforderungen Ihres Tierheims. Typische Einteilung:

**Einstiegsfarbe** (z.B. Grün):
- Ruhige, gut erzogene Hunde
- Kleine bis mittelgroße Hunde
- Hunde ohne besondere Anforderungen

**Mittlere Farbe:**
- Energiegeladene Hunde
- Große Hunde
- Hunde mit leichten besonderen Bedürfnissen

**Fortgeschrittene Farbe:**
- Sehr große oder kräftige Hunde
- Hunde mit Verhaltensproblemen
- Hunde, die besondere Erfahrung erfordern

### Farbanfragen genehmigen

**Empfohlene Kriterien:**
- Mindestens 10 abgeschlossene Spaziergänge
- Keine Stornierungen in letzter Minute
- Positive Notizen
- Zuverlässigkeit über Zeit bewiesen

**Für fortgeschrittene Farben:**
- Mindestens 25 abgeschlossene Spaziergänge
- Erfahrung mit Hunden mittlerer Schwierigkeit
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

2. **Farbanfragen bearbeiten**:
   - Alle ausstehenden Anfragen prüfen
   - Spaziergangshistorie bewerten

3. **Statistiken analysieren**:
   - Beliebte Hunde identifizieren
   - Buchungstrends erkennen

4. **Genehmigungspflichtige Buchungen**:
   - Alle ausstehenden Buchungen prüfen
   - Zeitnah genehmigen oder ablehnen

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

2. **Nutzer fehlt die Farbe**
   - Prüfen: Nutzer-Farben und Hund-Farbe
   - Lösung: Farbanfrage genehmigen oder dem Nutzer die Farbe zuweisen

3. **Datum gesperrt**
   - Prüfen: Gesperrte Tage und Feiertage
   - Lösung: Sperrung aufheben, falls angebracht

4. **Doppelbuchung**
   - Prüfen: Buchungen für das Datum
   - Lösung: Anderer Zeitpunkt vorschlagen

5. **Zeitfenster gesperrt**
   - Prüfen: Buchungszeit-Regeln
   - Lösung: Zeitregel anpassen oder anderen Zeitpunkt wählen

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
- [ ] Farbanfrage genehmigt (Test)
- [ ] Buchungszeiten verstanden
- [ ] Feiertage konfiguriert
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
