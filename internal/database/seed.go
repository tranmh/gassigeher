package database

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// TestUser holds test user credentials for display
type TestUser struct {
	Name     string
	Email    string
	Password string
	Level    string
}

// SeedDatabase generates initial seed data for first-time installations
// Only runs if users table is empty
// DONE
func SeedDatabase(db *sql.DB, superAdminEmail string) error {
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
			id, name, email, password_hash, experience_level,
			is_admin, is_super_admin, is_verified, is_active,
			terms_accepted_at, last_activity_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, 1, "Super Admin", superAdminEmail, string(hashedPassword), "orange",
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

	// 6. Generate bookings
	err = generateBookings(db)
	if err != nil {
		return fmt.Errorf("failed to generate bookings: %w", err)
	}

	// 7. Initialize default settings (if not exists)
	err = initializeSystemSettings(db)
	if err != nil {
		return fmt.Errorf("failed to initialize system settings: %w", err)
	}

	// 8. Write credentials to file
	err = writeCredentialsFile(superAdminEmail, superAdminPassword)
	if err != nil {
		log.Printf("Warning: Failed to write credentials file: %v", err)
	}

	// 9. Print setup complete message
	printSetupComplete(superAdminEmail, superAdminPassword, testUsers)

	log.Println("✓ Seed data generation completed successfully")
	return nil
}

// generateSecurePassword generates a secure random password
// DONE
func generateSecurePassword(length int) string {
	// Character sets for password generation
	lowercase := "abcdefghijklmnopqrstuvwxyz"
	uppercase := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numbers := "0123456789"
	special := "!@#$%^&*"
	allChars := lowercase + uppercase + numbers + special

	rand.Seed(time.Now().UnixNano())

	password := make([]byte, length)
	// Ensure at least one of each type
	password[0] = lowercase[rand.Intn(len(lowercase))]
	password[1] = uppercase[rand.Intn(len(uppercase))]
	password[2] = numbers[rand.Intn(len(numbers))]
	password[3] = special[rand.Intn(len(special))]

	// Fill rest randomly
	for i := 4; i < length; i++ {
		password[i] = allChars[rand.Intn(len(allChars))]
	}

	// Shuffle
	rand.Shuffle(len(password), func(i, j int) {
		password[i], password[j] = password[j], password[i]
	})

	return string(password)
}

// generateTestUsers creates 3 test users with different experience levels
// DONE
func generateTestUsers(db *sql.DB) ([]TestUser, error) {
	users := []TestUser{
		{Name: "Test Walker (Green)", Email: "green-walker@test.com", Level: "green"},
		{Name: "Test Walker (Blue)", Email: "blue-walker@test.com", Level: "blue"},
		{Name: "Test Walker (Orange)", Email: "orange-walker@test.com", Level: "orange"},
	}

	now := time.Now()
	for i := range users {
		users[i].Password = generateSecurePassword(12)
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(users[i].Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash test user password: %w", err)
		}

		_, err = db.Exec(`
			INSERT INTO users (name, email, password_hash, experience_level,
				is_admin, is_super_admin, is_verified, is_active,
				terms_accepted_at, last_activity_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, users[i].Name, users[i].Email, string(hashedPassword), users[i].Level,
			false, false, true, true, now, now, now, now)

		if err != nil {
			return nil, fmt.Errorf("failed to create test user %s: %w", users[i].Email, err)
		}
	}

	log.Printf("✓ Created %d test users", len(users))
	return users, nil
}

// generateDogs creates 5 sample dogs with different categories
// DONE
func generateDogs(db *sql.DB) error {
	dogs := []struct {
		Name     string
		Category string
		Breed    string
		Size     string
		Age      int
	}{
		{"Bella", "green", "Labrador Retriever", "large", 3},
		{"Max", "green", "Golden Retriever", "large", 5},
		{"Luna", "blue", "Deutscher Schäferhund", "large", 4},
		{"Charlie", "blue", "Border Collie", "medium", 2},
		{"Rocky", "orange", "Belgischer Malinois", "large", 6},
	}

	now := time.Now()
	for _, dog := range dogs {
		_, err := db.Exec(`
			INSERT INTO dogs (name, category, breed, size, age,
				special_needs, is_available, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, dog.Name, dog.Category, dog.Breed, dog.Size, dog.Age,
			"Keine besonderen Bedürfnisse", true, now, now)
		if err != nil {
			return fmt.Errorf("failed to create dog %s: %w", dog.Name, err)
		}
	}

	log.Printf("✓ Created %d dogs", len(dogs))
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
		fmt.Printf("  %d. %s / %s\n", i+1, user.Email, user.Password)
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
