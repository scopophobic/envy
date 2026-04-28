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

	// In development, keep schema in sync automatically so local auth/setup
	// does not fail when new model fields are introduced.
	if cfg.IsDevelopment() {
		log.Println("🔄 Development mode: applying automatic database migrations...")
		if err := models.AutoMigrate(database.GetDB()); err != nil {
			log.Fatalf("❌ Auto-migration failed: %v", err)
		}
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
	var emailSender services.EmailSender = &services.LogEmailSender{}
	if smtpSender, smtpErr := services.NewSMTPEmailSender(cfg); smtpErr == nil {
		emailSender = smtpSender
		log.Println("✉️ SMTP invite email sender enabled")
	} else {
		log.Printf("⚠️  SMTP email not configured, falling back to log sender: %v", smtpErr)
	}
	orgService := services.NewOrgService(tierService, emailSender, cfg.FrontendURL, cfg.InviteTokenTTLHours)
	projectService := services.NewProjectService(tierService)
	envService := services.NewEnvironmentService()
	auditService := services.NewAuditService()
	adminService := services.NewAdminService()

	// Initialize encryption: primary (KMS or local) + always local for decrypting mixed storage
	localEncryptor := services.NewLocalEncryptionService(cfg.JWTSecret)
	var encryptor services.Encryptor
	if cfg.AWSKMSKeyID != "" {
		kmsService, kmsErr := services.NewKMSService(cfg)
		if kmsErr != nil {
			log.Printf("⚠️  Warning: Failed to initialize KMS service: %v", kmsErr)
			log.Println("⚠️  Falling back to local encryption (dev only, not for production!)")
			encryptor = localEncryptor
		} else {
			log.Println("✅ KMS service initialized successfully")
			encryptor = kmsService
		}
	} else {
		log.Println("⚠️  No AWS_KMS_KEY_ID configured, using local encryption (dev only)")
		encryptor = localEncryptor
	}

	// Billing: routes are always registered; without keys, handlers return 503 + JSON (no more 404 on /billing/*).
	var billingService *services.BillingService
	if cfg.RazorpayKeyID != "" && cfg.RazorpayKeySecret != "" {
		razorpayProvider := services.NewRazorpayProvider(
			cfg.RazorpayKeyID,
			cfg.RazorpayKeySecret,
			cfg.RazorpayWebhookSecret,
			cfg.RazorpayPlanStarter,
			cfg.RazorpayPlanTeam,
		)
		billingService = services.NewBillingService(razorpayProvider, cfg.FrontendURL)
		log.Println("💳 Razorpay billing enabled (checkout + webhooks active)")
	} else {
		log.Println("⚠️  RAZORPAY_KEY_ID / RAZORPAY_KEY_SECRET not set — billing returns 503 until configured")
	}
	billingHandler := handlers.NewBillingHandler(billingService)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService, tierService, cfg.FrontendURL)
	orgHandler := handlers.NewOrgHandler(orgService)
	projectHandler := handlers.NewProjectHandler(projectService)
	envHandler := handlers.NewEnvironmentHandler(envService, projectService, tierService)
	secretService := services.NewSecretService(encryptor, localEncryptor, tierService, auditService)
	secretHandler := handlers.NewSecretHandler(secretService)
	platformService := services.NewPlatformService(encryptor, localEncryptor, secretService)
	platformHandler := handlers.NewPlatformHandler(platformService)
	auditHandler := handlers.NewAuditHandler(auditService)
	adminHandler := handlers.NewAdminHandler(adminService)

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
			"kms":     encryptor != nil,
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
			protected.GET("/auth/tier-info", authHandler.GetTierInfo)

			// Organizations
			protected.GET("/orgs", orgHandler.ListOrganizations)
			protected.POST("/orgs", orgHandler.CreateOrganization)
			protected.GET("/orgs/:id", orgHandler.GetOrganization)
			protected.PATCH("/orgs/:id", middleware.RequireOrgPermission("id", models.PermissionOrgManage), orgHandler.UpdateOrganization)
			protected.DELETE("/orgs/:id", middleware.RequireOrgPermission("id", models.PermissionOrgManage), orgHandler.DeleteOrganization)

			// Organization members (blocked for personal workspaces)
			protected.POST("/orgs/:id/members", middleware.RejectIfPersonalWorkspace(), middleware.RequireOrgPermission("id", models.PermissionMembersInvite), orgHandler.InviteMember)
			protected.PATCH("/orgs/:id/members/:memberId", middleware.RejectIfPersonalWorkspace(), middleware.RequireOrgPermission("id", models.PermissionMembersManage), orgHandler.UpdateMemberRole)
			protected.DELETE("/orgs/:id/members/:memberId", middleware.RejectIfPersonalWorkspace(), middleware.RequireOrgPermission("id", models.PermissionMembersManage), orgHandler.RemoveMember)
			protected.GET("/orgs/:id/invites", middleware.RejectIfPersonalWorkspace(), middleware.RequireOrgPermission("id", models.PermissionMembersManage), orgHandler.ListInvitations)
			protected.POST("/orgs/:id/invites/:inviteId/resend", middleware.RejectIfPersonalWorkspace(), middleware.RequireOrgPermission("id", models.PermissionMembersInvite), orgHandler.ResendInvitation)
			protected.DELETE("/orgs/:id/invites/:inviteId", middleware.RejectIfPersonalWorkspace(), middleware.RequireOrgPermission("id", models.PermissionMembersManage), orgHandler.RevokeInvitation)
			protected.GET("/orgs/:id/roles", middleware.RequireOrgPermission("id", models.PermissionMembersManage), orgHandler.ListRoles)
			protected.POST("/orgs/:id/roles", middleware.RequireOrgPermission("id", models.PermissionMembersManage), orgHandler.CreateRole)
			protected.PATCH("/orgs/:id/roles/:roleId", middleware.RequireOrgPermission("id", models.PermissionMembersManage), orgHandler.UpdateRole)
			protected.DELETE("/orgs/:id/roles/:roleId", middleware.RequireOrgPermission("id", models.PermissionMembersManage), orgHandler.DeleteRole)

			// Projects (use :id for org to match GET /orgs/:id)
			protected.GET("/orgs/:id/projects", projectHandler.ListOrgProjects)
			protected.POST("/orgs/:id/projects", middleware.RequireOrgPermission("id", models.PermissionProjectsManage), projectHandler.CreateProject)
			protected.GET("/projects/:id", projectHandler.GetProject)
			protected.PATCH("/projects/:id", middleware.RequireProjectPermission("id", models.PermissionProjectsManage), projectHandler.UpdateProject)
			protected.DELETE("/projects/:id", middleware.RequireProjectPermission("id", models.PermissionProjectsManage), projectHandler.DeleteProject)

			// Environments (use :id for project to match GET /projects/:id)
			protected.GET("/projects/:id/environments", envHandler.ListProjectEnvironments)
			protected.POST("/projects/:id/environments", middleware.RequireProjectPermission("id", models.PermissionEnvironmentsManage), envHandler.CreateEnvironment)
			protected.GET("/environments/:id", envHandler.GetEnvironment)
			protected.PATCH("/environments/:id", middleware.RequireEnvironmentPermission("id", models.PermissionEnvironmentsManage), envHandler.UpdateEnvironment)
			protected.DELETE("/environments/:id", middleware.RequireEnvironmentPermission("id", models.PermissionEnvironmentsManage), envHandler.DeleteEnvironment)

			// Secrets (use :id for environment to match PATCH/DELETE /environments/:id)
			protected.GET("/environments/:id/secrets", middleware.RequireEnvironmentPermission("id", models.PermissionSecretsRead), secretHandler.ListSecrets)
			protected.POST("/environments/:id/secrets", middleware.RequireEnvironmentPermission("id", models.PermissionSecretsCreate), secretHandler.CreateSecret)
			protected.PATCH("/secrets/:id", middleware.RequireSecretPermission("id", models.PermissionSecretsUpdate), secretHandler.UpdateSecret)
			protected.DELETE("/secrets/:id", middleware.RequireSecretPermission("id", models.PermissionSecretsDelete), secretHandler.DeleteSecret)
			protected.DELETE("/secrets/:id/purge", middleware.RequireSecretPermission("id", models.PermissionSecretsDelete), secretHandler.PurgeSecret)

			// Secrets export for CLI
			protected.GET("/environments/:id/secrets/export", middleware.RequireEnvironmentPermission("id", models.PermissionSecretsRead), secretHandler.ExportEnvironmentSecrets)
			protected.POST("/environments/:id/sync", middleware.RequireEnvironmentPermission("id", models.PermissionSecretsRead), platformHandler.SyncEnvironment)

			// Deployment platform connections
			protected.GET("/platforms", platformHandler.ListConnections)
			protected.POST("/platforms", platformHandler.CreateConnection)
			protected.DELETE("/platforms/:id", platformHandler.DeleteConnection)

			// Audit logs
			protected.GET("/orgs/:id/audit-logs", middleware.RequireOrgPermission("id", models.PermissionAuditView), auditHandler.ListOrgAuditLogs)

			// Billing (protected)
			protected.GET("/billing/status", billingHandler.Status)
			protected.POST("/billing/checkout", billingHandler.CreateCheckoutSession)
			protected.POST("/billing/portal", billingHandler.CreatePortalSession)
			protected.POST("/billing/orders", billingHandler.CreateOrder)
			protected.POST("/billing/verify-payment", billingHandler.VerifyPayment)
			protected.POST("/invites/accept", orgHandler.AcceptInvitation)
			protected.GET("/invites/mine", orgHandler.ListMyInvitations)
			protected.POST("/invites/:inviteId/accept", orgHandler.AcceptMyInvitation)

			// Platform super-admin (v2)
			admin := protected.Group("/admin")
			admin.Use(middleware.RequireSuperAdmin())
			{
				admin.GET("/users", adminHandler.ListUsers)
				admin.PATCH("/users/:id/tier", adminHandler.UpdateUserTier)
			}
		}

		// Billing webhook (public — Razorpay sends without our JWT)
		v1.POST("/billing/webhook", billingHandler.HandleWebhook)
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
