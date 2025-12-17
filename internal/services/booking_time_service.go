package services

import (
	"fmt"
	"strconv"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
)

type BookingTimeService struct {
	bookingTimeRepo *repository.BookingTimeRepository
	holidayService  *HolidayService
	settingsRepo    *repository.SettingsRepository
}

func NewBookingTimeService(
	bookingTimeRepo *repository.BookingTimeRepository,
	holidayService *HolidayService,
	settingsRepo *repository.SettingsRepository,
) *BookingTimeService {
	return &BookingTimeService{
		bookingTimeRepo: bookingTimeRepo,
		holidayService:  holidayService,
		settingsRepo:    settingsRepo,
	}
}

// ValidateBookingTime validates if a time slot is allowed for a tenant
func (s *BookingTimeService) ValidateBookingTime(tenantID int, date string, scheduledTime string) error {
	// Parse date
	dateObj, err := time.Parse("2006-01-02", date)
	if err != nil {
		return fmt.Errorf("invalid date format")
	}

	// Parse time
	timeObj, err := time.Parse("15:04", scheduledTime)
	if err != nil {
		return fmt.Errorf("invalid time format")
	}

	// Determine day type
	dayType, err := s.getDayType(tenantID, date, dateObj)
	if err != nil {
		return err
	}

	// Get rules for day type
	rules, err := s.bookingTimeRepo.GetRulesByDayType(tenantID, dayType)
	if err != nil {
		return fmt.Errorf("failed to load time rules: %w", err)
	}

	// Check if time falls within any allowed window
	inAllowedWindow := false
	inBlockedWindow := false

	for _, rule := range rules {
		startTime, _ := time.Parse("15:04", rule.StartTime)
		endTime, _ := time.Parse("15:04", rule.EndTime)

		// Check if time is within this rule's window
		if !timeObj.Before(startTime) && timeObj.Before(endTime) {
			if rule.IsBlocked {
				inBlockedWindow = true
				return fmt.Errorf("Zeit ist gesperrt: %s (%s-%s)", rule.RuleName, rule.StartTime, rule.EndTime)
			} else {
				inAllowedWindow = true
			}
		}
	}

	if !inAllowedWindow {
		return fmt.Errorf("Zeit ist außerhalb der erlaubten Buchungszeiten")
	}

	if inBlockedWindow {
		return fmt.Errorf("Zeit fällt in eine Sperrzeit")
	}

	return nil
}

// GetAvailableTimeSlots returns all available time slots for a date within a tenant
func (s *BookingTimeService) GetAvailableTimeSlots(tenantID int, date string) ([]string, error) {
	// Parse date
	dateObj, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format")
	}

	// Determine day type
	dayType, err := s.getDayType(tenantID, date, dateObj)
	if err != nil {
		return nil, err
	}

	// Get rules
	rules, err := s.bookingTimeRepo.GetRulesByDayType(tenantID, dayType)
	if err != nil {
		return nil, err
	}

	// Get granularity
	granularity := 15 // Default
	if setting, err := s.settingsRepo.Get(tenantID, "booking_time_granularity"); err == nil && setting != nil {
		if g, err := strconv.Atoi(setting.Value); err == nil {
			granularity = g
		}
	}

	// Generate time slots
	var slots []string

	for _, rule := range rules {
		if rule.IsBlocked {
			continue // Skip blocked windows
		}

		startTime, _ := time.Parse("15:04", rule.StartTime)
		endTime, _ := time.Parse("15:04", rule.EndTime)

		// Generate slots in granularity intervals
		current := startTime
		for current.Before(endTime) {
			slots = append(slots, current.Format("15:04"))
			current = current.Add(time.Duration(granularity) * time.Minute)
		}
	}

	return slots, nil
}

// RequiresApproval checks if a booking requires admin approval for a tenant
func (s *BookingTimeService) RequiresApproval(tenantID int, scheduledTime string) (bool, error) {
	// Check setting
	setting, err := s.settingsRepo.Get(tenantID, "morning_walk_requires_approval")
	if err != nil || setting == nil || setting.Value != "true" {
		return false, nil // Setting disabled
	}

	// Parse time
	timeObj, err := time.Parse("15:04", scheduledTime)
	if err != nil {
		return false, err
	}

	// Morning window: 09:00 - 12:00
	morningStart, _ := time.Parse("15:04", "09:00")
	morningEnd, _ := time.Parse("15:04", "12:00")

	// Check if time falls in morning window
	if !timeObj.Before(morningStart) && timeObj.Before(morningEnd) {
		return true, nil
	}

	return false, nil
}

// getDayType determines if date is weekday, weekend, or holiday for a tenant
func (s *BookingTimeService) getDayType(tenantID int, date string, dateObj time.Time) (string, error) {
	// Check if holiday
	isHoliday, err := s.holidayService.IsHoliday(tenantID, date)
	if err != nil {
		return "", err
	}

	if isHoliday {
		return "weekend", nil // Holidays use weekend rules
	}

	// Check day of week
	weekday := dateObj.Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		return "weekend", nil
	}

	return "weekday", nil
}

// GetRulesForDate returns applicable rules for a specific date within a tenant
func (s *BookingTimeService) GetRulesForDate(tenantID int, date string) ([]models.BookingTimeRule, error) {
	dateObj, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, err
	}

	dayType, err := s.getDayType(tenantID, date, dateObj)
	if err != nil {
		return nil, err
	}

	return s.bookingTimeRepo.GetRulesByDayType(tenantID, dayType)
}
