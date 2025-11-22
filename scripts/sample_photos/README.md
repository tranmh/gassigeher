# Sample Dog Photos for Testing

This directory contains sample images for testing the dog photo upload functionality.

## Files

- `dog_sample_1.svg` - Sample Labrador (green category)
- `dog_sample_2.svg` - Sample German Shepherd (blue category)
- `dog_sample_3.svg` - Sample Beagle (orange category)

## Usage

### For Testing:

1. **Manual Upload Testing:**
   - Start the application
   - Login as admin
   - Navigate to admin-dogs.html
   - Upload these sample SVGs to test the upload functionality

2. **Automated Testing:**
   - These files can be used in integration tests
   - Copy to `uploads/dogs/` for visual testing

3. **Test Data Script:**
   - The `gentestdata.ps1` script can copy these files
   - Simulates dogs with photos in test database

## Notes

- These are SVG files for testing purposes
- In production, users will upload JPEG/PNG files
- Phase 1 (when implemented) will process JPEG/PNG and generate thumbnails
- These SVGs are lightweight and work for development/testing

## File Sizes

- Each SVG: ~3-5KB
- Actual dog photos (JPEG): ~100KB - 5MB before processing
- After Phase 1 processing: ~150KB full + ~30KB thumbnail

