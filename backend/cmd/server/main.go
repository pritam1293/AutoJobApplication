package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/jobhaunt/backend/internal/api"
	"github.com/jobhaunt/backend/internal/db"
)

func main() {
	godotenv.Load()

	dbPath := getEnv("DATABASE_PATH", "jobhaunt.db")
	serverPort := getEnv("SERVER_PORT", "8080")
	googleAIKey := getEnv("GOOGLE_API_KEY", "")
	linkedInEmail := getEnv("LINKEDIN_EMAIL", "")
	linkedInPass := getEnv("LINKEDIN_PASSWORD", "")
	resumeDir := getEnv("RESUME_DIR", "./uploads")
	mode := getEnv("GIN_MODE", "release")

	gin.SetMode(mode)

	db.Init(dbPath)

	handler := api.NewHandler(googleAIKey, linkedInEmail, linkedInPass, resumeDir)

	router := gin.Default()

	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	handler.RegisterRoutes(router)

	log.Printf("JobHaunt server starting on port %s", serverPort)
	if err := router.Run(":" + serverPort); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
