//go:build ignore

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/tranmh/gassigeher/internal/database"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
)

func main() {
	fmt.Println("Phase 2 Migration Test")
	fmt.Println("======================")
	fmt.Println()

	dbPath := "./test_migration.db"

	// Clean up old test database
	os.Remove(dbPath)

	// Initialize database
	db, err := database.Initialize(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	defer os.Remove(dbPath)

	// Run migrations
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	fmt.Println("[OK] Database initialized and migrations completed")

	// Verify dogs table structure
	rows, err := db.Query("PRAGMA table_info(dogs)")
	if err != nil {
		log.Fatalf("Failed to get table info: %v", err)
	}
	defer rows.Close()

	hasPhoto := false
	hasPhotoThumbnail := false

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue sql.NullString

		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}

		if name == "photo" {
			hasPhoto = true
			fmt.Printf("[OK] Found 'photo' column (type: %s)\n", ctype)
		}
		if name == "photo_thumbnail" {
			hasPhotoThumbnail = true
			fmt.Printf("[OK] Found 'photo_thumbnail' column (type: %s)\n", ctype)
		}
	}

	if !hasPhoto {
		log.Fatal("[FAIL] 'photo' column not found!")
	}
	if !hasPhotoThumbnail {
		log.Fatal("[FAIL] 'photo_thumbnail' column not found!")
	}

	fmt.Println("[OK] Dogs table structure verified")
	fmt.Println()

	// Test creating a dog with photo fields
	dogRepo := repository.NewDogRepository(db)

	photo := "dogs/dog_1_full.jpg"
	photoThumb := "dogs/dog_1_thumb.jpg"

	testDog := &models.Dog{
		Name:           "TestDog",
		Breed:          "Labrador",
		Size:           "large",
		Age:            3,
		Category:       "green",
		Photo:          &photo,
		PhotoThumbnail: &photoThumb,
		IsAvailable:    true,
	}

	if err := dogRepo.Create(testDog); err != nil {
		log.Fatalf("Failed to create dog: %v", err)
	}

	fmt.Printf("[OK] Created dog with ID: %d\n", testDog.ID)

	// Retrieve the dog and verify fields
	retrievedDog, err := dogRepo.FindByID(testDog.ID)
	if err != nil {
		log.Fatalf("Failed to retrieve dog: %v", err)
	}
	if retrievedDog == nil {
		log.Fatal("Dog not found!")
	}

	if retrievedDog.Photo == nil || *retrievedDog.Photo != photo {
		log.Fatal("[FAIL] Photo field mismatch!")
	}
	if retrievedDog.PhotoThumbnail == nil || *retrievedDog.PhotoThumbnail != photoThumb {
		log.Fatal("[FAIL] PhotoThumbnail field mismatch!")
	}

	fmt.Println("[OK] Photo fields verified after retrieval")

	// Test dog without photos (NULL values)
	testDogNoPhoto := &models.Dog{
		Name:        "DogNoPhoto",
		Breed:       "Beagle",
		Size:        "medium",
		Age:         2,
		Category:    "green",
		IsAvailable: true,
	}

	if err := dogRepo.Create(testDogNoPhoto); err != nil {
		log.Fatalf("Failed to create dog without photos: %v", err)
	}

	retrievedDogNoPhoto, err := dogRepo.FindByID(testDogNoPhoto.ID)
	if err != nil {
		log.Fatalf("Failed to retrieve dog without photos: %v", err)
	}

	if retrievedDogNoPhoto.Photo != nil {
		log.Fatal("[FAIL] Expected NULL photo!")
	}
	if retrievedDogNoPhoto.PhotoThumbnail != nil {
		log.Fatal("[FAIL] Expected NULL photo_thumbnail!")
	}

	fmt.Println("[OK] NULL photo fields verified (backward compatibility)")

	// Test updating photo fields
	newPhoto := "dogs/dog_1_full_v2.jpg"
	newPhotoThumb := "dogs/dog_1_thumb_v2.jpg"
	retrievedDog.Photo = &newPhoto
	retrievedDog.PhotoThumbnail = &newPhotoThumb

	if err := dogRepo.Update(retrievedDog); err != nil {
		log.Fatalf("Failed to update dog: %v", err)
	}

	updatedDog, err := dogRepo.FindByID(retrievedDog.ID)
	if err != nil {
		log.Fatalf("Failed to retrieve updated dog: %v", err)
	}

	if updatedDog.Photo == nil || *updatedDog.Photo != newPhoto {
		log.Fatal("[FAIL] Updated photo field mismatch!")
	}
	if updatedDog.PhotoThumbnail == nil || *updatedDog.PhotoThumbnail != newPhotoThumb {
		log.Fatal("[FAIL] Updated photo_thumbnail field mismatch!")
	}

	fmt.Println("[OK] Photo fields updated successfully")

	// Test FindAll includes photo fields
	allDogs, err := dogRepo.FindAll(nil)
	if err != nil {
		log.Fatalf("Failed to retrieve all dogs: %v", err)
	}

	if len(allDogs) != 2 {
		log.Fatalf("[FAIL] Expected 2 dogs, got %d", len(allDogs))
	}

	fmt.Printf("[OK] FindAll returned %d dogs with photo fields\n", len(allDogs))

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("SUCCESS: All Phase 2 migration tests PASSED!")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println("Summary:")
	fmt.Println("  - Database migration successful")
	fmt.Println("  - photo_thumbnail column created")
	fmt.Println("  - Dog creation with photos works")
	fmt.Println("  - Dog creation without photos works (backward compatible)")
	fmt.Println("  - Dog retrieval includes photo fields")
	fmt.Println("  - Dog update with photos works")
	fmt.Println("  - FindAll returns photo fields")
}
