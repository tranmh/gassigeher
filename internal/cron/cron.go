package cron

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/repository"
	"github.com/tranmh/gassigeher/internal/services"
)

// CronService handles scheduled tasks
type CronService struct {
	db           *sql.DB
	bookingRepo  *repository.BookingRepository
	userRepo     *repository.UserRepository
	settingsRepo *repository.SettingsRepository
	emailService *services.EmailService
	stopChan     chan bool
}

// NewCronService creates a new cron service
func NewCronService(db *sql.DB, cfg *config.Config) *CronService {
	// Initialize email service for reminders (fail gracefully if not configured)
	var emailService *services.EmailService
	if cfg != nil {
		var err error
		emailService, err = services.NewEmailService(services.ConfigToEmailConfig(cfg))
		if err != nil {
			log.Printf("Warning: Email service not available for cron jobs: %v", err)
		}
	}

	return &CronService{
		db:           db,
		bookingRepo:  repository.NewBookingRepository(db),
		userRepo:     repository.NewUserRepository(db),
		settingsRepo: repository.NewSettingsRepository(db),
		emailService: emailService,
		stopChan:     make(chan bool),
	}
}

// Start starts all cron jobs
func (s *CronService) Start() {
	log.Println("Starting cron service...")

	// Run auto-complete job every 15 minutes (allows users to add notes quickly after completion)
	go s.runPeriodically("Auto-complete bookings", 15*time.Minute, s.autoCompleteBookings)

	// Run auto-deactivation job daily at 3am (also runs once on startup)
	go s.runDaily("Auto-deactivate inactive users", 3, 0, s.autoDeactivateInactiveUsers)

	// Run booking reminder job every 15 minutes
	go s.runPeriodically("Send booking reminders", 15*time.Minute, s.sendBookingReminders)
}

// Stop stops all cron jobs
func (s *CronService) Stop() {
	log.Println("Stopping cron service...")
	close(s.stopChan)
}

// runPeriodically runs a function periodically
func (s *CronService) runPeriodically(name string, interval time.Duration, fn func()) {
	// Run immediately on start
	fn()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Printf("Running cron job: %s", name)
			fn()
		case <-s.stopChan:
			log.Printf("Stopped cron job: %s", name)
			return
		}
	}
}

// autoCompleteBookings marks past scheduled bookings as completed
func (s *CronService) autoCompleteBookings() {
	count, err := s.bookingRepo.AutoComplete()
	if err != nil {
		log.Printf("Error auto-completing bookings: %v", err)
		return
	}

	if count > 0 {
		log.Printf("Auto-completed %d booking(s)", count)
	} else {
		log.Println("Auto-complete check: no bookings to complete")
	}
}

// sendBookingReminders sends reminders for upcoming bookings (1-2 hours before)
func (s *CronService) sendBookingReminders() {
	// Check if email service is available
	if s.emailService == nil {
		log.Println("Reminder check: email service not configured, skipping")
		return
	}

	// Get bookings that need reminders
	bookings, err := s.bookingRepo.GetForReminders()
	if err != nil {
		log.Printf("Error getting bookings for reminders: %v", err)
		return
	}

	if len(bookings) == 0 {
		log.Println("Reminder check: no reminders to send")
		return
	}

	log.Printf("Found %d booking(s) that need reminders", len(bookings))

	// Send reminder for each booking
	for _, booking := range bookings {
		// Skip if user has no email
		if booking.User == nil || booking.User.Email == nil {
			log.Printf("Skipping reminder for booking %d: no user email", booking.ID)
			continue
		}

		// Skip if dog name is missing
		dogName := "Unbekannter Hund"
		if booking.Dog != nil && booking.Dog.Name != "" {
			dogName = booking.Dog.Name
		}

		// Format date for German locale (DD.MM.YYYY)
		formattedDate := booking.Date
		if t, err := time.Parse("2006-01-02", booking.Date); err == nil {
			formattedDate = t.Format("02.01.2006")
		}

		// Send reminder email
		err := s.emailService.SendBookingReminder(
			*booking.User.Email,
			booking.User.FirstName,
			dogName,
			formattedDate,
			booking.ScheduledTime,
		)

		if err != nil {
			log.Printf("Error sending reminder for booking %d: %v", booking.ID, err)
			continue
		}

		// Mark reminder as sent
		if err := s.bookingRepo.MarkReminderSent(booking.ID); err != nil {
			log.Printf("Error marking reminder sent for booking %d: %v", booking.ID, err)
			continue
		}

		log.Printf("Sent reminder for booking %d (user: %s, dog: %s, time: %s %s)",
			booking.ID, booking.User.FullName(), dogName, formattedDate, booking.ScheduledTime)
	}
}

// runDaily runs a function daily at a specific time (also runs once immediately on startup)
func (s *CronService) runDaily(name string, hour, minute int, fn func()) {
	// Run immediately on startup
	log.Printf("Running daily job on startup: %s", name)
	fn()

	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())

		// If we've passed today's scheduled time, schedule for tomorrow
		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}

		duration := next.Sub(now)
		log.Printf("Scheduling daily job '%s' to run in %v (at %s)", name, duration, next.Format("2006-01-02 15:04:05"))

		select {
		case <-time.After(duration):
			log.Printf("Running daily job: %s", name)
			fn()
		case <-s.stopChan:
			log.Printf("Stopped daily job: %s", name)
			return
		}
	}
}

// autoDeactivateInactiveUsers deactivates users who haven't been active for the configured period
func (s *CronService) autoDeactivateInactiveUsers() {
	// Get deactivation period from settings
	setting, err := s.settingsRepo.Get("auto_deactivation_days")
	if err != nil {
		log.Printf("Error getting auto_deactivation_days setting: %v", err)
		return
	}

	days := 365 // default 1 year
	if setting != nil {
		if d, err := strconv.Atoi(setting.Value); err == nil {
			days = d
		}
	}

	// Find inactive users
	users, err := s.userRepo.FindInactiveUsers(days)
	if err != nil {
		log.Printf("Error finding inactive users: %v", err)
		return
	}

	if len(users) == 0 {
		log.Println("No inactive users to deactivate")
		return
	}

	log.Printf("Found %d inactive user(s) to deactivate", len(users))

	// Deactivate each user
	for _, user := range users {
		if err := s.userRepo.Deactivate(user.ID, "auto_inactivity"); err != nil {
			log.Printf("Error deactivating user %d: %v", user.ID, err)
			continue
		}

		log.Printf("Auto-deactivated user %d (inactive for %d days)", user.ID, days)

		// Send email notification about deactivation
		if s.emailService != nil && user.Email != nil {
			reason := fmt.Sprintf("Keine Aktivität seit %d Tagen", days)
			go s.emailService.SendAccountDeactivated(*user.Email, user.FirstName, reason)
		}
	}
}
