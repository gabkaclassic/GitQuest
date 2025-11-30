package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/gabkaclassic/GitQuest/internal/cache"
	"github.com/gabkaclassic/GitQuest/internal/dto"
	"github.com/gabkaclassic/GitQuest/internal/repository"
	api "github.com/gabkaclassic/metrics/pkg/error"
)

type EventService interface {
	SaveAll(ctx context.Context, events *[]dto.Event) *api.APIError
	LoadUsersToCache(ctx context.Context) error
}

type eventService struct {
	repository       repository.EventRepository
	userCacheClient  cache.UserCacheClient
	eventCacheClient cache.EventCacheClient
}

func NewEventService(repository repository.EventRepository, eventCacheClient cache.EventCacheClient, userCacheClient cache.UserCacheClient) (EventService, error) {

	if repository == nil {
		return nil, errors.New("create new event service failed: repository is nil")
	}

	if eventCacheClient == nil {
		return nil, errors.New("create new event service failed: event cache client is nil")
	}

	if userCacheClient == nil {
		return nil, errors.New("create new event service failed: user cache client is nil")
	}

	return &eventService{
		repository:       repository,
		eventCacheClient: eventCacheClient,
		userCacheClient:  userCacheClient,
	}, nil
}

func (service *eventService) SaveAll(ctx context.Context, events *[]dto.Event) *api.APIError {

	err := service.repository.SaveAll(events)

	if err != nil {
		return api.Internal("save events error", err)
	}

	users, err := service.eventCacheClient.SaveNewEvents(ctx, events)

	if err != nil {
		return api.Internal("cache operations error", err)
	}

	err = service.userCacheClient.SaveAll(ctx, users)
	slog.Info("save all users", slog.Any("error", err), slog.Any("users", users))

	if err != nil {
		return api.Internal("cache operations error", err)
	}

	return nil
}

func (service *eventService) LoadUsersToCache(ctx context.Context) error {
	users, err := service.repository.GetAllUsersWithEvents()

	if err != nil {
		return err
	}

	err = service.userCacheClient.SaveAll(ctx, users)

	return err
}
