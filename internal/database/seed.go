package database

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// TestUser holds test user credentials for display
type TestUser struct {
	FirstName string
	LastName  string
	Email     string
	Password  string
	Level     string
}

// SeedDatabase generates initial seed data for first-time installations
// Only runs if users table is empty
// Set SKIP_SEED=true to skip (useful for E2E tests that manage their own data)
// DONE
func SeedDatabase(db *sql.DB, superAdminEmail string) error {
	// 0. Check if seeding is disabled (for E2E tests)
	if os.Getenv("SKIP_SEED") == "true" {
		log.Println("SKIP_SEED=true, skipping seed data generation")
		return nil
	}

	// 1. Check if users table is empty
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check users count: %w", err)
	}

	if count > 0 {
		log.Println("Database already seeded, skipping seed data generation")
		return nil
	}

	log.Println("Empty database detected, generating seed data...")

	// 2. Validate Super Admin email
	if superAdminEmail == "" {
		return fmt.Errorf("SUPER_ADMIN_EMAIL not set in .env - cannot create Super Admin")
	}

	// 3. Generate Super Admin
	superAdminPassword := generateSecurePassword(20)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(superAdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash super admin password: %w", err)
	}

	now := time.Now()
	_, err = db.Exec(`
		INSERT INTO users (
			id, name, first_name, last_name, email, password_hash, experience_level,
			is_admin, is_super_admin, is_verified, is_active,
			terms_accepted_at, last_activity_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, 1, "Super Admin", "Super", "Admin", superAdminEmail, string(hashedPassword), "orange",
		true, true, true, true, now, now, now, now)

	if err != nil {
		return fmt.Errorf("failed to create Super Admin: %w", err)
	}

	log.Println("✓ Super Admin created (ID: 1)")

	// 4. Generate test users
	testUsers, err := generateTestUsers(db)
	if err != nil {
		return fmt.Errorf("failed to generate test users: %w", err)
	}

	// 5. Generate dogs
	err = generateDogs(db)
	if err != nil {
		return fmt.Errorf("failed to generate dogs: %w", err)
	}

	// 6. Assign colors to users (all users get green/gruen color)
	err = assignUserColors(db)
	if err != nil {
		return fmt.Errorf("failed to assign user colors: %w", err)
	}

	// 7. Generate bookings
	err = generateBookings(db)
	if err != nil {
		return fmt.Errorf("failed to generate bookings: %w", err)
	}

	// 8. Initialize default settings (if not exists)
	err = initializeSystemSettings(db)
	if err != nil {
		return fmt.Errorf("failed to initialize system settings: %w", err)
	}

	// 9. Write credentials to file
	err = writeCredentialsFile(superAdminEmail, superAdminPassword)
	if err != nil {
		log.Printf("Warning: Failed to write credentials file: %v", err)
	}

	// 10. Print setup complete message
	printSetupComplete(superAdminEmail, superAdminPassword, testUsers)

	log.Println("✓ Seed data generation completed successfully")
	return nil
}

// generateSecurePassword generates a cryptographically secure random password
// Uses crypto/rand for unpredictable random bytes
func generateSecurePassword(length int) string {
	// Character sets for password generation
	lowercase := "abcdefghijklmnopqrstuvwxyz"
	uppercase := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numbers := "0123456789"
	special := "!@#$%^&*"
	allChars := lowercase + uppercase + numbers + special

	password := make([]byte, length)

	// Helper function for cryptographically secure random index
	secureRandomIndex := func(max int) int {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
		if err != nil {
			// Fallback should never happen with crypto/rand
			panic("crypto/rand failed: " + err.Error())
		}
		return int(n.Int64())
	}

	// Ensure at least one of each type
	password[0] = lowercase[secureRandomIndex(len(lowercase))]
	password[1] = uppercase[secureRandomIndex(len(uppercase))]
	password[2] = numbers[secureRandomIndex(len(numbers))]
	password[3] = special[secureRandomIndex(len(special))]

	// Fill rest randomly
	for i := 4; i < length; i++ {
		password[i] = allChars[secureRandomIndex(len(allChars))]
	}

	// Shuffle using Fisher-Yates with crypto/rand
	for i := len(password) - 1; i > 0; i-- {
		j := secureRandomIndex(i + 1)
		password[i], password[j] = password[j], password[i]
	}

	return string(password)
}

// generateTestUsers creates 3 test users with different experience levels
// DONE
func generateTestUsers(db *sql.DB) ([]TestUser, error) {
	users := []TestUser{
		{FirstName: "Test", LastName: "Walker (Green)", Email: "green-walker@test.com", Level: "green"},
		{FirstName: "Test", LastName: "Walker (Blue)", Email: "blue-walker@test.com", Level: "blue"},
		{FirstName: "Test", LastName: "Walker (Orange)", Email: "orange-walker@test.com", Level: "orange"},
	}

	now := time.Now()
	for i := range users {
		users[i].Password = generateSecurePassword(12)
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(users[i].Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash test user password: %w", err)
		}

		fullName := users[i].FirstName + " " + users[i].LastName
		_, err = db.Exec(`
			INSERT INTO users (name, first_name, last_name, email, password_hash, experience_level,
				is_admin, is_super_admin, is_verified, is_active,
				terms_accepted_at, last_activity_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, fullName, users[i].FirstName, users[i].LastName, users[i].Email, string(hashedPassword), users[i].Level,
			false, false, true, true, now, now, now, now)

		if err != nil {
			return nil, fmt.Errorf("failed to create test user %s: %w", users[i].Email, err)
		}
	}

	log.Printf("✓ Created %d test users", len(users))
	return users, nil
}

// generateDogs creates 5 sample dogs with different colors and care info
// Color IDs: 1=gruen, 2=gelb, 3=orange, 4=hellblau, 5=dunkelblau
func generateDogs(db *sql.DB) error {
	dogs := []struct {
		Name                string
		Category            string // Legacy field (kept for CHECK constraint)
		ColorID             int    // New color system
		Breed               string
		Size                string
		Age                 int
		SpecialNeeds        *string
		PickupLocation      *string
		WalkRoute           *string
		WalkDuration        *int
		SpecialInstructions *string
		DefaultMorningTime  *string
		DefaultEveningTime  *string
	}{
		{
			Name:                "Bella",
			Category:            "green",
			ColorID:             1, // gruen
			Breed:               "Labrador Retriever",
			Size:                "large",
			Age:                 3,
			SpecialNeeds:        strPtr("Keine besonderen Bedürfnisse"),
			PickupLocation:      strPtr("Zwinger 1, Gebäude A"),
			WalkRoute:           strPtr("Waldweg hinter dem Tierheim, geradeaus bis zur Bank, dann links zurück"),
			WalkDuration:        intPtr(45),
			SpecialInstructions: strPtr("Bella ist sehr freundlich und verträgt sich gut mit anderen Hunden. Liebt es, Stöckchen zu holen!"),
			DefaultMorningTime:  strPtr("09:00"),
			DefaultEveningTime:  strPtr("17:00"),
		},
		{
			Name:                "Max",
			Category:            "green",
			ColorID:             2, // gelb
			Breed:               "Golden Retriever",
			Size:                "large",
			Age:                 5,
			SpecialNeeds:        strPtr("Leichte Arthrose - bitte keine langen Spaziergänge"),
			PickupLocation:      strPtr("Zwinger 3, Gebäude A"),
			WalkRoute:           strPtr("Kurze Runde um den Teich, ebener Untergrund bevorzugt"),
			WalkDuration:        intPtr(30),
			SpecialInstructions: strPtr("Max braucht häufige Pausen. Bei Anzeichen von Müdigkeit bitte umkehren. Nach dem Spaziergang Leckerli als Belohnung."),
			DefaultMorningTime:  strPtr("10:00"),
			DefaultEveningTime:  strPtr("16:00"),
		},
		{
			Name:                "Luna",
			Category:            "green",
			ColorID:             3, // orange
			Breed:               "Deutscher Schäferhund",
			Size:                "large",
			Age:                 4,
			SpecialNeeds:        strPtr("Reaktiv gegenüber anderen Hunden - Abstand halten!"),
			PickupLocation:      strPtr("Zwinger 7, Gebäude B (separater Eingang)"),
			WalkRoute:           strPtr("Feldweg Richtung Süden, weg von den Hauptwegen. Karte am Zwinger."),
			WalkDuration:        intPtr(60),
			SpecialInstructions: strPtr("WICHTIG: Mindestens 10m Abstand zu anderen Hunden. Bei Begegnung: Luna ablenken mit Leckerli. Niemals an der kurzen Leine führen bei Hundebegegnungen."),
			DefaultMorningTime:  strPtr("08:00"),
			DefaultEveningTime:  strPtr("18:00"),
		},
		{
			Name:                "Charlie",
			Category:            "green",
			ColorID:             4, // hellblau
			Breed:               "Border Collie",
			Size:                "medium",
			Age:                 2,
			SpecialNeeds:        strPtr("Sehr energiegeladen - braucht viel Beschäftigung"),
			PickupLocation:      strPtr("Zwinger 2, Gebäude A"),
			WalkRoute:           strPtr("Große Runde durch den Wald, gerne mit Apportier-Spielen"),
			WalkDuration:        intPtr(60),
			SpecialInstructions: strPtr("Charlie liebt Kopfarbeit! Bitte Ball oder Frisbee mitnehmen (liegt am Zwinger). Kommandos: Sitz, Platz, Bleib funktionieren gut."),
			DefaultMorningTime:  strPtr("08:30"),
			DefaultEveningTime:  strPtr("17:30"),
		},
		{
			Name:                "Rocky",
			Category:            "green",
			ColorID:             5, // dunkelblau
			Breed:               "Belgischer Malinois",
			Size:                "large",
			Age:                 6,
			SpecialNeeds:        strPtr("Nur für erfahrene Hundeführer - stark und eigenwillig"),
			PickupLocation:      strPtr("Zwinger 10, Gebäude C (Schlüssel beim Pfleger holen)"),
			WalkRoute:           strPtr("Trainingsgelände hinter Gebäude C, dann Waldweg Nord"),
			WalkDuration:        intPtr(45),
			SpecialInstructions: strPtr("Rocky braucht klare Führung. Immer Leckerlis dabei haben. Bei Unsicherheit: Spaziergang abbrechen und zurückkehren. Notfall-Nummer Pfleger: Am Zwinger ausgehängt."),
			DefaultMorningTime:  strPtr("07:30"),
			DefaultEveningTime:  strPtr("16:30"),
		},
	}

	now := time.Now()
	for _, dog := range dogs {
		_, err := db.Exec(`
			INSERT INTO dogs (name, category, color_id, breed, size, age,
				special_needs, pickup_location, walk_route, walk_duration,
				special_instructions, default_morning_time, default_evening_time,
				is_available, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, dog.Name, dog.Category, dog.ColorID, dog.Breed, dog.Size, dog.Age,
			dog.SpecialNeeds, dog.PickupLocation, dog.WalkRoute, dog.WalkDuration,
			dog.SpecialInstructions, dog.DefaultMorningTime, dog.DefaultEveningTime,
			true, now, now)
		if err != nil {
			return fmt.Errorf("failed to create dog %s: %w", dog.Name, err)
		}
	}

	log.Printf("✓ Created %d dogs with care info", len(dogs))
	return nil
}

// Helper functions for creating pointers to values
func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

// assignUserColors assigns colors to users based on their experience_level
// Color IDs: 1=gruen, 2=gelb, 3=orange, 4=hellblau, 5=dunkelblau, 6=helllila, 7=dunkellila
// - green users: only gruen (1)
// - orange users: gruen (1), gelb (2), orange (3)
// - blue users: all colors (1-5)
func assignUserColors(db *sql.DB) error {
	// Define which colors each experience level gets
	colorsByLevel := map[string][]int{
		"green":  {1},                // only gruen
		"orange": {1, 2, 3},          // gruen, gelb, orange
		"blue":   {1, 2, 3, 4, 5},    // all main colors
	}

	// Get all users with their experience levels
	rows, err := db.Query("SELECT id, experience_level FROM users")
	if err != nil {
		return fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	type userInfo struct {
		id    int
		level string
	}
	var users []userInfo

	for rows.Next() {
		var u userInfo
		if err := rows.Scan(&u.id, &u.level); err != nil {
			return fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, u)
	}

	totalAssignments := 0
	for _, user := range users {
		colors, ok := colorsByLevel[user.level]
		if !ok {
			// Default to green if unknown level
			colors = colorsByLevel["green"]
		}

		for _, colorID := range colors {
			_, err := db.Exec(`
				INSERT INTO user_colors (user_id, color_id)
				VALUES (?, ?)
			`, user.id, colorID)
			if err != nil {
				return fmt.Errorf("failed to assign color %d to user %d: %w", colorID, user.id, err)
			}
			totalAssignments++
		}
	}

	log.Printf("✓ Assigned %d colors to %d users based on experience levels", totalAssignments, len(users))
	return nil
}

// generateBookings creates 3 sample bookings (past, present, future)
// DONE
func generateBookings(db *sql.DB) error {
	today := time.Now()
	yesterday := today.AddDate(0, 0, -1)
	tomorrow := today.AddDate(0, 0, 1)

	bookings := []struct {
		UserID int
		DogID  int
		Date   time.Time
		Time   string
		Status string
	}{
		{2, 1, yesterday, "09:00", "completed"},
		{3, 2, today, "14:00", "scheduled"},
		{4, 3, tomorrow, "10:30", "scheduled"},
	}

	now := time.Now()
	for _, booking := range bookings {
		_, err := db.Exec(`
			INSERT INTO bookings (user_id, dog_id, date, scheduled_time,
				status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, booking.UserID, booking.DogID,
			booking.Date.Format("2006-01-02"), booking.Time,
			booking.Status, now, now)
		if err != nil {
			return fmt.Errorf("failed to create booking: %w", err)
		}
	}

	log.Printf("✓ Created %d bookings", len(bookings))
	return nil
}

// initializeSystemSettings creates default system settings if not exists
// DONE
func initializeSystemSettings(db *sql.DB) error {
	settings := []struct {
		Key   string
		Value string
	}{
		{"booking_advance_days", "14"},
		{"cancellation_notice_hours", "12"},
		{"auto_deactivation_days", "365"},
	}

	now := time.Now()
	for _, setting := range settings {
		// Check if setting exists
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM system_settings WHERE key = ?", setting.Key).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check setting %s: %w", setting.Key, err)
		}

		if count == 0 {
			_, err = db.Exec(`
				INSERT INTO system_settings (key, value, updated_at)
				VALUES (?, ?, ?)
			`, setting.Key, setting.Value, now)
			if err != nil {
				return fmt.Errorf("failed to create setting %s: %w", setting.Key, err)
			}
		}
	}

	log.Printf("✓ Initialized system settings")
	return nil
}

// writeCredentialsFile writes Super Admin credentials to a file
// DONE
func writeCredentialsFile(email, password string) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	content := fmt.Sprintf(`=============================================================
GASSIGEHER - SUPER ADMIN CREDENTIALS
=============================================================

EMAIL: %s
PASSWORD: %s

CREATED: %s
LAST UPDATED: %s

=============================================================
HOW TO CHANGE PASSWORD:
=============================================================

1. Edit the PASSWORD line above with your new password
2. Save this file
3. Restart the Gassigeher server
4. Server will hash and save the new password
5. This file will be updated with confirmation

IMPORTANT:
- Keep this file secure (never commit to git)
- This is the ONLY way to change Super Admin password
- Super Admin email cannot be changed (defined in .env)

=============================================================
`, email, password, now, now)

	err := os.WriteFile("SUPER_ADMIN_CREDENTIALS.txt", []byte(content), 0600)
	if err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	log.Println("✓ Credentials file written: SUPER_ADMIN_CREDENTIALS.txt")
	return nil
}

// printSetupComplete prints the setup completion message to console
// DONE
func printSetupComplete(superAdminEmail, superAdminPassword string, testUsers []TestUser) {
	fmt.Println()
	fmt.Println("=============================================================")
	fmt.Println("  GASSIGEHER - INSTALLATION COMPLETE")
	fmt.Println("=============================================================")
	fmt.Println()
	fmt.Println("SUPER ADMIN CREDENTIALS (SAVE THESE!):")
	fmt.Printf("  Email:    %s\n", superAdminEmail)
	fmt.Printf("  Password: %s\n", superAdminPassword)
	fmt.Println()
	fmt.Println("TEST USER CREDENTIALS:")
	for i, user := range testUsers {
		fmt.Printf("  %d. %s %s / %s / %s\n", i+1, user.FirstName, user.LastName, user.Email, user.Password)
	}
	fmt.Println()
	fmt.Println("IMPORTANT:")
	fmt.Println("- Super Admin password saved to: SUPER_ADMIN_CREDENTIALS.txt")
	fmt.Println("- Change Super Admin password: Edit file and restart server")
	fmt.Println("- Test users can be deleted after setup")
	fmt.Println()
	fmt.Println("=============================================================")
	fmt.Println()
}

// DONE
