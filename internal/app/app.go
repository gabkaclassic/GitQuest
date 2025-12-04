package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gabkaclassic/GitQuest/internal/cache"
	"github.com/gabkaclassic/GitQuest/internal/config"
	"github.com/gabkaclassic/GitQuest/internal/dto"
	"github.com/gabkaclassic/GitQuest/internal/handler"
	"github.com/gabkaclassic/GitQuest/internal/job"
	"github.com/gabkaclassic/GitQuest/internal/repository"
	"github.com/gabkaclassic/GitQuest/internal/service"
	"github.com/gabkaclassic/GitQuest/internal/storage"
	"github.com/gabkaclassic/metrics/pkg/httpserver"
	"github.com/gabkaclassic/metrics/pkg/logger"
	"github.com/redis/go-redis/v9"
)

type App struct {
	cfg            *config.Config
	db             *sql.DB
	cacheStorage   *redis.Client
	router         http.Handler
	services       *servicesList
	cacheClients   *cacheClientsList
	repos          *repositoriesList
	server         *httpserver.Server
	backgroundJobs context.CancelFunc
}

func NewApp() (*App, error) {
	cfg, err := config.ParseConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	logger.SetupLogger(logger.LogConfig(cfg.Log))

	slog.Debug("DB connection setup...")
	db, err := storage.NewDBStorage(cfg.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database storage: %w", err)
	}

	slog.Debug("Initialize cache storage...")
	cacheStorage, err := storage.NewCacheStorage(cfg.Cache)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize cache storage: %w", err)
	}

	slog.Debug("Initialize repositories...")
	repos, err := initializeRepositories(db)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize repositories: %w", err)
	}

	slog.Debug("Initialize cache clients...")
	cacheClients, err := initializeCacheClients(cacheStorage)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize cache clients: %w", err)
	}

	slog.Debug("Initialize services...")
	services, err := initializeServices(&cfg.Notification, repos, cacheClients, cfg.Rules.Rules)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}

	if err := processRules(cfg, services); err != nil {
		db.Close()
		return nil, err
	}

	slog.Debug("Load users to cache...")
	ctx := context.Background()
	err = services.EventService.LoadUsersToCache(ctx)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed load users to cache: %w", err)
	}

	router, err := setupRouter(services)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to setup HTTP router: %w", err)
	}

	server := httpserver.New(
		httpserver.Address(cfg.Server.Address),
		httpserver.Handler(&router),
	)

	return &App{
		cfg:          cfg,
		db:           db,
		cacheStorage: cacheStorage,
		router:       router,
		services:     services,
		cacheClients: cacheClients,
		repos:        repos,
		server:       server,
	}, nil
}

func (a *App) Run() error {
	defer a.close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	bgCtx, bgCancel := context.WithCancel(ctx)
	defer bgCancel()
	a.backgroundJobs = bgCancel

	go job.StartBackgroundJobs(bgCtx, a.cacheClients.eventCacheClient,
		a.services.AchievementService, a.cfg.Rules.Rules, a.cfg.Jobs)

	go a.server.Run(ctx, stop)

	<-ctx.Done()
	slog.Info("Shutdown complete")
	return nil
}

func (a *App) close() {
	if a.db != nil {
		a.db.Close()
	}
	if a.backgroundJobs != nil {
		a.backgroundJobs()
	}
}

func processRules(cfg *config.Config, services *servicesList) error {
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
	err := services.RuleService.SaveAll(cfg.Rules.Rules)
	if err != nil {
		return fmt.Errorf("failed to save rules: %w", err)
	}

	return nil
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

	achievementRepository, err := repository.NewAchievementRepository(connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create achievement repository: %w", err)
	}

	return &repositoriesList{
		EventRepository:       eventRepository,
		UserRepository:        userRepository,
		RuleRepository:        ruleRepository,
		AchievementRepository: achievementRepository,
	}, nil
}

func initializeServices(cfg *config.Notification, repositories *repositoriesList,
	cacheClients *cacheClientsList, rules []dto.Rule) (*servicesList, error) {

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

	ruleService, err := service.NewRuleService(repositories.RuleRepository, rules)
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
		EventService:        eventService,
		UserService:         userService,
		RuleService:         ruleService,
		AchievementService:  achievementService,
		NotificationService: notificationService,
	}, nil
}

func setupRouter(services *servicesList) (http.Handler, error) {
	eventHandler, err := handler.NewEventHandler(services.EventService)
	if err != nil {
		return nil, err
	}

	ruleHandler, err := handler.NewRuleHandler(services.RuleService)
	if err != nil {
		return nil, err
	}

	achievementHandler, err := handler.NewAchievementHandler(services.AchievementService)
	if err != nil {
		return nil, err
	}

	return handler.SetupRouter(&handler.RouterConfiguration{
		EventHandler:       eventHandler,
		RuleHandler:        ruleHandler,
		AchievementHandler: achievementHandler,
	})
}
