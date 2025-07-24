package main

import (
	"log"
	"net/http"
	"os"
	"gwent-backend/internal/api"
	"gwent-backend/internal/db"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	database, err := db.NewDatabase(databaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	router := mux.NewRouter()
	router.Use(api.CORSMiddleware)

	authHandler := api.NewAuthHandler(database.DB, jwtSecret)
	gameHandler := api.NewGameHandler(database.DB)
	gameHub := api.NewGameHub(database.DB)
	
	go gameHub.Run()

	router.HandleFunc("/api/auth/register", authHandler.Register).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/auth/login", authHandler.Login).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/users/search", authHandler.GetUserByNameID).Methods("GET", "OPTIONS")

	protectedRoutes := router.PathPrefix("/api/protected").Subrouter()
	protectedRoutes.Use(api.AuthMiddleware(jwtSecret))
	
	protectedRoutes.HandleFunc("/challenges", gameHandler.CreateChallenge).Methods("POST", "OPTIONS")
	protectedRoutes.HandleFunc("/challenges", gameHandler.GetChallenges).Methods("GET", "OPTIONS")
	protectedRoutes.HandleFunc("/challenges/{id}/respond", gameHandler.RespondToChallenge).Methods("POST", "OPTIONS")
	protectedRoutes.HandleFunc("/games", gameHandler.GetActiveGames).Methods("GET", "OPTIONS")
	protectedRoutes.HandleFunc("/games/ai", gameHandler.CreateAIGame).Methods("POST", "OPTIONS")
	
	router.HandleFunc("/ws/game/{gameId}", api.HandleWebSocket(gameHub, jwtSecret))

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}