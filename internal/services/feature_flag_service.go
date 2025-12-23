package services

import (
	"errors"
	"sync"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
)

// FeatureFlagService provides feature flag checking with caching
type FeatureFlagService struct {
	repo  *repository.FeatureFlagRepository
	cache map[string]*flagCacheEntry
	mu    sync.RWMutex
	ttl   time.Duration
}

type flagCacheEntry struct {
	value     bool
	expiresAt time.Time
}

// NewFeatureFlagService creates a new feature flag service
func NewFeatureFlagService(repo *repository.FeatureFlagRepository) *FeatureFlagService {
	return &FeatureFlagService{
		repo:  repo,
		cache: make(map[string]*flagCacheEntry),
		ttl:   5 * time.Minute, // Cache for 5 minutes
	}
}

// cacheKey generates a cache key for tenant+flag combination
func cacheKey(tenantID int, flagKey string) string {
	return flagKey + ":" + string(rune(tenantID))
}

// IsEnabled checks if a feature is enabled for a tenant (with caching)
func (s *FeatureFlagService) IsEnabled(key string, tenantID int) bool {
	cKey := cacheKey(tenantID, key)

	// Check cache first
	s.mu.RLock()
	entry, exists := s.cache[cKey]
	s.mu.RUnlock()

	if exists && time.Now().Before(entry.expiresAt) {
		return entry.value
	}

	// Cache miss or expired - check database
	enabled, err := s.repo.IsEnabled(key, tenantID)
	if err != nil {
		// On error, default to disabled
		return false
	}

	// Update cache
	s.mu.Lock()
	s.cache[cKey] = &flagCacheEntry{
		value:     enabled,
		expiresAt: time.Now().Add(s.ttl),
	}
	s.mu.Unlock()

	return enabled
}

// InvalidateCache clears the cache for a specific flag or all flags
func (s *FeatureFlagService) InvalidateCache(key string, tenantID int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if key == "" {
		// Clear all cache
		s.cache = make(map[string]*flagCacheEntry)
	} else {
		// Clear specific entry
		delete(s.cache, cacheKey(tenantID, key))
	}
}

// GetAllFlags returns all feature flags
func (s *FeatureFlagService) GetAllFlags() ([]*models.FeatureFlag, error) {
	return s.repo.GetAll()
}

// GetFlagsForTenant returns all flags with their effective status for a tenant
func (s *FeatureFlagService) GetFlagsForTenant(tenantID int) ([]*models.FeatureFlagWithStatus, error) {
	return s.repo.GetAllWithTenantStatus(tenantID)
}

// GetFlag returns a single feature flag by key
func (s *FeatureFlagService) GetFlag(key string) (*models.FeatureFlag, error) {
	return s.repo.GetByKey(key)
}

// CreateFlag creates a new feature flag
func (s *FeatureFlagService) CreateFlag(req *models.CreateFeatureFlagRequest) (*models.FeatureFlag, error) {
	// Validate key format (alphanumeric with underscores)
	if req.Key == "" {
		return nil, errors.New("flag key is required")
	}
	for _, c := range req.Key {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
			return nil, errors.New("flag key must be lowercase alphanumeric with underscores only")
		}
	}

	if req.Name == "" {
		return nil, errors.New("flag name is required")
	}

	// Check if key already exists
	existing, err := s.repo.GetByKey(req.Key)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("flag with this key already exists")
	}

	flag := &models.FeatureFlag{
		Key:         req.Key,
		Name:        req.Name,
		Description: req.Description,
		IsGlobal:    req.IsGlobal,
		IsEnabled:   req.IsEnabled,
	}

	if err := s.repo.Create(flag); err != nil {
		return nil, err
	}

	return flag, nil
}

// UpdateFlag updates a feature flag
func (s *FeatureFlagService) UpdateFlag(id int, req *models.UpdateFeatureFlagRequest) (*models.FeatureFlag, error) {
	flag, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if flag == nil {
		return nil, errors.New("flag not found")
	}

	if req.Name != nil && *req.Name != "" {
		flag.Name = *req.Name
	}
	if req.Description != nil {
		flag.Description = *req.Description
	}
	if req.IsGlobal != nil {
		flag.IsGlobal = *req.IsGlobal
	}
	if req.IsEnabled != nil {
		flag.IsEnabled = *req.IsEnabled
	}

	if err := s.repo.Update(flag); err != nil {
		return nil, err
	}

	// Invalidate all cache entries for this flag
	s.mu.Lock()
	for k := range s.cache {
		// Remove all entries starting with this flag key
		if len(k) >= len(flag.Key) && k[:len(flag.Key)] == flag.Key {
			delete(s.cache, k)
		}
	}
	s.mu.Unlock()

	return flag, nil
}

// DeleteFlag deletes a feature flag
func (s *FeatureFlagService) DeleteFlag(id int) error {
	flag, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if flag == nil {
		return errors.New("flag not found")
	}

	// Invalidate cache for this flag
	s.mu.Lock()
	for k := range s.cache {
		if len(k) >= len(flag.Key) && k[:len(flag.Key)] == flag.Key {
			delete(s.cache, k)
		}
	}
	s.mu.Unlock()

	return s.repo.Delete(id)
}

// SetTenantFlag sets a feature flag for a specific tenant
func (s *FeatureFlagService) SetTenantFlag(tenantID, flagID int, isEnabled bool, enabledBy *int) error {
	flag, err := s.repo.GetByID(flagID)
	if err != nil {
		return err
	}
	if flag == nil {
		return errors.New("flag not found")
	}

	if err := s.repo.SetTenantFlag(tenantID, flagID, isEnabled, enabledBy); err != nil {
		return err
	}

	// Invalidate cache for this tenant+flag
	s.InvalidateCache(flag.Key, tenantID)

	return nil
}

// RemoveTenantFlag removes a tenant-specific feature flag override
func (s *FeatureFlagService) RemoveTenantFlag(tenantID, flagID int) error {
	flag, err := s.repo.GetByID(flagID)
	if err != nil {
		return err
	}
	if flag == nil {
		return errors.New("flag not found")
	}

	if err := s.repo.RemoveTenantFlag(tenantID, flagID); err != nil {
		return err
	}

	// Invalidate cache
	s.InvalidateCache(flag.Key, tenantID)

	return nil
}
