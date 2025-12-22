package services

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// Constants for demo tenant
const (
	DemoTenantSlug      = "demo"
	DemoTenantName      = "Demo Tierheim"
	DemoAdminEmail      = "admin@demo.gassigeher.org"
	DemoUserPassword    = "demo1234"
	DefaultFederalState = "BW"
)

// DemoSeedService handles demo tenant creation and reset
type DemoSeedService struct {
	db              *sql.DB
	tenantRepo      *repository.TenantRepository
	demoStateRepo   *repository.DemoTenantRepository
	userRepo        *repository.UserRepository
	dogRepo         *repository.DogRepository
	bookingRepo     *repository.BookingRepository
	colorRepo       *repository.ColorCategoryRepository
	userColorRepo   *repository.UserColorRepository
	settingsRepo    *repository.SettingsRepository
}

// NewDemoSeedService creates a new demo seed service
func NewDemoSeedService(db *sql.DB) *DemoSeedService {
	return &DemoSeedService{
		db:            db,
		tenantRepo:    repository.NewTenantRepository(db),
		demoStateRepo: repository.NewDemoTenantRepository(db),
		userRepo:      repository.NewUserRepository(db),
		dogRepo:       repository.NewDogRepository(db),
		bookingRepo:   repository.NewBookingRepository(db),
		colorRepo:     repository.NewColorCategoryRepository(db),
		userColorRepo: repository.NewUserColorRepository(db),
		settingsRepo:  repository.NewSettingsRepository(db),
	}
}

// EnsureDemoTenant ensures the demo tenant exists and has data
func (s *DemoSeedService) EnsureDemoTenant() error {
	// Check if demo tenant already exists
	tenant, err := s.tenantRepo.FindBySlug(DemoTenantSlug)
	if err != nil {
		return fmt.Errorf("failed to check demo tenant: %w", err)
	}

	if tenant != nil {
		log.Println("Demo tenant already exists, skipping creation")
		return nil
	}

	log.Println("Creating demo tenant...")

	// Generate admin password
	adminPassword := s.generateRandomPassword()

	// Create demo tenant
	tenant = &models.Tenant{
		Slug:         DemoTenantSlug,
		Name:         DemoTenantName,
		Status:       models.TenantStatusActive,
		ContactEmail: DemoAdminEmail,
		FederalState: DefaultFederalState,
		IsDemo:       true,
	}

	if err := s.tenantRepo.Create(tenant); err != nil {
		return fmt.Errorf("failed to create demo tenant: %w", err)
	}

	// Create tenant settings
	settings := &models.TenantSettings{
		TenantID:    tenant.ID,
		ThemePreset: "classic",
	}
	if err := s.tenantRepo.CreateSettings(settings); err != nil {
		return fmt.Errorf("failed to create demo tenant settings: %w", err)
	}

	// Seed demo data
	if err := s.SeedDemoData(tenant.ID, adminPassword); err != nil {
		return fmt.Errorf("failed to seed demo data: %w", err)
	}

	// Create demo state
	now := time.Now()
	nextReset := s.calculateNextResetTime()
	state := &models.DemoTenantState{
		TenantID:      tenant.ID,
		AdminPassword: adminPassword,
		LastResetAt:   &now,
		NextResetAt:   &nextReset,
	}

	if err := s.demoStateRepo.CreateState(state); err != nil {
		return fmt.Errorf("failed to create demo state: %w", err)
	}

	log.Printf("Demo tenant created successfully (ID: %d, Password: %s)", tenant.ID, adminPassword)
	return nil
}

// SeedDemoData populates the demo tenant with sample data
func (s *DemoSeedService) SeedDemoData(tenantID int, adminPassword string) error {
	// Seed color categories first
	if err := s.seedDemoColors(tenantID); err != nil {
		return fmt.Errorf("failed to seed colors: %w", err)
	}

	// Seed users
	userIDs, err := s.seedDemoUsers(tenantID, adminPassword)
	if err != nil {
		return fmt.Errorf("failed to seed users: %w", err)
	}

	// Seed dogs
	dogIDs, err := s.seedDemoDogs(tenantID)
	if err != nil {
		return fmt.Errorf("failed to seed dogs: %w", err)
	}

	// Seed bookings
	if err := s.seedDemoBookings(tenantID, userIDs, dogIDs); err != nil {
		return fmt.Errorf("failed to seed bookings: %w", err)
	}

	// Initialize system settings
	if err := s.initializeDemoSettings(tenantID); err != nil {
		return fmt.Errorf("failed to initialize settings: %w", err)
	}

	return nil
}

// seedDemoColors creates the default color categories for the demo tenant
// This function is idempotent - it skips colors that already exist
func (s *DemoSeedService) seedDemoColors(tenantID int) error {
	colors := []struct {
		Name      string
		HexCode   string
		SortOrder int
	}{
		{"Gruen", "#22c55e", 1},
		{"Gelb", "#eab308", 2},
		{"Orange", "#f97316", 3},
		{"Hellblau", "#38bdf8", 4},
		{"Dunkelblau", "#3b82f6", 5},
	}

	createdCount := 0
	for _, c := range colors {
		// Check if color already exists (idempotent)
		existing, err := s.colorRepo.FindByName(tenantID, c.Name)
		if err != nil {
			return fmt.Errorf("failed to check existing color %s: %w", c.Name, err)
		}
		if existing != nil {
			// Color already exists, skip
			continue
		}

		color := &models.ColorCategory{
			TenantID:  tenantID,
			Name:      c.Name,
			HexCode:   c.HexCode,
			SortOrder: c.SortOrder,
		}
		if err := s.colorRepo.Create(tenantID, color); err != nil {
			return fmt.Errorf("failed to create color %s: %w", c.Name, err)
		}
		createdCount++
	}

	log.Printf("Created %d color categories for demo tenant", createdCount)
	return nil
}

// seedDemoUsers creates demo users and returns their IDs
// Returns map: "admin" -> ID, "green" -> ID, "orange" -> ID, "blue" -> ID
func (s *DemoSeedService) seedDemoUsers(tenantID int, adminPassword string) (map[string]int, error) {
	userIDs := make(map[string]int)
	now := time.Now()

	// Hash passwords
	adminHash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash admin password: %w", err)
	}

	demoHash, err := bcrypt.GenerateFromPassword([]byte(DemoUserPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash demo password: %w", err)
	}

	users := []struct {
		Key          string
		FirstName    string
		LastName     string
		Email        string
		PasswordHash string
		IsSuperAdmin bool
		IsAdmin      bool
		ColorLevel   string // green, orange, blue
	}{
		{
			Key:          "admin",
			FirstName:    "Demo",
			LastName:     "Admin",
			Email:        DemoAdminEmail,
			PasswordHash: string(adminHash),
			IsSuperAdmin: true,
			IsAdmin:      true,
			ColorLevel:   "blue", // Admin gets all colors
		},
		{
			Key:          "green",
			FirstName:    "Anna",
			LastName:     "Gruen",
			Email:        "anna@demo.gassigeher.org",
			PasswordHash: string(demoHash),
			IsSuperAdmin: false,
			IsAdmin:      false,
			ColorLevel:   "green",
		},
		{
			Key:          "orange",
			FirstName:    "Bernd",
			LastName:     "Orange",
			Email:        "bernd@demo.gassigeher.org",
			PasswordHash: string(demoHash),
			IsSuperAdmin: false,
			IsAdmin:      false,
			ColorLevel:   "orange",
		},
		{
			Key:          "blue",
			FirstName:    "Clara",
			LastName:     "Blau",
			Email:        "clara@demo.gassigeher.org",
			PasswordHash: string(demoHash),
			IsSuperAdmin: false,
			IsAdmin:      false,
			ColorLevel:   "blue",
		},
	}

	for _, u := range users {
		email := u.Email
		user := &models.User{
			TenantID:        tenantID,
			FirstName:       u.FirstName,
			LastName:        u.LastName,
			Email:           &email,
			PasswordHash:    &u.PasswordHash,
			IsAdmin:         u.IsAdmin,
			IsSuperAdmin:    u.IsSuperAdmin,
			IsVerified:      true,
			IsActive:        true,
			TermsAcceptedAt: now,
			LastActivityAt:  now,
		}

		if err := s.userRepo.Create(user); err != nil {
			return nil, fmt.Errorf("failed to create user %s: %w", u.Email, err)
		}

		userIDs[u.Key] = user.ID

		// Assign color levels
		if err := s.assignUserColors(tenantID, user.ID, u.ColorLevel); err != nil {
			return nil, fmt.Errorf("failed to assign colors to user %s: %w", u.Email, err)
		}
	}

	log.Printf("Created %d demo users", len(users))
	return userIDs, nil
}

// assignUserColors assigns colors based on user level
func (s *DemoSeedService) assignUserColors(tenantID, userID int, level string) error {
	// Get color IDs for this tenant
	colors, err := s.colorRepo.FindAll(tenantID)
	if err != nil {
		return fmt.Errorf("failed to get colors: %w", err)
	}

	// Build color name to ID map
	colorMap := make(map[string]int)
	for _, c := range colors {
		colorMap[c.Name] = c.ID
	}

	// Determine which colors to assign based on level
	var colorNames []string
	switch level {
	case "green":
		colorNames = []string{"Gruen"}
	case "orange":
		colorNames = []string{"Gruen", "Gelb", "Orange"}
	case "blue":
		colorNames = []string{"Gruen", "Gelb", "Orange", "Hellblau", "Dunkelblau"}
	default:
		colorNames = []string{"Gruen"}
	}

	// Assign colors (grantedBy = userID means self-granted for demo)
	for _, name := range colorNames {
		if colorID, ok := colorMap[name]; ok {
			if err := s.userColorRepo.AddColorToUser(tenantID, userID, colorID, userID); err != nil {
				// Ignore duplicate errors
				log.Printf("Warning: failed to add color %s to user %d: %v", name, userID, err)
			}
		}
	}

	return nil
}

// seedDemoDogs creates demo dogs and returns their IDs
func (s *DemoSeedService) seedDemoDogs(tenantID int) ([]int, error) {
	// Get color IDs
	colors, err := s.colorRepo.FindAll(tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get colors: %w", err)
	}

	colorMap := make(map[string]int)
	for _, c := range colors {
		colorMap[c.Name] = c.ID
	}

	dogs := []struct {
		Name                string
		Breed               string
		Size                string
		Age                 int
		ColorName           string
		IsFeatured          bool
		SpecialNeeds        string
		PickupLocation      string
		WalkRoute           string
		WalkDuration        int
		SpecialInstructions string
	}{
		{
			Name:                "Bella",
			Breed:               "Labrador Retriever",
			Size:                "large",
			Age:                 3,
			ColorName:           "Gruen",
			IsFeatured:          true,
			SpecialNeeds:        "Keine besonderen Beduerfnisse",
			PickupLocation:      "Zwinger 1, Gebaeude A",
			WalkRoute:           "Waldweg hinter dem Tierheim",
			WalkDuration:        45,
			SpecialInstructions: "Bella ist sehr freundlich und vertraegt sich gut mit anderen Hunden.",
		},
		{
			Name:                "Max",
			Breed:               "Golden Retriever",
			Size:                "large",
			Age:                 5,
			ColorName:           "Gruen",
			IsFeatured:          true,
			SpecialNeeds:        "Leichte Arthrose - keine langen Spaziergaenge",
			PickupLocation:      "Zwinger 3, Gebaeude A",
			WalkRoute:           "Kurze Runde um den Teich",
			WalkDuration:        30,
			SpecialInstructions: "Max braucht haeufige Pausen. Bei Anzeichen von Muedigkeit bitte umkehren.",
		},
		{
			Name:                "Luna",
			Breed:               "Border Collie",
			Size:                "medium",
			Age:                 4,
			ColorName:           "Orange",
			IsFeatured:          false,
			SpecialNeeds:        "Reaktiv gegenueber anderen Hunden - Abstand halten!",
			PickupLocation:      "Zwinger 7, Gebaeude B",
			WalkRoute:           "Feldweg Richtung Sueden, weg von den Hauptwegen",
			WalkDuration:        60,
			SpecialInstructions: "WICHTIG: Mindestens 10m Abstand zu anderen Hunden.",
		},
		{
			Name:                "Rocky",
			Breed:               "Deutscher Schaeferhund",
			Size:                "large",
			Age:                 6,
			ColorName:           "Orange",
			IsFeatured:          false,
			SpecialNeeds:        "Braucht erfahrene Fuehrung",
			PickupLocation:      "Zwinger 5, Gebaeude B",
			WalkRoute:           "Trainingsgelaende, dann Waldweg Nord",
			WalkDuration:        45,
			SpecialInstructions: "Rocky braucht klare Fuehrung. Immer Leckerlis dabei haben.",
		},
		{
			Name:                "Duke",
			Breed:               "Rottweiler",
			Size:                "large",
			Age:                 4,
			ColorName:           "Dunkelblau",
			IsFeatured:          false,
			SpecialNeeds:        "Nur fuer sehr erfahrene Hundefuehrer",
			PickupLocation:      "Zwinger 10, Gebaeude C (Schluessel beim Pfleger)",
			WalkRoute:           "Abgelegener Waldweg, keine Begegnungen",
			WalkDuration:        45,
			SpecialInstructions: "Duke ist stark und kann ziehen. Nur mit Erfahrung!",
		},
	}

	var dogIDs []int
	for _, d := range dogs {
		colorID := colorMap[d.ColorName]
		specialNeeds := d.SpecialNeeds
		pickupLocation := d.PickupLocation
		walkRoute := d.WalkRoute
		walkDuration := d.WalkDuration
		specialInstructions := d.SpecialInstructions
		morningTime := "09:00"
		eveningTime := "17:00"

		dog := &models.Dog{
			TenantID:            tenantID,
			Name:                d.Name,
			Breed:               d.Breed,
			Size:                d.Size,
			Age:                 d.Age,
			ColorID:             &colorID,
			IsFeatured:          d.IsFeatured,
			IsAvailable:         true,
			SpecialNeeds:        &specialNeeds,
			PickupLocation:      &pickupLocation,
			WalkRoute:           &walkRoute,
			WalkDuration:        &walkDuration,
			SpecialInstructions: &specialInstructions,
			DefaultMorningTime:  &morningTime,
			DefaultEveningTime:  &eveningTime,
		}

		if err := s.dogRepo.Create(dog); err != nil {
			return nil, fmt.Errorf("failed to create dog %s: %w", d.Name, err)
		}

		dogIDs = append(dogIDs, dog.ID)
	}

	log.Printf("Created %d demo dogs", len(dogs))
	return dogIDs, nil
}

// seedDemoBookings creates sample bookings
func (s *DemoSeedService) seedDemoBookings(tenantID int, userIDs map[string]int, dogIDs []int) error {
	if len(dogIDs) < 3 || len(userIDs) < 3 {
		log.Println("Not enough users or dogs for demo bookings")
		return nil
	}

	today := time.Now()
	tomorrow := today.AddDate(0, 0, 1)
	dayAfter := today.AddDate(0, 0, 2)

	bookings := []struct {
		UserKey string
		DogIdx  int
		Date    time.Time
		Time    string
		Status  string
	}{
		{"green", 0, tomorrow, "09:00", "scheduled"},  // Anna + Bella
		{"orange", 1, tomorrow, "14:00", "scheduled"}, // Bernd + Max
		{"blue", 2, dayAfter, "10:00", "scheduled"},   // Clara + Luna
	}

	for _, b := range bookings {
		userID := userIDs[b.UserKey]
		booking := &models.Booking{
			TenantID:      tenantID,
			UserID:        userID,
			DogID:         dogIDs[b.DogIdx],
			Date:          b.Date.Format("2006-01-02"),
			ScheduledTime: b.Time,
			Status:        b.Status,
		}

		if err := s.bookingRepo.Create(booking); err != nil {
			log.Printf("Warning: failed to create demo booking: %v", err)
			continue
		}
	}

	log.Printf("Created %d demo bookings", len(bookings))
	return nil
}

// initializeDemoSettings sets up default system settings for demo tenant
func (s *DemoSeedService) initializeDemoSettings(tenantID int) error {
	settings := map[string]string{
		"booking_advance_days":     "14",
		"cancellation_notice_hours": "12",
		"auto_deactivation_days":   "365",
	}

	for key, value := range settings {
		if err := s.settingsRepo.Upsert(tenantID, key, value); err != nil {
			log.Printf("Warning: failed to set setting %s: %v", key, err)
		}
	}

	return nil
}

// ResetDemoTenant resets the demo tenant to initial state
func (s *DemoSeedService) ResetDemoTenant() error {
	log.Println("Starting demo tenant reset...")

	// Get demo tenant
	tenant, err := s.tenantRepo.FindBySlug(DemoTenantSlug)
	if err != nil {
		return fmt.Errorf("failed to find demo tenant: %w", err)
	}
	if tenant == nil {
		log.Println("Demo tenant not found, creating new one...")
		return s.EnsureDemoTenant()
	}

	// Delete all data for this tenant
	if err := s.deleteAllTenantData(tenant.ID); err != nil {
		return fmt.Errorf("failed to delete tenant data: %w", err)
	}

	// Generate new password
	newPassword := s.generateRandomPassword()

	// Re-seed data
	if err := s.SeedDemoData(tenant.ID, newPassword); err != nil {
		return fmt.Errorf("failed to re-seed demo data: %w", err)
	}

	// Update demo state
	now := time.Now()
	nextReset := s.calculateNextResetTime()
	if err := s.demoStateRepo.UpdateState(tenant.ID, newPassword, &now, &nextReset); err != nil {
		return fmt.Errorf("failed to update demo state: %w", err)
	}

	log.Printf("Demo tenant reset complete (new password: %s, next reset: %s)",
		newPassword, nextReset.Format("2006-01-02 15:04"))
	return nil
}

// deleteAllTenantData removes all data for a tenant
func (s *DemoSeedService) deleteAllTenantData(tenantID int) error {
	tables := []string{
		"walk_report_photos",
		"walk_reports",
		"bookings",
		"blocked_dates",
		"user_colors",
		"color_requests",
		"experience_requests",
		"reactivation_requests",
		"dogs",
		"users",
		"color_categories",
		"system_settings",
	}

	for _, table := range tables {
		query := fmt.Sprintf("DELETE FROM %s WHERE tenant_id = ?", table)
		if _, err := s.db.Exec(query, tenantID); err != nil {
			log.Printf("Warning: failed to delete from %s: %v", table, err)
		}
	}

	return nil
}

// generateRandomPassword generates a 12-character hex password
func (s *DemoSeedService) generateRandomPassword() string {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based
		return fmt.Sprintf("demo%d", time.Now().UnixNano()%1000000)
	}
	return hex.EncodeToString(bytes)
}

// calculateNextResetTime calculates the next midnight in Europe/Berlin
func (s *DemoSeedService) calculateNextResetTime() time.Time {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		loc = time.UTC
	}

	now := time.Now().In(loc)
	// Next midnight
	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc)
	return nextMidnight
}

// GetDemoTenantID returns the demo tenant ID, or 0 if not found
func (s *DemoSeedService) GetDemoTenantID() int {
	tenant, err := s.tenantRepo.FindBySlug(DemoTenantSlug)
	if err != nil || tenant == nil {
		return 0
	}
	return tenant.ID
}
