package handler

import (
	"encoding/json"
	"errors"
	"github.com/gabkaclassic/GitQuest/internal/dto"
	"github.com/gabkaclassic/GitQuest/internal/service"
	api "github.com/gabkaclassic/metrics/pkg/error"
	"net/http"
)

type EventHandler struct {
	service service.EventService
}

func NewEventHandler(service service.EventService) (*EventHandler, error) {

	if service == nil {
		return nil, errors.New("create new events handler failed: service is nil")
	}

	return &EventHandler{
		service: service,
	}, nil
}

func (handler *EventHandler) SaveAll(w http.ResponseWriter, r *http.Request) {
	events := make([]dto.Event, 0)
	err := json.NewDecoder(r.Body).Decode(&events)
	if err != nil {
		api.RespondError(w, api.UnprocessibleEntity("Invalid input JSON"))
		return
	}

	saveErr := handler.service.SaveAll(r.Context(), events)

	if saveErr != nil {
		api.RespondError(w, saveErr)
		return
	}
}
