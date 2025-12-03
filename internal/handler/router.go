package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/gabkaclassic/metrics/pkg/middleware"
)

type RouterConfiguration struct {
	EventHandler *EventHandler
	RuleHandler  *RuleHandler
}

func SetupRouter(config *RouterConfiguration) (http.Handler, error) {

	if config.EventHandler == nil {
		return nil, errors.New("setup router error: event handler is nil")
	}

	if config.RuleHandler == nil {
		return nil, errors.New("setup router error: rule handler is nil")
	}

	router := chi.NewRouter()

	router.Use(
		middleware.Logger,
	)

	// Ping endpoint
	router.Get("/ping", func(w http.ResponseWriter, r *http.Request) {})

	setupEventRouter(router, config.EventHandler, middleware.Decompress())
	setupRuleRouter(router, config.RuleHandler, middleware.Decompress())

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
