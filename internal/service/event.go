package service

import (
	"context"
	"errors"

	"github.com/gabkaclassic/GitQuest/internal/cache"
	"github.com/gabkaclassic/GitQuest/internal/dto"
	"github.com/gabkaclassic/GitQuest/internal/repository"
	api "github.com/gabkaclassic/metrics/pkg/error"
)

type EventService interface {
	SaveAll(ctx context.Context, events *[]dto.Event) *api.APIError
}

type eventService struct {
	repository repository.EventRepository
	cache      cache.EventCacheClient
}

func NewEventService(repository repository.EventRepository, cache cache.EventCacheClient) (EventService, error) {

	if repository == nil {
		return nil, errors.New("create new event service failed: repository is nil")
	}

	if cache == nil {
		return nil, errors.New("create new event service failed: cache client is nil")
	}

	return &eventService{
		repository: repository,
		cache:      cache,
	}, nil
}

func (service *eventService) SaveAll(ctx context.Context, events *[]dto.Event) *api.APIError {

	err := service.cache.SaveNewEvents(ctx, events)

	if err != nil {
		return api.Internal("cache operations error", err)
	}

	err = service.repository.SaveAll(events)

	if err != nil {
		return api.Internal("save events error", err)
	}

	return nil
}
