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
	"github.com/redis/go-redis/v9"

	"github.com/gabkaclassic/GitQuest/internal/cache"
	"github.com/gabkaclassic/GitQuest/internal/config"
	"github.com/gabkaclassic/GitQuest/internal/dto"
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

	slog.Debug("Initialize cache clients...")
	cacheClients, err := initializeCacheClients(cacheStorage)

	if err != nil {
		return fmt.Errorf("failed to initialize cache clients: %w", err)
	}

	slog.Debug("Initialize services...")
	services, err := initializeServices(&cfg.Notification, repositories, cacheClients)

	if err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
	}

	if cfg.Rules.Reeval {
		slog.Debug("Reeval start...")
		rulesDiffs, err := services.RuleService.GetRulesDiffs(cfg.Rules.Rules)

		if err != nil {
			return fmt.Errorf("failed to get rules diffs: %w", err)
		}

		if rulesDiffs != nil && len(rulesDiffs) > 0 {
			err := services.AchievementService.ReevalByDiffs(rulesDiffs)

			if err != nil {
				return fmt.Errorf("failed to reeval: %w", err)
			}
		} else {
			slog.Debug("Rules diffs list is empty, skip reeval")
		}
	}

	slog.Debug("Save rules...")
	err = services.RuleService.SaveAll(cfg.Rules.Rules)

	if err != nil {
		return fmt.Errorf("failed to save rules: %w", err)
	}

	slog.Debug("Load users to cache...")
	ctx := context.Background()
	err = services.EventService.LoadUsersToCache(ctx)

	if err != nil {
		return fmt.Errorf("failed load users to cache: %w", err)
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
	go startBackgroundJobs(ctx, cacheClients.eventCacheClient, services.AchievementService, cfg.Rules.Rules, cfg.Jobs)

	<-ctx.Done()
	slog.Info("Shutdown complete")

	return nil
}

func initializeCacheClients(connection *redis.Client) (*cacheClientsList, error) {
	eventCacheClient, err := cache.NewEventCacheClient(connection)

	if err != nil {
		return nil, fmt.Errorf("failed to initialize event cache client: %w", err)
	}

	userCacheClient, err := cache.NewUserCacheClient(connection)

	if err != nil {
		return nil, fmt.Errorf("failed to initialize user cache client: %w", err)
	}

	achievementCacheClient, err := cache.NewAchievementCacheClient(connection)

	if err != nil {
		return nil, fmt.Errorf("failed to initialize achievement cache client: %w", err)
	}

	return &cacheClientsList{
		eventCacheClient:       eventCacheClient,
		userCacheClient:        userCacheClient,
		achievementCacheClient: achievementCacheClient,
	}, nil
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

	achievmentRepository, err := repository.NewAchievementRepository(connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create achievement repository: %w", err)
	}

	return &repositoriesList{
		EventRepository:       eventRepository,
		UserRepository:        userRepository,
		RuleRepository:        ruleRepository,
		AchievementRepository: achievmentRepository,
	}, nil
}

func initializeServices(cfg *config.Notification, repositories *repositoriesList, cacheClients *cacheClientsList) (*servicesList, error) {

	eventService, err := service.NewEventService(
		repositories.EventRepository,
		cacheClients.eventCacheClient,
		cacheClients.userCacheClient,
	)
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

	notificationService, err := service.NewNotificationService(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification service: %w", err)
	}

	achievementService, err := service.NewAchievementService(
		repositories.AchievementRepository, repositories.UserRepository,
		cacheClients.eventCacheClient, cacheClients.userCacheClient,
		cacheClients.achievementCacheClient, notificationService,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create achievement service: %w", err)
	}

	return &servicesList{
		EventService:       eventService,
		UserService:        userService,
		RuleService:        ruleService,
		AchievementService: achievementService,
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

func startBackgroundJobs(ctx context.Context, eventCacheClient cache.EventCacheClient,
	achievementService service.AchievementService, rules []dto.Rule, cfg config.Jobs) {

	cleanupTicker := time.NewTicker(cfg.Cleanup.Interval)
	calculateTicker := time.NewTicker(cfg.Calculate.Interval)
	defer cleanupTicker.Stop()
	defer calculateTicker.Stop()

	for {
		select {
		case <-cleanupTicker.C:
			runCleanup(ctx, eventCacheClient, cfg)

		case <-calculateTicker.C:
			runCalculation(ctx, achievementService, rules, cfg)

		case <-ctx.Done():
			slog.Info("Background jobs shutting down...")
			return
		}
	}
}

func runCleanup(ctx context.Context, client cache.EventCacheClient, cfg config.Jobs) {
	cleanupCtx, cancel := context.WithTimeout(ctx, cfg.Cleanup.Timeout)
	defer cancel()

	err := client.CleanupOldEventsFromCache(cleanupCtx)
	if err != nil {
		slog.Error("Cleanup old events error", slog.Any("error", err))
	}
	slog.Info("Cleanup completed")
}

func runCalculation(ctx context.Context, service service.AchievementService,
	rules []dto.Rule, cfg config.Jobs) {

	calculationCtx, cancel := context.WithTimeout(ctx, cfg.Calculate.Timeout)
	defer cancel()

	err := service.CheckForNewAchievements(calculationCtx, rules)
	if err != nil {
		slog.Error("Calculate new achievements error", slog.Any("error", err))
	}
	slog.Info("Calculate completed")
}

type (
	repositoriesList struct {
		EventRepository       repository.EventRepository
		UserRepository        repository.UserRepository
		RuleRepository        repository.RuleRepository
		AchievementRepository repository.AchievementRepository
	}
	servicesList struct {
		EventService        service.EventService
		UserService         service.UserService
		RuleService         service.RuleService
		AchievementService  service.AchievementService
		NotificationService service.NotificationService
	}
	cacheClientsList struct {
		userCacheClient        cache.UserCacheClient
		eventCacheClient       cache.EventCacheClient
		achievementCacheClient cache.AchievementCacheClient
	}
)
