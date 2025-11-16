package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/tranm/gassigeher/internal/config"
	"github.com/tranm/gassigeher/internal/database"
	"github.com/tranm/gassigeher/internal/handlers"
	"github.com/tranm/gassigeher/internal/middleware"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.Initialize(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize router
	router := mux.NewRouter()

	// Apply global middleware
	router.Use(middleware.LoggingMiddleware)
	router.Use(middleware.CORSMiddleware)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db, cfg)
	userHandler := handlers.NewUserHandler(db, cfg)
	dogHandler := handlers.NewDogHandler(db, cfg)

	// Public routes
	router.HandleFunc("/api/auth/register", authHandler.Register).Methods("POST")
	router.HandleFunc("/api/auth/verify-email", authHandler.VerifyEmail).Methods("POST")
	router.HandleFunc("/api/auth/login", authHandler.Login).Methods("POST")
	router.HandleFunc("/api/auth/forgot-password", authHandler.ForgotPassword).Methods("POST")
	router.HandleFunc("/api/auth/reset-password", authHandler.ResetPassword).Methods("POST")

	// Protected routes (authenticated users)
	protected := router.PathPrefix("/api").Subrouter()
	protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))

	// Auth
	protected.HandleFunc("/auth/change-password", authHandler.ChangePassword).Methods("PUT")

	// Users
	protected.HandleFunc("/users/me", userHandler.GetMe).Methods("GET")
	protected.HandleFunc("/users/me", userHandler.UpdateMe).Methods("PUT")
	protected.HandleFunc("/users/me/photo", userHandler.UploadPhoto).Methods("POST")

	// Dogs (read-only for authenticated users)
	protected.HandleFunc("/dogs", dogHandler.ListDogs).Methods("GET")
	protected.HandleFunc("/dogs/breeds", dogHandler.GetBreeds).Methods("GET")
	protected.HandleFunc("/dogs/{id}", dogHandler.GetDog).Methods("GET")

	// Admin-only routes
	admin := protected.PathPrefix("").Subrouter()
	admin.Use(middleware.RequireAdmin)

	// Dog management (admin only)
	admin.HandleFunc("/dogs", dogHandler.CreateDog).Methods("POST")
	admin.HandleFunc("/dogs/{id}", dogHandler.UpdateDog).Methods("PUT")
	admin.HandleFunc("/dogs/{id}", dogHandler.DeleteDog).Methods("DELETE")
	admin.HandleFunc("/dogs/{id}/photo", dogHandler.UploadDogPhoto).Methods("POST")
	admin.HandleFunc("/dogs/{id}/availability", dogHandler.ToggleAvailability).Methods("PUT")

	// Static files
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./frontend")))

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s...", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
