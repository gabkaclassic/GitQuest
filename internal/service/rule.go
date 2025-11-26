package service

import (
	"errors"

	"github.com/gabkaclassic/GitQuest/internal/dto"
	"github.com/gabkaclassic/GitQuest/internal/repository"
)

type RuleService interface {
	SaveAll(*[]dto.Rule) error
}

type ruleService struct {
	repository repository.RuleRepository
}

func NewRuleService(repository repository.RuleRepository) (RuleService, error) {

	if repository == nil {
		return nil, errors.New("create new rule service failed: repository is nil")
	}

	return &ruleService{
		repository: repository,
	}, nil
}

func (service *ruleService) SaveAll(rules *[]dto.Rule) error {

	return service.repository.SaveAll(rules)
}
