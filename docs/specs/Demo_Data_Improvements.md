# Demo Data Improvements Specification

## Overview

This specification outlines improvements to the demo tenant seeding data to make the demo site more realistic and visually appealing. A live demo is more effective than screenshots for building trust with potential users.

## Goal

Make the demo site at `demo.gassigeher.org` look alive and realistic, showcasing the full capabilities of the Gassigeher booking system.

## Requirements

### 1. Dog Photos

**Current State:** Demo dogs are seeded without photos or with placeholder images.

**Target State:** Add realistic dog photos to all seeded demo dogs.

**Implementation:**
- Source royalty-free dog images (e.g., Unsplash, Pexels)
- Include variety: different breeds, sizes, and colors
- Store images in `internal/static/frontend/uploads/dogs/` or use URLs
- Update seed data to reference these images

**Suggested Dogs (5-8 dogs with variety):**
| Name | Breed | Color Category | Photo Description |
|------|-------|----------------|-------------------|
| Max | German Shepherd | Green | Adult shepherd, friendly pose |
| Luna | Labrador | Green | Golden lab, outdoor setting |
| Bello | Mixed Breed | Blue | Medium dog, playful |
| Rocky | Rottweiler | Orange | Strong dog, calm pose |
| Mia | Beagle | Green | Small beagle, cute expression |
| Bruno | Boxer | Blue | Athletic boxer |
| Lotte | Dachshund | Green | Classic dachshund |
| Thor | Husky | Orange | Blue-eyed husky |

### 2. User Profile Photos

**Current State:** Demo users have no profile photos.

**Target State:** Add realistic profile photos to seeded demo users.

**Implementation:**
- Use royalty-free portrait images or generated avatars
- Include diversity in appearances
- Store in `internal/static/frontend/uploads/users/`
- Update seed data to reference these images

### 3. Featured Dogs on Frontpage

**Current State:** `is_featured` flag may not be set for demo dogs.

**Target State:** Enable `is_featured` for all (or most) demo dogs so they appear on the frontpage.

**Implementation:**
- Update seed data: Set `is_featured = true` for demo dogs
- Ensure frontpage displays featured dogs prominently
- This creates an immediate visual impression for visitors

### 4. External Links for Dogs

**Current State:** Dog external links may be empty or point to invalid URLs.

**Target State:** All demo dog external links point to `https://gassigeher.org`

**Implementation:**
- Update seed data: Set `external_link = "https://gassigeher.org"` for all demo dogs
- This prevents broken links and reinforces branding

### 5. Realistic Bookings

**Current State:** May have few or no demo bookings.

**Target State:** Seed realistic booking data showing:
- Past completed walks
- Upcoming scheduled walks
- Mix of different users and dogs

**Implementation:**
- Create 10-15 bookings across different dates
- Include notes on completed bookings
- Show variety in walk types and times

## Files to Modify

1. **Seeding Logic:** `internal/services/provisioning_service.go` or similar
2. **Demo Tenant Setup:** Check where demo tenant data is created
3. **Static Assets:** Add image files for dogs and users

## Image Sources (Royalty-Free)

- [Unsplash Dogs](https://unsplash.com/s/photos/dog)
- [Pexels Dogs](https://pexels.com/search/dog/)
- [Generated Avatars](https://thispersondoesnotexist.com/) for users

## Acceptance Criteria

- [x] Demo site frontpage shows 5+ dogs with real photos (8 dogs with Unsplash photos)
- [x] All dogs have `is_featured = true`
- [x] All dog external links point to `https://gassigeher.org`
- [x] Demo users have profile photos (UI Avatars service)
- [x] Demo has realistic past and future bookings (10 completed, 8 scheduled)
- [x] Demo site loads quickly (images stored locally, embedded in binary)

## Priority

**Medium** - This improves conversion but is not blocking for launch.

## Estimated Effort

- Image sourcing and optimization: 1-2 hours
- Seed data updates: 1-2 hours
- Testing: 30 minutes

---

*Created: 2025-12-30*
*Implemented: 2025-12-30*
*Status: COMPLETED*

## Implementation Details

**Files Modified:**
- `internal/services/demo_seed_service.go` - Demo data definitions
- `internal/static/frontend/js/dog-photo-helpers.js` - Handle `/assets/` paths

**New Assets Added:**
- `internal/static/frontend/assets/images/demo/dogs/` - 8 dog photos (JPG, ~1MB total)
- `internal/static/frontend/assets/images/demo/users/` - 4 user avatars (SVG, ~1KB total)

**Changes:**
1. **Dogs (8 total):** Max, Luna, Bello, Rocky, Mia, Bruno, Lotte, Thor
   - All have local photos (downloaded from Unsplash, stored in `/assets/images/demo/dogs/`)
   - All have `is_featured = true`
   - All have `external_link = "https://gassigeher.org"`
   - Variety of sizes, breeds, and color categories

2. **Users (4 total):** Demo Admin, Anna Mueller, Bernd Schmidt, Clara Weber
   - All have local SVG avatars with initials (stored in `/assets/images/demo/users/`)
   - Color-coded backgrounds matching their permission levels (green, orange, blue)

3. **Bookings (18 total):**
   - 10 completed (past 5 days) with German notes
   - 8 scheduled (next 4 days)

**No External Dependencies:** All images are stored locally and embedded in the binary. The demo works completely offline without any external image services.
