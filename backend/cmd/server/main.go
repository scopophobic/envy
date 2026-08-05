package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

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
	if err := models.HashLegacyRefreshTokens(database.GetDB()); err != nil {
		log.Fatalf("❌ Failed to secure legacy refresh tokens: %v", err)
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

	tierService := services.NewTierService(cfg.TierCacheTTL)
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
		if kmsErr == nil {
			checkCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			kmsErr = kmsService.TestConnection(checkCtx)
			cancel()
		}
		if kmsErr != nil {
			if cfg.IsProduction() && !cfg.AllowLocalEncryptionInProduction {
				log.Fatalf("❌ Failed to initialize required production KMS service: %v", kmsErr)
			}
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
	authHandler := handlers.NewAuthHandler(authService, tierService, cfg.FrontendURL, cfg.IsProduction())
	orgHandler := handlers.NewOrgHandler(orgService)
	projectHandler := handlers.NewProjectHandler(projectService)
	envHandler := handlers.NewEnvironmentHandler(envService, projectService, tierService)
	secretService := services.NewSecretService(encryptor, localEncryptor, tierService, auditService, cfg.SecretDecryptConcurrency)
	secretHandler := handlers.NewSecretHandler(secretService)
	agentService := services.NewAgentService(auditService, cfg.AgentUsageWriteInterval)
	agentHandler := handlers.NewAgentHandler(agentService, secretService, auditService)
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
	if err := router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		log.Fatalf("❌ Invalid TRUSTED_PROXIES configuration: %v", err)
	}

	// CORS middleware
	router.Use(middleware.RequestID())
	router.Use(middleware.SetupCORS(cfg.FrontendURL))
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.NoStoreAPI())
	router.Use(middleware.RequestBodyLimit(cfg.MaxRequestBodyBytes))

	var authRateLimiter, secretExportRateLimiter, platformSyncRateLimiter, agentResolveRateLimiter *middleware.RateLimiter
	if cfg.RateLimitEnabled {
		authRateLimiter = middleware.NewRateLimiter(cfg.AuthRateLimitPerMinute, time.Minute)
		secretExportRateLimiter = middleware.NewRateLimiter(cfg.SecretExportRateLimitPerMinute, time.Minute)
		platformSyncRateLimiter = middleware.NewRateLimiter(cfg.PlatformSyncRateLimitPerMinute, time.Minute)
		agentResolveRateLimiter = middleware.NewRateLimiter(cfg.AgentResolveRateLimitPerMinute, time.Minute)
		log.Printf("🛡️ Rate limits enabled: auth=%d/min, exports=%d/min, sync=%d/min, agent-resolve=%d/min", cfg.AuthRateLimitPerMinute, cfg.SecretExportRateLimitPerMinute, cfg.PlatformSyncRateLimitPerMinute, cfg.AgentResolveRateLimitPerMinute)
	}

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "envo-backend",
			"version": "0.1.0",
			"env":     cfg.Env,
			"kms":     encryptor.KeyID() != "local",
		})
	})

	// Readiness includes the critical database dependency. Keep it separate
	// from liveness so orchestrators can remove an instance from service without
	// restarting a healthy process during a temporary database outage.
	router.GET("/ready", func(c *gin.Context) {
		sqlDB, dbErr := database.GetDB().DB()
		if dbErr == nil {
			readyCtx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			dbErr = sqlDB.PingContext(readyCtx)
			cancel()
		}
		if dbErr != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "database": "unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready", "database": "ok"})
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
		if authRateLimiter != nil {
			auth.Use(authRateLimiter.Middleware(middleware.ClientIPRateLimitKey))
		}
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

			// Organization-owned non-human identities. Agent credentials are
			// created here by humans, but cannot authenticate to these routes.
			protected.GET("/orgs/:id/agents", middleware.RequireOrgPermission("id", models.PermissionAgentsManage), agentHandler.List)
			protected.POST("/orgs/:id/agents", middleware.RequireOrgPermission("id", models.PermissionAgentsManage), agentHandler.Create)
			protected.PATCH("/orgs/:id/agents/:agentId", middleware.RequireOrgPermission("id", models.PermissionAgentsManage), agentHandler.Update)
			protected.GET("/orgs/:id/agents/:agentId/credentials", middleware.RequireOrgPermission("id", models.PermissionAgentsManage), agentHandler.ListCredentials)
			protected.POST("/orgs/:id/agents/:agentId/credentials", middleware.RequireOrgPermission("id", models.PermissionAgentsManage), agentHandler.CreateCredential)
			protected.DELETE("/orgs/:id/agents/:agentId/credentials/:credentialId", middleware.RequireOrgPermission("id", models.PermissionAgentsManage), agentHandler.RevokeCredential)
			protected.GET("/orgs/:id/agents/:agentId/grants", middleware.RequireOrgPermission("id", models.PermissionAgentsManage), agentHandler.ListGrants)
			protected.POST("/orgs/:id/agents/:agentId/grants", middleware.RequireOrgPermission("id", models.PermissionAgentsManage), agentHandler.CreateGrant)
			protected.DELETE("/orgs/:id/agents/:agentId/grants/:grantId", middleware.RequireOrgPermission("id", models.PermissionAgentsManage), agentHandler.RevokeGrant)

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
			exportHandlers := []gin.HandlerFunc{middleware.RequireEnvironmentPermission("id", models.PermissionSecretsRead), secretHandler.ExportEnvironmentSecrets}
			if secretExportRateLimiter != nil {
				exportHandlers = append([]gin.HandlerFunc{secretExportRateLimiter.Middleware(middleware.AuthenticatedRateLimitKey)}, exportHandlers...)
			}
			protected.GET("/environments/:id/secrets/export", exportHandlers...)

			syncHandlers := []gin.HandlerFunc{middleware.RequireEnvironmentPermission("id", models.PermissionSecretsRead), platformHandler.SyncEnvironment}
			if platformSyncRateLimiter != nil {
				syncHandlers = append([]gin.HandlerFunc{platformSyncRateLimiter.Middleware(middleware.AuthenticatedRateLimitKey)}, syncHandlers...)
			}
			protected.POST("/environments/:id/sync", syncHandlers...)

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

		// Agent-only API. This authentication domain is intentionally isolated
		// from human JWT routes.
		agentAPI := v1.Group("/agent")
		agentAPI.Use(middleware.AgentAuthMiddleware(agentService))
		if agentResolveRateLimiter != nil {
			agentAPI.Use(agentResolveRateLimiter.Middleware(middleware.AgentRateLimitKey))
		}
		{
			agentAPI.GET("/me", agentHandler.Me)
			agentAPI.POST("/secrets/resolve", agentHandler.ResolveSecrets)
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
	log.Printf("✅ Readiness: http://localhost%s/ready", addr)
	log.Printf("🔗 API v1: http://localhost%s/api/v1", addr)
	log.Printf("🔐 OAuth: http://localhost%s/api/v1/auth/google/login", addr)
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	server := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: cfg.HTTPReadHeaderTimeout,
		ReadTimeout:       cfg.HTTPReadTimeout,
		WriteTimeout:      cfg.HTTPWriteTimeout,
		IdleTimeout:       cfg.HTTPIdleTimeout,
		MaxHeaderBytes:    cfg.HTTPMaxHeaderBytes,
	}

	shutdownSignal, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("❌ Failed to start server: %v", err)
		}
	case <-shutdownSignal.Done():
		log.Println("🛑 Shutdown signal received; draining active requests...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTPShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("❌ Graceful shutdown timed out: %v", err)
			_ = server.Close()
		} else {
			log.Println("✅ Server stopped gracefully")
		}
	}
}
