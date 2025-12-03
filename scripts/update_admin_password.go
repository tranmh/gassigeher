// +build ignore

package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Generate bcrypt hash for "test123"
	password := "test123"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}

	hashStr := string(hash)
	fmt.Println("Password hash for 'test123':", hashStr)

	// Update database
	db, err := sql.Open("sqlite", "../gassigeher.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Update admin user: set password hash and is_verified = 1
	result, err := db.Exec(`
		UPDATE users
		SET password_hash = ?, is_verified = 1
		WHERE email = 'admin@tierheim-goeppingen.de'
	`, hashStr)
	if err != nil {
		log.Fatal("Failed to update user:", err)
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Updated %d user(s)\n", rowsAffected)
	fmt.Println("Admin user updated successfully!")
	fmt.Println("Email: admin@tierheim-goeppingen.de")
	fmt.Println("Password: test123")
	fmt.Println("is_verified: true")
}
