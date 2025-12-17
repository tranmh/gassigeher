package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
)

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
func (s *HolidayService) FetchAndCacheHolidays(tenantID int, year int) error {
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

	// Cache miss - fetch from API
	url := fmt.Sprintf("https://feiertage-api.de/api/?jahr=%d&nur_land=%s", year, state)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch holidays: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("holiday API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
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
	for name, holiday := range holidays {
		h := &models.CustomHoliday{
			Date:     holiday.Datum,
			Name:     name,
			IsActive: true,
			Source:   "api",
		}

		// Insert or ignore if already exists
		_ = s.holidayRepo.CreateHoliday(tenantID, h)
	}

	return nil
}

// IsHoliday checks if a date is a holiday for a tenant
func (s *HolidayService) IsHoliday(tenantID int, date string) (bool, error) {
	// Check if API usage is enabled
	if setting, err := s.settingsRepo.Get(tenantID, "use_feiertage_api"); err == nil && setting != nil && setting.Value == "true" {
		// Ensure holidays are cached for this year
		dateObj, _ := time.Parse("2006-01-02", date)
		year := dateObj.Year()
		_ = s.FetchAndCacheHolidays(tenantID, year)
	}

	// Check database
	return s.holidayRepo.IsHoliday(tenantID, date)
}

// GetHolidaysForYear returns all holidays in a year for a tenant
func (s *HolidayService) GetHolidaysForYear(tenantID int, year int) ([]models.CustomHoliday, error) {
	// Fetch and cache if API enabled
	if setting, err := s.settingsRepo.Get(tenantID, "use_feiertage_api"); err == nil && setting != nil && setting.Value == "true" {
		_ = s.FetchAndCacheHolidays(tenantID, year)
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

	for name, holiday := range holidays {
		h := &models.CustomHoliday{
			Date:     holiday.Datum,
			Name:     name,
			IsActive: true,
			Source:   "api",
		}
		_ = s.holidayRepo.CreateHoliday(tenantID, h)
	}

	return nil
}
