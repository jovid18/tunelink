package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/tunelink/api/internal/config"
	"github.com/tunelink/api/internal/handler"
	"github.com/tunelink/api/internal/repository"
	"github.com/tunelink/api/internal/service"
)

func main() {
	cfg := config.Load()

	// Initialize dependencies
	urlRepo := repository.NewURLRepository(cfg.DB)
	urlService := service.NewURLService(urlRepo, cfg.Redis)
	urlHandler := handler.NewURLHandler(urlService)

	// Setup router
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Routes
	r.GET("/health", handler.Health)
	r.POST("/api/urls", urlHandler.Create)
	r.GET("/r/:shortUrl", urlHandler.Redirect)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
