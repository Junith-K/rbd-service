package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/yourusername/rbd-service/internal/config"
	"github.com/yourusername/rbd-service/internal/handlers"
	"github.com/yourusername/rbd-service/internal/middleware"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Initialize Firebase
	if err := config.InitFirebase(); err != nil {
		log.Fatalf("Failed to initialize Firebase: %v", err)
	}
	defer config.CloseFirebase()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize Gin router
	router := gin.Default()

	// Apply middleware
	router.Use(middleware.CORS())

	// Initialize handlers
	authHandler := handlers.NewAuthHandler()
	friendHandler := handlers.NewFriendHandler()
	notificationHandler := handlers.NewNotificationHandler()
	historyHandler := handlers.NewHistoryHandler()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "Return By Death API is running",
		})
	})

	// API routes group
	api := router.Group("/api")
	{
		// Auth routes (public)
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			
			// Protected routes
			authProtected := auth.Group("")
			authProtected.Use(middleware.AuthMiddleware())
			{
				authProtected.POST("/update-fcm-token", authHandler.UpdateFCMToken)
				authProtected.POST("/refresh-token", authHandler.RefreshToken)
			}
		}

		// Friends routes (protected)
		friends := api.Group("/friends")
		friends.Use(middleware.AuthMiddleware())
		{
			friends.GET("", friendHandler.GetFriends)
			friends.GET("/pending", friendHandler.GetPendingRequests)
			friends.POST("/search", friendHandler.SearchUsers)
			friends.POST("/request", friendHandler.SendFriendRequest)
			friends.POST("/accept", friendHandler.AcceptFriendRequest)
			friends.POST("/reject", friendHandler.RejectFriendRequest)
			friends.DELETE("/:friendUserId", friendHandler.RemoveFriend)
			friends.POST("/mute", friendHandler.MuteFriend)
			friends.POST("/mute-all", friendHandler.MuteAll)
			friends.POST("/cooldown", friendHandler.UpdateCooldown)
		}

		// Notifications routes (protected)
		notifications := api.Group("/notifications")
		notifications.Use(middleware.AuthMiddleware())
		{
			notifications.POST("/trigger", notificationHandler.TriggerNotification)
			notifications.GET("/cooldown/:friendUserId", notificationHandler.CheckCooldown)
		}

		// History routes (protected)
		history := api.Group("/history")
		history.Use(middleware.AuthMiddleware())
		{
			history.GET("/:friendUserId", historyHandler.GetHistory)
		}
	}

	// Start server
	log.Printf("🚀 Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
