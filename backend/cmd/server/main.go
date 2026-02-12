package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/envo/backend/internal/config"
	"github.com/envo/backend/internal/database"
	"github.com/envo/backend/internal/handlers"
	"github.com/envo/backend/internal/middleware"
	"github.com/envo/backend/internal/models"
	"github.com/envo/backend/internal/services"
	"github.com/envo/backend/internal/utils"
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

	// Initialize services
	jwtManager, err := utils.NewJWTManager(
		cfg.JWTSecret,
		cfg.JWTAccessTokenExpiry,
		cfg.JWTRefreshTokenExpiry,
	)
	if err != nil {
		log.Fatalf("❌ Failed to create JWT manager: %v", err)
	}

	tierService := services.NewTierService()
	authService := services.NewAuthService(cfg, jwtManager)
	if cfg.GoogleRedirectURL != "" {
		log.Printf("🔐 Google OAuth redirect_uri (must match Google Console exactly): %s", cfg.GoogleRedirectURL)
	}
	orgService := services.NewOrgService(tierService)
	projectService := services.NewProjectService(tierService)
	envService := services.NewEnvironmentService()
	auditService := services.NewAuditService()

	// Initialize KMS service (optional - only if credentials are provided)
	var kmsService *services.KMSService
	if cfg.AWSKMSKeyID != "" {
		kmsService, err = services.NewKMSService(cfg)
		if err != nil {
			log.Printf("⚠️  Warning: Failed to initialize KMS service: %v", err)
			log.Println("⚠️  Secret encryption will not be available")
		} else {
			log.Println("✅ KMS service initialized successfully")
		}
	}

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	orgHandler := handlers.NewOrgHandler(orgService)
	projectHandler := handlers.NewProjectHandler(projectService)
	envHandler := handlers.NewEnvironmentHandler(envService, projectService)
	secretHandler := handlers.NewSecretHandler(services.NewSecretService(kmsService, tierService, auditService))
	auditHandler := handlers.NewAuditHandler(auditService)

	// Set Gin mode
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize router
	router := gin.Default()

	// CORS middleware
	router.Use(middleware.SetupCORS(cfg.FrontendURL))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "envo-backend",
			"version": "0.1.0",
			"env":     cfg.Env,
			"kms":     kmsService != nil,
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Public routes
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "pong",
			})
		})

		// Authentication routes (public)
		auth := v1.Group("/auth")
		{
			auth.GET("/google/login", authHandler.GoogleLogin)
			auth.GET("/google/redirect", authHandler.GoogleLoginRedirect)
			auth.GET("/google/callback", authHandler.GoogleCallback)
			auth.GET("/cli/google/start", authHandler.CLIGoogleStart)
			auth.POST("/cli/exchange", authHandler.CLIExchange)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/logout", authHandler.Logout)
		}

		// Protected routes (require authentication)
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(jwtManager))
		{
			// Current user
			protected.GET("/auth/me", authHandler.GetCurrentUser)

			// Organizations
			protected.GET("/orgs", orgHandler.ListOrganizations)
			protected.POST("/orgs", orgHandler.CreateOrganization)
			protected.GET("/orgs/:id", orgHandler.GetOrganization)
			protected.PATCH("/orgs/:id", middleware.RequirePermission(models.PermissionOrgManage), orgHandler.UpdateOrganization)
			protected.DELETE("/orgs/:id", middleware.RequirePermission(models.PermissionOrgManage), orgHandler.DeleteOrganization)

			// Organization members
			protected.POST("/orgs/:id/members", middleware.RequirePermission(models.PermissionMembersInvite), orgHandler.InviteMember)
			protected.PATCH("/orgs/:id/members/:memberId", middleware.RequirePermission(models.PermissionMembersManage), orgHandler.UpdateMemberRole)
			protected.DELETE("/orgs/:id/members/:memberId", middleware.RequirePermission(models.PermissionMembersManage), orgHandler.RemoveMember)

			// Projects (use :id for org to match GET /orgs/:id)
			protected.GET("/orgs/:id/projects", projectHandler.ListOrgProjects)
			protected.POST("/orgs/:id/projects", middleware.RequirePermission(models.PermissionProjectsManage), projectHandler.CreateProject)
			protected.GET("/projects/:id", projectHandler.GetProject)
			protected.PATCH("/projects/:id", middleware.RequirePermission(models.PermissionProjectsManage), projectHandler.UpdateProject)
			protected.DELETE("/projects/:id", middleware.RequirePermission(models.PermissionProjectsManage), projectHandler.DeleteProject)

			// Environments (use :id for project to match GET /projects/:id)
			protected.GET("/projects/:id/environments", envHandler.ListProjectEnvironments)
			protected.POST("/projects/:id/environments", middleware.RequirePermission(models.PermissionEnvironmentsManage), envHandler.CreateEnvironment)
			protected.PATCH("/environments/:id", middleware.RequirePermission(models.PermissionEnvironmentsManage), envHandler.UpdateEnvironment)
			protected.DELETE("/environments/:id", middleware.RequirePermission(models.PermissionEnvironmentsManage), envHandler.DeleteEnvironment)

			// Secrets (use :id for environment to match PATCH/DELETE /environments/:id)
			protected.GET("/environments/:id/secrets", middleware.RequirePermission(models.PermissionSecretsRead), secretHandler.ListSecrets)
			protected.POST("/environments/:id/secrets", middleware.RequirePermission(models.PermissionSecretsCreate), secretHandler.CreateSecret)
			protected.PATCH("/secrets/:id", middleware.RequirePermission(models.PermissionSecretsUpdate), secretHandler.UpdateSecret)
			protected.DELETE("/secrets/:id", middleware.RequirePermission(models.PermissionSecretsDelete), secretHandler.DeleteSecret)

			// Secrets export for CLI
			protected.GET("/environments/:id/secrets/export", middleware.RequirePermission(models.PermissionSecretsRead), secretHandler.ExportEnvironmentSecrets)

			// Audit logs
			protected.GET("/orgs/:id/audit-logs", middleware.RequirePermission(models.PermissionAuditView), auditHandler.ListOrgAuditLogs)
		}
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
	log.Printf("🔐 OAuth: http://localhost%s/api/v1/auth/google/login", addr)
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if err := router.Run(addr); err != nil {
		log.Fatalf("❌ Failed to start server: %v", err)
	}
}

