package services

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
)

// ImageService handles image processing operations
type ImageService struct {
	uploadDir string
}

// Image processing constants
const (
	MaxImageWidth   = 800 // Max width for full-size image
	MaxImageHeight  = 800 // Max height for full-size image
	ThumbnailSize   = 300 // Thumbnail dimensions (square max)
	JPEGQuality     = 85  // JPEG compression quality (1-100)
)

// NewImageService creates a new image service
func NewImageService(uploadDir string) *ImageService {
	return &ImageService{
		uploadDir: uploadDir,
	}
}

// ProcessDogPhoto processes an uploaded dog photo and creates both full-size and thumbnail versions
// Returns the relative paths (e.g., "dogs/dog_5_full.jpg", "dogs/dog_5_thumb.jpg")
func (s *ImageService) ProcessDogPhoto(file multipart.File, dogID int) (fullPath, thumbPath string, err error) {
	// Reset file pointer to beginning
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", "", fmt.Errorf("failed to seek file: %w", err)
	}

	// Decode the uploaded image
	img, err := imaging.Decode(file)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Create dogs directory if it doesn't exist
	dogsDir := filepath.Join(s.uploadDir, "dogs")
	if err := os.MkdirAll(dogsDir, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create dogs directory: %w", err)
	}

	// Process full-size image
	fullImg := s.resizeImage(img, MaxImageWidth, MaxImageHeight)
	fullFilename := fmt.Sprintf("dog_%d_full.jpg", dogID)
	fullFilePath := filepath.Join(dogsDir, fullFilename)

	if err := s.saveJPEG(fullImg, fullFilePath, JPEGQuality); err != nil {
		return "", "", fmt.Errorf("failed to save full-size image: %w", err)
	}

	// Process thumbnail
	thumbImg := s.resizeImage(img, ThumbnailSize, ThumbnailSize)
	thumbFilename := fmt.Sprintf("dog_%d_thumb.jpg", dogID)
	thumbFilePath := filepath.Join(dogsDir, thumbFilename)

	if err := s.saveJPEG(thumbImg, thumbFilePath, JPEGQuality); err != nil {
		// Clean up full image if thumbnail fails
		os.Remove(fullFilePath)
		return "", "", fmt.Errorf("failed to save thumbnail: %w", err)
	}

	// Return relative paths (as stored in database)
	fullRelPath := filepath.Join("dogs", fullFilename)
	thumbRelPath := filepath.Join("dogs", thumbFilename)

	return fullRelPath, thumbRelPath, nil
}

// resizeImage resizes an image to fit within maxWidth x maxHeight while maintaining aspect ratio
// Uses Lanczos resampling filter for high-quality results
func (s *ImageService) resizeImage(img image.Image, maxWidth, maxHeight int) image.Image {
	// Get original dimensions
	bounds := img.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()

	// If image is already smaller than max dimensions, return as-is
	if origWidth <= maxWidth && origHeight <= maxHeight {
		return img
	}

	// Calculate scaling to fit within max dimensions while maintaining aspect ratio
	// Use Fit function which resizes the image to fit within the specified dimensions
	return imaging.Fit(img, maxWidth, maxHeight, imaging.Lanczos)
}

// saveJPEG saves an image as JPEG with specified quality
func (s *ImageService) saveJPEG(img image.Image, path string, quality int) error {
	// Create output file
	outFile, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer outFile.Close()

	// Encode as JPEG with specified quality
	opts := &jpeg.Options{Quality: quality}
	if err := jpeg.Encode(outFile, img, opts); err != nil {
		return fmt.Errorf("failed to encode JPEG: %w", err)
	}

	return nil
}

// DeleteDogPhotos deletes both full-size and thumbnail photos for a dog
// Does not return error if files don't exist (idempotent)
func (s *ImageService) DeleteDogPhotos(dogID int) error {
	dogsDir := filepath.Join(s.uploadDir, "dogs")

	// Delete full-size image
	fullPath := filepath.Join(dogsDir, fmt.Sprintf("dog_%d_full.jpg", dogID))
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete full-size image: %w", err)
	}

	// Delete thumbnail
	thumbPath := filepath.Join(dogsDir, fmt.Sprintf("dog_%d_thumb.jpg", dogID))
	if err := os.Remove(thumbPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete thumbnail: %w", err)
	}

	return nil
}

// ResizeAndCompress is a helper function that resizes and compresses an image in memory
// Returns a buffer containing the JPEG data
// This is useful for testing or when you need the image data without saving to disk
func (s *ImageService) ResizeAndCompress(img image.Image, maxWidth, maxHeight, quality int) (*bytes.Buffer, error) {
	// Resize image
	resized := s.resizeImage(img, maxWidth, maxHeight)

	// Encode to JPEG in memory
	buf := new(bytes.Buffer)
	opts := &jpeg.Options{Quality: quality}
	if err := jpeg.Encode(buf, resized, opts); err != nil {
		return nil, fmt.Errorf("failed to encode JPEG: %w", err)
	}

	return buf, nil
}

// GetDogPhotoPath returns the absolute filesystem path for a dog photo
// photoRelPath should be the relative path stored in database (e.g., "dogs/dog_5_full.jpg")
func (s *ImageService) GetDogPhotoPath(photoRelPath string) string {
	return filepath.Join(s.uploadDir, photoRelPath)
}
