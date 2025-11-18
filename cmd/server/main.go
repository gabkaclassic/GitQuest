package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/gabkaclassic/metrics/pkg/httpserver"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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

	storage, err := storage.NewDBStorage(cfg.DB)
	if err != nil {
		return fmt.Errorf("failed to initialize database storage: %w", err)
	}
	defer storage.Close()

	slog.Debug("Initialize repositories...")
	repositories, err := initializeRepositories(storage)

	if err != nil {
		return fmt.Errorf("failed to initialize repositories: %w", err)
	}

	slog.Debug("Initialize services...")
	services, err := initializeServices(repositories)

	if err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
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

	<-ctx.Done()
	slog.Info("Shutdown complete")

	return nil
}

func initializeRepositories(connection *sql.DB) (*repositoriesList, error) {

	eventRepository, err := repository.NewEventRepository(connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create event repository: %w", err)
	}

	return &repositoriesList{
		EventRepository: eventRepository,
	}, nil
}

func initializeServices(repositories *repositoriesList) (*servicesList, error) {

	eventService, err := service.NewEventService(repositories.EventRepository)
	if err != nil {
		return nil, fmt.Errorf("failed to create event service: %w", err)
	}

	return &servicesList{
		EventService: eventService,
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

type repositoriesList struct {
	EventRepository repository.EventRepository
}

type servicesList struct {
	EventService service.EventService
}
