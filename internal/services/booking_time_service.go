package services

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
)

const (
	// MinWalkBufferMinutes is the minimum time buffer before period end
	// to ensure enough time for the actual walk (e.g., excludes 11:45 when period ends at 12:00)
	MinWalkBufferMinutes = 30
)

type BookingTimeService struct {
	bookingTimeRepo *repository.BookingTimeRepository
	bookingRepo     *repository.BookingRepository // For period-based booking checks
	holidayService  *HolidayService
	settingsRepo    *repository.SettingsRepository
}

func NewBookingTimeService(
	bookingTimeRepo *repository.BookingTimeRepository,
	holidayService *HolidayService,
	settingsRepo *repository.SettingsRepository,
	bookingRepo *repository.BookingRepository,
) *BookingTimeService {
	return &BookingTimeService{
		bookingTimeRepo: bookingTimeRepo,
		bookingRepo:     bookingRepo,
		holidayService:  holidayService,
		settingsRepo:    settingsRepo,
	}
}

// ValidateBookingTime validates if a time slot is allowed for a tenant
// BUG FIX: Added context parameter for cancellation support
func (s *BookingTimeService) ValidateBookingTime(ctx context.Context, tenantID int, date string, scheduledTime string) error {
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
	dayType, err := s.getDayType(ctx, tenantID, date, dateObj)
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
// BUG FIX: Added context parameter for cancellation support
func (s *BookingTimeService) GetAvailableTimeSlots(ctx context.Context, tenantID int, date string) ([]string, error) {
	// Parse date
	dateObj, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format")
	}

	// Determine day type
	dayType, err := s.getDayType(ctx, tenantID, date, dateObj)
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
		// Apply buffer time to ensure enough time for the walk before period end
		cutoffTime := endTime.Add(-MinWalkBufferMinutes * time.Minute)

		current := startTime
		for !current.After(cutoffTime) {
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
// BUG FIX: Added context parameter for cancellation support
func (s *BookingTimeService) getDayType(ctx context.Context, tenantID int, date string, dateObj time.Time) (string, error) {
	// Check if holiday
	isHoliday, err := s.holidayService.IsHoliday(ctx, tenantID, date)
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
// BUG FIX: Added context parameter for cancellation support
func (s *BookingTimeService) GetRulesForDate(ctx context.Context, tenantID int, date string) ([]models.BookingTimeRule, error) {
	dateObj, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, err
	}

	dayType, err := s.getDayType(ctx, tenantID, date, dateObj)
	if err != nil {
		return nil, err
	}

	return s.bookingTimeRepo.GetRulesByDayType(tenantID, dayType)
}

// GetPeriodForTime returns the booking period (non-blocked rule) containing the given time.
// TENANT ISOLATION: Uses tenant-specific booking_time_rules.
// Returns nil if the time is in a blocked period or outside all periods.
func (s *BookingTimeService) GetPeriodForTime(ctx context.Context, tenantID int, date, scheduledTime string) (*models.BookingTimeRule, error) {
	// Parse date
	dateObj, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format")
	}

	// Parse time
	timeObj, err := time.Parse("15:04", scheduledTime)
	if err != nil {
		return nil, fmt.Errorf("invalid time format")
	}

	// Determine day type
	dayType, err := s.getDayType(ctx, tenantID, date, dateObj)
	if err != nil {
		return nil, err
	}

	// Get rules for day type
	rules, err := s.bookingTimeRepo.GetRulesByDayType(tenantID, dayType)
	if err != nil {
		return nil, fmt.Errorf("failed to load time rules: %w", err)
	}

	// Find the non-blocked rule containing this time
	for _, rule := range rules {
		if rule.IsBlocked {
			continue // Skip blocked periods
		}

		startTime, _ := time.Parse("15:04", rule.StartTime)
		endTime, _ := time.Parse("15:04", rule.EndTime)

		// Check if time is within this rule's window (start inclusive, end exclusive)
		if !timeObj.Before(startTime) && timeObj.Before(endTime) {
			// Return a copy to avoid modifying the original
			result := rule
			return &result, nil
		}
	}

	// Time is not in any available period
	return nil, nil
}

// CheckPeriodAvailability checks if a dog can be booked at the given time.
// Returns: isAvailable, existingBooking (if not available), period, error
// TENANT ISOLATION: Both period lookup and booking check use tenant_id.
func (s *BookingTimeService) CheckPeriodAvailability(
	ctx context.Context,
	tenantID, dogID int,
	date, scheduledTime string,
) (bool, *models.Booking, *models.BookingTimeRule, error) {
	// Get the period for the requested time
	period, err := s.GetPeriodForTime(ctx, tenantID, date, scheduledTime)
	if err != nil {
		return false, nil, nil, err
	}

	// If time is not in any available period (blocked or outside all periods)
	if period == nil {
		return false, nil, nil, fmt.Errorf("Zeit ist außerhalb der erlaubten Buchungszeiten")
	}

	// Enforce buffer time: booking must be at least MinWalkBufferMinutes before period end
	// This prevents bookings like 11:45 when period ends at 12:00 (30-min buffer)
	scheduledTimeObj, err := time.Parse("15:04", scheduledTime)
	if err != nil {
		return false, nil, nil, fmt.Errorf("invalid time format")
	}
	periodEndTime, err := time.Parse("15:04", period.EndTime)
	if err != nil {
		return false, nil, nil, fmt.Errorf("invalid period end time")
	}
	cutoffTime := periodEndTime.Add(-MinWalkBufferMinutes * time.Minute)
	if scheduledTimeObj.After(cutoffTime) {
		return false, nil, nil, fmt.Errorf("Buchung muss mindestens %d Minuten vor Ende des Zeitraums (%s) liegen",
			int(MinWalkBufferMinutes), period.EndTime)
	}

	// Check if dog already has a booking in this period
	// TENANT ISOLATION: CheckPeriodBooking includes tenant_id in query
	existingBooking, err := s.bookingRepo.CheckPeriodBooking(
		tenantID, dogID, date, period.StartTime, period.EndTime,
	)
	if err != nil {
		return false, nil, nil, fmt.Errorf("failed to check period availability: %w", err)
	}

	if existingBooking != nil {
		// Period is already booked
		return false, existingBooking, period, nil
	}

	// Period is available
	return true, nil, period, nil
}

// FilterSlotsForDog filters available slots to exclude periods already booked by the dog.
// Returns: booked periods for this dog, filtered available slots, error
// TENANT ISOLATION: All lookups scoped to tenant_id.
func (s *BookingTimeService) FilterSlotsForDog(
	ctx context.Context,
	tenantID, dogID int,
	date string,
	allSlots []string,
) ([]models.BookingTimeRule, []string, error) {
	// Get all periods for this date
	rules, err := s.GetRulesForDate(ctx, tenantID, date)
	if err != nil {
		return nil, nil, err
	}

	// Check which periods are already booked by this dog
	var bookedPeriods []models.BookingTimeRule

	for _, rule := range rules {
		if rule.IsBlocked {
			continue // Skip blocked periods
		}

		// TENANT ISOLATION: CheckPeriodBooking includes tenant_id
		existingBooking, err := s.bookingRepo.CheckPeriodBooking(
			tenantID, dogID, date, rule.StartTime, rule.EndTime,
		)
		if err != nil {
			return nil, nil, err
		}

		if existingBooking != nil {
			bookedPeriods = append(bookedPeriods, rule)
		}
	}

	// If no periods are booked, return all slots
	if len(bookedPeriods) == 0 {
		return bookedPeriods, allSlots, nil
	}

	// Filter slots to exclude times in booked periods
	var filteredSlots []string
	for _, slot := range allSlots {
		slotTime, err := time.Parse("15:04", slot)
		if err != nil {
			continue
		}

		isBlocked := false
		for _, period := range bookedPeriods {
			startTime, _ := time.Parse("15:04", period.StartTime)
			endTime, _ := time.Parse("15:04", period.EndTime)

			// Check if slot is within booked period (start inclusive, end exclusive)
			if !slotTime.Before(startTime) && slotTime.Before(endTime) {
				isBlocked = true
				break
			}
		}

		if !isBlocked {
			filteredSlots = append(filteredSlots, slot)
		}
	}

	return bookedPeriods, filteredSlots, nil
}
