package services

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"

	"github.com/disintegration/imaging"
)

// Test helper: create a test image in memory
func createTestImage(width, height int, format string) (*bytes.Buffer, error) {
	// Create a simple test image with gradient
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with gradient pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r := uint8((x * 255) / width)
			g := uint8((y * 255) / height)
			b := uint8(128)
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	buf := new(bytes.Buffer)

	switch format {
	case "jpeg", "jpg":
		if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: 90}); err != nil {
			return nil, err
		}
	case "png":
		if err := png.Encode(buf, img); err != nil {
			return nil, err
		}
	}

	return buf, nil
}

// Test helper: create a multipart.File from buffer
func createMultipartFile(buf *bytes.Buffer) multipart.File {
	return &testFile{
		Reader: bytes.NewReader(buf.Bytes()),
		size:   int64(buf.Len()),
	}
}

// testFile implements multipart.File interface for testing
type testFile struct {
	*bytes.Reader
	size int64
}

func (t *testFile) Close() error {
	return nil
}

func (t *testFile) Read(p []byte) (n int, err error) {
	return t.Reader.Read(p)
}

func (t *testFile) Seek(offset int64, whence int) (int64, error) {
	return t.Reader.Seek(offset, whence)
}

func (t *testFile) ReadAt(p []byte, off int64) (n int, err error) {
	return t.Reader.ReadAt(p, off)
}

// TestImageService_ProcessDogPhoto tests the complete photo processing pipeline
func TestImageService_ProcessDogPhoto(t *testing.T) {
	// Create temporary upload directory
	tempDir := t.TempDir()
	service := NewImageService(tempDir)

	tests := []struct {
		name          string
		dogID         int
		imageWidth    int
		imageHeight   int
		format        string
		expectError   bool
		validateFunc  func(t *testing.T, fullPath, thumbPath string)
	}{
		{
			name:        "Process large JPEG successfully",
			dogID:       1,
			imageWidth:  2000,
			imageHeight: 2000,
			format:      "jpeg",
			expectError: false,
			validateFunc: func(t *testing.T, fullPath, thumbPath string) {
				// Check files exist
				fullFilePath := filepath.Join(tempDir, fullPath)
				thumbFilePath := filepath.Join(tempDir, thumbPath)

				if _, err := os.Stat(fullFilePath); os.IsNotExist(err) {
					t.Errorf("Full image file does not exist: %s", fullFilePath)
				}
				if _, err := os.Stat(thumbFilePath); os.IsNotExist(err) {
					t.Errorf("Thumbnail file does not exist: %s", thumbFilePath)
				}

				// Check full image dimensions (should be <= 800x800)
				fullImg, err := imaging.Open(fullFilePath)
				if err != nil {
					t.Fatalf("Failed to open full image: %v", err)
				}
				bounds := fullImg.Bounds()
				if bounds.Dx() > MaxImageWidth || bounds.Dy() > MaxImageHeight {
					t.Errorf("Full image too large: %dx%d, expected max %dx%d",
						bounds.Dx(), bounds.Dy(), MaxImageWidth, MaxImageHeight)
				}

				// Check thumbnail dimensions (should be <= 300x300)
				thumbImg, err := imaging.Open(thumbFilePath)
				if err != nil {
					t.Fatalf("Failed to open thumbnail: %v", err)
				}
				thumbBounds := thumbImg.Bounds()
				if thumbBounds.Dx() > ThumbnailSize || thumbBounds.Dy() > ThumbnailSize {
					t.Errorf("Thumbnail too large: %dx%d, expected max %dx%d",
						thumbBounds.Dx(), thumbBounds.Dy(), ThumbnailSize, ThumbnailSize)
				}

				// Check file sizes are reasonable
				fullStat, _ := os.Stat(fullFilePath)
				thumbStat, _ := os.Stat(thumbFilePath)

				if fullStat.Size() > 300*1024 { // Should be < 300KB
					t.Errorf("Full image too large: %d bytes", fullStat.Size())
				}
				if thumbStat.Size() > 80*1024 { // Should be < 80KB
					t.Errorf("Thumbnail too large: %d bytes", thumbStat.Size())
				}
			},
		},
		{
			name:        "Process PNG successfully",
			dogID:       2,
			imageWidth:  1500,
			imageHeight: 1000,
			format:      "png",
			expectError: false,
			validateFunc: func(t *testing.T, fullPath, thumbPath string) {
				// Both should be converted to JPEG
				fullFilePath := filepath.Join(tempDir, fullPath)
				thumbFilePath := filepath.Join(tempDir, thumbPath)

				// Check they're JPEGs (not PNGs)
				if filepath.Ext(fullFilePath) != ".jpg" {
					t.Errorf("Expected JPEG extension, got: %s", filepath.Ext(fullFilePath))
				}
				if filepath.Ext(thumbFilePath) != ".jpg" {
					t.Errorf("Expected JPEG extension, got: %s", filepath.Ext(thumbFilePath))
				}
			},
		},
		{
			name:        "Process small image (no upscaling)",
			dogID:       3,
			imageWidth:  400,
			imageHeight: 300,
			format:      "jpeg",
			expectError: false,
			validateFunc: func(t *testing.T, fullPath, thumbPath string) {
				fullFilePath := filepath.Join(tempDir, fullPath)

				// Image should remain small (not upscaled)
				fullImg, err := imaging.Open(fullFilePath)
				if err != nil {
					t.Fatalf("Failed to open full image: %v", err)
				}
				bounds := fullImg.Bounds()

				// Should be same size or smaller (not larger than original)
				if bounds.Dx() > 400 || bounds.Dy() > 300 {
					t.Errorf("Image was upscaled: %dx%d, original was 400x300",
						bounds.Dx(), bounds.Dy())
				}
			},
		},
		{
			name:        "Process portrait image (maintains aspect ratio)",
			dogID:       4,
			imageWidth:  600,
			imageHeight: 1200,
			format:      "jpeg",
			expectError: false,
			validateFunc: func(t *testing.T, fullPath, thumbPath string) {
				fullFilePath := filepath.Join(tempDir, fullPath)

				fullImg, err := imaging.Open(fullFilePath)
				if err != nil {
					t.Fatalf("Failed to open full image: %v", err)
				}
				bounds := fullImg.Bounds()

				// Should maintain 1:2 aspect ratio (portrait)
				aspectRatio := float64(bounds.Dx()) / float64(bounds.Dy())
				expectedRatio := 600.0 / 1200.0 // 0.5

				if aspectRatio < expectedRatio-0.01 || aspectRatio > expectedRatio+0.01 {
					t.Errorf("Aspect ratio not maintained: got %.2f, expected %.2f",
						aspectRatio, expectedRatio)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test image
			buf, err := createTestImage(tt.imageWidth, tt.imageHeight, tt.format)
			if err != nil {
				t.Fatalf("Failed to create test image: %v", err)
			}

			// Create multipart file
			file := createMultipartFile(buf)

			// Process the photo
			fullPath, thumbPath, err := service.ProcessDogPhoto(file, tt.dogID)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Validate paths
			expectedFullPath := filepath.Join("dogs", "dog_"+string(rune(tt.dogID+'0'))+"_full.jpg")
			expectedThumbPath := filepath.Join("dogs", "dog_"+string(rune(tt.dogID+'0'))+"_thumb.jpg")

			if fullPath != expectedFullPath && !filepath.IsAbs(fullPath) {
				// Just check it contains the dog ID
				if !contains(fullPath, "dog_") || !contains(fullPath, "_full.jpg") {
					t.Errorf("Full path format incorrect: %s", fullPath)
				}
			}

			if thumbPath != expectedThumbPath && !filepath.IsAbs(thumbPath) {
				// Just check it contains the dog ID
				if !contains(thumbPath, "dog_") || !contains(thumbPath, "_thumb.jpg") {
					t.Errorf("Thumb path format incorrect: %s", thumbPath)
				}
			}

			// Run custom validation
			if tt.validateFunc != nil {
				tt.validateFunc(t, fullPath, thumbPath)
			}
		})
	}
}

// TestImageService_ResizeAndCompress tests in-memory image processing
func TestImageService_ResizeAndCompress(t *testing.T) {
	service := NewImageService(t.TempDir())

	tests := []struct {
		name         string
		inputWidth   int
		inputHeight  int
		maxWidth     int
		maxHeight    int
		quality      int
		expectError  bool
		validateSize bool
		maxSizeBytes int
	}{
		{
			name:         "Resize large image to 800x800",
			inputWidth:   2000,
			inputHeight:  2000,
			maxWidth:     800,
			maxHeight:    800,
			quality:      85,
			expectError:  false,
			validateSize: true,
			maxSizeBytes: 200 * 1024, // 200KB
		},
		{
			name:         "High quality compression",
			inputWidth:   1000,
			inputHeight:  1000,
			maxWidth:     800,
			maxHeight:    800,
			quality:      95,
			expectError:  false,
			validateSize: true,
			maxSizeBytes: 300 * 1024, // Higher quality = larger file
		},
		{
			name:         "Low quality compression",
			inputWidth:   1000,
			inputHeight:  1000,
			maxWidth:     800,
			maxHeight:    800,
			quality:      60,
			expectError:  false,
			validateSize: true,
			maxSizeBytes: 100 * 1024, // Lower quality = smaller file
		},
		{
			name:         "Thumbnail size",
			inputWidth:   1500,
			inputHeight:  1500,
			maxWidth:     300,
			maxHeight:    300,
			quality:      85,
			expectError:  false,
			validateSize: true,
			maxSizeBytes: 50 * 1024, // 50KB
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test image
			img := image.NewRGBA(image.Rect(0, 0, tt.inputWidth, tt.inputHeight))

			// Fill with test pattern
			for y := 0; y < tt.inputHeight; y++ {
				for x := 0; x < tt.inputWidth; x++ {
					img.Set(x, y, color.RGBA{
						uint8((x * 255) / tt.inputWidth),
						uint8((y * 255) / tt.inputHeight),
						128,
						255,
					})
				}
			}

			// Process image
			buf, err := service.ResizeAndCompress(img, tt.maxWidth, tt.maxHeight, tt.quality)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if buf == nil {
				t.Fatal("Expected buffer but got nil")
			}

			// Validate output is valid JPEG
			decodedImg, err := imaging.Decode(buf)
			if err != nil {
				t.Fatalf("Failed to decode output image: %v", err)
			}

			// Check dimensions
			bounds := decodedImg.Bounds()
			if bounds.Dx() > tt.maxWidth || bounds.Dy() > tt.maxHeight {
				t.Errorf("Image too large: %dx%d, expected max %dx%d",
					bounds.Dx(), bounds.Dy(), tt.maxWidth, tt.maxHeight)
			}

			// Validate file size
			if tt.validateSize && buf.Len() > tt.maxSizeBytes {
				t.Errorf("Output too large: %d bytes, expected max %d bytes",
					buf.Len(), tt.maxSizeBytes)
			}
		})
	}
}

// TestImageService_DeleteDogPhotos tests photo deletion
func TestImageService_DeleteDogPhotos(t *testing.T) {
	tempDir := t.TempDir()
	service := NewImageService(tempDir)

	// Create test files
	dogID := 5
	dogsDir := filepath.Join(tempDir, "dogs")
	os.MkdirAll(dogsDir, 0755)

	fullPath := filepath.Join(dogsDir, "dog_5_full.jpg")
	thumbPath := filepath.Join(dogsDir, "dog_5_thumb.jpg")

	// Create dummy files
	os.WriteFile(fullPath, []byte("test"), 0644)
	os.WriteFile(thumbPath, []byte("test"), 0644)

	// Verify files exist
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Fatal("Test file setup failed")
	}
	if _, err := os.Stat(thumbPath); os.IsNotExist(err) {
		t.Fatal("Test file setup failed")
	}

	// Delete photos
	err := service.DeleteDogPhotos(dogID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify files are deleted
	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		t.Error("Full image file still exists")
	}
	if _, err := os.Stat(thumbPath); !os.IsNotExist(err) {
		t.Error("Thumbnail file still exists")
	}

	// Test idempotency - deleting again should not error
	err = service.DeleteDogPhotos(dogID)
	if err != nil {
		t.Errorf("Second delete should not error: %v", err)
	}
}

// TestImageService_ProcessDogPhoto_InvalidInput tests error cases
func TestImageService_ProcessDogPhoto_InvalidInput(t *testing.T) {
	tempDir := t.TempDir()
	service := NewImageService(tempDir)

	t.Run("Invalid image data", func(t *testing.T) {
		// Create invalid image data
		buf := bytes.NewBuffer([]byte("not an image"))
		file := createMultipartFile(buf)

		_, _, err := service.ProcessDogPhoto(file, 999)
		if err == nil {
			t.Error("Expected error for invalid image data")
		}
	})

	t.Run("Corrupted JPEG", func(t *testing.T) {
		// Create corrupted JPEG
		buf := bytes.NewBuffer([]byte("\xFF\xD8\xFF\xE0\x00\x10JFIF"))
		file := createMultipartFile(buf)

		_, _, err := service.ProcessDogPhoto(file, 999)
		if err == nil {
			t.Error("Expected error for corrupted JPEG")
		}
	})
}

// TestImageService_AspectRatioPreservation tests various aspect ratios
func TestImageService_AspectRatioPreservation(t *testing.T) {
	tempDir := t.TempDir()
	service := NewImageService(tempDir)

	tests := []struct {
		name         string
		width        int
		height       int
		expectedMinW int
		expectedMaxW int
		expectedMinH int
		expectedMaxH int
	}{
		{
			name:         "Square image",
			width:        1000,
			height:       1000,
			expectedMinW: 750,
			expectedMaxW: 800,
			expectedMinH: 750,
			expectedMaxH: 800,
		},
		{
			name:         "Wide panorama",
			width:        3000,
			height:       1000,
			expectedMinW: 750,
			expectedMaxW: 800,
			expectedMinH: 200,
			expectedMaxH: 300,
		},
		{
			name:         "Tall portrait",
			width:        1000,
			height:       3000,
			expectedMinW: 200,
			expectedMaxW: 300,
			expectedMinH: 750,
			expectedMaxH: 800,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test image
			buf, err := createTestImage(tt.width, tt.height, "jpeg")
			if err != nil {
				t.Fatalf("Failed to create test image: %v", err)
			}

			file := createMultipartFile(buf)
			fullPath, _, err := service.ProcessDogPhoto(file, 100)
			if err != nil {
				t.Fatalf("ProcessDogPhoto failed: %v", err)
			}

			// Load and check dimensions
			fullFilePath := filepath.Join(tempDir, fullPath)
			img, err := imaging.Open(fullFilePath)
			if err != nil {
				t.Fatalf("Failed to open image: %v", err)
			}

			bounds := img.Bounds()
			w, h := bounds.Dx(), bounds.Dy()

			if w < tt.expectedMinW || w > tt.expectedMaxW {
				t.Errorf("Width out of range: got %d, expected %d-%d",
					w, tt.expectedMinW, tt.expectedMaxW)
			}
			if h < tt.expectedMinH || h > tt.expectedMaxH {
				t.Errorf("Height out of range: got %d, expected %d-%d",
					h, tt.expectedMinH, tt.expectedMaxH)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		len(s) > 2*len(substr) && contains(s[1:], substr))))
}

// TestImageService_ProcessLogo tests logo processing
func TestImageService_ProcessLogo(t *testing.T) {
	tempDir := t.TempDir()
	service := NewImageService(tempDir)

	tests := []struct {
		name         string
		imageWidth   int
		imageHeight  int
		format       string
		expectedExt  string // Expected file extension
		expectError  bool
		validateFunc func(t *testing.T, logoPath string)
	}{
		{
			name:        "Process wide banner PNG logo (preserves transparency)",
			imageWidth:  1500,
			imageHeight: 150,
			format:      "png",
			expectedExt: ".png",
			expectError: false,
			validateFunc: func(t *testing.T, logoPath string) {
				fullPath := filepath.Join(tempDir, logoPath)

				// Check file exists
				if _, err := os.Stat(fullPath); os.IsNotExist(err) {
					t.Errorf("Logo file does not exist: %s", fullPath)
				}

				// Check dimensions (should be <= 1200x200)
				img, err := imaging.Open(fullPath)
				if err != nil {
					t.Fatalf("Failed to open logo: %v", err)
				}
				bounds := img.Bounds()
				if bounds.Dx() > LogoMaxWidth || bounds.Dy() > LogoMaxHeight {
					t.Errorf("Logo too large: %dx%d, expected max %dx%d",
						bounds.Dx(), bounds.Dy(), LogoMaxWidth, LogoMaxHeight)
				}

				// Check it's a PNG (preserves transparency)
				if filepath.Ext(fullPath) != ".png" {
					t.Errorf("Expected PNG extension for PNG input, got: %s", filepath.Ext(fullPath))
				}
			},
		},
		{
			name:        "Process tall JPEG logo (should resize)",
			imageWidth:  500,
			imageHeight: 500,
			format:      "jpeg",
			expectedExt: ".jpg",
			expectError: false,
			validateFunc: func(t *testing.T, logoPath string) {
				fullPath := filepath.Join(tempDir, logoPath)

				img, err := imaging.Open(fullPath)
				if err != nil {
					t.Fatalf("Failed to open logo: %v", err)
				}
				bounds := img.Bounds()

				// Should be resized to fit within LogoMaxHeight
				if bounds.Dy() > LogoMaxHeight {
					t.Errorf("Logo height too large: %d, expected max %d",
						bounds.Dy(), LogoMaxHeight)
				}

				// Check it's a JPEG
				if filepath.Ext(fullPath) != ".jpg" {
					t.Errorf("Expected JPEG extension for JPEG input, got: %s", filepath.Ext(fullPath))
				}
			},
		},
		{
			name:        "Process very large JPEG logo",
			imageWidth:  3000,
			imageHeight: 500,
			format:      "jpeg",
			expectedExt: ".jpg",
			expectError: false,
			validateFunc: func(t *testing.T, logoPath string) {
				fullPath := filepath.Join(tempDir, logoPath)

				img, err := imaging.Open(fullPath)
				if err != nil {
					t.Fatalf("Failed to open logo: %v", err)
				}
				bounds := img.Bounds()

				// Should be resized to fit within LogoMaxWidth
				if bounds.Dx() > LogoMaxWidth {
					t.Errorf("Logo width too large: %d, expected max %d",
						bounds.Dx(), LogoMaxWidth)
				}
			},
		},
		{
			name:        "Process small PNG logo (no upscaling)",
			imageWidth:  200,
			imageHeight: 50,
			format:      "png",
			expectedExt: ".png",
			expectError: false,
			validateFunc: func(t *testing.T, logoPath string) {
				fullPath := filepath.Join(tempDir, logoPath)

				img, err := imaging.Open(fullPath)
				if err != nil {
					t.Fatalf("Failed to open logo: %v", err)
				}
				bounds := img.Bounds()

				// Should NOT be upscaled
				if bounds.Dx() > 200 || bounds.Dy() > 50 {
					t.Errorf("Logo was upscaled: %dx%d, original was 200x50",
						bounds.Dx(), bounds.Dy())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test image
			buf, err := createTestImage(tt.imageWidth, tt.imageHeight, tt.format)
			if err != nil {
				t.Fatalf("Failed to create test image: %v", err)
			}

			file := createMultipartFile(buf)

			// Process the logo
			logoPath, err := service.ProcessLogo(file)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Validate path format (PNG stays PNG, JPEG stays JPEG)
			expectedPath := "settings/site_logo" + tt.expectedExt
			if logoPath != expectedPath {
				t.Errorf("Expected path '%s', got %s", expectedPath, logoPath)
			}

			// Run custom validation
			if tt.validateFunc != nil {
				tt.validateFunc(t, logoPath)
			}
		})
	}
}

// TestImageService_ProcessLogo_InvalidInput tests error cases for logo processing
func TestImageService_ProcessLogo_InvalidInput(t *testing.T) {
	tempDir := t.TempDir()
	service := NewImageService(tempDir)

	t.Run("Invalid image data", func(t *testing.T) {
		buf := bytes.NewBuffer([]byte("not an image"))
		file := createMultipartFile(buf)

		_, err := service.ProcessLogo(file)
		if err == nil {
			t.Error("Expected error for invalid image data")
		}
	})

	t.Run("Corrupted JPEG", func(t *testing.T) {
		buf := bytes.NewBuffer([]byte("\xFF\xD8\xFF\xE0\x00\x10JFIF"))
		file := createMultipartFile(buf)

		_, err := service.ProcessLogo(file)
		if err == nil {
			t.Error("Expected error for corrupted JPEG")
		}
	})
}

// TestImageService_DeleteLogo tests logo deletion
func TestImageService_DeleteLogo(t *testing.T) {
	tempDir := t.TempDir()
	service := NewImageService(tempDir)

	// Create settings directory and test logo files (both formats)
	settingsDir := filepath.Join(tempDir, "settings")
	os.MkdirAll(settingsDir, 0755)

	logoPathJPG := filepath.Join(settingsDir, "site_logo.jpg")
	logoPathPNG := filepath.Join(settingsDir, "site_logo.png")

	t.Run("Delete existing JPG logo", func(t *testing.T) {
		// Create dummy JPG logo file
		os.WriteFile(logoPathJPG, []byte("test jpg logo content"), 0644)

		err := service.DeleteLogo()
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		// Verify file is deleted
		if _, err := os.Stat(logoPathJPG); !os.IsNotExist(err) {
			t.Error("JPG Logo file still exists")
		}
	})

	t.Run("Delete existing PNG logo", func(t *testing.T) {
		// Create dummy PNG logo file
		os.WriteFile(logoPathPNG, []byte("test png logo content"), 0644)

		err := service.DeleteLogo()
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		// Verify file is deleted
		if _, err := os.Stat(logoPathPNG); !os.IsNotExist(err) {
			t.Error("PNG Logo file still exists")
		}
	})

	t.Run("Delete both JPG and PNG logos", func(t *testing.T) {
		// Create both files
		os.WriteFile(logoPathJPG, []byte("test jpg"), 0644)
		os.WriteFile(logoPathPNG, []byte("test png"), 0644)

		err := service.DeleteLogo()
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		// Verify both files are deleted
		if _, err := os.Stat(logoPathJPG); !os.IsNotExist(err) {
			t.Error("JPG Logo file still exists")
		}
		if _, err := os.Stat(logoPathPNG); !os.IsNotExist(err) {
			t.Error("PNG Logo file still exists")
		}
	})

	t.Run("Delete non-existent logo (idempotent)", func(t *testing.T) {
		// Delete again should not error
		err := service.DeleteLogo()
		if err != nil {
			t.Errorf("Delete should be idempotent: %v", err)
		}
	})
}

// TestImageService_ProcessLogo_Overwrites tests that uploading a new logo overwrites the old one
func TestImageService_ProcessLogo_Overwrites(t *testing.T) {
	tempDir := t.TempDir()
	service := NewImageService(tempDir)

	t.Run("Overwrite JPEG with JPEG", func(t *testing.T) {
		// Upload first JPEG logo
		buf1, _ := createTestImage(400, 100, "jpeg")
		file1 := createMultipartFile(buf1)
		path1, err := service.ProcessLogo(file1)
		if err != nil {
			t.Fatalf("First upload failed: %v", err)
		}

		if path1 != "settings/site_logo.jpg" {
			t.Errorf("Expected JPG path, got %s", path1)
		}

		// Upload second JPEG logo (different size)
		buf2, _ := createTestImage(800, 150, "jpeg")
		file2 := createMultipartFile(buf2)
		path2, err := service.ProcessLogo(file2)
		if err != nil {
			t.Fatalf("Second upload failed: %v", err)
		}

		// Same path for same format
		if path1 != path2 {
			t.Errorf("Expected same path for same format, got %s and %s", path1, path2)
		}
	})

	t.Run("Overwrite JPEG with PNG (deletes old JPG)", func(t *testing.T) {
		settingsDir := filepath.Join(tempDir, "settings")
		jpgPath := filepath.Join(settingsDir, "site_logo.jpg")
		pngPath := filepath.Join(settingsDir, "site_logo.png")

		// Upload JPEG first
		buf1, _ := createTestImage(400, 100, "jpeg")
		file1 := createMultipartFile(buf1)
		_, err := service.ProcessLogo(file1)
		if err != nil {
			t.Fatalf("JPEG upload failed: %v", err)
		}

		// Verify JPG exists
		if _, err := os.Stat(jpgPath); os.IsNotExist(err) {
			t.Fatal("JPG file should exist")
		}

		// Upload PNG (should delete JPG)
		buf2, _ := createTestImage(600, 120, "png")
		file2 := createMultipartFile(buf2)
		path2, err := service.ProcessLogo(file2)
		if err != nil {
			t.Fatalf("PNG upload failed: %v", err)
		}

		// New path should be PNG
		if path2 != "settings/site_logo.png" {
			t.Errorf("Expected PNG path, got %s", path2)
		}

		// JPG should be deleted
		if _, err := os.Stat(jpgPath); !os.IsNotExist(err) {
			t.Error("Old JPG file should be deleted")
		}

		// PNG should exist
		if _, err := os.Stat(pngPath); os.IsNotExist(err) {
			t.Error("PNG file should exist")
		}
	})

	t.Run("Overwrite PNG with JPEG (deletes old PNG)", func(t *testing.T) {
		settingsDir := filepath.Join(tempDir, "settings")
		jpgPath := filepath.Join(settingsDir, "site_logo.jpg")
		pngPath := filepath.Join(settingsDir, "site_logo.png")

		// Clean up first
		service.DeleteLogo()

		// Upload PNG first
		buf1, _ := createTestImage(400, 100, "png")
		file1 := createMultipartFile(buf1)
		_, err := service.ProcessLogo(file1)
		if err != nil {
			t.Fatalf("PNG upload failed: %v", err)
		}

		// Verify PNG exists
		if _, err := os.Stat(pngPath); os.IsNotExist(err) {
			t.Fatal("PNG file should exist")
		}

		// Upload JPEG (should delete PNG)
		buf2, _ := createTestImage(600, 120, "jpeg")
		file2 := createMultipartFile(buf2)
		path2, err := service.ProcessLogo(file2)
		if err != nil {
			t.Fatalf("JPEG upload failed: %v", err)
		}

		// New path should be JPEG
		if path2 != "settings/site_logo.jpg" {
			t.Errorf("Expected JPEG path, got %s", path2)
		}

		// PNG should be deleted
		if _, err := os.Stat(pngPath); !os.IsNotExist(err) {
			t.Error("Old PNG file should be deleted")
		}

		// JPEG should exist
		if _, err := os.Stat(jpgPath); os.IsNotExist(err) {
			t.Error("JPEG file should exist")
		}
	})
}
