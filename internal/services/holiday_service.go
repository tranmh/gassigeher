package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
)

// httpClient with timeout for external API calls
var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

type HolidayService struct {
	holidayRepo  *repository.HolidayRepository
	settingsRepo *repository.SettingsRepository
}

func NewHolidayService(holidayRepo *repository.HolidayRepository, settingsRepo *repository.SettingsRepository) *HolidayService {
	return &HolidayService{
		holidayRepo:  holidayRepo,
		settingsRepo: settingsRepo,
	}
}

// FetchAndCacheHolidays fetches holidays from API and stores in DB for a tenant
// BUG FIX: Added context parameter for cancellation support
func (s *HolidayService) FetchAndCacheHolidays(ctx context.Context, tenantID int, year int) error {
	// BUG FIX: Add explicit timeout if context doesn't have deadline
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	// BUG FIX: Check ctx.Err() before expensive operations
	if ctx.Err() != nil {
		return fmt.Errorf("context cancelled before start: %w", ctx.Err())
	}

	// Get state from settings
	state := "BW" // Default
	if setting, err := s.settingsRepo.Get(tenantID, "feiertage_state"); err == nil && setting != nil && setting.Value != "" {
		state = setting.Value
	}

	// Check cache first (global cache - same API data for all tenants)
	cached, err := s.holidayRepo.GetCachedHolidays(year, state)
	if err == nil && cached != "" {
		// Cache hit - populate custom_holidays table for this tenant
		return s.populateHolidaysFromCache(tenantID, cached, year)
	}

	// BUG FIX: Check ctx.Err() before making HTTP request
	if ctx.Err() != nil {
		return fmt.Errorf("context cancelled before API call: %w", ctx.Err())
	}

	// Cache miss - fetch from API (with 10s timeout)
	url := fmt.Sprintf("https://feiertage-api.de/api/?jahr=%d&nur_land=%s", year, state)

	// BUG FIX: Use context-aware request for cancellation support
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("request cancelled: %w", ctx.Err())
		}
		return fmt.Errorf("failed to fetch holidays: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("holiday API returned status %d", resp.StatusCode)
	}

	// Limit response body size to prevent DoS (1MB max - holiday data is small JSON)
	const maxBodySize = 1 * 1024 * 1024
	limitedReader := io.LimitReader(resp.Body, maxBodySize)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return fmt.Errorf("failed to read API response: %w", err)
	}

	// Parse response
	var holidays map[string]struct {
		Datum   string `json:"datum"`
		Hinweis string `json:"hinweis"`
	}

	if err := json.Unmarshal(body, &holidays); err != nil {
		return fmt.Errorf("failed to parse holidays: %w", err)
	}

	// Cache response (global cache)
	cacheDays := 7 // Default
	if setting, err := s.settingsRepo.Get(tenantID, "feiertage_cache_days"); err == nil && setting != nil {
		if days, err := strconv.Atoi(setting.Value); err == nil && days > 0 {
			cacheDays = days
		}
	}

	if err := s.holidayRepo.SetCachedHolidays(year, state, string(body), cacheDays); err != nil {
		// Log error but continue
		fmt.Printf("Warning: Failed to cache holidays: %v\n", err)
	}

	// Insert holidays into custom_holidays table for this tenant
	var createErrors []error
	for name, holiday := range holidays {
		h := &models.CustomHoliday{
			Date:     holiday.Datum,
			Name:     name,
			IsActive: true,
			Source:   "api",
		}

		// Insert or ignore if already exists (log errors but continue)
		if err := s.holidayRepo.CreateHoliday(tenantID, h); err != nil {
			createErrors = append(createErrors, fmt.Errorf("failed to create holiday %s: %w", name, err))
		}
	}

	// Log errors but don't fail the operation (some holidays may already exist)
	if len(createErrors) > 0 {
		fmt.Printf("Warning: %d holiday creation errors for tenant %d\n", len(createErrors), tenantID)
	}

	return nil
}

// IsHoliday checks if a date is a holiday for a tenant
// BUG FIX: Added context parameter for cancellation support
func (s *HolidayService) IsHoliday(ctx context.Context, tenantID int, date string) (bool, error) {
	// Validate date format first
	dateObj, err := time.Parse("2006-01-02", date)
	if err != nil {
		return false, fmt.Errorf("invalid date format: %w", err)
	}

	// Check if API usage is enabled
	if setting, err := s.settingsRepo.Get(tenantID, "use_feiertage_api"); err == nil && setting != nil && setting.Value == "true" {
		// Ensure holidays are cached for this year (log but don't fail on error)
		year := dateObj.Year()
		if err := s.FetchAndCacheHolidays(ctx, tenantID, year); err != nil {
			fmt.Printf("Warning: Failed to fetch holidays for tenant %d, year %d: %v\n", tenantID, year, err)
		}
	}

	// Check database
	return s.holidayRepo.IsHoliday(tenantID, date)
}

// GetHolidaysForYear returns all holidays in a year for a tenant
// BUG FIX: Added context parameter for cancellation support
func (s *HolidayService) GetHolidaysForYear(ctx context.Context, tenantID int, year int) ([]models.CustomHoliday, error) {
	// Fetch and cache if API enabled (log but don't fail on error)
	if setting, err := s.settingsRepo.Get(tenantID, "use_feiertage_api"); err == nil && setting != nil && setting.Value == "true" {
		if err := s.FetchAndCacheHolidays(ctx, tenantID, year); err != nil {
			fmt.Printf("Warning: Failed to fetch holidays for tenant %d, year %d: %v\n", tenantID, year, err)
		}
	}

	return s.holidayRepo.GetHolidaysByYear(tenantID, year)
}

// populateHolidaysFromCache helper
func (s *HolidayService) populateHolidaysFromCache(tenantID int, cached string, year int) error {
	var holidays map[string]struct {
		Datum string `json:"datum"`
	}

	if err := json.Unmarshal([]byte(cached), &holidays); err != nil {
		return err
	}

	var createErrors int
	for name, holiday := range holidays {
		h := &models.CustomHoliday{
			Date:     holiday.Datum,
			Name:     name,
			IsActive: true,
			Source:   "api",
		}
		if err := s.holidayRepo.CreateHoliday(tenantID, h); err != nil {
			createErrors++
		}
	}

	// Log errors but don't fail (some holidays may already exist)
	if createErrors > 0 {
		fmt.Printf("Warning: %d holiday creation errors for tenant %d from cache\n", createErrors, tenantID)
	}

	return nil
}
