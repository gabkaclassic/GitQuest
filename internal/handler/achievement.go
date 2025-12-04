package handler

import (
	"encoding/json"
	"errors"
	"github.com/gabkaclassic/GitQuest/internal/service"
	api "github.com/gabkaclassic/metrics/pkg/error"
	"net/http"
)

type AchievementHandler struct {
	service service.AchievementService
}

func NewAchievementHandler(service service.AchievementService) (*AchievementHandler, error) {

	if service == nil {
		return nil, errors.New("create new achievements handler failed: service is nil")
	}

	return &AchievementHandler{
		service: service,
	}, nil
}

func (handler *AchievementHandler) GetByUser(w http.ResponseWriter, r *http.Request) {

	user := r.PathValue("user")

	summary, getSummaryErr := handler.service.GetSummaryByUser(r.Context(), user)

	if getSummaryErr != nil {
		api.RespondError(w, getSummaryErr)
		return
	}

	encodeErr := json.NewEncoder(w).Encode(summary)

	if encodeErr != nil {
		api.Internal("Get user achievements summary error", encodeErr)
		return
	}
}
