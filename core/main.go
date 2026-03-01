package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Import PostgreSQL driver
	_ "github.com/lib/pq"

	"weather-station/core/config"
	"weather-station/core/database"
	"weather-station/core/handlers"
	"weather-station/shared/logger"
	"weather-station/shared/types"
	"weather-station/shared/workers"
)

func main() {
	// Load configuration from environment variables
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	// Connect to PostgreSQL database with retry logic
	db, err := database.ConnectWithRetry(cfg)
	if err != nil {
		logger.Error("Failed to connect to database: %v", err)
		os.Exit(1)
	}
	defer db.Close()

	// Prepare insert statement for reuse across all workers
	insertStmt, err := database.PrepareInsertStatement(db)
	if err != nil {
		logger.Error("Failed to prepare insert statement: %v", err)
		os.Exit(1)
	}
	defer insertStmt.Close()

	// Create HTTP server instance with dependencies
	server := handlers.NewServer(db, cfg)

	// Create job function for worker pool
	jobFunc := func(reading types.Reading) error {
		resultChan := make(chan error, 1)
		database.InsertReading(insertStmt, resultChan, reading)
		return <-resultChan
	}

	// Start worker pool
	workerPool := workers.StartWorkerPool(jobFunc, cfg.NumWorkers(), cfg.IngestionChanSize(), cfg.ResultQueueSize())
	defer workerPool.Shutdown()

	// Process results asynchronously
	go func() {
		for err := range workerPool.GetResultChan() {
			if err != nil {
				logger.Error("Failed to insert reading: %v", err)
			}
		}
	}()

	// Set up HTTP routes
	http.HandleFunc("/api/ingest", handlers.MethodCheck(http.MethodPost, handlers.IngestHandler(workerPool.GetJobChan())))
	http.HandleFunc("/api/weather/current", handlers.MethodCheck(http.MethodGet, server.CurrentWeatherHandler))
	http.HandleFunc("/api/weather/recent", handlers.MethodCheck(http.MethodGet, server.RecentWeatherHandler))
	http.HandleFunc("/api/weather/history", handlers.MethodCheck(http.MethodGet, server.HistoryWeatherHandler))
	http.HandleFunc("/api/weather/years", handlers.MethodCheck(http.MethodGet, server.AvailableYearsHandler))
	http.HandleFunc("/health", handlers.MethodCheck(http.MethodGet, server.HealthCheckHandler))

	// Serve static files for dashboard
	fileserver := http.FileServer(http.Dir("./static"))
	http.Handle("/", fileserver)

	// Apply logging middleware if enabled
	var handler http.Handler = http.DefaultServeMux
	if cfg.EnableHTTPLogging() {
		handler = handlers.LoggingMiddleware(handler, true)
	}

	// Start daily aggregator goroutine
	go func() {
		// Run immediately on startup for yesterday's data
		yesterday := time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour)
		if err := database.ComputeDailyRollup(db, yesterday); err != nil {
			logger.Error("Failed to compute initial daily rollup: %v", err)
		}

		// Set up ticker to run daily
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			previousDay := time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour)
			if err := database.ComputeDailyRollup(db, previousDay); err != nil {
				logger.Error("Failed to compute daily rollup: %v", err)
			}
		}
	}()

	// Start astronomical data update goroutine
	go func() {
		// Run immediately on startup
		server.UpdateAstronomicalData()

		// Calculate time until next midnight
		now := time.Now()
		nextMidnight := now.AddDate(0, 0, 1).Truncate(24 * time.Hour)
		initialDelay := nextMidnight.Sub(now)

		// Wait until midnight, then run daily
		time.Sleep(initialDelay)

		// Set up ticker to run daily
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			server.UpdateAstronomicalData()
		}
	}()

	// Start retention cleaner goroutine
	go func(retentionDays int) {
		// Wait 1 hour before first run to ensure aggregator runs first
		time.Sleep(1 * time.Hour)

		// Create ticker to run daily
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			if err := database.CleanOldReadings(db, retentionDays); err != nil {
				logger.Error("Failed to delete old readings: %v", err)
			}
		}
	}(cfg.RetentionDays())

	// Start HTTP server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.ServerPort()),
		Handler: handler,
	}

	go func() {
		logger.Info("Starting server on port %d", cfg.ServerPort())
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Failed to start HTTP server: %v", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)
	<-shutdownChan

	logger.Info("Shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("Failed to shutdown HTTP server: %v", err)
	}

	logger.Info("Shutdown complete")
}
