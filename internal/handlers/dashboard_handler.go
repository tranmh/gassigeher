package handlers

import (
	"database/sql"
	"html"
	"net/http"
	"time"

	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
)

// DashboardHandler handles admin dashboard endpoints
type DashboardHandler struct {
	db                   *sql.DB
	cfg                  *config.Config
	bookingRepo          *repository.BookingRepository
	userRepo             *repository.UserRepository
	dogRepo              *repository.DogRepository
	experienceRepo       *repository.ExperienceRequestRepository
	reactivationRepo     *repository.ReactivationRequestRepository
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(db *sql.DB, cfg *config.Config) *DashboardHandler {
	return &DashboardHandler{
		db:               db,
		cfg:              cfg,
		bookingRepo:      repository.NewBookingRepository(db),
		userRepo:         repository.NewUserRepository(db),
		dogRepo:          repository.NewDogRepository(db),
		experienceRepo:   repository.NewExperienceRequestRepository(db),
		reactivationRepo: repository.NewReactivationRequestRepository(db),
	}
}

// GetStats returns dashboard statistics (admin only)
func (h *DashboardHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats := &models.DashboardStats{}

	// Get total completed walks
	completedBookings, err := h.bookingRepo.FindAll(&models.BookingFilterRequest{
		Status: strPtr("completed"),
	})
	if err == nil {
		stats.TotalWalksCompleted = len(completedBookings)
	}

	// Get upcoming walks
	today := time.Now().Format("2006-01-02")
	upcomingBookings, err := h.bookingRepo.FindAll(&models.BookingFilterRequest{
		Status:   strPtr("scheduled"),
		DateFrom: &today,
	})
	if err == nil {
		stats.UpcomingWalksTotal = len(upcomingBookings)

		// Count today's walks
		for _, booking := range upcomingBookings {
			if booking.Date == today {
				stats.UpcomingWalksToday++
			}
		}
	}

	// Get active/inactive users
	activeUsers, err := h.userRepo.FindAll(boolPtr(true))
	if err == nil {
		stats.ActiveUsers = len(activeUsers)
	}

	inactiveUsers, err := h.userRepo.FindAll(boolPtr(false))
	if err == nil {
		stats.InactiveUsers = len(inactiveUsers)
	}

	// Get available/unavailable dogs
	availableDogs, err := h.dogRepo.FindAll(&models.DogFilterRequest{
		Available: boolPtr(true),
	})
	if err == nil {
		stats.AvailableDogs = len(availableDogs)
	}

	unavailableDogs, err := h.dogRepo.FindAll(&models.DogFilterRequest{
		Available: boolPtr(false),
	})
	if err == nil {
		stats.UnavailableDogs = len(unavailableDogs)
	}

	// Get pending experience requests
	pendingExperienceReqs, err := h.experienceRepo.FindAllPending()
	if err == nil {
		stats.PendingExperienceReqs = len(pendingExperienceReqs)
	}

	// Get pending reactivation requests
	pendingReactivationReqs, err := h.reactivationRepo.FindAllPending()
	if err == nil {
		stats.PendingReactivationReqs = len(pendingReactivationReqs)
	}

	respondJSON(w, http.StatusOK, stats)
}

// GetRecentActivity returns recent activity feed (admin only)
func (h *DashboardHandler) GetRecentActivity(w http.ResponseWriter, r *http.Request) {
	activities := []*models.ActivityItem{}

	// Get recent bookings (last 24 hours)
	yesterday := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	recentBookings, err := h.bookingRepo.FindAll(&models.BookingFilterRequest{
		DateFrom: &yesterday,
	})

	if err == nil {
		for _, booking := range recentBookings {
			// Get dog name
			dog, err := h.dogRepo.FindByID(booking.DogID)
			dogName := "Unknown"
			if err == nil && dog != nil {
				// HTML-escape dog name to prevent XSS attacks
				dogName = html.EscapeString(dog.Name)
			}

			var activityType, message string
			switch booking.Status {
			case "scheduled":
				activityType = "booking_created"
				message = "Neue Buchung für " + dogName
			case "completed":
				activityType = "booking_completed"
				message = "Spaziergang mit " + dogName + " abgeschlossen"
			case "cancelled":
				activityType = "booking_cancelled"
				message = "Buchung für " + dogName + " storniert"
			}

			activity := &models.ActivityItem{
				Type:      activityType,
				Message:   message,
				Timestamp: booking.CreatedAt.Format(time.RFC3339),
				UserID:    &booking.UserID,
				DogID:     &booking.DogID,
				DogName:   dogName,
			}

			activities = append(activities, activity)
		}
	}

	// Limit to 20 most recent activities
	if len(activities) > 20 {
		activities = activities[:20]
	}

	response := &models.RecentActivityResponse{
		Activities: activities,
	}

	respondJSON(w, http.StatusOK, response)
}

// Helper functions
func strPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
