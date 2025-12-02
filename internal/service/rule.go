package service

import (
	"errors"
	"strings"

	"github.com/gabkaclassic/GitQuest/internal/dto"
	"github.com/gabkaclassic/GitQuest/internal/repository"
	"github.com/gabkaclassic/GitQuest/internal/storage"
)

type RuleService interface {
	SaveAll([]dto.Rule) error
	GetRulesDiffs([]dto.Rule) ([]dto.RuleDiff, error)
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

func (service *ruleService) SaveAll(rules []dto.Rule) error {

	return service.repository.SaveAll(rules)
}

func (service *ruleService) GetRulesDiffs(rules []dto.Rule) ([]dto.RuleDiff, error) {
	diffs := make([]dto.RuleDiff, 0)

	for _, rule := range rules {
		lastRuleVersion, err := service.repository.GetLastRule(rule.Name)

		if err != nil {
			if storage.IsNotFoundError(err) {
				continue
			}
			return nil, err
		}

		if !strings.EqualFold(lastRuleVersion.Version, rule.Version) && lastRuleVersion.Reward < rule.Reward {
			diffs = append(diffs, dto.RuleDiff{
				Name:       rule.Name,
				OldVersion: lastRuleVersion.Version,
				NewVersion: rule.Version,
				RewardDiff: rule.Reward - lastRuleVersion.Reward,
			})
		}
	}

	return diffs, nil
}
