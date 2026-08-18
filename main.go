package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"apisql/config"
	"apisql/database"
	"apisql/handlers"
	"apisql/middleware"
	"apisql/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.Load()
	authCfg := &handlers.Config{
		JWTSecret:        cfg.JWTSecret,
		JWTAccessExpiry:  cfg.JWTAccessExpiry,
		JWTRefreshExpiry: cfg.JWTRefreshExpiry,
	}
	middlewareCfg := &middleware.Config{
		JWTSecret:        cfg.JWTSecret,
		JWTAccessExpiry:  cfg.JWTAccessExpiry,
		JWTRefreshExpiry: cfg.JWTRefreshExpiry,
	}

	// Set Gin mode
	gin.SetMode(cfg.GinMode)

	// Try to connect to PostgreSQL, but do not block server startup.
	if err := database.Connect(cfg.DSN()); err != nil {
		log.Printf("⚠️  Database connection failed at startup: %v", err)
		log.Printf("⚠️  API will start, but DB-backed endpoints may fail until DB is available")
	}
	defer database.Close()

	// Create Gin engine
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// Apply global middleware
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	// Create rate limiters
	authLimiter := middleware.NewRateLimiter(cfg.RateLimitAuth)
	writeLimiter := middleware.NewRateLimiter(cfg.RateLimitWrite)
	readLimiter := middleware.NewRateLimiter(cfg.RateLimitRead)

	// --- Health check ---
	r.GET("/", func(c *gin.Context) {
		utils.Success(c, http.StatusOK, "SpeedMap API v1", gin.H{
			"version": "1.0.0",
			"status":  "running",
		})
	})

	r.GET("/health", func(c *gin.Context) {
		if err := database.Health(); err != nil {
			utils.Error(c, http.StatusServiceUnavailable, "Database unhealthy")
			return
		}
		utils.Success(c, http.StatusOK, "OK", gin.H{"database": "connected"})
	})

	// --- API v1 Routes ---
	v1 := r.Group("/api/v1")

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authCfg)
	speedtestHandler := handlers.NewSpeedtestHandler()
	reportHandler := handlers.NewReportHandler()
	providerHandler := handlers.NewProviderHandler()
	zoneHandler := handlers.NewZoneHandler()
	mapHandler := handlers.NewMapHandler()
	runnerHandler := handlers.NewSpeedtestRunnerHandler()

	// --- Auth routes (rate limited: strict) ---
	auth := v1.Group("/auth")
	auth.Use(authLimiter.Middleware())
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.Refresh)
		auth.GET("/me", middleware.AuthRequired(middlewareCfg), authHandler.Me)
	}

	// --- Speedtest routes ---
	speedtests := v1.Group("/speedtests")
	{
		speedtests.POST("", writeLimiter.Middleware(), speedtestHandler.Create)    // Anonymous
		speedtests.GET("", readLimiter.Middleware(), speedtestHandler.List)        // Public
		speedtests.GET("/:id", readLimiter.Middleware(), speedtestHandler.GetByID) // Public
	}

	// --- Report routes ---
	reports := v1.Group("/reports")
	{
		reports.POST("", writeLimiter.Middleware(), reportHandler.Create) // Anonymous
		reports.GET("", readLimiter.Middleware(), reportHandler.List)     // Public
	}

	// --- Provider routes ---
	providers := v1.Group("/providers")
	providers.Use(readLimiter.Middleware())
	{
		providers.GET("", providerHandler.List)
		providers.GET("/:id", providerHandler.GetByID)
	}

	// --- Zone routes ---
	zones := v1.Group("/zones")
	{
		zones.GET("", readLimiter.Middleware(), zoneHandler.List)
		zones.POST("", writeLimiter.Middleware(), zoneHandler.Create)
		zones.GET("/:id/ranking", readLimiter.Middleware(), zoneHandler.Ranking)
	}

	// --- Map routes ---
	mapRoutes := v1.Group("/map")
	mapRoutes.Use(readLimiter.Middleware())
	{
		mapRoutes.GET("/points", mapHandler.Points)
		mapRoutes.GET("/heatmap", mapHandler.Heatmap)
		mapRoutes.GET("/providers", providerHandler.Nearby)
	}

	// --- Speedtest runner routes (for real speed measurement) ---
	runner := v1.Group("/speedtest")
	{
		runner.GET("/ping", runnerHandler.Ping)
		runner.GET("/download", runnerHandler.Download)
		runner.POST("/upload", runnerHandler.Upload)
	}

	// --- Serve static files (speedtest page) ---
	r.Static("/static", "./static")
	r.GET("/speedtest", func(c *gin.Context) {
		c.File("./static/speedtest.html")
	})

	// --- Graceful shutdown ---
	srv := &http.Server{
		Addr:         "0.0.0.0:" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 120 * time.Second, // Needs to be large for download tests
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("🚀 SpeedMap API running on http://localhost:%s", cfg.Port)
		log.Printf("🧪 Speedtest: http://localhost:%s/speedtest", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("⏳ Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("❌ Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server exited gracefully")
}
