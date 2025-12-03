package handler

import (
	"encoding/json"
	"errors"
	"github.com/gabkaclassic/GitQuest/internal/service"
	api "github.com/gabkaclassic/metrics/pkg/error"
	"net/http"
)

type RuleHandler struct {
	service service.RuleService
}

func NewRuleHandler(service service.RuleService) (*RuleHandler, error) {

	if service == nil {
		return nil, errors.New("create new rules handler failed: service is nil")
	}

	return &RuleHandler{
		service: service,
	}, nil
}

func (handler *RuleHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	rules := handler.service.GetAll()

	err := json.NewEncoder(w).Encode(rules)

	if err != nil {
		api.Internal("Get rules error", err)
		return
	}
}
