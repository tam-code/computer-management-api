package main

import (
	"computer-management-api/internal/config"
	"computer-management-api/internal/database"
	"computer-management-api/internal/handler"
	"computer-management-api/internal/middleware"
	"computer-management-api/internal/notification"
	"computer-management-api/internal/repository"
	"computer-management-api/internal/router"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize repository
	repo := repository.NewComputerRepository(db)

	// Initialize notification client with enhanced configuration
	notificationConfig := notification.NotificationConfig{
		URL:            cfg.NotificationService.URL,
		Timeout:        cfg.NotificationService.Timeout,
		RetryAttempts:  cfg.NotificationService.RetryAttempts,
		RetryDelay:     cfg.NotificationService.RetryDelay,
		MaxPayloadSize: cfg.NotificationService.MaxPayloadSize,
	}
	notifier := notification.NewNotifierWithConfig(notificationConfig)

	// Initialize handler with logger
	logger := log.Default()
	h := handler.NewComputerHandler(repo, notifier, logger)

	// Setup router with security configuration
	r := router.NewRouter(h, cfg)

	// Initialize logging middleware
	loggingMW := middleware.NewLoggingMiddleware(logger)

	// Wrap router with logging middleware
	finalHandler := loggingMW.LogRequests(r)

	// Configure server with security settings
	server := &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.Port),
		Handler:        finalHandler,
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		IdleTimeout:    cfg.Server.IdleTimeout,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
	}

	// Channel to listen for interrupt signal to gracefully shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on port %d with security features enabled", cfg.Port)
		log.Printf("Security: Rate limit=%d RPS, Burst=%d, CORS=%v, Timeout=%v",
			cfg.Security.RateLimitRPS,
			cfg.Security.RateLimitBurst,
			cfg.Security.EnableCORS,
			cfg.Security.RequestTimeout,
		)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Block until we receive a signal
	<-done
	log.Println("Server is shutting down...")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Security.ShutdownTimeout)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	} else {
		log.Println("Server exited gracefully")
	}
}
