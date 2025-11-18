package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/gabkaclassic/metrics/pkg/middleware"
)

type RouterConfiguration struct {
	EventHandler *EventHandler
}

func SetupRouter(config *RouterConfiguration) http.Handler {

	router := chi.NewRouter()

	router.Use(
		middleware.Logger,
	)

	// Ping endpoint
	router.Get("/ping", func(w http.ResponseWriter, r *http.Request) {})

	setupEventRouter(router, config.EventHandler, middleware.Decompress())

	return router
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
