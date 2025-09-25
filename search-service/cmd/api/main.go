package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/victoragudo/hotel-management-system/pkg/entities"

	"github.com/common-nighthawk/go-figure"
	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/victoragudo/hotel-management-system/pkg/database"
	"github.com/victoragudo/hotel-management-system/pkg/logger"
	"github.com/victoragudo/hotel-management-system/search-service/internal/application/usecase"
	"github.com/victoragudo/hotel-management-system/search-service/internal/infrastructure/adapter"
	"github.com/victoragudo/hotel-management-system/search-service/internal/infrastructure/config"
	"github.com/victoragudo/hotel-management-system/search-service/internal/infrastructure/handler"
	"gorm.io/gorm"

	_ "github.com/victoragudo/hotel-management-system/search-service/docs"
)

// @title Hotel Management & Search Service API
// @version 1.0
// @description API for hotel search and management services
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost:8080
// @BasePath /
// @schemes http https

type Application struct {
	config *config.Config
	db     *gorm.DB
	redis  *redis.Client
	logger *slog.Logger
	server *http.Server

	hotelRepo     *adapter.PostgresHotelRepository
	cache         *adapter.RedisCacheAdapter
	searchEngine  *adapter.TypesenseAdapter
	hotelProvider *adapter.CupidAPIAdapter

	getHotelByIDUseCase        *usecase.GetHotelByIDUseCase
	searchHotelsUseCase        *usecase.SearchHotelsUseCase
	getHotelSuggestionsUseCase *usecase.GetHotelSuggestionsUseCase
	syncHotelsUseCase          *usecase.SyncHotelsUseCase

	hotelHandler *handler.HotelHandler
}

func main() {
	applicationLogger := logger.SetupLogger("info")

	cfg, err := config.LoadConfig()
	if err != nil {
		applicationLogger.Error(fmt.Sprintf("Failed to load configuration: %s", err.Error()))
		os.Exit(1)
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	app, err := NewApplication(cfg, applicationLogger)
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	if err := app.Start(); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}
}

func NewApplication(cfg *config.Config, applicationLogger *slog.Logger) (*Application, error) {
	connectionString := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable", cfg.Database.Host, cfg.Database.Port, cfg.Database.Database, cfg.Database.Username, cfg.Database.Password)
	db, err := database.GormOpen(connectionString)
	if err != nil {
		return nil, err
	}

	err = database.RunMigrations(db, &entities.HotelData{}, &entities.ReviewData{}, &entities.HotelTranslation{})
	if err != nil {
		return nil, err
	}

	redisClient := initRedis(cfg.Redis, applicationLogger)

	hotelRepo := adapter.NewPostgresHotelRepository(db, applicationLogger)
	cache := adapter.NewRedisCacheAdapterWithClient(redisClient, applicationLogger)

	searchEngine, err := adapter.NewTypesenseAdapter(cfg.Typesense.Host, cfg.Typesense.ApiKey, cfg.Typesense.CollectionName, applicationLogger)
	if err != nil {
		return nil, err
	}

	hotelProvider := adapter.NewCupidAPIAdapter(
		cfg.CupidAPI.BaseURL,
		cfg.CupidAPI.APIKey,
		cfg.CupidAPI.Timeout,
		applicationLogger,
	)

	getHotelByIDUseCase := usecase.NewGetHotelByIDUseCase(
		hotelRepo,
		hotelProvider,
		searchEngine,
		cache,
		applicationLogger,
	)

	searchHotelsUseCase := usecase.NewSearchHotelsUseCase(
		searchEngine,
		cache,
		applicationLogger,
	)

	getHotelSuggestionsUseCase := usecase.NewGetHotelSuggestionsUseCase(
		searchEngine,
		cache,
		applicationLogger,
	)

	syncHotelsUseCase := usecase.NewSyncHotelsUseCase(
		hotelRepo,
		searchEngine,
		cache,
		applicationLogger,
	)

	hotelHandler := handler.NewHotelHandler(
		getHotelByIDUseCase,
		searchHotelsUseCase,
		getHotelSuggestionsUseCase,
		syncHotelsUseCase,
		applicationLogger,
	)

	server := initServer(cfg.Server, hotelHandler, applicationLogger)

	return &Application{
		config:                     cfg,
		db:                         db,
		redis:                      redisClient,
		logger:                     applicationLogger,
		server:                     server,
		hotelRepo:                  hotelRepo,
		cache:                      cache,
		searchEngine:               searchEngine,
		hotelProvider:              hotelProvider,
		getHotelByIDUseCase:        getHotelByIDUseCase,
		searchHotelsUseCase:        searchHotelsUseCase,
		getHotelSuggestionsUseCase: getHotelSuggestionsUseCase,
		syncHotelsUseCase:          syncHotelsUseCase,
		hotelHandler:               hotelHandler,
	}, nil
}

func (app *Application) Start() error {
	ctx := context.Background()

	app.logger.Info("Starting search service",
		"version", "1.0.0",
		"address", app.config.Server.Address())

	if err := app.performHealthChecks(ctx); err != nil {
		app.logger.Error("Health checks failed", "error", err)
		return err
	}

	if app.config.Sync.InitialSyncOnStart {
		go app.performInitialSync(ctx)
	}

	if app.config.Sync.IncrementalInterval > 0 {
		go app.startPeriodicSync(ctx)
	}

	go func() {
		figure.NewFigure("API", "", true).Print()
		fmt.Println("")
		fmt.Println("Search service started at " + app.config.Server.Address())
		fmt.Println("")
		if err := app.server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			app.logger.Error("HTTP server failed", "error", err)
		}
	}()

	app.waitForShutdown()

	return nil
}

func (app *Application) performHealthChecks(ctx context.Context) error {
	app.logger.Info("Performing health checks")

	if sqlDB, err := app.db.DB(); err == nil {
		if err := sqlDB.PingContext(ctx); err != nil {
			return err
		}
	}

	if err := app.cache.Ping(ctx); err != nil {
		app.logger.Warn("Redis health check failed", "error", err)
	}

	if err := app.searchEngine.HealthCheck(ctx); err != nil {
		app.logger.Warn("MeiliSearch health check failed", "error", err)
	}

	return nil
}

func (app *Application) performInitialSync(ctx context.Context) {
	app.logger.Info("Starting initial synchronization")

	options := usecase.SyncOptions{
		FullSync:         true,
		BatchSize:        app.config.Sync.BatchSize,
		ClearIndexFirst:  true,
		UpdateCacheAfter: true,
	}

	result, err := app.syncHotelsUseCase.Execute(ctx, options)
	if err != nil {
		app.logger.Error("Initial sync failed", "error", err)
		return
	}

	app.logger.Info("Initial sync completed",
		"total_hotels", result.TotalHotels,
		"indexed_hotels", result.IndexedHotels,
		"duration", result.Duration)
}

func (app *Application) startPeriodicSync(ctx context.Context) {
	ticker := time.NewTicker(app.config.Sync.IncrementalInterval)
	defer ticker.Stop()

	app.logger.Info("Starting periodic sync", "interval", app.config.Sync.IncrementalInterval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			app.logger.Debug("Running incremental sync")

			options := usecase.SyncOptions{
				BatchSize:        app.config.Sync.BatchSize,
				UpdateCacheAfter: true,
			}

			result, err := app.syncHotelsUseCase.Execute(ctx, options)
			if err != nil {
				app.logger.Error("Incremental sync failed", "error", err)
				continue
			}

			if result.TotalHotels > 0 {
				app.logger.Info("Incremental sync completed",
					"total_hotels", result.TotalHotels,
					"indexed_hotels", result.IndexedHotels,
					"duration", result.Duration)
			}
		}
	}
}

func (app *Application) waitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	app.logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.server.Shutdown(ctx); err != nil {
		app.logger.Error("Server forced to shutdown", "error", err)
	}

	if sqlDB, err := app.db.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			app.logger.Error("Error closing database", "error", err)
		}
	}

	if err := app.redis.Close(); err != nil {
		app.logger.Error("Error closing Redis", "error", err)
	}

	app.logger.Info("Server stopped gracefully")
}

func initRedis(cfg config.RedisConfig, logger *slog.Logger) *redis.Client {
	logger.Info("Connecting to Redis", "address", cfg.Address())

	client := redis.NewClient(&redis.Options{
		Addr:            cfg.Address(),
		Password:        cfg.Password,
		DB:              cfg.Database,
		PoolSize:        cfg.PoolSize,
		DialTimeout:     cfg.DialTimeout,
		ReadTimeout:     cfg.ReadTimeout,
		WriteTimeout:    cfg.WriteTimeout,
		ConnMaxIdleTime: cfg.IdleTimeout,
	})

	logger.Info("Redis client created")
	return client
}

func initServer(cfg config.ServerConfig, hotelHandler *handler.HotelHandler, logger *slog.Logger) *http.Server {
	router := mux.NewRouter()

	api := router.PathPrefix("/api/v1").Subrouter()

	api.HandleFunc("/hotels/{id}", hotelHandler.GetHotelByID).Methods("GET")

	api.HandleFunc("/search/hotels", hotelHandler.SearchHotels).Methods("GET")
	api.HandleFunc("/search/suggestions", hotelHandler.GetHotelSuggestions).Methods("GET")
	api.HandleFunc("/search/trending", hotelHandler.GetTrendingSuggestions).Methods("GET")
	api.HandleFunc("/search/facets", hotelHandler.GetFacets).Methods("GET")

	admin := api.PathPrefix("/admin").Subrouter()
	admin.HandleFunc("/sync", hotelHandler.TriggerSync).Methods("POST")
	admin.HandleFunc("/sync/stats", hotelHandler.GetSyncStats).Methods("GET")

	router.HandleFunc("/health", hotelHandler.HealthCheck).Methods("GET")

	router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	router.Use(rateLimitMiddleware(100, time.Minute))
	router.Use(loggingMiddleware(logger))
	if cfg.EnableCORS {
		router.Use(corsMiddleware)
	}

	printRoutes(router, logger)

	return &http.Server{
		Addr:         cfg.Address(),
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}
}

func printRoutes(router *mux.Router, logger *slog.Logger) {
	fmt.Println("API Routes Overview")
	fmt.Println("═══════════════════════════════════════════════════════════════")

	var routes []string

	err := router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err != nil {
			return nil
		}

		methods, err := route.GetMethods()
		if err != nil {
			methods = []string{"ALL"}
		}

		methodStr := strings.Join(methods, ", ")
		routeDesc := fmt.Sprintf("  %-8s %s", methodStr, pathTemplate)

		switch {
		case strings.Contains(pathTemplate, "/health"):
			routeDesc += " - Health check endpoint"
		case strings.Contains(pathTemplate, "/swagger"):
			routeDesc += " - API documentation (Swagger UI)"
		case strings.Contains(pathTemplate, "/hotels/{id}"):
			routeDesc += " - Get specific hotel by ID"
		case strings.Contains(pathTemplate, "/search/hotels"):
			routeDesc += " - Search hotels with filters"
		case strings.Contains(pathTemplate, "/search/suggestions"):
			routeDesc += " - Get hotel search suggestions"
		case strings.Contains(pathTemplate, "/search/trending"):
			routeDesc += " - Get trending hotel suggestions"
		case strings.Contains(pathTemplate, "/search/facets"):
			routeDesc += " - Get search facets for filtering"
		case strings.Contains(pathTemplate, "/admin/sync"):
			routeDesc += " - Trigger hotel data synchronization"
		case strings.Contains(pathTemplate, "/admin/sync/stats"):
			routeDesc += " - Get synchronization statistics"
		default:
			routeDesc += " - API endpoint"
		}

		routes = append(routes, routeDesc)
		return nil
	})

	if err != nil {
		logger.Error("Error walking routes", "error", err)
		return
	}

	for _, route := range routes {
		fmt.Println(route)
	}

	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Printf("Total registered routes: %d\n", len(routes))
	fmt.Println("Visit /swagger/ for interactive API documentation")
}

func loggingMiddleware(logger *slog.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}

			next.ServeHTTP(wrapped, r)

			logger.Info("HTTP request",
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
				"status_code", wrapped.statusCode,
				"duration", time.Since(start),
			)
		})
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type rateLimiter struct {
	clients map[string]*clientLimit
	mu      sync.RWMutex
}

type clientLimit struct {
	tokens    int
	lastReset time.Time
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{
		clients: make(map[string]*clientLimit),
	}
}

func (rl *rateLimiter) allow(clientID string, maxRequests int, window time.Duration) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	client, exists := rl.clients[clientID]

	if !exists || now.Sub(client.lastReset) > window {
		rl.clients[clientID] = &clientLimit{
			tokens:    maxRequests - 1,
			lastReset: now,
		}
		return true
	}

	if client.tokens > 0 {
		client.tokens--
		return true
	}

	return false
}

func rateLimitMiddleware(maxRequests int, window time.Duration) mux.MiddlewareFunc {
	limiter := newRateLimiter()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := r.RemoteAddr
			if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
				clientIP = forwarded
			}

			if !limiter.allow(clientIP, maxRequests, window) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"Rate limit exceeded"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
