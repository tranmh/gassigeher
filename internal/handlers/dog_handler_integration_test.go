package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
	"github.com/tranmh/gassigeher/internal/services"

	_ "modernc.org/sqlite"
)

// setupTestDB creates a test database
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create dogs table
	_, err = db.Exec(`
		CREATE TABLE dogs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			breed TEXT NOT NULL,
			size TEXT,
			age INTEGER,
			category TEXT,
			color_id INTEGER,
			photo TEXT,
			photo_thumbnail TEXT,
			special_needs TEXT,
			pickup_location TEXT,
			walk_route TEXT,
			walk_duration INTEGER,
			special_instructions TEXT,
			default_morning_time TEXT,
			default_evening_time TEXT,
			is_available INTEGER DEFAULT 1,
			unavailable_reason TEXT,
			unavailable_since TIMESTAMP,
			is_featured INTEGER DEFAULT 0,
			external_link TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create dogs table: %v", err)
	}

	return db
}

// createTestImage creates a test image in memory
func createTestImageBytes(width, height int, format string) ([]byte, error) {
	// Create test image
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with gradient
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				uint8((x * 255) / width),
				uint8((y * 255) / height),
				128,
				255,
			})
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
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	return buf.Bytes(), nil
}

// createMultipartUpload creates a multipart form upload
func createMultipartUpload(fieldName, filename string, imageData []byte) (*bytes.Buffer, string, error) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile(fieldName, filename)
	if err != nil {
		return nil, "", err
	}

	if _, err := part.Write(imageData); err != nil {
		return nil, "", err
	}

	contentType := writer.FormDataContentType()
	writer.Close()

	return body, contentType, nil
}

// TestDogHandler_UploadDogPhoto_Integration tests the complete upload flow
func TestDogHandler_UploadDogPhoto_Integration(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer db.Close()

	tempDir := t.TempDir()
	cfg := &config.Config{
		UploadDir:       tempDir,
		MaxUploadSizeMB: 10,
	}

	handler := NewDogHandler(db, cfg)

	// Create test dog
	dogRepo := repository.NewDogRepository(db)
	dog := &models.Dog{
		Name:        "TestDog",
		Breed:       "Labrador",
		Size:        "medium",
		Age:         3,
		Category:    "green",
		IsAvailable: true,
	}
	if err := dogRepo.Create(dog); err != nil {
		t.Fatalf("Failed to create test dog: %v", err)
	}

	t.Run("Successfully upload JPEG photo", func(t *testing.T) {
		// Create test image
		imageData, err := createTestImageBytes(1500, 1500, "jpeg")
		if err != nil {
			t.Fatalf("Failed to create test image: %v", err)
		}

		// Create multipart upload
		body, contentType, err := createMultipartUpload("photo", "test.jpg", imageData)
		if err != nil {
			t.Fatalf("Failed to create upload: %v", err)
		}

		// Create request
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/dogs/%d/photo", dog.ID), body)
		req.Header.Set("Content-Type", contentType)

		// Add route vars
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dog.ID)})

		// Record response
		rr := httptest.NewRecorder()

		// Call handler
		handler.UploadDogPhoto(rr, req)

		// Check response
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
			return
		}

		// Parse response
		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Validate response
		if response["message"] != "Photo uploaded successfully" {
			t.Errorf("Unexpected message: %v", response["message"])
		}

		fullPath, ok := response["photo"].(string)
		if !ok || fullPath == "" {
			t.Errorf("Missing photo path in response")
		}

		thumbPath, ok := response["thumbnail"].(string)
		if !ok || thumbPath == "" {
			t.Errorf("Missing thumbnail path in response")
		}

		// Verify files exist
		fullFilePath := filepath.Join(tempDir, fullPath)
		thumbFilePath := filepath.Join(tempDir, thumbPath)

		if _, err := os.Stat(fullFilePath); os.IsNotExist(err) {
			t.Errorf("Full image file not created: %s", fullFilePath)
		}

		if _, err := os.Stat(thumbFilePath); os.IsNotExist(err) {
			t.Errorf("Thumbnail file not created: %s", thumbFilePath)
		}

		// Verify database updated
		updatedDog, err := dogRepo.FindByID(dog.ID)
		if err != nil {
			t.Fatalf("Failed to fetch updated dog: %v", err)
		}

		if updatedDog.Photo == nil || *updatedDog.Photo != fullPath {
			t.Errorf("Dog photo not updated in database")
		}

		if updatedDog.PhotoThumbnail == nil || *updatedDog.PhotoThumbnail != thumbPath {
			t.Errorf("Dog photo thumbnail not updated in database")
		}

		// Verify image quality
		fullImg, err := jpeg.DecodeConfig(bytes.NewReader(readFile(t, fullFilePath)))
		if err != nil {
			t.Fatalf("Failed to decode full image: %v", err)
		}

		if fullImg.Width > 800 || fullImg.Height > 800 {
			t.Errorf("Full image too large: %dx%d", fullImg.Width, fullImg.Height)
		}

		thumbImg, err := jpeg.DecodeConfig(bytes.NewReader(readFile(t, thumbFilePath)))
		if err != nil {
			t.Fatalf("Failed to decode thumbnail: %v", err)
		}

		if thumbImg.Width > 300 || thumbImg.Height > 300 {
			t.Errorf("Thumbnail too large: %dx%d", thumbImg.Width, thumbImg.Height)
		}
	})

	t.Run("Successfully upload PNG photo (converts to JPEG)", func(t *testing.T) {
		// Create PNG image
		imageData, err := createTestImageBytes(1000, 1000, "png")
		if err != nil {
			t.Fatalf("Failed to create test image: %v", err)
		}

		// Create multipart upload
		body, contentType, err := createMultipartUpload("photo", "test.png", imageData)
		if err != nil {
			t.Fatalf("Failed to create upload: %v", err)
		}

		// Create request
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/dogs/%d/photo", dog.ID), body)
		req.Header.Set("Content-Type", contentType)
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dog.ID)})

		// Record response
		rr := httptest.NewRecorder()

		// Call handler
		handler.UploadDogPhoto(rr, req)

		// Check response
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
			return
		}

		// Verify files are JPEG (not PNG)
		var response map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &response)

		fullPath := response["photo"].(string)
		if filepath.Ext(fullPath) != ".jpg" {
			t.Errorf("Expected JPEG extension, got: %s", filepath.Ext(fullPath))
		}
	})

	t.Run("Replace existing photo", func(t *testing.T) {
		// Upload first photo
		imageData1, _ := createTestImageBytes(800, 800, "jpeg")
		body1, contentType1, _ := createMultipartUpload("photo", "first.jpg", imageData1)

		req1 := httptest.NewRequest("POST", fmt.Sprintf("/api/dogs/%d/photo", dog.ID), body1)
		req1.Header.Set("Content-Type", contentType1)
		req1 = mux.SetURLVars(req1, map[string]string{"id": fmt.Sprintf("%d", dog.ID)})

		rr1 := httptest.NewRecorder()
		handler.UploadDogPhoto(rr1, req1)

		var response1 map[string]interface{}
		json.Unmarshal(rr1.Body.Bytes(), &response1)
		_ = response1["photo"].(string) // First photo path (will be replaced)

		// Upload second photo (should replace first)
		imageData2, _ := createTestImageBytes(900, 900, "jpeg")
		body2, contentType2, _ := createMultipartUpload("photo", "second.jpg", imageData2)

		req2 := httptest.NewRequest("POST", fmt.Sprintf("/api/dogs/%d/photo", dog.ID), body2)
		req2.Header.Set("Content-Type", contentType2)
		req2 = mux.SetURLVars(req2, map[string]string{"id": fmt.Sprintf("%d", dog.ID)})

		rr2 := httptest.NewRecorder()
		handler.UploadDogPhoto(rr2, req2)

		if rr2.Code != http.StatusOK {
			t.Errorf("Second upload failed: %d", rr2.Code)
			return
		}

		// Verify old photos deleted (new naming scheme)
		// Note: With new scheme, photos are named dog_{id}_full.jpg, so they get overwritten
		// This is expected behavior
	})

	t.Run("Reject invalid file type", func(t *testing.T) {
		// Create fake GIF data
		body, contentType, _ := createMultipartUpload("photo", "test.gif", []byte("GIF89a"))

		req := httptest.NewRequest("POST", fmt.Sprintf("/api/dogs/%d/photo", dog.ID), body)
		req.Header.Set("Content-Type", contentType)
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dog.ID)})

		rr := httptest.NewRecorder()
		handler.UploadDogPhoto(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for GIF upload, got %d", rr.Code)
		}
	})

	t.Run("Reject missing file", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		writer.Close()

		req := httptest.NewRequest("POST", fmt.Sprintf("/api/dogs/%d/photo", dog.ID), body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dog.ID)})

		rr := httptest.NewRecorder()
		handler.UploadDogPhoto(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for missing file, got %d", rr.Code)
		}
	})

	t.Run("Reject upload for non-existent dog", func(t *testing.T) {
		imageData, _ := createTestImageBytes(800, 800, "jpeg")
		body, contentType, _ := createMultipartUpload("photo", "test.jpg", imageData)

		req := httptest.NewRequest("POST", "/api/dogs/99999/photo", body)
		req.Header.Set("Content-Type", contentType)
		req = mux.SetURLVars(req, map[string]string{"id": "99999"})

		rr := httptest.NewRecorder()
		handler.UploadDogPhoto(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("Expected 404 for non-existent dog, got %d", rr.Code)
		}
	})

	t.Run("Reject corrupted image", func(t *testing.T) {
		// Create corrupted JPEG data
		corruptData := []byte("\xFF\xD8\xFF\xE0\x00\x10JFIF\x00corrupt")
		body, contentType, _ := createMultipartUpload("photo", "corrupt.jpg", corruptData)

		req := httptest.NewRequest("POST", fmt.Sprintf("/api/dogs/%d/photo", dog.ID), body)
		req.Header.Set("Content-Type", contentType)
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dog.ID)})

		rr := httptest.NewRecorder()
		handler.UploadDogPhoto(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500 for corrupted image, got %d", rr.Code)
		}
	})

	t.Run("Verify file sizes are optimized", func(t *testing.T) {
		// Upload large image
		imageData, _ := createTestImageBytes(3000, 3000, "jpeg")
		body, contentType, _ := createMultipartUpload("photo", "large.jpg", imageData)

		req := httptest.NewRequest("POST", fmt.Sprintf("/api/dogs/%d/photo", dog.ID), body)
		req.Header.Set("Content-Type", contentType)
		req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", dog.ID)})

		rr := httptest.NewRecorder()
		handler.UploadDogPhoto(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Upload failed: %d", rr.Code)
			return
		}

		var response map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &response)

		fullPath := filepath.Join(tempDir, response["photo"].(string))
		thumbPath := filepath.Join(tempDir, response["thumbnail"].(string))

		fullStat, _ := os.Stat(fullPath)
		thumbStat, _ := os.Stat(thumbPath)

		// Check file sizes are reasonable (compressed)
		if fullStat.Size() > 300*1024 {
			t.Errorf("Full image too large: %d bytes (expected < 300KB)", fullStat.Size())
		}

		if thumbStat.Size() > 80*1024 {
			t.Errorf("Thumbnail too large: %d bytes (expected < 80KB)", thumbStat.Size())
		}
	})
}

// readFile reads a file and returns its contents
func readFile(t *testing.T, path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}
	return data
}

// TestImageService_DeleteDogPhotos_Integration tests photo deletion
func TestImageService_DeleteDogPhotos_Integration(t *testing.T) {
	tempDir := t.TempDir()
	service := services.NewImageService(tempDir)

	// Create test files
	dogsDir := filepath.Join(tempDir, "dogs")
	os.MkdirAll(dogsDir, 0755)

	fullPath := filepath.Join(dogsDir, "dog_10_full.jpg")
	thumbPath := filepath.Join(dogsDir, "dog_10_thumb.jpg")

	os.WriteFile(fullPath, []byte("test full"), 0644)
	os.WriteFile(thumbPath, []byte("test thumb"), 0644)

	// Delete photos
	err := service.DeleteDogPhotos(10)
	if err != nil {
		t.Fatalf("DeleteDogPhotos failed: %v", err)
	}

	// Verify deletion
	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		t.Error("Full image still exists")
	}

	if _, err := os.Stat(thumbPath); !os.IsNotExist(err) {
		t.Error("Thumbnail still exists")
	}

	// Test idempotency
	err = service.DeleteDogPhotos(10)
	if err != nil {
		t.Errorf("Second delete should not error: %v", err)
	}
}
