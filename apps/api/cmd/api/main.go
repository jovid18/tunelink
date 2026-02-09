package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	httpAdapter "github.com/tunelink/api/internal/adapter/in/http"
	noopAdapter "github.com/tunelink/api/internal/adapter/out/cache/noop"
	redisAdapter "github.com/tunelink/api/internal/adapter/out/cache/redis"
	resilientAdapter "github.com/tunelink/api/internal/adapter/out/cache/resilient"
	mysqlAdapter "github.com/tunelink/api/internal/adapter/out/persistence/mysql"
	"github.com/tunelink/api/internal/application/sync"
	urlApp "github.com/tunelink/api/internal/application/url"
	urlDomain "github.com/tunelink/api/internal/domain/url"
	"github.com/tunelink/api/internal/infrastructure"
)

func main() {
	cfg := infrastructure.Load()

	// Initialize adapters (outbound)
	urlRepo := mysqlAdapter.NewURLRepository(cfg.DB)

	// Initialize cache & click sync
	var urlCache urlDomain.Cache
	var resilientCache *resilientAdapter.ResilientCache
	var clickSync *sync.ClickSyncService

	if cfg.RedisEnabled {
		urlCache, resilientCache, clickSync = setupRedis(cfg, urlRepo)
	} else {
		urlCache = noopAdapter.NewURLCache()
		log.Println("Redis disabled, using noop cache")
	}

	// Initialize use cases (application)
	urlUseCase := urlApp.NewURLUseCase(urlRepo, urlCache)

	// Initialize handlers (inbound adapters)
	urlHandler := httpAdapter.NewURLHandler(urlUseCase, cfg.BaseURL)
	testHandler := httpAdapter.NewTestHandler(cfg.DB, urlCache)

	// Setup router
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Routes
	r.GET("/health", httpAdapter.Health)
	r.POST("/api/urls", urlHandler.Create)
	r.GET("/r/:shortUrl", urlHandler.Redirect)

	// Test routes (for load testing)
	r.GET("/api/test/stats", testHandler.GetStats)
	r.DELETE("/api/test/urls", testHandler.DeleteAllURLs)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		log.Printf("Server starting on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
	if resilientCache != nil {
		resilientCache.Stop()
	}
	if clickSync != nil {
		clickSync.Stop()
	}

	log.Println("Server exited")
}

// setupRedis creates a ResilientCache + ClickSync with state change wiring.
func setupRedis(cfg *infrastructure.Config, urlRepo urlDomain.Repository) (urlDomain.Cache, *resilientAdapter.ResilientCache, *sync.ClickSyncService) {
	primary := redisAdapter.NewURLCache(cfg.Redis)
	fallback := noopAdapter.NewURLCache()
	rc := resilientAdapter.New(cfg.Redis, primary, fallback, cfg.RedisHealthy)
	rc.Start()

	cs := sync.NewClickSyncService(urlRepo, rc, 10*time.Second)
	if !cfg.RedisHealthy {
		cs.Pause()
	}
	cs.Start()

	rc.OnStateChange(func(healthy bool) {
		if healthy {
			cs.Resume()
		} else {
			cs.Pause()
		}
	})

	log.Println("Using ResilientCache (Redis with automatic failover)")
	return rc, rc, cs
}
