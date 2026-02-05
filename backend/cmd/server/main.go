package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/envo/backend/internal/config"
	"github.com/envo/backend/internal/database"
	"github.com/envo/backend/internal/models"
	"github.com/gin-gonic/gin"
)

func main() {
	// Parse command line flags
	migrate := flag.Bool("migrate", false, "Run database migrations")
	seed := flag.Bool("seed", false, "Seed initial data (permissions, roles, tier limits)")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}

	// Connect to database
	if err := database.Connect(cfg); err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Run migrations if requested
	if *migrate {
		log.Println("🔄 Running database migrations...")
		if err := models.AutoMigrate(database.GetDB()); err != nil {
			log.Fatalf("❌ Migration failed: %v", err)
		}
		log.Println("✅ Migrations completed successfully!")
		return
	}

	// Seed initial data if requested
	if *seed {
		if err := database.SeedInitialData(database.GetDB()); err != nil {
			log.Fatalf("❌ Seeding failed: %v", err)
		}
		return
	}

	// Set Gin mode
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize router
	router := gin.Default()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "envo-backend",
			"version": "0.1.0",
			"env":     cfg.Env,
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "pong",
			})
		})
	}

	// Start server
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("🚀 Envo Backend Server")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("📝 Environment: %s", cfg.Env)
	log.Printf("🌐 Server: http://localhost%s", addr)
	log.Printf("❤️  Health: http://localhost%s/health", addr)
	log.Printf("🔗 API v1: http://localhost%s/api/v1", addr)
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if err := router.Run(addr); err != nil {
		log.Fatalf("❌ Failed to start server: %v", err)
	}
}
