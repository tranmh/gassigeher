# Browser Compatibility Test Script - Phase 10
## Booking Time Restrictions Feature

**Version:** 1.0
**Date:** 2025-01-23
**Purpose:** Manual browser and device testing checklist for time restrictions feature

---

## Test Environment Setup

### Browsers to Test
- ✅ **Chrome** (latest stable)
- ✅ **Firefox** (latest stable)
- ✅ **Safari** (latest stable)
- ✅ **Edge** (latest stable)

### Mobile Devices to Test
- ✅ **iPhone** (iOS Safari)
- ✅ **Android Phone** (Chrome)
- ✅ **Tablet** (iPad/Android)

### Test Credentials
- **Regular User:** green@test.com / password
- **Admin User:** admin@example.com / password
- **Server URL:** http://localhost:8080

---

## Section 1: Desktop Browser Testing

### 1.1 Booking Form (dogs.html)

**Test in Each Browser:** Chrome | Firefox | Safari | Edge

#### Test Steps:
1. ✅ Login as regular user
2. ✅ Navigate to dogs page
3. ✅ Click on a dog card
4. ✅ Click "Spaziergang buchen"
5. ✅ Select date field (input type="date")
6. ✅ Choose Monday date (weekday)
7. ✅ Verify time dropdown appears
8. ✅ Verify time slots displayed (15-minute intervals)
9. ✅ Verify blocked times NOT present (13:00-14:00, 17:00-18:00)
10. ✅ Select time: 10:00 (morning)
11. ✅ Verify warning appears: "⚠️ Vormittagsspaziergänge erfordern Admin-Genehmigung"
12. ✅ Select time: 15:00 (afternoon)
13. ✅ Verify warning disappears
14. ✅ Verify "Erlaubte Buchungszeiten" info box displays
15. ✅ Submit booking
16. ✅ Verify success message

**Expected Results:**
| Feature | Chrome | Firefox | Safari | Edge |
|---------|--------|---------|--------|------|
| Date picker works | ☐ | ☐ | ☐ | ☐ |
| Time dropdown populates | ☐ | ☐ | ☐ | ☐ |
| Warning shows/hides | ☐ | ☐ | ☐ | ☐ |
| Info box displays | ☐ | ☐ | ☐ | ☐ |
| Booking submits | ☐ | ☐ | ☐ | ☐ |

**Known Browser Issues:**
- **Safari:** Time input may display differently (use select dropdown, not time input)
- **Firefox:** Date picker format may vary by OS locale
- **Edge:** Should behave like Chrome (Chromium-based)

---

### 1.2 Admin Booking Times Page (admin-booking-times.html)

**Test in Each Browser:** Chrome | Firefox | Safari | Edge

#### Test Steps:
1. ✅ Login as admin
2. ✅ Navigate to admin-booking-times.html
3. ✅ Verify settings toggles render correctly
4. ✅ Toggle "Vormittagsspaziergänge erfordern Admin-Genehmigung"
5. ✅ Click "Einstellungen speichern"
6. ✅ Verify success message
7. ✅ Click "Wochentags" tab
8. ✅ Verify weekday rules table displays
9. ✅ Click "Wochenende/Feiertage" tab
10. ✅ Verify weekend rules table displays
11. ✅ Edit a time rule (change end time)
12. ✅ Click "Speichern" button for that rule
13. ✅ Verify inline success feedback
14. ✅ Scroll to "Feiertage verwalten"
15. ✅ Select year 2025
16. ✅ Click "Laden" button
17. ✅ Verify holiday list loads
18. ✅ Click "+ Feiertag hinzufügen"
19. ✅ Verify modal/form appears
20. ✅ Fill date and name
21. ✅ Save custom holiday
22. ✅ Verify holiday appears in list

**Expected Results:**
| Feature | Chrome | Firefox | Safari | Edge |
|---------|--------|---------|--------|------|
| Toggle switches work | ☐ | ☐ | ☐ | ☐ |
| Tab switching works | ☐ | ☐ | ☐ | ☐ |
| Time input fields work | ☐ | ☐ | ☐ | ☐ |
| Tables display correctly | ☐ | ☐ | ☐ | ☐ |
| Holiday fetch works | ☐ | ☐ | ☐ | ☐ |
| Modal/form works | ☐ | ☐ | ☐ | ☐ |

---

### 1.3 Admin Bookings Page (admin-bookings.html)

**Test in Each Browser:** Chrome | Firefox | Safari | Edge

#### Test Steps:
1. ✅ Create 2 morning bookings as regular user (different browser/tab)
2. ✅ Login as admin
3. ✅ Navigate to admin-bookings.html
4. ✅ Verify "Genehmigungsanfragen" section visible
5. ✅ Verify badge shows "2"
6. ✅ Verify pending bookings listed with details
7. ✅ Click "✓ Genehmigen" button
8. ✅ Verify success alert
9. ✅ Verify booking removed from pending list
10. ✅ Verify badge decremented to "1"
11. ✅ Click "✗ Ablehnen" on remaining booking
12. ✅ Enter rejection reason in prompt
13. ✅ Verify success alert
14. ✅ Verify section hides (count = 0)
15. ✅ Wait 30 seconds (auto-refresh test)

**Expected Results:**
| Feature | Chrome | Firefox | Safari | Edge |
|---------|--------|---------|--------|------|
| Section shows/hides | ☐ | ☐ | ☐ | ☐ |
| Badge updates | ☐ | ☐ | ☐ | ☐ |
| Approve works | ☐ | ☐ | ☐ | ☐ |
| Reject prompt works | ☐ | ☐ | ☐ | ☐ |
| Auto-refresh works | ☐ | ☐ | ☐ | ☐ |

---

### 1.4 User Dashboard (dashboard.html)

**Test in Each Browser:** Chrome | Firefox | Safari | Edge

#### Test Steps:
1. ✅ Login as regular user
2. ✅ Create morning booking (pending)
3. ✅ Navigate to dashboard
4. ✅ Verify booking card displays
5. ✅ Verify "⏳ Warte auf Admin-Genehmigung" warning box
6. ✅ Admin approves booking (different browser/tab)
7. ✅ Refresh dashboard
8. ✅ Verify warning removed
9. ✅ Create another morning booking
10. ✅ Admin rejects with reason
11. ✅ Refresh dashboard
12. ✅ Verify "✗ Abgelehnt: [reason]" alert displays

**Expected Results:**
| Feature | Chrome | Firefox | Safari | Edge |
|---------|--------|---------|--------|------|
| Booking cards display | ☐ | ☐ | ☐ | ☐ |
| Pending status shows | ☐ | ☐ | ☐ | ☐ |
| Rejected status shows | ☐ | ☐ | ☐ | ☐ |
| Styling consistent | ☐ | ☐ | ☐ | ☐ |

---

## Section 2: Mobile Device Testing

### 2.1 iPhone (iOS Safari)

#### Test Device Specs:
- **Device:** iPhone 12/13/14 (or similar)
- **OS:** iOS 15+
- **Browser:** Safari (default)
- **Screen Size:** 390 x 844 (or similar)

#### Test Steps:
1. ✅ Open http://localhost:8080 (or deployed URL)
2. ✅ Login as regular user
3. ✅ Navigate to dogs page
4. ✅ Verify dog cards responsive (stacked vertically)
5. ✅ Tap on a dog
6. ✅ Tap "Spaziergang buchen"
7. ✅ Verify booking modal/form responsive
8. ✅ Tap date field
9. ✅ Verify iOS date picker appears
10. ✅ Select date
11. ✅ Tap time dropdown
12. ✅ Verify time slots scrollable
13. ✅ Select time
14. ✅ Verify warning (if morning time)
15. ✅ Submit booking
16. ✅ Verify success message readable
17. ✅ Navigate to dashboard
18. ✅ Verify booking cards readable and properly sized
19. ✅ Test portrait orientation
20. ✅ Test landscape orientation

**Expected Results:**
| Feature | Pass | Notes |
|---------|------|-------|
| Layout responsive | ☐ | No horizontal scroll |
| Touch targets adequate | ☐ | Buttons ≥44px |
| Date picker native | ☐ | iOS wheel picker |
| Text readable | ☐ | Font size ≥16px |
| Forms usable | ☐ | Inputs accessible |
| Navigation works | ☐ | Hamburger menu? |
| Portrait mode | ☐ | Optimal layout |
| Landscape mode | ☐ | Usable layout |

---

### 2.2 Android Phone (Chrome)

#### Test Device Specs:
- **Device:** Samsung Galaxy/Pixel (or similar)
- **OS:** Android 10+
- **Browser:** Chrome (default)
- **Screen Size:** 412 x 915 (or similar)

#### Test Steps:
1. ✅ Open http://localhost:8080
2. ✅ Login as regular user
3. ✅ Navigate to dogs page
4. ✅ Verify layout responsive
5. ✅ Tap dog → book walk
6. ✅ Tap date field
7. ✅ Verify Android date picker
8. ✅ Select date
9. ✅ Tap time dropdown
10. ✅ Select time
11. ✅ Submit booking
12. ✅ Navigate to dashboard
13. ✅ Verify booking display
14. ✅ Test admin pages (if admin user)
15. ✅ Test portrait/landscape

**Expected Results:**
| Feature | Pass | Notes |
|---------|------|-------|
| Layout responsive | ☐ | Clean stacking |
| Date picker native | ☐ | Android calendar |
| Touch targets adequate | ☐ | Easy to tap |
| Performance smooth | ☐ | No lag |
| Forms usable | ☐ | Keyboard behavior |
| Admin tables scrollable | ☐ | Horizontal scroll if needed |

---

### 2.3 Tablet (iPad/Android Tablet)

#### Test Device Specs:
- **Device:** iPad (10.2" or similar) / Android tablet
- **OS:** iPadOS 15+ / Android 10+
- **Browser:** Safari / Chrome
- **Screen Size:** 810 x 1080 (or similar)

#### Test Steps:
1. ✅ Open application
2. ✅ Login as admin
3. ✅ Navigate to admin-booking-times.html
4. ✅ Verify layout uses available space
5. ✅ Verify tables don't require excessive scrolling
6. ✅ Test tab switching
7. ✅ Test time rule editing
8. ✅ Test holiday management
9. ✅ Navigate to admin-bookings.html
10. ✅ Verify pending approvals section
11. ✅ Test approve/reject actions
12. ✅ Login as regular user (different tab)
13. ✅ Test booking form
14. ✅ Test dashboard view

**Expected Results:**
| Feature | iPad | Android | Notes |
|---------|------|---------|-------|
| Admin tables optimal | ☐ | ☐ | Good use of space |
| Forms comfortable | ☐ | ☐ | Not cramped |
| Touch targets good | ☐ | ☐ | Easy interactions |
| Portrait usable | ☐ | ☐ | Readable |
| Landscape optimal | ☐ | ☐ | Best experience |

---

## Section 3: JavaScript Feature Testing

### 3.1 Core JavaScript APIs

**Test in Each Browser:** Chrome | Firefox | Safari | Edge

#### Features to Verify:

1. **Fetch API**
   ```javascript
   // Check browser console for errors
   fetch('/api/booking-times/available?date=2025-01-27')
     .then(r => r.json())
     .then(d => console.log('Fetch works:', d))
   ```
   - Chrome: ☐ Works
   - Firefox: ☐ Works
   - Safari: ☐ Works
   - Edge: ☐ Works

2. **LocalStorage**
   ```javascript
   // Check token storage
   console.log('Token:', localStorage.getItem('gassigeher_token'))
   ```
   - Chrome: ☐ Works
   - Firefox: ☐ Works
   - Safari: ☐ Works (check private browsing mode)
   - Edge: ☐ Works

3. **Date/Time Handling**
   ```javascript
   // Check date parsing
   const d = new Date('2025-01-27')
   console.log('Date parsed:', d)
   ```
   - Chrome: ☐ Works
   - Firefox: ☐ Works
   - Safari: ☐ Works
   - Edge: ☐ Works

4. **Array Methods (ES6+)**
   ```javascript
   // Check modern JS support
   const slots = ['09:00', '09:15', '09:30']
   const filtered = slots.filter(s => s >= '09:15')
   console.log('Filter works:', filtered)
   ```
   - Chrome: ☐ Works
   - Firefox: ☐ Works
   - Safari: ☐ Works
   - Edge: ☐ Works

---

## Section 4: CSS Feature Testing

### 4.1 Layout Features

**Test in Each Browser:** Chrome | Firefox | Safari | Edge

#### CSS Features to Verify:

1. **CSS Grid** (if used)
   - Chrome: ☐ Renders correctly
   - Firefox: ☐ Renders correctly
   - Safari: ☐ Renders correctly
   - Edge: ☐ Renders correctly

2. **Flexbox**
   - Chrome: ☐ Renders correctly
   - Firefox: ☐ Renders correctly
   - Safari: ☐ Renders correctly (check -webkit- prefixes)
   - Edge: ☐ Renders correctly

3. **CSS Variables** (if used)
   ```css
   :root {
     --primary-color: #82b965;
   }
   ```
   - Chrome: ☐ Applied
   - Firefox: ☐ Applied
   - Safari: ☐ Applied
   - Edge: ☐ Applied

4. **Transitions/Animations**
   - Chrome: ☐ Smooth
   - Firefox: ☐ Smooth
   - Safari: ☐ Smooth
   - Edge: ☐ Smooth

---

## Section 5: Input Field Testing

### 5.1 Date Input

**Browser-Specific Behavior:**

| Browser | Input Type | Fallback Needed |
|---------|------------|-----------------|
| Chrome | Native picker | No |
| Firefox | Native picker | No |
| Safari | Native picker | No |
| Edge | Native picker | No |

**Test:**
- ✅ Open booking form
- ✅ Click date field
- ✅ Verify picker appears
- ✅ Select date
- ✅ Verify format displayed (YYYY-MM-DD)

---

### 5.2 Time Input

**Browser-Specific Behavior:**

| Browser | Input Type | Fallback Needed |
|---------|------------|-----------------|
| Chrome | Native picker | No |
| Firefox | Native picker | No |
| Safari | ⚠️ May not support | Use select dropdown |
| Edge | Native picker | No |

**Current Implementation:**
- ✅ Uses `<select>` dropdown (not `<input type="time">`)
- ✅ Works consistently across all browsers
- ✅ No fallback needed

---

## Section 6: Network Testing

### 6.1 API Requests

**Test in Browser DevTools (Network Tab):**

1. ✅ Open DevTools → Network tab
2. ✅ Perform booking with time validation
3. ✅ Verify requests:
   - GET `/api/booking-times/available?date=...` → 200 OK
   - POST `/api/bookings` → 201 Created
4. ✅ Check response format (JSON)
5. ✅ Check Authorization header present
6. ✅ Check error handling (try invalid time)

**Expected in All Browsers:**
- ✅ Requests complete successfully
- ✅ Response times < 500ms
- ✅ No CORS errors
- ✅ No console errors

---

## Section 7: Accessibility Testing

### 7.1 Keyboard Navigation

**Test in Each Browser:**

1. ✅ Tab through booking form
2. ✅ Verify all inputs reachable
3. ✅ Verify focus indicators visible
4. ✅ Press Enter to submit
5. ✅ Tab through admin time rules table
6. ✅ Verify keyboard-accessible

---

### 7.2 Screen Reader (Optional)

**Test with:**
- Windows: NVDA (free)
- Mac: VoiceOver (built-in)

**Check:**
- ✅ Form labels announced
- ✅ Error messages announced
- ✅ Button purposes clear

---

## Section 8: Performance Testing

### 8.1 Page Load Time

**Test in Each Browser (DevTools → Performance):**

| Page | Chrome | Firefox | Safari | Edge |
|------|--------|---------|--------|------|
| dogs.html | ☐ <2s | ☐ <2s | ☐ <2s | ☐ <2s |
| dashboard.html | ☐ <2s | ☐ <2s | ☐ <2s | ☐ <2s |
| admin-booking-times.html | ☐ <2s | ☐ <2s | ☐ <2s | ☐ <2s |

---

### 8.2 Time Slot Generation

**Test:**
1. ✅ Open booking form
2. ✅ Select date
3. ✅ Measure time until dropdown populates
4. ✅ Verify < 500ms

---

## Section 9: Edge Cases

### 9.1 Slow Network

**Simulate in DevTools:**
1. ✅ Open DevTools → Network
2. ✅ Throttle to "Slow 3G"
3. ✅ Test booking flow
4. ✅ Verify loading indicators shown
5. ✅ Verify no timeouts/errors

---

### 9.2 Offline Mode

**Test:**
1. ✅ Open application
2. ✅ Disconnect network
3. ✅ Try to submit booking
4. ✅ Verify graceful error message
5. ✅ Reconnect network
6. ✅ Retry → should work

---

## Section 10: Bug Reporting

### Found Issues Template

```
**Browser:** [Chrome/Firefox/Safari/Edge]
**Version:** [Browser version]
**OS:** [Windows/Mac/Linux/iOS/Android]
**Device:** [Desktop/iPhone/Android/Tablet]

**Issue:**
[Clear description]

**Steps to Reproduce:**
1. Step 1
2. Step 2
3. Step 3

**Expected:**
[What should happen]

**Actual:**
[What actually happens]

**Screenshot:**
[Attach screenshot if applicable]

**Console Errors:**
[Paste any console errors]

**Workaround:**
[If available]
```

---

## Test Sign-Off

### Desktop Browsers
- ☐ Chrome - Tested by: _____________ Date: _______
- ☐ Firefox - Tested by: _____________ Date: _______
- ☐ Safari - Tested by: _____________ Date: _______
- ☐ Edge - Tested by: _____________ Date: _______

### Mobile Devices
- ☐ iPhone - Tested by: _____________ Date: _______
- ☐ Android - Tested by: _____________ Date: _______
- ☐ Tablet - Tested by: _____________ Date: _______

### Overall Result
- ☐ All critical features work across all browsers
- ☐ All critical features work on mobile devices
- ☐ Minor issues documented (non-blocking)
- ☐ Major issues resolved or documented

**Test Lead Sign-Off:** _________________________ Date: __________

---

**Phase 10 Status:** ✅ READY FOR TESTING

**Notes:**
- This is a manual testing script - requires human testers
- Estimated time: 2-3 hours for complete coverage
- Can be parallelized across multiple testers
- Focus on critical path first (booking form + admin approval)
