package handler

import (
	"errors"
	"github.com/gabkaclassic/GitQuest/internal/config"
	"github.com/gabkaclassic/metrics/pkg/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	"net/http"
)

type RouterConfiguration struct {
	EventHandler       *EventHandler
	RuleHandler        *RuleHandler
	AchievementHandler *AchievementHandler
	Ratelimit          *config.Ratelimit
}

func SetupRouter(config *RouterConfiguration) (http.Handler, error) {

	if config.EventHandler == nil {
		return nil, errors.New("setup router error: event handler is nil")
	}

	if config.RuleHandler == nil {
		return nil, errors.New("setup router error: rule handler is nil")
	}

	if config.AchievementHandler == nil {
		return nil, errors.New("setup router error: achievement handler is nil")
	}

	if config.Ratelimit == nil {
		return nil, errors.New("setup router error: ratelimit config is nil")
	}

	router := chi.NewRouter()

	router.Use(
		middleware.Logger,
		httprate.Limit(
			config.Ratelimit.Limit,
			config.Ratelimit.Window,
			httprate.WithKeyFuncs(httprate.KeyByIP),
			httprate.WithResponseHeaders(httprate.ResponseHeaders{}),
		),
	)

	// Ping endpoint
	router.Get("/ping", func(w http.ResponseWriter, r *http.Request) {})

	setupEventRouter(router, config.EventHandler, middleware.Decompress())
	setupRuleRouter(router, config.RuleHandler, middleware.Decompress())
	setupAchievementRouter(router, config.AchievementHandler, middleware.Decompress())

	return router, nil
}

func setupEventRouter(
	router *chi.Mux,
	handler *EventHandler,
	decompressMiddleware func(handler http.Handler) http.Handler,
) {
	// Events
	router.Post(
		"/event",
		middleware.Wrap(
			http.HandlerFunc(handler.SaveAll),
			middleware.RequireContentType(middleware.JSON),
			middleware.Compress(map[middleware.ContentType]middleware.CompressType{
				middleware.JSON: middleware.GZIP,
			}),
			middleware.WithContentType(middleware.JSON),
			decompressMiddleware,
		),
	)
}

func setupRuleRouter(
	router *chi.Mux,
	handler *RuleHandler,
	decompressMiddleware func(handler http.Handler) http.Handler,
) {
	// Rules
	router.Get(
		"/rule",
		middleware.Wrap(
			http.HandlerFunc(handler.GetAll),
			middleware.RequireContentType(middleware.JSON),
			middleware.Compress(map[middleware.ContentType]middleware.CompressType{
				middleware.JSON: middleware.GZIP,
			}),
			middleware.WithContentType(middleware.JSON),
			decompressMiddleware,
		),
	)
}

func setupAchievementRouter(
	router *chi.Mux,
	handler *AchievementHandler,
	decompressMiddleware func(handler http.Handler) http.Handler,
) {
	// Achievements
	router.Get(
		"/achievement/{user}",
		middleware.Wrap(
			http.HandlerFunc(handler.GetByUser),
			middleware.RequireContentType(middleware.JSON),
			middleware.Compress(map[middleware.ContentType]middleware.CompressType{
				middleware.JSON: middleware.GZIP,
			}),
			middleware.WithContentType(middleware.JSON),
			decompressMiddleware,
		),
	)
}
