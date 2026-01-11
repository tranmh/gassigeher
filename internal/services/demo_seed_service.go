package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/database"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// Constants for demo tenant (non-domain specific)
const (
	DemoTenantName      = "Demo Tierheim"
	DemoUserPassword    = "demo1234"
	DemoAdminPassword   = "demo1234" // Same as users for easy testing
	DefaultFederalState = "BW"
)

// formatDemoResetLogMessage creates a log message for demo tenant reset
// SECURITY: GASSI-2025-001 - This function intentionally omits the password
// to prevent sensitive data from being written to logs
func formatDemoResetLogMessage(_ string, nextReset time.Time) string {
	return fmt.Sprintf("Demo tenant reset complete (new password generated, next reset: %s)",
		nextReset.Format("2006-01-02 15:04"))
}

// DemoSeedService handles demo tenant creation and reset
type DemoSeedService struct {
	db            *database.DB
	cfg           *config.Config
	tenantRepo    *repository.TenantRepository
	demoStateRepo *repository.DemoTenantRepository
	userRepo      *repository.UserRepository
	dogRepo       *repository.DogRepository
	bookingRepo   *repository.BookingRepository
	colorRepo     *repository.ColorCategoryRepository
	userColorRepo *repository.UserColorRepository
	settingsRepo  *repository.SettingsRepository
}

// NewDemoSeedService creates a new demo seed service
func NewDemoSeedService(db *database.DB, cfg *config.Config) *DemoSeedService {
	return &DemoSeedService{
		db:            db,
		cfg:           cfg,
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
	demoSlug := s.cfg.DemoTenantSlug()
	tenant, err := s.tenantRepo.FindBySlug(demoSlug)
	if err != nil {
		return fmt.Errorf("failed to check demo tenant: %w", err)
	}

	if tenant != nil {
		// Demo tenant exists, but check if demo_tenant_state exists too
		existingState, err := s.demoStateRepo.GetState(tenant.ID)
		if err != nil {
			return fmt.Errorf("failed to check demo state: %w", err)
		}

		if existingState == nil {
			// Tenant exists but state doesn't - create the state
			log.Println("Demo tenant exists but state missing, creating state...")
			now := time.Now()
			nextReset := s.calculateNextResetTime()
			state := &models.DemoTenantState{
				TenantID:      tenant.ID,
				AdminPassword: DemoAdminPassword,
				LastResetAt:   &now,
				NextResetAt:   &nextReset,
			}
			if err := s.demoStateRepo.CreateState(state); err != nil {
				return fmt.Errorf("failed to create demo state: %w", err)
			}
			log.Printf("Demo tenant state created for tenant ID: %d", tenant.ID)
		} else {
			log.Println("Demo tenant already exists, skipping creation")
		}
		return nil
	}

	log.Println("Creating demo tenant...")

	// Use fixed password for easy testing
	adminPassword := DemoAdminPassword

	// Create demo tenant
	tenant = &models.Tenant{
		Slug:         demoSlug,
		Name:         DemoTenantName,
		Status:       models.TenantStatusActive,
		ContactEmail: s.cfg.DemoAdminEmail(),
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

	// Seed booking time rules (required for users to be able to book)
	if err := s.seedBookingTimeRules(tenantID); err != nil {
		return fmt.Errorf("failed to seed booking time rules: %w", err)
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
// Each color includes a unique pattern_icon for color-blind accessibility
func (s *DemoSeedService) seedDemoColors(tenantID int) error {
	colors := []struct {
		Name        string
		HexCode     string
		PatternIcon string
		SortOrder   int
	}{
		{"Gruen", "#22c55e", "circle", 1},
		{"Gelb", "#eab308", "star", 2},
		{"Orange", "#f97316", "triangle", 3},
		{"Hellblau", "#38bdf8", "hexagon", 4},
		{"Dunkelblau", "#3b82f6", "diamond", 5},
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

		patternIcon := c.PatternIcon
		color := &models.ColorCategory{
			TenantID:    tenantID,
			Name:        c.Name,
			HexCode:     c.HexCode,
			PatternIcon: &patternIcon,
			SortOrder:   c.SortOrder,
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
// All users have profile photos using UI Avatars service
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
		ProfilePhoto string // URL to avatar image
	}{
		{
			Key:          "admin",
			FirstName:    "Demo",
			LastName:     "Admin",
			Email:        s.cfg.DemoAdminEmail(),
			PasswordHash: string(adminHash),
			IsSuperAdmin: true,
			IsAdmin:      true,
			ColorLevel:   "blue", // Admin gets all colors
			ProfilePhoto: "/assets/images/demo/users/admin.svg",
		},
		{
			Key:          "green",
			FirstName:    "Anna",
			LastName:     "Mueller",
			Email:        s.cfg.DemoUserEmail("anna"),
			PasswordHash: string(demoHash),
			IsSuperAdmin: false,
			IsAdmin:      false,
			ColorLevel:   "single_green", // Anna can only book green dogs (e.g., Max, Luna, Mia)
			ProfilePhoto: "/assets/images/demo/users/anna.svg",
		},
		{
			Key:          "yellow",
			FirstName:    "Bernd",
			LastName:     "Schmidt",
			Email:        s.cfg.DemoUserEmail("bernd"),
			PasswordHash: string(demoHash),
			IsSuperAdmin: false,
			IsAdmin:      false,
			ColorLevel:   "single_yellow", // Bernd can only book yellow dogs (e.g., Bello, Bruno)
			ProfilePhoto: "/assets/images/demo/users/bernd.svg",
		},
		{
			Key:          "orange",
			FirstName:    "Clara",
			LastName:     "Weber",
			Email:        s.cfg.DemoUserEmail("clara"),
			PasswordHash: string(demoHash),
			IsSuperAdmin: false,
			IsAdmin:      false,
			ColorLevel:   "single_orange", // Clara can only book orange dogs (e.g., Rocky)
			ProfilePhoto: "/assets/images/demo/users/clara.svg",
		},
	}

	for _, u := range users {
		email := u.Email
		profilePhoto := u.ProfilePhoto
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
			ProfilePhoto:    &profilePhoto,
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

	log.Printf("Created %d demo users (all with profile photos)", len(users))
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
	// Single color levels give exactly ONE color to demo users
	// "blue" level gives ALL colors (for admin)
	var colorNames []string
	switch level {
	case "single_green":
		colorNames = []string{"Gruen"}
	case "single_yellow":
		colorNames = []string{"Gelb"}
	case "single_orange":
		colorNames = []string{"Orange"}
	case "blue":
		// Admin gets all colors
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
// All dogs are featured with photos and external links for a visually appealing demo
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

	// External link for all demo dogs - points to main Gassigeher site
	externalLink := s.cfg.MainSiteURL()

	dogs := []struct {
		Name                string
		Breed               string
		Size                string
		Age                 int
		ColorName           string
		Photo               string // Direct URL to Unsplash image
		SpecialNeeds        string
		PickupLocation      string
		WalkRoute           string
		WalkDuration        int
		SpecialInstructions string
	}{
		{
			Name:                "Max",
			Breed:               "Deutscher Schäferhund",
			Size:                "large",
			Age:                 4,
			ColorName:           "Gruen",
			Photo:               "/assets/images/demo/dogs/max.jpg",
			SpecialNeeds:        "Keine besonderen Bedürfnisse",
			PickupLocation:      "Zwinger 1, Gebäude A",
			WalkRoute:           "Waldweg hinter dem Tierheim",
			WalkDuration:        45,
			SpecialInstructions: "Max ist sehr freundlich und gut erzogen. Perfekt für Anfänger.",
		},
		{
			Name:                "Luna",
			Breed:               "Labrador Retriever",
			Size:                "large",
			Age:                 3,
			ColorName:           "Gruen",
			Photo:               "/assets/images/demo/dogs/luna.jpg",
			SpecialNeeds:        "Liebt Wasser - nicht am Teich vorbeigehen!",
			PickupLocation:      "Zwinger 2, Gebäude A",
			WalkRoute:           "Kurze Runde durch den Park",
			WalkDuration:        40,
			SpecialInstructions: "Luna ist verspielt und freundlich. Mag andere Hunde.",
		},
		{
			Name:                "Bello",
			Breed:               "Mischling",
			Size:                "medium",
			Age:                 5,
			ColorName:           "Gelb",
			Photo:               "/assets/images/demo/dogs/bello.jpg",
			SpecialNeeds:        "Leichte Arthrose - kürzere Spaziergänge bevorzugt",
			PickupLocation:      "Zwinger 4, Gebäude A",
			WalkRoute:           "Flache Strecke ohne Treppen",
			WalkDuration:        30,
			SpecialInstructions: "Bello braucht Pausen. Bei Müdigkeit bitte umkehren.",
		},
		{
			Name:                "Rocky",
			Breed:               "Rottweiler",
			Size:                "large",
			Age:                 6,
			ColorName:           "Orange",
			Photo:               "/assets/images/demo/dogs/rocky.jpg",
			SpecialNeeds:        "Braucht erfahrene Führung - stark an der Leine",
			PickupLocation:      "Zwinger 7, Gebäude B",
			WalkRoute:           "Trainingsgelände, dann Waldweg Nord",
			WalkDuration:        45,
			SpecialInstructions: "Rocky braucht klare Führung. Immer Leckerlis dabei haben.",
		},
		{
			Name:                "Mia",
			Breed:               "Beagle",
			Size:                "small",
			Age:                 2,
			ColorName:           "Gruen",
			Photo:               "/assets/images/demo/dogs/mia.jpg",
			SpecialNeeds:        "Folgt ihrer Nase - immer an der Leine halten!",
			PickupLocation:      "Zwinger 3, Gebäude A",
			WalkRoute:           "Eingezäunter Bereich oder Leine",
			WalkDuration:        35,
			SpecialInstructions: "Mia ist neugierig. Nicht ableinen, sie läuft weg!",
		},
		{
			Name:                "Bruno",
			Breed:               "Boxer",
			Size:                "large",
			Age:                 4,
			ColorName:           "Gelb",
			Photo:               "/assets/images/demo/dogs/bruno.jpg",
			SpecialNeeds:        "Viel Energie - braucht aktiven Spaziergang",
			PickupLocation:      "Zwinger 5, Gebäude B",
			WalkRoute:           "Lange Runde durch den Wald",
			WalkDuration:        60,
			SpecialInstructions: "Bruno liebt es zu rennen. Ideal für sportliche Gassigeher.",
		},
		{
			Name:                "Lotte",
			Breed:               "Dackel",
			Size:                "small",
			Age:                 7,
			ColorName:           "Gruen",
			Photo:               "/assets/images/demo/dogs/lotte.jpg",
			SpecialNeeds:        "Rückenschonend - keine Treppen!",
			PickupLocation:      "Zwinger 6, Gebäude A",
			WalkRoute:           "Ebene Strecke, keine Steigungen",
			WalkDuration:        25,
			SpecialInstructions: "Lotte darf nicht springen. Bitte hochheben bei Hindernissen.",
		},
		{
			Name:                "Thor",
			Breed:               "Husky",
			Size:                "large",
			Age:                 3,
			ColorName:           "Dunkelblau",
			Photo:               "/assets/images/demo/dogs/thor.jpg",
			SpecialNeeds:        "Sehr stark - nur für erfahrene Hundeführer",
			PickupLocation:      "Zwinger 10, Gebäude C (Schlüssel beim Pfleger)",
			WalkRoute:           "Abgelegener Waldweg, wenig Begegnungen",
			WalkDuration:        60,
			SpecialInstructions: "Thor zieht stark und braucht konsequente Führung. Nur mit Erfahrung!",
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
		photo := d.Photo
		extLink := externalLink
		morningTime := "09:00"
		eveningTime := "17:00"

		dog := &models.Dog{
			TenantID:            tenantID,
			Name:                d.Name,
			Breed:               d.Breed,
			Size:                d.Size,
			Age:                 d.Age,
			ColorID:             &colorID,
			IsFeatured:          true, // All demo dogs are featured
			IsAvailable:         true,
			Photo:               &photo,
			ExternalLink:        &extLink,
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

	log.Printf("Created %d demo dogs (all featured with photos)", len(dogs))
	return dogIDs, nil
}

// seedDemoBookings creates sample bookings with realistic past and future data
func (s *DemoSeedService) seedDemoBookings(tenantID int, userIDs map[string]int, dogIDs []int) error {
	if len(dogIDs) < 5 || len(userIDs) < 3 {
		log.Println("Not enough users or dogs for demo bookings")
		return nil
	}

	today := time.Now()

	// Past dates (completed walks)
	yesterday := today.AddDate(0, 0, -1)
	twoDaysAgo := today.AddDate(0, 0, -2)
	threeDaysAgo := today.AddDate(0, 0, -3)
	fourDaysAgo := today.AddDate(0, 0, -4)
	fiveDaysAgo := today.AddDate(0, 0, -5)

	// Future dates (scheduled walks)
	tomorrow := today.AddDate(0, 0, 1)
	dayAfter := today.AddDate(0, 0, 2)
	threeDaysFromNow := today.AddDate(0, 0, 3)
	fourDaysFromNow := today.AddDate(0, 0, 4)

	bookings := []struct {
		UserKey string
		DogIdx  int
		Date    time.Time
		Time    string
		Status  string
		Notes   string // Notes for completed walks
	}{
		// Past completed walks with notes
		{"green", 0, fiveDaysAgo, "09:00", "completed", "Max war heute sehr gut gelaunt! Wir sind die lange Runde gelaufen. Er hat brav auf andere Hunde reagiert."},
		{"orange", 3, fiveDaysAgo, "14:00", "completed", "Rocky war anfangs etwas aufgeregt, hat sich aber schnell beruhigt. Leckerlis haben gut geholfen."},
		{"blue", 7, fourDaysAgo, "10:00", "completed", "Thor hat heute viel Energie gehabt. Wir mussten mehrere Pausen machen, damit ich mithalten konnte!"},
		{"green", 1, fourDaysAgo, "15:00", "completed", "Luna ist so ein Schatz! Sie wollte unbedingt zum Teich, aber ich habe sie erfolgreich abgelenkt."},
		{"orange", 2, threeDaysAgo, "09:30", "completed", "Bello hat heute seinen guten Tag. Er hat sogar ein bisschen gerannt, obwohl er sonst lieber gemütlich geht."},
		{"blue", 4, threeDaysAgo, "11:00", "completed", "Mia hat wie immer versucht, jeder Spur zu folgen. Zum Glück war sie an der Leine!"},
		{"green", 5, twoDaysAgo, "14:00", "completed", "Bruno hatte richtig Spass heute. Wir haben Ball gespielt im eingezäunten Bereich."},
		{"orange", 6, twoDaysAgo, "16:00", "completed", "Lotte ist so süss! Musste sie zweimal über den Baumstamm heben. Sie hat sich gefreut."},
		{"blue", 0, yesterday, "09:00", "completed", "Max ist wirklich ein Traumhund. Perfektes Verhalten, hat sogar ein Kind begrüsst ohne zu springen."},
		{"green", 3, yesterday, "15:00", "completed", "Rocky macht Fortschritte! Er hat heute einen anderen Hund gesehen und ist ruhig geblieben."},

		// Future scheduled walks
		{"green", 0, tomorrow, "09:00", "scheduled", ""},         // Anna + Max
		{"orange", 1, tomorrow, "14:00", "scheduled", ""},        // Bernd + Luna
		{"blue", 2, tomorrow, "11:00", "scheduled", ""},          // Clara + Bello
		{"green", 4, dayAfter, "10:00", "scheduled", ""},         // Anna + Mia
		{"orange", 5, dayAfter, "15:00", "scheduled", ""},        // Bernd + Bruno
		{"blue", 7, threeDaysFromNow, "09:30", "scheduled", ""},  // Clara + Thor
		{"green", 6, threeDaysFromNow, "14:00", "scheduled", ""}, // Anna + Lotte
		{"orange", 0, fourDaysFromNow, "10:00", "scheduled", ""}, // Bernd + Max
	}

	completedCount := 0
	scheduledCount := 0

	for _, b := range bookings {
		userID := userIDs[b.UserKey]
		var notes *string
		if b.Notes != "" {
			notes = &b.Notes
		}

		booking := &models.Booking{
			TenantID:      tenantID,
			UserID:        userID,
			DogID:         dogIDs[b.DogIdx],
			Date:          b.Date.Format("2006-01-02"),
			ScheduledTime: b.Time,
			Status:        b.Status,
			UserNotes:     notes,
		}

		if err := s.bookingRepo.Create(booking); err != nil {
			log.Printf("Warning: failed to create demo booking: %v", err)
			continue
		}

		if b.Status == "completed" {
			completedCount++
		} else {
			scheduledCount++
		}
	}

	log.Printf("Created %d demo bookings (%d completed, %d scheduled)", len(bookings), completedCount, scheduledCount)
	return nil
}

// seedBookingTimeRules creates default booking time rules for the demo tenant
// Without these rules, users cannot book any time slots
func (s *DemoSeedService) seedBookingTimeRules(tenantID int) error {
	// Check if rules already exist (idempotent)
	var existingCount int
	countQuery := s.db.Rebind("SELECT COUNT(*) FROM booking_time_rules WHERE tenant_id = ?")
	err := s.db.QueryRow(countQuery, tenantID).Scan(&existingCount)
	if err != nil {
		return fmt.Errorf("failed to check existing rules: %w", err)
	}
	if existingCount > 0 {
		log.Printf("Booking time rules already exist for tenant %d, skipping", tenantID)
		return nil
	}

	// Default rules matching ProvisioningService
	rules := []struct {
		DayType   string
		RuleName  string
		StartTime string
		EndTime   string
		IsBlocked bool
	}{
		{"weekday", "morning", "08:00", "12:00", false},
		{"weekday", "lunch", "12:00", "14:00", true},
		{"weekday", "afternoon", "14:00", "18:00", false},
		{"weekend", "morning", "09:00", "12:00", false},
		{"weekend", "afternoon", "14:00", "17:00", false},
		{"holiday", "morning", "10:00", "12:00", false},
		{"holiday", "afternoon", "14:00", "16:00", false},
	}

	insertQuery := s.db.Rebind(`INSERT INTO booking_time_rules (tenant_id, day_type, rule_name, start_time, end_time, is_blocked) VALUES (?, ?, ?, ?, ?, ?)`)
	for _, r := range rules {
		_, err := s.db.Exec(
			insertQuery,
			tenantID, r.DayType, r.RuleName, r.StartTime, r.EndTime, r.IsBlocked,
		)
		if err != nil {
			return fmt.Errorf("failed to create booking time rule %s/%s: %w", r.DayType, r.RuleName, err)
		}
	}

	log.Printf("Created %d booking time rules for demo tenant", len(rules))
	return nil
}

// initializeDemoSettings sets up default system settings for demo tenant
func (s *DemoSeedService) initializeDemoSettings(tenantID int) error {
	settings := map[string]string{
		"booking_advance_days":      "14",
		"cancellation_notice_hours": "12",
		"auto_deactivation_days":    "365",
		// Demo branding - use bundled assets instead of external URLs
		"site_logo":    "/assets/images/demo/branding/logo.png",
		"site_favicon": "/assets/images/demo/branding/favicon.png",
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
	tenant, err := s.tenantRepo.FindBySlug(s.cfg.DemoTenantSlug())
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

	// Use fixed password for easy testing
	newPassword := DemoAdminPassword

	// Re-seed data
	if err := s.SeedDemoData(tenant.ID, newPassword); err != nil {
		return fmt.Errorf("failed to re-seed demo data: %w", err)
	}

	// Update or create demo state
	now := time.Now()
	nextReset := s.calculateNextResetTime()

	// Check if demo state exists
	existingState, err := s.demoStateRepo.GetState(tenant.ID)
	if err != nil {
		return fmt.Errorf("failed to check demo state: %w", err)
	}

	if existingState == nil {
		// Create demo state if it doesn't exist
		state := &models.DemoTenantState{
			TenantID:      tenant.ID,
			AdminPassword: newPassword,
			LastResetAt:   &now,
			NextResetAt:   &nextReset,
		}
		if err := s.demoStateRepo.CreateState(state); err != nil {
			return fmt.Errorf("failed to create demo state: %w", err)
		}
	} else {
		// Update existing demo state
		if err := s.demoStateRepo.UpdateState(tenant.ID, newPassword, &now, &nextReset); err != nil {
			return fmt.Errorf("failed to update demo state: %w", err)
		}
	}

	// SECURITY: GASSI-2025-001 - Use secure log message that doesn't expose password
	log.Print(formatDemoResetLogMessage(newPassword, nextReset))
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
		"booking_time_rules",
		"custom_holidays",
	}

	for _, table := range tables {
		query := s.db.Rebind(fmt.Sprintf("DELETE FROM %s WHERE tenant_id = ?", table))
		if _, err := s.db.Exec(query, tenantID); err != nil {
			log.Printf("Warning: failed to delete from %s: %v", table, err)
		}
	}

	// Reset tenant_settings to defaults (don't delete - it has 1:1 relationship with tenant)
	updateQuery := s.db.Rebind(`
		UPDATE tenant_settings SET
			theme_preset = 'classic',
			color_primary = NULL,
			color_secondary = NULL,
			color_accent = NULL,
			color_background = NULL,
			color_text = NULL,
			logo_url = NULL,
			favicon_url = NULL,
			welcome_message = NULL,
			tagline = NULL,
			description = NULL,
			footer_text = NULL,
			website_url = NULL,
			donation_url = NULL,
			updated_at = CURRENT_TIMESTAMP
		WHERE tenant_id = ?
	`)
	_, err := s.db.Exec(updateQuery, tenantID)
	if err != nil {
		log.Printf("Warning: failed to reset tenant_settings: %v", err)
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
	tenant, err := s.tenantRepo.FindBySlug(s.cfg.DemoTenantSlug())
	if err != nil || tenant == nil {
		return 0
	}
	return tenant.ID
}
