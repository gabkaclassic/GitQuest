package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/gabkaclassic/GitQuest/internal/dto"
	"github.com/gabkaclassic/GitQuest/internal/repository"
	api "github.com/gabkaclassic/metrics/pkg/error"
)

type EventService interface {
	SaveAll(ctx context.Context, events *[]dto.Event) *api.APIError
}

type eventService struct {
	repository repository.EventRepository
	cache      *redis.Client
}

func NewEventService(repository repository.EventRepository, cache *redis.Client) (EventService, error) {

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

	pipeline := service.cache.Pipeline()

	for _, event := range *events {
		eventKey := fmt.Sprintf("events:%s:%s", event.Actor, event.EventType)
		pipeline.ZAdd(
			ctx, eventKey,
			redis.Z{
				Score:  float64(event.Timestamp.Unix()),
				Member: event.ID,
			},
		)
	}

	_, err := pipeline.Exec(ctx)

	if err != nil {
		return api.Internal("cache operations error", err)
	}

	err = service.repository.SaveAll(events)

	if err != nil {
		return api.Internal("save events error", err)
	}

	return nil
}
