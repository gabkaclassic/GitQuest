package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gabkaclassic/metrics/pkg/httpserver"

	"github.com/gabkaclassic/GitQuest/internal/cache"
	"github.com/gabkaclassic/GitQuest/internal/config"
	"github.com/gabkaclassic/GitQuest/internal/handler"
	"github.com/gabkaclassic/GitQuest/internal/repository"
	"github.com/gabkaclassic/GitQuest/internal/service"
	"github.com/gabkaclassic/GitQuest/internal/storage"
	"github.com/gabkaclassic/metrics/pkg/logger"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {

	cfg, err := config.ParseConfig()

	if err != nil {
		return fmt.Errorf("failed to parse configuration: %w", err)
	}

	logger.SetupLogger(logger.LogConfig(cfg.Log))

	slog.Debug("DB connection setup...")

	db, err := storage.NewDBStorage(cfg.DB)
	if err != nil {
		return fmt.Errorf("failed to initialize database storage: %w", err)
	}
	defer db.Close()

	slog.Debug("Initialize repositories...")
	repositories, err := initializeRepositories(db)

	if err != nil {
		return fmt.Errorf("failed to initialize repositories: %w", err)
	}

	slog.Debug("Initialize cache storage...")
	cacheStorage, err := storage.NewCacheStorage(cfg.Cache)

	if err != nil {
		return fmt.Errorf("failed to initialize cache storage: %w", err)
	}

	slog.Debug("Initialize cache client...")
	cacheClient, err := cache.NewEventCacheClient(cacheStorage)

	if err != nil {
		return fmt.Errorf("failed to initialize cache client: %w", err)
	}

	slog.Debug("Initialize services...")
	services, err := initializeServices(repositories, cacheClient)

	if err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
	}

	slog.Debug("Save rules...")
	err = services.RuleService.SaveAll(&cfg.Rules.Rules)

	if err != nil {
		return fmt.Errorf("failed to save rules: %w", err)
	}

	router, err := setupRouter(services)
	if err != nil {
		return fmt.Errorf("failed to setup HTTP router: %w", err)
	}

	server := httpserver.New(
		httpserver.Address(cfg.Server.Address),
		httpserver.Handler(&router),
	)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go server.Run(ctx, stop)
	go startBackgroundJobs(ctx, cacheClient, cfg.Jobs)

	<-ctx.Done()
	slog.Info("Shutdown complete")

	return nil
}

func initializeRepositories(connection *sql.DB) (*repositoriesList, error) {

	eventRepository, err := repository.NewEventRepository(connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create event repository: %w", err)
	}

	userRepository, err := repository.NewUserRepository(connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create user repository: %w", err)
	}

	ruleRepository, err := repository.NewRuleRepository(connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create rule repository: %w", err)
	}

	return &repositoriesList{
		EventRepository: eventRepository,
		UserRepository:  userRepository,
		RuleRepository:  ruleRepository,
	}, nil
}

func initializeServices(repositories *repositoriesList, cacheClient cache.EventCacheClient) (*servicesList, error) {

	eventService, err := service.NewEventService(repositories.EventRepository, cacheClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create event service: %w", err)
	}

	userService, err := service.NewUserService(repositories.UserRepository)
	if err != nil {
		return nil, fmt.Errorf("failed to create user service: %w", err)
	}

	ruleService, err := service.NewRuleService(repositories.RuleRepository)
	if err != nil {
		return nil, fmt.Errorf("failed to create rule service: %w", err)
	}

	return &servicesList{
		EventService: eventService,
		UserService:  userService,
		RuleService:  ruleService,
	}, nil
}

func setupRouter(services *servicesList) (http.Handler, error) {

	eventsHandler, err := handler.NewEventHandler(services.EventService)

	if err != nil {
		return nil, err
	}

	return handler.SetupRouter(&handler.RouterConfiguration{
		EventHandler: eventsHandler,
	}), nil
}

func startBackgroundJobs(ctx context.Context, eventCacheClient cache.EventCacheClient, cfg config.Jobs) {
	cleanupTicker := time.NewTicker(cfg.Cleanup.Interval)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-cleanupTicker.C:
			ctx, cancel := context.WithTimeout(ctx, cfg.Cleanup.Timeout)
			defer cancel()
			err := eventCacheClient.CleanupOldEventsFromCache(ctx)
			if err != nil {
				slog.Error("Cleanup old events error", slog.Any("error", err))
			}
			slog.Info("Cleanup completed")
		case <-ctx.Done():
			slog.Info("Background jobs shutting down...")
			return
		}
	}
}

type repositoriesList struct {
	EventRepository repository.EventRepository
	UserRepository  repository.UserRepository
	RuleRepository  repository.RuleRepository
}

type servicesList struct {
	EventService service.EventService
	UserService  service.UserService
	RuleService  service.RuleService
}
