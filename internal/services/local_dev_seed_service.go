package services

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/database"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// LocalDevProfile defines different data profiles for local development tenants
type LocalDevProfile string

const (
	// ProfileEmpty - Only tenant + super admin, no dogs/users/bookings
	ProfileEmpty LocalDevProfile = "empty"
	// ProfileSmall - Basic shelter data (1 admin + 3 users, 5 dogs, 10 bookings)
	ProfileSmall LocalDevProfile = "small"
	// ProfileMedium - More data (1 admin + 10 users, 15 dogs, 50 bookings)
	ProfileMedium LocalDevProfile = "medium"
	// ProfileStress - Edge cases and stress test (1 admin + 100 users, 50 dogs, 500 bookings)
	ProfileStress LocalDevProfile = "stress"
)

// LocalDevPassword is the fixed password for all local development tenants
const LocalDevPassword = "localdev1234"

// LocalDevTenantConfig defines a local development tenant
type LocalDevTenantConfig struct {
	Slug    string
	Name    string
	Profile LocalDevProfile
}

// LocalDevTenants defines the local development tenants to create
var LocalDevTenants = []LocalDevTenantConfig{
	{"demo1", "Demo 1 - Leeres Tierheim", ProfileEmpty},
	{"demo2", "Demo 2 - Kleines Tierheim", ProfileSmall},
	{"demo3", "Demo 3 - Mittleres Tierheim", ProfileMedium},
	{"demo4", "Demo 4 - Stress Test", ProfileStress},
}

// LocalDevSeedService handles local development tenant creation and reset
type LocalDevSeedService struct {
	db            *database.DB
	cfg           *config.Config
	tenantRepo    *repository.TenantRepository
	userRepo      *repository.UserRepository
	dogRepo       *repository.DogRepository
	bookingRepo   *repository.BookingRepository
	colorRepo     *repository.ColorCategoryRepository
	userColorRepo *repository.UserColorRepository
	settingsRepo  *repository.SettingsRepository
	blockedRepo   *repository.BlockedDateRepository
}

// NewLocalDevSeedService creates a new local dev seed service
func NewLocalDevSeedService(db *database.DB, cfg *config.Config) *LocalDevSeedService {
	return &LocalDevSeedService{
		db:            db,
		cfg:           cfg,
		tenantRepo:    repository.NewTenantRepository(db),
		userRepo:      repository.NewUserRepository(db),
		dogRepo:       repository.NewDogRepository(db),
		bookingRepo:   repository.NewBookingRepository(db),
		colorRepo:     repository.NewColorCategoryRepository(db),
		userColorRepo: repository.NewUserColorRepository(db),
		settingsRepo:  repository.NewSettingsRepository(db),
		blockedRepo:   repository.NewBlockedDateRepository(db),
	}
}

// EnsureLocalDevTenants creates all local development tenants if they don't exist
func (s *LocalDevSeedService) EnsureLocalDevTenants() error {
	log.Println("Ensuring local development tenants...")

	for _, cfg := range LocalDevTenants {
		if err := s.ensureTenant(cfg); err != nil {
			return fmt.Errorf("failed to ensure tenant %s: %w", cfg.Slug, err)
		}
	}

	log.Println("Local development tenants ready")
	return nil
}

// ensureTenant creates a single tenant if it doesn't exist
func (s *LocalDevSeedService) ensureTenant(cfg LocalDevTenantConfig) error {
	// Check if tenant already exists
	tenant, err := s.tenantRepo.FindBySlug(cfg.Slug)
	if err != nil {
		return fmt.Errorf("failed to check tenant %s: %w", cfg.Slug, err)
	}

	if tenant != nil {
		log.Printf("Local dev tenant '%s' already exists, skipping", cfg.Slug)
		return nil
	}

	log.Printf("Creating local dev tenant '%s' with profile '%s'...", cfg.Slug, cfg.Profile)

	// Create tenant using config for domain
	adminEmail := fmt.Sprintf("admin@%s.%s", cfg.Slug, s.cfg.BaseDomain)
	tenant = &models.Tenant{
		Slug:         cfg.Slug,
		Name:         cfg.Name,
		Status:       models.TenantStatusActive,
		ContactEmail: adminEmail,
		FederalState: "BW",
		IsDemo:       false, // Not a demo tenant - data persists
	}

	if err := s.tenantRepo.Create(tenant); err != nil {
		return fmt.Errorf("failed to create tenant %s: %w", cfg.Slug, err)
	}

	// Create tenant settings
	settings := &models.TenantSettings{
		TenantID:    tenant.ID,
		ThemePreset: "classic",
	}
	if err := s.tenantRepo.CreateSettings(settings); err != nil {
		return fmt.Errorf("failed to create tenant settings: %w", err)
	}

	// Seed data based on profile
	if err := s.seedProfile(tenant.ID, cfg.Profile, adminEmail); err != nil {
		return fmt.Errorf("failed to seed profile %s: %w", cfg.Profile, err)
	}

	log.Printf("Local dev tenant '%s' created successfully (ID: %d)", cfg.Slug, tenant.ID)
	return nil
}

// seedProfile populates the tenant with data based on the profile
func (s *LocalDevSeedService) seedProfile(tenantID int, profile LocalDevProfile, adminEmail string) error {
	// Always seed colors first (needed for dogs and user assignments)
	if err := s.seedColors(tenantID); err != nil {
		return fmt.Errorf("failed to seed colors: %w", err)
	}

	// Always seed system settings
	if err := s.seedSettings(tenantID); err != nil {
		return fmt.Errorf("failed to seed settings: %w", err)
	}

	// Always seed booking time rules
	if err := s.seedBookingRules(tenantID); err != nil {
		return fmt.Errorf("failed to seed booking rules: %w", err)
	}

	// Always create admin user
	adminID, err := s.createAdminUser(tenantID, adminEmail)
	if err != nil {
		return fmt.Errorf("failed to create admin: %w", err)
	}

	// Profile-specific seeding
	switch profile {
	case ProfileEmpty:
		// Only admin, no other data
		log.Printf("Profile empty: created admin only for tenant %d", tenantID)

	case ProfileSmall:
		if err := s.seedSmallProfile(tenantID, adminID); err != nil {
			return err
		}

	case ProfileMedium:
		if err := s.seedMediumProfile(tenantID, adminID); err != nil {
			return err
		}

	case ProfileStress:
		if err := s.seedStressProfile(tenantID, adminID); err != nil {
			return err
		}
	}

	return nil
}

// seedColors creates default color categories
// Each color includes a unique pattern_icon for color-blind accessibility
func (s *LocalDevSeedService) seedColors(tenantID int) error {
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

	for _, c := range colors {
		existing, err := s.colorRepo.FindByName(tenantID, c.Name)
		if err != nil {
			// Log warning but continue
			log.Printf("Warning: error checking color %s: %v", c.Name, err)
		}
		if existing != nil {
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
			// Ignore duplicate errors (may happen if constraint is global)
			if strings.Contains(err.Error(), "UNIQUE constraint") ||
				strings.Contains(err.Error(), "Duplicate entry") {
				log.Printf("Warning: color %s already exists, skipping", c.Name)
				continue
			}
			return fmt.Errorf("failed to create color category: %w", err)
		}
	}

	return nil
}

// seedSettings creates default system settings
func (s *LocalDevSeedService) seedSettings(tenantID int) error {
	settings := map[string]string{
		"booking_advance_days":      "14",
		"cancellation_notice_hours": "12",
		"auto_deactivation_days":    "365",
	}

	for key, value := range settings {
		if err := s.settingsRepo.Upsert(tenantID, key, value); err != nil {
			log.Printf("Warning: failed to set setting %s: %v", key, err)
		}
	}

	return nil
}

// seedBookingRules creates default booking time rules for a tenant
func (s *LocalDevSeedService) seedBookingRules(tenantID int) error {
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
			log.Printf("Warning: failed to create booking rule %s/%s: %v", r.DayType, r.RuleName, err)
		}
	}

	return nil
}

// createAdminUser creates the admin user for a tenant
func (s *LocalDevSeedService) createAdminUser(tenantID int, email string) (int, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(LocalDevPassword), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}

	now := time.Now()
	passwordHash := string(hash)
	user := &models.User{
		TenantID:        tenantID,
		FirstName:       "Admin",
		LastName:        "Local",
		Email:           &email,
		PasswordHash:    &passwordHash,
		IsAdmin:         true,
		IsSuperAdmin:    true,
		IsVerified:      true,
		IsActive:        true,
		TermsAcceptedAt: now,
		LastActivityAt:  now,
	}

	if err := s.userRepo.Create(user); err != nil {
		return 0, err
	}

	// Assign all colors to admin
	if err := s.assignUserColors(tenantID, user.ID, "blue"); err != nil {
		log.Printf("Warning: failed to assign colors to admin: %v", err)
	}

	return user.ID, nil
}

// seedSmallProfile: 1 admin + 3 users, 5 dogs, 10 bookings
func (s *LocalDevSeedService) seedSmallProfile(tenantID, adminID int) error {
	// Create 3 regular users
	userIDs, err := s.createUsers(tenantID, []userConfig{
		{"Anna", "Gruen", "green"},
		{"Bernd", "Orange", "orange"},
		{"Clara", "Blau", "blue"},
	})
	if err != nil {
		return err
	}
	userIDs["admin"] = adminID

	// Create 5 dogs
	dogIDs, err := s.createDogs(tenantID, []dogConfig{
		{"Bella", "Labrador Retriever", "large", 3, "Gruen", true},
		{"Max", "Golden Retriever", "large", 5, "Gruen", true},
		{"Luna", "Border Collie", "medium", 4, "Orange", false},
		{"Rocky", "Deutscher Schäferhund", "large", 6, "Orange", false},
		{"Duke", "Rottweiler", "large", 4, "Dunkelblau", false},
	})
	if err != nil {
		return err
	}

	// Create 10 bookings (5 past, 5 future)
	return s.createBookings(tenantID, userIDs, dogIDs, 5, 5)
}

// seedMediumProfile: 1 admin + 10 users, 15 dogs, 50 bookings
func (s *LocalDevSeedService) seedMediumProfile(tenantID, adminID int) error {
	// Create 10 regular users
	users := []userConfig{
		{"Anna", "Gruen", "green"},
		{"Bernd", "Orange", "orange"},
		{"Clara", "Blau", "blue"},
		{"David", "Mueller", "green"},
		{"Eva", "Schmidt", "orange"},
		{"Felix", "Weber", "blue"},
		{"Greta", "Becker", "green"},
		{"Hans", "Fischer", "orange"},
		{"Ines", "Koch", "blue"},
		{"Jan", "Richter", "green"},
	}
	userIDs, err := s.createUsers(tenantID, users)
	if err != nil {
		return err
	}
	userIDs["admin"] = adminID

	// Create 15 dogs
	dogs := []dogConfig{
		{"Bella", "Labrador Retriever", "large", 3, "Gruen", true},
		{"Max", "Golden Retriever", "large", 5, "Gruen", true},
		{"Luna", "Border Collie", "medium", 4, "Orange", false},
		{"Rocky", "Deutscher Schäferhund", "large", 6, "Orange", false},
		{"Duke", "Rottweiler", "large", 4, "Dunkelblau", false},
		{"Buddy", "Beagle", "medium", 2, "Gruen", false},
		{"Lucy", "Dackel", "small", 7, "Gruen", false},
		{"Charlie", "Husky", "large", 3, "Orange", false},
		{"Daisy", "Pudel", "medium", 5, "Gelb", false},
		{"Cooper", "Boxer", "large", 4, "Orange", false},
		{"Sadie", "Mischling", "medium", 6, "Gruen", false},
		{"Bailey", "Australian Shepherd", "large", 2, "Hellblau", false},
		{"Molly", "Shih Tzu", "small", 8, "Gruen", false},
		{"Tucker", "Dobermann", "large", 5, "Dunkelblau", false},
		{"Maggie", "Chihuahua", "small", 4, "Gelb", false},
	}
	dogIDs, err := s.createDogs(tenantID, dogs)
	if err != nil {
		return err
	}

	// Create 50 bookings (25 past, 25 future)
	if err := s.createBookings(tenantID, userIDs, dogIDs, 25, 25); err != nil {
		return err
	}

	// Add 5 blocked dates
	return s.createBlockedDates(tenantID, adminID, 5)
}

// seedStressProfile: 1 admin + 100 users, 50 dogs, 500 bookings
func (s *LocalDevSeedService) seedStressProfile(tenantID, adminID int) error {
	// Create 100 regular users with varying levels
	var users []userConfig
	firstNames := []string{"Anna", "Bernd", "Clara", "David", "Eva", "Felix", "Greta", "Hans", "Ines", "Jan", "Katrin", "Lars", "Maria", "Nils", "Olga", "Peter", "Rita", "Stefan", "Tina", "Uwe"}
	lastNames := []string{"Mueller", "Schmidt", "Weber", "Becker", "Fischer", "Koch", "Richter", "Klein", "Wolf", "Schroeder"}
	levels := []string{"green", "green", "green", "orange", "orange", "blue"} // 50% green, 33% orange, 17% blue

	for i := 0; i < 100; i++ {
		firstName := firstNames[i%len(firstNames)]
		lastName := fmt.Sprintf("%s%d", lastNames[i%len(lastNames)], i/10)
		level := levels[i%len(levels)]
		users = append(users, userConfig{firstName, lastName, level})
	}

	userIDs, err := s.createUsers(tenantID, users)
	if err != nil {
		return err
	}
	userIDs["admin"] = adminID

	// Create 50 dogs
	var dogs []dogConfig
	breeds := []string{"Labrador", "Golden Retriever", "Border Collie", "Schäferhund", "Rottweiler", "Beagle", "Dackel", "Husky", "Pudel", "Boxer"}
	sizes := []string{"small", "medium", "large"}
	colorNames := []string{"Gruen", "Gelb", "Orange", "Hellblau", "Dunkelblau"}

	for i := 0; i < 50; i++ {
		name := fmt.Sprintf("Hund%d", i+1)
		breed := breeds[i%len(breeds)]
		size := sizes[i%len(sizes)]
		age := (i % 12) + 1
		color := colorNames[i%len(colorNames)]
		featured := i < 5 // First 5 are featured

		dogs = append(dogs, dogConfig{name, breed, size, age, color, featured})
	}

	dogIDs, err := s.createDogs(tenantID, dogs)
	if err != nil {
		return err
	}

	// Create 500 bookings (250 past, 250 future)
	if err := s.createBookings(tenantID, userIDs, dogIDs, 250, 250); err != nil {
		return err
	}

	// Add edge case users
	if err := s.createEdgeCaseUsers(tenantID); err != nil {
		log.Printf("Warning: failed to create edge case users: %v", err)
	}

	// Add 20 blocked dates
	return s.createBlockedDates(tenantID, adminID, 20)
}

// userConfig for user creation
type userConfig struct {
	FirstName string
	LastName  string
	Level     string // green, orange, blue
}

// dogConfig for dog creation
type dogConfig struct {
	Name       string
	Breed      string
	Size       string
	Age        int
	ColorName  string
	IsFeatured bool
}

// createUsers creates multiple users and returns their IDs
func (s *LocalDevSeedService) createUsers(tenantID int, configs []userConfig) (map[string]int, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(LocalDevPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	userIDs := make(map[string]int)
	now := time.Now()
	passwordHash := string(hash)

	for i, cfg := range configs {
		email := fmt.Sprintf("%s.%s.%d@local.test", strings.ToLower(cfg.FirstName), strings.ToLower(cfg.LastName), i)
		user := &models.User{
			TenantID:        tenantID,
			FirstName:       cfg.FirstName,
			LastName:        cfg.LastName,
			Email:           &email,
			PasswordHash:    &passwordHash,
			IsAdmin:         false,
			IsSuperAdmin:    false,
			IsVerified:      true,
			IsActive:        true,
			TermsAcceptedAt: now,
			LastActivityAt:  now,
		}

		if err := s.userRepo.Create(user); err != nil {
			return nil, fmt.Errorf("failed to create user %s: %w", email, err)
		}

		userIDs[cfg.Level] = user.ID // Last user of each level wins (for booking assignment)

		// Assign colors based on level
		if err := s.assignUserColors(tenantID, user.ID, cfg.Level); err != nil {
			log.Printf("Warning: failed to assign colors to user %d: %v", user.ID, err)
		}
	}

	return userIDs, nil
}

// assignUserColors assigns colors based on user level
func (s *LocalDevSeedService) assignUserColors(tenantID, userID int, level string) error {
	colors, err := s.colorRepo.FindAll(tenantID)
	if err != nil {
		return err
	}

	colorMap := make(map[string]int)
	for _, c := range colors {
		colorMap[c.Name] = c.ID
	}

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

	for _, name := range colorNames {
		if colorID, ok := colorMap[name]; ok {
			if err := s.userColorRepo.AddColorToUser(tenantID, userID, colorID, userID); err != nil {
				// Ignore duplicate errors
			}
		}
	}

	return nil
}

// createDogs creates multiple dogs and returns their IDs
func (s *LocalDevSeedService) createDogs(tenantID int, configs []dogConfig) ([]int, error) {
	colors, err := s.colorRepo.FindAll(tenantID)
	if err != nil {
		return nil, err
	}

	colorMap := make(map[string]int)
	for _, c := range colors {
		colorMap[c.Name] = c.ID
	}

	var dogIDs []int
	for _, cfg := range configs {
		colorID := colorMap[cfg.ColorName]
		morningTime := "09:00"
		eveningTime := "17:00"

		dog := &models.Dog{
			TenantID:           tenantID,
			Name:               cfg.Name,
			Breed:              cfg.Breed,
			Size:               cfg.Size,
			Age:                cfg.Age,
			ColorID:            &colorID,
			IsFeatured:         cfg.IsFeatured,
			IsAvailable:        true,
			DefaultMorningTime: &morningTime,
			DefaultEveningTime: &eveningTime,
		}

		if err := s.dogRepo.Create(dog); err != nil {
			return nil, fmt.Errorf("failed to create dog %s: %w", cfg.Name, err)
		}

		dogIDs = append(dogIDs, dog.ID)
	}

	return dogIDs, nil
}

// createBookings creates bookings (past and future)
func (s *LocalDevSeedService) createBookings(tenantID int, userIDs map[string]int, dogIDs []int, pastCount, futureCount int) error {
	if len(dogIDs) == 0 {
		return nil
	}

	// Get user IDs slice
	var userIDList []int
	for _, id := range userIDs {
		userIDList = append(userIDList, id)
	}
	if len(userIDList) == 0 {
		return nil
	}

	today := time.Now()
	times := []string{"09:00", "10:00", "11:00", "14:00", "15:00", "16:00"}

	// Create past bookings
	for i := 0; i < pastCount; i++ {
		date := today.AddDate(0, 0, -(i/3 + 1))
		booking := &models.Booking{
			TenantID:      tenantID,
			UserID:        userIDList[i%len(userIDList)],
			DogID:         dogIDs[i%len(dogIDs)],
			Date:          date.Format("2006-01-02"),
			ScheduledTime: times[i%len(times)],
			Status:        "completed",
		}
		if err := s.bookingRepo.Create(booking); err != nil {
			log.Printf("Warning: failed to create past booking: %v", err)
		}
	}

	// Create future bookings
	for i := 0; i < futureCount; i++ {
		date := today.AddDate(0, 0, (i/3 + 1))
		booking := &models.Booking{
			TenantID:      tenantID,
			UserID:        userIDList[i%len(userIDList)],
			DogID:         dogIDs[i%len(dogIDs)],
			Date:          date.Format("2006-01-02"),
			ScheduledTime: times[i%len(times)],
			Status:        "scheduled",
		}
		if err := s.bookingRepo.Create(booking); err != nil {
			log.Printf("Warning: failed to create future booking: %v", err)
		}
	}

	log.Printf("Created %d bookings for tenant %d", pastCount+futureCount, tenantID)
	return nil
}

// createBlockedDates creates blocked dates for testing
func (s *LocalDevSeedService) createBlockedDates(tenantID, adminID, count int) error {
	today := time.Now()

	for i := 0; i < count; i++ {
		// Spread blocked dates over next 3 months
		daysAhead := rand.Intn(90) + 1
		date := today.AddDate(0, 0, daysAhead)

		blocked := &models.BlockedDate{
			TenantID:  tenantID,
			Date:      date.Format("2006-01-02"),
			Reason:    fmt.Sprintf("Testsperrung %d", i+1),
			CreatedBy: adminID,
		}

		if err := s.blockedRepo.Create(blocked); err != nil {
			// Ignore duplicates
			log.Printf("Warning: failed to create blocked date: %v", err)
		}
	}

	return nil
}

// createEdgeCaseUsers creates users with edge case data for stress testing
func (s *LocalDevSeedService) createEdgeCaseUsers(tenantID int) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(LocalDevPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	now := time.Now()
	passwordHash := string(hash)

	// Long name (255 chars)
	longName := strings.Repeat("A", 255)
	longEmail := "longname@local.test"
	longUser := &models.User{
		TenantID:        tenantID,
		FirstName:       longName,
		LastName:        "Test",
		Email:           &longEmail,
		PasswordHash:    &passwordHash,
		IsVerified:      true,
		IsActive:        true,
		TermsAcceptedAt: now,
		LastActivityAt:  now,
	}
	if err := s.userRepo.Create(longUser); err != nil {
		log.Printf("Warning: failed to create long name user: %v", err)
	}

	// Special characters
	specialEmail := "special.chars@local.test"
	specialUser := &models.User{
		TenantID:        tenantID,
		FirstName:       "Test-User'Name",
		LastName:        "O'Brien-Schmidt",
		Email:           &specialEmail,
		PasswordHash:    &passwordHash,
		IsVerified:      true,
		IsActive:        true,
		TermsAcceptedAt: now,
		LastActivityAt:  now,
	}
	if err := s.userRepo.Create(specialUser); err != nil {
		log.Printf("Warning: failed to create special chars user: %v", err)
	}

	// Deactivated user
	deactivatedEmail := "deactivated@local.test"
	deactivatedUser := &models.User{
		TenantID:        tenantID,
		FirstName:       "Deaktiviert",
		LastName:        "Benutzer",
		Email:           &deactivatedEmail,
		PasswordHash:    &passwordHash,
		IsVerified:      true,
		IsActive:        false,
		TermsAcceptedAt: now,
		LastActivityAt:  now.AddDate(-1, 0, 0), // 1 year ago
	}
	if err := s.userRepo.Create(deactivatedUser); err != nil {
		log.Printf("Warning: failed to create deactivated user: %v", err)
	}

	return nil
}

// ResetTenant resets a specific local dev tenant to its initial state
func (s *LocalDevSeedService) ResetTenant(slug string) error {
	// Find tenant config
	var cfg *LocalDevTenantConfig
	for _, c := range LocalDevTenants {
		if c.Slug == slug {
			cfg = &c
			break
		}
	}

	if cfg == nil {
		return fmt.Errorf("unknown local dev tenant: %s", slug)
	}

	// Find tenant in database
	tenant, err := s.tenantRepo.FindBySlug(slug)
	if err != nil {
		return fmt.Errorf("failed to find tenant: %w", err)
	}
	if tenant == nil {
		return fmt.Errorf("tenant not found: %s", slug)
	}

	log.Printf("Resetting local dev tenant '%s' (ID: %d)...", slug, tenant.ID)

	// Delete all tenant data
	if err := s.deleteAllTenantData(tenant.ID); err != nil {
		return fmt.Errorf("failed to delete tenant data: %w", err)
	}

	// Re-seed data using config for domain
	adminEmail := fmt.Sprintf("admin@%s.%s", slug, s.cfg.BaseDomain)
	if err := s.seedProfile(tenant.ID, cfg.Profile, adminEmail); err != nil {
		return fmt.Errorf("failed to re-seed data: %w", err)
	}

	log.Printf("Local dev tenant '%s' reset complete", slug)
	return nil
}

// ResetAllTenants resets all local dev tenants
func (s *LocalDevSeedService) ResetAllTenants() error {
	for _, cfg := range LocalDevTenants {
		if err := s.ResetTenant(cfg.Slug); err != nil {
			log.Printf("Warning: failed to reset tenant %s: %v", cfg.Slug, err)
		}
	}
	return nil
}

// deleteAllTenantData removes all data for a tenant
func (s *LocalDevSeedService) deleteAllTenantData(tenantID int) error {
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
		query := s.db.Rebind(fmt.Sprintf("DELETE FROM %s WHERE tenant_id = ?", table))
		if _, err := s.db.Exec(query, tenantID); err != nil {
			log.Printf("Warning: failed to delete from %s: %v", table, err)
		}
	}

	return nil
}

// GetProfileFromSlug returns the profile for a given tenant slug
func GetProfileFromSlug(slug string) LocalDevProfile {
	for _, cfg := range LocalDevTenants {
		if cfg.Slug == slug {
			return cfg.Profile
		}
	}
	return ProfileEmpty
}
