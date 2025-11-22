# Dog Photo Upload - End-to-End Test Plan

**Created:** 2025-01-21
**Version:** 1.0
**Phase:** 6 - Testing & Documentation
**Scope:** Phases 2-5 (Backend processing Phase 1 pending)

---

## Test Overview

This document provides a comprehensive end-to-end testing plan for the dog photo upload feature, covering all implemented phases (2-5).

### Phases Covered

- ✅ **Phase 2:** Database Schema Updates
- ✅ **Phase 3:** Frontend Upload UI
- ✅ **Phase 4:** Placeholder Strategy
- ✅ **Phase 5:** Display Optimization

### Test Environment

**Prerequisites:**
- Application running (`go run cmd/server/main.go`)
- Admin user created and verified
- Test database with sample dogs
- Modern browser (Chrome, Firefox, or Safari)

---

## Test Categories

### 1. Database Tests (Phase 2)

**Test Suite:** `scripts/test_phase2.go`

**Run:**
```bash
go run scripts/test_phase2.go
```

**Expected:** 7/7 tests passing

**Tests:**
- [x] Database migration successful
- [x] photo_thumbnail column created
- [x] Dog creation with photos works
- [x] Dog creation without photos works (backward compatible)
- [x] Dog retrieval includes photo fields
- [x] Dog update with photos works
- [x] FindAll returns photo fields

---

### 2. Frontend Integration Tests (Phase 3)

**Test Suite:** `scripts/test_photo_upload_e2e.html`

**Run:**
```
1. Start server: go run cmd/server/main.go
2. Open: http://localhost:8080/scripts/test_photo_upload_e2e.html
3. Verify all tests pass
```

**Expected:** 20+ tests passing

**Categories:**
- DogPhotoManager class functionality
- File validation
- Upload methods
- Helper function integration

---

### 3. Performance Tests (Phase 5)

**Test Suite:** `scripts/test_phase5_performance.html`

**Run:**
```
1. Start server: go run cmd/server/main.go
2. Open: http://localhost:8080/scripts/test_phase5_performance.html
3. Verify all 18 tests pass
```

**Expected:** 18/18 tests passing

**Tests:**
- Helper functions
- Lazy loading
- Skeleton loader
- Responsive images
- Calendar optimization
- Performance with 20 dogs

---

## Manual Testing Checklist

### Test 1: Photo Upload (New Dog)

**Objective:** Verify photo upload when creating a new dog

**Steps:**
1. Login as admin
2. Navigate to `/admin-dogs.html`
3. Click "Hund hinzufügen"
4. Fill in required fields (name, breed, size, age, category)
5. In "Foto" section, click upload zone
6. Select a JPEG or PNG file (<10MB)
7. **Verify:** Preview appears
8. **Verify:** × button visible on preview
9. Click "Speichern"
10. **Verify:** Success message appears
11. **Verify:** Dog appears in list with photo
12. **Verify:** Photo displays correctly

**Expected Results:**
- ✅ Preview shows before upload
- ✅ Upload succeeds
- ✅ Photo visible in dog list
- ✅ No errors

**Test Data:**
- Use `scripts/sample_photos/dog_sample_1.svg`

---

### Test 2: Photo Upload (Drag & Drop)

**Objective:** Verify drag and drop functionality

**Steps:**
1. Login as admin
2. Navigate to `/admin-dogs.html`
3. Click "Hund hinzufügen"
4. Fill in required fields
5. **Drag** an image file over the upload zone
6. **Verify:** Zone highlights (green border, background tint)
7. **Drop** the file
8. **Verify:** Preview appears immediately
9. **Verify:** Upload prompt hidden
10. Click "Speichern"
11. **Verify:** Upload succeeds

**Expected Results:**
- ✅ Drag-over highlight works
- ✅ Drop triggers preview
- ✅ Upload succeeds
- ✅ Professional user experience

**Test Data:**
- Use `scripts/sample_photos/dog_sample_2.svg`

---

### Test 3: Photo Upload (Edit Existing Dog)

**Objective:** Add photo to dog that doesn't have one

**Steps:**
1. Login as admin
2. Navigate to `/admin-dogs.html`
3. Find dog without photo (shows placeholder)
4. Click ✏️ (edit)
5. **Verify:** Upload zone visible
6. Select/drag photo
7. **Verify:** Preview appears
8. Click "Speichern"
9. **Verify:** Photo uploaded
10. **Verify:** Dog now shows real photo (not placeholder)

**Expected Results:**
- ✅ Can add photo to existing dog
- ✅ Placeholder replaced with real photo
- ✅ Photo persists on page reload

---

### Test 4: Photo Change (Replace Existing)

**Objective:** Change photo of dog that already has one

**Steps:**
1. Login as admin
2. Navigate to `/admin-dogs.html`
3. Find dog with photo
4. Click ✏️ (edit)
5. **Verify:** Current photo displayed
6. **Verify:** "Foto ändern" button visible
7. Click "Foto ändern"
8. **Verify:** Upload zone appears
9. Select/drag new photo
10. **Verify:** Preview of new photo appears
11. Click "Speichern"
12. **Verify:** Photo updated
13. **Verify:** Old photo replaced with new photo

**Expected Results:**
- ✅ Current photo visible in edit mode
- ✅ Can change to new photo
- ✅ Old photo replaced
- ✅ Update succeeds

---

### Test 5: File Validation (Invalid Type)

**Objective:** Verify rejection of non-image files

**Steps:**
1. Login as admin
2. Navigate to `/admin-dogs.html`
3. Click "Hund hinzufügen"
4. Try to upload a .txt, .pdf, or .gif file
5. **Verify:** Error message appears (German)
6. **Verify:** Message: "Nur JPEG und PNG Dateien sind erlaubt"
7. **Verify:** No preview shown
8. **Verify:** File not uploaded

**Expected Results:**
- ✅ Invalid files rejected
- ✅ German error message
- ✅ No upload occurs

**Test Data:**
- Create a .txt file
- Or try .gif, .bmp, .webp

---

### Test 6: File Validation (Too Large)

**Objective:** Verify rejection of oversized files

**Steps:**
1. Login as admin
2. Navigate to `/admin-dogs.html`
3. Click "Hund hinzufügen"
4. Try to upload a file >10MB
5. **Verify:** Error message appears
6. **Verify:** Message includes "zu groß" and "10MB"
7. **Verify:** No preview shown
8. **Verify:** File not uploaded

**Expected Results:**
- ✅ Large files rejected
- ✅ German error message with size limit
- ✅ No upload occurs

**Test Data:**
- Create a large test file (>10MB)

---

### Test 7: Preview Functionality

**Objective:** Verify photo preview before upload

**Steps:**
1. Login as admin
2. Navigate to `/admin-dogs.html`
3. Click "Hund hinzufügen"
4. Select valid image file
5. **Verify:** Preview appears within 500ms
6. **Verify:** Preview shows correct image
7. **Verify:** × button visible
8. Click × button
9. **Verify:** Preview cleared
10. **Verify:** Upload prompt visible again
11. **Verify:** Can select different file

**Expected Results:**
- ✅ Preview generated instantly
- ✅ Correct image shown
- ✅ Can clear preview
- ✅ Can select again

---

### Test 8: Placeholder Display (Dog Without Photo)

**Objective:** Verify placeholder images for dogs without photos

**Steps:**
1. Navigate to `/dogs.html` (as regular user)
2. Find dogs without photos
3. **Verify:** SVG placeholder displayed (not emoji)
4. **Verify:** Green dogs show green placeholder
5. **Verify:** Blue dogs show blue placeholder
6. **Verify:** Orange dogs show orange placeholder
7. **Verify:** Placeholder has "G", "B", or "O" badge
8. **Verify:** Text says "Kein Foto (Kategorie)"
9. Resize browser window
10. **Verify:** Placeholder scales smoothly
11. **Verify:** No pixelation at any size

**Expected Results:**
- ✅ Professional SVG placeholders
- ✅ Category-specific colors
- ✅ Scalable (vector graphics)
- ✅ Consistent across browsers

---

### Test 9: Lazy Loading

**Objective:** Verify images load only when visible

**Steps:**
1. Navigate to `/dogs.html`
2. Open DevTools (F12)
3. Go to Network tab → Filter: Images
4. Clear network log
5. Scroll to page top (don't scroll down yet)
6. **Verify:** Only first 3-5 dog images loaded
7. **Verify:** Images below fold not loaded yet
8. Scroll down slowly
9. **Verify:** Images load as they enter viewport
10. **Verify:** Loading indicated in Network tab

**Expected Results:**
- ✅ Only visible images load initially
- ✅ Images load on scroll
- ✅ Bandwidth saved for off-screen images
- ✅ Smooth loading (no jank)

---

### Test 10: Skeleton Loader

**Objective:** Verify skeleton loader appears during image load

**Steps:**
1. Open DevTools → Network → Throttling → Slow 3G
2. Navigate to `/dogs.html`
3. Clear cache (Ctrl+Shift+R)
4. **Verify:** Animated shimmer gradient appears
5. **Verify:** Shimmer moves left-to-right
6. Wait for images to load
7. **Verify:** Shimmer stops when image loads
8. **Verify:** Image fades in smoothly
9. **Verify:** SVG placeholders have no skeleton (instant)

**Expected Results:**
- ✅ Skeleton shown during load
- ✅ Professional shimmer animation
- ✅ Smooth transition to image
- ✅ No skeleton for placeholders

---

### Test 11: Fade-in Animation

**Objective:** Verify smooth fade-in when images load

**Steps:**
1. Navigate to `/dogs.html`
2. Clear cache
3. Watch images as they load
4. **Verify:** Images start invisible
5. **Verify:** Images fade in over ~300ms
6. **Verify:** Not jarring "pop-in"
7. Reload page (images cached)
8. **Verify:** Cached images appear instantly (no animation)

**Expected Results:**
- ✅ Smooth fade-in for new images
- ✅ Instant appearance for cached images
- ✅ Professional user experience

---

### Test 12: Responsive Images (Mobile/Desktop)

**Objective:** Verify mobile gets thumbnails, desktop gets full images

**Steps:**
1. Navigate to `/dogs.html` (desktop view)
2. Open DevTools → Network → Images
3. Clear network log
4. Reload page
5. **Verify:** Full-size images loaded (dog_X_full.jpg)
6. Note file sizes (~150KB or actual size)
7. Switch to mobile view (DevTools → Device Toolbar → iPhone)
8. Clear network log
9. Reload page
10. **Verify:** Thumbnail images loaded (dog_X_thumb.jpg)
11. **Verify:** Smaller file sizes (~30KB)

**Expected Results:**
- ✅ Desktop loads full images
- ✅ Mobile loads thumbnails
- ✅ 70-80% bandwidth savings on mobile
- ✅ Automatic switching

**Note:** If Phase 1 not implemented, thumbnails may not exist yet (will fall back to full images)

---

### Test 13: Image Preloading

**Objective:** Verify first 3 images load immediately

**Steps:**
1. Navigate to `/dogs.html`
2. Open DevTools → Network → Images
3. Clear network log
4. Reload page
5. Look at first 3-5 image requests
6. **Verify:** First 3 have "Initiator: preload"
7. **Verify:** These load before other images
8. **Verify:** Higher priority in queue

**Expected Results:**
- ✅ First 3 images preloaded
- ✅ Load before other images
- ✅ Instant display above fold

---

### Test 14: Calendar View with Photos

**Objective:** Verify dog photos appear in calendar

**Steps:**
1. Navigate to `/calendar.html`
2. **Verify:** Dog names have small circular photos
3. **Verify:** Photos are 40x40 circles
4. **Verify:** Dogs with photos show photos
5. **Verify:** Dogs without photos show category placeholders
6. **Verify:** Layout not broken
7. Test on mobile
8. **Verify:** Photos still visible and proportional

**Expected Results:**
- ✅ Circular dog photos in calendar
- ✅ Placeholders for dogs without photos
- ✅ Mobile-friendly
- ✅ Better visual recognition

---

### Test 15: Accessibility (Screen Reader)

**Objective:** Verify accessibility for visually impaired users

**Steps:**
1. Enable screen reader (NVDA, JAWS, or VoiceOver)
2. Navigate to `/dogs.html`
3. Tab through dog cards
4. **Verify:** Dog with photo announced as "Image: Name (Breed)"
5. **Verify:** Dog without photo announced as "Image: Kein Foto für Name"
6. **Verify:** No "unlabeled image" announcements
7. **Verify:** All interactive elements keyboard accessible

**Expected Results:**
- ✅ Meaningful alt text for all images
- ✅ No accessibility errors
- ✅ Keyboard navigation works
- ✅ Screen reader friendly

---

### Test 16: Reduced Motion

**Objective:** Verify respect for motion preferences

**Steps:**
1. Enable "Reduce motion" in OS settings:
   - Windows: Settings → Accessibility → Visual effects → Off
   - Mac: System Preferences → Accessibility → Display → Reduce motion
2. Navigate to `/dogs.html`
3. Clear cache and reload
4. **Verify:** No skeleton animation
5. **Verify:** No fade-in animation
6. **Verify:** Images appear instantly
7. **Verify:** Placeholders show immediately

**Expected Results:**
- ✅ No animations when reduced motion enabled
- ✅ Instant appearance (no transitions)
- ✅ Respects user preferences
- ✅ WCAG 2.1 compliant

---

### Test 17: Cross-Browser Compatibility

**Objective:** Verify functionality across browsers

**Browsers to Test:**
- Chrome/Edge (Chromium)
- Firefox
- Safari (Mac/iOS)
- Mobile browsers (iOS Safari, Chrome Android)

**For Each Browser:**
1. Navigate to `/admin-dogs.html`
2. **Verify:** Upload UI displays correctly
3. Test file selection
4. Test drag and drop
5. **Verify:** Preview works
6. Navigate to `/dogs.html`
7. **Verify:** Placeholders display correctly
8. **Verify:** Lazy loading works
9. **Verify:** Fade-in works

**Expected Results:**
- ✅ Consistent appearance across browsers
- ✅ All functionality works
- ✅ No console errors
- ✅ Professional UX in all browsers

---

### Test 18: Mobile Responsiveness

**Objective:** Verify mobile device compatibility

**Test Devices:**
- iPhone (Safari)
- Android Phone (Chrome)
- iPad/Tablet
- Small screen (320px width)

**For Each Device:**
1. Navigate to `/dogs.html`
2. **Verify:** Dog cards display correctly
3. **Verify:** Photos/placeholders scale properly
4. **Verify:** Touch interactions work
5. Navigate to `/admin-dogs.html` (admin only)
6. **Verify:** Upload UI works on mobile
7. **Verify:** Drag and drop works (if supported)
8. **Verify:** File picker opens
9. **Verify:** Preview displays correctly

**Expected Results:**
- ✅ Mobile-friendly layouts
- ✅ Touch interactions work
- ✅ Photos scale appropriately
- ✅ Upload works on mobile

---

### Test 19: Performance Under Load

**Objective:** Verify performance with many dogs

**Setup:**
1. Use test data script to create 20+ dogs
2. Some with photos, some without

**Steps:**
1. Navigate to `/dogs.html`
2. Open DevTools → Performance → Start recording
3. Reload page
4. Stop recording after page loads
5. **Analyze:**
   - First Contentful Paint (target: <1.5s)
   - Largest Contentful Paint (target: <2.5s)
   - Time to Interactive (target: <3s)
   - Layout shifts (target: minimal)
6. **Verify:** Smooth scrolling (no frame drops)
7. **Verify:** No "jank" when loading images

**Expected Results:**
- ✅ Page loads in <3s
- ✅ First paint <1.5s
- ✅ Smooth scrolling (60fps)
- ✅ No layout shifts

**Metrics:**
- Page load: <3s
- Render time: <100ms
- Smooth scrolling: 60fps
- No jank

---

### Test 20: Error Handling

**Objective:** Verify graceful error handling

**Test Cases:**

#### Test 20a: Upload to Non-Existent Dog
```
Steps:
1. Try to upload via API: POST /api/dogs/99999/photo
2. Verify: 404 Not Found
3. Verify: Error message in response
```

#### Test 20b: Upload Without Authentication
```
Steps:
1. Logout
2. Try to upload photo
3. Verify: 401 Unauthorized or redirect
```

#### Test 20c: Upload as Non-Admin
```
Steps:
1. Login as regular user (not admin)
2. Try to access /admin-dogs.html
3. Verify: Access denied
```

#### Test 20d: Network Error During Upload
```
Steps:
1. Start upload
2. Disconnect network mid-upload
3. Verify: Error message appears
4. Verify: Dog still saved (graceful fallback)
5. Verify: Can retry photo upload
```

#### Test 20e: Corrupted Image File
```
Steps:
1. Create corrupted JPEG (rename .txt to .jpg)
2. Try to upload
3. Verify: Error message (may be backend validation)
```

**Expected Results:**
- ✅ All errors handled gracefully
- ✅ German error messages
- ✅ No application crashes
- ✅ Clear user feedback

---

### Test 21: Placeholder Visual Quality

**Objective:** Verify placeholder appearance

**Steps:**
1. Navigate to `/dogs.html`
2. Find dogs without photos
3. **Verify for each category:**

**Green Dog (No Photo):**
- [ ] Shows dog-placeholder-green.svg
- [ ] Light green background
- [ ] Green border
- [ ] Green dog silhouette
- [ ] "G" badge in top-left
- [ ] Text: "Kein Foto (Grün)"

**Blue Dog (No Photo):**
- [ ] Shows dog-placeholder-blue.svg
- [ ] Light blue background
- [ ] Blue border
- [ ] Blue dog silhouette
- [ ] "B" badge in top-left
- [ ] Text: "Kein Foto (Blau)"

**Orange Dog (No Photo):**
- [ ] Shows dog-placeholder-orange.svg
- [ ] Light orange background
- [ ] Orange border
- [ ] Orange dog silhouette
- [ ] "O" badge in top-left
- [ ] Text: "Kein Foto (Orange)"

4. Resize browser window
5. **Verify:** Placeholders scale smoothly
6. **Verify:** No pixelation at any size
7. **Verify:** Proportions maintained

**Expected Results:**
- ✅ All 3 category placeholders display correctly
- ✅ Professional appearance
- ✅ Scalable (SVG)
- ✅ Category differentiation clear

---

### Test 22: Integration Flow (Complete User Journey)

**Objective:** Test complete workflow from create to display

**Scenario:** Admin adds new dog with photo, user books walk

**Steps:**

**Part 1: Admin Creates Dog with Photo**
1. Login as admin
2. Navigate to `/admin-dogs.html`
3. Click "Hund hinzufügen"
4. Fill in:
   - Name: "Bruno"
   - Breed: "Boxer"
   - Size: "large"
   - Age: 4
   - Category: "green"
5. Drag photo into upload zone
6. Verify preview
7. Click "Speichern"
8. **Verify:** Success message
9. **Verify:** Bruno appears in list with photo

**Part 2: User Views Dog**
10. Logout
11. Login as regular user (green level)
12. Navigate to `/dogs.html`
13. **Verify:** Bruno visible in list
14. **Verify:** Bruno's photo displayed
15. **Verify:** Photo is thumbnail (on mobile)
16. **Verify:** Photo is full size (on desktop)
17. Click on Bruno's card
18. **Verify:** Booking modal opens
19. **Verify:** Photo shown in modal

**Part 3: User Books Walk**
20. Select date, time
21. Create booking
22. **Verify:** Booking successful
23. Navigate to `/dashboard.html`
24. **Verify:** Booking visible
25. **Optional:** Dog photo shown with booking

**Expected Results:**
- ✅ Complete workflow works end-to-end
- ✅ Photo visible at all stages
- ✅ No errors or breaks
- ✅ Professional user experience

---

## Test Data Setup

### Automated Setup

**Run:**
```bash
# 1. Copy sample photos to uploads directory
.\scripts\setup_sample_photos.ps1

# 2. Generate test data
.\scripts\gentestdata.ps1

# 3. Start application
go run cmd/server/main.go
```

### Manual Setup

**Create Test Photos:**
1. Prepare 3-5 dog photos (JPEG or PNG)
2. Size: Various (100KB to 5MB)
3. Dimensions: Various (500x500 to 3000x3000)
4. Name: dog_test_1.jpg, dog_test_2.jpg, etc.

**Upload via UI:**
1. Login as admin
2. Create/edit dogs
3. Upload test photos
4. Verify photos display correctly

---

## Regression Testing

### After Each Change

**Quick Regression Checklist:**
- [ ] Existing dogs without photos still show placeholders
- [ ] Existing dogs with photos still show photos
- [ ] Upload UI still works
- [ ] Drag and drop still works
- [ ] Preview still works
- [ ] No console errors
- [ ] No visual glitches
- [ ] Mobile still works

### After Deployment

**Post-Deployment Verification:**
- [ ] All automated tests still pass
- [ ] Upload functionality works
- [ ] Photos display on all pages
- [ ] Placeholders display correctly
- [ ] Performance acceptable
- [ ] No errors in logs

---

## Performance Benchmarks

### Target Metrics

| Metric | Target | Threshold |
|--------|--------|-----------|
| **Page Load (20 dogs)** | <3s | <5s |
| **First Contentful Paint** | <1.5s | <2.5s |
| **Largest Contentful Paint** | <2.5s | <4s |
| **Time to Interactive** | <3s | <5s |
| **Render Time** | <100ms | <200ms |
| **Upload Time (1MB)** | <2s | <5s |
| **Upload Time (5MB)** | <5s | <10s |

### How to Measure

**Using Browser DevTools:**
1. Open DevTools → Lighthouse
2. Run audit (Performance category)
3. Check metrics against targets
4. Review suggestions

**Using Performance Tab:**
1. DevTools → Performance
2. Start recording
3. Reload page
4. Stop recording
5. Analyze waterfall
6. Check for bottlenecks

---

## Test Report Template

### Test Execution Report

**Date:** _____________
**Tester:** _____________
**Environment:** _____________
**Browser:** _____________

**Test Results:**

| Test # | Test Name | Status | Notes |
|--------|-----------|--------|-------|
| 1 | Photo Upload (New Dog) | ⬜ Pass / ⬜ Fail | |
| 2 | Drag & Drop | ⬜ Pass / ⬜ Fail | |
| 3 | Edit Existing Dog | ⬜ Pass / ⬜ Fail | |
| 4 | Change Photo | ⬜ Pass / ⬜ Fail | |
| 5 | Invalid File Type | ⬜ Pass / ⬜ Fail | |
| 6 | File Too Large | ⬜ Pass / ⬜ Fail | |
| 7 | Preview Functionality | ⬜ Pass / ⬜ Fail | |
| 8 | Placeholder Display | ⬜ Pass / ⬜ Fail | |
| 9 | Lazy Loading | ⬜ Pass / ⬜ Fail | |
| 10 | Skeleton Loader | ⬜ Pass / ⬜ Fail | |
| 11 | Fade-in Animation | ⬜ Pass / ⬜ Fail | |
| 12 | Responsive Images | ⬜ Pass / ⬜ Fail | |
| 13 | Image Preloading | ⬜ Pass / ⬜ Fail | |
| 14 | Calendar Photos | ⬜ Pass / ⬜ Fail | |
| 15 | Accessibility | ⬜ Pass / ⬜ Fail | |
| 16 | Reduced Motion | ⬜ Pass / ⬜ Fail | |
| 17 | Cross-Browser | ⬜ Pass / ⬜ Fail | |
| 18 | Mobile Devices | ⬜ Pass / ⬜ Fail | |
| 19 | Performance Load | ⬜ Pass / ⬜ Fail | |
| 20 | Error Handling | ⬜ Pass / ⬜ Fail | |
| 21 | Placeholder Quality | ⬜ Pass / ⬜ Fail | |
| 22 | Integration Flow | ⬜ Pass / ⬜ Fail | |

**Summary:**
- **Passed:** ___ / 22
- **Failed:** ___
- **Skipped:** ___
- **Success Rate:** ___%

**Issues Found:**
1. _____________
2. _____________

**Recommendations:**
1. _____________
2. _____________

---

## Conclusion

This E2E test plan provides comprehensive coverage of all dog photo upload functionality across Phases 2-5. Follow the checklists systematically to ensure quality and reliability before production deployment.

**Total Tests:** 22 manual + 43 automated = 65 tests

**Estimated Testing Time:** 2-3 hours for complete manual testing

**Recommended:** Run automated tests first, then selective manual testing based on results

---

**Next Steps After Testing:**
1. ✅ Complete all tests
2. ✅ Document results
3. ⏳ Implement Phase 1 (if not done)
4. ⏳ Retest with Phase 1
5. 🚀 Deploy to production

---

**Test Tools:**
- Automated: `scripts/test_phase2.go`
- Automated: `scripts/test_photo_upload_e2e.html`
- Automated: `scripts/test_phase5_performance.html`
- Manual: This document

**Documentation:**
- [Phase6_CompletionReport.md](Phase6_CompletionReport.md) - Detailed report
- [PhotoUpload_E2E_TestPlan.md](PhotoUpload_E2E_TestPlan.md) - This document
