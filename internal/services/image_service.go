package services

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
)

// ImageService handles image processing operations
type ImageService struct {
	uploadDir  string
	s3Service  *S3Service // Optional S3 service for cloud storage
	tenantSlug string     // Tenant slug for S3 path organization
	useS3      bool       // Flag to use S3 storage
}

// Image processing constants
const (
	MaxImageWidth   = 800  // Max width for full-size image
	MaxImageHeight  = 800  // Max height for full-size image
	ThumbnailSize   = 300  // Thumbnail dimensions (square max)
	JPEGQuality     = 85   // JPEG compression quality (1-100)
	LogoMaxWidth    = 1200 // Max width for site logo
	LogoMaxHeight   = 200  // Max height for site logo (banner format)
)

// NewImageService creates a new image service with local filesystem storage
// For Simple-Mode (single tenant) - no tenant prefix in paths
func NewImageService(uploadDir string) *ImageService {
	return &ImageService{
		uploadDir:  uploadDir,
		tenantSlug: "", // Empty = no tenant isolation (Simple-Mode)
		useS3:      false,
	}
}

// NewImageServiceWithTenant creates a new image service with tenant isolation for local storage
// For SaaS-Mode - all file paths will be prefixed with tenant slug
func NewImageServiceWithTenant(uploadDir string, tenantSlug string) *ImageService {
	return &ImageService{
		uploadDir:  uploadDir,
		tenantSlug: tenantSlug,
		useS3:      false,
	}
}

// NewImageServiceWithS3 creates a new image service with S3 storage support
func NewImageServiceWithS3(uploadDir string, s3Service *S3Service, tenantSlug string) *ImageService {
	return &ImageService{
		uploadDir:  uploadDir,
		s3Service:  s3Service,
		tenantSlug: tenantSlug,
		useS3:      s3Service != nil,
	}
}

// SetTenantSlug updates the tenant slug (used for S3 path organization)
func (s *ImageService) SetTenantSlug(slug string) {
	s.tenantSlug = slug
}

// getTenantBasePath returns the base path for the current tenant
// For SaaS-Mode: returns "{uploadDir}/{tenantSlug}"
// For Simple-Mode: returns "{uploadDir}" (no tenant prefix)
func (s *ImageService) getTenantBasePath() string {
	if s.tenantSlug != "" {
		return filepath.Join(s.uploadDir, s.tenantSlug)
	}
	return s.uploadDir
}

// getTenantRelativePath returns a relative path including tenant prefix if set
// For SaaS-Mode: returns "{tenantSlug}/{subPath}"
// For Simple-Mode: returns "{subPath}" (no tenant prefix)
func (s *ImageService) getTenantRelativePath(subPath string) string {
	if s.tenantSlug != "" {
		return filepath.Join(s.tenantSlug, subPath)
	}
	return subPath
}

// ProcessDogPhoto processes an uploaded dog photo and creates both full-size and thumbnail versions
// Returns the relative paths (e.g., "dogs/dog_5_full.jpg", "dogs/dog_5_thumb.jpg")
// When using S3, returns full URLs instead of relative paths
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

	// Process full-size image
	fullImg := s.resizeImage(img, MaxImageWidth, MaxImageHeight)
	fullFilename := fmt.Sprintf("dog_%d_full.jpg", dogID)

	// Process thumbnail
	thumbImg := s.resizeImage(img, ThumbnailSize, ThumbnailSize)
	thumbFilename := fmt.Sprintf("dog_%d_thumb.jpg", dogID)

	// Use S3 if configured
	if s.useS3 && s.s3Service != nil {
		// Encode images to JPEG in memory
		fullBuf, err := s.encodeJPEGToBuffer(fullImg, JPEGQuality)
		if err != nil {
			return "", "", fmt.Errorf("failed to encode full-size image: %w", err)
		}

		thumbBuf, err := s.encodeJPEGToBuffer(thumbImg, JPEGQuality)
		if err != nil {
			return "", "", fmt.Errorf("failed to encode thumbnail: %w", err)
		}

		// Upload to S3
		ctx := context.Background()
		fullURL, err := s.s3Service.Upload(ctx, s.tenantSlug,
			fmt.Sprintf("dogs/%s", fullFilename), fullBuf.Bytes(), "image/jpeg")
		if err != nil {
			return "", "", fmt.Errorf("failed to upload full-size image to S3: %w", err)
		}

		thumbURL, err := s.s3Service.Upload(ctx, s.tenantSlug,
			fmt.Sprintf("dogs/%s", thumbFilename), thumbBuf.Bytes(), "image/jpeg")
		if err != nil {
			// Try to clean up the full-size image
			s.s3Service.DeleteByPath(ctx, s.tenantSlug, fmt.Sprintf("dogs/%s", fullFilename))
			return "", "", fmt.Errorf("failed to upload thumbnail to S3: %w", err)
		}

		return fullURL, thumbURL, nil
	}

	// Local filesystem storage (default)
	// Use tenant-aware base path for SaaS-Mode isolation
	dogsDir := filepath.Join(s.getTenantBasePath(), "dogs")
	if err := os.MkdirAll(dogsDir, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create dogs directory: %w", err)
	}

	fullFilePath := filepath.Join(dogsDir, fullFilename)
	if err := s.saveJPEG(fullImg, fullFilePath, JPEGQuality); err != nil {
		return "", "", fmt.Errorf("failed to save full-size image: %w", err)
	}

	thumbFilePath := filepath.Join(dogsDir, thumbFilename)
	if err := s.saveJPEG(thumbImg, thumbFilePath, JPEGQuality); err != nil {
		// Clean up full image if thumbnail fails
		os.Remove(fullFilePath)
		return "", "", fmt.Errorf("failed to save thumbnail: %w", err)
	}

	// Return relative paths including tenant prefix (as stored in database)
	// SaaS-Mode: "{tenantSlug}/dogs/dog_{id}_full.jpg"
	// Simple-Mode: "dogs/dog_{id}_full.jpg"
	fullRelPath := s.getTenantRelativePath(filepath.Join("dogs", fullFilename))
	thumbRelPath := s.getTenantRelativePath(filepath.Join("dogs", thumbFilename))

	return fullRelPath, thumbRelPath, nil
}

// encodeJPEGToBuffer encodes an image to JPEG format in memory
func (s *ImageService) encodeJPEGToBuffer(img image.Image, quality int) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	opts := &jpeg.Options{Quality: quality}
	if err := jpeg.Encode(buf, img, opts); err != nil {
		return nil, fmt.Errorf("failed to encode JPEG: %w", err)
	}
	return buf, nil
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

// savePNG saves an image as PNG (preserves transparency)
func (s *ImageService) savePNG(img image.Image, path string) error {
	// Create output file
	outFile, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer outFile.Close()

	// Encode as PNG (lossless, preserves transparency)
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(outFile, img); err != nil {
		return fmt.Errorf("failed to encode PNG: %w", err)
	}

	return nil
}

// ProcessUserPhoto processes an uploaded user profile photo
// Returns the relative path (e.g., "users/user_5_profile.jpg")
// Uses userID in filename to prevent collisions
func (s *ImageService) ProcessUserPhoto(file multipart.File, userID int, ext string) (string, error) {
	// Reset file pointer to beginning
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("failed to seek file: %w", err)
	}

	// Decode the uploaded image
	img, err := imaging.Decode(file)
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Create users directory if it doesn't exist
	// Use tenant-aware base path for SaaS-Mode isolation
	usersDir := filepath.Join(s.getTenantBasePath(), "users")
	if err := os.MkdirAll(usersDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create users directory: %w", err)
	}

	// Resize to fit profile dimensions
	resized := s.resizeImage(img, MaxImageWidth, MaxImageHeight)

	// Generate unique filename based on user ID
	filename := fmt.Sprintf("user_%d_profile%s", userID, ext)
	filePath := filepath.Join(usersDir, filename)

	// Save as JPEG (convert all formats to JPEG for consistency and smaller size)
	if err := s.saveJPEG(resized, filePath, JPEGQuality); err != nil {
		return "", fmt.Errorf("failed to save user photo: %w", err)
	}

	// Return relative path including tenant prefix (as stored in database)
	// SaaS-Mode: "{tenantSlug}/users/user_{id}_profile.jpg"
	// Simple-Mode: "users/user_{id}_profile.jpg"
	return s.getTenantRelativePath(filepath.Join("users", filename)), nil
}

// DeleteUserPhoto deletes a user's profile photo
// Does not return error if file doesn't exist (idempotent)
func (s *ImageService) DeleteUserPhoto(userID int) error {
	// Use tenant-aware base path for SaaS-Mode isolation
	usersDir := filepath.Join(s.getTenantBasePath(), "users")

	// Delete possible photo files (both extensions since we might have old data)
	for _, ext := range []string{".jpg", ".jpeg", ".png"} {
		photoPath := filepath.Join(usersDir, fmt.Sprintf("user_%d_profile%s", userID, ext))
		if err := os.Remove(photoPath); err != nil && !os.IsNotExist(err) {
			// Log but continue - not a critical error
			continue
		}
	}

	return nil
}

// DeleteDogPhotos deletes both full-size and thumbnail photos for a dog
// Does not return error if files don't exist (idempotent)
func (s *ImageService) DeleteDogPhotos(dogID int) error {
	fullFilename := fmt.Sprintf("dog_%d_full.jpg", dogID)
	thumbFilename := fmt.Sprintf("dog_%d_thumb.jpg", dogID)

	// Use S3 if configured
	if s.useS3 && s.s3Service != nil {
		ctx := context.Background()
		// Delete from S3 (errors are ignored for idempotency)
		s.s3Service.DeleteByPath(ctx, s.tenantSlug, fmt.Sprintf("dogs/%s", fullFilename))
		s.s3Service.DeleteByPath(ctx, s.tenantSlug, fmt.Sprintf("dogs/%s", thumbFilename))
		return nil
	}

	// Local filesystem deletion - use tenant-aware path for SaaS-Mode isolation
	dogsDir := filepath.Join(s.getTenantBasePath(), "dogs")

	// Delete full-size image
	fullPath := filepath.Join(dogsDir, fullFilename)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete full-size image: %w", err)
	}

	// Delete thumbnail
	thumbPath := filepath.Join(dogsDir, thumbFilename)
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

// ProcessLogo processes an uploaded site logo image
// Returns the relative path (e.g., "settings/site_logo.png" or "settings/site_logo.jpg")
// PNG files are preserved as PNG to maintain transparency
func (s *ImageService) ProcessLogo(file multipart.File) (string, error) {
	// Reset file pointer to beginning
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("failed to seek file: %w", err)
	}

	// Detect image format by reading header bytes
	header := make([]byte, 8)
	if _, err := file.Read(header); err != nil {
		return "", fmt.Errorf("failed to read file header: %w", err)
	}

	// Reset file pointer again after reading header
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("failed to seek file: %w", err)
	}

	// Check if PNG (PNG magic bytes: 137 80 78 71 13 10 26 10)
	isPNG := header[0] == 0x89 && header[1] == 0x50 && header[2] == 0x4E && header[3] == 0x47

	// Decode the uploaded image
	img, err := imaging.Decode(file)
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Create settings directory if it doesn't exist
	// Use tenant-aware base path for SaaS-Mode isolation
	settingsDir := filepath.Join(s.getTenantBasePath(), "settings")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create settings directory: %w", err)
	}

	// Delete existing logo files first (both .jpg and .png)
	s.DeleteLogo()

	// Resize to fit banner dimensions
	resized := s.resizeImage(img, LogoMaxWidth, LogoMaxHeight)

	// Save in original format to preserve transparency for PNGs
	var filename string
	var filePath string

	if isPNG {
		filename = "site_logo.png"
		filePath = filepath.Join(settingsDir, filename)
		if err := s.savePNG(resized, filePath); err != nil {
			return "", fmt.Errorf("failed to save logo: %w", err)
		}
	} else {
		filename = "site_logo.jpg"
		filePath = filepath.Join(settingsDir, filename)
		if err := s.saveJPEG(resized, filePath, JPEGQuality); err != nil {
			return "", fmt.Errorf("failed to save logo: %w", err)
		}
	}

	// Return relative path including tenant prefix
	// SaaS-Mode: "{tenantSlug}/settings/site_logo.png"
	// Simple-Mode: "settings/site_logo.png"
	return s.getTenantRelativePath(filepath.Join("settings", filename)), nil
}

// DeleteLogo removes the custom site logo file (both .jpg and .png variants)
// Does not return error if files don't exist (idempotent)
func (s *ImageService) DeleteLogo() error {
	// Use tenant-aware base path for SaaS-Mode isolation
	settingsDir := filepath.Join(s.getTenantBasePath(), "settings")

	// Delete both possible logo files
	for _, ext := range []string{".jpg", ".png"} {
		logoPath := filepath.Join(settingsDir, "site_logo"+ext)
		if err := os.Remove(logoPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete logo: %w", err)
		}
	}
	return nil
}

// ProcessWalkReportPhoto processes an uploaded walk report photo
// Returns the relative paths (e.g., "walk_reports/report_5_1_full.jpg", "walk_reports/report_5_1_thumb.jpg")
func (s *ImageService) ProcessWalkReportPhoto(file multipart.File, reportID int, photoIndex int) (fullPath, thumbPath string, err error) {
	// Reset file pointer to beginning
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", "", fmt.Errorf("failed to seek file: %w", err)
	}

	// Decode the uploaded image
	img, err := imaging.Decode(file)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Create walk_reports directory if it doesn't exist
	// Use tenant-aware base path for SaaS-Mode isolation
	reportsDir := filepath.Join(s.getTenantBasePath(), "walk_reports")
	if err := os.MkdirAll(reportsDir, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create walk_reports directory: %w", err)
	}

	// Process full-size image
	fullImg := s.resizeImage(img, MaxImageWidth, MaxImageHeight)
	fullFilename := fmt.Sprintf("report_%d_%d_full.jpg", reportID, photoIndex)
	fullFilePath := filepath.Join(reportsDir, fullFilename)

	if err := s.saveJPEG(fullImg, fullFilePath, JPEGQuality); err != nil {
		return "", "", fmt.Errorf("failed to save full-size image: %w", err)
	}

	// Process thumbnail
	thumbImg := s.resizeImage(img, ThumbnailSize, ThumbnailSize)
	thumbFilename := fmt.Sprintf("report_%d_%d_thumb.jpg", reportID, photoIndex)
	thumbFilePath := filepath.Join(reportsDir, thumbFilename)

	if err := s.saveJPEG(thumbImg, thumbFilePath, JPEGQuality); err != nil {
		// Clean up full image if thumbnail fails
		os.Remove(fullFilePath)
		return "", "", fmt.Errorf("failed to save thumbnail: %w", err)
	}

	// Return relative paths including tenant prefix (as stored in database)
	// SaaS-Mode: "{tenantSlug}/walk_reports/report_{id}_{index}_full.jpg"
	// Simple-Mode: "walk_reports/report_{id}_{index}_full.jpg"
	fullRelPath := s.getTenantRelativePath(filepath.Join("walk_reports", fullFilename))
	thumbRelPath := s.getTenantRelativePath(filepath.Join("walk_reports", thumbFilename))

	return fullRelPath, thumbRelPath, nil
}

// safeJoinPath safely joins a base directory with a relative path,
// preventing path traversal attacks. Returns error if the resulting path
// would escape the base directory.
func (s *ImageService) safeJoinPath(relativePath string) (string, error) {
	// Reject absolute paths
	if filepath.IsAbs(relativePath) {
		return "", fmt.Errorf("absolute paths not allowed")
	}

	// Reject paths with ".." components
	if strings.Contains(relativePath, "..") {
		return "", fmt.Errorf("path traversal not allowed")
	}

	// Clean and join the path
	cleanPath := filepath.Clean(relativePath)
	fullPath := filepath.Join(s.uploadDir, cleanPath)

	// Verify the result is still within uploadDir
	absUploadDir, err := filepath.Abs(s.uploadDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve upload directory: %w", err)
	}

	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	// Ensure the resolved path starts with the upload directory
	if !strings.HasPrefix(absFullPath, absUploadDir+string(filepath.Separator)) &&
		absFullPath != absUploadDir {
		return "", fmt.Errorf("path escapes upload directory")
	}

	return fullPath, nil
}

// DeleteWalkReportPhoto deletes a walk report photo (both full and thumbnail)
// Does not return error if files don't exist (idempotent)
// Includes path traversal protection to prevent deletion of files outside upload directory
func (s *ImageService) DeleteWalkReportPhoto(fullPath, thumbPath string) error {
	// Validate and resolve full-size image path (with path traversal protection)
	fullAbsPath, err := s.safeJoinPath(fullPath)
	if err != nil {
		return fmt.Errorf("invalid full image path: %w", err)
	}

	// Delete full-size image
	if err := os.Remove(fullAbsPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete full-size image: %w", err)
	}

	// Validate and resolve thumbnail path (with path traversal protection)
	thumbAbsPath, err := s.safeJoinPath(thumbPath)
	if err != nil {
		return fmt.Errorf("invalid thumbnail path: %w", err)
	}

	// Delete thumbnail
	if err := os.Remove(thumbAbsPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete thumbnail: %w", err)
	}

	return nil
}
