package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/your-username/click-lite-log-analytics/backend/internal/api"
	"github.com/your-username/click-lite-log-analytics/backend/internal/config"
	"github.com/your-username/click-lite-log-analytics/backend/internal/dashboard"
	"github.com/your-username/click-lite-log-analytics/backend/internal/database"
	"github.com/your-username/click-lite-log-analytics/backend/internal/ingestion"
	"github.com/your-username/click-lite-log-analytics/backend/internal/monitoring"
	"github.com/your-username/click-lite-log-analytics/backend/internal/websocket"
)

var version = "dev"

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Debug().Err(err).Msg("No .env file found")
	}

	// Setup logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if os.Getenv("LOG_LEVEL") == "debug" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	log.Info().Str("version", version).Msg("Starting Click-Lite Log Analytics")

	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.New(cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}
	defer db.Close()

	// Initialize WebSocket hub for real-time log tailing
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// Initialize dashboard service (singleton for in-memory storage)
	dashboardService := dashboard.NewService(db)

	// Initialize monitoring
	metrics := monitoring.NewMetricsCollector()
	metrics.SetDescription("total_logs_ingested", "Total number of logs ingested")
	metrics.SetDescription("total_queries_executed", "Total number of queries executed")
	metrics.SetDescription("query_duration_ms", "Query execution duration in milliseconds")
	metrics.SetDescription("storage_size_bytes", "Storage size in bytes")
	
	healthMonitor := monitoring.NewHealthMonitor(version)
	healthMonitor.RegisterChecker(monitoring.NewStorageHealthChecker("./data"))
	healthMonitor.RegisterChecker(monitoring.NewAPIHealthChecker("http://localhost:"+cfg.Server.Port, 5*time.Second))
	healthMonitor.RegisterChecker(monitoring.NewIngestionHealthChecker(metrics))
	healthMonitor.RegisterChecker(monitoring.NewQueryEngineHealthChecker(metrics))
	
	alertManager := monitoring.NewAlertManager(metrics)
	alertManager.AddListener(monitoring.NewLogAlertListener(log.Logger))
	
	// Initialize log tailer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Start alert checking
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				alertManager.CheckAlerts()
			case <-ctx.Done():
				return
			}
		}
	}()
	logTailer := websocket.NewLogTailer(db, wsHub)
	go logTailer.Start(ctx)

	// Initialize batch processor for ingestion
	batchProcessor := ingestion.NewBatchProcessor(db, 500, 5*time.Second)
	defer batchProcessor.Stop()

	// Initialize ingestion handlers
	httpHandler := ingestion.NewHTTPHandlerWithMetrics(batchProcessor, wsHub, metrics)
	
	// Start TCP server
	tcpServer := ingestion.NewTCPServer(":20003", batchProcessor, wsHub)
	if err := tcpServer.Start(); err != nil {
		log.Error().Err(err).Msg("Failed to start TCP server")
	} else {
		defer tcpServer.Stop()
	}
	
	// Start Syslog server
	syslogServer := ingestion.NewSyslogServer(":20004", batchProcessor, wsHub)
	if err := syslogServer.Start(); err != nil {
		log.Error().Err(err).Msg("Failed to start Syslog server")
	} else {
		defer syslogServer.Stop()
	}

	// Setup routes
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:3001"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", api.HealthCheck(db))
		r.Post("/logs", api.IngestLogs(db))
		r.Get("/logs", api.QueryLogs(db))
		r.Get("/storage/stats", api.StorageStats(db))
		r.HandleFunc("/ws", websocket.HandleWebSocket(wsHub))
		r.Get("/ws/stats", api.WebSocketStats(wsHub))
		
		// SQL Query endpoints
		r.Route("/query", func(r chi.Router) {
			r.Post("/execute", api.ExecuteQuery(db))
			r.Get("/saved", api.ListQueries(db))
			r.Post("/saved", api.SaveQuery(db))
			r.Get("/saved/{id}", api.GetQuery(db))
			r.Put("/saved/{id}", api.UpdateQuery(db))
			r.Delete("/saved/{id}", api.DeleteQuery(db))
			r.Post("/saved/{id}/execute", api.ExecuteSavedQuery(db))
			r.Get("/saved/{id}/execute", api.ExecuteSavedQuery(db))
		})

		// Query Builder endpoints
		r.Route("/query-builder", func(r chi.Router) {
			r.Get("/fields", api.GetAvailableFields(db))
			r.Post("/generate-sql", api.GenerateSQL(db))
			r.Post("/execute", api.ExecuteQueryBuilder(db))
			r.Post("/validate", api.ValidateQueryBuilder(db))
		})

		// Dashboard endpoints
		r.Route("/dashboards", func(r chi.Router) {
			r.Get("/", api.ListDashboards(dashboardService))
			r.Post("/", api.CreateDashboard(dashboardService))
			r.Get("/{id}", api.GetDashboard(dashboardService))
			r.Put("/{id}", api.UpdateDashboard(dashboardService))
			r.Delete("/{id}", api.DeleteDashboard(dashboardService))
			r.Post("/{id}/share", api.ShareDashboard(dashboardService))
			r.Get("/{dashboard_id}/widgets/{widget_id}/query", api.ExecuteWidgetQuery(dashboardService))
			r.Get("/{dashboard_id}/widgets/{widget_id}/data", api.GetWidgetData(dashboardService))
		})

		// Shared dashboard endpoints
		r.Get("/shared/{token}", api.GetSharedDashboard(dashboardService))
		
		// Ingestion endpoints
		r.Route("/ingest", func(r chi.Router) {
			r.Get("/health", httpHandler.HealthCheck())
			r.Post("/logs", httpHandler.IngestLogs())
			r.Post("/bulk", httpHandler.BulkIngestLogs())
		})
		
		// Monitoring endpoints
		r.Route("/monitoring", func(r chi.Router) {
			r.Get("/health", healthMonitor.HTTPHandler())
			r.Get("/health/live", healthMonitor.LivenessHandler())
			r.Get("/health/ready", healthMonitor.ReadinessHandler())
			r.Get("/metrics", api.GetMetrics(metrics))
			r.Get("/alerts", api.GetAlerts(alertManager))
			r.Get("/alerts/active", api.GetActiveAlerts(alertManager))
		})
	})

	// Start server
	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: r,
	}

	// Graceful shutdown
	done := make(chan bool, 1)
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Info().Msg("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		srv.SetKeepAlivesEnabled(false)
		if err := srv.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("Server shutdown failed")
		}
		close(done)
	}()

	log.Info().Str("port", cfg.Server.Port).Msg("Server started")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("Server failed to start")
	}

	<-done
	log.Info().Msg("Server stopped")
}