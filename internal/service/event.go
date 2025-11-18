package service

import (
	"errors"

	"github.com/gabkaclassic/GitQuest/internal/dto"
	"github.com/gabkaclassic/GitQuest/internal/repository"
	api "github.com/gabkaclassic/metrics/pkg/error"
)

type EventService interface {
	SaveAll(events *[]dto.Event) *api.APIError
}

type eventService struct {
	repository repository.EventRepository
}

func NewEventService(repository repository.EventRepository) (EventService, error) {

	if repository == nil {
		return nil, errors.New("create new event service failed: repository is nil")
	}

	return &eventService{
		repository: repository,
	}, nil
}

func (service *eventService) SaveAll(events *[]dto.Event) *api.APIError {

	err := service.repository.SaveAll(events)

	if err != nil {
		return api.Internal("save events error", err)
	}

	return nil
}
